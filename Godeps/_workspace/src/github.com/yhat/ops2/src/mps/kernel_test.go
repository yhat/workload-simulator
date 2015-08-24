package mps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/yhat/go-docker"
)

func testRKernel(t testing.TB, test func(k *kernel)) {
	requiresImage(t, "yhat/scienceops-r:0.0.2")
	requiresCommand(t, "docker")
	d := DeployInfo{
		Username: "eric", Modelname: "hellor", Version: 4, Lang: R,
	}
	bundle := "bundles/r-bundle.json"
	testKernel(t, &d, bundle, test)
}

func testPyKernel(t testing.TB, test func(k *kernel)) {
	requiresImage(t, "yhat/scienceops-python:0.0.2")
	requiresCommand(t, "docker")
	d := DeployInfo{
		Username: "eric", Modelname: "hellopy", Version: 4, Lang: Python2,
	}
	bundle := "bundles/py-bundle.json"
	testKernel(t, &d, bundle, test)
}

func testKernel(t testing.TB, d *DeployInfo, bundle string, test func(k *kernel)) {

	cli, err := docker.NewDefaultClient(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	imgName := randSeq(16)
	containerName := randSeq(16)

	var stderr bytes.Buffer

	dockerFile, err := createDockerfile(d)
	if err != nil {
		t.Errorf("failed to create dockerfile: %v", err)
		return
	}

	if err := buildImage(dockerFile, bundle, imgName, &stderr); err != nil {
		t.Errorf("could not build image: %v %s", err, stderr.String())
		return
	}
	defer exec.Command("docker", "rmi", imgName).Run()
	stderr.Reset()

	cid, conn, err := startContainer(imgName, containerName)
	if err != nil {
		t.Errorf("could not start container: %v", err)
		return
	}
	defer exec.Command("docker", "rm", "-f", cid).Run()

	errc := make(chan error, 1)
	go func() {
		cli.Wait(cid)
		errc <- fmt.Errorf("container exited")
	}()

	go func() {
		<-time.After(10 * time.Second)
		errc <- fmt.Errorf("prediction timed out")
	}()

	go func() {
		kernel, err := newKernel(conn, &stderr)
		if err != nil {
			errc <- fmt.Errorf("could not init model: %v %s", err, stderr.String())
			return
		}
		test(kernel)
		errc <- nil
	}()
	if err = <-errc; err != nil {
		t.Error(err)
	}
}

func TestOneRPrediction(t *testing.T) {
	test := func(k *kernel) {
		resp, err := k.Predict(map[string]string{"name": "eric"}, nil)

		if err != nil {
			t.Errorf("could not make prediction: %v", err)
			return
		}
		if _, ok := resp["result"]; !ok {
			t.Errorf("bad result from model")
			return
		}
	}
	testRKernel(t, test)
}

// Test to make sure that if the user passes a request with "yhat_id" that
// they get a response with the same "yhat_id"
func TestYhatId(t *testing.T) {
	test := func(k *kernel) {

		yhatId := "a random id"
		var req interface{}
		err := json.Unmarshal([]byte(`{"name:":"eric","yhat_id":"a random id"}`), &req)
		if err != nil {
			t.Errorf("could not encode request: %v", err)
			return
		}

		resp, err := k.Predict(req, nil)

		if err != nil {
			t.Errorf("could not make prediction: %v", err)
			return
		}
		respId, ok := resp["yhat_id"]
		if !ok {
			t.Errorf("response did not have a yhat_id!")
			return
		}
		if yhatId != respId {
			t.Errorf("response id did not match request id '%s' vs '%s'", respId, yhatId)
		}
	}
	testRKernel(t, test)
}

func TestOnePyPrediction(t *testing.T) {
	test := func(k *kernel) {
		resp, err := k.Predict(map[string]string{"name": "eric"}, nil)

		if err != nil {
			t.Errorf("could not make prediction: %v", err)
			return
		}
		if _, ok := resp["result"]; !ok {
			t.Errorf("bad result from model")
			return
		}
	}
	testPyKernel(t, test)
}

func TestOneRHTTPPrediction(t *testing.T) {

	test := func(k *kernel) {
		body := bytes.NewBuffer([]byte(`{"name":"eric"}`))
		req, err := http.NewRequest("POST", "/eric/models/helloworld", body)
		if err != nil {
			t.Errorf("could not create request: %v", err)
			return
		}
		rr := httptest.NewRecorder()
		k.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("bad response from model: %v", err)
			return
		}
		var resp struct {
			R struct {
				G []string `json:"greeting"`
			} `json:"result"`
			Id    string `json:"yhat_id"`
			Model string `json:"yhat_model"`
		}
		if err = json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Errorf("could not decode response: %v", err)
			return
		}
		if len(resp.R.G) != 1 || resp.R.G[0] != "hello!" {
			t.Errorf("bad response from model %s", resp.R.G)
		}
	}
	testRKernel(t, test)
}

func TestOnePyHTTPPrediction(t *testing.T) {
	test := func(k *kernel) {
		body := bytes.NewBuffer([]byte(`{"name":"eric"}`))
		req, err := http.NewRequest("POST", "/eric/models/helloworld", body)
		if err != nil {
			t.Errorf("could not create request: %v", err)
			return
		}
		rr := httptest.NewRecorder()
		k.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("bad response from model: %v", err)
			return
		}
		var resp struct {
			R struct {
				G string `json:"greeting"`
			} `json:"result"`
			YhatID    string `json:"yhat_id"`
			YhatModel string `json:"yhat_model"`
		}
		if err = json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Errorf("could not decode response: %v", err)
			return
		}
		if resp.R.G != "Hello eric!" {
			t.Errorf("bad response from model %s", resp.R.G)
		}
	}
	testPyKernel(t, test)
}

func TestRWebSocketPrediction(t *testing.T) {

	verifyResp := func(id int, data []byte) {
		idStr := strconv.Itoa(id)
		var resp struct {
			Result struct {
				Greeting string `json:"greeting"`
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
		if resp.Name != "hellopy" {
			t.Errorf("expected model name to be 'hellopy' got '%s'", resp.Name)
		}
		if resp.Result.Greeting == "" {
			t.Errorf("got no response")
		}
	}

	test := func(k *kernel) {
		s := httptest.NewServer(k)
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

	}
	testPyKernel(t, test)
}

func TestRHeartbeat(t *testing.T) {
	test := func(k *kernel) {
		if err := k.heartbeat(); err != nil {
			t.Error(err)
		}
	}
	testRKernel(t, test)
}

func TestPyHeartbeat(t *testing.T) {
	test := func(k *kernel) {
		if err := k.heartbeat(); err != nil {
			t.Error(err)
		}
	}
	testPyKernel(t, test)

}

func benchmarkPredict(b *testing.B, k *kernel) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		data := map[string]string{"name": "eric"}
		result, err := k.Predict(data, nil)
		if err != nil {
			b.Error(fmt.Errorf("could not make prediction: %v", err))
			return
		}
		if _, ok := result["result"]; !ok {
			b.Error(fmt.Errorf("bad result from model"))
			return
		}
	}
	b.StopTimer()
}

func BenchmarkRPredict(b *testing.B) {
	test := func(k *kernel) { benchmarkPredict(b, k) }
	testRKernel(b, test)
}

func BenchmarkPyPredict(b *testing.B) {
	test := func(k *kernel) { benchmarkPredict(b, k) }
	testPyKernel(b, test)
}
