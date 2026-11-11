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

const communicationRoomsAPIVersion = "2025-03-13"

type communicationRoomsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationRoomsSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationRoomsSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationRoomsClient(endpoint, account, subscriptionKey string) *communicationRoomsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-rooms-2025-03-13",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationRoomsSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationRoomsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationRoomsClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Raw().Header.Set(key, value)
	}
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_COMMUNICATION_ROOMS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_ROOMS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Rooms example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationRoomsClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-rooms/2025-03-13/rooms"
	repeatabilityHeaders := map[string]string{
		"Repeatability-Request-ID": "6f9619ff-8b86-d011-b42d-00cf4fc964ff",
		"Repeatability-First-Sent": "Sun, 12 Apr 2026 09:00:00 GMT",
	}

	createPayload, createStatus, err := client.doJSON(ctx, http.MethodPost, base+"?api-version="+communicationRoomsAPIVersion, map[string]any{
		"validFrom":          "2026-04-12T09:00:00Z",
		"validUntil":         "2026-04-13T09:00:00Z",
		"pstnDialOutEnabled": true,
		"participants": []map[string]any{
			{"rawId": "8:acs:user-a", "role": "Presenter"},
		},
	}, repeatabilityHeaders, http.StatusCreated)
	if err != nil {
		fatalf("CreateRoom failed: %v", err)
	}
	printJSON("CreateRoom", createStatus, createPayload)
	roomID := stringValue(createPayload["id"])

	listPayload, listStatus, err := client.doJSON(ctx, http.MethodGet, base+"?api-version="+communicationRoomsAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListRooms failed: %v", err)
	}
	printJSON("ListRooms", listStatus, listPayload)

	getPayload, getStatus, err := client.doJSON(ctx, http.MethodGet, base+"/"+roomID+"?api-version="+communicationRoomsAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetRoom failed: %v", err)
	}
	printJSON("GetRoom", getStatus, getPayload)

	updatePayload, updateStatus, err := client.doJSON(ctx, http.MethodPatch, base+"/"+roomID+"?api-version="+communicationRoomsAPIVersion, map[string]any{
		"pstnDialOutEnabled": false,
		"validUntil":         "2026-04-14T09:00:00Z",
	}, nil, http.StatusOK)
	if err != nil {
		fatalf("UpdateRoom failed: %v", err)
	}
	printJSON("UpdateRoom", updateStatus, updatePayload)

	participantsPayload, participantsStatus, err := client.doJSON(ctx, http.MethodGet, base+"/"+roomID+"/participants?api-version="+communicationRoomsAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListParticipants failed: %v", err)
	}
	printJSON("ListParticipants", participantsStatus, participantsPayload)

	patchParticipantsPayload, patchParticipantsStatus, err := client.doJSON(ctx, http.MethodPatch, base+"/"+roomID+"/participants?api-version="+communicationRoomsAPIVersion, map[string]any{
		"participants": map[string]any{
			"8:acs:user-b": map[string]any{"role": "Attendee"},
			"8:acs:user-a": nil,
		},
	}, nil, http.StatusOK)
	if err != nil {
		fatalf("PatchParticipants failed: %v", err)
	}
	printJSON("PatchParticipants", patchParticipantsStatus, patchParticipantsPayload)

	participantsPayload, participantsStatus, err = client.doJSON(ctx, http.MethodGet, base+"/"+roomID+"/participants?api-version="+communicationRoomsAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("ListParticipants after patch failed: %v", err)
	}
	printJSON("ListParticipantsAfterPatch", participantsStatus, participantsPayload)

	_, deleteStatus, err := client.doJSON(ctx, http.MethodDelete, base+"/"+roomID+"?api-version="+communicationRoomsAPIVersion, nil, nil, http.StatusNoContent)
	if err != nil {
		fatalf("DeleteRoom failed: %v", err)
	}
	fmt.Printf("\n== DeleteRoom (%d) ==\n{}\n", deleteStatus)
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
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
