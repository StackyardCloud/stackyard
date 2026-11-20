package server

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	azureOracleADBType    = "autonomousDatabases"
	azureOracleBackupType = azureOracleADBType + "/autonomousDatabaseBackups"
	// azureOracleAutoRetention is the emulator's retention, in days, for
	// the automatic backup each database starts with.
	azureOracleAutoRetention = 60
)

var (
	// azureOracleADBFixed never changes after creation.
	azureOracleADBFixed = []string{"dataBaseType", "characterSet", "ncharacterSet", "computeModel", "dbWorkload", "dbVersion",
		"subnetId", "vnetId", "sourceId", "cloneType", "source", "timestamp", "useLatestAvailableBackupTimeStamp"}
	// azureOracleADBUpdatable is Learn's Autonomous Database update model.
	azureOracleADBUpdatable = []string{"adminPassword", "autonomousMaintenanceScheduleType", "backupRetentionPeriodInDays", "computeCount",
		"cpuCoreCount", "customerContacts", "dataStorageSizeInGbs", "dataStorageSizeInTbs", "databaseEdition", "displayName",
		"isAutoScalingEnabled", "isAutoScalingForStorageEnabled", "isLocalDataGuardEnabled", "isMtlsConnectionRequired", "licenseModel",
		"localAdgAutoFailoverMaxDataLossLimit", "longTermBackupSchedule", "openMode", "peerDbId", "permissionLevel", "role",
		"scheduledOperationsList", "whitelistedIps"}
)

func azureOracleRegisterADB() {
	ns := azureOracleNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureOracleADBType,
		ValidateName: func(name string) string {
			if name == "" || len(name) > 30 {
				return "An Autonomous Database name is 1 to 30 characters."
			}
			return ""
		},
		Validate:            azureOracleADBValidate,
		WriteOnlyProperties: []string{"adminPassword"},
		PatchBody:           azureOracleADBPatch,
		Reconcile:           azureOracleADBReconcile,
		DefaultChildren:     azureOracleADBBackups,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"action":                              azureOracleADBLifecycle,
			"failover":                            azureOracleADBRoleChange("timeOfLastFailover"),
			"switchover":                          azureOracleADBRoleChange("timeOfLastSwitchover"),
			"restore":                             azureOracleADBRestore,
			"shrink":                              azureOracleADBShrink,
			"changedisasterrecoveryconfiguration": azureOracleADBDisasterRecovery,
		},
		Actions: map[string]azureARMAction{"generatewallet": azureOracleWallet},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureOracleBackupType, Proxy: true,
		Validate:  azureOracleBackupValidate,
		Reconcile: azureOracleBackupReconcile,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if msg := azureOracleOnly(payload, "", []string{"properties"}); msg != "" {
				return nil, nil, msg
			}
			props := azureARMObject(payload["properties"])
			if msg := azureOracleOnly(props, "properties.", []string{"retentionPeriodInDays"}); msg != "" {
				return nil, nil, msg
			}
			return payload, nil, azureOracleNumber(props, "properties.", "retentionPeriodInDays", 1, 36500, true)
		},
		ValidatePatch: func(_ *Server, existing *azureARMResource, _ map[string]any) string {
			return azureOracleManualBackup(existing)
		},
		ValidateDelete: func(_ *Server, existing *azureARMResource) string { return azureOracleManualBackup(existing) },
	})
}

func azureOracleADBValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	kind := azureARMString(props["dataBaseType"])
	switch kind {
	case "", "Regular":
	case "Clone":
		if azureARMString(props["sourceId"]) == "" {
			return "A Clone needs properties.sourceId, the Autonomous Database to clone."
		}
		if msg := azureLabEnum(props, "properties.", "cloneType", []string{"Full", "Metadata"}); msg != "" || props["cloneType"] == nil {
			return "A Clone needs properties.cloneType: Full or Metadata."
		}
	case "CloneFromBackupTimestamp":
		if azureARMString(props["sourceId"]) == "" {
			return "A CloneFromBackupTimestamp needs properties.sourceId."
		}
		_, hasTime := props["timestamp"]
		if hasTime == (props["useLatestAvailableBackupTimeStamp"] == true) {
			return "A CloneFromBackupTimestamp needs either properties.timestamp or useLatestAvailableBackupTimeStamp, not both."
		}
		if hasTime {
			at, err := time.Parse(time.RFC3339, azureARMString(props["timestamp"]))
			if err != nil || !at.Before(time.Now()) {
				return "properties.timestamp must be an RFC 3339 time in the past."
			}
		}
	case "CrossRegionDisasterRecovery":
		// ponytail: one emulator has one region set; cross-region peers
		// would need a second.
		return "properties.dataBaseType CrossRegionDisasterRecovery isn't supported by the emulator."
	default:
		return "properties.dataBaseType must be one of Regular, Clone, CloneFromBackupTimestamp, CrossRegionDisasterRecovery."
	}
	checks := []string{
		azureLabEnum(props, "properties.", "computeModel", []string{"ECPU", "OCPU"}),
		azureLabEnum(props, "properties.", "dbWorkload", []string{"OLTP", "DW", "AJD", "APEX", "LH"}),
		azureLabLength(props, "properties.", "characterSet", 1, 255),
		azureLabLength(props, "properties.", "ncharacterSet", 1, 255),
		azureLabLength(props, "properties.", "dbVersion", 1, 255),
		azureOracleNetwork(props, false),
	}
	for _, msg := range checks {
		if msg != "" {
			return msg
		}
	}
	return azureOracleADBFields(props)
}

// azureOracleADBFields checks the fields PUT and PATCH share, with Learn's
// ranges.
func azureOracleADBFields(props map[string]any) string {
	checks := []string{
		azureLabLength(props, "properties.", "adminPassword", 12, 30),
		azureLabLength(props, "properties.", "displayName", 1, 255),
		azureOracleNumber(props, "properties.", "computeCount", 0.1, 512, false),
		azureOracleNumber(props, "properties.", "cpuCoreCount", 1, 128, true),
		azureOracleNumber(props, "properties.", "dataStorageSizeInGbs", 20, 393216, true),
		azureOracleNumber(props, "properties.", "dataStorageSizeInTbs", 1, 384, true),
		azureOracleNumber(props, "properties.", "localAdgAutoFailoverMaxDataLossLimit", 0, 3600, true),
		azureOracleNumber(props, "properties.", "backupRetentionPeriodInDays", 1, 36500, true),
		azureLabEnum(props, "properties.", "licenseModel", []string{"LicenseIncluded", "BringYourOwnLicense"}),
		azureLabEnum(props, "properties.", "permissionLevel", []string{"Restricted", "Unrestricted"}),
		azureLabEnum(props, "properties.", "openMode", []string{"ReadOnly", "ReadWrite"}),
		azureLabEnum(props, "properties.", "role", []string{"Primary", "Standby", "DisabledStandby", "BackupCopy", "SnapshotStandby"}),
		azureLabEnum(props, "properties.", "databaseEdition", []string{"StandardEdition", "EnterpriseEdition"}),
		azureLabEnum(props, "properties.", "autonomousMaintenanceScheduleType", []string{"Early", "Regular"}),
		azureOracleBools(props, "properties.", "isAutoScalingEnabled", "isAutoScalingForStorageEnabled", "isLocalDataGuardEnabled", "isMtlsConnectionRequired"),
		azureOracleContacts(props, 0),
		azureOracleWhitelist(props),
	}
	for _, msg := range checks {
		if msg != "" {
			return msg
		}
	}
	return ""
}

func azureOracleADBPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureOracleOnly(payload, "", []string{"tags", "properties"}); msg != "" {
		return nil, nil, msg
	}
	if msg := azureDataBoxObjects(payload, "", "tags", "properties"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureOracleOnly(props, "properties.", azureOracleADBUpdatable); msg != "" {
		return nil, nil, msg
	}
	return payload, nil, azureOracleADBFields(props)
}

