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

func TestEC2Stage126SDKLifecycle(t *testing.T) {
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
		PrefixListName: aws.String("stage126-prefix-list"),
	})
	if err != nil || createManagedPrefixListOut.PrefixList == nil || createManagedPrefixListOut.PrefixList.PrefixListId == nil {
		t.Fatalf("create managed prefix list: %v", err)
	}
	prefixListID := aws.ToString(createManagedPrefixListOut.PrefixList.PrefixListId)

	getManagedPrefixListEntriesOut, err := client.GetManagedPrefixListEntries(ctx, &awsec2.GetManagedPrefixListEntriesInput{
		PrefixListId: aws.String(prefixListID),
		MaxResults:   aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get managed prefix list entries: %v", err)
	}
	if len(getManagedPrefixListEntriesOut.Entries) == 0 || aws.ToString(getManagedPrefixListEntriesOut.Entries[0].Cidr) == "" {
		t.Fatalf("expected managed prefix list entries")
	}

	createScopeOut, err := client.CreateNetworkInsightsAccessScope(ctx, &awsec2.CreateNetworkInsightsAccessScopeInput{
		ClientToken: aws.String("stage126-nias-token"),
	})
	if err != nil || createScopeOut.NetworkInsightsAccessScope == nil || createScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId == nil {
		t.Fatalf("create network insights access scope: %v", err)
	}
	scopeID := aws.ToString(createScopeOut.NetworkInsightsAccessScope.NetworkInsightsAccessScopeId)

	describeNetworkInsightsAccessScopeAnalysesOut, err := client.DescribeNetworkInsightsAccessScopeAnalyses(ctx, &awsec2.DescribeNetworkInsightsAccessScopeAnalysesInput{
		NetworkInsightsAccessScopeId: aws.String(scopeID),
		MaxResults:                   aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe network insights access scope analyses: %v", err)
	}
	if len(describeNetworkInsightsAccessScopeAnalysesOut.NetworkInsightsAccessScopeAnalyses) == 0 ||
		describeNetworkInsightsAccessScopeAnalysesOut.NetworkInsightsAccessScopeAnalyses[0].NetworkInsightsAccessScopeAnalysisId == nil {
		t.Fatalf("expected network insights access scope analyses")
	}
	analysisID := aws.ToString(describeNetworkInsightsAccessScopeAnalysesOut.NetworkInsightsAccessScopeAnalyses[0].NetworkInsightsAccessScopeAnalysisId)

	getNetworkInsightsAccessScopeAnalysisFindingsOut, err := client.GetNetworkInsightsAccessScopeAnalysisFindings(ctx, &awsec2.GetNetworkInsightsAccessScopeAnalysisFindingsInput{
		NetworkInsightsAccessScopeAnalysisId: aws.String(analysisID),
		MaxResults:                           aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get network insights access scope analysis findings: %v", err)
	}
	if len(getNetworkInsightsAccessScopeAnalysisFindingsOut.AnalysisFindings) == 0 {
		t.Fatalf("expected network insights access scope analysis findings")
	}
	if aws.ToString(getNetworkInsightsAccessScopeAnalysisFindingsOut.AnalysisFindings[0].NetworkInsightsAccessScopeId) != scopeID {
		t.Fatalf("unexpected network insights access scope id in findings")
	}

	getNetworkInsightsAccessScopeContentOut, err := client.GetNetworkInsightsAccessScopeContent(ctx, &awsec2.GetNetworkInsightsAccessScopeContentInput{
		NetworkInsightsAccessScopeId: aws.String(scopeID),
	})
	if err != nil {
		t.Fatalf("get network insights access scope content: %v", err)
	}
	if getNetworkInsightsAccessScopeContentOut.NetworkInsightsAccessScopeContent == nil ||
		aws.ToString(getNetworkInsightsAccessScopeContentOut.NetworkInsightsAccessScopeContent.NetworkInsightsAccessScopeId) != scopeID {
		t.Fatalf("unexpected network insights access scope content: %#v", getNetworkInsightsAccessScopeContentOut.NetworkInsightsAccessScopeContent)
	}

	getReservedInstancesExchangeQuoteOut, err := client.GetReservedInstancesExchangeQuote(ctx, &awsec2.GetReservedInstancesExchangeQuoteInput{
		ReservedInstanceIds: []string{"ri-stage126"},
	})
	if err != nil {
		t.Fatalf("get reserved instances exchange quote: %v", err)
	}
	if getReservedInstancesExchangeQuoteOut.IsValidExchange == nil || !aws.ToBool(getReservedInstancesExchangeQuoteOut.IsValidExchange) {
		t.Fatalf("expected valid reserved instances exchange quote")
	}

	getSpotPlacementScoresOut, err := client.GetSpotPlacementScores(ctx, &awsec2.GetSpotPlacementScoresInput{
		TargetCapacity:         aws.Int32(10),
		InstanceTypes:          []string{"t3.micro"},
		RegionNames:            []string{testRegion},
		SingleAvailabilityZone: aws.Bool(true),
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get spot placement scores: %v", err)
	}
	if len(getSpotPlacementScoresOut.SpotPlacementScores) == 0 || getSpotPlacementScoresOut.SpotPlacementScores[0].Score == nil {
		t.Fatalf("expected spot placement scores")
	}

	importImageOut, err := client.ImportImage(ctx, &awsec2.ImportImageInput{
		Architecture: aws.String("x86_64"),
		Description:  aws.String("stage126 import image"),
		Platform:     aws.String("Linux"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeImportImageTask,
				Tags: []awsec2types.Tag{
					{Key: aws.String("stage"), Value: aws.String("126")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("import image: %v", err)
	}
	if !strings.HasPrefix(aws.ToString(importImageOut.ImportTaskId), "import-ami-") || !strings.HasPrefix(aws.ToString(importImageOut.ImageId), "ami-") {
		t.Fatalf("unexpected import image output: %#v", importImageOut)
	}

	importInstanceOut, err := client.ImportInstance(ctx, &awsec2.ImportInstanceInput{
		Description: aws.String("stage126 import instance"),
		Platform:    awsec2types.PlatformValuesWindows,
	})
	if err != nil {
		t.Fatalf("import instance: %v", err)
	}
	if importInstanceOut.ConversionTask == nil || !strings.HasPrefix(aws.ToString(importInstanceOut.ConversionTask.ConversionTaskId), "import-i-") {
		t.Fatalf("unexpected import instance output: %#v", importInstanceOut.ConversionTask)
	}

	importSnapshotOut, err := client.ImportSnapshot(ctx, &awsec2.ImportSnapshotInput{
		Description: aws.String("stage126 import snapshot"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeImportSnapshotTask,
				Tags: []awsec2types.Tag{
					{Key: aws.String("stage"), Value: aws.String("126")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("import snapshot: %v", err)
	}
	if !strings.HasPrefix(aws.ToString(importSnapshotOut.ImportTaskId), "import-snap-") ||
		importSnapshotOut.SnapshotTaskDetail == nil || !strings.HasPrefix(aws.ToString(importSnapshotOut.SnapshotTaskDetail.SnapshotId), "snap-") {
		t.Fatalf("unexpected import snapshot output: %#v", importSnapshotOut)
	}

	importVolumeOut, err := client.ImportVolume(ctx, &awsec2.ImportVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Description:      aws.String("stage126 import volume"),
		Image: &awsec2types.DiskImageDetail{
			Format:            awsec2types.DiskImageFormatRaw,
			Bytes:             aws.Int64(1024),
			ImportManifestUrl: aws.String("https://example.com/stage126.manifest"),
		},
		Volume: &awsec2types.VolumeDetail{Size: aws.Int64(8)},
	})
	if err != nil {
		t.Fatalf("import volume: %v", err)
	}
	if importVolumeOut.ConversionTask == nil || !strings.HasPrefix(aws.ToString(importVolumeOut.ConversionTask.ConversionTaskId), "import-vol-") {
		t.Fatalf("unexpected import volume output: %#v", importVolumeOut.ConversionTask)
	}

	listImagesInRecycleBinOut, err := client.ListImagesInRecycleBin(ctx, &awsec2.ListImagesInRecycleBinInput{
		ImageIds:   []string{aws.ToString(importImageOut.ImageId)},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("list images in recycle bin: %v", err)
	}
	if len(listImagesInRecycleBinOut.Images) == 0 || aws.ToString(listImagesInRecycleBinOut.Images[0].ImageId) != aws.ToString(importImageOut.ImageId) {
		t.Fatalf("unexpected list images in recycle bin output: %#v", listImagesInRecycleBinOut.Images)
	}
}

func TestEC2Stage126ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"GetManagedPrefixListEntries",
		"GetNetworkInsightsAccessScopeAnalysisFindings",
		"GetNetworkInsightsAccessScopeContent",
		"GetReservedInstancesExchangeQuote",
		"GetSpotPlacementScores",
		"ImportImage",
		"ImportInstance",
		"ImportSnapshot",
		"ImportVolume",
		"ListImagesInRecycleBin",
	}

	paramsByAction := map[string]map[string]string{
		"GetManagedPrefixListEntries": {
			"PrefixListId": "pl-0000000126",
		},
		"GetNetworkInsightsAccessScopeAnalysisFindings": {
			"NetworkInsightsAccessScopeAnalysisId": "niasa-0000000126",
		},
		"GetNetworkInsightsAccessScopeContent": {
			"NetworkInsightsAccessScopeId": "nias-0000000126",
		},
		"GetReservedInstancesExchangeQuote": {
			"ReservedInstanceId.1": "ri-0000000126",
		},
		"GetSpotPlacementScores": {
			"TargetCapacity": "10",
		},
		"ImportImage": {},
		"ImportInstance": {
			"Platform": "Windows",
		},
		"ImportSnapshot": {},
		"ImportVolume": {
			"AvailabilityZone": "us-east-1a",
		},
		"ListImagesInRecycleBin": {},
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
