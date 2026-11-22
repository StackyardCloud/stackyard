package server

import (
	"encoding/json"
	"regexp"
	"strings"
)

const (
	bkVaultStandard      = "BACKUP_VAULT"
	bkVaultAirGapped     = "LOGICALLY_AIR_GAPPED_BACKUP_VAULT"
	bkVaultRestoreAccess = "RESTORE_ACCESS_BACKUP_VAULT"
	bkMaxRetentionDays   = 36500
	bkDaySeconds         = 86400
)

var (
	bkSNSTopicPattern = regexp.MustCompile(`^arn:aws[a-z-]*:sns:[a-z0-9-]+:[0-9]{12}:.+$`)
	bkMpaTeamPattern  = regexp.MustCompile(`^arn:aws[a-z-]*:mpa:.+$`)
	bkVaultEvents     = []string{"BACKUP_JOB_STARTED", "BACKUP_JOB_COMPLETED", "BACKUP_JOB_SUCCESSFUL", "BACKUP_JOB_FAILED", "BACKUP_JOB_EXPIRED", "RESTORE_JOB_STARTED", "RESTORE_JOB_COMPLETED", "RESTORE_JOB_SUCCESSFUL", "RESTORE_JOB_FAILED", "COPY_JOB_STARTED", "COPY_JOB_SUCCESSFUL", "COPY_JOB_FAILED", "RECOVERY_POINT_MODIFIED", "BACKUP_PLAN_CREATED", "BACKUP_PLAN_MODIFIED", "S3_BACKUP_OBJECT_FAILED", "S3_RESTORE_OBJECT_FAILED", "CONTINUOUS_BACKUP_INTERRUPTED", "RECOVERY_POINT_INDEX_COMPLETED", "RECOVERY_POINT_INDEX_DELETED", "RECOVERY_POINT_INDEXING_FAILED", "EKS_RESTORE_OBJECT_FAILED", "EKS_RESTORE_OBJECT_SKIPPED", "EKS_BACKUP_OBJECT_FAILED", "EKS_RESTORE_MODIFICATION_SUCCESS", "EKS_RESTORE_MODIFICATION_FAILED", "ACCESS_POINT_AVAILABLE", "ACCESS_POINT_CREATION_FAILED", "ACCESS_POINT_DELETED", "ACCESS_POINT_DELETION_FAILED", "ACCESS_POINT_EXPIRED", "ACCESS_POINT_DISASSOCIATED"}
)

// bkVault is a vault record plus its policy, notifications and (for an
// air-gapped vault) the restore-access vaults that point at it.
type bkVault struct {
	rec           map[string]any
	policy        string
	notifications map[string]any
	restoreAccess map[string]map[string]any // by restore-access vault ARN
	pending       bool                      // CREATING until first read
}

func bkVaultHandlers() map[string]bkHandler {
	return map[string]bkHandler{
		"CreateBackupVault":                      (*backupStore).createBackupVault,
		"CreateLogicallyAirGappedBackupVault":    (*backupStore).createAirGappedVault,
		"CreateRestoreAccessBackupVault":         (*backupStore).createRestoreAccessVault,
		"DescribeBackupVault":                    (*backupStore).describeBackupVault,
		"DeleteBackupVault":                      (*backupStore).deleteBackupVault,
		"ListBackupVaults":                       (*backupStore).listBackupVaults,
		"PutBackupVaultAccessPolicy":             (*backupStore).putVaultPolicy,
		"GetBackupVaultAccessPolicy":             (*backupStore).getVaultPolicy,
		"DeleteBackupVaultAccessPolicy":          (*backupStore).deleteVaultPolicy,
		"PutBackupVaultNotifications":            (*backupStore).putVaultNotifications,
		"GetBackupVaultNotifications":            (*backupStore).getVaultNotifications,
		"DeleteBackupVaultNotifications":         (*backupStore).deleteVaultNotifications,
		"PutBackupVaultLockConfiguration":        (*backupStore).putVaultLock,
		"DeleteBackupVaultLockConfiguration":     (*backupStore).deleteVaultLock,
		"AssociateBackupVaultMpaApprovalTeam":    (*backupStore).associateMpaTeam,
		"DisassociateBackupVaultMpaApprovalTeam": (*backupStore).disassociateMpaTeam,
		"ListRestoreAccessBackupVaults":          (*backupStore).listRestoreAccessVaults,
		"RevokeRestoreAccessBackupVault":         (*backupStore).revokeRestoreAccessVault,
	}
}

