package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	rapidmigrationassessment "cloud.google.com/go/rapidmigrationassessment/apiv1"
	"cloud.google.com/go/rapidmigrationassessment/apiv1/rapidmigrationassessmentpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *rapidmigrationassessment.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	collectorID := getenv("STACKYARD_GCP_COLLECTOR_ID", "collector-1")
	annotationID := getenv("STACKYARD_GCP_ANNOTATION_ID", "annotation-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := "projects/" + projectID
	locationName := parent
	collectorName := parent + "/collectors/" + collectorID
	annotationName := parent + "/annotations/" + annotationID
	operationName := parent + "/operations/createCollector." + collectorID

	fmt.Printf("Stackyard GCP Rapid Migration Assessment apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "rapidmigrationassessment",
		},
	}

	client, err := rapidmigrationassessment.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create rapidmigrationassessment client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				it := c.ListLocations(ctx, &location.ListLocationsRequest{
					Name:     projectName,
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
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.GetLocation(ctx, &location.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListCollectors",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				it := c.ListCollectors(ctx, &rapidmigrationassessmentpb.ListCollectorsRequest{
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
			name: "GetCollector",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.GetCollector(ctx, &rapidmigrationassessmentpb.GetCollectorRequest{Name: collectorName})
				return err
			},
		},
		{
			name: "CreateCollector",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				op, err := c.CreateCollector(ctx, &rapidmigrationassessmentpb.CreateCollectorRequest{
					Parent:      parent,
					CollectorId: collectorID,
					Collector: &rapidmigrationassessmentpb.Collector{
						DisplayName: "Stackyard Collector",
						Description: "Stackyard staged collector",
						Labels: map[string]string{
							"env": "staged",
						},
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "UpdateCollector",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.UpdateCollector(ctx, &rapidmigrationassessmentpb.UpdateCollectorRequest{
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"display_name"},
					},
					Collector: &rapidmigrationassessmentpb.Collector{
						Name:        collectorName,
						DisplayName: "Stackyard Collector Updated",
						Description: "Stackyard staged collector updated",
					},
				})
				return err
			},
		},
		{
			name: "PauseCollector",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.PauseCollector(ctx, &rapidmigrationassessmentpb.PauseCollectorRequest{
					Name: collectorName,
				})
				return err
			},
		},
		{
			name: "ResumeCollector",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.ResumeCollector(ctx, &rapidmigrationassessmentpb.ResumeCollectorRequest{
					Name: collectorName,
				})
				return err
			},
		},
		{
			name: "RegisterCollector",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.RegisterCollector(ctx, &rapidmigrationassessmentpb.RegisterCollectorRequest{
					Name: collectorName,
				})
				return err
			},
		},
		{
			name: "CreateAnnotation",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.CreateAnnotation(ctx, &rapidmigrationassessmentpb.CreateAnnotationRequest{
					Parent: parent,
					Annotation: &rapidmigrationassessmentpb.Annotation{
						Type: rapidmigrationassessmentpb.Annotation_TYPE_LEGACY_EXPORT_CONSENT,
						Labels: map[string]string{
							"source": "stackyard",
						},
					},
				})
				return err
			},
		},
		{
			name: "GetAnnotation",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.GetAnnotation(ctx, &rapidmigrationassessmentpb.GetAnnotationRequest{Name: annotationName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "DeleteCollector",
			call: func(ctx context.Context, c *rapidmigrationassessment.Client) error {
				_, err := c.DeleteCollector(ctx, &rapidmigrationassessmentpb.DeleteCollectorRequest{Name: collectorName})
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
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close rapidmigrationassessment client: %v\n", err)
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
