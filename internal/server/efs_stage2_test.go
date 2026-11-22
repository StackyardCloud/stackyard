package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestEFSStage2Policies(t *testing.T) {
	_, ts := newEFSTestServer(t)
	fsID := efsCreateFS(t, ts, `{"CreationToken":"pol"}`)
	base := "file-systems/" + fsID

	efsCallError(t, ts, http.MethodGet, base+"/policy", "", http.StatusNotFound, "PolicyNotFound")
	efsCallError(t, ts, http.MethodPut, base+"/policy", `{"Policy":"not json"}`, http.StatusBadRequest, "InvalidPolicyException")
	efsCallError(t, ts, http.MethodPut, base+"/policy", `{"Policy":"{\"Version\":\"2012-10-17\"}"}`, http.StatusBadRequest, "InvalidPolicyException")
	efsCallError(t, ts, http.MethodPut, base+"/policy", `{}`, http.StatusBadRequest, "BadRequest")
	policy := `{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"AWS\":\"*\"},\"Action\":\"elasticfilesystem:ClientMount\"}]}`
	put := efsCall(t, ts, http.MethodPut, base+"/policy", `{"Policy":"`+policy+`"}`, http.StatusOK)
	if put["FileSystemId"] != fsID || !strings.Contains(put["Policy"].(string), "ClientMount") {
		t.Fatalf("put policy: %v", put)
	}
	efsCall(t, ts, http.MethodGet, base+"/policy", "", http.StatusOK)
	efsCall(t, ts, http.MethodDelete, base+"/policy", "", http.StatusOK)
	efsCallError(t, ts, http.MethodGet, base+"/policy", "", http.StatusNotFound, "PolicyNotFound")

	// Backup policy: defaults off, only ENABLED/DISABLED can be set.
	if out := efsCall(t, ts, http.MethodGet, base+"/backup-policy", "", http.StatusOK); out["BackupPolicy"].(map[string]any)["Status"] != "DISABLED" {
		t.Fatalf("default backup: %v", out)
	}
	efsCallError(t, ts, http.MethodPut, base+"/backup-policy", `{"BackupPolicy":{"Status":"ENABLING"}}`, http.StatusBadRequest, "ValidationException")
	efsCallError(t, ts, http.MethodPut, base+"/backup-policy", `{}`, http.StatusBadRequest, "ValidationException")
	if out := efsCall(t, ts, http.MethodPut, base+"/backup-policy", `{"BackupPolicy":{"Status":"ENABLED"}}`, http.StatusOK); out["BackupPolicy"].(map[string]any)["Status"] != "ENABLED" {
		t.Fatalf("put backup: %v", out)
	}

	// Lifecycle: one transition per policy, each kind once, archive needs elastic.
	if out := efsCall(t, ts, http.MethodGet, base+"/lifecycle-configuration", "", http.StatusOK); len(out["LifecyclePolicies"].([]any)) != 0 {
		t.Fatalf("default lifecycle: %v", out)
	}
	efsCallError(t, ts, http.MethodPut, base+"/lifecycle-configuration", `{"LifecyclePolicies":[{"TransitionToIA":"AFTER_2_DAYS"}]}`, http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodPut, base+"/lifecycle-configuration", `{"LifecyclePolicies":[{"TransitionToIA":"AFTER_7_DAYS","TransitionToPrimaryStorageClass":"AFTER_1_ACCESS"}]}`, http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodPut, base+"/lifecycle-configuration", `{"LifecyclePolicies":[{"TransitionToIA":"AFTER_7_DAYS"},{"TransitionToIA":"AFTER_14_DAYS"}]}`, http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodPut, base+"/lifecycle-configuration", `{"LifecyclePolicies":[{"TransitionToArchive":"AFTER_90_DAYS"}]}`, http.StatusBadRequest, "BadRequest")
	out := efsCall(t, ts, http.MethodPut, base+"/lifecycle-configuration", `{"LifecyclePolicies":[{"TransitionToIA":"AFTER_30_DAYS"},{"TransitionToPrimaryStorageClass":"AFTER_1_ACCESS"}]}`, http.StatusOK)
	if len(out["LifecyclePolicies"].([]any)) != 2 {
		t.Fatalf("put lifecycle: %v", out)
	}
	efsCall(t, ts, http.MethodPut, "file-systems/"+fsID, `{"ThroughputMode":"elastic"}`, http.StatusAccepted)
	efsCall(t, ts, http.MethodPut, base+"/lifecycle-configuration", `{"LifecyclePolicies":[{"TransitionToArchive":"AFTER_90_DAYS"}]}`, http.StatusOK)
	efsCall(t, ts, http.MethodPut, base+"/lifecycle-configuration", `{"LifecyclePolicies":[]}`, http.StatusOK)
}

