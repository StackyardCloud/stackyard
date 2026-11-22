package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

const backupTestRole = "arn:aws:iam::123456789012:role/backup"

func TestBackupStage1VaultsPoliciesNotificationsAndLocks(t *testing.T) {
	_, ts := newBackupTestServer(t)
	out := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/main", map[string]any{"BackupVaultTags": map[string]any{"team": "a"}, "CreatorRequestId": "req-1"})
	arn, _ := out["BackupVaultArn"].(string)
	if out["BackupVaultName"] != "main" || arn != "arn:aws:backup:us-east-1:123456789012:backup-vault:main" {
		t.Fatalf("create vault: %v", out)
	}
	// Same CreatorRequestId is an idempotent retry; another request is a conflict.
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/main", map[string]any{"CreatorRequestId": "req-1"})
	backupFail(t, ts, "AlreadyExistsException", http.MethodPut, "/backup-vaults/main", nil)
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-vaults/x", nil)
	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, "/backup-vaults/missing", nil)

	desc := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/main", nil)
	if desc["VaultType"] != "BACKUP_VAULT" || desc["Locked"] != false || desc["NumberOfRecoveryPoints"] != float64(0) {
		t.Fatalf("describe vault: %v", desc)
	}
	tags := backupWant(t, ts, http.StatusOK, http.MethodGet, "/tags/"+url.PathEscape(arn)+"/", nil)
	if tags["Tags"].(map[string]any)["team"] != "a" {
		t.Fatalf("vault tags: %v", tags)
	}

	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, "/backup-vaults/main/access-policy", nil)
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-vaults/main/access-policy", map[string]any{"Policy": "not json"})
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/main/access-policy", map[string]any{"Policy": `{"Version":"2012-10-17","Statement":[]}`})
	if p := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/main/access-policy", nil); p["Policy"] == nil {
		t.Fatalf("policy: %v", p)
	}

	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-vaults/main/notification-configuration", map[string]any{"SNSTopicArn": "arn:aws:sns:us-east-1:123456789012:t", "BackupVaultEvents": []string{"NOPE"}})
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/main/notification-configuration", map[string]any{"SNSTopicArn": "arn:aws:sns:us-east-1:123456789012:t", "BackupVaultEvents": []string{"BACKUP_JOB_COMPLETED"}})
	if n := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/main/notification-configuration", nil); n["SNSTopicArn"] == nil {
		t.Fatalf("notifications: %v", n)
	}

	// Governance lock (no ChangeableForDays) can be removed; compliance with
	// a grace period can be changed until the lock date.
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup-vaults/main/vault-lock", map[string]any{"MinRetentionDays": 10, "MaxRetentionDays": 5})
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/main/vault-lock", map[string]any{"MinRetentionDays": 1, "MaxRetentionDays": 30, "ChangeableForDays": 3})
	locked := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/main", nil)
	if locked["Locked"] != true || locked["LockDate"] == nil || locked["MaxRetentionDays"] != float64(30) {
		t.Fatalf("locked vault: %v", locked)
	}
	backupWant(t, ts, http.StatusOK, http.MethodDelete, "/backup-vaults/main/vault-lock", nil)

	// Air-gapped vaults start CREATING, are always locked and back restore-access vaults.
	lag := backupWant(t, ts, http.StatusOK, http.MethodPut, "/logically-air-gapped-backup-vaults/lag", map[string]any{"MinRetentionDays": 7, "MaxRetentionDays": 30})
	if lag["VaultState"] != "CREATING" {
		t.Fatalf("air-gapped vault: %v", lag)
	}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/logically-air-gapped-backup-vaults/lag2", map[string]any{"MinRetentionDays": 1, "MaxRetentionDays": 30})
	if d := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/lag", nil); d["VaultState"] != "AVAILABLE" || d["Locked"] != true {
		t.Fatalf("air-gapped describe: %v", d)
	}
	backupFail(t, ts, "InvalidRequestException", http.MethodPut, "/backup-vaults/lag/vault-lock", map[string]any{"MinRetentionDays": 1})
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/restore-access-backup-vaults", map[string]any{"SourceBackupVaultArn": arn})
	ra := backupWant(t, ts, http.StatusOK, http.MethodPut, "/restore-access-backup-vaults", map[string]any{"SourceBackupVaultArn": lag["BackupVaultArn"], "BackupVaultName": "ra"})
	list := backupWant(t, ts, http.StatusOK, http.MethodGet, "/logically-air-gapped-backup-vaults/lag/restore-access-backup-vaults/", nil)
	if len(list["RestoreAccessBackupVaults"].([]any)) != 1 {
		t.Fatalf("restore access list: %v", list)
	}
	backupFail(t, ts, "InvalidRequestException", http.MethodDelete, "/backup-vaults/lag", nil)
	backupWant(t, ts, http.StatusOK, http.MethodDelete, "/logically-air-gapped-backup-vaults/lag/restore-access-backup-vaults/"+url.PathEscape(ra["RestoreAccessBackupVaultArn"].(string)), nil)
	backupFail(t, ts, "ResourceNotFoundException", http.MethodGet, "/backup-vaults/ra", nil)

	vaults := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/?vaultType=LOGICALLY_AIR_GAPPED_BACKUP_VAULT", nil)
	if l := vaults["BackupVaultList"].([]any); len(l) != 1 || l[0].(map[string]any)["BackupVaultName"] != "lag" {
		t.Fatalf("filtered vault list: %v", vaults)
	}
	page := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-vaults/?maxResults=1", nil)
	if page["NextToken"] == nil || len(page["BackupVaultList"].([]any)) != 1 {
		t.Fatalf("paged vault list: %v", page)
	}
	backupWant(t, ts, http.StatusOK, http.MethodDelete, "/backup-vaults/main", nil)
}

