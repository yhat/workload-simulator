package app

import (
	"encoding/gob"
	"net/http"

	"github.com/yhat/ops2/src/db"
)

func init() {
	gob.Register(&User{})
}

// User definition is purposefully kept minimal.
// Because information like APIKeys and admin status can change, we want
// the app to have to check the db for that data.
type User struct {
	Id   int64
	Name string
}

var userSessionName = "scienceops-user"

func (app *App) getUser(r *http.Request) (*User, bool) {
	session, _ := app.store.Get(r, userSessionName)
	user, ok := session.Values["user"].(*User)
	if ok {
		return user, true
	}
	return nil, false
}

func (app *App) setUser(r *http.Request, w http.ResponseWriter, u *User) error {
	session, _ := app.store.Get(r, userSessionName)
	session.Values["user"] = u
	return session.Save(r, w)
}

func (app *App) logout(r *http.Request, w http.ResponseWriter) error {
	session, _ := app.store.Get(r, userSessionName)
	delete(session.Values, "user")
	return session.Save(r, w)
}

func (app *App) restrict(h http.Handler) http.Handler {
	hf := func(w http.ResponseWriter, r *http.Request) {
		if _, ok := app.getUser(r); ok {
			h.ServeHTTP(w, r)
			return
		}
		if r.Method == "GET" {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
	return http.HandlerFunc(hf)
}

func (app *App) restrictAdmin(h http.Handler) http.Handler {
	hf := func(w http.ResponseWriter, r *http.Request) {
		if user, ok := app.getUser(r); ok {
			tx, err := app.db.Begin()
			if err != nil {
				app.dbError(w, r, err)
				return
			}

			u, err := db.GetUser(tx, user.Name)
			tx.Rollback()

			if err != nil {
				app.dbError(w, r, err)
				return
			}

			if u.Admin {
				h.ServeHTTP(w, r)
				return
			}
		}
		if r.Method == "GET" {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
	return http.HandlerFunc(hf)
}
