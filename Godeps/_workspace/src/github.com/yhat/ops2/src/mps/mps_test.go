package mps

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestEncodeDeployment(t *testing.T) {
	d := DeployInfo{
		Username:      "bigdatabob",
		Modelname:     "foo",
		Lang:          R,
		CondaChannels: []string{"foobar", "barfoo"},
		CRANMirror:    YhatCRANMirror,
	}
	bundle := make([]byte, 2048)
	if _, err := io.ReadFull(rand.Reader, bundle); err != nil {
		t.Fatal(err)
	}

	// stand in for the network connection
	var networkConn bytes.Buffer

	dockerFile, err := createDockerfile(&d)
	if err != nil {
		t.Errorf("failed to create dockerfile: %v", err)
		return
	}

	err = encodeDeployment(&networkConn, dockerFile, &d, bytes.NewReader(bundle))
	if err != nil {
		t.Fatalf("failed to send deployment: %v", err)
	}

	var bundleWriter bytes.Buffer
	_, d2, err := decodeDeployment(&networkConn, &bundleWriter)
	if err != nil {
		t.Fatalf("failed to receive deployment: %v", err)
	}

	fields := []struct {
		Name, Exp, Act string
	}{
		{"Username", d.Username, d2.Username},
		{"Modelname", d.Modelname, d2.Modelname},
	}
	for _, f := range fields {
		if f.Exp != f.Act {
			t.Errorf("expected %s to be '%s' received '%s'", f.Name, f.Exp, f.Act)
		}
	}
	b2 := bundleWriter.Bytes()
	if n, m := len(bundle), len(b2); n != m {
		t.Errorf("received bundle was of size %d expected %d", m, n)
	}
	if 0 != bytes.Compare(bundle, b2) {
		t.Errorf("bundles where different")
	}
}

func TestMPSWithClient(t *testing.T) {
	mps, err := NewMPS()
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(mps)
	defer s.Close()

	client, err := NewMPSClient(s.URL, nil)
	if err != nil {
		t.Errorf("could not construct client: %v", err)
		return
	}
	stats, err := client.Status()
	if err != nil {
		t.Errorf("status request failed: %v", err)
	}
	if err := client.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}
	if n := len(stats.Deployments); n != 0 {
		t.Errorf("expected no deployments, got %d", n)
	}
}

func testMPSWithDeployment(t *testing.T, f func(m *MPS, c *MPSClient, deploymentId int64)) {
	requiresImage(t, "yhat/scienceops-r:0.0.2")

	mps, err := NewMPS()
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(mps)
	defer s.Close()

	client, err := NewMPSClient(s.URL, nil)
	if err != nil {
		t.Errorf("could not construct client: %v", err)
		return
	}
	d := DeployInfo{
		Username:  "eric",
		Modelname: "hellor",
		Lang:      R,
	}
	b := "bundles/r-bundle.json"

	id := int64(10)
	if err := client.Deploy(id, &d, b); err != nil {
		t.Errorf("could not deploy bundle: %v", err)
		return
	}
	defer func() {
		if err := client.Destroy(id); err != nil {
			t.Error("could not stop deployment")
		}
	}()
	f(mps, client, id)
}

func TestMPSDeployment(t *testing.T) {
	test := func(mps *MPS, client *MPSClient, id int64) {
		err := client.Heartbeat(id)
		if err != nil {
			t.Errorf("heartbeat request failed: %v", err)
			return
		}
		return
	}
	testMPSWithDeployment(t, test)
}

func TestMPSLogs(t *testing.T) {
	test := func(mps *MPS, client *MPSClient, id int64) {
		logLines, err := client.Logs()
		if err != nil {
			t.Errorf("could not get lines for logs: %v", err)
			return
		}
		for _, line := range logLines {
			if line.InstanceId != id {
				t.Errorf("expected id to be '%s' got '%s'", id, line.InstanceId)
			}
		}
		return
	}
	testMPSWithDeployment(t, test)
}

func TestMPSStatus(t *testing.T) {
	test := func(mps *MPS, client *MPSClient, id int64) {
		status, err := client.Status()
		if err != nil {
			t.Errorf("heartbeat request failed: %v", err)
			return
		}
		if n := len(status.Deployments); n != 1 {
			t.Errorf("expected one deployment, got %d", n)
			return
		}
		d := status.Deployments[0]
		if d.Id != id {
			t.Errorf("expected deployment id '%s' got '%s'", id, d.Id)
		}
		if !d.Ready {
			t.Errorf("deployment is not ready")
		}
	}
	testMPSWithDeployment(t, test)
}

