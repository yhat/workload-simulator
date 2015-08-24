package mps

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/yhat/go-docker"
)

var (
	routeDeploy    = "/deploy"
	routeDestroy   = "/stop"
	routePredict   = "/predict"
	routeHeartbeat = "/heartbeat"
	routeStatus    = "/status"
	routePing      = "/ping"
	routeLogs      = "/logs"
)

type deployment struct {
	// parent mps
	mps *MPS

	mu     *sync.Mutex
	ready  bool
	kernel *kernel
	cid    string
	img    string
	ctx    context.Context
	cancel func()
}

func (mps *MPS) newDeployment() *deployment {
	ctx, cancel := context.WithCancel(context.Background())
	return &deployment{mps: mps, mu: new(sync.Mutex), ctx: ctx, cancel: cancel}
}

func (d *deployment) Kernel() (*kernel, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.kernel, d.ready
}

func (d *deployment) kill() {
	d.cancel()
	d.mu.Lock()
	if d.kernel != nil {
		d.kernel.Close()
	}
	d.mu.Unlock()
	if d.cid != "" {
		if err := d.mps.docker.RemoveContainer(d.cid, true, false); err != nil {
			d.mps.logf("could not remove container '%s' %v", d.cid, err)
		}
	}

	if d.img != "" {
		if d.mps.ImageDeletionPolicy != nil {
			d.mps.ImageDeletionPolicy(d.img)
		} else {
			if _, err := d.mps.docker.RemoveImage(d.img); err != nil {
				d.mps.logf("could not reomve docker image %s: %v", d.img, err)
			}
		}
	}
}

type model struct {
	kernel  *kernel
	cleanup func()
}

var (
	DefaultHeartbeatTimeout = time.Second
	DefaultLogLinesCapacity = 10000
)

// MPS is the server running on an worker node.
type MPS struct {
	docker *docker.Client

	mu *sync.Mutex

	deployments map[int64]*deployment
	// destroyed marks ids that have been destroyed
	// If the MPS sees a deployment request for an id that is already in this
	// map, the deployment is ignored with ErrDeploymentCancelled
	destroyed map[int64]struct{}

	// logger for model logs
	logger *logger

	// Is this a mock kernel mps?
	// If true then the MPS
	isMock bool

	// ImageDeletionPolicy defines a function to call when
	// an container is removed.
	// If nil, the container's image is deleted.
	ImageDeletionPolicy func(img string)

	// ErrorLog specifies a logger to print errors to.
	// If nil, the log package's Printf function is used.
	ErrorLog *log.Logger

	// HeartbeatTimeout defines a timeout for all kernel heartbeats.
	// It defaults to 1 second.
	HeartbeatTimeout time.Duration
}

func (mps *MPS) logf(format string, a ...interface{}) {
	if mps.ErrorLog == nil {
		log.Printf(format, a...)
	} else {
		mps.ErrorLog.Printf(format, a...)
	}
}

// NewMPS initializes an MPS.
func NewMPS() (*MPS, error) {
	cli, err := docker.NewDefaultClient(3 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %v", err)
	}

	info, err := cli.Info()
	if err != nil {
		return nil, fmt.Errorf("could not query docker info: %v", err)
	}
	if info.Driver != "aufs" {
		return nil, fmt.Errorf("mps requires docker driver to be 'aufs', got '%s'", info.Driver)
	}

	// TODO: It should be the ALB's responsibility to check these
	// See https://github.com/yhat/ops2/issues/38
	for _, img := range requiredDockerImages {
		if _, err := cli.InspectImage(img); err != nil {
			if err == docker.ErrNotFound {
				return nil, fmt.Errorf("MPS requires docker image: %s", img)
			}
			return nil, fmt.Errorf("could not inspect docker image %s: %v", img, err)
		}
	}
	mps := MPS{
		deployments:      make(map[int64]*deployment),
		destroyed:        make(map[int64]struct{}),
		mu:               new(sync.Mutex),
		docker:           cli,
		HeartbeatTimeout: DefaultHeartbeatTimeout,
		logger:           newLogger(DefaultLogLinesCapacity),
	}
	return &mps, nil
}

