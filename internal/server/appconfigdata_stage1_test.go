package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func appConfigDataStart(t *testing.T, ts *httptest.Server, body string) string {
	t.Helper()
	resp := appConfigDataRequest(t, ts, http.MethodPost, "/configurationsessions", []byte(body), nil)
	assertStatus(t, resp, http.StatusCreated)
	var out struct{ InitialConfigurationToken string }
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.InitialConfigurationToken == "" {
		t.Fatalf("expected InitialConfigurationToken, err=%v", err)
	}
	return out.InitialConfigurationToken
}

func appConfigDataPoll(t *testing.T, ts *httptest.Server, token string) (*http.Response, string) {
	t.Helper()
	resp := appConfigDataRequest(t, ts, http.MethodGet, "/configuration?configuration_token="+url.QueryEscape(token), nil, nil)
	return resp, string(mustBody(t, resp))
}

func appConfigDataControl(t *testing.T, ts *httptest.Server, method, path, body string, headers map[string]string) map[string]any {
	t.Helper()
	resp := appConfigDataRequest(t, ts, method, path, []byte(body), headers)
	assertStatus(t, resp, http.StatusOK)
	out := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal %s %s: %v", method, path, err)
	}
	return out
}

func TestAppConfigDataStage1SeededSession(t *testing.T) {
	_, ts := newAppConfigDataTestServer(t)
	token := appConfigDataStart(t, ts, `{"ApplicationIdentifier":"app-000001","EnvironmentIdentifier":"env-000001","ConfigurationProfileIdentifier":"cp-000001"}`)

	resp, body := appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusOK)
	if body != `{"stackyard":true}` || resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected seeded configuration %q (%s)", body, resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("Next-Poll-Interval-In-Seconds") != "60" {
		t.Fatalf("expected default poll interval 60, got %q", resp.Header.Get("Next-Poll-Interval-In-Seconds"))
	}
	next := resp.Header.Get("Next-Poll-Configuration-Token")
	if next == "" || next == token {
		t.Fatalf("expected a new token, got %q", next)
	}

	// Same version again: empty body, no Version-Label.
	resp, body = appConfigDataPoll(t, ts, next)
	assertStatus(t, resp, http.StatusOK)
	if body != "" {
		t.Fatalf("expected empty body for an unchanged configuration, got %q", body)
	}

	// Tokens are single-use.
	resp, body = appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(body, `"Problem":"Corrupted"`) {
		t.Fatalf("expected Corrupted token problem, got %q", body)
	}
}

func TestAppConfigDataStage1ControlPlaneLifecycle(t *testing.T) {
	_, ts := newAppConfigDataTestServer(t)
	app := appConfigDataControl(t, ts, http.MethodPost, "/applications", `{"Name":"orders"}`, nil)["Id"].(string)
	env := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/environments", `{"Name":"prod"}`, nil)["Id"].(string)
	profile := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/configurationprofiles", `{"Name":"flags","LocationUri":"hosted"}`, nil)["Id"].(string)

	versionPath := "/applications/" + app + "/configurationprofiles/" + profile + "/hostedconfigurationversions"
	resp := appConfigDataRequest(t, ts, http.MethodPost, versionPath, []byte("feature=on"), map[string]string{"Content-Type": "text/plain", "VersionLabel": "v1"})
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("Version-Number") != "1" {
		t.Fatalf("expected Version-Number 1, got %q", resp.Header.Get("Version-Number"))
	}

	// Sessions resolve names; nothing is deployed yet.
	token := appConfigDataStart(t, ts, `{"ApplicationIdentifier":"orders","EnvironmentIdentifier":"prod","ConfigurationProfileIdentifier":"flags"}`)
	resp, body := appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusNotFound)
	if !strings.Contains(body, `"ResourceType":"Deployment"`) {
		t.Fatalf("expected missing Deployment, got %q", body)
	}

	deployments := 0
	deployProfile := func(profileID, version string) {
		t.Helper()
		out := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/environments/"+env+"/deployments",
			`{"ConfigurationProfileId":"`+profileID+`","ConfigurationVersion":"`+version+`","DeploymentStrategyId":"ds-000001"}`, nil)
		deployments++
		if got, _ := out["DeploymentNumber"].(float64); int(got) != deployments {
			t.Fatalf("expected DeploymentNumber %d, got %v", deployments, out["DeploymentNumber"])
		}
	}
	deploy := func(version string) { t.Helper(); deployProfile(profile, version) }
	deploy("1")

	// A failed poll leaves the token usable.
	resp, body = appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusOK)
	if body != "feature=on" || resp.Header.Get("Version-Label") != "v1" || resp.Header.Get("Content-Type") != "text/plain" {
		t.Fatalf("unexpected v1 response %q label=%q type=%q", body, resp.Header.Get("Version-Label"), resp.Header.Get("Content-Type"))
	}
	token = resp.Header.Get("Next-Poll-Configuration-Token")

	resp = appConfigDataRequest(t, ts, http.MethodPost, versionPath, []byte("feature=off"), map[string]string{"Content-Type": "text/plain"})
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("Version-Number") != "2" {
		t.Fatalf("expected Version-Number 2, got %q", resp.Header.Get("Version-Number"))
	}
	deploy("2")
	resp, body = appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusOK)
	if body != "feature=off" || resp.Header.Get("Version-Label") != "" {
		t.Fatalf("expected unlabelled v2, got %q label=%q", body, resp.Header.Get("Version-Label"))
	}
	token = resp.Header.Get("Next-Poll-Configuration-Token")

	// A deployment of another profile to the same environment doesn't reach this session.
	other := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/configurationprofiles", `{"Name":"other","LocationUri":"hosted"}`, nil)["Id"].(string)
	resp = appConfigDataRequest(t, ts, http.MethodPost, "/applications/"+app+"/configurationprofiles/"+other+"/hostedconfigurationversions", []byte("other=1"), map[string]string{"Content-Type": "text/plain"})
	assertStatus(t, resp, http.StatusOK)
	deployProfile(other, "1")
	resp, body = appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusOK)
	if body != "" {
		t.Fatalf("expected no change from another profile's deployment, got %q", body)
	}
	token = resp.Header.Get("Next-Poll-Configuration-Token")

	// Stopping deployment 2 rolls the environment back to version 1.
	resp = appConfigDataRequest(t, ts, http.MethodDelete, "/applications/"+app+"/environments/"+env+"/deployments/2", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp, body = appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusOK)
	if body != "feature=on" {
		t.Fatalf("expected rollback to v1, got %q", body)
	}

	// A deleted application can't start new sessions.
	resp = appConfigDataRequest(t, ts, http.MethodDelete, "/applications/"+app, nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp = appConfigDataRequest(t, ts, http.MethodPost, "/configurationsessions", []byte(`{"ApplicationIdentifier":"orders","EnvironmentIdentifier":"prod","ConfigurationProfileIdentifier":"flags"}`), nil)
	assertStatus(t, resp, http.StatusNotFound)
}

