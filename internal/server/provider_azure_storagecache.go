package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// Microsoft.StorageCache HPC caches with their storage targets, and Azure
// Managed Lustre file systems, served by the shared ARM handler. Nothing is
// mounted or cached: caches report healthy at stable private addresses, and
// flushes, priming and archives finish at once.
func init() {
	const ns = "Microsoft.StorageCache"
	accepted := func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil }
	healthy := map[string]any{"state": "Healthy", "statusDescription": "The cache is in Running state"}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "caches",
		ValidateName:       azureStorageCacheValidateName,
		Validate:           azureStorageCacheValidate,
		ReadOnlyProperties: azureStorageCacheProperties,
		StateProperties:    map[string]any{"health": healthy},
		Transitions: map[string]map[string]any{
			"start": {"health": healthy},
			"stop":  {"health": map[string]any{"state": "Stopped", "statusDescription": "The cache is in Stopped state"}},
		},
		Actions: map[string]azureARMAction{"flush": accepted, "upgradefirmware": accepted, "debuginfo": accepted},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "caches/storageTargets", Proxy: true,
		ValidateName:    azureStorageCacheValidateName,
		Validate:        azureStorageTargetValidate,
		StateProperties: map[string]any{"state": "Ready"},
		Transitions: map[string]map[string]any{
			"suspend": {"state": "Suspended"},
			"resume":  {"state": "Ready"},
		},
		Actions: map[string]azureARMAction{"flush": accepted, "invalidate": accepted},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "amlFilesystems",
		ValidateName:       azureStorageCacheValidateName,
		Validate:           azureAMLFilesystemValidate,
		ReadOnlyProperties: azureAMLFilesystemProperties,
		Actions:            map[string]azureARMAction{"archive": azureStorageCacheOK, "cancelarchive": azureStorageCacheOK},
	})
	registerAzureARMProviderAction(ns, "getRequiredAmlFSSubnetsSize", func(_ *Server, payload map[string]any) (int, any) {
		size, msg := azureAMLFilesystemSubnetSize(azureARMString(azureARMObject(payload["sku"])["name"]), payload["storageCapacityTiB"])
		if msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidParameter", msg)
		}
		return http.StatusOK, map[string]any{"filesystemSubnetSize": size}
	})
	registerAzureARMProviderAction(ns, "checkAmlFSSubnets", func(_ *Server, payload map[string]any) (int, any) {
		if !azureStorageCacheSubnetID(payload["filesystemSubnet"]) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidParameter", "filesystemSubnet must be a subnet ID.")
		}
		if _, msg := azureAMLFilesystemSubnetSize(azureARMString(azureARMObject(payload["sku"])["name"]), payload["storageCapacityTiB"]); msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidParameter", msg)
		}
		return http.StatusOK, nil
	})
}

func azureStorageCacheOK(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, nil }

var azureStorageCacheNamePattern = regexp.MustCompile(`^[-0-9a-zA-Z_]{1,80}$`)

func azureStorageCacheValidateName(name string) string {
	if !azureStorageCacheNamePattern.MatchString(name) {
		return "Names are 1-80 letters, digits, hyphens or underscores."
	}
	return ""
}

func azureStorageCacheSubnetID(raw any) bool {
	id := strings.ToLower(azureARMString(raw))
	return strings.Contains(id, "/providers/microsoft.network/virtualnetworks/") && strings.Contains(id, "/subnets/")
}

// azureStorageCacheSizes maps each HPC Cache SKU to the cache sizes it
// offers, in GB.
var azureStorageCacheSizes = map[string][]float64{
	"Standard_2G":    {3072, 6144, 12288},
	"Standard_4G":    {6144, 12288, 24576},
	"Standard_8G":    {12288, 24576, 49152},
	"Standard_L4_5G": {21623},
	"Standard_L9G":   {43246},
	"Standard_L16G":  {86491},
}

func azureStorageCacheValidate(payload map[string]any) string {
	sku := azureARMString(azureARMObject(payload["sku"])["name"])
	sizes, ok := azureStorageCacheSizes[sku]
	if !ok {
		return "sku.name must be an HPC Cache SKU such as Standard_2G, Standard_4G or Standard_8G."
	}
	props := azureARMObject(payload["properties"])
	if size, _ := props["cacheSizeGB"].(float64); !slices.Contains(sizes, size) {
		return fmt.Sprintf("properties.cacheSizeGB must be one of %v for %s.", sizes, sku)
	}
	if !azureStorageCacheSubnetID(props["subnet"]) {
		return "properties.subnet must be a subnet ID."
	}
	if mtu, present := azureARMObject(props["networkSettings"])["mtu"]; present {
		if value, ok := mtu.(float64); !ok || value != float64(int(value)) || value < 576 || value > 1500 {
			return "properties.networkSettings.mtu must be an integer from 576 to 1500."
		}
	}
	return ""
}

// azureStorageCacheAddresses returns n stable private addresses for rec.
func azureStorageCacheAddresses(rec *azureARMResource, n int) []any {
	sum := sha256.Sum256([]byte(azureARMKey(rec.ID)))
	out := make([]any, 0, n)
	for i := range n {
		out = append(out, fmt.Sprintf("10.%d.%d.%d", sum[0], sum[1], 10+i))
	}
	return out
}

// azureStorageCacheProperties reports the cache's mount addresses and an
// up-to-date firmware.
func azureStorageCacheProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"mountAddresses": azureStorageCacheAddresses(rec, 3),
		"upgradeStatus": map[string]any{
			"currentFirmwareVersion": "6.4.4",
			"firmwareUpdateStatus":   "unavailable",
			"lastFirmwareUpdate":     rec.CreatedAt.Format("2006-01-02T15:04:05Z"),
			"pendingFirmwareVersion": nil,
			"firmwareUpdateDeadline": nil,
		},
		"upgradeSettings": map[string]any{"upgradeScheduleEnabled": false},
	}
}

