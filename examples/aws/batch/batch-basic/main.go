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
	computeEnv := getenv("STACKYARD_COMPUTE_ENV", "batch-basic-ce")
	jobQueue := getenv("STACKYARD_JOB_QUEUE", "batch-basic-queue")
	jobDef := getenv("STACKYARD_JOB_DEFINITION", "batch-basic-jobdef")
	jobName := getenv("STACKYARD_JOB_NAME", "batch-basic-job")

	ctx := context.Background()
	client := newBatchClient(ctx, endpoint)

	fmt.Printf("Stackyard Batch basic client using %s\n", endpoint)

	createCE, err := client.CreateComputeEnvironment(ctx, &batch.CreateComputeEnvironmentInput{
		ComputeEnvironmentName: aws.String(computeEnv),
		Type:                   types.CETypeUnmanaged,
		State:                  types.CEStateEnabled,
		UnmanagedvCpus:         aws.Int32(16),
	})
	if err != nil {
		exitf("create compute environment: %v", err)
	}
	logf("created compute environment: %s", aws.ToString(createCE.ComputeEnvironmentName))

	createQueue, err := client.CreateJobQueue(ctx, &batch.CreateJobQueueInput{
		JobQueueName: aws.String(jobQueue),
		Priority:     aws.Int32(1),
		State:        types.JQStateEnabled,
		ComputeEnvironmentOrder: []types.ComputeEnvironmentOrder{{
			Order:              aws.Int32(1),
			ComputeEnvironment: aws.String(computeEnv),
		}},
	})
	if err != nil {
		exitf("create job queue: %v", err)
	}
	logf("created job queue: %s", aws.ToString(createQueue.JobQueueName))

	register, err := client.RegisterJobDefinition(ctx, &batch.RegisterJobDefinitionInput{
		JobDefinitionName: aws.String(jobDef),
		Type:              types.JobDefinitionTypeContainer,
	})
	if err != nil {
		exitf("register job definition: %v", err)
	}
	logf("registered job definition: %s", aws.ToString(register.JobDefinitionName))

	submit, err := client.SubmitJob(ctx, &batch.SubmitJobInput{
		JobName:       aws.String(jobName),
		JobQueue:      aws.String(jobQueue),
		JobDefinition: aws.String(jobDef),
	})
	if err != nil {
		exitf("submit job: %v", err)
	}
	logf("submitted job: %s", aws.ToString(submit.JobId))

	listOut, err := client.ListJobs(ctx, &batch.ListJobsInput{JobQueue: aws.String(jobQueue)})
	if err != nil {
		exitf("list jobs: %v", err)
	}
	logf("jobs in queue: %d", len(listOut.JobSummaryList))

	descOut, err := client.DescribeJobs(ctx, &batch.DescribeJobsInput{Jobs: []string{aws.ToString(submit.JobId)}})
	if err != nil {
		exitf("describe jobs: %v", err)
	}
	logf("described jobs: %d", len(descOut.Jobs))

	if _, err := client.TerminateJob(ctx, &batch.TerminateJobInput{JobId: submit.JobId, Reason: aws.String("basic cleanup")}); err != nil {
		exitf("terminate job: %v", err)
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
