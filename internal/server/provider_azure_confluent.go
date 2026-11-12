package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const azureConfluentAPIVersion = "2024-07-01"

var (
	azureConfluentSubscriptionPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)
	azureConfluentOrganizationPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{1,62}$`)
)

type azureConfluentOrganization struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureConfluentMarketplaceAgreement struct {
	SubscriptionID string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureConfluentChildResource struct {
	SubscriptionID   string
	ResourceGroup    string
	OrganizationName string
	EnvironmentID    string
	ClusterID        string
	Collection       string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ETag             string
}

type azureConfluentInvitation struct {
	SubscriptionID   string
	ResourceGroup    string
	OrganizationName string
	ID               string
	Email            string
	AuthType         string
	Status           string
	CreatedAt        time.Time
	ExpiresAt        time.Time
}

type azureConfluentRoleBinding struct {
	SubscriptionID   string
	ResourceGroup    string
	OrganizationName string
	ID               string
	Principal        string
	RoleName         string
	CRNPattern       string
	CreatedAt        time.Time
}

func (s *Server) handleAzureConfluentRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureConfluentIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureConfluentAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Confluent") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureConfluentOperationsFixture()})
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureConfluentRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}
	if !azureConfluentSubscriptionPattern.MatchString(subscriptionID) {
		respondAzureInvalidRequest(w, path, "subscriptionId is invalid")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "agreements":
		if resourceGroup != "" {
			respondAzureImplemented(w, path)
			return true
		}
		s.handleAzureConfluentAgreements(w, r, path, subscriptionID, tail)
		return true
	case "organizations":
		if resourceGroup == "" {
			s.handleAzureConfluentOrganizationsSubscription(w, r, path, subscriptionID, tail)
			return true
		}
		s.handleAzureConfluentOrganizationsResourceGroup(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	case "validations":
		if resourceGroup == "" {
			respondAzureImplemented(w, path)
			return true
		}
		s.handleAzureConfluentValidations(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		respondAzureImplemented(w, path)
		return true
	}
}

func azureConfluentIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Confluent") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureConfluentRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "agreements", "organizations", "validations":
		return true
	default:
		return false
	}
}

func azureConfluentAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureConfluentRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Confluent") {
			return subscriptionID, resourceGroup, segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureConfluentAgreements(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.providerStorageMu.Lock()
		agreement := s.azureConfluentMarketplaceAgreements[azureConfluentAgreementKey(subscriptionID)]
		s.providerStorageMu.Unlock()
		items := []map[string]any{}
		if agreement != nil {
			items = append(items, azureConfluentMarketplaceAgreementResponse(agreement))
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
		return
	}
	if len(tail) != 2 || !strings.EqualFold(tail[1], "default") {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		s.providerStorageMu.Lock()
		agreement := s.azureConfluentMarketplaceAgreements[azureConfluentAgreementKey(subscriptionID)]
		s.providerStorageMu.Unlock()
		if agreement == nil {
			azureConfluentRespondNotFound(w, "AgreementNotFound", "marketplace agreement was not found")
			return
		}
		respondJSON(w, http.StatusOK, azureConfluentMarketplaceAgreementResponse(agreement))
	case http.MethodPut:
		_, payload, ok := azureConfluentReadJSONBody(w, r, path)
		if !ok {
			return
		}
		now := time.Now().UTC()
		properties, _ := payload["properties"].(map[string]any)
		accepted := azureConfluentBool(properties["accepted"])
		s.providerStorageMu.Lock()
		key := azureConfluentAgreementKey(subscriptionID)
		agreement := s.azureConfluentMarketplaceAgreements[key]
		created := false
		if agreement == nil {
			agreement = &azureConfluentMarketplaceAgreement{
				SubscriptionID: subscriptionID,
				CreatedAt:      now,
			}
			created = true
		}
		agreement.Properties = map[string]any{
			"accepted":          accepted,
			"licenseTextLink":   azureConfluentStringOr(properties["licenseTextLink"], "https://aka.ms/stackyard/confluent/license"),
			"plan":              azureConfluentStringOr(properties["plan"], "enterprise"),
			"privacyPolicyLink": azureConfluentStringOr(properties["privacyPolicyLink"], "https://aka.ms/stackyard/confluent/privacy"),
			"product":           azureConfluentStringOr(properties["product"], "AzureConfluent"),
			"publisher":         azureConfluentStringOr(properties["publisher"], "Microsoft"),
			"retrieveDatetime":  now.Format(time.RFC3339),
			"signature":         azureConfluentStringOr(properties["signature"], "stackyard-local-signature"),
		}
		agreement.UpdatedAt = now
		s.azureConfluentMarketplaceAgreements[key] = agreement
		s.providerStorageMu.Unlock()
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		respondJSON(w, status, azureConfluentMarketplaceAgreementResponse(agreement))
	case http.MethodDelete:
		s.providerStorageMu.Lock()
		delete(s.azureConfluentMarketplaceAgreements, azureConfluentAgreementKey(subscriptionID))
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureConfluentOrganizationsSubscription(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) != 1 || !strings.EqualFold(tail[0], "organizations") {
		respondAzureImplemented(w, path)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, organization := range s.azureConfluentOrganizations {
		if strings.EqualFold(organization.SubscriptionID, subscriptionID) {
			items = append(items, azureConfluentOrganizationResponse(organization))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureConfluentString(items[i]["id"])), strings.ToLower(azureConfluentString(items[j]["id"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureConfluentOrganizationsResourceGroup(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		items := make([]map[string]any, 0)
		for _, organization := range s.azureConfluentOrganizations {
			if strings.EqualFold(organization.SubscriptionID, subscriptionID) && strings.EqualFold(organization.ResourceGroup, resourceGroup) {
				items = append(items, azureConfluentOrganizationResponse(organization))
			}
		}
		sort.Slice(items, func(i, j int) bool {
			return strings.Compare(strings.ToLower(azureConfluentString(items[i]["name"])), strings.ToLower(azureConfluentString(items[j]["name"]))) < 0
		})
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
		return
	}

	organizationName := strings.TrimSpace(tail[1])
	if !azureConfluentOrganizationPattern.MatchString(organizationName) {
		respondAzureInvalidRequest(w, path, "organization name is invalid")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			s.providerStorageMu.Lock()
			organization := s.azureConfluentOrganizations[azureConfluentOrganizationKey(subscriptionID, resourceGroup, organizationName)]
			s.providerStorageMu.Unlock()
			if organization == nil {
				azureConfluentRespondNotFound(w, "OrganizationNotFound", "organization was not found")
				return
			}
			respondJSON(w, http.StatusOK, azureConfluentOrganizationResponse(organization))
		case http.MethodPut:
			s.handleAzureConfluentPutOrganization(w, r, path, subscriptionID, resourceGroup, organizationName)
		case http.MethodPatch:
			s.handleAzureConfluentPatchOrganization(w, r, path, subscriptionID, resourceGroup, organizationName)
		case http.MethodDelete:
			s.handleAzureConfluentDeleteOrganization(w, subscriptionID, resourceGroup, organizationName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "access":
		s.handleAzureConfluentAccess(w, r, path, subscriptionID, resourceGroup, organizationName, tail[2:])
	case "environments":
		s.handleAzureConfluentEnvironments(w, r, path, subscriptionID, resourceGroup, organizationName, tail[2:])
	case "listregions":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureConfluentRegionsFixture()})
	case "schemaregistryclusters":
		s.handleAzureConfluentSchemaRegistryClusters(w, r, path, subscriptionID, resourceGroup, organizationName, tail[2:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureConfluentValidations(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) != 3 || r.Method != http.MethodPost {
		if len(tail) >= 3 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	organizationName := strings.TrimSpace(tail[1])
	validationName := strings.ToLower(strings.TrimSpace(tail[2]))
	if organizationName == "" {
		respondAzureInvalidRequest(w, path, "organizationName is required")
		return
	}
	if validationName != "orgvalidate" && validationName != "orgvalidatev2" {
		respondAzureImplemented(w, path)
		return
	}
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureConfluentString(payload["location"]))
	properties, _ := payload["properties"].(map[string]any)
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	if _, ok := properties["offerDetail"].(map[string]any); !ok {
		respondAzureInvalidRequest(w, path, "properties.offerDetail is required")
		return
	}
	now := time.Now().UTC()
	response := map[string]any{
		"id":       azureConfluentOrganizationID(subscriptionID, resourceGroup, organizationName),
		"name":     organizationName,
		"type":     "Microsoft.Confluent/organizations",
		"location": location,
		"tags":     azureConfluentObjectMap(payload["tags"]),
		"properties": map[string]any{
			"createdTime":       now.Format(time.RFC3339),
			"offerDetail":       azureConfluentCloneMap(properties["offerDetail"]),
			"organizationId":    azureConfluentStableToken("org", subscriptionID, resourceGroup, organizationName),
			"provisioningState": "Succeeded",
			"ssoUrl":            fmt.Sprintf("https://confluent.cloud/organizations/%s", organizationName),
			"userDetail":        azureConfluentCloneMap(properties["userDetail"]),
		},
		"systemData": azureConfluentSystemData(now),
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleAzureConfluentPutOrganization(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string) {
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureConfluentString(payload["location"]))
	properties, _ := payload["properties"].(map[string]any)
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	if _, ok := properties["offerDetail"].(map[string]any); !ok {
		respondAzureInvalidRequest(w, path, "properties.offerDetail is required")
		return
	}
	now := time.Now().UTC()
	key := azureConfluentOrganizationKey(subscriptionID, resourceGroup, organizationName)
	s.providerStorageMu.Lock()
	organization := s.azureConfluentOrganizations[key]
	created := false
	if organization == nil {
		organization = &azureConfluentOrganization{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           organizationName,
			CreatedAt:      now,
		}
		created = true
	}
	organization.Location = location
	organization.Tags = azureConfluentNormalizeTags(payload["tags"])
	organization.Properties = map[string]any{
		"createdTime":       organization.CreatedAt.Format(time.RFC3339),
		"linkOrganization":  azureConfluentCloneMap(properties["linkOrganization"]),
		"offerDetail":       azureConfluentCloneMap(properties["offerDetail"]),
		"organizationId":    azureConfluentStableToken("org", subscriptionID, resourceGroup, organizationName),
		"provisioningState": "Succeeded",
		"ssoUrl":            fmt.Sprintf("https://confluent.cloud/organizations/%s", organizationName),
		"userDetail":        azureConfluentCloneMap(properties["userDetail"]),
	}
	organization.UpdatedAt = now
	organization.ETag = azureConfluentETag("organization", subscriptionID, resourceGroup, organizationName, now.Format(time.RFC3339Nano))
	s.azureConfluentOrganizations[key] = organization
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureConfluentOrganizationResponse(organization))
}

func (s *Server) handleAzureConfluentPatchOrganization(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string) {
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	organization := s.azureConfluentOrganizations[azureConfluentOrganizationKey(subscriptionID, resourceGroup, organizationName)]
	if organization == nil {
		azureConfluentRespondNotFound(w, "OrganizationNotFound", "organization was not found")
		return
	}

	updated := false
	if tags, ok := payload["tags"]; ok {
		organization.Tags = azureConfluentNormalizeTags(tags)
		updated = true
	}
	if properties, ok := payload["properties"].(map[string]any); ok {
		for _, key := range []string{"offerDetail", "userDetail", "linkOrganization"} {
			if value, exists := properties[key]; exists {
				organization.Properties[key] = azureConfluentCloneMap(value)
				updated = true
			}
		}
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	organization.UpdatedAt = time.Now().UTC()
	organization.ETag = azureConfluentETag("organization", subscriptionID, resourceGroup, organizationName, organization.UpdatedAt.Format(time.RFC3339Nano))
	respondJSON(w, http.StatusOK, azureConfluentOrganizationResponse(organization))
}

func (s *Server) handleAzureConfluentDeleteOrganization(w http.ResponseWriter, subscriptionID, resourceGroup, organizationName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	orgKey := azureConfluentOrganizationKey(subscriptionID, resourceGroup, organizationName)
	delete(s.azureConfluentOrganizations, orgKey)
	for key, item := range s.azureConfluentResources {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.OrganizationName, organizationName) {
			delete(s.azureConfluentResources, key)
		}
	}
	for key, item := range s.azureConfluentInvitations {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.OrganizationName, organizationName) {
			delete(s.azureConfluentInvitations, key)
		}
	}
	for key, item := range s.azureConfluentRoleBindings {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.OrganizationName, organizationName) {
			delete(s.azureConfluentRoleBindings, key)
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleAzureConfluentAccess(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string, tail []string) {
	if len(tail) < 3 || !strings.EqualFold(tail[1], "default") {
		respondAzureImplemented(w, path)
		return
	}
	action := strings.ToLower(strings.TrimSpace(tail[2]))
	switch action {
	case "listclusters":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodPost {
			if _, _, ok := azureConfluentReadJSONBody(w, r, path); !ok {
				return
			}
		}
		respondJSON(w, http.StatusOK, s.azureConfluentAccessListClusters(subscriptionID, resourceGroup, organizationName))
	case "listenvironments":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodPost {
			if _, _, ok := azureConfluentReadJSONBody(w, r, path); !ok {
				return
			}
		}
		respondJSON(w, http.StatusOK, s.azureConfluentAccessListEnvironments(subscriptionID, resourceGroup, organizationName))
	case "listinvitations":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, s.azureConfluentAccessListInvitations(subscriptionID, resourceGroup, organizationName))
	case "listrolebindingnamelist":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"data": []map[string]any{
				{"id": "OrganizationAdmin", "display_name": "OrganizationAdmin"},
				{"id": "EnvironmentAdmin", "display_name": "EnvironmentAdmin"},
				{"id": "CloudClusterAdmin", "display_name": "CloudClusterAdmin"},
				{"id": "DeveloperRead", "display_name": "DeveloperRead"},
			},
			"total_size": 4,
		})
	case "listrolebindings":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, s.azureConfluentAccessListRoleBindings(subscriptionID, resourceGroup, organizationName))
	case "listserviceaccounts":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, s.azureConfluentAccessListServiceAccounts(subscriptionID, resourceGroup, organizationName))
	case "listusers":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, s.azureConfluentAccessListUsers(subscriptionID, resourceGroup, organizationName))
	case "createinvitation":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConfluentCreateInvitation(w, r, path, subscriptionID, resourceGroup, organizationName)
	case "createrolebinding":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConfluentCreateRoleBinding(w, r, path, subscriptionID, resourceGroup, organizationName)
	case "deleterolebinding":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConfluentDeleteRoleBinding(w, r, path, subscriptionID, resourceGroup, organizationName)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureConfluentCreateInvitation(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string) {
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	invitedUserDetails, _ := payload["invitedUserDetails"].(map[string]any)
	email := azureConfluentStringOr(invitedUserDetails["invitedEmail"], azureConfluentString(payload["email"]))
	if strings.TrimSpace(email) == "" {
		respondAzureInvalidRequest(w, path, "invited user email is required")
		return
	}
	authType := azureConfluentStringOr(invitedUserDetails["auth_type"], "AUTH_TYPE_SSO")
	now := time.Now().UTC()
	invitation := &azureConfluentInvitation{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		OrganizationName: organizationName,
		ID:               "inv-" + azureConfluentStableToken("inv", organizationName, email)[:10],
		Email:            email,
		AuthType:         authType,
		Status:           "INVITE_STATUS_SENT",
		CreatedAt:        now,
		ExpiresAt:        now.Add(24 * time.Hour),
	}
	s.providerStorageMu.Lock()
	s.azureConfluentInvitations[azureConfluentInvitationKey(subscriptionID, resourceGroup, organizationName, invitation.ID)] = invitation
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureConfluentInvitationRecord(invitation))
}

func (s *Server) handleAzureConfluentCreateRoleBinding(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string) {
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	roleName := azureConfluentStringOr(payload["role_name"], "DeveloperRead")
	principal := azureConfluentStringOr(payload["principal"], azureConfluentString(payload["principal_id"]))
	if strings.TrimSpace(principal) == "" {
		principal = "sa-" + azureConfluentStableToken("principal", organizationName)[:8]
	}
	crnPattern := azureConfluentStringOr(payload["crn_pattern"], fmt.Sprintf("crn://confluent.cloud/organization=%s", organizationName))
	now := time.Now().UTC()
	roleBinding := &azureConfluentRoleBinding{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		OrganizationName: organizationName,
		ID:               "rb-" + azureConfluentStableToken("rolebinding", organizationName, principal, roleName)[:12],
		Principal:        principal,
		RoleName:         roleName,
		CRNPattern:       crnPattern,
		CreatedAt:        now,
	}
	s.providerStorageMu.Lock()
	s.azureConfluentRoleBindings[azureConfluentRoleBindingKey(subscriptionID, resourceGroup, organizationName, roleBinding.ID)] = roleBinding
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureConfluentRoleBindingRecord(roleBinding))
}

func (s *Server) handleAzureConfluentDeleteRoleBinding(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string) {
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	roleBindingID := strings.TrimSpace(azureConfluentString(payload["id"]))
	if roleBindingID == "" {
		roleBindingID = strings.TrimSpace(azureConfluentString(payload["roleBindingId"]))
	}
	if roleBindingID == "" {
		roleBindingID = strings.TrimSpace(azureConfluentString(payload["name"]))
	}
	if roleBindingID == "" {
		respondAzureInvalidRequest(w, path, "role binding id is required")
		return
	}
	s.providerStorageMu.Lock()
	delete(s.azureConfluentRoleBindings, azureConfluentRoleBindingKey(subscriptionID, resourceGroup, organizationName, roleBindingID))
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": roleBindingID})
}

func (s *Server) handleAzureConfluentEnvironments(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConfluentListResources(w, subscriptionID, resourceGroup, organizationName, "", "environments")
		return
	}
	environmentID := strings.TrimSpace(tail[1])
	if environmentID == "" {
		respondAzureInvalidRequest(w, path, "environmentId is required")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			s.handleAzureConfluentGetResource(w, subscriptionID, resourceGroup, organizationName, environmentID, "", "environments", environmentID)
		case http.MethodPut:
			s.handleAzureConfluentPutChild(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, "", "environments", environmentID)
		case http.MethodDelete:
			s.handleAzureConfluentDeleteChild(w, subscriptionID, resourceGroup, organizationName, environmentID, "", "environments", environmentID)
		case http.MethodPatch:
			s.handleAzureConfluentPatchChild(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, "", "environments", environmentID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "clusters":
		s.handleAzureConfluentClusters(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, tail[2:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureConfluentClusters(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName, environmentID string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConfluentListResources(w, subscriptionID, resourceGroup, organizationName, environmentID, "clusters")
		return
	}
	clusterID := strings.TrimSpace(tail[1])
	if clusterID == "" {
		respondAzureInvalidRequest(w, path, "clusterId is required")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			s.handleAzureConfluentGetResource(w, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, "clusters", clusterID)
		case http.MethodPut:
			s.handleAzureConfluentPutChild(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, "clusters", clusterID)
		case http.MethodDelete:
			s.handleAzureConfluentDeleteChild(w, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, "clusters", clusterID)
		case http.MethodPatch:
			s.handleAzureConfluentPatchChild(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, "clusters", clusterID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "topics":
		s.handleAzureConfluentTopicLikeCollection(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, "topics", tail[2:])
	case "connectors":
		s.handleAzureConfluentTopicLikeCollection(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, "connectors", tail[2:])
	case "apikeys":
		s.handleAzureConfluentTopicLikeCollection(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, "apiKeys", tail[2:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureConfluentSchemaRegistryClusters(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConfluentListResources(w, subscriptionID, resourceGroup, organizationName, "", "schemaRegistryClusters")
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "schema registry cluster id is required")
		return
	}
	if len(tail) != 2 || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureConfluentGetResource(w, subscriptionID, resourceGroup, organizationName, "", "", "schemaRegistryClusters", name)
}

func (s *Server) handleAzureConfluentTopicLikeCollection(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureConfluentListResources(w, subscriptionID, resourceGroup, organizationName, environmentID+"::"+clusterID, collection)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, strings.TrimSuffix(collection, "s")+" name is required")
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		s.handleAzureConfluentGetResource(w, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)
	case http.MethodPut:
		s.handleAzureConfluentPutChild(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)
	case http.MethodPatch:
		s.handleAzureConfluentPatchChild(w, r, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)
	case http.MethodDelete:
		s.handleAzureConfluentDeleteChild(w, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureConfluentListResources(w http.ResponseWriter, subscriptionID, resourceGroup, organizationName, scope, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureConfluentResources {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) ||
			!strings.EqualFold(resource.ResourceGroup, resourceGroup) ||
			!strings.EqualFold(resource.OrganizationName, organizationName) ||
			!strings.EqualFold(resource.Collection, collection) {
			continue
		}
		if scope != "" && collection == "clusters" && !strings.Contains(scope, "::") && !strings.EqualFold(resource.EnvironmentID, scope) {
			continue
		}
		if strings.Contains(scope, "::") {
			parts := strings.SplitN(scope, "::", 2)
			if len(parts) == 2 {
				if !strings.EqualFold(resource.EnvironmentID, parts[0]) || !strings.EqualFold(resource.ClusterID, parts[1]) {
					continue
				}
			}
		}
		items = append(items, azureConfluentChildResponse(resource))
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureConfluentString(items[i]["name"])), strings.ToLower(azureConfluentString(items[j]["name"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureConfluentGetResource(w http.ResponseWriter, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureConfluentResources[azureConfluentResourceKey(subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		if collection == "schemaRegistryClusters" {
			resource = azureConfluentSeededSchemaRegistryResource(subscriptionID, resourceGroup, organizationName, name)
		}
	}
	if resource == nil {
		azureConfluentRespondNotFound(w, "ResourceNotFound", "resource was not found")
		return
	}
	respondJSON(w, http.StatusOK, azureConfluentChildResponse(resource))
}

func (s *Server) handleAzureConfluentPutChild(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name string) {
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureConfluentString(payload["location"]))
	if collection == "environments" {
		location = azureConfluentStringOr(payload["location"], "global")
	}
	if location == "" {
		location = "global"
	}
	now := time.Now().UTC()
	key := azureConfluentResourceKey(subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)
	s.providerStorageMu.Lock()
	resource := s.azureConfluentResources[key]
	created := false
	if resource == nil {
		resource = &azureConfluentChildResource{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			OrganizationName: organizationName,
			EnvironmentID:    environmentID,
			ClusterID:        clusterID,
			Collection:       collection,
			Name:             name,
			CreatedAt:        now,
		}
		created = true
	}
	resource.Location = location
	resource.Tags = azureConfluentNormalizeTags(payload["tags"])
	resource.Properties = azureConfluentConfluentChildProperties(collection, organizationName, environmentID, clusterID, name, payload["properties"])
	resource.UpdatedAt = now
	resource.ETag = azureConfluentETag(collection, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, name, now.Format(time.RFC3339Nano))
	s.azureConfluentResources[key] = resource
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureConfluentChildResponse(resource))
}

func (s *Server) handleAzureConfluentPatchChild(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name string) {
	_, payload, ok := azureConfluentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	key := azureConfluentResourceKey(subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	resource := s.azureConfluentResources[key]
	if resource == nil {
		azureConfluentRespondNotFound(w, "ResourceNotFound", "resource was not found")
		return
	}
	updated := false
	if tags, ok := payload["tags"]; ok {
		resource.Tags = azureConfluentNormalizeTags(tags)
		updated = true
	}
	if properties, ok := payload["properties"].(map[string]any); ok {
		if resource.Properties == nil {
			resource.Properties = map[string]any{}
		}
		for key, value := range properties {
			resource.Properties[key] = value
		}
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureConfluentETag(collection, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, name, resource.UpdatedAt.Format(time.RFC3339Nano))
	respondJSON(w, http.StatusOK, azureConfluentChildResponse(resource))
}

func (s *Server) handleAzureConfluentDeleteChild(w http.ResponseWriter, subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	delete(s.azureConfluentResources, azureConfluentResourceKey(subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name))
	if collection == "environments" {
		for key, item := range s.azureConfluentResources {
			if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
				strings.EqualFold(item.ResourceGroup, resourceGroup) &&
				strings.EqualFold(item.OrganizationName, organizationName) &&
				strings.EqualFold(item.EnvironmentID, name) {
				delete(s.azureConfluentResources, key)
			}
		}
	}
	if collection == "clusters" {
		for key, item := range s.azureConfluentResources {
			if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
				strings.EqualFold(item.ResourceGroup, resourceGroup) &&
				strings.EqualFold(item.OrganizationName, organizationName) &&
				strings.EqualFold(item.EnvironmentID, environmentID) &&
				strings.EqualFold(item.ClusterID, name) {
				delete(s.azureConfluentResources, key)
			}
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) azureConfluentAccessListEnvironments(subscriptionID, resourceGroup, organizationName string) map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureConfluentResources {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) &&
			strings.EqualFold(resource.ResourceGroup, resourceGroup) &&
			strings.EqualFold(resource.OrganizationName, organizationName) &&
			strings.EqualFold(resource.Collection, "environments") {
			items = append(items, map[string]any{
				"id":    resource.Name,
				"name":  resource.Name,
				"kind":  azureConfluentStringOr(resource.Properties["kind"], "StreamGovernance"),
				"topic": fmt.Sprintf("crn://confluent.cloud/organization=%s/environment=%s", organizationName, resource.Name),
			})
		}
	}
	return azureConfluentListEnvelope(items)
}

func (s *Server) azureConfluentAccessListClusters(subscriptionID, resourceGroup, organizationName string) map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureConfluentResources {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) &&
			strings.EqualFold(resource.ResourceGroup, resourceGroup) &&
			strings.EqualFold(resource.OrganizationName, organizationName) &&
			strings.EqualFold(resource.Collection, "clusters") {
			items = append(items, map[string]any{
				"id":   resource.Name,
				"name": resource.Name,
				"spec": map[string]any{
					"environment": map[string]any{"id": resource.EnvironmentID},
					"config":      map[string]any{"cloud": "AZURE", "region": azureConfluentStringOr(resource.Properties["region"], "eastus")},
				},
				"status": map[string]any{
					"phase": azureConfluentStringOr(resource.Properties["provisioningState"], "PROVISIONED"),
				},
			})
		}
	}
	return azureConfluentListEnvelope(items)
}

func (s *Server) azureConfluentAccessListInvitations(subscriptionID, resourceGroup, organizationName string) map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, invitation := range s.azureConfluentInvitations {
		if strings.EqualFold(invitation.SubscriptionID, subscriptionID) &&
			strings.EqualFold(invitation.ResourceGroup, resourceGroup) &&
			strings.EqualFold(invitation.OrganizationName, organizationName) {
			items = append(items, azureConfluentInvitationRecord(invitation))
		}
	}
	return azureConfluentListEnvelope(items)
}

func (s *Server) azureConfluentAccessListRoleBindings(subscriptionID, resourceGroup, organizationName string) map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := []map[string]any{
		{
			"id":          "rb-owner",
			"principal":   "stackyard-owner",
			"role_name":   "OrganizationAdmin",
			"crn_pattern": fmt.Sprintf("crn://confluent.cloud/organization=%s", organizationName),
		},
	}
	for _, binding := range s.azureConfluentRoleBindings {
		if strings.EqualFold(binding.SubscriptionID, subscriptionID) &&
			strings.EqualFold(binding.ResourceGroup, resourceGroup) &&
			strings.EqualFold(binding.OrganizationName, organizationName) {
			items = append(items, azureConfluentRoleBindingRecord(binding))
		}
	}
	return azureConfluentListEnvelope(items)
}

func (s *Server) azureConfluentAccessListServiceAccounts(subscriptionID, resourceGroup, organizationName string) map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := []map[string]any{
		{
			"id":          "sa-stackyard",
			"name":        "stackyard-sa",
			"description": "Seeded Stackyard service account",
		},
	}
	for _, resource := range s.azureConfluentResources {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) &&
			strings.EqualFold(resource.ResourceGroup, resourceGroup) &&
			strings.EqualFold(resource.OrganizationName, organizationName) &&
			strings.EqualFold(resource.Collection, "apiKeys") {
			items = append(items, map[string]any{
				"id":          resource.Name,
				"name":        resource.Name,
				"description": "API-key backed service account",
			})
		}
	}
	return azureConfluentListEnvelope(items)
}

func (s *Server) azureConfluentAccessListUsers(subscriptionID, resourceGroup, organizationName string) map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := []map[string]any{
		{
			"id":    "user-owner",
			"email": "owner@example.com",
			"name":  "Stackyard Owner",
		},
	}
	if organization := s.azureConfluentOrganizations[azureConfluentOrganizationKey(subscriptionID, resourceGroup, organizationName)]; organization != nil {
		if detail, ok := organization.Properties["userDetail"].(map[string]any); ok {
			email := azureConfluentStringOr(detail["emailAddress"], azureConfluentString(detail["aadEmail"]))
			if email != "" {
				items = append(items, map[string]any{
					"id":    "user-subscriber",
					"email": email,
					"name":  strings.TrimSpace(azureConfluentString(detail["firstName"]) + " " + azureConfluentString(detail["lastName"])),
				})
			}
		}
	}
	return azureConfluentListEnvelope(items)
}

func azureConfluentOperationsFixture() []map[string]any {
	names := []string{
		"Microsoft.Confluent/organizations/read",
		"Microsoft.Confluent/organizations/write",
		"Microsoft.Confluent/organizations/delete",
		"Microsoft.Confluent/organizations/access/read",
		"Microsoft.Confluent/organizations/access/write",
		"Microsoft.Confluent/organizations/environments/read",
		"Microsoft.Confluent/organizations/environments/write",
		"Microsoft.Confluent/organizations/environments/delete",
		"Microsoft.Confluent/organizations/environments/clusters/read",
		"Microsoft.Confluent/organizations/environments/clusters/write",
		"Microsoft.Confluent/organizations/environments/clusters/delete",
		"Microsoft.Confluent/organizations/environments/clusters/topics/read",
		"Microsoft.Confluent/organizations/environments/clusters/topics/write",
		"Microsoft.Confluent/organizations/environments/clusters/topics/delete",
		"Microsoft.Confluent/organizations/environments/clusters/connectors/read",
		"Microsoft.Confluent/organizations/environments/clusters/connectors/write",
		"Microsoft.Confluent/organizations/environments/clusters/connectors/delete",
		"Microsoft.Confluent/organizations/environments/clusters/apiKeys/read",
		"Microsoft.Confluent/organizations/environments/clusters/apiKeys/write",
		"Microsoft.Confluent/organizations/environments/clusters/apiKeys/delete",
		"Microsoft.Confluent/agreements/read",
		"Microsoft.Confluent/agreements/write",
		"Microsoft.Confluent/agreements/delete",
		"Microsoft.Confluent/validations/action",
	}
	ops := make([]map[string]any, 0, len(names))
	for _, name := range names {
		verb := "Read"
		switch {
		case strings.HasSuffix(name, "/write"):
			verb = "Write"
		case strings.HasSuffix(name, "/delete"):
			verb = "Delete"
		case strings.HasSuffix(name, "/action"):
			verb = "Action"
		}
		ops = append(ops, map[string]any{
			"name":   name,
			"origin": "system",
			"display": map[string]any{
				"provider":  "Microsoft Confluent",
				"resource":  "Confluent",
				"operation": verb,
			},
			"isDataAction": false,
		})
	}
	return ops
}

func azureConfluentRegionsFixture() []map[string]any {
	return []map[string]any{
		{"id": "eastus", "cloud": "AZURE", "displayName": "East US"},
		{"id": "westus2", "cloud": "AZURE", "displayName": "West US 2"},
	}
}

func azureConfluentOrganizationResponse(resource *azureConfluentOrganization) map[string]any {
	systemData := azureConfluentSystemData(resource.UpdatedAt)
	systemData["createdAt"] = resource.CreatedAt.Format(time.RFC3339)
	systemData["lastModifiedAt"] = resource.UpdatedAt.Format(time.RFC3339)
	return map[string]any{
		"id":         azureConfluentOrganizationID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.Confluent/organizations",
		"location":   resource.Location,
		"tags":       azureConfluentStringMapAny(resource.Tags),
		"etag":       resource.ETag,
		"properties": azureConfluentCloneMap(resource.Properties),
		"systemData": systemData,
	}
}

func azureConfluentMarketplaceAgreementResponse(resource *azureConfluentMarketplaceAgreement) map[string]any {
	return map[string]any{
		"id":         fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Confluent/agreements/default", resource.SubscriptionID),
		"name":       "default",
		"type":       "Microsoft.Confluent/agreements",
		"properties": azureConfluentCloneMap(resource.Properties),
		"systemData": azureConfluentSystemData(resource.UpdatedAt),
	}
}

func azureConfluentChildResponse(resource *azureConfluentChildResource) map[string]any {
	systemData := azureConfluentSystemData(resource.UpdatedAt)
	systemData["createdAt"] = resource.CreatedAt.Format(time.RFC3339)
	systemData["lastModifiedAt"] = resource.UpdatedAt.Format(time.RFC3339)
	return map[string]any{
		"id":         azureConfluentChildID(resource),
		"name":       resource.Name,
		"type":       azureConfluentChildType(resource.Collection),
		"location":   resource.Location,
		"tags":       azureConfluentStringMapAny(resource.Tags),
		"etag":       resource.ETag,
		"properties": azureConfluentCloneMap(resource.Properties),
		"systemData": systemData,
	}
}

func azureConfluentInvitationRecord(invitation *azureConfluentInvitation) map[string]any {
	return map[string]any{
		"accepted_at": invitation.CreatedAt.Format(time.RFC3339),
		"auth_type":   invitation.AuthType,
		"email":       invitation.Email,
		"expires_at":  invitation.ExpiresAt.Format(time.RFC3339),
		"id":          invitation.ID,
		"kind":        "Invitation",
		"metadata": map[string]any{
			"created_at":    invitation.CreatedAt.Format(time.RFC3339),
			"resource_name": fmt.Sprintf("crn://confluent.cloud/organization=%s/invitation=%s", invitation.OrganizationName, invitation.ID),
			"self":          fmt.Sprintf("https://api.confluent.cloud/iam/v2/invitations/%s", invitation.ID),
			"updated_at":    invitation.CreatedAt.Format(time.RFC3339),
		},
		"status": invitation.Status,
	}
}

func azureConfluentRoleBindingRecord(roleBinding *azureConfluentRoleBinding) map[string]any {
	return map[string]any{
		"id":          roleBinding.ID,
		"principal":   roleBinding.Principal,
		"role_name":   roleBinding.RoleName,
		"crn_pattern": roleBinding.CRNPattern,
		"metadata": map[string]any{
			"created_at":    roleBinding.CreatedAt.Format(time.RFC3339),
			"resource_name": fmt.Sprintf("crn://confluent.cloud/organization=%s/rolebinding=%s", roleBinding.OrganizationName, roleBinding.ID),
			"updated_at":    roleBinding.CreatedAt.Format(time.RFC3339),
		},
	}
}

func azureConfluentSeededSchemaRegistryResource(subscriptionID, resourceGroup, organizationName, name string) *azureConfluentChildResource {
	now := time.Now().UTC()
	return &azureConfluentChildResource{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		OrganizationName: organizationName,
		Collection:       "schemaRegistryClusters",
		Name:             name,
		Location:         "global",
		CreatedAt:        now,
		UpdatedAt:        now,
		ETag:             azureConfluentETag("schemaRegistryClusters", subscriptionID, resourceGroup, organizationName, name),
		Properties: map[string]any{
			"clusterType":       "SCHEMA_REGISTRY",
			"environmentId":     "env-default",
			"resourceName":      name,
			"restEndpoint":      fmt.Sprintf("https://%s.aws.confluent.cloud", strings.ToLower(name)),
			"provisioningState": "Succeeded",
		},
	}
}

func azureConfluentConfluentChildProperties(collection, organizationName, environmentID, clusterID, name string, source any) map[string]any {
	properties := azureConfluentCloneMap(source)
	switch collection {
	case "environments":
		properties["kind"] = azureConfluentStringOr(properties["kind"], "StreamGovernance")
		properties["displayName"] = azureConfluentStringOr(properties["displayName"], name)
		properties["provisioningState"] = "Succeeded"
	case "clusters":
		properties["environmentId"] = environmentID
		properties["bootstrapEndpoint"] = azureConfluentStringOr(properties["bootstrapEndpoint"], fmt.Sprintf("%s-%s.confluent.cloud:9092", organizationName, name))
		properties["region"] = azureConfluentStringOr(properties["region"], "eastus")
		properties["provisioningState"] = "Succeeded"
	case "topics":
		properties["clusterId"] = clusterID
		if _, ok := properties["partitionsCount"]; !ok {
			properties["partitionsCount"] = 3
		}
		if _, ok := properties["replicationFactor"]; !ok {
			properties["replicationFactor"] = 3
		}
		properties["provisioningState"] = "Succeeded"
	case "connectors":
		properties["clusterId"] = clusterID
		properties["connectorType"] = azureConfluentStringOr(properties["connectorType"], "Sink")
		properties["connectorClass"] = azureConfluentStringOr(properties["connectorClass"], "FileStreamSink")
		properties["status"] = azureConfluentStringOr(properties["status"], "RUNNING")
	case "apiKeys":
		properties["clusterId"] = clusterID
		properties["key"] = azureConfluentStringOr(properties["key"], "key-"+azureConfluentStableToken("apikey", name)[:10])
		properties["secret"] = azureConfluentStringOr(properties["secret"], "secret-"+azureConfluentStableToken("secret", name)[:18])
		properties["description"] = azureConfluentStringOr(properties["description"], "Stackyard Confluent API key")
	case "schemaRegistryClusters":
		properties["restEndpoint"] = azureConfluentStringOr(properties["restEndpoint"], fmt.Sprintf("https://%s.aws.confluent.cloud", strings.ToLower(name)))
		properties["clusterType"] = azureConfluentStringOr(properties["clusterType"], "SCHEMA_REGISTRY")
	}
	return properties
}

func azureConfluentChildID(resource *azureConfluentChildResource) string {
	base := azureConfluentOrganizationID(resource.SubscriptionID, resource.ResourceGroup, resource.OrganizationName)
	switch resource.Collection {
	case "environments":
		return base + "/environments/" + resource.Name
	case "clusters":
		return base + "/environments/" + resource.EnvironmentID + "/clusters/" + resource.Name
	case "topics":
		return base + "/environments/" + resource.EnvironmentID + "/clusters/" + resource.ClusterID + "/topics/" + resource.Name
	case "connectors":
		return base + "/environments/" + resource.EnvironmentID + "/clusters/" + resource.ClusterID + "/connectors/" + resource.Name
	case "apiKeys":
		return base + "/environments/" + resource.EnvironmentID + "/clusters/" + resource.ClusterID + "/apiKeys/" + resource.Name
	case "schemaRegistryClusters":
		return base + "/schemaRegistryClusters/" + resource.Name
	default:
		return base + "/" + resource.Collection + "/" + resource.Name
	}
}

func azureConfluentChildType(collection string) string {
	switch collection {
	case "environments":
		return "Microsoft.Confluent/organizations/environments"
	case "clusters":
		return "Microsoft.Confluent/organizations/environments/clusters"
	case "topics":
		return "Microsoft.Confluent/organizations/environments/clusters/topics"
	case "connectors":
		return "Microsoft.Confluent/organizations/environments/clusters/connectors"
	case "apiKeys":
		return "Microsoft.Confluent/organizations/environments/clusters/apiKeys"
	case "schemaRegistryClusters":
		return "Microsoft.Confluent/organizations/schemaRegistryClusters"
	default:
		return "Microsoft.Confluent/" + collection
	}
}

func azureConfluentOrganizationID(subscriptionID, resourceGroup, organizationName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Confluent/organizations/%s", subscriptionID, resourceGroup, organizationName)
}

func azureConfluentOrganizationKey(subscriptionID, resourceGroup, organizationName string) string {
	return azureConfluentKey(subscriptionID, resourceGroup, organizationName)
}

func azureConfluentAgreementKey(subscriptionID string) string {
	return azureConfluentKey(subscriptionID, "agreement", "default")
}

func azureConfluentResourceKey(subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name string) string {
	return azureConfluentKey(subscriptionID, resourceGroup, organizationName, environmentID, clusterID, collection, name)
}

func azureConfluentInvitationKey(subscriptionID, resourceGroup, organizationName, invitationID string) string {
	return azureConfluentKey(subscriptionID, resourceGroup, organizationName, "invitation", invitationID)
}

func azureConfluentRoleBindingKey(subscriptionID, resourceGroup, organizationName, roleBindingID string) string {
	return azureConfluentKey(subscriptionID, resourceGroup, organizationName, "rolebinding", roleBindingID)
}

func azureConfluentKey(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(part)))
	}
	return strings.Join(normalized, "::")
}

func azureConfluentReadJSONBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	body, err := io.ReadAll(r.Body)
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
		return body, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureConfluentRespondNotFound(w http.ResponseWriter, code, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azureConfluentString(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func azureConfluentStringOr(value any, fallback string) string {
	if s := azureConfluentString(value); s != "" {
		return s
	}
	return fallback
}

func azureConfluentBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}

func azureConfluentCloneMap(value any) map[string]any {
	source, ok := value.(map[string]any)
	if !ok || len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, item := range source {
		cloned[key] = item
	}
	return cloned
}

func azureConfluentObjectMap(value any) map[string]any {
	source, ok := value.(map[string]any)
	if !ok || len(source) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(source))
	for key, item := range source {
		out[key] = item
	}
	return out
}

func azureConfluentNormalizeTags(value any) map[string]string {
	source, ok := value.(map[string]any)
	if !ok || len(source) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for key, item := range source {
		out[key] = azureConfluentString(item)
	}
	return out
}

func azureConfluentStringMapAny(source map[string]string) map[string]any {
	out := make(map[string]any, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func azureConfluentStableToken(parts ...string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, strings.Join(parts, "|"))
	return hex.EncodeToString(h.Sum(nil))
}

func azureConfluentETag(parts ...string) string {
	return `"` + azureConfluentStableToken(parts...)[:24] + `"`
}

func azureConfluentSystemData(now time.Time) map[string]any {
	return map[string]any{
		"createdAt":          now.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"lastModifiedAt":     now.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
	}
}

func azureConfluentListEnvelope(items []map[string]any) map[string]any {
	return map[string]any{
		"data":       items,
		"total_size": len(items),
		"next":       "",
		"prev":       "",
		"last":       "",
	}
}
