package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const dcdevTestQuery = "?api-version=2025-02-01"

// dcdevTestSetup creates a dev center with a catalog and environment type, and
// a project with a pool, schedule and environment type.
func dcdevTestSetup(t *testing.T) func(method, path, body string) *http.Response {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-dev")
	arm := sub + "/resourceGroups/rg-dev/providers/Microsoft.DevCenter/"
	q := "?api-version=2024-02-01"
	steps := []struct{ path, body string }{
		{"devcenters/dc1", `{"location":"eastus"}`},
		{"devcenters/dc1/catalogs/envs", `{"properties":{"gitHub":{"uri":"https://github.com/contoso/envs.git"}}}`},
		{"projects/web", `{"location":"eastus","properties":{"devCenterId":"` + strings.TrimPrefix(arm, "/azure") + `devcenters/dc1"}}`},
		{"projects/web/environmentTypes/dev", `{"properties":{"deploymentTargetId":"/subscriptions/00000000-0000-0000-0000-000000000000"}}`},
		{"projects/web/pools/pool1", `{"location":"eastus","properties":{"devBoxDefinitionName":"d","networkConnectionName":"n","licenseType":"Windows_Client","localAdministrator":"Enabled"}}`},
		{"projects/web/pools/pool1/schedules/default", `{"properties":{"type":"StopDevBox","frequency":"Daily","time":"19:00","timeZone":"UTC"}}`},
	}
	for _, step := range steps {
		azureResourcesExpectStatus(t, do(http.MethodPut, arm+step.path+q, step.body), http.StatusCreated)
	}
	return func(method, path, body string) *http.Response {
		return do(method, azureDevCenterDeveloper20250201Prefix+path, body)
	}
}

func dcdevTestValues(t *testing.T, resp *http.Response) []any {
	t.Helper()
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	values, _ := providerContractJSONMap(t, resp)["value"].([]any)
	return values
}

