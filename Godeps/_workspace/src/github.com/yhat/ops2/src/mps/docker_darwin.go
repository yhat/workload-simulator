package mps

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func dialDocker() (net.Conn, error) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		return nil, fmt.Errorf("DOCKER_HOST environment variable not set")
	}
	host = strings.TrimPrefix(host, "tcp://")
	certPath := os.Getenv("DOCKER_CERT_PATH")
	if certPath == "" {
		return net.Dial("tcp", host)
	}

	tlsConfig := tls.Config{}
	tlsConfig.InsecureSkipVerify = os.Getenv("DOCKER_TLS_VERIFY") != "1"

	if !tlsConfig.InsecureSkipVerify {

		ca := filepath.Join(certPath, "ca.pem")
		file, err := ioutil.ReadFile(ca)
		if err != nil {
			return nil, fmt.Errorf("Couldn't read ca cert %s: %s", ca, err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(file) {
			return nil, fmt.Errorf("%s contained no certs", ca)
		}
		tlsConfig.RootCAs = certPool

		certFile := filepath.Join(certPath, "cert.pem")
		keyFile := filepath.Join(certPath, "key.pem")
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("Couldn't load X509 key pair: %v", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
		// Avoid fallback to SSL protocols < TLS1.0
		tlsConfig.MinVersion = tls.VersionTLS10
	}

	return tls.Dial("tcp", host, &tlsConfig)
}
