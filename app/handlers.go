package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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
	data := map[string]interface{}{
		"Live":       false,
		"Redirected": false,
		"MaxDial":    app.config.MaxDial,
		"Host":       app.config.OpsHost,
		"ApiKey":     app.config.OpsApiKey,
		"User":       app.config.OpsUser,
		"Workers":    app.config.MaxWorkers,
	}
	app.Render("index", w, r, data)
}

type modelData struct {
	Query string
	QPS   int
}

type queryData struct {
	Model string
	Input map[string]interface{}
}

// handleWorkload sends workload to worker goroutines.
func (app *App) handleWorkload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		// Decode json-encoded form values
		wl := r.FormValue("workload")
		s := r.FormValue("settings")

		work := make(map[string]modelData)
		err := json.Unmarshal([]byte(wl), &work)
		if err != nil {
			fmt.Printf("error parsing workload form: %v\n", err)
		}

		settings := &Settings{}
		err = json.Unmarshal([]byte(s), settings)
		if err != nil {
			fmt.Printf("error parsing form settings: %v\n", err)
		}

		// This builds a map that maps a model prediction window to a model
		// input for Ops.
		wrk := make(map[string]*ModelInput)
		for k, v := range work {
			q := queryData{}
			err = json.Unmarshal([]byte(v.Query), &q)
			if err != nil {
				fmt.Printf("error parsing workload form: %v\n", err)
			}

			modelName := q.Model
			input := q.Input

			// create model input for each window.
			modelInput := &ModelInput{
				name:  modelName,
				input: input,
				qps:   v.QPS,
			}
			wrk[k] = modelInput

		}

		workld := Workload{
			workload: wrk,
			settings: settings,
		}
		// TODO: iterate over workload map and create a work struct for each
		// window.
		wk0 := &Work{modelId: "0", workload: &workld, reqTotal: 100000}

		kill := make(chan int)
		report := make(chan string)
		stats := StatsMonitor(report, kill, 100*time.Millisecond)
		app.kill = kill
		app.report = report

		for i := 0; i < 100; i++ {
			Worker(stats, kill, time.Second, wk0)
		}

	default:
		http.Error(w, "I only respond to POSTs.", http.StatusNotImplemented)
	}

	data := make(map[string]interface{})
	b, err := formatJSONresp(true, data)
	if err != nil {
		http.Error(w, "failed to marshal data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func formatJSONresp(running bool, data map[string]interface{}) ([]byte, error) {
	data["running"] = running
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return b, nil
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

}

// handleStats asks worker goroutines to report stats to the app
func (app *App) handleStats(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	stats := make(map[string]int)
	select {
	case statReport := <-app.report:
		var report map[string]int
		err := json.Unmarshal([]byte(statReport), &report)
		if err != nil {
			http.Error(w, "Stats error", http.StatusInternalServerError)
			return
		}
		data["running"] = true
		for k, v := range report {
			stats[k] = int(v)
		}
		data["stats"] = stats
		fmt.Printf("data = %v", data)

	case <-time.After(time.Second):
		data["running"] = false
	}

	b, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "failed to marshal data", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(b)
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
