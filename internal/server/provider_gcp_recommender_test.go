package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRecommenderRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommenderContractServer(t)

	insightParent := "/gcp/v1/projects/stackyard/locations/us-central1/insightTypes/google.iam.policy.Insight"
	recommenderParent := "/gcp/v1/projects/stackyard/locations/us-central1/recommenders/google.compute.instance.MachineTypeRecommender"
	insightName := insightParent + "/insights/insight-1"
	recommendationName := recommenderParent + "/recommendations/recommendation-1"
	recommenderConfigName := recommenderParent + "/config"
	insightTypeConfigName := insightParent + "/config"

	assertGCPRecommenderSuccess(t, ts, http.MethodGet, insightParent+"/insights?pageSize=1", nil, "insights")
	assertGCPRecommenderSuccess(t, ts, http.MethodGet, insightName, nil, "insight-1")
	assertGCPRecommenderSuccess(t, ts, http.MethodPost, insightName+":markAccepted", []byte(`{"name":"projects/stackyard/locations/us-central1/insightTypes/google.iam.policy.Insight/insights/insight-1","etag":"etag-insight-1","stateMetadata":{"ticket":"INC-100"}}`), "ACCEPTED")

	assertGCPRecommenderSuccess(t, ts, http.MethodGet, recommenderParent+"/recommendations?pageSize=1", nil, "recommendations")
	assertGCPRecommenderSuccess(t, ts, http.MethodGet, recommendationName, nil, "recommendation-1")
	assertGCPRecommenderSuccess(t, ts, http.MethodPost, recommendationName+":markDismissed", []byte(`{"name":"projects/stackyard/locations/us-central1/recommenders/google.compute.instance.MachineTypeRecommender/recommendations/recommendation-1","etag":"etag-recommendation-1"}`), "DISMISSED")
	assertGCPRecommenderSuccess(t, ts, http.MethodPost, recommendationName+":markClaimed", []byte(`{"etag":"etag-recommendation-1"}`), "CLAIMED")
	assertGCPRecommenderSuccess(t, ts, http.MethodPost, recommendationName+":markSucceeded", []byte(`{"etag":"etag-recommendation-1"}`), "SUCCEEDED")
	assertGCPRecommenderSuccess(t, ts, http.MethodPost, recommendationName+":markFailed", []byte(`{"etag":"etag-recommendation-1"}`), "FAILED")

	assertGCPRecommenderSuccess(t, ts, http.MethodGet, recommenderConfigName, nil, "recommenderGenerationConfig")
	assertGCPRecommenderSuccess(t, ts, http.MethodPatch, recommenderConfigName+"?updateMask=displayName", []byte(`{"name":"projects/stackyard/locations/us-central1/recommenders/google.compute.instance.MachineTypeRecommender/config","displayName":"Updated Recommender Config"}`), "Updated Recommender Config")

	assertGCPRecommenderSuccess(t, ts, http.MethodGet, insightTypeConfigName, nil, "insightTypeGenerationConfig")
	assertGCPRecommenderSuccess(t, ts, http.MethodPatch, insightTypeConfigName+"?updateMask=displayName", []byte(`{"name":"projects/stackyard/locations/us-central1/insightTypes/google.iam.policy.Insight/config","displayName":"Updated Insight Config"}`), "Updated Insight Config")
}

func TestGCPRecommenderRouter_ListInsightsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommenderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/insightTypes/google.iam.policy.Insight/insights?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "recommender",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommender router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommenderRouter_MarkRecommendationRequiresEtag(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommenderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/recommenders/google.compute.instance.MachineTypeRecommender/recommendations/recommendation-1:markClaimed", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommender",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommender router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommenderRouter_MarkRecommendationRejectsEtagMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommenderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/recommenders/google.compute.instance.MachineTypeRecommender/recommendations/recommendation-1:markSucceeded", []byte(`{"etag":"wrong-etag"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommender",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommender router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommenderRouter_UpdateConfigNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommenderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/recommenders/google.compute.instance.MachineTypeRecommender/config?updateMask=displayName", []byte(`{"name":"projects/stackyard/locations/us-central1/recommenders/google.compute.instance.Different/config"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommender",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommender router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommenderRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/recommender?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp recommender contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "recommender" {
		t.Fatalf("expected service=recommender, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPRecommenderContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRecommenderSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "recommender",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp recommender router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
