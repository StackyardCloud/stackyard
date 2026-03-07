package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var gcpRecaptchaReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPRecaptchaEnterpriseRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_recaptchaenterprise(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPRecaptchaEnterprisePath(path, hasGCPRecaptchaEnterpriseHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRecaptchaEnterpriseListKeys(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseGetKey(w, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseRetrieveLegacySecretKey(w, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseGetMetrics(w, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseListFirewallPolicies(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseGetFirewallPolicy(w, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseListIpOverrides(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseListRelatedAccountGroups(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseListRelatedAccountGroupMemberships(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseSearchRelatedAccountGroupMemberships(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseGetProjectMetadata(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRecaptchaEnterpriseCreateAssessment(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseAnnotateAssessment(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseCreateKey(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseMigrateKey(w, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseAddIpOverride(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseRemoveIpOverride(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseCreateFirewallPolicy(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseReorderFirewallPolicies(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseSearchRelatedAccountGroupMemberships(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRecaptchaEnterpriseUpdateKey(w, r, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseUpdateFirewallPolicy(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPRecaptchaEnterpriseDeleteKey(w, path) {
			return true
		}
		if handleGCPRecaptchaEnterpriseDeleteFirewallPolicy(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPRecaptchaEnterpriseHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "recaptchaenterprise" || service == "recaptcha-enterprise" || service == "recaptcha_enterprise" {
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-recaptchaenterprise-v2-apiv1") || strings.Contains(ua, "recaptchaenterprise")
}

func isGCPRecaptchaEnterprisePath(path string, includeProjectMetadata bool) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || project == "" || len(tail) == 0 {
		return false
	}
	first := tail[0]
	if first == "assessments" || first == "keys" || first == "firewallpolicies" || first == "firewallpolicies:reorder" {
		return true
	}
	if first == "relatedaccountgroups" || first == "relatedaccountgroupmemberships:search" {
		return true
	}
	if includeProjectMetadata && first == "projectmetadata" {
		return true
	}
	return false
}

func handleGCPRecaptchaEnterpriseCreateAssessment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || !isGCPRecaptchaCollectionTail(tail, "assessments") || r.Method != http.MethodPost {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	assessment := gcpRecaptchaBodyMap(body, "assessment")
	if len(assessment) == 0 {
		respondGCPRecaptchaInvalidArgument(w, path, "assessment is required")
		return true
	}
	event := gcpRecaptchaNestedMap(assessment, "event")
	token := gcpRecaptchaString(event, "token")
	siteKey := gcpRecaptchaString(event, "siteKey")
	if token == "" && siteKey == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "assessment.event.token or assessment.event.siteKey is required")
		return true
	}
	assessmentID := strings.TrimSpace(r.URL.Query().Get("assessmentId"))
	if assessmentID == "" {
		assessmentID = "assessment-1"
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaAssessmentFixture(project, assessmentID, token, siteKey, gcpRecaptchaString(event, "accountId")))
	return true
}

func handleGCPRecaptchaEnterpriseAnnotateAssessment(w http.ResponseWriter, r *http.Request, path string) bool {
	project, assessmentID, ok := parseGCPRecaptchaAssessmentActionPath(path, "annotate")
	if !ok || r.Method != http.MethodPost {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	annotation := strings.TrimSpace(gcpRecaptchaString(body, "annotation"))
	if annotation == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "annotation is required")
		return true
	}
	reasons := gcpRecaptchaStringSlice(body["reasons"])
	if len(reasons) == 0 {
		reasons = []string{"CHARGEBACK"}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":       fmt.Sprintf("projects/%s/assessments/%s", project, assessmentID),
		"annotation": annotation,
		"reasons":    reasons,
		"provider":   providerGCP,
	})
	return true
}

func handleGCPRecaptchaEnterpriseCreateKey(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || !isGCPRecaptchaCollectionTail(tail, "keys") || r.Method != http.MethodPost {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	key := gcpRecaptchaBodyMap(body, "key")
	if len(key) == 0 {
		respondGCPRecaptchaInvalidArgument(w, path, "key is required")
		return true
	}
	displayName := gcpRecaptchaString(key, "displayName")
	if displayName == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "key.displayName is required")
		return true
	}
	keyID := strings.TrimSpace(r.URL.Query().Get("keyId"))
	if keyID == "" {
		keyID = "site-key-1"
	}
	fixture := gcpRecaptchaKeyFixture(project, keyID)
	fixture["displayName"] = displayName
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRecaptchaEnterpriseListKeys(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || !isGCPRecaptchaCollectionTail(tail, "keys") || r.Method != http.MethodGet {
		return false
	}
	pageSize, start, valid := parseGCPRecaptchaPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRecaptchaKeyFixture(project, "site-key-1"),
		gcpRecaptchaKeyFixture(project, "site-key-2"),
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaPaginatedResponse("keys", items, pageSize, start))
	return true
}

func handleGCPRecaptchaEnterpriseGetKey(w http.ResponseWriter, path string) bool {
	project, keyID, ok := parseGCPRecaptchaKeyPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaKeyFixture(project, keyID))
	return true
}

func handleGCPRecaptchaEnterpriseUpdateKey(w http.ResponseWriter, r *http.Request, path string) bool {
	project, keyID, ok := parseGCPRecaptchaKeyPath(path)
	if !ok || r.Method != http.MethodPatch {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	key := gcpRecaptchaBodyMap(body, "key")
	if len(key) == 0 {
		respondGCPRecaptchaInvalidArgument(w, path, "key is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/keys/%s", project, keyID)
	if got := gcpRecaptchaString(key, "name"); got == "" || got != expectedName {
		respondGCPRecaptchaInvalidArgument(w, path, "key.name must match the requested resource")
		return true
	}
	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = strings.TrimSpace(gcpRecaptchaString(body, "updateMask"))
	}
	if mask == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "updateMask is required")
		return true
	}
	fixture := gcpRecaptchaKeyFixture(project, keyID)
	if displayName := gcpRecaptchaString(key, "displayName"); displayName != "" {
		fixture["displayName"] = displayName
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRecaptchaEnterpriseDeleteKey(w http.ResponseWriter, path string) bool {
	project, keyID, ok := parseGCPRecaptchaKeyPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"deleted":  fmt.Sprintf("projects/%s/keys/%s", project, keyID),
		"provider": providerGCP,
	})
	return true
}

func handleGCPRecaptchaEnterpriseMigrateKey(w http.ResponseWriter, path string) bool {
	project, keyID, ok := parseGCPRecaptchaKeyActionPath(path, "migrate")
	if !ok {
		return false
	}
	fixture := gcpRecaptchaKeyFixture(project, keyID)
	labels := map[string]string{"source": "stackyard", "migrated": "true"}
	fixture["labels"] = labels
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRecaptchaEnterpriseRetrieveLegacySecretKey(w http.ResponseWriter, path string) bool {
	_, keyID, ok := parseGCPRecaptchaKeyActionPath(path, "retrieveLegacySecretKey")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"legacySecretKey": "legacy-secret-" + keyID,
	})
	return true
}

func handleGCPRecaptchaEnterpriseGetMetrics(w http.ResponseWriter, path string) bool {
	project, keyID, ok := parseGCPRecaptchaMetricsPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaMetricsFixture(project, keyID))
	return true
}

func handleGCPRecaptchaEnterpriseListIpOverrides(w http.ResponseWriter, r *http.Request, path string) bool {
	project, keyID, ok := parseGCPRecaptchaIpOverridesPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRecaptchaPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{"ip": "203.0.113.1", "overrideType": "ALLOW"},
		{"ip": "198.51.100.0/24", "overrideType": "BLOCK"},
	}
	response := gcpRecaptchaPaginatedResponse("ipOverrides", items, pageSize, start)
	response["name"] = fmt.Sprintf("projects/%s/keys/%s", project, keyID)
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecaptchaEnterpriseAddIpOverride(w http.ResponseWriter, r *http.Request, path string) bool {
	project, keyID, ok := parseGCPRecaptchaKeyActionPath(path, "addIpOverride")
	if !ok {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	override := gcpRecaptchaBodyMap(body, "ipOverrideData")
	if len(override) == 0 || gcpRecaptchaString(override, "ip") == "" || gcpRecaptchaString(override, "overrideType") == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "ipOverrideData.ip and ipOverrideData.overrideType are required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":           fmt.Sprintf("projects/%s/keys/%s", project, keyID),
		"ipOverrideData": override,
		"provider":       providerGCP,
	})
	return true
}

func handleGCPRecaptchaEnterpriseRemoveIpOverride(w http.ResponseWriter, r *http.Request, path string) bool {
	project, keyID, ok := parseGCPRecaptchaKeyActionPath(path, "removeIpOverride")
	if !ok {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	override := gcpRecaptchaBodyMap(body, "ipOverrideData")
	if len(override) == 0 || gcpRecaptchaString(override, "ip") == "" || gcpRecaptchaString(override, "overrideType") == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "ipOverrideData.ip and ipOverrideData.overrideType are required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":           fmt.Sprintf("projects/%s/keys/%s", project, keyID),
		"ipOverrideData": override,
		"provider":       providerGCP,
	})
	return true
}

func handleGCPRecaptchaEnterpriseCreateFirewallPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || !isGCPRecaptchaCollectionTail(tail, "firewallpolicies") {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpRecaptchaBodyMap(body, "firewallPolicy")
	if len(policy) == 0 {
		respondGCPRecaptchaInvalidArgument(w, path, "firewallPolicy is required")
		return true
	}
	policyID := strings.TrimSpace(r.URL.Query().Get("firewallPolicyId"))
	if policyID == "" {
		policyID = "policy-1"
	}
	fixture := gcpRecaptchaFirewallPolicyFixture(project, policyID)
	if condition := gcpRecaptchaString(policy, "condition"); condition != "" {
		fixture["condition"] = condition
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRecaptchaEnterpriseListFirewallPolicies(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || !isGCPRecaptchaCollectionTail(tail, "firewallpolicies") {
		return false
	}
	pageSize, start, valid := parseGCPRecaptchaPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRecaptchaFirewallPolicyFixture(project, "policy-1"),
		gcpRecaptchaFirewallPolicyFixture(project, "policy-2"),
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaPaginatedResponse("firewallPolicies", items, pageSize, start))
	return true
}

func handleGCPRecaptchaEnterpriseGetFirewallPolicy(w http.ResponseWriter, path string) bool {
	project, policyID, ok := parseGCPRecaptchaFirewallPolicyPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaFirewallPolicyFixture(project, policyID))
	return true
}

func handleGCPRecaptchaEnterpriseUpdateFirewallPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	project, policyID, ok := parseGCPRecaptchaFirewallPolicyPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpRecaptchaBodyMap(body, "firewallPolicy")
	if len(policy) == 0 {
		respondGCPRecaptchaInvalidArgument(w, path, "firewallPolicy is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/firewallpolicies/%s", project, policyID)
	if got := gcpRecaptchaString(policy, "name"); got == "" || got != expectedName {
		respondGCPRecaptchaInvalidArgument(w, path, "firewallPolicy.name must match the requested resource")
		return true
	}
	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = strings.TrimSpace(gcpRecaptchaString(body, "updateMask"))
	}
	if mask == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "updateMask is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaFirewallPolicyFixture(project, policyID))
	return true
}

func handleGCPRecaptchaEnterpriseDeleteFirewallPolicy(w http.ResponseWriter, path string) bool {
	project, policyID, ok := parseGCPRecaptchaFirewallPolicyPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"deleted":  fmt.Sprintf("projects/%s/firewallpolicies/%s", project, policyID),
		"provider": providerGCP,
	})
	return true
}

func handleGCPRecaptchaEnterpriseReorderFirewallPolicies(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 1 || tail[0] != "firewallpolicies:reorder" {
		return false
	}
	body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, exists := body["names"]; !exists {
		respondGCPRecaptchaInvalidArgument(w, path, "names is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"firewallPolicies": []any{
			gcpRecaptchaFirewallPolicyFixture(project, "policy-1"),
		},
	})
	return true
}

func handleGCPRecaptchaEnterpriseListRelatedAccountGroups(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || !isGCPRecaptchaCollectionTail(tail, "relatedaccountgroups") {
		return false
	}
	pageSize, start, valid := parseGCPRecaptchaPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"name": fmt.Sprintf("projects/%s/relatedaccountgroups/group-1", project),
		},
		{
			"name": fmt.Sprintf("projects/%s/relatedaccountgroups/group-2", project),
		},
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaPaginatedResponse("relatedAccountGroups", items, pageSize, start))
	return true
}

func handleGCPRecaptchaEnterpriseListRelatedAccountGroupMemberships(w http.ResponseWriter, r *http.Request, path string) bool {
	project, groupID, ok := parseGCPRecaptchaRelatedAccountGroupMembershipCollection(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRecaptchaPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"name": fmt.Sprintf("projects/%s/relatedaccountgroups/%s/memberships/member-1", project, groupID),
		},
		{
			"name": fmt.Sprintf("projects/%s/relatedaccountgroups/%s/memberships/member-2", project, groupID),
		},
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaPaginatedResponse("relatedAccountGroupMemberships", items, pageSize, start))
	return true
}

func handleGCPRecaptchaEnterpriseSearchRelatedAccountGroupMemberships(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 1 || tail[0] != "relatedaccountgroupmemberships:search" {
		return false
	}
	accountID := strings.TrimSpace(r.URL.Query().Get("accountId"))
	if accountID == "" && r.Method == http.MethodPost {
		body, valid := decodeGCPRecaptchaJSONBody(w, r, path)
		if !valid {
			return true
		}
		accountID = gcpRecaptchaString(body, "accountId")
	}
	if accountID == "" {
		respondGCPRecaptchaInvalidArgument(w, path, "accountId is required")
		return true
	}
	pageSize, start, valid := parseGCPRecaptchaPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"name":      fmt.Sprintf("projects/%s/relatedaccountgroups/group-1/memberships/%s", project, accountID),
			"accountId": accountID,
		},
	}
	respondJSON(w, http.StatusOK, gcpRecaptchaPaginatedResponse("relatedAccountGroupMemberships", items, pageSize, start))
	return true
}

func handleGCPRecaptchaEnterpriseGetProjectMetadata(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 1 || tail[0] != "projectmetadata" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":         fmt.Sprintf("projects/%s/projectmetadata", project),
		"billingTier":  "ENTERPRISE",
		"defaultScore": 0.7,
	})
	return true
}

func parseGCPRecaptchaProjectPath(path string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 {
		return "", nil, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", nil, false
	}
	return project, parts[4:], true
}

func parseGCPRecaptchaProjectName(name string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 2 || parts[0] != "projects" || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}

func parseGCPRecaptchaKeyPath(path string) (project, keyID string, ok bool) {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 2 || tail[0] != "keys" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", false
	}
	return project, strings.TrimSpace(tail[1]), true
}

func parseGCPRecaptchaKeyName(name string) (project, keyID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "keys" {
		return "", "", false
	}
	if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[3]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[3]), true
}

