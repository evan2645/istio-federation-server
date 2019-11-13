package main

import (
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/zeebo/errs"
)

const (
	defaultLogLevel = "info"
)

type Config struct {
	LogLevel string `hcl:"log_level"`

	// LogRequests is a debug option that logs all incoming requests
	LogRequests bool `hcl:"log_requests"`

	// RootCAPath is the file path to the root CA certificates representing
	// this trust domain
	RootCAPath string `hcl:"root_ca_path"`

	// CertPath is the file path to the certificate to use for serving
	// the bundle endpoint
	CertPath string `hcl:"cert_path"`

	// KeyPath is the file path to the private key for the certificate
	// we use to serve the bundle endpoint
	KeyPath string `hcl:"key_path"`
}

func LoadConfig(path string) (*Config, error) {
	hclBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errs.New("unable to load configuration: %v", err)
	}
	return ParseConfig(string(hclBytes))
}

func ParseConfig(hclConfig string) (_ *Config, err error) {
	c := new(Config)
	if err := hcl.Decode(c, hclConfig); err != nil {
		return nil, errs.New("unable to decode configuration: %v", err)
	}

	if c.LogLevel == "" {
		c.LogLevel = defaultLogLevel
	}

	if c.RootCAPath == "" {
		return nil, errs.New("Root CA path must be configured")
	}

	if c.CertPath == "" {
		return nil, errs.New("Cert path must be configured")
	}

	if c.KeyPath == "" {
		return nil, errs.New("Key path must be configured")
	}

	return c, nil
}
