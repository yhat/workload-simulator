package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Pallinder/go-randomdata"
	_ "github.com/timob/go-mysql"
	"github.com/yhat/ops2/src/db"
)

const (
	mysqlImage = "mysql:5.6"
	name       = "scienceops-mysql-dev"
)

// - [x] create N users
// - [x] create N apikeys & read-only apikeys
// - [x] create M models for each user
//     - [x] create J versions for model
//     - [x] create model status for each model
//     - [ ] associate model w/ MPS (?)
//     - [x] give (some) models JSON/HTML examples
//     - [x] share model with K users
func seedFunc(conn *sql.DB) error {
	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("cannot begin transaction %v", err)
	}
	defer tx.Rollback()

	log.Println("truncating tables in db")
	tx.Exec(`use scienceops;`)
	tx.Exec(`SET FOREIGN_KEY_CHECKS=0;`)
	rows, err := tx.Query(`show tables;`)
	if err != nil {
		return fmt.Errorf("could not truncate database: ", err)
	}
	var tables []string

	defer rows.Close()
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tables = append(tables, t)
	}
	for _, t := range tables {
		q := fmt.Sprintf("TRUNCATE TABLE %s;", t)
		if _, err := tx.Exec(q); err != nil {
			fmt.Println(err)
			return fmt.Errorf("Could not truncate table %s: %v", t, err)
		}
	}
	tx.Exec(`SET FOREIGN_KEY_CHECKS=1;`)

	log.Println("seeding db")

	nUsers := 3
	nModels := 10
	nVersions := 15

	os.MkdirAll("/tmp/bundles/", 0777)

	hashedPass := "sha1$V50iYdII$1$e4e846ce24046a7677553d3a2c68d14813d842af"
	for _, user := range []string{"eric", "ryan", "greg", "sush", "colin", "brandon", "austin", "charlie"} {
		_, err := db.NewUser(tx, user, hashedPass, user+"@yhathq.com", true)
		if err != nil {
			return fmt.Errorf("could not create user: %v", err)
		}
	}
	for _, user := range []string{"bigdatabob"} {
		_, err := db.NewUser(tx, user, hashedPass, user+"@yhathq.com", false)
		if err != nil {
			return fmt.Errorf("could not create user: %v", err)
		}
	}

	for i := 0; i < nUsers; i++ {
		username := fmt.Sprintf("user-%d", i)
		email := fmt.Sprintf("%s@yhathq.com", username)
		user, err := db.NewUser(tx, username, hashedPass, email, true)
		if err != nil {
			return fmt.Errorf("could not create user: %v", err)
		}
		log.Printf("Created user %s", user.Name)

		for j := 0; j < nModels; j++ {
			name := randomdata.SillyName()

			params := &db.NewVersionParams{
				UserId:         user.Id,
				Model:          name,
				Lang:           db.LangPython2,
				SourceCode:     "print HI!",
				BundleFilename: "/foobar/bundle.json",
			}

			for v := 0; v < nVersions; v++ {
				if _, err := db.NewModelVersion(tx, params); err != nil {
					return fmt.Errorf("could not create version: %v", err)
				}
			}
			model, err := db.GetModel(tx, username, name)
			if err != nil {
				return fmt.Errorf("could not get model %s/%s: %v", username, name, err)
			}
			err = db.SetModelStatus(tx, model.Id, "online")
			if err != nil {
				fmt.Println(err)
				return fmt.Errorf("could not insert model status: %v", err)
			}

		}

	}
	log.Printf("added %d users to db\n", nUsers)
	return tx.Commit()
}

func createDatabase(sqlDir string, useExisting bool) (*sql.DB, error) {

	// run the docker container
	cmd := exec.Command("docker", "run", "-d", "--name="+name, "-e", "MYSQL_ROOT_PASSWORD=password",
		"-p", "3308:3306", "mysql:5.6")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("can't create container: %v %s", err, out)
	}
	defer func(cid string) {
		if err != nil {
			out, rErr := exec.Command("docker", "rm", "-f", cid).CombinedOutput()
			if rErr != nil {
				log.Println("could not remove container %s: %s", cid, out)
			}
		}
	}(string(bytes.TrimSpace(out)))

	rootStr := "mysql://root:password@0.0.0.0:3308/mysql?client-multi-results&strict"

	// wait for the database to become available
	conn, err := sql.Open("mysql", rootStr)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %v", err)
	}
	log.Println("waiting for mysql database to come up")
	opened := false
	for i := 0; i < 30; i++ {
		// It takes a couple seconds to start MySQL.
		if err = conn.Ping(); err == nil {
			opened = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !opened {
		return nil, fmt.Errorf("can't ping database %s, %v", rootStr, err)
	}

	// the database is up, time to intialize it
	log.Println("database is up running init scripts")
	tx, err := conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("cannot begin transaction")
	}
	defer tx.Rollback()

	if err = db.InitTables(tx, sqlDir); err != nil {
		return nil, fmt.Errorf("could not initalize database: %v", err)
	}
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("could not commit initialized database: %v", err)
	}
	return conn, nil
}

var (
	reseed = flag.Bool("reseed", false, "truncates all tables and reseeds database")
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: dbinit [sqldir] \n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// check if the container already exists
	if err := exec.Command("docker", "inspect", mysqlImage).Run(); err != nil {
		log.Fatalf("command requires '%s' please pull it from docker hub", mysqlImage)
	}
	if err := exec.Command("docker", "inspect", name).Run(); err == nil {
		log.Println("database already running")
		rootStr := "mysql://root:password@0.0.0.0:3308/mysql?client-multi-results&strict"
		// wait for the database to become available
		conn, err := sql.Open("mysql", rootStr)
		if err != nil {
			return
		}

		if *reseed == true {
			if err := seedFunc(conn); err != nil {
				log.Fatalf("error seeding db: %v\n", err)
			}
		}
	} else {
		if len(flag.Args()) < 1 {
			flag.Usage()
		}

		// start a database
		conn, err := createDatabase(flag.Args()[0], true)
		if err != nil {
			log.Fatal(err)
		}

		if err := seedFunc(conn); err != nil {
			log.Fatal(err)
		}
	}
	log.Println("success")
}
