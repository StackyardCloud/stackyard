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
	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	"cloud.google.com/go/securitycenter/apiv1/securitycenterpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context, *securitycenter.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	orgID := getenv("STACKYARD_GCP_ORGANIZATION_ID", "123456")
	sourceID := getenv("STACKYARD_GCP_SOURCE_ID", "source-1")
	findingID := getenv("STACKYARD_GCP_FINDING_ID", "finding-1")
	muteConfigID := getenv("STACKYARD_GCP_MUTE_CONFIG_ID", "mute-config-1")
	notificationConfigID := getenv("STACKYARD_GCP_NOTIFICATION_CONFIG_ID", "notify-1")
	bigQueryExportID := getenv("STACKYARD_GCP_BIGQUERY_EXPORT_ID", "export-1")
	operationID := getenv("STACKYARD_GCP_OPERATION_ID", "op-1")

	orgName := fmt.Sprintf("organizations/%s", orgID)
	sourceName := orgName + "/sources/" + sourceID
	findingName := sourceName + "/findings/" + findingID
	muteConfigName := orgName + "/muteConfigs/" + muteConfigID
	notificationConfigName := orgName + "/notificationConfigs/" + notificationConfigID
	bigQueryExportName := orgName + "/bigQueryExports/" + bigQueryExportID
	orgSettingsName := orgName + "/organizationSettings"
	operationName := orgName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Security Command Center apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "securitycenter",
		},
	}

	client, err := securitycenter.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create securitycenter client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListSources",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				it := c.ListSources(ctx, &securitycenterpb.ListSourcesRequest{
					Parent:   orgName,
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
			name: "GetSource",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.GetSource(ctx, &securitycenterpb.GetSourceRequest{Name: sourceName})
				return err
			},
		},
		{
			name: "CreateSource",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.CreateSource(ctx, &securitycenterpb.CreateSourceRequest{
					Parent: orgName,
					Source: &securitycenterpb.Source{
						DisplayName: "Stackyard Source",
						Description: "staged source",
					},
				})
				return err
			},
		},
		{
			name: "UpdateSource",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.UpdateSource(ctx, &securitycenterpb.UpdateSourceRequest{
					Source: &securitycenterpb.Source{
						Name:        sourceName,
						DisplayName: "Stackyard Source Updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "ListFindings",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				it := c.ListFindings(ctx, &securitycenterpb.ListFindingsRequest{
					Parent:   sourceName,
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
			name: "GroupFindings",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				it := c.GroupFindings(ctx, &securitycenterpb.GroupFindingsRequest{
					Parent:   sourceName,
					GroupBy:  "category",
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
			name: "CreateFinding",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.CreateFinding(ctx, &securitycenterpb.CreateFindingRequest{
					Parent:    sourceName,
					FindingId: findingID,
					Finding: &securitycenterpb.Finding{
						Name:     findingName,
						Category: "OPEN_FIREWALL",
					},
				})
				return err
			},
		},
		{
			name: "UpdateFinding",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.UpdateFinding(ctx, &securitycenterpb.UpdateFindingRequest{
					Finding: &securitycenterpb.Finding{
						Name:     findingName,
						Severity: securitycenterpb.Finding_HIGH,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"severity"}},
				})
				return err
			},
		},
		{
			name: "SetFindingState",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.SetFindingState(ctx, &securitycenterpb.SetFindingStateRequest{
					Name:      findingName,
					State:     securitycenterpb.Finding_INACTIVE,
					StartTime: timestamppb.New(time.Now().UTC()),
				})
				return err
			},
		},
		{
			name: "SetMute",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.SetMute(ctx, &securitycenterpb.SetMuteRequest{
					Name: findingName,
					Mute: securitycenterpb.Finding_MUTED,
				})
				return err
			},
		},
		{
			name: "ListMuteConfigs",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				it := c.ListMuteConfigs(ctx, &securitycenterpb.ListMuteConfigsRequest{
					Parent:   orgName,
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
			name: "GetMuteConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.GetMuteConfig(ctx, &securitycenterpb.GetMuteConfigRequest{Name: muteConfigName})
				return err
			},
		},
		{
			name: "CreateMuteConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.CreateMuteConfig(ctx, &securitycenterpb.CreateMuteConfigRequest{
					Parent:       orgName,
					MuteConfigId: muteConfigID,
					MuteConfig: &securitycenterpb.MuteConfig{
						Filter: "severity=\"HIGH\"",
					},
				})
				return err
			},
		},
		{
			name: "UpdateMuteConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.UpdateMuteConfig(ctx, &securitycenterpb.UpdateMuteConfigRequest{
					MuteConfig: &securitycenterpb.MuteConfig{
						Name:   muteConfigName,
						Filter: "severity=\"CRITICAL\"",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"filter"}},
				})
				return err
			},
		},
		{
			name: "DeleteMuteConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				return c.DeleteMuteConfig(ctx, &securitycenterpb.DeleteMuteConfigRequest{Name: muteConfigName})
			},
		},
		{
			name: "ListNotificationConfigs",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				it := c.ListNotificationConfigs(ctx, &securitycenterpb.ListNotificationConfigsRequest{
					Parent:   orgName,
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
			name: "GetNotificationConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.GetNotificationConfig(ctx, &securitycenterpb.GetNotificationConfigRequest{Name: notificationConfigName})
				return err
			},
		},
		{
			name: "CreateNotificationConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.CreateNotificationConfig(ctx, &securitycenterpb.CreateNotificationConfigRequest{
					Parent:   orgName,
					ConfigId: notificationConfigID,
					NotificationConfig: &securitycenterpb.NotificationConfig{
						PubsubTopic: "projects/stackyard/topics/security-alerts",
					},
				})
				return err
			},
		},
		{
			name: "UpdateNotificationConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.UpdateNotificationConfig(ctx, &securitycenterpb.UpdateNotificationConfigRequest{
					NotificationConfig: &securitycenterpb.NotificationConfig{
						Name: notificationConfigName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
				})
				return err
			},
		},
		{
			name: "DeleteNotificationConfig",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				return c.DeleteNotificationConfig(ctx, &securitycenterpb.DeleteNotificationConfigRequest{Name: notificationConfigName})
			},
		},
		{
			name: "GetOrganizationSettings",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.GetOrganizationSettings(ctx, &securitycenterpb.GetOrganizationSettingsRequest{Name: orgSettingsName})
				return err
			},
		},
		{
			name: "UpdateOrganizationSettings",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.UpdateOrganizationSettings(ctx, &securitycenterpb.UpdateOrganizationSettingsRequest{
					OrganizationSettings: &securitycenterpb.OrganizationSettings{Name: orgSettingsName},
					UpdateMask:           &fieldmaskpb.FieldMask{Paths: []string{"enable_asset_discovery"}},
				})
				return err
			},
		},
		{
			name: "ListBigQueryExports",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				it := c.ListBigQueryExports(ctx, &securitycenterpb.ListBigQueryExportsRequest{
					Parent:   orgName,
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
			name: "GetBigQueryExport",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.GetBigQueryExport(ctx, &securitycenterpb.GetBigQueryExportRequest{Name: bigQueryExportName})
				return err
			},
		},
		{
			name: "CreateBigQueryExport",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.CreateBigQueryExport(ctx, &securitycenterpb.CreateBigQueryExportRequest{
					Parent:           orgName,
					BigQueryExportId: bigQueryExportID,
					BigQueryExport: &securitycenterpb.BigQueryExport{
						Dataset: "projects/stackyard/datasets/security",
					},
				})
				return err
			},
		},
		{
			name: "UpdateBigQueryExport",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.UpdateBigQueryExport(ctx, &securitycenterpb.UpdateBigQueryExportRequest{
					BigQueryExport: &securitycenterpb.BigQueryExport{
						Name: bigQueryExportName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"filter"}},
				})
				return err
			},
		},
		{
			name: "DeleteBigQueryExport",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				return c.DeleteBigQueryExport(ctx, &securitycenterpb.DeleteBigQueryExportRequest{Name: bigQueryExportName})
			},
		},
		{
			name: "BulkMuteFindings",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				op, err := c.BulkMuteFindings(ctx, &securitycenterpb.BulkMuteFindingsRequest{
					Parent: sourceName,
					Filter: "severity=\"HIGH\"",
				})
				if err != nil {
					return err
				}
				_ = op.Name()
				return nil
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: sourceName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: sourceName,
					Policy: &iampb.Policy{
						Version: 1,
						Bindings: []*iampb.Binding{
							{
								Role:    "roles/securitycenter.findingsEditor",
								Members: []string{"user:analyst@example.invalid"},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    sourceName,
					Permissions: []string{"securitycenter.sources.get"},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: orgName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *securitycenter.Client) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
	}

	for _, call := range calls {
		err := runWithStartupRetry(ctx, client, call.call)
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

func runWithStartupRetry(ctx context.Context, client *securitycenter.Client, fn func(context.Context, *securitycenter.Client) error) error {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		lastErr = fn(ctx, client)
		if lastErr == nil || !isRetryableStartupError(lastErr) {
			return lastErr
		}
		time.Sleep(250 * time.Millisecond)
	}
	return lastErr
}

func isRetryableStartupError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "dial tcp")
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

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unable to resolve \"type.googleapis.com/google.cloud.securitycenter.v1.operationmetadata\"") {
		return true
	}

	return strings.Contains(msg, "notimplemented")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close securitycenter client: %v\n", err)
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
