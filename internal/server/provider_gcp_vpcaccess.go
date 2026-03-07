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

const gcpVPCAccessGRPCPathPrefix = "/gcp/google.cloud.vpcaccess.v1.VpcAccessService/"

var gcpVPCAccessReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPVPCAccessRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_vpcaccess(w, r) {
		return true
	}

	path := normalizeGCPVPCAccessPath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpVPCAccessGRPCPathPrefix) {
		return handleGCPVPCAccessGRPCBridge(w, r, path)
	}

	if isGCPVPCAccessLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVPCAccessListLocations(w, r, path) {
			return true
		}
		if handleGCPVPCAccessGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPVPCAccessRESTPath(path, hasGCPVPCAccessHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPVPCAccessListOperations(w, r, path) {
			return true
		}
		if handleGCPVPCAccessGetOperation(w, path) {
			return true
		}
		if handleGCPVPCAccessListConnectors(w, r, path) {
			return true
		}
		if handleGCPVPCAccessGetConnector(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPVPCAccessCancelOperation(w, path) {
			return true
		}
		if handleGCPVPCAccessCreateConnector(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPVPCAccessDeleteOperation(w, path) {
			return true
		}
		if handleGCPVPCAccessDeleteConnector(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPVPCAccessPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVPCAccessHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "vpcaccess",
		"vpcaccess-apiv1",
		"vpcaccess_apiv1",
		"vpc-access",
		"vpc_access",
		"serverless-vpc-access",
		"serverless_vpc_access",
		"gcp-serverless-vpc-access",
		"gcp-vpcaccess",
		"gcp-vpc-access":
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-vpcaccess-apiv1") || strings.Contains(ua, "cloud.google.com/go/vpcaccess")
}

func isGCPVPCAccessLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVPCAccessHint(r) {
		return false
	}
	_, _, _, ok := parseGCPVPCAccessProjectLocationsPath(path)
	return ok
}

func isGCPVPCAccessRESTPath(path string, includeHint bool) bool {
	if _, _, _, ok := parseGCPVPCAccessProjectLocationsPath(path); ok {
		return includeHint
	}

	project, location, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok {
		return false
	}
	if project == "" || location == "" {
		return false
	}
	if len(tail) == 0 {
		return includeHint
	}
	if isGCPVPCAccessConnectorsCollectionTail(tail) || isGCPVPCAccessConnectorTail(tail) || isGCPVPCAccessOperationsTail(tail) {
		return true
	}
	return includeHint
}

func handleGCPVPCAccessListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPVPCAccessProjectLocationsPath(path)
	if !ok || !list {
		return false
	}

	pageSize, start, valid := parseGCPVPCAccessPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpVPCAccessLocation(project, "us-central1"),
		gcpVPCAccessLocation(project, "global"),
	}
	return respondGCPVPCAccessList(w, "locations", items, pageSize, start, path)
}

func handleGCPVPCAccessGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPVPCAccessProjectLocationsPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVPCAccessLocation(project, location))
	return true
}

func handleGCPVPCAccessListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessOperationsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPVPCAccessPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpVPCAccessOperation(project, location, "vpcaccess-op-1", "google.cloud.vpcaccess.v1.VpcAccessService.CreateConnector", fmt.Sprintf("projects/%s/locations/%s/connectors/connector-1", project, location), false),
		gcpVPCAccessOperation(project, location, "vpcaccess-op-2", "google.cloud.vpcaccess.v1.VpcAccessService.DeleteConnector", fmt.Sprintf("projects/%s/locations/%s/connectors/connector-2", project, location), true),
	}
	return respondGCPVPCAccessList(w, "operations", items, pageSize, start, path)
}

func handleGCPVPCAccessGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessOperationResourceTail(tail) {
		return false
	}
	opID := strings.TrimSpace(tail[1])
	if isGCPVPCAccessMissingID(opID) {
		respondGCPVPCAccessNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVPCAccessOperation(
		project,
		location,
		opID,
		"google.cloud.vpcaccess.v1.VpcAccessService.CreateConnector",
		fmt.Sprintf("projects/%s/locations/%s/connectors/connector-1", project, location),
		strings.Contains(strings.ToLower(opID), "done"),
	))
	return true
}

func handleGCPVPCAccessCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessOperationActionTail(tail, "cancel") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVPCAccessDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessOperationResourceTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVPCAccessListConnectors(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessConnectorsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPVPCAccessPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpVPCAccessConnector(project, location, "connector-1"),
		gcpVPCAccessConnector(project, location, "connector-2"),
	}
	return respondGCPVPCAccessList(w, "connectors", items, pageSize, start, path)
}

