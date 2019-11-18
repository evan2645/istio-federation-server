package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/pkg/common/log"
	"github.com/zeebo/errs"
)

var (
	rootCAPath   = flag.String("rootCAPath", "/etc/client/roots.pem", "File containing this trust domain's root certificates")
	leafCertPath = flag.String("leafCertPath", "/etc/server/cert-chain.pem", "The leaf certificate to use for serving TLS")
	leafKeyPath  = flag.String("leafKeyPath", "/etc/server/key.pem", "The private key of the leaf certificate to serve TLS with")

	peerTrustDomainName = flag.String("peerTrustDomain", "spiffe://cluster-2", "The trust domain name to federate with")
	peerEndpointAddress = flag.String("peerEndpointAddress", "240.0.0.10", "The address of the remote trust domain's bundle endpoint")
	peerSpiffeID        = flag.String("peerSpiffeID", "spiffe://cluster-2/spire/server", "The SPIFFE ID of the remote trust domain's bundle endpoint")

	namespace     = flag.String("namespace", "istio-system", "The namespace of the config map to keep updated with the peer's CA certificates")
	configMapName = flag.String("configMapName", "spiffe-tb-1", "The name of the config map to keep updated with the peer's CA certificates")

	logLevel = flag.String("logLevel", "DEBUG", "The level to log at")
)

func main() {
	flag.Parse()
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	log, err := log.NewLogger(*logLevel, "", "")
	if err != nil {
		return errs.Wrap(err)
	}
	defer log.Close()

	var handler http.Handler = NewHandler(*rootCAPath, log)
	handler = logHandler(log, handler)

	clientConfig := &BundleEndpointClientConfig{
		TrustDomain:      *peerTrustDomainName,
		EndpointAddress:  *peerEndpointAddress,
		EndpointSpiffeID: *peerSpiffeID,

		Namespace:     *namespace,
		ConfigMapName: *configMapName,

		Log: log,
	}
	log.Info("Starting SPIFFE bundle endpoint client")
	err = StartBundleEndpointClient(ctx, clientConfig)
	if err != nil {
		return err
	}

	log.Info("Starting SPIFFE bundle endpoint server")
	return http.ListenAndServeTLS("0.0.0.0:443", *leafCertPath, *leafKeyPath, handler)
}

func logHandler(log logrus.FieldLogger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(logrus.Fields{
			"remote-addr": r.RemoteAddr,
			"method":      r.Method,
			"url":         r.URL,
			"user-agent":  r.UserAgent,
		}).Info("Incoming request")
		handler.ServeHTTP(w, r)
	})
}
