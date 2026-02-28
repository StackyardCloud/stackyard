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

func TestEC2Stage21SDKLifecycle(t *testing.T) {
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

	createEndpointOut, err := client.CreateClientVpnEndpoint(ctx, &awsec2.CreateClientVpnEndpointInput{
		AuthenticationOptions: []awsec2types.ClientVpnAuthenticationRequest{
			{
				Type: awsec2types.ClientVpnAuthenticationTypeCertificateAuthentication,
				MutualAuthentication: &awsec2types.CertificateAuthenticationRequest{
					ClientRootCertificateChainArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/client-root"),
				},
			},
		},
		ClientCidrBlock:      aws.String("172.16.0.0/22"),
		ConnectionLogOptions: &awsec2types.ConnectionLogOptions{Enabled: aws.Bool(false)},
		ServerCertificateArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/server"),
		Description:          aws.String("stage21-endpoint"),
		VpcId:                aws.String("vpc-00000001"),
		DnsServers:           []string{"10.0.0.2"},
		SecurityGroupIds:     []string{"sg-00000000"},
		SplitTunnel:          aws.Bool(true),
		SessionTimeoutHours:  aws.Int32(24),
		TransportProtocol:    awsec2types.TransportProtocolUdp,
	})
	if err != nil || createEndpointOut.ClientVpnEndpointId == nil {
		t.Fatalf("create client vpn endpoint: %v", err)
	}
	endpointID := aws.ToString(createEndpointOut.ClientVpnEndpointId)

	describeEndpointsOut, err := client.DescribeClientVpnEndpoints(ctx, &awsec2.DescribeClientVpnEndpointsInput{
		ClientVpnEndpointIds: []string{endpointID},
	})
	if err != nil || len(describeEndpointsOut.ClientVpnEndpoints) != 1 || aws.ToString(describeEndpointsOut.ClientVpnEndpoints[0].ClientVpnEndpointId) != endpointID {
		t.Fatalf("describe client vpn endpoints: %v", err)
	}

	if _, err := client.ModifyClientVpnEndpoint(ctx, &awsec2.ModifyClientVpnEndpointInput{
		ClientVpnEndpointId: aws.String(endpointID),
		Description:         aws.String("stage21-endpoint-updated"),
		SecurityGroupIds:    []string{"sg-00000000"},
	}); err != nil {
		t.Fatalf("modify client vpn endpoint: %v", err)
	}

	associateTargetNetworkOut, err := client.AssociateClientVpnTargetNetwork(ctx, &awsec2.AssociateClientVpnTargetNetworkInput{
		ClientVpnEndpointId: aws.String(endpointID),
		SubnetId:            aws.String("subnet-00000001"),
	})
	if err != nil || associateTargetNetworkOut.AssociationId == nil {
		t.Fatalf("associate client vpn target network: %v", err)
	}
	associationID := aws.ToString(associateTargetNetworkOut.AssociationId)

	applySecurityGroupsOut, err := client.ApplySecurityGroupsToClientVpnTargetNetwork(ctx, &awsec2.ApplySecurityGroupsToClientVpnTargetNetworkInput{
		ClientVpnEndpointId: aws.String(endpointID),
		VpcId:               aws.String("vpc-00000001"),
		SecurityGroupIds:    []string{"sg-00000000"},
	})
	if err != nil || len(applySecurityGroupsOut.SecurityGroupIds) != 1 || applySecurityGroupsOut.SecurityGroupIds[0] != "sg-00000000" {
		t.Fatalf("apply security groups to client vpn target network: %v", err)
	}

	createRouteOut, err := client.CreateClientVpnRoute(ctx, &awsec2.CreateClientVpnRouteInput{
		ClientVpnEndpointId:  aws.String(endpointID),
		DestinationCidrBlock: aws.String("10.240.0.0/16"),
		TargetVpcSubnetId:    aws.String("subnet-00000001"),
		Description:          aws.String("stage21-route"),
	})
	if err != nil || createRouteOut.Status == nil || createRouteOut.Status.Code != awsec2types.ClientVpnRouteStatusCodeActive {
		t.Fatalf("create client vpn route: %v", err)
	}

	describeRoutesOut, err := client.DescribeClientVpnRoutes(ctx, &awsec2.DescribeClientVpnRoutesInput{
		ClientVpnEndpointId: aws.String(endpointID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("destination-cidr"), Values: []string{"10.240.0.0/16"}},
			{Name: aws.String("origin"), Values: []string{"add-route"}},
			{Name: aws.String("target-subnet"), Values: []string{"subnet-00000001"}},
		},
	})
	if err != nil || len(describeRoutesOut.Routes) != 1 || aws.ToString(describeRoutesOut.Routes[0].DestinationCidr) != "10.240.0.0/16" {
		t.Fatalf("describe client vpn routes: %v", err)
	}

	authorizeIngressOut, err := client.AuthorizeClientVpnIngress(ctx, &awsec2.AuthorizeClientVpnIngressInput{
		ClientVpnEndpointId: aws.String(endpointID),
		TargetNetworkCidr:   aws.String("10.240.0.0/16"),
		AccessGroupId:       aws.String("grp-stage21"),
		Description:         aws.String("stage21-rule"),
	})
	if err != nil || authorizeIngressOut.Status == nil || authorizeIngressOut.Status.Code != awsec2types.ClientVpnAuthorizationRuleStatusCodeActive {
		t.Fatalf("authorize client vpn ingress: %v", err)
	}

	describeAuthorizationRulesOut, err := client.DescribeClientVpnAuthorizationRules(ctx, &awsec2.DescribeClientVpnAuthorizationRulesInput{
		ClientVpnEndpointId: aws.String(endpointID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("description"), Values: []string{"stage21-rule"}},
			{Name: aws.String("destination-cidr"), Values: []string{"10.240.0.0/16"}},
			{Name: aws.String("group-id"), Values: []string{"grp-stage21"}},
		},
	})
	if err != nil || len(describeAuthorizationRulesOut.AuthorizationRules) != 1 || aws.ToString(describeAuthorizationRulesOut.AuthorizationRules[0].DestinationCidr) != "10.240.0.0/16" {
		t.Fatalf("describe client vpn authorization rules: %v", err)
	}

	describeConnectionsOut, err := client.DescribeClientVpnConnections(ctx, &awsec2.DescribeClientVpnConnectionsInput{
		ClientVpnEndpointId: aws.String(endpointID),
	})
	if err != nil || len(describeConnectionsOut.Connections) == 0 || describeConnectionsOut.Connections[0].ConnectionId == nil {
		t.Fatalf("describe client vpn connections: %v", err)
	}
	connectionID := aws.ToString(describeConnectionsOut.Connections[0].ConnectionId)

	terminateConnectionsOut, err := client.TerminateClientVpnConnections(ctx, &awsec2.TerminateClientVpnConnectionsInput{
		ClientVpnEndpointId: aws.String(endpointID),
		ConnectionId:        aws.String(connectionID),
	})
	if err != nil || len(terminateConnectionsOut.ConnectionStatuses) != 1 || aws.ToString(terminateConnectionsOut.ConnectionStatuses[0].ConnectionId) != connectionID || terminateConnectionsOut.ConnectionStatuses[0].CurrentStatus == nil || terminateConnectionsOut.ConnectionStatuses[0].CurrentStatus.Code != awsec2types.ClientVpnConnectionStatusCodeTerminated {
		t.Fatalf("terminate client vpn connections: %v", err)
	}

	describeTargetNetworksOut, err := client.DescribeClientVpnTargetNetworks(ctx, &awsec2.DescribeClientVpnTargetNetworksInput{
		ClientVpnEndpointId: aws.String(endpointID),
		AssociationIds:      []string{associationID},
		Filters: []awsec2types.Filter{
			{Name: aws.String("association-id"), Values: []string{associationID}},
			{Name: aws.String("target-network-id"), Values: []string{"subnet-00000001"}},
			{Name: aws.String("vpc-id"), Values: []string{"vpc-00000001"}},
		},
	})
	if err != nil || len(describeTargetNetworksOut.ClientVpnTargetNetworks) != 1 || aws.ToString(describeTargetNetworksOut.ClientVpnTargetNetworks[0].AssociationId) != associationID {
		t.Fatalf("describe client vpn target networks: %v", err)
	}

	exportClientConfigurationOut, err := client.ExportClientVpnClientConfiguration(ctx, &awsec2.ExportClientVpnClientConfigurationInput{
		ClientVpnEndpointId: aws.String(endpointID),
	})
	if err != nil || exportClientConfigurationOut.ClientConfiguration == nil || !strings.Contains(aws.ToString(exportClientConfigurationOut.ClientConfiguration), endpointID) {
		t.Fatalf("export client vpn client configuration: %v", err)
	}

	exportCertificateRevocationListOut, err := client.ExportClientVpnClientCertificateRevocationList(ctx, &awsec2.ExportClientVpnClientCertificateRevocationListInput{
		ClientVpnEndpointId: aws.String(endpointID),
	})
	if err != nil || exportCertificateRevocationListOut.CertificateRevocationList == nil || exportCertificateRevocationListOut.Status == nil || exportCertificateRevocationListOut.Status.Code != awsec2types.ClientCertificateRevocationListStatusCodeActive {
		t.Fatalf("export client vpn client certificate revocation list: %v", err)
	}

	importCertificateRevocationListOut, err := client.ImportClientVpnClientCertificateRevocationList(ctx, &awsec2.ImportClientVpnClientCertificateRevocationListInput{
		ClientVpnEndpointId:       aws.String(endpointID),
		CertificateRevocationList: aws.String("-----BEGIN X509 CRL-----\nSTAGE21\n-----END X509 CRL-----"),
	})
	if err != nil || importCertificateRevocationListOut.Return == nil || !aws.ToBool(importCertificateRevocationListOut.Return) {
		t.Fatalf("import client vpn client certificate revocation list: %v", err)
	}

	revokeIngressOut, err := client.RevokeClientVpnIngress(ctx, &awsec2.RevokeClientVpnIngressInput{
		ClientVpnEndpointId: aws.String(endpointID),
		TargetNetworkCidr:   aws.String("10.240.0.0/16"),
		AccessGroupId:       aws.String("grp-stage21"),
	})
	if err != nil || revokeIngressOut.Status == nil || revokeIngressOut.Status.Code != awsec2types.ClientVpnAuthorizationRuleStatusCodeRevoking {
		t.Fatalf("revoke client vpn ingress: %v", err)
	}

	deleteRouteOut, err := client.DeleteClientVpnRoute(ctx, &awsec2.DeleteClientVpnRouteInput{
		ClientVpnEndpointId:  aws.String(endpointID),
		DestinationCidrBlock: aws.String("10.240.0.0/16"),
		TargetVpcSubnetId:    aws.String("subnet-00000001"),
	})
	if err != nil || deleteRouteOut.Status == nil || deleteRouteOut.Status.Code != awsec2types.ClientVpnRouteStatusCodeDeleting {
		t.Fatalf("delete client vpn route: %v", err)
	}

	disassociateTargetNetworkOut, err := client.DisassociateClientVpnTargetNetwork(ctx, &awsec2.DisassociateClientVpnTargetNetworkInput{
		ClientVpnEndpointId: aws.String(endpointID),
		AssociationId:       aws.String(associationID),
	})
	if err != nil || disassociateTargetNetworkOut.AssociationId == nil || aws.ToString(disassociateTargetNetworkOut.AssociationId) != associationID || disassociateTargetNetworkOut.Status == nil || disassociateTargetNetworkOut.Status.Code != awsec2types.AssociationStatusCodeDisassociated {
		t.Fatalf("disassociate client vpn target network: %v", err)
	}

	deleteEndpointOut, err := client.DeleteClientVpnEndpoint(ctx, &awsec2.DeleteClientVpnEndpointInput{
		ClientVpnEndpointId: aws.String(endpointID),
	})
	if err != nil || deleteEndpointOut.Status == nil || deleteEndpointOut.Status.Code != awsec2types.ClientVpnEndpointStatusCodeDeleted {
		t.Fatalf("delete client vpn endpoint: %v", err)
	}

	describeEndpointsAfterDeleteOut, err := client.DescribeClientVpnEndpoints(ctx, &awsec2.DescribeClientVpnEndpointsInput{
		ClientVpnEndpointIds: []string{endpointID},
	})
	if err != nil || len(describeEndpointsAfterDeleteOut.ClientVpnEndpoints) != 1 || describeEndpointsAfterDeleteOut.ClientVpnEndpoints[0].Status == nil || describeEndpointsAfterDeleteOut.ClientVpnEndpoints[0].Status.Code != awsec2types.ClientVpnEndpointStatusCodeDeleted {
		t.Fatalf("describe client vpn endpoints after delete: %v", err)
	}
}

