package mps

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// LogLine is a line from a model log with associated meta information.
type LogLine struct {
	InstanceId   int64
	User         string
	Model        string
	Timestamp    time.Time
	Data         string
	DeploymentId int64
}

// logger holds a buffered "queue" of log lines.
type logger struct {
	logs chan *LogLine
}

// newLogger returns a logger which will buffer a given number of lines.
// It is used to create writers for model instances.
func newLogger(lineCap int) *logger {
	return &logger{
		logs: make(chan *LogLine, lineCap),
	}
}

// logWriter is a io.Write for a specific instance.
// Writes to the logWriter queue in the logger which spawned it.
type logWriter struct {
	instanceId int64
	user       string
	model      string
	logs       chan *LogLine
}

// NewWriter returns a io.Writer for a specific instance.
func (l *logger) NewWriter(user, model string, instanceId int64) *logWriter {
	return &logWriter{
		instanceId: instanceId,
		user:       user,
		model:      model,
		logs:       l.logs,
	}
}

// logWriter implements io.Writer
func (w *logWriter) Write(p []byte) (n int, err error) {
	n = len(p)

	// Caller owns the byte slice. Must make copy before using.
	cp := make([]byte, n)
	copy(cp, p)
	cp = bytes.Trim(cp, "\n")

	lines := bytes.Split(cp, []byte{'\n'})
	for _, line := range lines {

		ll := LogLine{
			InstanceId: w.instanceId,
			User:       w.user,
			Model:      w.model,
			Timestamp:  time.Now(),
			Data:       string(line),
		}

		select {
		case w.logs <- &ll:
		default:
			// If the buffered channel doesn't take, the logger is at capacity.
			log.Println("log writer over capacity")
		}
	}
	return n, nil
}

// The logger is drained through HTTP requests.
// It writes each LogLine as a JSON stream.
func (l *logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e := json.NewEncoder(w)
	var line *LogLine
	for {
		select {
		case line = <-l.logs:
			e.Encode(line)
		default:
			return
		}
	}
}

// ParseLogs parses a stream of JSON encoded LogLines.
func ParseLogs(r io.Reader) ([]*LogLine, error) {
	d := json.NewDecoder(r)
	logs := []*LogLine{}

	for {
		line := LogLine{}
		if err := d.Decode(&line); err != nil {
			if err == io.EOF {
				return logs, nil
			}
			return nil, err
		}
		logs = append(logs, &line)
	}
}
