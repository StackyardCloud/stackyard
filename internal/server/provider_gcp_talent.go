package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpTalentReferenceTime  = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpTalentProjectIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	gcpTalentResourceRegex  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
)

func (s *Server) handleGCPTalentRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_talent(w, r) {
		return true
	}

	path := normalizeGCPTalentPath(rawRequestPath(r))
	if !isGCPTalentPath(path, hasGCPTalentHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPTalentListTenants(w, r, path) {
			return true
		}
		if handleGCPTalentCompleteQuery(w, r, path) {
			return true
		}
		if handleGCPTalentGetTenant(w, path) {
			return true
		}
		if handleGCPTalentListCompanies(w, r, path) {
			return true
		}
		if handleGCPTalentGetCompany(w, path) {
			return true
		}
		if handleGCPTalentListJobs(w, r, path) {
			return true
		}
		if handleGCPTalentGetJob(w, path) {
			return true
		}
		if handleGCPTalentListOperations(w, r, path) {
			return true
		}
		if handleGCPTalentGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPTalentCreateTenant(w, r, path) {
			return true
		}
		if handleGCPTalentCreateCompany(w, r, path) {
			return true
		}
		if handleGCPTalentCreateJob(w, r, path) {
			return true
		}
		if handleGCPTalentBatchCreateJobs(w, r, path) {
			return true
		}
		if handleGCPTalentBatchUpdateJobs(w, r, path) {
			return true
		}
		if handleGCPTalentBatchDeleteJobs(w, r, path) {
			return true
		}
		if handleGCPTalentSearchJobs(w, r, path) {
			return true
		}
		if handleGCPTalentSearchJobsForAlert(w, r, path) {
			return true
		}
		if handleGCPTalentCreateClientEvent(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPTalentUpdateTenant(w, r, path) {
			return true
		}
		if handleGCPTalentUpdateCompany(w, r, path) {
			return true
		}
		if handleGCPTalentUpdateJob(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPTalentDeleteTenant(w, path) {
			return true
		}
		if handleGCPTalentDeleteCompany(w, path) {
			return true
		}
		if handleGCPTalentDeleteJob(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPTalentPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPTalentHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "talent",
		"talent-apiv4",
		"talent_apiv4",
		"cloud-talent",
		"cloud_talent",
		"talent-solution",
		"talentsolution",
		"gcp-talent-solution":
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-talent-apiv4") || strings.Contains(ua, "cloud.google.com/go/talent/apiv4")
}

func isGCPTalentPath(path string, includeHint bool) bool {
	if isGCPTalentGRPCPath(path) {
		return true
	}
	if _, ok := parseGCPTalentTenantsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPTalentTenantPath(path); ok {
		return true
	}
	if _, ok := parseGCPTalentCompaniesCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPTalentCompanyPath(path); ok {
		return true
	}
	if _, ok := parseGCPTalentJobsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPTalentJobPath(path); ok {
		return true
	}
	if _, action, ok := parseGCPTalentJobsActionPath(path); ok {
		return action == "batchCreate" || action == "batchUpdate" || action == "batchDelete" || action == "search" || action == "searchForAlert"
	}
	if _, ok := parseGCPTalentCompleteQueryPath(path); ok {
		return true
	}
	if _, ok := parseGCPTalentClientEventsCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPTalentOperationsCollectionPath(path); ok {
		return includeHint
	}
	if _, _, _, ok := parseGCPTalentOperationPath(path); ok {
		return includeHint
	}

	return includeHint && strings.HasPrefix(path, "/gcp/v4/projects/")
}

func isGCPTalentGRPCPath(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/gcp/google.cloud.talent.v4.CompanyService/") ||
		strings.HasPrefix(trimmed, "/gcp/google.cloud.talent.v4.TenantService/") ||
		strings.HasPrefix(trimmed, "/gcp/google.cloud.talent.v4.JobService/") ||
		strings.HasPrefix(trimmed, "/gcp/google.cloud.talent.v4.Completion/") ||
		strings.HasPrefix(trimmed, "/gcp/google.cloud.talent.v4.EventService/")
}

func handleGCPTalentListTenants(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, ok := parseGCPTalentTenantsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentProjectID(projectID) {
		respondGCPTalentInvalidArgument(w, path, "project is invalid")
		return true
	}
	pageSize, start, ok := parseGCPTalentPagination(w, r, path, 100, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTalentTenant(fmt.Sprintf("projects/%s/tenants/tenant-1", projectID), "tenant-ext-1"),
		gcpTalentTenant(fmt.Sprintf("projects/%s/tenants/tenant-2", projectID), "tenant-ext-2"),
	}
	return respondGCPTalentList(w, "tenants", items, pageSize, start, path)
}

func handleGCPTalentGetTenant(w http.ResponseWriter, path string) bool {
	name, _, tenantID, ok := parseGCPTalentTenantPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentTenantName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(tenantID), "missing") {
		respondGCPTalentNotFound(w, path, "tenant not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTalentTenant(name, "tenant-ext-"+tenantID))
	return true
}

func handleGCPTalentCreateTenant(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, ok := parseGCPTalentTenantsCollectionPath(path)
	if !ok {
		return false
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	externalID := strings.TrimSpace(gcpTalentString(body, "externalId"))
	if externalID == "" {
		respondGCPTalentInvalidArgument(w, path, "tenant.externalId is required")
		return true
	}

	name := strings.TrimSpace(gcpTalentString(body, "name"))
	tenantID := "tenant-created-1"
	if name != "" {
		if !strings.HasPrefix(name, fmt.Sprintf("projects/%s/tenants/", projectID)) || !isGCPTalentTenantName(name) {
			respondGCPTalentInvalidArgument(w, path, "tenant.name must match parent")
			return true
		}
		tenantID = pathBase(name)
	}
	name = fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID)
	respondJSON(w, http.StatusOK, gcpTalentTenant(name, externalID))
	return true
}

func handleGCPTalentUpdateTenant(w http.ResponseWriter, r *http.Request, path string) bool {
	name, _, _, ok := parseGCPTalentTenantPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentTenantName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if bodyName := strings.TrimSpace(gcpTalentString(body, "name")); bodyName == "" || bodyName != name {
		respondGCPTalentInvalidArgument(w, path, "tenant.name must match requested resource")
		return true
	}
	mask, ok := parseGCPTalentUpdateMask(w, r, path)
	if !ok {
		return true
	}
	if len(mask) == 0 {
		respondGCPTalentInvalidArgument(w, path, "updateMask is required")
		return true
	}
	for _, field := range mask {
		if field != "external_id" && field != "externalId" {
			respondGCPTalentInvalidArgument(w, path, "updateMask has unsupported fields")
			return true
		}
	}
	externalID := strings.TrimSpace(gcpTalentString(body, "externalId"))
	if externalID == "" {
		respondGCPTalentInvalidArgument(w, path, "tenant.externalId is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTalentTenant(name, externalID))
	return true
}

func handleGCPTalentDeleteTenant(w http.ResponseWriter, path string) bool {
	name, _, _, ok := parseGCPTalentTenantPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentTenantName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTalentListCompanies(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPTalentCompaniesCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentTenantName(parent) {
		respondGCPTalentInvalidArgument(w, path, "parent is invalid")
		return true
	}
	pageSize, start, ok := parseGCPTalentPagination(w, r, path, 100, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTalentCompany(parent+"/companies/company-1", "Stackyard Inc", "company-ext-1"),
		gcpTalentCompany(parent+"/companies/company-2", "Example Corp", "company-ext-2"),
	}
	if requireOpenRaw := strings.TrimSpace(r.URL.Query().Get("requireOpenJobs")); requireOpenRaw != "" {
		requireOpenJobs, err := parseOptionalBool(requireOpenRaw, false)
		if err != nil {
			respondGCPTalentInvalidArgument(w, path, "requireOpenJobs must be boolean")
			return true
		}
		if requireOpenJobs {
			items = items[:1]
		}
	}
	return respondGCPTalentList(w, "companies", items, pageSize, start, path)
}

func handleGCPTalentGetCompany(w http.ResponseWriter, path string) bool {
	name, _, _, companyID, ok := parseGCPTalentCompanyPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentCompanyName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(companyID), "missing") {
		respondGCPTalentNotFound(w, path, "company not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTalentCompany(name, "Company "+companyID, "company-ext-"+companyID))
	return true
}

func handleGCPTalentCreateCompany(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPTalentCompaniesCollectionPath(path)
	if !ok {
		return false
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	displayName := strings.TrimSpace(gcpTalentString(body, "displayName"))
	externalID := strings.TrimSpace(gcpTalentString(body, "externalId"))
	if displayName == "" || externalID == "" {
		respondGCPTalentInvalidArgument(w, path, "company.displayName and company.externalId are required")
		return true
	}
	name := strings.TrimSpace(gcpTalentString(body, "name"))
	companyID := "company-created-1"
	if name != "" {
		if !strings.HasPrefix(name, parent+"/companies/") || !isGCPTalentCompanyName(name) {
			respondGCPTalentInvalidArgument(w, path, "company.name must match parent")
			return true
		}
		companyID = pathBase(name)
	}
	name = parent + "/companies/" + companyID
	respondJSON(w, http.StatusOK, gcpTalentCompany(name, displayName, externalID))
	return true
}

func handleGCPTalentUpdateCompany(w http.ResponseWriter, r *http.Request, path string) bool {
	name, _, _, _, ok := parseGCPTalentCompanyPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentCompanyName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if bodyName := strings.TrimSpace(gcpTalentString(body, "name")); bodyName == "" || bodyName != name {
		respondGCPTalentInvalidArgument(w, path, "company.name must match requested resource")
		return true
	}
	mask, ok := parseGCPTalentUpdateMask(w, r, path)
	if !ok {
		return true
	}
	if len(mask) == 0 {
		respondGCPTalentInvalidArgument(w, path, "updateMask is required")
		return true
	}
	allowed := map[string]struct{}{
		"display_name": {}, "displayName": {},
		"external_id": {}, "externalId": {},
		"website_uri": {}, "websiteUri": {},
	}
	for _, field := range mask {
		if _, ok := allowed[field]; !ok {
			respondGCPTalentInvalidArgument(w, path, "updateMask has unsupported fields")
			return true
		}
	}
	displayName := strings.TrimSpace(gcpTalentString(body, "displayName"))
	externalID := strings.TrimSpace(gcpTalentString(body, "externalId"))
	if displayName == "" || externalID == "" {
		respondGCPTalentInvalidArgument(w, path, "company.displayName and company.externalId are required")
		return true
	}
	out := gcpTalentCompany(name, displayName, externalID)
	if website := strings.TrimSpace(gcpTalentString(body, "websiteUri")); website != "" {
		out["websiteUri"] = website
	}
	respondJSON(w, http.StatusOK, out)
	return true
}

func handleGCPTalentDeleteCompany(w http.ResponseWriter, path string) bool {
	name, _, _, _, ok := parseGCPTalentCompanyPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentCompanyName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTalentListJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPTalentJobsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentTenantName(parent) {
		respondGCPTalentInvalidArgument(w, path, "parent is invalid")
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter == "" {
		respondGCPTalentInvalidArgument(w, path, "filter is required")
		return true
	}
	filterSpec, ok := parseGCPTalentListJobsFilter(filter, parent)
	if !ok {
		respondGCPTalentInvalidArgument(w, path, "filter is invalid")
		return true
	}
	pageSize, start, ok := parseGCPTalentPagination(w, r, path, 100, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTalentJob(parent+"/jobs/job-1", parent+"/companies/company-1", "req-1", "Software Engineer", "Build distributed systems"),
		gcpTalentJob(parent+"/jobs/job-2", parent+"/companies/company-2", "req-2", "Site Reliability Engineer", "Operate reliable infrastructure"),
	}
	items = gcpTalentFilterJobs(items, filterSpec)
	response, ok := gcpTalentPaginateList("jobs", items, pageSize, start, path)
	if !ok {
		respondGCPTalentInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	response["metadata"] = map[string]any{"requestId": "talent-listjobs-req-1"}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPTalentGetJob(w http.ResponseWriter, path string) bool {
	name, parent, _, jobID, ok := parseGCPTalentJobPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentJobName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(jobID), "missing") {
		respondGCPTalentNotFound(w, path, "job not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTalentJob(name, parent+"/companies/company-1", "req-"+jobID, "Job "+jobID, "Deterministic staged job"))
	return true
}

func handleGCPTalentCreateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPTalentJobsCollectionPath(path)
	if !ok {
		return false
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	company := strings.TrimSpace(gcpTalentString(body, "company"))
	requisitionID := strings.TrimSpace(gcpTalentString(body, "requisitionId"))
	title := strings.TrimSpace(gcpTalentString(body, "title"))
	description := strings.TrimSpace(gcpTalentString(body, "description"))
	if company == "" || requisitionID == "" || title == "" || description == "" {
		respondGCPTalentInvalidArgument(w, path, "job.company, job.requisitionId, job.title, and job.description are required")
		return true
	}
	if !isGCPTalentCompanyName(company) || !strings.HasPrefix(company, parent+"/companies/") {
		respondGCPTalentInvalidArgument(w, path, "job.company is invalid")
		return true
	}
	jobID := "job-created-1"
	if name := strings.TrimSpace(gcpTalentString(body, "name")); name != "" {
		if !strings.HasPrefix(name, parent+"/jobs/") || !isGCPTalentJobName(name) {
			respondGCPTalentInvalidArgument(w, path, "job.name must match parent")
			return true
		}
		jobID = pathBase(name)
	}
	name := parent + "/jobs/" + jobID
	out := gcpTalentJob(name, company, requisitionID, title, description)
	if rawAddresses, ok := body["addresses"].([]any); ok && len(rawAddresses) > 0 {
		addresses := make([]string, 0, len(rawAddresses))
		for _, raw := range rawAddresses {
			val, _ := raw.(string)
			val = strings.TrimSpace(val)
			if val != "" {
				addresses = append(addresses, val)
			}
		}
		if len(addresses) > 0 {
			out["addresses"] = addresses
		}
	}
	if languageCode := strings.TrimSpace(gcpTalentString(body, "languageCode")); languageCode != "" {
		out["languageCode"] = languageCode
	}
	respondJSON(w, http.StatusOK, out)
	return true
}

func handleGCPTalentUpdateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	name, parent, _, _, ok := parseGCPTalentJobPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentJobName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if bodyName := strings.TrimSpace(gcpTalentString(body, "name")); bodyName == "" || bodyName != name {
		respondGCPTalentInvalidArgument(w, path, "job.name must match requested resource")
		return true
	}
	mask, ok := parseGCPTalentUpdateMask(w, r, path)
	if !ok {
		return true
	}
	if len(mask) == 0 {
		respondGCPTalentInvalidArgument(w, path, "updateMask is required")
		return true
	}
	allowed := map[string]struct{}{
		"company": {}, "requisition_id": {}, "requisitionId": {}, "title": {}, "description": {}, "addresses": {}, "language_code": {}, "languageCode": {},
	}
	for _, field := range mask {
		if _, ok := allowed[field]; !ok {
			respondGCPTalentInvalidArgument(w, path, "updateMask has unsupported fields")
			return true
		}
	}
	company := strings.TrimSpace(gcpTalentString(body, "company"))
	requisitionID := strings.TrimSpace(gcpTalentString(body, "requisitionId"))
	title := strings.TrimSpace(gcpTalentString(body, "title"))
	description := strings.TrimSpace(gcpTalentString(body, "description"))
	if company == "" || requisitionID == "" || title == "" || description == "" {
		respondGCPTalentInvalidArgument(w, path, "job.company, job.requisitionId, job.title, and job.description are required")
		return true
	}
	if !isGCPTalentCompanyName(company) || !strings.HasPrefix(company, parent+"/companies/") {
		respondGCPTalentInvalidArgument(w, path, "job.company is invalid")
		return true
	}
	out := gcpTalentJob(name, company, requisitionID, title, description)
	if rawAddresses, ok := body["addresses"].([]any); ok && len(rawAddresses) > 0 {
		addresses := make([]string, 0, len(rawAddresses))
		for _, raw := range rawAddresses {
			val, _ := raw.(string)
			val = strings.TrimSpace(val)
			if val != "" {
				addresses = append(addresses, val)
			}
		}
		if len(addresses) > 0 {
			out["addresses"] = addresses
		}
	}
	if languageCode := strings.TrimSpace(gcpTalentString(body, "languageCode")); languageCode != "" {
		out["languageCode"] = languageCode
	}
	respondJSON(w, http.StatusOK, out)
	return true
}

func handleGCPTalentDeleteJob(w http.ResponseWriter, path string) bool {
	name, _, _, _, ok := parseGCPTalentJobPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentJobName(name) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTalentBatchCreateJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, action, ok := parseGCPTalentJobsActionPath(path)
	if !ok || action != "batchCreate" {
		return false
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if requestParent := strings.TrimSpace(gcpTalentString(body, "parent")); requestParent == "" || requestParent != parent {
		respondGCPTalentInvalidArgument(w, path, "parent is required and must match requested resource")
		return true
	}
	rawJobs, ok := body["jobs"].([]any)
	if !ok || len(rawJobs) == 0 {
		respondGCPTalentInvalidArgument(w, path, "jobs must include at least one entry")
		return true
	}
	if len(rawJobs) > 200 {
		respondGCPTalentOutOfRange(w, path, "jobs cannot include more than 200 entries")
		return true
	}

	results := make([]map[string]any, 0, len(rawJobs))
	resourceNames := make([]string, 0, len(rawJobs))
	for idx, raw := range rawJobs {
		job, ok := raw.(map[string]any)
		if !ok {
			respondGCPTalentInvalidArgument(w, path, fmt.Sprintf("jobs[%d] is invalid", idx))
			return true
		}
		parsed, resourceName, err := parseAndValidateGCPTalentJobBody(job, parent, false)
		if err != "" {
			respondGCPTalentInvalidArgument(w, path, fmt.Sprintf("jobs[%d] %s", idx, err))
			return true
		}
		results = append(results, map[string]any{
			"job":    parsed,
			"status": map[string]any{"code": 0},
		})
		resourceNames = append(resourceNames, resourceName)
	}

	projectID, tenantID := gcpTalentProjectTenantFromParent(parent)
	operationID := "batchCreateJobs-1"
	respondJSON(w, http.StatusOK, gcpTalentOperation(projectID, tenantID, operationID, "type.googleapis.com/google.cloud.talent.v4.BatchCreateJobsResponse", map[string]any{
		"jobResults": results,
	}, resourceNames))
	return true
}

func handleGCPTalentBatchUpdateJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, action, ok := parseGCPTalentJobsActionPath(path)
	if !ok || action != "batchUpdate" {
		return false
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if requestParent := strings.TrimSpace(gcpTalentString(body, "parent")); requestParent == "" || requestParent != parent {
		respondGCPTalentInvalidArgument(w, path, "parent is required and must match requested resource")
		return true
	}
	rawJobs, ok := body["jobs"].([]any)
	if !ok || len(rawJobs) == 0 {
		respondGCPTalentInvalidArgument(w, path, "jobs must include at least one entry")
		return true
	}
	if len(rawJobs) > 200 {
		respondGCPTalentOutOfRange(w, path, "jobs cannot include more than 200 entries")
		return true
	}

	results := make([]map[string]any, 0, len(rawJobs))
	resourceNames := make([]string, 0, len(rawJobs))
	for idx, raw := range rawJobs {
		job, ok := raw.(map[string]any)
		if !ok {
			respondGCPTalentInvalidArgument(w, path, fmt.Sprintf("jobs[%d] is invalid", idx))
			return true
		}
		parsed, resourceName, err := parseAndValidateGCPTalentJobBody(job, parent, true)
		if err != "" {
			respondGCPTalentInvalidArgument(w, path, fmt.Sprintf("jobs[%d] %s", idx, err))
			return true
		}
		results = append(results, map[string]any{
			"job":    parsed,
			"status": map[string]any{"code": 0},
		})
		resourceNames = append(resourceNames, resourceName)
	}

	projectID, tenantID := gcpTalentProjectTenantFromParent(parent)
	operationID := "batchUpdateJobs-1"
	respondJSON(w, http.StatusOK, gcpTalentOperation(projectID, tenantID, operationID, "type.googleapis.com/google.cloud.talent.v4.BatchUpdateJobsResponse", map[string]any{
		"jobResults": results,
	}, resourceNames))
	return true
}

func handleGCPTalentBatchDeleteJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, action, ok := parseGCPTalentJobsActionPath(path)
	if !ok || action != "batchDelete" {
		return false
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if requestParent := strings.TrimSpace(gcpTalentString(body, "parent")); requestParent == "" || requestParent != parent {
		respondGCPTalentInvalidArgument(w, path, "parent is required and must match requested resource")
		return true
	}
	rawNames, ok := body["names"].([]any)
	if !ok || len(rawNames) == 0 {
		respondGCPTalentInvalidArgument(w, path, "names must include at least one entry")
		return true
	}
	if len(rawNames) > 200 {
		respondGCPTalentOutOfRange(w, path, "names cannot include more than 200 entries")
		return true
	}

	results := make([]map[string]any, 0, len(rawNames))
	resourceNames := make([]string, 0, len(rawNames))
	for idx, raw := range rawNames {
		name, _ := raw.(string)
		name = strings.TrimSpace(name)
		if !isGCPTalentJobName(name) || !strings.HasPrefix(name, parent+"/jobs/") {
			respondGCPTalentInvalidArgument(w, path, fmt.Sprintf("names[%d] is invalid", idx))
			return true
		}
		resourceNames = append(resourceNames, name)
		results = append(results, map[string]any{
			"job": map[string]any{
				"name":          name,
				"company":       parent + "/companies/company-1",
				"languageCode":  "en-US",
				"requisitionId": "req-" + pathBase(name),
			},
			"status": map[string]any{"code": 0},
		})
	}

	projectID, tenantID := gcpTalentProjectTenantFromParent(parent)
	operationID := "batchDeleteJobs-1"
	respondJSON(w, http.StatusOK, gcpTalentOperation(projectID, tenantID, operationID, "type.googleapis.com/google.cloud.talent.v4.BatchDeleteJobsResponse", map[string]any{
		"jobResults": results,
	}, resourceNames))
	return true
}

func handleGCPTalentSearchJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, action, ok := parseGCPTalentJobsActionPath(path)
	if !ok || action != "search" {
		return false
	}
	return handleGCPTalentSearchJobsCommon(w, r, path, parent, false)
}

func handleGCPTalentSearchJobsForAlert(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, action, ok := parseGCPTalentJobsActionPath(path)
	if !ok || action != "searchForAlert" {
		return false
	}
	return handleGCPTalentSearchJobsCommon(w, r, path, parent, true)
}

func handleGCPTalentSearchJobsCommon(w http.ResponseWriter, r *http.Request, path, parent string, alert bool) bool {
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if requestParent := strings.TrimSpace(gcpTalentString(body, "parent")); requestParent == "" || requestParent != parent {
		respondGCPTalentInvalidArgument(w, path, "parent is required and must match requested resource")
		return true
	}
	requestMetadata := gcpTalentBodyMap(body, "requestMetadata")
	if !isGCPTalentRequestMetadataValid(requestMetadata) {
		respondGCPTalentInvalidArgument(w, path, "requestMetadata is required")
		return true
	}
	pageSize, start, ok := parseGCPTalentBodyPagination(w, path, body, "maxPageSize", 100, 10)
	if !ok {
		return true
	}

	query := strings.TrimSpace(gcpTalentString(gcpTalentBodyMap(body, "jobQuery"), "query"))
	items := []map[string]any{
		gcpTalentJob(parent+"/jobs/job-1", parent+"/companies/company-1", "req-1", "Software Engineer", "Build distributed systems"),
		gcpTalentJob(parent+"/jobs/job-2", parent+"/companies/company-2", "req-2", "Site Reliability Engineer", "Operate reliable infrastructure"),
	}
	if query != "" {
		filtered := make([]map[string]any, 0, len(items))
		queryLower := strings.ToLower(query)
		for _, item := range items {
			title := strings.ToLower(gcpTalentString(item, "title"))
			description := strings.ToLower(gcpTalentString(item, "description"))
			if strings.Contains(title, queryLower) || strings.Contains(description, queryLower) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if strings.Contains(strings.ToLower(query), "nomatch") {
		items = []map[string]any{}
	}
	if alert && len(items) > 1 {
		items = items[:1]
	}

	matches := make([]map[string]any, 0, len(items))
	for _, item := range items {
		matches = append(matches, map[string]any{
			"job":               item,
			"jobSummary":        "Summary for " + gcpTalentString(item, "title"),
			"jobTitleSnippet":   "<b>" + gcpTalentString(item, "title") + "</b>",
			"searchTextSnippet": gcpTalentString(item, "description"),
		})
	}
	response, ok := gcpTalentPaginateList("matchingJobs", matches, pageSize, start, path)
	if !ok {
		respondGCPTalentInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	response["totalSize"] = len(matches)
	if alert {
		response["metadata"] = map[string]any{"requestId": "talent-search-alert-req-1"}
	} else {
		response["metadata"] = map[string]any{"requestId": "talent-search-req-1"}
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPTalentCompleteQuery(w http.ResponseWriter, r *http.Request, path string) bool {
	tenant, ok := parseGCPTalentCompleteQueryPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentTenantName(tenant) {
		respondGCPTalentInvalidArgument(w, path, "tenant is invalid")
		return true
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		respondGCPTalentInvalidArgument(w, path, "query is required")
		return true
	}
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPTalentInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if pageSize == 0 {
		pageSize = 10
	}
	if pageSize > 10 {
		respondGCPTalentOutOfRange(w, path, "pageSize cannot exceed 10")
		return true
	}
	if company := strings.TrimSpace(r.URL.Query().Get("company")); company != "" {
		if !isGCPTalentCompanyName(company) || !strings.HasPrefix(company, tenant+"/companies/") {
			respondGCPTalentInvalidArgument(w, path, "company is invalid")
			return true
		}
	}
	completionType, valid := parseGCPTalentCompletionType(r.URL.Query().Get("type"))
	if !valid {
		respondGCPTalentInvalidArgument(w, path, "type must be JOB_TITLE, COMPANY_NAME, COMBINED, or 0..3")
		return true
	}
	scope, valid := parseGCPTalentCompletionScope(r.URL.Query().Get("scope"))
	if !valid {
		respondGCPTalentInvalidArgument(w, path, "scope must be TENANT, PUBLIC, or 0..2")
		return true
	}

	results := []map[string]any{
		{
			"suggestion": "Software Engineer",
			"type":       "JOB_TITLE",
			"imageUri":   "",
		},
		{
			"suggestion": "Stackyard Inc",
			"type":       "COMPANY_NAME",
			"imageUri":   "https://example.com/logo.png",
		},
		{
			"suggestion": "Stackyard Engineer",
			"type":       "COMBINED",
			"imageUri":   "",
		},
	}
	filtered := make([]map[string]any, 0, len(results))
	queryLower := strings.ToLower(query)
	for _, item := range results {
		suggestion := strings.ToLower(gcpTalentString(item, "suggestion"))
		if strings.Contains(suggestion, queryLower) {
			filtered = append(filtered, item)
		}
	}
	results = filtered
	if completionType != "" && completionType != "COMBINED" {
		filtered = filtered[:0]
		for _, item := range results {
			if gcpTalentString(item, "type") == completionType {
				filtered = append(filtered, item)
			}
		}
		results = filtered
	}
	if scope == "TENANT" && len(results) > 1 {
		results = results[:1]
	}
	if pageSize < len(results) {
		results = results[:pageSize]
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"completionResults": results,
		"metadata": map[string]any{
			"requestId": "talent-complete-req-1",
		},
	})
	return true
}

func handleGCPTalentCreateClientEvent(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPTalentClientEventsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentTenantName(parent) {
		respondGCPTalentInvalidArgument(w, path, "parent is invalid")
		return true
	}
	body, ok := decodeGCPTalentJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	eventID := strings.TrimSpace(gcpTalentString(body, "eventId"))
	if eventID == "" {
		respondGCPTalentInvalidArgument(w, path, "clientEvent.eventId is required")
		return true
	}
	createTime := strings.TrimSpace(gcpTalentString(body, "createTime"))
	if createTime == "" {
		respondGCPTalentInvalidArgument(w, path, "clientEvent.createTime is required")
		return true
	}
	jobEvent := gcpTalentBodyMap(body, "jobEvent")
	if len(jobEvent) == 0 {
		respondGCPTalentInvalidArgument(w, path, "clientEvent.jobEvent is required")
		return true
	}
	eventType := strings.TrimSpace(gcpTalentString(jobEvent, "type"))
	if !isGCPTalentJobEventType(eventType) {
		respondGCPTalentInvalidArgument(w, path, "clientEvent.jobEvent.type is required")
		return true
	}
	rawJobs, ok := jobEvent["jobs"].([]any)
	if !ok || len(rawJobs) == 0 {
		respondGCPTalentInvalidArgument(w, path, "clientEvent.jobEvent.jobs is required")
		return true
	}
	jobs := make([]string, 0, len(rawJobs))
	for idx, raw := range rawJobs {
		jobName, _ := raw.(string)
		jobName = strings.TrimSpace(jobName)
		if !isGCPTalentJobName(jobName) || !strings.HasPrefix(jobName, parent+"/jobs/") {
			respondGCPTalentInvalidArgument(w, path, fmt.Sprintf("clientEvent.jobEvent.jobs[%d] is invalid", idx))
			return true
		}
		jobs = append(jobs, jobName)
	}

	response := map[string]any{
		"requestId":  strings.TrimSpace(gcpTalentString(body, "requestId")),
		"eventId":    eventID,
		"createTime": createTime,
		"jobEvent": map[string]any{
			"type": eventType,
			"jobs": jobs,
		},
	}
	if response["requestId"] == "" {
		response["requestId"] = "talent-event-req-1"
	}
	if eventNotes := strings.TrimSpace(gcpTalentString(body, "eventNotes")); eventNotes != "" {
		response["eventNotes"] = eventNotes
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPTalentListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, tenantID, ok := parseGCPTalentOperationsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentProjectID(projectID) || !isGCPTalentResourceID(tenantID) {
		respondGCPTalentInvalidArgument(w, path, "parent is invalid")
		return true
	}
	pageSize, start, ok := parseGCPTalentPagination(w, r, path, 100, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpTalentOperation(projectID, tenantID, "batchCreateJobs-1", "type.googleapis.com/google.cloud.talent.v4.BatchCreateJobsResponse", map[string]any{"jobResults": []map[string]any{}}, []string{}),
		gcpTalentOperation(projectID, tenantID, "batchUpdateJobs-1", "type.googleapis.com/google.cloud.talent.v4.BatchUpdateJobsResponse", map[string]any{"jobResults": []map[string]any{}}, []string{}),
		gcpTalentOperation(projectID, tenantID, "batchDeleteJobs-1", "type.googleapis.com/google.cloud.talent.v4.BatchDeleteJobsResponse", map[string]any{"jobResults": []map[string]any{}}, []string{}),
	}
	return respondGCPTalentList(w, "operations", items, pageSize, start, path)
}

func handleGCPTalentGetOperation(w http.ResponseWriter, path string) bool {
	projectID, tenantID, operationID, ok := parseGCPTalentOperationPath(path)
	if !ok {
		return false
	}
	if !isGCPTalentProjectID(projectID) || !isGCPTalentResourceID(tenantID) || !isGCPTalentResourceID(operationID) {
		respondGCPTalentInvalidArgument(w, path, "name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTalentOperation(projectID, tenantID, operationID, gcpTalentOperationResponseType(operationID), map[string]any{
		"jobResults": []map[string]any{},
	}, []string{}))
	return true
}

func parseGCPTalentTenantsCollectionPath(path string) (projectID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" {
		return "", false
	}
	projectID = strings.TrimSpace(parts[3])
	if projectID == "" {
		return "", false
	}
	return projectID, true
}

func parseGCPTalentTenantPath(path string) (name, projectID, tenantID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" {
		return "", "", "", false
	}
	projectID = strings.TrimSpace(parts[3])
	tenantID = strings.TrimSpace(parts[5])
	if projectID == "" || tenantID == "" || strings.Contains(tenantID, ":") {
		return "", "", "", false
	}
	name = fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID)
	return name, projectID, tenantID, true
}

func parseGCPTalentCompaniesCollectionPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" || parts[6] != "companies" {
		return "", false
	}
	projectID := strings.TrimSpace(parts[3])
	tenantID := strings.TrimSpace(parts[5])
	if projectID == "" || tenantID == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID), true
}

func parseGCPTalentCompanyPath(path string) (name, parent, projectID, companyID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" || parts[6] != "companies" {
		return "", "", "", "", false
	}
	projectID = strings.TrimSpace(parts[3])
	tenantID := strings.TrimSpace(parts[5])
	companyID = strings.TrimSpace(parts[7])
	if projectID == "" || tenantID == "" || companyID == "" {
		return "", "", "", "", false
	}
	parent = fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID)
	name = fmt.Sprintf("%s/companies/%s", parent, companyID)
	return name, parent, projectID, companyID, true
}

func parseGCPTalentJobsCollectionPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" || parts[6] != "jobs" {
		return "", false
	}
	projectID := strings.TrimSpace(parts[3])
	tenantID := strings.TrimSpace(parts[5])
	if projectID == "" || tenantID == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID), true
}

func parseGCPTalentJobPath(path string) (name, parent, projectID, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" || parts[6] != "jobs" {
		return "", "", "", "", false
	}
	projectID = strings.TrimSpace(parts[3])
	tenantID := strings.TrimSpace(parts[5])
	jobID = strings.TrimSpace(parts[7])
	if projectID == "" || tenantID == "" || jobID == "" {
		return "", "", "", "", false
	}
	parent = fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID)
	name = fmt.Sprintf("%s/jobs/%s", parent, jobID)
	return name, parent, projectID, jobID, true
}

func parseGCPTalentJobsActionPath(path string) (parent, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" {
		return "", "", false
	}
	projectID := strings.TrimSpace(parts[3])
	tenantID := strings.TrimSpace(parts[5])
	if projectID == "" || tenantID == "" {
		return "", "", false
	}
	name, action, found := strings.Cut(strings.TrimSpace(parts[6]), ":")
	if !found || name != "jobs" || action == "" {
		return "", "", false
	}
	parent = fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID)
	return parent, action, true
}

func parseGCPTalentCompleteQueryPath(path string) (tenant string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" {
		return "", false
	}
	projectID := strings.TrimSpace(parts[3])
	tenantPart := strings.TrimSpace(parts[5])
	tenantID, action, found := strings.Cut(tenantPart, ":")
	if !found || action != "completeQuery" || tenantID == "" || projectID == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID), true
}

