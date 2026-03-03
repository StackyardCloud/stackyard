package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	imageID := getenv("STACKYARD_IMAGE_ID", "ami-12345678")
	securityGroupName := getenv("STACKYARD_SECURITY_GROUP", "ec2-advanced-sg")

	ctx := context.Background()
	client := newEC2Client(ctx, endpoint)

	fmt.Printf("Stackyard EC2 advanced client using %s\n", endpoint)

	createSGOut, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(securityGroupName),
		Description: aws.String("advanced security group"),
		VpcId:       aws.String("vpc-00000001"),
	})
	if err != nil {
		exitf("create security group: %v", err)
	}
	groupID := aws.ToString(createSGOut.GroupId)
	logf("created security group: %s", groupID)

	if _, err := client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(groupID),
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(80),
				ToPort:     aws.Int32(80),
				IpRanges:   []types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
			},
		},
	}); err != nil {
		exitf("authorize security group ingress: %v", err)
	}

	runOut, err := client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:          aws.String(imageID),
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		InstanceType:     types.InstanceTypeT3Micro,
		SecurityGroupIds: []string{groupID},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}},
			},
		},
	})
	if err != nil {
		exitf("run instances: %v", err)
	}
	if len(runOut.Instances) == 0 || runOut.Instances[0].InstanceId == nil {
		exitf("run instances: missing instance id")
	}
	instanceID := aws.ToString(runOut.Instances[0].InstanceId)
	logf("created instance: %s", instanceID)

	if _, err := client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{instanceID},
		Tags:      []types.Tag{{Key: aws.String("team"), Value: aws.String("platform")}},
	}); err != nil {
		exitf("create tags: %v", err)
	}

	tagsOut, err := client.DescribeTags(ctx, &ec2.DescribeTagsInput{})
	if err != nil {
		exitf("describe tags: %v", err)
	}
	logf("tag count: %d", len(tagsOut.Tags))

	createVolumeOut, err := client.CreateVolume(ctx, &ec2.CreateVolumeInput{
		AvailabilityZone: aws.String("us-east-1a"),
		Size:             aws.Int32(10),
		VolumeType:       types.VolumeTypeGp3,
	})
	if err != nil {
		exitf("create volume: %v", err)
	}
	volumeID := aws.ToString(createVolumeOut.VolumeId)

	if _, err := client.AttachVolume(ctx, &ec2.AttachVolumeInput{
		VolumeId:   aws.String(volumeID),
		InstanceId: aws.String(instanceID),
		Device:     aws.String("/dev/xvdf"),
	}); err != nil {
		exitf("attach volume: %v", err)
	}

	if _, err := client.DetachVolume(ctx, &ec2.DetachVolumeInput{
		VolumeId:   aws.String(volumeID),
		InstanceId: aws.String(instanceID),
		Device:     aws.String("/dev/xvdf"),
	}); err != nil {
		exitf("detach volume: %v", err)
	}

	createSnapshotOut, err := client.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{
		VolumeId:    aws.String(volumeID),
		Description: aws.String("advanced snapshot"),
	})
	if err != nil {
		exitf("create snapshot: %v", err)
	}
	snapshotID := aws.ToString(createSnapshotOut.SnapshotId)

	if _, err := client.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{SnapshotIds: []string{snapshotID}}); err != nil {
		exitf("describe snapshots: %v", err)
	}
	if _, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{VolumeIds: []string{volumeID}}); err != nil {
		exitf("describe volumes: %v", err)
	}

	if _, err := client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{InstanceIds: []string{instanceID}, IncludeAllInstances: aws.Bool(true)}); err != nil {
		exitf("describe instance status: %v", err)
	}

	if _, err := client.RebootInstances(ctx, &ec2.RebootInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		exitf("reboot instances: %v", err)
	}

	if _, err := client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{SnapshotId: aws.String(snapshotID)}); err != nil {
		exitf("delete snapshot: %v", err)
	}
	if _, err := client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{VolumeId: aws.String(volumeID)}); err != nil {
		exitf("delete volume: %v", err)
	}
	if _, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{instanceID}}); err != nil {
		exitf("terminate instances: %v", err)
	}
	if _, err := client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: aws.String(groupID)}); err != nil {
		exitf("delete security group: %v", err)
	}

	fmt.Println("Done.")
}

func newEC2Client(ctx context.Context, endpoint string) *ec2.Client {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		)),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	return ec2.NewFromConfig(cfg, func(o *ec2.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
}

func getenv(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
