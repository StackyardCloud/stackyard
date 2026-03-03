package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	computeEnv := getenv("STACKYARD_COMPUTE_ENV", "batch-advanced-ce")
	jobQueue := getenv("STACKYARD_JOB_QUEUE", "batch-advanced-queue")
	schedulingPolicy := getenv("STACKYARD_SCHEDULING_POLICY", "batch-advanced-policy")
	jobDef := getenv("STACKYARD_JOB_DEFINITION", "batch-advanced-jobdef")

	ctx := context.Background()
	client := newBatchClient(ctx, endpoint)

	fmt.Printf("Stackyard Batch advanced client using %s\n", endpoint)

	sp, err := client.CreateSchedulingPolicy(ctx, &batch.CreateSchedulingPolicyInput{Name: aws.String(schedulingPolicy)})
	if err != nil {
		exitf("create scheduling policy: %v", err)
	}

	createCE, err := client.CreateComputeEnvironment(ctx, &batch.CreateComputeEnvironmentInput{
		ComputeEnvironmentName: aws.String(computeEnv),
		Type:                   types.CETypeUnmanaged,
		State:                  types.CEStateEnabled,
		UnmanagedvCpus:         aws.Int32(32),
	})
	if err != nil {
		exitf("create compute environment: %v", err)
	}

	createQueue, err := client.CreateJobQueue(ctx, &batch.CreateJobQueueInput{
		JobQueueName: aws.String(jobQueue),
		Priority:     aws.Int32(5),
		State:        types.JQStateEnabled,
		ComputeEnvironmentOrder: []types.ComputeEnvironmentOrder{{
			Order:              aws.Int32(1),
			ComputeEnvironment: aws.String(computeEnv),
		}},
		SchedulingPolicyArn: sp.Arn,
	})
	if err != nil {
		exitf("create job queue: %v", err)
	}

	if _, err := client.UpdateComputeEnvironment(ctx, &batch.UpdateComputeEnvironmentInput{
		ComputeEnvironment: aws.String(computeEnv),
		State:              types.CEStateEnabled,
		UnmanagedvCpus:     aws.Int32(64),
	}); err != nil {
		exitf("update compute environment: %v", err)
	}

	if _, err := client.UpdateJobQueue(ctx, &batch.UpdateJobQueueInput{
		JobQueue: aws.String(jobQueue),
		Priority: aws.Int32(10),
		State:    types.JQStateEnabled,
	}); err != nil {
		exitf("update job queue: %v", err)
	}

	register, err := client.RegisterJobDefinition(ctx, &batch.RegisterJobDefinitionInput{
		JobDefinitionName: aws.String(jobDef),
		Type:              types.JobDefinitionTypeContainer,
		Tags:              map[string]string{"env": "advanced"},
	})
	if err != nil {
		exitf("register job definition: %v", err)
	}

	submitA, err := client.SubmitJob(ctx, &batch.SubmitJobInput{
		JobName:       aws.String("batch-advanced-job-a"),
		JobQueue:      aws.String(jobQueue),
		JobDefinition: aws.String(jobDef),
		Tags:          map[string]string{"flow": "advanced"},
	})
	if err != nil {
		exitf("submit job A: %v", err)
	}
	submitB, err := client.SubmitJob(ctx, &batch.SubmitJobInput{
		JobName:       aws.String("batch-advanced-job-b"),
		JobQueue:      aws.String(jobQueue),
		JobDefinition: aws.String(jobDef),
	})
	if err != nil {
		exitf("submit job B: %v", err)
	}

	if _, err := client.DescribeJobDefinitions(ctx, &batch.DescribeJobDefinitionsInput{JobDefinitions: []string{jobDef}}); err != nil {
		exitf("describe job definitions: %v", err)
	}
	if _, err := client.DescribeComputeEnvironments(ctx, &batch.DescribeComputeEnvironmentsInput{ComputeEnvironments: []string{computeEnv}}); err != nil {
		exitf("describe compute environments: %v", err)
	}
	if _, err := client.DescribeJobQueues(ctx, &batch.DescribeJobQueuesInput{JobQueues: []string{jobQueue}}); err != nil {
		exitf("describe job queues: %v", err)
	}
	if _, err := client.DescribeSchedulingPolicies(ctx, &batch.DescribeSchedulingPoliciesInput{Arns: []string{aws.ToString(sp.Arn)}}); err != nil {
		exitf("describe scheduling policies: %v", err)
	}
	if _, err := client.ListSchedulingPolicies(ctx, &batch.ListSchedulingPoliciesInput{}); err != nil {
		exitf("list scheduling policies: %v", err)
	}

	if _, err := client.TagResource(ctx, &batch.TagResourceInput{ResourceArn: register.JobDefinitionArn, Tags: map[string]string{"owner": "platform"}}); err != nil {
		exitf("tag resource: %v", err)
	}
	tagsOut, err := client.ListTagsForResource(ctx, &batch.ListTagsForResourceInput{ResourceArn: register.JobDefinitionArn})
	if err != nil {
		exitf("list tags for resource: %v", err)
	}
	logf("job definition tags: %d", len(tagsOut.Tags))
	if _, err := client.UntagResource(ctx, &batch.UntagResourceInput{ResourceArn: register.JobDefinitionArn, TagKeys: []string{"owner"}}); err != nil {
		exitf("untag resource: %v", err)
	}

	if _, err := client.ListJobs(ctx, &batch.ListJobsInput{JobQueue: aws.String(jobQueue)}); err != nil {
		exitf("list jobs: %v", err)
	}
	if _, err := client.DescribeJobs(ctx, &batch.DescribeJobsInput{Jobs: []string{aws.ToString(submitA.JobId), aws.ToString(submitB.JobId)}}); err != nil {
		exitf("describe jobs: %v", err)
	}

	if _, err := client.CancelJob(ctx, &batch.CancelJobInput{JobId: submitA.JobId, Reason: aws.String("advanced cancel")}); err != nil {
		exitf("cancel job A: %v", err)
	}
	if _, err := client.TerminateJob(ctx, &batch.TerminateJobInput{JobId: submitB.JobId, Reason: aws.String("advanced terminate")}); err != nil {
		exitf("terminate job B: %v", err)
	}

	if _, err := client.UpdateSchedulingPolicy(ctx, &batch.UpdateSchedulingPolicyInput{Arn: sp.Arn}); err != nil {
		exitf("update scheduling policy: %v", err)
	}

	if _, err := client.DeregisterJobDefinition(ctx, &batch.DeregisterJobDefinitionInput{JobDefinition: register.JobDefinitionArn}); err != nil {
		exitf("deregister job definition: %v", err)
	}
	if _, err := client.DeleteJobQueue(ctx, &batch.DeleteJobQueueInput{JobQueue: createQueue.JobQueueArn}); err != nil {
		exitf("delete job queue: %v", err)
	}
	if _, err := client.DeleteComputeEnvironment(ctx, &batch.DeleteComputeEnvironmentInput{ComputeEnvironment: createCE.ComputeEnvironmentArn}); err != nil {
		exitf("delete compute environment: %v", err)
	}
	if _, err := client.DeleteSchedulingPolicy(ctx, &batch.DeleteSchedulingPolicyInput{Arn: sp.Arn}); err != nil {
		exitf("delete scheduling policy: %v", err)
	}

	fmt.Println("Done.")
}

func newBatchClient(ctx context.Context, endpoint string) *batch.Client {
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

	return batch.NewFromConfig(cfg, func(o *batch.Options) {
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
