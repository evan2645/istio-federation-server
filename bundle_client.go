package main

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/pkg/common/pemutil"
	"github.com/spiffe/spire/pkg/server/bundle/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// BundleEndpointClientConfig is the configuration for the BundleEndpointClient.
type BundleEndpointClientConfig struct {
	TrustDomain      string
	EndpointAddress  string
	EndpointSpiffeID string

	Namespace     string
	ConfigMapName string

	Log logrus.FieldLogger
}

// BundleEndpointClient is the client to retrieve the SPIFFE trust bundle.
type BundleEndpointClient struct {
	cfg        *BundleEndpointClientConfig
	kubeClient *kubernetes.Clientset
}

// StartBundleEndpointClient starts a client to retrieve the SPIFFE trust bundle.
func StartBundleEndpointClient(ctx context.Context, cfg *BundleEndpointClientConfig) error {
	kubeClient, err := newKubeClient()
	if err != nil {
		return err
	}

	b := &BundleEndpointClient{
		cfg:        cfg,
		kubeClient: kubeClient,
	}

	go b.start(ctx)

	return nil
}

func (b *BundleEndpointClient) start(ctx context.Context) {
	// Start with 10 second initial delay.
	initialInterval := 10 * time.Second
	pollInterval := 5 * time.Minute
	retryInterval := 10 * time.Second

	var failing bool
	ticker := time.NewTicker(initialInterval)
	for {
		select {
		case <-ticker.C:
			ok := b.trySync(ctx)

			// Manipulate ticker frequency based on state changes
			// between success and failure
			if !ok && !failing {
				failing = true
				ticker = time.NewTicker(retryInterval)
			} else if ok && failing {
				failing = false
				ticker = time.NewTicker(pollInterval)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *BundleEndpointClient) trySync(ctx context.Context) bool {
	roots, err := b.getEndpointRoots(ctx)
	if err != nil {
		b.cfg.Log.Errorf("Could not retrieve root CAs to validate bundle endpoint for %v: %v", b.cfg.TrustDomain, err)
		return false
	}

	b.cfg.Log.Info("Successfully retrieved roots from ConfigMap")
	currentRoots, err := b.callBundleEndpoint(ctx, roots)
	if err != nil {
		b.cfg.Log.Errorf("Could not retrieve current root CAs from bundle endpoint for %v: %v", b.cfg.TrustDomain, err)
		return false
	}

	b.cfg.Log.Info("Successfully called bundle endpoint to get the current roots")
	err = b.updateRoots(ctx, roots, currentRoots)
	if err != nil {
		b.cfg.Log.Errorf("Could not persist root CA update for %v: %v", b.cfg.TrustDomain, err)
		return false
	}

	return true
}

func (b *BundleEndpointClient) getEndpointRoots(ctx context.Context) ([]*x509.Certificate, error) {
	configMap, err := b.getConfigMap(ctx, b.cfg.Namespace, b.cfg.ConfigMapName)
	if err != nil {
		return nil, err
	}

	certDecoded, err := base64.StdEncoding.DecodeString(configMap.Data["trust_bundle"])
	if err != nil {
		return nil, fmt.Errorf("cannot decode the CA TLS root cert: %v", err)
	}

	roots, err := pemutil.ParseCertificates(certDecoded)
	if err != nil {
		return nil, err
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("no certs found")
	}

	return roots, nil
}

func (b *BundleEndpointClient) callBundleEndpoint(ctx context.Context, roots []*x509.Certificate) ([]*x509.Certificate, error) {
	clientConfig := client.ClientConfig{
		TrustDomain:      b.cfg.TrustDomain,
		EndpointAddress:  b.cfg.EndpointAddress,
		EndpointSpiffeID: b.cfg.EndpointSpiffeID,
		RootCAs:          roots,
	}
	client := client.NewClient(clientConfig)

	bundle, err := client.FetchBundle(ctx)
	if err != nil {
		return nil, err
	}

	return bundle.RootCAs(), nil
}

func (b *BundleEndpointClient) updateRoots(ctx context.Context, roots, currentRoots []*x509.Certificate) error {
	// TODO: Check if we need to actually update anything

	configMap, err := b.getConfigMap(ctx, b.cfg.Namespace, b.cfg.ConfigMapName)
	if err != nil {
		return err
	}

	pemBytes := pemutil.EncodeCertificates(currentRoots)
	base64Byes := base64.StdEncoding.EncodeToString(pemBytes)
	configMap.Data["trust_domain"] = b.cfg.TrustDomain
	configMap.Data["trust_bundle"] = string(base64Byes)

	return b.updateConfigMap(ctx, b.cfg.Namespace, configMap)
}

func (b *BundleEndpointClient) getConfigMap(ctx context.Context, ns, name string) (*corev1.ConfigMap, error) {
	return b.kubeClient.CoreV1().ConfigMaps(ns).Get(name, metav1.GetOptions{})
}

func (b *BundleEndpointClient) updateConfigMap(ctx context.Context, ns string, configMap *corev1.ConfigMap) error {
	_, err := b.kubeClient.CoreV1().ConfigMaps(ns).Update(configMap)
	return err
}

func newKubeClient() (*kubernetes.Clientset, error) {
	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(c)
}
