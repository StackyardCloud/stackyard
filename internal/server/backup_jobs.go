package server

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	bkJobBackup  = "backup"
	bkJobCopy    = "copy"
	bkJobRestore = "restore"
	bkJobScan    = "scan"
	bkJobReport  = "report"
	bkMaxJobs    = 10000
)

var bkResourceARNPattern = regexp.MustCompile(`^arn:aws[a-z-]*:[a-z0-9-]+:[a-z0-9-]*:[0-9]{0,12}:.+$`)

// bkJob is any asynchronous job. Each read advances a running job one
// step (CREATED -> RUNNING -> COMPLETED), so callers see it progress.
type bkJob struct {
	kind     string
	rec      map[string]any
	stateKey string // "State" or "Status"
	steps    []string
	metadata map[string]any // restore metadata
}

func (j *bkJob) state() string { return bkStr(j.rec, j.stateKey) }

func (j *bkJob) running() bool {
	return j.state() != "" && j.state() != j.steps[len(j.steps)-1] && containsString(j.steps, j.state())
}

func bkJobHandlers() map[string]bkHandler {
	return map[string]bkHandler{
		"StartBackupJob":                     (*backupStore).startBackupJob,
		"DescribeBackupJob":                  (*backupStore).describeBackupJob,
		"StopBackupJob":                      (*backupStore).stopBackupJob,
		"ListBackupJobs":                     (*backupStore).listBackupJobs,
		"ListBackupJobSummaries":             bkJobSummaries(bkJobBackup, "BackupJobSummaries"),
		"StartCopyJob":                       (*backupStore).startCopyJob,
		"DescribeCopyJob":                    (*backupStore).describeCopyJob,
		"ListCopyJobs":                       (*backupStore).listCopyJobs,
		"ListCopyJobSummaries":               bkJobSummaries(bkJobCopy, "CopyJobSummaries"),
		"StartRestoreJob":                    (*backupStore).startRestoreJob,
		"DescribeRestoreJob":                 (*backupStore).describeRestoreJob,
		"ListRestoreJobs":                    (*backupStore).listRestoreJobs,
		"ListRestoreJobsByProtectedResource": (*backupStore).listRestoreJobsByResource,
		"ListRestoreJobSummaries":            bkJobSummaries(bkJobRestore, "RestoreJobSummaries"),
		"GetRestoreJobMetadata":              (*backupStore).getRestoreJobMetadata,
		"PutRestoreValidationResult":         (*backupStore).putRestoreValidationResult,
		"StartScanJob":                       (*backupStore).startScanJob,
		"DescribeScanJob":                    (*backupStore).describeScanJob,
		"ListScanJobs":                       (*backupStore).listScanJobs,
		"ListScanJobSummaries":               bkJobSummaries(bkJobScan, "ScanJobSummaries"),
		"GetPITRMalwareScanResults":          (*backupStore).getPITRScanResults,
	}
}

// bkResourceType maps a resource ARN to its AWS Backup resource type.
func bkResourceType(arn string) (string, string, bool) {
	if !bkResourceARNPattern.MatchString(arn) {
		return "", "", false
	}
	parts := strings.SplitN(arn, ":", 6)
	service, resource := parts[2], parts[5]
	name := resource
	if i := strings.LastIndexAny(resource, "/:"); i >= 0 {
		name = resource[i+1:]
	}
	switch service {
	case "dynamodb":
		if t, ok := strings.CutPrefix(resource, "table/"); ok {
			return "DynamoDB", strings.SplitN(t, "/", 2)[0], true
		}
	case "ec2":
		switch {
		case strings.HasPrefix(resource, "volume/"):
			return "EBS", name, true
		case strings.HasPrefix(resource, "instance/"):
			return "EC2", name, true
		}
	case "rds":
		switch {
		case strings.HasPrefix(resource, "cluster:"):
			return "Aurora", name, true
		case strings.HasPrefix(resource, "db:"):
			return "RDS", name, true
		}
	case "elasticfilesystem":
		return "EFS", name, true
	case "s3":
		return "S3", resource, true
	case "fsx":
		return "FSx", name, true
	case "storagegateway":
		return "Storage Gateway", name, true
	case "redshift":
		return "Redshift", name, true
	case "redshift-serverless":
		return "Redshift Serverless", name, true
	case "cloudformation":
		return "CloudFormation", name, true
	case "timestream":
		return "Timestream", name, true
	case "ssm-sap":
		return "SAP HANA on Amazon EC2", name, true
	case "backup-gateway":
		return "VirtualMachine", name, true
	}
	return "", "", false
}

func (st *backupStore) addJob(j *bkJob, idKey string) (string, *bedrockAgentError) {
	if len(st.jobs) >= bkMaxJobs {
		return "", bkErr("LimitExceededException", "too many jobs")
	}
	id := bkNewID()
	j.rec[idKey] = id
	j.rec["CreationDate"] = bkNow()
	j.rec["AccountId"] = backupAccount
	st.jobs[id] = j
	return id, nil
}

