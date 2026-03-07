package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSecurityCenterRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterContractServer(t)

	versions := []struct {
		name          string
		apiVersion    string
		serviceHeader string
	}{
		{name: "V1", apiVersion: "v1", serviceHeader: "securitycenter"},
		{name: "V2", apiVersion: "v2", serviceHeader: "securitycenter-apiv2"},
	}

	for _, version := range versions {
		version := version
		t.Run(version.name, func(t *testing.T) {
			t.Parallel()

			root := "/gcp/" + version.apiVersion
			source := root + "/organizations/123456/sources/source-1"
			finding := source + "/findings/finding-1"
			muteConfig := root + "/organizations/123456/muteConfigs/mute-config-1"
			notificationConfig := root + "/organizations/123456/notificationConfigs/notify-1"
			bigQueryExport := root + "/organizations/123456/bigQueryExports/export-1"
			orgSettings := root + "/organizations/123456/organizationSettings"
			operation := root + "/organizations/123456/operations/op-1"
			simulation := root + "/organizations/123456/simulations/latest"
			valuedResource := simulation + "/valuedResources/resource-1"
			resourceValueConfig := root + "/organizations/123456/resourceValueConfigs/config-1"

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, root+"/organizations/123456/sources?pageSize=1", nil, `"sources":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, source, nil, "sources/source-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, root+"/organizations/123456/sources", []byte(`{"source":{"displayName":"Stackyard Source","description":"staged"}}`), "sources/source-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPatch, source+"?updateMask=display_name", []byte(`{"source":{"name":"organizations/123456/sources/source-1","displayName":"Updated Source"}}`), "Updated Source")

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, source+"/findings?pageSize=1", nil, `"listFindingsResults":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, source+"/findings:group?groupBy=category&pageSize=1", nil, `"groupByResults":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, source+"/findings?findingId=finding-2", []byte(`{"finding":{"name":"organizations/123456/sources/source-1/findings/finding-2","category":"OPEN_FIREWALL"}}`), "findings/finding-2")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPatch, finding+"?updateMask=severity", []byte(`{"finding":{"name":"organizations/123456/sources/source-1/findings/finding-1","severity":"HIGH"}}`), `"severity":"HIGH"`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, finding+":setState", []byte(`{"name":"organizations/123456/sources/source-1/findings/finding-1","state":"inactive"}`), `"state":"INACTIVE"`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, finding+":setMute", []byte(`{"name":"organizations/123456/sources/source-1/findings/finding-1","mute":"muted"}`), `"mute":"MUTED"`)

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, root+"/organizations/123456/muteConfigs?pageSize=1", nil, `"muteConfigs":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, muteConfig, nil, "muteConfigs/mute-config-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, root+"/organizations/123456/muteConfigs?muteConfigId=mute-config-3", []byte(`{"muteConfig":{"filter":"severity=\"HIGH\""}}`), "muteConfigs/mute-config-3")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPatch, muteConfig+"?updateMask=filter", []byte(`{"muteConfig":{"name":"organizations/123456/muteConfigs/mute-config-1","filter":"severity=\"CRITICAL\""}}`), "muteConfigs/mute-config-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodDelete, muteConfig, nil, "{}")

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, root+"/organizations/123456/notificationConfigs?pageSize=1", nil, `"notificationConfigs":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, notificationConfig, nil, "notificationConfigs/notify-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, root+"/organizations/123456/notificationConfigs?configId=notify-3", []byte(`{"notificationConfig":{"pubsubTopic":"projects/stackyard/topics/alerts"}}`), "notificationConfigs/notify-3")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPatch, notificationConfig+"?updateMask=description", []byte(`{"notificationConfig":{"name":"organizations/123456/notificationConfigs/notify-1"}}`), "notificationConfigs/notify-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodDelete, notificationConfig, nil, "{}")

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, orgSettings, nil, "organizationSettings")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPatch, orgSettings+"?updateMask=enable_asset_discovery", []byte(`{"organizationSettings":{"name":"organizations/123456/organizationSettings"}}`), "organizationSettings")

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, root+"/organizations/123456/bigQueryExports?pageSize=1", nil, `"bigQueryExports":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, bigQueryExport, nil, "bigQueryExports/export-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, root+"/organizations/123456/bigQueryExports?bigQueryExportId=export-3", []byte(`{"bigQueryExport":{"dataset":"projects/stackyard/datasets/security"}}`), "bigQueryExports/export-3")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPatch, bigQueryExport+"?updateMask=filter", []byte(`{"bigQueryExport":{"name":"organizations/123456/bigQueryExports/export-1"}}`), "bigQueryExports/export-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodDelete, bigQueryExport, nil, "{}")

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, source+"/findings:bulkMute", []byte(`{"filter":"severity=\"HIGH\""}`), "operations/")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, source+":getIamPolicy", []byte(`{}`), `"bindings":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, source+":setIamPolicy", []byte(`{"policy":{"version":1,"bindings":[{"role":"roles/securitycenter.findingsEditor","members":["user:analyst@example.invalid"]}]}}`), `"bindings":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, source+":testIamPermissions", []byte(`{"permissions":["securitycenter.sources.get"]}`), `"permissions":[`)

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, root+"/organizations/123456/operations?pageSize=1", nil, `"operations":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, operation, nil, "operations/op-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, operation+":cancel", []byte(`{}`), "{}")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodDelete, operation, nil, "{}")

			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, simulation, nil, `"state":"SUCCEEDED"`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, valuedResource, nil, "valuedResources/resource-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, simulation+"/attackPaths?pageSize=1", nil, `"attackPaths":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, root+"/organizations/123456/resourceValueConfigs?pageSize=1", nil, `"resourceValueConfigs":[`)
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodGet, resourceValueConfig, nil, "resourceValueConfigs/config-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, root+"/organizations/123456/resourceValueConfigs", []byte(`{"resourceValueConfig":{"resourceValue":"HIGH"}}`), "resourceValueConfigs/config-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPatch, resourceValueConfig+"?updateMask=resource_value", []byte(`{"resourceValueConfig":{"name":"organizations/123456/resourceValueConfigs/config-1","resourceValue":"MEDIUM"}}`), "resourceValueConfigs/config-1")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodDelete, resourceValueConfig, nil, "{}")
			assertGCPSecurityCenterSuccess(t, ts, version.serviceHeader, http.MethodPost, root+"/organizations/123456/resourceValueConfigs:batchCreate", []byte(`{"requests":[{"resourceValueConfig":{"name":"organizations/123456/resourceValueConfigs/config-3","resourceValue":"HIGH"}}]}`), `"resourceValueConfigs":[`)
		})
	}
}

