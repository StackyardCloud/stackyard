package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage22GlobalMisc(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage22-instance"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateDisk", []byte(`{"availabilityZone":"us-east-1a","diskName":"stage22-disk","sizeInGb":32}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "CreateDiskSnapshot", []byte(`{"diskName":"stage22-disk","diskSnapshotName":"stage22-disk-snapshot"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "ExportSnapshot", []byte(`{"sourceSnapshotName":"stage22-disk-snapshot"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetExportSnapshotRecords", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var exportRecordsOut struct {
		ExportSnapshotRecords []struct {
			Name string `json:"name"`
		} `json:"exportSnapshotRecords"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &exportRecordsOut); err != nil {
		t.Fatalf("unmarshal GetExportSnapshotRecords: %v", err)
	}
	if len(exportRecordsOut.ExportSnapshotRecords) == 0 {
		t.Fatalf("expected export snapshot records")
	}

	resp = lightsailRequest(t, ts, "GetBlueprints", []byte(`{"includeInactive":true}`))
	assertStatus(t, resp, http.StatusOK)
	var blueprintsOut struct {
		Blueprints []struct {
			BlueprintID string `json:"blueprintId"`
		} `json:"blueprints"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &blueprintsOut); err != nil {
		t.Fatalf("unmarshal GetBlueprints: %v", err)
	}
	if len(blueprintsOut.Blueprints) == 0 {
		t.Fatalf("expected blueprints")
	}

	resp = lightsailRequest(t, ts, "GetBundles", []byte(`{"includeInactive":true}`))
	assertStatus(t, resp, http.StatusOK)
	var bundlesOut struct {
		Bundles []struct {
			BundleID string `json:"bundleId"`
		} `json:"bundles"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &bundlesOut); err != nil {
		t.Fatalf("unmarshal GetBundles: %v", err)
	}
	if len(bundlesOut.Bundles) == 0 {
		t.Fatalf("expected bundles")
	}

	resp = lightsailRequest(t, ts, "GetActiveNames", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var activeNamesOut struct {
		ActiveNames []string `json:"activeNames"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &activeNamesOut); err != nil {
		t.Fatalf("unmarshal GetActiveNames: %v", err)
	}
	found := false
	for _, name := range activeNamesOut.ActiveNames {
		if name == "stage22-instance" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected stage22-instance in active names")
	}

	resp = lightsailRequest(t, ts, "GetSetupHistory", []byte(`{"resourceName":"stage22-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	var setupHistoryOut struct {
		SetupHistory []struct {
			OperationID string `json:"operationId"`
		} `json:"setupHistory"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &setupHistoryOut); err != nil {
		t.Fatalf("unmarshal GetSetupHistory: %v", err)
	}
	if len(setupHistoryOut.SetupHistory) == 0 {
		t.Fatalf("expected setup history")
	}

	now := time.Now().UTC()
	resp = lightsailRequest(t, ts, "GetCostEstimate", []byte(`{"resourceName":"stage22-instance","startTime":`+timeString(now.Add(-time.Hour))+`,"endTime":`+timeString(now)+`}`))
	assertStatus(t, resp, http.StatusOK)
	var costOut struct {
		ResourcesBudgetEstimate []struct {
			ResourceName string `json:"resourceName"`
		} `json:"resourcesBudgetEstimate"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &costOut); err != nil {
		t.Fatalf("unmarshal GetCostEstimate: %v", err)
	}
	if len(costOut.ResourcesBudgetEstimate) == 0 || costOut.ResourcesBudgetEstimate[0].ResourceName != "stage22-instance" {
		t.Fatalf("unexpected cost estimate output: %+v", costOut)
	}

	resp = lightsailRequest(t, ts, "IsVpcPeered", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var vpcStateOut struct {
		IsPeered bool `json:"isPeered"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &vpcStateOut); err != nil {
		t.Fatalf("unmarshal IsVpcPeered: %v", err)
	}
	if vpcStateOut.IsPeered {
		t.Fatalf("expected VPC to start unpeered")
	}

	resp = lightsailRequest(t, ts, "PeerVpc", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "IsVpcPeered", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &vpcStateOut); err != nil {
		t.Fatalf("unmarshal IsVpcPeered after peer: %v", err)
	}
	if !vpcStateOut.IsPeered {
		t.Fatalf("expected VPC to be peered")
	}

	resp = lightsailRequest(t, ts, "UnpeerVpc", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateCloudFormationStack", []byte(`{"instances":[{"availabilityZone":"us-east-1a","instanceType":"t3.micro","portInfoSource":"DEFAULT","sourceName":"`+exportRecordsOut.ExportSnapshotRecords[0].Name+`"}]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "GetCloudFormationStackRecords", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var stackRecordsOut struct {
		CloudFormationStackRecords []struct {
			Name string `json:"name"`
		} `json:"cloudFormationStackRecords"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &stackRecordsOut); err != nil {
		t.Fatalf("unmarshal GetCloudFormationStackRecords: %v", err)
	}
	if len(stackRecordsOut.CloudFormationStackRecords) == 0 {
		t.Fatalf("expected cloudformation stack records")
	}

	resp = lightsailRequest(t, ts, "CreateGUISessionAccessDetails", []byte(`{"resourceName":"stage22-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	var guiOut struct {
		ResourceName string `json:"resourceName"`
		Sessions     []struct {
			URL string `json:"url"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &guiOut); err != nil {
		t.Fatalf("unmarshal CreateGUISessionAccessDetails: %v", err)
	}
	if guiOut.ResourceName != "stage22-instance" || len(guiOut.Sessions) == 0 || guiOut.Sessions[0].URL == "" {
		t.Fatalf("unexpected gui session access details: %+v", guiOut)
	}

	resp = lightsailRequest(t, ts, "StartGUISession", []byte(`{"resourceName":"stage22-instance"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "StopGUISession", []byte(`{"resourceName":"stage22-instance"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetInstanceMetricData", []byte(`{"instanceName":"stage22-instance","metricName":"CPUUtilization","startTime":`+timeString(now.Add(-30*time.Minute))+`,"endTime":`+timeString(now)+`,"period":300,"statistics":["Average","Maximum"],"unit":"Percent"}`))
	assertStatus(t, resp, http.StatusOK)
	var metricsOut struct {
		MetricName string `json:"metricName"`
		MetricData []struct {
			Timestamp float64  `json:"timestamp"`
			Average   *float64 `json:"average"`
		} `json:"metricData"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &metricsOut); err != nil {
		t.Fatalf("unmarshal GetInstanceMetricData: %v", err)
	}
	if metricsOut.MetricName != "CPUUtilization" || len(metricsOut.MetricData) == 0 || metricsOut.MetricData[0].Average == nil {
		t.Fatalf("unexpected instance metric data: %+v", metricsOut)
	}
}

func TestLightsailStage22SDKClientGlobalMisc(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := awslightsail.NewFromConfig(cfg, func(o *awslightsail.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	if _, err := client.CreateInstances(ctx, &awslightsail.CreateInstancesInput{
		AvailabilityZone: aws.String("us-east-1a"),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{"sdk-stage22-instance"},
	}); err != nil {
		t.Fatalf("create instances: %v", err)
	}
	if _, err := client.CreateDisk(ctx, &awslightsail.CreateDiskInput{
		AvailabilityZone: aws.String("us-east-1a"),
		DiskName:         aws.String("sdk-stage22-disk"),
		SizeInGb:         aws.Int32(32),
	}); err != nil {
		t.Fatalf("create disk: %v", err)
	}
	if _, err := client.CreateDiskSnapshot(ctx, &awslightsail.CreateDiskSnapshotInput{
		DiskName:         aws.String("sdk-stage22-disk"),
		DiskSnapshotName: aws.String("sdk-stage22-disk-snapshot"),
	}); err != nil {
		t.Fatalf("create disk snapshot: %v", err)
	}
	if _, err := client.ExportSnapshot(ctx, &awslightsail.ExportSnapshotInput{SourceSnapshotName: aws.String("sdk-stage22-disk-snapshot")}); err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	exportRecordsOut, err := client.GetExportSnapshotRecords(ctx, &awslightsail.GetExportSnapshotRecordsInput{})
	if err != nil {
		t.Fatalf("get export snapshot records: %v", err)
	}
	if len(exportRecordsOut.ExportSnapshotRecords) == 0 || exportRecordsOut.ExportSnapshotRecords[0].Name == nil {
		t.Fatalf("expected export snapshot records")
	}

	blueprintsOut, err := client.GetBlueprints(ctx, &awslightsail.GetBlueprintsInput{IncludeInactive: aws.Bool(true)})
	if err != nil {
		t.Fatalf("get blueprints: %v", err)
	}
	if len(blueprintsOut.Blueprints) == 0 {
		t.Fatalf("expected blueprints")
	}

	bundlesOut, err := client.GetBundles(ctx, &awslightsail.GetBundlesInput{IncludeInactive: aws.Bool(true)})
	if err != nil {
		t.Fatalf("get bundles: %v", err)
	}
	if len(bundlesOut.Bundles) == 0 {
		t.Fatalf("expected bundles")
	}

	activeNamesOut, err := client.GetActiveNames(ctx, &awslightsail.GetActiveNamesInput{})
	if err != nil {
		t.Fatalf("get active names: %v", err)
	}
	if len(activeNamesOut.ActiveNames) == 0 {
		t.Fatalf("expected active names")
	}

	setupHistoryOut, err := client.GetSetupHistory(ctx, &awslightsail.GetSetupHistoryInput{ResourceName: aws.String("sdk-stage22-instance")})
	if err != nil {
		t.Fatalf("get setup history: %v", err)
	}
	if len(setupHistoryOut.SetupHistory) == 0 {
		t.Fatalf("expected setup history")
	}

	now := time.Now().UTC()
	costOut, err := client.GetCostEstimate(ctx, &awslightsail.GetCostEstimateInput{
		ResourceName: aws.String("sdk-stage22-instance"),
		StartTime:    aws.Time(now.Add(-time.Hour)),
		EndTime:      aws.Time(now),
	})
	if err != nil {
		t.Fatalf("get cost estimate: %v", err)
	}
	if len(costOut.ResourcesBudgetEstimate) == 0 {
		t.Fatalf("expected resources budget estimate")
	}

	isPeeredOut, err := client.IsVpcPeered(ctx, &awslightsail.IsVpcPeeredInput{})
	if err != nil {
		t.Fatalf("is vpc peered: %v", err)
	}
	if aws.ToBool(isPeeredOut.IsPeered) {
		t.Fatalf("expected VPC to start unpeered")
	}

	peerOut, err := client.PeerVpc(ctx, &awslightsail.PeerVpcInput{})
	if err != nil {
		t.Fatalf("peer vpc: %v", err)
	}
	if peerOut.Operation == nil || peerOut.Operation.OperationType == "" {
		t.Fatalf("expected peer operation")
	}

	isPeeredOut, err = client.IsVpcPeered(ctx, &awslightsail.IsVpcPeeredInput{})
	if err != nil {
		t.Fatalf("is vpc peered after peer: %v", err)
	}
	if !aws.ToBool(isPeeredOut.IsPeered) {
		t.Fatalf("expected VPC to be peered")
	}

	if _, err := client.UnpeerVpc(ctx, &awslightsail.UnpeerVpcInput{}); err != nil {
		t.Fatalf("unpeer vpc: %v", err)
	}

	cfOut, err := client.CreateCloudFormationStack(ctx, &awslightsail.CreateCloudFormationStackInput{
		Instances: []awslightsailtypes.InstanceEntry{
			{
				AvailabilityZone: aws.String("us-east-1a"),
				InstanceType:     aws.String("t3.micro"),
				PortInfoSource:   awslightsailtypes.PortInfoSourceTypeDefault,
				SourceName:       exportRecordsOut.ExportSnapshotRecords[0].Name,
			},
		},
	})
	if err != nil {
		t.Fatalf("create cloudformation stack: %v", err)
	}
	if len(cfOut.Operations) != 1 {
		t.Fatalf("expected cloudformation operation")
	}

	stackRecordsOut, err := client.GetCloudFormationStackRecords(ctx, &awslightsail.GetCloudFormationStackRecordsInput{})
	if err != nil {
		t.Fatalf("get cloudformation stack records: %v", err)
	}
	if len(stackRecordsOut.CloudFormationStackRecords) == 0 {
		t.Fatalf("expected cloudformation stack records")
	}

	guiOut, err := client.CreateGUISessionAccessDetails(ctx, &awslightsail.CreateGUISessionAccessDetailsInput{ResourceName: aws.String("sdk-stage22-instance")})
	if err != nil {
		t.Fatalf("create gui session access details: %v", err)
	}
	if guiOut.ResourceName == nil || *guiOut.ResourceName != "sdk-stage22-instance" || len(guiOut.Sessions) == 0 {
		t.Fatalf("unexpected gui session access details")
	}

	startGUIOut, err := client.StartGUISession(ctx, &awslightsail.StartGUISessionInput{ResourceName: aws.String("sdk-stage22-instance")})
	if err != nil {
		t.Fatalf("start gui session: %v", err)
	}
	if len(startGUIOut.Operations) != 1 {
		t.Fatalf("expected start GUI operation")
	}

	stopGUIOut, err := client.StopGUISession(ctx, &awslightsail.StopGUISessionInput{ResourceName: aws.String("sdk-stage22-instance")})
	if err != nil {
		t.Fatalf("stop gui session: %v", err)
	}
	if len(stopGUIOut.Operations) != 1 {
		t.Fatalf("expected stop GUI operation")
	}

	metricOut, err := client.GetInstanceMetricData(ctx, &awslightsail.GetInstanceMetricDataInput{
		InstanceName: aws.String("sdk-stage22-instance"),
		MetricName:   awslightsailtypes.InstanceMetricNameCPUUtilization,
		StartTime:    aws.Time(now.Add(-30 * time.Minute)),
		EndTime:      aws.Time(now),
		Period:       aws.Int32(300),
		Statistics:   []awslightsailtypes.MetricStatistic{awslightsailtypes.MetricStatisticAverage},
		Unit:         awslightsailtypes.MetricUnitPercent,
	})
	if err != nil {
		t.Fatalf("get instance metric data: %v", err)
	}
	if metricOut.MetricName != awslightsailtypes.InstanceMetricNameCPUUtilization || len(metricOut.MetricData) == 0 {
		t.Fatalf("unexpected instance metric output")
	}
}

func timeString(ts time.Time) string {
	return strconv.FormatInt(ts.Unix(), 10)
}
