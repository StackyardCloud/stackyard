package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func TestCleanRoomsMLStage0CatalogCoverage(t *testing.T) {
	if len(cleanRoomsMLOperations) != 59 {
		t.Fatalf("expected 59 AWS Clean Rooms ML actions from docs, got %d", len(cleanRoomsMLOperations))
	}
	if len(cleanRoomsMLOperationByName) != len(cleanRoomsMLOperations) {
		t.Fatalf("expected unique AWS Clean Rooms ML action names")
	}

	requiredActions := []string{
		"CreateTrainingDataset",
		"CreateAudienceModel",
		"CreateConfiguredAudienceModel",
		"CreateTrainedModel",
		"StartTrainedModelInferenceJob",
		"ListTrainedModels",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := cleanRoomsMLOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(cleanRoomsMLDataTypes) != 75 {
		t.Fatalf("expected 75 AWS Clean Rooms ML data types from docs, got %d", len(cleanRoomsMLDataTypes))
	}
	if len(cleanRoomsMLDataTypeByName) != len(cleanRoomsMLDataTypes) {
		t.Fatalf("expected unique AWS Clean Rooms ML data type names")
	}

	requiredTypes := []string{
		"TrainingDatasetSummary",
		"AudienceModelSummary",
		"ConfiguredAudienceModelSummary",
		"TrainedModelSummary",
		"StatusDetails",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cleanRoomsMLDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func cleanRoomsMLRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "cleanrooms-ml")
}

func TestCleanRoomsMLStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cleanRoomsMLRequest(t, ts, http.MethodPost, "/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCleanRoomsMLStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cleanRoomsMLRequest(t, ts, http.MethodGet, "/audience-model", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
}

func TestCleanRoomsMLStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cleanRoomsMLOperations {
		path := cleanRoomsMLRenderTestURI(op.URI)
		payload := "{}"
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}

		resp := cleanRoomsMLRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s (path=%s)", op.Name, resp.StatusCode, body, path)
		}
	}
}

var cleanRoomsMLURIPlaceholderPattern = regexp.MustCompile(`\{([^}]+)\}`)

func cleanRoomsMLRenderTestURI(uriTemplate string) string {
	return cleanRoomsMLURIPlaceholderPattern.ReplaceAllStringFunc(uriTemplate, func(raw string) string {
		placeholder := strings.TrimSuffix(strings.Trim(raw, "{}"), "+")
		switch strings.ToLower(placeholder) {
		case "membershipidentifier":
			return "mem-000001"
		case "collaborationidentifier", "collaborationid":
			return "col-000001"
		case "trainedmodelarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:trained-model/tm-000001")
		case "trainedmodelinferencejobarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:trained-model-inference-job/tmij-000001")
		case "audiencemodelarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:audience-model/am-000001")
		case "audiencegenerationjobarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:audience-generation-job/agj-000001")
		case "configuredaudiencemodelarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:configured-audience-model/cam-000001")
		case "configuredmodelalgorithmarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:configured-model-algorithm/cma-000001")
		case "configuredmodelalgorithmassociationarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:configured-model-algorithm-association/cmaa-000001")
		case "trainingdatasetarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:training-dataset/td-000001")
		case "mlinputchannelarn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:ml-input-channel/mlic-000001")
		case "resourcearn":
			return url.PathEscape("arn:aws:cleanrooms-ml:us-east-1:123456789012:configured-audience-model/cam-000001")
		case "versionidentifier", "trainedmodelversionidentifier":
			return "1"
		case "maxresults":
			return "10"
		case "nexttoken":
			return "token-000001"
		case "status":
			return "ACTIVE"
		case "tagkeys":
			return "env"
		default:
			return "stackyard"
		}
	})
}
