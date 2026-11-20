package server

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"
)

func azureOraADB(props map[string]any) map[string]any {
	out := map[string]any{"displayName": "adb 1", "adminPassword": "Secret-Admin-123", "computeCount": 2.0, "dataStorageSizeInTbs": 1.0}
	for k, v := range props {
		out[k] = v
	}
	return map[string]any{"location": "eastus", "properties": out}
}

func TestAzureOracleAutonomousDatabase(t *testing.T) {
	t.Parallel()
	h := newAzureOraTest(t)
	adbs := h.base + "autonomousDatabases/"
	h.rejects(http.MethodPut, adbs+"db1", azureOraADB(nil), "no Oracle Database@Azure offer")
	h.subscribe()
	for name, c := range map[string]struct {
		props map[string]any
		text  string
	}{
		"password":    {map[string]any{"adminPassword": "short"}, "adminPassword"},
		"compute":     {map[string]any{"computeCount": 0.05}, "computeCount"},
		"cores":       {map[string]any{"cpuCoreCount": 129.0}, "cpuCoreCount"},
		"storage gb":  {map[string]any{"dataStorageSizeInGbs": 10.0}, "dataStorageSizeInGbs"},
		"storage tb":  {map[string]any{"dataStorageSizeInTbs": 385.0}, "dataStorageSizeInTbs"},
		"workload":    {map[string]any{"dbWorkload": "OLAP"}, "dbWorkload"},
		"model":       {map[string]any{"computeModel": "GPU"}, "computeModel"},
		"data loss":   {map[string]any{"localAdgAutoFailoverMaxDataLossLimit": 3601.0}, "localAdgAutoFailoverMaxDataLossLimit"},
		"ips":         {map[string]any{"whitelistedIps": []any{"1.2.3.4", "nope"}}, "whitelistedIps"},
		"charset":     {map[string]any{"characterSet": "KLINGON"}, "characterSet must be one of"},
		"version":     {map[string]any{"dbVersion": "11g"}, "dbVersion must be one of"},
		"subnet":      {map[string]any{"subnetId": "/subscriptions/x/resourceGroups/r/providers/Microsoft.Network/virtualNetworks/v/subnets/s"}, "vnetId"},
		"type":        {map[string]any{"dataBaseType": "Copy"}, "dataBaseType must be"},
		"cross":       {map[string]any{"dataBaseType": "CrossRegionDisasterRecovery"}, "isn't supported"},
		"clone":       {map[string]any{"dataBaseType": "Clone", "sourceId": h.arm + "autonomousDatabases/db1"}, "cloneType"},
		"clone src":   {map[string]any{"dataBaseType": "Clone", "cloneType": "Full", "sourceId": h.arm + "autonomousDatabases/missing"}, "was not found"},
		"pitr both":   {map[string]any{"dataBaseType": "CloneFromBackupTimestamp", "sourceId": "x", "timestamp": "2020-01-01T00:00:00Z", "useLatestAvailableBackupTimeStamp": true}, "not both"},
		"pitr future": {map[string]any{"dataBaseType": "CloneFromBackupTimestamp", "sourceId": "x", "timestamp": "2999-01-01T00:00:00Z"}, "in the past"},
	} {
		t.Run(name, func(t *testing.T) { h.rejects(http.MethodPut, adbs+"bad", azureOraADB(c.props), c.text) })
	}
	h.rejects(http.MethodPut, adbs+strings.Repeat("d", 31), azureOraADB(nil), "1 to 30")

	created := h.props(http.MethodPut, adbs+"db1", azureOraADB(map[string]any{"isLocalDataGuardEnabled": true}), http.StatusCreated)
	if created["lifecycleState"] != "Available" || created["dbWorkload"] != "OLTP" || created["characterSet"] != "AL32UTF8" || created["role"] != "Primary" ||
		created["adminPassword"] != nil || azureARMObject(created["connectionStrings"])["high"] == nil || created["source"] != "None" {
		t.Fatalf("adb = %#v", created)
	}
	if body := string(mustBody(t, h.do(http.MethodGet, adbs+"db1"+azureOraTestAPI, ""))); strings.Contains(body, "Secret-Admin-123") {
		t.Fatalf("admin password returned: %s", body)
	}
	// A re-PUT can't overwrite what the service owns.
	for key, value := range map[string]any{"ocid": "ocid1.forged", "lifecycleState": "Stopped", "timeOfLastFailover": "2020-01-01T00:00:00Z"} {
		h.rejects(http.MethodPut, adbs+"db1", azureOraADB(map[string]any{"isLocalDataGuardEnabled": true, key: value}), key+" is read-only")
	}
	// Echoing them back unchanged is fine.
	h.call(http.MethodPut, adbs+"db1", azureOraADB(map[string]any{"isLocalDataGuardEnabled": true, "ocid": created["ocid"], "lifecycleState": "Available"}), http.StatusOK)

	// Lifecycle actions follow the state.
	h.fails(http.MethodPost, adbs+"db1/action", map[string]any{"action": "Start"}, http.StatusConflict, "Stopped")
	h.rejects(http.MethodPost, adbs+"db1/action", map[string]any{"action": "Pause"}, "action must be")
	if got := h.props(http.MethodPost, adbs+"db1/action", map[string]any{"action": "Stop"}, http.StatusOK); got["lifecycleState"] != "Stopped" {
		t.Fatalf("stopped = %#v", got)
	}
	h.fails(http.MethodPost, adbs+"db1/shrink", nil, http.StatusConflict, "Available")
	h.rejects(http.MethodPut, adbs+"db1/autonomousDatabaseBackups/manual", map[string]any{}, "needs an Available database")
	h.call(http.MethodPost, adbs+"db1/action", map[string]any{"action": "Start"}, http.StatusOK)
	h.call(http.MethodPost, adbs+"db1/action", map[string]any{"action": "Restart"}, http.StatusOK)
	h.call(http.MethodPost, adbs+"db1/shrink", nil, http.StatusOK)

	// Updates follow Learn's update model; the database's make-up is fixed.
	h.rejects(http.MethodPatch, adbs+"db1", map[string]any{"properties": map[string]any{"characterSet": "UTF8"}}, "can't be updated")
	h.rejects(http.MethodPatch, adbs+"db1", map[string]any{"properties": map[string]any{"cpuCoreCount": 0.0}}, "cpuCoreCount")
	h.rejects(http.MethodPut, adbs+"db1", azureOraADB(map[string]any{"dbWorkload": "DW"}), "dbWorkload can't be changed")
	patched := h.props(http.MethodPatch, adbs+"db1", map[string]any{"properties": map[string]any{"computeCount": 4.0, "adminPassword": "Another-Secret-456", "whitelistedIps": []any{"10.0.0.0/24"}}}, http.StatusOK)
	if patched["computeCount"] != 4.0 || patched["adminPassword"] != nil || patched["ocid"] != created["ocid"] {
		t.Fatalf("patched = %#v", patched)
	}
	if again := h.props(http.MethodPut, adbs+"db1", azureOraADB(map[string]any{"displayName": "renamed"}), http.StatusOK); again["ocid"] != created["ocid"] ||
		again["lifecycleState"] != "Available" || again["isLocalDataGuardEnabled"] != true || again["displayName"] != "renamed" {
		t.Fatalf("re-PUT = %#v", again)
	}

	// Data Guard and recovery.
	if got := h.props(http.MethodPost, adbs+"db1/switchover", map[string]any{}, http.StatusOK); got["timeOfLastSwitchover"] == nil {
		t.Fatalf("switchover = %#v", got)
	}
	h.call(http.MethodPost, adbs+"db1/failover", map[string]any{}, http.StatusOK)
	h.rejects(http.MethodPost, adbs+"db1/restore", map[string]any{}, "timestamp is required")
	h.rejects(http.MethodPost, adbs+"db1/restore", map[string]any{"timestamp": "2001-01-01T00:00:00Z"}, "between the database's creation")
	h.call(http.MethodPost, adbs+"db1/restore", map[string]any{"timestamp": time.Now().UTC().Format(time.RFC3339)}, http.StatusOK)
	h.rejects(http.MethodPost, adbs+"db1/changeDisasterRecoveryConfiguration", map[string]any{"disasterRecoveryType": "Tape"}, "disasterRecoveryType")
	h.fails(http.MethodPost, adbs+"db1/changeDisasterRecoveryConfiguration", map[string]any{"isSnapshotStandby": true}, http.StatusConflict, "Only a standby")
	if got := h.props(http.MethodPost, adbs+"db1/changeDisasterRecoveryConfiguration", map[string]any{"disasterRecoveryType": "BackupBased", "isReplicateAutomaticBackups": true}, http.StatusOK); azureARMObject(got["remoteDisasterRecoveryConfiguration"])["disasterRecoveryType"] != "BackupBased" {
		t.Fatalf("dr config = %#v", got)
	}

	// The wallet is a zip of connection files; its password isn't kept.
	h.rejects(http.MethodPost, adbs+"db1/generateWallet", map[string]any{"password": "short"}, "at least 8")
	h.rejects(http.MethodPost, adbs+"db1/generateWallet", map[string]any{"password": "WalletPass1", "generateType": "Some"}, "generateType")
	wallet := h.call(http.MethodPost, adbs+"db1/generateWallet", map[string]any{"password": "WalletPass1", "generateType": "Single"}, http.StatusOK)
	raw, err := base64.StdEncoding.DecodeString(azureARMString(wallet["walletFiles"]))
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(archive.File) != 2 {
		t.Fatalf("wallet zip: %v", err)
	}

	// Standby-less databases can't change roles.
	h.call(http.MethodPut, adbs+"db2", azureOraADB(map[string]any{"dbWorkload": "DW", "characterSet": "UTF8", "computeModel": "OCPU"}), http.StatusCreated)
	h.fails(http.MethodPost, adbs+"db2/switchover", map[string]any{"peerDbId": "/x"}, http.StatusConflict, "no standby")

	// Clones take their source's character sets.
	clone := h.props(http.MethodPut, adbs+"db3", azureOraADB(map[string]any{"dataBaseType": "Clone", "cloneType": "Metadata", "sourceId": h.arm + "autonomousDatabases/db2"}), http.StatusCreated)
	if clone["characterSet"] != "UTF8" || clone["source"] != "Database" {
		t.Fatalf("clone = %#v", clone)
	}
	h.rejects(http.MethodPut, adbs+"db4", azureOraADB(map[string]any{"dataBaseType": "CloneFromBackupTimestamp", "sourceId": h.arm + "autonomousDatabases/db2", "timestamp": "2001-01-01T00:00:00Z"}), "before the source")
	pitr := h.props(http.MethodPut, adbs+"db4", azureOraADB(map[string]any{"dataBaseType": "CloneFromBackupTimestamp", "sourceId": h.arm + "autonomousDatabases/db2", "useLatestAvailableBackupTimeStamp": true}), http.StatusCreated)
	if pitr["source"] != "BackupFromTimestamp" {
		t.Fatalf("pitr clone = %#v", pitr)
	}
	if items := h.call(http.MethodGet, h.sub+"/providers/Oracle.Database/autonomousDatabases", nil, http.StatusOK)["value"].([]any); len(items) != 4 {
		t.Fatalf("subscription databases = %d", len(items))
	}
}

