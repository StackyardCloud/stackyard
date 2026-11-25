package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestIoTDataPlaneStage1ShadowLifecycle(t *testing.T) {
	_, ts := iotdpServer(t)
	const path = "/things/pump-1/shadow"
	up := iotdpOK(t, ts, "iotdata", http.MethodPost, path, `{"state":{"desired":{"temp":20,"mode":{"fan":"on","led":1}},"reported":{"temp":18,"mode":{"fan":"on"}}},"clientToken":"c1"}`)
	if up["version"] != float64(1) || up["clientToken"] != "c1" {
		t.Fatalf("first update: %v", up)
	}
	got := iotdpOK(t, ts, "iotdata", http.MethodGet, path, "")
	delta := got["state"].(map[string]any)["delta"].(map[string]any)
	if delta["temp"] != float64(20) || len(delta["mode"].(map[string]any)) != 1 || delta["mode"].(map[string]any)["led"] != float64(1) {
		t.Fatalf("delta: %v", delta)
	}
	if _, ok := got["metadata"].(map[string]any)["desired"].(map[string]any)["temp"].(map[string]any)["timestamp"]; !ok {
		t.Fatalf("metadata: %v", got["metadata"])
	}
	// A stale version conflicts; the current one is accepted.
	iotdpErr(t, ts, "iotdata", http.MethodPost, path, `{"state":{"reported":{"temp":20}},"version":7}`, http.StatusConflict, "ConflictException")
	iotdpOK(t, ts, "iotdata", http.MethodPost, path, `{"state":{"reported":{"temp":20,"mode":{"led":1}}},"version":1}`)
	got = iotdpOK(t, ts, "iotdata", http.MethodGet, path, "")
	if _, has := got["state"].(map[string]any)["delta"]; has || got["version"] != float64(2) {
		t.Fatalf("converged shadow: %v", got)
	}
	// null removes a key and a whole section.
	iotdpOK(t, ts, "iotdata", http.MethodPost, path, `{"state":{"desired":null,"reported":{"temp":null}}}`)
	state := iotdpOK(t, ts, "iotdata", http.MethodGet, path, "")["state"].(map[string]any)
	if _, has := state["desired"]; has || state["reported"].(map[string]any)["temp"] != nil {
		t.Fatalf("after nulls: %v", state)
	}
	// Bad documents.
	for body, code := range map[string]string{
		`not json`:                    "InvalidRequestException",
		`{"state":{}}`:                "InvalidRequestException",
		`{"state":{"delta":{"a":1}}}`: "InvalidRequestException",
		`{"state":{"desired":1}}`:     "InvalidRequestException",
		`{"stat":{"desired":{}}}`:     "InvalidRequestException",
		`{"state":{"desired":{"a":{"b":{"c":{"d":{"e":{"f":{"g":1}}}}}}}}}`: "InvalidRequestException",
	} {
		iotdpErr(t, ts, "iotdata", http.MethodPost, path, body, http.StatusBadRequest, code)
	}
	big := `{"state":{"desired":{"blob":"` + strings.Repeat("x", 9000) + `"}}}`
	iotdpErr(t, ts, "iotdata", http.MethodPost, path, big, http.StatusRequestEntityTooLarge, "RequestEntityTooLargeException")
	// The merged document is re-checked: two halves that fit alone don't fit together.
	half := strings.Repeat("y", 4500)
	iotdpOK(t, ts, "iotdata", http.MethodPost, path, `{"state":{"desired":{"a":"`+half+`"}}}`)
	iotdpErr(t, ts, "iotdata", http.MethodPost, path, `{"state":{"desired":{"b":"`+half+`"}}}`, http.StatusRequestEntityTooLarge, "RequestEntityTooLargeException")

	// Delete keeps the version: a re-created shadow continues counting.
	del := iotdpOK(t, ts, "iotdata", http.MethodDelete, path, "")
	if del["version"] != float64(4) {
		t.Fatalf("delete: %v", del)
	}
	iotdpErr(t, ts, "iotdata", http.MethodGet, path, "", http.StatusNotFound, "ResourceNotFoundException")
	iotdpErr(t, ts, "iotdata", http.MethodDelete, path, "", http.StatusNotFound, "ResourceNotFoundException")
	if up := iotdpOK(t, ts, "iotdata", http.MethodPost, path, `{"state":{"reported":{"on":true}}}`); up["version"] != float64(5) {
		t.Fatalf("re-created: %v", up)
	}
}

