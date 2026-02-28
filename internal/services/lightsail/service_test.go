package lightsail

import (
	"testing"
	"time"
)

func TestServiceInstanceLifecycleAndOperations(t *testing.T) {
	svc := NewService()

	createOps, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"web-1"}, map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create instances: %v", err)
	}
	if len(createOps) != 1 {
		t.Fatalf("expected 1 create op, got %d", len(createOps))
	}

	instance, ok := svc.GetInstance("web-1")
	if !ok {
		t.Fatalf("expected instance to exist")
	}
	if instance.StateName != "running" {
		t.Fatalf("expected running state, got %q", instance.StateName)
	}

	if _, _, ok := svc.GetInstanceState("web-1"); !ok {
		t.Fatalf("expected instance state")
	}

	if _, err := svc.StopInstance("web-1"); err != nil {
		t.Fatalf("stop instance: %v", err)
	}
	if _, err := svc.StartInstance("web-1"); err != nil {
		t.Fatalf("start instance: %v", err)
	}
	if _, err := svc.RebootInstance("web-1"); err != nil {
		t.Fatalf("reboot instance: %v", err)
	}

	if _, err := svc.CreateInstanceSnapshot("web-1", "web-1-snap", map[string]string{"team": "infra"}); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if _, ok := svc.GetInstanceSnapshot("web-1-snap"); !ok {
		t.Fatalf("expected snapshot to exist")
	}
	if got := len(svc.GetInstanceSnapshots()); got != 1 {
		t.Fatalf("expected 1 snapshot, got %d", got)
	}

	if _, err := svc.AllocateStaticIP("web-1-ip"); err != nil {
		t.Fatalf("allocate static ip: %v", err)
	}
	if _, ok := svc.GetStaticIP("web-1-ip"); !ok {
		t.Fatalf("expected static ip to exist")
	}
	if _, err := svc.AttachStaticIP("web-1-ip", "web-1"); err != nil {
		t.Fatalf("attach static ip: %v", err)
	}
	if _, err := svc.DetachStaticIP("web-1-ip"); err != nil {
		t.Fatalf("detach static ip: %v", err)
	}
	if _, err := svc.ReleaseStaticIP("web-1-ip"); err != nil {
		t.Fatalf("release static ip: %v", err)
	}

	if _, err := svc.TagResource("web-1", map[string]string{"role": "frontend"}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	if _, err := svc.UntagResource("web-1", []string{"role"}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}

	if _, err := svc.DeleteInstanceSnapshot("web-1-snap"); err != nil {
		t.Fatalf("delete snapshot: %v", err)
	}
	if _, err := svc.DeleteInstance("web-1"); err != nil {
		t.Fatalf("delete instance: %v", err)
	}

	allOps := svc.GetOperations()
	if len(allOps) == 0 {
		t.Fatalf("expected operation history")
	}
	if allOps[0].ID == "" {
		t.Fatalf("expected operation id")
	}
	if _, ok := svc.GetOperation(allOps[0].ID); !ok {
		t.Fatalf("expected operation lookup by id")
	}
	resourceOps := svc.GetOperationsForResource("web-1")
	if len(resourceOps) == 0 {
		t.Fatalf("expected resource operations")
	}
}

func TestServiceRegionsAndArnHelpers(t *testing.T) {
	svc := NewService()

	regions := svc.GetRegions(true, true)
	if len(regions) == 0 {
		t.Fatalf("expected regions")
	}

	if got := svc.ResourceNameFromARN("arn:aws:lightsail:us-east-1:123456789012:Instance/example"); got != "example" {
		t.Fatalf("unexpected resource name from arn: %q", got)
	}
}

func TestServiceStage5InstanceAccessAndNetworking(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage5-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := svc.CreateInstanceSnapshot("stage5-instance", "stage5-snapshot", nil); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	if _, err := svc.OpenInstancePublicPorts("stage5-instance", PortInfo{
		FromPort: 443,
		ToPort:   443,
		Protocol: "tcp",
		Cidrs:    []string{"0.0.0.0/0"},
	}); err != nil {
		t.Fatalf("open instance public ports: %v", err)
	}

	portStates, err := svc.GetInstancePortStates("stage5-instance")
	if err != nil {
		t.Fatalf("get instance port states: %v", err)
	}
	if len(portStates) < 2 {
		t.Fatalf("expected at least two port states after opening an additional port")
	}

	if _, err := svc.CloseInstancePublicPorts("stage5-instance", PortInfo{
		FromPort: 443,
		ToPort:   443,
		Protocol: "tcp",
	}); err != nil {
		t.Fatalf("close instance public ports: %v", err)
	}

	if _, err := svc.PutInstancePublicPorts("stage5-instance", []PortInfo{{
		FromPort: 80,
		ToPort:   80,
		Protocol: "tcp",
		Cidrs:    []string{"0.0.0.0/0"},
	}}); err != nil {
		t.Fatalf("put instance public ports: %v", err)
	}

	portStates, err = svc.GetInstancePortStates("stage5-instance")
	if err != nil {
		t.Fatalf("get instance port states after put: %v", err)
	}
	if len(portStates) != 1 || portStates[0].FromPort != 80 {
		t.Fatalf("expected only port 80 to remain after put: %+v", portStates)
	}

	accessDetails, err := svc.GetInstanceAccessDetails("stage5-instance", "ssh")
	if err != nil {
		t.Fatalf("get instance access details: %v", err)
	}
	if accessDetails.Protocol != "ssh" || accessDetails.PrivateKey == "" || accessDetails.Username == "" {
		t.Fatalf("unexpected ssh access details: %+v", accessDetails)
	}

	hopLimit := int32(3)
	if _, err := svc.UpdateInstanceMetadataOptions("stage5-instance", "enabled", "disabled", "required", &hopLimit); err != nil {
		t.Fatalf("update instance metadata options: %v", err)
	}
	instance, ok := svc.GetInstance("stage5-instance")
	if !ok {
		t.Fatalf("expected instance to exist")
	}
	if instance.MetadataOptions.HttpTokens != "required" || instance.MetadataOptions.HttpPutResponseHopLimit != 3 {
		t.Fatalf("metadata options were not updated: %+v", instance.MetadataOptions)
	}

	if _, err := svc.DeleteKnownHostKeys("stage5-instance"); err != nil {
		t.Fatalf("delete known host keys: %v", err)
	}
	instance, ok = svc.GetInstance("stage5-instance")
	if !ok {
		t.Fatalf("expected instance to exist after deleting known host keys")
	}
	if len(instance.HostKeys) == 0 {
		t.Fatalf("expected regenerated host keys")
	}

	ops, err := svc.CreateInstancesFromSnapshot(
		"us-east-1a",
		"micro_2_0",
		[]string{"stage5-from-snapshot"},
		"stage5-snapshot",
		"",
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("create instances from snapshot: %v", err)
	}
	if len(ops) != 1 || ops[0].OperationType != "CreateInstancesFromSnapshot" {
		t.Fatalf("unexpected create from snapshot operations: %+v", ops)
	}
}

