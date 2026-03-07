package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	privateca "cloud.google.com/go/security/privateca/apiv1"
	"cloud.google.com/go/security/privateca/apiv1/privatecapb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
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
	caPoolID := getenv("STACKYARD_GCP_CA_POOL_ID", "pool-1")
	caID := getenv("STACKYARD_GCP_CA_ID", "ca-1")
	certificateID := getenv("STACKYARD_GCP_CERTIFICATE_ID", "cert-1")
	templateID := getenv("STACKYARD_GCP_CERTIFICATE_TEMPLATE_ID", "template-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	caPoolName := fmt.Sprintf("%s/caPools/%s", locationName, caPoolID)
	certificateAuthorityName := fmt.Sprintf("%s/certificateAuthorities/%s", caPoolName, caID)
	disabledCertificateAuthorityName := fmt.Sprintf("%s/certificateAuthorities/ca-disabled", caPoolName)
	awaitingCertificateAuthorityName := fmt.Sprintf("%s/certificateAuthorities/ca-awaiting", caPoolName)
	certificateName := fmt.Sprintf("%s/certificates/%s", caPoolName, certificateID)
	certificateTemplateName := fmt.Sprintf("%s/certificateTemplates/%s", locationName, templateID)
	operationName := fmt.Sprintf("%s/operations/operation-1", locationName)

	fmt.Printf("Stackyard GCP Certificate Authority security/privateca/apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "security-privateca",
		},
	}

	client, err := privateca.NewCertificateAuthorityRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create security privateca client: %v", err)
	}
	defer closeClient("security-privateca", client.Close)

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
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
			call: func(ctx context.Context) error {
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListCaPools",
			call: func(ctx context.Context) error {
				it := client.ListCaPools(ctx, &privatecapb.ListCaPoolsRequest{
					Parent:   locationName,
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
			name: "GetCaPool",
			call: func(ctx context.Context) error {
				_, err := client.GetCaPool(ctx, &privatecapb.GetCaPoolRequest{Name: caPoolName})
				return err
			},
		},
		{
			name: "CreateCaPool",
			call: func(ctx context.Context) error {
				op, err := client.CreateCaPool(ctx, &privatecapb.CreateCaPoolRequest{
					Parent:   locationName,
					CaPoolId: caPoolID,
					CaPool: &privatecapb.CaPool{
						Tier: privatecapb.CaPool_ENTERPRISE,
					},
				})
				if err == nil && op != nil {
					logf("CreateCaPool operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "UpdateCaPool",
			call: func(ctx context.Context) error {
				op, err := client.UpdateCaPool(ctx, &privatecapb.UpdateCaPoolRequest{
					CaPool: &privatecapb.CaPool{
						Name: caPoolName,
						Labels: map[string]string{
							"team": "platform",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				})
				if err == nil && op != nil {
					logf("UpdateCaPool operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "FetchCaCerts",
			call: func(ctx context.Context) error {
				_, err := client.FetchCaCerts(ctx, &privatecapb.FetchCaCertsRequest{CaPool: caPoolName})
				return err
			},
		},
		{
			name: "ListCertificateAuthorities",
			call: func(ctx context.Context) error {
				it := client.ListCertificateAuthorities(ctx, &privatecapb.ListCertificateAuthoritiesRequest{
					Parent:   caPoolName,
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
			name: "GetCertificateAuthority",
			call: func(ctx context.Context) error {
				_, err := client.GetCertificateAuthority(ctx, &privatecapb.GetCertificateAuthorityRequest{Name: certificateAuthorityName})
				return err
			},
		},
		{
			name: "CreateCertificateAuthority",
			call: func(ctx context.Context) error {
				op, err := client.CreateCertificateAuthority(ctx, &privatecapb.CreateCertificateAuthorityRequest{
					Parent:                 caPoolName,
					CertificateAuthorityId: caID,
					CertificateAuthority: &privatecapb.CertificateAuthority{
						Type: privatecapb.CertificateAuthority_SELF_SIGNED,
						Config: &privatecapb.CertificateConfig{
							SubjectConfig: &privatecapb.CertificateConfig_SubjectConfig{
								Subject: &privatecapb.Subject{
									CommonName: "Stackyard Root CA",
								},
							},
						},
						Lifetime: durationpb.New(365 * 24 * time.Hour),
						KeySpec: &privatecapb.CertificateAuthority_KeyVersionSpec{
							KeyVersion: &privatecapb.CertificateAuthority_KeyVersionSpec_Algorithm{
								Algorithm: privatecapb.CertificateAuthority_RSA_PKCS1_4096_SHA256,
							},
						},
					},
				})
				if err == nil && op != nil {
					logf("CreateCertificateAuthority operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "FetchCertificateAuthorityCsr",
			call: func(ctx context.Context) error {
				_, err := client.FetchCertificateAuthorityCsr(ctx, &privatecapb.FetchCertificateAuthorityCsrRequest{
					Name: awaitingCertificateAuthorityName,
				})
				return err
			},
		},
		{
			name: "DisableCertificateAuthority",
			call: func(ctx context.Context) error {
				op, err := client.DisableCertificateAuthority(ctx, &privatecapb.DisableCertificateAuthorityRequest{
					Name: certificateAuthorityName,
				})
				if err == nil && op != nil {
					logf("DisableCertificateAuthority operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "EnableCertificateAuthority",
			call: func(ctx context.Context) error {
				op, err := client.EnableCertificateAuthority(ctx, &privatecapb.EnableCertificateAuthorityRequest{
					Name: disabledCertificateAuthorityName,
				})
				if err == nil && op != nil {
					logf("EnableCertificateAuthority operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "ActivateCertificateAuthority",
			call: func(ctx context.Context) error {
				op, err := client.ActivateCertificateAuthority(ctx, &privatecapb.ActivateCertificateAuthorityRequest{
					Name: awaitingCertificateAuthorityName,
				})
				if err == nil && op != nil {
					logf("ActivateCertificateAuthority operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "ListCertificates",
			call: func(ctx context.Context) error {
				it := client.ListCertificates(ctx, &privatecapb.ListCertificatesRequest{
					Parent:   caPoolName,
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
			call: func(ctx context.Context) error {
				_, err := client.GetCertificate(ctx, &privatecapb.GetCertificateRequest{Name: certificateName})
				return err
			},
		},
		{
			name: "CreateCertificate",
			call: func(ctx context.Context) error {
				_, err := client.CreateCertificate(ctx, &privatecapb.CreateCertificateRequest{
					Parent:        caPoolName,
					CertificateId: certificateID,
					Certificate: &privatecapb.Certificate{
						Lifetime: durationpb.New(24 * time.Hour),
						CertificateConfig: &privatecapb.Certificate_PemCsr{
							PemCsr: "-----BEGIN CERTIFICATE REQUEST-----\nSTACKYARD\n-----END CERTIFICATE REQUEST-----",
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateCertificate",
			call: func(ctx context.Context) error {
				_, err := client.UpdateCertificate(ctx, &privatecapb.UpdateCertificateRequest{
					Certificate: &privatecapb.Certificate{
						Name: certificateName,
						Labels: map[string]string{
							"env": "staged",
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
			name: "RevokeCertificate",
			call: func(ctx context.Context) error {
				_, err := client.RevokeCertificate(ctx, &privatecapb.RevokeCertificateRequest{
					Name:   certificateName,
					Reason: privatecapb.RevocationReason_KEY_COMPROMISE,
				})
				return err
			},
		},
		{
			name: "ListCertificateRevocationLists",
			call: func(ctx context.Context) error {
				it := client.ListCertificateRevocationLists(ctx, &privatecapb.ListCertificateRevocationListsRequest{
					Parent:   certificateAuthorityName,
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
			name: "GetCertificateRevocationList",
			call: func(ctx context.Context) error {
				_, err := client.GetCertificateRevocationList(ctx, &privatecapb.GetCertificateRevocationListRequest{
					Name: certificateAuthorityName + "/certificateRevocationLists/crl-1",
				})
				return err
			},
		},
		{
			name: "UpdateCertificateRevocationList",
			call: func(ctx context.Context) error {
				op, err := client.UpdateCertificateRevocationList(ctx, &privatecapb.UpdateCertificateRevocationListRequest{
					CertificateRevocationList: &privatecapb.CertificateRevocationList{
						Name: certificateAuthorityName + "/certificateRevocationLists/crl-1",
						Labels: map[string]string{
							"env": "prod",
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				})
				if err == nil && op != nil {
					logf("UpdateCertificateRevocationList operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "ListCertificateTemplates",
			call: func(ctx context.Context) error {
				it := client.ListCertificateTemplates(ctx, &privatecapb.ListCertificateTemplatesRequest{
					Parent:   locationName,
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
			name: "GetCertificateTemplate",
			call: func(ctx context.Context) error {
				_, err := client.GetCertificateTemplate(ctx, &privatecapb.GetCertificateTemplateRequest{
					Name: certificateTemplateName,
				})
				return err
			},
		},
		{
			name: "CreateCertificateTemplate",
			call: func(ctx context.Context) error {
				op, err := client.CreateCertificateTemplate(ctx, &privatecapb.CreateCertificateTemplateRequest{
					Parent:                locationName,
					CertificateTemplateId: templateID,
					CertificateTemplate: &privatecapb.CertificateTemplate{
						Description:     "Stackyard template",
						MaximumLifetime: durationpb.New(30 * 24 * time.Hour),
					},
				})
				if err == nil && op != nil {
					logf("CreateCertificateTemplate operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "UpdateCertificateTemplate",
			call: func(ctx context.Context) error {
				op, err := client.UpdateCertificateTemplate(ctx, &privatecapb.UpdateCertificateTemplateRequest{
					CertificateTemplate: &privatecapb.CertificateTemplate{
						Name:        certificateTemplateName,
						Description: "Stackyard template updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"description"},
					},
				})
				if err == nil && op != nil {
					logf("UpdateCertificateTemplate operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "DeleteCertificateTemplate",
			call: func(ctx context.Context) error {
				op, err := client.DeleteCertificateTemplate(ctx, &privatecapb.DeleteCertificateTemplateRequest{
					Name: certificateTemplateName,
				})
				if err == nil && op != nil {
					logf("DeleteCertificateTemplate operation=%s", op.Name())
				}
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := client.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: caPoolName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := client.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: caPoolName,
					Policy: &iampb.Policy{
						Version: 1,
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/privateca.admin",
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
					Resource:    caPoolName,
					Permissions: []string{"privateca.caPools.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				_, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := client.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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
			name: "CancelOperation",
			call: func(ctx context.Context) error {
				return client.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context) error {
				return client.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
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
