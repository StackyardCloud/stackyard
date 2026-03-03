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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type apiCall struct {
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

	fmt.Printf("Stackyard Payment Cryptography Data advanced client using %s\n", endpoint)

	keyARN := "arn:aws:payment-cryptography:us-east-1:123456789012:key/stackyard-key"
	keyParam := url.PathEscape(keyARN)

	calls := []apiCall{
		{Name: "DecryptData", Method: http.MethodPost, Path: "/keys/" + keyParam + "/decrypt", Payload: map[string]any{}},
		{Name: "EncryptData", Method: http.MethodPost, Path: "/keys/" + keyParam + "/encrypt", Payload: map[string]any{}},
		{Name: "GenerateAs2805KekValidation", Method: http.MethodPost, Path: "/as2805kekvalidation/generate", Payload: map[string]any{}},
		{Name: "GenerateCardValidationData", Method: http.MethodPost, Path: "/cardvalidationdata/generate", Payload: map[string]any{}},
		{Name: "GenerateMac", Method: http.MethodPost, Path: "/mac/generate", Payload: map[string]any{}},
		{Name: "GenerateMacEmvPinChange", Method: http.MethodPost, Path: "/macemvpinchange/generate", Payload: map[string]any{}},
		{Name: "GeneratePinData", Method: http.MethodPost, Path: "/pindata/generate", Payload: map[string]any{}},
		{Name: "ReEncryptData", Method: http.MethodPost, Path: "/keys/" + keyParam + "/reencrypt", Payload: map[string]any{}},
		{Name: "TranslateKeyMaterial", Method: http.MethodPost, Path: "/keymaterial/translate", Payload: map[string]any{}},
		{Name: "TranslatePinData", Method: http.MethodPost, Path: "/pindata/translate", Payload: map[string]any{}},
		{Name: "VerifyAuthRequestCryptogram", Method: http.MethodPost, Path: "/cryptogram/verify", Payload: map[string]any{}},
		{Name: "VerifyCardValidationData", Method: http.MethodPost, Path: "/cardvalidationdata/verify", Payload: map[string]any{}},
		{Name: "VerifyMac", Method: http.MethodPost, Path: "/mac/verify", Payload: map[string]any{}},
		{Name: "VerifyPinData", Method: http.MethodPost, Path: "/pindata/verify", Payload: map[string]any{}},
	}

	for _, call := range calls {
		status, body, err := paymentCryptographyDataRequest(
			ctx,
			endpoint,
			region,
			creds,
			call.Method,
			call.Path,
			call.Payload,
		)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", call.Name, status)
	}

	fmt.Println("Done.")
}

func paymentCryptographyDataRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	url := strings.TrimRight(endpoint, "/") + path
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payloadBytes), "payment-cryptography", region, time.Now()); err != nil {
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
