package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gcpTelcoAutomationReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	gcpTelcoAutomationProjectIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	gcpTelcoAutomationLocationPattern  = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	gcpTelcoAutomationIDPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	gcpTelcoAutomationRequestIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)

func (s *Server) handleGCPTelcoAutomationRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_telcoautomation(w, r) {
		return true
	}

	path := normalizeGCPTelcoAutomationPath(rawRequestPath(r))

	if isGCPTelcoAutomationLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPTelcoAutomationListLocations(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPTelcoAutomationPath(path, hasGCPTelcoAutomationHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPTelcoAutomationListOrchestrationClusters(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationGetOrchestrationCluster(w, path) {
			return true
		}
		if handleGCPTelcoAutomationListEdgeSlms(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationGetEdgeSlm(w, path) {
			return true
		}
		if handleGCPTelcoAutomationGetBlueprint(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationListBlueprints(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationListBlueprintRevisions(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationSearchBlueprintRevisions(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationSearchDeploymentRevisions(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationGetPublicBlueprint(w, path) {
			return true
		}
		if handleGCPTelcoAutomationListPublicBlueprints(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationGetDeployment(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationListDeployments(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationListDeploymentRevisions(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationComputeDeploymentStatus(w, path) {
			return true
		}
		if handleGCPTelcoAutomationGetHydratedDeployment(w, path) {
			return true
		}
		if handleGCPTelcoAutomationListHydratedDeployments(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationListOperations(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPTelcoAutomationCreateOrchestrationCluster(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationCreateEdgeSlm(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationCreateBlueprint(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationApproveBlueprint(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationProposeBlueprint(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationRejectBlueprint(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationDiscardBlueprintChanges(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationCreateDeployment(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationRemoveDeployment(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationDiscardDeploymentChanges(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationApplyDeployment(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationRollbackDeployment(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationApplyHydratedDeployment(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPTelcoAutomationUpdateBlueprint(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationUpdateDeployment(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationUpdateHydratedDeployment(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPTelcoAutomationDeleteOrchestrationCluster(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationDeleteEdgeSlm(w, r, path) {
			return true
		}
		if handleGCPTelcoAutomationDeleteBlueprint(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPTelcoAutomationPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPTelcoAutomationHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "telcoautomation",
		"telcoautomation-apiv1",
		"telcoautomation_apiv1",
		"telco-automation",
		"telco_automation",
		"gcp-telcoautomation",
		"gcp-telco-automation":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-telcoautomation-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/telcoautomation/apiv1")
}

func isGCPTelcoAutomationGRPCPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/gcp/google.cloud.telcoautomation.v1.TelcoAutomation/")
}

func isGCPTelcoAutomationLocationRequest(r *http.Request, path string) bool {
	return hasGCPTelcoAutomationHint(r) && isGCPProjectLocationDiscoveryPath(path)
}

func isGCPTelcoAutomationPath(path string, includeHint bool) bool {
	if isGCPTelcoAutomationGRPCPath(path) {
		return true
	}

	_, _, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) == 0 {
		return false
	}

	if len(tail) >= 1 {
		switch tail[0] {
		case "orchestrationClusters", "edgeSlms", "publicBlueprints":
			return true
		case "operations":
			return includeHint
		}
	}
	if len(tail) >= 3 && tail[0] == "orchestrationClusters" {
		return true
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/projects/")
}

func handleGCPTelcoAutomationListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationLocation(project, "us-central1"),
		gcpTelcoAutomationLocation(project, "global"),
	}
	return respondGCPTelcoAutomationList(w, "locations", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationLocation(project, location))
	return true
}

func handleGCPTelcoAutomationListOrchestrationClusters(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || !isGCPTelcoAutomationOrchestrationClustersCollectionTail(tail) {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter != "" && !isGCPTelcoAutomationSimpleStateFilter(filter, []string{"ACTIVE", "CREATING", "FAILED", "DELETING"}) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "filter must be state = ACTIVE|CREATING|FAILED|DELETING")
		return true
	}
	orderBy := strings.TrimSpace(r.URL.Query().Get("orderBy"))
	if orderBy != "" && orderBy != "name" && orderBy != "create_time" && orderBy != "create_time desc" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "orderBy must be one of name, create_time, create_time desc")
		return true
	}

	items := []map[string]any{
		gcpTelcoAutomationOrchestrationCluster(project, location, "cluster-1", 2),
		gcpTelcoAutomationOrchestrationCluster(project, location, "cluster-2", 1),
	}
	sort.Slice(items, func(i, j int) bool {
		return gcpTelcoAutomationString(items[i], "name") < gcpTelcoAutomationString(items[j], "name")
	})
	return respondGCPTelcoAutomationList(w, "orchestrationClusters", items, pageSize, start, path, true)
}

func handleGCPTelcoAutomationGetOrchestrationCluster(w http.ResponseWriter, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterPath(path)
	if !ok {
		return false
	}
	if isGCPTelcoAutomationMissingID(clusterID) {
		respondGCPTelcoAutomationNotFound(w, path, "orchestration cluster not found")
		return true
	}
	state := int32(2)
	if strings.Contains(strings.ToLower(clusterID), "creating") {
		state = 1
	}
	if strings.Contains(strings.ToLower(clusterID), "failed") {
		state = 4
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationOrchestrationCluster(project, location, clusterID, state))
	return true
}

func handleGCPTelcoAutomationCreateOrchestrationCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || !isGCPTelcoAutomationOrchestrationClustersCollectionTail(tail) {
		return false
	}
	clusterID := strings.TrimSpace(r.URL.Query().Get("orchestrationClusterId"))
	if clusterID == "" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "orchestrationClusterId is required")
		return true
	}
	if !isGCPTelcoAutomationResourceID(clusterID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "orchestrationClusterId is invalid")
		return true
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	clusterName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != clusterName {
		respondGCPTelcoAutomationInvalidArgument(w, path, "orchestrationCluster.name must match parent and orchestrationClusterId")
		return true
	}

	cluster := gcpTelcoAutomationOrchestrationCluster(project, location, clusterID, 2)
	applyGCPTelcoAutomationOrchestrationClusterOverrides(cluster, body)
	respondJSON(w, http.StatusOK, gcpTelcoAutomationOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/createOrchestrationCluster.%s", project, location, clusterID),
		"create",
		clusterName,
		map[string]any{
			"@type": "type.googleapis.com/google.cloud.telcoautomation.v1.OrchestrationCluster",
			"name":  cluster["name"],
			"managementConfig": map[string]any{
				"fullManagementConfig": map[string]any{},
			},
			"createTime": cluster["createTime"],
			"updateTime": cluster["updateTime"],
			"labels":     cluster["labels"],
			"tnaVersion": cluster["tnaVersion"],
			"state":      cluster["state"],
		},
	))
	return true
}

func handleGCPTelcoAutomationDeleteOrchestrationCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterPath(path)
	if !ok {
		return false
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}
	if isGCPTelcoAutomationMissingID(clusterID) {
		respondGCPTelcoAutomationNotFound(w, path, "orchestration cluster not found")
		return true
	}
	clusterName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
	respondJSON(w, http.StatusOK, gcpTelcoAutomationOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/deleteOrchestrationCluster.%s", project, location, clusterID),
		"delete",
		clusterName,
		map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	))
	return true
}

func handleGCPTelcoAutomationListEdgeSlms(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || !isGCPTelcoAutomationEdgeSlmsCollectionTail(tail) {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter != "" && !isGCPTelcoAutomationSimpleStateFilter(filter, []string{"ACTIVE", "CREATING", "FAILED", "DELETING"}) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "filter must be state = ACTIVE|CREATING|FAILED|DELETING")
		return true
	}
	orderBy := strings.TrimSpace(r.URL.Query().Get("orderBy"))
	if orderBy != "" && orderBy != "name" && orderBy != "create_time" && orderBy != "create_time desc" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "orderBy must be one of name, create_time, create_time desc")
		return true
	}

	items := []map[string]any{
		gcpTelcoAutomationEdgeSlm(project, location, "edgeslm-1", "cluster-1", 2),
		gcpTelcoAutomationEdgeSlm(project, location, "edgeslm-2", "cluster-2", 1),
	}
	sort.Slice(items, func(i, j int) bool {
		return gcpTelcoAutomationString(items[i], "name") < gcpTelcoAutomationString(items[j], "name")
	})
	return respondGCPTelcoAutomationList(w, "edgeSlms", items, pageSize, start, path, true)
}

func handleGCPTelcoAutomationGetEdgeSlm(w http.ResponseWriter, path string) bool {
	project, location, edgeID, ok := parseGCPTelcoAutomationEdgeSlmPath(path)
	if !ok {
		return false
	}
	if isGCPTelcoAutomationMissingID(edgeID) {
		respondGCPTelcoAutomationNotFound(w, path, "edge slm not found")
		return true
	}
	state := int32(2)
	if strings.Contains(strings.ToLower(edgeID), "creating") {
		state = 1
	}
	if strings.Contains(strings.ToLower(edgeID), "failed") {
		state = 4
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationEdgeSlm(project, location, edgeID, "cluster-1", state))
	return true
}

func handleGCPTelcoAutomationCreateEdgeSlm(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || !isGCPTelcoAutomationEdgeSlmsCollectionTail(tail) {
		return false
	}
	edgeID := strings.TrimSpace(r.URL.Query().Get("edgeSlmId"))
	if edgeID == "" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "edgeSlmId is required")
		return true
	}
	if !isGCPTelcoAutomationResourceID(edgeID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "edgeSlmId is invalid")
		return true
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	edgeName := fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != edgeName {
		respondGCPTelcoAutomationInvalidArgument(w, path, "edgeSlm.name must match parent and edgeSlmId")
		return true
	}
	clusterName := gcpTelcoAutomationString(body, "orchestrationCluster")
	if clusterName == "" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "edgeSlm.orchestrationCluster is required")
		return true
	}
	if _, _, _, ok := parseGCPTelcoAutomationOrchestrationClusterName(clusterName); !ok {
		respondGCPTelcoAutomationInvalidArgument(w, path, "edgeSlm.orchestrationCluster is invalid")
		return true
	}

	edge := gcpTelcoAutomationEdgeSlm(project, location, edgeID, pathBase(clusterName), 2)
	applyGCPTelcoAutomationEdgeSlmOverrides(edge, body)
	respondJSON(w, http.StatusOK, gcpTelcoAutomationOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/createEdgeSlm.%s", project, location, edgeID),
		"create",
		edgeName,
		map[string]any{
			"@type":                "type.googleapis.com/google.cloud.telcoautomation.v1.EdgeSlm",
			"name":                 edge["name"],
			"orchestrationCluster": edge["orchestrationCluster"],
			"createTime":           edge["createTime"],
			"updateTime":           edge["updateTime"],
			"labels":               edge["labels"],
			"tnaVersion":           edge["tnaVersion"],
			"state":                edge["state"],
			"workloadClusterType":  edge["workloadClusterType"],
		},
	))
	return true
}

func handleGCPTelcoAutomationDeleteEdgeSlm(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, edgeID, ok := parseGCPTelcoAutomationEdgeSlmPath(path)
	if !ok {
		return false
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}
	if isGCPTelcoAutomationMissingID(edgeID) {
		respondGCPTelcoAutomationNotFound(w, path, "edge slm not found")
		return true
	}
	edgeName := fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID)
	respondJSON(w, http.StatusOK, gcpTelcoAutomationOperation(
		fmt.Sprintf("projects/%s/locations/%s/operations/deleteEdgeSlm.%s", project, location, edgeID),
		"delete",
		edgeName,
		map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	))
	return true
}

func handleGCPTelcoAutomationCreateBlueprint(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationBlueprintParentPath(path)
	if !ok {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	blueprintID := strings.TrimSpace(r.URL.Query().Get("blueprintId"))
	if blueprintID == "" {
		blueprintID = "blueprint-created-1"
	}
	if !isGCPTelcoAutomationResourceID(blueprintID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "blueprintId is invalid")
		return true
	}
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s", project, location, clusterID, blueprintID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != name {
		respondGCPTelcoAutomationInvalidArgument(w, path, "blueprint.name must match parent and blueprintId")
		return true
	}
	if sourceBlueprint := gcpTelcoAutomationString(body, "sourceBlueprint"); sourceBlueprint == "" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "blueprint.sourceBlueprint is required")
		return true
	}

	blueprint := gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-1", 1)
	applyGCPTelcoAutomationBlueprintOverrides(blueprint, body)
	respondJSON(w, http.StatusOK, blueprint)
	return true
}

func handleGCPTelcoAutomationUpdateBlueprint(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, blueprintID, action, ok := parseGCPTelcoAutomationBlueprintPath(path)
	if !ok || action != "" {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s", project, location, clusterID, blueprintID)
	if provided := gcpTelcoAutomationString(body, "name"); provided == "" || provided != name {
		respondGCPTelcoAutomationInvalidArgument(w, path, "blueprint.name must match requested resource")
		return true
	}
	updateMask, ok := parseGCPTelcoAutomationUpdateMask(r, body)
	if !ok || len(updateMask) == 0 {
		respondGCPTelcoAutomationInvalidArgument(w, path, "updateMask is required")
		return true
	}
	allowed := map[string]struct{}{
		"display_name": {}, "displayName": {},
		"files": {}, "labels": {}, "*": {},
	}
	for _, field := range updateMask {
		if _, exists := allowed[field]; !exists {
			respondGCPTelcoAutomationInvalidArgument(w, path, "updateMask has unsupported fields")
			return true
		}
	}

	state := gcpTelcoAutomationBlueprintStateForName(blueprintID)
	blueprint := gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-2", state)
	applyGCPTelcoAutomationBlueprintOverrides(blueprint, body)
	respondJSON(w, http.StatusOK, blueprint)
	return true
}

func handleGCPTelcoAutomationGetBlueprint(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, blueprintID, action, ok := parseGCPTelcoAutomationBlueprintPath(path)
	if !ok || action != "" {
		return false
	}
	if strings.Contains(strings.ToLower(blueprintID), "missing") {
		respondGCPTelcoAutomationNotFound(w, path, "blueprint not found")
		return true
	}
	plainBlueprintID, revisionID := splitGCPTelcoAutomationRevision(blueprintID)
	if !isGCPTelcoAutomationResourceID(plainBlueprintID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name is invalid")
		return true
	}
	if revisionID != "" && !isGCPTelcoAutomationResourceID(revisionID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "revision id is invalid")
		return true
	}
	state := gcpTelcoAutomationBlueprintStateForName(plainBlueprintID)
	if revisionID == "" {
		switch state {
		case 1:
			revisionID = "rev-1"
		case 2:
			revisionID = "rev-2"
		default:
			revisionID = "rev-3"
		}
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationBlueprint(project, location, clusterID, plainBlueprintID, revisionID, state))
	return true
}

func handleGCPTelcoAutomationDeleteBlueprint(w http.ResponseWriter, path string) bool {
	_, _, _, blueprintID, action, ok := parseGCPTelcoAutomationBlueprintPath(path)
	if !ok || action != "" {
		return false
	}
	if strings.Contains(blueprintID, "@") {
		respondGCPTelcoAutomationInvalidArgument(w, path, "blueprint name must not include revision id for delete")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTelcoAutomationListBlueprints(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationBlueprintParentPath(path)
	if !ok {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter != "" && !isGCPTelcoAutomationSimpleStateFilter(filter, []string{"DRAFT", "PROPOSED", "APPROVED"}) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "filter only supports state equality")
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationBlueprint(project, location, clusterID, "blueprint-draft", "rev-1", 1),
		gcpTelcoAutomationBlueprint(project, location, clusterID, "blueprint-proposed", "rev-2", 2),
		gcpTelcoAutomationBlueprint(project, location, clusterID, "blueprint-approved", "rev-3", 3),
	}
	items = filterGCPTelcoAutomationBlueprints(items, filter)
	return respondGCPTelcoAutomationList(w, "blueprints", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationApproveBlueprint(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPTelcoAutomationBlueprintAction(w, r, path, "approve")
}

func handleGCPTelcoAutomationProposeBlueprint(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPTelcoAutomationBlueprintAction(w, r, path, "propose")
}

func handleGCPTelcoAutomationRejectBlueprint(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPTelcoAutomationBlueprintAction(w, r, path, "reject")
}

func handleGCPTelcoAutomationBlueprintAction(w http.ResponseWriter, r *http.Request, path, action string) bool {
	project, location, clusterID, blueprintID, parsedAction, ok := parseGCPTelcoAutomationBlueprintPath(path)
	if !ok || parsedAction != action {
		return false
	}
	if isGCPTelcoAutomationMissingID(blueprintID) {
		respondGCPTelcoAutomationNotFound(w, path, "blueprint not found")
		return true
	}
	body, ok := decodeGCPTelcoAutomationJSONBody(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s", project, location, clusterID, blueprintID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != expectedName {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name must match requested resource")
		return true
	}

	currentState := gcpTelcoAutomationBlueprintStateForName(blueprintID)
	switch action {
	case "propose":
		if currentState != 1 {
			respondGCPTelcoAutomationFailedPrecondition(w, path, "blueprint must be in DRAFT state to propose")
			return true
		}
		respondJSON(w, http.StatusOK, gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-2", 2))
	case "approve":
		if currentState != 2 {
			respondGCPTelcoAutomationFailedPrecondition(w, path, "blueprint must be in PROPOSED state to approve")
			return true
		}
		respondJSON(w, http.StatusOK, gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-3", 3))
	case "reject":
		if currentState != 2 {
			respondGCPTelcoAutomationFailedPrecondition(w, path, "blueprint must be in PROPOSED state to reject")
			return true
		}
		respondJSON(w, http.StatusOK, gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-2", 1))
	}
	return true
}

func handleGCPTelcoAutomationListBlueprintRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, blueprintID, action, ok := parseGCPTelcoAutomationBlueprintPath(path)
	if !ok || action != "listRevisions" {
		return false
	}
	if strings.Contains(blueprintID, "@") {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name must not include revision id")
		return true
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-1", 1),
		gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-2", 3),
	}
	return respondGCPTelcoAutomationList(w, "blueprints", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationSearchBlueprintRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationBlueprintSearchPath(path)
	if !ok {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 100)
	if !ok {
		return true
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if !isGCPTelcoAutomationRevisionSearchQuery(query, true) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "query is invalid")
		return true
	}

	items := []map[string]any{
		gcpTelcoAutomationBlueprint(project, location, clusterID, "blueprint-draft", "rev-1", 1),
		gcpTelcoAutomationBlueprint(project, location, clusterID, "blueprint-approved", "rev-3", 3),
	}
	items = filterGCPTelcoAutomationBlueprintRevisionSearch(items, query)
	return respondGCPTelcoAutomationList(w, "blueprints", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationDiscardBlueprintChanges(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, _, action, ok := parseGCPTelcoAutomationBlueprintPath(path)
	if !ok || action != "discard" {
		return false
	}
	if _, ok := decodeGCPTelcoAutomationJSONBody(w, r, path); !ok {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTelcoAutomationListPublicBlueprints(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || !isGCPTelcoAutomationPublicBlueprintsCollectionTail(tail) {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationPublicBlueprint(project, location, "public-blueprint-1"),
		gcpTelcoAutomationPublicBlueprint(project, location, "public-blueprint-2"),
	}
	return respondGCPTelcoAutomationList(w, "publicBlueprints", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationGetPublicBlueprint(w http.ResponseWriter, path string) bool {
	project, location, publicBlueprintID, ok := parseGCPTelcoAutomationPublicBlueprintPath(path)
	if !ok {
		return false
	}
	if isGCPTelcoAutomationMissingID(publicBlueprintID) {
		respondGCPTelcoAutomationNotFound(w, path, "public blueprint not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationPublicBlueprint(project, location, publicBlueprintID))
	return true
}

func handleGCPTelcoAutomationCreateDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationDeploymentParentPath(path)
	if !ok {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	deploymentID := strings.TrimSpace(r.URL.Query().Get("deploymentId"))
	if deploymentID == "" {
		deploymentID = "deployment-created-1"
	}
	if !isGCPTelcoAutomationResourceID(deploymentID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "deploymentId is invalid")
		return true
	}
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != name {
		respondGCPTelcoAutomationInvalidArgument(w, path, "deployment.name must match parent and deploymentId")
		return true
	}
	if sourceBlueprintRevision := gcpTelcoAutomationString(body, "sourceBlueprintRevision"); sourceBlueprintRevision == "" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "deployment.sourceBlueprintRevision is required")
		return true
	}
	deployment := gcpTelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-1", 1)
	applyGCPTelcoAutomationDeploymentOverrides(deployment, body)
	respondJSON(w, http.StatusOK, deployment)
	return true
}

func handleGCPTelcoAutomationUpdateDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "" {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if provided := gcpTelcoAutomationString(body, "name"); provided == "" || provided != name {
		respondGCPTelcoAutomationInvalidArgument(w, path, "deployment.name must match requested resource")
		return true
	}
	updateMask, ok := parseGCPTelcoAutomationUpdateMask(r, body)
	if !ok || len(updateMask) == 0 {
		respondGCPTelcoAutomationInvalidArgument(w, path, "updateMask is required")
		return true
	}
	allowed := map[string]struct{}{
		"display_name": {}, "displayName": {},
		"files": {}, "labels": {},
		"source_blueprint_revision": {}, "sourceBlueprintRevision": {},
		"workload_cluster": {}, "workloadCluster": {},
		"*": {},
	}
	for _, field := range updateMask {
		if _, exists := allowed[field]; !exists {
			respondGCPTelcoAutomationInvalidArgument(w, path, "updateMask has unsupported fields")
			return true
		}
	}
	if sourceBlueprintRevision := gcpTelcoAutomationString(body, "sourceBlueprintRevision"); sourceBlueprintRevision == "" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "deployment.sourceBlueprintRevision is required")
		return true
	}
	state := gcpTelcoAutomationDeploymentStateForName(deploymentID)
	deployment := gcpTelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-2", state)
	applyGCPTelcoAutomationDeploymentOverrides(deployment, body)
	respondJSON(w, http.StatusOK, deployment)
	return true
}

func handleGCPTelcoAutomationGetDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "" {
		return false
	}
	if strings.Contains(strings.ToLower(deploymentID), "missing") {
		respondGCPTelcoAutomationNotFound(w, path, "deployment not found")
		return true
	}
	plainDeploymentID, revisionID := splitGCPTelcoAutomationRevision(deploymentID)
	if !isGCPTelcoAutomationResourceID(plainDeploymentID) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name is invalid")
		return true
	}
	if revisionID == "" {
		revisionID = "rev-2"
	}
	state := gcpTelcoAutomationDeploymentStateForName(plainDeploymentID)
	respondJSON(w, http.StatusOK, gcpTelcoAutomationDeployment(project, location, clusterID, plainDeploymentID, revisionID, state))
	return true
}

func handleGCPTelcoAutomationRemoveDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "remove" {
		return false
	}
	if _, ok := decodeGCPTelcoAutomationJSONBody(w, r, path); !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if state := gcpTelcoAutomationDeploymentStateForName(deploymentID); state == 3 {
		respondGCPTelcoAutomationFailedPrecondition(w, path, "deployment is already deleting")
		return true
	}
	if isGCPTelcoAutomationMissingID(deploymentID) {
		respondGCPTelcoAutomationNotFound(w, path, "deployment not found")
		return true
	}
	_ = expectedName
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTelcoAutomationListDeployments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationDeploymentParentPath(path)
	if !ok {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter != "" && !isGCPTelcoAutomationSimpleStateFilter(filter, []string{"DRAFT", "APPLIED", "DELETING"}) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "filter only supports state equality")
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationDeployment(project, location, clusterID, "deployment-draft", "rev-1", 1),
		gcpTelcoAutomationDeployment(project, location, clusterID, "deployment-applied", "rev-2", 2),
		gcpTelcoAutomationDeployment(project, location, clusterID, "deployment-deleting", "rev-3", 3),
	}
	items = filterGCPTelcoAutomationDeployments(items, filter)
	return respondGCPTelcoAutomationList(w, "deployments", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationListDeploymentRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "listRevisions" {
		return false
	}
	if strings.Contains(deploymentID, "@") {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name must not include revision id")
		return true
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-1", 1),
		gcpTelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-2", 2),
	}
	return respondGCPTelcoAutomationList(w, "deployments", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationSearchDeploymentRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPTelcoAutomationDeploymentSearchPath(path)
	if !ok {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 100)
	if !ok {
		return true
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if !isGCPTelcoAutomationRevisionSearchQuery(query, false) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "query is invalid")
		return true
	}

	items := []map[string]any{
		gcpTelcoAutomationDeployment(project, location, clusterID, "deployment-draft", "rev-1", 1),
		gcpTelcoAutomationDeployment(project, location, clusterID, "deployment-applied", "rev-2", 2),
	}
	items = filterGCPTelcoAutomationDeploymentRevisionSearch(items, query)
	return respondGCPTelcoAutomationList(w, "deployments", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationDiscardDeploymentChanges(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, _, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "discard" {
		return false
	}
	if _, ok := decodeGCPTelcoAutomationJSONBody(w, r, path); !ok {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTelcoAutomationApplyDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "apply" {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBody(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != expectedName {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	if state := gcpTelcoAutomationDeploymentStateForName(deploymentID); state != 1 {
		respondGCPTelcoAutomationFailedPrecondition(w, path, "deployment must be in DRAFT state to apply")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-applied-1", 2))
	return true
}

func handleGCPTelcoAutomationComputeDeploymentStatus(w http.ResponseWriter, path string) bool {
	project, location, clusterID, deploymentID, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "computeDeploymentStatus" {
		return false
	}
	if isGCPTelcoAutomationMissingID(deploymentID) {
		respondGCPTelcoAutomationNotFound(w, path, "deployment not found")
		return true
	}
	deploymentName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	respondJSON(w, http.StatusOK, gcpTelcoAutomationComputeDeploymentStatus(deploymentName))
	return true
}

func handleGCPTelcoAutomationRollbackDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, action, ok := parseGCPTelcoAutomationDeploymentPath(path)
	if !ok || action != "rollback" {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != expectedName {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	revisionID := strings.TrimSpace(gcpTelcoAutomationString(body, "revisionId"))
	if revisionID == "" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "revisionId is required")
		return true
	}
	if state := gcpTelcoAutomationDeploymentStateForName(deploymentID); state != 2 {
		respondGCPTelcoAutomationFailedPrecondition(w, path, "deployment must be in APPLIED state to rollback")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationDeployment(project, location, clusterID, deploymentID, revisionID, 2))
	return true
}

func handleGCPTelcoAutomationGetHydratedDeployment(w http.ResponseWriter, path string) bool {
	project, location, clusterID, deploymentID, hydratedID, action, ok := parseGCPTelcoAutomationHydratedDeploymentPath(path)
	if !ok || action != "" {
		return false
	}
	if isGCPTelcoAutomationMissingID(hydratedID) {
		respondGCPTelcoAutomationNotFound(w, path, "hydrated deployment not found")
		return true
	}
	state := int32(1)
	if strings.Contains(strings.ToLower(hydratedID), "applied") {
		state = 2
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID, state))
	return true
}

func handleGCPTelcoAutomationListHydratedDeployments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, ok := parseGCPTelcoAutomationHydratedDeploymentParentPath(path)
	if !ok {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, "hydrated-draft", 1),
		gcpTelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, "hydrated-applied", 2),
	}
	return respondGCPTelcoAutomationList(w, "hydratedDeployments", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationUpdateHydratedDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, hydratedID, action, ok := parseGCPTelcoAutomationHydratedDeploymentPath(path)
	if !ok || action != "" {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s/hydratedDeployments/%s", project, location, clusterID, deploymentID, hydratedID)
	if provided := gcpTelcoAutomationString(body, "name"); provided == "" || provided != expectedName {
		respondGCPTelcoAutomationInvalidArgument(w, path, "hydratedDeployment.name must match requested resource")
		return true
	}
	updateMask, ok := parseGCPTelcoAutomationUpdateMask(r, body)
	if !ok || len(updateMask) == 0 {
		respondGCPTelcoAutomationInvalidArgument(w, path, "updateMask is required")
		return true
	}
	for _, field := range updateMask {
		if field != "files" && field != "*" {
			respondGCPTelcoAutomationInvalidArgument(w, path, "updateMask has unsupported fields")
			return true
		}
	}
	hydrated := gcpTelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID, 1)
	applyGCPTelcoAutomationHydratedDeploymentOverrides(hydrated, body)
	respondJSON(w, http.StatusOK, hydrated)
	return true
}

func handleGCPTelcoAutomationApplyHydratedDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, deploymentID, hydratedID, action, ok := parseGCPTelcoAutomationHydratedDeploymentPath(path)
	if !ok || action != "apply" {
		return false
	}
	body, ok := decodeGCPTelcoAutomationJSONBody(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s/hydratedDeployments/%s", project, location, clusterID, deploymentID, hydratedID)
	if provided := gcpTelcoAutomationString(body, "name"); provided != "" && provided != expectedName {
		respondGCPTelcoAutomationInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	if strings.Contains(strings.ToLower(hydratedID), "applied") {
		respondGCPTelcoAutomationFailedPrecondition(w, path, "hydrated deployment is already applied")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID, 2))
	return true
}

func handleGCPTelcoAutomationListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || !isGCPTelcoAutomationOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, ok := parseGCPTelcoAutomationPagination(w, r, path, 1000)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTelcoAutomationOperation(fmt.Sprintf("projects/%s/locations/%s/operations/createOrchestrationCluster.cluster-1", project, location), "create", fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/cluster-1", project, location), nil),
		gcpTelcoAutomationOperation(fmt.Sprintf("projects/%s/locations/%s/operations/createEdgeSlm.edgeslm-1", project, location), "create", fmt.Sprintf("projects/%s/locations/%s/edgeSlms/edgeslm-1", project, location), nil),
	}
	return respondGCPTelcoAutomationList(w, "operations", items, pageSize, start, path, false)
}

func handleGCPTelcoAutomationGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPTelcoAutomationOperationPath(path)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(operationID), "missing") {
		respondGCPTelcoAutomationNotFound(w, path, "operation not found")
		return true
	}
	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
	target := ""
	var response map[string]any
	switch {
	case strings.HasPrefix(operationID, "createOrchestrationCluster."):
		clusterID := strings.TrimPrefix(operationID, "createOrchestrationCluster.")
		target = fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
		cluster := gcpTelcoAutomationOrchestrationCluster(project, location, clusterID, 2)
		response = map[string]any{
			"@type": "type.googleapis.com/google.cloud.telcoautomation.v1.OrchestrationCluster",
			"name":  cluster["name"],
			"managementConfig": map[string]any{
				"fullManagementConfig": map[string]any{},
			},
			"createTime": cluster["createTime"],
			"updateTime": cluster["updateTime"],
			"labels":     cluster["labels"],
			"tnaVersion": cluster["tnaVersion"],
			"state":      cluster["state"],
		}
	case strings.HasPrefix(operationID, "deleteOrchestrationCluster."):
		clusterID := strings.TrimPrefix(operationID, "deleteOrchestrationCluster.")
		target = fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
		response = map[string]any{"@type": "type.googleapis.com/google.protobuf.Empty"}
	case strings.HasPrefix(operationID, "createEdgeSlm."):
		edgeID := strings.TrimPrefix(operationID, "createEdgeSlm.")
		target = fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID)
		edge := gcpTelcoAutomationEdgeSlm(project, location, edgeID, "cluster-1", 2)
		response = map[string]any{
			"@type":                "type.googleapis.com/google.cloud.telcoautomation.v1.EdgeSlm",
			"name":                 edge["name"],
			"orchestrationCluster": edge["orchestrationCluster"],
			"createTime":           edge["createTime"],
			"updateTime":           edge["updateTime"],
			"labels":               edge["labels"],
			"tnaVersion":           edge["tnaVersion"],
			"state":                edge["state"],
			"workloadClusterType":  edge["workloadClusterType"],
		}
	case strings.HasPrefix(operationID, "deleteEdgeSlm."):
		edgeID := strings.TrimPrefix(operationID, "deleteEdgeSlm.")
		target = fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID)
		response = map[string]any{"@type": "type.googleapis.com/google.protobuf.Empty"}
	default:
		target = fmt.Sprintf("projects/%s/locations/%s", project, location)
	}
	respondJSON(w, http.StatusOK, gcpTelcoAutomationOperation(name, "get", target, response))
	return true
}

func parseGCPTelcoAutomationLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func isGCPTelcoAutomationOrchestrationClustersCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "orchestrationClusters"
}

func isGCPTelcoAutomationEdgeSlmsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "edgeSlms"
}

func isGCPTelcoAutomationPublicBlueprintsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "publicBlueprints"
}

func isGCPTelcoAutomationOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func parseGCPTelcoAutomationOrchestrationClusterPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "orchestrationClusters" {
		return "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	if !isGCPTelcoAutomationResourceID(clusterID) {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPTelcoAutomationEdgeSlmPath(path string) (project, location, edgeID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "edgeSlms" {
		return "", "", "", false
	}
	edgeID = strings.TrimSpace(tail[1])
	if !isGCPTelcoAutomationResourceID(edgeID) {
		return "", "", "", false
	}
	return project, location, edgeID, true
}

func parseGCPTelcoAutomationBlueprintParentPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "orchestrationClusters" || tail[2] != "blueprints" {
		return "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	if !isGCPTelcoAutomationResourceID(clusterID) {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPTelcoAutomationBlueprintPath(path string) (project, location, clusterID, blueprintID, action string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "orchestrationClusters" || tail[2] != "blueprints" {
		return "", "", "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	raw := strings.TrimSpace(tail[3])
	if !isGCPTelcoAutomationResourceID(clusterID) || raw == "" {
		return "", "", "", "", "", false
	}
	blueprintID, action = splitGCPTelcoAutomationAction(raw)
	baseID, revision := splitGCPTelcoAutomationRevision(blueprintID)
	if !isGCPTelcoAutomationResourceID(baseID) {
		return "", "", "", "", "", false
	}
	if revision != "" && !isGCPTelcoAutomationResourceID(revision) {
		return "", "", "", "", "", false
	}
	switch action {
	case "", "approve", "propose", "reject", "discard", "listRevisions":
	default:
		return "", "", "", "", "", false
	}
	return project, location, clusterID, blueprintID, action, true
}

func parseGCPTelcoAutomationBlueprintSearchPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "orchestrationClusters" || tail[2] != "blueprints:searchRevisions" {
		return "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	if !isGCPTelcoAutomationResourceID(clusterID) {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPTelcoAutomationDeploymentParentPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "orchestrationClusters" || tail[2] != "deployments" {
		return "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	if !isGCPTelcoAutomationResourceID(clusterID) {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPTelcoAutomationDeploymentPath(path string) (project, location, clusterID, deploymentID, action string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "orchestrationClusters" || tail[2] != "deployments" {
		return "", "", "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	raw := strings.TrimSpace(tail[3])
	if !isGCPTelcoAutomationResourceID(clusterID) || raw == "" {
		return "", "", "", "", "", false
	}
	deploymentID, action = splitGCPTelcoAutomationAction(raw)
	baseID, revision := splitGCPTelcoAutomationRevision(deploymentID)
	if !isGCPTelcoAutomationResourceID(baseID) {
		return "", "", "", "", "", false
	}
	if revision != "" && !isGCPTelcoAutomationResourceID(revision) {
		return "", "", "", "", "", false
	}
	switch action {
	case "", "remove", "discard", "apply", "computeDeploymentStatus", "rollback", "listRevisions":
	default:
		return "", "", "", "", "", false
	}
	return project, location, clusterID, deploymentID, action, true
}

func parseGCPTelcoAutomationDeploymentSearchPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "orchestrationClusters" || tail[2] != "deployments:searchRevisions" {
		return "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	if !isGCPTelcoAutomationResourceID(clusterID) {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPTelcoAutomationHydratedDeploymentParentPath(path string) (project, location, clusterID, deploymentID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "orchestrationClusters" || tail[2] != "deployments" || tail[4] != "hydratedDeployments" {
		return "", "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	deploymentID = strings.TrimSpace(tail[3])
	if !isGCPTelcoAutomationResourceID(clusterID) || !isGCPTelcoAutomationResourceID(deploymentID) {
		return "", "", "", "", false
	}
	return project, location, clusterID, deploymentID, true
}

func parseGCPTelcoAutomationHydratedDeploymentPath(path string) (project, location, clusterID, deploymentID, hydratedID, action string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 6 || tail[0] != "orchestrationClusters" || tail[2] != "deployments" || tail[4] != "hydratedDeployments" {
		return "", "", "", "", "", "", false
	}
	clusterID = strings.TrimSpace(tail[1])
	deploymentID = strings.TrimSpace(tail[3])
	raw := strings.TrimSpace(tail[5])
	if !isGCPTelcoAutomationResourceID(clusterID) || !isGCPTelcoAutomationResourceID(deploymentID) || raw == "" {
		return "", "", "", "", "", "", false
	}
	hydratedID, action = splitGCPTelcoAutomationAction(raw)
	if !isGCPTelcoAutomationResourceID(hydratedID) {
		return "", "", "", "", "", "", false
	}
	switch action {
	case "", "apply":
	default:
		return "", "", "", "", "", "", false
	}
	return project, location, clusterID, deploymentID, hydratedID, action, true
}

func parseGCPTelcoAutomationPublicBlueprintPath(path string) (project, location, publicBlueprintID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "publicBlueprints" {
		return "", "", "", false
	}
	publicBlueprintID = strings.TrimSpace(tail[1])
	if !isGCPTelcoAutomationResourceID(publicBlueprintID) {
		return "", "", "", false
	}
	return project, location, publicBlueprintID, true
}

func parseGCPTelcoAutomationOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPTelcoAutomationLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", false
	}
	operationID = strings.TrimSpace(tail[1])
	if operationID == "" {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPTelcoAutomationPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPTelcoAutomationInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > maxPageSize {
		respondGCPTelcoAutomationOutOfRange(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
		return 0, 0, false
	}

	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = strconv.Atoi(token)
		if err != nil || start < 0 {
			respondGCPTelcoAutomationInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPTelcoAutomationList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string, includeUnreachable bool) bool {
	if start > len(items) {
		respondGCPTelcoAutomationInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	out := map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	}
	if includeUnreachable {
		out["unreachable"] = []string{}
	}
	respondJSON(w, http.StatusOK, out)
	return true
}

func decodeGCPTelcoAutomationJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPTelcoAutomationInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPTelcoAutomationJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPTelcoAutomationJSONBody(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPTelcoAutomationInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func parseGCPTelcoAutomationUpdateMask(r *http.Request, body map[string]any) ([]string, bool) {
	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = strings.TrimSpace(gcpTelcoAutomationString(body, "updateMask"))
	}
	if mask == "" {
		return nil, false
	}
	fields := strings.Split(mask, ",")
	out := make([]string, 0, len(fields))
	for _, raw := range fields {
		field := strings.Trim(strings.TrimSpace(raw), `"`)
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out, len(out) > 0
}

func splitGCPTelcoAutomationAction(segment string) (resourceID, action string) {
	resourceID = strings.TrimSpace(segment)
	action = ""
	if id, a, ok := strings.Cut(resourceID, ":"); ok {
		resourceID = strings.TrimSpace(id)
		action = strings.TrimSpace(a)
	}
	return resourceID, action
}

func splitGCPTelcoAutomationRevision(resourceID string) (id, revision string) {
	resourceID = strings.TrimSpace(resourceID)
	if id, rev, ok := strings.Cut(resourceID, "@"); ok {
		return strings.TrimSpace(id), strings.TrimSpace(rev)
	}
	return resourceID, ""
}

func isGCPTelcoAutomationResourceID(value string) bool {
	return gcpTelcoAutomationIDPattern.MatchString(strings.TrimSpace(value))
}

func isGCPTelcoAutomationRequestID(value string) bool {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "00000000-0000-0000-0000-000000000000") {
		return false
	}
	return gcpTelcoAutomationRequestIDPattern.MatchString(value)
}

func isGCPTelcoAutomationMissingID(value string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(value)), "missing")
}

func isGCPTelcoAutomationSimpleStateFilter(filter string, allowedStates []string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	allowed := make(map[string]struct{}, len(allowedStates))
	for _, state := range allowedStates {
		allowed["state = "+state] = struct{}{}
		allowed["state="+state] = struct{}{}
	}
	for _, part := range strings.Split(filter, " OR ") {
		part = strings.TrimSpace(part)
		if _, ok := allowed[part]; !ok {
			return false
		}
	}
	return true
}

func isGCPTelcoAutomationRevisionSearchQuery(query string, blueprint bool) bool {
	query = strings.TrimSpace(query)
	if query == "" || query == "latest=true" {
		return true
	}
	if strings.HasPrefix(query, "name=") {
		name := strings.TrimSpace(strings.TrimPrefix(query, "name="))
		if strings.HasSuffix(name, " latest=true") {
			name = strings.TrimSpace(strings.TrimSuffix(name, " latest=true"))
		}
		if blueprint {
			_, _, _, _, _, ok := parseGCPTelcoAutomationBlueprintName(name)
			return ok
		}
		_, _, _, _, _, ok := parseGCPTelcoAutomationDeploymentName(name)
		return ok
	}
	return false
}

func parseGCPTelcoAutomationLocationName(name string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) {
		return "", "", false
	}
	return project, location, true
}

func parseGCPTelcoAutomationOrchestrationClusterName(name string) (project, location, clusterID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "orchestrationClusters" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) || !isGCPTelcoAutomationResourceID(clusterID) {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPTelcoAutomationEdgeSlmName(name string) (project, location, edgeID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "edgeSlms" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	edgeID = strings.TrimSpace(parts[5])
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) || !isGCPTelcoAutomationResourceID(edgeID) {
		return "", "", "", false
	}
	return project, location, edgeID, true
}

func parseGCPTelcoAutomationBlueprintName(name string) (project, location, clusterID, blueprintID, revisionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "orchestrationClusters" || parts[6] != "blueprints" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	blueprintID, revisionID = splitGCPTelcoAutomationRevision(strings.TrimSpace(parts[7]))
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) || !isGCPTelcoAutomationResourceID(clusterID) || !isGCPTelcoAutomationResourceID(blueprintID) {
		return "", "", "", "", "", false
	}
	if revisionID != "" && !isGCPTelcoAutomationResourceID(revisionID) {
		return "", "", "", "", "", false
	}
	return project, location, clusterID, blueprintID, revisionID, true
}

func parseGCPTelcoAutomationDeploymentName(name string) (project, location, clusterID, deploymentID, revisionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "orchestrationClusters" || parts[6] != "deployments" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	deploymentID, revisionID = splitGCPTelcoAutomationRevision(strings.TrimSpace(parts[7]))
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) || !isGCPTelcoAutomationResourceID(clusterID) || !isGCPTelcoAutomationResourceID(deploymentID) {
		return "", "", "", "", "", false
	}
	if revisionID != "" && !isGCPTelcoAutomationResourceID(revisionID) {
		return "", "", "", "", "", false
	}
	return project, location, clusterID, deploymentID, revisionID, true
}

func parseGCPTelcoAutomationHydratedDeploymentName(name string) (project, location, clusterID, deploymentID, hydratedID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "orchestrationClusters" || parts[6] != "deployments" || parts[8] != "hydratedDeployments" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	deploymentID = strings.TrimSpace(parts[7])
	hydratedID = strings.TrimSpace(parts[9])
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) || !isGCPTelcoAutomationResourceID(clusterID) || !isGCPTelcoAutomationResourceID(deploymentID) || !isGCPTelcoAutomationResourceID(hydratedID) {
		return "", "", "", "", "", false
	}
	return project, location, clusterID, deploymentID, hydratedID, true
}

func parseGCPTelcoAutomationPublicBlueprintName(name string) (project, location, publicBlueprintID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "publicBlueprints" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	publicBlueprintID = strings.TrimSpace(parts[5])
	if !gcpTelcoAutomationProjectIDPattern.MatchString(project) || !gcpTelcoAutomationLocationPattern.MatchString(location) || !isGCPTelcoAutomationResourceID(publicBlueprintID) {
		return "", "", "", false
	}
	return project, location, publicBlueprintID, true
}

func gcpTelcoAutomationLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Telco Automation " + location,
	}
}

