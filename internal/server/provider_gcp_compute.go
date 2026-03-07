package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPComputeRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPComputePath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPComputeListZones(w, r, path) {
			return true
		}
		if handleGCPComputeGetZone(w, path) {
			return true
		}
		if handleGCPComputeListNetworks(w, r, path) {
			return true
		}
		if handleGCPComputeGetNetwork(w, path) {
			return true
		}
		if handleGCPComputeListInstances(w, r, path) {
			return true
		}
		if handleGCPComputeGetInstance(w, path) {
			return true
		}
		if handleGCPComputeListZoneOperations(w, r, path) {
			return true
		}
		if handleGCPComputeGetZoneOperation(w, path) {
			return true
		}
		if handleGCPComputeGetGlobalOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPComputeInsertNetwork(w, r, path) {
			return true
		}
		if handleGCPComputeInsertInstance(w, r, path) {
			return true
		}
		if handleGCPComputeStartInstance(w, path) {
			return true
		}
		if handleGCPComputeStopInstance(w, path) {
			return true
		}
		if handleGCPComputeWaitZoneOperation(w, path) {
			return true
		}
		if handleGCPComputeWaitGlobalOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPComputeDeleteInstance(w, path) {
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

func isGCPComputePath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/compute/v1/projects/") {
		return false
	}
	if _, ok := parseGCPComputeZonesCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPComputeZonePath(path); ok {
		return true
	}
	if _, ok := parseGCPComputeNetworksCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPComputeNetworkPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPComputeInstancesCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPComputeInstancePath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPComputeInstanceActionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPComputeZoneOperationsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPComputeZoneOperationPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPComputeZoneOperationActionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPComputeGlobalOperationPath(path); ok {
		return true
	}
	_, _, _, ok := parseGCPComputeGlobalOperationActionPath(path)
	return ok
}

func handleGCPComputeListZones(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPComputeZonesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPComputePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpComputeZone(project, "us-central1-a")}
	return respondGCPComputeList(w, items, pageSize, start, path)
}

func handleGCPComputeGetZone(w http.ResponseWriter, path string) bool {
	project, zone, ok := parseGCPComputeZonePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeZone(project, zone))
	return true
}

func handleGCPComputeListNetworks(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPComputeNetworksCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPComputePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpComputeNetwork(project, "team-network")}
	return respondGCPComputeList(w, items, pageSize, start, path)
}

func handleGCPComputeGetNetwork(w http.ResponseWriter, path string) bool {
	project, network, ok := parseGCPComputeNetworkPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeNetwork(project, network))
	return true
}

func handleGCPComputeInsertNetwork(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPComputeNetworksCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPComputeJSONBody(w, r, path)
	if !valid {
		return true
	}
	name, _ := body["name"].(string)
	if strings.TrimSpace(name) == "" {
		networkResource, _ := body["networkResource"].(map[string]any)
		name, _ = networkResource["name"].(string)
	}
	if strings.TrimSpace(name) == "" {
		respondGCPComputeInvalidArgument(w, path, "network name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpComputeOperation(project, "global", "", "insert-network", "insertNetwork", fmt.Sprintf("projects/%s/global/networks/%s", project, name)))
	return true
}

func handleGCPComputeListInstances(w http.ResponseWriter, r *http.Request, path string) bool {
	project, zone, ok := parseGCPComputeInstancesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPComputePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpComputeInstance(project, zone, "team-vm")}
	return respondGCPComputeList(w, items, pageSize, start, path)
}

func handleGCPComputeGetInstance(w http.ResponseWriter, path string) bool {
	project, zone, instance, ok := parseGCPComputeInstancePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeInstance(project, zone, instance))
	return true
}

func handleGCPComputeInsertInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, zone, ok := parseGCPComputeInstancesCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPComputeJSONBody(w, r, path)
	if !valid {
		return true
	}
	name, _ := body["name"].(string)
	if strings.TrimSpace(name) == "" {
		instanceResource, _ := body["instanceResource"].(map[string]any)
		name, _ = instanceResource["name"].(string)
	}
	if strings.TrimSpace(name) == "" {
		respondGCPComputeInvalidArgument(w, path, "instance name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpComputeOperation(project, "zone", zone, "insert-instance", "insertInstance", fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, name)))
	return true
}

func handleGCPComputeStartInstance(w http.ResponseWriter, path string) bool {
	project, zone, instance, action, ok := parseGCPComputeInstanceActionPath(path)
	if !ok || action != "start" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeOperation(project, "zone", zone, "start-instance", "start", fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, instance)))
	return true
}

