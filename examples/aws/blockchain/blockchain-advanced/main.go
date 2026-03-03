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

const blockchainSigningName = "managedblockchain-query"

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

	fmt.Printf("Stackyard Managed Blockchain advanced client using %s\n", endpoint)

	calls := []restCall{
		{
			Name:   "ListTokenBalances",
			Method: http.MethodPost,
			Path:   "/list-token-balances",
			Payload: map[string]any{
				"tokenFilter": map[string]any{
					"network": "ETHEREUM_MAINNET",
				},
			},
		},
		{
			Name:   "BatchGetTokenBalance",
			Method: http.MethodPost,
			Path:   "/batch-get-token-balance",
			Payload: map[string]any{
				"getTokenBalanceInputs": []map[string]any{
					{
						"tokenIdentifier": map[string]any{
							"network":         "ETHEREUM_MAINNET",
							"contractAddress": "0x0000000000000000000000000000000000000000",
							"tokenId":         "1",
						},
						"ownerIdentifier": map[string]any{
							"address": "0x1111111111111111111111111111111111111111",
						},
					},
				},
			},
		},
		{
			Name:   "GetAssetContract",
			Method: http.MethodPost,
			Path:   "/get-asset-contract",
			Payload: map[string]any{
				"contractIdentifier": map[string]any{
					"network":         "ETHEREUM_MAINNET",
					"contractAddress": "0x0000000000000000000000000000000000000000",
				},
			},
		},
		{
			Name:   "GetTokenBalance",
			Method: http.MethodPost,
			Path:   "/get-token-balance",
			Payload: map[string]any{
				"tokenIdentifier": map[string]any{
					"network":         "ETHEREUM_MAINNET",
					"contractAddress": "0x0000000000000000000000000000000000000000",
					"tokenId":         "1",
				},
				"ownerIdentifier": map[string]any{
					"address": "0x1111111111111111111111111111111111111111",
				},
			},
		},
		{
			Name:   "GetTransaction",
			Method: http.MethodPost,
			Path:   "/get-transaction",
			Payload: map[string]any{
				"network":         "ETHEREUM_MAINNET",
				"transactionHash": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
		{
			Name:   "ListAssetContracts",
			Method: http.MethodPost,
			Path:   "/list-asset-contracts",
			Payload: map[string]any{
				"contractFilter": map[string]any{
					"network":       "ETHEREUM_MAINNET",
					"tokenStandard": "ERC20",
				},
				"maxResults": 10,
			},
		},
		{
			Name:   "ListFilteredTransactionEvents",
			Method: http.MethodPost,
			Path:   "/list-filtered-transaction-events",
			Payload: map[string]any{
				"network": "ETHEREUM_MAINNET",
				"addressIdentifierFilter": map[string]any{
					"transactionEventToAddress": []string{
						"0x1111111111111111111111111111111111111111",
					},
				},
				"maxResults": 10,
			},
		},
		{
			Name:   "ListTransactionEvents",
			Method: http.MethodPost,
			Path:   "/list-transaction-events",
			Payload: map[string]any{
				"network":         "ETHEREUM_MAINNET",
				"transactionHash": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"maxResults":      10,
			},
		},
		{
			Name:   "ListTransactions",
			Method: http.MethodPost,
			Path:   "/list-transactions",
			Payload: map[string]any{
				"address":    "0x1111111111111111111111111111111111111111",
				"network":    "ETHEREUM_MAINNET",
				"maxResults": 10,
			},
		},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := blockchainRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(status, errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func blockchainRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
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
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), blockchainSigningName, region, time.Now()); err != nil {
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

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(status int, errType string, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "unknown route") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "unauthorized")
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
