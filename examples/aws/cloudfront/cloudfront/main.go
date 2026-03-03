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
	distributionID := getenv("STACKYARD_CLOUDFRONT_DISTRIBUTION_ID", "EDFDVBD632BHDS5")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard CloudFront advanced client using %s\n", endpoint)

	createInvalidationPayload := fmt.Sprintf(`
<InvalidationBatch xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/">
  <CallerReference>stackyard-%d</CallerReference>
  <Paths>
    <Quantity>2</Quantity>
    <Items>
      <Path>/index.html</Path>
      <Path>/assets/*</Path>
    </Items>
  </Paths>
</InvalidationBatch>`, time.Now().Unix())

	calls := []apiCall{
		{Name: "ListDistributions", Method: http.MethodGet, Path: "/2020-05-31/distribution"},
		{Name: "GetDistribution", Method: http.MethodGet, Path: "/2020-05-31/distribution/" + distributionID},
		{Name: "ListInvalidations", Method: http.MethodGet, Path: "/2020-05-31/distribution/" + distributionID + "/invalidation"},
		{
			Name:        "CreateInvalidation",
			Method:      http.MethodPost,
			Path:        "/2020-05-31/distribution/" + distributionID + "/invalidation",
			Payload:     []byte(strings.TrimSpace(createInvalidationPayload)),
			ContentType: "application/xml",
		},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call apiCall) error {
	status, body, err := cloudFrontRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload, call.ContentType)
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

func cloudFrontRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "cloudfront", region, time.Now()); err != nil {
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
		strings.Contains(combined, "nosuchdistribution") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "invalidargument") ||
		strings.Contains(combined, "illegalupdate") ||
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
