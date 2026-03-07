package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPConfigDeliveryRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if isGCPConfigDeliveryLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPConfigDeliveryListLocations(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPConfigDeliveryPath(path, hasGCPConfigDeliveryHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPConfigDeliveryListResourceBundles(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryGetResourceBundle(w, path) {
			return true
		}
		if handleGCPConfigDeliveryListFleetPackages(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryGetFleetPackage(w, path) {
			return true
		}
		if handleGCPConfigDeliveryListReleases(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryGetRelease(w, path) {
			return true
		}
		if handleGCPConfigDeliveryListVariants(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryGetVariant(w, path) {
			return true
		}
		if handleGCPConfigDeliveryListRollouts(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryGetRollout(w, path) {
			return true
		}
		if handleGCPConfigDeliveryListOperations(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPConfigDeliveryCreateResourceBundle(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryCreateFleetPackage(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryCreateRelease(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryCreateVariant(w, r, path) {
			return true
		}
		if handleGCPConfigDeliveryRolloutAction(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPConfigDeliveryDeleteResourceBundle(w, path) {
			return true
		}
		if handleGCPConfigDeliveryDeleteFleetPackage(w, path) {
			return true
		}
		if handleGCPConfigDeliveryDeleteRelease(w, path) {
			return true
		}
		if handleGCPConfigDeliveryDeleteVariant(w, path) {
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

func isGCPConfigDeliveryPath(path string, includeOperations bool) bool {
	_, _, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	if isGCPConfigDeliveryResourceBundlesCollectionTail(tail) ||
		isGCPConfigDeliveryResourceBundleTail(tail) ||
		isGCPConfigDeliveryFleetPackagesCollectionTail(tail) ||
		isGCPConfigDeliveryFleetPackageTail(tail) ||
		isGCPConfigDeliveryReleasesCollectionTail(tail) ||
		isGCPConfigDeliveryReleaseTail(tail) ||
		isGCPConfigDeliveryVariantsCollectionTail(tail) ||
		isGCPConfigDeliveryVariantTail(tail) ||
		isGCPConfigDeliveryRolloutsCollectionTail(tail) ||
		isGCPConfigDeliveryRolloutTail(tail) ||
		isGCPConfigDeliveryRolloutActionTail(tail, "suspend") ||
		isGCPConfigDeliveryRolloutActionTail(tail, "resume") ||
		isGCPConfigDeliveryRolloutActionTail(tail, "abort") {
		return true
	}
	return includeOperations && (isGCPConfigDeliveryOperationsCollectionTail(tail) || isGCPConfigDeliveryOperationTail(tail))
}

func isGCPConfigDeliveryLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPConfigDeliveryHint(r)
}

func hasGCPConfigDeliveryHint(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")), "configdelivery") {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-configdelivery-apiv1") || strings.Contains(userAgent, "configdelivery")
}

func handleGCPConfigDeliveryListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPConfigDeliveryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpConfigDeliveryLocation(project, "us-central1"),
		gcpConfigDeliveryLocation(project, "global"),
	}
	return respondGCPConfigDeliveryList(w, "locations", items, pageSize, start, path)
}

func handleGCPConfigDeliveryGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryLocation(project, location))
	return true
}

func handleGCPConfigDeliveryListResourceBundles(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryResourceBundlesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPConfigDeliveryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpConfigDeliveryResourceBundle(project, location, "platform-bundle")}
	return respondGCPConfigDeliveryList(w, "resourceBundles", items, pageSize, start, path)
}

func handleGCPConfigDeliveryGetResourceBundle(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryResourceBundleTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryResourceBundle(project, location, tail[1]))
	return true
}

func handleGCPConfigDeliveryCreateResourceBundle(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryResourceBundlesCollectionTail(tail) {
		return false
	}
	resourceBundleID := strings.TrimSpace(r.URL.Query().Get("resourceBundleId"))
	if resourceBundleID == "" {
		respondGCPConfigDeliveryInvalidArgument(w, path, "resourceBundleId is required")
		return true
	}
	body, valid := decodeGCPConfigDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	resourceBundle := gcpConfigDeliveryBodyMap(body, "resourceBundle")
	if len(resourceBundle) == 0 {
		respondGCPConfigDeliveryInvalidArgument(w, path, "resourceBundle is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "createResourceBundle."+resourceBundleID))
	return true
}

func handleGCPConfigDeliveryDeleteResourceBundle(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryResourceBundleTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "deleteResourceBundle."+tail[1]))
	return true
}

func handleGCPConfigDeliveryListFleetPackages(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryFleetPackagesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPConfigDeliveryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpConfigDeliveryFleetPackage(project, location, "platform-package", "platform-bundle")}
	return respondGCPConfigDeliveryList(w, "fleetPackages", items, pageSize, start, path)
}

func handleGCPConfigDeliveryGetFleetPackage(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryFleetPackageTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryFleetPackage(project, location, tail[1], "platform-bundle"))
	return true
}

func handleGCPConfigDeliveryCreateFleetPackage(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryFleetPackagesCollectionTail(tail) {
		return false
	}
	fleetPackageID := strings.TrimSpace(r.URL.Query().Get("fleetPackageId"))
	if fleetPackageID == "" {
		respondGCPConfigDeliveryInvalidArgument(w, path, "fleetPackageId is required")
		return true
	}
	body, valid := decodeGCPConfigDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	fleetPackage := gcpConfigDeliveryBodyMap(body, "fleetPackage")
	if len(fleetPackage) == 0 {
		respondGCPConfigDeliveryInvalidArgument(w, path, "fleetPackage is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "createFleetPackage."+fleetPackageID))
	return true
}

func handleGCPConfigDeliveryDeleteFleetPackage(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryFleetPackageTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "deleteFleetPackage."+tail[1]))
	return true
}

func handleGCPConfigDeliveryListReleases(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryReleasesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPConfigDeliveryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpConfigDeliveryRelease(project, location, tail[1], "r-1")}
	return respondGCPConfigDeliveryList(w, "releases", items, pageSize, start, path)
}

func handleGCPConfigDeliveryGetRelease(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryReleaseTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryRelease(project, location, tail[1], tail[3]))
	return true
}

func handleGCPConfigDeliveryCreateRelease(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryReleasesCollectionTail(tail) {
		return false
	}
	releaseID := strings.TrimSpace(r.URL.Query().Get("releaseId"))
	if releaseID == "" {
		respondGCPConfigDeliveryInvalidArgument(w, path, "releaseId is required")
		return true
	}
	body, valid := decodeGCPConfigDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	release := gcpConfigDeliveryBodyMap(body, "release")
	if len(release) == 0 {
		respondGCPConfigDeliveryInvalidArgument(w, path, "release is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "createRelease."+releaseID))
	return true
}

func handleGCPConfigDeliveryDeleteRelease(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryReleaseTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "deleteRelease."+tail[3]))
	return true
}

func handleGCPConfigDeliveryListVariants(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryVariantsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPConfigDeliveryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpConfigDeliveryVariant(project, location, tail[1], tail[3], "default")}
	return respondGCPConfigDeliveryList(w, "variants", items, pageSize, start, path)
}

