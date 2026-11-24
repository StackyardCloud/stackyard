package server

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Azure Event Grid control plane (Microsoft.EventGrid 2025-02-15) on the
// shared ARM handler: custom topics, domains and their topics, system
// topics, partner registrations, namespaces, channels, topics and
// configurations, and Event Grid namespaces with their topics, MQTT topic
// spaces, clients, client groups, permission bindings and CA certificates.
// Event subscriptions, at any scope or under a topic, and the topic-type and
// event-type catalogs are in azure_eventgrid_controlplane_2025_02_15_subscriptions.go.
// Topics publish through the data planes (provider_azure_eventgrid_dataplane_2018_01_01.go
// and provider_azure_eventgrid_namespace_2024_06_01.go).

const (
	azureEventGridControlPlane20250215APIVersion = "2025-02-15"
	egcpNamespace                                = "Microsoft.EventGrid"
)

var (
	egcpTopicNameRE = regexp.MustCompile(`^[a-zA-Z0-9-]{3,50}$`)
	egcpChildNameRE = regexp.MustCompile(`^[a-zA-Z0-9-]{3,50}$`)
	egcpMQTTNameRE  = regexp.MustCompile(`^[a-zA-Z0-9-]{3,50}$`)
)

func init() {
	ns := egcpNamespace
	keys := egcpKeyActions()
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "topics",
		ValidateName:       egcpName("topic", egcpTopicNameRE),
		Validate:           egcpValidateTopic,
		ReadOnlyProperties: egcpEndpointProperties("/api/events"),
		ServerActions:      keys,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "domains",
		ValidateName: egcpName("domain", egcpTopicNameRE),
		Validate:     egcpValidateTopic,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := egcpEndpointProperties("/api/events")(rec)
			for key, value := range map[string]any{"autoCreateTopicWithFirstSubscription": true, "autoDeleteTopicWithLastSubscription": true} {
				if _, set := rec.Properties[key]; !set {
					props[key] = value
				}
			}
			return props
		},
		ServerActions: keys,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "domains/topics", Proxy: true,
		ValidateName: egcpName("domain topic", egcpChildNameRE),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "systemTopics",
		ValidateName: egcpName("system topic", egcpTopicNameRE),
		Validate:     egcpValidateSystemTopic,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"metricResourceId": egcpMetricID(rec)}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "partnerRegistrations",
		ValidateName: egcpName("partner registration", egcpTopicNameRE),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"partnerRegistrationImmutableId": egcpMetricID(rec)}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "partnerNamespaces",
		ValidateName:       egcpName("partner namespace", egcpTopicNameRE),
		Validate:           egcpValidatePartnerNamespace,
		ReadOnlyProperties: egcpEndpointProperties("/api/events"),
		ServerActions:      keys,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "partnerNamespaces/channels", Proxy: true,
		ValidateName: egcpName("channel", egcpChildNameRE),
		Validate:     egcpValidateChannel,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"readinessState": "NeverActivated"}
		},
		Actions: map[string]azureARMAction{"getfullurl": egcpChannelFullURL},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "partnerTopics",
		ValidateName:    egcpName("partner topic", egcpTopicNameRE),
		StateProperties: map[string]any{"activationState": "NeverActivated"},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"activate":   egcpPartnerTopicActivation("Activated"),
			"deactivate": egcpPartnerTopicActivation("Deactivated"),
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "partnerConfigurations",
		ValidateName: func(name string) string {
			if name != "default" {
				return "The partner configuration of a resource group is named default."
			}
			return ""
		},
		Validate: egcpValidatePartnerConfiguration,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"authorizepartner":   egcpAuthorizePartner,
			"unauthorizepartner": egcpUnauthorizePartner,
		},
	})
	egcpRegisterNamespaces(keys)
	for _, parent := range []struct{ typ, group string }{{"topics", "topic"}, {"domains", "domain"}, {"partnerNamespaces", "partnernamespace"}, {"namespaces", "topic"}} {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: parent.typ + "/privateEndpointConnections", Proxy: true,
			Validate: azurePrivateEndpointConnectionStateValidate,
		})
		group := parent.group
		registerAzureARMView(ns, parent.typ+"/privateLinkResources", func(_ *Server, scope string) []map[string]any {
			return []map[string]any{{
				"id": scope + "/privateLinkResources/" + group, "name": group, "type": ns + "/privateLinkResources",
				"properties": map[string]any{"groupId": group, "displayName": group, "requiredMembers": []any{group}, "requiredZoneNames": []any{"privatelink.eventgrid.azure.net"}},
			}}
		})
	}
	egcpRegisterSubscriptions()
	registerAzureARMView(ns, "verifiedPartners", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, partner := range []string{"Auth0", "MicrosoftGraphAPI", "SAPEventMesh"} {
			out = append(out, map[string]any{
				"id": scope + "/providers/" + ns + "/verifiedPartners/" + partner, "name": partner, "type": ns + "/verifiedPartners",
				"properties": map[string]any{
					"partnerRegistrationImmutableId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(partner)).String(),
					"organizationName":               partner, "partnerDisplayName": partner, "provisioningState": "Succeeded",
					"partnerTopicDetails": map[string]any{"description": partner + " events", "setupUri": "https://learn.microsoft.com/azure/event-grid/partner-events-overview"},
				},
			})
		}
		return out
	})
	registerAzureARMView(ns, "operations", func(*Server, string) []map[string]any {
		var ops []map[string]any
		for _, typ := range []string{"topics", "domains", "domains/topics", "systemTopics", "eventSubscriptions", "namespaces", "namespaces/topics", "partnerNamespaces", "partnerTopics", "partnerRegistrations", "partnerConfigurations"} {
			for _, verb := range []string{"read", "write", "delete"} {
				ops = append(ops, map[string]any{
					"name": ns + "/" + typ + "/" + verb, "isDataAction": false, "origin": "user,system",
					"display": map[string]any{"provider": "Microsoft Event Grid", "resource": typ, "operation": verb + " " + typ, "description": verb + " " + typ},
				})
			}
		}
		return ops
	})
}