// vault looks a vault up by name and settles its CREATING state.
func (st *backupStore) vault(name string) (*bkVault, *bedrockAgentError) {
	v := st.vaults[name]
	if v == nil {
		return nil, bkNotFound("Backup vault", name)
	}
	if v.pending {
		v.pending = false
		v.rec["VaultState"] = "AVAILABLE"
	}
	return v, nil
}

func (st *backupStore) vaultByARN(arn string) *bkVault {
	for _, v := range st.vaults {
		if v.rec["BackupVaultArn"] == arn {
			return v
		}
	}
	return nil
}

func (st *backupStore) vaultRecoveryPoints(name string) int {
	n := 0
	for _, rp := range st.recoveryPoints {
		if rp.rec["BackupVaultName"] == name {
			n++
		}
	}
	return n
}

// newVault validates the shared create members and registers the vault.
func (st *backupStore) newVault(in map[string]any, name, vaultType string) (*bkVault, *bedrockAgentError) {
	if !bkVaultNamePattern.MatchString(name) {
		return nil, bkInvalid("BackupVaultName must be 2-50 letters, numbers, hyphens or underscores")
	}
	if err := bkCheckToken(in, "CreatorRequestId"); err != nil {
		return nil, err
	}
	if v := st.vaults[name]; v != nil {
		if tok := bkStr(in, "CreatorRequestId"); tok != "" && v.rec["CreatorRequestId"] == tok {
			return v, nil // idempotent retry
		}
		return nil, bkExists("Backup vault", name)
	}
	if len(st.vaults) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many backup vaults")
	}
	keyType, key := "AWS_OWNED_KMS_KEY", bkStr(in, "EncryptionKeyArn")
	if key != "" {
		if !strings.HasPrefix(key, "arn:aws") || !strings.Contains(key, ":kms:") {
			return nil, bkInvalid("EncryptionKeyArn must be a KMS key ARN")
		}
		keyType = "CUSTOMER_MANAGED_KMS_KEY"
	} else if vaultType == bkVaultStandard {
		key = "arn:aws:kms:" + backupRegion + ":" + backupAccount + ":key/" + bkNewID()
	}
	arn := bkARN("backup-vault", name)
	rec := map[string]any{
		"BackupVaultName":   name,
		"BackupVaultArn":    arn,
		"VaultType":         vaultType,
		"VaultState":        "AVAILABLE",
		"EncryptionKeyArn":  key,
		"EncryptionKeyType": keyType,
		"CreationDate":      bkNow(),
		"CreatorRequestId":  in["CreatorRequestId"],
		"Locked":            false,
	}
	if key == "" {
		delete(rec, "EncryptionKeyArn")
	}
	if err := st.setTags(arn, bkMap(in, "BackupVaultTags")); err != nil {
		return nil, err
	}
	v := &bkVault{rec: rec, restoreAccess: map[string]map[string]any{}}
	st.vaults[name] = v
	return v, nil
}

func (st *backupStore) createBackupVault(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.newVault(in, bkStr(in, "BackupVaultName"), bkVaultStandard)
	if err != nil {
		return nil, err
	}
	return v.rec, nil
}

