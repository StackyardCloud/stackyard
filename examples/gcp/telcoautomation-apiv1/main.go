package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	telcoautomation "cloud.google.com/go/telcoautomation/apiv1"
	"cloud.google.com/go/telcoautomation/apiv1/telcoautomationpb"
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
	call func(context.Context, *telcoautomation.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	clusterID := getenv("STACKYARD_GCP_CLUSTER_ID", "cluster-1")
	edgeSlmID := getenv("STACKYARD_GCP_EDGE_SLM_ID", "edgeslm-1")
	blueprintID := getenv("STACKYARD_GCP_BLUEPRINT_ID", "blueprint-draft")
	deploymentID := getenv("STACKYARD_GCP_DEPLOYMENT_ID", "deployment-draft")
	hydratedID := getenv("STACKYARD_GCP_HYDRATED_ID", "hydrated-draft")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := "projects/" + projectID
	clusterName := parent + "/orchestrationClusters/" + clusterID
	edgeSlmName := parent + "/edgeSlms/" + edgeSlmID
	blueprintName := clusterName + "/blueprints/" + blueprintID
	proposedBlueprintName := clusterName + "/blueprints/blueprint-proposed"
	publicBlueprintName := parent + "/publicBlueprints/public-blueprint-1"
	deploymentName := clusterName + "/deployments/" + deploymentID
	deploymentAppliedName := clusterName + "/deployments/deployment-applied"
	hydratedName := deploymentName + "/hydratedDeployments/" + hydratedID
	operationName := parent + "/operations/createOrchestrationCluster." + clusterID

	fmt.Printf("Stackyard GCP Telco Automation apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "telcoautomation",
		},
	}

	client, err := telcoautomation.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create telcoautomation client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
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
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListOrchestrationClusters",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListOrchestrationClusters(ctx, &telcoautomationpb.ListOrchestrationClustersRequest{
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
			name: "GetOrchestrationCluster",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetOrchestrationCluster(ctx, &telcoautomationpb.GetOrchestrationClusterRequest{Name: clusterName})
				return err
			},
		},
		{
			name: "CreateOrchestrationCluster",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				op, err := c.CreateOrchestrationCluster(ctx, &telcoautomationpb.CreateOrchestrationClusterRequest{
					Parent:                 parent,
					OrchestrationClusterId: clusterID,
					OrchestrationCluster: &telcoautomationpb.OrchestrationCluster{
						Name: clusterName,
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				cluster, err := op.Wait(ctx)
				if err == nil && cluster != nil && strings.TrimSpace(cluster.GetName()) != "" {
					clusterName = cluster.GetName()
				}
				return err
			},
		},
		{
			name: "ListEdgeSlms",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListEdgeSlms(ctx, &telcoautomationpb.ListEdgeSlmsRequest{
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
			name: "GetEdgeSlm",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetEdgeSlm(ctx, &telcoautomationpb.GetEdgeSlmRequest{Name: edgeSlmName})
				return err
			},
		},
		{
			name: "CreateEdgeSlm",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				op, err := c.CreateEdgeSlm(ctx, &telcoautomationpb.CreateEdgeSlmRequest{
					Parent:    parent,
					EdgeSlmId: edgeSlmID,
					EdgeSlm: &telcoautomationpb.EdgeSlm{
						Name:                 edgeSlmName,
						OrchestrationCluster: clusterName,
					},
				})
				if err != nil {
					return err
				}
				edgeSlm, err := op.Wait(ctx)
				if err == nil && edgeSlm != nil && strings.TrimSpace(edgeSlm.GetName()) != "" {
					edgeSlmName = edgeSlm.GetName()
				}
				return err
			},
		},
		{
			name: "CreateBlueprint",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				blueprint, err := c.CreateBlueprint(ctx, &telcoautomationpb.CreateBlueprintRequest{
					Parent:      clusterName,
					BlueprintId: blueprintID,
					Blueprint: &telcoautomationpb.Blueprint{
						Name:            blueprintName,
						SourceBlueprint: publicBlueprintName,
						DisplayName:     "Stackyard Blueprint",
					},
				})
				if err == nil && blueprint != nil && strings.TrimSpace(blueprint.GetName()) != "" {
					blueprintName = strings.Split(blueprint.GetName(), "@")[0]
				}
				return err
			},
		},
		{
			name: "UpdateBlueprint",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.UpdateBlueprint(ctx, &telcoautomationpb.UpdateBlueprintRequest{
					Blueprint: &telcoautomationpb.Blueprint{
						Name:            blueprintName,
						SourceBlueprint: publicBlueprintName,
						DisplayName:     "Stackyard Blueprint Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "GetBlueprint",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetBlueprint(ctx, &telcoautomationpb.GetBlueprintRequest{Name: blueprintName})
				return err
			},
		},
		{
			name: "ListBlueprints",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListBlueprints(ctx, &telcoautomationpb.ListBlueprintsRequest{
					Parent:   clusterName,
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
			name: "ProposeBlueprint",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.ProposeBlueprint(ctx, &telcoautomationpb.ProposeBlueprintRequest{Name: blueprintName})
				return err
			},
		},
		{
			name: "ApproveBlueprint",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.ApproveBlueprint(ctx, &telcoautomationpb.ApproveBlueprintRequest{Name: proposedBlueprintName})
				return err
			},
		},
		{
			name: "RejectBlueprint",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.RejectBlueprint(ctx, &telcoautomationpb.RejectBlueprintRequest{Name: proposedBlueprintName})
				return err
			},
		},
		{
			name: "ListBlueprintRevisions",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListBlueprintRevisions(ctx, &telcoautomationpb.ListBlueprintRevisionsRequest{
					Name:     blueprintName,
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
			name: "SearchBlueprintRevisions",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.SearchBlueprintRevisions(ctx, &telcoautomationpb.SearchBlueprintRevisionsRequest{
					Parent:   clusterName,
					Query:    "latest=true",
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
			name: "DiscardBlueprintChanges",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.DiscardBlueprintChanges(ctx, &telcoautomationpb.DiscardBlueprintChangesRequest{Name: blueprintName})
				return err
			},
		},
		{
			name: "ListPublicBlueprints",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListPublicBlueprints(ctx, &telcoautomationpb.ListPublicBlueprintsRequest{
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
			name: "GetPublicBlueprint",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetPublicBlueprint(ctx, &telcoautomationpb.GetPublicBlueprintRequest{Name: publicBlueprintName})
				return err
			},
		},
		{
			name: "CreateDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				deployment, err := c.CreateDeployment(ctx, &telcoautomationpb.CreateDeploymentRequest{
					Parent:       clusterName,
					DeploymentId: deploymentID,
					Deployment: &telcoautomationpb.Deployment{
						Name:                    deploymentName,
						SourceBlueprintRevision: clusterName + "/blueprints/blueprint-approved@rev-3",
						DisplayName:             "Stackyard Deployment",
					},
				})
				if err == nil && deployment != nil && strings.TrimSpace(deployment.GetName()) != "" {
					deploymentName = strings.Split(deployment.GetName(), "@")[0]
				}
				return err
			},
		},
		{
			name: "UpdateDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.UpdateDeployment(ctx, &telcoautomationpb.UpdateDeploymentRequest{
					Deployment: &telcoautomationpb.Deployment{
						Name:                    deploymentName,
						SourceBlueprintRevision: clusterName + "/blueprints/blueprint-approved@rev-3",
						DisplayName:             "Stackyard Deployment Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "source_blueprint_revision"}},
				})
				return err
			},
		},
		{
			name: "GetDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetDeployment(ctx, &telcoautomationpb.GetDeploymentRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "ListDeployments",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListDeployments(ctx, &telcoautomationpb.ListDeploymentsRequest{
					Parent:   clusterName,
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
			name: "ListDeploymentRevisions",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListDeploymentRevisions(ctx, &telcoautomationpb.ListDeploymentRevisionsRequest{
					Name:     deploymentName,
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
			name: "SearchDeploymentRevisions",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.SearchDeploymentRevisions(ctx, &telcoautomationpb.SearchDeploymentRevisionsRequest{
					Parent:   clusterName,
					Query:    "latest=true",
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
			name: "DiscardDeploymentChanges",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.DiscardDeploymentChanges(ctx, &telcoautomationpb.DiscardDeploymentChangesRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "ApplyDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.ApplyDeployment(ctx, &telcoautomationpb.ApplyDeploymentRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "ComputeDeploymentStatus",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.ComputeDeploymentStatus(ctx, &telcoautomationpb.ComputeDeploymentStatusRequest{Name: deploymentName})
				return err
			},
		},
		{
			name: "RollbackDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.RollbackDeployment(ctx, &telcoautomationpb.RollbackDeploymentRequest{
					Name:       deploymentAppliedName,
					RevisionId: "rev-1",
				})
				return err
			},
		},
		{
			name: "RemoveDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				return c.RemoveDeployment(ctx, &telcoautomationpb.RemoveDeploymentRequest{Name: deploymentName})
			},
		},
		{
			name: "GetHydratedDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetHydratedDeployment(ctx, &telcoautomationpb.GetHydratedDeploymentRequest{Name: hydratedName})
				return err
			},
		},
		{
			name: "ListHydratedDeployments",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				it := c.ListHydratedDeployments(ctx, &telcoautomationpb.ListHydratedDeploymentsRequest{
					Parent:   deploymentName,
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
			name: "UpdateHydratedDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.UpdateHydratedDeployment(ctx, &telcoautomationpb.UpdateHydratedDeploymentRequest{
					HydratedDeployment: &telcoautomationpb.HydratedDeployment{
						Name: hydratedName,
						Files: []*telcoautomationpb.File{
							{
								Path:     "hydrated/site.yaml",
								Content:  "kind: ConfigMap",
								Editable: true,
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"files"}},
				})
				return err
			},
		},
		{
			name: "ApplyHydratedDeployment",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.ApplyHydratedDeployment(ctx, &telcoautomationpb.ApplyHydratedDeploymentRequest{Name: hydratedName})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "DeleteEdgeSlm",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				op, err := c.DeleteEdgeSlm(ctx, &telcoautomationpb.DeleteEdgeSlmRequest{Name: edgeSlmName})
				if err != nil {
					return err
				}
				return op.Wait(ctx)
			},
		},
		{
			name: "DeleteOrchestrationCluster",
			call: func(ctx context.Context, c *telcoautomation.Client) error {
				op, err := c.DeleteOrchestrationCluster(ctx, &telcoautomationpb.DeleteOrchestrationClusterRequest{Name: clusterName})
				if err != nil {
					return err
				}
				return op.Wait(ctx)
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
		fmt.Fprintf(os.Stderr, "warning: close telcoautomation client: %v\n", err)
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
