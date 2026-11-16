package server

import (
	"regexp"
	"slices"
	"strings"
)

// Microsoft.Maintenance configurations, their assignments to resources,
// subscriptions and resource groups, on-demand update runs and the public
// SQL maintenance windows, served by the shared ARM handler. No update
// ever becomes pending, so the updates list is always empty and an update
// run completes at once.
func init() {
	const ns = "Microsoft.Maintenance"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "maintenanceConfigurations",
		Validate: azureMaintenanceConfigurationValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "configurationAssignments", Extension: true, KeepLocation: true,
		Validate:           azureMaintenanceAssignmentValidate,
		ReadOnlyProperties: azureMaintenanceScopeProperty,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "applyUpdates", Extension: true,
		ValidateScope: azureMaintenanceResourceScope,
		ValidateName:  azureSingletonName("default"),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := azureMaintenanceScopeProperty(rec)
			props["status"] = "Completed"
			props["lastUpdateTime"] = rec.UpdatedAt.Format("2006-01-02T15:04:05Z")
			return props
		},
	})
	registerAzureARMView(ns, "publicMaintenanceConfigurations", azureMaintenancePublicConfigurations)
	registerAzureARMView(ns, "updates", func(*Server, string) []map[string]any { return nil })
}

var (
	azureMaintenanceScopes   = []string{"Host", "Resource", "OSImage", "Extension", "InGuestPatch", "SQLDB", "SQLManagedInstance"}
	azureMaintenanceDateTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}(:\d{2})?$`)
	azureMaintenanceDuration = regexp.MustCompile(`^\d{2}:\d{2}(:\d{2})?$`)
)

func azureMaintenanceConfigurationValidate(payload map[string]any) string {
	for _, check := range []func(map[string]any) string{
		azureEnumRequired("maintenanceScope", azureMaintenanceScopes...),
		azureEnumProperty("visibility", "Custom", "Public"),
	} {
		if msg := check(payload); msg != "" {
			return msg
		}
	}
	props := azureARMObject(payload["properties"])
	guestPatch := props["maintenanceScope"] == "InGuestPatch"
	window, hasWindow := props["maintenanceWindow"]
	if hasWindow {
		if msg := azureMaintenanceWindowValidate(azureARMObject(window), guestPatch); msg != "" {
			return msg
		}
	}
	if !guestPatch {
		return ""
	}
	// Guest patching runs on a schedule with patch settings.
	if !hasWindow {
		return "properties.maintenanceWindow is required for InGuestPatch configurations."
	}
	if !slices.Contains([]string{"User", "Platform"}, azureARMString(azureARMObject(props["extensionProperties"])["InGuestPatchMode"])) {
		return "properties.extensionProperties.InGuestPatchMode must be User or Platform for InGuestPatch configurations."
	}
	patches, ok := props["installPatches"].(map[string]any)
	if !ok {
		return "properties.installPatches is required for InGuestPatch configurations."
	}
	if raw, present := patches["rebootSetting"]; present && !slices.Contains([]string{"IfRequired", "Never", "Always"}, azureARMString(raw)) {
		return "properties.installPatches.rebootSetting must be IfRequired, Never or Always."
	}
	return ""
}

// azureMaintenanceWindowValidate checks a window's formats; guest patching
// also needs its time zone and recurrence.
func azureMaintenanceWindowValidate(window map[string]any, guestPatch bool) string {
	if !azureMaintenanceDateTime.MatchString(azureARMString(window["startDateTime"])) {
		return "properties.maintenanceWindow.startDateTime must be in the form yyyy-mm-dd hh:mm."
	}
	if raw, present := window["expirationDateTime"]; present && !azureMaintenanceDateTime.MatchString(azureARMString(raw)) {
		return "properties.maintenanceWindow.expirationDateTime must be in the form yyyy-mm-dd hh:mm."
	}
	if raw, present := window["duration"]; present && !azureMaintenanceDuration.MatchString(azureARMString(raw)) {
		return "properties.maintenanceWindow.duration must be in the form hh:mm."
	}
	for _, key := range []string{"timeZone", "recurEvery"} {
		if raw, present := window[key]; (guestPatch || present) && strings.TrimSpace(azureARMString(raw)) == "" {
			return "properties.maintenanceWindow." + key + " is required."
		}
	}
	return ""
}

func azureMaintenanceAssignmentValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.Contains(strings.ToLower(azureARMString(props["maintenanceConfigurationId"])), "/providers/microsoft.maintenance/maintenanceconfigurations/") {
		return "properties.maintenanceConfigurationId must be a maintenance configuration ID."
	}
	filter := azureARMObject(props["filter"])
	for _, os := range azureARMSlice(filter["osTypes"]) {
		if !slices.Contains([]string{"Windows", "Linux"}, azureARMString(os)) {
			return "properties.filter.osTypes items must be Windows or Linux."
		}
	}
	if raw, present := azureARMObject(filter["tagSettings"])["filterOperator"]; present && !slices.Contains([]string{"All", "Any"}, azureARMString(raw)) {
		return "properties.filter.tagSettings.filterOperator must be All or Any."
	}
	return ""
}

// azureMaintenanceResourceScope accepts only a resource, not a subscription
// or resource group.
func azureMaintenanceResourceScope(scope string) string {
	if !strings.Contains(strings.ToLower(scope), "/providers/") {
		return "Updates can only be applied to a resource."
	}
	return ""
}

// azureMaintenanceScopeProperty reports the scope an extension record
// attaches to as its resourceId.
func azureMaintenanceScopeProperty(rec *azureARMResource) map[string]any {
	i := strings.LastIndex(strings.ToLower(rec.ID), "/providers/microsoft.maintenance/")
	return map[string]any{"resourceId": rec.ID[:i]}
}

// azureMaintenancePublicRegions maps a few regions to the casing public
// SQL configuration names use.
// ponytail: three regions; add more when a test needs one.
var azureMaintenancePublicRegions = map[string]string{"eastus": "EastUS", "westus2": "WestUS2", "westeurope": "WestEurope"}

// azureMaintenancePublicConfigurations lists SQL_Default plus the weekday
// and weekend windows of each region, for SQL databases and managed
// instances.
func azureMaintenancePublicConfigurations(_ *Server, scope string) []map[string]any {
	if strings.Contains(strings.ToLower(scope), "/resourcegroups/") {
		return nil
	}
	item := func(name, location, maintenanceScope, recurEvery string) map[string]any {
		props := map[string]any{"namespace": "Microsoft.Maintenance", "maintenanceScope": maintenanceScope, "visibility": "Public"}
		if recurEvery != "" {
			props["maintenanceWindow"] = map[string]any{
				"startDateTime": "2020-01-01 22:00", "expirationDateTime": "9999-12-31 23:59",
				"duration": "08:00", "timeZone": "UTC", "recurEvery": recurEvery,
			}
		}
		out := map[string]any{
			"id":   scope + "/providers/Microsoft.Maintenance/publicMaintenanceConfigurations/" + name,
			"name": name, "type": "Microsoft.Maintenance/publicMaintenanceConfigurations",
			"properties": props,
		}
		if location != "" {
			out["location"] = location
		}
		return out
	}
	items := []map[string]any{item("SQL_Default", "", "SQLDB", "")}
	regions := make([]string, 0, len(azureMaintenancePublicRegions))
	for region := range azureMaintenancePublicRegions {
		regions = append(regions, region)
	}
	slices.Sort(regions)
	for _, region := range regions {
		prefix := "SQL_" + azureMaintenancePublicRegions[region]
		items = append(items,
			item(prefix+"_DB_1", region, "SQLDB", "Week Monday,Tuesday,Wednesday,Thursday"),
			item(prefix+"_DB_2", region, "SQLDB", "Week Friday,Saturday,Sunday"),
			item(prefix+"_MI_1", region, "SQLManagedInstance", "Week Monday,Tuesday,Wednesday,Thursday"),
			item(prefix+"_MI_2", region, "SQLManagedInstance", "Week Friday,Saturday,Sunday"),
		)
	}
	return items
}
