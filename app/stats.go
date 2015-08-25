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

type Metric struct {
	reqSent     int
	reqComplete int
	reqPerSec   int
}

type Report struct {
	batchId   string
	modelName string
	modelId   string

	// ops server data
	opsHost string
	user    string

	// requests per second
	requestPerS string

	// cumulative stats
	requestSent int
	requestDone int
}

// Stat represents request statistics
type Stat struct {
	// Contains metadata about work being done by the worker.
	workload *Workload

	// Variable statistics.
	nreqSent int
	nreqDone int
}

func StatsMonitor(report chan<- *Report, dt time.Duration) chan *Stat {
	stats := make(chan *Stat)
	ticker := time.NewTicker(dt)

	var isent int
	var idone int

	// batchId, modelId, and modelName
	var bid string
	var mid string
	var mn string

	var host string
	var user string

	// maps modelId to req/s for the front end.
	requestPerSec := make(map[string]int)

	// other stats not used in the front end.
	requestMetrics := make(map[string]Metric)

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
				report <- &Report{
					batchId:     bid,
					modelName:   mn,
					modelId:     mid,
					opsHost:     host,
					user:        user,
					requestPerS: b.String(),
					requestSent: isent,
					requestDone: idone,
				}
			case s := <-stats:
				// increment state counters
				tt := s.workload.dt
				reqPerS := float64(s.nreqDone) / tt.Seconds()

				bid = s.workload.batchId
				mn = s.workload.modelName
				mid = s.workload.modelId

				host = s.workload.opsHost
				user = s.workload.user

				isent += s.nreqSent
				idone += s.nreqDone

				newStat := Metric{isent, idone, int(reqPerS)}
				requestMetrics[mid] = newStat
				requestPerSec[mid] = int(reqPerS)
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

	// worker data
	nWorkers int

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
		strconv.Itoa(c.nWorkers),
		strconv.Itoa(c.reqSent),
		strconv.Itoa(c.reqComplete),
		strconv.Itoa(c.reqPerSec),
	}
	return s
}

func WriteHeader(w io.Writer) error {
	wcsv := csv.NewWriter(w)
	header := []string{
		"timestamp",
		"batch_id",
		"ops_host",
		"ops_user",
		"model_name",
		"workers",
		"requests_sent",
		"requests_completed",
		"requests_per_second",
	}
	if err := wcsv.Write(header); err != nil {
		log.Fatalln("error writing record to csv:", err)
	}
	wcsv.Flush()
	if err := wcsv.Error(); err != nil {
		return fmt.Errorf("error writing csv: %v", err)
	}
	return nil
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
