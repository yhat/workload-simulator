package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	OpsHost    string `json:"ops_host"`
	ApiKey     string `json:"ops_apikey"`
	User       string `json:"ops_user"`
	MaxDialVal int    `json:"dial_max_value"`
	Workers    string `json:"workers"`
}

type ModelInput struct {
	// model name
	name string

	// input data
	input map[string]interface{}

	// queries per second
	qps int
}

type Workload struct {
	workload map[string]*ModelInput
	settings *Settings
}

// Predict sends a POST request to an ops model. It chooses a random model from the
// workload map. The way that the ui is set up, will allow you to run n requests in
// order. We want to change this eventually, but for now each prediction will choose
// a random model from the workload map and send a request.
func (w *Workload) Predict() error {
	n := len(w.workload)
	if n == 0 {
		return fmt.Errorf("no work to be done")
	}

	// Choose a model from the workload at random.
	model := w.workload[strconv.Itoa(rand.Intn(n))]
	data := model.input

	// Make a prediciton.
	username := w.settings.User
	modelname := model.name
	apikey := w.settings.ApiKey
	host := w.settings.OpsHost

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

type Work struct {
	modelId  string
	workload *Workload
	reqTotal int
}

// Worker func that spawns a goroutine that does work and emits statistics to a stats channel
// every period duratio1vn.
func Worker(stats chan<- *Stat, kill <-chan int, dt time.Duration, w *Work) {
	ticker := time.NewTicker(dt)
	predCount := 0
	go func() {
		for i := 0; i < w.reqTotal; i++ {
			select {
			case <-ticker.C:
				// send stats and reset request counter
				stats <- &Stat{w.modelId, predCount, int(dt)}
				predCount = 0
			case <-kill:
				fmt.Printf("im dying!!!")
				return
			default:
				// Do work and increment counter
				err := w.workload.Predict()
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
