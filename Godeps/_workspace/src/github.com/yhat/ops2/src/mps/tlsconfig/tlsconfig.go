/*
Package tlsconfig implements certificate generation needed for secure communication between a client and server.

Server side:

	log.Fatal(tlsconfig.Listen(":8008", "/var/yhat/server/certs", handler))

Client side:

	cli, err := NewClient("/var/yhat/client/certs")
	if err != nil {
		// handle error
	}

	if err = cli.Handshake("10.0.0.1:8008"); err != nil {
		// handle error
	}

	// We're all set! The server is now listening securely on https

	httpclient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: cli.TLSConfig(),
		},
	}

	resp, err := httpclient.Get("https://10.0.0.1:8008/")
*/
package tlsconfig

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	RootCert   string = "root_cert.crt"
	RootKey    string = "root_key.crt"
	ServerCert string = "server_cert.crt"
	ServerKey  string = "server_key.crt"
	ClientKey  string = "client_key.crt"
	ClientCert string = "client_cert.crt"
)

func pemEncode(bytes []byte, t string) []byte {
	b := pem.Block{Type: t, Bytes: bytes}
	return pem.EncodeToMemory(&b)
}

type Client struct {
	rootCert *x509.Certificate
	rootKey  *rsa.PrivateKey

	config *tls.Config
}

type Listener struct {
	ErrorLog *log.Logger

	tlsDir  string
	handler http.Handler
}

func NewListener(tlsDir string, handler http.Handler) (*Listener, error) {
	stat, err := os.Stat(tlsDir)
	if err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("could not access directory %s", tlsDir)
	}
	return &Listener{nil, tlsDir, handler}, nil
}

func Listen(addr, tlsDir string, handler http.Handler) error {
	listener, err := NewListener(tlsDir, handler)
	if err != nil {
		return err
	}

	return listener.Listen(addr)
}

func (h *Listener) listenTLS(addr string) error {
	cert, err := loadCert(filepath.Join(h.tlsDir, RootCert))
	if err != nil {
		return err
	}

	pool := x509.NewCertPool()
	pool.AddCert(cert)
	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  pool,
	}

	s := http.Server{
		Addr:      addr,
		Handler:   h.handler,
		TLSConfig: tlsConfig,
		ErrorLog:  h.ErrorLog,
	}
	certFile := filepath.Join(h.tlsDir, ServerCert)
	keyFile := filepath.Join(h.tlsDir, ServerKey)

	return s.ListenAndServeTLS(certFile, keyFile)
}

func (h *Listener) Listen(addr string) error {
	h.listenTLS(addr) // ignore error

	configured := make(chan struct{})
	errc := make(chan error, 2)

	configure := func(certTmpl *x509.Certificate) {
		certTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
		certTmpl.ExtKeyUsage = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		}
	}

	// insecure cert
	serverCert, serverKey, err := newSelfSignedCert(configure)
	if err != nil {
		return err
	}
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return fmt.Errorf("failed to load x509 cert: %v", err)
	}
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	tlsListener := tls.NewListener(listener, &config)

	callback := func(rootCertPEM, serverCertPEM, serverKeyPEM []byte) {
		files := map[string][]byte{
			RootCert:   rootCertPEM,
			ServerCert: serverCertPEM,
			ServerKey:  serverKeyPEM,
		}
		for name, data := range files {
			name = filepath.Join(h.tlsDir, name)
			if err := ioutil.WriteFile(name, data, 0644); err != nil {
				errc <- err
				return
			}
		}
		configured <- struct{}{}
	}

	hf := func(w http.ResponseWriter, r *http.Request) {
		handleHandshake(w, r, callback)
	}
	s := http.Server{
		Handler: http.HandlerFunc(hf),
	}

	go func() {
		errc <- s.Serve(tlsListener)
	}()

	select {
	case err := <-errc:
		listener.Close()
		return err
	case <-configured:
		listener.Close()
	}
	return h.listenTLS(addr)
}

