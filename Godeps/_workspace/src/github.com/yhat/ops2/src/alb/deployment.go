package alb

import (
	"log"
	"sync"

	"golang.org/x/net/context"

	"github.com/yhat/ops2/src/mps"
)

type Deployment struct {
	// The expected number of instances behind a route.
	ExpNum int

	Username  string
	Modelname string
	Version   int

	// Information necessary for the deployment
	Info *mps.DeployInfo
	// Path to the bundle file
	Bundle string

	DeployId int64

	mu *sync.Mutex

	cancel context.CancelFunc
	ctx    context.Context

	instanceIds []int64
	pool        *WorkerPool
}

func NewDeployment(pool *WorkerPool) *Deployment {

	d := &Deployment{mu: new(sync.Mutex), pool: pool, instanceIds: []int64{}}

	d.ctx, d.cancel = context.WithCancel(context.Background())
	return d
}

func (r *Deployment) Kill() {
	// TODO: Logging
	r.cancel()
	for _, inst := range r.instanceIds {
		if err := r.pool.RemoveInstance(inst); err != nil {
			log.Println("could not remove instance: %v", err)
		}
	}
}

func (r *Deployment) IsCancelled() bool {
	select {
	case <-r.ctx.Done():
		return true
	default:
		return false
	}
}

func (r *Deployment) Instances() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]int64, len(r.instanceIds))
	copy(cp, r.instanceIds)
	return cp
}

func (d *Deployment) Monitor(user, model string, pool *WorkerPool) (toBuild int) {
	ids := d.Instances()
	toDelete := map[int64]struct{}{}
	nOk := 0
	for _, id := range ids {
		if err := pool.PingInstance(id); err != nil {
			log.Printf("instance heartbeat failed: %s", id)
			toDelete[id] = struct{}{}
		} else {
			nOk++
		}
	}
	n := 0
	d.mu.Lock()
	for _, id := range d.instanceIds {
		if _, ok := toDelete[id]; !ok {
			d.instanceIds[n] = id
			n++
		}
	}
	d.instanceIds = d.instanceIds[:n]
	d.mu.Unlock()
	for inst, _ := range toDelete {
		if err := pool.RemoveInstance(inst); err != nil {
			log.Printf("could not remove instance: %v", err)
		}
	}
	return d.ExpNum - nOk
}

func (d *Deployment) BuildInstances(ids []int64) (err error) {

	for _, instId := range ids {
		if _, err := d.pool.QueueBuild(instId); err != nil {
			return err
		}
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(ids))

	// done is a channel that closes when the function ends
	done := make(chan struct{})
	defer func() { close(done) }()

	// wait group waits for the instances to deploy successfully
	wgDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	errc := make(chan error)

	for _, instId := range ids {
		go func(id int64) {
			defer wg.Done()

			// attempt to build the instance
			if err := d.pool.Build(id, d.Info, d.Bundle); err != nil {
				// wait for the error channel to take or for the global
				// function to exit
				select {
				case errc <- err:
				case <-done:
				}
			}
		}(instId)

		defer func(instId int64) {
			// if the global function has exited unsuccessfully, remove each instance
			if err != nil {
				if err := d.pool.RemoveInstance(instId); err != nil {
					log.Printf("could not remove instance %d: %v", instId, err)
				}
			}
		}(instId)
	}

	select {
	case err = <-errc:
		// there was a failure
		return err
	case <-d.ctx.Done():
		// deployment was cancelled
		return ErrDeploymentCancelled
	case <-wgDone:
		return nil
	}
}
