package server

import (
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
)

// Azure Firewall (Microsoft.Network 2025-05-01) on the shared ARM handler:
// firewalls, firewall policies with rule collection groups, draft policies
// and rule collection groups that deploy replaces the live ones with, IDPS
// signature overrides and the signature catalog, learned prefixes and packet
// capture, plus the FQDN tag and web category catalogs. No traffic flows:
// packet captures capture nothing.

const azureFirewall20250501APIVersion = "2025-05-01"

// afwSignatures is the IDPS signature catalog listIdpsSignatures searches.
var afwSignatures = []map[string]any{
	{"signatureId": 2008983, "severity": 1, "direction": 0, "group": "Malware", "description": "ET MALWARE Suspicious User-Agent", "protocol": "TCP", "sourcePorts": []any{"any"}, "destinationPorts": []any{"80"}},
	{"signatureId": 2010935, "severity": 2, "direction": 1, "group": "Policy", "description": "ET POLICY Suspicious inbound to MSSQL port 1433", "protocol": "TCP", "sourcePorts": []any{"any"}, "destinationPorts": []any{"1433"}},
	{"signatureId": 2024897, "severity": 1, "direction": 2, "group": "Exploit", "description": "ET EXPLOIT Apache Struts RCE attempt", "protocol": "TCP", "sourcePorts": []any{"any"}, "destinationPorts": []any{"$HTTP_PORTS"}},
	{"signatureId": 2027865, "severity": 3, "direction": 0, "group": "Info", "description": "ET INFO Observed DNS Query to .cloud TLD", "protocol": "UDP", "sourcePorts": []any{"any"}, "destinationPorts": []any{"53"}},
	{"signatureId": 2210050, "severity": 3, "direction": 2, "group": "Protocol", "description": "SURICATA STREAM reassembly overlap with different data", "protocol": "TCP", "sourcePorts": []any{"any"}, "destinationPorts": []any{"any"}},
}

func init() {
	ns := "Microsoft.Network"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "azureFirewalls",
		Validate:  afwValidateFirewall,
		Reconcile: afwFirewallAddresses,
		Actions: map[string]azureARMAction{
			"learnedipprefixes": func(*azureARMResource, map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"ipPrefixes": []any{}}
			},
			"packetcapture":          afwPacketCapture,
			"packetcaptureoperation": afwPacketCaptureOperation,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "firewallPolicies",
		Validate: afwValidatePolicy,
		ChildProperties: func(s *Server, rec *azureARMResource) map[string]any {
			groups := []any{}
			for _, group := range s.azureARMResourcesMatchingLocked(func(child *azureARMResource) bool {
				return strings.EqualFold(child.Type, ns+"/firewallPolicies/ruleCollectionGroups") && strings.HasPrefix(azureARMKey(child.ID), azureARMKey(rec.ID)+"/")
			}) {
				groups = append(groups, map[string]any{"id": group.ID})
			}
			return map[string]any{"ruleCollectionGroups": groups}
		},
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			overrides, _ := lookupAzureARMResourceType(ns, "firewallPolicies/signatureOverrides")
			return []*azureARMResource{azureARMChild(parent, overrides, "default", map[string]any{"signatures": map[string]any{}})}
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"deploy":                afwDeploy,
			"listidpssignatures":    afwListSignatures,
			"listidpsfilteroptions": afwListFilterOptions,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "firewallPolicies/ruleCollectionGroups", Proxy: true,
		Validate: afwValidateRuleCollectionGroup,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "firewallPolicies/firewallPolicyDrafts", Proxy: true,
		ValidateName: afwDefaultName, Validate: afwValidatePolicy,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "firewallPolicies/ruleCollectionGroups/ruleCollectionGroupDrafts", Proxy: true,
		ValidateName: afwDefaultName, Validate: afwValidateRuleCollectionGroup,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "firewallPolicies/signatureOverrides", Proxy: true,
		ValidateName: afwDefaultName, Validate: afwValidateOverrides, NoDelete: true,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return payload, nil, afwValidateOverrides(payload)
		},
	})
	registerAzureARMView(ns, "azureFirewallFqdnTags", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, tag := range []string{"WindowsUpdate", "WindowsDiagnostics", "MicrosoftActiveProtectionService", "AppServiceEnvironment", "AzureBackup", "AzureKubernetesService"} {
			out = append(out, map[string]any{
				"id": scope + "/providers/" + ns + "/azureFirewallFqdnTags/" + tag, "name": tag, "type": ns + "/azureFirewallFqdnTags",
				"properties": map[string]any{"fqdnTagName": tag, "provisioningState": "Succeeded"},
			})
		}
		return out
	})
	registerAzureARMView(ns, "azureWebCategories", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, category := range []struct{ name, group string }{{"Gambling", "Liability"}, {"Hacking", "Liability"}, {"SocialNetworking", "Productivity"}, {"News", "GeneralSurfing"}, {"ComputersAndTechnology", "GeneralSurfing"}} {
			out = append(out, map[string]any{
				"id": scope + "/providers/" + ns + "/azureWebCategories/" + category.name, "name": category.name, "type": ns + "/azureWebCategories",
				"properties": map[string]any{"group": category.group},
			})
		}
		return out
	})
}

