package server

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Event Grid event subscriptions: at any scope (a subscription, resource
// group or resource, as {scope}/providers/Microsoft.EventGrid/
// eventSubscriptions) and as children of topics, domains, domain topics,
// system topics, partner topics and namespace topics; the global, regional
// and per-topic-type lists over them; and the topic-type, event-type and
// extension-topic catalogs.

// egcpSubscriptionTypes are the event subscription resource types.
var egcpSubscriptionTypes = []string{
	"eventSubscriptions", "topics/eventSubscriptions", "domains/eventSubscriptions", "domains/topics/eventSubscriptions",
	"systemTopics/eventSubscriptions", "partnerTopics/eventSubscriptions", "namespaces/topics/eventSubscriptions",
}

// egcpTopicTypes is the topic-type catalog: each type's event types, and
// whether its resources are regional.
var egcpTopicTypes = []struct {
	name     string
	regional bool
	events   []string
}{
	{"Microsoft.Resources.Subscriptions", false, []string{"Microsoft.Resources.ResourceWriteSuccess", "Microsoft.Resources.ResourceDeleteSuccess", "Microsoft.Resources.ResourceActionSuccess"}},
	{"Microsoft.Resources.ResourceGroups", false, []string{"Microsoft.Resources.ResourceWriteSuccess", "Microsoft.Resources.ResourceDeleteSuccess", "Microsoft.Resources.ResourceActionSuccess"}},
	{"Microsoft.Storage.StorageAccounts", true, []string{"Microsoft.Storage.BlobCreated", "Microsoft.Storage.BlobDeleted"}},
	{"Microsoft.EventHub.Namespaces", true, []string{"Microsoft.EventHub.CaptureFileCreated"}},
	{"Microsoft.KeyVault.vaults", true, []string{"Microsoft.KeyVault.SecretNewVersionCreated", "Microsoft.KeyVault.SecretNearExpiry", "Microsoft.KeyVault.SecretExpired"}},
	{"Microsoft.EventGrid.Topics", true, nil},
	{"Microsoft.EventGrid.Domains", true, nil},
}

func egcpRegisterSubscriptions() {
	ns := egcpNamespace
	for _, typePath := range egcpSubscriptionTypes {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: typePath, Proxy: true,
			Extension:    typePath == "eventSubscriptions",
			ValidateName: egcpName("event subscription", egcpChildNameRE),
			Validate:     egcpSubscriptionValidator(typePath == "namespaces/topics/eventSubscriptions"),
			ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
				return map[string]any{"topic": egcpSubscriptionScope(rec.ID)}
			},
			Actions: map[string]azureARMAction{
				"getfullurl":            egcpSubscriptionFullURL,
				"getdeliveryattributes": egcpSubscriptionDeliveryAttributes,
			},
		})
	}
	registerAzureARMView(ns, "topicTypes", func(_ *Server, scope string) []map[string]any {
		return egcpTopicTypeItems(scope + "/providers/" + ns + "/topicTypes")
	})
	registerAzureARMView(ns, "locations/topicTypes", func(_ *Server, scope string) []map[string]any {
		return egcpTopicTypeItems(scope + "/topicTypes")
	})
	registerAzureARMView(ns, "topicTypes/eventTypes", func(_ *Server, scope string) []map[string]any {
		return egcpEventTypeItems(scope, scope[strings.LastIndex(scope, "/")+1:])
	})
	// A resource's event types, at {resourceId}/providers/Microsoft.EventGrid/eventTypes.
	registerAzureARMView(ns, "eventTypes", func(_ *Server, scope string) []map[string]any {
		return egcpEventTypeItems(scope+"/providers/"+ns, egcpScopeTopicType(scope))
	})
	registerAzureARMView(ns, "extensionTopics", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/providers/" + ns + "/extensionTopics/default", "name": "default", "type": ns + "/extensionTopics",
			"properties": map[string]any{"description": "Extension topic of " + scope, "systemTopic": scope},
		}}
	})
	registerAzureARMQueryView(ns, "topicTypes/eventSubscriptions", func(s *Server, scope string, query url.Values) []map[string]any {
		return s.egcpListSubscriptions(scope, "", scope[strings.LastIndex(scope, "/")+1:], query)
	})
	registerAzureARMQueryView(ns, "locations/eventSubscriptions", func(s *Server, scope string, query url.Values) []map[string]any {
		return s.egcpListSubscriptions(scope, scope[strings.LastIndex(scope, "/")+1:], "", query)
	})
	registerAzureARMQueryView(ns, "locations/topicTypes/eventSubscriptions", func(s *Server, scope string, query url.Values) []map[string]any {
		parts := strings.Split(scope, "/")
		// .../locations/{location}/topicTypes/{topicType}
		return s.egcpListSubscriptions(scope, parts[len(parts)-3], parts[len(parts)-1], query)
	})
}

