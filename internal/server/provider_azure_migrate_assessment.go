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

const azureMigrateAssessmentAPIVersion = "2024-01-15"

var azureMigrateAssessmentCollectorCollections = map[string]string{
	"vmwarecollectors": "vmwareCollectors",
	"hypervcollectors": "hypervCollectors",
	"servercollectors": "serverCollectors",
	"importcollectors": "importCollectors",
	"sqlcollectors":    "sqlCollectors",
	"webappcollectors": "webAppCollectors",
}

type azureMigrateAssessmentProject struct {
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

type azureMigrateAssessmentGroup struct {
	ProjectID    string
	Name         string
	Properties   map[string]any
	MachineNames []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ETag         string
}

type azureMigrateAssessmentMachine struct {
	ProjectID  string
	Name       string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type azureMigrateAssessment struct {
	ProjectID  string
	GroupName  string
	Name       string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureMigrateAssessmentCollector struct {
	ProjectID  string
	Collection string
	Name       string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureMigrateAssessmentPrivateEndpointConnection struct {
	ProjectID  string
	Name       string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

func (s *Server) handleAzureMigrateAssessmentRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsMigrateAssessmentPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureMigrateAssessmentAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Migrate") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureMigrateAssessmentOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureMigrateAssessmentRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}
	if !strings.EqualFold(tail[0], "assessmentProjects") {
		respondAzureImplemented(w, path)
		return true
	}

	s.handleAzureMigrateAssessmentProjectsRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsMigrateAssessmentPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Migrate") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureMigrateAssessmentRouteContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "assessmentProjects")
}

func azureMigrateAssessmentRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Migrate") {
			return subscriptionID, resourceGroup, segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureMigrateAssessmentAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureMigrateAssessmentProjectsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureMigrateAssessmentListProjectsBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureMigrateAssessmentListProjectsByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	projectName := strings.TrimSpace(tail[1])
	if projectName == "" {
		respondAzureInvalidRequest(w, path, "assessment project name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureMigrateAssessmentGetProject(w, path, subscriptionID, resourceGroup, projectName, false)
		case http.MethodHead:
			s.handleAzureMigrateAssessmentGetProject(w, path, subscriptionID, resourceGroup, projectName, true)
		case http.MethodPut:
			s.handleAzureMigrateAssessmentCreateProject(w, r, path, subscriptionID, resourceGroup, projectName)
		case http.MethodPatch:
			s.handleAzureMigrateAssessmentPatchProject(w, r, path, subscriptionID, resourceGroup, projectName)
		case http.MethodDelete:
			s.handleAzureMigrateAssessmentDeleteProject(w, subscriptionID, resourceGroup, projectName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "groups":
		s.handleAzureMigrateAssessmentGroupsRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	case "machines":
		s.handleAzureMigrateAssessmentMachinesRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	case "assessmentoptions":
		s.handleAzureMigrateAssessmentOptionsRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	case "projectsummary":
		s.handleAzureMigrateAssessmentProjectSummaryRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	case "privateendpointconnections":
		s.handleAzureMigrateAssessmentPrivateEndpointConnectionsRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	case "privatelinkresources":
		s.handleAzureMigrateAssessmentPrivateLinkResourcesRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	case "aksoptions":
		s.handleAzureMigrateAssessmentAKSOptionsRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	case "akssummary":
		s.handleAzureMigrateAssessmentAKSSummaryRoute(w, r, path, subscriptionID, resourceGroup, projectName, tail[3:])
	default:
		if _, ok := azureMigrateAssessmentCollectorCollections[strings.ToLower(strings.TrimSpace(tail[2]))]; ok {
			s.handleAzureMigrateAssessmentCollectorsRoute(w, r, path, subscriptionID, resourceGroup, projectName, strings.ToLower(strings.TrimSpace(tail[2])), tail[3:])
			return
		}
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureMigrateAssessmentCreateProject(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string) {
	body, payload, ok := azureMigrateAssessmentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureMigrateAssessmentString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()
	key := azureMigrateAssessmentProjectKey(subscriptionID, resourceGroup, projectName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	project := s.azureMigrateAssessmentProjects[key]
	status := http.StatusOK
	createdAt := now
	if project == nil {
		status = http.StatusCreated
		project = &azureMigrateAssessmentProject{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           projectName,
		}
	} else {
		createdAt = project.CreatedAt
	}
	project.Location = location
	project.Tags = azureMigrateAssessmentStringMap(payload["tags"])
	project.Properties = azureMigrateAssessmentNormalizeProjectProperties(projectName, location, azureMigrateAssessmentObject(payload["properties"]))
	project.CreatedAt = createdAt
	project.UpdatedAt = now
	project.ETag = azureMigrateAssessmentETag(key, body, now)
	s.azureMigrateAssessmentProjects[key] = project
	s.azureMigrateAssessmentEnsureProjectFixturesLocked(project)

	respondJSON(w, status, azureMigrateAssessmentProjectFixture(project))
}

func (s *Server) handleAzureMigrateAssessmentPatchProject(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string) {
	body, payload, ok := azureMigrateAssessmentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	key := azureMigrateAssessmentProjectKey(subscriptionID, resourceGroup, projectName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	project := s.azureMigrateAssessmentProjects[key]
	if project == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectName))
		return
	}
	if _, ok := payload["tags"]; ok {
		project.Tags = azureMigrateAssessmentStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		project.Properties = azureMigrateAssessmentMergeObjects(project.Properties, azureMigrateAssessmentObject(payload["properties"]))
		project.Properties = azureMigrateAssessmentNormalizeProjectProperties(project.Name, project.Location, project.Properties)
	}
	project.UpdatedAt = now
	project.ETag = azureMigrateAssessmentETag(key, body, now)
	respondJSON(w, http.StatusOK, azureMigrateAssessmentProjectFixture(project))
}

func (s *Server) handleAzureMigrateAssessmentGetProject(w http.ResponseWriter, path, subscriptionID, resourceGroup, projectName string, headOnly bool) {
	s.providerStorageMu.Lock()
	project := s.azureMigrateAssessmentProjects[azureMigrateAssessmentProjectKey(subscriptionID, resourceGroup, projectName)]
	s.providerStorageMu.Unlock()
	if project == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectName))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateAssessmentProjectFixture(project))
}

func (s *Server) handleAzureMigrateAssessmentDeleteProject(w http.ResponseWriter, subscriptionID, resourceGroup, projectName string) {
	key := azureMigrateAssessmentProjectKey(subscriptionID, resourceGroup, projectName)
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)

	s.providerStorageMu.Lock()
	_, exists := s.azureMigrateAssessmentProjects[key]
	if exists {
		delete(s.azureMigrateAssessmentProjects, key)
		for k, item := range s.azureMigrateAssessmentGroups {
			if item.ProjectID == projectID {
				delete(s.azureMigrateAssessmentGroups, k)
			}
		}
		for k, item := range s.azureMigrateAssessmentMachines {
			if item.ProjectID == projectID {
				delete(s.azureMigrateAssessmentMachines, k)
			}
		}
		for k, item := range s.azureMigrateAssessments {
			if item.ProjectID == projectID {
				delete(s.azureMigrateAssessments, k)
			}
		}
		for k, item := range s.azureMigrateAssessmentCollectors {
			if item.ProjectID == projectID {
				delete(s.azureMigrateAssessmentCollectors, k)
			}
		}
		for k, item := range s.azureMigrateAssessmentPrivateEndpointConnections {
			if item.ProjectID == projectID {
				delete(s.azureMigrateAssessmentPrivateEndpointConnections, k)
			}
		}
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureMigrateAssessmentListProjectsBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	for _, project := range s.azureMigrateAssessmentProjects {
		if project.SubscriptionID == subscriptionID {
			values = append(values, azureMigrateAssessmentProjectFixture(project))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureMigrateAssessmentString(values[i]["id"]) < azureMigrateAssessmentString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureMigrateAssessmentListProjectsByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	for _, project := range s.azureMigrateAssessmentProjects {
		if project.SubscriptionID == subscriptionID && strings.EqualFold(project.ResourceGroup, resourceGroup) {
			values = append(values, azureMigrateAssessmentProjectFixture(project))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureMigrateAssessmentString(values[i]["id"]) < azureMigrateAssessmentString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureMigrateAssessmentGroupsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	if len(tail) == 0 && r.Method == http.MethodGet {
		s.handleAzureMigrateAssessmentListGroups(w, path, projectID)
		return
	}
	if len(tail) < 1 {
		respondAzureImplemented(w, path)
		return
	}
	groupName := strings.TrimSpace(tail[0])
	if groupName == "" {
		respondAzureInvalidRequest(w, path, "group name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureMigrateAssessmentGetGroup(w, path, projectID, groupName)
		case http.MethodPut:
			s.handleAzureMigrateAssessmentCreateGroup(w, r, path, projectID, groupName)
		case http.MethodDelete:
			s.handleAzureMigrateAssessmentDeleteGroup(w, projectID, groupName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "updatemachines":
		if r.Method == http.MethodPost {
			s.handleAzureMigrateAssessmentUpdateGroupMachines(w, r, path, projectID, groupName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "assessments":
		s.handleAzureMigrateAssessmentAssessmentsRoute(w, r, path, projectID, groupName, tail[2:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureMigrateAssessmentCreateGroup(w http.ResponseWriter, r *http.Request, path, projectID, groupName string) {
	body, payload, ok := azureMigrateAssessmentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureMigrateAssessmentGroupKey(projectID, groupName)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectID))
		return
	}
	group := s.azureMigrateAssessmentGroups[key]
	status := http.StatusOK
	createdAt := now
	if group == nil {
		status = http.StatusCreated
		group = &azureMigrateAssessmentGroup{ProjectID: projectID, Name: groupName}
	} else {
		createdAt = group.CreatedAt
	}
	group.Properties = azureMigrateAssessmentNormalizeGroupProperties(azureMigrateAssessmentObject(payload["properties"]))
	group.CreatedAt = createdAt
	group.UpdatedAt = now
	group.ETag = azureMigrateAssessmentETag(key, body, now)
	s.azureMigrateAssessmentGroups[key] = group
	respondJSON(w, status, azureMigrateAssessmentGroupFixture(group))
}

func (s *Server) handleAzureMigrateAssessmentGetGroup(w http.ResponseWriter, path, projectID, groupName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	group := s.azureMigrateAssessmentGroups[azureMigrateAssessmentGroupKey(projectID, groupName)]
	if group == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, groupName))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateAssessmentGroupFixture(group))
}

func (s *Server) handleAzureMigrateAssessmentListGroups(w http.ResponseWriter, path, projectID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectID))
		return
	}
	values := make([]map[string]any, 0)
	for _, group := range s.azureMigrateAssessmentGroups {
		if group.ProjectID == projectID {
			values = append(values, azureMigrateAssessmentGroupFixture(group))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureMigrateAssessmentString(values[i]["id"]) < azureMigrateAssessmentString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureMigrateAssessmentDeleteGroup(w http.ResponseWriter, projectID, groupName string) {
	key := azureMigrateAssessmentGroupKey(projectID, groupName)
	s.providerStorageMu.Lock()
	delete(s.azureMigrateAssessmentGroups, key)
	for k, item := range s.azureMigrateAssessments {
		if item.ProjectID == projectID && strings.EqualFold(item.GroupName, groupName) {
			delete(s.azureMigrateAssessments, k)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureMigrateAssessmentUpdateGroupMachines(w http.ResponseWriter, r *http.Request, path, projectID, groupName string) {
	_, payload, ok := azureMigrateAssessmentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	group := s.azureMigrateAssessmentGroups[azureMigrateAssessmentGroupKey(projectID, groupName)]
	if group == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, groupName))
		return
	}
	machineNames := azureMigrateAssessmentMembershipNames(payload)
	if len(machineNames) == 0 {
		for _, machine := range s.azureMigrateAssessmentProjectMachinesLocked(projectID) {
			machineNames = append(machineNames, machine.Name)
		}
	}
	group.MachineNames = machineNames
	group.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureMigrateAssessmentGroupFixture(group))
}

func (s *Server) handleAzureMigrateAssessmentMachinesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	if len(tail) == 0 && r.Method == http.MethodGet {
		s.handleAzureMigrateAssessmentListMachines(w, path, projectID)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureMigrateAssessmentGetMachine(w, path, projectID, tail[0])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateAssessmentListMachines(w http.ResponseWriter, path, projectID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectID))
		return
	}
	values := make([]map[string]any, 0)
	for _, machine := range s.azureMigrateAssessmentProjectMachinesLocked(projectID) {
		values = append(values, azureMigrateAssessmentMachineFixture(machine))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureMigrateAssessmentGetMachine(w http.ResponseWriter, path, projectID, machineName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	machine := s.azureMigrateAssessmentMachines[azureMigrateAssessmentMachineKey(projectID, machineName)]
	if machine == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, machineName))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateAssessmentMachineFixture(machine))
}

func (s *Server) handleAzureMigrateAssessmentOptionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) > 1 {
		respondAzureImplemented(w, path)
		return
	}
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	s.providerStorageMu.Lock()
	project := s.azureMigrateAssessmentProjects[azureMigrateAssessmentProjectKey(subscriptionID, resourceGroup, projectName)]
	s.providerStorageMu.Unlock()
	if project == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectName))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":   projectID + "/assessmentOptions/default",
		"name": "default",
		"type": "Microsoft.Migrate/assessmentProjects/assessmentOptions",
		"properties": map[string]any{
			"reservedInstanceSupported":   true,
			"azureHybridBenefitSupported": true,
			"currency":                    "USD",
			"targetLocations": []any{
				project.Location,
				"eastus2",
			},
		},
	})
}

