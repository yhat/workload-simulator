package app

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/yhat/ops2/src/alb"
	"github.com/yhat/ops2/src/db"
	"github.com/yhat/phash"
)

// The default format for timestamps. If you are rendering a time.Time
// value, first format it with this value.
// For example:
//
//    t := time.Now().Format(ScienceOpsDateFormat)
//
const ScienceOpsDateFormat = "Jan _2, 2006 15:04 MST"

func (app *App) dbError(w http.ResponseWriter, r *http.Request, err error) {
	stackTrace := debug.Stack()
	app.logf("database error: %v\n%s", err, stackTrace)
	http.Error(w, "Database unavailable", 500)
}

func (app *App) writeJson(w http.ResponseWriter, data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		app.logf("could not marshall data: %v", err)
		http.Error(w, "Unrecognized internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		app.serveFile("login.html").ServeHTTP(w, r)
	case "POST":

		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" {
			http.Error(w, "No username provided", http.StatusBadRequest)
			return
		}
		if password == "" {
			http.Error(w, "No password provided", http.StatusBadRequest)
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
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				app.dbError(w, r, err)
			}
			return
		}
		if phash.Verify(password, user.Password) {
			u := &User{Id: user.Id, Name: user.Name}
			app.setUser(r, w, u)
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "Invalid username passsword combination.", http.StatusBadRequest)
		}

	default:
		http.Error(w, "I only respond to GET and POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleVerifyPassword(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		// user should already be logged in, we're just validating the password
		password := r.FormValue("password")
		sessionUser, ok := app.getUser(r)

		if !ok {
			http.Error(w, "No user logged in", http.StatusBadRequest)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		user, err := db.GetUser(tx, sessionUser.Name)
		if err != nil {
			if db.IsNotFound(err) {
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				app.dbError(w, r, err)
			}
			return
		}

		if password == "" {
			http.Error(w, "No password provided", http.StatusBadRequest)
			return
		}

		if phash.Verify(password, user.Password) {
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, "Invalid username passsword combination.", http.StatusBadRequest)
		}

	default:
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	app.logout(r, w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (app *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Register is only displayed if there are no users on the system.
	// It is only for the inital login.
	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()

	users, err := db.AllUsers(tx)
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	if len(users) != 0 {
		if r.Method == "GET" {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		return
	}

	if r.Method == "GET" {
		app.serveFile("register.html").ServeHTTP(w, r)
		return
	} else if r.Method != "POST" {
		http.Error(w, "I only respond to GET and POSTs", http.StatusNotImplemented)
		return
	}

	username := r.PostFormValue("username")
	pass := r.PostFormValue("password")
	email := r.PostFormValue("email")

	if username == "" {
		http.Error(w, "No username provided", http.StatusBadRequest)
		return
	}

	if pass == "" {
		http.Error(w, "Empty password provided", http.StatusBadRequest)
		return
	}
	hashedPass := phash.Gen(pass)

	user, err := db.NewUser(tx, username, hashedPass, email, true)
	if err != nil {
		http.Error(w, "Could not save user to database: "+err.Error(),
			http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		app.dbError(w, r, err)
		return
	}

	u := &User{Id: user.Id, Name: user.Name}
	if err := app.setUser(r, w, u); err != nil {
		http.Error(w, "Failed to set session cookie: "+err.Error(),
			http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *App) handleUsersCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
		return
	}

	username := r.PostFormValue("username")
	pass := r.PostFormValue("password")
	email := r.PostFormValue("email")
	admin := r.PostFormValue("admin") == "true"

	if username == "" {
		http.Error(w, "No username provided", http.StatusBadRequest)
		return
	}

	if pass == "" {
		http.Error(w, "Empty password provided", http.StatusBadRequest)
		return
	}
	hashedPass := phash.Gen(pass)

	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()

	if _, err := db.NewUser(tx, username, hashedPass, email, admin); err != nil {
		http.Error(w, "Could not save user to database: "+err.Error(),
			http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		app.dbError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func (app *App) handleUsersData(w http.ResponseWriter, r *http.Request) {
	// it's already implied that this user is an admin, we can skip auth
	switch r.Method {
	case "GET":
		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		users, err := db.AllUsers(tx)
		if err != nil {
			app.dbError(w, r, err)
			return
		}

		// ViewUser is a type which is okay to display in the view.
		// It omits fields like "password" (because we would NEVER display
		// something like that).
		type ViewUser struct {
			Name     string
			Email    string
			Admin    bool
			Apikey   string
			ROApikey string
		}
		viewUsers := make([]ViewUser, len(users))
		for i, user := range users {
			viewUsers[i] = ViewUser{
				user.Name, user.Email, user.Admin,
				user.Apikey, user.ReadOnlyApikey,
			}
		}
		app.writeJson(w, viewUsers)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
		return
	}
	u, ok := app.getUser(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()
	user, err := db.GetUser(tx, u.Name)
	if err != nil {
		if db.IsNotFound(err) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			app.dbError(w, r, err)
		}
		return
	}
	viewUser := struct {
		Name   string
		Email  string
		Admin  bool
		Apikey string
	}{user.Name, user.Email, user.Admin, user.Apikey}
	app.writeJson(w, &viewUser)
}

func (app *App) handleUserByName(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		vars := mux.Vars(r)
		username := vars["name"]
		// only return all user data if the session user has
		// admin privlages. Need to check the session user against
		// the User table.
		u, ok := app.getUser(r)
		if !ok || (u.Name != username) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				app.dbError(w, r, err)
			}
			return
		}
		app.writeJson(w, user)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		app.serveFile("404.html").ServeHTTP(w, r)
	case "POST":
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "no username provided", http.StatusBadRequest)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		bundles, err := db.DeleteUser(tx, username)
		if err != nil {
			http.Error(w, "could not delete user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			app.dbError(w, r, err)
			return
		}

		app.sup.DeleteUser(username)
		for _, bundle := range bundles {
			bundle := filepath.Join(app.bundleDir, bundle)
			if err := os.Remove(bundle); err != nil {
				app.logf("failed to remove bundle %s: %v", bundle, err)
			}
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleUserMakeAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		app.serveFile("404.html").ServeHTTP(w, r)
	case "POST":
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "no username provided", http.StatusBadRequest)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		if err := db.MakeAdmin(tx, username); err != nil {
			http.Error(w, "could not make admin: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			app.dbError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleUserUnmakeAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		app.serveFile("404.html").ServeHTTP(w, r)
	case "POST":
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "no username provided", http.StatusBadRequest)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		if err := db.UnmakeAdmin(tx, username); err != nil {
			http.Error(w, "could not make admin: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			app.dbError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleUserSetPass(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		app.serveFile("404.html").ServeHTTP(w, r)
	case "POST":
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")

		if username == "" || password == "" {
			http.Error(w, "must provide username and password fields", http.StatusBadRequest)
			return
		}
		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		if err := db.SetPass(tx, username, phash.Gen(password)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()
		if err := tx.Commit(); err != nil {
			app.dbError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleUserModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		u, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()
		models, err := db.UserModels(tx, u.Name)

		if err != nil {
			app.dbError(w, r, err)
			return
		}
		app.writeJson(w, models)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleAllUserModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		_, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// TODO
		// if u.Admin == false {
		// 	http.Error(w, "Unauthorized", http.StatusUnauthorized)
		// 	return
		// }

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		allUsers, err := db.AllUsers(tx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		allModels := []db.Model{}

		for _, user := range allUsers {

			models, err := db.UserModels(tx, user.Name)
			for i, _ := range models {
				models[i].Owner = user.Name
			}

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			allModels = append(allModels, models...)
		}

		if err != nil {
			app.dbError(w, r, err)
			return
		}
		app.writeJson(w, allModels)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleModelVersions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		u, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		vars := mux.Vars(r)
		modelName := vars["name"]
		if modelName == "" {
			http.Error(w, "No model id provided", http.StatusBadRequest)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()
		versions, err := db.GetModelVersions(tx, u.Name, modelName)
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		app.writeJson(w, versions)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

type sharedUser struct {
	Id     int64
	Name   string
	Shared bool
}

type sharedByName []sharedUser

func (s sharedByName) Len() int           { return len(s) }
func (s sharedByName) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s sharedByName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (app *App) handleModelSharedUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		reqUser, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if app.sharingDisabled {
			// If sharing is disabled return an empty list
			w.Write([]byte(`[]`))
			return
		}

		modelname := mux.Vars(r)["name"]

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		users, err := db.ModelSharedUsers(tx, reqUser.Name, modelname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		isShared := func(userId int64) bool {
			for _, user := range users {
				if user.Id == userId {
					return true
				}
			}
			return false
		}

		allUsers, err := db.AllUsers(tx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		shared := []sharedUser{}

		for _, user := range allUsers {
			if user.Id == reqUser.Id {
				continue
			}

			shared = append(shared, sharedUser{
				Id:     user.Id,
				Name:   user.Name,
				Shared: isShared(user.Id),
			})
		}

		app.writeJson(w, shared)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleModelStartSharing(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
		return
	}
	reqUser, ok := app.getUser(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if app.sharingDisabled {
		http.Error(w, "Sharing is disabled", http.StatusUnauthorized)
		return
	}

	modelname := mux.Vars(r)["name"]
	username := mux.Vars(r)["user"]

	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := db.StartSharing(tx, reqUser.Name, modelname, username); err != nil {
		http.Error(w, "Could not start sharing "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		app.dbError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *App) handleModelStopSharing(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
		return
	}
	reqUser, ok := app.getUser(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if app.sharingDisabled {
		http.Error(w, "Sharing is disabled", http.StatusUnauthorized)
		return
	}

	modelname := mux.Vars(r)["name"]
	username := mux.Vars(r)["user"]

	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := db.StopSharing(tx, reqUser.Name, modelname, username); err != nil {
		http.Error(w, "Could not stops sharing "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		app.dbError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *App) handleWhoami(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		user, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		u, err := db.GetUser(tx, user.Name)
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		resp := struct {
			Username       string `json:"username"`
			Apikey         string `json:"apikey"`
			ReadOnlyApikey string `json:"read_only_apikey"`
		}{user.Name, u.Apikey, u.ReadOnlyApikey}
		data, err := json.Marshal(&resp)
		if err != nil {
			http.Error(w, "Unrecognized internal error", 500)
			app.logf("could not encode json: %v", err)
			return
		}
		w.Write(data)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleModel(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		vars := mux.Vars(r)
		name := vars["name"]
		user, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()
		model, err := db.GetModel(tx, user.Name, name)
		if err != nil {
			if db.IsNotFound(err) {
				http.Error(w, "Model not found", http.StatusNotFound)
			} else {
				app.dbError(w, r, err)
			}
			return
		}
		v := model.ActiveVersion
		activeVersion, err := db.GetModelVersion(tx, user.Name, name, v)
		if err != nil {
			if db.IsNotFound(err) {
				http.Error(w, "Model not found", http.StatusNotFound)
			} else {
				app.dbError(w, r, err)
			}
			return
		}
		app.writeJson(w, map[string]interface{}{
			"Model":   model,
			"Version": activeVersion,
		})
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleModelExample(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET", "POST":
		modelname := mux.Vars(r)["name"]
		user, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		if r.Method == "GET" {
			example, err := db.ModelExample(tx, user.Name, modelname)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				w.Write([]byte(example))
			}
			return
		}

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "could not read body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = db.SetModelExample(tx, user.Name, modelname, string(data))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			app.dbError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "I only respond to GET and POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleSharedModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
		return
	}
	user, ok := app.getUser(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()
	models, err := db.SharedModels(tx, user.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// format the timestamps as a string for viewing on the front end.
	type SharedModel struct {
		Owner       string
		Name        string
		LastUpdated string
	}
	sharedModels := make([]SharedModel, len(models))
	for i, model := range models {
		sharedModels[i] = SharedModel{
			Owner:       model.Owner,
			Name:        model.Name,
			LastUpdated: model.LastUpdated.Format(ScienceOpsDateFormat),
		}
	}
	app.writeJson(w, sharedModels)
}

func (app *App) handleModelLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		vars := mux.Vars(r)
		name := vars["name"]
		user, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()
		model, err := db.GetModel(tx, user.Name, name)
		if err != nil {
			if db.IsNotFound(err) {
				http.Error(w, "Model not found", http.StatusNotFound)
			} else {
				app.dbError(w, r, err)
			}
			return
		}
		logLines, err := alb.ReadDeploymentLogs(app.modelLogsDir,
			user.Name, name, model.LastDeployment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		app.writeJson(w, logLines)
	default:
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
	}
}

func (app *App) handleApikey(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		q := r.URL.Query()
		isReadOnly := q.Get("readonly") == "true"

		user, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tx, err := app.db.Begin()
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		defer tx.Rollback()

		newApikey, err := uuid()
		if err != nil {
			http.Error(w, "Failed to generate apikey: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = db.UpdateApiKey(tx, user.Id, newApikey, isReadOnly)
		if err != nil {
			app.dbError(w, r, err)
			return
		}
		if err := tx.Commit(); err != nil {
			app.dbError(w, r, err)
			return
		}

		resp := struct {
			Username string `json:"username"`
			Apikey   string `json:"apikey"`
		}{user.Name, newApikey}
		data, err := json.Marshal(&resp)
		w.Write(data)

	default:
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
	}
}

func (app *App) handleModelRedeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
		return
	}

	user, ok := app.getUser(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	modelname := vars["name"]
	version, err := strconv.Atoi(vars["version"])
	if err != nil {
		http.Error(w, "Invalid version provided "+vars["version"], http.StatusBadRequest)
		return
	}

	if err := app.sup.Deploy(user.Name, modelname, version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *App) handleModelStateChange(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		user, ok := app.getUser(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		vars := mux.Vars(r)
		modelname := vars["name"]
		action := vars["action"]
		if modelname == "" {
			http.Error(w, "no modelname provided", http.StatusBadRequest)
			return
		}
		log.Printf("modelname '%s' action '%s'", modelname, action)

		var err error
		switch action {
		case "restart":
			err = app.sup.Restart(user.Name, modelname)
		case "sleep":
			app.sup.Sleep(user.Name, modelname)
		case "wake":
			err = app.sup.Wake(user.Name, modelname)
		case "delete":
			tx, err := app.db.Begin()
			if err != nil {
				app.dbError(w, r, err)
				return
			}
			defer tx.Rollback()

			bundles, err := db.DeleteModel(tx, user.Name, modelname)
			if err != nil {
				msg := fmt.Sprintf("Could not delete model from db: %v", err)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			if err := tx.Commit(); err != nil {
				app.dbError(w, r, err)
			}
			go func(bundles []string) {
				for _, bundle := range bundles {
					bundle := filepath.Join(app.bundleDir, bundle)
					if err := os.Remove(bundle); err != nil {
						app.logf("could not remove bundle '%s': %v", bundle, err)
					}
				}
			}(bundles)

			app.sup.Delete(user.Name, modelname)

		default:
			msg := fmt.Sprintf("Method %s not implemented for models", action)
			http.Error(w, msg, http.StatusNotImplemented)
			return
		}
		if err != nil {
			msg := fmt.Sprintf("%s failed: %v", action, err)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		// reply with OK status
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
	}
}

// doTransaction beings and attempts to commit a database transaction.
// Between those actions, the provided function is called with the given
// transcation object.
// If transaction returns an error, the transaction is rolled back.
func (app *App) doTransaction(transaction func(tx *sql.Tx) error) error {
	tx, err := app.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := transaction(tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (app *App) handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		app.serveFile("admin/servers.html").ServeHTTP(w, r)
	case "POST":

		hostPort := r.FormValue("host")
		if hostPort == "" {
			http.Error(w, "No host provided", http.StatusBadRequest)
			return
		}
		host, port, err := net.SplitHostPort(hostPort)
		if err != nil || host == "" || port == "" {
			http.Error(w, "Invalid 'host:port' combination provided", http.StatusBadRequest)
			return
		}

		// attempt to add the new worker to the database and supervisor's rotation
		trans := func(tx *sql.Tx) error {
			worker, err := db.NewWorker(tx, hostPort)
			if err != nil {
				return err
			}
			if err = app.sup.AddWorker(hostPort, worker.Id); err != nil {
				return err
			}
			return err
		}
		if err := app.doTransaction(trans); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not implemented", http.StatusNotImplemented)
	}
}

func (app *App) handleServersRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not implemented", http.StatusNotImplemented)
		return
	}
	idStr := r.FormValue("id")
	if idStr == "" {
		http.Error(w, "No host Id", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid Id provided", http.StatusBadRequest)
		return
	}

	// attempt to reomve the worker from the database
	trans := func(tx *sql.Tx) error {
		return db.RemoveWorker(tx, id)
	}
	if err = app.doTransaction(trans); err != nil {
		app.dbError(w, r, err)
		return
	}
	app.sup.RemoveWorker(id)
	w.WriteHeader(http.StatusOK)
}

type workersById []db.Worker

func (w workersById) Len() int           { return len(w) }
func (w workersById) Less(i, j int) bool { return w[i].Id < w[j].Id }
func (w workersById) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }

func (app *App) handleServersData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "I only respond to GETs", http.StatusNotImplemented)
		return
	}
	tx, err := app.db.Begin()
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	defer tx.Rollback()
	workers, err := db.Workers(tx)
	if err != nil {
		app.dbError(w, r, err)
		return
	}
	sort.Sort(workersById(workers))
	app.writeJson(w, workers)
}

func uuid() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
