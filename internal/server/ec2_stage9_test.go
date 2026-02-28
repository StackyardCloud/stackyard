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

func TestEC2Stage9SDKLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-stage9"),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})
	if err != nil || len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("run instances: %v", err)
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	describeTypeOut, err := client.DescribeInstanceAttribute(ctx, &awsec2.DescribeInstanceAttributeInput{
		InstanceId: aws.String(instanceID),
		Attribute:  awsec2types.InstanceAttributeNameInstanceType,
	})
	if err != nil || describeTypeOut.InstanceType == nil || aws.ToString(describeTypeOut.InstanceType.Value) != "t3.micro" {
		t.Fatalf("describe instance type attribute: %v", err)
	}

	if _, err := client.ModifyInstanceAttribute(ctx, &awsec2.ModifyInstanceAttributeInput{
		InstanceId:   aws.String(instanceID),
		Attribute:    awsec2types.InstanceAttributeNameInstanceType,
		InstanceType: &awsec2types.AttributeValue{Value: aws.String("t3.small")},
	}); err != nil {
		t.Fatalf("modify instance type: %v", err)
	}
	describeTypeOutAfterModify, err := client.DescribeInstanceAttribute(ctx, &awsec2.DescribeInstanceAttributeInput{
		InstanceId: aws.String(instanceID),
		Attribute:  awsec2types.InstanceAttributeNameInstanceType,
	})
	if err != nil || describeTypeOutAfterModify.InstanceType == nil || aws.ToString(describeTypeOutAfterModify.InstanceType.Value) != "t3.small" {
		t.Fatalf("describe instance type after modify: %v", err)
	}

	if _, err := client.ModifyInstanceAttribute(ctx, &awsec2.ModifyInstanceAttributeInput{
		InstanceId:      aws.String(instanceID),
		Attribute:       awsec2types.InstanceAttributeNameSourceDestCheck,
		SourceDestCheck: &awsec2types.AttributeBooleanValue{Value: aws.Bool(false)},
	}); err != nil {
		t.Fatalf("modify source dest check: %v", err)
	}
	describeSourceDestOut, err := client.DescribeInstanceAttribute(ctx, &awsec2.DescribeInstanceAttributeInput{
		InstanceId: aws.String(instanceID),
		Attribute:  awsec2types.InstanceAttributeNameSourceDestCheck,
	})
	if err != nil || describeSourceDestOut.SourceDestCheck == nil || aws.ToBool(describeSourceDestOut.SourceDestCheck.Value) {
		t.Fatalf("describe source dest check after modify: %v", err)
	}

	if _, err := client.ResetInstanceAttribute(ctx, &awsec2.ResetInstanceAttributeInput{
		InstanceId: aws.String(instanceID),
		Attribute:  awsec2types.InstanceAttributeNameSourceDestCheck,
	}); err != nil {
		t.Fatalf("reset source dest check: %v", err)
	}
	describeSourceDestOutAfterReset, err := client.DescribeInstanceAttribute(ctx, &awsec2.DescribeInstanceAttributeInput{
		InstanceId: aws.String(instanceID),
		Attribute:  awsec2types.InstanceAttributeNameSourceDestCheck,
	})
	if err != nil || describeSourceDestOutAfterReset.SourceDestCheck == nil || !aws.ToBool(describeSourceDestOutAfterReset.SourceDestCheck.Value) {
		t.Fatalf("describe source dest check after reset: %v", err)
	}

	monitorOut, err := client.MonitorInstances(ctx, &awsec2.MonitorInstancesInput{InstanceIds: []string{instanceID}})
	if err != nil || len(monitorOut.InstanceMonitorings) != 1 || monitorOut.InstanceMonitorings[0].Monitoring == nil || string(monitorOut.InstanceMonitorings[0].Monitoring.State) != "enabled" {
		t.Fatalf("monitor instances: %v", err)
	}

	unmonitorOut, err := client.UnmonitorInstances(ctx, &awsec2.UnmonitorInstancesInput{InstanceIds: []string{instanceID}})
	if err != nil || len(unmonitorOut.InstanceMonitorings) != 1 || unmonitorOut.InstanceMonitorings[0].Monitoring == nil || string(unmonitorOut.InstanceMonitorings[0].Monitoring.State) != "disabled" {
		t.Fatalf("unmonitor instances: %v", err)
	}
}

func TestEC2Stage9ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeInstanceAttribute",
		"ModifyInstanceAttribute",
		"ResetInstanceAttribute",
		"MonitorInstances",
		"UnmonitorInstances",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DescribeInstanceAttribute":
			params["InstanceId"] = "i-00000001"
			params["Attribute"] = "instanceType"
		case "ModifyInstanceAttribute":
			params["InstanceId"] = "i-00000001"
			params["Attribute"] = "instanceType"
			params["InstanceType.Value"] = "t3.small"
		case "ResetInstanceAttribute":
			params["InstanceId"] = "i-00000001"
			params["Attribute"] = "sourceDestCheck"
		case "MonitorInstances", "UnmonitorInstances":
			params["InstanceId.1"] = "i-0000000" + strconv.Itoa(idx+1)
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