func parseGCPRecaptchaMetricsPath(path string) (project, keyID string, ok bool) {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 3 || tail[0] != "keys" || tail[2] != "metrics" || strings.TrimSpace(tail[1]) == "" {
		return "", "", false
	}
	return project, strings.TrimSpace(tail[1]), true
}

func parseGCPRecaptchaMetricsName(name string) (project, keyID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 5 || parts[0] != "projects" || parts[2] != "keys" || parts[4] != "metrics" {
		return "", "", false
	}
	if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[3]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[3]), true
}

func parseGCPRecaptchaKeyActionPath(path, action string) (project, keyID string, ok bool) {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 2 || tail[0] != "keys" {
		return "", "", false
	}
	parts := strings.Split(strings.TrimSpace(tail[1]), ":")
	if len(parts) != 2 || parts[1] != action || strings.TrimSpace(parts[0]) == "" {
		return "", "", false
	}
	return project, strings.TrimSpace(parts[0]), true
}

func parseGCPRecaptchaIpOverridesPath(path string) (project, keyID string, ok bool) {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 3 || tail[0] != "keys" || tail[2] != "ipOverrides" || strings.TrimSpace(tail[1]) == "" {
		return "", "", false
	}
	return project, strings.TrimSpace(tail[1]), true
}

func parseGCPRecaptchaFirewallPolicyPath(path string) (project, policyID string, ok bool) {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 2 || tail[0] != "firewallpolicies" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", false
	}
	return project, strings.TrimSpace(tail[1]), true
}