// idempotent returns the job started earlier with the same token, if any.
func (st *backupStore) idempotent(kind string, in map[string]any) *bkJob {
	tok := bkStr(in, "IdempotencyToken")
	if tok == "" {
		return nil
	}
	return st.jobs[st.idempotentJobs[kind+"\x00"+tok]]
}

func (st *backupStore) remember(kind string, in map[string]any, id string) {
	if tok := bkStr(in, "IdempotencyToken"); tok != "" {
		st.idempotentJobs[kind+"\x00"+tok] = id
	}
}

func (st *backupStore) job(kind, id string) (*bkJob, *bedrockAgentError) {
	j := st.jobs[id]
	if j == nil || j.kind != kind {
		return nil, bkNotFound(strings.ToUpper(kind[:1])+kind[1:]+" job", id)
	}
	return j, nil
}

// tick advances a running job one step and applies its completion.
func (st *backupStore) tick(s *Server, j *bkJob) {
	if !j.running() {
		return
	}
	i := 0
	for i < len(j.steps) && j.steps[i] != j.state() {
		i++
	}
	next := j.steps[i+1]
	j.rec[j.stateKey] = next
	if next != j.steps[len(j.steps)-1] {
		if j.kind != bkJobScan && j.kind != bkJobReport {
			j.rec["PercentDone"] = "50.0"
		}
		return
	}
	now := bkNow()
	switch j.kind {
	case bkJobBackup:
		st.completeBackup(j, now)
	case bkJobCopy:
		st.completeCopy(j, now)
	case bkJobRestore:
		st.completeRestore(s, j, now)
	case bkJobScan:
		st.completeScan(j, now)
	case bkJobReport:
		j.rec["CompletionTime"] = now
		return
	}
	j.rec["CompletionDate"] = now
	if j.state() != j.steps[len(j.steps)-1] {
		return // the completion step failed (for example a DynamoDB restore)
	}
	j.rec["MessageCategory"] = "SUCCESS"
	if j.kind != bkJobScan {
		j.rec["PercentDone"] = "100.0"
	}
}

func (st *backupStore) completeBackup(j *bkJob, now float64) {
	rp := st.recoveryPoints[bkStr(j.rec, "RecoveryPointArn")]
	if rp == nil {
		return
	}
	rp.rec["Status"] = "COMPLETED"
	rp.rec["CompletionDate"] = now
	j.rec["BytesTransferred"] = j.rec["BackupSizeInBytes"]
	pr := st.protected[bkStr(j.rec, "ResourceArn")]
	if pr == nil {
		pr = map[string]any{"ResourceArn": j.rec["ResourceArn"], "ResourceType": j.rec["ResourceType"], "ResourceName": j.rec["ResourceName"]}
		st.protected[bkStr(j.rec, "ResourceArn")] = pr
	}
	pr["LastBackupTime"] = now
	pr["LastBackupVaultArn"] = j.rec["BackupVaultArn"]
	pr["LastRecoveryPointArn"] = rp.rec["RecoveryPointArn"]
}

func (st *backupStore) completeCopy(j *bkJob, now float64) {
	src := st.recoveryPoints[bkStr(j.rec, "SourceRecoveryPointArn")]
	dest := st.vaultByARN(bkStr(j.rec, "DestinationBackupVaultArn"))
	if src == nil || dest == nil {
		return // cross-account or cross-Region destination: nothing to record here
	}
	arn := bkARN("recovery-point", bkNewID())
	rec := baClone(src.rec)
	rec["RecoveryPointArn"] = arn
	rec["BackupVaultName"] = dest.rec["BackupVaultName"]
	rec["BackupVaultArn"] = dest.rec["BackupVaultArn"]
	rec["VaultType"] = dest.rec["VaultType"]
	rec["SourceBackupVaultArn"] = j.rec["SourceBackupVaultArn"]
	rec["CreationDate"], rec["CompletionDate"] = now, now
	rec["Status"] = "COMPLETED"
	if lc := bkMap(j.rec, "Lifecycle"); lc != nil {
		rec["Lifecycle"] = lc
	}
	rec["CalculatedLifecycle"] = bkCalculatedLifecycle(bkMap(rec, "Lifecycle"), now)
	if dest.rec["EncryptionKeyArn"] != nil {
		rec["EncryptionKeyArn"] = dest.rec["EncryptionKeyArn"]
	}
	delete(rec, "IndexStatus")
	delete(rec, "ScanResults")
	st.recoveryPoints[arn] = &bkRecoveryPoint{rec: rec, dynamoBackupARN: src.dynamoBackupARN}
	j.rec["DestinationRecoveryPointArn"] = arn
}

