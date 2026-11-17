package server

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

const azureReservationsTestAPIVersion = "?api-version=2022-11-01"

const azureReservationPurchase = `{"location":"westus","sku":{"name":"Standard_D2s_v5"},"properties":{"reservedResourceType":"VirtualMachines","billingScopeId":"/subscriptions/00000000-0000-0000-0000-000000000000","term":"P1Y","billingPlan":"Monthly","quantity":4,"displayName":"web tier","appliedScopeType":"Shared","renew":false,"reservedResourceProperties":{"instanceFlexibility":"On"}}}`

// azureReservationPurchased prices and purchases an order, returning its
// path and its reservation's ID.
func azureReservationPurchased(t *testing.T, do func(method, path, body string) *http.Response) (string, string) {
	t.Helper()
	price := providerContractJSONMap(t, do(http.MethodPost, "/azure/providers/Microsoft.Capacity/calculatePrice"+azureReservationsTestAPIVersion, azureReservationPurchase))
	props := price["properties"].(map[string]any)
	if props["grandTotal"] != float64(4800) {
		t.Fatalf("4 units for 12 months at the sample rate should cost 4800: %#v", props)
	}
	order := "/azure/providers/Microsoft.Capacity/reservationOrders/" + props["reservationOrderId"].(string)
	body := providerContractJSONMap(t, do(http.MethodPut, order+azureReservationsTestAPIVersion, azureReservationPurchase))
	orderProps := body["properties"].(map[string]any)
	if orderProps["originalQuantity"] != float64(4) || orderProps["term"] != "P1Y" || orderProps["quantity"] != nil || orderProps["expiryDate"] == nil {
		t.Fatalf("unexpected order: %#v", body)
	}
	return order, orderProps["reservations"].([]any)[0].(map[string]any)["id"].(string)
}

func TestAzureReservationsPurchaseAndUpdate(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-reservations")
	order, reservationID := azureReservationPurchased(t, do)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, order+azureReservationsTestAPIVersion, azureReservationPurchase), http.StatusBadRequest, "InvalidRequestContent")

	reservation := providerContractJSONMap(t, do(http.MethodGet, "/azure"+reservationID+azureReservationsTestAPIVersion, ""))
	props := reservation["properties"].(map[string]any)
	if props["quantity"] != float64(4) || props["instanceFlexibility"] != "On" || props["appliedScopeType"] != "Shared" || reservation["location"] != "westus" || reservation["sku"].(map[string]any)["name"] != "Standard_D2s_v5" {
		t.Fatalf("unexpected reservation: %#v", reservation)
	}
	// Reservations come only from purchases.
	azureResourcesExpectErrorCode(t, do(http.MethodPut, order+"/reservations/extra"+azureReservationsTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")

	patched := providerContractJSONMap(t, do(http.MethodPatch, "/azure"+reservationID+azureReservationsTestAPIVersion, `{"properties":{"name":"renamed","appliedScopeType":"Single","appliedScopes":["/subscriptions/00000000-0000-0000-0000-000000000000"]}}`))
	if p := patched["properties"].(map[string]any); p["displayName"] != "renamed" || p["appliedScopeType"] != "Single" || p["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected patched reservation: %#v", p)
	}
	for name, payload := range map[string]string{
		"single":     `{"properties":{"appliedScopeType":"Single"}}`,
		"shared":     `{"properties":{"appliedScopeType":"Shared","appliedScopes":["/subscriptions/x"]}}`,
		"mgmt group": `{"properties":{"appliedScopeType":"ManagementGroup","appliedScopeProperties":{"managementGroupId":"/providers/Microsoft.Management/managementGroups/mg"}}}`,
		"flexible":   `{"properties":{"instanceFlexibility":"Maybe"}}`,
	} {
		resp := do(http.MethodPatch, "/azure"+reservationID+azureReservationsTestAPIVersion, payload)
		t.Run("patch "+name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	rescoped := providerContractJSONMap(t, do(http.MethodPatch, "/azure"+reservationID+azureReservationsTestAPIVersion, `{"properties":{"appliedScopeType":"Single","appliedScopeProperties":{"resourceGroupId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-reservations"}}}`))
	scopeProps := rescoped["properties"].(map[string]any)["appliedScopeProperties"].(map[string]any)
	if scopeProps["resourceGroupId"] == nil || scopeProps["subscriptionId"] != nil || rescoped["properties"].(map[string]any)["appliedScopes"] != nil {
		t.Fatalf("a new scope should replace the old one: %#v", rescoped["properties"])
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, "/azure"+reservationID+azureReservationsTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodDelete, order+azureReservationsTestAPIVersion, ""), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure"+reservationID+"/archive"+azureReservationsTestAPIVersion, ""), http.StatusOK)
	if got := providerContractJSONMap(t, do(http.MethodGet, "/azure"+reservationID+azureReservationsTestAPIVersion, ""))["properties"].(map[string]any)["archived"]; got != true {
		t.Fatalf("archive should set archived, got %v", got)
	}
	scopes := providerContractJSONMap(t, do(http.MethodPost, "/azure"+reservationID+"/availableScopes"+azureReservationsTestAPIVersion, `{"properties":{"scopes":["/subscriptions/00000000-0000-0000-0000-000000000000","/providers/x"]}}`))
	if s := scopes["properties"].(map[string]any)["scopes"].([]any); s[0].(map[string]any)["valid"] != true || s[1].(map[string]any)["valid"] != false {
		t.Fatalf("unexpected scopes: %#v", s)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, "/azure/providers/Microsoft.Capacity/reservations"+azureReservationsTestAPIVersion, ""))["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one reservation in the tenant list, got %d", len(got))
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, order+"/reservations"+azureReservationsTestAPIVersion, ""))["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one reservation in the order, got %d", len(got))
	}
}

