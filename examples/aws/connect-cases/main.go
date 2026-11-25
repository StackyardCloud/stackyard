// Command connect-cases runs Amazon Connect Cases against Stackyard: create a
// domain with a custom field, a layout and a template, open two cases, comment
// on one, search them, close one and merge the other into it.
//
// Requests are signed by hand (service cases, REST JSON).
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
	fmt.Printf("Stackyard Connect Cases client using %s\n", c.endpoint)
	call := func(method, path string, body any) map[string]any { return c.call("cases", method, path, body, "") }

	d := call(http.MethodPost, "/domains", map[string]any{"name": fmt.Sprintf("support-%d", time.Now().UnixNano())})["domainId"].(string)
	base := "/domains/" + d
	prio := call(http.MethodPost, base+"/fields", map[string]any{"name": "priority", "type": "Number"})["fieldId"].(string)
	layout := call(http.MethodPost, base+"/layouts", map[string]any{"name": "main", "content": map[string]any{"basic": map[string]any{
		"topPanel": map[string]any{"sections": []any{map[string]any{"fieldGroup": map[string]any{"fields": []any{
			map[string]any{"id": "title"}, map[string]any{"id": prio}}}}}}}}})["layoutId"].(string)
	tmpl := call(http.MethodPost, base+"/templates", map[string]any{"name": "default",
		"layoutConfiguration": map[string]any{"defaultLayout": layout}, "requiredFields": []any{map[string]any{"fieldId": "title"}}})["templateId"].(string)
	fmt.Println("domain:", d)

	newCase := func(title string, priority float64) string {
		return call(http.MethodPost, base+"/cases", map[string]any{"templateId": tmpl, "fields": []any{
			map[string]any{"id": "title", "value": map[string]any{"stringValue": title}},
			map[string]any{"id": prio, "value": map[string]any{"doubleValue": priority}},
		}})["caseId"].(string)
	}
	printer, dup := newCase("Printer on floor 3 is jammed", 2), newCase("Floor 3 printer jam (duplicate)", 1)
	call(http.MethodPost, base+"/cases/"+printer+"/related-items/", map[string]any{"type": "Comment",
		"content": map[string]any{"comment": map[string]any{"body": "Technician dispatched", "contentType": "Text/Plain"}}})

	found := call(http.MethodPost, base+"/cases-search", map[string]any{"searchTerm": "printer", "fields": []any{map[string]any{"id": "title"}},
		"sorts": []any{map[string]any{"fieldId": prio, "sortOrder": "Desc"}}})
	fmt.Println("search hits:", found["totalCount"])
	for _, item := range found["cases"].([]any) {
		fmt.Println(" -", item.(map[string]any)["fields"])
	}

	call(http.MethodPost, base+"/cases/"+printer+"/merge", map[string]any{"secondaryCaseIds": []any{dup}})
	fmt.Println("merge:", call(http.MethodPost, base+"/cases/"+dup+"/merge-status", nil)["status"])
	call(http.MethodPut, base+"/cases/"+printer, map[string]any{"fields": []any{
		map[string]any{"id": "status", "value": map[string]any{"stringValue": "closed"}}}})
	got := call(http.MethodPost, base+"/cases/"+printer, map[string]any{"fields": []any{
		map[string]any{"id": "reference_number"}, map[string]any{"id": "status"}}})
	fmt.Println("case:", got["fields"])
	for _, ev := range call(http.MethodPost, base+"/cases/"+printer+"/audit-history", nil)["auditEvents"].([]any) {
		fmt.Println("audit:", ev.(map[string]any)["type"])
	}
	call(http.MethodDelete, base, nil)
	fmt.Println("deleted domain", d)
}

// call sends a JSON request signed for service (or unsigned with a bearer
// token) and exits unless it returns 2xx.
func (c client) call(service, method, path string, payload any, bearer string) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s %s: %v", method, path, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("X-Amz-Bearer", bearer)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s %s: decode: %v", method, path, err)
		}
	}
	return out
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
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
