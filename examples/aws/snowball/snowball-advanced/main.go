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

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Snowball advanced client using %s\n", endpoint)

	addressID := "ADID000000000000000000000000000000000000"
	clusterID := "CID000000000000000000000000000000000000"
	jobID := "JID000000000000000000000000000000000000"
	pricingID := "LTP000000000000000000000"

	status, body, err := snowballRequest(ctx, endpoint, region, creds, "CreateAddress", map[string]any{
		"Address": map[string]any{
			"Name":            "Stackyard Coverage",
			"Street1":         "1 Stackyard Way",
			"City":            "Seattle",
			"StateOrProvince": "WA",
			"Country":         "US",
			"PostalCode":      "98101",
		},
	})
	mustSuccess("CreateAddress", status, body, err)
	addressID = extractString(body, "AddressId", addressID)

	status, body, err = snowballRequest(ctx, endpoint, region, creds, "CreateCluster", map[string]any{
		"AddressId":      addressID,
		"JobType":        "IMPORT",
		"ShippingOption": "SECOND_DAY",
		"SnowballType":   "STANDARD",
		"Description":    "Stackyard cluster",
	})
	mustSuccess("CreateCluster", status, body, err)
	clusterID = extractString(body, "ClusterId", clusterID)

	status, body, err = snowballRequest(ctx, endpoint, region, creds, "CreateJob", map[string]any{
		"AddressId":      addressID,
		"ClusterId":      clusterID,
		"JobType":        "IMPORT",
		"ShippingOption": "SECOND_DAY",
		"SnowballType":   "STANDARD",
		"Description":    "Stackyard job",
	})
	mustSuccess("CreateJob", status, body, err)
	jobID = extractString(body, "JobId", jobID)

	status, body, err = snowballRequest(ctx, endpoint, region, creds, "CreateLongTermPricing", map[string]any{
		"LongTermPricingType": "ONE_YEAR",
		"SnowballType":        "STANDARD",
	})
	mustSuccess("CreateLongTermPricing", status, body, err)
	pricingID = extractString(body, "LongTermPricingId", pricingID)

	calls := []apiCall{
		{Action: "DescribeAddress", Payload: map[string]any{"AddressId": addressID}},
		{Action: "DescribeAddresses", Payload: map[string]any{}},
		{Action: "DescribeCluster", Payload: map[string]any{"ClusterId": clusterID}},
		{Action: "DescribeJob", Payload: map[string]any{"JobId": jobID}},
		{Action: "DescribeReturnShippingLabel", Payload: map[string]any{"JobId": jobID}},
		{Action: "GetJobManifest", Payload: map[string]any{"JobId": jobID}},
		{Action: "GetJobUnlockCode", Payload: map[string]any{"JobId": jobID}},
		{Action: "GetSnowballUsage", Payload: map[string]any{}},
		{Action: "GetSoftwareUpdates", Payload: map[string]any{"JobId": jobID}},
		{Action: "ListClusterJobs", Payload: map[string]any{"ClusterId": clusterID}},
		{Action: "ListClusters", Payload: map[string]any{"MaxResults": 10}},
		{Action: "ListCompatibleImages", Payload: map[string]any{}},
		{Action: "ListJobs", Payload: map[string]any{"MaxResults": 10}},
		{Action: "ListLongTermPricing", Payload: map[string]any{"MaxResults": 10}},
		{Action: "ListPickupLocations", Payload: map[string]any{}},
		{Action: "ListServiceVersions", Payload: map[string]any{"ServiceName": "snowball"}},
		{Action: "UpdateCluster", Payload: map[string]any{"ClusterId": clusterID, "Description": "updated"}},
		{Action: "UpdateJob", Payload: map[string]any{"JobId": jobID, "Description": "updated"}},
		{Action: "UpdateJobShipmentState", Payload: map[string]any{"JobId": jobID, "ShipmentState": "RECEIVED"}},
		{Action: "UpdateLongTermPricing", Payload: map[string]any{"LongTermPricingId": pricingID, "IsLongTermPricingAutoRenew": true}},
		{Action: "CreateReturnShippingLabel", Payload: map[string]any{"JobId": jobID}},
		{Action: "CancelJob", Payload: map[string]any{"JobId": jobID}},
		{Action: "CancelCluster", Payload: map[string]any{"ClusterId": clusterID}},
	}

	for _, call := range calls {
		status, body, err = snowballRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
		mustSuccess(call.Action, status, body, err)
		fmt.Printf("%s returned %d\n", call.Action, status)
	}

	fmt.Println("Done.")
}

func snowballRequest(
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

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSIESnowballJobManagementService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "snowball", region, time.Now()); err != nil {
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

func extractString(body []byte, key, fallback string) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	value, ok := payload[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return strings.TrimSpace(text)
}

func mustSuccess(action string, status int, body []byte, err error) {
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", action, status, strings.TrimSpace(string(body)))
	}
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
