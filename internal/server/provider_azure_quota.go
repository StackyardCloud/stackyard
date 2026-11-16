package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Quota limits, usages and quota requests at a resource provider
// location scope, /subscriptions/{id}/providers/{namespace}/locations/
// {location}, served by the shared ARM handler. GET lists default limits,
// with the ones a client set by PUT on top; every PUT is approved at once
// and recorded as a succeeded quota request. Network usage counts the
// emulator's stored network resources; other usage is zero.
func init() {
	const ns = "Microsoft.Quota"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "quotas", Extension: true,
		// The quota already exists, so a PUT only ever updates it.
		CreateStatus:       http.StatusOK,
		ValidateScope:      azureQuotaScope,
		Validate:           azureQuotaValidate,
		ReadOnlyProperties: azureQuotaLimitProperties,
	})
	registerAzureARMView(ns, "quotas", azureQuotaLimits)
	registerAzureARMView(ns, "usages", azureQuotaUsages)
	registerAzureARMView(ns, "quotaRequests", azureQuotaRequests)
}

// azureQuotaDefault is a limit every subscription starts with. Counts is
// the stored resource type whose records count as usage, if any.
type azureQuotaDefault struct {
	Name, Display string
	Limit         int
	Counts        string
}

// azureQuotaDefaults lists the default limits per resource provider.
// ponytail: a few common Compute and Network limits; add others when needed.
var azureQuotaDefaults = map[string][]azureQuotaDefault{
	"microsoft.compute": {
		{Name: "cores", Display: "Total Regional vCPUs", Limit: 100},
		{Name: "virtualMachines", Display: "Virtual Machines", Limit: 25000},
		{Name: "availabilitySets", Display: "Availability Sets", Limit: 2500},
		{Name: "virtualMachineScaleSets", Display: "Virtual Machine Scale Sets", Limit: 2500},
		{Name: "standardBSFamily", Display: "Standard BS Family vCPUs", Limit: 100},
		{Name: "standardDSv5Family", Display: "Standard DSv5 Family vCPUs", Limit: 100},
		{Name: "standardDv5Family", Display: "Standard Dv5 Family vCPUs", Limit: 100},
		{Name: "standardEv5Family", Display: "Standard Ev5 Family vCPUs", Limit: 100},
		{Name: "standardFSv2Family", Display: "Standard FSv2 Family vCPUs", Limit: 100},
		{Name: "standardNCADSA100v4Family", Display: "Standard NCADS_A100_v4 Family vCPUs", Limit: 0},
	},
	"microsoft.network": {
		{Name: "VirtualNetworks", Display: "Virtual Networks", Limit: 1000, Counts: "Microsoft.Network/virtualNetworks"},
		{Name: "PublicIPAddresses", Display: "Public IP Addresses", Limit: 1000, Counts: "Microsoft.Network/publicIPAddresses"},
		{Name: "NetworkSecurityGroups", Display: "Network Security Groups", Limit: 5000, Counts: "Microsoft.Network/networkSecurityGroups"},
		{Name: "NetworkInterfaces", Display: "Network Interfaces", Limit: 65536, Counts: "Microsoft.Network/networkInterfaces"},
		{Name: "LoadBalancers", Display: "Load Balancers", Limit: 1000, Counts: "Microsoft.Network/loadBalancers"},
		{Name: "RouteTables", Display: "Route Tables", Limit: 200, Counts: "Microsoft.Network/routeTables"},
	},
}

// azureQuotaScopeParts splits /subscriptions/{id}/providers/{namespace}/
// locations/{location}.
func azureQuotaScopeParts(scope string) (sub, namespace, location string, ok bool) {
	seg := strings.Split(strings.Trim(scope, "/"), "/")
	if len(seg) != 6 || !strings.EqualFold(seg[0], "subscriptions") || !strings.EqualFold(seg[2], "providers") || !strings.EqualFold(seg[4], "locations") {
		return "", "", "", false
	}
	return seg[1], seg[3], seg[5], true
}

func azureQuotaScope(scope string) string {
	if _, _, _, ok := azureQuotaScopeParts(scope); !ok {
		return "The quota scope must be /subscriptions/{subscriptionId}/providers/{resourceProvider}/locations/{location}."
	}
	return ""
}

func azureQuotaValidate(payload map[string]any) string {
	limit := azureARMObject(azureARMObject(payload["properties"])["limit"])
	if limit["limitObjectType"] != "LimitValue" {
		return "properties.limit.limitObjectType must be LimitValue."
	}
	if !azureWholeNumber(limit["value"], 0, 1<<31-1) {
		return "properties.limit.value must be a non-negative 32-bit integer."
	}
	if raw, present := limit["limitType"]; present && raw != "Independent" && raw != "Shared" {
		return "properties.limit.limitType must be Independent or Shared."
	}
	return ""
}

