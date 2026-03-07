package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpShoppingMerchantAccountsIDRe      = regexp.MustCompile(`^[A-Za-z0-9._~@+-]+$`)
	gcpShoppingMerchantAccountsEmailRe   = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	gcpShoppingMerchantAccountsRegionRe  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantAccountsProgramRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func (s *Server) handleGCPShoppingMerchantAccountsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantAccounts(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantAccountsPath(rawRequestPath(r))
	if !isGCPShoppingMerchantAccountsPath(path, hasGCPShoppingMerchantAccountsHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantAccountsRESTTail(path)
	if !ok {
		// gRPC calls are handled by grpc foundation.
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.accounts.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantAccountsGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantAccountsPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPShoppingMerchantAccountsPATCH(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantAccountsDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantAccountsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantAccountsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_accounts",
		"shopping-merchant-accounts",
		"shopping-merchant-accounts-apiv1",
		"shopping_merchant_accounts_apiv1",
		"merchant_accounts",
		"merchant-accounts",
		"merchantaccounts",
		"gcp-shopping-merchant-accounts":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-accounts-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/accounts")
}

func isGCPShoppingMerchantAccountsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/accounts/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.accounts.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/accounts/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantAccountsRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/accounts/v1/") {
		return "", false
	}
	tail := strings.TrimPrefix(path, "/gcp/accounts/v1/")
	tail = strings.TrimSpace(tail)
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantAccountsGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if tail == "accounts" {
		return handleGCPShoppingMerchantAccountsListAccounts(w, r, path)
	}
	if tail == "accounts:getAccountForGcpRegistration" {
		return handleGCPShoppingMerchantAccountsGetAccountForGCPRegistration(w, r, path)
	}
	if tail == "termsOfService:retrieveLatest" {
		return handleGCPShoppingMerchantAccountsRetrieveLatestTermsOfService(w, r, path)
	}
	if strings.HasPrefix(tail, "termsOfService/") && !strings.Contains(tail, ":") {
		return handleGCPShoppingMerchantAccountsGetTermsOfService(w, path, tail)
	}

	account, suffix, ok := parseGCPShoppingMerchantAccountsAccountScope(tail)
	if !ok {
		return false
	}

	switch {
	case suffix == "":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsAccountFixture(account))
		return true
	case suffix == ":listSubaccounts":
		return handleGCPShoppingMerchantAccountsListSubAccounts(w, r, path, account)
	case suffix == "/issues":
		return handleGCPShoppingMerchantAccountsListAccountIssues(w, r, path, account)
	case suffix == "/relationships":
		return handleGCPShoppingMerchantAccountsListRelationships(w, r, path, account)
	case strings.HasPrefix(suffix, "/relationships/") && !strings.Contains(suffix, ":"):
		relID := strings.TrimPrefix(suffix, "/relationships/")
		if !isGCPShoppingMerchantAccountsID(relID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "relationship name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsRelationshipFixture(account, relID, "providers/123"))
		return true
	case suffix == "/services":
		return handleGCPShoppingMerchantAccountsListServices(w, r, path, account)
	case strings.HasPrefix(suffix, "/services/") && !strings.Contains(suffix, ":"):
		serviceID := strings.TrimPrefix(suffix, "/services/")
		if !isGCPShoppingMerchantAccountsID(serviceID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "service name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsServiceFixture(account, serviceID, "ACTIVE"))
		return true
	case suffix == "/autofeedSettings":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsAutofeedSettingsFixture(account))
		return true
	case suffix == "/automaticImprovements":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsAutomaticImprovementsFixture(account))
		return true
	case suffix == "/businessIdentity":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsBusinessIdentityFixture(account))
		return true
	case suffix == "/businessInfo":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsBusinessInfoFixture(account))
		return true
	case suffix == "/checkoutSettings":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsCheckoutSettingsFixture(account))
		return true
	case suffix == "/developerRegistration":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsDeveloperRegistrationFixture(account))
		return true
	case suffix == "/emailPreferences":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsEmailPreferencesFixture(account))
		return true
	case suffix == "/gbpAccounts":
		return handleGCPShoppingMerchantAccountsListGBPAccounts(w, r, path, account)
	case suffix == "/homepage":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsHomepageFixture(account, true))
		return true
	case suffix == "/lfpProviders:find":
		return handleGCPShoppingMerchantAccountsFindLFPProviders(w, r, path, account)
	case suffix == "/omnichannelSettings":
		return handleGCPShoppingMerchantAccountsListOmnichannelSettings(w, r, path, account)
	case strings.HasPrefix(suffix, "/omnichannelSettings/") && !strings.Contains(suffix, ":"):
		settingID := strings.TrimPrefix(suffix, "/omnichannelSettings/")
		if !isGCPShoppingMerchantAccountsID(settingID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "omnichannel setting is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsOmnichannelSettingFixture(account, settingID))
		return true
	case suffix == "/onlineReturnPolicies":
		return handleGCPShoppingMerchantAccountsListOnlineReturnPolicies(w, r, path, account)
	case strings.HasPrefix(suffix, "/onlineReturnPolicies/") && !strings.Contains(suffix, ":"):
		policyID := strings.TrimPrefix(suffix, "/onlineReturnPolicies/")
		if !isGCPShoppingMerchantAccountsID(policyID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "online return policy name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsOnlineReturnPolicyFixture(account, policyID))
		return true
	case suffix == "/programs":
		return handleGCPShoppingMerchantAccountsListPrograms(w, r, path, account)
	case strings.HasPrefix(suffix, "/programs/") && !strings.Contains(suffix, ":"):
		programID := strings.TrimPrefix(suffix, "/programs/")
		if !gcpShoppingMerchantAccountsProgramRe.MatchString(programID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "program name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsProgramFixture(account, programID, "ENABLED"))
		return true
	case suffix == "/regions":
		return handleGCPShoppingMerchantAccountsListRegions(w, r, path, account)
	case strings.HasPrefix(suffix, "/regions/") && !strings.Contains(suffix, ":"):
		regionID := strings.TrimPrefix(suffix, "/regions/")
		if !gcpShoppingMerchantAccountsRegionRe.MatchString(regionID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "region name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsRegionFixture(account, regionID, "Stackyard Region "+regionID))
		return true
	case suffix == "/shippingSettings":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsShippingSettingsFixture(account))
		return true
	case strings.HasPrefix(suffix, "/termsOfServiceAgreementStates/") && !strings.Contains(suffix, ":"):
		stateID := strings.TrimPrefix(suffix, "/termsOfServiceAgreementStates/")
		if !isGCPShoppingMerchantAccountsID(stateID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "terms of service agreement state name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsTermsOfServiceAgreementStateFixture(account, stateID))
		return true
	case suffix == "/termsOfServiceAgreementStates:retrieveForApplication":
		respondJSON(w, http.StatusOK, map[string]any{
			"name":            fmt.Sprintf("accounts/%s/termsOfServiceAgreementStates/application", account),
			"account":         "accounts/" + account,
			"termsOfService":  "termsOfService/latest",
			"regionCode":      "US",
			"accepted":        true,
			"requiredNotices": []any{"STACKYARD_NOTICE"},
		})
		return true
	case suffix == "/users":
		return handleGCPShoppingMerchantAccountsListUsers(w, r, path, account)
	case strings.HasPrefix(suffix, "/users/") && !strings.Contains(suffix, ":"):
		userID := strings.TrimPrefix(suffix, "/users/")
		if !isGCPShoppingMerchantAccountsUserID(userID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "user name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsUserFixture(account, userID))
		return true
	default:
		return false
	}
}

func handleGCPShoppingMerchantAccountsPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if tail == "accounts:createAndConfigure" {
		body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
		if !ok {
			return true
		}
		accountBody := gcpShoppingMerchantAccountsMap(body, "account")
		accountName := strings.TrimSpace(gcpShoppingMerchantAccountsString(accountBody, "name"))
		accountID := strings.TrimPrefix(accountName, "accounts/")
		if accountName == "" || accountID == "" || !isGCPShoppingMerchantAccountsID(accountID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "account.name is required")
			return true
		}
		if rawServices, ok := body["service"].([]any); !ok || len(rawServices) == 0 {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "service is required")
			return true
		}
		if strings.Contains(strings.ToLower(accountName), "existing") {
			respondGCPShoppingMerchantAccountsAlreadyExists(w, path, "account already exists")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsAccountFixture(accountID))
		return true
	}
	if tail == "accounts:getAccountForGcpRegistration" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "accounts/123456",
			"accountId": "123456",
		})
		return true
	}
	if strings.HasPrefix(tail, "termsOfService/") && strings.HasSuffix(tail, ":accept") {
		return handleGCPShoppingMerchantAccountsAcceptTermsOfService(w, r, path, tail)
	}

	account, suffix, ok := parseGCPShoppingMerchantAccountsAccountScope(tail)
	if !ok {
		return false
	}

	switch {
	case suffix == "/users":
		return handleGCPShoppingMerchantAccountsCreateUser(w, r, path, account)
	case suffix == "/services:propose":
		return handleGCPShoppingMerchantAccountsProposeService(w, r, path, account)
	case strings.HasPrefix(suffix, "/services/") && strings.HasSuffix(suffix, ":approve"):
		serviceID := strings.TrimSuffix(strings.TrimPrefix(suffix, "/services/"), ":approve")
		return handleGCPShoppingMerchantAccountsServiceStateAction(w, r, path, account, serviceID, "APPROVED")
	case strings.HasPrefix(suffix, "/services/") && strings.HasSuffix(suffix, ":reject"):
		serviceID := strings.TrimSuffix(strings.TrimPrefix(suffix, "/services/"), ":reject")
		return handleGCPShoppingMerchantAccountsServiceStateAction(w, r, path, account, serviceID, "REJECTED")
	case suffix == "/developerRegistration:registerGcp":
		return handleGCPShoppingMerchantAccountsRegisterGCP(w, r, path, account)
	case suffix == "/developerRegistration:unregisterGcp":
		return handleGCPShoppingMerchantAccountsUnregisterGCP(w, r, path, account)
	case suffix == "/gbpAccounts:linkGbpAccount":
		return handleGCPShoppingMerchantAccountsLinkGBPAccount(w, r, path, account)
	case suffix == "/homepage:claim":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsHomepageFixture(account, true))
		return true
	case suffix == "/homepage:unclaim":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsHomepageFixture(account, false))
		return true
	case suffix == ":linkLfpProvider":
		return handleGCPShoppingMerchantAccountsLinkLFPProvider(w, r, path, account)
	case suffix == "/omnichannelSettings":
		return handleGCPShoppingMerchantAccountsCreateOmnichannelSetting(w, r, path, account)
	case strings.HasPrefix(suffix, "/omnichannelSettings/") && strings.HasSuffix(suffix, ":requestInventoryVerification"):
		settingID := strings.TrimSuffix(strings.TrimPrefix(suffix, "/omnichannelSettings/"), ":requestInventoryVerification")
		if !isGCPShoppingMerchantAccountsID(settingID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "omnichannel setting name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsOmnichannelSettingFixture(account, settingID))
		return true
	case suffix == "/onlineReturnPolicies":
		return handleGCPShoppingMerchantAccountsCreateOnlineReturnPolicy(w, r, path, account)
	case strings.HasPrefix(suffix, "/programs/") && strings.HasSuffix(suffix, ":enable"):
		programID := strings.TrimSuffix(strings.TrimPrefix(suffix, "/programs/"), ":enable")
		if !gcpShoppingMerchantAccountsProgramRe.MatchString(programID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "program name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsProgramFixture(account, programID, "ENABLED"))
		return true
	case strings.HasPrefix(suffix, "/programs/") && strings.HasSuffix(suffix, ":disable"):
		programID := strings.TrimSuffix(strings.TrimPrefix(suffix, "/programs/"), ":disable")
		if !gcpShoppingMerchantAccountsProgramRe.MatchString(programID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "program name is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsProgramFixture(account, programID, "DISABLED"))
		return true
	case suffix == "/regions":
		return handleGCPShoppingMerchantAccountsCreateRegion(w, r, path, account)
	case suffix == "/regions:batchCreate":
		return handleGCPShoppingMerchantAccountsBatchCreateRegions(w, r, path, account)
	case suffix == "/regions:batchUpdate":
		return handleGCPShoppingMerchantAccountsBatchUpdateRegions(w, r, path, account)
	case suffix == "/regions:batchDelete":
		return handleGCPShoppingMerchantAccountsBatchDeleteRegions(w, r, path, account)
	case suffix == "/shippingSettings:insert":
		return handleGCPShoppingMerchantAccountsInsertShippingSettings(w, r, path, account)
	case suffix == "/checkoutSettings":
		return handleGCPShoppingMerchantAccountsCreateCheckoutSettings(w, r, path, account)
	default:
		return false
	}
}

func handleGCPShoppingMerchantAccountsPATCH(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, suffix, ok := parseGCPShoppingMerchantAccountsAccountScope(tail)
	if !ok {
		return false
	}

	switch {
	case suffix == "/users/me:verifySelf":
		body, ok := decodeGCPShoppingMerchantAccountsJSONBody(w, r, path, false)
		if !ok {
			return true
		}
		if got := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "account")); got != "" && got != "accounts/"+account {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "account must match requested resource")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsUserFixture(account, "me"))
		return true
	case suffix == "":
		if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "updateMask is required")
			return true
		}
		body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
		if !ok {
			return true
		}
		accountBody := gcpShoppingMerchantAccountsMap(body, "account")
		if len(accountBody) == 0 {
			accountBody = body
		}
		expected := "accounts/" + account
		if name := strings.TrimSpace(gcpShoppingMerchantAccountsString(accountBody, "name")); name == "" || name != expected {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "account.name must match requested resource")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsAccountFixture(account))
		return true
	case strings.HasPrefix(suffix, "/users/") && suffix != "/users/me:verifySelf":
		if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "updateMask is required")
			return true
		}
		body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
		if !ok {
			return true
		}
		userBody := gcpShoppingMerchantAccountsMap(body, "user")
		if len(userBody) == 0 {
			userBody = body
		}
		expected := "accounts/" + account + strings.TrimPrefix(strings.TrimSuffix(suffix, ":verifySelf"), "")
		name := strings.TrimSpace(gcpShoppingMerchantAccountsString(userBody, "name"))
		if name == "" || name != expected {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "user.name must match requested resource")
			return true
		}
		userID := strings.TrimPrefix(suffix, "/users/")
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsUserFixture(account, userID))
		return true
	case strings.HasPrefix(suffix, "/relationships/"):
		relID := strings.TrimPrefix(suffix, "/relationships/")
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsRelationshipFixture(account, relID, "providers/123"))
		return true
	case suffix == "/autofeedSettings":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsAutofeedSettingsFixture(account))
		return true
	case suffix == "/automaticImprovements":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsAutomaticImprovementsFixture(account))
		return true
	case suffix == "/businessIdentity":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsBusinessIdentityFixture(account))
		return true
	case suffix == "/businessInfo":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsBusinessInfoFixture(account))
		return true
	case suffix == "/checkoutSettings":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsCheckoutSettingsFixture(account))
		return true
	case suffix == "/emailPreferences":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsEmailPreferencesFixture(account))
		return true
	case suffix == "/homepage":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsHomepageFixture(account, true))
		return true
	case strings.HasPrefix(suffix, "/omnichannelSettings/"):
		settingID := strings.TrimPrefix(suffix, "/omnichannelSettings/")
		if !isGCPShoppingMerchantAccountsID(settingID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "omnichannel setting is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsOmnichannelSettingFixture(account, settingID))
		return true
	case strings.HasPrefix(suffix, "/regions/"):
		regionID := strings.TrimPrefix(suffix, "/regions/")
		if !gcpShoppingMerchantAccountsRegionRe.MatchString(regionID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "region is invalid")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsRegionFixture(account, regionID, "Stackyard Region "+regionID))
		return true
	default:
		return false
	}
}

