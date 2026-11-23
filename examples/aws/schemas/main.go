// Command schemas walks an EventBridge Schema Registry lifecycle against
// Stackyard: create a registry and a schema, add a version, export it, infer a
// schema from sample events, generate a Python code binding, run a discoverer
// on the default event bus, then tear everything down.
package main

import (
	"archive/zip"
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

const content = `{"openapi":"3.0.0","info":{"version":"1.0.0","title":"OrderPlaced"},"paths":{},` +
	`"components":{"schemas":{"AWSEvent":{"type":"object","properties":{"source":{"type":"string"},"detail":{"$ref":"#/components/schemas/Order"}}},` +
	`"Order":{"type":"object","properties":{"orderId":{"type":"string"},"total":{"type":"number"}}}}}}`

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard EventBridge Schemas client using %s\n", c.endpoint)

	registry := fmt.Sprintf("shop-%d", time.Now().UnixNano())
	reg := "/v1/registries/name/" + registry
	fmt.Println("Registry:", c.call("CreateRegistry", http.MethodPost, reg, map[string]any{"Description": "Shop events"})["RegistryArn"])

	schema := reg + "/schemas/name/shop@OrderPlaced"
	created := c.call("CreateSchema", http.MethodPost, schema, map[string]any{"Type": "OpenApi3", "Content": content})
	fmt.Println("Schema:", created["SchemaArn"], "version", created["SchemaVersion"])
	v2 := strings.Replace(content, `"total":{"type":"number"}`, `"total":{"type":"number"},"currency":{"type":"string"}`, 1)
	fmt.Println("Updated to version", c.call("UpdateSchema", http.MethodPut, schema, map[string]any{"Content": v2})["SchemaVersion"])

	exported := c.call("ExportSchema", http.MethodGet, schema+"/export?type=JSONSchemaDraft4", nil)
	fmt.Println("Exported as", exported["Type"], "-", len(exported["Content"].(string)), "bytes")

	found := c.call("SearchSchemas", http.MethodGet, reg+"/schemas/search?keywords=currency", nil)["Schemas"].([]any)
	fmt.Println("Search for currency:", len(found), "schema")

	discovered := c.call("GetDiscoveredSchema", http.MethodPost, "/v1/discover", map[string]any{
		"Type": "JSONSchemaDraft4", "Events": []string{`{"source":"shop","detail":{"orderId":"o-1","total":9.5}}`},
	})["Content"].(string)
	fmt.Println("Discovered schema:", discovered)

	binding := schema + "/language/Python36"
	fmt.Println("Code binding:", c.call("PutCodeBinding", http.MethodPost, binding, nil)["Status"])
	c.call("DescribeCodeBinding", http.MethodGet, binding, nil)
	fmt.Println("Code binding:", c.call("DescribeCodeBinding", http.MethodGet, binding, nil)["Status"])
	archive := c.send("GetCodeBindingSource", "schemas", http.MethodGet, c.endpoint+binding+"/source", nil, "")
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil || len(zr.File) == 0 {
		exitf("GetCodeBindingSource: not a zip archive: %v", err)
	}
	fmt.Println("Generated file:", zr.File[0].Name)

	bus := "arn:aws:events:" + c.region + ":123456789012:event-bus/default"
	discoverer := c.call("CreateDiscoverer", http.MethodPost, "/v1/discoverers", map[string]any{"SourceArn": bus})
	id := discoverer["DiscovererId"].(string)
	fmt.Println("Discoverer:", id, discoverer["State"])
	fmt.Println("Discoverer:", c.call("StopDiscoverer", http.MethodPost, "/v1/discoverers/id/"+id+"/stop", nil)["State"])

	c.call("DeleteDiscoverer", http.MethodDelete, "/v1/discoverers/id/"+id, nil)
	c.call("DeleteSchema", http.MethodDelete, schema, nil)
	c.call("DeleteRegistry", http.MethodDelete, reg, nil)
	fmt.Println("Deleted registry", registry)
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, "schemas", method, c.endpoint+path, body, "application/json")
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, service, method, target string, body []byte, contentType string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", contentType)
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
