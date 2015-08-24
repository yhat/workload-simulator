package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/yhat/workload-simulator/app"
)

var c = flag.String("config", "", "file path for app configuration yaml")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	cfgPath := *c
	cfg, err := app.ReadConfig(cfgPath)
	if err != nil {
		log.Println("error parsing app configuration yaml")
		os.Exit(1)
	}

	app, err := app.New(cfg)
	if err != nil {
		log.Println(err)
	}

	log.Printf("serving http on port: %d\n", cfg.Web.HttpPort)

	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Web.HttpPort), app)
	if err != nil {
		log.Println(err)
	}
}
