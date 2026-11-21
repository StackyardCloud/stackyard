package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// IDs come only from creates, so replaying the journaled writes (reads are
// not journaled) reproduces the live IDs.
func TestAppConfigDataStage2IDsSurviveReplay(t *testing.T) {
	create := func(st *appConfigStore) string {
		out, fault := st.Handle("CreateApplication", map[string]any{"Name": "x"}, map[string]string{}, nil)
		if fault != nil {
			t.Fatalf("create: %v", fault)
		}
		return out["Id"].(string)
	}
	live := newAppConfigStore()
	first := create(live)
	for range 2 {
		live.Handle("GetApplication", map[string]any{}, map[string]string{"ApplicationId": first}, nil)
		live.Handle("UpdateApplication", map[string]any{}, map[string]string{"ApplicationId": first}, nil)
	}
	second := create(live)
	if first != "app-000002" || second != "app-000003" {
		t.Fatalf("expected consecutive IDs app-000002, app-000003, got %s, %s", first, second)
	}
	replay := newAppConfigStore()
	if a, b := create(replay), create(replay); a != first || b != second {
		t.Fatalf("replay gave %s, %s; live gave %s, %s", a, b, first, second)
	}
}

func TestAppConfigDataStage2HostedVersionsAreRawContent(t *testing.T) {
	_, ts := newAppConfigDataTestServer(t)
	app := appConfigDataControl(t, ts, http.MethodPost, "/applications", `{"Name":"raw"}`, nil)["Id"].(string)
	env := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/environments", `{"Name":"prod"}`, nil)["Id"].(string)
	profile := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/configurationprofiles", `{"Name":"p","LocationUri":"hosted"}`, nil)["Id"].(string)
	versions := "/applications/" + app + "/configurationprofiles/" + profile + "/hostedconfigurationversions"

	cases := []struct{ content, contentType string }{
		{`[1,2,3]`, "application/json"},                  // not a JSON object
		{`42`, "application/json"},                       // scalar
		{"\x00\xffbinary\n", "application/octet-stream"}, // not JSON at all
	}
	for i, tc := range cases {
		number := string(rune('1' + i))
		resp := appConfigDataRequest(t, ts, http.MethodPost, versions, []byte(tc.content),
			map[string]string{"Content-Type": tc.contentType, "VersionLabel": "label-" + number, "Description": "desc " + number})
		assertStatus(t, resp, http.StatusCreated)
		if body := string(mustBody(t, resp)); body != tc.content {
			t.Fatalf("create %d: body %q, want the content %q", i, body, tc.content)
		}
		for header, want := range map[string]string{"Version-Number": number, "Content-Type": tc.contentType, "VersionLabel": "label-" + number, "Description": "desc " + number, "Application-Id": app} {
			if got := resp.Header.Get(header); got != want {
				t.Fatalf("create %d: %s = %q, want %q", i, header, got, want)
			}
		}
		resp = appConfigDataRequest(t, ts, http.MethodGet, versions+"/"+number, nil, nil)
		assertStatus(t, resp, http.StatusOK)
		if body := string(mustBody(t, resp)); body != tc.content || resp.Header.Get("Content-Type") != tc.contentType {
			t.Fatalf("get %d: %q (%s)", i, body, resp.Header.Get("Content-Type"))
		}
	}

	// The data plane serves the bytes unchanged.
	appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/environments/"+env+"/deployments",
		`{"ConfigurationProfileId":"`+profile+`","ConfigurationVersion":"3","DeploymentStrategyId":"ds-000001"}`, nil)
	token := appConfigDataStart(t, ts, `{"ApplicationIdentifier":"`+app+`","EnvironmentIdentifier":"prod","ConfigurationProfileIdentifier":"p"}`)
	resp, body := appConfigDataGet(t, ts, token)
	assertStatus(t, resp, http.StatusOK)
	if body != cases[2].content {
		t.Fatalf("data plane returned %q, want %q", body, cases[2].content)
	}

	// Redeploying the same version is not a new configuration.
	appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/environments/"+env+"/deployments",
		`{"ConfigurationProfileId":"`+profile+`","ConfigurationVersion":"3","DeploymentStrategyId":"ds-000001"}`, nil)
	next := resp.Header.Get("Next-Poll-Configuration-Token")
	resp, body = appConfigDataGet(t, ts, next)
	assertStatus(t, resp, http.StatusOK)
	if body != "" {
		t.Fatalf("expected no content after redeploying the same version, got %q", body)
	}
	if newer := resp.Header.Get("Next-Poll-Configuration-Token"); newer == "" || newer == next {
		t.Fatalf("unchanged poll must still issue a new token, got %q", newer)
	}

	// Deleting the deployed version stops it being served; deploying a missing version fails.
	next = resp.Header.Get("Next-Poll-Configuration-Token")
	resp = appConfigDataRequest(t, ts, http.MethodDelete, versions+"/3", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	resp, body = appConfigDataGet(t, ts, next)
	assertStatus(t, resp, http.StatusNotFound)
	if !strings.Contains(body, `"ResourceType":"Configuration"`) {
		t.Fatalf("expected deleted version to be missing, got %q", body)
	}
	for _, version := range []string{"3", "9"} {
		resp = appConfigDataRequest(t, ts, http.MethodPost, "/applications/"+app+"/environments/"+env+"/deployments",
			[]byte(`{"ConfigurationProfileId":"`+profile+`","ConfigurationVersion":"`+version+`","DeploymentStrategyId":"ds-000001"}`), nil)
		assertStatus(t, resp, http.StatusNotFound)
		if resp.Header.Get("X-Amzn-ErrorType") != "ResourceNotFoundException" {
			t.Fatalf("deploying version %s: expected ResourceNotFoundException, got %q", version, resp.Header.Get("X-Amzn-ErrorType"))
		}
	}
}

