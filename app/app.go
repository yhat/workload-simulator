package app

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gorilla/handlers"
)

// Configuration for web app.
type AppConfig struct {
	// web stuff
	Host      string
	Port      int
	PublicDir string
	ViewsDir  string
	ReportDir string

	// Settings for worker concurrency and display settings for dials.
	MaxDial    int
	MaxWorkers int

	// used to define the target Ops server.
	OpsHost   string
	OpsApiKey string
	OpsUser   string
}

// App defines the app and configs
type App struct {
	// Configuration for web app.
	config *AppConfig

	// Go template map
	templates map[string]*template.Template

	// http router
	router http.Handler

	// channels for workers and statMonitor
	kill   chan int
	report chan string
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}

// New constructs a pointer to a new App and returns any error encountered.
func New(config *Config) (*App, error) {
	if config.Web.PublicDir == "" {
		config.Web.PublicDir = "/var/workload-simulator/public/static"
	}
	// create a new app config from config yaml and a new App.
	// OpsConfig can be nil on start since it is specified by the UI.
	appCfg := AppConfig{
		Host:      config.Web.Hostname,
		Port:      config.Web.HttpPort,
		PublicDir: config.Web.PublicDir,
		ViewsDir:  config.Web.ViewsDir,
		ReportDir: config.Web.ReportDir,

		MaxDial:    config.Settings.MaxDial,
		MaxWorkers: config.Settings.MaxWorkers,
	}

	app := App{
		config:    &appCfg,
		templates: make(map[string]*template.Template),
	}

	// Register handlers with ServeMux.
	r := http.NewServeMux()

	// Static assets
	pubDir := config.Web.PublicDir
	serveStatic := func(name string) {
		fs := http.FileServer(http.Dir(filepath.Join(pubDir, name)))
		prefix := "/static/" + name + "/"
		r.Handle(prefix, http.StripPrefix(prefix, fs))
	}

	serveStatic(`img`)
	serveStatic(`css`)
	serveStatic(`js`)
	serveStatic(`lang`)

	r.HandleFunc("/", app.handleRoot)
	r.HandleFunc("/workload", app.handleWorkload)
	r.HandleFunc("/ping", app.handlePing)
	r.HandleFunc("/unload", app.handleUnload)
	r.HandleFunc("/pause", app.handlePause)
	r.HandleFunc("/stats", app.handleStats)
	r.HandleFunc("/status", app.handleStatus)
	r.HandleFunc("/live", app.handleLive)
	r.HandleFunc("/live/stats", app.handleLiveStats)
	r.HandleFunc("/sql", app.handleSql)
	r.HandleFunc("/save", app.handleSave)
	r.HandleFunc("/kill", app.handleKill)

	// Add router to app.
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	app.router = loggedRouter

	return &app, nil
}

func (app *App) compileTemplates(viewsDir string) error {
	templatesListing, err := ioutil.ReadDir(viewsDir)
	if err != nil {
		return fmt.Errorf("%s", err.Error())
	}

	for _, info := range templatesListing {
		templatePath := filepath.Join(viewsDir, info.Name())
		t, err := template.New("").ParseFiles(templatePath)
		if err != nil {
			return fmt.Errorf("error parsing template %s: %v", info.Name(), err)
		}
		app.templates[info.Name()] = t
	}
	return nil
}

func (app *App) Render(name string, w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	if err := app.compileTemplates(app.config.ViewsDir); err != nil {
		msg := fmt.Sprintf("error compiling templates: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	templateName := name + ".html"
	t, _ := app.templates[templateName]
	if err := t.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("error rendering template %s: %v", templateName, err)
	}
}
