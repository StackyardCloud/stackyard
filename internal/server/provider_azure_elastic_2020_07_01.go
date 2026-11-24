package server

import (
	"cmp"
	"net/http"
	"slices"
	"strings"
	"sync"
)

const azureElastic20200701APIVersion = "2020-07-01"

// Elastic Cloud (Microsoft.Elastic 2020-07-01) monitors on the shared ARM
// handler: monitors with their tag rules, the deployment, monitored
// resource and VM actions, and the operations list. No Elastic deployment
// runs; monitored resources are the subscription's stored ARM resources
// that the monitor's tag rules select.

const (
	elasNamespace   = "Microsoft.Elastic"
	elasMonitorType = elasNamespace + "/monitors"
	elasMaxVMs      = 1000 // VM resources per monitor's agent collection
	elasMaxTags     = 50   // filtering tags per rule set
	elasMaxListed   = 1000 // monitored resources one list returns
)

var (
	elasEnabled     = []string{"Enabled", "Disabled"}
	elasTagActions  = []string{"Include", "Exclude"}
	elasVMOps       = []string{"Add", "Delete"}
	elasIdentityTyp = []string{"SystemAssigned"}
	elasVMIDRE      = azureNCIDRE("Microsoft.Compute/virtualMachines")
)

// elasVMs holds each server's monitor agent VM collections, keyed by the
// lower-cased monitor ID.
// ponytail: package-level, keyed by server; entries go with their monitor.
var elasVMs = struct {
	sync.Mutex
	m map[*Server]map[string][]string
}{m: map[*Server]map[string][]string{}}

func init() {
	ns := elasNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors",
		Validate: elasValidateMonitor,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			id := azureHZGUID(rec.ID)
			props := map[string]any{
				"elasticProperties": map[string]any{
					"elasticCloudUser": map[string]any{
						"emailAddress": azureARMString(azureARMObject(rec.Properties["userInfo"])["emailAddress"]),
						"id":           id[:8], "elasticCloudSsoDefaultUrl": "https://cloud.elastic.co/sso/" + id[:8],
					},
					"elasticCloudDeployment": map[string]any{
						"name": rec.Name, "deploymentId": id[:12], "azureSubscriptionId": rec.SubscriptionID,
						"elasticsearchRegion": "azure-" + rec.Location, "elasticsearchServiceUrl": "https://" + id[:12] + "." + rec.Location + ".azure.elastic-cloud.com",
						"kibanaServiceUrl": "https://" + id[:12] + ".kb." + rec.Location + ".azure.elastic-cloud.com",
						"kibanaSsoUrl":     "https://" + id[:12] + ".kb." + rec.Location + ".azure.elastic-cloud.com/sso",
					},
				},
				"liftrResourceCategory":   "MonitorLogs",
				"liftrResourcePreference": 0,
			}
			if _, set := rec.Properties["monitoringStatus"]; !set {
				props["monitoringStatus"] = "Enabled"
			}
			return props
		},
		WriteOnlyProperties: []string{"userInfo"},
		Deleted: func(s *Server, removed *azureARMResource) {
			elasVMs.Lock()
			delete(elasVMs.m[s], azureARMKey(removed.ID))
			elasVMs.Unlock()
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"listmonitoredresources": elasMonitoredResources,
			"vmcollectionupdate":     elasVMCollectionUpdate,
			"listvmhost":             elasVMHosts,
		},
		Actions: map[string]azureARMAction{
			"listdeploymentinfo": func(rec *azureARMResource, _ map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"status": "Healthy", "version": "7.17.3", "memoryCapacity": "4096", "diskCapacity": "122880"}
			},
			"vmingestiondetails": func(rec *azureARMResource, _ map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"cloudId": rec.Name + ":" + azureARMDerivedKey(rec.ID, "cloud")[:24], "ingestionKey": azureARMDerivedKey(rec.ID, "ingestion")}
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/tagRules", Proxy: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The tag rule set name must be 'default'."
			}
			return ""
		},
		Validate: elasValidateTagRules,
	})
	registerAzureARMView(ns, "operations", elasOperations("Microsoft Elastic", ns, "monitors", "monitors/tagRules"))
}

// handleAzureElastic20200701Router stays as the stage-0 dispatch hook; the
// shared ARM handler, which runs first, serves every route.
func (s *Server) handleAzureElastic20200701Router(http.ResponseWriter, *http.Request) bool {
	return false
}

// elasOperations returns an operations view listing read, write and delete
// for each resource type of a provider namespace.
func elasOperations(provider, ns string, types ...string) azureARMView {
	return func(*Server, string) []map[string]any {
		var ops []map[string]any
		for _, typ := range types {
			for _, verb := range []string{"read", "write", "delete"} {
				ops = append(ops, map[string]any{
					"name": ns + "/" + typ + "/" + verb, "isDataAction": false, "origin": "user,system",
					"display": map[string]any{"provider": provider, "resource": typ, "operation": verb + " " + typ, "description": verb + " " + typ + "."},
				})
			}
		}
		return ops
	}
}

