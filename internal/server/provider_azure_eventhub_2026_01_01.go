package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Azure Event Hubs (Microsoft.EventHub 2026-01-01) on the shared ARM handler:
// namespaces with their authorization rules, network rule set, event hubs,
// consumer groups, geo-disaster-recovery aliases, application groups, schema
// groups and private endpoint connections, plus dedicated clusters and their
// quota configuration. Namespace names and SAS rules are shared with the
// Service Bus family (azure_sas_namespaces.go); routes the shared handler
// can't express, such as alias rules and perimeter reconciliation, are in
// azure_eventhub_2026_01_01_routes.go. No events flow: this is the control
// plane only.

const (
	azureEventHub20260101APIVersion = "2026-01-01"
	azureEventHub20260101Prefix     = "/azure/eventhub/2026-01-01"
	evhNamespace                    = "Microsoft.EventHub"
)

func init() {
	ns := evhNamespace
	nsRule := azureSASRuleType(ns, "namespaces/authorizationRules", false)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "namespaces",
		ValidateName:       azureSASValidateNamespaceName,
		Validate:           evhValidateNamespace,
		ReadOnlyProperties: evhNamespaceProperties,
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			rules, _ := lookupAzureARMResourceType(ns, "namespaces/networkRuleSets")
			return append(azureSASRootRule(nsRule)(parent), azureARMChild(parent, rules, "default",
				map[string]any{"defaultAction": "Allow", "publicNetworkAccess": "Enabled", "ipRules": []any{}, "virtualNetworkRules": []any{}}))
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"failover": evhNamespaceFailover,
		},
	})
	registerAzureARMResourceType(nsRule)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/networkRuleSets", Proxy: true,
		ValidateName: evhDefaultName,
		Validate:     evhValidateNetworkRuleSet,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/eventhubs", Proxy: true,
		Validate:           evhValidateEventHub,
		ReadOnlyProperties: evhEventHubProperties,
		ValidatePut: func(_ *Server, existing, rec *azureARMResource) (string, string) {
			if existing != nil && evhPartitionCount(rec.Properties) < evhPartitionCount(existing.Properties) {
				return "", "The partition count of an event hub can't be decreased."
			}
			return "", ""
		},
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			groups, _ := lookupAzureARMResourceType(ns, "namespaces/eventhubs/consumergroups")
			return []*azureARMResource{azureARMChild(parent, groups, "$Default", map[string]any{})}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/eventhubs/consumergroups", Proxy: true,
		ValidateName: evhEntityName("consumer group"),
		Validate: func(payload map[string]any) string {
			if len(azureARMString(azureARMObject(payload["properties"])["userMetadata"])) > 1024 {
				return "properties.userMetadata must be at most 1024 characters."
			}
			return ""
		},
		ReadOnlyProperties: evhTimestamps,
		ValidateDelete: func(_ *Server, existing *azureARMResource) string {
			if existing.Name == "$Default" {
				return "The $Default consumer group can't be deleted."
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureSASRuleType(ns, "namespaces/eventhubs/authorizationRules", true))
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/disasterRecoveryConfigs", Proxy: true,
		ValidateName: azureSASValidateNamespaceName,
		Validate:     evhValidateAlias,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"role": "Primary", "pendingReplicationOperationsCount": 0}
		},
		CollectionActions: map[string]func(*Server, string, map[string]any) (int, any){
			"checknameavailability": evhAliasNameAvailability,
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			// Breaking or failing over ends the pairing; the alias stays.
			"breakpairing": evhAliasUnpair("Primary"),
			"failover":     evhAliasUnpair("PrimaryNotReplicating"),
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/applicationGroups", Proxy: true,
		ValidateName: evhEntityName("application group"),
		Validate:     evhValidateApplicationGroup,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/schemagroups", Proxy: true,
		ValidateName:       evhEntityName("schema group"),
		Validate:           evhValidateSchemaGroup,
		ReadOnlyProperties: evhSchemaGroupProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "namespaces/privateEndpointConnections", Proxy: true,
		Validate: azurePrivateEndpointConnectionStateValidate,
	})
	registerAzureARMView(ns, "namespaces/privateLinkResources", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/privateLinkResources/namespace", "name": "namespace", "type": ns + "/namespaces/privateLinkResources",
			"properties": map[string]any{"groupId": "namespace", "requiredMembers": []any{"namespace"}, "requiredZoneNames": []any{"privatelink.servicebus.windows.net"}},
		}}
	})
	registerAzureARMView(ns, "namespaces/networkSecurityPerimeterConfigurations", func(*Server, string) []map[string]any {
		return nil // No network security perimeters exist in the emulator.
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "clusters",
		Validate:           evhValidateCluster,
		ReadOnlyProperties: evhClusterProperties,
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			quota, _ := lookupAzureARMResourceType(ns, "clusters/quotaConfiguration")
			return []*azureARMResource{azureARMChild(parent, quota, "default", map[string]any{"settings": map[string]any{}})}
		},
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			if len(evhClusterNamespacesLocked(s, existing.ID)) > 0 {
				return "The cluster still hosts namespaces; delete them first."
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/quotaConfiguration", Proxy: true,
		NoPut: true, NoDelete: true,
		Validate: evhValidateQuota,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if msg := evhValidateQuota(payload); msg != "" {
				return nil, nil, msg
			}
			return payload, nil, ""
		},
	})
	registerAzureARMView(ns, "clusters/namespaces", func(s *Server, scope string) []map[string]any {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		var out []map[string]any
		for _, rec := range evhClusterNamespacesLocked(s, scope) {
			out = append(out, map[string]any{"id": rec.ID})
		}
		return out
	})
	registerAzureARMView(ns, "availableClusterRegions", func(*Server, string) []map[string]any {
		var out []map[string]any
		for _, region := range []string{"eastus", "eastus2", "westus2", "westeurope", "northeurope", "southeastasia"} {
			out = append(out, map[string]any{"location": region})
		}
		return out
	})
	registerAzureARMView(ns, "operations", func(*Server, string) []map[string]any {
		var ops []map[string]any
		for _, typ := range []string{"namespaces", "namespaces/eventhubs", "namespaces/eventhubs/consumergroups", "namespaces/authorizationRules", "namespaces/disasterRecoveryConfigs", "clusters"} {
			for _, verb := range []string{"read", "write", "delete"} {
				ops = append(ops, map[string]any{
					"name": ns + "/" + typ + "/" + verb, "isDataAction": false,
					"display": map[string]any{"provider": "Microsoft Event Hubs", "resource": typ, "operation": verb + " " + typ, "description": verb + " " + typ},
				})
			}
		}
		return ops
	})
	registerAzureARMProviderAction(ns, "CheckNameAvailability", azureServiceBusCheckNameAvailability)
}

