package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureCloudNGFWAPIVersion = "2025-10-08"

type azureCloudNGFWFirewall struct {
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

type azureCloudNGFWLocalRulestack struct {
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

type azureCloudNGFWGlobalRulestack struct {
	Name       string
	Location   string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureCloudNGFWChildResource struct {
	SubscriptionID string
	ResourceGroup  string
	ParentName     string
	Collection     string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureCloudNGFWGlobalChildResource struct {
	ParentName string
	Collection string
	Name       string
	Location   string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

var azureCloudNGFWLocalChildCollections = map[string]string{
	"localrules":   "localRules",
	"prefixlists":  "prefixlists",
	"fqdnlists":    "fqdnlists",
	"certificates": "certificates",
}

var azureCloudNGFWGlobalChildCollections = map[string]string{
	"prefixlists":  "prefixlists",
	"fqdnlists":    "fqdnlists",
	"certificates": "certificates",
}

func (s *Server) handleAzureCloudNGFWRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureCloudNGFWIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureCloudNGFWAllowedMethod(r.Method) {
		return false
	}

	if tail, ok := azureCloudNGFWProviderRouteContext(segments); ok {
		if len(tail) == 1 && strings.EqualFold(tail[0], "operations") {
			if r.Method == http.MethodGet {
				respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWOperationsFixture()})
				return true
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		if len(tail) > 0 && strings.EqualFold(tail[0], "globalRulestacks") {
			s.handleAzureCloudNGFWGlobalRulestacksRoute(w, r, path, tail)
			return true
		}
		return false
	}

	subscriptionID, resourceGroup, tail, ok := azureCloudNGFWRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "firewalls":
		s.handleAzureCloudNGFWFirewallsRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	case "localrulestacks":
		s.handleAzureCloudNGFWLocalRulestacksRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		return false
	}
}

func azureCloudNGFWIsPath(segments []string) bool {
	if tail, ok := azureCloudNGFWProviderRouteContext(segments); ok && len(tail) > 0 {
		switch strings.ToLower(strings.TrimSpace(tail[0])) {
		case "operations", "globalrulestacks":
			return true
		}
	}
	_, _, tail, ok := azureCloudNGFWRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "firewalls", "localrulestacks":
		return true
	default:
		return false
	}
}

func azureCloudNGFWAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureCloudNGFWProviderRouteContext(segments []string) (tail []string, ok bool) {
	if len(segments) < 3 {
		return nil, false
	}
	if strings.EqualFold(segments[0], "providers") && strings.EqualFold(segments[1], "PaloAltoNetworks.Cloudngfw") {
		return segments[2:], true
	}
	return nil, false
}

func azureCloudNGFWRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
		for idx := 4; idx+1 < len(segments); idx++ {
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "PaloAltoNetworks.Cloudngfw") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "PaloAltoNetworks.Cloudngfw") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureCloudNGFWFirewallsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureCloudNGFWFirewallsListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureCloudNGFWFirewallsListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCloudNGFWValidName(name) {
		respondAzureInvalidRequest(w, path, "firewall name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCloudNGFWFirewallGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureCloudNGFWFirewallGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureCloudNGFWFirewallCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureCloudNGFWFirewallPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureCloudNGFWFirewallDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "statuses":
		s.handleAzureCloudNGFWFirewallStatusesRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	case "getglobalrulestack":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWFirewallGetGlobalRulestack(w, path, subscriptionID, resourceGroup, name)
			return
		}
	case "getlogprofile":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWFirewallGetLogProfile(w, path, subscriptionID, resourceGroup, name)
			return
		}
	case "savelogprofile":
		if r.Method == http.MethodPost {
			s.handleAzureCloudNGFWFirewallSaveLogProfile(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	case "getsupportinfo":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWFirewallGetSupportInfo(w, path, subscriptionID, resourceGroup, name)
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCloudNGFWFirewallStatusesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, firewallName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, map[string]any{"value": s.azureCloudNGFWFirewallStatusesFixture(subscriptionID, resourceGroup, firewallName)})
		return
	}
	if len(tail) == 2 && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		statusName := strings.TrimSpace(tail[1])
		items := s.azureCloudNGFWFirewallStatusesFixture(subscriptionID, resourceGroup, firewallName)
		for _, item := range items {
			if strings.EqualFold(azureCloudNGFWString(item["name"]), statusName) {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusOK)
					return
				}
				respondJSON(w, http.StatusOK, item)
				return
			}
		}
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "status", statusName))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCloudNGFWLocalRulestacksRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureCloudNGFWLocalRulestacksListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureCloudNGFWLocalRulestacksListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCloudNGFWValidName(name) {
		respondAzureInvalidRequest(w, path, "local rulestack name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCloudNGFWLocalRulestackGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureCloudNGFWLocalRulestackGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureCloudNGFWLocalRulestackCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureCloudNGFWLocalRulestackPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureCloudNGFWLocalRulestackDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "commit":
		if r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackCommit(w, path, subscriptionID, resourceGroup, name, false)
			return
		}
	case "revert":
		if r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackRevert(w, path, subscriptionID, resourceGroup, name, false)
			return
		}
	case "getchangelog":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackChangeLog(w, path, subscriptionID, resourceGroup, name, false)
			return
		}
	case "getsupportinfo":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackSupportInfo(w, path, subscriptionID, resourceGroup, name, false)
			return
		}
	case "listfirewalls":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackListFirewalls(w, subscriptionID, resourceGroup, name, false)
			return
		}
	case "listcountries":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWCountriesFixture()})
			return
		}
	case "listappids":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWAppIDsFixture()})
			return
		}
	case "listpredefinedurlcategories":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWPredefinedURLCategoriesFixture()})
			return
		}
	case "listsecurityservices":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWSecurityServicesFixture()})
			return
		}
	case "listadvancedsecurityobjects":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWAdvancedSecurityObjectsFixture()})
			return
		}
	default:
		if collection, ok := azureCloudNGFWLocalChildCollections[strings.ToLower(strings.TrimSpace(tail[2]))]; ok {
			s.handleAzureCloudNGFWLocalChildRoute(w, r, path, subscriptionID, resourceGroup, name, collection, tail[2:])
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCloudNGFWGlobalRulestacksRoute(w http.ResponseWriter, r *http.Request, path string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureCloudNGFWGlobalRulestacksList(w)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCloudNGFWValidName(name) {
		respondAzureInvalidRequest(w, path, "global rulestack name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCloudNGFWGlobalRulestackGet(w, path, name, false)
		case http.MethodHead:
			s.handleAzureCloudNGFWGlobalRulestackGet(w, path, name, true)
		case http.MethodPut:
			s.handleAzureCloudNGFWGlobalRulestackCreateOrUpdate(w, r, path, name)
		case http.MethodPatch:
			s.handleAzureCloudNGFWGlobalRulestackPatch(w, r, path, name)
		case http.MethodDelete:
			s.handleAzureCloudNGFWGlobalRulestackDelete(w, path, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "commit":
		if r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackCommit(w, path, "", "", name, true)
			return
		}
	case "revert":
		if r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackRevert(w, path, "", "", name, true)
			return
		}
	case "getchangelog":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackChangeLog(w, path, "", "", name, true)
			return
		}
	case "listfirewalls":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureCloudNGFWRulestackListFirewalls(w, "", "", name, true)
			return
		}
	case "listcountries":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWCountriesFixture()})
			return
		}
	case "listappids":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWAppIDsFixture()})
			return
		}
	case "listpredefinedurlcategories":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWPredefinedURLCategoriesFixture()})
			return
		}
	case "listsecurityservices":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWSecurityServicesFixture()})
			return
		}
	case "listadvancedsecurityobjects":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWAdvancedSecurityObjectsFixture()})
			return
		}
	default:
		if collection, ok := azureCloudNGFWGlobalChildCollections[strings.ToLower(strings.TrimSpace(tail[2]))]; ok {
			s.handleAzureCloudNGFWGlobalChildRoute(w, r, path, name, collection, tail[2:])
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCloudNGFWLocalChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, parentName, collection string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureCloudNGFWLocalChildList(w, path, subscriptionID, resourceGroup, parentName, collection)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "child resource name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCloudNGFWLocalChildGet(w, path, subscriptionID, resourceGroup, parentName, collection, name, false)
		case http.MethodHead:
			s.handleAzureCloudNGFWLocalChildGet(w, path, subscriptionID, resourceGroup, parentName, collection, name, true)
		case http.MethodPut:
			s.handleAzureCloudNGFWLocalChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, parentName, collection, name)
		case http.MethodDelete:
			s.handleAzureCloudNGFWLocalChildDelete(w, path, subscriptionID, resourceGroup, parentName, collection, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.EqualFold(collection, "localRules") {
		switch strings.ToLower(strings.TrimSpace(tail[2])) {
		case "getcounters":
			if r.Method == http.MethodGet || r.Method == http.MethodPost {
				s.handleAzureCloudNGFWLocalRuleCounters(w, path, subscriptionID, resourceGroup, parentName, name)
				return
			}
		case "refreshcounters":
			if r.Method == http.MethodPost {
				s.handleAzureCloudNGFWLocalRuleRefreshCounters(w, path, subscriptionID, resourceGroup, parentName, name)
				return
			}
		case "resetcounters":
			if r.Method == http.MethodPost {
				s.handleAzureCloudNGFWLocalRuleResetCounters(w, path, subscriptionID, resourceGroup, parentName, name)
				return
			}
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCloudNGFWGlobalChildRoute(w http.ResponseWriter, r *http.Request, path, parentName, collection string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureCloudNGFWGlobalChildList(w, path, parentName, collection)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "child resource name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureCloudNGFWGlobalChildGet(w, path, parentName, collection, name, false)
	case http.MethodHead:
		s.handleAzureCloudNGFWGlobalChildGet(w, path, parentName, collection, name, true)
	case http.MethodPut:
		s.handleAzureCloudNGFWGlobalChildCreateOrUpdate(w, r, path, parentName, collection, name)
	case http.MethodDelete:
		s.handleAzureCloudNGFWGlobalChildDelete(w, path, parentName, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCloudNGFWFirewallsListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, firewall := range s.azureCloudNGFWFirewalls {
		if strings.EqualFold(firewall.SubscriptionID, subscriptionID) {
			items = append(items, azureCloudNGFWFirewallFixture(firewall))
		}
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureCloudNGFWFirewallsListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, firewall := range s.azureCloudNGFWFirewalls {
		if strings.EqualFold(firewall.SubscriptionID, subscriptionID) && strings.EqualFold(firewall.ResourceGroup, resourceGroup) {
			items = append(items, azureCloudNGFWFirewallFixture(firewall))
		}
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureCloudNGFWFirewallGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	firewall, ok := s.azureCloudNGFWFirewalls[azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "firewall", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCloudNGFWFirewallFixture(firewall))
}

func (s *Server) handleAzureCloudNGFWFirewallCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureCloudNGFWString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()
	props := azureCloudNGFWNormalizeFirewallProperties(payload["properties"], now)
	tags := azureCloudNGFWStringMap(payload["tags"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)
	if existing, ok := s.azureCloudNGFWFirewalls[key]; ok {
		existing.Location = location
		existing.Tags = tags
		existing.Properties = props
		existing.UpdatedAt = now
		existing.ETag = azureCloudNGFWETag(existing.ID(), now)
		respondJSON(w, http.StatusOK, azureCloudNGFWFirewallFixture(existing))
		return
	}
	firewall := &azureCloudNGFWFirewall{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           name,
		Location:       location,
		Tags:           tags,
		Properties:     props,
		CreatedAt:      now,
		UpdatedAt:      now,
		ETag:           azureCloudNGFWETag(azureCloudNGFWFirewallID(subscriptionID, resourceGroup, name), now),
	}
	s.azureCloudNGFWFirewalls[key] = firewall
	respondJSON(w, http.StatusCreated, azureCloudNGFWFirewallFixture(firewall))
}

func (f *azureCloudNGFWFirewall) ID() string {
	return azureCloudNGFWFirewallID(f.SubscriptionID, f.ResourceGroup, f.Name)
}

func (s *Server) handleAzureCloudNGFWFirewallPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	firewall, ok := s.azureCloudNGFWFirewalls[azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "firewall", name))
		return
	}
	if tags := azureCloudNGFWStringMap(payload["tags"]); tags != nil {
		firewall.Tags = tags
	}
	if props := azureCloudNGFWObject(payload["properties"]); props != nil {
		firewall.Properties = azureCloudNGFWMergeAnyMap(firewall.Properties, props)
		firewall.Properties = azureCloudNGFWNormalizeFirewallProperties(firewall.Properties, time.Now().UTC())
	}
	firewall.UpdatedAt = time.Now().UTC()
	firewall.ETag = azureCloudNGFWETag(firewall.ID(), firewall.UpdatedAt)
	respondJSON(w, http.StatusOK, azureCloudNGFWFirewallFixture(firewall))
}

func (s *Server) handleAzureCloudNGFWFirewallDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)
	if _, ok := s.azureCloudNGFWFirewalls[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "firewall", name))
		return
	}
	delete(s.azureCloudNGFWFirewalls, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCloudNGFWFirewallGetGlobalRulestack(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	firewall, ok := s.azureCloudNGFWFirewalls[azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "firewall", name))
		return
	}
	globalName := strings.TrimSpace(azureCloudNGFWString(firewall.Properties["globalRulestackName"]))
	if globalName == "" {
		globalName = "global-default"
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":   azureCloudNGFWGlobalRulestackID(globalName),
		"name": globalName,
		"type": "PaloAltoNetworks.Cloudngfw/globalRulestacks",
	})
}

