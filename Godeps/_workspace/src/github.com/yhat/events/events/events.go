package events

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/lib/pq"
)

var Endpoint = "https://et1-phone-home-metrics.herokuapp.com/v2"

type Deployment struct {

	// UNIX timestamps
	StartTime int64
	EndTime   int64

	Username  string
	ModelName string

	// 'python' or 'r'
	ModelLang string

	ModelDeps string

	ModelVer  int64
	ModelSize int64

	ClientIP string
	Service  string

	Error string
}

// connect to a postgres database
func conn(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("opening connection: %v", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging db: %v", err)
	}
	return db, nil
}

type EventLogger struct {
	connStr string
}

func NewEventLogger(connStr string) (*EventLogger, error) {
	db, err := conn(connStr)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %v", err)
	}

	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %v", err)
	}
	defer tx.Rollback()
	q := `CREATE TABLE IF NOT EXISTS Metrics
	(timestmp int, start_time int, end_time int, username varchar(255),
	 modelname varchar(255), modellang varchar(255), modeldeps varchar(255),
	 modelver varchar(255), modelsize int);`
	if _, err := tx.Exec(q); err != nil {
		return nil, fmt.Errorf("create table: %v", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %v", err)
	}

	tx2, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %v", err)
	}
	defer tx2.Rollback()

	logger := &EventLogger{
		connStr: connStr,
	}

	update := `ALTER TABLE Metrics
	ADD client_ip varchar(255),
	ADD service varchar(255),
	ADD error TEXT;`
	if _, err := tx2.Exec(update); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42701" {
			return logger, nil
		}
		return nil, fmt.Errorf("alert table: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %v", err)
	}
	return logger, nil
}

func (logger *EventLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		io.WriteString(w, "I'm up!")
	case "POST":
		switch r.URL.Path {
		case "/metrics":
			logger.handleV1(w, r)
		case "/v2":
			logger.handleV2(w, r)
		default:
			http.NotFound(w, r)
		}
	default:
		http.Error(w, "I only respond to GET and POSTs", http.StatusNotImplemented)
	}
}

func (logger *EventLogger) handleV1(w http.ResponseWriter, r *http.Request) {
	em := &encodableMetric{}
	if err := json.NewDecoder(r.Body).Decode(em); err != nil {
		http.Error(w, "could not decode body", http.StatusBadRequest)
		return
	}
	d, err := transform(em)
	if err != nil {
		http.Error(w, "could not decode body", http.StatusBadRequest)
		return
	}
	if err := logger.Save(d); err != nil {
		log.Printf("could not save data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (logger *EventLogger) handleV2(w http.ResponseWriter, r *http.Request) {
	d := &Deployment{}
	if err := json.NewDecoder(r.Body).Decode(d); err != nil {
		http.Error(w, "could not decode body", http.StatusBadRequest)
		return
	}
	if err := logger.Save(d); err != nil {
		log.Printf("could not save data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (logger *EventLogger) Save(m *Deployment) error {

	checks := []struct {
		ok  bool
		msg string
	}{
		{m.Username != "", "username cannot be empty"},
		{m.ModelName != "", "modelname cannot be empty"},
		{m.ModelVer != 0, "model version cannot be zero"},
		{m.ModelLang != "", "model lang cannot be empty"},
	}
	for _, check := range checks {
		if !check.ok {
			return errors.New(check.msg)
		}
	}
	db, err := conn(logger.connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	q := `INSERT INTO metrics
        (timestmp, start_time, end_time, username, modelname, modellang, 
		 modeldeps, modelver, modelsize, client_ip, service, error)
        VALUES
        ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err = db.Exec(q, time.Now().Unix(), m.StartTime, m.EndTime, m.Username,
		m.ModelName, m.ModelLang, m.ModelDeps, m.ModelVer, m.ModelSize,
		m.ClientIP, m.Service, m.Error)
	return err
}