func TestIoTDataPlaneStage1NamedShadows(t *testing.T) {
	_, ts := iotdpServer(t)
	for _, name := range []string{"cfg", "fw", "net"} {
		iotdpOK(t, ts, "iotdata", http.MethodPost, "/things/hub/shadow?name="+name, `{"state":{"reported":{"ok":true}}}`)
	}
	iotdpOK(t, ts, "iotdata", http.MethodPost, "/things/hub/shadow", `{"state":{"reported":{"classic":true}}}`)
	iotdpErr(t, ts, "iotdata", http.MethodPost, "/things/hub/shadow?name=bad%20name", `{"state":{"reported":{}}}`, http.StatusBadRequest, "InvalidRequestException")
	page := iotdpOK(t, ts, "iotdata", http.MethodGet, "/api/things/shadow/ListNamedShadowsForThing/hub?pageSize=2", "")
	if len(page["results"].([]any)) != 2 || page["nextToken"] != "2" {
		t.Fatalf("page 1: %v", page)
	}
	page = iotdpOK(t, ts, "iotdata", http.MethodGet, "/api/things/shadow/ListNamedShadowsForThing/hub?pageSize=2&nextToken=2", "")
	if r := page["results"].([]any); len(r) != 1 || r[0] != "net" || page["nextToken"] != nil {
		t.Fatalf("page 2: %v", page)
	}
	// An out-of-range token is clamped, not an index panic.
	page = iotdpOK(t, ts, "iotdata", http.MethodGet, "/api/things/shadow/ListNamedShadowsForThing/hub?nextToken=99", "")
	if len(page["results"].([]any)) != 0 {
		t.Fatalf("clamped page: %v", page)
	}
	iotdpErr(t, ts, "iotdata", http.MethodGet, "/api/things/shadow/ListNamedShadowsForThing/hub?pageSize=500", "", http.StatusBadRequest, "InvalidRequestException")
	if got := iotdpOK(t, ts, "iotdata", http.MethodGet, "/things/hub/shadow?name=fw", ""); got["state"].(map[string]any)["reported"].(map[string]any)["ok"] != true {
		t.Fatalf("named shadow: %v", got)
	}
}

// iotdpTokens holds each simulated client's session token, keyed by server
// URL and client ID, so the inbox helper can authenticate.
var iotdpTokens sync.Map

func iotdpConnect(t *testing.T, ts *httptest.Server, body string) int {
	t.Helper()
	var req struct {
		ClientID string `json:"clientId"`
	}
	_ = json.Unmarshal([]byte(body), &req)
	if tok, ok := iotdpTokens.Load(ts.URL + "|" + req.ClientID); ok {
		body = strings.Replace(body, "{", `{"sessionToken":"`+tok.(string)+`",`, 1)
	}
	resp, err := http.Post(ts.URL+"/_stackyard/iot-data/connections", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Token string `json:"sessionToken"`
	}
	if json.Unmarshal(mustBody(t, resp), &out) == nil && out.Token != "" {
		iotdpTokens.Store(ts.URL+"|"+req.ClientID, out.Token)
	}
	return resp.StatusCode
}

func iotdpInbox(t *testing.T, ts *httptest.Server, client string) []map[string]any {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/_stackyard/iot-data/connections/"+client+"/messages", nil)
	if tok, ok := iotdpTokens.Load(ts.URL + "|" + client); ok {
		req.Header.Set(iotdSessionTokenHeader, tok.(string))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatal(err)
	}
	return out.Messages
}

