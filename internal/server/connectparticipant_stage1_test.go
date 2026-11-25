package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// cnpCall sends a participant call signed for execute-api, as the CLI does.
func cnpCall(t *testing.T, ts *httptest.Server, method, path, bearer, body string) (int, map[string]any) {
	t.Helper()
	return cnxDo(t, ts, "execute-api", method, path, body, map[string]string{"X-Amz-Bearer": bearer})
}

func cnpOK(t *testing.T, ts *httptest.Server, path, bearer, body string) map[string]any {
	t.Helper()
	status, out := cnpCall(t, ts, http.MethodPost, path, bearer, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", path, status, out)
	}
	return out
}

// cnpStartChat starts a chat through Connect and connects its customer.
func cnpStartChat(t *testing.T, ts *httptest.Server, initial string) (contactID, connToken string) {
	t.Helper()
	status, out := cnxDo(t, ts, "connect", http.MethodPut, "/contact/chat", fmt.Sprintf(
		`{"InstanceId":"instance-000001","ContactFlowId":"contact-flow-000001","ParticipantDetails":{"DisplayName":"Ann"},"InitialMessage":{"ContentType":"text/plain","Content":%q}}`, initial), nil)
	if status != http.StatusOK {
		t.Fatalf("StartChatContact: %d %v", status, out)
	}
	conn := cnpOK(t, ts, "/participant/connection", out["ParticipantToken"].(string), `{"Type":["CONNECTION_CREDENTIALS"],"ConnectParticipant":true}`)
	return out["ContactId"].(string), conn["ConnectionCredentials"].(map[string]any)["ConnectionToken"].(string)
}

func TestConnectParticipantStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.ConnectParticipantModel()
	if len(connectParticipantOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(connectParticipantOperations), len(model.Operations))
	}
	for _, op := range connectParticipantOperations {
		if _, ok := model.Operations[op.Name]; !ok || cnpHandlers[op.Name] == nil {
			t.Errorf("%s: in model %v, handled %v", op.Name, ok, cnpHandlers[op.Name] != nil)
		}
	}
}

func TestConnectParticipantStage0Routing(t *testing.T) {
	_, ts := newConnectTestServer(t)
	// Bearer tokens are the credential; a bad one is refused, missing is invalid.
	if status, out := cnpCall(t, ts, http.MethodPost, "/participant/transcript", "bogus", `{}`); status != http.StatusForbidden || out["__type"] != "AccessDeniedException" {
		t.Fatalf("bad token: %d %v", status, out)
	}
	if status, _ := cnxDo(t, ts, "execute-api", http.MethodPost, "/participant/transcript", `{}`, nil); status != http.StatusBadRequest {
		t.Fatalf("missing token: %d", status)
	}
	// Unsigned with a bearer token (as browser chat clients send) is claimed too.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/participant/transcript", strings.NewReader(`{}`))
	req.Header.Set("X-Amz-Bearer", "bogus")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unsigned bearer call: %d", resp.StatusCode)
	}
	// API Gateway Management (also execute-api) keeps /@connections.
	if status, out := cnxDo(t, ts, "execute-api", http.MethodGet, "/@connections/nope", "", nil); status != http.StatusGone {
		t.Fatalf("apigatewaymanagementapi lost /@connections: %d %v", status, out)
	}
	// S3 keeps /participant/... object paths signed for s3.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/participant/connection", nil, nil, "s3")
	body := mustBody(t, resp)
	if strings.Contains(string(body), "AccessDeniedException") || strings.Contains(string(body), "X-Amz-Bearer") {
		t.Fatalf("s3 path claimed by participant: %d %s", resp.StatusCode, body)
	}
	// Connect keeps its own traffic.
	if status, out := cnxDo(t, ts, "connect", http.MethodGet, "/instance", "", nil); status != http.StatusOK || out["InstanceSummaryList"] == nil {
		t.Fatalf("connect ListInstances: %d %v", status, out)
	}
}