func handleGCPShoppingMerchantAccountsDELETE(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, suffix, ok := parseGCPShoppingMerchantAccountsAccountScope(tail)
	if !ok {
		return false
	}
	_ = account

	switch {
	case suffix == "":
		if strings.Contains(tail, "missing") {
			respondGCPShoppingMerchantAccountsNotFound(w, path, "account not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case strings.HasPrefix(suffix, "/users/"):
		userID := strings.TrimPrefix(suffix, "/users/")
		if strings.Contains(strings.ToLower(userID), "missing") {
			respondGCPShoppingMerchantAccountsNotFound(w, path, "user not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case suffix == "/checkoutSettings":
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case strings.HasPrefix(suffix, "/onlineReturnPolicies/"):
		policyID := strings.TrimPrefix(suffix, "/onlineReturnPolicies/")
		if strings.Contains(strings.ToLower(policyID), "missing") {
			respondGCPShoppingMerchantAccountsNotFound(w, path, "online return policy not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case strings.HasPrefix(suffix, "/regions/"):
		regionID := strings.TrimPrefix(suffix, "/regions/")
		if strings.Contains(strings.ToLower(regionID), "missing") {
			respondGCPShoppingMerchantAccountsNotFound(w, path, "region not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	default:
		return false
	}
}

func parseGCPShoppingMerchantAccountsAccountScope(tail string) (account, suffix string, ok bool) {
	if !strings.HasPrefix(tail, "accounts/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(tail, "accounts/")
	if rest == "" {
		return "", "", false
	}
	idxSlash := strings.Index(rest, "/")
	idxColon := strings.Index(rest, ":")
	idx := -1
	switch {
	case idxSlash == -1 && idxColon == -1:
		idx = -1
	case idxSlash == -1:
		idx = idxColon
	case idxColon == -1:
		idx = idxSlash
	case idxSlash < idxColon:
		idx = idxSlash
	default:
		idx = idxColon
	}
	if idx == -1 {
		account = rest
		suffix = ""
	} else {
		account = rest[:idx]
		suffix = rest[idx:]
	}
	if !isGCPShoppingMerchantAccountsID(account) {
		return "", "", false
	}
	return account, suffix, true
}

func handleGCPShoppingMerchantAccountsListAccounts(w http.ResponseWriter, r *http.Request, path string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 250)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsAccountFixture("123456"),
		gcpShoppingMerchantAccountsAccountFixture("123457"),
	}
	if !respondGCPShoppingMerchantAccountsList(w, "accounts", items, pageSize, start, path) {
		return true
	}
	return true
}

func handleGCPShoppingMerchantAccountsListSubAccounts(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 250)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsAccountFixture(account + "-sub-1"),
		gcpShoppingMerchantAccountsAccountFixture(account + "-sub-2"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "accounts", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsListUsers(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 200)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsUserFixture(account, "owner@example.com"),
		gcpShoppingMerchantAccountsUserFixture(account, "analyst@example.com"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "users", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsCreateUser(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(r.URL.Query().Get("parent"))
	}
	if parent != "" && parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	userID := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "userId", "user_id"))
	if userID == "" {
		userID = strings.TrimSpace(r.URL.Query().Get("userId"))
	}
	if userID == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "userId is required")
		return true
	}
	if strings.Contains(strings.ToLower(userID), "existing") {
		respondGCPShoppingMerchantAccountsAlreadyExists(w, path, "user already exists")
		return true
	}
	if !isGCPShoppingMerchantAccountsUserID(userID) {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "userId is invalid")
		return true
	}
	userBody := gcpShoppingMerchantAccountsMap(body, "user")
	if len(userBody) == 0 {
		userBody = body
	}
	if name := strings.TrimSpace(gcpShoppingMerchantAccountsString(userBody, "name")); name != "" {
		expectedName := fmt.Sprintf("accounts/%s/users/%s", account, userID)
		if name != expectedName {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "user.name must match requested resource")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsUserFixture(account, userID))
	return true
}

func handleGCPShoppingMerchantAccountsListAccountIssues(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 250)
	if !ok {
		return true
	}
	items := []map[string]any{
		{
			"name":             fmt.Sprintf("accounts/%s/issues/issue-1", account),
			"title":            "Missing shipping attributes",
			"severity":         "ERROR",
			"detail":           "A product is missing shipping dimensions",
			"impact":           "ACCOUNT_LEVEL",
			"documentationUri": "https://support.google.com/merchants/answer/123",
		},
	}
	return respondGCPShoppingMerchantAccountsList(w, "accountIssues", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsListRelationships(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 200)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsRelationshipFixture(account, "relationship-1", "providers/123"),
		gcpShoppingMerchantAccountsRelationshipFixture(account, "relationship-2", "providers/GOOGLE_ADS"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "accountRelationships", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsListServices(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 200)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsServiceFixture(account, "shopping-ads", "ACTIVE"),
		gcpShoppingMerchantAccountsServiceFixture(account, "free-listings", "ACTIVE"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "accountServices", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsProposeService(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	if parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent")); parent == "" || parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	provider := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "provider"))
	if provider == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "provider is required")
		return true
	}
	accountService := gcpShoppingMerchantAccountsMap(body, "accountService", "account_service")
	name := strings.TrimSpace(gcpShoppingMerchantAccountsString(accountService, "name"))
	serviceID := "proposed-service"
	if name != "" {
		serviceID = strings.TrimPrefix(name, "accounts/"+account+"/services/")
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsServiceFixture(account, serviceID, "PENDING"))
	return true
}

func handleGCPShoppingMerchantAccountsServiceStateAction(w http.ResponseWriter, r *http.Request, path, account, serviceID, state string) bool {
	if !isGCPShoppingMerchantAccountsID(serviceID) {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "service name is invalid")
		return true
	}
	body, ok := decodeGCPShoppingMerchantAccountsJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	if len(body) > 0 {
		if name := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "name")); name != "" {
			expected := fmt.Sprintf("accounts/%s/services/%s", account, serviceID)
			if name != expected {
				respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "name must match requested resource")
				return true
			}
		}
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsServiceFixture(account, serviceID, state))
	return true
}

