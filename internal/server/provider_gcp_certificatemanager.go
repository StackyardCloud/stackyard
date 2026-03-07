package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCertificateManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if strings.HasPrefix(path, "/gcp/google.cloud.certificatemanager.v1.CertificateManager/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}
	if !isGCPCertificateManagerPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCertificateManagerListCertificates(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerGetCertificate(w, path) {
			return true
		}
		if handleGCPCertificateManagerListCertificateMaps(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerGetCertificateMap(w, path) {
			return true
		}
		if handleGCPCertificateManagerListCertificateMapEntries(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerGetCertificateMapEntry(w, path) {
			return true
		}
		if handleGCPCertificateManagerListDNSAuthorizations(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerGetDNSAuthorization(w, path) {
			return true
		}
		if handleGCPCertificateManagerListTrustConfigs(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerGetTrustConfig(w, path) {
			return true
		}
		if handleGCPCertificateManagerListIssuanceConfigs(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerGetIssuanceConfig(w, path) {
			return true
		}
		if handleGCPCertificateManagerListOperations(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCertificateManagerCreateCertificate(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerCreateCertificateMap(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerCreateCertificateMapEntry(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerCreateDNSAuthorization(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerCreateTrustConfig(w, r, path) {
			return true
		}
		if handleGCPCertificateManagerCreateIssuanceConfig(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPCertificateManagerDeleteResource(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch, http.MethodPut:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCertificateManagerPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}
	markers := []string{
		"/certificates",
		"/certificateMaps",
		"/certificateMapEntries",
		"/dnsAuthorizations",
		"/certificateIssuanceConfigs",
		"/trustConfigs",
		"/operations",
		":setIamPolicy",
		":getIamPolicy",
		":testIamPermissions",
	}
	for _, marker := range markers {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

func handleGCPCertificateManagerListCertificates(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "certificates" {
		return false
	}
	pageSize, start, valid := parseGCPCertificateManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCertificateManagerCertificate(project, location, "team-certificate")}
	return respondGCPCertificateManagerList(w, "certificates", items, pageSize, start, path)
}

func handleGCPCertificateManagerGetCertificate(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "certificates" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerCertificate(project, location, tail[1]))
	return true
}

func handleGCPCertificateManagerCreateCertificate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "certificates" {
		return false
	}
	body, valid := decodeGCPCertificateManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	certificate := gcpCertificateManagerBodyMap(body, "certificate")
	if len(certificate) == 0 {
		respondGCPCertificateManagerInvalidArgument(w, path, "certificate is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("certificateId"))
	if id == "" {
		id = "team-certificate"
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "createCertificate."+id))
	return true
}

func handleGCPCertificateManagerListCertificateMaps(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "certificateMaps" {
		return false
	}
	pageSize, start, valid := parseGCPCertificateManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCertificateManagerCertificateMap(project, location, "team-map")}
	return respondGCPCertificateManagerList(w, "certificateMaps", items, pageSize, start, path)
}

func handleGCPCertificateManagerGetCertificateMap(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "certificateMaps" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerCertificateMap(project, location, tail[1]))
	return true
}

func handleGCPCertificateManagerCreateCertificateMap(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "certificateMaps" {
		return false
	}
	body, valid := decodeGCPCertificateManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	certificateMap := gcpCertificateManagerBodyMap(body, "certificateMap")
	if len(certificateMap) == 0 {
		respondGCPCertificateManagerInvalidArgument(w, path, "certificateMap is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("certificateMapId"))
	if id == "" {
		id = "team-map"
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "createCertificateMap."+id))
	return true
}

func handleGCPCertificateManagerListCertificateMapEntries(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "certificateMaps" || strings.TrimSpace(tail[1]) == "" || tail[2] != "certificateMapEntries" {
		return false
	}
	pageSize, start, valid := parseGCPCertificateManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCertificateManagerCertificateMapEntry(project, location, tail[1], "entry-1")}
	return respondGCPCertificateManagerList(w, "certificateMapEntries", items, pageSize, start, path)
}

func handleGCPCertificateManagerGetCertificateMapEntry(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "certificateMaps" || strings.TrimSpace(tail[1]) == "" || tail[2] != "certificateMapEntries" || strings.TrimSpace(tail[3]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerCertificateMapEntry(project, location, tail[1], tail[3]))
	return true
}

func handleGCPCertificateManagerCreateCertificateMapEntry(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "certificateMaps" || strings.TrimSpace(tail[1]) == "" || tail[2] != "certificateMapEntries" {
		return false
	}
	body, valid := decodeGCPCertificateManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	entry := gcpCertificateManagerBodyMap(body, "certificateMapEntry")
	if len(entry) == 0 {
		respondGCPCertificateManagerInvalidArgument(w, path, "certificateMapEntry is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("certificateMapEntryId"))
	if id == "" {
		id = "entry-1"
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "createCertificateMapEntry."+id))
	return true
}

func handleGCPCertificateManagerListDNSAuthorizations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "dnsAuthorizations" {
		return false
	}
	pageSize, start, valid := parseGCPCertificateManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCertificateManagerDNSAuthorization(project, location, "team-dns-auth")}
	return respondGCPCertificateManagerList(w, "dnsAuthorizations", items, pageSize, start, path)
}

func handleGCPCertificateManagerGetDNSAuthorization(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "dnsAuthorizations" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerDNSAuthorization(project, location, tail[1]))
	return true
}

func handleGCPCertificateManagerCreateDNSAuthorization(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "dnsAuthorizations" {
		return false
	}
	body, valid := decodeGCPCertificateManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	dnsAuthorization := gcpCertificateManagerBodyMap(body, "dnsAuthorization")
	if len(dnsAuthorization) == 0 {
		respondGCPCertificateManagerInvalidArgument(w, path, "dnsAuthorization is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("dnsAuthorizationId"))
	if id == "" {
		id = "team-dns-auth"
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "createDNSAuthorization."+id))
	return true
}

func handleGCPCertificateManagerListTrustConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "trustConfigs" {
		return false
	}
	pageSize, start, valid := parseGCPCertificateManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCertificateManagerTrustConfig(project, location, "team-trust")}
	return respondGCPCertificateManagerList(w, "trustConfigs", items, pageSize, start, path)
}

func handleGCPCertificateManagerGetTrustConfig(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "trustConfigs" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerTrustConfig(project, location, tail[1]))
	return true
}

func handleGCPCertificateManagerCreateTrustConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "trustConfigs" {
		return false
	}
	body, valid := decodeGCPCertificateManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	trustConfig := gcpCertificateManagerBodyMap(body, "trustConfig")
	if len(trustConfig) == 0 {
		respondGCPCertificateManagerInvalidArgument(w, path, "trustConfig is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("trustConfigId"))
	if id == "" {
		id = "team-trust"
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "createTrustConfig."+id))
	return true
}

func handleGCPCertificateManagerListIssuanceConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "certificateIssuanceConfigs" {
		return false
	}
	pageSize, start, valid := parseGCPCertificateManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCertificateManagerIssuanceConfig(project, location, "team-issuance")}
	return respondGCPCertificateManagerList(w, "certificateIssuanceConfigs", items, pageSize, start, path)
}

func handleGCPCertificateManagerGetIssuanceConfig(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "certificateIssuanceConfigs" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerIssuanceConfig(project, location, tail[1]))
	return true
}

func handleGCPCertificateManagerCreateIssuanceConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "certificateIssuanceConfigs" {
		return false
	}
	body, valid := decodeGCPCertificateManagerJSONBody(w, r, path)
	if !valid {
		return true
	}
	cfg := gcpCertificateManagerBodyMap(body, "certificateIssuanceConfig")
	if len(cfg) == 0 {
		respondGCPCertificateManagerInvalidArgument(w, path, "certificateIssuanceConfig is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("certificateIssuanceConfigId"))
	if id == "" {
		id = "team-issuance"
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "createIssuanceConfig."+id))
	return true
}

func handleGCPCertificateManagerDeleteResource(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) < 2 {
		return false
	}
	switch {
	case len(tail) == 2 && (tail[0] == "certificates" || tail[0] == "certificateMaps" || tail[0] == "dnsAuthorizations" || tail[0] == "trustConfigs" || tail[0] == "certificateIssuanceConfigs"):
		respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "delete"+strings.Title(tail[0])+"."+tail[1]))
		return true
	case len(tail) == 5 && tail[0] == "certificateMaps" && tail[2] == "certificateMapEntries":
		respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, "deleteCertificateMapEntry."+tail[4]))
		return true
	default:
		return false
	}
}

func handleGCPCertificateManagerListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return false
	}
	pageSize, start, valid := parseGCPCertificateManagerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCertificateManagerOperation(project, location, "team-operation")}
	return respondGCPCertificateManagerList(w, "operations", items, pageSize, start, path)
}

func handleGCPCertificateManagerGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCertificateManagerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCertificateManagerOperation(project, location, tail[1]))
	return true
}

func parseGCPCertificateManagerLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func parseGCPCertificateManagerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCertificateManagerInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPCertificateManagerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPCertificateManagerList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCertificateManagerInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPCertificateManagerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCertificateManagerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCertificateManagerBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpCertificateManagerCertificate(project, location, certificateID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/certificates/%s", project, location, certificateID),
		"description": "Stackyard certificate",
		"managed": map[string]any{
			"domains": []any{"example.com"},
		},
	}
}

func gcpCertificateManagerCertificateMap(project, location, mapID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/certificateMaps/%s", project, location, mapID),
		"description": "Stackyard certificate map",
	}
}

func gcpCertificateManagerCertificateMapEntry(project, location, mapID, entryID string) map[string]any {
	return map[string]any{
		"name":     fmt.Sprintf("projects/%s/locations/%s/certificateMaps/%s/certificateMapEntries/%s", project, location, mapID, entryID),
		"hostname": "example.com",
		"certificates": []any{
			fmt.Sprintf("projects/%s/locations/%s/certificates/team-certificate", project, location),
		},
	}
}

func gcpCertificateManagerDNSAuthorization(project, location, id string) map[string]any {
	return map[string]any{
		"name":   fmt.Sprintf("projects/%s/locations/%s/dnsAuthorizations/%s", project, location, id),
		"domain": "example.com",
		"dnsResourceRecord": map[string]any{
			"name": "_acme-challenge.example.com.",
			"type": "CNAME",
			"data": "token.acme.invalid.",
			"ttl":  "300s",
		},
	}
}

func gcpCertificateManagerTrustConfig(project, location, id string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/trustConfigs/%s", project, location, id),
		"trustStores": []any{
			map[string]any{
				"trustAnchors": []any{},
			},
		},
	}
}

func gcpCertificateManagerIssuanceConfig(project, location, id string) map[string]any {
	return map[string]any{
		"name":                     fmt.Sprintf("projects/%s/locations/%s/certificateIssuanceConfigs/%s", project, location, id),
		"rotationWindowPercentage": 50,
		"lifetime":                 "2592000s",
	}
}

func gcpCertificateManagerOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name":     fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done":     true,
		"response": map[string]any{},
	}
}

func respondGCPCertificateManagerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
