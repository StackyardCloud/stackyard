package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var gcpSecurityPostureReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSecurityPostureRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_securityposture(w, r) {
		return true
	}

	path := normalizeGCPSecurityPosturePath(rawRequestPath(r))
	if isGCPSecurityPostureLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSecurityPostureListLocations(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSecurityPosturePath(path, hasGCPSecurityPostureHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSecurityPostureListPostures(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureListPostureRevisions(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureGetPosture(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureListPostureDeployments(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureGetPostureDeployment(w, path) {
			return true
		}
		if handleGCPSecurityPostureListPostureTemplates(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureGetPostureTemplate(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureListOperations(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSecurityPostureCreatePosture(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureExtractPosture(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureCreatePostureDeployment(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSecurityPostureUpdatePosture(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureUpdatePostureDeployment(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSecurityPostureDeletePosture(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureDeletePostureDeployment(w, r, path) {
			return true
		}
		if handleGCPSecurityPostureDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSecurityPosturePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSecurityPostureHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "securityposture",
		"securityposture-apiv1",
		"securityposture_apiv1",
		"security-posture",
		"security_posture",
		"gcp-security-posture":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-securityposture-apiv1") || strings.Contains(ua, "cloud.google.com/go/securityposture")
}

func isGCPSecurityPostureLocationRequest(r *http.Request, path string) bool {
	if !hasGCPSecurityPostureHint(r) {
		return false
	}
	_, _, _, ok := parseGCPSecurityPostureLocationPath(path)
	return ok
}

func isGCPSecurityPosturePath(path string, includeAmbiguous bool) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.securityposture.v1.SecurityPosture/") {
		return true
	}
	_, _, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	switch tail[0] {
	case "postures", "postureDeployments", "postureTemplates", "operations":
		return true
	default:
		return includeAmbiguous
	}
}

func handleGCPSecurityPostureListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, _, list, ok := parseGCPSecurityPostureLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPosturePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPostureLocation(orgID, "global"),
		gcpSecurityPostureLocation(orgID, "us-central1"),
	}
	return respondGCPSecurityPostureList(w, "locations", items, pageSize, start, path)
}

func handleGCPSecurityPostureGetLocation(w http.ResponseWriter, path string) bool {
	orgID, location, list, ok := parseGCPSecurityPostureLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureLocation(orgID, location))
	return true
}

func handleGCPSecurityPostureListPostures(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || !isGCPSecurityPosturePostureCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPosturePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPosturePosture(orgID, location, "posture-1", "0000000a", "ACTIVE"),
		gcpSecurityPosturePosture(orgID, location, "posture-draft", "0000000b", "DRAFT"),
	}
	return respondGCPSecurityPostureListWithExtras(w, "postures", items, pageSize, start, path, map[string]any{"unreachable": []any{}})
}

func handleGCPSecurityPostureListPostureRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, postureID, ok := parseGCPSecurityPosturePostureListRevisionsPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPosturePagination(w, r, path)
	if !valid {
		return true
	}
	state := "ACTIVE"
	if strings.Contains(strings.ToLower(postureID), "draft") {
		state = "DRAFT"
	}
	items := []map[string]any{
		gcpSecurityPosturePosture(orgID, location, postureID, "00000009", state),
		gcpSecurityPosturePosture(orgID, location, postureID, "0000000a", state),
	}
	return respondGCPSecurityPostureList(w, "revisions", items, pageSize, start, path)
}

func handleGCPSecurityPostureGetPosture(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, postureID, ok := parseGCPSecurityPosturePosturePath(path)
	if !ok {
		return false
	}
	revision := strings.TrimSpace(r.URL.Query().Get("revisionId"))
	if revision == "" {
		revision = "0000000a"
	}
	state := gcpSecurityPostureStateForID(postureID)
	respondJSON(w, http.StatusOK, gcpSecurityPosturePosture(orgID, location, postureID, revision, state))
	return true
}

func handleGCPSecurityPostureCreatePosture(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || !isGCPSecurityPosturePostureCollectionTail(tail) {
		return false
	}
	postureID := strings.TrimSpace(r.URL.Query().Get("postureId"))
	if postureID == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureId is required")
		return true
	}

	body, valid := decodeGCPSecurityPostureJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	posture := gcpSecurityPostureBodyMap(body, "posture")
	if len(posture) == 0 {
		posture = body
	}
	if len(posture) == 0 {
		respondGCPSecurityPostureInvalidArgument(w, path, "posture is required")
		return true
	}
	expectedName := gcpSecurityPosturePostureName(orgID, location, postureID)
	if got := strings.TrimSpace(gcpSecurityPostureString(posture, "name")); got != "" && got != expectedName {
		respondGCPSecurityPostureInvalidArgument(w, path, "posture.name must match parent and postureId")
		return true
	}
	if !gcpSecurityPostureHasArray(posture, "policySets") {
		respondGCPSecurityPostureInvalidArgument(w, path, "posture.policySets is required")
		return true
	}

	opID := "createPosture." + postureID
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, opID, expectedName, "create", false))
	return true
}

func handleGCPSecurityPostureUpdatePosture(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, postureID, ok := parseGCPSecurityPosturePosturePath(path)
	if !ok {
		return false
	}
	revisionID := strings.TrimSpace(r.URL.Query().Get("revisionId"))
	if revisionID == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "revisionId is required")
		return true
	}
	updateMask, valid := parseGCPSecurityPostureUpdateMask(w, path, r.URL.Query().Get("updateMask"), []string{"state", "description", "policySets", "annotations", "etag"})
	if !valid {
		return true
	}
	if len(updateMask) == 0 {
		respondGCPSecurityPostureInvalidArgument(w, path, "updateMask is required")
		return true
	}

	body, valid := decodeGCPSecurityPostureJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	posture := gcpSecurityPostureBodyMap(body, "posture")
	if len(posture) == 0 {
		posture = body
	}
	if len(posture) == 0 {
		respondGCPSecurityPostureInvalidArgument(w, path, "posture is required")
		return true
	}
	expectedName := gcpSecurityPosturePostureName(orgID, location, postureID)
	if got := strings.TrimSpace(gcpSecurityPostureString(posture, "name")); got == "" || got != expectedName {
		respondGCPSecurityPostureInvalidArgument(w, path, "posture.name must match requested resource")
		return true
	}
	if gcpSecurityPostureMaskContains(updateMask, "policySets") && !gcpSecurityPostureHasArray(posture, "policySets") {
		respondGCPSecurityPostureInvalidArgument(w, path, "posture.policySets is required by updateMask")
		return true
	}
	if gcpSecurityPostureMaskContains(updateMask, "state") && (gcpSecurityPostureMaskContains(updateMask, "description") || gcpSecurityPostureMaskContains(updateMask, "policySets") || gcpSecurityPostureMaskContains(updateMask, "annotations")) {
		respondGCPSecurityPostureInvalidArgument(w, path, "state update cannot be combined with description, policySets, or annotations")
		return true
	}
	if etag := strings.TrimSpace(gcpSecurityPostureString(posture, "etag")); etag != "" && etag != gcpSecurityPostureEtag(postureID) {
		respondGCPSecurityPostureAborted(w, path, "etag mismatch")
		return true
	}
	if gcpSecurityPostureMaskContains(updateMask, "state") {
		requested := strings.ToUpper(strings.TrimSpace(gcpSecurityPostureString(posture, "state")))
		if requested == "" {
			respondGCPSecurityPostureInvalidArgument(w, path, "posture.state is required by updateMask")
			return true
		}
		if !gcpSecurityPostureStateTransitionAllowed(gcpSecurityPostureStateForID(postureID), requested) {
			respondGCPSecurityPostureFailedPrecondition(w, path, "state transition is not allowed")
			return true
		}
	}

	opID := "updatePosture." + postureID + "." + revisionID
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, opID, expectedName, "update", false))
	return true
}

