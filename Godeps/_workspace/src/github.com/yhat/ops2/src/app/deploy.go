package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yhat/events/events"
	"github.com/yhat/ops2/src/db"
	"github.com/yhat/ops2/src/mps"
)

var modelnameValidator = regexp.MustCompile(`\A[a-zA-Z0-9_-]+\z`)

func isValidModelName(name string) bool {
	return modelnameValidator.MatchString(name)
}

type packages []mps.Package

func (p packages) String() string {
	s := make([]string, len(p))
	for i, pkg := range p {
		if pkg.Version == "" {
			s[i] = pkg.Name
		} else {
			s[i] = pkg.Name + "==" + pkg.Version
		}
	}
	return strings.Join(s, ",")
}

// handleDeployment listens on the /deployer route and handles requests
// from the R and Python clients.
func (app *App) handleDeployment(w http.ResponseWriter, r *http.Request) {

	// for phone home metrics
	start := time.Now()

	username, apikey, ok := r.BasicAuth()
	if !ok || username == "" {
		username = r.FormValue("username")
		apikey = r.FormValue("apikey")
	}

	if username == "" {
		http.Error(w, "No auth provided", http.StatusUnauthorized)
		return
	}

	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()

	user, err := db.GetUser(tx, username)
	if err != nil {
		if db.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("user not found"))
		} else {
			app.dbError(w, r, err)
		}
		return
	}
	if apikey != user.Apikey {
		http.Error(w, "APIKEY did not match", http.StatusUnauthorized)
		return
	}

	info, bundle, err := mps.ReadBundle(r)
	if err != nil {
		http.Error(w, "Could not parse upload request "+err.Error(), http.StatusBadRequest)
		return
	}

	modelSize := int64(len(bundle))

	// validate model info
	if !isValidModelName(info.Modelname) {
		http.Error(w, fmt.Sprintf("'%s' is an invalid model name", info.Modelname), http.StatusBadRequest)
		return
	}

	// Write bundle to disk
	timestamp := time.Now().UTC().Format("2006_01_02_15_04_05")
	bundleFilename := user.Name + "_" + info.Modelname + "_" + timestamp + ".json"
	b := filepath.Join(app.bundleDir, bundleFilename)
	log.Println(b)
	if err := ioutil.WriteFile(b, bundle, 0644); err != nil {
		http.Error(w, fmt.Sprintf("could not write model file to disk %v", err), http.StatusInternalServerError)
		return
	}

	// add bundle to database
	p := db.NewVersionParams{
		UserId:         user.Id,
		Model:          info.Modelname,
		Lang:           info.Lang,
		LangPackages:   info.LanguagePackages,
		UbuntuPackages: info.UbuntuPackages,
		SourceCode:     info.SourceCode,
		BundleFilename: bundleFilename,
	}
	versionNum, err := db.NewModelVersion(tx, &p)
	if err != nil {
		http.Error(w, "could not create new model version: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		app.dbError(w, r, err)
		return
	}

	clientIP := r.RemoteAddr

	go func(user, model string, version int) {
		app.logf("deployment of %s:%s:%d started", user, model, version)
		err := app.sup.Deploy(user, model, version)
		if err != nil {
			app.logf("deployment of %s:%s:%d failed %v", user, model, version, err)
		} else {
			app.logf("deployment of %s:%s:%d succeeded", user, model, version)
		}

		if app.isdev {
			return
		}

		// Send phone home metrics
		errmsg := ""
		if err != nil {
			errmsg = err.Error()
		}

		d := &events.Deployment{
			StartTime: start.Unix(),
			EndTime:   time.Now().Unix(),
			Username:  user,
			ModelName: model,
			ModelLang: info.Lang,
			ModelVer:  int64(version),
			Service:   app.serviceName,
			ClientIP:  clientIP,
			Error:     errmsg,
			ModelSize: modelSize,
			ModelDeps: packages(info.LanguagePackages).String(),
		}
		data, err := json.Marshal(&d)
		if err != nil {
			app.logf("could not marshal deployment: %v", err)
			return
		}

		resp, err := http.Post(events.Endpoint, "application/json", bytes.NewReader(data))
		if err != nil {
			app.logf("failed to make request: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				app.logf("could not read body: %v", err)
				return
			}
			app.logf("could not send info: %s %v", resp.Status, body)
		}

	}(user.Name, info.Modelname, versionNum)

	// send a JSON response to the user
	resp := struct {
		Status  string `json:"status"`
		Version int    `json:"version"`
	}{"Successfully deployed", versionNum}
	data, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// handleOldVerify handles the the /verify route hit by the R and Python
// clients.
func (app *App) handleOldVerify(w http.ResponseWriter, r *http.Request) {
	username, apikey, ok := r.BasicAuth()
	if !ok || username == "" {
		username = r.FormValue("username")
		apikey = r.FormValue("apikey")
	}

	if username == "" {
		http.Error(w, "No auth provided", http.StatusUnauthorized)
		return
	}

	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()

	user, err := db.GetUser(tx, username)
	if err != nil {
		if db.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("user not found"))
			app.logf("authentication failed %s:%s", username, apikey)
		} else {
			app.dbError(w, r, err)
		}
		return
	}
	if apikey != user.Apikey {
		app.logf("authentication failed %s:%s", username, apikey)
		http.Error(w, "APIKEY did not match", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": "true"}`))
}