func handleGCPShoppingMerchantAccountsRegisterGCP(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expected := fmt.Sprintf("accounts/%s/developerRegistration", account)
	if name := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "name")); name != "" && name != expected {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsDeveloperRegistrationFixture(account))
	return true
}

func handleGCPShoppingMerchantAccountsUnregisterGCP(w http.ResponseWriter, r *http.Request, path, account string) bool {
	_, ok := decodeGCPShoppingMerchantAccountsJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantAccountsGetAccountForGCPRegistration(w http.ResponseWriter, r *http.Request, path string) bool {
	if strings.TrimSpace(r.URL.Query().Get("gcpIds")) == "" && strings.TrimSpace(r.URL.Query().Get("gcp_id")) == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "gcpIds is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":      "accounts/123456",
		"accountId": "123456",
	})
	return true
}

func handleGCPShoppingMerchantAccountsLinkGBPAccount(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	if parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent")); parent == "" || parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	if email := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "gbpEmail", "gbp_email")); !gcpShoppingMerchantAccountsEmailRe.MatchString(email) {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "gbpEmail is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantAccountsListGBPAccounts(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 200)
	if !ok {
		return true
	}
	items := []map[string]any{
		{"name": fmt.Sprintf("accounts/%s/gbpAccounts/gbp-1", account), "gbpEmail": "owner@example.com", "type": "OWNER"},
	}
	return respondGCPShoppingMerchantAccountsList(w, "gbpAccounts", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsFindLFPProviders(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		{"name": fmt.Sprintf("accounts/%s/omnichannelSettings/default/lfpProviders/provider-1", account), "displayName": "Stackyard LFP"},
	}
	return respondGCPShoppingMerchantAccountsList(w, "lfpProviders", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsLinkLFPProvider(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	name := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "name"))
	if name == "" || !strings.HasPrefix(name, "accounts/"+account+"/") {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "name is required and must match requested account")
		return true
	}
	if strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "externalAccountId", "external_account_id")) == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "externalAccountId is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantAccountsListOmnichannelSettings(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsOmnichannelSettingFixture(account, "default"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "omnichannelSettings", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsCreateOmnichannelSetting(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	if parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent")); parent == "" || parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	settingID := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "omnichannelSettingId", "omnichannel_setting_id"))
	if settingID == "" {
		settingID = "default"
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsOmnichannelSettingFixture(account, settingID))
	return true
}