func handleGCPSecurityPostureDeletePosture(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, postureID, ok := parseGCPSecurityPosturePosturePath(path)
	if !ok {
		return false
	}
	if etag := strings.TrimSpace(r.URL.Query().Get("etag")); etag != "" && etag != gcpSecurityPostureEtag(postureID) {
		respondGCPSecurityPostureAborted(w, path, "etag mismatch")
		return true
	}
	if strings.Contains(strings.ToLower(postureID), "deployed") {
		respondGCPSecurityPostureFailedPrecondition(w, path, "posture has active deployments")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, "deletePosture."+postureID, gcpSecurityPosturePostureName(orgID, location, postureID), "delete", false))
	return true
}

func handleGCPSecurityPostureExtractPosture(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || !isGCPSecurityPostureExtractTail(tail) {
		return false
	}
	body, valid := decodeGCPSecurityPostureJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	expectedParent := gcpSecurityPostureParent(orgID, location)
	if parent := strings.TrimSpace(gcpSecurityPostureString(body, "parent")); parent != "" && parent != expectedParent {
		respondGCPSecurityPostureInvalidArgument(w, path, "parent in body must match request path")
		return true
	}
	postureID := strings.TrimSpace(gcpSecurityPostureString(body, "postureId"))
	if postureID == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureId is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityPostureString(body, "workload")) == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "workload is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, "extractPosture."+postureID, gcpSecurityPosturePostureName(orgID, location, postureID), "extract", false))
	return true
}

