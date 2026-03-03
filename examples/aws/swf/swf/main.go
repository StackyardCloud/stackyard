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
	domain := getenv("STACKYARD_DOMAIN", "demo-swf")
	workflowName := getenv("STACKYARD_WORKFLOW", "demo-workflow")
	workflowVersion := getenv("STACKYARD_WORKFLOW_VERSION", "1")
	activityName := getenv("STACKYARD_ACTIVITY", "demo-activity")
	activityVersion := getenv("STACKYARD_ACTIVITY_VERSION", "1")
	taskList := getenv("STACKYARD_TASK_LIST", "advanced-tasks")
	region := getenv("AWS_REGION", "us-east-1")
	accountID := getenv("AWS_ACCOUNT_ID", "123456789012")

	ctx := context.Background()
	client := newSWFClient(ctx, endpoint)

	fmt.Printf("Stackyard SWF advanced client using %s\n", endpoint)

	if err := registerDomain(ctx, client, domain); err != nil {
		exitf("register domain: %v", err)
	}
	logf("registered domain: %s", domain)

	if err := listDomains(ctx, client); err != nil {
		exitf("list domains: %v", err)
	}

	if err := describeDomain(ctx, client, domain); err != nil {
		exitf("describe domain: %v", err)
	}

	if err := registerActivityType(ctx, client, domain, activityName, activityVersion); err != nil {
		exitf("register activity type: %v", err)
	}
	logf("registered activity: %s:%s", activityName, activityVersion)

	if err := listActivityTypes(ctx, client, domain); err != nil {
		exitf("list activity types: %v", err)
	}

	if err := describeActivityType(ctx, client, domain, activityName, activityVersion); err != nil {
		exitf("describe activity type: %v", err)
	}

	if err := registerWorkflowType(ctx, client, domain, workflowName, workflowVersion, taskList); err != nil {
		exitf("register workflow type: %v", err)
	}
	logf("registered workflow: %s:%s", workflowName, workflowVersion)

	if err := listWorkflowTypes(ctx, client, domain); err != nil {
		exitf("list workflow types: %v", err)
	}

	if err := describeWorkflowType(ctx, client, domain, workflowName, workflowVersion); err != nil {
		exitf("describe workflow type: %v", err)
	}

	workflowID := fmt.Sprintf("advanced-%d", time.Now().Unix())
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

	if err := signalWorkflow(ctx, client, domain, workflowID, runID); err != nil {
		exitf("signal workflow: %v", err)
	}

	if err := countPendingTasks(ctx, client, domain, taskList); err != nil {
		exitf("count pending tasks: %v", err)
	}

	if err := pollTasks(ctx, client, domain, taskList); err != nil {
		exitf("poll tasks: %v", err)
	}

	if err := recordHeartbeat(ctx, client); err != nil {
		exitf("record heartbeat: %v", err)
	}

	if err := getHistory(ctx, client, domain, workflowID, runID); err != nil {
		exitf("get workflow history: %v", err)
	}

	if err := terminateWorkflow(ctx, client, domain, workflowID, runID); err != nil {
		exitf("terminate workflow: %v", err)
	}
	logf("terminated workflow")

	if err := listClosedWorkflows(ctx, client, domain); err != nil {
		exitf("list closed workflows: %v", err)
	}

	if err := countClosedWorkflows(ctx, client, domain); err != nil {
		exitf("count closed workflows: %v", err)
	}

	domainArn := fmt.Sprintf("arn:aws:swf:%s:%s:domain/%s", region, accountID, domain)
	if err := tagResource(ctx, client, domainArn); err != nil {
		exitf("tag resource: %v", err)
	}

	if err := listTags(ctx, client, domainArn); err != nil {
		exitf("list tags: %v", err)
	}

	if err := untagResource(ctx, client, domainArn); err != nil {
		exitf("untag resource: %v", err)
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
		Description:                            aws.String("stackyard swf advanced domain"),
	})
	return err
}

