/*
package supervisor implements the MPS supervisor and associated data structures.
*/
package alb

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"golang.org/x/net/context"

	"github.com/yhat/ops2/src/db"
	"github.com/yhat/ops2/src/mps"
	"github.com/yhat/ops2/src/mps/tlsconfig"
)

var (
	StatusQueued   string = "queued"
	StatusBuilding string = "building"
	StatusOnline   string = "online"
	StatusFailed   string = "failed"
	StatusAsleep   string = "asleep"
)

var (
	ErrDeploymentCancelled = errors.New("deployment cancelled")
)

// ModelStorage is an interface for the ALB's database.
// It allows us to test the supervisor without spinning up a MySQL instance.
type ModelStorage interface {
	// Get retrieves deployment information for a model.
	Get(user, model string, ver int) (info *mps.DeployInfo, bundlePath string, err error)

	// GetLatest should return the latest verison of a given model.
	GetLatest(user, model string) (version int, err error)

	// SetBuildStatus sets the build status for a given model.
	// If the model's status is "asleep", this function should ignore any failed
	// statuses.
	SetBuildStatus(user, model, status string) error

	// NewDeployment registers a deployment with the storage. It should return
	// the deployment id, and a slice of instance ids for which to build.
	// It is NewDeployment's job to determine how many instances should be
	// built for a single deployment.
	NewDeployment(user, model string, version int) (deployId int64, instIds []int64, err error)
}

type routeName struct {
	User  string
	Model string
}

type Supervisor struct {
	Logger *log.Logger

	// Generic model storage to retrieve deployment info and bundles from.
	storage ModelStorage

	// pool is the underlying WorkerPool.
	pool *WorkerPool

	// To use either of the following maps, you MUST aquire this mutex.
	// While holding the mutex no predictions can be made, so don't do I/O
	// bounded operations if you've locked it (e.g. DB call).
	mu *sync.Mutex

	// routes and deployments marks active deployments for a given routeName.
	// routes are used for deployments
	routes      map[routeName]*Deployment
	deployments map[routeName]*Deployment

	// asleep marks routes as asleep or awake
	asleep map[routeName]bool

	// deploymentPool provides the deployment queue.
	// It is a buffered channel of length MAX_CONCURRENT_DEPLOYMENTS.
	deploymentPool chan struct{}

	// instMu and insts maps instance ids to deployment ids
	instMu *sync.Mutex
	insts  map[int64]int64

	// tlsconfig Client for secure communication.
	// Set package 'github.com/yhat/ops2/src/ops/tlsconfig' for details.
	tlsHandshaker *tlsconfig.Client
}

type SupervisorConfig struct {
	// Cannot be nil
	Storage ModelStorage
	// May be nil
	TLSHandshaker *tlsconfig.Client
	// May be nil
	Workers []db.Worker
	// May be nil
	Deployments []db.DeploymentReq

	Logger *log.Logger
}

