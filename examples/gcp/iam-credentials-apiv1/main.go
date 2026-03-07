package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

type callSpec struct {
	name string
	call func(context.Context, *credentials.IamCredentialsClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	serviceAccount := getenv("STACKYARD_GCP_IAM_CREDENTIALS_SERVICE_ACCOUNT", "stackyard@example.iam.gserviceaccount.com")
	audience := getenv("STACKYARD_GCP_IAM_CREDENTIALS_AUDIENCE", "https://example.com")
	scope := getenv("STACKYARD_GCP_IAM_CREDENTIALS_SCOPE", "https://www.googleapis.com/auth/cloud-platform")
	serviceAccountName := "projects/" + projectID + "/serviceAccounts/" + serviceAccount

	fmt.Printf("Stackyard GCP IAM Credentials apiv1 client using %s\n", apiEndpoint)

	client, err := credentials.NewIamCredentialsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iam credentials client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "GenerateAccessToken",
			call: func(ctx context.Context, c *credentials.IamCredentialsClient) error {
				_, err := c.GenerateAccessToken(ctx, &credentialspb.GenerateAccessTokenRequest{
					Name:      serviceAccountName,
					Scope:     []string{scope},
					Lifetime:  durationpb.New(time.Hour),
					Delegates: []string{},
				})
				return err
			},
		},
		{
			name: "GenerateIdToken",
			call: func(ctx context.Context, c *credentials.IamCredentialsClient) error {
				_, err := c.GenerateIdToken(ctx, &credentialspb.GenerateIdTokenRequest{
					Name:         serviceAccountName,
					Audience:     audience,
					IncludeEmail: true,
					Delegates:    []string{},
				})
				return err
			},
		},
		{
			name: "SignBlob",
			call: func(ctx context.Context, c *credentials.IamCredentialsClient) error {
				_, err := c.SignBlob(ctx, &credentialspb.SignBlobRequest{
					Name:      serviceAccountName,
					Payload:   []byte("stackyard-blob"),
					Delegates: []string{},
				})
				return err
			},
		},
		{
			name: "SignJwt",
			call: func(ctx context.Context, c *credentials.IamCredentialsClient) error {
				_, err := c.SignJwt(ctx, &credentialspb.SignJwtRequest{
					Name:      serviceAccountName,
					Payload:   `{"sub":"stackyard@example.com","iat":1710000000}`,
					Delegates: []string{},
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
		fmt.Fprintf(os.Stderr, "warning: close iam credentials client: %v\n", err)
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
