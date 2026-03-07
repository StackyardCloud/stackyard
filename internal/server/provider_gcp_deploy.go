package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudDeployRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPCloudDeployPath(path, hasGCPCloudDeployHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPDeployListDeliveryPipelines(w, r, path) {
			return true
		}
		if handleGCPDeployGetDeliveryPipeline(w, path) {
			return true
		}
		if handleGCPDeployListTargets(w, r, path) {
			return true
		}
		if handleGCPDeployGetTarget(w, path) {
			return true
		}
		if handleGCPDeployListReleases(w, r, path) {
			return true
		}
		if handleGCPDeployGetRelease(w, path) {
			return true
		}
		if handleGCPDeployListRollouts(w, r, path) {
			return true
		}
		if handleGCPDeployGetRollout(w, path) {
			return true
		}
		if handleGCPDeployListJobRuns(w, r, path) {
			return true
		}
		if handleGCPDeployGetJobRun(w, path) {
			return true
		}
		if handleGCPDeployListDeployPolicies(w, r, path) {
			return true
		}
		if handleGCPDeployGetDeployPolicy(w, path) {
			return true
		}
		if handleGCPDeployGetConfig(w, path) {
			return true
		}
		if handleGCPDeployGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPDeployListOperations(w, r, path) {
			return true
		}
		if handleGCPDeployGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDeployCreateDeliveryPipeline(w, r, path) {
			return true
		}
		if handleGCPDeployCreateTarget(w, r, path) {
			return true
		}
		if handleGCPDeployRollbackTarget(w, path) {
			return true
		}
		if handleGCPDeployCreateRelease(w, r, path) {
			return true
		}
		if handleGCPDeployAbandonRelease(w, path) {
			return true
		}
		if handleGCPDeployCreateRollout(w, r, path) {
			return true
		}
		if handleGCPDeployApproveRollout(w, path) {
			return true
		}
		if handleGCPDeployAdvanceRollout(w, r, path) {
			return true
		}
		if handleGCPDeployCancelRollout(w, path) {
			return true
		}
		if handleGCPDeployIgnoreJob(w, r, path) {
			return true
		}
		if handleGCPDeployRetryJob(w, r, path) {
			return true
		}
		if handleGCPDeployTerminateJobRun(w, path) {
			return true
		}
		if handleGCPDeploySetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPDeployTestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPDeployCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPDeployUpdateDeliveryPipeline(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPDeployDeleteDeliveryPipeline(w, path) {
			return true
		}
		if handleGCPDeployDeleteTarget(w, path) {
			return true
		}
		if handleGCPDeployDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPCloudDeployHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "deploy" || service == "clouddeploy" {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-deploy-apiv1")
}

func isGCPCloudDeployPath(path string, includeOperations bool) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || project == "" || location == "" || len(tail) == 0 {
		return false
	}
	if isGCPDeployDeliveryPipelinesCollectionTail(tail) || isGCPDeployDeliveryPipelineTail(tail) || isGCPDeployDeliveryPipelineActionTail(tail) {
		return true
	}
	if isGCPDeployTargetsCollectionTail(tail) || isGCPDeployTargetTail(tail) || isGCPDeployTargetActionTail(tail) {
		return true
	}
	if isGCPDeployReleasesCollectionTail(tail) || isGCPDeployReleaseTail(tail) || isGCPDeployReleaseActionTail(tail) {
		return true
	}
	if isGCPDeployRolloutsCollectionTail(tail) || isGCPDeployRolloutTail(tail) || isGCPDeployRolloutActionTail(tail) {
		return true
	}
	if isGCPDeployJobRunsCollectionTail(tail) || isGCPDeployJobRunTail(tail) || isGCPDeployJobRunActionTail(tail) {
		return true
	}
	if isGCPDeployDeployPoliciesCollectionTail(tail) || isGCPDeployDeployPolicyTail(tail) {
		return true
	}
	if isGCPDeployConfigTail(tail) {
		return true
	}
	if includeOperations && (isGCPDeployOperationsCollectionTail(tail) || isGCPDeployOperationTail(tail) || isGCPDeployOperationActionTail(tail)) {
		return true
	}
	return false
}

func handleGCPDeployListDeliveryPipelines(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelinesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDeployPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeployDeliveryPipeline(project, location, "team-pipeline")}
	return respondGCPDeployList(w, "deliveryPipelines", items, pageSize, start, path)
}

func handleGCPDeployGetDeliveryPipeline(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelineTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployDeliveryPipeline(project, location, tail[1]))
	return true
}

