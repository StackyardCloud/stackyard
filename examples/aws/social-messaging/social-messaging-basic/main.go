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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Amazon End User Messaging Social basic client using %s\n", endpoint)

	status, body, err := socialMessagingRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodGet,
		"/v1/whatsapp/waba/list",
		nil,
		nil,
	)
	if err != nil {
		exitf("ListLinkedWhatsAppBusinessAccounts request failed: %v", err)
	}

	if status < 200 || status >= 300 {
		exitf("ListLinkedWhatsAppBusinessAccounts returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}

	logf("ListLinkedWhatsAppBusinessAccounts returned %d", status)
	fmt.Println(strings.TrimSpace(string(body)))
}

func socialMessagingRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method string,
	path string,
	query map[string]string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil && method != http.MethodGet && method != http.MethodDelete {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	u, err := url.Parse(strings.TrimRight(endpoint, "/") + path)
	if err != nil {
		return 0, nil, err
	}
	if len(query) > 0 {
		q := u.Query()
		for key, value := range query {
			if strings.TrimSpace(value) == "" {
				continue
			}
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "social-messaging", region, time.Now()); err != nil {
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