// handleAzureEventHub20260101Router serves the Event Hubs routes the shared
// ARM handler can't express (azure_eventhub_2026_01_01_routes.go).
func (s *Server) handleAzureEventHub20260101Router(w http.ResponseWriter, r *http.Request) bool {
	return s.evhServeExtraRoutes(w, r)
}

func evhDefaultName(name string) string {
	if name != "default" {
		return "The only network rule set is named default."
	}
	return ""
}

// evhEntityName returns a ValidateName hook for entity names: 1-256
// characters of letters, digits, periods, hyphens, underscores and $.
func evhEntityName(what string) func(string) string {
	return func(name string) string {
		if len(name) == 0 || len(name) > 256 || strings.Trim(name, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-_$") != "" {
			return "The " + what + " name must be 1-256 letters, digits, periods, hyphens, underscores or $."
		}
		return ""
	}
}

func evhValidateNamespace(payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	switch name := azureARMString(sku["name"]); name {
	case "Basic", "Standard", "Premium":
		if tier, ok := sku["tier"]; ok && tier != name {
			return "sku.tier must match sku.name."
		}
	default:
		return "sku.name must be one of Basic, Standard, Premium."
	}
	if capacity, ok := sku["capacity"]; ok && !azureWholeNumber(capacity, 1, 40) {
		return "sku.capacity must be an integer from 1 to 40."
	}
	props := azureARMObject(payload["properties"])
	if units, ok := props["maximumThroughputUnits"]; ok && !azureWholeNumber(units, 0, 40) {
		return "properties.maximumThroughputUnits must be an integer from 0 to 40."
	}
	if tls, ok := props["minimumTlsVersion"]; ok && tls != "1.0" && tls != "1.1" && tls != "1.2" {
		return "properties.minimumTlsVersion must be 1.0, 1.1 or 1.2."
	}
	if access, ok := props["publicNetworkAccess"]; ok && access != "Enabled" && access != "Disabled" && access != "SecuredByPerimeter" {
		return "properties.publicNetworkAccess must be Enabled, Disabled or SecuredByPerimeter."
	}
	if cluster := azureARMString(props["clusterArmId"]); cluster != "" && !strings.Contains(strings.ToLower(cluster), "/providers/microsoft.eventhub/clusters/") {
		return "properties.clusterArmId must be an Event Hubs cluster resource ID."
	}
	return ""
}

