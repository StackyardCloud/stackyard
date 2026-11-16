package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureSentinelTestAPIVersion = "?api-version=2025-09-01"

// newAzureSentinelTestServer returns a server with an onboarded workspace
// and the workspace's Sentinel path prefix.
func newAzureSentinelTestServer(t *testing.T) (string, string, func(method, path, body string) *http.Response) {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-sentinel")
	workspace := sub + "/resourceGroups/rg-sentinel/providers/Microsoft.OperationalInsights/workspaces/soc-workspace"
	azureResourcesExpectStatus(t, do(http.MethodPut, workspace+"?api-version=2022-10-01", `{"location":"eastus","properties":{"sku":{"name":"PerGB2018"}}}`), http.StatusCreated)
	sentinel := workspace + "/providers/Microsoft.SecurityInsights"
	azureResourcesExpectStatus(t, do(http.MethodPut, sentinel+"/onboardingStates/default"+azureSentinelTestAPIVersion, `{"properties":{"customerManagedKey":false}}`), http.StatusCreated)
	return workspace, sentinel, do
}

const azureSentinelScheduledRule = `{"kind":"Scheduled","properties":{"displayName":"Failed sign-ins","enabled":true,"query":"SigninLogs | where ResultType != 0","queryFrequency":"PT1H","queryPeriod":"P1D","severity":"Medium","triggerOperator":"GreaterThan","triggerThreshold":10,"suppressionDuration":"PT5H","suppressionEnabled":false,"tactics":["CredentialAccess"],"entityMappings":[{"entityType":"Account","fieldMappings":[{"identifier":"FullName","columnName":"UserPrincipalName"}]}],"incidentConfiguration":{"createIncident":true,"groupingConfiguration":{"enabled":true,"reopenClosedIncident":false,"lookbackDuration":"PT5H","matchingMethod":"AllEntities"}}}}`

func TestAzureSentinelOnboardingAndScope(t *testing.T) {
	t.Parallel()
	workspace, sentinel, do := newAzureSentinelTestServer(t)
	other := strings.Replace(workspace, "soc-workspace", "fresh-workspace", 1)
	azureResourcesExpectStatus(t, do(http.MethodPut, other+"?api-version=2022-10-01", `{"location":"eastus"}`), http.StatusCreated)
	// Rules need an onboarded workspace.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, other+"/providers/Microsoft.SecurityInsights/alertRules/r1"+azureSentinelTestAPIVersion, azureSentinelScheduledRule), http.StatusBadRequest, "InvalidRequestContent")
	// Only "default" names an onboarding state, and only workspaces are scopes.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, other+"/providers/Microsoft.SecurityInsights/onboardingStates/other"+azureSentinelTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidResourceName")
	vault := strings.Replace(workspace, "Microsoft.OperationalInsights/workspaces", "Microsoft.KeyVault/vaults", 1)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+"/providers/Microsoft.SecurityInsights/onboardingStates/default"+azureSentinelTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidScope")
	missing := strings.Replace(workspace, "soc-workspace", "missing", 1)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, missing+"/providers/Microsoft.SecurityInsights/alertRules"+azureSentinelTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")

	azureResourcesExpectStatus(t, do(http.MethodPut, sentinel+"/alertRules/r1"+azureSentinelTestAPIVersion, azureSentinelScheduledRule), http.StatusCreated)
	// Deleting the workspace deletes its Sentinel resources.
	azureResourcesExpectStatus(t, do(http.MethodDelete, workspace+"?api-version=2022-10-01", ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, workspace+"?api-version=2022-10-01", `{"location":"eastus"}`), http.StatusCreated)
	if got := providerContractJSONMap(t, do(http.MethodGet, sentinel+"/onboardingStates"+azureSentinelTestAPIVersion, ""))["value"].([]any); len(got) != 0 {
		t.Fatalf("expected no onboarding state after the workspace was recreated, got %v", got)
	}
}

