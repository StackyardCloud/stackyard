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

func TestEC2Stage10SDKLifecycle(t *testing.T) {
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

	runOut, err := client.RunInstances(ctx, &awsec2.RunInstancesInput{
		ImageId:      aws.String("ami-stage10"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	consoleOut, err := client.GetConsoleOutput(ctx, &awsec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
		Latest:     aws.Bool(true),
	})
	if err != nil || consoleOut.InstanceId == nil || aws.ToString(consoleOut.InstanceId) != instanceID || consoleOut.Output == nil || aws.ToString(consoleOut.Output) == "" || consoleOut.Timestamp == nil {
		t.Fatalf("get console output: %v", err)
	}

	screenshotOut, err := client.GetConsoleScreenshot(ctx, &awsec2.GetConsoleScreenshotInput{
		InstanceId: aws.String(instanceID),
		WakeUp:     aws.Bool(true),
	})
	if err != nil || screenshotOut.InstanceId == nil || aws.ToString(screenshotOut.InstanceId) != instanceID || screenshotOut.ImageData == nil || aws.ToString(screenshotOut.ImageData) == "" {
		t.Fatalf("get console screenshot: %v", err)
	}

	passwordOut, err := client.GetPasswordData(ctx, &awsec2.GetPasswordDataInput{
		InstanceId: aws.String(instanceID),
	})
	if err != nil || passwordOut.InstanceId == nil || aws.ToString(passwordOut.InstanceId) != instanceID || passwordOut.PasswordData == nil || aws.ToString(passwordOut.PasswordData) == "" || passwordOut.Timestamp == nil {
		t.Fatalf("get password data: %v", err)
	}
}

func TestEC2Stage10ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"GetConsoleOutput",
		"GetConsoleScreenshot",
		"GetPasswordData",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "GetConsoleOutput":
			params["InstanceId"] = "i-0000000" + strconv.Itoa(idx+1)
			params["Latest"] = "true"
		case "GetConsoleScreenshot":
			params["InstanceId"] = "i-0000000" + strconv.Itoa(idx+1)
			params["WakeUp"] = "true"
		case "GetPasswordData":
			params["InstanceId"] = "i-0000000" + strconv.Itoa(idx+1)
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
