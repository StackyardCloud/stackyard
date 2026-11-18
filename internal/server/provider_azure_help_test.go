package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const azureHelpTestAPIVersion = "?api-version=2024-03-01-preview"

func TestAzureHelpDiscovery(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-help")
	discover := func(filter string) *http.Response {
		return do(http.MethodGet, "/azure/providers/Microsoft.Help/discoverySolutions"+azureHelpTestAPIVersion+"&$filter="+url.QueryEscape(filter), "")
	}
	items := providerContractJSONMap(t, discover("ProblemClassificationId eq 'stackyard-vm-connectivity'"))["value"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one problem, got %#v", items)
	}
	props := items[0].(map[string]any)["properties"].(map[string]any)
	if props["problemTitle"] == nil || len(props["solutions"].([]any)) != 3 {
		t.Fatalf("expected the three VM solutions: %#v", props)
	}
	items = providerContractJSONMap(t, discover("ProblemClassificationId eq 'stackyard-vm-connectivity' and SolutionType eq 'Troubleshooters'"))["value"].([]any)
	if sols := items[0].(map[string]any)["properties"].(map[string]any)["solutions"].([]any); len(sols) != 1 || sols[0].(map[string]any)["solutionType"] != "Troubleshooters" {
		t.Fatalf("SolutionType should narrow the solutions: %#v", sols)
	}
	items = providerContractJSONMap(t, discover("ProblemClassificationId eq 'stackyard-vm-connectivity' or ProblemClassificationId eq 'stackyard-keyvault-recovery'"))["value"].([]any)
	if len(items) != 2 {
		t.Fatalf("or should match both problems: %#v", items)
	}
	items = providerContractJSONMap(t, discover("ProblemClassificationId eq 'stackyard-keyvault-recovery' and ResourceType eq 'Microsoft.Compute/virtualMachines'"))["value"].([]any)
	if len(items) != 0 {
		t.Fatalf("ResourceType should exclude the Key Vault problem: %#v", items)
	}
	azureResourcesExpectErrorCode(t, discover("SolutionType eq 'Solutions'"), http.StatusBadRequest, "InvalidFilter")
	azureResourcesExpectErrorCode(t, discover("ProblemClassificationId eq 'stackyard-vm-connectivity' or SolutionType eq 'Diagnostics'"), http.StatusBadRequest, "InvalidFilter")
	azureResourcesExpectErrorCode(t, discover("ProblemClassificationId ne 'x'"), http.StatusBadRequest, "InvalidFilter")
	azureResourcesExpectErrorCode(t, discover("ProblemClassificationId eq 'x' and Title eq 'y'"), http.StatusBadRequest, "InvalidFilter")

	// Natural language discovery matches keywords and resource types.
	nlp := func(path, body string) []any {
		return providerContractJSONMap(t, do(http.MethodPost, path+azureHelpTestAPIVersion, body))["value"].([]any)
	}
	if got := nlp("/azure/providers/Microsoft.Help/discoverSolutions", `{"issueSummary":"I can't RDP into my VM"}`); len(got) != 1 || got[0].(map[string]any)["name"] != "stackyard-vm-connectivity" {
		t.Fatalf("unexpected tenant discovery: %#v", got)
	}
	got := nlp(sub+"/providers/Microsoft.Help/discoverSolutions", `{"issueSummary":"something broke","resourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/kv"}`)
	if len(got) != 1 || !strings.HasPrefix(azureARMString(got[0].(map[string]any)["id"]), "/subscriptions/") {
		t.Fatalf("subscription discovery should match the resource type: %#v", got)
	}
	if got := nlp(sub+"/providers/Microsoft.Help/discoverSolutions", `{"issueSummary":"billing question"}`); len(got) != 0 {
		t.Fatalf("expected no matches: %#v", got)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, "/azure/providers/Microsoft.Help/discoverSolutions"+azureHelpTestAPIVersion, `{}`), "issueSummary is required")
	azureSentinelExpectRejected(t, do(http.MethodPost, sub+"/providers/Microsoft.Help/discoverSolutions"+azureHelpTestAPIVersion, `{"issueSummary":"x","serviceId":1}`), "serviceId must be a string")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/rg-help/providers/Microsoft.Help/discoverSolutions"+azureHelpTestAPIVersion, `{"issueSummary":"vm"}`), http.StatusBadRequest, "InvalidScope")

	// Self-help articles are read by solution ID.
	article := providerContractJSONMap(t, do(http.MethodGet, "/azure/providers/Microsoft.Help/selfHelp/stackyard-keyvault-recover-selfhelp"+azureHelpTestAPIVersion, ""))
	if ap := article["properties"].(map[string]any); ap["title"] == nil || len(ap["sections"].([]any)) != 1 {
		t.Fatalf("unexpected article: %#v", article)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.Help/selfHelp/stackyard-vm-connectivity-solution"+azureHelpTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/providers/Microsoft.Help/selfHelp/stackyard-keyvault-recover-selfhelp"+azureHelpTestAPIVersion, ""), http.StatusMethodNotAllowed)
}

