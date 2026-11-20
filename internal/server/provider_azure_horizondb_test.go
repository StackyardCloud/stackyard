package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

const azureHZTestAPI = "?api-version=2026-10-01-preview"

type azureHZTest struct {
	t    *testing.T
	sub  string // /azure/subscriptions/{id}
	base string // the resource group's Microsoft.HorizonDb path
	arm  string // base without /azure, as resource IDs have it
	do   func(method, path, body string) *http.Response
}

func newAzureHZTest(t *testing.T) *azureHZTest {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-hz")
	base := sub + "/resourceGroups/rg-hz/providers/Microsoft.HorizonDb/"
	return &azureHZTest{t: t, sub: sub, base: base, arm: strings.TrimPrefix(base, "/azure"), do: do}
}

func (h *azureHZTest) call(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	text := ""
	if body != nil {
		raw, _ := json.Marshal(body)
		text = string(raw)
	}
	sep := azureHZTestAPI
	if strings.Contains(path, "?") {
		sep = "&" + strings.TrimPrefix(azureHZTestAPI, "?")
	}
	resp := h.do(method, path+sep, text)
	raw := mustBody(h.t, resp)
	if resp.StatusCode != want {
		h.t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func (h *azureHZTest) props(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	return azureARMObject(h.call(method, path, body, want)["properties"])
}

// fails checks that a request fails with status and a message naming text.
func (h *azureHZTest) fails(method, path string, body any, status int, text string) map[string]any {
	h.t.Helper()
	out := h.call(method, path, body, status)
	if msg := azureARMString(azureARMObject(out["error"])["message"]); !strings.Contains(msg, text) {
		h.t.Fatalf("%s %s error = %q, want it to mention %q", method, path, msg, text)
	}
	return out
}

func (h *azureHZTest) rejects(method, path string, body any, text string) {
	h.t.Helper()
	h.fails(method, path, body, http.StatusBadRequest, text)
}

func (h *azureHZTest) list(path string) []any {
	h.t.Helper()
	return azureARMSlice(h.call(http.MethodGet, path, nil, http.StatusOK)["value"])
}

func azureHZCluster(props map[string]any) map[string]any {
	out := map[string]any{"postgreSqlVersion": "17", "backup": map[string]any{"geoRedundancy": "Disabled"}}
	for k, v := range props {
		out[k] = v
		if v == nil {
			delete(out, k)
		}
	}
	return map[string]any{"location": "eastus", "properties": out}
}

func azureHZPool(nodes float64, mode string) map[string]any {
	return map[string]any{"properties": map[string]any{
		"computeModel": map[string]any{"type": "Provisioned", "computeProcessor": "AMD", "computeGeneration": 3.0, "vCores": 2.0},
		"nodeCount":    nodes, "highAvailability": map[string]any{"mode": mode},
	}}
}

// cluster creates cluster c1 with pool p1 of three nodes.
func (h *azureHZTest) cluster() string {
	h.t.Helper()
	h.call(http.MethodPut, h.base+"clusters/c1", azureHZCluster(nil), http.StatusCreated)
	h.call(http.MethodPut, h.base+"clusters/c1/pools/p1", azureHZPool(3, "AnyZone"), http.StatusCreated)
	return h.base + "clusters/c1"
}

func TestAzureHorizonDBClusterLifecycle(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	props := h.props(http.MethodPut, h.base+"clusters/c1", azureHZCluster(nil), http.StatusCreated)
	if props["creationMode"] != "Create" || props["state"] != "Ready" || props["provisioningState"] != "Succeeded" ||
		azureARMObject(props["network"])["publicAccess"] != "Disabled" || azureARMObject(props["backup"])["retentionDays"] != 7.0 ||
		azureARMObject(props["dataEncryption"])["type"] != "SystemManaged" || len(azureARMObject(props["readWritePool"])) != 0 {
		t.Fatalf("defaults = %v", props)
	}
	if azureARMObject(props["backup"])["earliestRestorePointUtc"] == nil {
		t.Fatalf("backup = %v", props["backup"])
	}
	// A replacement keeps creation-time properties it omits.
	replaced := h.props(http.MethodPut, h.base+"clusters/c1", map[string]any{"location": "eastus", "properties": map[string]any{"aiModelManagement": "Enabled"}}, http.StatusOK)
	if replaced["postgreSqlVersion"] != "17" || azureARMObject(replaced["backup"])["geoRedundancy"] != "Disabled" || replaced["aiModelManagement"] != "Enabled" {
		t.Fatalf("replaced = %v", replaced)
	}
	h.rejects(http.MethodPut, h.base+"clusters/c1", azureHZCluster(map[string]any{"postgreSqlVersion": "18"}), "postgreSqlVersion can't be changed")
	h.rejects(http.MethodPut, h.base+"clusters/c1", azureHZCluster(map[string]any{"backup": map[string]any{"geoRedundancy": "Enabled"}}), "geoRedundancy can't be changed")

	patched := h.props(http.MethodPatch, h.base+"clusters/c1", map[string]any{"properties": map[string]any{"network": map[string]any{"publicAccess": "Enabled"}, "backup": map[string]any{"retentionDays": 14.0}}}, http.StatusOK)
	if azureARMObject(patched["network"])["publicAccess"] != "Enabled" || azureARMObject(patched["backup"])["retentionDays"] != 14.0 || patched["postgreSqlVersion"] != "17" {
		t.Fatalf("patched = %v", patched)
	}
	h.rejects(http.MethodPatch, h.base+"clusters/c1", map[string]any{"properties": map[string]any{"backup": map[string]any{"geoRedundancy": "Enabled"}}}, "properties.backup.geoRedundancy can't be updated")
	h.rejects(http.MethodPatch, h.base+"clusters/c1", map[string]any{"properties": map[string]any{"postgreSqlVersion": "18"}}, "properties.postgreSqlVersion can't be updated")

	if got := h.list(h.base + "clusters"); len(got) != 1 {
		t.Fatalf("list = %v", got)
	}
	if got := h.list(h.sub + "/providers/Microsoft.HorizonDb/clusters"); len(got) != 1 {
		t.Fatalf("subscription list = %v", got)
	}
	auths := h.list(h.base + "clusters/c1/authentications")
	if len(auths) != 2 {
		t.Fatalf("authentications = %v", auths)
	}
	h.call(http.MethodDelete, h.base+"clusters/c1", nil, http.StatusNoContent)
	h.call(http.MethodGet, h.base+"clusters/c1", nil, http.StatusNotFound)
	h.call(http.MethodGet, h.base+"clusters/c1/authentications/passwordMethod", nil, http.StatusNotFound)
}

func TestAzureHorizonDBClusterValidation(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	c := h.base + "clusters/"
	h.call(http.MethodPut, c+"Bad_Name", azureHZCluster(nil), http.StatusBadRequest)
	cases := []struct {
		props map[string]any
		text  string
	}{
		{map[string]any{"postgreSqlVersion": nil}, "postgreSqlVersion is required"},
		{map[string]any{"postgreSqlVersion": "16"}, "postgreSqlVersion must be one of 17, 18"},
		{map[string]any{"backup": map[string]any{}}, "geoRedundancy is required"},
		{map[string]any{"backup": map[string]any{"geoRedundancy": "Disabled", "retentionDays": 36.0}}, "retentionDays must be a whole number from 1 to 35"},
		{map[string]any{"network": map[string]any{"publicAccess": "Maybe"}}, "publicAccess must be one of"},
		{map[string]any{"pointInTimeUtc": "2026-01-01T00:00:00Z"}, "only allowed when properties.creationMode is PointInTimeRestore"},
		{map[string]any{"dataEncryption": map[string]any{"primaryKeyUri": "https://v.vault.azure.net/keys/k"}}, "dataEncryption.type is required"},
		{map[string]any{"dataEncryption": map[string]any{"type": "AzureKeyVault", "primaryKeyUri": "https://v.vault.azure.net/keys/k/0123"}}, "versionless"},
		{map[string]any{"dataEncryption": map[string]any{"type": "SystemManaged", "primaryKeyUri": "https://v.vault.azure.net/keys/k"}}, "must be omitted for SystemManaged"},
	}
	for _, tc := range cases {
		h.rejects(http.MethodPut, c+"c1", azureHZCluster(tc.props), tc.text)
	}
	// Customer-managed keys need an assigned identity and no geo-redundancy.
	uai := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-hz/providers/Microsoft.ManagedIdentity/userAssignedIdentities/"
	cmk := func(identity string, geo string) map[string]any {
		body := azureHZCluster(map[string]any{
			"backup":         map[string]any{"geoRedundancy": geo},
			"dataEncryption": map[string]any{"type": "AzureKeyVault", "primaryKeyUri": "https://v.vault.azure.net/keys/k", "primaryUserAssignedIdentityId": uai + "key"},
		})
		body["identity"] = map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{uai + identity: map[string]any{}}}
		return body
	}
	h.rejects(http.MethodPut, c+"c1", cmk("other", "Disabled"), "must also be assigned")
	h.rejects(http.MethodPut, c+"c1", cmk("key", "Enabled"), "requires properties.backup.geoRedundancy Disabled")
	out := h.call(http.MethodPut, c+"c1", cmk("key", "Disabled"), http.StatusCreated)
	enc := azureARMObject(azureARMObject(out["properties"])["dataEncryption"])
	ids := azureARMObject(azureARMObject(out["identity"])["userAssignedIdentities"])
	if enc["primaryEncryptionKeyStatus"] != "Valid" || enc["primaryEncryptionKeyCurrentVersion"] == nil || azureARMObject(ids[uai+"key"])["principalId"] == nil {
		t.Fatalf("cmk cluster = %v", out)
	}
	h.rejects(http.MethodPatch, c+"c1", map[string]any{"properties": map[string]any{"dataEncryption": map[string]any{"type": "SystemManaged"}}}, "type can't be changed")
	h.rejects(http.MethodPatch, c+"c1", map[string]any{"identity": map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{uai + "k2": map[string]any{}}},
		"properties": map[string]any{"dataEncryption": map[string]any{"primaryKeyUri": "https://v.vault.azure.net/keys/k2", "primaryUserAssignedIdentityId": uai + "k2"}}}, "separate updates")
	// A PUT that omits identity keeps it.
	kept := h.call(http.MethodPut, c+"c1", map[string]any{"location": "eastus", "properties": map[string]any{}}, http.StatusOK)
	if len(azureARMObject(azureARMObject(kept["identity"])["userAssignedIdentities"])) != 1 {
		t.Fatalf("identity after PUT = %v", kept["identity"])
	}
}

