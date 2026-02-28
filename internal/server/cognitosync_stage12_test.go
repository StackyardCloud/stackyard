package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func cognitoSyncRequest(t *testing.T, ts *httptest.Server, method, path string, payload any) *http.Response {
	t.Helper()
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = encoded
	}
	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "cognito-sync")
}

func decodeCognitoSyncBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw := mustBody(t, resp)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	out := map[string]any{}
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decode JSON body: %v body=%s", err, string(raw))
	}
	return out
}

func TestCognitoSyncStage12ReadAndConfigurationFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	identityPoolID := url.PathEscape("us-east-1:sync-stage12-pool")
	identityID := url.PathEscape("us-east-1:sync-stage12-identity")
	datasetName := url.PathEscape("profile")

	describePoolResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID, nil)
	assertStatus(t, describePoolResp, http.StatusOK)
	describePoolBody := decodeCognitoSyncBody(t, describePoolResp)
	poolUsage, ok := describePoolBody["IdentityPoolUsage"].(map[string]any)
	if !ok {
		t.Fatalf("expected IdentityPoolUsage object, got %#v", describePoolBody["IdentityPoolUsage"])
	}
	if got, _ := poolUsage["IdentityPoolId"].(string); got != "us-east-1:sync-stage12-pool" {
		t.Fatalf("expected IdentityPoolId in usage output, got %#v", poolUsage["IdentityPoolId"])
	}

	listPoolResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools?maxResults=10", nil)
	assertStatus(t, listPoolResp, http.StatusOK)
	listPoolBody := decodeCognitoSyncBody(t, listPoolResp)
	poolUsages, ok := listPoolBody["IdentityPoolUsages"].([]any)
	if !ok || len(poolUsages) == 0 {
		t.Fatalf("expected IdentityPoolUsages list, got %#v", listPoolBody["IdentityPoolUsages"])
	}

	describeIdentityResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID+"/identities/"+identityID, nil)
	assertStatus(t, describeIdentityResp, http.StatusOK)
	describeIdentityBody := decodeCognitoSyncBody(t, describeIdentityResp)
	identityUsage, ok := describeIdentityBody["IdentityUsage"].(map[string]any)
	if !ok {
		t.Fatalf("expected IdentityUsage object, got %#v", describeIdentityBody["IdentityUsage"])
	}
	if got, _ := identityUsage["IdentityId"].(string); got != "us-east-1:sync-stage12-identity" {
		t.Fatalf("expected IdentityId in usage output, got %#v", identityUsage["IdentityId"])
	}

	listDatasetsResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets", nil)
	assertStatus(t, listDatasetsResp, http.StatusOK)

	describeDatasetResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName, nil)
	assertStatus(t, describeDatasetResp, http.StatusOK)
	describeDatasetBody := decodeCognitoSyncBody(t, describeDatasetResp)
	datasetOut, ok := describeDatasetBody["Dataset"].(map[string]any)
	if !ok {
		t.Fatalf("expected Dataset object, got %#v", describeDatasetBody["Dataset"])
	}
	if got, _ := datasetOut["DatasetName"].(string); got != "profile" {
		t.Fatalf("expected DatasetName=profile, got %#v", datasetOut["DatasetName"])
	}

	listRecordsResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0",
		nil,
	)
	assertStatus(t, listRecordsResp, http.StatusOK)
	listRecordsBody := decodeCognitoSyncBody(t, listRecordsResp)
	if exists, _ := listRecordsBody["DatasetExists"].(bool); !exists {
		t.Fatalf("expected DatasetExists=true, got %#v", listRecordsBody["DatasetExists"])
	}

	getEventsResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID+"/events", nil)
	assertStatus(t, getEventsResp, http.StatusOK)
	setEventsResp := cognitoSyncRequest(t, ts, http.MethodPost, "/identitypools/"+identityPoolID+"/events", map[string]any{
		"Events": map[string]any{
			"syncTrigger": "arn:aws:lambda:us-east-1:123456789012:function:stackyard-cognitosync-sync",
		},
	})
	assertStatus(t, setEventsResp, http.StatusOK)
	setEventsBody := decodeCognitoSyncBody(t, setEventsResp)
	eventsMap, ok := setEventsBody["Events"].(map[string]any)
	if !ok {
		t.Fatalf("expected Events map in set response, got %#v", setEventsBody["Events"])
	}
	if got, _ := eventsMap["syncTrigger"].(string); !strings.Contains(got, "stackyard-cognitosync-sync") {
		t.Fatalf("expected persisted sync trigger event, got %#v", eventsMap["syncTrigger"])
	}

	getEventsAgainResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID+"/events", nil)
	assertStatus(t, getEventsAgainResp, http.StatusOK)
	getEventsAgainBody := decodeCognitoSyncBody(t, getEventsAgainResp)
	eventsAgain, ok := getEventsAgainBody["Events"].(map[string]any)
	if !ok {
		t.Fatalf("expected Events map from GET, got %#v", getEventsAgainBody["Events"])
	}
	if got, _ := eventsAgain["syncTrigger"].(string); !strings.Contains(got, "stackyard-cognitosync-sync") {
		t.Fatalf("expected event to persist across get/set, got %#v", eventsAgain["syncTrigger"])
	}

	getConfigResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID+"/configuration", nil)
	assertStatus(t, getConfigResp, http.StatusOK)
	getConfigBody := decodeCognitoSyncBody(t, getConfigResp)
	if got, _ := getConfigBody["IdentityPoolId"].(string); got != "us-east-1:sync-stage12-pool" {
		t.Fatalf("expected IdentityPoolId in config output, got %#v", getConfigBody["IdentityPoolId"])
	}

	setConfigResp := cognitoSyncRequest(t, ts, http.MethodPost, "/identitypools/"+identityPoolID+"/configuration", map[string]any{
		"PushSync": map[string]any{
			"ApplicationArns": []string{"arn:aws:sns:us-east-1:123456789012:stackyard-mobile"},
			"RoleArn":         "arn:aws:iam::123456789012:role/stackyard-cognitosync-push",
		},
		"CognitoStreams": map[string]any{
			"StreamName":      "stackyard-sync-stream",
			"RoleArn":         "arn:aws:iam::123456789012:role/stackyard-cognitosync-stream",
			"StreamingStatus": "ENABLED",
		},
	})
	assertStatus(t, setConfigResp, http.StatusOK)
	setConfigBody := decodeCognitoSyncBody(t, setConfigResp)
	pushSync, ok := setConfigBody["PushSync"].(map[string]any)
	if !ok {
		t.Fatalf("expected PushSync map in set config output, got %#v", setConfigBody["PushSync"])
	}
	if got, _ := pushSync["RoleArn"].(string); !strings.Contains(got, "stackyard-cognitosync-push") {
		t.Fatalf("expected RoleArn in PushSync output, got %#v", pushSync["RoleArn"])
	}

	getConfigAgainResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools/"+identityPoolID+"/configuration", nil)
	assertStatus(t, getConfigAgainResp, http.StatusOK)
	getConfigAgainBody := decodeCognitoSyncBody(t, getConfigAgainResp)
	streams, ok := getConfigAgainBody["CognitoStreams"].(map[string]any)
	if !ok {
		t.Fatalf("expected CognitoStreams in config output, got %#v", getConfigAgainBody["CognitoStreams"])
	}
	if got, _ := streams["StreamingStatus"].(string); got != "ENABLED" {
		t.Fatalf("expected StreamingStatus=ENABLED, got %#v", streams["StreamingStatus"])
	}
}

func TestCognitoSyncStage12ValidationBoundaries(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	identityPoolID := url.PathEscape("us-east-1:sync-stage12-boundary-pool")

	invalidConfigResp := cognitoSyncRequest(t, ts, http.MethodPost, "/identitypools/"+identityPoolID+"/configuration", map[string]any{
		"CognitoStreams": map[string]any{
			"StreamName":      "stream",
			"RoleArn":         "arn:aws:iam::123456789012:role/stackyard",
			"StreamingStatus": "UNKNOWN",
		},
	})
	assertStatus(t, invalidConfigResp, http.StatusBadRequest)
	invalidConfigBody := string(mustBody(t, invalidConfigResp))
	if !strings.Contains(invalidConfigBody, "InvalidParameterException") {
		t.Fatalf("expected InvalidParameterException for invalid streaming status, got %q", invalidConfigBody)
	}
}
