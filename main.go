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
	rootCAPath   = flag.String("rootCAPath", "./roots.pem", "File containing trust domain root certificates")
	leafCertPath = flag.String("leafCertPath", "./leaf.pem", "The leaf certificate to use for serving TLS")
	leafKeyPath  = flag.String("leafKeyPath", "./leaf.key", "The private key of the leaf certificate to serve TLS with")

	logLevel    = flag.String("logLevel", "DEBUG", "The level to log at")
	logRequests = flag.Bool("logRequests", false, "If set to true, all requests will be logged")
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
	if *logRequests {
		log.Info("Logging all requests")
		handler = logHandler(log, handler)
	}

	log.Info("Serving HTTPS")
	return http.ListenAndServeTLS("0.0.0.0:443", *leafCertPath, *leafKeyPath, handler)
}

func logHandler(log logrus.FieldLogger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(logrus.Fields{
			"remote-addr": r.RemoteAddr,
			"method":      r.Method,
			"url":         r.URL,
			"user-agent":  r.UserAgent,
		}).Debug("Incoming request")
		handler.ServeHTTP(w, r)
	})
}
