// Command iot-device-advisor runs AWS IoT Core Device Advisor against
// Stackyard: create an MQTT connect suite for a device, read the test
// endpoint, start a suite run and poll it to completion, print the test
// results, then delete the suite.
//
// Stackyard advances a run one step per read (PENDING, RUNNING, PASS); no
// device has to connect. The SDK has no Device Advisor module in this
// repository's go.mod, so requests are signed by hand.
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

const signingName = "iotdeviceadvisor"

func main() {
	c := newClient()
	fmt.Printf("Stackyard IoT Device Advisor client using %s\n", c.endpoint)

	thingArn := "arn:aws:iot:us-east-1:123456789012:thing/sensor-1"
	root, _ := json.Marshal(map[string]any{"configuration": map[string]any{}, "tests": []any{map[string]any{
		"name": "MQTT Connect", "configuration": map[string]any{},
		"tests": []any{map[string]any{"name": "MQTT_Connect", "configuration": map[string]any{}, "test": map[string]any{"id": "MQTT_Connect", "version": "0.0.0"}}},
	}}})
	suite := c.rest("CreateSuiteDefinition", http.MethodPost, "/suiteDefinitions", map[string]any{
		"suiteDefinitionConfiguration": map[string]any{
			"suiteDefinitionName": "sensor-mqtt", "rootGroup": string(root), "protocol": "MqttV5",
			"devicePermissionRoleArn": "arn:aws:iam::123456789012:role/device-advisor",
			"devices":                 []any{map[string]any{"thingArn": thingArn}},
		},
		"tags": map[string]any{"team": "firmware"},
	})
	id := suite["suiteDefinitionId"].(string)
	fmt.Printf("suite %s (%v)\n", id, suite["suiteDefinitionName"])
	fmt.Println("test endpoint:", c.rest("GetEndpoint", http.MethodGet, "/endpoint?thingArn="+thingArn, nil)["endpoint"])

	run := c.rest("StartSuiteRun", http.MethodPost, "/suiteDefinitions/"+id+"/suiteRuns", map[string]any{
		"suiteRunConfiguration": map[string]any{"primaryDevice": map[string]any{"thingArn": thingArn}},
	})
	path := "/suiteDefinitions/" + id + "/suiteRuns/" + run["suiteRunId"].(string)
	var result map[string]any
	for i := 0; i < 5; i++ {
		result = c.rest("GetSuiteRun", http.MethodGet, path, nil)
		fmt.Println("run status:", result["status"])
		if result["status"] == "PASS" || result["status"] == "FAIL" {
			break
		}
	}
	for _, g := range result["testResult"].(map[string]any)["groups"].([]any) {
		for _, t := range g.(map[string]any)["tests"].([]any) {
			fmt.Printf("  %v / %v: %v\n", g.(map[string]any)["groupName"], t.(map[string]any)["testCaseDefinitionName"], t.(map[string]any)["status"])
		}
	}
	for _, r := range c.rest("ListSuiteRuns", http.MethodGet, "/suiteRuns?suiteDefinitionId="+id, nil)["suiteRunsList"].([]any) {
		fmt.Printf("run %v: %v passed, %v failed\n", r.(map[string]any)["suiteRunId"], r.(map[string]any)["passed"], r.(map[string]any)["failed"])
	}

	c.rest("DeleteSuiteDefinition", http.MethodDelete, "/suiteDefinitions/"+id, nil)
	fmt.Println("deleted suite", id)
}

// rest sends one REST JSON operation and exits unless it returns 2xx.
func (c client) rest(op, method, path string, payload any) map[string]any {
	var body []byte
	headers := map[string]string{}
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
		headers["Content-Type"] = "application/json"
	}
	raw := c.send(op, method, signingName, c.endpoint+path, body, headers)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
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
