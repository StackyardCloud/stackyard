package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	recaptchaenterprise "cloud.google.com/go/recaptchaenterprise/v2/apiv1"
	"cloud.google.com/go/recaptchaenterprise/v2/apiv1/recaptchaenterprisepb"
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
	call func(context.Context, *recaptchaenterprise.Client) error
}

func main() {
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	keyID := getenv("STACKYARD_GCP_RECAPTCHA_KEY_ID", "site-key-1")
	assessmentID := getenv("STACKYARD_GCP_RECAPTCHA_ASSESSMENT_ID", "assessment-1")
	grpcEndpoint := grpcEndpointFromEnv()

	project := "projects/" + projectID
	keyName := project + "/keys/" + keyID
	assessmentName := project + "/assessments/" + assessmentID

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP reCAPTCHA Enterprise v2/apiv1 client using %s\n", grpcEndpoint)

	client, err := recaptchaenterprise.NewClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		exitf("failed to create recaptchaenterprise client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "CreateKey",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				_, err := c.CreateKey(ctx, &recaptchaenterprisepb.CreateKeyRequest{
					Parent: project,
					Key: &recaptchaenterprisepb.Key{
						DisplayName: "Stackyard reCAPTCHA key",
						PlatformSettings: &recaptchaenterprisepb.Key_WebSettings{
							WebSettings: &recaptchaenterprisepb.WebKeySettings{
								AllowAllDomains: true,
								IntegrationType: recaptchaenterprisepb.WebKeySettings_SCORE,
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetKey",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				_, err := c.GetKey(ctx, &recaptchaenterprisepb.GetKeyRequest{Name: keyName})
				return err
			},
		},
		{
			name: "ListKeys",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				it := c.ListKeys(ctx, &recaptchaenterprisepb.ListKeysRequest{Parent: project, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "UpdateKey",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				_, err := c.UpdateKey(ctx, &recaptchaenterprisepb.UpdateKeyRequest{
					Key: &recaptchaenterprisepb.Key{
						Name:        keyName,
						DisplayName: "Stackyard reCAPTCHA key updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "CreateAssessment",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				_, err := c.CreateAssessment(ctx, &recaptchaenterprisepb.CreateAssessmentRequest{
					Parent: project,
					Assessment: &recaptchaenterprisepb.Assessment{
						Event: &recaptchaenterprisepb.Event{
							Token:   "stackyard-token",
							SiteKey: keyName,
							UserInfo: &recaptchaenterprisepb.UserInfo{
								AccountId: "user-1",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "AnnotateAssessment",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				_, err := c.AnnotateAssessment(ctx, &recaptchaenterprisepb.AnnotateAssessmentRequest{
					Name:       assessmentName,
					Annotation: recaptchaenterprisepb.AnnotateAssessmentRequest_LEGITIMATE,
					Reasons:    []recaptchaenterprisepb.AnnotateAssessmentRequest_Reason{recaptchaenterprisepb.AnnotateAssessmentRequest_CHARGEBACK},
				})
				return err
			},
		},
		{
			name: "GetMetrics",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				_, err := c.GetMetrics(ctx, &recaptchaenterprisepb.GetMetricsRequest{Name: keyName + "/metrics"})
				return err
			},
		},
		{
			name: "ListFirewallPolicies",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				it := c.ListFirewallPolicies(ctx, &recaptchaenterprisepb.ListFirewallPoliciesRequest{Parent: project, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "SearchRelatedAccountGroupMemberships",
			call: func(ctx context.Context, c *recaptchaenterprise.Client) error {
				it := c.SearchRelatedAccountGroupMemberships(ctx, &recaptchaenterprisepb.SearchRelatedAccountGroupMembershipsRequest{
					Project:   project,
					AccountId: "user-1",
					PageSize:  1,
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
		callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		callCancel()

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
		fmt.Fprintf(os.Stderr, "warning: close recaptchaenterprise client: %v\n", err)
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
