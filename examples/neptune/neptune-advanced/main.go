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

type requestCase struct {
	Action string
	Params url.Values
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	clusterID := getenv("STACKYARD_NEPTUNE_CLUSTER", "neptune-advanced-cluster")
	instanceID := getenv("STACKYARD_NEPTUNE_INSTANCE", clusterID+"-instance")
	snapshotID := getenv("STACKYARD_NEPTUNE_SNAPSHOT", clusterID+"-snapshot")
	endpointID := getenv("STACKYARD_NEPTUNE_ENDPOINT", clusterID+"-endpoint")
	globalClusterID := getenv("STACKYARD_NEPTUNE_GLOBAL_CLUSTER", clusterID+"-global")
	subscriptionName := getenv("STACKYARD_NEPTUNE_SUBSCRIPTION", clusterID+"-subscription")
	parameterGroupName := getenv("STACKYARD_NEPTUNE_PARAMETER_GROUP", clusterID+"-params")
	subnetGroupName := getenv("STACKYARD_NEPTUNE_SUBNET_GROUP", clusterID+"-subnets")
	defaultClusterARN := getenv(
		"STACKYARD_NEPTUNE_CLUSTER_ARN",
		"arn:aws:rds:us-east-1:123456789012:cluster:neptune-advanced-cluster",
	)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Neptune advanced client using %s\n", endpoint)

	createStatus, createBody, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", withAction("CreateDBCluster", url.Values{
		"DBClusterIdentifier": []string{clusterID},
		"Engine":              []string{"neptune"},
	}))
	if err != nil {
		exitf("CreateDBCluster request failed: %v", err)
	}
	if err := expectSuccess("CreateDBCluster", createStatus, createBody); err != nil {
		exitf("CreateDBCluster response validation failed: %v", err)
	}
	logf("CreateDBCluster succeeded (%d)", createStatus)

	clusterARN := xmlTagValue(string(createBody), "DBClusterArn")
	if clusterARN == "" {
		clusterARN = defaultClusterARN
	}

	requests := []requestCase{
		{Action: "DescribeDBClusters", Params: url.Values{"DBClusterIdentifier": []string{clusterID}}},
		{Action: "DescribeEvents", Params: url.Values{"SourceIdentifier": []string{clusterID}, "SourceType": []string{"db-cluster"}}},
		{Action: "ListTagsForResource", Params: url.Values{"ResourceName": []string{clusterARN}}},
		{
			Action: "CreateDBInstance",
			Params: url.Values{
				"DBInstanceIdentifier": []string{instanceID},
				"Engine":               []string{"neptune"},
			},
		},
		{
			Action: "CreateDBParameterGroup",
			Params: url.Values{
				"DBParameterGroupName":   []string{parameterGroupName},
				"DBParameterGroupFamily": []string{"neptune1"},
				"Description":            []string{"advanced neptune parameters"},
			},
		},
		{
			Action: "CreateDBSubnetGroup",
			Params: url.Values{
				"DBSubnetGroupName": []string{subnetGroupName},
				"SubnetIds.member.1": []string{
					"subnet-12345678",
				},
			},
		},
		{
			Action: "CreateDBClusterSnapshot",
			Params: url.Values{
				"DBClusterSnapshotIdentifier": []string{snapshotID},
				"DBClusterIdentifier":         []string{clusterID},
			},
		},
		{
			Action: "CreateDBClusterEndpoint",
			Params: url.Values{
				"DBClusterEndpointIdentifier": []string{endpointID},
				"DBClusterIdentifier":         []string{clusterID},
				"EndpointType":                []string{"READER"},
			},
		},
		{
			Action: "CreateGlobalCluster",
			Params: url.Values{
				"GlobalClusterIdentifier":   []string{globalClusterID},
				"SourceDBClusterIdentifier": []string{clusterID},
			},
		},
		{
			Action: "CreateEventSubscription",
			Params: url.Values{
				"SubscriptionName": []string{subscriptionName},
				"SnsTopicArn":      []string{"arn:aws:sns:us-east-1:123456789012:stackyard-neptune-topic"},
				"SourceType":       []string{"db-cluster"},
			},
		},
		{
			Action: "AddTagsToResource",
			Params: url.Values{
				"ResourceName":     []string{clusterARN},
				"Tags.Tag.1.Key":   []string{"env"},
				"Tags.Tag.1.Value": []string{"advanced"},
			},
		},
		{
			Action: "AddRoleToDBCluster",
			Params: url.Values{
				"DBClusterIdentifier": []string{clusterID},
				"RoleArn":             []string{"arn:aws:iam::123456789012:role/stackyard-neptune-role"},
			},
		},
		{
			Action: "ModifyDBCluster",
			Params: url.Values{
				"DBClusterIdentifier":   []string{clusterID},
				"BackupRetentionPeriod": []string{"7"},
				"ApplyImmediately":      []string{"true"},
			},
		},
		{
			Action: "FailoverDBCluster",
			Params: url.Values{
				"DBClusterIdentifier": []string{clusterID},
			},
		},
		{
			Action: "RemoveRoleFromDBCluster",
			Params: url.Values{
				"DBClusterIdentifier": []string{clusterID},
				"RoleArn":             []string{"arn:aws:iam::123456789012:role/stackyard-neptune-role"},
			},
		},
		{
			Action: "DeleteEventSubscription",
			Params: url.Values{
				"SubscriptionName": []string{subscriptionName},
			},
		},
		{
			Action: "DeleteGlobalCluster",
			Params: url.Values{
				"GlobalClusterIdentifier": []string{globalClusterID},
			},
		},
		{
			Action: "DeleteDBClusterEndpoint",
			Params: url.Values{
				"DBClusterEndpointIdentifier": []string{endpointID},
			},
		},
		{
			Action: "DeleteDBInstance",
			Params: url.Values{
				"DBInstanceIdentifier": []string{instanceID},
				"SkipFinalSnapshot":    []string{"true"},
			},
		},
		{
			Action: "DeleteDBCluster",
			Params: url.Values{
				"DBClusterIdentifier": []string{clusterID},
				"SkipFinalSnapshot":   []string{"true"},
			},
		},
	}

	for _, reqCase := range requests {
		status, body, err := neptuneRequest(ctx, endpoint, region, creds, http.MethodPost, "/neptune", withAction(reqCase.Action, reqCase.Params))
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	fmt.Println("Done.")
}

func withAction(action string, params url.Values) url.Values {
	values := cloneValues(params)
	values.Set("Action", action)
	return values
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