// NewSupervisor generates a supervisor.
// If tlsHandshaker is not nil, it is used to coordinate secure communcation
// between the supervisor and workers.
func NewSupervisor(c SupervisorConfig) (*Supervisor, error) {
	if c.Storage == nil {
		return nil, fmt.Errorf("SupervisorConfig: storage cannot be nil")
	}

	var tlsConfig *tls.Config
	if c.TLSHandshaker != nil {
		tlsConfig = c.TLSHandshaker.TLSConfig()
	}
	pool := NewPool(tlsConfig)

	s := &Supervisor{
		Logger:         c.Logger,
		tlsHandshaker:  c.TLSHandshaker,
		storage:        c.Storage,
		mu:             new(sync.Mutex),
		routes:         make(map[routeName]*Deployment),
		deployments:    make(map[routeName]*Deployment),
		asleep:         make(map[routeName]bool),
		deploymentPool: make(chan struct{}, 4),
		pool:           pool,
		instMu:         new(sync.Mutex),
		insts:          make(map[int64]int64),
	}

	for _, worker := range c.Workers {
		s.logf("adding worker: %s", worker.Host)
		if err := s.AddWorker(worker.Host, worker.Id); err != nil {
			return nil, fmt.Errorf("could not add worker: %v", err)
		}
	}

	toDelete := map[int64]bool{}
	instances := pool.Instances()
	for _, inst := range instances {
		toDelete[inst] = true
	}

	if n := len(toDelete); n != 0 {
		s.logf("found %d existing instances from last startup", n)
	}

	// We want to clean up before redeploying
	// Accumulate deployments to make in this slice.
	redeployments := []db.DeploymentReq{}

	// Evaluate Deployments. These signify where the system should be.
	// For each deployment attempt to find instances living on the workers
	// which satisfy the request.
	// If none are found, trigger a redeployment.
	for _, req := range c.Deployments {

		r := routeName{req.Username, req.Modelname}

		if req.Asleep {
			// model is asleep. mark it as and keep going without trying to
			// redeploy the model
			s.mu.Lock()
			s.asleep[r] = true
			s.mu.Unlock()
			continue
		}

		instIds := []int64{}
		for _, inst := range req.ValidInstanceIds {
			if toDelete[inst] {
				s.insts[inst] = req.LastDeployId
				instIds = append(instIds, inst)
				delete(toDelete, inst)
			}
		}

		if len(instIds) == 0 {
			// There are no containers around from the old deployment.
			// We'll have to redeploy after setup
			redeployments = append(redeployments, req)
			continue
		}

		s.logf("associated %d instances with model %s/%s", len(instIds), req.Username, req.Modelname)

		d := NewDeployment(pool)

		var err error
		d.Info, d.Bundle, err = c.Storage.Get(req.Username, req.Modelname, req.Version)
		if err != nil {
			return nil, fmt.Errorf("could not get model info for %s:%s %v", req.Username, req.Modelname, err)
		}

		d.Username = req.Username
		d.Modelname = req.Modelname
		d.Version = req.Version
		d.DeployId = req.LastDeployId
		d.instanceIds = instIds

		s.mu.Lock()
		if _, ok := s.routes[r]; ok {
			s.mu.Unlock()
			return nil, fmt.Errorf("route /%s/models/%s requested twice", req.Username, req.Modelname)
		}
		s.routes[r] = d
		s.mu.Unlock()
	}

	s.logf("removing %d instances", len(toDelete))
	for inst := range toDelete {
		if err := pool.RemoveInstance(inst); err != nil {
			s.logf("failed to remove instance %s: %v", inst, err)
		}
	}

	// begin redeployments for deployments missing instances
	for _, req := range redeployments {
		go func(req db.DeploymentReq) {
			err := s.Deploy(req.Username, req.Modelname, req.Version)
			if err != nil {
				s.logf("failed to make redeployment %s:%s %s", req.Username, req.Modelname, err)
			}
		}(req)
	}

	return s, nil
}

// AddWorker adds a given worker to the Supervisor's rotation.
func (s *Supervisor) AddWorker(addr string, id int64) error {
	scheme := "http"
	if s.tlsHandshaker != nil {
		if err := s.tlsHandshaker.Handshake(addr); err != nil {
			return fmt.Errorf("could not configure tls between ALB and worker: %v", err)
		}
		scheme = "https"
	}
	return s.pool.Add(scheme+"://"+addr+"/", id)
}

func (s *Supervisor) Sleep(user, model string) {
	s.delete(user, model, true)
	s.storage.SetBuildStatus(user, model, StatusAsleep)
}

func (s *Supervisor) Delete(user, model string) {
	s.delete(user, model, false)
}

func (s *Supervisor) delete(user, model string, sleep bool) {
	r := routeName{user, model}

	s.mu.Lock()
	if sleep {
		s.asleep[r] = true
	}
	deployment, deployOK := s.deployments[r]
	route, routeOK := s.routes[r]
	delete(s.routes, r)
	delete(s.deployments, r)
	s.mu.Unlock()

	if deployOK {
		go deployment.Kill()
	}
	if routeOK {
		go route.Kill()
	}
}

// DeleteUser removes all instances associated with a user.
func (s *Supervisor) DeleteUser(user string) {
	s.deleteUser(user, false)
}

