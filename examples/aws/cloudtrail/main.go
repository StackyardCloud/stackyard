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

type call struct {
	Action  string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	trailName := getenv("STACKYARD_TRAIL_NAME", "stackyard-cloudtrail")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	calls := []call{
		{Action: "CreateTrail", Payload: map[string]any{"Name": trailName, "S3BucketName": "stackyard-cloudtrail-bucket"}},
		{Action: "DescribeTrails", Payload: map[string]any{"trailNameList": []string{trailName}}},
		{Action: "StartLogging", Payload: map[string]any{"Name": trailName}},
		{Action: "LookupEvents", Payload: map[string]any{"MaxResults": 1}},
		{Action: "StopLogging", Payload: map[string]any{"Name": trailName}},
		{Action: "DeleteTrail", Payload: map[string]any{"Name": trailName}},
	}

	fmt.Printf("Stackyard CloudTrail advanced client using %s\n", endpoint)
	for _, c := range calls {
		status, body, err := cloudTrailRequest(ctx, endpoint, region, creds, c.Action, c.Payload)
		if err != nil {
			exitf("%s failed: %v", c.Action, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", c.Action, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s succeeded\n", c.Action)
	}

	fmt.Println("Done.")
}

func cloudTrailRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, action string, payload map[string]any) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "com.amazonaws.cloudtrail.v20131101.CloudTrail_20131101."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "cloudtrail", region, time.Now()); err != nil {
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
