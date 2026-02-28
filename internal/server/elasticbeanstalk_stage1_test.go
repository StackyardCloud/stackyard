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
)

func TestElasticBeanstalkStage1RoleAndAccountActions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createApp := url.Values{}
	createApp.Set("Action", "CreateApplication")
	createApp.Set("Version", "2010-12-01")
	createApp.Set("ApplicationName", "stage1-app")
	resp := elasticBeanstalkRequest(t, ts, createApp)
	assertStatus(t, resp, http.StatusOK)

	createVersion := url.Values{}
	createVersion.Set("Action", "CreateApplicationVersion")
	createVersion.Set("ApplicationName", "stage1-app")
	createVersion.Set("VersionLabel", "v1")
	resp = elasticBeanstalkRequest(t, ts, createVersion)
	assertStatus(t, resp, http.StatusOK)

	createEnv := url.Values{}
	createEnv.Set("Action", "CreateEnvironment")
	createEnv.Set("ApplicationName", "stage1-app")
	createEnv.Set("EnvironmentName", "stage1-env")
	createEnv.Set("VersionLabel", "v1")
	createEnv.Set("SolutionStackName", "64bit Amazon Linux 2 v3.6.3 running Go 1")
	resp = elasticBeanstalkRequest(t, ts, createEnv)
	assertStatus(t, resp, http.StatusOK)

	describeAccount := url.Values{}
	describeAccount.Set("Action", "DescribeAccountAttributes")
	resp = elasticBeanstalkRequest(t, ts, describeAccount)
	assertStatus(t, resp, http.StatusOK)
	var accountResp struct {
		Result elasticBeanstalkDescribeAccountAttributesResult `xml:"DescribeAccountAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &accountResp); err != nil {
		t.Fatalf("unmarshal describe account attributes: %v", err)
	}
	if accountResp.Result.ResourceQuotas == nil || accountResp.Result.ResourceQuotas.ApplicationQuota == nil || accountResp.Result.ResourceQuotas.ApplicationQuota.Maximum == nil {
		t.Fatalf("expected resource quotas in describe account attributes response")
	}
	if aws.ToInt32(accountResp.Result.ResourceQuotas.ApplicationQuota.Maximum) <= 0 {
		t.Fatalf("expected positive application quota")
	}

	roleARN := "arn:aws:iam::123456789012:role/AWSElasticBeanstalkOperationsRole"
	associate := url.Values{}
	associate.Set("Action", "AssociateEnvironmentOperationsRole")
	associate.Set("EnvironmentName", "stage1-env")
	associate.Set("OperationsRole", roleARN)
	resp = elasticBeanstalkRequest(t, ts, associate)
	assertStatus(t, resp, http.StatusOK)

	describeEnvs := url.Values{}
	describeEnvs.Set("Action", "DescribeEnvironments")
	describeEnvs.Set("EnvironmentNames.member.1", "stage1-env")
	resp = elasticBeanstalkRequest(t, ts, describeEnvs)
	assertStatus(t, resp, http.StatusOK)
	var describeResp struct {
		Result elasticBeanstalkDescribeEnvironmentsResult `xml:"DescribeEnvironmentsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeResp); err != nil {
		t.Fatalf("unmarshal describe environments: %v", err)
	}
	if len(describeResp.Result.Environments) != 1 {
		t.Fatalf("expected one environment in describe response")
	}
	if aws.ToString(describeResp.Result.Environments[0].OperationsRole) != roleARN {
		t.Fatalf("expected operations role %q, got %q", roleARN, aws.ToString(describeResp.Result.Environments[0].OperationsRole))
	}

	compose := url.Values{}
	compose.Set("Action", "ComposeEnvironments")
	compose.Set("ApplicationName", "stage1-app")
	compose.Set("VersionLabels.member.1", "v1")
	resp = elasticBeanstalkRequest(t, ts, compose)
	assertStatus(t, resp, http.StatusOK)
	var composeResp struct {
		Result elasticBeanstalkComposeEnvironmentsResult `xml:"ComposeEnvironmentsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &composeResp); err != nil {
		t.Fatalf("unmarshal compose environments: %v", err)
	}
	if len(composeResp.Result.Environments) != 1 {
		t.Fatalf("expected one composed environment, got %d", len(composeResp.Result.Environments))
	}

	disassociate := url.Values{}
	disassociate.Set("Action", "DisassociateEnvironmentOperationsRole")
	disassociate.Set("EnvironmentName", "stage1-env")
	resp = elasticBeanstalkRequest(t, ts, disassociate)
	assertStatus(t, resp, http.StatusOK)

	describeAll := url.Values{}
	describeAll.Set("Action", "DescribeEnvironments")
	describeAll.Set("ApplicationName", "stage1-app")
	resp = elasticBeanstalkRequest(t, ts, describeAll)
	assertStatus(t, resp, http.StatusOK)
	var describeAfterDisassociate struct {
		Result elasticBeanstalkDescribeEnvironmentsResult `xml:"DescribeEnvironmentsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeAfterDisassociate); err != nil {
		t.Fatalf("unmarshal describe environments after disassociate: %v", err)
	}
	if len(describeAfterDisassociate.Result.Environments) == 0 {
		t.Fatalf("expected environment after disassociate")
	}
	if aws.ToString(describeAfterDisassociate.Result.Environments[0].OperationsRole) != "" {
		t.Fatalf("expected operations role to be cleared")
	}
}