func TestAzureReservationsPurchaseValidation(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-reservations")
	order := "/azure/providers/Microsoft.Capacity/reservationOrders/1f14354c-dc12-4c8d-8090-6f295a3a34aa" + azureReservationsTestAPIVersion
	for name, payload := range map[string]string{
		"sku":           strings.Replace(azureReservationPurchase, `"name":"Standard_D2s_v5"`, `"name":""`, 1),
		"resource type": strings.Replace(azureReservationPurchase, `"VirtualMachines"`, `"Toasters"`, 1),
		"billing scope": strings.Replace(azureReservationPurchase, `/subscriptions/00000000-0000-0000-0000-000000000000`, `/subscriptions/mine`, 1),
		"term":          strings.Replace(azureReservationPurchase, `"P1Y"`, `"P2Y"`, 1),
		"plan":          strings.Replace(azureReservationPurchase, `"Monthly"`, `"Weekly"`, 1),
		"quantity":      strings.Replace(azureReservationPurchase, `"quantity":4`, `"quantity":0`, 1),
		"scope":         strings.Replace(azureReservationPurchase, `"appliedScopeType":"Shared",`, ``, 1),
		"single":        strings.Replace(azureReservationPurchase, `"Shared"`, `"Single"`, 1),
		"flexibility":   strings.Replace(azureReservationPurchase, `"instanceFlexibility":"On"`, `"instanceFlexibility":"Yes"`, 1),
	} {
		resp := do(http.MethodPut, order, payload)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure/providers/Microsoft.Capacity/reservationOrders/my-order"+azureReservationsTestAPIVersion, azureReservationPurchase), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, "/azure/providers/Microsoft.Capacity/calculatePrice"+azureReservationsTestAPIVersion, `{"sku":{"name":"x"},"properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
	if got := providerContractJSONMap(t, do(http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Capacity/catalogs"+azureReservationsTestAPIVersion, ""))["value"].([]any); len(got) == 0 {
		t.Fatal("expected a sample catalog")
	}
}

func TestAzureReservationsSplitMergeAndReturn(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-reservations")
	order, reservationID := azureReservationPurchased(t, do)
	split := order + "/split" + azureReservationsTestAPIVersion
	azureResourcesExpectErrorCode(t, do(http.MethodPost, split, `{"properties":{"quantities":[1,2],"reservationId":"`+reservationID+`"}}`), http.StatusBadRequest, "InvalidRequestContent")
	parts := string(providerContractBody(t, do(http.MethodPost, split, `{"properties":{"quantities":[1,3],"reservationId":"`+reservationID+`"}}`)))
	if !strings.Contains(parts, `"quantity":1`) || !strings.Contains(parts, `"quantity":3`) || !strings.Contains(parts, `"Split"`) {
		t.Fatalf("split should return both parts and the Split source: %s", parts)
	}
	resp := do(http.MethodPost, split, `{"properties":{"quantities":[2,2],"reservationId":"`+reservationID+`"}}`)
	// The source is now Split, so it can't be split again.
	azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")

	orderBody := providerContractJSONMap(t, do(http.MethodGet, order+azureReservationsTestAPIVersion, ""))
	list := orderBody["properties"].(map[string]any)["reservations"].([]any)
	if len(list) != 3 {
		t.Fatalf("the order should list the source and both parts, got %#v", list)
	}
	source := providerContractJSONMap(t, do(http.MethodGet, "/azure"+reservationID+azureReservationsTestAPIVersion, ""))["properties"].(map[string]any)
	destinations := source["splitProperties"].(map[string]any)["splitDestinations"].([]any)
	if source["provisioningState"] != "Split" || len(destinations) != 2 {
		t.Fatalf("unexpected split source: %#v", source)
	}
	a, b := destinations[0].(string), destinations[1].(string)

	azureResourcesExpectErrorCode(t, do(http.MethodPost, order+"/merge"+azureReservationsTestAPIVersion, `{"properties":{"sources":["`+a+`","`+a+`"]}}`), http.StatusBadRequest, "InvalidRequestContent")
	merged := providerContractBody(t, do(http.MethodPost, order+"/merge"+azureReservationsTestAPIVersion, `{"properties":{"sources":["`+a+`","`+b+`"]}}`))
	if !strings.Contains(string(merged), `"quantity":4`) || !strings.Contains(string(merged), `"Merged"`) {
		t.Fatalf("merge should make one reservation of 4 and mark the sources Merged: %s", merged)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, order+"/merge"+azureReservationsTestAPIVersion, `{"properties":{"sources":["`+a+`","`+b+`"]}}`), http.StatusBadRequest, "InvalidRequestContent")
	mergedID := providerContractJSONMap(t, do(http.MethodGet, "/azure"+a+azureReservationsTestAPIVersion, ""))["properties"].(map[string]any)["mergeProperties"].(map[string]any)["mergeDestination"].(string)
	if got := providerContractJSONMap(t, do(http.MethodGet, "/azure"+mergedID+azureReservationsTestAPIVersion, ""))["properties"].(map[string]any); got["splitProperties"] != nil {
		t.Fatalf("a merged reservation shouldn't inherit its sources' split history: %#v", got)
	}

	// Returning needs the session calculateRefund gives.
	refundBody := `{"properties":{"scope":"Reservation","reservationToReturn":{"reservationId":"` + mergedID + `","quantity":4}}}`
	refund := providerContractJSONMap(t, do(http.MethodPost, order+"/calculateRefund"+azureReservationsTestAPIVersion, refundBody))
	refundProps := refund["properties"].(map[string]any)
	session := refundProps["sessionId"].(string)
	// Refunds prorate by whole UTC days, so a day boundary between the
	// purchase and the refund may take off one day's share.
	if amount := refundProps["billingRefundAmount"].(map[string]any)["amount"].(float64); amount > 4800 || amount < 4780 {
		t.Fatalf("unexpected refund: %#v", refundProps)
	}
	returnPath := order + "/return" + azureReservationsTestAPIVersion
	azureResourcesExpectErrorCode(t, do(http.MethodPost, returnPath, `{"properties":{"sessionId":"wrong","scope":"Reservation","reservationToReturn":{"reservationId":"`+mergedID+`","quantity":4}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPost, returnPath, `{"properties":{"sessionId":"`+session+`","scope":"Reservation","reservationToReturn":{"reservationId":"`+mergedID+`","quantity":5}}}`), http.StatusBadRequest, "InvalidRequestContent")
	returned := providerContractJSONMap(t, do(http.MethodPost, returnPath, `{"properties":{"sessionId":"`+session+`","scope":"Reservation","reservationToReturn":{"reservationId":"`+mergedID+`","quantity":4}}}`))
	if returned["properties"].(map[string]any)["provisioningState"] != "Cancelled" {
		t.Fatalf("returning the last reservation should cancel the order: %#v", returned)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, "/azure"+mergedID+azureReservationsTestAPIVersion, ""))["properties"].(map[string]any)["provisioningState"]; got != "Cancelled" {
		t.Fatalf("the returned reservation should be Cancelled, got %v", got)
	}
	moved := providerContractJSONMap(t, do(http.MethodPost, order+"/changeDirectory"+azureReservationsTestAPIVersion, `{"destinationTenantId":"72f988bf-86f1-41af-91ab-2d7cd011db47"}`))
	if moved["reservationOrder"].(map[string]any)["isSucceeded"] != true || len(moved["reservations"].([]any)) != 4 {
		t.Fatalf("unexpected change directory result: %#v", moved)
	}
}

func TestAzureReservationsPartialReturn(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-reservations")
	order, reservationID := azureReservationPurchased(t, do)
	body := `{"properties":{"scope":"Reservation","reservationToReturn":{"reservationId":"` + reservationID + `","quantity":1}}}`
	session := providerContractJSONMap(t, do(http.MethodPost, order+"/calculateRefund"+azureReservationsTestAPIVersion, body))["properties"].(map[string]any)["sessionId"].(string)
	returned := providerContractJSONMap(t, do(http.MethodPost, order+"/return"+azureReservationsTestAPIVersion, strings.Replace(body, `"properties":{`, `"properties":{"sessionId":"`+session+`",`, 1)))
	if returned["properties"].(map[string]any)["provisioningState"] != "Succeeded" {
		t.Fatalf("a partial return keeps the order: %#v", returned)
	}
	if got := providerContractJSONMap(t, do(http.MethodGet, "/azure"+reservationID+azureReservationsTestAPIVersion, ""))["properties"].(map[string]any)["quantity"]; got != float64(3) {
		t.Fatalf("a partial return should leave 3 units, got %v", got)
	}
}

// azureReservationReturnUnits prices and returns quantity units, and
// reports the refund policy errors and the return's status.
func azureReservationReturnUnits(t *testing.T, do func(method, path, body string) *http.Response, order, reservationID string, quantity int) ([]any, int) {
	t.Helper()
	body := `{"properties":{"scope":"Reservation","reservationToReturn":{"reservationId":"` + reservationID + `","quantity":` + strconv.Itoa(quantity) + `}}}`
	refund := providerContractJSONMap(t, do(http.MethodPost, order+"/calculateRefund"+azureReservationsTestAPIVersion, body))["properties"].(map[string]any)
	errors := refund["policyResult"].(map[string]any)["properties"].(map[string]any)["policyErrors"].([]any)
	resp := do(http.MethodPost, order+"/return"+azureReservationsTestAPIVersion, strings.Replace(body, `"properties":{`, `"properties":{"sessionId":"`+refund["sessionId"].(string)+`",`, 1))
	providerContractBody(t, resp)
	return errors, resp.StatusCode
}

func TestAzureReservationsRefundLimit(t *testing.T) {
	t.Parallel()
	_, do := newAzureDatabaseTestServer(t, "rg-reservations")
	// 50 units for 12 months at the sample rate commit USD 60,000.
	purchase := strings.Replace(azureReservationPurchase, `"quantity":4`, `"quantity":50`, 1)
	order := "/azure/providers/Microsoft.Capacity/reservationOrders/2a14354c-dc12-4c8d-8090-6f295a3a34aa"
	body := providerContractJSONMap(t, do(http.MethodPut, order+azureReservationsTestAPIVersion, purchase))
	reservationID := body["properties"].(map[string]any)["reservations"].([]any)[0].(map[string]any)["id"].(string)

	if errors, status := azureReservationReturnUnits(t, do, order, reservationID, 50); len(errors) != 1 || status != http.StatusBadRequest {
		t.Fatalf("returning USD 60,000 should exceed the limit: errors %v, status %d", errors, status)
	}
	if errors, status := azureReservationReturnUnits(t, do, order, reservationID, 41); len(errors) != 0 || status != http.StatusOK {
		t.Fatalf("returning USD 49,200 should succeed: errors %v, status %d", errors, status)
	}
	// The next USD 1,200 would bring the 12-month total to USD 50,400.
	if errors, status := azureReservationReturnUnits(t, do, order, reservationID, 1); len(errors) != 1 || status != http.StatusBadRequest {
		t.Fatalf("the limit should count earlier refunds: errors %v, status %d", errors, status)
	}

	redHat := "/azure/providers/Microsoft.Capacity/reservationOrders/3b14354c-dc12-4c8d-8090-6f295a3a34aa"
	body = providerContractJSONMap(t, do(http.MethodPut, redHat+azureReservationsTestAPIVersion, strings.Replace(azureReservationPurchase, `"VirtualMachines"`, `"RedHat"`, 1)))
	redHatID := body["properties"].(map[string]any)["reservations"].([]any)[0].(map[string]any)["id"].(string)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, redHat+"/calculateRefund"+azureReservationsTestAPIVersion, `{"properties":{"scope":"Reservation","reservationToReturn":{"reservationId":"`+redHatID+`","quantity":1}}}`), http.StatusBadRequest, "InvalidRequestContent")
}
