package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	clusterName := getenv("STACKYARD_CLUSTER_NAME", "ecs-basic-cluster")
	taskFamily := getenv("STACKYARD_TASK_FAMILY", "ecs-basic-task")

	ctx := context.Background()
	client := newECSClient(ctx, endpoint)

	fmt.Printf("Stackyard ECS basic client using %s\n", endpoint)

	createClusterOut, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{ClusterName: aws.String(clusterName)})
	if err != nil {
		exitf("create cluster: %v", err)
	}
	logf("created cluster: %s", aws.ToString(createClusterOut.Cluster.ClusterArn))

	registerTaskDefOut, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family: aws.String(taskFamily),
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:  aws.String("app"),
				Image: aws.String("public.ecr.aws/docker/library/busybox:latest"),
			},
		},
	})
	if err != nil {
		exitf("register task definition: %v", err)
	}
	taskDefinitionArn := aws.ToString(registerTaskDefOut.TaskDefinition.TaskDefinitionArn)
	logf("registered task definition: %s", taskDefinitionArn)

	runTaskOut, err := client.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:        aws.String(clusterName),
		TaskDefinition: aws.String(taskDefinitionArn),
		Count:          aws.Int32(1),
	})
	if err != nil {
		exitf("run task: %v", err)
	}
	if len(runTaskOut.Tasks) == 0 {
		exitf("run task: no tasks returned")
	}
	taskArn := aws.ToString(runTaskOut.Tasks[0].TaskArn)
	logf("started task: %s", taskArn)

	listTasksOut, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		exitf("list tasks: %v", err)
	}
	logf("task count: %d", len(listTasksOut.TaskArns))

	describeTasksOut, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   []string{taskArn},
	})
	if err != nil {
		exitf("describe tasks: %v", err)
	}
	logf("described tasks: %d", len(describeTasksOut.Tasks))

	if _, err := client.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: aws.String(clusterName),
		Task:    aws.String(taskArn),
		Reason:  aws.String("ecs-basic cleanup"),
	}); err != nil {
		exitf("stop task: %v", err)
	}
	logf("stopped task: %s", taskArn)

	if _, err := client.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefinitionArn),
	}); err != nil {
		exitf("deregister task definition: %v", err)
	}
	logf("deregistered task definition: %s", taskDefinitionArn)

	if _, err := client.DeleteCluster(ctx, &ecs.DeleteClusterInput{Cluster: aws.String(clusterName)}); err != nil {
		exitf("delete cluster: %v", err)
	}
	logf("deleted cluster: %s", clusterName)

	fmt.Println("Done.")
}

func newECSClient(ctx context.Context, endpoint string) *ecs.Client {
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

	return ecs.NewFromConfig(cfg, func(o *ecs.Options) {
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
