package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPDataprocRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPDataprocPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPDataprocListClusters(w, r, path) {
			return true
		}
		if handleGCPDataprocGetCluster(w, path) {
			return true
		}
		if handleGCPDataprocListJobs(w, r, path) {
			return true
		}
		if handleGCPDataprocGetJob(w, path) {
			return true
		}
		if handleGCPDataprocListWorkflowTemplates(w, r, path) {
			return true
		}
		if handleGCPDataprocGetWorkflowTemplate(w, path) {
			return true
		}
		if handleGCPDataprocListAutoscalingPolicies(w, r, path) {
			return true
		}
		if handleGCPDataprocGetAutoscalingPolicy(w, path) {
			return true
		}
		if handleGCPDataprocListBatches(w, r, path) {
			return true
		}
		if handleGCPDataprocGetBatch(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDataprocCreateCluster(w, r, path) {
			return true
		}
		if handleGCPDataprocStartCluster(w, path) {
			return true
		}
		if handleGCPDataprocStopCluster(w, path) {
			return true
		}
		if handleGCPDataprocDiagnoseCluster(w, path) {
			return true
		}
		if handleGCPDataprocSubmitJob(w, r, path) {
			return true
		}
		if handleGCPDataprocSubmitJobAsOperation(w, r, path) {
			return true
		}
		if handleGCPDataprocCancelJob(w, path) {
			return true
		}
		if handleGCPDataprocCreateWorkflowTemplate(w, r, path) {
			return true
		}
		if handleGCPDataprocInstantiateWorkflowTemplate(w, path) {
			return true
		}
		if handleGCPDataprocInstantiateInlineWorkflowTemplate(w, r, path) {
			return true
		}
		if handleGCPDataprocCreateAutoscalingPolicy(w, r, path) {
			return true
		}
		if handleGCPDataprocCreateBatch(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPDataprocUpdateCluster(w, r, path) {
			return true
		}
		if handleGCPDataprocUpdateJob(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPut:
		if handleGCPDataprocUpdateWorkflowTemplate(w, r, path) {
			return true
		}
		if handleGCPDataprocUpdateAutoscalingPolicy(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPDataprocDeleteCluster(w, path) {
			return true
		}
		if handleGCPDataprocDeleteJob(w, path) {
			return true
		}
		if handleGCPDataprocDeleteWorkflowTemplate(w, path) {
			return true
		}
		if handleGCPDataprocDeleteAutoscalingPolicy(w, path) {
			return true
		}
		if handleGCPDataprocDeleteBatch(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPDataprocHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "dataproc" {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-dataproc-apiv1")
}

func isGCPDataprocPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}

	if _, _, tail, ok := parseGCPDataprocRegionTail(path); ok {
		if isGCPDataprocClustersCollectionTail(tail) || isGCPDataprocClusterTail(tail) || isGCPDataprocClusterActionTail(tail) {
			return true
		}
		if isGCPDataprocJobsCollectionTail(tail) || isGCPDataprocJobTail(tail) || isGCPDataprocJobActionTail(tail) || isGCPDataprocJobsActionTail(tail) {
			return true
		}
		if isGCPDataprocWorkflowTemplatesCollectionTail(tail) || isGCPDataprocWorkflowTemplateTail(tail) || isGCPDataprocWorkflowTemplateActionTail(tail) || isGCPDataprocWorkflowTemplatesActionTail(tail) {
			return true
		}
		if isGCPDataprocAutoscalingPoliciesCollectionTail(tail) || isGCPDataprocAutoscalingPolicyTail(tail) {
			return true
		}
	}

	if _, _, tail, ok := parseGCPDataprocLocationTail(path); ok {
		if isGCPDataprocBatchesCollectionTail(tail) || isGCPDataprocBatchTail(tail) {
			return true
		}
		if isGCPDataprocWorkflowTemplatesActionTail(tail) {
			return true
		}
	}

	return false
}

func handleGCPDataprocListClusters(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClustersCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDataprocPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDataprocCluster(project, region, "team-cluster")}
	return respondGCPDataprocList(w, "clusters", items, pageSize, start, path)
}

func handleGCPDataprocGetCluster(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClusterTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataprocCluster(project, region, tail[1]))
	return true
}

func handleGCPDataprocCreateCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClustersCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	cluster := gcpDataprocBodyMap(body, "cluster")
	if len(cluster) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "cluster is required")
		return true
	}
	clusterName := strings.TrimSpace(stringFromMap(cluster, "clusterName"))
	if clusterName == "" {
		clusterName = "team-cluster"
	}
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.createCluster."+clusterName))
	return true
}

func handleGCPDataprocUpdateCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClusterTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	cluster := gcpDataprocBodyMap(body, "cluster")
	if len(cluster) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "cluster is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.updateCluster."+tail[1]))
	return true
}

func handleGCPDataprocStartCluster(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClusterActionTail(tail) || !strings.HasSuffix(normalizeGCPDataprocActionSegment(tail[1]), ":start") {
		return false
	}
	clusterID, _, _ := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.startCluster."+clusterID))
	return true
}

func handleGCPDataprocStopCluster(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClusterActionTail(tail) || !strings.HasSuffix(normalizeGCPDataprocActionSegment(tail[1]), ":stop") {
		return false
	}
	clusterID, _, _ := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.stopCluster."+clusterID))
	return true
}

func handleGCPDataprocDiagnoseCluster(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClusterActionTail(tail) || !strings.HasSuffix(normalizeGCPDataprocActionSegment(tail[1]), ":diagnose") {
		return false
	}
	clusterID, _, _ := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.diagnoseCluster."+clusterID))
	return true
}

func handleGCPDataprocDeleteCluster(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocClusterTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.deleteCluster."+tail[1]))
	return true
}

func handleGCPDataprocSubmitJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocJobsActionTail(tail) || !strings.HasSuffix(normalizeGCPDataprocActionSegment(tail[0]), ":submit") {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	job := gcpDataprocBodyMap(body, "job")
	if len(job) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "job is required")
		return true
	}
	jobID := gcpDataprocExtractJobID(job)
	respondJSON(w, http.StatusOK, gcpDataprocJob(project, region, jobID, "PENDING"))
	return true
}

func handleGCPDataprocSubmitJobAsOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocJobsActionTail(tail) || !strings.HasSuffix(normalizeGCPDataprocActionSegment(tail[0]), ":submitAsOperation") {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	job := gcpDataprocBodyMap(body, "job")
	if len(job) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "job is required")
		return true
	}
	jobID := gcpDataprocExtractJobID(job)
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.submitJobAsOperation."+jobID))
	return true
}

func handleGCPDataprocListJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocJobsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDataprocPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDataprocJob(project, region, "team-job", "RUNNING")}
	return respondGCPDataprocList(w, "jobs", items, pageSize, start, path)
}

func handleGCPDataprocGetJob(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocJobTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataprocJob(project, region, tail[1], "RUNNING"))
	return true
}

func handleGCPDataprocUpdateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocJobTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	job := gcpDataprocBodyMap(body, "job")
	if len(job) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "job is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataprocJob(project, region, tail[1], "RUNNING"))
	return true
}

func handleGCPDataprocCancelJob(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocJobActionTail(tail) || !strings.HasSuffix(normalizeGCPDataprocActionSegment(tail[1]), ":cancel") {
		return false
	}
	jobID, _, _ := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	respondJSON(w, http.StatusOK, gcpDataprocJob(project, region, jobID, "CANCEL_PENDING"))
	return true
}

func handleGCPDataprocDeleteJob(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocJobTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDataprocListWorkflowTemplates(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocWorkflowTemplatesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDataprocPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDataprocWorkflowTemplate(project, region, "team-template")}
	return respondGCPDataprocList(w, "templates", items, pageSize, start, path)
}

func handleGCPDataprocGetWorkflowTemplate(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocWorkflowTemplateTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataprocWorkflowTemplate(project, region, tail[1]))
	return true
}

func handleGCPDataprocCreateWorkflowTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocWorkflowTemplatesCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	template := gcpDataprocBodyMap(body, "template")
	if len(template) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "template is required")
		return true
	}
	templateID := strings.TrimSpace(stringFromMap(template, "id"))
	if templateID == "" {
		templateID = "team-template"
	}
	respondJSON(w, http.StatusOK, gcpDataprocWorkflowTemplate(project, region, templateID))
	return true
}

func handleGCPDataprocUpdateWorkflowTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocWorkflowTemplateTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	template := gcpDataprocBodyMap(body, "template")
	if len(template) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "template is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataprocWorkflowTemplate(project, region, tail[1]))
	return true
}

