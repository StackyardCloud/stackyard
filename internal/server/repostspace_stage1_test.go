package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func rpsOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	status, out := collabCall(t, ts, "repostspace", method, path, body, nil)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func TestRepostSpaceLifecycle(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	srv.repostSpace.mu.Lock()
	srv.repostSpace.delay = 300 * time.Millisecond
	srv.repostSpace.mu.Unlock()
	id := rpsOK(t, ts, http.MethodPost, "/spaces", `{"name":"Eng","subdomain":"eng-help","tier":"BASIC","tags":{"team":"eng"}}`)["spaceId"].(string)
	if status, out := collabCall(t, ts, "repostspace", http.MethodPost, "/spaces", `{"name":"Dup","subdomain":"eng-help","tier":"BASIC"}`, nil); status != http.StatusConflict || out["resourceId"] != id {
		t.Fatalf("duplicate subdomain: %d %v", status, out)
	}
	sp := rpsOK(t, ts, http.MethodGet, "/spaces/"+id, "")
	if sp["status"] != "CREATING" || sp["configurationStatus"] != "UNCONFIGURED" || sp["storageLimit"] != float64(rpsBasicStorage) {
		t.Fatalf("new space: %v", sp)
	}
	if status, _ := collabCall(t, ts, "repostspace", http.MethodPost, "/spaces/"+id+"/invite", `{"accessorIds":["u1"],"title":"Join","body":"Welcome"}`, nil); status != http.StatusConflict {
		t.Fatalf("invite while creating: %d", status)
	}
	time.Sleep(350 * time.Millisecond)
	rpsOK(t, ts, http.MethodPost, "/spaces/"+id+"/admins/admin-1", "")
	rpsOK(t, ts, http.MethodPut, "/spaces/"+id, `{"tier":"STANDARD","description":"Engineering Q&A"}`)
	added := rpsOK(t, ts, http.MethodPost, "/spaces/"+id+"/roles", `{"accessorIds":["u1","u2"],"role":"EXPERT"}`)
	if len(added["addedAccessorIds"].([]any)) != 2 {
		t.Fatalf("BatchAddRole: %v", added)
	}
	rpsOK(t, ts, http.MethodPatch, "/spaces/"+id+"/roles", `{"accessorIds":["u2"],"role":"EXPERT"}`)
	sp = rpsOK(t, ts, http.MethodGet, "/spaces/"+id, "")
	if sp["status"] != "CREATED" || sp["configurationStatus"] != "CONFIGURED" || sp["tier"] != "STANDARD" || sp["userCount"] != float64(2) {
		t.Fatalf("configured space: %v", sp)
	}
	if roles := sp["roles"].(map[string]any); len(roles) != 1 || sp["userAdmins"].([]any)[0] != "admin-1" {
		t.Fatalf("roles/admins: %v", sp)
	}
	rpsOK(t, ts, http.MethodPost, "/spaces/"+id+"/invite", `{"accessorIds":["u1"],"title":"Join","body":"Welcome"}`)
	// Channels provision like spaces.
	ch := rpsOK(t, ts, http.MethodPost, "/spaces/"+id+"/channels", `{"channelName":"general"}`)["channelId"].(string)
	if len(ch) != 24 {
		t.Fatalf("channel id %q", ch)
	}
	if status, _ := collabCall(t, ts, "repostspace", http.MethodPost, "/spaces/"+id+"/channels", `{"channelName":"general"}`, nil); status != http.StatusConflict {
		t.Fatalf("duplicate channel: %d", status)
	}
	if got := rpsOK(t, ts, http.MethodGet, "/spaces/"+id+"/channels/"+ch, ""); got["channelStatus"] != "CREATING" {
		t.Fatalf("new channel: %v", got)
	}
	rpsOK(t, ts, http.MethodPost, "/spaces/"+id+"/channels/"+ch+"/roles", `{"accessorIds":["u1"],"channelRole":"MODERATOR"}`)
	rpsOK(t, ts, http.MethodPut, "/spaces/"+id+"/channels/"+ch, `{"channelName":"general-help","channelDescription":"Anything"}`)
	time.Sleep(350 * time.Millisecond)
	got := rpsOK(t, ts, http.MethodGet, "/spaces/"+id+"/channels/"+ch, "")
	if got["channelStatus"] != "CREATED" || got["channelName"] != "general-help" || got["channelRoles"].(map[string]any)["u1"] == nil {
		t.Fatalf("channel: %v", got)
	}
	rpsOK(t, ts, http.MethodPatch, "/spaces/"+id+"/channels/"+ch+"/roles", `{"accessorIds":["u1"],"channelRole":"MODERATOR"}`)
	if list := rpsOK(t, ts, http.MethodGet, "/spaces/"+id+"/channels?maxResults=1", "")["channels"].([]any); len(list) != 1 {
		t.Fatalf("ListChannels: %v", list)
	}
	// Tags by space ARN.
	arn := "/tags/" + url.PathEscape(rpsSpaceArn(id))
	rpsOK(t, ts, http.MethodPost, arn, `{"tags":{"env":"prod"}}`)
	rpsOK(t, ts, http.MethodDelete, arn+"?tagKeys=team", "")
	if tags := rpsOK(t, ts, http.MethodGet, arn, "")["tags"].(map[string]any); len(tags) != 1 || tags["env"] != "prod" {
		t.Fatalf("tags: %v", tags)
	}
	if status, _ := collabCall(t, ts, "repostspace", http.MethodPost, arn, `{"tags":{"aws:x":"y"}}`, nil); status != http.StatusBadRequest {
		t.Fatalf("aws: tag: %d", status)
	}
	rpsOK(t, ts, http.MethodDelete, "/spaces/"+id+"/admins/admin-1", "")
	if status, _ := collabCall(t, ts, "repostspace", http.MethodDelete, "/spaces/"+id+"/admins/admin-1", "", nil); status != http.StatusNotFound {
		t.Fatalf("deregister twice: %d", status)
	}
	// Delete: DELETING, then gone.
	rpsOK(t, ts, http.MethodDelete, "/spaces/"+id, "")
	if sp := rpsOK(t, ts, http.MethodGet, "/spaces/"+id, ""); sp["status"] != "DELETING" {
		t.Fatalf("deleting: %v", sp)
	}
	if status, _ := collabCall(t, ts, "repostspace", http.MethodPost, "/spaces/"+id+"/channels", `{"channelName":"late"}`, nil); status != http.StatusConflict {
		t.Fatalf("channel on deleting space: %d", status)
	}
	time.Sleep(350 * time.Millisecond)
	if status, out := collabCall(t, ts, "repostspace", http.MethodGet, "/spaces/"+id, "", nil); status != http.StatusNotFound || out["resourceType"] != "space" {
		t.Fatalf("after delete: %d %v", status, out)
	}
	if list := rpsOK(t, ts, http.MethodGet, "/spaces", "")["spaces"].([]any); len(list) != 0 {
		t.Fatalf("ListSpaces: %v", list)
	}
	// The subdomain is free again; the space quota holds.
	for i := 0; i < rpsMaxSpaces; i++ {
		rpsOK(t, ts, http.MethodPost, "/spaces", `{"name":"S","subdomain":"s`+string(rune('a'+i))+`","tier":"BASIC"}`)
	}
	if status, out := collabCall(t, ts, "repostspace", http.MethodPost, "/spaces", `{"name":"S","subdomain":"eng-help","tier":"BASIC"}`, nil); status != http.StatusPaymentRequired || out["quotaCode"] == nil {
		t.Fatalf("quota: %d %v", status, out)
	}
}

// Errors carry the rest-json error type header and the model's members.
func TestRepostSpaceErrorShape(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/spaces/missing", nil, nil, "repostspace")
	raw := mustBody(t, resp)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if resp.StatusCode != http.StatusNotFound || resp.Header.Get("X-Amzn-ErrorType") != "ResourceNotFoundException" ||
		resp.Header.Get("Content-Type") != rpsContentType || out["resourceId"] != "missing" || out["resourceType"] != "space" || out["message"] == nil {
		t.Fatalf("not found: %d %v %v", resp.StatusCode, resp.Header, out)
	}
}