func gcpTelcoAutomationOrchestrationCluster(project, location, clusterID string, state int32) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
	return map[string]any{
		"name": name,
		"managementConfig": map[string]any{
			"fullManagementConfig": map[string]any{},
		},
		"createTime": gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"updateTime": gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"labels": map[string]string{
			"env": "staged",
		},
		"tnaVersion": "1.0.0",
		"state":      state,
	}
}

func gcpTelcoAutomationEdgeSlm(project, location, edgeID, clusterID string, state int32) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID)
	clusterName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
	return map[string]any{
		"name":                 name,
		"orchestrationCluster": clusterName,
		"createTime":           gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"updateTime":           gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"labels": map[string]string{
			"env": "staged",
		},
		"tnaVersion":          "1.0.0",
		"state":               state,
		"workloadClusterType": 2,
	}
}

func gcpTelcoAutomationBlueprint(project, location, clusterID, blueprintID, revisionID string, approvalState int32) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s", project, location, clusterID, blueprintID)
	if revisionID != "" {
		name += "@" + revisionID
	}
	return map[string]any{
		"name":               name,
		"revisionId":         revisionID,
		"sourceBlueprint":    fmt.Sprintf("projects/%s/locations/%s/publicBlueprints/public-blueprint-1", project, location),
		"revisionCreateTime": gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"approvalState":      approvalState,
		"displayName":        "Stackyard Blueprint " + blueprintID,
		"repository":         fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s/repository", project, location, clusterID, blueprintID),
		"files": []map[string]any{
			{
				"path":     "deployments/main.yaml",
				"content":  "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: stackyard-blueprint",
				"editable": true,
			},
		},
		"labels": map[string]string{
			"env": "staged",
		},
		"createTime":      gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"updateTime":      gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"sourceProvider":  "Google",
		"deploymentLevel": 2,
		"rollbackSupport": true,
	}
}