func TestAzureDevCenterDeveloper20250201DevBoxes(t *testing.T) {
	t.Parallel()
	do := dcdevTestSetup(t)
	if n := len(dcdevTestValues(t, do(http.MethodGet, "/projects"+dcdevTestQuery, ""))); n != 1 {
		t.Fatalf("listed %d projects", n)
	}
	if n := len(dcdevTestValues(t, do(http.MethodGet, "/projects/web/pools/pool1/schedules"+dcdevTestQuery, ""))); n != 1 {
		t.Fatalf("listed %d schedules", n)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, "/projects/web/users/me/abilities"+dcdevTestQuery, ""), http.StatusOK)

	box := "/projects/web/users/me/devboxes/box1"
	resp := do(http.MethodPut, box+dcdevTestQuery, `{"poolName":"pool1"}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	opURL := resp.Header.Get("Operation-Location")
	if !strings.Contains(opURL, "/projects/web/operationstatuses/") {
		t.Fatalf("unexpected Operation-Location %q", opURL)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, strings.TrimPrefix(opURL, azureDevCenterDeveloper20250201Prefix), ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodPost, box+":stop"+dcdevTestQuery+"&hibernate=true", ""), http.StatusAccepted)
	if state := providerContractJSONMap(t, do(http.MethodGet, box+dcdevTestQuery, ""))["powerState"]; state != "Hibernated" {
		t.Fatalf("powerState %v after hibernate", state)
	}
	actions := dcdevTestValues(t, do(http.MethodGet, box+"/actions"+dcdevTestQuery, ""))
	if len(actions) != 1 {
		t.Fatalf("expected the schedule's stop action, got %d", len(actions))
	}
	resp = do(http.MethodPost, box+"/actions/schedule-default:delay"+dcdevTestQuery+"&until=2030-01-01T00:00:00Z", "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if next := providerContractJSONMap(t, resp)["next"].(map[string]any)["scheduledTime"]; next != "2030-01-01T00:00:00Z" {
		t.Fatalf("delay set next to %v", next)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, box+":captureSnapshot"+dcdevTestQuery, ""), http.StatusAccepted)
	snaps := dcdevTestValues(t, do(http.MethodGet, box+"/snapshots"+dcdevTestQuery, ""))
	id := snaps[0].(map[string]any)["snapshotId"].(string)
	azureResourcesExpectStatus(t, do(http.MethodPost, box+":restoreSnapshot"+dcdevTestQuery+"&snapshotId="+id, ""), http.StatusAccepted)

	azureResourcesExpectStatus(t, do(http.MethodPut, box+"/customizationGroups/setup"+dcdevTestQuery, `{"tasks":[{"name":"winget","parameters":{"package":"git"}}]}`), http.StatusCreated)
	resp = do(http.MethodGet, box+"/customizationGroups/setup/logs/setup-a"+dcdevTestQuery, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if body, _ := io.ReadAll(resp.Body); !strings.Contains(string(body), "Completed") || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain") {
		t.Fatalf("unexpected task log %q", body)
	}
	if n := len(dcdevTestValues(t, do(http.MethodGet, "/devboxes"+dcdevTestQuery, ""))); n != 1 {
		t.Fatalf("listed %d dev boxes", n)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, box+dcdevTestQuery, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodGet, box+dcdevTestQuery, ""), http.StatusNotFound)
}

func TestAzureDevCenterDeveloper20250201Environments(t *testing.T) {
	t.Parallel()
	do := dcdevTestSetup(t)
	defs := dcdevTestValues(t, do(http.MethodGet, "/projects/web/catalogs/envs/environmentDefinitions"+dcdevTestQuery, ""))
	if len(defs) != 1 || defs[0].(map[string]any)["name"] != "Sandbox" {
		t.Fatalf("unexpected definitions %v", defs)
	}
	env := "/projects/web/users/me/environments/sandbox1"
	resp := do(http.MethodPut, env+dcdevTestQuery, `{"environmentType":"dev","catalogName":"envs","environmentDefinitionName":"Sandbox","expirationDate":"2030-01-01T00:00:00Z"}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	var created map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if !strings.HasSuffix(created["resourceGroupId"].(string), "/resourceGroups/web-sandbox1") {
		t.Fatalf("unexpected resourceGroupId %v", created["resourceGroupId"])
	}
	azureResourcesExpectStatus(t, do(http.MethodPatch, env+dcdevTestQuery, `{"expirationDate":"2031-01-01T00:00:00Z"}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, env+"/outputs"+dcdevTestQuery, ""), http.StatusOK)
	ops := dcdevTestValues(t, do(http.MethodGet, env+"/operations"+dcdevTestQuery, ""))
	opID := ops[0].(map[string]any)["operationId"].(string)
	azureResourcesExpectStatus(t, do(http.MethodGet, env+"/operations/"+opID+"/logs"+dcdevTestQuery, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, env+"/actions/expiration:skip"+dcdevTestQuery, ""), http.StatusNoContent)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/projects/web/users/me/environmentTypes/dev/abilities"+dcdevTestQuery, ""), http.StatusOK)
	if n := len(dcdevTestValues(t, do(http.MethodGet, "/projects/web/environments"+dcdevTestQuery, ""))); n != 1 {
		t.Fatalf("listed %d environments", n)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, env+dcdevTestQuery, ""), http.StatusAccepted)
}

func TestAzureDevCenterDeveloper20250201Validation(t *testing.T) {
	t.Parallel()
	do := dcdevTestSetup(t)
	cases := []struct {
		name, method, path, body string
		status                   int
	}{
		{"api version", http.MethodGet, "/projects?api-version=2020-01-01", "", http.StatusBadRequest},
		{"unknown project", http.MethodGet, "/projects/nope" + dcdevTestQuery, "", http.StatusNotFound},
		{"missing pool", http.MethodPut, "/projects/web/users/me/devboxes/box" + dcdevTestQuery, `{"poolName":"nope"}`, http.StatusBadRequest},
		{"bad name", http.MethodPut, "/projects/web/users/me/devboxes/x" + dcdevTestQuery, `{"poolName":"pool1"}`, http.StatusBadRequest},
		{"env fields", http.MethodPut, "/projects/web/users/me/environments/env" + dcdevTestQuery, `{"environmentType":"dev"}`, http.StatusBadRequest},
		{"env type", http.MethodPut, "/projects/web/users/me/environments/env" + dcdevTestQuery, `{"environmentType":"prod","catalogName":"envs","environmentDefinitionName":"Sandbox"}`, http.StatusBadRequest},
		{"unknown task", http.MethodPost, "/projects/web/customizationTasks:validateGroup" + dcdevTestQuery, `{"tasks":[]}`, http.StatusBadRequest},
		{"missing dev box", http.MethodGet, "/projects/web/users/me/devboxes/none" + dcdevTestQuery, "", http.StatusNotFound},
	}
	for _, tc := range cases {
		resp := do(tc.method, tc.path, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectStatus(t, resp, tc.status) })
	}
	resp := do(http.MethodPost, "/projects/web/customizationTasks:validateGroup"+dcdevTestQuery, `{"tasks":[{"name":"winget"},{"name":"bogus"}]}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["status"] != "Failed" {
		t.Fatal("expected validation to fail for an unknown task")
	}
}
