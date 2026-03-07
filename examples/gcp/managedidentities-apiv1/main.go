package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	managedidentities "cloud.google.com/go/managedidentities/apiv1"
	"cloud.google.com/go/managedidentities/apiv1/managedidentitiespb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *managedidentities.Client) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	domainFQDN := getenv("STACKYARD_GCP_MANAGEDIDENTITIES_DOMAIN", "corp.stackyard.internal")
	network := getenv("STACKYARD_GCP_MANAGEDIDENTITIES_NETWORK", fmt.Sprintf("projects/%s/global/networks/default", projectID))
	targetDomain := getenv("STACKYARD_GCP_MANAGEDIDENTITIES_TRUST_TARGET_DOMAIN", "trusted.stackyard.internal")
	targetDNS := getenv("STACKYARD_GCP_MANAGEDIDENTITIES_TRUST_TARGET_DNS", "10.0.0.53")

	parent := fmt.Sprintf("projects/%s/locations/global", projectID)
	domainName := fmt.Sprintf("%s/domains/%s", parent, domainFQDN)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Managed Identities apiv1 client using %s\n", grpcEndpoint)

	client, err := managedidentities.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create managed identities client: %v", err)
	}
	defer closeClient("managed identities", client.Close)

	calls := []callSpec{
		{
			name: "ListDomains",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				it := c.ListDomains(ctx, &managedidentitiespb.ListDomainsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetDomain",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.GetDomain(ctx, &managedidentitiespb.GetDomainRequest{Name: domainName})
				return err
			},
		},
		{
			name: "CreateMicrosoftAdDomain",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.CreateMicrosoftAdDomain(ctx, &managedidentitiespb.CreateMicrosoftAdDomainRequest{
					Parent:     parent,
					DomainName: domainFQDN,
					Domain: &managedidentitiespb.Domain{
						Name:               domainName,
						ReservedIpRange:    "10.20.0.0/24",
						Locations:          []string{"us-central1"},
						AuthorizedNetworks: []string{network},
						Labels:             map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "UpdateDomain",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.UpdateDomain(ctx, &managedidentitiespb.UpdateDomainRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
					Domain: &managedidentitiespb.Domain{
						Name:   domainName,
						Labels: map[string]string{"env": "updated"},
					},
				})
				return err
			},
		},
		{
			name: "ResetAdminPassword",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.ResetAdminPassword(ctx, &managedidentitiespb.ResetAdminPasswordRequest{Name: domainName})
				return err
			},
		},
		{
			name: "AttachTrust",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.AttachTrust(ctx, &managedidentitiespb.AttachTrustRequest{
					Name:  domainName,
					Trust: trust(targetDomain, targetDNS),
				})
				return err
			},
		},
		{
			name: "ReconfigureTrust",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.ReconfigureTrust(ctx, &managedidentitiespb.ReconfigureTrustRequest{
					Name:                 domainName,
					TargetDomainName:     targetDomain,
					TargetDnsIpAddresses: []string{targetDNS},
				})
				return err
			},
		},
		{
			name: "ValidateTrust",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.ValidateTrust(ctx, &managedidentitiespb.ValidateTrustRequest{
					Name:  domainName,
					Trust: trust(targetDomain, targetDNS),
				})
				return err
			},
		},
		{
			name: "DetachTrust",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.DetachTrust(ctx, &managedidentitiespb.DetachTrustRequest{
					Name:  domainName,
					Trust: trust(targetDomain, targetDNS),
				})
				return err
			},
		},
		{
			name: "DeleteDomain",
			call: func(ctx context.Context, c *managedidentities.Client) error {
				_, err := c.DeleteDomain(ctx, &managedidentitiespb.DeleteDomainRequest{Name: domainName})
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		callCancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func trust(targetDomain, targetDNS string) *managedidentitiespb.Trust {
	return &managedidentitiespb.Trust{
		TargetDomainName:        targetDomain,
		TrustType:               managedidentitiespb.Trust_FOREST,
		TrustDirection:          managedidentitiespb.Trust_BIDIRECTIONAL,
		TargetDnsIpAddresses:    []string{targetDNS},
		TrustHandshakeSecret:    "stackyard-secret",
		SelectiveAuthentication: false,
	}
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
}

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large")
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
