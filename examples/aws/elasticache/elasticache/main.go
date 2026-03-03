package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type queryCall struct {
	Name   string
	Params url.Values
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard ElastiCache advanced client using %s\n", endpoint)

	resourceArn := "arn:aws:elasticache:us-east-1:123456789012:cluster:stackyard-cache"
	calls := []queryCall{
		{Name: "DescribeCacheClusters", Params: query("DescribeCacheClusters", map[string]string{"MaxRecords": "20"})},
		{Name: "CreateCacheCluster", Params: query("CreateCacheCluster", map[string]string{"CacheClusterId": "stackyard-cache", "CacheNodeType": "cache.t4g.micro", "Engine": "redis", "NumCacheNodes": "1"})},
		{Name: "DescribeReplicationGroups", Params: query("DescribeReplicationGroups", map[string]string{"MaxRecords": "20"})},
		{Name: "CreateReplicationGroup", Params: query("CreateReplicationGroup", map[string]string{"ReplicationGroupId": "stackyard-rg", "ReplicationGroupDescription": "stackyard replication group", "CacheNodeType": "cache.t4g.micro", "Engine": "redis", "NumNodeGroups": "1", "ReplicasPerNodeGroup": "1"})},
		{Name: "ListTagsForResource", Params: query("ListTagsForResource", map[string]string{"ResourceName": resourceArn})},
		{Name: "AddTagsToResource", Params: query("AddTagsToResource", map[string]string{"ResourceName": resourceArn, "Tags.member.1.Key": "env", "Tags.member.1.Value": "dev", "Tags.member.2.Key": "owner", "Tags.member.2.Value": "stackyard"})},
		{Name: "RemoveTagsFromResource", Params: query("RemoveTagsFromResource", map[string]string{"ResourceName": resourceArn, "TagKeys.member.1": "owner"})},
		{Name: "StartMigration", Params: query("StartMigration", map[string]string{"ReplicationGroupId": "stackyard-rg", "CustomerNodeEndpointList.member.1.Address": "10.0.0.10", "CustomerNodeEndpointList.member.1.Port": "6379"})},
		{Name: "TestMigration", Params: query("TestMigration", map[string]string{"ReplicationGroupId": "stackyard-rg", "CustomerNodeEndpointList.member.1.Address": "10.0.0.10", "CustomerNodeEndpointList.member.1.Port": "6379"})},
	}

	for _, call := range calls {
		status, body, err := elastiCacheQueryRequest(ctx, endpoint, region, creds, call.Params)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", call.Name, status)
	}

	fmt.Println("Done.")
}

func query(action string, extras map[string]string) url.Values {
	v := url.Values{}
	v.Set("Action", action)
	v.Set("Version", "2015-02-02")
	for key, value := range extras {
		v.Set(key, value)
	}
	return v
}

func elastiCacheQueryRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	params url.Values,
) (int, []byte, error) {
	body := []byte(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "elasticache", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
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