func parseGCPRecaptchaFirewallPolicyName(name string) (project, policyID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "firewallpolicies" {
		return "", "", false
	}
	if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[3]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[3]), true
}

func parseGCPRecaptchaAssessmentActionPath(path, action string) (project, assessmentID string, ok bool) {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 2 || tail[0] != "assessments" {
		return "", "", false
	}
	parts := strings.Split(strings.TrimSpace(tail[1]), ":")
	if len(parts) != 2 || parts[1] != action || strings.TrimSpace(parts[0]) == "" {
		return "", "", false
	}
	return project, strings.TrimSpace(parts[0]), true
}

func parseGCPRecaptchaAssessmentName(name string) (project, assessmentID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "assessments" {
		return "", "", false
	}
	if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[3]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[3]), true
}

func parseGCPRecaptchaRelatedAccountGroupMembershipCollection(path string) (project, groupID string, ok bool) {
	project, tail, ok := parseGCPRecaptchaProjectPath(path)
	if !ok || len(tail) != 3 || tail[0] != "relatedaccountgroups" || tail[2] != "memberships" || strings.TrimSpace(tail[1]) == "" {
		return "", "", false
	}
	return project, strings.TrimSpace(tail[1]), true
}

func parseGCPRecaptchaRelatedAccountGroupName(name string) (project, groupID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "relatedaccountgroups" {
		return "", "", false
	}
	if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[3]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[3]), true
}

