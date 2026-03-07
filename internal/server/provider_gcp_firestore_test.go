package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPFirestoreRouter_DocumentRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFirestoreContractServer(t)
	assertGCPFirestoreSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/databases/(default)/documents/users?pageSize=1", nil, "documents")
	assertGCPFirestoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents/users", nil, "createTime")
	assertGCPFirestoreSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/databases/(default)/documents/users/u-1", nil, "/documents/users/u-1")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/databases/(default)/documents/users/u-1?updateMask.fieldPaths=displayName", "/documents/users/u-1")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/databases/(default)/documents/users/u-1", "/documents/users/u-1")
}

func TestGCPFirestoreRouter_QueryAndTransactionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFirestoreContractServer(t)
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:batchGet", ":batchGet")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:beginTransaction", ":beginTransaction")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:commit", ":commit")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:rollback", ":rollback")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:runQuery", ":runQuery")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:runAggregationQuery", ":runAggregationQuery")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:partitionQuery", ":partitionQuery")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents/users/u-1:listCollectionIds", ":listCollectionIds")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents:batchWrite", ":batchWrite")
}

func TestGCPFirestoreRouter_OperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFirestoreContractServer(t)
	assertGCPFirestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/databases/(default)/operations/op-1", "/operations/op-1")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/databases/(default)/operations?pageSize=1", "/operations")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/operations/op-1:cancel", ":cancel")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/databases/(default)/operations/op-1", "/operations/op-1")
}

func TestGCPFirestoreRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFirestoreContractServer(t)
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/GetDocument", "Firestore/GetDocument")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/ListDocuments", "Firestore/ListDocuments")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/CreateDocument", "Firestore/CreateDocument")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/RunQuery", "Firestore/RunQuery")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/RunAggregationQuery", "Firestore/RunAggregationQuery")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/PartitionQuery", "Firestore/PartitionQuery")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/BatchWrite", "Firestore/BatchWrite")
	assertGCPFirestoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.firestore.v1.Firestore/Listen", "Firestore/Listen")
}

func TestGCPFirestoreRouter_ListDocumentsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPFirestoreContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/databases/(default)/documents/users?pageSize=oops", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp firestore router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPFirestoreRouter_CreateDocumentInvalidJSONBody(t *testing.T) {
	t.Parallel()

	ts := newGCPFirestoreContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/(default)/documents/users", []byte("{"), nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp firestore router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPFirestoreContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPFirestoreNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp firestore router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPFirestoreSuccess(t *testing.T, ts *httptest.Server, method, path string, body []byte, expectBodyFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, body, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp firestore router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	respBody := string(providerContractBody(t, resp))
	if !strings.Contains(respBody, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, respBody)
	}
}
