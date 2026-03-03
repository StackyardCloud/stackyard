package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type requestSpec struct {
	name   string
	method string
	path   string
	body   []byte
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

	fmt.Printf("Stackyard Recovery Cluster advanced client using %s\n", endpoint)

	calls := []requestSpec{
		{name: "ListClusters", method: http.MethodGet, path: "/cluster"},
		{name: "ListControlPanels", method: http.MethodGet, path: "/controlpanels"},
		{name: "ListRoutingControls", method: http.MethodGet, path: "/controlpanel/stackyard-controlpanel/routingcontrols"},
		{name: "ListSafetyRules", method: http.MethodGet, path: "/controlpanel/stackyard-controlpanel/safetyrules"},
		{name: "ListAssociatedRoute53HealthChecks", method: http.MethodGet, path: "/routingcontrol/stackyard-routingcontrol/associatedRoute53HealthChecks"},
	}

	for _, call := range calls {
		status, body, err := recoveryClusterRequest(ctx, endpoint, region, creds, call.method, call.path, call.body)
		if err != nil {
			exitf("%s request failed: %v", call.name, err)
		}
		trimmed := strings.TrimSpace(string(body))
		if status <= 0 || status >= 500 {
			exitf("%s returned unexpected HTTP %d: %s", call.name, status, trimmed)
		}
		logf("%s returned %d", call.name, status)
	}

	fmt.Println("Done.")
}

func recoveryClusterRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
) (int, []byte, error) {
	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "route53-recovery-control-config", region, time.Now()); err != nil {
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