func handleGCPSecurityPostureListPostureDeployments(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || !isGCPSecurityPostureDeploymentCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPosturePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPostureDeployment(orgID, location, "deployment-1"),
		gcpSecurityPostureDeployment(orgID, location, "deployment-2"),
	}
	if filter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("filter"))); strings.Contains(filter, "targetresource") {
		items = items[:1]
	}
	return respondGCPSecurityPostureListWithExtras(w, "postureDeployments", items, pageSize, start, path, map[string]any{"unreachable": []any{}})
}

func handleGCPSecurityPostureGetPostureDeployment(w http.ResponseWriter, path string) bool {
	orgID, location, deploymentID, ok := parseGCPSecurityPostureDeploymentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureDeployment(orgID, location, deploymentID))
	return true
}

func handleGCPSecurityPostureCreatePostureDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || !isGCPSecurityPostureDeploymentCollectionTail(tail) {
		return false
	}
	deploymentID := strings.TrimSpace(r.URL.Query().Get("postureDeploymentId"))
	if deploymentID == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeploymentId is required")
		return true
	}
	body, valid := decodeGCPSecurityPostureJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	deployment := gcpSecurityPostureBodyMap(body, "postureDeployment")
	if len(deployment) == 0 {
		deployment = body
	}
	if len(deployment) == 0 {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment is required")
		return true
	}
	expectedName := gcpSecurityPostureDeploymentName(orgID, location, deploymentID)
	if got := strings.TrimSpace(gcpSecurityPostureString(deployment, "name")); got != "" && got != expectedName {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment.name must match parent and postureDeploymentId")
		return true
	}
	if strings.TrimSpace(gcpSecurityPostureString(deployment, "targetResource")) == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment.targetResource is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityPostureString(deployment, "postureId")) == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment.postureId is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityPostureString(deployment, "postureRevisionId")) == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment.postureRevisionId is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, "createPostureDeployment."+deploymentID, expectedName, "create", false))
	return true
}

func handleGCPSecurityPostureUpdatePostureDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, deploymentID, ok := parseGCPSecurityPostureDeploymentPath(path)
	if !ok {
		return false
	}
	updateMask, valid := parseGCPSecurityPostureUpdateMask(w, path, r.URL.Query().Get("updateMask"), []string{"description", "postureId", "postureRevisionId", "annotations", "targetResource", "etag"})
	if !valid {
		return true
	}
	if len(updateMask) == 0 {
		respondGCPSecurityPostureInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityPostureJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	deployment := gcpSecurityPostureBodyMap(body, "postureDeployment")
	if len(deployment) == 0 {
		deployment = body
	}
	if len(deployment) == 0 {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment is required")
		return true
	}
	expectedName := gcpSecurityPostureDeploymentName(orgID, location, deploymentID)
	if got := strings.TrimSpace(gcpSecurityPostureString(deployment, "name")); got == "" || got != expectedName {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment.name must match requested resource")
		return true
	}
	if etag := strings.TrimSpace(gcpSecurityPostureString(deployment, "etag")); etag != "" && etag != gcpSecurityPostureEtag(deploymentID) {
		respondGCPSecurityPostureAborted(w, path, "etag mismatch")
		return true
	}
	if gcpSecurityPostureMaskContains(updateMask, "targetResource") && strings.TrimSpace(gcpSecurityPostureString(deployment, "targetResource")) == "" {
		respondGCPSecurityPostureInvalidArgument(w, path, "postureDeployment.targetResource is required by updateMask")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, "updatePostureDeployment."+deploymentID, expectedName, "update", false))
	return true
}