func (st *backupStore) completeRestore(s *Server, j *bkJob, now float64) {
	rp := st.recoveryPoints[bkStr(j.rec, "RecoveryPointArn")]
	created := bkStr(j.rec, "SourceResourceArn") + "-restored-" + bkNewID()[:8]
	if rp != nil && rp.dynamoBackupARN != "" {
		target := fmt.Sprint(j.metadata["targetTableName"])
		desc, err := s.dynamodb.RestoreTableFromBackup(target, rp.dynamoBackupARN)
		if err != nil {
			j.rec["Status"] = "FAILED"
			j.rec["StatusMessage"] = "restoring DynamoDB table " + target + ": " + err.Error()
			return
		}
		created = desc.TableArn
	}
	j.rec["CreatedResourceArn"] = created
	if rp != nil {
		rp.rec["LastRestoreTime"] = now
	}
	if pr := st.protected[bkStr(j.rec, "SourceResourceArn")]; pr != nil {
		pr["LatestRestoreJobCreationDate"] = j.rec["CreationDate"]
		pr["LatestRestoreRecoveryPointCreationDate"] = j.rec["RecoveryPointCreationDate"]
		pr["LatestRestoreExecutionTimeMinutes"] = int((now - bkFloat(j.rec["CreationDate"])) / 60)
	}
}

func (st *backupStore) completeScan(j *bkJob, now float64) {
	j.rec["CompletionDate"] = now
	j.rec["ScanId"] = bkNewID()
	j.rec["ScanResult"] = map[string]any{"ScanResultStatus": "NO_THREATS_FOUND"}
	if rp := st.recoveryPoints[bkStr(j.rec, "RecoveryPointArn")]; rp != nil {
		rp.rec["ScanResults"] = []any{map[string]any{"MalwareScanner": j.rec["MalwareScanner"], "ScanJobState": "COMPLETED", "LastScanTimestamp": now, "Findings": []any{}}}
	}
}

// bkFilter maps a list filter member to the record member it constrains:
// '=' equality, '<' record time before the filter, '>' record time after it.
type bkFilter struct {
	in, rec string
	op      byte
}

func bkMatch(rec, in map[string]any, filters []bkFilter) (bool, *bedrockAgentError) {
	for _, f := range filters {
		if f.op == '=' {
			want := bkStr(in, f.in)
			if want != "" && fmt.Sprint(rec[f.rec]) != want {
				return false, nil
			}
			continue
		}
		t, ok, err := bkTime(in, f.in)
		if err != nil {
			return false, err
		}
		if !ok {
			continue
		}
		got, has := rec[f.rec].(float64)
		if !has || (f.op == '<' && got >= t) || (f.op == '>' && got <= t) {
			return false, nil
		}
	}
	return true, nil
}

// listJobs ticks and filters jobs of one kind, oldest first.
func (st *backupStore) listJobs(s *Server, kind string, in map[string]any, filters []bkFilter, extra func(*bkJob) bool, shape, listKey string) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, j := range bkSorted(st.jobs, func(j *bkJob) map[string]any { return j.rec }, "CreationDate") {
		if j.kind != kind || (extra != nil && !extra(j)) {
			continue
		}
		st.tick(s, j)
		ok, err := bkMatch(j.rec, in, filters)
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, bkShape(j.rec, shape))
		}
	}
	return bkPage(items, in, listKey)
}

// --- backup jobs ---

func bkCalculatedLifecycle(lc map[string]any, created float64) map[string]any {
	out := map[string]any{}
	if d, absent, _ := bkInt(lc, "DeleteAfterDays"); !absent {
		out["DeleteAt"] = created + float64(d*bkDaySeconds)
	}
	if d, absent, _ := bkInt(lc, "MoveToColdStorageAfterDays"); !absent {
		out["MoveToColdStorageAt"] = created + float64(d*bkDaySeconds)
	}
	return out
}

// bkCheckVaultRetention enforces a locked vault's retention range.
func bkCheckVaultRetention(v *bkVault, lc map[string]any) *bedrockAgentError {
	if v.rec["Locked"] != true || lc == nil {
		return nil
	}
	days, absent, _ := bkInt(lc, "DeleteAfterDays")
	if absent {
		return nil
	}
	if lo, ok := v.rec["MinRetentionDays"].(int); ok && days < lo {
		return bkInvalid(fmt.Sprintf("DeleteAfterDays must be at least the vault's MinRetentionDays (%d)", lo))
	}
	if hi, ok := v.rec["MaxRetentionDays"].(int); ok && days > hi {
		return bkInvalid(fmt.Sprintf("DeleteAfterDays must be at most the vault's MaxRetentionDays (%d)", hi))
	}
	return nil
}

