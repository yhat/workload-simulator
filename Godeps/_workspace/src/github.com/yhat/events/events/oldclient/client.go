package oldclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// ModelInfo represents a model that is deployable to ScienceOps.
//
// ModelInfo can be used to record metrics related to model
// deployments to a SQL database. Metrics written to the SQL database
// include the fields defined in ModelInfo and additional metadata regarding
// a username and deployment start and end time.
type ModelInfo struct {
	// ModelName specifies a descritpitve model name
	// for the deployment. See the yhat-client for more
	// info on naming of models https://github.com/yhat/yhat-client.
	ModelName string
	// ModelLang is the programming language used to
	// implement the model. Python and R are currently
	// supported.
	ModelLang string
	// ModelDeps specifies a comma delimited list of
	// package and version names for the model dependencies
	// associated with this production model
	// (i.e., "fuzzywuzzy==1.0.0,sklearn==1.4.0").
	ModelDeps string
	// ModelVer is the deployment version for a production model.
	ModelVer int64
	// ModelSize is the memory size of the model in MB.
	ModelSize int64
}

// SendDeploy creates a new metrics struct, appends metrics about the model deployment,
// and sends a POST request to the metrics server.
func (mod *ModelInfo) SendDeploy(urlStr, username string, start, end time.Time) error {
	// Add data to metrics struct
	m := NewMetric()
	m.Add("StartTime", strconv.FormatInt(start.Unix(), 10))
	m.Add("EndTime", strconv.FormatInt(end.Unix(), 10))
	m.Add("Username", username)
	m.Add("ModelName", mod.ModelName)
	m.Add("ModelLang", mod.ModelLang)
	m.Add("ModelDeps", mod.ModelDeps)
	m.Add("ModelVer", strconv.FormatInt(mod.ModelVer, 10))
	m.Add("ModelSize", strconv.FormatInt(mod.ModelSize, 10))

	// Encode metrics into JSON
	body := bytes.NewBuffer([]byte{})
	err := Encode(body, m)
	if err != nil {
		return fmt.Errorf("failed to encode metrics: %v", err)
	}
	// Send request to SQL metrics server.
	req, err := http.NewRequest("POST", urlStr, body)
	if err != nil {
		return fmt.Errorf("failed to format request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send deployment: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", resp.Status, respBody)
	}
	return nil
}

// A Metric contains a timestamp and a slice of KeyVals.
//
// The caller can send a set of arbitrary KeyVals in a JSON
// body of a request. The Decode func will iterate through
// the KeyVals in the JSON body and append each arbitray pair
// to the Data KeyVal slice.
type Metric struct {
	// A timestamp that indicates when the metric
	// event was recorded.
	Timestamp time.Time
	// A slice of KeyVals used to send arbitrary
	// key-value pairs od data.
	Data []KeyVal
}

// KeyVal represents a key-value pair of data.
type KeyVal struct {
	Key, Val string
}

// NewMetric allocates a new Metric.
func NewMetric() *Metric {
	return &Metric{time.Now(), make([]KeyVal, 0)}
}

// Add adds a KeyVal to a Metric's Data slice.
func (m *Metric) Add(key, value string) {
	d := append(m.Data, KeyVal{key, value})
	m.Data = d
}

type encodableMetric struct {
	TimestampUnix int64
	Data          []KeyVal
}

// Encode returns encoded JSON data from a Metrics struct.
// A timestamp is added to the Metrics struct at call time.
func Encode(w io.Writer, m *Metric) error {
	em := encodableMetric{m.Timestamp.Unix(), m.Data}
	return json.NewEncoder(w).Encode(&em)
}

// Decode reads encoded metrics data from an io.Reader and decodes data
// and returns a pointer to a Metrics struct.
func Decode(r io.Reader) (*Metric, error) {
	em := encodableMetric{}
	err := json.NewDecoder(r).Decode(&em)
	if err != nil {
		return nil, fmt.Errorf("metrics decode failed:", err)
	}
	return &Metric{time.Unix(em.TimestampUnix, 0), em.Data}, nil
}
