package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	domains "cloud.google.com/go/domains/apiv1beta1"
	"cloud.google.com/go/domains/apiv1beta1/domainspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/genproto/googleapis/type/postaladdress"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *domains.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	domainName := getenv("STACKYARD_GCP_DOMAINS_DOMAIN_NAME", "example.test")
	registrationID := getenv("STACKYARD_GCP_DOMAINS_REGISTRATION_ID", "example.test")
	transferAuthCode := getenv("STACKYARD_GCP_DOMAINS_TRANSFER_AUTH_CODE", "transfer-code-123")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	location := parent
	registrationName := parent + "/registrations/" + registrationID

	fmt.Printf("Stackyard GCP Domains apiv1beta1 SDK client using %s\n", apiEndpoint)

	client, err := domains.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create domains client: %v", err)
	}
	defer closeClient("domains", client.Close)

	calls := []callSpec{
		{
			name: "SearchDomains",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.SearchDomains(ctx, &domainspb.SearchDomainsRequest{
					Query:    domainName,
					Location: location,
				})
				return err
			},
		},
		{
			name: "RetrieveRegisterParameters",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.RetrieveRegisterParameters(ctx, &domainspb.RetrieveRegisterParametersRequest{
					DomainName: domainName,
					Location:   location,
				})
				return err
			},
		},
		{
			name: "RegisterDomain",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.RegisterDomain(ctx, &domainspb.RegisterDomainRequest{
					Parent:       parent,
					Registration: sampleRegistration(domainName, "domains+register@example.test"),
					YearlyPrice:  sampleYearlyPrice(),
					ValidateOnly: true,
				})
				return err
			},
		},
		{
			name: "RetrieveTransferParameters",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.RetrieveTransferParameters(ctx, &domainspb.RetrieveTransferParametersRequest{
					DomainName: domainName,
					Location:   location,
				})
				return err
			},
		},
		{
			name: "TransferDomain",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.TransferDomain(ctx, &domainspb.TransferDomainRequest{
					Parent:       parent,
					Registration: sampleRegistration(domainName, "domains+transfer@example.test"),
					YearlyPrice:  sampleYearlyPrice(),
					AuthorizationCode: &domainspb.AuthorizationCode{
						Code: transferAuthCode,
					},
					ValidateOnly: true,
				})
				return err
			},
		},
		{
			name: "ListRegistrations",
			call: func(ctx context.Context, c *domains.Client) error {
				it := c.ListRegistrations(ctx, &domainspb.ListRegistrationsRequest{
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
			name: "GetRegistration",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.GetRegistration(ctx, &domainspb.GetRegistrationRequest{
					Name: registrationName,
				})
				return err
			},
		},
		{
			name: "UpdateRegistration",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.UpdateRegistration(ctx, &domainspb.UpdateRegistrationRequest{
					Registration: &domainspb.Registration{
						Name: registrationName,
						Labels: map[string]string{
							"owner": "stackyard",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				})
				return err
			},
		},
		{
			name: "ConfigureManagementSettings",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.ConfigureManagementSettings(ctx, &domainspb.ConfigureManagementSettingsRequest{
					Registration: registrationName,
					ManagementSettings: &domainspb.ManagementSettings{
						TransferLockState: domainspb.TransferLockState_LOCKED,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"transfer_lock_state"},
					},
				})
				return err
			},
		},
		{
			name: "ConfigureDnsSettings",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.ConfigureDnsSettings(ctx, &domainspb.ConfigureDnsSettingsRequest{
					Registration: registrationName,
					DnsSettings: &domainspb.DnsSettings{
						DnsProvider: &domainspb.DnsSettings_CustomDns_{
							CustomDns: &domainspb.DnsSettings_CustomDns{
								NameServers: []string{
									"ns1.stackyard.test",
									"ns2.stackyard.test",
								},
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"custom_dns"},
					},
					ValidateOnly: true,
				})
				return err
			},
		},
		{
			name: "ConfigureContactSettings",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.ConfigureContactSettings(ctx, &domainspb.ConfigureContactSettingsRequest{
					Registration: registrationName,
					ContactSettings: &domainspb.ContactSettings{
						Privacy:           domainspb.ContactPrivacy_PRIVATE_CONTACT_DATA,
						RegistrantContact: sampleContact("domains+registrant@example.test"),
						AdminContact:      sampleContact("domains+admin@example.test"),
						TechnicalContact:  sampleContact("domains+tech@example.test"),
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"privacy", "registrant_contact"},
					},
					ValidateOnly: true,
				})
				return err
			},
		},
		{
			name: "ExportRegistration",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.ExportRegistration(ctx, &domainspb.ExportRegistrationRequest{
					Name: registrationName,
				})
				return err
			},
		},
		{
			name: "DeleteRegistration",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.DeleteRegistration(ctx, &domainspb.DeleteRegistrationRequest{
					Name: registrationName,
				})
				return err
			},
		},
		{
			name: "RetrieveAuthorizationCode",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.RetrieveAuthorizationCode(ctx, &domainspb.RetrieveAuthorizationCodeRequest{
					Registration: registrationName,
				})
				return err
			},
		},
		{
			name: "ResetAuthorizationCode",
			call: func(ctx context.Context, c *domains.Client) error {
				_, err := c.ResetAuthorizationCode(ctx, &domainspb.ResetAuthorizationCodeRequest{
					Registration: registrationName,
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

func sampleRegistration(domainName, email string) *domainspb.Registration {
	return &domainspb.Registration{
		DomainName: domainName,
		Labels: map[string]string{
			"env": "local",
		},
		ManagementSettings: &domainspb.ManagementSettings{
			TransferLockState: domainspb.TransferLockState_UNLOCKED,
		},
		DnsSettings: &domainspb.DnsSettings{
			DnsProvider: &domainspb.DnsSettings_CustomDns_{
				CustomDns: &domainspb.DnsSettings_CustomDns{
					NameServers: []string{
						"ns1.stackyard.test",
						"ns2.stackyard.test",
					},
				},
			},
		},
		ContactSettings: &domainspb.ContactSettings{
			Privacy:           domainspb.ContactPrivacy_PRIVATE_CONTACT_DATA,
			RegistrantContact: sampleContact(email),
			AdminContact:      sampleContact(email),
			TechnicalContact:  sampleContact(email),
		},
	}
}

func sampleContact(email string) *domainspb.ContactSettings_Contact {
	return &domainspb.ContactSettings_Contact{
		Email:       email,
		PhoneNumber: "+1-800-555-0100",
		PostalAddress: &postaladdress.PostalAddress{
			RegionCode:         "US",
			AdministrativeArea: "NY",
			Locality:           "New York",
			PostalCode:         "10001",
			AddressLines: []string{
				"123 Stackyard Street",
			},
		},
	}
}

func sampleYearlyPrice() *money.Money {
	return &money.Money{
		CurrencyCode: "USD",
		Units:        12,
	}
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
