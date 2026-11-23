// Command opensearch-serverless walks an OpenSearch Serverless setup against
// Stackyard: an encryption policy that covers the collection name, a network
// and a data access policy, the collection itself (CREATING, then ACTIVE on
// the next read), an index, and a retention policy whose effective rule is
// looked up for that index. Everything is deleted at the end.
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

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard OpenSearch Serverless client using %s\n", c.endpoint)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	name := "logs-" + suffix

	enc := map[string]any{"Rules": []any{map[string]any{"ResourceType": "collection", "Resource": []any{"collection/" + name}}}, "AWSOwnedKey": true}
	c.call("CreateSecurityPolicy", map[string]any{"type": "encryption", "name": "enc-" + suffix, "policy": jsonText(enc)})
	net := []any{map[string]any{"Rules": []any{map[string]any{"ResourceType": "collection", "Resource": []any{"collection/" + name}}}, "AllowFromPublic": true}}
	c.call("CreateSecurityPolicy", map[string]any{"type": "network", "name": "net-" + suffix, "policy": jsonText(net)})
	data := []any{map[string]any{
		"Rules":     []any{map[string]any{"ResourceType": "index", "Resource": []any{"index/" + name + "/*"}, "Permission": []any{"aoss:ReadDocument", "aoss:WriteDocument"}}},
		"Principal": []any{"arn:aws:iam::123456789012:root"},
	}}
	c.call("CreateAccessPolicy", map[string]any{"type": "data", "name": "data-" + suffix, "policy": jsonText(data)})
	fmt.Println("Policies: encryption, network, data")

	created := c.call("CreateCollection", map[string]any{"name": name, "type": "SEARCH"})["createCollectionDetail"].(map[string]any)
	id := created["id"].(string)
	fmt.Println("Collection:", id, created["status"])
	got := c.call("BatchGetCollection", map[string]any{"ids": []any{id}})["collectionDetails"].([]any)[0].(map[string]any)
	fmt.Println("After read:", got["status"], got["collectionEndpoint"])

	c.call("CreateIndex", map[string]any{"id": id, "indexName": "books", "indexSchema": map[string]any{
		"mappings": map[string]any{"properties": map[string]any{"title": map[string]any{"type": "text"}}},
	}})
	schema := c.call("GetIndex", map[string]any{"id": id, "indexName": "books"})["indexSchema"]
	fmt.Println("Index schema:", jsonText(schema))

	retention := map[string]any{"Rules": []any{map[string]any{"ResourceType": "index", "Resource": []any{"index/" + name + "/*"}, "MinIndexRetention": "15d"}}}
	c.call("CreateLifecyclePolicy", map[string]any{"type": "retention", "name": "keep-" + suffix, "policy": jsonText(retention)})
	eff := c.call("BatchGetEffectiveLifecyclePolicy", map[string]any{"resourceIdentifiers": []any{
		map[string]any{"type": "retention", "resource": "index/" + name + "/books"},
	}})["effectiveLifecyclePolicyDetails"].([]any)
	fmt.Println("Effective retention:", eff[0].(map[string]any)["retentionPeriod"])

	c.call("DeleteIndex", map[string]any{"id": id, "indexName": "books"})
	c.call("DeleteCollection", map[string]any{"id": id})
	c.call("DeleteLifecyclePolicy", map[string]any{"type": "retention", "name": "keep-" + suffix})
	c.call("DeleteAccessPolicy", map[string]any{"type": "data", "name": "data-" + suffix})
	c.call("DeleteSecurityPolicy", map[string]any{"type": "network", "name": "net-" + suffix})
	c.call("DeleteSecurityPolicy", map[string]any{"type": "encryption", "name": "enc-" + suffix})
	fmt.Println("Deleted collection and policies")
}

func jsonText(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		exitf("marshal: %v", err)
	}
	return string(b)
}

// call sends one signed awsJson1_0 request and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "OpenSearchServerless."+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "aoss", c.region, time.Now()); err != nil {
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
