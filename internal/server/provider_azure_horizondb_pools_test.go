package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAzureHorizonDBPools(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	c := h.base + "clusters/c1"
	h.call(http.MethodPut, c, azureHZCluster(nil), http.StatusCreated)
	h.call(http.MethodPut, c+"/pools/c1", azureHZPool(1, "Disabled"), http.StatusBadRequest)
	h.rejects(http.MethodPut, c+"/pools/p1", azureHZPool(1, "AnyZone"), "must be Disabled when nodeCount is 1")
	h.rejects(http.MethodPut, c+"/pools/p1", azureHZPool(2, "Disabled"), "must be Disabled when nodeCount is 1")
	h.rejects(http.MethodPut, c+"/pools/p1", azureHZPool(17, "AnyZone"), "nodeCount must be a whole number from 1 to 16")
	bad := azureHZPool(2, "AnyZone")
	azureARMObject(azureARMObject(bad["properties"])["computeModel"])["vCores"] = 1.0
	h.rejects(http.MethodPut, c+"/pools/p1", bad, "vCores must be a whole number from 2 to 192")
	delete(azureARMObject(bad["properties"]), "computeModel")
	h.rejects(http.MethodPut, c+"/pools/p1", bad, "computeModel is required")
	custom := azureHZPool(2, "AnyZone")
	azureARMObject(custom["properties"])["parameterGroup"] = map[string]any{"parameterGroupResourceId": h.arm + "parameterGroups/pg1", "applyMethod": "Immediate"}
	h.rejects(http.MethodPut, c+"/pools/p1", custom, "must be the cluster's default parameter group")

	pool := h.props(http.MethodPut, c+"/pools/p1", azureHZPool(3, "ZoneRedundant"), http.StatusCreated)
	group := azureARMObject(pool["parameterGroup"])
	if pool["state"] != "Ready" || !strings.HasSuffix(azureARMString(group["parameterGroupResourceId"]), "/locations/eastus/parameterGroups/default-postgresql-17-provisioned") ||
		group["syncStatus"] != "InSync" || azureARMObject(pool["scheduledMaintenance"])["maintenanceManagementType"] != "System" {
		t.Fatalf("pool = %v", pool)
	}
	h.fails(http.MethodPut, c+"/pools/p2", azureHZPool(1, "Disabled"), http.StatusConflict, "one pool per cluster")

	nodes := h.list(c + "/pools/p1/nodes")
	zones := []string{}
	for _, n := range nodes {
		zones = append(zones, azureARMString(azureARMObject(azureARMObject(n)["properties"])["availabilityZone"]))
	}
	if len(nodes) != 3 || strings.Join(zones, ",") != "1,2,3" {
		t.Fatalf("nodes = %v", nodes)
	}
	if got := h.props(http.MethodGet, c+"/pools/p1/nodes/node-01", nil, http.StatusOK); got["role"] != "ReadWrite" {
		t.Fatalf("node-01 = %v", got)
	}
	// PATCH merges, then checks the effective node count and mode.
	h.rejects(http.MethodPatch, c+"/pools/p1", map[string]any{"properties": map[string]any{"nodeCount": 1.0}}, "must be Disabled when nodeCount is 1")
	scaled := h.props(http.MethodPatch, c+"/pools/p1", map[string]any{"properties": map[string]any{"nodeCount": 1.0, "highAvailability": map[string]any{"mode": "Disabled"}}}, http.StatusOK)
	if scaled["nodeCount"] != 1.0 || azureARMObject(scaled["computeModel"])["vCores"] != 2.0 {
		t.Fatalf("scaled = %v", scaled)
	}
	h.fails(http.MethodPost, c+"/failover", nil, http.StatusConflict, "at least two nodes")
	h.props(http.MethodPatch, c+"/pools/p1", map[string]any{"properties": map[string]any{"nodeCount": 2.0, "highAvailability": map[string]any{"mode": "AnyZone"}}}, http.StatusOK)
	h.call(http.MethodPost, c+"/failover", nil, http.StatusOK)

	// Power actions: stop deallocates the nodes, restart needs a running pool.
	if got := h.props(http.MethodPost, c+"/pools/p1/stop", nil, http.StatusOK); got["state"] != "Stopped" {
		t.Fatalf("stopped = %v", got)
	}
	h.props(http.MethodPost, c+"/pools/p1/stop", nil, http.StatusOK)
	if got := h.list(c + "/pools/p1/nodes"); len(got) != 0 {
		t.Fatalf("nodes while stopped = %v", got)
	}
	h.fails(http.MethodPost, c+"/pools/p1/restart", nil, http.StatusConflict, "needs it Ready")
	h.fails(http.MethodPost, c+"/failover", nil, http.StatusConflict, "must be Ready")
	if got := h.props(http.MethodPut, c+"/pools/p1", azureHZPool(2, "AnyZone"), http.StatusOK); got["state"] != "Stopped" {
		t.Fatalf("replace changed state = %v", got)
	}
	h.props(http.MethodPost, c+"/pools/p1/start", nil, http.StatusOK)
	h.props(http.MethodPost, c+"/pools/p1/start", nil, http.StatusOK)
	if got := h.props(http.MethodPost, c+"/pools/p1/restart", nil, http.StatusOK); got["state"] != "Ready" {
		t.Fatalf("restarted = %v", got)
	}

	fw := c + "/pools/p1/firewallRules/"
	rule := func(start, end string) map[string]any {
		return map[string]any{"properties": map[string]any{"startIpAddress": start, "endIpAddress": end, "description": "office"}}
	}
	h.rejects(http.MethodPut, fw+"office", rule("10.0.0.9", "10.0.0.1"), "must not be greater")
	h.rejects(http.MethodPut, fw+"office", rule("::ffff:10.0.0.1", "10.0.0.1"), "dotted-decimal IPv4")
	h.call(http.MethodPut, fw+"-bad", rule("10.0.0.1", "10.0.0.1"), http.StatusBadRequest)
	h.call(http.MethodPut, fw+"office", rule("10.0.0.1", "10.0.0.9"), http.StatusCreated)
	h.call(http.MethodPatch, fw+"office", rule("10.0.0.1", "10.0.0.9"), http.StatusMethodNotAllowed)
	h.call(http.MethodDelete, fw+"office", nil, http.StatusNoContent)
}

