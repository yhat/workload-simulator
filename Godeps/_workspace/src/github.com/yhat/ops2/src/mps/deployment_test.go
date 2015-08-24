package mps

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yhat/ops2/src/mps/integration"
)

func getLocalAddress(h http.Handler) (string, error) {
	cmd := exec.Command("ifconfig")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")

	var ipAddress string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 2 && fields[0] == "inet" {
			ipAddress = fields[1]
		}
	}

	if ipAddress == "" {
		return "", err
	}

	return ipAddress, nil
}

func NewMacTestServer(h http.Handler) (*httptest.Server, error) {
	// in order for boot2docker to connect to a server running on the local machine,
	// that server must listen on an ip address the boot2docker VM can find, not 127.0.0.1

	// Get ip address by parsing ifconfig
	ipAddr, err := getLocalAddress(h)
	if err != nil {
		return nil, err
	}

	// start a listener at that address
	listener, err := net.Listen("tcp", ipAddr+":0")
	if err != nil {
		return nil, err
	}

	// create a httptest server
	s := httptest.NewUnstartedServer(h)
	s.Listener = listener
	s.Start()
	return s, nil
}

func TestDeployHelloR(t *testing.T) {

	test := func(k *kernel) {
		input := map[string]interface{}{"name": "bigdatabob"}
		output, err := k.Predict(input, nil)
		if err != nil {
			t.Errorf("failed to make prediction: %v", err)
		}
		if err != nil {
			t.Errorf("failed to predict %s %v:", integration.RHello, err)
			return
		}
		data, err := json.Marshal(output)
		if err != nil {
			t.Errorf("could not marshal output: %v", err)
			return
		}
		var resp struct {
			Result struct {
				Greeting []string `json:"greeting"`
			} `json:"result"`
		}
		if err = json.Unmarshal(data, &resp); err != nil {
			t.Errorf("could not unmarshal output: %v", err)
			return
		}
		g := resp.Result.Greeting
		exp := "Hello bigdatabob !"
		if len(g) != 1 && g[0] != exp {
			t.Errorf("expected result to be '%s' got '%s'", exp, g)
		}
	}

	testDeployment(t, integration.RHello, test)
}

func TestDeployPyBeerRec(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping beer rec test for short tests")
	}

	test := func(k *kernel) {

		// test a utf-8 character
		input := map[string]interface{}{
			"beers": []string{"Black Horse Black Beer", "Rauch Ür Bock"},
		}

		output, err := k.Predict(input, nil)
		if err != nil {
			t.Error(err)
			return
		}
		data, err := json.Marshal(output)
		if err != nil {
			t.Errorf("could not marshal output: %v", err)
			return
		}
		var resp struct {
			Result []struct {
				Beer string
			}
		}
		if err = json.Unmarshal(data, &resp); err != nil {
			t.Errorf("could not unmarshal output: %v", err)
			return
		}
		if n := len(resp.Result); n != 15 {
			t.Errorf("expected 15 beers, got %d", n)
		}
	}

	testDeployment(t, integration.PyBeerRec, test)
}

func TestDeployWithSubpackage(t *testing.T) {

	test := func(k *kernel) {

		input := map[string]interface{}{
			"name": "bigdatabob",
		}

		output, err := k.Predict(input, nil)
		if err != nil {
			t.Error(err)
			return
		}
		data, err := json.Marshal(output)
		if err != nil {
			t.Errorf("could not marshal output: %v", err)
			return
		}
		var resp struct {
			Result struct {
				Greeting string `json:"greeting"`
			} `json:"result"`
		}
		if err = json.Unmarshal(data, &resp); err != nil {
			t.Errorf("could not unmarshal output: %v", err)
			return
		}
		g := resp.Result.Greeting
		exp := "Hello bigdatabob!"
		if g != exp {
			t.Errorf("expected result to be '%s' got '%s'", exp, g)
		}
	}

	testDeployment(t, integration.PyWithSubpackage, test)
}

