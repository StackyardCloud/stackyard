package server

import (
	"encoding/json"
	"regexp"
	"strings"
)

var bkAccessPointNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// bkRecoveryPoint is a recovery point record plus the job that creates it and,
// for DynamoDB, the table backup that holds its data.
type bkRecoveryPoint struct {
	rec             map[string]any
	jobID           string
	dynamoBackupARN string
	index           map[string]any // GetRecoveryPointIndexDetails members, nil when never indexed
}

func bkRecoveryPointHandlers() map[string]bkHandler {
	return map[string]bkHandler{
		"DescribeRecoveryPoint":                 (*backupStore).describeRecoveryPoint,
		"DeleteRecoveryPoint":                   (*backupStore).deleteRecoveryPoint,
		"UpdateRecoveryPointLifecycle":          (*backupStore).updateRecoveryPointLifecycle,
		"ListRecoveryPointsByBackupVault":       (*backupStore).listRecoveryPointsByVault,
		"ListRecoveryPointsByResource":          (*backupStore).listRecoveryPointsByResource,
		"ListRecoveryPointsByLegalHold":         (*backupStore).listRecoveryPointsByLegalHold,
		"GetRecoveryPointRestoreMetadata":       (*backupStore).getRestoreMetadata,
		"GetRestoreTestingInferredMetadata":     (*backupStore).getInferredMetadata,
		"DisassociateRecoveryPoint":             (*backupStore).disassociateRecoveryPoint,
		"DisassociateRecoveryPointFromParent":   (*backupStore).disassociateFromParent,
		"UpdateRecoveryPointIndexSettings":      (*backupStore).updateIndexSettings,
		"GetRecoveryPointIndexDetails":          (*backupStore).getIndexDetails,
		"ListIndexedRecoveryPoints":             (*backupStore).listIndexedRecoveryPoints,
		"DescribeProtectedResource":             (*backupStore).describeProtectedResource,
		"ListProtectedResources":                (*backupStore).listProtectedResources,
		"ListProtectedResourcesByBackupVault":   (*backupStore).listProtectedResourcesByVault,
		"CreateLegalHold":                       (*backupStore).createLegalHold,
		"GetLegalHold":                          (*backupStore).getLegalHold,
		"CancelLegalHold":                       (*backupStore).cancelLegalHold,
		"ListLegalHolds":                        (*backupStore).listLegalHolds,
		"CreateBackupAccessPoint":               (*backupStore).createAccessPoint,
		"DescribeBackupAccessPoint":             (*backupStore).describeAccessPoint,
		"DeleteBackupAccessPoint":               (*backupStore).deleteAccessPoint,
		"ListBackupAccessPoints":                bkListAccessPoints(""),
		"ListBackupAccessPointsByRecoveryPoint": bkListAccessPoints("RecoveryPointArn"),
		"ListBackupAccessPointsByResource":      bkListAccessPoints("ResourceArn"),
	}
}

// tickRecoveryPoint advances the job that is still creating rp, so reading
// a recovery point shows its backup progressing.
func (st *backupStore) tickRecoveryPoint(s *Server, rp *bkRecoveryPoint) {
	if j := st.jobs[rp.jobID]; j != nil {
		st.tick(s, j)
	}
}

func (st *backupStore) vaultRecoveryPoint(vaultName, arn string) (*bkVault, *bkRecoveryPoint, *bedrockAgentError) {
	v, err := st.vault(vaultName)
	if err != nil {
		return nil, nil, err
	}
	rp := st.recoveryPoints[arn]
	if rp == nil || rp.rec["BackupVaultName"] != v.rec["BackupVaultName"] {
		return nil, nil, bkNotFound("Recovery point", arn)
	}
	return v, rp, nil
}

func bkIndexable(resourceType string) bool { return resourceType == "EBS" || resourceType == "S3" }

func (st *backupStore) describeRecoveryPoint(s *Server, in map[string]any) (any, *bedrockAgentError) {
	_, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	st.tickRecoveryPoint(s, rp)
	return rp.rec, nil
}

// heldBy reports an active legal hold that covers rp.
func (st *backupStore) heldBy(rp *bkRecoveryPoint) string {
	for id, hold := range st.legalHolds {
		if hold["Status"] == "ACTIVE" && bkHoldCovers(hold, rp) {
			return id
		}
	}
	return ""
}

