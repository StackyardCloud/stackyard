package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
)

func TestAzureBlobStoresMetadataAndETag(t *testing.T) {
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

	resp := providerContractRequest(t, ts, http.MethodPut, "/azure/devstoreaccount1/artifacts?restype=container", nil, authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating container, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	putHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
		"x-ms-meta-env": "test",
		"x-ms-meta-app": "stackyard",
	}
	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/devstoreaccount1/artifacts/releases/v1.json", []byte(`{"version":"v1"}`), putHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 putting blob, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	etag := strings.TrimSpace(resp.Header.Get("ETag"))
	if etag == "" {
		t.Fatalf("expected ETag header on put response")
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/artifacts/releases/v1.json", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 reading blob, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := strings.TrimSpace(resp.Header.Get("ETag")); got != etag {
		t.Fatalf("expected ETag %q, got %q", etag, got)
	}
	if got := strings.TrimSpace(resp.Header.Get("x-ms-meta-env")); got != "test" {
		t.Fatalf("expected x-ms-meta-env test, got %q", got)
	}
	if got := strings.TrimSpace(resp.Header.Get("x-ms-meta-app")); got != "stackyard" {
		t.Fatalf("expected x-ms-meta-app stackyard, got %q", got)
	}
	if got := strings.TrimSpace(resp.Header.Get("Content-Type")); !strings.HasPrefix(strings.ToLower(got), "application/json") {
		t.Fatalf("expected content-type application/json, got %q", got)
	}
	if got := string(providerContractBody(t, resp)); got != `{"version":"v1"}` {
		t.Fatalf("unexpected blob payload %q", got)
	}
}

func TestAzureBlobConditionalHeaders(t *testing.T) {
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

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/devstoreaccount1/ingress/orders/state.json", []byte(`{"state":"ready"}`), authHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 putting blob, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	etag := strings.TrimSpace(resp.Header.Get("ETag"))
	if etag == "" {
		t.Fatalf("expected non-empty ETag after put")
	}
	_ = providerContractBody(t, resp)

	noneMatchHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"If-None-Match": etag,
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/ingress/orders/state.json", nil, noneMatchHeaders)
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected 304 for If-None-Match hit, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	ifMatchHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"If-Match":      `"stale-etag"`,
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/ingress/orders/state.json", nil, ifMatchHeaders)
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 for If-Match mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if body["error"] != "ConditionNotMet" {
		t.Fatalf("expected ConditionNotMet error, got %#v", body["error"])
	}
}

func TestAzurePaginationForContainersAndBlobs(t *testing.T) {
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

	for _, container := range []string{"alpha", "bravo", "charlie"} {
		resp := providerContractRequest(t, ts, http.MethodPut, "/azure/devstoreaccount1/"+container+"?restype=container", nil, authHeaders)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 creating %s, got %d body=%s", container, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		_ = providerContractBody(t, resp)
	}

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/?comp=list&maxresults=2", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing containers page one, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	type containerEntry struct {
		Name string `xml:"Name"`
	}
	type containerEnvelope struct {
		Containers struct {
			Container []containerEntry `xml:"Container"`
		} `xml:"Containers"`
		NextMarker string `xml:"NextMarker"`
	}
	var pageOne containerEnvelope
	if err := xml.Unmarshal(providerContractBody(t, resp), &pageOne); err != nil {
		t.Fatalf("unmarshal container page one: %v", err)
	}
	if len(pageOne.Containers.Container) != 2 {
		t.Fatalf("expected 2 containers on first page, got %d", len(pageOne.Containers.Container))
	}
	if strings.TrimSpace(pageOne.NextMarker) == "" {
		t.Fatalf("expected non-empty next marker for first page")
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/?comp=list&maxresults=2&marker="+pageOne.NextMarker, nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing containers page two, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	var pageTwo containerEnvelope
	if err := xml.Unmarshal(providerContractBody(t, resp), &pageTwo); err != nil {
		t.Fatalf("unmarshal container page two: %v", err)
	}
	if len(pageTwo.Containers.Container) != 1 {
		t.Fatalf("expected 1 container on second page, got %d", len(pageTwo.Containers.Container))
	}

	for _, blob := range []string{"a.json", "b.json", "c.json"} {
		resp := providerContractRequest(t, ts, http.MethodPut, "/azure/devstoreaccount1/alpha/"+blob, []byte(`{}`), authHeaders)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 putting blob %s, got %d body=%s", blob, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		_ = providerContractBody(t, resp)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/devstoreaccount1/alpha?restype=container&comp=list&maxresults=2", nil, authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing blobs page one, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	type blobEntry struct {
		Name string `xml:"Name"`
	}
	type blobEnvelope struct {
		Blobs struct {
			Blob []blobEntry `xml:"Blob"`
		} `xml:"Blobs"`
		NextMarker string `xml:"NextMarker"`
	}
	var blobPageOne blobEnvelope
	if err := xml.Unmarshal(providerContractBody(t, resp), &blobPageOne); err != nil {
		t.Fatalf("unmarshal blob page one: %v", err)
	}
	if len(blobPageOne.Blobs.Blob) != 2 {
		t.Fatalf("expected 2 blobs on first page, got %d", len(blobPageOne.Blobs.Blob))
	}
	if strings.TrimSpace(blobPageOne.NextMarker) == "" {
		t.Fatalf("expected non-empty blob next marker")
	}
}