func (mps *MPS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case routeDeploy:
		mps.handleDeploy(w, r)
	case routePredict:
		mps.handlePredict(w, r)
	case routeHeartbeat:
		mps.handleHeartbeat(w, r)
	case routeDestroy:
		mps.handleDestroy(w, r)
	case routeStatus:
		mps.handleStatus(w, r)
	case routePing:
		mps.handlePing(w, r)
	case routeLogs:
		mps.logger.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (mps *MPS) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

// Reset causes the MPS to release all net connections, containers, and images
// under it's control.
// It is safe to continue using the MPS after calling this.
func (mps *MPS) Reset() {
	mps.mu.Lock()
	defer mps.mu.Unlock()
	for _, d := range mps.deployments {
		d.kill()
	}
	mps.deployments = make(map[int64]*deployment)
}

// heartbeat queries the underlying kernel for a heartbeat
// with a given timeout.
// If the kernel fails to respond, the underlying Docker container is removed
// and the kernel is removed from rotation.
func (mps *MPS) heartbeat(kernelId int64, timeout time.Duration) error {
	mps.mu.Lock()
	d, ok := mps.deployments[kernelId]
	if !ok {
		mps.mu.Unlock()
		return fmt.Errorf("kernel not found: %s", kernelId)
	}
	k, ok := d.Kernel()
	mps.mu.Unlock()
	if !ok {
		return fmt.Errorf("no kernel set")
	}

	hbChan := make(chan error, 1)
	go func() { hbChan <- k.heartbeat() }()

	var err error
	select {
	case err = <-hbChan:
	case <-time.After(timeout):
		err = fmt.Errorf("heartbeat timed out")
	}

	return err
}

func (mps *MPS) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
		return
	}

	var statuses []DeploymentStatus

	mps.mu.Lock()
	statuses = make([]DeploymentStatus, len(mps.deployments))
	i := 0
	for id, d := range mps.deployments {
		_, ready := d.Kernel()
		statuses[i] = DeploymentStatus{Id: id, Ready: ready}
		i++
	}
	mps.mu.Unlock()

	s := MPSStatus{
		Deployments: statuses,
	}
	data, err := json.Marshal(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (mps *MPS) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
		return
	}
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id provided: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := mps.heartbeat(id, mps.HeartbeatTimeout); err != nil {
		http.Error(w, "Heartbeat failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

var (
	ErrNotFound = errors.New("not found")
)

func (mps *MPS) handleDestroy(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "I only respond to POST requests", http.StatusNotImplemented)
		return
	}
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id provided: "+err.Error(), http.StatusBadRequest)
		return
	}
	mps.mu.Lock()
	mps.destroyed[id] = struct{}{}
	deployment, ok := mps.deployments[id]
	delete(mps.deployments, id)
	mps.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	deployment.kill()
	w.WriteHeader(http.StatusOK)
}

var errConnClosed = errors.New("connection closed")