func handleGCPConfigDeliveryGetVariant(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryVariantTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryVariant(project, location, tail[1], tail[3], tail[5]))
	return true
}

func handleGCPConfigDeliveryCreateVariant(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryVariantsCollectionTail(tail) {
		return false
	}
	variantID := strings.TrimSpace(r.URL.Query().Get("variantId"))
	if variantID == "" {
		respondGCPConfigDeliveryInvalidArgument(w, path, "variantId is required")
		return true
	}
	body, valid := decodeGCPConfigDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	variant := gcpConfigDeliveryBodyMap(body, "variant")
	if len(variant) == 0 {
		respondGCPConfigDeliveryInvalidArgument(w, path, "variant is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "createVariant."+variantID))
	return true
}

func handleGCPConfigDeliveryDeleteVariant(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryVariantTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, "deleteVariant."+tail[5]))
	return true
}

func handleGCPConfigDeliveryListRollouts(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryRolloutsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPConfigDeliveryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpConfigDeliveryRollout(project, location, tail[1], "rollout-1", "ACTIVE")}
	return respondGCPConfigDeliveryList(w, "rollouts", items, pageSize, start, path)
}

func handleGCPConfigDeliveryGetRollout(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryRolloutTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryRollout(project, location, tail[1], tail[3], "ACTIVE"))
	return true
}

func handleGCPConfigDeliveryRolloutAction(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok {
		return false
	}
	rolloutID, action, validAction := parseGCPConfigDeliveryRolloutAction(tail)
	if !validAction {
		return false
	}
	body, valid := decodeGCPConfigDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	reason := strings.TrimSpace(gcpConfigDeliveryString(body, "reason"))
	if reason == "" {
		respondGCPConfigDeliveryInvalidArgument(w, path, "reason is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, action+"Rollout."+rolloutID))
	return true
}

func handleGCPConfigDeliveryListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPConfigDeliveryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpConfigDeliveryOperation(project, location, "rollout-1")}
	return respondGCPConfigDeliveryList(w, "operations", items, pageSize, start, path)
}

func handleGCPConfigDeliveryGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPConfigDeliveryLocationTail(path)
	if !ok || !isGCPConfigDeliveryOperationTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpConfigDeliveryOperation(project, location, tail[1]))
	return true
}

func parseGCPConfigDeliveryLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func isGCPConfigDeliveryResourceBundlesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "resourceBundles"
}

func isGCPConfigDeliveryResourceBundleTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "resourceBundles" && strings.TrimSpace(tail[1]) != ""
}

func isGCPConfigDeliveryReleasesCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "resourceBundles" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases"
}

func isGCPConfigDeliveryReleaseTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "resourceBundles" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != ""
}

func isGCPConfigDeliveryVariantsCollectionTail(tail []string) bool {
	return len(tail) == 5 && tail[0] == "resourceBundles" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != "" && tail[4] == "variants"
}

func isGCPConfigDeliveryVariantTail(tail []string) bool {
	return len(tail) == 6 && tail[0] == "resourceBundles" && strings.TrimSpace(tail[1]) != "" && tail[2] == "releases" && strings.TrimSpace(tail[3]) != "" && tail[4] == "variants" && strings.TrimSpace(tail[5]) != ""
}

func isGCPConfigDeliveryFleetPackagesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "fleetPackages"
}

func isGCPConfigDeliveryFleetPackageTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "fleetPackages" && strings.TrimSpace(tail[1]) != ""
}

func isGCPConfigDeliveryRolloutsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "fleetPackages" && strings.TrimSpace(tail[1]) != "" && tail[2] == "rollouts"
}

func isGCPConfigDeliveryRolloutTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "fleetPackages" && strings.TrimSpace(tail[1]) != "" && tail[2] == "rollouts" && strings.TrimSpace(tail[3]) != "" && !strings.Contains(tail[3], ":")
}

func isGCPConfigDeliveryRolloutActionTail(tail []string, action string) bool {
	if len(tail) != 4 || tail[0] != "fleetPackages" || strings.TrimSpace(tail[1]) == "" || tail[2] != "rollouts" {
		return false
	}
	rolloutID, parsedAction, found := strings.Cut(strings.TrimSpace(tail[3]), ":")
	return found && strings.TrimSpace(rolloutID) != "" && parsedAction == action
}

func parseGCPConfigDeliveryRolloutAction(tail []string) (rolloutID, action string, ok bool) {
	if len(tail) != 4 || tail[0] != "fleetPackages" || strings.TrimSpace(tail[1]) == "" || tail[2] != "rollouts" {
		return "", "", false
	}
	rolloutID, action, ok = strings.Cut(strings.TrimSpace(tail[3]), ":")
	if !ok {
		return "", "", false
	}
	rolloutID = strings.TrimSpace(rolloutID)
	action = strings.TrimSpace(action)
	if rolloutID == "" {
		return "", "", false
	}
	switch action {
	case "suspend", "resume", "abort":
		return rolloutID, action, true
	default:
		return "", "", false
	}
}

func isGCPConfigDeliveryOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPConfigDeliveryOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != ""
}

func parseGCPConfigDeliveryPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPConfigDeliveryInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPConfigDeliveryInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPConfigDeliveryList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPConfigDeliveryInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPConfigDeliveryJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPConfigDeliveryInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpConfigDeliveryBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpConfigDeliveryString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpConfigDeliveryLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Config Delivery " + location,
	}
}

func gcpConfigDeliveryResourceBundle(project, location, resourceBundleID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/resourceBundles/%s", project, location, resourceBundleID),
		"description": "Stackyard sample resource bundle",
		"uid":         "resource-bundle-" + resourceBundleID,
	}
}

func gcpConfigDeliveryFleetPackage(project, location, fleetPackageID, resourceBundleID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/fleetPackages/%s", project, location, fleetPackageID),
		"resourceBundleSelector": map[string]any{
			"resourceBundle": map[string]any{
				"name": fmt.Sprintf("projects/%s/locations/%s/resourceBundles/%s", project, location, resourceBundleID),
				"tag":  "v1.0.0",
			},
		},
		"variantSelector": map[string]any{
			"variantNameTemplate": "default",
		},
	}
}

func gcpConfigDeliveryRelease(project, location, resourceBundleID, releaseID string) map[string]any {
	return map[string]any{
		"name":    fmt.Sprintf("projects/%s/locations/%s/resourceBundles/%s/releases/%s", project, location, resourceBundleID, releaseID),
		"version": "v1.0.0",
	}
}

func gcpConfigDeliveryVariant(project, location, resourceBundleID, releaseID, variantID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/resourceBundles/%s/releases/%s/variants/%s", project, location, resourceBundleID, releaseID, variantID),
		"resources": []string{
			"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sample\n",
		},
	}
}

func gcpConfigDeliveryRollout(project, location, fleetPackageID, rolloutID, state string) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/fleetPackages/%s/rollouts/%s", project, location, fleetPackageID, rolloutID),
		"state": state,
	}
}

func gcpConfigDeliveryOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": false,
	}
}

func respondGCPConfigDeliveryInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
