// Command bcm-dashboards runs AWS Billing and Cost Management Dashboards
// against Stackyard: create a dashboard with a Cost Explorer-style widget,
// schedule a weekly report of it, run the report, and show how the report
// turns unhealthy once its dashboard is deleted.
//
// Stackyard checks and stores widget queries; it does not compute costs. The
// SDK has no Dashboards module in this repository's go.mod, so requests are
// signed by hand.
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
	signingName  = "bcm-dashboards"
	targetPrefix = "AWSBCMDashboardsService."
	contentType  = "application/x-amz-json-1.0"
	widgetID     = "5f1c0e2a9b7d4c3e8a6f1b2d3c4e5f60"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard BCM Dashboards client using %s\n", c.endpoint)

	relative := func(v string) map[string]any { return map[string]any{"type": "RELATIVE", "value": v} }
	dashboard := c.call("CreateDashboard", map[string]any{
		"name": fmt.Sprintf("Monthly spend %d", time.Now().Unix()), "description": "Cost by service",
		"widgets": []any{map[string]any{
			"id": widgetID, "title": "Cost by service", "width": 6, "height": 5,
			"configs": []any{map[string]any{
				"queryParameters": map[string]any{"costAndUsage": map[string]any{
					"metrics": []string{"UnblendedCost"}, "granularity": "MONTHLY",
					"timeRange": map[string]any{"startTime": relative("-P6M"), "endTime": relative("P0D")},
					"groupBy":   []any{map[string]any{"key": "SERVICE", "type": "DIMENSION"}},
				}},
				"displayConfig": map[string]any{"graph": map[string]any{"cost": map[string]any{"visualType": "STACK"}}},
			}},
		}},
		"resourceTags": []any{map[string]any{"key": "team", "value": "finance"}},
	})["arn"].(string)
	fmt.Println("dashboard:", dashboard)
	got := c.call("GetDashboard", map[string]any{"arn": dashboard})
	fmt.Printf("widgets: %d\n", len(got["widgets"].([]any)))

	report := c.call("CreateScheduledReport", map[string]any{"scheduledReport": map[string]any{
		"name": fmt.Sprintf("Weekly spend %d", time.Now().Unix()), "dashboardArn": dashboard,
		"scheduledReportExecutionRoleArn": "arn:aws:iam::123456789012:role/dashboard-reports",
		"scheduleConfig":                  map[string]any{"scheduleExpression": "cron(0 8 ? * MON *)", "scheduleExpressionTimeZone": "UTC"},
		"widgetIds":                       []string{widgetID},
	}})["arn"].(string)
	fmt.Println("scheduled report:", report)

	run := c.call("ExecuteScheduledReport", map[string]any{"arn": report})
	fmt.Printf("run: triggered=%v health=%v\n", run["executionTriggered"], run["healthStatus"].(map[string]any)["statusCode"])

	c.call("DeleteDashboard", map[string]any{"arn": dashboard})
	run = c.call("ExecuteScheduledReport", map[string]any{"arn": report, "dryRun": true})
	health := run["healthStatus"].(map[string]any)
	fmt.Printf("after deleting the dashboard: %v %v\n", health["statusCode"], health["statusReasons"])

	c.call("DeleteScheduledReport", map[string]any{"arn": report})
	fmt.Println("deleted scheduled report")
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

// call sends one awsJson operation and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, http.MethodPost, signingName, c.endpoint+"/", body, map[string]string{
		"Content-Type": contentType, "X-Amz-Target": targetPrefix + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
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
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
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
