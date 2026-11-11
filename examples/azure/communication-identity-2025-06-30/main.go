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

const communicationIdentityAPIVersion = "2025-06-30"

type communicationIdentityClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationIdentitySharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationIdentitySharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationIdentityClient(endpoint, account, subscriptionKey string) *communicationIdentityClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-identity-2025-06-30",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationIdentitySharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationIdentityClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationIdentityClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
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
	account := getenv("STACKYARD_AZURE_COMMUNICATION_IDENTITY_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_IDENTITY_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Identity example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationIdentityClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-identity/2025-06-30"

	createPayload, status, _, err := client.doJSON(ctx, http.MethodPost, base+"/identities?api-version="+communicationIdentityAPIVersion, map[string]any{
		"createTokenWithScopes": []string{"chat"},
		"expiresInMinutes":      120,
	}, nil, http.StatusCreated)
	if err != nil {
		fatalf("CreateIdentity failed: %v", err)
	}
	printJSON("CreateIdentity", status, createPayload)

	identity := createPayload["identity"].(map[string]any)
	identityID := fmt.Sprintf("%v", identity["id"])

	issuePayload, status, _, err := client.doJSON(ctx, http.MethodPost, base+"/identities/"+identityID+"/:issueAccessToken?api-version="+communicationIdentityAPIVersion, map[string]any{
		"scopes":           []string{"chat.join", "voip.join"},
		"expiresInMinutes": 180,
	}, nil, http.StatusOK)
	if err != nil {
		fatalf("IssueAccessToken failed: %v", err)
	}
	printJSON("IssueAccessToken", status, issuePayload)

	upsertAssignmentPayload, status, _, err := client.doJSON(ctx, http.MethodPut, base+"/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version="+communicationIdentityAPIVersion, map[string]any{
		"principalType": "resourceAccount",
		"clientIds":     []string{"client-a", "client-b"},
	}, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		fatalf("UpsertTeamsExtensionAssignment failed: %v", err)
	}
	printJSON("UpsertTeamsExtensionAssignment", status, upsertAssignmentPayload)

	getAssignmentPayload, status, _, err := client.doJSON(ctx, http.MethodGet, base+"/access/teamsExtension/tenants/tenant-123/assignments/object-123?api-version="+communicationIdentityAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetTeamsExtensionAssignment failed: %v", err)
	}
	printJSON("GetTeamsExtensionAssignment", status, getAssignmentPayload)

	exchangePayload, status, _, err := client.doJSON(ctx, http.MethodPost, base+"/teamsUser/:exchangeAccessToken?api-version="+communicationIdentityAPIVersion, map[string]any{
		"token":  "teams-user-token",
		"appId":  "app-123",
		"userId": "user-123",
	}, nil, http.StatusOK)
	if err != nil {
		fatalf("ExchangeTeamsUserAccessToken failed: %v", err)
	}
	printJSON("ExchangeTeamsUserAccessToken", status, exchangePayload)
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
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
