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
	profileName := getenv("STACKYARD_SIGNER_PROFILE", "stackyard-signer-basic")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Signer basic client using %s\n", endpoint)

	putPath := "/signing-profiles/" + url.PathEscape(profileName)
	putStatus, putBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodPut, putPath, map[string]any{
		"platformId": "AWSLambda-SHA384-ECDSA",
	})
	if err != nil {
		exitf("put signing profile: %v", err)
	}
	if err := expectOK("PutSigningProfile", putStatus, putBody); err != nil {
		exitf("put signing profile: %v", err)
	}
	logf("PutSigningProfile succeeded (%d)", putStatus)

	getStatus, getBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodGet, putPath, nil)
	if err != nil {
		exitf("get signing profile: %v", err)
	}
	if err := expectOK("GetSigningProfile", getStatus, getBody); err != nil {
		exitf("get signing profile: %v", err)
	}
	logf("GetSigningProfile succeeded (%d)", getStatus)

	listStatus, listBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodGet, "/signing-profiles?maxResults=10", nil)
	if err != nil {
		exitf("list signing profiles: %v", err)
	}
	if err := expectOK("ListSigningProfiles", listStatus, listBody); err != nil {
		exitf("list signing profiles: %v", err)
	}
	logf("ListSigningProfiles succeeded (%d)", listStatus)

	platformStatus, platformBody, err := signerRequest(ctx, endpoint, region, creds, http.MethodGet, "/signing-platforms/AWSLambda-SHA384-ECDSA", nil)
	if err != nil {
		exitf("get signing platform: %v", err)
	}
	if err := expectOK("GetSigningPlatform", platformStatus, platformBody); err != nil {
		exitf("get signing platform: %v", err)
	}
	logf("GetSigningPlatform succeeded (%d)", platformStatus)

	fmt.Println("Done.")
}

func signerRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	requestURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "signer", region, time.Now()); err != nil {
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

func expectOK(action string, status int, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	return nil
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
