package mps

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/yhat/go-docker"
	"github.com/yhat/wsutil"
)

// Kernel coordinates request through the stdin and stdout of a model.
// It does not monitor or control the underlying container in any way.
type kernel struct {
	mu *sync.Mutex

	conn io.Closer

	// Send json data through in and receive through out
	in  *json.Encoder
	out *json.Decoder
}

// NewKernel wraps a TCP connection in model logic, writing to stdin
// during predictions and parsing stdout.
// stderr from the model will be redirected to the provided writer.
func newKernel(tcpConn io.ReadWriteCloser, stderr io.Writer) (*kernel, error) {
	stdoutReader, stdoutWriter := io.Pipe()

	go func() {
		err := docker.SplitStream(tcpConn, stdoutWriter, stderr)
		stdoutWriter.CloseWithError(err)
	}()

	m := kernel{
		mu:   new(sync.Mutex),
		conn: tcpConn,
		in:   json.NewEncoder(tcpConn),
		out:  json.NewDecoder(stdoutReader),
	}

	if err := m.online(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *kernel) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conn.Close()
}

// online waits for a model to produce an "online" status code
func (m *kernel) online() error {
	s := struct {
		Status string `json:"status"`
	}{}
	if err := m.out.Decode(&s); err != nil {
		return fmt.Errorf("could not decode status response from model: %v", err)
	}
	if strings.ToLower(s.Status) != "up" {
		return fmt.Errorf("model failed to build")
	}
	return nil
}

func init() {
	mathrand.Seed(time.Now().Unix())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[mathrand.Intn(len(letters))]
	}
	return string(b)
}

type heartbeatReq struct {
	Heartbeat string `json:"heartbeat"`
	YhatId    string `json:"yhat_id"` // kernels reject things without ids
}

type heartbeatResp struct {
	Response string `json:"heartbeat_response"`
}

func (k *kernel) heartbeat() error {
	req := heartbeatReq{randSeq(32), randSeq(32)}
	var resp heartbeatResp
	k.mu.Lock()
	if err := k.in.Encode(&req); err != nil {
		k.mu.Unlock()
		return fmt.Errorf("could not make heartbeat request: %v", err)
	}
	err := k.out.Decode(&resp)
	k.mu.Unlock()
	if err != nil {
		return fmt.Errorf("could not decode heartbeat from kernel: %v", err)
	}
	if resp.Response != req.Heartbeat {
		return fmt.Errorf("response heartbeat did not match requested '%s' vs '%s'",
			resp.Response, req.Heartbeat)
	}
	return nil
}

func (m *kernel) Predict(data interface{}, meta url.Values) (map[string]interface{}, error) {
	req := map[string]interface{}{"body": data}
	if meta != nil {
		for k, vv := range meta {
			for _, v := range vv {
				req[k] = v
			}
		}
	}

	yhatId := ""
	if m, ok := data.(map[string]interface{}); ok {
		id, ok := m["yhat_id"].(string)
		if ok {
			yhatId = id
		}
	}

	if yhatId == "" {
		var err error
		req["yhat_id"], err = uuid()
		if err != nil {
			return nil, err
		}
	} else {
		req["yhat_id"] = yhatId
	}

	m.mu.Lock()
	err := m.in.Encode(&req)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}

	resp := map[string]interface{}{}
	err = m.out.Decode(&resp)
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}

	// If the repsonse from a R model has one row we have to correct the JSON
	oneRowDF := "one_row_dataframe"
	if _, ok := resp[oneRowDF]; ok {
		delete(resp, oneRowDF)
		result, ok := resp["result"].(map[string]interface{})
		if ok {
			for k, v := range result {
				result[k] = []interface{}{v}
			}
			resp["result"] = result
		}
	}

	return resp, nil
}

func uuid() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	err := struct {
		Error string `json:"error"`
	}{msg}
	json.NewEncoder(w).Encode(&err)
}

func (k *kernel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if wsutil.IsWebSocketRequest(r) {
		websocket.Handler(k.handleWS).ServeHTTP(w, r)
		return
	}

	switch r.Method {
	case "GET":

		// TODO: add more meta data to response such as model version
		err := k.heartbeat()
		resp := struct {
			Status string `json:"status"`
			Time   string `json:"date"`
		}{
			Time: time.Now().UTC().Format(time.RFC3339),
		}
		if err != nil {
			resp.Status = "ERROR"
		} else {
			resp.Status = "OK"
		}
		json.NewEncoder(w).Encode(&resp)
	case "POST":
		query := r.URL.Query()
		var data interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			jsonError(w, "could not decode body: "+err.Error(), http.StatusBadRequest)
			return
		}
		resp, err := k.Predict(data, query)
		if err != nil {
			jsonError(w, "model experienced a fatal error "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, ok := resp["result"]; !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(&resp)
	default:
		http.Error(w, "I only respond to GET and POSTs", http.StatusNotImplemented)
	}
}

func (k *kernel) handleWS(ws *websocket.Conn) {

	r := ws.Request()
	query := r.URL.Query()

	defer ws.Close()
	for {
		data := map[string]interface{}{}
		if err := websocket.JSON.Receive(ws, &data); err != nil {
			return
		}
		resp, err := k.Predict(data, query)
		if err != nil {
			d := &map[string]string{
				"error": "model experienced a fatal error " + err.Error(),
			}
			websocket.JSON.Send(ws, d)
			return
		}
		if err := websocket.JSON.Send(ws, &resp); err != nil {
			return
		}
	}
}
