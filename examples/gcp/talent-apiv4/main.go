package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	talent "cloud.google.com/go/talent/apiv4"
	"cloud.google.com/go/talent/apiv4/talentpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"
	projectID := getenv("STACKYARD_GCP_PROJECT", "stackyard")
	projectParent := "projects/" + projectID

	tenantName := projectParent + "/tenants/tenant-1"
	companyName := tenantName + "/companies/company-1"
	jobName := tenantName + "/jobs/job-1"

	fmt.Printf("Stackyard GCP Talent Solution v4 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "talent",
		},
	}

	tenantClient, err := talent.NewTenantRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create tenant client: %v", err)
	}
	defer closeClient("tenant", tenantClient.Close)

	companyClient, err := talent.NewCompanyRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create company client: %v", err)
	}
	defer closeClient("company", companyClient.Close)

	jobClient, err := talent.NewJobRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create job client: %v", err)
	}
	defer closeClient("job", jobClient.Close)

	completionClient, err := talent.NewCompletionRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create completion client: %v", err)
	}
	defer closeClient("completion", completionClient.Close)

	eventClient, err := talent.NewEventRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create event client: %v", err)
	}
	defer closeClient("event", eventClient.Close)

	calls := []callSpec{
		{
			name: "CreateTenant",
			call: func(ctx context.Context) error {
				created, err := tenantClient.CreateTenant(ctx, &talentpb.CreateTenantRequest{
					Parent: projectParent,
					Tenant: &talentpb.Tenant{
						ExternalId: "tenant-ext-created",
					},
				})
				if err != nil {
					return err
				}
				if created != nil && strings.TrimSpace(created.GetName()) != "" {
					tenantName = created.GetName()
				}
				return nil
			},
		},
		{
			name: "ListTenants",
			call: func(ctx context.Context) error {
				it := tenantClient.ListTenants(ctx, &talentpb.ListTenantsRequest{
					Parent:   projectParent,
					PageSize: 1,
				})
				tenant, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if tenant != nil && strings.TrimSpace(tenant.GetName()) != "" {
					tenantName = tenant.GetName()
				}
				return nil
			},
		},
		{
			name: "GetTenant",
			call: func(ctx context.Context) error {
				_, err := tenantClient.GetTenant(ctx, &talentpb.GetTenantRequest{Name: tenantName})
				return err
			},
		},
		{
			name: "UpdateTenant",
			call: func(ctx context.Context) error {
				updated, err := tenantClient.UpdateTenant(ctx, &talentpb.UpdateTenantRequest{
					Tenant: &talentpb.Tenant{
						Name:       tenantName,
						ExternalId: "tenant-ext-updated",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"external_id"}},
				})
				if err != nil {
					return err
				}
				if updated != nil && strings.TrimSpace(updated.GetName()) != "" {
					tenantName = updated.GetName()
				}
				return nil
			},
		},
		{
			name: "CreateCompany",
			call: func(ctx context.Context) error {
				created, err := companyClient.CreateCompany(ctx, &talentpb.CreateCompanyRequest{
					Parent: tenantName,
					Company: &talentpb.Company{
						DisplayName: "Stackyard Inc",
						ExternalId:  "company-ext-created",
					},
				})
				if err != nil {
					return err
				}
				if created != nil && strings.TrimSpace(created.GetName()) != "" {
					companyName = created.GetName()
				}
				return nil
			},
		},
		{
			name: "ListCompanies",
			call: func(ctx context.Context) error {
				it := companyClient.ListCompanies(ctx, &talentpb.ListCompaniesRequest{
					Parent:          tenantName,
					PageSize:        1,
					RequireOpenJobs: true,
				})
				company, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if company != nil && strings.TrimSpace(company.GetName()) != "" {
					companyName = company.GetName()
				}
				return nil
			},
		},
		{
			name: "GetCompany",
			call: func(ctx context.Context) error {
				_, err := companyClient.GetCompany(ctx, &talentpb.GetCompanyRequest{Name: companyName})
				return err
			},
		},
		{
			name: "UpdateCompany",
			call: func(ctx context.Context) error {
				updated, err := companyClient.UpdateCompany(ctx, &talentpb.UpdateCompanyRequest{
					Company: &talentpb.Company{
						Name:        companyName,
						DisplayName: "Updated Stackyard Inc",
						ExternalId:  "company-ext-updated",
						WebsiteUri:  "https://example.com",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "external_id", "website_uri"}},
				})
				if err != nil {
					return err
				}
				if updated != nil && strings.TrimSpace(updated.GetName()) != "" {
					companyName = updated.GetName()
				}
				return nil
			},
		},
		{
			name: "CreateJob",
			call: func(ctx context.Context) error {
				created, err := jobClient.CreateJob(ctx, &talentpb.CreateJobRequest{
					Parent: tenantName,
					Job: &talentpb.Job{
						Company:       companyName,
						RequisitionId: "req-created-1",
						Title:         "Platform Engineer",
						Description:   "Build distributed systems",
					},
				})
				if err != nil {
					return err
				}
				if created != nil && strings.TrimSpace(created.GetName()) != "" {
					jobName = created.GetName()
				}
				return nil
			},
		},
		{
			name: "ListJobs",
			call: func(ctx context.Context) error {
				it := jobClient.ListJobs(ctx, &talentpb.ListJobsRequest{
					Parent:   tenantName,
					Filter:   fmt.Sprintf(`companyName = "%s"`, companyName),
					PageSize: 1,
				})
				job, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				if err != nil {
					return err
				}
				if job != nil && strings.TrimSpace(job.GetName()) != "" {
					jobName = job.GetName()
				}
				return nil
			},
		},
		{
			name: "GetJob",
			call: func(ctx context.Context) error {
				_, err := jobClient.GetJob(ctx, &talentpb.GetJobRequest{Name: jobName})
				return err
			},
		},
		{
			name: "UpdateJob",
			call: func(ctx context.Context) error {
				updated, err := jobClient.UpdateJob(ctx, &talentpb.UpdateJobRequest{
					Job: &talentpb.Job{
						Name:          jobName,
						Company:       companyName,
						RequisitionId: "req-updated-1",
						Title:         "Updated Platform Engineer",
						Description:   "Build and operate distributed systems",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"company", "requisition_id", "title", "description"}},
				})
				if err != nil {
					return err
				}
				if updated != nil && strings.TrimSpace(updated.GetName()) != "" {
					jobName = updated.GetName()
				}
				return nil
			},
		},
		{
			name: "SearchJobs",
			call: func(ctx context.Context) error {
				_, err := jobClient.SearchJobs(ctx, &talentpb.SearchJobsRequest{
					Parent: tenantName,
					RequestMetadata: &talentpb.RequestMetadata{
						Domain:    "example.com",
						SessionId: "session-1",
						UserId:    "user-1",
					},
					JobQuery:    &talentpb.JobQuery{Query: "Engineer"},
					MaxPageSize: 1,
				})
				return err
			},
		},
		{
			name: "SearchJobsForAlert",
			call: func(ctx context.Context) error {
				_, err := jobClient.SearchJobsForAlert(ctx, &talentpb.SearchJobsRequest{
					Parent: tenantName,
					RequestMetadata: &talentpb.RequestMetadata{
						Domain:    "example.com",
						SessionId: "session-1",
						UserId:    "user-1",
					},
					JobQuery:    &talentpb.JobQuery{Query: "Engineer"},
					MaxPageSize: 1,
				})
				return err
			},
		},
		{
			name: "BatchCreateJobs",
			call: func(ctx context.Context) error {
				op, err := jobClient.BatchCreateJobs(ctx, &talentpb.BatchCreateJobsRequest{
					Parent: tenantName,
					Jobs: []*talentpb.Job{
						{
							Company:       companyName,
							RequisitionId: "req-batch-create-1",
							Title:         "Batch Create Job",
							Description:   "Batch create description",
						},
					},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "BatchUpdateJobs",
			call: func(ctx context.Context) error {
				op, err := jobClient.BatchUpdateJobs(ctx, &talentpb.BatchUpdateJobsRequest{
					Parent: tenantName,
					Jobs: []*talentpb.Job{
						{
							Name:          jobName,
							Company:       companyName,
							RequisitionId: "req-batch-update-1",
							Title:         "Batch Update Job",
							Description:   "Batch update description",
						},
					},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "BatchDeleteJobs",
			call: func(ctx context.Context) error {
				op, err := jobClient.BatchDeleteJobs(ctx, &talentpb.BatchDeleteJobsRequest{
					Parent: tenantName,
					Names:  []string{jobName},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "CompleteQuery",
			call: func(ctx context.Context) error {
				_, err := completionClient.CompleteQuery(ctx, &talentpb.CompleteQueryRequest{
					Tenant:   tenantName,
					Query:    "Stack",
					PageSize: 2,
					Company:  companyName,
					Scope:    talentpb.CompleteQueryRequest_PUBLIC,
					Type:     talentpb.CompleteQueryRequest_COMBINED,
				})
				return err
			},
		},
		{
			name: "CreateClientEvent",
			call: func(ctx context.Context) error {
				_, err := eventClient.CreateClientEvent(ctx, &talentpb.CreateClientEventRequest{
					Parent: tenantName,
					ClientEvent: &talentpb.ClientEvent{
						RequestId:  "talent-event-req-1",
						EventId:    "event-1",
						CreateTime: timestamppb.Now(),
						Event: &talentpb.ClientEvent_JobEvent{
							JobEvent: &talentpb.JobEvent{
								Type: talentpb.JobEvent_VIEW,
								Jobs: []string{jobName},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteJob",
			call: func(ctx context.Context) error {
				return jobClient.DeleteJob(ctx, &talentpb.DeleteJobRequest{Name: jobName})
			},
		},
		{
			name: "DeleteCompany",
			call: func(ctx context.Context) error {
				return companyClient.DeleteCompany(ctx, &talentpb.DeleteCompanyRequest{Name: companyName})
			},
		},
		{
			name: "DeleteTenant",
			call: func(ctx context.Context) error {
				return tenantClient.DeleteTenant(ctx, &talentpb.DeleteTenantRequest{Name: tenantName})
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	if strings.TrimSpace(t.serviceName) != "" {
		clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	}
	return t.baseTransport().RoundTrip(clone)
}

func (t stackyardHeaderTransport) baseTransport() http.RoundTripper {
	if t.base != nil {
		return t.base
	}
	return http.DefaultTransport
}
