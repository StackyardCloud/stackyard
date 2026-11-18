package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const azureFileSharesTestAPIVersion = "?api-version=2026-09-01"

func TestAzureFileSharesProvisioning(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-files")
	share := sub + "/resourceGroups/rg-files/providers/Microsoft.FileShares/fileShares/share1"

	// Defaults and recommendations: 100 GiB recommends 3,100 IOPS and
	// 110 MiB/s, and bursting follows the provisioned IOPS.
	resp := do(http.MethodPut, share+azureFileSharesTestAPIVersion, `{"location":"eastus","tags":{"env":"dev"},
		"properties":{"redundancy":"Local","provisionedStorageGiB":100,"nfsProtocolProperties":{"rootSquash":"RootSquash"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	for key, want := range map[string]any{
		"mediaTier": "SSD", "protocol": "NFS", "scaleMode": "Compatibility", "mountName": "share1", "provisioningState": "Succeeded",
		"provisionedIOPerSec": float64(3100), "provisionedThroughputMiBPerSec": float64(110),
		"includedBurstIOPerSec": float64(10000), "maxBurstIOPerSecCredits": float64((10000 - 3100) * 3600),
	} {
		if props[key] != want {
			t.Fatalf("%s = %#v, want %#v: %#v", key, props[key], want, props)
		}
	}
	if !strings.HasSuffix(azureARMString(props["hostName"]), ".file.stackyard.invalid") || props["provisionedStorageNextAllowedDowngrade"] == nil {
		t.Fatalf("missing host name or downgrade time: %#v", props)
	}

	// Increasing is allowed; decreasing within 24 hours of an increase isn't.
	patched := providerContractJSONMap(t, do(http.MethodPatch, share+azureFileSharesTestAPIVersion, `{"properties":{"provisionedStorageGiB":200,"provisionedIOPerSec":6000}}`))
	pp := patched["properties"].(map[string]any)
	if pp["provisionedStorageGiB"] != float64(200) || pp["provisionedIOPerSec"] != float64(6000) || pp["includedBurstIOPerSec"] != float64(18000) || pp["provisionedThroughputMiBPerSec"] != float64(110) {
		t.Fatalf("unexpected patched share: %#v", pp)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, share+azureFileSharesTestAPIVersion, `{"properties":{"provisionedStorageGiB":150}}`), "can't decrease until")
	azureSentinelExpectRejected(t, do(http.MethodPut, share+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Local","provisionedStorageGiB":200}}`), "provisionedIOPerSec can't decrease")
	unchanged := providerContractJSONMap(t, do(http.MethodPatch, share+azureFileSharesTestAPIVersion, `{"properties":{"provisionedThroughputMiBPerSec":110,"publicNetworkAccess":"Disabled"}}`))
	if up := unchanged["properties"].(map[string]any); up["provisionedStorageNextAllowedDowngrade"] != pp["provisionedStorageNextAllowedDowngrade"] {
		t.Fatalf("an unchanged quantity should keep its downgrade time: %#v", up)
	}

	// Limits and guardrails: 5x the recommendation for the storage.
	azureSentinelExpectRejected(t, do(http.MethodPatch, share+azureFileSharesTestAPIVersion, `{"properties":{"provisionedIOPerSec":16001}}`), "provisionedIOPerSec must be from 3000 to 16000")
	azureSentinelExpectRejected(t, do(http.MethodPatch, share+azureFileSharesTestAPIVersion, `{"properties":{"provisionedThroughputMiBPerSec":99}}`), "provisionedThroughputMiBPerSec must be from 100 to 600")
	azureSentinelExpectRejected(t, do(http.MethodPatch, share+azureFileSharesTestAPIVersion, `{"properties":{"provisionedStorageGiB":262145}}`), "provisionedStorageGiB must be from 32 to 262144")
	other := sub + "/resourceGroups/rg-files/providers/Microsoft.FileShares/fileShares/share2"
	azureSentinelExpectRejected(t, do(http.MethodPut, other+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Zone","provisionedStorageGiB":31}}`), "provisionedStorageGiB must be from 32")

	// Fixed properties and the update whitelist.
	azureSentinelExpectRejected(t, do(http.MethodPut, share+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Zone","provisionedStorageGiB":200,"provisionedIOPerSec":6000}}`), "start a redundancy conversion")
	azureSentinelExpectRejected(t, do(http.MethodPut, share+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Local","mountName":"other","provisionedStorageGiB":200,"provisionedIOPerSec":6000}}`), "mountName can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPatch, share+azureFileSharesTestAPIVersion, `{"properties":{"redundancy":"Zone"}}`), "redundancy can't be updated")
	kept := providerContractJSONMap(t, do(http.MethodPut, share+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Local","provisionedStorageGiB":200,"provisionedIOPerSec":6000}}`))
	if kp := kept["properties"].(map[string]any); kp["mountName"] != "share1" || kp["hostName"] != props["hostName"] || kp["provisionedThroughputMiBPerSec"] != float64(120) {
		t.Fatalf("PUT should keep fixed properties and recommend throughput again: %#v", kp)
	}
}

func TestAzureFileSharesValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-files")
	share := sub + "/resourceGroups/rg-files/providers/Microsoft.FileShares/fileShares/share1" + azureFileSharesTestAPIVersion
	body := func(props string) string {
		return `{"location":"eastus","properties":{"redundancy":"Local","provisionedStorageGiB":64` + props + `}}`
	}
	for _, tc := range []struct{ body, want string }{
		{`{"location":"eastus","properties":{"redundancy":"Local"}}`, "provisionedStorageGiB is required"},
		{`{"location":"eastus","properties":{"provisionedStorageGiB":64}}`, "redundancy is required"},
		{body(`,"mediaTier":"HDD"`), "properties.mediaTier must be one of SSD"},
		{body(`,"protocol":"SMB"`), "properties.protocol"},
		{body(`,"redundancy":"Geo"`), "properties.redundancy"},
		{body(`,"scaleMode":"Huge"`), "properties.scaleMode"},
		{body(`,"publicNetworkAccess":"On"`), "properties.publicNetworkAccess"},
		{body(`,"provisionedIOPerSec":1.5`), "provisionedIOPerSec must be a whole number"},
		{body(`,"nfsProtocolProperties":{"rootSquash":"AllSquash"}`), "rootSquash"},
		{body(`,"nfsProtocolProperties":{"encryptionInTransitRequired":"Yes"}`), "encryptionInTransitRequired"},
		{body(`,"publicAccessProperties":{"allowedSubnets":["x"]}`), "allowedSubnets must list subnet IDs"},
		{body(`,"encryption":{"infrastructureEncryption":"on"}`), "infrastructureEncryption"},
		{body(`,"encryption":{"customerManagedKeyEncryption":{"keyEncryptionKeyUrl":"http://x"}}`), "keyEncryptionKeyUrl"},
		{body(`,"encryption":{"customerManagedKeyEncryption":{"keyEncryptionKeyIdentity":{"identityType":"userAssignedIdentity"}}}`), "userAssignedIdentityResourceId is required"},
		{body(`,"encryption":{"customerManagedKeyEncryption":{"keyEncryptionKeyIdentity":{"identityType":"x"}}}`), "identityType"},
		{body(`,"mountName":"Bad--Name"`), "mountName"},
		{`{"location":"eastus","identity":{"type":"Managed"},"properties":{"redundancy":"Local","provisionedStorageGiB":64}}`, "identity.type"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, share, tc.body), tc.want)
	}
	for _, name := range []string{"ab", "Upper", "bad--name", "-lead", "trail-"} {
		resp := do(http.MethodPut, sub+"/resourceGroups/rg-files/providers/Microsoft.FileShares/fileShares/"+name+azureFileSharesTestAPIVersion, body(""))
		azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidResourceName")
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, share, body(`,"publicAccessProperties":{"allowedSubnets":["/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/v/subnets/s"]},"encryption":{"infrastructureEncryption":"enabled","customerManagedKeyEncryption":{"keyEncryptionKeyUrl":"https://v.vault.azure.net/keys/k","keyEncryptionKeyIdentity":{"identityType":"systemAssignedIdentity"}}}`)), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPatch, share, `{"properties":{"scaleMode":"Scale"}}`), "scaleMode can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPatch, share, `{"properties":{"mediaTier":"SSD"}}`), "mediaTier can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPatch, share, `{"properties":{"publicNetworkAccess":"Off"}}`), "publicNetworkAccess")
}

