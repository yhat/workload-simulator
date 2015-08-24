package mps

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger(t *testing.T) {
	l := newLogger(10)

	w := l.NewWriter("bigdatabob", "helloworld", 10)

	data := []byte("hello\nworld\n")

	_, err := w.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	if n := len(l.logs); n != 2 {
		t.Fatalf("expected 3 lines of logs queued, got %d", n)
	}

	// writer should be able to own the byte slice
	newData := []byte("foo\nbar")
	data = data[:len(newData)]
	copy(data, newData)

	if _, err = w.Write(data); err != nil {
		t.Fatal(err)
	}

	if n := len(l.logs); n != 4 {
		t.Fatalf("expected 5 lines of logs queued, got %d", n)
	}

	s := httptest.NewServer(l)
	resp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	if n := len(l.logs); n != 0 {
		t.Fatalf("expected no lines in queue after drain, got %d", n)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bad status code returned: %s", resp.Status)
	}

	lines, err := ParseLogs(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("could not parse logs: %v", err)
	}
	if n := len(lines); n != 4 {
		t.Fatalf("expected 3 lines of logs, got %d", n)
	}
	for _, line := range lines {
		if line.InstanceId != 10 {
			t.Fatalf("expected instance id to be '10', got '%d'", line.InstanceId)
		}
	}
	for i, exp := range []string{"hello", "world", "foo", "bar"} {
		got := lines[i].Data
		if got != exp {
			t.Fatalf("for line %d expected '%s' got '%s'", i, exp, got)
		}
	}
}