func (st *backupStore) startBackupJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "BackupVaultName", "ResourceArn", "IamRoleArn"); err != nil {
		return nil, err
	}
	if err := bkCheckRole(in, "IamRoleArn"); err != nil {
		return nil, err
	}
	if err := bkEnum(in, "Index", "ENABLED", "DISABLED"); err != nil {
		return nil, err
	}
	if j := st.idempotent(bkJobBackup, in); j != nil {
		return j.rec, nil
	}
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if v.rec["VaultType"] == bkVaultRestoreAccess {
		return nil, bkRequest("backups cannot be written to a restore access backup vault")
	}
	if arn := bkStr(in, "LogicallyAirGappedBackupVaultArn"); arn != "" {
		if lag := st.vaultByARN(arn); lag == nil || lag.rec["VaultType"] != bkVaultAirGapped {
			return nil, bkInvalid("LogicallyAirGappedBackupVaultArn must name a logically air-gapped backup vault")
		}
	}
	resourceARN := bkStr(in, "ResourceArn")
	resourceType, resourceName, ok := bkResourceType(resourceARN)
	if !ok {
		return nil, bkInvalid("ResourceArn " + resourceARN + " is not a resource type AWS Backup supports")
	}
	if opt, _ := bkMap(st.regionSettings, "ResourceTypeOptInPreference")[resourceType].(bool); !opt {
		return nil, bkRequest("resource type " + resourceType + " is not opted in for this Region")
	}
	if resourceType == "VirtualMachine" {
		// Lock order: Backup's store (held here) before Backup gateway's.
		name, known := s.backupGateway.vmName(resourceARN)
		if !known {
			return nil, bkNotFound("virtual machine", resourceARN)
		}
		resourceName = name
	}
	lc := bkMap(in, "Lifecycle")
	if err := bkCheckLifecycle(lc); err != nil {
		return nil, err
	}
	if err := bkCheckVaultRetention(v, lc); err != nil {
		return nil, err
	}
	startWindow, startAbsent, err := bkInt(in, "StartWindowMinutes")
	if err != nil {
		return nil, err
	}
	if startAbsent {
		startWindow = 480
	}
	completeWindow, completeAbsent, err := bkInt(in, "CompleteWindowMinutes")
	if err != nil {
		return nil, err
	}
	if completeAbsent {
		completeWindow = 10080
	}
	if startWindow < 60 || completeWindow < 1 {
		return nil, bkInvalid("StartWindowMinutes must be at least 60 and CompleteWindowMinutes positive")
	}
	if bkStr(in, "Index") == "ENABLED" && !bkIndexable(resourceType) {
		return nil, bkInvalid("backup indexes are only supported for EBS and S3 resources")
	}
	if err := st.checkTags("", bkMap(in, "RecoveryPointTags")); err != nil {
		return nil, err
	}
	if len(st.recoveryPoints) >= bkMaxJobs {
		return nil, bkErr("LimitExceededException", "too many recovery points")
	}
	var size int64
	dynamoBackup := ""
	if resourceType == "DynamoDB" {
		table, tErr := s.dynamodb.DescribeTable(resourceName)
		if tErr != nil {
			return nil, bkNotFound("DynamoDB table", resourceName)
		}
		size = table.TableSizeBytes
		bk, bErr := s.dynamodb.CreateBackup(resourceName, "awsbackup-"+bkNewID())
		if bErr != nil {
			return nil, bkErr("DependencyFailureException", "DynamoDB backup failed: "+bErr.Error())
		}
		dynamoBackup = bk.BackupArn
	}
	now := bkNow()
	rpARN := bkARN("recovery-point", bkNewID())
	j := &bkJob{kind: bkJobBackup, stateKey: "State", steps: []string{"CREATED", "RUNNING", "COMPLETED"}, rec: map[string]any{
		"BackupVaultName": v.rec["BackupVaultName"], "BackupVaultArn": v.rec["BackupVaultArn"], "VaultType": v.rec["VaultType"],
		"RecoveryPointArn": rpARN, "ResourceArn": resourceARN, "ResourceType": resourceType, "ResourceName": resourceName,
		"IamRoleArn": in["IamRoleArn"], "State": "CREATED", "PercentDone": "0.0", "BackupSizeInBytes": size,
		"StartBy": now + float64(startWindow*60), "ExpectedCompletionDate": now + float64(completeWindow*60),
		"BackupOptions": in["BackupOptions"], "BackupType": "FULL", "IsParent": false, "InitiationDate": now,
		"EncryptionKeyArn": v.rec["EncryptionKeyArn"], "IsEncrypted": true, "RecoveryPointLifecycle": lc,
	}}
	if v.rec["Locked"] == true {
		j.rec["VaultLockState"] = "LOCKED"
	}
	id, err := st.addJob(j, "BackupJobId")
	if err != nil {
		if dynamoBackup != "" {
			_, _ = s.dynamodb.DeleteBackup(dynamoBackup)
		}
		return nil, err
	}
	rp := map[string]any{
		"RecoveryPointArn": rpARN, "BackupVaultName": v.rec["BackupVaultName"], "BackupVaultArn": v.rec["BackupVaultArn"],
		"ResourceArn": resourceARN, "ResourceType": resourceType, "ResourceName": resourceName,
		"CreatedBy": map[string]any{}, "IamRoleArn": in["IamRoleArn"], "Status": "CREATING",
		"CreationDate": now, "InitiationDate": now, "BackupSizeInBytes": size,
		"Lifecycle": lc, "CalculatedLifecycle": bkCalculatedLifecycle(lc, now),
		"EncryptionKeyArn": v.rec["EncryptionKeyArn"], "EncryptionKeyType": v.rec["EncryptionKeyType"], "IsEncrypted": true,
		"StorageClass": "WARM", "IsParent": false, "VaultType": v.rec["VaultType"],
	}
	if bkStr(in, "Index") == "ENABLED" {
		rp["IndexStatus"] = "PENDING"
	}
	st.recoveryPoints[rpARN] = &bkRecoveryPoint{rec: rp, jobID: id, dynamoBackupARN: dynamoBackup}
	_ = st.setTags(rpARN, bkMap(in, "RecoveryPointTags")) // validated above
	st.remember(bkJobBackup, in, id)
	return j.rec, nil
}

