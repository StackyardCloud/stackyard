package server

import (
	"net/http"
	"testing"
	"time"
)

const azureCertificateOrderTestAPIVersion = "?api-version=2023-12-01"

func TestAzureCertificateOrderLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-certs")
	order := sub + "/resourceGroups/rg-certs/providers/Microsoft.CertificateRegistration/certificateOrders/contoso"

	resp := do(http.MethodPut, order+azureCertificateOrderTestAPIVersion, `{"location":"global","properties":{"productType":"StandardDomainValidatedSsl","distinguishedName":"CN=contoso.com","validityInYears":2,"csr":"-----BEGIN CERTIFICATE REQUEST-----","status":"Issued"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	issued, _ := time.Parse(time.RFC3339, props["lastCertificateIssuanceTime"].(string))
	if props["expirationTime"] != issued.AddDate(2, 0, 0).Format(time.RFC3339) || props["status"] != "Pendingissuance" || props["csr"] != nil || props["autoRenew"] != true || len(props["domainVerificationToken"].(string)) != 32 {
		t.Fatalf("unexpected order properties: %#v", props)
	}

	// Verifying domain ownership issues the certificate; a PATCH can't.
	azureResourcesExpectStatus(t, do(http.MethodPost, order+"/verifyDomainOwnership"+azureCertificateOrderTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodPatch, order+azureCertificateOrderTestAPIVersion, `{"properties":{"autoRenew":false,"status":"Pendingissuance"}}`)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["status"] != "Issued" || props["autoRenew"] != false {
		t.Fatalf("unexpected order after PATCH: %#v", props)
	}

	for _, action := range []string{"renew", "reissue", "resendEmail"} {
		azureResourcesExpectStatus(t, do(http.MethodPost, order+"/"+action+azureCertificateOrderTestAPIVersion, `{}`), http.StatusNoContent)
	}
	resp = do(http.MethodPost, order+"/retrieveSiteSeal"+azureCertificateOrderTestAPIVersion, `{"lightTheme":true}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if providerContractJSONMap(t, resp)["html"] == "" {
		t.Fatal("expected site seal HTML")
	}

	cert := order + "/certificates/contoso-kv"
	vault := sub + "/resourceGroups/rg-certs/providers/Microsoft.KeyVault/vaults/certs"
	azureResourcesExpectStatus(t, do(http.MethodPut, cert+azureCertificateOrderTestAPIVersion, `{"location":"global","properties":{"keyVaultId":"`+vault+`","keyVaultSecretName":"contoso"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, cert+azureCertificateOrderTestAPIVersion, `{"location":"global","properties":{"keyVaultId":"/subscriptions/x","keyVaultSecretName":"contoso"}}`), http.StatusBadRequest, "InvalidRequestContent")

	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.CertificateRegistration/validateCertificateRegistrationInformation"+azureCertificateOrderTestAPIVersion, `{"location":"global","properties":{"productType":"StandardDomainValidatedWildCardSsl"}}`), http.StatusNoContent)
	azureResourcesExpectStatus(t, do(http.MethodDelete, order+azureCertificateOrderTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, cert+azureCertificateOrderTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureCertificateOrderValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-certs")
	order := sub + "/resourceGroups/rg-certs/providers/Microsoft.CertificateRegistration/certificateOrders/contoso"
	cases := []struct{ name, body string }{
		{"regional location", `{"location":"eastus","properties":{"productType":"StandardDomainValidatedSsl"}}`},
		{"no product", `{"location":"global","properties":{}}`},
		{"bad product", `{"location":"global","properties":{"productType":"ExtendedValidation"}}`},
		{"bad dn", `{"location":"global","properties":{"productType":"StandardDomainValidatedSsl","distinguishedName":"contoso.com"}}`},
		{"bad validity", `{"location":"global","properties":{"productType":"StandardDomainValidatedSsl","validityInYears":4}}`},
		{"bad key size", `{"location":"global","properties":{"productType":"StandardDomainValidatedSsl","keySize":1024}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, order+azureCertificateOrderTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