func TestAzureSentinelAlertRules(t *testing.T) {
	t.Parallel()
	_, sentinel, do := newAzureSentinelTestServer(t)
	rule := sentinel + "/alertRules/r1"
	azureResourcesExpectStatus(t, do(http.MethodPut, rule+azureSentinelTestAPIVersion, azureSentinelScheduledRule), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPatch, rule+azureSentinelTestAPIVersion, `{}`), http.StatusMethodNotAllowed)

	fusion := providerContractJSONMap(t, do(http.MethodPut, sentinel+"/alertRules/BuiltInFusion"+azureSentinelTestAPIVersion, `{"kind":"Fusion","properties":{"alertRuleTemplateName":"f71aba3d-28fb-450b-b192-4e76a83015c8","enabled":true,"displayName":"mine"}}`))
	if props := fusion["properties"].(map[string]any); props["displayName"] != "Advanced Multistage Attack Detection" || props["severity"] != "High" {
		t.Fatalf("Fusion rule should take its details from the template: %#v", props)
	}
	templates := providerContractJSONMap(t, do(http.MethodGet, sentinel+"/alertRuleTemplates"+azureSentinelTestAPIVersion, ""))["value"].([]any)
	if len(templates) != 1 || templates[0].(map[string]any)["properties"].(map[string]any)["alertRulesCreatedByTemplateCount"] != float64(1) {
		t.Fatalf("unexpected templates: %#v", templates)
	}

	// Actions are nested extension resources; the trigger URI is write-only.
	action := rule + "/actions/notify"
	body := providerContractJSONMap(t, do(http.MethodPut, action+azureSentinelTestAPIVersion, `{"properties":{"logicAppResourceId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Logic/workflows/notify","triggerUri":"https://prod-31.eastus.logic.azure.com:443/workflows/0b5c6c1d/triggers/manual/paths/invoke?sig=secret"}}`))
	props := body["properties"].(map[string]any)
	if props["workflowId"] != "0b5c6c1d" || props["triggerUri"] != nil || body["id"] != strings.TrimPrefix(action, "/azure") {
		t.Fatalf("unexpected action: %#v", body)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, rule+"/actions"+azureSentinelTestAPIVersion, ""))["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one action, got %d", len(got))
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, sentinel+"/alertRules/missing/actions"+azureSentinelTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sentinel+"/alertRules/missing/actions/a"+azureSentinelTestAPIVersion, `{"properties":{"logicAppResourceId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Logic/workflows/w","triggerUri":"https://x"}}`), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectStatus(t, do(http.MethodDelete, rule+azureSentinelTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, action+azureSentinelTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSentinelAlertRuleValidation(t *testing.T) {
	t.Parallel()
	_, sentinel, do := newAzureSentinelTestServer(t)
	rule := sentinel + "/alertRules/r1" + azureSentinelTestAPIVersion
	replace := func(old, new string) string { return strings.Replace(azureSentinelScheduledRule, old, new, 1) }
	// The action case needs its rule, created before the loop.
	azureResourcesExpectStatus(t, do(http.MethodPut, sentinel+"/alertRules/parent"+azureSentinelTestAPIVersion, azureSentinelScheduledRule), http.StatusCreated)
	for name, c := range map[string]struct{ body, want string }{
		"kind":             {`{"kind":"NRT","properties":{}}`, "kind must be"},
		"frequency floor":  {replace(`"queryFrequency":"PT1H"`, `"queryFrequency":"PT4M"`), "queryFrequency"},
		"period ceiling":   {replace(`"queryPeriod":"P1D"`, `"queryPeriod":"P15D"`), "queryPeriod"},
		"period short":     {replace(`"queryPeriod":"P1D"`, `"queryPeriod":"PT30M"`), "at least properties.queryFrequency"},
		"suppression":      {replace(`"suppressionDuration":"PT5H"`, `"suppressionDuration":"P2D"`), "suppressionDuration"},
		"duration syntax":  {replace(`"suppressionDuration":"PT5H"`, `"suppressionDuration":"5h"`), "suppressionDuration"},
		"severity":         {replace(`"severity":"Medium"`, `"severity":"Critical"`), "severity"},
		"operator":         {replace(`"triggerOperator":"GreaterThan"`, `"triggerOperator":"Gt"`), "triggerOperator"},
		"threshold":        {replace(`"triggerThreshold":10`, `"triggerThreshold":-1`), "triggerThreshold"},
		"tactic":           {replace(`["CredentialAccess"]`, `["Hacking"]`), "tactics"},
		"entity":           {replace(`"entityType":"Account"`, `"entityType":"Person"`), "entityType"},
		"too many":         {replace(`"entityMappings":[`, `"entityMappings":[`+strings.Repeat(`{"entityType":"IP","fieldMappings":[{"identifier":"Address","columnName":"IP"}]},`, 10)), "at most 10"},
		"grouping":         {replace(`"matchingMethod":"AllEntities"`, `"matchingMethod":"Some"`), "matchingMethod"},
		"create incident":  {replace(`"createIncident":true`, `"createIncident":"yes"`), "createIncident"},
		"query":            {replace(`"query":"SigninLogs | where ResultType != 0"`, `"query":""`), "query is required"},
		"fusion template":  {`{"kind":"Fusion","properties":{"alertRuleTemplateName":"00000000-0000-0000-0000-000000000001","enabled":true}}`, "Fusion template"},
		"msic product":     {`{"kind":"MicrosoftSecurityIncidentCreation","properties":{"displayName":"x","enabled":true,"productFilter":"Defender"}}`, "productFilter"},
		"msic severities":  {`{"kind":"MicrosoftSecurityIncidentCreation","properties":{"displayName":"x","enabled":true,"productFilter":"Azure Security Center","severitiesFilter":["Severe"]}}`, "severitiesFilter"},
		"action logic app": {`{"properties":{"logicAppResourceId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Web/sites/w","triggerUri":"https://x"}}`, "logicAppResourceId"},
	} {
		path := rule
		if name == "action logic app" {
			path = sentinel + "/alertRules/parent/actions/a" + azureSentinelTestAPIVersion
		}
		resp := do(http.MethodPut, path, c.body)
		t.Run(name, func(t *testing.T) {
			azureSentinelExpectRejected(t, resp, c.want)
		})
	}
	// The bounds themselves are accepted.
	azureResourcesExpectStatus(t, do(http.MethodPut, sentinel+"/alertRules/edge"+azureSentinelTestAPIVersion, strings.NewReplacer(`"queryFrequency":"PT1H"`, `"queryFrequency":"PT5M"`, `"queryPeriod":"P1D"`, `"queryPeriod":"P14D"`, `"suppressionDuration":"PT5H"`, `"suppressionDuration":"PT24H"`).Replace(azureSentinelScheduledRule)), http.StatusCreated)
}

func TestAzureSentinelAutomationRulesAndConnectors(t *testing.T) {
	t.Parallel()
	_, sentinel, do := newAzureSentinelTestServer(t)
	automation := sentinel + "/automationRules/close-noise" + azureSentinelTestAPIVersion
	valid := `{"properties":{"displayName":"Close noise","order":1,"triggeringLogic":{"isEnabled":true,"triggersOn":"Incidents","triggersWhen":"Created","conditions":[{"conditionType":"Property","conditionProperties":{"propertyName":"IncidentTitle","operator":"Contains","propertyValues":["test"]}}]},"actions":[{"order":1,"actionType":"ModifyProperties","actionConfiguration":{"status":"Closed","classification":"BenignPositive","classificationReason":"SuspiciousButExpected"}}]}}`
	body := providerContractJSONMap(t, do(http.MethodPut, automation, valid))
	if props := body["properties"].(map[string]any); props["createdTimeUtc"] == nil || props["createdBy"] == nil {
		t.Fatalf("automation rule should have server times and author: %#v", props)
	}
	for name, payload := range map[string]string{
		"order":       strings.Replace(valid, `"order":1,"triggering`, `"order":1001,"triggering`, 1),
		"triggers on": strings.Replace(valid, `"Incidents"`, `"Hunts"`, 1),
		"no actions":  `{"properties":{"displayName":"x","order":1,"triggeringLogic":{"isEnabled":true,"triggersOn":"Incidents","triggersWhen":"Created"},"actions":[]}}`,
		"alerts":      strings.Replace(valid, `"Incidents"`, `"Alerts"`, 1),
		"status":      strings.Replace(valid, `"status":"Closed"`, `"status":"Done"`, 1),
		"playbook":    strings.Replace(valid, `"actionType":"ModifyProperties","actionConfiguration":{"status":"Closed","classification":"BenignPositive","classificationReason":"SuspiciousButExpected"}`, `"actionType":"RunPlaybook","actionConfiguration":{}`, 1),
		"condition":   strings.Replace(valid, `"conditionType":"Property"`, `"conditionType":"Regex"`, 1),
	} {
		resp := do(http.MethodPut, automation, payload)
		t.Run("automation "+name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	connector := sentinel + "/dataConnectors/aad" + azureSentinelTestAPIVersion
	azureResourcesExpectStatus(t, do(http.MethodPut, connector, `{"kind":"AzureActiveDirectory","properties":{"tenantId":"72f988bf-86f1-41af-91ab-2d7cd011db47","dataTypes":{"alerts":{"state":"Enabled"}}}}`), http.StatusCreated)
	for name, payload := range map[string]string{
		"kind":       `{"kind":"Splunk","properties":{}}`,
		"tenant":     `{"kind":"AzureActiveDirectory","properties":{"tenantId":"contoso"}}`,
		"data types": `{"kind":"Office365","properties":{"tenantId":"72f988bf-86f1-41af-91ab-2d7cd011db47"}}`,
		"state":      `{"kind":"AmazonWebServicesCloudTrail","properties":{"dataTypes":{"logs":{"state":"On"}}}}`,
		"poller":     `{"kind":"RestApiPoller","properties":{"connectorDefinitionName":"x"}}`,
	} {
		resp := do(http.MethodPut, connector, payload)
		t.Run("connector "+name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	// Metadata is the one type with PATCH.
	metadata := sentinel + "/metadata/rule-metadata" + azureSentinelTestAPIVersion
	azureResourcesExpectStatus(t, do(http.MethodPut, metadata, `{"properties":{"kind":"AnalyticsRule","parentId":"/subscriptions/s/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/w/providers/Microsoft.SecurityInsights/alertRules/r1","version":"1.0.0"}}`), http.StatusCreated)
	if got := providerContractJSONMap(t, do(http.MethodPatch, metadata, `{"properties":{"version":"1.0.1"}}`))["properties"].(map[string]any)["version"]; got != "1.0.1" {
		t.Fatalf("metadata PATCH should update the version, got %v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, metadata, `{"properties":{"kind":"Spreadsheet","parentId":"/subscriptions/s"}}`), http.StatusBadRequest, "InvalidRequestContent")
}

func TestAzureSentinelIncidents(t *testing.T) {
	t.Parallel()
	_, sentinel, do := newAzureSentinelTestServer(t)
	first := sentinel + "/incidents/73e01a99-5cd7-4139-a149-9f2736ff2ab5"
	body := providerContractJSONMap(t, do(http.MethodPut, first+azureSentinelTestAPIVersion, `{"properties":{"title":"My incident","severity":"High","status":"New","labels":[{"labelName":"triage"}]}}`))
	props := body["properties"].(map[string]any)
	if props["incidentNumber"] != float64(1) || props["providerName"] != "Azure Sentinel" || props["labels"].([]any)[0].(map[string]any)["labelType"] != "User" {
		t.Fatalf("unexpected incident: %#v", props)
	}
	second := providerContractJSONMap(t, do(http.MethodPut, sentinel+"/incidents/second"+azureSentinelTestAPIVersion, `{"properties":{"title":"Another","severity":"Low","status":"Active"}}`))
	if second["properties"].(map[string]any)["incidentNumber"] != float64(2) {
		t.Fatalf("second incident should be number 2: %#v", second)
	}
	// Numbers are never reused, even after a delete.
	azureResourcesExpectStatus(t, do(http.MethodDelete, sentinel+"/incidents/second"+azureSentinelTestAPIVersion, ""), http.StatusOK)
	third := providerContractJSONMap(t, do(http.MethodPut, sentinel+"/incidents/third"+azureSentinelTestAPIVersion, `{"properties":{"title":"Third","severity":"Low","status":"New"}}`))
	if third["properties"].(map[string]any)["incidentNumber"] != float64(3) {
		t.Fatalf("a new incident after a delete should be number 3: %#v", third)
	}
	// Closing keeps the number and needs a classification.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, first+azureSentinelTestAPIVersion, `{"properties":{"title":"My incident","severity":"High","status":"Closed"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, first+azureSentinelTestAPIVersion, `{"properties":{"title":"My incident","severity":"High","status":"Closed","classification":"TruePositive","classificationReason":"InaccurateData"}}`), http.StatusBadRequest, "InvalidRequestContent")
	closed := providerContractJSONMap(t, do(http.MethodPut, first+azureSentinelTestAPIVersion, `{"properties":{"title":"My incident","severity":"High","status":"Closed","classification":"FalsePositive","classificationReason":"IncorrectAlertLogic"}}`))
	if closed["properties"].(map[string]any)["incidentNumber"] != float64(1) {
		t.Fatalf("an update should keep the incident number: %#v", closed)
	}

	comment := providerContractJSONMap(t, do(http.MethodPut, first+"/comments/c1"+azureSentinelTestAPIVersion, `{"properties":{"message":"Looks benign"}}`))
	if comment["properties"].(map[string]any)["author"] == nil {
		t.Fatalf("comment should have an author: %#v", comment)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, first+"/comments/c2"+azureSentinelTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, first+"/tasks/t1"+azureSentinelTestAPIVersion, `{"properties":{"title":"Reset password","status":"New"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, first+"/tasks/t2"+azureSentinelTestAPIVersion, `{"properties":{"title":"x","status":"Open"}}`), http.StatusBadRequest, "InvalidRequestContent")

	// Relations link bookmarks, which the incident's bookmarks action lists.
	bookmark := sentinel + "/bookmarks/b1"
	azureResourcesExpectStatus(t, do(http.MethodPut, bookmark+azureSentinelTestAPIVersion, `{"properties":{"displayName":"Suspicious sign-in","query":"SigninLogs | take 1","queryStartTime":"2026-01-01T00:00:00Z","queryEndTime":"2026-01-02T00:00:00Z"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sentinel+"/bookmarks/b2"+azureSentinelTestAPIVersion, `{"properties":{"displayName":"x","query":"q","queryStartTime":"2026-01-02T00:00:00Z","queryEndTime":"2026-01-01T00:00:00Z"}}`), http.StatusBadRequest, "InvalidRequestContent")
	relation := providerContractJSONMap(t, do(http.MethodPut, first+"/relations/rel-1"+azureSentinelTestAPIVersion, `{"properties":{"relatedResourceId":"`+strings.TrimPrefix(bookmark, "/azure")+`"}}`))
	if relation["properties"].(map[string]any)["relatedResourceName"] != "b1" {
		t.Fatalf("unexpected relation: %#v", relation)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, first+"/relations/rel-2"+azureSentinelTestAPIVersion, `{"properties":{"relatedResourceId":"`+strings.TrimPrefix(sentinel, "/azure")+`/bookmarks/missing"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, first+"/relations/r1"+azureSentinelTestAPIVersion, `{"properties":{"relatedResourceId":"`+strings.TrimPrefix(bookmark, "/azure")+`"}}`), http.StatusBadRequest, "InvalidResourceName")
	bookmarks := providerContractJSONMap(t, do(http.MethodPost, first+"/bookmarks"+azureSentinelTestAPIVersion, ""))["value"].([]any)
	if len(bookmarks) != 1 {
		t.Fatalf("expected the related bookmark, got %#v", bookmarks)
	}
	if alerts := providerContractJSONMap(t, do(http.MethodPost, first+"/alerts"+azureSentinelTestAPIVersion, ""))["value"].([]any); len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %#v", alerts)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, first+"/comments"+azureSentinelTestAPIVersion, ""))["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one comment, got %d", len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, first+azureSentinelTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, first+"/comments/c1"+azureSentinelTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSentinelWatchlists(t *testing.T) {
	t.Parallel()
	_, sentinel, do := newAzureSentinelTestServer(t)
	watchlist := sentinel + "/watchlists/highValueAsset"
	body := providerContractJSONMap(t, do(http.MethodPut, watchlist+azureSentinelTestAPIVersion, `{"properties":{"displayName":"High Value Assets","provider":"Microsoft","source":"watchlist.csv","sourceType":"Local","numberOfLinesToSkip":1,"rawContent":"This line will be skipped\nheader1,header2\nvalue1,value2\nvalue3,value4","itemsSearchKey":"header1","contentType":"text/csv"}}`))
	if props := body["properties"].(map[string]any); props["watchlistAlias"] != "highValueAsset" || props["watchlistId"] == "" {
		t.Fatalf("unexpected watchlist: %#v", props)
	}
	items := providerContractJSONMap(t, do(http.MethodGet, watchlist+"/watchlistItems"+azureSentinelTestAPIVersion, ""))["value"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["properties"].(map[string]any)["itemsKeyValue"].(map[string]any)["header2"] == nil {
		t.Fatalf("rawContent should become two items: %#v", items)
	}
	item := providerContractJSONMap(t, do(http.MethodPut, watchlist+"/watchlistItems/extra"+azureSentinelTestAPIVersion, `{"properties":{"itemsKeyValue":{"header1":"value5","header2":"value6"}}}`))
	if item["properties"].(map[string]any)["watchlistItemId"] != "extra" {
		t.Fatalf("unexpected item: %#v", item)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, watchlist+"/watchlistItems/empty"+azureSentinelTestAPIVersion, `{"properties":{"itemsKeyValue":{}}}`), http.StatusBadRequest, "InvalidRequestContent")

	base := `{"properties":{"displayName":"Assets","provider":"Microsoft","itemsSearchKey":"ip"`
	for name, payload := range map[string]string{
		"search key":   base + `,"contentType":"text/csv","rawContent":"host,owner\na,b"}}`,
		"content type": base + `,"contentType":"application/json","rawContent":"ip\n1.2.3.4"}}`,
		"ragged":       base + `,"contentType":"text/csv","rawContent":"ip,owner\n1.2.3.4"}}`,
		"too large":    base + `,"contentType":"text/csv","rawContent":"ip\n` + strings.Repeat("1", 3_800_001) + `"}}`,
		"skip":         base + `,"contentType":"text/csv","numberOfLinesToSkip":3,"rawContent":"ip\n1.2.3.4"}}`,
		"display name": `{"properties":{"displayName":"-bad-","provider":"Microsoft","itemsSearchKey":"ip"}}`,
		"provider":     `{"properties":{"displayName":"Assets","itemsSearchKey":"ip"}}`,
	} {
		resp := do(http.MethodPut, sentinel+"/watchlists/assets"+azureSentinelTestAPIVersion, payload)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sentinel+"/watchlists/ab"+azureSentinelTestAPIVersion, base+`}}`), http.StatusBadRequest, "InvalidResourceName")
	// Deleting a watchlist answers 204, final for its long-running delete.
	azureResourcesExpectStatus(t, do(http.MethodDelete, watchlist+azureSentinelTestAPIVersion, ""), http.StatusNoContent)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, watchlist+"/watchlistItems/extra"+azureSentinelTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

// azureSentinelExpectRejected checks for a 400 InvalidRequestContent whose
// message names want, so each case fails on the check it targets.
func azureSentinelExpectRejected(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	body := providerContractJSONMap(t, resp)
	errBody, _ := body["error"].(map[string]any)
	if resp.StatusCode != http.StatusBadRequest || errBody["code"] != "InvalidRequestContent" || !strings.Contains(azureARMString(errBody["message"]), want) {
		t.Fatalf("expected 400 InvalidRequestContent mentioning %q, got %d %#v", want, resp.StatusCode, body)
	}
}
