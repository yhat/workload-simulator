package app

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"
)

// Stat represents request statistics
type Stat struct {
	batchId   string
	modelId   string
	modelName string
	nreqSent  int
	nreqDone  int
	dt        float64
}

type Metric struct {
	reqSent     int
	reqComplete int
	reqPerSec   int
}

type Report struct {
	batchId     string
	modelName   string
	modelId     string
	requestData string
	requestSent int
	requestDone int
}

func StatsMonitor(report chan<- *Report, dt time.Duration) chan *Stat {
	stats := make(chan *Stat)
	// TODO: Merge these stats into one map after we figure out how
	// to change the front end.
	ticker := time.NewTicker(dt)

	// intialize vars
	requestPerSec := make(map[string]int)
	requestMetrics := make(map[string]Metric)
	reqSent := 0
	reqDone := 0
	batchId := ""
	modelId := ""
	modelName := ""

	go func() {
		for {
			select {
			case <-ticker.C:
				// send report of stats.
				r, err := json.Marshal(requestPerSec)
				if err != nil {
					fmt.Printf("error marshalling json stats: %v\n", err)
					return
				}
				b := bytes.NewBuffer(r)
				report <- &Report{batchId, modelName, modelId, b.String(), reqSent, reqDone}
			case s := <-stats:
				// increment state
				batchId = s.batchId
				modelName = s.modelName
				reqs := float64(s.nreqDone) / s.dt
				reqSent += s.nreqSent
				reqDone += s.nreqDone
				newStat := Metric{reqSent, reqDone, int(reqs)}
				requestMetrics[s.modelId] = newStat
				requestPerSec[s.modelId] = int(reqs)
			}

		}
	}()
	return stats
}

// CsvMmetric represents a row in a Csv output.
type CsvMetric struct {
	// timestamp and unique batchId
	ts      time.Time
	batchId string

	// ops related data
	opsHost      string
	opsUser      string
	opsModelName string

	// request data.
	reqSent     int
	reqComplete int
	reqPerSec   int
}

func (c *CsvMetric) ConvertCsvMetric() []string {
	s := []string{
		c.ts.String(),
		c.batchId,
		c.opsHost,
		c.opsUser,
		c.opsModelName,
		strconv.Itoa(c.reqSent),
		strconv.Itoa(c.reqComplete),
		strconv.Itoa(c.reqPerSec),
	}
	return s
}

// WriteCsv writes a slice of CsvMetrics to a writer and returns any error encountered.
func WriteCsv(w io.Writer, records []*CsvMetric) error {
	wcsv := csv.NewWriter(w)
	for _, record := range records {
		s := record.ConvertCsvMetric()
		if err := wcsv.Write(s); err != nil {
			log.Fatalln("error writing record to csv:", err)
		}
	}

	// Write any buffered data to the underlying writer (standard output).
	wcsv.Flush()

	if err := wcsv.Error(); err != nil {
		return fmt.Errorf("error writing csv: %v", err)
	}
	return nil
}
