package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	servicedirectory "cloud.google.com/go/servicedirectory/apiv1"
	"cloud.google.com/go/servicedirectory/apiv1/servicedirectorypb"
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

	project := getenv("STACKYARD_GCP_PROJECT", "stackyard")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	namespaceID := getenv("STACKYARD_GCP_NAMESPACE_ID", "ns-1")
	serviceID := getenv("STACKYARD_GCP_SERVICE_ID", "svc-1")
	endpointID := getenv("STACKYARD_GCP_ENDPOINT_ID", "ep-1")

	locationParent := fmt.Sprintf("projects/%s", project)
	locationName := fmt.Sprintf("projects/%s/locations/%s", project, location)
	namespaceParent := locationName
	namespaceName := fmt.Sprintf("%s/namespaces/%s", namespaceParent, namespaceID)
	serviceParent := namespaceName
	serviceName := fmt.Sprintf("%s/services/%s", serviceParent, serviceID)
	endpointParent := serviceName
	serviceEndpointName := fmt.Sprintf("%s/endpoints/%s", endpointParent, endpointID)

	fmt.Printf("Stackyard GCP Service Directory apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "servicedirectory",
		},
	}

	clientOpts := []option.ClientOption{
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	}

	registrationClient, err := servicedirectory.NewRegistrationRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create servicedirectory registration client: %v", err)
	}
	defer closeClient("registration", registrationClient.Close)

	lookupClient, err := servicedirectory.NewLookupRESTClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create servicedirectory lookup client: %v", err)
	}
	defer closeClient("lookup", lookupClient.Close)

	calls := []callSpec{
		{
			name: "Registration.ListLocations",
			call: func(ctx context.Context) error {
				it := registrationClient.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     locationParent,
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
			name: "Registration.GetLocation",
			call: func(ctx context.Context) error {
				_, err := registrationClient.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "Registration.CreateNamespace",
			call: func(ctx context.Context) error {
				_, err := registrationClient.CreateNamespace(ctx, &servicedirectorypb.CreateNamespaceRequest{
					Parent:      namespaceParent,
					NamespaceId: namespaceID,
					Namespace: &servicedirectorypb.Namespace{
						Labels: map[string]string{"env": "stackyard"},
					},
				})
				return err
			},
		},
		{
			name: "Registration.ListNamespaces",
			call: func(ctx context.Context) error {
				it := registrationClient.ListNamespaces(ctx, &servicedirectorypb.ListNamespacesRequest{
					Parent:   namespaceParent,
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
			name: "Registration.GetNamespace",
			call: func(ctx context.Context) error {
				_, err := registrationClient.GetNamespace(ctx, &servicedirectorypb.GetNamespaceRequest{Name: namespaceName})
				return err
			},
		},
		{
			name: "Registration.UpdateNamespace",
			call: func(ctx context.Context) error {
				_, err := registrationClient.UpdateNamespace(ctx, &servicedirectorypb.UpdateNamespaceRequest{
					Namespace: &servicedirectorypb.Namespace{
						Name:   namespaceName,
						Labels: map[string]string{"team": "platform"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "Registration.CreateService",
			call: func(ctx context.Context) error {
				_, err := registrationClient.CreateService(ctx, &servicedirectorypb.CreateServiceRequest{
					Parent:    serviceParent,
					ServiceId: serviceID,
					Service: &servicedirectorypb.Service{
						Annotations: map[string]string{"owner": "api-team"},
					},
				})
				return err
			},
		},
		{
			name: "Registration.ListServices",
			call: func(ctx context.Context) error {
				it := registrationClient.ListServices(ctx, &servicedirectorypb.ListServicesRequest{
					Parent:   serviceParent,
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
			name: "Registration.GetService",
			call: func(ctx context.Context) error {
				_, err := registrationClient.GetService(ctx, &servicedirectorypb.GetServiceRequest{Name: serviceName})
				return err
			},
		},
		{
			name: "Registration.UpdateService",
			call: func(ctx context.Context) error {
				_, err := registrationClient.UpdateService(ctx, &servicedirectorypb.UpdateServiceRequest{
					Service: &servicedirectorypb.Service{
						Name:        serviceName,
						Annotations: map[string]string{"tier": "gold"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"annotations"}},
				})
				return err
			},
		},
		{
			name: "Registration.CreateEndpoint",
			call: func(ctx context.Context) error {
				_, err := registrationClient.CreateEndpoint(ctx, &servicedirectorypb.CreateEndpointRequest{
					Parent:     endpointParent,
					EndpointId: endpointID,
					Endpoint: &servicedirectorypb.Endpoint{
						Address:     "10.10.0.9",
						Port:        8088,
						Annotations: map[string]string{"backend": "primary"},
					},
				})
				return err
			},
		},
		{
			name: "Registration.ListEndpoints",
			call: func(ctx context.Context) error {
				it := registrationClient.ListEndpoints(ctx, &servicedirectorypb.ListEndpointsRequest{
					Parent:   endpointParent,
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
			name: "Registration.GetEndpoint",
			call: func(ctx context.Context) error {
				_, err := registrationClient.GetEndpoint(ctx, &servicedirectorypb.GetEndpointRequest{Name: serviceEndpointName})
				return err
			},
		},
		{
			name: "Registration.UpdateEndpoint",
			call: func(ctx context.Context) error {
				_, err := registrationClient.UpdateEndpoint(ctx, &servicedirectorypb.UpdateEndpointRequest{
					Endpoint: &servicedirectorypb.Endpoint{
						Name:    serviceEndpointName,
						Address: "10.10.0.15",
						Port:    9090,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"address", "port"}},
				})
				return err
			},
		},
		{
			name: "Lookup.ResolveService",
			call: func(ctx context.Context) error {
				_, err := lookupClient.ResolveService(ctx, &servicedirectorypb.ResolveServiceRequest{
					Name:         serviceName,
					MaxEndpoints: 1,
				})
				return err
			},
		},
		{
			name: "Lookup.ListLocations",
			call: func(ctx context.Context) error {
				it := lookupClient.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     locationParent,
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
			name: "Registration.GetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := registrationClient.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: serviceName})
				return err
			},
		},
		{
			name: "Registration.SetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := registrationClient.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: serviceName,
					Policy: &iampb.Policy{
						Version: 1,
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/servicedirectory.viewer",
								Members: []string{"user:stackyard@example.com"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "Registration.TestIamPermissions",
			call: func(ctx context.Context) error {
				_, err := registrationClient.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    serviceName,
					Permissions: []string{"servicedirectory.services.get"},
				})
				return err
			},
		},
		{
			name: "Registration.DeleteEndpoint",
			call: func(ctx context.Context) error {
				return registrationClient.DeleteEndpoint(ctx, &servicedirectorypb.DeleteEndpointRequest{Name: serviceEndpointName})
			},
		},
		{
			name: "Registration.DeleteService",
			call: func(ctx context.Context) error {
				return registrationClient.DeleteService(ctx, &servicedirectorypb.DeleteServiceRequest{Name: serviceName})
			},
		},
		{
			name: "Registration.DeleteNamespace",
			call: func(ctx context.Context) error {
				return registrationClient.DeleteNamespace(ctx, &servicedirectorypb.DeleteNamespaceRequest{Name: namespaceName})
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

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", label, err)
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
