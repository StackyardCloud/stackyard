package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNotificationsContactsStage123LifecycleAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := notificationsContactsRequest(
		t,
		ts,
		http.MethodPost,
		"/2022-09-19/emailcontacts",
		`{"name":"stage123-contact","emailAddress":"stage123@example.com"}`,
	)
	assertStatus(t, createResp, http.StatusOK)
	createBody := mustBody(t, createResp)

	var created map[string]any
	if err := json.Unmarshal(createBody, &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	arn, _ := created["arn"].(string)
	if arn == "" {
		t.Fatalf("expected create response to include arn: %s", string(createBody))
	}
	encodedARN := url.PathEscape(arn)

	getResp := notificationsContactsRequest(t, ts, http.MethodGet, "/emailcontacts/"+encodedARN, "")
	assertStatus(t, getResp, http.StatusOK)

	sendCodeResp := notificationsContactsRequest(t, ts, http.MethodPost, "/2022-10-31/emailcontacts/"+encodedARN+"/activate/send", `{}`)
	assertStatus(t, sendCodeResp, http.StatusOK)

	activateResp := notificationsContactsRequest(t, ts, http.MethodPut, "/emailcontacts/"+encodedARN+"/activate/123456", `{}`)
	assertStatus(t, activateResp, http.StatusOK)

	tagResp := notificationsContactsRequest(t, ts, http.MethodPost, "/tags/"+encodedARN, `{"tags":{"env":"stage123","team":"qa"}}`)
	assertStatus(t, tagResp, http.StatusOK)

	listTagsResp := notificationsContactsRequest(t, ts, http.MethodGet, "/tags/"+encodedARN, "")
	assertStatus(t, listTagsResp, http.StatusOK)
	listTagsBody := string(mustBody(t, listTagsResp))
	if !(notificationsContactsContainsAll(listTagsBody, []string{"env", "stage123", "team", "qa"})) {
		t.Fatalf("expected tags in response body, got %s", listTagsBody)
	}

	untagResp := notificationsContactsRequest(t, ts, http.MethodDelete, "/tags/"+encodedARN+"?tagKeys=team", ``)
	assertStatus(t, untagResp, http.StatusOK)

	listTagsResp2 := notificationsContactsRequest(t, ts, http.MethodGet, "/tags/"+encodedARN, "")
	assertStatus(t, listTagsResp2, http.StatusOK)
	listTagsBody2 := string(mustBody(t, listTagsResp2))
	if !(notificationsContactsContainsAll(listTagsBody2, []string{"env", "stage123"})) || notificationsContactsContainsAll(listTagsBody2, []string{"team"}) {
		t.Fatalf("expected env tag preserved and team tag removed, got %s", listTagsBody2)
	}

	deleteResp := notificationsContactsRequest(t, ts, http.MethodDelete, "/emailcontacts/"+encodedARN, "")
	assertStatus(t, deleteResp, http.StatusOK)
}

func notificationsContactsContainsAll(body string, pieces []string) bool {
	for _, piece := range pieces {
		if !strings.Contains(body, piece) {
			return false
		}
	}
	return true
}
