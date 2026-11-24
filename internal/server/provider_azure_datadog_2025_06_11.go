package server

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Datadog on Azure (Microsoft.Datadog 2025-06-11): monitors with their API
// keys, hosts, linked and monitored resources, billing info and
// resubscribe, monitored subscriptions, tag rules and single sign-on
// configurations, served by the shared ARM handler, plus marketplace
// agreements, creation support and the operations list. No Datadog
// organization exists: keys and links are deterministic placeholders.

const (
	azureDatadog20250611APIVersion = "2025-06-11"
	azureDDOGNamespace             = "Microsoft.Datadog"
)

func init() {
	ns := azureDDOGNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors",
		Validate:           azureDDOGMonitorValidate,
		ReadOnlyProperties: azureDDOGMonitorReadOnly,
		WriteOnlyProperties: []string{
			"datadogOrganizationProperties.apiKey", "datadogOrganizationProperties.applicationKey",
			"datadogOrganizationProperties.linkingAuthCode", "datadogOrganizationProperties.linkingClientId",
			"datadogOrganizationProperties.redirectUri",
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"listapikeys":   azureDDOGListAPIKeys,
			"getdefaultkey": azureDDOGDefaultKey,
			"setdefaultkey": azureDDOGSetDefaultKey,
		},
		Actions: map[string]azureARMAction{
			"listhosts":              azureDDOGEmptyList,
			"listmonitoredresources": azureDDOGEmptyList,
			"listlinkedresources": func(rec *azureARMResource, _ map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"value": []any{map[string]any{"id": rec.ID, "location": rec.Location}}}
			},
			"refreshsetpasswordlink": func(rec *azureARMResource, _ map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"setPasswordLink": "https://app.datadoghq.com/account/reset/" + azureDataPlatGUID(rec.ID+"/password")}
			},
			"getbillinginfo": azureDDOGBillingInfo,
			"resubscribe": func(rec *azureARMResource, payload map[string]any) (int, any) {
				if sku := azureARMObject(payload["sku"]); len(sku) > 0 && azureARMString(sku["name"]) == "" {
					return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "sku.name is required.")
				}
				return http.StatusOK, rec.fixture()
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/monitoredSubscriptions", Proxy: true,
		ValidateName: azureDataPlatOnlyName("default"),
		Validate:     azureDataPlatEnum("operation", "AddBegin", "AddComplete", "DeleteBegin", "DeleteComplete", "Active"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/tagRules", CreateStatus: http.StatusOK, Proxy: true,
		ValidateName: azureDataPlatOnlyName("default"),
		NoPatch:      true, NoDelete: true,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/singleSignOnConfigurations", Proxy: true,
		Validate: azureDataPlatAll(
			azureDataPlatRequired("singleSignOnState"),
			azureDataPlatEnum("singleSignOnState", "Initial", "Enable", "Disable", "Existing"),
		),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if azureARMString(rec.Properties["singleSignOnState"]) != "Enable" {
				return nil
			}
			return map[string]any{"singleSignOnUrl": "https://app.datadoghq.com/account/saml/" + azureDataPlatGUID(rec.ID)}
		},
		NoPatch: true, NoDelete: true,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "agreements", CreateStatus: http.StatusOK, Subscription: true,
		ValidateName: azureDataPlatOnlyName("default"),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := map[string]any{
				"licenseTextLink":   "https://www.datadoghq.com/legal/eula/",
				"privacyPolicyLink": "https://www.datadoghq.com/legal/privacy/",
				"retrieveDatetime":  rec.UpdatedAt.Format(time.RFC3339),
				"signature":         azureDataPlatGUID(rec.ID),
			}
			// The Datadog marketplace offer, unless the client names another.
			for key, value := range map[string]string{"publisher": "datadog1591740804488", "product": "dd_liftr_v2", "plan": "payg_v2"} {
				if azureARMString(rec.Properties[key]) == "" {
					props[key] = value
				}
			}
			return props
		},
		NoPatch: true, NoDelete: true,
	})
	registerAzureARMQueryView(ns, "subscriptionStatuses", func(_ *Server, scope string, _ url.Values) []map[string]any {
		// ponytail: creation is always supported; the datadogOrganizationId
		// filter is accepted and ignored.
		if !strings.HasPrefix(strings.ToLower(scope), "/subscriptions/") {
			return nil
		}
		return []map[string]any{{
			"id": scope + "/providers/" + ns + "/subscriptionStatuses/default", "name": "default",
			"properties": map[string]any{"name": strings.TrimPrefix(scope, "/subscriptions/"), "creationSupported": true},
		}}
	})
	registerAzureARMView(ns, "operations", azureDataPlatOperations(ns, "Microsoft Datadog",
		"monitors/read", "monitors/write", "monitors/delete",
		"monitors/listApiKeys/action", "monitors/getDefaultKey/action", "monitors/setDefaultKey/action",
		"monitors/listHosts/action", "monitors/listLinkedResources/action", "monitors/listMonitoredResources/action",
		"monitors/refreshSetPasswordLink/action", "monitors/getBillingInfo/action", "monitors/resubscribe/action",
		"monitors/monitoredSubscriptions/read", "monitors/monitoredSubscriptions/write", "monitors/monitoredSubscriptions/delete",
		"monitors/tagRules/read", "monitors/tagRules/write",
		"monitors/singleSignOnConfigurations/read", "monitors/singleSignOnConfigurations/write",
		"agreements/read", "agreements/write", "subscriptionStatuses/read"))
}

