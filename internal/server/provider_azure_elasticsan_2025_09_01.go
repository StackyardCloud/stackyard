package server

import (
	"cmp"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const azureElasticSan20250901APIVersion = "2025-09-01"

// Azure Elastic SAN (Microsoft.ElasticSan 2025-09-01) on the shared ARM
// handler: SANs, volume groups, volumes, volume snapshots and private
// endpoint connections, plus the SKU, private link and operations views.
// No storage exists: volumes report an iSCSI target nothing listens on.

const (
	esanNamespace     = "Microsoft.ElasticSan"
	esanMaxGroups     = 200  // volume groups per SAN
	esanMaxVolumes    = 1000 // volumes per volume group
	esanMaxSnapshots  = 1000 // snapshots per volume group
	esanMaxBaseTiB    = 100
	esanMaxExtended   = 600
	esanMaxVolumeGiB  = 65536
	esanIOPSPerTiB    = 5000
	esanMBpsPerTiB    = 200
	esanGiBPerTiB     = 1024
	esanISCSIPort     = 3260
	esanVolumeTypeKey = esanNamespace + "/elasticSans/volumegroups/volumes"
)

var (
	esanSKUs          = []string{"Premium_LRS", "Premium_ZRS"}
	esanEnabled       = []string{"Enabled", "Disabled"}
	esanProtocols     = []string{"Iscsi", "None"}
	esanEncryption    = []string{"EncryptionAtRestWithPlatformKey", "EncryptionAtRestWithCustomerManagedKey"}
	esanIdentityTypes = []string{"None", "SystemAssigned", "UserAssigned"}
	esanCreateSources = []string{"None", "VolumeSnapshot", "DiskSnapshot", "Disk", "DiskRestorePoint"}
	esanLinkStates    = []string{"Pending", "Approved", "Failed", "Rejected"}
)

func init() {
	ns := esanNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "elasticSans",
		Validate: esanValidateSAN,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			base, _ := esanInt(rec.Properties["baseSizeTiB"])
			extended, _ := esanInt(rec.Properties["extendedCapacitySizeTiB"])
			props := map[string]any{
				"totalSizeTiB": base + extended,
				"totalIops":    base * esanIOPSPerTiB,
				"totalMBps":    base * esanMBpsPerTiB,
			}
			if _, set := rec.Properties["publicNetworkAccess"]; !set {
				props["publicNetworkAccess"] = "Enabled"
			}
			return props
		},
		// ponytail: refreshed when a volume group changes, so volume writes
		// show in totalVolumeSizeGiB on the next volume group write.
		ChildProperties: func(s *Server, rec *azureARMResource) map[string]any {
			groups, volumes := s.esanCountLocked(rec.ID)
			return map[string]any{"volumeGroupCount": groups, "totalVolumeSizeGiB": volumes}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "elasticSans/volumegroups", Proxy: true,
		Validate: esanValidateGroup,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := map[string]any{}
			if _, set := rec.Properties["protocolType"]; !set {
				props["protocolType"] = "Iscsi"
			}
			if _, set := rec.Properties["encryption"]; !set {
				props["encryption"] = "EncryptionAtRestWithPlatformKey"
			}
			return props
		},
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			return esanSiblingLimit(s, existing, rec, "volume groups", esanMaxGroups)
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"prebackup":  esanPreBackup,
			"prerestore": esanPreRestore,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "elasticSans/volumegroups/volumes", Proxy: true,
		Validate: esanValidateVolume,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			san := "es-" + azureHZGUID(esanSANID(rec.ID))[:12]
			return map[string]any{
				"volumeId": azureHZGUID(rec.ID),
				"storageTarget": map[string]any{
					"targetIqn":            "iqn.2023-05.net.windows.core.blob.elasticsan." + san + ":" + strings.ToLower(rec.Name),
					"targetPortalHostname": san + ".z1.blob.storage.azure.net",
					"targetPortalPort":     esanISCSIPort,
					"provisioningState":    "Succeeded",
					"status":               "Running",
				},
			}
		},
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			return esanSiblingLimit(s, existing, rec, "volumes", esanMaxVolumes)
		},
		Reconcile: esanReconcileVolume,
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			for _, snap := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
				return strings.EqualFold(r.Type, esanNamespace+"/elasticSans/volumegroups/snapshots")
			}) {
				if strings.EqualFold(azureARMString(azureARMObject(snap.Properties["creationData"])["sourceId"]), existing.ID) {
					return "Volume " + existing.Name + " has snapshot " + snap.Name + "; delete it first."
				}
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "elasticSans/volumegroups/snapshots", Proxy: true,
		Validate: func(payload map[string]any) string {
			if strings.TrimSpace(azureARMString(azureARMObject(azureARMObject(payload["properties"])["creationData"])["sourceId"])) == "" {
				return "properties.creationData.sourceId is required."
			}
			return ""
		},
		NoPatch: true,
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			return esanSiblingLimit(s, existing, rec, "snapshots", esanMaxSnapshots)
		},
		Reconcile: esanReconcileSnapshot,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "elasticSans/privateEndpointConnections", Proxy: true,
		Validate: func(payload map[string]any) string {
			state, ok := azureARMObject(payload["properties"])["privateLinkServiceConnectionState"].(map[string]any)
			if !ok {
				return "properties.privateLinkServiceConnectionState is required."
			}
			return azureLabEnum(state, "properties.privateLinkServiceConnectionState.", "status", esanLinkStates)
		},
	})
	// A private endpoint targets one volume group: each is a group ID.
	registerAzureARMView(ns, "elasticSans/privateLinkResources", func(s *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, group := range s.azureARMResourcesWhere(func(r *azureARMResource) bool {
			return strings.EqualFold(r.Type, ns+"/elasticSans/volumegroups") && strings.HasPrefix(azureARMKey(r.ID), azureARMKey(scope+"/"))
		}) {
			out = append(out, map[string]any{
				"id": scope + "/privateLinkResources/" + group.Name, "name": group.Name, "type": ns + "/elasticSans/privateLinkResources",
				"properties": map[string]any{
					"groupId": group.Name, "requiredMembers": []any{group.Name},
					"requiredZoneNames": []any{"privatelink.blob.core.windows.net"},
				},
			})
		}
		return out
	})
	registerAzureARMView(ns, "skus", func(*Server, string) []map[string]any {
		var out []map[string]any
		for _, name := range esanSKUs {
			out = append(out, map[string]any{
				"name": name, "tier": "Premium", "resourceType": "elasticSans",
				"locations": []any{"eastus", "westus2", "westeurope"},
				"capabilities": []any{
					map[string]any{"name": "MaxSizeTiB", "value": "100"},
					map[string]any{"name": "MaxIops", "value": "500000"},
				},
			})
		}
		return out
	})
	registerAzureARMView(ns, "operations", elasOperations("Microsoft Elastic SAN", ns, "elasticSans", "elasticSans/volumegroups", "elasticSans/volumegroups/volumes", "elasticSans/volumegroups/snapshots"))
}

