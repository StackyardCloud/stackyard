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

const gcpVMwareEngineGRPCPathPrefix = "/gcp/google.cloud.vmwareengine.v1.VmwareEngine/"

var (
	gcpVMwareEngineReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	gcpVMwareEngineCollectionDefaults = map[string]string{
		"privateClouds":                  "private-cloud-1",
		"clusters":                       "cluster-1",
		"nodes":                          "node-1",
		"externalAddresses":              "external-address-1",
		"subnets":                        "subnet-1",
		"networkPolicies":                "network-policy-1",
		"externalAccessRules":            "external-access-rule-1",
		"loggingServers":                 "logging-server-1",
		"nodeTypes":                      "node-type-1",
		"hcxActivationKeys":              "hcx-activation-key-1",
		"managementDnsZoneBindings":      "management-dns-zone-binding-1",
		"vmwareEngineNetworks":           "vmware-engine-network-1",
		"privateConnections":             "private-connection-1",
		"networkPeerings":                "network-peering-1",
		"peeringRoutes":                  "peering-route-1",
		"privateConnectionPeeringRoutes": "peering-route-1",
		"dnsForwarding":                  "dns-forwarding",
		"dnsBindPermission":              "dns-bind-permission",
	}

	gcpVMwareEngineCreateIDQueryParam = map[string]string{
		"privateClouds":             "privateCloudId",
		"clusters":                  "clusterId",
		"externalAddresses":         "externalAddressId",
		"externalAccessRules":       "externalAccessRuleId",
		"loggingServers":            "loggingServerId",
		"hcxActivationKeys":         "hcxActivationKeyId",
		"networkPolicies":           "networkPolicyId",
		"managementDnsZoneBindings": "managementDnsZoneBindingId",
		"vmwareEngineNetworks":      "vmwareEngineNetworkId",
		"privateConnections":        "privateConnectionId",
		"networkPeerings":           "networkPeeringId",
	}

	gcpVMwareEngineCreateBodyKey = map[string]string{
		"privateClouds":             "privateCloud",
		"clusters":                  "cluster",
		"externalAddresses":         "externalAddress",
		"externalAccessRules":       "externalAccessRule",
		"loggingServers":            "loggingServer",
		"hcxActivationKeys":         "hcxActivationKey",
		"networkPolicies":           "networkPolicy",
		"managementDnsZoneBindings": "managementDnsZoneBinding",
		"vmwareEngineNetworks":      "vmwareEngineNetwork",
		"privateConnections":        "privateConnection",
		"networkPeerings":           "networkPeering",
	}

	gcpVMwareEngineUpdateBodyKey = map[string]string{
		"privateClouds":             "privateCloud",
		"clusters":                  "cluster",
		"externalAddresses":         "externalAddress",
		"subnets":                   "subnet",
		"externalAccessRules":       "externalAccessRule",
		"loggingServers":            "loggingServer",
		"dnsForwarding":             "dnsForwarding",
		"networkPeerings":           "networkPeering",
		"networkPolicies":           "networkPolicy",
		"managementDnsZoneBindings": "managementDnsZoneBinding",
		"vmwareEngineNetworks":      "vmwareEngineNetwork",
		"privateConnections":        "privateConnection",
	}
)

func (s *Server) handleGCPVMwareEngineRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_vmwareengine(w, r) {
		return true
	}

	path := normalizeGCPVMwareEnginePath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpVMwareEngineGRPCPathPrefix) {
		return handleGCPVMwareEngineGRPCBridge(w, r, path)
	}

	if isGCPVMwareEngineLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVMwareEngineListLocations(w, r, path) {
			return true
		}
		if handleGCPVMwareEngineGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPVMwareEngineRESTPath(path, hasGCPVMwareEngineHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPVMwareEngineListOperations(w, r, path) {
			return true
		}
		if handleGCPVMwareEngineGetOperation(w, path) {
			return true
		}
		if handleGCPVMwareEngineRESTGet(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPVMwareEngineCancelOperation(w, path) {
			return true
		}
		if handleGCPVMwareEngineRESTPost(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPVMwareEngineRESTPatch(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPVMwareEngineDeleteOperation(w, path) {
			return true
		}
		if handleGCPVMwareEngineRESTDelete(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPVMwareEnginePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVMwareEngineHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "vmwareengine",
		"vmwareengine-apiv1",
		"vmwareengine_apiv1",
		"vmware-engine",
		"vmware_engine",
		"cloud-vmware-engine",
		"cloud_vmware_engine",
		"gcp-vmware-engine",
		"gcp-vmwareengine":
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-vmwareengine-apiv1") || strings.Contains(ua, "cloud.google.com/go/vmwareengine")
}

func isGCPVMwareEngineLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVMwareEngineHint(r) {
		return false
	}
	_, _, _, ok := parseGCPVMwareEngineProjectLocationsPath(path)
	return ok
}

func isGCPVMwareEngineRESTPath(path string, includeHint bool) bool {
	if _, _, _, ok := parseGCPVMwareEngineProjectLocationsPath(path); ok {
		return includeHint
	}
	project, location, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok {
		return false
	}
	if project == "" || location == "" {
		return false
	}
	if len(tail) == 0 {
		return includeHint
	}
	if isGCPVMwareEngineOperationsTail(tail) {
		return true
	}
	_, parsed := parseGCPVMwareEngineResourceTail(tail)
	return parsed || includeHint
}

func handleGCPVMwareEngineListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPVMwareEngineProjectLocationsPath(path)
	if !ok || !list {
		return false
	}

	pageSize, start, ok := parseGCPVMwareEnginePagination(w, r, path, 1000)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpVMwareEngineLocation(project, "us-central1"),
		gcpVMwareEngineLocation(project, "global"),
	}
	return respondGCPVMwareEngineList(w, "locations", items, pageSize, start, path)
}

func handleGCPVMwareEngineGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPVMwareEngineProjectLocationsPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVMwareEngineLocation(project, location))
	return true
}

func handleGCPVMwareEngineListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok || !isGCPVMwareEngineOperationsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPVMwareEnginePagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpVMwareEngineOperation(project, location, "vmwareengine-op-1", false),
		gcpVMwareEngineOperation(project, location, "vmwareengine-op-2", true),
	}
	return respondGCPVMwareEngineList(w, "operations", items, pageSize, start, path)
}

func handleGCPVMwareEngineGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok || !isGCPVMwareEngineOperationResourceTail(tail) {
		return false
	}
	operationID := strings.TrimSpace(tail[1])
	if isGCPVMwareEngineMissingID(operationID) {
		respondGCPVMwareEngineNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, operationID, strings.Contains(strings.ToLower(operationID), "done")))
	return true
}

func handleGCPVMwareEngineCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok || !isGCPVMwareEngineOperationActionTail(tail, "cancel") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVMwareEngineDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok || !isGCPVMwareEngineOperationResourceTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVMwareEngineRESTGet(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMwareEngineResourceTail(tail)
	if !ok {
		return false
	}

	if info.action != "" {
		switch info.action {
		case "showNsxCredentials", "showVcenterCredentials":
			respondJSON(w, http.StatusOK, gcpVMwareEngineCredentialsFixture())
			return true
		case "fetchNetworkPolicyExternalAddresses", "fetchExternalAddresses":
			respondJSON(w, http.StatusOK, map[string]any{
				"externalAddresses": []map[string]any{gcpVMwareEngineResourceFixture(project, location, "externalAddresses", append(append([]string{}, info.segments...), "external-address-1"))},
				"nextPageToken":     "",
			})
			return true
		default:
			return false
		}
	}

	if info.collection == "dnsBindPermission" && info.isCollection {
		name := gcpVMwareEngineRESTResourceName(project, location, info.segments)
		respondJSON(w, http.StatusOK, gcpVMwareEngineDNSBindPermissionFixture(name))
		return true
	}
	if info.collection == "dnsForwarding" && info.isCollection {
		respondJSON(w, http.StatusOK, gcpVMwareEngineResourceFixture(project, location, "dnsForwarding", info.segments))
		return true
	}

	if info.isCollection {
		pageSize, start, valid := parseGCPVMwareEnginePagination(w, r, path, 1000)
		if !valid {
			return true
		}
		id := gcpVMwareEngineCollectionDefaults[info.collection]
		itemTail := append(append([]string{}, info.segments...), id)
		items := []map[string]any{gcpVMwareEngineResourceFixture(project, location, info.collection, itemTail)}
		return respondGCPVMwareEngineList(w, gcpVMwareEngineListFieldForCollection(info.collection), items, pageSize, start, path)
	}

	if isGCPVMwareEngineMissingID(info.id) {
		respondGCPVMwareEngineNotFound(w, path, "resource not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVMwareEngineResourceFixture(project, location, info.collection, info.segments))
	return true
}

func handleGCPVMwareEngineRESTPost(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMwareEngineResourceTail(tail)
	if !ok {
		return false
	}

	body, valid := decodeGCPVMwareEngineJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}

	if info.action != "" {
		targetID := info.id
		if targetID == "" && info.collection == "dnsBindPermission" && len(info.segments) >= 2 {
			targetID = info.segments[len(info.segments)-2]
		}
		if targetID == "" && len(info.segments) > 0 {
			targetID = info.segments[len(info.segments)-1]
		}
		switch info.action {
		case "undelete", "resetNsxCredentials", "resetVcenterCredentials", "repair", "grant", "grantDnsBindPermission", "revoke", "revokeDnsBindPermission":
			if info.collection == "dnsBindPermission" && location != "global" {
				respondGCPVMwareEngineInvalidArgument(w, path, "dnsBindPermission actions must use locations/global")
				return true
			}
			respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, info.action+"."+targetID, false))
			return true
		case "fetchNetworkPolicyExternalAddresses", "fetchExternalAddresses":
			respondJSON(w, http.StatusOK, map[string]any{
				"externalAddresses": []map[string]any{gcpVMwareEngineResourceFixture(project, location, "externalAddresses", []string{"externalAddresses", "external-address-1"})},
				"nextPageToken":     "",
			})
			return true
		default:
			respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, info.action+"."+targetID, false))
			return true
		}
	}

	if !info.isCollection {
		return false
	}

	idQueryParam, createable := gcpVMwareEngineCreateIDQueryParam[info.collection]
	if !createable {
		return false
	}

	resourceID := strings.TrimSpace(r.URL.Query().Get(idQueryParam))
	if resourceID == "" {
		resourceID = gcpVMwareEngineCollectionDefaults[info.collection]
	}
	if isGCPVMwareEngineMissingID(resourceID) || strings.Contains(strings.ToLower(resourceID), "existing") {
		respondGCPVMwareEngineAlreadyExists(w, path, "resource already exists")
		return true
	}

	requiredBodyKey := gcpVMwareEngineCreateBodyKey[info.collection]
	if requiredBodyKey != "" {
		if wrapped, exists := body[requiredBodyKey]; exists {
			if _, ok := wrapped.(map[string]any); !ok {
				respondGCPVMwareEngineInvalidArgument(w, path, requiredBodyKey+" must be an object")
				return true
			}
		} else if info.collection == "privateClouds" && len(body) == 0 {
			respondGCPVMwareEngineInvalidArgument(w, path, requiredBodyKey+" is required")
			return true
		}
	}

	if info.collection == "networkPeerings" && location != "global" {
		respondGCPVMwareEngineInvalidArgument(w, path, "networkPeerings must use locations/global")
		return true
	}

	respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, "create."+info.collection+"."+resourceID, false))
	return true
}

