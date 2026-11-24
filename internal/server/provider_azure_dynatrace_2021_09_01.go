package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/mail"
	"strings"
)

// Azure Native Dynatrace Service (Dynatrace.Observability, api-version
// 2021-09-01) on the shared ARM handler: monitors with their credential,
// SSO and listing actions, single sign-on configurations and tag rules. No
// Dynatrace environment exists; keys are derived from the monitor ID.
const (
	dynaNS                 = "Dynatrace.Observability"
	dynaMaxFilteringTags   = 50
	dynaMaxMonitoredListed = 1000
)

func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: dynaNS, Type: "monitors",
		Validate:            dynaValidateMonitor,
		WriteOnlyProperties: []string{"userInfo"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"marketplaceSubscriptionStatus": "Active", "liftrResourceCategory": "MonitorLogs", "liftrResourcePreference": 0,
				"dynatraceEnvironmentProperties": map[string]any{
					"accountInfo":     map[string]any{"accountId": dynaDerive(rec.ID, "account")[:12], "regionId": "us-east"},
					"environmentInfo": map[string]any{"environmentId": dynaEnvironmentID(rec.ID), "landingURL": "https://" + dynaEnvironmentID(rec.ID) + ".live.dynatrace.com"},
				},
			}
		},
		Actions: map[string]azureARMAction{
			"getaccountcredentials": func(rec *azureARMResource, _ map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"accountId": dynaDerive(rec.ID, "account")[:12], "apiKey": "dt0c01." + dynaDerive(rec.ID, "api"), "regionId": "us-east"}
			},
			"getvmhostpayload": func(rec *azureARMResource, _ map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"environmentId": dynaEnvironmentID(rec.ID), "ingestionKey": "dt0c01." + dynaDerive(rec.ID, "ingest")}
			},
			"listappservices": dynaEmptyList,
			"listhosts":       dynaEmptyList,
			"listlinkableenvironments": func(rec *azureARMResource, payload map[string]any) (int, any) {
				if azureARMString(payload["userPrincipal"]) == "" {
					return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "userPrincipal is missing.")
				}
				return http.StatusOK, map[string]any{"value": []any{map[string]any{
					"environmentId": dynaEnvironmentID(rec.ID), "environmentName": rec.Name, "planData": rec.Properties["planData"],
				}}}
			},
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"getssodetails":          dynaSSODetails,
			"listmonitoredresources": dynaMonitoredResources,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: dynaNS, Type: "monitors/singleSignOnConfigurations", Proxy: true, NoPatch: true, NoDelete: true,
		Validate: dynaValidateSSO,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: dynaNS, Type: "monitors/tagRules", Proxy: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The tag rule set must be named 'default'."
			}
			return ""
		},
		Validate: dynaValidateTagRules,
	})
	registerAzureARMOperations(dynaNS, "Dynatrace", [][2]string{
		{"monitors", "monitors"}, {"monitors/singleSignOnConfigurations", "single sign-on configurations"}, {"monitors/tagRules", "tag rules"},
	})
}

func dynaDerive(id, label string) string {
	sum := sha256.Sum256([]byte(label + "\x00" + azureARMKey(id)))
	return hex.EncodeToString(sum[:])
}

func dynaEnvironmentID(id string) string { return "env" + dynaDerive(id, "environment")[:8] }

func dynaEmptyList(*azureARMResource, map[string]any) (int, any) {
	return http.StatusOK, map[string]any{"value": []any{}}
}

func dynaValidateMonitor(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	plan := azureARMObject(props["planData"])
	if azureARMString(plan["planDetails"]) == "" {
		return "properties.planData.planDetails is missing."
	}
	switch azureARMString(plan["billingCycle"]) {
	case "", "MONTHLY", "WEEKLY":
	default:
		return "properties.planData.billingCycle must be MONTHLY or WEEKLY."
	}
	if email := azureARMString(azureARMObject(props["userInfo"])["emailAddress"]); email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return "properties.userInfo.emailAddress is invalid."
		}
	}
	for _, field := range []string{"monitoringStatus", "marketplaceSubscriptionStatus"} {
		switch azureARMString(props[field]) {
		case "", "Enabled", "Disabled", "Active", "Suspended":
		default:
			return "properties." + field + " is invalid."
		}
	}
	return ""
}

func dynaValidateSSO(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch azureARMString(props["singleSignOnState"]) {
	case "Initial", "Enable", "Disable", "Existing":
	default:
		return "properties.singleSignOnState must be Initial, Enable, Disable or Existing."
	}
	if url := azureARMString(props["singleSignOnUrl"]); url != "" && !strings.HasPrefix(url, "https://") {
		return "properties.singleSignOnUrl must be an https URL."
	}
	return ""
}

func dynaValidateTagRules(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, rules := range []string{"logRules", "metricRules"} {
		tags := azureARMSlice(azureARMObject(props[rules])["filteringTags"])
		if len(tags) > dynaMaxFilteringTags {
			return "properties." + rules + ".filteringTags holds too many tags."
		}
		for _, item := range tags {
			tag := azureARMObject(item)
			if azureARMString(tag["name"]) == "" {
				return "filteringTags[].name is missing."
			}
			switch azureARMString(tag["action"]) {
			case "Include", "Exclude":
			default:
				return "filteringTags[].action must be Include or Exclude."
			}
		}
	}
	return ""
}

// dynaSSODetails reports the monitor's single sign-on state from its
// "default" configuration.
func dynaSSODetails(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if azureARMString(payload["userPrincipal"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "userPrincipal is missing.")
	}
	state, url := "Disabled", ""
	if sso, _ := s.getAzureARMResource(rec.ID+"/singleSignOnConfigurations/default", rec.SubscriptionID, rec.ResourceGroup, rec.ID); sso != nil {
		if st := azureARMString(sso.Properties["singleSignOnState"]); st == "Enable" || st == "Existing" {
			state, url = "Enabled", azureARMString(sso.Properties["singleSignOnUrl"])
		}
	}
	return http.StatusOK, map[string]any{
		"isSsoEnabled": state, "singleSignOnUrl": url, "metadataUrl": "https://" + dynaEnvironmentID(rec.ID) + ".live.dynatrace.com/sso/metadata",
		"aadDomains": []any{}, "adminUsers": []any{},
	}
}

// dynaMonitoredResources lists the subscription's resources the monitor
// covers, other than Dynatrace's own.
func dynaMonitoredResources(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	sub := "/subscriptions/" + strings.ToLower(rec.SubscriptionID) + "/"
	resources := s.azureARMResourcesWhere(func(r *azureARMResource) bool {
		return strings.HasPrefix(azureARMKey(r.ID), sub) && !strings.HasPrefix(strings.ToLower(r.Type), "dynatrace.")
	})
	status := "Enabled"
	if azureARMString(rec.Properties["monitoringStatus"]) == "Disabled" {
		status = "Disabled"
	}
	out := make([]any, 0, min(len(resources), dynaMaxMonitoredListed))
	for _, r := range resources[:min(len(resources), dynaMaxMonitoredListed)] {
		out = append(out, map[string]any{"id": r.ID, "sendingMetrics": status, "sendingLogs": status})
	}
	return http.StatusOK, map[string]any{"value": out}
}
