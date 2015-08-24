package mps

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/yhat/go-docker"
)

const (
	Python2 string = "python2"
	R       string = "r"
)

var (
	YhatCRANMirror   = "http://cran.yhathq.com/"
	YhatCondaChannel = "http://yhat-conda-channel.s3-website-us-east-1.amazonaws.com"
)

type Package struct {
	Name    string
	Version string // empty string implies unversioned
}

// Deployment holds the necessary meta information for performing a deployment.
type DeployInfo struct {
	Username  string
	Modelname string
	Version   int
	Lang      string // "python2" or "r"

	// Additional Conda channels to use besides Yhat's default one.
	CondaChannels []string
	// URL of CRAN mirror. If not set defaults to the Yhat CRAN mirror.
	CRANMirror string
	// Additional sources for apt-get'able packages.
	// If length is greater than 1, an `apt-get update` is performed.
	AptGetSources []string

	// Languages specific packages to install. For example "scikit-learn" or "ggplot2"
	LanguagePackages []Package
	// apt-get'able packages to install.
	UbuntuPackages []Package

	// Base image used to create Docker image
	// Will use reasonable defaults
	BaseImage string 
}

// BuildImage builds the docker image used to run this model.
func buildImage(dockerFile []byte, bundle, imgName string, logs io.Writer) error {
	if logs == nil {
		logs = ioutil.Discard
	}
	tempDir, err := ioutil.TempDir("", "scienceops_")
	if err != nil {
		return fmt.Errorf("could not create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err != nil {
		return fmt.Errorf("could not create model dockerfile: %v", err)
	}
	dfPath := filepath.Join(tempDir, "Dockerfile")
	if err = ioutil.WriteFile(dfPath, dockerFile, 0644); err != nil {
		return fmt.Errorf("could not write model dockerfile: %v", err)
	}

	// Copy the bundle and necessary source files to the temp directory
	dest := filepath.Join(tempDir, "bundle.json")
	if err := copyFile(dest, bundle); err != nil {
		return fmt.Errorf("could not copy bundle: %v", err)
	}

	pyPath := filepath.Join(tempDir, "pykernel.py")
	rPath := filepath.Join(tempDir, "rkernel.R")

	if err = ioutil.WriteFile(pyPath, pyKernel, 0644); err != nil {
		return fmt.Errorf("could not write kernel: %v", err)
	}
	if err = ioutil.WriteFile(rPath, rKernel, 0644); err != nil {
		return fmt.Errorf("could not write kernel: %v", err)
	}

	// build the dockerimage using the "docker" command line tool
	cmd := exec.Command("docker", "build", "--force-rm=true", "-t", imgName, ".")
	cmd.Dir = tempDir
	cmd.Stdout = logs
	cmd.Stderr = logs

	logs.Write([]byte(strings.Join(cmd.Args, " ")))

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("could not build docker image: %v", err)
	}
	return nil
}

func copyFile(dest, src string) error {
	srcFi, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFi.Close()
	stat, err := srcFi.Stat()
	if err != nil {
		return err
	}
	f := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	destFi, err := os.OpenFile(dest, f, stat.Mode())
	if err != nil {
		return err
	}
	defer destFi.Close()
	_, err = io.Copy(destFi, srcFi)
	return err
}

// StartContainer starts a given docker container and attaches to stdin,
// stdout and stderr of that container.
// Writing to conn will write to the stdin of the process.
// stdout and stderr must be split as a docker stream.
// see: https://docs.docker.com/reference/api/docker_remote_api_v1.17/#attach-to-a-container
// It's the callers responsiblity to destory the container and close the net
// connection when finished.
func startContainer(img, name string) (cid string, tcpConn net.Conn, err error) {
	cli, err := docker.NewDefaultClient(3 * time.Second)
	if err != nil {
		return "", nil, err
	}

	// Create a container. OpenStdin is neccessary to attach to stdin.
	config := &docker.ContainerConfig{Image: img, OpenStdin: true}
	cid, err = cli.CreateContainer(config, name)
	if err != nil {
		return "", nil, err
	}
	defer func(cid string) {
		if err != nil {
			cli.RemoveContainer(cid, true, false)
		}
	}(cid)
	// attach to the stdin, stdout and stderr of the container
	tcpConn, err = attachTCP(cid)
	if err != nil {
		return "", nil, fmt.Errorf("failed to attach to the model's container: %v", err)
	}

	if err = cli.StartContainer(cid, &docker.HostConfig{}); err != nil {
		return "", nil, fmt.Errorf("could not start docker container: %v", err)
	}

	return cid, tcpConn, nil
}

