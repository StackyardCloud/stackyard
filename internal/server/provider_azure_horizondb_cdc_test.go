package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureHZTestIdentity = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-hz/providers/Microsoft.ManagedIdentity/userAssignedIdentities/cdc"

// newAzureHZCDCTest is newAzureHZTest whose requests send *ifMatch as the
// If-Match header when it's set.
func newAzureHZCDCTest(t *testing.T) (*azureHZTest, *string) {
	t.Helper()
	ts := newAzureResourcesTestServer(t)
	ifMatch := new(string)
	do := func(method, path, body string) *http.Response {
		headers := azureResourcesTestHeaders()
		if *ifMatch != "" {
			headers["If-Match"] = *ifMatch
		}
		var payload []byte
		if body != "" {
			payload = []byte(body)
		}
		return providerContractRequest(t, ts, method, path, payload, headers)
	}
	sub := "/azure/subscriptions/" + azureResourcesTestSubscription
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourcegroups/rg-hz?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	base := sub + "/resourceGroups/rg-hz/providers/Microsoft.HorizonDb/"
	return &azureHZTest{t: t, sub: sub, base: base, arm: strings.TrimPrefix(base, "/azure"), do: do}, ifMatch
}

func azureHZDestinationBody(state string) map[string]any {
	return map[string]any{"properties": map[string]any{
		"desiredState": state,
		"destination":  map[string]any{"destinationType": "Fabric", "protocol": "Native", "workspaceId": "ws-1"},
	}}
}

