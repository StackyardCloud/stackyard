package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDynamoDBStage4BackupAndRestoreCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage4-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "PutItem", []byte(`{"TableName":"stage4-table","Item":{"pk":{"S":"backup#1"},"status":{"S":"ACTIVE"}}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "CreateBackup", []byte(`{"TableName":"stage4-table","BackupName":"stage4-backup"}`))
	assertStatus(t, resp, http.StatusOK)
	var createBackupOut struct {
		BackupDetails struct {
			BackupArn string `json:"BackupArn"`
		} `json:"BackupDetails"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createBackupOut); err != nil {
		t.Fatalf("unmarshal create backup response: %v", err)
	}
	if createBackupOut.BackupDetails.BackupArn == "" {
		t.Fatalf("expected BackupArn")
	}
	backupArn := createBackupOut.BackupDetails.BackupArn

	resp = dynamodbRequest(t, ts, "DescribeBackup", []byte(`{"BackupArn":"`+backupArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ListBackups", []byte(`{"TableName":"stage4-table","Limit":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "RestoreTableFromBackup", []byte(`{"TargetTableName":"stage4-restored","BackupArn":"`+backupArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage4-restored"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "RestoreTableToPointInTime", []byte(`{"SourceTableName":"stage4-table","TargetTableName":"stage4-pitr"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage4-pitr"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DeleteBackup", []byte(`{"BackupArn":"`+backupArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestDynamoDBStage4ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage4-implemented",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "CreateBackup", []byte(`{"TableName":"stage4-implemented","BackupName":"stage4-implemented-backup"}`))
	assertStatus(t, resp, http.StatusOK)
	var createBackupOut struct {
		BackupDetails struct {
			BackupArn string `json:"BackupArn"`
		} `json:"BackupDetails"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createBackupOut); err != nil {
		t.Fatalf("unmarshal create backup response: %v", err)
	}

	backupArn := createBackupOut.BackupDetails.BackupArn
	actions := []struct {
		action string
		body   []byte
	}{
		{action: "CreateBackup", body: []byte(`{"TableName":"stage4-implemented","BackupName":"stage4-implemented-backup-2"}`)},
		{action: "DescribeBackup", body: []byte(`{"BackupArn":"` + backupArn + `"}`)},
		{action: "DeleteBackup", body: []byte(`{"BackupArn":"` + backupArn + `"}`)},
		{action: "ListBackups", body: []byte(`{"TableName":"stage4-implemented","Limit":10}`)},
		{action: "RestoreTableFromBackup", body: []byte(`{"TargetTableName":"stage4-from-backup","BackupArn":"` + backupArn + `"}`)},
		{action: "RestoreTableToPointInTime", body: []byte(`{"SourceTableName":"stage4-implemented","TargetTableName":"stage4-from-pitr"}`)},
	}

	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}
