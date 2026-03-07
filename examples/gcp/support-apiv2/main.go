package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	support "cloud.google.com/go/support/apiv2"
	"cloud.google.com/go/support/apiv2/supportpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
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

	parent := getenv("STACKYARD_GCP_PARENT", "projects/stackyard")
	caseName := parent + "/cases/case-open-1"
	classificationID := getenv("STACKYARD_GCP_CASE_CLASSIFICATION", "technical-issue/compute-engine")

	fmt.Printf("Stackyard GCP Cloud Support V2 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "support",
		},
	}

	caseClient, err := support.NewCaseRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create support case client: %v", err)
	}
	defer closeClient("case", caseClient.Close)

	commentClient, err := support.NewCommentRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create support comment client: %v", err)
	}
	defer closeClient("comment", commentClient.Close)

	attachmentClient, err := support.NewCaseAttachmentRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create support attachment client: %v", err)
	}
	defer closeClient("attachment", attachmentClient.Close)

	calls := []callSpec{
		{
			name: "SearchCaseClassifications",
			call: func(ctx context.Context) error {
				it := caseClient.SearchCaseClassifications(ctx, &supportpb.SearchCaseClassificationsRequest{PageSize: 1})
				classification, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if classification != nil && strings.TrimSpace(classification.GetId()) != "" {
					classificationID = classification.GetId()
				}
				return nil
			},
		},
		{
			name: "CreateCase",
			call: func(ctx context.Context) error {
				created, err := caseClient.CreateCase(ctx, &supportpb.CreateCaseRequest{
					Parent: parent,
					Case: &supportpb.Case{
						DisplayName: "Stackyard staging case",
						Description: "Deterministic staged support case",
						Classification: &supportpb.CaseClassification{
							Id: classificationID,
						},
						Priority:                 supportpb.Case_P2,
						TestCase:                 true,
						SubscriberEmailAddresses: []string{"ops@example.com"},
					},
				})
				if err != nil {
					return err
				}
				if created != nil && strings.TrimSpace(created.GetName()) != "" {
					caseName = created.GetName()
				}
				return nil
			},
		},
		{
			name: "GetCase",
			call: func(ctx context.Context) error {
				_, err := caseClient.GetCase(ctx, &supportpb.GetCaseRequest{Name: caseName})
				return err
			},
		},
		{
			name: "ListCases",
			call: func(ctx context.Context) error {
				it := caseClient.ListCases(ctx, &supportpb.ListCasesRequest{Parent: parent, PageSize: 1, Filter: "state=OPEN"})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "SearchCases",
			call: func(ctx context.Context) error {
				it := caseClient.SearchCases(ctx, &supportpb.SearchCasesRequest{Parent: parent, Query: "state=OPEN", PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "UpdateCase",
			call: func(ctx context.Context) error {
				_, err := caseClient.UpdateCase(ctx, &supportpb.UpdateCaseRequest{
					Case: &supportpb.Case{
						Name:                     caseName,
						DisplayName:              "Updated Stackyard staging case",
						Priority:                 supportpb.Case_P1,
						SubscriberEmailAddresses: []string{"ops@example.com"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "priority", "subscriber_email_addresses"}},
				})
				return err
			},
		},
		{
			name: "CreateComment",
			call: func(ctx context.Context) error {
				_, err := commentClient.CreateComment(ctx, &supportpb.CreateCommentRequest{
					Parent: caseName,
					Comment: &supportpb.Comment{
						Body: "Any updates on the incident?",
					},
				})
				return err
			},
		},
		{
			name: "ListComments",
			call: func(ctx context.Context) error {
				it := commentClient.ListComments(ctx, &supportpb.ListCommentsRequest{Parent: caseName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "ListAttachments",
			call: func(ctx context.Context) error {
				it := attachmentClient.ListAttachments(ctx, &supportpb.ListAttachmentsRequest{Parent: caseName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "EscalateCase",
			call: func(ctx context.Context) error {
				_, err := caseClient.EscalateCase(ctx, &supportpb.EscalateCaseRequest{
					Name: caseName,
					Escalation: &supportpb.Escalation{
						Reason:        supportpb.Escalation_TECHNICAL_EXPERTISE,
						Justification: "Production impact is increasing",
					},
				})
				return err
			},
		},
		{
			name: "CloseCase",
			call: func(ctx context.Context) error {
				_, err := caseClient.CloseCase(ctx, &supportpb.CloseCaseRequest{Name: caseName})
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
