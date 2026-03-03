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

type queryCall struct {
	Name   string
	Params url.Values
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

	fmt.Printf("Stackyard Elastic Beanstalk advanced client using %s\n", endpoint)

	appName := "stackyard-eb-app"
	envName := "stackyard-eb-env"
	arn := "arn:aws:elasticbeanstalk:us-east-1:123456789012:application/" + appName
	solutionStack := "64bit Amazon Linux 2 v3.6.3 running Go 1"

	calls := []queryCall{
		{Name: "CreateApplication", Params: query("CreateApplication", map[string]string{"ApplicationName": appName, "Description": "stackyard elastic beanstalk app"})},
		{Name: "CreateApplicationVersion", Params: query("CreateApplicationVersion", map[string]string{"ApplicationName": appName, "VersionLabel": "v1", "Description": "stackyard version"})},
		{Name: "CreateEnvironment", Params: query("CreateEnvironment", map[string]string{"ApplicationName": appName, "EnvironmentName": envName, "VersionLabel": "v1", "SolutionStackName": solutionStack})},
		{Name: "DescribeEnvironments", Params: query("DescribeEnvironments", map[string]string{"ApplicationName": appName})},
		{Name: "DescribeEnvironmentHealth", Params: query("DescribeEnvironmentHealth", map[string]string{"EnvironmentName": envName})},
		{Name: "DescribeEvents", Params: query("DescribeEvents", map[string]string{"EnvironmentName": envName, "MaxRecords": "10"})},
		{Name: "ListAvailableSolutionStacks", Params: query("ListAvailableSolutionStacks", nil)},
		{Name: "CheckDNSAvailability", Params: query("CheckDNSAvailability", map[string]string{"CNAMEPrefix": envName})},
		{Name: "UpdateTagsForResource", Params: query("UpdateTagsForResource", map[string]string{"ResourceArn": arn, "TagsToAdd.member.1.Key": "env", "TagsToAdd.member.1.Value": "dev"})},
		{Name: "ListTagsForResource", Params: query("ListTagsForResource", map[string]string{"ResourceArn": arn})},
		{Name: "TerminateEnvironment", Params: query("TerminateEnvironment", map[string]string{"EnvironmentName": envName})},
		{Name: "DeleteApplicationVersion", Params: query("DeleteApplicationVersion", map[string]string{"ApplicationName": appName, "VersionLabel": "v1"})},
		{Name: "DeleteApplication", Params: query("DeleteApplication", map[string]string{"ApplicationName": appName, "TerminateEnvByForce": "true"})},
	}

	for _, call := range calls {
		status, body, err := elasticBeanstalkQueryRequest(ctx, endpoint, region, creds, call.Params)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", call.Name, status)
	}

	fmt.Println("Done.")
}

func query(action string, extras map[string]string) url.Values {
	v := url.Values{}
	v.Set("Action", action)
	v.Set("Version", "2010-12-01")
	for key, value := range extras {
		v.Set(key, value)
	}
	return v
}

func elasticBeanstalkQueryRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	params url.Values,
) (int, []byte, error) {
	body := []byte(params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "elasticbeanstalk", region, time.Now()); err != nil {
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