func TestEFSStage2Replication(t *testing.T) {
	_, ts := newEFSTestServer(t)
	src := efsCreateFS(t, ts, `{"CreationToken":"src","Encrypted":true}`)

	efsCallError(t, ts, http.MethodPost, "file-systems/"+src+"/replication-configuration", `{"Destinations":[]}`, http.StatusBadRequest, "ValidationException")
	rep := efsCall(t, ts, http.MethodPost, "file-systems/"+src+"/replication-configuration", `{"Destinations":[{"Region":"us-west-2"}]}`, http.StatusOK)
	dest := rep["Destinations"].([]any)[0].(map[string]any)
	destID := dest["FileSystemId"].(string)
	if dest["Status"] != "ENABLING" || dest["Region"] != "us-west-2" || rep["SourceFileSystemId"] != src {
		t.Fatalf("create replication: %v", rep)
	}
	efsCallError(t, ts, http.MethodPost, "file-systems/"+src+"/replication-configuration", `{"Destinations":[{"Region":"us-west-2"}]}`, http.StatusBadRequest, "ValidationException")

	// The destination exists, is encrypted like its source and is locked.
	desc := efsCall(t, ts, http.MethodGet, "file-systems?FileSystemId="+destID, "", http.StatusOK)["FileSystems"].([]any)[0].(map[string]any)
	if desc["Encrypted"] != true || desc["FileSystemProtection"].(map[string]any)["ReplicationOverwriteProtection"] != "REPLICATING" {
		t.Fatalf("destination: %v", desc)
	}
	efsCallError(t, ts, http.MethodPut, "file-systems/"+destID+"/protection", `{"ReplicationOverwriteProtection":"DISABLED"}`, http.StatusConflict, "ReplicationAlreadyExists")
	efsCallError(t, ts, http.MethodDelete, "file-systems/"+src, "", http.StatusConflict, "FileSystemInUse")

	// Described from either side, and ENABLED once reported.
	for _, id := range []string{src, destID} {
		got := efsCall(t, ts, http.MethodGet, "file-systems/replication-configurations?FileSystemId="+id, "", http.StatusOK)
		if d := got["Replications"].([]any)[0].(map[string]any)["Destinations"].([]any)[0].(map[string]any); d["Status"] != "ENABLED" {
			t.Fatalf("describe via %s: %v", id, got)
		}
	}
	other := efsCreateFS(t, ts, `{"CreationToken":"other"}`)
	efsCallError(t, ts, http.MethodGet, "file-systems/replication-configurations?FileSystemId="+other, "", http.StatusNotFound, "ReplicationNotFound")

	// An existing destination must have overwrite protection disabled.
	efsCallError(t, ts, http.MethodPost, "file-systems/"+other+"/replication-configuration", `{"Destinations":[{"FileSystemId":"`+src+`"}]}`, http.StatusBadRequest, "ValidationException")
	third := efsCreateFS(t, ts, `{"CreationToken":"third"}`)
	efsCallError(t, ts, http.MethodPost, "file-systems/"+other+"/replication-configuration", `{"Destinations":[{"FileSystemId":"`+third+`"}]}`, http.StatusConflict, "ConflictException")
	efsCall(t, ts, http.MethodPut, "file-systems/"+third+"/protection", `{"ReplicationOverwriteProtection":"DISABLED"}`, http.StatusOK)
	efsCallError(t, ts, http.MethodPut, "file-systems/"+third+"/protection", `{"ReplicationOverwriteProtection":"REPLICATING"}`, http.StatusBadRequest, "BadRequest")
	efsCall(t, ts, http.MethodPost, "file-systems/"+other+"/replication-configuration", `{"Destinations":[{"FileSystemId":"`+third+`"}]}`, http.StatusOK)

	all := efsCall(t, ts, http.MethodGet, "file-systems/replication-configurations", "", http.StatusOK)
	if len(all["Replications"].([]any)) != 2 {
		t.Fatalf("all replications: %v", all)
	}

	// Deleting from the destination side frees both; the destination is protected again.
	efsCall(t, ts, http.MethodDelete, "file-systems/"+destID+"/replication-configuration", "", http.StatusNoContent)
	efsCallError(t, ts, http.MethodDelete, "file-systems/"+src+"/replication-configuration", "", http.StatusNotFound, "ReplicationNotFound")
	desc = efsCall(t, ts, http.MethodGet, "file-systems?FileSystemId="+destID, "", http.StatusOK)["FileSystems"].([]any)[0].(map[string]any)
	if desc["FileSystemProtection"].(map[string]any)["ReplicationOverwriteProtection"] != "ENABLED" {
		t.Fatalf("destination after delete: %v", desc)
	}
	efsCall(t, ts, http.MethodDelete, "file-systems/"+src, "", http.StatusNoContent)
}

