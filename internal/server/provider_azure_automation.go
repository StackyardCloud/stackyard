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

type azureAutomationAccount struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Identity       map[string]any
	SKU            map[string]any
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureAutomationAsset struct {
	SubscriptionID string
	ResourceGroup  string
	AccountName    string
	Collection     string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	Content        string
	DraftContent   string
	URI            string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureAutomationJobStream struct {
	ID         string
	StreamType string
	Summary    string
	Text       string
	CreatedAt  time.Time
}

type azureAutomationJob struct {
	SubscriptionID string
	ResourceGroup  string
	AccountName    string
	Name           string
	RunbookName    string
	Parameters     map[string]any
	Status         string
	Output         string
	RunbookContent string
	Streams        []azureAutomationJobStream
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StartedAt      time.Time
}

type azureAutomationSyncJobStream struct {
	ID         string
	StreamType string
	Summary    string
	Text       string
	CreatedAt  time.Time
}

type azureAutomationSyncJob struct {
	SubscriptionID    string
	ResourceGroup     string
	AccountName       string
	SourceControlName string
	Name              string
	Properties        map[string]any
	Streams           []azureAutomationSyncJobStream
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

var azureAutomationGenericCollections = map[string]struct{}{
	"certificates":               {},
	"connectiontypes":            {},
	"connections":                {},
	"credentials":                {},
	"jobschedules":               {},
	"modules":                    {},
	"privateendpointconnections": {},
	"runtimeenvironments":        {},
	"schedules":                  {},
	"variables":                  {},
	"webhooks":                   {},
}

func (s *Server) handleAzureAutomationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsAutomationPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureAutomationAllowedMethod(r.Method) {
		return false
	}

	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Automation") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureAutomationOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureAutomationRouteContext(segments)
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		if len(tail) == 1 && strings.EqualFold(tail[0], "automationAccounts") && r.Method == http.MethodGet {
			s.handleAzureAutomationAccountListBySubscription(w, subscriptionID)
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "automationAccounts") && r.Method == http.MethodGet:
		s.handleAzureAutomationAccountListByResourceGroup(w, subscriptionID, resourceGroup)
		return true
	case len(tail) >= 2 && strings.EqualFold(tail[0], "automationAccounts"):
		accountName := strings.TrimSpace(tail[1])
		if accountName == "" {
			respondAzureInvalidRequest(w, path, "automation account name is required")
			return true
		}
		if len(tail) == 2 {
			switch r.Method {
			case http.MethodGet:
				s.handleAzureAutomationAccountGet(w, path, subscriptionID, resourceGroup, accountName, false)
			case http.MethodHead:
				s.handleAzureAutomationAccountGet(w, path, subscriptionID, resourceGroup, accountName, true)
			case http.MethodPut:
				s.handleAzureAutomationAccountCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName)
			case http.MethodPatch:
				s.handleAzureAutomationAccountPatch(w, r, path, subscriptionID, resourceGroup, accountName)
			case http.MethodDelete:
				s.handleAzureAutomationAccountDelete(w, subscriptionID, resourceGroup, accountName)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return true
		}
		s.handleAzureAutomationAccountChildRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[2:])
		return true
	default:
		respondAzureImplemented(w, path)
		return true
	}
}

func (s *Server) handleAzureAutomationAccountChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "statistics":
		if len(tail) == 1 && r.Method == http.MethodGet {
			s.handleAzureAutomationStatisticsList(w, subscriptionID, resourceGroup, accountName)
			return
		}
	case "usages":
		if len(tail) == 1 && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureAutomationUsagesFixture(subscriptionID, resourceGroup, accountName)})
			return
		}
	case "privatelinkresources":
		if len(tail) == 1 && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureAutomationPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName)})
			return
		}
	case "runbooks":
		s.handleAzureAutomationRunbookRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[1:])
		return
	case "jobs":
		s.handleAzureAutomationJobRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[1:])
		return
	case "sourcecontrols":
		s.handleAzureAutomationSourceControlRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[1:])
		return
	case "runtimeenvironments":
		s.handleAzureAutomationRuntimeEnvironmentRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[1:])
		return
	case "webhooks":
		s.handleAzureAutomationWebhookRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[1:])
		return
	}

	if _, ok := azureAutomationGenericCollections[strings.ToLower(strings.TrimSpace(tail[0]))]; ok {
		s.handleAzureAutomationGenericAssetRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[0], tail[1:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureAutomationGenericAssetRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, collection string, tail []string) {
	account, ok := s.azureAutomationGetAccount(subscriptionID, resourceGroup, accountName)
	if !ok {
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}

	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationAssetList(w, subscriptionID, resourceGroup, accountName, collection)
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, strings.TrimSuffix(collection, "s")+" name is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureAutomationAssetGet(w, path, subscriptionID, resourceGroup, accountName, collection, name)
	case http.MethodPut:
		s.handleAzureAutomationAssetCreateOrUpdate(w, r, path, account, collection, name)
	case http.MethodPatch:
		s.handleAzureAutomationAssetPatch(w, r, path, account, collection, name)
	case http.MethodDelete:
		s.handleAzureAutomationAssetDelete(w, subscriptionID, resourceGroup, accountName, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureAutomationRunbookRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	account, ok := s.azureAutomationGetAccount(subscriptionID, resourceGroup, accountName)
	if !ok {
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureAutomationAssetList(w, subscriptionID, resourceGroup, accountName, "runbooks")
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "runbook name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureAutomationAssetGet(w, path, subscriptionID, resourceGroup, accountName, "runbooks", name)
		case http.MethodPut:
			s.handleAzureAutomationRunbookCreateOrUpdate(w, r, path, account, name)
		case http.MethodPatch:
			s.handleAzureAutomationAssetPatch(w, r, path, account, "runbooks", name)
		case http.MethodDelete:
			s.handleAzureAutomationAssetDelete(w, subscriptionID, resourceGroup, accountName, "runbooks", name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "content":
		if r.Method != http.MethodGet || len(tail) != 2 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationRunbookContentGet(w, path, subscriptionID, resourceGroup, accountName, name, false)
	case "draft":
		s.handleAzureAutomationRunbookDraftRoute(w, r, path, subscriptionID, resourceGroup, accountName, name, tail[2:])
	case "publish":
		if r.Method != http.MethodPost || len(tail) != 2 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationRunbookPublish(w, path, subscriptionID, resourceGroup, accountName, name)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAutomationRunbookDraftRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, runbookName string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzureAutomationRunbookDraftGet(w, path, subscriptionID, resourceGroup, accountName, runbookName)
	case len(tail) == 1 && strings.EqualFold(tail[0], "content") && r.Method == http.MethodGet:
		s.handleAzureAutomationRunbookContentGet(w, path, subscriptionID, resourceGroup, accountName, runbookName, true)
	case len(tail) == 1 && strings.EqualFold(tail[0], "undoedit") && r.Method == http.MethodPost:
		s.handleAzureAutomationRunbookDraftUndoEdit(w, path, subscriptionID, resourceGroup, accountName, runbookName)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAutomationWebhookRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	account, ok := s.azureAutomationGetAccount(subscriptionID, resourceGroup, accountName)
	if !ok {
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "generateUri") && r.Method == http.MethodPost {
		respondJSON(w, http.StatusOK, azureAutomationGenerateWebhookURI(subscriptionID, resourceGroup, accountName, ""))
		return
	}
	s.handleAzureAutomationGenericAssetRoute(w, r, path, account.SubscriptionID, account.ResourceGroup, account.Name, "webhooks", tail)
}

func (s *Server) handleAzureAutomationJobRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if _, ok := s.azureAutomationGetAccount(subscriptionID, resourceGroup, accountName); !ok {
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationJobList(w, subscriptionID, resourceGroup, accountName)
		return
	}

	jobName := strings.TrimSpace(tail[0])
	if jobName == "" {
		respondAzureInvalidRequest(w, path, "job name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureAutomationJobGet(w, path, subscriptionID, resourceGroup, accountName, jobName)
		case http.MethodPut, http.MethodPost:
			s.handleAzureAutomationJobCreate(w, r, path, subscriptionID, resourceGroup, accountName, jobName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "resume":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationJobAction(w, path, subscriptionID, resourceGroup, accountName, jobName, "Running")
	case "stop":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationJobAction(w, path, subscriptionID, resourceGroup, accountName, jobName, "Stopped")
	case "suspend":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationJobAction(w, path, subscriptionID, resourceGroup, accountName, jobName, "Suspended")
	case "output":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationJobOutput(w, path, subscriptionID, resourceGroup, accountName, jobName)
	case "runbookcontent":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationJobRunbookContent(w, path, subscriptionID, resourceGroup, accountName, jobName)
	case "streams":
		s.handleAzureAutomationJobStreamRoute(w, r, path, subscriptionID, resourceGroup, accountName, jobName, tail[2:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAutomationJobStreamRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, jobName string, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationJobStreamList(w, path, subscriptionID, resourceGroup, accountName, jobName)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.handleAzureAutomationJobStreamGet(w, path, subscriptionID, resourceGroup, accountName, jobName, tail[0])
}

func (s *Server) handleAzureAutomationSourceControlRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	account, ok := s.azureAutomationGetAccount(subscriptionID, resourceGroup, accountName)
	if !ok {
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationAssetList(w, subscriptionID, resourceGroup, accountName, "sourceControls")
		return
	}

	sourceControlName := strings.TrimSpace(tail[0])
	if sourceControlName == "" {
		respondAzureInvalidRequest(w, path, "source control name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureAutomationAssetGet(w, path, subscriptionID, resourceGroup, accountName, "sourceControls", sourceControlName)
		case http.MethodPut:
			s.handleAzureAutomationAssetCreateOrUpdate(w, r, path, account, "sourceControls", sourceControlName)
		case http.MethodPatch:
			s.handleAzureAutomationAssetPatch(w, r, path, account, "sourceControls", sourceControlName)
		case http.MethodDelete:
			s.handleAzureAutomationSourceControlDelete(w, subscriptionID, resourceGroup, accountName, sourceControlName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if !strings.EqualFold(tail[1], "sourceControlSyncJobs") {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureAutomationSourceControlSyncJobRoute(w, r, path, subscriptionID, resourceGroup, accountName, sourceControlName, tail[2:])
}

func (s *Server) handleAzureAutomationSourceControlSyncJobRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, sourceControlName string, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationSyncJobList(w, subscriptionID, resourceGroup, accountName, sourceControlName)
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "source control sync job id is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureAutomationSyncJobGet(w, path, subscriptionID, resourceGroup, accountName, sourceControlName, name)
		case http.MethodPut, http.MethodPost:
			s.handleAzureAutomationSyncJobCreate(w, r, path, subscriptionID, resourceGroup, accountName, sourceControlName, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if !strings.EqualFold(tail[1], "streams") {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureAutomationSyncJobStreamRoute(w, r, path, subscriptionID, resourceGroup, accountName, sourceControlName, name, tail[2:])
}

func (s *Server) handleAzureAutomationSyncJobStreamRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName string, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationSyncJobStreamList(w, path, subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.handleAzureAutomationSyncJobStreamGet(w, path, subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName, tail[0])
}

func (s *Server) handleAzureAutomationRuntimeEnvironmentRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	account, ok := s.azureAutomationGetAccount(subscriptionID, resourceGroup, accountName)
	if !ok {
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureAutomationAssetList(w, subscriptionID, resourceGroup, accountName, "runtimeEnvironments")
		return
	}

	runtimeName := strings.TrimSpace(tail[0])
	if runtimeName == "" {
		respondAzureInvalidRequest(w, path, "runtime environment name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureAutomationAssetGet(w, path, subscriptionID, resourceGroup, accountName, "runtimeEnvironments", runtimeName)
		case http.MethodPut:
			s.handleAzureAutomationAssetCreateOrUpdate(w, r, path, account, "runtimeEnvironments", runtimeName)
		case http.MethodPatch:
			s.handleAzureAutomationAssetPatch(w, r, path, account, "runtimeEnvironments", runtimeName)
		case http.MethodDelete:
			s.handleAzureAutomationAssetDelete(w, subscriptionID, resourceGroup, accountName, "runtimeEnvironments", runtimeName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	collection := ""
	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "packages":
		collection = fmt.Sprintf("runtimeEnvironments/%s/packages", runtimeName)
	case "python3packages":
		collection = fmt.Sprintf("runtimeEnvironments/%s/python3Packages", runtimeName)
	default:
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureAutomationGenericAssetRoute(w, r, path, subscriptionID, resourceGroup, accountName, collection, tail[2:])
}

func (s *Server) handleAzureAutomationAccountCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string) {
	body, payload, ok := azureAutomationReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureAutomationString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	key := azureAutomationAccountKey(subscriptionID, resourceGroup, accountName)

	s.providerStorageMu.Lock()
	current := s.azureAutomationAccounts[key]
	if current == nil {
		current = &azureAutomationAccount{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           accountName,
			CreatedAt:      now,
		}
	}
	current.Location = location
	current.Tags = azureAutomationStringMap(payload["tags"])
	current.Identity = azureAutomationMap(payload["identity"])
	current.SKU = azureAutomationMap(payload["sku"])
	current.Properties = azureAutomationMap(payload["properties"])
	current.UpdatedAt = now
	current.ETag = azureAutomationETag(key, body, now)
	s.azureAutomationAccounts[key] = current
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureAutomationAccountFixture(current))
}

func (s *Server) handleAzureAutomationAccountPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string) {
	body, payload, ok := azureAutomationReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureAutomationAccountKey(subscriptionID, resourceGroup, accountName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	current := s.azureAutomationAccounts[key]
	if current == nil {
		s.providerStorageMu.Unlock()
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}
	if tags, ok := payload["tags"]; ok {
		current.Tags = azureAutomationStringMap(tags)
	}
	if identity, ok := payload["identity"]; ok {
		current.Identity = azureAutomationMap(identity)
	}
	if properties, ok := payload["properties"]; ok {
		current.Properties = azureAutomationMergeMaps(current.Properties, azureAutomationMap(properties))
	}
	current.UpdatedAt = now
	current.ETag = azureAutomationETag(key, body, now)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureAutomationAccountFixture(current))
}

func (s *Server) handleAzureAutomationAccountGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName string, headOnly bool) {
	account, ok := s.azureAutomationGetAccount(subscriptionID, resourceGroup, accountName)
	if !ok {
		azureAutomationNotFound(w, path, "AutomationAccountNotFound", "automation account not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureAutomationAccountFixture(account))
}

func (s *Server) handleAzureAutomationAccountDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName string) {
	prefix := strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName))
	s.providerStorageMu.Lock()
	delete(s.azureAutomationAccounts, prefix)
	for key := range s.azureAutomationAssets {
		if strings.HasPrefix(key, prefix+"::") {
			delete(s.azureAutomationAssets, key)
		}
	}
	for key := range s.azureAutomationJobs {
		if strings.HasPrefix(key, prefix+"::job::") {
			delete(s.azureAutomationJobs, key)
		}
	}
	for key := range s.azureAutomationSyncJobs {
		if strings.HasPrefix(key, prefix+"::sourcecontrol::") {
			delete(s.azureAutomationSyncJobs, key)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureAutomationAccountListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0, len(s.azureAutomationAccounts))
	for _, account := range s.azureAutomationAccounts {
		if account.SubscriptionID != subscriptionID {
			continue
		}
		items = append(items, azureAutomationAccountFixture(account))
	}
	s.providerStorageMu.Unlock()
	azureAutomationSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAutomationAccountListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0, len(s.azureAutomationAccounts))
	for _, account := range s.azureAutomationAccounts {
		if account.SubscriptionID != subscriptionID || !strings.EqualFold(account.ResourceGroup, resourceGroup) {
			continue
		}
		items = append(items, azureAutomationAccountFixture(account))
	}
	s.providerStorageMu.Unlock()
	azureAutomationSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAutomationAssetCreateOrUpdate(w http.ResponseWriter, r *http.Request, path string, account *azureAutomationAccount, collection, name string) {
	body, payload, ok := azureAutomationReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !azureAutomationValidateAssetPayload(w, path, collection, payload) {
		return
	}

	now := time.Now().UTC()
	key := azureAutomationAssetKey(account.SubscriptionID, account.ResourceGroup, account.Name, collection, name)

	s.providerStorageMu.Lock()
	current := s.azureAutomationAssets[key]
	if current == nil {
		current = &azureAutomationAsset{
			SubscriptionID: account.SubscriptionID,
			ResourceGroup:  account.ResourceGroup,
			AccountName:    account.Name,
			Collection:     collection,
			Name:           name,
			CreatedAt:      now,
		}
	}
	current.Location = azureAutomationCoalesceString(azureAutomationString(payload["location"]), account.Location)
	current.Tags = azureAutomationStringMap(payload["tags"])
	current.Properties = azureAutomationMap(payload["properties"])
	current.Content = strings.TrimSpace(azureAutomationString(current.Properties["content"]))
	if current.Content == "" && strings.EqualFold(collection, "runbooks") {
		current.Content = fmt.Sprintf("workflow %s {\n    Write-Output \"stackyard\"\n}\n", name)
	}
	if current.DraftContent == "" && strings.EqualFold(collection, "runbooks") {
		current.DraftContent = current.Content + "\n# draft\n"
	}
	if strings.EqualFold(collection, "webhooks") && current.URI == "" {
		current.URI = azureAutomationGenerateWebhookURI(account.SubscriptionID, account.ResourceGroup, account.Name, name)
	}
	if uri := strings.TrimSpace(azureAutomationString(current.Properties["uri"])); uri != "" {
		current.URI = uri
	}
	current.UpdatedAt = now
	current.ETag = azureAutomationETag(key, body, now)
	s.azureAutomationAssets[key] = current
	s.providerStorageMu.Unlock()

	statusCode := http.StatusOK
	if strings.EqualFold(azureAutomationCollectionKind(collection), "jobschedules") {
		statusCode = http.StatusCreated
	}
	respondJSON(w, statusCode, azureAutomationAssetFixture(current))
}

func (s *Server) handleAzureAutomationAssetPatch(w http.ResponseWriter, r *http.Request, path string, account *azureAutomationAccount, collection, name string) {
	body, payload, ok := azureAutomationReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureAutomationAssetKey(account.SubscriptionID, account.ResourceGroup, account.Name, collection, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	current := s.azureAutomationAssets[key]
	if current == nil {
		s.providerStorageMu.Unlock()
		azureAutomationNotFound(w, path, "ResourceNotFound", "automation resource not found")
		return
	}
	if tags, ok := payload["tags"]; ok {
		current.Tags = azureAutomationStringMap(tags)
	}
	if properties, ok := payload["properties"]; ok {
		current.Properties = azureAutomationMergeMaps(current.Properties, azureAutomationMap(properties))
	}
	if strings.EqualFold(collection, "runbooks") {
		if content := strings.TrimSpace(azureAutomationString(current.Properties["content"])); content != "" {
			current.Content = content
		}
	}
	if strings.EqualFold(collection, "webhooks") {
		if uri := strings.TrimSpace(azureAutomationString(current.Properties["uri"])); uri != "" {
			current.URI = uri
		}
	}
	current.UpdatedAt = now
	current.ETag = azureAutomationETag(key, body, now)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureAutomationAssetFixture(current))
}

func (s *Server) handleAzureAutomationAssetGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, collection, name string) {
	key := azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, collection, name)
	s.providerStorageMu.Lock()
	current := s.azureAutomationAssets[key]
	s.providerStorageMu.Unlock()
	if current == nil {
		azureAutomationNotFound(w, path, "ResourceNotFound", "automation resource not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAutomationAssetFixture(current))
}

func (s *Server) handleAzureAutomationAssetDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, collection, name string) {
	key := azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, collection, name)
	prefix := key + "::"

	s.providerStorageMu.Lock()
	delete(s.azureAutomationAssets, key)
	for assetKey := range s.azureAutomationAssets {
		if strings.HasPrefix(assetKey, prefix) {
			delete(s.azureAutomationAssets, assetKey)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureAutomationAssetList(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, collection string) {
	prefix := strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName)) + "::asset::" + strings.ToLower(collection) + "::"
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for key, asset := range s.azureAutomationAssets {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if strings.Count(strings.TrimPrefix(key, prefix), "::") != 0 {
			continue
		}
		items = append(items, azureAutomationAssetFixture(asset))
	}
	s.providerStorageMu.Unlock()
	azureAutomationSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAutomationRunbookCreateOrUpdate(w http.ResponseWriter, r *http.Request, path string, account *azureAutomationAccount, name string) {
	s.handleAzureAutomationAssetCreateOrUpdate(w, r, path, account, "runbooks", name)
}

func (s *Server) handleAzureAutomationRunbookContentGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, name string, draft bool) {
	key := azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, "runbooks", name)
	s.providerStorageMu.Lock()
	current := s.azureAutomationAssets[key]
	s.providerStorageMu.Unlock()
	if current == nil {
		azureAutomationNotFound(w, path, "RunbookNotFound", "runbook not found")
		return
	}
	content := current.Content
	if draft {
		content = current.DraftContent
	}
	if strings.TrimSpace(content) == "" {
		content = fmt.Sprintf("workflow %s {\n    Write-Output \"stackyard\"\n}\n", name)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (s *Server) handleAzureAutomationRunbookDraftGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, name string) {
	key := azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, "runbooks", name)
	s.providerStorageMu.Lock()
	current := s.azureAutomationAssets[key]
	s.providerStorageMu.Unlock()
	if current == nil {
		azureAutomationNotFound(w, path, "RunbookNotFound", "runbook not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":         azureAutomationAssetID(current) + "/draft",
		"name":       "draft",
		"type":       "Microsoft.Automation/automationAccounts/runbooks/draft",
		"properties": map[string]any{"inEdit": true, "draftContentLink": map[string]any{"uri": current.URI}},
	})
}

func (s *Server) handleAzureAutomationRunbookDraftUndoEdit(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, name string) {
	key := azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, "runbooks", name)
	s.providerStorageMu.Lock()
	current := s.azureAutomationAssets[key]
	if current != nil {
		current.DraftContent = current.Content
		current.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	if current == nil {
		azureAutomationNotFound(w, path, "RunbookNotFound", "runbook not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": "Succeeded"})
}

func (s *Server) handleAzureAutomationRunbookPublish(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, name string) {
	key := azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, "runbooks", name)
	s.providerStorageMu.Lock()
	current := s.azureAutomationAssets[key]
	if current != nil {
		current.Content = current.DraftContent
		current.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	if current == nil {
		azureAutomationNotFound(w, path, "RunbookNotFound", "runbook not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": "Succeeded"})
}

func (s *Server) handleAzureAutomationJobCreate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, jobName string) {
	_, payload, ok := azureAutomationReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties := azureAutomationMap(payload["properties"])
	runbookValue := azureAutomationMap(properties["runbook"])
	runbookName := strings.TrimSpace(azureAutomationString(runbookValue["name"]))
	if runbookName == "" {
		respondAzureInvalidRequest(w, path, "properties.runbook.name is required")
		return
	}

	runbookKey := azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, "runbooks", runbookName)
	s.providerStorageMu.Lock()
	runbook := s.azureAutomationAssets[runbookKey]
	now := time.Now().UTC()
	job := &azureAutomationJob{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		AccountName:    accountName,
		Name:           jobName,
		RunbookName:    runbookName,
		Parameters:     azureAutomationMap(properties["parameters"]),
		Status:         "Running",
		Output:         fmt.Sprintf("job %s completed successfully", jobName),
		CreatedAt:      now,
		UpdatedAt:      now,
		StartedAt:      now,
		RunbookContent: fmt.Sprintf("workflow %s {\n    Write-Output \"stackyard\"\n}\n", runbookName),
		Streams: []azureAutomationJobStream{
			{ID: "stream-1", StreamType: "Output", Summary: "Job started", Text: "Job execution started", CreatedAt: now},
			{ID: "stream-2", StreamType: "Output", Summary: "Job completed", Text: fmt.Sprintf("Job %s completed successfully", jobName), CreatedAt: now.Add(1 * time.Second)},
		},
	}
	if runbook != nil && strings.TrimSpace(runbook.Content) != "" {
		job.RunbookContent = runbook.Content
	}
	s.azureAutomationJobs[azureAutomationJobKey(subscriptionID, resourceGroup, accountName, jobName)] = job
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureAutomationJobFixture(job))
}

func (s *Server) handleAzureAutomationJobGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, jobName string) {
	job, ok := s.azureAutomationGetJob(subscriptionID, resourceGroup, accountName, jobName)
	if !ok {
		azureAutomationNotFound(w, path, "JobNotFound", "job not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAutomationJobFixture(job))
}

func (s *Server) handleAzureAutomationJobList(w http.ResponseWriter, subscriptionID, resourceGroup, accountName string) {
	prefix := strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName)) + "::job::"
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for key, job := range s.azureAutomationJobs {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		items = append(items, azureAutomationJobFixture(job))
	}
	s.providerStorageMu.Unlock()
	azureAutomationSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAutomationJobAction(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, jobName, status string) {
	s.providerStorageMu.Lock()
	job := s.azureAutomationJobs[azureAutomationJobKey(subscriptionID, resourceGroup, accountName, jobName)]
	if job != nil {
		job.Status = status
		job.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	if job == nil {
		azureAutomationNotFound(w, path, "JobNotFound", "job not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAutomationJobFixture(job))
}

func (s *Server) handleAzureAutomationJobOutput(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, jobName string) {
	job, ok := s.azureAutomationGetJob(subscriptionID, resourceGroup, accountName, jobName)
	if !ok {
		azureAutomationNotFound(w, path, "JobNotFound", "job not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(job.Output))
}

func (s *Server) handleAzureAutomationJobRunbookContent(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, jobName string) {
	job, ok := s.azureAutomationGetJob(subscriptionID, resourceGroup, accountName, jobName)
	if !ok {
		azureAutomationNotFound(w, path, "JobNotFound", "job not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(job.RunbookContent))
}

func (s *Server) handleAzureAutomationJobStreamList(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, jobName string) {
	job, ok := s.azureAutomationGetJob(subscriptionID, resourceGroup, accountName, jobName)
	if !ok {
		azureAutomationNotFound(w, path, "JobNotFound", "job not found")
		return
	}
	items := make([]map[string]any, 0, len(job.Streams))
	for _, stream := range job.Streams {
		items = append(items, azureAutomationJobStreamFixture(subscriptionID, resourceGroup, accountName, jobName, stream))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAutomationJobStreamGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, jobName, streamID string) {
	job, ok := s.azureAutomationGetJob(subscriptionID, resourceGroup, accountName, jobName)
	if !ok {
		azureAutomationNotFound(w, path, "JobNotFound", "job not found")
		return
	}
	for _, stream := range job.Streams {
		if strings.EqualFold(stream.ID, streamID) {
			respondJSON(w, http.StatusOK, azureAutomationJobStreamFixture(subscriptionID, resourceGroup, accountName, jobName, stream))
			return
		}
	}
	azureAutomationNotFound(w, path, "JobStreamNotFound", "job stream not found")
}

func (s *Server) handleAzureAutomationSourceControlDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, sourceControlName string) {
	s.handleAzureAutomationAssetDelete(w, subscriptionID, resourceGroup, accountName, "sourceControls", sourceControlName)
	prefix := strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName)) + "::sourcecontrol::" + strings.ToLower(sourceControlName) + "::"
	s.providerStorageMu.Lock()
	for key := range s.azureAutomationSyncJobs {
		if strings.HasPrefix(key, prefix) {
			delete(s.azureAutomationSyncJobs, key)
		}
	}
	s.providerStorageMu.Unlock()
}

func (s *Server) handleAzureAutomationSyncJobCreate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, sourceControlName, name string) {
	_, payload, ok := azureAutomationReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureAutomationSyncJobKey(subscriptionID, resourceGroup, accountName, sourceControlName, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	job := &azureAutomationSyncJob{
		SubscriptionID:    subscriptionID,
		ResourceGroup:     resourceGroup,
		AccountName:       accountName,
		SourceControlName: sourceControlName,
		Name:              name,
		Properties:        azureAutomationMap(payload["properties"]),
		CreatedAt:         now,
		UpdatedAt:         now,
		Streams: []azureAutomationSyncJobStream{
			{ID: "stream-1", StreamType: "Output", Summary: "Sync job started", Text: "Synchronization started", CreatedAt: now},
			{ID: "stream-2", StreamType: "Output", Summary: "Sync job completed", Text: "Synchronization completed", CreatedAt: now.Add(1 * time.Second)},
		},
	}
	s.azureAutomationSyncJobs[key] = job
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureAutomationSyncJobFixture(job))
}

func (s *Server) handleAzureAutomationSyncJobGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, sourceControlName, name string) {
	job, ok := s.azureAutomationGetSyncJob(subscriptionID, resourceGroup, accountName, sourceControlName, name)
	if !ok {
		azureAutomationNotFound(w, path, "SourceControlSyncJobNotFound", "source control sync job not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAutomationSyncJobFixture(job))
}

func (s *Server) handleAzureAutomationSyncJobList(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, sourceControlName string) {
	prefix := strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName)) + "::sourcecontrol::" + strings.ToLower(sourceControlName) + "::"
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for key, job := range s.azureAutomationSyncJobs {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		items = append(items, azureAutomationSyncJobFixture(job))
	}
	s.providerStorageMu.Unlock()
	azureAutomationSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAutomationSyncJobStreamList(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName string) {
	job, ok := s.azureAutomationGetSyncJob(subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName)
	if !ok {
		azureAutomationNotFound(w, path, "SourceControlSyncJobNotFound", "source control sync job not found")
		return
	}
	items := make([]map[string]any, 0, len(job.Streams))
	for _, stream := range job.Streams {
		items = append(items, azureAutomationSyncJobStreamFixture(subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName, stream))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAutomationSyncJobStreamGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName, streamID string) {
	job, ok := s.azureAutomationGetSyncJob(subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName)
	if !ok {
		azureAutomationNotFound(w, path, "SourceControlSyncJobNotFound", "source control sync job not found")
		return
	}
	for _, stream := range job.Streams {
		if strings.EqualFold(stream.ID, streamID) {
			respondJSON(w, http.StatusOK, azureAutomationSyncJobStreamFixture(subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName, stream))
			return
		}
	}
	azureAutomationNotFound(w, path, "SourceControlSyncJobStreamNotFound", "source control sync job stream not found")
}

func (s *Server) handleAzureAutomationStatisticsList(w http.ResponseWriter, subscriptionID, resourceGroup, accountName string) {
	accountKey := strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName))

	s.providerStorageMu.Lock()
	var runbookCount, variableCount, jobCount int
	for key, asset := range s.azureAutomationAssets {
		if !strings.HasPrefix(key, accountKey+"::asset::") {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(asset.Collection)) {
		case "runbooks":
			runbookCount++
		case "variables":
			variableCount++
		}
	}
	for key := range s.azureAutomationJobs {
		if strings.HasPrefix(key, accountKey+"::job::") {
			jobCount++
		}
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"value": []map[string]any{
			{"name": "Runbooks", "value": runbookCount},
			{"name": "Variables", "value": variableCount},
			{"name": "Jobs", "value": jobCount},
		},
	})
}

func azureIsAutomationPath(segments []string) bool {
	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Automation") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	if len(segments) < 2 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Automation") {
			return true
		}
	}
	return false
}

func azureAutomationRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.Automation") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureAutomationAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureAutomationReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	payload := map[string]any{}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, payload, true
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	return body, payload, true
}

func (s *Server) azureAutomationGetAccount(subscriptionID, resourceGroup, accountName string) (*azureAutomationAccount, bool) {
	s.providerStorageMu.Lock()
	account := s.azureAutomationAccounts[azureAutomationAccountKey(subscriptionID, resourceGroup, accountName)]
	s.providerStorageMu.Unlock()
	return account, account != nil
}

func (s *Server) azureAutomationGetJob(subscriptionID, resourceGroup, accountName, jobName string) (*azureAutomationJob, bool) {
	s.providerStorageMu.Lock()
	job := s.azureAutomationJobs[azureAutomationJobKey(subscriptionID, resourceGroup, accountName, jobName)]
	s.providerStorageMu.Unlock()
	return job, job != nil
}

func (s *Server) azureAutomationGetSyncJob(subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName string) (*azureAutomationSyncJob, bool) {
	s.providerStorageMu.Lock()
	job := s.azureAutomationSyncJobs[azureAutomationSyncJobKey(subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName)]
	s.providerStorageMu.Unlock()
	return job, job != nil
}

func azureAutomationValidateAssetPayload(w http.ResponseWriter, path, collection string, payload map[string]any) bool {
	properties := azureAutomationMap(payload["properties"])
	switch strings.ToLower(azureAutomationCollectionKind(collection)) {
	case "certificates":
		if strings.TrimSpace(azureAutomationString(properties["base64Value"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.base64Value is required")
			return false
		}
	case "credentials":
		if strings.TrimSpace(azureAutomationString(properties["userName"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.userName is required")
			return false
		}
	case "variables":
		if _, ok := properties["value"]; !ok {
			respondAzureInvalidRequest(w, path, "properties.value is required")
			return false
		}
	case "schedules":
		if strings.TrimSpace(azureAutomationString(properties["startTime"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.startTime is required")
			return false
		}
	case "jobschedules":
		if azureAutomationMap(properties["schedule"])["name"] == nil {
			respondAzureInvalidRequest(w, path, "properties.schedule.name is required")
			return false
		}
		if azureAutomationMap(properties["runbook"])["name"] == nil {
			respondAzureInvalidRequest(w, path, "properties.runbook.name is required")
			return false
		}
	case "webhooks":
		runbook := azureAutomationMap(properties["runbook"])
		if strings.TrimSpace(azureAutomationString(runbook["name"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.runbook.name is required")
			return false
		}
	case "sourcecontrols":
		if strings.TrimSpace(azureAutomationString(properties["repoUrl"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.repoUrl is required")
			return false
		}
	case "privateendpointconnections":
		privateEndpoint := azureAutomationMap(properties["privateEndpoint"])
		if strings.TrimSpace(azureAutomationString(privateEndpoint["id"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.privateEndpoint.id is required")
			return false
		}
	}
	return true
}

func azureAutomationCollectionKind(collection string) string {
	parts := splitPathSegments(collection)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func azureAutomationAccountKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + name)
}

func azureAutomationAssetKey(subscriptionID, resourceGroup, accountName, collection, name string) string {
	return strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName) + "::asset::" + collection + "::" + name)
}

func azureAutomationJobKey(subscriptionID, resourceGroup, accountName, jobName string) string {
	return strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName) + "::job::" + jobName)
}

func azureAutomationSyncJobKey(subscriptionID, resourceGroup, accountName, sourceControlName, name string) string {
	return strings.ToLower(azureAutomationAccountKey(subscriptionID, resourceGroup, accountName) + "::sourcecontrol::" + sourceControlName + "::syncjob::" + name)
}

func azureAutomationAccountID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Automation/automationAccounts/%s", subscriptionID, resourceGroup, name)
}

func azureAutomationAssetID(asset *azureAutomationAsset) string {
	return azureAutomationAccountID(asset.SubscriptionID, asset.ResourceGroup, asset.AccountName) + "/" + asset.Collection + "/" + asset.Name
}

func azureAutomationJobID(job *azureAutomationJob) string {
	return azureAutomationAccountID(job.SubscriptionID, job.ResourceGroup, job.AccountName) + "/jobs/" + job.Name
}

func azureAutomationSyncJobID(job *azureAutomationSyncJob) string {
	return azureAutomationAccountID(job.SubscriptionID, job.ResourceGroup, job.AccountName) + "/sourceControls/" + job.SourceControlName + "/sourceControlSyncJobs/" + job.Name
}

func azureAutomationETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+now.UTC().Format(time.RFC3339Nano)), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureAutomationAccountFixture(account *azureAutomationAccount) map[string]any {
	properties := azureAutomationMergeMaps(map[string]any{
		"state":               "Ok",
		"creationTime":        account.CreatedAt.Format(time.RFC3339),
		"lastModifiedTime":    account.UpdatedAt.Format(time.RFC3339),
		"lastModifiedBy":      "stackyard",
		"publicNetworkAccess": "Enabled",
		"disableLocalAuth":    false,
	}, account.Properties)
	if len(account.SKU) > 0 {
		properties["sku"] = account.SKU
	}
	return map[string]any{
		"id":         azureAutomationAccountID(account.SubscriptionID, account.ResourceGroup, account.Name),
		"name":       account.Name,
		"type":       "Microsoft.Automation/automationAccounts",
		"location":   account.Location,
		"tags":       azureAutomationAnyMap(account.Tags),
		"identity":   account.Identity,
		"etag":       account.ETag,
		"properties": properties,
	}
}

func azureAutomationAssetFixture(asset *azureAutomationAsset) map[string]any {
	properties := azureAutomationAssetPropertiesFixture(asset)
	return map[string]any{
		"id":         azureAutomationAssetID(asset),
		"name":       asset.Name,
		"type":       "Microsoft.Automation/automationAccounts/" + asset.Collection,
		"location":   asset.Location,
		"tags":       azureAutomationAnyMap(asset.Tags),
		"etag":       asset.ETag,
		"properties": properties,
	}
}

func azureAutomationAssetPropertiesFixture(asset *azureAutomationAsset) map[string]any {
	base := azureAutomationMergeMaps(map[string]any{
		"creationTime":     asset.CreatedAt.Format(time.RFC3339),
		"lastModifiedTime": asset.UpdatedAt.Format(time.RFC3339),
		"description":      fmt.Sprintf("stackyard %s %s", azureAutomationCollectionKind(asset.Collection), asset.Name),
	}, asset.Properties)

	switch strings.ToLower(azureAutomationCollectionKind(asset.Collection)) {
	case "certificates":
		base["thumbprint"] = azureAutomationThumbprint(asset.Name)
		base["isExportable"] = true
	case "credentials":
		base["userName"] = azureAutomationCoalesceString(azureAutomationString(base["userName"]), asset.Name+"-user")
	case "variables":
		if _, ok := base["isEncrypted"]; !ok {
			base["isEncrypted"] = false
		}
	case "schedules":
		base["frequency"] = azureAutomationCoalesceString(azureAutomationString(base["frequency"]), "Hour")
		base["timeZone"] = azureAutomationCoalesceString(azureAutomationString(base["timeZone"]), "UTC")
	case "jobSchedules":
		if _, ok := base["jobScheduleId"]; !ok {
			base["jobScheduleId"] = asset.Name
		}
	case "runbooks":
		base["runbookType"] = azureAutomationCoalesceString(azureAutomationString(base["runbookType"]), "PowerShellWorkflow")
		base["state"] = azureAutomationCoalesceString(azureAutomationString(base["state"]), "Published")
		base["logVerbose"] = azureAutomationBool(base["logVerbose"])
		base["logProgress"] = azureAutomationBool(base["logProgress"])
	case "webhooks":
		base["uri"] = asset.URI
		base["isEnabled"] = true
		base["expiryTime"] = asset.UpdatedAt.Add(24 * time.Hour).Format(time.RFC3339)
	case "sourcecontrols":
		base["sourceType"] = azureAutomationCoalesceString(azureAutomationString(base["sourceType"]), "GitHub")
		base["autoSync"] = true
		base["folderPath"] = azureAutomationCoalesceString(azureAutomationString(base["folderPath"]), "/")
	case "runtimeenvironments":
		base["runtime"] = azureAutomationMergeMaps(map[string]any{"language": "PowerShell", "version": "7.4"}, azureAutomationMap(base["runtime"]))
	case "packages", "python3packages":
		base["provisioningState"] = "Succeeded"
	case "privateendpointconnections":
		state := azureAutomationMergeMaps(map[string]any{"status": "Approved", "description": "stackyard approved"}, azureAutomationMap(base["privateLinkServiceConnectionState"]))
		base["privateLinkServiceConnectionState"] = state
	}
	return base
}

func azureAutomationJobFixture(job *azureAutomationJob) map[string]any {
	return map[string]any{
		"id":       azureAutomationJobID(job),
		"name":     job.Name,
		"type":     "Microsoft.Automation/automationAccounts/jobs",
		"location": "eastus",
		"properties": map[string]any{
			"runbook":           map[string]any{"name": job.RunbookName},
			"status":            job.Status,
			"creationTime":      job.CreatedAt.Format(time.RFC3339),
			"startTime":         job.StartedAt.Format(time.RFC3339),
			"lastModifiedTime":  job.UpdatedAt.Format(time.RFC3339),
			"provisioningState": "Succeeded",
			"parameters":        job.Parameters,
		},
	}
}

func azureAutomationJobStreamFixture(subscriptionID, resourceGroup, accountName, jobName string, stream azureAutomationJobStream) map[string]any {
	return map[string]any{
		"id":   azureAutomationAccountID(subscriptionID, resourceGroup, accountName) + "/jobs/" + jobName + "/streams/" + stream.ID,
		"name": stream.ID,
		"type": "Microsoft.Automation/automationAccounts/jobs/streams",
		"properties": map[string]any{
			"streamType": stream.StreamType,
			"summary":    stream.Summary,
			"streamText": stream.Text,
			"time":       stream.CreatedAt.Format(time.RFC3339),
		},
	}
}

func azureAutomationSyncJobFixture(job *azureAutomationSyncJob) map[string]any {
	return map[string]any{
		"id":   azureAutomationSyncJobID(job),
		"name": job.Name,
		"type": "Microsoft.Automation/automationAccounts/sourceControls/sourceControlSyncJobs",
		"properties": azureAutomationMergeMaps(map[string]any{
			"provisioningState": "Succeeded",
			"syncType":          "FullSync",
			"creationTime":      job.CreatedAt.Format(time.RFC3339),
			"lastModifiedTime":  job.UpdatedAt.Format(time.RFC3339),
		}, job.Properties),
	}
}

func azureAutomationSyncJobStreamFixture(subscriptionID, resourceGroup, accountName, sourceControlName, syncJobName string, stream azureAutomationSyncJobStream) map[string]any {
	return map[string]any{
		"id":   azureAutomationAccountID(subscriptionID, resourceGroup, accountName) + "/sourceControls/" + sourceControlName + "/sourceControlSyncJobs/" + syncJobName + "/streams/" + stream.ID,
		"name": stream.ID,
		"type": "Microsoft.Automation/automationAccounts/sourceControls/sourceControlSyncJobs/streams",
		"properties": map[string]any{
			"streamType": stream.StreamType,
			"summary":    stream.Summary,
			"streamText": stream.Text,
			"time":       stream.CreatedAt.Format(time.RFC3339),
		},
	}
}

func azureAutomationUsagesFixture(subscriptionID, resourceGroup, accountName string) []map[string]any {
	accountID := azureAutomationAccountID(subscriptionID, resourceGroup, accountName)
	return []map[string]any{
		{
			"id":           accountID + "/usages/jobs",
			"name":         map[string]any{"value": "jobs", "localizedValue": "Jobs"},
			"unit":         "Count",
			"currentValue": 3,
			"limit":        1000,
		},
		{
			"id":           accountID + "/usages/runbooks",
			"name":         map[string]any{"value": "runbooks", "localizedValue": "Runbooks"},
			"unit":         "Count",
			"currentValue": 2,
			"limit":        100,
		},
	}
}

func azureAutomationPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName string) []map[string]any {
	return []map[string]any{
		{
			"id":   azureAutomationAccountID(subscriptionID, resourceGroup, accountName) + "/privateLinkResources/automationAccount",
			"name": "automationAccount",
			"type": "Microsoft.Automation/automationAccounts/privateLinkResources",
			"properties": map[string]any{
				"groupId":           "automationAccount",
				"requiredMembers":   []string{"automationAccount"},
				"requiredZoneNames": []string{"privatelink.azure-automation.net"},
				"displayName":       "Automation Account",
			},
		},
	}
}

func azureAutomationOperationsFixture() []map[string]any {
	return []map[string]any{
		azureAutomationOperation("Microsoft.Automation/automationAccounts/read", "Read Automation Account"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/write", "Create or Update Automation Account"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/delete", "Delete Automation Account"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/runbooks/read", "Read Runbook"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/runbooks/write", "Create or Update Runbook"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/jobs/read", "Read Job"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/jobs/write", "Create Job"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/variables/read", "Read Variable"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/variables/write", "Create or Update Variable"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/webhooks/read", "Read Webhook"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/webhooks/write", "Create or Update Webhook"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/sourceControls/read", "Read Source Control"),
		azureAutomationOperation("Microsoft.Automation/automationAccounts/sourceControls/write", "Create or Update Source Control"),
	}
}

func azureAutomationOperation(name, display string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":  "Microsoft.Automation",
			"resource":  "Automation",
			"operation": display,
		},
	}
}

func azureAutomationGenerateWebhookURI(subscriptionID, resourceGroup, accountName, webhookName string) string {
	name := webhookName
	if strings.TrimSpace(name) == "" {
		name = "generated"
	}
	return fmt.Sprintf("https://stackyard.local/azure/automation/%s/%s/%s/%s", subscriptionID, resourceGroup, accountName, name)
}

func azureAutomationNotFound(w http.ResponseWriter, path, code, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerAzure,
		"path":     path,
	})
}

func azureAutomationAnyMap(tags map[string]string) map[string]any {
	if len(tags) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func azureAutomationSortByID(items []map[string]any) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureAutomationString(items[i]["id"])) < strings.ToLower(azureAutomationString(items[j]["id"]))
	})
}

func azureAutomationString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func azureAutomationBool(value any) bool {
	boolean, ok := value.(bool)
	return ok && boolean
}

func azureAutomationMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if existing, ok := value.(map[string]any); ok {
		return existing
	}
	return map[string]any{}
}

func azureAutomationStringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	if existing, ok := value.(map[string]string); ok {
		return existing
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, item := range raw {
		if str, ok := item.(string); ok {
			out[key] = str
		}
	}
	return out
}

func azureAutomationMergeMaps(base, overlay map[string]any) map[string]any {
	if len(base) == 0 && len(overlay) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func azureAutomationCoalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func azureAutomationThumbprint(name string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(name))))
	return strings.ToUpper(hex.EncodeToString(sum[:8]))
}
