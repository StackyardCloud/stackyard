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

type restCall struct {
	Name    string
	Method  string
	Path    string
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

	fmt.Printf("Stackyard Artifact advanced client using %s\n", endpoint)

	agreementID := getenv("STACKYARD_ARTIFACT_AGREEMENT_ID", "agr-000001")
	customerAgreementID := getenv("STACKYARD_ARTIFACT_CUSTOMER_AGREEMENT_ID", "cagr-000001")
	reportID := getenv("STACKYARD_ARTIFACT_REPORT_ID", "rpt-000001")

	calls := []restCall{
		{Name: "ListAgreements", Method: http.MethodGet, Path: "/v1/agreement/list?maxResults=10"},
		{Name: "AcceptAgreement", Method: http.MethodPost, Path: "/v1/agreement/accept", Payload: map[string]any{"agreementId": agreementID}},
		{Name: "GetAgreement", Method: http.MethodGet, Path: "/v1/agreement/get?agreementId=" + agreementID},
		{Name: "AcceptNdaForAgreement", Method: http.MethodPost, Path: "/v1/agreement/acceptNdaForAgreement", Payload: map[string]any{"agreementId": agreementID}},
		{Name: "GetNdaForAgreement", Method: http.MethodGet, Path: "/v1/agreement/getNdaForAgreement?agreementId=" + agreementID},
		{Name: "ListCustomerAgreements", Method: http.MethodGet, Path: "/v1/customer-agreement/list?maxResults=10"},
		{Name: "GetCustomerAgreement", Method: http.MethodGet, Path: "/v1/customer-agreement/get?customerAgreementId=" + customerAgreementID},
		{Name: "ListReports", Method: http.MethodGet, Path: "/v1/report/list?maxResults=10"},
		{Name: "ListReportVersions", Method: http.MethodGet, Path: "/v1/report/listVersions?reportId=" + reportID + "&maxResults=10"},
		{Name: "GetReportMetadata", Method: http.MethodGet, Path: "/v1/report/getMetadata?reportId=" + reportID},
		{Name: "GetTermForReport", Method: http.MethodGet, Path: "/v1/report/getTermForReport?reportId=" + reportID},
		{Name: "GetReport", Method: http.MethodGet, Path: "/v1/report/get?reportId=" + reportID},
		{Name: "GetAccountSettings", Method: http.MethodGet, Path: "/v1/account-settings/get"},
		{Name: "PutAccountSettings", Method: http.MethodPut, Path: "/v1/account-settings/put", Payload: map[string]any{"notificationsEnabled": false, "defaultReportFormat": "JSON"}},
		{Name: "TerminateAgreement", Method: http.MethodPost, Path: "/v1/customer-agreement/terminate", Payload: map[string]any{"customerAgreementId": customerAgreementID}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	var payload []byte
	var err error
	if call.Payload != nil {
		payload, err = json.Marshal(call.Payload)
		if err != nil {
			return err
		}
	}

	status, body, err := artifactRequest(ctx, endpoint, region, creds, call.Method, call.Path, payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			trimmed = "<empty body>"
		}
		return fmt.Errorf("HTTP %d: %s", status, trimmed)
	}

	fmt.Printf("%s returned %d\n", call.Name, status)
	return nil
}

func artifactRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	body := payload
	if body == nil {
		body = []byte{}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "artifact", region, time.Now()); err != nil {
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
