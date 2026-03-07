package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	certificatemanager "cloud.google.com/go/certificatemanager/apiv1"
	"cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *certificatemanager.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	location := getenv("STACKYARD_GCP_LOCATION", "projects/stackyard/locations/global")
	certificateID := getenv("STACKYARD_GCP_CERTIFICATE_ID", "team-certificate")
	certificateMapID := getenv("STACKYARD_GCP_CERTIFICATE_MAP_ID", "team-map")
	certificateMapEntryID := getenv("STACKYARD_GCP_CERTIFICATE_MAP_ENTRY_ID", "entry-1")
	dnsAuthorizationID := getenv("STACKYARD_GCP_DNS_AUTHORIZATION_ID", "team-dns-auth")
	trustConfigID := getenv("STACKYARD_GCP_TRUST_CONFIG_ID", "team-trust")
	issuanceConfigID := getenv("STACKYARD_GCP_ISSUANCE_CONFIG_ID", "team-issuance")

	certificateName := location + "/certificates/" + certificateID
	certificateMapName := location + "/certificateMaps/" + certificateMapID
	certificateMapEntryName := certificateMapName + "/certificateMapEntries/" + certificateMapEntryID
	dnsAuthorizationName := location + "/dnsAuthorizations/" + dnsAuthorizationID
	trustConfigName := location + "/trustConfigs/" + trustConfigID
	issuanceConfigName := location + "/certificateIssuanceConfigs/" + issuanceConfigID

	fmt.Printf("Stackyard GCP Certificate Manager apiv1 client using %s\n", apiEndpoint)

	client, err := certificatemanager.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create certificatemanager client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListCertificates",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				it := c.ListCertificates(ctx, &certificatemanagerpb.ListCertificatesRequest{
					Parent:   location,
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
			name: "GetCertificate",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.GetCertificate(ctx, &certificatemanagerpb.GetCertificateRequest{
					Name: certificateName,
				})
				return err
			},
		},
		{
			name: "ListCertificateMaps",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				it := c.ListCertificateMaps(ctx, &certificatemanagerpb.ListCertificateMapsRequest{
					Parent:   location,
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
			name: "GetCertificateMap",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.GetCertificateMap(ctx, &certificatemanagerpb.GetCertificateMapRequest{
					Name: certificateMapName,
				})
				return err
			},
		},
		{
			name: "ListCertificateMapEntries",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				it := c.ListCertificateMapEntries(ctx, &certificatemanagerpb.ListCertificateMapEntriesRequest{
					Parent:   certificateMapName,
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
			name: "GetCertificateMapEntry",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.GetCertificateMapEntry(ctx, &certificatemanagerpb.GetCertificateMapEntryRequest{
					Name: certificateMapEntryName,
				})
				return err
			},
		},
		{
			name: "ListDnsAuthorizations",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				it := c.ListDnsAuthorizations(ctx, &certificatemanagerpb.ListDnsAuthorizationsRequest{
					Parent:   location,
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
			name: "GetDnsAuthorization",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.GetDnsAuthorization(ctx, &certificatemanagerpb.GetDnsAuthorizationRequest{
					Name: dnsAuthorizationName,
				})
				return err
			},
		},
		{
			name: "ListTrustConfigs",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				it := c.ListTrustConfigs(ctx, &certificatemanagerpb.ListTrustConfigsRequest{
					Parent:   location,
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
			name: "GetTrustConfig",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.GetTrustConfig(ctx, &certificatemanagerpb.GetTrustConfigRequest{
					Name: trustConfigName,
				})
				return err
			},
		},
		{
			name: "ListCertificateIssuanceConfigs",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				it := c.ListCertificateIssuanceConfigs(ctx, &certificatemanagerpb.ListCertificateIssuanceConfigsRequest{
					Parent:   location,
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
			name: "GetCertificateIssuanceConfig",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.GetCertificateIssuanceConfig(ctx, &certificatemanagerpb.GetCertificateIssuanceConfigRequest{
					Name: issuanceConfigName,
				})
				return err
			},
		},
		{
			name: "CreateCertificate",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.CreateCertificate(ctx, &certificatemanagerpb.CreateCertificateRequest{
					Parent:        location,
					CertificateId: certificateID,
					Certificate: &certificatemanagerpb.Certificate{
						Type: &certificatemanagerpb.Certificate_SelfManaged{
							SelfManaged: &certificatemanagerpb.Certificate_SelfManagedCertificate{
								PemCertificate: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
								PemPrivateKey:  "-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "CreateCertificateMap",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.CreateCertificateMap(ctx, &certificatemanagerpb.CreateCertificateMapRequest{
					Parent:           location,
					CertificateMapId: certificateMapID,
					CertificateMap: &certificatemanagerpb.CertificateMap{
						Description: "Team certificate map",
					},
				})
				return err
			},
		},
		{
			name: "CreateDnsAuthorization",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.CreateDnsAuthorization(ctx, &certificatemanagerpb.CreateDnsAuthorizationRequest{
					Parent:             location,
					DnsAuthorizationId: dnsAuthorizationID,
					DnsAuthorization: &certificatemanagerpb.DnsAuthorization{
						Domain: "example.com",
					},
				})
				return err
			},
		},
		{
			name: "DeleteTrustConfig",
			call: func(ctx context.Context, c *certificatemanager.Client) error {
				_, err := c.DeleteTrustConfig(ctx, &certificatemanagerpb.DeleteTrustConfigRequest{
					Name: trustConfigName,
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
		fmt.Fprintf(os.Stderr, "warning: close certificatemanager client: %v\n", err)
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