func egcpTopicTypeItems(collection string) []map[string]any {
	out := make([]map[string]any, 0, len(egcpTopicTypes))
	for _, tt := range egcpTopicTypes {
		region := "GlobalResource"
		if tt.regional {
			region = "RegionalResource"
		}
		provider := tt.name[:strings.LastIndex(tt.name, ".")]
		out = append(out, map[string]any{
			"id": collection + "/" + tt.name, "name": tt.name, "type": egcpNamespace + "/topicTypes",
			"properties": map[string]any{
				"provider": provider, "displayName": tt.name, "description": tt.name + " events",
				"resourceRegionType": region, "provisioningState": "Succeeded",
				"supportedScopesForSource": []any{"Resource"},
			},
		})
	}
	return out
}

func egcpEventTypeItems(collection, topicType string) []map[string]any {
	var out []map[string]any
	for _, tt := range egcpTopicTypes {
		if !strings.EqualFold(tt.name, topicType) {
			continue
		}
		for _, event := range tt.events {
			out = append(out, map[string]any{
				"id": collection + "/eventTypes/" + event, "name": event, "type": egcpNamespace + "/topicTypes/eventTypes",
				"properties": map[string]any{"displayName": event, "description": event, "schemaUrl": "https://learn.microsoft.com/azure/event-grid/event-schema", "isInDefaultSet": true},
			})
		}
	}
	return out
}

// egcpSubscriptionScope returns the topic an event subscription belongs to:
// the scope of an extension subscription, or the parent of a child one.
func egcpSubscriptionScope(id string) string {
	lower := strings.ToLower(id)
	i := strings.LastIndex(lower, "/eventsubscriptions/")
	if i < 0 {
		return ""
	}
	scope := id[:i]
	if strings.HasSuffix(strings.ToLower(scope), "/providers/"+strings.ToLower(egcpNamespace)) {
		scope = scope[:len(scope)-len("/providers/"+egcpNamespace)]
	}
	return scope
}

// egcpScopeTopicType returns the topic type of a scope, such as
// Microsoft.Storage.StorageAccounts for a storage account.
func egcpScopeTopicType(scope string) string {
	parts := strings.Split(strings.Trim(scope, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.EqualFold(parts[i], "providers") && i+2 < len(parts) {
			return parts[i+1] + "." + parts[i+2]
		}
	}
	if len(parts) >= 4 && strings.EqualFold(parts[2], "resourceGroups") {
		return "Microsoft.Resources.ResourceGroups"
	}
	return "Microsoft.Resources.Subscriptions"
}

// egcpListSubscriptions lists the event subscriptions under the
// subscription (and resource group) the view scope names. With a location,
// only those on regional resources there; without one, only those on global
// topics (subscriptions and resource groups). topicType narrows the list to
// that topic type. $filter is not supported; $top caps the list.
func (s *Server) egcpListSubscriptions(scope, location, topicType string, query url.Values) []map[string]any {
	parts := strings.Split(strings.Trim(scope, "/"), "/")
	if len(parts) < 2 {
		return nil
	}
	prefix := "/subscriptions/" + strings.ToLower(parts[1]) + "/"
	if len(parts) >= 4 && strings.EqualFold(parts[2], "resourceGroups") {
		prefix += "resourcegroups/" + strings.ToLower(parts[3]) + "/"
	}
	top := 1000
	if raw := query.Get("$top"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n > 0 && n < int64(top) {
			top = int(n)
		}
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	var out []map[string]any
	for _, rec := range s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return egcpIsSubscriptionType(rec.Type) && strings.HasPrefix(azureARMKey(rec.ID), prefix)
	}) {
		topic := egcpSubscriptionScope(rec.ID)
		recType := egcpScopeTopicType(topic)
		if topicType != "" && !strings.EqualFold(recType, topicType) {
			continue
		}
		global := strings.HasPrefix(recType, "Microsoft.Resources.")
		if location == "" && !global {
			continue
		}
		if location != "" {
			owner := s.azureARMResources[azureARMKey(topic)]
			if global || owner == nil || !strings.EqualFold(strings.ReplaceAll(owner.Location, " ", ""), location) {
				continue
			}
		}
		if len(out) == top {
			break
		}
		out = append(out, rec.fixture())
	}
	return out
}

