package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureTerraformTestAPIVersion = "?api-version=2026-09-01-preview"

// newAzureTerraformTestServer stores a virtual network and a Lab Services
// lab plan (which azurerm doesn't map) in rg-tf.
func newAzureTerraformTestServer(t *testing.T) (string, func(method, path, body string) *http.Response, func(body string) map[string]any) {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-tf")
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-tf/providers/Microsoft.Network/virtualNetworks/vnet1?api-version=2024-05-01",
		`{"location":"eastus","tags":{"env":"${dev}"},"properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-tf/providers/Microsoft.LabServices/labPlans/plan1?api-version=2023-06-07",
		`{"location":"eastus","properties":{}}`), http.StatusCreated)
	export := func(body string) map[string]any {
		t.Helper()
		resp := do(http.MethodPost, sub+"/providers/Microsoft.AzureTerraform/exportTerraform"+azureTerraformTestAPIVersion, body)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		out := providerContractJSONMap(t, resp)
		if out["status"] != "Succeeded" || !strings.Contains(azureARMString(out["id"]), "/providers/Microsoft.AzureTerraform/operationStatuses/") {
			t.Fatalf("unexpected export status: %#v", out)
		}
		return out["properties"].(map[string]any)
	}
	return sub, do, export
}

func TestAzureTerraformExportResourceGroup(t *testing.T) {
	t.Parallel()
	sub, _, export := newAzureTerraformTestServer(t)
	props := export(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf"}`)
	config, imports := azureARMString(props["configuration"]), azureARMString(props["import"])
	for _, want := range []string{
		`provider "azurerm"`, `resource "azurerm_resource_group" "res-0"`, `name      = "rg-tf"`,
		`resource "azurerm_virtual_network" "res-1"`, `resource_group_name = "rg-tf"`, `"env":"$${dev}"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("configuration lacks %q:\n%s", want, config)
		}
	}
	if !strings.Contains(imports, "to = azurerm_virtual_network.res-1") || !strings.Contains(imports, `id = "`+strings.TrimPrefix(sub, "/azure")+`/resourceGroups/rg-tf"`) {
		t.Fatalf("unexpected import blocks:\n%s", imports)
	}
	// The lab plan and the images ARM created with it have no azurerm type.
	skipped := props["skippedResources"].([]any)
	if len(skipped) != 3 || !strings.HasSuffix(azureARMString(skipped[0]), "/labPlans/plan1") || !strings.Contains(azureARMString(skipped[1]), "/labPlans/plan1/images/") {
		t.Fatalf("azurerm should skip the lab plan and its images: %#v", skipped)
	}

	// azapi takes every type, with the version it was written with.
	props = export(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","targetProvider":"azapi","namePattern":"x-","excludeAzureResource":[".*/virtualNetworks/.*"]}`)
	config = azureARMString(props["configuration"])
	for _, want := range []string{
		`type      = "Microsoft.Resources/resourceGroups@2021-04-01"`, `type      = "Microsoft.LabServices/labPlans@2023-06-07"`,
		`parent_id = "` + strings.TrimPrefix(sub, "/azure") + `/resourceGroups/rg-tf"`, `resource "azapi_resource" "x-1"`, `body      = {"properties":`,
		`type      = "Microsoft.LabServices/labPlans/images@2023-06-07"`, `parent_id = "` + strings.TrimPrefix(sub, "/azure") + `/resourceGroups/rg-tf/providers/Microsoft.LabServices/labPlans/plan1"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("azapi configuration lacks %q:\n%s", want, config)
		}
	}
	if strings.Contains(config, "virtualNetworks") || len(props["skippedResources"].([]any)) != 0 {
		t.Fatalf("excluded network exported or resources skipped:\n%s %#v", config, props["skippedResources"])
	}
	props = export(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","excludeTerraformResource":["azurerm_virtual_network"]}`)
	if strings.Contains(azureARMString(props["configuration"]), "azurerm_virtual_network") {
		t.Fatal("excludeTerraformResource should drop the network")
	}
}

