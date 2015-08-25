package app

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
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

// structs to parse workload JSON.
type modelData struct {
	Query string
	QPS   string
}

type queryData struct {
	Model string
	Input map[string]interface{}
}

type settings struct {
	OpsHost    string `json:"ops_host"`
	ApiKey     string `json:"ops_apikey"`
	User       string `json:"ops_user"`
	MaxDialVal int    `json:"dial_max_value"`
	Workers    string `json:"workers"`
}

type modelInput struct {
	// model name
	name string

	// input data
	input map[string]interface{}

	// queries per second
	qps int
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
			return
		}

		settings := &settings{}
		err = json.Unmarshal([]byte(s), settings)
		if err != nil {
			fmt.Printf("error parsing form settings: %v\n", err)
		}

		// This builds a map that maps a model prediction window to a model
		// input for Ops.
		wrk := make(map[string]*modelInput)
		for k, v := range work {
			q := queryData{}
			err = json.Unmarshal([]byte(v.Query), &q)
			if err != nil {
				fmt.Printf("error parsing workload form: %v\n", err)
				data := make(map[string]interface{})
				b, err := formatJSONresp(false, data)
				if err != nil {
					http.Error(w, "failed to parse workload data", http.StatusInternalServerError)

					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write(b)
			}

			modelName := q.Model
			input := q.Input

			iqps, err := strconv.Atoi(v.QPS)
			if err != nil {
				fmt.Println("could not parse qps into int:")
				return
			}
			// create model input for each window.
			modelInput := &modelInput{
				name:  modelName,
				input: input,
				qps:   iqps,
			}
			wrk[k] = modelInput

		}

		// Open file for csv output on a per workload basis.
		// The pause button event should close this file.
		batchId, err := uuid()
		if err != nil {
			log.Printf("error generating uuid: %v", err)
		}

		filename := app.config.ReportDir + "/" + "workload_data_" + batchId
		outfile, err := os.Create(filename)
		if err != nil {
			log.Printf("failed to create report file: %v", err)
		}
		app.reportfile = outfile

		if err = WriteHeader(outfile); err != nil {
			log.Printf("failed to write csv header: %v", err)
		}

		// Spawn goroutines and randomly assign work
		n := len(wrk)
		if n == 0 {
			http.Error(w, "no work to be done", http.StatusInternalServerError)
			return
		}
		nw, err := strconv.Atoi(settings.Workers)
		if err != nil {
			http.Error(w, "error parsing worker count", http.StatusInternalServerError)
			return
		}
		app.config.currentWorkers = nw

		for i := 0; i < nw; i++ {
			// Choose a model from the workload at random.
			modelId := strconv.Itoa(rand.Intn(n))
			model := wrk[modelId]
			work := &Workload{
				dt:         500 * time.Millisecond,
				batchId:    batchId,
				workerId:   i,
				opsHost:    settings.OpsHost,
				apiKey:     settings.ApiKey,
				user:       settings.User,
				nrequests:  model.qps,
				modelId:    modelId,
				modelName:  model.name,
				modelInput: model.input,
			}
			Worker(app.Statc, app.Killc, work)
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

// handlePause kills the worker goroutines.
func (app *App) handlePause(w http.ResponseWriter, r *http.Request) {
	nw := app.config.currentWorkers
	for i := 0; i < nw; i++ {
		app.Killc <- 1
	}
	// avoid trying to close a nil file handler
	if app.reportfile != nil {
		err := app.reportfile.Close()
		if err != nil {
			log.Printf("failed to close report file: %v", err)
		}
		app.reportfile = nil
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleStats asks worker goroutines to report stats to the app
func (app *App) handleStats(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	stats := make(map[string]int)
	select {
	case statReport := <-app.Reportc:
		var r map[string]int

		err := json.Unmarshal([]byte(statReport.requestPerS), &r)
		if err != nil {
			http.Error(w, "Stats error", http.StatusInternalServerError)
			return
		}
		data["running"] = true

		// iterate over models and map modelId to requests per second.
		ts := time.Now()
		records := make([]*CsvMetric, 0)
		for k, v := range r {
			stats[k] = int(v)
			csv := &CsvMetric{
				ts:           ts,
				batchId:      statReport.batchId,
				opsHost:      statReport.opsHost,
				opsUser:      statReport.user,
				opsModelName: statReport.modelName,
				nWorkers:     app.config.currentWorkers,
				reqSent:      statReport.requestSent,
				reqComplete:  statReport.requestDone,
				reqPerSec:    v,
			}
			records = append(records, csv)
		}
		data["stats"] = stats
		if app.reportfile != nil {
			if err := WriteCsv(app.reportfile, records); err != nil {
				log.Printf("failed to write csv stats in /stats handler: %v", err)
			}
		}
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
	nw := app.config.currentWorkers
	for i := 0; i < nw; i++ {
		app.Killc <- 1
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