func listDomains(ctx context.Context, client *swf.Client) error {
	resp, err := client.ListDomains(ctx, &swf.ListDomainsInput{
		RegistrationStatus: types.RegistrationStatusRegistered,
	})
	if err != nil {
		return err
	}
	logf("domains: %d", len(resp.DomainInfos))
	return nil
}

func describeDomain(ctx context.Context, client *swf.Client, domain string) error {
	resp, err := client.DescribeDomain(ctx, &swf.DescribeDomainInput{Name: aws.String(domain)})
	if err != nil {
		return err
	}
	logf("domain status: %v", resp.DomainInfo.Status)
	return nil
}

func registerActivityType(ctx context.Context, client *swf.Client, domain, name, version string) error {
	_, err := client.RegisterActivityType(ctx, &swf.RegisterActivityTypeInput{
		Domain:      aws.String(domain),
		Name:        aws.String(name),
		Version:     aws.String(version),
		Description: aws.String("advanced activity type"),
	})
	return err
}

func listActivityTypes(ctx context.Context, client *swf.Client, domain string) error {
	resp, err := client.ListActivityTypes(ctx, &swf.ListActivityTypesInput{
		Domain:             aws.String(domain),
		RegistrationStatus: types.RegistrationStatusRegistered,
	})
	if err != nil {
		return err
	}
	logf("activity types: %d", len(resp.TypeInfos))
	return nil
}

func describeActivityType(ctx context.Context, client *swf.Client, domain, name, version string) error {
	_, err := client.DescribeActivityType(ctx, &swf.DescribeActivityTypeInput{
		Domain: aws.String(domain),
		ActivityType: &types.ActivityType{
			Name:    aws.String(name),
			Version: aws.String(version),
		},
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
		Description: aws.String("advanced workflow type"),
	})
	return err
}

func listWorkflowTypes(ctx context.Context, client *swf.Client, domain string) error {
	resp, err := client.ListWorkflowTypes(ctx, &swf.ListWorkflowTypesInput{
		Domain:             aws.String(domain),
		RegistrationStatus: types.RegistrationStatusRegistered,
	})
	if err != nil {
		return err
	}
	logf("workflow types: %d", len(resp.TypeInfos))
	return nil
}

func describeWorkflowType(ctx context.Context, client *swf.Client, domain, name, version string) error {
	_, err := client.DescribeWorkflowType(ctx, &swf.DescribeWorkflowTypeInput{
		Domain: aws.String(domain),
		WorkflowType: &types.WorkflowType{
			Name:    aws.String(name),
			Version: aws.String(version),
		},
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
		TagList: []string{"team:stackyard", "env:demo"},
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
	resp, err := client.ListOpenWorkflowExecutions(ctx, &swf.ListOpenWorkflowExecutionsInput{
		Domain: aws.String(domain),
		StartTimeFilter: &types.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-1 * time.Hour)),
		},
	})
	if err != nil {
		return err
	}
	logf("open workflows: %d", len(resp.ExecutionInfos))
	return nil
}

func listClosedWorkflows(ctx context.Context, client *swf.Client, domain string) error {
	resp, err := client.ListClosedWorkflowExecutions(ctx, &swf.ListClosedWorkflowExecutionsInput{
		Domain: aws.String(domain),
		CloseTimeFilter: &types.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-1 * time.Hour)),
		},
	})
	if err != nil {
		return err
	}
	logf("closed workflows: %d", len(resp.ExecutionInfos))
	return nil
}

func countOpenWorkflows(ctx context.Context, client *swf.Client, domain string) error {
	resp, err := client.CountOpenWorkflowExecutions(ctx, &swf.CountOpenWorkflowExecutionsInput{
		Domain: aws.String(domain),
		StartTimeFilter: &types.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-1 * time.Hour)),
		},
	})
	if err != nil {
		return err
	}
	logf("open workflow count: %d", resp.Count)
	return nil
}

func countClosedWorkflows(ctx context.Context, client *swf.Client, domain string) error {
	resp, err := client.CountClosedWorkflowExecutions(ctx, &swf.CountClosedWorkflowExecutionsInput{
		Domain: aws.String(domain),
		CloseTimeFilter: &types.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-1 * time.Hour)),
		},
	})
	if err != nil {
		return err
	}
	logf("closed workflow count: %d", resp.Count)
	return nil
}

