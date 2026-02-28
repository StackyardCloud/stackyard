package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func dataExchangeRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "dataexchange")
}

func TestDataExchangeStage0CatalogCoverage(t *testing.T) {
	if len(dataExchangeOperations) != 37 {
		t.Fatalf("expected 37 Data Exchange operations from docs, got %d", len(dataExchangeOperations))
	}
	if len(dataExchangeOperationByName) != len(dataExchangeOperations) {
		t.Fatalf("expected unique Data Exchange operation names")
	}

	requiredActions := []string{
		"CreateDataSet",
		"ListDataSets",
		"CreateRevision",
		"CreateJob",
		"ListJobs",
		"TagResource",
		"SendApiAsset",
	}
	for _, action := range requiredActions {
		if _, ok := dataExchangeOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(dataExchangeDataTypes) != 67 {
		t.Fatalf("expected 67 Data Exchange data types from docs, got %d", len(dataExchangeDataTypes))
	}
	if len(dataExchangeDataTypeByName) != len(dataExchangeDataTypes) {
		t.Fatalf("expected unique Data Exchange data type names")
	}

	requiredTypes := []string{
		"DataSetEntry",
		"RevisionEntry",
		"JobEntry",
		"EventActionEntry",
		"DataGrantSummaryEntry",
		"UpdateRevision",
	}
	for _, typeName := range requiredTypes {
		if _, ok := dataExchangeDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDataExchangeStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataExchangeRequest(t, ts, http.MethodPost, "/unknown-dataexchange-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDataExchangeStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataExchangeRequest(t, ts, http.MethodGet, "/v1/data-sets?maxResults=10", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DataSets") {
		t.Fatalf("expected ListDataSets response body to include DataSets, got %q", body)
	}
}

func TestDataExchangeStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range dataExchangeOperations {
		path := dataExchangeRenderTestURI(op.URI)
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPatch {
			payload = `{}`
		}

		resp := dataExchangeRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s (path=%s)", op.Name, resp.StatusCode, body, path)
		}
	}
}

var dataExchangeURIPlaceholderPattern = regexp.MustCompile(`\{([^}]+)\}`)

func dataExchangeRenderTestURI(uriTemplate string) string {
	return dataExchangeURIPlaceholderPattern.ReplaceAllStringFunc(uriTemplate, func(raw string) string {
		placeholder := strings.TrimSuffix(strings.Trim(raw, "{}"), "+")
		switch strings.ToLower(placeholder) {
		case "datagrantarn":
			return url.QueryEscape("arn:aws:dataexchange:us-east-1:123456789012:data-grants/dg-000001")
		case "datagrantid":
			return "dg-000001"
		case "jobid":
			return "job-000001"
		case "datasetid":
			return "ds-000001"
		case "revisionid":
			return "rev-000001"
		case "assetid":
			return "asset-000001"
		case "eventactionid":
			return "ea-000001"
		case "resourcearn":
			return url.QueryEscape("arn:aws:dataexchange:us-east-1:123456789012:data-sets/ds-000001")
		case "maxresults":
			return "10"
		case "nexttoken":
			return "token-000001"
		case "origin":
			return "OWNED"
		case "eventsourceid":
			return "evt-src-000001"
		case "acceptancestate":
			return "PENDING"
		case "tagkeys":
			return "env"
		case "querystringparameters":
			return "assetId=asset-000001"
		default:
			return "stackyard"
		}
	})
}
