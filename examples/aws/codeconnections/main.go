// Command codeconnections walks an AWS CodeConnections lifecycle against
// Stackyard: create a GitHub connection, complete its handshake through the
// Stackyard activate endpoint, link a repository, add a CloudFormation Git
// sync configuration and read its status, read the connection through the
// CodeStar Connections name too, then tear it all down.
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

const (
	codeConnectionsTarget = "com.amazonaws.codeconnections.CodeConnections_20231201."
	codeStarTarget        = "com.amazonaws.codestar.connections.CodeStar_connections_20191201."
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
	fmt.Printf("Stackyard CodeConnections client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)

	conn := c.call("codeconnections", codeConnectionsTarget+"CreateConnection", map[string]any{"ProviderType": "GitHub", "ConnectionName": "example-" + suffix})
	arn, _ := conn["ConnectionArn"].(string)
	fmt.Printf("created connection %s (PENDING until the GitHub handshake)\n", arn)
	c.post("codeconnections", "/_stackyard/codeconnections/connections/"+arn[strings.LastIndex(arn, "/")+1:]+"/activate", nil)

	got := c.call("codestar-connections", codeStarTarget+"GetConnection", map[string]any{"ConnectionArn": arn})
	info, _ := got["Connection"].(map[string]any)
	fmt.Printf("CodeStar Connections sees %v as %v\n", info["ConnectionArn"], info["ConnectionStatus"])

	link := c.call("codeconnections", codeConnectionsTarget+"CreateRepositoryLink", map[string]any{"ConnectionArn": arn, "OwnerId": "acme", "RepositoryName": "infra-" + suffix})
	linkInfo, _ := link["RepositoryLinkInfo"].(map[string]any)
	linkID, _ := linkInfo["RepositoryLinkId"].(string)
	fmt.Printf("linked repository acme/infra-%s as %s\n", suffix, linkID)

	resource := "examplestack" + suffix
	c.call("codeconnections", codeConnectionsTarget+"CreateSyncConfiguration", map[string]any{
		"Branch": "main", "ConfigFile": "deploy/stack.yaml", "RepositoryLinkId": linkID, "ResourceName": resource,
		"RoleArn": "arn:aws:iam::123456789012:role/git-sync", "SyncType": "CFN_STACK_SYNC",
	})
	status := c.call("codeconnections", codeConnectionsTarget+"GetResourceSyncStatus", map[string]any{"ResourceName": resource, "SyncType": "CFN_STACK_SYNC"})
	latest, _ := status["LatestSync"].(map[string]any)
	fmt.Printf("stack %s last sync: %v\n", resource, latest["Status"])

	c.call("codeconnections", codeConnectionsTarget+"DeleteSyncConfiguration", map[string]any{"ResourceName": resource, "SyncType": "CFN_STACK_SYNC"})
	c.call("codeconnections", codeConnectionsTarget+"DeleteRepositoryLink", map[string]any{"RepositoryLinkId": linkID})
	c.call("codeconnections", codeConnectionsTarget+"DeleteConnection", map[string]any{"ConnectionArn": arn})
	fmt.Println("CodeConnections example completed")
}

// call sends a signed awsJson1_0 request and exits unless it returns 2xx.
func (c client) call(service, target string, payload any) map[string]any {
	return c.do(service, "/", payload, map[string]string{"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": target})
}

// post calls a Stackyard-only endpoint, signed like the API.
func (c client) post(service, path string, payload any) map[string]any {
	return c.do(service, path, payload, map[string]string{"Content-Type": "application/json"})
}

func (c client) do(service, path string, payload any, headers map[string]string) map[string]any {
	body := []byte("{}")
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", path, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", path, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
		exitf("%s: %v", path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s %s: HTTP %d: %s", path, headers["X-Amz-Target"], resp.StatusCode, raw)
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", path, err)
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
