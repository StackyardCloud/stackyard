package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const azureWorkloadsTestAPIVersion = "?api-version=2023-04-01"

const azureWorkloadsTestSubnet = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-sap/providers/Microsoft.Network/virtualNetworks/vnet1/subnets/subnet1"

const azureWorkloadsTestIdentity = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-sap/providers/Microsoft.ManagedIdentity/userAssignedIdentities/acss"

// azureWorkloadsTestServer returns a layer of a three-tier system.
func azureWorkloadsTestServer(count int) map[string]any {
	return map[string]any{
		"subnetId": azureWorkloadsTestSubnet, "instanceCount": count,
		"virtualMachineConfiguration": map[string]any{
			"vmSize":         "Standard_E16ds_v4",
			"imageReference": map[string]any{"publisher": "RedHat", "offer": "RHEL-SAP", "sku": "84sapha-gen2", "version": "latest"},
			"osProfile": map[string]any{"adminUsername": "azureuser", "osConfiguration": map[string]any{
				"osType": "Linux", "disablePasswordAuthentication": true,
				"sshKeyPair": map[string]any{"publicKey": "ssh-rsa AAAAB3Nza", "privateKey": "-----BEGIN RSA PRIVATE KEY-----"},
			}},
		},
	}
}

// azureWorkloadsTestVIS returns a three-tier S/4HANA deployment, with its
// software when software is set, changed by edit.
func azureWorkloadsTestVIS(software bool, edit func(props, infra map[string]any)) string {
	db := azureWorkloadsTestServer(1)
	db["databaseType"] = "HANA"
	infra := map[string]any{
		"deploymentType": "ThreeTier", "appResourceGroup": "X00-RG",
		"centralServer": azureWorkloadsTestServer(1), "applicationServer": azureWorkloadsTestServer(2), "databaseServer": db,
	}
	config := map[string]any{"configurationType": "Deployment", "appLocation": "eastus", "infrastructureConfiguration": infra}
	if software {
		config["softwareConfiguration"] = azureWorkloadsTestSoftware()
	}
	props := map[string]any{"environment": "Prod", "sapProduct": "S4HANA", "configuration": config}
	if edit != nil {
		edit(props, infra)
	}
	body, _ := json.Marshal(map[string]any{"location": "eastus", "properties": props})
	return string(body)
}

func azureWorkloadsTestSoftware() map[string]any {
	return map[string]any{
		"softwareInstallationType": "ServiceInitiated", "bomUrl": "https://teststorageaccount.blob.core.windows.net/sapbits/S4HANA_2021.yaml",
		"softwareVersion": "SAP S/4HANA 2021 ISS 00", "sapFqdn": "sap.bpaas.com",
		"sapBitsStorageAccountId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-sap/providers/Microsoft.Storage/storageAccounts/sapbits",
	}
}

func newAzureWorkloadsTestServer(t *testing.T) (string, string, func(method, path, body string) *http.Response) {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-sap")
	return sub, sub + "/resourceGroups/rg-sap/providers/Microsoft.Workloads/", do
}

