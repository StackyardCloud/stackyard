// Command codestar-notifications creates a CodeStar Notifications rule on
// Stackyard's seeded CodeBuild project with an SNS target, then walks the
// rule through subscribe, list, tag and delete.
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

var (
	endpoint = getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region   = getenv("AWS_REGION", "us-east-1")
	account  = getenv("AWS_ACCOUNT_ID", "123456789012")
	creds    = credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
)

func main() {
	fmt.Printf("Stackyard CodeStar Notifications client using %s\n", endpoint)
	topic := createTopic("codestar-notifications-example")
	project := fmt.Sprintf("arn:aws:codebuild:%s:%s:project/stackyard-codebuild-project", region, account)

	created := call("/createNotificationRule", map[string]any{
		"Name":               fmt.Sprintf("build-failures-%d", time.Now().UnixNano()),
		"EventTypeIds":       []string{"codebuild-project-build-state-failed"},
		"Resource":           project,
		"Targets":            []map[string]string{{"TargetType": "SNS", "TargetAddress": topic}},
		"DetailType":         "FULL",
		"ClientRequestToken": "codestar-notifications-example",
		"Tags":               map[string]string{"example": "codestar-notifications"},
	})
	arn, _ := created["Arn"].(string)
	logf("created rule %s", arn)

	call("/subscribe", map[string]any{"Arn": arn, "Target": map[string]string{"TargetType": "AWSChatbotSlack", "TargetAddress": "arn:aws:chatbot::" + account + ":chat-configuration/slack-channel/builds"}})
	rule := call("/describeNotificationRule", map[string]any{"Arn": arn})
	if targets, _ := rule["Targets"].([]any); len(targets) != 2 {
		exitf("expected 2 targets, got %v", rule["Targets"])
	}
	logf("rule %v has %d targets and status %v", rule["Name"], len(rule["Targets"].([]any)), rule["Status"])

	events := call("/listEventTypes", map[string]any{"Filters": []map[string]string{{"Name": "SERVICE_NAME", "Value": "CodeBuild"}}})
	logf("CodeBuild offers %d event types", len(events["EventTypes"].([]any)))

	call("/updateNotificationRule", map[string]any{"Arn": arn, "Status": "DISABLED"})
	call("/tagResource", map[string]any{"Arn": arn, "Tags": map[string]string{"team": "ci"}})
	tags := call("/listTagsForResource", map[string]any{"Arn": arn})
	logf("tags: %v", tags["Tags"])

	call("/deleteNotificationRule", map[string]any{"Arn": arn})
	call("/deleteTarget", map[string]any{"TargetAddress": topic, "ForceUnsubscribeAll": true})
	logf("deleted rule and target")
}

func createTopic(name string) string {
	form := url.Values{"Action": {"CreateTopic"}, "Version": {"2010-03-31"}, "Name": {name}}
	status, body := send(http.MethodPost, "/", "sns", "application/x-www-form-urlencoded", []byte(form.Encode()))
	expect2xx("sns:CreateTopic", status, body)
	start := bytes.Index(body, []byte("<TopicArn>"))
	end := bytes.Index(body, []byte("</TopicArn>"))
	if start < 0 || end < start {
		exitf("sns:CreateTopic: no TopicArn in %s", body)
	}
	return string(body[start+len("<TopicArn>") : end])
}

func call(path string, payload map[string]any) map[string]any {
	raw, _ := json.Marshal(payload)
	status, body := send(http.MethodPost, path, "codestar-notifications", "application/json", raw)
	expect2xx(strings.TrimPrefix(path, "/"), status, body)
	out := map[string]any{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			exitf("%s: decode %s: %v", path, body, err)
		}
	}
	return out
}

func send(method, path, service, contentType string, body []byte) (int, []byte) {
	req, err := http.NewRequest(method, strings.TrimRight(endpoint, "/")+path, bytes.NewReader(body))
	if err != nil {
		exitf("build request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	ctx := context.Background()
	c, err := creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	if err := v4.NewSigner().SignHTTP(ctx, aws.Credentials(c), req, hash, service, region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func expect2xx(op string, status int, body []byte) {
	if status < 200 || status > 299 {
		exitf("%s: HTTP %d: %s", op, status, body)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func logf(format string, args ...any) { fmt.Printf(format+"\n", args...) }

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
