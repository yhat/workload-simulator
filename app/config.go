package app

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Web struct {
		Hostname  string `yaml:"hostname,omitempty"`
		HttpPort  int    `yaml:"http_port,omitempty"`
		PublicDir string `yaml:"public_dir,omitempty"`
		ViewsDir  string `yaml:"views_dir,omitempty"`
		ReportDir string `yaml:"report_dir,omitempty"`
	}

	Settings struct {
		MaxDial    int `yaml:"max_dial,omitempty"`
		MaxWorkers int `yaml:"max_workers,omitempty"`
	}
}

// ReadConfig reads in a YAML config file for the Workload simulator app.
func ReadConfig(configPath string) (Config, error) {
	var cfg Config

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("error opening config file %s: %v", configPath, err)
	}

	if err = yaml.Unmarshal(content, &cfg); err != nil {
		return cfg, fmt.Errorf("error reading config file %s: %v", configPath, err)
	}
	return cfg, err
}
