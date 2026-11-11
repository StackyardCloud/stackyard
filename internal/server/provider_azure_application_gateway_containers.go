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

type azureApplicationGatewayContainersTrafficController struct {
	SubscriptionID      string
	ResourceGroup       string
	Name                string
	Location            string
	Tags                map[string]string
	WafSecurityPolicyID string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type azureApplicationGatewayContainersFrontend struct {
	SubscriptionID        string
	ResourceGroup         string
	TrafficControllerName string
	Name                  string
	Location              string
	Tags                  map[string]string
	FQDN                  string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type azureApplicationGatewayContainersAssociation struct {
	SubscriptionID        string
	ResourceGroup         string
	TrafficControllerName string
	Name                  string
	Location              string
	Tags                  map[string]string
	AssociationType       string
	SubnetID              string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type azureApplicationGatewayContainersSecurityPolicy struct {
	SubscriptionID        string
	ResourceGroup         string
	TrafficControllerName string
	Name                  string
	Location              string
	Tags                  map[string]string
	WafPolicyID           string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (s *Server) handleAzureApplicationGatewayContainersRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsApplicationGatewayContainersPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ServiceNetworking") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{
				"value": azureApplicationGatewayContainersOperationsFixture(),
			})
			return true
		}
		return false
	}

	subscriptionID, resourceGroup, tail, ok := azureApplicationGatewayContainersRouteContext(segments)
	if !ok {
		return false
	}
	if !azureApplicationGatewayContainersAllowedMethod(resourceGroup, tail, r.Method) {
		return false
	}

	if resourceGroup == "" {
		if len(tail) == 1 && strings.EqualFold(tail[0], "trafficControllers") && r.Method == http.MethodGet {
			s.handleAzureApplicationGatewayContainersTrafficControllerListBySubscription(w, subscriptionID)
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "trafficControllers") && r.Method == http.MethodGet:
		s.handleAzureApplicationGatewayContainersTrafficControllerListByResourceGroup(w, subscriptionID, resourceGroup)
		return true
	case len(tail) >= 2 && strings.EqualFold(tail[0], "trafficControllers"):
		trafficControllerName := strings.TrimSpace(tail[1])
		if trafficControllerName == "" {
			respondAzureInvalidRequest(w, path, "traffic controller name is required")
			return true
		}
		if len(tail) == 2 {
			switch r.Method {
			case http.MethodGet:
				s.handleAzureApplicationGatewayContainersTrafficControllerGet(w, path, subscriptionID, resourceGroup, trafficControllerName)
			case http.MethodPut:
				s.handleAzureApplicationGatewayContainersTrafficControllerCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, trafficControllerName)
			case http.MethodPatch:
				s.handleAzureApplicationGatewayContainersTrafficControllerPatch(w, r, path, subscriptionID, resourceGroup, trafficControllerName)
			case http.MethodDelete:
				s.handleAzureApplicationGatewayContainersTrafficControllerDelete(w, subscriptionID, resourceGroup, trafficControllerName)
			default:
				respondAzureImplemented(w, path)
			}
			return true
		}
		switch strings.ToLower(tail[2]) {
		case "frontends":
			s.handleAzureApplicationGatewayContainersFrontendRoute(w, r, path, subscriptionID, resourceGroup, trafficControllerName, tail[3:])
			return true
		case "associations":
			s.handleAzureApplicationGatewayContainersAssociationRoute(w, r, path, subscriptionID, resourceGroup, trafficControllerName, tail[3:])
			return true
		case "securitypolicies":
			s.handleAzureApplicationGatewayContainersSecurityPolicyRoute(w, r, path, subscriptionID, resourceGroup, trafficControllerName, tail[3:])
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

func (s *Server) handleAzureApplicationGatewayContainersFrontendRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureApplicationGatewayContainersFrontendList(w, path, subscriptionID, resourceGroup, trafficControllerName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "frontend name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureApplicationGatewayContainersFrontendGet(w, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodPut:
		s.handleAzureApplicationGatewayContainersFrontendCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodPatch:
		s.handleAzureApplicationGatewayContainersFrontendPatch(w, r, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodDelete:
		s.handleAzureApplicationGatewayContainersFrontendDelete(w, subscriptionID, resourceGroup, trafficControllerName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureApplicationGatewayContainersAssociationRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureApplicationGatewayContainersAssociationList(w, path, subscriptionID, resourceGroup, trafficControllerName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "association name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureApplicationGatewayContainersAssociationGet(w, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodPut:
		s.handleAzureApplicationGatewayContainersAssociationCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodPatch:
		s.handleAzureApplicationGatewayContainersAssociationPatch(w, r, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodDelete:
		s.handleAzureApplicationGatewayContainersAssociationDelete(w, subscriptionID, resourceGroup, trafficControllerName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureApplicationGatewayContainersSecurityPolicyRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureApplicationGatewayContainersSecurityPolicyList(w, path, subscriptionID, resourceGroup, trafficControllerName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "security policy name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureApplicationGatewayContainersSecurityPolicyGet(w, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodPut:
		s.handleAzureApplicationGatewayContainersSecurityPolicyCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodPatch:
		s.handleAzureApplicationGatewayContainersSecurityPolicyPatch(w, r, path, subscriptionID, resourceGroup, trafficControllerName, name)
	case http.MethodDelete:
		s.handleAzureApplicationGatewayContainersSecurityPolicyDelete(w, subscriptionID, resourceGroup, trafficControllerName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureApplicationGatewayContainersTrafficControllerCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureApplicationGatewayContainersString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, name)
	current := s.azureApplicationGatewayContainersTrafficControllers[key]
	wafSecurityPolicyID := ""
	if current != nil {
		wafSecurityPolicyID = current.WafSecurityPolicyID
	}
	if next := azureApplicationGatewayContainersTrafficControllerWAFIDFromPayload(payload); next != "" {
		wafSecurityPolicyID = next
	}

	resource := &azureApplicationGatewayContainersTrafficController{
		SubscriptionID:      subscriptionID,
		ResourceGroup:       resourceGroup,
		Name:                name,
		Location:            location,
		Tags:                azureApplicationGatewayContainersStringMap(payload["tags"]),
		WafSecurityPolicyID: wafSecurityPolicyID,
		UpdatedAt:           now,
	}
	if current != nil {
		resource.CreatedAt = current.CreatedAt
	} else {
		resource.CreatedAt = now
	}
	s.azureApplicationGatewayContainersTrafficControllers[key] = resource

	etag := azureApplicationGatewayContainersETag(key, body, now)
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersTrafficControllerFixture(resource, etag, s.azureApplicationGatewayContainersTrafficControllerFrontendRefs(subscriptionID, resourceGroup, name), s.azureApplicationGatewayContainersTrafficControllerAssociationRefs(subscriptionID, resourceGroup, name), s.azureApplicationGatewayContainersTrafficControllerSecurityPolicyRefs(subscriptionID, resourceGroup, name)))
}

func (s *Server) handleAzureApplicationGatewayContainersTrafficControllerPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, name)
	current := s.azureApplicationGatewayContainersTrafficControllers[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}

	if tags, ok := payload["tags"]; ok {
		current.Tags = azureApplicationGatewayContainersStringMap(tags)
	}
	if next := azureApplicationGatewayContainersTrafficControllerWAFIDFromPayload(payload); next != "" {
		current.WafSecurityPolicyID = next
	}
	current.UpdatedAt = time.Now().UTC()
	etag := azureApplicationGatewayContainersETag(key, body, current.UpdatedAt)
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersTrafficControllerFixture(current, etag, s.azureApplicationGatewayContainersTrafficControllerFrontendRefs(subscriptionID, resourceGroup, name), s.azureApplicationGatewayContainersTrafficControllerAssociationRefs(subscriptionID, resourceGroup, name), s.azureApplicationGatewayContainersTrafficControllerSecurityPolicyRefs(subscriptionID, resourceGroup, name)))
}

func (s *Server) handleAzureApplicationGatewayContainersTrafficControllerGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, name)
	current := s.azureApplicationGatewayContainersTrafficControllers[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}
	etag := azureApplicationGatewayContainersETag(key, []byte(name), current.UpdatedAt)
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersTrafficControllerFixture(current, etag, s.azureApplicationGatewayContainersTrafficControllerFrontendRefs(subscriptionID, resourceGroup, name), s.azureApplicationGatewayContainersTrafficControllerAssociationRefs(subscriptionID, resourceGroup, name), s.azureApplicationGatewayContainersTrafficControllerSecurityPolicyRefs(subscriptionID, resourceGroup, name)))
}

