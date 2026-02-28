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

func TestEC2Stage69SDKLifecycle(t *testing.T) {
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

	createInstanceOut, err := client.CreateVerifiedAccessInstance(ctx, &awsec2.CreateVerifiedAccessInstanceInput{
		Description: aws.String("stage69 instance"),
	})
	if err != nil || createInstanceOut.VerifiedAccessInstance == nil || createInstanceOut.VerifiedAccessInstance.VerifiedAccessInstanceId == nil {
		t.Fatalf("create verified access instance: %v", err)
	}
	instanceID := aws.ToString(createInstanceOut.VerifiedAccessInstance.VerifiedAccessInstanceId)

	createGroupOut, err := client.CreateVerifiedAccessGroup(ctx, &awsec2.CreateVerifiedAccessGroupInput{
		VerifiedAccessInstanceId: aws.String(instanceID),
		Description:              aws.String("stage69 group"),
	})
	if err != nil || createGroupOut.VerifiedAccessGroup == nil || createGroupOut.VerifiedAccessGroup.VerifiedAccessGroupId == nil {
		t.Fatalf("create verified access group: %v", err)
	}
	groupID := aws.ToString(createGroupOut.VerifiedAccessGroup.VerifiedAccessGroupId)

	createEndpointOut, err := client.CreateVerifiedAccessEndpoint(ctx, &awsec2.CreateVerifiedAccessEndpointInput{
		VerifiedAccessGroupId: aws.String(groupID),
		AttachmentType:        awsec2types.VerifiedAccessEndpointAttachmentTypeVpc,
		EndpointType:          awsec2types.VerifiedAccessEndpointTypeLoadBalancer,
		ApplicationDomain:     aws.String("app.stage69.example.com"),
		SecurityGroupIds:      []string{"sg-00000000"},
	})
	if err != nil || createEndpointOut.VerifiedAccessEndpoint == nil || createEndpointOut.VerifiedAccessEndpoint.VerifiedAccessEndpointId == nil {
		t.Fatalf("create verified access endpoint: %v", err)
	}
	endpointID := aws.ToString(createEndpointOut.VerifiedAccessEndpoint.VerifiedAccessEndpointId)

	createTrustProviderOut, err := client.CreateVerifiedAccessTrustProvider(ctx, &awsec2.CreateVerifiedAccessTrustProviderInput{
		PolicyReferenceName:   aws.String("stage69-policy"),
		TrustProviderType:     awsec2types.TrustProviderTypeUser,
		UserTrustProviderType: awsec2types.UserTrustProviderTypeIamIdentityCenter,
	})
	if err != nil || createTrustProviderOut.VerifiedAccessTrustProvider == nil || createTrustProviderOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId == nil {
		t.Fatalf("create verified access trust provider: %v", err)
	}
	trustProviderID := aws.ToString(createTrustProviderOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId)

	attachOut, err := client.AttachVerifiedAccessTrustProvider(ctx, &awsec2.AttachVerifiedAccessTrustProviderInput{
		VerifiedAccessInstanceId:      aws.String(instanceID),
		VerifiedAccessTrustProviderId: aws.String(trustProviderID),
	})
	if err != nil || attachOut.VerifiedAccessTrustProvider == nil {
		t.Fatalf("attach verified access trust provider: %v", err)
	}
	if aws.ToString(attachOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId) != trustProviderID {
		t.Fatalf("unexpected attached trust provider id: %q", aws.ToString(attachOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId))
	}

	modifyInstanceOut, err := client.ModifyVerifiedAccessInstance(ctx, &awsec2.ModifyVerifiedAccessInstanceInput{
		VerifiedAccessInstanceId:     aws.String(instanceID),
		Description:                  aws.String("stage69 instance updated"),
		CidrEndpointsCustomSubDomain: aws.String("stage69"),
	})
	if err != nil || modifyInstanceOut.VerifiedAccessInstance == nil {
		t.Fatalf("modify verified access instance: %v", err)
	}
	if aws.ToString(modifyInstanceOut.VerifiedAccessInstance.Description) != "stage69 instance updated" {
		t.Fatalf("unexpected modified instance description: %q", aws.ToString(modifyInstanceOut.VerifiedAccessInstance.Description))
	}

	modifyLoggingOut, err := client.ModifyVerifiedAccessInstanceLoggingConfiguration(ctx, &awsec2.ModifyVerifiedAccessInstanceLoggingConfigurationInput{
		VerifiedAccessInstanceId: aws.String(instanceID),
		AccessLogs: &awsec2types.VerifiedAccessLogOptions{
			IncludeTrustContext: aws.Bool(true),
			LogVersion:          aws.String("ocsf-1.0.0-rc.2"),
			CloudWatchLogs: &awsec2types.VerifiedAccessLogCloudWatchLogsDestinationOptions{
				Enabled:  aws.Bool(true),
				LogGroup: aws.String("stage69-log-group"),
			},
		},
	})
	if err != nil || modifyLoggingOut.LoggingConfiguration == nil || modifyLoggingOut.LoggingConfiguration.AccessLogs == nil {
		t.Fatalf("modify verified access instance logging configuration: %v", err)
	}
	if !aws.ToBool(modifyLoggingOut.LoggingConfiguration.AccessLogs.IncludeTrustContext) {
		t.Fatalf("expected include trust context true after logging configuration update")
	}

	describeLoggingOut, err := client.DescribeVerifiedAccessInstanceLoggingConfigurations(ctx, &awsec2.DescribeVerifiedAccessInstanceLoggingConfigurationsInput{
		VerifiedAccessInstanceIds: []string{instanceID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("verified-access-instance-id"), Values: []string{instanceID}},
			{Name: aws.String("access-logs.include-trust-context"), Values: []string{"true"}},
		},
	})
	if err != nil {
		t.Fatalf("describe verified access instance logging configurations: %v", err)
	}
	if len(describeLoggingOut.LoggingConfigurations) != 1 {
		t.Fatalf("expected one logging configuration, got %d", len(describeLoggingOut.LoggingConfigurations))
	}
	if aws.ToString(describeLoggingOut.LoggingConfigurations[0].VerifiedAccessInstanceId) != instanceID {
		t.Fatalf("unexpected logging configuration instance id: %q", aws.ToString(describeLoggingOut.LoggingConfigurations[0].VerifiedAccessInstanceId))
	}

	exportConfigOut, err := client.ExportVerifiedAccessInstanceClientConfiguration(ctx, &awsec2.ExportVerifiedAccessInstanceClientConfigurationInput{
		VerifiedAccessInstanceId: aws.String(instanceID),
	})
	if err != nil {
		t.Fatalf("export verified access instance client configuration: %v", err)
	}
	if aws.ToString(exportConfigOut.VerifiedAccessInstanceId) != instanceID {
		t.Fatalf("unexpected exported config instance id: %q", aws.ToString(exportConfigOut.VerifiedAccessInstanceId))
	}
	if len(exportConfigOut.OpenVpnConfigurations) == 0 {
		t.Fatalf("expected at least one openvpn configuration")
	}

	modifyGroupOut, err := client.ModifyVerifiedAccessGroup(ctx, &awsec2.ModifyVerifiedAccessGroupInput{
		VerifiedAccessGroupId: aws.String(groupID),
		Description:           aws.String("stage69 group updated"),
	})
	if err != nil || modifyGroupOut.VerifiedAccessGroup == nil {
		t.Fatalf("modify verified access group: %v", err)
	}
	if aws.ToString(modifyGroupOut.VerifiedAccessGroup.Description) != "stage69 group updated" {
		t.Fatalf("unexpected modified group description: %q", aws.ToString(modifyGroupOut.VerifiedAccessGroup.Description))
	}

	modifyGroupPolicyOut, err := client.ModifyVerifiedAccessGroupPolicy(ctx, &awsec2.ModifyVerifiedAccessGroupPolicyInput{
		VerifiedAccessGroupId: aws.String(groupID),
		PolicyEnabled:         aws.Bool(true),
		PolicyDocument:        aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`),
	})
	if err != nil {
		t.Fatalf("modify verified access group policy: %v", err)
	}
	if !aws.ToBool(modifyGroupPolicyOut.PolicyEnabled) {
		t.Fatalf("expected modified group policy enabled true")
	}

	getGroupPolicyOut, err := client.GetVerifiedAccessGroupPolicy(ctx, &awsec2.GetVerifiedAccessGroupPolicyInput{
		VerifiedAccessGroupId: aws.String(groupID),
	})
	if err != nil {
		t.Fatalf("get verified access group policy: %v", err)
	}
	if !aws.ToBool(getGroupPolicyOut.PolicyEnabled) {
		t.Fatalf("expected get group policy enabled true")
	}

	modifyEndpointOut, err := client.ModifyVerifiedAccessEndpoint(ctx, &awsec2.ModifyVerifiedAccessEndpointInput{
		VerifiedAccessEndpointId: aws.String(endpointID),
		VerifiedAccessGroupId:    aws.String(groupID),
		Description:              aws.String("stage69 endpoint updated"),
	})
	if err != nil || modifyEndpointOut.VerifiedAccessEndpoint == nil {
		t.Fatalf("modify verified access endpoint: %v", err)
	}
	if aws.ToString(modifyEndpointOut.VerifiedAccessEndpoint.Description) != "stage69 endpoint updated" {
		t.Fatalf("unexpected modified endpoint description: %q", aws.ToString(modifyEndpointOut.VerifiedAccessEndpoint.Description))
	}

	modifyEndpointPolicyOut, err := client.ModifyVerifiedAccessEndpointPolicy(ctx, &awsec2.ModifyVerifiedAccessEndpointPolicyInput{
		VerifiedAccessEndpointId: aws.String(endpointID),
		PolicyEnabled:            aws.Bool(true),
		PolicyDocument:           aws.String(`{"Version":"2012-10-17","Statement":[]}`),
	})
	if err != nil {
		t.Fatalf("modify verified access endpoint policy: %v", err)
	}
	if !aws.ToBool(modifyEndpointPolicyOut.PolicyEnabled) {
		t.Fatalf("expected modified endpoint policy enabled true")
	}

	getEndpointPolicyOut, err := client.GetVerifiedAccessEndpointPolicy(ctx, &awsec2.GetVerifiedAccessEndpointPolicyInput{
		VerifiedAccessEndpointId: aws.String(endpointID),
	})
	if err != nil {
		t.Fatalf("get verified access endpoint policy: %v", err)
	}
	if !aws.ToBool(getEndpointPolicyOut.PolicyEnabled) {
		t.Fatalf("expected get endpoint policy enabled true")
	}

	getTargetsOut, err := client.GetVerifiedAccessEndpointTargets(ctx, &awsec2.GetVerifiedAccessEndpointTargetsInput{
		VerifiedAccessEndpointId: aws.String(endpointID),
		MaxResults:               aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("get verified access endpoint targets: %v", err)
	}
	if len(getTargetsOut.VerifiedAccessEndpointTargets) != 1 {
		t.Fatalf("expected one endpoint target, got %d", len(getTargetsOut.VerifiedAccessEndpointTargets))
	}

	modifyTrustProviderOut, err := client.ModifyVerifiedAccessTrustProvider(ctx, &awsec2.ModifyVerifiedAccessTrustProviderInput{
		VerifiedAccessTrustProviderId: aws.String(trustProviderID),
		Description:                   aws.String("stage69 trust provider updated"),
	})
	if err != nil || modifyTrustProviderOut.VerifiedAccessTrustProvider == nil {
		t.Fatalf("modify verified access trust provider: %v", err)
	}
	if aws.ToString(modifyTrustProviderOut.VerifiedAccessTrustProvider.Description) != "stage69 trust provider updated" {
		t.Fatalf("unexpected modified trust provider description: %q", aws.ToString(modifyTrustProviderOut.VerifiedAccessTrustProvider.Description))
	}

	detachOut, err := client.DetachVerifiedAccessTrustProvider(ctx, &awsec2.DetachVerifiedAccessTrustProviderInput{
		VerifiedAccessInstanceId:      aws.String(instanceID),
		VerifiedAccessTrustProviderId: aws.String(trustProviderID),
	})
	if err != nil || detachOut.VerifiedAccessTrustProvider == nil {
		t.Fatalf("detach verified access trust provider: %v", err)
	}
	if aws.ToString(detachOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId) != trustProviderID {
		t.Fatalf("unexpected detached trust provider id: %q", aws.ToString(detachOut.VerifiedAccessTrustProvider.VerifiedAccessTrustProviderId))
	}

	if _, err := client.DeleteVerifiedAccessEndpoint(ctx, &awsec2.DeleteVerifiedAccessEndpointInput{
		VerifiedAccessEndpointId: aws.String(endpointID),
	}); err != nil {
		t.Fatalf("delete verified access endpoint: %v", err)
	}

	if _, err := client.DeleteVerifiedAccessGroup(ctx, &awsec2.DeleteVerifiedAccessGroupInput{
		VerifiedAccessGroupId: aws.String(groupID),
	}); err != nil {
		t.Fatalf("delete verified access group: %v", err)
	}

	if _, err := client.DeleteVerifiedAccessTrustProvider(ctx, &awsec2.DeleteVerifiedAccessTrustProviderInput{
		VerifiedAccessTrustProviderId: aws.String(trustProviderID),
	}); err != nil {
		t.Fatalf("delete verified access trust provider: %v", err)
	}

	if _, err := client.DeleteVerifiedAccessInstance(ctx, &awsec2.DeleteVerifiedAccessInstanceInput{
		VerifiedAccessInstanceId: aws.String(instanceID),
	}); err != nil {
		t.Fatalf("delete verified access instance: %v", err)
	}
}

func TestEC2Stage69ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"AttachVerifiedAccessTrustProvider",
		"DescribeVerifiedAccessInstanceLoggingConfigurations",
		"DetachVerifiedAccessTrustProvider",
		"ExportVerifiedAccessInstanceClientConfiguration",
		"GetVerifiedAccessEndpointPolicy",
		"GetVerifiedAccessEndpointTargets",
		"GetVerifiedAccessGroupPolicy",
		"ModifyVerifiedAccessEndpoint",
		"ModifyVerifiedAccessEndpointPolicy",
		"ModifyVerifiedAccessGroup",
		"ModifyVerifiedAccessGroupPolicy",
		"ModifyVerifiedAccessInstance",
		"ModifyVerifiedAccessInstanceLoggingConfiguration",
		"ModifyVerifiedAccessTrustProvider",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "AttachVerifiedAccessTrustProvider", "DetachVerifiedAccessTrustProvider":
			params["VerifiedAccessInstanceId"] = "vai-00000000"
			params["VerifiedAccessTrustProviderId"] = "vatp-00000000"
		case "ExportVerifiedAccessInstanceClientConfiguration":
			params["VerifiedAccessInstanceId"] = "vai-00000000"
		case "GetVerifiedAccessEndpointPolicy", "GetVerifiedAccessEndpointTargets":
			params["VerifiedAccessEndpointId"] = "vae-00000000"
		case "GetVerifiedAccessGroupPolicy", "ModifyVerifiedAccessGroup", "ModifyVerifiedAccessGroupPolicy":
			params["VerifiedAccessGroupId"] = "vag-00000000"
		case "ModifyVerifiedAccessEndpoint":
			params["VerifiedAccessEndpointId"] = "vae-00000000"
		case "ModifyVerifiedAccessEndpointPolicy":
			params["VerifiedAccessEndpointId"] = "vae-00000000"
		case "ModifyVerifiedAccessInstance":
			params["VerifiedAccessInstanceId"] = "vai-00000000"
		case "ModifyVerifiedAccessInstanceLoggingConfiguration":
			params["VerifiedAccessInstanceId"] = "vai-00000000"
			params["AccessLogs.IncludeTrustContext"] = "true"
		case "ModifyVerifiedAccessTrustProvider":
			params["VerifiedAccessTrustProviderId"] = "vatp-00000000"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
