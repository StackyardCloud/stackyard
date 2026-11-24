package server

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.Network Front Doors (classic, api-version 2019-05-01), served by
// the shared ARM handler. A front door's routing configuration (frontend
// endpoints, backend pools, routing rules, load balancing and health probe
// settings) is embedded in it, as in ARM; frontend endpoints are also served
// read-only at their own URLs. No traffic is routed and nothing is cached.
const azureAfdcNamespace = "Microsoft.Network"

var (
	azureAfdcName = regexp.MustCompile(`^[a-zA-Z0-9]+([-a-zA-Z0-9]?[a-zA-Z0-9])*$`)
	// azureAfdcCollections are the embedded sub-resource lists, with the
	// type segment ARM reports for their items.
	azureAfdcCollections = [][2]string{
		{"frontendEndpoints", "FrontendEndpoints"}, {"backendPools", "BackendPools"}, {"routingRules", "RoutingRules"},
		{"loadBalancingSettings", "LoadBalancingSettings"}, {"healthProbeSettings", "HealthProbeSettings"},
	}
)

// azureAfdcMaxItems bounds each embedded list.
const azureAfdcMaxItems = 100

func init() {
	ns := azureAfdcNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "frontDoors",
		NoPatch:            true,
		ValidateName:       azureAfdcValidateName,
		Validate:           azureAfdcValidate,
		Reconcile:          azureAfdcCheckReferences,
		ReadOnlyProperties: azureAfdcProperties,
		Actions: map[string]azureARMAction{
			"purge":                azureAfdcPurge,
			"validatecustomdomain": azureAfdcValidateCustomDomain,
		},
	})
	registerAzureARMView(ns, "frontDoors/frontendEndpoints", func(s *Server, scope string) []map[string]any {
		fd, _ := s.getAzureARMResource(scope, "", "", "")
		if fd == nil {
			return nil
		}
		var out []map[string]any
		for _, raw := range azureARMSlice(fd.Properties["frontendEndpoints"]) {
			out = append(out, azureARMObject(raw))
		}
		return out
	})
	registerAzureARMProviderAction(ns, "checkFrontDoorNameAvailability", azureAfdcCheckName)
}

func azureAfdcValidateName(name string) string {
	if len(name) < 5 || len(name) > 64 || !azureAfdcName.MatchString(name) {
		return "The front door name is invalid: use 5 to 64 letters, digits and single hyphens, starting and ending with a letter or digit."
	}
	return ""
}

func azureAfdcValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureWorkloadsEnums(props, "properties.", nil, map[string][]string{"enabledState": {"Enabled", "Disabled"}}); msg != "" {
		return msg
	}
	for _, c := range azureAfdcCollections {
		items := azureARMSlice(props[c[0]])
		if len(items) == 0 || len(items) > azureAfdcMaxItems {
			return fmt.Sprintf("properties.%s is missing or invalid: list 1 to %d items.", c[0], azureAfdcMaxItems)
		}
		seen := map[string]bool{}
		for i, raw := range items {
			name := strings.ToLower(azureARMString(azureARMObject(raw)["name"]))
			if name == "" || seen[name] {
				return fmt.Sprintf("properties.%s[%d].name is missing or duplicated.", c[0], i)
			}
			seen[name] = true
		}
	}
	for i, raw := range azureARMSlice(props["frontendEndpoints"]) {
		fe := azureARMObject(azureARMObject(raw)["properties"])
		if strings.TrimSpace(azureARMString(fe["hostName"])) == "" {
			return fmt.Sprintf("properties.frontendEndpoints[%d].properties.hostName is required.", i)
		}
		if msg := azureWorkloadsEnums(fe, fmt.Sprintf("properties.frontendEndpoints[%d].properties.", i), nil, map[string][]string{"sessionAffinityEnabledState": {"Enabled", "Disabled"}}); msg != "" {
			return msg
		}
	}
	for i, raw := range azureARMSlice(props["backendPools"]) {
		backends := azureARMSlice(azureARMObject(azureARMObject(raw)["properties"])["backends"])
		if len(backends) == 0 || len(backends) > azureAfdcMaxItems {
			return fmt.Sprintf("properties.backendPools[%d].properties.backends is missing or invalid: list 1 to %d backends.", i, azureAfdcMaxItems)
		}
		for j, rawBackend := range backends {
			if msg := azureAfdcValidateBackend(azureARMObject(rawBackend), fmt.Sprintf("properties.backendPools[%d].properties.backends[%d].", i, j)); msg != "" {
				return msg
			}
		}
	}
	for i, raw := range azureARMSlice(props["routingRules"]) {
		rule := azureARMObject(azureARMObject(raw)["properties"])
		at := fmt.Sprintf("properties.routingRules[%d].properties.", i)
		if len(azureARMSlice(rule["frontendEndpoints"])) == 0 {
			return at + "frontendEndpoints is required."
		}
		for _, p := range azureARMSlice(rule["acceptedProtocols"]) {
			if !slices.Contains([]string{"Http", "Https"}, azureARMString(p)) {
				return at + "acceptedProtocols must hold Http or Https."
			}
		}
		route := azureARMObject(rule["routeConfiguration"])
		switch azureARMString(route["@odata.type"]) {
		case "#Microsoft.Azure.FrontDoor.Models.FrontdoorForwardingConfiguration":
			if azureARMString(azureARMObject(route["backendPool"])["id"]) == "" {
				return at + "routeConfiguration.backendPool.id is required for forwarding."
			}
		case "#Microsoft.Azure.FrontDoor.Models.FrontdoorRedirectConfiguration":
		default:
			return at + "routeConfiguration.@odata.type is missing or invalid: use the forwarding or redirect configuration type."
		}
	}
	return ""
}