func (st *backupStore) describeBackupJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobBackup, bkStr(in, "BackupJobId"))
	if err != nil {
		return nil, err
	}
	st.tick(s, j)
	return j.rec, nil
}

func (st *backupStore) stopBackupJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobBackup, bkStr(in, "BackupJobId"))
	if err != nil {
		return nil, err
	}
	if !j.running() {
		return nil, bkRequest("backup job " + bkStr(in, "BackupJobId") + " is " + j.state() + " and cannot be stopped")
	}
	j.rec["State"] = "ABORTED"
	j.rec["StatusMessage"] = "Backup job was stopped by the user"
	j.rec["CompletionDate"] = bkNow()
	if rp := st.recoveryPoints[bkStr(j.rec, "RecoveryPointArn")]; rp != nil {
		st.dropRecoveryPoint(s, rp, true)
	}
	return nil, nil
}

var bkBackupJobFilters = []bkFilter{
	{"ByResourceArn", "ResourceArn", '='}, {"ByState", "State", '='}, {"ByBackupVaultName", "BackupVaultName", '='},
	{"ByResourceType", "ResourceType", '='}, {"ByAccountId", "AccountId", '='}, {"ByParentJobId", "ParentJobId", '='},
	{"ByMessageCategory", "MessageCategory", '='},
	{"ByCreatedBefore", "CreationDate", '<'}, {"ByCreatedAfter", "CreationDate", '>'},
	{"ByCompleteBefore", "CompletionDate", '<'}, {"ByCompleteAfter", "CompletionDate", '>'},
}

func (st *backupStore) listBackupJobs(s *Server, in map[string]any) (any, *bedrockAgentError) {
	return st.listJobs(s, bkJobBackup, in, bkBackupJobFilters, nil, "BackupJob", "BackupJobs")
}

// --- copy jobs ---

func (st *backupStore) startCopyJob(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "RecoveryPointArn", "SourceBackupVaultName", "DestinationBackupVaultArn", "IamRoleArn"); err != nil {
		return nil, err
	}
	if err := bkCheckRole(in, "IamRoleArn"); err != nil {
		return nil, err
	}
	if j := st.idempotent(bkJobCopy, in); j != nil {
		return j.rec, nil
	}
	src, rp, err := st.vaultRecoveryPoint(bkStr(in, "SourceBackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	if rp.rec["Status"] != "COMPLETED" {
		return nil, bkRequest("recovery point " + bkStr(in, "RecoveryPointArn") + " is not completed")
	}
	destARN := bkStr(in, "DestinationBackupVaultArn")
	if !strings.HasPrefix(destARN, "arn:aws") || !strings.Contains(destARN, ":backup-vault:") {
		return nil, bkInvalid("DestinationBackupVaultArn must be a backup vault ARN")
	}
	dest := st.vaultByARN(destARN)
	if dest == nil && strings.Contains(destARN, ":"+backupRegion+":"+backupAccount+":") {
		return nil, bkNotFound("Backup vault", destARN)
	}
	lc := bkMap(in, "Lifecycle")
	if err := bkCheckLifecycle(lc); err != nil {
		return nil, err
	}
	if dest != nil {
		if err := bkCheckVaultRetention(dest, lc); err != nil {
			return nil, err
		}
	}
	j := &bkJob{kind: bkJobCopy, stateKey: "State", steps: []string{"CREATED", "RUNNING", "COMPLETED"}, rec: map[string]any{
		"SourceBackupVaultArn": src.rec["BackupVaultArn"], "SourceRecoveryPointArn": rp.rec["RecoveryPointArn"],
		"DestinationBackupVaultArn": destARN, "ResourceArn": rp.rec["ResourceArn"], "ResourceType": rp.rec["ResourceType"],
		"ResourceName": rp.rec["ResourceName"], "BackupSizeInBytes": rp.rec["BackupSizeInBytes"], "IamRoleArn": in["IamRoleArn"],
		"State": "CREATED", "IsParent": false, "Lifecycle": lc, "DestinationRecoveryPointLifecycle": lc,
	}}
	if dest != nil {
		j.rec["DestinationVaultType"] = dest.rec["VaultType"]
		j.rec["DestinationEncryptionKeyArn"] = dest.rec["EncryptionKeyArn"]
	}
	id, err := st.addJob(j, "CopyJobId")
	if err != nil {
		return nil, err
	}
	st.remember(bkJobCopy, in, id)
	return j.rec, nil
}

