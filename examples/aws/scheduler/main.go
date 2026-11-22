// Command scheduler walks an EventBridge Scheduler lifecycle against
// Stackyard: create a schedule group and an SQS-targeted schedule, fire it
// through Stackyard's on-demand endpoint, read the delivered message, then
// tear everything down.
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
	account := getenv("AWS_ACCOUNT_ID", "123456789012")
	suffix := fmt.Sprint(time.Now().UnixNano())
	queue, group, name := "scheduler-demo-"+suffix, "demo-"+suffix, "hourly-"+suffix
	fmt.Printf("Stackyard EventBridge Scheduler client using %s\n", c.endpoint)

	c.sqs(url.Values{"Action": {"CreateQueue"}, "QueueName": {queue}})
	c.json("CreateScheduleGroup", http.MethodPost, "/schedule-groups/"+group, map[string]any{
		"Tags": []map[string]string{{"Key": "team", "Value": "demo"}},
	})
	created := c.json("CreateSchedule", http.MethodPost, "/schedules/"+name, map[string]any{
		"GroupName":                  group,
		"ScheduleExpression":         "cron(0 9 ? * MON-FRI *)",
		"ScheduleExpressionTimezone": "America/New_York",
		"FlexibleTimeWindow":         map[string]any{"Mode": "FLEXIBLE", "MaximumWindowInMinutes": 15},
		"Target": map[string]any{
			"Arn":         fmt.Sprintf("arn:aws:sqs:%s:%s:%s", c.region, account, queue),
			"RoleArn":     fmt.Sprintf("arn:aws:iam::%s:role/scheduler-demo", account),
			"Input":       `{"report":"daily"}`,
			"RetryPolicy": map[string]any{"MaximumRetryAttempts": 3},
		},
	})
	fmt.Printf("created %v\n", created["ScheduleArn"])

	got := c.json("GetSchedule", http.MethodGet, "/schedules/"+name+"?groupName="+group, nil)
	fmt.Printf("schedule is %v, runs %v in %v\n", got["State"], got["ScheduleExpression"], got["ScheduleExpressionTimezone"])

	fired := c.json("Fire", http.MethodPost, "/_stackyard/scheduler/schedules/"+group+"/"+name+"/fire", nil)
	fmt.Printf("fired schedule, SQS message %v\n", fired["MessageId"])
	received := c.sqs(url.Values{"Action": {"ReceiveMessage"}, "QueueUrl": {c.endpoint + "/" + account + "/" + queue}})
	if !strings.Contains(received, "daily") {
		exitf("ReceiveMessage: the scheduled input was not delivered: %s", received)
	}
	fmt.Println("queue received the scheduled input")

	listed := c.json("ListSchedules", http.MethodGet, "/schedules?ScheduleGroup="+group, nil)
	fmt.Printf("group %s has %d schedule(s)\n", group, len(listed["Schedules"].([]any)))
	c.json("DeleteSchedule", http.MethodDelete, "/schedules/"+name+"?groupName="+group, nil)
	c.json("DeleteScheduleGroup", http.MethodDelete, "/schedule-groups/"+group, nil)
	c.sqs(url.Values{"Action": {"DeleteQueue"}, "QueueUrl": {c.endpoint + "/" + account + "/" + queue}})
	fmt.Println("EventBridge Scheduler example completed")
}

// json sends a signed Scheduler REST call and exits unless it returns 2xx.
func (c client) json(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, "scheduler", method, path, "application/json", body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// sqs sends a signed SQS Query call and returns the XML reply.
func (c client) sqs(form url.Values) string {
	return string(c.send(form.Get("Action"), "sqs", http.MethodPost, "/", "application/x-www-form-urlencoded", []byte(form.Encode())))
}

func (c client) send(op, service, method, path, contentType string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", contentType)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
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