func gcpTelcoAutomationPublicBlueprint(project, location, publicBlueprintID string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("projects/%s/locations/%s/publicBlueprints/%s", project, location, publicBlueprintID),
		"displayName":     "Public Blueprint " + publicBlueprintID,
		"description":     "Stackyard staged public blueprint",
		"deploymentLevel": 2,
		"sourceProvider":  "Google",
		"rollbackSupport": true,
	}
}

func gcpTelcoAutomationDeployment(project, location, clusterID, deploymentID, revisionID string, state int32) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if revisionID != "" {
		name += "@" + revisionID
	}
	return map[string]any{
		"name":                    name,
		"revisionId":              revisionID,
		"sourceBlueprintRevision": fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/blueprint-approved@rev-3", project, location, clusterID),
		"revisionCreateTime":      gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"state":                   state,
		"displayName":             "Stackyard Deployment " + deploymentID,
		"repository":              fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s/repository", project, location, clusterID, deploymentID),
		"files": []map[string]any{
			{
				"path":     "deployments/workload.yaml",
				"content":  "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: stackyard-workload",
				"editable": true,
			},
		},
		"labels": map[string]string{
			"env": "staged",
		},
		"createTime":      gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"updateTime":      gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"sourceProvider":  "Google",
		"deploymentLevel": 2,
		"rollbackSupport": true,
		"workloadCluster": fmt.Sprintf("projects/%s/locations/%s/workloadClusters/workload-1", project, location),
	}
}

func gcpTelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID string, state int32) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s/hydratedDeployments/%s", project, location, clusterID, deploymentID, hydratedID),
		"state": state,
		"files": []map[string]any{
			{
				"path":     "hydrated/site.yaml",
				"content":  "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: hydrated-site",
				"editable": true,
			},
		},
		"workloadCluster": fmt.Sprintf("projects/%s/locations/%s/workloadClusters/workload-1", project, location),
	}
}

func gcpTelcoAutomationComputeDeploymentStatus(name string) map[string]any {
	return map[string]any{
		"name":             name,
		"aggregatedStatus": 2,
		"resourceStatuses": []map[string]any{
			{
				"name":              "nfdeploy-sample",
				"resourceNamespace": "default",
				"group":             "nf.google.com",
				"version":           "v1",
				"kind":              "NFDeploy",
				"resourceType":      1,
				"status":            2,
				"nfDeployStatus": map[string]any{
					"targetedNfs": 1,
					"readyNfs":    1,
					"sites": []map[string]any{
						{
							"site":            "site-1",
							"pendingDeletion": false,
							"hydration": map[string]any{
								"siteVersion": map[string]any{
									"name":        "projects/stackyard/locations/us-central1/siteVersions/site-1",
									"nfVendor":    "stackyard",
									"nfType":      "sample",
									"nfVersion":   "1.0.0",
									"description": "staged site version",
								},
								"status": "READY",
							},
							"workload": map[string]any{
								"siteVersion": map[string]any{
									"name":        "projects/stackyard/locations/us-central1/siteVersions/site-1",
									"nfVendor":    "stackyard",
									"nfType":      "sample",
									"nfVersion":   "1.0.0",
									"description": "staged site version",
								},
								"status": "READY",
							},
						},
					},
				},
			},
		},
	}
}