func TestConnectParticipantStage1ChatFlow(t *testing.T) {
	srv, ts := newConnectTestServer(t)
	status, out := cnxDo(t, ts, "connect", http.MethodPut, "/contact/chat", `{"InstanceId":"nope","ContactFlowId":"f","ParticipantDetails":{"DisplayName":"Ann"}}`, nil)
	if status != http.StatusNotFound {
		t.Fatalf("unknown instance: %d %v", status, out)
	}
	status, _ = cnxDo(t, ts, "connect", http.MethodPut, "/contact/chat", `{"InstanceId":"instance-000001","ContactFlowId":"f"}`, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("missing display name: %d", status)
	}
	contact, conn := cnpStartChat(t, ts, "hello")

	// A connection token isn't a participant token.
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/connection", conn, `{"Type":["CONNECTION_CREDENTIALS"]}`); status != http.StatusForbidden {
		t.Fatalf("connection token used as participant token: %d", status)
	}
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/message", conn, `{"ContentType":"text/html","Content":"<b>x</b>"}`); status != http.StatusBadRequest {
		t.Fatalf("html content type: %d", status)
	}
	m1 := cnpOK(t, ts, "/participant/message", conn, `{"ContentType":"text/plain","Content":"second","ClientToken":"c1"}`)
	if again := cnpOK(t, ts, "/participant/message", conn, `{"ContentType":"text/plain","Content":"second","ClientToken":"c1"}`); again["Id"] != m1["Id"] {
		t.Fatalf("client token: %v vs %v", again, m1)
	}

	// A bot joins through Connect's CreateParticipant and reads the message.
	status, bot := cnxDo(t, ts, "connect", http.MethodPost, "/contact/create-participant", fmt.Sprintf(
		`{"InstanceId":"instance-000001","ContactId":%q,"ParticipantDetails":{"ParticipantRole":"CUSTOM_BOT","DisplayName":"Bot"}}`, contact), nil)
	if status != http.StatusOK {
		t.Fatalf("CreateParticipant: %d %v", status, bot)
	}
	botConn := cnpOK(t, ts, "/participant/connection", bot["ParticipantCredentials"].(map[string]any)["ParticipantToken"].(string), `{"Type":["CONNECTION_CREDENTIALS"]}`)["ConnectionCredentials"].(map[string]any)["ConnectionToken"].(string)
	cnpOK(t, ts, "/participant/event", botConn, fmt.Sprintf(`{"ContentType":"application/vnd.amazonaws.connect.event.message.read","Content":%q}`, fmt.Sprintf(`{"messageId":%q}`, m1["Id"])))
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/event", conn, fmt.Sprintf(`{"ContentType":"application/vnd.amazonaws.connect.event.message.read","Content":%q}`, fmt.Sprintf(`{"messageId":%q}`, m1["Id"]))); status != http.StatusConflict {
		t.Fatalf("receipt for own message: %d", status)
	}

	// Transcript paging forward, with receipts and without any token.
	page := cnpOK(t, ts, "/participant/transcript", conn, `{"ScanDirection":"FORWARD","SortOrder":"ASCENDING","MaxResults":3}`)
	items := page["Transcript"].([]any)
	if len(items) != 3 || items[0].(map[string]any)["Type"] != "PARTICIPANT_JOINED" || items[1].(map[string]any)["Content"] != "hello" || page["NextToken"] != "3" {
		t.Fatalf("first page: %v", page)
	}
	rest := cnpOK(t, ts, "/participant/transcript", conn, `{"ScanDirection":"FORWARD","NextToken":"3","MaxResults":100}`)
	raw, _ := json.Marshal(rest)
	if !strings.Contains(string(raw), "MESSAGE_READ") || !strings.Contains(string(raw), "CUSTOM_BOT") {
		t.Fatalf("rest of transcript: %s", raw)
	}
	all, _ := json.Marshal(cnpOK(t, ts, "/participant/transcript", conn, `{"MaxResults":100}`))
	if !strings.Contains(string(all), `"ReadTimestamp"`) {
		t.Fatalf("the bot's read receipt should be on the message: %s", all)
	}
	srv.connectParticipant.mu.Lock()
	chat := srv.connectParticipant.chats[contact]
	tokens := []string{chat.participants[0].partToken, chat.participants[0].connToken, chat.participants[1].connToken}
	srv.connectParticipant.mu.Unlock()
	_, described := cnxDo(t, ts, "connect", http.MethodGet, "/contacts/instance-000001/"+contact, "", nil)
	desc, _ := json.Marshal(described)
	for _, tok := range tokens {
		if strings.Contains(string(all), tok) || strings.Contains(string(desc), tok) {
			t.Fatalf("a bearer token leaked into a transcript or describe")
		}
	}
	if !strings.Contains(string(desc), `"CHAT"`) {
		t.Fatalf("DescribeContact: %s", desc)
	}
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/transcript", conn, `{"NextToken":"99999999999"}`); status != http.StatusBadRequest {
		t.Fatalf("bad next token: %d", status)
	}

	// Authentication sessions and views.
	sess := `"SessionId":"123e4567-e89b-12d3-a456-426614174000"`
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/cancel-authentication", conn, `{`+sess+`}`); status != http.StatusBadRequest {
		t.Fatalf("cancel without session: %d", status)
	}
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/authentication-url", conn, `{`+sess+`,"RedirectUri":"javascript:alert(1)"}`); status != http.StatusBadRequest {
		t.Fatalf("bad redirect: %d", status)
	}
	u := cnpOK(t, ts, "/participant/authentication-url", conn, `{`+sess+`,"RedirectUri":"https://example.com/cb"}`)["AuthenticationUrl"].(string)
	if !strings.Contains(u, "state=123e4567") {
		t.Fatalf("auth url: %s", u)
	}
	cnpOK(t, ts, "/participant/cancel-authentication", conn, `{`+sess+`}`)
	if status, view := cnpCall(t, ts, http.MethodGet, "/participant/views/arn%3Aaws%3Aconnect%3Aus-east-1%3Aaws%3Aview%2Fform", conn, ""); status != http.StatusOK || view["View"].(map[string]any)["Name"] != "Form" {
		t.Fatalf("DescribeView: %d %v", status, view)
	}
	if status, _ := cnpCall(t, ts, http.MethodGet, "/participant/views/nope", conn, ""); status != http.StatusNotFound {
		t.Fatalf("unknown view: %d", status)
	}

	// The customer leaving ends the chat; its token stops working.
	cnpOK(t, ts, "/participant/disconnect", conn, `{"ClientToken":"bye"}`)
	cnpOK(t, ts, "/participant/disconnect", conn, `{"ClientToken":"bye"}`) // idempotent retry
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/disconnect", conn, `{"ClientToken":"other"}`); status != http.StatusForbidden {
		t.Fatalf("disconnect after disconnect: %d", status)
	}
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/message", conn, `{"ContentType":"text/plain","Content":"x"}`); status != http.StatusForbidden {
		t.Fatalf("message after disconnect: %d", status)
	}
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/message", botConn, `{"ContentType":"text/plain","Content":"x"}`); status != http.StatusForbidden {
		t.Fatalf("message after chat ended: %d", status)
	}
	ended := cnpOK(t, ts, "/participant/transcript", botConn, `{"MaxResults":1}`)["Transcript"].([]any)
	if ended[0].(map[string]any)["Type"] != "CHAT_ENDED" {
		t.Fatalf("last item: %v", ended)
	}
}

