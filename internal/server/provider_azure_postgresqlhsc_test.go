package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	azurePGTestAPIVersion = "?api-version=2023-03-02-preview"
	azurePGTestPassword   = "Pa55word!"
	azurePGSingleNode     = `{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!","coordinatorServerEdition":"GeneralPurpose",
		"coordinatorStorageQuotaInMb":131072,"coordinatorVCores":4,"nodeCount":0}}`
	azurePGMultiNode = `{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!","coordinatorServerEdition":"GeneralPurpose",
		"coordinatorStorageQuotaInMb":524288,"coordinatorVCores":4,"nodeCount":2,"nodeStorageQuotaInMb":524288,"nodeVCores":8,"enableHa":true}}`
)

func azurePGClusterPath(sub, rg, name string) string {
	return sub + "/resourceGroups/" + rg + "/providers/Microsoft.DBforPostgreSQL/serverGroupsv2/" + name
}

func azurePGList(t *testing.T, resp *http.Response) []any {
	t.Helper()
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	return providerContractJSONMap(t, resp)["value"].([]any)
}

func TestAzurePGClusters(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	cluster := azurePGClusterPath(sub, "rg-pg", "pgcluster")
	resp := do(http.MethodPut, cluster+azurePGTestAPIVersion, azurePGSingleNode)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	for key, want := range map[string]any{
		"administratorLogin": "citus", "state": "Ready", "postgresqlVersion": "16", "citusVersion": "12.1", "databaseName": "citus",
		"aadAuthEnabled": "disabled", "passwordEnabled": "enabled", "enableShardsOnCoordinator": true, "provisioningState": "Succeeded",
	} {
		if props[key] != want {
			t.Fatalf("properties.%s = %#v, want %#v in %#v", key, props[key], want, props)
		}
	}
	if _, leaked := props["administratorLoginPassword"]; leaked {
		t.Fatalf("password returned: %#v", props)
	}
	if names := props["serverNames"].([]any); len(names) != 1 || names[0].(map[string]any)["fullyQualifiedDomainName"] != "pgcluster-c.postgres.database.azure.com" {
		t.Fatalf("serverNames = %#v", names)
	}

	// Scaling out to a multi-node cluster adds worker servers.
	scaled := providerContractJSONMap(t, do(http.MethodPatch, cluster+azurePGTestAPIVersion,
		`{"properties":{"nodeCount":2,"nodeVCores":4,"nodeStorageQuotaInMb":524288}}`))
	if n := len(scaled["properties"].(map[string]any)["serverNames"].([]any)); n != 3 {
		t.Fatalf("scaled serverNames = %d, want 3", n)
	}
	servers := azurePGList(t, do(http.MethodGet, cluster+"/servers"+azurePGTestAPIVersion, ""))
	if len(servers) != 3 || servers[2].(map[string]any)["name"] != "pgcluster-w1" {
		t.Fatalf("servers = %#v", servers)
	}
	worker := providerContractJSONMap(t, do(http.MethodGet, cluster+"/servers/pgcluster-w0"+azurePGTestAPIVersion, ""))
	if p := worker["properties"].(map[string]any); p["role"] != "Worker" || p["vCores"] != float64(4) || p["serverEdition"] != "MemoryOptimized" || p["haState"] != "NotEnabled" {
		t.Fatalf("worker = %#v", worker)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, cluster+"/servers/pgcluster-w9"+azurePGTestAPIVersion, ""), http.StatusNotFound)
	if links := azurePGList(t, do(http.MethodGet, cluster+"/privateLinkResources"+azurePGTestAPIVersion, "")); len(links) != 3 {
		t.Fatalf("private link resources = %#v", links)
	}
	link := providerContractJSONMap(t, do(http.MethodGet, cluster+"/privateLinkResources/worker-1"+azurePGTestAPIVersion, ""))
	if link["properties"].(map[string]any)["requiredZoneNames"].([]any)[0] != "privatelink.pgcluster.postgres.database.azure.com" {
		t.Fatalf("link = %#v", link)
	}

	for _, tc := range []struct{ body, want string }{
		{`{"properties":{"nodeCount":0}}`, "nodeCount can't decrease"},
		{`{"properties":{"coordinatorStorageQuotaInMb":200000}}`, "coordinatorStorageQuotaInMb must be one of"},
		{`{"properties":{"nodeCount":21}}`, "nodeCount must be 0"},
		{`{"properties":{"nodeVCores":2}}`, "nodeVCores must be one of"},
		{`{"properties":{"coordinatorVCores":2}}`, "coordinatorVCores must be one of"},
		{`{"properties":{"nodeServerEdition":"GeneralPurpose"}}`, "nodeServerEdition must be MemoryOptimized"},
		{`{"properties":{"databaseName":"other"}}`, "databaseName can't be updated"},
		{`{"properties":{"postgresqlVersion":"15"}}`, "can only be upgraded"},
		{`{"properties":{"citusVersion":"12.0"}}`, "can only be upgraded"},
		{`{"properties":{"citusVersion":"11.3"}}`, "can only be upgraded"},
		{`{"properties":{"administratorLoginPassword":"password"}}`, "must contain uppercase letters"},
		{`{"properties":{"maintenanceWindow":{"customWindow":"Enabled","dayOfWeek":7}}}`, "dayOfWeek must be a whole number from 0 to 6"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPatch, cluster+azurePGTestAPIVersion, tc.body), tc.want)
	}
	azureResourcesExpectStatus(t, do(http.MethodPatch, cluster+azurePGTestAPIVersion, `{"properties":{"coordinatorStorageQuotaInMb":1048576,"administratorLoginPassword":"N3w-Secret"}}`), http.StatusOK)
	azureSentinelExpectRejected(t, do(http.MethodPatch, cluster+azurePGTestAPIVersion, `{"properties":{"coordinatorStorageQuotaInMb":524288}}`), "can't decrease")

	// A PUT that omits settings keeps them; creation-only ones can't change.
	kept := providerContractJSONMap(t, do(http.MethodPut, cluster+azurePGTestAPIVersion, `{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!"}}`))
	if p := kept["properties"].(map[string]any); p["nodeCount"] != float64(2) || p["coordinatorStorageQuotaInMb"] != float64(1048576) {
		t.Fatalf("PUT should keep omitted settings: %#v", p)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, cluster+azurePGTestAPIVersion, `{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!","enableGeoBackup":true}}`), "enableGeoBackup can only be set at creation")

	// start and stop own the state; restart answers 202 like Learn.
	azureResourcesExpectStatus(t, do(http.MethodPost, cluster+"/stop"+azurePGTestAPIVersion, ""), http.StatusOK)
	if state := providerContractJSONMap(t, do(http.MethodGet, cluster+azurePGTestAPIVersion, ""))["properties"].(map[string]any)["state"]; state != "Stopped" {
		t.Fatalf("state after stop = %v", state)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, cluster+"/start"+azurePGTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, cluster+"/restart"+azurePGTestAPIVersion, ""), http.StatusAccepted)

	// Names are global, and checkNameAvailability shares the flexible
	// server endpoint.
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-other"+azurePGTestAPIVersion, `{"location":"westus"}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, azurePGClusterPath(sub, "rg-other", "pgcluster")+azurePGTestAPIVersion, azurePGSingleNode), "already exists")
	check := sub + "/providers/Microsoft.DBforPostgreSQL/checkNameAvailability" + azurePGTestAPIVersion
	for name, available := range map[string]bool{"pgcluster": false, "free-name": true, "-bad": false, "12345": false, "Upper": false, "ab": false} {
		got := providerContractJSONMap(t, do(http.MethodPost, check, `{"name":"`+name+`","type":"Microsoft.DBforPostgreSQL/serverGroupsv2"}`))
		if got["nameAvailable"] != available || got["type"] != azurePGClusterType {
			t.Fatalf("check %q = %#v", name, got)
		}
	}
	if got := providerContractJSONMap(t, do(http.MethodPost, check, `{"name":"pgcluster","type":"Microsoft.DBforPostgreSQL/flexibleServers"}`)); got["nameAvailable"] != true {
		t.Fatalf("flexible server check = %#v", got)
	}
	for _, name := range []string{"-bad", "bad-", "12345", "Upper", "ab", strings.Repeat("a", 41)} {
		azureResourcesExpectStatus(t, do(http.MethodPut, azurePGClusterPath(sub, "rg-pg", name)+azurePGTestAPIVersion, azurePGSingleNode), http.StatusBadRequest)
	}

	if list := azurePGList(t, do(http.MethodGet, sub+"/providers/Microsoft.DBforPostgreSQL/serverGroupsv2"+azurePGTestAPIVersion, "")); len(list) != 1 {
		t.Fatalf("subscription list = %#v", list)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+azurePGTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, cluster+"/servers"+azurePGTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzurePGClusterValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	cluster := azurePGClusterPath(sub, "rg-pg", "pgvalid") + azurePGTestAPIVersion
	single := func(extra string) string {
		return `{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!","coordinatorServerEdition":"GeneralPurpose",
			"coordinatorStorageQuotaInMb":131072,"coordinatorVCores":4,"nodeCount":0` + extra + `}}`
	}
	for _, tc := range []struct{ body, want string }{
		{`{"location":"westus"}`, "properties is required"},
		{`{"location":"westus","properties":{"coordinatorServerEdition":"GeneralPurpose","coordinatorStorageQuotaInMb":131072,"coordinatorVCores":4,"nodeCount":0}}`, "administratorLoginPassword is required"},
		{`{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!","coordinatorStorageQuotaInMb":131072,"coordinatorVCores":4,"nodeCount":0}}`, "coordinatorServerEdition is required"},
		{`{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!","coordinatorServerEdition":"GeneralPurpose","coordinatorStorageQuotaInMb":131072,"coordinatorVCores":4}}`, "nodeCount is required"},
		{single(`,"administratorLoginPassword":"Short1!"`), "8 to 128 characters"},
		{single(`,"administratorLoginPassword":"Passw0rdOnly"`), "must contain uppercase letters"},
		{single(`,"administratorLoginPassword":"MyCitus99!"`), "role name 'citus'"},
		{single(`,"administratorLoginPassword":"MyTus99!x"`), "role name 'citus'"},
		{single(`,"postgresqlVersion":"14"`), "PostgreSQL 14 is retired"},
		{single(`,"postgresqlVersion":"10"`), "from 11 to 16"},
		{single(`,"citusVersion":"11.3"`), "doesn't match PostgreSQL 16"},
		{single(`,"citusVersion":"twelve"`), "Citus version such as"},
		{single(`,"coordinatorVCores":1`), "coordinatorVCores must be one of"},
		{single(`,"coordinatorStorageQuotaInMb":1048576`), "need 8 or more vCores"},
		{single(`,"coordinatorServerEdition":"BurstableMemoryOptimized","coordinatorVCores":1,"coordinatorStorageQuotaInMb":32768,"enableHa":true`), "don't support high availability"},
		{single(`,"coordinatorServerEdition":"MemoryOptimized"`), "single node cluster must be"},
		{single(`,"nodeCount":2,"nodeVCores":4,"nodeStorageQuotaInMb":524288,"coordinatorServerEdition":"BurstableGeneralPurpose"`), "multi-node cluster must be GeneralPurpose"},
		{single(`,"nodeCount":2,"nodeVCores":4`), "nodeStorageQuotaInMb must be one of"},
		{single(`,"nodeCount":1`), "nodeCount must be 0"},
		{single(`,"enableHa":"yes"`), "enableHa must be a boolean"},
		{single(`,"nodeCount":2.5`), "nodeCount must be a whole number"},
		{single(`,"databaseName":5`), "databaseName must be a string"},
		{single(`,"authConfig":{"activeDirectoryAuth":"on"}`), "activeDirectoryAuth must be one of"},
		{single(`,"authConfig":{"activeDirectoryAuth":"disabled","passwordAuth":"disabled"}`), "must enable password or Microsoft Entra ID"},
		{single(`,"postgresqlVersion":"15","authConfig":{"activeDirectoryAuth":"enabled"},"pointInTimeUTC":"yesterday"`), "pointInTimeUTC must be an RFC 3339"},
		{single(`,"pointInTimeUTC":"2024-01-01T00:00:00Z"`), "sourceResourceId is required"},
		{single(`,"sourceResourceId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.Storage/storageAccounts/z"`), "must be a cluster ID"},
		{single(`,"dataEncryption":{"type":"AzureKeyVault"}`), "primaryKeyUri and primaryUserAssignedIdentityId are required"},
		{single(`,"dataEncryption":{"type":"AzureKeyVault","primaryKeyUri":"https://kv.vault.azure.net/keys/k","primaryUserAssignedIdentityId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id"}`), "must be one of identity.userAssignedIdentities"},
		{single(`,"enableGeoBackup":true,"dataEncryption":{"type":"AzureKeyVault","primaryKeyUri":"k","primaryUserAssignedIdentityId":"i"}`), "aren't supported with geo-redundant backup"},
		{single(`,"dataEncryption":{"type":"Customer"}`), "dataEncryption.type must be one of"},
		{single(`,"maintenanceWindow":{"customWindow":"Sometimes"}`), "customWindow must be one of"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, cluster, tc.body), tc.want)
	}

	// Entra ID authentication alone needs no password, and PostgreSQL 14+.
	azureSentinelExpectRejected(t, do(http.MethodPut, cluster, `{"location":"westus","properties":{"postgresqlVersion":"13","authConfig":{"activeDirectoryAuth":"enabled","passwordAuth":"disabled"},
		"coordinatorServerEdition":"GeneralPurpose","coordinatorStorageQuotaInMb":131072,"coordinatorVCores":4,"nodeCount":0}}`), "PostgreSQL 13 is retired")
	aad := providerContractJSONMap(t, do(http.MethodPut, cluster, `{"location":"westus","identity":{"type":"UserAssigned","userAssignedIdentities":{"/subscriptions/x/resourceGroups/y/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id":{}}},
		"properties":{"authConfig":{"activeDirectoryAuth":"enabled","passwordAuth":"disabled"},"coordinatorServerEdition":"BurstableMemoryOptimized",
		"coordinatorStorageQuotaInMb":32768,"coordinatorVCores":1,"nodeCount":0,
		"dataEncryption":{"type":"AzureKeyVault","primaryKeyUri":"https://kv.vault.azure.net/keys/k","primaryUserAssignedIdentityId":"/subscriptions/x/resourceGroups/y/providers/Microsoft.ManagedIdentity/userAssignedIdentities/ID"}}}`))
	if p := aad["properties"].(map[string]any); p["aadAuthEnabled"] != "enabled" || p["passwordEnabled"] != "disabled" {
		t.Fatalf("aad cluster = %#v", aad)
	}
	// Burstable clusters can't upgrade.
	azureSentinelExpectRejected(t, do(http.MethodPatch, cluster, `{"properties":{"citusVersion":"12.2"}}`), "burstable compute can't upgrade")
}