func gcpTelcoAutomationOperation(name, verb, target string, response map[string]any) map[string]any {
	meta := map[string]any{
		"@type":                 "type.googleapis.com/google.cloud.telcoautomation.v1.OperationMetadata",
		"createTime":            gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"endTime":               gcpTelcoAutomationReferenceTime.Format(time.RFC3339),
		"target":                target,
		"verb":                  verb,
		"statusMessage":         "completed",
		"requestedCancellation": false,
		"apiVersion":            "v1",
	}
	out := map[string]any{
		"name":     name,
		"metadata": meta,
		"done":     true,
	}
	if response != nil {
		out["response"] = response
	}
	return out
}

func gcpTelcoAutomationString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func applyGCPTelcoAutomationOrchestrationClusterOverrides(out map[string]any, body map[string]any) {
	if labels, ok := body["labels"].(map[string]any); ok && len(labels) > 0 {
		typed := map[string]string{}
		for k, v := range labels {
			val, _ := v.(string)
			if strings.TrimSpace(k) != "" && strings.TrimSpace(val) != "" {
				typed[k] = val
			}
		}
		if len(typed) > 0 {
			out["labels"] = typed
		}
	}
	if mgmt, ok := body["managementConfig"].(map[string]any); ok && len(mgmt) > 0 {
		out["managementConfig"] = mgmt
	}
}