func bkRetention(in map[string]any, required bool) (minDays, maxDays int, err *bedrockAgentError) {
	minDays, minAbsent, err := bkInt(in, "MinRetentionDays")
	if err != nil {
		return 0, 0, err
	}
	maxDays, maxAbsent, err := bkInt(in, "MaxRetentionDays")
	if err != nil {
		return 0, 0, err
	}
	if required && (minAbsent || maxAbsent) {
		return 0, 0, bkMissing("MinRetentionDays and MaxRetentionDays are required")
	}
	if !minAbsent && (minDays < 1 || minDays > bkMaxRetentionDays) {
		return 0, 0, bkInvalid("MinRetentionDays must be between 1 and 36500")
	}
	if !maxAbsent && (maxDays < 1 || maxDays > bkMaxRetentionDays) {
		return 0, 0, bkInvalid("MaxRetentionDays must be between 1 and 36500")
	}
	if !minAbsent && !maxAbsent && minDays > maxDays {
		return 0, 0, bkInvalid("MinRetentionDays must not exceed MaxRetentionDays")
	}
	return minDays, maxDays, nil
}

func (st *backupStore) createAirGappedVault(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	minDays, maxDays, err := bkRetention(in, true)
	if err != nil {
		return nil, err
	}
	if minDays < 7 {
		return nil, bkInvalid("MinRetentionDays must be at least 7 for a logically air-gapped vault")
	}
	if st.vaults[bkStr(in, "BackupVaultName")] != nil {
		return nil, bkExists("Backup vault", bkStr(in, "BackupVaultName"))
	}
	v, err := st.newVault(in, bkStr(in, "BackupVaultName"), bkVaultAirGapped)
	if err != nil {
		return nil, err
	}
	v.pending = true
	v.rec["VaultState"] = "CREATING"
	v.rec["Locked"] = true
	v.rec["LockDate"] = v.rec["CreationDate"]
	v.rec["MinRetentionDays"] = minDays
	v.rec["MaxRetentionDays"] = maxDays
	return v.rec, nil
}

func (st *backupStore) createRestoreAccessVault(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "SourceBackupVaultArn"); err != nil {
		return nil, err
	}
	src := st.vaultByARN(bkStr(in, "SourceBackupVaultArn"))
	if src == nil {
		return nil, bkNotFound("Backup vault", bkStr(in, "SourceBackupVaultArn"))
	}
	if src.rec["VaultType"] != bkVaultAirGapped {
		return nil, bkInvalid("SourceBackupVaultArn must name a logically air-gapped backup vault")
	}
	if len(bkStr(in, "RequesterComment")) > 1024 {
		return nil, bkInvalid("RequesterComment must be at most 1024 characters")
	}
	name := bkStr(in, "BackupVaultName")
	if name == "" {
		name = "restore-access-" + strings.ReplaceAll(bkNewID(), "-", "")[:20]
	}
	v, err := st.newVault(in, name, bkVaultRestoreAccess)
	if err != nil {
		return nil, err
	}
	v.pending = true
	v.rec["VaultState"] = "CREATING"
	v.rec["SourceBackupVaultArn"] = src.rec["BackupVaultArn"]
	src.restoreAccess[v.rec["BackupVaultArn"].(string)] = map[string]any{
		"RestoreAccessBackupVaultArn": v.rec["BackupVaultArn"],
		"CreationDate":                v.rec["CreationDate"],
		"ApprovalDate":                v.rec["CreationDate"],
		"VaultState":                  "AVAILABLE",
	}
	return map[string]any{
		"RestoreAccessBackupVaultArn":  v.rec["BackupVaultArn"],
		"RestoreAccessBackupVaultName": name,
		"VaultState":                   "CREATING",
		"CreationDate":                 v.rec["CreationDate"],
	}, nil
}

func (st *backupStore) describeBackupVault(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	out := bkShape(v.rec, "BackupVaultListMember")
	for _, k := range []string{"SourceBackupVaultArn", "MpaApprovalTeamArn", "LatestMpaApprovalTeamUpdate"} {
		if v.rec[k] != nil {
			out[k] = v.rec[k]
		}
	}
	out["NumberOfRecoveryPoints"] = st.vaultRecoveryPoints(v.rec["BackupVaultName"].(string))
	return out, nil
}