func (st *backupStore) deleteRecoveryPoint(s *Server, in map[string]any) (any, *bedrockAgentError) {
	v, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	if rp.rec["Status"] == "CREATING" {
		return nil, bkErr("InvalidResourceStateException", "the recovery point is still being created")
	}
	if hold := st.heldBy(rp); hold != "" {
		return nil, bkErr("InvalidResourceStateException", "the recovery point is under legal hold "+hold)
	}
	if lo, ok := v.rec["MinRetentionDays"].(int); ok && v.rec["Locked"] == true {
		if bkNow()-bkFloat(rp.rec["CreationDate"]) < float64(lo*bkDaySeconds) {
			return nil, bkRequest("the vault lock keeps recovery points for at least the minimum retention period")
		}
	}
	st.dropRecoveryPoint(s, rp, true)
	return nil, nil
}

// dropRecoveryPoint removes rp and its access points; withData also deletes
// the DynamoDB backup behind it unless another recovery point (a copy) uses it.
func (st *backupStore) dropRecoveryPoint(s *Server, rp *bkRecoveryPoint, withData bool) {
	arn := bkStr(rp.rec, "RecoveryPointArn")
	delete(st.recoveryPoints, arn)
	delete(st.tags, arn)
	for id, ap := range st.accessPoints {
		if ap["RecoveryPointArn"] == arn {
			delete(st.accessPoints, id)
		}
	}
	if !withData || rp.dynamoBackupARN == "" {
		return
	}
	for _, other := range st.recoveryPoints {
		if other.dynamoBackupARN == rp.dynamoBackupARN {
			return
		}
	}
	_, _ = s.dynamodb.DeleteBackup(rp.dynamoBackupARN)
}

func (st *backupStore) updateRecoveryPointLifecycle(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	lc := bkMap(in, "Lifecycle")
	if err := bkCheckLifecycle(lc); err != nil {
		return nil, err
	}
	if err := bkCheckVaultRetention(v, lc); err != nil {
		return nil, err
	}
	if old := bkMap(rp.rec, "Lifecycle"); old != nil {
		if was, _, _ := bkInt(old, "MoveToColdStorageAfterDays"); was > 0 {
			if now, absent, _ := bkInt(lc, "MoveToColdStorageAfterDays"); absent || now != was {
				if bkNow()-bkFloat(rp.rec["CreationDate"]) >= float64(was*bkDaySeconds) {
					return nil, bkInvalid("a recovery point already in cold storage cannot change MoveToColdStorageAfterDays")
				}
			}
		}
	}
	rp.rec["Lifecycle"] = lc
	rp.rec["CalculatedLifecycle"] = bkCalculatedLifecycle(lc, bkFloat(rp.rec["CreationDate"]))
	return rp.rec, nil
}

func (st *backupStore) listRecoveryPointsByVault(s *Server, in map[string]any) (any, *bedrockAgentError) {
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	if bkStr(in, "ByBackupPlanId") != "" {
		// Recovery points here come from on-demand jobs, never from a plan.
		return bkPage([]map[string]any{}, in, "RecoveryPoints")
	}
	filters := []bkFilter{
		{"ByResourceArn", "ResourceArn", '='}, {"ByResourceType", "ResourceType", '='},
		{"ByParentRecoveryPointArn", "ParentRecoveryPointArn", '='},
		{"ByCreatedBefore", "CreationDate", '<'}, {"ByCreatedAfter", "CreationDate", '>'},
	}
	items := []map[string]any{}
	for _, rp := range bkSorted(st.recoveryPoints, func(rp *bkRecoveryPoint) map[string]any { return rp.rec }, "CreationDate") {
		if rp.rec["BackupVaultName"] != v.rec["BackupVaultName"] {
			continue
		}
		st.tickRecoveryPoint(s, rp)
		if ok, err := bkMatch(rp.rec, in, filters); err != nil {
			return nil, err
		} else if ok {
			items = append(items, bkShape(rp.rec, "RecoveryPointByBackupVault"))
		}
	}
	return bkPage(items, in, "RecoveryPoints")
}