func TestBackupStage1PlansSelectionsAndTemplates(t *testing.T) {
	_, ts := newBackupTestServer(t)
	plan := map[string]any{"BackupPlanName": "daily", "Rules": []any{map[string]any{
		"RuleName": "r1", "TargetBackupVaultName": "Default", "Lifecycle": map[string]any{"MoveToColdStorageAfterDays": 30, "DeleteAfterDays": 120},
	}}}
	created := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup/plans/", map[string]any{"BackupPlan": plan, "BackupPlanTags": map[string]any{"k": "v"}})
	id, _ := created["BackupPlanId"].(string)
	v1, _ := created["VersionId"].(string)
	if id == "" || v1 == "" {
		t.Fatalf("create plan: %v", created)
	}
	backupFail(t, ts, "AlreadyExistsException", http.MethodPut, "/backup/plans/", map[string]any{"BackupPlan": plan})
	bad := map[string]any{"BackupPlanName": "bad", "Rules": []any{map[string]any{"RuleName": "r", "TargetBackupVaultName": "v", "Lifecycle": map[string]any{"MoveToColdStorageAfterDays": 30, "DeleteAfterDays": 60}}}}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup/plans/", map[string]any{"BackupPlan": bad})
	backupFail(t, ts, "MissingParameterValueException", http.MethodPut, "/backup/plans/", map[string]any{})

	got := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/"+id+"/", nil)
	rule := got["BackupPlan"].(map[string]any)["Rules"].([]any)[0].(map[string]any)
	ruleID := rule["RuleId"]
	if ruleID == nil || rule["ScheduleExpression"] != "cron(0 5 ? * * *)" || rule["StartWindowMinutes"] != float64(480) {
		t.Fatalf("plan defaults: %v", rule)
	}

	plan["Rules"] = append(plan["Rules"].([]any), map[string]any{"RuleName": "r2", "TargetBackupVaultName": "Default", "ScheduleExpression": "rate(12 hours)"})
	updated := backupWant(t, ts, http.StatusOK, http.MethodPost, "/backup/plans/"+id, map[string]any{"BackupPlan": plan})
	if updated["VersionId"] == v1 {
		t.Fatalf("update must create a version: %v", updated)
	}
	got = backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/"+id+"/", nil)
	if got["BackupPlan"].(map[string]any)["Rules"].([]any)[0].(map[string]any)["RuleId"] != ruleID {
		t.Fatalf("a rule that keeps its name keeps its RuleId")
	}
	old := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/"+id+"/?versionId="+v1, nil)
	if len(old["BackupPlan"].(map[string]any)["Rules"].([]any)) != 1 {
		t.Fatalf("old version: %v", old)
	}
	versions := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/"+id+"/versions/", nil)
	if len(versions["BackupPlanVersionsList"].([]any)) != 2 {
		t.Fatalf("versions: %v", versions)
	}

	exported := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/"+id+"/toTemplate/", nil)
	parsed := backupWant(t, ts, http.StatusOK, http.MethodPost, "/backup/template/json/toPlan", map[string]any{"BackupPlanTemplateJson": exported["BackupPlanTemplateJson"]})
	if parsed["BackupPlan"].(map[string]any)["BackupPlanName"] != "daily" {
		t.Fatalf("template round trip: %v", parsed)
	}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPost, "/backup/template/json/toPlan", map[string]any{"BackupPlanTemplateJson": "{"})
	templates := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/template/plans", nil)
	tid := templates["BackupPlanTemplatesList"].([]any)[0].(map[string]any)["BackupPlanTemplateId"].(string)
	if doc := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/template/plans/"+tid+"/toPlan", nil); doc["BackupPlanDocument"] == nil {
		t.Fatalf("template document: %v", doc)
	}

	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup/plans/"+id+"/selections/", map[string]any{"BackupSelection": map[string]any{"SelectionName": "s", "IamRoleArn": backupTestRole}})
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/backup/plans/"+id+"/selections/", map[string]any{"BackupSelection": map[string]any{"SelectionName": "s", "IamRoleArn": "role", "Resources": []string{"*"}}})
	sel := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup/plans/"+id+"/selections/", map[string]any{"BackupSelection": map[string]any{"SelectionName": "all", "IamRoleArn": backupTestRole, "Resources": []string{"*"}}})
	sid := sel["SelectionId"].(string)
	if s := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/"+id+"/selections/"+sid, nil); s["BackupSelection"].(map[string]any)["SelectionName"] != "all" {
		t.Fatalf("get selection: %v", s)
	}
	backupFail(t, ts, "InvalidRequestException", http.MethodDelete, "/backup/plans/"+id, nil)
	backupWant(t, ts, http.StatusOK, http.MethodDelete, "/backup/plans/"+id+"/selections/"+sid, nil)
	deleted := backupWant(t, ts, http.StatusOK, http.MethodDelete, "/backup/plans/"+id, nil)
	if deleted["DeletionDate"] == nil {
		t.Fatalf("delete plan: %v", deleted)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/", nil); len(l["BackupPlansList"].([]any)) != 0 {
		t.Fatalf("deleted plans are hidden: %v", l)
	}
	if l := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup/plans/?includeDeleted=true", nil); len(l["BackupPlansList"].([]any)) != 1 {
		t.Fatalf("includeDeleted: %v", l)
	}
}

