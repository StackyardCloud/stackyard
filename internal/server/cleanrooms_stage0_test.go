package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func cleanRoomsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "cleanrooms")
}

func TestCleanRoomsStage0CatalogCoverage(t *testing.T) {
	if len(cleanRoomsOperations) != 88 {
		t.Fatalf("expected 88 Clean Rooms actions from docs, got %d", len(cleanRoomsOperations))
	}
	if len(cleanRoomsOperationByName) != len(cleanRoomsOperations) {
		t.Fatalf("expected unique Clean Rooms action names")
	}

	requiredActions := []string{
		"CreateCollaboration",
		"CreateMembership",
		"CreateConfiguredTable",
		"CreateConfiguredTableAssociation",
		"CreateAnalysisTemplate",
		"StartProtectedQuery",
		"StartProtectedJob",
		"TagResource",
		"ListTagsForResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := cleanRoomsOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(cleanRoomsDataTypes) != 195 {
		t.Fatalf("expected 195 Clean Rooms data types from docs, got %d", len(cleanRoomsDataTypes))
	}
	if len(cleanRoomsDataTypeByName) != len(cleanRoomsDataTypes) {
		t.Fatalf("expected unique Clean Rooms data type names")
	}

	requiredTypes := []string{
		"Collaboration",
		"Membership",
		"ConfiguredTable",
		"ConfiguredTableAssociation",
		"AnalysisTemplate",
		"ProtectedQuery",
		"ProtectedJob",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cleanRoomsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCleanRoomsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cleanRoomsRequest(t, ts, http.MethodGet, "/cleanrooms/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCleanRoomsStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cleanRoomsRequest(t, ts, http.MethodGet, "/collaborations", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "collaborationList") {
		t.Fatalf("expected ListCollaborations response to include collaborationList, got %q", body)
	}
}

func TestCleanRoomsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cleanRoomsOperations {
		path := cleanRoomsRenderTestURI(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}
		switch op.Name {
		case "CreateCollaboration":
			body = []byte(`{"name":"stage0-collaboration"}`)
		case "CreateMembership":
			body = []byte(`{"collaborationIdentifier":"col-000001"}`)
		case "CreateConfiguredTable":
			body = []byte(`{"name":"stage0-table"}`)
		case "CreateConfiguredTableAssociation":
			body = []byte(`{"configuredTableIdentifier":"tbl-000001"}`)
		case "CreateAnalysisTemplate":
			body = []byte(`{"name":"stage0-template"}`)
		case "TagResource":
			body = []byte(`{"tags":{"env":"stage0"}}`)
		}

		resp := cleanRoomsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s path=%s", op.Name, resp.StatusCode, respBody, path)
		}
	}
}

var cleanRoomsURIPlaceholderPattern = regexp.MustCompile(`\{([^}]+)\}`)

func cleanRoomsRenderTestURI(uriTemplate string) string {
	return cleanRoomsURIPlaceholderPattern.ReplaceAllStringFunc(uriTemplate, func(raw string) string {
		placeholder := strings.TrimSpace(strings.Trim(raw, "{}"))
		switch strings.ToLower(placeholder) {
		case "collaborationidentifier":
			return "col-000001"
		case "membershipidentifier":
			return "mem-000001"
		case "configuredtableidentifier":
			return "tbl-000001"
		case "configuredtableassociationidentifier":
			return "cta-000001"
		case "analysistemplateidentifier":
			return "at-000001"
		case "analysistemplatearn":
			return url.PathEscape("arn:aws:cleanrooms:us-east-1:123456789012:membership/mem-000001/analysistemplate/at-000001")
		case "analysisruletype":
			return "AGGREGATION"
		case "protectedqueryidentifier":
			return "pq-000001"
		case "protectedjobidentifier":
			return "pj-000001"
		case "privacybudgettemplateidentifier":
			return "pbt-000001"
		case "configuredaudiencemodelassociationidentifier":
			return "ama-000001"
		case "idmappingtableidentifier":
			return "imt-000001"
		case "idnamespaceassociationidentifier":
			return "ina-000001"
		case "changerequestidentifier":
			return "cr-000001"
		case "name":
			return "stackyard_schema"
		case "type":
			return "AGGREGATION"
		case "accountid":
			return "111122223333"
		case "resourcearn":
			return url.PathEscape("arn:aws:cleanrooms:us-east-1:123456789012:collaboration/col-000001")
		case "tagkeys":
			return "env"
		case "maxresults":
			return "10"
		case "nexttoken":
			return "token-000001"
		case "status":
			return "ACTIVE"
		case "memberstatus":
			return "ACTIVE"
		case "privacybudgettype":
			return "DIFFERENTIAL_PRIVACY"
		case "accessbudgetresourcearn":
			return "arn-cleanrooms-resource"
		case "schematype":
			return "TABLE"
		default:
			return "stackyard"
		}
	})
}
