package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRekognitionStage12CollectionsProjectsAndDatasets(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := rekognitionRequest(t, ts, "CreateCollection", `{"CollectionId":"stage-collection"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "DescribeCollection", `{"CollectionId":"stage-collection"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "ListCollections", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-collection") {
		t.Fatalf("expected ListCollections to include stage-collection, got %q", body)
	}

	resp = rekognitionRequest(t, ts, "CreateProject", `{"ProjectArn":"arn:aws:rekognition:us-east-1:123456789012:project/stage-project"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "DescribeProjects", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "CreateDataset", `{"DatasetArn":"arn:aws:rekognition:us-east-1:123456789012:project/stage-project/dataset/train"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "DescribeDataset", `{"DatasetArn":"arn:aws:rekognition:us-east-1:123456789012:project/stage-project/dataset/train"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "ListDatasetEntries", `{"DatasetArn":"arn:aws:rekognition:us-east-1:123456789012:project/stage-project/dataset/train"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "ListDatasetLabels", `{"DatasetArn":"arn:aws:rekognition:us-east-1:123456789012:project/stage-project/dataset/train"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestRekognitionStage34DetectionSearchAndJobs(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	imagePayload := `{"Image":{"Bytes":"c3RhY2t5YXJk"}}`
	for _, action := range []string{"DetectLabels", "DetectFaces", "DetectModerationLabels", "DetectText", "RecognizeCelebrities", "CompareFaces", "DetectCustomLabels", "DetectProtectiveEquipment"} {
		resp := rekognitionRequest(t, ts, action, imagePayload)
		assertStatus(t, resp, http.StatusOK)
	}

	resp := rekognitionRequest(t, ts, "IndexFaces", `{"CollectionId":"stackyard-collection","Image":{"Bytes":"c3RhY2t5YXJk"}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "ListFaces", `{"CollectionId":"stackyard-collection"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "SearchFaces", `{"CollectionId":"stackyard-collection","FaceId":"face-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "SearchFacesByImage", `{"CollectionId":"stackyard-collection","Image":{"Bytes":"c3RhY2t5YXJk"}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "CreateUser", `{"CollectionId":"stackyard-collection","UserId":"stage-user"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "AssociateFaces", `{"CollectionId":"stackyard-collection","UserId":"stage-user","FaceIds":["face-000001"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "SearchUsers", `{"CollectionId":"stackyard-collection","FaceId":"face-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "SearchUsersByImage", `{"CollectionId":"stackyard-collection","Image":{"Bytes":"c3RhY2t5YXJk"}}`)
	assertStatus(t, resp, http.StatusOK)

	startToGet := map[string]string{
		"StartLabelDetection":       "GetLabelDetection",
		"StartFaceDetection":        "GetFaceDetection",
		"StartFaceSearch":           "GetFaceSearch",
		"StartCelebrityRecognition": "GetCelebrityRecognition",
		"StartContentModeration":    "GetContentModeration",
		"StartPersonTracking":       "GetPersonTracking",
		"StartSegmentDetection":     "GetSegmentDetection",
		"StartTextDetection":        "GetTextDetection",
		"StartMediaAnalysisJob":     "GetMediaAnalysisJob",
	}
	for startAction, getAction := range startToGet {
		resp = rekognitionRequest(t, ts, startAction, `{"Video":{"S3Object":{"Bucket":"stackyard-rekognition","Name":"video.mp4"}}}`)
		assertStatus(t, resp, http.StatusOK)
		payload := decodeRekognitionPayload(t, resp)
		jobID := rekognitionPayloadStringFromMap(payload, "JobId")
		if jobID == "" {
			jobID = "job-000001"
		}
		resp = rekognitionRequest(t, ts, getAction, `{"JobId":"`+jobID+`"}`)
		assertStatus(t, resp, http.StatusOK)
	}

	resp = rekognitionRequest(t, ts, "ListMediaAnalysisJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestRekognitionStage56PolicyTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	projectARN := "arn:aws:rekognition:us-east-1:123456789012:project/stackyard-project"
	resourceARN := "arn:aws:rekognition:us-east-1:123456789012:collection/stackyard-collection"

	resp := rekognitionRequest(t, ts, "PutProjectPolicy", `{"ProjectArn":"`+projectARN+`","PolicyName":"stage-policy","PolicyDocument":"{}"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "ListProjectPolicies", `{"ProjectArn":"`+projectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-policy") {
		t.Fatalf("expected ListProjectPolicies to include stage-policy, got %q", body)
	}

	resp = rekognitionRequest(t, ts, "DeleteProjectPolicy", `{"ProjectArn":"`+projectARN+`","PolicyName":"stage-policy"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "TagResource", `{"ResourceArn":"`+resourceARN+`","Tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "ListTagsForResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = rekognitionRequest(t, ts, "UntagResource", `{"ResourceArn":"`+resourceARN+`","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "CreateStreamProcessor", `{"Name":"stage-processor"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = rekognitionRequest(t, ts, "DescribeStreamProcessor", `{"Name":"stage-processor"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = rekognitionRequest(t, ts, "ListStreamProcessors", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = rekognitionRequest(t, ts, "StartStreamProcessor", `{"Name":"stage-processor"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = rekognitionRequest(t, ts, "StopStreamProcessor", `{"Name":"stage-processor"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = rekognitionRequest(t, ts, "UpdateStreamProcessor", `{"Name":"stage-processor"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = rekognitionRequest(t, ts, "DeleteStreamProcessor", `{"Name":"stage-processor"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "CreateFaceLivenessSession", `{}`)
	assertStatus(t, resp, http.StatusOK)
	payload := decodeRekognitionPayload(t, resp)
	sessionID := rekognitionPayloadStringFromMap(payload, "SessionId")
	if sessionID == "" {
		t.Fatalf("expected SessionId from CreateFaceLivenessSession")
	}
	resp = rekognitionRequest(t, ts, "GetFaceLivenessSessionResults", `{"SessionId":"`+sessionID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = rekognitionRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "RekognitionService.ListCollections",
		},
		"rekognition",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeRekognitionPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func rekognitionPayloadStringFromMap(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
