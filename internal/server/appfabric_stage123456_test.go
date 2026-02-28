package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAppFabricStage12AppBundleAndAuthorizationLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bundleID := "ab-stage-001"
	authID := "auth-stage-001"

	resp := appFabricRequest(t, ts, http.MethodPost, "/appbundles", []byte(`{"appBundleIdentifier":"`+bundleID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(t, ts, http.MethodGet, "/appbundles/"+url.PathEscape(bundleID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/appbundles/"+url.PathEscape(bundleID)+"/appauthorizations",
		[]byte(`{"appAuthorizationIdentifier":"`+authID+`","app":"okta"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/appbundles/"+url.PathEscape(bundleID)+"/appauthorizations/"+url.PathEscape(authID)+"/connect",
		[]byte(`{}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPatch,
		"/appbundles/"+url.PathEscape(bundleID)+"/appauthorizations/"+url.PathEscape(authID),
		[]byte(`{"status":"CONNECTED"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodGet,
		"/appbundles/"+url.PathEscape(bundleID)+"/appauthorizations/"+url.PathEscape(authID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(t, ts, http.MethodGet, "/appbundles/"+url.PathEscape(bundleID)+"/appauthorizations", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodDelete,
		"/appbundles/"+url.PathEscape(bundleID)+"/appauthorizations/"+url.PathEscape(authID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(t, ts, http.MethodDelete, "/appbundles/"+url.PathEscape(bundleID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestAppFabricStage34IngestionAndDestinationLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bundleID := "ab-stage-002"
	ingestionID := "ing-stage-001"
	destinationID := "dest-stage-001"

	resp := appFabricRequest(t, ts, http.MethodPost, "/appbundles", []byte(`{"appBundleIdentifier":"`+bundleID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions",
		[]byte(`{"ingestionIdentifier":"`+ingestionID+`"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID)+"/start",
		[]byte(`{}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID)+"/ingestiondestinations",
		[]byte(`{"ingestionDestinationIdentifier":"`+destinationID+`"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPatch,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID)+"/ingestiondestinations/"+url.PathEscape(destinationID),
		[]byte(`{"state":"ACTIVE"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(t, ts, http.MethodGet, "/appbundles/"+url.PathEscape(bundleID)+"/ingestions", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodGet,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodGet,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID)+"/ingestiondestinations",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodGet,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID)+"/ingestiondestinations/"+url.PathEscape(destinationID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID)+"/stop",
		[]byte(`{}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodDelete,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID)+"/ingestiondestinations/"+url.PathEscape(destinationID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodDelete,
		"/appbundles/"+url.PathEscape(bundleID)+"/ingestions/"+url.PathEscape(ingestionID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(t, ts, http.MethodDelete, "/appbundles/"+url.PathEscape(bundleID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestAppFabricStage56UserAccessTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/useraccess/start",
		[]byte(`{"appBundleIdentifier":"ab-000001","userAccessTaskId":"uat-stage-001"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/useraccess/batchget",
		[]byte(`{"userAccessTaskIds":["uat-stage-001"]}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:appfabric:us-east-1:123456789012:appbundle/ab-000001"
	escapedARN := url.PathEscape(resourceARN)

	resp = appFabricRequest(
		t,
		ts,
		http.MethodPost,
		"/tags/"+escapedARN,
		[]byte(`{"tags":{"env":"stage","owner":"qa"}}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = appFabricRequest(t, ts, http.MethodDelete, "/tags/"+escapedARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = appFabricRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "owner") {
		t.Fatalf("expected owner tag removed, got %q", body)
	}

	resp = appFabricRequest(t, ts, http.MethodGet, "/appfabric/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/appbundles",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"appfabric",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	idempotentBundle := "ab-stage-idempotent-001"
	resp = appFabricRequest(t, ts, http.MethodPost, "/appbundles", []byte(`{"appBundleIdentifier":"`+idempotentBundle+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appFabricRequest(t, ts, http.MethodPost, "/appbundles", []byte(`{"appBundleIdentifier":"`+idempotentBundle+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appFabricRequest(t, ts, http.MethodGet, "/appbundles/"+url.PathEscape(idempotentBundle), nil)
	assertStatus(t, resp, http.StatusOK)
}