func afwDefaultName(name string) string {
	if name != "default" {
		return "This resource is a singleton named default."
	}
	return ""
}

func afwValidateFirewall(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	sku := azureARMObject(props["sku"])
	if !egcpOneOf(sku, "name", "AZFW_VNet", "AZFW_Hub") {
		return "properties.sku.name must be AZFW_VNet or AZFW_Hub."
	}
	if !egcpOneOf(sku, "tier", "Basic", "Standard", "Premium") {
		return "properties.sku.tier must be Basic, Standard or Premium."
	}
	if !egcpOneOf(props, "threatIntelMode", "Alert", "Deny", "Off") {
		return "properties.threatIntelMode must be Alert, Deny or Off."
	}
	if policy, present := props["firewallPolicy"]; present && !azureNetworkIDOf(policy, "firewallpolicies") {
		return "properties.firewallPolicy.id must be a firewall policy resource ID."
	}
	if sku["name"] == "AZFW_Hub" {
		if !strings.Contains(strings.ToLower(azureARMString(azureARMObject(props["virtualHub"])["id"])), "/providers/microsoft.network/virtualhubs/") {
			return "A hub firewall needs properties.virtualHub.id."
		}
		return ""
	}
	configs := azureARMSlice(props["ipConfigurations"])
	if len(configs) == 0 {
		return "properties.ipConfigurations is required for a virtual network firewall."
	}
	for i, raw := range configs {
		at := "properties.ipConfigurations[" + strconv.Itoa(i) + "].properties"
		config := azureARMObject(azureARMObject(raw)["properties"])
		subnet := azureARMString(azureARMObject(config["subnet"])["id"])
		if subnet != "" && !strings.HasSuffix(strings.ToLower(subnet), "/subnets/azurefirewallsubnet") {
			return at + ".subnet.id must be a subnet named AzureFirewallSubnet."
		}
		if i == 0 && subnet == "" {
			return at + ".subnet.id is required on the first IP configuration."
		}
		if pip, present := config["publicIPAddress"]; present && !azureNetworkIDOf(pip, "publicipaddresses") {
			return at + ".publicIPAddress.id must be a public IP address resource ID."
		}
	}
	if mgmt := azureARMObject(props["managementIpConfiguration"]); len(mgmt) != 0 {
		subnet := azureARMString(azureARMObject(azureARMObject(mgmt["properties"])["subnet"])["id"])
		if !strings.HasSuffix(strings.ToLower(subnet), "/subnets/azurefirewallmanagementsubnet") {
			return "properties.managementIpConfiguration.properties.subnet.id must be a subnet named AzureFirewallManagementSubnet."
		}
	}
	return ""
}