func parseGCPRecaptchaRelatedAccountGroupMembershipName(name string) (project, groupID, membershipID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "relatedaccountgroups" || parts[4] != "memberships" {
		return "", "", "", false
	}
	if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[3]) == "" || strings.TrimSpace(parts[5]) == "" {
		return "", "", "", false
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[3]), strings.TrimSpace(parts[5]), true
}

func parseGCPRecaptchaPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPRecaptchaInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPRecaptchaInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func decodeGCPRecaptchaJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.UseNumber()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPRecaptchaInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpRecaptchaBodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return map[string]any{}
	}
	raw, ok := body[key]
	if !ok {
		return map[string]any{}
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return m
}

func gcpRecaptchaNestedMap(body map[string]any, key string) map[string]any {
	return gcpRecaptchaBodyMap(body, key)
}

func gcpRecaptchaString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	raw, ok := body[key]
	if !ok {
		return ""
	}
	val, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(val)
}

func gcpRecaptchaStringSlice(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		text, ok := item.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func gcpRecaptchaPaginatedResponse(field string, items []map[string]any, pageSize, start int) map[string]any {
	if start > len(items) {
		start = len(items)
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = fmt.Sprintf("%d", end)
	}
	trimmed := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		trimmed = append(trimmed, item)
	}
	return map[string]any{
		field:           trimmed,
		"nextPageToken": next,
	}
}