// actual example from customer
// var rawInput = []byte(`{"student":{"student_id":1,"high_school_gpa":3.5,"transfer_student":"Y"},"terms":{"term_id":[1,2,3,4],"lifetime_credits_at_inst":[0,15,30,45],"lifetime_transfer_credits":[10,10,15,15],"lifetime_attempted_credits":[10,25,45,60],"cumulative_gpa":[3.5,3.2,3.1,3.3],"major_cd":[null,"PSY","ENG","ENG"]},"courses":{"term_id":[1,1,2,2,3,3,4,4],"course_cd":["ENGL1101","PSYC1101","ENGL1102","PSYC1102","ENGL1103","ENGL1104","ENGL1105","ENGL1106"],"grade":["A","B","A-","B","B","B","A","A"]}}`)

var rawInput = []byte(`{"bar":"baz","foo":[1,2,3]}`)

func TestRawInput(t *testing.T) {

	test := func(k *kernel) {

		input := make(map[string]interface{})
		if err := json.Unmarshal(rawInput, &input); err != nil {
			t.Errorf("could not unmarshal input: %v", err)
			return
		}
		output, err := k.Predict(input, nil)
		if err != nil {
			t.Error(err)
			return
		}
		if result, ok := output["result"]; ok {
			out, err := json.Marshal(result)
			if err != nil {
				t.Errorf("could not marshal output: %v", err)
				return
			}
			if bytes.Compare(rawInput, out) != 0 {
				t.Errorf("expected '%s' got '%s'", rawInput, out)
			}
		} else {
			t.Errorf("error from model")
		}
	}

	testDeployment(t, integration.REcho, test)
}

func BenchmarkHelloPyPredict(b *testing.B) {
	test := func(k *kernel) {
		b.ResetTimer()
		defer b.StopTimer()
		input := map[string]interface{}{"name": "bigdatabob"}
		for i := 0; i < b.N; i++ {
			_, err := k.Predict(input, nil)
			if err != nil {
				b.Errorf("prediction failed: %v", err)
				return
			}

		}

	}
	testDeployment(b, integration.PyHello, test)
}

