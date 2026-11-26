package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func cgsCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "codeguru-security")
	raw := mustBody(t, resp)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("%s %s: content type %q", method, path, ct)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
	}
	return resp.StatusCode, out
}

func cgsOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	status, out := cgsCall(t, ts, method, path, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func cgsFail(t *testing.T, ts *httptest.Server, method, path, body string, want int) string {
	t.Helper()
	status, out := cgsCall(t, ts, method, path, body)
	if status != want {
		t.Fatalf("%s %s: want %d, got %d %v", method, path, want, status, out)
	}
	code, _ := out["errorCode"].(string)
	return code
}

func cgsZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(content))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func cgsPut(t *testing.T, url, contentType string, body []byte) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// cgsUpload creates an upload URL for scan, puts files there and returns the artifact ID.
func cgsUpload(t *testing.T, ts *httptest.Server, scan string, files map[string]string) string {
	t.Helper()
	out := cgsOK(t, ts, http.MethodPost, "/uploadUrl", `{"scanName":"`+scan+`"}`)
	if status := cgsPut(t, out["s3Url"].(string), "application/zip", cgsZip(t, files)); status != http.StatusOK {
		t.Fatalf("upload: %d", status)
	}
	return out["codeArtifactId"].(string)
}

// cgsFinishRuns makes every in-progress run due.
func cgsFinishRuns(srv *Server) {
	st := srv.codeGuruSecurity
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, sc := range st.scans {
		for _, r := range sc.runs {
			r.due = r.due.Add(-time.Minute)
		}
	}
}

const cgsVulnerable = "import hashlib, os\n" +
	"KEY = \"AKIAABCDEFGHIJKLMNOP\"\n" +
	"password = \"hunter22\"\n" +
	"def run(cmd):\n" +
	"    os.system(cmd)  # TODO validate\n" +
	"    return hashlib.md5(cmd.encode()).hexdigest()\n"