func TestServiceStage6KeyPairs(t *testing.T) {
	svc := NewService()

	keyPair, op, err := svc.CreateKeyPair("stage6-key", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create key pair: %v", err)
	}
	if keyPair.Name != "stage6-key" || op.OperationType != "CreateKeyPair" {
		t.Fatalf("unexpected create key pair result: keyPair=%+v operation=%+v", keyPair, op)
	}
	if keyPair.PublicKeyBase64 == "" || keyPair.PrivateKeyBase64 == "" {
		t.Fatalf("expected key material for created key pair")
	}

	got, ok := svc.GetKeyPair("stage6-key")
	if !ok {
		t.Fatalf("expected key pair to exist")
	}
	if got.Fingerprint == "" {
		t.Fatalf("expected key pair fingerprint")
	}

	keyPairs := svc.GetKeyPairs(false)
	if len(keyPairs) != 1 {
		t.Fatalf("expected one non-default key pair, got %d", len(keyPairs))
	}

	if _, err := svc.ImportKeyPair("stage6-imported", "c3NoLXJzYSBTVEFDS1lBUkQtaW1wb3J0ZWQ="); err != nil {
		t.Fatalf("import key pair: %v", err)
	}
	keyPairs = svc.GetKeyPairs(false)
	if len(keyPairs) != 2 {
		t.Fatalf("expected two non-default key pairs, got %d", len(keyPairs))
	}

	createdAt, privateKeyBase64, publicKeyBase64, err := svc.DownloadDefaultKeyPair()
	if err != nil {
		t.Fatalf("download default key pair: %v", err)
	}
	if createdAt.IsZero() || privateKeyBase64 == "" || publicKeyBase64 == "" {
		t.Fatalf("expected default key pair material")
	}

	withDefault := svc.GetKeyPairs(true)
	if len(withDefault) != 3 {
		t.Fatalf("expected three key pairs including default, got %d", len(withDefault))
	}

	defaultKey, ok := svc.GetKeyPair(DefaultKeyPair)
	if !ok {
		t.Fatalf("expected default key pair to exist")
	}
	if _, err := svc.DeleteKeyPair(DefaultKeyPair, "wrong-fingerprint"); err == nil {
		t.Fatalf("expected delete with wrong fingerprint to fail")
	}
	if _, err := svc.DeleteKeyPair(DefaultKeyPair, defaultKey.Fingerprint); err != nil {
		t.Fatalf("delete default key pair: %v", err)
	}
	if _, ok := svc.GetKeyPair(DefaultKeyPair); ok {
		t.Fatalf("expected default key pair to be deleted")
	}

	if _, err := svc.DeleteKeyPair("stage6-key", ""); err != nil {
		t.Fatalf("delete key pair: %v", err)
	}
	if _, ok := svc.GetKeyPair("stage6-key"); ok {
		t.Fatalf("expected deleted key pair to not exist")
	}
}

func TestServiceStage7DisksCore(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage7-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	createOps, err := svc.CreateDisk("us-east-1a", "stage7-disk", 32, map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create disk: %v", err)
	}
	if len(createOps) != 1 || createOps[0].OperationType != "CreateDisk" {
		t.Fatalf("unexpected create disk operation result: %+v", createOps)
	}

	disk, ok := svc.GetDisk("stage7-disk")
	if !ok {
		t.Fatalf("expected disk to exist")
	}
	if disk.SizeInGb != 32 || disk.State != "available" {
		t.Fatalf("unexpected disk after create: %+v", disk)
	}

	disks := svc.GetDisks()
	if len(disks) != 1 || disks[0].Name != "stage7-disk" {
		t.Fatalf("unexpected disks list: %+v", disks)
	}

	attachOps, err := svc.AttachDisk("stage7-disk", "/dev/xvdf", "stage7-instance", true)
	if err != nil {
		t.Fatalf("attach disk: %v", err)
	}
	if len(attachOps) != 1 || attachOps[0].OperationType != "AttachDisk" {
		t.Fatalf("unexpected attach disk operation result: %+v", attachOps)
	}
	disk, ok = svc.GetDisk("stage7-disk")
	if !ok {
		t.Fatalf("expected disk after attach")
	}
	if !disk.IsAttached || disk.AttachedTo != "stage7-instance" || disk.Path != "/dev/xvdf" || disk.State != "in-use" {
		t.Fatalf("unexpected disk after attach: %+v", disk)
	}

	detachOps, err := svc.DetachDisk("stage7-disk")
	if err != nil {
		t.Fatalf("detach disk: %v", err)
	}
	if len(detachOps) != 1 || detachOps[0].OperationType != "DetachDisk" {
		t.Fatalf("unexpected detach disk operation result: %+v", detachOps)
	}
	disk, ok = svc.GetDisk("stage7-disk")
	if !ok {
		t.Fatalf("expected disk after detach")
	}
	if disk.IsAttached || disk.AttachedTo != "" || disk.Path != "" || disk.State != "available" {
		t.Fatalf("unexpected disk after detach: %+v", disk)
	}

	deleteOps, err := svc.DeleteDisk("stage7-disk")
	if err != nil {
		t.Fatalf("delete disk: %v", err)
	}
	if len(deleteOps) != 1 || deleteOps[0].OperationType != "DeleteDisk" {
		t.Fatalf("unexpected delete disk operation result: %+v", deleteOps)
	}
	if _, ok := svc.GetDisk("stage7-disk"); ok {
		t.Fatalf("expected disk to be deleted")
	}
}

func TestServiceStage8DiskSnapshotsAndMigration(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage8-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := svc.CreateDisk("us-east-1a", "stage8-disk", 64, map[string]string{"env": "test"}); err != nil {
		t.Fatalf("create disk: %v", err)
	}

	createSnapshotOps, err := svc.CreateDiskSnapshot("stage8-disk", "", "stage8-snap", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create disk snapshot: %v", err)
	}
	if len(createSnapshotOps) != 1 || createSnapshotOps[0].OperationType != "CreateDiskSnapshot" {
		t.Fatalf("unexpected create snapshot operations: %+v", createSnapshotOps)
	}

	snapshot, ok := svc.GetDiskSnapshot("stage8-snap")
	if !ok {
		t.Fatalf("expected disk snapshot to exist")
	}
	if snapshot.FromDiskName != "stage8-disk" || snapshot.SizeInGb != 64 || snapshot.State != "completed" {
		t.Fatalf("unexpected disk snapshot: %+v", snapshot)
	}

	snapshots := svc.GetDiskSnapshots()
	if len(snapshots) != 1 || snapshots[0].Name != "stage8-snap" {
		t.Fatalf("unexpected disk snapshots list: %+v", snapshots)
	}

	createFromSnapshotOps, err := svc.CreateDiskFromSnapshot("us-east-1a", "stage8-disk-restored", "stage8-snap", "", 64, nil)
	if err != nil {
		t.Fatalf("create disk from snapshot: %v", err)
	}
	if len(createFromSnapshotOps) != 1 || createFromSnapshotOps[0].OperationType != "CreateDiskFromSnapshot" {
		t.Fatalf("unexpected create disk from snapshot operations: %+v", createFromSnapshotOps)
	}
	restoredDisk, ok := svc.GetDisk("stage8-disk-restored")
	if !ok || restoredDisk.SizeInGb != 64 {
		t.Fatalf("unexpected restored disk: %+v", restoredDisk)
	}

	copyOps, err := svc.CopySnapshot("us-east-1", "stage8-snap-copy", "stage8-snap", "")
	if err != nil {
		t.Fatalf("copy snapshot: %v", err)
	}
	if len(copyOps) != 1 || copyOps[0].OperationType != "CopySnapshot" {
		t.Fatalf("unexpected copy snapshot operations: %+v", copyOps)
	}
	if _, ok := svc.GetDiskSnapshot("stage8-snap-copy"); !ok {
		t.Fatalf("expected copied snapshot to exist")
	}

	exportOps, err := svc.ExportSnapshot("stage8-snap")
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	if len(exportOps) != 1 || exportOps[0].OperationType != "ExportSnapshot" {
		t.Fatalf("unexpected export snapshot operations: %+v", exportOps)
	}
	exportRecords := svc.GetExportSnapshotRecords()
	if len(exportRecords) != 1 {
		t.Fatalf("expected one export snapshot record, got %d", len(exportRecords))
	}
	if exportRecords[0].SourceSnapshotName != "stage8-snap" || exportRecords[0].DestinationService == "" {
		t.Fatalf("unexpected export snapshot record: %+v", exportRecords[0])
	}

	deleteSnapshotOps, err := svc.DeleteDiskSnapshot("stage8-snap")
	if err != nil {
		t.Fatalf("delete disk snapshot: %v", err)
	}
	if len(deleteSnapshotOps) != 1 || deleteSnapshotOps[0].OperationType != "DeleteDiskSnapshot" {
		t.Fatalf("unexpected delete snapshot operations: %+v", deleteSnapshotOps)
	}
	if _, ok := svc.GetDiskSnapshot("stage8-snap"); ok {
		t.Fatalf("expected disk snapshot to be deleted")
	}
}

