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

func TestElasticBeanstalkStage4ManagedActionsAndLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createApp := url.Values{}
	createApp.Set("Action", "CreateApplication")
	createApp.Set("Version", "2010-12-01")
	createApp.Set("ApplicationName", "stage4-app")
	resp := elasticBeanstalkRequest(t, ts, createApp)
	assertStatus(t, resp, http.StatusOK)

	createVersion := url.Values{}
	createVersion.Set("Action", "CreateApplicationVersion")
	createVersion.Set("ApplicationName", "stage4-app")
	createVersion.Set("VersionLabel", "v1")
	resp = elasticBeanstalkRequest(t, ts, createVersion)
	assertStatus(t, resp, http.StatusOK)

	createEnv := url.Values{}
	createEnv.Set("Action", "CreateEnvironment")
	createEnv.Set("ApplicationName", "stage4-app")
	createEnv.Set("EnvironmentName", "stage4-env")
	createEnv.Set("VersionLabel", "v1")
	resp = elasticBeanstalkRequest(t, ts, createEnv)
	assertStatus(t, resp, http.StatusOK)

	describeManaged := url.Values{}
	describeManaged.Set("Action", "DescribeEnvironmentManagedActions")
	describeManaged.Set("EnvironmentName", "stage4-env")
	describeManaged.Set("Status", "Scheduled")
	resp = elasticBeanstalkRequest(t, ts, describeManaged)
	assertStatus(t, resp, http.StatusOK)
	var describeManagedResp struct {
		Result elasticBeanstalkDescribeEnvironmentManagedActionsResult `xml:"DescribeEnvironmentManagedActionsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeManagedResp); err != nil {
		t.Fatalf("unmarshal describe managed actions: %v", err)
	}
	if len(describeManagedResp.Result.ManagedActions) != 1 {
		t.Fatalf("expected one scheduled managed action, got %d", len(describeManagedResp.Result.ManagedActions))
	}
	actionID := aws.ToString(describeManagedResp.Result.ManagedActions[0].ActionId)
	if actionID == "" {
		t.Fatalf("expected managed action id")
	}

	applyManaged := url.Values{}
	applyManaged.Set("Action", "ApplyEnvironmentManagedAction")
	applyManaged.Set("EnvironmentName", "stage4-env")
	applyManaged.Set("ActionId", actionID)
	resp = elasticBeanstalkRequest(t, ts, applyManaged)
	assertStatus(t, resp, http.StatusOK)
	var applyResp struct {
		Result elasticBeanstalkApplyEnvironmentManagedActionResult `xml:"ApplyEnvironmentManagedActionResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &applyResp); err != nil {
		t.Fatalf("unmarshal apply managed action: %v", err)
	}
	if aws.ToString(applyResp.Result.ActionId) != actionID {
		t.Fatalf("expected action id %q, got %q", actionID, aws.ToString(applyResp.Result.ActionId))
	}

	resp = elasticBeanstalkRequest(t, ts, describeManaged)
	assertStatus(t, resp, http.StatusOK)
	var describeManagedRespAfterApply struct {
		Result elasticBeanstalkDescribeEnvironmentManagedActionsResult `xml:"DescribeEnvironmentManagedActionsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeManagedRespAfterApply); err != nil {
		t.Fatalf("unmarshal describe managed actions after apply: %v", err)
	}
	if len(describeManagedRespAfterApply.Result.ManagedActions) != 0 {
		t.Fatalf("expected no scheduled managed actions after apply")
	}

	describeHistory := url.Values{}
	describeHistory.Set("Action", "DescribeEnvironmentManagedActionHistory")
	describeHistory.Set("EnvironmentName", "stage4-env")
	describeHistory.Set("MaxItems", "10")
	resp = elasticBeanstalkRequest(t, ts, describeHistory)
	assertStatus(t, resp, http.StatusOK)
	var historyResp struct {
		Result elasticBeanstalkDescribeEnvironmentManagedActionHistoryResult `xml:"DescribeEnvironmentManagedActionHistoryResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &historyResp); err != nil {
		t.Fatalf("unmarshal describe managed action history: %v", err)
	}
	if len(historyResp.Result.ManagedActionHistoryItems) != 1 {
		t.Fatalf("expected one managed action history item, got %d", len(historyResp.Result.ManagedActionHistoryItems))
	}
	if historyResp.Result.ManagedActionHistoryItems[0].Status != awsebtypes.ActionHistoryStatusCompleted {
		t.Fatalf("expected completed history status, got %q", historyResp.Result.ManagedActionHistoryItems[0].Status)
	}

	updateLifecycle := url.Values{}
	updateLifecycle.Set("Action", "UpdateApplicationResourceLifecycle")
	updateLifecycle.Set("ApplicationName", "stage4-app")
	updateLifecycle.Set("ResourceLifecycleConfig.ServiceRole", "arn:aws:iam::123456789012:role/eb-service-role")
	updateLifecycle.Set("ResourceLifecycleConfig.VersionLifecycleConfig.MaxAgeRule.Enabled", "true")
	updateLifecycle.Set("ResourceLifecycleConfig.VersionLifecycleConfig.MaxAgeRule.DeleteSourceFromS3", "true")
	updateLifecycle.Set("ResourceLifecycleConfig.VersionLifecycleConfig.MaxAgeRule.MaxAgeInDays", "30")
	updateLifecycle.Set("ResourceLifecycleConfig.VersionLifecycleConfig.MaxCountRule.Enabled", "true")
	updateLifecycle.Set("ResourceLifecycleConfig.VersionLifecycleConfig.MaxCountRule.DeleteSourceFromS3", "false")
	updateLifecycle.Set("ResourceLifecycleConfig.VersionLifecycleConfig.MaxCountRule.MaxCount", "50")
	resp = elasticBeanstalkRequest(t, ts, updateLifecycle)
	assertStatus(t, resp, http.StatusOK)
	var lifecycleResp struct {
		Result elasticBeanstalkUpdateApplicationResourceLifecycleResult `xml:"UpdateApplicationResourceLifecycleResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &lifecycleResp); err != nil {
		t.Fatalf("unmarshal update application resource lifecycle: %v", err)
	}
	if aws.ToString(lifecycleResp.Result.ApplicationName) != "stage4-app" {
		t.Fatalf("expected application name stage4-app")
	}
	if lifecycleResp.Result.ResourceLifecycleConfig == nil {
		t.Fatalf("expected lifecycle config in response")
	}
	if aws.ToString(lifecycleResp.Result.ResourceLifecycleConfig.ServiceRole) != "arn:aws:iam::123456789012:role/eb-service-role" {
		t.Fatalf("unexpected service role in response")
	}
}

func TestElasticBeanstalkStage4SDKClientManagedActionsAndLifecycle(t *testing.T) {
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

	if _, err := client.CreateApplication(ctx, &awseb.CreateApplicationInput{ApplicationName: aws.String("sdk-stage4-app")}); err != nil {
		t.Fatalf("create application: %v", err)
	}
	if _, err := client.CreateApplicationVersion(ctx, &awseb.CreateApplicationVersionInput{
		ApplicationName: aws.String("sdk-stage4-app"),
		VersionLabel:    aws.String("v1"),
	}); err != nil {
		t.Fatalf("create application version: %v", err)
	}
	if _, err := client.CreateEnvironment(ctx, &awseb.CreateEnvironmentInput{
		ApplicationName: aws.String("sdk-stage4-app"),
		EnvironmentName: aws.String("sdk-stage4-env"),
		VersionLabel:    aws.String("v1"),
	}); err != nil {
		t.Fatalf("create environment: %v", err)
	}

	managedOut, err := client.DescribeEnvironmentManagedActions(ctx, &awseb.DescribeEnvironmentManagedActionsInput{
		EnvironmentName: aws.String("sdk-stage4-env"),
		Status:          awsebtypes.ActionStatusScheduled,
	})
	if err != nil {
		t.Fatalf("describe managed actions: %v", err)
	}
	if len(managedOut.ManagedActions) == 0 {
		t.Fatalf("expected at least one managed action")
	}
	actionID := aws.ToString(managedOut.ManagedActions[0].ActionId)
	if actionID == "" {
		t.Fatalf("expected managed action id")
	}

	if _, err := client.ApplyEnvironmentManagedAction(ctx, &awseb.ApplyEnvironmentManagedActionInput{
		EnvironmentName: aws.String("sdk-stage4-env"),
		ActionId:        aws.String(actionID),
	}); err != nil {
		t.Fatalf("apply managed action: %v", err)
	}

	historyOut, err := client.DescribeEnvironmentManagedActionHistory(ctx, &awseb.DescribeEnvironmentManagedActionHistoryInput{
		EnvironmentName: aws.String("sdk-stage4-env"),
		MaxItems:        aws.Int32(10),
	})
	if err != nil {
		t.Fatalf("describe managed action history: %v", err)
	}
	if len(historyOut.ManagedActionHistoryItems) == 0 {
		t.Fatalf("expected managed action history entries")
	}

	updateOut, err := client.UpdateApplicationResourceLifecycle(ctx, &awseb.UpdateApplicationResourceLifecycleInput{
		ApplicationName: aws.String("sdk-stage4-app"),
		ResourceLifecycleConfig: &awsebtypes.ApplicationResourceLifecycleConfig{
			ServiceRole: aws.String("arn:aws:iam::123456789012:role/eb-service-role"),
			VersionLifecycleConfig: &awsebtypes.ApplicationVersionLifecycleConfig{
				MaxAgeRule: &awsebtypes.MaxAgeRule{
					Enabled:            aws.Bool(true),
					DeleteSourceFromS3: aws.Bool(true),
					MaxAgeInDays:       aws.Int32(14),
				},
				MaxCountRule: &awsebtypes.MaxCountRule{
					Enabled:            aws.Bool(true),
					DeleteSourceFromS3: aws.Bool(false),
					MaxCount:           aws.Int32(25),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("update application resource lifecycle: %v", err)
	}
	if updateOut.ResourceLifecycleConfig == nil {
		t.Fatalf("expected lifecycle config in sdk response")
	}
	if aws.ToString(updateOut.ApplicationName) != "sdk-stage4-app" {
		t.Fatalf("unexpected sdk application name")
	}
}