func (s *Server) handleAzureCloudNGFWFirewallGetLogProfile(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	firewall, ok := s.azureCloudNGFWFirewalls[azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "firewall", name))
		return
	}
	profile := azureCloudNGFWObject(firewall.Properties["logProfile"])
	if profile == nil {
		profile = map[string]any{
			"logType":    "Traffic",
			"storageURL": "https://logs.stackyard.local/" + name,
		}
	}
	respondJSON(w, http.StatusOK, profile)
}

func (s *Server) handleAzureCloudNGFWFirewallSaveLogProfile(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	firewall, ok := s.azureCloudNGFWFirewalls[azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "firewall", name))
		return
	}
	firewall.Properties["logProfile"] = azureCloudNGFWObject(payload)
	firewall.UpdatedAt = time.Now().UTC()
	firewall.ETag = azureCloudNGFWETag(firewall.ID(), firewall.UpdatedAt)
	respondJSON(w, http.StatusOK, azureCloudNGFWObject(payload))
}

func (s *Server) handleAzureCloudNGFWFirewallGetSupportInfo(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	firewall, ok := s.azureCloudNGFWFirewalls[azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "firewall", name))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"firewallId":   firewall.ID(),
		"serialNumber": "stackyard-" + name,
		"supportLevel": "Standard",
		"status":       azureCloudNGFWString(firewall.Properties["provisioningState"]),
	})
}

