package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/instruqt/jumppad/pkg/clients/connector/types"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/jumppad-labs/connector/crypto"
	"github.com/jumppad-labs/connector/protos/shipyard"
	"github.com/jumppad-labs/gohup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Connector defines a client which can be used for interfacing with the
// Shipyard connector

//go:generate mockery --name Connector --filename connector.go
type Connector interface {
	// Start the Connector, returns an error on failure
	Start(*types.CertBundle) error
	// Stop the Connector, returns an error on failure
	Stop() error
	// IsRunning returns true when the Connector is running
	IsRunning() bool

	// GenerateLocalCertBundle generates a root CA and leaf certificate for
	// securing connector communications for the local instance
	// this function is a convenience function which wraps other
	// methods
	GenerateLocalCertBundle(out string) (*types.CertBundle, error)

	// Fetches the local certificate bundle from the given directory
	// if any of the required files do not exist an error and a nil
	// CertBundle will be returned
	GetLocalCertBundle(dir string) (*types.CertBundle, error)

	// Generates a Leaf certificate for securing a connector
	GenerateLeafCert(
		privateKey, rootCA string,
		hosts, ips []string,
		dir string) (*types.CertBundle, error)

	// ExposeService allows you to expose a local or remote
	// service with another connector
	ExposeService(
		name string,
		port int,
		remoteAddr string,
		destAddr string,
		direction string,
	) (string, error)

	// RemoveService removes a previously exposed service
	RemoveService(id string) error

	// ListServices returns a slice of active services
	ListServices() ([]*shipyard.Service, error)
}

// ConnectorImpl is a concrete implementation of the Connector interface
type ConnectorImpl struct {
	options ConnectorOptions
}

type ConnectorOptions struct {
	LogDirectory string
	BinaryPath   string
	GrpcBind     string
	HTTPBind     string
	APIBind      string
	LogLevel     string
	PidFile      string
}

func DefaultConnectorOptions() ConnectorOptions {
	co := ConnectorOptions{}
	co.LogDirectory = utils.LogsDir()
	co.BinaryPath = utils.GetJumppadBinaryPath()
	co.GrpcBind = ":30001"
	co.HTTPBind = ":30002"
	co.APIBind = ":30003"
	co.LogLevel = "info"
	co.PidFile = utils.GetConnectorPIDFile()

	return co
}

// NewConnector creates a new connector with the given options
func NewConnector(opts ConnectorOptions) Connector {
	return &ConnectorImpl{options: opts}
}

// Start the Connector, returns an error on failure
func (c *ConnectorImpl) Start(cb *types.CertBundle) error {
	// get the log level from the environment variable
	ll := os.Getenv("LOG_LEVEL")
	if ll == "" {
		ll = "info"
	}

	args := []string{
		"--non-interactive",
		"connector",
		"run",
		"--grpc-bind", c.options.GrpcBind,
		"--http-bind", c.options.HTTPBind,
		"--api-bind", c.options.APIBind,
		"--root-cert-path", cb.RootCertPath,
		"--server-cert-path", cb.LeafCertPath,
		"--server-key-path", cb.LeafKeyPath,
		"--log-level", ll,
	}

	// if the binary path contains a space, split this to args
	if strings.Contains(c.options.BinaryPath, " ") {
		parts := strings.Split(c.options.BinaryPath, " ")
		c.options.BinaryPath = parts[0]

		args = append(parts[1:], args...)
	}

	lp := &gohup.LocalProcess{}
	o := gohup.Options{
		Path:    c.options.BinaryPath,
		Args:    args,
		Logfile: filepath.Join(c.options.LogDirectory, "connector.log"),
		Pidfile: c.options.PidFile,
	}

	var err error
	_, c.options.PidFile, err = lp.Start(o)
	return err
}

// Stop the Connector, returns an error on failure
func (c *ConnectorImpl) Stop() error {
	lp := &gohup.LocalProcess{}
	return lp.Stop(c.options.PidFile)
}

// IsRunning returns true when the Connector is running
func (c *ConnectorImpl) IsRunning() bool {
	lp := &gohup.LocalProcess{}
	status, err := lp.QueryStatus(c.options.PidFile)
	if err != nil {
		return false
	}

	if status == gohup.StatusRunning {
		return true
	}

	return false
}

// creates a CA and local leaf cert
func (c *ConnectorImpl) GenerateLocalCertBundle(out string) (*types.CertBundle, error) {
	cb := &types.CertBundle{
		RootCertPath: filepath.Join(out, "root.cert"),
		RootKeyPath:  filepath.Join(out, "root.key"),
		LeafCertPath: filepath.Join(out, "leaf.cert"),
		LeafKeyPath:  filepath.Join(out, "leaf.key"),
	}

	// create the CA
	rk, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	ca, err := crypto.GenerateCA("Connector CA", rk.Private)
	if err != nil {
		return nil, err
	}

	os.Remove(cb.RootKeyPath)
	err = rk.Private.WriteFile(cb.RootKeyPath)
	if err != nil {
		return nil, err
	}

	os.Remove(cb.RootCertPath)
	err = ca.WriteFile(cb.RootCertPath)
	if err != nil {
		return nil, err
	}

	grcpParts := strings.Split(c.options.GrpcBind, ":")
	httpParts := strings.Split(c.options.GrpcBind, ":")

	ips := utils.GetLocalIPAddresses()
	host := []string{
		utils.GetHostname(),
		fmt.Sprintf("localhost:%s", grcpParts[1]),
		fmt.Sprintf("localhost:%s", httpParts[1]),
	}

	return c.GenerateLeafCert(cb.RootKeyPath, cb.RootCertPath, host, ips, out)
}

