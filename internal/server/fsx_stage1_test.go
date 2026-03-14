package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFSxStage1MutationResponseShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fsxRequest(t, ts, "CreateBackup", `{"FileSystemId":"fs-0123456789abcdef0","BackupName":"stackyard-backup"}`)
	assertStatus(t, resp, http.StatusOK)
	var createBackupOut struct {
		Backup struct {
			BackupId   string         `json:"BackupId"`
			Lifecycle  string         `json:"Lifecycle"`
			Type       string         `json:"Type"`
			FileSystem map[string]any `json:"FileSystem"`
		} `json:"Backup"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createBackupOut); err != nil {
		t.Fatalf("unmarshal create backup response: %v", err)
	}
	if createBackupOut.Backup.BackupId == "" || createBackupOut.Backup.Lifecycle == "" || createBackupOut.Backup.Type == "" {
		t.Fatalf("expected CreateBackup to return modeled backup identifiers and lifecycle")
	}
	if createBackupOut.Backup.FileSystem["FileSystemId"] == nil {
		t.Fatalf("expected CreateBackup to include Backup.FileSystem")
	}

	resp = fsxRequest(t, ts, "CreateDataRepositoryTask", `{"FileSystemId":"fs-0123456789abcdef0","Type":"EXPORT_TO_REPOSITORY"}`)
	assertStatus(t, resp, http.StatusOK)
	var createTaskOut struct {
		DataRepositoryTask struct {
			TaskId       string `json:"TaskId"`
			Lifecycle    string `json:"Lifecycle"`
			Type         string `json:"Type"`
			CreationTime string `json:"CreationTime"`
		} `json:"DataRepositoryTask"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createTaskOut); err != nil {
		t.Fatalf("unmarshal create data repository task response: %v", err)
	}
	if createTaskOut.DataRepositoryTask.TaskId == "" ||
		createTaskOut.DataRepositoryTask.Lifecycle == "" ||
		createTaskOut.DataRepositoryTask.Type == "" ||
		createTaskOut.DataRepositoryTask.CreationTime == "" {
		t.Fatalf("expected CreateDataRepositoryTask to return required modeled fields")
	}

	resp = fsxRequest(t, ts, "UpdateFileSystem", `{"FileSystemId":"fs-0123456789abcdef0"}`)
	assertStatus(t, resp, http.StatusOK)
	var updateFileSystemOut struct {
		FileSystem map[string]any `json:"FileSystem"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateFileSystemOut); err != nil {
		t.Fatalf("unmarshal update file system response: %v", err)
	}
	if updateFileSystemOut.FileSystem["FileSystemId"] == nil {
		t.Fatalf("expected UpdateFileSystem to return FileSystem")
	}

	resp = fsxRequest(t, ts, "RestoreVolumeFromSnapshot", `{"VolumeId":"fsvol-0123456789abcdef0","SnapshotId":"snapshot-0123456789abcdef0"}`)
	assertStatus(t, resp, http.StatusOK)
	var restoreVolumeOut struct {
		VolumeId  string `json:"VolumeId"`
		Lifecycle string `json:"Lifecycle"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &restoreVolumeOut); err != nil {
		t.Fatalf("unmarshal restore volume response: %v", err)
	}
	if restoreVolumeOut.VolumeId == "" || restoreVolumeOut.Lifecycle == "" {
		t.Fatalf("expected RestoreVolumeFromSnapshot to return VolumeId and Lifecycle")
	}

	resp = fsxRequest(t, ts, "DeleteFileSystem", `{"FileSystemId":"fs-0123456789abcdef0"}`)
	assertStatus(t, resp, http.StatusOK)
	var deleteFileSystemOut struct {
		FileSystemId string `json:"FileSystemId"`
		Lifecycle    string `json:"Lifecycle"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &deleteFileSystemOut); err != nil {
		t.Fatalf("unmarshal delete file system response: %v", err)
	}
	if deleteFileSystemOut.FileSystemId == "" || deleteFileSystemOut.Lifecycle == "" {
		t.Fatalf("expected DeleteFileSystem to return FileSystemId and Lifecycle")
	}
}

func TestFSxStage1DescribeAndAliasResponseShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := fsxRequest(t, ts, "AssociateFileSystemAliases", `{"FileSystemId":"fs-0123456789abcdef0","Aliases":["files.example.com"]}`)
	assertStatus(t, resp, http.StatusOK)
	var associateAliasesOut struct {
		Aliases []map[string]any `json:"Aliases"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &associateAliasesOut); err != nil {
		t.Fatalf("unmarshal associate aliases response: %v", err)
	}
	if len(associateAliasesOut.Aliases) != 1 || associateAliasesOut.Aliases[0]["Lifecycle"] == nil {
		t.Fatalf("expected AssociateFileSystemAliases to return Aliases with lifecycle")
	}

	resp = fsxRequest(t, ts, "DescribeFileSystemAliases", `{"FileSystemId":"fs-0123456789abcdef0"}`)
	assertStatus(t, resp, http.StatusOK)
	var describeAliasesOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeAliasesOut); err != nil {
		t.Fatalf("unmarshal describe aliases response: %v", err)
	}
	if _, ok := describeAliasesOut["FileSystemAliases"]; ok {
		t.Fatalf("expected DescribeFileSystemAliases to use Aliases")
	}
	if _, ok := describeAliasesOut["Aliases"]; !ok {
		t.Fatalf("expected DescribeFileSystemAliases to return Aliases")
	}

	resp = fsxRequest(t, ts, "DescribeDataRepositoryAssociations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var describeAssociationsOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeAssociationsOut); err != nil {
		t.Fatalf("unmarshal describe data repository associations response: %v", err)
	}
	if _, ok := describeAssociationsOut["DataRepositoryAssociations"]; ok {
		t.Fatalf("expected DescribeDataRepositoryAssociations to use Associations")
	}
	if _, ok := describeAssociationsOut["Associations"]; !ok {
		t.Fatalf("expected DescribeDataRepositoryAssociations to return Associations")
	}

	resp = fsxRequest(t, ts, "DescribeSharedVpcConfiguration", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var sharedVpcOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &sharedVpcOut); err != nil {
		t.Fatalf("unmarshal describe shared vpc configuration response: %v", err)
	}
	if _, ok := sharedVpcOut["EnableFsxRouteTableUpdatesFromParticipantAccounts"]; !ok {
		t.Fatalf("expected DescribeSharedVpcConfiguration to return EnableFsxRouteTableUpdatesFromParticipantAccounts")
	}

	resp = fsxRequest(t, ts, "CreateAndAttachS3AccessPoint", `{"Name":"stackyard-s3-access-point","Type":"OPENZFS","OpenZFSConfiguration":{"VolumeId":"fsvol-0123456789abcdef0"}}`)
	assertStatus(t, resp, http.StatusOK)
	var createAccessPointOut struct {
		S3AccessPointAttachment map[string]any `json:"S3AccessPointAttachment"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createAccessPointOut); err != nil {
		t.Fatalf("unmarshal create and attach s3 access point response: %v", err)
	}
	if createAccessPointOut.S3AccessPointAttachment["Name"] == nil || createAccessPointOut.S3AccessPointAttachment["Type"] == nil {
		t.Fatalf("expected CreateAndAttachS3AccessPoint to return S3AccessPointAttachment")
	}
}
