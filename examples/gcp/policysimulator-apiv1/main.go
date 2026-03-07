package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	policysimulator "cloud.google.com/go/policysimulator/apiv1"
	"cloud.google.com/go/policysimulator/apiv1/policysimulatorpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type simulatorCallSpec struct {
	name string
	call func(context.Context, *policysimulator.SimulatorClient) error
}

type orgPolicyPreviewCallSpec struct {
	name string
	call func(context.Context, *policysimulator.OrgPolicyViolationsPreviewClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	orgID := getenv("STACKYARD_GCP_ORGANIZATION_ID", "123456789012")
	location := getenv("STACKYARD_GCP_LOCATION", "global")
	replayID := getenv("STACKYARD_GCP_REPLAY_ID", "replay-1")
	previewID := getenv("STACKYARD_GCP_POLICY_PREVIEW_ID", "preview-1")

	replayParent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	replayName := fmt.Sprintf("%s/replays/%s", replayParent, replayID)
	previewParent := fmt.Sprintf("organizations/%s/locations/%s", orgID, location)
	previewName := fmt.Sprintf("%s/orgPolicyViolationsPreviews/%s", previewParent, previewID)

	fmt.Printf("Stackyard GCP Policy Simulator apiv1 client using %s\n", apiEndpoint)

	simulatorClient, err := policysimulator.NewSimulatorRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create policysimulator simulator client: %v", err)
	}
	defer closeClient("policysimulator simulator", simulatorClient.Close)

	orgPolicyPreviewClient, err := policysimulator.NewOrgPolicyViolationsPreviewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create policysimulator org policy preview client: %v", err)
	}
	defer closeClient("policysimulator org policy preview", orgPolicyPreviewClient.Close)

	simulatorCalls := []simulatorCallSpec{
		{
			name: "CreateReplay",
			call: func(ctx context.Context, c *policysimulator.SimulatorClient) error {
				_, err := c.CreateReplay(ctx, &policysimulatorpb.CreateReplayRequest{
					Parent: replayParent,
					Replay: &policysimulatorpb.Replay{
						Config: &policysimulatorpb.ReplayConfig{},
					},
				})
				return err
			},
		},
		{
			name: "GetReplay",
			call: func(ctx context.Context, c *policysimulator.SimulatorClient) error {
				_, err := c.GetReplay(ctx, &policysimulatorpb.GetReplayRequest{Name: replayName})
				return err
			},
		},
		{
			name: "ListReplayResults",
			call: func(ctx context.Context, c *policysimulator.SimulatorClient) error {
				it := c.ListReplayResults(ctx, &policysimulatorpb.ListReplayResultsRequest{
					Parent:   replayName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
	}

	orgPolicyPreviewCalls := []orgPolicyPreviewCallSpec{
		{
			name: "ListOrgPolicyViolationsPreviews",
			call: func(ctx context.Context, c *policysimulator.OrgPolicyViolationsPreviewClient) error {
				it := c.ListOrgPolicyViolationsPreviews(ctx, &policysimulatorpb.ListOrgPolicyViolationsPreviewsRequest{
					Parent:   previewParent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOrgPolicyViolationsPreview",
			call: func(ctx context.Context, c *policysimulator.OrgPolicyViolationsPreviewClient) error {
				_, err := c.GetOrgPolicyViolationsPreview(ctx, &policysimulatorpb.GetOrgPolicyViolationsPreviewRequest{Name: previewName})
				return err
			},
		},
		{
			name: "CreateOrgPolicyViolationsPreview",
			call: func(ctx context.Context, c *policysimulator.OrgPolicyViolationsPreviewClient) error {
				_, err := c.CreateOrgPolicyViolationsPreview(ctx, &policysimulatorpb.CreateOrgPolicyViolationsPreviewRequest{
					Parent:                       previewParent,
					OrgPolicyViolationsPreviewId: previewID,
					OrgPolicyViolationsPreview: &policysimulatorpb.OrgPolicyViolationsPreview{
						Overlay: &policysimulatorpb.OrgPolicyOverlay{},
					},
				})
				return err
			},
		},
		{
			name: "ListOrgPolicyViolations",
			call: func(ctx context.Context, c *policysimulator.OrgPolicyViolationsPreviewClient) error {
				it := c.ListOrgPolicyViolations(ctx, &policysimulatorpb.ListOrgPolicyViolationsRequest{
					Parent:   previewName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
	}

	for _, call := range simulatorCalls {
		err := call.call(ctx, simulatorClient)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	for _, call := range orgPolicyPreviewCalls {
		err := call.call(ctx, orgPolicyPreviewClient)
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

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "notimplemented") || strings.Contains(lower, "not implemented")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
