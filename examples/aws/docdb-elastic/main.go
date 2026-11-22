// Command docdb-elastic walks an Amazon DocumentDB Elastic Clusters lifecycle
// against Stackyard: create a cluster, wait for it, scale it, snapshot and
// restore it, opt in to maintenance, tag it, then delete everything.
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
	fmt.Printf("Stackyard DocumentDB Elastic Clusters client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano())

	created := c.call("CreateCluster", http.MethodPost, "/cluster", map[string]any{
		"clusterName": "orders-" + suffix, "adminUserName": "admin", "adminUserPassword": "ExamplePassw0rd",
		"authType": "PLAIN_TEXT", "shardCapacity": 2, "shardCount": 2, "tags": map[string]string{"team": "orders"},
	})["cluster"].(map[string]any)
	arn := created["clusterArn"].(string)
	fmt.Printf("created cluster %s (%v)\n", created["clusterName"], created["status"])
	waitFor(c, "/cluster/"+url.PathEscape(arn), "cluster", "ACTIVE")

	c.call("UpdateCluster", http.MethodPut, "/cluster/"+url.PathEscape(arn), map[string]any{"shardCount": 3})
	scaled := waitFor(c, "/cluster/"+url.PathEscape(arn), "cluster", "ACTIVE")
	fmt.Printf("scaled to %v shards\n", scaled["shardCount"])

	snap := c.call("CreateClusterSnapshot", http.MethodPost, "/cluster-snapshot", map[string]any{
		"clusterArn": arn, "snapshotName": "orders-snap-" + suffix,
	})["snapshot"].(map[string]any)
	snapARN := snap["snapshotArn"].(string)
	waitFor(c, "/cluster-snapshot/"+url.PathEscape(snapARN), "snapshot", "ACTIVE")
	restored := c.call("RestoreClusterFromSnapshot", http.MethodPost, "/cluster-snapshot/"+url.PathEscape(snapARN)+"/restore", map[string]any{
		"clusterName": "orders-restored-" + suffix,
	})["cluster"].(map[string]any)
	restoredARN := restored["clusterArn"].(string)
	waitFor(c, "/cluster/"+url.PathEscape(restoredARN), "cluster", "ACTIVE")
	fmt.Printf("restored %s from %s\n", restored["clusterName"], snap["snapshotName"])

	pending := c.call("ListPendingMaintenanceActions", http.MethodGet, "/pending-actions", nil)
	fmt.Printf("%d cluster(s) have pending maintenance\n", len(pending["resourcePendingMaintenanceActions"].([]any)))
	c.call("ApplyPendingMaintenanceAction", http.MethodPost, "/pending-action", map[string]any{
		"resourceArn": arn, "applyAction": "SYSTEM_UPDATE", "optInType": "NEXT_MAINTENANCE",
	})

	c.call("TagResource", http.MethodPost, "/tags/"+url.PathEscape(arn), map[string]any{"tags": map[string]string{"env": "dev"}})
	tags := c.call("ListTagsForResource", http.MethodGet, "/tags/"+url.PathEscape(arn), nil)
	fmt.Printf("tags: %v\n", tags["tags"])

	for _, a := range []string{restoredARN, arn} {
		c.call("DeleteCluster", http.MethodDelete, "/cluster/"+url.PathEscape(a), nil)
	}
	c.call("DeleteClusterSnapshot", http.MethodDelete, "/cluster-snapshot/"+url.PathEscape(snapARN), nil)
	fmt.Println("DocumentDB Elastic Clusters example completed")
}

// waitFor reads a resource until it reaches status (reads settle transitions).
func waitFor(c client, path, key, status string) map[string]any {
	for i := 0; i < 5; i++ {
		got := c.call("Get", http.MethodGet, path, nil)[key].(map[string]any)
		if got["status"] == status {
			return got
		}
	}
	exitf("%s never reached %s", path, status)
	return nil
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, method, c.endpoint+path, body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a docdb-elastic request and exits unless it returns 2xx.
func (c client) send(op, method, target string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "docdb-elastic", c.region, time.Now()); err != nil {
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