func (s *Server) handleAzureMigrateAssessmentProjectSummaryRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectName))
		return
	}
	values := []map[string]any{azureMigrateAssessmentProjectSummaryFixture(s, projectID)}
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": values})
		return
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "default") {
		respondJSON(w, http.StatusOK, values[0])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateAssessmentAssessmentsRoute(w http.ResponseWriter, r *http.Request, path, projectID, groupName string, tail []string) {
	if len(tail) == 0 && r.Method == http.MethodGet {
		s.handleAzureMigrateAssessmentListAssessments(w, path, projectID, groupName)
		return
	}
	if len(tail) < 1 {
		respondAzureImplemented(w, path)
		return
	}
	assessmentName := strings.TrimSpace(tail[0])
	if assessmentName == "" {
		respondAzureInvalidRequest(w, path, "assessment name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodPut:
			s.handleAzureMigrateAssessmentCreateAssessment(w, r, path, projectID, groupName, assessmentName)
		case http.MethodGet:
			s.handleAzureMigrateAssessmentGetAssessment(w, path, projectID, groupName, assessmentName)
		case http.MethodDelete:
			s.handleAzureMigrateAssessmentDeleteAssessment(w, projectID, groupName, assessmentName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) >= 2 && strings.EqualFold(tail[1], "assessedMachines") {
		s.handleAzureMigrateAssessmentAssessedMachinesRoute(w, r, path, projectID, groupName, assessmentName, tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateAssessmentCreateAssessment(w http.ResponseWriter, r *http.Request, path, projectID, groupName, assessmentName string) {
	body, payload, ok := azureMigrateAssessmentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureMigrateAssessmentAssessmentKey(projectID, groupName, assessmentName)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureMigrateAssessmentGroups[azureMigrateAssessmentGroupKey(projectID, groupName)] == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, groupName))
		return
	}
	item := s.azureMigrateAssessments[key]
	status := http.StatusOK
	createdAt := now
	if item == nil {
		status = http.StatusCreated
		item = &azureMigrateAssessment{ProjectID: projectID, GroupName: groupName, Name: assessmentName}
	} else {
		createdAt = item.CreatedAt
	}
	item.Properties = azureMigrateAssessmentNormalizeAssessmentProperties(azureMigrateAssessmentObject(payload["properties"]))
	item.CreatedAt = createdAt
	item.UpdatedAt = now
	item.ETag = azureMigrateAssessmentETag(key, body, now)
	s.azureMigrateAssessments[key] = item
	respondJSON(w, status, azureMigrateAssessmentFixture(item))
}

func (s *Server) handleAzureMigrateAssessmentGetAssessment(w http.ResponseWriter, path, projectID, groupName, assessmentName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	item := s.azureMigrateAssessments[azureMigrateAssessmentAssessmentKey(projectID, groupName, assessmentName)]
	if item == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, assessmentName))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateAssessmentFixture(item))
}