func (st *backupStore) listRecoveryPointsByResource(s *Server, in map[string]any) (any, *bedrockAgentError) {
	arn := bkStr(in, "ResourceArn")
	if _, _, ok := bkResourceType(arn); !ok {
		return nil, bkInvalid("ResourceArn " + arn + " is not a resource type AWS Backup supports")
	}
	items := []map[string]any{}
	for _, rp := range bkSorted(st.recoveryPoints, func(rp *bkRecoveryPoint) map[string]any { return rp.rec }, "CreationDate") {
		if rp.rec["ResourceArn"] != arn {
			continue
		}
		st.tickRecoveryPoint(s, rp)
		item := bkShape(rp.rec, "RecoveryPointByResource")
		item["BackupSizeBytes"] = rp.rec["BackupSizeInBytes"]
		items = append(items, item)
	}
	return bkPage(items, in, "RecoveryPoints")
}

func bkRestoreMetadata(rp *bkRecoveryPoint) map[string]any {
	meta := map[string]any{"resourceArn": rp.rec["ResourceArn"]}
	if rp.rec["ResourceType"] == "DynamoDB" {
		meta = map[string]any{"targetTableName": rp.rec["ResourceName"], "encryptionType": "Default"}
	}
	return meta
}

func (st *backupStore) getRestoreMetadata(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	v, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"BackupVaultArn": v.rec["BackupVaultArn"], "RecoveryPointArn": rp.rec["RecoveryPointArn"], "ResourceType": rp.rec["ResourceType"], "RestoreMetadata": bkRestoreMetadata(rp)}, nil
}

func (st *backupStore) getInferredMetadata(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "BackupVaultName", "RecoveryPointArn"); err != nil {
		return nil, err
	}
	if acct := bkStr(in, "BackupVaultAccountId"); acct != "" && acct != backupAccount {
		return nil, bkNotFound("Backup vault", bkStr(in, "BackupVaultName"))
	}
	_, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"InferredMetadata": bkRestoreMetadata(rp)}, nil
}

// disassociateRecoveryPoint stops AWS Backup managing rp; the underlying data
// (a DynamoDB backup) stays with the source service.
// ponytail: any recovery point may be disassociated, not only continuous ones.
func (st *backupStore) disassociateRecoveryPoint(s *Server, in map[string]any) (any, *bedrockAgentError) {
	_, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	if rp.rec["Status"] == "CREATING" {
		return nil, bkErr("InvalidResourceStateException", "the recovery point is still being created")
	}
	st.dropRecoveryPoint(s, rp, false)
	return nil, nil
}

// ponytail: recovery points here have no composite parents, so this only
// clears a ParentRecoveryPointArn if one was ever set.
func (st *backupStore) disassociateFromParent(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	_, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	delete(rp.rec, "ParentRecoveryPointArn")
	return nil, nil
}

func (st *backupStore) updateIndexSettings(s *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "Index"); err != nil {
		return nil, err
	}
	if err := bkEnum(in, "Index", "ENABLED", "DISABLED"); err != nil {
		return nil, err
	}
	if err := bkCheckRole(in, "IamRoleArn"); err != nil {
		return nil, err
	}
	v, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	if !bkIndexable(bkStr(rp.rec, "ResourceType")) {
		return nil, bkInvalid("backup indexes are only supported for EBS and S3 recovery points")
	}
	now := bkNow()
	if bkStr(in, "Index") == "ENABLED" {
		if rp.rec["Status"] != "COMPLETED" {
			return nil, bkErr("InvalidResourceStateException", "the recovery point is not completed")
		}
		rp.rec["IndexStatus"] = "ACTIVE"
		rp.index = map[string]any{
			"RecoveryPointArn": rp.rec["RecoveryPointArn"], "BackupVaultArn": v.rec["BackupVaultArn"],
			"SourceResourceArn": rp.rec["ResourceArn"], "IndexCreationDate": now, "IndexCompletionDate": now,
			"IndexStatus": "ACTIVE", "TotalItemsIndexed": 0, "IamRoleArn": in["IamRoleArn"],
		}
	} else {
		if rp.index == nil {
			return nil, bkErr("InvalidResourceStateException", "the recovery point has no index")
		}
		delete(rp.rec, "IndexStatus")
		rp.index = nil
	}
	return map[string]any{"BackupVaultName": v.rec["BackupVaultName"], "RecoveryPointArn": rp.rec["RecoveryPointArn"], "IndexStatus": rp.rec["IndexStatus"], "Index": in["Index"]}, nil
}

