package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage4SDKLifecycle(t *testing.T) {
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

	allocateOut, err := client.AllocateAddress(ctx, &awsec2.AllocateAddressInput{
		Domain: awsec2types.DomainTypeVpc,
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeElasticIp,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("stage4")}},
			},
		},
	})
	if err != nil || allocateOut.AllocationId == nil {
		t.Fatalf("allocate address: %v", err)
	}
	allocationID := aws.ToString(allocateOut.AllocationId)

	runOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage4"),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		InstanceType: awsec2types.InstanceTypeT3Micro,
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	associateOut, err := client.AssociateAddress(ctx, &awsec2.AssociateAddressInput{
		AllocationId: aws.String(allocationID),
		InstanceId:   aws.String(instanceID),
	})
	if err != nil || associateOut.AssociationId == nil {
		t.Fatalf("associate address: %v", err)
	}
	associationID := aws.ToString(associateOut.AssociationId)

	describeOut, err := client.DescribeAddresses(ctx, &awsec2.DescribeAddressesInput{
		AllocationIds: []string{allocationID},
	})
	if err != nil || len(describeOut.Addresses) != 1 {
		t.Fatalf("describe addresses: %v", err)
	}
	if describeOut.Addresses[0].AssociationId == nil || aws.ToString(describeOut.Addresses[0].AssociationId) != associationID {
		t.Fatalf("expected association id in described address")
	}

	if _, err := client.DisassociateAddress(ctx, &awsec2.DisassociateAddressInput{
		AssociationId: aws.String(associationID),
	}); err != nil {
		t.Fatalf("disassociate address: %v", err)
	}
	if _, err := client.ReleaseAddress(ctx, &awsec2.ReleaseAddressInput{
		AllocationId: aws.String(allocationID),
	}); err != nil {
		t.Fatalf("release address: %v", err)
	}

	createDefaultVpcOut, err := client.CreateDefaultVpc(ctx, &awsec2.CreateDefaultVpcInput{})
	if err != nil || createDefaultVpcOut.Vpc == nil || createDefaultVpcOut.Vpc.VpcId == nil {
		t.Fatalf("create default vpc: %v", err)
	}

	createDefaultSubnetOut, err := client.CreateDefaultSubnet(ctx, &awsec2.CreateDefaultSubnetInput{
		AvailabilityZone: aws.String("us-east-1b"),
	})
	if err != nil || createDefaultSubnetOut.Subnet == nil || createDefaultSubnetOut.Subnet.SubnetId == nil {
		t.Fatalf("create default subnet: %v", err)
	}
}

func TestEC2Stage4ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AllocateAddress",
		"DescribeAddresses",
		"AssociateAddress",
		"DisassociateAddress",
		"ReleaseAddress",
		"CreateDefaultVpc",
		"CreateDefaultSubnet",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AllocateAddress":
			params["Domain"] = "vpc"
		case "DescribeAddresses":
			params["AllocationId.1"] = "eipalloc-00000001"
		case "AssociateAddress":
			params["AllocationId"] = "eipalloc-00000001"
			params["InstanceId"] = "i-00000001"
			params["AllowReassociation"] = "true"
		case "DisassociateAddress":
			params["AssociationId"] = "eipassoc-00000001"
		case "ReleaseAddress":
			params["AllocationId"] = "eipalloc-0000000" + strconv.Itoa(idx+1)
		case "CreateDefaultSubnet":
			params["AvailabilityZone"] = "us-east-1b"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
