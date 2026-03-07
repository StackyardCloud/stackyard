package server

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPOptimizationRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_optimization(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPOptimizationPath(path) {
		return false
	}
	if shouldSkipGCPOptimizationPath(path, r) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPOptimizationPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.optimization.v1.FleetRouting/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/v1/operations/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}

	return strings.Contains(path, ":optimizeTours") ||
		strings.Contains(path, ":batchOptimizeTours") ||
		strings.Contains(path, "/operations/")
}

func shouldSkipGCPOptimizationPath(path string, r *http.Request) bool {
	if isGCPMapsRouteOptimizationClient(r.Header.Get("x-goog-api-client")) {
		return true
	}
	if strings.Contains(path, "/operations/routeopt-") {
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(path, ":optimizeTours") && !strings.Contains(path, ":batchOptimizeTours") {
		return false
	}
	if r.Body == nil {
		return false
	}

	rawBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(rawBody))

	lowerBody := strings.ToLower(string(rawBody))
	return strings.Contains(lowerBody, "routeopt") || strings.Contains(lowerBody, "routeoptimization")
}

func isGCPMapsRouteOptimizationClient(headerValue string) bool {
	lowerHeader := strings.ToLower(strings.TrimSpace(headerValue))
	if lowerHeader == "" {
		return false
	}
	idx := strings.Index(lowerHeader, "gapic/")
	if idx < 0 {
		return false
	}

	versionToken := lowerHeader[idx+len("gapic/"):]
	if fields := strings.Fields(versionToken); len(fields) > 0 {
		versionToken = fields[0]
	}
	parts := strings.Split(versionToken, ".")
	if len(parts) < 2 {
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	// Route Optimization client comes from cloud.google.com/go/maps and
	// currently advertises gapic 1.2x+, while Cloud Optimization FleetRouting
	// uses cloud.google.com/go/optimization 1.x with single-digit minors.
	return major > 1 || (major == 1 && minor >= 20)
}

func handleGCPContractProbe_optimization(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "optimization") {
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
			"name":     "projects/stackyard/locations/us-central1/optimization/sample",
			"service":  "optimization",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
