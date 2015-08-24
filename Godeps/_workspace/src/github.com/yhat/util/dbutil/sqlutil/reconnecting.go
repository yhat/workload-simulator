package sqlutil

import (
	"database/sql"
	"errors"
	"log"
	"sync"
)

// ReconnectingDB is an SQL database connection which attempts to reconnect to
// the database in the case of errors.
type ReconnectingDB struct {
	// ReconnectingDB will print to ErrorLog when errors occur.
	// If ErrorLog is nil, it will use the default logger from the log package.
	ErrorLog *log.Logger

	// mu locks against multiple attempts to reconnect
	mu *sync.Mutex

	// The function to use to reconnect
	connect func() (*sql.DB, error)

	// underlying connection
	dbConn *sql.DB
}

// NewReconnectingDB opens a new database connection using the standard
// library's sql.Open function. It then pings the database to ensure a
// connection has been made.
func NewReconnectingDB(driverName, dataSourceName string) (*ReconnectingDB, error) {
	connect := func() (*sql.DB, error) {
		return sql.Open(driverName, dataSourceName)
	}
	dbConn, err := connect()
	if err != nil {
		return nil, err
	}
	if err = dbConn.Ping(); err != nil {
		return nil, err
	}
	return &ReconnectingDB{
		connect: connect,
		mu:      new(sync.Mutex),
		dbConn:  dbConn,
	}, nil
}

func (db *ReconnectingDB) reconnect() error {
	if db.dbConn != nil {
		db.dbConn.Close()
	}
	db.dbConn = nil
	conn, err := db.connect()
	if err != nil {
		db.logf("connection failed: %v", err)
		return errors.New("can't connect to database")
	}
	if err = conn.Ping(); err != nil {
		db.logf("database unreachable: %v", err)
		return errors.New("can't connect to database")
	}
	db.dbConn = conn
	return nil
}

func (db *ReconnectingDB) Close() error {
	if db.dbConn == nil {
		return errors.New("no db connection active")
	}
	err := db.dbConn.Close()
	db.dbConn = nil
	return err
}

// Begin begins a sql transaction. If the database connection fails,
// the ReconnectingDB will attempt to reconnect once and try again.
func (db *ReconnectingDB) Begin() (*sql.Tx, error) {
	// must block to ensure we don't try to reconnect concurrently
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.dbConn == nil {
		if err := db.reconnect(); err != nil {
			return nil, err
		}
	}
	tx, err := db.dbConn.Begin()
	if err == nil {
		return tx, err
	}
	db.logf("error starting transaction %v. attempting reconnect", err)
	if err = db.reconnect(); err != nil {
		return nil, err
	}
	return db.dbConn.Begin()
}

func (db *ReconnectingDB) logf(format string, a ...interface{}) {
	if db.ErrorLog == nil {
		log.Printf(format, a...)
	} else {
		db.ErrorLog.Printf(format, a...)
	}
}