func (s *Server) azureCloudNGFWFirewallStatusesFixture(subscriptionID, resourceGroup, firewallName string) []map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	firewall, ok := s.azureCloudNGFWFirewalls[azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, firewallName)]
	if !ok {
		return []map[string]any{}
	}
	state := azureCloudNGFWString(firewall.Properties["provisioningState"])
	if state == "" {
		state = "Succeeded"
	}
	return []map[string]any{{
		"id":       firewall.ID() + "/statuses/default",
		"name":     "default",
		"type":     "PaloAltoNetworks.Cloudngfw/firewalls/statuses",
		"location": firewall.Location,
		"properties": map[string]any{
			"status":             state,
			"healthStatus":       "Healthy",
			"lastStatusChange":   firewall.UpdatedAt.UTC().Format(time.RFC3339),
			"connectedRulestack": azureCloudNGFWString(firewall.Properties["localRulestackName"]),
		},
	}}
}

func (s *Server) handleAzureCloudNGFWLocalRulestacksListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, stack := range s.azureCloudNGFWLocalRulestacks {
		if strings.EqualFold(stack.SubscriptionID, subscriptionID) {
			items = append(items, azureCloudNGFWLocalRulestackFixture(stack))
		}
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureCloudNGFWLocalRulestacksListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, stack := range s.azureCloudNGFWLocalRulestacks {
		if strings.EqualFold(stack.SubscriptionID, subscriptionID) && strings.EqualFold(stack.ResourceGroup, resourceGroup) {
			items = append(items, azureCloudNGFWLocalRulestackFixture(stack))
		}
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureCloudNGFWLocalRulestackGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stack, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCloudNGFWLocalRulestackFixture(stack))
}

func (s *Server) handleAzureCloudNGFWLocalRulestackCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureCloudNGFWString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()
	props := azureCloudNGFWNormalizeRulestackProperties(payload["properties"], now)
	tags := azureCloudNGFWStringMap(payload["tags"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)
	if existing, ok := s.azureCloudNGFWLocalRulestacks[key]; ok {
		existing.Location = location
		existing.Tags = tags
		existing.Properties = props
		existing.UpdatedAt = now
		existing.ETag = azureCloudNGFWETag(existing.ID(), now)
		respondJSON(w, http.StatusOK, azureCloudNGFWLocalRulestackFixture(existing))
		return
	}
	stack := &azureCloudNGFWLocalRulestack{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           name,
		Location:       location,
		Tags:           tags,
		Properties:     props,
		CreatedAt:      now,
		UpdatedAt:      now,
		ETag:           azureCloudNGFWETag(azureCloudNGFWLocalRulestackID(subscriptionID, resourceGroup, name), now),
	}
	s.azureCloudNGFWLocalRulestacks[key] = stack
	respondJSON(w, http.StatusCreated, azureCloudNGFWLocalRulestackFixture(stack))
}

func (l *azureCloudNGFWLocalRulestack) ID() string {
	return azureCloudNGFWLocalRulestackID(l.SubscriptionID, l.ResourceGroup, l.Name)
}

