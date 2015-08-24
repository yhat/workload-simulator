package mps

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"runtime"

	"github.com/yhat/ops2/src/mps/integration"
)

func handleVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": "true"}`))
}

var metaExp = map[string]*ModelInfo{
	integration.RHello: {
		Modelname: "HelloWorldR",
		Lang:      R,
	},
	integration.RSVC: {
		Modelname:        "SupportVectorClassifierR",
		Lang:             R,
		LanguagePackages: []Package{{"e1071", "1.6-4"}},
	},
	integration.RAptGet: {
		Modelname:      "HelloWorldAptPkgR",
		Lang:           R,
		UbuntuPackages: []Package{{"nmap", ""}, {"tree", ""}},
	},
	integration.PyRelayRides: {
		Modelname: "RelayRidesPricing",
		Lang:      Python2,
		LanguagePackages: []Package{
			{"yhat", "1.3.6"}, {"numpy", ""}, {"pandas", "0.15.2"},
		},
	},
	integration.PyAptGet: {
		Modelname:        "PyAptGet",
		Lang:             Python2,
		LanguagePackages: []Package{{"yhat", "1.3.6"}},
		UbuntuPackages:   []Package{{"tree", ""}},
	},
}

type bundleReader func(r *http.Request) (*ModelInfo, []byte, error)

func testReadBundle(t *testing.T, reader bundleReader, imgs []string) {
	for _, img := range imgs {
		requiresImage(t, img)
	}
	var bundle []byte
	var meta *ModelInfo
	var err error

	hf := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/verify" {
			handleVerify(w, r)
			return
		}
		meta, bundle, err = reader(r)
		if err != nil {
			t.Errorf("failed to read bundle: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}

	var s *httptest.Server

	if runtime.GOOS == "linux" {
		s = httptest.NewServer(http.HandlerFunc(hf))
	} else {
		// for Macs using Boot2Docker, we need to extract the computer's local IP
		// create a listener accordingly and then create a test server that Boot2Docker can find
		s, err = NewMacTestServer(http.HandlerFunc(hf))
		if err != nil {
			t.Errorf("could not create test server for Mac: %v", err)
			return
		}
	}

	defer s.Close()

	for _, img := range imgs {

		bundle = nil
		meta = nil

		err := integration.Run(img, "bob", "apikey", s.URL+"/")
		if err != nil {
			t.Errorf("could not make request: %v", err)
			continue
		}
		if bundle == nil {
			t.Errorf("bundle not set")
			continue
		}
		expMeta, ok := metaExp[img]
		if !ok {
			t.Errorf("no expected meta found for: %v", img)
			continue
		}
		// R doesn't do verisoning very well. Don't bother comparing
		compareMeta(t, img, meta, expMeta, expMeta.Lang == Python2)
		if meta.SourceCode == "" {
			t.Errorf("no source code detected")
		}
	}
}

func TestReadRBundle(t *testing.T) {

	imgs := []string{
		integration.RHello,
		integration.RSVC,
		integration.RAptGet,
	}
	testReadBundle(t, readRBundle, imgs)
}

func TestReadPyBundle(t *testing.T) {

	imgs := []string{
		integration.PyRelayRides,
		integration.PyAptGet,
	}
	testReadBundle(t, readPyBundle, imgs)
}

func compareMeta(t *testing.T, img string, m1, m2 *ModelInfo, cmpVersion bool) {
	if m1.Modelname != m2.Modelname {
		t.Errorf("%s: modelname '%s' and '%s' did not match", img, m1.Modelname, m2.Modelname)
	}
	if err := comparePkgs(m1.LanguagePackages, m2.LanguagePackages, cmpVersion); err != nil {
		t.Errorf("%s: language %s", img, err)
	}
	if err := comparePkgs(m1.UbuntuPackages, m2.UbuntuPackages, cmpVersion); err != nil {
		t.Errorf("%s: ubuntu %s", img, err)
	}
}

func comparePkgs(pkgs1, pkgs2 []Package, cmpVersion bool) error {
	n := len(pkgs1)
	m := len(pkgs2)
	if n != m {
		return fmt.Errorf("package lengths did not match %d vs %d", n, m)
	}
	sort.Sort(pkgsByName(pkgs1))
	sort.Sort(pkgsByName(pkgs2))
	for i, pkg1 := range pkgs1 {
		pkg2 := pkgs2[i]
		if pkg1.Name != pkg2.Name {
			return fmt.Errorf("package lists did not match")
		}
		if cmpVersion && (pkg1.Version != pkg2.Version) {
			return fmt.Errorf("package versions for %s did not match, %s vs %s",
				pkg1.Name, pkg1.Version, pkg2.Version)
		}
	}
	return nil
}

type pkgsByName []Package

func (p pkgsByName) Len() int           { return len(p) }
func (p pkgsByName) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p pkgsByName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func TestParsePyPackages(t *testing.T) {
	tests := []struct {
		Str string
		Exp []Package
	}{
		{
			Str: "yhat==1.2.10\nnumpy\npandas==0.15.2",
			Exp: []Package{
				{"yhat", "1.2.10"}, {"numpy", ""}, {"pandas", "0.15.2"},
			},
		},
	}
	for _, test := range tests {
		result := parsePyPackages(test.Str)
		if err := comparePkgs(test.Exp, result, true); err != nil {
			t.Errorf("str %s failed to parse %v", test.Str, err)
		}
	}
}
