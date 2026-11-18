package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.AzureTerraform: exports resource groups and the resources the
// shared ARM handler stores as Terraform configuration and import blocks.
// Exports finish synchronously, so they answer 200 with the result and
// never 202.
// ponytail: azurerm blocks carry name, location, resource group and tags
// only, for the types below; per-type attributes (SKUs, settings) are the
// upgrade. azapi blocks carry the stored body for every type.
const (
	azureTerraformNamespace = "Microsoft.AzureTerraform"
	// azureTerraformGroupAPIVersion is the Microsoft.Resources version
	// azapi blocks name for resource groups.
	azureTerraformGroupAPIVersion = "2021-04-01"
)

// azureTerraformAzureRM maps lower-cased ARM types to azurerm resource
// types; resources of other types are skipped unless the target is azapi.
var azureTerraformAzureRM = map[string]string{
	"microsoft.resources/resourcegroups":               "azurerm_resource_group",
	"microsoft.containerregistry/registries":           "azurerm_container_registry",
	"microsoft.logic/workflows":                        "azurerm_logic_app_workflow",
	"microsoft.managedidentity/userassignedidentities": "azurerm_user_assigned_identity",
	"microsoft.maps/accounts":                          "azurerm_maps_account",
	"microsoft.network/applicationsecuritygroups":      "azurerm_application_security_group",
	"microsoft.network/natgateways":                    "azurerm_nat_gateway",
	"microsoft.network/networksecuritygroups":          "azurerm_network_security_group",
	"microsoft.network/publicipaddresses":              "azurerm_public_ip",
	"microsoft.network/routetables":                    "azurerm_route_table",
	"microsoft.network/virtualnetworks":                "azurerm_virtual_network",
	"microsoft.operationalinsights/workspaces":         "azurerm_log_analytics_workspace",
	"microsoft.recoveryservices/vaults":                "azurerm_recovery_services_vault",
}

