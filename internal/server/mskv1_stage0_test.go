package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func mskv1Request(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "kafka")
}

func TestMSKV1Stage0CatalogCoverage(t *testing.T) {
	if len(mskv1Operations) != 5 {
		t.Fatalf("expected 5 MSK Replicator operations from docs, got %d", len(mskv1Operations))
	}
	if len(mskv1OperationByName) != len(mskv1Operations) {
		t.Fatalf("expected unique MSK Replicator operation names")
	}

	requiredOperations := []string{
		"CreateReplicator",
		"DeleteReplicator",
		"DescribeReplicator",
		"ListReplicators",
		"UpdateReplicationInfo",
	}
	for _, operation := range requiredOperations {
		if _, ok := mskv1OperationByName[operation]; !ok {
			t.Fatalf("missing documented operation %s", operation)
		}
	}

	if len(mskv1Resources) != 27 {
		t.Fatalf("expected 27 MSK Replicator resources/models from docs, got %d", len(mskv1Resources))
	}
	if len(mskv1ResourceByName) != len(mskv1Resources) {
		t.Fatalf("expected unique MSK Replicator resource names")
	}

	requiredResources := []string{
		"CreateReplicatorRequest",
		"CreateReplicatorResponse",
		"DescribeReplicatorResponse",
		"ListReplicatorsResponse",
		"UpdateReplicationInfoResponse",
		"ReplicatorSummary",
	}
	for _, resourceName := range requiredResources {
		if _, ok := mskv1ResourceByName[resourceName]; !ok {
			t.Fatalf("missing documented resource %s", resourceName)
		}
	}
}

func TestMSKV1Stage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskv1Request(t, ts, http.MethodPost, "/replication/v1/mskv1-unknown-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMSKV1Stage0KnownActionReturnsListReplicators(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskv1Request(t, ts, http.MethodGet, "/replication/v1/replicators?maxResults=20", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "replicators") {
		t.Fatalf("expected ListReplicators response body to include replicators, got %q", body)
	}
}

func TestMSKV1Stage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"replicatorArn": mskv1SeedReplicatorARN,
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range mskv1Operations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := mskv1Request(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
