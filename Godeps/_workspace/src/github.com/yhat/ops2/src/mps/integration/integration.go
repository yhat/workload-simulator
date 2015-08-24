package integration

import (
	"bytes"
	"fmt"
	"os/exec"
)

var (
	RHello           = "yhat/integration-r-helloworld"
	RSVC             = "yhat/integration-r-svc"
	RAptGet          = "yhat/integration-r-helloworld-apt"
	REcho            = "yhat/integration-r-echo"
	PyHello          = "yhat/integration-python-helloworld"
	PyWithSubpackage = "yhat/integration-python-hellopkg:tip"
	PyRelayRides     = "yhat/integration-python-relayrides"
	PyAptGet         = "yhat/integration-python-aptget"
	PyBeerRec        = "yhat/integration-python-beer"
)

func Run(modelImage, username, apikey, endpoint string) error {
	if modelImage == "" {
		return fmt.Errorf("no model provided")
	}
	args := []string{
		"run", "--rm", "--net='host'",
		"-e", "USERNAME=" + username,
		"-e", "APIKEY=" + apikey,
		"-e", "OPS_ENDPOINT=" + endpoint,
		modelImage,
	}

	cmd := exec.Command("docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		s := "command %s failed %v: %s"
		return fmt.Errorf(s, cmd.Args, err, stderr.String())
	}
	return nil
}
