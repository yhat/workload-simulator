package sqlutil

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestReconnectingDB(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	must := func(err error, msg string) {
		if err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("%s: %v", msg, err)
		}
	}

	db, err := NewReconnectingDB("sqlite3", filepath.Join(tempDir, "test.db"))
	must(err, "create ReconnectingDB")

	db.ErrorLog = log.New(ioutil.Discard, "", 0)
	tx, err := db.Begin()
	must(err, "create transaction")

	_, err = tx.Exec(`CREATE TABLE Person (name text, age int);`)
	must(err, "create table")

	name, age := "eric", 12
	_, err = tx.Exec(`INSERT INTO Person (name, age) VALUES (?, ?);`, name, age)
	must(err, "insert into database")

	err = tx.Commit()
	must(err, "committing transaction")

	// Oh no! The database connection went down!
	db.dbConn.Close()

	tx, err = db.Begin()
	must(err, "create transaction after db connection closed")

	var rowName string
	var rowAge int
	err = tx.QueryRow("SELECT name, age FROM Person;").Scan(&rowName, &rowAge)
	must(err, "scanning row")

	if rowName != name {
		t.Errorf("expected name to be '%s' got '%s'", name, rowName)
	}
	if rowAge != age {
		t.Errorf("expected age to be '%d' got '%d'", age, rowAge)
	}
	err = tx.Rollback()
	must(err, "rollback")
}

// TestConcurrentDB ensures that a broken connection only attempts to
// reconnect once under concurrent requests.
func TestConcurrentDB(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	db, err := NewReconnectingDB("sqlite3", filepath.Join(tempDir, "test.db"))
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("creating ReconnectingDB: %v", err)
	}
	db.ErrorLog = log.New(ioutil.Discard, "", 0)
	timesReconnected := 0
	conn := db.connect
	db.connect = func() (*sql.DB, error) {
		timesReconnected++
		return conn()
	}
	start := make(chan bool)
	var ready sync.WaitGroup
	var done sync.WaitGroup
	n := 100
	ready.Add(n)
	done.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			ready.Done()
			<-start
			_, err := db.Begin()
			if err != nil {
				t.Errorf("could not connect to database: %v", err)
			}
			done.Done()
		}()
	}
	ready.Wait()
	// close the connection to the database
	db.dbConn.Close()
	// trigger all of the goroutines
	close(start)
	done.Wait()
	if timesReconnected != 1 {
		t.Errorf("expected one reconnect request, got %d", timesReconnected)
	}
}
