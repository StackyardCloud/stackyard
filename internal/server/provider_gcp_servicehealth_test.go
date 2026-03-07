package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPServiceHealthRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)

	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global", nil, "projects/stackyard/locations/global")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events?pageSize=1&view=EVENT_VIEW_FULL", nil, "events")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events/event-1", nil, "events/event-1")

	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations?pageSize=1", nil, "locations")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global", nil, "organizations/123456789/locations/global")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationEvents?pageSize=1&view=ORGANIZATION_EVENT_VIEW_FULL", nil, "organizationEvents")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationEvents/org-event-1", nil, "organizationEvents/org-event-1")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationImpacts?pageSize=1", nil, "organizationImpacts")
	assertGCPServiceHealthSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationImpacts/impact-1", nil, "organizationImpacts/impact-1")
}

func TestGCPServiceHealthRouter_ListEventsRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events?pageSize=0", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicehealth router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceHealthRouter_ListEventsRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events?pageToken=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicehealth router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceHealthRouter_ListEventsRejectsInvalidView(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events?view=BAD_VIEW", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicehealth router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceHealthRouter_ListEventsRejectsMalformedFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events?filter=category==INCIDENT", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicehealth router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceHealthRouter_ListOrganizationEventsRejectsInvalidView(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationEvents?view=EVENT_VIEW_BASIC", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicehealth router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceHealthRouter_ListOrganizationImpactsRejectsMalformedFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationImpacts?filter=events!!bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicehealth router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceHealthRouter_ListOrganizationImpactsRejectsOutOfRangePageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationImpacts?pageToken=99", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicehealth router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceHealthRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceHealthContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	}

	eventResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events/event-1", nil, headers)
	if eventResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicehealth get event, got %d body=%s", eventResp.StatusCode, string(providerContractBody(t, eventResp)))
	}
	eventBody := providerContractJSONMap(t, eventResp)
	if _, ok := eventBody["name"].(string); !ok {
		t.Fatalf("expected event.name string, got %#v", eventBody["name"])
	}
	if _, ok := eventBody["category"].(string); !ok {
		t.Fatalf("expected event.category string, got %#v", eventBody["category"])
	}
	eventImpacts, ok := eventBody["eventImpacts"].([]any)
	if !ok || len(eventImpacts) == 0 {
		t.Fatalf("expected eventImpacts array, got %#v", eventBody["eventImpacts"])
	}
	firstImpact, _ := eventImpacts[0].(map[string]any)
	if _, ok := firstImpact["product"].(map[string]any); !ok {
		t.Fatalf("expected eventImpacts[0].product object, got %#v", firstImpact["product"])
	}
	if _, ok := eventBody["updates"].([]any); !ok {
		t.Fatalf("expected event.updates array, got %#v", eventBody["updates"])
	}

	orgEventResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationEvents/org-event-1", nil, headers)
	if orgEventResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicehealth get organization event, got %d body=%s", orgEventResp.StatusCode, string(providerContractBody(t, orgEventResp)))
	}
	orgEventBody := providerContractJSONMap(t, orgEventResp)
	if _, ok := orgEventBody["name"].(string); !ok {
		t.Fatalf("expected organizationEvent.name string, got %#v", orgEventBody["name"])
	}
	if _, ok := orgEventBody["eventImpacts"].([]any); !ok {
		t.Fatalf("expected organizationEvent.eventImpacts array, got %#v", orgEventBody["eventImpacts"])
	}

	impactResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/locations/global/organizationImpacts/impact-1", nil, headers)
	if impactResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicehealth get organization impact, got %d body=%s", impactResp.StatusCode, string(providerContractBody(t, impactResp)))
	}
	impactBody := providerContractJSONMap(t, impactResp)
	if _, ok := impactBody["events"].([]any); !ok {
		t.Fatalf("expected organizationImpact.events array, got %#v", impactBody["events"])
	}
	if _, ok := impactBody["asset"].(map[string]any); !ok {
		t.Fatalf("expected organizationImpact.asset object, got %#v", impactBody["asset"])
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/events?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicehealth list events, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	if _, ok := listBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected list nextPageToken string, got %#v", listBody["nextPageToken"])
	}
	if _, ok := listBody["unreachable"].([]any); !ok {
		t.Fatalf("expected list unreachable array, got %#v", listBody["unreachable"])
	}
}

func TestGCPServiceHealthRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/servicehealth?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicehealth contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "servicehealth" {
		t.Fatalf("expected service=servicehealth, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPServiceHealthContractServer(t *testing.T) *httptest.Server {
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

func assertGCPServiceHealthSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicehealth",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicehealth router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