func (st *backupStore) describeCopyJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobCopy, bkStr(in, "CopyJobId"))
	if err != nil {
		return nil, err
	}
	st.tick(s, j)
	return map[string]any{"CopyJob": bkShape(j.rec, "CopyJob")}, nil
}

func (st *backupStore) listCopyJobs(s *Server, in map[string]any) (any, *bedrockAgentError) {
	filters := []bkFilter{
		{"ByResourceArn", "ResourceArn", '='}, {"ByState", "State", '='}, {"ByResourceType", "ResourceType", '='},
		{"ByDestinationVaultArn", "DestinationBackupVaultArn", '='}, {"ByAccountId", "AccountId", '='},
		{"ByParentJobId", "ParentJobId", '='}, {"ByMessageCategory", "MessageCategory", '='},
		{"BySourceRecoveryPointArn", "SourceRecoveryPointArn", '='},
		{"ByCreatedBefore", "CreationDate", '<'}, {"ByCreatedAfter", "CreationDate", '>'},
		{"ByCompleteBefore", "CompletionDate", '<'}, {"ByCompleteAfter", "CompletionDate", '>'},
	}
	return st.listJobs(s, bkJobCopy, in, filters, nil, "CopyJob", "CopyJobs")
}

// --- restore jobs ---

func (st *backupStore) startRestoreJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "RecoveryPointArn", "Metadata"); err != nil {
		return nil, err
	}
	if err := bkCheckRole(in, "IamRoleArn"); err != nil {
		return nil, err
	}
	if j := st.idempotent(bkJobRestore, in); j != nil {
		return j.rec, nil
	}
	rp := st.recoveryPoints[bkStr(in, "RecoveryPointArn")]
	if rp == nil {
		return nil, bkNotFound("Recovery point", bkStr(in, "RecoveryPointArn"))
	}
	if rp.rec["Status"] != "COMPLETED" {
		return nil, bkRequest("recovery point " + bkStr(in, "RecoveryPointArn") + " is not completed")
	}
	metadata := bkMap(in, "Metadata")
	if len(metadata) > 100 {
		return nil, bkInvalid("Metadata can have at most 100 entries")
	}
	for k, v := range metadata {
		if _, ok := v.(string); !ok {
			return nil, bkInvalid("Metadata value for " + k + " must be a string")
		}
	}
	if t := bkStr(in, "ResourceType"); t != "" && t != rp.rec["ResourceType"] {
		return nil, bkInvalid("ResourceType " + t + " does not match the recovery point's " + bkStr(rp.rec, "ResourceType"))
	}
	if rp.dynamoBackupARN != "" && bkStr(metadata, "targetTableName") == "" {
		return nil, bkInvalid("restoring a DynamoDB recovery point needs Metadata targetTableName")
	}
	j := &bkJob{kind: bkJobRestore, stateKey: "Status", steps: []string{"PENDING", "RUNNING", "COMPLETED"}, metadata: metadata, rec: map[string]any{
		"RecoveryPointArn": rp.rec["RecoveryPointArn"], "SourceResourceArn": rp.rec["ResourceArn"],
		"BackupVaultArn": rp.rec["BackupVaultArn"], "Status": "PENDING", "PercentDone": "0.0",
		"BackupSizeInBytes": rp.rec["BackupSizeInBytes"], "IamRoleArn": in["IamRoleArn"], "ExpectedCompletionTimeMinutes": 5,
		"ResourceType": rp.rec["ResourceType"], "RecoveryPointCreationDate": rp.rec["CreationDate"], "IsParent": false,
	}}
	id, err := st.addJob(j, "RestoreJobId")
	if err != nil {
		return nil, err
	}
	st.remember(bkJobRestore, in, id)
	return j.rec, nil
}

func (st *backupStore) describeRestoreJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobRestore, bkStr(in, "RestoreJobId"))
	if err != nil {
		return nil, err
	}
	st.tick(s, j)
	return j.rec, nil
}

func (st *backupStore) listRestoreJobs(s *Server, in map[string]any) (any, *bedrockAgentError) {
	filters := []bkFilter{
		{"ByAccountId", "AccountId", '='}, {"ByResourceType", "ResourceType", '='}, {"ByStatus", "Status", '='},
		{"ByRestoreTestingPlanArn", "RestoreTestingPlanArn", '='}, {"ByParentJobId", "ParentJobId", '='},
		{"ByCreatedBefore", "CreationDate", '<'}, {"ByCreatedAfter", "CreationDate", '>'},
		{"ByCompleteBefore", "CompletionDate", '<'}, {"ByCompleteAfter", "CompletionDate", '>'},
	}
	return st.listJobs(s, bkJobRestore, in, filters, nil, "RestoreJobsListMember", "RestoreJobs")
}