func TestAzureTerraformExportEscapingAndExtensions(t *testing.T) {
	t.Parallel()
	sub, do, export := newAzureTerraformTestServer(t)
	arm := strings.TrimPrefix(sub, "/azure")
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-tf/providers/Microsoft.Network/virtualNetworks/vnet1?api-version=2024-05-01",
		`{"location":"eastus","tags":{"a":"%{x}","b":"say \"hi\"\n"},"properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`), http.StatusOK)
	config := azureARMString(export(`{"type":"ExportQuery","query":"name == 'vnet1'"}`)["configuration"])
	if !strings.Contains(config, `"a":"%%{x}"`) || !strings.Contains(config, `"b":"say \"hi\"\n"`) {
		t.Fatalf("template sequences, quotes and newlines should be escaped:\n%s", config)
	}

	vm := arm + "/resourceGroups/rg-tf/providers/Microsoft.Compute/virtualMachines/vm1"
	azureResourcesExpectStatus(t, do(http.MethodPut, "/azure"+vm+"/providers/Microsoft.SerialConsole/serialPorts/0?api-version=2024-07-01", `{"properties":{"state":"enabled"}}`), http.StatusCreated)
	config = azureARMString(export(`{"type":"ExportResource","resourceIds":["` + vm + `/providers/Microsoft.SerialConsole/serialPorts/0"],"targetProvider":"azapi"}`)["configuration"])
	if !strings.Contains(config, `type      = "Microsoft.SerialConsole/serialPorts@2024-07-01"`) || !strings.Contains(config, `parent_id = "`+vm+`"`) {
		t.Fatalf("an extension resource's parent is its scope:\n%s", config)
	}
}

