package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lightsail"
	"github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	zone := getenv("STACKYARD_AVAILABILITY_ZONE", "us-east-1a")
	instance := getenv("STACKYARD_INSTANCE_NAME", "lightsail-instance")
	snapshot := getenv("STACKYARD_SNAPSHOT_NAME", "lightsail-snapshot")
	staticIP := getenv("STACKYARD_STATIC_IP_NAME", "lightsail-ip")

	ctx := context.Background()
	client := newLightsailClient(ctx, endpoint)

	fmt.Printf("Stackyard Lightsail advanced client using %s\n", endpoint)

	createOut, err := client.CreateInstances(ctx, &lightsail.CreateInstancesInput{
		AvailabilityZone: aws.String(zone),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{instance},
		Tags:             []types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}},
	})
	if err != nil {
		exitf("create instance: %v", err)
	}
	logf("created instance: %s", instance)

	if _, err := client.StopInstance(ctx, &lightsail.StopInstanceInput{InstanceName: aws.String(instance)}); err != nil {
		exitf("stop instance: %v", err)
	}
	if _, err := client.StartInstance(ctx, &lightsail.StartInstanceInput{InstanceName: aws.String(instance)}); err != nil {
		exitf("start instance: %v", err)
	}
	if _, err := client.RebootInstance(ctx, &lightsail.RebootInstanceInput{InstanceName: aws.String(instance)}); err != nil {
		exitf("reboot instance: %v", err)
	}

	if _, err := client.CreateInstanceSnapshot(ctx, &lightsail.CreateInstanceSnapshotInput{
		InstanceName:         aws.String(instance),
		InstanceSnapshotName: aws.String(snapshot),
	}); err != nil {
		exitf("create snapshot: %v", err)
	}
	if _, err := client.GetInstanceSnapshot(ctx, &lightsail.GetInstanceSnapshotInput{
		InstanceSnapshotName: aws.String(snapshot),
	}); err != nil {
		exitf("get snapshot: %v", err)
	}
	if _, err := client.GetInstanceSnapshots(ctx, &lightsail.GetInstanceSnapshotsInput{}); err != nil {
		exitf("list snapshots: %v", err)
	}

	if _, err := client.AllocateStaticIp(ctx, &lightsail.AllocateStaticIpInput{StaticIpName: aws.String(staticIP)}); err != nil {
		exitf("allocate static ip: %v", err)
	}
	if _, err := client.AttachStaticIp(ctx, &lightsail.AttachStaticIpInput{
		StaticIpName: aws.String(staticIP),
		InstanceName: aws.String(instance),
	}); err != nil {
		exitf("attach static ip: %v", err)
	}
	if _, err := client.DetachStaticIp(ctx, &lightsail.DetachStaticIpInput{StaticIpName: aws.String(staticIP)}); err != nil {
		exitf("detach static ip: %v", err)
	}
	if _, err := client.ReleaseStaticIp(ctx, &lightsail.ReleaseStaticIpInput{StaticIpName: aws.String(staticIP)}); err != nil {
		exitf("release static ip: %v", err)
	}

	if _, err := client.TagResource(ctx, &lightsail.TagResourceInput{
		ResourceName: aws.String(instance),
		Tags: []types.Tag{
			{Key: aws.String("team"), Value: aws.String("platform")},
		},
	}); err != nil {
		exitf("tag resource: %v", err)
	}
	if _, err := client.UntagResource(ctx, &lightsail.UntagResourceInput{
		ResourceName: aws.String(instance),
		TagKeys:      []string{"team"},
	}); err != nil {
		exitf("untag resource: %v", err)
	}

	regionsOut, err := client.GetRegions(ctx, &lightsail.GetRegionsInput{
		IncludeAvailabilityZones:                   aws.Bool(true),
		IncludeRelationalDatabaseAvailabilityZones: aws.Bool(true),
	})
	if err != nil {
		exitf("get regions: %v", err)
	}
	logf("regions: %d", len(regionsOut.Regions))

	opsOut, err := client.GetOperationsForResource(ctx, &lightsail.GetOperationsForResourceInput{
		ResourceName: aws.String(instance),
	})
	if err != nil {
		exitf("get operations for resource: %v", err)
	}
	logf("operations for %s: %d", instance, len(opsOut.Operations))

	if len(createOut.Operations) > 0 && createOut.Operations[0].Id != nil {
		if _, err := client.GetOperation(ctx, &lightsail.GetOperationInput{OperationId: createOut.Operations[0].Id}); err != nil {
			exitf("get operation: %v", err)
		}
	}
	if _, err := client.GetOperations(ctx, &lightsail.GetOperationsInput{}); err != nil {
		exitf("get operations: %v", err)
	}

	if _, err := client.DeleteInstanceSnapshot(ctx, &lightsail.DeleteInstanceSnapshotInput{
		InstanceSnapshotName: aws.String(snapshot),
	}); err != nil {
		exitf("delete snapshot: %v", err)
	}
	if _, err := client.DeleteInstance(ctx, &lightsail.DeleteInstanceInput{
		InstanceName: aws.String(instance),
	}); err != nil {
		exitf("delete instance: %v", err)
	}

	fmt.Println("Done.")
}

func newLightsailClient(ctx context.Context, endpoint string) *lightsail.Client {
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

	return lightsail.NewFromConfig(cfg, func(o *lightsail.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
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