func handleGCPDataprocInstantiateWorkflowTemplate(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocWorkflowTemplateActionTail(tail) || !strings.HasSuffix(normalizeGCPDataprocActionSegment(tail[1]), ":instantiate") {
		return false
	}
	templateID, _, _ := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.instantiateWorkflowTemplate."+templateID))
	return true
}

func handleGCPDataprocInstantiateInlineWorkflowTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	if project, region, tail, ok := parseGCPDataprocRegionTail(path); ok && isGCPDataprocWorkflowTemplatesActionTail(tail) {
		body, valid := decodeGCPDataprocJSONBody(w, r, path)
		if !valid {
			return true
		}
		template := gcpDataprocBodyMap(body, "template")
		if len(template) == 0 {
			respondGCPDataprocInvalidArgument(w, path, "template is required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpDataprocRegionOperation(project, region, "dataproc.instantiateInlineWorkflowTemplate"))
		return true
	}

	project, location, tail, ok := parseGCPDataprocLocationTail(path)
	if !ok || !isGCPDataprocWorkflowTemplatesActionTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	template := gcpDataprocBodyMap(body, "template")
	if len(template) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "template is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataprocLocationOperation(project, location, "dataproc.instantiateInlineWorkflowTemplate"))
	return true
}

func handleGCPDataprocDeleteWorkflowTemplate(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocWorkflowTemplateTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDataprocListAutoscalingPolicies(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocAutoscalingPoliciesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDataprocPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDataprocAutoscalingPolicy(project, region, "team-policy")}
	return respondGCPDataprocList(w, "policies", items, pageSize, start, path)
}

func handleGCPDataprocGetAutoscalingPolicy(w http.ResponseWriter, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocAutoscalingPolicyTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataprocAutoscalingPolicy(project, region, tail[1]))
	return true
}

func handleGCPDataprocCreateAutoscalingPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocAutoscalingPoliciesCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpDataprocBodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "policy is required")
		return true
	}
	policyID := strings.TrimSpace(stringFromMap(policy, "id"))
	if policyID == "" {
		policyID = "team-policy"
	}
	respondJSON(w, http.StatusOK, gcpDataprocAutoscalingPolicy(project, region, policyID))
	return true
}

func handleGCPDataprocUpdateAutoscalingPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	project, region, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocAutoscalingPolicyTail(tail) {
		return false
	}
	body, valid := decodeGCPDataprocJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpDataprocBodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPDataprocInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataprocAutoscalingPolicy(project, region, tail[1]))
	return true
}

func handleGCPDataprocDeleteAutoscalingPolicy(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDataprocRegionTail(path)
	if !ok || !isGCPDataprocAutoscalingPolicyTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDataprocListBatches(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDataprocLocationTail(path)
	if !ok || !isGCPDataprocBatchesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPDataprocPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDataprocBatch(project, location, "team-batch")}
	return respondGCPDataprocList(w, "batches", items, pageSize, start, path)
}

func handleGCPDataprocGetBatch(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDataprocLocationTail(path)
	if !ok || !isGCPDataprocBatchTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataprocBatch(project, location, tail[1]))
	return true
}

func handleGCPDataprocCreateBatch(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDataprocLocationTail(path)
	if !ok || !isGCPDataprocBatchesCollectionTail(tail) {
		return false
	}
	batchID := strings.TrimSpace(r.URL.Query().Get("batchId"))
	if batchID == "" {
		respondGCPDataprocInvalidArgument(w, path, "batchId is required")
		return true
	}
	if _, valid := decodeGCPDataprocJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataprocLocationOperation(project, location, "dataproc.createBatch."+batchID))
	return true
}

func handleGCPDataprocDeleteBatch(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDataprocLocationTail(path)
	if !ok || !isGCPDataprocBatchTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPDataprocRegionTail(path string) (project, region string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "regions" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	region = strings.TrimSpace(parts[5])
	if project == "" || region == "" {
		return "", "", nil, false
	}
	tail = parts[6:]
	return project, region, tail, len(tail) > 0
}

func parseGCPDataprocLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func isGCPDataprocClustersCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "clusters"
}

func isGCPDataprocClusterTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "clusters" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDataprocActionSegment(tail[1]), ":")
}

func isGCPDataprocClusterActionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "clusters" {
		return false
	}
	resource, action, ok := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDataprocJobsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "jobs"
}

func isGCPDataprocJobsActionTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	resource, action, ok := strings.Cut(normalizeGCPDataprocActionSegment(tail[0]), ":")
	return ok && resource == "jobs" && strings.TrimSpace(action) != ""
}

func isGCPDataprocJobTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "jobs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDataprocActionSegment(tail[1]), ":")
}

func isGCPDataprocJobActionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "jobs" {
		return false
	}
	resource, action, ok := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDataprocWorkflowTemplatesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "workflowTemplates"
}

func isGCPDataprocWorkflowTemplatesActionTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	resource, action, ok := strings.Cut(normalizeGCPDataprocActionSegment(tail[0]), ":")
	return ok && resource == "workflowTemplates" && strings.TrimSpace(action) != ""
}

func isGCPDataprocWorkflowTemplateTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "workflowTemplates" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDataprocActionSegment(tail[1]), ":")
}

func isGCPDataprocWorkflowTemplateActionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "workflowTemplates" {
		return false
	}
	resource, action, ok := strings.Cut(normalizeGCPDataprocActionSegment(tail[1]), ":")
	return ok && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) != ""
}

func isGCPDataprocAutoscalingPoliciesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "autoscalingPolicies"
}

func isGCPDataprocAutoscalingPolicyTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "autoscalingPolicies" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDataprocActionSegment(tail[1]), ":")
}

func isGCPDataprocBatchesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "batches"
}

func isGCPDataprocBatchTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "batches" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(normalizeGCPDataprocActionSegment(tail[1]), ":")
}

func parseGCPDataprocPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDataprocInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPDataprocInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPDataprocList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDataprocInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPDataprocJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDataprocInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpDataprocBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func gcpDataprocExtractJobID(job map[string]any) string {
	if ref, ok := job["reference"].(map[string]any); ok {
		if jobID, ok := ref["jobId"].(string); ok && strings.TrimSpace(jobID) != "" {
			return strings.TrimSpace(jobID)
		}
	}
	if jobID, ok := job["jobId"].(string); ok && strings.TrimSpace(jobID) != "" {
		return strings.TrimSpace(jobID)
	}
	return "team-job"
}

func normalizeGCPDataprocActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpDataprocCluster(project, region, clusterID string) map[string]any {
	return map[string]any{
		"clusterName": clusterID,
		"clusterUuid": "00000000-0000-0000-0000-000000000001",
		"status": map[string]any{
			"state": "RUNNING",
		},
		"projectId": project,
		"config": map[string]any{
			"configBucket": fmt.Sprintf("%s-dataproc-%s", project, region),
		},
	}
}

func gcpDataprocJob(project, region, jobID, state string) map[string]any {
	return map[string]any{
		"reference": map[string]any{
			"projectId": project,
			"jobId":     jobID,
		},
		"placement": map[string]any{
			"clusterName": "team-cluster",
		},
		"status": map[string]any{
			"state": state,
		},
		"labels": map[string]any{
			"region": region,
		},
	}
}

func gcpDataprocWorkflowTemplate(project, region, templateID string) map[string]any {
	return map[string]any{
		"id":   templateID,
		"name": fmt.Sprintf("projects/%s/regions/%s/workflowTemplates/%s", project, region, templateID),
		"dagTimeout": map[string]any{
			"seconds": "3600",
		},
	}
}

func gcpDataprocAutoscalingPolicy(project, region, policyID string) map[string]any {
	return map[string]any{
		"id":   policyID,
		"name": fmt.Sprintf("projects/%s/regions/%s/autoscalingPolicies/%s", project, region, policyID),
		"workerConfig": map[string]any{
			"minInstances": 2,
			"maxInstances": 6,
		},
	}
}

func gcpDataprocBatch(project, location, batchID string) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/batches/%s", project, location, batchID),
		"state": "SUCCEEDED",
		"runtimeInfo": map[string]any{
			"endpoints": map[string]any{
				"historyServerEndpoint": "https://history.stackyard.local",
			},
		},
	}
}

func gcpDataprocRegionOperation(project, region, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/regions/%s/operations/%s", project, region, operationID),
		"done": true,
	}
}

func gcpDataprocLocationOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": true,
	}
}

func respondGCPDataprocInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