func handleGCPVMwareEngineRESTPatch(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMwareEngineResourceTail(tail)
	if !ok || info.action != "" {
		return false
	}
	if info.isCollection && info.collection != "dnsForwarding" {
		return false
	}
	bodyKey, updateable := gcpVMwareEngineUpdateBodyKey[info.collection]
	if !updateable {
		return false
	}

	body, valid := decodeGCPVMwareEngineJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}

	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = gcpVMwareEngineUpdateMaskFromBody(body)
	}
	if mask == "" {
		respondGCPVMwareEngineInvalidArgument(w, path, "updateMask is required")
		return true
	}

	resourceBody := gcpVMwareEngineBodyResource(body, bodyKey)
	if len(resourceBody) == 0 {
		respondGCPVMwareEngineInvalidArgument(w, path, bodyKey+" is required")
		return true
	}

	expectedName := gcpVMwareEngineRESTResourceName(project, location, info.segments)
	if name := strings.TrimSpace(gcpVMwareEngineString(resourceBody, "name")); name != "" && name != expectedName {
		respondGCPVMwareEngineInvalidArgument(w, path, bodyKey+".name must match requested resource")
		return true
	}

	if info.collection == "networkPeerings" && location != "global" {
		respondGCPVMwareEngineInvalidArgument(w, path, "networkPeerings must use locations/global")
		return true
	}

	opID := "update." + info.collection
	if info.id != "" {
		opID += "." + info.id
	}
	respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, opID, false))
	return true
}

