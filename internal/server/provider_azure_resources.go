package server

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Microsoft.Resources: subscriptions, resource groups, and provider registration.
// ponytail: deployments and the tags API are not emulated yet; deployments need
// an ARM template evaluator and are tracked separately.

const (
	azureDefaultSubscriptionID = "00000000-0000-0000-0000-000000000000"
	azureDefaultTenantID       = "00000000-0000-0000-0000-000000000000"
)

var azureNamespacePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)+$`)

type azureResourceGroup struct {
	SubscriptionID string
	Name           string
	Location       string
	ManagedBy      string
	Tags           map[string]any
	CreatedAt      time.Time
}

func (g *azureResourceGroup) id() string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", g.SubscriptionID, g.Name)
}

func azureResourceGroupKey(sub, name string) string {
	return strings.ToLower(sub + "/" + name)
}

func (s *Server) handleAzureResourcesRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path != "/azure/subscriptions" && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	n := len(seg)
	isRG := n >= 3 && strings.EqualFold(seg[2], "resourceGroups")
	isProviders := n >= 3 && strings.EqualFold(seg[2], "providers")

	var handle func()
	switch {
	case n == 1 && r.Method == http.MethodGet:
		handle = func() {
			respondJSON(w, http.StatusOK, map[string]any{"value": []any{azureSubscriptionFixture(azureDefaultSubscriptionID)}})
		}
	case n == 2 && r.Method == http.MethodGet:
		handle = func() { respondJSON(w, http.StatusOK, azureSubscriptionFixture(seg[1])) }
	case isRG && n == 3:
		handle = func() { s.listAzureResourceGroups(w, r, seg[1]) }
	case isRG && n == 4:
		handle = func() { s.handleAzureResourceGroup(w, r, seg[1], seg[3]) }
	case isRG && n == 5 && strings.EqualFold(seg[4], "resources") && r.Method == http.MethodGet:
		handle = func() { s.listAzureResourceGroupResources(w, seg[1], seg[3]) }
	case isProviders && n == 3 && r.Method == http.MethodGet:
		handle = func() { s.listAzureProviders(w, seg[1]) }
	case isProviders && n == 4 && r.Method == http.MethodGet:
		handle = func() { respondAzureProvider(w, seg[1], seg[3]) }
	case isProviders && n == 5 && strings.EqualFold(seg[4], "register") && r.Method == http.MethodPost:
		handle = func() { respondAzureProvider(w, seg[1], seg[3]) }
	default:
		// Anything else under a subscription belongs to a service router.
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	handle()
	return true
}

func (s *Server) handleAzureResourceGroup(w http.ResponseWriter, r *http.Request, sub, name string) {
	key := azureResourceGroupKey(sub, name)
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", sub, name)
	switch r.Method {
	case http.MethodPut:
		s.putAzureResourceGroup(w, r, sub, name)
	case http.MethodPatch:
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		s.providerStorageMu.Lock()
		existing := s.azureResourceGroups[key]
		if existing == nil {
			s.providerStorageMu.Unlock()
			azureARMNotFound(w, "ResourceGroupNotFound", id, "")
			return
		}
		updated := *existing
		if tags, present := payload["tags"]; present {
			updated.Tags = azureARMObject(tags)
		}
		if managedBy, present := payload["managedBy"]; present {
			updated.ManagedBy = azureARMString(managedBy)
		}
		s.azureResourceGroups[key] = &updated
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, azureResourceGroupFixture(&updated))
	case http.MethodGet, http.MethodHead:
		s.providerStorageMu.Lock()
		group := s.azureResourceGroups[key]
		s.providerStorageMu.Unlock()
		switch {
		case group != nil && r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNoContent)
		case group != nil:
			respondJSON(w, http.StatusOK, azureResourceGroupFixture(group))
		case r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		default:
			azureARMNotFound(w, "ResourceGroupNotFound", id, "")
		}
	case http.MethodDelete:
		s.providerStorageMu.Lock()
		group := s.azureResourceGroups[key]
		if group != nil {
			delete(s.azureResourceGroups, key)
			prefix := azureARMKey(group.id() + "/")
			for k := range s.azureARMResources {
				if strings.HasPrefix(k, prefix) {
					delete(s.azureARMResources, k)
				}
			}
		}
		s.providerStorageMu.Unlock()
		if group == nil {
			azureARMNotFound(w, "ResourceGroupNotFound", id, "")
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) putAzureResourceGroup(w http.ResponseWriter, r *http.Request, sub, name string) {
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureARMString(payload["location"]))
	if location == "" {
		azureARMError(w, http.StatusBadRequest, "LocationRequired", "The location property is required for this definition.")
		return
	}
	key := azureResourceGroupKey(sub, name)
	s.providerStorageMu.Lock()
	existing := s.azureResourceGroups[key]
	if existing != nil && !strings.EqualFold(existing.Location, location) {
		s.providerStorageMu.Unlock()
		azureARMError(w, http.StatusConflict, "InvalidResourceGroupLocation",
			fmt.Sprintf("Invalid resource group location '%s'. The Resource group already exists in location '%s'.", location, existing.Location))
		return
	}
	group := &azureResourceGroup{
		SubscriptionID: sub,
		Name:           name,
		Location:       location,
		ManagedBy:      azureARMString(payload["managedBy"]),
		Tags:           azureARMObject(payload["tags"]),
		CreatedAt:      time.Now().UTC(),
	}
	if existing != nil {
		group.Name = existing.Name // keep the casing it was created with
		group.CreatedAt = existing.CreatedAt
	}
	s.azureResourceGroups[key] = group
	s.providerStorageMu.Unlock()

	status := http.StatusCreated
	if existing != nil {
		status = http.StatusOK
	}
	respondJSON(w, status, azureResourceGroupFixture(group))
}

func (s *Server) listAzureResourceGroups(w http.ResponseWriter, r *http.Request, sub string) {
	if r.Method != http.MethodGet {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	s.providerStorageMu.Lock()
	groups := make([]*azureResourceGroup, 0)
	for _, g := range s.azureResourceGroups {
		if strings.EqualFold(g.SubscriptionID, sub) {
			groups = append(groups, g)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(groups, func(i, j int) bool { return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name) })
	items := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		items = append(items, azureResourceGroupFixture(g))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) listAzureResourceGroupResources(w http.ResponseWriter, sub, rg string) {
	s.providerStorageMu.Lock()
	if s.azureResourceGroups[azureResourceGroupKey(sub, rg)] == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ResourceGroupNotFound", "/subscriptions/"+sub+"/resourceGroups/"+rg, "")
		return
	}
	records := s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		// Like ARM, nested child and extension resources are not listed at group scope.
		return strings.EqualFold(rec.SubscriptionID, sub) && strings.EqualFold(rec.ResourceGroup, rg) &&
			strings.Count(rec.Type, "/") == 1 && !azureARMResourceTypes[strings.ToLower(rec.Type)].Extension
	})
	s.providerStorageMu.Unlock()
	respondAzureARMList(w, records)
}

func (s *Server) listAzureProviders(w http.ResponseWriter, sub string) {
	namespaces := map[string]bool{"Microsoft.Resources": true}
	for _, t := range azureARMResourceTypes {
		namespaces[t.Namespace] = true
	}
	names := make([]string, 0, len(namespaces))
	for ns := range namespaces {
		names = append(names, ns)
	}
	sort.Strings(names)
	items := make([]map[string]any, 0, len(names))
	for _, ns := range names {
		items = append(items, azureProviderFixture(sub, ns))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

// respondAzureProvider reports every well-formed namespace as Registered so
// clients that register providers on startup (e.g. Terraform azurerm) succeed.
func respondAzureProvider(w http.ResponseWriter, sub, namespace string) {
	if !azureNamespacePattern.MatchString(namespace) {
		azureARMError(w, http.StatusNotFound, "InvalidResourceNamespace", "The resource namespace '"+namespace+"' is invalid.")
		return
	}
	respondJSON(w, http.StatusOK, azureProviderFixture(sub, namespace))
}

func azureProviderFixture(sub, namespace string) map[string]any {
	resourceTypes := make([]map[string]any, 0)
	for _, t := range azureARMResourceTypes {
		if strings.EqualFold(t.Namespace, namespace) {
			resourceTypes = append(resourceTypes, map[string]any{"resourceType": t.Type, "locations": []string{}})
		}
	}
	sort.Slice(resourceTypes, func(i, j int) bool {
		return azureARMString(resourceTypes[i]["resourceType"]) < azureARMString(resourceTypes[j]["resourceType"])
	})
	return map[string]any{
		"id":                 "/subscriptions/" + sub + "/providers/" + namespace,
		"namespace":          namespace,
		"registrationState":  "Registered",
		"registrationPolicy": "RegistrationRequired",
		"resourceTypes":      resourceTypes,
	}
}

func azureSubscriptionFixture(sub string) map[string]any {
	return map[string]any{
		"id":                   "/subscriptions/" + sub,
		"subscriptionId":       sub,
		"tenantId":             azureDefaultTenantID,
		"displayName":          "Stackyard Local",
		"state":                "Enabled",
		"authorizationSource":  "RoleBased",
		"subscriptionPolicies": map[string]any{"locationPlacementId": "Public_2014-09-01", "quotaId": "PayAsYouGo_2014-09-01", "spendingLimit": "Off"},
	}
}

func azureResourceGroupFixture(g *azureResourceGroup) map[string]any {
	out := map[string]any{
		"id":         g.id(),
		"name":       g.Name,
		"type":       "Microsoft.Resources/resourceGroups",
		"location":   g.Location,
		"tags":       azureARMMerge(g.Tags, nil),
		"properties": map[string]any{"provisioningState": "Succeeded"},
	}
	if g.ManagedBy != "" {
		out["managedBy"] = g.ManagedBy
	}
	return out
}
