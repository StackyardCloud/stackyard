package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"
)

// Microsoft.FileShares: top-level NFS file shares on the provisioned v2
// billing model, with snapshots, private endpoint connections and
// redundancy conversion. No data is stored: a share is its provisioning.
// Limits and formulas follow the Azure Files provisioned v2 documentation.
const azureFileSharesNamespace = "Microsoft.FileShares"

const (
	azureFileShareMinStorage    = 32
	azureFileShareMaxStorage    = 262144
	azureFileShareMinIOPS       = 3000
	azureFileShareMaxIOPS       = 102400
	azureFileShareMinThroughput = 100
	azureFileShareMaxThroughput = 10340
	azureFileShareGuardrail     = 5
	azureFileShareMaxShares     = 10000 // per subscription and region
	azureFileShareMaxSnapshots  = 200   // per share
	// azureFileShareDowngradeWait is how long after an increase a
	// provisioned quantity can't decrease.
	azureFileShareDowngradeWait = 24 * time.Hour
	// azureFileShareConversionType stores a share's latest redundancy
	// conversion as a record of an unregistered type under the share.
	azureFileShareConversionType = azureFileSharesNamespace + "/fileShares/redundancyConversions"
)

// azureFileShareQuantities pairs each provisioned quantity with the
// property holding the time it may next decrease.
var azureFileShareQuantities = [][2]string{
	{"provisionedStorageGiB", "provisionedStorageNextAllowedDowngrade"},
	{"provisionedIOPerSec", "provisionedIOPerSecNextAllowedDowngrade"},
	{"provisionedThroughputMiBPerSec", "provisionedThroughputNextAllowedDowngrade"},
}

func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureFileSharesNamespace, Type: "fileShares",
		ValidateName:   azureFileShareNameError,
		Validate:       azureFileShareValidate,
		PatchBody:      azureFileSharePatch,
		Reconcile:      azureFileShareReconcile,
		InlineChildren: map[string]string{"privateEndpointConnections": "fileShares/privateEndpointConnections"},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"startredundancyconversion":     azureFileShareStartConversion,
			"getredundancyconversionstatus": azureFileShareConversionStatus,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureFileSharesNamespace, Type: "fileShares/fileShareSnapshots", Proxy: true,
		Validate: azureFileShareSnapshotValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"snapshotTime": rec.CreatedAt.Format(time.RFC3339Nano)}
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
				return nil, nil, msg
			}
			props := azureARMObject(payload["properties"])
			for key := range props {
				if key != "metadata" {
					return nil, nil, "properties." + key + " can't be updated."
				}
			}
			return payload, nil, azureFileShareMetadata(props)
		},
		Reconcile: azureFileShareSnapshotReconcile,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureFileSharesNamespace, Type: "fileShares/privateEndpointConnections", Proxy: true,
		Validate: azureFileShareConnectionValidate,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"provisioningState": "Succeeded", "groupIds": []any{"fileshare"}}
		},
		NoPatch: true,
	})
	registerAzureARMView(azureFileSharesNamespace, "fileShares/privateLinkResources", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/privateLinkResources/fileshare", "name": "fileshare", "type": azureFileSharesNamespace + "/privateLinkResources",
			"properties": map[string]any{"groupId": "fileshare", "requiredMembers": []any{"file"}, "requiredZoneNames": []any{"privatelink.file.core.windows.net"}},
		}}
	})
	registerAzureARMLocationAction(azureFileSharesNamespace, "checkNameAvailability", azureFileShareCheckName)
	registerAzureARMLocationAction(azureFileSharesNamespace, "getLimits", azureFileShareLimits)
	registerAzureARMLocationAction(azureFileSharesNamespace, "getProvisioningRecommendation", azureFileShareRecommendation)
	registerAzureARMLocationAction(azureFileSharesNamespace, "getUsageData", func(s *Server, sub, location string, _ map[string]any) (int, any) {
		count := len(s.azureFileSharesIn(sub, location))
		return http.StatusOK, map[string]any{"properties": map[string]any{"liveShares": map[string]any{"fileShareCount": count}}}
	})
}

// azureFileShareRecommended returns the IOPS and throughput Azure Files
// recommends for an amount of provisioned storage.
func azureFileShareRecommended(storage int) (iops, throughput int) {
	iops = min(max(azureFileShareMinIOPS+storage, azureFileShareMinIOPS), azureFileShareMaxIOPS)
	throughput = min(max(azureFileShareMinThroughput+int(math.Ceil(0.1*float64(storage))), azureFileShareMinThroughput), azureFileShareMaxThroughput)
	return iops, throughput
}

// azureFileShareInt reads a whole number stored from JSON (float64) or by
// the emulator (int).
func azureFileShareInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case float64:
		return int(v), v == math.Trunc(v)
	case int:
		return v, true
	}
	return 0, false
}

