package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/yhat/go-docker"

	_ "github.com/timob/go-mysql"
)

func TestMain(m *testing.M) {
	c, err := NewSQLContainer()
	if err != nil {
		log.Fatal(err)
	}
	// Assign to a global so we can share across tests.
	NewTestDB = c.NewDB
	returnCode := m.Run()
	c.Close()
	os.Exit(returnCode)
}

func RunDBTest(t *testing.T, test func(tx *sql.Tx)) {
	testDB, err := NewTestDB()
	if err != nil {
		t.Error(err)
		return
	}
	defer testDB.Close()

	tx, err := testDB.Begin()
	if err != nil {
		t.Errorf("could not create db: %v", err)
		return
	}
	defer tx.Rollback()
	test(tx)
}

type SQLContainer struct {
	cid     string
	rootStr string // string to connect as root
	connStr string

	// the current test's open connection
	// calling NewDB will close this
	conn *sql.DB
}

// NewSQLContainer starts a docker container running mysql. It is the callers
// responsiblity to call Close().
func NewSQLContainer() (c *SQLContainer, err error) {

	cid := os.Getenv("OPS_MYSQL_CID")
	if cid == "" {
		cli, err := docker.NewDefaultClient(3 * time.Second)
		if err != nil {
			return nil, fmt.Errorf("could not create client")
		}
		log.Println("creating mysql container")
		// Depricating to mysql:5.6 because I don't know how to CREATE USER IF NOT EXISTS
		// GRANT semantic is not supported in 5.7.
		cmd := exec.Command("docker", "run", "-d", "-e", "MYSQL_ROOT_PASSWORD=password",
			"-p", "3307:3306", "mysql:5.6")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("can't create container: %v, %s", err, out)
		}
		cid = string(bytes.TrimSpace(out))
		defer func() {
			if err != nil {
				rErr := cli.RemoveContainer(cid, true, false)
				if rErr != nil {
					log.Println("could not remove container: %v", err)
				}
			}
		}()
	}

	connStr := "mysql://scienceops:scienceops@0.0.0.0:3307/scienceops?client-multi-results&strict"
	rootStr := "mysql://root:password@0.0.0.0:3307/mysql?client-multi-results&strict"
	db, err := sql.Open("mysql", rootStr)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %v", err)
	}
	log.Println("waiting for mysql database to come up")
	opened := false
	for i := 0; i < 30; i++ {
		// It takes a couple seconds to start MySQL.
		if err := db.Ping(); err == nil {
			opened = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !opened {
		err = fmt.Errorf("can't ping database %s", rootStr)
		return
	}
	return &SQLContainer{cid: cid, connStr: connStr, rootStr: rootStr}, nil
}

func (c *SQLContainer) Close() error {
	cid := os.Getenv("OPS_MYSQL_CID")
	if cid == "" {
		cli, err := docker.NewDefaultClient(3 * time.Second)
		if err != nil {
			return err
		}
		return cli.RemoveContainer(c.cid, true, false)
	}
	return nil
}

func (c *SQLContainer) ConnStr() string { return c.connStr }

// NewDB reinitializes the mysql database and returns a connection to the db.
func (c *SQLContainer) NewDB() (*sql.DB, error) {
	// if a connection is open, close it
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	db, err := sql.Open("mysql", c.rootStr)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not begin tx: %v", err)
	}
	defer tx.Rollback()
	if err = DropTables(tx); err != nil {
		return nil, fmt.Errorf("could not drop tables: %v", err)
	}
	if err = InitTables(tx, "mysql"); err != nil {
		return nil, fmt.Errorf("could not drop tables: %v", err)
	}
	db.Close()
	conn, err := sql.Open("mysql", c.connStr)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return conn, nil
}
