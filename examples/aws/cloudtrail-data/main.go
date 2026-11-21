package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// Creates a CloudTrail Lake event data store and channel, ingests an external audit
// event with CloudTrail Data PutAuditEvents, and reads it back with a Lake query.
func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	accountID := getenv("AWS_ACCOUNT_ID", "123456789012")
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	c := client{ctx: context.Background(), endpoint: endpoint, region: region, creds: creds}
	fmt.Printf("Stackyard CloudTrail Data client using %s\n", endpoint)

	edsName := "cloudtrail-data-example-eds"
	c.cloudTrail("CreateEventDataStore", map[string]any{"Name": edsName})
	channel := c.cloudTrail("CreateChannel", map[string]any{
		"Name":   "cloudtrail-data-example-channel",
		"Source": "Custom",
		"Destinations": []map[string]string{{
			"Type":     "EventDataStore",
			"Location": fmt.Sprintf("arn:aws:cloudtrail:%s:%s:eventdatastore/%s", region, accountID, edsName),
		}},
	})
	channelArn, _ := channel["ChannelArn"].(string)
	fmt.Printf("channel %s\n", channelArn)

	eventData, _ := json.Marshal(map[string]any{
		"version":            "1.0",
		"userIdentity":       map[string]string{"type": "CustomUser", "principalId": "example-user"},
		"eventSource":        "app.example.com",
		"eventName":          "ExampleLogin",
		"eventTime":          time.Now().UTC().Format(time.RFC3339),
		"UID":                "example-uid-1",
		"recipientAccountId": accountID,
	})
	sum := sha256.Sum256(eventData)
	put := c.call("cloudtrail-data", "/PutAuditEvents?channelArn="+url.QueryEscape(channelArn), nil, map[string]any{
		"auditEvents": []map[string]string{{
			"id":                "example-event-1",
			"eventData":         string(eventData),
			"eventDataChecksum": base64.StdEncoding.EncodeToString(sum[:]),
		}},
	})
	if ok, _ := put["successful"].([]any); len(ok) != 1 {
		exitf("PutAuditEvents: expected one successful event, got %v", put)
	}
	fmt.Printf("PutAuditEvents: %v\n", put["successful"])

	started := c.cloudTrail("StartQuery", map[string]any{
		"QueryStatement": "SELECT eventID, eventData.eventName FROM " + edsName + " WHERE eventData.eventName = 'ExampleLogin'",
	})
	results := c.cloudTrail("GetQueryResults", map[string]any{"QueryId": started["QueryId"]})
	if rows, _ := results["QueryResultRows"].([]any); len(rows) != 1 {
		exitf("GetQueryResults: expected one row, got %v", results)
	}
	fmt.Printf("query rows: %v\n", results["QueryResultRows"])
}

type client struct {
	ctx      context.Context
	endpoint string
	region   string
	creds    aws.CredentialsProvider
}

func (c client) cloudTrail(action string, payload map[string]any) map[string]any {
	return c.call("cloudtrail", "/", map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "com.amazonaws.cloudtrail.v20131101.CloudTrail_20131101." + action,
	}, payload)
}

func (c client) call(service, path string, headers map[string]string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: marshal: %v", path, err)
	}
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(c.ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	hash := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(c.ctx, creds, req, hex.EncodeToString(hash[:]), service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", service, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s %s: status %d: %s", service, path, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s %s: decode: %v", service, path, err)
	}
	return out
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