func azureAfdcValidateBackend(b map[string]any, at string) string {
	if strings.TrimSpace(azureARMString(b["address"])) == "" {
		return at + "address is required."
	}
	for key, bounds := range map[string][2]float64{"httpPort": {1, 65535}, "httpsPort": {1, 65535}, "priority": {1, 5}, "weight": {1, 1000}} {
		if raw, ok := b[key]; ok {
			if n, ok := raw.(float64); !ok || n < bounds[0] || n > bounds[1] || n != float64(int(n)) {
				return fmt.Sprintf("%s%s must be an integer from %d to %d.", at, key, int(bounds[0]), int(bounds[1]))
			}
		}
	}
	return azureWorkloadsEnums(b, at, nil, map[string][]string{"enabledState": {"Enabled", "Disabled"}})
}

// azureAfdcCheckReferences requires routing rules to name this front door's
// own frontend endpoints, backend pools and settings.
func azureAfdcCheckReferences(_ *Server, _, rec *azureARMResource) (map[string]any, string) {
	names := map[string]bool{}
	for _, c := range azureAfdcCollections {
		for _, raw := range azureARMSlice(rec.Properties[c[0]]) {
			names[azureARMKey(rec.ID+"/"+c[0]+"/"+azureARMString(azureARMObject(raw)["name"]))] = true
		}
	}
	check := func(at string, ref any) string {
		if id := azureARMString(azureARMObject(ref)["id"]); id != "" && !names[azureARMKey(id)] {
			return at + " refers to " + id + ", which is not part of this front door."
		}
		return ""
	}
	for i, raw := range azureARMSlice(rec.Properties["routingRules"]) {
		rule := azureARMObject(azureARMObject(raw)["properties"])
		for j, fe := range azureARMSlice(rule["frontendEndpoints"]) {
			if msg := check(fmt.Sprintf("properties.routingRules[%d].properties.frontendEndpoints[%d]", i, j), fe); msg != "" {
				return nil, msg
			}
		}
		if msg := check(fmt.Sprintf("properties.routingRules[%d].properties.routeConfiguration.backendPool", i), azureARMObject(rule["routeConfiguration"])["backendPool"]); msg != "" {
			return nil, msg
		}
	}
	for i, raw := range azureARMSlice(rec.Properties["backendPools"]) {
		pool := azureARMObject(azureARMObject(raw)["properties"])
		for _, key := range []string{"loadBalancingSettings", "healthProbeSettings"} {
			if msg := check(fmt.Sprintf("properties.backendPools[%d].properties.%s", i, key), pool[key]); msg != "" {
				return nil, msg
			}
		}
	}
	return nil, ""
}

// azureAfdcProperties gives each embedded item its ID, type and state, and
// the front door its default hostname.
func azureAfdcProperties(rec *azureARMResource) map[string]any {
	out := map[string]any{
		"cname":         strings.ToLower(rec.Name) + ".azurefd.net",
		"frontdoorId":   uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"resourceState": "Enabled",
		"enabledState":  "Enabled",
	}
	if state := azureARMString(rec.Properties["enabledState"]); state != "" {
		out["enabledState"] = state
	}
	for _, c := range azureAfdcCollections {
		items := make([]any, 0)
		for _, raw := range azureARMSlice(rec.Properties[c[0]]) {
			item := azureARMObject(raw)
			name := azureARMString(item["name"])
			itemProps := azureARMMerge(azureARMObject(item["properties"]), map[string]any{"resourceState": "Enabled"})
			if c[0] == "frontendEndpoints" {
				itemProps = azureARMMerge(map[string]any{"customHttpsProvisioningState": "Disabled", "customHttpsProvisioningSubstate": "None"}, itemProps)
			}
			items = append(items, map[string]any{
				"id": rec.ID + "/" + c[0] + "/" + name, "name": name, "type": "Microsoft.Network/Frontdoors/" + c[1], "properties": itemProps,
			})
		}
		out[c[0]] = items
	}
	return out
}

