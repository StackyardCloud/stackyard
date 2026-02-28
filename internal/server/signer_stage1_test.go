package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type signerProfileCreateOut struct {
	ARN               string `json:"arn"`
	ProfileVersion    string `json:"profileVersion"`
	ProfileVersionARN string `json:"profileVersionArn"`
}

type signerStartJobOut struct {
	JobID    string `json:"jobId"`
	JobOwner string `json:"jobOwner"`
}

func signerCreateProfileForTest(t *testing.T, ts *httptest.Server, profileName string) signerProfileCreateOut {
	t.Helper()

	resp := signerRequest(t, ts, http.MethodPut, "/signing-profiles/"+url.PathEscape(profileName), mustJSON(t, map[string]any{
		"platformId": "AWSLambda-SHA384-ECDSA",
	}))
	assertStatus(t, resp, http.StatusOK)

	var out signerProfileCreateOut
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal put signing profile response: %v", err)
	}
	if out.ARN == "" || out.ProfileVersion == "" || out.ProfileVersionARN == "" {
		t.Fatalf("expected arn/profileVersion/profileVersionArn in create profile response")
	}
	return out
}

func signerStartJobForTest(t *testing.T, ts *httptest.Server, profileName, token string) signerStartJobOut {
	t.Helper()

	resp := signerRequest(t, ts, http.MethodPost, "/signing-jobs", mustJSON(t, map[string]any{
		"profileName":        profileName,
		"clientRequestToken": token,
		"source": map[string]any{
			"s3": map[string]any{
				"bucketName": "stackyard-source",
				"key":        "artifact.zip",
				"version":    "1",
			},
		},
		"destination": map[string]any{
			"s3": map[string]any{
				"bucketName": "stackyard-destination",
				"prefix":     "signed/",
			},
		},
	}))
	assertStatus(t, resp, http.StatusOK)

	var out signerStartJobOut
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal start signing job response: %v", err)
	}
	if out.JobID == "" || out.JobOwner == "" {
		t.Fatalf("expected jobId/jobOwner in start signing job response")
	}
	return out
}

func TestSignerStage1ProfileLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	profileName := "stackyard-signer-stage1"
	created := signerCreateProfileForTest(t, ts, profileName)

	resp := signerRequest(t, ts, http.MethodGet, "/signing-profiles/"+url.PathEscape(profileName)+"?profileOwner=123456789012", nil)
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		ProfileName    string `json:"profileName"`
		ProfileVersion string `json:"profileVersion"`
		Status         string `json:"status"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal get signing profile response: %v", err)
	}
	if getOut.ProfileName != profileName || getOut.ProfileVersion != created.ProfileVersion || getOut.Status != "Active" {
		t.Fatalf("unexpected get signing profile payload")
	}

	resp = signerRequest(t, ts, http.MethodGet, "/signing-profiles?includeCanceled=true&maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		Profiles []struct {
			ProfileName string `json:"profileName"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list signing profiles response: %v", err)
	}
	found := false
	for _, p := range listOut.Profiles {
		if p.ProfileName == profileName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %q in list signing profiles response", profileName)
	}

	resp = signerRequest(t, ts, http.MethodDelete, "/signing-profiles/"+url.PathEscape(profileName), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signerRequest(t, ts, http.MethodGet, "/signing-profiles/"+url.PathEscape(profileName), nil)
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal get signing profile response after cancel: %v", err)
	}
	if getOut.Status != "Canceled" {
		t.Fatalf("expected canceled profile status, got %q", getOut.Status)
	}
}

func TestSignerStage2JobLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	profileName := "stackyard-signer-stage2"
	signerCreateProfileForTest(t, ts, profileName)

	started := signerStartJobForTest(t, ts, profileName, "stackyard-stage2-token")
	started2 := signerStartJobForTest(t, ts, profileName, "stackyard-stage2-token")
	if started2.JobID != started.JobID {
		t.Fatalf("expected idempotent start signing job response")
	}

	resp := signerRequest(t, ts, http.MethodGet, "/signing-jobs/"+url.PathEscape(started.JobID), nil)
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		JobID    string `json:"jobId"`
		Status   string `json:"status"`
		JobOwner string `json:"jobOwner"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe signing job response: %v", err)
	}
	if describeOut.JobID != started.JobID || describeOut.JobOwner != started.JobOwner || describeOut.Status == "" {
		t.Fatalf("unexpected describe signing job payload")
	}

	resp = signerRequest(t, ts, http.MethodGet, "/signing-jobs?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		Jobs []struct {
			JobID string `json:"jobId"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list signing jobs response: %v", err)
	}
	if len(listOut.Jobs) == 0 || listOut.Jobs[0].JobID == "" {
		t.Fatalf("expected at least one signing job")
	}
}

