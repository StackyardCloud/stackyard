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
	dbInstanceID := getenv("STACKYARD_DB_INSTANCE", "rds-basic-instance")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard RDS basic client using %s\n", endpoint)

	if err := createDBInstance(ctx, endpoint, region, creds, dbInstanceID); err != nil {
		exitf("create db instance: %v", err)
	}
	logf("created db instance: %s", dbInstanceID)

	info, err := describeDBInstance(ctx, endpoint, region, creds, dbInstanceID)
	if err != nil {
		exitf("describe db instance: %v", err)
	}
	fmt.Printf("Describe DB instance %s -> status=%s engine=%s endpoint=%s\n", info.ID, info.Status, info.Engine, info.Endpoint)

	if err := deleteDBInstance(ctx, endpoint, region, creds, dbInstanceID); err != nil {
		exitf("delete db instance: %v", err)
	}
	logf("deleted db instance: %s", dbInstanceID)

	fmt.Println("Done.")
}

func rdsRequest(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, params url.Values) (int, []byte, error) {
	if params.Get("Version") == "" {
		params.Set("Version", "2014-10-31")
	}
	body := []byte(params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", strings.NewReader(string(body)))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "rds", region, time.Now()); err != nil {
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

func createDBInstance(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, dbInstanceID string) error {
	status, body, err := rdsRequest(ctx, endpoint, region, creds, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{dbInstanceID},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
		"PubliclyAccessible":   []string{"true"},
	})
	if err != nil {
		return err
	}
	return expectStatus("CreateDBInstance", status, body)
}

type dbInstanceInfo struct {
	ID       string
	Status   string
	Engine   string
	Endpoint string
}

func describeDBInstance(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, dbInstanceID string) (dbInstanceInfo, error) {
	status, body, err := rdsRequest(ctx, endpoint, region, creds, url.Values{
		"Action":               []string{"DescribeDBInstances"},
		"DBInstanceIdentifier": []string{dbInstanceID},
	})
	if err != nil {
		return dbInstanceInfo{}, err
	}
	if err := expectStatus("DescribeDBInstances", status, body); err != nil {
		return dbInstanceInfo{}, err
	}

	payload := string(body)
	id := findTag(payload, "DBInstanceIdentifier")
	if id == "" {
		return dbInstanceInfo{}, fmt.Errorf("db instance not found in response")
	}
	return dbInstanceInfo{
		ID:       id,
		Status:   findTag(payload, "DBInstanceStatus"),
		Engine:   findTag(payload, "Engine"),
		Endpoint: findTag(payload, "Address"),
	}, nil
}

func deleteDBInstance(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, dbInstanceID string) error {
	status, body, err := rdsRequest(ctx, endpoint, region, creds, url.Values{
		"Action":               []string{"DeleteDBInstance"},
		"DBInstanceIdentifier": []string{dbInstanceID},
		"SkipFinalSnapshot":    []string{"true"},
	})
	if err != nil {
		return err
	}
	return expectStatus("DeleteDBInstance", status, body)
}

func expectStatus(action string, status int, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	return nil
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
