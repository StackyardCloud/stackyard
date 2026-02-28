package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudControlAPIStage12ResourceLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := cloudControlAPIRequest(t, ts, "CreateResource", map[string]any{
		"TypeName":     "AWS::S3::Bucket",
		"DesiredState": `{"BucketName":"stackyard-stage12-bucket"}`,
		"ClientToken":  "stage12-create-token",
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := cloudControlAPIDecodeBody(t, createResp)
	progress, ok := createBody["ProgressEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected ProgressEvent in CreateResource response")
	}
	identifier, _ := progress["Identifier"].(string)
	if strings.TrimSpace(identifier) == "" {
		t.Fatalf("expected identifier in create progress event")
	}
	requestToken, _ := progress["RequestToken"].(string)
	if strings.TrimSpace(requestToken) == "" {
		t.Fatalf("expected request token in create progress event")
	}

	getResp := cloudControlAPIRequest(t, ts, "GetResource", map[string]any{
		"TypeName":   "AWS::S3::Bucket",
		"Identifier": identifier,
	})
	assertStatus(t, getResp, http.StatusOK)
	getBody := cloudControlAPIDecodeBody(t, getResp)
	if gotType, _ := getBody["TypeName"].(string); gotType != "AWS::S3::Bucket" {
		t.Fatalf("expected TypeName AWS::S3::Bucket, got %#v", getBody["TypeName"])
	}
	resourceDesc, ok := getBody["ResourceDescription"].(map[string]any)
	if !ok {
		t.Fatalf("expected ResourceDescription object")
	}
	if gotID, _ := resourceDesc["Identifier"].(string); gotID != identifier {
		t.Fatalf("expected identifier %q, got %#v", identifier, resourceDesc["Identifier"])
	}

	listResp := cloudControlAPIRequest(t, ts, "ListResources", map[string]any{
		"TypeName":   "AWS::S3::Bucket",
		"MaxResults": 10,
	})
	assertStatus(t, listResp, http.StatusOK)
	listBody := cloudControlAPIDecodeBody(t, listResp)
	descriptions, ok := listBody["ResourceDescriptions"].([]any)
	if !ok || len(descriptions) == 0 {
		t.Fatalf("expected at least one listed resource, got %#v", listBody["ResourceDescriptions"])
	}

	updateResp := cloudControlAPIRequest(t, ts, "UpdateResource", map[string]any{
		"TypeName":      "AWS::S3::Bucket",
		"Identifier":    identifier,
		"PatchDocument": `[{"op":"replace","path":"/BucketName","value":"stackyard-stage12-bucket-updated"}]`,
		"ClientToken":   "stage12-update-token",
	})
	assertStatus(t, updateResp, http.StatusOK)
	updateBody := cloudControlAPIDecodeBody(t, updateResp)
	updateProgress, ok := updateBody["ProgressEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected ProgressEvent in UpdateResource response")
	}
	updateRequestToken, _ := updateProgress["RequestToken"].(string)
	if strings.TrimSpace(updateRequestToken) == "" {
		t.Fatalf("expected request token in update progress event")
	}

	statusResp := cloudControlAPIRequest(t, ts, "GetResourceRequestStatus", map[string]any{
		"RequestToken": updateRequestToken,
	})
	assertStatus(t, statusResp, http.StatusOK)
	statusBody := cloudControlAPIDecodeBody(t, statusResp)
	statusProgress, ok := statusBody["ProgressEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected ProgressEvent in GetResourceRequestStatus response")
	}
	if gotStatus, _ := statusProgress["OperationStatus"].(string); gotStatus == "" {
		t.Fatalf("expected operation status in request status response")
	}

	listRequestsResp := cloudControlAPIRequest(t, ts, "ListResourceRequests", map[string]any{
		"MaxResults": 10,
	})
	assertStatus(t, listRequestsResp, http.StatusOK)
	listRequestsBody := cloudControlAPIDecodeBody(t, listRequestsResp)
	summaries, ok := listRequestsBody["ResourceRequestStatusSummaries"].([]any)
	if !ok || len(summaries) == 0 {
		t.Fatalf("expected resource request summaries, got %#v", listRequestsBody["ResourceRequestStatusSummaries"])
	}

	cancelResp := cloudControlAPIRequest(t, ts, "CancelResourceRequest", map[string]any{
		"RequestToken": requestToken,
	})
	assertStatus(t, cancelResp, http.StatusOK)
	cancelBody := cloudControlAPIDecodeBody(t, cancelResp)
	cancelProgress, ok := cancelBody["ProgressEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected ProgressEvent in cancel response")
	}
	if gotStatus, _ := cancelProgress["OperationStatus"].(string); gotStatus != "CANCEL_COMPLETE" {
		t.Fatalf("expected CANCEL_COMPLETE status, got %#v", cancelProgress["OperationStatus"])
	}

	deleteResp := cloudControlAPIRequest(t, ts, "DeleteResource", map[string]any{
		"TypeName":   "AWS::S3::Bucket",
		"Identifier": identifier,
	})
	assertStatus(t, deleteResp, http.StatusOK)

	listAfterDeleteResp := cloudControlAPIRequest(t, ts, "ListResources", map[string]any{
		"TypeName":   "AWS::S3::Bucket",
		"MaxResults": 10,
	})
	assertStatus(t, listAfterDeleteResp, http.StatusOK)
	listAfterDeleteBody := cloudControlAPIDecodeBody(t, listAfterDeleteResp)
	afterDescriptions, ok := listAfterDeleteBody["ResourceDescriptions"].([]any)
	if !ok {
		t.Fatalf("expected ResourceDescriptions list, got %#v", listAfterDeleteBody["ResourceDescriptions"])
	}
	if len(afterDescriptions) != 0 {
		t.Fatalf("expected zero resources after delete, got %#v", afterDescriptions)
	}
}
