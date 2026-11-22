package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const cloudTrailDataTestEvent = `{"version":"1.0","userIdentity":{"type":"CustomUser","principalId":"alice"},` +
	`"eventSource":"app.example.com","eventName":"Login","eventTime":"2026-11-21T10:00:00Z",` +
	`"UID":"uid-1","recipientAccountId":"123456789012"}`

func newCloudTrailDataTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := cloudTrailRequest(t, ts, "CreateEventDataStore", `{"Name":"audit-eds"}`)
	assertStatus(t, resp, http.StatusOK)
	mustBody(t, resp)
	resp = cloudTrailRequest(t, ts, "CreateChannel", `{"Name":"app-channel","Source":"Custom",`+
		`"Destinations":[{"Type":"EventDataStore","Location":"arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/audit-eds"}]}`)
	assertStatus(t, resp, http.StatusOK)
	var out struct{ ChannelArn string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.ChannelArn == "" {
		t.Fatalf("CreateChannel: %v %+v", err, out)
	}
	return ts, out.ChannelArn
}

func putAuditEvents(t *testing.T, ts *httptest.Server, channel, externalID, body string) *http.Response {
	t.Helper()
	q := url.Values{"channelArn": {channel}}
	if externalID != "" {
		q.Set("externalId", externalID)
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/PutAuditEvents?"+q.Encode(), []byte(body),
		map[string]string{"Content-Type": "application/json"}, "cloudtrail-data")
}

func auditEventsJSON(t *testing.T, events ...map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"auditEvents": events})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestCloudTrailDataStage1PutAuditEventsSuccessAndFailures(t *testing.T) {
	ts, channel := newCloudTrailDataTestServer(t)
	sum := sha256.Sum256([]byte(cloudTrailDataTestEvent))
	good := base64.StdEncoding.EncodeToString(sum[:])

	body := auditEventsJSON(t,
		map[string]any{"id": "ok-1", "eventData": cloudTrailDataTestEvent, "eventDataChecksum": good},
		map[string]any{"id": "bad-sum", "eventData": cloudTrailDataTestEvent, "eventDataChecksum": "AAAA"},
		map[string]any{"id": "bad-json", "eventData": "not json"},
		map[string]any{"id": "no-data"},
		map[string]any{"id": "no-field", "eventData": `{"version":"1.0"}`},
		map[string]any{"id": "bad-recipient", "eventData": strings.Replace(cloudTrailDataTestEvent, "123456789012", "999999999999", 1)},
	)
	resp := putAuditEvents(t, ts, channel, "", body)
	assertStatus(t, resp, http.StatusOK)
	var out cloudtraildataPutResponse
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Successful) != 1 || out.Successful[0].ID != "ok-1" || out.Successful[0].EventID == "" {
		t.Fatalf("unexpected successful list: %+v", out.Successful)
	}
	want := map[string]string{"bad-sum": "InvalidChecksum", "bad-json": "InvalidData", "no-data": "FieldNotFound",
		"no-field": "FieldNotFound", "bad-recipient": "InvalidRecipient"}
	if len(out.Failed) != len(want) {
		t.Fatalf("unexpected failed list: %+v", out.Failed)
	}
	for _, f := range out.Failed {
		if want[f.ID] != f.ErrorCode || f.ErrorMessage == "" {
			t.Fatalf("event %s: got %s (%s), want %s", f.ID, f.ErrorCode, f.ErrorMessage, want[f.ID])
		}
	}

	// The channel ID (ARN suffix) is accepted as well as the ARN.
	resp = putAuditEvents(t, ts, "app-channel", "", auditEventsJSON(t, map[string]any{"id": "ok-2", "eventData": cloudTrailDataTestEvent}))
	assertStatus(t, resp, http.StatusOK)
	mustBody(t, resp)
}

func TestCloudTrailDataStage1RequestErrors(t *testing.T) {
	ts, channel := newCloudTrailDataTestServer(t)
	one := auditEventsJSON(t, map[string]any{"id": "e1", "eventData": cloudTrailDataTestEvent})
	cases := []struct {
		name, channel, externalID, body, code string
	}{
		{"missing channel", "", "", one, "ValidationException"},
		{"bad json", channel, "", "{", "ValidationException"},
		{"no events", channel, "", `{"auditEvents":[]}`, "ValidationException"},
		{"bad id", channel, "", auditEventsJSON(t, map[string]any{"id": "has space", "eventData": "{}"}), "ValidationException"},
		{"duplicate id", channel, "", auditEventsJSON(t, map[string]any{"id": "d", "eventData": "{}"}, map[string]any{"id": "d", "eventData": "{}"}), "DuplicatedAuditEventId"},
		{"not a channel arn", "arn:aws:s3:::bucket", "", one, "InvalidChannelARN"},
		{"other account", "arn:aws:cloudtrail:us-east-1:111111111111:channel/app-channel", "", one, "ChannelInsufficientPermission"},
		{"unknown channel", "arn:aws:cloudtrail:us-east-1:123456789012:channel/missing", "", one, "ChannelNotFound"},
		{"bad external id", channel, "x", one, "ValidationException"},
	}
	for _, tc := range cases {
		resp := putAuditEvents(t, ts, tc.channel, tc.externalID, tc.body)
		assertStatus(t, resp, http.StatusBadRequest)
		var out struct {
			Type string `json:"__type"`
		}
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.Type != tc.code {
			t.Fatalf("%s: got %q (%v), want %s", tc.name, out.Type, err, tc.code)
		}
	}

	// A deleted channel no longer accepts events.
	resp := cloudTrailRequest(t, ts, "DeleteChannel", `{"Channel":"`+channel+`"}`)
	assertStatus(t, resp, http.StatusOK)
	mustBody(t, resp)
	resp = putAuditEvents(t, ts, channel, "", one)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ChannelNotFound") {
		t.Fatalf("expected ChannelNotFound after delete, got %s", body)
	}
}

