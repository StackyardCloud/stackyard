package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const azureCommunicationAPIVersion = "2026-03-18"

var azureCommunicationServiceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{1,61}[A-Za-z0-9]$`)
var azureCommunicationChildNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)
var azureCommunicationGUIDPattern = regexp.MustCompile(`^[0-9a-fA-F-]{8,64}$`)

type azureCommunicationServiceResource struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Identity       map[string]any
	Tags           map[string]string
	Properties     map[string]any
	PrimaryKey     string
	SecondaryKey   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureCommunicationEmailServiceResource struct {
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

type azureCommunicationDomainResource struct {
	SubscriptionID     string
	ResourceGroup      string
	EmailServiceName   string
	Name               string
	Location           string
	Tags               map[string]string
	Properties         map[string]any
	VerificationStatus map[string]string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ETag               string
}

type azureCommunicationNestedResource struct {
	SubscriptionID      string
	ResourceGroup       string
	CommunicationName   string
	EmailServiceName    string
	DomainName          string
	SuppressionListName string
	Collection          string
	Name                string
	Location            string
	Tags                map[string]string
	Properties          map[string]any
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ETag                string
}

func (s *Server) handleAzureCommunicationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsCommunicationPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureCommunicationAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Communication") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureCommunicationOperationsFixture()})
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureCommunicationRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	if resourceGroup == "" {
		switch strings.ToLower(strings.TrimSpace(tail[0])) {
		case "checknameavailability":
			if r.Method != http.MethodPost || len(tail) != 1 {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return true
			}
			s.handleAzureCommunicationCheckNameAvailability(w, r, path, subscriptionID)
			return true
		case "listverifiedexchangeonlinedomains":
			if (r.Method != http.MethodPost && r.Method != http.MethodGet) || len(tail) != 1 {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return true
			}
			respondJSON(w, http.StatusOK, map[string]any{"value": azureCommunicationVerifiedExchangeOnlineDomainsFixture(subscriptionID)})
			return true
		case "communicationservices":
			if len(tail) == 1 && r.Method == http.MethodGet {
				s.handleAzureCommunicationListServicesBySubscription(w, subscriptionID)
				return true
			}
		case "emailservices":
			if len(tail) == 1 && r.Method == http.MethodGet {
				s.handleAzureCommunicationListEmailServicesBySubscription(w, subscriptionID)
				return true
			}
		}
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "communicationservices":
		s.handleAzureCommunicationServicesRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	case "emailservices":
		s.handleAzureCommunicationEmailServicesRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		respondAzureImplemented(w, path)
		return true
	}
}

func azureIsCommunicationPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Communication") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureCommunicationRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability", "listverifiedexchangeonlinedomains", "communicationservices", "emailservices":
		return true
	default:
		return false
	}
}

func azureCommunicationAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureCommunicationRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Communication") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Communication") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureCommunicationCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	_ = subscriptionID

	name := strings.TrimSpace(azureCommunicationString(payload["name"]))
	resourceType := strings.TrimSpace(azureCommunicationString(payload["type"]))
	if name == "" || resourceType == "" {
		respondAzureInvalidRequest(w, path, "name and type are required")
		return
	}
	if !strings.EqualFold(resourceType, "Microsoft.Communication/CommunicationServices") {
		respondAzureInvalidRequest(w, path, "type must be Microsoft.Communication/CommunicationServices")
		return
	}
	if !azureCommunicationValidServiceName(name) {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "resource name must be 3-63 characters of letters, digits, or hyphens",
		})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, item := range s.azureCommunicationServices {
		if strings.EqualFold(item.Name, name) {
			respondJSON(w, http.StatusOK, map[string]any{
				"nameAvailable": false,
				"reason":        "AlreadyExists",
				"message":       "resource name is already in use",
			})
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": true})
}

func (s *Server) handleAzureCommunicationServicesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationListServicesByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCommunicationValidServiceName(name) {
		respondAzureInvalidRequest(w, path, "communication service name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationServiceGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureCommunicationServiceGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureCommunicationServiceCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureCommunicationServicePatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureCommunicationServiceDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "listkeys":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureCommunicationServiceListKeys(w, path, subscriptionID, resourceGroup, name)
			return
		}
	case "regeneratekey":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureCommunicationServiceRegenerateKey(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	case "linknotificationhub":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureCommunicationServiceLinkNotificationHub(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	case "smtpusernames":
		s.handleAzureCommunicationSMTPUsernamesRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCommunicationEmailServicesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationListEmailServicesByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCommunicationValidServiceName(name) {
		respondAzureInvalidRequest(w, path, "email service name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationEmailServiceGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureCommunicationEmailServiceGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureCommunicationEmailServiceCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureCommunicationEmailServicePatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureCommunicationEmailServiceDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "domains":
		s.handleAzureCommunicationDomainsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCommunicationDomainsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationDomainsList(w, subscriptionID, resourceGroup, emailServiceName)
		return
	}

	domainName := strings.TrimSpace(tail[1])
	if !azureCommunicationValidDomainName(domainName) {
		respondAzureInvalidRequest(w, path, "domain name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationDomainGet(w, path, subscriptionID, resourceGroup, emailServiceName, domainName, false)
		case http.MethodHead:
			s.handleAzureCommunicationDomainGet(w, path, subscriptionID, resourceGroup, emailServiceName, domainName, true)
		case http.MethodPut:
			s.handleAzureCommunicationDomainCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName)
		case http.MethodPatch:
			s.handleAzureCommunicationDomainPatch(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName)
		case http.MethodDelete:
			s.handleAzureCommunicationDomainDelete(w, path, subscriptionID, resourceGroup, emailServiceName, domainName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "initiateverification":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureCommunicationDomainVerificationAction(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, "Pending")
			return
		}
	case "cancelverification":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureCommunicationDomainVerificationAction(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, "Canceled")
			return
		}
	case "senderusernames":
		s.handleAzureCommunicationSenderUsernamesRoute(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, tail[2:])
		return
	case "suppressionlists":
		s.handleAzureCommunicationSuppressionListsRoute(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCommunicationSenderUsernamesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationNestedList(w, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "senderUsernames")
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCommunicationValidChildName(name) {
		respondAzureInvalidRequest(w, path, "sender username name is invalid")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "senderUsernames", name, false)
	case http.MethodHead:
		s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "senderUsernames", name, true)
	case http.MethodPut:
		s.handleAzureCommunicationSenderUsernameCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, name)
	case http.MethodDelete:
		s.handleAzureCommunicationNestedDelete(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "senderUsernames", name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCommunicationSMTPUsernamesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, communicationName string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationNestedList(w, subscriptionID, resourceGroup, communicationName, "", "", "", "smtpUsernames")
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCommunicationValidChildName(name) {
		respondAzureInvalidRequest(w, path, "smtp username name is invalid")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, communicationName, "", "", "", "smtpUsernames", name, false)
	case http.MethodHead:
		s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, communicationName, "", "", "", "smtpUsernames", name, true)
	case http.MethodPut:
		s.handleAzureCommunicationSMTPUsernameCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, communicationName, name)
	case http.MethodDelete:
		s.handleAzureCommunicationNestedDelete(w, path, subscriptionID, resourceGroup, communicationName, "", "", "", "smtpUsernames", name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCommunicationSuppressionListsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationNestedList(w, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "suppressionLists")
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCommunicationValidChildName(name) {
		respondAzureInvalidRequest(w, path, "suppression list name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "suppressionLists", name, false)
		case http.MethodHead:
			s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "suppressionLists", name, true)
		case http.MethodPut:
			s.handleAzureCommunicationSuppressionListCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, name)
		case http.MethodDelete:
			s.handleAzureCommunicationNestedDelete(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "suppressionLists", name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.EqualFold(tail[2], "suppressionListAddresses") {
		s.handleAzureCommunicationSuppressionListAddressesRoute(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, name, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCommunicationSuppressionListAddressesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName, suppressionListName string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationNestedList(w, subscriptionID, resourceGroup, "", emailServiceName, domainName, suppressionListName, "suppressionListAddresses")
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureCommunicationValidChildName(name) {
		respondAzureInvalidRequest(w, path, "suppression list address name is invalid")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, suppressionListName, "suppressionListAddresses", name, false)
	case http.MethodHead:
		s.handleAzureCommunicationNestedGet(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, suppressionListName, "suppressionListAddresses", name, true)
	case http.MethodPut:
		s.handleAzureCommunicationSuppressionListAddressCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, emailServiceName, domainName, suppressionListName, name)
	case http.MethodDelete:
		s.handleAzureCommunicationNestedDelete(w, path, subscriptionID, resourceGroup, "", emailServiceName, domainName, suppressionListName, "suppressionListAddresses", name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCommunicationListServicesBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationServiceResource, 0, len(s.azureCommunicationServices))
	for _, item := range s.azureCommunicationServices {
		if item != nil && strings.EqualFold(item.SubscriptionID, subscriptionID) {
			items = append(items, item)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureCommunicationServiceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureCommunicationListServicesByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationServiceResource, 0, len(s.azureCommunicationServices))
	for _, item := range s.azureCommunicationServices {
		if item != nil && strings.EqualFold(item.SubscriptionID, subscriptionID) && strings.EqualFold(item.ResourceGroup, resourceGroup) {
			items = append(items, item)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureCommunicationServiceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureCommunicationServiceCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}

	location := strings.TrimSpace(azureCommunicationString(payload["location"]))
	properties := azureCommunicationObject(payload["properties"])
	dataLocation := strings.TrimSpace(azureCommunicationString(properties["dataLocation"]))
	if location == "" || dataLocation == "" {
		respondAzureInvalidRequest(w, path, "location and properties.dataLocation are required")
		return
	}

	now := time.Now().UTC()
	key := azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)
	created := false

	s.providerStorageMu.Lock()
	resource := s.azureCommunicationServices[key]
	if resource == nil {
		created = true
		resource = &azureCommunicationServiceResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			PrimaryKey:     azureCommunicationDigest("primary|" + key)[:32],
			SecondaryKey:   azureCommunicationDigest("secondary|" + key)[:32],
			CreatedAt:      now,
		}
		s.azureCommunicationServices[key] = resource
	}
	resource.Location = location
	resource.Identity = azureCommunicationCloneObject(azureCommunicationObject(payload["identity"]))
	resource.Tags = azureCommunicationStringMap(payload["tags"])
	resource.Properties = azureCommunicationCloneObject(properties)
	resource.Properties["provisioningState"] = "Succeeded"
	resource.Properties["hostName"] = fmt.Sprintf("%s.communication.azure.local", strings.ToLower(name))
	resource.Properties["version"] = azureCommunicationAPIVersion
	resource.UpdatedAt = now
	resource.ETag = azureCommunicationETag("communication-service", key, now)
	fixture := azureCommunicationServiceFixture(resource)
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationServicePatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}

	key := azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationServices[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "communication service not found")
		return
	}
	if tagsRaw, exists := payload["tags"]; exists {
		resource.Tags = azureCommunicationStringMap(tagsRaw)
	}
	if identityRaw, exists := payload["identity"]; exists {
		resource.Identity = azureCommunicationCloneObject(azureCommunicationObject(identityRaw))
	}
	if propertiesRaw, exists := payload["properties"]; exists {
		for key, value := range azureCommunicationObject(propertiesRaw) {
			resource.Properties[key] = value
		}
		resource.Properties["provisioningState"] = "Succeeded"
	}
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureCommunicationETag("communication-service", key, resource.UpdatedAt)
	fixture := azureCommunicationServiceFixture(resource)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationServiceGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationServices[azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondAzureInvalidRequest(w, path, "communication service not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationServiceFixture(resource))
}

func (s *Server) handleAzureCommunicationServiceDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	if s.azureCommunicationServices[key] == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "communication service not found")
		return
	}
	delete(s.azureCommunicationServices, key)
	for nestedKey, item := range s.azureCommunicationNestedResources {
		if item != nil && item.Collection == "smtpUsernames" &&
			strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.CommunicationName, name) {
			delete(s.azureCommunicationNestedResources, nestedKey)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureCommunicationServiceListKeys(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationServices[azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondAzureInvalidRequest(w, path, "communication service not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"primaryKey":                resource.PrimaryKey,
		"secondaryKey":              resource.SecondaryKey,
		"primaryConnectionString":   fmt.Sprintf("endpoint=https://%s.communication.azure.local/;accesskey=%s", strings.ToLower(resource.Name), resource.PrimaryKey),
		"secondaryConnectionString": fmt.Sprintf("endpoint=https://%s.communication.azure.local/;accesskey=%s", strings.ToLower(resource.Name), resource.SecondaryKey),
	})
}

func (s *Server) handleAzureCommunicationServiceRegenerateKey(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	keyType := strings.ToLower(strings.TrimSpace(azureCommunicationString(payload["keyType"])))
	if keyType != "primary" && keyType != "secondary" {
		respondAzureInvalidRequest(w, path, "keyType must be Primary or Secondary")
		return
	}

	key := azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationServices[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "communication service not found")
		return
	}
	now := time.Now().UTC()
	if keyType == "primary" {
		resource.PrimaryKey = azureCommunicationDigest("primary|" + key + "|" + now.Format(time.RFC3339Nano))[:32]
	} else {
		resource.SecondaryKey = azureCommunicationDigest("secondary|" + key + "|" + now.Format(time.RFC3339Nano))[:32]
	}
	resource.UpdatedAt = now
	resource.ETag = azureCommunicationETag("communication-service", key, now)
	response := map[string]any{
		"primaryKey":   resource.PrimaryKey,
		"secondaryKey": resource.SecondaryKey,
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleAzureCommunicationServiceLinkNotificationHub(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	key := azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	resource := s.azureCommunicationServices[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "communication service not found")
		return
	}
	resource.Properties["linkedNotificationHub"] = azureCommunicationCloneObject(payload)
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureCommunicationETag("communication-service", key, resource.UpdatedAt)
	fixture := azureCommunicationServiceFixture(resource)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationListEmailServicesBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationEmailServiceResource, 0, len(s.azureCommunicationEmailServices))
	for _, item := range s.azureCommunicationEmailServices {
		if item != nil && strings.EqualFold(item.SubscriptionID, subscriptionID) {
			items = append(items, item)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureCommunicationEmailServiceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureCommunicationListEmailServicesByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationEmailServiceResource, 0, len(s.azureCommunicationEmailServices))
	for _, item := range s.azureCommunicationEmailServices {
		if item != nil && strings.EqualFold(item.SubscriptionID, subscriptionID) && strings.EqualFold(item.ResourceGroup, resourceGroup) {
			items = append(items, item)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureCommunicationEmailServiceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureCommunicationEmailServiceCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureCommunicationString(payload["location"]))
	properties := azureCommunicationObject(payload["properties"])
	dataLocation := strings.TrimSpace(azureCommunicationString(properties["dataLocation"]))
	if location == "" || dataLocation == "" {
		respondAzureInvalidRequest(w, path, "location and properties.dataLocation are required")
		return
	}

	now := time.Now().UTC()
	key := azureCommunicationEmailServiceStoreKey(subscriptionID, resourceGroup, name)
	created := false
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationEmailServices[key]
	if resource == nil {
		created = true
		resource = &azureCommunicationEmailServiceResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			CreatedAt:      now,
		}
		s.azureCommunicationEmailServices[key] = resource
	}
	resource.Location = location
	resource.Tags = azureCommunicationStringMap(payload["tags"])
	resource.Properties = azureCommunicationCloneObject(properties)
	resource.Properties["provisioningState"] = "Succeeded"
	resource.UpdatedAt = now
	resource.ETag = azureCommunicationETag("email-service", key, now)
	fixture := azureCommunicationEmailServiceFixture(resource)
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationEmailServicePatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	key := azureCommunicationEmailServiceStoreKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationEmailServices[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "email service not found")
		return
	}
	if tagsRaw, exists := payload["tags"]; exists {
		resource.Tags = azureCommunicationStringMap(tagsRaw)
	}
	if propertiesRaw, exists := payload["properties"]; exists {
		for key, value := range azureCommunicationObject(propertiesRaw) {
			resource.Properties[key] = value
		}
		resource.Properties["provisioningState"] = "Succeeded"
	}
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureCommunicationETag("email-service", key, resource.UpdatedAt)
	fixture := azureCommunicationEmailServiceFixture(resource)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationEmailServiceGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationEmailServices[azureCommunicationEmailServiceStoreKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondAzureInvalidRequest(w, path, "email service not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationEmailServiceFixture(resource))
}

func (s *Server) handleAzureCommunicationEmailServiceDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureCommunicationEmailServiceStoreKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	if s.azureCommunicationEmailServices[key] == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "email service not found")
		return
	}
	delete(s.azureCommunicationEmailServices, key)
	for childKey, child := range s.azureCommunicationDomains {
		if child != nil &&
			strings.EqualFold(child.SubscriptionID, subscriptionID) &&
			strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.EmailServiceName, name) {
			delete(s.azureCommunicationDomains, childKey)
		}
	}
	for childKey, child := range s.azureCommunicationNestedResources {
		if child != nil &&
			strings.EqualFold(child.SubscriptionID, subscriptionID) &&
			strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.EmailServiceName, name) {
			delete(s.azureCommunicationNestedResources, childKey)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureCommunicationDomainsList(w http.ResponseWriter, subscriptionID, resourceGroup, emailServiceName string) {
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationDomainResource, 0, len(s.azureCommunicationDomains))
	for _, item := range s.azureCommunicationDomains {
		if item != nil &&
			strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.EmailServiceName, emailServiceName) {
			items = append(items, item)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureCommunicationDomainFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureCommunicationDomainCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureCommunicationString(payload["location"]))
	properties := azureCommunicationObject(payload["properties"])
	domainManagement := strings.TrimSpace(azureCommunicationString(properties["domainManagement"]))
	if location == "" || domainManagement == "" {
		respondAzureInvalidRequest(w, path, "location and properties.domainManagement are required")
		return
	}

	if !s.azureCommunicationEmailServiceExists(subscriptionID, resourceGroup, emailServiceName) {
		respondAzureInvalidRequest(w, path, "parent email service not found")
		return
	}

	now := time.Now().UTC()
	key := azureCommunicationDomainStoreKey(subscriptionID, resourceGroup, emailServiceName, domainName)
	created := false
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationDomains[key]
	if resource == nil {
		created = true
		resource = &azureCommunicationDomainResource{
			SubscriptionID:     subscriptionID,
			ResourceGroup:      resourceGroup,
			EmailServiceName:   emailServiceName,
			Name:               domainName,
			VerificationStatus: map[string]string{"Domain": "NotStarted", "SPF": "NotStarted", "DKIM": "NotStarted"},
			CreatedAt:          now,
		}
		s.azureCommunicationDomains[key] = resource
	}
	resource.Location = location
	resource.Tags = azureCommunicationStringMap(payload["tags"])
	resource.Properties = azureCommunicationCloneObject(properties)
	resource.Properties["provisioningState"] = "Succeeded"
	resource.UpdatedAt = now
	resource.ETag = azureCommunicationETag("domain", key, now)
	fixture := azureCommunicationDomainFixture(resource)
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationDomainPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	key := azureCommunicationDomainStoreKey(subscriptionID, resourceGroup, emailServiceName, domainName)
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationDomains[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "domain not found")
		return
	}
	if tagsRaw, exists := payload["tags"]; exists {
		resource.Tags = azureCommunicationStringMap(tagsRaw)
	}
	if propertiesRaw, exists := payload["properties"]; exists {
		for key, value := range azureCommunicationObject(propertiesRaw) {
			resource.Properties[key] = value
		}
		resource.Properties["provisioningState"] = "Succeeded"
	}
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureCommunicationETag("domain", key, resource.UpdatedAt)
	fixture := azureCommunicationDomainFixture(resource)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationDomainGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, emailServiceName, domainName string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationDomains[azureCommunicationDomainStoreKey(subscriptionID, resourceGroup, emailServiceName, domainName)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondAzureInvalidRequest(w, path, "domain not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationDomainFixture(resource))
}

func (s *Server) handleAzureCommunicationDomainDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, emailServiceName, domainName string) {
	key := azureCommunicationDomainStoreKey(subscriptionID, resourceGroup, emailServiceName, domainName)
	s.providerStorageMu.Lock()
	if s.azureCommunicationDomains[key] == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "domain not found")
		return
	}
	delete(s.azureCommunicationDomains, key)
	for childKey, child := range s.azureCommunicationNestedResources {
		if child != nil &&
			strings.EqualFold(child.SubscriptionID, subscriptionID) &&
			strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.EmailServiceName, emailServiceName) &&
			strings.EqualFold(child.DomainName, domainName) {
			delete(s.azureCommunicationNestedResources, childKey)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureCommunicationDomainVerificationAction(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName, status string) {
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	verificationType := strings.TrimSpace(azureCommunicationString(payload["verificationType"]))
	if verificationType == "" {
		verificationType = "Domain"
	}

	key := azureCommunicationDomainStoreKey(subscriptionID, resourceGroup, emailServiceName, domainName)
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationDomains[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "domain not found")
		return
	}
	resource.VerificationStatus[verificationType] = status
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureCommunicationETag("domain", key, resource.UpdatedAt)
	fixture := azureCommunicationDomainFixture(resource)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationSenderUsernameCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName, name string) {
	if !s.azureCommunicationDomainExists(subscriptionID, resourceGroup, emailServiceName, domainName) {
		respondAzureInvalidRequest(w, path, "parent domain not found")
		return
	}
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	properties := azureCommunicationObject(payload["properties"])
	username := strings.TrimSpace(azureCommunicationString(properties["username"]))
	if username == "" {
		respondAzureInvalidRequest(w, path, "properties.username is required")
		return
	}
	s.handleAzureCommunicationNestedCreateOrUpdate(w, http.StatusCreated, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "senderUsernames", name, "", azureCommunicationStringMap(payload["tags"]), properties)
}

func (s *Server) handleAzureCommunicationSMTPUsernameCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, communicationName, name string) {
	if !s.azureCommunicationServiceExists(subscriptionID, resourceGroup, communicationName) {
		respondAzureInvalidRequest(w, path, "parent communication service not found")
		return
	}
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	properties := azureCommunicationObject(payload["properties"])
	username := strings.TrimSpace(azureCommunicationString(properties["username"]))
	entraApplicationID := strings.TrimSpace(azureCommunicationString(properties["entraApplicationId"]))
	tenantID := strings.TrimSpace(azureCommunicationString(properties["tenantId"]))
	if username == "" || entraApplicationID == "" || tenantID == "" {
		respondAzureInvalidRequest(w, path, "properties.username, properties.entraApplicationId, and properties.tenantId are required")
		return
	}
	if !azureCommunicationGUIDPattern.MatchString(entraApplicationID) || !azureCommunicationGUIDPattern.MatchString(tenantID) {
		respondAzureInvalidRequest(w, path, "properties.entraApplicationId and properties.tenantId must be GUID-like strings")
		return
	}
	s.handleAzureCommunicationNestedCreateOrUpdate(w, http.StatusCreated, subscriptionID, resourceGroup, communicationName, "", "", "", "smtpUsernames", name, "", azureCommunicationStringMap(payload["tags"]), properties)
}

func (s *Server) handleAzureCommunicationSuppressionListCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName, name string) {
	if !s.azureCommunicationDomainExists(subscriptionID, resourceGroup, emailServiceName, domainName) {
		respondAzureInvalidRequest(w, path, "parent domain not found")
		return
	}
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	properties := azureCommunicationObject(payload["properties"])
	listName := strings.TrimSpace(azureCommunicationString(properties["listName"]))
	if listName == "" {
		respondAzureInvalidRequest(w, path, "properties.listName is required")
		return
	}
	s.handleAzureCommunicationNestedCreateOrUpdate(w, http.StatusCreated, subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "suppressionLists", name, "", azureCommunicationStringMap(payload["tags"]), properties)
}

func (s *Server) handleAzureCommunicationSuppressionListAddressCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, emailServiceName, domainName, suppressionListName, name string) {
	if !s.azureCommunicationNestedExists(subscriptionID, resourceGroup, "", emailServiceName, domainName, "", "suppressionLists", suppressionListName) {
		respondAzureInvalidRequest(w, path, "parent suppression list not found")
		return
	}
	_, payload, ok := azureCommunicationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	properties := azureCommunicationObject(payload["properties"])
	email := strings.TrimSpace(azureCommunicationString(properties["email"]))
	if !azureCommunicationValidEmail(email) {
		respondAzureInvalidRequest(w, path, "properties.email must be a valid email address")
		return
	}
	s.handleAzureCommunicationNestedCreateOrUpdate(w, http.StatusCreated, subscriptionID, resourceGroup, "", emailServiceName, domainName, suppressionListName, "suppressionListAddresses", name, "", azureCommunicationStringMap(payload["tags"]), properties)
}

func (s *Server) handleAzureCommunicationNestedCreateOrUpdate(w http.ResponseWriter, createStatus int, subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name, location string, tags map[string]string, properties map[string]any) {
	now := time.Now().UTC()
	key := azureCommunicationNestedStoreKey(subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name)
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationNestedResources[key]
	created := false
	if resource == nil {
		created = true
		resource = &azureCommunicationNestedResource{
			SubscriptionID:      subscriptionID,
			ResourceGroup:       resourceGroup,
			CommunicationName:   communicationName,
			EmailServiceName:    emailServiceName,
			DomainName:          domainName,
			SuppressionListName: suppressionListName,
			Collection:          collection,
			Name:                name,
			CreatedAt:           now,
		}
		s.azureCommunicationNestedResources[key] = resource
	}
	resource.Location = location
	resource.Tags = azureCommunicationCloneStringMap(tags)
	resource.Properties = azureCommunicationCloneObject(properties)
	resource.Properties["provisioningState"] = "Succeeded"
	resource.UpdatedAt = now
	resource.ETag = azureCommunicationETag(collection, key, now)
	fixture := azureCommunicationNestedFixture(resource)
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = createStatus
	}
	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationNestedList(w http.ResponseWriter, subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection string) {
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationNestedResource, 0, len(s.azureCommunicationNestedResources))
	for _, item := range s.azureCommunicationNestedResources {
		if item == nil || item.Collection != collection {
			continue
		}
		if !strings.EqualFold(item.SubscriptionID, subscriptionID) || !strings.EqualFold(item.ResourceGroup, resourceGroup) {
			continue
		}
		if !strings.EqualFold(item.CommunicationName, communicationName) ||
			!strings.EqualFold(item.EmailServiceName, emailServiceName) ||
			!strings.EqualFold(item.DomainName, domainName) ||
			!strings.EqualFold(item.SuppressionListName, suppressionListName) {
			continue
		}
		items = append(items, item)
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureCommunicationNestedFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureCommunicationNestedGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureCommunicationNestedResources[azureCommunicationNestedStoreKey(subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondAzureInvalidRequest(w, path, "resource not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationNestedFixture(resource))
}

func (s *Server) handleAzureCommunicationNestedDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name string) {
	key := azureCommunicationNestedStoreKey(subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name)
	s.providerStorageMu.Lock()
	if s.azureCommunicationNestedResources[key] == nil {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "resource not found")
		return
	}
	delete(s.azureCommunicationNestedResources, key)
	if collection == "suppressionLists" {
		for childKey, item := range s.azureCommunicationNestedResources {
			if item != nil && item.Collection == "suppressionListAddresses" &&
				strings.EqualFold(item.SubscriptionID, subscriptionID) &&
				strings.EqualFold(item.ResourceGroup, resourceGroup) &&
				strings.EqualFold(item.EmailServiceName, emailServiceName) &&
				strings.EqualFold(item.DomainName, domainName) &&
				strings.EqualFold(item.SuppressionListName, name) {
				delete(s.azureCommunicationNestedResources, childKey)
			}
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) azureCommunicationServiceExists(subscriptionID, resourceGroup, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureCommunicationServices[azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name)] != nil
}

func (s *Server) azureCommunicationEmailServiceExists(subscriptionID, resourceGroup, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureCommunicationEmailServices[azureCommunicationEmailServiceStoreKey(subscriptionID, resourceGroup, name)] != nil
}

func (s *Server) azureCommunicationDomainExists(subscriptionID, resourceGroup, emailServiceName, domainName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureCommunicationDomains[azureCommunicationDomainStoreKey(subscriptionID, resourceGroup, emailServiceName, domainName)] != nil
}

func (s *Server) azureCommunicationNestedExists(subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureCommunicationNestedResources[azureCommunicationNestedStoreKey(subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name)] != nil
}

func azureCommunicationServiceFixture(resource *azureCommunicationServiceResource) map[string]any {
	fixture := map[string]any{
		"id":         azureCommunicationServiceResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.Communication/communicationServices",
		"location":   resource.Location,
		"tags":       azureCommunicationCloneStringMap(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureCommunicationSystemDataFixture(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureCommunicationCloneObject(resource.Properties),
	}
	if len(resource.Identity) > 0 {
		fixture["identity"] = azureCommunicationCloneObject(resource.Identity)
	}
	return fixture
}

func azureCommunicationEmailServiceFixture(resource *azureCommunicationEmailServiceResource) map[string]any {
	return map[string]any{
		"id":         azureCommunicationEmailServiceResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.Communication/emailServices",
		"location":   resource.Location,
		"tags":       azureCommunicationCloneStringMap(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureCommunicationSystemDataFixture(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureCommunicationCloneObject(resource.Properties),
	}
}

func azureCommunicationDomainFixture(resource *azureCommunicationDomainResource) map[string]any {
	properties := azureCommunicationCloneObject(resource.Properties)
	verification := make(map[string]any, len(resource.VerificationStatus))
	for key, value := range resource.VerificationStatus {
		verification[key] = map[string]any{"status": value}
	}
	properties["verificationRecords"] = verification
	return map[string]any{
		"id":         azureCommunicationDomainResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.EmailServiceName, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.Communication/emailServices/domains",
		"location":   resource.Location,
		"tags":       azureCommunicationCloneStringMap(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureCommunicationSystemDataFixture(resource.CreatedAt, resource.UpdatedAt),
		"properties": properties,
	}
}

func azureCommunicationNestedFixture(resource *azureCommunicationNestedResource) map[string]any {
	return map[string]any{
		"id":         azureCommunicationNestedResourceID(resource),
		"name":       resource.Name,
		"type":       azureCommunicationNestedResourceType(resource.Collection),
		"location":   resource.Location,
		"tags":       azureCommunicationCloneStringMap(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureCommunicationSystemDataFixture(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureCommunicationCloneObject(resource.Properties),
	}
}

func azureCommunicationOperationsFixture() []any {
	operations := []string{
		"Microsoft.Communication/communicationServices/read",
		"Microsoft.Communication/communicationServices/write",
		"Microsoft.Communication/communicationServices/delete",
		"Microsoft.Communication/communicationServices/listKeys/action",
		"Microsoft.Communication/communicationServices/regenerateKey/action",
		"Microsoft.Communication/emailServices/read",
		"Microsoft.Communication/emailServices/write",
		"Microsoft.Communication/emailServices/delete",
		"Microsoft.Communication/emailServices/domains/read",
		"Microsoft.Communication/emailServices/domains/write",
		"Microsoft.Communication/emailServices/domains/delete",
		"Microsoft.Communication/emailServices/domains/senderUsernames/read",
		"Microsoft.Communication/emailServices/domains/suppressionLists/read",
	}
	values := make([]any, 0, len(operations))
	for _, name := range operations {
		values = append(values, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":  "Microsoft.Communication",
				"resource":  "Communication",
				"operation": name,
			},
		})
	}
	return values
}

func azureCommunicationVerifiedExchangeOnlineDomainsFixture(subscriptionID string) []any {
	seed := azureCommunicationDigest(subscriptionID)
	return []any{
		"verified-" + seed[:8] + ".contoso.com",
		"verified-" + seed[8:16] + ".fabrikam.com",
	}
}

func azureCommunicationReadJSONBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureCommunicationString(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func azureCommunicationObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureCommunicationCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureCommunicationCloneObject(typed)
		case []any:
			cloned := make([]any, len(typed))
			for i, item := range typed {
				if object, ok := item.(map[string]any); ok {
					cloned[i] = azureCommunicationCloneObject(object)
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

func azureCommunicationStringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	items, ok := value.(map[string]any)
	if !ok {
		if typed, ok := value.(map[string]string); ok {
			return azureCommunicationCloneStringMap(typed)
		}
		return map[string]string{}
	}
	out := make(map[string]string, len(items))
	for key, value := range items {
		out[key] = strings.TrimSpace(azureCommunicationString(value))
	}
	return out
}

func azureCommunicationCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func azureCommunicationSystemDataFixture(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339Nano),
		"createdBy":          "Stackyard",
		"createdByType":      "Application",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339Nano),
		"lastModifiedBy":     "Stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureCommunicationValidServiceName(value string) bool {
	if len(value) < 3 || len(value) > 63 {
		return false
	}
	return azureCommunicationServiceNamePattern.MatchString(value)
}

func azureCommunicationValidDomainName(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 3 || len(value) > 253 || strings.ContainsAny(value, " /") {
		return false
	}
	return strings.Contains(value, ".")
}

func azureCommunicationValidChildName(value string) bool {
	if len(value) < 1 || len(value) > 63 {
		return false
	}
	return azureCommunicationChildNamePattern.MatchString(value)
}

func azureCommunicationValidEmail(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 3 || len(value) > 320 || strings.Count(value, "@") != 1 {
		return false
	}
	parts := strings.Split(value, "@")
	return parts[0] != "" && azureCommunicationValidDomainName(parts[1])
}

func azureCommunicationDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationETag(kind, seed string, now time.Time) string {
	return `"` + kind + "-" + azureCommunicationDigest(seed + "|" + now.Format(time.RFC3339Nano))[:16] + `"`
}

func azureCommunicationServiceStoreKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID) + "|" + strings.TrimSpace(resourceGroup) + "|" + strings.TrimSpace(name))
}

func azureCommunicationEmailServiceStoreKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID) + "|" + strings.TrimSpace(resourceGroup) + "|" + strings.TrimSpace(name))
}

func azureCommunicationDomainStoreKey(subscriptionID, resourceGroup, emailServiceName, domainName string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID) + "|" + strings.TrimSpace(resourceGroup) + "|" + strings.TrimSpace(emailServiceName) + "|" + strings.TrimSpace(domainName))
}

func azureCommunicationNestedStoreKey(subscriptionID, resourceGroup, communicationName, emailServiceName, domainName, suppressionListName, collection, name string) string {
	return strings.ToLower(strings.Join([]string{
		strings.TrimSpace(subscriptionID),
		strings.TrimSpace(resourceGroup),
		strings.TrimSpace(communicationName),
		strings.TrimSpace(emailServiceName),
		strings.TrimSpace(domainName),
		strings.TrimSpace(suppressionListName),
		strings.TrimSpace(collection),
		strings.TrimSpace(name),
	}, "|"))
}

func azureCommunicationServiceResourceID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Communication/communicationServices/%s", subscriptionID, resourceGroup, name)
}

func azureCommunicationEmailServiceResourceID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Communication/emailServices/%s", subscriptionID, resourceGroup, name)
}

func azureCommunicationDomainResourceID(subscriptionID, resourceGroup, emailServiceName, domainName string) string {
	return fmt.Sprintf("%s/domains/%s", azureCommunicationEmailServiceResourceID(subscriptionID, resourceGroup, emailServiceName), domainName)
}

func azureCommunicationNestedResourceID(resource *azureCommunicationNestedResource) string {
	switch resource.Collection {
	case "smtpUsernames":
		return fmt.Sprintf("%s/smtpUsernames/%s", azureCommunicationServiceResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.CommunicationName), resource.Name)
	case "senderUsernames":
		return fmt.Sprintf("%s/senderUsernames/%s", azureCommunicationDomainResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.EmailServiceName, resource.DomainName), resource.Name)
	case "suppressionLists":
		return fmt.Sprintf("%s/suppressionLists/%s", azureCommunicationDomainResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.EmailServiceName, resource.DomainName), resource.Name)
	case "suppressionListAddresses":
		return fmt.Sprintf("%s/suppressionListAddresses/%s", fmt.Sprintf("%s/suppressionLists/%s", azureCommunicationDomainResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.EmailServiceName, resource.DomainName), resource.SuppressionListName), resource.Name)
	default:
		return ""
	}
}

func azureCommunicationNestedResourceType(collection string) string {
	switch collection {
	case "smtpUsernames":
		return "Microsoft.Communication/communicationServices/smtpUsernames"
	case "senderUsernames":
		return "Microsoft.Communication/emailServices/domains/senderUsernames"
	case "suppressionLists":
		return "Microsoft.Communication/emailServices/domains/suppressionLists"
	case "suppressionListAddresses":
		return "Microsoft.Communication/emailServices/domains/suppressionLists/suppressionListAddresses"
	default:
		return "Microsoft.Communication/unknown"
	}
}
