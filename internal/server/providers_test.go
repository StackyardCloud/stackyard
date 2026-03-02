package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNormalizeEnabledProvidersDefaultsToAWS(t *testing.T) {
	t.Parallel()

	got := normalizeEnabledProviders(nil)
	want := []string{providerAWS}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected default providers %v, got %v", want, got)
	}
}

func TestNormalizeEnabledProvidersDedupesAndFilters(t *testing.T) {
	t.Parallel()

	got := normalizeEnabledProviders([]string{" AWS ", "gcp", "azure", "aws", "unknown", "oci"})
	want := []string{providerAWS, providerGCP, providerAzure, providerOCI}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected providers %v, got %v", want, got)
	}
}

func TestProviderFromPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want string
	}{
		{path: "/", want: providerAWS},
		{path: "/s3/buckets", want: providerAWS},
		{path: "/gcp/storage/v1/b", want: providerGCP},
		{path: "/azure/storage", want: providerAzure},
		{path: "/oci/objectstorage", want: providerOCI},
	}
	for _, tc := range cases {
		if got := providerFromPath(tc.path); got != tc.want {
			t.Fatalf("providerFromPath(%q) expected %q, got %q", tc.path, tc.want, got)
		}
	}
}

func TestProviderRouterReturnsDisabledForNonEnabledProvider(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Addr:      "127.0.0.1:0",
		Providers: []string{providerAWS},
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	req := httptest.NewRequest(http.MethodGet, "/gcp/storage/v1/b", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if body["error"] != "ProviderDisabled" {
		t.Fatalf("expected ProviderDisabled, got %#v", body["error"])
	}
}

func TestProviderRouterReturnsNotImplementedForEnabledProviderStub(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Addr:      "127.0.0.1:0",
		Providers: []string{providerAWS, providerGCP},
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	req := httptest.NewRequest(http.MethodGet, "/gcp/not-implemented", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if body["provider"] != providerGCP {
		t.Fatalf("expected provider %q, got %#v", providerGCP, body["provider"])
	}
}

func TestHealthIncludesEnabledProviders(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Addr:      "127.0.0.1:0",
		Providers: []string{providerAWS, providerGCP},
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	req := httptest.NewRequest(http.MethodGet, "/_stackyard/health", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	providersRaw, ok := body["providers"].([]any)
	if !ok {
		t.Fatalf("expected providers array in health payload, got %#v", body["providers"])
	}
	if len(providersRaw) != 2 {
		t.Fatalf("expected 2 enabled providers, got %#v", providersRaw)
	}
}

func TestProvidersEndpoint(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Addr:      "127.0.0.1:0",
		Providers: []string{providerAWS, providerGCP},
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	req := httptest.NewRequest(http.MethodGet, "/_stackyard/providers", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	enabledRaw, ok := body["enabled"].([]any)
	if !ok || len(enabledRaw) != 2 {
		t.Fatalf("expected enabled providers in payload, got %#v", body["enabled"])
	}
	supportedRaw, ok := body["supported"].([]any)
	if !ok || len(supportedRaw) != 4 {
		t.Fatalf("expected supported providers in payload, got %#v", body["supported"])
	}
}