// afwFirewallAddresses gives the firewall the fourth address of its subnet,
// the first Azure doesn't reserve, as its private IP.
func afwFirewallAddresses(s *Server, _, rec *azureARMResource) (map[string]any, string) {
	configs := azureARMSlice(rec.Properties["ipConfigurations"])
	if len(configs) == 0 {
		return map[string]any{"hubIPAddresses": map[string]any{"privateIPAddress": "10.0.0.132"}}, ""
	}
	out := make([]any, 0, len(configs))
	for i, raw := range configs {
		config := azureARMObject(raw)
		props := azureARMObject(config["properties"])
		if i == 0 {
			subnetID := azureARMString(azureARMObject(props["subnet"])["id"])
			subnet := s.azureARMResources[azureARMKey(subnetID)]
			if subnet == nil {
				return nil, "Subnet " + subnetID + " was not found."
			}
			if prefix, err := netip.ParsePrefix(azureARMString(subnet.Properties["addressPrefix"])); err == nil {
				addr := prefix.Masked().Addr()
				for range 4 {
					addr = addr.Next()
				}
				props = azureARMMerge(props, map[string]any{"privateIPAddress": addr.String(), "privateIPAllocationMethod": "Dynamic"})
			}
		}
		out = append(out, azureARMMerge(config, map[string]any{"properties": props}))
	}
	return map[string]any{"ipConfigurations": out}, ""
}

func afwPacketCapture(_ *azureARMResource, payload map[string]any) (int, any) {
	props := azureARMObject(payload["properties"])
	switch {
	case !azureWholeNumber(props["durationInSeconds"], 30, 1800):
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.durationInSeconds must be an integer from 30 to 1800.")
	case !azureWholeNumber(props["numberOfPacketsToCapture"], 100, 90000):
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.numberOfPacketsToCapture must be an integer from 100 to 90000.")
	case !strings.HasPrefix(azureARMString(props["sasUrl"]), "https://"):
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.sasUrl must be an https URL.")
	case len(azureARMSlice(props["filters"])) > 10:
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.filters may hold at most 10 filters.")
	}
	return http.StatusAccepted, nil
}

// afwPacketCaptureOperation answers Start, Status and Stop.
// ponytail: stateless; Status always reports NotStarted since nothing is
// captured. Track captures per firewall if a client polls for completion.
func afwPacketCaptureOperation(_ *azureARMResource, payload map[string]any) (int, any) {
	status := map[string]string{"Start": "AvailableToRun", "Status": "NotStarted", "Stop": "Success"}[azureARMString(payload["operation"])]
	if status == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "operation must be Start, Status or Stop.")
	}
	return http.StatusOK, map[string]any{"statusCode": status, "message": "No traffic flows through the emulator."}
}

func afwValidatePolicy(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	tier := azureARMString(azureARMObject(props["sku"])["tier"])
	if tier != "" && tier != "Basic" && tier != "Standard" && tier != "Premium" {
		return "properties.sku.tier must be Basic, Standard or Premium."
	}
	if !egcpOneOf(props, "threatIntelMode", "Alert", "Deny", "Off") {
		return "properties.threatIntelMode must be Alert, Deny or Off."
	}
	if base, present := props["basePolicy"]; present && !azureNetworkIDOf(base, "firewallpolicies") {
		return "properties.basePolicy.id must be a firewall policy resource ID."
	}
	if idps := azureARMObject(props["intrusionDetection"]); len(idps) != 0 {
		if tier != "Premium" {
			return "properties.intrusionDetection requires sku.tier Premium."
		}
		if !egcpOneOf(idps, "mode", "Off", "Alert", "Deny") {
			return "properties.intrusionDetection.mode must be Off, Alert or Deny."
		}
	}
	return ""
}

