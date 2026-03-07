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

	metricNamespace := "Stackyard/CloudWatch"
	metricName := "CoverageMetric"

	calls := []call{
		{Action: "PutMetricData", Params: url.Values{
			"Namespace":                      []string{metricNamespace},
			"MetricData.member.1.MetricName": []string{metricName},
			"MetricData.member.1.Value":      []string{"1"},
			"MetricData.member.1.Unit":       []string{"Count"},
		}},
		{Action: "ListMetrics", Params: url.Values{
			"Namespace": []string{metricNamespace},
		}},
		{Action: "GetMetricStatistics", Params: url.Values{
			"Namespace":           []string{metricNamespace},
			"MetricName":          []string{metricName},
			"Statistics.member.1": []string{"Average"},
			"Period":              []string{"60"},
			"StartTime":           []string{time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)},
			"EndTime":             []string{time.Now().UTC().Format(time.RFC3339)},
		}},
		{Action: "DescribeAlarms", Params: url.Values{}},
	}

	fmt.Printf("Stackyard CloudWatch advanced client using %s\n", endpoint)
	for _, c := range calls {
		params := cloneValues(c.Params)
		params.Set("Action", c.Action)
		status, body, err := cloudWatchRequest(ctx, endpoint, region, creds, params)
		if err != nil {
			exitf("%s failed: %v", c.Action, err)
		}
		if status != http.StatusOK {
			exitf("%s returned HTTP %d: %s", c.Action, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s succeeded\n", c.Action)
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

func cloudWatchRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, params url.Values) (int, []byte, error) {
	if params.Get("Version") == "" {
		params.Set("Version", "2010-08-01")
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "monitoring", region, time.Now()); err != nil {
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
