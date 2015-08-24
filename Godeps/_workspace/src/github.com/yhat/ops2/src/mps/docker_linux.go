package mps

import "net"

func dialDocker() (net.Conn, error) {
	return net.Dial("unix", "/var/run/docker.sock")
}
