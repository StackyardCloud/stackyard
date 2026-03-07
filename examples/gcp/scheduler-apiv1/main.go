package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	scheduler "cloud.google.com/go/scheduler/apiv1"
	"cloud.google.com/go/scheduler/apiv1/schedulerpb"
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
	jobID := getenv("STACKYARD_GCP_SCHEDULER_JOB_ID", "job-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	jobName := fmt.Sprintf("%s/jobs/%s", parent, jobID)

	fmt.Printf("Stackyard GCP Cloud Scheduler scheduler/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "scheduler",
		},
	}

	client, err := scheduler.NewCloudSchedulerRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create scheduler client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     fmt.Sprintf("projects/%s", projectID),
					PageSize: 1,
				})
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
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context) error {
				it := client.ListJobs(ctx, &schedulerpb.ListJobsRequest{
					Parent:   parent,
					PageSize: 1,
				})
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
				_, err := client.GetJob(ctx, &schedulerpb.GetJobRequest{Name: jobName})
				return err
			},
		},
		{
			name: "CreateJob",
			call: func(ctx context.Context) error {
				_, err := client.CreateJob(ctx, &schedulerpb.CreateJobRequest{
					Parent: parent,
					Job: &schedulerpb.Job{
						Name:        jobName,
						Description: "Stackyard scheduler job",
						Schedule:    "*/15 * * * *",
						TimeZone:    "UTC",
						Target: &schedulerpb.Job_HttpTarget{
							HttpTarget: &schedulerpb.HttpTarget{
								Uri:        "https://example.com/hook",
								HttpMethod: schedulerpb.HttpMethod_POST,
								Headers: map[string]string{
									"Content-Type": "application/json",
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateJob",
			call: func(ctx context.Context) error {
				_, err := client.UpdateJob(ctx, &schedulerpb.UpdateJobRequest{
					Job: &schedulerpb.Job{
						Name:        jobName,
						Description: "Stackyard scheduler job updated",
						Schedule:    "*/10 * * * *",
						TimeZone:    "UTC",
						Target: &schedulerpb.Job_HttpTarget{
							HttpTarget: &schedulerpb.HttpTarget{
								Uri:        "https://example.com/hook",
								HttpMethod: schedulerpb.HttpMethod_POST,
								Headers: map[string]string{
									"Content-Type": "application/json",
								},
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"description", "schedule", "time_zone", "http_target"},
					},
				})
				return err
			},
		},
		{
			name: "PauseJob",
			call: func(ctx context.Context) error {
				_, err := client.PauseJob(ctx, &schedulerpb.PauseJobRequest{Name: jobName})
				return err
			},
		},
		{
			name: "ResumeJob",
			call: func(ctx context.Context) error {
				_, err := client.ResumeJob(ctx, &schedulerpb.ResumeJobRequest{Name: fmt.Sprintf("%s/jobs/job-paused", parent)})
				return err
			},
		},
		{
			name: "RunJob",
			call: func(ctx context.Context) error {
				_, err := client.RunJob(ctx, &schedulerpb.RunJobRequest{Name: jobName})
				return err
			},
		},
		{
			name: "DeleteJob",
			call: func(ctx context.Context) error {
				return client.DeleteJob(ctx, &schedulerpb.DeleteJobRequest{Name: jobName})
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close scheduler client: %v\n", err)
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
