package server

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func ebsPathForOperation(op ebsOperation) string {
	path := op.URI
	path = strings.ReplaceAll(path, "{snapshotId}", "snap-00000000000000001")
	path = strings.ReplaceAll(path, "{secondSnapshotId}", "snap-00000000000000001")
	path = strings.ReplaceAll(path, "{blockIndex}", "0")
	return path
}

func ebsSignedRequest(
	t *testing.T,
	ts *httptest.Server,
	op ebsOperation,
	body []byte,
	headers map[string]string,
	query string,
) *http.Response {
	t.Helper()
	path := ebsPathForOperation(op)
	if strings.TrimSpace(query) != "" {
		path += "?" + strings.TrimPrefix(query, "?")
	}
	return signedRequestWithService(t, op.Method, ts.URL+path, body, headers, "ebs")
}

func TestEBSStage1UnknownRouteReturnsValidationException(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodGet,
		ts.URL+"/snapshots/unknown/route",
		nil,
		nil,
		"ebs",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestEBSStage1AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	expectedStatus := map[string]int{
		"StartSnapshot":      http.StatusCreated,
		"PutSnapshotBlock":   http.StatusCreated,
		"CompleteSnapshot":   http.StatusAccepted,
		"GetSnapshotBlock":   http.StatusOK,
		"ListSnapshotBlocks": http.StatusOK,
		"ListChangedBlocks":  http.StatusOK,
	}

	blockData := []byte("stackyard-ebs-test-block")
	sum := sha256.Sum256(blockData)
	blockChecksum := base64.StdEncoding.EncodeToString(sum[:])

	for _, op := range ebsOperations {
		reqHeaders := map[string]string{}
		reqBody := []byte{}
		query := ""

		switch op.Name {
		case "StartSnapshot":
			reqHeaders["Content-Type"] = "application/json"
			reqBody = []byte(`{"VolumeSize":1,"Description":"stackyard coverage"}`)
		case "PutSnapshotBlock":
			reqBody = blockData
			reqHeaders["Content-Type"] = "application/octet-stream"
			reqHeaders["x-amz-Data-Length"] = strconv.Itoa(len(blockData))
			reqHeaders["x-amz-Checksum"] = blockChecksum
			reqHeaders["x-amz-Checksum-Algorithm"] = "SHA256"
		case "GetSnapshotBlock":
			query = "blockToken=token-snap-00000000000000001-0-seed"
		case "ListChangedBlocks":
			query = "firstSnapshotId=snap-00000000000000001"
		case "CompleteSnapshot":
			reqHeaders["x-amz-ChangedBlocksCount"] = "1"
		}

		resp := ebsSignedRequest(t, ts, op, reqBody, reqHeaders, query)
		body := mustBody(t, resp)

		wantStatus := expectedStatus[op.Name]
		if wantStatus == 0 {
			wantStatus = http.StatusOK
		}
		if resp.StatusCode != wantStatus {
			t.Fatalf("%s returned status=%d, expected %d, body=%q", op.Name, resp.StatusCode, wantStatus, string(body))
		}

		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(string(body), "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%q", op.Name, resp.StatusCode, string(body))
		}

		if op.Name == "GetSnapshotBlock" {
			if strings.TrimSpace(resp.Header.Get("x-amz-Data-Length")) == "" {
				t.Fatalf("GetSnapshotBlock response missing x-amz-Data-Length header")
			}
			if len(body) == 0 {
				t.Fatalf("GetSnapshotBlock response body was empty")
			}
		}
	}
}