func TestGCPSecurityCenterRouter_InvalidRequests(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterContractServer(t)

	versions := []struct {
		name          string
		apiVersion    string
		serviceHeader string
	}{
		{name: "V1", apiVersion: "v1", serviceHeader: "securitycenter"},
		{name: "V2", apiVersion: "v2", serviceHeader: "securitycenter-apiv2"},
	}

	for _, version := range versions {
		version := version
		t.Run(version.name, func(t *testing.T) {
			t.Parallel()

			root := "/gcp/" + version.apiVersion
			cases := []struct {
				name   string
				method string
				path   string
				body   []byte
				error  string
			}{
				{
					name:   "ListSourcesInvalidPageSize",
					method: http.MethodGet,
					path:   root + "/organizations/123456/sources?pageSize=bad",
					error:  `"error":"InvalidArgument"`,
				},
				{
					name:   "CreateSourceRequiresDisplayName",
					method: http.MethodPost,
					path:   root + "/organizations/123456/sources",
					body:   []byte(`{"source":{"description":"missing display name"}}`),
					error:  `"error":"InvalidArgument"`,
				},
				{
					name:   "UpdateSourceNameMustMatchPath",
					method: http.MethodPatch,
					path:   root + "/organizations/123456/sources/source-1?updateMask=display_name",
					body:   []byte(`{"source":{"name":"organizations/123456/sources/source-2","displayName":"bad"}}`),
					error:  `"error":"InvalidArgument"`,
				},
				{
					name:   "GroupFindingsRequiresGroupBy",
					method: http.MethodGet,
					path:   root + "/organizations/123456/sources/source-1/findings:group",
					error:  `"error":"InvalidArgument"`,
				},
				{
					name:   "SetMuteAlreadyMuted",
					method: http.MethodPost,
					path:   root + "/organizations/123456/sources/source-1/findings/already-muted:setMute",
					body:   []byte(`{"name":"organizations/123456/sources/source-1/findings/already-muted","mute":"MUTED"}`),
					error:  `"error":"FailedPrecondition"`,
				},
				{
					name:   "CreateResourceValueConfigRequiresResourceValue",
					method: http.MethodPost,
					path:   root + "/organizations/123456/resourceValueConfigs",
					body:   []byte(`{"resourceValueConfig":{"tagValues":["env:prod"]}}`),
					error:  `"error":"InvalidArgument"`,
				},
			}

			for _, tc := range cases {
				tc := tc
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					headers := map[string]string{
						"X-Stackyard-GCP-Service": version.serviceHeader,
					}
					if tc.body != nil {
						headers["Content-Type"] = "application/json"
					}
					resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, headers)
					if resp.StatusCode != http.StatusBadRequest {
						t.Fatalf("expected 400 from gcp securitycenter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
					}
					if body := string(providerContractBody(t, resp)); !strings.Contains(body, tc.error) {
						t.Fatalf("unexpected response body: %s", body)
					}
				})
			}
		})
	}
}

func TestGCPSecurityCenterRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterContractServer(t)

	for _, apiVersion := range []string{"v1", "v2"} {
		apiVersion := apiVersion
		t.Run(strings.ToUpper(apiVersion), func(t *testing.T) {
			t.Parallel()
			resp := providerContractRequest(
				t,
				ts,
				http.MethodGet,
				"/gcp/"+apiVersion+"/projects/stackyard/locations/us-central1/securitycenter?stackyard_contract_probe=1&typedSuccess=1",
				nil,
				nil,
			)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 from gcp securitycenter contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
			}

			body := providerContractJSONMap(t, resp)
			if got, _ := body["service"].(string); got != "securitycenter" {
				t.Fatalf("expected service=securitycenter, got %#v", body["service"])
			}
			if got, _ := body["provider"].(string); got != providerGCP {
				t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
			}
			if _, ok := body["name"].(string); !ok {
				t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
			}
		})
	}
}

func TestGCPSecurityCenterRouter_OutputShapesV2(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "securitycenter-apiv2",
	}

	sourceResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/sources/source-1", nil, headers)
	if sourceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 source response, got %d body=%s", sourceResp.StatusCode, string(providerContractBody(t, sourceResp)))
	}
	source := providerContractJSONMap(t, sourceResp)
	for _, key := range []string{"name", "displayName", "description", "canonicalName"} {
		if _, ok := source[key]; !ok {
			t.Fatalf("expected source.%s in response payload", key)
		}
	}

	findingsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/sources/source-1/findings?pageSize=1", nil, headers)
	if findingsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 findings response, got %d body=%s", findingsResp.StatusCode, string(providerContractBody(t, findingsResp)))
	}
	findingsBody := providerContractJSONMap(t, findingsResp)
	findings, ok := findingsBody["listFindingsResults"].([]any)
	if !ok || len(findings) == 0 {
		t.Fatalf("expected listFindingsResults in payload, got %#v", findingsBody["listFindingsResults"])
	}
	firstFindingResult, _ := findings[0].(map[string]any)
	finding, _ := firstFindingResult["finding"].(map[string]any)
	for _, key := range []string{"name", "state", "severity", "category", "eventTime", "createTime", "securityMarks"} {
		if _, ok := finding[key]; !ok {
			t.Fatalf("expected finding.%s in response payload", key)
		}
	}

	muteResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/muteConfigs/mute-config-1", nil, headers)
	if muteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 mute config response, got %d body=%s", muteResp.StatusCode, string(providerContractBody(t, muteResp)))
	}
	muteConfig := providerContractJSONMap(t, muteResp)
	for _, key := range []string{"name", "displayName", "filter", "createTime", "updateTime"} {
		if _, ok := muteConfig[key]; !ok {
			t.Fatalf("expected muteConfig.%s in response payload", key)
		}
	}

	notificationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/notificationConfigs/notify-1", nil, headers)
	if notificationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 notification config response, got %d body=%s", notificationResp.StatusCode, string(providerContractBody(t, notificationResp)))
	}
	notificationConfig := providerContractJSONMap(t, notificationResp)
	for _, key := range []string{"name", "pubsubTopic", "serviceAccount"} {
		if _, ok := notificationConfig[key]; !ok {
			t.Fatalf("expected notificationConfig.%s in response payload", key)
		}
	}

	exportResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/bigQueryExports/export-1", nil, headers)
	if exportResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 bigquery export response, got %d body=%s", exportResp.StatusCode, string(providerContractBody(t, exportResp)))
	}
	export := providerContractJSONMap(t, exportResp)
	for _, key := range []string{"name", "dataset", "createTime", "updateTime", "principal"} {
		if _, ok := export[key]; !ok {
			t.Fatalf("expected bigQueryExport.%s in response payload", key)
		}
	}

	simulationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/simulations/latest", nil, headers)
	if simulationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 simulation response, got %d body=%s", simulationResp.StatusCode, string(providerContractBody(t, simulationResp)))
	}
	simulation := providerContractJSONMap(t, simulationResp)
	for _, key := range []string{"name", "state", "createTime"} {
		if _, ok := simulation[key]; !ok {
			t.Fatalf("expected simulation.%s in response payload", key)
		}
	}

	valuedResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/simulations/latest/valuedResources/resource-1", nil, headers)
	if valuedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 valued resource response, got %d body=%s", valuedResp.StatusCode, string(providerContractBody(t, valuedResp)))
	}
	valued := providerContractJSONMap(t, valuedResp)
	for _, key := range []string{"name", "displayName", "resource"} {
		if _, ok := valued[key]; !ok {
			t.Fatalf("expected valuedResource.%s in response payload", key)
		}
	}

	attackPathsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/simulations/latest/attackPaths?pageSize=1", nil, headers)
	if attackPathsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 attack paths response, got %d body=%s", attackPathsResp.StatusCode, string(providerContractBody(t, attackPathsResp)))
	}
	attackPathsBody := providerContractJSONMap(t, attackPathsResp)
	attackPaths, ok := attackPathsBody["attackPaths"].([]any)
	if !ok || len(attackPaths) == 0 {
		t.Fatalf("expected attackPaths list in payload, got %#v", attackPathsBody["attackPaths"])
	}
	firstAttackPath, _ := attackPaths[0].(map[string]any)
	if _, ok := firstAttackPath["name"]; !ok {
		t.Fatalf("expected attackPath.name in payload")
	}

	valueCfgResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/resourceValueConfigs/config-1", nil, headers)
	if valueCfgResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 resource value config response, got %d body=%s", valueCfgResp.StatusCode, string(providerContractBody(t, valueCfgResp)))
	}
	valueCfg := providerContractJSONMap(t, valueCfgResp)
	for _, key := range []string{"name", "resourceValue", "tagValues"} {
		if _, ok := valueCfg[key]; !ok {
			t.Fatalf("expected resourceValueConfig.%s in response payload", key)
		}
	}

	opResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/organizations/123456/operations/op-1", nil, headers)
	if opResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 operation response, got %d body=%s", opResp.StatusCode, string(providerContractBody(t, opResp)))
	}
	op := providerContractJSONMap(t, opResp)
	if _, ok := op["name"]; !ok {
		t.Fatalf("expected operation.name in payload")
	}
	if _, ok := op["done"]; !ok {
		t.Fatalf("expected operation.done in payload")
	}
}

func newGCPSecurityCenterContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSecurityCenterSuccess(t *testing.T, ts *httptest.Server, service, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": service,
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securitycenter router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
