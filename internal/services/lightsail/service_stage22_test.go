package lightsail

import (
	"testing"
	"time"
)

func TestServiceStage22GlobalMisc(t *testing.T) {
	svc := NewService()

	if _, err := svc.CreateInstances("us-east-1a", "amazon_linux_2", "micro_2_0", []string{"stage22-instance"}, nil); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := svc.CreateDisk("us-east-1a", "stage22-disk", 32, nil); err != nil {
		t.Fatalf("create disk: %v", err)
	}
	if _, err := svc.CreateDiskSnapshot("stage22-disk", "", "stage22-disk-snapshot", nil); err != nil {
		t.Fatalf("create disk snapshot: %v", err)
	}
	if _, err := svc.ExportSnapshot("stage22-disk-snapshot"); err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	exportRecords := svc.GetExportSnapshotRecords()
	if len(exportRecords) == 0 {
		t.Fatalf("expected export snapshot records")
	}

	blueprintsPage, err := svc.GetBlueprints(true, "", "")
	if err != nil {
		t.Fatalf("get blueprints: %v", err)
	}
	if len(blueprintsPage.Blueprints) == 0 {
		t.Fatalf("expected blueprints")
	}

	lfrBlueprintsPage, err := svc.GetBlueprints(true, "LfR", "")
	if err != nil {
		t.Fatalf("get LfR blueprints: %v", err)
	}
	if len(lfrBlueprintsPage.Blueprints) == 0 {
		t.Fatalf("expected Lightsail for Research blueprints")
	}

	bundlesPage, err := svc.GetBundles(true, "", "")
	if err != nil {
		t.Fatalf("get bundles: %v", err)
	}
	if len(bundlesPage.Bundles) == 0 {
		t.Fatalf("expected bundles")
	}

	activeNames, _, err := svc.GetActiveNames("")
	if err != nil {
		t.Fatalf("get active names: %v", err)
	}
	foundInstance := false
	for _, name := range activeNames {
		if name == "stage22-instance" {
			foundInstance = true
			break
		}
	}
	if !foundInstance {
		t.Fatalf("expected stage22-instance in active names: %+v", activeNames)
	}

	historyPage, err := svc.GetSetupHistory("stage22-instance", "")
	if err != nil {
		t.Fatalf("get setup history: %v", err)
	}
	if len(historyPage.SetupHistory) == 0 {
		t.Fatalf("expected setup history entries")
	}

	start := time.Now().UTC().Add(-30 * time.Minute)
	end := time.Now().UTC()
	estimates, err := svc.GetCostEstimate("stage22-instance", start, end)
	if err != nil {
		t.Fatalf("get cost estimate: %v", err)
	}
	if len(estimates) != 1 || estimates[0].ResourceName != "stage22-instance" {
		t.Fatalf("unexpected cost estimate payload: %+v", estimates)
	}

	if svc.IsVpcPeered() {
		t.Fatalf("expected VPC to start unpeered")
	}
	peerOp, err := svc.PeerVpc()
	if err != nil {
		t.Fatalf("peer vpc: %v", err)
	}
	if peerOp.OperationType != "PeerVpc" || !svc.IsVpcPeered() {
		t.Fatalf("expected VPC to be peered")
	}
	unpeerOp, err := svc.UnpeerVpc()
	if err != nil {
		t.Fatalf("unpeer vpc: %v", err)
	}
	if unpeerOp.OperationType != "UnpeerVpc" || svc.IsVpcPeered() {
		t.Fatalf("expected VPC to be unpeered")
	}

	cfOps, err := svc.CreateCloudFormationStack([]InstanceEntry{{
		AvailabilityZone: "us-east-1a",
		InstanceType:     "t3.micro",
		PortInfoSource:   "DEFAULT",
		SourceName:       exportRecords[0].Name,
	}})
	if err != nil {
		t.Fatalf("create cloudformation stack: %v", err)
	}
	if len(cfOps) != 1 || cfOps[0].OperationType != "CreateCloudFormationStack" {
		t.Fatalf("unexpected cloudformation operations: %+v", cfOps)
	}

	recordsPage, err := svc.GetCloudFormationStackRecords("")
	if err != nil {
		t.Fatalf("get cloudformation stack records: %v", err)
	}
	if len(recordsPage.CloudFormationStackRecords) == 0 {
		t.Fatalf("expected cloudformation stack records")
	}

	guiDetails, err := svc.CreateGUISessionAccessDetails("stage22-instance")
	if err != nil {
		t.Fatalf("create GUI session access details: %v", err)
	}
	if guiDetails.ResourceName != "stage22-instance" || len(guiDetails.Sessions) == 0 {
		t.Fatalf("unexpected GUI session access details: %+v", guiDetails)
	}

	startGUIOps, err := svc.StartGUISession("stage22-instance")
	if err != nil {
		t.Fatalf("start GUI session: %v", err)
	}
	if len(startGUIOps) != 1 || startGUIOps[0].OperationType != "StartGUISession" {
		t.Fatalf("unexpected start GUI session operations: %+v", startGUIOps)
	}

	stopGUIOps, err := svc.StopGUISession("stage22-instance")
	if err != nil {
		t.Fatalf("stop GUI session: %v", err)
	}
	if len(stopGUIOps) != 1 || stopGUIOps[0].OperationType != "StopGUISession" {
		t.Fatalf("unexpected stop GUI session operations: %+v", stopGUIOps)
	}

	metricName, metricData, err := svc.GetInstanceMetricData(InstanceMetricInput{
		InstanceName: "stage22-instance",
		EndTime:      end,
		MetricName:   "CPUUtilization",
		Period:       300,
		StartTime:    start,
		Statistics:   []string{"Average", "Maximum", "Minimum", "SampleCount", "Sum"},
		Unit:         "Percent",
	})
	if err != nil {
		t.Fatalf("get instance metric data: %v", err)
	}
	if metricName != "CPUUtilization" || len(metricData) == 0 {
		t.Fatalf("unexpected metric output: metricName=%s count=%d", metricName, len(metricData))
	}
	if metricData[0].Average == nil || metricData[0].Maximum == nil || metricData[0].Minimum == nil || metricData[0].SampleCount == nil || metricData[0].Sum == nil {
		t.Fatalf("expected all metric statistics to be populated: %+v", metricData[0])
	}
}