func handleGCPShoppingMerchantAccountsListOnlineReturnPolicies(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsOnlineReturnPolicyFixture(account, "default-policy"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "onlineReturnPolicies", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsCreateOnlineReturnPolicy(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	if parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent")); parent == "" || parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	policyID := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "onlineReturnPolicyId", "online_return_policy_id"))
	if policyID == "" {
		policyID = "default-policy"
	}
	if strings.Contains(strings.ToLower(policyID), "existing") {
		respondGCPShoppingMerchantAccountsAlreadyExists(w, path, "online return policy already exists")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsOnlineReturnPolicyFixture(account, policyID))
	return true
}

func handleGCPShoppingMerchantAccountsListPrograms(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsProgramFixture(account, "free-listings", "ENABLED"),
		gcpShoppingMerchantAccountsProgramFixture(account, "shopping-ads", "DISABLED"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "programs", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsListRegions(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantAccountsPagination(w, r, path, 200)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantAccountsRegionFixture(account, "us-east", "US East"),
		gcpShoppingMerchantAccountsRegionFixture(account, "us-west", "US West"),
	}
	return respondGCPShoppingMerchantAccountsList(w, "regions", items, pageSize, start, path)
}

func handleGCPShoppingMerchantAccountsCreateRegion(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(r.URL.Query().Get("parent"))
	}
	if parent != "" && parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	regionID := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "regionId", "region_id"))
	if regionID == "" {
		regionID = strings.TrimSpace(r.URL.Query().Get("regionId"))
	}
	if regionID == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "regionId is required")
		return true
	}
	if !gcpShoppingMerchantAccountsRegionRe.MatchString(regionID) {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "regionId is invalid")
		return true
	}
	region := gcpShoppingMerchantAccountsMap(body, "region")
	if len(region) == 0 {
		region = body
	}
	displayName := strings.TrimSpace(gcpShoppingMerchantAccountsString(region, "displayName", "display_name"))
	if displayName == "" {
		displayName = "Stackyard Region " + regionID
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsRegionFixture(account, regionID, displayName))
	return true
}