func azureDDOGMonitorValidate(payload map[string]any) string {
	if azureARMString(azureARMObject(payload["sku"])["name"]) == "" {
		return "sku.name is required."
	}
	if msg := azureDataPlatEnum("monitoringStatus", "Enabled", "Disabled")(payload); msg != "" {
		return msg
	}
	email := azureARMString(azureDataPlatLookup(azureARMObject(payload["properties"]), "userInfo.emailAddress"))
	if email != "" && !strings.Contains(email, "@") {
		return "properties.userInfo.emailAddress must be an email address."
	}
	return ""
}

// azureDDOGMonitorReadOnly reports the monitor's marketplace subscription
// as active and names its Datadog organization after the monitor.
func azureDDOGMonitorReadOnly(rec *azureARMResource) map[string]any {
	org := azureARMMerge(azureARMObject(rec.Properties["datadogOrganizationProperties"]), nil)
	for _, secret := range []string{"apiKey", "applicationKey", "linkingAuthCode", "linkingClientId", "redirectUri"} {
		delete(org, secret)
	}
	name := azureARMString(org["name"])
	if name == "" {
		name = rec.Name
	}
	props := map[string]any{
		"marketplaceSubscriptionStatus": "Active",
		"liftrResourceCategory":         "MonitorLogs",
		"liftrResourcePreference":       1,
		"datadogOrganizationProperties": azureARMMerge(org, map[string]any{"name": name, "id": azureDataPlatGUID(rec.ID + "/organization")}),
	}
	if _, present := rec.Properties["monitoringStatus"]; !present {
		props["monitoringStatus"] = "Enabled"
	}
	return props
}

// azureDDOGKeyType is the type of the stored default API key a client set,
// kept under its monitor (so deleting the monitor removes it) but not an ARM
// resource.
const azureDDOGKeyType = azureDDOGNamespace + "/monitors/apiKeys"

// azureDDOGAPIKey is the API key every monitor starts with.
func azureDDOGAPIKey(rec *azureARMResource) map[string]any {
	return map[string]any{
		"name": "Default", "key": strings.ReplaceAll(azureDataPlatGUID(rec.ID+"/apikey"), "-", ""),
		"createdBy": "stackyard", "created": rec.CreatedAt.Format(time.RFC3339),
	}
}

// azureDDOGKeysLocked returns the monitor's API keys, default first: the
// one a client set with setDefaultKey, if any, then the original. Caller
// holds providerStorageMu.
func (s *Server) azureDDOGKeysLocked(rec *azureARMResource) []any {
	original := azureDDOGAPIKey(rec)
	set := s.azureDataPlatStoredLocked(rec.ID+"/apiKeys/default", azureDDOGKeyType)
	if set == nil || set.Properties["key"] == original["key"] {
		return []any{original}
	}
	return []any{set.Properties, original}
}

func azureDDOGListAPIKeys(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return http.StatusOK, map[string]any{"value": s.azureDDOGKeysLocked(rec)}
}

func azureDDOGDefaultKey(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return http.StatusOK, s.azureDDOGKeysLocked(rec)[0]
}

// azureDDOGSetDefaultKey makes the key it is given the monitor's default.
func azureDDOGSetDefaultKey(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	key := azureARMString(payload["key"])
	if strings.TrimSpace(key) == "" || len(key) > 256 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "key is required and must be at most 256 characters.")
	}
	name := azureARMString(payload["name"])
	if len(name) > 256 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name must be at most 256 characters.")
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	monitor := s.azureDataPlatStoredLocked(rec.ID, azureDDOGNamespace+"/monitors")
	if monitor == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The monitor '"+rec.ID+"' was not found.")
	}
	s.azureDataPlatPutChildLocked(monitor, azureDDOGKeyType, "apiKeys", "default", map[string]any{
		"name": name, "key": key, "createdBy": "stackyard", "created": time.Now().UTC().Format(time.RFC3339),
	}, 1)
	return http.StatusOK, nil
}

func azureDDOGEmptyList(*azureARMResource, map[string]any) (int, any) {
	return http.StatusOK, map[string]any{"value": []any{}}
}

func azureDDOGBillingInfo(rec *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{
		"marketplaceSaasInfo": map[string]any{
			"marketplaceSubscriptionId": azureDataPlatGUID(rec.ID + "/saas"), "marketplaceName": rec.Name,
			"marketplaceStatus": "Subscribed", "billedAzureSubscriptionId": rec.SubscriptionID, "subscribed": true,
		},
		"partnerBillingEntity": map[string]any{"id": azureDataPlatGUID(rec.ID + "/organization"), "name": rec.Name, "partnerEntityUri": "https://app.datadoghq.com"},
	}
}
