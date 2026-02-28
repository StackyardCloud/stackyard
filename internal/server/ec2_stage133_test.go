package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage133SDKLifecycle(t *testing.T) {
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
	client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-00000000133"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	createVolumeOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(8),
	})
	if err != nil || createVolumeOut.VolumeId == nil {
		t.Fatalf("create volume: %v", err)
	}
	createSnapshotOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{
		VolumeId: createVolumeOut.VolumeId,
	})
	if err != nil || createSnapshotOut.SnapshotId == nil {
		t.Fatalf("create snapshot: %v", err)
	}
	snapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	if _, err := client.LockSnapshot(ctx, &awsec2.LockSnapshotInput{
		SnapshotId: aws.String(snapshotID),
		LockMode:   awsec2types.LockModeGovernance,
	}); err != nil {
		t.Fatalf("lock snapshot: %v", err)
	}

	restoreSnapshotFromRecycleBinOut, err := client.RestoreSnapshotFromRecycleBin(ctx, &awsec2.RestoreSnapshotFromRecycleBinInput{
		SnapshotId: aws.String(snapshotID),
	})
	if err != nil {
		t.Fatalf("restore snapshot from recycle bin: %v", err)
	}
	if aws.ToString(restoreSnapshotFromRecycleBinOut.SnapshotId) != snapshotID {
		t.Fatalf("unexpected restore snapshot from recycle bin output: %#v", restoreSnapshotFromRecycleBinOut)
	}

	if _, err := client.ModifySnapshotTier(ctx, &awsec2.ModifySnapshotTierInput{
		SnapshotId:  aws.String(snapshotID),
		StorageTier: awsec2types.TargetStorageTierArchive,
	}); err != nil {
		t.Fatalf("modify snapshot tier: %v", err)
	}
	restoreSnapshotTierOut, err := client.RestoreSnapshotTier(ctx, &awsec2.RestoreSnapshotTierInput{
		SnapshotId:           aws.String(snapshotID),
		TemporaryRestoreDays: aws.Int32(3),
		PermanentRestore:     aws.Bool(false),
	})
	if err != nil {
		t.Fatalf("restore snapshot tier: %v", err)
	}
	if aws.ToString(restoreSnapshotTierOut.SnapshotId) != snapshotID || aws.ToInt32(restoreSnapshotTierOut.RestoreDuration) != 3 {
		t.Fatalf("unexpected restore snapshot tier output: %#v", restoreSnapshotTierOut)
	}

	unlockSnapshotOut, err := client.UnlockSnapshot(ctx, &awsec2.UnlockSnapshotInput{
		SnapshotId: aws.String(snapshotID),
	})
	if err != nil {
		t.Fatalf("unlock snapshot: %v", err)
	}
	if aws.ToString(unlockSnapshotOut.SnapshotId) != snapshotID {
		t.Fatalf("unexpected unlock snapshot output: %#v", unlockSnapshotOut)
	}

	runScheduledInstancesOut, err := client.RunScheduledInstances(ctx, &awsec2.RunScheduledInstancesInput{
		ScheduledInstanceId: aws.String("sci-stage133"),
		LaunchSpecification: &awsec2types.ScheduledInstancesLaunchSpecification{
			ImageId:      aws.String("ami-stage133-scheduled"),
			InstanceType: aws.String("t3.micro"),
		},
		InstanceCount: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("run scheduled instances: %v", err)
	}
	if len(runScheduledInstancesOut.InstanceIdSet) == 0 || !strings.HasPrefix(runScheduledInstancesOut.InstanceIdSet[0], "i-") {
		t.Fatalf("unexpected run scheduled instances output: %#v", runScheduledInstancesOut.InstanceIdSet)
	}

	createLocalGatewayRouteTableOut, err := client.CreateLocalGatewayRouteTable(ctx, &awsec2.CreateLocalGatewayRouteTableInput{
		LocalGatewayId: aws.String("lgw-00000000133"),
		Mode:           awsec2types.LocalGatewayRouteTableMode("direct-vpc-routing"),
	})
	if err != nil || createLocalGatewayRouteTableOut.LocalGatewayRouteTable == nil || createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId == nil {
		t.Fatalf("create local gateway route table: %v", err)
	}
	localGatewayRouteTableID := aws.ToString(createLocalGatewayRouteTableOut.LocalGatewayRouteTable.LocalGatewayRouteTableId)

	if _, err := client.CreateLocalGatewayRoute(ctx, &awsec2.CreateLocalGatewayRouteInput{
		LocalGatewayRouteTableId: aws.String(localGatewayRouteTableID),
		DestinationCidrBlock:     aws.String("10.133.0.0/16"),
		NetworkInterfaceId:       aws.String("eni-00000000133"),
	}); err != nil {
		t.Fatalf("create local gateway route: %v", err)
	}

	searchLocalGatewayRoutesOut, err := client.SearchLocalGatewayRoutes(ctx, &awsec2.SearchLocalGatewayRoutesInput{
		LocalGatewayRouteTableId: aws.String(localGatewayRouteTableID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("route-search.exact-match"), Values: []string{"10.133.0.0/16"}},
		},
	})
	if err != nil {
		t.Fatalf("search local gateway routes: %v", err)
	}
	if len(searchLocalGatewayRoutesOut.Routes) == 0 {
		t.Fatalf("expected local gateway routes in search output")
	}

	if _, err := client.SendDiagnosticInterrupt(ctx, &awsec2.SendDiagnosticInterruptInput{
		InstanceId: aws.String(instanceID),
	}); err != nil {
		t.Fatalf("send diagnostic interrupt: %v", err)
	}

	createNetworkInsightsAccessScopeOut, err := client.CreateNetworkInsightsAccessScope(ctx, &awsec2.CreateNetworkInsightsAccessScopeInput{
		ClientToken: aws.String("stage133-access-scope"),
	})
	if err != nil || createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope == nil || createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId == nil {
		t.Fatalf("create network insights access scope: %v", err)
	}
	networkInsightsAccessScopeID := aws.ToString(createNetworkInsightsAccessScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId)

	startNetworkInsightsAccessScopeAnalysisOut, err := client.StartNetworkInsightsAccessScopeAnalysis(ctx, &awsec2.StartNetworkInsightsAccessScopeAnalysisInput{
		ClientToken:                  aws.String("stage133-start-access-scope-analysis"),
		NetworkInsightsAccessScopeId: aws.String(networkInsightsAccessScopeID),
	})
	if err != nil {
		t.Fatalf("start network insights access scope analysis: %v", err)
	}
	if startNetworkInsightsAccessScopeAnalysisOut.NetworkInsightsAccessScopeAnalysis == nil ||
		!strings.HasPrefix(aws.ToString(startNetworkInsightsAccessScopeAnalysisOut.NetworkInsightsAccessScopeAnalysis.NetworkInsightsAccessScopeAnalysisId), "niasa-") {
		t.Fatalf("unexpected start network insights access scope analysis output: %#v", startNetworkInsightsAccessScopeAnalysisOut.NetworkInsightsAccessScopeAnalysis)
	}

	createNetworkInsightsPathOut, err := client.CreateNetworkInsightsPath(ctx, &awsec2.CreateNetworkInsightsPathInput{
		ClientToken: aws.String("stage133-network-insights-path"),
		Protocol:    awsec2types.ProtocolTcp,
		Source:      aws.String(instanceID),
		Destination: aws.String("eni-000000001331"),
	})
	if err != nil || createNetworkInsightsPathOut.NetworkInsightsPath == nil || createNetworkInsightsPathOut.NetworkInsightsPath.NetworkInsightsPathId == nil {
		t.Fatalf("create network insights path: %v", err)
	}
	networkInsightsPathID := aws.ToString(createNetworkInsightsPathOut.NetworkInsightsPath.NetworkInsightsPathId)

	startNetworkInsightsAnalysisOut, err := client.StartNetworkInsightsAnalysis(ctx, &awsec2.StartNetworkInsightsAnalysisInput{
		ClientToken:           aws.String("stage133-start-network-insights-analysis"),
		NetworkInsightsPathId: aws.String(networkInsightsPathID),
	})
	if err != nil {
		t.Fatalf("start network insights analysis: %v", err)
	}
	if startNetworkInsightsAnalysisOut.NetworkInsightsAnalysis == nil ||
		!strings.HasPrefix(aws.ToString(startNetworkInsightsAnalysisOut.NetworkInsightsAnalysis.NetworkInsightsAnalysisId), "nia-") {
		t.Fatalf("unexpected start network insights analysis output: %#v", startNetworkInsightsAnalysisOut.NetworkInsightsAnalysis)
	}

	if _, err := client.ProvisionByoipCidr(ctx, &awsec2.ProvisionByoipCidrInput{
		Cidr: aws.String("198.51.133.0/24"),
	}); err != nil {
		t.Fatalf("provision byoip cidr: %v", err)
	}

	withdrawByoipCidrOut, err := client.WithdrawByoipCidr(ctx, &awsec2.WithdrawByoipCidrInput{
		Cidr: aws.String("198.51.133.0/24"),
	})
	if err != nil {
		t.Fatalf("withdraw byoip cidr: %v", err)
	}
	if withdrawByoipCidrOut.ByoipCidr == nil ||
		aws.ToString(withdrawByoipCidrOut.ByoipCidr.Cidr) != "198.51.133.0/24" ||
		string(withdrawByoipCidrOut.ByoipCidr.State) != "withdrawn" {
		t.Fatalf("unexpected withdraw byoip cidr output: %#v", withdrawByoipCidrOut.ByoipCidr)
	}
}