func (s *Server) handleAzureApplicationGatewayContainersTrafficControllerListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, item := range s.azureApplicationGatewayContainersTrafficControllers {
		if item.SubscriptionID != subscriptionID || !strings.EqualFold(item.ResourceGroup, resourceGroup) {
			continue
		}
		key := azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, item.Name)
		items = append(items, azureApplicationGatewayContainersTrafficControllerFixture(item, azureApplicationGatewayContainersETag(key, []byte(item.Name), item.UpdatedAt), s.azureApplicationGatewayContainersTrafficControllerFrontendRefs(subscriptionID, resourceGroup, item.Name), s.azureApplicationGatewayContainersTrafficControllerAssociationRefs(subscriptionID, resourceGroup, item.Name), s.azureApplicationGatewayContainersTrafficControllerSecurityPolicyRefs(subscriptionID, resourceGroup, item.Name)))
	}
	sort.Slice(items, func(i, j int) bool {
		return azureApplicationGatewayContainersString(items[i]["name"]) < azureApplicationGatewayContainersString(items[j]["name"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureApplicationGatewayContainersTrafficControllerListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, item := range s.azureApplicationGatewayContainersTrafficControllers {
		if item.SubscriptionID != subscriptionID {
			continue
		}
		key := azureApplicationGatewayContainersTrafficControllerKey(item.SubscriptionID, item.ResourceGroup, item.Name)
		items = append(items, azureApplicationGatewayContainersTrafficControllerFixture(item, azureApplicationGatewayContainersETag(key, []byte(item.Name), item.UpdatedAt), s.azureApplicationGatewayContainersTrafficControllerFrontendRefs(item.SubscriptionID, item.ResourceGroup, item.Name), s.azureApplicationGatewayContainersTrafficControllerAssociationRefs(item.SubscriptionID, item.ResourceGroup, item.Name), s.azureApplicationGatewayContainersTrafficControllerSecurityPolicyRefs(item.SubscriptionID, item.ResourceGroup, item.Name)))
	}
	sort.Slice(items, func(i, j int) bool {
		return azureApplicationGatewayContainersString(items[i]["id"]) < azureApplicationGatewayContainersString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureApplicationGatewayContainersTrafficControllerDelete(w http.ResponseWriter, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, name)
	if _, ok := s.azureApplicationGatewayContainersTrafficControllers[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureApplicationGatewayContainersTrafficControllers, key)
	for childKey, child := range s.azureApplicationGatewayContainersFrontends {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) && strings.EqualFold(child.TrafficControllerName, name) {
			delete(s.azureApplicationGatewayContainersFrontends, childKey)
		}
	}
	for childKey, child := range s.azureApplicationGatewayContainersAssociations {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) && strings.EqualFold(child.TrafficControllerName, name) {
			delete(s.azureApplicationGatewayContainersAssociations, childKey)
		}
	}
	for childKey, child := range s.azureApplicationGatewayContainersSecurityPolicies {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) && strings.EqualFold(child.TrafficControllerName, name) {
			delete(s.azureApplicationGatewayContainersSecurityPolicies, childKey)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureApplicationGatewayContainersFrontendCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureApplicationGatewayContainersString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureApplicationGatewayContainersTrafficControllerExists(subscriptionID, resourceGroup, trafficControllerName) {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}

	now := time.Now().UTC()
	key := azureApplicationGatewayContainersFrontendKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersFrontends[key]
	resource := &azureApplicationGatewayContainersFrontend{
		SubscriptionID:        subscriptionID,
		ResourceGroup:         resourceGroup,
		TrafficControllerName: trafficControllerName,
		Name:                  name,
		Location:              location,
		Tags:                  azureApplicationGatewayContainersStringMap(payload["tags"]),
		FQDN:                  azureApplicationGatewayContainersFrontendFQDN(name, trafficControllerName, location),
		UpdatedAt:             now,
	}
	if current != nil {
		resource.CreatedAt = current.CreatedAt
	} else {
		resource.CreatedAt = now
	}
	s.azureApplicationGatewayContainersFrontends[key] = resource
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersFrontendFixture(resource, azureApplicationGatewayContainersETag(key, body, now)))
}

func (s *Server) handleAzureApplicationGatewayContainersFrontendPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureApplicationGatewayContainersFrontendKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersFrontends[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "frontend not found")
		return
	}
	if tags, ok := payload["tags"]; ok {
		current.Tags = azureApplicationGatewayContainersStringMap(tags)
	}
	current.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersFrontendFixture(current, azureApplicationGatewayContainersETag(key, body, current.UpdatedAt)))
}

