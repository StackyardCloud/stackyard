package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type apiCall struct {
	Action string
	Method string
	Path   string
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

	fmt.Printf("Stackyard Audit Manager advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Action: "GetAccountStatus", Method: http.MethodGet, Path: "/account/status"},
		{Action: "GetOrganizationAdminAccount", Method: http.MethodGet, Path: "/account/organizationAdminAccount"},
		{Action: "ListAssessments", Method: http.MethodGet, Path: "/assessments"},
		{Action: "ListAssessmentFrameworks", Method: http.MethodGet, Path: "/assessmentFrameworks"},
		{Action: "ListControls", Method: http.MethodGet, Path: "/controls"},
		{Action: "ListNotifications", Method: http.MethodGet, Path: "/notifications"},
	}

	for _, call := range calls {
		status, body, err := apiRequest(ctx, endpoint, region, creds, call.Method, call.Path)
		if err != nil {
			exitf("%s request failed: %v", call.Action, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Action, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s returned %d\n", call.Action, status)
	}

	fmt.Println("Done.")
}

func apiRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, requestPath string,
) (int, []byte, error) {
	url := strings.TrimRight(endpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	emptyBodyHash := hashSHA256(nil)
	if err := signer.SignHTTP(ctx, credValue, req, emptyBodyHash, "auditmanager", region, time.Now()); err != nil {
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
