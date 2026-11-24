package server

import (
	"net/http"
	"strings"
	"testing"
)

const aduDPTestBase = azureDeviceUpdateDataPlane20221001Prefix + "deviceUpdate/inst1"
const aduDPTestQuery = "?api-version=2022-10-01"

func aduDPTestServer(t *testing.T) func(method, path, body string) *http.Response {
	t.Helper()
	_, do := newAzureDatabaseTestServer(t, "rg-adudp")
	return func(method, path, body string) *http.Response { return do(method, aduDPTestBase+path, body) }
}

func aduDPTestValues(t *testing.T, resp *http.Response) []any {
	t.Helper()
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	values, _ := providerContractJSONMap(t, resp)["value"].([]any)
	return values
}

func TestAzureDeviceUpdateDataPlane20221001Updates(t *testing.T) {
	t.Parallel()
	do := aduDPTestServer(t)
	resp := do(http.MethodPost, "/updates:import"+aduDPTestQuery, `[{"importManifest":{"url":"https://acct.blob.core.windows.net/u/Contoso.Sensor.1.0.2.importmanifest.json","sizeInBytes":100,"hashes":{"sha256":"x"}},
		"files":[{"filename":"firmware.bin","url":"https://acct.blob.core.windows.net/u/firmware.bin"}]}]`)
	azureResourcesExpectStatus(t, resp, http.StatusAccepted)
	op := resp.Header.Get("Operation-Location")
	if !strings.Contains(op, "/deviceUpdate/inst1/updates/operations/") {
		t.Fatalf("unexpected Operation-Location %q", op)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, strings.TrimPrefix(op, aduDPTestBase), ""), http.StatusOK)

	if got := aduDPTestValues(t, do(http.MethodGet, "/updates/providers"+aduDPTestQuery, "")); len(got) != 1 || got[0] != "Contoso" {
		t.Fatalf("unexpected providers %v", got)
	}
	if got := aduDPTestValues(t, do(http.MethodGet, "/updates/providers/Contoso/names/Sensor/versions"+aduDPTestQuery, "")); len(got) != 1 || got[0] != "1.0.2" {
		t.Fatalf("unexpected versions %v", got)
	}
	version := "/updates/providers/Contoso/names/Sensor/versions/1.0.2"
	resp = do(http.MethodGet, version+aduDPTestQuery, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if update := providerContractJSONMap(t, resp); update["_files"] != nil || update["isDeployable"] != true {
		t.Fatalf("unexpected update %v", update)
	}
	files := aduDPTestValues(t, do(http.MethodGet, version+"/files"+aduDPTestQuery, ""))
	azureResourcesExpectStatus(t, do(http.MethodGet, version+"/files/"+files[0].(string)+aduDPTestQuery, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, version+aduDPTestQuery, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodGet, version+aduDPTestQuery, ""), http.StatusNotFound)
}

