package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type azureApplicationGatewayResource struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	SKU            map[string]any
	Zones          []string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Operational    string
	ETag           string
}

func (s *Server) handleAzureApplicationGatewayRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsApplicationGatewayPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureApplicationGatewayAllowedMethod(r.Method) {
		return false
	}

	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Network") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureApplicationGatewayOperationsFixture()})
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureApplicationGatewayRouteContext(segments)
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		switch {
		case len(tail) == 1 && strings.EqualFold(tail[0], "applicationGateways") && r.Method == http.MethodGet:
			s.handleAzureApplicationGatewayListAll(w, subscriptionID)
			return true
		case len(tail) == 1 && strings.EqualFold(tail[0], "applicationGatewayAvailableRequestHeaders") && r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, azureApplicationGatewayAvailableRequestHeadersFixture())
			return true
		case len(tail) == 1 && strings.EqualFold(tail[0], "applicationGatewayAvailableResponseHeaders") && r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, azureApplicationGatewayAvailableResponseHeadersFixture())
			return true
		case len(tail) == 1 && strings.EqualFold(tail[0], "applicationGatewayAvailableServerVariables") && r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, azureApplicationGatewayAvailableServerVariablesFixture())
			return true
		case len(tail) == 1 && strings.EqualFold(tail[0], "applicationGatewayAvailableWafRuleSets") && r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, azureApplicationGatewayAvailableWAFRuleSetsFixture())
			return true
		case len(tail) == 2 &&
			strings.EqualFold(tail[0], "applicationGatewayAvailableSslOptions") &&
			strings.EqualFold(tail[1], "default") &&
			r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, azureApplicationGatewayAvailableSSLOptionsFixture(subscriptionID))
			return true
		case len(tail) == 3 &&
			strings.EqualFold(tail[0], "applicationGatewayAvailableSslOptions") &&
			strings.EqualFold(tail[1], "default") &&
			strings.EqualFold(tail[2], "predefinedPolicies") &&
			r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, map[string]any{"value": azureApplicationGatewaySSLPredefinedPoliciesFixture(subscriptionID)})
			return true
		case len(tail) == 4 &&
			strings.EqualFold(tail[0], "applicationGatewayAvailableSslOptions") &&
			strings.EqualFold(tail[1], "default") &&
			strings.EqualFold(tail[2], "predefinedPolicies") &&
			r.Method == http.MethodGet:
			policy := azureApplicationGatewaySSLPredefinedPolicyFixture(subscriptionID, tail[3])
			if policy == nil {
				respondJSON(w, http.StatusNotFound, map[string]any{
					"error":    "NotFound",
					"message":  "SSL predefined policy not found",
					"provider": providerAzure,
					"path":     path,
				})
				return true
			}
			respondJSON(w, http.StatusOK, policy)
			return true
		default:
			respondAzureImplemented(w, path)
			return true
		}
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "applicationGateways") && r.Method == http.MethodGet:
		s.handleAzureApplicationGatewayListByResourceGroup(w, subscriptionID, resourceGroup)
		return true
	case len(tail) >= 2 && strings.EqualFold(tail[0], "applicationGateways"):
		name := strings.TrimSpace(tail[1])
		if name == "" {
			respondAzureInvalidRequest(w, path, "application gateway name is required")
			return true
		}
		if len(tail) == 2 {
			switch r.Method {
			case http.MethodGet:
				s.handleAzureApplicationGatewayGet(w, path, subscriptionID, resourceGroup, name, false)
			case http.MethodHead:
				s.handleAzureApplicationGatewayGet(w, path, subscriptionID, resourceGroup, name, true)
			case http.MethodPut:
				s.handleAzureApplicationGatewayCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
			case http.MethodPatch:
				s.handleAzureApplicationGatewayPatch(w, r, path, subscriptionID, resourceGroup, name)
			case http.MethodDelete:
				s.handleAzureApplicationGatewayDelete(w, path, subscriptionID, resourceGroup, name)
			default:
				respondAzureImplemented(w, path)
			}
			return true
		}
		switch {
		case len(tail) == 3 && strings.EqualFold(tail[2], "backendhealth") && (r.Method == http.MethodPost || r.Method == http.MethodGet):
			s.handleAzureApplicationGatewayBackendHealth(w, path, subscriptionID, resourceGroup, name)
			return true
		case len(tail) == 3 && strings.EqualFold(tail[2], "getBackendHealthOnDemand") && r.Method == http.MethodPost:
			s.handleAzureApplicationGatewayBackendHealthOnDemand(w, r, path, subscriptionID, resourceGroup, name)
			return true
		case len(tail) == 3 && strings.EqualFold(tail[2], "start") && r.Method == http.MethodPost:
			s.handleAzureApplicationGatewaySetOperationalState(w, path, subscriptionID, resourceGroup, name, "Running")
			return true
		case len(tail) == 3 && strings.EqualFold(tail[2], "stop") && r.Method == http.MethodPost:
			s.handleAzureApplicationGatewaySetOperationalState(w, path, subscriptionID, resourceGroup, name, "Stopped")
			return true
		default:
			respondAzureImplemented(w, path)
			return true
		}
	default:
		respondAzureImplemented(w, path)
		return true
	}
}

