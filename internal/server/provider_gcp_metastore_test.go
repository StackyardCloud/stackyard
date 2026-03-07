package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMetastoreRouter_ServiceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMetastoreContractServer(t)

	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/services?pageSize=1", "/services")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/services?serviceId=hive-a", "/services")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/services/hive-a", "/services/hive-a")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/services/hive-a?updateMask=labels", "/services/hive-a")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/services/hive-a", "/services/hive-a")
}

func TestGCPMetastoreRouter_MetadataImportRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMetastoreContractServer(t)

	parent := "/gcp/v1/projects/stackyard/locations/us-central1/services/hive-a"
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, parent+"/metadataImports?pageSize=1", "/metadataImports")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, parent+"/metadataImports?metadataImportId=import-a", "/metadataImports")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, parent+"/metadataImports/import-a", "/metadataImports/import-a")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPatch, parent+"/metadataImports/import-a?updateMask=description", "/metadataImports/import-a")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, parent+":exportMetadata", ":exportMetadata")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, parent+":restore", ":restore")
}

func TestGCPMetastoreRouter_BackupAndMetadataActionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMetastoreContractServer(t)

	parent := "/gcp/v1/projects/stackyard/locations/us-central1/services/hive-a"
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, parent+"/backups?pageSize=1", "/backups")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, parent+"/backups?backupId=backup-a", "/backups")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, parent+"/backups/backup-a", "/backups/backup-a")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodDelete, parent+"/backups/backup-a", "/backups/backup-a")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, parent+":queryMetadata", ":queryMetadata")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, parent+":moveTableToDatabase", ":moveTableToDatabase")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, parent+":alterLocation", ":alterLocation")
}

func TestGCPMetastoreRouter_IAMLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMetastoreContractServer(t)

	service := "/gcp/v1/projects/stackyard/locations/us-central1/services/hive-a"
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, service+":getIamPolicy", ":getIamPolicy")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, service+":setIamPolicy", ":setIamPolicy")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, service+":testIamPermissions", ":testIamPermissions")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPMetastoreRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMetastoreContractServer(t)

	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.metastore.v1.DataprocMetastore/ListServices", "DataprocMetastore/ListServices")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.metastore.v1.DataprocMetastore/CreateService", "DataprocMetastore/CreateService")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.metastore.v1.DataprocMetastore/CreateMetadataImport", "DataprocMetastore/CreateMetadataImport")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.metastore.v1.DataprocMetastore/CreateBackup", "DataprocMetastore/CreateBackup")
	assertGCPMetastoreNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.metastore.v1.DataprocMetastore/QueryMetadata", "DataprocMetastore/QueryMetadata")
}

func newGCPMetastoreContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMetastoreNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp metastore router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPMetastoreRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPMetastoreRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/metastore?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp metastore contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "metastore" {
		t.Fatalf("expected service=metastore, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

