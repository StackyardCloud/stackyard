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

func TestEC2Stage114SDKLifecycle(t *testing.T) {
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

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{})
	if err != nil {
		t.Fatalf("create ipam: %v", err)
	}
	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{IpamId: createIpamOut.Ipam.IpamId})
	if err != nil {
		t.Fatalf("create ipam scope: %v", err)
	}
	createIpamPoolOut, err := client.CreateIpamPool(ctx, &awsec2.CreateIpamPoolInput{
		AddressFamily: awsec2types.AddressFamilyIpv4,
		IpamScopeId:   createIpamScopeOut.IpamScope.IpamScopeId,
	})
	if err != nil {
		t.Fatalf("create ipam pool: %v", err)
	}
	allocateIpamPoolCidrOut, err := client.AllocateIpamPoolCidr(ctx, &awsec2.AllocateIpamPoolCidrInput{
		IpamPoolId:    createIpamPoolOut.IpamPool.IpamPoolId,
		NetmaskLength: aws.Int32(24),
	})
	if err != nil {
		t.Fatalf("allocate ipam pool cidr: %v", err)
	}

	if _, err := client.AssociateIpamByoasn(ctx, &awsec2.AssociateIpamByoasnInput{
		Asn:  aws.String("64512"),
		Cidr: aws.String("198.51.114.0/24"),
	}); err != nil {
		t.Fatalf("associate ipam byoasn: %v", err)
	}

	deprovisionIpamByoasnOut, err := client.DeprovisionIpamByoasn(ctx, &awsec2.DeprovisionIpamByoasnInput{
		Asn:    aws.String("64512"),
		IpamId: createIpamOut.Ipam.IpamId,
	})
	if err != nil {
		t.Fatalf("deprovision ipam byoasn: %v", err)
	}
	if deprovisionIpamByoasnOut.Byoasn == nil ||
		aws.ToString(deprovisionIpamByoasnOut.Byoasn.Asn) != "64512" ||
		aws.ToString(deprovisionIpamByoasnOut.Byoasn.IpamId) != aws.ToString(createIpamOut.Ipam.IpamId) ||
		string(deprovisionIpamByoasnOut.Byoasn.State) != "deprovisioned" {
		t.Fatalf("unexpected deprovision ipam byoasn output: %#v", deprovisionIpamByoasnOut.Byoasn)
	}

	deprovisionIpamPoolCidrOut, err := client.DeprovisionIpamPoolCidr(ctx, &awsec2.DeprovisionIpamPoolCidrInput{
		IpamPoolId: createIpamPoolOut.IpamPool.IpamPoolId,
		Cidr:       allocateIpamPoolCidrOut.IpamPoolAllocation.Cidr,
	})
	if err != nil {
		t.Fatalf("deprovision ipam pool cidr: %v", err)
	}
	if deprovisionIpamPoolCidrOut.IpamPoolCidr == nil ||
		aws.ToString(deprovisionIpamPoolCidrOut.IpamPoolCidr.Cidr) != aws.ToString(allocateIpamPoolCidrOut.IpamPoolAllocation.Cidr) ||
		string(deprovisionIpamPoolCidrOut.IpamPoolCidr.State) != "deprovisioned" {
		t.Fatalf("unexpected deprovision ipam pool cidr output: %#v", deprovisionIpamPoolCidrOut.IpamPoolCidr)
	}

	createPublicIpv4PoolOut, err := client.CreatePublicIpv4Pool(ctx, &awsec2.CreatePublicIpv4PoolInput{})
	if err != nil {
		t.Fatalf("create public ipv4 pool: %v", err)
	}
	deprovisionPublicIpv4PoolCidrOut, err := client.DeprovisionPublicIpv4PoolCidr(ctx, &awsec2.DeprovisionPublicIpv4PoolCidrInput{
		PoolId: createPublicIpv4PoolOut.PoolId,
		Cidr:   aws.String("198.51.114.0/24"),
	})
	if err != nil {
		t.Fatalf("deprovision public ipv4 pool cidr: %v", err)
	}
	if aws.ToString(deprovisionPublicIpv4PoolCidrOut.PoolId) != aws.ToString(createPublicIpv4PoolOut.PoolId) ||
		len(deprovisionPublicIpv4PoolCidrOut.DeprovisionedAddresses) != 1 ||
		deprovisionPublicIpv4PoolCidrOut.DeprovisionedAddresses[0] != "198.51.114.0/24" {
		t.Fatalf("unexpected deprovision public ipv4 pool cidr output: %#v", deprovisionPublicIpv4PoolCidrOut)
	}

	deregisterInstanceEventNotificationAttributesOut, err := client.DeregisterInstanceEventNotificationAttributes(ctx, &awsec2.DeregisterInstanceEventNotificationAttributesInput{
		InstanceTagAttribute: &awsec2types.DeregisterInstanceTagAttributeRequest{
			InstanceTagKeys: []string{"env", "team"},
		},
	})
	if err != nil {
		t.Fatalf("deregister instance event notification attributes: %v", err)
	}
	if deregisterInstanceEventNotificationAttributesOut.InstanceTagAttribute == nil ||
		aws.ToBool(deregisterInstanceEventNotificationAttributesOut.InstanceTagAttribute.IncludeAllTagsOfInstance) ||
		len(deregisterInstanceEventNotificationAttributesOut.InstanceTagAttribute.InstanceTagKeys) != 0 {
		t.Fatalf("unexpected deregister instance event notification attributes output: %#v", deregisterInstanceEventNotificationAttributesOut.InstanceTagAttribute)
	}

	bundleInstanceOut, err := client.BundleInstance(ctx, &awsec2.BundleInstanceInput{
		InstanceId: aws.String("i-00000000000000114"),
		Storage: &awsec2types.Storage{S3: &awsec2types.S3Storage{
			AWSAccessKeyId:        aws.String(testAccessKey),
			Bucket:                aws.String("stage114-bucket"),
			Prefix:                aws.String("stage114-prefix"),
			UploadPolicy:          []byte("stage114-policy"),
			UploadPolicySignature: aws.String("stage114-signature"),
		}},
	})
	if err != nil {
		t.Fatalf("bundle instance: %v", err)
	}

	describeBundleTasksOut, err := client.DescribeBundleTasks(ctx, &awsec2.DescribeBundleTasksInput{
		BundleIds: []string{aws.ToString(bundleInstanceOut.BundleTask.BundleId)},
	})
	if err != nil {
		t.Fatalf("describe bundle tasks: %v", err)
	}
	if len(describeBundleTasksOut.BundleTasks) != 1 ||
		aws.ToString(describeBundleTasksOut.BundleTasks[0].BundleId) != aws.ToString(bundleInstanceOut.BundleTask.BundleId) {
		t.Fatalf("unexpected describe bundle tasks output: %#v", describeBundleTasksOut.BundleTasks)
	}

	describeByoipCidrsOut, err := client.DescribeByoipCidrs(ctx, &awsec2.DescribeByoipCidrsInput{MaxResults: aws.Int32(10)})
	if err != nil {
		t.Fatalf("describe byoip cidrs: %v", err)
	}
	if len(describeByoipCidrsOut.ByoipCidrs) == 0 {
		t.Fatalf("expected describe byoip cidrs output")
	}

	describeCapacityBlockExtensionHistoryOut, err := client.DescribeCapacityBlockExtensionHistory(ctx, &awsec2.DescribeCapacityBlockExtensionHistoryInput{
		CapacityReservationIds: []string{"cr-114"},
		MaxResults:             aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity block extension history: %v", err)
	}
	if len(describeCapacityBlockExtensionHistoryOut.CapacityBlockExtensions) == 0 {
		t.Fatalf("expected describe capacity block extension history output")
	}

	describeCapacityBlockExtensionOfferingsOut, err := client.DescribeCapacityBlockExtensionOfferings(ctx, &awsec2.DescribeCapacityBlockExtensionOfferingsInput{
		CapacityReservationId:               aws.String("cr-114"),
		CapacityBlockExtensionDurationHours: aws.Int32(24),
		MaxResults:                          aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity block extension offerings: %v", err)
	}
	if len(describeCapacityBlockExtensionOfferingsOut.CapacityBlockExtensionOfferings) == 0 {
		t.Fatalf("expected describe capacity block extension offerings output")
	}

	describeCapacityBlockOfferingsOut, err := client.DescribeCapacityBlockOfferings(ctx, &awsec2.DescribeCapacityBlockOfferingsInput{
		CapacityDurationHours: aws.Int32(24),
		InstanceCount:         aws.Int32(2),
		InstanceType:          aws.String("p5.48xlarge"),
		MaxResults:            aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity block offerings: %v", err)
	}
	if len(describeCapacityBlockOfferingsOut.CapacityBlockOfferings) == 0 ||
		aws.ToInt32(describeCapacityBlockOfferingsOut.CapacityBlockOfferings[0].InstanceCount) != 2 {
		t.Fatalf("unexpected describe capacity block offerings output: %#v", describeCapacityBlockOfferingsOut.CapacityBlockOfferings)
	}

	describeCapacityBlockStatusOut, err := client.DescribeCapacityBlockStatus(ctx, &awsec2.DescribeCapacityBlockStatusInput{
		CapacityBlockIds: []string{"cb-114"},
		MaxResults:       aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe capacity block status: %v", err)
	}
	if len(describeCapacityBlockStatusOut.CapacityBlockStatuses) == 0 ||
		aws.ToString(describeCapacityBlockStatusOut.CapacityBlockStatuses[0].CapacityBlockId) != "cb-114" {
		t.Fatalf("unexpected describe capacity block status output: %#v", describeCapacityBlockStatusOut.CapacityBlockStatuses)
	}
}

func TestEC2Stage114ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DeprovisionIpamByoasn",
		"DeprovisionIpamPoolCidr",
		"DeprovisionPublicIpv4PoolCidr",
		"DeregisterInstanceEventNotificationAttributes",
		"DescribeBundleTasks",
		"DescribeByoipCidrs",
		"DescribeCapacityBlockExtensionHistory",
		"DescribeCapacityBlockExtensionOfferings",
		"DescribeCapacityBlockOfferings",
		"DescribeCapacityBlockStatus",
	}

	paramsByAction := map[string]map[string]string{
		"DeprovisionIpamByoasn": {
			"Asn":    "64512",
			"IpamId": "ipam-00000000114",
		},
		"DeprovisionIpamPoolCidr": {
			"IpamPoolId": "ipam-pool-00000000114",
		},
		"DeprovisionPublicIpv4PoolCidr": {
			"PoolId": "ipv4pool-ec2-00000000114",
			"Cidr":   "198.51.114.0/24",
		},
		"DeregisterInstanceEventNotificationAttributes": {
			"InstanceTagAttribute.IncludeAllTagsOfInstance": "false",
		},
		"DescribeBundleTasks": {},
		"DescribeByoipCidrs": {
			"MaxResults": "10",
		},
		"DescribeCapacityBlockExtensionHistory": {
			"MaxResults": "10",
		},
		"DescribeCapacityBlockExtensionOfferings": {
			"CapacityReservationId":               "cr-114",
			"CapacityBlockExtensionDurationHours": "24",
			"MaxResults":                          "10",
		},
		"DescribeCapacityBlockOfferings": {
			"CapacityDurationHours": "24",
			"MaxResults":            "10",
		},
		"DescribeCapacityBlockStatus": {
			"MaxResults": "10",
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