func TestAzurePGReplicasAndRestores(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	source := azurePGClusterPath(sub, "rg-pg", "pgsource")
	azureResourcesExpectStatus(t, do(http.MethodPut, source+azurePGTestAPIVersion, azurePGMultiNode), http.StatusCreated)
	replica := azurePGClusterPath(sub, "rg-pg", "pgreplica")
	// Request paths carry the emulator's /azure prefix; resource IDs don't.
	sourceID, replicaID, burstableID := strings.TrimPrefix(source, "/azure"), strings.TrimPrefix(replica, "/azure"), strings.TrimPrefix(azurePGClusterPath(sub, "rg-pg", "pgburst"), "/azure")
	resp := do(http.MethodPut, replica+azurePGTestAPIVersion, `{"location":"eastus","properties":{"sourceLocation":"West US","sourceResourceId":"`+strings.ToUpper(sourceID)+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["nodeCount"] != float64(2) || props["nodeVCores"] != float64(8) || props["postgresqlVersion"] != "16" || props["sourceResourceId"] != sourceID || props["sourceLocation"] != "westus" {
		t.Fatalf("replica = %#v", props)
	}

	for _, tc := range []struct{ path, body, want string }{
		{azurePGClusterPath(sub, "rg-pg", "pgchain"), `{"location":"westus","properties":{"sourceResourceId":"` + replicaID + `"}}`, "from another read replica"},
		{azurePGClusterPath(sub, "rg-pg", "pgnone"), `{"location":"westus","properties":{"sourceResourceId":"` + strings.TrimPrefix(azurePGClusterPath(sub, "rg-pg", "missing"), "/azure") + `"}}`, "was not found"},
		{azurePGClusterPath(sub, "rg-pg", "pgwhere"), `{"location":"westus","properties":{"sourceLocation":"eastus","sourceResourceId":"` + sourceID + `"}}`, "source cluster's region"},
		{azurePGClusterPath(sub, "rg-pg", "pgbigger"), `{"location":"westus","properties":{"nodeCount":3,"sourceResourceId":"` + sourceID + `"}}`, "must match its source"},
		{azurePGClusterPath(sub, "rg-pg", "pgfuture"), `{"location":"westus","properties":{"pointInTimeUTC":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `","sourceResourceId":"` + sourceID + `"}}`, "pointInTimeUTC must be between"},
		{azurePGClusterPath(sub, "rg-pg", "pgpast"), `{"location":"westus","properties":{"pointInTimeUTC":"2020-01-01T00:00:00Z","sourceResourceId":"` + sourceID + `"}}`, "pointInTimeUTC must be between"},
		{replica, `{"properties":{"nodeCount":4}}`, "read replica's storage and worker node count"},
	} {
		method := http.MethodPut
		if tc.path == replica {
			method = http.MethodPatch
		}
		azureSentinelExpectRejected(t, do(method, tc.path+azurePGTestAPIVersion, tc.body), tc.want)
	}
	// Replicas can scale compute.
	azureResourcesExpectStatus(t, do(http.MethodPatch, replica+azurePGTestAPIVersion, `{"properties":{"nodeVCores":16}}`), http.StatusOK)

	// A point-in-time restore copies the source's settings.
	at := time.Now().UTC().Format(time.RFC3339Nano)
	restored := providerContractJSONMap(t, do(http.MethodPut, azurePGClusterPath(sub, "rg-pg", "pgrestored")+azurePGTestAPIVersion,
		`{"location":"westus","properties":{"pointInTimeUTC":"`+at+`","sourceResourceId":"`+sourceID+`"}}`))
	if p := restored["properties"].(map[string]any); p["nodeStorageQuotaInMb"] != float64(524288) || p["pointInTimeUTC"] != at {
		t.Fatalf("restored = %#v", restored)
	}
	azureSentinelExpectRejected(t, do(http.MethodPost, azurePGClusterPath(sub, "rg-pg", "pgrestored")+"/promote"+azurePGTestAPIVersion, ""), "isn't a read replica")

	// A cluster has at most five replicas.
	for i := 2; i <= 5; i++ {
		azureResourcesExpectStatus(t, do(http.MethodPut, azurePGClusterPath(sub, "rg-pg", fmt.Sprintf("pgreplica%d", i))+azurePGTestAPIVersion,
			`{"location":"westus","properties":{"sourceResourceId":"`+sourceID+`"}}`), http.StatusCreated)
	}
	azureSentinelExpectRejected(t, do(http.MethodPut, azurePGClusterPath(sub, "rg-pg", "pgreplica6")+azurePGTestAPIVersion,
		`{"location":"westus","properties":{"sourceResourceId":"`+sourceID+`"}}`), "already has 5 read replicas")

	// Promotion makes the replica independent, which frees a replica slot.
	azureSentinelExpectRejected(t, do(http.MethodPost, replica+"/promote"+azurePGTestAPIVersion, `{"enableGeoBackup":"yes"}`), "must be a boolean")
	azureResourcesExpectStatus(t, do(http.MethodPost, replica+"/promote"+azurePGTestAPIVersion, `{"enableGeoBackup":true}`), http.StatusAccepted)
	promoted := providerContractJSONMap(t, do(http.MethodGet, replica+azurePGTestAPIVersion, ""))["properties"].(map[string]any)
	if _, has := promoted["sourceResourceId"]; has || promoted["enableGeoBackup"] != true {
		t.Fatalf("promoted = %#v", promoted)
	}
	azureResourcesExpectStatus(t, do(http.MethodPatch, replica+azurePGTestAPIVersion, `{"properties":{"nodeCount":4}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, azurePGClusterPath(sub, "rg-pg", "pgreplica6")+azurePGTestAPIVersion,
		`{"location":"westus","properties":{"sourceResourceId":"`+sourceID+`"}}`), http.StatusCreated)

	// Burstable clusters can't have replicas; PostgreSQL upgrades in place.
	burstable := azurePGClusterPath(sub, "rg-pg", "pgburst")
	azureResourcesExpectStatus(t, do(http.MethodPut, burstable+azurePGTestAPIVersion, `{"location":"westus","properties":{"administratorLoginPassword":"Pa55word!",
		"coordinatorServerEdition":"BurstableGeneralPurpose","coordinatorStorageQuotaInMb":65536,"coordinatorVCores":2,"nodeCount":0}}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, azurePGClusterPath(sub, "rg-pg", "pgburstrep")+azurePGTestAPIVersion,
		`{"location":"westus","properties":{"sourceResourceId":"`+burstableID+`"}}`), "burstable compute can't have read replicas")

	// A replica's creation-only settings compare without regard to case, and
	// promotion can't add geo-redundant backup to a customer-managed key.
	cmk := azurePGClusterPath(sub, "rg-pg", "pgcmk")
	identity := "/subscriptions/x/resourceGroups/y/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id"
	azureResourcesExpectStatus(t, do(http.MethodPut, cmk+azurePGTestAPIVersion, `{"location":"westus","identity":{"type":"UserAssigned","userAssignedIdentities":{"`+identity+`":{}}},
		"properties":{"sourceResourceId":"`+replicaID+`","dataEncryption":{"type":"AzureKeyVault","primaryKeyUri":"https://kv.vault.azure.net/keys/k","primaryUserAssignedIdentityId":"`+identity+`"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPatch, cmk+azurePGTestAPIVersion, `{"properties":{"coordinatorVCores":8}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, cmk+azurePGTestAPIVersion, `{"location":"westus","identity":{"type":"UserAssigned","userAssignedIdentities":{"`+identity+`":{}}},
		"properties":{"sourceResourceId":"`+strings.ToUpper(replicaID)+`","sourceLocation":"East US","dataEncryption":{"type":"AzureKeyVault","primaryKeyUri":"https://kv.vault.azure.net/keys/k","primaryUserAssignedIdentityId":"`+identity+`"}}}`), http.StatusOK)
	azureSentinelExpectRejected(t, do(http.MethodPost, cmk+"/promote"+azurePGTestAPIVersion, `{"enableGeoBackup":true}`), "aren't supported with geo-redundant backup")
}

// TestAzurePGClusterUpgrade checks in-place upgrades without depending on
// which versions are retired today.
func TestAzurePGClusterUpgrade(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              string
		old, props        map[string]any
		wantCitus, reject string
	}{
		{"patch takes the new default", map[string]any{"postgresqlVersion": "13", "citusVersion": "11.3"}, map[string]any{"postgresqlVersion": "14", "citusVersion": "11.3"}, "12.1", "PostgreSQL 14 is retired"},
		{"put omits citus", map[string]any{"postgresqlVersion": "13", "citusVersion": "11.3"}, map[string]any{"postgresqlVersion": "14"}, "12.1", "PostgreSQL 14 is retired"},
		{"explicit citus kept", map[string]any{"postgresqlVersion": "13", "citusVersion": "11.3"}, map[string]any{"postgresqlVersion": "14", "citusVersion": "12.2"}, "12.2", "PostgreSQL 14 is retired"},
		{"downgrade", map[string]any{"postgresqlVersion": "16", "citusVersion": "12.1"}, map[string]any{"postgresqlVersion": "15"}, "", "can only be upgraded"},
		{"burstable", map[string]any{"postgresqlVersion": "16", "citusVersion": "12.1", "coordinatorServerEdition": "BurstableGeneralPurpose"}, map[string]any{"citusVersion": "12.2"}, "", "burstable compute can't upgrade"},
	} {
		msg := azurePGClusterChange(tc.old, tc.props)
		// 14 is retired (2026-11-12) whatever today is, so those upgrades are
		// rejected after the Citus version is chosen.
		if !strings.Contains(msg, tc.reject) || (tc.wantCitus != "" && tc.props["citusVersion"] != tc.wantCitus) {
			t.Errorf("%s: message %q, citusVersion %v, want %q and %s", tc.name, msg, tc.props["citusVersion"], tc.reject, tc.wantCitus)
		}
	}
	if !azurePGRetired("14") || azurePGRetired("99") {
		t.Fatal("PostgreSQL 14 retired on 2026-11-12, and unknown versions never retire")
	}
}

func TestAzurePGParameters(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	cluster := azurePGClusterPath(sub, "rg-pg", "pgparams")
	azureResourcesExpectStatus(t, do(http.MethodPut, cluster+azurePGTestAPIVersion, azurePGMultiNode), http.StatusCreated)

	param := providerContractJSONMap(t, do(http.MethodGet, cluster+"/coordinatorConfigurations/work_mem"+azurePGTestAPIVersion, ""))
	if p := param["properties"].(map[string]any); p["value"] != "158720" || p["source"] != "system-default" || p["dataType"] != "Integer" {
		t.Fatalf("work_mem = %#v", param)
	}
	resp := do(http.MethodPut, cluster+"/nodeConfigurations/array_nulls"+azurePGTestAPIVersion, `{"properties":{"value":"off"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if p := providerContractJSONMap(t, resp)["properties"].(map[string]any); p["value"] != "off" || p["source"] != "user-override" || p["defaultValue"] != "on" {
		t.Fatalf("array_nulls = %#v", p)
	}
	for _, tc := range []struct{ path, body, want string }{
		{"/coordinatorConfigurations/array_nulls", `{"properties":{"value":"maybe"}}`, "must be one of on,off"},
		{"/coordinatorConfigurations/work_mem", `{"properties":{"value":"4095"}}`, "integer from 4096 to 2097151"},
		{"/coordinatorConfigurations/work_mem", `{"properties":{"value":"8MB"}}`, "integer from 4096"},
		{"/coordinatorConfigurations/work_mem", `{"properties":{"value":"5000x"}}`, "integer from 4096"},
		{"/coordinatorConfigurations/work_mem", `{"properties":{"value":4096}}`, "properties.value is required"},
		{"/nodeConfigurations/backslash_quote", `{"properties":{}}`, "properties.value is required"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, cluster+tc.path+azurePGTestAPIVersion, tc.body), tc.want)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, cluster+"/coordinatorConfigurations/work_mem"+azurePGTestAPIVersion, `{"properties":{"value":"2097151"}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, cluster+"/coordinatorConfigurations/shared_buffers"+azurePGTestAPIVersion, `{"properties":{"value":"1"}}`), http.StatusBadRequest)
	azureResourcesExpectStatus(t, do(http.MethodGet, cluster+"/coordinatorConfigurations/shared_buffers"+azurePGTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodDelete, cluster+"/coordinatorConfigurations/work_mem"+azurePGTestAPIVersion, ""), http.StatusMethodNotAllowed)

	all := providerContractJSONMap(t, do(http.MethodGet, cluster+"/configurations/array_nulls"+azurePGTestAPIVersion, ""))
	groups := all["properties"].(map[string]any)["serverRoleGroupConfigurations"].([]any)
	if len(groups) != 2 || groups[0].(map[string]any)["value"] != "on" || groups[1].(map[string]any)["role"] != "Worker" || groups[1].(map[string]any)["value"] != "off" {
		t.Fatalf("cluster configuration = %#v", all)
	}
	if list := azurePGList(t, do(http.MethodGet, cluster+"/configurations"+azurePGTestAPIVersion, "")); len(list) != len(azurePGParameters) {
		t.Fatalf("configurations = %d", len(list))
	}
	for server, want := range map[string]string{"pgparams-c": "2097151", "pgparams-w1": "115712"} {
		got := providerContractJSONMap(t, do(http.MethodGet, cluster+"/servers/"+server+"/configurations/work_mem"+azurePGTestAPIVersion, ""))
		if got["properties"].(map[string]any)["value"] != want {
			t.Fatalf("%s work_mem = %#v", server, got)
		}
	}
	if list := azurePGList(t, do(http.MethodGet, cluster+"/servers/pgparams-w0/configurations"+azurePGTestAPIVersion, "")); len(list) != len(azurePGParameters) {
		t.Fatalf("server configurations = %d", len(list))
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, cluster+"/servers/pgparams-w7/configurations"+azurePGTestAPIVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodGet, azurePGClusterPath(sub, "rg-pg", "missing")+"/servers/missing-c/configurations"+azurePGTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzurePGRolesFirewallAndConnections(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-pg")
	cluster := azurePGClusterPath(sub, "rg-pg", "pgaccess")
	azureResourcesExpectStatus(t, do(http.MethodPut, cluster+azurePGTestAPIVersion, azurePGSingleNode), http.StatusCreated)

	rule := cluster + "/firewallRules/rule1" + azurePGTestAPIVersion
	azureResourcesExpectStatus(t, do(http.MethodPut, rule, `{"properties":{"startIpAddress":"0.0.0.0","endIpAddress":"255.255.255.255"}}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, rule, `{"properties":{"startIpAddress":"10.0.0.2","endIpAddress":"10.0.0.1"}}`), "must not be greater")
	azureSentinelExpectRejected(t, do(http.MethodPut, rule, `{"properties":{"startIpAddress":"::1","endIpAddress":"::1"}}`), "IPv4")
	if rules := azurePGList(t, do(http.MethodGet, cluster+"/firewallRules"+azurePGTestAPIVersion, "")); len(rules) != 1 {
		t.Fatalf("rules = %#v", rules)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, rule, ""), http.StatusNoContent)

	role := cluster + "/roles/role1" + azurePGTestAPIVersion
	for _, tc := range []struct{ body, want string }{
		{`{"properties":{}}`, "password is required unless"},
		{`{"properties":{"password":"password"}}`, "must contain uppercase"},
		{`{"properties":{"password":"Role1-pass!"}}`, "role name 'role1'"},
		{`{"properties":{"password":"Pa55word!","roleType":"owner"}}`, "roleType must be one of"},
		{`{"properties":{"externalIdentity":{"principalType":"user"}}}`, "objectId is required"},
		{`{"properties":{"externalIdentity":{"objectId":"oid","principalType":"app"}}}`, "principalType must be"},
		{`{"properties":{"externalIdentity":{"objectId":"oid","principalType":"user"}}}`, "doesn't have Microsoft Entra ID authentication enabled"},
	} {
		azureSentinelExpectRejected(t, do(http.MethodPut, role, tc.body), tc.want)
	}
	resp := do(http.MethodPut, role, `{"properties":{"password":"`+azurePGTestPassword+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if p := providerContractJSONMap(t, resp)["properties"].(map[string]any); p["roleType"] != "user" || p["password"] != nil {
		t.Fatalf("role = %#v", p)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, role, ""), http.StatusNoContent)

	// Entra ID roles need an Entra-enabled cluster, and not groups.
	aad := azurePGClusterPath(sub, "rg-pg", "pgaad")
	azureResourcesExpectStatus(t, do(http.MethodPut, aad+azurePGTestAPIVersion, `{"location":"westus","properties":{"authConfig":{"activeDirectoryAuth":"enabled","passwordAuth":"disabled"},
		"coordinatorServerEdition":"GeneralPurpose","coordinatorStorageQuotaInMb":131072,"coordinatorVCores":4,"nodeCount":0}}`), http.StatusCreated)
	azureSentinelExpectRejected(t, do(http.MethodPut, aad+"/roles/r"+azurePGTestAPIVersion, `{"properties":{"password":"Pa55word!"}}`), "doesn't allow password authentication")
	azureSentinelExpectRejected(t, do(http.MethodPut, aad+"/roles/r"+azurePGTestAPIVersion, `{"properties":{"externalIdentity":{"objectId":"oid","principalType":"group"}}}`), "groups aren't supported")
	azureResourcesExpectStatus(t, do(http.MethodPut, aad+"/roles/r"+azurePGTestAPIVersion, `{"properties":{"roleType":"admin","externalIdentity":{"objectId":"oid","principalType":"servicePrincipal","tenantId":"tid"}}}`), http.StatusCreated)

	connection := cluster + "/privateEndpointConnections/pec1" + azurePGTestAPIVersion
	azureSentinelExpectRejected(t, do(http.MethodPut, connection, `{"properties":{}}`), "privateLinkServiceConnectionState is required")
	azureResourcesExpectStatus(t, do(http.MethodPut, connection, `{"properties":{"privateLinkServiceConnectionState":{"status":"Approved","description":"Approved by admin"}}}`), http.StatusCreated)
	listed := providerContractJSONMap(t, do(http.MethodGet, cluster+azurePGTestAPIVersion, ""))["properties"].(map[string]any)["privateEndpointConnections"].([]any)
	if len(listed) != 1 || listed[0].(map[string]any)["properties"].(map[string]any)["groupIds"].([]any)[0] != "coordinator" {
		t.Fatalf("cluster connections = %#v", listed)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, connection, ""), http.StatusNoContent)
	if listed := providerContractJSONMap(t, do(http.MethodGet, cluster+azurePGTestAPIVersion, ""))["properties"].(map[string]any)["privateEndpointConnections"].([]any); len(listed) != 0 {
		t.Fatalf("connections after delete = %#v", listed)
	}
}