func applyGCPTelcoAutomationEdgeSlmOverrides(out map[string]any, body map[string]any) {
	if cluster := gcpTelcoAutomationString(body, "orchestrationCluster"); cluster != "" {
		out["orchestrationCluster"] = cluster
	}
	if labels, ok := body["labels"].(map[string]any); ok && len(labels) > 0 {
		typed := map[string]string{}
		for k, v := range labels {
			val, _ := v.(string)
			if strings.TrimSpace(k) != "" && strings.TrimSpace(val) != "" {
				typed[k] = val
			}
		}
		if len(typed) > 0 {
			out["labels"] = typed
		}
	}
}

func applyGCPTelcoAutomationBlueprintOverrides(out map[string]any, body map[string]any) {
	if displayName := gcpTelcoAutomationString(body, "displayName"); displayName != "" {
		out["displayName"] = displayName
	}
	if sourceBlueprint := gcpTelcoAutomationString(body, "sourceBlueprint"); sourceBlueprint != "" {
		out["sourceBlueprint"] = sourceBlueprint
	}
	if files, ok := body["files"].([]any); ok {
		out["files"] = files
	}
	if labels, ok := body["labels"].(map[string]any); ok && len(labels) > 0 {
		typed := map[string]string{}
		for k, v := range labels {
			val, _ := v.(string)
			if strings.TrimSpace(k) != "" && strings.TrimSpace(val) != "" {
				typed[k] = val
			}
		}
		if len(typed) > 0 {
			out["labels"] = typed
		}
	}
}