func egcpIsSubscriptionType(fullType string) bool {
	for _, typePath := range egcpSubscriptionTypes {
		if strings.EqualFold(fullType, egcpNamespace+"/"+typePath) {
			return true
		}
	}
	return false
}

// egcpSubscriptionValidator returns the Validate hook for event
// subscriptions; namespace topic subscriptions use delivery configuration
// instead of a destination.
func egcpSubscriptionValidator(namespaceTopic bool) func(map[string]any) string {
	return func(payload map[string]any) string {
		props := azureARMObject(payload["properties"])
		if namespaceTopic {
			if msg := egcpValidateDeliveryConfiguration(azureARMObject(props["deliveryConfiguration"])); msg != "" {
				return msg
			}
		} else {
			destination := azureARMObject(props["destination"])
			if identity := azureARMObject(props["deliveryWithResourceIdentity"]); len(destination) == 0 && len(identity) != 0 {
				destination = azureARMObject(identity["destination"])
			}
			if msg := egcpValidateDestination("properties.destination", destination); msg != "" {
				return msg
			}
		}
		if !egcpOneOf(props, "eventDeliverySchema", "EventGridSchema", "CustomInputSchema", "CloudEventSchemaV1_0") {
			return "properties.eventDeliverySchema must be EventGridSchema, CustomInputSchema or CloudEventSchemaV1_0."
		}
		if raw := azureARMString(props["expirationTimeUtc"]); raw != "" {
			if _, err := time.Parse(time.RFC3339, raw); err != nil {
				return "properties.expirationTimeUtc must be an RFC 3339 timestamp."
			}
		}
		retry := azureARMObject(props["retryPolicy"])
		if n, present := retry["maxDeliveryAttempts"]; present && !azureWholeNumber(n, 1, 30) {
			return "properties.retryPolicy.maxDeliveryAttempts must be an integer from 1 to 30."
		}
		if n, present := retry["eventTimeToLiveInMinutes"]; present && !azureWholeNumber(n, 1, 1440) {
			return "properties.retryPolicy.eventTimeToLiveInMinutes must be an integer from 1 to 1440."
		}
		if namespaceTopic {
			return egcpValidateFilter(azureARMObject(props["filtersConfiguration"]), "properties.filtersConfiguration", "filters")
		}
		return egcpValidateFilter(azureARMObject(props["filter"]), "properties.filter", "advancedFilters")
	}
}

var egcpEndpointTypes = map[string]bool{
	"WebHook": true, "EventHub": true, "StorageQueue": true, "HybridConnection": true, "ServiceBusQueue": true,
	"ServiceBusTopic": true, "AzureFunction": true, "PartnerDestination": true, "MonitorAlert": true, "NamespaceTopic": true,
}

func egcpValidateDestination(at string, destination map[string]any) string {
	endpointType := azureARMString(destination["endpointType"])
	if !egcpEndpointTypes[endpointType] {
		return at + ".endpointType must be one of WebHook, EventHub, StorageQueue, HybridConnection, ServiceBusQueue, ServiceBusTopic, AzureFunction, PartnerDestination, MonitorAlert or NamespaceTopic."
	}
	props := azureARMObject(destination["properties"])
	switch endpointType {
	case "WebHook":
		if !strings.HasPrefix(azureARMString(props["endpointUrl"]), "https://") {
			return at + ".properties.endpointUrl must be an https URL."
		}
		if n, present := props["maxEventsPerBatch"]; present && !azureWholeNumber(n, 1, 5000) {
			return at + ".properties.maxEventsPerBatch must be an integer from 1 to 5000."
		}
		if n, present := props["preferredBatchSizeInKilobytes"]; present && !azureWholeNumber(n, 1, 1024) {
			return at + ".properties.preferredBatchSizeInKilobytes must be an integer from 1 to 1024."
		}
	case "MonitorAlert", "PartnerDestination":
	case "StorageQueue":
		if azureARMString(props["queueName"]) == "" {
			return at + ".properties.queueName is required."
		}
		fallthrough
	default:
		if !strings.HasPrefix(strings.ToLower(azureARMString(props["resourceId"])), "/subscriptions/") {
			return at + ".properties.resourceId must be an ARM resource ID."
		}
	}
	if len(azureARMSlice(props["deliveryAttributeMappings"])) > 10 {
		return at + ".properties.deliveryAttributeMappings may hold at most 10 mappings."
	}
	return ""
}

