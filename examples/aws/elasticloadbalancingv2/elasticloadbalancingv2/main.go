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

type call struct {
	Action string
	Params url.Values
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

	calls := []call{
		{Action: "DescribeLoadBalancers", Params: url.Values{}},
		{Action: "DescribeLoadBalancerAttributes", Params: url.Values{}},
		{Action: "DescribeInstanceHealth", Params: url.Values{}},
		{Action: "DescribeTags", Params: url.Values{}},
		{Action: "DescribeAccountLimits", Params: url.Values{}},
	}

	fmt.Printf("Stackyard ELB (2012-06-01) advanced client using %s\n", endpoint)
	for _, c := range calls {
		params := cloneValues(c.Params)
		params.Set("Action", c.Action)
		params.Set("Version", "2012-06-01")

		status, body, err := elbRequest(ctx, endpoint, region, creds, params)
		if err != nil {
			exitf("%s failed: %v", c.Action, err)
		}

		bodyStr := strings.TrimSpace(string(body))
		if status >= 200 && status < 300 {
			fmt.Printf("%s succeeded (%d)\n", c.Action, status)
			continue
		}
		if strings.Contains(bodyStr, "NotImplemented") {
			fmt.Printf("%s returned %d (NotImplemented): expected while staged plan is in progress\n", c.Action, status)
			continue
		}
		exitf("%s returned HTTP %d: %s", c.Action, status, bodyStr)
	}

	fmt.Println("Done.")
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vals := range v {
		cp := make([]string, len(vals))
		copy(cp, vals)
		out[k] = cp
	}
	return out
}

func elbRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, params url.Values) (int, []byte, error) {
	if params.Get("Version") == "" {
		params.Set("Version", "2012-06-01")
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
