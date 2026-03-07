package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var gcpSecureSourceManagerReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSecureSourceManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_securesourcemanager(w, r) {
		return true
	}

	path := normalizeGCPSecureSourceManagerPath(rawRequestPath(r))
	if isGCPSecureSourceManagerLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSecureSourceManagerListLocations(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSecureSourceManagerPath(path, hasGCPSecureSourceManagerHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSecureSourceManagerListInstances(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetInstance(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListRepositories(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetRepository(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListHooks(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetHook(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetIamPolicyRepo(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListBranchRules(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetBranchRule(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetPullRequest(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListPullRequests(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerListPullRequestFileDiffs(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerFetchTree(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerFetchBlob(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetIssue(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListIssues(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetPullRequestComment(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListPullRequestComments(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetIssueComment(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListIssueComments(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerGetOperation(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerListOperations(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSecureSourceManagerCreateInstance(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerCreateRepository(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerCreateHook(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerSetIamPolicyRepo(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerTestIamPermissionsRepo(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerCreateBranchRule(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerCreatePullRequest(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerPullRequestAction(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerCreateIssue(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerIssueAction(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerCreatePullRequestComment(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerBatchCreatePullRequestComments(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerResolvePullRequestComments(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerUnresolvePullRequestComments(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerCreateIssueComment(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSecureSourceManagerUpdateRepository(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerUpdateHook(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerUpdateBranchRule(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerUpdatePullRequest(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerUpdateIssue(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerUpdatePullRequestComment(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerUpdateIssueComment(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSecureSourceManagerDeleteInstance(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerDeleteRepository(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerDeleteHook(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerDeleteBranchRule(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerDeleteIssue(w, r, path) {
			return true
		}
		if handleGCPSecureSourceManagerDeletePullRequestComment(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerDeleteIssueComment(w, path) {
			return true
		}
		if handleGCPSecureSourceManagerDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSecureSourceManagerPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSecureSourceManagerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "securesourcemanager", "securesourcemanager-apiv1", "secure-source-manager", "secure_source_manager", "gcp-secure-source-manager":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-securesourcemanager-apiv1") || strings.Contains(ua, "cloud.google.com/go/securesourcemanager")
}

func isGCPSecureSourceManagerLocationRequest(r *http.Request, path string) bool {
	return hasGCPSecureSourceManagerHint(r) && isGCPProjectLocationDiscoveryPath(path)
}

func isGCPSecureSourceManagerPath(path string, hasHint bool) bool {
	if _, _, tail, ok := parseGCPSecureSourceManagerLocationTail(path); ok && len(tail) > 0 {
		switch tail[0] {
		case "instances", "repositories", "operations":
			return true
		}
	}
	if _, _, _, _, ok := parseGCPSecureSourceManagerRepositoryIAMPath(path); ok {
		return true
	}
	if hasHint && strings.HasPrefix(path, "/gcp/v1/projects/") {
		return true
	}
	return false
}

func handleGCPSecureSourceManagerListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerLocation(project, "us-central1"),
		gcpSecureSourceManagerLocation(project, "global"),
	}
	return respondGCPSecureSourceManagerList(w, "locations", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerLocation(project, location))
	return true
}

func handleGCPSecureSourceManagerListInstances(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || !isGCPSecureSourceManagerInstancesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerInstance(project, location, "instance-1"),
		gcpSecureSourceManagerInstance(project, location, "instance-2"),
	}
	return respondGCPSecureSourceManagerList(w, "instances", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerGetInstance(w http.ResponseWriter, path string) bool {
	project, location, instanceID, ok := parseGCPSecureSourceManagerInstancePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerInstance(project, location, instanceID))
	return true
}

func handleGCPSecureSourceManagerCreateInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || !isGCPSecureSourceManagerInstancesCollectionTail(tail) {
		return false
	}
	instanceID := strings.TrimSpace(r.URL.Query().Get("instanceId"))
	if instanceID == "" || !isGCPSecureSourceManagerID(instanceID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "instanceId is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	instance := gcpSecureSourceManagerBodyMap(body, "instance")
	if len(instance) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "instance is required")
		return true
	}
	if displayName := strings.TrimSpace(gcpSecureSourceManagerString(instance, "displayName")); displayName == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "instance.displayName is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(instance, "name")); name != "" {
		expected := gcpSecureSourceManagerInstanceName(project, location, instanceID)
		if name != expected {
			respondGCPSecureSourceManagerInvalidArgument(w, path, "instance.name must match parent and instanceId")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createInstance."+instanceID, gcpSecureSourceManagerInstanceName(project, location, instanceID), "create", false))
	return true
}

func handleGCPSecureSourceManagerDeleteInstance(w http.ResponseWriter, path string) bool {
	project, location, instanceID, ok := parseGCPSecureSourceManagerInstancePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "deleteInstance."+instanceID, gcpSecureSourceManagerInstanceName(project, location, instanceID), "delete", false))
	return true
}

func handleGCPSecureSourceManagerListRepositories(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || !isGCPSecureSourceManagerRepositoriesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerRepository(project, location, "repository-1"),
		gcpSecureSourceManagerRepository(project, location, "repository-2"),
	}
	return respondGCPSecureSourceManagerList(w, "repositories", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerGetRepository(w http.ResponseWriter, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerRepositoryPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerRepository(project, location, repoID))
	return true
}

func handleGCPSecureSourceManagerCreateRepository(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || !isGCPSecureSourceManagerRepositoriesCollectionTail(tail) {
		return false
	}
	repoID := strings.TrimSpace(r.URL.Query().Get("repositoryId"))
	if repoID == "" || !isGCPSecureSourceManagerID(repoID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "repositoryId is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	repo := gcpSecureSourceManagerBodyMap(body, "repository")
	if len(repo) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "repository is required")
		return true
	}
	if desc := strings.TrimSpace(gcpSecureSourceManagerString(repo, "description")); desc == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "repository.description is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(repo, "name")); name != "" {
		expected := gcpSecureSourceManagerRepositoryName(project, location, repoID)
		if name != expected {
			respondGCPSecureSourceManagerInvalidArgument(w, path, "repository.name must match parent and repositoryId")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createRepository."+repoID, gcpSecureSourceManagerRepositoryName(project, location, repoID), "create", false))
	return true
}

func handleGCPSecureSourceManagerUpdateRepository(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerRepositoryPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	repo := gcpSecureSourceManagerBodyMap(body, "repository")
	if len(repo) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "repository is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(repo, "name")); name != gcpSecureSourceManagerRepositoryName(project, location, repoID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "repository.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "updateRepository."+repoID, gcpSecureSourceManagerRepositoryName(project, location, repoID), "update", false))
	return true
}

func handleGCPSecureSourceManagerDeleteRepository(w http.ResponseWriter, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerRepositoryPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "deleteRepository."+repoID, gcpSecureSourceManagerRepositoryName(project, location, repoID), "delete", false))
	return true
}

func handleGCPSecureSourceManagerListHooks(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerHooksCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerHook(project, location, repoID, "hook-1"),
		gcpSecureSourceManagerHook(project, location, repoID, "hook-2"),
	}
	return respondGCPSecureSourceManagerList(w, "hooks", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerGetHook(w http.ResponseWriter, path string) bool {
	project, location, repoID, hookID, ok := parseGCPSecureSourceManagerHookPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerHook(project, location, repoID, hookID))
	return true
}

func handleGCPSecureSourceManagerCreateHook(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerHooksCollectionPath(path)
	if !ok {
		return false
	}
	hookID := strings.TrimSpace(r.URL.Query().Get("hookId"))
	if hookID == "" || !isGCPSecureSourceManagerID(hookID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "hookId is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	hook := gcpSecureSourceManagerBodyMap(body, "hook")
	if len(hook) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "hook is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(hook, "name")); name != "" {
		expected := gcpSecureSourceManagerHookName(project, location, repoID, hookID)
		if name != expected {
			respondGCPSecureSourceManagerInvalidArgument(w, path, "hook.name must match parent and hookId")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createHook."+hookID, gcpSecureSourceManagerHookName(project, location, repoID, hookID), "create", false))
	return true
}

func handleGCPSecureSourceManagerUpdateHook(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, hookID, ok := parseGCPSecureSourceManagerHookPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	hook := gcpSecureSourceManagerBodyMap(body, "hook")
	if len(hook) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "hook is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(hook, "name")); name != gcpSecureSourceManagerHookName(project, location, repoID, hookID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "hook.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "updateHook."+hookID, gcpSecureSourceManagerHookName(project, location, repoID, hookID), "update", false))
	return true
}

func handleGCPSecureSourceManagerDeleteHook(w http.ResponseWriter, path string) bool {
	project, location, repoID, hookID, ok := parseGCPSecureSourceManagerHookPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "deleteHook."+hookID, gcpSecureSourceManagerHookName(project, location, repoID, hookID), "delete", false))
	return true
}

func handleGCPSecureSourceManagerGetIamPolicyRepo(w http.ResponseWriter, path string) bool {
	project, location, repoID, action, ok := parseGCPSecureSourceManagerRepositoryIAMPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerIAMPolicy(repoID, nil, gcpSecureSourceManagerRepositoryName(project, location, repoID)))
	return true
}

func handleGCPSecureSourceManagerSetIamPolicyRepo(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, action, ok := parseGCPSecureSourceManagerRepositoryIAMPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpSecureSourceManagerBodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerIAMPolicy(repoID, policy, gcpSecureSourceManagerRepositoryName(project, location, repoID)))
	return true
}

func handleGCPSecureSourceManagerTestIamPermissionsRepo(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, action, ok := parseGCPSecureSourceManagerRepositoryIAMPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissionsRaw, _ := body["permissions"].([]any)
	if len(permissionsRaw) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "permissions is required")
		return true
	}
	permissions := make([]string, 0, len(permissionsRaw))
	for _, p := range permissionsRaw {
		if text, ok := p.(string); ok {
			text = strings.TrimSpace(text)
			if text != "" {
				permissions = append(permissions, text)
			}
		}
	}
	if len(permissions) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "permissions is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{"permissions": permissions})
	return true
}

func handleGCPSecureSourceManagerCreateBranchRule(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerBranchRulesCollectionPath(path)
	if !ok {
		return false
	}
	branchRuleID := strings.TrimSpace(r.URL.Query().Get("branchRuleId"))
	if branchRuleID == "" || !isGCPSecureSourceManagerID(branchRuleID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "branchRuleId is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	branchRule := gcpSecureSourceManagerBodyMap(body, "branchRule")
	if len(branchRule) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "branchRule is required")
		return true
	}
	if pattern := strings.TrimSpace(gcpSecureSourceManagerString(branchRule, "includePattern")); pattern == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "branchRule.includePattern is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(branchRule, "name")); name != "" {
		expected := gcpSecureSourceManagerBranchRuleName(project, location, repoID, branchRuleID)
		if name != expected {
			respondGCPSecureSourceManagerInvalidArgument(w, path, "branchRule.name must match parent and branchRuleId")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createBranchRule."+branchRuleID, gcpSecureSourceManagerBranchRuleName(project, location, repoID, branchRuleID), "create", false))
	return true
}

func handleGCPSecureSourceManagerListBranchRules(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerBranchRulesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerBranchRule(project, location, repoID, "main"),
		gcpSecureSourceManagerBranchRule(project, location, repoID, "release"),
	}
	return respondGCPSecureSourceManagerList(w, "branchRules", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerGetBranchRule(w http.ResponseWriter, path string) bool {
	project, location, repoID, branchRuleID, ok := parseGCPSecureSourceManagerBranchRulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerBranchRule(project, location, repoID, branchRuleID))
	return true
}

func handleGCPSecureSourceManagerUpdateBranchRule(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, branchRuleID, ok := parseGCPSecureSourceManagerBranchRulePath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	branchRule := gcpSecureSourceManagerBodyMap(body, "branchRule")
	if len(branchRule) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "branchRule is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(branchRule, "name")); name != gcpSecureSourceManagerBranchRuleName(project, location, repoID, branchRuleID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "branchRule.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "updateBranchRule."+branchRuleID, gcpSecureSourceManagerBranchRuleName(project, location, repoID, branchRuleID), "update", false))
	return true
}

func handleGCPSecureSourceManagerDeleteBranchRule(w http.ResponseWriter, path string) bool {
	project, location, repoID, branchRuleID, ok := parseGCPSecureSourceManagerBranchRulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "deleteBranchRule."+branchRuleID, gcpSecureSourceManagerBranchRuleName(project, location, repoID, branchRuleID), "delete", false))
	return true
}

func handleGCPSecureSourceManagerCreatePullRequest(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerPullRequestsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	pr := gcpSecureSourceManagerBodyMap(body, "pullRequest")
	if len(pr) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequest is required")
		return true
	}
	prID := strings.TrimSpace(gcpSecureSourceManagerString(pr, "pullRequestId"))
	if prID == "" {
		prID = "pull-request-1"
	}
	if !isGCPSecureSourceManagerID(prID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequest.pullRequestId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createPullRequest."+prID, gcpSecureSourceManagerPullRequestName(project, location, repoID, prID), "create", false))
	return true
}

func handleGCPSecureSourceManagerGetPullRequest(w http.ResponseWriter, path string) bool {
	project, location, repoID, prID, ok := parseGCPSecureSourceManagerPullRequestPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerPullRequest(project, location, repoID, prID, gcpSecureSourceManagerPullRequestState(prID)))
	return true
}

func handleGCPSecureSourceManagerListPullRequests(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerPullRequestsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerPullRequest(project, location, repoID, "pull-request-1", "OPEN"),
		gcpSecureSourceManagerPullRequest(project, location, repoID, "pull-request-closed", "CLOSED"),
		gcpSecureSourceManagerPullRequest(project, location, repoID, "pull-request-merged", "MERGED"),
	}
	return respondGCPSecureSourceManagerList(w, "pullRequests", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerUpdatePullRequest(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, ok := parseGCPSecureSourceManagerPullRequestPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	pr := gcpSecureSourceManagerBodyMap(body, "pullRequest")
	if len(pr) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequest is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(pr, "name")); name != gcpSecureSourceManagerPullRequestName(project, location, repoID, prID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequest.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "updatePullRequest."+prID, gcpSecureSourceManagerPullRequestName(project, location, repoID, prID), "update", false))
	return true
}

func handleGCPSecureSourceManagerPullRequestAction(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, action, ok := parseGCPSecureSourceManagerPullRequestActionPath(path)
	if !ok {
		return false
	}
	if _, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	state := gcpSecureSourceManagerPullRequestState(prID)
	switch action {
	case "merge":
		if state != "OPEN" {
			respondGCPSecureSourceManagerFailedPrecondition(w, path, "pull request must be OPEN to merge")
			return true
		}
	case "open":
		if state == "OPEN" {
			respondGCPSecureSourceManagerFailedPrecondition(w, path, "pull request is already open")
			return true
		}
	case "close":
		if state != "OPEN" {
			respondGCPSecureSourceManagerFailedPrecondition(w, path, "pull request must be OPEN to close")
			return true
		}
	default:
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, action+"PullRequest."+prID, gcpSecureSourceManagerPullRequestName(project, location, repoID, prID), action, false))
	return true
}

func handleGCPSecureSourceManagerListPullRequestFileDiffs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, ok := parseGCPSecureSourceManagerPullRequestFileDiffsPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerFileDiff(project, location, repoID, prID, "README.md"),
		gcpSecureSourceManagerFileDiff(project, location, repoID, prID, "go.mod"),
	}
	return respondGCPSecureSourceManagerList(w, "fileDiffs", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerFetchTree(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerFetchTreePath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerTreeEntry("TREE", "c0ffee111", "/", "040000", 0),
		gcpSecureSourceManagerTreeEntry("BLOB", "abc123def", "README.md", "100644", 128),
		gcpSecureSourceManagerTreeEntry("BLOB", "deadbeef1", "go.mod", "100644", 96),
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("recursive")), "false") {
		items = items[:2]
	}
	if ref := strings.TrimSpace(r.URL.Query().Get("ref")); ref != "" {
		for _, item := range items {
			item["ref"] = ref
		}
	}
	if !respondGCPSecureSourceManagerList(w, "treeEntries", items, pageSize, start, path) {
		return true
	}
	_ = project
	_ = location
	_ = repoID
	return true
}

func handleGCPSecureSourceManagerFetchBlob(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, ok := parseGCPSecureSourceManagerFetchBlobPath(path)
	if !ok {
		return false
	}
	sha := strings.TrimSpace(r.URL.Query().Get("sha"))
	if sha == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "sha is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"sha":     sha,
		"content": "c3RhY2t5YXJkLXNlY3VyZS1zb3VyY2UtbWFuYWdlcg==",
	})
	return true
}

func handleGCPSecureSourceManagerCreateIssue(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerIssuesCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	issue := gcpSecureSourceManagerBodyMap(body, "issue")
	if len(issue) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issue is required")
		return true
	}
	issueID := strings.TrimSpace(gcpSecureSourceManagerString(issue, "issueId"))
	if issueID == "" {
		issueID = "issue-1"
	}
	if !isGCPSecureSourceManagerID(issueID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issue.issueId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createIssue."+issueID, gcpSecureSourceManagerIssueName(project, location, repoID, issueID), "create", false))
	return true
}

func handleGCPSecureSourceManagerGetIssue(w http.ResponseWriter, path string) bool {
	project, location, repoID, issueID, ok := parseGCPSecureSourceManagerIssuePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerIssue(project, location, repoID, issueID, gcpSecureSourceManagerIssueState(issueID)))
	return true
}

func handleGCPSecureSourceManagerListIssues(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, ok := parseGCPSecureSourceManagerIssuesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerIssue(project, location, repoID, "issue-1", "OPEN"),
		gcpSecureSourceManagerIssue(project, location, repoID, "issue-closed", "CLOSED"),
	}
	return respondGCPSecureSourceManagerList(w, "issues", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerUpdateIssue(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, issueID, ok := parseGCPSecureSourceManagerIssuePath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	issue := gcpSecureSourceManagerBodyMap(body, "issue")
	if len(issue) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issue is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(issue, "name")); name != gcpSecureSourceManagerIssueName(project, location, repoID, issueID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issue.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "updateIssue."+issueID, gcpSecureSourceManagerIssueName(project, location, repoID, issueID), "update", false))
	return true
}

func handleGCPSecureSourceManagerDeleteIssue(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, issueID, ok := parseGCPSecureSourceManagerIssuePath(path)
	if !ok {
		return false
	}
	if etag := strings.TrimSpace(r.URL.Query().Get("etag")); etag != "" && etag != gcpSecureSourceManagerETag(issueID) {
		respondGCPSecureSourceManagerFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "deleteIssue."+issueID, gcpSecureSourceManagerIssueName(project, location, repoID, issueID), "delete", false))
	return true
}

func handleGCPSecureSourceManagerIssueAction(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, issueID, action, ok := parseGCPSecureSourceManagerIssueActionPath(path)
	if !ok {
		return false
	}
	if _, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path); !valid {
		return true
	}
	state := gcpSecureSourceManagerIssueState(issueID)
	switch action {
	case "open":
		if state == "OPEN" {
			respondGCPSecureSourceManagerFailedPrecondition(w, path, "issue is already open")
			return true
		}
	case "close":
		if state != "OPEN" {
			respondGCPSecureSourceManagerFailedPrecondition(w, path, "issue must be OPEN to close")
			return true
		}
	default:
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, action+"Issue."+issueID, gcpSecureSourceManagerIssueName(project, location, repoID, issueID), action, false))
	return true
}

func handleGCPSecureSourceManagerGetPullRequestComment(w http.ResponseWriter, path string) bool {
	project, location, repoID, prID, commentID, ok := parseGCPSecureSourceManagerPullRequestCommentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerPullRequestComment(project, location, repoID, prID, commentID, strings.Contains(commentID, "resolved")))
	return true
}

func handleGCPSecureSourceManagerListPullRequestComments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, ok := parseGCPSecureSourceManagerPullRequestCommentsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerPullRequestComment(project, location, repoID, prID, "comment-1", false),
		gcpSecureSourceManagerPullRequestComment(project, location, repoID, prID, "comment-resolved", true),
	}
	return respondGCPSecureSourceManagerList(w, "pullRequestComments", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerCreatePullRequestComment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, ok := parseGCPSecureSourceManagerPullRequestCommentsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	comment := gcpSecureSourceManagerBodyMap(body, "pullRequestComment")
	if len(comment) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequestComment is required")
		return true
	}
	commentID := strings.TrimSpace(gcpSecureSourceManagerString(comment, "commentId"))
	if commentID == "" {
		commentID = "comment-1"
	}
	if !isGCPSecureSourceManagerID(commentID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequestComment.commentId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createPullRequestComment."+commentID, gcpSecureSourceManagerPullRequestCommentName(project, location, repoID, prID, commentID), "create", false))
	return true
}

func handleGCPSecureSourceManagerUpdatePullRequestComment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, commentID, ok := parseGCPSecureSourceManagerPullRequestCommentPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	comment := gcpSecureSourceManagerBodyMap(body, "pullRequestComment")
	if len(comment) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequestComment is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(comment, "name")); name != gcpSecureSourceManagerPullRequestCommentName(project, location, repoID, prID, commentID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pullRequestComment.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "updatePullRequestComment."+commentID, gcpSecureSourceManagerPullRequestCommentName(project, location, repoID, prID, commentID), "update", false))
	return true
}

func handleGCPSecureSourceManagerDeletePullRequestComment(w http.ResponseWriter, path string) bool {
	project, location, repoID, prID, commentID, ok := parseGCPSecureSourceManagerPullRequestCommentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "deletePullRequestComment."+commentID, gcpSecureSourceManagerPullRequestCommentName(project, location, repoID, prID, commentID), "delete", false))
	return true
}

func handleGCPSecureSourceManagerBatchCreatePullRequestComments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, action, ok := parseGCPSecureSourceManagerPullRequestCommentsBatchActionPath(path)
	if !ok || action != "batchCreate" {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	requests, _ := body["requests"].([]any)
	if len(requests) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "requests is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "batchCreatePullRequestComments."+prID, gcpSecureSourceManagerPullRequestName(project, location, repoID, prID), "batchCreate", false))
	return true
}

func handleGCPSecureSourceManagerResolvePullRequestComments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, action, ok := parseGCPSecureSourceManagerPullRequestCommentsBatchActionPath(path)
	if !ok || action != "resolve" {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	names, _ := body["names"].([]any)
	if len(names) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "names is required")
		return true
	}
	for _, name := range names {
		if text, _ := name.(string); strings.Contains(strings.ToLower(text), "resolved") {
			respondGCPSecureSourceManagerFailedPrecondition(w, path, "comment already resolved")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "resolvePullRequestComments."+prID, gcpSecureSourceManagerPullRequestName(project, location, repoID, prID), "resolve", false))
	return true
}

func handleGCPSecureSourceManagerUnresolvePullRequestComments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, prID, action, ok := parseGCPSecureSourceManagerPullRequestCommentsBatchActionPath(path)
	if !ok || action != "unresolve" {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	names, _ := body["names"].([]any)
	if len(names) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "names is required")
		return true
	}
	for _, name := range names {
		if text, _ := name.(string); !strings.Contains(strings.ToLower(text), "resolved") {
			respondGCPSecureSourceManagerFailedPrecondition(w, path, "comment is not resolved")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "unresolvePullRequestComments."+prID, gcpSecureSourceManagerPullRequestName(project, location, repoID, prID), "unresolve", false))
	return true
}

func handleGCPSecureSourceManagerCreateIssueComment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, issueID, ok := parseGCPSecureSourceManagerIssueCommentsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	comment := gcpSecureSourceManagerBodyMap(body, "issueComment")
	if len(comment) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issueComment is required")
		return true
	}
	commentID := strings.TrimSpace(gcpSecureSourceManagerString(comment, "commentId"))
	if commentID == "" {
		commentID = "comment-1"
	}
	if !isGCPSecureSourceManagerID(commentID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issueComment.commentId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "createIssueComment."+commentID, gcpSecureSourceManagerIssueCommentName(project, location, repoID, issueID, commentID), "create", false))
	return true
}

func handleGCPSecureSourceManagerGetIssueComment(w http.ResponseWriter, path string) bool {
	project, location, repoID, issueID, commentID, ok := parseGCPSecureSourceManagerIssueCommentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerIssueComment(project, location, repoID, issueID, commentID))
	return true
}

func handleGCPSecureSourceManagerListIssueComments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, issueID, ok := parseGCPSecureSourceManagerIssueCommentsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerIssueComment(project, location, repoID, issueID, "comment-1"),
		gcpSecureSourceManagerIssueComment(project, location, repoID, issueID, "comment-2"),
	}
	return respondGCPSecureSourceManagerList(w, "issueComments", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerUpdateIssueComment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repoID, issueID, commentID, ok := parseGCPSecureSourceManagerIssueCommentPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecureSourceManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	comment := gcpSecureSourceManagerBodyMap(body, "issueComment")
	if len(comment) == 0 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issueComment is required")
		return true
	}
	if name := strings.TrimSpace(gcpSecureSourceManagerString(comment, "name")); name != gcpSecureSourceManagerIssueCommentName(project, location, repoID, issueID, commentID) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "issueComment.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "updateIssueComment."+commentID, gcpSecureSourceManagerIssueCommentName(project, location, repoID, issueID, commentID), "update", false))
	return true
}

func handleGCPSecureSourceManagerDeleteIssueComment(w http.ResponseWriter, path string) bool {
	project, location, repoID, issueID, commentID, ok := parseGCPSecureSourceManagerIssueCommentPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, "deleteIssueComment."+commentID, gcpSecureSourceManagerIssueCommentName(project, location, repoID, issueID, commentID), "delete", false))
	return true
}