func TestConnectParticipantStage1Attachments(t *testing.T) {
	_, ts := newConnectTestServer(t)
	_, conn := cnpStartChat(t, ts, "hi")
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/start-attachment-upload", conn, `{"ContentType":"text/html","AttachmentSizeInBytes":5,"AttachmentName":"a.html","ClientToken":"t"}`); status != http.StatusBadRequest {
		t.Fatalf("html attachment: %d", status)
	}
	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/start-attachment-upload", conn, fmt.Sprintf(`{"ContentType":"text/plain","AttachmentSizeInBytes":%d,"AttachmentName":"a.txt","ClientToken":"t"}`, cnpMaxAttachment+1)); status != http.StatusBadRequest {
		t.Fatalf("oversized attachment: %d", status)
	}
	start := cnpOK(t, ts, "/participant/start-attachment-upload", conn, `{"ContentType":"text/plain","AttachmentSizeInBytes":5,"AttachmentName":"a.txt","ClientToken":"t"}`)
	id := start["AttachmentId"].(string)
	upload := start["UploadMetadata"].(map[string]any)["Url"].(string)

	if status, _ := cnpCall(t, ts, http.MethodPost, "/participant/complete-attachment-upload", conn, fmt.Sprintf(`{"AttachmentIds":[%q],"ClientToken":"c"}`, id)); status != http.StatusConflict {
		t.Fatalf("complete before upload: %d", status)
	}
	put := func(u, body, ct string) int {
		req, _ := http.NewRequest(http.MethodPut, u, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", ct)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if s := put(strings.Replace(upload, "key=", "key=x", 1), "hello", "text/plain"); s != http.StatusForbidden {
		t.Fatalf("wrong key: %d", s)
	}
	if s := put(upload, "hello!", "text/plain"); s != http.StatusBadRequest {
		t.Fatalf("wrong size: %d", s)
	}
	if s := put(upload, "hello", "text/plain"); s != http.StatusOK {
		t.Fatalf("upload: %d", s)
	}
	if s := put(upload, "hello", "text/plain"); s != http.StatusForbidden {
		t.Fatalf("second upload: %d", s)
	}
	cnpOK(t, ts, "/participant/complete-attachment-upload", conn, fmt.Sprintf(`{"AttachmentIds":[%q],"ClientToken":"c"}`, id))
	got := cnpOK(t, ts, "/participant/attachment", conn, fmt.Sprintf(`{"AttachmentId":%q,"UrlExpiryInSeconds":60}`, id))
	if got["AttachmentSizeInBytes"].(float64) != 5 {
		t.Fatalf("GetAttachment: %v", got)
	}
	resp, err := http.Get(got["Url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(data) != "hello" || resp.Header.Get("Content-Type") != "application/octet-stream" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("download: %q %v", data, resp.Header)
	}
	raw, _ := json.Marshal(cnpOK(t, ts, "/participant/transcript", conn, `{}`))
	if !strings.Contains(string(raw), `"ATTACHMENT"`) || !strings.Contains(string(raw), "a.txt") {
		t.Fatalf("attachment item: %s", raw)
	}
}

// The transcript is bounded per chat; more items are refused, not stored.
func TestConnectParticipantStage1TranscriptLimit(t *testing.T) {
	srv, ts := newConnectTestServer(t)
	_, conn := cnpStartChat(t, ts, "hi")
	big := strings.Repeat("y", 16000)
	refused := false
	for i := 0; i < cnpMaxChatBytes/16000+2; i++ {
		status, _ := cnpCall(t, ts, http.MethodPost, "/participant/message", conn, fmt.Sprintf(`{"ContentType":"text/plain","Content":%q}`, big))
		if status == http.StatusPaymentRequired {
			refused = true
			break
		}
	}
	srv.connectParticipant.mu.Lock()
	bytes := srv.connectParticipant.bytes
	srv.connectParticipant.mu.Unlock()
	if !refused || bytes > cnpMaxChatBytes {
		t.Fatalf("refused=%v stored=%d", refused, bytes)
	}
}
