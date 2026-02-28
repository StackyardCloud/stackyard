package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCleanRoomsMLStage123456LifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	membershipID := "mem-stage-001"
	collaborationID := "col-stage-001"
	trainingDatasetARN := "arn:aws:cleanrooms-ml:us-east-1:123456789012:training-dataset/td-stage-001"
	audienceModelARN := "arn:aws:cleanrooms-ml:us-east-1:123456789012:audience-model/am-stage-001"
	configuredAudienceModelARN := "arn:aws:cleanrooms-ml:us-east-1:123456789012:configured-audience-model/cam-stage-001"
	trainedModelARN := "arn:aws:cleanrooms-ml:us-east-1:123456789012:trained-model/tm-stage-001"
	resourceARN := configuredAudienceModelARN

	cases := []struct {
		name    string
		method  string
		path    string
		payload string
	}{
		{name: "CreateTrainingDataset", method: http.MethodPost, path: "/training-dataset", payload: `{"name":"stage-training-dataset"}`},
		{name: "GetTrainingDataset", method: http.MethodGet, path: "/training-dataset/" + url.PathEscape(trainingDatasetARN), payload: ``},
		{name: "CreateAudienceModel", method: http.MethodPost, path: "/audience-model", payload: `{"name":"stage-audience-model","trainingDatasetArn":"` + trainingDatasetARN + `"}`},
		{name: "GetAudienceModel", method: http.MethodGet, path: "/audience-model/" + url.PathEscape(audienceModelARN), payload: ``},
		{name: "CreateConfiguredAudienceModel", method: http.MethodPost, path: "/configured-audience-model", payload: `{"name":"stage-configured-audience-model","audienceModelArn":"` + audienceModelARN + `"}`},
		{name: "GetConfiguredAudienceModel", method: http.MethodGet, path: "/configured-audience-model/" + url.PathEscape(configuredAudienceModelARN), payload: ``},
		{name: "PutMLConfiguration", method: http.MethodPut, path: "/memberships/" + url.PathEscape(membershipID) + "/ml-configurations", payload: `{"defaultOutputLocation":{"s3":{"bucket":"stage-bucket","keyPrefix":"cleanrooms-ml/"}}}`},
		{name: "CreateMLInputChannel", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/ml-input-channels", payload: `{"name":"stage-input-channel"}`},
		{name: "CreateTrainedModel", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/trained-models", payload: `{"name":"stage-trained-model"}`},
		{name: "GetTrainedModel", method: http.MethodGet, path: "/memberships/" + url.PathEscape(membershipID) + "/trained-models/" + url.PathEscape(trainedModelARN) + "?versionIdentifier=1", payload: ``},
		{name: "StartTrainedModelInferenceJob", method: http.MethodPost, path: "/memberships/" + url.PathEscape(membershipID) + "/trained-model-inference-jobs", payload: `{"trainedModelArn":"` + trainedModelARN + `","name":"stage-inference"}`},
		{name: "ListCollaborationTrainedModels", method: http.MethodGet, path: "/collaborations/" + url.PathEscape(collaborationID) + "/trained-models", payload: ``},
		{name: "TagResource", method: http.MethodPost, path: "/tags/" + url.PathEscape(resourceARN), payload: `{"tags":{"env":"stage","owner":"tests"}}`},
		{name: "ListTagsForResource", method: http.MethodGet, path: "/tags/" + url.PathEscape(resourceARN), payload: ``},
		{name: "UntagResource", method: http.MethodDelete, path: "/tags/" + url.PathEscape(resourceARN) + "?tagKeys=owner", payload: ``},
	}

	for _, tc := range cases {
		resp := cleanRoomsMLRequest(t, ts, tc.method, tc.path, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.name, resp.StatusCode, body)
		}
		if strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s: expected non-NotImplemented response, got: %s", tc.name, body)
		}
	}
}
