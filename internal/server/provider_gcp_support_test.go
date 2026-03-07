package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSupportRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	caseName := "projects/stackyard/cases/case-open-1"
	parent := "projects/stackyard"

	assertGCPSupportSuccess(t, ts, http.MethodGet, "/gcp/v2/"+caseName, nil, caseName)
	assertGCPSupportSuccess(t, ts, http.MethodGet, "/gcp/v2/"+parent+"/cases?pageSize=1", nil, `"cases"`)
	assertGCPSupportSuccess(t, ts, http.MethodGet, "/gcp/v2/"+parent+"/cases:search?pageSize=1&query=state=OPEN", nil, `"cases"`)
	assertGCPSupportSuccess(t, ts, http.MethodPost, "/gcp/v2/"+parent+"/cases", []byte(`{
		"displayName": "Broken VM",
		"description": "Instance does not boot",
		"classification": {"id": "technical-issue/compute-engine"},
		"priority": "P2",
		"testCase": true
	}`), `"state":"OPEN"`)
	assertGCPSupportSuccess(t, ts, http.MethodPatch, "/gcp/v2/"+caseName+"?updateMask=display_name,priority", []byte(`{
		"name": "`+caseName+`",
		"displayName": "Updated display",
		"priority": "P1"
	}`), `"displayName":"Updated display"`)
	assertGCPSupportSuccess(t, ts, http.MethodPost, "/gcp/v2/"+caseName+":escalate", []byte(`{
		"name": "`+caseName+`",
		"escalation": {
			"reason": "TECHNICAL_EXPERTISE",
			"justification": "Sev1 production outage"
		}
	}`), `"escalated":true`)
	assertGCPSupportSuccess(t, ts, http.MethodPost, "/gcp/v2/"+caseName+":close", []byte(`{"name":"`+caseName+`"}`), `"state":"CLOSED"`)
	assertGCPSupportSuccess(t, ts, http.MethodGet, "/gcp/v2/caseClassifications:search?pageSize=1", nil, `"caseClassifications"`)
	assertGCPSupportSuccess(t, ts, http.MethodGet, "/gcp/v2/"+caseName+"/comments?pageSize=1", nil, `"comments"`)
	assertGCPSupportSuccess(t, ts, http.MethodPost, "/gcp/v2/"+caseName+"/comments", []byte(`{"body":"Any update from support?"}`), `"Any update from support?"`)
	assertGCPSupportSuccess(t, ts, http.MethodGet, "/gcp/v2/"+caseName+"/attachments?pageSize=1", nil, `"attachments"`)
}

func TestGCPSupportRouter_GetCaseRejectsInvalidName(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/bad*name", nil, map[string]string{
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support get case, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_ListCasesRejectsInvalidFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases?filter=state==OPEN", nil, map[string]string{
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support list cases, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_CreateCaseRequiresFields(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/cases", []byte(`{"displayName":"only name"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support create case, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_UpdateCaseRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	caseName := "projects/stackyard/cases/case-open-1"
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v2/"+caseName, []byte(`{"name":"`+caseName+`"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support update case, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_UpdateCaseRejectsUnsupportedMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	caseName := "projects/stackyard/cases/case-open-1"
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v2/"+caseName+"?updateMask=description", []byte(`{"name":"`+caseName+`","description":"new"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support update case, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_EscalateRequiresEscalation(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/cases/case-open-1:escalate", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support escalate case, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_EscalateClosedCaseFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/cases/case-closed-1:escalate", []byte(`{
		"escalation": {"reason":"TECHNICAL_EXPERTISE","justification":"urgent"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support escalate closed case, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_CloseCaseAlreadyClosedFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/cases/case-closed-1:close", []byte(`{"name":"projects/stackyard/cases/case-closed-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support close closed case, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_CreateCommentRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/cases/case-open-1/comments", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support create comment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_ListCommentsRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/case-open-1/comments?pageToken=bad-token", nil, map[string]string{
		"X-Stackyard-GCP-Service": "support",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp support list comments, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSupportRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSupportContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "support",
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/cases", []byte(`{
		"displayName": "Cannot access service",
		"description": "Timed out",
		"classification": {"id": "technical-issue/compute-engine", "displayName": "Technical Issue > Compute > Compute Engine"},
		"priority": "P2"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "support",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp support create case, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected case.name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["state"].(string); !ok {
		t.Fatalf("expected case.state string, got %#v", createBody["state"])
	}
	classification, ok := createBody["classification"].(map[string]any)
	if !ok {
		t.Fatalf("expected case.classification object, got %#v", createBody["classification"])
	}
	if _, ok := classification["id"].(string); !ok {
		t.Fatalf("expected classification.id string, got %#v", classification["id"])
	}

	classificationsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/caseClassifications:search?pageSize=1", nil, headers)
	if classificationsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp support search classifications, got %d body=%s", classificationsResp.StatusCode, string(providerContractBody(t, classificationsResp)))
	}
	classificationsBody := providerContractJSONMap(t, classificationsResp)
	classifications, ok := classificationsBody["caseClassifications"].([]any)
	if !ok || len(classifications) == 0 {
		t.Fatalf("expected caseClassifications array, got %#v", classificationsBody["caseClassifications"])
	}
	firstClassification, _ := classifications[0].(map[string]any)
	if _, ok := firstClassification["displayName"].(string); !ok {
		t.Fatalf("expected caseClassifications[0].displayName string, got %#v", firstClassification["displayName"])
	}

	commentsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/case-open-1/comments?pageSize=1", nil, headers)
	if commentsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp support list comments, got %d body=%s", commentsResp.StatusCode, string(providerContractBody(t, commentsResp)))
	}
	commentsBody := providerContractJSONMap(t, commentsResp)
	comments, ok := commentsBody["comments"].([]any)
	if !ok || len(comments) == 0 {
		t.Fatalf("expected comments array, got %#v", commentsBody["comments"])
	}
	firstComment, _ := comments[0].(map[string]any)
	if _, ok := firstComment["body"].(string); !ok {
		t.Fatalf("expected comments[0].body string, got %#v", firstComment["body"])
	}

	attachmentsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/case-open-1/attachments?pageSize=1", nil, headers)
	if attachmentsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp support list attachments, got %d body=%s", attachmentsResp.StatusCode, string(providerContractBody(t, attachmentsResp)))
	}
	attachmentsBody := providerContractJSONMap(t, attachmentsResp)
	attachments, ok := attachmentsBody["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		t.Fatalf("expected attachments array, got %#v", attachmentsBody["attachments"])
	}
	firstAttachment, _ := attachments[0].(map[string]any)
	if _, ok := firstAttachment["mimeType"].(string); !ok {
		t.Fatalf("expected attachments[0].mimeType string, got %#v", firstAttachment["mimeType"])
	}
	if _, ok := firstAttachment["sizeBytes"].(float64); !ok {
		t.Fatalf("expected attachments[0].sizeBytes number, got %#v", firstAttachment["sizeBytes"])
	}
}

func TestGCPSupportRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/cases/case-probe-1?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp support contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "support" {
		t.Fatalf("expected service=support, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSupportContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSupportSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "support",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp support router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
