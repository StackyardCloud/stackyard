package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCognitoSyncStage345DeviceDatasetAndBulkPublishFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	identityPoolID := url.PathEscape("us-east-1:sync-stage345-pool")
	identityID := url.PathEscape("us-east-1:sync-stage345-identity")
	datasetName := url.PathEscape("profile")

	registerResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identity/"+identityID+"/device?platform=APNS",
		map[string]any{"Token": "device-token-stage345"},
	)
	assertStatus(t, registerResp, http.StatusOK)
	registerBody := decodeCognitoSyncBody(t, registerResp)
	deviceID, _ := registerBody["DeviceId"].(string)
	if strings.TrimSpace(deviceID) == "" {
		t.Fatalf("expected DeviceId from RegisterDevice, got %#v", registerBody)
	}

	subscribeResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/subscriptions/"+url.PathEscape(deviceID),
		nil,
	)
	assertStatus(t, subscribeResp, http.StatusOK)

	listRecordsBeforeResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0",
		nil,
	)
	assertStatus(t, listRecordsBeforeResp, http.StatusOK)
	listRecordsBeforeBody := decodeCognitoSyncBody(t, listRecordsBeforeResp)
	syncSessionToken, _ := listRecordsBeforeBody["SyncSessionToken"].(string)
	if strings.TrimSpace(syncSessionToken) == "" {
		t.Fatalf("expected SyncSessionToken in ListRecords response")
	}

	updateResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName,
		map[string]any{
			"DeviceId":         deviceID,
			"SyncSessionToken": syncSessionToken,
			"RecordPatches": []map[string]any{
				{"Op": "replace", "Key": "nickname", "Value": "stackyard", "SyncCount": 0},
			},
		},
	)
	assertStatus(t, updateResp, http.StatusOK)
	updateBody := decodeCognitoSyncBody(t, updateResp)
	records, ok := updateBody["Records"].([]any)
	if !ok || len(records) == 0 {
		t.Fatalf("expected updated records in UpdateRecords output, got %#v", updateBody["Records"])
	}

	listRecordsAfterResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0",
		nil,
	)
	assertStatus(t, listRecordsAfterResp, http.StatusOK)
	listRecordsAfterBody := decodeCognitoSyncBody(t, listRecordsAfterResp)
	afterRecords, ok := listRecordsAfterBody["Records"].([]any)
	if !ok || len(afterRecords) != 1 {
		t.Fatalf("expected one record after update, got %#v", listRecordsAfterBody["Records"])
	}

	staleUpdateResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName,
		map[string]any{
			"DeviceId":         deviceID,
			"SyncSessionToken": syncSessionToken,
			"RecordPatches": []map[string]any{
				{"Op": "replace", "Key": "nickname", "Value": "stackyard-2", "SyncCount": 0},
			},
		},
	)
	assertStatus(t, staleUpdateResp, http.StatusConflict)
	staleUpdateBody := string(mustBody(t, staleUpdateResp))
	if !strings.Contains(staleUpdateBody, "ResourceConflictException") {
		t.Fatalf("expected ResourceConflictException for stale sync count, got %q", staleUpdateBody)
	}

	removeResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName,
		map[string]any{
			"DeviceId":         deviceID,
			"SyncSessionToken": syncSessionToken,
			"RecordPatches": []map[string]any{
				{"Op": "remove", "Key": "nickname", "SyncCount": 1},
			},
		},
	)
	assertStatus(t, removeResp, http.StatusOK)

	unsubscribeResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodDelete,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/subscriptions/"+url.PathEscape(deviceID),
		nil,
	)
	assertStatus(t, unsubscribeResp, http.StatusOK)

	unsubscribeAgainResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodDelete,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/subscriptions/"+url.PathEscape(deviceID),
		nil,
	)
	assertStatus(t, unsubscribeAgainResp, http.StatusNotFound)

	deleteDatasetResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodDelete,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName,
		nil,
	)
	assertStatus(t, deleteDatasetResp, http.StatusOK)

	listAfterDeleteResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0",
		nil,
	)
	assertStatus(t, listAfterDeleteResp, http.StatusOK)
	listAfterDeleteBody := decodeCognitoSyncBody(t, listAfterDeleteResp)
	if exists, _ := listAfterDeleteBody["DatasetExists"].(bool); exists {
		t.Fatalf("expected DatasetExists=false after delete, got %#v", listAfterDeleteBody["DatasetExists"])
	}
	if deletedAfter, _ := listAfterDeleteBody["DatasetDeletedAfterRequestedSyncCount"].(bool); !deletedAfter {
		t.Fatalf("expected DatasetDeletedAfterRequestedSyncCount=true, got %#v", listAfterDeleteBody["DatasetDeletedAfterRequestedSyncCount"])
	}

	bulkPublishResp := cognitoSyncRequest(t, ts, http.MethodPost, "/identitypools/"+identityPoolID+"/bulkpublish", map[string]any{})
	assertStatus(t, bulkPublishResp, http.StatusOK)

	bulkDetailsResp := cognitoSyncRequest(t, ts, http.MethodPost, "/identitypools/"+identityPoolID+"/getBulkPublishDetails", map[string]any{})
	assertStatus(t, bulkDetailsResp, http.StatusOK)
	bulkDetailsBody := decodeCognitoSyncBody(t, bulkDetailsResp)
	if got, _ := bulkDetailsBody["BulkPublishStatus"].(string); got != "SUCCEEDED" {
		t.Fatalf("expected BulkPublishStatus=SUCCEEDED, got %#v", bulkDetailsBody["BulkPublishStatus"])
	}

	bulkPublishAgainResp := cognitoSyncRequest(t, ts, http.MethodPost, "/identitypools/"+identityPoolID+"/bulkpublish", map[string]any{})
	assertStatus(t, bulkPublishAgainResp, http.StatusBadRequest)
	bulkPublishAgainBody := string(mustBody(t, bulkPublishAgainResp))
	if !strings.Contains(bulkPublishAgainBody, "LimitExceededException") {
		t.Fatalf("expected LimitExceededException for repeated bulk publish, got %q", bulkPublishAgainBody)
	}
}