func TestElasticBeanstalkStage1SDKClientActions(t *testing.T) {
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

	if _, err := client.CreateApplication(ctx, &awseb.CreateApplicationInput{ApplicationName: aws.String("sdk-stage1-app")}); err != nil {
		t.Fatalf("create application: %v", err)
	}
	if _, err := client.CreateApplicationVersion(ctx, &awseb.CreateApplicationVersionInput{
		ApplicationName: aws.String("sdk-stage1-app"),
		VersionLabel:    aws.String("v1"),
	}); err != nil {
		t.Fatalf("create application version: %v", err)
	}
	if _, err := client.CreateEnvironment(ctx, &awseb.CreateEnvironmentInput{
		ApplicationName:   aws.String("sdk-stage1-app"),
		EnvironmentName:   aws.String("sdk-stage1-env"),
		VersionLabel:      aws.String("v1"),
		SolutionStackName: aws.String("64bit Amazon Linux 2 v3.6.3 running Go 1"),
	}); err != nil {
		t.Fatalf("create environment: %v", err)
	}

	outAccount, err := client.DescribeAccountAttributes(ctx, &awseb.DescribeAccountAttributesInput{})
	if err != nil {
		t.Fatalf("describe account attributes: %v", err)
	}
	if outAccount.ResourceQuotas == nil || outAccount.ResourceQuotas.ApplicationQuota == nil || outAccount.ResourceQuotas.ApplicationQuota.Maximum == nil {
		t.Fatalf("expected account resource quotas from SDK call")
	}

	roleARN := "arn:aws:iam::123456789012:role/AWSElasticBeanstalkOperationsRole"
	if _, err := client.AssociateEnvironmentOperationsRole(ctx, &awseb.AssociateEnvironmentOperationsRoleInput{
		EnvironmentName: aws.String("sdk-stage1-env"),
		OperationsRole:  aws.String(roleARN),
	}); err != nil {
		t.Fatalf("associate environment operations role: %v", err)
	}

	outCompose, err := client.ComposeEnvironments(ctx, &awseb.ComposeEnvironmentsInput{
		ApplicationName: aws.String("sdk-stage1-app"),
		VersionLabels:   []string{"v1"},
	})
	if err != nil {
		t.Fatalf("compose environments: %v", err)
	}
	if len(outCompose.Environments) == 0 {
		t.Fatalf("expected composed environments from SDK call")
	}

	if _, err := client.DisassociateEnvironmentOperationsRole(ctx, &awseb.DisassociateEnvironmentOperationsRoleInput{
		EnvironmentName: aws.String("sdk-stage1-env"),
	}); err != nil {
		t.Fatalf("disassociate environment operations role: %v", err)
	}
}
