package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func iotMIRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	body := []byte(payload)
	if payload == "" {
		body = nil
	}
	headers := map[string]string{"Content-Type": "application/json"}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "iotmanagedintegrations")
}

func TestIoTManagedIntegrationsStage0CatalogCoverage(t *testing.T) {
	if len(iotMIActions) != 83 {
		t.Fatalf("expected 83 IoT Managed Integrations actions from docs, got %d", len(iotMIActions))
	}
	if len(iotMIActionByName) != len(iotMIActions) {
		t.Fatalf("expected unique IoT Managed Integrations action names")
	}

	requiredActions := []string{
		"CreateManagedThing",
		"GetManagedThing",
		"ListManagedThings",
		"CreateOtaTask",
		"GetOtaTask",
		"ListOtaTasks",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := iotMIActionByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(iotMIDataTypes) != 62 {
		t.Fatalf("expected 62 IoT Managed Integrations data types from docs, got %d", len(iotMIDataTypes))
	}
	if len(iotMIDataTypeByName) != len(iotMIDataTypes) {
		t.Fatalf("expected unique IoT Managed Integrations data type names")
	}

	requiredTypes := []string{
		"ManagedThingSummary",
		"ManagedThingAssociation",
		"OtaTaskSummary",
		"OtaTaskExecutionSummary",
		"ProvisioningProfileSummary",
		"RuntimeLogConfigurations",
	}
	for _, typeName := range requiredTypes {
		if _, ok := iotMIDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestIoTManagedIntegrationsStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotMIRequest(t, ts, http.MethodPost, "/iot-mi/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIoTManagedIntegrationsStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotMIRequest(t, ts, http.MethodGet, "/custom-endpoint", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "endpointAddress") {
		t.Fatalf("expected GetCustomEndpoint response to include endpointAddress, got %q", body)
	}
}

func TestIoTManagedIntegrationsStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"AccountAssociationId":             "assoc-000001",
		"Identifier":                       "thing-000001",
		"Name":                             "stackyard-destination",
		"Id":                               "event-log-000001",
		"EventType":                        "DEVICE_CONNECTED",
		"Force":                            "true",
		"ManagedThingId":                   "thing-000001",
		"Type":                             "capability",
		"SchemaVersionedId":                "schema-000001",
		"Format":                           "JSON",
		"ConnectorDestinationId":           "connector-destination-000001",
		"MaxResults":                       "10",
		"NextToken":                        "token-000001",
		"LambdaArn":                        "arn:aws:lambda:us-east-1:123456789012:function:stackyard-iotmi",
		"CloudConnectorId":                 "connector-000001",
		"StatusFilter":                     "ACTIVE",
		"TypeFilter":                       "MATTER",
		"CapabilityIdFilter":               "switch",
		"EndpointIdFilter":                 "endpoint-1",
		"ConnectorDestinationIdFilter":     "connector-destination-000001",
		"ConnectorDeviceIdFilter":          "device-000001",
		"ConnectorPolicyIdFilter":          "policy-000001",
		"CredentialLockerFilter":           "locker-000001",
		"OwnerFilter":                      "SELF",
		"ParentControllerIdentifierFilter": "controller-000001",
		"ProvisioningStatusFilter":         "COMPLETED",
		"RoleFilter":                       "PRIMARY",
		"SerialNumberFilter":               "SN-000001",
		"Namespace":                        "stackyard",
		"SchemaId":                         "switch",
		"SemanticVersion":                  "1.0.0",
		"Visibility":                       "PUBLIC",
		"ResourceArn":                      "arn:aws:iotmanagedintegrations:us-east-1:123456789012:managed-thing/thing-000001",
		"TagKeys":                          "owner",
		"ConnectorId":                      "connector-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, action := range iotMIActions {
		path := placeholder.ReplaceAllStringFunc(action.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		payload := ""
		if action.Method == http.MethodPost || action.Method == http.MethodPut || action.Method == http.MethodPatch {
			payload = `{}`
		}
		resp := iotMIRequest(t, ts, action.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", action.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", action.Name, resp.StatusCode, body)
		}
	}
}
