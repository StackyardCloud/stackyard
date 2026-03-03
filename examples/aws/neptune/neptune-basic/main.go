package main

import (
	"bytes"
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
	clusterID := getenv("STACKYARD_NEPTUNE_CLUSTER", "neptune-basic-cluster")
	instanceID := getenv("STACKYARD_NEPTUNE_INSTANCE", clusterID+"-instance")
	snapshotID := getenv("STACKYARD_NEPTUNE_SNAPSHOT", clusterID+"-snapshot")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Neptune basic client using %s\n", endpoint)

	createStatus, createBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{clusterID},
		"Engine":              []string{"neptune"},
	})
	if err != nil {
		exitf("CreateDBCluster request failed: %v", err)
	}
	if err := expectSuccess("CreateDBCluster", createStatus, createBody); err != nil {
		exitf("%v", err)
	}
	logf("CreateDBCluster succeeded (%d)", createStatus)
	clusterARN := xmlTagValue(string(createBody), "DBClusterArn")
	if clusterARN == "" {
		clusterARN = "arn:aws:rds:us-east-1:123456789012:cluster:" + clusterID
	}

	createInstanceStatus, createInstanceBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"Engine":               []string{"neptune"},
	})
	if err != nil {
		exitf("CreateDBInstance request failed: %v", err)
	}
	if err := expectSuccess("CreateDBInstance", createInstanceStatus, createInstanceBody); err != nil {
		exitf("%v", err)
	}
	logf("CreateDBInstance succeeded (%d)", createInstanceStatus)

	postStatus, postBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", url.Values{
		"Action":              []string{"DescribeDBClusters"},
		"DBClusterIdentifier": []string{clusterID},
	})
	if err != nil {
		exitf("DescribeDBClusters (POST) request failed: %v", err)
	}
	if err := expectSuccess("DescribeDBClusters (POST)", postStatus, postBody); err != nil {
		exitf("%v", err)
	}
	logf("DescribeDBClusters (POST) succeeded (%d)", postStatus)

	getStatus, getBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodGet, "/neptune", url.Values{
		"Action": []string{"DescribeDBEngineVersions"},
	})
	if err != nil {
		exitf("DescribeDBEngineVersions (GET) request failed: %v", err)
	}
	if err := expectSuccess("DescribeDBEngineVersions (GET)", getStatus, getBody); err != nil {
		exitf("%v", err)
	}
	logf("DescribeDBEngineVersions (GET) succeeded (%d)", getStatus)

	snapshotStatus, snapshotBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", url.Values{
		"Action":                      []string{"CreateDBClusterSnapshot"},
		"DBClusterSnapshotIdentifier": []string{snapshotID},
		"DBClusterIdentifier":         []string{clusterID},
	})
	if err != nil {
		exitf("CreateDBClusterSnapshot request failed: %v", err)
	}
	if err := expectSuccess("CreateDBClusterSnapshot", snapshotStatus, snapshotBody); err != nil {
		exitf("%v", err)
	}
	logf("CreateDBClusterSnapshot succeeded (%d)", snapshotStatus)

	tagStatus, tagBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", url.Values{
		"Action":           []string{"AddTagsToResource"},
		"ResourceName":     []string{clusterARN},
		"Tags.Tag.1.Key":   []string{"env"},
		"Tags.Tag.1.Value": []string{"basic"},
	})
	if err != nil {
		exitf("AddTagsToResource request failed: %v", err)
	}
	if err := expectSuccess("AddTagsToResource", tagStatus, tagBody); err != nil {
		exitf("%v", err)
	}
	logf("AddTagsToResource succeeded (%d)", tagStatus)

	deleteInstanceStatus, deleteInstanceBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", url.Values{
		"Action":               []string{"DeleteDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"SkipFinalSnapshot":    []string{"true"},
	})
	if err != nil {
		exitf("DeleteDBInstance request failed: %v", err)
	}
	if err := expectSuccess("DeleteDBInstance", deleteInstanceStatus, deleteInstanceBody); err != nil {
		exitf("%v", err)
	}
	logf("DeleteDBInstance succeeded (%d)", deleteInstanceStatus)

	deleteStatus, deleteBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", url.Values{
		"Action":              []string{"DeleteDBCluster"},
		"DBClusterIdentifier": []string{clusterID},
		"SkipFinalSnapshot":   []string{"true"},
	})
	if err != nil {
		exitf("DeleteDBCluster request failed: %v", err)
	}
	if err := expectSuccess("DeleteDBCluster", deleteStatus, deleteBody); err != nil {
		exitf("%v", err)
	}
	logf("DeleteDBCluster succeeded (%d)", deleteStatus)

	fmt.Println("Done.")
}

func neptuneRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	params url.Values,
) (int, []byte, error) {
	values := cloneValues(params)
	if values.Get("Version") == "" {
		values.Set("Version", "2014-10-31")
	}

	var body []byte
	requestURL := strings.TrimRight(endpoint, "/") + path
	if method == http.MethodGet {
		encoded := values.Encode()
		if encoded != "" {
			requestURL += "?" + encoded
		}
	} else if method == http.MethodPost {
		body = []byte(values.Encode())
	} else {
		return 0, nil, fmt.Errorf("unsupported method: %s", method)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	}
	req.Header.Set("User-Agent", "aws-cli/2.0 md/command#neptune.example")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "rds", region, time.Now()); err != nil {
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

func cloneValues(in url.Values) url.Values {
	out := make(url.Values, len(in))
	for key, vals := range in {
		out[key] = append([]string(nil), vals...)
	}
	return out
}

func expectSuccess(action string, status int, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	if strings.Contains(string(body), "NotImplemented") {
		return fmt.Errorf("expected %s to be implemented, got: %s", action, strings.TrimSpace(string(body)))
	}
	return nil
}

func xmlTagValue(payload, tag string) string {
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
