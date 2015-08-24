package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/yhat/ops2/src/alb"
	"github.com/yhat/ops2/src/db"
	"github.com/yhat/ops2/src/mps/tlsconfig"
	"github.com/yhat/util/dbutil/sqlutil"

	_ "github.com/timob/go-mysql"
)

type AppConfig struct {
	// Unique session key for cookie cryptography
	SecretSessionKey string
	// Directory with static assets
	StaticDir string
	// Connection string for the database
	DBConnStr string
	// Directory to store bundles
	BundleDir string
	ModelLogs string
	// Directory to store and load certificates for communcation with MPSs
	CertDir   string
	AccessLog string
	// Number of times a model should be replicated
	ModelReplication int

	// If enabled, actions which attempt to view or modify shared model settings
	// will either return empty sets or fail to go through.
	// This is for sandbox.yhathq.com, where it would be inappropriate to display
	// every registered user to a non-admin user.
	DisableSharing bool

	// If true, don't send phone home metrics
	IsDev bool

	// A unique name for phone home metrics such as 'sandbox'
	ServiceName string
}

type App struct {
	// Ze router
	router *alb.PredictionRouter
	// A cookie store for session tokens
	store *sessions.CookieStore
	// Web app database
	db *sqlutil.ReconnectingDB

	sup *alb.Supervisor

	// directory for static assests
	staticDir string

	// storage directory for model bundles
	bundleDir string

	modelLogsDir string
	// number of times a model should be replicated
	modelReplication int

	// If enabled, actions which attempt to view or modify shared model settings
	// will either return empty sets or fail to go through.
	sharingDisabled bool

	isdev       bool
	serviceName string
}

// Start begins a goroutine to perform App's various actions which must be
// constantly performed.
// This includes draining logs from the MPSs and other functionality that
// will be added later.
// Call the returned function to stop this goroutine.
func (app *App) Start() (stop func()) {
	cancel := make(chan struct{})
	stop = func() { close(cancel) }

	go func() {
		for {
			select {
			case <-cancel:
				return
			case <-time.After(500 * time.Millisecond):
			}
			app.sup.WriteLogs(app.modelLogsDir)
		}
	}()

	go func() {
		for {
			select {
			case <-cancel:
				return
			case <-time.After(time.Second):
			}
			auth, err := func() (*db.PredictionAuth, error) {
				tx, err := app.db.Begin()
				if err != nil {
					return nil, err
				}
				defer tx.Rollback()
				return db.GetAuth(tx)
			}()
			if err != nil {
				app.logf("could not update auth: %v", err)
				continue
			}
			app.router.SetAuth(auth)
		}
	}()
	// TODO: add instance and worker monitoring
	return
}

func (app *App) logf(format string, a ...interface{}) {
	// one day this might actually go somewhere beside stderr
	log.Printf(format, a...)
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}

func (app *App) serveFile(filename string) http.Handler {
	filename = filepath.Join(app.staticDir, filename)
	hf := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	}
	return http.HandlerFunc(hf)
}

