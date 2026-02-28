package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudControlAPIStage345PaginationAndFiltering(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for i := 0; i < 3; i++ {
		createResp := cloudControlAPIRequest(t, ts, "CreateResource", map[string]any{
			"TypeName":     "AWS::SQS::Queue",
			"DesiredState": `{"QueueName":"stackyard-stage345-` + string(rune('a'+i)) + `"}`,
			"ClientToken":  "stage345-create-" + string(rune('a'+i)),
		})
		assertStatus(t, createResp, http.StatusOK)
	}

	pageOneResp := cloudControlAPIRequest(t, ts, "ListResources", map[string]any{
		"TypeName":   "AWS::SQS::Queue",
		"MaxResults": 2,
	})
	assertStatus(t, pageOneResp, http.StatusOK)
	pageOneBody := cloudControlAPIDecodeBody(t, pageOneResp)
	pageOneItems, ok := pageOneBody["ResourceDescriptions"].([]any)
	if !ok || len(pageOneItems) != 2 {
		t.Fatalf("expected 2 resources on first page, got %#v", pageOneBody["ResourceDescriptions"])
	}
	nextToken, _ := pageOneBody["NextToken"].(string)
	if strings.TrimSpace(nextToken) == "" {
		t.Fatalf("expected NextToken for paginated ListResources response")
	}

	pageTwoResp := cloudControlAPIRequest(t, ts, "ListResources", map[string]any{
		"TypeName":   "AWS::SQS::Queue",
		"MaxResults": 2,
		"NextToken":  nextToken,
	})
	assertStatus(t, pageTwoResp, http.StatusOK)
	pageTwoBody := cloudControlAPIDecodeBody(t, pageTwoResp)
	pageTwoItems, ok := pageTwoBody["ResourceDescriptions"].([]any)
	if !ok || len(pageTwoItems) != 1 {
		t.Fatalf("expected 1 resource on second page, got %#v", pageTwoBody["ResourceDescriptions"])
	}

	requestsResp := cloudControlAPIRequest(t, ts, "ListResourceRequests", map[string]any{
		"MaxResults": 10,
		"ResourceRequestStatusFilter": map[string]any{
			"TypeName":          "AWS::SQS::Queue",
			"Operations":        []string{"CREATE"},
			"OperationStatuses": []string{"SUCCESS"},
		},
	})
	assertStatus(t, requestsResp, http.StatusOK)
	requestsBody := cloudControlAPIDecodeBody(t, requestsResp)
	summaries, ok := requestsBody["ResourceRequestStatusSummaries"].([]any)
	if !ok || len(summaries) < 3 {
		t.Fatalf("expected at least 3 create request summaries, got %#v", requestsBody["ResourceRequestStatusSummaries"])
	}
}

func TestCloudControlAPIStage345ValidationAndNotFound(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	missingResourceResp := cloudControlAPIRequest(t, ts, "DeleteResource", map[string]any{
		"TypeName":   "AWS::S3::Bucket",
		"Identifier": "missing-resource",
	})
	assertStatus(t, missingResourceResp, http.StatusBadRequest)
	missingBody := string(mustBody(t, missingResourceResp))
	if !strings.Contains(missingBody, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException, got %q", missingBody)
	}

	missingRequestResp := cloudControlAPIRequest(t, ts, "GetResourceRequestStatus", map[string]any{
		"RequestToken": "req-does-not-exist",
	})
	assertStatus(t, missingRequestResp, http.StatusBadRequest)
	missingRequestBody := string(mustBody(t, missingRequestResp))
	if !strings.Contains(missingRequestBody, "RequestTokenNotFoundException") {
		t.Fatalf("expected RequestTokenNotFoundException, got %q", missingRequestBody)
	}

	createResp := cloudControlAPIRequest(t, ts, "CreateResource", map[string]any{
		"TypeName":     "AWS::S3::Bucket",
		"DesiredState": `{"BucketName":"stackyard-stage345-invalid-patch"}`,
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := cloudControlAPIDecodeBody(t, createResp)
	progress, ok := createBody["ProgressEvent"].(map[string]any)
	if !ok {
		t.Fatalf("expected ProgressEvent in create response")
	}
	identifier, _ := progress["Identifier"].(string)
	if strings.TrimSpace(identifier) == "" {
		t.Fatalf("expected identifier in create response")
	}

	invalidPatchResp := cloudControlAPIRequest(t, ts, "UpdateResource", map[string]any{
		"TypeName":      "AWS::S3::Bucket",
		"Identifier":    identifier,
		"PatchDocument": `not-json`,
	})
	assertStatus(t, invalidPatchResp, http.StatusBadRequest)
	invalidPatchBody := string(mustBody(t, invalidPatchResp))
	if !strings.Contains(invalidPatchBody, "ValidationException") {
		t.Fatalf("expected ValidationException for invalid patch document, got %q", invalidPatchBody)
	}
}
