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
	clusterID := getenv("STACKYARD_CLUSTER", "redshift-basic")

	ctx := context.Background()
	creds, err := loadCreds()
	if err != nil {
		exitf("load aws config: %v", err)
	}

	fmt.Printf("Stackyard Redshift basic client using %s\n", endpoint)

	if err := createCluster(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("create cluster: %v", err)
	}
	logf("created cluster: %s", clusterID)

	info, err := describeCluster(ctx, endpoint, region, creds, clusterID)
	if err != nil {
		exitf("describe cluster: %v", err)
	}
	fmt.Printf("Describe cluster %s -> status=%s nodeType=%s\n", info.ID, info.Status, info.NodeType)

	if err := deleteCluster(ctx, endpoint, region, creds, clusterID); err != nil {
		exitf("delete cluster: %v", err)
	}
	logf("deleted cluster: %s", clusterID)

	fmt.Println("Done.")
}

func loadCreds() (aws.CredentialsProvider, error) {
	return credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	), nil
}

func redshiftRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, params url.Values) ([]byte, error) {
	if params.Get("Version") == "" {
		params.Set("Version", "2012-12-01")
	}
	body := []byte(params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/redshift", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	credentials, err := creds.Retrieve(ctx)
	if err != nil {
		return nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentials, req, hashSHA256(body), "redshift", region, time.Now()); err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func createCluster(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{clusterID},
		"NodeType":           []string{"ra3.xlplus"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret1234"},
		"DBName":             []string{"dev"},
	})
	return err
}

type clusterInfo struct {
	ID       string
	Status   string
	NodeType string
}

func describeCluster(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) (clusterInfo, error) {
	body, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"DescribeClusters"},
		"ClusterIdentifier": []string{clusterID},
	})
	if err != nil {
		return clusterInfo{}, err
	}
	payload := string(body)
	id := findTag(payload, "ClusterIdentifier")
	if id == "" {
		return clusterInfo{}, fmt.Errorf("cluster not found in response")
	}
	return clusterInfo{
		ID:       id,
		Status:   findTag(payload, "ClusterStatus"),
		NodeType: findTag(payload, "NodeType"),
	}, nil
}

func deleteCluster(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, clusterID string) error {
	_, err := redshiftRequest(ctx, endpoint, region, creds, url.Values{
		"Action":            []string{"DeleteCluster"},
		"ClusterIdentifier": []string{clusterID},
	})
	return err
}

func findTag(payload, tag string) string {
	start := "<" + tag + ">"
	end := "</" + tag + ">"
	i := strings.Index(payload, start)
	if i == -1 {
		return ""
	}
	i += len(start)
	j := strings.Index(payload[i:], end)
	if j == -1 {
		return ""
	}
	return payload[i : i+j]
}

func hashSHA256(body []byte) string {
	sum := sha256.Sum256(body)
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

func init() {
	_ = time.Now()
}