func TestCodeGuruSecurityStage0Routing(t *testing.T) {
	_, ts := newC9TestServer(t)
	model := awsmodels.CodeGuruSecurityModel()
	if len(cgsOperations) != len(model.Operations) || len(cgsHandlers) != len(cgsOperations) {
		t.Fatalf("catalog %d, model %d, handlers %d", len(cgsOperations), len(model.Operations), len(cgsHandlers))
	}
	for _, op := range cgsOperations {
		if _, ok := model.Operations[op.Name]; !ok || cgsHandlers[op.Name] == nil {
			t.Errorf("%s is not in the model or has no handler", op.Name)
		}
	}
	cgsOK(t, ts, http.MethodGet, "/scans", "")

	// CodeGuru Reviewer keeps /tags and /associations.
	arn := "arn:aws:codeguru-reviewer:us-east-1:123456789012:association:a1"
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/associations", nil, nil, "codeguru-reviewer")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "RepositoryAssociationSummaries") {
		t.Fatalf("reviewer associations: %d %s", resp.StatusCode, body)
	}
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/tags/"+strings.ReplaceAll(arn, "/", "%2F"), nil, nil, "codeguru-reviewer")
	if body := mustBody(t, resp); strings.Contains(string(body), "errorCode") {
		t.Fatalf("reviewer tags served by CodeGuru Security: %s", body)
	}

	if code := cgsFail(t, ts, http.MethodGet, "/scans?maxResults=0", "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("maxResults 0: %s", code)
	}
	if code := cgsFail(t, ts, http.MethodGet, "/metrics/findings?startDate=2026-01-01", "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("missing endDate: %s", code)
	}
	if code := cgsFail(t, ts, http.MethodPost, "/scans", `{"scanName":"x","resourceId":{"codeArtifactId":"00000000-0000-0000-0000-000000000000"},"bogus":1}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
}

func TestCodeGuruSecurityStage1Uploads(t *testing.T) {
	_, ts := newC9TestServer(t)
	out := cgsOK(t, ts, http.MethodPost, "/uploadUrl", `{"scanName":"up"}`)
	url := out["s3Url"].(string)
	if h := out["requestHeaders"].(map[string]any); h["Content-Type"] != "application/zip" {
		t.Fatalf("headers: %v", h)
	}
	zipped := cgsZip(t, map[string]string{"a.py": "x = 1\n"})
	if status := cgsPut(t, url, "text/html", zipped); status != http.StatusBadRequest {
		t.Fatalf("wrong content type: %d", status)
	}
	if status := cgsPut(t, strings.Replace(url, "key=", "key=0", 1), "application/zip", zipped); status != http.StatusForbidden {
		t.Fatalf("wrong key: %d", status)
	}
	if status := cgsPut(t, url, "application/zip", []byte("not a zip")); status != http.StatusBadRequest {
		t.Fatalf("not a zip: %d", status)
	}
	if status := cgsPut(t, url, "application/zip", make([]byte, cgsMaxUpload+1)); status != http.StatusBadRequest {
		t.Fatalf("oversized: %d", status)
	}

	if m := cgsScanFile("cfg.py", ".py", []byte("DB_PASSWORD = \"abcd1234\"\n"), nil); len(m) != 1 || cgsRules[m[0].rule].id != "hardcoded-password" {
		t.Fatalf("suffixed password name: %+v", m)
	}

	// Matches are capped however many the archive holds.
	matches, files, err := cgsScanZip(cgsZip(t, map[string]string{"big.py": strings.Repeat("# TODO\n", 2*cgsMaxMatches), "skip.bin": "AKIAABCDEFGHIJKLMNOP"}))
	if err != nil || len(matches) != cgsMaxMatches || files != 1 {
		t.Fatalf("matches %d, files %d, err %v", len(matches), files, err)
	}
}

func TestCodeGuruSecurityStage1ScansAndFindings(t *testing.T) {
	srv, ts := newC9TestServer(t)
	art := cgsUpload(t, ts, "app", map[string]string{"src/app.py": cgsVulnerable})
	if code := cgsFail(t, ts, http.MethodPost, "/scans", `{"scanName":"other","resourceId":{"codeArtifactId":"`+art+`"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("artifact of another scan: %s", code)
	}
	if code := cgsFail(t, ts, http.MethodPost, "/scans", `{"scanName":"app","resourceId":{"codeArtifactId":"00000000-0000-0000-0000-000000000000"}}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("unknown artifact: %s", code)
	}
	run := cgsOK(t, ts, http.MethodPost, "/scans", `{"scanName":"app","resourceId":{"codeArtifactId":"`+art+`"},"analysisType":"All","tags":{"team":"a"}}`)
	if run["scanState"] != "InProgress" {
		t.Fatalf("create: %v", run)
	}
	if code := cgsFail(t, ts, http.MethodGet, "/findings/app", "", http.StatusConflict); code != "ConflictException" {
		t.Fatalf("findings while in progress: %s", code)
	}
	if code := cgsFail(t, ts, http.MethodPost, "/scans", `{"scanName":"app","resourceId":{"codeArtifactId":"`+art+`"}}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("second run while in progress: %s", code)
	}
	cgsFinishRuns(srv)
	if got := cgsOK(t, ts, http.MethodGet, "/scans/app", ""); got["scanState"] != "Successful" || got["numberOfRevisions"] != float64(1) {
		t.Fatalf("scan: %v", got)
	}
	findings := cgsOK(t, ts, http.MethodGet, "/findings/app", "")["findings"].([]any)
	var rules []string
	for _, f := range findings {
		rules = append(rules, f.(map[string]any)["ruleId"].(string))
	}
	want := "aws-hardcoded-access-key hardcoded-password os-command-injection weak-hash-algorithm unresolved-todo"
	if strings.Join(rules, " ") != want {
		t.Fatalf("findings: %v", rules)
	}
	first := findings[0].(map[string]any)
	if fp := first["vulnerability"].(map[string]any)["filePath"].(map[string]any); fp["startLine"] != float64(2) || fp["name"] != "app.py" || len(fp["codeSnippet"].([]any)) != 3 {
		t.Fatalf("file path: %v", fp)
	}

	// A security-only rescan of fixed code closes what went away.
	fixed := strings.Replace(cgsVulnerable, "KEY = \"AKIAABCDEFGHIJKLMNOP\"", "KEY = os.environ[\"KEY\"]", 1)
	art2 := cgsUpload(t, ts, "app", map[string]string{"src/app.py": fixed})
	cgsOK(t, ts, http.MethodPost, "/scans", `{"scanName":"app","resourceId":{"codeArtifactId":"`+art2+`"},"analysisType":"Security"}`)
	cgsFinishRuns(srv)
	closed := cgsOK(t, ts, http.MethodGet, "/findings/app?status=Closed", "")["findings"].([]any)
	if len(closed) != 2 {
		t.Fatalf("closed: %v", closed)
	}
	open := cgsOK(t, ts, http.MethodGet, "/findings/app?status=Open&maxResults=2", "")
	if len(open["findings"].([]any)) != 2 || open["nextToken"] == nil {
		t.Fatalf("open page: %v", open)
	}
	id := first["id"].(string)
	batch := cgsOK(t, ts, http.MethodPost, "/batchGetFindings", `{"findingIdentifiers":[{"scanName":"app","findingId":"`+id+`"},{"scanName":"app","findingId":"`+id+`"},{"scanName":"nope","findingId":"x"}]}`)
	if len(batch["findings"].([]any)) != 1 || len(batch["failedFindings"].([]any)) != 2 {
		t.Fatalf("batch: %v", batch)
	}

	today := time.Now().UTC().Format("2006-01-02")
	days := cgsOK(t, ts, http.MethodGet, "/metrics/findings?startDate="+today+"&endDate="+today, "")["findingsMetrics"].([]any)
	if len(days) != 1 {
		t.Fatalf("metrics: %v", days)
	}
	d := days[0].(map[string]any)
	if d["closedFindings"].(map[string]any)["critical"] != float64(1) || d["openFindings"].(map[string]any)["high"] != float64(2) {
		t.Fatalf("day: %v", d)
	}
	sum := cgsOK(t, ts, http.MethodGet, "/metrics/summary?date="+today, "")["metricsSummary"].(map[string]any)
	if top := sum["scansWithMostOpenFindings"].([]any); len(top) != 1 || top[0].(map[string]any)["findingNumber"] != float64(3) {
		t.Fatalf("summary: %v", sum)
	}

	// Nothing uploaded: the run fails.
	empty := cgsOK(t, ts, http.MethodPost, "/uploadUrl", `{"scanName":"empty"}`)["codeArtifactId"].(string)
	cgsOK(t, ts, http.MethodPost, "/scans", `{"scanName":"empty","resourceId":{"codeArtifactId":"`+empty+`"}}`)
	cgsFinishRuns(srv)
	if got := cgsOK(t, ts, http.MethodGet, "/scans/empty", ""); got["scanState"] != "Failed" || got["errorMessage"] == nil {
		t.Fatalf("empty scan: %v", got)
	}

	arn := strings.ReplaceAll(cgsScanArn("app"), "/", "%2F")
	cgsOK(t, ts, http.MethodPost, "/tags/"+arn, `{"tags":{"env":"dev"}}`)
	cgsOK(t, ts, http.MethodDelete, "/tags/"+arn+"?tagKeys=team", "")
	if tags := cgsOK(t, ts, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any); len(tags) != 1 || tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
}

func TestCodeGuruSecurityStage1AccountConfiguration(t *testing.T) {
	srv, ts := newC9TestServer(t)
	if cfg := cgsOK(t, ts, http.MethodGet, "/accountConfiguration/get", "")["encryptionConfig"].(map[string]any); len(cfg) != 0 {
		t.Fatalf("default: %v", cfg)
	}
	missing := `{"encryptionConfig":{"kmsKeyArn":"arn:aws:kms:us-east-1:123456789012:key/11111111-2222-3333-4444-555555555555"}}`
	if code := cgsFail(t, ts, http.MethodPut, "/updateAccountConfiguration", missing, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("missing key: %s", code)
	}
	key, err := srv.kms.CreateKey("codeguru", "ENCRYPT_DECRYPT", "SYMMETRIC_DEFAULT")
	if err != nil {
		t.Fatal(err)
	}
	cgsOK(t, ts, http.MethodPut, "/updateAccountConfiguration", `{"encryptionConfig":{"kmsKeyArn":"`+key.Arn+`"}}`)
	if cfg := cgsOK(t, ts, http.MethodGet, "/accountConfiguration/get", "")["encryptionConfig"].(map[string]any); cfg["kmsKeyArn"] != key.Arn {
		t.Fatalf("configured: %v", cfg)
	}
}
