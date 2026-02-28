package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage16SDKLifecycle(t *testing.T) {
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

	createOut, err := client.CreatePlacementGroup(ctx, &awsec2.CreatePlacementGroupInput{
		GroupName:      aws.String("stage16-group"),
		Strategy:       awsec2types.PlacementStrategyPartition,
		PartitionCount: aws.Int32(3),
		SpreadLevel:    awsec2types.SpreadLevelRack,
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypePlacementGroup,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage16")}},
			},
		},
	})
	if err != nil || createOut.PlacementGroup == nil || aws.ToString(createOut.PlacementGroup.GroupName) != "stage16-group" || createOut.PlacementGroup.Strategy != awsec2types.PlacementStrategyPartition {
		t.Fatalf("create placement group: %v", err)
	}

	describeOut, err := client.DescribePlacementGroups(ctx, &awsec2.DescribePlacementGroupsInput{
		GroupNames: []string{"stage16-group"},
	})
	if err != nil || len(describeOut.PlacementGroups) != 1 || aws.ToString(describeOut.PlacementGroups[0].GroupName) != "stage16-group" || describeOut.PlacementGroups[0].Strategy != awsec2types.PlacementStrategyPartition {
		t.Fatalf("describe placement groups: %v", err)
	}

	if _, err := client.DeletePlacementGroup(ctx, &awsec2.DeletePlacementGroupInput{
		GroupName: aws.String("stage16-group"),
	}); err != nil {
		t.Fatalf("delete placement group: %v", err)
	}

	describeAfterDeleteOut, err := client.DescribePlacementGroups(ctx, &awsec2.DescribePlacementGroupsInput{
		GroupNames: []string{"stage16-group"},
	})
	if err != nil {
		t.Fatalf("describe placement groups after delete: %v", err)
	}
	if len(describeAfterDeleteOut.PlacementGroups) != 0 {
		t.Fatalf("expected no placement groups after delete")
	}
}

func TestEC2Stage16ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreatePlacementGroup",
		"DescribePlacementGroups",
		"DeletePlacementGroup",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreatePlacementGroup":
			params["GroupName"] = "stage16-group"
			params["Strategy"] = "partition"
			params["PartitionCount"] = "3"
			params["SpreadLevel"] = "rack"
		case "DescribePlacementGroups":
			params["GroupName.1"] = "stage16-group"
		case "DeletePlacementGroup":
			params["GroupName"] = "stage16-group"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