func handleGCPSecurityPostureDeletePostureDeployment(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, deploymentID, ok := parseGCPSecurityPostureDeploymentPath(path)
	if !ok {
		return false
	}
	if etag := strings.TrimSpace(r.URL.Query().Get("etag")); etag != "" && etag != gcpSecurityPostureEtag(deploymentID) {
		respondGCPSecurityPostureAborted(w, path, "etag mismatch")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, "deletePostureDeployment."+deploymentID, gcpSecurityPostureDeploymentName(orgID, location, deploymentID), "delete", false))
	return true
}

func handleGCPSecurityPostureListPostureTemplates(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || !isGCPSecurityPostureTemplateCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPosturePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPostureTemplate(orgID, location, "template-1", "00000001"),
		gcpSecurityPostureTemplate(orgID, location, "template-2", "00000002"),
	}
	if filter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("filter"))); strings.Contains(filter, "deprecated") {
		items = items[1:]
	}
	return respondGCPSecurityPostureList(w, "postureTemplates", items, pageSize, start, path)
}

func handleGCPSecurityPostureGetPostureTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, templateID, ok := parseGCPSecurityPostureTemplatePath(path)
	if !ok {
		return false
	}
	revision := strings.TrimSpace(r.URL.Query().Get("revisionId"))
	if revision == "" {
		revision = "00000001"
	}
	respondJSON(w, http.StatusOK, gcpSecurityPostureTemplate(orgID, location, templateID, revision))
	return true
}

func handleGCPSecurityPostureListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || !isGCPSecurityPostureOperationCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityPosturePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityPostureOperation(orgID, location, "createPosture.posture-1", gcpSecurityPosturePostureName(orgID, location, "posture-1"), "create", false),
		gcpSecurityPostureOperation(orgID, location, "updatePosture.posture-1", gcpSecurityPosturePostureName(orgID, location, "posture-1"), "update", true),
	}
	return respondGCPSecurityPostureList(w, "operations", items, pageSize, start, path)
}

func handleGCPSecurityPostureGetOperation(w http.ResponseWriter, path string) bool {
	orgID, location, operationID, ok := parseGCPSecurityPostureOperationPath(path)
	if !ok {
		return false
	}
	done := strings.Contains(strings.ToLower(operationID), "done") || strings.Contains(strings.ToLower(operationID), "update")
	respondJSON(w, http.StatusOK, gcpSecurityPostureOperation(orgID, location, operationID, gcpSecurityPosturePostureName(orgID, location, "posture-1"), "poll", done))
	return true
}

func handleGCPSecurityPostureCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, _, _, ok := parseGCPSecurityPostureOperationActionPath(path, "cancel"); !ok {
		return false
	}
	if _, valid := decodeGCPSecurityPostureJSONBody(w, r, path, false); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityPostureDeleteOperation(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPSecurityPostureOperationPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPSecurityPostureLocationPath(path string) (orgID, location string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "organizations" || parts[4] != "locations" {
		return "", "", false, false
	}
	orgID = strings.TrimSpace(parts[3])
	if orgID == "" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return orgID, "", true, true
	}
	location = strings.TrimSpace(parts[5])
	if location == "" {
		return "", "", false, false
	}
	return orgID, location, false, true
}

func parseGCPSecurityPostureLocationTail(path string) (orgID, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return "", "", nil, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "organizations" || parts[4] != "locations" {
		return "", "", nil, false
	}
	orgID = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" {
		return "", "", nil, false
	}
	return orgID, location, parts[6:], true
}

func isGCPSecurityPosturePostureCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "postures"
}

func isGCPSecurityPostureExtractTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	base, action, ok := strings.Cut(tail[0], ":")
	return ok && base == "postures" && action == "extract"
}

