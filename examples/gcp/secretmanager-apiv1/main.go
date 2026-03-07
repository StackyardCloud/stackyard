package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
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
	secretID := getenv("STACKYARD_GCP_SECRET_ID", "secret-1")

	projectParent := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectParent, locationID)
	secretName := fmt.Sprintf("%s/secrets/%s", projectParent, secretID)
	versionName := secretName + "/versions/1"
	versionDisabledName := secretName + "/versions/2"

	fmt.Printf("Stackyard GCP Secret Manager apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "secretmanager",
		},
	}

	client, err := secretmanager.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create secretmanager client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectParent,
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
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListSecrets",
			call: func(ctx context.Context) error {
				it := client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
					Parent:   projectParent,
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
			name: "CreateSecret",
			call: func(ctx context.Context) error {
				_, err := client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
					Parent:   projectParent,
					SecretId: secretID,
					Secret: &secretmanagerpb.Secret{
						Replication: &secretmanagerpb.Replication{
							Replication: &secretmanagerpb.Replication_Automatic_{
								Automatic: &secretmanagerpb.Replication_Automatic{},
							},
						},
						Labels: map[string]string{
							"env": "test",
						},
					},
				})
				return err
			},
		},
		{
			name: "GetSecret",
			call: func(ctx context.Context) error {
				_, err := client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{Name: secretName})
				return err
			},
		},
		{
			name: "UpdateSecret",
			call: func(ctx context.Context) error {
				_, err := client.UpdateSecret(ctx, &secretmanagerpb.UpdateSecretRequest{
					Secret: &secretmanagerpb.Secret{
						Name: secretName,
						Replication: &secretmanagerpb.Replication{
							Replication: &secretmanagerpb.Replication_Automatic_{
								Automatic: &secretmanagerpb.Replication_Automatic{},
							},
						},
						Labels: map[string]string{
							"team": "platform",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "AddSecretVersion",
			call: func(ctx context.Context) error {
				_, err := client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
					Parent: secretName,
					Payload: &secretmanagerpb.SecretPayload{
						Data: []byte("stackyard-secret"),
					},
				})
				return err
			},
		},
		{
			name: "ListSecretVersions",
			call: func(ctx context.Context) error {
				it := client.ListSecretVersions(ctx, &secretmanagerpb.ListSecretVersionsRequest{
					Parent:   secretName,
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
			name: "GetSecretVersion",
			call: func(ctx context.Context) error {
				_, err := client.GetSecretVersion(ctx, &secretmanagerpb.GetSecretVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "AccessSecretVersion",
			call: func(ctx context.Context) error {
				_, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "DisableSecretVersion",
			call: func(ctx context.Context) error {
				_, err := client.DisableSecretVersion(ctx, &secretmanagerpb.DisableSecretVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "EnableSecretVersion",
			call: func(ctx context.Context) error {
				_, err := client.EnableSecretVersion(ctx, &secretmanagerpb.EnableSecretVersionRequest{Name: versionDisabledName})
				return err
			},
		},
		{
			name: "DestroySecretVersion",
			call: func(ctx context.Context) error {
				_, err := client.DestroySecretVersion(ctx, &secretmanagerpb.DestroySecretVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := client.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: secretName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := client.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: secretName,
					Policy: &iampb.Policy{
						Version: 1,
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/secretmanager.secretAccessor",
								Members: []string{"user:stackyard@example.com"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context) error {
				_, err := client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    secretName,
					Permissions: []string{"secretmanager.secrets.get"},
				})
				return err
			},
		},
		{
			name: "DeleteSecret",
			call: func(ctx context.Context) error {
				return client.DeleteSecret(ctx, &secretmanagerpb.DeleteSecretRequest{Name: secretName})
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
		fmt.Fprintf(os.Stderr, "warning: close secretmanager client: %v\n", err)
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
