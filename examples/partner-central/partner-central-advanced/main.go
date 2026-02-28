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
	Target  string
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

	fmt.Printf("Stackyard Partner Central advanced client using %s\n", endpoint)

	status, body, err := jsonRPCRequest(ctx, endpoint, region, creds, "PartnerCentralSelling.CreateOpportunity", map[string]any{"Catalog": "AWS", "Title": "stackyard-opportunity"})
	if err != nil {
		exitf("CreateOpportunity request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateOpportunity returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	opportunityID := extractIdentifier(body, "opty-0000000000001")

	status, body, err = jsonRPCRequest(ctx, endpoint, region, creds, "PartnerCentralSelling.CreateEngagement", map[string]any{"Catalog": "AWS", "OpportunityIdentifier": opportunityID})
	if err != nil {
		exitf("CreateEngagement request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateEngagement returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	engagementID := extractIdentifier(body, "engi-0000000000001")

	status, body, err = jsonRPCRequest(ctx, endpoint, region, creds, "PartnerCentralSelling.CreateEngagementInvitation", map[string]any{"Catalog": "AWS", "EngagementIdentifier": engagementID, "OpportunityIdentifier": opportunityID})
	if err != nil {
		exitf("CreateEngagementInvitation request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateEngagementInvitation returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	invitationID := extractIdentifier(body, "eginv-0000000000001")

	status, body, err = jsonRPCRequest(ctx, endpoint, region, creds, "PartnerCentralSelling.CreateResourceSnapshot", map[string]any{"Catalog": "AWS"})
	if err != nil {
		exitf("CreateResourceSnapshot request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateResourceSnapshot returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	snapshotID := extractIdentifier(body, "rsnp-0000000000001")

	status, body, err = jsonRPCRequest(ctx, endpoint, region, creds, "PartnerCentralSelling.CreateResourceSnapshotJob", map[string]any{"Catalog": "AWS", "ResourceSnapshotIdentifier": snapshotID})
	if err != nil {
		exitf("CreateResourceSnapshotJob request failed: %v", err)
	}
	if status < 200 || status >= 300 {
		exitf("CreateResourceSnapshotJob returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	snapshotJobID := extractIdentifier(body, "rsj-0000000000001")

	calls := []rpcCall{
		{Target: "PartnerCentralSelling.ListSolutions", Payload: map[string]any{"Catalog": "AWS"}},
		{Target: "PartnerCentralSelling.ListOpportunities", Payload: map[string]any{"Catalog": "AWS"}},
		{Target: "PartnerCentralSelling.GetOpportunity", Payload: map[string]any{"Catalog": "AWS", "Identifier": opportunityID}},
		{Target: "PartnerCentralSelling.UpdateOpportunity", Payload: map[string]any{"Catalog": "AWS", "Identifier": opportunityID, "Status": "Qualified"}},
		{Target: "PartnerCentralSelling.SubmitOpportunity", Payload: map[string]any{"Catalog": "AWS", "Identifier": opportunityID}},
		{Target: "PartnerCentralSelling.GetAwsOpportunitySummary", Payload: map[string]any{"Catalog": "AWS", "Identifier": opportunityID}},
		{Target: "PartnerCentralSelling.GetEngagement", Payload: map[string]any{"Catalog": "AWS", "Identifier": engagementID}},
		{Target: "PartnerCentralSelling.ListEngagements", Payload: map[string]any{"Catalog": "AWS"}},
		{Target: "PartnerCentralSelling.GetEngagementInvitation", Payload: map[string]any{"Catalog": "AWS", "Identifier": invitationID}},
		{Target: "PartnerCentralSelling.AcceptEngagementInvitation", Payload: map[string]any{"Catalog": "AWS", "Identifier": invitationID}},
		{Target: "PartnerCentralSelling.StartEngagementByAcceptingInvitationTask", Payload: map[string]any{"Catalog": "AWS", "Identifier": invitationID}},
		{Target: "PartnerCentralSelling.StartEngagementFromOpportunityTask", Payload: map[string]any{"Catalog": "AWS", "Identifier": opportunityID}},
		{Target: "PartnerCentralSelling.ListEngagementByAcceptingInvitationTasks", Payload: map[string]any{"Catalog": "AWS"}},
		{Target: "PartnerCentralSelling.ListEngagementFromOpportunityTasks", Payload: map[string]any{"Catalog": "AWS"}},
		{Target: "PartnerCentralSelling.GetSellingSystemSettings", Payload: map[string]any{}},
		{Target: "PartnerCentralSelling.PutSellingSystemSettings", Payload: map[string]any{"Catalog": "AWS", "OpportunityVisibility": "ALL"}},
		{Target: "PartnerCentralSelling.GetResourceSnapshot", Payload: map[string]any{"Catalog": "AWS", "Identifier": snapshotID}},
		{Target: "PartnerCentralSelling.ListResourceSnapshots", Payload: map[string]any{"Catalog": "AWS"}},
		{Target: "PartnerCentralSelling.GetResourceSnapshotJob", Payload: map[string]any{"Catalog": "AWS", "Identifier": snapshotJobID}},
		{Target: "PartnerCentralSelling.StartResourceSnapshotJob", Payload: map[string]any{"Catalog": "AWS", "Identifier": snapshotJobID}},
		{Target: "PartnerCentralSelling.StopResourceSnapshotJob", Payload: map[string]any{"Catalog": "AWS", "Identifier": snapshotJobID}},
		{Target: "PartnerCentralSelling.ListResourceSnapshotJobs", Payload: map[string]any{"Catalog": "AWS"}},
		{Target: "PartnerCentralSelling.DeleteResourceSnapshotJob", Payload: map[string]any{"Catalog": "AWS", "Identifier": snapshotJobID}},
	}

	for _, call := range calls {
		status, body, err := jsonRPCRequest(ctx, endpoint, region, creds, call.Target, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Target, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Target, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s returned %d\n", call.Target, status)
	}

	fmt.Println("Done.")
}

func extractIdentifier(body []byte, fallback string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	if v, ok := payload["Identifier"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["identifier"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func jsonRPCRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	target string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
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
	req.Header.Set("X-Amz-Target", target)
	req.Header.Set("Accept", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "partnercentral-selling", region, time.Now()); err != nil {
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