func TestAzureHorizonDBPointInTimeRestore(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	h.call(http.MethodPut, h.base+"clusters/src", azureHZCluster(map[string]any{"postgreSqlVersion": "18", "backup": map[string]any{"geoRedundancy": "Enabled"}}), http.StatusCreated)
	source := h.arm + "clusters/src"
	now := time.Now().UTC().Format(time.RFC3339)
	restore := func(location string, props map[string]any) map[string]any {
		out := map[string]any{"creationMode": "PointInTimeRestore", "creationSourceClusterResourceId": source, "pointInTimeUtc": now, "backup": map[string]any{"geoRedundancy": "Disabled"}}
		for k, v := range props {
			out[k] = v
			if v == nil {
				delete(out, k)
			}
		}
		return map[string]any{"location": location, "properties": out}
	}
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("eastus", map[string]any{"postgreSqlVersion": "18"}), "must be omitted for a point-in-time restore")
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("eastus", map[string]any{"pointInTimeUtc": nil}), "are required for a point-in-time restore")
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("eastus", map[string]any{"pointInTimeUtc": "2020-01-01T00:00:00Z"}), "earliest restore point")
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("eastus", map[string]any{"creationSourceClusterResourceId": h.arm + "clusters/nope"}), "was not found")
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("westus", nil), "creationSourceClusterLocation is required")
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("westus", map[string]any{"creationSourceClusterLocation": "centralus"}), "must be the source cluster's location")

	same := h.props(http.MethodPut, h.base+"clusters/r1", restore("eastus", nil), http.StatusCreated)
	if same["postgreSqlVersion"] != "18" || azureARMObject(same["backup"])["geoRedundancy"] != "Enabled" || same["pointInTimeUtc"] != now {
		t.Fatalf("same-location restore = %v", same)
	}
	cross := h.props(http.MethodPut, h.base+"clusters/r2", restore("westus", map[string]any{"creationSourceClusterLocation": "eastus"}), http.StatusCreated)
	if _, has := cross["pointInTimeUtc"]; has || cross["creationSourceClusterLocation"] != "eastus" {
		t.Fatalf("cross-location restore = %v", cross)
	}
	// Repeating the creation provenance on replacement is accepted.
	h.call(http.MethodPut, h.base+"clusters/r2", restore("westus", map[string]any{"creationSourceClusterLocation": "eastus", "backup": map[string]any{"geoRedundancy": "Enabled"}}), http.StatusOK)
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("eastus", map[string]any{"creationMode": "Create"}), "creationMode can't be changed")
	// The same instant written differently is the same pointInTimeUtc.
	at, _ := time.Parse(time.RFC3339, now)
	h.call(http.MethodPut, h.base+"clusters/r1", restore("eastus", map[string]any{"pointInTimeUtc": at.In(time.FixedZone("EST", -5*3600)).Format(time.RFC3339), "backup": map[string]any{"geoRedundancy": "Enabled"}}), http.StatusOK)
	h.rejects(http.MethodPut, h.base+"clusters/r1", restore("eastus", map[string]any{"pointInTimeUtc": at.Add(-time.Second).Format(time.RFC3339), "backup": map[string]any{"geoRedundancy": "Enabled"}}), "pointInTimeUtc can't be changed")
}

