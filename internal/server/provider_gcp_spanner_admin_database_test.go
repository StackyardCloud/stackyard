package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSpannerAdminDatabaseRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	instance := "/gcp/v1/projects/stackyard/instances/stackyard-instance"
	database := instance + "/databases/stackyard-db"
	backup := instance + "/backups/backup-1"
	schedule := database + "/backupSchedules/daily-full"
	operation := instance + "/operations/create-database-stackyard-db"

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, instance+"/databases?pageSize=1", nil, "databases")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, instance+"/databases", []byte("{\"createStatement\":\"CREATE DATABASE `stackyard-db`\"}"), "operations/create-database-stackyard-db")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, database, nil, "stackyard-db")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPatch, database+"?updateMask=enableDropProtection", []byte(`{"database":{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db","enableDropProtection":false}}`), "enableDropProtection")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPatch, database+"/ddl", []byte("{\"database\":\"projects/stackyard/instances/stackyard-instance/databases/stackyard-db\",\"statements\":[\"ALTER DATABASE `stackyard-db` SET OPTIONS (version_retention_period='24h')\"]}"), "UpdateDatabaseDdlMetadata")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, database+"/ddl", nil, "CREATE TABLE")

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, database+":setIamPolicy", []byte(`{"policy":{"bindings":[{"role":"roles/spanner.databaseAdmin","members":["user:stackyard@example.com"]}]}}`), "bindings")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, database+":getIamPolicy", nil, "roles/spanner.databaseAdmin")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, database+":testIamPermissions", []byte(`{"permissions":["spanner.databases.get","resourcemanager.projects.get"]}`), "spanner.databases.get")

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, instance+"/backups?backupId=backup-1", []byte(`{"backup":{"database":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db","expireTime":"2026-01-02T00:00:00Z"}}`), "create-backup-backup-1")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, instance+"/backups:copy", []byte(`{"backupId":"backup-copy-1","sourceBackup":"projects/stackyard/instances/stackyard-instance/backups/backup-1","expireTime":"2026-01-03T00:00:00Z"}`), "copy-backup-backup-copy-1")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, backup, nil, "backup-1")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPatch, backup+"?updateMask=expireTime", []byte(`{"backup":{"name":"projects/stackyard/instances/stackyard-instance/backups/backup-1","expireTime":"2026-01-04T00:00:00Z"}}`), "2026-01-04T00:00:00Z")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, instance+"/backups?pageSize=1", nil, "backups")

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, instance+"/databases:restore", []byte(`{"databaseId":"restored-db","backup":"projects/stackyard/instances/stackyard-instance/backups/backup-1"}`), "restore-database-restored-db")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, instance+"/databaseOperations?pageSize=1", nil, "operations")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, instance+"/backupOperations?pageSize=1", nil, "operations")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, database+"/databaseRoles?pageSize=1", nil, "databaseRoles")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, database+":addSplitPoints", []byte(`{"splitPoints":[{"index":"UsersByName"}]}`), "{}")

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, database+"/backupSchedules?backupScheduleId=daily-full", []byte(`{"backupSchedule":{"retentionDuration":"172800s"}}`), "backupSchedules/daily-full")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, schedule, nil, "daily-full")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPatch, schedule+"?updateMask=retentionDuration", []byte(`{"backupSchedule":{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/backupSchedules/daily-full","retentionDuration":"86400s"}}`), "86400s")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, database+"/backupSchedules?pageSize=1", nil, "backupSchedules")

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, operation, nil, "create-database-stackyard-db")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodGet, instance+"/operations?pageSize=1", nil, "operations")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodPost, operation+":cancel", []byte(`{}`), "{}")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodDelete, operation, nil, "{}")

	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodDelete, schedule, nil, "{}")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodDelete, backup, nil, "{}")
	assertGCPSpannerAdminDatabaseSuccess(t, ts, http.MethodDelete, database, nil, "{}")
}