func (s *Server) handleAzureApplicationGatewayContainersFrontendGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureApplicationGatewayContainersFrontendKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersFrontends[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "frontend not found")
		return
	}
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersFrontendFixture(current, azureApplicationGatewayContainersETag(key, []byte(name), current.UpdatedAt)))
}

func (s *Server) handleAzureApplicationGatewayContainersFrontendList(w http.ResponseWriter, path, subscriptionID, resourceGroup, trafficControllerName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureApplicationGatewayContainersTrafficControllerExists(subscriptionID, resourceGroup, trafficControllerName) {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}
	items := make([]map[string]any, 0)
	for _, item := range s.azureApplicationGatewayContainersFrontends {
		if item.SubscriptionID == subscriptionID && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.TrafficControllerName, trafficControllerName) {
			key := azureApplicationGatewayContainersFrontendKey(subscriptionID, resourceGroup, trafficControllerName, item.Name)
			items = append(items, azureApplicationGatewayContainersFrontendFixture(item, azureApplicationGatewayContainersETag(key, []byte(item.Name), item.UpdatedAt)))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureApplicationGatewayContainersString(items[i]["name"]) < azureApplicationGatewayContainersString(items[j]["name"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureApplicationGatewayContainersFrontendDelete(w http.ResponseWriter, subscriptionID, resourceGroup, trafficControllerName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	delete(s.azureApplicationGatewayContainersFrontends, azureApplicationGatewayContainersFrontendKey(subscriptionID, resourceGroup, trafficControllerName, name))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureApplicationGatewayContainersAssociationCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureApplicationGatewayContainersString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	associationType, subnetID := azureApplicationGatewayContainersAssociationFieldsFromPayload(payload)
	if associationType == "" {
		respondAzureInvalidRequest(w, path, "properties.associationType is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureApplicationGatewayContainersTrafficControllerExists(subscriptionID, resourceGroup, trafficControllerName) {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}

	now := time.Now().UTC()
	key := azureApplicationGatewayContainersAssociationKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersAssociations[key]
	resource := &azureApplicationGatewayContainersAssociation{
		SubscriptionID:        subscriptionID,
		ResourceGroup:         resourceGroup,
		TrafficControllerName: trafficControllerName,
		Name:                  name,
		Location:              location,
		Tags:                  azureApplicationGatewayContainersStringMap(payload["tags"]),
		AssociationType:       associationType,
		SubnetID:              subnetID,
		UpdatedAt:             now,
	}
	if current != nil {
		resource.CreatedAt = current.CreatedAt
	} else {
		resource.CreatedAt = now
	}
	s.azureApplicationGatewayContainersAssociations[key] = resource
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersAssociationFixture(resource, azureApplicationGatewayContainersETag(key, body, now)))
}

func (s *Server) handleAzureApplicationGatewayContainersAssociationPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureApplicationGatewayContainersAssociationKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersAssociations[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "association not found")
		return
	}
	if tags, ok := payload["tags"]; ok {
		current.Tags = azureApplicationGatewayContainersStringMap(tags)
	}
	if associationType, subnetID := azureApplicationGatewayContainersAssociationFieldsFromPayload(payload); associationType != "" || subnetID != "" {
		if associationType != "" {
			current.AssociationType = associationType
		}
		if subnetID != "" {
			current.SubnetID = subnetID
		}
	}
	current.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersAssociationFixture(current, azureApplicationGatewayContainersETag(key, body, current.UpdatedAt)))
}

func (s *Server) handleAzureApplicationGatewayContainersAssociationGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureApplicationGatewayContainersAssociationKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersAssociations[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "association not found")
		return
	}
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersAssociationFixture(current, azureApplicationGatewayContainersETag(key, []byte(name), current.UpdatedAt)))
}