func handleGCPDeployCreateDeliveryPipeline(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelinesCollectionTail(tail) {
		return false
	}
	pipelineID := strings.TrimSpace(r.URL.Query().Get("deliveryPipelineId"))
	if pipelineID == "" {
		respondGCPDeployInvalidArgument(w, path, "deliveryPipelineId is required")
		return true
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	pipeline, _ := body["deliveryPipeline"].(map[string]any)
	if len(pipeline) == 0 {
		pipeline = body
	}
	if len(pipeline) == 0 {
		respondGCPDeployInvalidArgument(w, path, "deliveryPipeline is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, "clouddeploy.createDeliveryPipeline."+pipelineID))
	return true
}

func handleGCPDeployUpdateDeliveryPipeline(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelineTail(tail) {
		return false
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	pipeline, _ := body["deliveryPipeline"].(map[string]any)
	if len(pipeline) == 0 {
		pipeline = body
	}
	name, _ := pipeline["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPDeployInvalidArgument(w, path, "deliveryPipeline.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, "clouddeploy.updateDeliveryPipeline."+tail[1]))
	return true
}

func handleGCPDeployDeleteDeliveryPipeline(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelineTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, "clouddeploy.deleteDeliveryPipeline."+tail[1]))
	return true
}

func handleGCPDeployListTargets(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployTargetsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDeployPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeployTarget(project, location, "team-target")}
	return respondGCPDeployList(w, "targets", items, pageSize, start, path)
}

func handleGCPDeployGetTarget(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployTargetTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployTarget(project, location, tail[1]))
	return true
}

func handleGCPDeployCreateTarget(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployTargetsCollectionTail(tail) {
		return false
	}
	targetID := strings.TrimSpace(r.URL.Query().Get("targetId"))
	if targetID == "" {
		respondGCPDeployInvalidArgument(w, path, "targetId is required")
		return true
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	target, _ := body["target"].(map[string]any)
	if len(target) == 0 {
		target = body
	}
	if len(target) == 0 {
		respondGCPDeployInvalidArgument(w, path, "target is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, "clouddeploy.createTarget."+targetID))
	return true
}

func handleGCPDeployRollbackTarget(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployTargetActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[1]), ":rollbackTarget") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"rollbackConfig": map[string]any{},
	})
	return true
}

func handleGCPDeployDeleteTarget(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployTargetTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, "clouddeploy.deleteTarget."+tail[1]))
	return true
}

func handleGCPDeployListReleases(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployReleasesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDeployPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeployRelease(project, location, tail[1], "release-1")}
	return respondGCPDeployList(w, "releases", items, pageSize, start, path)
}

func handleGCPDeployGetRelease(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployReleaseTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployRelease(project, location, tail[1], tail[3]))
	return true
}

func handleGCPDeployCreateRelease(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployReleasesCollectionTail(tail) {
		return false
	}
	releaseID := strings.TrimSpace(r.URL.Query().Get("releaseId"))
	if releaseID == "" {
		respondGCPDeployInvalidArgument(w, path, "releaseId is required")
		return true
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	release, _ := body["release"].(map[string]any)
	if len(release) == 0 {
		release = body
	}
	if len(release) == 0 {
		respondGCPDeployInvalidArgument(w, path, "release is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, "clouddeploy.createRelease."+tail[1]+"."+releaseID))
	return true
}

func handleGCPDeployAbandonRelease(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployReleaseActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[3]), ":abandon") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployListRollouts(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDeployPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeployRollout(project, location, tail[1], tail[3], "rollout-1")}
	return respondGCPDeployList(w, "rollouts", items, pageSize, start, path)
}

func handleGCPDeployGetRollout(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployRollout(project, location, tail[1], tail[3], tail[5]))
	return true
}

