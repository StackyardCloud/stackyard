package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	servicemanagement "cloud.google.com/go/servicemanagement/apiv1"
	"cloud.google.com/go/servicemanagement/apiv1/servicemanagementpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	serviceconfigpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"
	serviceName := getenv("STACKYARD_GCP_SERVICE_NAME", "stackyard.googleapis.com")

	fmt.Printf("Stackyard GCP Service Management apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "servicemanagement",
		},
	}

	client, err := servicemanagement.NewServiceManagerRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create servicemanagement client: %v", err)
	}
	defer closeClient("servicemanagement", client.Close)

	configID := "2026-01-01r0"
	rolloutID := "2026-01-01r0"
	iamResource := "services/" + serviceName

	calls := []callSpec{
		{
			name: "ListServices",
			call: func(ctx context.Context) error {
				it := client.ListServices(ctx, &servicemanagementpb.ListServicesRequest{
					PageSize: 1,
				})
				svc, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if svc != nil && strings.TrimSpace(svc.GetServiceName()) != "" {
					serviceName = svc.GetServiceName()
					iamResource = "services/" + serviceName
				}
				return nil
			},
		},
		{
			name: "GetService",
			call: func(ctx context.Context) error {
				_, err := client.GetService(ctx, &servicemanagementpb.GetServiceRequest{
					ServiceName: serviceName,
				})
				return err
			},
		},
		{
			name: "CreateService",
			call: func(ctx context.Context) error {
				_, err := client.CreateService(ctx, &servicemanagementpb.CreateServiceRequest{
					Service: &servicemanagementpb.ManagedService{
						ServiceName:       serviceName,
						ProducerProjectId: "stackyard-project",
					},
				})
				return err
			},
		},
		{
			name: "DeleteService",
			call: func(ctx context.Context) error {
				_, err := client.DeleteService(ctx, &servicemanagementpb.DeleteServiceRequest{
					ServiceName: serviceName,
				})
				return err
			},
		},
		{
			name: "UndeleteService",
			call: func(ctx context.Context) error {
				_, err := client.UndeleteService(ctx, &servicemanagementpb.UndeleteServiceRequest{
					ServiceName: serviceName,
				})
				return err
			},
		},
		{
			name: "ListServiceConfigs",
			call: func(ctx context.Context) error {
				it := client.ListServiceConfigs(ctx, &servicemanagementpb.ListServiceConfigsRequest{
					ServiceName: serviceName,
					PageSize:    1,
				})
				cfg, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if cfg != nil && strings.TrimSpace(cfg.GetId()) != "" {
					configID = cfg.GetId()
				}
				return nil
			},
		},
		{
			name: "GetServiceConfig",
			call: func(ctx context.Context) error {
				_, err := client.GetServiceConfig(ctx, &servicemanagementpb.GetServiceConfigRequest{
					ServiceName: serviceName,
					ConfigId:    configID,
					View:        servicemanagementpb.GetServiceConfigRequest_FULL,
				})
				return err
			},
		},
		{
			name: "CreateServiceConfig",
			call: func(ctx context.Context) error {
				_, err := client.CreateServiceConfig(ctx, &servicemanagementpb.CreateServiceConfigRequest{
					ServiceName: serviceName,
					ServiceConfig: &serviceconfigpb.Service{
						Name:  serviceName,
						Id:    "2026-01-03r0",
						Title: "Stackyard Service Config",
					},
				})
				return err
			},
		},
		{
			name: "SubmitConfigSource",
			call: func(ctx context.Context) error {
				_, err := client.SubmitConfigSource(ctx, &servicemanagementpb.SubmitConfigSourceRequest{
					ServiceName: serviceName,
					ConfigSource: &servicemanagementpb.ConfigSource{
						Id: "config-source-1",
						Files: []*servicemanagementpb.ConfigFile{
							{
								FilePath:     "service.yaml",
								FileContents: []byte("configVersion: 3\nname: " + serviceName + "\n"),
								FileType:     servicemanagementpb.ConfigFile_SERVICE_CONFIG_YAML,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListServiceRollouts",
			call: func(ctx context.Context) error {
				it := client.ListServiceRollouts(ctx, &servicemanagementpb.ListServiceRolloutsRequest{
					ServiceName: serviceName,
					PageSize:    1,
					Filter:      "status=SUCCESS",
				})
				rollout, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if rollout != nil && strings.TrimSpace(rollout.GetRolloutId()) != "" {
					rolloutID = rollout.GetRolloutId()
				}
				return nil
			},
		},
		{
			name: "GetServiceRollout",
			call: func(ctx context.Context) error {
				_, err := client.GetServiceRollout(ctx, &servicemanagementpb.GetServiceRolloutRequest{
					ServiceName: serviceName,
					RolloutId:   rolloutID,
				})
				return err
			},
		},
		{
			name: "CreateServiceRollout",
			call: func(ctx context.Context) error {
				_, err := client.CreateServiceRollout(ctx, &servicemanagementpb.CreateServiceRolloutRequest{
					ServiceName: serviceName,
					Rollout: &servicemanagementpb.Rollout{
						RolloutId: "2026-01-04r0",
						Strategy: &servicemanagementpb.Rollout_TrafficPercentStrategy_{
							TrafficPercentStrategy: &servicemanagementpb.Rollout_TrafficPercentStrategy{
								Percentages: map[string]float64{
									configID: 100,
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GenerateConfigReport",
			call: func(ctx context.Context) error {
				newConfig, err := anypb.New(&servicemanagementpb.ConfigRef{
					Name: fmt.Sprintf("services/%s/configs/%s", serviceName, configID),
				})
				if err != nil {
					return err
				}
				_, err = client.GenerateConfigReport(ctx, &servicemanagementpb.GenerateConfigReportRequest{
					NewConfig: newConfig,
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := client.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
					Resource: iamResource,
				})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context) error {
				_, err := client.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: iamResource,
					Policy: &iampb.Policy{
						Version: 1,
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/viewer",
								Members: []string{"user:tester@example.com"},
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
					Resource:    iamResource,
					Permissions: []string{"servicemanagement.services.get"},
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := client.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
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