func (mps *MPS) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
		return
	}
	// CloseNotifier tells us if the other end is still listening though
	// some middleware does not implement this.
	// To prevent hanging deployments this is a requirement for deployment
	// that is panic worthy.
	cn, ok := w.(http.CloseNotifier)
	if !ok {
		panic("deployment: ResponseWriter must be a close notifier")
	}
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id provided: "+err.Error(), http.StatusBadRequest)
		return
	}

	d := mps.newDeployment()

	mps.mu.Lock()
	_, destroyed := mps.destroyed[id]
	if !destroyed {
		if _, ok = mps.deployments[id]; !ok {
			mps.deployments[id] = d
		}
	}
	mps.mu.Unlock()
	if destroyed {
		http.Error(w, "Deployment cancelled", http.StatusBadRequest)
		return
	}
	if ok {
		http.Error(w, fmt.Sprintf("deployment id '%s' already taken", id), http.StatusBadRequest)
		return
	}

	err = mps.deploy(r, cn.CloseNotify(), d, id)
	if err != nil {
		// remove the deployment from the mps's lists of deployments and kill
		// the deployment
		mps.mu.Lock()
		delete(mps.deployments, id)
		mps.mu.Unlock()
		d.kill()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (mps *MPS) deploy(r *http.Request, closed <-chan bool, d *deployment, id int64) (err error) {
	isClosed := func() error {
		select {
		case <-closed:
			return errConnClosed
		case <-d.ctx.Done():
			return fmt.Errorf("deployment cancelled")
		default:
			return nil
		}
	}

	tempFile, err := ioutil.TempFile("", "scienceops_bundle_")
	if err != nil {
		return fmt.Errorf("could not make temp file: %v", err)
	}
	name := tempFile.Name()
	defer os.Remove(name)

	// parse the deployment data from the request
	dockerFile, deployInfo, err := decodeDeployment(r.Body, tempFile)
	tempFile.Close()
	if err != nil {
		return fmt.Errorf("could not receive deployment: %v", err)
	}

	if err = isClosed(); err != nil {
		return err
	}
	logs := mps.logger.NewWriter(deployInfo.Username, deployInfo.Modelname, id)

	// if the mps is a mock mps, just create a mock kernel and return
	if mps.isMock {
		conn := newMockContainer()
		k, err := newKernel(conn, logs)
		if err != nil {
			return fmt.Errorf("could not start mock kernel: %v", err)
		}
		d.mu.Lock()
		d.ready = true
		d.kernel = k
		d.mu.Unlock()
		return nil
	}

	// attempt to create the image
	imgName := randSeq(16)
	err = buildImage(dockerFile, name, imgName, logs)
	if err != nil {
		return fmt.Errorf("could not build image: %v", err)
	}
	d.mu.Lock()
	d.img = imgName
	d.mu.Unlock()

	if err = isClosed(); err != nil {
		return err
	}

	// start the container
	containerName := randSeq(16)
	cid, conn, err := startContainer(imgName, containerName)
	if err != nil {
		return fmt.Errorf("could not start container: %v", err)
	}
	d.mu.Lock()
	d.cid = cid
	d.mu.Unlock()

	if err = isClosed(); err != nil {
		return err
	}

	kernel, err := newKernel(conn, logs)
	if err != nil {
		return fmt.Errorf("could not start the kernel: %v", err)
	}
	d.mu.Lock()
	d.kernel = kernel
	d.ready = true
	d.mu.Unlock()

	if err = isClosed(); err != nil {
		return err
	}

	return nil
}

// encodeDeployment sends both the deployment and the bundle across a single
// connection.
// Use readDeployment to read the request.
func encodeDeployment(w io.Writer, dockerFile []byte, d *DeployInfo, bundleReader io.Reader) error {

	// Since we'll be sending some small chuncks, it's better to have a
	// buffered writer so we don't make a network call every time.
	wb := bufio.NewWriter(w)

	gzw := gzip.NewWriter(wb)

	jsonData, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("could not marshal deployment: %v", err)
	}

	writeHeader := func(n int) error {
		// Determine the size of the marshalled deployment struct and write it to
		// a header.
		header := make([]byte, 8)
		binary.BigEndian.PutUint32(header[4:], uint32(n))
		if _, err = gzw.Write(header); err != nil {
			return fmt.Errorf("error when writing header: %v", err)
		}
		return nil
	}

	if err := writeHeader(len(jsonData)); err != nil {
		return err
	}
	if _, err := gzw.Write(jsonData); err != nil {
		return err
	}
	if err := writeHeader(len(dockerFile)); err != nil {
		return err
	}
	if _, err := gzw.Write(dockerFile); err != nil {
		return err
	}

	if _, err = io.Copy(gzw, bundleReader); err != nil {
		return fmt.Errorf("error copying bundle: %v", err)
	}
	if err = gzw.Close(); err != nil {
		return err
	}
	return wb.Flush()
}

// decodeDeployment reads a bundle and a deployment struct from a
// deployment request.
// Use encodeDeployment to construct the request.
func decodeDeployment(r io.Reader, bundleWriter io.Writer) (dockerFile []byte, d *DeployInfo, err error) {

	readNext := func(r io.Reader) ([]byte, error) {
		header := make([]byte, 8)
		if _, err := io.ReadFull(r, header); err != nil {
			return nil, fmt.Errorf("error reading header: %v", err)
		}
		n := binary.BigEndian.Uint32(header[4:])

		data := make([]byte, n)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, fmt.Errorf("error reading data: %v", err)
		}		

		return data, nil
	}

	rb := bufio.NewReader(r)
	gzr, err := gzip.NewReader(rb)
	if err != nil {
		return nil, nil, err
	}
	defer gzr.Close()

	jsonData, err := readNext(gzr)
	if err != nil {
		return nil, nil, err
	}

	dockerFile, err = readNext(gzr)
	if err != nil {
		return nil, nil, err
	}

	var info DeployInfo

	if err = json.Unmarshal(jsonData, &info); err != nil {
		return nil, nil, fmt.Errorf("could not unmarshal deployment struct: %v", err)
	}

	// Copy the bundle.
	if _, err = io.Copy(bundleWriter, gzr); err != nil {
		return nil, nil, fmt.Errorf("could not copy bundle: %v", err)
	}
	return dockerFile, &info, nil
}

func (mps *MPS) handlePredict(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id provided: "+err.Error(), http.StatusBadRequest)
		return
	}
	var k *kernel
	mps.mu.Lock()
	deployment, ok := mps.deployments[id]
	if ok {
		k, ok = deployment.Kernel()
	}
	mps.mu.Unlock()
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"model not found"}`))
		return
	}
	k.ServeHTTP(w, r)
}
