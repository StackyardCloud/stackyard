package server

import (
	"net/http"
	"slices"
	"strings"
	"time"
)

const azureDesktopVirtualization20251010APIVersion = "2025-10-10"

const avdNamespace = "Microsoft.DesktopVirtualization"

// Microsoft.DesktopVirtualization (Azure Virtual Desktop), served by the shared
// ARM handler. No session host agent runs: session hosts and user sessions
// can't be created through ARM, so they list empty until something stores
// them, and registration tokens are derived from the host pool.
func init() {
	ns := avdNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "hostPools",
		Validate:            avdValidateHostPool,
		WriteOnlyProperties: []string{"registrationInfo.token"},
		Actions: map[string]azureARMAction{
			"retrieveregistrationtoken":  avdRetrieveRegistrationToken,
			"listregistrationtokens":     avdListRegistrationTokens,
			"importappattachpackageinfo": avdImportAppAttachPackageInfo,
			"expandmsiximage":            avdExpandMsixImage,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "hostPools/msixPackages", Proxy: true,
		Validate: func(payload map[string]any) string {
			return azureARMRequireProperties(azureARMObject(payload["properties"]), "imagePath", "packageName", "packageFamilyName")
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "hostPools/sessionHosts", Proxy: true, NoPut: true,
		Validate: avdValidateSessionHost,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "hostPools/sessionHosts/userSessions", Proxy: true, NoPut: true, NoPatch: true,
		Actions: map[string]azureARMAction{
			"disconnect":  func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, nil },
			"sendmessage": avdSendMessage,
		},
	})
	for _, parent := range []string{"hostPools", "workspaces"} {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: parent + "/privateEndpointConnections", Proxy: true,
			Validate: avdValidatePrivateEndpointConnection,
		})
	}
	registerAzureARMView(ns, "hostPools/privateLinkResources", avdPrivateLinkResources("connection"))
	registerAzureARMView(ns, "workspaces/privateLinkResources", avdPrivateLinkResources("feed"))
	registerAzureARMView(ns, "hostPools/userSessions", func(s *Server, scope string) []map[string]any {
		return avdChildFixtures(s, scope+"/sessionHosts/", "/userSessions/")
	})
	registerAzureARMView(ns, "hostPools/scalingPlans", avdScalingPlansForHostPool)

	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "applicationGroups",
		Validate:        avdValidateApplicationGroup,
		DefaultChildren: avdDefaultDesktop,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "applicationGroups/applications", Proxy: true,
		Validate: avdValidateApplication,
	})
	// Desktops come with Desktop application groups; they can only be read and updated.
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "applicationGroups/desktops", Proxy: true, NoPut: true, NoDelete: true,
	})
	registerAzureARMView(ns, "applicationGroups/startMenuItems", avdStartMenuItems)

	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "workspaces", Validate: avdValidateWorkspace})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "scalingPlans", Validate: avdValidateScalingPlan})
	for _, kind := range []string{"pooledSchedules", "personalSchedules"} {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: "scalingPlans/" + kind, Proxy: true,
			Validate: avdValidateSchedule,
		})
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "appAttachPackages",
		Validate: func(payload map[string]any) string {
			if azureARMObject(payload["properties"])["image"] == nil {
				return "properties.image is required."
			}
			return ""
		},
	})
	registerAzureARMView(ns, "operations", azureARMCrudOperations(ns, "Microsoft Desktop Virtualization",
		"hostPools", "applicationGroups", "workspaces", "scalingPlans", "appAttachPackages"))
}

// handleAzureDesktopVirtualization20251010Router keeps the http.go dispatch
// slot; the shared ARM handler (registered above) serves these routes first.
func (s *Server) handleAzureDesktopVirtualization20251010Router(http.ResponseWriter, *http.Request) bool {
	return false
}

// avdOneOf checks an optional (when allowEmpty) enum property.
func avdOneOf(props map[string]any, key string, allowEmpty bool, values ...string) string {
	v, present := props[key]
	if !present && allowEmpty {
		return ""
	}
	if !slices.Contains(values, azureARMString(v)) {
		return "properties." + key + " must be one of " + strings.Join(values, ", ") + "."
	}
	return ""
}

func avdValidateHostPool(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, msg := range []string{
		avdOneOf(props, "hostPoolType", false, "Personal", "Pooled", "BYODesktop"),
		avdOneOf(props, "loadBalancerType", false, "BreadthFirst", "DepthFirst", "Persistent", "MultiplePersistent"),
		avdOneOf(props, "preferredAppGroupType", false, "None", "Desktop", "RailApplications"),
		avdOneOf(props, "personalDesktopAssignmentType", true, "Automatic", "Direct"),
	} {
		if msg != "" {
			return msg
		}
	}
	pooled := azureARMString(props["hostPoolType"]) == "Pooled"
	persistent := strings.HasSuffix(azureARMString(props["loadBalancerType"]), "Persistent")
	if pooled == persistent {
		return "properties.loadBalancerType Persistent is only for Personal host pools, and Personal host pools require it."
	}
	if max, ok := props["maxSessionLimit"].(float64); ok && (max < 1 || max > 999999) {
		return "properties.maxSessionLimit must be between 1 and 999999."
	}
	if exp := azureARMString(azureARMObject(props["registrationInfo"])["expirationTime"]); exp != "" {
		if _, err := time.Parse(time.RFC3339, exp); err != nil {
			return "properties.registrationInfo.expirationTime must be an RFC 3339 time."
		}
	}
	return ""
}