func afwValidateRuleCollectionGroup(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !azureWholeNumber(props["priority"], 100, 65000) {
		return "properties.priority must be an integer from 100 to 65000."
	}
	collections := azureARMSlice(props["ruleCollections"])
	if len(collections) > 100 {
		return "properties.ruleCollections may hold at most 100 rule collections."
	}
	for i, raw := range collections {
		collection := azureARMObject(raw)
		at := "properties.ruleCollections[" + strconv.Itoa(i) + "]"
		if !azureWholeNumber(collection["priority"], 100, 65000) {
			return at + ".priority must be an integer from 100 to 65000."
		}
		ruleType := ""
		switch collection["ruleCollectionType"] {
		case "FirewallPolicyFilterRuleCollection":
			if action := azureARMString(azureARMObject(collection["action"])["type"]); action != "Allow" && action != "Deny" {
				return at + ".action.type must be Allow or Deny."
			}
		case "FirewallPolicyNatRuleCollection":
			ruleType = "NatRule"
			if azureARMString(azureARMObject(collection["action"])["type"]) != "DNAT" {
				return at + ".action.type must be DNAT."
			}
		default:
			return at + ".ruleCollectionType must be FirewallPolicyFilterRuleCollection or FirewallPolicyNatRuleCollection."
		}
		rules := azureARMSlice(collection["rules"])
		if len(rules) > 1000 {
			return at + ".rules may hold at most 1000 rules."
		}
		for j, rule := range rules {
			kind := azureARMString(azureARMObject(rule)["ruleType"])
			ok := kind == ruleType || (ruleType == "" && (kind == "ApplicationRule" || kind == "NetworkRule"))
			if !ok || azureARMString(azureARMObject(rule)["name"]) == "" {
				return at + ".rules[" + strconv.Itoa(j) + "] needs a name and a ruleType this collection allows."
			}
		}
	}
	return ""
}

func afwValidateOverrides(payload map[string]any) string {
	signatures := azureARMObject(azureARMObject(payload["properties"])["signatures"])
	if len(signatures) > 10000 {
		return "properties.signatures may override at most 10000 signatures."
	}
	for id, mode := range signatures {
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			return "properties.signatures keys must be signature IDs."
		}
		if mode != "Off" && mode != "Alert" && mode != "Deny" {
			return "properties.signatures." + id + " must be Off, Alert or Deny."
		}
	}
	return ""
}

// afwDeploy replaces the live policy and rule collection groups with their
// drafts, then removes the drafts.
func afwDeploy(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureARMResources[azureARMKey(rec.ID)] == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The firewall policy "+rec.ID+" was not found.")
	}
	draftID := rec.ID + "/firewallPolicyDrafts/default"
	if draft := s.azureARMResources[azureARMKey(draftID)]; draft != nil {
		s.afwPromoteLocked(rec.ID, draft)
		s.deleteAzureARMTreeLocked(draftID)
	}
	for _, draft := range s.azureARMResourcesMatchingLocked(func(child *azureARMResource) bool {
		return strings.EqualFold(child.Type, "Microsoft.Network/firewallPolicies/ruleCollectionGroups/ruleCollectionGroupDrafts") &&
			strings.HasPrefix(azureARMKey(child.ID), azureARMKey(rec.ID)+"/")
	}) {
		groupID := draft.ID[:strings.LastIndex(strings.ToLower(draft.ID), "/rulecollectiongroupdrafts/")]
		s.afwPromoteLocked(groupID, draft)
		s.deleteAzureARMTreeLocked(draft.ID)
	}
	s.refreshAzureARMParentLocked(rec.ID)
	return http.StatusOK, nil
}

// afwPromoteLocked replaces the live resource's properties with the draft's.
func (s *Server) afwPromoteLocked(liveID string, draft *azureARMResource) {
	live := s.azureARMResources[azureARMKey(liveID)]
	if live == nil {
		return
	}
	updated := *live
	updated.Properties = azureARMMerge(draft.Properties, map[string]any{"provisioningState": "Succeeded"})
	updated.UpdatedAt = draft.UpdatedAt
	s.azureARMResources[azureARMKey(liveID)] = &updated
}

// afwSignatureModes returns the policy's overridden signature modes.
func (s *Server) afwSignatureModes(policyID string) map[string]any {
	if overrides := s.azureARMLookup(policyID + "/signatureOverrides/default"); overrides != nil {
		return azureARMObject(overrides.Properties["signatures"])
	}
	return nil
}