func egcpValidateDeliveryConfiguration(config map[string]any) string {
	switch config["deliveryMode"] {
	case "Queue":
		queue := azureARMObject(config["queue"])
		if n, present := queue["receiveLockDurationInSeconds"]; present && !azureWholeNumber(n, 60, 300) {
			return "properties.deliveryConfiguration.queue.receiveLockDurationInSeconds must be an integer from 60 to 300."
		}
		if n, present := queue["maxDeliveryCount"]; present && !azureWholeNumber(n, 1, 10) {
			return "properties.deliveryConfiguration.queue.maxDeliveryCount must be an integer from 1 to 10."
		}
	case "Push":
		return egcpValidateDestination("properties.deliveryConfiguration.push.destination", azureARMObject(azureARMObject(config["push"])["destination"]))
	default:
		return "properties.deliveryConfiguration.deliveryMode must be Queue or Push."
	}
	return ""
}

var egcpFilterOperators = map[string]bool{
	"NumberIn": true, "NumberNotIn": true, "NumberLessThan": true, "NumberGreaterThan": true, "NumberLessThanOrEquals": true,
	"NumberGreaterThanOrEquals": true, "BoolEquals": true, "StringIn": true, "StringNotIn": true, "StringBeginsWith": true,
	"StringEndsWith": true, "StringContains": true, "NumberInRange": true, "NumberNotInRange": true, "StringNotBeginsWith": true,
	"StringNotEndsWith": true, "StringNotContains": true, "IsNullOrUndefined": true, "IsNotNull": true,
}

// egcpValidateFilter checks an event subscription filter at path, whose
// advanced filters are under advancedKey.
func egcpValidateFilter(filter map[string]any, path, advancedKey string) string {
	if len(azureARMSlice(filter["includedEventTypes"])) > 100 {
		return path + ".includedEventTypes may hold at most 100 event types."
	}
	advanced := azureARMSlice(filter[advancedKey])
	if len(advanced) > 25 {
		return path + "." + advancedKey + " may hold at most 25 filters."
	}
	for i, raw := range advanced {
		f := azureARMObject(raw)
		at := path + "." + advancedKey + "[" + strconv.Itoa(i) + "]"
		if !egcpFilterOperators[azureARMString(f["operatorType"])] {
			return at + ".operatorType is not a supported operator."
		}
		if azureARMString(f["key"]) == "" {
			return at + ".key is required."
		}
	}
	return ""
}

func egcpSubscriptionDestination(rec *azureARMResource) map[string]any {
	if destination := azureARMObject(rec.Properties["destination"]); len(destination) != 0 {
		return destination
	}
	if identity := azureARMObject(rec.Properties["deliveryWithResourceIdentity"]); len(identity) != 0 {
		return azureARMObject(identity["destination"])
	}
	return azureARMObject(azureARMObject(azureARMObject(rec.Properties["deliveryConfiguration"])["push"])["destination"])
}

func egcpSubscriptionFullURL(rec *azureARMResource, _ map[string]any) (int, any) {
	destination := egcpSubscriptionDestination(rec)
	if destination["endpointType"] != "WebHook" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequest", "Only WebHook destinations have a full endpoint URL.")
	}
	return http.StatusOK, map[string]any{"endpointUrl": azureARMObject(destination["properties"])["endpointUrl"]}
}

func egcpSubscriptionDeliveryAttributes(rec *azureARMResource, _ map[string]any) (int, any) {
	mappings := azureARMSlice(azureARMObject(egcpSubscriptionDestination(rec)["properties"])["deliveryAttributeMappings"])
	if mappings == nil {
		mappings = []any{}
	}
	return http.StatusOK, map[string]any{"value": mappings}
}
