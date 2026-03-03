package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	clusterName := getenv("STACKYARD_CLUSTER_NAME", "ecs-advanced-cluster")
	taskFamily := getenv("STACKYARD_TASK_FAMILY", "ecs-advanced-task")
	serviceName := getenv("STACKYARD_SERVICE_NAME", "ecs-advanced-service")
	namespace := getenv("STACKYARD_NAMESPACE", "ecs-advanced-ns")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	client := ecs.NewFromConfig(cfg, func(o *ecs.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	fmt.Printf("Stackyard ECS advanced client using %s\n", endpoint)

	if _, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{ClusterName: aws.String(clusterName)}); err != nil {
		exitf("create cluster: %v", err)
	}

	registerContainerOut, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "RegisterContainerInstance", map[string]any{
		"cluster":       clusterName,
		"ec2InstanceId": "i-ecsadvanced001",
	})
	if err != nil {
		exitf("register container instance: %v", err)
	}
	containerInstanceARN := nestedString(registerContainerOut, "containerInstance", "containerInstanceArn")
	logf("registered container instance: %s", containerInstanceARN)

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
	taskDefinitionARN := aws.ToString(registerTaskDefOut.TaskDefinition.TaskDefinitionArn)
	logf("registered task definition: %s", taskDefinitionARN)

	createServiceOut, err := client.CreateService(ctx, &ecs.CreateServiceInput{
		Cluster:        aws.String(clusterName),
		ServiceName:    aws.String(serviceName),
		TaskDefinition: aws.String(taskDefinitionARN),
		LaunchType:     types.LaunchTypeFargate,
		DesiredCount:   aws.Int32(1),
		Tags: []types.Tag{
			{Key: aws.String("namespace"), Value: aws.String(namespace)},
			{Key: aws.String("env"), Value: aws.String("dev")},
		},
	})
	if err != nil {
		exitf("create service: %v", err)
	}
	serviceARN := aws.ToString(createServiceOut.Service.ServiceArn)
	logf("created service: %s", serviceARN)

	createTaskSetOut, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "CreateTaskSet", map[string]any{
		"cluster":        clusterName,
		"service":        serviceName,
		"taskDefinition": taskDefinitionARN,
		"launchType":     "FARGATE",
		"scale": map[string]any{
			"value": 100,
		},
	})
	if err != nil {
		exitf("create task set: %v", err)
	}
	taskSetARN := nestedString(createTaskSetOut, "taskSet", "taskSetArn")
	logf("created task set: %s", taskSetARN)

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "UpdateTaskSet", map[string]any{
		"cluster": clusterName,
		"service": serviceName,
		"taskSet": taskSetARN,
		"scale": map[string]any{
			"value": 80,
		},
	}); err != nil {
		exitf("update task set: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "UpdateServicePrimaryTaskSet", map[string]any{
		"cluster":        clusterName,
		"service":        serviceName,
		"primaryTaskSet": taskSetARN,
	}); err != nil {
		exitf("update service primary task set: %v", err)
	}

	runTaskOut, err := client.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:        aws.String(clusterName),
		TaskDefinition: aws.String(taskDefinitionARN),
		Count:          aws.Int32(1),
	})
	if err != nil {
		exitf("run task: %v", err)
	}
	if len(runTaskOut.Tasks) == 0 {
		exitf("run task: no tasks returned")
	}
	taskARN := aws.ToString(runTaskOut.Tasks[0].TaskArn)
	logf("ran task: %s", taskARN)

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "ExecuteCommand", map[string]any{
		"cluster":     clusterName,
		"task":        taskARN,
		"container":   "app",
		"command":     "echo stage14",
		"interactive": true,
	}); err != nil {
		exitf("execute command: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "GetTaskProtection", map[string]any{
		"cluster": clusterName,
		"tasks":   []string{taskARN},
	}); err != nil {
		exitf("get task protection: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "UpdateTaskProtection", map[string]any{
		"cluster":           clusterName,
		"tasks":             []string{taskARN},
		"protectionEnabled": true,
		"expiresInMinutes":  15,
	}); err != nil {
		exitf("update task protection: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "DiscoverPollEndpoint", map[string]any{
		"cluster":           clusterName,
		"containerInstance": containerInstanceARN,
	}); err != nil {
		exitf("discover poll endpoint: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "PutAttributes", map[string]any{
		"cluster": clusterName,
		"attributes": []map[string]any{
			{"name": "rack", "value": "r1", "targetType": "container-instance", "targetId": containerInstanceARN},
		},
	}); err != nil {
		exitf("put attributes: %v", err)
	}
	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "ListAttributes", map[string]any{
		"cluster":    clusterName,
		"targetType": "container-instance",
		"targetId":   containerInstanceARN,
	}); err != nil {
		exitf("list attributes: %v", err)
	}
	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "DeleteAttributes", map[string]any{
		"cluster": clusterName,
		"attributes": []map[string]any{
			{"name": "rack", "targetType": "container-instance", "targetId": containerInstanceARN},
		},
	}); err != nil {
		exitf("delete attributes: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "SubmitAttachmentStateChanges", map[string]any{
		"cluster":           clusterName,
		"containerInstance": containerInstanceARN,
		"attachments": []map[string]any{
			{"attachmentArn": "arn:aws:ecs:us-east-1:123456789012:attachment/demo", "status": "ATTACHED"},
		},
	}); err != nil {
		exitf("submit attachment state changes: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "SubmitContainerStateChange", map[string]any{
		"cluster":           clusterName,
		"containerInstance": containerInstanceARN,
		"task":              taskARN,
		"containerName":     "app",
		"status":            "RUNNING",
	}); err != nil {
		exitf("submit container state change: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "SubmitTaskStateChange", map[string]any{
		"cluster": clusterName,
		"task":    taskARN,
		"status":  "RUNNING",
	}); err != nil {
		exitf("submit task state change: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "SubmitTaskStateChangeByAgent", map[string]any{
		"cluster": clusterName,
		"task":    taskARN,
		"status":  "RUNNING",
	}); err != nil {
		exitf("submit task state change by agent: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "SubmitTaskStateChangeByManagedAgents", map[string]any{
		"cluster": clusterName,
		"task":    taskARN,
		"status":  "RUNNING",
	}); err != nil {
		exitf("submit task state change by managed agents: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "StartTelemetrySession", map[string]any{
		"cluster":           clusterName,
		"containerInstance": containerInstanceARN,
	}); err != nil {
		exitf("start telemetry session: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"resourceArn": serviceARN,
		"tags": []map[string]any{
			{"key": "team", "value": "platform"},
		},
	}); err != nil {
		exitf("tag resource: %v", err)
	}
	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "ListTagsForResource", map[string]any{
		"resourceArn": serviceARN,
	}); err != nil {
		exitf("list tags for resource: %v", err)
	}
	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "UntagResource", map[string]any{
		"resourceArn": serviceARN,
		"tagKeys":     []string{"team"},
	}); err != nil {
		exitf("untag resource: %v", err)
	}

	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "ListServicesByLaunchType", map[string]any{
		"cluster":    clusterName,
		"launchType": "FARGATE",
	}); err != nil {
		exitf("list services by launch type: %v", err)
	}
	if _, err := ecsRawJSONRequest(ctx, endpoint, region, creds, "ListServicesByNamespace", map[string]any{
		"cluster":   clusterName,
		"namespace": namespace,
	}); err != nil {
		exitf("list services by namespace: %v", err)
	}

	if _, err := client.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: aws.String(clusterName),
		Task:    aws.String(taskARN),
		Reason:  aws.String("ecs-advanced cleanup"),
	}); err != nil {
		exitf("stop task: %v", err)
	}

	if _, err := client.DeleteTaskSet(ctx, &ecs.DeleteTaskSetInput{
		Cluster: aws.String(clusterName),
		Service: aws.String(serviceName),
		TaskSet: aws.String(taskSetARN),
	}); err != nil {
		exitf("delete task set: %v", err)
	}

	if _, err := client.DeleteService(ctx, &ecs.DeleteServiceInput{
		Cluster: aws.String(clusterName),
		Service: aws.String(serviceName),
		Force:   aws.Bool(true),
	}); err != nil {
		exitf("delete service: %v", err)
	}

	if _, err := client.DeregisterContainerInstance(ctx, &ecs.DeregisterContainerInstanceInput{
		Cluster:           aws.String(clusterName),
		ContainerInstance: aws.String(containerInstanceARN),
	}); err != nil {
		exitf("deregister container instance: %v", err)
	}

	if _, err := client.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{TaskDefinition: aws.String(taskDefinitionARN)}); err != nil {
		exitf("deregister task definition: %v", err)
	}

	if _, err := client.DeleteCluster(ctx, &ecs.DeleteClusterInput{Cluster: aws.String(clusterName)}); err != nil {
		exitf("delete cluster: %v", err)
	}

	fmt.Println("Done.")
}

func ecsRawJSONRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, action string, payload map[string]any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerService_V20141113."+action)

	credentialsValue, err := creds.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialsValue, req, hashSHA256(body), "ecs", region, time.Now()); err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%s failed (%d): %s", action, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if len(respBody) == 0 {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func nestedString(m map[string]any, keys ...string) string {
	cur := any(m)
	for _, key := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = obj[key]
	}
	s, _ := cur.(string)
	return s
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
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
