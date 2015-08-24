package mps

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/yhat/wsutil"
)

// MPSClient is used to request actions from the MPS.
// To prevent unnecessary overhead it should be constructed once
// then used repeatedly.
type MPSClient struct {
	baseURL string // e.g. https://127.0.0.1:32432

	// These proxies are used to proxy prediction requests.
	// It's very important that we set them up with the correct TLS config.
	// The client and httpproxy share a http.Transport object so they
	// benefit from using the same connection pool.
	client    *http.Client
	httpProxy *httputil.ReverseProxy
	wsProxy   *wsutil.ReverseProxy

	transport *http.Transport
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// NewMPSClient constructs a MPSClient from the provided baseURL and
// TLS config.
// If TLS config is nil, the default configuration is used.
func NewMPSClient(baseURL string, tlsConfig *tls.Config) (*MPSClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse url: %v", err)
	}
	t := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: t}
	if (u.Scheme != "http") && (u.Scheme != "https") {
		return nil, fmt.Errorf("base url must have scheme 'http' or 'https'")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("no host in url")
	}

	var wsURL url.URL
	wsURL = *u
	wsURL.Scheme = "ws"
	if u.Scheme == "https" {
		wsURL.Scheme = "wss"
	}
	httpProxy := httputil.NewSingleHostReverseProxy(u)
	wsProxy := wsutil.NewSingleHostReverseProxy(&wsURL)

	httpProxy.Transport = t
	wsProxy.TLSClientConfig = t.TLSClientConfig
	return &MPSClient{baseURL, client, httpProxy, wsProxy, t}, nil
}

type DeploymentStatus struct {
	Id    int64
	Ready bool
}

type MPSStatus struct {
	Deployments []DeploymentStatus
}

func (client *MPSClient) Status() (*MPSStatus, error) {
	u := singleJoiningSlash(client.baseURL, routeStatus)
	resp, err := client.client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response from MPS %s: %s", resp.Status, body)
	}
	var status MPSStatus
	if err = json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("could not decode response: %v", err)
	}
	return &status, nil
}

// Heartbeat queries the Heartbeat of all of it's active kernels.
// It returns the deployment id of all heathy kernels.
func (client *MPSClient) Heartbeat(id int64) (err error) {
	idstr := strconv.FormatInt(id, 10)
	u := singleJoiningSlash(client.baseURL, routeHeartbeat+"?id="+idstr)
	resp, err := client.client.Get(u)
	if err != nil {
		return fmt.Errorf("failed to make POST request: %v", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("bad response from MPS %s: %v", resp.Status, err)
		} else {
			return fmt.Errorf("bad response from MPS %s: %s", resp.Status, body)
		}
	}
}

// Destroy stops and removes the kernel, container, and image associated with
// the provided deployment id.
func (client *MPSClient) Destroy(id int64) error {
	idstr := strconv.FormatInt(id, 10)
	u := singleJoiningSlash(client.baseURL, routeDestroy+"?id="+idstr)
	resp, err := client.client.Post(u, "", nil)
	if err != nil {
		return fmt.Errorf("failed to make POST request: %v", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("bad response from MPS %s: %v", resp.Status, err)
		} else {
			return fmt.Errorf("bad response from MPS %s: %s", resp.Status, body)
		}
	}
}

func (client *MPSClient) Ping() error {
	u := singleJoiningSlash(client.baseURL, routePing)
	resp, err := client.client.Get(u)
	if err != nil {
		return fmt.Errorf("failed to make GET request: %v", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("bad response from MPS %s: %v", resp.Status, err)
		} else {
			return fmt.Errorf("bad response from MPS %s: %s", resp.Status, body)
		}
	}
}

func (client *MPSClient) Logs() ([]*LogLine, error) {
	u := singleJoiningSlash(client.baseURL, routeLogs)
	resp, err := client.client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %v", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return ParseLogs(resp.Body)
	default:
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("bad response from MPS %s: %v", resp.Status, err)
		} else {
			return nil, fmt.Errorf("bad response from MPS %s: %s", resp.Status, body)
		}
	}
}

// Deploy requests and builds a deployment from the MPS.
// This builds an image, starts a container, and waits for the container
// to become ready to make predictions.
// id must be a new, unique string which the MPS hasn't associated with a
// deployment.
//
//   client := NewMPSClient(mpsURL, nil)
//   d := Deployment{
//       Username:  "bigdatabob",
//       Modelname: "hadoooooopdemo",
//       Lang:      R,
//   }
//   bundlePath := "/var/scienceops/bundle.json"
//   if err := client.Deploy("1234", &d, bundlePath); err != nil {
//       return fmt.Errorf("model failed to build: %v", err)
//   }
//
func (client *MPSClient) Deploy(id int64, d *DeployInfo, bundlePath string) error {
	idstr := strconv.FormatInt(id, 10)
	p := singleJoiningSlash(client.baseURL, routeDeploy+"?id="+idstr)

	bundle, err := os.Open(bundlePath)
	if err != nil {
		return fmt.Errorf("could not open bundle file: %v", err)
	}

	// Rather than encoding straight to the POST request body, we'll encode
	// to a file then have the POST request read from that file
	tempFile, err := ioutil.TempFile("", "scienceops_deployment_")
	if err != nil {
		bundle.Close()
		return fmt.Errorf("could not create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	//pass deployinfo to create Docker File
	dockerFile, err := createDockerfile(d)

	if err != nil {
		return fmt.Errorf("failed to create dockerfile: %v", err)
	}

	err = encodeDeployment(tempFile, dockerFile, d, bundle)
	bundle.Close()
	if err != nil {
		return fmt.Errorf("could not encode deployment: %v", err)
	}
	if _, err = tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning of deployment file: %v", err)
	}

	req, err := http.NewRequest("POST", p, tempFile)
	if err != nil {
		return fmt.Errorf("could not create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	var resp *http.Response

	resp, err = client.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("bad response from MPS %s: %v", resp.Status, err)
		} else {
			return fmt.Errorf("bad response from MPS %s: %s", resp.Status, body)
		}
	}
	return nil
}

// Predict takes a request and response writer and proxies the request to
// the deployment associated with the provided id.
func (client *MPSClient) Predict(w http.ResponseWriter, r *http.Request, id int64) {
	q := r.URL.Query()
	q.Set("id", strconv.FormatInt(id, 10))
	r.URL.RawQuery = q.Encode()
	r.URL.Path = routePredict

	if wsutil.IsWebSocketRequest(r) {
		client.wsProxy.ServeHTTP(w, r)
	} else {
		client.httpProxy.ServeHTTP(w, r)
	}
}

// PredictHandler returns an http.Handler which can be used to proxy
// prediction request to the deployment associated with provided id.
func (client *MPSClient) PredictHandler(id int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client.Predict(w, r, id)
	})
}
