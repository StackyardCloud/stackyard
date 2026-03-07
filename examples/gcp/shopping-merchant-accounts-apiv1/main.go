package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	accounts "cloud.google.com/go/shopping/merchant/accounts/apiv1"
	"cloud.google.com/go/shopping/merchant/accounts/apiv1/accountspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/proto"
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

	accountID := getenv("STACKYARD_GCP_MERCHANT_ACCOUNT_ID", "123456")
	userID := getenv("STACKYARD_GCP_MERCHANT_USER_ID", "owner@example.com")
	programID := getenv("STACKYARD_GCP_MERCHANT_PROGRAM_ID", "free-listings")
	regionID := getenv("STACKYARD_GCP_MERCHANT_REGION_ID", "us-east")

	accountName := fmt.Sprintf("accounts/%s", accountID)
	userName := fmt.Sprintf("%s/users/%s", accountName, userID)
	programName := fmt.Sprintf("%s/programs/%s", accountName, programID)
	regionName := fmt.Sprintf("%s/regions/%s", accountName, regionID)
	termsName := "termsOfService/latest"

	fmt.Printf("Stackyard GCP Shopping Merchant Accounts shopping/merchant/accounts/apiv1 clients using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shopping-merchant-accounts-apiv1",
		},
	}

	accountsClient, err := accounts.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant accounts client: %v", err)
	}
	defer closeClient("accounts", accountsClient.Close)

	usersClient, err := accounts.NewUserRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant users client: %v", err)
	}
	defer closeClient("users", usersClient.Close)

	programsClient, err := accounts.NewProgramsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant programs client: %v", err)
	}
	defer closeClient("programs", programsClient.Close)

	homepageClient, err := accounts.NewHomepageRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant homepage client: %v", err)
	}
	defer closeClient("homepage", homepageClient.Close)

	regionsClient, err := accounts.NewRegionsRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant regions client: %v", err)
	}
	defer closeClient("regions", regionsClient.Close)

	termsClient, err := accounts.NewTermsOfServiceRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create merchant terms client: %v", err)
	}
	defer closeClient("terms", termsClient.Close)

	calls := []callSpec{
		{
			name: "ListAccounts",
			call: func(ctx context.Context) error {
				it := accountsClient.ListAccounts(ctx, &accountspb.ListAccountsRequest{PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateAndConfigureAccount",
			call: func(ctx context.Context) error {
				_, err := accountsClient.CreateAndConfigureAccount(ctx, &accountspb.CreateAndConfigureAccountRequest{
					Account: &accountspb.Account{
						Name:         "accounts/123459",
						AccountName:  "Stackyard Test Account",
						LanguageCode: "en-US",
					},
					Service: []*accountspb.CreateAndConfigureAccountRequest_AddAccountService{
						{
							Provider: proto.String("providers/123"),
							ServiceType: &accountspb.CreateAndConfigureAccountRequest_AddAccountService_AccountAggregation{
								AccountAggregation: &accountspb.AccountAggregation{},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetAccount",
			call: func(ctx context.Context) error {
				_, err := accountsClient.GetAccount(ctx, &accountspb.GetAccountRequest{Name: accountName})
				return err
			},
		},
		{
			name: "UpdateAccount",
			call: func(ctx context.Context) error {
				_, err := accountsClient.UpdateAccount(ctx, &accountspb.UpdateAccountRequest{
					Account: &accountspb.Account{
						Name:         accountName,
						AccountName:  "Stackyard Merchant Updated",
						LanguageCode: "en-US",
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"account_name"}},
				})
				return err
			},
		},
		{
			name: "ListSubAccounts",
			call: func(ctx context.Context) error {
				it := accountsClient.ListSubAccounts(ctx, &accountspb.ListSubAccountsRequest{Provider: accountName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateUser",
			call: func(ctx context.Context) error {
				_, err := usersClient.CreateUser(ctx, &accountspb.CreateUserRequest{
					Parent: accountName,
					UserId: userID,
					User: &accountspb.User{
						Name:         userName,
						AccessRights: []accountspb.AccessRight{accountspb.AccessRight_ADMIN},
					},
				})
				return err
			},
		},
		{
			name: "ListUsers",
			call: func(ctx context.Context) error {
				it := usersClient.ListUsers(ctx, &accountspb.ListUsersRequest{Parent: accountName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetUser",
			call: func(ctx context.Context) error {
				_, err := usersClient.GetUser(ctx, &accountspb.GetUserRequest{Name: userName})
				return err
			},
		},
		{
			name: "VerifySelf",
			call: func(ctx context.Context) error {
				_, err := usersClient.VerifySelf(ctx, &accountspb.VerifySelfRequest{Account: accountName})
				return err
			},
		},
		{
			name: "UpdateUser",
			call: func(ctx context.Context) error {
				_, err := usersClient.UpdateUser(ctx, &accountspb.UpdateUserRequest{
					User: &accountspb.User{
						Name:         userName,
						AccessRights: []accountspb.AccessRight{accountspb.AccessRight_STANDARD},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"access_rights"}},
				})
				return err
			},
		},
		{
			name: "DeleteUser",
			call: func(ctx context.Context) error {
				return usersClient.DeleteUser(ctx, &accountspb.DeleteUserRequest{Name: userName})
			},
		},
		{
			name: "ListPrograms",
			call: func(ctx context.Context) error {
				it := programsClient.ListPrograms(ctx, &accountspb.ListProgramsRequest{Parent: accountName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetProgram",
			call: func(ctx context.Context) error {
				_, err := programsClient.GetProgram(ctx, &accountspb.GetProgramRequest{Name: programName})
				return err
			},
		},
		{
			name: "EnableProgram",
			call: func(ctx context.Context) error {
				_, err := programsClient.EnableProgram(ctx, &accountspb.EnableProgramRequest{Name: programName})
				return err
			},
		},
		{
			name: "DisableProgram",
			call: func(ctx context.Context) error {
				_, err := programsClient.DisableProgram(ctx, &accountspb.DisableProgramRequest{Name: programName})
				return err
			},
		},
		{
			name: "GetHomepage",
			call: func(ctx context.Context) error {
				_, err := homepageClient.GetHomepage(ctx, &accountspb.GetHomepageRequest{Name: accountName + "/homepage"})
				return err
			},
		},
		{
			name: "UpdateHomepage",
			call: func(ctx context.Context) error {
				_, err := homepageClient.UpdateHomepage(ctx, &accountspb.UpdateHomepageRequest{
					Homepage: &accountspb.Homepage{
						Name: accountName + "/homepage",
						Uri:  proto.String("https://merchant.stackyard.example"),
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"uri"}},
				})
				return err
			},
		},
		{
			name: "ClaimHomepage",
			call: func(ctx context.Context) error {
				_, err := homepageClient.ClaimHomepage(ctx, &accountspb.ClaimHomepageRequest{Name: accountName + "/homepage"})
				return err
			},
		},
		{
			name: "UnclaimHomepage",
			call: func(ctx context.Context) error {
				_, err := homepageClient.UnclaimHomepage(ctx, &accountspb.UnclaimHomepageRequest{Name: accountName + "/homepage"})
				return err
			},
		},
		{
			name: "CreateRegion",
			call: func(ctx context.Context) error {
				_, err := regionsClient.CreateRegion(ctx, &accountspb.CreateRegionRequest{
					Parent:   accountName,
					RegionId: regionID,
					Region: &accountspb.Region{
						Name:        regionName,
						DisplayName: proto.String("US East"),
					},
				})
				return err
			},
		},
		{
			name: "ListRegions",
			call: func(ctx context.Context) error {
				it := regionsClient.ListRegions(ctx, &accountspb.ListRegionsRequest{Parent: accountName, PageSize: 1})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetRegion",
			call: func(ctx context.Context) error {
				_, err := regionsClient.GetRegion(ctx, &accountspb.GetRegionRequest{Name: regionName})
				return err
			},
		},
		{
			name: "UpdateRegion",
			call: func(ctx context.Context) error {
				_, err := regionsClient.UpdateRegion(ctx, &accountspb.UpdateRegionRequest{
					Region: &accountspb.Region{
						Name:        regionName,
						DisplayName: proto.String("US East Updated"),
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "BatchUpdateRegions",
			call: func(ctx context.Context) error {
				_, err := regionsClient.BatchUpdateRegions(ctx, &accountspb.BatchUpdateRegionsRequest{
					Parent: accountName,
					Requests: []*accountspb.UpdateRegionRequest{
						{
							Region:     &accountspb.Region{Name: regionName, DisplayName: proto.String("US East Batch")},
							UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
						},
					},
				})
				return err
			},
		},
		{
			name: "BatchDeleteRegions",
			call: func(ctx context.Context) error {
				return regionsClient.BatchDeleteRegions(ctx, &accountspb.BatchDeleteRegionsRequest{
					Parent:   accountName,
					Requests: []*accountspb.DeleteRegionRequest{{Name: regionName}},
				})
			},
		},
		{
			name: "DeleteRegion",
			call: func(ctx context.Context) error {
				return regionsClient.DeleteRegion(ctx, &accountspb.DeleteRegionRequest{Name: regionName})
			},
		},
		{
			name: "RetrieveLatestTermsOfService",
			call: func(ctx context.Context) error {
				_, err := termsClient.RetrieveLatestTermsOfService(ctx, &accountspb.RetrieveLatestTermsOfServiceRequest{
					Kind:       accountspb.TermsOfServiceKind_MERCHANT_CENTER,
					RegionCode: "US",
				})
				return err
			},
		},
		{
			name: "GetTermsOfService",
			call: func(ctx context.Context) error {
				_, err := termsClient.GetTermsOfService(ctx, &accountspb.GetTermsOfServiceRequest{Name: termsName})
				return err
			},
		},
		{
			name: "AcceptTermsOfService",
			call: func(ctx context.Context) error {
				_, err := termsClient.AcceptTermsOfService(ctx, &accountspb.AcceptTermsOfServiceRequest{
					Name:       termsName,
					Account:    accountName,
					RegionCode: "US",
				})
				return err
			},
		},
	}

	for _, spec := range calls {
		if err := spec.call(ctx); err != nil {
			exitf("%s failed: %v", spec.name, err)
		}
		fmt.Printf("%s succeeded\n", spec.name)
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, endpoint string) error {
	target := strings.TrimRight(endpoint, "/") + "/accounts/v1/accounts?pageSize=1"
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "shopping-merchant-accounts-apiv1")
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("stackyard endpoint %s did not become ready in time", target)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func closeClient(name string, fn func() error) {
	if err := fn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close %s client: %v\n", name, err)
	}
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
	if strings.TrimSpace(clone.Header.Get("X-Stackyard-GCP-Service")) == "" {
		clone.Header.Set("X-Stackyard-GCP-Service", "shopping-merchant-accounts")
	}
	if strings.TrimSpace(clone.Header.Get("User-Agent")) == "" {
		clone.Header.Set("User-Agent", "stackyard-"+t.serviceName)
	}
	if t.base == nil {
		return http.DefaultTransport.RoundTrip(clone)
	}
	return t.base.RoundTrip(clone)
}
