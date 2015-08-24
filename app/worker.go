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

// Predict sends a POST request to an ops model endpoint.
func (w *Workload) Predict() error {
	data := w.modelInput

	// Make a prediciton to this remote host.
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

// Worker func that spawns a goroutine that does work and emits statistics to a stats
// channel every period duration seconds.
func Worker(stats chan<- *Stat, kill <-chan int, dt time.Duration, id int, w *Workload) {
	// TODO add predicitons made vs pred completed.
	ticker := time.NewTicker(dt)
	go func(n int) {
		predCount := 0
		predSent := 0
		for i := 0; i < n; i++ {
			predSent += 1
			fmt.Printf("id: %d, cnt:%+v\n", id, predCount)
			select {
			case <-ticker.C:
				// send stats and reset request counter
				// TODO: add predSent to Stat struct
				s := &Stat{w.modelId, predCount, dt.Seconds()}
				stats <- s
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
		// exit goroutine when work is done.
		return
	}(w.nrequests)
	return
}

// Stat represents request statistics
type Stat struct {
	modelId string
	nreq    int
	dt      float64
}

func StatsMonitor(report chan<- string, dt time.Duration) chan *Stat {
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
					fmt.Printf("error marshalling json stats: %v\n", err)
					return
				}
				b := bytes.NewBuffer(r)
				report <- b.String()
			case s := <-stats:
				reqs := float64(s.nreq) / s.dt
				requestStats[s.modelId] = int(reqs)
			}

		}
	}()
	return stats
}
