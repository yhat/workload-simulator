package app

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/yhat/yhat-go/yhat"
)

type Settings struct {
	opsHost    string
	apiKey     string
	user       string
	maxDialVal int
	workers    int
}

type ModelInput struct {
	// model name
	name string

	// input data
	input map[string]string

	// queries per second
	qps int
}

type Workload struct {
	workload   map[string]*ModelInput
	settings   *Settings
	yhatClient *yhat.Yhat
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
	_, err := w.yhatClient.Predict(model.name, data)
	if err != nil {
		return fmt.Errorf("yhat prediction failed: %v", err)
	}
	return nil
}
