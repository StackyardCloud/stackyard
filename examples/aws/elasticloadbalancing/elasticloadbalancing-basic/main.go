package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	fmt.Printf("Stackyard ELB basic client using %s\n", endpoint)

	status, body, err := elbRequest(ctx, endpoint, region, creds, url.Values{
		"Action":  []string{"DescribeLoadBalancers"},
		"Version": []string{"2015-12-01"},
	})
	if err != nil {
		exitf("DescribeLoadBalancers failed: %v", err)
	}

	bodyStr := strings.TrimSpace(string(body))
	if status >= 200 && status < 300 {
		fmt.Println(bodyStr)
		return
	}
	if strings.Contains(bodyStr, "NotImplemented") {
		fmt.Printf("DescribeLoadBalancers returned %d (NotImplemented): expected while staged plan is in progress\n", status)
		return
	}
	exitf("DescribeLoadBalancers returned HTTP %d: %s", status, bodyStr)
}

func elbRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, params url.Values) (int, []byte, error) {
	if params.Get("Version") == "" {
		params.Set("Version", "2015-12-01")
	}
	body := []byte(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", strings.NewReader(string(body)))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "elasticloadbalancing", region, time.Now()); err != nil {
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