// azureQuotaDefaultFor returns the default named name in scope's provider.
func azureQuotaDefaultFor(scope, name string) (azureQuotaDefault, bool) {
	_, namespace, _, _ := azureQuotaScopeParts(scope)
	for _, def := range azureQuotaDefaults[strings.ToLower(namespace)] {
		if strings.EqualFold(def.Name, name) {
			return def, true
		}
	}
	return azureQuotaDefault{}, false
}

func azureQuotaScopeOf(id string) string {
	return id[:strings.LastIndex(strings.ToLower(id), "/providers/microsoft.quota/")]
}

func azureQuotaNameProperty(scope, name string) map[string]any {
	display := name
	if def, ok := azureQuotaDefaultFor(scope, name); ok {
		name, display = def.Name, def.Display
	}
	return map[string]any{"value": name, "localizedValue": display}
}

func azureQuotaLimitProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"name":              azureQuotaNameProperty(azureQuotaScopeOf(rec.ID), rec.Name),
		"unit":              "Count",
		"isQuotaApplicable": true,
	}
}

// azureQuotaStored returns the limits set by PUT directly under scope.
func (s *Server) azureQuotaStored(scope string) []*azureARMResource {
	prefix := azureARMKey(scope + "/providers/Microsoft.Quota/quotas/")
	return s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.HasPrefix(azureARMKey(rec.ID), prefix)
	})
}

// azureQuotaLimits lists the defaults, each replaced by the limit a client
// set, followed by limits set for names without a default.
func azureQuotaLimits(s *Server, scope string) []map[string]any {
	_, namespace, _, ok := azureQuotaScopeParts(scope)
	if !ok {
		return nil
	}
	stored := map[string]map[string]any{}
	var extra []map[string]any
	for _, rec := range s.azureQuotaStored(scope) {
		stored[strings.ToLower(rec.Name)] = rec.fixture()
		if _, known := azureQuotaDefaultFor(scope, rec.Name); !known {
			extra = append(extra, rec.fixture())
		}
	}
	items := make([]map[string]any, 0, len(azureQuotaDefaults[strings.ToLower(namespace)])+len(extra))
	for _, def := range azureQuotaDefaults[strings.ToLower(namespace)] {
		if item := stored[strings.ToLower(def.Name)]; item != nil {
			items = append(items, item)
			continue
		}
		items = append(items, map[string]any{
			"id": scope + "/providers/Microsoft.Quota/quotas/" + def.Name, "name": def.Name, "type": "Microsoft.Quota/quotas",
			"properties": map[string]any{
				"name":              azureQuotaNameProperty(scope, def.Name),
				"limit":             map[string]any{"limitObjectType": "LimitValue", "limitType": "Independent", "value": def.Limit},
				"unit":              "Count",
				"isQuotaApplicable": true,
			},
		})
	}
	return append(items, extra...)
}

// azureQuotaUsages reports usage for every listed limit.
func azureQuotaUsages(s *Server, scope string) []map[string]any {
	sub, _, location, _ := azureQuotaScopeParts(scope)
	limits := azureQuotaLimits(s, scope)
	items := make([]map[string]any, 0, len(limits))
	for _, limit := range limits {
		name := azureARMString(limit["name"])
		used := 0
		if def, ok := azureQuotaDefaultFor(scope, name); ok && def.Counts != "" {
			used = len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
				return strings.EqualFold(rec.Type, def.Counts) && strings.EqualFold(rec.SubscriptionID, sub) && azureLocationHost(rec.Location) == azureLocationHost(location)
			}))
		}
		items = append(items, map[string]any{
			"id": scope + "/providers/Microsoft.Quota/usages/" + name, "name": name, "type": "Microsoft.Quota/usages",
			"properties": map[string]any{
				"name":              azureQuotaNameProperty(scope, name),
				"usages":            map[string]any{"value": used, "usagesType": "Individual"},
				"unit":              "Count",
				"isQuotaApplicable": true,
			},
		})
	}
	return items
}

// azureQuotaRequests reports one succeeded request per limit set by PUT,
// for its latest change.
func azureQuotaRequests(s *Server, scope string) []map[string]any {
	stored := s.azureQuotaStored(scope)
	items := make([]map[string]any, 0, len(stored))
	for _, rec := range stored {
		id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#"+rec.UpdatedAt.Format(time.RFC3339Nano))).String()
		submitted := rec.UpdatedAt.Format("2006-01-02T15:04:05Z")
		items = append(items, map[string]any{
			"id": scope + "/providers/Microsoft.Quota/quotaRequests/" + id, "name": id, "type": "Microsoft.Quota/quotaRequests",
			"properties": map[string]any{
				"provisioningState": "Succeeded",
				"message":           "Request completed",
				"requestSubmitTime": submitted,
				"value": []any{map[string]any{
					"name":              azureQuotaNameProperty(scope, rec.Name),
					"limit":             rec.Properties["limit"],
					"unit":              "Count",
					"provisioningState": "Succeeded",
					"message":           "Request completed",
				}},
			},
		})
	}
	return items
}