func (c *ConnectorImpl) GetLocalCertBundle(dir string) (*types.CertBundle, error) {
	cb := &types.CertBundle{
		RootCertPath: filepath.Join(dir, "root.cert"),
		RootKeyPath:  filepath.Join(dir, "root.key"),
		LeafCertPath: filepath.Join(dir, "leaf.cert"),
		LeafKeyPath:  filepath.Join(dir, "leaf.key"),
	}

	// test to see if files exist
	f1, err := os.Open(cb.RootCertPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to find root certificate")
	}
	defer f1.Close()

	f2, err := os.Open(cb.RootKeyPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to find root key")
	}
	defer f2.Close()

	f3, err := os.Open(cb.LeafCertPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to find leaf certificate")
	}
	defer f3.Close()

	f4, err := os.Open(cb.LeafKeyPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to find leaf key")
	}
	defer f4.Close()

	// check that the certificate still has 24hrs left on it
	leaf := &crypto.X509{}
	err = leaf.ReadFile(cb.LeafCertPath)
	if err != nil {
		return nil, err
	}

	if time.Until(leaf.NotAfter) < 24*time.Hour {
		return nil, fmt.Errorf("leaf certificate expires in under 24 hours")
	}

	return cb, nil
}

// GenerateLeafCert generates a x509 leaf certificate with the given details
func (c *ConnectorImpl) GenerateLeafCert(
	rootKey, rootCA string, host, ips []string, dir string) (*types.CertBundle, error) {

	cb := &types.CertBundle{
		RootCertPath: rootCA,
		RootKeyPath:  rootKey,
		LeafCertPath: path.Join(dir, "leaf.cert"),
		LeafKeyPath:  path.Join(dir, "leaf.key"),
	}

	// load the root key
	rk := &crypto.PrivateKey{}
	err := rk.ReadFile(cb.RootKeyPath)
	if err != nil {
		return nil, err
	}

	// load the ca
	ca := &crypto.X509{}
	err = ca.ReadFile(cb.RootCertPath)
	if err != nil {
		return nil, err
	}

	// generate a local cert
	k, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	os.Remove(cb.LeafKeyPath)
	err = k.Private.WriteFile(cb.LeafKeyPath)
	if err != nil {
		return nil, err
	}

	hosts := []string{"localhost", fmt.Sprintf("*.local.%s", utils.LocalTLD), c.options.GrpcBind}
	hosts = append(hosts, host...)

	lc, err := crypto.GenerateLeaf(
		"Connector Leaf",
		ips,
		hosts,
		ca,
		rk,
		k.Private)
	if err != nil {
		return nil, err
	}

	os.Remove(cb.LeafCertPath)
	err = lc.WriteFile(cb.LeafCertPath)
	if err != nil {
		return nil, err
	}

	return cb, nil
}

// ExposeService allows you to expose a local or remote
// service with another connector
func (c *ConnectorImpl) ExposeService(
	name string,
	port int,
	remoteAddr string,
	destAddr string,
	direction string,
) (string, error) {

	dir := utils.CertsDir("")
	cb, err := c.GetLocalCertBundle(dir)
	if err != nil {
		return "", fmt.Errorf("unable to find certificate at location: %s, error: %s", dir, err)
	}

	cl, err := getClient(cb, c.options.GrpcBind)
	if err != nil {
		return "", fmt.Errorf("unable to create grpc client: %s", err)
	}

	t := shipyard.ServiceType_LOCAL
	if direction == "remote" {
		t = shipyard.ServiceType_REMOTE
	}

	r := &shipyard.ExposeRequest{}
	r.Service = &shipyard.Service{
		Name:                name,
		RemoteConnectorAddr: remoteAddr,
		DestinationAddr:     destAddr,
		SourcePort:          int32(port),
		Type:                t,
	}

	er, err := cl.ExposeService(context.Background(), r)
	if err != nil {
		return "", err
	}

	return er.Id, nil
}

// RemoveService removes a previously exposed service
func (c *ConnectorImpl) RemoveService(id string) error {
	cb, err := c.GetLocalCertBundle(utils.CertsDir(""))
	if err != nil {
		return err
	}

	cl, err := getClient(cb, c.options.GrpcBind)
	if err != nil {
		return err
	}

	r := &shipyard.DestroyRequest{}
	r.Id = id

	_, err = cl.DestroyService(context.Background(), r)
	if err != nil {
		return err
	}

	return nil
}

// ListServices lists all active services
func (c *ConnectorImpl) ListServices() ([]*shipyard.Service, error) {
	cb, err := c.GetLocalCertBundle(utils.CertsDir(""))
	if err != nil {
		return nil, err
	}

	cl, err := getClient(cb, c.options.GrpcBind)
	if err != nil {
		return nil, err
	}

	lr, err := cl.ListServices(context.Background(), &shipyard.NullMessage{})
	if err != nil {
		return nil, err
	}

	return lr.Services, nil
}

func getClient(cert *types.CertBundle, uri string) (shipyard.RemoteConnectionClient, error) {
	// if we are using TLS create a TLS client
	certificate, err := tls.LoadX509KeyPair(cert.LeafCertPath, cert.LeafKeyPath)
	if err != nil {
		return nil, err
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(cert.RootCertPath)
	if err != nil {
		return nil, err
	}

	ok := certPool.AppendCertsFromPEM(ca)
	if !ok {
		return nil, fmt.Errorf("unable to append certs from ca pem")
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   uri,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	_ = creds

	// Create a connection with the TLS credentials
	conn, err := grpc.NewClient(uri, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	rc := shipyard.NewRemoteConnectionClient(conn)

	return rc, nil
}
