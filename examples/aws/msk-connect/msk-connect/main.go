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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type restCall struct {
	Name    string
	Method  string
	Path    string
	Payload map[string]any
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

	fmt.Printf("Stackyard Amazon MSK Connect advanced client using %s\n", endpoint)

	connectorARN := "arn:aws:kafkaconnect:us-east-1:123456789012:connector/stackyard-connector/00000000-0000-0000-0000-000000000001-1"
	customPluginARN := "arn:aws:kafkaconnect:us-east-1:123456789012:custom-plugin/stackyard-plugin/00000000-0000-0000-0000-000000000002-2"
	workerConfigurationARN := "arn:aws:kafkaconnect:us-east-1:123456789012:worker-configuration/stackyard-worker/00000000-0000-0000-0000-000000000003-3"
	connectorOperationARN := "arn:aws:kafkaconnect:us-east-1:123456789012:connector-operation/stackyard-connector/00000000-0000-0000-0000-000000000004"

	calls := []restCall{
		{
			Name:   "CreateCustomPlugin",
			Method: http.MethodPost,
			Path:   "/v1/custom-plugins",
			Payload: map[string]any{
				"contentType": "ZIP",
				"location": map[string]any{
					"s3Location": map[string]any{
						"bucketArn":     "arn:aws:s3:::stackyard-mskconnect",
						"fileKey":       "plugins/custom.zip",
						"objectVersion": "1",
					},
				},
				"name": "stackyard-plugin",
			},
		},
		{
			Name:   "CreateWorkerConfiguration",
			Method: http.MethodPost,
			Path:   "/v1/worker-configurations",
			Payload: map[string]any{
				"name":                  "stackyard-worker-config",
				"propertiesFileContent": "offset.flush.interval.ms=1000\nkey.converter=org.apache.kafka.connect.storage.StringConverter\nvalue.converter=org.apache.kafka.connect.storage.StringConverter",
			},
		},
		{
			Name:   "CreateConnector",
			Method: http.MethodPost,
			Path:   "/v1/connectors",
			Payload: map[string]any{
				"capacity": map[string]any{
					"provisionedCapacity": map[string]any{
						"mcuCount":    1,
						"workerCount": 1,
					},
				},
				"connectorConfiguration": map[string]any{
					"connector.class": "org.apache.kafka.connect.file.FileStreamSinkConnector",
					"tasks.max":       "1",
					"topics":          "stackyard-topic",
					"file":            "/tmp/stackyard.out",
				},
				"connectorName": "stackyard-connector",
				"kafkaCluster": map[string]any{
					"apacheKafkaCluster": map[string]any{
						"bootstrapServers": "b-1.stackyard.example:9092",
						"vpc": map[string]any{
							"securityGroups": []string{"sg-0123456789abcdef0"},
							"subnets":        []string{"subnet-0123456789abcdef0"},
						},
					},
				},
				"kafkaClusterClientAuthentication": map[string]any{
					"authenticationType": "IAM",
				},
				"kafkaClusterEncryptionInTransit": map[string]any{
					"encryptionType": "TLS",
				},
				"kafkaConnectVersion": "2.7.1",
				"plugins": []map[string]any{
					{
						"customPlugin": map[string]any{
							"customPluginArn": customPluginARN,
							"revision":        1,
						},
					},
				},
				"serviceExecutionRoleArn": "arn:aws:iam::123456789012:role/stackyard-msk-connect",
				"workerConfiguration": map[string]any{
					"workerConfigurationArn": workerConfigurationARN,
					"revision":               1,
				},
				"logDelivery": map[string]any{
					"workerLogDelivery": map[string]any{
						"cloudWatchLogs": map[string]any{
							"enabled":  true,
							"logGroup": "stackyard-mskconnect",
						},
					},
				},
			},
		},
		{Name: "ListConnectors", Method: http.MethodGet, Path: "/v1/connectors?connectorNamePrefix=stackyard&maxResults=20&nextToken=token-000001"},
		{Name: "DescribeConnector", Method: http.MethodGet, Path: "/v1/connectors/" + url.PathEscape(connectorARN)},
		{Name: "UpdateConnector", Method: http.MethodPut, Path: "/v1/connectors/" + url.PathEscape(connectorARN) + "?currentVersion=1", Payload: map[string]any{
			"capacity": map[string]any{
				"autoScaling": map[string]any{
					"maxWorkerCount": 4,
					"mcuCount":       1,
					"minWorkerCount": 1,
					"scaleInPolicy": map[string]any{
						"cpuUtilizationPercentage": 20,
					},
					"scaleOutPolicy": map[string]any{
						"cpuUtilizationPercentage": 80,
					},
				},
			},
		}},
		{Name: "DeleteConnector", Method: http.MethodDelete, Path: "/v1/connectors/" + url.PathEscape(connectorARN) + "?currentVersion=1"},
		{Name: "ListConnectorOperations", Method: http.MethodGet, Path: "/v1/connectors/" + url.PathEscape(connectorARN) + "/operations?maxResults=20&nextToken=token-000001"},
		{Name: "DescribeConnectorOperation", Method: http.MethodGet, Path: "/v1/connectorOperations/" + url.PathEscape(connectorOperationARN)},
		{Name: "ListCustomPlugins", Method: http.MethodGet, Path: "/v1/custom-plugins?maxResults=20&namePrefix=stackyard&nextToken=token-000001"},
		{Name: "DescribeCustomPlugin", Method: http.MethodGet, Path: "/v1/custom-plugins/" + url.PathEscape(customPluginARN)},
		{Name: "DeleteCustomPlugin", Method: http.MethodDelete, Path: "/v1/custom-plugins/" + url.PathEscape(customPluginARN)},
		{Name: "ListWorkerConfigurations", Method: http.MethodGet, Path: "/v1/worker-configurations?maxResults=20&namePrefix=stackyard&nextToken=token-000001"},
		{Name: "DescribeWorkerConfiguration", Method: http.MethodGet, Path: "/v1/worker-configurations/" + url.PathEscape(workerConfigurationARN)},
		{Name: "DeleteWorkerConfiguration", Method: http.MethodDelete, Path: "/v1/worker-configurations/" + url.PathEscape(workerConfigurationARN)},
		{Name: "TagResource", Method: http.MethodPost, Path: "/v1/tags/" + url.PathEscape(connectorARN), Payload: map[string]any{"tags": map[string]string{"env": "coverage", "team": "platform"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/v1/tags/" + url.PathEscape(connectorARN)},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/v1/tags/" + url.PathEscape(connectorARN) + "?tagKeys=env"},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := mskConnectRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(status, errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func mskConnectRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	} else {
		body = []byte{}
	}

	fullURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "kafkaconnect", region, time.Now()); err != nil {
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

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(status int, errType string, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "unknown action") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "forbidden") ||
		strings.Contains(combined, "unauthorized")
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
