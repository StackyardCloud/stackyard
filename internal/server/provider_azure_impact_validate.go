package server

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

var (
	// Name patterns from the Microsoft.Impact REST reference.
	azureImpactName          = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-:]*[a-zA-Z0-9]{3,120}$`)
	azureImpactInsightName   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]{3,180}$`)
	azureImpactConnectorName = regexp.MustCompile(`^[a-zA-Z0-9-]{3,24}$`)
	// azureImpactLogURL is a getUploadToken URL with its SAS token
	// trimmed: a storage account, a container and a blob named for the
	// upload time and an eight-digit hex suffix.
	azureImpactLogURL = regexp.MustCompile(`^https://[a-z0-9]+\.blob\.[a-z0-9.-]+/[a-z0-9-]{3,63}/[0-9]{14}_[0-9a-f]{8}\.gz$`)
)

// azureImpactLogHost is the emulator's stand-in for the Guest Health
// Reporting log storage account.
const azureImpactLogHost = "https://ghrlogupload.blob.stackyard.invalid"

func azureImpactNameError(pattern *regexp.Regexp, kind string) func(string) string {
	return func(name string) string {
		if !pattern.MatchString(name) {
			return kind + " name '" + name + "' must match " + pattern.String() + "."
		}
		return ""
	}
}

// azureImpactTime parses an RFC 3339 date-time property, reporting
// whether it's present.
func azureImpactTime(m map[string]any, path, key string) (time.Time, bool, string) {
	raw, present := m[key]
	if !present || raw == nil {
		return time.Time{}, false, ""
	}
	parsed, err := time.Parse(time.RFC3339, azureARMString(raw))
	if err != nil {
		return time.Time{}, false, path + key + " must be an RFC 3339 date-time."
	}
	return parsed, true, ""
}

func azureImpactStrings(m map[string]any, path, key string) string {
	raw, present := m[key]
	if !present {
		return ""
	}
	list, ok := raw.([]any)
	if !ok || slices.ContainsFunc(list, func(v any) bool { _, ok := v.(string); return !ok }) {
		return path + key + " must be an array of strings."
	}
	return ""
}

func azureImpactValidate(payload map[string]any) string {
	props, ok := payload["properties"].(map[string]any)
	if !ok {
		return "properties is required."
	}
	start, present, msg := azureImpactTime(props, "properties.", "startDateTime")
	if msg != "" {
		return msg
	}
	if !present {
		return "properties.startDateTime is required."
	}
	end, present, msg := azureImpactTime(props, "properties.", "endDateTime")
	if msg != "" {
		return msg
	}
	if present && end.Before(start) {
		return "properties.endDateTime can't be before properties.startDateTime."
	}
	id := strings.ToLower(azureARMString(props["impactedResourceId"]))
	if !strings.HasPrefix(id, "/subscriptions/") || !strings.Contains(id, "/providers/") {
		return "properties.impactedResourceId must be an Azure resource ID."
	}
	category, ok := azureImpactFindCategory(azureARMString(props["impactCategory"]))
	if !ok {
		return "properties.impactCategory must be one of the categories impactCategories lists."
	}
	if msg := azureImpactProperties(props); msg != "" {
		return msg
	}
	return azureImpactCategoryRules(category, props)
}

// azureImpactProperties checks the optional report properties' shapes and
// enums.
func azureImpactProperties(props map[string]any) string {
	for key, values := range map[string][]string{
		"confidenceLevel": {"Low", "Medium", "High"},
		"severity":        {"Critical", "High", "Medium", "Low"},
		"detectionType":   {"BusinessAlert", "MetricsThreshold", "MetricsAnomaly"},
	} {
		if msg := azureLabEnum(props, "properties.", key, values); msg != "" {
			return msg
		}
	}
	for _, key := range []string{"impactDescription", "impactGroupId"} {
		if raw, present := props[key]; present {
			if _, ok := raw.(string); !ok {
				return "properties." + key + " must be a string."
			}
		}
	}
	for _, key := range []string{"durationInSec", "durationMarginInSec"} {
		if raw, present := props[key]; present {
			if n, ok := raw.(float64); !ok || n < 0 {
				return "properties." + key + " must be a number of at least 0."
			}
		}
	}
	if raw, present := props["hitCount"]; present && !azureWholeNumber(raw, 0, 1<<31-1) {
		return "properties.hitCount must be a whole number of at least 0."
	}
	if raw, present := props["ongoingImpact"]; present {
		if _, ok := raw.(bool); !ok {
			return "properties.ongoingImpact must be a boolean."
		}
	}
	if msg := azureImpactStrings(props, "properties.", "armCorrelationIds"); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(props, "properties.", "workload", "clientIncidentDetails", "connectivity", "errorDetails", "additionalProperties"); msg != "" {
		return msg
	}
	if msg := azureLabEnum(azureARMObject(props["workload"]), "properties.workload.", "toolset", []string{"Terraform", "Puppet", "Chef", "SDK", "Ansible", "ARM", "Portal", "Shell", "Other"}); msg != "" {
		return msg
	}
	if msg := azureLabEnum(azureARMObject(props["clientIncidentDetails"]), "properties.clientIncidentDetails.", "clientIncidentSource", []string{"AzureDevops", "ICM", "Jira", "ServiceNow", "Other"}); msg != "" {
		return msg
	}
	if msg := azureImpactConnectivity(azureARMObject(props["connectivity"])); msg != "" {
		return msg
	}
	return azureImpactPerformance(props)
}

