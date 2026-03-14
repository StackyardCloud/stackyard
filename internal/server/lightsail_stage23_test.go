package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLightsailStage23DistributionContractShape(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateDistribution", []byte(`{
		"distributionName":"shape-dist",
		"bundleId":"small_1_0",
		"defaultCacheBehavior":{"behavior":"cache"},
		"origin":{"name":"shape-origin","protocolPolicy":"http-only","regionName":"us-east-1"}
	}`))
	assertStatus(t, resp, http.StatusOK)

	var createOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal CreateDistribution: %v", err)
	}
	distribution, ok := createOut["distribution"].(map[string]any)
	if !ok {
		t.Fatalf("expected distribution object, got %#v", createOut["distribution"])
	}
	if _, exists := distribution["viewerMinimumTlsProtocolVersion"]; exists {
		t.Fatalf("CreateDistribution returned viewerMinimumTlsProtocolVersion: %#v", distribution)
	}
	origin, ok := distribution["origin"].(map[string]any)
	if !ok {
		t.Fatalf("expected origin object, got %#v", distribution["origin"])
	}
	if _, exists := origin["responseTimeout"]; exists {
		t.Fatalf("CreateDistribution returned origin.responseTimeout: %#v", origin)
	}

	resp = lightsailRequest(t, ts, "GetDistributions", []byte(`{"distributionName":"shape-dist"}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal GetDistributions: %v", err)
	}
	distributions, ok := listOut["distributions"].([]any)
	if !ok || len(distributions) != 1 {
		t.Fatalf("expected one distribution, got %#v", listOut["distributions"])
	}
	firstDist, ok := distributions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected distribution object, got %#v", distributions[0])
	}
	if _, exists := firstDist["viewerMinimumTlsProtocolVersion"]; exists {
		t.Fatalf("GetDistributions returned viewerMinimumTlsProtocolVersion: %#v", firstDist)
	}
	firstOrigin, ok := firstDist["origin"].(map[string]any)
	if !ok {
		t.Fatalf("expected origin object, got %#v", firstDist["origin"])
	}
	if _, exists := firstOrigin["responseTimeout"]; exists {
		t.Fatalf("GetDistributions returned origin.responseTimeout: %#v", firstOrigin)
	}

	resp = lightsailRequest(t, ts, "GetDistributionLatestCacheReset", []byte(`{"distributionName":"shape-dist"}`))
	assertStatus(t, resp, http.StatusOK)
	var latestReset map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &latestReset); err != nil {
		t.Fatalf("unmarshal GetDistributionLatestCacheReset: %v", err)
	}
	if _, exists := latestReset["status"]; !exists {
		t.Fatalf("expected status in GetDistributionLatestCacheReset: %#v", latestReset)
	}
	if _, exists := latestReset["createTime"]; !exists {
		t.Fatalf("expected createTime in GetDistributionLatestCacheReset: %#v", latestReset)
	}
}
