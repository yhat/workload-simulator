package app

import "net/http"

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
		"Port":       7077,
		"User":       "ec2",
		"Pass":       "",
		"DB":         "",
		"Settings":   s,
	}
	app.Render("index", w, r, data)
}

// handleWorkload sends workload to worker goroutines.
func (app *App) handleHelloWorld(w http.ResponseWriter, r *http.Request) {
	app.Render("helloWorld", w, r, map[string]interface{}{"Text": "Hello World!"})
}

// handleWorkload sends workload to worker goroutines.
func (app *App) handleWorkload(w http.ResponseWriter, r *http.Request) {

}

// handlePing returns ok or Timeout.
func (app *App) handlePing(w http.ResponseWriter, r *http.Request) {

}

func (app *App) handleUnload(w http.ResponseWriter, r *http.Request) {

}

// handlePause pauses the worker goroutines.
func (app *App) handlePause(w http.ResponseWriter, r *http.Request) {

}

// handleStats asks worker goroutines to report stats to the app
func (app *App) handleStats(w http.ResponseWriter, r *http.Request) {

}

// handleStatus asks worker goroutines to give their status
func (app *App) handleStatus(w http.ResponseWriter, r *http.Request) {

}

// handleLive handles the live server request
func (app *App) handleLive(w http.ResponseWriter, r *http.Request) {

}

// handleLive handles the live server request
func (app *App) handleLiveStats(w http.ResponseWriter, r *http.Request) {

}

// handleSave saves the workloads being run
func (app *App) handleSave(w http.ResponseWriter, r *http.Request) {

}

// handleKill kills all worker goroutines
func (app *App) handleKill(w http.ResponseWriter, r *http.Request) {

}
