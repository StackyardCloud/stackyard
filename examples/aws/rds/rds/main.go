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
	prefix := getenv("STACKYARD_RDS_PREFIX", "rds")

	dbInstanceID := makeName(prefix, "instance")
	subnetGroup := makeName(prefix, "subnet")
	parameterGroup := makeName(prefix, "param")
	optionGroup := makeName(prefix, "option")
	snapshotID := makeName(prefix, "snapshot")
	dbInstanceARN := fmt.Sprintf("arn:aws:rds:%s:123456789012:db:%s", region, dbInstanceID)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard RDS advanced client using %s\n", endpoint)

	steps := []requestCase{
		{
			Action: "CreateDBSubnetGroup",
			Params: url.Values{
				"Action":                   []string{"CreateDBSubnetGroup"},
				"DBSubnetGroupName":        []string{subnetGroup},
				"DBSubnetGroupDescription": []string{"stackyard rds subnet group"},
				"SubnetIds.member.1":       []string{"subnet-12345678"},
			},
		},
		{
			Action: "CreateDBParameterGroup",
			Params: url.Values{
				"Action":                 []string{"CreateDBParameterGroup"},
				"DBParameterGroupName":   []string{parameterGroup},
				"DBParameterGroupFamily": []string{"mysql8.0"},
				"Description":            []string{"stackyard rds parameter group"},
			},
		},
		{
			Action: "ModifyDBParameterGroup",
			Params: url.Values{
				"Action":                             []string{"ModifyDBParameterGroup"},
				"DBParameterGroupName":               []string{parameterGroup},
				"Parameters.member.1.ParameterName":  []string{"autocommit"},
				"Parameters.member.1.ParameterValue": []string{"1"},
				"Parameters.member.1.ApplyMethod":    []string{"immediate"},
			},
		},
		{
			Action: "CreateOptionGroup",
			Params: url.Values{
				"Action":                 []string{"CreateOptionGroup"},
				"OptionGroupName":        []string{optionGroup},
				"EngineName":             []string{"mysql"},
				"MajorEngineVersion":     []string{"8.0"},
				"OptionGroupDescription": []string{"stackyard rds option group"},
			},
		},
		{
			Action: "CreateDBInstance",
			Params: url.Values{
				"Action":               []string{"CreateDBInstance"},
				"DBInstanceIdentifier": []string{dbInstanceID},
				"Engine":               []string{"mysql"},
				"DBInstanceClass":      []string{"db.t3.micro"},
				"AllocatedStorage":     []string{"20"},
				"MasterUsername":       []string{"admin"},
				"MasterUserPassword":   []string{"Secret1234"},
				"DBSubnetGroupName":    []string{subnetGroup},
				"DBParameterGroupName": []string{parameterGroup},
				"OptionGroupName":      []string{optionGroup},
			},
		},
		{
			Action: "ModifyDBInstance",
			Params: url.Values{
				"Action":                []string{"ModifyDBInstance"},
				"DBInstanceIdentifier":  []string{dbInstanceID},
				"ApplyImmediately":      []string{"true"},
				"BackupRetentionPeriod": []string{"3"},
			},
		},
		{
			Action: "CreateDBSnapshot",
			Params: url.Values{
				"Action":               []string{"CreateDBSnapshot"},
				"DBInstanceIdentifier": []string{dbInstanceID},
				"DBSnapshotIdentifier": []string{snapshotID},
			},
		},
		{
			Action: "DescribeDBSnapshots",
			Params: url.Values{
				"Action":               []string{"DescribeDBSnapshots"},
				"DBSnapshotIdentifier": []string{snapshotID},
			},
		},
		{
			Action: "RebootDBInstance",
			Params: url.Values{
				"Action":               []string{"RebootDBInstance"},
				"DBInstanceIdentifier": []string{dbInstanceID},
			},
		},
		{
			Action: "AddTagsToResource",
			Params: url.Values{
				"Action":              []string{"AddTagsToResource"},
				"ResourceName":        []string{dbInstanceARN},
				"Tags.member.1.Key":   []string{"env"},
				"Tags.member.1.Value": []string{"dev"},
				"Tags.member.2.Key":   []string{"owner"},
				"Tags.member.2.Value": []string{"stackyard"},
			},
		},
		{
			Action: "ListTagsForResource",
			Params: url.Values{
				"Action":       []string{"ListTagsForResource"},
				"ResourceName": []string{dbInstanceARN},
			},
		},
		{
			Action: "RemoveTagsFromResource",
			Params: url.Values{
				"Action":           []string{"RemoveTagsFromResource"},
				"ResourceName":     []string{dbInstanceARN},
				"TagKeys.member.1": []string{"owner"},
			},
		},
		{
			Action: "DescribeDBInstances",
			Params: url.Values{
				"Action":               []string{"DescribeDBInstances"},
				"DBInstanceIdentifier": []string{dbInstanceID},
			},
		},
	}

	for _, step := range steps {
		if err := perform(ctx, endpoint, region, creds, step); err != nil {
			exitf("%s: %v", step.Action, err)
		}
		logf("%s succeeded", step.Action)
	}

	teardown := []requestCase{
		{
			Action: "DeleteDBSnapshot",
			Params: url.Values{
				"Action":               []string{"DeleteDBSnapshot"},
				"DBSnapshotIdentifier": []string{snapshotID},
			},
		},
		{
			Action: "DeleteDBInstance",
			Params: url.Values{
				"Action":               []string{"DeleteDBInstance"},
				"DBInstanceIdentifier": []string{dbInstanceID},
				"SkipFinalSnapshot":    []string{"true"},
			},
		},
		{
			Action: "DeleteOptionGroup",
			Params: url.Values{
				"Action":          []string{"DeleteOptionGroup"},
				"OptionGroupName": []string{optionGroup},
			},
		},
		{
			Action: "DeleteDBParameterGroup",
			Params: url.Values{
				"Action":               []string{"DeleteDBParameterGroup"},
				"DBParameterGroupName": []string{parameterGroup},
			},
		},
		{
			Action: "DeleteDBSubnetGroup",
			Params: url.Values{
				"Action":            []string{"DeleteDBSubnetGroup"},
				"DBSubnetGroupName": []string{subnetGroup},
			},
		},
	}

	for _, step := range teardown {
		if err := perform(ctx, endpoint, region, creds, step); err != nil {
			exitf("%s: %v", step.Action, err)
		}
		logf("%s succeeded", step.Action)
	}

	fmt.Println("Done.")
}

type requestCase struct {
	Action string
	Params url.Values
}

func perform(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, step requestCase) error {
	status, body, err := rdsRequest(ctx, endpoint, region, creds, step.Params)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected %d, got %d: %s", http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	return nil
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

func makeName(prefix, suffix string) string {
	base := strings.TrimSpace(prefix)
	if base == "" {
		base = "rds"
	}
	name := strings.ToLower(base + "-" + suffix)
	if len(name) <= 60 {
		return name
	}
	return name[:60]
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
