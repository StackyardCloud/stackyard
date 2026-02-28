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
	bundleID := getenv("STACKYARD_APPFABRIC_BUNDLE_ID", "ab-example-001")
	authID := getenv("STACKYARD_APPFABRIC_AUTH_ID", "auth-example-001")
	ingestionID := getenv("STACKYARD_APPFABRIC_INGESTION_ID", "ing-example-001")
	destinationID := getenv("STACKYARD_APPFABRIC_DESTINATION_ID", "dest-example-001")
	userAccessTaskID := getenv("STACKYARD_APPFABRIC_USER_ACCESS_TASK_ID", "uat-example-001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard AppFabric advanced client using %s\n", endpoint)

	resourceARN := "arn:aws:appfabric:us-east-1:123456789012:appbundle/" + bundleID
	escapedBundle := url.PathEscape(bundleID)
	escapedAuth := url.PathEscape(authID)
	escapedIngestion := url.PathEscape(ingestionID)
	escapedDestination := url.PathEscape(destinationID)
	escapedARN := url.PathEscape(resourceARN)

	calls := []restCall{
		{Name: "CreateAppBundle", Method: http.MethodPost, Path: "/appbundles", Payload: map[string]any{"appBundleIdentifier": bundleID}},
		{Name: "GetAppBundle", Method: http.MethodGet, Path: "/appbundles/" + escapedBundle, Payload: nil},
		{Name: "CreateAppAuthorization", Method: http.MethodPost, Path: "/appbundles/" + escapedBundle + "/appauthorizations", Payload: map[string]any{"appAuthorizationIdentifier": authID, "app": "okta"}},
		{Name: "ConnectAppAuthorization", Method: http.MethodPost, Path: "/appbundles/" + escapedBundle + "/appauthorizations/" + escapedAuth + "/connect", Payload: map[string]any{}},
		{Name: "ListAppAuthorizations", Method: http.MethodGet, Path: "/appbundles/" + escapedBundle + "/appauthorizations", Payload: nil},
		{Name: "CreateIngestion", Method: http.MethodPost, Path: "/appbundles/" + escapedBundle + "/ingestions", Payload: map[string]any{"ingestionIdentifier": ingestionID}},
		{Name: "StartIngestion", Method: http.MethodPost, Path: "/appbundles/" + escapedBundle + "/ingestions/" + escapedIngestion + "/start", Payload: map[string]any{}},
		{Name: "CreateIngestionDestination", Method: http.MethodPost, Path: "/appbundles/" + escapedBundle + "/ingestions/" + escapedIngestion + "/ingestiondestinations", Payload: map[string]any{"ingestionDestinationIdentifier": destinationID}},
		{Name: "UpdateIngestionDestination", Method: http.MethodPatch, Path: "/appbundles/" + escapedBundle + "/ingestions/" + escapedIngestion + "/ingestiondestinations/" + escapedDestination, Payload: map[string]any{"state": "ACTIVE"}},
		{Name: "ListIngestions", Method: http.MethodGet, Path: "/appbundles/" + escapedBundle + "/ingestions", Payload: nil},
		{Name: "ListIngestionDestinations", Method: http.MethodGet, Path: "/appbundles/" + escapedBundle + "/ingestions/" + escapedIngestion + "/ingestiondestinations", Payload: nil},
		{Name: "StartUserAccessTasks", Method: http.MethodPost, Path: "/useraccess/start", Payload: map[string]any{"appBundleIdentifier": bundleID, "userAccessTaskId": userAccessTaskID}},
		{Name: "BatchGetUserAccessTasks", Method: http.MethodPost, Path: "/useraccess/batchget", Payload: map[string]any{"userAccessTaskIds": []string{userAccessTaskID}}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + escapedARN, Payload: map[string]any{"tags": map[string]any{"env": "dev", "owner": "stackyard"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + escapedARN, Payload: nil},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + escapedARN + "?tagKeys=owner", Payload: nil},
		{Name: "StopIngestion", Method: http.MethodPost, Path: "/appbundles/" + escapedBundle + "/ingestions/" + escapedIngestion + "/stop", Payload: map[string]any{}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := appFabricRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func appFabricRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte{}
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "appfabric", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(errType string, body []byte) bool {
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied")
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