var (
	// azureTerraformPredicate matches one `field op "value"` term of an
	// export query and the `and` after it, or the end of the query.
	azureTerraformPredicate = regexp.MustCompile(`^\s*(\w+)\s*(==|=~|!=|!~)\s*(?:"([^"]*)"|'([^']*)')\s*(?:(?i:and)\s+|$)`)
	// Terraform labels: names start with a letter or underscore (a name
	// pattern gets a counter appended), and types are snake case.
	azureTerraformLabel = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	azureTerraformType  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

func init() {
	registerAzureARMScopeAction(azureTerraformNamespace, "exportTerraform", azureTerraformExport)
}

// azureTerraformTarget is a resource group or stored resource to export.
type azureTerraformTarget struct {
	id, armType, name, location, parentID, apiVersion string
	tags, body                                        map[string]any
}

func azureTerraformExport(s *Server, scope string, payload map[string]any) (int, any) {
	sub, ok := azureARMSubscriptionScope(scope)
	if !ok {
		return http.StatusBadRequest, azureARMErrorBody("InvalidScope", "Terraform exports run at subscription scope.")
	}
	if msg := azureTerraformValidate(payload); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	targets, errs, status, body := s.azureTerraformTargets(sub, payload)
	if body != nil {
		return status, body
	}
	start := time.Now().UTC()
	props := azureTerraformRender(payload, targets)
	if len(errs) > 0 {
		props["errors"] = errs
	}
	name := uuid.NewString()
	return http.StatusOK, map[string]any{
		"id":   "/subscriptions/" + sub + "/providers/" + azureTerraformNamespace + "/operationStatuses/" + name,
		"name": name, "status": "Succeeded", "percentComplete": 100,
		"startTime": start.Format(time.RFC3339Nano), "endTime": time.Now().UTC().Format(time.RFC3339Nano),
		"properties": props,
	}
}

func azureTerraformValidate(payload map[string]any) string {
	kind := azureARMString(payload["type"])
	switch kind {
	case "ExportResource":
		ids, ok := payload["resourceIds"].([]any)
		if !ok || len(ids) == 0 || slices.ContainsFunc(ids, func(v any) bool { return azureARMString(v) == "" }) {
			return "resourceIds must list at least one resource ID."
		}
	case "ExportResourceGroup":
		if azureARMString(payload["resourceGroupName"]) == "" {
			return "resourceGroupName is required."
		}
	case "ExportQuery":
		if strings.TrimSpace(azureARMString(payload["query"])) == "" {
			return "query is required."
		}
		if table, present := payload["table"]; present && !strings.EqualFold(azureARMString(table), "Resources") {
			return "table must be Resources: the emulator queries only stored resources."
		}
		if raw, present := payload["authorizationScopeFilter"]; present && !slices.Contains([]string{"AtScopeAndBelow", "AtScopeAndAbove", "AtScopeAboveAndBelow", "AtScopeExact"}, azureARMString(raw)) {
			return "authorizationScopeFilter must be one of AtScopeAndBelow, AtScopeAndAbove, AtScopeAboveAndBelow, AtScopeExact."
		}
	default:
		return "type must be one of ExportResource, ExportResourceGroup, ExportQuery."
	}
	for _, key := range []string{"fullProperties", "includeManagedResource", "includeResourceGroup", "includeRoleAssignment", "maskSensitive", "recursive"} {
		if raw, present := payload[key]; present {
			if _, ok := raw.(bool); !ok {
				return key + " must be true or false."
			}
		}
	}
	for key, pattern := range map[string]*regexp.Regexp{"namePattern": azureTerraformLabel, "resourceName": azureTerraformLabel, "resourceType": azureTerraformType} {
		if raw, present := payload[key]; present {
			if v, ok := raw.(string); !ok || !pattern.MatchString(v) {
				return key + " must be a Terraform identifier."
			}
		}
	}
	if raw, present := payload["targetProvider"]; present && raw != "azurerm" && raw != "azapi" {
		return "targetProvider must be azurerm or azapi."
	}
	if msg := azureTerraformStrings(payload, "includeExtensions", func(v string) string {
		if !slices.Contains([]string{"role-assignments", "locks", "diagnostic-settings"}, v) {
			return "includeExtensions items must be role-assignments, locks or diagnostic-settings."
		}
		return ""
	}); msg != "" {
		return msg
	}
	if msg := azureTerraformStrings(payload, "excludeAzureResource", func(v string) string {
		if _, err := regexp.Compile(v); err != nil {
			return "excludeAzureResource items must be regular expressions: " + err.Error()
		}
		return ""
	}); msg != "" {
		return msg
	}
	return azureTerraformStrings(payload, "excludeTerraformResource", func(string) string { return "" })
}

// azureTerraformStrings checks that payload[key], when present, is a list of
// strings each passing check.
func azureTerraformStrings(payload map[string]any, key string, check func(string) string) string {
	raw, present := payload[key]
	if !present {
		return ""
	}
	items, ok := raw.([]any)
	if !ok {
		return key + " must be a list of strings."
	}
	for _, item := range items {
		v, ok := item.(string)
		if !ok {
			return key + " must be a list of strings."
		}
		if msg := check(v); msg != "" {
			return msg
		}
	}
	return ""
}

// azureTerraformTargets resolves what an export names: resource groups
// first, then resources by ID. IDs that name nothing become export errors;
// a missing resource group fails the request.
func (s *Server) azureTerraformTargets(sub string, payload map[string]any) (targets []azureTerraformTarget, errs []any, status int, body any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	var groups []*azureResourceGroup
	var recs []*azureARMResource
	inSub := func(rec *azureARMResource) bool { return strings.EqualFold(rec.SubscriptionID, sub) }
	switch payload["type"] {
	case "ExportResourceGroup":
		name := azureARMString(payload["resourceGroupName"])
		group := s.azureResourceGroups[azureResourceGroupKey(sub, name)]
		if group == nil {
			return nil, nil, http.StatusNotFound, azureARMErrorBody("ResourceGroupNotFound", "Resource group '"+name+"' could not be found.")
		}
		groups = append(groups, group)
		recs = s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
			return inSub(rec) && strings.EqualFold(rec.ResourceGroup, group.Name)
		})
	case "ExportResource":
		recursive := payload["recursive"] == true
		for _, raw := range payload["resourceIds"].([]any) {
			id := strings.TrimSuffix(azureARMString(raw), "/")
			seg := strings.Split(strings.TrimPrefix(id, "/"), "/")
			var found bool
			if len(seg) == 4 && strings.EqualFold(seg[0], "subscriptions") && strings.EqualFold(seg[1], sub) && strings.EqualFold(seg[2], "resourceGroups") {
				if group := s.azureResourceGroups[azureResourceGroupKey(sub, seg[3])]; group != nil {
					found = true
					if !slices.Contains(groups, group) {
						groups = append(groups, group)
					}
				}
			} else if rec := s.azureARMResources[azureARMKey(id)]; rec != nil && inSub(rec) {
				recs, found = append(recs, rec), true
			}
			if !found {
				errs = append(errs, map[string]any{"code": "ResourceNotFound", "message": "The resource '" + id + "' was not found.", "target": id})
				continue
			}
			if recursive {
				prefix := azureARMKey(id + "/")
				recs = append(recs, s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
					return strings.HasPrefix(azureARMKey(rec.ID), prefix)
				})...)
			}
		}
	case "ExportQuery":
		match, msg := azureTerraformQuery(azureARMString(payload["query"]))
		if msg != "" {
			return nil, nil, http.StatusBadRequest, azureARMErrorBody("InvalidQuery", msg)
		}
		recs = s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool { return inSub(rec) && match(rec) })
	}
	if payload["includeResourceGroup"] == true {
		for _, rec := range recs {
			if group := s.azureResourceGroups[azureResourceGroupKey(sub, rec.ResourceGroup)]; group != nil && !slices.Contains(groups, group) {
				groups = append(groups, group)
			}
		}
	}
	sort.Slice(groups, func(i, j int) bool { return azureARMKey(groups[i].id()) < azureARMKey(groups[j].id()) })
	sort.Slice(recs, func(i, j int) bool { return azureARMKey(recs[i].ID) < azureARMKey(recs[j].ID) })
	for _, g := range groups {
		targets = append(targets, azureTerraformTarget{
			id: g.id(), armType: "Microsoft.Resources/resourceGroups", name: g.Name, location: g.Location,
			parentID: "/subscriptions/" + g.SubscriptionID, apiVersion: azureTerraformGroupAPIVersion, tags: g.Tags,
		})
	}
	seen := map[string]bool{}
	for _, rec := range recs {
		if seen[azureARMKey(rec.ID)] {
			continue
		}
		seen[azureARMKey(rec.ID)] = true
		body := map[string]any{"properties": rec.Properties}
		if len(rec.SKU) > 0 {
			body["sku"] = rec.SKU
		}
		if rec.Kind != "" {
			body["kind"] = rec.Kind
		}
		targets = append(targets, azureTerraformTarget{
			id: rec.ID, armType: rec.Type, name: rec.Name, location: rec.Location,
			parentID: azureTerraformParent(rec.ID), apiVersion: rec.APIVersion, tags: rec.Tags, body: body,
		})
	}
	return targets, errs, 0, nil
}

