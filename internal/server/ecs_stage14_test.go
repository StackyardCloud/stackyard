package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	awsecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestECSStage14OperationCoverageImplementedVsNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range ecsOperations {
		resp := ecsRequest(t, ts, op.Name, []byte(`{}`))
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("expected %s to be implemented, got %d", op.Name, resp.StatusCode)
		}
	}
}

func TestECSStage14SDKUpdateClusterContainerAgentAndExpressGateway(t *testing.T) {
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
	client := awsecs.NewFromConfig(cfg, func(o *awsecs.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	clusterName := "ecs-stage14-cluster"
	_, err = client.CreateCluster(ctx, &awsecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}

	updateClusterOut, err := client.UpdateCluster(ctx, &awsecs.UpdateClusterInput{
		Cluster: aws.String(clusterName),
		Settings: []awsecstypes.ClusterSetting{
			{
				Name:  awsecstypes.ClusterSettingNameContainerInsights,
				Value: aws.String("enabled"),
			},
		},
		ServiceConnectDefaults: &awsecstypes.ClusterServiceConnectDefaultsRequest{
			Namespace: aws.String("stage14.local"),
		},
		Configuration: &awsecstypes.ClusterConfiguration{},
	})
	if err != nil {
		t.Fatalf("update cluster: %v", err)
	}
	if updateClusterOut.Cluster == nil {
		t.Fatalf("update cluster: expected cluster in response")
	}
	if updateClusterOut.Cluster.ServiceConnectDefaults == nil {
		t.Fatalf("update cluster: expected serviceConnectDefaults in response")
	}
	if got := aws.ToString(updateClusterOut.Cluster.ServiceConnectDefaults.Namespace); got != "stage14.local" {
		t.Fatalf("update cluster: unexpected service connect namespace %q", got)
	}

	registerOut, err := client.RegisterContainerInstance(ctx, &awsecs.RegisterContainerInstanceInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		t.Fatalf("register container instance: %v", err)
	}
	if registerOut.ContainerInstance == nil || registerOut.ContainerInstance.ContainerInstanceArn == nil {
		t.Fatalf("register container instance: missing container instance arn")
	}

	updateAgentOut, err := client.UpdateContainerAgent(ctx, &awsecs.UpdateContainerAgentInput{
		Cluster:           aws.String(clusterName),
		ContainerInstance: registerOut.ContainerInstance.ContainerInstanceArn,
	})
	if err != nil {
		t.Fatalf("update container agent: %v", err)
	}
	if updateAgentOut.ContainerInstance == nil {
		t.Fatalf("update container agent: expected container instance in response")
	}
	if string(updateAgentOut.ContainerInstance.AgentUpdateStatus) != "UPDATED" {
		t.Fatalf("update container agent: unexpected agent update status %q", updateAgentOut.ContainerInstance.AgentUpdateStatus)
	}
	if updateAgentOut.ContainerInstance.Version < 2 {
		t.Fatalf("update container agent: expected incremented version, got %d", updateAgentOut.ContainerInstance.Version)
	}

	createExpressOut, err := client.CreateExpressGatewayService(ctx, &awsecs.CreateExpressGatewayServiceInput{
		Cluster:               aws.String(clusterName),
		ServiceName:           aws.String("stage14-express"),
		ExecutionRoleArn:      aws.String("arn:aws:iam::123456789012:role/ecsTaskExecutionRole"),
		InfrastructureRoleArn: aws.String("arn:aws:iam::123456789012:role/ecsInfrastructureRole"),
		TaskRoleArn:           aws.String("arn:aws:iam::123456789012:role/ecsTaskRole"),
		Cpu:                   aws.String("256"),
		Memory:                aws.String("512"),
		HealthCheckPath:       aws.String("/ping"),
		PrimaryContainer: &awsecstypes.ExpressGatewayContainer{
			Image: aws.String("public.ecr.aws/docker/library/nginx:latest"),
		},
		Tags: []awsecstypes.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	})
	if err != nil {
		t.Fatalf("create express gateway service: %v", err)
	}
	if createExpressOut.Service == nil || createExpressOut.Service.ServiceArn == nil {
		t.Fatalf("create express gateway service: missing service arn")
	}
	serviceARN := aws.ToString(createExpressOut.Service.ServiceArn)

	describeExpressOut, err := client.DescribeExpressGatewayService(ctx, &awsecs.DescribeExpressGatewayServiceInput{
		ServiceArn: aws.String(serviceARN),
		Include:    []awsecstypes.ExpressGatewayServiceInclude{awsecstypes.ExpressGatewayServiceIncludeTags},
	})
	if err != nil {
		t.Fatalf("describe express gateway service: %v", err)
	}
	if describeExpressOut.Service == nil {
		t.Fatalf("describe express gateway service: expected service in response")
	}
	if len(describeExpressOut.Service.Tags) != 1 {
		t.Fatalf("describe express gateway service: expected 1 tag, got %d", len(describeExpressOut.Service.Tags))
	}

	updateExpressOut, err := client.UpdateExpressGatewayService(ctx, &awsecs.UpdateExpressGatewayServiceInput{
		ServiceArn: aws.String(serviceARN),
		Cpu:        aws.String("512"),
	})
	if err != nil {
		t.Fatalf("update express gateway service: %v", err)
	}
	if updateExpressOut.Service == nil || updateExpressOut.Service.TargetConfiguration == nil {
		t.Fatalf("update express gateway service: missing target configuration")
	}
	if got := aws.ToString(updateExpressOut.Service.TargetConfiguration.Cpu); got != "512" {
		t.Fatalf("update express gateway service: expected target cpu 512, got %q", got)
	}

	deleteExpressOut, err := client.DeleteExpressGatewayService(ctx, &awsecs.DeleteExpressGatewayServiceInput{
		ServiceArn: aws.String(serviceARN),
	})
	if err != nil {
		t.Fatalf("delete express gateway service: %v", err)
	}
	if deleteExpressOut.Service == nil || deleteExpressOut.Service.Status == nil {
		t.Fatalf("delete express gateway service: missing service status")
	}
	if deleteExpressOut.Service.Status.StatusCode != awsecstypes.ExpressGatewayServiceStatusCodeInactive {
		t.Fatalf("delete express gateway service: expected INACTIVE status, got %q", deleteExpressOut.Service.Status.StatusCode)
	}
}
