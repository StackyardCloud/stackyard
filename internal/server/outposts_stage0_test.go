package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// edgeCall sends a signed REST JSON request and decodes the reply; the error
// code (X-Amzn-ErrorType) comes back as out["__code"].
func edgeCall(t *testing.T, ts *httptest.Server, signing, method, path, body string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, signing)
	data := mustBody(t, resp)
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	if code := resp.Header.Get("X-Amzn-ErrorType"); code != "" {
		out["__code"] = code
	}
	return resp.StatusCode, out
}

func edgeOK(t *testing.T, ts *httptest.Server, signing, method, path, body string) map[string]any {
	t.Helper()
	status, out := edgeCall(t, ts, signing, method, path, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func edgeErr(t *testing.T, ts *httptest.Server, signing, method, path, body string, want int) string {
	t.Helper()
	status, out := edgeCall(t, ts, signing, method, path, body)
	if status != want {
		t.Fatalf("%s %s: want %d, got %d %v", method, path, want, status, out)
	}
	code, _ := out["__code"].(string)
	return code
}

func TestEdgeBatchStage0CatalogsMatchModels(t *testing.T) {
	names := func(routes []opstRoute) []string {
		out := []string{}
		for _, r := range routes {
			out = append(out, r.Name)
		}
		return out
	}
	bgw := []string{}
	for _, op := range backupGatewayOperations {
		bgw = append(bgw, op.Name)
	}
	for _, c := range []struct {
		name     string
		ops      []string
		model    *awsmodels.Model
		handlers int
	}{
		{"outposts", names(outpostsOperations), awsmodels.OutpostsModel(), len(opstHandlers)},
		{"snowdevicemanagement", names(snowDeviceManagementOperations), awsmodels.SnowDeviceManagementModel(), len(sdmHandlers)},
		{"backupgateway", bgw, awsmodels.BackupGatewayModel(), len(bgwAPI.handlers)},
	} {
		if len(c.ops) != len(c.model.Operations) || c.handlers != len(c.ops) {
			t.Fatalf("%s: catalog %d, model %d, handlers %d", c.name, len(c.ops), len(c.model.Operations), c.handlers)
		}
		for _, op := range c.ops {
			if _, ok := c.model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", c.name, op)
			}
		}
	}
}

// TestEdgeBatchStage0Routing proves the three services claim only their own
// traffic and Backup, EC2 and Snowball keep theirs.
func TestEdgeBatchStage0Routing(t *testing.T) {
	_, ts := newBudTestServer(t)
	edgeOK(t, ts, "outposts", http.MethodGet, "/outposts", "")
	edgeOK(t, ts, "snow-device-management", http.MethodGet, "/managed-devices", "")
	budOKAPI(t, ts, bgwAPI, "ListGateways", `{}`)

	// Backup keeps its REST paths.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/backup-vaults/", nil, nil, "backup")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "BackupVaultList") {
		t.Fatalf("backup ListBackupVaults: %d %s", resp.StatusCode, body)
	}
	// EC2 keeps its query API.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte("Action=DescribeRegions&Version=2016-11-15"), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded"}, "ec2")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "DescribeRegionsResponse") {
		t.Fatalf("ec2 DescribeRegions: %d %s", resp.StatusCode, body)
	}
	// Snowball keeps its JSON target.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSIESnowballJobManagementService.ListJobs"}, "snowball")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "JobListEntries") {
		t.Fatalf("snowball ListJobs: %d %s", resp.StatusCode, body)
	}
	// Outposts paths signed for another service are not served by Outposts.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/outposts", nil, nil, "ec2")
	if body := mustBody(t, resp); strings.Contains(string(body), `"Outposts"`) {
		t.Fatalf("ec2-signed request was served by Outposts: %s", body)
	}
	// A Backup gateway target signed for Backup is refused.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "BackupOnPremises_v20210101.ListGateways"}, "backup")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("backup-signed gateway request: %d %s", resp.StatusCode, body)
	}
	// Unknown members and chunked bodies are refused.
	if code := budErrAPI(t, ts, bgwAPI, "ListGateways", `{"Nope":1}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
	if code := edgeErr(t, ts, "outposts", http.MethodPost, "/sites", `{"Name":"x","Nope":1}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/sites", strings.NewReader(`{"Name":"x"}`))
	req.ContentLength = -1
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=x/20260101/us-east-1/outposts/aws4_request, SignedHeaders=host, Signature=x")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if body := mustBody(t, chunked); chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d %s", chunked.StatusCode, body)
	}
}
