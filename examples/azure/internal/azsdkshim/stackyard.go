package azsdkshim

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

const defaultAccessToken = "stackyard-token"

// StaticTokenCredential is a minimal TokenCredential for local emulator usage.
type StaticTokenCredential struct {
	Token     string
	ExpiresOn time.Time
}

// GetToken satisfies azcore.TokenCredential.
func (c StaticTokenCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	token := strings.TrimSpace(c.Token)
	if token == "" {
		token = defaultAccessToken
	}
	expiresOn := c.ExpiresOn
	if expiresOn.IsZero() {
		expiresOn = time.Now().Add(24 * time.Hour)
	}
	return azcore.AccessToken{
		Token:     token,
		ExpiresOn: expiresOn,
	}, nil
}

// Transport adapts Azure SDK and net/http clients to Stackyard Azure routes.
type Transport struct {
	Base            http.RoundTripper
	PathPrefix      string
	Account         string
	SubscriptionKey string
}

// NewTransport builds a transport that can prepend a route prefix and inject
// SharedKey/Ocp-Apim headers expected by Stackyard Azure auth mode.
func NewTransport(pathPrefix, account, subscriptionKey string) *Transport {
	return &Transport{
		Base:            http.DefaultTransport,
		PathPrefix:      normalizePrefix(pathPrefix),
		Account:         strings.TrimSpace(account),
		SubscriptionKey: strings.TrimSpace(subscriptionKey),
	}
}

// Do implements policy.Transporter for Azure Track 2 clients.
func (t *Transport) Do(req *http.Request) (*http.Response, error) {
	return t.RoundTrip(req)
}

// RoundTrip implements http.RoundTripper for generic HTTP clients.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	cloned := req.Clone(req.Context())
	cloned.URL = cloneURL(req.URL)
	cloned.Header = req.Header.Clone()

	if t.PathPrefix != "" {
		cloned.URL.Path = withPrefix(cloned.URL.Path, t.PathPrefix)
		cloned.URL.RawPath = ""
	}
	if t.Account != "" {
		cloned.Header.Set("Authorization", "SharedKey "+t.Account+":signature")
	}
	if t.SubscriptionKey != "" {
		cloned.Header.Set("Ocp-Apim-Subscription-Key", t.SubscriptionKey)
	}

	return base.RoundTrip(cloned)
}

func normalizePrefix(prefix string) string {
	trimmed := strings.Trim(strings.TrimSpace(prefix), "/")
	if trimmed == "" {
		return ""
	}
	return "/" + trimmed
}

func withPrefix(path, prefix string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path == prefix || strings.HasPrefix(path, prefix+"/") {
		return path
	}
	if path == "/" {
		return prefix
	}
	return prefix + path
}

func cloneURL(in *url.URL) *url.URL {
	if in == nil {
		return &url.URL{}
	}
	out := *in
	return &out
}
