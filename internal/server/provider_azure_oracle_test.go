package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const azureOraTestAPI = "?api-version=2026-06-01"

type azureOraTest struct {
	t    *testing.T
	sub  string // /azure/subscriptions/{id}
	base string // the resource group's Oracle.Database path
	arm  string // base without /azure, as resource IDs have it
	do   func(method, path, body string) *http.Response
}

func newAzureOraTest(t *testing.T) *azureOraTest {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-ora")
	base := sub + "/resourceGroups/rg-ora/providers/Oracle.Database/"
	return &azureOraTest{t: t, sub: sub, base: base, arm: strings.TrimPrefix(base, "/azure"), do: do}
}

func (h *azureOraTest) call(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	text := ""
	if body != nil {
		raw, _ := json.Marshal(body)
		text = string(raw)
	}
	resp := h.do(method, path+azureOraTestAPI, text)
	raw := mustBody(h.t, resp)
	if resp.StatusCode != want {
		h.t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func (h *azureOraTest) props(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	return azureARMObject(h.call(method, path, body, want)["properties"])
}

// fails checks that a request fails with status naming text.
func (h *azureOraTest) fails(method, path string, body any, status int, text string) {
	h.t.Helper()
	out := h.call(method, path, body, status)
	if msg := azureARMString(azureARMObject(out["error"])["message"]); !strings.Contains(msg, text) {
		h.t.Fatalf("%s %s error = %q, want it to mention %q", method, path, msg, text)
	}
}

func (h *azureOraTest) rejects(method, path string, body any, text string) {
	h.t.Helper()
	h.fails(method, path, body, http.StatusBadRequest, text)
}

// subscribe purchases the Oracle offer for the test subscription.
func (h *azureOraTest) subscribe() {
	h.t.Helper()
	h.call(http.MethodPut, h.sub+"/providers/Oracle.Database/oracleSubscriptions/default", map[string]any{
		"plan": map[string]any{"name": "pay-as-you-go", "product": "oracle-db-azure", "publisher": "oracle"}, "properties": map[string]any{"termUnit": "P1Y"},
	}, http.StatusCreated)
}

func azureOraInfra(props map[string]any) map[string]any {
	out := map[string]any{"displayName": "infra 1", "shape": "Exadata.X9M", "computeCount": 2.0, "storageCount": 3.0}
	for k, v := range props {
		out[k] = v
	}
	return map[string]any{"location": "eastus", "zones": []any{"1"}, "properties": out}
}

func (h *azureOraTest) cluster(props map[string]any) map[string]any {
	vnet := strings.TrimPrefix(h.sub, "/azure") + "/resourceGroups/rg-ora/providers/Microsoft.Network/virtualNetworks/vnet"
	out := map[string]any{
		"cloudExadataInfrastructureId": h.arm + "cloudExadataInfrastructures/infra", "displayName": "cluster 1", "giVersion": "19.0.0.0",
		"hostname": "host", "cpuCoreCount": 8.0, "memorySizeInGbs": 120.0, "sshPublicKeys": []any{"ssh-rsa AAAAB3Nza test@host"},
		"vnetId": vnet, "subnetId": vnet + "/subnets/oracle",
	}
	for k, v := range props {
		out[k] = v
	}
	return map[string]any{"location": "eastus", "properties": out}
}

func TestAzureOracleSubscription(t *testing.T) {
	t.Parallel()
	h := newAzureOraTest(t)
	sub := h.sub + "/providers/Oracle.Database/oracleSubscriptions/"
	// Nothing provisions before the offer is purchased.
	h.rejects(http.MethodPut, h.base+"cloudExadataInfrastructures/infra", azureOraInfra(nil), "no Oracle Database@Azure offer")
	h.rejects(http.MethodPut, sub+"other", map[string]any{}, "named default")
	h.rejects(http.MethodPut, sub+"default", map[string]any{"properties": map[string]any{"termUnit": "1 year"}}, "termUnit")
	h.rejects(http.MethodPut, sub+"default", map[string]any{"properties": map[string]any{"intent": "Keep"}}, "intent")
	h.rejects(http.MethodPut, sub+"default", map[string]any{"plan": map[string]any{"sku": "x"}}, "plan.sku")
	h.subscribe()
	got := h.call(http.MethodGet, sub+"default", nil, http.StatusOK)
	props := azureARMObject(got["properties"])
	if azureARMObject(got["plan"])["product"] != "oracle-db-azure" || props["cloudAccountState"] != "Available" || props["saasSubscriptionId"] == nil {
		t.Fatalf("oracle subscription = %#v", got)
	}
	if items := h.call(http.MethodGet, sub, nil, http.StatusOK)["value"].([]any); len(items) != 1 {
		t.Fatalf("list = %d", len(items))
	}
	if patched := h.call(http.MethodPatch, sub+"default", map[string]any{"plan": map[string]any{"version": "2"}, "properties": map[string]any{"intent": "Reset"}}, http.StatusOK); azureARMObject(patched["plan"])["product"] != "oracle-db-azure" ||
		azureARMObject(patched["plan"])["version"] != "2" || azureARMObject(patched["properties"])["intent"] != "Reset" {
		t.Fatalf("patched = %#v", patched)
	}
	h.rejects(http.MethodPatch, sub+"default", map[string]any{"properties": map[string]any{"termUnit": "P3Y"}}, "can't be updated")
	h.rejects(http.MethodPatch, sub+"default", map[string]any{"properties": map[string]any{"azureSubscriptionIds": []any{}}}, "can't be updated")
	if links := h.call(http.MethodPost, sub+"default/listActivationLinks", nil, http.StatusOK); !strings.HasPrefix(azureARMString(links["newCloudAccountActivationLink"]), "https://") {
		t.Fatalf("links = %#v", links)
	}
	if details := h.call(http.MethodPost, sub+"default/listSaasSubscriptionDetails", nil, http.StatusOK); details["offerId"] != "oracle-db-azure" || details["termUnit"] != "P1Y" {
		t.Fatalf("saas = %#v", details)
	}
	if account := h.call(http.MethodPost, sub+"default/listCloudAccountDetails", nil, http.StatusOK); account["cloudAccountName"] == nil {
		t.Fatalf("account = %#v", account)
	}
	h.rejects(http.MethodPost, sub+"default/addAzureSubscriptions", map[string]any{"azureSubscriptionIds": []any{"not-a-guid"}}, "isn't a subscription ID")
	h.rejects(http.MethodPost, sub+"default/addAzureSubscriptions", map[string]any{}, "non-empty")
	other := "11111111-1111-1111-1111-111111111111"
	h.call(http.MethodPost, sub+"default/addAzureSubscriptions", map[string]any{"azureSubscriptionIds": []any{other, strings.ToUpper(other)}}, http.StatusAccepted)
	if ids := azureARMSlice(h.props(http.MethodGet, sub+"default", nil, http.StatusOK)["azureSubscriptionIds"]); len(ids) != 1 || ids[0] != other {
		t.Fatalf("added subscriptions = %#v", ids)
	}
	// The added subscription may provision too.
	otherBase := "/azure/subscriptions/" + other
	h.call(http.MethodPut, otherBase+"/resourcegroups/rg-ora", map[string]any{"location": "eastus"}, http.StatusCreated)
	h.call(http.MethodPut, otherBase+"/resourceGroups/rg-ora/providers/Oracle.Database/cloudExadataInfrastructures/infra", azureOraInfra(nil), http.StatusCreated)
}

func TestAzureOracleCatalogs(t *testing.T) {
	t.Parallel()
	h := newAzureOraTest(t)
	loc := h.sub + "/providers/Oracle.Database/locations/eastus/"
	shapes := h.call(http.MethodGet, loc+"dbSystemShapes", nil, http.StatusOK)["value"].([]any)
	if len(shapes) != 2 || azureARMObject(azureARMObject(shapes[0])["properties"])["shapeName"] != "Exadata.X9M" {
		t.Fatalf("shapes = %#v", shapes)
	}
	if got := h.props(http.MethodGet, loc+"dbSystemShapes/Exadata.X11M", nil, http.StatusOK); got["computeModel"] != "ECPU" {
		t.Fatalf("shape = %#v", got)
	}
	h.call(http.MethodGet, loc+"dbSystemShapes/Exadata.X1M", nil, http.StatusNotFound)
	if gi := h.call(http.MethodGet, loc+"giVersions", nil, http.StatusOK)["value"].([]any); len(gi) != 2 {
		t.Fatalf("gi versions = %#v", gi)
	}
	resp := h.do(http.MethodGet, loc+"giVersions"+azureOraTestAPI+"&shape=ExaDbXS", "")
	if gi := providerContractJSONMap(t, resp)["value"].([]any); len(gi) != 1 || azureARMObject(azureARMObject(gi[0])["properties"])["version"] != "23.0.0.0" {
		t.Fatalf("filtered gi versions = %#v", gi)
	}
	for collection, want := range map[string]int{"systemVersions": 2, "autonomousDatabaseCharacterSets": 3, "autonomousDatabaseNationalCharacterSets": 2, "autonomousDbVersions": 2, "dnsPrivateViews": 1, "dnsPrivateZones": 1} {
		if items := h.call(http.MethodGet, loc+collection, nil, http.StatusOK)["value"].([]any); len(items) != want {
			t.Fatalf("%s = %d items, want %d", collection, len(items), want)
		}
	}
	zone := azureARMObject(h.call(http.MethodGet, loc+"dnsPrivateZones", nil, http.StatusOK)["value"].([]any)[0])
	view := azureARMObject(h.call(http.MethodGet, loc+"dnsPrivateViews", nil, http.StatusOK)["value"].([]any)[0])
	if azureARMObject(zone["properties"])["viewId"] != view["name"] {
		t.Fatalf("zone %#v isn't in view %#v", zone, view)
	}
	h.call(http.MethodGet, loc+"autonomousDatabaseCharacterSets/AL32UTF8", nil, http.StatusOK)
	h.fails(http.MethodPut, loc+"giVersions/19.0.0.0", map[string]any{}, http.StatusMethodNotAllowed, "not allowed")
}

func TestAzureOracleExadataInfrastructure(t *testing.T) {
	t.Parallel()
	h := newAzureOraTest(t)
	h.subscribe()
	infra := h.base + "cloudExadataInfrastructures/infra"
	for name, body := range map[string]map[string]any{
		"shape":        azureOraInfra(map[string]any{"shape": "Exadata.X8"}),
		"compute":      azureOraInfra(map[string]any{"computeCount": 1.0}),
		"storage":      azureOraInfra(map[string]any{"storageCount": 65.0}),
		"display":      azureOraInfra(map[string]any{"displayName": ""}),
		"contacts":     azureOraInfra(map[string]any{"customerContacts": []any{map[string]any{"email": "nope"}}}),
		"hours":        azureOraInfra(map[string]any{"maintenanceWindow": map[string]any{"hoursOfDay": []any{3.0}}}),
		"timeout":      azureOraInfra(map[string]any{"maintenanceWindow": map[string]any{"customActionTimeoutInMins": 10.0}}),
		"day":          azureOraInfra(map[string]any{"maintenanceWindow": map[string]any{"daysOfWeek": []any{map[string]any{"name": "Funday"}}}}),
		"custom hours": azureOraInfra(map[string]any{"maintenanceWindow": map[string]any{"preference": "CustomPreference"}}),
	} {
		t.Run(name, func(t *testing.T) {
			resp := h.do(http.MethodPut, infra+azureOraTestAPI, string(mustJSON(t, body)))
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	noZone := azureOraInfra(nil)
	delete(noZone, "zones")
	h.rejects(http.MethodPut, infra, noZone, "zones")
	created := h.call(http.MethodPut, infra, azureOraInfra(map[string]any{"maintenanceWindow": map[string]any{
		"preference": "CustomPreference", "hoursOfDay": []any{4.0}, "daysOfWeek": []any{map[string]any{"name": "Sunday"}}, "leadTimeInWeeks": 2.0,
	}}), http.StatusCreated)
	p := azureARMObject(created["properties"])
	if created["zones"].([]any)[0] != "1" || p["maxCpuCount"] != 252.0 || p["activatedStorageCount"] != 3.0 || p["additionalStorageCount"] != 0.0 ||
		p["computeModel"] != "OCPU" || p["lifecycleState"] != "Available" || !strings.HasPrefix(azureARMString(p["ocid"]), "ocid1.cloudexadatainfrastructure.") {
		t.Fatalf("infra = %#v", created)
	}
	servers := h.call(http.MethodGet, infra+"/dbServers", nil, http.StatusOK)["value"].([]any)
	if len(servers) != 2 || azureARMObject(azureARMObject(servers[0])["properties"])["maxCpuCount"] != 126.0 {
		t.Fatalf("db servers = %#v", servers)
	}
	h.call(http.MethodGet, infra+"/dbServers/"+azureARMString(azureARMObject(servers[1])["name"]), nil, http.StatusOK)
	h.fails(http.MethodPut, infra+"/dbServers/x", map[string]any{}, http.StatusMethodNotAllowed, "not allowed")

	h.rejects(http.MethodPut, infra, azureOraInfra(map[string]any{"shape": "Exadata.X11M"}), "shape can't be changed")
	h.fails(http.MethodPost, infra+"/addStorageCapacity", nil, http.StatusConflict, "no storage capacity")
	scaled := h.props(http.MethodPatch, infra, map[string]any{"properties": map[string]any{"storageCount": 5.0, "computeCount": 3.0}, "zones": []any{"2"}}, http.StatusOK)
	if scaled["additionalStorageCount"] != 2.0 || scaled["activatedStorageCount"] != 3.0 || scaled["maxCpuCount"] != 378.0 || scaled["ocid"] != p["ocid"] {
		t.Fatalf("scaled = %#v", scaled)
	}
	added := h.call(http.MethodPost, infra+"/addStorageCapacity", nil, http.StatusOK)
	if ap := azureARMObject(added["properties"]); ap["activatedStorageCount"] != 5.0 || ap["additionalStorageCount"] != 0.0 || added["zones"].([]any)[0] != "2" {
		t.Fatalf("after addStorageCapacity = %#v", added)
	}
	h.rejects(http.MethodPatch, infra, map[string]any{"properties": map[string]any{"storageCount": 4.0}}, "already activated")
	h.rejects(http.MethodPatch, infra, map[string]any{"properties": map[string]any{"shape": "Exadata.X11M"}}, "can't be updated")
	h.rejects(http.MethodPatch, infra, map[string]any{"zones": []any{"1", "2"}}, "zones")
	if got := h.props(http.MethodPut, infra, azureOraInfra(map[string]any{"computeCount": 3.0, "storageCount": 5.0}), http.StatusOK); got["ocid"] != p["ocid"] || got["timeCreated"] != p["timeCreated"] {
		t.Fatalf("re-PUT lost server fields: %#v", got)
	}
	h.call(http.MethodDelete, infra, nil, http.StatusOK)
	h.call(http.MethodGet, infra+"/dbServers", nil, http.StatusNotFound)
}

func TestAzureOracleVMClusters(t *testing.T) {
	t.Parallel()
	h := newAzureOraTest(t)
	h.subscribe()
	infra := h.base + "cloudExadataInfrastructures/infra"
	h.call(http.MethodPut, infra, azureOraInfra(map[string]any{"computeCount": 3.0}), http.StatusCreated)
	servers := []string{}
	for _, item := range h.call(http.MethodGet, infra+"/dbServers", nil, http.StatusOK)["value"].([]any) {
		servers = append(servers, azureARMString(azureARMObject(item)["name"]))
	}
	vm := h.base + "cloudVmClusters/"
	for name, c := range map[string]struct {
		props map[string]any
		text  string
	}{
		"infra":      {map[string]any{"cloudExadataInfrastructureId": h.arm + "cloudExadataInfrastructures/missing"}, "was not found"},
		"gi":         {map[string]any{"giVersion": "18.0.0.0"}, "giVersion"},
		"subnet":     {map[string]any{"subnetId": "/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/other/subnets/s"}, "subnet of properties.vnetId"},
		"cluster":    {map[string]any{"clusterName": "my_cluster"}, "clusterName"},
		"percentage": {map[string]any{"dataStoragePercentage": 50.0}, "35, 40, 60 or 80"},
		"ssh":        {map[string]any{"sshPublicKeys": []any{"not a key"}}, "OpenSSH"},
		"ocpu":       {map[string]any{"ocpuCount": 1.25}, "one decimal"},
		"server":     {map[string]any{"dbServers": []any{"ocid1.dbserver.oc1..nope"}}, "isn't a database server"},
		"twice":      {map[string]any{"dbServers": []any{servers[0], servers[0]}}, "twice"},
		"cores":      {map[string]any{"cpuCoreCount": 400.0, "dbServers": []any{servers[0], servers[1]}}, "has 126 left"},
		"nsg":        {map[string]any{"nsgCidrs": []any{map[string]any{"source": "10.0.0.0/16", "destinationPortRange": map[string]any{"min": 9.0, "max": 1.0}}}}, "can't exceed"},
		"required":   {map[string]any{"hostname": ""}, "hostname is required"},
	} {
		t.Run(name, func(t *testing.T) { h.rejects(http.MethodPut, vm+"bad", h.cluster(c.props), c.text) })
	}
	west := h.cluster(nil)
	west["location"] = "westus"
	h.rejects(http.MethodPut, vm+"bad", west, "region")

	one := h.props(http.MethodPut, vm+"one", h.cluster(map[string]any{"cpuCoreCount": 200.0, "ocpuCount": 1.1, "dbServers": []any{servers[0], servers[1]}}), http.StatusCreated)
	if one["nodeCount"] != 2.0 || one["dataStoragePercentage"] != 80.0 || one["licenseModel"] != "LicenseIncluded" || one["listenerPort"] != 1521.0 ||
		one["computeModel"] != "OCPU" || one["domain"] != "one.oraclevcn.com" || one["lifecycleState"] != "Available" {
		t.Fatalf("cluster = %#v", one)
	}
	// The second cluster has to fit beside the first: 100 cores per node
	// leave 26 on the shared servers.
	h.rejects(http.MethodPut, vm+"two", h.cluster(map[string]any{"cpuCoreCount": 60.0, "dbServers": []any{servers[1], servers[2]}}), "has 26 left")
	two := h.props(http.MethodPut, vm+"two", h.cluster(map[string]any{"cpuCoreCount": 40.0, "dbServers": []any{servers[1], servers[2]}}), http.StatusCreated)
	if azureARMSlice(two["dbServers"])[0] != servers[1] {
		t.Fatalf("cluster two = %#v", two)
	}
	shared := h.props(http.MethodGet, infra+"/dbServers/"+servers[1], nil, http.StatusOK)
	// Local storage nobody sized defaults to 60 GB a node per cluster.
	if shared["dbNodeStorageSizeInGbs"] != 120.0 {
		t.Fatalf("shared server storage = %#v", shared)
	}
	if shared["cpuCoreCount"] != 120.0 || len(azureARMSlice(shared["vmClusterIds"])) != 2 || len(azureARMSlice(shared["dbNodeIds"])) != 2 {
		t.Fatalf("shared server = %#v", shared)
	}
	h.rejects(http.MethodPatch, infra, map[string]any{"properties": map[string]any{"computeCount": 2.0}}, "VM cluster two uses")

	nodes := h.call(http.MethodGet, vm+"one/dbNodes", nil, http.StatusOK)["value"].([]any)
	if len(nodes) != 2 || azureARMObject(azureARMObject(nodes[0])["properties"])["cpuCoreCount"] != 100.0 {
		t.Fatalf("nodes = %#v", nodes)
	}
	node := vm + "one/dbNodes/" + azureARMString(azureARMObject(nodes[0])["name"])
	h.fails(http.MethodPut, node, map[string]any{}, http.StatusMethodNotAllowed, "not allowed")
	h.fails(http.MethodDelete, node, nil, http.StatusMethodNotAllowed, "not allowed")
	h.fails(http.MethodPost, node+"/action", map[string]any{"action": "Start"}, http.StatusConflict, "needs a node that is Stopped")
	h.rejects(http.MethodPost, node+"/action", map[string]any{"action": "Reboot"}, "action must be")
	if got := h.props(http.MethodPost, node+"/action", map[string]any{"action": "Stop"}, http.StatusOK); got["lifecycleState"] != "Stopped" {
		t.Fatalf("stopped node = %#v", got)
	}
	h.fails(http.MethodPost, node+"/action", map[string]any{"action": "Reset"}, http.StatusConflict, "Available")
	h.call(http.MethodPost, node+"/action", map[string]any{"action": "Start"}, http.StatusOK)

	// Immutable fields, the update model and capacity on update.
	h.rejects(http.MethodPut, vm+"one", h.cluster(map[string]any{"hostname": "other", "cpuCoreCount": 200.0, "dbServers": []any{servers[0], servers[1]}}), "hostname can't be changed")
	h.rejects(http.MethodPatch, vm+"one", map[string]any{"properties": map[string]any{"giVersion": "23.0.0.0"}}, "can't be updated")
	h.rejects(http.MethodPatch, vm+"one", map[string]any{"properties": map[string]any{"cpuCoreCount": 220.0}}, "has 106 left")
	if got := h.props(http.MethodPatch, vm+"one", map[string]any{"properties": map[string]any{"cpuCoreCount": 210.0, "displayName": "renamed"}}, http.StatusOK); got["displayName"] != "renamed" || got["ocid"] != one["ocid"] {
		t.Fatalf("patched cluster = %#v", got)
	}
	if got := h.props(http.MethodGet, node, nil, http.StatusOK); got["cpuCoreCount"] != 105.0 {
		t.Fatalf("node share after resize = %#v", got)
	}

	// addVms and removeVms move nodes between database servers.
	h.rejects(http.MethodPost, vm+"one/addVms", map[string]any{"dbServers": []any{servers[0]}}, "already hosts")
	h.rejects(http.MethodPost, vm+"one/addVms", map[string]any{}, "non-empty")
	// A stopped node stays stopped, and keeps its hostname, across a resize.
	twoNodes := h.call(http.MethodGet, vm+"two/dbNodes", nil, http.StatusOK)["value"].([]any)
	stopped := vm + "two/dbNodes/" + azureARMString(azureARMObject(twoNodes[1])["name"])
	before := h.props(http.MethodPost, stopped+"/action", map[string]any{"action": "Stop"}, http.StatusOK)
	grown := h.props(http.MethodPost, vm+"two/addVms", map[string]any{"dbServers": []any{servers[0]}}, http.StatusOK)
	grownNodes := h.call(http.MethodGet, vm+"two/dbNodes", nil, http.StatusOK)["value"].([]any)
	if grown["nodeCount"] != 3.0 || len(grownNodes) != 3 {
		t.Fatalf("after addVms = %#v", grown)
	}
	if after := h.props(http.MethodGet, stopped, nil, http.StatusOK); after["lifecycleState"] != "Stopped" || after["hostname"] != before["hostname"] {
		t.Fatalf("node after addVms = %#v, before %#v", after, before)
	}
	hosts, hostOf := map[any]bool{}, map[any]any{}
	for _, n := range grownNodes {
		p := azureARMObject(azureARMObject(n)["properties"])
		hosts[p["hostname"]], hostOf[p["dbServerId"]] = true, p["hostname"]
	}
	if len(hosts) != 3 {
		t.Fatalf("hostnames must stay unique: %v", hosts)
	}
	h.rejects(http.MethodPost, vm+"two/removeVms", map[string]any{"dbServers": []any{"ocid1.dbserver.oc1..nope"}}, "hosts no node")
	h.rejects(http.MethodPost, vm+"one/removeVms", map[string]any{"dbServers": []any{servers[0], servers[1]}}, "at least one")
	h.call(http.MethodPost, vm+"two/removeVms", map[string]any{"dbServers": []any{servers[2]}}, http.StatusOK)
	if left := h.call(http.MethodGet, vm+"two/dbNodes", nil, http.StatusOK)["value"].([]any); len(left) != 2 || azureARMObject(azureARMObject(left[0])["properties"])["cpuCoreCount"] != 20.0 {
		t.Fatalf("after removeVms = %#v", left)
	}
	// The nodes left keep their hostnames.
	for _, n := range h.call(http.MethodGet, vm+"two/dbNodes", nil, http.StatusOK)["value"].([]any) {
		if p := azureARMObject(azureARMObject(n)["properties"]); p["hostname"] != hostOf[p["dbServerId"]] {
			t.Fatalf("hostname of %v changed on removeVms: %v, was %v", p["dbServerId"], p["hostname"], hostOf[p["dbServerId"]])
		}
	}

	// Private IPs and virtual network addresses.
	subnet := azureARMString(one["subnetOcid"])
	vnic := azureARMString(azureARMObject(azureARMObject(nodes[1])["properties"])["vnicId"])
	if ips := h.do(http.MethodPost, vm+"one/listPrivateIpAddresses"+azureOraTestAPI, `{"subnetId":"`+subnet+`","vnicId":"`+vnic+`"}`); true {
		var got []any
		_ = json.Unmarshal(mustBody(t, ips), &got)
		if ips.StatusCode != http.StatusOK || len(got) != 1 {
			t.Fatalf("private IPs = %d %#v", ips.StatusCode, got)
		}
	}
	h.rejects(http.MethodPost, vm+"one/listPrivateIpAddresses", map[string]any{"subnetId": "subnet", "vnicId": vnic}, "OCIDs")
	vna := vm + "one/virtualNetworkAddresses/"
	h.rejects(http.MethodPut, vna+"vip1", map[string]any{"properties": map[string]any{"ipAddress": "999.1.1.1"}}, "IPv4")
	h.rejects(http.MethodPut, vna+"vip1", map[string]any{"properties": map[string]any{"vmOcid": "ocid1.dbnode.oc1..nope"}}, "database nodes")
	h.rejects(http.MethodPut, vna+strings.Repeat("v", 64), map[string]any{}, "63")
	vip := h.props(http.MethodPut, vna+"vip1", map[string]any{"properties": map[string]any{"ipAddress": "10.0.1.5", "vmOcid": azureARMObject(nodes[0])["name"]}}, http.StatusCreated)
	if vip["domain"] != "vip1.one.oraclevcn.com" || vip["lifecycleState"] != "Available" {
		t.Fatalf("vip = %#v", vip)
	}
	h.rejects(http.MethodPut, vna+"vip2", map[string]any{"properties": map[string]any{"ipAddress": "10.0.1.5"}}, "already a virtual IP")
	if auto := h.props(http.MethodPut, vna+"vip2", map[string]any{}, http.StatusCreated); auto["ipAddress"] != "10.0.1.100" {
		t.Fatalf("auto vip = %#v", auto)
	}
	// A node holding a virtual IP can't be removed.
	held := azureARMObject(azureARMObject(nodes[0])["properties"])["dbServerId"]
	h.fails(http.MethodPost, vm+"one/removeVms", map[string]any{"dbServers": []any{held}}, http.StatusConflict, "vip1")
	h.rejects(http.MethodPut, vna+"vip1", map[string]any{"properties": map[string]any{"ipAddress": "10.0.1.6"}}, "can't be changed")
	if again := h.props(http.MethodPut, vna+"vip1", map[string]any{}, http.StatusOK); again["ipAddress"] != "10.0.1.5" || again["ocid"] != vip["ocid"] {
		t.Fatalf("re-PUT vip = %#v", again)
	}
	h.fails(http.MethodPatch, vna+"vip1", map[string]any{}, http.StatusMethodNotAllowed, "not allowed")

	// Deleting a cluster frees its servers and takes its nodes and VIPs.
	h.call(http.MethodDelete, vm+"one", nil, http.StatusOK)
	h.call(http.MethodGet, vm+"one/dbNodes", nil, http.StatusNotFound)
	h.call(http.MethodGet, vna+"vip1", nil, http.StatusNotFound)
	if free := h.props(http.MethodGet, infra+"/dbServers/"+servers[0], nil, http.StatusOK); free["cpuCoreCount"] != 20.0 {
		t.Fatalf("freed server = %#v", free)
	}
	if items := h.call(http.MethodGet, h.sub+"/providers/Oracle.Database/cloudVmClusters", nil, http.StatusOK)["value"].([]any); len(items) != 1 {
		t.Fatalf("subscription clusters = %d", len(items))
	}
}