func TestAzureOracleAutonomousDatabaseBackups(t *testing.T) {
	t.Parallel()
	h := newAzureOraTest(t)
	h.subscribe()
	adb := h.base + "autonomousDatabases/db1"
	h.call(http.MethodPut, adb, azureOraADB(nil), http.StatusCreated)
	backups := adb + "/autonomousDatabaseBackups/"
	list := h.call(http.MethodGet, backups, nil, http.StatusOK)["value"].([]any)
	if len(list) != 1 || azureARMObject(azureARMObject(list[0])["properties"])["isAutomatic"] != true {
		t.Fatalf("backups = %#v", list)
	}
	auto := backups + azureARMString(azureARMObject(list[0])["name"])
	h.fails(http.MethodDelete, auto, nil, http.StatusConflict, "Automatic backups")
	h.fails(http.MethodPatch, auto, map[string]any{"properties": map[string]any{"retentionPeriodInDays": 90.0}}, http.StatusConflict, "Automatic backups")
	h.rejects(http.MethodPut, auto, map[string]any{}, "Automatic backups")

	h.rejects(http.MethodPut, backups+"manual", map[string]any{"properties": map[string]any{"retentionPeriodInDays": 0.0}}, "retentionPeriodInDays")
	full := h.props(http.MethodPut, backups+"full", map[string]any{"properties": map[string]any{"displayName": "before upgrade"}}, http.StatusCreated)
	if full["backupType"] != "Full" || full["isAutomatic"] != false || full["timeAvailableTil"] != nil {
		t.Fatalf("full backup = %#v", full)
	}
	long := h.props(http.MethodPut, backups+"long", map[string]any{"properties": map[string]any{"retentionPeriodInDays": 365.0}}, http.StatusCreated)
	if long["backupType"] != "LongTerm" || long["timeAvailableTil"] == nil {
		t.Fatalf("long-term backup = %#v", long)
	}
	extended := h.props(http.MethodPatch, backups+"long", map[string]any{"properties": map[string]any{"retentionPeriodInDays": 730.0}}, http.StatusOK)
	if extended["retentionPeriodInDays"] != 730.0 || extended["timeAvailableTil"] == long["timeAvailableTil"] || extended["ocid"] != long["ocid"] {
		t.Fatalf("extended backup = %#v", extended)
	}
	h.rejects(http.MethodPatch, backups+"long", map[string]any{"properties": map[string]any{"displayName": "x"}}, "can't be updated")
	h.call(http.MethodDelete, backups+"full", nil, http.StatusOK)
	h.call(http.MethodDelete, adb, nil, http.StatusOK)
	h.call(http.MethodGet, backups+"long", nil, http.StatusNotFound)
}
