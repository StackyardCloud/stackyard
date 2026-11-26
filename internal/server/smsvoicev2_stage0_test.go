package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func smsv2TestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// smsv2Call sends a signed awsJson1_0 request with the given target and
// raw body, and decodes the JSON response.
func smsv2Call(t *testing.T, ts *httptest.Server, target, body string, wantStatus int) map[string]any {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": target,
	}, "sms-voice")
	assertStatus(t, resp, wantStatus)
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-amz-json-1.1" {
		t.Fatalf("%s: content type %q", target, ct)
	}
	out := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("%s: decode response: %v", target, err)
	}
	return out
}

func smsv2OK(t *testing.T, ts *httptest.Server, op string, in map[string]any) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(in)
	return smsv2Call(t, ts, "PinpointSMSVoiceV2."+op, string(raw), http.StatusOK)
}

func TestSMSVoiceV2Stage0RequestValidation(t *testing.T) {
	ts := smsv2TestServer(t)
	cases := []struct {
		name, target, body string
	}{
		{"unknown operation", "PinpointSMSVoiceV2.NotAnOperation", `{}`},
		{"malformed JSON", "PinpointSMSVoiceV2.DescribePools", `{"PoolIds":`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := smsv2Call(t, ts, c.target, c.body, http.StatusBadRequest)
			if out["__type"] != "ValidationException" || out["message"] == "" {
				t.Fatalf("error shape: %v", out)
			}
		})
	}
}

func TestSMSVoiceV2Stage0ConfigurationSetsAndOptOuts(t *testing.T) {
	ts := smsv2TestServer(t)

	created := smsv2OK(t, ts, "CreateConfigurationSet", map[string]any{"ConfigurationSetName": "alerts"})
	if created["ConfigurationSetName"] != "alerts" ||
		created["ConfigurationSetArn"] != "arn:aws:sms-voice:us-east-1:123456789012:configuration-set/alerts" {
		t.Fatalf("CreateConfigurationSet: %v", created)
	}
	sets, _ := smsv2OK(t, ts, "DescribeConfigurationSets", map[string]any{})["ConfigurationSets"].([]any)
	found := false
	for _, s := range sets {
		if m, _ := s.(map[string]any); m["ConfigurationSetName"] == "alerts" {
			found = true
		}
	}
	if !found {
		t.Fatalf("alerts missing from DescribeConfigurationSets: %v", sets)
	}

	// Opted-out numbers round-trip per list.
	list := map[string]any{"OptOutListName": "marketing"}
	if out := smsv2OK(t, ts, "CreateOptOutList", list); out["OptOutListName"] != "marketing" {
		t.Fatalf("CreateOptOutList: %v", out)
	}
	for _, n := range []string{"+12065550199", "+12065550111"} {
		smsv2OK(t, ts, "PutOptedOutNumber", map[string]any{"OptOutListName": "marketing", "OptedOutNumber": n})
	}
	numbers := func() []string {
		raw, _ := smsv2OK(t, ts, "DescribeOptedOutNumbers", list)["OptedOutNumbers"].([]any)
		var out []string
		for _, r := range raw {
			out = append(out, r.(map[string]any)["OptedOutNumber"].(string))
		}
		return out
	}
	if got := numbers(); len(got) != 2 || got[0] != "+12065550111" || got[1] != "+12065550199" {
		t.Fatalf("opted-out numbers: %v", got)
	}
	smsv2OK(t, ts, "DeleteOptedOutNumber", map[string]any{"OptOutListName": "marketing", "OptedOutNumber": "+12065550111"})
	if got := numbers(); len(got) != 1 || got[0] != "+12065550199" {
		t.Fatalf("after delete: %v", got)
	}

	// Tags round-trip on the configuration set.
	arn := created["ConfigurationSetArn"]
	smsv2OK(t, ts, "TagResource", map[string]any{"ResourceArn": arn, "Tags": []any{
		map[string]any{"Key": "team", "Value": "ops"}, map[string]any{"Key": "env", "Value": "dev"}}})
	smsv2OK(t, ts, "UntagResource", map[string]any{"ResourceArn": arn, "TagKeys": []any{"env"}})
	tags, _ := smsv2OK(t, ts, "ListTagsForResource", map[string]any{"ResourceArn": arn})["Tags"].([]any)
	got := map[string]any{}
	for _, tag := range tags {
		m := tag.(map[string]any)
		got[m["Key"].(string)] = m["Value"]
	}
	if _, kept := got["env"]; kept || got["team"] != "ops" {
		t.Fatalf("tags: %v", tags)
	}
}
