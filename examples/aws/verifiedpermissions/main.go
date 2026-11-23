// Command verifiedpermissions walks an Amazon Verified Permissions setup
// against Stackyard: a policy store with a Cedar schema, a static permit and a
// forbid, then IsAuthorized decisions that show group membership, an action
// group from the schema, and forbid overriding permit. Everything is deleted
// at the end.
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

const schema = `{"Photo": {
  "entityTypes": {"User": {"memberOfTypes": ["Group"]}, "Group": {}, "Image": {}},
  "actions": {
    "read": {},
    "view": {"memberOf": [{"id": "read"}], "appliesTo": {"principalTypes": ["User"], "resourceTypes": ["Image"]}}
  }
}}`

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Verified Permissions client using %s\n", c.endpoint)

	store := c.call("CreatePolicyStore", map[string]any{"validationSettings": map[string]any{"mode": "STRICT"}})["policyStoreId"].(string)
	fmt.Println("Policy store:", store)
	c.call("PutSchema", map[string]any{"policyStoreId": store, "definition": map[string]any{"cedarJson": schema}})
	static := func(stmt string) string {
		return c.call("CreatePolicy", map[string]any{"policyStoreId": store,
			"definition": map[string]any{"static": map[string]any{"statement": stmt}}})["policyId"].(string)
	}
	permit := static(`permit(principal in Photo::Group::"family", action in Photo::Action::"read", resource);`)
	forbid := static(`forbid(principal, action, resource == Photo::Image::"private");`)
	fmt.Println("Policies:", permit, forbid)

	alice := map[string]any{"identifier": map[string]any{"entityType": "Photo::User", "entityId": "alice"},
		"parents": []any{map[string]any{"entityType": "Photo::Group", "entityId": "family"}}}
	decide := func(user, image string) map[string]any {
		return c.call("IsAuthorized", map[string]any{
			"policyStoreId": store,
			"principal":     map[string]any{"entityType": "Photo::User", "entityId": user},
			"action":        map[string]any{"actionType": "Photo::Action", "actionId": "view"},
			"resource":      map[string]any{"entityType": "Photo::Image", "entityId": image},
			"entities":      map[string]any{"entityList": []any{alice}},
		})
	}
	for _, tc := range []struct{ user, image, want string }{
		{"alice", "beach", "ALLOW"},  // family member, view is in the read group
		{"alice", "private", "DENY"}, // the forbid overrides the permit
		{"bob", "beach", "DENY"},     // no matching permit: default deny
	} {
		res := decide(tc.user, tc.image)
		fmt.Printf("%s viewing %s: %s %v\n", tc.user, tc.image, res["decision"], res["determiningPolicies"])
		if res["decision"] != tc.want {
			exitf("%s viewing %s: got %v, want %s", tc.user, tc.image, res["decision"], tc.want)
		}
	}

	c.call("DeletePolicy", map[string]any{"policyStoreId": store, "policyId": forbid})
	c.call("DeletePolicy", map[string]any{"policyStoreId": store, "policyId": permit})
	c.call("DeletePolicyStore", map[string]any{"policyStoreId": store})
	fmt.Println("Deleted policy store")
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
	req.Header.Set("X-Amz-Target", "VerifiedPermissions."+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "verifiedpermissions", c.region, time.Now()); err != nil {
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