func handleGCPVPCAccessGetConnector(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessConnectorTail(tail) {
		return false
	}

	connectorID := strings.TrimSpace(tail[1])
	if isGCPVPCAccessMissingID(connectorID) {
		respondGCPVPCAccessNotFound(w, path, "connector not found")
		return true
	}

	respondJSON(w, http.StatusOK, gcpVPCAccessConnector(project, location, connectorID))
	return true
}

func handleGCPVPCAccessCreateConnector(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessConnectorsCollectionTail(tail) {
		return false
	}

	connectorID := strings.TrimSpace(r.URL.Query().Get("connectorId"))
	if connectorID == "" {
		respondGCPVPCAccessInvalidArgument(w, path, "connectorId is required")
		return true
	}
	if strings.Contains(strings.ToLower(connectorID), "existing") {
		respondGCPVPCAccessAlreadyExists(w, path, "connector already exists")
		return true
	}

	body, valid := decodeGCPVPCAccessJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	connector := gcpVPCAccessConnectorFromBody(body)
	if len(connector) == 0 {
		respondGCPVPCAccessInvalidArgument(w, path, "connector is required")
		return true
	}

	if !validateGCPVPCAccessConnector(w, path, connector) {
		return true
	}

	expectedName := fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, location, connectorID)
	if providedName := strings.TrimSpace(gcpVPCAccessString(connector, "name")); providedName != "" && providedName != expectedName {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.name must match connectorId and parent")
		return true
	}

	respondJSON(w, http.StatusOK, gcpVPCAccessOperation(project, location, "createConnector."+connectorID, "google.cloud.vpcaccess.v1.VpcAccessService.CreateConnector", expectedName, false))
	return true
}

func handleGCPVPCAccessDeleteConnector(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVPCAccessLocationTail(path)
	if !ok || !isGCPVPCAccessConnectorTail(tail) {
		return false
	}
	connectorID := strings.TrimSpace(tail[1])
	if isGCPVPCAccessMissingID(connectorID) {
		respondGCPVPCAccessNotFound(w, path, "connector not found")
		return true
	}
	if strings.Contains(strings.ToLower(connectorID), "in-use") {
		respondGCPVPCAccessFailedPrecondition(w, path, "connector is in use")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVPCAccessOperation(
		project,
		location,
		"deleteConnector."+connectorID,
		"google.cloud.vpcaccess.v1.VpcAccessService.DeleteConnector",
		fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, location, connectorID),
		false,
	))
	return true
}