func evhNamespaceProperties(rec *azureARMResource) map[string]any {
	props := azureSASNamespaceProperties(rec)
	props["metricId"] = rec.SubscriptionID + ":" + strings.ToLower(rec.Name)
	props["createdAt"] = rec.CreatedAt.Format(time.RFC3339)
	props["updatedAt"] = rec.UpdatedAt.Format(time.RFC3339)
	// Kafka is available on every tier but Basic.
	props["kafkaEnabled"] = azureARMString(rec.SKU["name"]) != "Basic"
	return props
}

func evhNamespaceFailover(_ *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	props := azureARMObject(payload["properties"])
	if azureARMString(props["primaryLocation"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.primaryLocation is required.")
	}
	if azureARMString(rec.SKU["name"]) != "Premium" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Namespace failover requires a Premium namespace with geo-replication.")
	}
	return http.StatusAccepted, nil
}

func evhValidateNetworkRuleSet(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if action, ok := props["defaultAction"]; ok && action != "Allow" && action != "Deny" {
		return "properties.defaultAction must be Allow or Deny."
	}
	for i, rule := range azureARMSlice(props["ipRules"]) {
		rule := azureARMObject(rule)
		if azureARMString(rule["ipMask"]) == "" {
			return "properties.ipRules[" + strconv.Itoa(i) + "].ipMask is required."
		}
		if action, ok := rule["action"]; ok && action != "Allow" {
			return "properties.ipRules[" + strconv.Itoa(i) + "].action must be Allow."
		}
	}
	for i, rule := range azureARMSlice(props["virtualNetworkRules"]) {
		if !azureNetworkIDOf(azureARMObject(rule)["subnet"], "virtualnetworks/subnets") {
			return "properties.virtualNetworkRules[" + strconv.Itoa(i) + "].subnet.id must be a subnet resource ID."
		}
	}
	return ""
}

// evhPartitionCount reads partitionCount as the client sent it (float64)
// or as stored (int), defaulting to 4.
func evhPartitionCount(props map[string]any) int {
	switch n := props["partitionCount"].(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 4
}

func evhValidateEventHub(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if n, ok := props["partitionCount"]; ok && !azureWholeNumber(n, 1, 1024) {
		return "properties.partitionCount must be an integer from 1 to 1024."
	}
	if days, ok := props["messageRetentionInDays"]; ok && !azureWholeNumber(days, 1, 90) {
		return "properties.messageRetentionInDays must be an integer from 1 to 90."
	}
	if status, ok := props["status"]; ok {
		switch status {
		case "Active", "Disabled", "SendDisabled", "ReceiveDisabled":
		default:
			return "properties.status must be Active, Disabled, SendDisabled or ReceiveDisabled."
		}
	}
	retention := azureARMObject(props["retentionDescription"])
	if policy, ok := retention["cleanupPolicy"]; ok && policy != "Delete" && policy != "Compact" && policy != "DeleteOrCompact" {
		return "properties.retentionDescription.cleanupPolicy must be Delete, Compact or DeleteOrCompact."
	}
	if hours, ok := retention["retentionTimeInHours"]; ok && !azureWholeNumber(hours, 1, 2160) {
		return "properties.retentionDescription.retentionTimeInHours must be an integer from 1 to 2160."
	}
	return ""
}

func evhTimestamps(rec *azureARMResource) map[string]any {
	return map[string]any{"createdAt": rec.CreatedAt.Format(time.RFC3339), "updatedAt": rec.UpdatedAt.Format(time.RFC3339)}
}

func evhEventHubProperties(rec *azureARMResource) map[string]any {
	n := evhPartitionCount(rec.Properties)
	ids := make([]any, n)
	for i := range ids {
		ids[i] = strconv.Itoa(i)
	}
	props := evhTimestamps(rec)
	props["partitionCount"] = n
	props["partitionIds"] = ids
	if _, ok := rec.Properties["status"]; !ok {
		props["status"] = "Active"
	}
	return props
}

func evhValidateAlias(payload map[string]any) string {
	partner := azureARMString(azureARMObject(payload["properties"])["partnerNamespace"])
	if partner == "" {
		return "properties.partnerNamespace is required."
	}
	if !strings.Contains(strings.ToLower(partner), "/providers/microsoft.eventhub/namespaces/") {
		return "properties.partnerNamespace must be an Event Hubs namespace resource ID."
	}
	return ""
}

// evhAliasUnpair returns an alias action that ends the pairing, leaving the
// namespace it runs on in role.
func evhAliasUnpair(role string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
		s.azureARMUpdateProperties(rec.ID, map[string]any{"partnerNamespace": "", "role": role})
		return http.StatusOK, nil
	}
}

