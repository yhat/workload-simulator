package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Workload struct {
	// remote ops server info
	opsHost string
	apiKey  string
	user    string

	// model request data
	nrequests  int
	modelId    string
	modelName  string
	modelInput map[string]interface{}
}

// Predict sends a POST request to an ops model.
func (w *Workload) Predict() error {
	data := w.modelInput

	// Make a prediciton.
	username := w.user
	modelname := w.modelName
	apikey := w.apiKey
	host := w.opsHost

	_, err := opsPredictHTTP(username, modelname, apikey, host, data)
	if err != nil {
		return fmt.Errorf("yhat prediction failed: %v", err)
	}
	return nil
}

func opsPredictHTTP(username, model, apikey, host string, input interface{}) (*http.Response, error) {
	url := host + "/" + username + "/models/" + model + "/"
	b, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("could not josn marshal input data: %v\n", err)
	}
	sinput := string(b)

	req, err := http.NewRequest("POST", url, strings.NewReader(sinput))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(username, apikey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ops prediction request failed: %v", err)
	}
	defer resp.Body.Close()
	return resp, nil
}

// Worker func that spawns a goroutine that does work and emits statistics to a stats channel
// every period duratio1vn.
func Worker(stats chan<- *Stat, kill <-chan int, dt time.Duration, w *Workload) {
	ticker := time.NewTicker(dt)
	predCount := 0
	go func() {
		for i := 0; i < w.nrequests; i++ {
			select {
			case <-ticker.C:
				// send stats and reset request counter
				stats <- &Stat{w.modelId, predCount, int(dt.Seconds())}
				predCount = 0
			case <-kill:
				fmt.Printf("im dying!!!")
				return
			default:
				// Do work and increment counter
				err := w.Predict()
				if err != nil {
					fmt.Printf("Prediction error: %v\n", err)
					return
				}
				predCount += 1
			}
		}
	}()
	return
}

// Stat represents request statistics
type Stat struct {
	modelId string
	nreq    int
	dt      int
}

func StatsMonitor(report chan<- string, kill <-chan int, dt time.Duration) chan<- *Stat {
	stats := make(chan *Stat)
	requestStats := make(map[string]int)

	ticker := time.NewTicker(dt)
	go func() {
		for {
			select {
			case <-ticker.C:
				// send report of stats.
				r, err := json.Marshal(requestStats)
				if err != nil {
					fmt.Printf("error marshalling json stats: %v", err)
					return
				}
				b := bytes.NewBuffer(r)
				report <- b.String()
			case s := <-stats:
				idt := dt.Seconds()
				reqs := float64(s.nreq) / idt
				requestStats[s.modelId] = int(reqs)
			case <-kill:
				fmt.Println("[StatsMonitor] got KILLSIG exiting.")
				return
			}

		}
	}()
	return stats
}