func (s *Server) handleAzureCloudNGFWLocalRulestackPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stack, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", name))
		return
	}
	if tags := azureCloudNGFWStringMap(payload["tags"]); tags != nil {
		stack.Tags = tags
	}
	if props := azureCloudNGFWObject(payload["properties"]); props != nil {
		stack.Properties = azureCloudNGFWMergeAnyMap(stack.Properties, props)
		stack.Properties = azureCloudNGFWNormalizeRulestackProperties(stack.Properties, time.Now().UTC())
	}
	stack.UpdatedAt = time.Now().UTC()
	stack.ETag = azureCloudNGFWETag(stack.ID(), stack.UpdatedAt)
	respondJSON(w, http.StatusOK, azureCloudNGFWLocalRulestackFixture(stack))
}

func (s *Server) handleAzureCloudNGFWLocalRulestackDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)
	if _, ok := s.azureCloudNGFWLocalRulestacks[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", name))
		return
	}
	delete(s.azureCloudNGFWLocalRulestacks, key)
	prefix := strings.ToLower(azureCloudNGFWLocalRulestackID(subscriptionID, resourceGroup, name) + "/")
	for key := range s.azureCloudNGFWLocalChildren {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			delete(s.azureCloudNGFWLocalChildren, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCloudNGFWGlobalRulestacksList(w http.ResponseWriter) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, stack := range s.azureCloudNGFWGlobalRulestacks {
		items = append(items, azureCloudNGFWGlobalRulestackFixture(stack))
	}
	if len(items) == 0 {
		now := time.Now().UTC()
		stack := &azureCloudNGFWGlobalRulestack{
			Name:       "global-default",
			Location:   "global",
			Properties: azureCloudNGFWNormalizeRulestackProperties(nil, now),
			CreatedAt:  now,
			UpdatedAt:  now,
			ETag:       azureCloudNGFWETag(azureCloudNGFWGlobalRulestackID("global-default"), now),
		}
		s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(stack.Name)] = stack
		items = append(items, azureCloudNGFWGlobalRulestackFixture(stack))
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureCloudNGFWGlobalRulestackGet(w http.ResponseWriter, path, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stack, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCloudNGFWGlobalRulestackFixture(stack))
}

func (s *Server) handleAzureCloudNGFWGlobalRulestackCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureCloudNGFWString(payload["location"]))
	if location == "" {
		location = "global"
	}
	now := time.Now().UTC()
	props := azureCloudNGFWNormalizeRulestackProperties(payload["properties"], now)
	tags := azureCloudNGFWStringMap(payload["tags"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWGlobalRulestackKey(name)
	if existing, ok := s.azureCloudNGFWGlobalRulestacks[key]; ok {
		existing.Location = location
		existing.Tags = tags
		existing.Properties = props
		existing.UpdatedAt = now
		existing.ETag = azureCloudNGFWETag(existing.ID(), now)
		respondJSON(w, http.StatusOK, azureCloudNGFWGlobalRulestackFixture(existing))
		return
	}
	stack := &azureCloudNGFWGlobalRulestack{
		Name:       name,
		Location:   location,
		Tags:       tags,
		Properties: props,
		CreatedAt:  now,
		UpdatedAt:  now,
		ETag:       azureCloudNGFWETag(azureCloudNGFWGlobalRulestackID(name), now),
	}
	s.azureCloudNGFWGlobalRulestacks[key] = stack
	respondJSON(w, http.StatusCreated, azureCloudNGFWGlobalRulestackFixture(stack))
}

func (g *azureCloudNGFWGlobalRulestack) ID() string {
	return azureCloudNGFWGlobalRulestackID(g.Name)
}

func (s *Server) handleAzureCloudNGFWGlobalRulestackPatch(w http.ResponseWriter, r *http.Request, path, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stack, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", name))
		return
	}
	if tags := azureCloudNGFWStringMap(payload["tags"]); tags != nil {
		stack.Tags = tags
	}
	if props := azureCloudNGFWObject(payload["properties"]); props != nil {
		stack.Properties = azureCloudNGFWMergeAnyMap(stack.Properties, props)
		stack.Properties = azureCloudNGFWNormalizeRulestackProperties(stack.Properties, time.Now().UTC())
	}
	stack.UpdatedAt = time.Now().UTC()
	stack.ETag = azureCloudNGFWETag(stack.ID(), stack.UpdatedAt)
	respondJSON(w, http.StatusOK, azureCloudNGFWGlobalRulestackFixture(stack))
}

