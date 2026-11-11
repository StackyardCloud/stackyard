package server

import (
	"net/http"
	"net/url"
	"testing"
)

func TestAzureCommunicationPhoneNumbersDiscoveryAndSearch(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	base := "/azure/communication-phonenumbers/2025-06-01"

	resp := providerContractRequest(t, ts, http.MethodGet, base+"/availablePhoneNumbers/countries?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list countries, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	countriesPayload := providerContractJSONMap(t, resp)
	countries := countriesPayload["value"].([]any)
	if len(countries) < 2 {
		t.Fatalf("expected at least two countries, got %#v", countriesPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/availablePhoneNumbers/countries/US/areaCodes?phoneNumberType=geographic&api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list area codes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/availablePhoneNumbers/countries/US/:browse?api-version="+azureCommunicationPhoneNumbersAPIVersion, []byte(`{
		"phoneNumberType":"geographic",
		"assignmentType":"person",
		"capabilities":{"calling":"inbound+outbound","sms":"inbound+outbound"},
		"phoneNumberPrefixes":["+1425"]
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for browse, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	browsePayload := providerContractJSONMap(t, resp)
	browseValues := browsePayload["value"].([]any)
	if len(browseValues) == 0 {
		t.Fatalf("expected browse results, got %#v", browsePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/availablePhoneNumbers/countries/US/:search?api-version="+azureCommunicationPhoneNumbersAPIVersion, []byte(`{
		"phoneNumberType":"geographic",
		"assignmentType":"person",
		"capabilities":{"calling":"inbound+outbound","sms":"inbound+outbound"},
		"areaCode":"425",
		"quantity":2
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for search, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	searchHeaders := resp.Header
	searchPayload := providerContractJSONMap(t, resp)
	operationID := providerContractJSONString(t, searchPayload["operationId"])
	searchID := providerContractJSONString(t, searchPayload["searchId"])
	if searchHeaders.Get("Operation-Location") == "" || searchHeaders.Get("Location") == "" || searchHeaders.Get("search-id") == "" {
		t.Fatalf("expected lro headers for search, got %#v", searchHeaders)
	}
	if operationID == "" || searchID == "" {
		t.Fatalf("expected operationId and searchId, got %#v", searchPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, searchHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for first operation poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	firstPoll := providerContractJSONMap(t, resp)
	if providerContractJSONString(t, firstPoll["status"]) != "running" {
		t.Fatalf("expected running status on first poll, got %#v", firstPoll)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, searchHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for second operation poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	secondPoll := providerContractJSONMap(t, resp)
	if providerContractJSONString(t, secondPoll["status"]) != "succeeded" {
		t.Fatalf("expected succeeded status on second poll, got %#v", secondPoll)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/availablePhoneNumbers/searchResults/"+searchID+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get search result, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resultPayload := providerContractJSONMap(t, resp)
	phoneNumbers := resultPayload["phoneNumbers"].([]any)
	if len(phoneNumbers) != 2 {
		t.Fatalf("expected 2 search results, got %#v", resultPayload)
	}
}

func TestAzureCommunicationPhoneNumbersReservationPurchaseAndInventory(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	base := "/azure/communication-phonenumbers/2025-06-01"

	resp := providerContractRequest(t, ts, http.MethodPost, base+"/availablePhoneNumbers/countries/US/:search?api-version="+azureCommunicationPhoneNumbersAPIVersion, []byte(`{
		"phoneNumberType":"geographic",
		"assignmentType":"person",
		"capabilities":{"calling":"inbound+outbound","sms":"inbound+outbound"},
		"areaCode":"425",
		"quantity":1
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for search, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	searchPayload := providerContractJSONMap(t, resp)
	searchID := providerContractJSONString(t, searchPayload["searchId"])

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/availablePhoneNumbers/searchResults/"+searchID+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for search result, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	searchResult := providerContractJSONMap(t, resp)
	first := providerContractJSONMapFromAny(t, searchResult["phoneNumbers"].([]any)[0])
	phoneNumber := providerContractJSONString(t, first["phoneNumber"])

	reservationID := "reservation-seattle-001"
	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/availablePhoneNumbers/reservations/"+reservationID+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, []byte(`{
		"searchId":"`+searchID+`",
		"displayName":"Seattle Reservation",
		"phoneNumbers":["`+phoneNumber+`"]
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for reservation create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/availablePhoneNumbers/reservations/"+reservationID+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get reservation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/availablePhoneNumbers/reservations/"+reservationID+"/:purchase?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for purchase reservation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	purchaseHeaders := resp.Header
	if purchaseHeaders.Get("Operation-Location") == "" {
		t.Fatalf("expected purchase operation location header, got %#v", purchaseHeaders)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, purchaseHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for first purchase poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, purchaseHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for second purchase poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	pollPayload := providerContractJSONMap(t, resp)
	if providerContractJSONString(t, pollPayload["status"]) != "succeeded" {
		t.Fatalf("expected succeeded purchase operation, got %#v", pollPayload)
	}

	encoded := url.PathEscape(phoneNumber)
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/phoneNumbers/"+encoded+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for purchased phone number, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	numberPayload := providerContractJSONMap(t, resp)
	if providerContractJSONString(t, numberPayload["phoneNumber"]) != phoneNumber {
		t.Fatalf("expected purchased phone number payload, got %#v", numberPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/phoneNumbers?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list phone numbers, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listPayload := providerContractJSONMap(t, resp)
	if len(listPayload["value"].([]any)) == 0 {
		t.Fatalf("expected purchased number in list, got %#v", listPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/phoneNumbers/"+encoded+"/capabilities?api-version="+azureCommunicationPhoneNumbersAPIVersion, []byte(`{
		"capabilities":{"calling":"inbound","sms":"outbound"}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for update capabilities, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	updateHeaders := resp.Header
	resp = providerContractRequest(t, ts, http.MethodGet, updateHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for first update poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, updateHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for second update poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/phoneNumbers/"+encoded+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for updated phone number, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	updatedPayload := providerContractJSONMap(t, resp)
	caps := providerContractJSONMapFromAny(t, updatedPayload["capabilities"])
	if providerContractJSONString(t, caps["calling"]) != "inbound" || providerContractJSONString(t, caps["sms"]) != "outbound" {
		t.Fatalf("expected updated capabilities, got %#v", updatedPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/operatorInformation/:search?api-version="+azureCommunicationPhoneNumbersAPIVersion, []byte(`{"phoneNumbers":["`+phoneNumber+`"]}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for operator info search, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/phoneNumbers/"+encoded+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 for release phone number, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	releaseHeaders := resp.Header
	resp = providerContractRequest(t, ts, http.MethodGet, releaseHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for first release poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, releaseHeaders.Get("Operation-Location"), nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for second release poll, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	releasePayload := providerContractJSONMap(t, resp)
	if providerContractJSONString(t, releasePayload["status"]) != "succeeded" {
		t.Fatalf("expected succeeded release operation, got %#v", releasePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/phoneNumbers/"+encoded+"?api-version="+azureCommunicationPhoneNumbersAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after release, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureCommunicationPhoneNumbersValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/communication-phonenumbers/2025-06-01/availablePhoneNumbers/countries?api-version=2024-01-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidApiVersion",
		},
		{
			name:       "missing phone number type for area codes",
			method:     http.MethodGet,
			path:       "/azure/communication-phonenumbers/2025-06-01/availablePhoneNumbers/countries/US/areaCodes?api-version=2025-06-01",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid search payload",
			method:     http.MethodPost,
			path:       "/azure/communication-phonenumbers/2025-06-01/availablePhoneNumbers/countries/US/:search?api-version=2025-06-01",
			body:       []byte(`{"phoneNumberType":"geographic"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid reservation json",
			method:     http.MethodPatch,
			path:       "/azure/communication-phonenumbers/2025-06-01/availablePhoneNumbers/reservations/oops?api-version=2025-06-01",
			body:       []byte(`{`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing phone number for operator search",
			method:     http.MethodPost,
			path:       "/azure/communication-phonenumbers/2025-06-01/operatorInformation/:search?api-version=2025-06-01",
			body:       []byte(`{}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "missing phone number",
			method:     http.MethodGet,
			path:       "/azure/communication-phonenumbers/2025-06-01/phoneNumbers/%2B14255559999?api-version=2025-06-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "PhoneNumberNotFound",
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/azure/communication-phonenumbers/2025-06-01/unknown?api-version=2025-06-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "PathNotFound",
		},
		{
			name:       "wrong method",
			method:     http.MethodGet,
			path:       "/azure/communication-phonenumbers/2025-06-01/operatorInformation/:search?api-version=2025-06-01",
			headers:    headers,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, tc.headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			if tc.wantCode == "" {
				return
			}
			payload := providerContractJSONMap(t, resp)
			errPayload := providerContractJSONMapFromAny(t, payload["error"])
			if providerContractJSONString(t, errPayload["code"]) != tc.wantCode {
				t.Fatalf("expected error code %q, got %#v", tc.wantCode, payload)
			}
		})
	}
}