func TestEFSStage2AccountPreferencesAndPaging(t *testing.T) {
	_, ts := newEFSTestServer(t)
	prefs := efsCall(t, ts, http.MethodGet, "account-preferences", "", http.StatusOK)
	if prefs["ResourceIdPreference"].(map[string]any)["ResourceIdType"] != "LONG_ID" {
		t.Fatalf("default preferences: %v", prefs)
	}
	long := efsCreateFS(t, ts, `{"CreationToken":"long"}`)
	if len(long) != len("fs-")+17 {
		t.Fatalf("long ID %q", long)
	}
	efsCallError(t, ts, http.MethodPut, "account-preferences", `{"ResourceIdType":"MEDIUM"}`, http.StatusBadRequest, "BadRequest")
	efsCallError(t, ts, http.MethodPut, "account-preferences", `{}`, http.StatusBadRequest, "BadRequest")
	efsCall(t, ts, http.MethodPut, "account-preferences", `{"ResourceIdType":"SHORT_ID"}`, http.StatusOK)
	short := efsCreateFS(t, ts, `{"CreationToken":"short"}`)
	if len(short) != len("fs-")+8 {
		t.Fatalf("short ID %q", short)
	}
	efsCreateFS(t, ts, `{"CreationToken":"third"}`)

	page := efsCall(t, ts, http.MethodGet, "file-systems?MaxItems=2", "", http.StatusOK)
	next, _ := page["NextMarker"].(string)
	if len(page["FileSystems"].([]any)) != 2 || next == "" {
		t.Fatalf("page 1: %v", page)
	}
	rest := efsCall(t, ts, http.MethodGet, "file-systems?MaxItems=2&Marker="+next, "", http.StatusOK)
	if len(rest["FileSystems"].([]any)) != 1 || rest["Marker"] != next || rest["NextMarker"] != nil {
		t.Fatalf("page 2: %v", rest)
	}
	efsCallError(t, ts, http.MethodGet, "file-systems?Marker=bogus", "", http.StatusBadRequest, "BadRequest")
}