func (s *Server) handleAzureApplicationGatewayContainersAssociationList(w http.ResponseWriter, path, subscriptionID, resourceGroup, trafficControllerName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureApplicationGatewayContainersTrafficControllerExists(subscriptionID, resourceGroup, trafficControllerName) {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}
	items := make([]map[string]any, 0)
	for _, item := range s.azureApplicationGatewayContainersAssociations {
		if item.SubscriptionID == subscriptionID && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.TrafficControllerName, trafficControllerName) {
			key := azureApplicationGatewayContainersAssociationKey(subscriptionID, resourceGroup, trafficControllerName, item.Name)
			items = append(items, azureApplicationGatewayContainersAssociationFixture(item, azureApplicationGatewayContainersETag(key, []byte(item.Name), item.UpdatedAt)))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureApplicationGatewayContainersString(items[i]["name"]) < azureApplicationGatewayContainersString(items[j]["name"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureApplicationGatewayContainersAssociationDelete(w http.ResponseWriter, subscriptionID, resourceGroup, trafficControllerName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	delete(s.azureApplicationGatewayContainersAssociations, azureApplicationGatewayContainersAssociationKey(subscriptionID, resourceGroup, trafficControllerName, name))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureApplicationGatewayContainersSecurityPolicyCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureApplicationGatewayContainersString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	wafPolicyID := azureApplicationGatewayContainersSecurityPolicyWAFIDFromPayload(payload)
	if wafPolicyID == "" {
		respondAzureInvalidRequest(w, path, "properties.wafPolicy.id is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	parentKey := azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, trafficControllerName)
	parent := s.azureApplicationGatewayContainersTrafficControllers[parentKey]
	if parent == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}

	now := time.Now().UTC()
	key := azureApplicationGatewayContainersSecurityPolicyKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersSecurityPolicies[key]
	resource := &azureApplicationGatewayContainersSecurityPolicy{
		SubscriptionID:        subscriptionID,
		ResourceGroup:         resourceGroup,
		TrafficControllerName: trafficControllerName,
		Name:                  name,
		Location:              location,
		Tags:                  azureApplicationGatewayContainersStringMap(payload["tags"]),
		WafPolicyID:           wafPolicyID,
		UpdatedAt:             now,
	}
	if current != nil {
		resource.CreatedAt = current.CreatedAt
	} else {
		resource.CreatedAt = now
	}
	s.azureApplicationGatewayContainersSecurityPolicies[key] = resource
	parent.WafSecurityPolicyID = azureApplicationGatewayContainersSecurityPolicyID(subscriptionID, resourceGroup, trafficControllerName, name)
	parent.UpdatedAt = now
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersSecurityPolicyFixture(resource, azureApplicationGatewayContainersETag(key, body, now)))
}

func (s *Server) handleAzureApplicationGatewayContainersSecurityPolicyPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	body, payload, ok := azureApplicationGatewayContainersReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureApplicationGatewayContainersSecurityPolicyKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersSecurityPolicies[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "security policy not found")
		return
	}
	if tags, ok := payload["tags"]; ok {
		current.Tags = azureApplicationGatewayContainersStringMap(tags)
	}
	if next := azureApplicationGatewayContainersSecurityPolicyWAFIDFromPayload(payload); next != "" {
		current.WafPolicyID = next
		if parent := s.azureApplicationGatewayContainersTrafficControllers[azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, trafficControllerName)]; parent != nil {
			parent.WafSecurityPolicyID = key
			parent.UpdatedAt = time.Now().UTC()
		}
	}
	current.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersSecurityPolicyFixture(current, azureApplicationGatewayContainersETag(key, body, current.UpdatedAt)))
}