func (s *Server) handleAzureCloudNGFWGlobalRulestackDelete(w http.ResponseWriter, path, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWGlobalRulestackKey(name)
	if _, ok := s.azureCloudNGFWGlobalRulestacks[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", name))
		return
	}
	delete(s.azureCloudNGFWGlobalRulestacks, key)
	prefix := strings.ToLower(azureCloudNGFWGlobalRulestackID(name) + "/")
	for key := range s.azureCloudNGFWGlobalChildren {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			delete(s.azureCloudNGFWGlobalChildren, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCloudNGFWLocalChildList(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentName, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, parentName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", parentName))
		return
	}
	items := make([]map[string]any, 0)
	for _, child := range s.azureCloudNGFWLocalChildren {
		if strings.EqualFold(child.SubscriptionID, subscriptionID) && strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.ParentName, parentName) && strings.EqualFold(child.Collection, collection) {
			items = append(items, azureCloudNGFWLocalChildFixture(child))
		}
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureCloudNGFWLocalChildGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentName, collection, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child, ok := s.azureCloudNGFWLocalChildren[azureCloudNGFWLocalChildKey(subscriptionID, resourceGroup, parentName, collection, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, collection, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCloudNGFWLocalChildFixture(child))
}

func (s *Server) handleAzureCloudNGFWLocalChildCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, parentName, collection, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, parentName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", parentName))
		return
	}
	now := time.Now().UTC()
	location := strings.TrimSpace(azureCloudNGFWString(payload["location"]))
	if location == "" {
		location = "eastus"
	}
	props := azureCloudNGFWNormalizeChildProperties(collection, name, payload["properties"], now)
	tags := azureCloudNGFWStringMap(payload["tags"])
	key := azureCloudNGFWLocalChildKey(subscriptionID, resourceGroup, parentName, collection, name)
	if existing, ok := s.azureCloudNGFWLocalChildren[key]; ok {
		existing.Location = location
		existing.Tags = tags
		existing.Properties = props
		existing.UpdatedAt = now
		existing.ETag = azureCloudNGFWETag(existing.ID(), now)
		respondJSON(w, http.StatusOK, azureCloudNGFWLocalChildFixture(existing))
		return
	}
	child := &azureCloudNGFWChildResource{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ParentName:     parentName,
		Collection:     collection,
		Name:           name,
		Location:       location,
		Tags:           tags,
		Properties:     props,
		CreatedAt:      now,
		UpdatedAt:      now,
		ETag:           azureCloudNGFWETag(azureCloudNGFWLocalChildID(subscriptionID, resourceGroup, parentName, collection, name), now),
	}
	s.azureCloudNGFWLocalChildren[key] = child
	respondJSON(w, http.StatusCreated, azureCloudNGFWLocalChildFixture(child))
}

func (c *azureCloudNGFWChildResource) ID() string {
	return azureCloudNGFWLocalChildID(c.SubscriptionID, c.ResourceGroup, c.ParentName, c.Collection, c.Name)
}

func (s *Server) handleAzureCloudNGFWLocalChildDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentName, collection, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWLocalChildKey(subscriptionID, resourceGroup, parentName, collection, name)
	if _, ok := s.azureCloudNGFWLocalChildren[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, collection, name))
		return
	}
	delete(s.azureCloudNGFWLocalChildren, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCloudNGFWGlobalChildList(w http.ResponseWriter, path, parentName, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(parentName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", parentName))
		return
	}
	items := make([]map[string]any, 0)
	for _, child := range s.azureCloudNGFWGlobalChildren {
		if strings.EqualFold(child.ParentName, parentName) && strings.EqualFold(child.Collection, collection) {
			items = append(items, azureCloudNGFWGlobalChildFixture(child))
		}
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureCloudNGFWGlobalChildGet(w http.ResponseWriter, path, parentName, collection, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child, ok := s.azureCloudNGFWGlobalChildren[azureCloudNGFWGlobalChildKey(parentName, collection, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, collection, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCloudNGFWGlobalChildFixture(child))
}

func (s *Server) handleAzureCloudNGFWGlobalChildCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, parentName, collection, name string) {
	_, payload, ok := azureCloudNGFWReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(parentName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", parentName))
		return
	}
	now := time.Now().UTC()
	location := strings.TrimSpace(azureCloudNGFWString(payload["location"]))
	if location == "" {
		location = "global"
	}
	props := azureCloudNGFWNormalizeChildProperties(collection, name, payload["properties"], now)
	tags := azureCloudNGFWStringMap(payload["tags"])
	key := azureCloudNGFWGlobalChildKey(parentName, collection, name)
	if existing, ok := s.azureCloudNGFWGlobalChildren[key]; ok {
		existing.Location = location
		existing.Tags = tags
		existing.Properties = props
		existing.UpdatedAt = now
		existing.ETag = azureCloudNGFWETag(existing.ID(), now)
		respondJSON(w, http.StatusOK, azureCloudNGFWGlobalChildFixture(existing))
		return
	}
	child := &azureCloudNGFWGlobalChildResource{
		ParentName: parentName,
		Collection: collection,
		Name:       name,
		Location:   location,
		Tags:       tags,
		Properties: props,
		CreatedAt:  now,
		UpdatedAt:  now,
		ETag:       azureCloudNGFWETag(azureCloudNGFWGlobalChildID(parentName, collection, name), now),
	}
	s.azureCloudNGFWGlobalChildren[key] = child
	respondJSON(w, http.StatusCreated, azureCloudNGFWGlobalChildFixture(child))
}

func (g *azureCloudNGFWGlobalChildResource) ID() string {
	return azureCloudNGFWGlobalChildID(g.ParentName, g.Collection, g.Name)
}

func (s *Server) handleAzureCloudNGFWGlobalChildDelete(w http.ResponseWriter, path, parentName, collection, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureCloudNGFWGlobalChildKey(parentName, collection, name)
	if _, ok := s.azureCloudNGFWGlobalChildren[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, collection, name))
		return
	}
	delete(s.azureCloudNGFWGlobalChildren, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCloudNGFWLocalRuleCounters(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentName, priority string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child, ok := s.azureCloudNGFWLocalChildren[azureCloudNGFWLocalChildKey(subscriptionID, resourceGroup, parentName, "localRules", priority)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rule", priority))
		return
	}
	respondJSON(w, http.StatusOK, azureCloudNGFWRuleCounterFixture(child))
}

func (s *Server) handleAzureCloudNGFWLocalRuleRefreshCounters(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentName, priority string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child, ok := s.azureCloudNGFWLocalChildren[azureCloudNGFWLocalChildKey(subscriptionID, resourceGroup, parentName, "localRules", priority)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rule", priority))
		return
	}
	counters := azureCloudNGFWObject(child.Properties["counters"])
	if counters == nil {
		counters = map[string]any{"hitCount": 0, "byteCount": 0}
	}
	counters["hitCount"] = azureCloudNGFWIntValue(counters["hitCount"]) + 5
	counters["byteCount"] = azureCloudNGFWIntValue(counters["byteCount"]) + 1024
	child.Properties["counters"] = counters
	child.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureCloudNGFWRuleCounterFixture(child))
}

func (s *Server) handleAzureCloudNGFWLocalRuleResetCounters(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentName, priority string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child, ok := s.azureCloudNGFWLocalChildren[azureCloudNGFWLocalChildKey(subscriptionID, resourceGroup, parentName, "localRules", priority)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rule", priority))
		return
	}
	child.Properties["counters"] = map[string]any{"hitCount": 0, "byteCount": 0}
	child.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureCloudNGFWRuleCounterFixture(child))
}

func (s *Server) handleAzureCloudNGFWRulestackCommit(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, global bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if global {
		stack, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(name)]
		if !ok {
			respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", name))
			return
		}
		revision := azureCloudNGFWIntValue(stack.Properties["revision"]) + 1
		stack.Properties["revision"] = revision
		stack.Properties["lastCommittedAt"] = time.Now().UTC().Format(time.RFC3339)
		stack.Properties["changeLog"] = azureCloudNGFWAppendChangeLog(stack.Properties["changeLog"], "commit", revision)
		stack.UpdatedAt = time.Now().UTC()
		respondJSON(w, http.StatusOK, azureCloudNGFWGlobalRulestackFixture(stack))
		return
	}
	stack, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", name))
		return
	}
	revision := azureCloudNGFWIntValue(stack.Properties["revision"]) + 1
	stack.Properties["revision"] = revision
	stack.Properties["lastCommittedAt"] = time.Now().UTC().Format(time.RFC3339)
	stack.Properties["changeLog"] = azureCloudNGFWAppendChangeLog(stack.Properties["changeLog"], "commit", revision)
	stack.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureCloudNGFWLocalRulestackFixture(stack))
}

