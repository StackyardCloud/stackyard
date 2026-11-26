package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// chimeSDKCall sends a chime-signed REST JSON request (with an optional
// x-amz-chime-bearer) and returns the status, decoded body and error type.
func chimeSDKCall(t *testing.T, ts *httptest.Server, method, path, body, bearer string) (int, map[string]any, string) {
	t.Helper()
	headers := map[string]string{"Content-Type": "application/json"}
	if bearer != "" {
		headers["X-Amz-Chime-Bearer"] = bearer
	}
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "chime")
	return resp.StatusCode, msgDecode(t, resp), resp.Header.Get("X-Amzn-ErrorType")
}

// chimeSDKOK calls and requires status want.
func chimeSDKOK(t *testing.T, ts *httptest.Server, want int, method, path, body, bearer string) map[string]any {
	t.Helper()
	status, out, code := chimeSDKCall(t, ts, method, path, body, bearer)
	if status != want {
		t.Fatalf("%s %s: want %d, got %d %s %v", method, path, want, status, code, out)
	}
	return out
}

// chimeSDKFail calls and requires the error status and type, with a message.
func chimeSDKFail(t *testing.T, ts *httptest.Server, status int, code, method, path, body, bearer string) {
	t.Helper()
	got, out, gotCode := chimeSDKCall(t, ts, method, path, body, bearer)
	if got != status || gotCode != code || out["Message"] == nil {
		t.Fatalf("%s %s: want %d %s, got %d %s %v", method, path, status, code, got, gotCode, out)
	}
}

func chimeSDKEscape(arn string) string { return url.PathEscape(arn) }

func TestChimeSDKStage0CatalogsMatchModels(t *testing.T) {
	check := func(name string, model *awsmodels.Model, ops []chimeOperation, handled func(string) bool) {
		if len(ops) != len(model.Operations) {
			t.Fatalf("%s: catalog has %d operations, model %d", name, len(ops), len(model.Operations))
		}
		for _, op := range ops {
			if _, ok := model.Operations[op.Name]; !ok {
				t.Errorf("%s: %s is not in the model", name, op.Name)
			}
			if !handled(op.Name) {
				t.Errorf("%s: %s has no handler", name, op.Name)
			}
		}
	}
	check("chime-sdk-meetings", awsmodels.ChimeSDKMeetingsModel(), chimeSDKMeetingsOperations, func(op string) bool { return cmtgHandlers[op] != nil })
	check("chime-sdk-messaging", awsmodels.ChimeSDKMessagingModel(), chimeSDKMessagingOperations, func(op string) bool { return cmsgHandlers[op] != nil })
	check("chime-sdk-media-pipelines", awsmodels.ChimeSDKMediaPipelinesModel(), chimeSDKMediaPipelinesOperations, func(op string) bool { return cmplHandlers[op] != nil })
}

// The Chime SDK APIs and the legacy Chime API all sign as chime: each keeps
// its own traffic, and /tags goes to the API owning the ARN.
func TestChimeSDKStage0Routing(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	// Legacy Chime keeps /accounts and /meetings/{id}/dial-outs.
	if status, out, _ := chimeSDKCall(t, ts, http.MethodGet, "/accounts", "", ""); status != http.StatusOK || out["Accounts"] == nil {
		t.Fatalf("Chime ListAccounts: %d %v", status, out)
	}
	if status, out, _ := chimeSDKCall(t, ts, http.MethodPost, "/meetings/meeting-000001/dial-outs", `{"ToPhoneNumber":"+12065550100"}`, ""); status != http.StatusOK {
		t.Fatalf("Chime CreateMeetingDialOut: %d %v", status, out)
	}
	meeting := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, "/meetings", cmtgMeetingBody("route"), "")["Meeting"].(map[string]any)
	if status, out, _ := chimeSDKCall(t, ts, http.MethodGet, "/meetings/"+meeting["MeetingId"].(string), "", ""); status != http.StatusOK || out["Meeting"] == nil {
		t.Fatalf("GetMeeting: %d %v", status, out)
	}
	app := "arn:aws:chime:us-east-1:123456789012:app-instance/route"
	user := app + "/user/alice"
	ch := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/channels", `{"AppInstanceArn":"`+app+`","Name":"general","ClientRequestToken":"tok-1"}`, user)["ChannelArn"].(string)
	pool := chimeSDKOK(t, ts, http.StatusCreated, http.MethodPost, "/media-pipeline-kinesis-video-stream-pools", `{"PoolName":"route","StreamConfiguration":{"Region":"us-east-1"}}`, "")
	poolArn := pool["KinesisVideoStreamPoolConfiguration"].(map[string]any)["PoolArn"].(string)

	for owner, arn := range map[string]string{"meetings": meeting["MeetingArn"].(string), "messaging": ch, "pipelines": poolArn} {
		chimeSDKOK(t, ts, http.StatusNoContent, http.MethodPost, "/tags?operation=tag-resource", `{"ResourceARN":"`+arn+`","Tags":[{"Key":"owner","Value":"`+owner+`"}]}`, "")
		out := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, "/tags?arn="+url.QueryEscape(arn), "", "")
		tags := out["Tags"].([]any)
		if tags[0].(map[string]any)["Key"] != "owner" || tags[0].(map[string]any)["Value"] != owner {
			t.Fatalf("%s tags: %v", owner, out)
		}
	}
	chimeSDKFail(t, ts, http.StatusNotFound, "ResourceNotFoundException", http.MethodGet, "/tags?arn="+url.QueryEscape("arn:aws:chime:us-east-1:123456789012:meeting/missing"), "", "")

	// Requests signed for another service aren't claimed.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/sdk-media-pipelines", nil, nil, "s3")
	if body := string(mustBody(t, resp)); resp.StatusCode == http.StatusOK && strings.Contains(body, "MediaPipelines") {
		t.Fatalf("s3-signed media pipelines request was served: %s", body)
	}
	// Chunked bodies are refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/meetings", io.MultiReader(strings.NewReader(`{}`)))
	req.Host = "meetings-chime.us-east-1.amazonaws.com"
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	// Unknown members and model violations are rejected.
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/meetings", `{"ClientRequestToken":"tok","MediaRegion":"us-east-1","ExternalMeetingId":"x1","Smuggled":1}`, "")
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodGet, "/sdk-media-pipelines/not-a-uuid", "", "")
}