func (s *Server) handleAzureApplicationGatewayContainersSecurityPolicyGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, trafficControllerName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureApplicationGatewayContainersSecurityPolicyKey(subscriptionID, resourceGroup, trafficControllerName, name)
	current := s.azureApplicationGatewayContainersSecurityPolicies[key]
	if current == nil {
		respondAzureApplicationGatewayContainersNotFound(w, path, "security policy not found")
		return
	}
	respondJSON(w, http.StatusOK, azureApplicationGatewayContainersSecurityPolicyFixture(current, azureApplicationGatewayContainersETag(key, []byte(name), current.UpdatedAt)))
}

func (s *Server) handleAzureApplicationGatewayContainersSecurityPolicyList(w http.ResponseWriter, path, subscriptionID, resourceGroup, trafficControllerName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureApplicationGatewayContainersTrafficControllerExists(subscriptionID, resourceGroup, trafficControllerName) {
		respondAzureApplicationGatewayContainersNotFound(w, path, "traffic controller not found")
		return
	}
	items := make([]map[string]any, 0)
	for _, item := range s.azureApplicationGatewayContainersSecurityPolicies {
		if item.SubscriptionID == subscriptionID && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.TrafficControllerName, trafficControllerName) {
			key := azureApplicationGatewayContainersSecurityPolicyKey(subscriptionID, resourceGroup, trafficControllerName, item.Name)
			items = append(items, azureApplicationGatewayContainersSecurityPolicyFixture(item, azureApplicationGatewayContainersETag(key, []byte(item.Name), item.UpdatedAt)))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureApplicationGatewayContainersString(items[i]["name"]) < azureApplicationGatewayContainersString(items[j]["name"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureApplicationGatewayContainersSecurityPolicyDelete(w http.ResponseWriter, subscriptionID, resourceGroup, trafficControllerName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureApplicationGatewayContainersSecurityPolicyKey(subscriptionID, resourceGroup, trafficControllerName, name)
	delete(s.azureApplicationGatewayContainersSecurityPolicies, key)
	parentKey := azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, trafficControllerName)
	if parent := s.azureApplicationGatewayContainersTrafficControllers[parentKey]; parent != nil && strings.EqualFold(parent.WafSecurityPolicyID, key) {
		parent.WafSecurityPolicyID = ""
		parent.UpdatedAt = time.Now().UTC()
	}
	w.WriteHeader(http.StatusNoContent)
}

func azureIsApplicationGatewayContainersPath(segments []string) bool {
	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ServiceNetworking") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureApplicationGatewayContainersRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	return strings.EqualFold(tail[0], "trafficControllers")
}

func azureApplicationGatewayContainersRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.ServiceNetworking") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureApplicationGatewayContainersAllowedMethod(resourceGroup string, tail []string, method string) bool {
	if len(tail) == 0 {
		return false
	}
	if resourceGroup == "" {
		return len(tail) == 1 && strings.EqualFold(tail[0], "trafficControllers") && method == http.MethodGet
	}
	switch len(tail) {
	case 1:
		return strings.EqualFold(tail[0], "trafficControllers") && method == http.MethodGet
	case 2:
		return strings.EqualFold(tail[0], "trafficControllers") && (method == http.MethodGet || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete)
	case 3:
		switch strings.ToLower(tail[2]) {
		case "frontends", "associations", "securitypolicies":
			return method == http.MethodGet
		default:
			return false
		}
	case 4:
		switch strings.ToLower(tail[2]) {
		case "frontends", "associations", "securitypolicies":
			return method == http.MethodGet || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
		default:
			return false
		}
	default:
		return false
	}
}

func azureApplicationGatewayContainersReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	return body, payload, true
}

func azureApplicationGatewayContainersString(value any) string {
	str, _ := value.(string)
	return strings.TrimSpace(str)
}

func azureApplicationGatewayContainersStringMap(value any) map[string]string {
	out := map[string]string{}
	raw, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for k, v := range raw {
		if str := azureApplicationGatewayContainersString(v); str != "" {
			out[k] = str
		}
	}
	return out
}

func azureApplicationGatewayContainersAssociationFieldsFromPayload(payload map[string]any) (associationType, subnetID string) {
	properties, _ := payload["properties"].(map[string]any)
	associationType = azureApplicationGatewayContainersString(properties["associationType"])
	if subnet, ok := properties["subnet"].(map[string]any); ok {
		subnetID = azureApplicationGatewayContainersString(subnet["id"])
	}
	return associationType, subnetID
}

func azureApplicationGatewayContainersSecurityPolicyWAFIDFromPayload(payload map[string]any) string {
	properties, _ := payload["properties"].(map[string]any)
	if waf, ok := properties["wafPolicy"].(map[string]any); ok {
		return azureApplicationGatewayContainersString(waf["id"])
	}
	return ""
}

func azureApplicationGatewayContainersTrafficControllerWAFIDFromPayload(payload map[string]any) string {
	properties, _ := payload["properties"].(map[string]any)
	if cfg, ok := properties["securityPolicyConfigurations"].(map[string]any); ok {
		if waf, ok := cfg["wafSecurityPolicy"].(map[string]any); ok {
			return azureApplicationGatewayContainersString(waf["id"])
		}
	}
	return ""
}

func azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "::" + strings.ToLower(resourceGroup) + "::" + strings.ToLower(name)
}