// handleAzureEventGridControlPlane20250215Router is kept for the router
// chain; the shared ARM handler serves every Microsoft.EventGrid route.
func (s *Server) handleAzureEventGridControlPlane20250215Router(http.ResponseWriter, *http.Request) bool {
	return false
}

// egcpName returns a ValidateName hook for names matching re.
func egcpName(what string, re *regexp.Regexp) func(string) string {
	return func(name string) string {
		if !re.MatchString(name) {
			return "The " + what + " name must be 3-50 letters, digits or hyphens."
		}
		return ""
	}
}

// egcpKeyActions are listKeys and regenerateKey, for resources with
// key1/key2 shared access keys.
func egcpKeyActions() map[string]func(*Server, *azureARMResource, map[string]any) (int, any) {
	list := func(s *Server, rec *azureARMResource) map[string]any {
		return map[string]any{"key1": s.azureResourceKey(rec, "key1"), "key2": s.azureResourceKey(rec, "key2")}
	}
	return map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
		"listkeys": func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
			return http.StatusOK, list(s, rec)
		},
		"regeneratekey": func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
			name := azureARMString(payload["keyName"])
			if name != "key1" && name != "key2" {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "keyName must be key1 or key2.")
			}
			s.azureRegenerateResourceKey(rec, name, "")
			return http.StatusOK, list(s, rec)
		},
	}
}

func egcpMetricID(rec *azureARMResource) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()
}

// egcpHost is the regional host name Event Grid gives a resource.
func egcpHost(rec *azureARMResource, label string) string {
	return strings.ToLower(rec.Name) + "." + strings.ToLower(strings.ReplaceAll(rec.Location, " ", "")) + "-1." + label + "eventgrid.azure.net"
}

// egcpEndpointProperties returns a ReadOnlyProperties hook giving the
// resource its publishing endpoint.
func egcpEndpointProperties(suffix string) func(*azureARMResource) map[string]any {
	return func(rec *azureARMResource) map[string]any {
		return map[string]any{"endpoint": "https://" + egcpHost(rec, "") + suffix, "metricResourceId": egcpMetricID(rec)}
	}
}

