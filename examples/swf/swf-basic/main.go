package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/swf"
	"github.com/aws/aws-sdk-go-v2/service/swf/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	domain := getenv("STACKYARD_DOMAIN", "demo-swf-basic")
	workflowName := getenv("STACKYARD_WORKFLOW", "demo-workflow-basic")
	workflowVersion := getenv("STACKYARD_WORKFLOW_VERSION", "1")
	taskList := getenv("STACKYARD_TASK_LIST", "basic-tasks")

	ctx := context.Background()
	client := newSWFClient(ctx, endpoint)

	fmt.Printf("Stackyard SWF basic client using %s\n", endpoint)

	if err := registerDomain(ctx, client, domain); err != nil {
		exitf("register domain: %v", err)
	}
	logf("registered domain: %s", domain)

	if err := registerWorkflowType(ctx, client, domain, workflowName, workflowVersion, taskList); err != nil {
		exitf("register workflow type: %v", err)
	}
	logf("registered workflow: %s:%s", workflowName, workflowVersion)

	workflowID := fmt.Sprintf("basic-%d", time.Now().Unix())
	runID, err := startWorkflow(ctx, client, domain, workflowID, workflowName, workflowVersion, taskList)
	if err != nil {
		exitf("start workflow: %v", err)
	}
	logf("started workflow: %s (run %s)", workflowID, runID)

	if err := describeWorkflow(ctx, client, domain, workflowID, runID); err != nil {
		exitf("describe workflow: %v", err)
	}

	if err := listOpenWorkflows(ctx, client, domain); err != nil {
		exitf("list open workflows: %v", err)
	}

	if err := countOpenWorkflows(ctx, client, domain); err != nil {
		exitf("count open workflows: %v", err)
	}

	fmt.Println("Done.")
}

func newSWFClient(ctx context.Context, endpoint string) *swf.Client {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == swf.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		)),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	return swf.NewFromConfig(cfg)
}

func registerDomain(ctx context.Context, client *swf.Client, domain string) error {
	_, err := client.RegisterDomain(ctx, &swf.RegisterDomainInput{
		Name:                                   aws.String(domain),
		WorkflowExecutionRetentionPeriodInDays: aws.String("1"),
		Description:                            aws.String("stackyard swf basic domain"),
	})
	return err
}

func registerWorkflowType(ctx context.Context, client *swf.Client, domain, name, version, taskList string) error {
	_, err := client.RegisterWorkflowType(ctx, &swf.RegisterWorkflowTypeInput{
		Domain:  aws.String(domain),
		Name:    aws.String(name),
		Version: aws.String(version),
		DefaultTaskList: &types.TaskList{
			Name: aws.String(taskList),
		},
		Description: aws.String("basic workflow type"),
	})
	return err
}

func startWorkflow(ctx context.Context, client *swf.Client, domain, workflowID, name, version, taskList string) (string, error) {
	resp, err := client.StartWorkflowExecution(ctx, &swf.StartWorkflowExecutionInput{
		Domain:     aws.String(domain),
		WorkflowId: aws.String(workflowID),
		WorkflowType: &types.WorkflowType{
			Name:    aws.String(name),
			Version: aws.String(version),
		},
		TaskList: &types.TaskList{
			Name: aws.String(taskList),
		},
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(resp.RunId), nil
}

func describeWorkflow(ctx context.Context, client *swf.Client, domain, workflowID, runID string) error {
	_, err := client.DescribeWorkflowExecution(ctx, &swf.DescribeWorkflowExecutionInput{
		Domain: aws.String(domain),
		Execution: &types.WorkflowExecution{
			WorkflowId: aws.String(workflowID),
			RunId:      aws.String(runID),
		},
	})
	return err
}

func listOpenWorkflows(ctx context.Context, client *swf.Client, domain string) error {
	_, err := client.ListOpenWorkflowExecutions(ctx, &swf.ListOpenWorkflowExecutionsInput{
		Domain: aws.String(domain),
		StartTimeFilter: &types.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-1 * time.Hour)),
		},
	})
	return err
}

func countOpenWorkflows(ctx context.Context, client *swf.Client, domain string) error {
	_, err := client.CountOpenWorkflowExecutions(ctx, &swf.CountOpenWorkflowExecutionsInput{
		Domain: aws.String(domain),
		StartTimeFilter: &types.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-1 * time.Hour)),
		},
	})
	return err
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
