package server

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func acpDefJSON(id, bucket, format, frequency string) string {
	return `{"reportId":"` + id + `","reportDescription":"tenant costs","reportFrequency":"` + frequency + `","format":"` + format +
		`","destinationS3Location":{"bucket":"` + bucket + `","prefix":"reports/"}}`
}

func TestApplicationCostProfilerStage1Definitions(t *testing.T) {
	srv, ts := newBudTestServer(t)
	if status, out := acpCall(t, ts, http.MethodPost, "/reportDefinition", acpDefJSON("daily", "missing-bucket", "CSV", "DAILY")); status != http.StatusBadRequest || out["__code"] != "ValidationException" || out["message"] == nil {
		t.Fatalf("missing bucket: %d %v", status, out)
	}
	if _, err := srv.s3.CreateBucket("tenant-reports"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"daily", "monthly"} {
		if status, out := acpCall(t, ts, http.MethodPost, "/reportDefinition", acpDefJSON(id, "tenant-reports", "CSV", "DAILY")); status != http.StatusOK || out["reportId"] != id {
			t.Fatalf("put %s: %d %v", id, status, out)
		}
	}
	if status, _ := acpCall(t, ts, http.MethodPost, "/reportDefinition", acpDefJSON("daily", "tenant-reports", "CSV", "DAILY")); status != http.StatusBadRequest {
		t.Fatalf("duplicate: %d", status)
	}
	if status, _ := acpCall(t, ts, http.MethodPost, "/reportDefinition", acpDefJSON("x", "tenant-reports", "XML", "DAILY")); status != http.StatusBadRequest {
		t.Fatalf("bad format: %d", status)
	}
	status, out := acpCall(t, ts, http.MethodPut, "/reportDefinition/monthly", strings.Replace(acpDefJSON("monthly", "tenant-reports", "PARQUET", "MONTHLY"), `"reportId":"monthly",`, "", 1))
	if status != http.StatusOK {
		t.Fatalf("update: %d %v", status, out)
	}
	status, got := acpCall(t, ts, http.MethodGet, "/reportDefinition/monthly", "")
	if status != http.StatusOK || got["format"] != "PARQUET" || got["lastUpdated"] == nil || got["destinationS3Location"].(map[string]any)["prefix"] != "reports" {
		t.Fatalf("get: %d %v", status, got)
	}
	status, page := acpCall(t, ts, http.MethodGet, "/reportDefinition?maxResults=1", "")
	defs := page["reportDefinitions"].([]any)
	if status != http.StatusOK || len(defs) != 1 || defs[0].(map[string]any)["lastUpdatedAt"] == nil || page["nextToken"] == nil {
		t.Fatalf("page: %d %v", status, page)
	}
	_, page = acpCall(t, ts, http.MethodGet, "/reportDefinition?maxResults=1&nextToken="+page["nextToken"].(string), "")
	if defs := page["reportDefinitions"].([]any); len(defs) != 1 || defs[0].(map[string]any)["reportId"] != "monthly" || page["nextToken"] != nil {
		t.Fatalf("page 2: %v", page)
	}
	if status, _ := acpCall(t, ts, http.MethodDelete, "/reportDefinition/monthly", ""); status != http.StatusOK {
		t.Fatalf("delete: %d", status)
	}
	if status, _ := acpCall(t, ts, http.MethodGet, "/reportDefinition/monthly", ""); status != http.StatusBadRequest {
		t.Fatalf("get deleted: %d", status)
	}
	// The definition quota holds.
	srv.applicationCostProfiler.mu.Lock()
	for i := len(srv.applicationCostProfiler.defs); i < acpMaxDefinitions; i++ {
		srv.applicationCostProfiler.defs[strings.Repeat("z", i+1)] = &acpDefinition{id: strings.Repeat("z", i+1)}
	}
	srv.applicationCostProfiler.mu.Unlock()
	if status, out := acpCall(t, ts, http.MethodPost, "/reportDefinition", acpDefJSON("over", "tenant-reports", "CSV", "DAILY")); status != http.StatusPaymentRequired || out["__code"] != "ServiceQuotaExceededException" {
		t.Fatalf("quota: %d %v", status, out)
	}
}

func TestApplicationCostProfilerStage1ImportWritesReports(t *testing.T) {
	srv, ts := newBudTestServer(t)
	for _, b := range []string{"tenant-reports", "usage"} {
		if _, err := srv.s3.CreateBucket(b); err != nil {
			t.Fatal(err)
		}
	}
	acpCall(t, ts, http.MethodPost, "/reportDefinition", acpDefJSON("daily", "tenant-reports", "CSV", "DAILY"))
	usage := "ApplicationId,TenantId,TenantDesc,ResourceId,StartTime,EndTime\n" +
		"app,a,Tenant A,i-1,2026-01-01T00:00:00Z,2026-01-01T03:00:00Z\n" +
		"app,b,Tenant B,i-1,1767225600,1767229200\n"
	if _, err := srv.s3.PutObject("usage", "usage.csv", "text/csv", []byte(usage), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	status, out := acpCall(t, ts, http.MethodPost, "/importApplicationUsage", `{"sourceS3Location":{"bucket":"usage","key":"usage.csv"}}`)
	if status != http.StatusOK || out["importId"] == nil {
		t.Fatalf("import: %d %v", status, out)
	}
	key := "reports/daily/" + time.Now().UTC().Format("2006/01/02") + "/" + out["importId"].(string) + ".csv"
	obj, err := srv.s3.GetObject("tenant-reports", key)
	if err != nil {
		t.Fatalf("report %s: %v", key, err)
	}
	want := "ApplicationId,TenantId,TenantDesc,ResourceId,UsageHours,UsagePercentage\n" +
		"app,a,Tenant A,i-1,3.0000,75.0000\napp,b,Tenant B,i-1,1.0000,25.0000\n"
	if string(obj.Body) != want {
		t.Fatalf("report:\n%s", obj.Body)
	}
	for name, body := range map[string]string{
		"no header column": "ApplicationId,TenantId\napp,a\n",
		"end before start": "ApplicationId,TenantId,ResourceId,StartTime,EndTime\napp,a,i-1,2026-01-02T00:00:00Z,2026-01-01T00:00:00Z\n",
	} {
		_, _ = srv.s3.PutObject("usage", "bad.csv", "text/csv", []byte(body), nil, "", "")
		if status, _ := acpCall(t, ts, http.MethodPost, "/importApplicationUsage", `{"sourceS3Location":{"bucket":"usage","key":"bad.csv"}}`); status != http.StatusBadRequest {
			t.Fatalf("%s: %d", name, status)
		}
	}
	if status, _ := acpCall(t, ts, http.MethodPost, "/importApplicationUsage", `{"sourceS3Location":{"bucket":"usage","key":"none.csv"}}`); status != http.StatusBadRequest {
		t.Fatalf("missing object: %d", status)
	}
}
