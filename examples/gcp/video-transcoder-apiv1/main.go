package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	transcoder "cloud.google.com/go/video/transcoder/apiv1"
	"cloud.google.com/go/video/transcoder/apiv1/transcoderpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *transcoder.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	jobID := getenv("STACKYARD_GCP_TRANSCODER_JOB_ID", "job-1")
	jobTemplateID := getenv("STACKYARD_GCP_TRANSCODER_JOB_TEMPLATE_ID", "template-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	jobName := parent + "/jobs/" + jobID
	jobTemplateName := parent + "/jobTemplates/" + jobTemplateID

	fmt.Printf("Stackyard GCP Video Transcoder video/transcoder/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "video-transcoder",
		},
	}

	client, err := transcoder.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create video transcoder client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "CreateJobTemplate",
			call: func(ctx context.Context, c *transcoder.Client) error {
				_, err := c.CreateJobTemplate(ctx, &transcoderpb.CreateJobTemplateRequest{
					Parent:        parent,
					JobTemplateId: jobTemplateID,
					JobTemplate: &transcoderpb.JobTemplate{
						Name: jobTemplateName,
						Config: &transcoderpb.JobConfig{
							Output: &transcoderpb.Output{
								Uri: fmt.Sprintf("gs://stackyard-outputs/templates/%s/", jobTemplateID),
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListJobTemplates",
			call: func(ctx context.Context, c *transcoder.Client) error {
				it := c.ListJobTemplates(ctx, &transcoderpb.ListJobTemplatesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				item, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListJobTemplates returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(item.GetName()) == "" {
					return errors.New("ListJobTemplates returned template without name")
				}
				return nil
			},
		},
		{
			name: "GetJobTemplate",
			call: func(ctx context.Context, c *transcoder.Client) error {
				template, err := c.GetJobTemplate(ctx, &transcoderpb.GetJobTemplateRequest{Name: jobTemplateName})
				if err != nil {
					return err
				}
				if template.GetName() != jobTemplateName {
					return fmt.Errorf("GetJobTemplate returned unexpected name: %q", template.GetName())
				}
				return nil
			},
		},
		{
			name: "CreateJob",
			call: func(ctx context.Context, c *transcoder.Client) error {
				_, err := c.CreateJob(ctx, &transcoderpb.CreateJobRequest{
					Parent: parent,
					Job: &transcoderpb.Job{
						Name:      jobName,
						InputUri:  fmt.Sprintf("gs://stackyard-inputs/%s.mp4", jobID),
						OutputUri: fmt.Sprintf("gs://stackyard-outputs/%s/", jobID),
						JobConfig: &transcoderpb.Job_TemplateId{TemplateId: "preset/web-hd"},
					},
				})
				return err
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context, c *transcoder.Client) error {
				it := c.ListJobs(ctx, &transcoderpb.ListJobsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				item, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListJobs returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(item.GetName()) == "" {
					return errors.New("ListJobs returned job without name")
				}
				return nil
			},
		},
		{
			name: "GetJob",
			call: func(ctx context.Context, c *transcoder.Client) error {
				job, err := c.GetJob(ctx, &transcoderpb.GetJobRequest{Name: jobName})
				if err != nil {
					return err
				}
				if job.GetName() != jobName {
					return fmt.Errorf("GetJob returned unexpected name: %q", job.GetName())
				}
				return nil
			},
		},
		{
			name: "DeleteJob",
			call: func(ctx context.Context, c *transcoder.Client) error {
				return c.DeleteJob(ctx, &transcoderpb.DeleteJobRequest{Name: jobName})
			},
		},
		{
			name: "DeleteJobTemplate",
			call: func(ctx context.Context, c *transcoder.Client) error {
				return c.DeleteJobTemplate(ctx, &transcoderpb.DeleteJobTemplateRequest{Name: jobTemplateName})
			},
		},
		{
			name: "CreateJobTemplateInvalidID",
			call: func(ctx context.Context, c *transcoder.Client) error {
				_, err := c.CreateJobTemplate(ctx, &transcoderpb.CreateJobTemplateRequest{
					Parent:        parent,
					JobTemplateId: "1bad-template-id",
					JobTemplate: &transcoderpb.JobTemplate{
						Name: parent + "/jobTemplates/1bad-template-id",
					},
				})
				if err == nil {
					return errors.New("expected invalid argument for CreateJobTemplateInvalidID")
				}
				if isInvalidArgument(err) {
					return nil
				}
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
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
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotImplemented {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func isInvalidArgument(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.InvalidArgument {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusBadRequest {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "invalidargument")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close video transcoder client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
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
