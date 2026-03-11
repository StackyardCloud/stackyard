package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const speakerRecognitionAPIVersion = "2021-09-05"

type speakerRecognitionClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyAndSubscriptionPolicy struct {
	account         string
	subscriptionKey string
}

func (p *sharedKeyAndSubscriptionPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newSpeakerRecognitionClient(endpoint, account, subscriptionKey string) *speakerRecognitionClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"speaker-recognition-2021-09-05",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &speakerRecognitionClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *speakerRecognitionClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}

	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	locale := getenv("STACKYARD_AZURE_SPEAKER_RECOGNITION_LOCALE", "en-us")
	tdProfileID := getenv("STACKYARD_AZURE_SPEAKER_RECOGNITION_TEXT_DEPENDENT_PROFILE_ID", "td-profile-a")
	tiProfileID := getenv("STACKYARD_AZURE_SPEAKER_RECOGNITION_TEXT_INDEPENDENT_PROFILE_ID", "ti-profile-a")
	tivProfileID := getenv("STACKYARD_AZURE_SPEAKER_RECOGNITION_TEXT_INDEPENDENT_VERIFICATION_PROFILE_ID", "tiv-profile-a")

	fmt.Printf("Stackyard Azure Speaker Recognition (speaker-recognition-2021-09-05) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newSpeakerRecognitionClient(endpoint, account, subscriptionKey)
	profileIDs := url.QueryEscape(strings.Join([]string{tiProfileID, "ti-profile-b"}, ","))

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateTextDependentProfile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles?api-version=" + speakerRecognitionAPIVersion,
			payload: map[string]any{
				"locale": locale,
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:     "ListTextDependentPhrases",
			method:   http.MethodGet,
			path:     "/azure/speaker-recognition/verification/text-dependent/phrases/" + locale + "?api-version=" + speakerRecognitionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "VerifyTextDependentProfile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-dependent/profiles/" + tdProfileID + ":verify?api-version=" + speakerRecognitionAPIVersion,
			payload: map[string]any{
				"audioData": "binary-audio-data",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateTextIndependentProfile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/identification/text-independent/profiles?api-version=" + speakerRecognitionAPIVersion,
			payload: map[string]any{
				"locale": locale,
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:   "IdentifySingleSpeaker",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/identification/text-independent/profiles:identifySingleSpeaker?api-version=" + speakerRecognitionAPIVersion + "&profileIds=" + profileIDs,
			payload: map[string]any{
				"audioData": "binary-audio-data",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTextIndependentActivationPhrases",
			method:   http.MethodGet,
			path:     "/azure/speaker-recognition/identification/text-independent/phrases/" + locale + "?api-version=" + speakerRecognitionAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateTextIndependentVerificationProfile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-independent/profiles?api-version=" + speakerRecognitionAPIVersion,
			payload: map[string]any{
				"locale": locale,
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusNotImplemented},
		},
		{
			name:   "VerifyTextIndependentVerificationProfile",
			method: http.MethodPost,
			path:   "/azure/speaker-recognition/verification/text-independent/profiles/" + tivProfileID + ":verify?api-version=" + speakerRecognitionAPIVersion,
			payload: map[string]any{
				"audioData": "binary-audio-data",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.path)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All speaker-recognition routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func notImplemented(path string) {
	fmt.Printf("Route is recognized but not implemented yet: %s\n", path)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
