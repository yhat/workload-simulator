package mps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/websocket"
)

func TestMockContainer(t *testing.T) {
	conn := newMockContainer()
	k, err := newKernel(conn, ioutil.Discard)
	if err != nil {
		t.Error("could not start kernel: %v", err)
		return
	}
	if err = k.heartbeat(); err != nil {
		t.Errorf("heartbeat failed: %v", err)
		return
	}
	pred := map[string]interface{}{
		"name":    "bigdatabob",
		"yhat_id": "foo",
	}
	resp, err := k.Predict(pred, nil)
	if err != nil {
		t.Errorf("failed to make prediction: %v", err)
	}
	id, ok := resp["yhat_id"].(string)
	if ok {
		if id != "foo" {
			t.Errorf("expected yhat_id to be 'foo' got '%s'", id)
		}
	} else {
		t.Errorf("no yhat_id in response")
	}
	if _, ok := resp["result"]; !ok {
		t.Error("no result in model response")
	}
}

func testMockMPSWithDeployment(t *testing.T, f func(m *MPS, c *MPSClient, deploymentId int64)) {
	mps, err := NewMPSMock()
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

func TestMPSMockDeployment(t *testing.T) {
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

func TestMPSMockPredict(t *testing.T) {
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

func TestMPSMockWebSocketPredict(t *testing.T) {

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
