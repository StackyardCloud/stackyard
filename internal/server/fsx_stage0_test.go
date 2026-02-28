package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFSxStage0CatalogCoverage(t *testing.T) {
	if len(fsxOperations) != 48 {
		t.Fatalf("expected 48 FSx operations from docs, got %d", len(fsxOperations))
	}
	if len(fsxOperationByName) != len(fsxOperations) {
		t.Fatalf("expected unique FSx operation names")
	}

	requiredActions := []string{
		"CreateFileSystem",
		"DescribeFileSystems",
		"UpdateFileSystem",
		"DeleteFileSystem",
		"CreateBackup",
		"DescribeVolumes",
		"CreateStorageVirtualMachine",
		"CreateVolume",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := fsxOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(fsxDataTypes) != 119 {
		t.Fatalf("expected 119 FSx data types from docs, got %d", len(fsxDataTypes))
	}
	if len(fsxDataTypeByName) != len(fsxDataTypes) {
		t.Fatalf("expected unique FSx data type names")
	}

	requiredTypes := []string{
		"FileSystem",
		"FileCache",
		"Snapshot",
		"Backup",
		"StorageVirtualMachine",
		"Volume",
		"OntapFileSystemConfiguration",
		"OpenZFSVolumeConfiguration",
		"WindowsFileSystemConfiguration",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := fsxDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func fsxRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSFSx." + action,
		},
		"fsx",
	)
}

func TestFSxStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fsxRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestFSxStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fsxRequest(t, ts, "DescribeFileSystems", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "FileSystems") {
		t.Fatalf("expected DescribeFileSystems response body to include FileSystems, got %q", body)
	}
}

func TestFSxStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range fsxOperations {
		resp := fsxRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}

func TestFSxStage0AcceptsAWSSimbaTargetPrefix(t *testing.T) {
	parsed := parseFSxTarget("AWSSimbaAPIService_v20180301.DescribeFileSystems")
	if parsed != "DescribeFileSystems" {
		t.Fatalf("expected parser to return DescribeFileSystems, got %q", parsed)
	}
	if _, ok := fsxOperationByName[parsed]; !ok {
		t.Fatalf("expected %q to be a known FSx operation", parsed)
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{}`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSSimbaAPIService_v20180301.DescribeFileSystems",
		},
		"fsx",
	)
	body := string(mustBody(t, resp))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: ct=%q errtype=%q body=%s", resp.StatusCode, resp.Header.Get("Content-Type"), resp.Header.Get("X-Amzn-Errortype"), body)
	}
	if !strings.Contains(body, "FileSystems") {
		t.Fatalf("expected DescribeFileSystems response body to include FileSystems, got %q", body)
	}
}
