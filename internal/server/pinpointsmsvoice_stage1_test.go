package server

import (
	"net/http"
	"testing"
)

func TestPinpointSMSVoiceStage1(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	call := func(method, path, body string) (int, map[string]any) {
		return msgREST(t, ts, "sms-voice", method, path, body)
	}
	expect := func(method, path, body string, status int, code string) {
		t.Helper()
		resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "sms-voice")
		out := msgDecode(t, resp)
		if resp.StatusCode != status || resp.Header.Get("X-Amzn-ErrorType") != code || out["Message"] == nil {
			t.Fatalf("%s %s: want %d %s, got %d %s %v", method, path, status, code, resp.StatusCode, resp.Header.Get("X-Amzn-ErrorType"), out)
		}
	}
	const sets = "/v1/sms-voice/configuration-sets"
	if status, out := call(http.MethodPost, sets, `{"ConfigurationSetName":"alerts"}`); status != http.StatusOK {
		t.Fatalf("CreateConfigurationSet: %d %v", status, out)
	}
	expect(http.MethodPost, sets, `{"ConfigurationSetName":"alerts"}`, http.StatusConflict, "AlreadyExistsException")
	expect(http.MethodPost, sets, `{"ConfigurationSetName":"bad name"}`, http.StatusBadRequest, "BadRequestException")
	call(http.MethodPost, sets, `{"ConfigurationSetName":"backup"}`)
	_, page := call(http.MethodGet, sets+"?PageSize=1", "")
	_, rest := call(http.MethodGet, sets+"?PageSize=1&NextToken="+page["NextToken"].(string), "")
	if len(page["ConfigurationSets"].([]any)) != 1 || rest["ConfigurationSets"].([]any)[0] != "backup" || rest["NextToken"] != nil {
		t.Fatalf("paging: %v / %v", page, rest)
	}
	expect(http.MethodGet, sets+"?PageSize=0", "", http.StatusBadRequest, "BadRequestException")

	dest := `{"EventDestinationName":"logs","EventDestination":{"Enabled":true,"MatchingEventTypes":["ANSWERED","FAILED"],
		"CloudWatchLogsDestination":{"IamRoleArn":"arn:aws:iam::123456789012:role/voice","LogGroupArn":"arn:aws:logs:us-east-1:123456789012:log-group:voice"}}}`
	if status, out := call(http.MethodPost, sets+"/alerts/event-destinations", dest); status != http.StatusOK {
		t.Fatalf("CreateConfigurationSetEventDestination: %d %v", status, out)
	}
	expect(http.MethodPost, sets+"/alerts/event-destinations", dest, http.StatusConflict, "AlreadyExistsException")
	expect(http.MethodPost, sets+"/missing/event-destinations", dest, http.StatusNotFound, "NotFoundException")
	expect(http.MethodPost, sets+"/alerts/event-destinations", `{"EventDestinationName":"empty","EventDestination":{"Enabled":true}}`, http.StatusBadRequest, "BadRequestException")
	if status, _ := call(http.MethodPut, sets+"/alerts/event-destinations/logs", `{"EventDestination":{"Enabled":false,"MatchingEventTypes":["BUSY"],"SnsDestination":{"TopicArn":"arn:aws:sns:us-east-1:123456789012:voice"}}}`); status != http.StatusOK {
		t.Fatalf("UpdateConfigurationSetEventDestination: %d", status)
	}
	expect(http.MethodPut, sets+"/alerts/event-destinations/nope", `{"EventDestination":{"SnsDestination":{"TopicArn":"x"}}}`, http.StatusNotFound, "NotFoundException")
	_, got := call(http.MethodGet, sets+"/alerts/event-destinations", "")
	d := got["EventDestinations"].([]any)[0].(map[string]any)
	if d["Name"] != "logs" || d["Enabled"] != false || d["SnsDestination"] == nil || d["CloudWatchLogsDestination"] != nil {
		t.Fatalf("event destinations: %v", got)
	}

	voice := `{"OriginationPhoneNumber":"+12065550100","DestinationPhoneNumber":"+12065550199","ConfigurationSetName":"alerts",
		"Content":{"SSMLMessage":{"LanguageCode":"en-US","VoiceId":"Joanna","Text":"<speak>Your code is <say-as interpret-as=\"digits\">1234</say-as></speak>"}}}`
	if status, out := call(http.MethodPost, "/v1/sms-voice/voice/message", voice); status != http.StatusOK || out["MessageId"] == nil {
		t.Fatalf("SendVoiceMessage: %d %v", status, out)
	}
	expect(http.MethodPost, "/v1/sms-voice/voice/message", `{"OriginationPhoneNumber":"+12065550100","DestinationPhoneNumber":"555","Content":{"PlainTextMessage":{"Text":"hi"}}}`, http.StatusBadRequest, "BadRequestException")
	expect(http.MethodPost, "/v1/sms-voice/voice/message", `{"OriginationPhoneNumber":"+12065550100","DestinationPhoneNumber":"+12065550199","Content":{"SSMLMessage":{"Text":"no speak tag"}}}`, http.StatusBadRequest, "BadRequestException")
	expect(http.MethodPost, "/v1/sms-voice/voice/message", `{"OriginationPhoneNumber":"+12065550100","DestinationPhoneNumber":"+12065550199","Content":{"PlainTextMessage":{"Text":"a"},"CallInstructionsMessage":{"Text":"b"}}}`, http.StatusBadRequest, "BadRequestException")
	expect(http.MethodPost, "/v1/sms-voice/voice/message", `{"OriginationPhoneNumber":"+12065550100","DestinationPhoneNumber":"+12065550199","ConfigurationSetName":"missing","Content":{"PlainTextMessage":{"Text":"hi"}}}`, http.StatusNotFound, "NotFoundException")

	if status, _ := call(http.MethodDelete, sets+"/alerts/event-destinations/logs", ""); status != http.StatusOK {
		t.Fatalf("DeleteConfigurationSetEventDestination: %d", status)
	}
	expect(http.MethodDelete, sets+"/alerts/event-destinations/logs", "", http.StatusNotFound, "NotFoundException")
	if status, _ := call(http.MethodDelete, sets+"/alerts", ""); status != http.StatusOK {
		t.Fatalf("DeleteConfigurationSet: %d", status)
	}
	expect(http.MethodGet, sets+"/alerts/event-destinations", "", http.StatusNotFound, "NotFoundException")
}