// esanInt returns raw as a whole number, if it is one: a JSON number, or
// an int64 the server stored.
func esanInt(raw any) (int64, bool) {
	if n, ok := raw.(int64); ok {
		return n, true
	}
	f, ok := raw.(float64)
	if !ok || f != float64(int64(f)) {
		return 0, false
	}
	return int64(f), true
}

// esanIntIn checks that props[key] is a whole number in [least, most].
func esanIntIn(props map[string]any, key string, least, most int64, required bool) string {
	raw, present := props[key]
	if !present {
		if required {
			return "properties." + key + " is required."
		}
		return ""
	}
	if n, ok := esanInt(raw); !ok || n < least || n > most {
		return "properties." + key + " is invalid: it must be a whole number from " + itoa64(least) + " to " + itoa64(most) + "."
	}
	return ""
}

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

func esanValidateSAN(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	sku, ok := props["sku"].(map[string]any)
	if !ok {
		return "properties.sku is required."
	}
	return cmp.Or(
		azureWorkloadsEnums(sku, "properties.sku.", []string{"name"}, map[string][]string{"name": esanSKUs}),
		esanIntIn(props, "baseSizeTiB", 1, esanMaxBaseTiB, true),
		esanIntIn(props, "extendedCapacitySizeTiB", 0, esanMaxExtended, true),
		azureLabEnum(props, "properties.", "publicNetworkAccess", esanEnabled),
	)
}