func handleGCPSecureSourceManagerGetOperation(w http.ResponseWriter, path string) bool {
	project, location, opID, ok := parseGCPSecureSourceManagerOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecureSourceManagerOperation(project, location, opID, fmt.Sprintf("projects/%s/locations/%s", project, location), "poll", true))
	return true
}

func handleGCPSecureSourceManagerListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPSecureSourceManagerOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecureSourceManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecureSourceManagerOperation(project, location, "sample-operation", fmt.Sprintf("projects/%s/locations/%s", project, location), "list", true),
	}
	return respondGCPSecureSourceManagerList(w, "operations", items, pageSize, start, path)
}

func handleGCPSecureSourceManagerDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPSecureSourceManagerOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPSecureSourceManagerLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	for _, part := range parts[6:] {
		if strings.TrimSpace(part) == "" {
			return "", "", nil, false
		}
	}
	return project, location, parts[6:], true
}

func parseGCPSecureSourceManagerInstancePath(path string) (project, location, instanceID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "instances" {
		return "", "", "", false
	}
	instanceID = strings.TrimSpace(tail[1])
	if !isGCPSecureSourceManagerID(instanceID) {
		return "", "", "", false
	}
	return project, location, instanceID, true
}

func parseGCPSecureSourceManagerRepositoryPath(path string) (project, location, repoID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "repositories" {
		return "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	if !isGCPSecureSourceManagerID(repoID) {
		return "", "", "", false
	}
	return project, location, repoID, true
}

