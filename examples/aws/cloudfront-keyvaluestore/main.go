// Command cloudfront-keyvaluestore creates a CloudFront KeyValueStore with the
// CloudFront control plane, then reads and writes keys through the
// cloudfront-keyvaluestore data plane using ETag optimistic concurrency.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint string
	region   string
	creds    aws.CredentialsProvider
	http     *http.Client
}

type response struct {
	status int
	etag   string
	body   []byte
}

func main() {
	c := client{
		endpoint: getenv("STACKYARD_ENDPOINT", "http://localhost:4566"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
		http: &http.Client{Timeout: 15 * time.Second},
	}
	ctx := context.Background()
	name := fmt.Sprintf("example-kvs-%d", time.Now().UnixNano())
	fmt.Printf("Stackyard CloudFront KeyValueStore example using %s\n", c.endpoint)

	created := c.must(ctx, "CreateKeyValueStore", "cloudfront", http.MethodPost, "/2020-05-31/key-value-store", "",
		[]byte(`<CreateKeyValueStoreRequest xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/"><Name>`+name+`</Name><Comment>example</Comment></CreateKeyValueStoreRequest>`))
	var kvs struct{ ARN string }
	if err := xml.Unmarshal(created.body, &kvs); err != nil || kvs.ARN == "" {
		exitf("CreateKeyValueStore returned no ARN: %s", created.body)
	}
	storeETag := created.etag
	fmt.Printf("created key value store %s\n", kvs.ARN)

	base := "/key-value-stores/" + url.PathEscape(kvs.ARN)
	etag := c.must(ctx, "DescribeKeyValueStore", "cloudfront-keyvaluestore", http.MethodGet, base, "", nil).etag

	put := c.must(ctx, "PutKey", "cloudfront-keyvaluestore", http.MethodPut, base+"/keys/"+url.PathEscape("feature/flag"), etag, []byte(`{"Value":"on"}`))
	if stale, _ := c.do(ctx, "cloudfront-keyvaluestore", http.MethodPut, base+"/keys/other", etag, []byte(`{"Value":"x"}`)); stale.status != http.StatusConflict {
		exitf("expected 409 for a stale If-Match, got %d", stale.status)
	}
	fmt.Println("stale If-Match rejected with 409 ConflictException")

	updated := c.must(ctx, "UpdateKeys", "cloudfront-keyvaluestore", http.MethodPost, base+"/keys", put.etag,
		[]byte(`{"Puts":[{"Key":"greeting","Value":"hello"},{"Key":"color","Value":"blue"}],"Deletes":[{"Key":"feature/flag"}]}`))

	got := c.must(ctx, "GetKey", "cloudfront-keyvaluestore", http.MethodGet, base+"/keys/greeting", "", nil)
	fmt.Printf("GetKey greeting: %s\n", got.body)
	list := c.must(ctx, "ListKeys", "cloudfront-keyvaluestore", http.MethodGet, base+"/keys?MaxResults=10", "", nil)
	fmt.Printf("ListKeys: %s\n", list.body)

	c.must(ctx, "DeleteKey", "cloudfront-keyvaluestore", http.MethodDelete, base+"/keys/color", updated.etag, nil)
	c.must(ctx, "DeleteKeyValueStore", "cloudfront", http.MethodDelete, "/2020-05-31/key-value-store/"+name, storeETag, nil)
	fmt.Println("CloudFront KeyValueStore example complete")
}

func (c client) must(ctx context.Context, op, service, method, path, ifMatch string, body []byte) response {
	resp, err := c.do(ctx, service, method, path, ifMatch, body)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if resp.status < 200 || resp.status > 299 {
		exitf("%s: status %d: %s", op, resp.status, resp.body)
	}
	return resp
}

// do signs with SigV4; Stackyard also accepts the SigV4A signatures that the
// AWS SDKs use for cloudfront-keyvaluestore.
func (c client) do(ctx context.Context, service, method, path, ifMatch string, body []byte) (response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return response{}, err
	}
	if service == "cloudfront" {
		req.Header.Set("Content-Type", "application/xml")
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		return response{}, err
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hash, service, c.region, time.Now().UTC()); err != nil {
		return response{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return response{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return response{}, err
	}
	return response{status: resp.StatusCode, etag: resp.Header.Get("ETag"), body: data}, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