func TestEC2Stage133ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"RestoreSnapshotFromRecycleBin",
		"RestoreSnapshotTier",
		"RunScheduledInstances",
		"SearchLocalGatewayRoutes",
		"SendDiagnosticInterrupt",
		"StartNetworkInsightsAccessScopeAnalysis",
		"StartNetworkInsightsAnalysis",
		"UnlockSnapshot",
		"WithdrawByoipCidr",
	}

	paramsByAction := map[string]map[string]string{
		"RestoreSnapshotFromRecycleBin": {
			"SnapshotId": "snap-00000000133",
		},
		"RestoreSnapshotTier": {
			"SnapshotId": "snap-00000000133",
		},
		"RunScheduledInstances": {
			"ScheduledInstanceId":              "sci-00000000133",
			"LaunchSpecification.ImageId":      "ami-00000000133",
			"LaunchSpecification.InstanceType": "t3.micro",
		},
		"SearchLocalGatewayRoutes": {
			"LocalGatewayRouteTableId": "lgw-rtb-00000000133",
		},
		"SendDiagnosticInterrupt": {
			"InstanceId": "i-00000000133",
		},
		"StartNetworkInsightsAccessScopeAnalysis": {
			"ClientToken":                  "stage133-start-scope",
			"NetworkInsightsAccessScopeId": "nias-00000000133",
		},
		"StartNetworkInsightsAnalysis": {
			"ClientToken":           "stage133-start-analysis",
			"NetworkInsightsPathId": "nip-00000000133",
		},
		"UnlockSnapshot": {
			"SnapshotId": "snap-00000000133",
		},
		"WithdrawByoipCidr": {
			"Cidr": "198.51.133.0/24",
		},
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, paramsByAction[action])
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