func handleGCPVMwareEngineRESTDelete(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVMwareEngineLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMwareEngineResourceTail(tail)
	if !ok || info.isCollection || info.action != "" {
		return false
	}
	if isGCPVMwareEngineMissingID(info.id) {
		respondGCPVMwareEngineNotFound(w, path, "resource not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, "delete."+info.collection+"."+info.id, false))
	return true
}

func handleGCPVMwareEngineGRPCBridge(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	method := strings.TrimPrefix(path, gcpVMwareEngineGRPCPathPrefix)
	if method == "" || strings.Contains(method, "/") {
		return false
	}

	body, ok := decodeGCPVMwareEngineJSONBodyOptional(w, r, path)
	if !ok {
		return true
	}

	if method == "FetchNetworkPolicyExternalAddresses" {
		name := strings.TrimSpace(gcpVMwareEngineString(body, "networkPolicy"))
		project, location, tail, parsed := parseGCPVMwareEngineResourceName(name)
		if !parsed || !gcpVMwareEngineTailHasCollection(tail, "networkPolicies") {
			respondGCPVMwareEngineInvalidArgument(w, path, "networkPolicy is required")
			return true
		}
		pageSize, start, valid := parseGCPVMwareEngineBridgePagination(w, body, path)
		if !valid {
			return true
		}
		items := []map[string]any{gcpVMwareEngineResourceFixture(project, location, "externalAddresses", append(tail, "externalAddresses", "external-address-1"))}
		return respondGCPVMwareEngineList(w, "externalAddresses", items, pageSize, start, path)
	}

	if listSpec, found := gcpVMwareEngineGRPCListMethodSpec(method); found {
		parent := strings.TrimSpace(gcpVMwareEngineString(body, listSpec.parentField))
		project, location, _, parsed := parseGCPVMwareEngineResourceName(parent)
		if !parsed {
			respondGCPVMwareEngineInvalidArgument(w, path, listSpec.parentField+" is required")
			return true
		}
		pageSize, start, valid := parseGCPVMwareEngineBridgePagination(w, body, path)
		if !valid {
			return true
		}
		id := gcpVMwareEngineCollectionDefaults[listSpec.collection]
		var tail []string
		if parentTail, parsedParent := parseGCPVMwareEngineParentTail(parent); parsedParent {
			tail = append(parentTail, listSpec.collection, id)
		}
		item := gcpVMwareEngineResourceFixture(project, location, listSpec.collection, tail)
		return respondGCPVMwareEngineList(w, listSpec.responseField, []map[string]any{item}, pageSize, start, path)
	}

	if getSpec, found := gcpVMwareEngineGRPCGetMethodSpec(method); found {
		name := strings.TrimSpace(gcpVMwareEngineString(body, getSpec.nameField))
		project, location, tail, parsed := parseGCPVMwareEngineResourceName(name)
		if !parsed {
			respondGCPVMwareEngineInvalidArgument(w, path, getSpec.nameField+" is required")
			return true
		}
		if getSpec.collection == "dnsBindPermission" {
			respondJSON(w, http.StatusOK, gcpVMwareEngineDNSBindPermissionFixture(name))
			return true
		}
		if getSpec.collection == "dnsForwarding" {
			respondJSON(w, http.StatusOK, gcpVMwareEngineResourceFixture(project, location, "dnsForwarding", tail))
			return true
		}
		if !gcpVMwareEngineTailHasCollection(tail, getSpec.collection) {
			respondGCPVMwareEngineInvalidArgument(w, path, getSpec.nameField+" is invalid")
			return true
		}
		resourceID := tail[len(tail)-1]
		if isGCPVMwareEngineMissingID(resourceID) {
			respondGCPVMwareEngineNotFound(w, path, "resource not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVMwareEngineResourceFixture(project, location, getSpec.collection, tail))
		return true
	}

	if createSpec, found := gcpVMwareEngineGRPCCreateMethodSpec(method); found {
		parent := strings.TrimSpace(gcpVMwareEngineString(body, createSpec.parentField))
		project, location, _, parsed := parseGCPVMwareEngineResourceName(parent)
		if !parsed {
			respondGCPVMwareEngineInvalidArgument(w, path, createSpec.parentField+" is required")
			return true
		}
		resourceID := strings.TrimSpace(gcpVMwareEngineString(body, createSpec.idField))
		if resourceID == "" {
			respondGCPVMwareEngineInvalidArgument(w, path, createSpec.idField+" is required")
			return true
		}
		rawResource, exists := body[createSpec.bodyField]
		if !exists {
			respondGCPVMwareEngineInvalidArgument(w, path, createSpec.bodyField+" is required")
			return true
		}
		if _, ok := rawResource.(map[string]any); !ok {
			respondGCPVMwareEngineInvalidArgument(w, path, createSpec.bodyField+" must be an object")
			return true
		}
		if createSpec.collection == "networkPeerings" && location != "global" {
			respondGCPVMwareEngineInvalidArgument(w, path, "networkPeerings must use locations/global")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, "create."+createSpec.collection+"."+resourceID, false))
		return true
	}

	if updateSpec, found := gcpVMwareEngineGRPCUpdateMethodSpec(method); found {
		resourceBody := gcpVMwareEngineBodyResource(body, updateSpec.bodyField)
		if len(resourceBody) == 0 {
			respondGCPVMwareEngineInvalidArgument(w, path, updateSpec.bodyField+" is required")
			return true
		}
		name := strings.TrimSpace(gcpVMwareEngineString(resourceBody, "name"))
		project, location, _, parsed := parseGCPVMwareEngineResourceName(name)
		if !parsed {
			respondGCPVMwareEngineInvalidArgument(w, path, updateSpec.bodyField+".name is required")
			return true
		}
		if gcpVMwareEngineUpdateMaskFromBody(body) == "" {
			respondGCPVMwareEngineInvalidArgument(w, path, "updateMask is required")
			return true
		}
		if updateSpec.collection == "networkPeerings" && location != "global" {
			respondGCPVMwareEngineInvalidArgument(w, path, "networkPeerings must use locations/global")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, "update."+updateSpec.collection, false))
		return true
	}

	if found := gcpVMwareEngineGRPCDeleteMethod(method); found {
		name := strings.TrimSpace(gcpVMwareEngineString(body, "name"))
		project, location, tail, parsed := parseGCPVMwareEngineResourceName(name)
		if !parsed {
			respondGCPVMwareEngineInvalidArgument(w, path, "name is required")
			return true
		}
		id := tail[len(tail)-1]
		respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, "delete."+id, false))
		return true
	}

	if actionSpec, found := gcpVMwareEngineGRPCActionSpec(method); found {
		name := strings.TrimSpace(gcpVMwareEngineString(body, actionSpec.nameField))
		project, location, tail, parsed := parseGCPVMwareEngineResourceName(name)
		if !parsed {
			respondGCPVMwareEngineInvalidArgument(w, path, actionSpec.nameField+" is required")
			return true
		}
		if actionSpec.responseType == "credentials" {
			respondJSON(w, http.StatusOK, gcpVMwareEngineCredentialsFixture())
			return true
		}
		id := "resource"
		if gcpVMwareEngineTailHasCollection(tail, "dnsBindPermission") && len(tail) >= 2 {
			id = tail[len(tail)-2]
		} else if len(tail) > 0 {
			id = tail[len(tail)-1]
		}
		respondJSON(w, http.StatusOK, gcpVMwareEngineOperation(project, location, actionSpec.action+"."+id, false))
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

type gcpVMwareEngineListSpec struct {
	collection    string
	responseField string
	parentField   string
}

func gcpVMwareEngineGRPCListMethodSpec(method string) (gcpVMwareEngineListSpec, bool) {
	specs := map[string]gcpVMwareEngineListSpec{
		"ListPrivateClouds":                  {collection: "privateClouds", responseField: "privateClouds", parentField: "parent"},
		"ListClusters":                       {collection: "clusters", responseField: "clusters", parentField: "parent"},
		"ListNodes":                          {collection: "nodes", responseField: "nodes", parentField: "parent"},
		"ListExternalAddresses":              {collection: "externalAddresses", responseField: "externalAddresses", parentField: "parent"},
		"ListSubnets":                        {collection: "subnets", responseField: "subnets", parentField: "parent"},
		"ListExternalAccessRules":            {collection: "externalAccessRules", responseField: "externalAccessRules", parentField: "parent"},
		"ListLoggingServers":                 {collection: "loggingServers", responseField: "loggingServers", parentField: "parent"},
		"ListNodeTypes":                      {collection: "nodeTypes", responseField: "nodeTypes", parentField: "parent"},
		"ListNetworkPeerings":                {collection: "networkPeerings", responseField: "networkPeerings", parentField: "parent"},
		"ListPeeringRoutes":                  {collection: "peeringRoutes", responseField: "peeringRoutes", parentField: "parent"},
		"ListHcxActivationKeys":              {collection: "hcxActivationKeys", responseField: "hcxActivationKeys", parentField: "parent"},
		"ListNetworkPolicies":                {collection: "networkPolicies", responseField: "networkPolicies", parentField: "parent"},
		"ListManagementDnsZoneBindings":      {collection: "managementDnsZoneBindings", responseField: "managementDnsZoneBindings", parentField: "parent"},
		"ListVmwareEngineNetworks":           {collection: "vmwareEngineNetworks", responseField: "vmwareEngineNetworks", parentField: "parent"},
		"ListPrivateConnections":             {collection: "privateConnections", responseField: "privateConnections", parentField: "parent"},
		"ListPrivateConnectionPeeringRoutes": {collection: "peeringRoutes", responseField: "peeringRoutes", parentField: "parent"},
	}
	spec, ok := specs[method]
	return spec, ok
}

type gcpVMwareEngineGetSpec struct {
	collection string
	nameField  string
}

func gcpVMwareEngineGRPCGetMethodSpec(method string) (gcpVMwareEngineGetSpec, bool) {
	specs := map[string]gcpVMwareEngineGetSpec{
		"GetPrivateCloud":             {collection: "privateClouds", nameField: "name"},
		"GetCluster":                  {collection: "clusters", nameField: "name"},
		"GetNode":                     {collection: "nodes", nameField: "name"},
		"GetExternalAddress":          {collection: "externalAddresses", nameField: "name"},
		"GetSubnet":                   {collection: "subnets", nameField: "name"},
		"GetExternalAccessRule":       {collection: "externalAccessRules", nameField: "name"},
		"GetLoggingServer":            {collection: "loggingServers", nameField: "name"},
		"GetNodeType":                 {collection: "nodeTypes", nameField: "name"},
		"GetNetworkPeering":           {collection: "networkPeerings", nameField: "name"},
		"GetHcxActivationKey":         {collection: "hcxActivationKeys", nameField: "name"},
		"GetNetworkPolicy":            {collection: "networkPolicies", nameField: "name"},
		"GetManagementDnsZoneBinding": {collection: "managementDnsZoneBindings", nameField: "name"},
		"GetVmwareEngineNetwork":      {collection: "vmwareEngineNetworks", nameField: "name"},
		"GetPrivateConnection":        {collection: "privateConnections", nameField: "name"},
		"GetDnsForwarding":            {collection: "dnsForwarding", nameField: "name"},
		"GetDnsBindPermission":        {collection: "dnsBindPermission", nameField: "name"},
	}
	spec, ok := specs[method]
	return spec, ok
}

type gcpVMwareEngineCreateSpec struct {
	collection  string
	idField     string
	bodyField   string
	parentField string
}

func gcpVMwareEngineGRPCCreateMethodSpec(method string) (gcpVMwareEngineCreateSpec, bool) {
	specs := map[string]gcpVMwareEngineCreateSpec{
		"CreatePrivateCloud":             {collection: "privateClouds", idField: "privateCloudId", bodyField: "privateCloud", parentField: "parent"},
		"CreateCluster":                  {collection: "clusters", idField: "clusterId", bodyField: "cluster", parentField: "parent"},
		"CreateExternalAddress":          {collection: "externalAddresses", idField: "externalAddressId", bodyField: "externalAddress", parentField: "parent"},
		"CreateExternalAccessRule":       {collection: "externalAccessRules", idField: "externalAccessRuleId", bodyField: "externalAccessRule", parentField: "parent"},
		"CreateLoggingServer":            {collection: "loggingServers", idField: "loggingServerId", bodyField: "loggingServer", parentField: "parent"},
		"CreateNetworkPeering":           {collection: "networkPeerings", idField: "networkPeeringId", bodyField: "networkPeering", parentField: "parent"},
		"CreateHcxActivationKey":         {collection: "hcxActivationKeys", idField: "hcxActivationKeyId", bodyField: "hcxActivationKey", parentField: "parent"},
		"CreateNetworkPolicy":            {collection: "networkPolicies", idField: "networkPolicyId", bodyField: "networkPolicy", parentField: "parent"},
		"CreateManagementDnsZoneBinding": {collection: "managementDnsZoneBindings", idField: "managementDnsZoneBindingId", bodyField: "managementDnsZoneBinding", parentField: "parent"},
		"CreateVmwareEngineNetwork":      {collection: "vmwareEngineNetworks", idField: "vmwareEngineNetworkId", bodyField: "vmwareEngineNetwork", parentField: "parent"},
		"CreatePrivateConnection":        {collection: "privateConnections", idField: "privateConnectionId", bodyField: "privateConnection", parentField: "parent"},
	}
	spec, ok := specs[method]
	return spec, ok
}

type gcpVMwareEngineUpdateSpec struct {
	collection string
	bodyField  string
}

func gcpVMwareEngineGRPCUpdateMethodSpec(method string) (gcpVMwareEngineUpdateSpec, bool) {
	specs := map[string]gcpVMwareEngineUpdateSpec{
		"UpdatePrivateCloud":             {collection: "privateClouds", bodyField: "privateCloud"},
		"UpdateCluster":                  {collection: "clusters", bodyField: "cluster"},
		"UpdateExternalAddress":          {collection: "externalAddresses", bodyField: "externalAddress"},
		"UpdateSubnet":                   {collection: "subnets", bodyField: "subnet"},
		"UpdateExternalAccessRule":       {collection: "externalAccessRules", bodyField: "externalAccessRule"},
		"UpdateLoggingServer":            {collection: "loggingServers", bodyField: "loggingServer"},
		"UpdateDnsForwarding":            {collection: "dnsForwarding", bodyField: "dnsForwarding"},
		"UpdateNetworkPeering":           {collection: "networkPeerings", bodyField: "networkPeering"},
		"UpdateNetworkPolicy":            {collection: "networkPolicies", bodyField: "networkPolicy"},
		"UpdateManagementDnsZoneBinding": {collection: "managementDnsZoneBindings", bodyField: "managementDnsZoneBinding"},
		"UpdateVmwareEngineNetwork":      {collection: "vmwareEngineNetworks", bodyField: "vmwareEngineNetwork"},
		"UpdatePrivateConnection":        {collection: "privateConnections", bodyField: "privateConnection"},
	}
	spec, ok := specs[method]
	return spec, ok
}

func gcpVMwareEngineGRPCDeleteMethod(method string) bool {
	switch method {
	case "DeletePrivateCloud", "DeleteCluster", "DeleteExternalAddress", "DeleteExternalAccessRule", "DeleteLoggingServer", "DeleteNetworkPeering", "DeleteNetworkPolicy", "DeleteManagementDnsZoneBinding", "DeleteVmwareEngineNetwork", "DeletePrivateConnection":
		return true
	default:
		return false
	}
}

type gcpVMwareEngineActionSpec struct {
	action       string
	nameField    string
	responseType string
}

func gcpVMwareEngineGRPCActionSpec(method string) (gcpVMwareEngineActionSpec, bool) {
	specs := map[string]gcpVMwareEngineActionSpec{
		"UndeletePrivateCloud":           {action: "undelete", nameField: "name", responseType: "operation"},
		"ShowNsxCredentials":             {action: "showNsxCredentials", nameField: "privateCloud", responseType: "credentials"},
		"ShowVcenterCredentials":         {action: "showVcenterCredentials", nameField: "privateCloud", responseType: "credentials"},
		"ResetNsxCredentials":            {action: "resetNsxCredentials", nameField: "privateCloud", responseType: "operation"},
		"ResetVcenterCredentials":        {action: "resetVcenterCredentials", nameField: "privateCloud", responseType: "operation"},
		"RepairManagementDnsZoneBinding": {action: "repair", nameField: "name", responseType: "operation"},
		"GrantDnsBindPermission":         {action: "grantDnsBindPermission", nameField: "name", responseType: "operation"},
		"RevokeDnsBindPermission":        {action: "revokeDnsBindPermission", nameField: "name", responseType: "operation"},
	}
	spec, ok := specs[method]
	return spec, ok
}

func parseGCPVMwareEngineBridgePagination(w http.ResponseWriter, body map[string]any, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw, exists := body["pageSize"]; exists {
		value, valid := gcpVMwareEngineNumberToInt(raw)
		if !valid || value < 0 || value > 1000 {
			respondGCPVMwareEngineInvalidArgument(w, path, "pageSize must be a non-negative integer <= 1000")
			return 0, 0, false
		}
		pageSize = value
	}
	if raw, exists := body["pageToken"]; exists {
		token, ok := raw.(string)
		if !ok {
			respondGCPVMwareEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		if strings.TrimSpace(token) != "" {
			value, err := strconv.Atoi(strings.TrimSpace(token))
			if err != nil || value < 0 {
				respondGCPVMwareEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = value
		}
	}
	return pageSize, start, true
}

func parseGCPVMwareEngineProjectLocationsPath(path string) (project, location string, list bool, ok bool) {
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

func parseGCPVMwareEngineLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func parseGCPVMwareEngineResourceName(name string) (project, location string, tail []string, ok bool) {
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

func parseGCPVMwareEngineParentTail(name string) (tail []string, ok bool) {
	_, _, tail, ok = parseGCPVMwareEngineResourceName(name)
	return tail, ok
}

type gcpVMwareEngineTailInfo struct {
	segments     []string
	collection   string
	id           string
	action       string
	isCollection bool
}

func parseGCPVMwareEngineResourceTail(tail []string) (gcpVMwareEngineTailInfo, bool) {
	if len(tail) == 0 {
		return gcpVMwareEngineTailInfo{}, false
	}
	segments := append([]string{}, tail...)
	last := strings.TrimSpace(segments[len(segments)-1])
	if last == "" {
		return gcpVMwareEngineTailInfo{}, false
	}
	action := ""
	if resource, act, found := strings.Cut(last, ":"); found {
		if strings.TrimSpace(resource) == "" || strings.TrimSpace(act) == "" {
			return gcpVMwareEngineTailInfo{}, false
		}
		segments[len(segments)-1] = strings.TrimSpace(resource)
		action = strings.TrimSpace(act)
	}
	for _, part := range segments {
		if strings.TrimSpace(part) == "" {
			return gcpVMwareEngineTailInfo{}, false
		}
	}

	isCollection := len(segments)%2 == 1
	collection := ""
	resourceID := ""
	if isCollection {
		collection = segments[len(segments)-1]
	} else {
		collection = segments[len(segments)-2]
		resourceID = segments[len(segments)-1]
	}
	if _, ok := gcpVMwareEngineCollectionDefaults[collection]; !ok {
		return gcpVMwareEngineTailInfo{}, false
	}
	return gcpVMwareEngineTailInfo{segments: segments, collection: collection, id: resourceID, action: action, isCollection: isCollection}, true
}

func gcpVMwareEngineTailHasCollection(tail []string, collection string) bool {
	if len(tail) == 0 {
		return false
	}
	if len(tail)%2 == 1 {
		return tail[len(tail)-1] == collection
	}
	return tail[len(tail)-2] == collection
}

func isGCPVMwareEngineOperationsTail(tail []string) bool {
	return isGCPVMwareEngineOperationsCollectionTail(tail) || isGCPVMwareEngineOperationResourceTail(tail) || isGCPVMwareEngineOperationActionTail(tail, "cancel")
}

func isGCPVMwareEngineOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPVMwareEngineOperationResourceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPVMwareEngineOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	id, parsedAction, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return found && strings.TrimSpace(id) != "" && parsedAction == action
}

func parseGCPVMwareEnginePagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVMwareEngineInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPVMwareEngineOutOfRange(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVMwareEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}

	return pageSize, start, true
}

func respondGCPVMwareEngineList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPVMwareEngineOutOfRange(w, path, "pageToken is out of range")
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

func decodeGCPVMwareEngineJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPVMwareEngineInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPVMwareEngineInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPVMwareEngineJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPVMwareEngineJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPVMwareEngineInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpVMwareEngineBodyResource(body map[string]any, key string) map[string]any {
	if strings.TrimSpace(key) == "" {
		return body
	}
	if nested, ok := body[key].(map[string]any); ok {
		return nested
	}
	return body
}

func gcpVMwareEngineString(body map[string]any, key string) string {
	raw, ok := body[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func gcpVMwareEngineUpdateMaskFromBody(body map[string]any) string {
	if value := strings.TrimSpace(gcpVMwareEngineString(body, "updateMask")); value != "" {
		return value
	}
	mask, _ := body["updateMask"].(map[string]any)
	if len(mask) == 0 {
		return ""
	}
	if paths, ok := mask["paths"].([]any); ok && len(paths) > 0 {
		return "paths"
	}
	return ""
}

func gcpVMwareEngineNumberToInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v < 0 || v != float64(int(v)) {
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

func gcpVMwareEngineRESTResourceName(project, location string, tail []string) string {
	return fmt.Sprintf("projects/%s/locations/%s/%s", project, location, strings.Join(tail, "/"))
}

func gcpVMwareEngineListFieldForCollection(collection string) string {
	switch collection {
	case "privateConnectionPeeringRoutes":
		return "peeringRoutes"
	default:
		return collection
	}
}

func gcpVMwareEngineCredentialsFixture() map[string]any {
	return map[string]any{
		"username": "admin",
		"password": "stackyard-password",
	}
}

func gcpVMwareEngineDNSBindPermissionFixture(name string) map[string]any {
	return map[string]any{
		"name": name,
		"principals": []map[string]any{
			{"serviceAccount": "vmwareengine@stackyard.iam.gserviceaccount.com"},
		},
	}
}

func gcpVMwareEngineResourceFixture(project, location, collection string, tail []string) map[string]any {
	name := gcpVMwareEngineRESTResourceName(project, location, tail)
	switch collection {
	case "privateClouds":
		return map[string]any{
			"name":        name,
			"description": "Stackyard VMware Engine private cloud",
			"createTime":  gcpVMwareEngineReferenceTime.Format(time.RFC3339),
			"updateTime":  gcpVMwareEngineReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		}
	case "clusters":
		return map[string]any{
			"name":        name,
			"description": "Stackyard VMware Engine cluster",
			"createTime":  gcpVMwareEngineReferenceTime.Format(time.RFC3339),
		}
	case "nodes":
		return map[string]any{
			"name": name,
			"fqdn": "node-1.stackyard.internal",
		}
	case "externalAddresses":
		return map[string]any{
			"name":       name,
			"internalIp": "10.10.0.10",
			"externalIp": "35.10.10.10",
		}
	case "subnets":
		return map[string]any{
			"name":        name,
			"ipCidrRange": "10.20.0.0/24",
		}
	case "networkPolicies":
		return map[string]any{
			"name":        name,
			"description": "Stackyard network policy",
		}
	case "externalAccessRules":
		return map[string]any{
			"name":        name,
			"description": "Stackyard external access rule",
			"priority":    1000,
		}
	case "loggingServers":
		return map[string]any{
			"name":     name,
			"hostname": "logs.stackyard.internal",
			"port":     514,
		}
	case "nodeTypes":
		return map[string]any{
			"name":            name,
			"nodeTypeId":      "standard-72",
			"virtualCpuCount": 72,
			"totalCoreCount":  36,
		}
	case "hcxActivationKeys":
		return map[string]any{
			"name":          name,
			"activationKey": "stackyard-hcx-key",
		}
	case "managementDnsZoneBindings":
		return map[string]any{
			"name":       name,
			"vpcNetwork": "projects/stackyard/global/networks/default",
		}
	case "vmwareEngineNetworks":
		return map[string]any{
			"name":        name,
			"description": "Stackyard VMware Engine network",
		}
	case "privateConnections":
		return map[string]any{
			"name":        name,
			"description": "Stackyard private connection",
			"routingMode": "GLOBAL",
		}
	case "networkPeerings":
		return map[string]any{
			"name":        name,
			"description": "Stackyard network peering",
		}
	case "peeringRoutes", "privateConnectionPeeringRoutes":
		return map[string]any{
			"name":          name,
			"destination":   "10.0.0.0/16",
			"nextHopRegion": "us-central1",
		}
	case "dnsForwarding":
		return map[string]any{
			"name": name,
			"forwardingRules": []map[string]any{
				{"domain": "corp.internal", "nameServers": []string{"8.8.8.8"}},
			},
		}
	case "dnsBindPermission":
		return gcpVMwareEngineDNSBindPermissionFixture(name)
	default:
		return map[string]any{"name": name}
	}
}

func gcpVMwareEngineOperation(project, location, operationID string, done bool) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": done,
		"metadata": map[string]any{
			"@type":         "type.googleapis.com/google.cloud.vmwareengine.v1.OperationMetadata",
			"createTime":    gcpVMwareEngineReferenceTime.Format(time.RFC3339),
			"endTime":       gcpVMwareEngineReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
			"target":        "vmwareengine",
			"verb":          "execute",
			"statusMessage": "Stackyard staged VMware Engine operation",
		},
	}
}

func gcpVMwareEngineLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"metadata": map[string]any{
			"provider": providerGCP,
			"service":  "vmwareengine",
		},
	}
}

func isGCPVMwareEngineMissingID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(id, "missing") || strings.Contains(id, "notfound")
}

func respondGCPVMwareEngineInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPVMwareEngineError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPVMwareEngineNotFound(w http.ResponseWriter, path, message string) {
	respondGCPVMwareEngineError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPVMwareEngineAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPVMwareEngineError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPVMwareEngineFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPVMwareEngineError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPVMwareEngineOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPVMwareEngineError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPVMwareEngineError(w http.ResponseWriter, status int, errType, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errType,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_vmwareengine(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "vmwareengine") {
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
			"name":     "projects/stackyard/locations/us-central1/privateClouds/private-cloud-1",
			"service":  "vmwareengine",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