func TestBackupStage1SettingsAndTags(t *testing.T) {
	_, ts := newBackupTestServer(t)
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/global-settings", map[string]any{"GlobalSettings": map[string]any{"isCrossAccountBackupEnabled": "true"}})
	g := backupWant(t, ts, http.StatusOK, http.MethodGet, "/global-settings", nil)
	if g["GlobalSettings"].(map[string]any)["isCrossAccountBackupEnabled"] != "true" || g["LastUpdateTime"] == nil {
		t.Fatalf("global settings: %v", g)
	}
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPut, "/global-settings", map[string]any{"GlobalSettings": map[string]any{"nope": "true"}})
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/account-settings", map[string]any{"ResourceTypeOptInPreference": map[string]any{"EBS": false}})
	r := backupWant(t, ts, http.StatusOK, http.MethodGet, "/account-settings", nil)
	if r["ResourceTypeOptInPreference"].(map[string]any)["EBS"] != false {
		t.Fatalf("region settings: %v", r)
	}
	// Opted-out types cannot be backed up.
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/v1", nil)
	backupFail(t, ts, "InvalidRequestException", http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "v1", "IamRoleArn": backupTestRole, "ResourceArn": "arn:aws:ec2:us-east-1:123456789012:volume/vol-1"})
	if types := backupWant(t, ts, http.StatusOK, http.MethodGet, "/supported-resource-types", nil); len(types["ResourceTypes"].([]any)) == 0 {
		t.Fatalf("supported types: %v", types)
	}

	arn := "arn:aws:backup:us-east-1:123456789012:backup-vault:v1"
	backupFail(t, ts, "ResourceNotFoundException", http.MethodPost, "/tags/"+url.PathEscape(arn+"x"), map[string]any{"Tags": map[string]any{"a": "b"}})
	backupFail(t, ts, "InvalidParameterValueException", http.MethodPost, "/tags/"+url.PathEscape(arn), map[string]any{"Tags": map[string]any{"aws:x": "b"}})
	backupWant(t, ts, http.StatusOK, http.MethodPost, "/tags/"+url.PathEscape(arn), map[string]any{"Tags": map[string]any{"a": "b", "c": "d"}})
	backupWant(t, ts, http.StatusOK, http.MethodPost, "/untag/"+url.PathEscape(arn), map[string]any{"TagKeyList": []string{"a"}})
	tags := backupWant(t, ts, http.StatusOK, http.MethodGet, "/tags/"+url.PathEscape(arn)+"/", nil)
	raw, _ := json.Marshal(tags["Tags"])
	if string(raw) != `{"c":"d"}` {
		t.Fatalf("tags after untag: %s", raw)
	}
}