func parseGCPSecurityPosturePosturePath(path string) (orgID, location, postureID string, ok bool) {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "postures" {
		return "", "", "", false
	}
	postureID = strings.TrimSpace(tail[1])
	if postureID == "" || strings.Contains(postureID, ":") {
		return "", "", "", false
	}
	return orgID, location, postureID, true
}

func parseGCPSecurityPosturePostureListRevisionsPath(path string) (orgID, location, postureID string, ok bool) {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "postures" {
		return "", "", "", false
	}
	postureID, action, found := splitGCPSecurityPostureActionSegment(tail[1])
	if !found || action != "listRevisions" {
		return "", "", "", false
	}
	return orgID, location, postureID, true
}

func isGCPSecurityPostureDeploymentCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "postureDeployments"
}

func parseGCPSecurityPostureDeploymentPath(path string) (orgID, location, deploymentID string, ok bool) {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "postureDeployments" {
		return "", "", "", false
	}
	deploymentID = strings.TrimSpace(tail[1])
	if deploymentID == "" || strings.Contains(deploymentID, ":") {
		return "", "", "", false
	}
	return orgID, location, deploymentID, true
}

func isGCPSecurityPostureTemplateCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "postureTemplates"
}

func parseGCPSecurityPostureTemplatePath(path string) (orgID, location, templateID string, ok bool) {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "postureTemplates" {
		return "", "", "", false
	}
	templateID = strings.TrimSpace(tail[1])
	if templateID == "" || strings.Contains(templateID, ":") {
		return "", "", "", false
	}
	return orgID, location, templateID, true
}

func isGCPSecurityPostureOperationCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func parseGCPSecurityPostureOperationPath(path string) (orgID, location, operationID string, ok bool) {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", false
	}
	operationID = strings.TrimSpace(tail[1])
	if operationID == "" || strings.Contains(operationID, ":") {
		return "", "", "", false
	}
	return orgID, location, operationID, true
}

func parseGCPSecurityPostureOperationActionPath(path, action string) (orgID, location, operationID string, ok bool) {
	orgID, location, tail, ok := parseGCPSecurityPostureLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", false
	}
	id, parsedAction, found := splitGCPSecurityPostureActionSegment(tail[1])
	if !found || parsedAction != action {
		return "", "", "", false
	}
	return orgID, location, id, true
}

func splitGCPSecurityPostureActionSegment(raw string) (id, action string, ok bool) {
	segment := strings.TrimSpace(raw)
	if segment == "" {
		return "", "", false
	}
	if decoded, err := url.PathUnescape(segment); err == nil {
		segment = decoded
	}
	id, action, ok = strings.Cut(segment, ":")
	if !ok {
		return "", "", false
	}
	id = strings.TrimSpace(id)
	action = strings.TrimSpace(action)
	if id == "" || action == "" {
		return "", "", false
	}
	return id, action, true
}

func parseGCPSecurityPosturePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPSecurityPostureInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPSecurityPostureInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPSecurityPostureInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPSecurityPostureUpdateMask(w http.ResponseWriter, path, raw string, allowed []string) ([]string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, true
	}
	allowedSet := make(map[string]string, len(allowed))
	for _, item := range allowed {
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(item, "_", ""), ".", ""))
		allowedSet[normalized] = item
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		field := strings.TrimSpace(part)
		if field == "" {
			continue
		}
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(field, "_", ""), ".", ""))
		canonical, ok := allowedSet[normalized]
		if !ok {
			respondGCPSecurityPostureInvalidArgument(w, path, "updateMask contains unsupported path: "+field)
			return nil, false
		}
		out = append(out, canonical)
	}
	if len(out) == 0 {
		respondGCPSecurityPostureInvalidArgument(w, path, "updateMask must include at least one path")
		return nil, false
	}
	return out, true
}

func respondGCPSecurityPostureList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	return respondGCPSecurityPostureListWithExtras(w, key, items, pageSize, start, path, nil)
}

