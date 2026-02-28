package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage2SnapshotsRestoreExportAndAutomatedBackups(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage2-db"},
		"Engine":               []string{"postgres"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret123!"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create instance 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBSnapshot"},
		"DBSnapshotIdentifier": []string{"rds-stage2-snap"},
		"DBInstanceIdentifier": []string{"rds-stage2-db"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create snapshot 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBSnapshotIdentifier>rds-stage2-snap</DBSnapshotIdentifier>")) {
		t.Fatalf("missing snapshot identifier: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{"Action": []string{"DescribeDBSnapshots"}})
	if status != http.StatusOK {
		t.Fatalf("expected describe snapshots 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBSnapshots>")) {
		t.Fatalf("missing snapshots collection: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                     []string{"CopyDBSnapshot"},
		"SourceDBSnapshotIdentifier": []string{"rds-stage2-snap"},
		"TargetDBSnapshotIdentifier": []string{"rds-stage2-snap-copy"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected copy snapshot 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"RestoreDBInstanceFromDBSnapshot"},
		"DBInstanceIdentifier": []string{"rds-stage2-restore"},
		"DBSnapshotIdentifier": []string{"rds-stage2-snap"},
		"DBInstanceClass":      []string{"db.t3.small"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected restore from snapshot 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstanceIdentifier>rds-stage2-restore</DBInstanceIdentifier>")) {
		t.Fatalf("missing restored instance identifier: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                     []string{"RestoreDBInstanceToPointInTime"},
		"SourceDBInstanceIdentifier": []string{"rds-stage2-db"},
		"TargetDBInstanceIdentifier": []string{"rds-stage2-pitr"},
		"DBInstanceClass":            []string{"db.t3.micro"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected restore to point in time 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"StartExportTask"},
		"ExportTaskIdentifier": []string{"rds-stage2-export"},
		"SourceArn":            []string{"arn:aws:rds:us-east-1:123456789012:snapshot:rds-stage2-snap"},
		"S3BucketName":         []string{"demo-export-bucket"},
		"S3Prefix":             []string{"exports/rds"},
		"KmsKeyId":             []string{"alias/aws/rds"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected start export task 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ExportTaskIdentifier>rds-stage2-export</ExportTaskIdentifier>")) {
		t.Fatalf("missing export task identifier: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{"Action": []string{"DescribeExportTasks"}})
	if status != http.StatusOK {
		t.Fatalf("expected describe export tasks 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ExportTasks>")) {
		t.Fatalf("missing export tasks collection: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CancelExportTask"},
		"ExportTaskIdentifier": []string{"rds-stage2-export"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected cancel export task 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Status>canceled</Status>")) {
		t.Fatalf("expected canceled status: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"StartDBInstanceAutomatedBackupsReplication"},
		"SourceDBInstanceArn": []string{"arn:aws:rds:us-east-1:123456789012:db:rds-stage2-db"},
		"SourceRegion":        []string{"us-east-1"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected start automated backups replication 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"DescribeDBInstanceAutomatedBackups"},
		"DBInstanceIdentifier": []string{"rds-stage2-db"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe automated backups 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBInstanceAutomatedBackups>")) {
		t.Fatalf("missing automated backup collection: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"StopDBInstanceAutomatedBackupsReplication"},
		"SourceDBInstanceArn": []string{"arn:aws:rds:us-east-1:123456789012:db:rds-stage2-db"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected stop automated backups replication 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":        []string{"DeleteDBInstanceAutomatedBackup"},
		"DbiResourceId": []string{"dbi-rds-stage2-db"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected delete automated backup 200, got %d: %s", status, string(body))
	}

	for _, snapshotID := range []string{"rds-stage2-snap-copy", "rds-stage2-snap"} {
		status, body = rdsRequest(t, ts, url.Values{
			"Action":               []string{"DeleteDBSnapshot"},
			"DBSnapshotIdentifier": []string{snapshotID},
		})
		if status != http.StatusOK {
			t.Fatalf("expected delete snapshot %s 200, got %d: %s", snapshotID, status, string(body))
		}
	}
}

func TestRDSStage2ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage2-impl-db"},
		"Engine":               []string{"postgres"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret123!"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBSnapshot"},
		"DBSnapshotIdentifier": []string{"rds-stage2-impl-snap"},
		"DBInstanceIdentifier": []string{"rds-stage2-impl-db"},
	})

	cases := []url.Values{
		{"Action": []string{"DescribeDBSnapshots"}},
		{"Action": []string{"CopyDBSnapshot"}, "SourceDBSnapshotIdentifier": []string{"rds-stage2-impl-snap"}, "TargetDBSnapshotIdentifier": []string{"rds-stage2-impl-snap-copy"}},
		{"Action": []string{"RestoreDBInstanceFromDBSnapshot"}, "DBInstanceIdentifier": []string{"rds-stage2-impl-restore"}, "DBSnapshotIdentifier": []string{"rds-stage2-impl-snap"}},
		{"Action": []string{"RestoreDBInstanceToPointInTime"}, "SourceDBInstanceIdentifier": []string{"rds-stage2-impl-db"}, "TargetDBInstanceIdentifier": []string{"rds-stage2-impl-pitr"}},
		{"Action": []string{"StartExportTask"}, "ExportTaskIdentifier": []string{"rds-stage2-impl-export"}, "SourceArn": []string{"arn:aws:rds:us-east-1:123456789012:snapshot:rds-stage2-impl-snap"}, "S3BucketName": []string{"demo"}},
		{"Action": []string{"DescribeExportTasks"}},
		{"Action": []string{"CancelExportTask"}, "ExportTaskIdentifier": []string{"rds-stage2-impl-export"}},
		{"Action": []string{"StartDBInstanceAutomatedBackupsReplication"}, "SourceDBInstanceArn": []string{"arn:aws:rds:us-east-1:123456789012:db:rds-stage2-impl-db"}},
		{"Action": []string{"DescribeDBInstanceAutomatedBackups"}, "DBInstanceIdentifier": []string{"rds-stage2-impl-db"}},
		{"Action": []string{"StopDBInstanceAutomatedBackupsReplication"}, "SourceDBInstanceArn": []string{"arn:aws:rds:us-east-1:123456789012:db:rds-stage2-impl-db"}},
		{"Action": []string{"DeleteDBInstanceAutomatedBackup"}, "DbiResourceId": []string{"dbi-rds-stage2-impl-db"}},
		{"Action": []string{"DeleteDBSnapshot"}, "DBSnapshotIdentifier": []string{"rds-stage2-impl-snap-copy"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