func handleHandshake(w http.ResponseWriter, r *http.Request, callback func(rootCertPEM, serverCertPEM, serverKeyPEM []byte)) {
	if r.Method != "POST" {
		http.Error(w, "I only respond to POSTs", http.StatusNotImplemented)
		return

	}
	rootCertPEM := []byte(r.FormValue(RootCert))
	serverCertPEM := []byte(r.FormValue(ServerCert))
	serverKeyPEM := []byte(r.FormValue(ServerKey))

	// verify that these values are valid
	for _, v := range [][]byte{rootCertPEM, serverCertPEM, serverKeyPEM} {
		if len(v) == 0 {
			http.Error(w, "Post requires root cert, server cert and server key", http.StatusInternalServerError)
			return
		}
	}
	if _, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM); err != nil {
		http.Error(w, "Invalid server key pair: "+err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := parseCert(rootCertPEM); err != nil {
		http.Error(w, "Invalid root cert: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	callback(rootCertPEM, serverCertPEM, serverKeyPEM)
}

func hasPort(s string) bool { return strings.LastIndex(s, ":") > strings.LastIndex(s, "]") }

// Return a valid server tls configuration to use with this client.
// This is largely for use with the httptest package.
func (c *Client) ServerConfig(ipAddr net.IP) (*tls.Config, error) {
	serverCertPEM, serverKeyPEM, err := newServerCert(c.rootCert, c.rootKey, ipAddr)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AddCert(c.rootCert)
	return &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		Certificates: []tls.Certificate{cert},
	}, nil
}

func (c *Client) Handshake(addr string) error {
	// if it's already configured this will work
	conn, err := tls.Dial("tcp", addr, c.TLSConfig())
	if err == nil {
		conn.Close()
		return nil
	}

	host := addr
	if host == "" {
		return fmt.Errorf("no address provided")
	}
	if hasPort(host) {
		host, _, err = net.SplitHostPort(addr)
		if err != nil || host == "" {
			return fmt.Errorf("failed to parse host from address %s", addr)
		}
	}

	ipAddr := net.ParseIP(host)
	if ipAddr == nil {
		return fmt.Errorf("failed to parse ip from addr '%s'", addr)
	}
	serverCertPEM, serverKeyPEM, err := newServerCert(c.rootCert, c.rootKey, ipAddr)
	if err != nil {
		return err
	}

	rootCertPEM := pemEncode(c.rootCert.Raw, "CERTIFICIATE")

	values := url.Values{}
	values.Add(RootCert, string(rootCertPEM))
	values.Add(ServerCert, string(serverCertPEM))
	values.Add(ServerKey, string(serverKeyPEM))

	httpclient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := httpclient.PostForm("https://"+addr+"/", values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := ""
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			msg = err.Error()
		} else {
			msg = string(body)
		}
		return fmt.Errorf("bad response from listener: %s %s", resp.Status, msg)
	}

	timedOut := time.After(time.Second * 5)
	for {
		select {
		case <-timedOut:
			return fmt.Errorf("tcp dial with new config timed out")
		case <-time.After(100 * time.Millisecond):
		}

		// if it's already configured this will work
		conn, err := tls.Dial("tcp", addr, c.TLSConfig())
		if err == nil {
			conn.Close()
			return nil
		}
	}
}

func newClient(tlsDir string) (*Client, error) {
	certFile := filepath.Join(tlsDir, ClientCert)
	keyFile := filepath.Join(tlsDir, ClientKey)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	rootCert, err := loadCert(filepath.Join(tlsDir, RootCert))
	if err != nil {
		return nil, err
	}
	rootKey, err := loadKey(filepath.Join(tlsDir, RootKey))
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	caPool.AddCert(rootCert)

	config := &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{cert},
	}
	return &Client{rootCert, rootKey, config}, nil
}

func NewClient(tlsDir string) (*Client, error) {

	stat, err := os.Stat(tlsDir)
	if err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("could not access directory %s", tlsDir)
	}

	client, err := newClient(tlsDir)
	if err == nil {
		return client, nil
	}

	rootCert, rootKey, cliCert, cliKey, err := genRootAndClient()
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{
		RootCert:   rootCert,
		RootKey:    rootKey,
		ClientCert: cliCert,
		ClientKey:  cliKey,
	}
	for filename, data := range files {
		filename = filepath.Join(tlsDir, filename)
		if err = ioutil.WriteFile(filename, data, 0644); err != nil {
			return nil, fmt.Errorf("could not write file %s: %v", filename, err)
		}
	}
	return newClient(tlsDir)
}

