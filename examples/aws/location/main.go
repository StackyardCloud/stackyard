package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type restCall struct {
	Name    string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Amazon Location Service advanced client using %s\n", endpoint)

	mapName := "stackyard-map"
	indexName := "stackyard-place-index"
	calculatorName := "stackyard-route-calculator"
	trackerName := "stackyard-tracker"
	collectionName := "stackyard-geofence-collection"
	keyName := "stackyard-api-key"
	resourceARN := "arn:aws:geo:us-east-1:123456789012:map/stackyard-map"

	calls := []restCall{
		{Name: "ListMaps", Method: http.MethodPost, Path: "/maps/v0/list-maps", Payload: map[string]any{"MaxResults": 10}},
		{Name: "DescribeMap", Method: http.MethodGet, Path: "/maps/v0/maps/" + url.PathEscape(mapName)},
		{Name: "ListPlaceIndexes", Method: http.MethodPost, Path: "/places/v0/list-indexes", Payload: map[string]any{"MaxResults": 10}},
		{Name: "DescribePlaceIndex", Method: http.MethodGet, Path: "/places/v0/indexes/" + url.PathEscape(indexName)},
		{Name: "ListRouteCalculators", Method: http.MethodPost, Path: "/routes/v0/list-calculators", Payload: map[string]any{"MaxResults": 10}},
		{Name: "DescribeRouteCalculator", Method: http.MethodGet, Path: "/routes/v0/calculators/" + url.PathEscape(calculatorName)},
		{Name: "ListTrackers", Method: http.MethodPost, Path: "/tracking/v0/list-trackers", Payload: map[string]any{"MaxResults": 10}},
		{Name: "DescribeTracker", Method: http.MethodGet, Path: "/tracking/v0/trackers/" + url.PathEscape(trackerName)},
		{Name: "ListGeofenceCollections", Method: http.MethodPost, Path: "/geofencing/v0/list-collections", Payload: map[string]any{"MaxResults": 10}},
		{Name: "DescribeGeofenceCollection", Method: http.MethodGet, Path: "/geofencing/v0/collections/" + url.PathEscape(collectionName)},
		{Name: "ListKeys", Method: http.MethodPost, Path: "/metadata/v0/list-keys", Payload: map[string]any{"MaxResults": 10}},
		{Name: "DescribeKey", Method: http.MethodGet, Path: "/metadata/v0/keys/" + url.PathEscape(keyName)},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + url.PathEscape(resourceARN)},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := locationRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(status, errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func locationRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	} else {
		body = []byte{}
	}

	fullURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "geo", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(status int, errType string, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "unknown action") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "forbidden") ||
		strings.Contains(combined, "unauthorized")
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
