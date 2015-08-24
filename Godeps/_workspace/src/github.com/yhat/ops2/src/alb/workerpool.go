package alb

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/yhat/ops2/src/mps"
)

// WorkerPool is a structure which intends to make serveral worker nodes look
// like a single machine.
// It's thread safe; multiple goroutines may call any function.
//
// WorkerPool also controls shutdowns. If something is remove from it,
// WorkerPool attempts to destroy it before returning.
type WorkerPool struct {
	mu *sync.Mutex

	tlsConfig *tls.Config

	// map from instance id to worker id
	instances map[int64]int64

	workers map[int64]*worker
}

// NewPool creates a new worker pool.
// tlsConfig is the configuration used to connect to the worker nodes.
func NewPool(tlsConfig *tls.Config) *WorkerPool {
	return &WorkerPool{
		mu:        new(sync.Mutex),
		tlsConfig: tlsConfig,
		instances: make(map[int64]int64),
		workers:   make(map[int64]*worker),
	}
}

type worker struct {
	cli        *mps.MPSClient
	nInstances int
}

// QueueBuild reserves a build on a worker.
func (pool *WorkerPool) QueueBuild(instId int64) (workerId int64, err error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	n := len(pool.workers)
	if n == 0 {
		return 0, fmt.Errorf("no workers available")
	}

	minInst := -1
	for id, worker := range pool.workers {
		if worker.nInstances < minInst || minInst < 0 {
			minInst = worker.nInstances
			workerId = id
		}
	}

	pool.instances[instId] = workerId
	pool.workers[workerId].nInstances++

	return workerId, nil
}

// Build triggers a build for the id reserved by QueueBuild.
// Once building, the caller may use "RemoveInstance" to stop the build.
func (pool *WorkerPool) Build(instanceId int64, info *mps.DeployInfo, bundle string) error {
	pool.mu.Lock()
	cli, ok := pool.instance(instanceId)
	pool.mu.Unlock()
	if !ok {
		return errors.New("instance not found")
	}
	if err := cli.Deploy(instanceId, info, bundle); err != nil {
		return err
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	_, ok = pool.instance(instanceId)
	if !ok {
		return errors.New("build cancelled")
	}
	return nil
}

// Workers lists all the workers in the worker pool.
func (pool *WorkerPool) Workers() []int64 {
	workers := []int64{}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for worker := range pool.workers {
		workers = append(workers, worker)
	}
	return workers
}

// Instances returns the ids of all instances tracked by the worker pool.
func (pool *WorkerPool) Instances() []int64 {
	n := 0

	pool.mu.Lock()
	defer pool.mu.Unlock()

	instances := make([]int64, len(pool.instances))
	for inst := range pool.instances {
		instances[n] = inst
		n++
	}
	return instances
}

// RemoveWorker pulls the worker from rotation and attempts to delete all
// instances on that worker.
func (pool *WorkerPool) RemoveWorker(workerId int64) error {
	return pool.removeWorker(workerId, true)
}

// ReleaseWorker removes a worker from rotation without destroying any
// instances on that worker.
func (pool *WorkerPool) ReleaseWorker(workerId int64) error {
	return pool.removeWorker(workerId, false)
}

func (pool *WorkerPool) removeWorker(workerId int64, kill bool) error {
	instIds := []int64{}
	pool.mu.Lock()
	worker, ok := pool.workers[workerId]
	if !ok {
		pool.mu.Unlock()
		return errors.New("worker not found")
	}
	delete(pool.workers, workerId)

	for id, wId := range pool.instances {
		if wId == workerId {
			instIds = append(instIds, id)
			delete(pool.instances, id)
		}
	}
	pool.mu.Unlock()

	if kill {
		var err error
		for _, instId := range instIds {
			dErr := worker.cli.Destroy(instId)
			if dErr != nil {
				if err != nil {
					err = dErr
				} else {
					log.Println("worker pool: could not remove destroy instance: %v", instId)
				}
			}
		}
	}
	return nil
}

// Add attempts to add a worker to the pool.
func (pool *WorkerPool) Add(workerBaseURL string, id int64) (err error) {
	cli, err := mps.NewMPSClient(workerBaseURL, pool.tlsConfig)
	if err != nil {
		return err
	}
	status, err := cli.Status()
	if err != nil {
		return fmt.Errorf("could not ping worker: %v", err)
	}
	deployments := status.Deployments
	n := 0
	for _, d := range deployments {
		// clean up any instances that aren't ready to make predictions
		if !d.Ready {
			if err := cli.Destroy(d.Id); err != nil {
				log.Printf("worker pool: could not destroy instance: %v", err)
			}
		} else {
			deployments[n] = d
			n++
		}
	}
	deployments = deployments[:n]

	pool.mu.Lock()
	defer pool.mu.Unlock()
	if _, ok := pool.workers[id]; ok {
		return errors.New("id already assigned to another worker")
	}
	pool.workers[id] = &worker{cli, 0}
	for _, d := range deployments {
		// register each instance with the worker pool
		pool.instances[d.Id] = id
	}
	return nil
}

// instance is a low level function for aquiring an instance.
// It is NOT thread safe. You must aquire pool.mu before calling
func (pool *WorkerPool) instance(id int64) (cli *mps.MPSClient, ok bool) {
	workerId, ok := pool.instances[id]
	if !ok {
		return nil, false
	}
	defer func() {
		if !ok {
			delete(pool.instances, id)
		}
	}()
	worker, ok := pool.workers[workerId]
	if !ok {
		return nil, false
	}
	return worker.cli, true
}

// Predict returns the handler associated with the given instance.
// If no instance is found, ok will be false.
func (pool *WorkerPool) Predict(instanceId int64) (h http.Handler, ok bool) {
	pool.mu.Lock()
	cli, ok := pool.instance(instanceId)
	pool.mu.Unlock()
	if !ok {
		return nil, false
	}
	return cli.PredictHandler(instanceId), true
}

// RemoveInstance removes and destroys a given instance from the worker pool.
func (pool *WorkerPool) RemoveInstance(id int64) error {
	pool.mu.Lock()
	workerId, ok := pool.instances[id]
	if !ok {
		pool.mu.Unlock()
		return errors.New("No such instance")
	}
	delete(pool.instances, id)
	worker, ok := pool.workers[workerId]
	if !ok {
		pool.mu.Unlock()
		return errors.New("No such instance")
	}

	worker.nInstances--
	pool.mu.Unlock()

	if err := worker.cli.Destroy(id); err != nil {
		return fmt.Errorf("bad response from worker: %v", err)
	}
	return nil
}

// PingInstance sends a heartbeat request to the given instance.
func (pool *WorkerPool) PingInstance(id int64) error {
	pool.mu.Lock()
	cli, ok := pool.instance(id)
	pool.mu.Unlock()
	if !ok {
		return errors.New("No such instance")
	}
	return cli.Heartbeat(id)
}

// PingWorker sends a heartbeat request to the given worker.
func (pool *WorkerPool) PingWorker(id int64) error {
	pool.mu.Lock()
	worker, ok := pool.workers[id]
	pool.mu.Unlock()
	if !ok {
		return errors.New("No such worker")
	}
	return worker.cli.Ping()
}

func (pool *WorkerPool) Logs(id int64) ([]*mps.LogLine, error) {
	pool.mu.Lock()
	worker, ok := pool.workers[id]
	pool.mu.Unlock()
	if !ok {
		return nil, errors.New("No such worker")
	}
	return worker.cli.Logs()
}

func init() {
	rand.Seed(time.Now().Unix())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