func TestAppConfigDataStage2UpdatesAndDeletes(t *testing.T) {
	_, ts := newAppConfigDataTestServer(t)
	app := appConfigDataControl(t, ts, http.MethodPost, "/applications", `{"Name":"shop"}`, nil)["Id"].(string)
	env := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/environments", `{"Name":"prod"}`, nil)["Id"].(string)
	profile := appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/configurationprofiles", `{"Name":"flags","LocationUri":"hosted"}`, nil)["Id"].(string)

	// Updates without a Name keep the recorded names.
	for _, path := range []string{"/applications/" + app, "/applications/" + app + "/environments/" + env, "/applications/" + app + "/configurationprofiles/" + profile} {
		resp := appConfigDataRequest(t, ts, http.MethodPatch, path, []byte(`{"Description":"x"}`), nil)
		assertStatus(t, resp, http.StatusOK)
	}
	// Mixed name and ID identifiers resolve.
	appConfigDataStart(t, ts, `{"ApplicationIdentifier":"shop","EnvironmentIdentifier":"prod","ConfigurationProfileIdentifier":"flags"}`)
	appConfigDataStart(t, ts, `{"ApplicationIdentifier":"`+app+`","EnvironmentIdentifier":"`+env+`","ConfigurationProfileIdentifier":"`+profile+`"}`)

	// A rename by Name takes effect.
	resp := appConfigDataRequest(t, ts, http.MethodPatch, "/applications/"+app+"/environments/"+env, []byte(`{"Name":"live"}`), nil)
	assertStatus(t, resp, http.StatusOK)
	appConfigDataStart(t, ts, `{"ApplicationIdentifier":"`+app+`","EnvironmentIdentifier":"live","ConfigurationProfileIdentifier":"`+profile+`"}`)

	// Deletes drop open sessions and the deleted resources.
	start := `{"ApplicationIdentifier":"shop","EnvironmentIdentifier":"live","ConfigurationProfileIdentifier":"flags"}`
	for _, step := range []struct{ path, missing string }{
		{"/applications/" + app + "/configurationprofiles/" + profile, "ConfigurationProfile"},
		{"/applications/" + app + "/environments/" + env, "Environment"},
		{"/applications/" + app, "Application"},
	} {
		token := appConfigDataStart(t, ts, start)
		resp = appConfigDataRequest(t, ts, http.MethodDelete, step.path, nil, nil)
		assertStatus(t, resp, http.StatusOK)
		resp, body := appConfigDataGet(t, ts, token)
		assertStatus(t, resp, http.StatusBadRequest)
		if !strings.Contains(body, `"Problem":"Corrupted"`) {
			t.Fatalf("after deleting %s the session should be gone, got %q", step.path, body)
		}
		resp = appConfigDataRequest(t, ts, http.MethodPost, "/configurationsessions", []byte(start), nil)
		assertStatus(t, resp, http.StatusNotFound)
		if body := string(mustBody(t, resp)); !strings.Contains(body, `"ResourceType":"`+step.missing+`"`) {
			t.Fatalf("expected missing %s, got %q", step.missing, body)
		}
		// Recreate what was deleted for the next step.
		switch step.missing {
		case "ConfigurationProfile":
			profile = appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/configurationprofiles", `{"Name":"flags","LocationUri":"hosted"}`, nil)["Id"].(string)
		case "Environment":
			env = appConfigDataControl(t, ts, http.MethodPost, "/applications/"+app+"/environments", `{"Name":"live"}`, nil)["Id"].(string)
		}
	}
}

func TestAppConfigDataStage2ConcurrentPollsUseTokenOnce(t *testing.T) {
	srv, ts := newAppConfigDataTestServer(t)
	token := appConfigDataStart(t, ts, `{"ApplicationIdentifier":"app-000001","EnvironmentIdentifier":"env-000001","ConfigurationProfileIdentifier":"cp-000001"}`)

	const pollers = 16
	codes := make(chan int, pollers)
	var wg sync.WaitGroup
	for range pollers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			srv.appConfigDataGetLatest(rec, httptest.NewRequest(http.MethodGet, "/configuration?configuration_token="+url.QueryEscape(token), nil))
			codes <- rec.Code
		}()
	}
	wg.Wait()
	close(codes)
	ok := 0
	for code := range codes {
		if code == http.StatusOK {
			ok++
		} else if code != http.StatusBadRequest {
			t.Fatalf("unexpected status %d", code)
		}
	}
	if ok != 1 {
		t.Fatalf("expected exactly one successful poll, got %d", ok)
	}
}

func TestAppConfigDataStage2TagListIsACopy(t *testing.T) {
	st := newAppConfigStore()
	arn := map[string]string{"ResourceArn": "arn:aws:appconfig:us-east-1:123456789012:application/app-000001"}
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			st.Handle("TagResource", map[string]any{"Tags": map[string]any{"k" + string(rune('a'+i)): "v"}}, arn, nil)
		}()
		go func() {
			defer wg.Done()
			out, _ := st.Handle("ListTagsForResource", map[string]any{}, arn, nil)
			if _, err := json.Marshal(out); err != nil {
				t.Errorf("marshal tags: %v", err)
			}
		}()
	}
	wg.Wait()
}
