package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// idaCall sends a signed Device Advisor request and decodes the reply.
func idaCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "iotdeviceadvisor")
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

func idaOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	status, out := idaCall(t, ts, method, path, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func idaLooksLikeList(body []byte) bool {
	return strings.Contains(string(body), "suiteDefinitionInformationList")
}

const idaRoot = `{\"configuration\":{},\"tests\":[{\"name\":\"MQTT Connect\",\"configuration\":{},\"tests\":[{\"name\":\"MQTT_Connect\",\"configuration\":{},\"test\":{\"id\":\"MQTT_Connect\",\"version\":\"0.0.0\"}}]}]}`

func idaSuiteBody(name string, qualification bool) string {
	root := idaRoot
	if qualification {
		root = ""
	}
	q := "false"
	if qualification {
		q = "true"
	}
	return `{"suiteDefinitionConfiguration":{"suiteDefinitionName":"` + name + `","rootGroup":"` + root +
		`","devicePermissionRoleArn":"arn:aws:iam::123456789012:role/device-advisor","intendedForQualification":` + q +
		`,"devices":[{"thingArn":"arn:aws:iot:us-east-1:123456789012:thing/pump-1"}]},"tags":{"team":"qa"}}`
}

const idaRunBody = `{"suiteRunConfiguration":{"primaryDevice":{"thingArn":"arn:aws:iot:us-east-1:123456789012:thing/pump-1"}}}`

func TestIoTDeviceAdvisorStage1SuiteLifecycle(t *testing.T) {
	_, ts := newBudTestServer(t)
	created := idaOK(t, ts, http.MethodPost, "/suiteDefinitions", idaSuiteBody("mqtt", false))
	id := created["suiteDefinitionId"].(string)
	updated := idaOK(t, ts, http.MethodPatch, "/suiteDefinitions/"+id, `{"suiteDefinitionConfiguration":{"suiteDefinitionName":"mqtt-v2","rootGroup":"`+idaRoot+`","devicePermissionRoleArn":"arn:aws:iam::123456789012:role/device-advisor"}}`)
	if updated["suiteDefinitionVersion"] != "v2" {
		t.Fatalf("update: %v", updated)
	}
	got := idaOK(t, ts, http.MethodGet, "/suiteDefinitions/"+id+"?suiteDefinitionVersion=v1", "")
	if got["latestVersion"] != "v2" || got["suiteDefinitionConfiguration"].(map[string]any)["suiteDefinitionName"] != "mqtt" {
		t.Fatalf("get v1: %v", got)
	}
	if status, out := idaCall(t, ts, http.MethodGet, "/suiteDefinitions/"+id+"?suiteDefinitionVersion=v9", ""); status != http.StatusNotFound {
		t.Fatalf("missing version: %d %v", status, out)
	}
	if status, out := idaCall(t, ts, http.MethodPost, "/suiteDefinitions", `{"suiteDefinitionConfiguration":{"suiteDefinitionName":"x","rootGroup":"not json","devicePermissionRoleArn":"arn:aws:iam::123456789012:role/device-advisor"}}`); status != http.StatusBadRequest || out["__code"] != "ValidationException" {
		t.Fatalf("bad rootGroup: %d %v", status, out)
	}

	run := idaOK(t, ts, http.MethodPost, "/suiteDefinitions/"+id+"/suiteRuns", idaRunBody)
	runID := run["suiteRunId"].(string)
	if status, out := idaCall(t, ts, http.MethodPost, "/suiteDefinitions/"+id+"/suiteRuns", idaRunBody); status != http.StatusConflict {
		t.Fatalf("second run while one is active: %d %v", status, out)
	}
	runPath := "/suiteDefinitions/" + id + "/suiteRuns/" + runID
	var statuses []string
	for i := 0; i < 3; i++ {
		statuses = append(statuses, idaOK(t, ts, http.MethodGet, runPath, "")["status"].(string))
	}
	if strings.Join(statuses, ",") != "RUNNING,PASS,PASS" {
		t.Fatalf("run statuses: %v", statuses)
	}
	res := idaOK(t, ts, http.MethodGet, runPath, "")["testResult"].(map[string]any)["groups"].([]any)
	if len(res) != 1 || res[0].(map[string]any)["groupName"] != "MQTT Connect" {
		t.Fatalf("test result: %v", res)
	}
	// Not a qualification suite: no report.
	if status, _ := idaCall(t, ts, http.MethodGet, runPath+"/report", ""); status != http.StatusBadRequest {
		t.Fatalf("report for a non-qualification run: %d", status)
	}
	if status, _ := idaCall(t, ts, http.MethodPost, runPath+"/stop", ""); status != http.StatusBadRequest {
		t.Fatalf("stop a finished run: %d", status)
	}
	runs := idaOK(t, ts, http.MethodGet, "/suiteRuns?suiteDefinitionId="+id, "")["suiteRunsList"].([]any)
	if len(runs) != 1 || runs[0].(map[string]any)["passed"].(float64) != 1 {
		t.Fatalf("ListSuiteRuns: %v", runs)
	}

	arn := url.PathEscape(created["suiteDefinitionArn"].(string))
	idaOK(t, ts, http.MethodPost, "/tags/"+arn, `{"tags":{"env":"dev"}}`)
	idaOK(t, ts, http.MethodDelete, "/tags/"+arn+"?tagKeys=team", "")
	tags := idaOK(t, ts, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any)
	if len(tags) != 1 || tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
	idaOK(t, ts, http.MethodDelete, "/suiteDefinitions/"+id, "")
	if status, _ := idaCall(t, ts, http.MethodGet, runPath, ""); status != http.StatusNotFound {
		t.Fatalf("run of a deleted suite: %d", status)
	}
}

func TestIoTDeviceAdvisorStage1QualificationAndStop(t *testing.T) {
	_, ts := newBudTestServer(t)
	id := idaOK(t, ts, http.MethodPost, "/suiteDefinitions", idaSuiteBody("qual", true))["suiteDefinitionId"].(string)
	runID := idaOK(t, ts, http.MethodPost, "/suiteDefinitions/"+id+"/suiteRuns", idaRunBody)["suiteRunId"].(string)
	runPath := "/suiteDefinitions/" + id + "/suiteRuns/" + runID
	idaOK(t, ts, http.MethodGet, runPath, "")
	idaOK(t, ts, http.MethodGet, runPath, "")
	report := idaOK(t, ts, http.MethodGet, runPath+"/report", "")
	if !strings.HasPrefix(report["qualificationReportDownloadUrl"].(string), "https://") {
		t.Fatalf("report: %v", report)
	}
	runID = idaOK(t, ts, http.MethodPost, "/suiteDefinitions/"+id+"/suiteRuns", idaRunBody)["suiteRunId"].(string)
	runPath = "/suiteDefinitions/" + id + "/suiteRuns/" + runID
	idaOK(t, ts, http.MethodPost, runPath+"/stop", "")
	if s := idaOK(t, ts, http.MethodGet, runPath, "")["status"]; s != "STOPPED" {
		t.Fatalf("stopped run: %v", s)
	}
	ep := idaOK(t, ts, http.MethodGet, "/endpoint?thingArn=arn:aws:iot:us-east-1:123456789012:thing/pump-1", "")["endpoint"].(string)
	if !strings.Contains(ep, "deviceadvisor.iot") {
		t.Fatalf("endpoint: %s", ep)
	}
	if status, _ := idaCall(t, ts, http.MethodGet, "/endpoint?authenticationMethod=SignatureVersion4", ""); status != http.StatusBadRequest {
		t.Fatalf("SigV4 endpoint without a role: %d", status)
	}
	if status, out := idaCall(t, ts, http.MethodGet, "/suiteDefinitions?maxResults=0", ""); status != http.StatusBadRequest {
		t.Fatalf("maxResults=0: %d %v", status, out)
	}
}
