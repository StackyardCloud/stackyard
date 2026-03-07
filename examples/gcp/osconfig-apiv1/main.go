package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	osconfig "cloud.google.com/go/osconfig/apiv1"
	"cloud.google.com/go/osconfig/apiv1/osconfigpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
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
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1-a")
	instanceID := getenv("STACKYARD_GCP_INSTANCE_ID", "instance-1")
	patchJobID := getenv("STACKYARD_GCP_OSCONFIG_PATCH_JOB_ID", "patch-job-1")
	patchDeploymentID := getenv("STACKYARD_GCP_OSCONFIG_PATCH_DEPLOYMENT_ID", "patch-deployment-1")
	assignmentID := getenv("STACKYARD_GCP_OSCONFIG_ASSIGNMENT_ID", "assignment-1")

	projectParent := fmt.Sprintf("projects/%s", projectID)
	patchJobName := fmt.Sprintf("%s/patchJobs/%s", projectParent, patchJobID)
	patchDeploymentName := fmt.Sprintf("%s/patchDeployments/%s", projectParent, patchDeploymentID)

	zonalParent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	assignmentName := fmt.Sprintf("%s/osPolicyAssignments/%s", zonalParent, assignmentID)
	inventoryName := fmt.Sprintf("%s/instances/%s/inventories/inventory", zonalParent, instanceID)
	inventoryParent := fmt.Sprintf("%s/instances/-", zonalParent)
	vulnerabilityName := fmt.Sprintf("%s/instances/%s/vulnerabilityReports/vulnerabilityReport", zonalParent, instanceID)
	reportsParent := fmt.Sprintf("%s/instances/-/osPolicyAssignments/-", zonalParent)

	fmt.Printf("Stackyard GCP OS Config apiv1 clients using %s\n", apiEndpoint)

	client, err := osconfig.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create osconfig client: %v", err)
	}
	defer closeClient("osconfig", client.Close)

	zonalClient, err := osconfig.NewOsConfigZonalRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create osconfig zonal client: %v", err)
	}
	defer closeClient("osconfig zonal", zonalClient.Close)

	calls := []callSpec{
		{
			name: "ExecutePatchJob",
			call: func(ctx context.Context) error {
				_, err := client.ExecutePatchJob(ctx, &osconfigpb.ExecutePatchJobRequest{
					Parent: projectParent,
					InstanceFilter: &osconfigpb.PatchInstanceFilter{
						GroupLabels: []*osconfigpb.PatchInstanceFilter_GroupLabel{},
					},
				})
				return err
			},
		},
		{
			name: "GetPatchJob",
			call: func(ctx context.Context) error {
				_, err := client.GetPatchJob(ctx, &osconfigpb.GetPatchJobRequest{Name: patchJobName})
				return err
			},
		},
		{
			name: "CancelPatchJob",
			call: func(ctx context.Context) error {
				_, err := client.CancelPatchJob(ctx, &osconfigpb.CancelPatchJobRequest{Name: patchJobName})
				return err
			},
		},
		{
			name: "ListPatchJobs",
			call: func(ctx context.Context) error {
				it := client.ListPatchJobs(ctx, &osconfigpb.ListPatchJobsRequest{
					Parent:   projectParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListPatchJobInstanceDetails",
			call: func(ctx context.Context) error {
				it := client.ListPatchJobInstanceDetails(ctx, &osconfigpb.ListPatchJobInstanceDetailsRequest{
					Parent:   patchJobName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreatePatchDeployment",
			call: func(ctx context.Context) error {
				_, err := client.CreatePatchDeployment(ctx, &osconfigpb.CreatePatchDeploymentRequest{
					Parent:            projectParent,
					PatchDeploymentId: patchDeploymentID,
					PatchDeployment: &osconfigpb.PatchDeployment{
						Name: patchDeploymentName,
					},
				})
				return err
			},
		},
		{
			name: "GetPatchDeployment",
			call: func(ctx context.Context) error {
				_, err := client.GetPatchDeployment(ctx, &osconfigpb.GetPatchDeploymentRequest{Name: patchDeploymentName})
				return err
			},
		},
		{
			name: "ListPatchDeployments",
			call: func(ctx context.Context) error {
				it := client.ListPatchDeployments(ctx, &osconfigpb.ListPatchDeploymentsRequest{
					Parent:   projectParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "UpdatePatchDeployment",
			call: func(ctx context.Context) error {
				_, err := client.UpdatePatchDeployment(ctx, &osconfigpb.UpdatePatchDeploymentRequest{
					PatchDeployment: &osconfigpb.PatchDeployment{
						Name:        patchDeploymentName,
						Description: "stackyard patched deployment",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "PausePatchDeployment",
			call: func(ctx context.Context) error {
				_, err := client.PausePatchDeployment(ctx, &osconfigpb.PausePatchDeploymentRequest{Name: patchDeploymentName})
				return err
			},
		},
		{
			name: "ResumePatchDeployment",
			call: func(ctx context.Context) error {
				_, err := client.ResumePatchDeployment(ctx, &osconfigpb.ResumePatchDeploymentRequest{Name: patchDeploymentName})
				return err
			},
		},
		{
			name: "DeletePatchDeployment",
			call: func(ctx context.Context) error {
				return client.DeletePatchDeployment(ctx, &osconfigpb.DeletePatchDeploymentRequest{Name: patchDeploymentName})
			},
		},
		{
			name: "CreateOSPolicyAssignment",
			call: func(ctx context.Context) error {
				_, err := zonalClient.CreateOSPolicyAssignment(ctx, &osconfigpb.CreateOSPolicyAssignmentRequest{
					Parent: zonalParent,
					OsPolicyAssignment: &osconfigpb.OSPolicyAssignment{
						Name: assignmentName,
					},
					OsPolicyAssignmentId: assignmentID,
				})
				return err
			},
		},
		{
			name: "GetOSPolicyAssignment",
			call: func(ctx context.Context) error {
				_, err := zonalClient.GetOSPolicyAssignment(ctx, &osconfigpb.GetOSPolicyAssignmentRequest{Name: assignmentName})
				return err
			},
		},
		{
			name: "ListOSPolicyAssignments",
			call: func(ctx context.Context) error {
				it := zonalClient.ListOSPolicyAssignments(ctx, &osconfigpb.ListOSPolicyAssignmentsRequest{
					Parent:   zonalParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "ListOSPolicyAssignmentRevisions",
			call: func(ctx context.Context) error {
				it := zonalClient.ListOSPolicyAssignmentRevisions(ctx, &osconfigpb.ListOSPolicyAssignmentRevisionsRequest{
					Name:     assignmentName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "UpdateOSPolicyAssignment",
			call: func(ctx context.Context) error {
				_, err := zonalClient.UpdateOSPolicyAssignment(ctx, &osconfigpb.UpdateOSPolicyAssignmentRequest{
					OsPolicyAssignment: &osconfigpb.OSPolicyAssignment{
						Name:        assignmentName,
						Description: "stackyard os policy assignment",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteOSPolicyAssignment",
			call: func(ctx context.Context) error {
				_, err := zonalClient.DeleteOSPolicyAssignment(ctx, &osconfigpb.DeleteOSPolicyAssignmentRequest{Name: assignmentName})
				return err
			},
		},
		{
			name: "GetOSPolicyAssignmentReport",
			call: func(ctx context.Context) error {
				_, err := zonalClient.GetOSPolicyAssignmentReport(ctx, &osconfigpb.GetOSPolicyAssignmentReportRequest{
					Name: reportsParent + "/reports/report-1",
				})
				return err
			},
		},
		{
			name: "ListOSPolicyAssignmentReports",
			call: func(ctx context.Context) error {
				it := zonalClient.ListOSPolicyAssignmentReports(ctx, &osconfigpb.ListOSPolicyAssignmentReportsRequest{
					Parent:   reportsParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetInventory",
			call: func(ctx context.Context) error {
				_, err := zonalClient.GetInventory(ctx, &osconfigpb.GetInventoryRequest{Name: inventoryName})
				return err
			},
		},
		{
			name: "ListInventories",
			call: func(ctx context.Context) error {
				it := zonalClient.ListInventories(ctx, &osconfigpb.ListInventoriesRequest{
					Parent:   inventoryParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetVulnerabilityReport",
			call: func(ctx context.Context) error {
				_, err := zonalClient.GetVulnerabilityReport(ctx, &osconfigpb.GetVulnerabilityReportRequest{Name: vulnerabilityName})
				return err
			},
		},
		{
			name: "ListVulnerabilityReports",
			call: func(ctx context.Context) error {
				it := zonalClient.ListVulnerabilityReports(ctx, &osconfigpb.ListVulnerabilityReportsRequest{
					Parent:   inventoryParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
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

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") || strings.Contains(text, "not implemented")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
