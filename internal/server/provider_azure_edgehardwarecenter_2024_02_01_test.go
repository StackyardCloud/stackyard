package server

import (
	"net/http"
	"strings"
	"testing"
)

const ehwcTestAPI = "?api-version=" + azureEdgeHardwareCenter20240201APIVersion

const ehwcTestContact = `{"contactName":"Pat","phone":"5550100","emailList":["pat@example.com"]}`
const ehwcTestShipping = `{"streetAddress1":"1 Main St","country":"US","city":"Seattle"}`

func ehwcTestItem(orderID string) string {
	return `{"location":"eastus","properties":{"orderId":"` + orderID + `","orderItemDetails":{"orderItemType":"Purchase",` +
		`"productDetails":{"hierarchyInformation":{"productFamilyName":"azurestackedge","productLineName":"azurestackedge","productName":"azurestackedgegpu","configurationName":"edgep_base"}}},` +
		`"addressDetails":{"forwardAddress":{"contactDetails":` + ehwcTestContact + `,"shippingAddress":` + ehwcTestShipping + `}}}}`
}

func TestAzureEdgeHardwareCenterLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ehwc")
	rg := sub + "/resourceGroups/rg-ehwc/providers/Microsoft.EdgeOrder"
	orderID := strings.TrimPrefix(rg, "/azure") + "/locations/eastus/orders/order1"

	resp := do(http.MethodPut, rg+"/addresses/home"+ehwcTestAPI, `{"location":"eastus","properties":{"contactDetails":`+ehwcTestContact+`,"shippingAddress":`+ehwcTestShipping+`}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["addressValidationStatus"]; got != "Valid" {
		t.Fatalf("addressValidationStatus = %v", got)
	}

	item := rg + "/orderItems/item1"
	resp = do(http.MethodPut, item+ehwcTestAPI, ehwcTestItem(orderID))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := ehwcTestStage(t, resp); got != "Placed" {
		t.Fatalf("new item stage = %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, rg+"/orderItems/item2"+ehwcTestAPI, ehwcTestItem(orderID)), http.StatusCreated)

	resp = do(http.MethodGet, rg+"/locations/eastus/orders/order1"+ehwcTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if ids := providerContractJSONMap(t, resp)["properties"].(map[string]any)["orderItemIds"].([]any); len(ids) != 2 {
		t.Fatalf("order holds %d items", len(ids))
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.EdgeOrder/orders"+ehwcTestAPI, "")
	if orders := providerContractJSONMap(t, resp)["value"].([]any); len(orders) != 1 {
		t.Fatalf("subscription lists %d orders", len(orders))
	}

	resp = do(http.MethodPatch, item+ehwcTestAPI, `{"properties":{"notificationEmailList":["ops@example.com"]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	details := providerContractJSONMap(t, resp)["properties"].(map[string]any)["orderItemDetails"].(map[string]any)
	if len(details["notificationEmailList"].([]any)) != 1 || details["productDetails"] == nil {
		t.Fatalf("PATCH lost or skipped details: %#v", details)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, item+ehwcTestAPI, `{"properties":{"orderId":"x"}}`), http.StatusBadRequest, "InvalidRequestContent")

	azureResourcesExpectErrorCode(t, do(http.MethodDelete, item+ehwcTestAPI, ""), http.StatusConflict, "Conflict")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, item+"/cancel"+ehwcTestAPI, `{}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPost, item+"/cancel"+ehwcTestAPI, `{"reason":"changed plans"}`), http.StatusNoContent)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, item+"/return"+ehwcTestAPI, `{"returnReason":"late"}`), http.StatusConflict, "Conflict")
	resp = do(http.MethodPost, rg+"/orderItems/item2/return"+ehwcTestAPI, `{"returnReason":"faulty"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := ehwcTestStage(t, resp); got != "ReturnInitiated" {
		t.Fatalf("returned item stage = %v", got)
	}
	// A PATCH keeps the stage the cancel set.
	resp = do(http.MethodPatch, item+ehwcTestAPI, `{"tags":{"a":"b"}}`)
	if got := ehwcTestStage(t, resp); got != "Cancelled" {
		t.Fatalf("stage after PATCH = %v", got)
	}

	resp = do(http.MethodPost, sub+"/providers/Microsoft.EdgeOrder/listProductFamilies"+ehwcTestAPI, `{"filterableProperties":{"azurestackedge":[{"type":"ShipToCountries","supportedValues":["US"]}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.EdgeOrder/listProductFamilies"+ehwcTestAPI, `{}`), http.StatusBadRequest, "InvalidRequestContent")
	resp = do(http.MethodPost, sub+"/providers/Microsoft.EdgeOrder/listConfigurations"+ehwcTestAPI, `{"configurationFilter":{"hierarchyInformation":{"productFamilyName":"azurestackedge","productLineName":"azurestackedge","productName":"azurestackedgegpu"}}}`)
	if configs := providerContractJSONMap(t, resp)["value"].([]any); len(configs) != 2 {
		t.Fatalf("listConfigurations returned %d configurations", len(configs))
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.EdgeOrder/productFamiliesMetadata"+ehwcTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.EdgeOrder/operations"+ehwcTestAPI, ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodDelete, item+ehwcTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, rg+"/orderItems/item2"+ehwcTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, rg+"/locations/eastus/orders/order1"+ehwcTestAPI, ""), http.StatusNotFound, "ResourceNotFound")
}

func TestAzureEdgeHardwareCenterValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ehwc")
	rg := sub + "/resourceGroups/rg-ehwc/providers/Microsoft.EdgeOrder"
	goodOrder := strings.TrimPrefix(rg, "/azure") + "/locations/eastus/orders/order1"
	for name, body := range map[string]string{
		"other location":  ehwcTestItem(strings.Replace(goodOrder, "eastus", "westus", 1)),
		"bad order id":    ehwcTestItem("/subscriptions/x/orders/o"),
		"unknown product": strings.Replace(ehwcTestItem(goodOrder), "edgep_base", "edgep_mega", 1),
		"read-only stage": strings.Replace(ehwcTestItem(goodOrder), `"orderItemType"`, `"currentStage":{},"orderItemType"`, 1),
		"no address":      strings.Replace(ehwcTestItem(goodOrder), `"addressDetails"`, `"otherDetails"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, do(http.MethodPut, rg+"/orderItems/item1"+ehwcTestAPI, body), http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, rg+"/addresses/a"+ehwcTestAPI, `{"location":"eastus","properties":{"contactDetails":{"contactName":"Pat","phone":"1","emailList":["nope"]},"shippingAddress":`+ehwcTestShipping+`}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, rg+"/addresses/a"+ehwcTestAPI, `{"location":"eastus","properties":{"contactDetails":`+ehwcTestContact+`}}`), http.StatusBadRequest, "InvalidRequestContent")
}

func ehwcTestStage(t *testing.T, resp *http.Response) any {
	t.Helper()
	details := providerContractJSONMap(t, resp)["properties"].(map[string]any)["orderItemDetails"].(map[string]any)
	return details["currentStage"].(map[string]any)["stageName"]
}