func TestAzureHorizonDBMaintenanceAndLogs(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	c := h.cluster()
	events := h.list(c + "/pools/p1/maintenanceEvents")
	if len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
	event := azureARMObject(events[0])
	id := azureARMString(event["name"])
	props := azureARMObject(event["properties"])
	if len(id) != 8 || id[4] != '-' || props["status"] != "Planned" || props["deferrable"] != true || props["startTimeUtc"] != props["originalStartTimeUtc"] {
		t.Fatalf("event = %v", event)
	}
	path := c + "/pools/p1/maintenanceEvents/" + id
	h.call(http.MethodPut, path, map[string]any{"properties": map[string]any{}}, http.StatusMethodNotAllowed)
	if got := h.list(c + "/pools/p1/maintenanceEvents?$filter=" + url.QueryEscape("properties/status eq 'Complete'")); len(got) != 0 {
		t.Fatalf("filtered = %v", got)
	}
	start, _ := time.Parse(time.RFC3339, azureARMString(props["startTimeUtc"]))
	deadline, _ := time.Parse(time.RFC3339, azureARMString(props["deferralDeadlineUtc"]))
	h.rejects(http.MethodPost, path+"/reschedule", map[string]any{"postponeToDateTimeUtc": deadline.Add(time.Hour).Format(time.RFC3339)}, "no later than the deferral deadline")
	h.rejects(http.MethodPost, path+"/reschedule", map[string]any{}, "postponeToDateTimeUtc is required")
	to := start.Add(48 * time.Hour).Format(time.RFC3339)
	moved := h.call(http.MethodPost, path+"/reschedule", map[string]any{"postponeToDateTimeUtc": to}, http.StatusOK)
	if moved["status"] != "Rescheduled" || moved["plannedStartTimeUtc"] != to || moved["appliedNow"] != false || moved["poolResourceId"] != h.arm+"clusters/c1/pools/p1" {
		t.Fatalf("rescheduled = %v", moved)
	}
	got := h.props(http.MethodGet, path, nil, http.StatusOK)
	if got["rescheduledFromUtc"] != props["startTimeUtc"] || got["originalStartTimeUtc"] != props["startTimeUtc"] {
		t.Fatalf("event after reschedule = %v", got)
	}
	filter := url.QueryEscape("properties/status eq 'Planned' or properties/status eq 'Rescheduled'")
	if got := h.list(c + "/pools/p1/maintenanceEvents?$filter=" + filter); len(got) != 1 {
		t.Fatalf("filtered = %v", got)
	}
	if applied := h.call(http.MethodPost, path+"/applyNow", nil, http.StatusOK); applied["status"] != "InProgress" || applied["appliedNow"] != true {
		t.Fatalf("applied = %v", applied)
	}
	h.fails(http.MethodPost, path+"/applyNow", nil, http.StatusConflict, "only planned or rescheduled")

	logs := h.list(c + "/logFiles")
	if len(logs) != 1 {
		t.Fatalf("log files = %v", logs)
	}
	log := azureARMObject(logs[0])
	if p := azureARMObject(log["properties"]); p["pool"] != "p1" || p["role"] != "ReadWrite" || p["logType"] != "PostgreSqlServer" {
		t.Fatalf("log file = %v", log)
	}
	urls := h.call(http.MethodPost, c+"/logFiles/"+azureARMString(log["name"])+"/listDownloadUrls", nil, http.StatusOK)
	if !strings.Contains(azureARMString(urls["downloadUri"]), "sig=") || urls["expiresAtUtc"] == nil {
		t.Fatalf("download = %v", urls)
	}
	h.call(http.MethodDelete, c+"/logFiles/"+azureARMString(log["name"]), nil, http.StatusMethodNotAllowed)

	capture := c + "/logCaptureConfigurations/PostgreSqlServer"
	h.rejects(http.MethodPatch, capture, map[string]any{"properties": map[string]any{"retentionDays": 40.0}}, "retentionDays must be a whole number from 1 to 35")
	if got := h.props(http.MethodPatch, capture, map[string]any{"properties": map[string]any{"state": "Disabled"}}, http.StatusOK); got["state"] != "Disabled" || got["retentionDays"] != 7.0 {
		t.Fatalf("capture = %v", got)
	}
	h.call(http.MethodPut, capture, map[string]any{"properties": map[string]any{"state": "Enabled", "retentionDays": 7.0}}, http.StatusMethodNotAllowed)
}