func (st *backupStore) getIndexDetails(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	_, rp, err := st.vaultRecoveryPoint(bkStr(in, "BackupVaultName"), bkStr(in, "RecoveryPointArn"))
	if err != nil {
		return nil, err
	}
	if rp.index == nil {
		return nil, bkNotFound("Index for recovery point", bkStr(in, "RecoveryPointArn"))
	}
	return rp.index, nil
}

func (st *backupStore) listIndexedRecoveryPoints(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkEnum(in, "IndexStatus", "PENDING", "ACTIVE", "FAILED", "DELETING"); err != nil {
		return nil, err
	}
	filters := []bkFilter{
		{"SourceResourceArn", "SourceResourceArn", '='}, {"ResourceType", "ResourceType", '='}, {"IndexStatus", "IndexStatus", '='},
		{"CreatedBefore", "BackupCreationDate", '<'}, {"CreatedAfter", "BackupCreationDate", '>'},
	}
	items := []map[string]any{}
	for _, rp := range bkSorted(st.recoveryPoints, func(rp *bkRecoveryPoint) map[string]any { return rp.rec }, "CreationDate") {
		if rp.index == nil {
			continue
		}
		item := map[string]any{
			"RecoveryPointArn": rp.rec["RecoveryPointArn"], "SourceResourceArn": rp.rec["ResourceArn"], "IamRoleArn": rp.index["IamRoleArn"],
			"BackupCreationDate": rp.rec["CreationDate"], "ResourceType": rp.rec["ResourceType"], "IndexCreationDate": rp.index["IndexCreationDate"],
			"IndexStatus": rp.index["IndexStatus"], "BackupVaultArn": rp.rec["BackupVaultArn"],
		}
		if ok, err := bkMatch(item, in, filters); err != nil {
			return nil, err
		} else if ok {
			items = append(items, bkShape(item, "IndexedRecoveryPoint"))
		}
	}
	return bkPage(items, in, "IndexedRecoveryPoints")
}

// --- protected resources ---

func (st *backupStore) describeProtectedResource(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	pr := st.protected[bkStr(in, "ResourceArn")]
	if pr == nil {
		return nil, bkNotFound("Protected resource", bkStr(in, "ResourceArn"))
	}
	return pr, nil
}

func (st *backupStore) listProtectedResources(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, pr := range bkSorted(st.protected, bkSelf, "LastBackupTime") {
		items = append(items, bkShape(pr, "ProtectedResource"))
	}
	return bkPage(items, in, "Results")
}

func (st *backupStore) listProtectedResourcesByVault(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if acct := bkStr(in, "BackupVaultAccountId"); acct != "" && acct != backupAccount {
		return nil, bkNotFound("Backup vault", bkStr(in, "BackupVaultName"))
	}
	v, err := st.vault(bkStr(in, "BackupVaultName"))
	if err != nil {
		return nil, err
	}
	inVault := map[string]bool{}
	for _, rp := range st.recoveryPoints {
		if rp.rec["BackupVaultName"] == v.rec["BackupVaultName"] {
			inVault[bkStr(rp.rec, "ResourceArn")] = true
		}
	}
	items := []map[string]any{}
	for _, pr := range bkSorted(st.protected, bkSelf, "LastBackupTime") {
		if inVault[bkStr(pr, "ResourceArn")] {
			items = append(items, bkShape(pr, "ProtectedResource"))
		}
	}
	return bkPage(items, in, "Results")
}

// --- legal holds ---