func BenchmarkHelloPyHTTP(b *testing.B) {
	test := func(k *kernel) {
		b.ResetTimer()
		defer b.StopTimer()
		for i := 0; i < b.N; i++ {
			body := bytes.NewBuffer([]byte(`{"name": "bigdatabob"}`))
			req, err := http.NewRequest("POST", "/", body)
			if err != nil {
				b.Errorf("could not create request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			k.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Errorf("bad response from kernel: %s", rec.Body.String())
				return
			}
		}
	}
	testDeployment(b, integration.PyHello, test)
}

func BenchmarkHelloPyServer(b *testing.B) {
	test := func(k *kernel) {
		server := httptest.NewServer(k)
		defer server.Close()
		cli := &http.Client{Transport: &http.Transport{}}
		b.ResetTimer()
		defer b.StopTimer()
		for i := 0; i < b.N; i++ {
			body := bytes.NewBuffer([]byte(`{"name": "bigdatabob"}`))
			req, err := http.NewRequest("POST", server.URL, body)
			if err != nil {
				b.Errorf("could not create request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := cli.Do(req)
			if err != nil {
				b.Errorf("could not make a request: %v", err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				b.Errorf("bad response from kernel: %s", resp.Status)
				return
			}
			resp.Body.Close()
		}
	}
	testDeployment(b, integration.PyHello, test)
}

func BenchmarkHelloRServer(b *testing.B) {
	test := func(k *kernel) {
		server := httptest.NewServer(k)
		defer server.Close()
		cli := &http.Client{Transport: &http.Transport{}}
		b.ResetTimer()
		defer b.StopTimer()
		for i := 0; i < b.N; i++ {
			body := bytes.NewBuffer([]byte(`{"name": "bigdatabob"}`))
			req, err := http.NewRequest("POST", server.URL, body)
			if err != nil {
				b.Errorf("could not create request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := cli.Do(req)
			if err != nil {
				b.Errorf("could not make a request: %v", err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				b.Errorf("bad response from kernel: %s", resp.Status)
				return
			}
			resp.Body.Close()
		}
	}
	testDeployment(b, integration.RHello, test)
}

func BenchmarkHelloRHTTP(b *testing.B) {
	test := func(k *kernel) {
		b.ResetTimer()
		defer b.StopTimer()
		for i := 0; i < b.N; i++ {
			body := bytes.NewBuffer([]byte(`{"name": "bigdatabob"}`))
			req, err := http.NewRequest("POST", "/", body)
			if err != nil {
				b.Errorf("could not create request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			k.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				b.Errorf("bad response from kernel: %s", rec.Body.String())
				return
			}
		}
	}
	testDeployment(b, integration.RHello, test)
}

func testDeployment(t testing.TB, img string, test func(k *kernel)) {

	requiresImage(t, img)

	imgName := randSeq(10)
	containerName := randSeq(10)

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Errorf("could not create tempdir: %v", err)
		return
	}
	defer os.RemoveAll(tempDir)
	bundleFile := filepath.Join(tempDir, "bundle.json")

	var meta *ModelInfo
	var bundle []byte
	var deployErr error
	hf := func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/verify" {
			handleVerify(w, r)
			return
		}
		meta, bundle, deployErr = ReadBundle(r)
		w.WriteHeader(http.StatusOK)
	}

	var s *httptest.Server

	if runtime.GOOS == "linux" {
		s = httptest.NewServer(http.HandlerFunc(hf))
	} else {
		// for macs using Boot2Docker, we need to extract the computer's local IP
		// create a listener accordingly and then create a test server
		// because otherwise Boot2Docker can't access the local IP
		s, err = NewMacTestServer(http.HandlerFunc(hf))
		if err != nil {
			t.Errorf("could not create test server for Mac: %v", err)
			return
		}
	}

	err = integration.Run(img, "bob", "apikey", s.URL+"/")
	s.Close()
	if err != nil {
		t.Errorf("could not make request: %v", err)
		return
	}
	if deployErr != nil {
		t.Errorf("could not do deployment: %v", deployErr)
		return
	}
	if err = ioutil.WriteFile(bundleFile, bundle, 0644); err != nil {
		t.Errorf("could not create bundle file: %v", err)
		return
	}
	d := DeployInfo{
		Username:         "bob",
		Modelname:        meta.Modelname,
		Version:          1,
		Lang:             meta.Lang,
		LanguagePackages: meta.LanguagePackages,
		UbuntuPackages:   meta.UbuntuPackages,
	}

	run := func(cmd string, args ...string) {
		out, err := exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			t.Errorf("command %s %s failed %s", cmd, args, out)
		}
	}

	var stderr bytes.Buffer

	dockerFile, err := createDockerfile(&d)
	if err != nil {
		t.Errorf("failed to create dockerfile: %v", err)
		return
	}

	if err = buildImage(dockerFile, bundleFile, imgName, &stderr); err != nil {
		t.Errorf("could not build image %v: %s", err, stderr.String())
		return
	}
	defer run("docker", "rmi", "-f", imgName)

	cid, conn, err := startContainer(imgName, containerName)
	if err != nil {
		t.Errorf("could not create container: %v", err)
		return
	}
	defer run("docker", "rm", "-f", cid)

	stderr.Reset()
	k, err := newKernel(conn, &stderr)
	if err != nil {
		t.Errorf("could not begin kernel %v: %s", err, stderr.String())
		return
	}
	if err = k.heartbeat(); err != nil {
		t.Errorf("kernel failed heartbeat: %v", err)
		return
	}
	test(k)
}
