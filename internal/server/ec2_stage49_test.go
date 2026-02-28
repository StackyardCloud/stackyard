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

func TestEC2Stage49SDKLifecycle(t *testing.T) {
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

	addPrincipal := "arn:aws:iam::123456789012:root"
	addOut, err := client.ModifyVpcEndpointServicePermissions(ctx, &awsec2.ModifyVpcEndpointServicePermissionsInput{
		ServiceId:            aws.String("vpce-svc-00000000"),
		AddAllowedPrincipals: []string{addPrincipal},
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint service permissions add principal: %v", err)
	}
	if !aws.ToBool(addOut.ReturnValue) {
		t.Fatalf("expected return value true for add")
	}
	if len(addOut.AddedPrincipals) != 1 {
		t.Fatalf("expected one added principal, got %d", len(addOut.AddedPrincipals))
	}
	if aws.ToString(addOut.AddedPrincipals[0].Principal) != addPrincipal {
		t.Fatalf("unexpected added principal: %q", aws.ToString(addOut.AddedPrincipals[0].Principal))
	}
	if addOut.AddedPrincipals[0].PrincipalType != awsec2types.PrincipalTypeAccount {
		t.Fatalf("unexpected principal type: %q", addOut.AddedPrincipals[0].PrincipalType)
	}
	if aws.ToString(addOut.AddedPrincipals[0].ServiceId) != "vpce-svc-00000000" {
		t.Fatalf("unexpected service id on added principal: %q", aws.ToString(addOut.AddedPrincipals[0].ServiceId))
	}
	if aws.ToString(addOut.AddedPrincipals[0].ServicePermissionId) == "" {
		t.Fatalf("expected service permission id")
	}

	removeOut, err := client.ModifyVpcEndpointServicePermissions(ctx, &awsec2.ModifyVpcEndpointServicePermissionsInput{
		ServiceId:               aws.String("vpce-svc-00000000"),
		RemoveAllowedPrincipals: []string{addPrincipal},
	})
	if err != nil {
		t.Fatalf("modify vpc endpoint service permissions remove principal: %v", err)
	}
	if !aws.ToBool(removeOut.ReturnValue) {
		t.Fatalf("expected return value true for remove")
	}
	if len(removeOut.AddedPrincipals) != 0 {
		t.Fatalf("expected no added principals on remove call, got %d", len(removeOut.AddedPrincipals))
	}
}

func TestEC2Stage49ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"ModifyVpcEndpointServicePermissions",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		resp := ec2Request(t, ts, action, map[string]string{
			"ServiceId":               "vpce-svc-00000000",
			"AddAllowedPrincipals.1":  "arn:aws:iam::123456789012:root",
			"RemoveAllowedPrincipals": "",
		})
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
