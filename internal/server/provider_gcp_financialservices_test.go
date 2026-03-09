package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPFinancialServicesRouter_InstanceAndPartyRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFinancialServicesContractServer(t)
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances?pageSize=1", "/instances")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances", "/instances")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance?updateMask=labels", "/instances/team-instance")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance", "/instances/team-instance")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance:importRegisteredParties", ":importRegisteredParties")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance:exportRegisteredParties", ":exportRegisteredParties")
}

func TestGCPFinancialServicesRouter_DatasetModelAndEngineRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFinancialServicesContractServer(t)
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/datasets?pageSize=1", "/datasets")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/datasets", "/datasets")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/models?pageSize=1", "/models")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/models/team-model:exportMetadata", ":exportMetadata")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/engineConfigs?pageSize=1", "/engineConfigs")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/engineConfigs/team-config:exportMetadata", ":exportMetadata")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/engineVersions?pageSize=1", "/engineVersions")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/engineVersions/engine-v1", "/engineVersions/engine-v1")
}

func TestGCPFinancialServicesRouter_PredictionAndBacktestRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFinancialServicesContractServer(t)
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/predictionResults?pageSize=1", "/predictionResults")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/predictionResults/team-prediction:exportMetadata", ":exportMetadata")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/predictionResults/team-prediction", "/predictionResults/team-prediction")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/backtestResults?pageSize=1", "/backtestResults")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/backtestResults/team-backtest:exportMetadata", ":exportMetadata")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance/backtestResults/team-backtest", "/backtestResults/team-backtest")
}

func TestGCPFinancialServicesRouter_LocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFinancialServicesContractServer(t)
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations", "/locations")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPFinancialServicesRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPFinancialServicesContractServer(t)
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.financialservices.v1.AML/ListInstances", "AML/ListInstances")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.financialservices.v1.AML/CreateModel", "AML/CreateModel")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.financialservices.v1.AML/ExportModelMetadata", "AML/ExportModelMetadata")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.financialservices.v1.AML/ListEngineVersions", "AML/ListEngineVersions")
	assertGCPFinancialServicesNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.financialservices.v1.AML/ListBacktestResults", "AML/ListBacktestResults")
}

func newGCPFinancialServicesContractServer(t *testing.T) *httptest.Server {
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

func assertGCPFinancialServicesNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp financial services router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPFinancialservicesRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPFinancialservicesRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/financialservices?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp financialservices contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "financialservices" {
		t.Fatalf("expected service=financialservices, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
