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

func TestEC2Stage74SDKLifecycle(t *testing.T) {
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

	getInitialStatusOut, err := client.GetSerialConsoleAccessStatus(ctx, &awsec2.GetSerialConsoleAccessStatusInput{})
	if err != nil {
		t.Fatalf("get serial console access status: %v", err)
	}
	if aws.ToBool(getInitialStatusOut.SerialConsoleAccessEnabled) {
		t.Fatalf("expected serial console access to be disabled initially")
	}
	if getInitialStatusOut.ManagedBy != awsec2types.ManagedByAccount {
		t.Fatalf("unexpected initial serial console managed by: %q", getInitialStatusOut.ManagedBy)
	}

	enableSerialOut, err := client.EnableSerialConsoleAccess(ctx, &awsec2.EnableSerialConsoleAccessInput{})
	if err != nil {
		t.Fatalf("enable serial console access: %v", err)
	}
	if !aws.ToBool(enableSerialOut.SerialConsoleAccessEnabled) {
		t.Fatalf("expected serial console access enabled=true after enable")
	}

	getEnabledStatusOut, err := client.GetSerialConsoleAccessStatus(ctx, &awsec2.GetSerialConsoleAccessStatusInput{})
	if err != nil {
		t.Fatalf("get serial console access status after enable: %v", err)
	}
	if !aws.ToBool(getEnabledStatusOut.SerialConsoleAccessEnabled) {
		t.Fatalf("expected serial console access to be enabled after enable")
	}
	if getEnabledStatusOut.ManagedBy != awsec2types.ManagedByAccount {
		t.Fatalf("unexpected serial console managed by after enable: %q", getEnabledStatusOut.ManagedBy)
	}

	disableSerialOut, err := client.DisableSerialConsoleAccess(ctx, &awsec2.DisableSerialConsoleAccessInput{})
	if err != nil {
		t.Fatalf("disable serial console access: %v", err)
	}
	if aws.ToBool(disableSerialOut.SerialConsoleAccessEnabled) {
		t.Fatalf("expected serial console access enabled=false after disable")
	}

	enableIPAMOut, err := client.EnableIpamOrganizationAdminAccount(ctx, &awsec2.EnableIpamOrganizationAdminAccountInput{
		DelegatedAdminAccountId: aws.String("111122223333"),
	})
	if err != nil {
		t.Fatalf("enable ipam organization admin account: %v", err)
	}
	if !aws.ToBool(enableIPAMOut.Success) {
		t.Fatalf("expected enable ipam organization admin account success=true")
	}

	disableIPAMOut, err := client.DisableIpamOrganizationAdminAccount(ctx, &awsec2.DisableIpamOrganizationAdminAccountInput{
		DelegatedAdminAccountId: aws.String("111122223333"),
	})
	if err != nil {
		t.Fatalf("disable ipam organization admin account: %v", err)
	}
	if !aws.ToBool(disableIPAMOut.Success) {
		t.Fatalf("expected disable ipam organization admin account success=true")
	}
}

func TestEC2Stage74ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DisableIpamOrganizationAdminAccount",
		"DisableSerialConsoleAccess",
		"EnableIpamOrganizationAdminAccount",
		"EnableSerialConsoleAccess",
		"GetSerialConsoleAccessStatus",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		if action == "EnableIpamOrganizationAdminAccount" || action == "DisableIpamOrganizationAdminAccount" {
			params["DelegatedAdminAccountId"] = "111122223333"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
