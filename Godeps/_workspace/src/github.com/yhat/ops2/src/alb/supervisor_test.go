package alb

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/yhat/ops2/src/db"
	"github.com/yhat/ops2/src/mps"
	"github.com/yhat/ops2/src/mps/tlsconfig"
)

// value is re0instantiated at the beginning of each supervisor test
var testStorage *mockStorage

func testSupervisor(t *testing.T, nWorkers int, test func(*Supervisor)) {
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tempdir)

	client, err := tlsconfig.NewClient(tempdir)
	if err != nil {
		t.Error(err)
		return
	}

	testStorage = NewTestStorage()
	c := SupervisorConfig{
		Storage:       testStorage,
		TLSHandshaker: client,
		// Discard logs for tests
		Logger: log.New(ioutil.Discard, "", 0),
	}
	supervisor, err := NewSupervisor(c)
	if err != nil {
		t.Errorf("failed to create supervisor: %v", err)
		return
	}
	// discard logs for tests
	supervisor.Logger = log.New(ioutil.Discard, "", 0)

	for i := 0; i < nWorkers; i++ {
		mps, err := mps.NewMPS()
		if err != nil {
			t.Error(err)
			return
		}
		defer mps.Reset()
		// configure server's TLS config directly rather than with a handshake
		server := httptest.NewUnstartedServer(mps)
		server.TLS, err = client.ServerConfig(net.ParseIP("127.0.0.1"))
		if err != nil {
			t.Error(err)
			return
		}
		server.StartTLS()
		defer server.Close()

		addr := strings.TrimPrefix(server.URL, "https://")

		if err := supervisor.AddWorker(addr, int64(i)); err != nil {
			t.Errorf("could not add worker: %v", err)
			return
		}
	}
	test(supervisor)
}

func TestSupervisorAddWorker(t *testing.T) {
	testSupervisor(t, 5, func(super *Supervisor) {
		if n := len(super.pool.Workers()); n != 5 {
			t.Errorf("expected 5 workers, got %d", n)
		}
	})
}

func TestSupervisorDeploy(t *testing.T) {
	test := func(super *Supervisor) {
		if err := super.Deploy("bigdatabob", "hellopy", 1); err != nil {
			t.Errorf("could not deploy: %v", err)
			return
		}
	}
	testSupervisor(t, 2, test)
}

func TestSupervisorRedeploy(t *testing.T) {
	test := func(super *Supervisor) {
		for version := 1; version < 4; version++ {
			if err := super.Deploy("bigdatabob", "hellopy", version); err != nil {
				t.Errorf("could not deploy version %d: %v", version, err)
				return
			}
		}
	}
	testSupervisor(t, 2, test)
}

func TestSupervisorLogs(t *testing.T) {
	test := func(super *Supervisor) {
		tempdir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Errorf("could not create tempdir: %v", err)
			return
		}
		defer os.RemoveAll(tempdir)

		totalExp := 3
		nInstances := 3
		testStorage.modelRep = nInstances

		checkNumUnique := func(lines []*mps.LogLine, exp int, msg string) {
			m := map[int64]struct{}{}

			for _, line := range lines {
				m[line.InstanceId] = struct{}{}
			}
			if len(m) != exp {
				t.Errorf("%s: expected logs for %d instances, got: %d", msg, exp, len(m))
			}
		}

		for i := 0; i < 3; i++ {

			deployId := testStorage.nextDeployId
			version := 3
			err := super.Deploy("bigdatabob", "hellopy", version)
			if err != nil {
				t.Errorf("could not deploy version %d: %v", version, err)
				return
			}

			super.WriteLogs(tempdir)

			lines, err := ReadDeploymentLogs(tempdir, "bigdatabob", "hellopy", deployId)
			if err != nil {
				t.Errorf("could not parse logs: %v", err)
				return
			}
			checkNumUnique(lines, nInstances, "checking deployment logs")

			lines, err = ReadModelLogs(tempdir, "bigdatabob", "hellopy")
			if err != nil {
				t.Errorf("could not parse logs: %v", err)
				return
			}
			checkNumUnique(lines, totalExp, "checking model logs")

			totalExp += nInstances
		}
	}
	testSupervisor(t, 2, test)
}

