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

type requestSpec struct {
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

	fmt.Printf("Stackyard AppConfig advanced client using %s\n", endpoint)

	requests := []requestSpec{
		{Name: "CreateApplication", Method: http.MethodPost, Path: "/applications", Payload: map[string]any{"Name": "stackyard-appconfig-app"}},
		{Name: "ListApplications", Method: http.MethodGet, Path: "/applications", Payload: nil},
		{Name: "CreateDeploymentStrategy", Method: http.MethodPost, Path: "/deploymentstrategies", Payload: map[string]any{"Name": "stackyard-strategy", "DeploymentDurationInMinutes": 0, "FinalBakeTimeInMinutes": 0, "GrowthFactor": 25.0, "ReplicateTo": "NONE"}},
		{Name: "ListDeploymentStrategies", Method: http.MethodGet, Path: "/deploymentstrategies", Payload: nil},
		{Name: "GetAccountSettings", Method: http.MethodGet, Path: "/settings", Payload: nil},
	}

	for _, reqSpec := range requests {
		if err := runCall(ctx, endpoint, region, creds, reqSpec); err != nil {
			exitf("%s failed: %v", reqSpec.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, spec requestSpec) error {
	var payloadBytes []byte
	if spec.Payload != nil {
		encoded, err := json.Marshal(spec.Payload)
		if err != nil {
			return err
		}
		payloadBytes = encoded
	}

	status, body, err := appConfigRequest(ctx, endpoint, region, creds, spec.Method, spec.Path, payloadBytes)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", spec.Name, status)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func appConfigRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "appconfig", region, time.Now()); err != nil {
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