var azureStorageTargetTypes = []string{"nfs3", "clfs", "blobNfs", "unknown"}

func azureStorageTargetValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	targetType := azureARMString(props["targetType"])
	if !slices.Contains(azureStorageTargetTypes, targetType) {
		return "properties.targetType must be nfs3, clfs, blobNfs or unknown."
	}
	switch target := azureARMString(azureARMObject(props[targetType])["target"]); targetType {
	case "nfs3":
		if strings.TrimSpace(target) == "" {
			return "properties.nfs3.target must name the NFS host."
		}
	case "clfs", "blobNfs":
		if id := strings.ToLower(target); !strings.Contains(id, "/providers/microsoft.storage/storageaccounts/") || !strings.Contains(id, "/blobservices/default/containers/") {
			return "properties." + targetType + ".target must be a blob container ID."
		}
	}
	junctions, _ := props["junctions"].([]any)
	seen := map[string]bool{}
	for i, raw := range junctions {
		junction := azureARMObject(raw)
		path := azureARMString(junction["namespacePath"])
		if !strings.HasPrefix(path, "/") || seen[path] {
			return fmt.Sprintf("properties.junctions[%d].namespacePath must be a unique path starting with /.", i)
		}
		if targetType == "nfs3" && !strings.HasPrefix(azureARMString(junction["nfsExport"]), "/") {
			return fmt.Sprintf("properties.junctions[%d].nfsExport must be the NFS export path.", i)
		}
		seen[path] = true
	}
	return ""
}

// azureAMLFilesystemTiers maps each Managed Lustre SKU to its throughput
// per TiB in MB/s, which is also the capacity step and minimum in TiB.
var azureAMLFilesystemTiers = map[string]struct{ throughput, step float64 }{
	"AMLFS-Durable-Premium-40":  {40, 48},
	"AMLFS-Durable-Premium-125": {125, 16},
	"AMLFS-Durable-Premium-250": {250, 8},
	"AMLFS-Durable-Premium-500": {500, 4},
}

// azureAMLFilesystemSubnetSize checks a SKU and capacity and returns the
// number of subnet addresses the file system needs.
func azureAMLFilesystemSubnetSize(sku string, rawCapacity any) (int, string) {
	tier, ok := azureAMLFilesystemTiers[sku]
	if !ok {
		return 0, "sku.name must be AMLFS-Durable-Premium-40, -125, -250 or -500."
	}
	capacity, ok := rawCapacity.(float64)
	// ponytail: 4096 TiB cap stands in for per-SKU quotas.
	if !ok || capacity < tier.step || capacity > 4096 || capacity != float64(int(capacity/tier.step))*tier.step {
		return 0, fmt.Sprintf("storageCapacityTiB must be a multiple of %v for %s.", tier.step, sku)
	}
	// ponytail: a flat base plus one address per capacity step; Azure's real
	// sizing formula is unpublished.
	return 16 + int(capacity/tier.step), ""
}

var azureAMLFilesystemTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func azureAMLFilesystemValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if _, msg := azureAMLFilesystemSubnetSize(azureARMString(azureARMObject(payload["sku"])["name"]), props["storageCapacityTiB"]); msg != "" {
		return msg
	}
	if !azureStorageCacheSubnetID(props["filesystemSubnet"]) {
		return "properties.filesystemSubnet must be a subnet ID."
	}
	window := azureARMObject(props["maintenanceWindow"])
	if !slices.Contains([]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}, azureARMString(window["dayOfWeek"])) ||
		!azureAMLFilesystemTimePattern.MatchString(azureARMString(window["timeOfDayUTC"])) {
		return "properties.maintenanceWindow needs a dayOfWeek and a timeOfDayUTC as HH:MM."
	}
	hsm := azureARMObject(azureARMObject(props["hsm"])["settings"])
	if len(hsm) != 0 {
		for _, key := range []string{"container", "loggingContainer"} {
			if !strings.Contains(strings.ToLower(azureARMString(hsm[key])), "/blobservices/default/containers/") {
				return "properties.hsm.settings." + key + " must be a blob container ID."
			}
		}
	}
	return ""
}

// azureAMLFilesystemProperties reports the file system healthy with its
// provisioned throughput and the client mount details.
func azureAMLFilesystemProperties(rec *azureARMResource) map[string]any {
	tier := azureAMLFilesystemTiers[azureARMString(rec.SKU["name"])]
	capacity, _ := rec.Properties["storageCapacityTiB"].(float64)
	mgs := azureStorageCacheAddresses(rec, 1)[0].(string)
	return map[string]any{
		"health":                    map[string]any{"state": "Available", "statusCode": "Ok", "statusDescription": "amlFilesystem is ok"},
		"throughputProvisionedMBps": int(capacity * tier.throughput),
		"clientInfo": map[string]any{
			"mgsAddress":    mgs,
			"mountCommand":  "sudo mount -t lustre -o noatime,flock " + mgs + "@tcp:/lustrefs /lustre/" + rec.Name,
			"lustreVersion": "2.15.4_42_gd6b9b5b",
			"containerStorageInterface": map[string]any{
				"persistentVolumeClaim": "pvc-" + rec.Name,
				"persistentVolume":      "pv-" + rec.Name,
				"storageClass":          "sc-" + rec.Name,
			},
		},
	}
}
