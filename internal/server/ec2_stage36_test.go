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

func TestEC2Stage36SDKLifecycle(t *testing.T) {
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

	createRouteOut, err := client.CreateTransitGatewayRouteTable(ctx, &awsec2.CreateTransitGatewayRouteTableInput{
		TransitGatewayId: aws.String("tgw-00000036"),
	})
	if err != nil || createRouteOut.TransitGatewayRouteTable == nil || createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId == nil {
		t.Fatalf("create transit gateway route table: %v", err)
	}
	routeTableID := aws.ToString(createRouteOut.TransitGatewayRouteTable.TransitGatewayRouteTableId)

	createPolicyOut, err := client.CreateTransitGatewayPolicyTable(ctx, &awsec2.CreateTransitGatewayPolicyTableInput{
		TransitGatewayId: aws.String("tgw-00000036"),
	})
	if err != nil || createPolicyOut.TransitGatewayPolicyTable == nil || createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId == nil {
		t.Fatalf("create transit gateway policy table: %v", err)
	}
	policyTableID := aws.ToString(createPolicyOut.TransitGatewayPolicyTable.TransitGatewayPolicyTableId)

	if _, err := client.AssociateTransitGatewayPolicyTable(ctx, &awsec2.AssociateTransitGatewayPolicyTableInput{
		TransitGatewayPolicyTableId: aws.String(policyTableID),
		TransitGatewayAttachmentId:  aws.String("tgw-attach-00000036"),
	}); err != nil {
		t.Fatalf("associate transit gateway policy table: %v", err)
	}
	if _, err := client.AssociateTransitGatewayRouteTable(ctx, &awsec2.AssociateTransitGatewayRouteTableInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000036"),
	}); err != nil {
		t.Fatalf("associate transit gateway route table: %v", err)
	}

	createReferenceOneOut, err := client.CreateTransitGatewayPrefixListReference(ctx, &awsec2.CreateTransitGatewayPrefixListReferenceInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		PrefixListId:               aws.String("pl-00000036"),
		TransitGatewayAttachmentId: aws.String("tgw-attach-00000036"),
	})
	if err != nil || createReferenceOneOut.TransitGatewayPrefixListReference == nil || createReferenceOneOut.TransitGatewayPrefixListReference.State != awsec2types.TransitGatewayPrefixListReferenceStateAvailable {
		t.Fatalf("create transit gateway prefix list reference #1: %v", err)
	}
	if _, err := client.CreateTransitGatewayPrefixListReference(ctx, &awsec2.CreateTransitGatewayPrefixListReferenceInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		PrefixListId:               aws.String("pl-00000037"),
		Blackhole:                  aws.Bool(true),
	}); err != nil {
		t.Fatalf("create transit gateway prefix list reference #2: %v", err)
	}

	getRefsPageOneOut, err := client.GetTransitGatewayPrefixListReferences(ctx, &awsec2.GetTransitGatewayPrefixListReferencesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		MaxResults:                 aws.Int32(1),
	})
	if err != nil || len(getRefsPageOneOut.TransitGatewayPrefixListReferences) != 1 || getRefsPageOneOut.NextToken == nil {
		t.Fatalf("get transit gateway prefix list references page 1: %v", err)
	}
	getRefsPageTwoOut, err := client.GetTransitGatewayPrefixListReferences(ctx, &awsec2.GetTransitGatewayPrefixListReferencesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		MaxResults:                 aws.Int32(1),
		NextToken:                  getRefsPageOneOut.NextToken,
	})
	if err != nil || len(getRefsPageTwoOut.TransitGatewayPrefixListReferences) != 1 {
		t.Fatalf("get transit gateway prefix list references page 2: %v", err)
	}
	getBlackholeRefsOut, err := client.GetTransitGatewayPrefixListReferences(ctx, &awsec2.GetTransitGatewayPrefixListReferencesInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		Filters: []awsec2types.Filter{
			{Name: aws.String("is-blackhole"), Values: []string{"true"}},
		},
	})
	if err != nil || len(getBlackholeRefsOut.TransitGatewayPrefixListReferences) != 1 || aws.ToBool(getBlackholeRefsOut.TransitGatewayPrefixListReferences[0].Blackhole) != true {
		t.Fatalf("get transit gateway prefix list references with filters: %v", err)
	}

	deleteReferenceOut, err := client.DeleteTransitGatewayPrefixListReference(ctx, &awsec2.DeleteTransitGatewayPrefixListReferenceInput{
		TransitGatewayRouteTableId: aws.String(routeTableID),
		PrefixListId:               aws.String("pl-00000036"),
	})
	if err != nil || deleteReferenceOut.TransitGatewayPrefixListReference == nil || deleteReferenceOut.TransitGatewayPrefixListReference.State != awsec2types.TransitGatewayPrefixListReferenceStateDeleting {
		t.Fatalf("delete transit gateway prefix list reference: %v", err)
	}

	getPolicyEntriesOut, err := client.GetTransitGatewayPolicyTableEntries(ctx, &awsec2.GetTransitGatewayPolicyTableEntriesInput{
		TransitGatewayPolicyTableId: aws.String(policyTableID),
	})
	if err != nil || len(getPolicyEntriesOut.TransitGatewayPolicyTableEntries) != 1 {
		t.Fatalf("get transit gateway policy table entries: %v", err)
	}
	entry := getPolicyEntriesOut.TransitGatewayPolicyTableEntries[0]
	if entry.PolicyRuleNumber == nil || entry.PolicyRule == nil || entry.TargetRouteTableId == nil || aws.ToString(entry.TargetRouteTableId) != routeTableID {
		t.Fatalf("unexpected transit gateway policy table entry payload")
	}
}

func TestEC2Stage36ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CreateTransitGatewayPrefixListReference",
		"DeleteTransitGatewayPrefixListReference",
		"GetTransitGatewayPrefixListReferences",
		"GetTransitGatewayPolicyTableEntries",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CreateTransitGatewayPrefixListReference":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000036"
			params["PrefixListId"] = "pl-00000036"
			params["TransitGatewayAttachmentId"] = "tgw-attach-00000036"
		case "DeleteTransitGatewayPrefixListReference":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000036"
			params["PrefixListId"] = "pl-00000036"
		case "GetTransitGatewayPrefixListReferences":
			params["TransitGatewayRouteTableId"] = "tgw-rtb-00000036"
		case "GetTransitGatewayPolicyTableEntries":
			params["TransitGatewayPolicyTableId"] = "tgw-ptb-00000036"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