func parseGCPTalentClientEventsCollectionPath(path string) (parent string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" || parts[6] != "clientEvents" {
		return "", false
	}
	projectID := strings.TrimSpace(parts[3])
	tenantID := strings.TrimSpace(parts[5])
	if projectID == "" || tenantID == "" {
		return "", false
	}
	return fmt.Sprintf("projects/%s/tenants/%s", projectID, tenantID), true
}

func parseGCPTalentOperationsCollectionPath(path string) (projectID, tenantID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" || parts[6] != "operations" {
		return "", "", false
	}
	projectID = strings.TrimSpace(parts[3])
	tenantID = strings.TrimSpace(parts[5])
	if projectID == "" || tenantID == "" {
		return "", "", false
	}
	return projectID, tenantID, true
}

func parseGCPTalentOperationPath(path string) (projectID, tenantID, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v4" || parts[2] != "projects" || parts[4] != "tenants" || parts[6] != "operations" {
		return "", "", "", false
	}
	projectID = strings.TrimSpace(parts[3])
	tenantID = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	if projectID == "" || tenantID == "" || operationID == "" {
		return "", "", "", false
	}
	return projectID, tenantID, operationID, true
}

func parseGCPTalentPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize, defaultPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPTalentInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		respondGCPTalentOutOfRange(w, path, fmt.Sprintf("pageSize cannot exceed %d", maxPageSize))
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPTalentInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPTalentBodyPagination(w http.ResponseWriter, path string, body map[string]any, pageSizeKey string, maxPageSize, defaultPageSize int) (pageSize, start int, ok bool) {
	pageSize = defaultPageSize
	if raw, exists := body[pageSizeKey]; exists {
		switch value := raw.(type) {
		case float64:
			pageSize = int(value)
		case string:
			parsed, err := parseOptionalNonNegativeInt(value)
			if err != nil {
				respondGCPTalentInvalidArgument(w, path, pageSizeKey+" must be a non-negative integer")
				return 0, 0, false
			}
			pageSize = parsed
		default:
			respondGCPTalentInvalidArgument(w, path, pageSizeKey+" must be a non-negative integer")
			return 0, 0, false
		}
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		respondGCPTalentOutOfRange(w, path, pageSizeKey+" cannot exceed "+strconv.Itoa(maxPageSize))
		return 0, 0, false
	}
	start = 0
	if raw, exists := body["pageToken"]; exists {
		switch value := raw.(type) {
		case string:
			parsed, err := parseOptionalNonNegativeInt(value)
			if err != nil {
				respondGCPTalentInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = parsed
		case float64:
			if value < 0 || value != float64(int(value)) {
				respondGCPTalentInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = int(value)
		default:
			respondGCPTalentInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPTalentList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	payload, ok := gcpTalentPaginateList(key, items, pageSize, start, path)
	if !ok {
		respondGCPTalentInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	respondJSON(w, http.StatusOK, payload)
	return true
}

func gcpTalentPaginateList(key string, items []map[string]any, pageSize, start int, path string) (map[string]any, bool) {
	_ = path
	if start > len(items) {
		return nil, false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	return map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	}, true
}

func decodeGCPTalentJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPTalentInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPTalentInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func decodeGCPTalentJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPTalentJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPTalentInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func parseGCPTalentUpdateMask(w http.ResponseWriter, r *http.Request, path string) ([]string, bool) {
	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		return nil, true
	}
	parts := splitGCPTalentCSV(mask)
	if len(parts) == 0 {
		respondGCPTalentInvalidArgument(w, path, "updateMask is invalid")
		return nil, false
	}
	return parts, true
}

func splitGCPTalentCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

type gcpTalentListJobsFilterSpec struct {
	CompanyName   string
	RequisitionID string
	Status        string
}

func parseGCPTalentListJobsFilter(filter, parent string) (gcpTalentListJobsFilterSpec, bool) {
	spec := gcpTalentListJobsFilterSpec{}
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return spec, false
	}
	parts := strings.Split(filter, "AND")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return spec, false
		}
		switch {
		case strings.HasPrefix(part, "companyName"):
			value, ok := parseGCPTalentFilterEquality(part, "companyName")
			if !ok {
				return spec, false
			}
			if !isGCPTalentCompanyName(value) || !strings.HasPrefix(value, parent+"/companies/") {
				return spec, false
			}
			spec.CompanyName = value
		case strings.HasPrefix(part, "requisitionId"):
			value, ok := parseGCPTalentFilterEquality(part, "requisitionId")
			if !ok || strings.TrimSpace(value) == "" {
				return spec, false
			}
			spec.RequisitionID = value
		case strings.HasPrefix(part, "status"):
			value, ok := parseGCPTalentFilterEquality(part, "status")
			if !ok {
				return spec, false
			}
			value = strings.ToUpper(strings.TrimSpace(value))
			if value != "OPEN" && value != "EXPIRED" && value != "ALL" {
				return spec, false
			}
			spec.Status = value
		default:
			return spec, false
		}
	}
	if spec.CompanyName == "" && spec.RequisitionID == "" {
		return spec, false
	}
	return spec, true
}

func parseGCPTalentFilterEquality(part, key string) (string, bool) {
	idx := strings.Index(part, "=")
	if idx <= 0 {
		return "", false
	}
	left := strings.TrimSpace(part[:idx])
	if left != key {
		return "", false
	}
	value := strings.TrimSpace(part[idx+1:])
	value = strings.Trim(value, `"`)
	if value == "" {
		return "", false
	}
	return value, true
}

func gcpTalentFilterJobs(items []map[string]any, filterSpec gcpTalentListJobsFilterSpec) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if filterSpec.CompanyName != "" && gcpTalentString(item, "company") != filterSpec.CompanyName {
			continue
		}
		if filterSpec.RequisitionID != "" && gcpTalentString(item, "requisitionId") != filterSpec.RequisitionID {
			continue
		}
		if filterSpec.Status != "" && filterSpec.Status != "ALL" {
			status := strings.ToUpper(strings.TrimSpace(gcpTalentString(item, "status")))
			if status == "" {
				status = "OPEN"
			}
			if status != filterSpec.Status {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func parseAndValidateGCPTalentJobBody(job map[string]any, parent string, requireName bool) (map[string]any, string, string) {
	company := strings.TrimSpace(gcpTalentString(job, "company"))
	requisitionID := strings.TrimSpace(gcpTalentString(job, "requisitionId"))
	title := strings.TrimSpace(gcpTalentString(job, "title"))
	description := strings.TrimSpace(gcpTalentString(job, "description"))
	if company == "" || requisitionID == "" || title == "" || description == "" {
		return nil, "", "job.company, job.requisitionId, job.title, and job.description are required"
	}
	if !isGCPTalentCompanyName(company) || !strings.HasPrefix(company, parent+"/companies/") {
		return nil, "", "job.company is invalid"
	}

	name := strings.TrimSpace(gcpTalentString(job, "name"))
	jobID := "job-created-1"
	if name != "" {
		if !isGCPTalentJobName(name) || !strings.HasPrefix(name, parent+"/jobs/") {
			return nil, "", "job.name is invalid"
		}
		jobID = pathBase(name)
	} else if requireName {
		return nil, "", "job.name is required"
	}
	name = parent + "/jobs/" + jobID
	parsed := gcpTalentJob(name, company, requisitionID, title, description)
	return parsed, name, ""
}

func isGCPTalentRequestMetadataValid(metadata map[string]any) bool {
	if len(metadata) == 0 {
		return false
	}
	allowMissing := false
	if raw, exists := metadata["allowMissingIds"]; exists {
		if b, ok := raw.(bool); ok {
			allowMissing = b
		}
	}
	if allowMissing {
		return true
	}
	return strings.TrimSpace(gcpTalentString(metadata, "domain")) != "" &&
		strings.TrimSpace(gcpTalentString(metadata, "sessionId")) != "" &&
		strings.TrimSpace(gcpTalentString(metadata, "userId")) != ""
}

func parseGCPTalentCompletionType(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	switch strings.ToUpper(value) {
	case "COMPLETION_TYPE_UNSPECIFIED", "JOB_TITLE", "COMPANY_NAME", "COMBINED":
		return strings.ToUpper(value), true
	case "0":
		return "", true
	case "1":
		return "JOB_TITLE", true
	case "2":
		return "COMPANY_NAME", true
	case "3":
		return "COMBINED", true
	default:
		return "", false
	}
}

func parseGCPTalentCompletionScope(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	switch strings.ToUpper(value) {
	case "COMPLETION_SCOPE_UNSPECIFIED", "TENANT", "PUBLIC":
		return strings.ToUpper(value), true
	case "0":
		return "", true
	case "1":
		return "TENANT", true
	case "2":
		return "PUBLIC", true
	default:
		return "", false
	}
}

func isGCPTalentJobEventType(raw string) bool {
	value := strings.TrimSpace(raw)
	switch value {
	case "VIEW", "IMPRESSION", "APPLICATION_START", "APPLICATION_FINISH", "APPLICATION_QUICK_SUBMISSION", "APPLICATION_REDIRECT", "APPLICATION_COMPANY_SUBMIT", "BOOKMARK", "NOTIFICATION", "HIRED", "SENT_CV", "INTERVIEW_GRANTED":
		return true
	case "1", "2", "4", "5", "6", "7", "10", "11", "12", "13", "14", "15":
		return true
	default:
		return false
	}
}

func gcpTalentTenant(name, externalID string) map[string]any {
	return map[string]any{
		"name":       name,
		"externalId": externalID,
	}
}

func gcpTalentCompany(name, displayName, externalID string) map[string]any {
	return map[string]any{
		"name":        name,
		"displayName": displayName,
		"externalId":  externalID,
		"size":        "SMALL",
		"websiteUri":  "https://example.com",
	}
}

func gcpTalentJob(name, company, requisitionID, title, description string) map[string]any {
	created := gcpTalentReferenceTime.Add(5 * time.Minute)
	updated := created.Add(15 * time.Minute)
	expire := created.Add(30 * 24 * time.Hour)
	return map[string]any{
		"name":               name,
		"company":            company,
		"requisitionId":      requisitionID,
		"title":              title,
		"description":        description,
		"addresses":          []string{"1600 Amphitheatre Parkway, Mountain View, CA, USA"},
		"languageCode":       "en-US",
		"status":             "OPEN",
		"companyDisplayName": "Stackyard Inc",
		"postingCreateTime":  created.Format(time.RFC3339),
		"postingUpdateTime":  updated.Format(time.RFC3339),
		"postingExpireTime":  expire.Format(time.RFC3339),
	}
}

func gcpTalentOperation(projectID, tenantID, operationID, responseType string, response map[string]any, resourceNames []string) map[string]any {
	if responseType == "" {
		responseType = "type.googleapis.com/google.protobuf.Empty"
	}
	name := fmt.Sprintf("projects/%s/tenants/%s/operations/%s", projectID, tenantID, operationID)
	metadata := map[string]any{
		"@type":         "type.googleapis.com/google.cloud.talent.v4.BatchOperationMetadata",
		"createTime":    gcpTalentReferenceTime.Add(10 * time.Minute).Format(time.RFC3339),
		"endTime":       gcpTalentReferenceTime.Add(11 * time.Minute).Format(time.RFC3339),
		"resourceNames": resourceNames,
	}
	responsePayload := map[string]any{
		"@type": responseType,
	}
	for key, value := range response {
		responsePayload[key] = value
	}
	return map[string]any{
		"name":     name,
		"metadata": metadata,
		"done":     true,
		"response": responsePayload,
	}
}

func gcpTalentOperationResponseType(operationID string) string {
	switch {
	case strings.HasPrefix(operationID, "batchCreateJobs"):
		return "type.googleapis.com/google.cloud.talent.v4.BatchCreateJobsResponse"
	case strings.HasPrefix(operationID, "batchUpdateJobs"):
		return "type.googleapis.com/google.cloud.talent.v4.BatchUpdateJobsResponse"
	case strings.HasPrefix(operationID, "batchDeleteJobs"):
		return "type.googleapis.com/google.cloud.talent.v4.BatchDeleteJobsResponse"
	default:
		return "type.googleapis.com/google.protobuf.Empty"
	}
}

func gcpTalentProjectTenantFromParent(parent string) (projectID, tenantID string) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 {
		return "", ""
	}
	return parts[1], parts[3]
}

func gcpTalentBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return map[string]any{}
}

func gcpTalentString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func isGCPTalentProjectID(projectID string) bool {
	return gcpTalentProjectIDRegex.MatchString(strings.TrimSpace(projectID))
}

func isGCPTalentResourceID(id string) bool {
	return gcpTalentResourceRegex.MatchString(strings.TrimSpace(id))
}

func isGCPTalentTenantName(name string) bool {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "tenants" {
		return false
	}
	return isGCPTalentProjectID(parts[1]) && isGCPTalentResourceID(parts[3])
}

func isGCPTalentCompanyName(name string) bool {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "tenants" || parts[4] != "companies" {
		return false
	}
	return isGCPTalentProjectID(parts[1]) && isGCPTalentResourceID(parts[3]) && isGCPTalentResourceID(parts[5])
}

func isGCPTalentJobName(name string) bool {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "tenants" || parts[4] != "jobs" {
		return false
	}
	return isGCPTalentProjectID(parts[1]) && isGCPTalentResourceID(parts[3]) && isGCPTalentResourceID(parts[5])
}

func respondGCPTalentInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPTalentError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPTalentFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPTalentError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPTalentOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPTalentError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPTalentNotFound(w http.ResponseWriter, path, message string) {
	respondGCPTalentError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPTalentError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_talent(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}
	path := normalizeGCPTalentPath(rawRequestPath(r))
	if !isGCPTalentPath(path, true) {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPTalentInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}
	payload := gcpTalentJob("projects/stackyard/tenants/tenant-1/jobs/job-probe-1", "projects/stackyard/tenants/tenant-1/companies/company-1", "req-probe-1", "Probe Job", "Probe description")
	payload["service"] = "talent"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}
