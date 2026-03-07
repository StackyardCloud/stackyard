package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	confidentialcomputing "cloud.google.com/go/confidentialcomputing/apiv1"
	"cloud.google.com/go/confidentialcomputing/apiv1/confidentialcomputingpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *confidentialcomputing.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_CONFIDENTIALCOMPUTING_LOCATION", "us-central1")
	challengeID := getenv("STACKYARD_GCP_CONFIDENTIALCOMPUTING_CHALLENGE_ID", "ch-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	challengeName := fmt.Sprintf("%s/challenges/%s", locationName, challengeID)

	fmt.Printf("Stackyard GCP Confidential Computing apiv1 client using %s\n", apiEndpoint)

	client, err := confidentialcomputing.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
		option.WithUserAgent("stackyard-confidentialcomputing-apiv1"),
	)
	if err != nil {
		exitf("failed to create confidential computing client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *confidentialcomputing.Client) error {
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
			call: func(ctx context.Context, c *confidentialcomputing.Client) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "CreateChallenge",
			call: func(ctx context.Context, c *confidentialcomputing.Client) error {
				_, err := c.CreateChallenge(ctx, &confidentialcomputingpb.CreateChallengeRequest{
					Parent:    locationName,
					Challenge: &confidentialcomputingpb.Challenge{},
				})
				return err
			},
		},
		{
			name: "VerifyAttestation",
			call: func(ctx context.Context, c *confidentialcomputing.Client) error {
				_, err := c.VerifyAttestation(ctx, &confidentialcomputingpb.VerifyAttestationRequest{
					Challenge:      challengeName,
					TpmAttestation: &confidentialcomputingpb.TpmAttestation{},
				})
				return err
			},
		},
		{
			name: "VerifyAttestationConfidentialSpaceProfile",
			call: func(ctx context.Context, c *confidentialcomputing.Client) error {
				_, err := c.VerifyAttestation(ctx, &confidentialcomputingpb.VerifyAttestationRequest{
					Challenge:             challengeName,
					TpmAttestation:        &confidentialcomputingpb.TpmAttestation{},
					ConfidentialSpaceInfo: &confidentialcomputingpb.ConfidentialSpaceInfo{},
					TokenOptions:          &confidentialcomputingpb.TokenOptions{},
					Attester:              "confidential-space",
				})
				return err
			},
		},
		{
			name: "VerifyAttestationConfidentialGKEProfile",
			call: func(ctx context.Context, c *confidentialcomputing.Client) error {
				_, err := c.VerifyAttestation(ctx, &confidentialcomputingpb.VerifyAttestationRequest{
					Challenge:      challengeName,
					TpmAttestation: &confidentialcomputingpb.TpmAttestation{},
					TokenOptions:   &confidentialcomputingpb.TokenOptions{},
					Attester:       "confidential-gke",
				})
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
		fmt.Fprintf(os.Stderr, "warning: close confidential computing client: %v\n", err)
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