func TestMPSPredict(t *testing.T) {
	test := func(mps *MPS, client *MPSClient, id int64) {
		body := bytes.NewBuffer([]byte(`{"name":"eric"}`))
		req, err := http.NewRequest("POST", "/eric/models/helloworld", body)
		if err != nil {
			t.Errorf("could not create request: %v", err)
			return
		}
		rr := httptest.NewRecorder()

		client.Predict(rr, req, id)
		if rr.Code != http.StatusOK {
			t.Errorf("bad response from model: %v", err)
			return
		}

		var resp struct {
			R struct {
				G []string `json:"greeting"`
			} `json:"result"`
			Model string `json:"yhat_model"`
		}
		if err = json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Errorf("could not decode response from model: %v", err)
			return
		}
		if resp.Model != "hellor" {
			t.Errorf("expected model name to be 'hellopy' got '%s'", resp.Model)
		}
		g := ""
		if len(resp.R.G) == 1 {
			g = resp.R.G[0]
		}
		if g != "hello!" {
			t.Errorf("expected greeting to be 'Hello eric!' got '%s'", g)
		}
		return
	}
	testMPSWithDeployment(t, test)
}

func TestMPSWebSocketPredict(t *testing.T) {

	verifyResp := func(id int, data []byte) {
		idStr := strconv.Itoa(id)
		var resp struct {
			Result struct {
				Greeting []string `json:"greeting"`
			} `json:"result"`
			Id   string `json:"yhat_id"`
			Name string `json:"yhat_model"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			t.Errorf("could not decode response '%s' %v", data, err)
			return
		}
		if resp.Id != idStr {
			t.Errorf("expected model id '%s' got '%s'", idStr, resp.Id)
		}
		if resp.Name != "hellor" {
			t.Errorf("expected model name to be 'hellopy' got '%s'", resp.Name)
		}
		if len(resp.Result.Greeting) == 0 {
			t.Errorf("got no response")
		}
	}

	test := func(mps *MPS, client *MPSClient, id int64) {

		hf := func(w http.ResponseWriter, r *http.Request) {
			client.Predict(w, r, id)
		}
		s := httptest.NewServer(http.HandlerFunc(hf))
		defer s.Close()
		u := strings.Replace(s.URL, "http", "ws", 1)
		ws, err := websocket.Dial(u, "", "http://localhost/")
		if err != nil {
			t.Errorf("could not create websocket connection: %v", err)
			return
		}
		defer ws.Close()
		resp := make([]byte, 2048)
		for i := 0; i < 5; i++ {
			req := []byte(fmt.Sprintf(`{"name":"eric","yhat_id":"%d"}`, i))
			if _, err = ws.Write(req); err != nil {
				t.Errorf("could not write to websocket connection: %v", err)
				return
			}
			n := 0
			for n == 0 {
				n, err = ws.Read(resp)
				if err != nil {
					t.Errorf("could not read from websocket connection: %v", err)
					return
				}
			}
			verifyResp(i, resp[:n])
		}
		return
	}
	testMPSWithDeployment(t, test)
}

func TestDestroyDeployment(t *testing.T) {

	requiresImage(t, "yhat/scienceops-r:0.0.2")

	mps, err := NewMPS()
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(mps)
	defer s.Close()

	client, err := NewMPSClient(s.URL, nil)
	if err != nil {
		t.Errorf("could not construct client: %v", err)
		return
	}
	d := DeployInfo{Username: "eric", Modelname: "hellor", Lang: R}
	b := "bundles/r-bundle.json"

	mathrand.Seed(int64(time.Now().UnixNano()))

	// randTime returns a duration between 0 and 1 second
	randTime := func(max time.Duration) time.Duration {
		return time.Duration(mathrand.Intn(int(max)))
	}

	for i := 0; i < 5; i++ {

		id := int64(i)
		wg := new(sync.WaitGroup)
		wg.Add(2)
		var destroyErr error
		go func() {
			time.Sleep(randTime(time.Second))
			_ = client.Deploy(id, &d, b)
			wg.Done()
		}()
		go func() {
			time.Sleep(randTime(time.Second))
			destroyErr = client.Destroy(id)
			wg.Done()
		}()
		wg.Wait()
		if destroyErr != nil && destroyErr != ErrNotFound {
			t.Errorf("unexpected error from destory: %v", err)
		}
		status, err := client.Status()
		if err != nil {
			t.Errorf("bad status from client status: %v", err)
			continue
		}
		if n := len(status.Deployments); n != 0 {
			t.Errorf("expected no deployments, got %d", n)
		}
	}
}