func parseGCPSecureSourceManagerHooksCollectionPath(path string) (project, location, repoID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "repositories" || tail[2] != "hooks" {
		return "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	if !isGCPSecureSourceManagerID(repoID) {
		return "", "", "", false
	}
	return project, location, repoID, true
}

func parseGCPSecureSourceManagerHookPath(path string) (project, location, repoID, hookID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "repositories" || tail[2] != "hooks" {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	hookID = strings.TrimSpace(tail[3])
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(hookID) {
		return "", "", "", "", false
	}
	return project, location, repoID, hookID, true
}

func parseGCPSecureSourceManagerRepositoryIAMPath(path string) (project, location, repoID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "repositories" {
		return "", "", "", "", false
	}
	repoID, action, hasAction := gcpSecureSourceManagerResourceActionSegment(tail[1])
	if !hasAction {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(repoID)
	action = strings.TrimSpace(action)
	if !isGCPSecureSourceManagerID(repoID) {
		return "", "", "", "", false
	}
	switch action {
	case "getIamPolicy", "setIamPolicy", "testIamPermissions", "fetchTree", "fetchBlob":
		return project, location, repoID, action, true
	default:
		return "", "", "", "", false
	}
}

func parseGCPSecureSourceManagerBranchRulesCollectionPath(path string) (project, location, repoID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "repositories" || tail[2] != "branchRules" {
		return "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	if !isGCPSecureSourceManagerID(repoID) {
		return "", "", "", false
	}
	return project, location, repoID, true
}

func parseGCPSecureSourceManagerBranchRulePath(path string) (project, location, repoID, branchRuleID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "repositories" || tail[2] != "branchRules" {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	branchRuleID = strings.TrimSpace(tail[3])
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(branchRuleID) {
		return "", "", "", "", false
	}
	return project, location, repoID, branchRuleID, true
}

func parseGCPSecureSourceManagerPullRequestsCollectionPath(path string) (project, location, repoID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "repositories" || tail[2] != "pullRequests" {
		return "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	if !isGCPSecureSourceManagerID(repoID) {
		return "", "", "", false
	}
	return project, location, repoID, true
}

func parseGCPSecureSourceManagerPullRequestPath(path string) (project, location, repoID, prID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "repositories" || tail[2] != "pullRequests" {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	prID, action, hasAction := gcpSecureSourceManagerResourceActionSegment(tail[3])
	if hasAction || action != "" {
		return "", "", "", "", false
	}
	prID = strings.TrimSpace(prID)
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(prID) {
		return "", "", "", "", false
	}
	return project, location, repoID, prID, true
}

func parseGCPSecureSourceManagerPullRequestActionPath(path string) (project, location, repoID, prID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "repositories" || tail[2] != "pullRequests" {
		return "", "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	prID, action, hasAction := gcpSecureSourceManagerResourceActionSegment(tail[3])
	if !hasAction {
		return "", "", "", "", "", false
	}
	prID = strings.TrimSpace(prID)
	action = strings.TrimSpace(action)
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(prID) {
		return "", "", "", "", "", false
	}
	if action != "merge" && action != "open" && action != "close" {
		return "", "", "", "", "", false
	}
	return project, location, repoID, prID, action, true
}

func parseGCPSecureSourceManagerPullRequestFileDiffsPath(path string) (project, location, repoID, prID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "repositories" || tail[2] != "pullRequests" {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	prID, action, hasAction := gcpSecureSourceManagerResourceActionSegment(tail[3])
	if !hasAction || action != "listFileDiffs" {
		return "", "", "", "", false
	}
	prID = strings.TrimSpace(prID)
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(prID) {
		return "", "", "", "", false
	}
	return project, location, repoID, prID, true
}

func parseGCPSecureSourceManagerFetchTreePath(path string) (project, location, repoID string, ok bool) {
	project, location, repoID, action, ok := parseGCPSecureSourceManagerRepositoryIAMPath(path)
	if !ok || action != "fetchTree" {
		return "", "", "", false
	}
	return project, location, repoID, true
}

func parseGCPSecureSourceManagerFetchBlobPath(path string) (project, location, repoID string, ok bool) {
	project, location, repoID, action, ok := parseGCPSecureSourceManagerRepositoryIAMPath(path)
	if !ok || action != "fetchBlob" {
		return "", "", "", false
	}
	return project, location, repoID, true
}

func parseGCPSecureSourceManagerIssuesCollectionPath(path string) (project, location, repoID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "repositories" || tail[2] != "issues" {
		return "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	if !isGCPSecureSourceManagerID(repoID) {
		return "", "", "", false
	}
	return project, location, repoID, true
}

func parseGCPSecureSourceManagerIssuePath(path string) (project, location, repoID, issueID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "repositories" || tail[2] != "issues" {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	issueID, action, hasAction := gcpSecureSourceManagerResourceActionSegment(tail[3])
	if hasAction || action != "" {
		return "", "", "", "", false
	}
	issueID = strings.TrimSpace(issueID)
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(issueID) {
		return "", "", "", "", false
	}
	return project, location, repoID, issueID, true
}

func parseGCPSecureSourceManagerIssueActionPath(path string) (project, location, repoID, issueID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "repositories" || tail[2] != "issues" {
		return "", "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	issueID, action, hasAction := gcpSecureSourceManagerResourceActionSegment(tail[3])
	if !hasAction {
		return "", "", "", "", "", false
	}
	issueID = strings.TrimSpace(issueID)
	action = strings.TrimSpace(action)
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(issueID) {
		return "", "", "", "", "", false
	}
	if action != "open" && action != "close" {
		return "", "", "", "", "", false
	}
	return project, location, repoID, issueID, action, true
}

func parseGCPSecureSourceManagerPullRequestCommentsCollectionPath(path string) (project, location, repoID, prID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "repositories" || tail[2] != "pullRequests" || tail[4] != "pullRequestComments" {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	prID = strings.TrimSpace(tail[3])
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(prID) {
		return "", "", "", "", false
	}
	return project, location, repoID, prID, true
}

func parseGCPSecureSourceManagerPullRequestCommentsBatchActionPath(path string) (project, location, repoID, prID, action string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "repositories" || tail[2] != "pullRequests" {
		return "", "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	prID = strings.TrimSpace(tail[3])
	collection, action, hasAction := gcpSecureSourceManagerResourceActionSegment(tail[4])
	if !hasAction || collection != "pullRequestComments" {
		return "", "", "", "", "", false
	}
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(prID) {
		return "", "", "", "", "", false
	}
	switch action {
	case "batchCreate", "resolve", "unresolve":
		return project, location, repoID, prID, action, true
	default:
		return "", "", "", "", "", false
	}
}

func parseGCPSecureSourceManagerPullRequestCommentPath(path string) (project, location, repoID, prID, commentID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 6 || tail[0] != "repositories" || tail[2] != "pullRequests" || tail[4] != "pullRequestComments" {
		return "", "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	prID = strings.TrimSpace(tail[3])
	commentID = strings.TrimSpace(tail[5])
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(prID) || !isGCPSecureSourceManagerID(commentID) {
		return "", "", "", "", "", false
	}
	return project, location, repoID, prID, commentID, true
}

func parseGCPSecureSourceManagerIssueCommentsCollectionPath(path string) (project, location, repoID, issueID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "repositories" || tail[2] != "issues" || tail[4] != "issueComments" {
		return "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	issueID = strings.TrimSpace(tail[3])
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(issueID) {
		return "", "", "", "", false
	}
	return project, location, repoID, issueID, true
}

func parseGCPSecureSourceManagerIssueCommentPath(path string) (project, location, repoID, issueID, commentID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 6 || tail[0] != "repositories" || tail[2] != "issues" || tail[4] != "issueComments" {
		return "", "", "", "", "", false
	}
	repoID = strings.TrimSpace(tail[1])
	issueID = strings.TrimSpace(tail[3])
	commentID = strings.TrimSpace(tail[5])
	if !isGCPSecureSourceManagerID(repoID) || !isGCPSecureSourceManagerID(issueID) || !isGCPSecureSourceManagerID(commentID) {
		return "", "", "", "", "", false
	}
	return project, location, repoID, issueID, commentID, true
}

func parseGCPSecureSourceManagerOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", false
	}
	operationID = strings.TrimSpace(tail[1])
	if operationID == "" {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPSecureSourceManagerOperationsCollectionPath(path string) (project, location string, ok bool) {
	project, location, tail, ok := parseGCPSecureSourceManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return "", "", false
	}
	return project, location, true
}

func isGCPSecureSourceManagerInstancesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "instances"
}

func isGCPSecureSourceManagerRepositoriesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "repositories"
}

func gcpSecureSourceManagerResourceActionSegment(segment string) (resource, action string, hasAction bool) {
	segment = strings.TrimSpace(segment)
	resource, action, hasAction = strings.Cut(segment, ":")
	if !hasAction {
		return segment, "", false
	}
	return strings.TrimSpace(resource), strings.TrimSpace(action), true
}

func parseGCPSecureSourceManagerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPSecureSourceManagerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPSecureSourceManagerList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	out := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, item)
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             out,
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPSecureSourceManagerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPSecureSourceManagerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpSecureSourceManagerBodyMap(body map[string]any, key string) map[string]any {
	if nested, ok := body[key].(map[string]any); ok && len(nested) > 0 {
		return nested
	}
	return body
}

func gcpSecureSourceManagerString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	raw, ok := body[key]
	if !ok {
		return ""
	}
	text, _ := raw.(string)
	return strings.TrimSpace(text)
}

func isGCPSecureSourceManagerID(id string) bool {
	id = strings.TrimSpace(id)
	if len(id) == 0 || len(id) > 63 {
		return false
	}
	for i := 0; i < len(id); i++ {
		ch := id[i]
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '-':
		default:
			return false
		}
		if i == 0 && !(ch >= 'a' && ch <= 'z') {
			return false
		}
		if i == len(id)-1 && ch == '-' {
			return false
		}
	}
	return true
}

func gcpSecureSourceManagerLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(location),
		"metadata": map[string]any{
			"service": "securesourcemanager.googleapis.com",
		},
	}
}

func gcpSecureSourceManagerInstance(project, location, instanceID string) map[string]any {
	return map[string]any{
		"name":        gcpSecureSourceManagerInstanceName(project, location, instanceID),
		"uid":         "ssm-instance-" + instanceID,
		"displayName": "Stackyard Instance " + instanceID,
		"createTime":  gcpSecureSourceManagerReferenceTime.Format(time.RFC3339),
		"updateTime":  gcpSecureSourceManagerReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		"state":       "ACTIVE",
		"hostConfig": map[string]any{
			"api":  fmt.Sprintf("https://%s-%s-api.example.com", instanceID, location),
			"html": fmt.Sprintf("https://%s-%s.example.com", instanceID, location),
		},
		"privateConfig": map[string]any{
			"caPool": fmt.Sprintf("projects/%s/locations/%s/caPools/default", project, location),
		},
	}
}

func gcpSecureSourceManagerRepository(project, location, repoID string) map[string]any {
	name := gcpSecureSourceManagerRepositoryName(project, location, repoID)
	return map[string]any{
		"name":        name,
		"uid":         "ssm-repo-" + repoID,
		"description": "Stackyard repository " + repoID,
		"createTime":  gcpSecureSourceManagerReferenceTime.Format(time.RFC3339),
		"updateTime":  gcpSecureSourceManagerReferenceTime.Add(90 * time.Minute).Format(time.RFC3339),
		"etag":        gcpSecureSourceManagerETag(repoID),
		"instance":    fmt.Sprintf("projects/%s/locations/%s/instances/instance-1", project, location),
		"uris": map[string]any{
			"https": fmt.Sprintf("https://source.developers.google.com/p/%s/r/%s", project, repoID),
			"ssh":   fmt.Sprintf("ssh://source.developers.google.com/p/%s/r/%s", project, repoID),
		},
	}
}

func gcpSecureSourceManagerHook(project, location, repoID, hookID string) map[string]any {
	return map[string]any{
		"name":       gcpSecureSourceManagerHookName(project, location, repoID, hookID),
		"uid":        "ssm-hook-" + hookID,
		"createTime": gcpSecureSourceManagerReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		"updateTime": gcpSecureSourceManagerReferenceTime.Add(95 * time.Minute).Format(time.RFC3339),
		"disabled":   false,
		"events":     []any{"PUSH", "PULL_REQUEST"},
		"webhookConfig": map[string]any{
			"url": "https://example.com/hooks/stackyard",
		},
	}
}

func gcpSecureSourceManagerBranchRule(project, location, repoID, branchRuleID string) map[string]any {
	return map[string]any{
		"name":                gcpSecureSourceManagerBranchRuleName(project, location, repoID, branchRuleID),
		"uid":                 "ssm-branch-rule-" + branchRuleID,
		"includePattern":      branchRuleID,
		"disabled":            false,
		"createTime":          gcpSecureSourceManagerReferenceTime.Add(10 * time.Minute).Format(time.RFC3339),
		"updateTime":          gcpSecureSourceManagerReferenceTime.Add(110 * time.Minute).Format(time.RFC3339),
		"minimumReviewsCount": 1,
	}
}

func gcpSecureSourceManagerPullRequest(project, location, repoID, prID, state string) map[string]any {
	return map[string]any{
		"name":        gcpSecureSourceManagerPullRequestName(project, location, repoID, prID),
		"uid":         "ssm-pr-" + prID,
		"title":       "Stackyard pull request " + prID,
		"description": "Synthetic pull request for contract validation",
		"state":       state,
		"createTime":  gcpSecureSourceManagerReferenceTime.Add(15 * time.Minute).Format(time.RFC3339),
		"updateTime":  gcpSecureSourceManagerReferenceTime.Add(140 * time.Minute).Format(time.RFC3339),
		"author": map[string]any{
			"name":  "users/stackyard",
			"email": "stackyard@example.com",
		},
		"base": map[string]any{"ref": "refs/heads/main", "sha": "c0ffee111"},
		"head": map[string]any{"ref": "refs/heads/feature", "sha": "deadbeef1"},
		"etag": gcpSecureSourceManagerETag(prID),
	}
}

func gcpSecureSourceManagerFileDiff(project, location, repoID, prID, filePath string) map[string]any {
	_ = project
	_ = location
	_ = repoID
	_ = prID
	return map[string]any{
		"path":       filePath,
		"oldPath":    filePath,
		"changeType": "MODIFIED",
		"additions":  3,
		"deletions":  1,
	}
}

func gcpSecureSourceManagerTreeEntry(objectType, sha, filePath, mode string, size int) map[string]any {
	return map[string]any{
		"type": objectType,
		"sha":  sha,
		"path": filePath,
		"mode": mode,
		"size": size,
	}
}

func gcpSecureSourceManagerIssue(project, location, repoID, issueID, state string) map[string]any {
	return map[string]any{
		"name":        gcpSecureSourceManagerIssueName(project, location, repoID, issueID),
		"uid":         "ssm-issue-" + issueID,
		"title":       "Stackyard issue " + issueID,
		"description": "Synthetic issue for contract validation",
		"state":       state,
		"createTime":  gcpSecureSourceManagerReferenceTime.Add(20 * time.Minute).Format(time.RFC3339),
		"updateTime":  gcpSecureSourceManagerReferenceTime.Add(150 * time.Minute).Format(time.RFC3339),
		"author": map[string]any{
			"name":  "users/stackyard",
			"email": "stackyard@example.com",
		},
		"etag": gcpSecureSourceManagerETag(issueID),
	}
}

func gcpSecureSourceManagerPullRequestComment(project, location, repoID, prID, commentID string, resolved bool) map[string]any {
	state := "OPEN"
	if resolved {
		state = "RESOLVED"
	}
	return map[string]any{
		"name":           gcpSecureSourceManagerPullRequestCommentName(project, location, repoID, prID, commentID),
		"uid":            "ssm-pr-comment-" + commentID,
		"body":           "Stackyard pull request comment " + commentID,
		"state":          state,
		"createTime":     gcpSecureSourceManagerReferenceTime.Add(25 * time.Minute).Format(time.RFC3339),
		"updateTime":     gcpSecureSourceManagerReferenceTime.Add(155 * time.Minute).Format(time.RFC3339),
		"line":           12,
		"path":           "README.md",
		"resolved":       resolved,
		"author":         map[string]any{"name": "users/reviewer", "email": "reviewer@example.com"},
		"conversationId": "conversation-1",
	}
}

func gcpSecureSourceManagerIssueComment(project, location, repoID, issueID, commentID string) map[string]any {
	return map[string]any{
		"name":       gcpSecureSourceManagerIssueCommentName(project, location, repoID, issueID, commentID),
		"uid":        "ssm-issue-comment-" + commentID,
		"body":       "Stackyard issue comment " + commentID,
		"createTime": gcpSecureSourceManagerReferenceTime.Add(30 * time.Minute).Format(time.RFC3339),
		"updateTime": gcpSecureSourceManagerReferenceTime.Add(160 * time.Minute).Format(time.RFC3339),
		"author":     map[string]any{"name": "users/stackyard", "email": "stackyard@example.com"},
	}
}

func gcpSecureSourceManagerIAMPolicy(resourceID string, in map[string]any, resource string) map[string]any {
	if len(in) == 0 {
		return map[string]any{
			"version": 1,
			"bindings": []any{
				map[string]any{
					"role":    "roles/securesourcemanager.reader",
					"members": []any{"user:stackyard@example.com"},
				},
			},
			"etag":     "policy-etag-" + resourceID,
			"resource": resource,
		}
	}
	if _, ok := in["version"]; !ok {
		in["version"] = 1
	}
	if _, ok := in["etag"]; !ok {
		in["etag"] = "policy-etag-" + resourceID
	}
	in["resource"] = resource
	return in
}

func gcpSecureSourceManagerOperation(project, location, operationID, target, verb string, done bool) map[string]any {
	op := map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.securesourcemanager.v1.OperationMetadata",
			"target":     target,
			"verb":       verb,
			"createTime": gcpSecureSourceManagerReferenceTime.Add(45 * time.Minute).Format(time.RFC3339),
			"apiVersion": "v1",
		},
		"done": done,
	}
	if done {
		op["response"] = map[string]any{"@type": "type.googleapis.com/google.protobuf.Empty"}
	}
	return op
}

func gcpSecureSourceManagerPullRequestState(prID string) string {
	prID = strings.ToLower(strings.TrimSpace(prID))
	switch {
	case strings.Contains(prID, "merged"):
		return "MERGED"
	case strings.Contains(prID, "closed"):
		return "CLOSED"
	default:
		return "OPEN"
	}
}

func gcpSecureSourceManagerIssueState(issueID string) string {
	issueID = strings.ToLower(strings.TrimSpace(issueID))
	if strings.Contains(issueID, "closed") {
		return "CLOSED"
	}
	return "OPEN"
}

func gcpSecureSourceManagerInstanceName(project, location, instanceID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, location, instanceID)
}

func gcpSecureSourceManagerRepositoryName(project, location, repoID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/repositories/%s", project, location, repoID)
}

func gcpSecureSourceManagerHookName(project, location, repoID, hookID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/repositories/%s/hooks/%s", project, location, repoID, hookID)
}

func gcpSecureSourceManagerBranchRuleName(project, location, repoID, branchRuleID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/repositories/%s/branchRules/%s", project, location, repoID, branchRuleID)
}

func gcpSecureSourceManagerPullRequestName(project, location, repoID, prID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/repositories/%s/pullRequests/%s", project, location, repoID, prID)
}

func gcpSecureSourceManagerIssueName(project, location, repoID, issueID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/repositories/%s/issues/%s", project, location, repoID, issueID)
}

func gcpSecureSourceManagerPullRequestCommentName(project, location, repoID, prID, commentID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/repositories/%s/pullRequests/%s/pullRequestComments/%s", project, location, repoID, prID, commentID)
}

func gcpSecureSourceManagerIssueCommentName(project, location, repoID, issueID, commentID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/repositories/%s/issues/%s/issueComments/%s", project, location, repoID, issueID, commentID)
}

func gcpSecureSourceManagerETag(id string) string {
	return "etag-" + strings.TrimSpace(id)
}

func respondGCPSecureSourceManagerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSecureSourceManagerError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSecureSourceManagerFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSecureSourceManagerError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSecureSourceManagerError(w http.ResponseWriter, status int, errCode, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errCode,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_securesourcemanager(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "securesourcemanager") {
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
			"name":     "projects/stackyard/locations/us-central1/repositories/sample",
			"service":  "securesourcemanager",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
