package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	securitycentermanagement "cloud.google.com/go/securitycentermanagement/apiv1"
	"cloud.google.com/go/securitycentermanagement/apiv1/securitycentermanagementpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	exprpb "google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
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
	shaModuleID := getenv("STACKYARD_GCP_SHA_MODULE_ID", "sha-module-1")
	etdModuleID := getenv("STACKYARD_GCP_ETD_MODULE_ID", "etd-module-1")
	serviceID := getenv("STACKYARD_GCP_SCCM_SERVICE_ID", "security-health-analytics")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	shaName := fmt.Sprintf("%s/securityHealthAnalyticsCustomModules/%s", parent, shaModuleID)
	effectiveSHAName := fmt.Sprintf("%s/effectiveSecurityHealthAnalyticsCustomModules/effective-sha-module-1", parent)
	etdName := fmt.Sprintf("%s/eventThreatDetectionCustomModules/%s", parent, etdModuleID)
	effectiveETDName := fmt.Sprintf("%s/effectiveEventThreatDetectionCustomModules/effective-etd-module-1", parent)
	serviceName := fmt.Sprintf("%s/securityCenterServices/%s", parent, serviceID)

	fmt.Printf("Stackyard GCP Security Command Center Management apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "securitycentermanagement",
		},
	}

	client, err := securitycentermanagement.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create securitycentermanagement client: %v", err)
	}
	defer closeClient(client.Close)

	resourceData, _ := structpb.NewStruct(map[string]any{
		"name":      "instances/i-1",
		"assetType": "compute.googleapis.com/Instance",
	})

	etdConfig, _ := structpb.NewStruct(map[string]any{
		"allowedIp": "10.0.0.1",
		"mode":      "BLOCK",
	})

	calls := []callSpec{
		{
			name: "ListLocations",
			call: func(ctx context.Context) error {
				it := client.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     fmt.Sprintf("projects/%s", projectID),
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
				_, err := client.GetLocation(ctx, &locationpb.GetLocationRequest{Name: parent})
				return err
			},
		},
		{
			name: "ListSecurityCenterServices",
			call: func(ctx context.Context) error {
				it := client.ListSecurityCenterServices(ctx, &securitycentermanagementpb.ListSecurityCenterServicesRequest{
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
			name: "GetSecurityCenterService",
			call: func(ctx context.Context) error {
				_, err := client.GetSecurityCenterService(ctx, &securitycentermanagementpb.GetSecurityCenterServiceRequest{
					Name:                    serviceName,
					ShowEligibleModulesOnly: true,
				})
				return err
			},
		},
		{
			name: "UpdateSecurityCenterService",
			call: func(ctx context.Context) error {
				_, err := client.UpdateSecurityCenterService(ctx, &securitycentermanagementpb.UpdateSecurityCenterServiceRequest{
					SecurityCenterService: &securitycentermanagementpb.SecurityCenterService{
						Name:                    serviceName,
						IntendedEnablementState: securitycentermanagementpb.SecurityCenterService_ENABLED,
						Modules: map[string]*securitycentermanagementpb.SecurityCenterService_ModuleSettings{
							"default-module": {
								IntendedEnablementState: securitycentermanagementpb.SecurityCenterService_ENABLED,
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"intended_enablement_state", "modules"}},
				})
				return err
			},
		},
		{
			name: "ListSecurityHealthAnalyticsCustomModules",
			call: func(ctx context.Context) error {
				it := client.ListSecurityHealthAnalyticsCustomModules(ctx, &securitycentermanagementpb.ListSecurityHealthAnalyticsCustomModulesRequest{
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
			name: "ListDescendantSecurityHealthAnalyticsCustomModules",
			call: func(ctx context.Context) error {
				it := client.ListDescendantSecurityHealthAnalyticsCustomModules(ctx, &securitycentermanagementpb.ListDescendantSecurityHealthAnalyticsCustomModulesRequest{
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
			name: "GetSecurityHealthAnalyticsCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.GetSecurityHealthAnalyticsCustomModule(ctx, &securitycentermanagementpb.GetSecurityHealthAnalyticsCustomModuleRequest{Name: shaName})
				return err
			},
		},
		{
			name: "CreateSecurityHealthAnalyticsCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.CreateSecurityHealthAnalyticsCustomModule(ctx, &securitycentermanagementpb.CreateSecurityHealthAnalyticsCustomModuleRequest{
					Parent: parent,
					SecurityHealthAnalyticsCustomModule: &securitycentermanagementpb.SecurityHealthAnalyticsCustomModule{
						DisplayName: "stackyard_sha_custom",
						CustomConfig: &securitycentermanagementpb.CustomConfig{
							Predicate: &exprpb.Expr{Expression: "true"},
						},
					},
				})
				return err
			},
		},
		{
			name: "UpdateSecurityHealthAnalyticsCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.UpdateSecurityHealthAnalyticsCustomModule(ctx, &securitycentermanagementpb.UpdateSecurityHealthAnalyticsCustomModuleRequest{
					SecurityHealthAnalyticsCustomModule: &securitycentermanagementpb.SecurityHealthAnalyticsCustomModule{
						Name:            shaName,
						EnablementState: securitycentermanagementpb.SecurityHealthAnalyticsCustomModule_DISABLED,
						CustomConfig: &securitycentermanagementpb.CustomConfig{
							Predicate: &exprpb.Expr{Expression: "false"},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"custom_config", "enablement_state"}},
				})
				return err
			},
		},
		{
			name: "DeleteSecurityHealthAnalyticsCustomModule",
			call: func(ctx context.Context) error {
				return client.DeleteSecurityHealthAnalyticsCustomModule(ctx, &securitycentermanagementpb.DeleteSecurityHealthAnalyticsCustomModuleRequest{Name: shaName})
			},
		},
		{
			name: "ListEffectiveSecurityHealthAnalyticsCustomModules",
			call: func(ctx context.Context) error {
				it := client.ListEffectiveSecurityHealthAnalyticsCustomModules(ctx, &securitycentermanagementpb.ListEffectiveSecurityHealthAnalyticsCustomModulesRequest{
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
			name: "GetEffectiveSecurityHealthAnalyticsCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.GetEffectiveSecurityHealthAnalyticsCustomModule(ctx, &securitycentermanagementpb.GetEffectiveSecurityHealthAnalyticsCustomModuleRequest{Name: effectiveSHAName})
				return err
			},
		},
		{
			name: "SimulateSecurityHealthAnalyticsCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.SimulateSecurityHealthAnalyticsCustomModule(ctx, &securitycentermanagementpb.SimulateSecurityHealthAnalyticsCustomModuleRequest{
					Parent: parent,
					CustomConfig: &securitycentermanagementpb.CustomConfig{
						Predicate: &exprpb.Expr{Expression: "true"},
					},
					Resource: &securitycentermanagementpb.SimulateSecurityHealthAnalyticsCustomModuleRequest_SimulatedResource{
						ResourceType: "compute.googleapis.com/Instance",
						ResourceData: resourceData,
					},
				})
				return err
			},
		},
		{
			name: "ListEventThreatDetectionCustomModules",
			call: func(ctx context.Context) error {
				it := client.ListEventThreatDetectionCustomModules(ctx, &securitycentermanagementpb.ListEventThreatDetectionCustomModulesRequest{
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
			name: "ListDescendantEventThreatDetectionCustomModules",
			call: func(ctx context.Context) error {
				it := client.ListDescendantEventThreatDetectionCustomModules(ctx, &securitycentermanagementpb.ListDescendantEventThreatDetectionCustomModulesRequest{
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
			name: "GetEventThreatDetectionCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.GetEventThreatDetectionCustomModule(ctx, &securitycentermanagementpb.GetEventThreatDetectionCustomModuleRequest{Name: etdName})
				return err
			},
		},
		{
			name: "CreateEventThreatDetectionCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.CreateEventThreatDetectionCustomModule(ctx, &securitycentermanagementpb.CreateEventThreatDetectionCustomModuleRequest{
					Parent: parent,
					EventThreatDetectionCustomModule: &securitycentermanagementpb.EventThreatDetectionCustomModule{
						Type:   "CONFIGURABLE_BAD_IP",
						Config: etdConfig,
					},
				})
				return err
			},
		},
		{
			name: "UpdateEventThreatDetectionCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.UpdateEventThreatDetectionCustomModule(ctx, &securitycentermanagementpb.UpdateEventThreatDetectionCustomModuleRequest{
					EventThreatDetectionCustomModule: &securitycentermanagementpb.EventThreatDetectionCustomModule{
						Name:            etdName,
						Type:            "CONFIGURABLE_BAD_IP",
						Config:          etdConfig,
						EnablementState: securitycentermanagementpb.EventThreatDetectionCustomModule_DISABLED,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"config", "enablement_state"}},
				})
				return err
			},
		},
		{
			name: "DeleteEventThreatDetectionCustomModule",
			call: func(ctx context.Context) error {
				return client.DeleteEventThreatDetectionCustomModule(ctx, &securitycentermanagementpb.DeleteEventThreatDetectionCustomModuleRequest{Name: etdName})
			},
		},
		{
			name: "ListEffectiveEventThreatDetectionCustomModules",
			call: func(ctx context.Context) error {
				it := client.ListEffectiveEventThreatDetectionCustomModules(ctx, &securitycentermanagementpb.ListEffectiveEventThreatDetectionCustomModulesRequest{
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
			name: "GetEffectiveEventThreatDetectionCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.GetEffectiveEventThreatDetectionCustomModule(ctx, &securitycentermanagementpb.GetEffectiveEventThreatDetectionCustomModuleRequest{Name: effectiveETDName})
				return err
			},
		},
		{
			name: "ValidateEventThreatDetectionCustomModule",
			call: func(ctx context.Context) error {
				_, err := client.ValidateEventThreatDetectionCustomModule(ctx, &securitycentermanagementpb.ValidateEventThreatDetectionCustomModuleRequest{
					Parent:  parent,
					RawText: "rule allow { when true }",
					Type:    "CONFIGURABLE_BAD_IP",
				})
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close securitycentermanagement client: %v\n", err)
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
