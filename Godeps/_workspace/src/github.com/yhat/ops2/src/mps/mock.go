package mps

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// mockContainer mocks a docker container.
type mockContainer struct {
	// reads read from this pipe
	readPR *io.PipeReader
	readPW *io.PipeWriter

	// writes write to this pipe
	writePR *io.PipeReader
	writePW *io.PipeWriter
	in      *bufio.Reader //always wraps the writePR
}

func newMockContainer() io.ReadWriteCloser {
	mc := mockContainer{}
	mc.readPR, mc.readPW = io.Pipe()
	mc.writePR, mc.writePW = io.Pipe()
	mc.in = bufio.NewReader(mc.writePR)
	go mc.run() // being listening for prediction requests
	return &mc
}

func (mc *mockContainer) Read(p []byte) (n int, err error) {
	return mc.readPR.Read(p)
}

func (mc *mockContainer) Write(p []byte) (n int, err error) {
	return mc.writePW.Write(p)
}

func (mc *mockContainer) close(err error) {
	mc.readPW.CloseWithError(err)
	mc.writePR.CloseWithError(err)
}

func (mc *mockContainer) Close() error {
	mc.close(fmt.Errorf("container was closed"))
	return nil
}

// writeDockerStream emulates the encoding of a docker stream.
// It writes the s as a byte slice while prepending the encoding header.
func writeDockerStream(w io.Writer, s string) (err error) {
	data := []byte(s)
	header := make([]byte, 8)
	header[0] = 0x1 // stdout
	binary.BigEndian.PutUint32(header[4:], uint32(len(data)))

	data = append(header, data...)
	_, err = w.Write(data)
	return
}

func (mc *mockContainer) run() {
	if err := writeDockerStream(mc.readPW, `{"status":"up"}`); err != nil {
		mc.close(err)
	}

	for {
		line, err := mc.in.ReadBytes('\n')
		if err != nil {
			mc.close(err)
			return
		}
		if len(line) == 0 {
			continue
		}
		var req map[string]interface{}
		if err = json.Unmarshal(line, &req); err != nil {
			mc.close(fmt.Errorf("could not decode request: %v", err.Error()))
		}

		get := func(key string) (string, bool) {
			if val, ok := req[key]; ok {
				if s, ok := val.(string); ok {
					return s, true
				}
			}
			return "", false
		}

		var resp = ""
		if heartbeat, ok := get("heartbeat"); ok {
			resp = fmt.Sprintf(`{"heartbeat_response":%s}`, strconv.Quote(heartbeat))
		} else {
			yhatid, _ := get("yhat_id")
			resp = fmt.Sprintf(`{"result":{"pred":42},"yhat_id":%s}`, strconv.Quote(yhatid))
		}

		if err = writeDockerStream(mc.readPW, resp); err != nil {
			mc.close(err)
		}
	}
}

// NewMPSMock creates a mock MPS server.
// It can be used just like a normal MPS, but will create in memory
// "containers" rather than coordinating with Docker.
func NewMPSMock() (*MPS, error) {
	mps := MPS{
		deployments:      make(map[int64]*deployment),
		mu:               new(sync.Mutex),
		HeartbeatTimeout: DefaultHeartbeatTimeout,
		isMock:           true,
	}
	return &mps, nil
}