// azureTerraformParent returns the ID a resource sits under: its parent
// resource, its resource group, or the scope of an extension resource.
func azureTerraformParent(id string) string {
	parent := id[:strings.LastIndex(id[:strings.LastIndex(id, "/")], "/")]
	if i := strings.LastIndex(strings.ToLower(parent), "/providers/"); i >= 0 && strings.Count(parent[i+len("/providers/"):], "/") == 0 {
		return parent[:i]
	}
	return parent
}

// azureTerraformQuery compiles the subset of Azure Resource Graph predicates
// the emulator evaluates: `field op "value"` terms joined by `and`, where
// field is id, name, type, location, resourceGroup or subscriptionId and op
// is == or != (case-sensitive) or =~ or !~ (case-insensitive). Like Resource
// Graph, type, location, resourceGroup and subscriptionId compare in lower
// case.
func azureTerraformQuery(query string) (func(*azureARMResource) bool, string) {
	var terms []func(*azureARMResource) bool
	rest := query
	for strings.TrimSpace(rest) != "" {
		m := azureTerraformPredicate.FindStringSubmatch(rest)
		if m == nil {
			return nil, "The emulator evaluates only `field op \"value\"` predicates joined by `and`; got `" + strings.TrimSpace(rest) + "`."
		}
		rest = rest[len(m[0]):]
		field, op, want := m[1], m[2], m[3]+m[4]
		var get func(*azureARMResource) string
		switch strings.ToLower(field) {
		case "id":
			get = func(r *azureARMResource) string { return r.ID }
		case "name":
			get = func(r *azureARMResource) string { return r.Name }
		case "type":
			get = func(r *azureARMResource) string { return strings.ToLower(r.Type) }
		case "location":
			get = func(r *azureARMResource) string { return strings.ToLower(r.Location) }
		case "resourcegroup":
			get = func(r *azureARMResource) string { return strings.ToLower(r.ResourceGroup) }
		case "subscriptionid":
			get = func(r *azureARMResource) string { return strings.ToLower(r.SubscriptionID) }
		default:
			return nil, "The emulator queries only id, name, type, location, resourceGroup and subscriptionId; got " + field + "."
		}
		terms = append(terms, func(r *azureARMResource) bool {
			got := get(r)
			switch op {
			case "==":
				return got == want
			case "!=":
				return got != want
			case "=~":
				return strings.EqualFold(got, want)
			}
			return !strings.EqualFold(got, want)
		})
	}
	return func(r *azureARMResource) bool {
		for _, term := range terms {
			if !term(r) {
				return false
			}
		}
		return true
	}, ""
}