func handleGCPShoppingMerchantAccountsBatchCreateRegions(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(r.URL.Query().Get("parent"))
	}
	if parent != "" && parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	rawRequests, ok := body["requests"].([]any)
	if !ok || len(rawRequests) == 0 {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "requests is required")
		return true
	}
	responses := make([]any, 0, len(rawRequests))
	for idx, raw := range rawRequests {
		item, ok := raw.(map[string]any)
		if !ok {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, fmt.Sprintf("requests[%d] must be an object", idx))
			return true
		}
		regionID := strings.TrimSpace(gcpShoppingMerchantAccountsString(item, "regionId", "region_id"))
		if regionID == "" {
			regionID = fmt.Sprintf("region-%d", idx+1)
		}
		responses = append(responses, map[string]any{
			"region": gcpShoppingMerchantAccountsRegionFixture(account, regionID, "Stackyard Region "+regionID),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{"responses": responses})
	return true
}

func handleGCPShoppingMerchantAccountsBatchUpdateRegions(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(r.URL.Query().Get("parent"))
	}
	if parent != "" && parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	rawRequests, ok := body["requests"].([]any)
	if !ok || len(rawRequests) == 0 {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "requests is required")
		return true
	}
	responses := make([]any, 0, len(rawRequests))
	for idx, raw := range rawRequests {
		item, ok := raw.(map[string]any)
		if !ok {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, fmt.Sprintf("requests[%d] must be an object", idx))
			return true
		}
		region := gcpShoppingMerchantAccountsMap(item, "region")
		name := strings.TrimSpace(gcpShoppingMerchantAccountsString(region, "name"))
		if name == "" {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, fmt.Sprintf("requests[%d].region.name is required", idx))
			return true
		}
		regionID := strings.TrimPrefix(name, "accounts/"+account+"/regions/")
		if !gcpShoppingMerchantAccountsRegionRe.MatchString(regionID) {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, fmt.Sprintf("requests[%d].region.name is invalid", idx))
			return true
		}
		displayName := strings.TrimSpace(gcpShoppingMerchantAccountsString(region, "displayName", "display_name"))
		if displayName == "" {
			displayName = "Stackyard Region " + regionID
		}
		responses = append(responses, map[string]any{
			"region": gcpShoppingMerchantAccountsRegionFixture(account, regionID, displayName),
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{"responses": responses})
	return true
}