func TestIoTDataPlaneStage1PublishRetainAndConnections(t *testing.T) {
	_, ts := iotdpServer(t)
	props := base64.StdEncoding.EncodeToString([]byte(`[{"site":"north"}]`))
	status, _, _ := iotdpCall(t, ts, "iotdata", http.MethodPost, "/topics/plant%2Fline1%2Ftemp?qos=1&retain=true", `{"c":21}`,
		map[string]string{"x-amz-mqtt5-user-properties": props})
	if status != http.StatusOK {
		t.Fatalf("publish: %d", status)
	}
	got := iotdpOK(t, ts, "iotdata", http.MethodGet, "/retainedMessage/plant%2Fline1%2Ftemp", "")
	if got["payload"] != base64.StdEncoding.EncodeToString([]byte(`{"c":21}`)) || got["qos"] != float64(1) || got["userProperties"] != props {
		t.Fatalf("retained: %v", got)
	}
	if list := iotdpOK(t, ts, "iotdata", http.MethodGet, "/retainedMessage", "")["retainedTopics"].([]any); len(list) != 1 {
		t.Fatalf("list retained: %v", list)
	}
	for _, bad := range []string{"/topics/a%2F%2B?qos=0", "/topics/a?qos=2", "/topics/%24SYS%2Fx", "/topics/" + strings.Repeat("a", 300)} {
		iotdpErr(t, ts, "iotdata", http.MethodPost, bad, "x", http.StatusBadRequest, "InvalidRequestException")
	}
	status, _, code := iotdpCall(t, ts, "iotdata", http.MethodPost, "/topics/a", "x", map[string]string{"x-amz-mqtt5-user-properties": "not base64!"})
	if status != http.StatusBadRequest || code != "InvalidRequestException" {
		t.Fatalf("bad user properties: %d %s", status, code)
	}

	// A connecting client receives matching retained messages, then live ones.
	if s := iotdpConnect(t, ts, `{"clientId":"dev-1","thingName":"pump-1","cleanSession":false,"subscriptions":[{"topicFilter":"plant/+/temp","qos":1},{"topicFilter":"$aws/things/pump-1/shadow/#","qos":0}]}`); s != http.StatusCreated {
		t.Fatalf("connect: %d", s)
	}
	if s := iotdpConnect(t, ts, `{"clientId":"dev-2","subscriptions":[{"topicFilter":"a/#/b","qos":0}]}`); s != http.StatusBadRequest {
		t.Fatalf("bad filter accepted: %d", s)
	}
	// Knowing a client ID isn't enough: the inbox and session resumption
	// need the token returned at connect.
	if resp, err := http.Get(ts.URL + "/_stackyard/iot-data/connections/dev-1/messages"); err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("inbox without token: %v %v", resp, err)
	}
	if resp, err := http.Post(ts.URL+"/_stackyard/iot-data/connections", "application/json", strings.NewReader(`{"clientId":"dev-1","cleanSession":false,"sessionToken":"guess"}`)); err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("session takeover without token: %v %v", resp, err)
	}
	if msgs := iotdpInbox(t, ts, "dev-1"); len(msgs) != 1 || msgs[0]["topic"] != "plant/line1/temp" {
		t.Fatalf("retained on subscribe: %v", msgs)
	}
	iotdpOK(t, ts, "iotdata", http.MethodPost, "/topics/plant%2Fline2%2Ftemp", `{"c":5}`)
	iotdpOK(t, ts, "iotdata", http.MethodPost, "/topics/plant%2Fline2%2Fhumidity", `{"h":5}`)
	// A shadow update over MQTT is applied and answered on /accepted.
	iotdpOK(t, ts, "iotdata", http.MethodPost, "/topics/%24aws%2Fthings%2Fpump-1%2Fshadow%2Fupdate", `{"state":{"reported":{"rpm":900}}}`)
	msgs := iotdpInbox(t, ts, "dev-1")
	// The # filter sees the update request itself and its answer.
	if len(msgs) != 3 || msgs[0]["topic"] != "plant/line2/temp" || msgs[2]["topic"] != "$aws/things/pump-1/shadow/update/accepted" {
		t.Fatalf("delivered: %v", msgs)
	}
	if rpm := iotdpOK(t, ts, "iotdata", http.MethodGet, "/things/pump-1/shadow", "")["state"].(map[string]any)["reported"].(map[string]any)["rpm"]; rpm != float64(900) {
		t.Fatalf("shadow via MQTT: %v", rpm)
	}

	conn := iotdpOK(t, ts, "iotdata", http.MethodGet, "/connections/dev-1?includeSocketInformation=true", "")
	if conn["connected"] != true || conn["thingName"] != "pump-1" || conn["sourceIp"] == nil {
		t.Fatalf("GetConnection: %v", conn)
	}
	subs := iotdpOK(t, ts, "iotdata", http.MethodGet, "/connections/dev-1/subscriptions?maxResults=1", "")
	if len(subs["subscriptions"].([]any)) != 1 || subs["nextToken"] != "1" {
		t.Fatalf("ListSubscriptions: %v", subs)
	}
	direct := iotdpOK(t, ts, "iotdata", http.MethodPost, "/connections/dev-1/messages?topic=cmd%2Fping", "hi")
	if direct["traceId"] == nil {
		t.Fatalf("SendDirectMessage: %v", direct)
	}
	if msgs := iotdpInbox(t, ts, "dev-1"); len(msgs) != 1 || msgs[0]["topic"] != "cmd/ping" {
		t.Fatalf("direct inbox: %v", msgs)
	}

	// Disconnecting a persistent session keeps it; QoS 1 messages queue.
	iotdpOK(t, ts, "iotdata", http.MethodDelete, "/connections/dev-1", "")
	conn = iotdpOK(t, ts, "iotdata", http.MethodGet, "/connections/dev-1", "")
	if conn["connected"] != false || conn["disconnectReason"] != "API_INITIATED_DISCONNECT" {
		t.Fatalf("after disconnect: %v", conn)
	}
	iotdpErr(t, ts, "iotdata", http.MethodPost, "/connections/dev-1/messages?topic=x", "hi", http.StatusNotFound, "ResourceNotFoundException")
	iotdpOK(t, ts, "iotdata", http.MethodPost, "/topics/plant%2Fline3%2Ftemp?qos=1", "q")
	iotdpOK(t, ts, "iotdata", http.MethodPost, "/topics/plant%2Fline3%2Ftemp?qos=0", "dropped")
	if msgs := iotdpInbox(t, ts, "dev-1"); len(msgs) != 1 {
		t.Fatalf("queued while offline: %v", msgs)
	}
	iotdpErr(t, ts, "iotdata", http.MethodDelete, "/connections/dev-1", "", http.StatusNotFound, "ResourceNotFoundException")
	iotdpConnect(t, ts, `{"clientId":"dev-1","cleanSession":false}`)
	iotdpOK(t, ts, "iotdata", http.MethodDelete, "/connections/dev-1?cleanSession=true", "")
	iotdpErr(t, ts, "iotdata", http.MethodGet, "/connections/dev-1", "", http.StatusNotFound, "ResourceNotFoundException")
	iotdpErr(t, ts, "iotdata", http.MethodGet, "/connections/%24bad", "", http.StatusBadRequest, "InvalidRequestException")

	// An empty retained publish clears the topic.
	iotdpOK(t, ts, "iotdata", http.MethodPost, "/topics/plant%2Fline1%2Ftemp?retain=true", "")
	iotdpErr(t, ts, "iotdata", http.MethodGet, "/retainedMessage/plant%2Fline1%2Ftemp", "", http.StatusNotFound, "ResourceNotFoundException")
}

