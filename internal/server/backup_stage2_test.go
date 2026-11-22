package server

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/services/dynamodb"
)

func backupTestTable(t *testing.T, srv *Server, name string) string {
	t.Helper()
	desc, err := srv.dynamodb.CreateTable(name, []dynamodb.AttributeDefinition{{AttributeName: "id", AttributeType: "S"}}, []dynamodb.KeySchemaElement{{AttributeName: "id", KeyType: "HASH"}}, "PAY_PER_REQUEST", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.dynamodb.PutItem(name, map[string]any{"id": map[string]any{"S": "1"}}, ""); err != nil {
		t.Fatal(err)
	}
	return desc.TableArn
}

func TestBackupStage2BackupCopyAndRestoreADynamoDBTable(t *testing.T) {
	srv, ts := newBackupTestServer(t)
	tableARN := backupTestTable(t, srv, "orders")
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/src", nil)
	dest := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/dst", nil)

	backupFail(t, ts, "ResourceNotFoundException", http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "src", "IamRoleArn": backupTestRole, "ResourceArn": "arn:aws:dynamodb:us-east-1:123456789012:table/missing"})
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "src", "IamRoleArn": backupTestRole, "ResourceArn": "not-an-arn"})
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "src", "IamRoleArn": "nope", "ResourceArn": tableARN})

	start := map[string]any{"BackupVaultName": "src", "IamRoleArn": backupTestRole, "ResourceArn": tableARN, "IdempotencyToken": "tok", "Lifecycle": map[string]any{"DeleteAfterDays": 7}}
	job := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-jobs", start)
	jobID, rpARN := job["BackupJobId"].(string), job["RecoveryPointArn"].(string)
	if again := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-jobs", start); again["BackupJobId"] != jobID {
		t.Fatalf("idempotency token must return the same job: %v", again)
	}
	rpPath := "/backup-vaults/src/recovery-points/" + url.PathEscape(rpARN)
	// A recovery point still being created can't feed a restore.
	backupFail(t, ts, "InvalidRequestException", http.MethodPut, "/restore-jobs", map[string]any{"RecoveryPointArn": rpARN, "Metadata": map[string]any{"targetTableName": "x"}})

	// Each read advances the job: RUNNING, then COMPLETED.
	for _, want := range []string{"RUNNING", "COMPLETED"} {
		if d := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-jobs/"+jobID, nil); d["State"] != want {
			t.Fatalf("job state: %v (want %s)", d["State"], want)
		}
	}
	rp := backupWant(t, ts, http.StatusOK, http.MethodGet, rpPath, nil)
	if rp["Status"] != "COMPLETED" || rp["ResourceType"] != "DynamoDB" || rp["ResourceName"] != "orders" || rp["CalculatedLifecycle"].(map[string]any)["DeleteAt"] == nil {
		t.Fatalf("recovery point: %v", rp)
	}
	backupFail(t, ts, "InvalidRequestException", http.MethodPost, "/backup-jobs/"+jobID, nil)
	pr := backupWant(t, ts, http.StatusOK, http.MethodGet, "/resources/"+url.PathEscape(tableARN), nil)
	if pr["LastRecoveryPointArn"] != rpARN {
		t.Fatalf("protected resource: %v", pr)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/resources/"+url.PathEscape(tableARN)+"/recovery-points/", nil); len(l["RecoveryPoints"].([]any)) != 1 {
		t.Fatalf("recovery points by resource: %v", l)
	}
	backupFail(t, ts, "InvalidRequestException", http.MethodDelete, "/backup-vaults/src", nil)

	copyJob := backupWant(t, ts, http.StatusOK, http.MethodPut, "/copy-jobs", map[string]any{"RecoveryPointArn": rpARN, "SourceBackupVaultName": "src", "DestinationBackupVaultArn": dest["BackupVaultArn"], "IamRoleArn": backupTestRole})
	copyID := copyJob["CopyJobId"].(string)
	backupWant(t, ts, http.StatusOK, http.MethodGet, "/copy-jobs/"+copyID, nil)
	copied := backupWant(t, ts, http.StatusOK, http.MethodGet, "/copy-jobs/"+copyID, nil)["CopyJob"].(map[string]any)
	if copied["State"] != "COMPLETED" || copied["DestinationRecoveryPointArn"] == nil {
		t.Fatalf("copy job: %v", copied)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/dst/recovery-points/", nil); len(l["RecoveryPoints"].([]any)) != 1 {
		t.Fatalf("copied recovery point: %v", l)
	}

	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/restore-jobs", map[string]any{"RecoveryPointArn": rpARN, "Metadata": map[string]any{}})
	restore := backupWant(t, ts, http.StatusOK, http.MethodPut, "/restore-jobs", map[string]any{"RecoveryPointArn": rpARN, "IamRoleArn": backupTestRole, "Metadata": map[string]any{"targetTableName": "orders_restored"}})
	restoreID := restore["RestoreJobId"].(string)
	backupFail(t, ts, "InvalidRequestException", http.MethodPut, "/restore-jobs/"+restoreID+"/validations", map[string]any{"ValidationStatus": "SUCCESSFUL"})
	backupWant(t, ts, http.StatusOK, http.MethodGet, "/restore-jobs/"+restoreID, nil)
	done := backupWant(t, ts, http.StatusOK, http.MethodGet, "/restore-jobs/"+restoreID, nil)
	if done["Status"] != "COMPLETED" || !strings.HasSuffix(done["CreatedResourceArn"].(string), "table/orders_restored") {
		t.Fatalf("restore job: %v", done)
	}
	// The restore really created the table, with the backed-up item.
	if desc, err := srv.dynamodb.DescribeTable("orders_restored"); err != nil || desc.ItemCount != 1 {
		t.Fatalf("restored table: %+v %v", desc, err)
	}
	backupWant(t, ts, http.StatusNoContent, http.MethodPut, "/restore-jobs/"+restoreID+"/validations", map[string]any{"ValidationStatus": "SUCCESSFUL"})
	if m := backupWant(t, ts, http.StatusOK, http.MethodGet, "/restore-jobs/"+restoreID+"/metadata", nil); m["Metadata"].(map[string]any)["targetTableName"] != "orders_restored" {
		t.Fatalf("restore metadata: %v", m)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/resources/"+url.PathEscape(tableARN)+"/restore-jobs/?status=COMPLETED", nil); len(l["RestoreJobs"].([]any)) != 1 {
		t.Fatalf("restore jobs by resource: %v", l)
	}

	sums := backupWant(t, ts, http.StatusOK, http.MethodGet, "/audit/backup-job-summaries", nil)
	if s := sums["BackupJobSummaries"].([]any); len(s) != 1 || s[0].(map[string]any)["Count"] != float64(1) || sums["AggregationPeriod"] != "ONE_DAY" {
		t.Fatalf("summaries: %v", sums)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-jobs/?state=COMPLETED&resourceType=DynamoDB", nil); len(l["BackupJobs"].([]any)) != 1 {
		t.Fatalf("filtered jobs: %v", l)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-jobs/?state=ABORTED", nil); len(l["BackupJobs"].([]any)) != 0 {
		t.Fatalf("state filter: %v", l)
	}

	// Legal holds block deletion; cancelling releases the recovery point.
	hold := backupWant(t, ts, http.StatusOK, http.MethodPost, "/legal-holds/", map[string]any{"Title": "audit", "Description": "keep", "RecoveryPointSelection": map[string]any{"VaultNames": []string{"src"}}})
	holdID := hold["LegalHoldId"].(string)
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/legal-holds/"+holdID+"/recovery-points", nil); len(l["RecoveryPoints"].([]any)) != 1 {
		t.Fatalf("held recovery points: %v", l)
	}
	backupFail(t, ts, "InvalidResourceStateException", http.MethodDelete, rpPath, nil)
	backupWant(t, ts, http.StatusCreated, http.MethodDelete, "/legal-holds/"+holdID+"?cancelDescription=done&retainRecordInDays=1", nil)
	backupFail(t, ts, "InvalidResourceStateException", http.MethodDelete, "/legal-holds/"+holdID+"?cancelDescription=again", nil)
	backupWant(t, ts, http.StatusOK, http.MethodDelete, rpPath, nil)
	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, rpPath, nil)
}