func TestServiceStage9AlarmsAddOnsAutoSnapshots(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage9-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	putOps, err := svc.PutAlarm(
		"stage9-cpu-high",
		"GreaterThanOrEqualToThreshold",
		"CPUUtilization",
		"stage9-instance",
		1,
		80,
		[]string{"Email"},
		nil,
		nil,
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("put alarm: %v", err)
	}
	if len(putOps) != 1 || putOps[0].OperationType != "PutAlarm" {
		t.Fatalf("unexpected put alarm operations: %+v", putOps)
	}

	alarms := svc.GetAlarms("stage9-cpu-high", "")
	if len(alarms) != 1 {
		t.Fatalf("expected one alarm by name, got %d", len(alarms))
	}
	alarms = svc.GetAlarms("", "stage9-instance")
	if len(alarms) != 1 {
		t.Fatalf("expected one alarm by resource, got %d", len(alarms))
	}

	testOps, err := svc.TestAlarm("stage9-cpu-high", "ALARM")
	if err != nil {
		t.Fatalf("test alarm: %v", err)
	}
	if len(testOps) != 1 || testOps[0].OperationType != "TestAlarm" {
		t.Fatalf("unexpected test alarm operations: %+v", testOps)
	}
	alarms = svc.GetAlarms("stage9-cpu-high", "")
	if len(alarms) != 1 || alarms[0].State != "ALARM" {
		t.Fatalf("expected alarm state ALARM after test: %+v", alarms)
	}

	deleteOps, err := svc.DeleteAlarm("stage9-cpu-high")
	if err != nil {
		t.Fatalf("delete alarm: %v", err)
	}
	if len(deleteOps) != 1 || deleteOps[0].OperationType != "DeleteAlarm" {
		t.Fatalf("unexpected delete alarm operations: %+v", deleteOps)
	}
	if got := len(svc.GetAlarms("stage9-cpu-high", "")); got != 0 {
		t.Fatalf("expected deleted alarm to be absent, got %d", got)
	}

	enableOps, err := svc.EnableAddOn("stage9-instance", "AutoSnapshot", "06:00")
	if err != nil {
		t.Fatalf("enable add-on: %v", err)
	}
	if len(enableOps) != 1 || enableOps[0].OperationType != "EnableAddOn" {
		t.Fatalf("unexpected enable add-on operations: %+v", enableOps)
	}

	autoSnapshots, resourceType, err := svc.GetAutoSnapshots("stage9-instance")
	if err != nil {
		t.Fatalf("get auto snapshots: %v", err)
	}
	if resourceType != "Instance" {
		t.Fatalf("expected resource type Instance, got %s", resourceType)
	}
	if len(autoSnapshots) == 0 {
		t.Fatalf("expected at least one auto snapshot")
	}

	deleteAutoOps, err := svc.DeleteAutoSnapshot("stage9-instance", autoSnapshots[0].Date)
	if err != nil {
		t.Fatalf("delete auto snapshot: %v", err)
	}
	if len(deleteAutoOps) != 1 || deleteAutoOps[0].OperationType != "DeleteAutoSnapshot" {
		t.Fatalf("unexpected delete auto snapshot operations: %+v", deleteAutoOps)
	}

	disableOps, err := svc.DisableAddOn("stage9-instance", "AutoSnapshot")
	if err != nil {
		t.Fatalf("disable add-on: %v", err)
	}
	if len(disableOps) != 1 || disableOps[0].OperationType != "DisableAddOn" {
		t.Fatalf("unexpected disable add-on operations: %+v", disableOps)
	}
}

func TestServiceStage10LoadBalancerCore(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage10-instance-1", "stage10-instance-2"}, nil); err != nil {
		t.Fatalf("create instances: %v", err)
	}

	createOps, err := svc.CreateLoadBalancer(
		"stage10-lb",
		80,
		"",
		"",
		nil,
		"/health",
		"dualstack",
		"TLS-1-2-2018-06",
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("create load balancer: %v", err)
	}
	if len(createOps) != 1 || createOps[0].OperationType != "CreateLoadBalancer" {
		t.Fatalf("unexpected create load balancer operations: %+v", createOps)
	}

	attachOps, err := svc.AttachInstancesToLoadBalancer("stage10-lb", []string{"stage10-instance-1", "stage10-instance-2"})
	if err != nil {
		t.Fatalf("attach instances to load balancer: %v", err)
	}
	if len(attachOps) != 1 || attachOps[0].OperationType != "AttachInstancesToLoadBalancer" {
		t.Fatalf("unexpected attach operations: %+v", attachOps)
	}

	lb, ok := svc.GetLoadBalancer("stage10-lb")
	if !ok {
		t.Fatalf("expected load balancer to exist")
	}
	if lb.Name != "stage10-lb" || lb.InstancePort != 80 || lb.HealthCheckPath != "/health" {
		t.Fatalf("unexpected load balancer: %+v", lb)
	}
	if len(lb.InstanceHealthSummary) != 2 {
		t.Fatalf("expected two instance health summaries, got %d", len(lb.InstanceHealthSummary))
	}

	page, err := svc.GetLoadBalancers("")
	if err != nil {
		t.Fatalf("get load balancers: %v", err)
	}
	if len(page.LoadBalancers) != 1 || page.LoadBalancers[0].Name != "stage10-lb" {
		t.Fatalf("unexpected load balancers page: %+v", page)
	}

	updateOps, err := svc.UpdateLoadBalancerAttribute("stage10-lb", "HttpsRedirectionEnabled", "true")
	if err != nil {
		t.Fatalf("update load balancer attribute: %v", err)
	}
	if len(updateOps) != 1 || updateOps[0].OperationType != "UpdateLoadBalancerAttribute" {
		t.Fatalf("unexpected update operations: %+v", updateOps)
	}

	startTime := time.Now().UTC().Add(-5 * time.Minute)
	endTime := time.Now().UTC()
	metricName, metricData, err := svc.GetLoadBalancerMetricData(LoadBalancerMetricInput{
		LoadBalancerName: "stage10-lb",
		StartTime:        startTime,
		EndTime:          endTime,
		Period:           60,
		MetricName:       "RequestCount",
		Statistics:       []string{"Average", "Sum"},
		Unit:             "Count",
	})
	if err != nil {
		t.Fatalf("get load balancer metric data: %v", err)
	}
	if metricName != "RequestCount" {
		t.Fatalf("expected metric name RequestCount, got %s", metricName)
	}
	if len(metricData) == 0 {
		t.Fatalf("expected metric datapoints")
	}

	detachOps, err := svc.DetachInstancesFromLoadBalancer("stage10-lb", []string{"stage10-instance-2"})
	if err != nil {
		t.Fatalf("detach instances from load balancer: %v", err)
	}
	if len(detachOps) != 1 || detachOps[0].OperationType != "DetachInstancesFromLoadBalancer" {
		t.Fatalf("unexpected detach operations: %+v", detachOps)
	}

	lb, ok = svc.GetLoadBalancer("stage10-lb")
	if !ok {
		t.Fatalf("expected load balancer to exist after detach")
	}
	if len(lb.InstanceHealthSummary) != 1 || lb.InstanceHealthSummary[0].InstanceName != "stage10-instance-1" {
		t.Fatalf("unexpected instance health summary after detach: %+v", lb.InstanceHealthSummary)
	}

	deleteOps, err := svc.DeleteLoadBalancer("stage10-lb")
	if err != nil {
		t.Fatalf("delete load balancer: %v", err)
	}
	if len(deleteOps) != 1 || deleteOps[0].OperationType != "DeleteLoadBalancer" {
		t.Fatalf("unexpected delete operations: %+v", deleteOps)
	}
	if _, ok := svc.GetLoadBalancer("stage10-lb"); ok {
		t.Fatalf("expected load balancer to be deleted")
	}
}

