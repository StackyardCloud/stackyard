package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func firehoseRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "Firehose_20150804." + action,
		},
		"firehose",
	)
}

func TestFirehoseStage0CatalogCoverage(t *testing.T) {
	if len(firehoseOperations) != 12 {
		t.Fatalf("expected 12 Firehose operations from docs, got %d", len(firehoseOperations))
	}
	if len(firehoseOperationByName) != len(firehoseOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateDeliveryStream",
		"DeleteDeliveryStream",
		"DescribeDeliveryStream",
		"ListDeliveryStreams",
		"PutRecord",
		"PutRecordBatch",
		"TagDeliveryStream",
		"UntagDeliveryStream",
	}
	for _, action := range requiredActions {
		if _, ok := firehoseOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(firehoseTypes) != 103 {
		t.Fatalf("expected 103 Firehose data types from docs, got %d", len(firehoseTypes))
	}
	if len(firehoseTypeByName) != len(firehoseTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"DeliveryStreamDescription",
		"DestinationDescription",
		"S3DestinationConfiguration",
		"Record",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := firehoseTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestFirehoseStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := firehoseRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestFirehoseKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := firehoseRequest(t, ts, "ListDeliveryStreams", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DeliveryStreamNames") {
		t.Fatalf("expected ListDeliveryStreams response body to include DeliveryStreamNames, got %q", body)
	}
}

func TestFirehoseStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range firehoseOperations {
		resp := firehoseRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
