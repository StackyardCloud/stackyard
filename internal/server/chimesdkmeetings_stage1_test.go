package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func cmtgMeetingBody(token string) string {
	return `{"ClientRequestToken":"` + token + `","MediaRegion":"us-east-1","ExternalMeetingId":"ext-` + token + `","Tags":[{"Key":"team","Value":"rtc"}]}`
}

func TestChimeSDKMeetingsStage1(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	out := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, "/meetings", cmtgMeetingBody("m1"), "")
	m := out["Meeting"].(map[string]any)
	id := m["MeetingId"].(string)
	if m["MediaPlacement"].(map[string]any)["SignalingUrl"] == nil || m["MeetingArn"] != "arn:aws:chime:us-east-1:123456789012:meeting/"+id {
		t.Fatalf("CreateMeeting: %v", out)
	}
	// ClientRequestToken makes creates idempotent.
	again := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, "/meetings", cmtgMeetingBody("m1"), "")
	if again["Meeting"].(map[string]any)["MeetingId"] != id {
		t.Fatalf("idempotent create: %v", again)
	}
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/meetings", `{"ClientRequestToken":"m2","MediaRegion":"mars-1","ExternalMeetingId":"x2"}`, "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodPost, "/meetings", `{"ClientRequestToken":"m3","MediaRegion":"us-east-1","ExternalMeetingId":"x3","PrimaryMeetingId":"`+fakeUUID(1)+`"}`, "")

	base := "/meetings/" + id + "/attendees"
	att := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, base, `{"ExternalUserId":"alice-1"}`, "")["Attendee"].(map[string]any)
	if att["JoinToken"] == nil || att["Capabilities"].(map[string]any)["Video"] != "SendReceive" {
		t.Fatalf("CreateAttendee: %v", att)
	}
	aid := att["AttendeeId"].(string)
	// Join tokens are bearer credentials: only creates return them.
	got := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, base+"/"+aid, "", "")["Attendee"].(map[string]any)
	if got["JoinToken"] != nil || got["ExternalUserId"] != "alice-1" {
		t.Fatalf("GetAttendee: %v", got)
	}
	// Content needs video.
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPut, base+"/"+aid+"/capabilities",
		`{"Capabilities":{"Audio":"SendReceive","Video":"None","Content":"Receive"}}`, "")
	upd := chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, base+"/"+aid+"/capabilities", `{"Capabilities":{"Audio":"Receive","Video":"Receive","Content":"None"}}`, "")
	if upd["Attendee"].(map[string]any)["Capabilities"].(map[string]any)["Audio"] != "Receive" {
		t.Fatalf("UpdateAttendeeCapabilities: %v", upd)
	}
	batch := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, base+"?operation=batch-create",
		`{"Attendees":[{"ExternalUserId":"bob-1"},{"ExternalUserId":"carol-1","Capabilities":{"Audio":"None","Video":"None","Content":"SendReceive"}}]}`, "")
	if len(batch["Attendees"].([]any)) != 1 || len(batch["Errors"].([]any)) != 1 {
		t.Fatalf("BatchCreateAttendee: %v", batch)
	}
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPut, base+"/capabilities?operation=batch-update-except",
		`{"ExcludedAttendeeIds":[{"AttendeeId":"`+aid+`"}],"Capabilities":{"Audio":"None","Video":"None","Content":"None"}}`, "")
	page := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, base+"?max-results=1", "", "")
	rest := chimeSDKOK(t, ts, http.StatusOK, http.MethodGet, base+"?max-results=1&next-token="+page["NextToken"].(string), "", "")
	if len(page["Attendees"].([]any)) != 1 || len(rest["Attendees"].([]any)) != 1 || rest["NextToken"] != nil {
		t.Fatalf("ListAttendees paging: %v / %v", page, rest)
	}
	for _, a := range append(page["Attendees"].([]any), rest["Attendees"].([]any)...) {
		a := a.(map[string]any)
		want := "None"
		if a["AttendeeId"] == aid {
			want = "Receive"
		}
		if a["Capabilities"].(map[string]any)["Audio"] != want || a["JoinToken"] != nil {
			t.Fatalf("batch update except: %v", a)
		}
	}
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodGet, base+"?max-results=1000", "", "")

	// MeetingFeatures cap attendees and disable video.
	small := chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, "/meetings?operation=create-attendees",
		`{"ClientRequestToken":"small","MediaRegion":"eu-west-1","ExternalMeetingId":"small","MeetingFeatures":{"Video":{"MaxResolution":"None"},"Attendee":{"MaxCount":1}},
		"Attendees":[{"ExternalUserId":"one-1"},{"ExternalUserId":"two-2"}]}`, "")
	if len(small["Attendees"].([]any)) != 1 || small["Errors"].([]any)[0].(map[string]any)["ErrorCode"] != "ResourceLimitExceeded" {
		t.Fatalf("CreateMeetingWithAttendees: %v", small)
	}
	if small["Attendees"].([]any)[0].(map[string]any)["Capabilities"].(map[string]any)["Video"] != "None" {
		t.Fatalf("video disabled: %v", small)
	}
	sid := small["Meeting"].(map[string]any)["MeetingId"].(string)
	chimeSDKFail(t, ts, http.StatusBadRequest, "LimitExceededException", http.MethodPost, "/meetings/"+sid+"/attendees", `{"ExternalUserId":"three-3"}`, "")

	// Transcription runs once at a time.
	tr := "/meetings/" + id + "/transcription"
	cfg := `{"TranscriptionConfiguration":{"EngineTranscribeSettings":{"LanguageCode":"en-US"}}}`
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, tr+"?operation=start", `{"TranscriptionConfiguration":{}}`, "")
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, tr+"?operation=start", `{"TranscriptionConfiguration":{"EngineTranscribeSettings":{"LanguageCode":"en-US","IdentifyLanguage":true}}}`, "")
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, tr+"?operation=start", cfg, "")
	chimeSDKFail(t, ts, http.StatusConflict, "ConflictException", http.MethodPost, tr+"?operation=start", cfg, "")
	chimeSDKOK(t, ts, http.StatusOK, http.MethodPost, tr+"?operation=stop", "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodPost, tr+"?operation=stop", "", "")

	// Tags.
	arn := m["MeetingArn"].(string)
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodPost, "/tags?operation=untag-resource", `{"ResourceARN":"`+arn+`","TagKeys":["team"]}`, "")
	var many []string
	for i := 0; i < 51; i++ {
		many = append(many, fmt.Sprintf(`{"Key":"k%d","Value":"v"}`, i))
	}
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/tags?operation=tag-resource", `{"ResourceARN":"`+arn+`","Tags":[`+strings.Join(many, ",")+`]}`, "")
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodPost, "/tags?operation=tag-resource", `{"ResourceARN":"`+arn+`","Tags":[`+strings.Join(many[:50], ",")+`]}`, "")
	// The cap is re-checked against the merged tags.
	chimeSDKFail(t, ts, http.StatusBadRequest, "TooManyTagsException", http.MethodPost, "/tags?operation=tag-resource", `{"ResourceARN":"`+arn+`","Tags":[{"Key":"x","Value":"y"}]}`, "")
	chimeSDKFail(t, ts, http.StatusBadRequest, "BadRequestException", http.MethodPost, "/tags?operation=tag-resource", `{"ResourceARN":"`+arn+`","Tags":[{"Key":"aws:x","Value":"y"}]}`, "")

	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, base+"/"+aid, "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, base+"/"+aid, "", "")
	chimeSDKOK(t, ts, http.StatusNoContent, http.MethodDelete, "/meetings/"+id, "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodGet, "/meetings/"+id, "", "")
	chimeSDKFail(t, ts, http.StatusNotFound, "NotFoundException", http.MethodPost, base, `{"ExternalUserId":"late-1"}`, "")
}

func fakeUUID(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }
