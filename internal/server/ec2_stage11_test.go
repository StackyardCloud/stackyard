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
)

func TestEC2Stage11SDKLifecycle(t *testing.T) {
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

	getBeforeOut, err := client.GetEbsEncryptionByDefault(ctx, &awsec2.GetEbsEncryptionByDefaultInput{})
	if err != nil || aws.ToBool(getBeforeOut.EbsEncryptionByDefault) {
		t.Fatalf("get ebs encryption by default before enable: %v", err)
	}

	enableOut, err := client.EnableEbsEncryptionByDefault(ctx, &awsec2.EnableEbsEncryptionByDefaultInput{})
	if err != nil || !aws.ToBool(enableOut.EbsEncryptionByDefault) {
		t.Fatalf("enable ebs encryption by default: %v", err)
	}

	getAfterEnableOut, err := client.GetEbsEncryptionByDefault(ctx, &awsec2.GetEbsEncryptionByDefaultInput{})
	if err != nil || !aws.ToBool(getAfterEnableOut.EbsEncryptionByDefault) {
		t.Fatalf("get ebs encryption by default after enable: %v", err)
	}

	customKMS := "arn:aws:kms:us-east-1:123456789012:key/stage11-sdk"
	modifyKMSOut, err := client.ModifyEbsDefaultKmsKeyId(ctx, &awsec2.ModifyEbsDefaultKmsKeyIdInput{
		KmsKeyId: aws.String(customKMS),
	})
	if err != nil || aws.ToString(modifyKMSOut.KmsKeyId) != customKMS {
		t.Fatalf("modify ebs default kms key id: %v", err)
	}

	getKMSOut, err := client.GetEbsDefaultKmsKeyId(ctx, &awsec2.GetEbsDefaultKmsKeyIdInput{})
	if err != nil || aws.ToString(getKMSOut.KmsKeyId) != customKMS {
		t.Fatalf("get ebs default kms key id: %v", err)
	}

	resetKMSOut, err := client.ResetEbsDefaultKmsKeyId(ctx, &awsec2.ResetEbsDefaultKmsKeyIdInput{})
	if err != nil || aws.ToString(resetKMSOut.KmsKeyId) == "" || aws.ToString(resetKMSOut.KmsKeyId) == customKMS {
		t.Fatalf("reset ebs default kms key id: %v", err)
	}

	getKMSAfterResetOut, err := client.GetEbsDefaultKmsKeyId(ctx, &awsec2.GetEbsDefaultKmsKeyIdInput{})
	if err != nil || aws.ToString(getKMSAfterResetOut.KmsKeyId) != aws.ToString(resetKMSOut.KmsKeyId) {
		t.Fatalf("get ebs default kms key id after reset: %v", err)
	}

	disableOut, err := client.DisableEbsEncryptionByDefault(ctx, &awsec2.DisableEbsEncryptionByDefaultInput{})
	if err != nil || aws.ToBool(disableOut.EbsEncryptionByDefault) {
		t.Fatalf("disable ebs encryption by default: %v", err)
	}

	getAfterDisableOut, err := client.GetEbsEncryptionByDefault(ctx, &awsec2.GetEbsEncryptionByDefaultInput{})
	if err != nil || aws.ToBool(getAfterDisableOut.EbsEncryptionByDefault) {
		t.Fatalf("get ebs encryption by default after disable: %v", err)
	}
}

func TestEC2Stage11ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"EnableEbsEncryptionByDefault",
		"DisableEbsEncryptionByDefault",
		"GetEbsEncryptionByDefault",
		"GetEbsDefaultKmsKeyId",
		"ModifyEbsDefaultKmsKeyId",
		"ResetEbsDefaultKmsKeyId",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		if action == "ModifyEbsDefaultKmsKeyId" {
			params["KmsKeyId"] = "arn:aws:kms:us-east-1:123456789012:key/stage11-notimplemented"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
