package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Microsoft.DomainRegistration App Service domains, served by the shared ARM
// handler. No domain is registered with a registrar: a domain is Active at
// once on Azure DNS name servers, which do not answer for it.
func init() {
	const ns = "Microsoft.DomainRegistration"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "domains",
		ReadOnlyProperties: azureDomainProperties,
		ValidateName:       azureDomainValidateName,
		Validate:           azureDomainValidate,
		Actions: map[string]azureARMAction{"renew": func(*azureARMResource, map[string]any) (int, any) {
			return http.StatusNoContent, nil
		}},
		// Registrant contact details and the transfer code are never returned.
		WriteOnlyProperties: append(slices.Clone(azureDomainContacts), "authCode"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "domains/domainOwnershipIdentifiers", Proxy: true,
		Validate: func(payload map[string]any) string {
			if strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])["ownershipId"])) == "" {
				return "properties.ownershipId is required."
			}
			return ""
		},
	})
	registerAzureARMProviderAction(ns, "checkDomainAvailability", azureDomainAvailability)
	registerAzureARMProviderAction(ns, "listDomainRecommendations", func(*Server, map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"value": []any{}}
	})
	registerAzureARMView(ns, "topLevelDomains", func(_ *Server, scope string) []map[string]any {
		if !strings.HasPrefix(scope, "/subscriptions/") || strings.Count(scope, "/") != 2 {
			return nil
		}
		items := make([]map[string]any, 0, len(azureDomainTopLevelDomains))
		for _, tld := range azureDomainTopLevelDomains {
			items = append(items, map[string]any{
				"id": scope + "/providers/" + ns + "/topLevelDomains/" + tld, "name": tld, "type": ns + "/topLevelDomains",
				"properties": map[string]any{"privacy": true},
			})
		}
		return items
	})
	for _, tld := range azureDomainTopLevelDomains {
		registerAzureARMScopeAction(ns, "topLevelDomains/"+tld+"/listAgreements", func(*Server, string, map[string]any) (int, any) {
			return http.StatusOK, map[string]any{"value": []any{
				map[string]any{"agreementKey": "DNRA", "title": "Domain Name Registration Agreement", "url": "https://example.com/legal/agreements/dnra"},
				map[string]any{"agreementKey": "DNPA", "title": "Domain Name Proxy Agreement", "url": "https://example.com/legal/agreements/dnpa"},
			}}
		})
	}
}

var (
	azureDomainTopLevelDomains = []string{"com", "net", "org", "info", "biz", "co.uk", "org.uk", "nl", "in"}
	azureDomainContacts        = []string{"contactAdmin", "contactBilling", "contactRegistrant", "contactTech"}
	azureDomainLabelPattern    = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// azureDomainValidateName accepts a second-level domain under a supported
// top-level domain, such as contoso.com or contoso.co.uk.
func azureDomainValidateName(name string) string {
	name = strings.ToLower(name)
	for _, tld := range azureDomainTopLevelDomains {
		if label, ok := strings.CutSuffix(name, "."+tld); ok && azureDomainLabelPattern.MatchString(label) {
			return ""
		}
	}
	return "Domain names must be a label of letters, digits or hyphens under one of: " + strings.Join(azureDomainTopLevelDomains, ", ") + "."
}

func azureDomainValidate(payload map[string]any) string {
	if msg := azureValidateGlobalLocation(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	for _, key := range azureDomainContacts {
		contact := azureARMObject(props[key])
		for _, field := range []string{"email", "nameFirst", "nameLast", "phone"} {
			if strings.TrimSpace(azureARMString(contact[field])) == "" {
				return "properties." + key + "." + field + " is required."
			}
		}
	}
	consent := azureARMObject(props["consent"])
	if keys, _ := consent["agreementKeys"].([]any); len(keys) == 0 || azureARMString(consent["agreedBy"]) == "" {
		return "properties.consent needs agreementKeys and agreedBy."
	}
	if raw, present := consent["agreedAt"]; present {
		if _, err := time.Parse(time.RFC3339, azureARMString(raw)); err != nil {
			return "properties.consent.agreedAt must be an RFC 3339 time."
		}
	}
	return azureEnumProperty("dnsType", "AzureDns", "DefaultDomainRegistrarDns")(payload)
}

// azureDomainProperties reports an active one-year registration on Azure DNS
// name servers, with the renewal and privacy defaults.
func azureDomainProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"registrationStatus":          "Active",
		"createdTime":                 rec.CreatedAt.Format(time.RFC3339),
		"lastRenewedTime":             rec.CreatedAt.Format(time.RFC3339),
		"expirationTime":              rec.CreatedAt.AddDate(1, 0, 0).Format(time.RFC3339),
		"nameServers":                 []any{"ns1-01.azure-dns.com", "ns2-01.azure-dns.net", "ns3-01.azure-dns.org", "ns4-01.azure-dns.info"},
		"readyForDnsRecordManagement": true,
		"managedHostNames":            []any{},
		"domainNotRenewableReasons":   []any{},
	}
	for key, value := range map[string]any{"autoRenew": true, "privacy": false, "dnsType": "AzureDns"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

// azureDomainAvailability reports a domain unavailable when it is invalid
// or any subscription holds it.
func azureDomainAvailability(s *Server, payload map[string]any) (int, any) {
	name := strings.ToLower(strings.TrimSpace(azureARMString(payload["name"])))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	available := azureDomainValidateName(name) == "" && len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.DomainRegistration/domains") && strings.EqualFold(rec.Name, name)
	})) == 0
	return http.StatusOK, map[string]any{"name": name, "available": available, "domainType": "Regular"}
}