func (s *Server) handleAzureMigrateAssessmentListAssessments(w http.ResponseWriter, path, projectID, groupName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureMigrateAssessmentGroups[azureMigrateAssessmentGroupKey(projectID, groupName)] == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, groupName))
		return
	}
	values := make([]map[string]any, 0)
	for _, item := range s.azureMigrateAssessments {
		if item.ProjectID == projectID && strings.EqualFold(item.GroupName, groupName) {
			values = append(values, azureMigrateAssessmentFixture(item))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureMigrateAssessmentString(values[i]["id"]) < azureMigrateAssessmentString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureMigrateAssessmentDeleteAssessment(w http.ResponseWriter, projectID, groupName, assessmentName string) {
	s.providerStorageMu.Lock()
	delete(s.azureMigrateAssessments, azureMigrateAssessmentAssessmentKey(projectID, groupName, assessmentName))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureMigrateAssessmentAssessedMachinesRoute(w http.ResponseWriter, r *http.Request, path, projectID, groupName, assessmentName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	assessment := s.azureMigrateAssessments[azureMigrateAssessmentAssessmentKey(projectID, groupName, assessmentName)]
	group := s.azureMigrateAssessmentGroups[azureMigrateAssessmentGroupKey(projectID, groupName)]
	if assessment == nil || group == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, assessmentName))
		return
	}
	values := make([]map[string]any, 0)
	for _, machineName := range group.MachineNames {
		machine := s.azureMigrateAssessmentMachines[azureMigrateAssessmentMachineKey(projectID, machineName)]
		if machine == nil {
			continue
		}
		values = append(values, azureMigrateAssessmentAssessedMachineFixture(projectID, groupName, assessmentName, machine))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureMigrateAssessmentString(values[i]["id"]) < azureMigrateAssessmentString(values[j]["id"])
	})
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": values})
		return
	}
	if len(tail) == 1 {
		for _, value := range values {
			if strings.EqualFold(azureMigrateAssessmentString(value["name"]), tail[0]) {
				respondJSON(w, http.StatusOK, value)
				return
			}
		}
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, tail[0]))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateAssessmentCollectorsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName, collection string, tail []string) {
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	if len(tail) == 0 && r.Method == http.MethodGet {
		s.handleAzureMigrateAssessmentListCollectors(w, path, projectID, collection)
		return
	}
	if len(tail) < 1 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "collector name is required")
		return
	}
	switch r.Method {
	case http.MethodPut:
		s.handleAzureMigrateAssessmentCreateCollector(w, r, path, projectID, collection, name)
	case http.MethodGet:
		s.handleAzureMigrateAssessmentGetCollector(w, path, projectID, collection, name)
	case http.MethodDelete:
		s.handleAzureMigrateAssessmentDeleteCollector(w, projectID, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureMigrateAssessmentCreateCollector(w http.ResponseWriter, r *http.Request, path, projectID, collection, name string) {
	body, payload, ok := azureMigrateAssessmentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureMigrateAssessmentCollectorKey(projectID, collection, name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectID))
		return
	}
	item := s.azureMigrateAssessmentCollectors[key]
	status := http.StatusOK
	createdAt := now
	if item == nil {
		status = http.StatusCreated
		item = &azureMigrateAssessmentCollector{ProjectID: projectID, Collection: collection, Name: name}
	} else {
		createdAt = item.CreatedAt
	}
	item.Properties = azureMigrateAssessmentNormalizeCollectorProperties(collection, azureMigrateAssessmentObject(payload["properties"]))
	item.CreatedAt = createdAt
	item.UpdatedAt = now
	item.ETag = azureMigrateAssessmentETag(key, body, now)
	s.azureMigrateAssessmentCollectors[key] = item
	respondJSON(w, status, azureMigrateAssessmentCollectorFixture(item))
}