func TestCloudTrailDataStage1ExternalIDPolicy(t *testing.T) {
	ts, channel := newCloudTrailDataTestServer(t)
	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},"Action":"cloudtrail-data:PutAuditEvents",` +
		`"Resource":"` + channel + `","Condition":{"StringEquals":{"cloudtrail:ExternalId":"secret-id"}}}]}`
	raw, _ := json.Marshal(map[string]string{"ResourceArn": channel, "ResourcePolicy": policy})
	resp := cloudTrailRequest(t, ts, "PutResourcePolicy", string(raw))
	assertStatus(t, resp, http.StatusOK)
	mustBody(t, resp)

	one := auditEventsJSON(t, map[string]any{"id": "e1", "eventData": cloudTrailDataTestEvent})
	for _, id := range []string{"", "wrong-id"} {
		resp = putAuditEvents(t, ts, channel, id, one)
		assertStatus(t, resp, http.StatusBadRequest)
		if body := string(mustBody(t, resp)); !strings.Contains(body, "ChannelInsufficientPermission") {
			t.Fatalf("externalId %q: expected ChannelInsufficientPermission, got %s", id, body)
		}
	}
	resp = putAuditEvents(t, ts, channel, "secret-id", one)
	assertStatus(t, resp, http.StatusOK)
	mustBody(t, resp)
}

func TestCloudTrailDataStage1EventsAreQueryable(t *testing.T) {
	ts, channel := newCloudTrailDataTestServer(t)
	other := strings.Replace(cloudTrailDataTestEvent, `"Login"`, `"Logout"`, 1)
	resp := putAuditEvents(t, ts, channel, "", auditEventsJSON(t,
		map[string]any{"id": "e1", "eventData": cloudTrailDataTestEvent},
		map[string]any{"id": "e2", "eventData": other}))
	assertStatus(t, resp, http.StatusOK)
	mustBody(t, resp)

	query := func(statement string) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(map[string]string{"QueryStatement": statement})
		resp := cloudTrailRequest(t, ts, "StartQuery", string(raw))
		assertStatus(t, resp, http.StatusOK)
		var started struct{ QueryId string }
		if err := json.Unmarshal(mustBody(t, resp), &started); err != nil {
			t.Fatal(err)
		}
		resp = cloudTrailRequest(t, ts, "GetQueryResults", `{"QueryId":"`+started.QueryId+`"}`)
		assertStatus(t, resp, http.StatusOK)
		var out map[string]any
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	out := query("SELECT eventData.eventName, eventData.userIdentity.principalId FROM audit-eds WHERE eventData.eventName = 'Logout'")
	rows, _ := out["QueryResultRows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("expected one filtered row, got %v", out)
	}
	row := rows[0].([]any)
	if row[0].(map[string]any)["eventData.eventName"] != "Logout" || row[1].(map[string]any)["eventData.userIdentity.principalId"] != "alice" {
		t.Fatalf("unexpected row %v", row)
	}

	out = query("SELECT eventID FROM arn:aws:cloudtrail:us-east-1:123456789012:eventdatastore/audit-eds LIMIT 1")
	if rows, _ := out["QueryResultRows"].([]any); len(rows) != 1 {
		t.Fatalf("expected LIMIT 1 to return one row, got %v", out)
	}

	out = query("SELECT eventID FROM audit-eds WHERE eventTime > '2026'")
	if out["QueryStatus"] != "FAILED" || out["ErrorMessage"] == "" {
		t.Fatalf("expected unsupported WHERE to fail, got %v", out)
	}

	// Queries over unknown sources keep the stub (empty) result.
	out = query("SELECT * FROM stackyard LIMIT 25")
	if rows, _ := out["QueryResultRows"].([]any); len(rows) != 0 || out["QueryStatus"] != "FINISHED" {
		t.Fatalf("expected empty stub result, got %v", out)
	}
}
