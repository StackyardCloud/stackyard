package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	securityposture "cloud.google.com/go/securityposture/apiv1"
	"cloud.google.com/go/securityposture/apiv1/securityposturepb"
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
	call func(context.Context, *securityposture.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	orgID := getenv("STACKYARD_GCP_ORGANIZATION_ID", "123456")
	locationID := getenv("STACKYARD_GCP_LOCATION", "global")
	postureID := getenv("STACKYARD_GCP_SECURITY_POSTURE_ID", "posture-1")
	deploymentID := getenv("STACKYARD_GCP_SECURITY_POSTURE_DEPLOYMENT_ID", "deployment-1")
	templateID := getenv("STACKYARD_GCP_SECURITY_POSTURE_TEMPLATE_ID", "template-1")

	orgParent := fmt.Sprintf("organizations/%s", orgID)
	parent := fmt.Sprintf("%s/locations/%s", orgParent, locationID)
	postureName := fmt.Sprintf("%s/postures/%s", parent, postureID)
	deploymentName := fmt.Sprintf("%s/postureDeployments/%s", parent, deploymentID)
	templateName := fmt.Sprintf("%s/postureTemplates/%s", parent, templateID)
	operationName := fmt.Sprintf("%s/operations/op-1", parent)

	fmt.Printf("Stackyard GCP Security Posture apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "securityposture",
		},
	}

	client, err := securityposture.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create securityposture client: %v", err)
	}
	defer closeClient(client.Close)

	basePosture := &securityposturepb.Posture{
		Name:        postureName,
		State:       securityposturepb.Posture_ACTIVE,
		Description: "Stackyard posture",
		PolicySets: []*securityposturepb.PolicySet{
			{
				PolicySetId: "baseline",
				Policies: []*securityposturepb.Policy{
					{
						PolicyId: "sha-001",
						Constraint: &securityposturepb.Constraint{
							Implementation: &securityposturepb.Constraint_SecurityHealthAnalyticsModule{
								SecurityHealthAnalyticsModule: &securityposturepb.SecurityHealthAnalyticsModule{
									ModuleName:            "BIGQUERY_TABLE_CMEK_DISABLED",
									ModuleEnablementState: securityposturepb.EnablementState_ENABLED,
								},
							},
						},
					},
				},
			},
		},
	}

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *securityposture.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: orgParent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *securityposture.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListPostures",
			call: func(ctx context.Context, c *securityposture.Client) error {
				it := c.ListPostures(ctx, &securityposturepb.ListPosturesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListPostureRevisions",
			call: func(ctx context.Context, c *securityposture.Client) error {
				it := c.ListPostureRevisions(ctx, &securityposturepb.ListPostureRevisionsRequest{Name: postureName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetPosture",
			call: func(ctx context.Context, c *securityposture.Client) error {
				_, err := c.GetPosture(ctx, &securityposturepb.GetPostureRequest{Name: postureName, RevisionId: "0000000a"})
				return err
			},
		},
		{
			name: "CreatePosture",
			call: func(ctx context.Context, c *securityposture.Client) error {
				op, err := c.CreatePosture(ctx, &securityposturepb.CreatePostureRequest{
					Parent:    parent,
					PostureId: postureID,
					Posture:   basePosture,
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
			name: "UpdatePosture",
			call: func(ctx context.Context, c *securityposture.Client) error {
				op, err := c.UpdatePosture(ctx, &securityposturepb.UpdatePostureRequest{
					Posture: &securityposturepb.Posture{
						Name:        postureName,
						Description: "Updated Stackyard posture",
						Etag:        "etag-" + postureID,
					},
					RevisionId: "0000000a",
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
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
			name: "DeletePosture",
			call: func(ctx context.Context, c *securityposture.Client) error {
				op, err := c.DeletePosture(ctx, &securityposturepb.DeletePostureRequest{Name: postureName})
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
			name: "ExtractPosture",
			call: func(ctx context.Context, c *securityposture.Client) error {
				op, err := c.ExtractPosture(ctx, &securityposturepb.ExtractPostureRequest{
					Parent:    parent,
					PostureId: "posture-extracted",
					Workload:  "project/123456789",
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
			name: "ListPostureDeployments",
			call: func(ctx context.Context, c *securityposture.Client) error {
				it := c.ListPostureDeployments(ctx, &securityposturepb.ListPostureDeploymentsRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetPostureDeployment",
			call: func(ctx context.Context, c *securityposture.Client) error {
				_, err := c.GetPostureDeployment(ctx, &securityposturepb.GetPostureDeploymentRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "CreatePostureDeployment",
			call: func(ctx context.Context, c *securityposture.Client) error {
				op, err := c.CreatePostureDeployment(ctx, &securityposturepb.CreatePostureDeploymentRequest{
					Parent:              parent,
					PostureDeploymentId: deploymentID,
					PostureDeployment: &securityposturepb.PostureDeployment{
						Name:              deploymentName,
						TargetResource:    "projects/123456789",
						PostureId:         postureName,
						PostureRevisionId: "0000000a",
						Description:       "Stackyard deployment",
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
			name: "UpdatePostureDeployment",
			call: func(ctx context.Context, c *securityposture.Client) error {
				op, err := c.UpdatePostureDeployment(ctx, &securityposturepb.UpdatePostureDeploymentRequest{
					PostureDeployment: &securityposturepb.PostureDeployment{
						Name:        deploymentName,
						Description: "Updated Stackyard deployment",
						Etag:        "etag-" + deploymentID,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
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
			name: "DeletePostureDeployment",
			call: func(ctx context.Context, c *securityposture.Client) error {
				op, err := c.DeletePostureDeployment(ctx, &securityposturepb.DeletePostureDeploymentRequest{Name: deploymentName})
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
			name: "ListPostureTemplates",
			call: func(ctx context.Context, c *securityposture.Client) error {
				it := c.ListPostureTemplates(ctx, &securityposturepb.ListPostureTemplatesRequest{Parent: parent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetPostureTemplate",
			call: func(ctx context.Context, c *securityposture.Client) error {
				_, err := c.GetPostureTemplate(ctx, &securityposturepb.GetPostureTemplateRequest{Name: templateName, RevisionId: "00000001"})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *securityposture.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *securityposture.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: parent, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *securityposture.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *securityposture.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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
		fmt.Fprintf(os.Stderr, "warning: close securityposture client: %v\n", err)
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