func handleGCPVPCAccessGRPCBridge(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	method := strings.TrimPrefix(path, gcpVPCAccessGRPCPathPrefix)
	if method == "" || strings.Contains(method, "/") {
		return false
	}

	body, ok := decodeGCPVPCAccessJSONBodyOptional(w, r, path)
	if !ok {
		return true
	}

	switch method {
	case "ListConnectors":
		parent := strings.TrimSpace(gcpVPCAccessString(body, "parent"))
		project, location, tail, parsed := parseGCPVPCAccessResourceName(parent)
		if !parsed || len(tail) != 0 {
			respondGCPVPCAccessInvalidArgument(w, path, "parent is required")
			return true
		}

		pageSize, start, valid := parseGCPVPCAccessBridgePagination(w, body, path)
		if !valid {
			return true
		}
		items := []map[string]any{
			gcpVPCAccessConnector(project, location, "connector-1"),
			gcpVPCAccessConnector(project, location, "connector-2"),
		}
		respondGCPVPCAccessList(w, "connectors", items, pageSize, start, path)
		return true
	case "GetConnector":
		name := strings.TrimSpace(gcpVPCAccessString(body, "name"))
		project, location, connectorID, parsed := parseGCPVPCAccessConnectorName(name)
		if !parsed {
			respondGCPVPCAccessInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPVPCAccessMissingID(connectorID) {
			respondGCPVPCAccessNotFound(w, path, "connector not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVPCAccessConnector(project, location, connectorID))
		return true
	case "CreateConnector":
		parent := strings.TrimSpace(gcpVPCAccessString(body, "parent"))
		project, location, tail, parsed := parseGCPVPCAccessResourceName(parent)
		if !parsed || len(tail) != 0 {
			respondGCPVPCAccessInvalidArgument(w, path, "parent is required")
			return true
		}
		connectorID := strings.TrimSpace(gcpVPCAccessString(body, "connectorId"))
		if connectorID == "" {
			respondGCPVPCAccessInvalidArgument(w, path, "connectorId is required")
			return true
		}
		rawConnector, exists := body["connector"]
		if !exists {
			respondGCPVPCAccessInvalidArgument(w, path, "connector is required")
			return true
		}
		connector, ok := rawConnector.(map[string]any)
		if !ok {
			respondGCPVPCAccessInvalidArgument(w, path, "connector must be an object")
			return true
		}
		if strings.Contains(strings.ToLower(connectorID), "existing") {
			respondGCPVPCAccessAlreadyExists(w, path, "connector already exists")
			return true
		}
		if !validateGCPVPCAccessConnector(w, path, connector) {
			return true
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, location, connectorID)
		if providedName := strings.TrimSpace(gcpVPCAccessString(connector, "name")); providedName != "" && providedName != expectedName {
			respondGCPVPCAccessInvalidArgument(w, path, "connector.name must match connectorId and parent")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVPCAccessOperation(project, location, "createConnector."+connectorID, "google.cloud.vpcaccess.v1.VpcAccessService.CreateConnector", expectedName, false))
		return true
	case "DeleteConnector":
		name := strings.TrimSpace(gcpVPCAccessString(body, "name"))
		project, location, connectorID, parsed := parseGCPVPCAccessConnectorName(name)
		if !parsed {
			respondGCPVPCAccessInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPVPCAccessMissingID(connectorID) {
			respondGCPVPCAccessNotFound(w, path, "connector not found")
			return true
		}
		if strings.Contains(strings.ToLower(connectorID), "in-use") {
			respondGCPVPCAccessFailedPrecondition(w, path, "connector is in use")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVPCAccessOperation(project, location, "deleteConnector."+connectorID, "google.cloud.vpcaccess.v1.VpcAccessService.DeleteConnector", name, false))
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func parseGCPVPCAccessProjectLocationsPath(path string) (project, location string, list bool, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 5 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		if project == "" {
			return "", "", false, false
		}
		return project, "", true, true
	}
	if len(parts) == 6 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		location = strings.TrimSpace(parts[5])
		if project == "" || location == "" {
			return "", "", false, false
		}
		return project, location, false, true
	}
	return "", "", false, false
}

func parseGCPVPCAccessLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func parseGCPVPCAccessResourceName(name string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) < 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[4:], true
}

func parseGCPVPCAccessConnectorName(name string) (project, location, connectorID string, ok bool) {
	project, location, tail, parsed := parseGCPVPCAccessResourceName(name)
	if !parsed || len(tail) != 2 || tail[0] != "connectors" {
		return "", "", "", false
	}
	connectorID = strings.TrimSpace(tail[1])
	if connectorID == "" {
		return "", "", "", false
	}
	return project, location, connectorID, true
}

func isGCPVPCAccessConnectorsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "connectors"
}

func isGCPVPCAccessConnectorTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "connectors" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPVPCAccessOperationsTail(tail []string) bool {
	return isGCPVPCAccessOperationsCollectionTail(tail) || isGCPVPCAccessOperationResourceTail(tail) || isGCPVPCAccessOperationActionTail(tail, "cancel")
}

func isGCPVPCAccessOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPVPCAccessOperationResourceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPVPCAccessOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	id, parsedAction, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return found && strings.TrimSpace(id) != "" && parsedAction == action
}

func parseGCPVPCAccessPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVPCAccessInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPVPCAccessOutOfRange(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVPCAccessInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func parseGCPVPCAccessBridgePagination(w http.ResponseWriter, body map[string]any, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw, exists := body["pageSize"]; exists {
		value, valid := gcpVPCAccessNumberToInt(raw)
		if !valid || value < 0 || value > 1000 {
			respondGCPVPCAccessInvalidArgument(w, path, "pageSize must be a non-negative integer <= 1000")
			return 0, 0, false
		}
		pageSize = value
	}

	if raw, exists := body["pageToken"]; exists {
		token, ok := raw.(string)
		if !ok {
			respondGCPVPCAccessInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		token = strings.TrimSpace(token)
		if token != "" {
			value, err := strconv.Atoi(token)
			if err != nil || value < 0 {
				respondGCPVPCAccessInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = value
		}
	}
	return pageSize, start, true
}

func respondGCPVPCAccessList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPVPCAccessOutOfRange(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           items[start:end],
		"nextPageToken": next,
	})
	return true
}

func decodeGCPVPCAccessJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPVPCAccessInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPVPCAccessInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPVPCAccessJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPVPCAccessJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPVPCAccessInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpVPCAccessConnectorFromBody(body map[string]any) map[string]any {
	if connector, ok := body["connector"].(map[string]any); ok {
		return connector
	}
	return body
}

func gcpVPCAccessString(m map[string]any, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func gcpVPCAccessNumberToInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v != float64(int(v)) {
			return 0, false
		}
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func validateGCPVPCAccessConnector(w http.ResponseWriter, path string, connector map[string]any) bool {
	network := strings.TrimSpace(gcpVPCAccessString(connector, "network"))
	var subnet map[string]any
	if rawSubnet, exists := connector["subnet"]; exists {
		cast, ok := rawSubnet.(map[string]any)
		if !ok {
			respondGCPVPCAccessInvalidArgument(w, path, "connector.subnet must be an object")
			return false
		}
		subnet = cast
	}
	subnetName := ""
	if len(subnet) > 0 {
		subnetName = strings.TrimSpace(gcpVPCAccessString(subnet, "name"))
	}
	if network == "" && subnetName == "" {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.network or connector.subnet.name is required")
		return false
	}

	if cidr := strings.TrimSpace(gcpVPCAccessString(connector, "ipCidrRange")); cidr != "" && !strings.Contains(cidr, "/") {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.ipCidrRange must use CIDR notation")
		return false
	}

	minThroughput, minThroughputSet, ok := gcpVPCAccessOptionalNonNegativeInt(connector, "minThroughput")
	if !ok {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.minThroughput must be a non-negative integer")
		return false
	}
	maxThroughput, maxThroughputSet, ok := gcpVPCAccessOptionalNonNegativeInt(connector, "maxThroughput")
	if !ok {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.maxThroughput must be a non-negative integer")
		return false
	}
	if minThroughputSet && maxThroughputSet && maxThroughput < minThroughput {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.maxThroughput must be >= connector.minThroughput")
		return false
	}

	minInstances, minInstancesSet, ok := gcpVPCAccessOptionalNonNegativeInt(connector, "minInstances")
	if !ok {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.minInstances must be a non-negative integer")
		return false
	}
	maxInstances, maxInstancesSet, ok := gcpVPCAccessOptionalNonNegativeInt(connector, "maxInstances")
	if !ok {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.maxInstances must be a non-negative integer")
		return false
	}
	if minInstancesSet && maxInstancesSet && maxInstances < minInstances {
		respondGCPVPCAccessInvalidArgument(w, path, "connector.maxInstances must be >= connector.minInstances")
		return false
	}
	return true
}

func gcpVPCAccessOptionalNonNegativeInt(m map[string]any, key string) (value int, present bool, ok bool) {
	raw, exists := m[key]
	if !exists {
		return 0, false, true
	}
	n, valid := gcpVPCAccessNumberToInt(raw)
	if !valid || n < 0 {
		return 0, true, false
	}
	return n, true, true
}

func gcpVPCAccessLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"metadata": map[string]any{
			"provider": providerGCP,
			"service":  "vpcaccess",
		},
	}
}

func gcpVPCAccessConnector(project, location, connectorID string) map[string]any {
	return map[string]any{
		"name":              fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, location, connectorID),
		"network":           "default",
		"ipCidrRange":       "10.8.0.0/28",
		"state":             "READY",
		"minThroughput":     200,
		"maxThroughput":     300,
		"machineType":       "e2-micro",
		"minInstances":      2,
		"maxInstances":      3,
		"connectedProjects": []string{project},
		"subnet": map[string]any{
			"name":      "default",
			"projectId": project,
		},
	}
}

func gcpVPCAccessOperation(project, location, operationID, method, target string, done bool) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": done,
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.vpcaccess.v1.OperationMetadata",
			"method":     method,
			"createTime": gcpVPCAccessReferenceTime.Format(time.RFC3339),
			"endTime":    gcpVPCAccessReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
			"target":     target,
		},
	}
}

func isGCPVPCAccessMissingID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(id, "missing") || strings.Contains(id, "notfound")
}

func respondGCPVPCAccessInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPVPCAccessError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPVPCAccessNotFound(w http.ResponseWriter, path, message string) {
	respondGCPVPCAccessError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPVPCAccessAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPVPCAccessError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPVPCAccessFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPVPCAccessError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPVPCAccessOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPVPCAccessError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPVPCAccessError(w http.ResponseWriter, status int, errType, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errType,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_vpcaccess(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "vpcaccess") {
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
			"name":     "projects/stackyard/locations/us-central1/connectors/connector-1",
			"service":  "vpcaccess",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