func handleGCPDeployCreateRollout(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutsCollectionTail(tail) {
		return false
	}
	rolloutID := strings.TrimSpace(r.URL.Query().Get("rolloutId"))
	if rolloutID == "" {
		respondGCPDeployInvalidArgument(w, path, "rolloutId is required")
		return true
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	rollout, _ := body["rollout"].(map[string]any)
	if len(rollout) == 0 {
		rollout = body
	}
	if len(rollout) == 0 {
		respondGCPDeployInvalidArgument(w, path, "rollout is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, "clouddeploy.createRollout."+tail[1]+"."+tail[3]+"."+rolloutID))
	return true
}

func handleGCPDeployApproveRollout(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[5]), ":approve") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployAdvanceRollout(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[5]), ":advance") {
		return false
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	phaseID, _ := body["phaseId"].(string)
	if strings.TrimSpace(phaseID) == "" {
		respondGCPDeployInvalidArgument(w, path, "phaseId is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployCancelRollout(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[5]), ":cancel") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployListJobRuns(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployJobRunsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDeployPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeployJobRun(project, location, tail[1], tail[3], tail[5], "job-1")}
	return respondGCPDeployList(w, "jobRuns", items, pageSize, start, path)
}

func handleGCPDeployGetJobRun(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployJobRunTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployJobRun(project, location, tail[1], tail[3], tail[5], tail[7]))
	return true
}

func handleGCPDeployIgnoreJob(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[5]), ":ignoreJob") {
		return false
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	phaseID, _ := body["phaseId"].(string)
	jobID, _ := body["jobId"].(string)
	if strings.TrimSpace(phaseID) == "" || strings.TrimSpace(jobID) == "" {
		respondGCPDeployInvalidArgument(w, path, "phaseId and jobId are required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployRetryJob(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployRolloutActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[5]), ":retryJob") {
		return false
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	phaseID, _ := body["phaseId"].(string)
	jobID, _ := body["jobId"].(string)
	if strings.TrimSpace(phaseID) == "" || strings.TrimSpace(jobID) == "" {
		respondGCPDeployInvalidArgument(w, path, "phaseId and jobId are required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployTerminateJobRun(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployJobRunActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[7]), ":terminate") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployListDeployPolicies(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeployPoliciesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDeployPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeployDeployPolicy(project, location, "policy-1")}
	return respondGCPDeployList(w, "deployPolicies", items, pageSize, start, path)
}

func handleGCPDeployGetDeployPolicy(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeployPolicyTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployDeployPolicy(project, location, tail[1]))
	return true
}

func handleGCPDeployGetConfig(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployConfigTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployConfig(project, location))
	return true
}

func handleGCPDeployGetIAMPolicy(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelineActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[1]), ":getIamPolicy") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"version": 1,
		"bindings": []any{
			map[string]any{
				"role": "roles/clouddeploy.viewer",
				"members": []any{
					"user:dev@stackyard.local",
				},
			},
		},
		"etag": "c3RhY2t5YXJkLWRlcGxveS1wb2xpY3k=",
	})
	return true
}

func handleGCPDeploySetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelineActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[1]), ":setIamPolicy") {
		return false
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, ok := body["policy"]; !ok {
		respondGCPDeployInvalidArgument(w, path, "policy is required")
		return true
	}
	policy, _ := body["policy"].(map[string]any)
	etag, _ := policy["etag"].(string)
	if strings.TrimSpace(etag) == "" {
		etag = "c3RhY2t5YXJkLWRlcGxveS1wb2xpY3k="
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"version":  1,
		"bindings": []any{},
		"etag":     etag,
	})
	return true
}

func handleGCPDeployTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployDeliveryPipelineActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[1]), ":testIamPermissions") {
		return false
	}
	body, valid := decodeGCPDeployJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissions, _ := body["permissions"].([]any)
	if len(permissions) == 0 {
		respondGCPDeployInvalidArgument(w, path, "permissions must contain at least one value")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{"permissions": permissions})
	return true
}

func handleGCPDeployListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDeployPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDeployOperation(project, location, "op-1")}
	return respondGCPDeployList(w, "operations", items, pageSize, start, path)
}

func handleGCPDeployGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployOperationTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDeployOperation(project, location, tail[1]))
	return true
}

func handleGCPDeployCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployOperationActionTail(tail) || !strings.HasSuffix(normalizeGCPDeployActionSegment(tail[1]), ":cancel") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDeployDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDeployLocationTail(path)
	if !ok || !isGCPDeployOperationTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPDeployLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	tail = parts[6:]
	return project, location, tail, len(tail) > 0
}

func isGCPDeployDeliveryPipelinesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "deliveryPipelines"
}

func isGCPDeployDeliveryPipelineTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "deliveryPipelines" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDeployActionSegment(tail[1]), ":")
}

func isGCPDeployDeliveryPipelineActionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "deliveryPipelines" {
		return false
	}
	resourceAction := normalizeGCPDeployActionSegment(tail[1])
	resource, action, ok := strings.Cut(resourceAction, ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDeployTargetsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "targets"
}

func isGCPDeployTargetTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "targets" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDeployActionSegment(tail[1]), ":")
}

func isGCPDeployTargetActionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "targets" {
		return false
	}
	resourceAction := normalizeGCPDeployActionSegment(tail[1])
	resource, action, ok := strings.Cut(resourceAction, ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDeployReleasesCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "deliveryPipelines" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases"
}

func isGCPDeployReleaseTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "deliveryPipelines" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != "" && !strings.Contains(normalizeGCPDeployActionSegment(tail[3]), ":")
}