func (s *Server) handleAzureMigrateAssessmentGetCollector(w http.ResponseWriter, path, projectID, collection, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	item := s.azureMigrateAssessmentCollectors[azureMigrateAssessmentCollectorKey(projectID, collection, name)]
	if item == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateAssessmentCollectorFixture(item))
}

func (s *Server) handleAzureMigrateAssessmentListCollectors(w http.ResponseWriter, path, projectID, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectID))
		return
	}
	values := make([]map[string]any, 0)
	for _, item := range s.azureMigrateAssessmentCollectors {
		if item.ProjectID == projectID && strings.EqualFold(item.Collection, collection) {
			values = append(values, azureMigrateAssessmentCollectorFixture(item))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureMigrateAssessmentString(values[i]["id"]) < azureMigrateAssessmentString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureMigrateAssessmentDeleteCollector(w http.ResponseWriter, projectID, collection, name string) {
	s.providerStorageMu.Lock()
	delete(s.azureMigrateAssessmentCollectors, azureMigrateAssessmentCollectorKey(projectID, collection, name))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureMigrateAssessmentPrivateEndpointConnectionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzureMigrateAssessmentListPrivateEndpointConnections(w, path, projectID)
	case len(tail) == 1 && r.Method == http.MethodGet:
		s.handleAzureMigrateAssessmentGetPrivateEndpointConnection(w, path, projectID, tail[0])
	case len(tail) == 1 && r.Method == http.MethodPatch:
		s.handleAzureMigrateAssessmentPatchPrivateEndpointConnection(w, r, path, projectID, tail[0])
	case len(tail) == 0 || len(tail) == 1:
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureMigrateAssessmentListPrivateEndpointConnections(w http.ResponseWriter, path, projectID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectID))
		return
	}
	values := make([]map[string]any, 0)
	for _, item := range s.azureMigrateAssessmentPrivateEndpointConnectionsLocked(projectID) {
		values = append(values, azureMigrateAssessmentPrivateEndpointConnectionFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureMigrateAssessmentGetPrivateEndpointConnection(w http.ResponseWriter, path, projectID, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	item := s.azureMigrateAssessmentEnsurePrivateEndpointConnectionLocked(projectID, name)
	if item == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateAssessmentPrivateEndpointConnectionFixture(item))
}

func (s *Server) handleAzureMigrateAssessmentPatchPrivateEndpointConnection(w http.ResponseWriter, r *http.Request, path, projectID, name string) {
	body, payload, ok := azureMigrateAssessmentReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureMigrateAssessmentPrivateEndpointConnectionKey(projectID, name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	item := s.azureMigrateAssessmentEnsurePrivateEndpointConnectionLocked(projectID, name)
	item.Properties = azureMigrateAssessmentMergeObjects(item.Properties, azureMigrateAssessmentObject(payload["properties"]))
	item.UpdatedAt = now
	item.ETag = azureMigrateAssessmentETag(key, body, now)
	s.azureMigrateAssessmentPrivateEndpointConnections[key] = item
	respondJSON(w, http.StatusOK, azureMigrateAssessmentPrivateEndpointConnectionFixture(item))
}

func (s *Server) handleAzureMigrateAssessmentPrivateLinkResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectName))
		return
	}
	values := azureMigrateAssessmentPrivateLinkResourceFixtures(projectID)
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": values})
		return
	}
	if len(tail) == 1 {
		for _, value := range values {
			if strings.EqualFold(azureMigrateAssessmentString(value["name"]), tail[0]) {
				respondJSON(w, http.StatusOK, value)
				return
			}
		}
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, tail[0]))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateAssessmentAKSOptionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	if r.Method != http.MethodGet || len(tail) > 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else {
			respondAzureImplemented(w, path)
		}
		return
	}
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	s.providerStorageMu.Lock()
	_, exists := s.azureMigrateAssessmentProjects[azureMigrateAssessmentProjectKey(subscriptionID, resourceGroup, projectName)]
	s.providerStorageMu.Unlock()
	if !exists {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectName))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":   projectID + "/aksOptions/default",
		"name": "default",
		"type": "Microsoft.Migrate/assessmentProjects/aksOptions",
		"properties": map[string]any{
			"targetLocations": []any{"eastus", "eastus2"},
			"currencies":      []any{"USD"},
		},
	})
}