func handleGCPComputeStopInstance(w http.ResponseWriter, path string) bool {
	project, zone, instance, action, ok := parseGCPComputeInstanceActionPath(path)
	if !ok || action != "stop" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeOperation(project, "zone", zone, "stop-instance", "stop", fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, instance)))
	return true
}

func handleGCPComputeDeleteInstance(w http.ResponseWriter, path string) bool {
	project, zone, instance, ok := parseGCPComputeInstancePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeOperation(project, "zone", zone, "delete-instance", "delete", fmt.Sprintf("projects/%s/zones/%s/instances/%s", project, zone, instance)))
	return true
}

func handleGCPComputeListZoneOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, zone, ok := parseGCPComputeZoneOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPComputePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpComputeOperation(project, "zone", zone, "op-zone-1", "insert", fmt.Sprintf("projects/%s/zones/%s/instances/team-vm", project, zone))}
	return respondGCPComputeList(w, items, pageSize, start, path)
}

func handleGCPComputeGetZoneOperation(w http.ResponseWriter, path string) bool {
	project, zone, operation, ok := parseGCPComputeZoneOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeOperation(project, "zone", zone, operation, "insert", fmt.Sprintf("projects/%s/zones/%s/instances/team-vm", project, zone)))
	return true
}

func handleGCPComputeWaitZoneOperation(w http.ResponseWriter, path string) bool {
	project, zone, operation, action, ok := parseGCPComputeZoneOperationActionPath(path)
	if !ok || action != "wait" {
		return false
	}
	op := gcpComputeOperation(project, "zone", zone, operation, "wait", fmt.Sprintf("projects/%s/zones/%s/instances/team-vm", project, zone))
	op["status"] = "DONE"
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPComputeGetGlobalOperation(w http.ResponseWriter, path string) bool {
	project, operation, ok := parseGCPComputeGlobalOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpComputeOperation(project, "global", "", operation, "insert", fmt.Sprintf("projects/%s/global/networks/team-network", project)))
	return true
}

func handleGCPComputeWaitGlobalOperation(w http.ResponseWriter, path string) bool {
	project, operation, action, ok := parseGCPComputeGlobalOperationActionPath(path)
	if !ok || action != "wait" {
		return false
	}
	op := gcpComputeOperation(project, "global", "", operation, "wait", fmt.Sprintf("projects/%s/global/networks/team-network", project))
	op["status"] = "DONE"
	respondJSON(w, http.StatusOK, op)
	return true
}

func parseGCPComputeZonesCollectionPath(path string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" {
		return "", false
	}
	project = strings.TrimSpace(parts[4])
	if project == "" {
		return "", false
	}
	return project, true
}

func parseGCPComputeZonePath(path string) (project, zone string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[4])
	zone = strings.TrimSpace(parts[6])
	if project == "" || zone == "" {
		return "", "", false
	}
	return project, zone, true
}

func parseGCPComputeNetworksCollectionPath(path string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "global" || parts[6] != "networks" {
		return "", false
	}
	project = strings.TrimSpace(parts[4])
	if project == "" {
		return "", false
	}
	return project, true
}

func parseGCPComputeNetworkPath(path string) (project, network string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "global" || parts[6] != "networks" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[4])
	network = strings.TrimSpace(parts[7])
	if project == "" || network == "" || strings.Contains(network, ":") {
		return "", "", false
	}
	return project, network, true
}

func parseGCPComputeInstancesCollectionPath(path string) (project, zone string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" || parts[7] != "instances" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[4])
	zone = strings.TrimSpace(parts[6])
	if project == "" || zone == "" {
		return "", "", false
	}
	return project, zone, true
}

func parseGCPComputeInstancePath(path string) (project, zone, instance string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" || parts[7] != "instances" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[4])
	zone = strings.TrimSpace(parts[6])
	instance = strings.TrimSpace(parts[8])
	if project == "" || zone == "" || instance == "" || strings.Contains(instance, ":") {
		return "", "", "", false
	}
	return project, zone, instance, true
}

func parseGCPComputeInstanceActionPath(path string) (project, zone, instance, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" || parts[7] != "instances" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[4])
	zone = strings.TrimSpace(parts[6])
	instance = strings.TrimSpace(parts[8])
	action = strings.TrimSpace(parts[9])
	if project == "" || zone == "" || instance == "" || action == "" {
		return "", "", "", "", false
	}
	return project, zone, instance, action, true
}

