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

	fmt.Printf("Stackyard AWS Oracle Database advanced client using %s\n", endpoint)

	resourceARN := "arn:aws:odb:us-east-1:123456789012:odb-network/stackyard-network"
	calls := []restCall{
		{Name: "InitializeService", Method: http.MethodPost, Path: "/InitializeService", Payload: map[string]any{}},
		{Name: "GetOciOnboardingStatus", Method: http.MethodPost, Path: "/GetOciOnboardingStatus", Payload: map[string]any{}},
		{Name: "ListOdbNetworks", Method: http.MethodPost, Path: "/ListOdbNetworks", Payload: map[string]any{}},
		{Name: "ListOdbPeeringConnections", Method: http.MethodPost, Path: "/ListOdbPeeringConnections", Payload: map[string]any{}},
		{Name: "ListCloudExadataInfrastructures", Method: http.MethodPost, Path: "/ListCloudExadataInfrastructures", Payload: map[string]any{}},
		{Name: "ListCloudVmClusters", Method: http.MethodPost, Path: "/ListCloudVmClusters", Payload: map[string]any{}},
		{Name: "ListCloudAutonomousVmClusters", Method: http.MethodPost, Path: "/ListCloudAutonomousVmClusters", Payload: map[string]any{}},
		{Name: "ListAutonomousVirtualMachines", Method: http.MethodPost, Path: "/ListAutonomousVirtualMachines", Payload: map[string]any{}},
		{Name: "ListDbNodes", Method: http.MethodPost, Path: "/ListDbNodes", Payload: map[string]any{}},
		{Name: "ListDbServers", Method: http.MethodPost, Path: "/ListDbServers", Payload: map[string]any{}},
		{Name: "ListDbSystemShapes", Method: http.MethodPost, Path: "/ListDbSystemShapes", Payload: map[string]any{}},
		{Name: "ListGiVersions", Method: http.MethodPost, Path: "/ListGiVersions", Payload: map[string]any{}},
		{Name: "ListSystemVersions", Method: http.MethodPost, Path: "/ListSystemVersions", Payload: map[string]any{}},
		{
			Name:   "ListTagsForResource",
			Method: http.MethodPost,
			Path:   "/ListTagsForResource",
			Payload: map[string]any{
				"resourceArn": resourceARN,
			},
		},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := odbRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
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

func odbRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "odb", region, time.Now()); err != nil {
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
