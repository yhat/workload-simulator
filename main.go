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
	reportc := make(chan *app.Report)
	defer close(killc)
	defer close(reportc)

	a.Killc = killc
	a.Reportc = reportc

	// Start a StatMonitor goroutine that maintains a map of models to stats.
	fmt.Println("starting stats mointor")
	stats := app.StatsMonitor(reportc, 100*time.Millisecond)
	a.Statc = stats

	log.Printf("serving http on port: %d\n", cfg.Web.HttpPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Web.HttpPort), a)
	if err != nil {
		log.Println(err)
	}

}
