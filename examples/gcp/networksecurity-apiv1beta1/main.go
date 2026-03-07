package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	networksecurity "cloud.google.com/go/networksecurity/apiv1beta1"
	"cloud.google.com/go/networksecurity/apiv1beta1/networksecuritypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *networksecurity.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	authorizationPolicyID := getenv("STACKYARD_GCP_NETWORKSECURITY_AUTHZ_POLICY_ID", "authz-a")
	clientTLSPolicyID := getenv("STACKYARD_GCP_NETWORKSECURITY_CLIENT_TLS_POLICY_ID", "clienttls-a")
	serverTLSPolicyID := getenv("STACKYARD_GCP_NETWORKSECURITY_SERVER_TLS_POLICY_ID", "servertls-a")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	authorizationPolicyName := fmt.Sprintf("%s/authorizationPolicies/%s", parent, authorizationPolicyID)
	clientTLSPolicyName := fmt.Sprintf("%s/clientTlsPolicies/%s", parent, clientTLSPolicyID)
	serverTLSPolicyName := fmt.Sprintf("%s/serverTlsPolicies/%s", parent, serverTLSPolicyID)

	fmt.Printf("Stackyard GCP Network Security apiv1beta1 client using %s\n", apiEndpoint)

	client, err := networksecurity.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create network security client: %v", err)
	}
	defer closeClient("network security", client.Close)

	calls := []callSpec{
		{
			name: "ListAuthorizationPolicies",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				it := c.ListAuthorizationPolicies(ctx, &networksecuritypb.ListAuthorizationPoliciesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetAuthorizationPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.GetAuthorizationPolicy(ctx, &networksecuritypb.GetAuthorizationPolicyRequest{Name: authorizationPolicyName})
				return err
			},
		},
		{
			name: "CreateAuthorizationPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.CreateAuthorizationPolicy(ctx, &networksecuritypb.CreateAuthorizationPolicyRequest{
					Parent:                parent,
					AuthorizationPolicyId: authorizationPolicyID,
					AuthorizationPolicy: &networksecuritypb.AuthorizationPolicy{
						Name:        authorizationPolicyName,
						Description: "stackyard authorization policy",
						Action:      networksecuritypb.AuthorizationPolicy_ALLOW,
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteAuthorizationPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.DeleteAuthorizationPolicy(ctx, &networksecuritypb.DeleteAuthorizationPolicyRequest{Name: authorizationPolicyName})
				return err
			},
		},
		{
			name: "ListClientTlsPolicies",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				it := c.ListClientTlsPolicies(ctx, &networksecuritypb.ListClientTlsPoliciesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetClientTlsPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.GetClientTlsPolicy(ctx, &networksecuritypb.GetClientTlsPolicyRequest{Name: clientTLSPolicyName})
				return err
			},
		},
		{
			name: "CreateClientTlsPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.CreateClientTlsPolicy(ctx, &networksecuritypb.CreateClientTlsPolicyRequest{
					Parent:            parent,
					ClientTlsPolicyId: clientTLSPolicyID,
					ClientTlsPolicy: &networksecuritypb.ClientTlsPolicy{
						Name:        clientTLSPolicyName,
						Description: "stackyard client tls policy",
						Sni:         "secure.example.com",
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteClientTlsPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.DeleteClientTlsPolicy(ctx, &networksecuritypb.DeleteClientTlsPolicyRequest{Name: clientTLSPolicyName})
				return err
			},
		},
		{
			name: "ListServerTlsPolicies",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				it := c.ListServerTlsPolicies(ctx, &networksecuritypb.ListServerTlsPoliciesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetServerTlsPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.GetServerTlsPolicy(ctx, &networksecuritypb.GetServerTlsPolicyRequest{Name: serverTLSPolicyName})
				return err
			},
		},
		{
			name: "CreateServerTlsPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.CreateServerTlsPolicy(ctx, &networksecuritypb.CreateServerTlsPolicyRequest{
					Parent:            parent,
					ServerTlsPolicyId: serverTLSPolicyID,
					ServerTlsPolicy: &networksecuritypb.ServerTlsPolicy{
						Name:        serverTLSPolicyName,
						Description: "stackyard server tls policy",
						AllowOpen:   true,
						Labels:      map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteServerTlsPolicy",
			call: func(ctx context.Context, c *networksecurity.Client) error {
				_, err := c.DeleteServerTlsPolicy(ctx, &networksecuritypb.DeleteServerTlsPolicyRequest{Name: serverTLSPolicyName})
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

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
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