func parseGCPComputeZoneOperationsCollectionPath(path string) (project, zone string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" || parts[7] != "operations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[4])
	zone = strings.TrimSpace(parts[6])
	if project == "" || zone == "" {
		return "", "", false
	}
	return project, zone, true
}

func parseGCPComputeZoneOperationPath(path string) (project, zone, operation string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" || parts[7] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[4])
	zone = strings.TrimSpace(parts[6])
	operation = strings.TrimSpace(parts[8])
	if project == "" || zone == "" || operation == "" || strings.Contains(operation, ":") {
		return "", "", "", false
	}
	return project, zone, operation, true
}

func parseGCPComputeZoneOperationActionPath(path string) (project, zone, operation, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "zones" || parts[7] != "operations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[4])
	zone = strings.TrimSpace(parts[6])
	operation = strings.TrimSpace(parts[8])
	action = strings.TrimSpace(parts[9])
	if project == "" || zone == "" || operation == "" || action == "" {
		return "", "", "", "", false
	}
	return project, zone, operation, action, true
}

func parseGCPComputeGlobalOperationPath(path string) (project, operation string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "global" || parts[6] != "operations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[4])
	operation = strings.TrimSpace(parts[7])
	if project == "" || operation == "" || strings.Contains(operation, ":") {
		return "", "", false
	}
	return project, operation, true
}

func parseGCPComputeGlobalOperationActionPath(path string) (project, operation, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "compute" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "global" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[4])
	operation = strings.TrimSpace(parts[7])
	action = strings.TrimSpace(parts[8])
	if project == "" || operation == "" || action == "" {
		return "", "", "", false
	}
	return project, operation, action, true
}

func parseGCPComputePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	rawPageSize := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if rawPageSize == "" {
		rawPageSize = strings.TrimSpace(r.URL.Query().Get("maxResults"))
	}
	size, err := parseOptionalNonNegativeInt(rawPageSize)
	if err != nil {
		respondGCPComputeInvalidArgument(w, path, "pageSize/maxResults must be a non-negative integer")
		return 0, 0, false
	}
	token := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if token == "" {
		return size, 0, true
	}
	start, err = parseOptionalNonNegativeInt(token)
	if err != nil {
		respondGCPComputeInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return size, start, true
}

func respondGCPComputeList(w http.ResponseWriter, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPComputeInvalidArgument(w, path, "pageToken is out of range")
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
		"items":         items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPComputeJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPComputeInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpComputeZone(project, zone string) map[string]any {
	return map[string]any{
		"name":   zone,
		"status": "UP",
		"region": fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/us-central1", project),
		"selfLink": fmt.Sprintf(
			"https://compute.googleapis.com/compute/v1/projects/%s/zones/%s",
			project,
			zone,
		),
	}
}

func gcpComputeNetwork(project, network string) map[string]any {
	return map[string]any{
		"name":                                  network,
		"autoCreateSubnetworks":                 true,
		"routingConfig":                         map[string]any{"routingMode": "REGIONAL"},
		"selfLink":                              fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%s", project, network),
		"mtu":                                   1460,
		"networkFirewallPolicyEnforcementOrder": "AFTER_CLASSIC_FIREWALL",
	}
}

func gcpComputeInstance(project, zone, instance string) map[string]any {
	return map[string]any{
		"name":   instance,
		"status": "RUNNING",
		"zone":   fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s", project, zone),
		"machineType": fmt.Sprintf(
			"https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/e2-medium",
			project,
			zone,
		),
		"selfLink": fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s", project, zone, instance),
	}
}

func gcpComputeOperation(project, scope, scopeID, operation, operationType, targetLink string) map[string]any {
	op := map[string]any{
		"name":          operation,
		"operationType": operationType,
		"status":        "PENDING",
		"targetLink":    fmt.Sprintf("https://compute.googleapis.com/compute/v1/%s", strings.TrimPrefix(targetLink, "projects/")),
		"selfLink":      fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/%ss/%s", project, scope, operation),
	}
	if scope == "zone" {
		op["zone"] = fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s", project, scopeID)
		op["selfLink"] = fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/operations/%s", project, scopeID, operation)
	}
	if scope == "global" {
		op["selfLink"] = fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/operations/%s", project, operation)
	}
	return op
}

func respondGCPComputeInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