// bkHoldCovers reports whether a hold's RecoveryPointSelection matches rp.
func bkHoldCovers(hold map[string]any, rp *bkRecoveryPoint) bool {
	sel := bkMap(hold, "RecoveryPointSelection")
	if sel == nil {
		return false
	}
	if vaults := bkList(sel, "VaultNames"); len(vaults) > 0 && !containsAny(vaults, rp.rec["BackupVaultName"]) {
		return false
	}
	if ids := bkList(sel, "ResourceIdentifiers"); len(ids) > 0 && !containsAny(ids, rp.rec["ResourceArn"]) {
		return false
	}
	if dr := bkMap(sel, "DateRange"); dr != nil {
		from, _, _ := bkTime(dr, "FromDate")
		to, _, _ := bkTime(dr, "ToDate")
		created := bkFloat(rp.rec["CreationDate"])
		if created < from || created > to {
			return false
		}
	}
	return true
}

func containsAny(list []any, v any) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (st *backupStore) createLegalHold(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "Title", "Description"); err != nil {
		return nil, err
	}
	if len(bkStr(in, "Title")) > 256 || len(bkStr(in, "Description")) > 1024 {
		return nil, bkInvalid("Title must be at most 256 and Description at most 1024 characters")
	}
	if tok := bkStr(in, "IdempotencyToken"); tok != "" {
		for _, h := range st.legalHolds {
			if h["IdempotencyToken"] == tok {
				return h, nil
			}
		}
	}
	sel := bkMap(in, "RecoveryPointSelection")
	if dr := bkMap(sel, "DateRange"); dr != nil {
		if err := bkRequire(dr, "FromDate", "ToDate"); err != nil {
			return nil, err
		}
		from, _, err := bkTime(dr, "FromDate")
		if err != nil {
			return nil, err
		}
		to, _, err := bkTime(dr, "ToDate")
		if err != nil {
			return nil, err
		}
		if from > to {
			return nil, bkInvalid("DateRange FromDate must not be after ToDate")
		}
	}
	for _, name := range bkList(sel, "VaultNames") {
		if s, _ := name.(string); st.vaults[s] == nil {
			return nil, bkNotFound("Backup vault", s)
		}
	}
	if len(st.legalHolds) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many legal holds")
	}
	id := bkNewID()
	arn := bkARN("legal-hold", id)
	hold := map[string]any{
		"Title": in["Title"], "Description": in["Description"], "Status": "ACTIVE", "LegalHoldId": id, "LegalHoldArn": arn,
		"CreationDate": bkNow(), "RecoveryPointSelection": sel, "IdempotencyToken": in["IdempotencyToken"],
	}
	if err := st.setTags(arn, bkMap(in, "Tags")); err != nil {
		return nil, err
	}
	st.legalHolds[id] = hold
	return hold, nil
}

func (st *backupStore) legalHold(id string) (map[string]any, *bedrockAgentError) {
	h := st.legalHolds[id]
	if h == nil {
		return nil, bkNotFound("Legal hold", id)
	}
	return h, nil
}

func (st *backupStore) getLegalHold(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	return st.legalHold(bkStr(in, "LegalHoldId"))
}

func (st *backupStore) cancelLegalHold(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	h, err := st.legalHold(bkStr(in, "LegalHoldId"))
	if err != nil {
		return nil, err
	}
	if err := bkRequire(in, "CancelDescription"); err != nil {
		return nil, err
	}
	if h["Status"] != "ACTIVE" {
		return nil, bkErr("InvalidResourceStateException", "legal hold "+bkStr(in, "LegalHoldId")+" is "+bkStr(h, "Status"))
	}
	days, absent, err := bkInt(in, "RetainRecordInDays")
	if err != nil {
		return nil, err
	}
	if !absent && (days < 0 || days > bkMaxRetentionDays) {
		return nil, bkInvalid("RetainRecordInDays must be between 0 and 36500")
	}
	now := bkNow()
	h["Status"] = "CANCELED"
	h["CancelDescription"] = in["CancelDescription"]
	h["CancellationDate"] = now
	h["RetainRecordUntil"] = now + float64(days*bkDaySeconds)
	return nil, nil
}

func (st *backupStore) listLegalHolds(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	items := []map[string]any{}
	for _, h := range bkSorted(st.legalHolds, bkSelf, "CreationDate") {
		items = append(items, bkShape(h, "LegalHold"))
	}
	return bkPage(items, in, "LegalHolds")
}

