package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPStorageRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)

	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"stackyard-bucket","location":"US","storageClass":"STANDARD"}`), `"name":"stackyard-bucket"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket", nil, `"kind":"storage#bucket"`)
	assertGCPStorageSuccess(t, ts, http.MethodPatch, "/gcp/storage/v1/b/stackyard-bucket", []byte(`{"storageClass":"NEARLINE","versioning":{"enabled":true}}`), `"storageClass":"NEARLINE"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b?maxResults=10", nil, `"items":[`)

	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/upload/storage/v1/b/stackyard-bucket/o?uploadType=media&name=source.txt", []byte("hello"), `"name":"source.txt"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/o?maxResults=10", nil, `"kind":"storage#objects"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt", nil, `"name":"source.txt"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt?alt=media", nil, "hello")
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/download/storage/v1/b/stackyard-bucket/o/source.txt", nil, "hello")
	assertGCPStorageSuccess(t, ts, http.MethodPatch, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt", []byte(`{"contentType":"text/plain","metadata":{"env":"test"}}`), `"contentType":"text/plain"`)

	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt/copyTo/b/stackyard-bucket/o/copied.txt", nil, `"name":"copied.txt"`)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt/rewriteTo/b/stackyard-bucket/o/rewritten.txt", nil, `"name":"rewritten.txt"`)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/o/composed.txt/compose", []byte(`{"sourceObjects":[{"name":"source.txt"},{"name":"copied.txt"}]}`), `"name":"composed.txt"`)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/o/copied.txt/moveTo/o/moved.txt", nil, `"name":"moved.txt"`)
	assertGCPStorageSuccess(t, ts, http.MethodDelete, "/gcp/storage/v1/b/stackyard-bucket/o/moved.txt", nil, `{}`)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/o/moved.txt/restore", nil, `"name":"moved.txt"`)

	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/acl", []byte(`{"entity":"allUsers","role":"READER"}`), `"entity":"allUsers"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/acl", nil, `"items":[`)
	assertGCPStorageSuccess(t, ts, http.MethodDelete, "/gcp/storage/v1/b/stackyard-bucket/acl/allUsers", nil, `{}`)

	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt/acl", []byte(`{"entity":"allUsers","role":"READER"}`), `"entity":"allUsers"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt/acl", nil, `"items":[`)
	assertGCPStorageSuccess(t, ts, http.MethodDelete, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt/acl/allUsers", nil, `{}`)

	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/iam", nil, `"bindings":[`)
	assertGCPStorageSuccess(t, ts, http.MethodPut, "/gcp/storage/v1/b/stackyard-bucket/iam", []byte(`{"version":1,"bindings":[{"role":"roles/storage.admin","members":["user:stackyard@example.com"]}]}`), `"roles/storage.admin"`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/iam/testPermissions?permissions=storage.buckets.get,storage.objects.get", nil, `"permissions":[`)

	addNotificationResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/notificationConfigs", []byte(`{"topic":"projects/stackyard/topics/storage-events","payload_format":"JSON_API_V1"}`), gcpStorageHeaders(true))
	if addNotificationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from add notification, got %d body=%s", addNotificationResp.StatusCode, string(providerContractBody(t, addNotificationResp)))
	}
	notification := providerContractJSONMap(t, addNotificationResp)
	notificationID, _ := notification["id"].(string)
	if notificationID == "" {
		t.Fatalf("expected notification id, got %#v", notification["id"])
	}
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/notificationConfigs", nil, `"items":[`)
	assertGCPStorageSuccess(t, ts, http.MethodDelete, "/gcp/storage/v1/b/stackyard-bucket/notificationConfigs/"+notificationID, nil, `{}`)

	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/projects/stackyard/serviceAccount", nil, `"email_address":"service-stackyard@stackyard.iam.gserviceaccount.com"`)

	hmacCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/projects/stackyard/hmacKeys", []byte(`{"serviceAccountEmail":"service-stackyard@stackyard.iam.gserviceaccount.com"}`), gcpStorageHeaders(true))
	if hmacCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from create hmac key, got %d body=%s", hmacCreateResp.StatusCode, string(providerContractBody(t, hmacCreateResp)))
	}
	hmacCreated := providerContractJSONMap(t, hmacCreateResp)
	metadata, _ := hmacCreated["metadata"].(map[string]any)
	accessID, _ := metadata["accessId"].(string)
	if accessID == "" {
		t.Fatalf("expected metadata.accessId, got %#v", metadata["accessId"])
	}
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/projects/stackyard/hmacKeys?maxResults=10", nil, `"items":[`)
	assertGCPStorageSuccess(t, ts, http.MethodGet, "/gcp/storage/v1/projects/stackyard/hmacKeys/"+accessID, nil, `"accessId":"`+accessID+`"`)
	assertGCPStorageSuccess(t, ts, http.MethodPut, "/gcp/storage/v1/projects/stackyard/hmacKeys/"+accessID, []byte(`{"state":"INACTIVE"}`), `"state":"INACTIVE"`)
	assertGCPStorageSuccess(t, ts, http.MethodDelete, "/gcp/storage/v1/projects/stackyard/hmacKeys/"+accessID, nil, `{}`)
}