// avdToken returns the host pool's registration token, or nil when none was
// requested with registrationInfo.registrationTokenOperation Update.
func avdToken(rec *azureARMResource) map[string]any {
	info := azureARMObject(rec.Properties["registrationInfo"])
	if azureARMString(info["registrationTokenOperation"]) != "Update" {
		return nil
	}
	exp := azureARMString(info["expirationTime"])
	if exp == "" {
		exp = rec.UpdatedAt.Add(24 * time.Hour).UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"expirationTime": exp, "registrationTokenOperation": "Update",
		"token": "avd-" + azureARMDerivedKey(rec.ID, "registration:"+exp),
	}
}

func avdRetrieveRegistrationToken(rec *azureARMResource, _ map[string]any) (int, any) {
	if token := avdToken(rec); token != nil {
		return http.StatusOK, token
	}
	return http.StatusOK, map[string]any{"registrationTokenOperation": "None"}
}

func avdListRegistrationTokens(rec *azureARMResource, _ map[string]any) (int, any) {
	value := []any{}
	if token := avdToken(rec); token != nil {
		value = append(value, map[string]any{"expirationTime": token["expirationTime"], "token": token["token"]})
	}
	return http.StatusOK, map[string]any{"value": value}
}

// avdImportAppAttachPackageInfo describes the package at path as an MSIX
// package named after the file.
func avdImportAppAttachPackageInfo(rec *azureARMResource, payload map[string]any) (int, any) {
	path := strings.TrimSpace(azureARMString(payload["path"]))
	arch := azureARMString(payload["packageArchitecture"])
	if path == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "path is required.")
	}
	if arch != "" && !slices.Contains([]string{"ARM", "ARM64", "x86", "x64", "Neutral", "x86a64", "ALL"}, arch) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "packageArchitecture is invalid.")
	}
	info := avdPackageInfo(path)
	info["packageArchitecture"] = arch
	return http.StatusOK, map[string]any{"value": []any{map[string]any{
		"id": rec.ID + "/importAppAttachPackageInfo/" + info["packageAlias"].(string), "name": info["packageAlias"],
		"type": avdNamespace + "/appAttachPackages", "properties": map[string]any{"image": info},
	}}}
}

func avdExpandMsixImage(rec *azureARMResource, payload map[string]any) (int, any) {
	uri := strings.TrimSpace(azureARMString(payload["uri"]))
	if uri == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "uri is required.")
	}
	info := avdPackageInfo(uri)
	return http.StatusOK, map[string]any{"value": []any{map[string]any{
		"id": rec.ID + "/expandMsixImage/" + info["packageAlias"].(string), "name": info["packageAlias"],
		"type": avdNamespace + "/hostPools/expandMsixImage", "properties": info,
	}}}
}

