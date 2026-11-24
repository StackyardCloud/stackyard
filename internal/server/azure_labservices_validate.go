package server

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	azureLabName = regexp.MustCompile(`^[-\w.]{1,100}$`)
	// ponytail: IANA zone names by shape; the emulator ships no tz database.
	azureLabTimeZone     = regexp.MustCompile(`^(UTC|[A-Za-z_]+(/[-+\w]+){1,2})$`)
	azureLabEnableStates = []string{"Enabled", "Disabled"}
	azureLabAccess       = []string{"Public", "Private", "None"}
	azureLabWeekDays     = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	azureLabMaxQuota     = 9999 * time.Hour
)

func azureLabValidateName(name string) string {
	if !azureLabName.MatchString(name) {
		return "Names are 1 to 100 letters, digits, hyphens, underscores or periods."
	}
	return ""
}

// azureLabFirst returns the first non-empty message.
func azureLabFirst(msgs ...string) string {
	for _, msg := range msgs {
		if msg != "" {
			return msg
		}
	}
	return ""
}

func azureLabEnum(m map[string]any, path, key string, values []string) string {
	if raw, present := m[key]; present && !slices.Contains(values, azureARMString(raw)) {
		return path + key + " must be one of " + strings.Join(values, ", ") + "."
	}
	return ""
}

// azureLabLength checks that m[key], when present, is a string of least
// to most characters.
func azureLabLength(m map[string]any, path, key string, least, most int) string {
	raw, present := m[key]
	if !present {
		return ""
	}
	if text, ok := raw.(string); !ok || len(text) < least || len(text) > most {
		return path + key + " must be a string of " + strconv.Itoa(least) + " to " + strconv.Itoa(most) + " characters."
	}
	return ""
}

func azureLabDuration(m map[string]any, path, key string) string {
	raw, present := m[key]
	if !present {
		return ""
	}
	if d, ok := azureSentinelDurationOf(raw); !ok || d > azureLabMaxQuota {
		return path + key + " must be an ISO 8601 duration of at most 9999 hours, such as PT10H."
	}
	return ""
}

func azureLabTime(m map[string]any, path, key string) (time.Time, string) {
	raw, present := m[key]
	if !present {
		return time.Time{}, ""
	}
	at, err := time.Parse(time.RFC3339, azureARMString(raw))
	if err != nil {
		return time.Time{}, path + key + " must be an RFC 3339 date and time."
	}
	return at, ""
}

// azureLabPatchOnly rejects PATCH properties other than keys, then runs
// check on the properties.
func azureLabPatchOnly(payload map[string]any, check func(map[string]any) string, keys ...string) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	for key := range props {
		if !slices.Contains(keys, key) {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	return payload, nil, check(props)
}

func azureLabShutdownCheck(raw any, path string) string {
	profile := azureARMObject(raw)
	return azureLabFirst(
		azureLabEnum(profile, path, "shutdownOnDisconnect", azureLabEnableStates),
		azureLabEnum(profile, path, "shutdownWhenNotConnected", azureLabEnableStates),
		azureLabEnum(profile, path, "shutdownOnIdle", []string{"None", "UserAbsence", "LowUsage"}),
		azureLabDuration(profile, path, "disconnectDelay"),
		azureLabDuration(profile, path, "noConnectDelay"),
		azureLabDuration(profile, path, "idleDelay"),
	)
}

func azureLabConnectionCheck(raw any, path string) string {
	profile := azureARMObject(raw)
	return azureLabFirst(
		azureLabEnum(profile, path, "clientRdpAccess", azureLabAccess),
		azureLabEnum(profile, path, "clientSshAccess", azureLabAccess),
		azureLabEnum(profile, path, "webRdpAccess", azureLabAccess),
		azureLabEnum(profile, path, "webSshAccess", azureLabAccess),
	)
}

func azureLabPlanValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties", "identity"); msg != "" {
		return msg
	}
	if raw, present := azureARMObject(payload["identity"])["type"]; present && raw != "SystemAssigned" {
		return "identity.type must be SystemAssigned."
	}
	return azureLabPlanCheck(azureARMObject(payload["properties"]))
}

func azureLabPlanPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureLabPlanValidate(payload); msg != "" {
		return nil, nil, msg
	}
	return azureLabPatchOnly(payload, azureLabPlanCheck, "allowedRegions", "defaultAutoShutdownProfile", "defaultConnectionProfile",
		"defaultNetworkProfile", "linkedLmsInstance", "sharedGalleryId", "supportInfo")
}

func azureLabPlanCheck(props map[string]any) string {
	if msg := azureDataBoxObjects(props, "properties.", "defaultAutoShutdownProfile", "defaultConnectionProfile", "defaultNetworkProfile", "supportInfo"); msg != "" {
		return msg
	}
	return azureLabFirst(
		azureLabRegions(props, "allowedRegions"),
		azureLabShutdownCheck(props["defaultAutoShutdownProfile"], "properties.defaultAutoShutdownProfile."),
		azureLabConnectionCheck(props["defaultConnectionProfile"], "properties.defaultConnectionProfile."),
		azureLabLength(azureARMObject(props["defaultNetworkProfile"]), "properties.defaultNetworkProfile.", "subnetId", 3, 2000),
		azureLabLength(props, "properties.", "linkedLmsInstance", 3, 2000),
		azureLabLength(props, "properties.", "sharedGalleryId", 3, 2000),
	)
}

// azureLabRegions checks that props[key], when present, lists region names.
func azureLabRegions(props map[string]any, key string) string {
	raw, present := props[key]
	if !present {
		return ""
	}
	if regions, ok := raw.([]any); !ok || slices.ContainsFunc(regions, func(r any) bool { return strings.TrimSpace(azureARMString(r)) == "" }) {
		return "properties." + key + " must be a list of region names."
	}
	return ""
}

func azureLabImageValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	// PUT, like PATCH, changes only what the client may set.
	if _, _, msg := azureLabPatchOnly(payload, azureLabImageCheck, "enabledState", "availableRegions"); msg != "" {
		return msg
	}
	if _, present := azureARMObject(payload["properties"])["enabledState"]; !present {
		return "properties.enabledState is required."
	}
	return ""
}

func azureLabImageCheck(props map[string]any) string {
	return azureLabFirst(azureLabRegions(props, "availableRegions"), azureLabEnum(props, "properties.", "enabledState", azureLabEnableStates))
}

func azureLabValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	for _, key := range []string{"autoShutdownProfile", "connectionProfile", "securityProfile", "virtualMachineProfile"} {
		if _, ok := props[key].(map[string]any); !ok {
			return "properties." + key + " is required."
		}
	}
	const path = "properties.virtualMachineProfile."
	vmProfile := azureARMObject(props["virtualMachineProfile"])
	for _, key := range []string{"createOption", "usageQuota"} {
		if _, present := vmProfile[key]; !present {
			return path + key + " is required."
		}
	}
	ref, ok := vmProfile["imageReference"].(map[string]any)
	if !ok || (azureARMString(ref["id"]) == "" && (azureARMString(ref["publisher"]) == "" || azureARMString(ref["offer"]) == "" || azureARMString(ref["sku"]) == "")) {
		return path + "imageReference needs an image id, or a marketplace publisher, offer and sku."
	}
	if azureARMString(azureARMObject(vmProfile["sku"])["name"]) == "" {
		return path + "sku.name is required."
	}
	admin := azureARMObject(vmProfile["adminUser"])
	if azureARMString(admin["username"]) == "" {
		return path + "adminUser.username is required."
	}
	if vmProfile["createOption"] == "TemplateVM" && azureARMString(admin["password"]) == "" {
		return path + "adminUser.password is required for TemplateVM labs."
	}
	return azureLabFirst(azureLabLength(ref, path+"imageReference.", "id", 3, 2000), azureLabCheck(props))
}