func TestAppConfigDataStage1Validation(t *testing.T) {
	srv, ts := newAppConfigDataTestServer(t)
	seeded := func(extra string) string {
		return `{"ApplicationIdentifier":"app-000001","EnvironmentIdentifier":"env-000001","ConfigurationProfileIdentifier":"cp-000001"` + extra + `}`
	}
	cases := []struct {
		name, body string
		status     int
		want       string
	}{
		{"missing environment", `{"ApplicationIdentifier":"app-000001","ConfigurationProfileIdentifier":"cp-000001"}`, http.StatusBadRequest, "EnvironmentIdentifier"},
		{"identifier too long", `{"ApplicationIdentifier":"` + strings.Repeat("a", 129) + `","EnvironmentIdentifier":"e","ConfigurationProfileIdentifier":"c"}`, http.StatusBadRequest, "ApplicationIdentifier"},
		{"poll too short", seeded(`,"RequiredMinimumPollIntervalInSeconds":14`), http.StatusBadRequest, "RequiredMinimumPollIntervalInSeconds"},
		{"poll too long", seeded(`,"RequiredMinimumPollIntervalInSeconds":86401`), http.StatusBadRequest, "RequiredMinimumPollIntervalInSeconds"},
		{"bad json", `{`, http.StatusBadRequest, "invalid JSON"},
		{"unknown application", `{"ApplicationIdentifier":"nope","EnvironmentIdentifier":"env-000001","ConfigurationProfileIdentifier":"cp-000001"}`, http.StatusNotFound, `"ResourceType":"Application"`},
		{"unknown environment", `{"ApplicationIdentifier":"app-000001","EnvironmentIdentifier":"nope","ConfigurationProfileIdentifier":"cp-000001"}`, http.StatusNotFound, `"ResourceType":"Environment"`},
		{"unknown profile", `{"ApplicationIdentifier":"app-000001","EnvironmentIdentifier":"env-000001","ConfigurationProfileIdentifier":"nope"}`, http.StatusNotFound, `"ResourceType":"ConfigurationProfile"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := appConfigDataRequest(t, ts, http.MethodPost, "/configurationsessions", []byte(tc.body), nil)
			assertStatus(t, resp, tc.status)
			if body := string(mustBody(t, resp)); !strings.Contains(body, tc.want) {
				t.Fatalf("expected %q in %q", tc.want, body)
			}
		})
	}

	resp, body := appConfigDataPoll(t, ts, "")
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(body, "ConfigurationToken is required") {
		t.Fatalf("expected missing token error, got %q", body)
	}

	// The minimum poll interval is enforced, and the early token stays valid.
	token := appConfigDataStart(t, ts, seeded(`,"RequiredMinimumPollIntervalInSeconds":15`))
	resp, _ = appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusOK)
	if resp.Header.Get("Next-Poll-Interval-In-Seconds") != "15" {
		t.Fatalf("expected poll interval 15, got %q", resp.Header.Get("Next-Poll-Interval-In-Seconds"))
	}
	next := resp.Header.Get("Next-Poll-Configuration-Token")
	resp, body = appConfigDataPoll(t, ts, next)
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(body, `"Problem":"PollIntervalNotSatisfied"`) {
		t.Fatalf("expected PollIntervalNotSatisfied, got %q", body)
	}
	srv.appconfig.mu.Lock()
	session := srv.appconfig.sessions[next]
	session.NotBefore = time.Now().Add(-time.Second)
	srv.appconfig.sessions[next] = session
	srv.appconfig.mu.Unlock()
	resp, _ = appConfigDataPoll(t, ts, next)
	assertStatus(t, resp, http.StatusOK)

	// Tokens expire after 24 hours.
	token = appConfigDataStart(t, ts, seeded(""))
	srv.appconfig.mu.Lock()
	session = srv.appconfig.sessions[token]
	session.IssuedAt = time.Now().Add(-25 * time.Hour)
	srv.appconfig.sessions[token] = session
	srv.appconfig.mu.Unlock()
	resp, body = appConfigDataPoll(t, ts, token)
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(body, `"Problem":"Expired"`) {
		t.Fatalf("expected Expired, got %q", body)
	}
}
