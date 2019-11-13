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
	configFlag = flag.String("config", "istio-federation-server.conf", "configuration file")
)

func main() {
	flag.Parse()
	if err := run(context.Background(), *configFlag); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	log, err := log.NewLogger(config.LogLevel, "", "")
	if err != nil {
		return errs.Wrap(err)
	}
	defer log.Close()

	var handler http.Handler = NewHandler(config.RootCAPath, log)
	if config.LogRequests {
		log.Info("Logging all requests")
		handler = logHandler(log, handler)
	}

	log.Info("Serving HTTPS")
	return http.ListenAndServeTLS("0.0.0.0:443", config.CertPath, config.KeyPath, handler)
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