func (s *Server) handleAzureMigrateAssessmentAKSSummaryRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, projectName string, tail []string) {
	if r.Method != http.MethodGet || len(tail) > 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else {
			respondAzureImplemented(w, path)
		}
		return
	}
	projectID := azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, projectName)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		respondJSON(w, http.StatusNotFound, azureMigrateAssessmentNotFound(path, projectName))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":   projectID + "/aksSummary/default",
		"name": "default",
		"type": "Microsoft.Migrate/assessmentProjects/aksSummary",
		"properties": map[string]any{
			"clustersCount":     len(s.azureMigrateAssessmentProjectMachinesLocked(projectID)),
			"readyClusterCount": len(s.azureMigrateAssessmentProjectMachinesLocked(projectID)),
			"monthlyCost": map[string]any{
				"amount":   123.45,
				"currency": "USD",
			},
		},
	})
}

func (s *Server) azureMigrateAssessmentEnsureProjectFixturesLocked(project *azureMigrateAssessmentProject) {
	projectID := azureMigrateAssessmentProjectID(project.SubscriptionID, project.ResourceGroup, project.Name)
	if len(s.azureMigrateAssessmentProjectMachinesLocked(projectID)) == 0 {
		now := time.Now().UTC()
		for idx, name := range []string{"machine-1", "machine-2"} {
			s.azureMigrateAssessmentMachines[azureMigrateAssessmentMachineKey(projectID, name)] = &azureMigrateAssessmentMachine{
				ProjectID: projectID,
				Name:      name,
				CreatedAt: now,
				UpdatedAt: now,
				Properties: map[string]any{
					"discoverySource": "Vmware",
					"groups":          []any{},
					"sqlInstances":    []any{},
					"webApplications": []any{},
					"workloadSummary": map[string]any{
						"webAppCount": idx,
						"sqlCount":    idx + 1,
					},
					"disks": []any{
						map[string]any{"name": "disk-os", "sizeInGB": 128},
					},
					"networkAdapters": []any{
						map[string]any{"name": "nic0", "ipAddresses": []any{"10.0.0.10"}},
					},
					"productSupportStatus": "Supported",
				},
			}
		}
	}
	if _, ok := s.azureMigrateAssessmentGroups[azureMigrateAssessmentGroupKey(projectID, "group-default")]; !ok {
		s.azureMigrateAssessmentGroups[azureMigrateAssessmentGroupKey(projectID, "group-default")] = &azureMigrateAssessmentGroup{
			ProjectID:    projectID,
			Name:         "group-default",
			MachineNames: []string{"machine-1", "machine-2"},
			Properties: map[string]any{
				"groupType": "Default",
			},
			CreatedAt: project.CreatedAt,
			UpdatedAt: project.UpdatedAt,
		}
	}
	s.azureMigrateAssessmentEnsurePrivateEndpointConnectionLocked(projectID, "default")
}