func TestBackupStage2StopJobsScansIndexesAndAccessPoints(t *testing.T) {
	srv, ts := newBackupTestServer(t)
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/vlt", nil)
	tableARN := backupTestTable(t, srv, "t")
	job := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "vlt", "IamRoleArn": backupTestRole, "ResourceArn": tableARN})
	dynamoBackup := srv.backup.recoveryPoints[job["RecoveryPointArn"].(string)].dynamoBackupARN
	if _, err := srv.dynamodb.DescribeBackup(dynamoBackup); err != nil {
		t.Fatalf("a DynamoDB backup job takes a table backup: %v", err)
	}
	backupWant(t, ts, http.StatusOK, http.MethodPost, "/backup-jobs/"+job["BackupJobId"].(string), nil)
	if _, err := srv.dynamodb.DescribeBackup(dynamoBackup); err == nil {
		t.Fatalf("stopping the job must delete its DynamoDB backup")
	}
	if d := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-jobs/"+job["BackupJobId"].(string), nil); d["State"] != "ABORTED" {
		t.Fatalf("stopped job: %v", d)
	}
	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, "/backup-vaults/vlt/recovery-points/"+url.PathEscape(job["RecoveryPointArn"].(string)), nil)

	// Indexes are only for EBS and S3; an S3 backup is accepted generically.
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "vlt", "IamRoleArn": backupTestRole, "ResourceArn": tableARN, "Index": "ENABLED"})
	s3 := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "vlt", "IamRoleArn": backupTestRole, "ResourceArn": "arn:aws:s3:::bucket"})
	rpARN := s3["RecoveryPointArn"].(string)
	rpPath := "/backup-vaults/vlt/recovery-points/" + url.PathEscape(rpARN)
	backupWant(t, ts, http.StatusOK, http.MethodGet, rpPath, nil)
	backupWant(t, ts, http.StatusOK, http.MethodGet, rpPath, nil)
	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, rpPath+"/index", nil)
	idx := backupWant(t, ts, http.StatusOK, http.MethodPost, rpPath+"/index", map[string]any{"Index": "ENABLED"})
	if idx["IndexStatus"] != "ACTIVE" {
		t.Fatalf("index: %v", idx)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/indexes/recovery-point/?indexStatus=ACTIVE", nil); len(l["IndexedRecoveryPoints"].([]any)) != 1 {
		t.Fatalf("indexed recovery points: %v", l)
	}

	scanBody := map[string]any{"BackupVaultName": "vlt", "IamRoleArn": backupTestRole, "ScannerRoleArn": backupTestRole, "MalwareScanner": "GUARDDUTY", "RecoveryPointArn": rpARN, "ScanMode": "FULL_SCAN"}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/scan/job", map[string]any{"BackupVaultName": "vlt", "IamRoleArn": backupTestRole, "ScannerRoleArn": backupTestRole, "MalwareScanner": "OTHER", "RecoveryPointArn": rpARN, "ScanMode": "FULL_SCAN"})
	scan := backupWant(t, ts, http.StatusCreated, http.MethodPut, "/scan/job", scanBody)
	scanID := scan["ScanJobId"].(string)
	q := "?RecoveryPointArn=" + url.QueryEscape(rpARN) + "&BackupVaultName=vlt&MalwareScanner=GUARDDUTY&ScanEndTime=4102444800"
	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, "/scan/pitr-malware-scan-results"+q, nil)
	backupWant(t, ts, http.StatusOK, http.MethodGet, "/scan/jobs/"+scanID, nil)
	scanned := backupWant(t, ts, http.StatusOK, http.MethodGet, "/scan/jobs/"+scanID, nil)
	if scanned["State"] != "COMPLETED" || scanned["ScanResult"].(map[string]any)["ScanResultStatus"] != "NO_THREATS_FOUND" {
		t.Fatalf("scan job: %v", scanned)
	}
	if r := backupWant(t, ts, http.StatusOK, http.MethodGet, "/scan/pitr-malware-scan-results"+q, nil); r["ScanId"] == nil {
		t.Fatalf("pitr scan results: %v", r)
	}

	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-access-point/create", map[string]any{"Name": "Bad_Name", "RecoveryPointArn": rpARN})
	ap := backupWant(t, ts, http.StatusCreated, http.MethodPut, "/backup-access-point/create", map[string]any{"Name": "ap-one", "RecoveryPointArn": rpARN})
	apARN := ap["AccessPointArn"].(string)
	backupFail(t, ts, "AlreadyExistsException", http.MethodPut, "/backup-access-point/create", map[string]any{"Name": "ap-one", "RecoveryPointArn": rpARN})
	if l := backupWant(t, ts, http.StatusOK, http.MethodPost, "/backup-access-point/recovery-point/"+url.PathEscape(rpARN), nil); len(l["BackupAccessPoints"].([]any)) != 1 {
		t.Fatalf("access points by recovery point: %v", l)
	}
	backupWant(t, ts, http.StatusNoContent, http.MethodDelete, "/backup-access-point/delete/"+url.PathEscape(apARN), nil)
	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, "/backup-access-point/"+url.PathEscape(apARN), nil)
}

