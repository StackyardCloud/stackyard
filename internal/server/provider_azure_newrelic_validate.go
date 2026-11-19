package server

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// azureNewRelicEmailPattern is Learn's email address pattern.
var azureNewRelicEmailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@(?:[A-Za-z0-9-]+\.)+[A-Za-z]{2,}$`)

// azureNewRelicEmail checks payload[key] (userEmail when key is empty)
// against Learn's pattern.
func azureNewRelicEmail(payload map[string]any, key string, required bool) string {
	if key == "" {
		key = "userEmail"
	}
	raw, present := payload[key]
	if !present {
		if required {
			return key + " is required."
		}
		return ""
	}
	if email, ok := raw.(string); !ok || !azureNewRelicEmailPattern.MatchString(email) {
		return key + " must be an email address."
	}
	return ""
}

func azureNewRelicIsSaaS(id string) bool {
	lower := strings.ToLower(id)
	return strings.HasPrefix(lower, "/subscriptions/") && strings.Contains(lower, "/providers/microsoft.saas/resources/")
}

func azureNewRelicMonitorValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureNewRelicMonitorProps(payload, props); msg != "" {
		return msg
	}
	// Linking an existing account or organization names it.
	account := azureARMObject(props["newRelicAccountProperties"])
	if props["accountCreationSource"] == "NEWRELIC" && azureARMString(azureARMObject(account["accountInfo"])["accountId"]) == "" {
		return "properties.newRelicAccountProperties.accountInfo.accountId is required to link an existing New Relic account."
	}
	if props["orgCreationSource"] == "NEWRELIC" && azureARMString(azureARMObject(account["organizationInfo"])["organizationId"]) == "" {
		return "properties.newRelicAccountProperties.organizationInfo.organizationId is required to link an existing New Relic organization."
	}
	return ""
}

// azureNewRelicMonitorPatch takes NewRelicMonitorResourceUpdate.
func azureNewRelicMonitorPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	for key := range payload {
		if key != "identity" && key != "tags" && key != "properties" {
			return nil, nil, key + " can't be updated."
		}
	}
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	for key := range props {
		switch key {
		case "accountCreationSource", "newRelicAccountProperties", "orgCreationSource", "planData", "saaSData", "userInfo":
		default:
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	return payload, nil, azureNewRelicMonitorProps(payload, props)
}

func azureNewRelicMonitorProps(payload, props map[string]any) string {
	for _, key := range []string{"accountCreationSource", "orgCreationSource"} {
		if msg := azureLabEnum(props, "properties.", key, []string{"LIFTR", "NEWRELIC"}); msg != "" {
			return msg
		}
	}
	for _, key := range []string{"saaSAzureSubscriptionStatus", "subscriptionState"} {
		if raw, present := props[key]; present {
			if _, ok := raw.(string); !ok {
				return "properties." + key + " must be a string."
			}
		}
	}
	if msg := azureDataBoxObjects(props, "properties.", "newRelicAccountProperties", "planData", "saaSData", "userInfo"); msg != "" {
		return msg
	}
	if msg := azureNewRelicPlan(azureARMObject(props["planData"]), "properties.planData."); msg != "" {
		return msg
	}
	if raw, present := azureARMObject(props["saaSData"])["saaSResourceId"]; present && !azureNewRelicIsSaaS(azureARMString(raw)) {
		return "properties.saaSData.saaSResourceId must be a Microsoft.SaaS/resources resource ID."
	}
	if msg := azureNewRelicUser(azureARMObject(props["userInfo"])); msg != "" {
		return msg
	}
	if msg := azureNewRelicAccountProps(azureARMObject(props["newRelicAccountProperties"])); msg != "" {
		return msg
	}
	return azureNewRelicIdentity(payload)
}

func azureNewRelicPlan(plan map[string]any, path string) string {
	if msg := azureLabEnum(plan, path, "usageType", []string{"PAYG", "COMMITTED"}); msg != "" {
		return msg
	}
	for _, key := range []string{"billingCycle", "planDetails"} {
		if raw, present := plan[key]; present {
			if _, ok := raw.(string); !ok {
				return path + key + " must be a string."
			}
		}
	}
	_, _, msg := azureImpactTime(plan, path, "effectiveDate")
	return msg
}

func azureNewRelicUser(user map[string]any) string {
	for key, most := range map[string]int{"firstName": 50, "lastName": 50, "phoneNumber": 40, "country": -1} {
		raw, present := user[key]
		if !present {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			return "properties.userInfo." + key + " must be a string."
		}
		if most >= 0 && utf8.RuneCountInString(value) > most {
			return "properties.userInfo." + key + " can be at most " + strconv.Itoa(most) + " characters."
		}
	}
	if msg := azureNewRelicEmail(user, "emailAddress", false); msg != "" {
		return "properties.userInfo." + msg
	}
	return ""
}

func azureNewRelicAccountProps(account map[string]any) string {
	const path = "properties.newRelicAccountProperties."
	if raw, present := account["userId"]; present {
		if _, ok := raw.(string); !ok {
			return path + "userId must be a string."
		}
	}
	if msg := azureDataBoxObjects(account, path, "accountInfo", "organizationInfo", "singleSignOnProperties"); msg != "" {
		return msg
	}
	for parent, keys := range map[string][]string{
		"accountInfo":            {"accountId", "ingestionKey", "region"},
		"organizationInfo":       {"organizationId"},
		"singleSignOnProperties": {"enterpriseAppId", "singleSignOnUrl"},
	} {
		for _, key := range keys {
			if raw, present := azureARMObject(account[parent])[key]; present {
				if _, ok := raw.(string); !ok {
					return path + parent + "." + key + " must be a string."
				}
			}
		}
	}
	return azureLabEnum(azureARMObject(account["singleSignOnProperties"]), path+"singleSignOnProperties.", "singleSignOnState", []string{"Initial", "Enable", "Disable", "Existing"})
}

func azureNewRelicIdentity(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "identity"); msg != "" {
		return msg
	}
	identity := azureARMObject(payload["identity"])
	if len(identity) == 0 {
		return ""
	}
	if msg := azureLabEnum(identity, "identity.", "type", []string{"None", "SystemAssigned", "UserAssigned", "SystemAssigned,UserAssigned"}); msg != "" {
		return msg
	}
	if !strings.Contains(azureARMString(identity["type"]), "UserAssigned") {
		return ""
	}
	ids, _ := identity["userAssignedIdentities"].(map[string]any)
	if len(ids) == 0 {
		return "identity.userAssignedIdentities is required for a user-assigned identity."
	}
	for id := range ids {
		if !strings.Contains(strings.ToLower(id), "/providers/microsoft.managedidentity/userassignedidentities/") {
			return "identity.userAssignedIdentities keys must be user-assigned identity IDs."
		}
	}
	return ""
}

// azureNewRelicTagRule checks a tag rule's log and metric rules.
func azureNewRelicTagRule(props map[string]any) string {
	if msg := azureDataBoxObjects(props, "properties.", "logRules", "metricRules"); msg != "" {
		return msg
	}
	logs, metrics := azureARMObject(props["logRules"]), azureARMObject(props["metricRules"])
	for _, key := range []string{"sendAadLogs", "sendActivityLogs", "sendSubscriptionLogs"} {
		if msg := azureLabEnum(logs, "properties.logRules.", key, []string{"Enabled", "Disabled"}); msg != "" {
			return msg
		}
	}
	if msg := azureLabEnum(metrics, "properties.metricRules.", "sendMetrics", []string{"Enabled", "Disabled"}); msg != "" {
		return msg
	}
	if msg := azureNewRelicEmail(metrics, "userEmail", false); msg != "" {
		return "properties.metricRules." + msg
	}
	for path, rules := range map[string]map[string]any{"properties.logRules.": logs, "properties.metricRules.": metrics} {
		if msg := azureNewRelicFilteringTags(rules, path); msg != "" {
			return msg
		}
	}
	return ""
}

func azureNewRelicFilteringTags(rules map[string]any, path string) string {
	raw, present := rules["filteringTags"]
	if !present {
		return ""
	}
	list, ok := raw.([]any)
	if !ok {
		return path + "filteringTags must be an array."
	}
	for _, rawTag := range list {
		tag, ok := rawTag.(map[string]any)
		if !ok {
			return path + "filteringTags items must be objects."
		}
		for _, key := range []string{"name", "value"} {
			if v, present := tag[key]; present {
				if _, ok := v.(string); !ok {
					return path + "filteringTags[]." + key + " must be a string."
				}
			}
		}
		if msg := azureLabEnum(tag, path+"filteringTags[].", "action", []string{"Include", "Exclude"}); msg != "" {
			return msg
		}
	}
	return ""
}

// azureNewRelicTagRulePatch takes TagRuleUpdate: the log and metric rules.
func azureNewRelicTagRulePatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	for key := range props {
		if key != "logRules" && key != "metricRules" {
			return nil, nil, "properties." + key + " can't be updated."
		}
	}
	return payload, nil, azureNewRelicTagRule(props)
}

var azureNewRelicSubscriptionID = regexp.MustCompile(`^(?i:/subscriptions/)?[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func azureNewRelicSubscriptionsValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureLabEnum(props, "properties.", "patchOperation", []string{"AddBegin", "AddComplete", "DeleteBegin", "DeleteComplete", "Active"}); msg != "" {
		return msg
	}
	// Add and delete name the subscriptions they change; on PATCH the
	// stored list would otherwise stand in for a missing one.
	if op := azureARMString(props["patchOperation"]); op != "" && op != "Active" && len(azureARMSlice(props["monitoredSubscriptionList"])) == 0 {
		return "properties.monitoredSubscriptionList is required for " + op + "."
	}
	return azureRecommendationsItems(props, "monitoredSubscriptionList", func(item map[string]any) string {
		if !azureNewRelicSubscriptionID.MatchString(azureARMString(item["subscriptionId"])) {
			return "properties.monitoredSubscriptionList[].subscriptionId must be a subscription ID."
		}
		return ""
	})
}
