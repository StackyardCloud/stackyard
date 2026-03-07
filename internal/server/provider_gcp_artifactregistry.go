package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPArtifactRegistryRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPArtifactRegistryPath(path, hasGCPArtifactRegistryHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPArtifactRegistryListRepositories(w, r, path) {
			return true
		}
		if handleGCPArtifactRegistryGetRepository(w, path) {
			return true
		}
		if handleGCPArtifactRegistryListPackages(w, r, path) {
			return true
		}
		if handleGCPArtifactRegistryGetPackage(w, path) {
			return true
		}
		if handleGCPArtifactRegistryListVersions(w, r, path) {
			return true
		}
		if handleGCPArtifactRegistryGetVersion(w, path) {
			return true
		}
		if handleGCPArtifactRegistryListTags(w, r, path) {
			return true
		}
		if handleGCPArtifactRegistryGetTag(w, path) {
			return true
		}
		if handleGCPArtifactRegistryListDockerImages(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPArtifactRegistryCreateRepository(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPArtifactRegistryDeleteRepository(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPArtifactRegistryHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "artifactregistry", "artifact-registry", "artifact_registry", "artifactregistry-apiv1", "artifactregistry_apiv1":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-artifactregistry-apiv1")
}

func isGCPArtifactRegistryPath(path string, includeHint bool) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if strings.Contains(path, "/projectSettings") || strings.Contains(path, "/vpcscConfig") {
		return true
	}
	if _, _, ok := parseGCPArtifactRegistryRepositoriesCollectionPath(path); ok {
		return includeHint
	}
	if _, _, _, ok := parseGCPArtifactRegistryRepositoryPath(path); ok {
		return includeHint
	}
	if _, _, _, ok := parseGCPArtifactRegistryPackagesCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPArtifactRegistryPackagePath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPArtifactRegistryVersionsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPArtifactRegistryVersionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPArtifactRegistryTagsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPArtifactRegistryTagPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPArtifactRegistryDockerImagesCollectionPath(path); ok {
		return true
	}

	return strings.Contains(path, "/locations/") &&
		(strings.Contains(path, ":import") ||
			strings.Contains(path, ":batchDelete") ||
			strings.Contains(path, ":exportArtifact") ||
			(strings.Contains(path, "/repositories/") &&
				(strings.Contains(path, "/files") ||
					strings.Contains(path, "/mavenArtifacts") ||
					strings.Contains(path, "/npmPackages") ||
					strings.Contains(path, "/pythonPackages") ||
					strings.Contains(path, "/attachments") ||
					strings.Contains(path, "/rules"))))
}

func handleGCPArtifactRegistryListRepositories(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPArtifactRegistryRepositoriesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPArtifactRegistryPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpArtifactRegistryRepository(project, location, "team-repo"),
	}
	return respondGCPArtifactRegistryList(w, "repositories", items, pageSize, start, path)
}

func handleGCPArtifactRegistryGetRepository(w http.ResponseWriter, path string) bool {
	project, location, repositoryID, ok := parseGCPArtifactRegistryRepositoryPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpArtifactRegistryRepository(project, location, repositoryID))
	return true
}

func handleGCPArtifactRegistryCreateRepository(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPArtifactRegistryRepositoriesCollectionPath(path)
	if !ok {
		return false
	}

	repositoryID := strings.TrimSpace(r.URL.Query().Get("repositoryId"))
	if repositoryID == "" {
		respondGCPArtifactRegistryInvalidArgument(w, path, "repositoryId is required")
		return true
	}

	body, valid := decodeGCPArtifactRegistryJSONBody(w, r, path)
	if !valid {
		return true
	}
	repository, _ := body["repository"].(map[string]any)
	if len(repository) == 0 {
		repository = body
	}
	if len(repository) == 0 {
		respondGCPArtifactRegistryInvalidArgument(w, path, "repository is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpArtifactRegistryOperation(
		fmt.Sprintf("operations/artifactregistry.createRepository.%s.%s.%s", project, location, repositoryID),
	))
	return true
}

func handleGCPArtifactRegistryDeleteRepository(w http.ResponseWriter, path string) bool {
	project, location, repositoryID, ok := parseGCPArtifactRegistryRepositoryPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpArtifactRegistryOperation(
		fmt.Sprintf("operations/artifactregistry.deleteRepository.%s.%s.%s", project, location, repositoryID),
	))
	return true
}

func handleGCPArtifactRegistryListPackages(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repositoryID, ok := parseGCPArtifactRegistryPackagesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPArtifactRegistryPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpArtifactRegistryPackage(project, location, repositoryID, "orders"),
	}
	return respondGCPArtifactRegistryList(w, "packages", items, pageSize, start, path)
}

func handleGCPArtifactRegistryGetPackage(w http.ResponseWriter, path string) bool {
	project, location, repositoryID, packageID, ok := parseGCPArtifactRegistryPackagePath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpArtifactRegistryPackage(project, location, repositoryID, packageID))
	return true
}

func handleGCPArtifactRegistryListVersions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repositoryID, packageID, ok := parseGCPArtifactRegistryVersionsCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPArtifactRegistryPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpArtifactRegistryVersion(project, location, repositoryID, packageID, "1.0.0"),
	}
	return respondGCPArtifactRegistryList(w, "versions", items, pageSize, start, path)
}

func handleGCPArtifactRegistryGetVersion(w http.ResponseWriter, path string) bool {
	project, location, repositoryID, packageID, versionID, ok := parseGCPArtifactRegistryVersionPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpArtifactRegistryVersion(project, location, repositoryID, packageID, versionID))
	return true
}

