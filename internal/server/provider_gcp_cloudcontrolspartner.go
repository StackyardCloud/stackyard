package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudControlsPartnerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if strings.HasPrefix(path, "/gcp/google.cloud.cloudcontrolspartner.v1.") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}
	if !isGCPCloudControlsPartnerPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCloudControlsPartnerListCustomers(w, r, path) {
			return true
		}
		if handleGCPCloudControlsPartnerGetCustomer(w, path) {
			return true
		}
		if handleGCPCloudControlsPartnerListWorkloads(w, r, path) {
			return true
		}
		if handleGCPCloudControlsPartnerGetWorkload(w, path) {
			return true
		}
		if handleGCPCloudControlsPartnerGetEkmConnections(w, path) {
			return true
		}
		if handleGCPCloudControlsPartnerGetPartnerPermissions(w, path) {
			return true
		}
		if handleGCPCloudControlsPartnerListAccessApprovalRequests(w, r, path) {
			return true
		}
		if handleGCPCloudControlsPartnerGetPartner(w, path) {
			return true
		}
		if handleGCPCloudControlsPartnerListViolations(w, r, path) {
			return true
		}
		if handleGCPCloudControlsPartnerGetViolation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCloudControlsPartnerCreateCustomer(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPCloudControlsPartnerUpdateCustomer(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPCloudControlsPartnerDeleteCustomer(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCloudControlsPartnerPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/organizations/") || !strings.Contains(path, "/locations/") {
		return false
	}
	markers := []string{
		"/customers",
		"/workloads",
		"/accessApprovalRequests",
		"/violations",
		"/ekmConnections",
		"/partnerPermissions",
		"/partner",
	}
	for _, marker := range markers {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

func handleGCPCloudControlsPartnerListCustomers(w http.ResponseWriter, r *http.Request, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "customers" {
		return false
	}
	pageSize, start, valid := parseGCPCloudControlsPartnerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudControlsPartnerCustomer(org, location, "team-customer")}
	return respondGCPCloudControlsPartnerList(w, "customers", items, pageSize, start, path)
}

func handleGCPCloudControlsPartnerGetCustomer(w http.ResponseWriter, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudControlsPartnerCustomer(org, location, tail[1]))
	return true
}

func handleGCPCloudControlsPartnerCreateCustomer(w http.ResponseWriter, r *http.Request, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "customers" {
		return false
	}
	body, valid := decodeGCPCloudControlsPartnerJSONBody(w, r, path)
	if !valid {
		return true
	}
	customer := gcpCloudControlsPartnerBodyMap(body, "customer")
	if len(customer) == 0 {
		respondGCPCloudControlsPartnerInvalidArgument(w, path, "customer is required")
		return true
	}
	customerID := strings.TrimSpace(r.URL.Query().Get("customerId"))
	if customerID == "" {
		customerID = "team-customer"
	}
	respondJSON(w, http.StatusOK, gcpCloudControlsPartnerOperation(org, location, "createCustomer."+customerID))
	return true
}

func handleGCPCloudControlsPartnerUpdateCustomer(w http.ResponseWriter, r *http.Request, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	body, valid := decodeGCPCloudControlsPartnerJSONBody(w, r, path)
	if !valid {
		return true
	}
	customer := gcpCloudControlsPartnerBodyMap(body, "customer")
	if len(customer) == 0 {
		respondGCPCloudControlsPartnerInvalidArgument(w, path, "customer is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCloudControlsPartnerOperation(org, location, "updateCustomer."+tail[1]))
	return true
}

func handleGCPCloudControlsPartnerDeleteCustomer(w http.ResponseWriter, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudControlsPartnerOperation(org, location, "deleteCustomer."+tail[1]))
	return true
}

func handleGCPCloudControlsPartnerListWorkloads(w http.ResponseWriter, r *http.Request, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workloads" {
		return false
	}
	pageSize, start, valid := parseGCPCloudControlsPartnerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudControlsPartnerWorkload(org, location, tail[1], "team-workload")}
	return respondGCPCloudControlsPartnerList(w, "workloads", items, pageSize, start, path)
}

func handleGCPCloudControlsPartnerGetWorkload(w http.ResponseWriter, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workloads" || strings.TrimSpace(tail[3]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudControlsPartnerWorkload(org, location, tail[1], tail[3]))
	return true
}

