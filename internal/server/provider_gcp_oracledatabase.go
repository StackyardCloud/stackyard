package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPOracleDatabaseRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_oracledatabase(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPOracleDatabasePath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPOracleDatabasePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.oracledatabase.v1.OracleDatabase/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPOracleDatabaseLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/cloudExadataInfrastructures") ||
		strings.Contains(path, "/cloudVmClusters") ||
		strings.Contains(path, "/entitlements") ||
		strings.Contains(path, "/dbServers") ||
		strings.Contains(path, "/dbNodes") ||
		strings.Contains(path, "/giVersions") ||
		strings.Contains(path, "/minorVersions") ||
		strings.Contains(path, "/dbSystemShapes") ||
		strings.Contains(path, "/autonomousDatabases") ||
		strings.Contains(path, "/autonomousDbVersions") ||
		strings.Contains(path, "/autonomousDatabaseCharacterSets") ||
		strings.Contains(path, "/autonomousDatabaseBackups") ||
		strings.Contains(path, "/odbNetworks") ||
		strings.Contains(path, "/odbSubnets") ||
		strings.Contains(path, "/exadbVmClusters") ||
		strings.Contains(path, "/exascaleDbStorageVaults") ||
		strings.Contains(path, "/dbSystemInitialStorageSizes") ||
		strings.Contains(path, "/dbSystems") ||
		strings.Contains(path, "/databases") ||
		strings.Contains(path, "/pluggableDatabases") ||
		strings.Contains(path, "/dbVersions") ||
		strings.Contains(path, "/databaseCharacterSets") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":restore") ||
		strings.Contains(path, ":generateWallet") ||
		strings.Contains(path, ":start") ||
		strings.Contains(path, ":stop") ||
		strings.Contains(path, ":restart") ||
		strings.Contains(path, ":switchover") ||
		strings.Contains(path, ":failover") ||
		strings.Contains(path, ":removeVirtualMachine") ||
		strings.Contains(path, ":cancel")
}

func isGCPOracleDatabaseLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_oracledatabase(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "oracledatabase") {
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
			"name":     "projects/stackyard/locations/us-central1/oracledatabase/sample",
			"service":  "oracledatabase",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
