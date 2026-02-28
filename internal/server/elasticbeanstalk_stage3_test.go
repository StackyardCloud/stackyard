package server

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awseb "github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	awsebtypes "github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk/types"
)

func TestElasticBeanstalkStage3HealthAndConfigActions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createApp := url.Values{}
	createApp.Set("Action", "CreateApplication")
	createApp.Set("Version", "2010-12-01")
	createApp.Set("ApplicationName", "stage3-app")
	resp := elasticBeanstalkRequest(t, ts, createApp)
	assertStatus(t, resp, http.StatusOK)

	createVersion := url.Values{}
	createVersion.Set("Action", "CreateApplicationVersion")
	createVersion.Set("ApplicationName", "stage3-app")
	createVersion.Set("VersionLabel", "v1")
	resp = elasticBeanstalkRequest(t, ts, createVersion)
	assertStatus(t, resp, http.StatusOK)

	createEnv := url.Values{}
	createEnv.Set("Action", "CreateEnvironment")
	createEnv.Set("ApplicationName", "stage3-app")
	createEnv.Set("EnvironmentName", "stage3-env")
	createEnv.Set("VersionLabel", "v1")
	createEnv.Set("OptionSettings.member.1.Namespace", "aws:elasticbeanstalk:application:environment")
	createEnv.Set("OptionSettings.member.1.OptionName", "K")
	createEnv.Set("OptionSettings.member.1.Value", "V")
	resp = elasticBeanstalkRequest(t, ts, createEnv)
	assertStatus(t, resp, http.StatusOK)

	describeEnvHealth := url.Values{}
	describeEnvHealth.Set("Action", "DescribeEnvironmentHealth")
	describeEnvHealth.Set("EnvironmentName", "stage3-env")
	resp = elasticBeanstalkRequest(t, ts, describeEnvHealth)
	assertStatus(t, resp, http.StatusOK)
	var envHealthResp struct {
		Result elasticBeanstalkDescribeEnvironmentHealthResult `xml:"DescribeEnvironmentHealthResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &envHealthResp); err != nil {
		t.Fatalf("unmarshal describe environment health: %v", err)
	}
	if aws.ToString(envHealthResp.Result.EnvironmentName) != "stage3-env" {
		t.Fatalf("unexpected environment name in health response")
	}

	describeInstancesHealth := url.Values{}
	describeInstancesHealth.Set("Action", "DescribeInstancesHealth")
	describeInstancesHealth.Set("EnvironmentName", "stage3-env")
	resp = elasticBeanstalkRequest(t, ts, describeInstancesHealth)
	assertStatus(t, resp, http.StatusOK)
	var instHealthResp struct {
		Result elasticBeanstalkDescribeInstancesHealthResult `xml:"DescribeInstancesHealthResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &instHealthResp); err != nil {
		t.Fatalf("unmarshal describe instances health: %v", err)
	}
	if len(instHealthResp.Result.InstanceHealthList) == 0 {
		t.Fatalf("expected instance health results")
	}

	deleteEnvConfig := url.Values{}
	deleteEnvConfig.Set("Action", "DeleteEnvironmentConfiguration")
	deleteEnvConfig.Set("ApplicationName", "stage3-app")
	deleteEnvConfig.Set("EnvironmentName", "stage3-env")
	resp = elasticBeanstalkRequest(t, ts, deleteEnvConfig)
	assertStatus(t, resp, http.StatusOK)

	describeConfig := url.Values{}
	describeConfig.Set("Action", "DescribeConfigurationSettings")
	describeConfig.Set("ApplicationName", "stage3-app")
	describeConfig.Set("EnvironmentName", "stage3-env")
	resp = elasticBeanstalkRequest(t, ts, describeConfig)
	assertStatus(t, resp, http.StatusOK)
	var configResp struct {
		Result elasticBeanstalkDescribeConfigurationSettingsResult `xml:"DescribeConfigurationSettingsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &configResp); err != nil {
		t.Fatalf("unmarshal describe configuration settings: %v", err)
	}
	if len(configResp.Result.ConfigurationSettings) != 1 {
		t.Fatalf("expected one configuration settings result")
	}
	if len(configResp.Result.ConfigurationSettings[0].OptionSettings) != 0 {
		t.Fatalf("expected option settings to be cleared after delete environment configuration")
	}

	createPlatform := url.Values{}
	createPlatform.Set("Action", "CreatePlatformVersion")
	createPlatform.Set("PlatformName", "stage3-platform")
	createPlatform.Set("PlatformVersion", "1.0.0")
	createPlatform.Set("PlatformDefinitionBundle.S3Bucket", "bundle-bucket")
	createPlatform.Set("PlatformDefinitionBundle.S3Key", "platform.zip")
	resp = elasticBeanstalkRequest(t, ts, createPlatform)
	assertStatus(t, resp, http.StatusOK)

	listBranches := url.Values{}
	listBranches.Set("Action", "ListPlatformBranches")
	listBranches.Set("Filters.member.1.Attribute", "PlatformName")
	listBranches.Set("Filters.member.1.Operator", "=")
	listBranches.Set("Filters.member.1.Values.member.1", "stage3-platform")
	resp = elasticBeanstalkRequest(t, ts, listBranches)
	assertStatus(t, resp, http.StatusOK)
	var branchesResp struct {
		Result elasticBeanstalkListPlatformBranchesResult `xml:"ListPlatformBranchesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &branchesResp); err != nil {
		t.Fatalf("unmarshal list platform branches: %v", err)
	}
	if len(branchesResp.Result.PlatformBranchSummaryList) != 1 {
		t.Fatalf("expected one platform branch summary, got %d", len(branchesResp.Result.PlatformBranchSummaryList))
	}
}

