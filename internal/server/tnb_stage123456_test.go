package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestTNBStage12PackageAndInstanceLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := tnbRequest(t, ts, http.MethodPost, "/sol/nsd/v1/ns_descriptors", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/nsd/v1/ns_descriptors/nsd-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/nsd/v1/ns_descriptors/nsd-000001/nsd", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPut, "/sol/nsd/v1/ns_descriptors/nsd-000001/nsd_content", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPut, "/sol/nsd/v1/ns_descriptors/nsd-000001/nsd_content/validate", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPatch, "/sol/nsd/v1/ns_descriptors/nsd-000001", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/nsd/v1/ns_descriptors?max_results=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodDelete, "/sol/nsd/v1/ns_descriptors/nsd-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = tnbRequest(t, ts, http.MethodPost, "/sol/vnfpkgm/v1/vnf_packages", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/vnfpkgm/v1/vnf_packages/vnfpkg-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/vnfpkgm/v1/vnf_packages/vnfpkg-000001/vnfd", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPut, "/sol/vnfpkgm/v1/vnf_packages/vnfpkg-000001/package_content", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPut, "/sol/vnfpkgm/v1/vnf_packages/vnfpkg-000001/package_content/validate", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPatch, "/sol/vnfpkgm/v1/vnf_packages/vnfpkg-000001", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/vnfpkgm/v1/vnf_packages?max_results=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodDelete, "/sol/vnfpkgm/v1/vnf_packages/vnfpkg-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = tnbRequest(t, ts, http.MethodPost, "/sol/nslcm/v1/ns_instances", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPost, "/sol/nslcm/v1/ns_instances/ns-instance-000001/instantiate?dry_run=false", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPost, "/sol/nslcm/v1/ns_instances/ns-instance-000001/update", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPost, "/sol/nslcm/v1/ns_instances/ns-instance-000001/terminate", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/nslcm/v1/ns_instances/ns-instance-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/nslcm/v1/ns_instances?max_results=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodDelete, "/sol/nslcm/v1/ns_instances/ns-instance-000001", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestTNBStage34OperationsAndFunctionInstances(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := tnbRequest(t, ts, http.MethodGet, "/sol/nslcm/v1/ns_lcm_op_occs/ns-lcm-op-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/nslcm/v1/ns_lcm_op_occs?max_results=10&nsInstanceId=ns-instance-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodPost, "/sol/nslcm/v1/ns_lcm_op_occs/ns-lcm-op-000001/cancel", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = tnbRequest(t, ts, http.MethodGet, "/sol/vnflcm/v1/vnf_instances/vnf-instance-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/sol/vnflcm/v1/vnf_instances?max_results=10", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestTNBStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := url.PathEscape("arn:aws:tnb:us-east-1:123456789012:nsd/nsd-000001")

	resp := tnbRequest(t, ts, http.MethodPost, "/tags/"+resourceARN, []byte(`{"tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = tnbRequest(t, ts, http.MethodGet, "/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = tnbRequest(t, ts, http.MethodDelete, "/tags/"+resourceARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = tnbRequest(t, ts, http.MethodPost, "/tnb/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/sol/nsd/v1/ns_descriptors",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"tnb",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
