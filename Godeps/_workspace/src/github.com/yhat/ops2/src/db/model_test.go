package db

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/yhat/ops2/src/mps"
)

func TestInsertModelVersion(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		user, err := NewUser(tx, "bigdatabob", "foo", "", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		langPackages := []mps.Package{
			{"scikit-learn", "1.3.4"},
		}

		ubuntuPackages := []mps.Package{
			{"wget", "12321"},
			{"tree", ""},
		}

		params := NewVersionParams{
			UserId:         user.Id,
			Model:          "hellopy",
			Lang:           LangPython2,
			LangPackages:   langPackages,
			UbuntuPackages: ubuntuPackages,
			BundleFilename: "bundle.json",
		}
		for i := 0; i < 3; i++ {

			expVer := i + 1

			version, err := NewModelVersion(tx, &params)
			if err != nil {
				t.Errorf("could not create model version: %v", err)
				return
			}
			if version != expVer {
				t.Errorf("expected version to be %d, got %d", expVer, version)
			}
			mv, err := GetModelVersion(tx, user.Name, params.Model, version)
			if err != nil {
				t.Errorf("could not get model version %d: %v", version, err)
				return
			}
			if mv.Lang != LangPython2 {
				t.Errorf("expected language to be '%s' got '%s'", LangPython2, mv.Lang)
			}
			v, err := GetLatestVersion(tx, user.Name, params.Model)
			if err != nil {
				t.Errorf("could not get model version: %v", err)
			} else if v != expVer {
				t.Errorf("expected version to be %d, got %d", expVer, v)
			}

			err = SetBuildStatus(tx, user.Name, params.Model, "building")
			if err != nil {
				t.Errorf("could not set build status: %v", err)
			}
			err = SetBuildStatus(tx, user.Name, params.Model, "online")
			if err != nil {
				t.Errorf("could not set build status: %v", err)
			}
		}
	})
}

func TestIssue53(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		users := map[string]*User{
			"bigdatabob":           nil,
			"hadoopheather":        nil,
			"petabytepeter":        nil,
			"mongodbmatthew":       nil,
			"visualizationvalarie": nil,
		}
		for name := range users {
			user, err := NewUser(tx, name, "foo", name+"@yhathq.com", true)
			if err != nil {
				t.Errorf("could not create user: %v", err)
				return
			}
			users[name] = user
		}

		for _, user := range users {
			langPackages := []mps.Package{
				{"scikit-learn", "1.3.4"},
			}

			ubuntuPackages := []mps.Package{
				{"wget", "12321"},
				{"tree", ""},
			}

			params := NewVersionParams{
				UserId:         user.Id,
				Model:          "hellopy",
				Lang:           LangPython2,
				LangPackages:   langPackages,
				UbuntuPackages: ubuntuPackages,
				BundleFilename: "bundle.json",
			}
			for i := 0; i < 3; i++ {

				expVer := i + 1

				version, err := NewModelVersion(tx, &params)
				if err != nil {
					t.Errorf("could not create model for user %s user id %d: %v", user.Name, user.Id, err)
					return
				}
				if version != expVer {
					t.Errorf("expected version to be %d, got %d", expVer, version)
				}
			}
		}
	})
}
func TestGetModel(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		_, err := GetModel(tx, "bigdatabob", "foo")
		if !IsNotFound(err) {
			t.Errorf("expected IsNotFound error got %v", err)
		}
		user, err := NewUser(tx, "bigdatabob", "foo", "", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}
		langPackages := []mps.Package{
			{"scikit-learn", "1.3.4"},
		}

		ubuntuPackages := []mps.Package{
			{"wget", "12321"},
			{"tree", ""},
		}

		params := NewVersionParams{
			UserId:         user.Id,
			Model:          "hellopy",
			Lang:           LangPython2,
			LangPackages:   langPackages,
			UbuntuPackages: ubuntuPackages,
			BundleFilename: "bundle.json",
		}
		last := time.Now().UTC()

		seenIds := map[int64]struct{}{}

		for i := 0; i < 3; i++ {
			version, err := NewModelVersion(tx, &params)
			if err != nil {
				t.Errorf("could not create model version: %v", err)
				return
			}
			id, _, err := NewDeployment(tx, user.Name, params.Model, version, 3)
			if err != nil {
				t.Errorf("could not get deployment id: %v", err)
			} else {
				if _, ok := seenIds[id]; ok {
					t.Errorf("already seen this id: %d", id)
				}
				seenIds[id] = struct{}{}
			}
			m, err := GetModel(tx, user.Name, params.Model)
			if err != nil {
				t.Errorf("could not get model data: %v", err)
				return
			}
			if m.LastDeployment != id {
				t.Errorf("expected last deployment to be %d, got %d", id, m.LastDeployment)
			}
			if m.NumVersions != (i + 1) {
				t.Errorf("expected %s versions, got %d", i+1, m.NumVersions)
			}
			if m.ActiveVersion != (i + 1) {
				t.Errorf("expected active version to be %s, got %d", i+1, m.ActiveVersion)
			}
			last = last.Truncate(time.Second) // sql is only accurate to the second
			if m.LastUpdated.Before(last) {
				t.Errorf("last updated %s for version %d was before last update %s",
					m.LastUpdated, i+1, last)
			}
			last = m.LastUpdated
		}
	})
}

