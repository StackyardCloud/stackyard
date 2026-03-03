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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type apiCall struct {
	Action  string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	policyID := "policy-advanced-0001"
	resourceARN := "arn:aws:fms:us-east-1:123456789012:policy/" + policyID

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard FMS advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Action: "PutAdminAccount", Payload: map[string]any{"AdminAccount": "123456789012"}},
		{Action: "PutNotificationChannel", Payload: map[string]any{"SnsTopicArn": "arn:aws:sns:us-east-1:123456789012:stackyard-fms", "SnsRoleName": "stackyard-fms-role"}},
		{Action: "PutAppsList", Payload: map[string]any{"AppsList": map[string]any{"ListId": "apps-advanced-0001", "ListName": "stackyard-apps-advanced", "AppsList": []any{map[string]any{"AppName": "example", "Protocol": "TCP", "Port": 443}}}}},
		{Action: "PutProtocolsList", Payload: map[string]any{"ProtocolsList": map[string]any{"ListId": "protocols-advanced-0001", "ListName": "stackyard-protocols-advanced", "ProtocolsList": []any{"TCP", "UDP"}}}},
		{Action: "PutResourceSet", Payload: map[string]any{"ResourceSet": map[string]any{"Id": "rs-advanced-0001", "Name": "stackyard-rs-advanced", "ResourceTypeList": []any{"AWS::EC2::Instance"}}}},
		{Action: "PutPolicy", Payload: map[string]any{"Policy": map[string]any{"PolicyId": policyID, "PolicyName": "stackyard-policy-advanced", "ResourceType": "AWS::EC2::Instance", "RemediationEnabled": true, "SecurityServicePolicyData": map[string]any{"Type": "WAF", "ManagedServiceData": "{}"}}}},
		{Action: "GetPolicy", Payload: map[string]any{"PolicyId": policyID}},
		{Action: "ListPolicies", Payload: map[string]any{"MaxResults": 10}},
		{Action: "TagResource", Payload: map[string]any{"ResourceArn": resourceARN, "TagList": []any{map[string]any{"Key": "env", "Value": "advanced"}}}},
		{Action: "ListTagsForResource", Payload: map[string]any{"ResourceArn": resourceARN}},
		{Action: "UntagResource", Payload: map[string]any{"ResourceArn": resourceARN, "TagKeys": []any{"env"}}},
		{Action: "AssociateThirdPartyFirewall", Payload: map[string]any{"ThirdPartyFirewall": "PALO_ALTO_NETWORKS", "FirewallPolicyName": "stackyard-third-party-policy"}},
		{Action: "GetThirdPartyFirewallAssociationStatus", Payload: map[string]any{}},
		{Action: "ListThirdPartyFirewallFirewallPolicies", Payload: map[string]any{"ThirdPartyFirewall": "PALO_ALTO_NETWORKS", "MaxResults": 10}},
	}

	for _, call := range calls {
		status, body, err := fmsRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Action, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Action, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s returned %d\n", call.Action, status)
	}

	fmt.Println("Done.")
}

func fmsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	url := strings.TrimRight(endpoint, "/") + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSFMS_20180101."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "fms", region, time.Now()); err != nil {
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