func TestAzureHorizonDBEndpointsAndPrivateLink(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	c := h.base + "clusters/c1"
	h.call(http.MethodPut, c, azureHZCluster(nil), http.StatusCreated)
	if got := h.list(c + "/endpoints"); len(got) != 1 {
		t.Fatalf("endpoints before pool = %v", got)
	}
	conn := map[string]any{"properties": map[string]any{"privateLinkServiceConnectionState": map[string]any{"status": "Approved", "description": "ok"}}}
	out := h.fails(http.MethodPut, c+"/privateEndpointConnections/pe1", conn, http.StatusConflict, "has a pool")
	if azureARMObject(out["error"])["code"] != "PrivateEndpointRequiresPool" {
		t.Fatalf("error = %v", out)
	}
	h.call(http.MethodPut, c+"/pools/p1", azureHZPool(2, "AnyZone"), http.StatusCreated)
	if got := h.props(http.MethodPut, c+"/privateEndpointConnections/pe1", conn, http.StatusCreated); azureARMSlice(got["groupIds"])[0] != "c1" {
		t.Fatalf("connection = %v", got)
	}
	ep := h.props(http.MethodGet, c+"/endpoints/p1", nil, http.StatusOK)
	rw := h.props(http.MethodGet, c+"/endpoints/c1", nil, http.StatusOK)
	if ep["scope"] != "Pool" || ep["role"] != "ReadOnly" || ep["associatedResourceId"] != h.arm+"clusters/c1/pools/p1" ||
		rw["role"] != "ReadWrite" || !strings.HasPrefix(azureARMString(rw["fullyQualifiedDomainName"]), "c1.") ||
		azureARMString(ep["fullyQualifiedDomainName"]) != "p1."+azureARMString(rw["fullyQualifiedDomainName"]) {
		t.Fatalf("endpoints = %v %v", ep, rw)
	}
	cluster := h.props(http.MethodGet, c, nil, http.StatusOK)
	if azureARMObject(cluster["readWritePool"])["designatedPoolResourceId"] != h.arm+"clusters/c1/pools/p1" {
		t.Fatalf("readWritePool = %v", cluster["readWritePool"])
	}
	plr := h.props(http.MethodGet, c+"/privateLinkResources/c1", nil, http.StatusOK)
	if plr["groupId"] != "c1" {
		t.Fatalf("private link resource = %v", plr)
	}
	// Deleting the last pool rejects the connections and clears the designation.
	h.call(http.MethodDelete, c+"/pools/p1", nil, http.StatusNoContent)
	state := azureARMObject(h.props(http.MethodGet, c+"/privateEndpointConnections/pe1", nil, http.StatusOK)["privateLinkServiceConnectionState"])
	if state["status"] != "Rejected" || !strings.Contains(azureARMString(state["description"]), "last pool") {
		t.Fatalf("connection after last pool = %v", state)
	}
	if got := h.props(http.MethodGet, c, nil, http.StatusOK)["readWritePool"]; len(azureARMObject(got)) != 0 {
		t.Fatalf("readWritePool after delete = %v", got)
	}
	ops := h.call(http.MethodGet, "/azure/providers/Microsoft.HorizonDb/operations", nil, http.StatusOK)
	if len(azureARMSlice(ops["value"])) == 0 {
		t.Fatalf("operations = %v", ops)
	}
}