func TestAzureWorkloadsVirtualInstances(t *testing.T) {
	t.Parallel()
	sub, base, do := newAzureWorkloadsTestServer(t)
	vis := base + "sapVirtualInstances/X00"
	created := providerContractJSONMap(t, do(http.MethodPut, vis+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, nil)))
	props := created["properties"].(map[string]any)
	if props["state"] != "RegistrationComplete" || props["status"] != "Running" || props["health"] != "Healthy" || props["provisioningState"] != "Succeeded" ||
		props["managedResourceGroupConfiguration"].(map[string]any)["name"] != "mrg-x00-00000000-0000-0000-0000-000000000000" {
		t.Fatalf("created = %#v", props)
	}
	if strings.Contains(string(mustBody(t, do(http.MethodGet, vis+azureWorkloadsTestAPIVersion, ""))), "PRIVATE KEY") {
		t.Fatal("private key returned")
	}
	for _, list := range []string{base + "sapVirtualInstances", sub + "/providers/Microsoft.Workloads/sapVirtualInstances"} {
		if items := providerContractJSONMap(t, do(http.MethodGet, list+azureWorkloadsTestAPIVersion, ""))["value"].([]any); len(items) != 1 {
			t.Fatalf("%s = %#v", list, items)
		}
	}

	// Learn's update model takes tags and identity; properties can't change
	// on PUT either.
	patched := providerContractJSONMap(t, do(http.MethodPatch, vis+azureWorkloadsTestAPIVersion, `{"tags":{"env":"prod"}}`))
	if patched["tags"].(map[string]any)["env"] != "prod" || patched["properties"].(map[string]any)["state"] != "RegistrationComplete" {
		t.Fatalf("patched = %#v", patched)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, vis+azureWorkloadsTestAPIVersion, `{"properties":{"environment":"NonProd"}}`), "properties can't be updated")
	azureSentinelExpectRejected(t, do(http.MethodPut, vis+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, func(props, _ map[string]any) { props["environment"] = "NonProd" })),
		"environment can't be changed after creation")
	azureSentinelExpectRejected(t, do(http.MethodPut, vis+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, func(_, infra map[string]any) { infra["appResourceGroup"] = "other" })),
		"configuration can't be changed")
	azureResourcesExpectStatus(t, do(http.MethodPut, vis+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, nil)), http.StatusOK)

	// Infrastructure alone waits for its software, which a later PUT adds.
	pending := base + "sapVirtualInstances/X01"
	got := providerContractJSONMap(t, do(http.MethodPut, pending+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(false, nil)))["properties"].(map[string]any)
	if got["state"] != "SoftwareInstallationPending" || got["status"] != "Unavailable" || got["health"] != "Unknown" {
		t.Fatalf("infrastructure only = %#v", got)
	}
	azureWorkloadsExpectError(t, do(http.MethodPost, pending+"/start"+azureWorkloadsTestAPIVersion, ""), http.StatusConflict, "Conflict", "until its software is installed")
	azureSentinelExpectRejected(t, do(http.MethodPut, pending+"/centralInstances/cs"+azureWorkloadsTestAPIVersion, `{"location":"eastus"}`), "registered system")
	// Installing software can't also change the infrastructure.
	azureSentinelExpectRejected(t, do(http.MethodPut, pending+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, func(_, infra map[string]any) { infra["appResourceGroup"] = "other" })),
		"configuration can't be changed")
	installed := providerContractJSONMap(t, do(http.MethodPut, pending+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, nil)))["properties"].(map[string]any)
	if installed["state"] != "RegistrationComplete" || installed["status"] != "Running" {
		t.Fatalf("installed = %#v", installed)
	}

	// Registering an existing system needs a user-assigned identity.
	discovery := `{"location":"eastus","properties":{"environment":"NonProd","sapProduct":"ECC","configuration":{"configurationType":"Discovery",
		"centralServerVmId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-sap/providers/Microsoft.Compute/virtualMachines/ascs"}}`
	azureSentinelExpectRejected(t, do(http.MethodPut, base+"sapVirtualInstances/X02"+azureWorkloadsTestAPIVersion, discovery+`}`), "user-assigned identity")
	registered := providerContractJSONMap(t, do(http.MethodPut, base+"sapVirtualInstances/X02"+azureWorkloadsTestAPIVersion,
		discovery+`,"identity":{"type":"UserAssigned","userAssignedIdentities":{"`+azureWorkloadsTestIdentity+`":{}}}}`))
	if registered["properties"].(map[string]any)["state"] != "RegistrationComplete" {
		t.Fatalf("registered = %#v", registered)
	}
}