func TestAzureFileSharesChildren(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-files")
	share := sub + "/resourceGroups/rg-files/providers/Microsoft.FileShares/fileShares/share1"
	azureResourcesExpectStatus(t, do(http.MethodPut, share+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Local","provisionedStorageGiB":64}}`), http.StatusCreated)

	// Snapshots take metadata and an initiator; PATCH changes metadata only.
	snap := share + "/fileShareSnapshots/snap1"
	created := providerContractJSONMap(t, do(http.MethodPut, snap+azureFileSharesTestAPIVersion, `{"properties":{"initiatorId":"backup","metadata":{"k":"v"}}}`))
	sp := created["properties"].(map[string]any)
	if sp["snapshotTime"] == nil || sp["initiatorId"] != "backup" || created["type"] != "Microsoft.FileShares/fileShares/fileShareSnapshots" {
		t.Fatalf("unexpected snapshot: %#v", created)
	}
	patched := providerContractJSONMap(t, do(http.MethodPatch, snap+azureFileSharesTestAPIVersion, `{"properties":{"metadata":{"k":"w"}}}`))
	if pp := patched["properties"].(map[string]any); pp["snapshotTime"] != sp["snapshotTime"] || pp["metadata"].(map[string]any)["k"] != "w" {
		t.Fatalf("PATCH should change metadata and keep the snapshot time: %#v", patched)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, snap+azureFileSharesTestAPIVersion, `{"properties":{"initiatorId":"x"}}`), "initiatorId can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPut, snap+azureFileSharesTestAPIVersion, `{"properties":{"metadata":{"k":1}}}`), "metadata must be an object of strings")
	azureSentinelExpectRejected(t, do(http.MethodPut, snap+azureFileSharesTestAPIVersion, `{"properties":{"initiatorId":1}}`), "initiatorId must be a string")
	if list := providerContractJSONMap(t, do(http.MethodGet, share+"/fileShareSnapshots"+azureFileSharesTestAPIVersion, ""))["value"].([]any); len(list) != 1 {
		t.Fatalf("expected one snapshot, got %#v", list)
	}

	// Private endpoint connections list on the share; private link
	// resources describe the fileshare group.
	conn := share + "/privateEndpointConnections/pe1"
	pec := providerContractJSONMap(t, do(http.MethodPut, conn+azureFileSharesTestAPIVersion, `{"properties":{"privateEndpoint":{"id":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/privateEndpoints/pe"},"privateLinkServiceConnectionState":{"status":"Approved","description":"ok"}}}`))
	if pp := pec["properties"].(map[string]any); pp["provisioningState"] != "Succeeded" || pp["groupIds"].([]any)[0] != "fileshare" {
		t.Fatalf("unexpected connection: %#v", pec)
	}
	got := providerContractJSONMap(t, do(http.MethodGet, share+azureFileSharesTestAPIVersion, ""))
	if conns := got["properties"].(map[string]any)["privateEndpointConnections"].([]any); len(conns) != 1 {
		t.Fatalf("share should list its connection: %#v", got)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, conn+azureFileSharesTestAPIVersion, `{"properties":{}}`), "privateLinkServiceConnectionState is required")
	azureSentinelExpectRejected(t, do(http.MethodPut, conn+azureFileSharesTestAPIVersion, `{"properties":{"privateLinkServiceConnectionState":{"status":"Removed"}}}`), "status must be Pending, Approved or Rejected")
	azureSentinelExpectRejected(t, do(http.MethodPut, conn+azureFileSharesTestAPIVersion, `{"properties":{"privateEndpoint":{"id":"x"},"privateLinkServiceConnectionState":{"status":"Pending"}}}`), "privateEndpoint.id")
	azureResourcesExpectStatus(t, do(http.MethodPatch, conn+azureFileSharesTestAPIVersion, `{}`), http.StatusMethodNotAllowed)
	plr := providerContractJSONMap(t, do(http.MethodGet, share+"/privateLinkResources/fileshare"+azureFileSharesTestAPIVersion, ""))
	if plr["properties"].(map[string]any)["groupId"] != "fileshare" {
		t.Fatalf("unexpected private link resource: %#v", plr)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, conn+azureFileSharesTestAPIVersion, ""), http.StatusOK)

	// Redundancy conversion completes at once.
	azureResourcesExpectErrorCode(t, do(http.MethodPost, share+"/getRedundancyConversionStatus"+azureFileSharesTestAPIVersion, ""), http.StatusNotFound, "NoRedundancyConversion")
	azureResourcesExpectStatus(t, do(http.MethodPost, share+"/startRedundancyConversion"+azureFileSharesTestAPIVersion, `{"properties":{"targetRedundancy":"Zone"}}`), http.StatusNoContent)
	status := providerContractJSONMap(t, do(http.MethodPost, share+"/getRedundancyConversionStatus"+azureFileSharesTestAPIVersion, ""))
	if sp := status["properties"].(map[string]any); sp["targetRedundancy"] != "Zone" || sp["conversionStatus"] != "Complete" {
		t.Fatalf("unexpected conversion status: %#v", status)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, share+azureFileSharesTestAPIVersion, "")); got["properties"].(map[string]any)["redundancy"] != "Zone" {
		t.Fatalf("conversion should change redundancy: %#v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, share+"/startRedundancyConversion"+azureFileSharesTestAPIVersion, `{"properties":{"targetRedundancy":"Zone"}}`), http.StatusConflict, "Conflict")
	azureSentinelExpectRejected(t, do(http.MethodPost, share+"/startRedundancyConversion"+azureFileSharesTestAPIVersion, `{"properties":{"targetRedundancy":"Geo"}}`), "targetRedundancy")

	// Deleting the share deletes its children.
	azureResourcesExpectStatus(t, do(http.MethodDelete, share+azureFileSharesTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, snap+azureFileSharesTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureFileSharesLocationActions(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-files")
	loc := sub + "/providers/Microsoft.FileShares/locations/eastus/"
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-files/providers/Microsoft.FileShares/fileShares/share1"+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Local","provisionedStorageGiB":64}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-files/providers/Microsoft.FileShares/fileShares/share2"+azureFileSharesTestAPIVersion, `{"location":"westus","properties":{"redundancy":"Local","provisionedStorageGiB":64}}`), http.StatusCreated)

	usage := providerContractJSONMap(t, do(http.MethodPost, loc+"getUsageData"+azureFileSharesTestAPIVersion, ""))
	if n := usage["properties"].(map[string]any)["liveShares"].(map[string]any)["fileShareCount"]; n != float64(1) {
		t.Fatalf("expected one share in eastus, got %#v", usage)
	}
	limits := providerContractJSONMap(t, do(http.MethodPost, loc+"getLimits"+azureFileSharesTestAPIVersion, `{"properties":{}}`))["properties"].(map[string]any)
	if l := limits["limits"].(map[string]any); l["maxFileShares"] != float64(10000) || l["maxFileShareSnapshots"] != float64(200) || l["maxProvisionedIOPerSec"] != float64(102400) {
		t.Fatalf("unexpected limits: %#v", limits)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, loc+"getLimits"+azureFileSharesTestAPIVersion, `{"properties":{"mediaTier":"HDD"}}`), "mediaTier")
	azureSentinelExpectRejected(t, do(http.MethodPost, loc+"getLimits"+azureFileSharesTestAPIVersion, `{}`), "properties is required")

	rec := providerContractJSONMap(t, do(http.MethodPost, loc+"getProvisioningRecommendation"+azureFileSharesTestAPIVersion, `{"properties":{"provisionedStorageGiB":1000}}`))["properties"].(map[string]any)
	if rec["provisionedIOPerSec"] != float64(4000) || rec["provisionedThroughputMiBPerSec"] != float64(200) {
		t.Fatalf("unexpected recommendation: %#v", rec)
	}
	big := providerContractJSONMap(t, do(http.MethodPost, loc+"getProvisioningRecommendation"+azureFileSharesTestAPIVersion, `{"properties":{"provisionedStorageGiB":262144}}`))["properties"].(map[string]any)
	if big["provisionedIOPerSec"] != float64(102400) || big["provisionedThroughputMiBPerSec"] != float64(10340) {
		t.Fatalf("recommendations should cap at the maximums: %#v", big)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, loc+"getProvisioningRecommendation"+azureFileSharesTestAPIVersion, `{"properties":{"provisionedStorageGiB":10}}`), "provisionedStorageGiB must be from 32")

	check := func(name string) map[string]any {
		return providerContractJSONMap(t, do(http.MethodPost, loc+"checkNameAvailability"+azureFileSharesTestAPIVersion, `{"name":"`+name+`","type":"Microsoft.FileShares/fileShares"}`))
	}
	if r := check("share1"); r["nameAvailable"] != false || r["reason"] != "AlreadyExists" {
		t.Fatalf("share1 is taken in eastus: %#v", r)
	}
	if r := check("share2"); r["nameAvailable"] != true {
		t.Fatalf("share2 is only in westus: %#v", r)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourcegroups/rg-other?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, sub+"/resourceGroups/rg-other/providers/Microsoft.FileShares/fileShares/share1"+azureFileSharesTestAPIVersion, `{"location":"eastus","properties":{"redundancy":"Local","provisionedStorageGiB":64}}`), "already exists in this subscription and region")
	if r := check("Bad"); r["reason"] != "Invalid" {
		t.Fatalf("expected an invalid name: %#v", r)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, loc+"checkNameAvailability"+azureFileSharesTestAPIVersion, `{"name":"x","type":"Microsoft.Storage/storageAccounts"}`), "type must be")
	azureResourcesExpectStatus(t, do(http.MethodGet, loc+"getUsageData"+azureFileSharesTestAPIVersion, ""), http.StatusMethodNotAllowed)
}

func TestAzureFileSharesRegionLimit(t *testing.T) {
	t.Parallel()
	s := &Server{azureARMResources: map[string]*azureARMResource{}}
	for i := range azureFileShareMaxShares {
		id := fmt.Sprintf("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.FileShares/fileShares/s%d", i)
		s.azureARMResources[azureARMKey(id)] = &azureARMResource{ID: id, SubscriptionID: "sub", Type: azureFileSharesNamespace + "/fileShares", Location: "East US"}
	}
	rec := &azureARMResource{ID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.FileShares/fileShares/one-more", Name: "one-more", SubscriptionID: "sub", Location: "eastus",
		Properties: map[string]any{"provisionedStorageGiB": float64(64)}}
	if _, msg := azureFileShareReconcile(s, nil, rec); !strings.Contains(msg, "at most 10000 file shares") {
		t.Fatalf("expected the region limit, got %q", msg)
	}
	rec.Location = "westus"
	if _, msg := azureFileShareReconcile(s, nil, rec); msg != "" {
		t.Fatalf("another region has room: %q", msg)
	}
}