func applyGCPTelcoAutomationDeploymentOverrides(out map[string]any, body map[string]any) {
	if displayName := gcpTelcoAutomationString(body, "displayName"); displayName != "" {
		out["displayName"] = displayName
	}
	if sourceBlueprintRevision := gcpTelcoAutomationString(body, "sourceBlueprintRevision"); sourceBlueprintRevision != "" {
		out["sourceBlueprintRevision"] = sourceBlueprintRevision
	}
	if workloadCluster := gcpTelcoAutomationString(body, "workloadCluster"); workloadCluster != "" {
		out["workloadCluster"] = workloadCluster
	}
	if files, ok := body["files"].([]any); ok {
		out["files"] = files
	}
	if labels, ok := body["labels"].(map[string]any); ok && len(labels) > 0 {
		typed := map[string]string{}
		for k, v := range labels {
			val, _ := v.(string)
			if strings.TrimSpace(k) != "" && strings.TrimSpace(val) != "" {
				typed[k] = val
			}
		}
		if len(typed) > 0 {
			out["labels"] = typed
		}
	}
}

func applyGCPTelcoAutomationHydratedDeploymentOverrides(out map[string]any, body map[string]any) {
	if files, ok := body["files"].([]any); ok {
		out["files"] = files
	}
}

func filterGCPTelcoAutomationBlueprints(items []map[string]any, filter string) []map[string]any {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return items
	}
	allow := map[int32]struct{}{}
	for _, part := range strings.Split(filter, " OR ") {
		part = strings.TrimSpace(part)
		switch part {
		case "state = DRAFT", "state=DRAFT":
			allow[1] = struct{}{}
		case "state = PROPOSED", "state=PROPOSED":
			allow[2] = struct{}{}
		case "state = APPROVED", "state=APPROVED":
			allow[3] = struct{}{}
		}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		state := int32(gcpTelcoAutomationFloat(item["approvalState"]))
		if _, ok := allow[state]; ok {
			out = append(out, item)
		}
	}
	return out
}