func TestAzureHorizonDBParameterGroups(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	pg := h.base + "parameterGroups/pg1"
	body := func(params ...map[string]any) map[string]any {
		items := []any{}
		for _, p := range params {
			items = append(items, p)
		}
		return map[string]any{"location": "eastus", "properties": map[string]any{"postgreSqlVersion": "17", "parameters": items}}
	}
	param := func(name, value string) map[string]any { return map[string]any{"name": name, "value": value} }
	h.call(http.MethodPut, h.base+"parameterGroups/x", body(), http.StatusBadRequest)
	h.rejects(http.MethodPut, pg, body(param("nope", "1")), "isn't a HorizonDB engine parameter")
	h.rejects(http.MethodPut, pg, body(param("allow_alter_system", "off")), "is read-only")
	h.rejects(http.MethodPut, pg, body(param("work_mem", "1")), "must be an integer from 4096")
	h.rejects(http.MethodPut, pg, body(param("application_name", "bad name")), "must match")
	h.rejects(http.MethodPut, pg, body(param("autovacuum_analyze_scale_factor", "101")), "must be a number from 0-100")
	h.rejects(http.MethodPut, pg, body(param("work_mem", "8192"), param("work_mem", "8192")), "more than once")

	created := h.props(http.MethodPut, pg, body(param("work_mem", "8192"), param("application_name", "app-1")), http.StatusCreated)
	values := map[string]any{}
	for _, item := range azureARMSlice(created["parameters"]) {
		entry := azureARMObject(item)
		values[azureARMString(entry["name"])] = entry["value"].(string) + "/" + azureARMString(entry["valueSource"])
	}
	if values["work_mem"] != "8192/UserOverride" || values["autovacuum"] != "on/SystemDefault" || len(values) != len(azureHZParameters) {
		t.Fatalf("parameters = %v", values)
	}
	// A GET body can be sent back as is.
	roundTrip := map[string]any{"location": "eastus", "properties": created}
	if got := h.props(http.MethodPut, pg, roundTrip, http.StatusOK); azureARMSlice(got["parameters"])[0] == nil {
		t.Fatalf("round trip = %v", got)
	}
	h.rejects(http.MethodPut, pg, map[string]any{"location": "eastus", "properties": map[string]any{"postgreSqlVersion": "18"}}, "can't be changed")
	h.rejects(http.MethodPatch, pg, map[string]any{"properties": map[string]any{"postgreSqlVersion": "18"}}, "can't be updated")
	h.props(http.MethodPatch, pg, map[string]any{"properties": map[string]any{"description": "tuned"}}, http.StatusOK)

	// An existing pool can switch to the group, which then can't be deleted.
	c := h.cluster()
	h.call(http.MethodPut, h.base+"parameterGroups/pg18", map[string]any{"location": "eastus", "properties": map[string]any{"postgreSqlVersion": "18"}}, http.StatusCreated)
	h.rejects(http.MethodPatch, c+"/pools/p1", map[string]any{"properties": map[string]any{"parameterGroup": map[string]any{"parameterGroupResourceId": h.arm + "parameterGroups/pg18", "applyMethod": "Immediate"}}}, "must match the cluster's")
	h.rejects(http.MethodPatch, c+"/pools/p1", map[string]any{"properties": map[string]any{"parameterGroup": map[string]any{"parameterGroupResourceId": h.arm + "parameterGroups/pg1"}}}, "applyMethod is required")
	h.props(http.MethodPatch, c+"/pools/p1", map[string]any{"properties": map[string]any{"parameterGroup": map[string]any{"parameterGroupResourceId": h.arm + "parameterGroups/pg1", "applyMethod": "PendingReboot"}}}, http.StatusOK)
	conns := h.list(pg + "/connections")
	if len(conns) != 1 || azureARMObject(conns[0])["poolResourceId"] != h.arm+"clusters/c1/pools/p1" {
		t.Fatalf("connections = %v", conns)
	}
	h.fails(http.MethodDelete, pg, nil, http.StatusConflict, "associated with one or more pools")
	h.call(http.MethodDelete, c+"/pools/p1", nil, http.StatusNoContent)
	h.call(http.MethodDelete, pg, nil, http.StatusNoContent)
	if got := h.list(h.sub + "/providers/Microsoft.HorizonDb/parameterGroups"); len(got) != 1 {
		t.Fatalf("subscription groups = %v", got)
	}

	loc := h.sub + "/providers/Microsoft.HorizonDb/locations/eastus/"
	defaults := h.list(loc + "parameterGroups")
	if len(defaults) != 2 {
		t.Fatalf("default groups = %v", defaults)
	}
	if got := h.props(http.MethodGet, loc+"parameterGroups/default-postgresql-18-provisioned", nil, http.StatusOK); got["postgreSqlVersion"] != "18" {
		t.Fatalf("default group = %v", got)
	}
	caps := h.props(http.MethodGet, loc+"capabilities/default", nil, http.StatusOK)
	if azureARMObject(caps["clusterCapabilities"])["recommendedPostgreSqlVersion"] != "17" || len(h.list(loc+"capabilities")) != 1 {
		t.Fatalf("capabilities = %v", caps)
	}
}