func handleGCPShoppingMerchantAccountsBatchDeleteRegions(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent"))
	if parent == "" {
		parent = strings.TrimSpace(r.URL.Query().Get("parent"))
	}
	if parent != "" && parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	rawNames, ok := body["names"].([]any)
	if !ok || len(rawNames) == 0 {
		rawRequests, reqOK := body["requests"].([]any)
		if !reqOK || len(rawRequests) == 0 {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "requests is required")
			return true
		}
	}
	if rawNames != nil && len(rawNames) > 0 {
		for idx, rawName := range rawNames {
			name, ok := rawName.(string)
			if !ok || strings.TrimSpace(name) == "" {
				respondGCPShoppingMerchantAccountsInvalidArgument(w, path, fmt.Sprintf("names[%d] must be a non-empty string", idx))
				return true
			}
		}
	}
	if rawRequests, ok := body["requests"].([]any); ok && len(rawRequests) > 0 {
		for idx, rawRequest := range rawRequests {
			reqMap, ok := rawRequest.(map[string]any)
			if !ok {
				respondGCPShoppingMerchantAccountsInvalidArgument(w, path, fmt.Sprintf("requests[%d] must be an object", idx))
				return true
			}
			if name := strings.TrimSpace(gcpShoppingMerchantAccountsString(reqMap, "name")); name == "" {
				respondGCPShoppingMerchantAccountsInvalidArgument(w, path, fmt.Sprintf("requests[%d].name is required", idx))
				return true
			}
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantAccountsInsertShippingSettings(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	if parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent")); parent == "" || parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsShippingSettingsFixture(account))
	return true
}

func handleGCPShoppingMerchantAccountsCreateCheckoutSettings(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedParent := "accounts/" + account
	if parent := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "parent")); parent == "" || parent != expectedParent {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "parent must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantAccountsCheckoutSettingsFixture(account))
	return true
}

func handleGCPShoppingMerchantAccountsRetrieveLatestTermsOfService(w http.ResponseWriter, r *http.Request, path string) bool {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	regionCode := strings.TrimSpace(r.URL.Query().Get("regionCode"))
	if kind == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "kind is required")
		return true
	}
	if regionCode == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "regionCode is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":          "termsOfService/latest",
		"kind":          kind,
		"regionCode":    strings.ToUpper(regionCode),
		"fileUri":       "https://merchant.stackyard.example/tos/latest",
		"external":      false,
		"version":       "latest",
		"effectiveTime": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
	})
	return true
}

