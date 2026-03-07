package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dlp "cloud.google.com/go/dlp/apiv2"
	"cloud.google.com/go/dlp/apiv2/dlppb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *dlp.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "global")
	templateID := getenv("STACKYARD_GCP_DLP_TEMPLATE_ID", "inspect-template-1")
	jobTriggerID := getenv("STACKYARD_GCP_DLP_JOB_TRIGGER_ID", "job-trigger-1")
	jobID := getenv("STACKYARD_GCP_DLP_JOB_ID", "dlp-job-1")
	storedInfoTypeID := getenv("STACKYARD_GCP_DLP_STORED_INFO_TYPE_ID", "stored-info-type-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	infoTypesParent := "locations/" + locationID
	inspectTemplateName := parent + "/inspectTemplates/" + templateID
	jobTriggerName := parent + "/jobTriggers/" + jobTriggerID
	dlpJobName := parent + "/dlpJobs/" + jobID
	storedInfoTypeName := parent + "/storedInfoTypes/" + storedInfoTypeID

	fmt.Printf("Stackyard GCP Sensitive Data Protection (DLP) apiv2 SDK client using %s (example directory dlp-apiv1)\n", apiEndpoint)

	client, err := dlp.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dlp client: %v", err)
	}
	defer closeClient("dlp client", client.Close)

	calls := []callSpec{
		{
			name: "InspectContent",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.InspectContent(ctx, &dlppb.InspectContentRequest{
					Parent: parent,
					Item:   sampleContentItem("customer email is user@example.com"),
				})
				return err
			},
		},
		{
			name: "DeidentifyContent",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.DeidentifyContent(ctx, &dlppb.DeidentifyContentRequest{
					Parent: parent,
					Item:   sampleContentItem("ssn 111-22-3333"),
				})
				return err
			},
		},
		{
			name: "ReidentifyContent",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.ReidentifyContent(ctx, &dlppb.ReidentifyContentRequest{
					Parent: parent,
					Item:   sampleContentItem("tokenized-value"),
				})
				return err
			},
		},
		{
			name: "ListInfoTypes",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.ListInfoTypes(ctx, &dlppb.ListInfoTypesRequest{
					Parent:       infoTypesParent,
					LanguageCode: "en-US",
					Filter:       "supported_by=INSPECT",
				})
				return err
			},
		},
		{
			name: "CreateInspectTemplate",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.CreateInspectTemplate(ctx, &dlppb.CreateInspectTemplateRequest{
					Parent:          parent,
					TemplateId:      templateID,
					InspectTemplate: &dlppb.InspectTemplate{},
				})
				return err
			},
		},
		{
			name: "GetInspectTemplate",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.GetInspectTemplate(ctx, &dlppb.GetInspectTemplateRequest{
					Name: inspectTemplateName,
				})
				return err
			},
		},
		{
			name: "ListInspectTemplates",
			call: func(ctx context.Context, c *dlp.Client) error {
				it := c.ListInspectTemplates(ctx, &dlppb.ListInspectTemplatesRequest{
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
			name: "DeleteInspectTemplate",
			call: func(ctx context.Context, c *dlp.Client) error {
				return c.DeleteInspectTemplate(ctx, &dlppb.DeleteInspectTemplateRequest{
					Name: inspectTemplateName,
				})
			},
		},
		{
			name: "CreateJobTrigger",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.CreateJobTrigger(ctx, &dlppb.CreateJobTriggerRequest{
					Parent:    parent,
					TriggerId: jobTriggerID,
					JobTrigger: &dlppb.JobTrigger{
						DisplayName: "stackyard-job-trigger",
					},
				})
				return err
			},
		},
		{
			name: "GetJobTrigger",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.GetJobTrigger(ctx, &dlppb.GetJobTriggerRequest{
					Name: jobTriggerName,
				})
				return err
			},
		},
		{
			name: "ListJobTriggers",
			call: func(ctx context.Context, c *dlp.Client) error {
				it := c.ListJobTriggers(ctx, &dlppb.ListJobTriggersRequest{
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
			name: "ActivateJobTrigger",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.ActivateJobTrigger(ctx, &dlppb.ActivateJobTriggerRequest{
					Name: jobTriggerName,
				})
				return err
			},
		},
		{
			name: "DeleteJobTrigger",
			call: func(ctx context.Context, c *dlp.Client) error {
				return c.DeleteJobTrigger(ctx, &dlppb.DeleteJobTriggerRequest{
					Name: jobTriggerName,
				})
			},
		},
		{
			name: "CreateDlpJob",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.CreateDlpJob(ctx, &dlppb.CreateDlpJobRequest{
					Parent: parent,
					JobId:  jobID,
					Job: &dlppb.CreateDlpJobRequest_InspectJob{
						InspectJob: &dlppb.InspectJobConfig{
							InspectConfig: &dlppb.InspectConfig{
								MinLikelihood: dlppb.Likelihood_POSSIBLE,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetDlpJob",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.GetDlpJob(ctx, &dlppb.GetDlpJobRequest{
					Name: dlpJobName,
				})
				return err
			},
		},
		{
			name: "ListDlpJobs",
			call: func(ctx context.Context, c *dlp.Client) error {
				it := c.ListDlpJobs(ctx, &dlppb.ListDlpJobsRequest{
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
			name: "CancelDlpJob",
			call: func(ctx context.Context, c *dlp.Client) error {
				return c.CancelDlpJob(ctx, &dlppb.CancelDlpJobRequest{
					Name: dlpJobName,
				})
			},
		},
		{
			name: "DeleteDlpJob",
			call: func(ctx context.Context, c *dlp.Client) error {
				return c.DeleteDlpJob(ctx, &dlppb.DeleteDlpJobRequest{
					Name: dlpJobName,
				})
			},
		},
		{
			name: "CreateStoredInfoType",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.CreateStoredInfoType(ctx, &dlppb.CreateStoredInfoTypeRequest{
					Parent:           parent,
					StoredInfoTypeId: storedInfoTypeID,
					Config:           &dlppb.StoredInfoTypeConfig{},
				})
				return err
			},
		},
		{
			name: "GetStoredInfoType",
			call: func(ctx context.Context, c *dlp.Client) error {
				_, err := c.GetStoredInfoType(ctx, &dlppb.GetStoredInfoTypeRequest{
					Name: storedInfoTypeName,
				})
				return err
			},
		},
		{
			name: "ListStoredInfoTypes",
			call: func(ctx context.Context, c *dlp.Client) error {
				it := c.ListStoredInfoTypes(ctx, &dlppb.ListStoredInfoTypesRequest{
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
			name: "DeleteStoredInfoType",
			call: func(ctx context.Context, c *dlp.Client) error {
				return c.DeleteStoredInfoType(ctx, &dlppb.DeleteStoredInfoTypeRequest{
					Name: storedInfoTypeName,
				})
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

func sampleContentItem(value string) *dlppb.ContentItem {
	return &dlppb.ContentItem{
		DataItem: &dlppb.ContentItem_Value{
			Value: value,
		},
	}
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
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