func azureImpactConnectivity(conn map[string]any) string {
	if msg := azureLabEnum(conn, "properties.connectivity.", "protocol", []string{"TCP", "UDP", "HTTP", "HTTPS", "RDP", "FTP", "SSH", "Other"}); msg != "" {
		return msg
	}
	if raw, present := conn["port"]; present && !azureWholeNumber(raw, 0, 65535) {
		return "properties.connectivity.port must be a port number."
	}
	if msg := azureDataBoxObjects(conn, "properties.connectivity.", "source", "target"); msg != "" {
		return msg
	}
	for _, key := range []string{"source", "target"} {
		if raw, present := azureARMObject(conn[key])["azureResourceId"]; present && !strings.HasPrefix(azureARMString(raw), "/") {
			return "properties.connectivity." + key + ".azureResourceId must be an Azure resource ID."
		}
	}
	return ""
}

func azureImpactPerformance(props map[string]any) string {
	raw, present := props["performance"]
	if !present {
		return ""
	}
	list, ok := raw.([]any)
	if !ok {
		return "properties.performance must be an array."
	}
	units := []string{"ByteSeconds", "Bytes", "BytesPerSecond", "Cores", "Count", "CountPerSecond", "MilliCores", "MilliSeconds", "NanoCores", "Percent", "Seconds", "Other"}
	for _, rawItem := range list {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return "properties.performance items must be objects."
		}
		if msg := azureLabEnum(item, "properties.performance[].", "unit", units); msg != "" {
			return msg
		}
		for _, key := range []string{"actual", "expected"} {
			if v, present := item[key]; present {
				if _, ok := v.(float64); !ok {
					return "properties.performance[]." + key + " must be a number."
				}
			}
		}
		if rawRange, present := item["expectedValueRange"]; present {
			r, ok := rawRange.(map[string]any)
			lo, loOK := r["min"].(float64)
			hi, hiOK := r["max"].(float64)
			if !ok || !loOK || !hiOK || lo > hi {
				return "properties.performance[].expectedValueRange must have a min no greater than its max."
			}
		}
	}
	return ""
}

// azureImpactCategoryRules enforces the properties a category requires.
func azureImpactCategoryRules(category string, props map[string]any) string {
	if strings.HasPrefix(category, "ARMOperation.") {
		if ids, _ := props["armCorrelationIds"].([]any); len(ids) == 0 {
			return "properties.armCorrelationIds is required for " + category + " impacts."
		}
	}
	if !azureImpactIsHpc(category) {
		return ""
	}
	if strings.TrimSpace(azureARMString(props["impactDescription"])) == "" {
		return "properties.impactDescription is required for Guest Health Reporting impacts."
	}
	extra := azureARMObject(props["additionalProperties"])
	if strings.TrimSpace(azureARMString(extra["PhysicalHostName"])) == "" {
		return "properties.additionalProperties.PhysicalHostName is required for Guest Health Reporting impacts."
	}
	if raw, present := extra["LogUrl"]; present && !azureImpactLogURL.MatchString(azureARMString(raw)) {
		return "INVALID_FORMAT: properties.additionalProperties.LogUrl must be a getUploadToken URL without its SAS token."
	}
	return ""
}

func azureImpactInsightValidate(payload map[string]any) string {
	props, ok := payload["properties"].(map[string]any)
	if !ok {
		return "properties is required."
	}
	if strings.TrimSpace(azureARMString(props["category"])) == "" {
		return "properties.category is required."
	}
	for _, key := range []string{"category", "status", "eventId", "groupId", "insightUniqueId"} {
		if raw, present := props[key]; present {
			if _, ok := raw.(string); !ok {
				return "properties." + key + " must be a string."
			}
		}
	}
	if _, _, msg := azureImpactTime(props, "properties.", "eventTime"); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(props, "properties.", "content", "impact", "additionalDetails"); msg != "" {
		return msg
	}
	impact := azureARMObject(props["impact"])
	for _, key := range []string{"startTime", "endTime"} {
		if _, _, msg := azureImpactTime(impact, "properties.impact.", key); msg != "" {
			return msg
		}
	}
	return ""
}

func azureImpactConnectorValidate(payload map[string]any) string {
	props, ok := payload["properties"].(map[string]any)
	if !ok {
		return "properties is required."
	}
	if _, present := props["connectorType"]; !present {
		return "properties.connectorType is required."
	}
	return azureImpactConnector(payload, props)
}

// azureImpactConnectorPatch takes ConnectorUpdate: the connector type and
// identity.
func azureImpactConnectorPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties", "identity"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	for key := range props {
		if key != "connectorType" {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	return payload, nil, azureImpactConnector(payload, props)
}

func azureImpactConnector(payload, props map[string]any) string {
	if msg := azureLabEnum(props, "properties.", "connectorType", []string{"AzureMonitor"}); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(payload, "", "identity"); msg != "" {
		return msg
	}
	identity := azureARMObject(payload["identity"])
	if len(identity) == 0 {
		return ""
	}
	if msg := azureLabEnum(identity, "identity.", "type", []string{"UserAssigned", "None"}); msg != "" {
		return msg
	}
	ids, _ := identity["userAssignedIdentities"].(map[string]any)
	if identity["type"] == "UserAssigned" && len(ids) == 0 {
		return "identity.userAssignedIdentities is required for a user-assigned identity."
	}
	for id := range ids {
		if !strings.Contains(strings.ToLower(id), "/providers/microsoft.managedidentity/userassignedidentities/") {
			return "identity.userAssignedIdentities keys must be user-assigned identity IDs."
		}
	}
	return ""
}
