package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCleanRoomsStage12CollaborationAndMembershipLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	collabID := "col-stage-001"
	memberID := "mem-stage-001"
	changeRequestID := "cr-stage-001"

	cases := []struct {
		name    string
		method  string
		path    string
		payload string
	}{
		{name: "CreateCollaboration", method: http.MethodPost, path: "/collaborations", payload: `{"collaborationIdentifier":"` + collabID + `","name":"stage-collaboration"}`},
		{name: "GetCollaboration", method: http.MethodGet, path: "/collaborations/" + url.PathEscape(collabID), payload: ``},
		{name: "CreateMembership", method: http.MethodPost, path: "/memberships", payload: `{"membershipIdentifier":"` + memberID + `","collaborationIdentifier":"` + collabID + `"}`},
		{name: "GetMembership", method: http.MethodGet, path: "/memberships/" + url.PathEscape(memberID), payload: ``},
		{name: "ListMemberships", method: http.MethodGet, path: "/memberships", payload: ``},
		{name: "CreateChangeRequest", method: http.MethodPost, path: "/collaborations/" + url.PathEscape(collabID) + "/changeRequests", payload: `{"changeRequestIdentifier":"` + changeRequestID + `"}`},
		{name: "GetChangeRequest", method: http.MethodGet, path: "/collaborations/" + url.PathEscape(collabID) + "/changeRequests/" + url.PathEscape(changeRequestID), payload: ``},
		{name: "ListChangeRequests", method: http.MethodGet, path: "/collaborations/" + url.PathEscape(collabID) + "/changeRequests", payload: ``},
		{name: "UpdateChangeRequest", method: http.MethodPatch, path: "/collaborations/" + url.PathEscape(collabID) + "/changeRequests/" + url.PathEscape(changeRequestID), payload: `{"status":"APPROVED"}`},
		{name: "ListMembers", method: http.MethodGet, path: "/collaborations/" + url.PathEscape(collabID) + "/members", payload: ``},
	}

	for _, tc := range cases {
		var body []byte
		if strings.TrimSpace(tc.payload) != "" {
			body = []byte(tc.payload)
		}
		resp := cleanRoomsRequest(t, ts, tc.method, tc.path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, respBody)
		}
		if strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, respBody)
		}
	}
}