func signalWorkflow(ctx context.Context, client *swf.Client, domain, workflowID, runID string) error {
	_, err := client.SignalWorkflowExecution(ctx, &swf.SignalWorkflowExecutionInput{
		Domain:     aws.String(domain),
		WorkflowId: aws.String(workflowID),
		RunId:      aws.String(runID),
		SignalName: aws.String("refresh"),
	})
	return err
}

func terminateWorkflow(ctx context.Context, client *swf.Client, domain, workflowID, runID string) error {
	_, err := client.TerminateWorkflowExecution(ctx, &swf.TerminateWorkflowExecutionInput{
		Domain:     aws.String(domain),
		WorkflowId: aws.String(workflowID),
		RunId:      aws.String(runID),
		Reason:     aws.String("demo cleanup"),
	})
	return err
}

func getHistory(ctx context.Context, client *swf.Client, domain, workflowID, runID string) error {
	_, err := client.GetWorkflowExecutionHistory(ctx, &swf.GetWorkflowExecutionHistoryInput{
		Domain: aws.String(domain),
		Execution: &types.WorkflowExecution{
			WorkflowId: aws.String(workflowID),
			RunId:      aws.String(runID),
		},
	})
	return err
}

func countPendingTasks(ctx context.Context, client *swf.Client, domain, taskList string) error {
	if _, err := client.CountPendingActivityTasks(ctx, &swf.CountPendingActivityTasksInput{
		Domain: aws.String(domain),
		TaskList: &types.TaskList{
			Name: aws.String(taskList),
		},
	}); err != nil {
		return err
	}

	if _, err := client.CountPendingDecisionTasks(ctx, &swf.CountPendingDecisionTasksInput{
		Domain: aws.String(domain),
		TaskList: &types.TaskList{
			Name: aws.String(taskList),
		},
	}); err != nil {
		return err
	}
	logf("pending task counts requested")
	return nil
}

func pollTasks(ctx context.Context, client *swf.Client, domain, taskList string) error {
	if _, err := client.PollForActivityTask(ctx, &swf.PollForActivityTaskInput{
		Domain: aws.String(domain),
		TaskList: &types.TaskList{
			Name: aws.String(taskList),
		},
		Identity: aws.String("worker-1"),
	}); err != nil {
		return err
	}

	if _, err := client.PollForDecisionTask(ctx, &swf.PollForDecisionTaskInput{
		Domain: aws.String(domain),
		TaskList: &types.TaskList{
			Name: aws.String(taskList),
		},
		Identity: aws.String("decider-1"),
	}); err != nil {
		return err
	}

	logf("polled for activity and decision tasks")
	return nil
}

func recordHeartbeat(ctx context.Context, client *swf.Client) error {
	_, err := client.RecordActivityTaskHeartbeat(ctx, &swf.RecordActivityTaskHeartbeatInput{
		TaskToken: aws.String("token"),
		Details:   aws.String("heartbeat"),
	})
	return err
}

func tagResource(ctx context.Context, client *swf.Client, arn string) error {
	_, err := client.TagResource(ctx, &swf.TagResourceInput{
		ResourceArn: aws.String(arn),
		Tags: []types.ResourceTag{
			{Key: aws.String("env"), Value: aws.String("demo")},
			{Key: aws.String("team"), Value: aws.String("stackyard")},
		},
	})
	return err
}

func listTags(ctx context.Context, client *swf.Client, arn string) error {
	resp, err := client.ListTagsForResource(ctx, &swf.ListTagsForResourceInput{ResourceArn: aws.String(arn)})
	if err != nil {
		return err
	}
	logf("tags: %d", len(resp.Tags))
	return nil
}

func untagResource(ctx context.Context, client *swf.Client, arn string) error {
	_, err := client.UntagResource(ctx, &swf.UntagResourceInput{
		ResourceArn: aws.String(arn),
		TagKeys:     []string{"env"},
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