// azureOracleADBReconcile checks the catalog values and a clone's source,
// and fills the defaults and connection details. What a database is made
// of (character sets, workload, network, source) never changes.
func azureOracleADBReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	out := map[string]any{}
	// What the service sets is read-only: a PUT may echo it back unchanged.
	for _, key := range slices.Sorted(maps.Keys(props)) {
		if !azureOracleADBServerOwned(key) {
			continue
		}
		if existing == nil || !azureOracleSame(props[key], existing.Properties[key]) {
			return nil, "properties." + key + " is read-only."
		}
	}
	if existing != nil {
		for _, key := range azureOracleADBFixed {
			if v, present := props[key]; present && !azureOracleSame(v, existing.Properties[key]) {
				return nil, "properties." + key + " can't be changed after creation."
			}
			if old, present := existing.Properties[key]; present {
				out[key] = old
			}
		}
		// Everything an update leaves out keeps its value.
		for key, old := range existing.Properties {
			if _, sent := props[key]; !sent && key != "provisioningState" {
				out[key] = old
			}
		}
		return out, ""
	}
	if msg := s.azureOracleEntitledLocked(rec.SubscriptionID); msg != "" {
		return nil, msg
	}
	source := "None"
	if id := azureARMString(props["sourceId"]); id != "" {
		src := s.azureARMResources[azureARMKey(id)]
		if src == nil || !strings.EqualFold(src.Type, azureOracleNamespace+"/"+azureOracleADBType) {
			return nil, "The source Autonomous Database '" + id + "' was not found."
		}
		source = "Database"
		if props["dataBaseType"] == "CloneFromBackupTimestamp" {
			source = "BackupFromTimestamp"
			if at, err := time.Parse(time.RFC3339, azureARMString(props["timestamp"])); err == nil && at.Before(src.CreatedAt) {
				return nil, "properties.timestamp is before the source database existed, so no backup covers it."
			}
		}
		// A clone keeps its source's character sets and version.
		for _, key := range []string{"characterSet", "ncharacterSet", "dbVersion"} {
			if props[key] == nil {
				out[key] = src.Properties[key]
			}
		}
	}
	defaults := map[string]any{
		"dataBaseType": "Regular", "computeModel": "ECPU", "dbWorkload": "OLTP", "licenseModel": "LicenseIncluded", "characterSet": "AL32UTF8",
		"ncharacterSet": "AL16UTF16", "dbVersion": "19c", "permissionLevel": "Unrestricted", "openMode": "ReadWrite", "role": "Primary",
		"isMtlsConnectionRequired": true, "isAutoScalingEnabled": false, "isAutoScalingForStorageEnabled": false, "isLocalDataGuardEnabled": false,
	}
	if props["dataStorageSizeInGbs"] == nil && props["dataStorageSizeInTbs"] == nil {
		defaults["dataStorageSizeInTbs"] = 1.0
	}
	if props["computeCount"] == nil && props["cpuCoreCount"] == nil {
		defaults["computeCount"] = 2.0
	}
	for key, value := range defaults {
		if _, present := props[key]; !present && out[key] == nil {
			out[key] = value
		}
	}
	merged := azureARMMerge(props, out)
	for key, values := range map[string][]string{"characterSet": azureOracleCharacterSets, "ncharacterSet": azureOracleNCharSets, "dbVersion": azureOracleADBVersions} {
		if !slices.Contains(values, azureARMString(merged[key])) {
			return nil, "properties." + key + " must be one of " + strings.Join(values, ", ") + "; see the location's catalog."
		}
	}
	host := strings.ToLower(rec.Name) + "-" + azureWorkloadsHash(rec.ID, 6) + ".adb." + strings.ToLower(rec.Location) + ".oraclecloudapps.com"
	if azureARMString(props["subnetId"]) != "" {
		out["privateEndpoint"], out["privateEndpointIp"], out["privateEndpointLabel"] = host, "10.0.2.10", strings.ToLower(rec.Name)
	}
	service := strings.ToLower(rec.Name)
	return azureARMMerge(out, map[string]any{
		"ocid": azureOracleOCID("autonomousdatabase", rec.ID), "lifecycleState": "Available", "source": source,
		"timeCreated": rec.CreatedAt.Format(time.RFC3339), "provisionableCpus": []any{2.0, 4.0, 8.0, 16.0},
		"serviceConsoleUrl":  "https://" + host + "/console/index.html",
		"sqlWebDeveloperUrl": "https://" + host + "/ords/sql-developer",
		"connectionStrings": map[string]any{"high": host + ":1522/" + service + "_high.adb.oraclecloud.com", "medium": host + ":1522/" + service + "_medium.adb.oraclecloud.com",
			"low": host + ":1522/" + service + "_low.adb.oraclecloud.com"},
		"connectionUrls": map[string]any{"apexUrl": "https://" + host + "/ords/apex", "databaseTransformsUrl": "https://" + host + "/odi/", "graphStudioUrl": "https://" + host + "/graphstudio/",
			"machineLearningUserManagementUrl": "https://" + host + "/omlusers/", "sqlDevWebUrl": "https://" + host + "/ords/sql-developer"},
	}), ""
}