func genRootAndClient() (rootCert, rootKey, cliCert, cliKey []byte, err error) {
	rootCert, rootKey, err = newRootCert()
	if err != nil {
		return
	}
	cliCert, cliKey, err = newClientCert(rootCert, rootKey)
	return
}

func (c *Client) TLSConfig() (clientConfig *tls.Config) {
	return c.config
}

func newRootCert() (certPEM, keyPEM []byte, err error) {

	configure := func(certTmpl *x509.Certificate) {
		certTmpl.IsCA = true
		certTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	}

	return newSelfSignedCert(configure)
}

func newSelfSignedCert(configure func(certTmpl *x509.Certificate)) (certPEM, keyPEM []byte, err error) {

	// generate a new key-pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	certTmpl, err := certTemplate()
	if err != nil {
		return
	}

	configure(certTmpl)

	certPEM, err = createCert(certTmpl, certTmpl, &key.PublicKey, key)
	if err != nil {
		return
	}

	data := x509.MarshalPKCS1PrivateKey(key)
	keyPEM = pemEncode(data, "RSA PRIVATE KEY")
	return
}

func newClientCert(rootCertPEM, rootKeyPEM []byte) (certPEM, keyPEM []byte, err error) {
	configure := func(certTmpl *x509.Certificate) {
		certTmpl.KeyUsage = x509.KeyUsageDigitalSignature
		certTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	return newCertFromPEM(rootCertPEM, rootKeyPEM, configure)
}

func newServerCert(rootCert *x509.Certificate, rootKey *rsa.PrivateKey, addr net.IP) (certPEM, keyPEM []byte, err error) {
	configure := func(certTmpl *x509.Certificate) {
		certTmpl.KeyUsage = x509.KeyUsageDigitalSignature
		certTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		certTmpl.IPAddresses = []net.IP{addr}
	}
	return newCert(rootCert, rootKey, configure)
}

func newCertFromPEM(rootCertPEM, rootKeyPEM []byte, configure func(*x509.Certificate)) (certPEM, keyPEM []byte, err error) {
	rootCert, err := parseCert(rootCertPEM)
	if err != nil {
		return
	}
	rootKey, err := parseKey(rootKeyPEM)
	if err != nil {
		return
	}
	return newCert(rootCert, rootKey, configure)
}

func newCert(rootCert *x509.Certificate, rootKey *rsa.PrivateKey, configure func(*x509.Certificate)) (certPEM, keyPEM []byte, err error) {
	// generate a new key-pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}
	certTmpl, err := certTemplate()
	if err != nil {
		return
	}
	configure(certTmpl)

	certPEM, err = createCert(certTmpl, rootCert, &key.PublicKey, rootKey)
	if err != nil {
		return
	}
	data := x509.MarshalPKCS1PrivateKey(key)
	keyPEM = pemEncode(data, "RSA PRIVATE KEY")
	return
}

func parseKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no block in PEM data")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func loadKey(filename string) (*rsa.PrivateKey, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return parseKey(data)
}

func parseCert(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no block in PEM data")
	}
	return x509.ParseCertificate(block.Bytes)
}

func loadCert(filename string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return parseCert(data)
}

// helper function to create a cert template with a serial number and other required fields
func certTemplate() (*x509.Certificate, error) {
	// generate a random serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.New("failed to generate serial number: " + err.Error())
	}
	now := time.Now()

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"Yhat, Inc."}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             now,
		NotAfter:              now.Add(time.Hour * 876581), // valid for 100 years
		BasicConstraintsValid: true,
	}
	return &tmpl, nil
}

func createCert(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (certPEM []byte, err error) {

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	return pemEncode(certDER, "CERTIFICATE"), nil
}
