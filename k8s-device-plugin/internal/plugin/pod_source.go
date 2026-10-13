// Copyright (c) 2026, ENFLAME INC.  All rights reserved.
// EF_SUPPORT
//
// Pending-pod source abstraction for by-card scheduling (large-scale support,
// design §5.5). by-card lists "this node's Pending pods that request GCU" on
// every (high-frequency, polled) Allocate. On very large clusters having every
// node's device-plugin hit the central apiserver List adds up, so the source is
// configurable via --pod-source / POD_SOURCE:
//
//	apiserver (default) : direct apiserver List (field selector node+Pending).
//	informer            : node-scoped SharedInformer local cache (one watch per
//	                      node, reuses the occupancy-check PodInformer).
//	kubelet             : node-local kubelet /pods (zero apiserver) for the very
//	                      largest clusters; tolerates kubelet view lag, which the
//	                      by-card annotation-wait loop already absorbs.
package plugin

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"enflame/k8s-device-plugin/internal/informer"
)

const (
	// --pod-source / POD_SOURCE values.
	PodSourceApiServer = "apiserver"
	PodSourceInformer  = "informer"
	PodSourceKubelet   = "kubelet"

	defaultKubeletPort             = "10250"
	defaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	kubeletPodsRequestTimeout      = 10 * time.Second
)

// pendingPodLister supplies this node's Pending pods from a concrete source.
// The returned pods are filtered/deduped by node + phase again in
// PodResource.GetPendingPodsInNode, so an implementation may over-return.
type pendingPodLister interface {
	ListPendingPods() ([]corev1.Pod, error)
}

// informerPendingSource reads node pods from the local informer cache, avoiding
// a per-Allocate apiserver List. Reuses the occupancy-check PodInformer's
// node-scoped (spec.nodeName) lister.
type informerPendingSource struct {
	informer *informer.PodInformer
	nodeName string
}

func (s *informerPendingSource) ListPendingPods() ([]corev1.Pod, error) {
	cached, err := s.informer.ListNodePods()
	if err != nil {
		return nil, err
	}
	out := make([]corev1.Pod, 0, len(cached))
	for _, p := range cached {
		if p == nil || p.Status.Phase != corev1.PodPending {
			continue
		}
		out = append(out, *p)
	}
	return out, nil
}

// kubeletPendingSource reads node pods from the node-local kubelet /pods API,
// completely bypassing the apiserver.
type kubeletPendingSource struct {
	client   *kubeletClient
	nodeName string
}

func (s *kubeletPendingSource) ListPendingPods() ([]corev1.Pod, error) {
	podList, err := s.client.GetPods()
	if err != nil {
		return nil, fmt.Errorf("list pods from kubelet: %w", err)
	}
	out := make([]corev1.Pod, 0, len(podList.Items))
	for _, p := range podList.Items {
		if p.Status.Phase != corev1.PodPending {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// kubeletClient talks to the node-local kubelet read-only-style /pods endpoint
// over its authenticated HTTPS port (default 10250). Auth is a Bearer token
// (the device-plugin ServiceAccount needs nodes/proxy RBAC so kubelet authz
// allows it); the token is re-read per call because projected tokens rotate.
type kubeletClient struct {
	podsURL   string
	tokenPath string
	http      *http.Client
}

// newKubeletClient builds a client for the kubelet on this node. Connection
// target resolves from KUBELET_HOST, else NODE_IP, else 127.0.0.1; port from
// KUBELET_PORT (default 10250). TLS verifies against KUBELET_CA_PATH when set,
// otherwise skips verification (the kubelet serving cert is node-local and
// commonly self-signed). Token path overridable via KUBELET_TOKEN_PATH.
func newKubeletClient(nodeName string) (*kubeletClient, error) {
	host := os.Getenv("KUBELET_HOST")
	if host == "" {
		host = os.Getenv("NODE_IP")
	}
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("KUBELET_PORT")
	if port == "" {
		port = defaultKubeletPort
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true} // #nosec G402 node-local kubelet serving cert
	if caPath := os.Getenv("KUBELET_CA_PATH"); caPath != "" {
		caCert, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("read kubelet CA %s: %w", caPath, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("parse kubelet CA %s", caPath)
		}
		tlsConfig = &tls.Config{RootCAs: pool}
	}

	tokenPath := os.Getenv("KUBELET_TOKEN_PATH")
	if tokenPath == "" {
		tokenPath = defaultServiceAccountTokenPath
	}

	return &kubeletClient{
		podsURL:   fmt.Sprintf("https://%s:%s/pods", host, port),
		tokenPath: tokenPath,
		http: &http.Client{
			Timeout:   kubeletPodsRequestTimeout,
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
		},
	}, nil
}

// GetPods fetches and decodes the kubelet /pods response (a v1.PodList of the
// pods bound to this node). A single attempt: the by-card Allocate loop already
// retries on the timeout window.
func (c *kubeletClient) GetPods() (*corev1.PodList, error) {
	req, err := http.NewRequest(http.MethodGet, c.podsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	// Optional bearer token; absent file (e.g. cert-only authn) is not fatal.
	if tok, terr := os.ReadFile(c.tokenPath); terr == nil {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(tok)))
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kubelet %s returned %d: %s", c.podsURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var podList corev1.PodList
	if err := json.Unmarshal(body, &podList); err != nil {
		return nil, fmt.Errorf("decode kubelet /pods response: %w", err)
	}
	return &podList, nil
}
