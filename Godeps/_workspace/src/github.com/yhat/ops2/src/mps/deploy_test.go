package mps

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/yhat/go-docker"
)

func requiresImage(t testing.TB, img string) {
	cli, err := docker.NewDefaultClient(time.Second * 3)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cli.InspectImage(img)
	if err == docker.ErrNotFound {
		if strings.HasPrefix(img, "yhat/integration") {
			t.Skipf("Integration tests require you to build image '%s' beforehand. Use integration/build-test-images.sh to generate the images.", img)
		} else {
			t.Skipf("test requires docker image %s, skipping for now", img)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}

func requiresCommand(t testing.TB, cmd string) {
	if _, err := exec.LookPath(cmd); err != nil {
		t.Skipf("required executable %s not found", cmd)
	}
}

func TestCreateDockerfile(t *testing.T) {
	testDeployments := []DeployInfo{
		{
			Username:         "eric",
			Modelname:        "beerrec",
			Lang:             Python2,
			CondaChannels:    []string{"foo", "bar"},
			LanguagePackages: []Package{{"scikit-learn", "1.2.3"}},
			UbuntuPackages:   []Package{{"nmap", ""}, {"tree", ""}},
			CRANMirror:       "afasdfads",
		},
		{
			Username:         "eric",
			Modelname:        "hellor",
			Lang:             R,
			LanguagePackages: []Package{{"e1071", ""}},
			UbuntuPackages:   []Package{{"nmap", ""}, {"tree", ""}},
			CRANMirror:       YhatCondaChannel,
		},
	}
	for _, d := range testDeployments {
		_, err := createDockerfile(&d)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestBuildRImage(t *testing.T) {

	requiresCommand(t, "docker")
	requiresImage(t, "yhat/scienceops-r:0.0.2")

	d := DeployInfo{
		Username:   "eric",
		Modelname:  "hellor",
		Lang:       R,
		CRANMirror: YhatCondaChannel,
	}
	b := "bundles/r-bundle.json"
	imgName := "yhat/test-r-image"

	var logs bytes.Buffer

	dockerFile, err := createDockerfile(&d)
	if err != nil {
		t.Errorf("failed to create dockerfile: %v", err)
		return
	}

	err = buildImage(dockerFile, b, imgName, &logs)
	if err != nil {
		t.Errorf("failed to build docker image: %v %s", err, logs.Bytes())
		return
	}
	defer func() {
		out, err := exec.Command("docker", "rmi", imgName).CombinedOutput()
		if err != nil {
			t.Error("could not remove docker image %s %v %s", imgName, err, out)
		}
	}()
}

func TestBuildAptGetImage(t *testing.T) {
	requiresCommand(t, "docker")
	requiresImage(t, "yhat/scienceops-python:0.0.2")

	d := DeployInfo{
		Username:       "eric",
		Modelname:      "hellopy",
		Lang:           Python2,
		UbuntuPackages: []Package{{"tree", ""}},
	}
	b := "bundles/py-bundle.json"
	imgName := "yhat/test-apt-get-image"

	var logs bytes.Buffer

	dockerFile, err := createDockerfile(&d)
	if err != nil {
		t.Errorf("failed to create dockerfile: %v", err)
		return
	}

	err = buildImage(dockerFile, b, imgName, &logs)

	if err != nil {
		t.Errorf("failed to build docker image: %v %s", err, logs.Bytes())
		return
	}
	defer func() {
		out, err := exec.Command("docker", "rmi", imgName).CombinedOutput()
		if err != nil {
			t.Error("could not remove docker image %s %v %s", imgName, err, out)
		}
	}()
}

func TestBuildPyImage(t *testing.T) {

	requiresCommand(t, "docker")
	requiresImage(t, "yhat/scienceops-python:0.0.2")

	d := DeployInfo{
		Username:  "eric",
		Modelname: "hellopy",
		Lang:      Python2,
	}
	b := "bundles/py-bundle.json"
	imgName := "yhat/test-py-image"

	var logs bytes.Buffer

	dockerFile, err := createDockerfile(&d)
	if err != nil {
		t.Errorf("failed to create dockerfile: %v", err)
		return
	}

	err = buildImage(dockerFile, b, imgName, &logs)

	if err != nil {
		t.Errorf("failed to build docker image: %v %s", err, logs.Bytes())
		return
	}
	defer func() {
		out, err := exec.Command("docker", "rmi", imgName).CombinedOutput()
		if err != nil {
			t.Error("could not remove docker image %s %v %s", imgName, err, out)
		}
	}()
}

func TestAttachTCP(t *testing.T) {
	testAttachNTimes(t, 1)
}

// can we reattach to a container's stdin and stdout?
func TestReAttachTCP(t *testing.T) {
	testAttachNTimes(t, 3)
}

func testAttachNTimes(t *testing.T, nTimes int) {

	img := "ubuntu:14.04"
	requiresImage(t, img)

	containerName := "yhat-testattachtcp"

	cli, err := docker.NewDefaultClient(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}

	cmd := []string{"/bin/bash", "-c", "cat /dev/stdin"}
	config := &docker.ContainerConfig{Image: img, OpenStdin: true, Cmd: cmd}
	cid, err := cli.CreateContainer(config, containerName)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := cli.RemoveContainer(cid, true, false)
		if err != nil {
			t.Errorf("could not remove container %s: %v", cid, err)
		}
	}()
	for i := 0; i < nTimes; i++ {
		conn, err := attachTCP(cid)
		if err != nil {
			t.Errorf("could not attach to container: %v", err)
			return
		}
		defer conn.Close()
		if i == 0 {
			if err = cli.StartContainer(cid, &docker.HostConfig{}); err != nil {
				t.Errorf("could not start docker container: %v", err)
				return
			}
		}

		n := 2048
		randBytes := make([]byte, n)
		if _, err = io.ReadFull(rand.Reader, randBytes); err != nil {
			t.Errorf("failed to initialize random buffer")
			return
		}

		if _, err := conn.Write(randBytes); err != nil {
			t.Errorf("failed to write to open connection: %v", err)
			return
		}
		b := make([]byte, n)
		r := NewStreamReader(conn)
		nn, err := io.ReadFull(r, b)
		if err != nil {
			t.Errorf("could not read from connnection: %v", err)
			return
		}
		if n != nn {
			t.Errorf("%d bytes written to docker container, read %d", n, nn)
		}
		if bytes.Compare(randBytes, b) != 0 {
			t.Errorf("bytes written were different from those read")
		}
		// stdout
		if streamType := r.Type(); streamType != 0x1 {
			t.Errorf("expected stream type to be 0x1 (stdout), got 0x%x", streamType)
		}
	}
}

type StreamReader struct {
	r          io.Reader
	streamType byte
	frame      []byte // the last frame read
	headerBuf  []byte // reused 8 bytes to hold the frame header
}

func NewStreamReader(r io.Reader) *StreamReader { return &StreamReader{r: r} }

func (sr *StreamReader) Read(b []byte) (n int, err error) {
	if sr.frame == nil || len(sr.frame) == 0 {
		h := sr.header()
		if _, err = io.ReadFull(sr.r, h); err != nil {
			return
		}
		sr.streamType = h[0]
		nn := binary.BigEndian.Uint32(h[4:])
		frame := make([]byte, nn)
		if _, err = io.ReadFull(sr.r, frame); err != nil {
			return
		}
		sr.frame = frame
	}
	n = len(sr.frame)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		b[i] = sr.frame[i]
	}
	sr.frame = sr.frame[n:]
	return
}

func (sr *StreamReader) header() []byte {
	if sr.headerBuf == nil {
		sr.headerBuf = make([]byte, 8)
	}
	return sr.headerBuf
}

func (sr *StreamReader) Type() byte { return sr.streamType }