func handleGCPArtifactRegistryListTags(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repositoryID, packageID, ok := parseGCPArtifactRegistryTagsCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPArtifactRegistryPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpArtifactRegistryTag(project, location, repositoryID, packageID, "latest", "1.0.0"),
	}
	return respondGCPArtifactRegistryList(w, "tags", items, pageSize, start, path)
}

func handleGCPArtifactRegistryGetTag(w http.ResponseWriter, path string) bool {
	project, location, repositoryID, packageID, tagID, ok := parseGCPArtifactRegistryTagPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpArtifactRegistryTag(project, location, repositoryID, packageID, tagID, "1.0.0"))
	return true
}

func handleGCPArtifactRegistryListDockerImages(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, repositoryID, ok := parseGCPArtifactRegistryDockerImagesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPArtifactRegistryPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpArtifactRegistryDockerImage(project, location, repositoryID, "sha256:1234567890abcdef", "latest"),
	}
	return respondGCPArtifactRegistryList(w, "dockerImages", items, pageSize, start, path)
}

func decodeGCPArtifactRegistryJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPArtifactRegistryInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPArtifactRegistryRepositoriesCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPArtifactRegistryRepositoryPath(path string) (project, location, repositoryID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || repositoryID == "" || strings.Contains(repositoryID, ":") {
		return "", "", "", false
	}
	return project, location, repositoryID, true
}

func parseGCPArtifactRegistryPackagesCollectionPath(path string) (project, location, repositoryID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" || parts[8] != "packages" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || repositoryID == "" {
		return "", "", "", false
	}
	return project, location, repositoryID, true
}

func parseGCPArtifactRegistryPackagePath(path string) (project, location, repositoryID, packageID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" || parts[8] != "packages" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	packageID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || repositoryID == "" || packageID == "" {
		return "", "", "", "", false
	}
	return project, location, repositoryID, packageID, true
}

func parseGCPArtifactRegistryVersionsCollectionPath(path string) (project, location, repositoryID, packageID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 11 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" || parts[8] != "packages" || parts[10] != "versions" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	packageID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || repositoryID == "" || packageID == "" {
		return "", "", "", "", false
	}
	return project, location, repositoryID, packageID, true
}

func parseGCPArtifactRegistryVersionPath(path string) (project, location, repositoryID, packageID, versionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 12 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" || parts[8] != "packages" || parts[10] != "versions" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	packageID = strings.TrimSpace(parts[9])
	versionID = strings.TrimSpace(parts[11])
	if project == "" || location == "" || repositoryID == "" || packageID == "" || versionID == "" {
		return "", "", "", "", "", false
	}
	return project, location, repositoryID, packageID, versionID, true
}

func parseGCPArtifactRegistryTagsCollectionPath(path string) (project, location, repositoryID, packageID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 11 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" || parts[8] != "packages" || parts[10] != "tags" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	packageID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || repositoryID == "" || packageID == "" {
		return "", "", "", "", false
	}
	return project, location, repositoryID, packageID, true
}

func parseGCPArtifactRegistryTagPath(path string) (project, location, repositoryID, packageID, tagID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 12 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" || parts[8] != "packages" || parts[10] != "tags" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	packageID = strings.TrimSpace(parts[9])
	tagID = strings.TrimSpace(parts[11])
	if project == "" || location == "" || repositoryID == "" || packageID == "" || tagID == "" {
		return "", "", "", "", "", false
	}
	return project, location, repositoryID, packageID, tagID, true
}

func parseGCPArtifactRegistryDockerImagesCollectionPath(path string) (project, location, repositoryID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "repositories" || parts[8] != "dockerImages" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	repositoryID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || repositoryID == "" {
		return "", "", "", false
	}
	return project, location, repositoryID, true
}

func parseGCPArtifactRegistryPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPArtifactRegistryInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}

	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}

	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPArtifactRegistryInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPArtifactRegistryList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPArtifactRegistryInvalidArgument(w, path, "pageToken is out of range")
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

func gcpArtifactRegistryRepository(project, location, repositoryID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/repositories/%s", project, location, repositoryID),
		"format":      "DOCKER",
		"description": "Team repository",
	}
}

func gcpArtifactRegistryPackage(project, location, repositoryID, packageID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s", project, location, repositoryID, packageID),
		"displayName": packageID,
	}
}

func gcpArtifactRegistryVersion(project, location, repositoryID, packageID, versionID string) map[string]any {
	versionName := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s/versions/%s", project, location, repositoryID, packageID, versionID)
	return map[string]any{
		"name":        versionName,
		"description": "stackyard artifact version",
	}
}

func gcpArtifactRegistryTag(project, location, repositoryID, packageID, tagID, versionID string) map[string]any {
	return map[string]any{
		"name":    fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s/tags/%s", project, location, repositoryID, packageID, tagID),
		"version": fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s/versions/%s", project, location, repositoryID, packageID, versionID),
	}
}

func gcpArtifactRegistryDockerImage(project, location, repositoryID, digest, tag string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/repositories/%s/dockerImages/%s", project, location, repositoryID, digest),
		"uri":  fmt.Sprintf("%s-docker.pkg.dev/%s/%s@%s", location, project, repositoryID, digest),
		"tags": []any{tag},
	}
}

func gcpArtifactRegistryOperation(name string) map[string]any {
	return map[string]any{
		"name": name,
		"done": true,
	}
}

func respondGCPArtifactRegistryInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
