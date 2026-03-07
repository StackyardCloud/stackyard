package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantAccountsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantAccountsContractServer(t)
	account := "accounts/123456"
	user := account + "/users/owner@example.com"
	program := account + "/programs/free-listings"
	region := account + "/regions/us-east"
	tos := "termsOfService/latest"

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/accounts?pageSize=1", nil, "accounts")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/accounts:createAndConfigure", []byte(`{
		"account":{"name":"accounts/123459","accountName":"Stackyard New"},
		"service":[{"provider":"providers/123","accountAggregation":{}}]
	}`), "accounts/123459")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account, nil, account)
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"?updateMask=account_name", []byte(`{
		"account":{"name":"accounts/123456","accountName":"Stackyard Updated"}
	}`), "Stackyard Merchant")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodDelete, "/gcp/accounts/v1/"+account, nil, "{}")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+":listSubaccounts?pageSize=1", nil, "accounts")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/users", []byte(`{
		"parent":"accounts/123456",
		"userId":"owner@example.com"
	}`), "users/owner@example.com")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/users?pageSize=1", nil, "users")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+user, nil, user)
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+user+"?updateMask=access_rights", []byte(`{
		"user":{"name":"accounts/123456/users/owner@example.com"}
	}`), "owner@example.com")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/users/me:verifySelf?updateMask=account", []byte(`{
		"account":"accounts/123456"
	}`), "users/me")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodDelete, "/gcp/accounts/v1/"+user, nil, "{}")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/issues?pageSize=1", nil, "accountIssues")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/relationships?pageSize=1", nil, "accountRelationships")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/relationships/relationship-1", nil, "relationship-1")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/relationships/relationship-1?updateMask=label", []byte(`{"accountRelationship":{"name":"accounts/123456/relationships/relationship-1"}}`), "relationship-1")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/services?pageSize=1", nil, "accountServices")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/services/shopping-ads", nil, "shopping-ads")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/services:propose", []byte(`{
		"parent":"accounts/123456",
		"provider":"providers/123",
		"accountService":{"name":"accounts/123456/services/custom-service"}
	}`), "custom-service")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/services/shopping-ads:approve", []byte(`{"name":"accounts/123456/services/shopping-ads"}`), "APPROVED")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/services/shopping-ads:reject", []byte(`{"name":"accounts/123456/services/shopping-ads"}`), "REJECTED")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/autofeedSettings", nil, "autofeedSettings")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/autofeedSettings?updateMask=enabled", []byte(`{"autofeedSettings":{"name":"accounts/123456/autofeedSettings"}}`), "autofeedSettings")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/automaticImprovements", nil, "automaticImprovements")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/automaticImprovements?updateMask=itemUpdates", []byte(`{"automaticImprovements":{"name":"accounts/123456/automaticImprovements"}}`), "automaticImprovements")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/businessIdentity", nil, "businessIdentity")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/businessIdentity?updateMask=smallBusiness", []byte(`{"businessIdentity":{"name":"accounts/123456/businessIdentity"}}`), "businessIdentity")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/businessInfo", nil, "businessInfo")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/businessInfo?updateMask=phoneNumber", []byte(`{"businessInfo":{"name":"accounts/123456/businessInfo"}}`), "businessInfo")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/checkoutSettings", []byte(`{"parent":"accounts/123456"}`), "checkoutSettings")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/checkoutSettings", nil, "checkoutSettings")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/checkoutSettings?updateMask=effectiveUri", []byte(`{"checkoutSettings":{"name":"accounts/123456/checkoutSettings"}}`), "checkoutSettings")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodDelete, "/gcp/accounts/v1/"+account+"/checkoutSettings", nil, "{}")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/developerRegistration:registerGcp", []byte(`{"name":"accounts/123456/developerRegistration"}`), "developerRegistration")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/developerRegistration", nil, "developerRegistration")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/developerRegistration:unregisterGcp", []byte(`{"name":"accounts/123456/developerRegistration"}`), "{}")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/accounts:getAccountForGcpRegistration?gcpIds=projects/stackyard", nil, "accountId")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/emailPreferences", nil, "emailPreferences")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/emailPreferences?updateMask=newsAndTips", []byte(`{"emailPreferences":{"name":"accounts/123456/emailPreferences"}}`), "emailPreferences")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/gbpAccounts?pageSize=1", nil, "gbpAccounts")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/gbpAccounts:linkGbpAccount", []byte(`{"parent":"accounts/123456","gbpEmail":"owner@example.com"}`), "{}")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/homepage", nil, "homepage")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/homepage?updateMask=uri", []byte(`{"homepage":{"name":"accounts/123456/homepage"}}`), "homepage")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/homepage:claim", []byte(`{"name":"accounts/123456/homepage"}`), "claimed")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/homepage:unclaim", []byte(`{"name":"accounts/123456/homepage"}`), "claimed")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/lfpProviders:find?pageSize=1", nil, "lfpProviders")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+":linkLfpProvider", []byte(`{"name":"accounts/123456/omnichannelSettings/default/lfpProviders/provider-1","externalAccountId":"store-123"}`), "{}")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/omnichannelSettings", []byte(`{"parent":"accounts/123456","omnichannelSettingId":"default"}`), "omnichannelSettings/default")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/omnichannelSettings?pageSize=1", nil, "omnichannelSettings")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/omnichannelSettings/default", nil, "omnichannelSettings/default")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+account+"/omnichannelSettings/default?updateMask=displayName", []byte(`{"omnichannelSetting":{"name":"accounts/123456/omnichannelSettings/default"}}`), "omnichannelSettings/default")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/omnichannelSettings/default:requestInventoryVerification", []byte(`{"name":"accounts/123456/omnichannelSettings/default"}`), "omnichannelSettings/default")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/onlineReturnPolicies", []byte(`{"parent":"accounts/123456","onlineReturnPolicyId":"default-policy"}`), "onlineReturnPolicies/default-policy")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/onlineReturnPolicies?pageSize=1", nil, "onlineReturnPolicies")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/onlineReturnPolicies/default-policy", nil, "default-policy")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodDelete, "/gcp/accounts/v1/"+account+"/onlineReturnPolicies/default-policy", nil, "{}")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/programs?pageSize=1", nil, "programs")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+program, nil, program)
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+program+":enable", []byte(`{"name":"accounts/123456/programs/free-listings"}`), "ENABLED")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+program+":disable", []byte(`{"name":"accounts/123456/programs/free-listings"}`), "DISABLED")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/regions", []byte(`{"parent":"accounts/123456","regionId":"us-east","region":{"displayName":"US East"}}`), "regions/us-east")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/regions?pageSize=1", nil, "regions")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+region, nil, region)
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPatch, "/gcp/accounts/v1/"+region+"?updateMask=displayName", []byte(`{"region":{"name":"accounts/123456/regions/us-east","displayName":"US East Updated"}}`), "regions/us-east")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/regions:batchCreate", []byte(`{"parent":"accounts/123456","requests":[{"regionId":"us-north","region":{"displayName":"US North"}}]}`), "responses")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/regions:batchUpdate", []byte(`{"parent":"accounts/123456","requests":[{"region":{"name":"accounts/123456/regions/us-east","displayName":"US East"},"updateMask":"displayName"}]}`), "responses")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/regions:batchDelete", []byte(`{"parent":"accounts/123456","names":["accounts/123456/regions/us-east"]}`), "{}")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodDelete, "/gcp/accounts/v1/"+region, nil, "{}")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+account+"/shippingSettings:insert", []byte(`{"parent":"accounts/123456"}`), "shippingSettings")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/shippingSettings", nil, "shippingSettings")

	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+tos, nil, tos)
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/termsOfService:retrieveLatest?kind=MERCHANT_CENTER&regionCode=US", nil, "termsOfService/latest")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodPost, "/gcp/accounts/v1/"+tos+":accept", []byte(`{"name":"termsOfService/latest","account":"accounts/123456","regionCode":"US"}`), "accepted")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/termsOfServiceAgreementStates/state-1", nil, "state-1")
	assertGCPShoppingMerchantAccountsSuccess(t, ts, http.MethodGet, "/gcp/accounts/v1/"+account+"/termsOfServiceAgreementStates:retrieveForApplication", nil, "requiredNotices")
}

func TestGCPShoppingMerchantAccountsRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantAccountsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/accounts?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant accounts list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantAccountsRouter_PatchRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantAccountsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/accounts/v1/accounts/123456", []byte(`{"account":{"name":"accounts/123456"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant accounts patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantAccountsRouter_CreateUserRejectsDuplicate(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantAccountsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/accounts/v1/accounts/123456/users", []byte(`{
		"parent":"accounts/123456",
		"userId":"existing-user@example.com"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp shopping merchant accounts create user, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"AlreadyExists"`) {
		t.Fatalf("expected AlreadyExists error in response")
	}
}

func TestGCPShoppingMerchantAccountsRouter_DeleteRegionMissingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantAccountsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/accounts/v1/accounts/123456/regions/missing-region", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant accounts delete region, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantAccountsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantAccountsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	}

	accountResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/accounts/123456", nil, headers)
	if accountResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant accounts get account, got %d body=%s", accountResp.StatusCode, string(providerContractBody(t, accountResp)))
	}
	accountBody := providerContractJSONMap(t, accountResp)
	if _, ok := accountBody["name"].(string); !ok {
		t.Fatalf("expected account name string, got %#v", accountBody["name"])
	}
	if _, ok := accountBody["accountId"].(string); !ok {
		t.Fatalf("expected account accountId string, got %#v", accountBody["accountId"])
	}
	if _, ok := accountBody["timeZone"].(map[string]any); !ok {
		t.Fatalf("expected account timeZone object, got %#v", accountBody["timeZone"])
	}

	usersResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/accounts/123456/users?pageSize=1", nil, headers)
	if usersResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant accounts list users, got %d body=%s", usersResp.StatusCode, string(providerContractBody(t, usersResp)))
	}
	usersBody := providerContractJSONMap(t, usersResp)
	users, ok := usersBody["users"].([]any)
	if !ok || len(users) == 0 {
		t.Fatalf("expected users array, got %#v", usersBody["users"])
	}
	firstUser, ok := users[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first user object, got %#v", users[0])
	}
	if _, ok := firstUser["name"].(string); !ok {
		t.Fatalf("expected user name string, got %#v", firstUser["name"])
	}
	if _, ok := firstUser["accessRights"].([]any); !ok {
		t.Fatalf("expected user accessRights array, got %#v", firstUser["accessRights"])
	}

	programsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/accounts/123456/programs?pageSize=1", nil, headers)
	if programsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant accounts list programs, got %d body=%s", programsResp.StatusCode, string(providerContractBody(t, programsResp)))
	}
	programsBody := providerContractJSONMap(t, programsResp)
	programs, ok := programsBody["programs"].([]any)
	if !ok || len(programs) == 0 {
		t.Fatalf("expected programs array, got %#v", programsBody["programs"])
	}
	firstProgram, ok := programs[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first program object, got %#v", programs[0])
	}
	if _, ok := firstProgram["name"].(string); !ok {
		t.Fatalf("expected program name string, got %#v", firstProgram["name"])
	}
	if _, ok := firstProgram["state"].(string); !ok {
		t.Fatalf("expected program state string, got %#v", firstProgram["state"])
	}

	regionsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/accounts/123456/regions?pageSize=1", nil, headers)
	if regionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant accounts list regions, got %d body=%s", regionsResp.StatusCode, string(providerContractBody(t, regionsResp)))
	}
	regionsBody := providerContractJSONMap(t, regionsResp)
	regions, ok := regionsBody["regions"].([]any)
	if !ok || len(regions) == 0 {
		t.Fatalf("expected regions array, got %#v", regionsBody["regions"])
	}
	firstRegion, ok := regions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first region object, got %#v", regions[0])
	}
	if _, ok := firstRegion["postalCodeArea"].(map[string]any); !ok {
		t.Fatalf("expected region postalCodeArea object, got %#v", firstRegion["postalCodeArea"])
	}

	tosResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/accounts/v1/termsOfService/latest", nil, headers)
	if tosResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant accounts get terms, got %d body=%s", tosResp.StatusCode, string(providerContractBody(t, tosResp)))
	}
	tosBody := providerContractJSONMap(t, tosResp)
	if _, ok := tosBody["name"].(string); !ok {
		t.Fatalf("expected terms name string, got %#v", tosBody["name"])
	}
	if _, ok := tosBody["kind"].(string); !ok {
		t.Fatalf("expected terms kind string, got %#v", tosBody["kind"])
	}
}

func TestGCPShoppingMerchantAccountsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_accounts/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant accounts contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_accounts" {
		t.Fatalf("expected service=shopping_merchant_accounts, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantAccountsContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPShoppingMerchantAccountsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-accounts",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant accounts router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
