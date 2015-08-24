package tlsconfig

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	if _, err := NewClient(tempDir); err != nil {
		t.Errorf("error creating new client: %v", err)
		return
	}
}

func TestListen(t *testing.T) {

	cliDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(cliDir)
	serverDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(serverDir)

	client, err := NewClient(cliDir)
	if err != nil {
		t.Errorf("error creating new client: %v", err)
		return
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hi!")
	}
	addr := "127.0.0.1:10123"
	listener, err := NewListener(serverDir, http.HandlerFunc(h))
	if err != nil {
		t.Error(err)
		return
	}
	listener.ErrorLog = log.New(ioutil.Discard, "", 0)

	go func() {
		if err := listener.Listen(addr); err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(4 * time.Second)

	if err := client.Handshake(addr); err != nil {
		t.Errorf("handshake failed: %s", err)
		return
	}

	if _, err := http.Get("https://" + addr + "/"); err == nil {
		t.Errorf("able to make get requests after configuration")
	}

	cli := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: client.TLSConfig(),
		},
	}
	resp, err := cli.Get("https://" + addr + "/")
	if err != nil {
		t.Errorf("unable to make get request: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 got %s", resp.Status)
	}
}