// egcpOneOf reports whether props[key], when present, is one of values.
func egcpOneOf(props map[string]any, key string, values ...string) bool {
	value, present := props[key]
	if !present {
		return true
	}
	for _, allowed := range values {
		if value == allowed {
			return true
		}
	}
	return false
}

func egcpValidateTopic(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch {
	case !egcpOneOf(props, "inputSchema", "EventGridSchema", "CustomEventSchema", "CloudEventSchemaV1_0"):
		return "properties.inputSchema must be EventGridSchema, CustomEventSchema or CloudEventSchemaV1_0."
	case !egcpOneOf(props, "publicNetworkAccess", "Enabled", "Disabled", "SecuredByPerimeter"):
		return "properties.publicNetworkAccess must be Enabled, Disabled or SecuredByPerimeter."
	case !egcpOneOf(props, "minimumTlsVersionAllowed", "1.0", "1.1", "1.2"):
		return "properties.minimumTlsVersionAllowed must be 1.0, 1.1 or 1.2."
	case !egcpOneOf(props, "dataResidencyBoundary", "WithinGeopair", "WithinRegion"):
		return "properties.dataResidencyBoundary must be WithinGeopair or WithinRegion."
	case !egcpOneOf(azureARMObject(payload["sku"]), "name", "Basic", "Premium"):
		return "sku.name must be Basic or Premium."
	}
	if mapping, present := props["inputSchemaMapping"]; present {
		if props["inputSchema"] != "CustomEventSchema" {
			return "properties.inputSchemaMapping requires inputSchema CustomEventSchema."
		}
		if azureARMObject(mapping)["inputSchemaMappingType"] != "Json" {
			return "properties.inputSchemaMapping.inputSchemaMappingType must be Json."
		}
	}
	return egcpValidateIPRules(props)
}

func egcpValidateIPRules(props map[string]any) string {
	rules := azureARMSlice(props["inboundIpRules"])
	if len(rules) > 128 {
		return "properties.inboundIpRules may hold at most 128 rules."
	}
	for i, raw := range rules {
		rule := azureARMObject(raw)
		if azureARMString(rule["ipMask"]) == "" {
			return "properties.inboundIpRules[" + strconv.Itoa(i) + "].ipMask is required."
		}
		if !egcpOneOf(rule, "action", "Allow") {
			return "properties.inboundIpRules[" + strconv.Itoa(i) + "].action must be Allow."
		}
	}
	return ""
}

func egcpValidateSystemTopic(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	source := azureARMString(props["source"])
	if !strings.HasPrefix(strings.ToLower(source), "/subscriptions/") {
		return "properties.source must be the ARM ID of the resource that emits the events."
	}
	topicType := azureARMString(props["topicType"])
	if strings.Count(topicType, ".") < 2 {
		return "properties.topicType must name a topic type such as Microsoft.Storage.StorageAccounts."
	}
	return ""
}

func egcpValidatePartnerNamespace(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.Contains(strings.ToLower(azureARMString(props["partnerRegistrationFullyQualifiedId"])), "/providers/microsoft.eventgrid/partnerregistrations/") {
		return "properties.partnerRegistrationFullyQualifiedId must be a partner registration resource ID."
	}
	if !egcpOneOf(props, "partnerTopicRoutingMode", "SourceEventAttribute", "ChannelNameHeader") {
		return "properties.partnerTopicRoutingMode must be SourceEventAttribute or ChannelNameHeader."
	}
	return egcpValidateIPRules(props)
}

func egcpValidateChannel(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch props["channelType"] {
	case "PartnerTopic":
		info := azureARMObject(props["partnerTopicInfo"])
		if azureARMString(info["azureSubscriptionId"]) == "" || azureARMString(info["resourceGroupName"]) == "" || azureARMString(info["name"]) == "" {
			return "properties.partnerTopicInfo needs azureSubscriptionId, resourceGroupName and name."
		}
	case "PartnerDestination":
		info := azureARMObject(props["partnerDestinationInfo"])
		if info["endpointType"] != "WebHook" {
			return "properties.partnerDestinationInfo.endpointType must be WebHook."
		}
		if !strings.HasPrefix(azureARMString(azureARMObject(info["properties"])["endpointUrl"]), "https://") {
			return "properties.partnerDestinationInfo.properties.endpointUrl must be an https URL."
		}
	default:
		return "properties.channelType must be PartnerTopic or PartnerDestination."
	}
	return ""
}

