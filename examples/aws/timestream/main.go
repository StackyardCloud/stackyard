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

	fmt.Printf("Stackyard Timestream for InfluxDB advanced client using %s\n", endpoint)

	createStatus, createBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "CreateDbCluster", map[string]any{
		"name":                "stackyard-cluster",
		"dbInstanceType":      "db.influx.medium",
		"vpcSubnetIds":        []string{"subnet-12345678", "subnet-87654321"},
		"vpcSecurityGroupIds": []string{"sg-12345678"},
		"tags": map[string]string{
			"env": "dev",
		},
	})
	if err != nil {
		exitf("CreateDbCluster request failed: %v", err)
	}
	createPayload, err := expectSuccess("CreateDbCluster", createStatus, createBody)
	if err != nil {
		exitf("CreateDbCluster response validation failed: %v", err)
	}
	clusterID, _ := createPayload["dbClusterId"].(string)
	if strings.TrimSpace(clusterID) == "" {
		exitf("CreateDbCluster response missing dbClusterId: %s", strings.TrimSpace(string(createBody)))
	}
	logf("CreateDbCluster returned implemented stage-2 response (%d)", createStatus)

	getStatus, getBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "GetDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	if err != nil {
		exitf("GetDbCluster request failed: %v", err)
	}
	getPayload, err := expectSuccess("GetDbCluster", getStatus, getBody)
	if err != nil {
		exitf("GetDbCluster response validation failed: %v", err)
	}
	clusterArn, _ := getPayload["arn"].(string)
	logf("GetDbCluster returned implemented stage-1 response (%d)", getStatus)

	createPGStatus, createPGBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "CreateDbParameterGroup", map[string]any{
		"name":        "stackyard-params",
		"description": "advanced example parameter group",
		"parameters": map[string]any{
			"influxDBv2": map[string]any{
				"logLevel": "info",
			},
		},
		"tags": map[string]string{
			"env": "dev",
		},
	})
	if err != nil {
		exitf("CreateDbParameterGroup request failed: %v", err)
	}
	createPGPayload, err := expectSuccess("CreateDbParameterGroup", createPGStatus, createPGBody)
	if err != nil {
		exitf("CreateDbParameterGroup response validation failed: %v", err)
	}
	pgID, _ := createPGPayload["id"].(string)
	if strings.TrimSpace(pgID) == "" {
		exitf("CreateDbParameterGroup response missing id: %s", strings.TrimSpace(string(createPGBody)))
	}
	logf("CreateDbParameterGroup returned implemented stage-4 response (%d)", createPGStatus)

	updateStatus, updateBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "UpdateDbCluster", map[string]any{
		"dbClusterId":    clusterID,
		"dbInstanceType": "db.influx.large",
	})
	if err != nil {
		exitf("UpdateDbCluster request failed: %v", err)
	}
	if _, err := expectSuccess("UpdateDbCluster", updateStatus, updateBody); err != nil {
		exitf("UpdateDbCluster response validation failed: %v", err)
	}
	logf("UpdateDbCluster returned implemented stage-2 response (%d)", updateStatus)

	rebootStatus, rebootBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "RebootDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	if err != nil {
		exitf("RebootDbCluster request failed: %v", err)
	}
	if _, err := expectSuccess("RebootDbCluster", rebootStatus, rebootBody); err != nil {
		exitf("RebootDbCluster response validation failed: %v", err)
	}
	logf("RebootDbCluster returned implemented stage-2 response (%d)", rebootStatus)

	listClusterInstancesStatus, listClusterInstancesBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "ListDbInstancesForCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	if err != nil {
		exitf("ListDbInstancesForCluster request failed: %v", err)
	}
	if _, err := expectSuccess("ListDbInstancesForCluster", listClusterInstancesStatus, listClusterInstancesBody); err != nil {
		exitf("ListDbInstancesForCluster response validation failed: %v", err)
	}
	logf("ListDbInstancesForCluster returned implemented stage-1 response (%d)", listClusterInstancesStatus)

	listTagsStatus, listTagsBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "ListTagsForResource", map[string]any{
		"resourceArn": clusterArn,
	})
	if err != nil {
		exitf("ListTagsForResource request failed: %v", err)
	}
	if _, err := expectSuccess("ListTagsForResource", listTagsStatus, listTagsBody); err != nil {
		exitf("ListTagsForResource response validation failed: %v", err)
	}
	logf("ListTagsForResource returned implemented stage-1 response (%d)", listTagsStatus)

	createInstanceStatus, createInstanceBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "CreateDbInstance", map[string]any{
		"name":                       "stackyard-instance",
		"password":                   "ChangeMe123!",
		"dbInstanceType":             "db.influx.medium",
		"vpcSubnetIds":               []string{"subnet-12345678"},
		"vpcSecurityGroupIds":        []string{"sg-12345678"},
		"allocatedStorage":           50,
		"dbParameterGroupIdentifier": pgID,
		"tags": map[string]string{
			"service": "timestream",
		},
	})
	if err != nil {
		exitf("CreateDbInstance request failed: %v", err)
	}
	createInstancePayload, err := expectSuccess("CreateDbInstance", createInstanceStatus, createInstanceBody)
	if err != nil {
		exitf("CreateDbInstance response validation failed: %v", err)
	}
	instanceID, _ := createInstancePayload["id"].(string)
	if strings.TrimSpace(instanceID) == "" {
		exitf("CreateDbInstance response missing id: %s", strings.TrimSpace(string(createInstanceBody)))
	}
	logf("CreateDbInstance returned implemented stage-3 response (%d)", createInstanceStatus)

	updateInstanceStatus, updateInstanceBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "UpdateDbInstance", map[string]any{
		"identifier":       instanceID,
		"dbInstanceType":   "db.influx.large",
		"allocatedStorage": 80,
	})
	if err != nil {
		exitf("UpdateDbInstance request failed: %v", err)
	}
	if _, err := expectSuccess("UpdateDbInstance", updateInstanceStatus, updateInstanceBody); err != nil {
		exitf("UpdateDbInstance response validation failed: %v", err)
	}
	logf("UpdateDbInstance returned implemented stage-3 response (%d)", updateInstanceStatus)

	rebootInstanceStatus, rebootInstanceBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "RebootDbInstance", map[string]any{
		"identifier": instanceID,
	})
	if err != nil {
		exitf("RebootDbInstance request failed: %v", err)
	}
	if _, err := expectSuccess("RebootDbInstance", rebootInstanceStatus, rebootInstanceBody); err != nil {
		exitf("RebootDbInstance response validation failed: %v", err)
	}
	logf("RebootDbInstance returned implemented stage-3 response (%d)", rebootInstanceStatus)

	tagResourceStatus, tagResourceBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"resourceArn": clusterArn,
		"tags": map[string]string{
			"owner": "stackyard",
		},
	})
	if err != nil {
		exitf("TagResource request failed: %v", err)
	}
	if _, err := expectSuccess("TagResource", tagResourceStatus, tagResourceBody); err != nil {
		exitf("TagResource response validation failed: %v", err)
	}
	logf("TagResource returned implemented stage-5 response (%d)", tagResourceStatus)

	untagResourceStatus, untagResourceBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "UntagResource", map[string]any{
		"resourceArn": clusterArn,
		"tagKeys":     []string{"owner"},
	})
	if err != nil {
		exitf("UntagResource request failed: %v", err)
	}
	if _, err := expectSuccess("UntagResource", untagResourceStatus, untagResourceBody); err != nil {
		exitf("UntagResource response validation failed: %v", err)
	}
	logf("UntagResource returned implemented stage-5 response (%d)", untagResourceStatus)

	invalidTokenStatus, invalidTokenBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "ListDbClusters", map[string]any{
		"nextToken": "bad-token",
	})
	if err != nil {
		exitf("ListDbClusters invalid token request failed: %v", err)
	}
	if err := expectValidation("ListDbClusters", invalidTokenStatus, invalidTokenBody); err != nil {
		exitf("ListDbClusters invalid token response validation failed: %v", err)
	}
	logf("ListDbClusters returned stage-6 validation response (%d)", invalidTokenStatus)

	deleteInstanceStatus, deleteInstanceBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "DeleteDbInstance", map[string]any{
		"identifier": instanceID,
	})
	if err != nil {
		exitf("DeleteDbInstance request failed: %v", err)
	}
	if _, err := expectSuccess("DeleteDbInstance", deleteInstanceStatus, deleteInstanceBody); err != nil {
		exitf("DeleteDbInstance response validation failed: %v", err)
	}
	logf("DeleteDbInstance returned implemented stage-3 response (%d)", deleteInstanceStatus)

	deleteStatus, deleteBody, err := callTimestreamInfluxDBAction(ctx, endpoint, region, creds, "DeleteDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	if err != nil {
		exitf("DeleteDbCluster request failed: %v", err)
	}
	if _, err := expectSuccess("DeleteDbCluster", deleteStatus, deleteBody); err != nil {
		exitf("DeleteDbCluster response validation failed: %v", err)
	}
	logf("DeleteDbCluster returned implemented stage-2 response (%d)", deleteStatus)

	fmt.Println("Done.")
}

func callTimestreamInfluxDBAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(encodedPayload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AmazonTimestreamInfluxDB."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(encodedPayload), "timestream-influxdb", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

func expectSuccess(action string, status int, body []byte) (map[string]any, error) {
	if status != http.StatusOK {
		return nil, fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("expected JSON response, got: %s", strings.TrimSpace(string(body)))
	}
	return payload, nil
}

func expectValidation(action string, status int, body []byte) error {
	if status != http.StatusBadRequest {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusBadRequest, status, strings.TrimSpace(string(body)))
	}

	text := strings.TrimSpace(string(body))
	if !strings.Contains(text, "ValidationException") {
		return fmt.Errorf("expected ValidationException marker in %s response: %s", action, text)
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
