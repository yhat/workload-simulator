package db

import (
	"database/sql"
	"testing"
)

// NewTestDB should be called by each test to create a connection to a fresh
// mysql database. It is set to a func value in this code by TestMain()
var NewTestDB func() (*sql.DB, error)

type TestUser struct {
	in   string
	want User
}

func TestAllUsers(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		usernames := []string{"eric", "greg", "ryan", "brandon", "sush"}
		for _, name := range usernames {
			_, err := NewUser(tx, name, "pass", name+"@yhathq.com", true)
			if err != nil {
				t.Errorf("could not add user: %v", err)
				return
			}
		}

		users, err := AllUsers(tx)
		if err != nil {
			t.Errorf("failed to get all users: %v", err)
			return
		}
		if len(users) != 5 {
			t.Errorf("expected 5 users got: %d", len(users))
		}
		hasUser := func(name string) bool {
			for _, user := range users {
				if user.Name == name {
					return true
				}
			}
			return false
		}

		for _, name := range usernames {
			if !hasUser(name) {
				t.Errorf("no user named %s in resulting users", name)
			}
		}
	})
}

func TestSetPass(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		bob := "bigdatabob"
		_, err := NewUser(tx, bob, "pass", bob+"@yhathq.com", true)
		if err != nil {
			t.Errorf("could not add user: %v", err)
			return
		}
		if err := SetPass(tx, bob, "newpass"); err != nil {
			t.Errorf("could not set password: %v", err)
		}
		user, err := GetUser(tx, bob)
		if err != nil {
			t.Errorf("could not get user: %v", err)
			return
		}
		got := user.Password
		if "newpass" != got {
			t.Errorf("expected password '%s' got '%s'", "newpass", got)
		}
	})
}

func TestGetUser(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		user, err := NewUser(tx, "bigdatabob", "pass", "bigdatabob@bigdata.com", true)
		if err != nil {
			t.Error(err)
			return
		}
		u, err := GetUser(tx, "bigdatabob")
		if err != nil {
			t.Errorf("could not get user: %v", err)
			return
		}
		checks := []struct {
			ok   bool
			name string
		}{
			{user.Id == u.Id, "id"},
			{user.Name == u.Name, "name"},
			{user.Password == u.Password, "password"},
			{user.Email == u.Email, "email"},
			{user.Admin == u.Admin, "admin"},
		}
		for _, check := range checks {
			if !check.ok {
				t.Errorf("value of %s did not match", check.name)
			}
		}
	})
}

func TestSetAdmin(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		name := "bigdatabob"
		_, err := NewUser(tx, name, "pass", "bigdatabob@bigdata.com", true)
		if err != nil {
			t.Error(err)
			return
		}
		if err := UnmakeAdmin(tx, name); err == nil {
			t.Errorf("expected removing admin privileges from only admin to fail")
		}

		_, err = NewUser(tx, "hadoopheather", "pass", "hadoopheather@yhathq.com", true)
		if err != nil {
			t.Error(err)
			return
		}

		if err := UnmakeAdmin(tx, name); err != nil {
			t.Errorf("unmake admin: %v", err)
		}
		u, err := GetUser(tx, name)
		if err != nil {
			t.Error(err)
			return
		}
		if u.Admin {
			t.Error("expected bigdatabob to not be admin")
		}
		if err := MakeAdmin(tx, name); err != nil {
			t.Errorf("unmake admin: %v", err)
		}
		u, err = GetUser(tx, name)
		if err != nil {
			t.Error(err)
			return
		}
		if !u.Admin {
			t.Error("expected bigdatabob to not be an admin")
		}
	})
}
func TestAddWorkers(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		hosts := []string{"10.0.0.1:70", "10.0.0.1:23"}
		for _, host := range hosts {
			_, err := NewWorker(tx, host)
			if err != nil {
				t.Errorf("could not add user: %v", err)
				return
			}
		}

		workers, err := Workers(tx)
		if err != nil {
			t.Errorf("could not get workers: %v", err)
			return
		}
		nHosts := len(hosts)
		if n := len(workers); n != nHosts {
			t.Errorf("expected %d workers got %d", nHosts, n)
			return
		}

		for _, worker := range workers {
			if err := worker.Remove(tx); err != nil {
				t.Errorf("failed to remove worker: %v", err)
				return
			}
			nHosts--
			workers, err := Workers(tx)
			if err != nil {
				t.Errorf("could not get workers: %v", err)
				return
			}
			if n := len(workers); n != nHosts {
				t.Errorf("expected %d workers got %d", nHosts, n)
				return
			}
		}
	})
}