// azureLabPatch takes the properties Labs - Update lists. The image a lab
// is built from can't change, and the OS type is the server's.
func azureLabPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	vmProfile := azureARMObject(azureARMObject(payload["properties"])["virtualMachineProfile"])
	for _, key := range []string{"createOption", "imageReference"} {
		if _, present := vmProfile[key]; present {
			return nil, nil, "properties.virtualMachineProfile." + key + " can't change after the lab is created."
		}
	}
	if _, present := vmProfile["osType"]; present {
		payload = azureARMMerge(payload, nil)
		payload["properties"] = azureARMWithout(azureARMObject(payload["properties"]), "virtualMachineProfile.osType")
	}
	return azureLabPatchOnly(payload, azureLabCheck, "autoShutdownProfile", "connectionProfile", "description", "labPlanId",
		"rosterProfile", "securityProfile", "title", "virtualMachineProfile")
}

// azureLabCheck checks the lab properties that are present.
func azureLabCheck(props map[string]any) string {
	if msg := azureDataBoxObjects(props, "properties.", "autoShutdownProfile", "connectionProfile", "securityProfile",
		"virtualMachineProfile", "networkProfile", "rosterProfile"); msg != "" {
		return msg
	}
	const path = "properties.virtualMachineProfile."
	vmProfile := azureARMObject(props["virtualMachineProfile"])
	if msg := azureDataBoxObjects(vmProfile, path, "imageReference", "sku", "adminUser", "nonAdminUser", "additionalCapabilities"); msg != "" {
		return msg
	}
	if raw, present := azureARMObject(vmProfile["sku"])["capacity"]; present && !azureWholeNumber(raw, 0, 400) {
		return path + "sku.capacity must be a whole number from 0 to 400."
	}
	if raw, present := props["description"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.description must be a string."
		}
	}
	network := azureARMObject(props["networkProfile"])
	return azureLabFirst(
		azureLabShutdownCheck(props["autoShutdownProfile"], "properties.autoShutdownProfile."),
		azureLabConnectionCheck(props["connectionProfile"], "properties.connectionProfile."),
		azureLabEnum(azureARMObject(props["securityProfile"]), "properties.securityProfile.", "openAccess", azureLabEnableStates),
		azureLabLength(props, "properties.", "title", 1, 120),
		azureLabLength(props, "properties.", "labPlanId", 3, 2000),
		azureLabLength(network, "properties.networkProfile.", "subnetId", 3, 2000),
		azureLabLength(network, "properties.networkProfile.", "loadBalancerId", 3, 2000),
		azureLabLength(network, "properties.networkProfile.", "publicIpId", 3, 2000),
		azureLabEnum(vmProfile, path, "createOption", []string{"Image", "TemplateVM"}),
		azureLabEnum(vmProfile, path, "useSharedPassword", azureLabEnableStates),
		azureLabEnum(azureARMObject(vmProfile["additionalCapabilities"]), path+"additionalCapabilities.", "installGpuDrivers", azureLabEnableStates),
		azureLabDuration(vmProfile, path, "usageQuota"),
		azureLabLength(azureARMObject(vmProfile["nonAdminUser"]), path+"nonAdminUser.", "username", 1, 64),
	)
}

