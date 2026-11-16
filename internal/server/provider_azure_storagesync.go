package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.StorageSync (Azure File Sync) sync services with their sync
// groups, cloud and server endpoints and registered servers, served by the
// shared ARM handler. No file share is synced and no agent connects:
// registered servers are records of what the client registers.
func init() {
	const ns = "Microsoft.StorageSync"
	accepted := func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil }
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageSyncServices",
		ValidateName:       azureStorageSyncValidateName,
		Validate:           azureEnumProperty("incomingTrafficPolicy", "AllowAllTraffic", "AllowVirtualNetworksOnly"),
		ReadOnlyProperties: azureStorageSyncServiceProperties,
	})
	registerAzureARMProviderAction(ns, "locations/checkNameAvailability", func(s *Server, payload map[string]any) (int, any) {
		return s.azureIoTNameAvailability(payload, ns+"/storageSyncServices", azureStorageSyncValidateName)
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageSyncServices/syncGroups", Proxy: true,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"uniqueId": azureStorageSyncID(rec, ""), "syncGroupStatus": "0"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageSyncServices/syncGroups/cloudEndpoints", Proxy: true,
		Validate:           azureStorageSyncCloudEndpointValidate,
		ReadOnlyProperties: azureStorageSyncCloudEndpointProperties,
		Actions: map[string]azureARMAction{
			"prebackup":              accepted,
			"postbackup":             azureStorageSyncPostBackup,
			"prerestore":             accepted,
			"postrestore":            accepted,
			"restoreheartbeat":       func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, nil },
			"triggerchangedetection": accepted,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageSyncServices/syncGroups/serverEndpoints", Proxy: true,
		Validate:           azureStorageSyncServerEndpointValidate,
		ReadOnlyProperties: azureStorageSyncServerEndpointProperties,
		Actions:            map[string]azureARMAction{"recallaction": accepted},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageSyncServices/registeredServers", Proxy: true,
		ValidateName: func(name string) string {
			if _, err := uuid.Parse(name); err != nil {
				return "Registered server names are the server's ID, a GUID."
			}
			return ""
		},
		Validate:           azureStorageSyncRegisteredServerValidate,
		ReadOnlyProperties: azureStorageSyncRegisteredServerProperties,
		Actions:            map[string]azureARMAction{"triggerrollover": func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, nil }},
	})
}

var azureStorageSyncNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,260}$`)

func azureStorageSyncValidateName(name string) string {
	if !azureStorageSyncNamePattern.MatchString(name) || strings.HasSuffix(name, ".") {
		return "Storage Sync Service names are 1-260 letters, digits, underscores, hyphens or periods, and cannot end with a period."
	}
	return ""
}

// azureStorageSyncID returns a stable GUID for rec, or for one of its
// facets when label is set.
func azureStorageSyncID(rec *azureARMResource, label string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+label)).String()
}

func azureStorageSyncServiceProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"storageSyncServiceStatus": 0,
		"storageSyncServiceUid":    azureStorageSyncID(rec, ""),
		"useIdentity":              len(rec.Identity) != 0,
	}
	if _, present := rec.Properties["incomingTrafficPolicy"]; !present {
		props["incomingTrafficPolicy"] = "AllowAllTraffic"
	}
	return props
}

func azureStorageSyncCloudEndpointValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.Contains(strings.ToLower(azureARMString(props["storageAccountResourceId"])), "/providers/microsoft.storage/storageaccounts/") {
		return "properties.storageAccountResourceId must be a storage account ID."
	}
	if strings.TrimSpace(azureARMString(props["azureFileShareName"])) == "" {
		return "properties.azureFileShareName is required."
	}
	if tenant, present := props["storageAccountTenantId"]; present {
		if _, err := uuid.Parse(azureARMString(tenant)); err != nil {
			return "properties.storageAccountTenantId must be a GUID."
		}
	}
	return ""
}

// azureStorageSyncCloudEndpointProperties names the endpoint after its
// share and reports it ready to sync.
func azureStorageSyncCloudEndpointProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"backupEnabled": "false",
		"partnershipId": azureStorageSyncID(rec, "#partnership"),
		"changeEnumerationStatus": map[string]any{
			"lastEnumerationStatus": map[string]any{"namespaceFilesCount": 0, "namespaceDirectoriesCount": 0, "namespaceSizeBytes": 0},
		},
	}
	for key, value := range map[string]any{"storageAccountTenantId": azureDefaultTenantID, "friendlyName": rec.Properties["azureFileShareName"]} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureStorageSyncPostBackup(rec *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{"backupMetadata": map[string]any{"cloudEndpointName": rec.Name}}
}

var (
	azureStorageSyncTieringValues  = []string{"on", "off"}
	azureStorageSyncDownloadPolicy = []string{"NamespaceThenModifiedFiles", "NamespaceOnly", "AvoidTieredFiles"}
	azureStorageSyncCacheModes     = []string{"DownloadNewAndModifiedFiles", "UpdateLocallyCachedFiles"}
)

func azureStorageSyncServerEndpointValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["serverLocalPath"])) == "" {
		return "properties.serverLocalPath is required."
	}
	if !strings.Contains(strings.ToLower(azureARMString(props["serverResourceId"])), "/providers/microsoft.storagesync/storagesyncservices/") ||
		!strings.Contains(strings.ToLower(azureARMString(props["serverResourceId"])), "/registeredservers/") {
		return "properties.serverResourceId must be a registered server ID."
	}
	if tiering, present := props["cloudTiering"]; present && !slices.Contains(azureStorageSyncTieringValues, strings.ToLower(azureARMString(tiering))) {
		return "properties.cloudTiering must be on or off."
	}
	if raw, present := props["volumeFreeSpacePercent"]; present {
		if percent, ok := raw.(float64); !ok || percent != float64(int(percent)) || percent < 0 || percent > 100 {
			return "properties.volumeFreeSpacePercent must be an integer from 0 to 100."
		}
	}
	if raw, present := props["tierFilesOlderThanDays"]; present {
		if days, ok := raw.(float64); !ok || days != float64(int(days)) || days < 0 || days > 2147483647 {
			return "properties.tierFilesOlderThanDays must be a non-negative integer."
		}
	}
	if msg := azureEnumProperty("initialDownloadPolicy", azureStorageSyncDownloadPolicy...)(payload); msg != "" {
		return msg
	}
	if msg := azureEnumProperty("localCacheMode", azureStorageSyncCacheModes...)(payload); msg != "" {
		return msg
	}
	return azureEnumProperty("initialUploadPolicy", "ServerAuthoritative", "Merge")(payload)
}

// azureStorageSyncServerEndpointProperties reports the endpoint in sync
// and fills Azure's tiering and download defaults.
func azureStorageSyncServerEndpointProperties(rec *azureARMResource) map[string]any {
	server := azureARMString(rec.Properties["serverResourceId"])
	props := map[string]any{
		"syncStatus": map[string]any{"combinedHealth": "Healthy", "syncActivity": "Upload", "uploadHealth": "Healthy", "downloadHealth": "Healthy"},
		"offlineDataTransferStorageAccountResourceId": nil,
	}
	defaults := map[string]any{
		"cloudTiering": "off", "volumeFreeSpacePercent": 20, "tierFilesOlderThanDays": 0,
		"initialDownloadPolicy": "NamespaceThenModifiedFiles", "initialUploadPolicy": "Merge",
		"localCacheMode": "UpdateLocallyCachedFiles", "offlineDataTransfer": "off",
		"friendlyName": server[strings.LastIndex(server, "/")+1:],
	}
	for key, value := range defaults {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureStorageSyncRegisteredServerValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["serverCertificate"])) == "" {
		return "properties.serverCertificate is required."
	}
	if strings.TrimSpace(azureARMString(props["agentVersion"])) == "" {
		return "properties.agentVersion is required."
	}
	return ""
}

// azureStorageSyncRegisteredServerProperties reports the server as online,
// identified by its name, and in the sync service's cluster.
func azureStorageSyncRegisteredServerProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"serverId":                  rec.Name,
		"serverManagementErrorCode": 0,
		"lastHeartBeat":             rec.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		"agentVersionStatus":        "Ok",
		"storageSyncServiceUid":     uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/registeredservers/")]))).String(),
	}
	for key, value := range map[string]any{"serverRole": "Standalone", "friendlyName": rec.Name} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}