func azureApplicationGatewayContainersFrontendKey(subscriptionID, resourceGroup, trafficControllerName, name string) string {
	return azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, trafficControllerName) + "::frontend::" + strings.ToLower(name)
}

func azureApplicationGatewayContainersAssociationKey(subscriptionID, resourceGroup, trafficControllerName, name string) string {
	return azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, trafficControllerName) + "::association::" + strings.ToLower(name)
}

func azureApplicationGatewayContainersSecurityPolicyKey(subscriptionID, resourceGroup, trafficControllerName, name string) string {
	return azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, trafficControllerName) + "::securitypolicy::" + strings.ToLower(name)
}

func azureApplicationGatewayContainersTrafficControllerID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ServiceNetworking/trafficControllers/%s", subscriptionID, resourceGroup, name)
}

func azureApplicationGatewayContainersFrontendID(subscriptionID, resourceGroup, trafficControllerName, name string) string {
	return azureApplicationGatewayContainersTrafficControllerID(subscriptionID, resourceGroup, trafficControllerName) + "/frontends/" + name
}

func azureApplicationGatewayContainersAssociationID(subscriptionID, resourceGroup, trafficControllerName, name string) string {
	return azureApplicationGatewayContainersTrafficControllerID(subscriptionID, resourceGroup, trafficControllerName) + "/associations/" + name
}

