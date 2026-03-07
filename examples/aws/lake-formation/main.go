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

	fmt.Printf("Stackyard AWS Lake Formation advanced client using %s\n", endpoint)
	if err := waitForStackyard(ctx, endpoint, 30*time.Second); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	resourceARN := "arn:aws:s3:::example-lakeformation-data"

	mustCall(ctx, endpoint, region, creds, "RegisterResource", map[string]any{
		"ResourceArn": resourceARN,
	})
	mustCall(ctx, endpoint, region, creds, "DescribeResource", map[string]any{
		"ResourceArn": resourceARN,
	})
	mustCall(ctx, endpoint, region, creds, "CreateLFTag", map[string]any{
		"TagKey":    "env",
		"TagValues": []string{"stage", "prod"},
	})
	mustCall(ctx, endpoint, region, creds, "AddLFTagsToResource", map[string]any{
		"Resource": map[string]any{
			"DataLocation": map[string]any{"ResourceArn": resourceARN},
		},
		"LFTags": []map[string]any{
			{"TagKey": "env", "TagValues": []string{"stage"}},
		},
	})
	mustCall(ctx, endpoint, region, creds, "GetResourceLFTags", map[string]any{
		"Resource": map[string]any{
			"DataLocation": map[string]any{"ResourceArn": resourceARN},
		},
	})
	mustCall(ctx, endpoint, region, creds, "GrantPermissions", map[string]any{
		"Principal": map[string]any{
			"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stage-admin",
		},
		"Resource": map[string]any{
			"DataLocation": map[string]any{"ResourceArn": resourceARN},
		},
		"Permissions": []string{"DATA_LOCATION_ACCESS"},
	})
	mustCall(ctx, endpoint, region, creds, "ListPermissions", map[string]any{})
	mustCall(ctx, endpoint, region, creds, "CreateDataCellsFilter", map[string]any{
		"TableCatalogId": "123456789012",
		"DatabaseName":   "stage_db",
		"TableName":      "stage_table",
		"Name":           "stage_filter",
		"RowFilter": map[string]any{
			"FilterExpression": "TRUE",
		},
	})
	mustCall(ctx, endpoint, region, creds, "GetDataCellsFilter", map[string]any{
		"DatabaseName": "stage_db",
		"TableName":    "stage_table",
		"Name":         "stage_filter",
	})

	tx := mustCall(ctx, endpoint, region, creds, "StartTransaction", map[string]any{
		"ClientRequestToken": "example-lf-transaction-token-000001",
	})
	txID := payloadString(tx, "TransactionId")
	if txID == "" {
		exitf("StartTransaction did not return TransactionId")
	}
	mustCall(ctx, endpoint, region, creds, "DescribeTransaction", map[string]any{
		"TransactionId": txID,
	})
	mustCall(ctx, endpoint, region, creds, "ExtendTransaction", map[string]any{
		"TransactionId": txID,
	})

	query := mustCall(ctx, endpoint, region, creds, "StartQueryPlanning", map[string]any{})
	queryID := payloadString(query, "QueryId")
	if queryID == "" {
		exitf("StartQueryPlanning did not return QueryId")
	}
	mustCall(ctx, endpoint, region, creds, "GetQueryState", map[string]any{"QueryId": queryID})
	mustCall(ctx, endpoint, region, creds, "GetQueryStatistics", map[string]any{"QueryId": queryID})
	mustCall(ctx, endpoint, region, creds, "GetWorkUnits", map[string]any{"QueryId": queryID, "PageSize": 10})
	mustCall(ctx, endpoint, region, creds, "GetWorkUnitResults", map[string]any{"QueryId": queryID, "WorkUnitId": 0})
	mustCall(ctx, endpoint, region, creds, "GetTemporaryDataLocationCredentials", map[string]any{"ResourceArn": resourceARN})
	mustCall(ctx, endpoint, region, creds, "GetTemporaryGluePartitionCredentials", map[string]any{"DatabaseName": "stage_db", "TableName": "stage_table"})
	mustCall(ctx, endpoint, region, creds, "GetTemporaryGlueTableCredentials", map[string]any{"DatabaseName": "stage_db", "TableName": "stage_table"})

	mustCall(ctx, endpoint, region, creds, "UpdateTableObjects", map[string]any{
		"DatabaseName": "stage_db",
		"TableName":    "stage_table",
		"WriteOperations": []map[string]any{
			{"AddObject": map[string]any{"Uri": "s3://example-lakeformation-data/stage_db/stage_table/object-000001.parquet"}},
		},
	})
	mustCall(ctx, endpoint, region, creds, "GetTableObjects", map[string]any{
		"DatabaseName": "stage_db",
		"TableName":    "stage_table",
	})

	mustCall(ctx, endpoint, region, creds, "UpdateTableStorageOptimizer", map[string]any{
		"DatabaseName":         "stage_db",
		"TableName":            "stage_table",
		"StorageOptimizerType": "COMPACTION",
		"Config": map[string]any{
			"RoleArn": "arn:aws:iam::123456789012:role/stage-admin",
		},
	})
	mustCall(ctx, endpoint, region, creds, "ListTableStorageOptimizers", map[string]any{})
	mustCall(ctx, endpoint, region, creds, "CreateLakeFormationOptIn", map[string]any{
		"Principal": map[string]any{
			"DataLakePrincipalIdentifier": "arn:aws:iam::123456789012:role/stage-admin",
		},
		"Resource": map[string]any{
			"Catalog": map[string]any{},
		},
	})
	mustCall(ctx, endpoint, region, creds, "ListLakeFormationOptIns", map[string]any{})

	mustCall(ctx, endpoint, region, creds, "CreateLakeFormationIdentityCenterConfiguration", map[string]any{
		"CatalogId":   "123456789012",
		"InstanceArn": "arn:aws:sso:::instance/ssoins-1234567890abcdef",
	})
	mustCall(ctx, endpoint, region, creds, "DescribeLakeFormationIdentityCenterConfiguration", map[string]any{})
	mustCall(ctx, endpoint, region, creds, "AssumeDecoratedRoleWithSAML", map[string]any{})
	mustCall(ctx, endpoint, region, creds, "DeleteObjectsOnCancel", map[string]any{
		"TransactionId": txID,
	})
	mustCall(ctx, endpoint, region, creds, "CommitTransaction", map[string]any{
		"TransactionId": txID,
	})
	mustCall(ctx, endpoint, region, creds, "DeleteLakeFormationIdentityCenterConfiguration", map[string]any{})
	mustCall(ctx, endpoint, region, creds, "DeleteLakeFormationOptIn", map[string]any{})
	mustCall(ctx, endpoint, region, creds, "DeleteDataCellsFilter", map[string]any{
		"DatabaseName": "stage_db",
		"TableName":    "stage_table",
		"Name":         "stage_filter",
	})
	mustCall(ctx, endpoint, region, creds, "DeleteLFTag", map[string]any{
		"TagKey": "env",
	})
	mustCall(ctx, endpoint, region, creds, "DeregisterResource", map[string]any{
		"ResourceArn": resourceARN,
	})

	fmt.Println("Done.")
}

func mustCall(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) map[string]any {
	status, body, err := lakeFormationRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", action, status, strings.TrimSpace(string(body)))
	}

	logf("%s returned %d", action, status)

	parsed := map[string]any{}
	if len(bytes.TrimSpace(body)) == 0 {
		return parsed
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return map[string]any{}
	}
	return parsed
}

func lakeFormationRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/"+action,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "lakeformation", region, time.Now()); err != nil {
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

func payloadString(payload map[string]any, key string) string {
	v, ok := payload[key]
	if !ok {
		return ""
	}
	text, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func waitForStackyard(ctx context.Context, endpoint string, timeout time.Duration) error {
	healthURL := strings.TrimRight(endpoint, "/") + "/_stackyard/health"
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, http.NoBody)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("health returned status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for health endpoint")
	}
	return fmt.Errorf("%s: %w", healthURL, lastErr)
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
