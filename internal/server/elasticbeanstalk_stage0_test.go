package server

import (
	"bytes"
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

func elasticBeanstalkRequest(t *testing.T, ts *httptest.Server, values url.Values) *http.Response {
	t.Helper()
	body := []byte(values.Encode())
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "elasticbeanstalk")
}

func TestElasticBeanstalkStage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createApp := url.Values{}
	createApp.Set("Action", "CreateApplication")
	createApp.Set("Version", "2010-12-01")
	createApp.Set("ApplicationName", "demo-app")
	createApp.Set("Description", "demo")
	resp := elasticBeanstalkRequest(t, ts, createApp)
	assertStatus(t, resp, http.StatusOK)

	createVersion := url.Values{}
	createVersion.Set("Action", "CreateApplicationVersion")
	createVersion.Set("ApplicationName", "demo-app")
	createVersion.Set("VersionLabel", "v1")
	createVersion.Set("Description", "version")
	createVersion.Set("SourceBundle.S3Bucket", "artifact-bucket")
	createVersion.Set("SourceBundle.S3Key", "app.zip")
	resp = elasticBeanstalkRequest(t, ts, createVersion)
	assertStatus(t, resp, http.StatusOK)

	createTemplate := url.Values{}
	createTemplate.Set("Action", "CreateConfigurationTemplate")
	createTemplate.Set("ApplicationName", "demo-app")
	createTemplate.Set("TemplateName", "default")
	createTemplate.Set("SolutionStackName", "64bit Amazon Linux 2 v3.6.3 running Go 1")
	resp = elasticBeanstalkRequest(t, ts, createTemplate)
	assertStatus(t, resp, http.StatusOK)

	createEnv := url.Values{}
	createEnv.Set("Action", "CreateEnvironment")
	createEnv.Set("ApplicationName", "demo-app")
	createEnv.Set("EnvironmentName", "demo-env")
	createEnv.Set("VersionLabel", "v1")
	createEnv.Set("TemplateName", "default")
	createEnv.Set("SolutionStackName", "64bit Amazon Linux 2 v3.6.3 running Go 1")
	resp = elasticBeanstalkRequest(t, ts, createEnv)
	assertStatus(t, resp, http.StatusOK)
	createEnvBody := mustBody(t, resp)
	var createEnvResp struct {
		Result elasticBeanstalkCreateEnvironmentResult `xml:"CreateEnvironmentResult"`
	}
	if err := xml.Unmarshal(createEnvBody, &createEnvResp); err != nil {
		t.Fatalf("unmarshal create env: %v", err)
	}
	if bytes.Contains(createEnvBody, []byte("<Environment>")) {
		t.Fatalf("expected CreateEnvironment response to use the modeled flat EnvironmentDescription shape")
	}
	if createEnvResp.Result.EnvironmentId == nil || aws.ToString(createEnvResp.Result.EnvironmentId) == "" {
		t.Fatalf("expected environment id")
	}

	updateEnv := url.Values{}
	updateEnv.Set("Action", "UpdateEnvironment")
	updateEnv.Set("EnvironmentName", "demo-env")
	updateEnv.Set("Description", "updated")
	resp = elasticBeanstalkRequest(t, ts, updateEnv)
	assertStatus(t, resp, http.StatusOK)
	updateEnvBody := mustBody(t, resp)
	var updateEnvResp struct {
		Result elasticBeanstalkUpdateEnvironmentResult `xml:"UpdateEnvironmentResult"`
	}
	if err := xml.Unmarshal(updateEnvBody, &updateEnvResp); err != nil {
		t.Fatalf("unmarshal update env: %v", err)
	}
	if bytes.Contains(updateEnvBody, []byte("<Environment>")) {
		t.Fatalf("expected UpdateEnvironment response to use the modeled flat EnvironmentDescription shape")
	}
	if updateEnvResp.Result.EnvironmentName == nil || aws.ToString(updateEnvResp.Result.EnvironmentName) != "demo-env" {
		t.Fatalf("expected updated environment name in flat response")
	}

	describeEnvs := url.Values{}
	describeEnvs.Set("Action", "DescribeEnvironments")
	describeEnvs.Set("ApplicationName", "demo-app")
	resp = elasticBeanstalkRequest(t, ts, describeEnvs)
	assertStatus(t, resp, http.StatusOK)
	var describeEnvsResp struct {
		Result elasticBeanstalkDescribeEnvironmentsResult `xml:"DescribeEnvironmentsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeEnvsResp); err != nil {
		t.Fatalf("unmarshal describe envs: %v", err)
	}
	if len(describeEnvsResp.Result.Environments) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(describeEnvsResp.Result.Environments))
	}

	describeEvents := url.Values{}
	describeEvents.Set("Action", "DescribeEvents")
	describeEvents.Set("EnvironmentName", "demo-env")
	resp = elasticBeanstalkRequest(t, ts, describeEvents)
	assertStatus(t, resp, http.StatusOK)
	var describeEventsResp struct {
		Result elasticBeanstalkDescribeEventsResult `xml:"DescribeEventsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeEventsResp); err != nil {
		t.Fatalf("unmarshal describe events: %v", err)
	}
	if len(describeEventsResp.Result.Events) == 0 {
		t.Fatalf("expected events")
	}

	checkDNS := url.Values{}
	checkDNS.Set("Action", "CheckDNSAvailability")
	checkDNS.Set("CNAMEPrefix", "demo-env")
	resp = elasticBeanstalkRequest(t, ts, checkDNS)
	assertStatus(t, resp, http.StatusOK)
	var checkResp struct {
		Result elasticBeanstalkCheckDNSAvailabilityResult `xml:"CheckDNSAvailabilityResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &checkResp); err != nil {
		t.Fatalf("unmarshal check dns: %v", err)
	}
	if checkResp.Result.Available == nil {
		t.Fatalf("expected availability result")
	}

	requestInfo := url.Values{}
	requestInfo.Set("Action", "RequestEnvironmentInfo")
	requestInfo.Set("EnvironmentName", "demo-env")
	requestInfo.Set("InfoType", "tail")
	resp = elasticBeanstalkRequest(t, ts, requestInfo)
	assertStatus(t, resp, http.StatusOK)

	retrieveInfo := url.Values{}
	retrieveInfo.Set("Action", "RetrieveEnvironmentInfo")
	retrieveInfo.Set("EnvironmentName", "demo-env")
	retrieveInfo.Set("InfoType", "tail")
	resp = elasticBeanstalkRequest(t, ts, retrieveInfo)
	assertStatus(t, resp, http.StatusOK)

	updateTags := url.Values{}
	updateTags.Set("Action", "UpdateTagsForResource")
	updateTags.Set("ResourceArn", "arn:aws:elasticbeanstalk:us-east-1:123456789012:application/demo-app")
	updateTags.Set("TagsToAdd.member.1.Key", "env")
	updateTags.Set("TagsToAdd.member.1.Value", "test")
	resp = elasticBeanstalkRequest(t, ts, updateTags)
	assertStatus(t, resp, http.StatusOK)

	listTags := url.Values{}
	listTags.Set("Action", "ListTagsForResource")
	listTags.Set("ResourceArn", "arn:aws:elasticbeanstalk:us-east-1:123456789012:application/demo-app")
	resp = elasticBeanstalkRequest(t, ts, listTags)
	assertStatus(t, resp, http.StatusOK)
	var listTagsResp struct {
		Result elasticBeanstalkListTagsForResourceResult `xml:"ListTagsForResourceResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listTagsResp); err != nil {
		t.Fatalf("unmarshal list tags: %v", err)
	}
	if len(listTagsResp.Result.ResourceTags) != 1 {
		t.Fatalf("expected one resource tag")
	}

	terminateEnv := url.Values{}
	terminateEnv.Set("Action", "TerminateEnvironment")
	terminateEnv.Set("EnvironmentName", "demo-env")
	resp = elasticBeanstalkRequest(t, ts, terminateEnv)
	assertStatus(t, resp, http.StatusOK)
	terminateEnvBody := mustBody(t, resp)
	var terminateEnvResp struct {
		Result elasticBeanstalkTerminateEnvironmentResult `xml:"TerminateEnvironmentResult"`
	}
	if err := xml.Unmarshal(terminateEnvBody, &terminateEnvResp); err != nil {
		t.Fatalf("unmarshal terminate env: %v", err)
	}
	if bytes.Contains(terminateEnvBody, []byte("<Environment>")) {
		t.Fatalf("expected TerminateEnvironment response to use the modeled flat EnvironmentDescription shape")
	}
	if terminateEnvResp.Result.Status == "" {
		t.Fatalf("expected terminate response to include environment status")
	}

	deleteTemplate := url.Values{}
	deleteTemplate.Set("Action", "DeleteConfigurationTemplate")
	deleteTemplate.Set("ApplicationName", "demo-app")
	deleteTemplate.Set("TemplateName", "default")
	resp = elasticBeanstalkRequest(t, ts, deleteTemplate)
	assertStatus(t, resp, http.StatusOK)

	deleteVersion := url.Values{}
	deleteVersion.Set("Action", "DeleteApplicationVersion")
	deleteVersion.Set("ApplicationName", "demo-app")
	deleteVersion.Set("VersionLabel", "v1")
	resp = elasticBeanstalkRequest(t, ts, deleteVersion)
	assertStatus(t, resp, http.StatusOK)

	deleteApp := url.Values{}
	deleteApp.Set("Action", "DeleteApplication")
	deleteApp.Set("ApplicationName", "demo-app")
	deleteApp.Set("TerminateEnvByForce", "true")
	resp = elasticBeanstalkRequest(t, ts, deleteApp)
	assertStatus(t, resp, http.StatusOK)

	invalidAction := url.Values{}
	invalidAction.Set("Action", "DefinitelyNotARealAction")
	resp = elasticBeanstalkRequest(t, ts, invalidAction)
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestElasticBeanstalkStage0OperationCoverage(t *testing.T) {
	if len(elasticBeanstalkOperations) != 47 {
		t.Fatalf("expected 47 Elastic Beanstalk operations from docs, got %d", len(elasticBeanstalkOperations))
	}
	if len(elasticBeanstalkOperationByName) != len(elasticBeanstalkOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateApplication",
		"CreateApplicationVersion",
		"CreateEnvironment",
		"DescribeApplications",
		"DescribeEnvironments",
		"DescribeEvents",
		"CheckDNSAvailability",
		"ListAvailableSolutionStacks",
		"UpdateTagsForResource",
		"ValidateConfigurationSettings",
	}
	for _, name := range required {
		if _, ok := elasticBeanstalkOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(elasticBeanstalkDataTypes) != 56 {
		t.Fatalf("expected 56 Elastic Beanstalk data types from docs, got %d", len(elasticBeanstalkDataTypes))
	}
	if len(elasticBeanstalkDataTypeByName) != len(elasticBeanstalkDataTypes) {
		t.Fatalf("expected unique data type names")
	}
	requiredTypes := []string{
		"ApplicationDescription",
		"ApplicationVersionDescription",
		"ConfigurationOptionDescription",
		"EnvironmentDescription",
		"EnvironmentResourcesDescription",
		"ManagedAction",
		"PlatformSummary",
		"ResourceQuotas",
		"ValidateConfigurationSettings",
		"ValidationMessage",
	}
	for _, typeName := range requiredTypes {
		if _, ok := elasticBeanstalkDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestElasticBeanstalkStage0SDKClientLifecycle(t *testing.T) {
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

	if _, err := client.CreateApplication(ctx, &awseb.CreateApplicationInput{ApplicationName: aws.String("sdk-app")}); err != nil {
		t.Fatalf("create application: %v", err)
	}
	if _, err := client.CreateApplicationVersion(ctx, &awseb.CreateApplicationVersionInput{
		ApplicationName: aws.String("sdk-app"),
		VersionLabel:    aws.String("v1"),
	}); err != nil {
		t.Fatalf("create application version: %v", err)
	}
	if _, err := client.CreateEnvironment(ctx, &awseb.CreateEnvironmentInput{
		ApplicationName:   aws.String("sdk-app"),
		EnvironmentName:   aws.String("sdk-env"),
		SolutionStackName: aws.String("64bit Amazon Linux 2 v3.6.3 running Go 1"),
		VersionLabel:      aws.String("v1"),
	}); err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if _, err := client.UpdateEnvironment(ctx, &awseb.UpdateEnvironmentInput{
		EnvironmentName: aws.String("sdk-env"),
		Description:     aws.String("updated"),
	}); err != nil {
		t.Fatalf("update environment: %v", err)
	}
	if _, err := client.DescribeEnvironments(ctx, &awseb.DescribeEnvironmentsInput{ApplicationName: aws.String("sdk-app")}); err != nil {
		t.Fatalf("describe environments: %v", err)
	}
	if _, err := client.TerminateEnvironment(ctx, &awseb.TerminateEnvironmentInput{EnvironmentName: aws.String("sdk-env")}); err != nil {
		t.Fatalf("terminate environment: %v", err)
	}
}
