package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func locationRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "geo")
}

func TestLocationStage0CatalogCoverage(t *testing.T) {
	if len(locationOperations) != 60 {
		t.Fatalf("expected 60 Location actions from docs, got %d", len(locationOperations))
	}
	if len(locationOperationByName) != len(locationOperations) {
		t.Fatalf("expected unique Location action names")
	}

	requiredActions := []string{
		"CreateMap",
		"DescribeMap",
		"ListMaps",
		"CreatePlaceIndex",
		"SearchPlaceIndexForText",
		"CreateRouteCalculator",
		"CalculateRoute",
		"CreateTracker",
		"BatchUpdateDevicePosition",
		"CreateGeofenceCollection",
		"PutGeofence",
		"CreateKey",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := locationOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(locationDataTypes) != 50 {
		t.Fatalf("expected 50 Location data types from docs, got %d", len(locationDataTypes))
	}
	if len(locationDataTypeByName) != len(locationDataTypes) {
		t.Fatalf("expected unique Location data type names")
	}

	requiredTypes := []string{
		"ApiKeyRestrictions",
		"BatchPutGeofenceRequestEntry",
		"CalculateRouteSummary",
		"DevicePosition",
		"GeofenceGeometry",
		"MapConfiguration",
		"Place",
		"SearchForTextResult",
	}
	for _, typeName := range requiredTypes {
		if _, ok := locationDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestLocationStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := locationRequest(t, ts, http.MethodPost, "/location/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestLocationStage0KnownActionReturnsListMaps(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := locationRequest(t, ts, http.MethodPost, "/maps/v0/list-maps", []byte(`{"MaxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Entries") {
		t.Fatalf("expected ListMaps response body to include Entries, got %q", body)
	}
}

func TestLocationStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"TrackerName":      "stackyard-tracker",
		"CollectionName":   "stackyard-geofence-collection",
		"CalculatorName":   "stackyard-route-calculator",
		"KeyName":          "stackyard-api-key",
		"MapName":          "stackyard-map",
		"IndexName":        "stackyard-place-index",
		"ConsumerArn":      "arn:aws:kinesis:us-east-1:123456789012:stream/stackyard-tracker-consumer",
		"DeviceId":         "device-000001",
		"GeofenceId":       "stackyard-geofence",
		"FontStack":        "Arial",
		"FontUnicodeRange": "0-255.pbf",
		"FileName":         "sprite@2x.png",
		"Z":                "0",
		"X":                "0",
		"Y":                "0",
		"PlaceId":          "place-000001",
		"ResourceArn":      "arn:aws:geo:us-east-1:123456789012:map/stackyard-map",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range locationOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := locationRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
