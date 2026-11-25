// Command mediastore walks an AWS Elemental MediaStore container lifecycle
// against Stackyard: create a container, wait for ACTIVE, set its CORS,
// container and metric policies, tag it, list containers, then delete it.
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

func main() {
	c := newClient()
	fmt.Printf("Stackyard MediaStore client using %s\n", c.endpoint)
	name := fmt.Sprintf("stackyard-example-%d", time.Now().UnixNano())

	created := c.mediastore("CreateContainer", map[string]any{"ContainerName": name, "Tags": []map[string]string{{"Key": "env", "Value": "dev"}}})["Container"].(map[string]any)
	fmt.Println("Created:", created["ARN"], created["Status"])
	got := c.mediastore("DescribeContainer", map[string]any{"ContainerName": name})["Container"].(map[string]any)
	fmt.Println("Status:", got["Status"], "Endpoint:", got["Endpoint"])

	c.mediastore("PutCorsPolicy", map[string]any{"ContainerName": name, "CorsPolicy": []map[string]any{
		{"AllowedOrigins": []string{"*"}, "AllowedMethods": []string{"GET", "HEAD"}, "AllowedHeaders": []string{"*"}, "MaxAgeSeconds": 3000},
	}})
	policy := `{"Version":"2012-10-17","Statement":[{"Sid":"PublicRead","Effect":"Allow","Principal":"*","Action":["mediastore:GetObject","mediastore:DescribeObject"],"Resource":"*"}]}`
	c.mediastore("PutContainerPolicy", map[string]any{"ContainerName": name, "Policy": policy})
	c.mediastore("PutMetricPolicy", map[string]any{"ContainerName": name, "MetricPolicy": map[string]any{"ContainerLevelMetrics": "ENABLED"}})
	fmt.Println("CORS rules:", len(c.mediastore("GetCorsPolicy", map[string]any{"ContainerName": name})["CorsPolicy"].([]any)))
	fmt.Println("Policy round-trips:", c.mediastore("GetContainerPolicy", map[string]any{"ContainerName": name})["Policy"] == policy)

	c.mediastore("TagResource", map[string]any{"Resource": created["ARN"], "Tags": []map[string]string{{"Key": "team", "Value": "video"}}})
	fmt.Println("Tags:", c.mediastore("ListTagsForResource", map[string]any{"Resource": created["ARN"]})["Tags"])
	fmt.Println("Containers:", len(c.mediastore("ListContainers", map[string]any{})["Containers"].([]any)))

	c.mediastore("DeleteContainer", map[string]any{"ContainerName": name})
	fmt.Println("Deleted", name)
}

// mediastore calls a MediaStore control-plane (JSON 1.1) operation.
func (c client) mediastore(op string, payload any) map[string]any {
	body, _ := json.Marshal(payload)
	_, raw := c.send(op, "mediastore", http.MethodPost, c.endpoint+"/", body,
		"Content-Type", "application/x-amz-json-1.1", "X-Amz-Target", "MediaStore_20170901."+op)
	return decode(op, raw)
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

// send signs a request for service and exits unless it returns 2xx. Extra
// headers come in name/value pairs.
func (c client) send(op, service, method, target string, body []byte, headers ...string) (http.Header, []byte) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
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
	return resp.Header, raw
}

func decode(op string, raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
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
