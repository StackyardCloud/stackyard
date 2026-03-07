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
	secretName := getenv("STACKYARD_SECRET_NAME", "stackyard-secret")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Secrets Manager advanced client using %s\n", endpoint)

	createStatus, createBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "CreateSecret", map[string]any{
		"Name":               secretName,
		"ClientRequestToken": "secretsmanager-create",
		"Description":        "advanced flow secret",
		"OwningService":      "stackyard-example",
		"SecretString":       "{\"username\":\"stackyard\",\"password\":\"initial\"}",
		"Tags":               []map[string]string{{"Key": "env", "Value": "dev"}},
	})
	if err != nil {
		exitf("create secret: %v", err)
	}
	if err := expectHTTPStatus("CreateSecret", createStatus, http.StatusOK, createBody); err != nil {
		exitf("create secret: %v", err)
	}
	secretARN, err := extractString(createBody, "ARN")
	if err != nil {
		exitf("extract secret arn: %v", err)
	}
	logf("CreateSecret succeeded")

	updateStatus, updateBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "UpdateSecret", map[string]any{
		"SecretId":           secretARN,
		"ClientRequestToken": "secretsmanager-update",
		"Description":        "advanced flow secret updated",
		"SecretString":       "{\"username\":\"stackyard\",\"password\":\"updated\"}",
	})
	if err != nil {
		exitf("update secret: %v", err)
	}
	if err := expectHTTPStatus("UpdateSecret", updateStatus, http.StatusOK, updateBody); err != nil {
		exitf("update secret: %v", err)
	}
	logf("UpdateSecret succeeded")

	putStatus, putBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "PutSecretValue", map[string]any{
		"SecretId":           secretARN,
		"ClientRequestToken": "secretsmanager-put",
		"SecretString":       "{\"username\":\"stackyard\",\"password\":\"put\"}",
		"VersionStages":      []string{"AWSCURRENT"},
	})
	if err != nil {
		exitf("put secret value: %v", err)
	}
	if err := expectHTTPStatus("PutSecretValue", putStatus, http.StatusOK, putBody); err != nil {
		exitf("put secret value: %v", err)
	}
	putVersionID, err := extractString(putBody, "VersionId")
	if err != nil {
		exitf("extract put version id: %v", err)
	}
	logf("PutSecretValue succeeded")

	getStatus, getBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "GetSecretValue", map[string]any{
		"SecretId": secretARN,
	})
	if err != nil {
		exitf("get secret value: %v", err)
	}
	if err := expectHTTPStatus("GetSecretValue", getStatus, http.StatusOK, getBody); err != nil {
		exitf("get secret value: %v", err)
	}
	logf("GetSecretValue succeeded")

	listFilteredStatus, listFilteredBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "ListSecrets", map[string]any{
		"Filters": []map[string]any{
			{"Key": "owning-service", "Values": []string{"stackyard-example"}},
		},
	})
	if err != nil {
		exitf("list secrets with owning-service filter: %v", err)
	}
	if err := expectHTTPStatus("ListSecrets(filtered)", listFilteredStatus, http.StatusOK, listFilteredBody); err != nil {
		exitf("list secrets with owning-service filter: %v", err)
	}
	logf("ListSecrets owning-service filter succeeded")

	listVersionsStatus, listVersionsBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "ListSecretVersionIds", map[string]any{
		"SecretId":   secretARN,
		"MaxResults": 10,
	})
	if err != nil {
		exitf("list secret version ids: %v", err)
	}
	if err := expectHTTPStatus("ListSecretVersionIds", listVersionsStatus, http.StatusOK, listVersionsBody); err != nil {
		exitf("list secret version ids: %v", err)
	}
	logf("ListSecretVersionIds succeeded")

	updateStageStatus, updateStageBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "UpdateSecretVersionStage", map[string]any{
		"SecretId":        secretARN,
		"VersionStage":    "AWSPREVIOUS",
		"MoveToVersionId": putVersionID,
	})
	if err != nil {
		exitf("update secret version stage: %v", err)
	}
	if err := expectHTTPStatus("UpdateSecretVersionStage", updateStageStatus, http.StatusOK, updateStageBody); err != nil {
		exitf("update secret version stage: %v", err)
	}
	logf("UpdateSecretVersionStage succeeded")

	batchStatus, batchBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "BatchGetSecretValue", map[string]any{
		"SecretIdList": []string{secretARN, "missing-secret"},
		"MaxResults":   10,
	})
	if err != nil {
		exitf("batch get secret value: %v", err)
	}
	if err := expectHTTPStatus("BatchGetSecretValue", batchStatus, http.StatusOK, batchBody); err != nil {
		exitf("batch get secret value: %v", err)
	}
	logf("BatchGetSecretValue succeeded")

	passwordStatus, passwordBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "GetRandomPassword", map[string]any{
		"PasswordLength":          24,
		"ExcludePunctuation":      true,
		"RequireEachIncludedType": true,
	})
	if err != nil {
		exitf("get random password: %v", err)
	}
	if err := expectHTTPStatus("GetRandomPassword", passwordStatus, http.StatusOK, passwordBody); err != nil {
		exitf("get random password: %v", err)
	}
	logf("GetRandomPassword succeeded")

	deleteStatus, deleteBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "DeleteSecret", map[string]any{
		"SecretId":             secretARN,
		"RecoveryWindowInDays": 7,
	})
	if err != nil {
		exitf("delete secret: %v", err)
	}
	if err := expectHTTPStatus("DeleteSecret", deleteStatus, http.StatusOK, deleteBody); err != nil {
		exitf("delete secret: %v", err)
	}
	logf("DeleteSecret succeeded")

	restoreStatus, restoreBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "RestoreSecret", map[string]any{
		"SecretId": secretARN,
	})
	if err != nil {
		exitf("restore secret: %v", err)
	}
	if err := expectHTTPStatus("RestoreSecret", restoreStatus, http.StatusOK, restoreBody); err != nil {
		exitf("restore secret: %v", err)
	}
	logf("RestoreSecret succeeded")

	rotateStatus, rotateBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "RotateSecret", map[string]any{
		"SecretId":           secretARN,
		"ClientRequestToken": "secretsmanager-rotate",
		"RotationLambdaARN":  "arn:aws:lambda:us-east-1:123456789012:function:stackyard-secrets-rotate",
		"RotateImmediately":  true,
		"RotationRules": map[string]any{
			"AutomaticallyAfterDays": 30,
		},
	})
	if err != nil {
		exitf("rotate secret: %v", err)
	}
	if err := expectHTTPStatus("RotateSecret", rotateStatus, http.StatusOK, rotateBody); err != nil {
		exitf("rotate secret: %v", err)
	}
	logf("RotateSecret succeeded")

	replicateStatus, replicateBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "ReplicateSecretToRegions", map[string]any{
		"SecretId": secretARN,
		"AddReplicaRegions": []map[string]any{
			{"Region": "us-west-2", "KmsKeyId": "alias/stage3"},
			{"Region": "us-west-1"},
		},
	})
	if err != nil {
		exitf("replicate secret: %v", err)
	}
	if err := expectHTTPStatus("ReplicateSecretToRegions", replicateStatus, http.StatusOK, replicateBody); err != nil {
		exitf("replicate secret: %v", err)
	}
	logf("ReplicateSecretToRegions succeeded")

	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"arn:aws:iam::123456789012:root"},"Action":"secretsmanager:GetSecretValue","Resource":"*"}]}`
	putPolicyStatus, putPolicyBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "PutResourcePolicy", map[string]any{
		"SecretId":          secretARN,
		"ResourcePolicy":    policy,
		"BlockPublicPolicy": true,
	})
	if err != nil {
		exitf("put resource policy: %v", err)
	}
	if err := expectHTTPStatus("PutResourcePolicy", putPolicyStatus, http.StatusOK, putPolicyBody); err != nil {
		exitf("put resource policy: %v", err)
	}
	logf("PutResourcePolicy succeeded")

	getPolicyStatus, getPolicyBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "GetResourcePolicy", map[string]any{
		"SecretId": secretARN,
	})
	if err != nil {
		exitf("get resource policy: %v", err)
	}
	if err := expectHTTPStatus("GetResourcePolicy", getPolicyStatus, http.StatusOK, getPolicyBody); err != nil {
		exitf("get resource policy: %v", err)
	}
	logf("GetResourcePolicy succeeded")

	validatePolicyStatus, validatePolicyBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "ValidateResourcePolicy", map[string]any{
		"SecretId":          secretARN,
		"BlockPublicPolicy": true,
	})
	if err != nil {
		exitf("validate resource policy: %v", err)
	}
	if err := expectHTTPStatus("ValidateResourcePolicy", validatePolicyStatus, http.StatusOK, validatePolicyBody); err != nil {
		exitf("validate resource policy: %v", err)
	}
	logf("ValidateResourcePolicy succeeded")

	tagStatus, tagBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"SecretId": secretARN,
		"Tags": []map[string]string{
			{"Key": "owner", "Value": "platform"},
		},
	})
	if err != nil {
		exitf("tag resource: %v", err)
	}
	if err := expectHTTPStatus("TagResource", tagStatus, http.StatusOK, tagBody); err != nil {
		exitf("tag resource: %v", err)
	}
	logf("TagResource succeeded")

	untagStatus, untagBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "UntagResource", map[string]any{
		"SecretId": secretARN,
		"TagKeys":  []string{"owner"},
	})
	if err != nil {
		exitf("untag resource: %v", err)
	}
	if err := expectHTTPStatus("UntagResource", untagStatus, http.StatusOK, untagBody); err != nil {
		exitf("untag resource: %v", err)
	}
	logf("UntagResource succeeded")

	removeReplicaStatus, removeReplicaBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "RemoveRegionsFromReplication", map[string]any{
		"SecretId":             secretARN,
		"RemoveReplicaRegions": []string{"us-west-2"},
	})
	if err != nil {
		exitf("remove regions from replication: %v", err)
	}
	if err := expectHTTPStatus("RemoveRegionsFromReplication", removeReplicaStatus, http.StatusOK, removeReplicaBody); err != nil {
		exitf("remove regions from replication: %v", err)
	}
	logf("RemoveRegionsFromReplication succeeded")

	stopReplicaStatus, stopReplicaBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "StopReplicationToReplica", map[string]any{
		"SecretId":      secretARN,
		"ReplicaRegion": "us-west-1",
	})
	if err != nil {
		exitf("stop replication to replica: %v", err)
	}
	if err := expectHTTPStatus("StopReplicationToReplica", stopReplicaStatus, http.StatusOK, stopReplicaBody); err != nil {
		exitf("stop replication to replica: %v", err)
	}
	logf("StopReplicationToReplica succeeded")

	cancelRotateStatus, cancelRotateBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "CancelRotateSecret", map[string]any{
		"SecretId": secretARN,
	})
	if err != nil {
		exitf("cancel rotate secret: %v", err)
	}
	if err := expectHTTPStatus("CancelRotateSecret", cancelRotateStatus, http.StatusOK, cancelRotateBody); err != nil {
		exitf("cancel rotate secret: %v", err)
	}
	logf("CancelRotateSecret succeeded")

	deletePolicyStatus, deletePolicyBody, err := secretsManagerRequest(ctx, endpoint, region, creds, "DeleteResourcePolicy", map[string]any{
		"SecretId": secretARN,
	})
	if err != nil {
		exitf("delete resource policy: %v", err)
	}
	if err := expectHTTPStatus("DeleteResourcePolicy", deletePolicyStatus, http.StatusOK, deletePolicyBody); err != nil {
		exitf("delete resource policy: %v", err)
	}
	logf("DeleteResourcePolicy succeeded")

	fmt.Println("Done.")
}

func secretsManagerRequest(
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
	req.Header.Set("X-Amz-Target", "secretsmanager."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "secretsmanager", region, time.Now()); err != nil {
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

func extractString(body []byte, key string) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	raw, ok := payload[key]
	if !ok {
		return "", fmt.Errorf("missing key %s", key)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("invalid key %s", key)
	}
	return value, nil
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