func TestAzureHorizonDBAuthentication(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	c := h.cluster()
	pw := c + "/authentications/passwordMethod"
	body := func(props map[string]any) map[string]any { return map[string]any{"properties": props} }
	h.rejects(http.MethodPut, pw, body(map[string]any{"state": "Enabled"}), "administratorLogin and administratorLoginPassword are required")
	h.rejects(http.MethodPut, pw, body(map[string]any{"state": "Disabled", "administratorLogin": "admin", "administratorLoginPassword": "Passw0rd!"}), "requires properties.state Enabled")
	h.rejects(http.MethodPut, pw, body(map[string]any{"state": "Enabled", "administratorLogin": "admin", "administratorLoginPassword": "short"}), "8 to 128")
	got := h.props(http.MethodPut, pw, body(map[string]any{"state": "Enabled", "administratorLogin": "admin", "administratorLoginPassword": "Passw0rd!"}), http.StatusOK)
	if got["administratorLogin"] != "admin" || got["administratorLoginPassword"] != nil {
		t.Fatalf("password method = %v", got)
	}
	h.rejects(http.MethodPut, pw, body(map[string]any{"state": "Enabled", "administratorLogin": "other", "administratorLoginPassword": "Passw0rd!"}), "can't be changed")
	h.rejects(http.MethodPatch, pw, body(map[string]any{"administratorLogin": "other"}), "can't be updated")
	h.rejects(http.MethodPatch, pw, body(map[string]any{"state": "Disabled", "administratorLoginPassword": "NewPassw0rd!"}), "can't include")
	h.props(http.MethodPatch, pw, body(map[string]any{"state": "Disabled"}), http.StatusOK)
	h.fails(http.MethodPatch, pw, body(map[string]any{"administratorLoginPassword": "NewPassw0rd!"}), http.StatusConflict, "requires properties.state Enabled")
	if got := h.props(http.MethodPatch, pw, body(map[string]any{"state": "Enabled", "administratorLoginPassword": "NewPassw0rd!"}), http.StatusOK); got["state"] != "Enabled" || got["administratorLogin"] != "admin" {
		t.Fatalf("re-enabled = %v", got)
	}
	entra := c + "/authentications/microsoftEntra"
	if got := h.props(http.MethodPut, entra, body(map[string]any{"state": "Enabled"}), http.StatusOK); got["state"] != "Enabled" {
		t.Fatalf("entra = %v", got)
	}
	h.rejects(http.MethodPut, entra, body(map[string]any{"state": "Enabled", "administratorLogin": "x"}), "can't be updated")
	h.call(http.MethodPut, c+"/authentications/other", body(map[string]any{"state": "Enabled"}), http.StatusBadRequest)
	h.call(http.MethodDelete, entra, nil, http.StatusMethodNotAllowed)

	admin := c + "/microsoftEntraAdministrators/11111111-1111-1111-1111-111111111111"
	h.call(http.MethodPut, c+"/microsoftEntraAdministrators/not-a-guid", body(map[string]any{"principalName": "a", "principalType": "User"}), http.StatusBadRequest)
	h.rejects(http.MethodPut, admin, body(map[string]any{"principalName": "a", "principalType": "Robot"}), "principalType must be one of")
	added := h.props(http.MethodPut, admin, body(map[string]any{"principalName": "DBAs", "principalType": "Group"}), http.StatusCreated)
	if added["objectId"] != "11111111-1111-1111-1111-111111111111" || added["tenantId"] != azureDefaultTenantID {
		t.Fatalf("administrator = %v", added)
	}
	h.call(http.MethodPut, admin, body(map[string]any{"principalName": "DBAs", "principalType": "Group"}), http.StatusOK)
	h.rejects(http.MethodPut, admin, body(map[string]any{"principalName": "Others", "principalType": "Group"}), "principalName can't be changed")
	h.rejects(http.MethodPut, admin, body(map[string]any{"principalName": "DBAs", "principalType": "Group", "tenantId": "22222222-2222-2222-2222-222222222222"}), "tenantId can't be changed")
	// Tenant IDs compare without case.
	other := c + "/microsoftEntraAdministrators/33333333-3333-3333-3333-333333333333"
	h.call(http.MethodPut, other, body(map[string]any{"principalName": "Ops", "principalType": "Group", "tenantId": "abcdefab-cdef-abcd-efab-cdefabcdefab"}), http.StatusCreated)
	if kept := h.props(http.MethodPut, other, body(map[string]any{"principalName": "Ops", "principalType": "Group", "tenantId": "ABCDEFAB-CDEF-ABCD-EFAB-CDEFABCDEFAB"}), http.StatusOK); kept["tenantId"] != "abcdefab-cdef-abcd-efab-cdefabcdefab" {
		t.Fatalf("tenant = %v", kept["tenantId"])
	}
	h.call(http.MethodDelete, admin, nil, http.StatusNoContent)
}
