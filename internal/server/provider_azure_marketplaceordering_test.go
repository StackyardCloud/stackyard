package server

import (
	"net/http"
	"testing"
)

const azureMarketplaceTestAPIVersion = "?api-version=2021-01-01"

func TestAzureMarketplaceOrderingTermsAndAgreements(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-marketplace")
	ns := sub + "/providers/Microsoft.MarketplaceOrdering"
	terms := ns + "/offerTypes/virtualmachine/publishers/contoso/offers/firewall/plans/byol/agreements/current" + azureMarketplaceTestAPIVersion
	agreement := ns + "/agreements/contoso/offers/firewall/plans/byol"

	// Terms exist before anyone accepts them; the agreement doesn't.
	fresh := providerContractJSONMap(t, do(http.MethodGet, terms, ""))
	props := fresh["properties"].(map[string]any)
	if fresh["type"] != "Microsoft.MarketplaceOrdering/offertypes" || fresh["name"] != "byol" || props["accepted"] != false || props["retrieveDatetime"] != "" ||
		props["publisher"] != "contoso" || props["product"] != "firewall" || props["plan"] != "byol" || props["signature"] == "" || props["licenseTextLink"] == nil {
		t.Fatalf("unexpected terms: %#v", fresh)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, agreement+azureMarketplaceTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, agreement+"/cancel"+azureMarketplaceTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	// Declining terms never accepted leaves no agreement.
	azureResourcesExpectStatus(t, do(http.MethodPut, terms, `{"properties":{"accepted":false}}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, agreement+azureMarketplaceTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")

	accepted := providerContractJSONMap(t, do(http.MethodPut, terms, `{"properties":{"accepted":true,"signature":"`+props["signature"].(string)+`"}}`))
	ap := accepted["properties"].(map[string]any)
	if ap["accepted"] != true || ap["retrieveDatetime"] == "" || ap["signature"] != props["signature"] {
		t.Fatalf("accepting should keep the signature and set the time: %#v", accepted)
	}
	active := providerContractJSONMap(t, do(http.MethodGet, agreement+azureMarketplaceTestAPIVersion, ""))
	activeProps := active["properties"].(map[string]any)
	if active["type"] != "Microsoft.MarketplaceOrdering/agreements" || activeProps["state"] != "Active" || activeProps["offer"] != "firewall" || activeProps["id"] != active["id"] || activeProps["signDate"] != ap["retrieveDatetime"] {
		t.Fatalf("accepting terms should sign the agreement: %#v", active)
	}
	// Signing again keeps the first sign date.
	resigned := providerContractJSONMap(t, do(http.MethodPost, agreement+"/sign"+azureMarketplaceTestAPIVersion, ""))
	if resigned["properties"].(map[string]any)["signDate"] != activeProps["signDate"] {
		t.Fatalf("signing an active agreement should keep its date: %#v", resigned)
	}

	cancelled := providerContractJSONMap(t, do(http.MethodPost, agreement+"/cancel"+azureMarketplaceTestAPIVersion, ""))
	cp := cancelled["properties"].(map[string]any)
	if cp["state"] != "Canceled" || cp["cancelDate"] == nil || cp["signDate"] != nil {
		t.Fatalf("unexpected cancelled agreement: %#v", cancelled)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, terms, ""))["properties"].(map[string]any); got["accepted"] != false || got["retrieveDatetime"] != "" {
		t.Fatalf("cancelling should unaccept the terms: %#v", got)
	}

	// Signing directly, then declining through the terms, cancels.
	other := ns + "/agreements/fabrikam/offers/gateway/plans/standard"
	azureResourcesExpectStatus(t, do(http.MethodPost, other+"/sign"+azureMarketplaceTestAPIVersion, ""), http.StatusOK)
	otherTerms := ns + "/offerTypes/VirtualMachine/publishers/fabrikam/offers/gateway/plans/standard/agreements/current" + azureMarketplaceTestAPIVersion
	if got := providerContractJSONMap(t, do(http.MethodGet, otherTerms, ""))["properties"].(map[string]any)["accepted"]; got != true {
		t.Fatalf("a signed agreement should accept the terms, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, otherTerms, `{"properties":{"accepted":false}}`), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodGet, other+azureMarketplaceTestAPIVersion, ""))["properties"].(map[string]any)["state"]; got != "Canceled" {
		t.Fatalf("declining terms should cancel the agreement, got %v", got)
	}

	list := providerContractJSONMap(t, do(http.MethodGet, ns+"/agreements"+azureMarketplaceTestAPIVersion, ""))["value"].([]any)
	if len(list) != 2 {
		t.Fatalf("expected two agreements, got %#v", list)
	}
	otherSub := "/azure/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.MarketplaceOrdering/agreements" + azureMarketplaceTestAPIVersion
	if items := providerContractJSONMap(t, do(http.MethodGet, otherSub, ""))["value"].([]any); len(items) != 0 {
		t.Fatalf("agreements belong to their subscription: %#v", items)
	}
}

func TestAzureMarketplaceOrderingValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-marketplace")
	ns := sub + "/providers/Microsoft.MarketplaceOrdering"
	terms := ns + "/offerTypes/virtualmachine/publishers/contoso/offers/firewall/plans/byol/agreements/current" + azureMarketplaceTestAPIVersion
	for _, body := range []string{`{}`, `{"properties":{"accepted":"yes"}}`} {
		azureSentinelExpectRejected(t, do(http.MethodPut, terms, body), "properties.accepted")
	}
	azureSentinelExpectRejected(t, do(http.MethodGet, ns+"/offerTypes/saas/publishers/contoso/offers/firewall/plans/byol/agreements/current"+azureMarketplaceTestAPIVersion, ""), "virtualmachine")
	azureResourcesExpectStatus(t, do(http.MethodDelete, terms, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodPost, ns+"/agreements"+azureMarketplaceTestAPIVersion, ""), http.StatusMethodNotAllowed)
	agreement := ns + "/agreements/contoso/offers/firewall/plans/byol"
	azureResourcesExpectStatus(t, do(http.MethodGet, agreement+"/sign"+azureMarketplaceTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodDelete, agreement+azureMarketplaceTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, agreement+"/renew"+azureMarketplaceTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodGet, ns+"/agreements"+"?api-version=", ""), http.StatusBadRequest, "InvalidApiVersionParameter")
}
