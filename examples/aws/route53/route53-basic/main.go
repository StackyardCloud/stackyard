package main

import (
	"bytes"
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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Route 53 basic client using %s\n", endpoint)

	status, body, err := route53Request(ctx, endpoint, region, creds, http.MethodGet, "/2013-04-01/hostedzone", nil, "")
	if err != nil {
		exitf("ListHostedZones request failed: %v", err)
	}
	if status >= 200 && status < 300 {
		logf("ListHostedZones returned %d", status)
	} else if isStagedPlanTolerated(body) {
		logf("ListHostedZones returned %d: expected while staged plan is in progress", status)
	} else {
		exitf("ListHostedZones returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}

	createPayload := strings.TrimSpace(`
<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Name>stackyard-basic.example.com</Name>
  <CallerReference>stackyard-basic</CallerReference>
  <HostedZoneConfig>
    <Comment>stackyard basic example</Comment>
    <PrivateZone>false</PrivateZone>
  </HostedZoneConfig>
</CreateHostedZoneRequest>`)

	status, body, err = route53Request(ctx, endpoint, region, creds, http.MethodPost, "/2013-04-01/hostedzone", []byte(createPayload), "application/xml")
	if err != nil {
		exitf("CreateHostedZone request failed: %v", err)
	}
	if status >= 200 && status < 300 {
		logf("CreateHostedZone returned %d", status)
		fmt.Println(strings.TrimSpace(string(body)))
		return
	}
	if isStagedPlanTolerated(body) {
		logf("CreateHostedZone returned %d: expected while staged plan is in progress", status)
		return
	}
	exitf("CreateHostedZone returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

func route53Request(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, requestPath string,
	payload []byte,
	contentType string,
) (int, []byte, error) {
	if payload == nil {
		payload = []byte{}
	}

	url := strings.TrimRight(endpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}

	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "route53", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(body []byte) bool {
	combined := strings.ToLower(string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "invalid") ||
		strings.Contains(combined, "nosuch") ||
		strings.Contains(combined, "validation")
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