func (st *backupStore) listRestoreJobsByResource(s *Server, in map[string]any) (any, *bedrockAgentError) {
	arn := bkStr(in, "ResourceArn")
	filters := []bkFilter{
		{"ByStatus", "Status", '='},
		{"ByRecoveryPointCreationDateBefore", "RecoveryPointCreationDate", '<'},
		{"ByRecoveryPointCreationDateAfter", "RecoveryPointCreationDate", '>'},
	}
	return st.listJobs(s, bkJobRestore, in, filters, func(j *bkJob) bool { return j.rec["SourceResourceArn"] == arn }, "RestoreJobsListMember", "RestoreJobs")
}

func (st *backupStore) getRestoreJobMetadata(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobRestore, bkStr(in, "RestoreJobId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"RestoreJobId": j.rec["RestoreJobId"], "Metadata": j.metadata}, nil
}

func (st *backupStore) putRestoreValidationResult(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobRestore, bkStr(in, "RestoreJobId"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "ValidationStatus"); err != nil {
		return nil, err
	}
	if err := bkEnum(in, "ValidationStatus", "FAILED", "SUCCESSFUL", "TIMED_OUT", "VALIDATING"); err != nil {
		return nil, err
	}
	if j.state() != "COMPLETED" {
		return nil, bkRequest("restore job " + bkStr(in, "RestoreJobId") + " has not completed")
	}
	if len(bkStr(in, "ValidationStatusMessage")) > 2048 {
		return nil, bkInvalid("ValidationStatusMessage must be at most 2048 characters")
	}
	j.rec["ValidationStatus"] = in["ValidationStatus"]
	j.rec["ValidationStatusMessage"] = in["ValidationStatusMessage"]
	return nil, nil
}

// --- scan jobs ---

func (st *backupStore) startScanJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "BackupVaultName", "IamRoleArn", "MalwareScanner", "RecoveryPointArn", "ScanMode", "ScannerRoleArn"); err != nil {
		return nil, err
	}
	for _, key := range []string{"IamRoleArn", "ScannerRoleArn"} {
		if err := bkCheckRole(in, key); err != nil {
			return nil, err
		}
	}
	if err := bkEnum(in, "MalwareScanner", "GUARDDUTY"); err != nil {
		return nil, err
	}
	if err := bkEnum(in, "ScanMode", "FULL_SCAN", "INCREMENTAL_SCAN"); err != nil {
		return nil, err
	}
	if j := st.idempotent(bkJobScan, in); j != nil {
		return j.rec, nil
	}
	v, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	if rp.rec["Status"] != "COMPLETED" {
		return nil, bkRequest("recovery point " + bkStr(in, "RecoveryPointArn") + " is not completed")
	}
	if base := bkStr(in, "ScanBaseRecoveryPointArn"); base != "" && st.recoveryPoints[base] == nil {
		return nil, bkNotFound("Recovery point", base)
	}
	j := &bkJob{kind: bkJobScan, stateKey: "State", steps: []string{"CREATED", "RUNNING", "COMPLETED"}, rec: map[string]any{
		"BackupVaultArn": v.rec["BackupVaultArn"], "BackupVaultName": v.rec["BackupVaultName"], "IamRoleArn": in["IamRoleArn"],
		"MalwareScanner": in["MalwareScanner"], "RecoveryPointArn": rp.rec["RecoveryPointArn"], "ResourceArn": rp.rec["ResourceArn"],
		"ResourceName": rp.rec["ResourceName"], "ResourceType": rp.rec["ResourceType"], "ScanMode": in["ScanMode"],
		"ScannerRoleArn": in["ScannerRoleArn"], "ScanBaseRecoveryPointArn": in["ScanBaseRecoveryPointArn"],
		"State": "CREATED",
		// ponytail: on-demand scans have no plan, but the model requires a creator.
		"CreatedBy": map[string]any{"BackupPlanArn": "", "BackupPlanId": "", "BackupPlanVersion": "", "BackupRuleId": ""},
	}}
	id, err := st.addJob(j, "ScanJobId")
	if err != nil {
		return nil, err
	}
	st.remember(bkJobScan, in, id)
	return map[string]any{"ScanJobId": id, "CreationDate": j.rec["CreationDate"]}, nil
}

func (st *backupStore) describeScanJob(s *Server, in map[string]any) (any, *bedrockAgentError) {
	j, err := st.job(bkJobScan, bkStr(in, "ScanJobId"))
	if err != nil {
		return nil, err
	}
	st.tick(s, j)
	return j.rec, nil
}

