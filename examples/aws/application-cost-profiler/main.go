// Command application-cost-profiler runs AWS Application Cost Profiler
// against Stackyard: create a bucket, upload a tenant usage file, define a
// daily CSV report, import the usage, and read back the tenant-share report
// Stackyard wrote to S3.
//
// The SDK has no Application Cost Profiler module in this repository's
// go.mod, so requests are signed by hand.
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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const signingName = "application-cost-profiler"

const usage = `ApplicationId,TenantId,TenantDesc,ResourceId,StartTime,EndTime
storefront,acme,Acme Corp,i-0a1b2c3d,2026-01-01T00:00:00Z,2026-01-01T18:00:00Z
storefront,globex,Globex,i-0a1b2c3d,2026-01-01T00:00:00Z,2026-01-01T06:00:00Z
`

func main() {
	c := newClient()
	fmt.Printf("Stackyard Application Cost Profiler client using %s\n", c.endpoint)

	bucket := fmt.Sprintf("tenant-costs-%d", time.Now().UnixNano())
	c.send("CreateBucket", http.MethodPut, "s3", c.endpoint+"/"+bucket, nil, nil)
	c.send("PutObject", http.MethodPut, "s3", c.endpoint+"/"+bucket+"/usage/day1.csv", []byte(usage), map[string]string{"Content-Type": "text/csv"})
	fmt.Println("bucket:", bucket)

	reportID := fmt.Sprintf("tenant-share-%d", time.Now().UnixNano())
	c.call(http.MethodPost, "/reportDefinition", map[string]any{
		"reportId": reportID, "reportDescription": "Daily tenant share", "reportFrequency": "DAILY", "format": "CSV",
		"destinationS3Location": map[string]any{"bucket": bucket, "prefix": "reports"},
	})
	def := c.call(http.MethodGet, "/reportDefinition/"+reportID, nil)
	fmt.Printf("report definition %v: %v %v\n", def["reportId"], def["reportFrequency"], def["format"])

	importID := c.call(http.MethodPost, "/importApplicationUsage", map[string]any{
		"sourceS3Location": map[string]any{"bucket": bucket, "key": "usage/day1.csv"},
	})["importId"].(string)
	fmt.Println("import:", importID)

	key := "reports/" + reportID + "/" + time.Now().UTC().Format("2006/01/02") + "/" + importID + ".csv"
	report := c.send("GetObject", http.MethodGet, "s3", c.endpoint+"/"+bucket+"/"+key, nil, nil)
	fmt.Printf("report %s:\n%s", key, report)

	listed := c.call(http.MethodGet, "/reportDefinition", nil)["reportDefinitions"].([]any)
	fmt.Println("definitions:", len(listed))
	c.call(http.MethodDelete, "/reportDefinition/"+reportID, nil)
	fmt.Println("deleted report definition", reportID)
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// call sends one REST JSON operation and exits unless it returns 2xx.
func (c client) call(method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s %s: %v", method, path, err)
		}
	}
	raw := c.send(method+" "+path, method, signingName, c.endpoint+path, body, map[string]string{"Content-Type": "application/json"})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s %s: decode: %v", method, path, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
