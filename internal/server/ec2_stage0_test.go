package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func ec2Request(t *testing.T, ts *httptest.Server, action string, params map[string]string) *http.Response {
	t.Helper()
	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2016-11-15")
	for key, value := range params {
		form.Set(key, value)
	}
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(form.Encode()),
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		"ec2",
	)
}

func TestEC2Stage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ec2Request(t, ts, "RunInstances", map[string]string{
		"ImageId":      "ami-12345678",
		"MinCount":     "1",
		"MaxCount":     "1",
		"InstanceType": "t3.micro",
	})
	assertStatus(t, resp, http.StatusOK)

	resp = ec2Request(t, ts, "DescribeInstances", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = ec2Request(t, ts, "DescribeRegions", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = ec2Request(t, ts, "DescribeAvailabilityZones", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = ec2Request(t, ts, "CreateTrafficMirrorFilter", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestEC2Stage0OperationCoverage(t *testing.T) {
	if len(ec2Operations) != 689 {
		t.Fatalf("expected 689 EC2 operations from docs/model, got %d", len(ec2Operations))
	}
	if len(ec2OperationByName) != len(ec2Operations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"RunInstances",
		"DescribeInstances",
		"TerminateInstances",
		"CreateSecurityGroup",
		"DescribeSecurityGroups",
		"CreateVolume",
		"DescribeVolumes",
		"CreateSnapshot",
		"DescribeSnapshots",
		"CreateVpc",
	}
	for _, name := range required {
		if _, ok := ec2OperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestEC2Stage0SDKClientLifecycle(t *testing.T) {
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
		ImageId:      aws.String("ami-12345678"),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		InstanceType: awsec2types.InstanceTypeT3Micro,
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeInstance,
				Tags:         []awsec2types.Tag{{Key: aws.String("env"), Value: aws.String("test")}},
			},
		},
	})
	if err != nil {
		t.Fatalf("run instances: %v", err)
	}
	if len(runOut.Instances) != 1 || runOut.Instances[0].InstanceId == nil {
		t.Fatalf("expected one instance id")
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)

	if _, err := client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}); err != nil {
		t.Fatalf("describe instances: %v", err)
	}

	if _, err := client.CreateTags(ctx, &awsec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      []awsec2types.Tag{{Key: aws.String("team"), Value: aws.String("platform")}},
	}); err != nil {
		t.Fatalf("create tags: %v", err)
	}
	if _, err := client.DescribeTags(ctx, &awsec2.DescribeTagsInput{}); err != nil {
		t.Fatalf("describe tags: %v", err)
	}

	createSGOut, err := client.CreateSecurityGroup(ctx, &awsec2.CreateSecurityGroupInput{
		GroupName:   aws.String("sdk-sg"),
		Description: aws.String("sdk security group"),
		VpcId:       aws.String("vpc-00000001"),
	})
	if err != nil {
		t.Fatalf("create security group: %v", err)
	}
	if createSGOut.GroupId == nil {
		t.Fatalf("expected security group id")
	}
	groupID := aws.ToString(createSGOut.GroupId)

	if _, err := client.AuthorizeSecurityGroupIngress(ctx, &awsec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(groupID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(80),
				ToPort:     aws.Int32(80),
				IpRanges:   []awsec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
		},
	}); err != nil {
		t.Fatalf("authorize security group ingress: %v", err)
	}
	if _, err := client.DescribeSecurityGroups(ctx, &awsec2.DescribeSecurityGroupsInput{GroupIds: []string{groupID}}); err != nil {
		t.Fatalf("describe security groups: %v", err)
	}
	if _, err := client.RevokeSecurityGroupIngress(ctx, &awsec2.RevokeSecurityGroupIngressInput{
		GroupId: aws.String(groupID),
		IpPermissions: []awsec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(80),
				ToPort:     aws.Int32(80),
				IpRanges:   []awsec2types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
		},
	}); err != nil {
		t.Fatalf("revoke security group ingress: %v", err)
	}

	createVolOut, err := client.CreateVolume(ctx, &awsec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(10),
		VolumeType:       awsec2types.VolumeTypeGp3,
	})
	if err != nil {
		t.Fatalf("create volume: %v", err)
	}
	if createVolOut.VolumeId == nil {
		t.Fatalf("expected volume id")
	}
	volumeID := aws.ToString(createVolOut.VolumeId)

	if _, err := client.DescribeVolumes(ctx, &awsec2.DescribeVolumesInput{VolumeIds: []string{volumeID}}); err != nil {
		t.Fatalf("describe volumes: %v", err)
	}
	if _, err := client.AttachVolume(ctx, &awsec2.AttachVolumeInput{
		VolumeId:   aws.String(volumeID),
		InstanceId: aws.String(instanceID),
		Device:     aws.String("/dev/xvdf"),
	}); err != nil {
		t.Fatalf("attach volume: %v", err)
	}
	if _, err := client.DetachVolume(ctx, &awsec2.DetachVolumeInput{
		VolumeId:   aws.String(volumeID),
		InstanceId: aws.String(instanceID),
		Device:     aws.String("/dev/xvdf"),
	}); err != nil {
		t.Fatalf("detach volume: %v", err)
	}

	createSnapOut, err := client.CreateSnapshot(ctx, &awsec2.CreateSnapshotInput{
		VolumeId:    aws.String(volumeID),
		Description: aws.String("sdk snapshot"),
	})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if createSnapOut.SnapshotId == nil {
		t.Fatalf("expected snapshot id")
	}
	snapshotID := aws.ToString(createSnapOut.SnapshotId)

	if _, err := client.DescribeSnapshots(ctx, &awsec2.DescribeSnapshotsInput{SnapshotIds: []string{snapshotID}}); err != nil {
		t.Fatalf("describe snapshots: %v", err)
	}

	if _, err := client.StopInstances(ctx, &awsec2.StopInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		t.Fatalf("stop instances: %v", err)
	}
	if _, err := client.StartInstances(ctx, &awsec2.StartInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		t.Fatalf("start instances: %v", err)
	}
	if _, err := client.RebootInstances(ctx, &awsec2.RebootInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		t.Fatalf("reboot instances: %v", err)
	}
	if _, err := client.DescribeInstanceStatus(ctx, &awsec2.DescribeInstanceStatusInput{InstanceIds: []string{instanceID}, IncludeAllInstances: aws.Bool(true)}); err != nil {
		t.Fatalf("describe instance status: %v", err)
	}

	if _, err := client.DeleteSnapshot(ctx, &awsec2.DeleteSnapshotInput{SnapshotId: aws.String(snapshotID)}); err != nil {
		t.Fatalf("delete snapshot: %v", err)
	}
	if _, err := client.DeleteVolume(ctx, &awsec2.DeleteVolumeInput{VolumeId: aws.String(volumeID)}); err != nil {
		t.Fatalf("delete volume: %v", err)
	}
	if _, err := client.TerminateInstances(ctx, &awsec2.TerminateInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		t.Fatalf("terminate instances: %v", err)
	}
	if _, err := client.DeleteSecurityGroup(ctx, &awsec2.DeleteSecurityGroupInput{GroupId: aws.String(groupID)}); err != nil {
		t.Fatalf("delete security group: %v", err)
	}
}

func TestEC2Stage0AllDocumentedOperationsAreKnown(t *testing.T) {
	for i, op := range ec2Operations {
		if strings.TrimSpace(op.Name) == "" {
			t.Fatalf("empty operation at index %d", i)
		}
		if _, ok := ec2OperationByName[op.Name]; !ok {
			t.Fatalf("operation %s missing from lookup", op.Name)
		}
	}
}

func TestEC2Stage0ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"RunInstances",
		"DescribeInstances",
		"StartInstances",
		"StopInstances",
		"RebootInstances",
		"TerminateInstances",
		"DescribeInstanceStatus",
		"CreateTags",
		"DeleteTags",
		"DescribeTags",
		"DescribeRegions",
		"DescribeAvailabilityZones",
		"CreateSecurityGroup",
		"DescribeSecurityGroups",
		"AuthorizeSecurityGroupIngress",
		"RevokeSecurityGroupIngress",
		"DeleteSecurityGroup",
		"CreateVolume",
		"DescribeVolumes",
		"AttachVolume",
		"DetachVolume",
		"DeleteVolume",
		"CreateSnapshot",
		"DescribeSnapshots",
		"DeleteSnapshot",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for idx, action := range implemented {
		params := map[string]string{}
		switch action {
		case "RunInstances":
			params["ImageId"] = "ami-12345678"
			params["MinCount"] = "1"
			params["MaxCount"] = "1"
		case "StartInstances", "StopInstances", "RebootInstances", "TerminateInstances", "DescribeInstanceStatus":
			params["InstanceId.1"] = "i-00000001"
		case "CreateTags", "DeleteTags":
			params["ResourceId.1"] = "i-00000001"
			params["Tag.1.Key"] = "env"
			params["Tag.1.Value"] = "test"
		case "CreateSecurityGroup":
			params["GroupName"] = "sg-" + strconv.Itoa(idx)
			params["GroupDescription"] = "test"
		case "DescribeSecurityGroups", "AuthorizeSecurityGroupIngress", "RevokeSecurityGroupIngress", "DeleteSecurityGroup":
			params["GroupId"] = "sg-00000000"
			params["IpPermissions.1.IpProtocol"] = "tcp"
			params["IpPermissions.1.FromPort"] = "80"
			params["IpPermissions.1.ToPort"] = "80"
			params["IpPermissions.1.IpRanges.1.CidrIp"] = "0.0.0.0/0"
		case "CreateVolume":
			params["AvailabilityZone"] = "us-east-1a"
			params["Size"] = "8"
		case "DescribeVolumes", "AttachVolume", "DetachVolume", "DeleteVolume":
			params["VolumeId"] = "vol-00000001"
			params["VolumeId.1"] = "vol-00000001"
			params["InstanceId"] = "i-00000001"
			params["Device"] = "/dev/xvdf"
		case "CreateSnapshot":
			params["VolumeId"] = "vol-00000001"
		case "DescribeSnapshots", "DeleteSnapshot":
			params["SnapshotId"] = "snap-00000001"
			params["SnapshotId.1"] = "snap-00000001"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
