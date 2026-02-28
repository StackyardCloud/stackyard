package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudControlAPIRequest(t *testing.T, ts *httptest.Server, action string, payload map[string]any) *http.Response {
	t.Helper()
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = encoded
	}
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "CloudApiService." + action,
		},
		"cloudcontrolapi",
	)
}

func cloudControlAPIDecodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	data := mustBody(t, resp)
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var out map[string]any
	if err := decoder.Decode(&out); err != nil {
		t.Fatalf("decode JSON body: %v body=%s", err, string(data))
	}
	return out
}

func TestCloudControlAPIStage0CatalogCoverage(t *testing.T) {
	if len(cloudControlAPIOperations) != 8 {
		t.Fatalf("expected 8 Cloud Control API operations from docs, got %d", len(cloudControlAPIOperations))
	}
	if len(cloudControlAPIOperationByName) != len(cloudControlAPIOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CancelResourceRequest",
		"CreateResource",
		"DeleteResource",
		"GetResource",
		"GetResourceRequestStatus",
		"ListResourceRequests",
		"ListResources",
		"UpdateResource",
	}
	for _, action := range requiredActions {
		if _, ok := cloudControlAPIOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudControlAPIDataTypes) != 4 {
		t.Fatalf("expected 4 Cloud Control API data types from docs, got %d", len(cloudControlAPIDataTypes))
	}
	if len(cloudControlAPIDataTypeByName) != len(cloudControlAPIDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{"ProgressEvent", "HookProgressEvent", "ResourceDescription", "ResourceRequestStatusFilter"}
	for _, typeName := range requiredTypes {
		if _, ok := cloudControlAPIDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCloudControlAPIStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudControlAPIRequest(t, ts, "TotallyUnknownAction", map[string]any{})
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudControlAPIStage0ImplementedActionDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudControlAPIRequest(t, ts, "CreateResource", map[string]any{
		"TypeName":     "AWS::S3::Bucket",
		"DesiredState": `{"BucketName":"stackyard-stage0"}`,
	})
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected CreateResource to be implemented")
	}
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected non-NotImplemented response body, got %q", body)
	}
}