func (st *backupStore) listScanJobs(s *Server, in map[string]any) (any, *bedrockAgentError) {
	filters := []bkFilter{
		{"ByAccountId", "AccountId", '='}, {"ByBackupVaultName", "BackupVaultName", '='}, {"ByMalwareScanner", "MalwareScanner", '='},
		{"ByRecoveryPointArn", "RecoveryPointArn", '='}, {"ByResourceArn", "ResourceArn", '='}, {"ByResourceType", "ResourceType", '='},
		{"ByState", "State", '='},
		{"ByCompleteBefore", "CompletionDate", '<'}, {"ByCompleteAfter", "CompletionDate", '>'},
	}
	status := bkStr(in, "ByScanResultStatus")
	return st.listJobs(s, bkJobScan, in, filters, func(j *bkJob) bool {
		return status == "" || bkStr(bkMap(j.rec, "ScanResult"), "ScanResultStatus") == status
	}, "ScanJob", "ScanJobs")
}

func (st *backupStore) getPITRScanResults(s *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "RecoveryPointArn", "BackupVaultName", "ScanEndTime", "MalwareScanner"); err != nil {
		return nil, err
	}
	if _, _, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn")); err != nil {
		return nil, err
	}
	end, _, err := bkTime(in, "ScanEndTime")
	if err != nil {
		return nil, err
	}
	var latest *bkJob
	for _, j := range st.jobs {
		if j.kind != bkJobScan || j.rec["RecoveryPointArn"] != in["RecoveryPointArn"] || j.rec["MalwareScanner"] != in["MalwareScanner"] {
			continue
		}
		st.tick(s, j)
		done := bkFloat(j.rec["CompletionDate"])
		if j.state() == "COMPLETED" && done <= end && (latest == nil || done > bkFloat(latest.rec["CompletionDate"])) {
			latest = j
		}
	}
	if latest == nil {
		return nil, bkNotFound("Malware scan results for", bkStr(in, "RecoveryPointArn"))
	}
	return map[string]any{
		"ScanEndTime": latest.rec["CompletionDate"], "LastScanJobTime": latest.rec["CompletionDate"],
		"ScanResult": latest.rec["ScanResult"], "ScanId": latest.rec["ScanId"], "ScanMode": latest.rec["ScanMode"],
	}, nil
}

// --- job summaries ---

// listJobSummaries counts jobs of one kind created in the aggregation
// period, grouped by state, resource type and message category.
func bkJobSummaries(kind, listKey string) bkHandler {
	return func(st *backupStore, s *Server, in map[string]any) (any, *bedrockAgentError) {
		if err := bkEnum(in, "AggregationPeriod", "ONE_DAY", "SEVEN_DAYS", "FOURTEEN_DAYS"); err != nil {
			return nil, err
		}
		period := bkStr(in, "AggregationPeriod")
		if period == "" {
			period = "ONE_DAY"
		}
		days := map[string]float64{"ONE_DAY": 1, "SEVEN_DAYS": 7, "FOURTEEN_DAYS": 14}[period]
		end := bkNow()
		start := end - days*bkDaySeconds
		groups := map[string]map[string]any{}
		for _, j := range st.jobs {
			if j.kind != kind {
				continue
			}
			st.tick(s, j)
			created := bkFloat(j.rec["CreationDate"])
			if created < start {
				continue
			}
			state, rtype, cat := j.state(), bkStr(j.rec, "ResourceType"), bkStr(j.rec, "MessageCategory")
			if want := bkStr(in, "State"); want != "" && want != "ANY" && want != "AGGREGATE_ALL" && want != state {
				continue
			}
			if want := bkStr(in, "ResourceType"); want != "" && want != "ANY" && want != "AGGREGATE_ALL" && want != rtype {
				continue
			}
			if want := bkStr(in, "MessageCategory"); want != "" && want != "ANY" && want != "AGGREGATE_ALL" && want != cat {
				continue
			}
			if want := bkStr(in, "MalwareScanner"); want != "" && want != j.rec["MalwareScanner"] {
				continue
			}
			if want := bkStr(in, "ScanResultStatus"); want != "" && want != bkStr(bkMap(j.rec, "ScanResult"), "ScanResultStatus") {
				continue
			}
			key := state + "\x00" + rtype + "\x00" + cat
			g := groups[key]
			if g == nil {
				g = map[string]any{"Region": backupRegion, "AccountId": backupAccount, "State": state, "ResourceType": rtype, "Count": 0, "StartTime": start, "EndTime": end}
				if kind == bkJobBackup || kind == bkJobCopy {
					g["MessageCategory"] = cat
				}
				if kind == bkJobScan {
					g["MalwareScanner"] = j.rec["MalwareScanner"]
					g["ScanResultStatus"] = bkStr(bkMap(j.rec, "ScanResult"), "ScanResultStatus")
				}
				groups[key] = g
			}
			g["Count"] = g["Count"].(int) + 1
		}
		items := bkSorted(groups, bkSelf, "StartTime")
		out, err := bkPage(items, in, listKey)
		if err != nil {
			return nil, err
		}
		out["AggregationPeriod"] = period
		return out, nil
	}
}