// azureLabRegisterMembers registers lab users and schedules.
func azureLabRegisterMembers() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureLabNamespace, Type: "labs/users", Proxy: true,
		ValidateName: azureLabValidateName,
		Validate: func(payload map[string]any) string {
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return msg
			}
			props := azureARMObject(payload["properties"])
			email := azureARMString(props["email"])
			if at := strings.Index(email, "@"); len(email) < 6 || len(email) > 254 || at < 1 || at != strings.LastIndex(email, "@") || at == len(email)-1 {
				return "properties.email must be an email address of 6 to 254 characters."
			}
			return azureLabDuration(props, "properties.", "additionalUsageQuota")
		},
		// Users register through the lab's website, never through the API.
		StateProperties: map[string]any{"invitationState": "NotSent", "registrationState": "NotRegistered", "invitationSent": nil},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"totalUsage": "PT0S"}
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return azureLabPatchOnly(payload, func(props map[string]any) string {
				return azureLabDuration(props, "properties.", "additionalUsageQuota")
			}, "additionalUsageQuota")
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"invite": (*Server).azureLabInvite},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureLabNamespace, Type: "labs/schedules", Proxy: true,
		ValidateName: azureLabValidateName,
		Validate: func(payload map[string]any) string {
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return msg
			}
			props := azureARMObject(payload["properties"])
			for _, key := range []string{"stopAt", "timeZoneId"} {
				if _, present := props[key]; !present {
					return "properties." + key + " is required."
				}
			}
			return azureLabScheduleCheck(props)
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return azureLabPatchOnly(payload, azureLabScheduleCheck, "notes", "recurrencePattern", "startAt", "stopAt", "timeZoneId")
		},
	})
	registerAzureARMView(azureLabNamespace, "skus", func(*Server, string) []map[string]any { return azureLabSKUs() })
	registerAzureARMView(azureLabNamespace, "locations/usages", azureLabUsages)
}

// azureLabScheduleCheck checks the schedule properties that are present.
// ponytail: a PATCH moving only startAt past the stored stopAt isn't caught.
func azureLabScheduleCheck(props map[string]any) string {
	start, msg := azureLabTime(props, "properties.", "startAt")
	stop, stopMsg := azureLabTime(props, "properties.", "stopAt")
	if msg = azureLabFirst(msg, stopMsg); msg != "" {
		return msg
	}
	if !start.IsZero() && !stop.IsZero() && !start.Before(stop) {
		return "properties.startAt must be before properties.stopAt."
	}
	if raw, present := props["timeZoneId"]; present && (len(azureARMString(raw)) > 50 || !azureLabTimeZone.MatchString(azureARMString(raw))) {
		return "properties.timeZoneId must be an IANA time zone ID, such as America/Los_Angeles."
	}
	if msg := azureLabLength(props, "properties.", "notes", 0, 1000); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(props, "properties.", "recurrencePattern"); msg != "" {
		return msg
	}
	raw, present := props["recurrencePattern"]
	if !present {
		return ""
	}
	const path = "properties.recurrencePattern."
	pattern := azureARMObject(raw)
	if _, present := pattern["frequency"]; !present {
		return path + "frequency is required."
	}
	if raw, present := pattern["interval"]; present && !azureWholeNumber(raw, 1, 365) {
		return path + "interval must be a whole number from 1 to 365."
	}
	days, isList := pattern["weekDays"].([]any)
	if _, present := pattern["weekDays"]; present && !isList {
		return path + "weekDays must be a list of days."
	}
	for _, day := range days {
		if !slices.Contains(azureLabWeekDays, azureARMString(day)) {
			return path + "weekDays items must be one of " + strings.Join(azureLabWeekDays, ", ") + "."
		}
	}
	if pattern["frequency"] == "Weekly" && len(days) == 0 {
		return path + "weekDays needs at least one day for a Weekly schedule."
	}
	_, msg = azureLabTime(pattern, path, "expirationDate")
	return azureLabFirst(azureLabEnum(pattern, path, "frequency", []string{"Daily", "Weekly"}), msg)
}

// azureLabInvite marks the user invited; no email is sent.
func (s *Server) azureLabInvite(rec *azureARMResource, payload map[string]any) (int, any) {
	if raw, present := payload["text"]; present {
		if _, ok := raw.(string); !ok {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "text must be a string.")
		}
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	user := s.azureARMResources[azureARMKey(rec.ID)]
	if user == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	updated := *user
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(user.Properties, map[string]any{"invitationState": "Sent", "invitationSent": updated.UpdatedAt.Format(time.RFC3339)})
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	return http.StatusOK, nil
}