func TestGCPSpannerAdminDatabaseRouter_ListDatabasesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin database list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminDatabaseRouter_CreateDatabaseRequiresCreateStatement(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin database create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminDatabaseRouter_UpdateDatabaseNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db?updateMask=enableDropProtection", []byte(`{"database":{"name":"projects/stackyard/instances/other-instance/databases/stackyard-db","enableDropProtection":true}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin database update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminDatabaseRouter_CreateBackupRequiresBackupPayload(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/backups?backupId=backup-1", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin database create backup, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminDatabaseRouter_CopyBackupRequiresReadySource(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/backups:copy", []byte(`{"backupId":"copy-1","sourceBackup":"projects/stackyard/instances/stackyard-instance/backups/creating-backup","expireTime":"2026-01-03T00:00:00Z"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner admin database copy backup, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminDatabaseRouter_RestoreDatabaseBackupNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases:restore", []byte(`{"databaseId":"restored-db","backup":"projects/stackyard/instances/stackyard-instance/backups/missing-backup"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp spanner admin database restore database, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminDatabaseRouter_CreateBackupScheduleAlreadyExists(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/backupSchedules?backupScheduleId=existing-schedule", []byte(`{"backupSchedule":{"retentionDuration":"172800s"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp spanner admin database create backup schedule, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"AlreadyExists"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdminDatabaseRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdminDatabaseContractServer(t)
	instance := "/gcp/v1/projects/stackyard/instances/stackyard-instance"
	database := instance + "/databases/stackyard-db"
	schedule := database + "/backupSchedules/daily-full"
	operation := instance + "/operations/create-database-stackyard-db"

	listDBResp := providerContractRequest(t, ts, http.MethodGet, instance+"/databases?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if listDBResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list databases, got %d body=%s", listDBResp.StatusCode, string(providerContractBody(t, listDBResp)))
	}
	listDBBody := providerContractJSONMap(t, listDBResp)
	databases, ok := listDBBody["databases"].([]any)
	if !ok || len(databases) == 0 {
		t.Fatalf("expected databases array, got %#v", listDBBody["databases"])
	}
	firstDatabase, _ := databases[0].(map[string]any)
	if _, ok := firstDatabase["name"].(string); !ok {
		t.Fatalf("expected databases[0].name string, got %#v", firstDatabase["name"])
	}
	if _, ok := firstDatabase["enableDropProtection"].(bool); !ok {
		t.Fatalf("expected databases[0].enableDropProtection bool, got %#v", firstDatabase["enableDropProtection"])
	}

	getBackupResp := providerContractRequest(t, ts, http.MethodGet, instance+"/backups/backup-1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if getBackupResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get backup, got %d body=%s", getBackupResp.StatusCode, string(providerContractBody(t, getBackupResp)))
	}
	getBackupBody := providerContractJSONMap(t, getBackupResp)
	if _, ok := getBackupBody["database"].(string); !ok {
		t.Fatalf("expected backup.database string, got %#v", getBackupBody["database"])
	}
	if _, ok := getBackupBody["sizeBytes"].(string); !ok {
		t.Fatalf("expected backup.sizeBytes string, got %#v", getBackupBody["sizeBytes"])
	}

	getScheduleResp := providerContractRequest(t, ts, http.MethodGet, schedule, nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if getScheduleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get backup schedule, got %d body=%s", getScheduleResp.StatusCode, string(providerContractBody(t, getScheduleResp)))
	}
	getScheduleBody := providerContractJSONMap(t, getScheduleResp)
	if _, ok := getScheduleBody["name"].(string); !ok {
		t.Fatalf("expected schedule.name string, got %#v", getScheduleBody["name"])
	}
	spec, _ := getScheduleBody["spec"].(map[string]any)
	if _, ok := spec["cronSpec"].(map[string]any); !ok {
		t.Fatalf("expected schedule.spec.cronSpec object, got %#v", spec["cronSpec"])
	}

	listRolesResp := providerContractRequest(t, ts, http.MethodGet, database+"/databaseRoles?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if listRolesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list database roles, got %d body=%s", listRolesResp.StatusCode, string(providerContractBody(t, listRolesResp)))
	}
	listRolesBody := providerContractJSONMap(t, listRolesResp)
	roles, _ := listRolesBody["databaseRoles"].([]any)
	if len(roles) == 0 {
		t.Fatalf("expected databaseRoles array, got %#v", listRolesBody["databaseRoles"])
	}
	firstRole, _ := roles[0].(map[string]any)
	if _, ok := firstRole["name"].(string); !ok {
		t.Fatalf("expected databaseRoles[0].name string, got %#v", firstRole["name"])
	}

	getOperationResp := providerContractRequest(t, ts, http.MethodGet, operation, nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if getOperationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get operation, got %d body=%s", getOperationResp.StatusCode, string(providerContractBody(t, getOperationResp)))
	}
	getOperationBody := providerContractJSONMap(t, getOperationResp)
	if _, ok := getOperationBody["name"].(string); !ok {
		t.Fatalf("expected operation.name string, got %#v", getOperationBody["name"])
	}
	if _, ok := getOperationBody["done"].(bool); !ok {
		t.Fatalf("expected operation.done bool, got %#v", getOperationBody["done"])
	}

	testIAMResp := providerContractRequest(t, ts, http.MethodPost, database+":testIamPermissions", []byte(`{"permissions":["spanner.databases.get"]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if testIAMResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from test iam permissions, got %d body=%s", testIAMResp.StatusCode, string(providerContractBody(t, testIAMResp)))
	}
	testIAMBody := providerContractJSONMap(t, testIAMResp)
	permissions, _ := testIAMBody["permissions"].([]any)
	if len(permissions) == 0 {
		t.Fatalf("expected permissions array, got %#v", testIAMBody["permissions"])
	}

	getDDLResp := providerContractRequest(t, ts, http.MethodGet, database+"/ddl", nil, map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	})
	if getDDLResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from get database ddl, got %d body=%s", getDDLResp.StatusCode, string(providerContractBody(t, getDDLResp)))
	}
	getDDLBody := providerContractJSONMap(t, getDDLResp)
	statements, _ := getDDLBody["statements"].([]any)
	if len(statements) == 0 {
		t.Fatalf("expected statements array, got %#v", getDDLBody["statements"])
	}
}

func TestGCPSpannerAdminDatabaseRouter_ParseIAMActionPath_Get(t *testing.T) {
	t.Parallel()

	resource, action, ok := parseGCPSpannerAdminDatabaseIAMActionPath("/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db:getIamPolicy")
	if !ok {
		t.Fatalf("expected parseGCPSpannerAdminDatabaseIAMActionPath to match getIamPolicy path")
	}
	if action != "getIamPolicy" {
		t.Fatalf("expected getIamPolicy action, got %q", action)
	}
	if resource != "projects/stackyard/instances/stackyard-instance/databases/stackyard-db" {
		t.Fatalf("unexpected resource %q", resource)
	}
}

func TestGCPSpannerAdminDatabaseRouter_DirectGetIAMPolicyRoute(t *testing.T) {
	t.Parallel()

	s := New(Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	req := httptest.NewRequest(http.MethodGet, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db:getIamPolicy", nil)
	req.Header.Set("X-Stackyard-GCP-Service", "spanner-admin-database")
	rr := httptest.NewRecorder()

	if !s.handleGCPSpannerAdminDatabaseRouter(rr, req) {
		t.Fatalf("expected spanner admin database router to handle getIamPolicy path")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from direct router call, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGCPSpannerAdminDatabaseRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/spanner_admin_database?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner admin database contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "spanner_admin_database" {
		t.Fatalf("expected service=spanner_admin_database, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSpannerAdminDatabaseContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPSpannerAdminDatabaseSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner-admin-database",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner admin database router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
