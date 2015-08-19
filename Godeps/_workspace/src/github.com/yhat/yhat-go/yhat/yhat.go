package yhat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

// the struct is pretty simple
type Yhat struct {
	username, apikey, env string
}

// Construct an instance of a Yhat struct
func New(username, apikey, env string) (*Yhat, error) {
	// Get the hostname without prefixed http and trailing slash
	startRe, _ := regexp.Compile("^http://")
	endRe, _ := regexp.Compile("/$")
	env = string(startRe.ReplaceAll([]byte(env), []byte("")))
	env = string(endRe.ReplaceAll([]byte(env), []byte("")))

	// Construct url for verification endpoint
	url := fmt.Sprintf("http://%s/verify?username=%s&apikey=%s",
		env, username, apikey)

	// Send the request
	req, _ := http.NewRequest("POST", url, nil)
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Could not connect to host: %s", env)
		return nil, errors.New(errMsg)
	}
	if res.StatusCode != 200 {
		errMsg := fmt.Sprintf("Bad status code (%d) from host: %s",
			res.StatusCode, env)
		return nil, errors.New(errMsg)
	}

	// Decode JSON response
	decoder := json.NewDecoder(res.Body)
	var resInterface interface{}
	err = decoder.Decode(&resInterface)
	if err != nil {
		errMsg := fmt.Sprintf("Invalid response from host: %s", env)
		return nil, errors.New(errMsg)
	}

	// Verify response is {"success":"true"}
	resJson := resInterface.(map[string]interface{})
	if val, ok := resJson["success"]; ok {
		if val == "true" {
			return &Yhat{username, apikey, env}, nil
		}
	}
	return nil, errors.New("Invalid username/apikey!")
}

// Make a prediction through the Yhat api
func (yhat *Yhat) Predict(modelname string, data interface{}) (
	map[string]interface{}, error) {
	// Construct url for prediction endpoint
	url := fmt.Sprintf("http://%s/%s/models/%s/",
		yhat.env, yhat.username, modelname)

	// JSON encode incoming data
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		errMsg := fmt.Sprintf("Could not JSON encode data: %s", data)
		return nil, errors.New(errMsg)
	}

	// Configure and send request
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(yhat.username, yhat.apikey)
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("Could not connect to endpoint: %s", url)
		return nil, errors.New(errMsg)
	}
	if res.StatusCode != 200 {
		errMsg := fmt.Sprintf("Bad status code (%d) from endpoint: %s",
			res.StatusCode, url)
		return nil, errors.New(errMsg)
	}

	// Decode response from server
	decoder := json.NewDecoder(res.Body)
	var resInterface map[string]interface{}
	err = decoder.Decode(&resInterface)
	if err != nil {
		return nil, errors.New("Invalid response from server")
	}
	return resInterface, nil
}