func TestAzureHelpDiagnosticsAndSolutions(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-help")
	vm := sub + "/resourceGroups/rg-help/providers/Microsoft.Compute/virtualMachines/vm1"

	// A diagnostic needing a resource runs at a resource scope; at a
	// subscription it reports missing inputs.
	diag := providerContractJSONMap(t, do(http.MethodPut, vm+"/providers/Microsoft.Help/diagnostics/d1"+azureHelpTestAPIVersion,
		`{"properties":{"insights":[{"solutionId":"stackyard-vm-rdp-diagnostic"},{"solutionId":"nope"}]}}`))
	dp := diag["properties"].(map[string]any)
	results := dp["diagnostics"].([]any)
	if dp["provisioningState"] != "PartialComplete" || dp["acceptedAt"] == nil || results[0].(map[string]any)["status"] != "Succeeded" || results[1].(map[string]any)["status"] != "Failed" {
		t.Fatalf("unexpected diagnostics: %#v", dp)
	}
	atSub := providerContractJSONMap(t, do(http.MethodPut, sub+"/providers/Microsoft.Help/diagnostics/d2"+azureHelpTestAPIVersion, `{"properties":{"insights":[{"solutionId":"stackyard-vm-rdp-diagnostic"}]}}`))
	if r := atSub["properties"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any); r["status"] != "MissingInputs" || atSub["properties"].(map[string]any)["provisioningState"] != "Failed" {
		t.Fatalf("expected missing inputs at subscription scope: %#v", atSub)
	}
	withURI := providerContractJSONMap(t, do(http.MethodPut, sub+"/providers/Microsoft.Help/diagnostics/d3"+azureHelpTestAPIVersion, `{"properties":{"globalParameters":{"ResourceUri":"x"},"insights":[{"solutionId":"stackyard-vm-rdp-diagnostic"}]}}`))
	if withURI["properties"].(map[string]any)["provisioningState"] != "Succeeded" {
		t.Fatalf("a ResourceUri parameter should satisfy the diagnostic: %#v", withURI)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, vm+"/providers/Microsoft.Help/diagnostics/d4"+azureHelpTestAPIVersion, `{"properties":{"insights":[]}}`), "properties.insights")
	azureSentinelExpectRejected(t, do(http.MethodPut, vm+"/providers/Microsoft.Help/diagnostics/d4"+azureHelpTestAPIVersion, `{"properties":{"insights":[{}]}}`), "need a solutionId")
	azureSentinelExpectRejected(t, do(http.MethodPut, vm+"/providers/Microsoft.Help/diagnostics/d4"+azureHelpTestAPIVersion, `{"properties":{"globalParameters":{"a":1},"insights":[{"solutionId":"x"}]}}`), "globalParameters must be an object of strings")
	azureResourcesExpectStatus(t, do(http.MethodDelete, vm+"/providers/Microsoft.Help/diagnostics/d1"+azureHelpTestAPIVersion, ""), http.StatusMethodNotAllowed)

	// Solutions run the diagnostics their content embeds and list their
	// problem's troubleshooters.
	solution := vm + "/providers/Microsoft.Help/solutions/s1"
	created := providerContractJSONMap(t, do(http.MethodPut, solution+azureHelpTestAPIVersion, `{"properties":{"triggerCriteria":[{"name":"ProblemClassificationId","value":"stackyard-vm-connectivity"}],"parameters":{"SearchText":"rdp"}}}`))
	sp := created["properties"].(map[string]any)
	maps := sp["replacementMaps"].(map[string]any)
	if sp["solutionId"] != "stackyard-vm-connectivity-solution" || len(maps["diagnostics"].([]any)) != 1 || len(maps["troubleshooters"].([]any)) != 1 || sp["provisioningState"] != "Succeeded" ||
		maps["troubleshooters"].([]any)[0].(map[string]any)["solutionId"] != "stackyard-vm-connectivity-troubleshooter" {
		t.Fatalf("unexpected solution: %#v", sp)
	}
	if key := maps["diagnostics"].([]any)[0].(map[string]any)["replacementKey"]; !strings.Contains(azureARMString(sp["content"]), azureARMString(key)) {
		t.Fatalf("the replacement key should appear in the content: %#v", sp)
	}
	patched := providerContractJSONMap(t, do(http.MethodPatch, solution+azureHelpTestAPIVersion, `{"properties":{"triggerCriteria":[{"name":"ReplacementKey","value":"<!--stackyard-vm-rdp-diagnostic-->"}]}}`))
	if patched["properties"].(map[string]any)["solutionId"] != "stackyard-vm-connectivity-solution" {
		t.Fatalf("PATCH should keep the solution: %#v", patched)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, solution+azureHelpTestAPIVersion, `{"properties":{"triggerCriteria":[{"name":"SolutionId","value":"x"}]}}`), "ReplacementKey")
	azureSentinelExpectRejected(t, do(http.MethodPatch, solution+azureHelpTestAPIVersion, `{"properties":{"title":"x"}}`), "title can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPut, solution+azureHelpTestAPIVersion, `{"properties":{"triggerCriteria":[{"name":"SolutionId","value":"stackyard-vm-rdp-diagnostic"}]}}`), "matches no solution")
	azureSentinelExpectRejected(t, do(http.MethodPut, solution+azureHelpTestAPIVersion, `{"properties":{"triggerCriteria":[]}}`), "one SolutionId or ProblemClassificationId")
	azureSentinelExpectRejected(t, do(http.MethodPut, solution+azureHelpTestAPIVersion, `{"properties":{}}`), "triggerCriteria must be a list")
	azureResourcesExpectStatus(t, do(http.MethodPost, solution+"/warmup"+azureHelpTestAPIVersion, `{"parameters":{"a":"b"}}`), http.StatusNoContent)
	azureSentinelExpectRejected(t, do(http.MethodPost, solution+"/warmup"+azureHelpTestAPIVersion, `{"parameters":{"a":1}}`), "parameters must be an object of strings")

	simple := providerContractJSONMap(t, do(http.MethodPut, vm+"/providers/Microsoft.Help/simplifiedSolutions/ss1"+azureHelpTestAPIVersion, `{"properties":{"solutionId":"stackyard-keyvault-recovery-solution"}}`))
	if ssp := simple["properties"].(map[string]any); ssp["title"] == nil || strings.Contains(azureARMString(ssp["content"]), "<!--") || ssp["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected simplified solution: %#v", simple)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, vm+"/providers/Microsoft.Help/simplifiedSolutions/ss2"+azureHelpTestAPIVersion, `{"properties":{"solutionId":"stackyard-keyvault-recover-selfhelp"}}`), "properties.solutionId must name a solution")

	// Names are checked per scope.
	check := func(scope, name, typ string) map[string]any {
		return providerContractJSONMap(t, do(http.MethodPost, scope+"/providers/Microsoft.Help/checkNameAvailability"+azureHelpTestAPIVersion, `{"name":"`+name+`","type":"`+typ+`"}`))
	}
	if r := check(vm, "s1", "Microsoft.Help/solutions"); r["nameAvailable"] != false || r["reason"] != "AlreadyExists" {
		t.Fatalf("s1 is taken at the VM: %#v", r)
	}
	if r := check(sub, "s1", "Microsoft.Help/solutions"); r["nameAvailable"] != true {
		t.Fatalf("s1 is free at the subscription: %#v", r)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, vm+"/providers/Microsoft.Help/checkNameAvailability"+azureHelpTestAPIVersion, `{"name":"x","type":"Microsoft.Help/discoverySolutions"}`), "type must be")
	azureSentinelExpectRejected(t, do(http.MethodPost, vm+"/providers/Microsoft.Help/checkNameAvailability"+azureHelpTestAPIVersion, `{"name":" ","type":"Microsoft.Help/solutions"}`), "name is required")
	azureSentinelExpectRejected(t, do(http.MethodPost, vm+"/providers/Microsoft.Help/checkNameAvailability"+azureHelpTestAPIVersion, `{"name":"s1/x","type":"Microsoft.Help/solutions"}`), "can't contain '/'")

	// A resource group isn't a resource scope for a diagnostic needing one.
	atGroup := providerContractJSONMap(t, do(http.MethodPut, sub+"/resourceGroups/rg-help/providers/Microsoft.Help/diagnostics/d5"+azureHelpTestAPIVersion, `{"properties":{"insights":[{"solutionId":"stackyard-vm-rdp-diagnostic"}]}}`))
	if atGroup["properties"].(map[string]any)["provisioningState"] != "Failed" {
		t.Fatalf("a resource group scope lacks the ResourceUri: %#v", atGroup)
	}
}