func TestBackupStage2AuditRestoreTestingAndTiering(t *testing.T) {
	_, ts := newBackupTestServer(t)
	controls := []any{map[string]any{"ControlName": "BACKUP_RECOVERY_POINT_ENCRYPTED"}}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPost, "/audit/frameworks", map[string]any{"FrameworkName": "f", "FrameworkControls": []any{map[string]any{"ControlName": "NOPE"}}})
	fw := backupWant(t, ts, http.StatusOK, http.MethodPost, "/audit/frameworks", map[string]any{"FrameworkName": "fw", "FrameworkControls": controls})
	backupFail(t, ts, "AlreadyExistsException", http.MethodPost, "/audit/frameworks", map[string]any{"FrameworkName": "fw", "FrameworkControls": controls})
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/audit/frameworks", nil); l["Frameworks"].([]any)[0].(map[string]any)["NumberOfControls"] != float64(1) {
		t.Fatalf("frameworks: %v", l)
	}

	report := map[string]any{"ReportPlanName": "rp", "ReportDeliveryChannel": map[string]any{"S3BucketName": "reports", "Formats": []string{"JSON"}}, "ReportSetting": map[string]any{"ReportTemplate": "CONTROL_COMPLIANCE_REPORT", "FrameworkArns": []any{fw["FrameworkArn"]}}}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPost, "/audit/report-plans", map[string]any{"ReportPlanName": "x", "ReportDeliveryChannel": map[string]any{"S3BucketName": "b"}, "ReportSetting": map[string]any{"ReportTemplate": "CONTROL_COMPLIANCE_REPORT"}})
	backupWant(t, ts, http.StatusOK, http.MethodPost, "/audit/report-plans", report)
	backupFail(t, ts, "ConflictException", http.MethodDelete, "/audit/frameworks/fw", nil)
	job := backupWant(t, ts, http.StatusOK, http.MethodPost, "/audit/report-jobs/rp", nil)
	backupWant(t, ts, http.StatusOK, http.MethodGet, "/audit/report-jobs/"+job["ReportJobId"].(string), nil)
	rj := backupWant(t, ts, http.StatusOK, http.MethodGet, "/audit/report-jobs/"+job["ReportJobId"].(string), nil)["ReportJob"].(map[string]any)
	keys := rj["ReportDestination"].(map[string]any)["S3Keys"].([]any)
	if rj["Status"] != "COMPLETED" || !strings.HasSuffix(keys[0].(string), ".json") {
		t.Fatalf("report job: %v", rj)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/audit/report-jobs?ReportPlanName=rp", nil); len(l["ReportJobs"].([]any)) != 1 {
		t.Fatalf("report jobs: %v", l)
	}
	backupWant(t, ts, http.StatusOK, http.MethodDelete, "/audit/report-plans/rp", nil)
	backupWant(t, ts, http.StatusOK, http.MethodDelete, "/audit/frameworks/fw", nil)

	plan := map[string]any{"RestoreTestingPlanName": "rtp", "ScheduleExpression": "cron(0 1 ? * * *)", "RecoveryPointSelection": map[string]any{"Algorithm": "LATEST_WITHIN_WINDOW", "IncludeVaults": []string{"*"}, "RecoveryPointTypes": []string{"SNAPSHOT"}}}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/restore-testing/plans", map[string]any{"RestoreTestingPlan": map[string]any{"RestoreTestingPlanName": "x", "ScheduleExpression": "cron(0 1 ? * * *)", "RecoveryPointSelection": map[string]any{"Algorithm": "NOPE", "IncludeVaults": []string{"*"}, "RecoveryPointTypes": []string{"SNAPSHOT"}}}})
	backupWant(t, ts, http.StatusCreated, http.MethodPut, "/restore-testing/plans", map[string]any{"RestoreTestingPlan": plan})
	sel := map[string]any{"RestoreTestingSelectionName": "sel", "ProtectedResourceType": "DynamoDB", "IamRoleArn": backupTestRole, "ProtectedResourceArns": []string{"*"}}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/restore-testing/plans/rtp/selections", map[string]any{"RestoreTestingSelection": map[string]any{"RestoreTestingSelectionName": "s2", "ProtectedResourceType": "DynamoDB", "IamRoleArn": backupTestRole, "ProtectedResourceArns": []string{"*"}, "ProtectedResourceConditions": map[string]any{}}})
	backupWant(t, ts, http.StatusCreated, http.MethodPut, "/restore-testing/plans/rtp/selections", map[string]any{"RestoreTestingSelection": sel})
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/restore-testing/plans/rtp/selections/sel", map[string]any{"RestoreTestingSelection": map[string]any{"ValidationWindowHours": 12}})
	got := backupWant(t, ts, http.StatusOK, http.MethodGet, "/restore-testing/plans/rtp/selections/sel", nil)["RestoreTestingSelection"].(map[string]any)
	if got["ValidationWindowHours"] != float64(12) {
		t.Fatalf("testing selection: %v", got)
	}
	backupFail(t, ts, "InvalidRequestException", http.MethodDelete, "/restore-testing/plans/rtp", nil)
	backupWant(t, ts, http.StatusNoContent, http.MethodDelete, "/restore-testing/plans/rtp/selections/sel", nil)
	backupWant(t, ts, http.StatusNoContent, http.MethodDelete, "/restore-testing/plans/rtp", nil)

	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/tv", nil)
	cfg := map[string]any{"TieringConfigurationName": "tier", "BackupVaultName": "tv", "ResourceSelection": []any{map[string]any{"ResourceType": "S3", "Resources": []string{"*"}, "TieringDownSettingsInDays": 60}}}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/tiering-configurations", map[string]any{"TieringConfiguration": map[string]any{"TieringConfigurationName": "t2", "BackupVaultName": "tv", "ResourceSelection": []any{map[string]any{"ResourceType": "S3", "Resources": []string{"*"}, "TieringDownSettingsInDays": 10}}}})
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/tiering-configurations", map[string]any{"TieringConfiguration": cfg})
	backupFail(t, ts, "AlreadyExistsException", http.MethodPut, "/tiering-configurations", map[string]any{"TieringConfiguration": cfg})
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/tiering-configurations/", nil); len(l["TieringConfigurations"].([]any)) != 1 {
		t.Fatalf("tiering list: %v", l)
	}
	backupWant(t, ts, http.StatusOK, http.MethodDelete, "/tiering-configurations/tier", nil)
}