func TestGetModels(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		bob, err := NewUser(tx, "bigdatabob", "foo", "bob@foobar.com", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		tess, err := NewUser(tx, "terabytetess", "foo", "tess@foobar.com", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		for _, name := range []string{"hellopy", "goodbyepy", "foo"} {
			for _, user := range []*User{tess, bob} {
				params := NewVersionParams{
					UserId:         user.Id,
					Model:          name + user.Name,
					Lang:           LangPython2,
					BundleFilename: "bundle.json",
				}
				for i := 0; i < 3; i++ {
					if _, err := NewModelVersion(tx, &params); err != nil {
						t.Errorf("could not create model version: %v", err)
						return
					}
				}
			}
		}

		models, err := UserModels(tx, "bigdatabob")
		if err != nil {
			t.Errorf("could not get user models: %v", err)
			return
		}
		if n := len(models); n != 3 {
			t.Errorf("expected 3 models, got %d", n)
			return
		}
	})
}

func TestDeleteModel(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		tempfile, err := ioutil.TempFile("", "")
		if err != nil {
			t.Error(err)
			return
		}
		defer tempfile.Close()
		bundle := tempfile.Name()
		defer os.Remove(bundle)
		user, err := NewUser(tx, "bigdatabob", "foo", "", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		params := NewVersionParams{
			UserId:         user.Id,
			Model:          "hello.py",
			Lang:           LangPython2,
			BundleFilename: bundle,
		}
		if _, err := NewModelVersion(tx, &params); err != nil {
			t.Errorf("could not create model version: %v", err)
			return
		}

		bundles, err := DeleteModel(tx, user.Name, params.Model)
		if err != nil {
			t.Errorf("could not delete model: %v", err)
		}
		if got := len(bundles); got != 1 {
			t.Errorf("expected 1 bundle got %d", got)
			return
		}
		got := bundles[0]
		if got != bundle {
			t.Errorf("expected bundle name to be %s got %s", bundle, got)
		}
	})
}

func TestActiveVersion(t *testing.T) {

	RunDBTest(t, func(tx *sql.Tx) {

		user, err := NewUser(tx, "bigdatabob", "foo", "", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		params := NewVersionParams{
			UserId:         user.Id,
			Model:          "hello.py",
			Lang:           LangPython2,
			BundleFilename: "/bundle.json",
		}
		nVersions := 3

		versions := []int{}

		for i := 0; i < nVersions; i++ {
			version, err := NewModelVersion(tx, &params)
			if err != nil {
				t.Errorf("could not create model version: %v", err)
				return
			}
			versions = append(versions, version)

			_, _, err = NewDeployment(tx, user.Name, params.Model, version, 3)
			if err != nil {
				t.Errorf("could not get deployment id: %v", err)
				return
			}
			m, err := GetModel(tx, user.Name, params.Model)
			if err != nil {
				t.Errorf("could not get model: %v", err)
				return
			}
			if m.ActiveVersion != version {
				t.Errorf("expected active version to be %d got %d", version, m.ActiveVersion)
			}
		}

		for _, version := range versions {

			_, _, err = NewDeployment(tx, user.Name, params.Model, version, 3)
			if err != nil {
				t.Errorf("could not get deployment id: %v", err)
				return
			}
			m, err := GetModel(tx, user.Name, params.Model)
			if err != nil {
				t.Errorf("could not get model: %v", err)
				return
			}
			if m.ActiveVersion != version {
				t.Errorf("expected active version to be %d got %d", version, m.ActiveVersion)
			}
		}
	})
}

func TestModelExample(t *testing.T) {
	RunDBTest(t, func(tx *sql.Tx) {
		tempfile, err := ioutil.TempFile("", "")
		if err != nil {
			t.Error(err)
			return
		}
		defer tempfile.Close()
		bundle := tempfile.Name()
		defer os.Remove(bundle)
		user, err := NewUser(tx, "bigdatabob", "foo", "", true)
		if err != nil {
			t.Errorf("could not create user: %v", err)
			return
		}

		params := NewVersionParams{
			UserId:         user.Id,
			Model:          "hello.py",
			Lang:           LangPython2,
			BundleFilename: bundle,
		}
		if _, err := NewModelVersion(tx, &params); err != nil {
			t.Errorf("could not create model version: %v", err)
			return
		}

		if _, err := ModelExample(tx, user.Name, params.Model); err != nil {
			t.Error("failed to get example:", err)
		}

		example := `{"hello":"world"}`
		if err := SetModelExample(tx, user.Name, params.Model, example); err != nil {
			t.Error("could not set model example:", err)
		}

		got, err := ModelExample(tx, user.Name, params.Model)
		if err != nil {
			t.Error("failed to get example:", err)
		}
		if example != got {
			t.Errorf("expected model example '%s' got '%s'", example, got)
		}
	})
}