func TestAzureHorizonDBChangeDataCapture(t *testing.T) {
	t.Parallel()
	h, ifMatch := newAzureHZCDCTest(t)
	cluster := h.base + "clusters/c1"
	h.call(http.MethodPut, cluster, azureHZCluster(map[string]any{}), http.StatusCreated)
	h.call(http.MethodPatch, cluster, map[string]any{"identity": map[string]any{"type": "UserAssigned", "userAssignedIdentities": map[string]any{azureHZTestIdentity: map[string]any{}}}}, http.StatusOK)
	capability, dests, configs := cluster+"/changeDataCapture/default", cluster+"/changeDataCaptureDestinations/", cluster+"/changeDataCaptureConfigurations/"

	// Destinations and configurations need the capability.
	h.fails(http.MethodPut, dests+"d1", azureHZDestinationBody("Enabled"), http.StatusConflict, "changeDataCapture/default")
	h.rejects(http.MethodPut, cluster+"/changeDataCapture/other", map[string]any{"properties": map[string]any{"desiredState": "Enabled"}}, "must be default")
	h.rejects(http.MethodPut, capability, map[string]any{"properties": map[string]any{}}, "desiredState")
	got := h.call(http.MethodPut, capability, map[string]any{"properties": map[string]any{"desiredState": "Enabled", "destinationCount": 9.0}}, http.StatusCreated)
	props := azureARMObject(got["properties"])
	if props["effectiveState"] != "Enabled" || props["destinationCount"] != 0.0 || azureARMObject(props["extension"])["state"] != "Ready" ||
		azureARMObject(props["prerequisites"])["readWriteEndpointState"] != "Unavailable" || got["etag"] == nil {
		t.Fatalf("capability = %v", got)
	}
	if list := h.list(cluster + "/changeDataCapture"); len(list) != 1 {
		t.Fatalf("capability list = %v", list)
	}
	h.rejects(http.MethodPatch, capability, map[string]any{"properties": map[string]any{"effectiveState": "Enabled"}}, "effectiveState can't be updated")

	// A destination waits for an active identity; its settings are fixed.
	h.rejects(http.MethodPut, dests+"d1", map[string]any{"properties": map[string]any{"desiredState": "Enabled", "destination": map[string]any{"destinationType": "Kafka", "protocol": "Native", "workspaceId": "w"}}}, "destinationType")
	forged := azureHZDestinationBody("Enabled")
	azureARMObject(forged["properties"])["activeIdentityAssignmentResourceId"] = strings.TrimPrefix(dests, "/azure") + "d1/identityAssignments/x"
	dest := h.props(http.MethodPut, dests+"d1", forged, http.StatusCreated)
	if dest["effectiveState"] != "WaitingForIdentity" || dest["activeIdentityAssignmentResourceId"] != nil {
		t.Fatalf("destination = %v", dest)
	}
	changed := azureHZDestinationBody("Enabled")
	azureARMObject(azureARMObject(changed["properties"])["destination"])["workspaceId"] = "ws-2"
	h.rejects(http.MethodPut, dests+"d1", changed, "delete and recreate")
	if c := h.props(http.MethodGet, capability, nil, http.StatusOK); c["destinationCount"] != 1.0 {
		t.Fatalf("capability after destination = %v", c)
	}

	// A configuration needs an enabled destination.
	config := map[string]any{"properties": map[string]any{"desiredState": "Enabled", "databaseName": "app", "usageProfile": "Mirroring", "destinationResourceId": strings.TrimPrefix(dests, "/azure") + "d1"}}
	h.rejects(http.MethodPut, configs+"m1", config, "isn't enabled")

	// Identity assignments must name an identity attached to the cluster.
	assignments := dests + "d1/identityAssignments/"
	other := strings.Replace(azureHZTestIdentity, "/cdc", "/other", 1)
	h.rejects(http.MethodPut, assignments+"a1", map[string]any{"properties": map[string]any{"identityResourceId": other}}, "must exactly match")
	assignment := h.props(http.MethodPut, assignments+"a1", map[string]any{"properties": map[string]any{"identityResourceId": azureHZTestIdentity, "activationState": "Active"}}, http.StatusCreated)
	if assignment["activationState"] != "Inactive" || assignment["principalId"] == nil || azureARMObject(assignment["validation"])["state"] != "Succeeded" {
		t.Fatalf("assignment = %v", assignment)
	}
	h.call(http.MethodPut, assignments+"a1", map[string]any{"properties": map[string]any{"identityResourceId": strings.ToUpper(azureHZTestIdentity)}}, http.StatusOK)
	h.rejects(http.MethodPut, assignments+"a1", map[string]any{"properties": map[string]any{"identityResourceId": other}}, "can't be changed")
	h.call(http.MethodPut, assignments+"a2", map[string]any{"properties": map[string]any{"identityResourceId": azureHZTestIdentity}}, http.StatusCreated)

	// activateIdentity requires the destination's current etag.
	activate := dests + "d1/activateIdentity"
	a1 := strings.TrimPrefix(assignments, "/azure") + "a1"
	h.call(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": a1}, http.StatusBadRequest)
	*ifMatch = `"stale"`
	h.call(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": a1}, http.StatusPreconditionFailed)
	before := azureARMString(h.call(http.MethodGet, dests+"d1", nil, http.StatusOK)["etag"])
	*ifMatch = "W/" + before
	h.rejects(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": strings.TrimPrefix(dests, "/azure") + "d1"}, "under this destination")
	h.call(http.MethodPut, dests+"d2", azureHZDestinationBody("Disabled"), http.StatusCreated)
	h.call(http.MethodPut, dests+"d2/identityAssignments/b1", map[string]any{"properties": map[string]any{"identityResourceId": azureHZTestIdentity}}, http.StatusCreated)
	h.rejects(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": strings.TrimPrefix(dests, "/azure") + "d2/identityAssignments/b1"}, "under this destination")
	result := h.call(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": a1}, http.StatusOK)
	if result["status"] != "Succeeded" || result["previousIdentityAssignmentResourceId"] != nil {
		t.Fatalf("activate = %v", result)
	}
	if after := h.call(http.MethodGet, dests+"d1", nil, http.StatusOK)["etag"]; after == before {
		t.Fatalf("etag unchanged by activation: %v", after)
	}
	h.call(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": a1}, http.StatusPreconditionFailed) // the old etag is now stale
	*ifMatch = "*"
	if again := h.call(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": a1}, http.StatusOK); again["previousIdentityAssignmentResourceId"] != nil {
		t.Fatalf("repeated activation = %v", again)
	}
	if d := h.props(http.MethodGet, dests+"d1", nil, http.StatusOK); d["effectiveState"] != "Enabled" || !strings.EqualFold(azureARMString(d["activeIdentityAssignmentResourceId"]), a1) {
		t.Fatalf("activated destination = %v", d)
	}
	*ifMatch = ""
	if d := h.props(http.MethodPatch, dests+"d1", map[string]any{"properties": map[string]any{"desiredState": "Enabled"}}, http.StatusOK); d["effectiveState"] != "Enabled" {
		t.Fatalf("patched destination = %v", d)
	}
	if d := h.props(http.MethodPut, dests+"d1", azureHZDestinationBody("Enabled"), http.StatusOK); !strings.EqualFold(azureARMString(d["activeIdentityAssignmentResourceId"]), a1) || d["effectiveState"] != "Enabled" {
		t.Fatalf("replaced destination = %v", d)
	}
	if a := h.props(http.MethodPut, assignments+"a1", map[string]any{"properties": map[string]any{"identityResourceId": azureHZTestIdentity}}, http.StatusOK); a["activationState"] != "Active" {
		t.Fatalf("replaced assignment = %v", a)
	}
	*ifMatch = "*"
	h.call(http.MethodPost, dests+"d2/activateIdentity", map[string]any{"identityAssignmentResourceId": strings.TrimPrefix(dests, "/azure") + "d2/identityAssignments/b1"}, http.StatusOK)
	*ifMatch = ""
	if d := h.props(http.MethodGet, dests+"d2", nil, http.StatusOK); d["effectiveState"] != "Disabled" || d["activeIdentityAssignmentResourceId"] == nil {
		t.Fatalf("disabled destination after activation = %v", d)
	}
	h.fails(http.MethodDelete, assignments+"a1", nil, http.StatusConflict, "is active")
	*ifMatch = "*"
	a2 := strings.TrimPrefix(assignments, "/azure") + "a2"
	if switched := h.call(http.MethodPost, activate, map[string]any{"identityAssignmentResourceId": a2}, http.StatusOK); !strings.EqualFold(azureARMString(switched["previousIdentityAssignmentResourceId"]), a1) {
		t.Fatalf("switch = %v", switched)
	}
	*ifMatch = ""
	if old := h.props(http.MethodGet, assignments+"a1", nil, http.StatusOK); old["activationState"] != "Inactive" {
		t.Fatalf("previous assignment = %v", old)
	}
	// The cluster can't drop an identity an assignment uses.
	h.rejects(http.MethodPatch, cluster, map[string]any{"identity": map[string]any{"type": "None"}}, "must keep")

	wrong := map[string]any{"properties": map[string]any{"desiredState": "Enabled", "databaseName": "app", "usageProfile": "Mirroring", "destinationResourceId": strings.TrimPrefix(capability, "/azure")}}
	h.rejects(http.MethodPut, configs+"m1", wrong, "under the same cluster")
	// Now the configuration is accepted; its enrollment is fixed.
	created := h.props(http.MethodPut, configs+"m1", config, http.StatusCreated)
	if created["logicalStreamId"] == nil || azureARMObject(created["replication"])["state"] != "NotConnected" || azureARMObject(created["topology"])["state"] != "Active" {
		t.Fatalf("configuration = %v", created)
	}
	azureARMObject(config["properties"])["destinationResourceId"] = strings.ToUpper(azureARMString(azureARMObject(config["properties"])["destinationResourceId"]))
	h.call(http.MethodPut, configs+"m1", config, http.StatusOK)
	azureARMObject(config["properties"])["databaseName"] = "other"
	h.rejects(http.MethodPut, configs+"m1", config, "databaseName can't be changed")
	if patched := h.props(http.MethodPatch, configs+"m1", map[string]any{"properties": map[string]any{"desiredState": "Disabled"}}, http.StatusOK); patched["effectiveState"] != "Disabled" || patched["logicalStreamId"] != created["logicalStreamId"] {
		t.Fatalf("patched configuration = %v", patched)
	}
	if d := h.props(http.MethodGet, dests+"d1", nil, http.StatusOK); d["configurationCount"] != 1.0 {
		t.Fatalf("destination count = %v", d)
	}

	// Deletes run in dependency order.
	h.fails(http.MethodDelete, capability, nil, http.StatusConflict, "destinations and configurations")
	h.fails(http.MethodDelete, dests+"d1", nil, http.StatusConflict, "first")
	h.call(http.MethodDelete, configs+"m1", nil, http.StatusNoContent)
	h.call(http.MethodDelete, dests+"d1", nil, http.StatusNoContent) // a2 is still active
	h.call(http.MethodGet, assignments+"a2", nil, http.StatusNotFound)
	if c := h.props(http.MethodGet, capability, nil, http.StatusOK); c["configurationCount"] != 0.0 || c["destinationCount"] != 1.0 {
		t.Fatalf("capability counts = %v", c)
	}
}

func TestAzureHorizonDBCDCETagChanges(t *testing.T) {
	t.Parallel()
	h, _ := newAzureHZCDCTest(t)
	cluster := h.base + "clusters/c1"
	h.call(http.MethodPut, cluster, azureHZCluster(map[string]any{}), http.StatusCreated)
	capability := cluster + "/changeDataCapture/default"
	before := h.call(http.MethodPut, capability, map[string]any{"properties": map[string]any{"desiredState": "Disabled"}}, http.StatusCreated)
	if again := h.call(http.MethodGet, capability, nil, http.StatusOK); again["etag"] != before["etag"] {
		t.Fatalf("etag changed on read: %v then %v", before["etag"], again["etag"])
	}
	h.call(http.MethodPatch, cluster, map[string]any{"tags": map[string]any{"env": "dev"}}, http.StatusOK)
	if again := h.call(http.MethodGet, capability, nil, http.StatusOK); again["etag"] != before["etag"] {
		t.Fatalf("etag changed by an unrelated cluster PATCH: %v then %v", before["etag"], again["etag"])
	}
	after := h.call(http.MethodPatch, capability, map[string]any{"properties": map[string]any{"desiredState": "Enabled", "autoUpgradeExtension": true}}, http.StatusOK)
	if after["etag"] == before["etag"] || azureARMObject(after["properties"])["effectiveState"] != "Enabled" {
		t.Fatalf("after = %v, before etag %v", after, before["etag"])
	}
	// A pool makes the read-write endpoint ready.
	h.call(http.MethodPut, cluster+"/pools/p1", azureHZPool(2, "AnyZone"), http.StatusCreated)
	if c := h.props(http.MethodGet, capability, nil, http.StatusOK); azureARMObject(c["prerequisites"])["readWriteEndpointState"] != "Ready" {
		t.Fatalf("prerequisites = %v", c["prerequisites"])
	}
}
