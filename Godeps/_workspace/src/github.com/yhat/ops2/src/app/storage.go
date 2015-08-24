package app

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/yhat/ops2/src/db"
	"github.com/yhat/ops2/src/mps"
)

// storage is an implementation of the supervisor.Storage interface
type storage struct {

	// Some db actions will cause deadlocks if they're called concurrently too
	// many times. See issue #65
	// This is a hack to solve this, though at some point we should look into
	// correcting the database queries.
	mu *sync.Mutex

	app *App
}

func (s *storage) Get(user, model string, version int) (*mps.DeployInfo, string, error) {
	tx, err := s.app.db.Begin()
	if err != nil {
		return nil, "", fmt.Errorf("database unavailabled: %v", err)
	}
	defer tx.Rollback()
	modelVersion, err := db.GetModelVersion(tx, user, model, version)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, "", errors.New("model version not found")
		}
		return nil, "", fmt.Errorf("could not query model version: %v", err)
	}

	baseImage, err := db.GetBaseImage(tx, modelVersion.Lang)
	if err != nil && err != sql.ErrNoRows {
		return nil, "", fmt.Errorf("could not query baseImage: %v", err)
	}

	// TODO: Conda channels, CRAN mirror and apt-get sources
	info := &mps.DeployInfo{
		Username:         user,
		Modelname:        model,
		Version:          version,
		Lang:             modelVersion.Lang,
		LanguagePackages: modelVersion.LangPackages,
		UbuntuPackages:   modelVersion.UbuntuPackages,
		BaseImage:        baseImage,
	}
	if modelVersion.BundleFilename == "" {
		return nil, "", fmt.Errorf("no bundle file provided")
	}
	bundlePath := filepath.Join(s.app.bundleDir, modelVersion.BundleFilename)
	log.Println("bundle path is", bundlePath)
	return info, bundlePath, nil
}

func (s *storage) GetLatest(user, model string) (version int, err error) {
	tx, err := s.app.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("database unavailabled: %v", err)
	}
	defer tx.Rollback()
	return db.GetLatestVersion(tx, user, model)
}

func (s *storage) SetBuildStatus(user, model, status string) error {
	tx, err := s.app.db.Begin()
	if err != nil {
		return fmt.Errorf("database unavailabled: %v", err)
	}
	defer tx.Rollback()
	s.app.logf("setting status of model %s:%s to %s", user, model, status)
	if err := db.SetBuildStatus(tx, user, model, status); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *storage) NewDeployment(user, model string, version int) (int64, []int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.app.db.Begin()
	if err != nil {
		return 0, nil, fmt.Errorf("database unavailabled: %v", err)
	}
	defer tx.Rollback()
	deployId, instIds, err := db.NewDeployment(tx, user, model, version, s.app.modelReplication)
	if err != nil {
		return 0, nil, err
	}
	if err = tx.Commit(); err != nil {
		return 0, nil, fmt.Errorf("database unavailabled: %v", err)
	}
	return deployId, instIds, err
}