func TestAzureTerraformExportResourcesAndQueries(t *testing.T) {
	t.Parallel()
	sub, do, export := newAzureTerraformTestServer(t)
	arm := strings.TrimPrefix(sub, "/azure")
	vnet := arm + "/resourceGroups/rg-tf/providers/Microsoft.Network/virtualNetworks/vnet1"

	props := export(`{"type":"ExportResource","resourceIds":["` + vnet + `","` + arm + `/resourceGroups/rg-tf/providers/Microsoft.Network/virtualNetworks/gone"],"resourceName":"net","resourceType":"azurerm_virtual_network_custom"}`)
	config := azureARMString(props["configuration"])
	if !strings.Contains(config, `resource "azurerm_virtual_network" "res-0"`) || strings.Contains(config, "azurerm_resource_group") {
		t.Fatalf("resourceName and resourceType apply only to a single resource:\n%s", config)
	}
	errs := props["errors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["code"] != "ResourceNotFound" {
		t.Fatalf("missing IDs should be export errors: %#v", errs)
	}
	props = export(`{"type":"ExportResource","resourceIds":["` + vnet + `"],"resourceName":"net","resourceType":"azurerm_virtual_network_custom","includeResourceGroup":true}`)
	config = azureARMString(props["configuration"])
	if !strings.Contains(config, `resource "azurerm_virtual_network_custom" "net"`) || !strings.Contains(config, "azurerm_resource_group") {
		t.Fatalf("single export should honor resourceName, resourceType and includeResourceGroup:\n%s", config)
	}
	props = export(`{"type":"ExportResource","resourceIds":["` + arm + `/resourceGroups/rg-tf"],"recursive":true}`)
	if config = azureARMString(props["configuration"]); !strings.Contains(config, "azurerm_virtual_network") {
		t.Fatalf("recursive export of a group should include its resources:\n%s", config)
	}

	props = export(`{"type":"ExportQuery","query":"resourceGroup =~ \"RG-TF\" and type =~ 'microsoft.network/virtualnetworks'"}`)
	if config = azureARMString(props["configuration"]); !strings.Contains(config, "azurerm_virtual_network") || strings.Contains(config, "azurerm_resource_group") {
		t.Fatalf("query should select the network only:\n%s", config)
	}
	props = export(`{"type":"ExportQuery","query":"name == \"VNET1\""}`)
	if strings.Contains(azureARMString(props["configuration"]), "azurerm_virtual_network") {
		t.Fatal("== should compare case-sensitively")
	}
	props = export(`{"type":"ExportQuery","query":"type !~ \"microsoft.network/virtualnetworks\" and name != \"x\""}`)
	if config = azureARMString(props["configuration"]); strings.Contains(config, "azurerm_virtual_network") || len(props["skippedResources"].([]any)) != 3 {
		t.Fatalf("!~ should drop the network and keep the lab plan:\n%s %#v", config, props["skippedResources"])
	}

	props = export(`{"type":"ExportQuery","query":"name != \"vnet1\""}`)
	if strings.Contains(azureARMString(props["configuration"]), "azurerm_virtual_network") {
		t.Fatal("!= should drop the named resource")
	}

	// Records outside the handler's PUT have no API version, so azapi
	// skips them; IDs in another subscription name nothing here.
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.SerialConsole/consoleServices/default/disableConsole?api-version=2024-07-01", ""), http.StatusOK)
	console := arm + "/providers/Microsoft.SerialConsole/consoleServices/default"
	props = export(`{"type":"ExportResource","resourceIds":["` + console + `"],"targetProvider":"azapi"}`)
	if skipped := props["skippedResources"].([]any); len(skipped) != 1 || skipped[0] != console {
		t.Fatalf("azapi should skip records without an API version: %#v", props)
	}
	other := "/azure/subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/rg-tf"
	azureResourcesExpectStatus(t, do(http.MethodPut, other+"?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, other+"/providers/Microsoft.Network/virtualNetworks/vnet2?api-version=2024-05-01", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.1.0.0/16"]}}}`), http.StatusCreated)
	props = export(`{"type":"ExportResource","resourceIds":["` + strings.TrimPrefix(other, "/azure") + `/providers/Microsoft.Network/virtualNetworks/vnet2"]}`)
	if len(props["errors"].([]any)) != 1 || strings.Contains(azureARMString(props["configuration"]), "vnet2") {
		t.Fatalf("another subscription's resource should not export: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourcegroups/rg-other?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-other/providers/Microsoft.Network/virtualNetworks/vnet3?api-version=2024-05-01", `{"location":"eastus","properties":{"addressSpace":{"addressPrefixes":["10.2.0.0/16"]}}}`), http.StatusCreated)
	props = export(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf"}`)
	if config := azureARMString(props["configuration"]); strings.Contains(config, "vnet2") || strings.Contains(config, "vnet3") {
		t.Fatalf("a group export should hold only that group's resources:\n%s", config)
	}

	// Like Resource Graph, type compares in lower case; quoted values may
	// hold "and"; a group listed twice exports once.
	props = export(`{"type":"ExportQuery","query":"type == 'microsoft.network/virtualnetworks' and name != 'a and b'"}`)
	if !strings.Contains(azureARMString(props["configuration"]), "azurerm_virtual_network") {
		t.Fatalf("lower-case type with == should match:\n%s", props["configuration"])
	}
	props = export(`{"type":"ExportResource","resourceIds":["` + arm + `/resourceGroups/rg-tf","` + arm + `/resourceGroups/rg-tf"]}`)
	if n := strings.Count(azureARMString(props["configuration"]), "azurerm_resource_group"); n != 1 {
		t.Fatalf("expected one resource group block, got %d", n)
	}

	expect := func(body, code, want string) {
		t.Helper()
		resp := do(http.MethodPost, sub+"/providers/Microsoft.AzureTerraform/exportTerraform"+azureTerraformTestAPIVersion, body)
		out := providerContractJSONMap(t, resp)
		errBody, _ := out["error"].(map[string]any)
		if resp.StatusCode/100 != 4 || errBody["code"] != code || !strings.Contains(azureARMString(errBody["message"]), want) {
			t.Fatalf("expected %s mentioning %q, got %d %#v", code, want, resp.StatusCode, out)
		}
	}
	expect(`{"type":"ExportAll"}`, "InvalidRequestContent", "type must be one of")
	expect(`{"type":"ExportResource","resourceIds":[]}`, "InvalidRequestContent", "resourceIds")
	expect(`{"type":"ExportResource","resourceIds":[1]}`, "InvalidRequestContent", "resourceIds")
	expect(`{"type":"ExportResourceGroup"}`, "InvalidRequestContent", "resourceGroupName")
	expect(`{"type":"ExportQuery","query":" "}`, "InvalidRequestContent", "query is required")
	expect(`{"type":"ExportQuery","query":"type == \"x\"","table":"ResourceContainers"}`, "InvalidRequestContent", "table must be Resources")
	expect(`{"type":"ExportQuery","query":"type == \"x\"","authorizationScopeFilter":"Everywhere"}`, "InvalidRequestContent", "authorizationScopeFilter")
	expect(`{"type":"ExportQuery","query":"tags.env == \"x\""}`, "InvalidQuery", "predicates")
	expect(`{"type":"ExportQuery","query":"sku == \"x\""}`, "InvalidQuery", "sku")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","recursive":"yes"}`, "InvalidRequestContent", "recursive must be true or false")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","namePattern":1}`, "InvalidRequestContent", "namePattern must be a Terraform identifier")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","namePattern":""}`, "InvalidRequestContent", "namePattern must be a Terraform identifier")
	expect(`{"type":"ExportResource","resourceIds":["`+vnet+`"],"resourceName":"my net"}`, "InvalidRequestContent", "resourceName must be a Terraform identifier")
	expect(`{"type":"ExportResource","resourceIds":["`+vnet+`"],"resourceType":"x {"}`, "InvalidRequestContent", "resourceType must be a Terraform identifier")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","targetProvider":"aws"}`, "InvalidRequestContent", "targetProvider")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","includeExtensions":["alerts"]}`, "InvalidRequestContent", "includeExtensions")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","excludeAzureResource":["("]}`, "InvalidRequestContent", "regular expressions")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"rg-tf","excludeTerraformResource":"x"}`, "InvalidRequestContent", "list of strings")
	expect(`{"type":"ExportResourceGroup","resourceGroupName":"missing"}`, "ResourceGroupNotFound", "missing")
	resp := do(http.MethodPost, sub+"/resourceGroups/rg-tf/providers/Microsoft.AzureTerraform/exportTerraform"+azureTerraformTestAPIVersion, `{"type":"ExportResourceGroup","resourceGroupName":"rg-tf"}`)
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidScope")
}