func TestAzureHelpTroubleshooters(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-help")
	ts := sub + "/resourceGroups/rg-help/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.Help/troubleshooters/t1"
	steps := func() (map[string]any, []any) {
		props := providerContractJSONMap(t, do(http.MethodGet, ts+azureHelpTestAPIVersion, ""))["properties"].(map[string]any)
		return props, props["steps"].([]any)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, ts+azureHelpTestAPIVersion, `{"properties":{"solutionId":"stackyard-vm-connectivity-troubleshooter","parameters":{"a":"b"}}}`), http.StatusCreated)
	props, list := steps()
	first := list[0].(map[string]any)
	if props["provisioningState"] != "Running" || len(list) != 1 || first["id"] != "running" || first["isLastStep"] != false {
		t.Fatalf("unexpected first step: %#v", props)
	}
	if opts := first["inputs"].([]any)[0].(map[string]any)["responseOptions"].([]any); len(opts) != 2 {
		t.Fatalf("expected Yes and No options: %#v", first)
	}

	cont := func(body string) *http.Response {
		return do(http.MethodPost, ts+"/continue"+azureHelpTestAPIVersion, body)
	}
	azureSentinelExpectRejected(t, cont(`{"stepId":"rules","responses":[]}`), "stepId must be the current step")
	azureSentinelExpectRejected(t, cont(`{"stepId":"running","responses":[{"questionId":"running","response":"Maybe"}]}`), "response options")
	azureResourcesExpectStatus(t, cont(`{"stepId":"running","responses":[{"questionId":"running","questionType":"RadioButton","response":"Yes"}]}`), http.StatusNoContent)
	props, list = steps()
	if len(list) != 2 || list[1].(map[string]any)["id"] != "rules" || list[0].(map[string]any)["inputs"].([]any)[0].(map[string]any)["selectedOptionValue"] != "Yes" {
		t.Fatalf("continue should record the answer and move on: %#v", list)
	}
	azureResourcesExpectStatus(t, cont(`{"stepId":"rules","responses":[{"questionId":"rules","response":"No"}]}`), http.StatusNoContent)
	props, list = steps()
	if props["provisioningState"] != "Succeeded" || list[2].(map[string]any)["id"] != "open-port" || list[2].(map[string]any)["isLastStep"] != true {
		t.Fatalf("the workflow should end at a solution step: %#v", props)
	}
	azureResourcesExpectErrorCode(t, cont(`{"stepId":"open-port"}`), http.StatusConflict, "Conflict")
	azureResourcesExpectStatus(t, do(http.MethodPost, ts+"/end"+azureHelpTestAPIVersion, ""), http.StatusNoContent)
	if props, _ = steps(); props["provisioningState"] != "Succeeded" {
		t.Fatalf("ending a finished troubleshooter should leave it Succeeded: %#v", props)
	}

	// Restart begins again; end cancels a running troubleshooter.
	restarted := providerContractJSONMap(t, do(http.MethodPost, ts+"/restart"+azureHelpTestAPIVersion, ""))
	if restarted["troubleshooterResourceName"] != "t1" {
		t.Fatalf("unexpected restart: %#v", restarted)
	}
	if props, list = steps(); props["provisioningState"] != "Running" || len(list) != 1 {
		t.Fatalf("restart should return to the first step: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, ts+"/end"+azureHelpTestAPIVersion, ""), http.StatusNoContent)
	if props, _ = steps(); props["provisioningState"] != "Canceled" {
		t.Fatalf("end should cancel: %#v", props)
	}

	// PUT keeps the workflow and can't switch troubleshooters; clients
	// can't set steps.
	again := providerContractJSONMap(t, do(http.MethodPut, ts+azureHelpTestAPIVersion, `{"properties":{"solutionId":"stackyard-vm-connectivity-troubleshooter","steps":[]}}`))
	if ap := again["properties"].(map[string]any); ap["provisioningState"] != "Canceled" || len(ap["steps"].([]any)) != 1 {
		t.Fatalf("PUT should keep the workflow: %#v", again)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, ts+azureHelpTestAPIVersion, `{"properties":{"solutionId":"stackyard-vm-rdp-diagnostic"}}`), "must name a troubleshooter")
	azureSentinelExpectRejected(t, do(http.MethodPut, ts+azureHelpTestAPIVersion, `{"properties":{"solutionId":"stackyard-keyvault-recovery-troubleshooter"}}`), "solutionId can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPut, ts+azureHelpTestAPIVersion, `{"properties":{"solutionId":"stackyard-vm-connectivity-troubleshooter","parameters":"x"}}`), "parameters must be an object of strings")
	azureResourcesExpectStatus(t, do(http.MethodPatch, ts+azureHelpTestAPIVersion, `{}`), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.Help/troubleshooters/none/continue"+azureHelpTestAPIVersion, `{}`), http.StatusNotFound)
}