func TestEC2Stage21ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ApplySecurityGroupsToClientVpnTargetNetwork",
		"AssociateClientVpnTargetNetwork",
		"AuthorizeClientVpnIngress",
		"CreateClientVpnEndpoint",
		"CreateClientVpnRoute",
		"DeleteClientVpnEndpoint",
		"DeleteClientVpnRoute",
		"DescribeClientVpnAuthorizationRules",
		"DescribeClientVpnConnections",
		"DescribeClientVpnEndpoints",
		"DescribeClientVpnRoutes",
		"DescribeClientVpnTargetNetworks",
		"DisassociateClientVpnTargetNetwork",
		"ExportClientVpnClientCertificateRevocationList",
		"ExportClientVpnClientConfiguration",
		"ImportClientVpnClientCertificateRevocationList",
		"ModifyClientVpnEndpoint",
		"RevokeClientVpnIngress",
		"TerminateClientVpnConnections",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "ApplySecurityGroupsToClientVpnTargetNetwork":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["VpcId"] = "vpc-00000001"
			params["SecurityGroupId.1"] = "sg-00000000"
		case "AssociateClientVpnTargetNetwork":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["SubnetId"] = "subnet-00000001"
		case "AuthorizeClientVpnIngress":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["TargetNetworkCidr"] = "10.240.0.0/16"
			params["AccessGroupId"] = "grp-stage21"
		case "CreateClientVpnEndpoint":
			params["Authentication.1.Type"] = "certificate-authentication"
			params["ClientCidrBlock"] = "172.16.0.0/22"
			params["ConnectionLogOptions.Enabled"] = "false"
			params["ServerCertificateArn"] = "arn:aws:acm:us-east-1:123456789012:certificate/server"
		case "CreateClientVpnRoute":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["DestinationCidrBlock"] = "10.240.0.0/16"
			params["TargetVpcSubnetId"] = "subnet-00000001"
		case "DeleteClientVpnEndpoint":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
		case "DeleteClientVpnRoute":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["DestinationCidrBlock"] = "10.240.0.0/16"
			params["TargetVpcSubnetId"] = "subnet-00000001"
		case "DescribeClientVpnAuthorizationRules", "DescribeClientVpnConnections", "DescribeClientVpnRoutes", "DescribeClientVpnTargetNetworks", "ExportClientVpnClientCertificateRevocationList", "ExportClientVpnClientConfiguration", "ModifyClientVpnEndpoint", "TerminateClientVpnConnections":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
		case "DescribeClientVpnEndpoints":
			params["ClientVpnEndpointId.1"] = "cvpn-endpoint-00000001"
		case "DisassociateClientVpnTargetNetwork":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["AssociationId"] = "cvpn-assoc-00000001"
		case "ImportClientVpnClientCertificateRevocationList":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["CertificateRevocationList"] = "-----BEGIN X509 CRL-----\nTEST\n-----END X509 CRL-----"
		case "RevokeClientVpnIngress":
			params["ClientVpnEndpointId"] = "cvpn-endpoint-00000001"
			params["TargetNetworkCidr"] = "10.240.0.0/16"
			params["AccessGroupId"] = "grp-stage21"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