func (st *backupStore) deleteBackupVault(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	name := bkStr(in, "BackupVaultName")
	v, err := st.vault(name)
	if err != nil {
		return nil, err
	}
	if st.vaultRecoveryPoints(name) > 0 {
		return nil, bkRequest("Backup vault " + name + " cannot be deleted because it contains recovery points")
	}
	if len(v.restoreAccess) > 0 {
		return nil, bkRequest("Backup vault " + name + " has restore access backup vaults; revoke them first")
	}
	arn, _ := v.rec["BackupVaultArn"].(string)
	if src := st.vaultByARN(bkStr(v.rec, "SourceBackupVaultArn")); src != nil {
		delete(src.restoreAccess, arn)
	}
	delete(st.tags, arn)
	delete(st.vaults, name)
	return nil, nil
}

func (st *backupStore) listBackupVaults(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkEnum(in, "ByVaultType", bkVaultStandard, bkVaultAirGapped, bkVaultRestoreAccess); err != nil {
		return nil, err
	}
	items := []map[string]any{}
	if baBool(in["ByShared"]) {
		// Nothing is shared with this single emulated account.
		return bkPage(items, in, "BackupVaultList")
	}
	for _, v := range bkSorted(st.vaults, func(v *bkVault) map[string]any { return v.rec }, "CreationDate") {
		if t := bkStr(in, "ByVaultType"); t != "" && v.rec["VaultType"] != t {
			continue
		}
		v, _ = st.vault(v.rec["BackupVaultName"].(string))
		item := bkShape(v.rec, "BackupVaultListMember")
		item["NumberOfRecoveryPoints"] = st.vaultRecoveryPoints(v.rec["BackupVaultName"].(string))
		items = append(items, item)
	}
	return bkPage(items, in, "BackupVaultList")
}

func (st *backupStore) putVaultPolicy(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "Policy"); err != nil {
		return nil, err
	}
	policy := bkStr(in, "Policy")
	var doc map[string]any
	if len(policy) > 20480 || json.Unmarshal([]byte(policy), &doc) != nil || doc["Statement"] == nil {
		return nil, bkInvalid("Policy must be a JSON policy document with a Statement")
	}
	v.policy = policy
	return nil, nil
}

func (st *backupStore) getVaultPolicy(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if v.policy == "" {
		return nil, bkNotFound("Access policy for backup vault", bkStr(in, "BackupVaultName"))
	}
	return map[string]any{"BackupVaultName": v.rec["BackupVaultName"], "BackupVaultArn": v.rec["BackupVaultArn"], "Policy": v.policy}, nil
}

func (st *backupStore) deleteVaultPolicy(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	v.policy = ""
	return nil, nil
}

func (st *backupStore) putVaultNotifications(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "SNSTopicArn", "BackupVaultEvents"); err != nil {
		return nil, err
	}
	if !bkSNSTopicPattern.MatchString(bkStr(in, "SNSTopicArn")) {
		return nil, bkInvalid("SNSTopicArn must be an SNS topic ARN")
	}
	events := bkList(in, "BackupVaultEvents")
	if len(events) == 0 || len(events) > len(bkVaultEvents) {
		return nil, bkInvalid("BackupVaultEvents must list at least one event")
	}
	for _, e := range events {
		if s, _ := e.(string); !containsString(bkVaultEvents, s) {
			return nil, bkInvalid("unsupported backup vault event " + s)
		}
	}
	v.notifications = map[string]any{"SNSTopicArn": in["SNSTopicArn"], "BackupVaultEvents": events}
	return nil, nil
}

func (st *backupStore) getVaultNotifications(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if v.notifications == nil {
		return nil, bkNotFound("Notification configuration for backup vault", bkStr(in, "BackupVaultName"))
	}
	return map[string]any{
		"BackupVaultName": v.rec["BackupVaultName"], "BackupVaultArn": v.rec["BackupVaultArn"],
		"SNSTopicArn": v.notifications["SNSTopicArn"], "BackupVaultEvents": v.notifications["BackupVaultEvents"],
	}, nil
}

func (st *backupStore) deleteVaultNotifications(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	v.notifications = nil
	return nil, nil
}