func esanValidateGroup(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	return cmp.Or(
		azureLabEnum(azureARMObject(payload["identity"]), "identity.", "type", esanIdentityTypes),
		azureLabEnum(props, "properties.", "protocolType", esanProtocols),
		azureLabEnum(props, "properties.", "encryption", esanEncryption),
	)
}

func esanValidateVolume(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	creation := azureARMObject(props["creationData"])
	msg := cmp.Or(
		esanIntIn(props, "sizeGiB", 1, esanMaxVolumeGiB, true),
		azureLabEnum(creation, "properties.creationData.", "createSource", esanCreateSources),
	)
	if msg == "" && azureARMString(creation["createSource"]) != "" && azureARMString(creation["createSource"]) != "None" && strings.TrimSpace(azureARMString(creation["sourceId"])) == "" {
		msg = "properties.creationData.sourceId is required when createSource is " + azureARMString(creation["createSource"]) + "."
	}
	return msg
}

// esanSANID returns the elastic SAN ID a nested resource ID sits under.
func esanSANID(id string) string {
	lower := strings.ToLower(id)
	i := strings.Index(lower, "/elasticsans/")
	if i < 0 {
		return id
	}
	end := i + len("/elasticsans/")
	if j := strings.Index(lower[end:], "/"); j >= 0 {
		end += j
	} else {
		end = len(id)
	}
	return id[:end]
}

// esanParentID returns the ID of the resource two segments up.
func esanParentID(id string) string {
	for range 2 {
		id = id[:max(strings.LastIndex(id, "/"), 0)]
	}
	return id
}

// esanSiblingLimit rejects creating one more than limit children of a parent.
func esanSiblingLimit(s *Server, existing, rec *azureARMResource, what string, limit int) (string, string) {
	if existing != nil {
		return "", ""
	}
	prefix := azureARMKey(esanParentID(rec.ID) + "/")
	n := len(s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		return strings.EqualFold(r.Type, rec.Type) && strings.HasPrefix(azureARMKey(r.ID), prefix)
	}))
	if n >= limit {
		return "QuotaExceeded", "The parent already has the maximum of " + itoa64(int64(limit)) + " " + what + "."
	}
	return "", ""
}

// esanCountLocked returns a SAN's volume group count and total volume size.
func (s *Server) esanCountLocked(sanID string) (int, int64) {
	prefix := azureARMKey(sanID + "/")
	groups, size := 0, int64(0)
	for _, r := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool { return strings.HasPrefix(azureARMKey(r.ID), prefix) }) {
		switch {
		case strings.EqualFold(r.Type, esanNamespace+"/elasticSans/volumegroups"):
			groups++
		case strings.EqualFold(r.Type, esanVolumeTypeKey):
			n, _ := esanInt(r.Properties["sizeGiB"])
			size += n
		}
	}
	return groups, size
}