func TestSupervisorPredict(t *testing.T) {
	test := func(super *Supervisor) {
		user, model := "bigdatabob", "hellopy"
		if err := super.Deploy(user, model, 1); err != nil {
			t.Errorf("could not deploy: %v", err)
			return
		}
		hf := func(w http.ResponseWriter, r *http.Request) {
			super.Predict(user, model, w, r)
		}
		s := httptest.NewServer(http.HandlerFunc(hf))
		defer s.Close()
		for i := 0; i < 100; i++ {
			body := bytes.NewReader([]byte(`{"name":"bigdatabob"}`))
			resp, err := http.Post(s.URL, "application/json", body)
			if err != nil {
				t.Errorf("request failed: %v", err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %s", resp.Status)
				return
			}
			_, err = ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Errorf("could not read body: %v", err)
				return
			}
		}
	}
	testSupervisor(t, 2, test)
}

func predict(h http.Handler) *httptest.ResponseRecorder {
	data := `{"name":"bigdatabob"}`
	body := bytes.NewReader([]byte(data))
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/bigdatabob/models/hellopy"},
		Header: map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(len(data)),
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestSupervisorSleep(t *testing.T) {
	test := func(super *Supervisor) {
		user, model := "bigdatabob", "hellopy"
		if err := super.Deploy(user, model, 1); err != nil {
			t.Errorf("could not deploy: %v", err)
			return
		}
		hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			super.Predict(user, model, w, r)
		})
		rr := predict(hf)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d %s", rr.Code, rr.Body.String())
			return
		}

		super.Sleep(user, model)

		rr = predict(hf)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d %s", rr.Code, rr.Body.String())
			return
		}
	}
	testSupervisor(t, 2, test)
}

func TestSupervisorWake(t *testing.T) {
	test := func(super *Supervisor) {
		user, model := "bigdatabob", "hellopy"
		if err := super.Deploy(user, model, 1); err != nil {
			t.Errorf("could not deploy: %v", err)
			return
		}
		hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			super.Predict(user, model, w, r)
		})
		rr := predict(hf)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code, rr.Body.String())
			return
		}

		super.Sleep(user, model)

		rr = predict(hf)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code, rr.Body.String())
			return
		}

		if err := super.Wake(user, model); err != nil {
			t.Errorf("wake failed: %v", err)
			return
		}

		rr = predict(hf)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 after wake, got %d", rr.Code, rr.Body.String())
			return
		}
	}
	testSupervisor(t, 2, test)
}

func TestSupervisorRestart(t *testing.T) {
	test := func(super *Supervisor) {
		user, model := "bigdatabob", "hellopy"
		if err := super.Deploy(user, model, 1); err != nil {
			t.Errorf("could not deploy: %v", err)
			return
		}
		hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			super.Predict(user, model, w, r)
		})
		rr := predict(hf)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
			return
		}

		if err := super.Restart(user, model); err != nil {
			t.Errorf("restart failed: %v", err)
			return
		}

		rr = predict(hf)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 after wake, got %d", rr.Code)
			return
		}
	}
	testSupervisor(t, 2, test)
}

func TestSupervisorDeleteUser(t *testing.T) {
	test := func(super *Supervisor) {

		testPredict := func(user, model string, expStatusCode int) error {
			data := strings.NewReader(`{"name":"foo"}`)
			req, err := http.NewRequest("POST", "/", data)
			if err != nil {
				return err
			}

			rr := httptest.NewRecorder()
			super.Predict(user, model, rr, req)
			if expStatusCode != rr.Code {
				return fmt.Errorf("for prediction %s:%s expected %d got %d: %s",
					user, model, expStatusCode, rr.Code, rr.Body.String())
			}
			return nil
		}

		models := []string{"hellopy_1", "hellopy_2", "hellopy_3"}
		users := []string{"hadoopheather", "bigdatabob"}
		for _, model := range models {
			for _, user := range users {
				if err := super.Deploy(user, model, 1); err != nil {
					t.Errorf("could not deploy: %v", err)
					return
				}
			}
		}

		for _, user := range users {
			for _, model := range models {
				if err := testPredict(user, model, http.StatusOK); err != nil {
					t.Error(err)
				}
			}
			super.deleteUser(user, true)
			for _, model := range models {
				if err := testPredict(user, model, http.StatusInternalServerError); err != nil {
					t.Error(err)
				}
			}
		}
	}
	testSupervisor(t, 2, test)
}

// shutdown acts as if a supervisor has been shut down so that it will
// create nil pointer panics if something tries to access it.
// This is a test only method.
func (s *Supervisor) shutdown() {
	s.Logger = nil
	s.mu = nil
	s.pool = nil
	s.routes = nil
	s.deployments = nil
	s.asleep = nil
	s.deploymentPool = nil
	s.instMu = nil
	s.insts = nil
	s.tlsHandshaker = nil
}

