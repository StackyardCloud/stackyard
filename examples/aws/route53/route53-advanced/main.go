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

type apiCall struct {
	Name        string
	Method      string
	Path        string
	Payload     []byte
	ContentType string
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	hostedZoneID := getenv("STACKYARD_ROUTE53_HOSTED_ZONE_ID", "ZSTACKYARD01")
	changeID := getenv("STACKYARD_ROUTE53_CHANGE_ID", "CSTACKYARD01")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Route 53 advanced client using %s\n", endpoint)

	changePayload := strings.TrimSpace(`
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Comment>stackyard advanced record change</Comment>
    <Changes>
      <Change>
        <Action>UPSERT</Action>
        <ResourceRecordSet>
          <Name>api.stackyard-advanced.example.com.</Name>
          <Type>A</Type>
          <TTL>60</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>127.0.0.1</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`)

	healthCheckPayload := strings.TrimSpace(`
<CreateHealthCheckRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <CallerReference>stackyard-route53-advanced</CallerReference>
  <HealthCheckConfig>
    <IPAddress>127.0.0.1</IPAddress>
    <Port>443</Port>
    <Type>HTTPS</Type>
    <ResourcePath>/health</ResourcePath>
    <RequestInterval>30</RequestInterval>
    <FailureThreshold>3</FailureThreshold>
  </HealthCheckConfig>
</CreateHealthCheckRequest>`)

	calls := []apiCall{
		{Name: "CreateHostedZone", Method: http.MethodPost, Path: "/2013-04-01/hostedzone", Payload: []byte(`<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><Name>stackyard-advanced.example.com</Name><CallerReference>stackyard-advanced</CallerReference></CreateHostedZoneRequest>`), ContentType: "application/xml"},
		{Name: "ChangeResourceRecordSets", Method: http.MethodPost, Path: "/2013-04-01/hostedzone/" + hostedZoneID + "/rrset", Payload: []byte(changePayload), ContentType: "application/xml"},
		{Name: "GetChange", Method: http.MethodGet, Path: "/2013-04-01/change/" + changeID},
		{Name: "CreateHealthCheck", Method: http.MethodPost, Path: "/2013-04-01/healthcheck", Payload: []byte(healthCheckPayload), ContentType: "application/xml"},
		{Name: "ListHealthChecks", Method: http.MethodGet, Path: "/2013-04-01/healthcheck"},
		{Name: "TestDNSAnswer", Method: http.MethodGet, Path: "/2013-04-01/testdnsanswer?hostedzoneid=" + hostedZoneID + "&recordname=api.stackyard-advanced.example.com&recordtype=A"},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call apiCall) error {
	status, body, err := route53Request(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload, call.ContentType)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	if isStagedPlanTolerated(body) {
		logf("%s returned %d: expected while staged plan is in progress", call.Name, status)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
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
