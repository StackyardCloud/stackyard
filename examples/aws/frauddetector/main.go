// Command frauddetector runs Amazon Fraud Detector against Stackyard: define
// variables, an entity type, labels and outcomes, an event type, a detector
// with a rule, activate a version, then score two payments.
//
// The SDK has no Fraud Detector module in this repository's go.mod, so
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
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region, service, target string
	creds                             aws.CredentialsProvider
}

func newClient(service, target string) client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		service:  service,
		target:   target,
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

func main() {
	c := newClient("frauddetector", "AWSHawksNestServiceFacade.")
	fmt.Printf("Stackyard Fraud Detector client using %s\n", c.endpoint)
	s := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)
	amount, event, det := "amount_"+s, "payment_"+s, "payments_"+s
	c.call("CreateVariable", map[string]any{"name": amount, "dataType": "FLOAT", "dataSource": "EVENT", "defaultValue": "0.0"})
	c.call("PutEntityType", map[string]any{"name": "customer_" + s})
	c.call("PutLabel", map[string]any{"name": "fraud_" + s})
	c.call("PutOutcome", map[string]any{"name": "review_" + s})
	c.call("PutOutcome", map[string]any{"name": "approve_" + s})
	c.call("PutEventType", map[string]any{"name": event, "eventVariables": []any{amount}, "labels": []any{"fraud_" + s}, "entityTypes": []any{"customer_" + s}})
	c.call("PutDetector", map[string]any{"detectorId": det, "eventTypeName": event})
	c.call("CreateRule", map[string]any{"ruleId": "large_" + s, "detectorId": det, "expression": "$" + amount + " > 1000",
		"language": "DETECTORPL", "outcomes": []any{"review_" + s}})
	c.call("CreateRule", map[string]any{"ruleId": "default_" + s, "detectorId": det, "expression": "$" + amount + " >= 0",
		"language": "DETECTORPL", "outcomes": []any{"approve_" + s}})
	ver := c.call("CreateDetectorVersion", map[string]any{"detectorId": det, "rules": []any{
		map[string]any{"detectorId": det, "ruleId": "large_" + s, "ruleVersion": "1"},
		map[string]any{"detectorId": det, "ruleId": "default_" + s, "ruleVersion": "1"},
	}})["detectorVersionId"].(string)
	c.call("UpdateDetectorVersionStatus", map[string]any{"detectorId": det, "detectorVersionId": ver, "status": "ACTIVE"})
	fmt.Printf("detector %s version %s is ACTIVE\n", det, ver)

	for i, value := range []string{"25.00", "4999.99"} {
		out := c.call("GetEventPrediction", map[string]any{"detectorId": det, "eventId": fmt.Sprintf("evt_%s_%d", s, i), "eventTypeName": event,
			"eventTimestamp": time.Now().UTC().Format(time.RFC3339), "entities": []any{map[string]any{"entityType": "customer_" + s, "entityId": "c1"}},
			"eventVariables": map[string]any{amount: value}})
		fmt.Printf("payment of %s -> %v\n", value, out["ruleResults"])
	}
	stored := c.call("GetEvent", map[string]any{"eventId": "evt_" + s + "_1", "eventTypeName": event})["event"].(map[string]any)
	fmt.Printf("stored event %v with variables %v\n", stored["eventId"], stored["eventVariables"])
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": c.target + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request and exits unless it returns 2xx.
func (c client) send(op string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
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
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, c.service, c.region, time.Now()); err != nil {
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