func TestAzureHorizonDBReviewRules(t *testing.T) {
	t.Parallel()
	h := newAzureHZTest(t)
	c := h.cluster()

	// PATCH overlays parameter overrides by name; PUT replaces them.
	pg := h.base + "parameterGroups/pg1"
	param := func(name, value string) map[string]any { return map[string]any{"name": name, "value": value} }
	override := func(props map[string]any, name string) any {
		for _, item := range azureARMSlice(props["parameters"]) {
			if entry := azureARMObject(item); entry["name"] == name && entry["valueSource"] == "UserOverride" {
				return entry["value"]
			}
		}
		return nil
	}
	h.call(http.MethodPut, pg, map[string]any{"location": "eastus", "properties": map[string]any{"postgreSqlVersion": "17"}}, http.StatusCreated)
	h.call(http.MethodPatch, pg, map[string]any{"properties": map[string]any{"parameters": []any{param("work_mem", "8192")}}}, http.StatusOK)
	both := h.props(http.MethodPatch, pg, map[string]any{"properties": map[string]any{"parameters": []any{param("application_name", "app-1")}}}, http.StatusOK)
	if override(both, "work_mem") != "8192" || override(both, "application_name") != "app-1" || both[azureHZPatchParameters] != nil {
		t.Fatalf("patched parameters = %v", both)
	}
	again := h.props(http.MethodPatch, pg, map[string]any{"properties": map[string]any{"parameters": []any{param("work_mem", "16384")}}}, http.StatusOK)
	if override(again, "work_mem") != "16384" || override(again, "application_name") != "app-1" {
		t.Fatalf("re-patched parameters = %v", again)
	}
	replaced := h.props(http.MethodPut, pg, map[string]any{"location": "eastus", "properties": map[string]any{"postgreSqlVersion": "17", "parameters": []any{param("application_name", "app-2")}}}, http.StatusOK)
	if override(replaced, "work_mem") != nil || override(replaced, "application_name") != "app-2" {
		t.Fatalf("replaced parameters = %v", replaced)
	}
	// A name is listed once, even when its first entry sets nothing.
	h.rejects(http.MethodPatch, pg, map[string]any{"properties": map[string]any{"parameters": []any{map[string]any{"name": "work_mem"}, param("work_mem", "8192")}}}, "more than once")

	// vCores must be a catalog size.
	odd := azureHZPool(1, "Disabled")
	azureARMObject(azureARMObject(odd["properties"])["computeModel"])["vCores"] = 5.0
	h.rejects(http.MethodPut, c+"/pools/p1", odd, "capability catalog")

	// First enabling password authentication needs the password too.
	h.rejects(http.MethodPut, c+"/authentications/passwordMethod", map[string]any{"properties": map[string]any{"state": "Enabled", "administratorLogin": "admin"}}, "administratorLoginPassword is required")

	// A reschedule moves the event later and keeps the original start.
	event := azureARMObject(h.list(c + "/pools/p1/maintenanceEvents")[0])
	props := azureARMObject(event["properties"])
	path := c + "/pools/p1/maintenanceEvents/" + azureARMString(event["name"])
	start, _ := time.Parse(time.RFC3339, azureARMString(props["startTimeUtc"]))
	h.call(http.MethodPost, path+"/reschedule", map[string]any{"postponeToDateTimeUtc": start.Add(48 * time.Hour).Format(time.RFC3339)}, http.StatusOK)
	h.rejects(http.MethodPost, path+"/reschedule", map[string]any{"postponeToDateTimeUtc": start.Add(24 * time.Hour).Format(time.RFC3339)}, "after the current start")
	h.call(http.MethodPost, path+"/reschedule", map[string]any{"postponeToDateTimeUtc": start.Add(72 * time.Hour).Format(time.RFC3339)}, http.StatusOK)
	if got := h.props(http.MethodGet, path, nil, http.StatusOK); got["rescheduledFromUtc"] != props["originalStartTimeUtc"] {
		t.Fatalf("rescheduled twice = %v", got)
	}
}