func egcpChannelFullURL(rec *azureARMResource, _ map[string]any) (int, any) {
	info := azureARMObject(rec.Properties["partnerDestinationInfo"])
	if rec.Properties["channelType"] != "PartnerDestination" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequest", "Only partner destination channels have a full URL.")
	}
	return http.StatusOK, map[string]any{"endpointUrl": azureARMObject(info["properties"])["endpointUrl"]}
}

func egcpPartnerTopicActivation(state string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
		updated := s.azureARMUpdateProperties(rec.ID, map[string]any{"activationState": state})
		if updated == nil {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The partner topic "+rec.ID+" was not found.")
		}
		return http.StatusOK, updated.fixture()
	}
}

func egcpValidatePartnerConfiguration(payload map[string]any) string {
	auth := azureARMObject(azureARMObject(payload["properties"])["partnerAuthorization"])
	if days, present := auth["defaultMaximumExpirationTimeInDays"]; present && !azureWholeNumber(days, 1, 365) {
		return "properties.partnerAuthorization.defaultMaximumExpirationTimeInDays must be an integer from 1 to 365."
	}
	if len(azureARMSlice(auth["authorizedPartnersList"])) > egcpMaxPartners {
		return "properties.partnerAuthorization.authorizedPartnersList may hold at most " + strconv.Itoa(egcpMaxPartners) + " partners."
	}
	return ""
}

const egcpMaxPartners = 100

// egcpPartnerKey identifies a partner by immutable ID, or by name when no ID is given.
func egcpPartnerKey(partner map[string]any) string {
	if id := azureARMString(partner["partnerRegistrationImmutableId"]); id != "" {
		return "id:" + strings.ToLower(id)
	}
	return "name:" + strings.ToLower(azureARMString(partner["partnerName"]))
}

func egcpAuthorizePartner(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if azureARMString(payload["partnerRegistrationImmutableId"]) == "" && azureARMString(payload["partnerName"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "partnerRegistrationImmutableId or partnerName is required.")
	}
	partner := map[string]any{}
	for _, key := range []string{"partnerRegistrationImmutableId", "partnerName", "authorizationExpirationTimeInUtc"} {
		if value := azureARMString(payload[key]); value != "" {
			partner[key] = value
		}
	}
	return egcpEditPartners(s, rec, func(list []any) ([]any, string) {
		kept := egcpWithoutPartner(list, egcpPartnerKey(partner))
		if len(kept) >= egcpMaxPartners {
			return nil, "A partner configuration may authorize at most " + strconv.Itoa(egcpMaxPartners) + " partners."
		}
		return append(kept, partner), ""
	})
}

func egcpUnauthorizePartner(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	key := egcpPartnerKey(payload)
	if key == "name:" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "partnerRegistrationImmutableId or partnerName is required.")
	}
	return egcpEditPartners(s, rec, func(list []any) ([]any, string) { return egcpWithoutPartner(list, key), "" })
}

func egcpWithoutPartner(list []any, key string) []any {
	kept := make([]any, 0, len(list))
	for _, item := range list {
		if egcpPartnerKey(azureARMObject(item)) != key {
			kept = append(kept, item)
		}
	}
	return kept
}

// egcpEditPartners rewrites the configuration's authorized partner list
// under the store lock, so concurrent edits don't lose each other's changes.
func egcpEditPartners(s *Server, rec *azureARMResource, edit func([]any) ([]any, string)) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The partner configuration "+rec.ID+" was not found.")
	}
	auth := azureARMObject(current.Properties["partnerAuthorization"])
	list, msg := edit(azureARMSlice(auth["authorizedPartnersList"]))
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	updated := s.azureARMUpdatePropertiesLocked(rec.ID, map[string]any{"partnerAuthorization": map[string]any{"authorizedPartnersList": list}})
	return http.StatusOK, updated.fixture()
}