func TestGetAuth(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		usernames := []string{"eric", "greg", "ryan", "brandon", "sush"}
		for _, name := range usernames {
			_, err := NewUser(tx, name, "pass", name+"@yhathq.com", true)
			if err != nil {
				t.Errorf("could not add user: %v", err)
				return
			}
		}
		auth, err := GetAuth(tx)
		if err != nil {
			t.Errorf("could not get auth: %v", err)
			return
		}
		if act, exp := len(auth.Users), len(usernames); act != exp {
			t.Errorf("expected %d users got %d", exp, act)
		}
	})
}

func TestSharedUsers(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		usernames := []string{"eric", "greg", "ryan", "brandon", "sush"}
		users := make([]*User, len(usernames))
		for i, name := range usernames {
			user, err := NewUser(tx, name, "pass", name+"@yhathq.com", true)
			if err != nil {
				t.Errorf("could not add user: %v", err)
				return
			}
			users[i] = user
		}

		user := users[0]
		params := NewVersionParams{
			UserId:         user.Id,
			Model:          "hellopy",
			Lang:           LangPython2,
			BundleFilename: "bundle.json",
		}
		_, err := NewModelVersion(tx, &params)
		if err != nil {
			t.Errorf("could not create model version: %v", err)
			return
		}

		for i, name := range usernames {
			if err := StartSharing(tx, user.Name, params.Model, name); err != nil {
				t.Errorf("could not share model with user: %v", err)
				continue
			}
			shared, err := ModelSharedUsers(tx, user.Name, params.Model)
			if err != nil {
				t.Errorf("could not get shared models: %v", err)
				continue
			}
			if exp, got := (i + 1), len(shared); exp != got {
				t.Errorf("expected %d shared users got %d", exp, got)
			}
		}

		for i, name := range usernames {
			if err := StopSharing(tx, user.Name, params.Model, name); err != nil {
				t.Errorf("could not share model with user: %v", err)
				continue
			}
			shared, err := ModelSharedUsers(tx, user.Name, params.Model)
			if err != nil {
				t.Errorf("could not get shared models: %v", err)
				continue
			}
			if exp, got := len(usernames)-(i+1), len(shared); exp != got {
				t.Errorf("expected %d shared users got %d", exp, got)
			}
		}
	})
}

func TestSharedModels(t *testing.T) {

	RunDBTest(t, func(tx *sql.Tx) {
		user1 := "bigdatabob"
		user2 := "hadooopheather"

		users := map[string]*User{}

		for _, name := range []string{user1, user2} {
			user, err := NewUser(tx, name, "pass", name+"@yhathq.com", true)
			if err != nil {
				t.Errorf("could not add user: %v", err)
				return
			}
			users[name] = user
		}

		models := []string{"hellopy", "hellopy2", "hellor"}
		for i, model := range models {
			for j := 0; j < 3; j++ {
				params := NewVersionParams{
					UserId:         users[user1].Id,
					Model:          model,
					Lang:           LangPython2,
					BundleFilename: "bundle.json",
				}
				_, err := NewModelVersion(tx, &params)
				if err != nil {
					t.Errorf("could not create model version: %v", err)
					return
				}
			}
			if err := StartSharing(tx, user1, model, user2); err != nil {
				t.Errorf("could not share model with user: %v", err)
				return
			}
			sharedModels, err := SharedModels(tx, user2)
			if err != nil {
				t.Errorf("could not get shared models: %v", err)
				continue
			}
			if exp, got := i+1, len(sharedModels); exp != got {
				t.Errorf("expected %d shared models got %d", exp, got)
			}
		}

	})
}

func TestDeleteUser(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		user1 := "bigdatabob"
		user2 := "hadooopheather"

		users := map[string]*User{}

		for _, name := range []string{user1, user2} {
			user, err := NewUser(tx, name, "pass", name+"@yhathq.com", true)
			if err != nil {
				t.Errorf("could not add user: %v", err)
				return
			}
			users[name] = user
		}

		nVersions := 3

		models := []string{"hellopy", "hellopy2", "hellor"}
		for _, user := range users {
			for _, model := range models {
				for j := 0; j < nVersions; j++ {
					params := NewVersionParams{
						UserId:         user.Id,
						Model:          model,
						Lang:           LangPython2,
						BundleFilename: "bundle.json",
					}
					_, err := NewModelVersion(tx, &params)
					if err != nil {
						t.Errorf("could not create model version: %v", err)
						return
					}
				}
			}
		}

		for _, user := range users {
			bundles, err := DeleteUser(tx, user.Name)
			if err != nil {
				t.Error(err)
			} else {
				exp := nVersions * len(models)
				got := len(bundles)
				if exp != got {
					t.Errorf("expected %d bundles when deleting use %s, got %d", exp, user, got)
				}

			}
		}

	})
}
