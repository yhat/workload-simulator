package main

// Config defines the configuration for the workload simulator
type Config struct {
	App struct {
		MaxDial   int
		ReportDir string
	}

	Ops struct {
		Host   string
		ApiKey string
		User   string
	}

	Workers struct {
		WorkerProcs int
	}
}
