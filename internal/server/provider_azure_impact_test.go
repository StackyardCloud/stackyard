package server

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

const azureImpactTestAPIVersion = "?api-version=2026-01-01-preview"

func TestAzureImpactWorkloadImpacts(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-impact")
	impacts := sub + "/providers/Microsoft.Impact/workloadImpacts"
	vm := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-impact/providers/Microsoft.Compute/virtualMachines/vm1"

	resp := do(http.MethodPut, impacts+"/impact-001"+azureImpactTestAPIVersion, `{"properties":{
		"impactedResourceId":"`+vm+`","startDateTime":"2026-06-15T05:59:46.6517821Z","endDateTime":null,
		"impactCategory":"resource.connectivity","impactDescription":"connection failure",
		"connectivity":{"protocol":"TCP","port":1443,"source":{"azureResourceId":"`+vm+`"}},
		"workload":{"context":"webapp/scenario1","toolset":"SDK"},"clientIncidentDetails":{"clientIncidentId":"AA123","clientIncidentSource":"Jira"},
		"performance":[{"metricName":"CPU","actual":90,"expected":60,"unit":"Percent","expectedValueRange":{"min":1,"max":60}}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	impact := providerContractJSONMap(t, resp)
	props := impact["properties"].(map[string]any)
	if props["impactCategory"] != "Resource.Connectivity" || props["provisioningState"] != "Succeeded" || props["reportedTimeUtc"] == nil {
		t.Fatalf("unexpected impact: %#v", props)
	}
	if !regexp.MustCompile(`^[0-9a-f-]{36}$`).MatchString(azureARMString(props["impactUniqueId"])) {
		t.Fatalf("impactUniqueId = %#v", props["impactUniqueId"])
	}
	if list, _ := props["insightsByCategory"].([]any); list == nil || len(list) != 0 {
		t.Fatalf("a non-HPC impact starts without insights: %#v", props["insightsByCategory"])
	}
	// Reports can't be changed.
	azureSentinelExpectRejected(t, do(http.MethodPut, impacts+"/impact-001"+azureImpactTestAPIVersion, `{"properties":{
		"impactedResourceId":"`+vm+`","startDateTime":"2026-06-15T05:59:46Z","impactCategory":"Resource.Availability"}}`), "already exists")
	azureResourcesExpectStatus(t, do(http.MethodPatch, impacts+"/impact-001"+azureImpactTestAPIVersion, `{"properties":{}}`), http.StatusMethodNotAllowed)

	// Insights, as Microsoft posts them, are summarized by category on the
	// impact: a category's status is its latest insight's.
	insights := impacts + "/impact-001/insights"
	resp = do(http.MethodPut, insights+"/insight1"+azureImpactTestAPIVersion, `{"properties":{"category":"repair","status":"InProgress",
		"eventTime":"2026-06-15T06:00:00Z","content":{"title":"Correlated outage","description":"An outage affected the resource."}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	ip := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if imp := ip["impact"].(map[string]any); !strings.EqualFold(azureARMString(imp["impactId"]), azureARMString(impact["id"])) || imp["impactedResourceId"] != vm || imp["startTime"] != "2026-06-15T05:59:46.6517821Z" || ip["insightUniqueId"] == nil {
		t.Fatalf("insight should be filled from its impact: %#v", ip)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, insights+"/insight2"+azureImpactTestAPIVersion, `{"properties":{"category":"repair","status":"resolved"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, insights+"/insight3"+azureImpactTestAPIVersion, `{"properties":{"category":"outage","status":"active"}}`), http.StatusCreated)
	byCategory := providerContractJSONMap(t, do(http.MethodGet, impacts+"/impact-001"+azureImpactTestAPIVersion, ""))["properties"].(map[string]any)["insightsByCategory"].([]any)
	if len(byCategory) != 2 {
		t.Fatalf("insightsByCategory = %#v", byCategory)
	}
	outage, repair := byCategory[0].(map[string]any), byCategory[1].(map[string]any)
	if outage["category"] != "outage" || repair["category"] != "repair" || repair["status"] != "resolved" || len(repair["insights"].([]any)) != 2 {
		t.Fatalf("insightsByCategory = %#v", byCategory)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, insights+"/insight4"+azureImpactTestAPIVersion, `{"properties":{"category":"repair",
		"impact":{"impactId":"`+strings.TrimPrefix(sub, "/azure")+`/providers/Microsoft.Impact/workloadImpacts/other"}}}`), "must be the insight's workload impact")
	azureResourcesExpectStatus(t, do(http.MethodDelete, insights+"/insight3"+azureImpactTestAPIVersion, ""), http.StatusOK)
	byCategory = providerContractJSONMap(t, do(http.MethodGet, impacts+"/impact-001"+azureImpactTestAPIVersion, ""))["properties"].(map[string]any)["insightsByCategory"].([]any)
	if len(byCategory) != 1 {
		t.Fatalf("deleting an insight should update the summary: %#v", byCategory)
	}
	if list := providerContractJSONMap(t, do(http.MethodGet, insights+azureImpactTestAPIVersion, ""))["value"].([]any); len(list) != 2 {
		t.Fatalf("insights = %d", len(list))
	}

	// Deleting the impact deletes its insights.
	azureResourcesExpectStatus(t, do(http.MethodDelete, impacts+"/impact-001"+azureImpactTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, insights+"/insight1"+azureImpactTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodDelete, impacts+"/impact-001"+azureImpactTestAPIVersion, ""), http.StatusNoContent)
}

func TestAzureImpactGuestHealth(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-impact")
	vm := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-impact/providers/Microsoft.Compute/virtualMachines/hpc1"
	token := providerContractJSONMap(t, do(http.MethodPost, sub+"/providers/Microsoft.Impact/getUploadToken"+azureImpactTestAPIVersion, ""))
	uploadURL := azureARMString(token["uploadUrl"])
	logURL, _, _ := strings.Cut(uploadURL, "?")
	if logURL == uploadURL || !azureImpactLogURL.MatchString(logURL) {
		t.Fatalf("uploadUrl = %q", uploadURL)
	}
	body := func(category, extra string) string {
		return `{"properties":{"impactedResourceId":"` + vm + `","startDateTime":"2026-05-13T01:06:21.3886467Z",
			"impactCategory":"` + category + `","impactDescription":"Missing GPU device","additionalProperties":{"PhysicalHostName":"GGBB90904476"` + extra + `}}}`
	}
	resp := do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/ghr1"+azureImpactTestAPIVersion, body("Resource.Hpc.Unhealthy.HpcMissingGpu", `,"LogUrl":"`+logURL+`"`))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	groups := providerContractJSONMap(t, resp)["properties"].(map[string]any)["insightsByCategory"].([]any)
	if len(groups) != 1 || groups[0].(map[string]any)["category"] != "MitigationAction" || groups[0].(map[string]any)["status"] != "Acknowledged" {
		t.Fatalf("an HPC report should be acknowledged: %#v", groups)
	}
	ack := providerContractJSONMap(t, do(http.MethodGet, "/azure"+azureARMString(groups[0].(map[string]any)["insights"].([]any)[0].(map[string]any)["id"])+azureImpactTestAPIVersion, ""))
	if details := ack["properties"].(map[string]any)["additionalDetails"].(map[string]any); details["statusCode"] != "AcknowledgedUnhealthy" || details["terminalInsight"] != false {
		t.Fatalf("acknowledgement = %#v", ack)
	}
	reset := providerContractJSONMap(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/ghr2"+azureImpactTestAPIVersion, body("Resource.Hpc.Reset", "")))
	if id := reset["properties"].(map[string]any)["insightsByCategory"].([]any)[0].(map[string]any)["insights"].([]any)[0].(map[string]any)["id"]; id == nil {
		t.Fatalf("reset = %#v", reset)
	}

	for _, tc := range []struct{ body, want string }{
		{body("Resource.Hpc.Reboot", `,"LogUrl":"`+uploadURL+`"`), "INVALID_FORMAT"},
		{body("Resource.Hpc.Reboot", `,"LogUrl":"https://example.com/log.gz"`), "INVALID_FORMAT"},
		{strings.Replace(body("Resource.Hpc.Reboot", ""), `"PhysicalHostName":"GGBB90904476"`, `"Manufacturer":"Nvidia"`, 1), "PhysicalHostName is required"},
		{strings.Replace(body("Resource.Hpc.Reboot", ""), `"impactDescription":"Missing GPU device",`, ``, 1), "impactDescription is required"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/bad-report"+azureImpactTestAPIVersion, tc.body), tc.want)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/resourceGroups/rg-impact/providers/Microsoft.Impact/getUploadToken"+azureImpactTestAPIVersion, ""), http.StatusBadRequest)
}

func TestAzureImpactValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-impact")
	impact := sub + "/providers/Microsoft.Impact/workloadImpacts/impact-100" + azureImpactTestAPIVersion
	vm := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-impact/providers/Microsoft.Compute/virtualMachines/vm1"
	body := func(extra string) string {
		return `{"properties":{"impactedResourceId":"` + vm + `","startDateTime":"2026-06-15T05:00:00Z","impactCategory":"Resource.Performance"` + extra + `}}`
	}
	for _, tc := range []struct{ body, want string }{
		{`{}`, "properties is required"},
		{`{"properties":{"impactedResourceId":"` + vm + `","impactCategory":"Resource.Performance"}}`, "startDateTime is required"},
		{strings.Replace(body(""), "2026-06-15T05:00:00Z", "yesterday", 1), "startDateTime must be an RFC 3339"},
		{body(`,"endDateTime":"2026-06-15T04:00:00Z"`), "endDateTime can't be before"},
		{strings.Replace(body(""), vm, "vm1", 1), "impactedResourceId must be an Azure resource ID"},
		{strings.Replace(body(""), vm, "/providers/Microsoft.Compute/virtualMachines/vm1", 1), "impactedResourceId must be an Azure resource ID"},
		{strings.Replace(body(""), "Resource.Performance", "Performance", 1), "impactCategory must be one of"},
		{strings.Replace(body(""), "Resource.Performance", "ARMOperation.Delete", 1), "armCorrelationIds is required"},
		{body(`,"severity":"Sev1"`), "properties.severity"},
		{body(`,"confidenceLevel":"Certain"`), "properties.confidenceLevel"},
		{body(`,"detectionType":"Manual"`), "properties.detectionType"},
		{body(`,"durationInSec":-1`), "durationInSec must be a number"},
		{body(`,"hitCount":1.5`), "hitCount must be a whole number"},
		{body(`,"ongoingImpact":"yes"`), "ongoingImpact must be a boolean"},
		{body(`,"armCorrelationIds":[1]`), "armCorrelationIds must be an array of strings"},
		{body(`,"workload":{"toolset":"Pulumi"}`), "properties.workload.toolset"},
		{body(`,"clientIncidentDetails":{"clientIncidentSource":"Email"}`), "clientIncidentSource"},
		{body(`,"connectivity":{"protocol":"ICMP"}`), "properties.connectivity.protocol"},
		{body(`,"connectivity":{"port":70000}`), "port must be a port number"},
		{body(`,"connectivity":{"target":{"azureResourceId":"vm2"}}`), "target.azureResourceId"},
		{body(`,"performance":[{"unit":"Furlongs"}]`), "performance[].unit"},
		{body(`,"performance":[{"actual":"high"}]`), "performance[].actual must be a number"},
		{body(`,"performance":[{"expectedValueRange":{"min":5,"max":1}}]`), "expectedValueRange"},
		{body(`,"performance":{}`), "performance must be an array"},
		{body(`,"workload":"web"`), "properties.workload must be an object"},
		{body(`,"impactDescription":5`), "impactDescription must be a string"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, impact, tc.body), tc.want)
	}
	// Learn's pattern ends in three alphanumerics.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/report-1"+azureImpactTestAPIVersion, body("")), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/a_b"+azureImpactTestAPIVersion, body("")), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodPut, impact, strings.Replace(body(`,"armCorrelationIds":["00000000-0000-0000-0000-000000000000"]`), "Resource.Performance", "ARMOperation.Delete", 1)), http.StatusCreated)

	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/parent"+azureImpactTestAPIVersion, body("")), http.StatusCreated)
	insight := sub + "/providers/Microsoft.Impact/workloadImpacts/parent/insights/insight1" + azureImpactTestAPIVersion
	for _, tc := range []struct{ body, want string }{
		{`{}`, "properties is required"},
		{`{"properties":{"status":"resolved"}}`, "properties.category is required"},
		{`{"properties":{"category":"repair","status":1}}`, "properties.status must be a string"},
		{`{"properties":{"category":"repair","eventTime":"soon"}}`, "eventTime must be an RFC 3339"},
		{`{"properties":{"category":"repair","content":"text"}}`, "properties.content must be an object"},
		{`{"properties":{"category":"repair","impact":{"startTime":"soon"}}}`, "properties.impact.startTime"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, insight, tc.body), tc.want)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/missing/insights/insight1"+azureImpactTestAPIVersion, `{"properties":{"category":"repair"}}`), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/workloadImpacts/parent/insights/i_1"+azureImpactTestAPIVersion, `{"properties":{"category":"repair"}}`), http.StatusBadRequest)
	// A PUT to an existing insight keeps its unique ID.
	first := providerContractJSONMap(t, do(http.MethodPut, insight, `{"properties":{"category":"repair"}}`))["properties"].(map[string]any)
	second := providerContractJSONMap(t, do(http.MethodPut, insight, `{"properties":{"category":"repair","status":"resolved"}}`))["properties"].(map[string]any)
	third := providerContractJSONMap(t, do(http.MethodPut, insight, `{"properties":{"category":"repair","insightUniqueId":"00000000-0000-0000-0000-000000000001"}}`))["properties"].(map[string]any)
	if first["insightUniqueId"] == nil || second["insightUniqueId"] != first["insightUniqueId"] || third["insightUniqueId"] != first["insightUniqueId"] {
		t.Fatalf("insightUniqueId changed: %#v then %#v", first, second)
	}
}

func TestAzureImpactCategories(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-impact")
	categories := sub + "/providers/Microsoft.Impact/impactCategories"
	all := providerContractJSONMap(t, do(http.MethodGet, categories+azureImpactTestAPIVersion, ""))["value"].([]any)
	if len(all) != len(azureImpactCategories) {
		t.Fatalf("categories = %d", len(all))
	}
	storage := providerContractJSONMap(t, do(http.MethodGet, categories+azureImpactTestAPIVersion+"&resourceType=Microsoft.Storage/storageAccounts", ""))["value"].([]any)
	vms := providerContractJSONMap(t, do(http.MethodGet, categories+azureImpactTestAPIVersion+"&resourceType=microsoft.compute/virtualmachines", ""))["value"].([]any)
	if len(storage) != 24 || len(vms) != len(azureImpactCategories) {
		t.Fatalf("HPC categories apply to virtual machines only: %d storage, %d VM", len(storage), len(vms))
	}
	availability := providerContractJSONMap(t, do(http.MethodGet, categories+azureImpactTestAPIVersion+"&categoryName=resource.availability", ""))["value"].([]any)
	if len(availability) != 9 {
		t.Fatalf("categoryName should keep the category and its subcategories: %d", len(availability))
	}

	restart := providerContractJSONMap(t, do(http.MethodGet, categories+"/Resource.Availability.Restart"+azureImpactTestAPIVersion, ""))["properties"].(map[string]any)
	parent := providerContractJSONMap(t, do(http.MethodGet, categories+"/Resource.Availability"+azureImpactTestAPIVersion, ""))["properties"].(map[string]any)
	if restart["parentCategoryId"] != parent["categoryId"] || restart["categoryId"] == parent["categoryId"] || restart["requiredImpactProperties"] != nil {
		t.Fatalf("restart = %#v, parent = %#v", restart, parent)
	}
	arm := providerContractJSONMap(t, do(http.MethodGet, categories+"/ARMOperation.Delete"+azureImpactTestAPIVersion, ""))["properties"].(map[string]any)
	if req := arm["requiredImpactProperties"].([]any); req[0].(map[string]any)["name"] != "armCorrelationIds" {
		t.Fatalf("ARM operation categories require correlation IDs: %#v", arm)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, categories+"/Resource.Nope"+azureImpactTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureImpactConnectors(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-impact")
	connector := sub + "/providers/Microsoft.Impact/connectors/AzureMonitorConnector" + azureImpactTestAPIVersion
	identity := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-impact/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id1"
	resp := do(http.MethodPut, connector, `{"properties":{"connectorType":"AzureMonitor"},"identity":{"type":"UserAssigned","userAssignedIdentities":{"`+identity+`":{}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["tenantId"] != azureDefaultTenantID || props["processingState"] != "Initialized" || props["provisioningState"] != "Succeeded" || props["connectorId"] == nil {
		t.Fatalf("unexpected connector: %#v", props)
	}
	patched := providerContractJSONMap(t, do(http.MethodPatch, connector, `{"identity":{"type":"None"}}`))
	if patched["properties"].(map[string]any)["connectorId"] != props["connectorId"] || patched["identity"].(map[string]any)["type"] != "None" {
		t.Fatalf("PATCH should keep the connector ID: %#v", patched)
	}
	put := providerContractJSONMap(t, do(http.MethodPut, connector, `{"properties":{"connectorType":"AzureMonitor"}}`))
	if put["properties"].(map[string]any)["connectorId"] != props["connectorId"] {
		t.Fatalf("PUT should keep the connector ID: %#v", put)
	}
	if list := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.Impact/connectors"+azureImpactTestAPIVersion, ""))["value"].([]any); len(list) != 1 {
		t.Fatalf("connectors = %d", len(list))
	}
	for _, tc := range []struct{ method, body, want string }{
		{http.MethodPut, `{}`, "properties is required"},
		{http.MethodPut, `{"properties":{}}`, "connectorType is required"},
		{http.MethodPut, `{"properties":{"connectorType":"Splunk"}}`, "properties.connectorType"},
		{http.MethodPut, `{"properties":{"connectorType":"AzureMonitor"},"identity":{"type":"SystemAssigned"}}`, "identity.type"},
		{http.MethodPut, `{"properties":{"connectorType":"AzureMonitor"},"identity":{"type":"UserAssigned"}}`, "userAssignedIdentities is required"},
		{http.MethodPut, `{"properties":{"connectorType":"AzureMonitor"},"identity":{"type":"UserAssigned","userAssignedIdentities":{"id1":{}}}}`, "user-assigned identity IDs"},
		{http.MethodPatch, `{"properties":{"tenantId":"other"}}`, "properties.tenantId can't be updated"},
		{http.MethodPatch, `{"properties":{"connectorType":"Splunk"}}`, "properties.connectorType"},
		{http.MethodPatch, `{"identity":"none"}`, "identity must be an object"},
	} {
		azureSentinelExpectRejected(t, do(tc.method, connector, tc.body), tc.want)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/providers/Microsoft.Impact/connectors/ab"+azureImpactTestAPIVersion, `{"properties":{"connectorType":"AzureMonitor"}}`), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodDelete, connector, ""), http.StatusOK)
}