func TestCleanRoomsStage34ConfiguredTableTemplateAndQueryLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	membershipID := "mem-stage-002"
	tableID := "tbl-stage-001"
	associationID := "cta-stage-001"
	templateID := "at-stage-001"
	queryID := "pq-stage-001"
	jobID := "pj-stage-001"
	privacyID := "pbt-stage-001"

	cases := []struct {
		name    string
		method  string
		path    string
		payload string
	}{
		{name: "CreateMembership", method: http.MethodPost, path: "/memberships", payload: `{"membershipIdentifier":"` + membershipID + `"}`},
		{name: "CreateConfiguredTable", method: http.MethodPost, path: "/configuredTables", payload: `{"configuredTableIdentifier":"` + tableID + `","name":"stage-table"}`},
		{name: "CreateConfiguredTableAnalysisRule", method: http.MethodPost, path: "/configuredTables/" + url.PathEscape(tableID) + "/analysisRule", payload: `{"analysisRuleType":"AGGREGATION"}`},
		{name: "CreateConfiguredTableAssociation", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/configuredTableAssociations", payload: `{"configuredTableAssociationIdentifier":"` + associationID + `","configuredTableIdentifier":"` + tableID + `"}`},
		{name: "CreateConfiguredTableAssociationAnalysisRule", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/configuredTableAssociations/" + url.PathEscape(associationID) + "/analysisRule", payload: `{"analysisRuleType":"AGGREGATION"}`},
		{name: "CreateAnalysisTemplate", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/analysistemplates", payload: `{"analysisTemplateIdentifier":"` + templateID + `","name":"stage-template"}`},
		{name: "GetAnalysisTemplate", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/analysistemplates/" + url.PathEscape(templateID), payload: ``},
		{name: "ListAnalysisTemplates", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/analysistemplates", payload: ``},
		{name: "CreatePrivacyBudgetTemplate", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/privacybudgettemplates", payload: `{"privacyBudgetTemplateIdentifier":"` + privacyID + `"}`},
		{name: "ListPrivacyBudgetTemplates", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/privacybudgettemplates", payload: ``},
		{name: "StartProtectedQuery", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/protectedQueries", payload: `{"protectedQueryIdentifier":"` + queryID + `"}`},
		{name: "GetProtectedQuery", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/protectedQueries/" + url.PathEscape(queryID), payload: ``},
		{name: "ListProtectedQueries", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/protectedQueries", payload: ``},
		{name: "StartProtectedJob", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/protectedJobs", payload: `{"protectedJobIdentifier":"` + jobID + `"}`},
		{name: "GetProtectedJob", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/protectedJobs/" + url.PathEscape(jobID), payload: ``},
		{name: "ListProtectedJobs", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/protectedJobs", payload: ``},
		{name: "PreviewPrivacyImpact", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/previewprivacyimpact", payload: `{"privacyBudgetType":"DIFFERENTIAL_PRIVACY"}`},
	}

	for _, tc := range cases {
		var body []byte
		if strings.TrimSpace(tc.payload) != "" {
			body = []byte(tc.payload)
		}
		resp := cleanRoomsRequest(t, ts, tc.method, tc.path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, respBody)
		}
		if strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, respBody)
		}
	}
}

func TestCleanRoomsStage56SchemaTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	membershipID := "mem-stage-003"
	idMappingID := "imt-stage-001"
	idNamespaceID := "ina-stage-001"
	resourceARN := "arn:aws:cleanrooms:us-east-1:123456789012:collaboration/col-stage-003"
	escapedResourceARN := url.PathEscape(resourceARN)

	cases := []struct {
		name    string
		method  string
		path    string
		payload string
	}{
		{name: "CreateMembership", method: http.MethodPost, path: "/memberships", payload: `{"membershipIdentifier":"` + membershipID + `"}`},
		{name: "BatchGetSchema", method: http.MethodPost, path: "/collaborations/col-stage-003/batch-schema", payload: `{"names":["orders"]}`},
		{name: "BatchGetSchemaAnalysisRule", method: http.MethodPost, path: "/collaborations/col-stage-003/batch-schema-analysis-rule", payload: `{"names":["orders"]}`},
		{name: "GetSchema", method: http.MethodGet, path: "/collaborations/col-stage-003/schemas/orders", payload: ``},
		{name: "GetSchemaAnalysisRule", method: http.MethodGet, path: "/collaborations/col-stage-003/schemas/orders/analysisRule/AGGREGATION", payload: ``},
		{name: "CreateIdMappingTable", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/idmappingtables", payload: `{"idMappingTableIdentifier":"` + idMappingID + `"}`},
		{name: "PopulateIdMappingTable", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/idmappingtables/" + url.PathEscape(idMappingID) + "/populate", payload: `{}`},
		{name: "CreateIdNamespaceAssociation", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/idnamespaceassociations", payload: `{"idNamespaceAssociationIdentifier":"` + idNamespaceID + `"}`},
		{name: "GetIdNamespaceAssociation", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/idnamespaceassociations/" + url.PathEscape(idNamespaceID), payload: ``},
		{name: "TagResource", method: http.MethodPost, path: "/tags/" + escapedResourceARN, payload: `{"tags":{"env":"stage","owner":"qa"}}`},
		{name: "ListTagsForResource", method: http.MethodGet, path: "/tags/" + escapedResourceARN, payload: ``},
		{name: "UntagResource", method: http.MethodDelete, path: "/tags/" + escapedResourceARN + "?tagKeys=owner", payload: ``},
		{name: "ListTagsForResourceAfterDelete", method: http.MethodGet, path: "/tags/" + escapedResourceARN, payload: ``},
	}

	for _, tc := range cases {
		var body []byte
		if strings.TrimSpace(tc.payload) != "" {
			body = []byte(tc.payload)
		}
		resp := cleanRoomsRequest(t, ts, tc.method, tc.path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, respBody)
		}
		if strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, respBody)
		}
		if tc.name == "ListTagsForResource" && !strings.Contains(respBody, "owner") {
			t.Fatalf("expected owner tag in list response, got %s", respBody)
		}
		if tc.name == "ListTagsForResourceAfterDelete" && strings.Contains(respBody, "owner") {
			t.Fatalf("expected owner tag removed, got %s", respBody)
		}
	}

	resp := cleanRoomsRequest(t, ts, http.MethodGet, "/cleanrooms/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/collaborations",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"cleanrooms",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	idempotentCollaboration := "col-stage-idempotent-001"
	resp = cleanRoomsRequest(
		t,
		ts,
		http.MethodPost,
		"/collaborations",
		[]byte(`{"collaborationIdentifier":"`+idempotentCollaboration+`","name":"first"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	resp = cleanRoomsRequest(
		t,
		ts,
		http.MethodPost,
		"/collaborations",
		[]byte(`{"collaborationIdentifier":"`+idempotentCollaboration+`","name":"second"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	resp = cleanRoomsRequest(t, ts, http.MethodGet, "/collaborations/"+url.PathEscape(idempotentCollaboration), nil)
	assertStatus(t, resp, http.StatusOK)
}