func azureApplicationGatewayContainersSecurityPolicyID(subscriptionID, resourceGroup, trafficControllerName, name string) string {
	return azureApplicationGatewayContainersTrafficControllerID(subscriptionID, resourceGroup, trafficControllerName) + "/securityPolicies/" + name
}

func azureApplicationGatewayContainersETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+now.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureApplicationGatewayContainersFrontendFQDN(frontendName, trafficControllerName, location string) string {
	location = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), " ", ""))
	if location == "" {
		location = "eastus"
	}
	return fmt.Sprintf("%s.%s.%s.agfc.stackyard.local", strings.ToLower(frontendName), strings.ToLower(trafficControllerName), location)
}

func (s *Server) azureApplicationGatewayContainersTrafficControllerExists(subscriptionID, resourceGroup, name string) bool {
	_, ok := s.azureApplicationGatewayContainersTrafficControllers[azureApplicationGatewayContainersTrafficControllerKey(subscriptionID, resourceGroup, name)]
	return ok
}

func (s *Server) azureApplicationGatewayContainersTrafficControllerFrontendRefs(subscriptionID, resourceGroup, trafficControllerName string) []map[string]any {
	refs := make([]string, 0)
	for _, item := range s.azureApplicationGatewayContainersFrontends {
		if item.SubscriptionID == subscriptionID && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.TrafficControllerName, trafficControllerName) {
			refs = append(refs, azureApplicationGatewayContainersFrontendID(subscriptionID, resourceGroup, trafficControllerName, item.Name))
		}
	}
	sort.Strings(refs)
	out := make([]map[string]any, 0, len(refs))
	for _, id := range refs {
		out = append(out, map[string]any{"id": id})
	}
	return out
}

func (s *Server) azureApplicationGatewayContainersTrafficControllerAssociationRefs(subscriptionID, resourceGroup, trafficControllerName string) []map[string]any {
	refs := make([]string, 0)
	for _, item := range s.azureApplicationGatewayContainersAssociations {
		if item.SubscriptionID == subscriptionID && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.TrafficControllerName, trafficControllerName) {
			refs = append(refs, azureApplicationGatewayContainersAssociationID(subscriptionID, resourceGroup, trafficControllerName, item.Name))
		}
	}
	sort.Strings(refs)
	out := make([]map[string]any, 0, len(refs))
	for _, id := range refs {
		out = append(out, map[string]any{"id": id})
	}
	return out
}

func (s *Server) azureApplicationGatewayContainersTrafficControllerSecurityPolicyRefs(subscriptionID, resourceGroup, trafficControllerName string) []map[string]any {
	refs := make([]string, 0)
	for _, item := range s.azureApplicationGatewayContainersSecurityPolicies {
		if item.SubscriptionID == subscriptionID && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.TrafficControllerName, trafficControllerName) {
			refs = append(refs, azureApplicationGatewayContainersSecurityPolicyID(subscriptionID, resourceGroup, trafficControllerName, item.Name))
		}
	}
	sort.Strings(refs)
	out := make([]map[string]any, 0, len(refs))
	for _, id := range refs {
		out = append(out, map[string]any{"id": id})
	}
	return out
}

func azureApplicationGatewayContainersTrafficControllerFixture(resource *azureApplicationGatewayContainersTrafficController, etag string, frontendRefs, associationRefs, securityPolicyRefs []map[string]any) map[string]any {
	wafSecurityPolicyID := resource.WafSecurityPolicyID
	if wafSecurityPolicyID == "" && len(securityPolicyRefs) > 0 {
		wafSecurityPolicyID = azureApplicationGatewayContainersString(securityPolicyRefs[0]["id"])
	}
	properties := map[string]any{
		"configurationEndpoints": []string{
			fmt.Sprintf("https://%s.control.stackyard.local/config", strings.ToLower(resource.Name)),
		},
		"frontends":         frontendRefs,
		"associations":      associationRefs,
		"securityPolicies":  securityPolicyRefs,
		"provisioningState": "Succeeded",
	}
	if wafSecurityPolicyID != "" {
		properties["securityPolicyConfigurations"] = map[string]any{
			"wafSecurityPolicy": map[string]any{
				"id": wafSecurityPolicyID,
			},
		}
	}
	return map[string]any{
		"id":         azureApplicationGatewayContainersTrafficControllerID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.ServiceNetworking/trafficControllers",
		"etag":       etag,
		"location":   resource.Location,
		"tags":       azureApplicationGatewayContainersAnyMap(resource.Tags),
		"properties": properties,
	}
}

