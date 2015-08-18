package app

import "net/http"

// OpsConfig is a type used to define the target Ops
// server
type OpsConfig struct {
	Host   string
	ApiKey string
	User   string
}

type AppConfig struct {
	// Configuration for web app.
	Host      string
	Port      int
	MaxDial   int
	ReportDir string
}

// App defines the app and configs
type App struct {
	// Configuration for web app.
	host      string
	port      int
	maxDial   int
	reportDir string

	// http router
	router http.Handler

	// Configuration for target ops server.
	ops *OpsConfig

	// Worker configuration.
	workerProcs int
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}

// NewApp constructs a pointer to a new App and returns any error
// encountered.
func NewApp(config *AppConfig) (*App, error) {
	// OpsConfig can be nil on start since it is specified by
	// the UI. Same with workerProcs
	app := App{
		host:      config.Host,
		port:      config.Port,
		maxDial:   config.MaxDial,
		reportDir: config.ReportDir,
	}

	// Register handlers with ServeMux
	r := http.NewServeMux()

	r.HandleFunc("/", app.handleRoot)
	r.HandleFunc("/workload", app.handleWorkload)
	r.HandleFunc("/ping", app.handlePing)
	r.HandleFunc("/unload", app.handleUnload)
	r.HandleFunc("/pause", app.handlePause)
	r.HandleFunc("/stats", app.handleStats)
	r.HandleFunc("/status", app.handleStatus)
	r.HandleFunc("/live", app.handleLive)
	r.HandleFunc("/live/stats", app.handleLiveStats)
	r.HandleFunc("/save", app.handleSave)
	r.HandleFunc("/kill", app.handleKill)

	// add router to app
	app.router = r
	return &app, nil
}
