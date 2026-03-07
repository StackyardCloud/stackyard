package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	chronicle "cloud.google.com/go/chronicle/apiv1"
	"cloud.google.com/go/chronicle/apiv1/chroniclepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *chronicle.InstanceClient, *chronicle.DataAccessControlClient, *chronicle.ReferenceListClient, *chronicle.RuleClient, *chronicle.EntityClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	instanceName := getenv("STACKYARD_GCP_CHRONICLE_INSTANCE", "projects/stackyard/locations/us/instances/default")
	dataAccessLabelName := getenv("STACKYARD_GCP_CHRONICLE_DATA_ACCESS_LABEL", instanceName+"/dataAccessLabels/team-label")
	dataAccessScopeName := getenv("STACKYARD_GCP_CHRONICLE_DATA_ACCESS_SCOPE", instanceName+"/dataAccessScopes/team-scope")
	referenceListName := getenv("STACKYARD_GCP_CHRONICLE_REFERENCE_LIST", instanceName+"/referenceLists/team-reference-list")
	ruleName := getenv("STACKYARD_GCP_CHRONICLE_RULE", instanceName+"/rules/team-rule")
	watchlistName := getenv("STACKYARD_GCP_CHRONICLE_WATCHLIST", instanceName+"/watchlists/high-risk-entities")
	ruleDeploymentName := getenv("STACKYARD_GCP_CHRONICLE_RULE_DEPLOYMENT", ruleName+"/deployment")
	retrohuntName := getenv("STACKYARD_GCP_CHRONICLE_RETROHUNT", ruleName+"/retrohunts/team-retrohunt")

	fmt.Printf("Stackyard GCP Chronicle apiv1 clients using %s\n", apiEndpoint)

	instanceClient, err := chronicle.NewInstanceRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create chronicle instance client: %v", err)
	}
	defer closeClient("instance", instanceClient.Close)

	dataAccessControlClient, err := chronicle.NewDataAccessControlRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create chronicle data access control client: %v", err)
	}
	defer closeClient("data access control", dataAccessControlClient.Close)

	referenceListClient, err := chronicle.NewReferenceListRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create chronicle reference list client: %v", err)
	}
	defer closeClient("reference list", referenceListClient.Close)

	ruleClient, err := chronicle.NewRuleRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create chronicle rule client: %v", err)
	}
	defer closeClient("rule", ruleClient.Close)

	entityClient, err := chronicle.NewEntityRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create chronicle entity client: %v", err)
	}
	defer closeClient("entity", entityClient.Close)

	calls := []callSpec{
		{
			name: "GetInstance",
			call: func(ctx context.Context, i *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, _ *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				_, err := i.GetInstance(ctx, &chroniclepb.GetInstanceRequest{
					Name: instanceName,
				})
				return err
			},
		},
		{
			name: "ListDataAccessLabels",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, d *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, _ *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				it := d.ListDataAccessLabels(ctx, &chroniclepb.ListDataAccessLabelsRequest{
					Parent:   instanceName,
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
			name: "GetDataAccessLabel",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, d *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, _ *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				_, err := d.GetDataAccessLabel(ctx, &chroniclepb.GetDataAccessLabelRequest{
					Name: dataAccessLabelName,
				})
				return err
			},
		},
		{
			name: "ListDataAccessScopes",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, d *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, _ *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				it := d.ListDataAccessScopes(ctx, &chroniclepb.ListDataAccessScopesRequest{
					Parent:   instanceName,
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
			name: "GetDataAccessScope",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, d *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, _ *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				_, err := d.GetDataAccessScope(ctx, &chroniclepb.GetDataAccessScopeRequest{
					Name: dataAccessScopeName,
				})
				return err
			},
		},
		{
			name: "ListReferenceLists",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, r *chronicle.ReferenceListClient, _ *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				it := r.ListReferenceLists(ctx, &chroniclepb.ListReferenceListsRequest{
					Parent:   instanceName,
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
			name: "GetReferenceList",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, r *chronicle.ReferenceListClient, _ *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				_, err := r.GetReferenceList(ctx, &chroniclepb.GetReferenceListRequest{
					Name: referenceListName,
				})
				return err
			},
		},
		{
			name: "ListRules",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, r *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				it := r.ListRules(ctx, &chroniclepb.ListRulesRequest{
					Parent:   instanceName,
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
			name: "GetRule",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, r *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				_, err := r.GetRule(ctx, &chroniclepb.GetRuleRequest{
					Name: ruleName,
				})
				return err
			},
		},
		{
			name: "ListRuleRevisions",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, r *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				it := r.ListRuleRevisions(ctx, &chroniclepb.ListRuleRevisionsRequest{
					Name:     ruleName,
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
			name: "ListRuleDeployments",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, r *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				it := r.ListRuleDeployments(ctx, &chroniclepb.ListRuleDeploymentsRequest{
					Parent:   instanceName + "/rules/-",
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
			name: "GetRuleDeployment",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, r *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				_, err := r.GetRuleDeployment(ctx, &chroniclepb.GetRuleDeploymentRequest{
					Name: ruleDeploymentName,
				})
				return err
			},
		},
		{
			name: "ListRetrohunts",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, r *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				it := r.ListRetrohunts(ctx, &chroniclepb.ListRetrohuntsRequest{
					Parent:   ruleName,
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
			name: "GetRetrohunt",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, r *chronicle.RuleClient, _ *chronicle.EntityClient) error {
				_, err := r.GetRetrohunt(ctx, &chroniclepb.GetRetrohuntRequest{
					Name: retrohuntName,
				})
				return err
			},
		},
		{
			name: "ListWatchlists",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, _ *chronicle.RuleClient, e *chronicle.EntityClient) error {
				it := e.ListWatchlists(ctx, &chroniclepb.ListWatchlistsRequest{
					Parent:   instanceName,
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
			name: "GetWatchlist",
			call: func(ctx context.Context, _ *chronicle.InstanceClient, _ *chronicle.DataAccessControlClient, _ *chronicle.ReferenceListClient, _ *chronicle.RuleClient, e *chronicle.EntityClient) error {
				_, err := e.GetWatchlist(ctx, &chroniclepb.GetWatchlistRequest{
					Name: watchlistName,
				})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, instanceClient, dataAccessControlClient, referenceListClient, ruleClient, entityClient)
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
