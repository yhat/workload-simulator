package alb

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/yhat/ops2/src/mps"
)

func requiresImage(t *testing.T, img string) {
	if err := exec.Command("docker", "inspect", img).Run(); err != nil {
		t.Fatalf("docker image %s not found", img)
	}
}

func TestWorkerPoolInit(t *testing.T) {
	testPool(t, 10, func(*WorkerPool) {})
}

func TestWorkerPoolOneBuild(t *testing.T) {
	testPoolWithBuild(t, func(*WorkerPool, int64) {})
}

func TestWorkerPoolPredict(t *testing.T) {
	test := func(pool *WorkerPool, id int64) {

		handler, ok := pool.Predict(id)
		if !ok {
			t.Errorf("instance not found")
			return
		}

		body := bytes.NewBuffer([]byte(`{"name":"bigdatabob"}`))

		req, err := http.NewRequest("POST", "/", body)
		if err != nil {
			t.Error(err)
			return
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var resp struct {
			R struct {
				G string `json:"greeting"`
			} `json:"result"`
		}
		b := rr.Body.Bytes()
		if err := json.Unmarshal(b, &resp); err != nil {
			t.Errorf("could not unmarshal result: %v", err)
			return
		}
		if resp.R.G != "Hello bigdatabob!" {
			t.Errorf("unexpected result from instance: %s", b)
		}
	}
	testPoolWithBuild(t, test)
}

func TestWorkerPoolReAdd(t *testing.T) {
	pool := NewPool(nil)

	workerURLs := map[int64]string{}

	for i := 0; i < 5; i++ {
		mps, err := mps.NewMPS()
		if err != nil {
			t.Error(err)
			return
		}
		s := httptest.NewServer(mps)
		defer s.Close()
		workerId := int64(i)
		workerURLs[workerId] = s.URL
		if err := pool.Add(s.URL, workerId); err != nil {
			t.Error(err)
			return
		}
		defer func(workerId int64) {
			if err := pool.RemoveWorker(workerId); err != nil {
				t.Errorf("error removing worker %s: %v", workerId, err)
			}
		}(workerId)

		if err = pool.PingWorker(workerId); err != nil {
			t.Errorf("worker heartbeat failed: %v", err)
			return
		}
	}

	id := int64(123)
	_, err := pool.QueueBuild(id)
	if err != nil {
		t.Errorf("failed to queue build: %v", err)
		return
	}
	defer func() {
		if err := pool.RemoveInstance(id); err != nil {
			t.Errorf("could not remove instance: %v", err)
		}
	}()

	info := mps.DeployInfo{
		Username:  "bigdatabob",
		Modelname: "hellopy",
		Lang:      mps.Python2,
	}
	if err := pool.Build(id, &info, "../mps/bundles/py-bundle.json"); err != nil {
		t.Errorf("could not build instance: %v", err)
		return
	}

	if err := pool.PingInstance(id); err != nil {
		t.Errorf("heartbeat failed: %v", err)
		return
	}
	workers := pool.Workers()
	for _, worker := range workers {
		if _, ok := workerURLs[worker]; !ok {
			t.Errorf("worker not in original group: %d", worker)
		}

		if err := pool.ReleaseWorker(worker); err != nil {
			t.Errorf("could not release worker: %v", err)
			return
		}
	}

	if n := len(pool.Instances()); n != 0 {
		t.Errorf("expected 0 instance in pool, got %d", n)
	}

	if err := pool.PingInstance(id); err == nil {
		t.Errorf("did not expect to be able to ping instance")
	}
	for worker, workerURL := range workerURLs {
		if err := pool.Add(workerURL, worker); err != nil {
			t.Errorf("could not add worker %s:%d: %v", workerURL, worker, err)
		}
	}
	if n := len(pool.Instances()); n != 1 {
		t.Errorf("expected one instance in pool, got %d", n)
	}

	if err := pool.PingInstance(id); err != nil {
		t.Errorf("heartbeat after re add failed: %v", err)
		return
	}
}

func testPool(t *testing.T, nWorkers int, test func(pool *WorkerPool)) {
	pool := NewPool(nil)
	for i := 0; i < nWorkers; i++ {
		mps, err := mps.NewMPS()
		if err != nil {
			t.Error(err)
			return
		}
		s := httptest.NewServer(mps)
		defer s.Close()
		workerId := int64(i)
		if err := pool.Add(s.URL, workerId); err != nil {
			t.Error(err)
			return
		}
		defer func(workerId int64) {
			if err := pool.RemoveWorker(workerId); err != nil {
				t.Errorf("error removing worker %s: %v", workerId, err)
			}
		}(workerId)

		if err = pool.PingWorker(workerId); err != nil {
			t.Errorf("worker heartbeat failed: %v", err)
			return
		}
	}
	test(pool)
}

func testPoolWithBuild(t *testing.T, test func(p *WorkerPool, instId int64)) {

	requiresImage(t, "yhat/scienceops-python:0.0.2")

	testPool(t, 5, func(pool *WorkerPool) {
		id := int64(123)
		_, err := pool.QueueBuild(id)
		if err != nil {
			t.Errorf("failed to queue build: %v", err)
			return
		}
		defer func() {
			if err := pool.RemoveInstance(id); err != nil {
				t.Errorf("could not remove instance: %v", err)
			}
		}()

		info := mps.DeployInfo{
			Username:  "bigdatabob",
			Modelname: "hellopy",
			Lang:      mps.Python2,
		}
		if err := pool.Build(id, &info, "../mps/bundles/py-bundle.json"); err != nil {
			t.Errorf("could not build instance: %v", err)
			return
		}

		if err := pool.PingInstance(id); err != nil {
			t.Errorf("heartbeat failed: %v", err)
			return
		}
		test(pool, id)
	})
}
