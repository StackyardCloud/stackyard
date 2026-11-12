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

type faceClient struct {
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

func newFaceClient(endpoint, account, subscriptionKey string) *faceClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"face-v1.2",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &faceClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *faceClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	faceListID := getenv("STACKYARD_AZURE_FACE_LIST_ID", "list-a")
	largeFaceListID := getenv("STACKYARD_AZURE_FACE_LARGE_FACE_LIST_ID", "large-list-a")
	personGroupID := getenv("STACKYARD_AZURE_FACE_PERSON_GROUP_ID", "pg-a")
	largePersonGroupID := getenv("STACKYARD_AZURE_FACE_LARGE_PERSON_GROUP_ID", "lpg-a")

	fmt.Printf("Stackyard Azure Face (face-v1.2) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newFaceClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "DetectFaceFromUrl",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/detect?_overload=detectFromUrl&returnFaceId=true",
			payload: map[string]any{
				"url": "https://example.com/face.jpg",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateFaceList",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/facelists/" + faceListID,
			payload: map[string]any{
				"name":             "stackyard-face-list",
				"recognitionModel": "recognition_04",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetFaceLists",
			method:   http.MethodGet,
			path:     "/azure/face/v1.2/facelists?top=10",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateLargeFaceList",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/largefacelists/" + largeFaceListID,
			payload: map[string]any{
				"name":             "stackyard-large-face-list",
				"recognitionModel": "recognition_04",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "TrainLargeFaceList",
			method:   http.MethodPost,
			path:     "/azure/face/v1.2/largefacelists/" + largeFaceListID + "/train",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "FindSimilar",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/findsimilars",
			payload: map[string]any{
				"faceId":                         "face-a",
				"faceIds":                        []string{"face-b", "face-c"},
				"maxNumOfCandidatesReturned":     2,
				"mode":                           "matchFace",
				"recognitionModel":               "recognition_04",
				"returnFaceAttributes":           true,
				"returnRecognitionModel":         true,
				"returnFaceLandmarks":            false,
				"faceDetectionModel":             "detection_03",
				"faceRecognitionModelOverride":   "recognition_04",
				"faceIdentificationModelVersion": "v1.0",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateLivenessSession",
			method: http.MethodPost,
			path:   "/azure/face/v1.2/detectLiveness-sessions",
			payload: map[string]any{
				"authTokenTimeToLiveInSeconds": 300,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetSessionImage",
			method:   http.MethodGet,
			path:     "/azure/face/v1.2/sessionImages/session-image-1",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreatePersonGroup",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/persongroups/" + personGroupID,
			payload: map[string]any{
				"name":             "stackyard-person-group",
				"recognitionModel": "recognition_04",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "TrainPersonGroup",
			method:   http.MethodPost,
			path:     "/azure/face/v1.2/persongroups/" + personGroupID + "/train",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateLargePersonGroup",
			method: http.MethodPut,
			path:   "/azure/face/v1.2/largepersongroups/" + largePersonGroupID,
			payload: map[string]any{
				"name":             "stackyard-large-person-group",
				"recognitionModel": "recognition_04",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "TrainLargePersonGroup",
			method:   http.MethodPost,
			path:     "/azure/face/v1.2/largepersongroups/" + largePersonGroupID + "/train",
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
		fmt.Println("All face routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
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
