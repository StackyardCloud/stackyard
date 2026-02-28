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

func TestEC2Stage120SDKLifecycle(t *testing.T) {
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

	createManagedPrefixListOut, err := client.CreateManagedPrefixList(ctx, &awsec2.CreateManagedPrefixListInput{
		AddressFamily:  aws.String("ipv4"),
		MaxEntries:     aws.Int32(10),
		PrefixListName: aws.String("stage120-prefix-list"),
	})
	if err != nil {
		t.Fatalf("create managed prefix list: %v", err)
	}
	if createManagedPrefixListOut.PrefixList == nil || createManagedPrefixListOut.PrefixList.PrefixListId == nil {
		t.Fatalf("expected created managed prefix list")
	}
	prefixListID := aws.ToString(createManagedPrefixListOut.PrefixList.PrefixListId)

	createScopeOut, err := client.CreateNetworkInsightsAccessScope(ctx, &awsec2.CreateNetworkInsightsAccessScopeInput{
		ClientToken: aws.String("stage120-nias-token"),
	})
	if err != nil {
		t.Fatalf("create network insights access scope: %v", err)
	}
	if createScopeOut.NetworkInsightsAccessScope == nil || createScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId == nil {
		t.Fatalf("expected created network insights access scope")
	}
	scopeID := aws.ToString(createScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId)

	runInstancesOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage120"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	createPathOut, err := client.CreateNetworkInsightsPath(ctx, &awsec2.CreateNetworkInsightsPathInput{
		ClientToken: aws.String("stage120-nip-token"),
		Protocol:    awsec2types.ProtocolTcp,
		Source:      aws.String(instanceID),
		Destination: aws.String("eni-00000000000000120"),
	})
	if err != nil {
		t.Fatalf("create network insights path: %v", err)
	}
	if createPathOut.NetworkInsightsPath == nil || createPathOut.NetworkInsightsPath.NetworkInsightsPathId == nil {
		t.Fatalf("expected created network insights path")
	}
	pathID := aws.ToString(createPathOut.NetworkInsightsPath.NetworkInsightsPathId)

	localGatewayID := "lgw-00000000120"
	createGroupOut, err := client.CreateLocalGatewayVirtualInterfaceGroup(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceGroupInput{
		LocalGatewayId: aws.String(localGatewayID),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface group: %v", err)
	}
	if createGroupOut.LocalGatewayVirtualInterfaceGroup == nil || createGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId == nil {
		t.Fatalf("expected created local gateway virtual interface group")
	}
	groupID := aws.ToString(createGroupOut.LocalGatewayVirtualInterfaceGroup.LocalGatewayVirtualInterfaceGroupId)

	lagID := "lag-120"
	_, err = client.CreateLocalGatewayVirtualInterface(ctx, &awsec2.CreateLocalGatewayVirtualInterfaceInput{
		LocalAddress:                        aws.String("169.254.120.1"),
		LocalGatewayVirtualInterfaceGroupId: aws.String(groupID),
		OutpostLagId:                        aws.String(lagID),
		PeerAddress:                         aws.String("169.254.120.2"),
		Vlan:                                aws.Int32(120),
	})
	if err != nil {
		t.Fatalf("create local gateway virtual interface: %v", err)
	}

	createPublicIpv4PoolOut, err := client.CreatePublicIpv4Pool(ctx, &awsec2.CreatePublicIpv4PoolInput{})
	if err != nil {
		t.Fatalf("create public ipv4 pool: %v", err)
	}
	if createPublicIpv4PoolOut.PoolId == nil {
		t.Fatalf("expected created public ipv4 pool")
	}
	poolID := aws.ToString(createPublicIpv4PoolOut.PoolId)

	createReplaceRootVolumeTaskOut, err := client.CreateReplaceRootVolumeTask(ctx, &awsec2.CreateReplaceRootVolumeTaskInput{
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		t.Fatalf("create replace root volume task: %v", err)
	}
	if createReplaceRootVolumeTaskOut.ReplaceRootVolumeTask == nil || createReplaceRootVolumeTaskOut.ReplaceRootVolumeTask.ReplaceRootVolumeTaskId == nil {
		t.Fatalf("expected created replace root volume task")
	}
	replaceRootVolumeTaskID := aws.ToString(createReplaceRootVolumeTaskOut.ReplaceRootVolumeTask.ReplaceRootVolumeTaskId)

	createReservedInstancesListingOut, err := client.CreateReservedInstancesListing(ctx, &awsec2.CreateReservedInstancesListingInput{
		ClientToken:         aws.String("stage120-ril-token"),
		InstanceCount:       aws.Int32(1),
		ReservedInstancesId: aws.String("ri-stage120"),
		PriceSchedules: []awsec2types.PriceScheduleSpecification{
			{Price: aws.Float64(1.25), Term: aws.Int64(1), CurrencyCode: awsec2types.CurrencyCodeValuesUsd},
		},
	})
	if err != nil {
		t.Fatalf("create reserved instances listing: %v", err)
	}
	if len(createReservedInstancesListingOut.ReservedInstancesListings) != 1 || createReservedInstancesListingOut.ReservedInstancesListings[0].ReservedInstancesListingId == nil {
		t.Fatalf("expected created reserved instances listing")
	}
	listingID := aws.ToString(createReservedInstancesListingOut.ReservedInstancesListings[0].ReservedInstancesListingId)
	reservedInstancesID := "ri-" + strings.TrimPrefix(listingID, "ril-")

	describeManagedPrefixListsOut, err := client.DescribeManagedPrefixLists(ctx, &awsec2.DescribeManagedPrefixListsInput{
		PrefixListIds: []string{prefixListID},
		MaxResults:    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe managed prefix lists: %v", err)
	}
	if len(describeManagedPrefixListsOut.PrefixLists) != 1 || aws.ToString(describeManagedPrefixListsOut.PrefixLists[0].PrefixListId) != prefixListID {
		t.Fatalf("unexpected describe managed prefix lists output: %#v", describeManagedPrefixListsOut.PrefixLists)
	}

	describeNetworkInsightsAccessScopeAnalysesOut, err := client.DescribeNetworkInsightsAccessScopeAnalyses(ctx, &awsec2.DescribeNetworkInsightsAccessScopeAnalysesInput{
		NetworkInsightsAccessScopeId: aws.String(scopeID),
		MaxResults:                   aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe network insights access scope analyses: %v", err)
	}
	if len(describeNetworkInsightsAccessScopeAnalysesOut.NetworkInsightsAccessScopeAnalyses) != 1 ||
		aws.ToString(describeNetworkInsightsAccessScopeAnalysesOut.NetworkInsightsAccessScopeAnalyses[0].NetworkInsightsAccessScopeId) != scopeID {
		t.Fatalf("unexpected describe network insights access scope analyses output: %#v", describeNetworkInsightsAccessScopeAnalysesOut.NetworkInsightsAccessScopeAnalyses)
	}

	describeNetworkInsightsAccessScopesOut, err := client.DescribeNetworkInsightsAccessScopes(ctx, &awsec2.DescribeNetworkInsightsAccessScopesInput{
		NetworkInsightsAccessScopeIds: []string{scopeID},
		MaxResults:                    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe network insights access scopes: %v", err)
	}
	if len(describeNetworkInsightsAccessScopesOut.NetworkInsightsAccessScopes) != 1 ||
		aws.ToString(describeNetworkInsightsAccessScopesOut.NetworkInsightsAccessScopes[0].NetworkInsightsAccessScopeId) != scopeID {
		t.Fatalf("unexpected describe network insights access scopes output: %#v", describeNetworkInsightsAccessScopesOut.NetworkInsightsAccessScopes)
	}

	describeNetworkInsightsAnalysesOut, err := client.DescribeNetworkInsightsAnalyses(ctx, &awsec2.DescribeNetworkInsightsAnalysesInput{
		NetworkInsightsPathId: aws.String(pathID),
		MaxResults:            aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe network insights analyses: %v", err)
	}
	if len(describeNetworkInsightsAnalysesOut.NetworkInsightsAnalyses) != 1 ||
		aws.ToString(describeNetworkInsightsAnalysesOut.NetworkInsightsAnalyses[0].NetworkInsightsPathId) != pathID {
		t.Fatalf("unexpected describe network insights analyses output: %#v", describeNetworkInsightsAnalysesOut.NetworkInsightsAnalyses)
	}

	describeNetworkInsightsPathsOut, err := client.DescribeNetworkInsightsPaths(ctx, &awsec2.DescribeNetworkInsightsPathsInput{
		NetworkInsightsPathIds: []string{pathID},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe network insights paths: %v", err)
	}
	if len(describeNetworkInsightsPathsOut.NetworkInsightsPaths) != 1 ||
		aws.ToString(describeNetworkInsightsPathsOut.NetworkInsightsPaths[0].NetworkInsightsPathId) != pathID {
		t.Fatalf("unexpected describe network insights paths output: %#v", describeNetworkInsightsPathsOut.NetworkInsightsPaths)
	}

	describeOutpostLagsOut, err := client.DescribeOutpostLags(ctx, &awsec2.DescribeOutpostLagsInput{
		OutpostLagIds: []string{lagID},
		MaxResults:    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe outpost lags: %v", err)
	}
	if len(describeOutpostLagsOut.OutpostLags) != 1 || aws.ToString(describeOutpostLagsOut.OutpostLags[0].OutpostLagId) != lagID {
		t.Fatalf("unexpected describe outpost lags output: %#v", describeOutpostLagsOut.OutpostLags)
	}

	describePrefixListsOut, err := client.DescribePrefixLists(ctx, &awsec2.DescribePrefixListsInput{
		PrefixListIds: []string{prefixListID},
		MaxResults:    aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe prefix lists: %v", err)
	}
	if len(describePrefixListsOut.PrefixLists) != 1 || aws.ToString(describePrefixListsOut.PrefixLists[0].PrefixListId) != prefixListID {
		t.Fatalf("unexpected describe prefix lists output: %#v", describePrefixListsOut.PrefixLists)
	}

	describePublicIpv4PoolsOut, err := client.DescribePublicIpv4Pools(ctx, &awsec2.DescribePublicIpv4PoolsInput{
		PoolIds:    []string{poolID},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe public ipv4 pools: %v", err)
	}
	if len(describePublicIpv4PoolsOut.PublicIpv4Pools) != 1 || aws.ToString(describePublicIpv4PoolsOut.PublicIpv4Pools[0].PoolId) != poolID {
		t.Fatalf("unexpected describe public ipv4 pools output: %#v", describePublicIpv4PoolsOut.PublicIpv4Pools)
	}

	describeReplaceRootVolumeTasksOut, err := client.DescribeReplaceRootVolumeTasks(ctx, &awsec2.DescribeReplaceRootVolumeTasksInput{
		ReplaceRootVolumeTaskIds: []string{replaceRootVolumeTaskID},
		MaxResults:               aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe replace root volume tasks: %v", err)
	}
	if len(describeReplaceRootVolumeTasksOut.ReplaceRootVolumeTasks) != 1 ||
		aws.ToString(describeReplaceRootVolumeTasksOut.ReplaceRootVolumeTasks[0].ReplaceRootVolumeTaskId) != replaceRootVolumeTaskID {
		t.Fatalf("unexpected describe replace root volume tasks output: %#v", describeReplaceRootVolumeTasksOut.ReplaceRootVolumeTasks)
	}

	describeReservedInstancesOut, err := client.DescribeReservedInstances(ctx, &awsec2.DescribeReservedInstancesInput{
		ReservedInstancesIds: []string{reservedInstancesID},
	})
	if err != nil {
		t.Fatalf("describe reserved instances: %v", err)
	}
	if len(describeReservedInstancesOut.ReservedInstances) != 1 ||
		aws.ToString(describeReservedInstancesOut.ReservedInstances[0].ReservedInstancesId) != reservedInstancesID {
		t.Fatalf("unexpected describe reserved instances output: %#v", describeReservedInstancesOut.ReservedInstances)
	}
}

func TestEC2Stage120ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeManagedPrefixLists",
		"DescribeNetworkInsightsAccessScopeAnalyses",
		"DescribeNetworkInsightsAccessScopes",
		"DescribeNetworkInsightsAnalyses",
		"DescribeNetworkInsightsPaths",
		"DescribeOutpostLags",
		"DescribePrefixLists",
		"DescribePublicIpv4Pools",
		"DescribeReplaceRootVolumeTasks",
		"DescribeReservedInstances",
	}

	paramsByAction := map[string]map[string]string{
		"DescribeManagedPrefixLists": {
			"PrefixListId.1": "pl-00000000120",
			"MaxResults":     "10",
		},
		"DescribeNetworkInsightsAccessScopeAnalyses": {
			"NetworkInsightsAccessScopeAnalysisId.1": "niasa-00000000120",
			"MaxResults":                             "10",
		},
		"DescribeNetworkInsightsAccessScopes": {
			"NetworkInsightsAccessScopeId.1": "nias-00000000120",
			"MaxResults":                     "10",
		},
		"DescribeNetworkInsightsAnalyses": {
			"NetworkInsightsAnalysisId.1": "nia-00000000120",
			"MaxResults":                  "10",
		},
		"DescribeNetworkInsightsPaths": {
			"NetworkInsightsPathId.1": "nip-00000000120",
			"MaxResults":              "10",
		},
		"DescribeOutpostLags": {
			"OutpostLagId.1": "lag-00000000120",
			"MaxResults":     "10",
		},
		"DescribePrefixLists": {
			"PrefixListId.1": "pl-00000000120",
			"MaxResults":     "10",
		},
		"DescribePublicIpv4Pools": {
			"PoolId.1":   "ipv4pool-00000000120",
			"MaxResults": "10",
		},
		"DescribeReplaceRootVolumeTasks": {
			"ReplaceRootVolumeTaskId.1": "replacevol-00000000120",
			"MaxResults":                "10",
		},
		"DescribeReservedInstances": {
			"ReservedInstancesId.1": "ri-00000000120",
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
