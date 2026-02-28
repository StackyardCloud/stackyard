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
	hsmType := getenv("STACKYARD_HSM_TYPE", "hsm1.medium")
	subnetID := getenv("STACKYARD_SUBNET_ID", "subnet-12345678")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard CloudHSM advanced client using %s\n", endpoint)

	createStatus, createBody, err := cloudhsmRequest(ctx, endpoint, region, creds, "CreateCluster", map[string]any{
		"HsmType":   hsmType,
		"SubnetIds": []string{subnetID},
		"BackupRetentionPolicy": map[string]any{
			"Type":  "DAYS",
			"Value": "30",
		},
		"TagList": []map[string]string{
			{"Key": "env", "Value": "dev"},
		},
	})
	if err != nil {
		exitf("create cluster: %v", err)
	}
	if err := expectHTTPStatus("CreateCluster", createStatus, http.StatusOK, createBody); err != nil {
		exitf("create cluster: %v", err)
	}
	clusterID, err := extractString(createBody, "Cluster", "ClusterId")
	if err != nil {
		exitf("parse create cluster response: %v", err)
	}
	clusterARN := fmt.Sprintf("arn:aws:cloudhsm:us-east-1:123456789012:cluster/%s", clusterID)
	logf("created cluster: %s", clusterID)

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "InitializeCluster", map[string]any{
		"ClusterId":   clusterID,
		"SignedCert":  "signed-cert",
		"TrustAnchor": "trust-anchor",
	}); err != nil {
		exitf("initialize cluster: %v", err)
	} else if err := expectHTTPStatus("InitializeCluster", status, http.StatusOK, body); err != nil {
		exitf("initialize cluster: %v", err)
	}
	logf("InitializeCluster succeeded")

	createHsmStatus, createHsmBody, err := cloudhsmRequest(ctx, endpoint, region, creds, "CreateHsm", map[string]any{
		"ClusterId":        clusterID,
		"AvailabilityZone": "us-east-1a",
	})
	if err != nil {
		exitf("create hsm: %v", err)
	}
	if err := expectHTTPStatus("CreateHsm", createHsmStatus, http.StatusOK, createHsmBody); err != nil {
		exitf("create hsm: %v", err)
	}
	hsmID, err := extractString(createHsmBody, "Hsm", "HsmId")
	if err != nil {
		exitf("parse create hsm response: %v", err)
	}
	logf("created hsm: %s", hsmID)

	describeBackupsStatus, describeBackupsBody, err := cloudhsmRequest(ctx, endpoint, region, creds, "DescribeBackups", map[string]any{
		"Filters": map[string]any{
			"clusterIds": []string{clusterID},
		},
		"MaxResults": 10,
	})
	if err != nil {
		exitf("describe backups: %v", err)
	}
	if err := expectHTTPStatus("DescribeBackups", describeBackupsStatus, http.StatusOK, describeBackupsBody); err != nil {
		exitf("describe backups: %v", err)
	}
	backupID, err := extractFirstBackupID(describeBackupsBody)
	if err != nil {
		exitf("extract backup id: %v", err)
	}
	logf("discovered backup: %s", backupID)

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "CopyBackupToRegion", map[string]any{
		"DestinationRegion": "us-west-2",
		"BackupId":          backupID,
		"TagList": []map[string]string{
			{"Key": "copied", "Value": "true"},
		},
	}); err != nil {
		exitf("copy backup to region: %v", err)
	} else if err := expectHTTPStatus("CopyBackupToRegion", status, http.StatusOK, body); err != nil {
		exitf("copy backup to region: %v", err)
	}
	logf("CopyBackupToRegion succeeded")

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "ModifyBackupAttributes", map[string]any{
		"BackupId":     backupID,
		"NeverExpires": true,
	}); err != nil {
		exitf("modify backup attributes: %v", err)
	} else if err := expectHTTPStatus("ModifyBackupAttributes", status, http.StatusOK, body); err != nil {
		exitf("modify backup attributes: %v", err)
	}
	logf("ModifyBackupAttributes succeeded")

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "RestoreBackup", map[string]any{
		"BackupId": backupID,
	}); err != nil {
		exitf("restore backup: %v", err)
	} else if err := expectHTTPStatus("RestoreBackup", status, http.StatusOK, body); err != nil {
		exitf("restore backup: %v", err)
	}
	logf("RestoreBackup succeeded")

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"ResourceId": clusterID,
		"TagList": []map[string]string{
			{"Key": "team", "Value": "platform"},
		},
	}); err != nil {
		exitf("tag resource: %v", err)
	} else if err := expectHTTPStatus("TagResource", status, http.StatusOK, body); err != nil {
		exitf("tag resource: %v", err)
	}
	logf("TagResource succeeded")

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "ListTags", map[string]any{
		"ResourceId": clusterID,
		"MaxResults": 10,
	}); err != nil {
		exitf("list tags: %v", err)
	} else if err := expectHTTPStatus("ListTags", status, http.StatusOK, body); err != nil {
		exitf("list tags: %v", err)
	}
	logf("ListTags succeeded")

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "UntagResource", map[string]any{
		"ResourceId": clusterID,
		"TagKeyList": []string{"team"},
	}); err != nil {
		exitf("untag resource: %v", err)
	} else if err := expectHTTPStatus("UntagResource", status, http.StatusOK, body); err != nil {
		exitf("untag resource: %v", err)
	}
	logf("UntagResource succeeded")

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "PutResourcePolicy", map[string]any{
		"ResourceArn": clusterARN,
		"Policy":      "{}",
	}); err != nil {
		exitf("put resource policy: %v", err)
	} else if err := expectHTTPStatus("PutResourcePolicy", status, http.StatusOK, body); err != nil {
		exitf("put resource policy: %v", err)
	}
	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "GetResourcePolicy", map[string]any{
		"ResourceArn": clusterARN,
	}); err != nil {
		exitf("get resource policy: %v", err)
	} else if err := expectHTTPStatus("GetResourcePolicy", status, http.StatusOK, body); err != nil {
		exitf("get resource policy: %v", err)
	}
	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "DeleteResourcePolicy", map[string]any{
		"ResourceArn": clusterARN,
	}); err != nil {
		exitf("delete resource policy: %v", err)
	} else if err := expectHTTPStatus("DeleteResourcePolicy", status, http.StatusOK, body); err != nil {
		exitf("delete resource policy: %v", err)
	}
	logf("Resource policy operations succeeded")

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "DeleteHsm", map[string]any{
		"ClusterId": clusterID,
		"HsmId":     hsmID,
	}); err != nil {
		exitf("delete hsm: %v", err)
	} else if err := expectHTTPStatus("DeleteHsm", status, http.StatusOK, body); err != nil {
		exitf("delete hsm: %v", err)
	}

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "DeleteBackup", map[string]any{
		"BackupId": backupID,
	}); err != nil {
		exitf("delete backup: %v", err)
	} else if err := expectHTTPStatus("DeleteBackup", status, http.StatusOK, body); err != nil {
		exitf("delete backup: %v", err)
	}

	if status, body, err := cloudhsmRequest(ctx, endpoint, region, creds, "DeleteCluster", map[string]any{
		"ClusterId": clusterID,
	}); err != nil {
		exitf("delete cluster: %v", err)
	} else if err := expectHTTPStatus("DeleteCluster", status, http.StatusOK, body); err != nil {
		exitf("delete cluster: %v", err)
	}
	logf("DeleteCluster succeeded")

	fmt.Println("Done.")
}

func cloudhsmRequest(
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

	requestURL := strings.TrimRight(endpoint, "/") + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "BaldrApiService."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "cloudhsm", region, time.Now()); err != nil {
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

func expectHTTPStatus(action string, status, expected int, body []byte) error {
	if status != expected {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, expected, status, strings.TrimSpace(string(body)))
	}
	return nil
}

func extractString(body []byte, keys ...string) (string, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	current := payload
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("key path %v: non-object segment", keys)
		}
		next, ok := object[key]
		if !ok {
			return "", fmt.Errorf("missing key %s", key)
		}
		current = next
	}
	value, ok := current.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("key path %v is empty or non-string", keys)
	}
	return value, nil
}

func extractFirstBackupID(body []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	items, ok := payload["Backups"].([]any)
	if !ok || len(items) == 0 {
		return "", fmt.Errorf("backups missing")
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("first backup is not an object")
	}
	backupID, ok := first["BackupId"].(string)
	if !ok || strings.TrimSpace(backupID) == "" {
		return "", fmt.Errorf("backup id missing")
	}
	return backupID, nil
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