func TestServiceStage11LoadBalancerTLS(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage11-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	createOps, err := svc.CreateLoadBalancerTLSCertificate(
		"stage11-lb",
		"stage11-cert",
		"example.com",
		[]string{"www.example.com"},
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("create load balancer tls certificate: %v", err)
	}
	if len(createOps) != 1 || createOps[0].OperationType != "CreateLoadBalancerTlsCertificate" {
		t.Fatalf("unexpected create certificate operations: %+v", createOps)
	}

	certs := svc.GetLoadBalancerTLSCertificates("stage11-lb")
	if len(certs) != 1 || certs[0].Name != "stage11-cert" {
		t.Fatalf("unexpected tls certificates after create: %+v", certs)
	}
	if certs[0].IsAttached {
		t.Fatalf("expected certificate to start detached")
	}

	attachOps, err := svc.AttachLoadBalancerTLSCertificate("stage11-lb", "stage11-cert")
	if err != nil {
		t.Fatalf("attach load balancer tls certificate: %v", err)
	}
	if len(attachOps) != 1 || attachOps[0].OperationType != "AttachLoadBalancerTlsCertificate" {
		t.Fatalf("unexpected attach certificate operations: %+v", attachOps)
	}
	certs = svc.GetLoadBalancerTLSCertificates("stage11-lb")
	if len(certs) != 1 || !certs[0].IsAttached {
		t.Fatalf("expected certificate to be attached: %+v", certs)
	}

	if _, err := svc.DeleteLoadBalancerTLSCertificate("stage11-lb", "stage11-cert", false); err == nil {
		t.Fatalf("expected deleting attached cert without force to fail")
	}
	deleteOps, err := svc.DeleteLoadBalancerTLSCertificate("stage11-lb", "stage11-cert", true)
	if err != nil {
		t.Fatalf("delete load balancer tls certificate: %v", err)
	}
	if len(deleteOps) != 1 || deleteOps[0].OperationType != "DeleteLoadBalancerTlsCertificate" {
		t.Fatalf("unexpected delete certificate operations: %+v", deleteOps)
	}
	if got := len(svc.GetLoadBalancerTLSCertificates("stage11-lb")); got != 0 {
		t.Fatalf("expected no certificates after delete, got %d", got)
	}

	policies := svc.GetLoadBalancerTLSPolicies()
	if len(policies) == 0 {
		t.Fatalf("expected load balancer tls policies")
	}

	setupOps, err := svc.SetupInstanceHTTPS("LetsEncrypt", []string{"example.com"}, "admin@example.com", "stage11-instance")
	if err != nil {
		t.Fatalf("setup instance https: %v", err)
	}
	if len(setupOps) != 1 || setupOps[0].OperationType != "SetupInstanceHttps" {
		t.Fatalf("unexpected setup instance https operations: %+v", setupOps)
	}
}

func TestServiceStage12Distributions(t *testing.T) {
	svc := NewService()

	distribution, op, err := svc.CreateDistribution(DistributionCreateInput{
		BundleID:             "small_1_0",
		DefaultCacheBehavior: DistributionCacheBehavior{Behavior: "cache"},
		DistributionName:     "stage12-dist",
		Origin: DistributionOrigin{
			Name:           "stage12-origin",
			ProtocolPolicy: "http-only",
			RegionName:     "us-east-1",
		},
		CacheBehaviors: []DistributionCacheBehaviorPerPath{
			{Behavior: "dont-cache", Path: "/api/*"},
		},
		Tags: map[string]string{"env": "test"},
	})
	if err != nil {
		t.Fatalf("create distribution: %v", err)
	}
	if distribution.Name != "stage12-dist" {
		t.Fatalf("unexpected distribution name: %q", distribution.Name)
	}
	if op.OperationType != "CreateDistribution" {
		t.Fatalf("unexpected create operation type: %s", op.OperationType)
	}

	distributions := svc.GetDistributions("")
	if len(distributions) != 1 {
		t.Fatalf("expected one distribution, got %d", len(distributions))
	}
	if got := len(svc.GetDistributions("stage12-dist")); got != 1 {
		t.Fatalf("expected one distribution by name, got %d", got)
	}

	enabled := false
	certificateName := "stage12-cert"
	updateOp, err := svc.UpdateDistribution(DistributionUpdateInput{
		DistributionName: "stage12-dist",
		IsEnabled:        &enabled,
		CertificateName:  &certificateName,
	})
	if err != nil {
		t.Fatalf("update distribution: %v", err)
	}
	if updateOp.OperationType != "UpdateDistribution" {
		t.Fatalf("unexpected update operation type: %s", updateOp.OperationType)
	}

	startTime := time.Now().UTC().Add(-5 * time.Minute)
	endTime := time.Now().UTC()
	metricName, metricData, err := svc.GetDistributionMetricData(DistributionMetricInput{
		DistributionName: "stage12-dist",
		StartTime:        startTime,
		EndTime:          endTime,
		Period:           60,
		MetricName:       "Requests",
		Statistics:       []string{"Average", "Sum"},
		Unit:             "Count",
	})
	if err != nil {
		t.Fatalf("get distribution metric data: %v", err)
	}
	if metricName != "Requests" {
		t.Fatalf("expected metric name Requests, got %s", metricName)
	}
	if len(metricData) == 0 {
		t.Fatalf("expected metric data points")
	}

	reset, resetOp, err := svc.ResetDistributionCache("stage12-dist")
	if err != nil {
		t.Fatalf("reset distribution cache: %v", err)
	}
	if reset.Status == "" {
		t.Fatalf("expected reset status")
	}
	if resetOp.OperationType != "ResetDistributionCache" {
		t.Fatalf("unexpected reset operation type: %s", resetOp.OperationType)
	}

	latestReset, found, err := svc.GetDistributionLatestCacheReset("stage12-dist")
	if err != nil {
		t.Fatalf("get latest cache reset: %v", err)
	}
	if !found || latestReset.CreateTime.IsZero() {
		t.Fatalf("expected latest cache reset")
	}

	bundles := svc.GetDistributionBundles()
	if len(bundles) == 0 {
		t.Fatalf("expected distribution bundles")
	}

	updateBundleOp, err := svc.UpdateDistributionBundle("stage12-dist", "medium_1_0")
	if err != nil {
		t.Fatalf("update distribution bundle: %v", err)
	}
	if updateBundleOp.OperationType != "UpdateDistributionBundle" {
		t.Fatalf("unexpected update bundle operation type: %s", updateBundleOp.OperationType)
	}

	deleteOp, err := svc.DeleteDistribution("stage12-dist")
	if err != nil {
		t.Fatalf("delete distribution: %v", err)
	}
	if deleteOp.OperationType != "DeleteDistribution" {
		t.Fatalf("unexpected delete operation type: %s", deleteOp.OperationType)
	}
	if got := len(svc.GetDistributions("stage12-dist")); got != 0 {
		t.Fatalf("expected distribution to be deleted, got %d entries", got)
	}
}

