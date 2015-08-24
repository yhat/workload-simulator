package mps

import (
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// ModelMeta holds meta data for a version of a model.
type ModelInfo struct {
	Modelname        string
	Lang             string
	LanguagePackages []Package
	UbuntuPackages   []Package
	SourceCode       string
}

// readBundle determines if a request is a Python or R model then attempts
// to decode the bundle and meta information associated with it.
// The URL is expected to be unmodified from the initial request.
func ReadBundle(r *http.Request) (meta *ModelInfo, bundle []byte, err error) {
	switch r.URL.Path {
	case "/deployer/model":
		return readRBundle(r)
	case "/deployer/model/large":
		return readPyBundle(r)
	}
	return nil, nil, fmt.Errorf("unrecognized route: %s", r.URL.Path)
}

// readRBundle parses the bundle and meta data from a R deployment request.
func readRBundle(r *http.Request) (meta *ModelInfo, bundle []byte, err error) {

	m := ModelInfo{Lang: R}

	// Decode (then encode) model image
	file, _, err := r.FormFile("model_image")
	if err != nil {
		return nil, nil, err
	}
	// TODO (eric): This can be done using a base64 Encoder for streaming if
	// we don't want to hold the whole bundle in memory.
	image, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}
	base64Image := base64.StdEncoding.EncodeToString(image)
	m.Modelname = r.FormValue("modelname")
	if m.Modelname == "" {
		return nil, nil, fmt.Errorf("no model name provided")
	}

	b := struct {
		Image string `json:"image"`
	}{Image: base64Image}

	bundle, err = json.Marshal(b)
	if err != nil {
		return nil, nil, fmt.Errorf("could not construct bundle: %v", err)
	}

	// grab meta information from the form
	m.SourceCode = r.FormValue("code")
	if m.SourceCode == "" {
		return nil, nil, fmt.Errorf("no model code provided")
	}
	p := r.FormValue("packages")
	if p != "" {
		if err = json.Unmarshal([]byte(p), &m.LanguagePackages); err != nil {
			return nil, nil, fmt.Errorf("could not parse r packages: %v", err)
		}
	}
	aptPkgs, ok := r.Form["apt_packages"]
	if ok {
		m.UbuntuPackages = make([]Package, len(aptPkgs))
		for i, pkg := range r.Form["apt_packages"] {
			m.UbuntuPackages[i] = Package{Name: pkg}
		}
	}

	return &m, bundle, nil
}

// readPyBundle reads the bundle from a Python deployment.
func readPyBundle(r *http.Request) (meta *ModelInfo, bundle []byte, err error) {
	mpr, err := r.MultipartReader()
	if err != nil {
		return nil, nil, fmt.Errorf("malformed multipart request: %v", err)
	}

	// only read one request
	part, err := mpr.NextPart()
	if err != nil {
		if err == io.EOF {
			return nil, nil, fmt.Errorf("no files sent")
		}
		return nil, nil, fmt.Errorf("read mulitpart part %v", err)
	}
	defer part.Close()

	rc, err := zlib.NewReader(part)
	if err != nil {
		return nil, nil, fmt.Errorf("zlib encoding error: %v", err)
	}
	defer rc.Close()

	bundle, err = ioutil.ReadAll(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("could not decode bundle: %v", err)
	}
	// decode meta data from the bundle
	var b struct {
		Modelname      string   `json:"modelname"`
		UbuntuPackages []string `json:"packages"`
		PyPkgs         string   `json:"reqs"`
		SourceCode     string   `json:"code"`
		Modules        interface{} `json:"modules"`
	}
	if err = json.Unmarshal(bundle, &b); err != nil {
		return nil, nil, fmt.Errorf("could not decode data from bundle: %v", err)
	}
	m := ModelInfo{
		Lang:             Python2,
		Modelname:        b.Modelname,
		UbuntuPackages:   listToPkgs(b.UbuntuPackages),
		LanguagePackages: parsePyPackages(b.PyPkgs),
		SourceCode:       b.SourceCode,
	}

	return &m, bundle, nil
}

func listToPkgs(pkgNames []string) []Package {
	pkgs := make([]Package, len(pkgNames))
	for i, p := range pkgNames {
		pkgs[i] = Package{Name: p}
	}
	return pkgs
}

func parsePyPackages(pkgStr string) []Package {
	lines := strings.Split(pkgStr, "\n")
	pkgs := make([]Package, len(lines))
	n := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		split := strings.SplitN(line, "==", 2)
		switch len(split) {
		case 1:
			pkgs[n] = Package{Name: split[0]}
			n++
		case 2:
			pkgs[n] = Package{Name: split[0], Version: split[1]}
			n++
		}
	}
	return pkgs[:n]
}