func azureIsApplicationGatewayPath(segments []string) bool {
	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Network") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureApplicationGatewayRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	if strings.EqualFold(tail[0], "applicationGateways") {
		return true
	}
	switch strings.ToLower(tail[0]) {
	case "applicationgatewayavailablerequestheaders",
		"applicationgatewayavailableresponseheaders",
		"applicationgatewayavailableservervariables",
		"applicationgatewayavailablessloptions",
		"applicationgatewayavailablewafrulesets":
		return true
	default:
		return false
	}
}

func azureApplicationGatewayRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.Network") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureApplicationGatewayAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureApplicationGatewayCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureApplicationGatewayReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureApplicationGatewayString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureApplicationGatewayKey(subscriptionID, resourceGroup, name)
	resource := s.azureApplicationGateways[key]
	if resource == nil {
		resource = &azureApplicationGatewayResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			CreatedAt:      now,
		}
	}
	resource.Location = location
	resource.Tags = azureApplicationGatewayStringMap(payload["tags"])
	resource.SKU = azureApplicationGatewayObject(payload["sku"])
	resource.Zones = azureApplicationGatewayStringSlice(payload["zones"])
	resource.Properties = azureApplicationGatewayObject(payload["properties"])
	resource.UpdatedAt = now
	resource.Operational = azureApplicationGatewayOperationalState(resource.Properties, resource.Operational)
	resource.ETag = azureApplicationGatewayETag(key, body, now)
	s.azureApplicationGateways[key] = resource

	respondJSON(w, http.StatusOK, azureApplicationGatewayResourceFixture(resource))
}

func (s *Server) handleAzureApplicationGatewayPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureApplicationGatewayReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureApplicationGatewayKey(subscriptionID, resourceGroup, name)
	resource := s.azureApplicationGateways[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{
			"error":    "NotFound",
			"message":  "application gateway not found",
			"provider": providerAzure,
			"path":     path,
		})
		return
	}

	if location := azureApplicationGatewayString(payload["location"]); location != "" {
		resource.Location = location
	}
	if tags, ok := payload["tags"]; ok {
		resource.Tags = azureApplicationGatewayStringMap(tags)
	}
	if sku, ok := payload["sku"]; ok {
		resource.SKU = azureApplicationGatewayObject(sku)
	}
	if zones, ok := payload["zones"]; ok {
		resource.Zones = azureApplicationGatewayStringSlice(zones)
	}
	if props, ok := payload["properties"]; ok {
		resource.Properties = azureApplicationGatewayMergeObject(resource.Properties, azureApplicationGatewayObject(props))
	}
	resource.UpdatedAt = time.Now().UTC()
	resource.Operational = azureApplicationGatewayOperationalState(resource.Properties, resource.Operational)
	resource.ETag = azureApplicationGatewayETag(key, body, resource.UpdatedAt)

	respondJSON(w, http.StatusOK, azureApplicationGatewayResourceFixture(resource))
}

func (s *Server) handleAzureApplicationGatewayGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureApplicationGateways[azureApplicationGatewayKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{
			"error":    "NotFound",
			"message":  "application gateway not found",
			"provider": providerAzure,
			"path":     path,
		})
		return
	}
	if headOnly {
		w.Header().Set("ETag", resource.ETag)
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureApplicationGatewayResourceFixture(resource))
}

func (s *Server) handleAzureApplicationGatewayDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	key := azureApplicationGatewayKey(subscriptionID, resourceGroup, name)
	if s.azureApplicationGateways[key] == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, map[string]any{
			"error":    "NotFound",
			"message":  "application gateway not found",
			"provider": providerAzure,
			"path":     path,
		})
		return
	}
	delete(s.azureApplicationGateways, key)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{"status": "Deleted"})
}

func (s *Server) handleAzureApplicationGatewayListAll(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]*azureApplicationGatewayResource, 0, len(s.azureApplicationGateways))
	for _, resource := range s.azureApplicationGateways {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return azureApplicationGatewayKey(items[i].SubscriptionID, items[i].ResourceGroup, items[i].Name) <
			azureApplicationGatewayKey(items[j].SubscriptionID, items[j].ResourceGroup, items[j].Name)
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": azureApplicationGatewayResourceListFixture(items)})
}

func (s *Server) handleAzureApplicationGatewayListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]*azureApplicationGatewayResource, 0, len(s.azureApplicationGateways))
	for _, resource := range s.azureApplicationGateways {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) && strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": azureApplicationGatewayResourceListFixture(items)})
}

func (s *Server) handleAzureApplicationGatewayBackendHealth(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureApplicationGateways[azureApplicationGatewayKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{
			"error":    "NotFound",
			"message":  "application gateway not found",
			"provider": providerAzure,
			"path":     path,
		})
		return
	}
	respondJSON(w, http.StatusOK, azureApplicationGatewayBackendHealthFixture(resource))
}

func (s *Server) handleAzureApplicationGatewayBackendHealthOnDemand(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureApplicationGatewayReadRequestBody(w, r, path)
	if !ok {
		return
	}
	_ = body

	s.providerStorageMu.Lock()
	resource := s.azureApplicationGateways[azureApplicationGatewayKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{
			"error":    "NotFound",
			"message":  "application gateway not found",
			"provider": providerAzure,
			"path":     path,
		})
		return
	}
	respondJSON(w, http.StatusOK, azureApplicationGatewayBackendHealthOnDemandFixture(resource, payload))
}

func (s *Server) handleAzureApplicationGatewaySetOperationalState(w http.ResponseWriter, path, subscriptionID, resourceGroup, name, state string) {
	s.providerStorageMu.Lock()
	resource := s.azureApplicationGateways[azureApplicationGatewayKey(subscriptionID, resourceGroup, name)]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, map[string]any{
			"error":    "NotFound",
			"message":  "application gateway not found",
			"provider": providerAzure,
			"path":     path,
		})
		return
	}
	resource.Operational = state
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureApplicationGatewayETag(azureApplicationGatewayKey(subscriptionID, resourceGroup, name), []byte(state), resource.UpdatedAt)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{"status": state})
}

func azureApplicationGatewayReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
		return nil, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "invalid JSON payload")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureApplicationGatewayResourceFixture(resource *azureApplicationGatewayResource) map[string]any {
	properties := azureApplicationGatewayMergeObject(resource.Properties, map[string]any{
		"provisioningState": "Succeeded",
		"operationalState":  resource.Operational,
		"resourceGuid":      azureApplicationGatewayResourceGUID(resource.Name),
	})
	if _, ok := properties["frontendPorts"]; !ok {
		properties["frontendPorts"] = []any{
			map[string]any{
				"name": "port-80",
				"properties": map[string]any{
					"port": 80,
				},
			},
		}
	}
	if _, ok := properties["backendAddressPools"]; !ok {
		properties["backendAddressPools"] = []any{
			map[string]any{
				"name": "pool-default",
				"properties": map[string]any{
					"backendAddresses": []any{
						map[string]any{"ipAddress": "10.0.0.4"},
					},
				},
			},
		}
	}
	return map[string]any{
		"id":         azureApplicationGatewayResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.Network/applicationGateways",
		"location":   resource.Location,
		"tags":       azureApplicationGatewayAnyStringMap(resource.Tags),
		"etag":       resource.ETag,
		"zones":      resource.Zones,
		"properties": properties,
		"sku":        resource.SKU,
	}
}

func azureApplicationGatewayResourceListFixture(resources []*azureApplicationGatewayResource) []map[string]any {
	out := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		out = append(out, azureApplicationGatewayResourceFixture(resource))
	}
	return out
}

func azureApplicationGatewayOperationsFixture() []map[string]any {
	return []map[string]any{
		{
			"name": "Microsoft.Network/applicationGateways/read",
			"display": map[string]any{
				"provider":    "Microsoft Network",
				"resource":    "ApplicationGateways",
				"operation":   "Get Application Gateway",
				"description": "Get an application gateway resource.",
			},
		},
		{
			"name": "Microsoft.Network/applicationGateways/write",
			"display": map[string]any{
				"provider":    "Microsoft Network",
				"resource":    "ApplicationGateways",
				"operation":   "Create Or Update Application Gateway",
				"description": "Create or update an application gateway resource.",
			},
		},
		{
			"name": "Microsoft.Network/applicationGateways/delete",
			"display": map[string]any{
				"provider":    "Microsoft Network",
				"resource":    "ApplicationGateways",
				"operation":   "Delete Application Gateway",
				"description": "Delete an application gateway resource.",
			},
		},
	}
}

func azureApplicationGatewayAvailableRequestHeadersFixture() map[string]any {
	headers := []string{"User-Agent", "X-Forwarded-For", "X-Original-Host"}
	return map[string]any{
		"availableRequestHeaders": headers,
		"value":                   headers,
	}
}

func azureApplicationGatewayAvailableResponseHeadersFixture() map[string]any {
	headers := []string{"Server", "Strict-Transport-Security", "X-Frame-Options"}
	return map[string]any{
		"availableResponseHeaders": headers,
		"value":                    headers,
	}
}

func azureApplicationGatewayAvailableServerVariablesFixture() map[string]any {
	vars := []string{"client_ip", "request_uri", "request_method"}
	return map[string]any{
		"availableServerVariables": vars,
		"value":                    vars,
	}
}

func azureApplicationGatewayAvailableSSLOptionsFixture(subscriptionID string) map[string]any {
	baseID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default", subscriptionID)
	return map[string]any{
		"id":   baseID,
		"name": "default",
		"type": "Microsoft.Network/applicationGatewayAvailableSslOptions",
		"properties": map[string]any{
			"availableCipherSuites": []string{
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			},
			"availableProtocols": []string{"TLSv1_2", "TLSv1_3"},
			"defaultPolicy":      "AppGwSslPolicy20220101",
		},
	}
}

func azureApplicationGatewaySSLPredefinedPoliciesFixture(subscriptionID string) []map[string]any {
	return []map[string]any{
		azureApplicationGatewaySSLPredefinedPolicyFixture(subscriptionID, "AppGwSslPolicy20220101"),
		azureApplicationGatewaySSLPredefinedPolicyFixture(subscriptionID, "AppGwSslPolicy20220102S"),
	}
}