// During tests, it's more convenient for container cleanup to block. When the
// app is running, it's better to respond before cleanup.
//
// Use this function during testing, and have DeleteUser always call with
// blocking turned off.
func (s *Supervisor) deleteUser(user string, blocking bool) {
	deployments := []*Deployment{}

	s.mu.Lock()

	for routeName, deployment := range s.routes {
		if routeName.User == user {
			delete(s.routes, routeName)
			deployments = append(deployments, deployment)
		}
	}
	for routeName, deployment := range s.deployments {
		if routeName.User == user {
			delete(s.deployments, routeName)
			deployments = append(deployments, deployment)
		}
	}
	s.mu.Unlock()

	for _, d := range deployments {
		if blocking {
			d.Kill()
		} else {
			go d.Kill()
		}
	}
}

func (s *Supervisor) Wake(user, model string) error {
	r := routeName{user, model}

	s.mu.Lock()
	if isAsleep := s.asleep[r]; !isAsleep {
		s.mu.Unlock()
		return fmt.Errorf("model is already awake")
	}
	delete(s.deployments, r)
	delete(s.asleep, r)
	s.mu.Unlock()

	latest, err := s.storage.GetLatest(user, model)
	if err != nil {
		return err
	}
	return s.Deploy(user, model, latest)
}

func (s *Supervisor) Restart(user, model string) error {
	r := routeName{user, model}

	var version int
	s.mu.Lock()
	routeInfo, ok := s.routes[r]
	s.mu.Unlock()
	if !ok {
		// If there is not an active build, rebuild the latest
		var err error
		version, err = s.storage.GetLatest(user, model)
		if err != nil {
			return err
		}
	} else {
		version = routeInfo.Version
	}
	return s.Deploy(user, model, version)
}

type buildParams struct {
	user       string
	model      string
	version    int
	nInstances int
	bundlePath string
	info       *mps.DeployInfo
}

// Deploy builds nInstances of a model on some combination of workers.
// It fails after the first error.
func (s *Supervisor) Deploy(user, model string, version int) (err error) {
	r := routeName{user, model}

	// get the bundle and model information form the storage.
	info, bundle, err := s.storage.Get(user, model, version)
	if err != nil {
		return err
	}
	deployId, ids, err := s.storage.NewDeployment(user, model, version)
	if err != nil {
		return err
	}
	s.instMu.Lock()
	for _, id := range ids {
		s.insts[id] = deployId
	}
	s.instMu.Unlock()
	deployment := &Deployment{
		ExpNum:      len(ids),
		Username:    user,
		Modelname:   model,
		Version:     version,
		Info:        info,
		Bundle:      bundle,
		DeployId:    deployId,
		instanceIds: ids,
		mu:          new(sync.Mutex),
		pool:        s.pool,
	}
	deployment.ctx, deployment.cancel = context.WithCancel(context.Background())

	s.mu.Lock()
	currDeployment, ok := s.deployments[r]
	// grab the space in the deployments map
	s.deployments[r] = deployment
	s.mu.Unlock()

	if ok {
		go func() {
			//TODO(eric): logging
			currDeployment.Kill()
		}()
	}

	setStatus := func(status string) {
		s.storage.SetBuildStatus(user, model, status)
	}
	defer func() {
		if err != nil && err != ErrDeploymentCancelled {
			setStatus(StatusFailed)
			s.mu.Lock()
			delete(s.deployments, r)
			s.mu.Unlock()
		}
	}()

	setStatus(StatusQueued)

	if deployment.IsCancelled() {
		return ErrDeploymentCancelled
	}

	s.deploymentPool <- struct{}{}
	// defer opening a space
	defer func() { <-s.deploymentPool }()
	setStatus(StatusBuilding)

	if err := deployment.BuildInstances(ids); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			deployment.Kill()
		}
	}()

	// set the route to the new route
	s.mu.Lock()
	if deployment.IsCancelled() {
		s.mu.Unlock()
		return ErrDeploymentCancelled
	}
	delete(s.deployments, r)
	oldDeployment, ok := s.routes[r]
	s.routes[r] = deployment
	s.mu.Unlock()

	s.storage.SetBuildStatus(user, model, StatusOnline)

	// clean up all the old instances
	if ok && oldDeployment != nil {
		oldDeployment.Kill()
	}
	return nil
}

func (s *Supervisor) logf(format string, a ...interface{}) {
	if s.Logger == nil {
		log.Printf(format, a...)
	} else {
		s.Logger.Printf(format, a...)
	}
}

