// Command repostspace runs AWS re:Post Private against Stackyard: create a
// space, wait for it to provision, register an administrator, grant expert
// roles, create a channel with a moderator, invite users, tag the space, and
// delete it (waiting until it is gone).
//
// The SDK has no re:Post Private module in this repository's go.mod, so
// requests are signed by hand.
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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const signingName = "repostspace"

func main() {
	c := newClient()
	fmt.Printf("Stackyard re:Post Private client using %s\n", c.endpoint)
	sub := fmt.Sprintf("eng-%d", time.Now().UnixNano())

	space := c.rest(http.MethodPost, "/spaces", map[string]any{
		"name": "Engineering", "subdomain": sub, "tier": "BASIC", "description": "Internal engineering Q&A",
		"tags": map[string]string{"team": "platform"},
	})["spaceId"].(string)
	status := c.waitSpace(space, "CREATING")
	fmt.Printf("space %s is %s\n", space, status)

	c.rest(http.MethodPost, "/spaces/"+space+"/admins/sso-admin-1", nil)
	added := c.rest(http.MethodPost, "/spaces/"+space+"/roles", map[string]any{"accessorIds": []string{"sso-user-1", "sso-user-2"}, "role": "EXPERT"})
	fmt.Printf("experts added: %v\n", added["addedAccessorIds"])
	c.rest(http.MethodPut, "/spaces/"+space, map[string]any{"tier": "STANDARD"})
	got := c.rest(http.MethodGet, "/spaces/"+space, nil)
	fmt.Printf("space %v: %v, %v users, tier %v, %v\n", got["name"], got["configurationStatus"], got["userCount"], got["tier"], got["vanityDomain"])

	channel := c.rest(http.MethodPost, "/spaces/"+space+"/channels", map[string]any{"channelName": "kubernetes", "channelDescription": "Cluster questions"})["channelId"].(string)
	c.rest(http.MethodPost, "/spaces/"+space+"/channels/"+channel+"/roles", map[string]any{"accessorIds": []string{"sso-user-1"}, "channelRole": "MODERATOR"})
	for i := 0; i < 20; i++ {
		ch := c.rest(http.MethodGet, "/spaces/"+space+"/channels/"+channel, nil)
		if ch["channelStatus"] != "CREATING" {
			fmt.Printf("channel %v is %v with roles %v\n", ch["channelName"], ch["channelStatus"], ch["channelRoles"])
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	c.rest(http.MethodPost, "/spaces/"+space+"/invite", map[string]any{"accessorIds": []string{"sso-user-3"}, "title": "Join us", "body": "Ask anything about the platform."})

	arn := "/tags/" + url.PathEscape(got["arn"].(string))
	c.rest(http.MethodPost, arn, map[string]any{"tags": map[string]string{"env": "dev"}})
	fmt.Printf("tags: %v\n", c.rest(http.MethodGet, arn, nil)["tags"])

	c.rest(http.MethodDelete, "/spaces/"+space, nil)
	fmt.Printf("space deletion: %s\n", c.waitSpace(space, "DELETING"))
}

// waitSpace polls GetSpace while the space is in state; a missing space
// reports DELETED.
func (c client) waitSpace(space, state string) string {
	for i := 0; i < 40; i++ {
		status, out := c.send(http.MethodGet, "/spaces/"+space, nil)
		if status == http.StatusNotFound {
			return "DELETED"
		}
		if out["status"] != state {
			return fmt.Sprint(out["status"])
		}
		time.Sleep(500 * time.Millisecond)
	}
	exitf("space %s stayed %s", space, state)
	return ""
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

// rest sends one request and exits unless it returns 2xx.
func (c client) rest(method, path string, payload any) map[string]any {
	status, out := c.send(method, path, payload)
	if status < 200 || status > 299 {
		exitf("%s %s: HTTP %d: %v", method, path, status, out)
	}
	return out
}

func (c client) send(method, path string, payload any) (int, map[string]any) {
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
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hash, signingName, c.region, time.Now()); err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s %s: decode: %v", method, path, err)
		}
	}
	return resp.StatusCode, out
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
