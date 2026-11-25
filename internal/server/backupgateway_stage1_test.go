package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestBackupGatewayStage1GatewaysHypervisorsAndBackup(t *testing.T) {
	_, ts := newBudTestServer(t)
	gw := budOKAPI(t, ts, bgwAPI, "CreateGateway", `{"ActivationKey":"ABCDE-12345","GatewayDisplayName":"gw1","GatewayType":"BACKUP_VM","Tags":[{"Key":"team","Value":"ops"}]}`)["GatewayArn"].(string)
	hv := budOKAPI(t, ts, bgwAPI, "ImportHypervisorConfiguration", `{"Name":"vcenter","Host":"vcenter.example.com","Username":"admin","Password":"s3cret-Pass"}`)["HypervisorArn"].(string)
	if code := budErrAPI(t, ts, bgwAPI, "ImportHypervisorConfiguration", `{"Name":"dup","Host":"VCENTER.example.com"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("duplicate host: %s", code)
	}
	if status, out := budCallAPI(t, ts, bgwAPI, "ImportHypervisorConfiguration", `{"Name":"x","Host":"other.example.com","Password":"p"}`); status != http.StatusBadRequest || out["ErrorCode"] != "ValidationException" {
		t.Fatalf("password without username: %d %v", status, out)
	}

	// PENDING -> ONLINE, then the sync runs CREATED -> RUNNING -> SUCCEEDED.
	var states []string
	for i := 0; i < 3; i++ {
		h := budOKAPI(t, ts, bgwAPI, "GetHypervisor", `{"HypervisorArn":"`+hv+`"}`)["Hypervisor"].(map[string]any)
		states = append(states, h["State"].(string)+"/"+h["LatestMetadataSyncStatus"].(string))
		raw := strings.ToLower(bgwJSON(t, h))
		if strings.Contains(raw, "s3cret") || strings.Contains(raw, "password") || strings.Contains(raw, "admin") {
			t.Fatalf("credentials leaked: %s", raw)
		}
	}
	if strings.Join(states, ",") != "ONLINE/CREATED,ONLINE/RUNNING,ONLINE/SUCCEEDED" {
		t.Fatalf("hypervisor states: %v", states)
	}
	vms := budOKAPI(t, ts, bgwAPI, "ListVirtualMachines", `{"HypervisorArn":"`+hv+`"}`)["VirtualMachines"].([]any)
	if len(vms) != bgwVMsPerHost {
		t.Fatalf("vms: %v", vms)
	}
	vm := vms[0].(map[string]any)
	vmARN := vm["ResourceArn"].(string)
	if vm["Name"] != "vcenter-vm-1" || vm["LastBackupDate"] != nil {
		t.Fatalf("vm: %v", vm)
	}

	// Associate, then the hypervisor can't be deleted.
	budOKAPI(t, ts, bgwAPI, "AssociateGatewayToServer", `{"GatewayArn":"`+gw+`","ServerArn":"`+hv+`"}`)
	if code := budErrAPI(t, ts, bgwAPI, "AssociateGatewayToServer", `{"GatewayArn":"`+gw+`","ServerArn":"`+hv+`"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("double associate: %s", code)
	}
	if code := budErrAPI(t, ts, bgwAPI, "DeleteHypervisor", `{"HypervisorArn":"`+hv+`"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("delete associated hypervisor: %s", code)
	}
	g := budOKAPI(t, ts, bgwAPI, "GetGateway", `{"GatewayArn":"`+gw+`"}`)["Gateway"].(map[string]any)
	if g["HypervisorId"] == nil || g["SoftwareVersion"] != bgwSoftware || g["NextUpdateAvailabilityTime"] == nil {
		t.Fatalf("gateway: %v", g)
	}
	budOKAPI(t, ts, bgwAPI, "UpdateGatewaySoftwareNow", `{"GatewayArn":"`+gw+`"}`)
	if g := budOKAPI(t, ts, bgwAPI, "GetGateway", `{"GatewayArn":"`+gw+`"}`)["Gateway"].(map[string]any); g["SoftwareVersion"] != bgwNextSoftware || g["NextUpdateAvailabilityTime"] != nil {
		t.Fatalf("gateway after update: %v", g)
	}

	// Maintenance and bandwidth checks.
	if code := budErrAPI(t, ts, bgwAPI, "PutMaintenanceStartTime", `{"GatewayArn":"`+gw+`","HourOfDay":1,"MinuteOfHour":0,"DayOfWeek":1,"DayOfMonth":2}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("weekly and monthly: %s", code)
	}
	budOKAPI(t, ts, bgwAPI, "PutMaintenanceStartTime", `{"GatewayArn":"`+gw+`","HourOfDay":1,"MinuteOfHour":30,"DayOfMonth":2}`)
	overlap := `{"GatewayArn":"` + gw + `","BandwidthRateLimitIntervals":[{"StartHourOfDay":1,"StartMinuteOfHour":0,"EndHourOfDay":3,"EndMinuteOfHour":0,"DaysOfWeek":[1]},{"StartHourOfDay":2,"StartMinuteOfHour":0,"EndHourOfDay":4,"EndMinuteOfHour":0,"DaysOfWeek":[1,2]}]}`
	if code := budErrAPI(t, ts, bgwAPI, "PutBandwidthRateLimitSchedule", overlap, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("overlapping intervals: %s", code)
	}
	budOKAPI(t, ts, bgwAPI, "PutBandwidthRateLimitSchedule", `{"GatewayArn":"`+gw+`","BandwidthRateLimitIntervals":[{"AverageUploadRateLimitInBitsPerSec":102400,"StartHourOfDay":1,"StartMinuteOfHour":0,"EndHourOfDay":3,"EndMinuteOfHour":0,"DaysOfWeek":[1,2]}]}`)
	if l := budOKAPI(t, ts, bgwAPI, "GetBandwidthRateLimitSchedule", `{"GatewayArn":"`+gw+`"}`)["BandwidthRateLimitIntervals"].([]any); len(l) != 1 {
		t.Fatalf("bandwidth: %v", l)
	}

	// Tag mappings show up as the VM's VMware tags.
	budOKAPI(t, ts, bgwAPI, "PutHypervisorPropertyMappings", `{"HypervisorArn":"`+hv+`","IamRoleArn":"arn:aws:iam::123456789012:role/bgw","VmwareToAwsTagMappings":[{"VmwareCategory":"tier","VmwareTagName":"gold","AwsTagKey":"tier","AwsTagValue":"gold"}]}`)
	got := budOKAPI(t, ts, bgwAPI, "GetVirtualMachine", `{"ResourceArn":"`+vmARN+`"}`)["VirtualMachine"].(map[string]any)
	if tags := got["VmwareTags"].([]any); len(tags) != 1 {
		t.Fatalf("vmware tags: %v", got)
	}

	// AWS Backup sees the VM: a backup job takes its name, and the
	// completed recovery point becomes the VM's LastBackupDate.
	backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-vaults/vmvault", nil)
	backupFail(t, ts, "ResourceNotFoundException", http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "vmvault", "IamRoleArn": backupTestRole, "ResourceArn": "arn:aws:backup-gateway:us-east-1:123456789012:vm/vm-00000000000000000"})
	job := backupWant(t, ts, http.StatusOK, http.MethodPut, "/backup-jobs", map[string]any{"BackupVaultName": "vmvault", "IamRoleArn": backupTestRole, "ResourceArn": vmARN})
	jobID := job["BackupJobId"].(string)
	backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-jobs/"+jobID, nil)
	if d := backupWant(t, ts, http.StatusOK, http.MethodGet, "/backup-jobs/"+jobID, nil); d["State"] != "COMPLETED" || d["ResourceName"] != "vcenter-vm-1" || d["ResourceType"] != "VirtualMachine" {
		t.Fatalf("vm backup job: %v", d)
	}
	if got := budOKAPI(t, ts, bgwAPI, "GetVirtualMachine", `{"ResourceArn":"`+vmARN+`"}`)["VirtualMachine"].(map[string]any); got["LastBackupDate"] == nil {
		t.Fatalf("last backup date: %v", got)
	}

	// Tags, disassociate, delete.
	budOKAPI(t, ts, bgwAPI, "TagResource", `{"ResourceARN":"`+hv+`","Tags":[{"Key":"env","Value":"lab"}]}`)
	budOKAPI(t, ts, bgwAPI, "UntagResource", `{"ResourceARN":"`+gw+`","TagKeys":["team"]}`)
	if tags := budOKAPI(t, ts, bgwAPI, "ListTagsForResource", `{"ResourceArn":"`+hv+`"}`)["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	budOKAPI(t, ts, bgwAPI, "DisassociateGatewayFromServer", `{"GatewayArn":"`+gw+`"}`)
	if code := budErrAPI(t, ts, bgwAPI, "DisassociateGatewayFromServer", `{"GatewayArn":"`+gw+`"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("double disassociate: %s", code)
	}
	budOKAPI(t, ts, bgwAPI, "DeleteHypervisor", `{"HypervisorArn":"`+hv+`"}`)
	if n := len(budOKAPI(t, ts, bgwAPI, "ListVirtualMachines", `{}`)["VirtualMachines"].([]any)); n != 0 {
		t.Fatalf("vms after delete: %d", n)
	}
	budOKAPI(t, ts, bgwAPI, "DeleteGateway", `{"GatewayArn":"`+gw+`"}`)
	if code := budErrAPI(t, ts, bgwAPI, "GetGateway", `{"GatewayArn":"`+gw+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted gateway: %s", code)
	}
}

func bgwJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