func filterGCPTelcoAutomationDeployments(items []map[string]any, filter string) []map[string]any {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return items
	}
	allow := map[int32]struct{}{}
	for _, part := range strings.Split(filter, " OR ") {
		part = strings.TrimSpace(part)
		switch part {
		case "state = DRAFT", "state=DRAFT":
			allow[1] = struct{}{}
		case "state = APPLIED", "state=APPLIED":
			allow[2] = struct{}{}
		case "state = DELETING", "state=DELETING":
			allow[3] = struct{}{}
		}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		state := int32(gcpTelcoAutomationFloat(item["state"]))
		if _, ok := allow[state]; ok {
			out = append(out, item)
		}
	}
	return out
}

func filterGCPTelcoAutomationBlueprintRevisionSearch(items []map[string]any, query string) []map[string]any {
	query = strings.TrimSpace(query)
	switch {
	case query == "", query == "latest=true":
		if query == "latest=true" {
			out := make([]map[string]any, 0, len(items))
			for _, item := range items {
				if strings.HasSuffix(gcpTelcoAutomationString(item, "revisionId"), "3") || strings.HasSuffix(gcpTelcoAutomationString(item, "revisionId"), "2") {
					out = append(out, item)
				}
			}
			return out
		}
		return items
	case strings.HasPrefix(query, "name="):
		value := strings.TrimSpace(strings.TrimPrefix(query, "name="))
		latestOnly := false
		if strings.HasSuffix(value, " latest=true") {
			value = strings.TrimSpace(strings.TrimSuffix(value, " latest=true"))
			latestOnly = true
		}
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			name := gcpTelcoAutomationString(item, "name")
			base, _ := splitGCPTelcoAutomationRevision(strings.TrimPrefix(name, "projects/"))
			_ = base
			if strings.Contains(name, value) {
				out = append(out, item)
			}
		}
		if latestOnly && len(out) > 1 {
			return out[len(out)-1:]
		}
		return out
	default:
		return items
	}
}

func filterGCPTelcoAutomationDeploymentRevisionSearch(items []map[string]any, query string) []map[string]any {
	query = strings.TrimSpace(query)
	switch {
	case query == "", query == "latest=true":
		if query == "latest=true" && len(items) > 1 {
			return items[len(items)-1:]
		}
		return items
	case strings.HasPrefix(query, "name="):
		value := strings.TrimSpace(strings.TrimPrefix(query, "name="))
		latestOnly := false
		if strings.HasSuffix(value, " latest=true") {
			value = strings.TrimSpace(strings.TrimSuffix(value, " latest=true"))
			latestOnly = true
		}
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.Contains(gcpTelcoAutomationString(item, "name"), value) {
				out = append(out, item)
			}
		}
		if latestOnly && len(out) > 1 {
			return out[len(out)-1:]
		}
		return out
	default:
		return items
	}
}

func gcpTelcoAutomationBlueprintStateForName(blueprintID string) int32 {
	id := strings.ToLower(blueprintID)
	switch {
	case strings.Contains(id, "proposed"):
		return 2
	case strings.Contains(id, "approved"):
		return 3
	default:
		return 1
	}
}

func gcpTelcoAutomationDeploymentStateForName(deploymentID string) int32 {
	id := strings.ToLower(deploymentID)
	switch {
	case strings.Contains(id, "applied"):
		return 2
	case strings.Contains(id, "deleting"):
		return 3
	default:
		return 1
	}
}

func gcpTelcoAutomationFloat(v any) float64 {
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func respondGCPTelcoAutomationInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTelcoAutomationOutOfRange(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "OutOfRange",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTelcoAutomationFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPTelcoAutomationNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_telcoautomation(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "telcoautomation") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1",
			"state":    2,
			"service":  "telcoautomation",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