// azureOracleADBBackups is the automatic backup a new database takes.
func azureOracleADBBackups(adb *azureARMResource) []*azureARMResource {
	typ := azureARMResourceTypes[strings.ToLower(azureOracleNamespace+"/"+azureOracleBackupType)]
	at := adb.CreatedAt
	return []*azureARMResource{azureARMChild(adb, typ, "automatic-"+at.Format("20060102150405"), map[string]any{
		"ocid": azureOracleOCID("autonomousdatabasebackup", adb.ID+"#auto"), "autonomousDatabaseOcid": adb.Properties["ocid"],
		"displayName": "Automatic backup " + at.Format(time.DateOnly), "isAutomatic": true, "isRestorable": true, "backupType": "Full",
		"lifecycleState": "Active", "retentionPeriodInDays": azureOracleAutoRetention, "dbVersion": adb.Properties["dbVersion"],
		"timeStarted": at.Format(time.RFC3339), "timeEnded": at.Format(time.RFC3339),
		"timeAvailableTil": at.AddDate(0, 0, azureOracleAutoRetention).Format(time.RFC3339), "backupDestination": map[string]any{"type": "ObjectStorage"},
	})}
}

func azureOracleADBLockedLookup(s *Server, id string) (*azureARMResource, int, any) {
	adb := s.azureARMResources[azureARMKey(id)]
	if adb == nil {
		status, body := http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+id+"' was not found.")
		return nil, status, body
	}
	return adb, 0, nil
}

// azureOracleADBLifecycle starts, stops or restarts a database.
func azureOracleADBLifecycle(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	action := azureARMString(payload["action"])
	from, to := map[string]string{"Start": "Stopped", "Stop": "Available", "Restart": "Available"}[action], "Available"
	if from == "" {
		return azureOracleBad("action must be one of Start, Stop, Restart.")
	}
	if action == "Stop" {
		to = "Stopped"
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	adb, status, body := azureOracleADBLockedLookup(s, rec.ID)
	if adb == nil {
		return status, body
	}
	if state := azureARMString(adb.Properties["lifecycleState"]); state != from {
		return azureOracleConflict(action + " needs a database that is " + from + "; this one is " + state + ".")
	}
	return http.StatusOK, s.azureOracleUpdateLocked(rec.ID, map[string]any{"lifecycleState": to}).fixture()
}

// azureOracleADBRoleChange runs a Data Guard failover or switchover, which
// needs a standby: local Data Guard, or the peer the request names.
func azureOracleADBRoleChange(stamp string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		adb, status, body := azureOracleADBLockedLookup(s, rec.ID)
		if adb == nil {
			return status, body
		}
		peer := azureARMString(payload["peerDbId"])
		peers := azureARMSlice(adb.Properties["peerDbIds"])
		hasPeer := peer != "" && (strings.EqualFold(peer, azureARMString(adb.Properties["peerDbId"])) ||
			slices.ContainsFunc(peers, func(v any) bool { return strings.EqualFold(azureARMString(v), peer) }))
		if !hasPeer && adb.Properties["isLocalDataGuardEnabled"] != true {
			return azureOracleConflict("This database has no standby: enable properties.isLocalDataGuardEnabled or name its peer in peerDbId.")
		}
		if state := azureARMString(adb.Properties["lifecycleState"]); state != "Available" {
			return azureOracleConflict("A role change needs an Available database; this one is " + state + ".")
		}
		now := time.Now().UTC().Format(time.RFC3339)
		return http.StatusOK, s.azureOracleUpdateLocked(rec.ID, map[string]any{stamp: now, "timeDataGuardRoleChanged": now}).fixture()
	}
}