func elasValidateMonitor(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	email := azureARMString(azureARMObject(props["userInfo"])["emailAddress"])
	msg := cmp.Or(
		azureLabEnum(azureARMObject(payload["identity"]), "identity.", "type", elasIdentityTyp),
		azureLabEnum(props, "properties.", "monitoringStatus", elasEnabled),
		azureLabEnum(props, "properties.", "provisioningState", []string{"Accepted", "Creating", "Updating", "Deleting", "Succeeded", "Failed", "Canceled", "Deleted", "NotSpecified"}),
	)
	if msg == "" && email != "" && !strings.Contains(email, "@") {
		msg = "properties.userInfo.emailAddress is invalid."
	}
	if sku, present := payload["sku"]; msg == "" && present && strings.TrimSpace(azureARMString(azureARMObject(sku)["name"])) == "" {
		msg = "sku.name is required."
	}
	return msg
}

func elasValidateTagRules(payload map[string]any) string {
	logRules := azureARMObject(azureARMObject(payload["properties"])["logRules"])
	tags := azureARMSlice(logRules["filteringTags"])
	if len(tags) > elasMaxTags {
		return "properties.logRules.filteringTags is invalid: at most 50 tags."
	}
	for _, raw := range tags {
		tag, ok := raw.(map[string]any)
		if !ok {
			return "properties.logRules.filteringTags is invalid: each tag must be an object."
		}
		if msg := azureWorkloadsEnums(tag, "properties.logRules.filteringTags[].", []string{"name", "action"}, map[string][]string{"action": elasTagActions}); msg != "" {
			return msg
		}
	}
	for _, key := range []string{"sendAadLogs", "sendSubscriptionLogs", "sendActivityLogs"} {
		if raw, present := logRules[key]; present {
			if _, ok := raw.(bool); !ok {
				return "properties.logRules." + key + " is invalid: it must be a boolean."
			}
		}
	}
	return ""
}

// elasTagMatch reports whether a resource's tags select it under the rule
// set: no include tags includes everything, and an exclude tag wins.
func elasTagMatch(tags map[string]any, rules []any) bool {
	included, hasInclude := false, false
	for _, raw := range rules {
		rule := azureARMObject(raw)
		value, present := tags[azureARMString(rule["name"])]
		hit := present && (azureARMString(rule["value"]) == "" || azureARMString(value) == azureARMString(rule["value"]))
		switch azureARMString(rule["action"]) {
		case "Exclude":
			if hit {
				return false
			}
		case "Include":
			hasInclude = true
			included = included || hit
		}
	}
	return included || !hasInclude
}

// elasMonitoredResources lists the subscription's resources the monitor's
// tag rules select, other than Elastic monitors themselves.
func elasMonitoredResources(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	rules, _ := s.getAzureARMResource(rec.ID+"/tagRules/default", "", "", "")
	var filtering []any
	sendLogs := false
	if rules != nil {
		logRules := azureARMObject(rules.Properties["logRules"])
		filtering = azureARMSlice(logRules["filteringTags"])
		sendLogs, _ = logRules["sendSubscriptionLogs"].(bool)
	}
	prefix := azureARMKey("/subscriptions/" + rec.SubscriptionID + "/")
	resources := s.azureARMResourcesWhere(func(r *azureARMResource) bool {
		return strings.HasPrefix(azureARMKey(r.ID), prefix) && !strings.HasPrefix(strings.ToLower(r.Type), strings.ToLower(elasNamespace+"/")) && elasTagMatch(r.Tags, filtering)
	})
	value := []any{}
	for _, r := range resources[:min(len(resources), elasMaxListed)] {
		logs, reason := "False", "Subscription logs aren't enabled in the tag rules."
		if sendLogs {
			logs, reason = "True", ""
		}
		value = append(value, map[string]any{"id": r.ID, "sendingLogs": logs, "reasonForLogsStatus": reason})
	}
	return http.StatusOK, map[string]any{"value": value}
}

// elasVMCollectionUpdate adds a VM to, or removes one from, the VMs the
// monitor's agent runs on.
func elasVMCollectionUpdate(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureWorkloadsEnums(payload, "", []string{"operationName", "vmResourceId"}, map[string][]string{"operationName": elasVMOps}); msg != "" {
		return azureOracleBad(msg)
	}
	vmID := azureARMString(payload["vmResourceId"])
	if !elasVMIDRE.MatchString(vmID) {
		return azureOracleBad("vmResourceId is invalid: it must be a virtual machine resource ID.")
	}
	elasVMs.Lock()
	defer elasVMs.Unlock()
	if elasVMs.m[s] == nil {
		elasVMs.m[s] = map[string][]string{}
	}
	key := azureARMKey(rec.ID)
	vms := elasVMs.m[s][key]
	present := slices.ContainsFunc(vms, func(id string) bool { return strings.EqualFold(id, vmID) })
	switch payload["operationName"] {
	case "Add":
		if !present {
			if len(vms) >= elasMaxVMs {
				return http.StatusConflict, azureARMErrorBody("Conflict", "The monitor already collects from the maximum of 1000 VMs.")
			}
			vms = append(append([]string(nil), vms...), vmID)
		}
	default:
		kept := make([]string, 0, len(vms))
		for _, id := range vms {
			if !strings.EqualFold(id, vmID) {
				kept = append(kept, id)
			}
		}
		vms = kept
	}
	elasVMs.m[s][key] = vms
	return http.StatusOK, nil
}

// elasVMHosts lists the VMs the monitor's agent runs on.
func elasVMHosts(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	elasVMs.Lock()
	vms := elasVMs.m[s][azureARMKey(rec.ID)]
	elasVMs.Unlock()
	value := []any{}
	for _, id := range vms {
		value = append(value, map[string]any{"vmResourceId": id})
	}
	return http.StatusOK, map[string]any{"value": value}
}