func TestElasticBeanstalkStage3SDKClientActions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == awseb.ServiceID {
			return aws.Endpoint{
				URL:               ts.URL,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	client := awseb.NewFromConfig(cfg)

	if _, err := client.CreateApplication(ctx, &awseb.CreateApplicationInput{ApplicationName: aws.String("sdk-stage3-app")}); err != nil {
		t.Fatalf("create application: %v", err)
	}
	if _, err := client.CreateApplicationVersion(ctx, &awseb.CreateApplicationVersionInput{
		ApplicationName: aws.String("sdk-stage3-app"),
		VersionLabel:    aws.String("v1"),
	}); err != nil {
		t.Fatalf("create application version: %v", err)
	}
	if _, err := client.CreateEnvironment(ctx, &awseb.CreateEnvironmentInput{
		ApplicationName: aws.String("sdk-stage3-app"),
		EnvironmentName: aws.String("sdk-stage3-env"),
		VersionLabel:    aws.String("v1"),
	}); err != nil {
		t.Fatalf("create environment: %v", err)
	}

	envHealthOut, err := client.DescribeEnvironmentHealth(ctx, &awseb.DescribeEnvironmentHealthInput{
		EnvironmentName: aws.String("sdk-stage3-env"),
	})
	if err != nil {
		t.Fatalf("describe environment health: %v", err)
	}
	if aws.ToString(envHealthOut.EnvironmentName) != "sdk-stage3-env" {
		t.Fatalf("unexpected environment name from sdk describe environment health")
	}

	instHealthOut, err := client.DescribeInstancesHealth(ctx, &awseb.DescribeInstancesHealthInput{
		EnvironmentName: aws.String("sdk-stage3-env"),
	})
	if err != nil {
		t.Fatalf("describe instances health: %v", err)
	}
	if len(instHealthOut.InstanceHealthList) == 0 {
		t.Fatalf("expected instance health list")
	}

	if _, err := client.DeleteEnvironmentConfiguration(ctx, &awseb.DeleteEnvironmentConfigurationInput{
		ApplicationName: aws.String("sdk-stage3-app"),
		EnvironmentName: aws.String("sdk-stage3-env"),
	}); err != nil {
		t.Fatalf("delete environment configuration: %v", err)
	}

	if _, err := client.CreatePlatformVersion(ctx, &awseb.CreatePlatformVersionInput{
		PlatformName:    aws.String("sdk-stage3-platform"),
		PlatformVersion: aws.String("1.0.0"),
		PlatformDefinitionBundle: &awsebtypes.S3Location{
			S3Bucket: aws.String("bundle-bucket"),
			S3Key:    aws.String("platform.zip"),
		},
	}); err != nil {
		t.Fatalf("create platform version: %v", err)
	}

	branchesOut, err := client.ListPlatformBranches(ctx, &awseb.ListPlatformBranchesInput{
		Filters: []awsebtypes.SearchFilter{{
			Attribute: aws.String("PlatformName"),
			Operator:  aws.String("="),
			Values:    []string{"sdk-stage3-platform"},
		}},
	})
	if err != nil {
		t.Fatalf("list platform branches: %v", err)
	}
	if len(branchesOut.PlatformBranchSummaryList) == 0 {
		t.Fatalf("expected platform branch summaries")
	}
}