func (s *Server) handleAzureCloudNGFWRulestackRevert(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, global bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if global {
		stack, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(name)]
		if !ok {
			respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", name))
			return
		}
		revision := azureCloudNGFWIntValue(stack.Properties["revision"])
		if revision > 0 {
			revision--
		}
		stack.Properties["revision"] = revision
		stack.Properties["changeLog"] = azureCloudNGFWAppendChangeLog(stack.Properties["changeLog"], "revert", revision)
		stack.UpdatedAt = time.Now().UTC()
		respondJSON(w, http.StatusOK, azureCloudNGFWGlobalRulestackFixture(stack))
		return
	}
	stack, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", name))
		return
	}
	revision := azureCloudNGFWIntValue(stack.Properties["revision"])
	if revision > 0 {
		revision--
	}
	stack.Properties["revision"] = revision
	stack.Properties["changeLog"] = azureCloudNGFWAppendChangeLog(stack.Properties["changeLog"], "revert", revision)
	stack.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureCloudNGFWLocalRulestackFixture(stack))
}

func (s *Server) handleAzureCloudNGFWRulestackChangeLog(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, global bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if global {
		stack, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(name)]
		if !ok {
			respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", name))
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWSlice(stack.Properties["changeLog"])})
		return
	}
	stack, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", name))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": azureCloudNGFWSlice(stack.Properties["changeLog"])})
}

func (s *Server) handleAzureCloudNGFWRulestackSupportInfo(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, global bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if global {
		if _, ok := s.azureCloudNGFWGlobalRulestacks[azureCloudNGFWGlobalRulestackKey(name)]; !ok {
			respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "global rulestack", name))
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"name": name, "supportLevel": "Premium", "scope": "global"})
		return
	}
	if _, ok := s.azureCloudNGFWLocalRulestacks[azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name)]; !ok {
		respondJSON(w, http.StatusNotFound, azureCloudNGFWNotFound(path, "local rulestack", name))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"name": name, "supportLevel": "Standard", "scope": "local"})
}