func TestServiceStage13CertificatesAndDistributionAttach(t *testing.T) {
	svc := NewService()

	if _, _, err := svc.CreateDistribution(DistributionCreateInput{
		BundleID:             "small_1_0",
		DefaultCacheBehavior: DistributionCacheBehavior{Behavior: "cache"},
		DistributionName:     "stage13-dist",
		Origin: DistributionOrigin{
			Name:       "stage13-origin",
			RegionName: "us-east-1",
		},
	}); err != nil {
		t.Fatalf("create distribution: %v", err)
	}

	certificate, createOps, err := svc.CreateCertificate(
		"stage13-cert",
		"example.com",
		[]string{"www.example.com"},
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	if certificate.Name != "stage13-cert" || certificate.Status != "ISSUED" {
		t.Fatalf("unexpected certificate after create: %+v", certificate)
	}
	if len(createOps) != 1 || createOps[0].OperationType != "CreateCertificate" {
		t.Fatalf("unexpected create certificate operations: %+v", createOps)
	}

	certs := svc.GetCertificates("stage13-cert", nil)
	if len(certs) != 1 {
		t.Fatalf("expected one certificate by name, got %d", len(certs))
	}
	certs = svc.GetCertificates("", []string{"ISSUED"})
	if len(certs) != 1 {
		t.Fatalf("expected one ISSUED certificate, got %d", len(certs))
	}

	attachOp, err := svc.AttachCertificateToDistribution("stage13-cert", "stage13-dist")
	if err != nil {
		t.Fatalf("attach certificate to distribution: %v", err)
	}
	if attachOp.OperationType != "AttachCertificateToDistribution" {
		t.Fatalf("unexpected attach operation type: %s", attachOp.OperationType)
	}
	certs = svc.GetCertificates("stage13-cert", nil)
	if len(certs) != 1 || certs[0].InUseResourceCount != 1 {
		t.Fatalf("expected certificate in use by one distribution: %+v", certs)
	}

	setOps, err := svc.SetIPAddressType("stage13-dist", "Distribution", "ipv4", nil)
	if err != nil {
		t.Fatalf("set distribution ip address type: %v", err)
	}
	if len(setOps) != 1 || setOps[0].OperationType != "SetIpAddressType" {
		t.Fatalf("unexpected set ip operation(s): %+v", setOps)
	}
	distributions := svc.GetDistributions("stage13-dist")
	if len(distributions) != 1 || distributions[0].IPAddressType != "ipv4" {
		t.Fatalf("expected distribution ipv4 after update: %+v", distributions)
	}

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage13-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	acceptBundleUpdate := true
	setOps, err = svc.SetIPAddressType("stage13-instance", "Instance", "ipv6", &acceptBundleUpdate)
	if err != nil {
		t.Fatalf("set instance ip address type: %v", err)
	}
	if len(setOps) != 1 || setOps[0].OperationType != "SetIpAddressType" {
		t.Fatalf("unexpected set ip operation(s) for instance: %+v", setOps)
	}
	instance, ok := svc.GetInstance("stage13-instance")
	if !ok || instance.IPAddressType != "ipv6" {
		t.Fatalf("expected instance ipv6 after update: %+v", instance)
	}

	detachOp, err := svc.DetachCertificateFromDistribution("stage13-dist")
	if err != nil {
		t.Fatalf("detach certificate from distribution: %v", err)
	}
	if detachOp.OperationType != "DetachCertificateFromDistribution" {
		t.Fatalf("unexpected detach operation type: %s", detachOp.OperationType)
	}

	deleteOps, err := svc.DeleteCertificate("stage13-cert")
	if err != nil {
		t.Fatalf("delete certificate: %v", err)
	}
	if len(deleteOps) != 1 || deleteOps[0].OperationType != "DeleteCertificate" {
		t.Fatalf("unexpected delete certificate operations: %+v", deleteOps)
	}
	if got := len(svc.GetCertificates("stage13-cert", nil)); got != 0 {
		t.Fatalf("expected certificate deleted, got %d entries", got)
	}
}

func TestServiceStage14DomainsDNS(t *testing.T) {
	svc := NewService()

	createDomainOp, err := svc.CreateDomain("example.com", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create domain: %v", err)
	}
	if createDomainOp.OperationType != "CreateDomain" {
		t.Fatalf("unexpected create domain operation type: %s", createDomainOp.OperationType)
	}

	entryOp, err := svc.CreateDomainEntry("example.com", DomainEntry{
		Name:   "www",
		Type:   "A",
		Target: "198.51.100.10",
	})
	if err != nil {
		t.Fatalf("create domain entry: %v", err)
	}
	if entryOp.OperationType != "CreateDomainEntry" {
		t.Fatalf("unexpected create domain entry operation type: %s", entryOp.OperationType)
	}

	domain, found := svc.GetDomain("example.com")
	if !found {
		t.Fatalf("expected domain to exist")
	}
	if len(domain.DomainEntries) != 1 {
		t.Fatalf("expected one domain entry, got %d", len(domain.DomainEntries))
	}

	if got := len(svc.GetDomains()); got != 1 {
		t.Fatalf("expected one domain in list, got %d", got)
	}

	updateOps, err := svc.UpdateDomainEntry("example.com", DomainEntry{
		Name:   "www",
		Type:   "A",
		Target: "198.51.100.11",
	})
	if err != nil {
		t.Fatalf("update domain entry: %v", err)
	}
	if len(updateOps) != 1 || updateOps[0].OperationType != "UpdateDomainEntry" {
		t.Fatalf("unexpected update domain entry operations: %+v", updateOps)
	}

	domain, found = svc.GetDomain("example.com")
	if !found || len(domain.DomainEntries) != 1 || domain.DomainEntries[0].Target != "198.51.100.11" {
		t.Fatalf("expected updated domain entry target, got %+v", domain)
	}

	deleteEntryOp, err := svc.DeleteDomainEntry("example.com", DomainEntry{
		Name: "www",
		Type: "A",
	})
	if err != nil {
		t.Fatalf("delete domain entry: %v", err)
	}
	if deleteEntryOp.OperationType != "DeleteDomainEntry" {
		t.Fatalf("unexpected delete domain entry operation type: %s", deleteEntryOp.OperationType)
	}

	deleteDomainOp, err := svc.DeleteDomain("example.com")
	if err != nil {
		t.Fatalf("delete domain: %v", err)
	}
	if deleteDomainOp.OperationType != "DeleteDomain" {
		t.Fatalf("unexpected delete domain operation type: %s", deleteDomainOp.OperationType)
	}
	if _, found := svc.GetDomain("example.com"); found {
		t.Fatalf("expected domain to be deleted")
	}
}

func TestServiceStage15BucketsCore(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage15-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	enableObjectVersioning := true
	bucket, createOps, err := svc.CreateBucket(
		"stage15-bucket",
		"small_1_0",
		&enableObjectVersioning,
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if bucket.Name != "stage15-bucket" || bucket.BundleID != "small_1_0" || bucket.ObjectVersioning != "Enabled" {
		t.Fatalf("unexpected bucket after create: %+v", bucket)
	}
	if len(createOps) != 1 || createOps[0].OperationType != "CreateBucket" {
		t.Fatalf("unexpected create bucket operations: %+v", createOps)
	}

	buckets := svc.GetBuckets("stage15-bucket", false)
	if len(buckets) != 1 {
		t.Fatalf("expected one bucket by name, got %d", len(buckets))
	}
	if len(buckets[0].ResourcesReceivingAccess) != 0 {
		t.Fatalf("expected no connected resources when includeConnectedResources=false")
	}

	setAccessOps, err := svc.SetResourceAccessForBucket("stage15-bucket", "stage15-instance", "allow")
	if err != nil {
		t.Fatalf("set resource access for bucket allow: %v", err)
	}
	if len(setAccessOps) != 1 || setAccessOps[0].OperationType != "SetResourceAccessForBucket" {
		t.Fatalf("unexpected set resource access operations: %+v", setAccessOps)
	}
	buckets = svc.GetBuckets("stage15-bucket", true)
	if len(buckets) != 1 || len(buckets[0].ResourcesReceivingAccess) != 1 {
		t.Fatalf("expected one connected resource after allow: %+v", buckets)
	}

	versioning := "Suspended"
	updatedBucket, updateOps, err := svc.UpdateBucket(BucketUpdateInput{
		BucketName: "stage15-bucket",
		AccessRules: &BucketAccessRules{
			AllowPublicOverrides: true,
			GetObject:            "public",
		},
		ReadonlyAccessAccounts:    []string{"111122223333", "111122223333", "222233334444"},
		HasReadonlyAccessAccounts: true,
		Versioning:                &versioning,
	})
	if err != nil {
		t.Fatalf("update bucket: %v", err)
	}
	if updatedBucket.ObjectVersioning != "Suspended" {
		t.Fatalf("expected versioning Suspended after update, got %s", updatedBucket.ObjectVersioning)
	}
	if updatedBucket.AccessRules == nil || updatedBucket.AccessRules.GetObject != "public" {
		t.Fatalf("expected public access rules after update: %+v", updatedBucket.AccessRules)
	}
	if len(updatedBucket.ReadonlyAccessAccounts) != 2 {
		t.Fatalf("expected deduplicated readonly accounts, got %+v", updatedBucket.ReadonlyAccessAccounts)
	}
	if len(updateOps) != 1 || updateOps[0].OperationType != "UpdateBucket" {
		t.Fatalf("unexpected update bucket operations: %+v", updateOps)
	}

	bundles := svc.GetBucketBundles(false)
	if len(bundles) == 0 {
		t.Fatalf("expected active bucket bundles")
	}
	allBundles := svc.GetBucketBundles(true)
	if len(allBundles) <= len(bundles) {
		t.Fatalf("expected includeInactive=true to include at least one extra bundle")
	}

	startTime := time.Now().UTC().Add(-24 * time.Hour)
	endTime := time.Now().UTC()
	metricName, metricData, err := svc.GetBucketMetricData(BucketMetricInput{
		BucketName: "stage15-bucket",
		StartTime:  startTime,
		EndTime:    endTime,
		MetricName: "BucketSizeBytes",
		Period:     86400,
		Statistics: []string{"Maximum", "Average"},
		Unit:       "Bytes",
	})
	if err != nil {
		t.Fatalf("get bucket metric data: %v", err)
	}
	if metricName != "BucketSizeBytes" || len(metricData) == 0 {
		t.Fatalf("unexpected metric output: metricName=%s metricData=%+v", metricName, metricData)
	}

	updateBundleOps, err := svc.UpdateBucketBundle("stage15-bucket", "medium_1_0")
	if err != nil {
		t.Fatalf("update bucket bundle: %v", err)
	}
	if len(updateBundleOps) != 1 || updateBundleOps[0].OperationType != "UpdateBucketBundle" {
		t.Fatalf("unexpected update bucket bundle operations: %+v", updateBundleOps)
	}

	denyOps, err := svc.SetResourceAccessForBucket("stage15-bucket", "stage15-instance", "deny")
	if err != nil {
		t.Fatalf("set resource access for bucket deny: %v", err)
	}
	if len(denyOps) != 1 || denyOps[0].OperationType != "SetResourceAccessForBucket" {
		t.Fatalf("unexpected deny resource access operations: %+v", denyOps)
	}

	deleteOps, err := svc.DeleteBucket("stage15-bucket", false)
	if err != nil {
		t.Fatalf("delete bucket: %v", err)
	}
	if len(deleteOps) != 1 || deleteOps[0].OperationType != "DeleteBucket" {
		t.Fatalf("unexpected delete bucket operations: %+v", deleteOps)
	}
	if got := len(svc.GetBuckets("stage15-bucket", true)); got != 0 {
		t.Fatalf("expected bucket to be deleted, got %d results", got)
	}
}

func TestServiceStage16BucketKeysAndContactMethods(t *testing.T) {
	svc := NewService()

	if _, _, err := svc.CreateBucket("stage16-bucket", "small_1_0", nil, nil); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	accessKey, createKeyOps, err := svc.CreateBucketAccessKey("stage16-bucket")
	if err != nil {
		t.Fatalf("create bucket access key: %v", err)
	}
	if accessKey.AccessKeyID == "" || accessKey.SecretAccessKey == "" {
		t.Fatalf("expected access key id and secret to be returned")
	}
	if len(createKeyOps) != 1 || createKeyOps[0].OperationType != "CreateBucketAccessKey" {
		t.Fatalf("unexpected create bucket access key operations: %+v", createKeyOps)
	}

	accessKeys, err := svc.GetBucketAccessKeys("stage16-bucket")
	if err != nil {
		t.Fatalf("get bucket access keys: %v", err)
	}
	if len(accessKeys) != 1 {
		t.Fatalf("expected one bucket access key, got %d", len(accessKeys))
	}
	if accessKeys[0].SecretAccessKey != "" {
		t.Fatalf("expected secret access key omitted from GetBucketAccessKeys")
	}

	deleteKeyOps, err := svc.DeleteBucketAccessKey("stage16-bucket", accessKeys[0].AccessKeyID)
	if err != nil {
		t.Fatalf("delete bucket access key: %v", err)
	}
	if len(deleteKeyOps) != 1 || deleteKeyOps[0].OperationType != "DeleteBucketAccessKey" {
		t.Fatalf("unexpected delete bucket access key operations: %+v", deleteKeyOps)
	}

	createContactOps, err := svc.CreateContactMethod("alerts@example.com", "email")
	if err != nil {
		t.Fatalf("create contact method: %v", err)
	}
	if len(createContactOps) != 1 || createContactOps[0].OperationType != "CreateContactMethod" {
		t.Fatalf("unexpected create contact method operations: %+v", createContactOps)
	}

	contactMethods := svc.GetContactMethods(nil)
	if len(contactMethods) != 1 || contactMethods[0].Protocol != "Email" || contactMethods[0].Status != "PendingVerification" {
		t.Fatalf("unexpected contact methods after create: %+v", contactMethods)
	}

	verifyOps, err := svc.SendContactMethodVerification("Email")
	if err != nil {
		t.Fatalf("send contact method verification: %v", err)
	}
	if len(verifyOps) != 1 || verifyOps[0].OperationType != "SendContactMethodVerification" {
		t.Fatalf("unexpected send verification operations: %+v", verifyOps)
	}

	contactMethods = svc.GetContactMethods([]string{"Email"})
	if len(contactMethods) != 1 || contactMethods[0].Status != "Valid" {
		t.Fatalf("expected verified contact method status Valid: %+v", contactMethods)
	}

	deleteContactOps, err := svc.DeleteContactMethod("EMAIL")
	if err != nil {
		t.Fatalf("delete contact method: %v", err)
	}
	if len(deleteContactOps) != 1 || deleteContactOps[0].OperationType != "DeleteContactMethod" {
		t.Fatalf("unexpected delete contact method operations: %+v", deleteContactOps)
	}
	if got := len(svc.GetContactMethods(nil)); got != 0 {
		t.Fatalf("expected no contact methods after delete, got %d", got)
	}
}

func TestServiceStage17ContainerServicesCore(t *testing.T) {
	svc := NewService()

	containerService, err := svc.CreateContainerService(
		"stage17-service",
		"nano",
		1,
		map[string][]string{"example.com": []string{"stage17-cert"}},
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("create container service: %v", err)
	}
	if containerService.Name != "stage17-service" || containerService.Power != "nano" || containerService.Scale != 1 {
		t.Fatalf("unexpected container service after create: %+v", containerService)
	}

	services := svc.GetContainerServices("")
	if len(services) != 1 {
		t.Fatalf("expected one container service, got %d", len(services))
	}
	services = svc.GetContainerServices("stage17-service")
	if len(services) != 1 || services[0].Name != "stage17-service" {
		t.Fatalf("unexpected filtered container services: %+v", services)
	}

	scale := int32(2)
	isDisabled := true
	power := "micro"
	updated, err := svc.UpdateContainerService(ContainerServiceUpdateInput{
		ServiceName: "stage17-service",
		Scale:       &scale,
		IsDisabled:  &isDisabled,
		Power:       &power,
	})
	if err != nil {
		t.Fatalf("update container service: %v", err)
	}
	if updated.Scale != 2 || !updated.IsDisabled || updated.State != "DISABLED" || updated.Power != "micro" {
		t.Fatalf("unexpected container service after update: %+v", updated)
	}

	metadata := svc.GetContainerAPIMetadata()
	if len(metadata) == 0 {
		t.Fatalf("expected container API metadata")
	}

	metricName, metricData, err := svc.GetContainerServiceMetricData(ContainerServiceMetricInput{
		ServiceName: "stage17-service",
		StartTime:   time.Now().UTC().Add(-15 * time.Minute),
		EndTime:     time.Now().UTC(),
		MetricName:  "CPUUtilization",
		Period:      300,
		Statistics:  []string{"Average", "Maximum"},
	})
	if err != nil {
		t.Fatalf("get container service metric data: %v", err)
	}
	if metricName != "CPUUtilization" || len(metricData) == 0 {
		t.Fatalf("unexpected metric output: metricName=%s metricData=%+v", metricName, metricData)
	}

	powers := svc.GetContainerServicePowers()
	if len(powers) == 0 {
		t.Fatalf("expected container service powers")
	}

	login := svc.CreateContainerServiceRegistryLogin()
	if login.Username == "" || login.Password == "" || login.Registry == "" || login.ExpiresAt.IsZero() {
		t.Fatalf("unexpected container service registry login: %+v", login)
	}

	if err := svc.DeleteContainerService("stage17-service"); err != nil {
		t.Fatalf("delete container service: %v", err)
	}
	if got := len(svc.GetContainerServices("stage17-service")); got != 0 {
		t.Fatalf("expected container service deleted, got %d entries", got)
	}
}

func TestServiceStage18ContainerDeploymentsImagesLogs(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateContainerService("stage18-service", "nano", 1, nil, nil); err != nil {
		t.Fatalf("create container service: %v", err)
	}

	firstImage, err := svc.RegisterContainerImage("stage18-service", "web", "sha256:first")
	if err != nil {
		t.Fatalf("register container image 1: %v", err)
	}
	secondImage, err := svc.RegisterContainerImage("stage18-service", "web", "sha256:second")
	if err != nil {
		t.Fatalf("register container image 2: %v", err)
	}
	if firstImage.Image == secondImage.Image {
		t.Fatalf("expected unique registered image names")
	}

	images, err := svc.GetContainerImages("stage18-service")
	if err != nil {
		t.Fatalf("get container images: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected two registered images, got %d", len(images))
	}

	healthyThreshold := int32(2)
	intervalSeconds := int32(10)
	containerService, err := svc.CreateContainerServiceDeployment(
		"stage18-service",
		map[string]ContainerServiceContainer{
			"web": {
				Image: secondImage.Image,
				Ports: map[string]string{"80": "HTTP"},
			},
		},
		&ContainerServiceEndpoint{
			ContainerName: "web",
			ContainerPort: 80,
			HealthCheck: &ContainerServiceHealthCheckConfig{
				HealthyThreshold: &healthyThreshold,
				IntervalSeconds:  &intervalSeconds,
			},
		},
	)
	if err != nil {
		t.Fatalf("create container service deployment: %v", err)
	}
	if containerService.Name != "stage18-service" {
		t.Fatalf("unexpected deployment response container service: %+v", containerService)
	}

	deployments, err := svc.GetContainerServiceDeployments("stage18-service")
	if err != nil {
		t.Fatalf("get container service deployments: %v", err)
	}
	if len(deployments) != 1 || deployments[0].State != "ACTIVE" || deployments[0].Version != 1 {
		t.Fatalf("unexpected deployments: %+v", deployments)
	}

	logEvents, nextPageToken, err := svc.GetContainerLog(ContainerLogInput{
		ServiceName:   "stage18-service",
		ContainerName: "web",
		FilterPattern: "activated",
	})
	if err != nil {
		t.Fatalf("get container log: %v", err)
	}
	if len(logEvents) == 0 || nextPageToken != "" {
		t.Fatalf("unexpected container log result: events=%+v nextPageToken=%q", logEvents, nextPageToken)
	}

	if _, _, err := svc.GetContainerLog(ContainerLogInput{
		ServiceName:   "stage18-service",
		ContainerName: "web",
		PageToken:     "bad-token",
	}); err != ErrInvalidParameter {
		t.Fatalf("expected invalid parameter for bad page token, got: %v", err)
	}

	if err := svc.DeleteContainerImage("stage18-service", firstImage.Image); err != nil {
		t.Fatalf("delete container image: %v", err)
	}
	images, err = svc.GetContainerImages("stage18-service")
	if err != nil {
		t.Fatalf("get container images after delete: %v", err)
	}
	if len(images) != 1 || images[0].Image != secondImage.Image {
		t.Fatalf("unexpected remaining images after delete: %+v", images)
	}

	if err := svc.DeleteContainerService("stage18-service"); err != nil {
		t.Fatalf("delete container service: %v", err)
	}
}

func TestServiceStage19RelationalDatabaseCore(t *testing.T) {
	svc := NewService()

	ops, err := svc.CreateRelationalDatabase(RelationalDatabaseCreateInput{
		RelationalDatabaseName:        "stage19-db",
		AvailabilityZone:              "us-east-1a",
		MasterDatabaseName:            "appdb",
		MasterUsername:                "admin",
		MasterUserPassword:            "Stage19pass!",
		RelationalDatabaseBlueprintID: "mysql_8_0",
		RelationalDatabaseBundleID:    "micro_1_0",
		PubliclyAccessible:            boolPtr(false),
		Tags:                          map[string]string{"env": "test"},
	})
	if err != nil {
		t.Fatalf("create relational database: %v", err)
	}
	if len(ops) != 1 || ops[0].OperationType != "CreateRelationalDatabase" {
		t.Fatalf("unexpected create operations: %+v", ops)
	}

	db, found := svc.GetRelationalDatabase("stage19-db")
	if !found || db.Name != "stage19-db" {
		t.Fatalf("unexpected relational database after create: found=%v db=%+v", found, db)
	}

	page, err := svc.GetRelationalDatabases("")
	if err != nil {
		t.Fatalf("get relational databases: %v", err)
	}
	if len(page.RelationalDatabases) != 1 {
		t.Fatalf("expected one relational database, got %d", len(page.RelationalDatabases))
	}

	updateOps, err := svc.UpdateRelationalDatabase(RelationalDatabaseUpdateInput{
		RelationalDatabaseName:     "stage19-db",
		PubliclyAccessible:         boolPtr(true),
		EnableBackupRetention:      boolPtr(true),
		RotateMasterUserPassword:   boolPtr(true),
		PreferredBackupWindow:      strPtr("04:00-04:30"),
		PreferredMaintenanceWindow: strPtr("Sun:05:00-Sun:05:30"),
		ApplyImmediately:           boolPtr(true),
	})
	if err != nil {
		t.Fatalf("update relational database: %v", err)
	}
	if len(updateOps) != 1 || updateOps[0].OperationType != "UpdateRelationalDatabase" {
		t.Fatalf("unexpected update operations: %+v", updateOps)
	}

	rebootOps, err := svc.RebootRelationalDatabase("stage19-db")
	if err != nil {
		t.Fatalf("reboot relational database: %v", err)
	}
	if len(rebootOps) != 1 || rebootOps[0].OperationType != "RebootRelationalDatabase" {
		t.Fatalf("unexpected reboot operations: %+v", rebootOps)
	}

	stopOps, err := svc.StopRelationalDatabase("stage19-db", "stage19-db-stop-snap")
	if err != nil {
		t.Fatalf("stop relational database: %v", err)
	}
	if len(stopOps) != 1 || stopOps[0].OperationType != "StopRelationalDatabase" {
		t.Fatalf("unexpected stop operations: %+v", stopOps)
	}

	startOps, err := svc.StartRelationalDatabase("stage19-db")
	if err != nil {
		t.Fatalf("start relational database: %v", err)
	}
	if len(startOps) != 1 || startOps[0].OperationType != "StartRelationalDatabase" {
		t.Fatalf("unexpected start operations: %+v", startOps)
	}

	deleteOps, err := svc.DeleteRelationalDatabase(RelationalDatabaseDeleteInput{
		RelationalDatabaseName: "stage19-db",
		SkipFinalSnapshot:      boolPtr(true),
	})
	if err != nil {
		t.Fatalf("delete relational database: %v", err)
	}
	if len(deleteOps) != 1 || deleteOps[0].OperationType != "DeleteRelationalDatabase" {
		t.Fatalf("unexpected delete operations: %+v", deleteOps)
	}

	if _, found := svc.GetRelationalDatabase("stage19-db"); found {
		t.Fatalf("expected relational database deleted")
	}
}

func TestServiceStage20RelationalSnapshotsRestoreLogging(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateRelationalDatabase(RelationalDatabaseCreateInput{
		RelationalDatabaseName:        "stage20-db",
		AvailabilityZone:              "us-east-1a",
		MasterDatabaseName:            "appdb",
		MasterUsername:                "admin",
		MasterUserPassword:            "Stage20pass!",
		RelationalDatabaseBlueprintID: "mysql_8_0",
		RelationalDatabaseBundleID:    "micro_1_0",
	}); err != nil {
		t.Fatalf("create relational database: %v", err)
	}

	createSnapshotOps, err := svc.CreateRelationalDatabaseSnapshot("stage20-db", "stage20-snap", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create relational database snapshot: %v", err)
	}
	if len(createSnapshotOps) != 1 || createSnapshotOps[0].OperationType != "CreateRelationalDatabaseSnapshot" {
		t.Fatalf("unexpected create snapshot operations: %+v", createSnapshotOps)
	}

	snapshot, found := svc.GetRelationalDatabaseSnapshot("stage20-snap")
	if !found || snapshot.Name != "stage20-snap" {
		t.Fatalf("unexpected snapshot after create: found=%v snapshot=%+v", found, snapshot)
	}

	snapshotPage, err := svc.GetRelationalDatabaseSnapshots("")
	if err != nil {
		t.Fatalf("get relational database snapshots: %v", err)
	}
	if len(snapshotPage.RelationalDatabaseSnapshots) != 1 {
		t.Fatalf("expected one snapshot, got %d", len(snapshotPage.RelationalDatabaseSnapshots))
	}

	createFromSnapshotOps, err := svc.CreateRelationalDatabaseFromSnapshot(RelationalDatabaseFromSnapshotInput{
		RelationalDatabaseName:         "stage20-db-restore",
		RelationalDatabaseSnapshotName: "stage20-snap",
		RelationalDatabaseBundleID:     "small_1_0",
		PubliclyAccessible:             boolPtr(true),
		Tags:                           map[string]string{"restore": "true"},
	})
	if err != nil {
		t.Fatalf("create relational database from snapshot: %v", err)
	}
	if len(createFromSnapshotOps) != 1 || createFromSnapshotOps[0].OperationType != "CreateRelationalDatabaseFromSnapshot" {
		t.Fatalf("unexpected create from snapshot operations: %+v", createFromSnapshotOps)
	}

	eventsPage, err := svc.GetRelationalDatabaseEvents("stage20-db", nil, "")
	if err != nil {
		t.Fatalf("get relational database events: %v", err)
	}
	if len(eventsPage.RelationalDatabaseEvents) == 0 {
		t.Fatalf("expected relational database events")
	}

	logStreams, err := svc.GetRelationalDatabaseLogStreams("stage20-db")
	if err != nil {
		t.Fatalf("get relational database log streams: %v", err)
	}
	if len(logStreams) == 0 {
		t.Fatalf("expected relational database log streams")
	}

	startFromHead := true
	logEventsPage, err := svc.GetRelationalDatabaseLogEvents(RelationalDatabaseLogEventsInput{
		RelationalDatabaseName: "stage20-db",
		LogStreamName:          logStreams[0],
		StartFromHead:          &startFromHead,
	})
	if err != nil {
		t.Fatalf("get relational database log events: %v", err)
	}
	if len(logEventsPage.ResourceLogEvents) == 0 {
		t.Fatalf("expected relational database log events")
	}

	deleteSnapshotOps, err := svc.DeleteRelationalDatabaseSnapshot("stage20-snap")
	if err != nil {
		t.Fatalf("delete relational database snapshot: %v", err)
	}
	if len(deleteSnapshotOps) != 1 || deleteSnapshotOps[0].OperationType != "DeleteRelationalDatabaseSnapshot" {
		t.Fatalf("unexpected delete snapshot operations: %+v", deleteSnapshotOps)
	}
}

func TestServiceStage21RelationalConfigCatalog(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateRelationalDatabase(RelationalDatabaseCreateInput{
		RelationalDatabaseName:        "stage21-db",
		AvailabilityZone:              "us-east-1a",
		MasterDatabaseName:            "appdb",
		MasterUsername:                "admin",
		MasterUserPassword:            "Stage21pass!",
		RelationalDatabaseBlueprintID: "mysql_8_0",
		RelationalDatabaseBundleID:    "micro_1_0",
	}); err != nil {
		t.Fatalf("create relational database: %v", err)
	}

	blueprintsPage, err := svc.GetRelationalDatabaseBlueprints("")
	if err != nil {
		t.Fatalf("get relational database blueprints: %v", err)
	}
	if len(blueprintsPage.Blueprints) == 0 {
		t.Fatalf("expected relational database blueprints")
	}

	bundlesPage, err := svc.GetRelationalDatabaseBundles(true, "")
	if err != nil {
		t.Fatalf("get relational database bundles: %v", err)
	}
	if len(bundlesPage.Bundles) == 0 {
		t.Fatalf("expected relational database bundles")
	}

	createdAt, masterUserPassword, err := svc.GetRelationalDatabaseMasterUserPassword("stage21-db", "CURRENT")
	if err != nil {
		t.Fatalf("get master user password: %v", err)
	}
	if createdAt.IsZero() || masterUserPassword == "" {
		t.Fatalf("unexpected master user password response: createdAt=%v password=%q", createdAt, masterUserPassword)
	}

	metricName, metricData, err := svc.GetRelationalDatabaseMetricData(RelationalDatabaseMetricInput{
		RelationalDatabaseName: "stage21-db",
		StartTime:              time.Now().UTC().Add(-5 * time.Minute),
		EndTime:                time.Now().UTC(),
		MetricName:             "CPUUtilization",
		Period:                 60,
		Statistics:             []string{"Average", "Maximum"},
		Unit:                   "Percent",
	})
	if err != nil {
		t.Fatalf("get relational database metric data: %v", err)
	}
	if metricName != "CPUUtilization" || len(metricData) == 0 {
		t.Fatalf("unexpected metric response: metric=%q points=%d", metricName, len(metricData))
	}

	parametersPage, err := svc.GetRelationalDatabaseParameters("stage21-db", "")
	if err != nil {
		t.Fatalf("get relational database parameters: %v", err)
	}
	if len(parametersPage.Parameters) == 0 {
		t.Fatalf("expected relational database parameters")
	}

	updateOps, err := svc.UpdateRelationalDatabaseParameters("stage21-db", []RelationalDatabaseParameter{
		{
			ParameterName:  "max_connections",
			ParameterValue: "200",
			ApplyMethod:    "pending-reboot",
		},
	})
	if err != nil {
		t.Fatalf("update relational database parameters: %v", err)
	}
	if len(updateOps) != 1 || updateOps[0].OperationType != "UpdateRelationalDatabaseParameters" {
		t.Fatalf("unexpected update parameter operations: %+v", updateOps)
	}

	parametersPage, err = svc.GetRelationalDatabaseParameters("stage21-db", "")
	if err != nil {
		t.Fatalf("get relational database parameters after update: %v", err)
	}
	found := false
	for _, parameter := range parametersPage.Parameters {
		if parameter.ParameterName == "max_connections" {
			found = true
			if parameter.ParameterValue != "200" {
				t.Fatalf("expected updated parameter value 200, got %q", parameter.ParameterValue)
			}
		}
	}
	if !found {
		t.Fatalf("expected max_connections parameter after update")
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func strPtr(v string) *string {
	return &v
}