// azureOracleADBRestore restores to a point between the database's
// creation and now.
func azureOracleADBRestore(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	at, err := time.Parse(time.RFC3339, azureARMString(payload["timestamp"]))
	if err != nil {
		return azureOracleBad("timestamp is required: an RFC 3339 time to restore the database to.")
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	adb, status, body := azureOracleADBLockedLookup(s, rec.ID)
	if adb == nil {
		return status, body
	}
	if at.Before(adb.CreatedAt.Truncate(time.Second)) || at.After(time.Now()) {
		return azureOracleBad("timestamp must be between the database's creation, " + adb.CreatedAt.Format(time.RFC3339) + ", and now.")
	}
	return http.StatusOK, s.azureOracleUpdateLocked(rec.ID, map[string]any{"lifecycleState": "Available", "timeOfLastRefreshPoint": at.UTC().Format(time.RFC3339)}).fixture()
}

// azureOracleADBShrink reclaims unused storage; an emulated database uses
// none, so it reports no used storage.
func azureOracleADBShrink(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	adb, status, body := azureOracleADBLockedLookup(s, rec.ID)
	if adb == nil {
		return status, body
	}
	if state := azureARMString(adb.Properties["lifecycleState"]); state != "Available" {
		return azureOracleConflict("Shrink needs an Available database; this one is " + state + ".")
	}
	return http.StatusOK, s.azureOracleUpdateLocked(rec.ID, map[string]any{"actualUsedDataStorageSizeInTbs": 0.0, "usedDataStorageSizeInGbs": 0}).fixture()
}

// azureOracleADBDisasterRecovery records the disaster recovery settings.
// Only a standby can become a snapshot standby.
func azureOracleADBDisasterRecovery(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureLabEnum(payload, "", "disasterRecoveryType", []string{"Adg", "BackupBased"}) + azureOracleBools(payload, "", "isReplicateAutomaticBackups", "isSnapshotStandby"); msg != "" {
		return azureOracleBad(msg)
	}
	if raw, present := payload["timeSnapshotStandbyEnabledTill"]; present {
		if _, err := time.Parse(time.RFC3339, azureARMString(raw)); err != nil {
			return azureOracleBad("timeSnapshotStandbyEnabledTill must be an RFC 3339 time.")
		}
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	adb, status, body := azureOracleADBLockedLookup(s, rec.ID)
	if adb == nil {
		return status, body
	}
	if payload["isSnapshotStandby"] == true && adb.Properties["role"] != "Standby" {
		return azureOracleConflict("Only a standby database can become a snapshot standby; this one is " + azureARMString(adb.Properties["role"]) + ".")
	}
	sent := map[string]any{}
	for _, key := range []string{"disasterRecoveryType", "isSnapshotStandby", "timeSnapshotStandbyEnabledTill", "isReplicateAutomaticBackups"} {
		if v, present := payload[key]; present {
			sent[key] = v
		}
	}
	config := azureARMMerge(azureARMObject(adb.Properties["remoteDisasterRecoveryConfiguration"]), sent)
	return http.StatusOK, s.azureOracleUpdateLocked(rec.ID, map[string]any{"remoteDisasterRecoveryConfiguration": config}).fixture()
}

// azureOracleWallet returns the client credentials as a base64 zip. The
// password protects the wallet's keys and is never stored.
func azureOracleWallet(rec *azureARMResource, payload map[string]any) (int, any) {
	if len(azureARMString(payload["password"])) < 8 {
		return azureOracleBad("password is required: at least 8 characters to encrypt the wallet's keys.")
	}
	if msg := azureLabEnum(payload, "", "generateType", []string{"Single", "All"}) + azureOracleBools(payload, "", "isRegional"); msg != "" {
		return azureOracleBad(msg)
	}
	var tns strings.Builder
	for _, level := range []string{"high", "medium", "low"} {
		tns.WriteString(strings.ToLower(rec.Name) + "_" + level + " = " + azureARMString(azureARMObject(rec.Properties["connectionStrings"])[level]) + "\n")
	}
	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	for name, text := range map[string]string{"tnsnames.ora": tns.String(), "sqlnet.ora": "WALLET_LOCATION = (SOURCE = (METHOD = file) (METHOD_DATA = (DIRECTORY=\"?/network/admin\")))\nSSL_SERVER_DN_MATCH=yes\n"} {
		f, _ := archive.Create(name) // Writes to a bytes.Buffer don't fail.
		_, _ = f.Write([]byte(text))
	}
	_ = archive.Close()
	return http.StatusOK, map[string]any{"walletFiles": base64.StdEncoding.EncodeToString(buf.Bytes())}
}

func azureOracleBackupValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if raw, present := props["displayName"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.displayName must be a string."
		}
	}
	return azureOracleNumber(props, "properties.", "retentionPeriodInDays", 1, 36500, true)
}

