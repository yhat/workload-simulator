package app

// OpsConfig is a type used to define the target Ops
// server
type OpsConfig struct {
	Host   string
	ApiKey string
	User   string
}

// App defines the app and configs
type App struct {
	Host string
	Port int

	MaxDial   int
	ReportDir string

	Ops *OpsConfig

	WorkersProcs int
}
