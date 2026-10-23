package room

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// nekoPostHTTP sends a JSON request to neko over the network, for runtimes
// that reach rooms by address (swarm, k8s)
func nekoPostHTTP(ctx context.Context, client *http.Client, addr, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+addr+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("neko answered %s", res.Status)
	}
	return data, nil
}

func (manager *SwarmManagerCtx) NekoPost(ctx context.Context, id string, path string, body []byte) ([]byte, error) {
	s, err := manager.service(ctx, id)
	if err != nil {
		return nil, err
	}
	return nekoPostHTTP(ctx, manager.http, s.Spec.Labels[ProxyHostLabel], path, body)
}

func (m *K8sManagerCtx) NekoPost(ctx context.Context, id string, path string, body []byte) ([]byte, error) {
	d, err := m.deployment(ctx, id)
	if err != nil {
		return nil, err
	}
	return nekoPostHTTP(ctx, m.http, d.Annotations[ProxyHostLabel], path, body)
}