func (s *Server) azureMigrateAssessmentEnsurePrivateEndpointConnectionLocked(projectID, name string) *azureMigrateAssessmentPrivateEndpointConnection {
	if !s.azureMigrateAssessmentProjectExistsLocked(projectID) {
		return nil
	}
	key := azureMigrateAssessmentPrivateEndpointConnectionKey(projectID, name)
	if item := s.azureMigrateAssessmentPrivateEndpointConnections[key]; item != nil {
		return item
	}
	now := time.Now().UTC()
	item := &azureMigrateAssessmentPrivateEndpointConnection{
		ProjectID: projectID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		Properties: map[string]any{
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Approved",
				"description":     "Approved by Stackyard",
				"actionsRequired": "None",
			},
			"provisioningState": "Succeeded",
		},
	}
	s.azureMigrateAssessmentPrivateEndpointConnections[key] = item
	return item
}

func (s *Server) azureMigrateAssessmentPrivateEndpointConnectionsLocked(projectID string) []*azureMigrateAssessmentPrivateEndpointConnection {
	items := make([]*azureMigrateAssessmentPrivateEndpointConnection, 0)
	for _, item := range s.azureMigrateAssessmentPrivateEndpointConnections {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		if item := s.azureMigrateAssessmentEnsurePrivateEndpointConnectionLocked(projectID, "default"); item != nil {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (s *Server) azureMigrateAssessmentProjectExistsLocked(projectID string) bool {
	for _, project := range s.azureMigrateAssessmentProjects {
		if azureMigrateAssessmentProjectID(project.SubscriptionID, project.ResourceGroup, project.Name) == projectID {
			return true
		}
	}
	return false
}

func (s *Server) azureMigrateAssessmentProjectMachinesLocked(projectID string) []*azureMigrateAssessmentMachine {
	items := make([]*azureMigrateAssessmentMachine, 0)
	for _, item := range s.azureMigrateAssessmentMachines {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func azureMigrateAssessmentOperationsFixture() []map[string]any {
	names := []string{
		"Microsoft.Migrate/assessmentProjects/read",
		"Microsoft.Migrate/assessmentProjects/write",
		"Microsoft.Migrate/assessmentProjects/delete",
		"Microsoft.Migrate/assessmentProjects/groups/read",
		"Microsoft.Migrate/assessmentProjects/groups/write",
		"Microsoft.Migrate/assessmentProjects/groups/delete",
		"Microsoft.Migrate/assessmentProjects/groups/updateMachines/action",
		"Microsoft.Migrate/assessmentProjects/machines/read",
		"Microsoft.Migrate/assessmentProjects/assessmentOptions/read",
		"Microsoft.Migrate/assessmentProjects/projectSummary/read",
		"Microsoft.Migrate/assessmentProjects/groups/assessments/read",
		"Microsoft.Migrate/assessmentProjects/groups/assessments/write",
		"Microsoft.Migrate/assessmentProjects/groups/assessments/delete",
		"Microsoft.Migrate/assessmentProjects/groups/assessments/assessedMachines/read",
		"Microsoft.Migrate/assessmentProjects/vmwarecollectors/read",
		"Microsoft.Migrate/assessmentProjects/vmwarecollectors/write",
		"Microsoft.Migrate/assessmentProjects/privateEndpointConnections/read",
		"Microsoft.Migrate/assessmentProjects/privateEndpointConnections/write",
		"Microsoft.Migrate/assessmentProjects/privateLinkResources/read",
		"Microsoft.Migrate/assessmentProjects/aksOptions/read",
		"Microsoft.Migrate/assessmentProjects/aksSummary/read",
	}
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, map[string]any{
			"name":         name,
			"isDataAction": false,
			"origin":       "system",
			"actionType":   "Internal",
			"display": map[string]any{
				"provider":    "Microsoft.Migrate",
				"resource":    strings.TrimPrefix(name, "Microsoft.Migrate/"),
				"operation":   name,
				"description": "Stackyard Azure Migrate Assessment operation",
			},
		})
	}
	return out
}

func azureMigrateAssessmentProjectFixture(project *azureMigrateAssessmentProject) map[string]any {
	return map[string]any{
		"id":         azureMigrateAssessmentProjectID(project.SubscriptionID, project.ResourceGroup, project.Name),
		"name":       project.Name,
		"type":       "Microsoft.Migrate/assessmentProjects",
		"location":   project.Location,
		"tags":       azureMigrateAssessmentAnyStringMap(project.Tags),
		"systemData": azureMigrateAssessmentSystemData(project.CreatedAt, project.UpdatedAt),
		"properties": azureMigrateAssessmentCloneObject(project.Properties),
	}
}

func azureMigrateAssessmentGroupFixture(group *azureMigrateAssessmentGroup) map[string]any {
	return map[string]any{
		"id":         group.ProjectID + "/groups/" + group.Name,
		"name":       group.Name,
		"type":       "Microsoft.Migrate/assessmentProjects/groups",
		"systemData": azureMigrateAssessmentSystemData(group.CreatedAt, group.UpdatedAt),
		"properties": map[string]any{
			"groupType":    azureMigrateAssessmentString(group.Properties["groupType"]),
			"machines":     append([]string(nil), group.MachineNames...),
			"machineCount": len(group.MachineNames),
		},
	}
}

func azureMigrateAssessmentMachineFixture(machine *azureMigrateAssessmentMachine) map[string]any {
	return map[string]any{
		"id":         machine.ProjectID + "/machines/" + machine.Name,
		"name":       machine.Name,
		"type":       "Microsoft.Migrate/assessmentProjects/machines",
		"systemData": azureMigrateAssessmentSystemData(machine.CreatedAt, machine.UpdatedAt),
		"properties": azureMigrateAssessmentCloneObject(machine.Properties),
	}
}

func azureMigrateAssessmentFixture(item *azureMigrateAssessment) map[string]any {
	return map[string]any{
		"id":         item.ProjectID + "/groups/" + item.GroupName + "/assessments/" + item.Name,
		"name":       item.Name,
		"type":       "Microsoft.Migrate/assessmentProjects/groups/assessments",
		"systemData": azureMigrateAssessmentSystemData(item.CreatedAt, item.UpdatedAt),
		"properties": azureMigrateAssessmentCloneObject(item.Properties),
	}
}

func azureMigrateAssessmentAssessedMachineFixture(projectID, groupName, assessmentName string, machine *azureMigrateAssessmentMachine) map[string]any {
	return map[string]any{
		"id":   projectID + "/groups/" + groupName + "/assessments/" + assessmentName + "/assessedMachines/" + machine.Name,
		"name": machine.Name,
		"type": "Microsoft.Migrate/assessmentProjects/groups/assessments/assessedMachines",
		"properties": map[string]any{
			"machineArmId":       machine.ProjectID + "/machines/" + machine.Name,
			"suitability":        "Ready",
			"monthlyComputeCost": map[string]any{"amount": 42.0, "currency": "USD"},
			"recommendedSize":    "Standard_D2s_v5",
			"targetLocations":    []any{"eastus"},
		},
	}
}

func azureMigrateAssessmentCollectorFixture(item *azureMigrateAssessmentCollector) map[string]any {
	return map[string]any{
		"id":         item.ProjectID + "/" + item.Collection + "/" + item.Name,
		"name":       item.Name,
		"type":       "Microsoft.Migrate/assessmentProjects/" + azureMigrateAssessmentCollectorCollections[item.Collection],
		"systemData": azureMigrateAssessmentSystemData(item.CreatedAt, item.UpdatedAt),
		"properties": azureMigrateAssessmentCloneObject(item.Properties),
	}
}

func azureMigrateAssessmentPrivateEndpointConnectionFixture(item *azureMigrateAssessmentPrivateEndpointConnection) map[string]any {
	return map[string]any{
		"id":         item.ProjectID + "/privateEndpointConnections/" + item.Name,
		"name":       item.Name,
		"type":       "Microsoft.Migrate/assessmentProjects/privateEndpointConnections",
		"systemData": azureMigrateAssessmentSystemData(item.CreatedAt, item.UpdatedAt),
		"properties": azureMigrateAssessmentCloneObject(item.Properties),
	}
}

func azureMigrateAssessmentPrivateLinkResourceFixtures(projectID string) []map[string]any {
	return []map[string]any{
		{
			"id":   projectID + "/privateLinkResources/default",
			"name": "default",
			"type": "Microsoft.Migrate/assessmentProjects/privateLinkResources",
			"properties": map[string]any{
				"groupId":         "assessmentProject",
				"requiredMembers": []any{"assessmentProject"},
				"requiredZoneNames": []any{
					"privatelink.migrate.azure.com",
				},
			},
		},
	}
}

func azureMigrateAssessmentProjectSummaryFixture(s *Server, projectID string) map[string]any {
	groupCount := 0
	assessmentCount := 0
	machineCount := len(s.azureMigrateAssessmentProjectMachinesLocked(projectID))
	for _, group := range s.azureMigrateAssessmentGroups {
		if group.ProjectID == projectID {
			groupCount++
		}
	}
	for _, assessment := range s.azureMigrateAssessments {
		if assessment.ProjectID == projectID {
			assessmentCount++
		}
	}
	return map[string]any{
		"id":   projectID + "/projectSummary/default",
		"name": "default",
		"type": "Microsoft.Migrate/assessmentProjects/projectSummary",
		"properties": map[string]any{
			"groupsCount":      groupCount,
			"machinesCount":    machineCount,
			"assessmentsCount": assessmentCount,
		},
	}
}

func azureMigrateAssessmentNormalizeProjectProperties(name, location string, properties map[string]any) map[string]any {
	out := azureMigrateAssessmentCloneObject(properties)
	if strings.TrimSpace(azureMigrateAssessmentString(out["provisioningState"])) == "" {
		out["provisioningState"] = "Succeeded"
	}
	if strings.TrimSpace(azureMigrateAssessmentString(out["projectStatus"])) == "" {
		out["projectStatus"] = "Active"
	}
	if strings.TrimSpace(azureMigrateAssessmentString(out["endpoint"])) == "" {
		out["endpoint"] = fmt.Sprintf("https://%s.%s.migrate.azure.example", strings.ToLower(name), strings.ToLower(location))
	}
	return out
}

func azureMigrateAssessmentNormalizeGroupProperties(properties map[string]any) map[string]any {
	out := azureMigrateAssessmentCloneObject(properties)
	if strings.TrimSpace(azureMigrateAssessmentString(out["groupType"])) == "" {
		out["groupType"] = "Custom"
	}
	return out
}

func azureMigrateAssessmentNormalizeAssessmentProperties(properties map[string]any) map[string]any {
	out := azureMigrateAssessmentCloneObject(properties)
	if strings.TrimSpace(azureMigrateAssessmentString(out["azureLocation"])) == "" {
		out["azureLocation"] = "eastus"
	}
	if strings.TrimSpace(azureMigrateAssessmentString(out["assessmentType"])) == "" {
		out["assessmentType"] = "Machine"
	}
	if _, ok := out["monthlyCost"]; !ok {
		out["monthlyCost"] = map[string]any{"amount": 99.9, "currency": "USD"}
	}
	if strings.TrimSpace(azureMigrateAssessmentString(out["suitability"])) == "" {
		out["suitability"] = "Ready"
	}
	return out
}

func azureMigrateAssessmentNormalizeCollectorProperties(collection string, properties map[string]any) map[string]any {
	out := azureMigrateAssessmentCloneObject(properties)
	if strings.TrimSpace(azureMigrateAssessmentString(out["provisioningState"])) == "" {
		out["provisioningState"] = "Succeeded"
	}
	if strings.TrimSpace(azureMigrateAssessmentString(out["discoverySiteId"])) == "" {
		out["discoverySiteId"] = "site-" + collection
	}
	return out
}

func azureMigrateAssessmentMembershipNames(payload map[string]any) []string {
	candidates := []any{
		payload["machines"],
		payload["machineNames"],
		payload["machineIds"],
		azureMigrateAssessmentObject(payload["properties"])["machines"],
		azureMigrateAssessmentObject(payload["properties"])["machineNames"],
	}
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		for _, value := range azureMigrateAssessmentStringSlice(candidate) {
			name := strings.TrimSpace(value)
			if name == "" {
				continue
			}
			if idx := strings.LastIndex(name, "/"); idx >= 0 {
				name = name[idx+1:]
			}
			lower := strings.ToLower(name)
			if _, ok := seen[lower]; ok {
				continue
			}
			seen[lower] = struct{}{}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func azureMigrateAssessmentReadJSONBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureMigrateAssessmentProjectKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "::" + resourceGroup + "::" + name)
}

func azureMigrateAssessmentGroupKey(projectID, name string) string {
	return strings.ToLower(projectID + "::" + name)
}

func azureMigrateAssessmentMachineKey(projectID, name string) string {
	return strings.ToLower(projectID + "::" + name)
}

func azureMigrateAssessmentAssessmentKey(projectID, groupName, name string) string {
	return strings.ToLower(projectID + "::" + groupName + "::" + name)
}

func azureMigrateAssessmentCollectorKey(projectID, collection, name string) string {
	return strings.ToLower(projectID + "::" + collection + "::" + name)
}

func azureMigrateAssessmentPrivateEndpointConnectionKey(projectID, name string) string {
	return strings.ToLower(projectID + "::pec::" + name)
}

func azureMigrateAssessmentProjectID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Migrate/assessmentProjects/%s", subscriptionID, resourceGroup, name)
}

func azureMigrateAssessmentNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("%s not found for path %s", name, path),
		},
	}
}