func afwRequirePremium(rec *azureARMResource) (int, any) {
	if azureARMString(azureARMObject(rec.Properties["sku"])["tier"]) != "Premium" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "IDPS signatures require a Premium firewall policy.")
	}
	return 0, nil
}

// afwCatalog returns the signature catalog with each signature's effective mode.
func (s *Server) afwCatalog(policyID string) []map[string]any {
	modes := s.afwSignatureModes(policyID)
	out := make([]map[string]any, 0, len(afwSignatures))
	for _, sig := range afwSignatures {
		mode := modes[strconv.Itoa(sig["signatureId"].(int))]
		if mode == nil {
			mode = "Alert"
		}
		out = append(out, azureARMMerge(sig, map[string]any{"mode": mode, "inheritedFromParentPolicy": false, "lastUpdated": "2026-01-15T00:00:00Z"}))
	}
	return out
}

func afwListSignatures(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if status, body := afwRequirePremium(rec); status != 0 {
		return status, body
	}
	perPage, skip := 10, 0
	if raw, present := payload["resultsPerPage"]; present {
		if !azureWholeNumber(raw, 1, 1000) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "resultsPerPage must be an integer from 1 to 1000.")
		}
		perPage = int(raw.(float64))
	}
	if raw, present := payload["skip"]; present {
		if !azureWholeNumber(raw, 0, 1e9) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "skip must be a non-negative integer.")
		}
		skip = int(raw.(float64))
	}
	search := strings.ToLower(azureARMString(payload["search"]))
	var matched []map[string]any
	for _, sig := range s.afwCatalog(rec.ID) {
		if search != "" && !strings.Contains(strings.ToLower(azureARMString(sig["description"])), search) && !strings.Contains(strconv.Itoa(sig["signatureId"].(int)), search) {
			continue
		}
		if afwFiltersMatch(sig, azureARMSlice(payload["filters"])) {
			matched = append(matched, sig)
		}
	}
	order := azureARMObject(payload["orderBy"])
	if field := azureARMString(order["field"]); field != "" {
		descending := order["order"] == "Descending"
		sort.SliceStable(matched, func(i, j int) bool {
			less := afwFieldText(matched[i], field) < afwFieldText(matched[j], field)
			if descending {
				return afwFieldText(matched[j], field) < afwFieldText(matched[i], field)
			}
			return less
		})
	}
	page := []any{}
	for i := skip; i < len(matched) && len(page) < perPage; i++ {
		page = append(page, matched[i])
	}
	return http.StatusOK, map[string]any{"matchingRecordsCount": len(matched), "signatures": page}
}

// afwFieldText renders a signature field for filtering and sorting; the
// API names fields in title case (Mode, Severity...).
func afwFieldText(sig map[string]any, field string) string {
	value := sig[strings.ToLower(field[:1])+field[1:]]
	if n, ok := value.(int); ok {
		return strconv.Itoa(n)
	}
	return azureARMString(value)
}

func afwFiltersMatch(sig map[string]any, filters []any) bool {
	for _, raw := range filters {
		filter := azureARMObject(raw)
		values := azureARMSlice(filter["values"])
		if len(values) == 0 {
			continue
		}
		have := afwFieldText(sig, azureARMString(filter["field"]))
		found := false
		for _, value := range values {
			found = found || strings.EqualFold(azureARMString(value), have)
		}
		if !found {
			return false
		}
	}
	return true
}

func afwListFilterOptions(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if status, body := afwRequirePremium(rec); status != 0 {
		return status, body
	}
	field := azureARMString(payload["filterName"])
	switch field {
	case "Mode", "Severity", "Direction", "Protocol", "Group":
	default:
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "filterName must be Mode, Severity, Direction, Protocol or Group.")
	}
	seen := map[string]bool{}
	values := []any{}
	for _, sig := range s.afwCatalog(rec.ID) {
		if text := afwFieldText(sig, field); !seen[text] {
			seen[text] = true
			values = append(values, text)
		}
	}
	return http.StatusOK, map[string]any{"filterValues": values}
}