func respondGCPSecurityPostureListWithExtras(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string, extras map[string]any) bool {
	if start > len(items) {
		respondGCPSecurityPostureInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	resp := map[string]any{
		key:             items[start:end],
		"nextPageToken": next,
	}
	for k, v := range extras {
		resp[k] = v
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func decodeGCPSecurityPostureJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	if r.Body == nil {
		if required {
			respondGCPSecurityPostureInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPSecurityPostureInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		if required {
			respondGCPSecurityPostureInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPSecurityPostureInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpSecurityPostureBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return map[string]any{}
}

func gcpSecurityPostureString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpSecurityPostureHasArray(body map[string]any, key string) bool {
	items, ok := body[key].([]any)
	return ok && len(items) > 0
}

func gcpSecurityPostureMaskContains(mask []string, field string) bool {
	for _, item := range mask {
		if item == field {
			return true
		}
	}
	return false
}

func gcpSecurityPostureStateForID(postureID string) string {
	id := strings.ToLower(strings.TrimSpace(postureID))
	switch {
	case strings.Contains(id, "draft"):
		return "DRAFT"
	case strings.Contains(id, "deprecated"):
		return "DEPRECATED"
	default:
		return "ACTIVE"
	}
}

func gcpSecurityPostureStateTransitionAllowed(current, requested string) bool {
	requested = strings.ToUpper(strings.TrimSpace(requested))
	current = strings.ToUpper(strings.TrimSpace(current))
	switch current {
	case "ACTIVE":
		return requested == "ACTIVE" || requested == "DRAFT" || requested == "DEPRECATED"
	case "DRAFT", "DEPRECATED":
		return requested == "ACTIVE"
	default:
		return requested == "ACTIVE"
	}
}

func gcpSecurityPostureParent(orgID, location string) string {
	return fmt.Sprintf("organizations/%s/locations/%s", orgID, location)
}

func gcpSecurityPosturePostureName(orgID, location, postureID string) string {
	return fmt.Sprintf("%s/postures/%s", gcpSecurityPostureParent(orgID, location), postureID)
}

func gcpSecurityPostureDeploymentName(orgID, location, deploymentID string) string {
	return fmt.Sprintf("%s/postureDeployments/%s", gcpSecurityPostureParent(orgID, location), deploymentID)
}

func gcpSecurityPostureTemplateName(orgID, location, templateID string) string {
	return fmt.Sprintf("%s/postureTemplates/%s", gcpSecurityPostureParent(orgID, location), templateID)
}

func gcpSecurityPostureOperationName(orgID, location, operationID string) string {
	return fmt.Sprintf("%s/operations/%s", gcpSecurityPostureParent(orgID, location), operationID)
}

func gcpSecurityPostureEtag(id string) string {
	return "etag-" + strings.TrimSpace(id)
}

func gcpSecurityPostureLocation(orgID, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("organizations/%s/locations/%s", orgID, location),
		"locationId":  location,
		"displayName": "Security Posture " + location,
		"labels": map[string]string{
			"service": "securityposture",
		},
	}
}

func gcpSecurityPosturePosture(orgID, location, postureID, revisionID, state string) map[string]any {
	return map[string]any{
		"name":        gcpSecurityPosturePostureName(orgID, location, postureID),
		"state":       state,
		"revisionId":  revisionID,
		"createTime":  gcpSecurityPostureReferenceTime.Format(time.RFC3339),
		"updateTime":  gcpSecurityPostureReferenceTime.Add(15 * time.Minute).Format(time.RFC3339),
		"description": "Stackyard security posture " + postureID,
		"policySets": []any{
			map[string]any{
				"policySetId": "baseline",
				"description": "Baseline controls",
				"policies": []any{
					map[string]any{
						"policyId":            "sha-001",
						"description":         "Enable SHA module",
						"constraint":          map[string]any{"securityHealthAnalyticsModule": map[string]any{"moduleName": "BIGQUERY_TABLE_CMEK_DISABLED", "moduleEnablementState": "ENABLED"}},
						"complianceStandards": []any{map[string]any{"standard": "CIS", "control": "1.1"}},
					},
				},
			},
		},
		"etag":        gcpSecurityPostureEtag(postureID),
		"annotations": map[string]string{"env": "test", "service": "securityposture"},
		"reconciling": false,
	}
}

func gcpSecurityPostureDeployment(orgID, location, deploymentID string) map[string]any {
	return map[string]any{
		"name":                     gcpSecurityPostureDeploymentName(orgID, location, deploymentID),
		"targetResource":           "projects/123456789",
		"state":                    "ACTIVE",
		"postureId":                gcpSecurityPosturePostureName(orgID, location, "posture-1"),
		"postureRevisionId":        "0000000a",
		"createTime":               gcpSecurityPostureReferenceTime.Format(time.RFC3339),
		"updateTime":               gcpSecurityPostureReferenceTime.Add(20 * time.Minute).Format(time.RFC3339),
		"description":              "Stackyard posture deployment " + deploymentID,
		"etag":                     gcpSecurityPostureEtag(deploymentID),
		"annotations":              map[string]string{"env": "test", "service": "securityposture"},
		"reconciling":              false,
		"desiredPostureId":         gcpSecurityPosturePostureName(orgID, location, "posture-1"),
		"desiredPostureRevisionId": "0000000a",
		"failureMessage":           "",
	}
}

func gcpSecurityPostureTemplate(orgID, location, templateID, revisionID string) map[string]any {
	return map[string]any{
		"name":        gcpSecurityPostureTemplateName(orgID, location, templateID),
		"revisionId":  revisionID,
		"description": "Stackyard template " + templateID,
		"state":       "ACTIVE",
		"policySets": []any{
			map[string]any{
				"policySetId": "template-baseline",
				"policies": []any{
					map[string]any{
						"policyId":    "template-sha-001",
						"description": "Template SHA policy",
						"constraint":  map[string]any{"securityHealthAnalyticsModule": map[string]any{"moduleName": "PUBLIC_BUCKET_ACL", "moduleEnablementState": "ENABLED"}},
					},
				},
			},
		},
	}
}

func gcpSecurityPostureOperation(orgID, location, operationID, target, verb string, done bool) map[string]any {
	metadata := map[string]any{
		"@type":                 "type.googleapis.com/google.cloud.securityposture.v1.OperationMetadata",
		"createTime":            gcpSecurityPostureReferenceTime.Format(time.RFC3339),
		"endTime":               gcpSecurityPostureReferenceTime.Add(30 * time.Second).Format(time.RFC3339),
		"target":                target,
		"verb":                  verb,
		"statusMessage":         "",
		"requestedCancellation": false,
		"apiVersion":            "v1",
	}
	if !done {
		delete(metadata, "endTime")
	}
	return map[string]any{
		"name":     gcpSecurityPostureOperationName(orgID, location, operationID),
		"metadata": metadata,
		"done":     done,
	}
}

func respondGCPSecurityPostureInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPSecurityPostureFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPSecurityPostureAborted(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "Aborted",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_securityposture(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "securityposture") {
		if r.URL.Query().Get("stackyard_contract_probe") != "1" {
			return false
		}
		parts := strings.Split(strings.Trim(normalizeGCPSecurityPosturePath(path), "/"), "/")
		if len(parts) != 7 || parts[0] != "gcp" || !strings.HasPrefix(parts[1], "v1") || parts[2] != "organizations" || parts[4] != "locations" || parts[6] != "securityposture" {
			return false
		}
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSecurityPostureInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "organizations/123456/locations/global/postures/posture-1",
			"service":  "securityposture",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}

func parseGCPSecurityPostureParentName(parent string) (orgID, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "organizations" || parts[2] != "locations" {
		return "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if orgID == "" || location == "" {
		return "", "", false
	}
	return orgID, location, true
}

func parseGCPSecurityPosturePostureName(name string) (orgID, location, postureID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "organizations" || parts[2] != "locations" || parts[4] != "postures" {
		return "", "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	postureID = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" || postureID == "" {
		return "", "", "", false
	}
	return orgID, location, postureID, true
}

func parseGCPSecurityPostureDeploymentName(name string) (orgID, location, deploymentID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "organizations" || parts[2] != "locations" || parts[4] != "postureDeployments" {
		return "", "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	deploymentID = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" || deploymentID == "" {
		return "", "", "", false
	}
	return orgID, location, deploymentID, true
}

func parseGCPSecurityPostureTemplateName(name string) (orgID, location, templateID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "organizations" || parts[2] != "locations" || parts[4] != "postureTemplates" {
		return "", "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	templateID = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" || templateID == "" {
		return "", "", "", false
	}
	return orgID, location, templateID, true
}

func parseGCPSecurityPostureOperationName(name string) (orgID, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "organizations" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	operationID = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" || operationID == "" {
		return "", "", "", false
	}
	return orgID, location, operationID, true
}
