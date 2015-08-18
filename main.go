package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/yhat/workload-simulator/app"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	config := &app.AppConfig{
		Host:      "localhost",
		Port:      8080,
		MaxDial:   1500,
		PublicDir: "/home/ec2/gopath/src/github.com/yhat/workload-simulator/app/public",
		ViewsDir:  "/home/ec2/gopath/src/github.com/yhat/workload-simulator/app/views",
		ReportDir: "/home/ec2",
	}
	app, err := app.New(config)
	if err != nil {
		log.Println(err)
	}

	log.Printf("serving http on port: %d\n", config.Port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), app)
	if err != nil {
		log.Println(err)
	}
}