func azureApplicationGatewaySSLPredefinedPolicyFixture(subscriptionID, policyName string) map[string]any {
	name := strings.TrimSpace(policyName)
	if name == "" {
		return nil
	}
	switch name {
	case "AppGwSslPolicy20220101", "AppGwSslPolicy20220102S":
	default:
		return nil
	}
	return map[string]any{
		"id":   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default/predefinedPolicies/%s", subscriptionID, name),
		"name": name,
		"type": "Microsoft.Network/applicationGatewayAvailableSslOptions/predefinedPolicies",
		"properties": map[string]any{
			"minProtocolVersion": "TLSv1_2",
			"cipherSuites": []string{
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			},
		},
	}
}

func azureApplicationGatewayAvailableWAFRuleSetsFixture() map[string]any {
	return map[string]any{
		"value": []map[string]any{
			{
				"ruleSetType":    "OWASP",
				"ruleSetVersion": "3.2",
				"ruleGroups": []map[string]any{
					{
						"ruleGroupName": "REQUEST-920-PROTOCOL-ENFORCEMENT",
						"description":   "Protocol enforcement rules.",
					},
				},
			},
		},
	}
}

func azureApplicationGatewayBackendHealthFixture(resource *azureApplicationGatewayResource) map[string]any {
	baseID := azureApplicationGatewayResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Name)
	return map[string]any{
		"backendAddressPools": []map[string]any{
			{
				"backendAddressPool": map[string]any{
					"id": baseID + "/backendAddressPools/pool-default",
				},
				"backendHttpSettingsCollection": []map[string]any{
					{
						"backendHttpSettings": map[string]any{
							"id": baseID + "/backendHttpSettingsCollection/setting-default",
						},
						"servers": []map[string]any{
							{
								"address": "10.0.0.4",
								"health":  "Healthy",
							},
						},
					},
				},
			},
		},
	}
}

func azureApplicationGatewayBackendHealthOnDemandFixture(resource *azureApplicationGatewayResource, payload map[string]any) map[string]any {
	baseID := azureApplicationGatewayResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Name)
	probe := azureApplicationGatewayObject(payload["properties"])
	return map[string]any{
		"backendAddressPool": map[string]any{
			"id": baseID + "/backendAddressPools/pool-default",
		},
		"backendSettingsCollection": map[string]any{
			"id": baseID + "/backendHttpSettingsCollection/setting-default",
		},
		"health": "Healthy",
		"servers": []map[string]any{
			{
				"address": "10.0.0.4",
				"health":  "Healthy",
			},
		},
		"probe": probe,
	}
}

func azureApplicationGatewayResourceID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationGateways/%s",
		subscriptionID,
		resourceGroup,
		name,
	)
}

func azureApplicationGatewayKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "|" + resourceGroup + "|" + name)
}

func azureApplicationGatewayETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%x", key, now.UnixNano(), body)))
	return fmt.Sprintf("W/\"%s\"", hex.EncodeToString(sum[:8]))
}

func azureApplicationGatewayResourceGUID(name string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(name))))
	encoded := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func azureApplicationGatewayOperationalState(properties map[string]any, current string) string {
	if props := azureApplicationGatewayObject(properties); props != nil {
		if value := azureApplicationGatewayString(props["operationalState"]); value != "" {
			return value
		}
	}
	if strings.TrimSpace(current) != "" {
		return current
	}
	return "Running"
}

func azureApplicationGatewayString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func azureApplicationGatewayObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	out := make(map[string]any, len(raw))
	for key, item := range raw {
		out[key] = item
	}
	return out
}

func azureApplicationGatewayStringMap(value any) map[string]string {
	raw, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, item := range raw {
		text := azureApplicationGatewayString(item)
		if strings.TrimSpace(key) == "" || text == "" {
			continue
		}
		out[key] = text
	}
	return out
}

func azureApplicationGatewayAnyStringMap(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func azureApplicationGatewayStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text := azureApplicationGatewayString(item)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func azureApplicationGatewayMergeObject(base, updates map[string]any) map[string]any {
	out := azureApplicationGatewayObject(base)
	for key, value := range updates {
		out[key] = value
	}
	return out
}