func (s *Server) handleAzureCloudNGFWRulestackListFirewalls(w http.ResponseWriter, subscriptionID, resourceGroup, name string, global bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, firewall := range s.azureCloudNGFWFirewalls {
		if global {
			if strings.EqualFold(azureCloudNGFWString(firewall.Properties["globalRulestackName"]), name) {
				items = append(items, azureCloudNGFWFirewallFixture(firewall))
			}
			continue
		}
		if strings.EqualFold(firewall.SubscriptionID, subscriptionID) && strings.EqualFold(firewall.ResourceGroup, resourceGroup) &&
			strings.EqualFold(azureCloudNGFWString(firewall.Properties["localRulestackName"]), name) {
			items = append(items, azureCloudNGFWFirewallFixture(firewall))
		}
	}
	azureCloudNGFWSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func azureCloudNGFWOperationsFixture() []map[string]any {
	return []map[string]any{
		{"name": "PaloAltoNetworks.Cloudngfw/operations/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Operations", "operation": "List Operations", "description": "List Cloud NGFW operations"}},
		{"name": "PaloAltoNetworks.Cloudngfw/firewalls/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Firewalls", "operation": "Read Firewalls", "description": "Get or list firewalls"}},
		{"name": "PaloAltoNetworks.Cloudngfw/firewalls/write", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Firewalls", "operation": "Write Firewalls", "description": "Create or update firewalls"}},
		{"name": "PaloAltoNetworks.Cloudngfw/firewalls/delete", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Firewalls", "operation": "Delete Firewalls", "description": "Delete firewalls"}},
		{"name": "PaloAltoNetworks.Cloudngfw/firewalls/statuses/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Firewall Statuses", "operation": "Read Firewall Statuses", "description": "Get firewall statuses"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Local Rulestacks", "operation": "Read Local Rulestacks", "description": "Get or list local rulestacks"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/write", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Local Rulestacks", "operation": "Write Local Rulestacks", "description": "Create or update local rulestacks"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/delete", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Local Rulestacks", "operation": "Delete Local Rulestacks", "description": "Delete local rulestacks"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/localRules/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Local Rules", "operation": "Read Local Rules", "description": "Get or list local rules"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/localRules/write", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Local Rules", "operation": "Write Local Rules", "description": "Create or update local rules"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/prefixlists/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Prefix Lists", "operation": "Read Prefix Lists", "description": "Get or list prefix lists"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/fqdnlists/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "FQDN Lists", "operation": "Read FQDN Lists", "description": "Get or list FQDN lists"}},
		{"name": "PaloAltoNetworks.Cloudngfw/localRulestacks/certificates/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Certificates", "operation": "Read Certificates", "description": "Get or list certificates"}},
		{"name": "PaloAltoNetworks.Cloudngfw/globalRulestacks/read", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Global Rulestacks", "operation": "Read Global Rulestacks", "description": "Get or list global rulestacks"}},
		{"name": "PaloAltoNetworks.Cloudngfw/globalRulestacks/write", "display": map[string]any{"provider": "Palo Alto Networks Cloud NGFW", "resource": "Global Rulestacks", "operation": "Write Global Rulestacks", "description": "Create or update global rulestacks"}},
	}
}

func azureCloudNGFWFirewallFixture(firewall *azureCloudNGFWFirewall) map[string]any {
	return map[string]any{
		"id":         firewall.ID(),
		"name":       firewall.Name,
		"type":       "PaloAltoNetworks.Cloudngfw/firewalls",
		"location":   firewall.Location,
		"tags":       azureCloudNGFWAnyMapFromStringMap(firewall.Tags),
		"properties": azureCloudNGFWCloneAnyMap(firewall.Properties),
		"systemData": azureCloudNGFWSystemData(firewall.CreatedAt, firewall.UpdatedAt),
		"etag":       firewall.ETag,
	}
}

func azureCloudNGFWLocalRulestackFixture(stack *azureCloudNGFWLocalRulestack) map[string]any {
	return map[string]any{
		"id":         stack.ID(),
		"name":       stack.Name,
		"type":       "PaloAltoNetworks.Cloudngfw/localRulestacks",
		"location":   stack.Location,
		"tags":       azureCloudNGFWAnyMapFromStringMap(stack.Tags),
		"properties": azureCloudNGFWCloneAnyMap(stack.Properties),
		"systemData": azureCloudNGFWSystemData(stack.CreatedAt, stack.UpdatedAt),
		"etag":       stack.ETag,
	}
}

func azureCloudNGFWGlobalRulestackFixture(stack *azureCloudNGFWGlobalRulestack) map[string]any {
	return map[string]any{
		"id":         stack.ID(),
		"name":       stack.Name,
		"type":       "PaloAltoNetworks.Cloudngfw/globalRulestacks",
		"location":   stack.Location,
		"tags":       azureCloudNGFWAnyMapFromStringMap(stack.Tags),
		"properties": azureCloudNGFWCloneAnyMap(stack.Properties),
		"systemData": azureCloudNGFWSystemData(stack.CreatedAt, stack.UpdatedAt),
		"etag":       stack.ETag,
	}
}

func azureCloudNGFWLocalChildFixture(child *azureCloudNGFWChildResource) map[string]any {
	return map[string]any{
		"id":         child.ID(),
		"name":       child.Name,
		"type":       azureCloudNGFWLocalChildType(child.Collection),
		"location":   child.Location,
		"tags":       azureCloudNGFWAnyMapFromStringMap(child.Tags),
		"properties": azureCloudNGFWCloneAnyMap(child.Properties),
		"systemData": azureCloudNGFWSystemData(child.CreatedAt, child.UpdatedAt),
		"etag":       child.ETag,
	}
}

func azureCloudNGFWGlobalChildFixture(child *azureCloudNGFWGlobalChildResource) map[string]any {
	return map[string]any{
		"id":         child.ID(),
		"name":       child.Name,
		"type":       azureCloudNGFWGlobalChildType(child.Collection),
		"location":   child.Location,
		"tags":       azureCloudNGFWAnyMapFromStringMap(child.Tags),
		"properties": azureCloudNGFWCloneAnyMap(child.Properties),
		"systemData": azureCloudNGFWSystemData(child.CreatedAt, child.UpdatedAt),
		"etag":       child.ETag,
	}
}

func azureCloudNGFWRuleCounterFixture(child *azureCloudNGFWChildResource) map[string]any {
	counters := azureCloudNGFWObject(child.Properties["counters"])
	if counters == nil {
		counters = map[string]any{"hitCount": 0, "byteCount": 0}
	}
	return map[string]any{
		"id":        child.ID() + "/getCounters",
		"name":      child.Name,
		"priority":  child.Name,
		"hitCount":  azureCloudNGFWIntValue(counters["hitCount"]),
		"byteCount": azureCloudNGFWIntValue(counters["byteCount"]),
	}
}

func azureCloudNGFWCountriesFixture() []map[string]any {
	return []map[string]any{{"code": "US", "name": "United States"}, {"code": "GB", "name": "United Kingdom"}}
}

func azureCloudNGFWAppIDsFixture() []map[string]any {
	return []map[string]any{{"id": "ssl", "category": "infrastructure"}, {"id": "web-browsing", "category": "internet"}}
}

func azureCloudNGFWPredefinedURLCategoriesFixture() []map[string]any {
	return []map[string]any{{"name": "financial-services"}, {"name": "high-risk"}}
}

func azureCloudNGFWSecurityServicesFixture() []map[string]any {
	return []map[string]any{{"name": "anti-spyware"}, {"name": "vulnerability-protection"}}
}

func azureCloudNGFWAdvancedSecurityObjectsFixture() []map[string]any {
	return []map[string]any{{"name": "dns-security"}, {"name": "url-filtering"}}
}

func azureCloudNGFWNormalizeFirewallProperties(raw any, now time.Time) map[string]any {
	props := azureCloudNGFWObject(raw)
	if props == nil {
		props = map[string]any{}
	}
	if strings.TrimSpace(azureCloudNGFWString(props["provisioningState"])) == "" {
		props["provisioningState"] = "Succeeded"
	}
	if strings.TrimSpace(azureCloudNGFWString(props["marketplaceDetails"])) == "" {
		props["marketplaceDetails"] = "byol"
	}
	if props["logProfile"] == nil {
		props["logProfile"] = map[string]any{
			"logType":    "Traffic",
			"storageURL": "https://logs.stackyard.local/default",
		}
	}
	props["lastModifiedAt"] = now.Format(time.RFC3339)
	return props
}

func azureCloudNGFWNormalizeRulestackProperties(raw any, now time.Time) map[string]any {
	props := azureCloudNGFWObject(raw)
	if props == nil {
		props = map[string]any{}
	}
	if strings.TrimSpace(azureCloudNGFWString(props["provisioningState"])) == "" {
		props["provisioningState"] = "Succeeded"
	}
	if props["revision"] == nil {
		props["revision"] = 1
	}
	if props["changeLog"] == nil {
		props["changeLog"] = []any{
			map[string]any{
				"action":    "create",
				"revision":  azureCloudNGFWIntValue(props["revision"]),
				"timestamp": now.Format(time.RFC3339),
			},
		}
	}
	props["lastModifiedAt"] = now.Format(time.RFC3339)
	return props
}

func azureCloudNGFWNormalizeChildProperties(collection, name string, raw any, now time.Time) map[string]any {
	props := azureCloudNGFWObject(raw)
	if props == nil {
		props = map[string]any{}
	}
	switch collection {
	case "prefixlists":
		if props["prefixes"] == nil {
			props["prefixes"] = []any{"10.0.0.0/24"}
		}
	case "fqdnlists":
		if props["fqdnList"] == nil {
			props["fqdnList"] = []any{"example.com"}
		}
	case "certificates":
		if props["certificateSelfSigned"] == nil {
			props["certificateSelfSigned"] = map[string]any{"commonName": name}
		}
	case "localRules":
		if props["actionType"] == nil {
			props["actionType"] = "Allow"
		}
		if props["counters"] == nil {
			props["counters"] = map[string]any{"hitCount": 0, "byteCount": 0}
		}
	}
	if strings.TrimSpace(azureCloudNGFWString(props["provisioningState"])) == "" {
		props["provisioningState"] = "Succeeded"
	}
	props["description"] = azureCloudNGFWDefaultString(props["description"], collection+" "+name)
	props["lastModifiedAt"] = now.Format(time.RFC3339)
	return props
}

func azureCloudNGFWNotFound(path, kind, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "NotFound",
			"message": fmt.Sprintf("%s %q was not found for %s", kind, name, path),
		},
	}
}

func azureCloudNGFWReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		respondAzureInvalidRequest(w, path, "request body is required")
		return body, nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return body, nil, false
	}
	return body, payload, true
}

func azureCloudNGFWString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func azureCloudNGFWDefaultString(value any, fallback string) string {
	if s := strings.TrimSpace(azureCloudNGFWString(value)); s != "" {
		return s
	}
	return fallback
}

func azureCloudNGFWObject(value any) map[string]any {
	if m, ok := value.(map[string]any); ok && m != nil {
		return m
	}
	return nil
}

func azureCloudNGFWSlice(value any) []any {
	if values, ok := value.([]any); ok && values != nil {
		return values
	}
	return []any{}
}

func azureCloudNGFWStringMap(value any) map[string]string {
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return nil
	}
	out := make(map[string]string, len(object))
	for key, raw := range object {
		if str, ok := raw.(string); ok {
			out[key] = str
		}
	}
	return out
}

func azureCloudNGFWAnyMapFromStringMap(in map[string]string) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func azureCloudNGFWCloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureCloudNGFWCloneAnyMap(typed)
		case []any:
			cloned := make([]any, len(typed))
			for i, item := range typed {
				if object, ok := item.(map[string]any); ok {
					cloned[i] = azureCloudNGFWCloneAnyMap(object)
				} else {
					cloned[i] = item
				}
			}
			out[key] = cloned
		default:
			out[key] = value
		}
	}
	return out
}

func azureCloudNGFWMergeAnyMap(base, patch map[string]any) map[string]any {
	out := azureCloudNGFWCloneAnyMap(base)
	if out == nil {
		out = map[string]any{}
	}
	for key, value := range patch {
		if existing, ok := out[key].(map[string]any); ok {
			if replacement, ok := value.(map[string]any); ok {
				out[key] = azureCloudNGFWMergeAnyMap(existing, replacement)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func azureCloudNGFWAppendChangeLog(value any, action string, revision int) []any {
	items := azureCloudNGFWSlice(value)
	out := make([]any, 0, len(items)+1)
	out = append(out, items...)
	out = append(out, map[string]any{
		"action":    action,
		"revision":  revision,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	return out
}

func azureCloudNGFWIntValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func azureCloudNGFWSortByID(items []map[string]any) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureCloudNGFWString(items[i]["id"])) < strings.ToLower(azureCloudNGFWString(items[j]["id"]))
	})
}

func azureCloudNGFWValidName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 128 {
		return false
	}
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") || strings.HasSuffix(name, "-") || strings.HasSuffix(name, "_") {
		return false
	}
	if strings.Contains(name, "--") || strings.Contains(name, "__") {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func azureCloudNGFWFirewallKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(azureCloudNGFWFirewallID(subscriptionID, resourceGroup, name))
}

func azureCloudNGFWLocalRulestackKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(azureCloudNGFWLocalRulestackID(subscriptionID, resourceGroup, name))
}

func azureCloudNGFWGlobalRulestackKey(name string) string {
	return strings.ToLower(azureCloudNGFWGlobalRulestackID(name))
}

func azureCloudNGFWLocalChildKey(subscriptionID, resourceGroup, parentName, collection, name string) string {
	return strings.ToLower(azureCloudNGFWLocalChildID(subscriptionID, resourceGroup, parentName, collection, name))
}

func azureCloudNGFWGlobalChildKey(parentName, collection, name string) string {
	return strings.ToLower(azureCloudNGFWGlobalChildID(parentName, collection, name))
}

func azureCloudNGFWFirewallID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/PaloAltoNetworks.Cloudngfw/firewalls/%s", subscriptionID, resourceGroup, name)
}

func azureCloudNGFWLocalRulestackID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/PaloAltoNetworks.Cloudngfw/localRulestacks/%s", subscriptionID, resourceGroup, name)
}

func azureCloudNGFWGlobalRulestackID(name string) string {
	return fmt.Sprintf("/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks/%s", name)
}

func azureCloudNGFWLocalChildID(subscriptionID, resourceGroup, parentName, collection, name string) string {
	return azureCloudNGFWLocalRulestackID(subscriptionID, resourceGroup, parentName) + "/" + collection + "/" + name
}

func azureCloudNGFWGlobalChildID(parentName, collection, name string) string {
	return azureCloudNGFWGlobalRulestackID(parentName) + "/" + collection + "/" + name
}

func azureCloudNGFWLocalChildType(collection string) string {
	return "PaloAltoNetworks.Cloudngfw/localRulestacks/" + collection
}

func azureCloudNGFWGlobalChildType(collection string) string {
	return "PaloAltoNetworks.Cloudngfw/globalRulestacks/" + collection
}

func azureCloudNGFWETag(seed string, now time.Time) string {
	sum := sha1.Sum([]byte(seed + now.UTC().Format(time.RFC3339Nano)))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureCloudNGFWSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}