func NewApp(config *AppConfig) (*App, error) {

	if config.ModelReplication < 1 {
		return nil, fmt.Errorf("model replication level cannot be less than 1")
	}
	var accessLog io.Writer
	if config.AccessLog == "" {
		accessLog = os.Stderr
	} else {
		flags := os.O_WRONLY | os.O_APPEND | os.O_CREATE
		file, err := os.OpenFile(config.AccessLog, flags, 0644)
		if err != nil {
			return nil, err
		}
		log.Println("using access log:", config.AccessLog)
		accessLog = file
	}

	appDB, err := sqlutil.NewReconnectingDB("mysql", config.DBConnStr)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %v", err)
	}

	// do the static directories exist?
	for _, dir := range []string{"css", "js", "fonts", "img", "model"} {
		d := filepath.Join(config.StaticDir, dir)
		_, err := os.Stat(d)
		if err != nil {
			return nil, fmt.Errorf("directory %s doesn't exist", d)
		}
	}

	serveDir := func(dirName string) http.Handler {
		d := filepath.Join(config.StaticDir, dirName)
		return http.StripPrefix("/"+dirName, http.FileServer(http.Dir(d)))
	}

	store := sessions.NewCookieStore([]byte(config.SecretSessionKey))

	client, err := tlsconfig.NewClient(config.CertDir)
	if err != nil {
		return nil, fmt.Errorf("certificate loading failed: %v", err)
	}

	app := App{
		store:            store,
		db:               appDB,
		staticDir:        config.StaticDir,
		bundleDir:        config.BundleDir,
		modelLogsDir:     config.ModelLogs,
		modelReplication: config.ModelReplication,
		isdev:            config.IsDev,
		serviceName:      config.ServiceName,
		sharingDisabled:  config.DisableSharing,
	}

	tx, err := app.db.Begin()
	if err != nil {
		return nil, err
	}
	workers, err := db.Workers(tx)
	if err != nil {
		return nil, err
	}
	deploymentReqs, err := db.DeploymentRequests(tx)
	if err != nil {
		return nil, err
	}
	tx.Rollback()
	c := alb.SupervisorConfig{
		Storage:       &storage{new(sync.Mutex), &app},
		TLSHandshaker: client,
		Workers:       workers,
		Deployments:   deploymentReqs,
	}
	app.sup, err = alb.NewSupervisor(c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize communcation with MPSs: %v", err)
	}
	app.sup.Logger = log.New(os.Stderr, "supervisor: ", log.LstdFlags)

	r := mux.NewRouter()
	r.PathPrefix("/css/").Handler(serveDir("css"))
	r.PathPrefix("/js/").Handler(serveDir("js"))
	r.PathPrefix("/fonts/").Handler(serveDir("fonts"))
	r.PathPrefix("/img/").Handler(serveDir("img"))

	r.HandleFunc("/login", app.handleLogin)
	r.HandleFunc("/register", app.handleRegister)
	r.HandleFunc("/verify-password", app.handleVerifyPassword)
	r.HandleFunc("/logout", app.handleLogout)

	r.Handle("/favicon.ico", app.serveFile("favicon.ico"))

	// deployment routes
	r.PathPrefix("/deployer").HandlerFunc(app.handleDeployment)
	r.HandleFunc("/verify", app.handleOldVerify)

	authedRouter := mux.NewRouter()
	authedRouter.NotFoundHandler = app.serveFile("404.html")

	authedRouter.PathPrefix("/model/").Handler(serveDir("model"))
	authedRouter.Handle("/", app.serveFile("index.html"))
	authedRouter.Handle("/account", app.serveFile("account.html"))
	authedRouter.HandleFunc("/user.json", app.handleUser)
	authedRouter.HandleFunc("/users/{name}", app.handleUserByName)

	// apikey stuff
	authedRouter.HandleFunc("/apikey", app.handleApikey)

	// model specific pages
	authedRouter.Handle("/models/{name}", app.serveFile("model/index.html"))
	authedRouter.HandleFunc("/models/{name}/json", app.handleModel)
	authedRouter.Handle("/models/{name}/scoring", app.serveFile("model/scoring.html"))
	authedRouter.Handle("/models/{name}/versions", app.serveFile("model/versions.html"))
	authedRouter.HandleFunc("/models/{name}/versions.json", app.handleModelVersions)
	authedRouter.Handle("/models/{name}/logs", app.serveFile("model/logs.html"))
	authedRouter.HandleFunc("/models/{name}/logs/json", app.handleModelLogs)
	authedRouter.Handle("/models/{name}/settings", app.serveFile("model/settings.html"))
	authedRouter.Handle("/models/{name}/form-builder", app.serveFile("model/form-builder.html"))
	authedRouter.HandleFunc("/models/{name}/redeploy/{version}", app.handleModelRedeploy)
	authedRouter.HandleFunc("/models/{name}/shared", app.handleModelSharedUsers)
	authedRouter.HandleFunc("/models/{name}/startshare/{user}", app.handleModelStartSharing)
	authedRouter.HandleFunc("/models/{name}/stopshare/{user}", app.handleModelStopSharing)
	authedRouter.HandleFunc("/models/{name}/example", app.handleModelExample)

	// model actions
	authedRouter.HandleFunc("/models/{name}/action/{action}", app.handleModelStateChange)

	authedRouter.HandleFunc("/models", app.handleUserModels)
	authedRouter.HandleFunc("/shared", app.handleSharedModels)
	authedRouter.HandleFunc("/whoami", app.handleWhoami)

	adminRouter := mux.NewRouter()
	authedRouter.PathPrefix("/admin").Handler(app.restrictAdmin(adminRouter))

	adminRouter.Handle("/admin", app.serveFile("admin/index.html"))
	adminRouter.Handle("/admin/users", app.serveFile("admin/users.html"))
	adminRouter.HandleFunc("/admin/users.json", app.handleUsersData)
	adminRouter.HandleFunc("/admin/users/create", app.handleUsersCreate)
	adminRouter.HandleFunc("/admin/users/delete", app.handleUserDelete)
	adminRouter.HandleFunc("/admin/users/makeadmin", app.handleUserMakeAdmin)
	adminRouter.HandleFunc("/admin/users/unmakeadmin", app.handleUserUnmakeAdmin)
	adminRouter.HandleFunc("/admin/users/setpass", app.handleUserSetPass)

	adminRouter.Handle("/admin/models", app.serveFile("admin/models.html"))
	adminRouter.HandleFunc("/admin/models.json", app.handleAllUserModels)

	adminRouter.HandleFunc("/admin/servers", app.handleServers)
	adminRouter.HandleFunc("/admin/servers.json", app.handleServersData)
	adminRouter.HandleFunc("/admin/servers/remove", app.handleServersRemove)

	// get initial authentication for models
	auth, err := func() (*db.PredictionAuth, error) {
		tx, err := app.db.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		return db.GetAuth(tx)
	}()
	if err != nil {
		return nil, err
	}

	// must enforce no caching for jsx pages
	noCaching := func(handler http.Handler) http.Handler {
		hf := func(w http.ResponseWriter, r *http.Request) {
			// see: http://goo.gl/itaIDo
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			handler.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hf)
	}

	h := handlers.LoggingHandler(accessLog, noCaching(app.restrict(authedRouter)))
	r.NotFoundHandler = h

	predRouter := alb.NewPredictionRouter(r, app.sup, auth)

	// TODO: add api routes

	app.router = predRouter

	return &app, nil
}
