package server

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.CertificateRegistration App Service certificate orders and the
// Key Vault bindings of their certificates, served by the shared ARM
// handler. No certificate is purchased or issued by a CA: an order is
// Pendingissuance until verifyDomainOwnership marks it Issued.
func init() {
	const ns = "Microsoft.CertificateRegistration"
	noContent := func(*azureARMResource, map[string]any) (int, any) { return http.StatusNoContent, nil }
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "certificateOrders",
		ReadOnlyProperties: azureCertificateOrderProperties,
		Validate:           azureCertificateOrderValidate,
		StateProperties:    map[string]any{"status": "Pendingissuance"},
		Transitions:        map[string]map[string]any{"verifydomainownership": {"status": "Issued"}},
		Actions: map[string]azureARMAction{
			"renew":                      noContent,
			"reissue":                    noContent,
			"resendemail":                noContent,
			"resendrequestemails":        noContent,
			"retrievesiteseal":           azureCertificateOrderSiteSeal,
			"retrievecertificateactions": func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, []any{} },
			"retrieveemailhistory":       func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, []any{} },
		},
		// The CSR may embed a private key reference and is never returned.
		WriteOnlyProperties: []string{"csr"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "certificateOrders/certificates",
		Validate: func(payload map[string]any) string {
			if msg := azureValidateGlobalLocation(payload); msg != "" {
				return msg
			}
			props := azureARMObject(payload["properties"])
			if !strings.Contains(strings.ToLower(azureARMString(props["keyVaultId"])), "/providers/microsoft.keyvault/vaults/") {
				return "properties.keyVaultId must be a key vault ID."
			}
			if strings.TrimSpace(azureARMString(props["keyVaultSecretName"])) == "" {
				return "properties.keyVaultSecretName is required."
			}
			return ""
		},
	})
	registerAzureARMProviderAction(ns, "validateCertificateRegistrationInformation", func(_ *Server, payload map[string]any) (int, any) {
		if msg := azureCertificateOrderValidate(payload); msg != "" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
		}
		return http.StatusNoContent, nil
	})
}

// azureValidateGlobalLocation requires location "global", as for App Service
// certificate orders and domains.
func azureValidateGlobalLocation(payload map[string]any) string {
	if !strings.EqualFold(azureARMString(payload["location"]), "global") {
		return "The location must be global."
	}
	return ""
}

var azureCertificateOrderDNPattern = regexp.MustCompile(`(?i)^CN=[^,]+`)

func azureCertificateOrderValidate(payload map[string]any) string {
	if msg := azureValidateGlobalLocation(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if !slices.Contains([]string{"StandardDomainValidatedSsl", "StandardDomainValidatedWildCardSsl"}, azureARMString(props["productType"])) {
		return "properties.productType must be StandardDomainValidatedSsl or StandardDomainValidatedWildCardSsl."
	}
	if dn, present := props["distinguishedName"]; present && !azureCertificateOrderDNPattern.MatchString(azureARMString(dn)) {
		return "properties.distinguishedName must start with CN=."
	}
	if raw, present := props["validityInYears"]; present {
		if years, ok := raw.(float64); !ok || years != float64(int(years)) || years < 1 || years > 3 {
			return "properties.validityInYears must be 1, 2 or 3."
		}
	}
	if size, present := props["keySize"]; present && size != float64(2048) && size != float64(4096) {
		return "properties.keySize must be 2048 or 4096."
	}
	return ""
}

// azureCertificateOrderProperties reports the order's verification token,
// one-year validity and the renewal defaults.
func azureCertificateOrderProperties(rec *azureARMResource) map[string]any {
	years := 1
	if raw, ok := rec.Properties["validityInYears"].(float64); ok {
		years = int(raw)
	}
	expires := rec.CreatedAt.AddDate(years, 0, 0)
	token := sha256.Sum256([]byte(azureARMKey(rec.ID) + "#domain-verification"))
	props := map[string]any{
		"domainVerificationToken":                  hex.EncodeToString(token[:16]),
		"serialNumber":                             strings.ToUpper(strings.ReplaceAll(uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(), "-", ""))[:16],
		"expirationTime":                           expires.Format(time.RFC3339),
		"lastCertificateIssuanceTime":              rec.CreatedAt.Format(time.RFC3339),
		"nextAutoRenewalTimeStamp":                 expires.AddDate(0, 0, -30).Format(time.RFC3339),
		"appServiceCertificateNotRenewableReasons": []any{},
		"isPrivateKeyExternal":                     false,
	}
	for key, value := range map[string]any{"validityInYears": 1, "keySize": 2048, "autoRenew": true} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureCertificateOrderSiteSeal(rec *azureARMResource, payload map[string]any) (int, any) {
	theme := "dark"
	if light, _ := payload["lightTheme"].(bool); light {
		theme = "light"
	}
	return http.StatusOK, map[string]any{
		"html": `<span id="siteseal" class="` + theme + `">` + html.EscapeString(rec.Name) + `</span>`,
	}
}
