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

func TestEC2Stage111SDKLifecycle(t *testing.T) {
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

	createFlowLogsOut, err := client.CreateFlowLogs(ctx, &awsec2.CreateFlowLogsInput{
		ResourceIds:  []string{"vpc-00000001"},
		ResourceType: awsec2types.FlowLogsResourceTypeVpc,
		TrafficType:  awsec2types.TrafficTypeAll,
	})
	if err != nil {
		t.Fatalf("create flow logs: %v", err)
	}
	deleteFlowLogsOut, err := client.DeleteFlowLogs(ctx, &awsec2.DeleteFlowLogsInput{
		FlowLogIds: createFlowLogsOut.FlowLogIds,
	})
	if err != nil {
		t.Fatalf("delete flow logs: %v", err)
	}
	if len(deleteFlowLogsOut.Unsuccessful) != 0 {
		t.Fatalf("expected no unsuccessful flow log deletions: %#v", deleteFlowLogsOut.Unsuccessful)
	}

	createFpgaImageOut, err := client.CreateFpgaImage(ctx, &awsec2.CreateFpgaImageInput{
		InputStorageLocation: &awsec2types.StorageLocation{
			Bucket: aws.String("stage111-bucket"),
			Key:    aws.String("stage111/input.xclbin"),
		},
	})
	if err != nil {
		t.Fatalf("create fpga image: %v", err)
	}
	deleteFpgaImageOut, err := client.DeleteFpgaImage(ctx, &awsec2.DeleteFpgaImageInput{
		FpgaImageId: createFpgaImageOut.FpgaImageId,
	})
	if err != nil {
		t.Fatalf("delete fpga image: %v", err)
	}
	if !aws.ToBool(deleteFpgaImageOut.Return) {
		t.Fatalf("expected delete fpga image return=true")
	}

	createInstanceConnectEndpointOut, err := client.CreateInstanceConnectEndpoint(ctx, &awsec2.CreateInstanceConnectEndpointInput{
		SubnetId: aws.String("subnet-00000001"),
	})
	if err != nil {
		t.Fatalf("create instance connect endpoint: %v", err)
	}
	deleteInstanceConnectEndpointOut, err := client.DeleteInstanceConnectEndpoint(ctx, &awsec2.DeleteInstanceConnectEndpointInput{
		InstanceConnectEndpointId: createInstanceConnectEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId,
	})
	if err != nil {
		t.Fatalf("delete instance connect endpoint: %v", err)
	}
	if deleteInstanceConnectEndpointOut.InstanceConnectEndpoint == nil ||
		aws.ToString(deleteInstanceConnectEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId) != aws.ToString(createInstanceConnectEndpointOut.InstanceConnectEndpoint.InstanceConnectEndpointId) {
		t.Fatalf("unexpected delete instance connect endpoint output: %#v", deleteInstanceConnectEndpointOut.InstanceConnectEndpoint)
	}

	createInstanceEventWindowOut, err := client.CreateInstanceEventWindow(ctx, &awsec2.CreateInstanceEventWindowInput{
		Name:           aws.String("stage111-window"),
		CronExpression: aws.String("cron(0 10 ? * SUN *)"),
	})
	if err != nil {
		t.Fatalf("create instance event window: %v", err)
	}
	deleteInstanceEventWindowOut, err := client.DeleteInstanceEventWindow(ctx, &awsec2.DeleteInstanceEventWindowInput{
		InstanceEventWindowId: createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId,
	})
	if err != nil {
		t.Fatalf("delete instance event window: %v", err)
	}
	if deleteInstanceEventWindowOut.InstanceEventWindowState == nil ||
		aws.ToString(deleteInstanceEventWindowOut.InstanceEventWindowState.InstanceEventWindowId) != aws.ToString(createInstanceEventWindowOut.InstanceEventWindow.InstanceEventWindowId) ||
		deleteInstanceEventWindowOut.InstanceEventWindowState.State != awsec2types.InstanceEventWindowStateDeleted {
		t.Fatalf("unexpected delete instance event window output: %#v", deleteInstanceEventWindowOut.InstanceEventWindowState)
	}

	createIpamOut, err := client.CreateIpam(ctx, &awsec2.CreateIpamInput{
		Description: aws.String("stage111-ipam"),
	})
	if err != nil {
		t.Fatalf("create ipam: %v", err)
	}

	createIpamExternalResourceVerificationTokenOut, err := client.CreateIpamExternalResourceVerificationToken(ctx, &awsec2.CreateIpamExternalResourceVerificationTokenInput{
		IpamId: createIpamOut.Ipam.IpamId,
	})
	if err != nil {
		t.Fatalf("create ipam external resource verification token: %v", err)
	}

	createIpamScopeOut, err := client.CreateIpamScope(ctx, &awsec2.CreateIpamScopeInput{
		IpamId: createIpamOut.Ipam.IpamId,
	})
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

	createIpamResourceDiscoveryOut, err := client.CreateIpamResourceDiscovery(ctx, &awsec2.CreateIpamResourceDiscoveryInput{
		Description: aws.String("stage111-discovery"),
	})
	if err != nil {
		t.Fatalf("create ipam resource discovery: %v", err)
	}

	deleteIpamExternalResourceVerificationTokenOut, err := client.DeleteIpamExternalResourceVerificationToken(ctx, &awsec2.DeleteIpamExternalResourceVerificationTokenInput{
		IpamExternalResourceVerificationTokenId: createIpamExternalResourceVerificationTokenOut.IpamExternalResourceVerificationToken.IpamExternalResourceVerificationTokenId,
	})
	if err != nil {
		t.Fatalf("delete ipam external resource verification token: %v", err)
	}
	if deleteIpamExternalResourceVerificationTokenOut.IpamExternalResourceVerificationToken == nil {
		t.Fatalf("expected deleted ipam external resource verification token")
	}

	deleteIpamPoolOut, err := client.DeleteIpamPool(ctx, &awsec2.DeleteIpamPoolInput{
		IpamPoolId: createIpamPoolOut.IpamPool.IpamPoolId,
	})
	if err != nil {
		t.Fatalf("delete ipam pool: %v", err)
	}
	if deleteIpamPoolOut.IpamPool == nil || aws.ToString(deleteIpamPoolOut.IpamPool.IpamPoolId) != aws.ToString(createIpamPoolOut.IpamPool.IpamPoolId) {
		t.Fatalf("unexpected delete ipam pool output: %#v", deleteIpamPoolOut.IpamPool)
	}

	deleteIpamScopeOut, err := client.DeleteIpamScope(ctx, &awsec2.DeleteIpamScopeInput{
		IpamScopeId: createIpamScopeOut.IpamScope.IpamScopeId,
	})
	if err != nil {
		t.Fatalf("delete ipam scope: %v", err)
	}
	if deleteIpamScopeOut.IpamScope == nil || aws.ToString(deleteIpamScopeOut.IpamScope.IpamScopeId) != aws.ToString(createIpamScopeOut.IpamScope.IpamScopeId) {
		t.Fatalf("unexpected delete ipam scope output: %#v", deleteIpamScopeOut.IpamScope)
	}

	deleteIpamResourceDiscoveryOut, err := client.DeleteIpamResourceDiscovery(ctx, &awsec2.DeleteIpamResourceDiscoveryInput{
		IpamResourceDiscoveryId: createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId,
	})
	if err != nil {
		t.Fatalf("delete ipam resource discovery: %v", err)
	}
	if deleteIpamResourceDiscoveryOut.IpamResourceDiscovery == nil ||
		aws.ToString(deleteIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId) != aws.ToString(createIpamResourceDiscoveryOut.IpamResourceDiscovery.IpamResourceDiscoveryId) {
		t.Fatalf("unexpected delete ipam resource discovery output: %#v", deleteIpamResourceDiscoveryOut.IpamResourceDiscovery)
	}

	deleteIpamOut, err := client.DeleteIpam(ctx, &awsec2.DeleteIpamInput{
		IpamId: createIpamOut.Ipam.IpamId,
	})
	if err != nil {
		t.Fatalf("delete ipam: %v", err)
	}
	if deleteIpamOut.Ipam == nil || aws.ToString(deleteIpamOut.Ipam.IpamId) != aws.ToString(createIpamOut.Ipam.IpamId) {
		t.Fatalf("unexpected delete ipam output: %#v", deleteIpamOut.Ipam)
	}

	createLaunchTemplateOut, err := client.CreateLaunchTemplate(ctx, &awsec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String("stage111-template"),
		LaunchTemplateData: &awsec2types.RequestLaunchTemplateData{
			ImageId: aws.String("ami-00000000000000111"),
		},
	})
	if err != nil {
		t.Fatalf("create launch template: %v", err)
	}
	deleteLaunchTemplateOut, err := client.DeleteLaunchTemplate(ctx, &awsec2.DeleteLaunchTemplateInput{
		LaunchTemplateId: createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId,
	})
	if err != nil {
		t.Fatalf("delete launch template: %v", err)
	}
	if deleteLaunchTemplateOut.LaunchTemplate == nil || aws.ToString(deleteLaunchTemplateOut.LaunchTemplate.LaunchTemplateId) != aws.ToString(createLaunchTemplateOut.LaunchTemplate.LaunchTemplateId) {
		t.Fatalf("unexpected delete launch template output: %#v", deleteLaunchTemplateOut.LaunchTemplate)
	}
}

