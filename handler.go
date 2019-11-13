package main

import (
	"bytes"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/pkg/common/bundleutil"
	"github.com/spiffe/spire/pkg/common/pemutil"
)

type Handler struct {
	log        logrus.FieldLogger
	rootCAPath string

	http.Handler
}

func NewHandler(rootCAPath string, log logrus.FieldLogger) *Handler {
	h := &Handler{
		log:        log,
		rootCAPath: rootCAPath,
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(h.serveKeys))

	h.Handler = mux
	return h
}

func (h *Handler) serveKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fi, err := os.Stat(h.rootCAPath)
	if err != nil {
		h.log.Errorf("Could not read root CAs from %v: %v", h.rootCAPath, err)
		http.Error(w, "document not available", http.StatusInternalServerError)
		return
	}
	modTime := fi.ModTime()

	certs, err := pemutil.LoadCertificates(h.rootCAPath)
	if err != nil {
		h.log.Errorf("Could not read root CAs from %v: %v", h.rootCAPath, err)
		http.Error(w, "document not available", http.StatusInternalServerError)
		return
	}

	// TODO: it wants trust domain here, is it really needed?
	bundle := bundleutil.BundleFromRootCAs("", certs)
	jwksBytes, err := bundleutil.Marshal(bundle, bundleutil.NoJWTSVIDKeys())
	if err != nil {
		h.log.Errorf("Could not parse root CAs from %v: %v", h.rootCAPath, err)
		http.Error(w, "document not available", http.StatusInternalServerError)
		return
	}

	// Disable caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	w.Header().Set("Content-Type", "application/json")
	http.ServeContent(w, r, "keys", modTime, bytes.NewReader(jwksBytes))
}
