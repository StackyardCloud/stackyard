package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRecaptchaenterpriseRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRecaptchaenterpriseContractServer(t)

	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/keys?keyId=site-key-1", []byte(`{"key":{"displayName":"Stackyard Key"}}`), "keys/site-key-1")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/keys?pageSize=1", nil, "\"keys\"")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/keys/site-key-1", nil, "keys/site-key-1")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/keys/site-key-1?updateMask=display_name", []byte(`{"key":{"name":"projects/stackyard/keys/site-key-1","displayName":"Updated"}}`), "Updated")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/keys/site-key-1", nil, "\"deleted\"")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/keys/site-key-1:migrate", []byte(`{}`), "migrated")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/keys/site-key-1:retrieveLegacySecretKey", nil, "legacy-secret-site-key-1")

	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/assessments", []byte(`{"assessment":{"event":{"token":"stackyard-token","siteKey":"projects/stackyard/keys/site-key-1"}}}`), "assessments/")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/assessments/assessment-1:annotate", []byte(`{"annotation":"LEGITIMATE","reasons":["CHARGEBACK"]}`), "LEGITIMATE")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/keys/site-key-1/metrics", nil, "scoreMetrics")

	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/firewallpolicies?firewallPolicyId=policy-1", []byte(`{"firewallPolicy":{"condition":"true"}}`), "firewallpolicies/policy-1")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/firewallpolicies?pageSize=1", nil, "firewallPolicies")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/firewallpolicies/policy-1", nil, "firewallpolicies/policy-1")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/firewallpolicies/policy-1?updateMask=condition", []byte(`{"firewallPolicy":{"name":"projects/stackyard/firewallpolicies/policy-1","condition":"request.ip=='203.0.113.1'"}}`), "firewallpolicies/policy-1")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/firewallpolicies/policy-1", nil, "\"deleted\"")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/firewallpolicies:reorder", []byte(`{"names":["projects/stackyard/firewallpolicies/policy-1"]}`), "firewallPolicies")

	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/keys/site-key-1/ipOverrides?pageSize=1", nil, "ipOverrides")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/keys/site-key-1:addIpOverride", []byte(`{"ipOverrideData":{"ip":"203.0.113.10","overrideType":"ALLOW"}}`), "203.0.113.10")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/keys/site-key-1:removeIpOverride", []byte(`{"ipOverrideData":{"ip":"203.0.113.10","overrideType":"ALLOW"}}`), "203.0.113.10")

	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/relatedaccountgroups?pageSize=1", nil, "relatedAccountGroups")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/relatedaccountgroups/group-1/memberships?pageSize=1", nil, "relatedAccountGroupMemberships")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/relatedaccountgroupmemberships:search?accountId=user-1&pageSize=1", nil, "user-1")
	assertGCPRecaptchaenterpriseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/projectmetadata", nil, "ENTERPRISE")
}

func TestGCPRecaptchaenterpriseRouter_ListKeysInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRecaptchaenterpriseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/keys?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "recaptchaenterprise",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recaptchaenterprise router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecaptchaenterpriseRouter_CreateKeyRequiresDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPRecaptchaenterpriseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/keys", []byte(`{"key":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recaptchaenterprise",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recaptchaenterprise router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecaptchaenterpriseRouter_UpdateKeyNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPRecaptchaenterpriseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/keys/site-key-1?updateMask=display_name", []byte(`{"key":{"name":"projects/stackyard/keys/site-key-2","displayName":"Updated"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recaptchaenterprise",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recaptchaenterprise router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecaptchaenterpriseRouter_SearchMembershipsRequiresAccountID(t *testing.T) {
	t.Parallel()

	ts := newGCPRecaptchaenterpriseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/relatedaccountgroupmemberships:search", nil, map[string]string{
		"X-Stackyard-GCP-Service": "recaptchaenterprise",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recaptchaenterprise router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecaptchaenterpriseRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/recaptchaenterprise?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp recaptchaenterprise contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "recaptchaenterprise" {
		t.Fatalf("expected service=recaptchaenterprise, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPRecaptchaenterpriseContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRecaptchaenterpriseSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "recaptchaenterprise",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp recaptchaenterprise router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