// esanReconcileVolume keeps a SAN's volumes within its capacity and checks
// that a volume created from a snapshot names one that exists.
func esanReconcileVolume(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	size, _ := esanInt(rec.Properties["sizeGiB"])
	if existing != nil {
		if old, _ := esanInt(existing.Properties["sizeGiB"]); size < old {
			return nil, "properties.sizeGiB is invalid: a volume can't shrink from " + itoa64(old) + " GiB."
		}
	}
	san := s.azureARMResources[azureARMKey(esanSANID(rec.ID))]
	if san == nil {
		return nil, "The elastic SAN for volume " + rec.Name + " is missing."
	}
	_, used := s.esanCountLocked(san.ID)
	if existing != nil {
		old, _ := esanInt(existing.Properties["sizeGiB"])
		used -= old
	}
	base, _ := esanInt(san.Properties["baseSizeTiB"])
	extended, _ := esanInt(san.Properties["extendedCapacitySizeTiB"])
	capacity := base + extended
	if used+size > capacity*esanGiBPerTiB {
		return nil, "properties.sizeGiB is invalid: the elastic SAN has " + itoa64(capacity*esanGiBPerTiB-used) + " GiB free."
	}
	creation := azureARMObject(rec.Properties["creationData"])
	if azureARMString(creation["createSource"]) == "VolumeSnapshot" {
		if s.azureARMResources[azureARMKey(azureARMString(creation["sourceId"]))] == nil {
			return nil, "properties.creationData.sourceId is invalid: snapshot " + azureARMString(creation["sourceId"]) + " was not found."
		}
	}
	return nil, ""
}

// esanReconcileSnapshot records the size and name of the volume a snapshot copies.
func esanReconcileSnapshot(s *Server, _, rec *azureARMResource) (map[string]any, string) {
	sourceID := azureARMString(azureARMObject(rec.Properties["creationData"])["sourceId"])
	source := s.azureARMResources[azureARMKey(sourceID)]
	if source == nil || !strings.EqualFold(source.Type, esanVolumeTypeKey) {
		return nil, "properties.creationData.sourceId is invalid: volume " + sourceID + " was not found."
	}
	if !strings.EqualFold(esanSANID(source.ID), esanSANID(rec.ID)) {
		return nil, "properties.creationData.sourceId is invalid: the volume belongs to another elastic SAN."
	}
	size, _ := esanInt(source.Properties["sizeGiB"])
	return map[string]any{"sourceVolumeSizeGiB": size, "volumeName": source.Name}, ""
}

// esanPreBackup checks that the named volumes exist in the volume group.
func esanPreBackup(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	names := azureARMSlice(payload["volumeNames"])
	if len(names) == 0 {
		return azureOracleBad("volumeNames is required.")
	}
	if len(names) > esanMaxVolumes {
		return azureOracleBad("volumeNames is invalid: at most " + itoa64(esanMaxVolumes) + " volumes.")
	}
	for _, raw := range names {
		name, _ := raw.(string)
		if vol, _ := s.getAzureARMResource(rec.ID+"/volumes/"+name, "", "", ""); vol == nil || name == "" {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "Volume '"+name+"' was not found in volume group "+rec.Name+".")
		}
	}
	return http.StatusOK, map[string]any{"validationStatus": "Success"}
}

// esanPreRestore checks the disk snapshots a restore would read.
func esanPreRestore(_ *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	snapshots := azureARMSlice(payload["diskSnapshotIds"])
	if len(snapshots) == 0 {
		return azureOracleBad("diskSnapshotIds is required.")
	}
	if len(snapshots) > esanMaxVolumes {
		return azureOracleBad("diskSnapshotIds is invalid: at most " + itoa64(esanMaxVolumes) + " snapshots.")
	}
	for _, raw := range snapshots {
		if id, _ := raw.(string); !azureNCIDRE("Microsoft.Compute/snapshots").MatchString(id) {
			return azureOracleBad("diskSnapshotIds is invalid: '" + azureARMString(raw) + "' isn't a disk snapshot resource ID.")
		}
	}
	return http.StatusOK, map[string]any{"validationStatus": "Success", "checkedAt": time.Now().UTC().Format(time.RFC3339), "volumeGroup": rec.Name}
}