func TestAzureDeviceUpdateDataPlane20221001Management(t *testing.T) {
	t.Parallel()
	do := aduDPTestServer(t)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/updates:import"+aduDPTestQuery, `[{"importManifest":{"url":"https://x/Contoso.Sensor.2.0.importmanifest.json"}}]`), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/management/devices:import"+aduDPTestQuery, `"All"`), http.StatusAccepted)
	if n := len(aduDPTestValues(t, do(http.MethodGet, "/management/devices"+aduDPTestQuery, ""))); n != 2 {
		t.Fatalf("listed %d devices", n)
	}
	if c := providerContractJSONMap(t, do(http.MethodGet, "/management/groups/default/updateCompliance"+aduDPTestQuery, "")); c["totalDeviceCount"] != float64(2) {
		t.Fatalf("unexpected compliance %v", c)
	}
	if n := len(aduDPTestValues(t, do(http.MethodGet, "/management/groups/default/bestUpdates"+aduDPTestQuery, ""))); n != 1 {
		t.Fatalf("listed %d best updates", n)
	}
	dep := "/management/groups/default/deployments/rollout"
	azureResourcesExpectStatus(t, do(http.MethodPut, dep+aduDPTestQuery, `{"deploymentId":"rollout","groupId":"default","startDateTime":"2026-01-01T00:00:00Z","update":{"updateId":{"provider":"Contoso","name":"Sensor","version":"2.0"}}}`), http.StatusOK)
	status := providerContractJSONMap(t, do(http.MethodGet, dep+"/status"+aduDPTestQuery, ""))
	if status["deploymentState"] != "Active" || len(status["subgroupStatus"].([]any)) != 1 {
		t.Fatalf("unexpected status %v", status)
	}
	sub := "/management/groups/default/deviceClassSubgroups/" + aduDPDeviceClass
	if n := len(aduDPTestValues(t, do(http.MethodGet, sub+"/deployments/rollout/deviceStates"+aduDPTestQuery, ""))); n != 2 {
		t.Fatalf("listed %d device states", n)
	}
	resp := do(http.MethodPost, dep+":cancel"+aduDPTestQuery, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["isCanceled"] != true {
		t.Fatal("expected a canceled deployment")
	}
	azureResourcesExpectStatus(t, do(http.MethodPatch, "/management/deviceClasses/"+aduDPDeviceClass+aduDPTestQuery, `{"friendlyName":"Sensors"}`), http.StatusOK)
	if n := len(aduDPTestValues(t, do(http.MethodGet, "/management/deviceClasses/"+aduDPDeviceClass+"/installableUpdates"+aduDPTestQuery, ""))); n != 1 {
		t.Fatalf("listed %d installable updates", n)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, "/management/deviceDiagnostics/logCollections/col1"+aduDPTestQuery, `{"deviceList":[{"deviceId":"device-1"}],"description":"logs"}`), http.StatusCreated)
	detail := providerContractJSONMap(t, do(http.MethodGet, "/management/deviceDiagnostics/logCollections/col1/detailedStatus"+aduDPTestQuery, ""))
	if len(detail["deviceStatus"].([]any)) != 1 {
		t.Fatalf("unexpected detailed status %v", detail)
	}
	if n := len(aduDPTestValues(t, do(http.MethodGet, "/management/deviceDiagnostics/deviceHealth"+aduDPTestQuery+"&filter=state%20eq%20'Healthy'", ""))); n != 2 {
		t.Fatalf("listed %d device health entries", n)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, dep+aduDPTestQuery, ""), http.StatusNoContent)
}

func TestAzureDeviceUpdateDataPlane20221001Validation(t *testing.T) {
	t.Parallel()
	do := aduDPTestServer(t)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/management/devices:import"+aduDPTestQuery, `"All"`), http.StatusAccepted)
	cases := []struct {
		name, method, path, body string
		status                   int
	}{
		{"api version", http.MethodGet, "/updates?api-version=2021-01-01", "", http.StatusBadRequest},
		{"empty import", http.MethodPost, "/updates:import" + aduDPTestQuery, `[]`, http.StatusBadRequest},
		{"manifest name", http.MethodPost, "/updates:import" + aduDPTestQuery, `[{"importManifest":{"url":"https://x/manifest.json"}}]`, http.StatusBadRequest},
		{"device import kind", http.MethodPost, "/management/devices:import" + aduDPTestQuery, `"Everything"`, http.StatusBadRequest},
		{"unknown update", http.MethodPut, "/management/groups/default/deployments/d" + aduDPTestQuery, `{"deploymentId":"d","groupId":"default","startDateTime":"2026-01-01T00:00:00Z","update":{"updateId":{"provider":"x","name":"y","version":"1"}}}`, http.StatusBadRequest},
		{"missing group", http.MethodGet, "/management/groups/nope" + aduDPTestQuery, "", http.StatusNotFound},
		{"friendly name", http.MethodPatch, "/management/deviceClasses/" + aduDPDeviceClass + aduDPTestQuery, `{}`, http.StatusBadRequest},
		{"device list", http.MethodPut, "/management/deviceDiagnostics/logCollections/c" + aduDPTestQuery, `{"deviceList":[]}`, http.StatusBadRequest},
		{"health filter", http.MethodGet, "/management/deviceDiagnostics/deviceHealth" + aduDPTestQuery, "", http.StatusBadRequest},
	}
	for _, tc := range cases {
		resp := do(tc.method, tc.path, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectStatus(t, resp, tc.status) })
	}
}
