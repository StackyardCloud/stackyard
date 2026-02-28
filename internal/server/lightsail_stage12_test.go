package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage12Distributions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateDistribution", []byte(`{
		"distributionName":"stage12-dist",
		"bundleId":"small_1_0",
		"defaultCacheBehavior":{"behavior":"cache"},
		"origin":{"name":"stage12-origin","protocolPolicy":"http-only","regionName":"us-east-1"},
		"cacheBehaviors":[{"behavior":"dont-cache","path":"/api/*"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetDistributions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var getDistributionsOut struct {
		Distributions []struct {
			Name string `json:"name"`
		} `json:"distributions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getDistributionsOut); err != nil {
		t.Fatalf("unmarshal GetDistributions: %v", err)
	}
	if len(getDistributionsOut.Distributions) != 1 || getDistributionsOut.Distributions[0].Name != "stage12-dist" {
		t.Fatalf("unexpected GetDistributions output: %+v", getDistributionsOut)
	}

	resp = lightsailRequest(t, ts, "UpdateDistribution", []byte(`{"distributionName":"stage12-dist","isEnabled":false,"certificateName":"stage12-cert"}`))
	assertStatus(t, resp, http.StatusOK)

	start := float64(time.Now().UTC().Add(-5 * time.Minute).Unix())
	end := float64(time.Now().UTC().Unix())
	resp = lightsailRequest(t, ts, "GetDistributionMetricData", []byte(fmt.Sprintf(`{
		"distributionName":"stage12-dist",
		"startTime":%.0f,
		"endTime":%.0f,
		"period":60,
		"metricName":"Requests",
		"statistics":["Average","Sum"],
		"unit":"Count"
	}`, start, end)))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "ResetDistributionCache", []byte(`{"distributionName":"stage12-dist"}`))
	assertStatus(t, resp, http.StatusOK)
	var resetOut struct {
		Status string  `json:"status"`
		Time   float64 `json:"createTime"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &resetOut); err != nil {
		t.Fatalf("unmarshal ResetDistributionCache: %v", err)
	}
	if resetOut.Status == "" || resetOut.Time == 0 {
		t.Fatalf("unexpected ResetDistributionCache output: %+v", resetOut)
	}

	resp = lightsailRequest(t, ts, "GetDistributionLatestCacheReset", []byte(`{"distributionName":"stage12-dist"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetDistributionBundles", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var bundlesOut struct {
		Bundles []struct {
			BundleID string `json:"bundleId"`
		} `json:"bundles"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &bundlesOut); err != nil {
		t.Fatalf("unmarshal GetDistributionBundles: %v", err)
	}
	if len(bundlesOut.Bundles) == 0 {
		t.Fatalf("expected distribution bundles")
	}

	resp = lightsailRequest(t, ts, "UpdateDistributionBundle", []byte(`{"distributionName":"stage12-dist","bundleId":"medium_1_0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteDistribution", []byte(`{"distributionName":"stage12-dist"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage12SDKClientDistributions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := awslightsail.NewFromConfig(cfg, func(o *awslightsail.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	createOut, err := client.CreateDistribution(ctx, &awslightsail.CreateDistributionInput{
		DistributionName:     aws.String("sdk-stage12-dist"),
		BundleId:             aws.String("small_1_0"),
		DefaultCacheBehavior: &awslightsailtypes.CacheBehavior{Behavior: awslightsailtypes.BehaviorEnum("cache")},
		Origin: &awslightsailtypes.InputOrigin{
			Name:           aws.String("sdk-stage12-origin"),
			ProtocolPolicy: awslightsailtypes.OriginProtocolPolicyEnum("http-only"),
			RegionName:     awslightsailtypes.RegionName("us-east-1"),
		},
		CacheBehaviors: []awslightsailtypes.CacheBehaviorPerPath{
			{Behavior: awslightsailtypes.BehaviorEnum("dont-cache"), Path: aws.String("/api/*")},
		},
	})
	if err != nil {
		t.Fatalf("create distribution: %v", err)
	}
	if createOut.Distribution == nil || createOut.Distribution.Name == nil || *createOut.Distribution.Name != "sdk-stage12-dist" {
		t.Fatalf("unexpected create distribution output: %+v", createOut.Distribution)
	}

	distributionsOut, err := client.GetDistributions(ctx, &awslightsail.GetDistributionsInput{
		DistributionName: aws.String("sdk-stage12-dist"),
	})
	if err != nil {
		t.Fatalf("get distributions: %v", err)
	}
	if len(distributionsOut.Distributions) != 1 {
		t.Fatalf("expected one distribution, got %d", len(distributionsOut.Distributions))
	}

	if _, err := client.UpdateDistribution(ctx, &awslightsail.UpdateDistributionInput{
		DistributionName: aws.String("sdk-stage12-dist"),
		IsEnabled:        aws.Bool(false),
		CertificateName:  aws.String("sdk-stage12-cert"),
	}); err != nil {
		t.Fatalf("update distribution: %v", err)
	}

	if _, err := client.ResetDistributionCache(ctx, &awslightsail.ResetDistributionCacheInput{
		DistributionName: aws.String("sdk-stage12-dist"),
	}); err != nil {
		t.Fatalf("reset distribution cache: %v", err)
	}

	if _, err := client.GetDistributionLatestCacheReset(ctx, &awslightsail.GetDistributionLatestCacheResetInput{
		DistributionName: aws.String("sdk-stage12-dist"),
	}); err != nil {
		t.Fatalf("get distribution latest cache reset: %v", err)
	}

	metricOut, err := client.GetDistributionMetricData(ctx, &awslightsail.GetDistributionMetricDataInput{
		DistributionName: aws.String("sdk-stage12-dist"),
		StartTime:        aws.Time(time.Now().UTC().Add(-5 * time.Minute)),
		EndTime:          aws.Time(time.Now().UTC()),
		Period:           aws.Int32(60),
		MetricName:       awslightsailtypes.DistributionMetricName("Requests"),
		Statistics: []awslightsailtypes.MetricStatistic{
			awslightsailtypes.MetricStatistic("Average"),
			awslightsailtypes.MetricStatistic("Sum"),
		},
		Unit: awslightsailtypes.MetricUnit("Count"),
	})
	if err != nil {
		t.Fatalf("get distribution metric data: %v", err)
	}
	if len(metricOut.MetricData) == 0 {
		t.Fatalf("expected metric datapoints")
	}

	bundlesOut, err := client.GetDistributionBundles(ctx, &awslightsail.GetDistributionBundlesInput{})
	if err != nil {
		t.Fatalf("get distribution bundles: %v", err)
	}
	if len(bundlesOut.Bundles) == 0 {
		t.Fatalf("expected bundles")
	}

	if _, err := client.UpdateDistributionBundle(ctx, &awslightsail.UpdateDistributionBundleInput{
		DistributionName: aws.String("sdk-stage12-dist"),
		BundleId:         aws.String("medium_1_0"),
	}); err != nil {
		t.Fatalf("update distribution bundle: %v", err)
	}

	if _, err := client.DeleteDistribution(ctx, &awslightsail.DeleteDistributionInput{
		DistributionName: aws.String("sdk-stage12-dist"),
	}); err != nil {
		t.Fatalf("delete distribution: %v", err)
	}
}