func TestAzureWorkloadsVirtualInstanceValidation(t *testing.T) {
	t.Parallel()
	_, base, do := newAzureWorkloadsTestServer(t)
	osProfile := func(infra map[string]any) map[string]any {
		return infra["centralServer"].(map[string]any)["virtualMachineConfiguration"].(map[string]any)["osProfile"].(map[string]any)
	}
	volumes := map[string]any{}
	for _, name := range azureWorkloadsHANAVolumes {
		volumes[name] = map[string]any{"count": 1, "sizeGB": 128, "sku": map[string]any{"name": "Premium_LRS"}}
	}
	for _, tc := range []struct {
		name, want string
		edit       func(props, infra map[string]any)
	}{
		{"X00", "environment must be one of", func(props, _ map[string]any) { props["environment"] = "Dev" }},
		{"X00", "sapProduct is required", func(props, _ map[string]any) { delete(props, "sapProduct") }},
		{"X00", "configuration is required", func(props, _ map[string]any) { delete(props, "configuration") }},
		{"X00", "configurationType must be one of", func(props, _ map[string]any) {
			props["configuration"].(map[string]any)["configurationType"] = "Other"
		}},
		{"X00", "appLocation is required", func(props, _ map[string]any) { delete(props["configuration"].(map[string]any), "appLocation") }},
		{"X00", "deploymentType must be one of", func(_, infra map[string]any) { infra["deploymentType"] = "Distributed" }},
		{"X00", "centralServer.subnetId must be a subnet", func(_, infra map[string]any) {
			infra["centralServer"].(map[string]any)["subnetId"] = "subnet1"
		}},
		{"X00", "centralServer.instanceCount must be 1 for a system without high availability", func(_, infra map[string]any) {
			infra["centralServer"].(map[string]any)["instanceCount"] = 2
		}},
		{"X00", "databaseServer.instanceCount must be 2 for a highly available system", func(_, infra map[string]any) {
			infra["highAvailabilityConfig"] = map[string]any{"highAvailabilityType": "AvailabilityZone"}
			infra["centralServer"].(map[string]any)["instanceCount"] = 2
		}},
		{"X00", "fencingClientId is required", func(props, infra map[string]any) {
			infra["highAvailabilityConfig"] = map[string]any{"highAvailabilityType": "AvailabilitySet"}
			azureARMObject(infra["centralServer"])["instanceCount"] = 2
			azureARMObject(infra["databaseServer"])["instanceCount"] = 2
			infra["centralServer"].(map[string]any)["instanceCount"] = 2
			infra["databaseServer"].(map[string]any)["instanceCount"] = 2
			props["configuration"].(map[string]any)["softwareConfiguration"] = azureWorkloadsTestSoftware()
		}},
		{"X00", "applicationServer.instanceCount must be a positive integer", func(_, infra map[string]any) {
			infra["applicationServer"].(map[string]any)["instanceCount"] = 0
		}},
		{"X00", "adminUsername 'admin' isn't allowed", func(_, infra map[string]any) { osProfile(infra)["adminUsername"] = "admin" }},
		{"X00", "ssh-rsa format", func(_, infra map[string]any) {
			osProfile(infra)["osConfiguration"].(map[string]any)["sshKeyPair"] = map[string]any{"publicKey": "ecdsa-sha2 AAAA"}
		}},
		{"X00", "needs an SSH public key", func(_, infra map[string]any) {
			delete(osProfile(infra)["osConfiguration"].(map[string]any), "sshKeyPair")
		}},
		{"X00", "adminPassword is required", func(_, infra map[string]any) {
			osProfile(infra)["osConfiguration"].(map[string]any)["disablePasswordAuthentication"] = false
		}},
		{"X00", "adminPassword is a disallowed value", func(_, infra map[string]any) { osProfile(infra)["adminPassword"] = "P@ssw0rd" }},
		{"X00", "adminPassword must have three of", func(_, infra map[string]any) { osProfile(infra)["adminPassword"] = "lowercase123" }},
		{"X00", "adminPassword must be 6 to 72 characters", func(_, infra map[string]any) { osProfile(infra)["adminPassword"] = "Ab1!" }},
		{"X00", "needs the hana/log volume", func(_, infra map[string]any) {
			infra["databaseServer"].(map[string]any)["diskConfiguration"] = map[string]any{"diskVolumeConfigurations": map[string]any{"hana/data": volumes["hana/data"]}}
		}},
		{"X00", "can't contain 'sapdata'", func(_, infra map[string]any) {
			infra["databaseServer"].(map[string]any)["diskConfiguration"] = map[string]any{"diskVolumeConfigurations": map[string]any{"sapdata": volumes["os"]}}
		}},
		{"X00", "hostName must be at most 13 characters", func(_, infra map[string]any) {
			infra["customResourceNames"] = map[string]any{"namingPatternType": "FullResourceName", "centralServer": map[string]any{
				"virtualMachines": []any{map[string]any{"vmName": "ascsvm", "hostName": "averylonghostname"}}}}
		}},
		{"X00", "virtualMachines must have one entry per VM (2)", func(_, infra map[string]any) {
			infra["customResourceNames"] = map[string]any{"applicationServer": map[string]any{"virtualMachines": []any{map[string]any{"vmName": "app0"}}}}
		}},
		{"X00", "dataDiskNames can't name the 'hana/data' volume", func(_, infra map[string]any) {
			infra["customResourceNames"] = map[string]any{"applicationServer": map[string]any{"virtualMachines": []any{
				map[string]any{"dataDiskNames": map[string]any{"hana/data": []any{"d1"}}}, map[string]any{}}}}
		}},
		{"X00", "configurationType must be one of CreateAndMount, Mount, Skip", func(_, infra map[string]any) {
			infra["storageConfiguration"] = map[string]any{"transportFileShareConfiguration": map[string]any{"configurationType": "Nfs"}}
		}},
		{"X00", "privateEndpointId is required", func(_, infra map[string]any) {
			infra["storageConfiguration"] = map[string]any{"transportFileShareConfiguration": map[string]any{"configurationType": "Mount", "id": "share"}}
		}},
		{"X00", "sapBitsStorageAccountId must be a storage account", func(props, _ map[string]any) {
			sw := azureWorkloadsTestSoftware()
			sw["sapBitsStorageAccountId"] = "sapbits"
			props["configuration"].(map[string]any)["softwareConfiguration"] = sw
		}},
		{"X00", "sapFqdn is required", func(props, _ map[string]any) {
			sw := azureWorkloadsTestSoftware()
			delete(sw, "sapFqdn")
			props["configuration"].(map[string]any)["softwareConfiguration"] = sw
		}},
		{"X00", "softwareInstallationType must be one of", func(props, _ map[string]any) {
			props["configuration"].(map[string]any)["softwareConfiguration"] = map[string]any{"softwareInstallationType": "Manual"}
		}},
		{"X00", "identity must be a user-assigned identity", func(props, _ map[string]any) {
			props["configuration"].(map[string]any)["softwareConfiguration"] = map[string]any{"softwareInstallationType": "External",
				"centralServerVmId": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm"}
		}},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, base+"sapVirtualInstances/"+tc.name+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(false, tc.edit)), tc.want)
	}
	// The VIS name is the three-character SID.
	for _, name := range []string{"X0", "X0A1", "X0!"} {
		azureWorkloadsExpectError(t, do(http.MethodPut, base+"sapVirtualInstances/"+name+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(false, nil)),
			http.StatusBadRequest, "InvalidResourceName", "three letters or digits")
	}

	// A single server takes its own disks, but no high availability.
	single := `{"location":"eastus","properties":{"environment":"NonProd","sapProduct":"S4HANA","configuration":{"configurationType":"DeploymentWithOSConfig","appLocation":"eastus",
		"infrastructureConfiguration":{"deploymentType":"SingleServer","appResourceGroup":"X05-RG","databaseType":"HANA","subnetId":"` + azureWorkloadsTestSubnet + `",
		"virtualMachineConfiguration":{"vmSize":"Standard_E32ds_v4","imageReference":{"publisher":"SUSE","offer":"sles-sap-15-sp3","sku":"gen2","version":"latest"},
		"osProfile":{"adminUsername":"azureuser","adminPassword":"Sap-Admin-123","osConfiguration":{"osType":"Linux"}}}%s}}}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"sapVirtualInstances/X05"+azureWorkloadsTestAPIVersion, strings.Replace(single, "%s", "", 1)), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, base+"sapVirtualInstances/X06"+azureWorkloadsTestAPIVersion,
		strings.Replace(single, "%s", `,"highAvailabilityConfig":{"highAvailabilityType":"AvailabilitySet"}`, 1)), "need a ThreeTier deployment")
}

func TestAzureWorkloadsInstances(t *testing.T) {
	t.Parallel()
	_, base, do := newAzureWorkloadsTestServer(t)
	vis := base + "sapVirtualInstances/X00"
	azureResourcesExpectStatus(t, do(http.MethodPut, vis+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, nil)), http.StatusCreated)
	central, db := vis+"/centralInstances/cs", vis+"/databaseInstances/db"
	app1, app2 := vis+"/applicationInstances/app01", vis+"/applicationInstances/app02"
	for _, id := range []string{central, db, app1, app2} {
		azureResourcesExpectStatus(t, do(http.MethodPut, id+azureWorkloadsTestAPIVersion, `{"location":"East US","properties":{}}`), http.StatusCreated)
	}
	got := providerContractJSONMap(t, do(http.MethodGet, db+azureWorkloadsTestAPIVersion, ""))["properties"].(map[string]any)
	if got["databaseSid"] != "X00" || got["status"] != "Running" || got["provisioningState"] != "Succeeded" {
		t.Fatalf("database = %#v", got)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, vis+"/centralInstances/cs2"+azureWorkloadsTestAPIVersion, `{"location":"eastus"}`), "already has the central services instance cs")
	azureSentinelExpectRejected(t, do(http.MethodPut, vis+"/applicationInstances/app03"+azureWorkloadsTestAPIVersion, `{"location":"westus"}`), "location must be the SAP system's location")
	azureSentinelExpectRejected(t, do(http.MethodPatch, app1+azureWorkloadsTestAPIVersion, `{"properties":{}}`), "properties can't be updated")
	if apps := providerContractJSONMap(t, do(http.MethodGet, vis+"/applicationInstances"+azureWorkloadsTestAPIVersion, ""))["value"].([]any); len(apps) != 2 {
		t.Fatalf("applications = %#v", apps)
	}

	props := func(id string) map[string]any {
		return providerContractJSONMap(t, do(http.MethodGet, id+azureWorkloadsTestAPIVersion, ""))["properties"].(map[string]any)
	}
	status := func(id string) any { return props(id)["status"] }
	azureSentinelExpectRejected(t, do(http.MethodPost, vis+"/stop"+azureWorkloadsTestAPIVersion, `{"softStopTimeoutSeconds":-1}`), "non-negative integer")
	stopped := providerContractJSONMap(t, do(http.MethodPost, vis+"/stop"+azureWorkloadsTestAPIVersion, `{"softStopTimeoutSeconds":300}`))
	if stopped["status"] != "Succeeded" || !strings.Contains(azureARMString(stopped["id"]), "/providers/Microsoft.Workloads/locations/eastus/operationStatuses/") {
		t.Fatalf("stop = %#v", stopped)
	}
	for _, id := range []string{vis, central, db, app1, app2} {
		if status(id) != "Offline" {
			t.Fatalf("%s after stop = %v", id, status(id))
		}
	}
	// An instance added to a stopped system starts offline.
	azureResourcesExpectStatus(t, do(http.MethodPut, vis+"/applicationInstances/app03"+azureWorkloadsTestAPIVersion, `{"location":"eastus"}`), http.StatusCreated)
	if status(vis+"/applicationInstances/app03") != "Offline" {
		t.Fatal("new instance of a stopped system isn't offline")
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, app1+"/start"+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	if status(app1) != "Running" || status(app2) != "Offline" || status(vis) != "PartiallyRunning" {
		t.Fatalf("one instance started: app1 %v, app2 %v, system %v", status(app1), status(app2), status(vis))
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, vis+"/start"+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	if status(vis) != "Running" || status(app2) != "Running" || status(central) != "Running" {
		t.Fatalf("after start: system %v, app2 %v, central %v", status(vis), status(app2), status(central))
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, central+"/stop"+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	if status(vis) != "PartiallyRunning" {
		t.Fatalf("after central stop: system %v", status(vis))
	}
	for _, id := range []string{central, db, app1, app2, vis + "/applicationInstances/app03"} {
		azureResourcesExpectStatus(t, do(http.MethodPost, id+"/stop"+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	}
	if status(vis) != "Offline" {
		t.Fatalf("every instance stopped: system %v; %v %v %v %v %v", status(vis), status(central), status(db), status(app1), status(app2), status(vis+"/applicationInstances/app03"))
	}

	// Deleting an instance updates the system's status from the rest.
	azureResourcesExpectStatus(t, do(http.MethodPost, app2+"/start"+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	if status(vis) != "PartiallyRunning" {
		t.Fatalf("app2 started: system %v", status(vis))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, app2+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	if status(vis) != "Offline" {
		t.Fatalf("running app2 deleted: system %v", status(vis))
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, vis+"/start"+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, app1+"/stop"+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, app1+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	if status(vis) != "Running" || props(vis)["health"] != "Healthy" {
		t.Fatalf("stopped app1 deleted: system %v", props(vis))
	}

	// Deleting the system deletes its instances.
	azureResourcesExpectStatus(t, do(http.MethodDelete, vis+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, vis+azureWorkloadsTestAPIVersion, azureWorkloadsTestVIS(true, nil)), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodGet, central+azureWorkloadsTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureWorkloadsMonitors(t *testing.T) {
	t.Parallel()
	_, base, do := newAzureWorkloadsTestServer(t)
	monitor := base + "monitors/mon1"
	got := providerContractJSONMap(t, do(http.MethodPut, monitor+azureWorkloadsTestAPIVersion,
		`{"location":"westus","properties":{"monitorSubnet":"`+azureWorkloadsTestSubnet+`","managedResourceGroupConfiguration":{"name":"myManagedRg"}}}`))
	props := got["properties"].(map[string]any)
	if props["routingPreference"] != "Default" || props["appLocation"] != "westus" ||
		!strings.Contains(azureARMString(props["logAnalyticsWorkspaceArmId"]), "/resourceGroups/myManagedRg/providers/Microsoft.OperationalInsights/workspaces/") ||
		!strings.Contains(azureARMString(props["storageAccountArmId"]), "/resourceGroups/myManagedRg/providers/Microsoft.Storage/storageAccounts/sapmon") {
		t.Fatalf("monitor = %#v", props)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, monitor+azureWorkloadsTestAPIVersion, `{"location":"westus","properties":{"managedResourceGroupConfiguration":{"name":"other"}}}`),
		"can't be changed after creation")
	kept := providerContractJSONMap(t, do(http.MethodPut, monitor+azureWorkloadsTestAPIVersion, `{"location":"westus","properties":{"routingPreference":"RouteAll"}}`))["properties"].(map[string]any)
	if kept["managedResourceGroupConfiguration"].(map[string]any)["name"] != "myManagedRg" || kept["routingPreference"] != "RouteAll" {
		t.Fatalf("re-PUT = %#v", kept)
	}
	for body, want := range map[string]string{
		`{"location":"westus","properties":{"routingPreference":"Some"}}`:                        "routingPreference must be one of",
		`{"location":"westus","properties":{"monitorSubnet":"subnet1"}}`:                         "monitorSubnet must be a subnet resource ID",
		`{"location":"westus","properties":{"managedResourceGroupConfiguration":{"name":5}}}`:    "name must be 1 to 90 characters",
		`{"location":"westus","properties":{"logAnalyticsWorkspaceArmId":"ws"}}`:                 "Log Analytics workspace resource ID",
		`{"location":"westus","identity":{"type":"SystemAssigned"},"properties":{}}`:             "identity.type must be one of",
		`{"location":"westus","identity":{"type":"UserAssigned"},"properties":{}}`:               "userAssignedIdentities is required",
		`{"location":"westus","properties":{"managedResourceGroupConfiguration":"myManagedRg"}}`: "must be an object",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, base+"monitors/bad"+azureWorkloadsTestAPIVersion, body), want)
	}
	azureSentinelExpectRejected(t, do(http.MethodPatch, monitor+azureWorkloadsTestAPIVersion, `{"properties":{"routingPreference":"Default"}}`), "properties can't be updated")
	// A workspace the client names is used instead of a managed one.
	workspace := "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/rg-sap/providers/microsoft.operationalinsights/workspaces/myWorkspace"
	own := providerContractJSONMap(t, do(http.MethodPut, base+"monitors/mon2"+azureWorkloadsTestAPIVersion, `{"location":"westus","properties":{"logAnalyticsWorkspaceArmId":"`+workspace+`"}}`))
	if own["properties"].(map[string]any)["logAnalyticsWorkspaceArmId"] != workspace {
		t.Fatalf("own workspace = %#v", own["properties"])
	}

	// Provider instances follow Learn's provider guides.
	hana := monitor + "/providerInstances/hana"
	azureResourcesExpectStatus(t, do(http.MethodPut, hana+azureWorkloadsTestAPIVersion, `{"properties":{"providerSettings":{"providerType":"SapHana","hostname":"10.0.0.6",
		"dbUsername":"SYSTEM","dbPassword":"Secret-1","sapSid":"X00","instanceNumber":"00","sqlPort":"30013","sslPreference":"Disabled"}}}`), http.StatusCreated)
	if settings := providerContractJSONMap(t, do(http.MethodGet, hana+azureWorkloadsTestAPIVersion, ""))["properties"].(map[string]any)["providerSettings"].(map[string]any); settings["dbPassword"] != nil || settings["hostname"] != "10.0.0.6" {
		t.Fatalf("provider = %#v", settings)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, hana+azureWorkloadsTestAPIVersion, `{"properties":{"providerSettings":{"providerType":"PrometheusOS","prometheusUrl":"http://10.0.0.6:9100/metrics"}}}`),
		"providerType can't be changed")
	azureResourcesExpectStatus(t, do(http.MethodPatch, hana+azureWorkloadsTestAPIVersion, `{"properties":{}}`), http.StatusMethodNotAllowed)
	for body, want := range map[string]string{
		`{"properties":{}}`: "providerSettings is required",
		`{"properties":{"providerSettings":{"providerType":"Oracle"}}}`:                                                                                                                                                                "providerType must be one of",
		`{"properties":{"providerSettings":{"providerType":"SapHana","hostname":"h","dbUsername":"u","sapSid":"X00"}}}`:                                                                                                                "instanceNumber is required",
		`{"properties":{"providerSettings":{"providerType":"SapHana","hostname":"h","dbUsername":"u","sapSid":"X00","instanceNumber":"100"}}}`:                                                                                         "instance number from 00 to 99",
		`{"properties":{"providerSettings":{"providerType":"SapHana","hostname":"h","dbUsername":"u","sapSid":"X00","instanceNumber":"00"}}}`:                                                                                          "dbPassword or dbPasswordUri",
		`{"properties":{"providerSettings":{"providerType":"Db2","hostname":"h","dbName":"OPA","dbPort":"70000","dbUsername":"u","sapSid":"X00"}}}`:                                                                                    "dbPort must be a port",
		`{"properties":{"providerSettings":{"providerType":"MsSqlServer","hostname":"h","dbPort":"1433","dbUsername":"u","sapSid":"X00","dbPasswordUri":"https://kv.vault.azure.net/secrets/p","sslPreference":"ServerCertificate"}}}`: "sslCertificateUri is required",
		`{"properties":{"providerSettings":{"providerType":"PrometheusOS","prometheusUrl":"10.0.0.6:9100"}}}`:                                                                                                                          "http or https URL",
		`{"properties":{"providerSettings":{"providerType":"PrometheusHaCluster","prometheusUrl":"http://10.0.0.6:9664/metrics","hostname":"h","sid":"X00"}}}`:                                                                         "clusterName is required",
		`{"properties":{"providerSettings":{"providerType":"SapNetWeaver","sapSid":"X00","sapHostname":"h","sapInstanceNr":"1"}}}`:                                                                                                     "instance number from 00 to 99",
		`{"properties":{"providerSettings":{"providerType":"PrometheusOS","prometheusUrl":"http://h:9100","sslPreference":"Some"}}}`:                                                                                                   "sslPreference must be one of",
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, monitor+"/providerInstances/bad"+azureWorkloadsTestAPIVersion, body), want)
	}

	// The landscape monitor is a single "default" dashboard.
	landscape := monitor + "/sapLandscapeMonitor/default"
	body := `{"properties":{"grouping":{"landscape":[{"name":"Prod","topSid":["SID1","SID2"]}],"sapApplication":[{"name":"ERP1","topSid":["SID1"]}]},
		"topMetricsThresholds":[{"name":"Instance Availability","green":90,"yellow":75,"red":50}]}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, landscape+azureWorkloadsTestAPIVersion, body), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+"/sapLandscapeMonitor/other"+azureWorkloadsTestAPIVersion, body), http.StatusBadRequest)
	azureSentinelExpectRejected(t, do(http.MethodPatch, landscape+azureWorkloadsTestAPIVersion, `{"properties":{"topMetricsThresholds":[{"name":"x","green":"high"}]}}`), "green must be a number")
	azureSentinelExpectRejected(t, do(http.MethodPatch, landscape+azureWorkloadsTestAPIVersion, `{"properties":{"grouping":{"landscape":[{"topSid":["SID1"]}]}}}`), "properties.grouping.landscape[].name is required")
	azureSentinelExpectRejected(t, do(http.MethodPatch, landscape+azureWorkloadsTestAPIVersion, `{"properties":{"grouping":{"sapApplication":"ERP1"}}}`), "properties.grouping.sapApplication must be an array")
	patched := providerContractJSONMap(t, do(http.MethodPatch, landscape+azureWorkloadsTestAPIVersion, `{"properties":{"topMetricsThresholds":[{"name":"Instance Availability","green":95,"yellow":80,"red":60}]}}`))
	if patched["properties"].(map[string]any)["grouping"] == nil {
		t.Fatalf("patch dropped the grouping: %#v", patched)
	}
	if list := providerContractJSONMap(t, do(http.MethodGet, monitor+"/sapLandscapeMonitor"+azureWorkloadsTestAPIVersion, ""))["value"].([]any); len(list) != 1 {
		t.Fatalf("landscape list = %#v", list)
	}

	// Deleting the monitor deletes its providers and dashboard.
	azureResourcesExpectStatus(t, do(http.MethodDelete, monitor+azureWorkloadsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, monitor+azureWorkloadsTestAPIVersion, `{"location":"westus","properties":{}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodGet, hana+azureWorkloadsTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodGet, landscape+azureWorkloadsTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureWorkloadsMetadata(t *testing.T) {
	t.Parallel()
	sub, _, do := newAzureWorkloadsTestServer(t)
	post := func(action, body string) *http.Response {
		return do(http.MethodPost, sub+"/providers/Microsoft.Workloads/locations/eastus/sapVirtualInstanceMetadata/default/"+action+azureWorkloadsTestAPIVersion, body)
	}
	const system = `"appLocation":"eastus","environment":"Prod","sapProduct":"S4HANA","databaseType":"HANA"`
	three := providerContractJSONMap(t, post("getSizingRecommendations", `{`+system+`,"deploymentType":"ThreeTier","saps":60000,"dbMemory":1024,"dbScaleMethod":"ScaleUp"}`))
	if three["deploymentType"] != "ThreeTier" || three["dbVmSku"] != "Standard_M64s" || three["applicationServerInstanceCount"] != float64(3) || three["centralServerInstanceCount"] != float64(1) {
		t.Fatalf("three tier = %#v", three)
	}
	ha := providerContractJSONMap(t, post("getSizingRecommendations", `{`+system+`,"deploymentType":"ThreeTier","saps":1000,"dbMemory":200,"highAvailabilityType":"AvailabilityZone"}`))
	if ha["dbVmSku"] != "Standard_M32ls" || ha["databaseInstanceCount"] != float64(2) || ha["applicationServerInstanceCount"] != float64(2) {
		t.Fatalf("highly available = %#v", ha)
	}
	if single := providerContractJSONMap(t, post("getSizingRecommendations", `{`+system+`,"deploymentType":"SingleServer","saps":1000,"dbMemory":192}`)); single["vmSku"] != "Standard_M32ts" {
		t.Fatalf("single = %#v", single)
	}
	for body, want := range map[string]string{
		`{` + system + `,"deploymentType":"SingleServer","saps":1000,"dbMemory":100,"highAvailabilityType":"AvailabilitySet"}`:   "needs a ThreeTier deployment",
		`{` + system + `,"deploymentType":"ThreeTier","saps":1000,"dbMemory":8000}`:                                              "No supported database VM size",
		`{` + system + `,"deploymentType":"ThreeTier","saps":0,"dbMemory":100}`:                                                  "positive integers",
		`{` + system + `,"deploymentType":"ThreeTier","saps":10,"dbMemory":100,"dbScaleMethod":"ScaleOut"}`:                      "dbScaleMethod must be one of",
		`{"appLocation":"eastus","environment":"Prod","sapProduct":"S4HANA","deploymentType":"ThreeTier","saps":1,"dbMemory":1}`: "databaseType is required",
	} {
		azureSentinelExpectRejected(t, post("getSizingRecommendations", body), want)
	}

	skus := providerContractJSONMap(t, post("getSapSupportedSku", `{`+system+`,"deploymentType":"ThreeTier"}`))["supportedSkus"].([]any)
	if len(skus) != len(azureWorkloadsSKUs) || skus[0].(map[string]any)["isAppServerCertified"] != true || skus[len(skus)-1].(map[string]any)["isDatabaseCertified"] != true {
		t.Fatalf("skus = %#v", skus)
	}

	disks := providerContractJSONMap(t, post("getDiskConfigurations", `{`+system+`,"deploymentType":"ThreeTier","dbVmSku":"Standard_M64s"}`))["volumeConfigurations"].(map[string]any)
	data := disks["hana/data"].(map[string]any)["recommendedConfiguration"].(map[string]any)
	log := disks["hana/log"].(map[string]any)["recommendedConfiguration"].(map[string]any)
	// Learn's M64s row: data 4 x P15, log 3 x P15, shared 1 x P30.
	shared := disks["hana/shared"].(map[string]any)["recommendedConfiguration"].(map[string]any)
	if data["count"] != float64(4) || data["sizeGB"] != float64(256) || log["count"] != float64(3) || log["sizeGB"] != float64(256) ||
		shared["sizeGB"] != float64(1024) || len(disks) != 5 {
		t.Fatalf("disks = %#v", disks)
	}
	db2 := providerContractJSONMap(t, post("getDiskConfigurations", `{"appLocation":"eastus","environment":"Prod","sapProduct":"ECC","databaseType":"DB2","deploymentType":"ThreeTier","dbVmSku":"Standard_M64s"}`))
	if len(db2["volumeConfigurations"].(map[string]any)) != 2 {
		t.Fatalf("db2 disks = %#v", db2)
	}
	azureSentinelExpectRejected(t, post("getDiskConfigurations", `{`+system+`,"deploymentType":"ThreeTier","dbVmSku":"Standard_E8ds_v4"}`), "supported database VM size")

	zones := providerContractJSONMap(t, post("getAvailabilityZoneDetails", `{"appLocation":"eastus","sapProduct":"S4HANA","databaseType":"HANA"}`))["availabilityZonePairs"].([]any)
	if len(zones) != 1 || zones[0].(map[string]any)["zoneA"] != float64(1) {
		t.Fatalf("zones = %#v", zones)
	}
	azureSentinelExpectRejected(t, post("getAvailabilityZoneDetails", `{"appLocation":"eastus","databaseType":"Oracle","sapProduct":"S4HANA"}`), "databaseType must be one of")
}

func azureWorkloadsExpectError(t *testing.T, resp *http.Response, status int, code, want string) {
	t.Helper()
	body := providerContractJSONMap(t, resp)
	errBody, _ := body["error"].(map[string]any)
	if resp.StatusCode != status || errBody["code"] != code || !strings.Contains(azureARMString(errBody["message"]), want) {
		t.Fatalf("expected %d %s mentioning %q, got %d %#v", status, code, want, resp.StatusCode, body)
	}
}

func TestAzureWorkloadsSecretsNeverReturned(t *testing.T) {
	t.Parallel()
	sub, base, do := newAzureWorkloadsTestServer(t)
	osProfile := func(infra map[string]any, layer string) map[string]any {
		vm := azureARMObject(infra[layer])["virtualMachineConfiguration"].(map[string]any)
		return vm["osProfile"].(map[string]any)
	}
	threeTier := azureWorkloadsTestVIS(true, func(props, infra map[string]any) {
		for _, layer := range []string{"centralServer", "applicationServer", "databaseServer"} {
			osConfig := osProfile(infra, layer)["osConfiguration"].(map[string]any)
			osConfig["sshKeyPair"].(map[string]any)["privateKey"] = "leak-key-" + layer
		}
		app := osProfile(infra, "applicationServer")
		app["adminPassword"] = "Leak-Pass-1"
		app["osConfiguration"].(map[string]any)["disablePasswordAuthentication"] = false
		infra["highAvailabilityConfig"] = map[string]any{"highAvailabilityType": "AvailabilitySet"}
		azureARMObject(infra["centralServer"])["instanceCount"] = 2
		azureARMObject(infra["databaseServer"])["instanceCount"] = 2
		software := props["configuration"].(map[string]any)["softwareConfiguration"].(map[string]any)
		software["sshPrivateKey"] = "leak-key-software"
		software["highAvailabilitySoftwareConfiguration"] = map[string]any{"fencingClientId": "fence-client", "fencingClientPassword": "Leak-Fence-1"}
	})
	single := `{"location":"eastus","properties":{"environment":"NonProd","sapProduct":"S4HANA","configuration":{"configurationType":"DeploymentWithOSConfig","appLocation":"eastus",
		"infrastructureConfiguration":{"deploymentType":"SingleServer","appResourceGroup":"X05-RG","databaseType":"HANA","subnetId":"` + azureWorkloadsTestSubnet + `",
		"virtualMachineConfiguration":{"vmSize":"Standard_E32ds_v4","imageReference":{"publisher":"SUSE","offer":"sles-sap-15-sp3","sku":"gen2","version":"latest"},
		"osProfile":{"adminUsername":"azureuser","adminPassword":"Leak-Single-1","osConfiguration":{"osType":"Linux"}}}}}}}`
	noLeak := func(what string, resp *http.Response) {
		t.Helper()
		body := string(mustBody(t, resp))
		if resp.StatusCode >= 300 {
			t.Fatalf("%s = %d %s", what, resp.StatusCode, body)
		}
		if strings.Contains(body, "leak-") || strings.Contains(body, "Leak-") {
			t.Fatalf("%s returned a secret: %s", what, body)
		}
	}
	for name, body := range map[string]string{"X00": threeTier, "X05": single} {
		vis := base + "sapVirtualInstances/" + name + azureWorkloadsTestAPIVersion
		noLeak("PUT "+name, do(http.MethodPut, vis, body))
		noLeak("GET "+name, do(http.MethodGet, vis, ""))
	}
	noLeak("list", do(http.MethodGet, base+"sapVirtualInstances"+azureWorkloadsTestAPIVersion, ""))
	noLeak("subscription list", do(http.MethodGet, sub+"/providers/Microsoft.Workloads/sapVirtualInstances"+azureWorkloadsTestAPIVersion, ""))
}