// azureTerraformRender writes the configuration and import blocks for the
// targets that survive the exclusions, and lists those it can't map.
func azureTerraformRender(payload map[string]any, targets []azureTerraformTarget) map[string]any {
	provider := azureARMString(payload["targetProvider"])
	if provider == "" {
		provider = "azurerm"
	}
	prefix := "res-"
	if raw, present := payload["namePattern"]; present {
		prefix = azureARMString(raw)
	}
	var excludeIDs []*regexp.Regexp
	for _, raw := range azureARMSlice(payload["excludeAzureResource"]) {
		excludeIDs = append(excludeIDs, regexp.MustCompile(`(?i)^(?:`+azureARMString(raw)+`)$`))
	}
	excludeTypes := azureARMSlice(payload["excludeTerraformResource"])
	// resourceName and resourceType name the one resource an ExportResource
	// lists, not a resource group includeResourceGroup adds.
	single := ""
	if ids := azureARMSlice(payload["resourceIds"]); payload["type"] == "ExportResource" && len(ids) == 1 {
		single = strings.TrimSuffix(azureARMString(ids[0]), "/")
	}

	var config, imports strings.Builder
	config.WriteString("terraform {\n  required_providers {\n")
	if provider == "azapi" {
		config.WriteString("    azapi = {\n      source = \"Azure/azapi\"\n    }\n  }\n}\nprovider \"azapi\" {\n}\n")
	} else {
		config.WriteString("    azurerm = {\n      source = \"hashicorp/azurerm\"\n    }\n  }\n}\nprovider \"azurerm\" {\n  features {}\n}\n")
	}
	skipped := []any{}
	n := 0
	for _, t := range targets {
		if slices.ContainsFunc(excludeIDs, func(re *regexp.Regexp) bool { return re.MatchString(t.id) }) {
			continue
		}
		tfType := "azapi_resource"
		if provider == "azurerm" {
			tfType = azureTerraformAzureRM[strings.ToLower(t.armType)]
			if custom := azureARMString(payload["resourceType"]); strings.EqualFold(t.id, single) && custom != "" {
				tfType = custom
			}
		}
		if tfType == "" || (provider == "azapi" && t.apiVersion == "") {
			skipped = append(skipped, t.id)
			continue
		}
		if slices.Contains(excludeTypes, any(tfType)) {
			continue
		}
		name := prefix + strconv.Itoa(n)
		if custom := azureARMString(payload["resourceName"]); strings.EqualFold(t.id, single) && custom != "" {
			name = custom
		}
		n++
		config.WriteString("resource " + azureTerraformHCL(tfType) + " " + azureTerraformHCL(name) + " {\n")
		if provider == "azapi" {
			config.WriteString("  type      = " + azureTerraformHCL(t.armType+"@"+t.apiVersion) + "\n")
			config.WriteString("  parent_id = " + azureTerraformHCL(t.parentID) + "\n")
		}
		config.WriteString("  name      = " + azureTerraformHCL(t.name) + "\n")
		if t.location != "" {
			config.WriteString("  location  = " + azureTerraformHCL(t.location) + "\n")
		}
		if provider == "azurerm" && tfType != "azurerm_resource_group" {
			config.WriteString("  resource_group_name = " + azureTerraformHCL(azureTerraformGroupName(t.id)) + "\n")
		}
		if t.tags == nil {
			t.tags = map[string]any{}
		}
		config.WriteString("  tags      = " + azureTerraformHCL(t.tags) + "\n")
		if provider == "azapi" && t.body != nil {
			config.WriteString("  body      = " + azureTerraformHCL(t.body) + "\n")
		}
		config.WriteString("}\n")
		imports.WriteString("import {\n  id = " + azureTerraformHCL(t.id) + "\n  to = " + tfType + "." + name + "\n}\n")
	}
	return map[string]any{"configuration": config.String(), "import": imports.String(), "skippedResources": skipped}
}

// azureTerraformHCL writes v as an HCL expression: JSON is valid HCL for
// strings, numbers, lists and objects, once template sequences are escaped.
func azureTerraformHCL(v any) string {
	raw, _ := json.Marshal(v)
	return strings.NewReplacer("${", "$${", "%{", "%%{").Replace(string(raw))
}

func azureTerraformGroupName(id string) string {
	seg := strings.Split(id, "/")
	for i := 0; i+1 < len(seg); i++ {
		if strings.EqualFold(seg[i], "resourceGroups") {
			return seg[i+1]
		}
	}
	return ""
}
