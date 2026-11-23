// Command waf-regional runs AWS WAF Classic Regional against Stackyard: build
// an IP set, a rule that matches it and a web ACL that blocks it, attach the
// web ACL to a load balancer, then take it all apart again. Every change uses a
// fresh change token from GetChangeToken.
//
// The SDK has no WAF Regional module in this repository's go.mod, so requests
// are signed by hand.
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

const alb = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard-web/50dc6c495c0c9188"

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
	fmt.Printf("Stackyard WAF Classic Regional client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)

	ipset := c.change("CreateIPSet", map[string]any{"Name": "blocked-" + suffix})["IPSet"].(map[string]any)["IPSetId"].(string)
	c.change("UpdateIPSet", map[string]any{"IPSetId": ipset, "Updates": []any{
		map[string]any{"Action": "INSERT", "IPSetDescriptor": map[string]any{"Type": "IPV4", "Value": "192.0.2.0/24"}},
	}})
	rule := c.change("CreateRule", map[string]any{"Name": "block-" + suffix, "MetricName": "block" + suffix})["Rule"].(map[string]any)["RuleId"].(string)
	c.change("UpdateRule", map[string]any{"RuleId": rule, "Updates": []any{
		map[string]any{"Action": "INSERT", "Predicate": map[string]any{"Negated": false, "Type": "IPMatch", "DataId": ipset}},
	}})
	acl := c.change("CreateWebACL", map[string]any{"Name": "web-" + suffix, "MetricName": "web" + suffix,
		"DefaultAction": map[string]any{"Type": "ALLOW"}})["WebACL"].(map[string]any)["WebACLId"].(string)
	activated := map[string]any{"Priority": 1, "RuleId": rule, "Action": map[string]any{"Type": "BLOCK"}}
	c.change("UpdateWebACL", map[string]any{"WebACLId": acl, "Updates": []any{map[string]any{"Action": "INSERT", "ActivatedRule": activated}}})
	fmt.Printf("web ACL %s blocks rule %s matching IP set %s\n", acl, rule, ipset)

	c.call("AssociateWebACL", map[string]any{"WebACLId": acl, "ResourceArn": alb})
	fmt.Printf("protected: %v\n", c.call("ListResourcesForWebACL", map[string]any{"WebACLId": acl})["ResourceArns"])

	// Tear down in dependency order: a referenced or non-empty entity can't be deleted.
	c.call("DisassociateWebACL", map[string]any{"ResourceArn": alb})
	c.change("UpdateWebACL", map[string]any{"WebACLId": acl, "Updates": []any{map[string]any{"Action": "DELETE", "ActivatedRule": activated}}})
	c.change("DeleteWebACL", map[string]any{"WebACLId": acl})
	c.change("UpdateRule", map[string]any{"RuleId": rule, "Updates": []any{
		map[string]any{"Action": "DELETE", "Predicate": map[string]any{"Negated": false, "Type": "IPMatch", "DataId": ipset}},
	}})
	c.change("DeleteRule", map[string]any{"RuleId": rule})
	c.change("UpdateIPSet", map[string]any{"IPSetId": ipset, "Updates": []any{
		map[string]any{"Action": "DELETE", "IPSetDescriptor": map[string]any{"Type": "IPV4", "Value": "192.0.2.0/24"}},
	}})
	token := c.change("DeleteIPSet", map[string]any{"IPSetId": ipset})["ChangeToken"].(string)
	fmt.Printf("cleaned up; last change token is %v\n", c.call("GetChangeTokenStatus", map[string]any{"ChangeToken": token})["ChangeTokenStatus"])
}

// change runs a mutating call with a fresh change token.
func (c client) change(op string, payload map[string]any) map[string]any {
	payload["ChangeToken"] = c.call("GetChangeToken", map[string]any{})["ChangeToken"]
	return c.call(op, payload)
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSWAF_Regional_20161128." + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a waf-regional request and exits unless it returns 2xx.
func (c client) send(op string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
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
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, "waf-regional", c.region, time.Now()); err != nil {
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