func evhValidateApplicationGroup(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	id := azureARMString(props["clientAppGroupIdentifier"])
	if !strings.HasPrefix(id, "SASKeyName=") && !strings.HasPrefix(id, "AADAppID=") {
		return "properties.clientAppGroupIdentifier must start with SASKeyName= or AADAppID=."
	}
	for i, raw := range azureARMSlice(props["policies"]) {
		policy := azureARMObject(raw)
		at := "properties.policies[" + strconv.Itoa(i) + "]"
		switch {
		case policy["type"] != "ThrottlingPolicy":
			return at + ".type must be ThrottlingPolicy."
		case azureARMString(policy["name"]) == "":
			return at + ".name is required."
		case !azureWholeNumber(policy["rateLimitThreshold"], 1, 1e12):
			return at + ".rateLimitThreshold must be a positive integer."
		}
		switch policy["metricId"] {
		case "IncomingBytes", "OutgoingBytes", "IncomingMessages", "OutgoingMessages":
		default:
			return at + ".metricId must be IncomingBytes, OutgoingBytes, IncomingMessages or OutgoingMessages."
		}
	}
	return ""
}

func evhValidateSchemaGroup(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if v, ok := props["schemaCompatibility"]; ok && v != "None" && v != "Backward" && v != "Forward" {
		return "properties.schemaCompatibility must be None, Backward or Forward."
	}
	if v, ok := props["schemaType"]; ok && v != "Unknown" && v != "Avro" && v != "Json" && v != "ProtoBuf" {
		return "properties.schemaType must be Unknown, Avro, Json or ProtoBuf."
	}
	return ""
}

func evhSchemaGroupProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"createdAtUtc": rec.CreatedAt.Format(time.RFC3339), "updatedAtUtc": rec.UpdatedAt.Format(time.RFC3339),
		"eTag": strings.Trim(rec.etag(), `"`),
	}
}

func evhValidateCluster(payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	if sku["name"] != "Dedicated" {
		return "sku.name must be Dedicated."
	}
	if capacity, ok := sku["capacity"]; ok && !azureWholeNumber(capacity, 1, 20) {
		return "sku.capacity must be an integer from 1 to 20."
	}
	return ""
}

func evhClusterProperties(rec *azureARMResource) map[string]any {
	props := evhTimestamps(rec)
	props["metricId"] = rec.SubscriptionID + ":" + strings.ToLower(rec.Name)
	props["status"] = "Active"
	props["supportsScaling"] = true
	return props
}

// evhClusterNamespacesLocked returns the namespaces hosted on the cluster.
// Caller holds the store lock.
func evhClusterNamespacesLocked(s *Server, clusterID string) []*azureARMResource {
	return s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, evhNamespace+"/namespaces") && strings.EqualFold(azureARMString(rec.Properties["clusterArmId"]), clusterID)
	})
}

func evhValidateQuota(payload map[string]any) string {
	for key, value := range azureARMObject(azureARMObject(payload["properties"])["settings"]) {
		if _, err := strconv.ParseInt(azureARMString(value), 10, 64); err != nil {
			return "properties.settings." + key + " must be an integer string."
		}
	}
	return ""
}

// evhAliasNameAvailability checks a geo-DR alias name, a DNS label under
// servicebus.windows.net like namespace names.
func evhAliasNameAvailability(s *Server, _ string, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureSASValidateNamespaceName(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "InvalidName", "message": msg}
	}
	taken := s.azureSASNamespaceNameTaken(name) || len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, evhNamespace+"/namespaces/disasterRecoveryConfigs") && strings.EqualFold(rec.Name, name)
	})) > 0
	if taken {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "NameInUse", "message": "The specified name is not available."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true, "reason": "None", "message": ""}
}
