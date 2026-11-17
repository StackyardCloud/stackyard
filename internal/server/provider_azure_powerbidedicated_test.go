package server

import (
	"net/http"
	"testing"
)

const azurePowerBITestAPIVersion = "?api-version=2021-01-01"

func TestAzurePowerBIDedicatedCapacityLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-powerbi")
	ns := sub + "/resourceGroups/rg-powerbi/providers/Microsoft.PowerBIDedicated"
	capacity := ns + "/capacities/embedded1"

	created := providerContractJSONMap(t, do(http.MethodPut, capacity+azurePowerBITestAPIVersion,
		`{"location":"westus","sku":{"name":"A1"},"properties":{"administration":{"members":["admin@contoso.com"]}}}`))
	props := created["properties"].(map[string]any)
	if created["type"] != "Microsoft.PowerBIDedicated/capacities" || created["sku"].(map[string]any)["tier"] != "PBIE_Azure" || props["state"] != "Succeeded" ||
		props["provisioningState"] != "Succeeded" || props["mode"] != "Gen2" {
		t.Fatalf("unexpected capacity: %#v", created)
	}

	scaled := providerContractJSONMap(t, do(http.MethodPatch, capacity+azurePowerBITestAPIVersion, `{"sku":{"name":"A3"},"tags":{"env":"test"}}`))
	if sku := scaled["sku"].(map[string]any); sku["name"] != "A3" || sku["tier"] != "PBIE_Azure" || scaled["tags"].(map[string]any)["env"] != "test" {
		t.Fatalf("scaling should keep the tier: %#v", scaled)
	}
	skus := providerContractJSONMap(t, do(http.MethodGet, capacity+"/skus"+azurePowerBITestAPIVersion, ""))["value"].([]any)
	if len(skus) != 5 {
		t.Fatalf("expected every A SKU but A3: %#v", skus)
	}
	for _, item := range skus {
		if item.(map[string]any)["sku"].(map[string]any)["name"] == "A3" {
			t.Fatalf("the current SKU shouldn't be listed: %#v", skus)
		}
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, ns+"/capacities/missing1/skus"+azurePowerBITestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")

	azureResourcesExpectStatus(t, do(http.MethodPost, capacity+"/suspend"+azurePowerBITestAPIVersion, ""), http.StatusOK)
	paused := providerContractJSONMap(t, do(http.MethodGet, capacity+azurePowerBITestAPIVersion, ""))["properties"].(map[string]any)
	if paused["state"] != "Paused" || paused["provisioningState"] != "Paused" {
		t.Fatalf("suspend should pause the capacity: %#v", paused)
	}
	// Updates keep the paused state.
	updated := providerContractJSONMap(t, do(http.MethodPatch, capacity+azurePowerBITestAPIVersion, `{"properties":{"mode":"Gen1"}}`))["properties"].(map[string]any)
	if updated["state"] != "Paused" || updated["provisioningState"] != "Paused" || updated["mode"] != "Gen1" {
		t.Fatalf("PATCH should keep the paused state: %#v", updated)
	}
	// A PUT to a paused capacity keeps it paused, whatever state it sends.
	reput := providerContractJSONMap(t, do(http.MethodPut, capacity+azurePowerBITestAPIVersion,
		`{"location":"westus","sku":{"name":"A3"},"properties":{"state":"Succeeded","provisioningState":"Succeeded","mode":"Gen1"}}`))["properties"].(map[string]any)
	if reput["state"] != "Paused" || reput["provisioningState"] != "Paused" {
		t.Fatalf("a PUT should keep the paused state: %#v", reput)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, capacity+"/resume"+azurePowerBITestAPIVersion, ""), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodGet, capacity+azurePowerBITestAPIVersion, ""))["properties"].(map[string]any); got["state"] != "Succeeded" || got["provisioningState"] != "Succeeded" {
		t.Fatalf("resume should restart the capacity: %#v", got)
	}

	for payload, want := range map[string]string{
		`{"sku":{"name":"P1"}}`:                            "sku.name",
		`{"sku":{"name":"A2","tier":"Premium"}}`:           "sku.tier",
		`{"properties":{"mode":"Gen3"}}`:                   "properties.mode",
		`{"properties":{"administration":{"members":[]}}}`: "at least one administrator",
		`{"properties":{"state":"Paused"}}`:                "properties.state can't be updated",
		`{"properties":"Gen1"}`:                            "properties must be an object",
		`{"sku":"A2"}`:                                     "sku must be an object",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPatch, capacity+azurePowerBITestAPIVersion, payload), want)
	}

	subSkus := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.PowerBIDedicated/skus"+azurePowerBITestAPIVersion, ""))["value"].([]any)
	if len(subSkus) != 6 || subSkus[0].(map[string]any)["name"] != "A1" {
		t.Fatalf("unexpected subscription skus: %#v", subSkus)
	}
	check := sub + "/providers/Microsoft.PowerBIDedicated/locations/westus/checkNameAvailability" + azurePowerBITestAPIVersion
	if got := providerContractJSONMap(t, do(http.MethodPost, check, `{"name":"embedded1","type":"Microsoft.PowerBIDedicated/capacities"}`)); got["nameAvailable"] != false || got["reason"] != "Unavailable" {
		t.Fatalf("a used name should be unavailable: %#v", got)
	}
	if got := providerContractJSONMap(t, do(http.MethodPost, check, `{"name":"Embedded-2"}`)); got["nameAvailable"] != false || got["reason"] != "Invalid" {
		t.Fatalf("an invalid name should be reported: %#v", got)
	}
	if got := providerContractJSONMap(t, do(http.MethodPost, check, `{"name":"embedded2"}`)); got["nameAvailable"] != true {
		t.Fatalf("a free name should be available: %#v", got)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, check, `{"name":"embedded2","type":"Microsoft.Web/sites"}`), "type must be")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.PowerBIDedicated/locations/westus/checkNameAvailability?api-version=", `{"name":"embedded2"}`),
		http.StatusBadRequest, "InvalidApiVersionParameter")
	fresh := providerContractJSONMap(t, do(http.MethodPut, ns+"/capacities/embedded3"+azurePowerBITestAPIVersion,
		`{"location":"westus","sku":{"name":"A1"},"properties":{"state":"Paused","provisioningState":"Paused"}}`))["properties"].(map[string]any)
	if fresh["state"] != "Succeeded" || fresh["provisioningState"] != "Succeeded" {
		t.Fatalf("a new capacity starts running whatever state it sends: %#v", fresh)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, capacity+azurePowerBITestAPIVersion, ""), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodPost, check, `{"name":"embedded1"}`)); got["nameAvailable"] != true {
		t.Fatalf("a deleted capacity's name should be free: %#v", got)
	}
}

func TestAzurePowerBIDedicatedValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-powerbi")
	ns := sub + "/resourceGroups/rg-powerbi/providers/Microsoft.PowerBIDedicated"
	for name, tc := range map[string]struct{ path, body, want string }{
		"no sku":      {"/capacities/cap1", `{"location":"westus"}`, "sku must be an object"},
		"sku":         {"/capacities/cap1", `{"location":"westus","sku":{"name":"F2"}}`, "sku.name"},
		"admins":      {"/capacities/cap1", `{"location":"westus","sku":{"name":"A1"},"properties":{"administration":{"members":[""]}}}`, "administration.members must hold"},
		"mode":        {"/capacities/cap1", `{"location":"westus","sku":{"name":"A1"},"properties":{"mode":"Gen9"}}`, "properties.mode"},
		"vcore sku":   {"/autoScaleVCores/vcore1", `{"location":"westus","sku":{"name":"A1"}}`, "sku.name must be AutoScale"},
		"vcore tier":  {"/autoScaleVCores/vcore1", `{"location":"westus","sku":{"name":"AutoScale","tier":"PBIE_Azure"}}`, "sku.tier"},
		"vcore size":  {"/autoScaleVCores/vcore1", `{"location":"westus","sku":{"name":"AutoScale","capacity":-1}}`, "sku.capacity"},
		"vcore limit": {"/autoScaleVCores/vcore1", `{"location":"westus","sku":{"name":"AutoScale"},"properties":{"capacityLimit":1.5}}`, "capacityLimit"},
		"properties":  {"/capacities/cap1", `{"location":"westus","sku":{"name":"A1"},"properties":[]}`, "properties must be an object"},
		"vcore owner": {"/autoScaleVCores/vcore1", `{"location":"westus","sku":{"name":"AutoScale"},"properties":{"capacityObjectId":"cap1"}}`, "capacityObjectId"},
	} {
		resp := do(http.MethodPut, ns+tc.path+azurePowerBITestAPIVersion, tc.body)
		t.Run(name, func(t *testing.T) { azureSentinelExpectRejected(t, resp, tc.want) })
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, ns+"/capacities/Cap1"+azurePowerBITestAPIVersion, `{"location":"westus","sku":{"name":"A1"}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, ns+"/autoScaleVCores/v!"+azurePowerBITestAPIVersion, `{"location":"westus","sku":{"name":"AutoScale"}}`), http.StatusBadRequest, "InvalidResourceName")
}

func TestAzurePowerBIDedicatedAutoScaleVCores(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-powerbi")
	vcore := sub + "/resourceGroups/rg-powerbi/providers/Microsoft.PowerBIDedicated/autoScaleVCores/burst_1"
	created := providerContractJSONMap(t, do(http.MethodPut, vcore+azurePowerBITestAPIVersion,
		`{"location":"westus","sku":{"name":"AutoScale","capacity":0},"properties":{"capacityLimit":10,"capacityObjectId":"a28f00bd-5330-4572-88f1-fa883e074785"}}`))
	if created["sku"].(map[string]any)["tier"] != "AutoScale" {
		t.Fatalf("the tier should be filled in: %#v", created)
	}
	rescaled := providerContractJSONMap(t, do(http.MethodPatch, vcore+azurePowerBITestAPIVersion, `{"sku":{"name":"AutoScale","capacity":2}}`))
	if sku := rescaled["sku"].(map[string]any); sku["tier"] != "AutoScale" || sku["capacity"] != float64(2) {
		t.Fatalf("PATCH should keep the tier: %#v", rescaled)
	}
	if p := created["properties"].(map[string]any); created["type"] != "Microsoft.PowerBIDedicated/autoScaleVCores" || p["capacityLimit"] != float64(10) || p["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected v-core: %#v", created)
	}
	updated := providerContractJSONMap(t, do(http.MethodPatch, vcore+azurePowerBITestAPIVersion, `{"properties":{"capacityLimit":20}}`))
	if p := updated["properties"].(map[string]any); p["capacityLimit"] != float64(20) || p["capacityObjectId"] != "a28f00bd-5330-4572-88f1-fa883e074785" {
		t.Fatalf("unexpected updated v-core: %#v", updated)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, vcore+azurePowerBITestAPIVersion, `{"properties":{"capacityObjectId":"a28f00bd-5330-4572-88f1-fa883e074786"}}`), "capacityObjectId can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPatch, vcore+azurePowerBITestAPIVersion, `{"sku":{"name":"A1"}}`), "AutoScale")
	azureSentinelExpectRejected(t, do(http.MethodPatch, vcore+azurePowerBITestAPIVersion, `{"properties":{"capacityLimit":-2}}`), "capacityLimit")
	list := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.PowerBIDedicated/autoScaleVCores"+azurePowerBITestAPIVersion, ""))["value"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected one v-core in the subscription: %#v", list)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, vcore+azurePowerBITestAPIVersion, ""), http.StatusOK)
}
