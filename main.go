package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

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

	a, err := app.New(cfg)
	if err != nil {
		log.Println(err)
	}

	// Init communication channels for StatsMonitor and Workers
	killc := make(chan int)
	reportc := make(chan string)

	a.Killc = killc
	a.Reportc = reportc

	// init statsMonitor. Todo: Move this into app constructor.
	stats := app.StatsMonitor(reportc, killc, 100*time.Millisecond)
	a.Statc = stats

	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Web.HttpPort), a)
	if err != nil {
		log.Println(err)
	}

	log.Printf("serving http on port: %d\n", cfg.Web.HttpPort)
}