func TestSupervisorReboot(t *testing.T) {

	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tempdir)

	client, err := tlsconfig.NewClient(tempdir)
	if err != nil {
		t.Error(err)
		return
	}

	testStorage = NewTestStorage()
	c := SupervisorConfig{
		Storage:       testStorage,
		TLSHandshaker: client,
		// Discard logs for tests
		Logger: log.New(ioutil.Discard, "", 0),
	}
	supervisor1, err := NewSupervisor(c)
	if err != nil {
		t.Errorf("failed to create supervisor: %v", err)
		return
	}

	addrs := [5]string{"", "", "", "", ""}

	for i := 0; i < 5; i++ {
		mps, err := mps.NewMPS()
		if err != nil {
			t.Error(err)
			return
		}
		defer mps.Reset()
		// configure server's TLS config directly rather than with a handshake
		server := httptest.NewUnstartedServer(mps)
		server.TLS, err = client.ServerConfig(net.ParseIP("127.0.0.1"))
		if err != nil {
			t.Error(err)
			return
		}
		server.StartTLS()
		defer server.Close()

		addrs[i] = strings.TrimPrefix(server.URL, "https://")
	}

	type Model struct {
		User, Model string
		Version     int
		Instances   []int64
	}
	deployments := map[int64]*Model{}

	for i, addr := range addrs {
		if err := supervisor1.AddWorker(addr, int64(i)); err != nil {
			t.Errorf("could not add worker: %v", err)
			return
		}
	}

	user := "bigdatabob"
	// Deploy 5 different models
	modelnames := []string{
		"hellor_0", "hellor_1", "hellor_2", "hellor_3", "hellor_4",
	}

	// Override the storage's NewDeployment function so we can track which
	// deployment is associated with which model version.
	// This is normally done by the database.
	nInstances := 2
	var nextId int64 = 1
	var nextDeployId int64 = 1
	testStorage.newDeployment = func(user, model string, version int) (deployId int64, instIds []int64, err error) {
		deployId = nextDeployId
		nextDeployId++

		ids := make([]int64, nInstances)
		instIds = make([]int64, nInstances)
		for i := range ids {
			ids[i] = nextId
			instIds[i] = nextId
			nextId++
		}

		deployments[deployId] = &Model{user, model, version, ids}
		return deployId, instIds, nil
	}
	for _, model := range modelnames {
		if err := supervisor1.Deploy(user, model, 1); err != nil {
			t.Errorf("could not deploy: %v", err)
			return
		}
	}

	nExp := len(modelnames)

	if n := len(deployments); n != nExp {
		t.Errorf("expected %d deployments, got: %d", nExp, n)
		return
	}
	for _, model := range deployments {
		if n := len(model.Instances); n != nInstances {
			t.Errorf("expected %d deployments per model, got %d", nInstances, n)
			return
		}
	}

	testPred := func(super *Supervisor) {
		for _, model := range modelnames {
			hf := func(w http.ResponseWriter, r *http.Request) {
				super.Predict(user, model, w, r)
			}
			s := httptest.NewServer(http.HandlerFunc(hf))
			defer s.Close()
			for i := 0; i < 100; i++ {
				body := bytes.NewReader([]byte(`{"name":"bigdatabob"}`))
				resp, err := http.Post(s.URL, "application/json", body)
				if err != nil {
					t.Errorf("request failed: %v", err)
					return
				}
				respbody, err := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					t.Errorf("could not read body: %v", err)
					return
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("expected 200, got %s: %s", resp.Status, respbody)
					return
				}
			}
		}
	}
	testPred(supervisor1)

	supervisor1.shutdown()

	workers := make([]db.Worker, len(addrs))
	for i, addr := range addrs {
		workers[i] = db.Worker{int64(i), addr}
	}

	deploymentReqs := make([]db.DeploymentReq, len(deployments))
	i := 0
	for deployId, model := range deployments {
		deploymentReqs[i] = db.DeploymentReq{
			Username:         model.User,
			Modelname:        model.Model,
			Version:          model.Version,
			LastDeployId:     deployId,
			ValidInstanceIds: model.Instances,
		}
		i++
	}

	c2 := SupervisorConfig{
		Storage:       testStorage,
		TLSHandshaker: client,
		Workers:       workers,
		Deployments:   deploymentReqs,
		// Discard logs for tests
		Logger: log.New(ioutil.Discard, "", 0),
	}
	supervisor2, err := NewSupervisor(c2)
	if err != nil {
		t.Errorf("failed to start second supervisor: %v", err)
		return
	}
	// discard logs for tests
	supervisor2.Logger = log.New(ioutil.Discard, "", 0)

	testPred(supervisor2)
}
