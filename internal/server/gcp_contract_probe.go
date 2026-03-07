package server

import (
	"fmt"
	"net/http"
	"strings"
)

func isGCPContractProbeRequestForService(r *http.Request, path, service string) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}
	return isGCPContractProbeServicePath(path, service)
}

func handleGCPContractProbeGeneric(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := rawRequestPath(r)
	project, location, service, ok := parseGCPContractProbeServicePath(path)
	if !ok {
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
			"name":     fmt.Sprintf("projects/%s/locations/%s/%s/sample", project, location, service),
			"service":  service,
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}

func isGCPContractProbeServicePath(path, service string) bool {
	service = strings.TrimSpace(service)
	if service == "" {
		return false
	}
	_, _, pathService, ok := parseGCPContractProbeServicePath(path)
	if !ok {
		return false
	}
	return pathService == service
}

func parseGCPContractProbeServicePath(path string) (project, location, service string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return "", "", "", false
	}
	version := strings.TrimSpace(parts[1])
	isSupportedVersion := strings.HasPrefix(version, "v1") || strings.HasPrefix(version, "v2")
	if parts[0] != "gcp" || !isSupportedVersion || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	service = strings.TrimSpace(parts[6])
	if project == "" || location == "" || service == "" {
		return "", "", "", false
	}
	return project, location, service, true
}
