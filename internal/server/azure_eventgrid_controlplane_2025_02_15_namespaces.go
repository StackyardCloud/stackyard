package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Event Grid namespaces: pull-delivery namespace topics and the MQTT broker's
// topic spaces, clients, client groups, permission bindings and CA
// certificates. Namespace topics' event subscriptions are registered with the
// other subscription types.

func egcpRegisterNamespaces(keys map[string]func(*Server, *azureARMResource, map[string]any) (int, any)) {
	ns := egcpNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces",
		ValidateName:       egcpName("namespace", egcpTopicNameRE),
		Validate:           egcpValidateNamespace,
		ReadOnlyProperties: egcpNamespaceProperties,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"listkeys":                      keys["listkeys"],
			"regeneratekey":                 keys["regeneratekey"],
			"validatecustomdomainownership": egcpValidateCustomDomains,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/topics", Proxy: true,
		ValidateName:  egcpName("namespace topic", egcpMQTTNameRE),
		Validate:      egcpValidateNamespaceTopic,
		ServerActions: keys,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := map[string]any{}
			for key, value := range map[string]any{"publisherType": "Custom", "inputSchema": "CloudEventSchemaV1_0", "eventRetentionInDays": 1} {
				if _, set := rec.Properties[key]; !set {
					props[key] = value
				}
			}
			return props
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/topicSpaces", Proxy: true,
		ValidateName: egcpName("topic space", egcpMQTTNameRE),
		Validate:     egcpValidateTopicSpace,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/clients", Proxy: true,
		ValidateName: egcpName("client", egcpMQTTNameRE),
		Validate:     egcpValidateClient,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if _, set := rec.Properties["state"]; set {
				return nil
			}
			return map[string]any{"state": "Enabled"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/clientGroups", Proxy: true,
		ValidateName: egcpName("client group", egcpMQTTNameRE),
		Validate: func(payload map[string]any) string {
			query := azureARMString(azureARMObject(payload["properties"])["query"])
			if query == "" || len(query) > 500 {
				return "properties.query is required and must be at most 500 characters."
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/permissionBindings", Proxy: true,
		ValidateName: egcpName("permission binding", egcpMQTTNameRE),
		Validate:     egcpValidatePermissionBinding,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/caCertificates", Proxy: true,
		ValidateName: egcpName("CA certificate", egcpMQTTNameRE),
		Validate: func(payload map[string]any) string {
			cert := azureARMString(azureARMObject(payload["properties"])["encodedCertificate"])
			if !strings.HasPrefix(strings.TrimSpace(cert), "-----BEGIN CERTIFICATE-----") || len(cert) > 16<<10 {
				return "properties.encodedCertificate must be a PEM certificate of at most 16 KiB."
			}
			return ""
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"issueTimeInUtc":  rec.CreatedAt.Format(time.RFC3339),
				"expiryTimeInUtc": rec.CreatedAt.AddDate(1, 0, 0).Format(time.RFC3339),
			}
		},
	})
}

func egcpValidateNamespace(payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	if !egcpOneOf(sku, "name", "Standard") {
		return "sku.name must be Standard."
	}
	if capacity, present := sku["capacity"]; present && !azureWholeNumber(capacity, 1, 40) {
		return "sku.capacity must be an integer from 1 to 40."
	}
	props := azureARMObject(payload["properties"])
	if !egcpOneOf(props, "publicNetworkAccess", "Enabled", "Disabled", "SecuredByPerimeter") {
		return "properties.publicNetworkAccess must be Enabled, Disabled or SecuredByPerimeter."
	}
	if !egcpOneOf(azureARMObject(props["topicSpacesConfiguration"]), "state", "Enabled", "Disabled") {
		return "properties.topicSpacesConfiguration.state must be Enabled or Disabled."
	}
	return egcpValidateIPRules(props)
}

func egcpNamespaceProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{"topicsConfiguration": map[string]any{"hostname": egcpHost(rec, "")}}
	spaces := azureARMObject(rec.Properties["topicSpacesConfiguration"])
	if spaces["state"] == "Enabled" {
		props["topicSpacesConfiguration"] = map[string]any{"hostname": egcpHost(rec, "ts.")}
	}
	return props
}

func egcpValidateCustomDomains(_ *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	approve := func(config any) []any {
		var out []any
		for _, domain := range azureARMSlice(azureARMObject(config)["customDomains"]) {
			out = append(out, azureARMMerge(azureARMObject(domain), map[string]any{"validationState": "Approved"}))
		}
		return out
	}
	return http.StatusOK, map[string]any{
		"customDomainsForTopicsConfiguration":      approve(rec.Properties["topicsConfiguration"]),
		"customDomainsForTopicSpacesConfiguration": approve(rec.Properties["topicSpacesConfiguration"]),
	}
}

func egcpValidateNamespaceTopic(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch {
	case !egcpOneOf(props, "publisherType", "Custom"):
		return "properties.publisherType must be Custom."
	case !egcpOneOf(props, "inputSchema", "CloudEventSchemaV1_0"):
		return "properties.inputSchema must be CloudEventSchemaV1_0."
	}
	if days, present := props["eventRetentionInDays"]; present && !azureWholeNumber(days, 1, 7) {
		return "properties.eventRetentionInDays must be an integer from 1 to 7."
	}
	return ""
}

func egcpValidateTopicSpace(payload map[string]any) string {
	templates := azureARMSlice(azureARMObject(payload["properties"])["topicTemplates"])
	if len(templates) == 0 || len(templates) > 10 {
		return "properties.topicTemplates must hold 1 to 10 topic templates."
	}
	for i, template := range templates {
		if value := azureARMString(template); value == "" || len(value) > 256 {
			return "properties.topicTemplates[" + strconv.Itoa(i) + "] must be 1-256 characters."
		}
	}
	return ""
}

func egcpValidateClient(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if len(azureARMString(props["authenticationName"])) > 128 {
		return "properties.authenticationName must be at most 128 characters."
	}
	if !egcpOneOf(props, "state", "Enabled", "Disabled") {
		return "properties.state must be Enabled or Disabled."
	}
	auth := azureARMObject(props["clientCertificateAuthentication"])
	if !egcpOneOf(auth, "validationScheme", "SubjectMatchesAuthenticationName", "DnsMatchesAuthenticationName", "UriMatchesAuthenticationName", "IpMatchesAuthenticationName", "EmailMatchesAuthenticationName", "ThumbprintMatch") {
		return "properties.clientCertificateAuthentication.validationScheme is not a supported scheme."
	}
	if len(azureARMObject(props["attributes"])) > 64 {
		return "properties.attributes may hold at most 64 attributes."
	}
	return ""
}

func egcpValidatePermissionBinding(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch {
	case azureARMString(props["topicSpaceName"]) == "":
		return "properties.topicSpaceName is required."
	case azureARMString(props["clientGroupName"]) == "":
		return "properties.clientGroupName is required."
	case props["permission"] != "Publisher" && props["permission"] != "Subscriber":
		return "properties.permission must be Publisher or Subscriber."
	}
	return ""
}