func azureApplicationGatewayContainersFrontendFixture(resource *azureApplicationGatewayContainersFrontend, etag string) map[string]any {
	return map[string]any{
		"id":       azureApplicationGatewayContainersFrontendID(resource.SubscriptionID, resource.ResourceGroup, resource.TrafficControllerName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.ServiceNetworking/trafficControllers/frontends",
		"etag":     etag,
		"location": resource.Location,
		"tags":     azureApplicationGatewayContainersAnyMap(resource.Tags),
		"properties": map[string]any{
			"fqdn":              resource.FQDN,
			"provisioningState": "Succeeded",
		},
	}
}

func azureApplicationGatewayContainersAssociationFixture(resource *azureApplicationGatewayContainersAssociation, etag string) map[string]any {
	properties := map[string]any{
		"associationType":   resource.AssociationType,
		"provisioningState": "Succeeded",
	}
	if resource.SubnetID != "" {
		properties["subnet"] = map[string]any{"id": resource.SubnetID}
	}
	return map[string]any{
		"id":         azureApplicationGatewayContainersAssociationID(resource.SubscriptionID, resource.ResourceGroup, resource.TrafficControllerName, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.ServiceNetworking/trafficControllers/associations",
		"etag":       etag,
		"location":   resource.Location,
		"tags":       azureApplicationGatewayContainersAnyMap(resource.Tags),
		"properties": properties,
	}
}

func azureApplicationGatewayContainersSecurityPolicyFixture(resource *azureApplicationGatewayContainersSecurityPolicy, etag string) map[string]any {
	return map[string]any{
		"id":       azureApplicationGatewayContainersSecurityPolicyID(resource.SubscriptionID, resource.ResourceGroup, resource.TrafficControllerName, resource.Name),
		"name":     resource.Name,
		"type":     "Microsoft.ServiceNetworking/trafficControllers/securityPolicies",
		"etag":     etag,
		"location": resource.Location,
		"tags":     azureApplicationGatewayContainersAnyMap(resource.Tags),
		"properties": map[string]any{
			"wafPolicy": map[string]any{
				"id": resource.WafPolicyID,
			},
			"policyType":        "waf",
			"provisioningState": "Succeeded",
		},
	}
}

func azureApplicationGatewayContainersAnyMap(tags map[string]string) map[string]any {
	out := map[string]any{}
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func respondAzureApplicationGatewayContainersNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerAzure,
		"path":     path,
	})
}

func azureApplicationGatewayContainersOperationsFixture() []map[string]any {
	operations := []struct {
		name        string
		resource    string
		operation   string
		description string
	}{
		{"Microsoft.ServiceNetworking/trafficControllers/read", "Traffic Controllers", "Get Traffic Controller", "Get a traffic controller resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/write", "Traffic Controllers", "Create Or Update Traffic Controller", "Create or update a traffic controller resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/delete", "Traffic Controllers", "Delete Traffic Controller", "Delete a traffic controller resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/frontends/read", "Frontends", "Get Frontend", "Get a frontend child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/frontends/write", "Frontends", "Create Or Update Frontend", "Create or update a frontend child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/frontends/delete", "Frontends", "Delete Frontend", "Delete a frontend child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/associations/read", "Associations", "Get Association", "Get an association child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/associations/write", "Associations", "Create Or Update Association", "Create or update an association child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/associations/delete", "Associations", "Delete Association", "Delete an association child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/securityPolicies/read", "Security Policies", "Get Security Policy", "Get a security policy child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/securityPolicies/write", "Security Policies", "Create Or Update Security Policy", "Create or update a security policy child resource"},
		{"Microsoft.ServiceNetworking/trafficControllers/securityPolicies/delete", "Security Policies", "Delete Security Policy", "Delete a security policy child resource"},
	}

	value := make([]map[string]any, 0, len(operations))
	for _, operation := range operations {
		value = append(value, map[string]any{
			"name":         operation.name,
			"isDataAction": false,
			"origin":       "user,system",
			"display": map[string]any{
				"provider":    "Microsoft Service Networking",
				"resource":    operation.resource,
				"operation":   operation.operation,
				"description": operation.description,
			},
		})
	}
	return value
}