func gcpRecaptchaAssessmentFixture(project, assessmentID, token, siteKey, accountID string) map[string]any {
	if token == "" {
		token = "stackyard-token"
	}
	if siteKey == "" {
		siteKey = fmt.Sprintf("projects/%s/keys/site-key-1", project)
	}
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/assessments/%s", project, assessmentID),
		"event": map[string]any{
			"token":     token,
			"siteKey":   siteKey,
			"accountId": accountID,
		},
		"riskAnalysis": map[string]any{
			"score": 0.9,
			"reasons": []any{
				"AUTOMATION",
			},
		},
		"tokenProperties": map[string]any{
			"valid":         true,
			"action":        "login",
			"invalidReason": "INVALID_REASON_UNSPECIFIED",
		},
		"createTime": gcpRecaptchaReferenceTime.Format(time.RFC3339Nano),
	}
}

func gcpRecaptchaKeyFixture(project, keyID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/keys/%s", project, keyID),
		"displayName": "Stackyard Key " + keyID,
		"labels": map[string]any{
			"env": "staged",
		},
		"createTime": gcpRecaptchaReferenceTime.Format(time.RFC3339Nano),
		"webSettings": map[string]any{
			"allowAllDomains": true,
			"integrationType": "SCORE",
		},
	}
}

func gcpRecaptchaMetricsFixture(project, keyID string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/keys/%s/metrics", project, keyID),
		"startTime": gcpRecaptchaReferenceTime.Format(time.RFC3339Nano),
		"scoreMetrics": []any{
			map[string]any{
				"score": 0.9,
				"overallMetrics": []any{
					map[string]any{"action": "login", "count": "12"},
				},
			},
		},
		"challengeMetrics": []any{
			map[string]any{
				"pageloadCount":  "20",
				"nocaptchaCount": "15",
			},
		},
	}
}

func gcpRecaptchaFirewallPolicyFixture(project, policyID string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("projects/%s/firewallpolicies/%s", project, policyID),
		"path":      "/*",
		"condition": "true",
		"action":    "ALLOW",
	}
}

func respondGCPRecaptchaInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPRecaptchaError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPRecaptchaError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func isGCPRecaptchaCollectionTail(tail []string, collection string) bool {
	return len(tail) == 1 && tail[0] == collection
}

func handleGCPContractProbe_recaptchaenterprise(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "recaptchaenterprise") {
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
			"name":     "projects/stackyard/locations/us-central1/recaptchaenterprise/sample",
			"service":  "recaptchaenterprise",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
