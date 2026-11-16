package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureDomainTestAPIVersion = "?api-version=2023-12-01"

const azureDomainTestContact = `{"email":"admin@contoso.com","nameFirst":"Ada","nameLast":"Admin","phone":"+1.4255550100","addressMailing":{"address1":"1 Main St","city":"Redmond","country":"US","postalCode":"98052","state":"WA"}}`

func azureDomainTestBody() string {
	return `{"location":"global","properties":{"contactAdmin":` + azureDomainTestContact + `,"contactBilling":` + azureDomainTestContact + `,"contactRegistrant":` + azureDomainTestContact + `,"contactTech":` + azureDomainTestContact + `,"consent":{"agreementKeys":["DNRA"],"agreedBy":"203.0.113.10","agreedAt":"2026-11-15T00:00:00Z"},"authCode":"transfer-secret","privacy":true}}`
}

func TestAzureDomainLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-domains")
	domain := sub + "/resourceGroups/rg-domains/providers/Microsoft.DomainRegistration/domains/contoso.com"
	check := func(name string) any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.DomainRegistration/checkDomainAvailability"+azureDomainTestAPIVersion, `{"name":"`+name+`"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)["available"]
	}
	if check("contoso.com") != true || check("contoso.dev") != false {
		t.Fatal("unexpected availability before registration")
	}

	resp := do(http.MethodPut, domain+azureDomainTestAPIVersion, azureDomainTestBody())
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["contactAdmin"] != nil || props["contactRegistrant"] != nil || props["authCode"] != nil {
		t.Fatalf("expected contacts and auth code stripped, got %#v", props)
	}
	if props["registrationStatus"] != "Active" || props["privacy"] != true || props["autoRenew"] != true || len(props["nameServers"].([]any)) != 4 {
		t.Fatalf("unexpected domain properties: %#v", props)
	}
	if check("contoso.com") != false {
		t.Fatal("expected a registered domain to be unavailable")
	}

	azureResourcesExpectStatus(t, do(http.MethodPost, domain+"/renew"+azureDomainTestAPIVersion, ""), http.StatusNoContent)
	azureResourcesExpectStatus(t, do(http.MethodPut, domain+"/domainOwnershipIdentifiers/verify"+azureDomainTestAPIVersion, `{"properties":{"ownershipId":"abc123"}}`), http.StatusCreated)

	resp = do(http.MethodGet, sub+"/providers/Microsoft.DomainRegistration/topLevelDomains"+azureDomainTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != len(azureDomainTopLevelDomains) {
		t.Fatalf("expected %d top-level domains, got %d", len(azureDomainTopLevelDomains), len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/providers/Microsoft.DomainRegistration/topLevelDomains/co.uk"+azureDomainTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodPost, sub+"/providers/Microsoft.DomainRegistration/topLevelDomains/com/listAgreements"+azureDomainTestAPIVersion, `{"includePrivacy":true}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) == 0 {
		t.Fatal("expected agreements")
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, domain+azureDomainTestAPIVersion, ""), http.StatusOK)
}

func TestAzureDomainValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-domains")
	domains := sub + "/resourceGroups/rg-domains/providers/Microsoft.DomainRegistration/domains/"
	body := azureDomainTestBody()
	cases := []struct{ name, path, body, code string }{
		{"unsupported tld", domains + "contoso.dev", body, "InvalidResourceName"},
		{"bad label", domains + "-contoso.com", body, "InvalidResourceName"},
		{"regional location", domains + "contoso.com", strings.Replace(body, `"global"`, `"eastus"`, 1), "InvalidRequestContent"},
		{"no email", domains + "contoso.com", strings.Replace(body, `"email":"admin@contoso.com",`, ``, 1), "InvalidRequestContent"},
		{"no consent", domains + "contoso.com", strings.Replace(body, `"agreementKeys":["DNRA"]`, `"agreementKeys":[]`, 1), "InvalidRequestContent"},
		{"bad dns type", domains + "contoso.com", strings.Replace(body, `"privacy":true`, `"dnsType":"Route53"`, 1), "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureDomainTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
}
