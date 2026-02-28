package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func mskRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "kafka")
}

func TestMSKStage0CatalogCoverage(t *testing.T) {
	if len(mskOperations) != 5 {
		t.Fatalf("expected 5 MSK operations from docs, got %d", len(mskOperations))
	}
	if len(mskOperationByName) != len(mskOperations) {
		t.Fatalf("expected unique MSK operation names")
	}

	requiredActions := []string{
		"CreateClusterV2",
		"DescribeClusterV2",
		"ListClustersV2",
		"DescribeClusterOperationV2",
		"ListClusterOperationsV2",
	}
	for _, action := range requiredActions {
		if _, ok := mskOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mskResources) != 74 {
		t.Fatalf("expected 74 MSK resources/models from docs, got %d", len(mskResources))
	}
	if len(mskResourceByName) != len(mskResources) {
		t.Fatalf("expected unique MSK resource names")
	}

	requiredResources := []string{
		"Cluster",
		"CreateClusterV2Request",
		"CreateClusterV2Response",
		"ListClustersV2Response",
		"DescribeClusterOperationV2Response",
		"ClusterOperationV2Summary",
	}
	for _, resourceName := range requiredResources {
		if _, ok := mskResourceByName[resourceName]; !ok {
			t.Fatalf("missing documented resource %s", resourceName)
		}
	}
}

func TestMSKStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskRequest(t, ts, http.MethodPost, "/api/v2/unknown-msk-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMSKStage0KnownActionReturnsListClustersV2(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskRequest(t, ts, http.MethodGet, "/api/v2/clusters", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "clusterInfoList") {
		t.Fatalf("expected ListClustersV2 response body to include clusterInfoList, got %q", body)
	}
}

func TestMSKStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"clusterArn":          "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7",
		"clusterOperationArn": "arn:aws:kafka:us-east-1:123456789012:cluster-operation/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range mskOperations {
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

		resp := mskRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