func handleGCPCloudControlsPartnerGetEkmConnections(w http.ResponseWriter, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workloads" || strings.TrimSpace(tail[3]) == "" || tail[4] != "ekmConnections" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("organizations/%s/locations/%s/customers/%s/workloads/%s/ekmConnections", org, location, tail[1], tail[3]),
		"ekmConnections": []any{
			map[string]any{
				"name":  "projects/stackyard/locations/us-central1/keyRings/team-ring/cryptoKeys/team-key",
				"state": "ACTIVE",
			},
		},
	})
	return true
}

func handleGCPCloudControlsPartnerGetPartnerPermissions(w http.ResponseWriter, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workloads" || strings.TrimSpace(tail[3]) == "" || tail[4] != "partnerPermissions" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("organizations/%s/locations/%s/customers/%s/workloads/%s/partnerPermissions", org, location, tail[1], tail[3]),
		"partnerPermissions": []any{
			map[string]any{"permission": "cloudcontrolspartner.workloads.get"},
		},
	})
	return true
}

func handleGCPCloudControlsPartnerListAccessApprovalRequests(w http.ResponseWriter, r *http.Request, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workloads" || strings.TrimSpace(tail[3]) == "" || tail[4] != "accessApprovalRequests" {
		return false
	}
	pageSize, start, valid := parseGCPCloudControlsPartnerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"name":  fmt.Sprintf("organizations/%s/locations/%s/customers/%s/workloads/%s/accessApprovalRequests/request-1", org, location, tail[1], tail[3]),
			"state": "PENDING",
		},
	}
	return respondGCPCloudControlsPartnerList(w, "accessApprovalRequests", items, pageSize, start, path)
}

func handleGCPCloudControlsPartnerGetPartner(w http.ResponseWriter, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "partner" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":           fmt.Sprintf("organizations/%s/locations/%s/partner", org, location),
		"partnerProject": "projects/stackyard-partner",
	})
	return true
}

func handleGCPCloudControlsPartnerListViolations(w http.ResponseWriter, r *http.Request, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workloads" || strings.TrimSpace(tail[3]) == "" || tail[4] != "violations" {
		return false
	}
	pageSize, start, valid := parseGCPCloudControlsPartnerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudControlsPartnerViolation(org, location, tail[1], tail[3], "violation-1")}
	return respondGCPCloudControlsPartnerList(w, "violations", items, pageSize, start, path)
}

func handleGCPCloudControlsPartnerGetViolation(w http.ResponseWriter, path string) bool {
	org, location, tail, ok := parseGCPCloudControlsPartnerLocationTail(path)
	if !ok || len(tail) != 6 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workloads" || strings.TrimSpace(tail[3]) == "" || tail[4] != "violations" || strings.TrimSpace(tail[5]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudControlsPartnerViolation(org, location, tail[1], tail[3], tail[5]))
	return true
}

func parseGCPCloudControlsPartnerLocationTail(path string) (org, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "organizations" || parts[4] != "locations" {
		return "", "", nil, false
	}
	org = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if org == "" || location == "" {
		return "", "", nil, false
	}
	return org, location, parts[6:], true
}

func parseGCPCloudControlsPartnerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCloudControlsPartnerInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPCloudControlsPartnerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPCloudControlsPartnerList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCloudControlsPartnerInvalidArgument(w, path, "pageToken is out of range")
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
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPCloudControlsPartnerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCloudControlsPartnerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCloudControlsPartnerBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpCloudControlsPartnerCustomer(org, location, customerID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("organizations/%s/locations/%s/customers/%s", org, location, customerID),
		"displayName": "Team Customer",
	}
}

func gcpCloudControlsPartnerWorkload(org, location, customerID, workloadID string) map[string]any {
	return map[string]any{
		"name":         fmt.Sprintf("organizations/%s/locations/%s/customers/%s/workloads/%s", org, location, customerID, workloadID),
		"displayName":  "Team Workload",
		"workloadType": "PRODUCTION",
	}
}

func gcpCloudControlsPartnerViolation(org, location, customerID, workloadID, violationID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("organizations/%s/locations/%s/customers/%s/workloads/%s/violations/%s", org, location, customerID, workloadID, violationID),
		"description": "Sample violation",
		"state":       "OPEN",
	}
}

func gcpCloudControlsPartnerOperation(org, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("organizations/%s/locations/%s/operations/%s", org, location, operationID),
		"done": true,
	}
}

func respondGCPCloudControlsPartnerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
