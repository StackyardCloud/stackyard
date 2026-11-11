package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const communicationPhoneNumbersAPIVersion = "2025-06-01"

type communicationPhoneNumbersClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationPhoneNumbersSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationPhoneNumbersSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationPhoneNumbersClient(endpoint, account, subscriptionKey string) *communicationPhoneNumbersClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-phonenumbers-2025-06-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationPhoneNumbersSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationPhoneNumbersClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationPhoneNumbersClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Raw().Header.Set(key, value)
	}
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, resp.Header, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, resp.Header, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_COMMUNICATION_PHONENUMBERS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_PHONENUMBERS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Phone Numbers example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationPhoneNumbersClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-phonenumbers/2025-06-01"

	countries, status, _, err := client.doJSON(ctx, http.MethodGet, base+"/availablePhoneNumbers/countries?api-version="+communicationPhoneNumbersAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListAvailableCountries failed: %v", err)
	}
	printJSON("ListAvailableCountries", status, countries)

	searchPayload, searchStatus, searchHeaders, err := client.doJSON(ctx, http.MethodPost, base+"/availablePhoneNumbers/countries/US/:search?api-version="+communicationPhoneNumbersAPIVersion, map[string]any{
		"phoneNumberType": "geographic",
		"assignmentType":  "person",
		"capabilities": map[string]any{
			"calling": "inbound+outbound",
			"sms":     "inbound+outbound",
		},
		"areaCode": "425",
		"quantity": 1,
	}, nil, http.StatusAccepted)
	if err != nil {
		fatalf("SearchAvailablePhoneNumbers failed: %v", err)
	}
	printJSON("SearchAvailablePhoneNumbers", searchStatus, searchPayload)

	operationLocation := searchHeaders.Get("Operation-Location")
	for attempt := 1; attempt <= 3; attempt++ {
		pollPayload, pollStatus, _, err := client.doJSON(ctx, http.MethodGet, operationLocation, nil, nil, http.StatusOK)
		if err != nil {
			fatalf("GetOperation failed: %v", err)
		}
		printJSON(fmt.Sprintf("GetOperation attempt %d", attempt), pollStatus, pollPayload)
		if stringValue(pollPayload["status"]) != "running" {
			break
		}
	}

	searchID := stringValue(searchPayload["searchId"])
	resultPayload, resultStatus, _, err := client.doJSON(ctx, http.MethodGet, base+"/availablePhoneNumbers/searchResults/"+searchID+"?api-version="+communicationPhoneNumbersAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetSearchResult failed: %v", err)
	}
	printJSON("GetSearchResult", resultStatus, resultPayload)

	searchResults := anySlice(resultPayload["phoneNumbers"])
	if len(searchResults) == 0 {
		fatalf("search result did not include any phone numbers")
	}
	firstNumber := stringValue(mapValue(searchResults[0])["phoneNumber"])
	reservationID := "example-reservation-001"

	reservationPayload, reservationStatus, _, err := client.doJSON(ctx, http.MethodPatch, base+"/availablePhoneNumbers/reservations/"+reservationID+"?api-version="+communicationPhoneNumbersAPIVersion, map[string]any{
		"searchId":     searchID,
		"displayName":  "Example Reservation",
		"phoneNumbers": []string{firstNumber},
	}, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		fatalf("CreateOrUpdateReservation failed: %v", err)
	}
	printJSON("CreateOrUpdateReservation", reservationStatus, reservationPayload)

	purchasePayload, purchaseStatus, purchaseHeaders, err := client.doJSON(ctx, http.MethodPost, base+"/availablePhoneNumbers/reservations/"+reservationID+"/:purchase?api-version="+communicationPhoneNumbersAPIVersion, nil, nil, http.StatusAccepted)
	if err != nil {
		fatalf("PurchaseReservation failed: %v", err)
	}
	printJSON("PurchaseReservation", purchaseStatus, purchasePayload)

	purchaseOperation := purchaseHeaders.Get("Operation-Location")
	for attempt := 1; attempt <= 3; attempt++ {
		pollPayload, pollStatus, _, err := client.doJSON(ctx, http.MethodGet, purchaseOperation, nil, nil, http.StatusOK)
		if err != nil {
			fatalf("Poll purchase operation failed: %v", err)
		}
		printJSON(fmt.Sprintf("PurchaseOperation attempt %d", attempt), pollStatus, pollPayload)
		if stringValue(pollPayload["status"]) != "running" {
			break
		}
	}

	listPayload, listStatus, _, err := client.doJSON(ctx, http.MethodGet, base+"/phoneNumbers?api-version="+communicationPhoneNumbersAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListPhoneNumbers failed: %v", err)
	}
	printJSON("ListPhoneNumbers", listStatus, listPayload)
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func anySlice(value any) []any {
	items, _ := value.([]any)
	return items
}

func mapValue(value any) map[string]any {
	item, _ := value.(map[string]any)
	if item == nil {
		return map[string]any{}
	}
	return item
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("format %s response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(formatted))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
