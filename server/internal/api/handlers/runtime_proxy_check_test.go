package handlers

import (
	"net/http"
	"testing"
	"time"
)

func TestBuildProxyHTTPClient_ForceAttemptHTTP2Disabled(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		proxyType string
		port      int
	}{
		{"http", 7890},
		{"socks", 7891},
	} {
		client, closeFn, err := buildProxyHTTPClient(tc.proxyType, tc.port, 2*time.Second)
		if err != nil {
			t.Fatalf("%s: %v", tc.proxyType, err)
		}
		closeFn()
		tr, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("%s: expected *http.Transport", tc.proxyType)
		}
		if tr.ForceAttemptHTTP2 {
			t.Fatalf("%s: ForceAttemptHTTP2 should be false for local proxy compatibility", tc.proxyType)
		}
	}
}