func handleGCPShoppingMerchantAccountsGetTermsOfService(w http.ResponseWriter, path, tail string) bool {
	version := strings.TrimPrefix(tail, "termsOfService/")
	if !isGCPShoppingMerchantAccountsID(version) {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "terms of service name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":       "termsOfService/" + version,
		"kind":       "MERCHANT_CENTER",
		"regionCode": "US",
		"version":    version,
		"fileUri":    "https://merchant.stackyard.example/tos/" + version,
	})
	return true
}

func handleGCPShoppingMerchantAccountsAcceptTermsOfService(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	name := strings.TrimSuffix(tail, ":accept")
	if !strings.HasPrefix(name, "termsOfService/") {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "name must be termsOfService/{version}")
		return true
	}
	body, ok := decodeGCPShoppingMerchantAccountsJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	if got := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "name")); got != "" && got != name {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	account := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "account"))
	if account == "" {
		account = strings.TrimSpace(r.URL.Query().Get("account"))
	}
	if !strings.HasPrefix(account, "accounts/") {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "account is required")
		return true
	}
	regionCode := strings.TrimSpace(gcpShoppingMerchantAccountsString(body, "regionCode", "region_code"))
	if regionCode == "" {
		regionCode = strings.TrimSpace(r.URL.Query().Get("regionCode"))
	}
	if regionCode == "" {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "regionCode is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"accepted": true,
	})
	return true
}

func parseGCPShoppingMerchantAccountsPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseGCPShoppingMerchantAccountsOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > maxPageSize {
		respondGCPShoppingMerchantAccountsOutOfRange(w, path, fmt.Sprintf("pageSize cannot exceed %d", maxPageSize))
		return 0, 0, false
	}
	start, err = parseGCPShoppingMerchantAccountsOptionalNonNegativeInt(r.URL.Query().Get("pageToken"))
	if err != nil {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPShoppingMerchantAccountsList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start < 0 || start > len(items) {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}
	values := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		values = append(values, item)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           values,
		"nextPageToken": nextToken,
	})
	return true
}

func decodeGCPShoppingMerchantAccountsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPShoppingMerchantAccountsJSONBody(w, r, path, true)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func decodeGCPShoppingMerchantAccountsJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	if r.Body == nil {
		if required {
			respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.UseNumber()

	body := map[string]any{}
	if err := decoder.Decode(&body); err != nil {
		if err == io.EOF && !required {
			return map[string]any{}, true
		}
		respondGCPShoppingMerchantAccountsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpShoppingMerchantAccountsString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if m == nil {
			continue
		}
		if raw, ok := m[key]; ok {
			switch value := raw.(type) {
			case string:
				if strings.TrimSpace(value) != "" {
					return value
				}
			case fmt.Stringer:
				if strings.TrimSpace(value.String()) != "" {
					return value.String()
				}
			}
		}
	}
	return ""
}

func gcpShoppingMerchantAccountsMap(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			if child, ok := raw.(map[string]any); ok {
				return child
			}
		}
	}
	return map[string]any{}
}

func parseGCPShoppingMerchantAccountsOptionalNonNegativeInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, fmt.Errorf("must be non-negative")
	}
	return parsed, nil
}

func isGCPShoppingMerchantAccountsID(value string) bool {
	return gcpShoppingMerchantAccountsIDRe.MatchString(strings.TrimSpace(value))
}

func isGCPShoppingMerchantAccountsUserID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "me" {
		return true
	}
	if gcpShoppingMerchantAccountsEmailRe.MatchString(value) {
		return true
	}
	return gcpShoppingMerchantAccountsIDRe.MatchString(value)
}

func gcpShoppingMerchantAccountsAccountFixture(account string) map[string]any {
	accountID := gcpShoppingMerchantAccountsNumericID(account, 123456)
	return map[string]any{
		"name":         "accounts/" + account,
		"accountId":    strconv.FormatInt(accountID, 10),
		"accountName":  "Stackyard Merchant " + account,
		"languageCode": "en-US",
		"timeZone": map[string]any{
			"id": "America/New_York",
		},
		"adultContent": false,
		"testAccount":  false,
	}
}

func gcpShoppingMerchantAccountsUserFixture(account, userID string) map[string]any {
	name := fmt.Sprintf("accounts/%s/users/%s", account, userID)
	return map[string]any{
		"name":         name,
		"state":        "ACTIVE",
		"accessRights": []any{"ADMIN"},
	}
}

func gcpShoppingMerchantAccountsRelationshipFixture(account, relationshipID, provider string) map[string]any {
	return map[string]any{
		"name":     fmt.Sprintf("accounts/%s/relationships/%s", account, relationshipID),
		"provider": provider,
		"state":    "ACTIVE",
		"label":    "Stackyard Relationship " + relationshipID,
	}
}

func gcpShoppingMerchantAccountsServiceFixture(account, serviceID, state string) map[string]any {
	if state == "" {
		state = "ACTIVE"
	}
	return map[string]any{
		"name":        fmt.Sprintf("accounts/%s/services/%s", account, serviceID),
		"provider":    "providers/123",
		"state":       state,
		"serviceType": "PRODUCTS_MANAGEMENT",
	}
}

