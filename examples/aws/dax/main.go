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

type cluster struct {
	ClusterName    string
	Status         string
	TotalNodes     int
	ActiveNodes    int
	Nodes          []struct{ NodeId, NodeStatus string }
	ParameterGroup struct{ ParameterGroupName, ParameterApplyStatus string }
}

// A DAX control-plane walkthrough: a subnet group and parameter group, a
// cluster that scales out and back in, a tuned TTL, then cleanup.
func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	name := getenv("STACKYARD_DAX_CLUSTER", "orders-cache")
	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	c := client{endpoint: endpoint, region: region, creds: creds}
	fmt.Printf("Stackyard DAX client using %s\n", endpoint)

	c.call(ctx, "CreateSubnetGroup", map[string]any{"SubnetGroupName": "cache-subnets", "SubnetIds": []string{"subnet-00000001"}}, nil)
	c.call(ctx, "CreateParameterGroup", map[string]any{"ParameterGroupName": "short-ttl", "Description": "one-minute query cache"}, nil)
	c.call(ctx, "UpdateParameterGroup", map[string]any{
		"ParameterGroupName":  "short-ttl",
		"ParameterNameValues": []map[string]string{{"ParameterName": "query-ttl-millis", "ParameterValue": "60000"}},
	}, nil)

	var created struct{ Cluster cluster }
	c.call(ctx, "CreateCluster", map[string]any{
		"ClusterName": name, "NodeType": "dax.t3.small", "ReplicationFactor": 2,
		"IamRoleArn":      "arn:aws:iam::123456789012:role/DAXServiceRole",
		"SubnetGroupName": "cache-subnets", "ParameterGroupName": "short-ttl",
		"SSESpecification": map[string]bool{"Enabled": true},
		"Tags":             []map[string]string{{"Key": "team", "Value": "orders"}},
	}, &created)
	logf("cluster %s is %s with %d nodes", created.Cluster.ClusterName, created.Cluster.Status, created.Cluster.TotalNodes)

	available := c.describe(ctx, name)
	if available.Status != "available" || available.ActiveNodes != 2 {
		exitf("expected an available 2-node cluster, got %+v", available)
	}

	c.call(ctx, "IncreaseReplicationFactor", map[string]any{"ClusterName": name, "NewReplicationFactor": 3}, nil)
	if got := c.describe(ctx, name); got.ActiveNodes != 3 {
		exitf("expected 3 nodes after scaling out, got %+v", got)
	}
	c.call(ctx, "DecreaseReplicationFactor", map[string]any{"ClusterName": name, "NewReplicationFactor": 2, "NodeIdsToRemove": []string{name + "-c"}}, nil)
	if got := c.describe(ctx, name); got.TotalNodes != 2 {
		exitf("expected 2 nodes after scaling in, got %+v", got)
	}

	var params struct {
		Parameters []struct{ ParameterName, ParameterValue string }
	}
	c.call(ctx, "DescribeParameters", map[string]any{"ParameterGroupName": "short-ttl", "Source": "user"}, &params)
	if len(params.Parameters) != 1 || params.Parameters[0].ParameterValue != "60000" {
		exitf("expected the tuned query TTL, got %+v", params.Parameters)
	}

	c.call(ctx, "DeleteCluster", map[string]any{"ClusterName": name}, nil)
	c.call(ctx, "DeleteParameterGroup", map[string]any{"ParameterGroupName": "short-ttl"}, nil)
	c.call(ctx, "DeleteSubnetGroup", map[string]any{"SubnetGroupName": "cache-subnets"}, nil)
	logf("DAX walkthrough complete")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

// describe reads one cluster; describing also completes pending changes.
func (c client) describe(ctx context.Context, name string) cluster {
	var out struct{ Clusters []cluster }
	c.call(ctx, "DescribeClusters", map[string]any{"ClusterNames": []string{name}}, &out)
	if len(out.Clusters) != 1 {
		exitf("DescribeClusters: expected one cluster, got %d", len(out.Clusters))
	}
	return out.Clusters[0]
}

// call sends one awsJson1_1 request signed for "dax", exits on a non-2xx
// reply, and decodes into out.
func (c client) call(ctx context.Context, action string, payload, out any) {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonDAXV3."+action)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "dax", c.region, time.Now()); err != nil {
		exitf("%s: %v", action, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		exitf("expected %s to return 2xx, got %d: %s", action, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	logf("%s returned %d", action, resp.StatusCode)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			exitf("%s: decode %s: %v", action, raw, err)
		}
	}
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