func avdPackageInfo(path string) map[string]any {
	name := path[strings.LastIndexAny(path, `/\`)+1:]
	if dot := strings.LastIndex(name, "."); dot > 0 {
		name = name[:dot]
	}
	alias := strings.ToLower(name)
	return map[string]any{
		"imagePath": path, "packageName": name, "packageAlias": alias, "packageFamilyName": name + "_8wekyb3d8bbwe",
		"packageFullName": name + "_1.0.0.0_x64__8wekyb3d8bbwe", "displayName": name, "packageRelativePath": `\` + name,
		"version": "1.0.0.0", "isRegularRegistration": false, "isActive": false, "packageApplications": []any{},
		"lastUpdated": time.Unix(0, 0).UTC().Format(time.RFC3339),
	}
}

func avdValidateSessionHost(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if v, ok := props["allowNewSession"]; ok {
		if _, isBool := v.(bool); !isBool {
			return "properties.allowNewSession must be a boolean."
		}
	}
	return ""
}

func avdSendMessage(_ *azureARMResource, payload map[string]any) (int, any) {
	if strings.TrimSpace(azureARMString(payload["messageBody"])) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "messageBody is required.")
	}
	return http.StatusOK, nil
}

func avdValidatePrivateEndpointConnection(payload map[string]any) string {
	state := azureARMObject(azureARMObject(payload["properties"])["privateLinkServiceConnectionState"])
	if !slices.Contains([]string{"Pending", "Approved", "Rejected"}, azureARMString(state["status"])) {
		return "properties.privateLinkServiceConnectionState.status must be Pending, Approved or Rejected."
	}
	return ""
}

func avdPrivateLinkResources(group string) azureARMView {
	return func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/privateLinkResources/" + group, "name": group, "type": avdNamespace + "/privateLinkResources",
			"properties": map[string]any{"groupId": group, "requiredMembers": []any{group}, "requiredZoneNames": []any{"privatelink.wvd.microsoft.com"}},
		}}
	}
}

// avdChildFixtures lists stored records whose IDs sit under prefix and contain marker.
func avdChildFixtures(s *Server, prefix, marker string) []map[string]any {
	recs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.HasPrefix(strings.ToLower(rec.ID), strings.ToLower(strings.TrimPrefix(prefix, "/azure"))) &&
			strings.Contains(strings.ToLower(rec.ID), strings.ToLower(marker))
	})
	out := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		out = append(out, rec.fixture())
	}
	return out
}

func avdScalingPlansForHostPool(s *Server, scope string) []map[string]any {
	hostPool := strings.ToLower(scope)
	recs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		if !strings.EqualFold(rec.Type, avdNamespace+"/scalingPlans") {
			return false
		}
		refs, _ := rec.Properties["hostPoolReferences"].([]any)
		return slices.ContainsFunc(refs, func(ref any) bool {
			return strings.ToLower(azureARMString(azureARMObject(ref)["hostPoolArmPath"])) == hostPool
		})
	})
	out := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		out = append(out, rec.fixture())
	}
	return out
}

func avdValidateApplicationGroup(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureARMRequireProperties(props, "hostPoolArmPath"); msg != "" {
		return msg
	}
	return avdOneOf(props, "applicationGroupType", false, "RemoteApp", "Desktop")
}

// avdDefaultDesktop gives a Desktop application group its SessionDesktop.
func avdDefaultDesktop(parent *azureARMResource) []*azureARMResource {
	if azureARMString(parent.Properties["applicationGroupType"]) != "Desktop" {
		return nil
	}
	typ, _ := lookupAzureARMResourceType(avdNamespace, "applicationGroups/desktops")
	return []*azureARMResource{azureARMChild(parent, typ, "SessionDesktop", map[string]any{
		"friendlyName": "SessionDesktop", "objectId": azureARMDerivedKey(parent.ID, "desktop"),
	})}
}

func avdValidateApplication(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := avdOneOf(props, "commandLineSetting", false, "DoNotAllow", "Allow", "Require"); msg != "" {
		return msg
	}
	if azureARMString(props["applicationType"]) != "MsixApplication" && strings.TrimSpace(azureARMString(props["filePath"])) == "" {
		return "properties.filePath is required for in-built applications."
	}
	return avdOneOf(props, "applicationType", true, "InBuilt", "MsixApplication")
}

// avdStartMenuItems lists the group's applications as start menu items.
func avdStartMenuItems(s *Server, scope string) []map[string]any {
	recs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, avdNamespace+"/applicationGroups/applications") &&
			strings.HasPrefix(strings.ToLower(rec.ID), strings.ToLower(scope)+"/")
	})
	out := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		out = append(out, map[string]any{
			"id": scope + "/startMenuItems/" + rec.Name, "name": rec.Name, "type": avdNamespace + "/applicationGroups/startMenuItems",
			"properties": map[string]any{
				"appAlias": strings.ToLower(rec.Name), "filePath": rec.Properties["filePath"], "commandLineArguments": rec.Properties["commandLineArguments"],
				"iconPath": rec.Properties["iconPath"], "iconIndex": rec.Properties["iconIndex"],
			},
		})
	}
	return out
}

func avdValidateWorkspace(payload map[string]any) string {
	refs, present := azureARMObject(payload["properties"])["applicationGroupReferences"]
	if _, isList := refs.([]any); present && refs != nil && !isList {
		return "properties.applicationGroupReferences must be an array."
	}
	return ""
}

func avdValidateScalingPlan(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureARMRequireProperties(props, "timeZone"); msg != "" {
		return msg
	}
	if msg := avdOneOf(props, "hostPoolType", true, "Pooled", "Personal"); msg != "" {
		return msg
	}
	refs, _ := props["hostPoolReferences"].([]any)
	for _, ref := range refs {
		if strings.TrimSpace(azureARMString(azureARMObject(ref)["hostPoolArmPath"])) == "" {
			return "properties.hostPoolReferences[].hostPoolArmPath is required."
		}
	}
	return ""
}

var avdDays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

func avdValidateSchedule(payload map[string]any) string {
	days, _ := azureARMObject(payload["properties"])["daysOfWeek"].([]any)
	if len(days) == 0 {
		return "properties.daysOfWeek is required."
	}
	for _, day := range days {
		if !slices.Contains(avdDays, azureARMString(day)) {
			return "properties.daysOfWeek holds an invalid day."
		}
	}
	return ""
}