func azureMigrateAssessmentETag(key string, body []byte, updatedAt time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+updatedAt.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureMigrateAssessmentSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdBy":          "Stackyard",
		"createdByType":      "Application",
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"lastModifiedBy":     "Stackyard",
		"lastModifiedByType": "Application",
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
	}
}

func azureMigrateAssessmentString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func azureMigrateAssessmentStringMap(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for k, v := range typed {
			out[k] = v
		}
	case map[string]any:
		for k, v := range typed {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

func azureMigrateAssessmentAnyStringMap(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func azureMigrateAssessmentObject(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureMigrateAssessmentCloneObject(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azureMigrateAssessmentCloneObject(nested)
			continue
		}
		if list, ok := value.([]any); ok {
			cloned := make([]any, len(list))
			copy(cloned, list)
			out[key] = cloned
			continue
		}
		out[key] = value
	}
	return out
}

func azureMigrateAssessmentMergeObjects(base, patch map[string]any) map[string]any {
	out := azureMigrateAssessmentCloneObject(base)
	for key, value := range patch {
		if nested, ok := value.(map[string]any); ok {
			if existing, ok := out[key].(map[string]any); ok {
				out[key] = azureMigrateAssessmentMergeObjects(existing, nested)
			} else {
				out[key] = azureMigrateAssessmentCloneObject(nested)
			}
			continue
		}
		out[key] = value
	}
	return out
}

func azureMigrateAssessmentStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}