func TestSignerStage3PlatformsAndPermissions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	profileName := "stackyard-signer-stage3"
	signerCreateProfileForTest(t, ts, profileName)

	resp := signerRequest(t, ts, http.MethodPost, "/signing-profiles/"+url.PathEscape(profileName)+"/permissions", mustJSON(t, map[string]any{
		"action":      "signer:StartSigningJob",
		"principal":   "123456789012",
		"statementId": "stackyard-statement",
	}))
	assertStatus(t, resp, http.StatusOK)
	var addOut struct {
		RevisionID string `json:"revisionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &addOut); err != nil {
		t.Fatalf("unmarshal add profile permission response: %v", err)
	}
	if addOut.RevisionID == "" {
		t.Fatalf("expected revisionId from add profile permission")
	}

	resp = signerRequest(t, ts, http.MethodGet, "/signing-profiles/"+url.PathEscape(profileName)+"/permissions", nil)
	assertStatus(t, resp, http.StatusOK)
	var listPermOut struct {
		Permissions []struct {
			StatementID string `json:"statementId"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listPermOut); err != nil {
		t.Fatalf("unmarshal list profile permissions response: %v", err)
	}
	if len(listPermOut.Permissions) == 0 || listPermOut.Permissions[0].StatementID != "stackyard-statement" {
		t.Fatalf("expected permission statement in list profile permissions")
	}

	removePath := "/signing-profiles/" + url.PathEscape(profileName) + "/permissions/stackyard-statement?revisionId=" + url.QueryEscape(addOut.RevisionID)
	resp = signerRequest(t, ts, http.MethodDelete, removePath, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signerRequest(t, ts, http.MethodGet, "/signing-platforms/AWSLambda-SHA384-ECDSA", nil)
	assertStatus(t, resp, http.StatusOK)
	var getPlatformOut struct {
		PlatformID string `json:"platformId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getPlatformOut); err != nil {
		t.Fatalf("unmarshal get signing platform response: %v", err)
	}
	if getPlatformOut.PlatformID != "AWSLambda-SHA384-ECDSA" {
		t.Fatalf("unexpected platform id %q", getPlatformOut.PlatformID)
	}

	resp = signerRequest(t, ts, http.MethodGet, "/signing-platforms?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	var listPlatformOut struct {
		Platforms []struct {
			PlatformID string `json:"platformId"`
		} `json:"platforms"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listPlatformOut); err != nil {
		t.Fatalf("unmarshal list signing platforms response: %v", err)
	}
	if len(listPlatformOut.Platforms) == 0 {
		t.Fatalf("expected at least one platform")
	}
}

func TestSignerStage4RevocationAndTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	profileName := "stackyard-signer-stage4"
	profile := signerCreateProfileForTest(t, ts, profileName)
	job := signerStartJobForTest(t, ts, profileName, "stackyard-stage4-token")
	jobARN := "arn:aws:signer:us-east-1:123456789012:/signing-jobs/" + job.JobID

	tagPath := "/tags/" + url.PathEscape(profile.ARN)
	resp := signerRequest(t, ts, http.MethodPost, tagPath, mustJSON(t, map[string]any{
		"tags": map[string]string{"env": "test", "team": "stackyard"},
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = signerRequest(t, ts, http.MethodGet, tagPath, nil)
	assertStatus(t, resp, http.StatusOK)
	var listTagsOut struct {
		Tags map[string]string `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsOut); err != nil {
		t.Fatalf("unmarshal list tags response: %v", err)
	}
	if len(listTagsOut.Tags) != 2 {
		t.Fatalf("expected two tags")
	}

	resp = signerRequest(t, ts, http.MethodDelete, tagPath+"?tagKeys=team", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = signerRequest(t, ts, http.MethodPut, "/signing-profiles/"+url.PathEscape(profileName)+"/revoke", mustJSON(t, map[string]any{
		"profileVersion": profile.ProfileVersion,
		"reason":         "Compromised",
		"effectiveTime":  "2024-01-01T00:00:00",
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = signerRequest(t, ts, http.MethodPut, "/signing-jobs/"+url.PathEscape(job.JobID)+"/revoke", mustJSON(t, map[string]any{
		"reason": "Compromised",
	}))
	assertStatus(t, resp, http.StatusOK)

	revocationPath := "/revocations?signatureTimestamp=2024-01-01T00:00:00&platformId=AWSLambda-SHA384-ECDSA&profileVersionArn=" +
		url.QueryEscape(profile.ProfileVersionARN) +
		"&jobArn=" + url.QueryEscape(jobARN) +
		"&certificateHashes=" + strings.Repeat("0", 64)
	resp = signerRequest(t, ts, http.MethodGet, revocationPath, nil)
	assertStatus(t, resp, http.StatusOK)
	var revocationOut struct {
		RevokedEntities []string `json:"revokedEntities"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &revocationOut); err != nil {
		t.Fatalf("unmarshal get revocation status response: %v", err)
	}
	if len(revocationOut.RevokedEntities) == 0 {
		t.Fatalf("expected revoked entities in revocation response")
	}
}

func TestSignerStage5SignPayload(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	profileName := "stackyard-signer-stage5"
	signerCreateProfileForTest(t, ts, profileName)

	resp := signerRequest(t, ts, http.MethodPost, "/signing-jobs/with-payload", mustJSON(t, map[string]any{
		"profileName":   profileName,
		"payload":       "c3RhY2t5YXJk",
		"payloadFormat": "RAW",
	}))
	assertStatus(t, resp, http.StatusOK)

	var out struct {
		JobID     string            `json:"jobId"`
		JobOwner  string            `json:"jobOwner"`
		Metadata  map[string]string `json:"metadata"`
		Signature []byte            `json:"signature"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal sign payload response: %v", err)
	}
	if out.JobID == "" || out.JobOwner == "" {
		t.Fatalf("expected job id and owner from sign payload")
	}
	if len(out.Signature) == 0 {
		t.Fatalf("expected non-empty signature in sign payload response")
	}
	if out.Metadata["profileVersionArn"] == "" {
		t.Fatalf("expected profileVersionArn metadata")
	}
}
