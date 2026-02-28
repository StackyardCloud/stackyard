package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func iotSiteWiseRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"iotsitewise",
	)
}

func TestIoTSiteWiseStage0CatalogCoverage(t *testing.T) {
	if len(iotSiteWiseOperations) != 104 {
		t.Fatalf("expected 104 IoT SiteWise operations from docs, got %d", len(iotSiteWiseOperations))
	}
	if len(iotSiteWiseOperationByName) != len(iotSiteWiseOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateAsset",
		"CreateAssetModel",
		"ListAssets",
		"DescribeAsset",
		"BatchPutAssetPropertyValue",
		"ExecuteQuery",
		"UpdateProject",
	}
	for _, action := range requiredActions {
		if _, ok := iotSiteWiseOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(iotSiteWiseDataTypes) != 158 {
		t.Fatalf("expected 158 IoT SiteWise data types from docs, got %d", len(iotSiteWiseDataTypes))
	}
	if len(iotSiteWiseDataTypeByName) != len(iotSiteWiseDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AssetSummary",
		"AssetModelSummary",
		"DashboardSummary",
		"GatewaySummary",
		"ProjectSummary",
		"TimeSeriesSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := iotSiteWiseDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestIoTSiteWiseStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotSiteWiseRequest(t, ts, http.MethodPost, "/unknown-iotsitewise-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIoTSiteWiseKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotSiteWiseRequest(t, ts, http.MethodGet, "/assets?maxResults=1", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "assetSummaries") {
		t.Fatalf("expected ListAssets response body to include assetSummaries, got %q", body)
	}
}