func azureAfdcPurge(_ *azureARMResource, payload map[string]any) (int, any) {
	paths := azureARMSlice(payload["contentPaths"])
	if len(paths) == 0 || len(paths) > azureAfdcMaxItems {
		return azureOracleBad(fmt.Sprintf("contentPaths is missing or invalid: list 1 to %d paths.", azureAfdcMaxItems))
	}
	for i, p := range paths {
		if !strings.HasPrefix(azureARMString(p), "/") {
			return azureOracleBad(fmt.Sprintf("contentPaths[%d] is invalid: it must start with /.", i))
		}
	}
	return http.StatusAccepted, nil
}

// azureAfdcValidateCustomDomain approves any host name; no DNS is checked.
func azureAfdcValidateCustomDomain(_ *azureARMResource, payload map[string]any) (int, any) {
	host := strings.TrimSpace(azureARMString(payload["hostName"]))
	if host == "" {
		return azureOracleBad("hostName is required.")
	}
	return http.StatusOK, map[string]any{"customDomainValidated": true, "reason": "", "message": ""}
}

func azureAfdcCheckName(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	typ := azureARMString(payload["type"])
	if name == "" || (!strings.EqualFold(typ, "Microsoft.Network/frontDoors") && !strings.EqualFold(typ, "Microsoft.Network/frontDoors/frontendEndpoints")) {
		return azureOracleBad("name and type (Microsoft.Network/frontDoors or Microsoft.Network/frontDoors/frontendEndpoints) are required.")
	}
	if msg := azureAfdcValidateName(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailability": "Unavailable", "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Network/frontDoors") && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailability": "Unavailable", "reason": "AlreadyExists", "message": "The name " + name + ".azurefd.net is already in use."}
	}
	return http.StatusOK, map[string]any{"nameAvailability": "Available", "reason": "", "message": ""}
}

// handleAzureFrontDoorServiceFrontDoor20190501Router serves enableHttps and
// disableHttps on a frontend endpoint, which change the item embedded in
// its front door; everything else is the shared ARM handler's.
func (s *Server) handleAzureFrontDoorServiceFrontDoor20190501Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if r.Method != http.MethodPost || !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(seg) != 11 || !strings.EqualFold(seg[2], "resourceGroups") || !strings.EqualFold(seg[4], "providers") ||
		!strings.EqualFold(seg[5], azureAfdcNamespace) || !strings.EqualFold(seg[6], "frontDoors") || !strings.EqualFold(seg[8], "frontendEndpoints") {
		return false
	}
	action := strings.ToLower(seg[10])
	if action != "enablehttps" && action != "disablehttps" {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return true
	}
	change := map[string]any{"customHttpsProvisioningState": "Disabled", "customHttpsProvisioningSubstate": "None"}
	if action == "enablehttps" {
		source := azureARMString(payload["certificateSource"])
		if source != "FrontDoor" && source != "AzureKeyVault" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "certificateSource is missing or invalid: use FrontDoor or AzureKeyVault.")
			return true
		}
		change = map[string]any{"customHttpsProvisioningState": "Enabled", "customHttpsProvisioningSubstate": "CertificateDeployed", "customHttpsConfiguration": payload}
	}
	fdID := "/subscriptions/" + seg[1] + "/resourceGroups/" + seg[3] + "/providers/" + azureAfdcNamespace + "/frontDoors/" + seg[7]
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	fd := s.azureARMResources[azureARMKey(fdID)]
	if fd == nil {
		azureARMNotFound(w, "ParentResourceNotFound", fdID+"/frontendEndpoints/"+seg[9], fdID)
		return true
	}
	items := azureARMSlice(fd.Properties["frontendEndpoints"])
	next := make([]any, len(items))
	found := false
	for i, raw := range items {
		item := azureARMObject(raw)
		if strings.EqualFold(azureARMString(item["name"]), seg[9]) {
			found = true
			item = azureARMMerge(item, map[string]any{"properties": change})
		}
		next[i] = item
	}
	if !found {
		azureARMNotFound(w, "ResourceNotFound", fdID+"/frontendEndpoints/"+seg[9], "")
		return true
	}
	s.azureOracleUpdateLocked(fd.ID, map[string]any{"frontendEndpoints": next})
	w.WriteHeader(http.StatusAccepted)
	return true
}