// Predict proxies the ResponseWriter and Request to a random instance.
func (s *Supervisor) Predict(user, model string, w http.ResponseWriter, r *http.Request) {
	route := routeName{user, model}
	s.mu.Lock()
	deployment, ok := s.routes[route]
	s.mu.Unlock()
	if !ok {
		data := []byte(`{"error":"model not found"}`)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(data)
		return
	}
	ids := deployment.Instances()
	shuffle(ids)
	for _, id := range ids {
		h, ok := s.pool.Predict(id)
		if ok {
			h.ServeHTTP(w, r)
			return
		}
	}
	s.mu.Lock()
	_, ok = s.deployments[route]
	asleep := s.asleep[route]
	s.mu.Unlock()
	var data []byte
	errCode := http.StatusInternalServerError
	if asleep {
		data = []byte(`{"error":"model is asleep"}`)
	} else if ok {
		data = []byte(`{"error":"model still building"}`)
	} else {
		data = []byte(`{"error":"no instances of model found"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(errCode)
	w.Write(data)
	return
	return
}

// Fisher–Yates shuffle
func shuffle(sli []int64) {
	for i := range sli {
		j := rand.Intn(i + 1)
		sli[j], sli[i] = sli[i], sli[j]
	}
}

func (s *Supervisor) MonitorWorker(id int64) {
	err := s.pool.PingWorker(id)
	if err == nil {
		return
	}
	s.logf("worker heartbeat failed: %s", id)
	s.RemoveWorker(id)
}

func (s *Supervisor) RemoveWorker(id int64) {

	if err := s.pool.RemoveWorker(id); err != nil {
		s.logf("could not remove worker: %v", err)
	}
}

func logPath(logdir, user, model string) string {
	p := fmt.Sprintf("%s_%s.jsonl", user, model)
	return filepath.Join(logdir, p)
}

func ReadDeploymentLogs(logdir, user, model string, deployId int64) ([]*mps.LogLine, error) {
	p := logPath(logdir, user, model)
	flags := os.O_RDONLY | os.O_EXCL
	file, err := os.OpenFile(p, flags, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return []*mps.LogLine{}, nil
		}
		return nil, err
	}
	defer file.Close()

	lines := []*mps.LogLine{}
	d := json.NewDecoder(file)
	for {
		line := mps.LogLine{}
		err := d.Decode(&line)
		if err != nil {
			if err == io.EOF {
				return lines, nil
			}
			return nil, err
		}
		if line.DeploymentId == deployId {
			lines = append(lines, &line)
		}
	}
}

func ReadModelLogs(logdir, user, model string) ([]*mps.LogLine, error) {
	p := logPath(logdir, user, model)
	flags := os.O_RDONLY | os.O_EXCL
	file, err := os.OpenFile(p, flags, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			return []*mps.LogLine{}, nil
		}
		return nil, err
	}
	defer file.Close()

	return mps.ParseLogs(file)
}

func (s *Supervisor) WriteLogs(logdir string) {

	lines := []*mps.LogLine{}

	workers := s.pool.Workers()
	for _, worker := range workers {
		logLines, err := s.pool.Logs(worker)
		if err != nil {
			log.Printf("supervisor: could not get logs from worker %d: %v", worker, err)
			continue
		}
		lines = append(lines, logLines...)
	}

	instData := func(instId int64) (int64, bool) {
		s.instMu.Lock()
		id, ok := s.insts[instId]
		s.instMu.Unlock()
		return id, ok
	}

	files := map[int64]*os.File{}
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()

	for _, line := range lines {
		instId := line.InstanceId
		deployId, ok := instData(instId)
		if !ok {
			log.Println("supervisor: got an unknown instance id: %s", instId)
			continue
		}
		line.DeploymentId = deployId
		file, ok := files[deployId]
		if !ok {
			logpath := logPath(logdir, line.User, line.Model)
			var err error
			flags := os.O_WRONLY | os.O_APPEND | os.O_CREATE
			file, err = os.OpenFile(logpath, flags, 0644)
			if err != nil {
				log.Println("supervisor: could not open log file for writing: %v", err)
				continue
			}
		}
		b, err := json.Marshal(line)
		if err != nil {
			log.Println("supervisor: could not marshal logs %v", err)
			continue
		}
		file.Write(b)
		file.Write([]byte("\n"))
	}
}