func TestGCPStorageRouter_CreateBucketInvalidName(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"InvalidBucket"}`), gcpStorageHeaders(true))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storage create bucket, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageRouter_UploadRequiresObjectName(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"stackyard-bucket"}`), `"name":"stackyard-bucket"`)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/upload/storage/v1/b/stackyard-bucket/o?uploadType=media", []byte("hello"), gcpStorageHeaders(false))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storage upload, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageRouter_ListObjectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"stackyard-bucket"}`), `"name":"stackyard-bucket"`)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/o?pageToken=bad", nil, gcpStorageHeaders(false))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storage list objects, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageRouter_GetObjectMissingBucket(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/storage/v1/b/missing-bucket/o/missing.txt", nil, gcpStorageHeaders(false))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp storage get object, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageRouter_RestoreRequiresDeletedObject(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"stackyard-bucket"}`), `"name":"stackyard-bucket"`)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/upload/storage/v1/b/stackyard-bucket/o?uploadType=media&name=source.txt", []byte("hello"), `"name":"source.txt"`)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/b/stackyard-bucket/o/source.txt/restore", nil, gcpStorageHeaders(false))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storage restore, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageRouter_UpdateHMACKeyInvalidState(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/projects/stackyard/hmacKeys", []byte(`{"serviceAccountEmail":"service-stackyard@stackyard.iam.gserviceaccount.com"}`), gcpStorageHeaders(true))
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from create hmac key, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	created := providerContractJSONMap(t, createResp)
	metadata, _ := created["metadata"].(map[string]any)
	accessID, _ := metadata["accessId"].(string)
	if accessID == "" {
		t.Fatalf("expected metadata.accessId, got %#v", metadata["accessId"])
	}

	resp := providerContractRequest(t, ts, http.MethodPut, "/gcp/storage/v1/projects/stackyard/hmacKeys/"+accessID, []byte(`{"state":"BROKEN"}`), gcpStorageHeaders(true))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storage update hmac key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"stackyard-bucket","location":"US","storageClass":"STANDARD"}`), `"name":"stackyard-bucket"`)
	assertGCPStorageSuccess(t, ts, http.MethodPost, "/gcp/upload/storage/v1/b/stackyard-bucket/o?uploadType=media&name=shape.txt", []byte("shape"), `"name":"shape.txt"`)

	getBucketResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket", nil, gcpStorageHeaders(false))
	if getBucketResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storage get bucket, got %d body=%s", getBucketResp.StatusCode, string(providerContractBody(t, getBucketResp)))
	}
	bucket := providerContractJSONMap(t, getBucketResp)
	if _, ok := bucket["name"].(string); !ok {
		t.Fatalf("expected string name field, got %#v", bucket["name"])
	}
	if _, ok := bucket["metageneration"].(string); !ok {
		t.Fatalf("expected string metageneration field, got %#v", bucket["metageneration"])
	}

	getObjectResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/storage/v1/b/stackyard-bucket/o/shape.txt", nil, gcpStorageHeaders(false))
	if getObjectResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storage get object, got %d body=%s", getObjectResp.StatusCode, string(providerContractBody(t, getObjectResp)))
	}
	object := providerContractJSONMap(t, getObjectResp)
	if _, ok := object["bucket"].(string); !ok {
		t.Fatalf("expected string bucket field, got %#v", object["bucket"])
	}
	if _, ok := object["generation"].(string); !ok {
		t.Fatalf("expected string generation field, got %#v", object["generation"])
	}
	if _, ok := object["md5Hash"].(string); !ok {
		t.Fatalf("expected string md5Hash field, got %#v", object["md5Hash"])
	}
}

func TestGCPStorageRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/storage/v1/b?stackyard_contract_probe=1&typedSuccess=1", nil, gcpStorageHeaders(false))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storage contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "storage" {
		t.Fatalf("expected service=storage, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["bucket"].(string); !ok {
		t.Fatalf("expected typed bucket field in contract probe response, got %#v", body["bucket"])
	}
}

func newGCPStorageContractServer(t *testing.T) *httptest.Server {
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

func gcpStorageHeaders(withJSON bool) map[string]string {
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storage-apiv1",
	}
	if withJSON {
		headers["Content-Type"] = "application/json"
	}
	return headers
}

func assertGCPStorageSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := gcpStorageHeaders(payload != nil)
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storage router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