// azureFileShareReconcile fills in defaults and recommendations, enforces
// the provisioned v2 limits, guardrails and 24-hour downgrade wait, and
// derives the burst and downgrade properties.
func azureFileShareReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	out := map[string]any{"provisioningState": "Succeeded", "migrationState": map[string]any{"redundancyConversionInProgress": false}}
	if existing == nil {
		shares := s.azureFileSharesInLocked(rec.SubscriptionID, rec.Location)
		if len(shares) >= azureFileShareMaxShares {
			return nil, fmt.Sprintf("A subscription can have at most %d file shares in a region.", azureFileShareMaxShares)
		}
		// Names are unique per subscription and region, as checkNameAvailability reports.
		if slices.ContainsFunc(shares, func(other *azureARMResource) bool { return strings.EqualFold(other.Name, rec.Name) }) {
			return nil, "A file share named '" + rec.Name + "' already exists in this subscription and region."
		}
		sum := sha256.Sum256([]byte(azureARMKey(rec.ID)))
		out["hostName"] = "fs-" + hex.EncodeToString(sum[:8]) + ".file.stackyard.invalid"
		for key, value := range map[string]any{"mediaTier": "SSD", "protocol": "NFS", "scaleMode": "Compatibility", "mountName": rec.Name} {
			if props[key] == nil {
				out[key] = value
			}
		}
	} else {
		out["hostName"] = existing.Properties["hostName"]
		for _, key := range azureFileShareFixed {
			sent, present := props[key]
			switch {
			case !present:
				out[key] = existing.Properties[key]
			case !reflect.DeepEqual(sent, existing.Properties[key]) && key == "redundancy":
				return nil, "properties.redundancy can't be updated; start a redundancy conversion instead."
			case !reflect.DeepEqual(sent, existing.Properties[key]):
				return nil, "properties." + key + " can't be updated."
			}
		}
	}

	storage, _ := azureFileShareInt(props["provisionedStorageGiB"])
	if storage < azureFileShareMinStorage || storage > azureFileShareMaxStorage {
		return nil, fmt.Sprintf("properties.provisionedStorageGiB must be from %d to %d.", azureFileShareMinStorage, azureFileShareMaxStorage)
	}
	recIOPS, recThroughput := azureFileShareRecommended(storage)
	values := map[string]int{"provisionedStorageGiB": storage}
	for key, limits := range map[string][3]int{
		"provisionedIOPerSec":            {recIOPS, azureFileShareMinIOPS, azureFileShareMaxIOPS},
		"provisionedThroughputMiBPerSec": {recThroughput, azureFileShareMinThroughput, azureFileShareMaxThroughput},
	} {
		// Omitted (including from a PUT that replaces the share), the
		// quantity is the recommendation.
		value, ok := azureFileShareInt(props[key])
		if !ok {
			value = limits[0]
		}
		most := min(azureFileShareGuardrail*limits[0], limits[2])
		if value < limits[1] || value > most {
			return nil, fmt.Sprintf("properties.%s must be from %d to %d for %d GiB of provisioned storage.", key, limits[1], most, storage)
		}
		values[key] = value
	}

	now := time.Now().UTC()
	for _, q := range azureFileShareQuantities {
		value, stamp := values[q[0]], now.Add(azureFileShareDowngradeWait).Format(time.RFC3339Nano)
		if existing != nil {
			old, _ := azureFileShareInt(existing.Properties[q[0]])
			last := azureARMString(existing.Properties[q[1]])
			wait, _ := time.Parse(time.RFC3339Nano, last)
			switch {
			case value < old && now.Before(wait):
				return nil, "properties." + q[0] + " can't decrease until " + last + ", 24 hours after its last increase."
			case value <= old:
				stamp = last
			}
		}
		out[q[0]], out[q[1]] = value, stamp
	}
	burst := min(max(3*values["provisionedIOPerSec"], 10000), azureFileShareMaxIOPS)
	out["includedBurstIOPerSec"] = burst
	out["maxBurstIOPerSecCredits"] = (burst - values["provisionedIOPerSec"]) * 3600
	return out, ""
}

// azureFileSharesIn returns the file shares in a subscription and region.
func (s *Server) azureFileSharesIn(sub, location string) []*azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureFileSharesInLocked(sub, location)
}

func (s *Server) azureFileSharesInLocked(sub, location string) []*azureARMResource {
	return s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return rec.Type == azureFileSharesNamespace+"/fileShares" && strings.EqualFold(rec.SubscriptionID, sub) &&
			strings.EqualFold(strings.ReplaceAll(rec.Location, " ", ""), strings.ReplaceAll(location, " ", ""))
	})
}

func azureFileShareSnapshotReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if existing != nil {
		return nil, ""
	}
	parent := rec.ID[:strings.LastIndex(rec.ID, "/fileShareSnapshots/")]
	typ, _ := lookupAzureARMResourceType(azureFileSharesNamespace, "fileShares/fileShareSnapshots")
	siblings := s.azureARMInlineChildrenLocked(parent, typ)
	if len(siblings) >= azureFileShareMaxSnapshots {
		return nil, fmt.Sprintf("A file share can have at most %d snapshots.", azureFileShareMaxSnapshots)
	}
	return nil, ""
}

// azureFileShareStartConversion converts a share to the target redundancy.
// ponytail: conversions complete at once; reporting InProgress first is
// the upgrade if clients need to poll.
func azureFileShareStartConversion(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	target := azureARMString(azureARMObject(payload["properties"])["targetRedundancy"])
	if target != "Local" && target != "Zone" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.targetRedundancy must be Local or Zone.")
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The resource '"+rec.ID+"' was not found.")
	}
	if current.Properties["redundancy"] == target {
		return http.StatusConflict, azureARMErrorBody("Conflict", "The file share's redundancy is already "+target+".")
	}
	now := time.Now().UTC()
	updated := *current
	updated.Properties = azureARMMerge(current.Properties, map[string]any{"redundancy": target})
	updated.UpdatedAt = now
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	id := rec.ID + "/redundancyConversions/latest"
	s.azureARMResources[azureARMKey(id)] = &azureARMResource{
		ID: id, SubscriptionID: rec.SubscriptionID, ResourceGroup: rec.ResourceGroup, Name: "latest", Type: azureFileShareConversionType,
		Properties: map[string]any{"targetRedundancy": target, "conversionStatus": "Complete"}, CreatedAt: now, UpdatedAt: now,
	}
	return http.StatusNoContent, nil
}

func azureFileShareConversionStatus(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	conversion := s.azureARMResources[azureARMKey(rec.ID+"/redundancyConversions/latest")]
	s.providerStorageMu.Unlock()
	if conversion == nil {
		return http.StatusNotFound, azureARMErrorBody("NoRedundancyConversion", "No redundancy conversion has been started for file share '"+rec.Name+"'.")
	}
	return http.StatusOK, map[string]any{"properties": conversion.Properties}
}

func azureFileShareCheckName(s *Server, sub, location string, payload map[string]any) (int, any) {
	name := azureARMString(payload["name"])
	if typ := azureARMString(payload["type"]); !strings.EqualFold(typ, azureFileSharesNamespace+"/fileShares") {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be Microsoft.FileShares/fileShares.")
	}
	switch {
	case azureFileShareNameError(name) != "":
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": azureFileShareNameError(name)}
	case slices.ContainsFunc(s.azureFileSharesIn(sub, location), func(rec *azureARMResource) bool { return strings.EqualFold(rec.Name, name) }):
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "A file share named '" + name + "' already exists."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}

func azureFileShareLimits(_ *Server, _, _ string, payload map[string]any) (int, any) {
	if _, msg := azureFileShareOffer(payload); msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	return http.StatusOK, map[string]any{"properties": map[string]any{
		"limits": map[string]any{
			"maxFileShares": azureFileShareMaxShares, "maxFileShareSnapshots": azureFileShareMaxSnapshots,
			"minProvisionedStorageGiB": azureFileShareMinStorage, "maxProvisionedStorageGiB": azureFileShareMaxStorage,
			"minProvisionedIOPerSec": azureFileShareMinIOPS, "maxProvisionedIOPerSec": azureFileShareMaxIOPS,
			"minProvisionedThroughputMiBPerSec": azureFileShareMinThroughput, "maxProvisionedThroughputMiBPerSec": azureFileShareMaxThroughput,
		},
		"provisioningConstants": map[string]any{
			"baseIOPerSec": azureFileShareMinIOPS, "scalarIOPerSec": 1.0,
			"baseThroughputMiBPerSec": azureFileShareMinThroughput, "scalarThroughputMiBPerSec": 0.1,
			"guardrailIOPerSecScalar": float64(azureFileShareGuardrail), "guardrailThroughputScalar": float64(azureFileShareGuardrail),
		},
	}}
}

func azureFileShareRecommendation(_ *Server, _, _ string, payload map[string]any) (int, any) {
	props, msg := azureFileShareOffer(payload)
	if msg != "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	storage, ok := azureFileShareInt(props["provisionedStorageGiB"])
	if !ok || storage < azureFileShareMinStorage || storage > azureFileShareMaxStorage {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", fmt.Sprintf("properties.provisionedStorageGiB must be from %d to %d.", azureFileShareMinStorage, azureFileShareMaxStorage))
	}
	iops, throughput := azureFileShareRecommended(storage)
	return http.StatusOK, map[string]any{"properties": map[string]any{
		"provisionedIOPerSec": iops, "provisionedThroughputMiBPerSec": throughput, "availableRedundancyOptions": []any{"Local", "Zone"},
	}}
}