// complianceLocked reports a vault whose lock's grace period has ended.
func (v *bkVault) complianceLocked() bool {
	lock := bkFloat(v.rec["LockDate"])
	return v.rec["Locked"] == true && lock > 0 && lock <= bkNow()
}

func (st *backupStore) putVaultLock(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if v.rec["VaultType"] != bkVaultStandard {
		return nil, bkRequest("vault lock can only be configured on a standard backup vault")
	}
	if v.complianceLocked() {
		return nil, bkRequest("the vault lock is in compliance mode and can no longer be changed")
	}
	minDays, maxDays, err := bkRetention(in, false)
	if err != nil {
		return nil, err
	}
	changeable, changeAbsent, err := bkInt(in, "ChangeableForDays")
	if err != nil {
		return nil, err
	}
	if !changeAbsent && (changeable < 3 || changeable > bkMaxRetentionDays) {
		return nil, bkInvalid("ChangeableForDays must be between 3 and 36500")
	}
	v.rec["Locked"] = true
	delete(v.rec, "MinRetentionDays")
	delete(v.rec, "MaxRetentionDays")
	delete(v.rec, "LockDate")
	if minDays > 0 {
		v.rec["MinRetentionDays"] = minDays
	}
	if maxDays > 0 {
		v.rec["MaxRetentionDays"] = maxDays
	}
	if !changeAbsent {
		v.rec["LockDate"] = bkNow() + float64(changeable*bkDaySeconds)
	}
	return nil, nil
}

func (st *backupStore) deleteVaultLock(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if v.rec["VaultType"] != bkVaultStandard {
		return nil, bkRequest("the lock of a logically air-gapped vault cannot be removed")
	}
	if v.complianceLocked() {
		return nil, bkRequest("the vault lock is in compliance mode and cannot be deleted")
	}
	v.rec["Locked"] = false
	for _, k := range []string{"MinRetentionDays", "MaxRetentionDays", "LockDate"} {
		delete(v.rec, k)
	}
	return nil, nil
}

func (st *backupStore) associateMpaTeam(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "MpaApprovalTeamArn"); err != nil {
		return nil, err
	}
	if !bkMpaTeamPattern.MatchString(bkStr(in, "MpaApprovalTeamArn")) {
		return nil, bkInvalid("MpaApprovalTeamArn must be a Multi-party approval team ARN")
	}
	if v.rec["VaultType"] != bkVaultAirGapped {
		return nil, bkInvalid("an approval team can only be associated with a logically air-gapped backup vault")
	}
	v.rec["MpaApprovalTeamArn"] = in["MpaApprovalTeamArn"]
	v.rec["LatestMpaApprovalTeamUpdate"] = map[string]any{"Status": "APPROVED", "InitiationDate": bkNow()}
	return nil, nil
}

func (st *backupStore) disassociateMpaTeam(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if v.rec["MpaApprovalTeamArn"] == nil {
		return nil, bkRequest("backup vault " + bkStr(in, "BackupVaultName") + " has no approval team")
	}
	delete(v.rec, "MpaApprovalTeamArn")
	delete(v.rec, "LatestMpaApprovalTeamUpdate")
	return nil, nil
}

func (st *backupStore) listRestoreAccessVaults(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, rec := range bkSorted(v.restoreAccess, bkSelf, "CreationDate") {
		items = append(items, bkShape(rec, "RestoreAccessBackupVaultListMember"))
	}
	return bkPage(items, in, "RestoreAccessBackupVaults")
}

func (st *backupStore) revokeRestoreAccessVault(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	arn := bkStr(in, "RestoreAccessBackupVaultArn")
	if v.restoreAccess[arn] == nil {
		return nil, bkNotFound("Restore access backup vault", arn)
	}
	delete(v.restoreAccess, arn)
	if rv := st.vaultByARN(arn); rv != nil {
		delete(st.tags, arn)
		delete(st.vaults, rv.rec["BackupVaultName"].(string))
	}
	return nil, nil
}
