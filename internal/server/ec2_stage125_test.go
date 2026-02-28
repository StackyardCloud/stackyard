package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage125SDKLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-stage125"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runInstancesOut.Instances) == 0 || runInstancesOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runInstancesOut.Instances[0].InstanceId)

	getInstanceUefiDataOut, err := client.GetInstanceUefiData(ctx, &awsec2.GetInstanceUefiDataInput{
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		t.Fatalf("get instance uefi data: %v", err)
	}
	if aws.ToString(getInstanceUefiDataOut.InstanceId) != instanceID || aws.ToString(getInstanceUefiDataOut.UefiData) == "" {
		t.Fatalf("unexpected get instance uefi data output: %#v", getInstanceUefiDataOut)
	}

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{
		Description: aws.String("stage125-ipam"),
	})
	if err != nil || createIpamOut.Ipam == nil || createIpamOut.Ipam.IpamId == nil {
		t.Fatalf("create ipam: %v", err)
	}
	ipamID := aws.ToString(createIpamOut.Ipam.IpamId)

	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{
		IpamId: aws.String(ipamID),
	})
	if err != nil || createIpamScopeOut.IpamScope == nil || createIpamScopeOut.IpamScope.IpamScopeId == nil {
		t.Fatalf("create ipam scope: %v", err)
	}
	ipamScopeID := aws.ToString(createIpamScopeOut.IpamScope.IpamScopeId)

	createIpamResourceDiscoveryOut, err := client.CreateIpamResourceDiscovery(ctx, &awsec2.CreateIpamResourceDiscoveryInput{
		Description: aws.String("stage125-discovery"),
	})
	if err != nil || createIpamResourceDiscoveryOut.IpamResourceDiscovery == nil || createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId == nil {
		t.Fatalf("create ipam resource discovery: %v", err)
	}
	ipamResourceDiscoveryID := aws.ToString(createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId)

	createIpamPoolOut, err := client.CreateIpamPool(ctx, &awsec2.CreateIpamPoolInput{
		AddressFamily: awsec2types.AddressFamilyIpv4,
		IpamScopeId:   aws.String(ipamScopeID),
		Description:   aws.String("stage125-pool"),
	})
	if err != nil || createIpamPoolOut.IpamPool == nil || createIpamPoolOut.IpamPool.IpamPoolId == nil {
		t.Fatalf("create ipam pool: %v", err)
	}
	ipamPoolID := aws.ToString(createIpamPoolOut.IpamPool.IpamPoolId)

	allocateIpamPoolCidrOut, err := client.AllocateIpamPoolCidr(ctx, &awsec2.AllocateIpamPoolCidrInput{
		IpamPoolId: aws.String(ipamPoolID),
		Cidr:       aws.String("10.125.0.0/24"),
	})
	if err != nil || allocateIpamPoolCidrOut.IpamPoolAllocation == nil {
		t.Fatalf("allocate ipam pool cidr: %v", err)
	}

	now := time.Now().UTC()
	getIpamAddressHistoryOut, err := client.GetIpamAddressHistory(ctx, &awsec2.GetIpamAddressHistoryInput{
		Cidr:        aws.String("10.0.0.0/16"),
		IpamScopeId: aws.String(ipamScopeID),
		StartTime:   aws.Time(now.Add(-10 * time.Minute)),
		EndTime:     aws.Time(now),
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get ipam address history: %v", err)
	}
	if len(getIpamAddressHistoryOut.HistoryRecords) == 0 {
		t.Fatalf("expected ipam address history records")
	}

	getIpamDiscoveredAccountsOut, err := client.GetIpamDiscoveredAccounts(ctx, &awsec2.GetIpamDiscoveredAccountsInput{
		DiscoveryRegion:         aws.String(testRegion),
		IpamResourceDiscoveryId: aws.String(ipamResourceDiscoveryID),
		MaxResults:              aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get ipam discovered accounts: %v", err)
	}
	if len(getIpamDiscoveredAccountsOut.IpamDiscoveredAccounts) == 0 {
		t.Fatalf("expected ipam discovered accounts")
	}

	getIpamDiscoveredPublicAddressesOut, err := client.GetIpamDiscoveredPublicAddresses(ctx, &awsec2.GetIpamDiscoveredPublicAddressesInput{
		AddressRegion:           aws.String(testRegion),
		IpamResourceDiscoveryId: aws.String(ipamResourceDiscoveryID),
		MaxResults:              aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get ipam discovered public addresses: %v", err)
	}
	if len(getIpamDiscoveredPublicAddressesOut.IpamDiscoveredPublicAddresses) == 0 {
		t.Fatalf("expected ipam discovered public addresses")
	}

	getIpamDiscoveredResourceCidrsOut, err := client.GetIpamDiscoveredResourceCidrs(ctx, &awsec2.GetIpamDiscoveredResourceCidrsInput{
		IpamResourceDiscoveryId: aws.String(ipamResourceDiscoveryID),
		ResourceRegion:          aws.String(testRegion),
		MaxResults:              aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get ipam discovered resource cidrs: %v", err)
	}
	if len(getIpamDiscoveredResourceCidrsOut.IpamDiscoveredResourceCidrs) == 0 {
		t.Fatalf("expected ipam discovered resource cidrs")
	}

	getIpamPoolAllocationsOut, err := client.GetIpamPoolAllocations(ctx, &awsec2.GetIpamPoolAllocationsInput{
		IpamPoolId: aws.String(ipamPoolID),
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get ipam pool allocations: %v", err)
	}
	if len(getIpamPoolAllocationsOut.IpamPoolAllocations) == 0 {
		t.Fatalf("expected ipam pool allocations")
	}

	getIpamPoolCidrsOut, err := client.GetIpamPoolCidrs(ctx, &awsec2.GetIpamPoolCidrsInput{
		IpamPoolId: aws.String(ipamPoolID),
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get ipam pool cidrs: %v", err)
	}
	if len(getIpamPoolCidrsOut.IpamPoolCidrs) == 0 {
		t.Fatalf("expected ipam pool cidrs")
	}

	getIpamResourceCidrsOut, err := client.GetIpamResourceCidrs(ctx, &awsec2.GetIpamResourceCidrsInput{
		IpamPoolId:  aws.String(ipamPoolID),
		IpamScopeId: aws.String(ipamScopeID),
		MaxResults:  aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get ipam resource cidrs: %v", err)
	}
	if len(getIpamResourceCidrsOut.IpamResourceCidrs) == 0 {
		t.Fatalf("expected ipam resource cidrs")
	}

	getLaunchTemplateDataOut, err := client.GetLaunchTemplateData(ctx, &awsec2.GetLaunchTemplateDataInput{
		InstanceId: aws.String(instanceID),
	})
	if err != nil {
		t.Fatalf("get launch template data: %v", err)
	}
	if getLaunchTemplateDataOut.LaunchTemplateData == nil || aws.ToString(getLaunchTemplateDataOut.LaunchTemplateData.ImageId) == "" {
		t.Fatalf("expected launch template data")
	}

	createManagedPrefixListOut, err := client.CreateManagedPrefixList(ctx, &awsec2.CreateManagedPrefixListInput{
		AddressFamily:  aws.String("ipv4"),
		MaxEntries:     aws.Int32(5),
		PrefixListName: aws.String("stage125-prefix-list"),
	})
	if err != nil || createManagedPrefixListOut.PrefixList == nil || createManagedPrefixListOut.PrefixList.PrefixListId == nil {
		t.Fatalf("create managed prefix list: %v", err)
	}
	prefixListID := aws.ToString(createManagedPrefixListOut.PrefixList.PrefixListId)

	getManagedPrefixListAssociationsOut, err := client.GetManagedPrefixListAssociations(ctx, &awsec2.GetManagedPrefixListAssociationsInput{
		PrefixListId: aws.String(prefixListID),
		MaxResults:   aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("get managed prefix list associations: %v", err)
	}
	if len(getManagedPrefixListAssociationsOut.PrefixListAssociations) == 0 {
		t.Fatalf("expected managed prefix list associations")
	}
}

func TestEC2Stage125ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"GetInstanceUefiData",
		"GetIpamAddressHistory",
		"GetIpamDiscoveredAccounts",
		"GetIpamDiscoveredPublicAddresses",
		"GetIpamDiscoveredResourceCidrs",
		"GetIpamPoolAllocations",
		"GetIpamPoolCidrs",
		"GetIpamResourceCidrs",
		"GetLaunchTemplateData",
		"GetManagedPrefixListAssociations",
	}

	paramsByAction := map[string]map[string]string{
		"GetInstanceUefiData": {
			"InstanceId": "i-0000000125",
		},
		"GetIpamAddressHistory": {
			"Cidr":        "10.0.0.0/16",
			"IpamScopeId": "ipam-scope-0000000125",
		},
		"GetIpamDiscoveredAccounts": {
			"IpamResourceDiscoveryId": "ipam-rd-0000000125",
		},
		"GetIpamDiscoveredPublicAddresses": {
			"IpamResourceDiscoveryId": "ipam-rd-0000000125",
		},
		"GetIpamDiscoveredResourceCidrs": {
			"IpamResourceDiscoveryId": "ipam-rd-0000000125",
		},
		"GetIpamPoolAllocations": {
			"IpamPoolId": "ipam-pool-0000000125",
		},
		"GetIpamPoolCidrs": {
			"IpamPoolId": "ipam-pool-0000000125",
		},
		"GetIpamResourceCidrs": {
			"IpamPoolId": "ipam-pool-0000000125",
		},
		"GetLaunchTemplateData": {
			"InstanceId": "i-0000000125",
		},
		"GetManagedPrefixListAssociations": {
			"PrefixListId": "pl-0000000125",
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
