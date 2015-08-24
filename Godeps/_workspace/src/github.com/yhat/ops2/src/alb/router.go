package alb

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/yhat/ops2/src/db"
)

// PredictionRouter determines if a request should be a web request or a
// prediction request.
// It also does prediction request authentication.
type PredictionRouter struct {
	Logger *log.Logger

	webApp     http.Handler
	supervisor *Supervisor

	mu       *sync.Mutex
	fastauth fastauth
}

func NewPredictionRouter(app http.Handler, sup *Supervisor, auth *db.PredictionAuth) *PredictionRouter {

	fastauth := newFastAuth(auth)

	rr := &PredictionRouter{
		webApp:     app,
		supervisor: sup,
		Logger:     log.New(os.Stderr, "PredictionRouter: ", log.LstdFlags),
		mu:         new(sync.Mutex),
		fastauth:   fastauth,
	}
	return rr
}

func (rr *PredictionRouter) SetAuth(auth *db.PredictionAuth) {
	fastauth := newFastAuth(auth)
	rr.mu.Lock()
	rr.fastauth = fastauth
	rr.mu.Unlock()
	return
}

type apikeys map[string]struct{}

func (keys apikeys) contains(key string) bool {
	_, ok := keys[key]
	return ok
}

type modelIndex struct {
	user  string
	model string
}

// fastauth uses maps to improve lookup time
type fastauth struct {
	userApikeys  map[string]apikeys
	modelApikeys map[modelIndex]apikeys
}

func newFastAuth(auth *db.PredictionAuth) fastauth {
	userApikeys := make(map[string]apikeys)
	modelApikeys := make(map[modelIndex]apikeys)

	for user, keys := range auth.Users {
		userApikeys[user] = apikeys{
			keys.Apikey:   struct{}{},
			keys.ReadOnly: struct{}{},
		}
	}

	for _, shared := range auth.Shared {
		sharedUserKeys, ok := auth.Users[shared.User]
		if !ok {
			// shared user does not have an apikey
			continue
		}
		i := modelIndex{shared.Owner, shared.Model}
		keys, ok := modelApikeys[i]
		if !ok {
			keys = make(apikeys)
		}
		keys[sharedUserKeys.Apikey] = struct{}{}
		modelApikeys[i] = keys
	}

	return fastauth{userApikeys, modelApikeys}
}

func (rr *PredictionRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// is the request of the form "/{PART1}/models/{PART2}"?
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[2] != "models" || parts[3] == "" {
		rr.webApp.ServeHTTP(w, r)
		return
	}

	user := parts[1]
	model := parts[3]

	username, password, ok := r.BasicAuth()
	if !ok {
		http.Error(w, `{"error":"No basic authentication provided"}`,
			http.StatusUnauthorized)
		return
	}

	authed := false

	if user == username {
		// user is requesting their own model. check user's apikeys and read only apikeys
		rr.mu.Lock()
		apikeys, ok := rr.fastauth.userApikeys[user]
		rr.mu.Unlock()
		if ok {
			authed = apikeys.contains(password)
		}
	} else {
		// user is requesting another model, check shared apikeys
		i := modelIndex{user, model}
		rr.mu.Lock()
		apikeys, ok := rr.fastauth.modelApikeys[i]
		rr.mu.Unlock()
		if ok {
			authed = apikeys.contains(password)
		}
	}

	if !authed {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rr.supervisor.Predict(user, model, w, r)
}