// attachTCP attaches to the stdin, stdout, and stderr of a docker container.
// Containers must be started with OpenStdin set to true.
// attachTCP returns the negotiated TCP connection of the attach remote api.
// It is the callers responsiblity to close the connection.
func attachTCP(cid string) (net.Conn, error) {
	// add a bunch of options to the attach request
	vals := url.Values{}
	for _, k := range []string{"stdin", "stdout", "stderr", "stream"} {
		vals.Add(k, "1")
	}
	var u url.URL
	u.Path = path.Join("/containers", cid, "attach")
	u.RawQuery = vals.Encode()
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// these headers will request an "upgrade" to "tcp"
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "tcp")

	// Dial docker with an OS specific configuration
	// On linux this will dial the docker socket, on mac this will dial boot2docker
	conn, err := dialDocker()
	if err != nil {
		return nil, fmt.Errorf("could not dial docker: %v", err)
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	// write the response to the open tcp connection
	if err := req.Write(conn); err != nil {
		return nil, fmt.Errorf("could not write request to docker: %v", err)
	}
	r := bufio.NewReader(conn)
	resp, err := http.ReadResponse(r, req)
	if err != nil {
		return nil, fmt.Errorf("could not read response from docker: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("expected StatusSwitchingProtocols, got %s", resp.Status)
	}
	// we've successfully negotiated tcp from docker
	return conn, nil
}

// Generate a dockerfile for the given deployment
func createDockerfile(d *DeployInfo) ([]byte, error) {
	// use de-reference to create a copy
	info := *d

	var buf bytes.Buffer
	var tmpl *template.Template
	switch d.Lang {
	case Python2:
		tmpl = pyTmpl

		if info.BaseImage == "" {
			info.BaseImage = "yhat/scienceops-python:0.0.2"
		}

	case R:
		if d.CRANMirror == "" {
			// Create a copy of the DeployInfo to appease the race detector.
			info.CRANMirror = YhatCRANMirror
		}
		if info.BaseImage == "" {
			info.BaseImage = "yhat/scienceops-r:0.0.2"
		}

		tmpl = rTmpl
	default:
		return nil, fmt.Errorf("unrecognized language passed to dockerfile generation '%s'", d.Lang)
	}
	if err := tmpl.Execute(&buf, &info); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var (
	requiredDockerImages = []string{
		"yhat/scienceops-python:0.0.2",
		"yhat/scienceops-r:0.0.2",
	}
	funcs  = map[string]interface{}{"quote": strconv.Quote}
	pyTmpl = template.Must(template.New("pyDockerfile").Funcs(funcs).Parse(`
FROM {{ .BaseImage }}

RUN conda config --add channels http://yhat-conda-channel.s3-website-us-east-1.amazonaws.com

{{ range $i, $chan := .CondaChannels }}
RUN conda config --add channels {{ quote $chan }}
{{ end }}

{{ range $i, $source := .AptGetSources }}
RUN echo {{ quote $source }} >> /etc/apt/sources.list 
{{ end }}

{{ if .AptGetSources }}
RUN apt-get update
{{ end }}

{{ range $i, $pkg := .UbuntuPackages }}
RUN apt-get install -y {{ $pkg.Name }}
{{ end }}

{{ range $i, $pkg := .LanguagePackages }}
RUN conda install -y -q {{ $pkg.Name }}{{ if $pkg.Version }}=={{ $pkg.Version }}{{ end }}
{{ end }}

COPY pykernel.py /src/pykernel.py
COPY bundle.json /src/bundle.json

ENV MODELNAME {{ .Modelname }}
ENV MODEL_VERSION {{ .Version }}

CMD ["python", "/src/pykernel.py", "/src/bundle.json"]
`))
	rTmpl = template.Must(template.New("rDockerfile").Funcs(funcs).Parse(`
FROM {{ .BaseImage }}

{{ range $i, $source := .AptGetSources }}
RUN echo {{ quote $source }} >> /etc/apt/sources.list 
{{ end }}

{{ if .AptGetSources }}
RUN apt-get update
{{ end }}

{{ range $i, $pkg := .UbuntuPackages }}
RUN apt-get install -y {{ $pkg.Name }}
{{ end }}

{{ range $i, $pkg := .LanguagePackages }}
RUN Rscript -e 'install.packages({{ quote $pkg.Name }}, repos={{ quote $.CRANMirror }})'
{{ end }}

COPY rkernel.R /src/rkernel.R
COPY bundle.json /src/bundle.json

ENV MODELNAME {{ .Modelname }}
ENV MODEL_VERSION {{ .Version }}

CMD ["Rscript", "/src/rkernel.R", "/src/bundle.json"]
`))
)