func (st *backupStore) listRecoveryPointsByLegalHold(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	h, err := st.legalHold(bkStr(in, "LegalHoldId"))
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, rp := range bkSorted(st.recoveryPoints, func(rp *bkRecoveryPoint) map[string]any { return rp.rec }, "CreationDate") {
		if bkHoldCovers(h, rp) {
			items = append(items, bkShape(rp.rec, "RecoveryPointMember"))
		}
	}
	return bkPage(items, in, "RecoveryPoints")
}

// --- backup access points ---

func (st *backupStore) createAccessPoint(s *Server, in map[string]any) (any, *bedrockAgentError) {
	if err := bkRequire(in, "Name", "RecoveryPointArn"); err != nil {
		return nil, err
	}
	name := bkStr(in, "Name")
	if !bkAccessPointNamePattern.MatchString(name) || strings.HasSuffix(name, "-s3alias") {
		return nil, bkInvalid("Name must be 3-50 lowercase letters, numbers or hyphens")
	}
	if p := bkStr(in, "AccessPointPolicy"); p != "" && (len(p) > 200000 || !json.Valid([]byte(p))) {
		return nil, bkInvalid("AccessPointPolicy must be a JSON policy document")
	}
	if len(bkMap(in, "AccessPointMetadata")) > 50 {
		return nil, bkInvalid("AccessPointMetadata can have at most 50 entries")
	}
	rp := st.recoveryPoints[bkStr(in, "RecoveryPointArn")]
	if rp == nil {
		return nil, bkNotFound("Recovery point", bkStr(in, "RecoveryPointArn"))
	}
	if rp.rec["Status"] != "COMPLETED" {
		return nil, bkErr("InvalidResourceStateException", "the recovery point is not completed")
	}
	for _, ap := range st.accessPoints {
		if ap["Name"] == name {
			return nil, bkExists("Backup access point", name)
		}
	}
	if len(st.accessPoints) >= backupMaxItems {
		return nil, bkErr("LimitExceededException", "too many backup access points")
	}
	arn := bkARN("backup-access-point", name)
	metadata := bkMap(in, "AccessPointMetadata")
	if metadata == nil {
		metadata = map[string]any{}
	}
	ap := map[string]any{
		"AccessPointArn": arn, "AccessPointMetadata": metadata, "BackupVaultArn": rp.rec["BackupVaultArn"],
		"BackupVaultName": rp.rec["BackupVaultName"], "CreationTime": bkNow(), "Name": name,
		"RecoveryPointArn": rp.rec["RecoveryPointArn"], "ResourceArn": rp.rec["ResourceArn"], "ResourceType": rp.rec["ResourceType"],
		"Status": "AVAILABLE",
	}
	if err := st.setTags(arn, bkMap(in, "Tags")); err != nil {
		return nil, err
	}
	st.accessPoints[arn] = ap
	return ap, nil
}

func (st *backupStore) accessPoint(arn string) (map[string]any, *bedrockAgentError) {
	ap := st.accessPoints[arn]
	if ap == nil {
		return nil, bkNotFound("Backup access point", arn)
	}
	return ap, nil
}

func (st *backupStore) describeAccessPoint(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	return st.accessPoint(bkStr(in, "AccessPointArn"))
}

func (st *backupStore) deleteAccessPoint(_ *Server, in map[string]any) (any, *bedrockAgentError) {
	arn := bkStr(in, "AccessPointArn")
	if _, err := st.accessPoint(arn); err != nil {
		return nil, err
	}
	delete(st.accessPoints, arn)
	delete(st.tags, arn)
	return nil, nil
}

// listAccessPoints lists access points, optionally only those whose member
// key equals the same input member.
func bkListAccessPoints(key string) bkHandler {
	return func(st *backupStore, _ *Server, in map[string]any) (any, *bedrockAgentError) {
		n, absent, err := bkInt(in, "MaxResults")
		if err != nil {
			return nil, err
		}
		if !absent && (n < 1 || n > 100) {
			return nil, bkInvalid("MaxResults must be between 1 and 100")
		}
		items := []map[string]any{}
		for _, ap := range bkSorted(st.accessPoints, bkSelf, "CreationTime") {
			if key != "" && ap[key] != in[key] {
				continue
			}
			items = append(items, bkShape(ap, "ListAccessPointsMember"))
		}
		return bkPage(items, in, "BackupAccessPoints")
	}
}