func TestCognitoSyncStage6PaginationAndValidationHardening(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	identityPoolID := url.PathEscape("us-east-1:sync-stage6-pool")
	identityID := url.PathEscape("us-east-1:sync-stage6-identity")
	datasetName := url.PathEscape("stage6")

	registerResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identity/"+identityID+"/device?platform=APNS",
		map[string]any{"Token": "device-token-stage6"},
	)
	assertStatus(t, registerResp, http.StatusOK)
	registerBody := decodeCognitoSyncBody(t, registerResp)
	deviceID, _ := registerBody["DeviceId"].(string)
	if strings.TrimSpace(deviceID) == "" {
		t.Fatalf("expected DeviceId from RegisterDevice, got %#v", registerBody)
	}

	listBeforeResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0",
		nil,
	)
	assertStatus(t, listBeforeResp, http.StatusOK)
	listBeforeBody := decodeCognitoSyncBody(t, listBeforeResp)
	syncSessionToken, _ := listBeforeBody["SyncSessionToken"].(string)
	if strings.TrimSpace(syncSessionToken) == "" {
		t.Fatalf("expected SyncSessionToken from ListRecords")
	}

	seedResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName,
		map[string]any{
			"DeviceId":         deviceID,
			"SyncSessionToken": syncSessionToken,
			"RecordPatches": []map[string]any{
				{"Op": "replace", "Key": "a", "Value": "1", "SyncCount": 0},
				{"Op": "replace", "Key": "b", "Value": "2", "SyncCount": 0},
				{"Op": "replace", "Key": "c", "Value": "3", "SyncCount": 0},
			},
		},
	)
	assertStatus(t, seedResp, http.StatusOK)

	page1Resp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0&maxResults=2",
		nil,
	)
	assertStatus(t, page1Resp, http.StatusOK)
	page1Body := decodeCognitoSyncBody(t, page1Resp)
	recordsPage1, ok := page1Body["Records"].([]any)
	if !ok || len(recordsPage1) != 2 {
		t.Fatalf("expected 2 records on page 1, got %#v", page1Body["Records"])
	}
	nextToken, _ := page1Body["NextToken"].(string)
	if strings.TrimSpace(nextToken) == "" {
		t.Fatalf("expected NextToken on page 1")
	}

	page2Resp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0&maxResults=2&nextToken="+url.QueryEscape(nextToken),
		nil,
	)
	assertStatus(t, page2Resp, http.StatusOK)
	page2Body := decodeCognitoSyncBody(t, page2Resp)
	recordsPage2, ok := page2Body["Records"].([]any)
	if !ok || len(recordsPage2) != 1 {
		t.Fatalf("expected 1 record on page 2, got %#v", page2Body["Records"])
	}

	invalidNextTokenResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodGet,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName+"/records?lastSyncCount=0&nextToken=invalid-token",
		nil,
	)
	assertStatus(t, invalidNextTokenResp, http.StatusBadRequest)
	invalidNextTokenBody := string(mustBody(t, invalidNextTokenResp))
	if !strings.Contains(invalidNextTokenBody, "InvalidParameterException") {
		t.Fatalf("expected InvalidParameterException for invalid next token, got %q", invalidNextTokenBody)
	}

	invalidMaxResultsResp := cognitoSyncRequest(t, ts, http.MethodGet, "/identitypools?maxResults=1000", nil)
	assertStatus(t, invalidMaxResultsResp, http.StatusBadRequest)

	invalidSessionTokenResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identities/"+identityID+"/datasets/"+datasetName,
		map[string]any{
			"DeviceId":         deviceID,
			"SyncSessionToken": "invalid-session-token",
			"RecordPatches": []map[string]any{
				{"Op": "replace", "Key": "a", "Value": "9", "SyncCount": 1},
			},
		},
	)
	assertStatus(t, invalidSessionTokenResp, http.StatusConflict)
	invalidSessionTokenBody := string(mustBody(t, invalidSessionTokenResp))
	if !strings.Contains(invalidSessionTokenBody, "ResourceConflictException") {
		t.Fatalf("expected ResourceConflictException for invalid session token, got %q", invalidSessionTokenBody)
	}

	invalidPlatformResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/identity/"+identityID+"/device?platform=INVALID",
		map[string]any{"Token": "token-invalid-platform"},
	)
	assertStatus(t, invalidPlatformResp, http.StatusBadRequest)

	arns := make([]string, 11)
	for i := range arns {
		arns[i] = fmt.Sprintf("arn:aws:sns:us-east-1:123456789012:app-%d", i)
	}
	tooManyArnsResp := cognitoSyncRequest(
		t,
		ts,
		http.MethodPost,
		"/identitypools/"+identityPoolID+"/configuration",
		map[string]any{
			"PushSync": map[string]any{
				"ApplicationArns": arns,
				"RoleArn":         "arn:aws:iam::123456789012:role/stackyard-cognitosync-push",
			},
		},
	)
	assertStatus(t, tooManyArnsResp, http.StatusBadRequest)
	tooManyArnsBody := string(mustBody(t, tooManyArnsResp))
	if !strings.Contains(tooManyArnsBody, "LimitExceededException") {
		t.Fatalf("expected LimitExceededException for excessive ApplicationArns, got %q", tooManyArnsBody)
	}
}
