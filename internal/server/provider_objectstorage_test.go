package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newProviderContractServer(t *testing.T, cfg Config) *httptest.Server {
	t.Helper()
	srv := New(cfg)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func providerContractRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, ts.URL+path, reader)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, path, err)
	}
	return resp
}

func providerContractBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	_ = resp.Body.Close()
	return body
}

func providerContractJSONMap(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	body := providerContractBody(t, resp)
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal response JSON: %v (body=%s)", err, string(body))
	}
	return out
}

func TestProviderContract_GCPObjectStorageLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"team-a-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating bucket, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/gcp/upload/storage/v1/b/team-a-bucket/o?uploadType=media&name=orders%2F2026-03-02.json", []byte(`{"orderId":"o-1"}`), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 uploading object, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodGet, "/gcp/storage/v1/b/team-a-bucket/o", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing objects, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listBody := providerContractJSONMap(t, resp)
	itemsRaw, ok := listBody["items"].([]any)
	if !ok || len(itemsRaw) != 1 {
		t.Fatalf("expected one object item, got %#v", listBody["items"])
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/gcp/storage/v1/b/team-a-bucket/o/orders%2F2026-03-02.json?alt=media", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 downloading object, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := string(providerContractBody(t, resp)); got != `{"orderId":"o-1"}` {
		t.Fatalf("unexpected object payload: %q", got)
	}
}

func TestProviderContract_AzureBlobLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})
	authHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	}

	resp := providerContractRequest(t, ts, http.MethodPut, "/azure/devstoreaccount1/ingress?restype=container", nil, authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating container, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/?comp=list", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing containers, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, "<Name>ingress</Name>") {
		t.Fatalf("expected list payload to include container name, got %q", body)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/devstoreaccount1/ingress/domain/orders.json", []byte(`{"event":"created"}`), authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 putting blob, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodHead, "/azure/devstoreaccount1/ingress?restype=container", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading container, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodHead, "/azure/devstoreaccount1/ingress/domain/orders.json", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading blob, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := strings.TrimSpace(string(providerContractBody(t, resp))); got != "" {
		t.Fatalf("expected empty HEAD blob body, got %q", got)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/ingress/domain/orders.json", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting blob, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := string(providerContractBody(t, resp)); got != `{"event":"created"}` {
		t.Fatalf("unexpected blob payload: %q", got)
	}
}

func TestProviderContract_OCIObjectStorageLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerOCI},
		OCIAuthMode: "signature",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
	authHeaders := map[string]string{
		"Date":          "Mon, 02 Mar 2026 16:00:00 GMT",
		"Authorization": `Signature keyId="ocid1.user.oc1..aaaa",algorithm="rsa-sha256",headers="date (request-target) host",signature="abc123"`,
	}

	resp := providerContractRequest(t, ts, http.MethodGet, "/oci/n", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting namespace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if body["value"] != defaultOCINamespace {
		t.Fatalf("expected namespace %q, got %#v", defaultOCINamespace, body["value"])
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/oci/n/"+defaultOCINamespace+"/b/mesh-data", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating OCI bucket, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPut, "/oci/n/"+defaultOCINamespace+"/b/mesh-data/o/projections/domain-a/state.json", []byte(`{"status":"ready"}`), authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 putting OCI object, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodGet, "/oci/n/"+defaultOCINamespace+"/b/mesh-data/o/projections/domain-a/state.json", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting OCI object, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := string(providerContractBody(t, resp)); got != `{"status":"ready"}` {
		t.Fatalf("unexpected OCI object payload: %q", got)
	}
}

func TestProviderContract_AzureRejectsMissingAuthByDefault(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:      "127.0.0.1:0",
		Providers: []string{providerAzure},
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/?comp=list", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing Azure auth, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)
}

func TestProviderContract_OCIRejectsMissingSignatureByDefault(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:      "127.0.0.1:0",
		Providers: []string{providerOCI},
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/oci/n", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing OCI signature, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)
}

func TestProviderContract_GCPRejectsDuplicateBucket(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"dup-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating first bucket, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"dup-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate bucket, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)
}

func TestProviderContract_OCIReturnsNotFoundForMissingObject(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerOCI},
		OCIAuthMode: "signature",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
	authHeaders := map[string]string{
		"Date":          "Mon, 02 Mar 2026 16:00:00 GMT",
		"Authorization": `Signature keyId="ocid1.user.oc1..aaaa",algorithm="rsa-sha256",headers="date (request-target) host",signature="abc123"`,
	}

	resp := providerContractRequest(t, ts, http.MethodPut, "/oci/n/"+defaultOCINamespace+"/b/mesh-data", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating OCI bucket, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodGet, "/oci/n/"+defaultOCINamespace+"/b/mesh-data/o/missing.json", nil, authHeaders)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing OCI object, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)
}

func BenchmarkProviderContract_GCPUploadAndRead(b *testing.B) {
	srv := New(Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	doReq := func(method, path string, body []byte, headers map[string]string) *http.Response {
		b.Helper()
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, ts.URL+path, reader)
		if err != nil {
			b.Fatalf("new request %s %s: %v", method, path, err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			b.Fatalf("request %s %s: %v", method, path, err)
		}
		return resp
	}

	readBody := func(resp *http.Response) []byte {
		b.Helper()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			b.Fatalf("read response body: %v", err)
		}
		_ = resp.Body.Close()
		return body
	}

	resp := doReq(http.MethodPost, "/gcp/storage/v1/b", []byte(`{"name":"bench-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		b.Fatalf("create bench bucket failed: %d", resp.StatusCode)
	}
	_ = readBody(resp)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		putResp := doReq(http.MethodPost, "/gcp/upload/storage/v1/b/bench-bucket/o?uploadType=media&name=bench-object", []byte("x"), nil)
		if putResp.StatusCode != http.StatusOK {
			b.Fatalf("put object failed: %d", putResp.StatusCode)
		}
		_ = readBody(putResp)

		getResp := doReq(http.MethodGet, "/gcp/storage/v1/b/bench-bucket/o/bench-object?alt=media", nil, nil)
		if getResp.StatusCode != http.StatusOK {
			b.Fatalf("get object failed: %d", getResp.StatusCode)
		}
		_ = readBody(getResp)
	}
}