func isGCPDeployReleaseActionTail(tail []string) bool {
	if len(tail) != 4 || tail[0] != "deliveryPipelines" || strings.TrimSpace(tail[1]) == "" || tail[2] != "releases" {
		return false
	}
	resourceAction := normalizeGCPDeployActionSegment(tail[3])
	resource, action, ok := strings.Cut(resourceAction, ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDeployRolloutsCollectionTail(tail []string) bool {
	return len(tail) == 5 && tail[0] == "deliveryPipelines" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != "" && tail[4] == "rollouts"
}

func isGCPDeployRolloutTail(tail []string) bool {
	return len(tail) == 6 && tail[0] == "deliveryPipelines" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != "" && tail[4] == "rollouts" && strings.TrimSpace(tail[5]) != "" && !strings.Contains(normalizeGCPDeployActionSegment(tail[5]), ":")
}

func isGCPDeployRolloutActionTail(tail []string) bool {
	if len(tail) != 6 || tail[0] != "deliveryPipelines" || strings.TrimSpace(tail[1]) == "" || tail[2] != "releases" || strings.TrimSpace(tail[3]) == "" || tail[4] != "rollouts" {
		return false
	}
	resourceAction := normalizeGCPDeployActionSegment(tail[5])
	resource, action, ok := strings.Cut(resourceAction, ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDeployJobRunsCollectionTail(tail []string) bool {
	return len(tail) == 7 && tail[0] == "deliveryPipelines" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != "" && tail[4] == "rollouts" && strings.TrimSpace(tail[5]) != "" && tail[6] == "jobRuns"
}

func isGCPDeployJobRunTail(tail []string) bool {
	return len(tail) == 8 && tail[0] == "deliveryPipelines" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != "" && tail[4] == "rollouts" && strings.TrimSpace(tail[5]) != "" && tail[6] == "jobRuns" && strings.TrimSpace(tail[7]) != "" && !strings.Contains(normalizeGCPDeployActionSegment(tail[7]), ":")
}

func isGCPDeployJobRunActionTail(tail []string) bool {
	if len(tail) != 8 || tail[0] != "deliveryPipelines" || strings.TrimSpace(tail[1]) == "" || tail[2] != "releases" || strings.TrimSpace(tail[3]) == "" || tail[4] != "rollouts" || strings.TrimSpace(tail[5]) == "" || tail[6] != "jobRuns" {
		return false
	}
	resourceAction := normalizeGCPDeployActionSegment(tail[7])
	resource, action, ok := strings.Cut(resourceAction, ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDeployDeployPoliciesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "deployPolicies"
}

func isGCPDeployDeployPolicyTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "deployPolicies" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDeployActionSegment(tail[1]), ":")
}

func isGCPDeployConfigTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "config"
}

func isGCPDeployOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPDeployOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDeployActionSegment(tail[1]), ":")
}

func isGCPDeployOperationActionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	resourceAction := normalizeGCPDeployActionSegment(tail[1])
	resource, action, ok := strings.Cut(resourceAction, ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func parseGCPDeployPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDeployInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPDeployInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPDeployList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDeployInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPDeployJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDeployInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func normalizeGCPDeployActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpDeployDeliveryPipeline(project, location, pipelineID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/deliveryPipelines/%s", project, location, pipelineID),
		"description": "Team delivery pipeline",
		"suspended":   false,
	}
}

func gcpDeployTarget(project, location, targetID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/targets/%s", project, location, targetID),
		"targetId":    targetID,
		"description": "Team target",
	}
}

func gcpDeployRelease(project, location, pipelineID, releaseID string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/locations/%s/deliveryPipelines/%s/releases/%s", project, location, pipelineID, releaseID),
		"releaseId": releaseID,
		"abandoned": false,
	}
}

func gcpDeployRollout(project, location, pipelineID, releaseID, rolloutID string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/locations/%s/deliveryPipelines/%s/releases/%s/rollouts/%s", project, location, pipelineID, releaseID, rolloutID),
		"rolloutId": rolloutID,
		"state":     "SUCCEEDED",
	}
}

func gcpDeployJobRun(project, location, pipelineID, releaseID, rolloutID, jobRunID string) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/deliveryPipelines/%s/releases/%s/rollouts/%s/jobRuns/%s", project, location, pipelineID, releaseID, rolloutID, jobRunID),
		"state": "SUCCEEDED",
	}
}

func gcpDeployDeployPolicy(project, location, policyID string) map[string]any {
	return map[string]any{
		"name":           fmt.Sprintf("projects/%s/locations/%s/deployPolicies/%s", project, location, policyID),
		"deployPolicyId": policyID,
		"description":    "Stackyard policy",
		"annotations":    map[string]any{"stackyard": "true"},
	}
}

func gcpDeployConfig(project, location string) map[string]any {
	return map[string]any{
		"name":                   fmt.Sprintf("projects/%s/locations/%s/config", project, location),
		"defaultSkaffoldVersion": "2.13.0",
		"supportedVersions": []any{
			map[string]any{"version": "2.13.0"},
		},
	}
}

func gcpDeployOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": true,
	}
}

func respondGCPDeployInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
