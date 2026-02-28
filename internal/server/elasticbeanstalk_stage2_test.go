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

func TestElasticBeanstalkStage2PlatformLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createPlatform := url.Values{}
	createPlatform.Set("Action", "CreatePlatformVersion")
	createPlatform.Set("Version", "2010-12-01")
	createPlatform.Set("PlatformName", "demo-platform")
	createPlatform.Set("PlatformVersion", "1.0.0")
	createPlatform.Set("PlatformDefinitionBundle.S3Bucket", "bundle-bucket")
	createPlatform.Set("PlatformDefinitionBundle.S3Key", "platform.zip")
	resp := elasticBeanstalkRequest(t, ts, createPlatform)
	assertStatus(t, resp, http.StatusOK)
	var createResp struct {
		Result elasticBeanstalkCreatePlatformVersionResult `xml:"CreatePlatformVersionResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("unmarshal create platform version: %v", err)
	}
	platformArn := aws.ToString(createResp.Result.PlatformSummary.PlatformArn)
	if platformArn == "" {
		t.Fatalf("expected platform arn in create response")
	}

	describePlatform := url.Values{}
	describePlatform.Set("Action", "DescribePlatformVersion")
	describePlatform.Set("PlatformArn", platformArn)
	resp = elasticBeanstalkRequest(t, ts, describePlatform)
	assertStatus(t, resp, http.StatusOK)
	var describeResp struct {
		Result elasticBeanstalkDescribePlatformVersionResult `xml:"DescribePlatformVersionResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeResp); err != nil {
		t.Fatalf("unmarshal describe platform version: %v", err)
	}
	if describeResp.Result.PlatformDescription == nil || aws.ToString(describeResp.Result.PlatformDescription.PlatformName) != "demo-platform" {
		t.Fatalf("unexpected describe platform response: %+v", describeResp.Result.PlatformDescription)
	}

	listPlatforms := url.Values{}
	listPlatforms.Set("Action", "ListPlatformVersions")
	listPlatforms.Set("Filters.member.1.Type", "PlatformName")
	listPlatforms.Set("Filters.member.1.Operator", "=")
	listPlatforms.Set("Filters.member.1.Values.member.1", "demo-platform")
	resp = elasticBeanstalkRequest(t, ts, listPlatforms)
	assertStatus(t, resp, http.StatusOK)
	var listResp struct {
		Result elasticBeanstalkListPlatformVersionsResult `xml:"ListPlatformVersionsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("unmarshal list platform versions: %v", err)
	}
	if len(listResp.Result.PlatformSummaryList) != 1 {
		t.Fatalf("expected one platform summary, got %d", len(listResp.Result.PlatformSummaryList))
	}

	deletePlatform := url.Values{}
	deletePlatform.Set("Action", "DeletePlatformVersion")
	deletePlatform.Set("PlatformArn", platformArn)
	resp = elasticBeanstalkRequest(t, ts, deletePlatform)
	assertStatus(t, resp, http.StatusOK)
	var deleteResp struct {
		Result elasticBeanstalkDeletePlatformVersionResult `xml:"DeletePlatformVersionResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &deleteResp); err != nil {
		t.Fatalf("unmarshal delete platform version: %v", err)
	}
	if deleteResp.Result.PlatformSummary.PlatformStatus != awsebtypes.PlatformStatusDeleted {
		t.Fatalf("expected deleted platform status, got %q", deleteResp.Result.PlatformSummary.PlatformStatus)
	}
}

func TestElasticBeanstalkStage2SDKClientPlatformLifecycle(t *testing.T) {
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

	createOut, err := client.CreatePlatformVersion(ctx, &awseb.CreatePlatformVersionInput{
		PlatformName:    aws.String("sdk-demo-platform"),
		PlatformVersion: aws.String("1.0.0"),
		PlatformDefinitionBundle: &awsebtypes.S3Location{
			S3Bucket: aws.String("bundle-bucket"),
			S3Key:    aws.String("platform.zip"),
		},
	})
	if err != nil {
		t.Fatalf("create platform version: %v", err)
	}
	platformArn := aws.ToString(createOut.PlatformSummary.PlatformArn)
	if platformArn == "" {
		t.Fatalf("expected platform arn from sdk create")
	}

	describeOut, err := client.DescribePlatformVersion(ctx, &awseb.DescribePlatformVersionInput{
		PlatformArn: aws.String(platformArn),
	})
	if err != nil {
		t.Fatalf("describe platform version: %v", err)
	}
	if describeOut.PlatformDescription == nil || aws.ToString(describeOut.PlatformDescription.PlatformVersion) != "1.0.0" {
		t.Fatalf("unexpected describe output")
	}

	listOut, err := client.ListPlatformVersions(ctx, &awseb.ListPlatformVersionsInput{
		Filters: []awsebtypes.PlatformFilter{{
			Type:     aws.String("PlatformName"),
			Operator: aws.String("="),
			Values:   []string{"sdk-demo-platform"},
		}},
	})
	if err != nil {
		t.Fatalf("list platform versions: %v", err)
	}
	if len(listOut.PlatformSummaryList) == 0 {
		t.Fatalf("expected platform list items")
	}

	deleteOut, err := client.DeletePlatformVersion(ctx, &awseb.DeletePlatformVersionInput{
		PlatformArn: aws.String(platformArn),
	})
	if err != nil {
		t.Fatalf("delete platform version: %v", err)
	}
	if deleteOut.PlatformSummary == nil || deleteOut.PlatformSummary.PlatformStatus != awsebtypes.PlatformStatusDeleted {
		t.Fatalf("unexpected delete output")
	}
}