func TestEC2Stage111ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DeleteFlowLogs",
		"DeleteFpgaImage",
		"DeleteInstanceConnectEndpoint",
		"DeleteInstanceEventWindow",
		"DeleteIpam",
		"DeleteIpamExternalResourceVerificationToken",
		"DeleteIpamPool",
		"DeleteIpamResourceDiscovery",
		"DeleteIpamScope",
		"DeleteLaunchTemplate",
	}

	paramsByAction := map[string]map[string]string{
		"DeleteFlowLogs": {
			"FlowLogId.1": "fl-00000000111",
		},
		"DeleteFpgaImage": {
			"FpgaImageId": "afi-00000000111",
		},
		"DeleteInstanceConnectEndpoint": {
			"InstanceConnectEndpointId": "eice-00000000111",
		},
		"DeleteInstanceEventWindow": {
			"InstanceEventWindowId": "iew-00000000111",
		},
		"DeleteIpam": {
			"IpamId": "ipam-00000000111",
		},
		"DeleteIpamExternalResourceVerificationToken": {
			"IpamExternalResourceVerificationTokenId": "ipam-ervt-00000000111",
		},
		"DeleteIpamPool": {
			"IpamPoolId": "ipam-pool-00000000111",
		},
		"DeleteIpamResourceDiscovery": {
			"IpamResourceDiscoveryId": "ipam-rd-00000000111",
		},
		"DeleteIpamScope": {
			"IpamScopeId": "ipam-scope-00000000111",
		},
		"DeleteLaunchTemplate": {
			"LaunchTemplateId": "lt-00000000111",
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
