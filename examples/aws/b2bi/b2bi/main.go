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

type rpcCall struct {
	Name    string
	Action  string
	Payload map[string]any
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

	fmt.Printf("Stackyard B2BI advanced client using %s\n", endpoint)

	capabilityOut, err := runCall(ctx, endpoint, region, creds, rpcCall{
		Name:   "CreateCapability",
		Action: "CreateCapability",
		Payload: map[string]any{
			"capabilityId": "cap-001",
			"name":         "advanced-capability",
		},
	})
	if err != nil {
		exitf("CreateCapability failed: %v", err)
	}
	capabilityID := nestedString(capabilityOut, "capability", "capabilityId", "cap-001")

	profileOut, err := runCall(ctx, endpoint, region, creds, rpcCall{
		Name:   "CreateProfile",
		Action: "CreateProfile",
		Payload: map[string]any{
			"profileId": "prof-001",
			"name":      "advanced-profile",
		},
	})
	if err != nil {
		exitf("CreateProfile failed: %v", err)
	}
	profileID := nestedString(profileOut, "profile", "profileId", "prof-001")

	partnershipOut, err := runCall(ctx, endpoint, region, creds, rpcCall{
		Name:   "CreatePartnership",
		Action: "CreatePartnership",
		Payload: map[string]any{
			"partnershipId": "part-001",
			"profileId":     profileID,
			"capabilityId":  capabilityID,
			"name":          "advanced-partnership",
		},
	})
	if err != nil {
		exitf("CreatePartnership failed: %v", err)
	}
	partnershipID := nestedString(partnershipOut, "partnership", "partnershipId", "part-001")

	transformerOut, err := runCall(ctx, endpoint, region, creds, rpcCall{
		Name:   "CreateTransformer",
		Action: "CreateTransformer",
		Payload: map[string]any{
			"transformerId": "trf-001",
			"name":          "advanced-transformer",
		},
	})
	if err != nil {
		exitf("CreateTransformer failed: %v", err)
	}
	transformerID := nestedString(transformerOut, "transformer", "transformerId", "trf-001")
	resourceARN := nestedString(transformerOut, "transformer", "transformerArn", "arn:aws:b2bi:us-east-1:123456789012:transformer/trf-001")

	startJobOut, err := runCall(ctx, endpoint, region, creds, rpcCall{
		Name:   "StartTransformerJob",
		Action: "StartTransformerJob",
		Payload: map[string]any{
			"transformerId":    transformerID,
			"transformerJobId": "job-001",
		},
	})
	if err != nil {
		exitf("StartTransformerJob failed: %v", err)
	}
	transformerJobID := nestedString(startJobOut, "transformerJob", "transformerJobId", "job-001")

	calls := []rpcCall{
		{Name: "GetCapability", Action: "GetCapability", Payload: map[string]any{"capabilityId": capabilityID}},
		{Name: "ListCapabilities", Action: "ListCapabilities", Payload: map[string]any{}},
		{Name: "UpdateCapability", Action: "UpdateCapability", Payload: map[string]any{"capabilityId": capabilityID, "name": "advanced-capability-updated"}},
		{Name: "GetProfile", Action: "GetProfile", Payload: map[string]any{"profileId": profileID}},
		{Name: "ListProfiles", Action: "ListProfiles", Payload: map[string]any{}},
		{Name: "UpdateProfile", Action: "UpdateProfile", Payload: map[string]any{"profileId": profileID, "name": "advanced-profile-updated"}},
		{Name: "GetPartnership", Action: "GetPartnership", Payload: map[string]any{"partnershipId": partnershipID}},
		{Name: "ListPartnerships", Action: "ListPartnerships", Payload: map[string]any{}},
		{Name: "UpdatePartnership", Action: "UpdatePartnership", Payload: map[string]any{"partnershipId": partnershipID, "name": "advanced-partnership-updated"}},
		{Name: "GetTransformer", Action: "GetTransformer", Payload: map[string]any{"transformerId": transformerID}},
		{Name: "ListTransformers", Action: "ListTransformers", Payload: map[string]any{}},
		{Name: "UpdateTransformer", Action: "UpdateTransformer", Payload: map[string]any{"transformerId": transformerID, "name": "advanced-transformer-updated"}},
		{Name: "GetTransformerJob", Action: "GetTransformerJob", Payload: map[string]any{"transformerJobId": transformerJobID}},
		{Name: "CreateStarterMappingTemplate", Action: "CreateStarterMappingTemplate", Payload: map[string]any{}},
		{Name: "GenerateMapping", Action: "GenerateMapping", Payload: map[string]any{}},
		{Name: "TestConversion", Action: "TestConversion", Payload: map[string]any{}},
		{Name: "TestMapping", Action: "TestMapping", Payload: map[string]any{}},
		{Name: "TestParsing", Action: "TestParsing", Payload: map[string]any{}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"resourceArn": resourceARN, "tags": map[string]any{"env": "advanced", "owner": "qa"}}},
		{Name: "ListTagsForResource", Action: "ListTagsForResource", Payload: map[string]any{"resourceArn": resourceARN}},
		{Name: "UntagResource", Action: "UntagResource", Payload: map[string]any{"resourceArn": resourceARN, "tagKeys": []string{"owner"}}},
		{Name: "DeletePartnership", Action: "DeletePartnership", Payload: map[string]any{"partnershipId": partnershipID}},
		{Name: "DeleteProfile", Action: "DeleteProfile", Payload: map[string]any{"profileId": profileID}},
		{Name: "DeleteCapability", Action: "DeleteCapability", Payload: map[string]any{"capabilityId": capabilityID}},
		{Name: "DeleteTransformer", Action: "DeleteTransformer", Payload: map[string]any{"transformerId": transformerID}},
	}

	for _, call := range calls {
		if _, err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) (map[string]any, error) {
	status, body, err := b2biRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			trimmed = "<empty body>"
		}
		return nil, fmt.Errorf("HTTP %d: %s", status, trimmed)
	}

	fmt.Printf("%s returned %d\n", call.Name, status)

	out := map[string]any{}
	if len(strings.TrimSpace(string(body))) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return out, nil
}

func b2biRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte("{}")
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "B2BI."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "b2bi", region, time.Now()); err != nil {
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

func nestedString(payload map[string]any, container, key, def string) string {
	if payload == nil {
		return def
	}
	raw, ok := payload[container]
	if !ok || raw == nil {
		return def
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return def
	}
	value, ok := m[key].(string)
	if !ok {
		return def
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return def
	}
	return value
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
