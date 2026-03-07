package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	serviceID := getenv("STACKYARD_GCP_RUN_SERVICE_ID", "service-1")
	jobID := getenv("STACKYARD_GCP_RUN_JOB_ID", "job-1")
	executionID := getenv("STACKYARD_GCP_RUN_EXECUTION_ID", "execution-1")
	taskID := getenv("STACKYARD_GCP_RUN_TASK_ID", "task-1")
	revisionID := getenv("STACKYARD_GCP_RUN_REVISION_ID", serviceID+"-00001")

	projectName := "projects/" + projectID
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	serviceName := fmt.Sprintf("%s/services/%s", parent, serviceID)
	jobName := fmt.Sprintf("%s/jobs/%s", parent, jobID)
	executionName := fmt.Sprintf("%s/executions/%s", jobName, executionID)
	taskName := fmt.Sprintf("%s/tasks/%s", executionName, taskID)
	revisionName := fmt.Sprintf("%s/revisions/%s", serviceName, revisionID)

	fmt.Printf("Stackyard GCP Cloud Run Admin run/apiv2 clients using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "run",
		},
	}

	locationsClient, err := run.NewLocationsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create run locations client: %v", err)
	}
	defer closeClient("locations", locationsClient.Close)

	servicesClient, err := run.NewServicesRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create run services client: %v", err)
	}
	defer closeClient("services", servicesClient.Close)

	jobsClient, err := run.NewJobsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create run jobs client: %v", err)
	}
	defer closeClient("jobs", jobsClient.Close)

	executionsClient, err := run.NewExecutionsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create run executions client: %v", err)
	}
	defer closeClient("executions", executionsClient.Close)

	tasksClient, err := run.NewTasksRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create run tasks client: %v", err)
	}
	defer closeClient("tasks", tasksClient.Close)

	revisionsClient, err := run.NewRevisionsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create run revisions client: %v", err)
	}
	defer closeClient("revisions", revisionsClient.Close)

	serviceOperationName := ""
	jobOperationName := ""

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := locationsClient.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: projectName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context) error {
				_, err := locationsClient.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListServices",
			call: func(ctx context.Context) error {
				it := servicesClient.ListServices(ctx, &runpb.ListServicesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetService",
			call: func(ctx context.Context) error {
				_, err := servicesClient.GetService(ctx, &runpb.GetServiceRequest{Name: serviceName})
				return err
			},
		},
		{
			name: "CreateService",
			call: func(ctx context.Context) error {
				op, err := servicesClient.CreateService(ctx, &runpb.CreateServiceRequest{
					Parent:    parent,
					ServiceId: serviceID,
					Service: &runpb.Service{
						Template: &runpb.RevisionTemplate{
							Containers: []*runpb.Container{{
								Name:  "app",
								Image: "us-docker.pkg.dev/cloudrun/container/hello",
							}},
						},
					},
				})
				if err != nil {
					return err
				}
				serviceOperationName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "UpdateService",
			call: func(ctx context.Context) error {
				op, err := servicesClient.UpdateService(ctx, &runpb.UpdateServiceRequest{
					Service: &runpb.Service{
						Name: serviceName,
						Template: &runpb.RevisionTemplate{
							Containers: []*runpb.Container{{
								Name:  "app",
								Image: "us-docker.pkg.dev/cloudrun/container/hello",
							}},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"template.containers"}},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					serviceOperationName = name
				}
				return nil
			},
		},
		{
			name: "GetServiceOperation",
			call: func(ctx context.Context) error {
				if serviceOperationName == "" {
					return nil
				}
				_, err := servicesClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: serviceOperationName})
				return err
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context) error {
				it := jobsClient.ListJobs(ctx, &runpb.ListJobsRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetJob",
			call: func(ctx context.Context) error {
				_, err := jobsClient.GetJob(ctx, &runpb.GetJobRequest{Name: jobName})
				return err
			},
		},
		{
			name: "CreateJob",
			call: func(ctx context.Context) error {
				op, err := jobsClient.CreateJob(ctx, &runpb.CreateJobRequest{
					Parent: parent,
					JobId:  jobID,
					Job: &runpb.Job{
						Template: &runpb.ExecutionTemplate{
							Parallelism: 1,
							TaskCount:   1,
							Template: &runpb.TaskTemplate{
								Containers: []*runpb.Container{{
									Name:  "job",
									Image: "us-docker.pkg.dev/cloudrun/container/job",
								}},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				jobOperationName = strings.TrimSpace(op.Name())
				return nil
			},
		},
		{
			name: "UpdateJob",
			call: func(ctx context.Context) error {
				op, err := jobsClient.UpdateJob(ctx, &runpb.UpdateJobRequest{
					Job: &runpb.Job{
						Name: jobName,
						Template: &runpb.ExecutionTemplate{
							Parallelism: 1,
							TaskCount:   1,
							Template: &runpb.TaskTemplate{
								Containers: []*runpb.Container{{
									Name:  "job",
									Image: "us-docker.pkg.dev/cloudrun/container/job",
								}},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					jobOperationName = name
				}
				return nil
			},
		},
		{
			name: "RunJob",
			call: func(ctx context.Context) error {
				op, err := jobsClient.RunJob(ctx, &runpb.RunJobRequest{Name: jobName})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					jobOperationName = name
				}
				return nil
			},
		},
		{
			name: "GetJobOperation",
			call: func(ctx context.Context) error {
				if jobOperationName == "" {
					return nil
				}
				_, err := jobsClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: jobOperationName})
				return err
			},
		},
		{
			name: "ListExecutions",
			call: func(ctx context.Context) error {
				it := executionsClient.ListExecutions(ctx, &runpb.ListExecutionsRequest{Parent: jobName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetExecution",
			call: func(ctx context.Context) error {
				_, err := executionsClient.GetExecution(ctx, &runpb.GetExecutionRequest{Name: executionName})
				return err
			},
		},
		{
			name: "ListTasks",
			call: func(ctx context.Context) error {
				it := tasksClient.ListTasks(ctx, &runpb.ListTasksRequest{Parent: executionName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetTask",
			call: func(ctx context.Context) error {
				_, err := tasksClient.GetTask(ctx, &runpb.GetTaskRequest{Name: taskName})
				return err
			},
		},
		{
			name: "ListRevisions",
			call: func(ctx context.Context) error {
				it := revisionsClient.ListRevisions(ctx, &runpb.ListRevisionsRequest{Parent: serviceName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRevision",
			call: func(ctx context.Context) error {
				_, err := revisionsClient.GetRevision(ctx, &runpb.GetRevisionRequest{Name: revisionName})
				return err
			},
		},
		{
			name: "DeleteRevision",
			call: func(ctx context.Context) error {
				op, err := revisionsClient.DeleteRevision(ctx, &runpb.DeleteRevisionRequest{Name: revisionName})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "DeleteExecution",
			call: func(ctx context.Context) error {
				op, err := executionsClient.DeleteExecution(ctx, &runpb.DeleteExecutionRequest{Name: executionName})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "DeleteJob",
			call: func(ctx context.Context) error {
				op, err := jobsClient.DeleteJob(ctx, &runpb.DeleteJobRequest{Name: jobName})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "DeleteService",
			call: func(ctx context.Context) error {
				op, err := servicesClient.DeleteService(ctx, &runpb.DeleteServiceRequest{Name: serviceName})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(clientName string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close run %s client: %v\n", clientName, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