// azureOracleBackupReconcile takes a manual backup of an Available
// database: long-term when it has a retention period, full otherwise.
func azureOracleBackupReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if existing != nil {
		if msg := azureOracleManualBackup(existing); msg != "" {
			return nil, msg
		}
		out := azureARMMerge(existing.Properties, nil)
		for _, key := range []string{"displayName", "retentionPeriodInDays"} {
			if v, present := rec.Properties[key]; present {
				out[key] = v
			}
		}
		return azureOracleBackupAvailability(out), ""
	}
	adbID := azureWorkloadsParentID(rec.ID)
	adb := s.azureARMResources[azureARMKey(adbID)]
	if adb == nil {
		return nil, "The Autonomous Database '" + adbID + "' was not found."
	}
	if state := azureARMString(adb.Properties["lifecycleState"]); state != "Available" {
		// ponytail: Reconcile errors are 400; Learn doesn't say which status.
		return nil, "A backup needs an Available database; this one is " + state + "."
	}
	kind := "Full"
	if rec.Properties["retentionPeriodInDays"] != nil {
		kind = "LongTerm"
	}
	at := rec.CreatedAt.Format(time.RFC3339)
	return azureOracleBackupAvailability(azureARMMerge(rec.Properties, map[string]any{
		"ocid": azureOracleOCID("autonomousdatabasebackup", rec.ID), "autonomousDatabaseOcid": adb.Properties["ocid"], "isAutomatic": false,
		"isRestorable": true, "backupType": kind, "lifecycleState": "Active", "dbVersion": adb.Properties["dbVersion"],
		"timeStarted": at, "timeEnded": at, "backupDestination": map[string]any{"type": "ObjectStorage"},
	})), ""
}

func azureOracleBackupAvailability(props map[string]any) map[string]any {
	if days, ok := props["retentionPeriodInDays"].(float64); ok {
		if ended, err := time.Parse(time.RFC3339, azureARMString(props["timeEnded"])); err == nil {
			props["timeAvailableTil"] = ended.AddDate(0, 0, int(days)).Format(time.RFC3339)
		}
	}
	return props
}

// azureOracleManualBackup refuses changes to automatic backups, which the
// service manages.
func azureOracleManualBackup(existing *azureARMResource) string {
	if existing.Properties["isAutomatic"] == true {
		return "Automatic backups are managed by the service and can't be changed or deleted."
	}
	return ""
}

// azureOracleADBServerOwned reports whether the service, not the client,
// sets a database property.
func azureOracleADBServerOwned(key string) bool {
	switch key {
	case "ocid", "lifecycleState", "source", "timeCreated", "provisionableCpus", "serviceConsoleUrl", "sqlWebDeveloperUrl",
		"connectionStrings", "connectionUrls", "privateEndpoint", "privateEndpointIp", "privateEndpointLabel", "peerDbIds",
		"remoteDisasterRecoveryConfiguration":
		return true
	}
	return strings.HasPrefix(key, "timeOfLast") || strings.Contains(key, "UsedDataStorage") || strings.HasPrefix(key, "usedDataStorage")
}