func TestIoTDataPlaneStage1Bounds(t *testing.T) {
	srv, ts := iotdpServer(t)
	st := srv.iotData
	st.mu.Lock()
	for i := 0; i < iotdMaxRetained; i++ {
		st.retained[string(rune('a'+i%26))+strings.Repeat("x", i/26)] = &iotdMessage{Payload: []byte("1")}
	}
	st.mu.Unlock()
	iotdpErr(t, ts, "iotdata", http.MethodPost, "/topics/new?retain=true", "1", http.StatusTooManyRequests, "ThrottlingException")
	if s := iotdpConnect(t, ts, `{"clientId":"c","subscriptions":[`+strings.TrimSuffix(strings.Repeat(`{"topicFilter":"t","qos":0},`, iotdMaxSubscriptions+1), ",")+`]}`); s != http.StatusBadRequest {
		t.Fatalf("too many subscriptions: %d", s)
	}
	iotdpConnect(t, ts, `{"clientId":"c","subscriptions":[{"topicFilter":"#","qos":0}]}`)
	for i := 0; i < iotdMaxInbox+20; i++ {
		iotdpOK(t, ts, "iotdata", http.MethodPost, "/topics/t", "m")
	}
	if msgs := iotdpInbox(t, ts, "c"); len(msgs) != iotdMaxInbox {
		t.Fatalf("inbox not bounded: %d", len(msgs))
	}
}
