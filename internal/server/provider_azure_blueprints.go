package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureBlueprintsAPIVersion = "2018-11-01-preview"

type azureBlueprintDefinition struct {
	Scope      string
	Name       string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureBlueprintArtifact struct {
	Scope         string
	BlueprintName string
	Name          string
	Kind          string
	Properties    map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ETag          string
}

type azureBlueprintVersion struct {
	Scope         string
	BlueprintName string
	Name          string
	Properties    map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ETag          string
}

type azureBlueprintPublishedArtifact struct {
	Scope         string
	BlueprintName string
	VersionName   string
	Name          string
	Kind          string
	Properties    map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ETag          string
}

type azureBlueprintAssignment struct {
	Scope      string
	Name       string
	Location   string
	Identity   map[string]any
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureBlueprintAssignmentOperation struct {
	Scope          string
	AssignmentName string
	Name           string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Server) handleAzureBlueprintsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	providerIndex := azureBlueprintsProviderIndex(segments)
	if providerIndex < 0 {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureBlueprintsAllowedMethod(r.Method) {
		return false
	}

	scope, ok := azureBlueprintsScopeFromSegments(segments[:providerIndex])
	if !ok {
		return false
	}
	tail := segments[providerIndex+2:]
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "blueprints":
		s.handleAzureBlueprintDefinitionsRoute(w, r, path, scope, tail)
	case "blueprintassignments":
		s.handleAzureBlueprintAssignmentsRoute(w, r, path, scope, tail)
	default:
		respondAzureImplemented(w, path)
	}
	return true
}

func azureBlueprintsProviderIndex(segments []string) int {
	last := -1
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Blueprint") {
			last = idx
		}
	}
	return last
}

func azureBlueprintsScopeFromSegments(segments []string) (string, bool) {
	switch {
	case len(segments) == 2 && strings.EqualFold(segments[0], "subscriptions"):
		subscriptionID := strings.TrimSpace(segments[1])
		if subscriptionID == "" {
			return "", false
		}
		return "/subscriptions/" + subscriptionID, true
	case len(segments) == 2 && strings.EqualFold(segments[0], "managementGroups"):
		managementGroup := strings.TrimSpace(segments[1])
		if managementGroup == "" {
			return "", false
		}
		return "/providers/Microsoft.Management/managementGroups/" + managementGroup, true
	case len(segments) == 4 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Management") &&
		strings.EqualFold(segments[2], "managementGroups"):
		managementGroup := strings.TrimSpace(segments[3])
		if managementGroup == "" {
			return "", false
		}
		return "/providers/Microsoft.Management/managementGroups/" + managementGroup, true
	default:
		return "", false
	}
}

func azureBlueprintsAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureBlueprintDefinitionsRoute(w http.ResponseWriter, r *http.Request, path, scope string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBlueprintDefinitionsList(w, scope)
		return
	}

	blueprintName := strings.TrimSpace(tail[1])
	if blueprintName == "" {
		respondAzureInvalidRequest(w, path, "blueprint name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBlueprintDefinitionGet(w, path, scope, blueprintName, false)
		case http.MethodHead:
			s.handleAzureBlueprintDefinitionGet(w, path, scope, blueprintName, true)
		case http.MethodPut:
			s.handleAzureBlueprintDefinitionCreateOrUpdate(w, r, path, scope, blueprintName)
		case http.MethodDelete:
			s.handleAzureBlueprintDefinitionDelete(w, path, scope, blueprintName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "artifacts":
		s.handleAzureBlueprintArtifactsRoute(w, r, path, scope, blueprintName, tail[3:])
	case "versions":
		s.handleAzureBlueprintVersionsRoute(w, r, path, scope, blueprintName, tail[3:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureBlueprintDefinitionsList(w http.ResponseWriter, scope string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, blueprint := range s.azureBlueprintDefinitions {
		if !strings.EqualFold(blueprint.Scope, scope) {
			continue
		}
		items = append(items, azureBlueprintDefinitionFixture(blueprint))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureBlueprintsString(items[i]["id"])) < strings.ToLower(azureBlueprintsString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureBlueprintDefinitionCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, scope, blueprintName string) {
	body, payload, ok := azureBlueprintsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties := azureBlueprintsMap(payload["properties"])
	targetScope := strings.TrimSpace(azureBlueprintsString(properties["targetScope"]))
	if targetScope == "" {
		targetScope = "subscription"
	}
	if targetScope != "subscription" && targetScope != "managementGroup" {
		respondAzureInvalidRequest(w, path, "properties.targetScope must be subscription or managementGroup")
		return
	}

	now := time.Now().UTC()
	key := azureBlueprintsStorageKey(azureBlueprintDefinitionID(scope, blueprintName))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	record := s.azureBlueprintDefinitions[key]
	status := http.StatusOK
	createdAt := now
	if record == nil {
		record = &azureBlueprintDefinition{
			Scope: scope,
			Name:  blueprintName,
		}
		status = http.StatusCreated
	} else {
		createdAt = record.CreatedAt
	}
	record.Properties = azureBlueprintsNormalizeDefinitionProperties(blueprintName, properties, createdAt, now)
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	record.ETag = azureBlueprintsETag(key, body, now)
	s.azureBlueprintDefinitions[key] = record

	respondJSON(w, status, azureBlueprintDefinitionFixture(record))
}

func (s *Server) handleAzureBlueprintDefinitionGet(w http.ResponseWriter, path, scope, blueprintName string, headOnly bool) {
	s.providerStorageMu.Lock()
	record := s.azureBlueprintDefinitions[azureBlueprintsStorageKey(azureBlueprintDefinitionID(scope, blueprintName))]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondAzureBlueprintNotFound(w, path, "blueprint definition not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureBlueprintDefinitionFixture(record))
}

func (s *Server) handleAzureBlueprintDefinitionDelete(w http.ResponseWriter, path, scope, blueprintName string) {
	id := azureBlueprintDefinitionID(scope, blueprintName)
	key := azureBlueprintsStorageKey(id)

	s.providerStorageMu.Lock()
	record := s.azureBlueprintDefinitions[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		respondAzureBlueprintNotFound(w, path, "blueprint definition not found")
		return
	}
	delete(s.azureBlueprintDefinitions, key)
	for artifactKey, artifact := range s.azureBlueprintArtifacts {
		if strings.EqualFold(artifact.Scope, scope) && strings.EqualFold(artifact.BlueprintName, blueprintName) {
			delete(s.azureBlueprintArtifacts, artifactKey)
		}
	}
	for versionKey, version := range s.azureBlueprintVersions {
		if strings.EqualFold(version.Scope, scope) && strings.EqualFold(version.BlueprintName, blueprintName) {
			delete(s.azureBlueprintVersions, versionKey)
		}
	}
	for publishedArtifactKey, artifact := range s.azureBlueprintPublishedArtifacts {
		if strings.EqualFold(artifact.Scope, scope) && strings.EqualFold(artifact.BlueprintName, blueprintName) {
			delete(s.azureBlueprintPublishedArtifacts, publishedArtifactKey)
		}
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureBlueprintDefinitionFixture(record))
}

func (s *Server) handleAzureBlueprintArtifactsRoute(w http.ResponseWriter, r *http.Request, path, scope, blueprintName string, tail []string) {
	if !s.azureBlueprintDefinitionExists(scope, blueprintName) {
		respondAzureBlueprintNotFound(w, path, "blueprint definition not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBlueprintArtifactsList(w, scope, blueprintName)
		return
	}

	artifactName := strings.TrimSpace(tail[0])
	if artifactName == "" {
		respondAzureInvalidRequest(w, path, "artifact name is required")
		return
	}
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureBlueprintArtifactGet(w, path, scope, blueprintName, artifactName)
	case http.MethodPut:
		s.handleAzureBlueprintArtifactCreateOrUpdate(w, r, path, scope, blueprintName, artifactName)
	case http.MethodDelete:
		s.handleAzureBlueprintArtifactDelete(w, path, scope, blueprintName, artifactName)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureBlueprintArtifactsList(w http.ResponseWriter, scope, blueprintName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, artifact := range s.azureBlueprintArtifacts {
		if !strings.EqualFold(artifact.Scope, scope) || !strings.EqualFold(artifact.BlueprintName, blueprintName) {
			continue
		}
		items = append(items, azureBlueprintArtifactFixture(artifact))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureBlueprintsString(items[i]["id"])) < strings.ToLower(azureBlueprintsString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureBlueprintArtifactCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, scope, blueprintName, artifactName string) {
	body, payload, ok := azureBlueprintsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	kind := strings.TrimSpace(azureBlueprintsString(payload["kind"]))
	if kind == "" {
		kind = "template"
	}
	properties := azureBlueprintsMap(payload["properties"])
	if len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "artifact properties are required")
		return
	}

	now := time.Now().UTC()
	key := azureBlueprintsStorageKey(azureBlueprintArtifactID(scope, blueprintName, artifactName))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	record := s.azureBlueprintArtifacts[key]
	status := http.StatusOK
	createdAt := now
	if record == nil {
		record = &azureBlueprintArtifact{
			Scope:         scope,
			BlueprintName: blueprintName,
			Name:          artifactName,
		}
		status = http.StatusCreated
	} else {
		createdAt = record.CreatedAt
	}
	record.Kind = kind
	record.Properties = azureBlueprintsCloneMap(properties)
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	record.ETag = azureBlueprintsETag(key, body, now)
	s.azureBlueprintArtifacts[key] = record

	respondJSON(w, status, azureBlueprintArtifactFixture(record))
}

func (s *Server) handleAzureBlueprintArtifactGet(w http.ResponseWriter, path, scope, blueprintName, artifactName string) {
	s.providerStorageMu.Lock()
	record := s.azureBlueprintArtifacts[azureBlueprintsStorageKey(azureBlueprintArtifactID(scope, blueprintName, artifactName))]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondAzureBlueprintNotFound(w, path, "blueprint artifact not found")
		return
	}
	respondJSON(w, http.StatusOK, azureBlueprintArtifactFixture(record))
}

func (s *Server) handleAzureBlueprintArtifactDelete(w http.ResponseWriter, path, scope, blueprintName, artifactName string) {
	key := azureBlueprintsStorageKey(azureBlueprintArtifactID(scope, blueprintName, artifactName))

	s.providerStorageMu.Lock()
	record := s.azureBlueprintArtifacts[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		respondAzureBlueprintNotFound(w, path, "blueprint artifact not found")
		return
	}
	delete(s.azureBlueprintArtifacts, key)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureBlueprintArtifactFixture(record))
}

func (s *Server) handleAzureBlueprintVersionsRoute(w http.ResponseWriter, r *http.Request, path, scope, blueprintName string, tail []string) {
	if !s.azureBlueprintDefinitionExists(scope, blueprintName) {
		respondAzureBlueprintNotFound(w, path, "blueprint definition not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBlueprintVersionsList(w, scope, blueprintName)
		return
	}

	versionName := strings.TrimSpace(tail[0])
	if versionName == "" {
		respondAzureInvalidRequest(w, path, "version id is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBlueprintVersionGet(w, path, scope, blueprintName, versionName)
		case http.MethodPut:
			s.handleAzureBlueprintVersionCreateOrUpdate(w, r, path, scope, blueprintName, versionName)
		case http.MethodDelete:
			s.handleAzureBlueprintVersionDelete(w, path, scope, blueprintName, versionName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) >= 2 && strings.EqualFold(tail[1], "artifacts") {
		s.handleAzureBlueprintPublishedArtifactsRoute(w, r, path, scope, blueprintName, versionName, tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureBlueprintVersionsList(w http.ResponseWriter, scope, blueprintName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, version := range s.azureBlueprintVersions {
		if !strings.EqualFold(version.Scope, scope) || !strings.EqualFold(version.BlueprintName, blueprintName) {
			continue
		}
		items = append(items, azureBlueprintVersionFixture(version))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureBlueprintsString(items[i]["id"])) < strings.ToLower(azureBlueprintsString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureBlueprintVersionCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, scope, blueprintName, versionName string) {
	body, payload, ok := azureBlueprintsReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	definition := s.azureBlueprintDefinitions[azureBlueprintsStorageKey(azureBlueprintDefinitionID(scope, blueprintName))]
	if definition == nil {
		respondAzureBlueprintNotFound(w, path, "blueprint definition not found")
		return
	}

	now := time.Now().UTC()
	key := azureBlueprintsStorageKey(azureBlueprintVersionID(scope, blueprintName, versionName))
	record := s.azureBlueprintVersions[key]
	status := http.StatusOK
	createdAt := now
	if record == nil {
		record = &azureBlueprintVersion{
			Scope:         scope,
			BlueprintName: blueprintName,
			Name:          versionName,
		}
		status = http.StatusCreated
	} else {
		createdAt = record.CreatedAt
	}
	record.Properties = azureBlueprintsNormalizeVersionProperties(blueprintName, versionName, definition.Properties, azureBlueprintsMap(payload), createdAt, now)
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	record.ETag = azureBlueprintsETag(key, body, now)
	s.azureBlueprintVersions[key] = record

	for _, artifact := range s.azureBlueprintArtifacts {
		if !strings.EqualFold(artifact.Scope, scope) || !strings.EqualFold(artifact.BlueprintName, blueprintName) {
			continue
		}
		publishedKey := azureBlueprintsStorageKey(azureBlueprintPublishedArtifactID(scope, blueprintName, versionName, artifact.Name))
		s.azureBlueprintPublishedArtifacts[publishedKey] = &azureBlueprintPublishedArtifact{
			Scope:         scope,
			BlueprintName: blueprintName,
			VersionName:   versionName,
			Name:          artifact.Name,
			Kind:          artifact.Kind,
			Properties:    azureBlueprintsCloneMap(artifact.Properties),
			CreatedAt:     createdAt,
			UpdatedAt:     now,
			ETag:          azureBlueprintsETag(publishedKey, body, now),
		}
	}

	respondJSON(w, status, azureBlueprintVersionFixture(record))
}

func (s *Server) handleAzureBlueprintVersionGet(w http.ResponseWriter, path, scope, blueprintName, versionName string) {
	s.providerStorageMu.Lock()
	record := s.azureBlueprintVersions[azureBlueprintsStorageKey(azureBlueprintVersionID(scope, blueprintName, versionName))]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondAzureBlueprintNotFound(w, path, "published blueprint not found")
		return
	}
	respondJSON(w, http.StatusOK, azureBlueprintVersionFixture(record))
}

func (s *Server) handleAzureBlueprintVersionDelete(w http.ResponseWriter, path, scope, blueprintName, versionName string) {
	key := azureBlueprintsStorageKey(azureBlueprintVersionID(scope, blueprintName, versionName))

	s.providerStorageMu.Lock()
	record := s.azureBlueprintVersions[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		respondAzureBlueprintNotFound(w, path, "published blueprint not found")
		return
	}
	delete(s.azureBlueprintVersions, key)
	for publishedArtifactKey, artifact := range s.azureBlueprintPublishedArtifacts {
		if strings.EqualFold(artifact.Scope, scope) &&
			strings.EqualFold(artifact.BlueprintName, blueprintName) &&
			strings.EqualFold(artifact.VersionName, versionName) {
			delete(s.azureBlueprintPublishedArtifacts, publishedArtifactKey)
		}
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureBlueprintVersionFixture(record))
}

func (s *Server) handleAzureBlueprintPublishedArtifactsRoute(w http.ResponseWriter, r *http.Request, path, scope, blueprintName, versionName string, tail []string) {
	if !s.azureBlueprintVersionExists(scope, blueprintName, versionName) {
		respondAzureBlueprintNotFound(w, path, "published blueprint not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBlueprintPublishedArtifactsList(w, scope, blueprintName, versionName)
		return
	}

	artifactName := strings.TrimSpace(tail[0])
	if artifactName == "" {
		respondAzureInvalidRequest(w, path, "artifact name is required")
		return
	}
	if len(tail) != 1 || r.Method != http.MethodGet {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureBlueprintPublishedArtifactGet(w, path, scope, blueprintName, versionName, artifactName)
}

func (s *Server) handleAzureBlueprintPublishedArtifactsList(w http.ResponseWriter, scope, blueprintName, versionName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, artifact := range s.azureBlueprintPublishedArtifacts {
		if !strings.EqualFold(artifact.Scope, scope) ||
			!strings.EqualFold(artifact.BlueprintName, blueprintName) ||
			!strings.EqualFold(artifact.VersionName, versionName) {
			continue
		}
		items = append(items, azureBlueprintPublishedArtifactFixture(artifact))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureBlueprintsString(items[i]["id"])) < strings.ToLower(azureBlueprintsString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureBlueprintPublishedArtifactGet(w http.ResponseWriter, path, scope, blueprintName, versionName, artifactName string) {
	s.providerStorageMu.Lock()
	record := s.azureBlueprintPublishedArtifacts[azureBlueprintsStorageKey(azureBlueprintPublishedArtifactID(scope, blueprintName, versionName, artifactName))]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondAzureBlueprintNotFound(w, path, "published artifact not found")
		return
	}
	respondJSON(w, http.StatusOK, azureBlueprintPublishedArtifactFixture(record))
}

func (s *Server) handleAzureBlueprintAssignmentsRoute(w http.ResponseWriter, r *http.Request, path, scope string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBlueprintAssignmentsList(w, scope)
		return
	}

	assignmentName := strings.TrimSpace(tail[1])
	if assignmentName == "" {
		respondAzureInvalidRequest(w, path, "assignment name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBlueprintAssignmentGet(w, path, scope, assignmentName, false)
		case http.MethodHead:
			s.handleAzureBlueprintAssignmentGet(w, path, scope, assignmentName, true)
		case http.MethodPut:
			s.handleAzureBlueprintAssignmentCreateOrUpdate(w, r, path, scope, assignmentName)
		case http.MethodDelete:
			s.handleAzureBlueprintAssignmentDelete(w, path, scope, assignmentName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "whoisblueprint":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureBlueprintWhoIsBlueprint(w, path, scope, assignmentName)
			return
		}
	case "assignmentoperations":
		s.handleAzureBlueprintAssignmentOperationsRoute(w, r, path, scope, assignmentName, tail[3:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureBlueprintAssignmentsList(w http.ResponseWriter, scope string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, assignment := range s.azureBlueprintAssignments {
		if !strings.EqualFold(assignment.Scope, scope) {
			continue
		}
		items = append(items, azureBlueprintAssignmentFixture(assignment))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureBlueprintsString(items[i]["id"])) < strings.ToLower(azureBlueprintsString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureBlueprintAssignmentCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, scope, assignmentName string) {
	body, payload, ok := azureBlueprintsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureBlueprintsString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	identity := azureBlueprintsMap(payload["identity"])
	if len(identity) == 0 {
		respondAzureInvalidRequest(w, path, "identity is required")
		return
	}
	properties := azureBlueprintsMap(payload["properties"])
	parameters := azureBlueprintsMap(properties["parameters"])
	resourceGroups := azureBlueprintsMap(properties["resourceGroups"])
	if len(parameters) == 0 {
		respondAzureInvalidRequest(w, path, "properties.parameters is required")
		return
	}
	if len(resourceGroups) == 0 {
		respondAzureInvalidRequest(w, path, "properties.resourceGroups is required")
		return
	}
	if strings.EqualFold(azureBlueprintsScopeKind(scope), "managementgroup") && strings.TrimSpace(azureBlueprintsString(properties["scope"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.scope is required for management group assignments")
		return
	}

	now := time.Now().UTC()
	key := azureBlueprintsStorageKey(azureBlueprintAssignmentID(scope, assignmentName))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	record := s.azureBlueprintAssignments[key]
	status := http.StatusOK
	createdAt := now
	if record == nil {
		record = &azureBlueprintAssignment{
			Scope: scope,
			Name:  assignmentName,
		}
		status = http.StatusCreated
	} else {
		createdAt = record.CreatedAt
	}
	record.Location = location
	record.Identity = azureBlueprintsNormalizeIdentity(identity)
	record.Properties = azureBlueprintsNormalizeAssignmentProperties(properties)
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	record.ETag = azureBlueprintsETag(key, body, now)
	s.azureBlueprintAssignments[key] = record

	op := azureBlueprintAssignmentOperationFixtureRecord(scope, assignmentName, record.Properties, now, len(s.azureBlueprintAssignmentOperations)+1)
	s.azureBlueprintAssignmentOperations[azureBlueprintsStorageKey(azureBlueprintAssignmentOperationID(scope, assignmentName, op.Name))] = op

	respondJSON(w, status, azureBlueprintAssignmentFixture(record))
}

func (s *Server) handleAzureBlueprintAssignmentGet(w http.ResponseWriter, path, scope, assignmentName string, headOnly bool) {
	s.providerStorageMu.Lock()
	record := s.azureBlueprintAssignments[azureBlueprintsStorageKey(azureBlueprintAssignmentID(scope, assignmentName))]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondAzureBlueprintNotFound(w, path, "blueprint assignment not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureBlueprintAssignmentFixture(record))
}

func (s *Server) handleAzureBlueprintAssignmentDelete(w http.ResponseWriter, path, scope, assignmentName string) {
	key := azureBlueprintsStorageKey(azureBlueprintAssignmentID(scope, assignmentName))

	s.providerStorageMu.Lock()
	record := s.azureBlueprintAssignments[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		respondAzureBlueprintNotFound(w, path, "blueprint assignment not found")
		return
	}
	delete(s.azureBlueprintAssignments, key)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureBlueprintAssignmentFixture(record))
}

func (s *Server) handleAzureBlueprintWhoIsBlueprint(w http.ResponseWriter, path, scope, assignmentName string) {
	if !s.azureBlueprintAssignmentExists(scope, assignmentName) {
		respondAzureBlueprintNotFound(w, path, "blueprint assignment not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"objectId": "00000000-1111-0000-1111-000000000000",
	})
}

func (s *Server) handleAzureBlueprintAssignmentOperationsRoute(w http.ResponseWriter, r *http.Request, path, scope, assignmentName string, tail []string) {
	if !s.azureBlueprintAssignmentExists(scope, assignmentName) {
		respondAzureBlueprintNotFound(w, path, "blueprint assignment not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBlueprintAssignmentOperationsList(w, scope, assignmentName)
		return
	}

	operationName := strings.TrimSpace(tail[0])
	if operationName == "" {
		respondAzureInvalidRequest(w, path, "operation name is required")
		return
	}
	if len(tail) != 1 || r.Method != http.MethodGet {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureBlueprintAssignmentOperationGet(w, path, scope, assignmentName, operationName)
}

func (s *Server) handleAzureBlueprintAssignmentOperationsList(w http.ResponseWriter, scope, assignmentName string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, operation := range s.azureBlueprintAssignmentOperations {
		if !strings.EqualFold(operation.Scope, scope) || !strings.EqualFold(operation.AssignmentName, assignmentName) {
			continue
		}
		items = append(items, azureBlueprintAssignmentOperationFixture(operation))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureBlueprintsString(items[i]["id"])) < strings.ToLower(azureBlueprintsString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    items,
		"nextLink": nil,
	})
}

func (s *Server) handleAzureBlueprintAssignmentOperationGet(w http.ResponseWriter, path, scope, assignmentName, operationName string) {
	s.providerStorageMu.Lock()
	record := s.azureBlueprintAssignmentOperations[azureBlueprintsStorageKey(azureBlueprintAssignmentOperationID(scope, assignmentName, operationName))]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondAzureBlueprintNotFound(w, path, "assignment operation not found")
		return
	}
	respondJSON(w, http.StatusOK, azureBlueprintAssignmentOperationFixture(record))
}

func (s *Server) azureBlueprintDefinitionExists(scope, blueprintName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureBlueprintDefinitions[azureBlueprintsStorageKey(azureBlueprintDefinitionID(scope, blueprintName))] != nil
}

func (s *Server) azureBlueprintVersionExists(scope, blueprintName, versionName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureBlueprintVersions[azureBlueprintsStorageKey(azureBlueprintVersionID(scope, blueprintName, versionName))] != nil
}

func (s *Server) azureBlueprintAssignmentExists(scope, assignmentName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureBlueprintAssignments[azureBlueprintsStorageKey(azureBlueprintAssignmentID(scope, assignmentName))] != nil
}

func azureBlueprintDefinitionID(scope, blueprintName string) string {
	return azureBlueprintsNormalizeScope(scope) + "/providers/Microsoft.Blueprint/blueprints/" + strings.TrimSpace(blueprintName)
}

func azureBlueprintArtifactID(scope, blueprintName, artifactName string) string {
	return azureBlueprintDefinitionID(scope, blueprintName) + "/artifacts/" + strings.TrimSpace(artifactName)
}

func azureBlueprintVersionID(scope, blueprintName, versionName string) string {
	return azureBlueprintDefinitionID(scope, blueprintName) + "/versions/" + strings.TrimSpace(versionName)
}

func azureBlueprintPublishedArtifactID(scope, blueprintName, versionName, artifactName string) string {
	return azureBlueprintVersionID(scope, blueprintName, versionName) + "/artifacts/" + strings.TrimSpace(artifactName)
}

func azureBlueprintAssignmentID(scope, assignmentName string) string {
	return azureBlueprintsNormalizeScope(scope) + "/providers/Microsoft.Blueprint/blueprintAssignments/" + strings.TrimSpace(assignmentName)
}

func azureBlueprintAssignmentOperationID(scope, assignmentName, operationName string) string {
	return azureBlueprintAssignmentID(scope, assignmentName) + "/assignmentOperations/" + strings.TrimSpace(operationName)
}

func azureBlueprintDefinitionFixture(record *azureBlueprintDefinition) map[string]any {
	return map[string]any{
		"id":          azureBlueprintDefinitionID(record.Scope, record.Name),
		"name":        record.Name,
		"type":        "Microsoft.Blueprint/blueprints",
		"properties":  azureBlueprintsCloneMap(record.Properties),
		"etag":        record.ETag,
		"description": azureBlueprintsString(record.Properties["description"]),
		"displayName": azureBlueprintsString(record.Properties["displayName"]),
	}
}

func azureBlueprintArtifactFixture(record *azureBlueprintArtifact) map[string]any {
	return map[string]any{
		"id":         azureBlueprintArtifactID(record.Scope, record.BlueprintName, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.Blueprint/blueprints/artifacts",
		"kind":       record.Kind,
		"properties": azureBlueprintsCloneMap(record.Properties),
		"etag":       record.ETag,
	}
}

func azureBlueprintVersionFixture(record *azureBlueprintVersion) map[string]any {
	return map[string]any{
		"id":          azureBlueprintVersionID(record.Scope, record.BlueprintName, record.Name),
		"name":        record.Name,
		"type":        "Microsoft.Blueprint/blueprints/versions",
		"properties":  azureBlueprintsCloneMap(record.Properties),
		"etag":        record.ETag,
		"description": azureBlueprintsString(record.Properties["description"]),
		"displayName": azureBlueprintsString(record.Properties["displayName"]),
	}
}

func azureBlueprintPublishedArtifactFixture(record *azureBlueprintPublishedArtifact) map[string]any {
	return map[string]any{
		"id":         azureBlueprintPublishedArtifactID(record.Scope, record.BlueprintName, record.VersionName, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.Blueprint/blueprints/versions/artifacts",
		"kind":       record.Kind,
		"properties": azureBlueprintsCloneMap(record.Properties),
		"etag":       record.ETag,
	}
}

func azureBlueprintAssignmentFixture(record *azureBlueprintAssignment) map[string]any {
	return map[string]any{
		"id":         azureBlueprintAssignmentID(record.Scope, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.Blueprint/Assignment",
		"location":   record.Location,
		"identity":   azureBlueprintsCloneMap(record.Identity),
		"properties": azureBlueprintsCloneMap(record.Properties),
		"etag":       record.ETag,
	}
}

func azureBlueprintAssignmentOperationFixture(record *azureBlueprintAssignmentOperation) map[string]any {
	return map[string]any{
		"id":         azureBlueprintAssignmentOperationID(record.Scope, record.AssignmentName, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.Blueprint/blueprintAssignments/operations",
		"properties": azureBlueprintsCloneMap(record.Properties),
	}
}

func azureBlueprintAssignmentOperationFixtureRecord(scope, assignmentName string, assignmentProperties map[string]any, now time.Time, sequence int) *azureBlueprintAssignmentOperation {
	versionName := "latest"
	blueprintID := strings.TrimSpace(azureBlueprintsString(assignmentProperties["blueprintId"]))
	if idx := strings.LastIndex(strings.ToLower(blueprintID), "/versions/"); idx >= 0 {
		versionName = strings.TrimSpace(blueprintID[idx+len("/versions/"):])
		if versionName == "" {
			versionName = "latest"
		}
	}
	name := fmt.Sprintf("%040x", sequence)
	return &azureBlueprintAssignmentOperation{
		Scope:          scope,
		AssignmentName: assignmentName,
		Name:           name,
		CreatedAt:      now,
		UpdatedAt:      now,
		Properties: map[string]any{
			"blueprintVersion": versionName,
			"assignmentState":  "succeed",
			"timeCreated":      now.Format(time.RFC3339),
			"timeStarted":      now.Format(time.RFC3339),
			"timeFinished":     now.Format(time.RFC3339),
			"deployments": []any{
				map[string]any{
					"kind":       "azureResource",
					"action":     "put",
					"jobState":   "succeeded",
					"requestUri": "/providers/Microsoft.Resources/deployments/" + assignmentName + "-deploy",
					"result": map[string]any{
						"resources": []any{
							map[string]any{
								"id":   "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/stackyard-rg/providers/Microsoft.Storage/storageAccounts/stackyardbp",
								"name": "stackyardbp",
								"type": "Microsoft.Storage/storageAccounts",
							},
						},
					},
					"history": []any{},
				},
			},
		},
	}
}

func azureBlueprintsNormalizeDefinitionProperties(blueprintName string, incoming map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureBlueprintsCloneMap(incoming)
	if strings.TrimSpace(azureBlueprintsString(props["displayName"])) == "" {
		props["displayName"] = blueprintName
	}
	if _, ok := props["parameters"]; !ok {
		props["parameters"] = map[string]any{}
	}
	if _, ok := props["resourceGroups"]; !ok {
		props["resourceGroups"] = map[string]any{}
	}
	if strings.TrimSpace(azureBlueprintsString(props["targetScope"])) == "" {
		props["targetScope"] = "subscription"
	}
	props["status"] = map[string]any{
		"timeCreated":  createdAt.Format(time.RFC3339),
		"lastModified": updatedAt.Format(time.RFC3339),
	}
	return props
}

func azureBlueprintsNormalizeVersionProperties(blueprintName, versionName string, definitionProperties map[string]any, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureBlueprintsCloneMap(definitionProperties)
	if displayName := strings.TrimSpace(azureBlueprintsString(payload["displayName"])); displayName != "" {
		props["displayName"] = displayName
	}
	if description := strings.TrimSpace(azureBlueprintsString(payload["description"])); description != "" {
		props["description"] = description
	}
	if nestedProps := azureBlueprintsMap(payload["properties"]); len(nestedProps) > 0 {
		for key, value := range nestedProps {
			props[key] = value
		}
	}
	props["blueprintName"] = blueprintName
	if strings.TrimSpace(azureBlueprintsString(props["displayName"])) == "" {
		props["displayName"] = versionName
	}
	props["status"] = map[string]any{
		"timeCreated":  createdAt.Format(time.RFC3339),
		"lastModified": updatedAt.Format(time.RFC3339),
	}
	return props
}

func azureBlueprintsNormalizeAssignmentProperties(incoming map[string]any) map[string]any {
	props := azureBlueprintsCloneMap(incoming)
	props["parameters"] = azureBlueprintsCloneMap(azureBlueprintsMap(incoming["parameters"]))
	props["resourceGroups"] = azureBlueprintsCloneMap(azureBlueprintsMap(incoming["resourceGroups"]))
	props["locks"] = azureBlueprintsNormalizeAssignmentLocks(azureBlueprintsMap(incoming["locks"]))
	props["provisioningState"] = "creating"
	return props
}

func azureBlueprintsNormalizeAssignmentLocks(incoming map[string]any) map[string]any {
	if len(incoming) == 0 {
		return map[string]any{"mode": "None"}
	}
	return azureBlueprintsCloneMap(incoming)
}

func azureBlueprintsNormalizeIdentity(identity map[string]any) map[string]any {
	out := azureBlueprintsCloneMap(identity)
	if strings.TrimSpace(azureBlueprintsString(out["type"])) == "" {
		out["type"] = "SystemAssigned"
	}
	if strings.EqualFold(azureBlueprintsString(out["type"]), "SystemAssigned") {
		out["tenantId"] = "00000000-0000-0000-0000-000000000000"
		out["principalId"] = "00000000-0000-0000-0000-000000000000"
	}
	return out
}

func azureBlueprintsNormalizeScope(scope string) string {
	switch {
	case strings.HasPrefix(strings.ToLower(scope), "/managementgroups/"):
		managementGroup := strings.TrimSpace(scope[len("/managementGroups/"):])
		return "/providers/Microsoft.Management/managementGroups/" + managementGroup
	default:
		return "/" + strings.Trim(strings.TrimSpace(scope), "/")
	}
}

func azureBlueprintsScopeKind(scope string) string {
	scope = azureBlueprintsNormalizeScope(scope)
	switch {
	case strings.HasPrefix(strings.ToLower(scope), "/subscriptions/"):
		return "subscription"
	case strings.HasPrefix(strings.ToLower(scope), "/providers/microsoft.management/managementgroups/"):
		return "managementGroup"
	default:
		return ""
	}
}

func azureBlueprintsStorageKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func azureBlueprintsETag(key string, body []byte, updatedAt time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"), append(body, []byte(updatedAt.UTC().Format(time.RFC3339Nano))...)...))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureBlueprintsReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
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
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func respondAzureBlueprintNotFound(w http.ResponseWriter, path, message string) {
	if strings.TrimSpace(message) == "" {
		message = "resource not found"
	}
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": message,
		},
		"path": path,
	})
}

func azureBlueprintsString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func azureBlueprintsMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureBlueprintsCloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, entry := range value {
		out[key] = azureBlueprintsCloneValue(entry)
	}
	return out
}

func azureBlueprintsCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return azureBlueprintsCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, azureBlueprintsCloneValue(item))
		}
		return out
	default:
		return typed
	}
}
