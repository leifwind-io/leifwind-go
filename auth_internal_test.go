// SPDX-License-Identifier: MPL-2.0

package client

import (
	"net/http"
	"testing"
)

// Token holds c.mu across the token request; with no timeout, an endpoint that
// accepts the connection and never answers stalls every request sharing the
// TokenSource when the caller's context has no deadline.
func TestClientCredentialsDefaultHTTPClientIsBounded(t *testing.T) {
	t.Parallel()
	ts, ok := ClientCredentials("https://issuer.example", "id", "secret").(*ccTokenSource)
	if !ok {
		t.Fatal("ClientCredentials did not return a *ccTokenSource")
	}
	if ts.hc == http.DefaultClient || ts.hc.Timeout <= 0 {
		t.Errorf("default token HTTP client has no timeout (Timeout=%v)", ts.hc.Timeout)
	}
}

func TestClientCredentialsKeepsTheCallersHTTPClient(t *testing.T) {
	t.Parallel()
	hc := &http.Client{}
	ts, ok := ClientCredentials("https://issuer.example", "id", "secret",
		WithCredentialHTTPClient(hc)).(*ccTokenSource)
	if !ok {
		t.Fatal("ClientCredentials did not return a *ccTokenSource")
	}
	if ts.hc != hc {
		t.Error("WithCredentialHTTPClient's client was replaced")
	}
}