func gcpShoppingMerchantAccountsAutofeedSettingsFixture(account string) map[string]any {
	return map[string]any{
		"name":    fmt.Sprintf("accounts/%s/autofeedSettings", account),
		"enabled": true,
	}
}

func gcpShoppingMerchantAccountsAutomaticImprovementsFixture(account string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("accounts/%s/automaticImprovements", account),
		"itemUpdates": map[string]any{
			"allowPriceUpdates": true,
		},
	}
}

func gcpShoppingMerchantAccountsBusinessIdentityFixture(account string) map[string]any {
	return map[string]any{
		"name":              fmt.Sprintf("accounts/%s/businessIdentity", account),
		"blackOwned":        false,
		"womenOwned":        false,
		"veteranOwned":      false,
		"smallBusiness":     true,
		"promotionsConsent": "PROMOTIONS_CONSENT_GIVEN",
	}
}

func gcpShoppingMerchantAccountsBusinessInfoFixture(account string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("accounts/%s/businessInfo", account),
		"phoneNumber":     "+1-555-0100",
		"customerService": map[string]any{"uri": "https://merchant.stackyard.example/support"},
	}
}

func gcpShoppingMerchantAccountsCheckoutSettingsFixture(account string) map[string]any {
	return map[string]any{
		"name":                    fmt.Sprintf("accounts/%s/checkoutSettings", account),
		"effectiveUri":            "https://merchant.stackyard.example/checkout",
		"checkoutEnrollmentState": "CHECKOUT_ENROLLMENT_STATE_ENROLLED",
	}
}

func gcpShoppingMerchantAccountsDeveloperRegistrationFixture(account string) map[string]any {
	return map[string]any{
		"name":     fmt.Sprintf("accounts/%s/developerRegistration", account),
		"gcpIds":   []any{"projects/stackyard"},
		"verified": true,
	}
}

func gcpShoppingMerchantAccountsEmailPreferencesFixture(account string) map[string]any {
	return map[string]any{
		"name":                   fmt.Sprintf("accounts/%s/emailPreferences", account),
		"newsAndTips":            "OPT_IN_STATE_OPTED_IN",
		"performanceSuggestions": "OPT_IN_STATE_OPTED_IN",
	}
}

func gcpShoppingMerchantAccountsHomepageFixture(account string, claimed bool) map[string]any {
	return map[string]any{
		"name":    fmt.Sprintf("accounts/%s/homepage", account),
		"uri":     "https://merchant.stackyard.example",
		"claimed": claimed,
	}
}

func gcpShoppingMerchantAccountsOmnichannelSettingFixture(account, settingID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("accounts/%s/omnichannelSettings/%s", account, settingID),
		"displayName": "Default Omnichannel Setting",
		"lsfType":     "LSF_TYPE_UNSPECIFIED",
	}
}

func gcpShoppingMerchantAccountsOnlineReturnPolicyFixture(account, policyID string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("accounts/%s/onlineReturnPolicies/%s", account, policyID),
		"label":         "Standard Return Policy",
		"countries":     []any{"US"},
		"returnMethods": []any{"RETURN_METHOD_BY_MAIL"},
	}
}

func gcpShoppingMerchantAccountsProgramFixture(account, programID, state string) map[string]any {
	if state == "" {
		state = "ENABLED"
	}
	return map[string]any{
		"name":              fmt.Sprintf("accounts/%s/programs/%s", account, programID),
		"documentationUri":  "https://support.google.com/merchants/answer/13889434",
		"state":             state,
		"activeRegionCodes": []any{"001"},
	}
}

func gcpShoppingMerchantAccountsRegionFixture(account, regionID, displayName string) map[string]any {
	if displayName == "" {
		displayName = "Stackyard Region " + regionID
	}
	return map[string]any{
		"name":        fmt.Sprintf("accounts/%s/regions/%s", account, regionID),
		"displayName": displayName,
		"postalCodeArea": map[string]any{
			"regionCode": "US",
			"postalCodes": []any{
				map[string]any{"begin": "10001", "end": "10002"},
			},
		},
		"regionalInventoryEligible": true,
		"shippingEligible":          true,
	}
}

func gcpShoppingMerchantAccountsShippingSettingsFixture(account string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("accounts/%s/shippingSettings", account),
		"services": []any{
			map[string]any{
				"serviceName": "Standard Shipping",
				"active":      true,
			},
		},
	}
}

func gcpShoppingMerchantAccountsTermsOfServiceAgreementStateFixture(account, stateID string) map[string]any {
	return map[string]any{
		"name":           fmt.Sprintf("accounts/%s/termsOfServiceAgreementStates/%s", account, stateID),
		"account":        "accounts/" + account,
		"termsOfService": "termsOfService/latest",
		"regionCode":     "US",
		"accepted":       true,
	}
}

func gcpShoppingMerchantAccountsNumericID(value string, fallback int64) int64 {
	digits := make([]rune, 0, len(value))
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) == 0 {
		return fallback
	}
	parsed, err := strconv.ParseInt(string(digits), 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func respondGCPShoppingMerchantAccountsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantAccountsOutOfRange(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "OutOfRange",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantAccountsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantAccountsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantAccountsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantAccounts(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_accounts") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/shopping_merchant_accounts/sample",
			"service":  "shopping_merchant_accounts",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
