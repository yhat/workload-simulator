package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// handleRoot renders the home page or redirects if ping timeout is
// reached.
func (app *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	// The "/" pattern matches with everything, so we need to check
	// that we are root here.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s := struct {
		memsql_host string
		memsql_port string
		memsql_user string
		memsql_pass string
		memsql_db   string
		workers     string
	}{"memhost", "6000", "bob", "foo", "mydb", "100"}
	data := map[string]interface{}{
		"Live":       false,
		"Redirected": false,
		"MaxDial":    50,
		"Host":       "sandbox.yhathq.com",
		"ApiKey":     "2463b3c71264ef61de1f6af8338d22e7",
		"User":       "colin",
		"Pass":       "",
		"DB":         "",
		"Workers":    50,
		"Settings":   s,
	}
	app.Render("index", w, r, data)
}

// handleWorkload sends workload to worker goroutines.
func (app *App) handleWorkload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		// Decode json-encoded form values
		wl := r.FormValue("workload")
		s := r.FormValue("settings")

		var work map[string]interface{}
		err := json.Unmarshal([]byte(wl), &work)
		if err != nil {
			fmt.Printf("error parsing workload form: %v\n", err)
		}

		settings := &Settings{}
		err = json.Unmarshal([]byte(s), settings)
		if err != nil {
			fmt.Printf("error parsing form settings: %v\n", err)
		}

		// Some nasty type conversion to parse a nested json. This builds
		// a map that maps a window to a model input for Ops.
		var wrk map[string]*ModelInput
		wrk = make(map[string]*ModelInput)
		for k, v := range work {
			vv := v.(map[string]interface{})

			qps := vv["qps"]
			iqps, err := strconv.Atoi(qps.(string))
			if err != nil {
				http.Error(w, "error parsing model inputs", http.StatusInternalServerError)
			}

			query := vv["query"]
			var mm map[string]interface{}
			err = json.Unmarshal([]byte(query.(string)), &mm)
			if err != nil {
				fmt.Printf("error parsing workload form: %v\n", err)
			}

			modelName := mm["model"]
			input := mm["input"]

			// create model input for each window.
			modelInput := &ModelInput{
				name:  modelName.(string),
				input: input.(map[string]interface{}),
				qps:   iqps,
			}
			wrk[k] = modelInput

		}

		workld := Workload{
			workload: wrk,
			settings: settings,
		}
		for i := 0; i < 100; i++ {
			err := workld.Predict()
			if err != nil {
				http.Error(w, "Bad prediction", http.StatusBadRequest)
			}
		}
	default:
		http.Error(w, "I only respond to POSTs.", http.StatusNotImplemented)
	}

	w.WriteHeader(http.StatusOK)
}

// handlePing returns ok or Timeout.
func (app *App) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (app *App) handleUnload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handlePause pauses the worker goroutines.
func (app *App) handlePause(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleStats asks worker goroutines to report stats to the app
func (app *App) handleStats(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleStatus asks worker goroutines to give their status
func (app *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleLive handles the live server request
func (app *App) handleLive(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleLive handles the live server request
func (app *App) handleLiveStats(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleSql sql connection
func (app *App) handleSql(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleSave saves the workloads being run
func (app *App) handleSave(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleKill kills all worker goroutines
func (app *App) handleKill(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
