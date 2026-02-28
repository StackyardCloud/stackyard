package server

import (
	"context"
	"encoding/json"
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

func TestLightsailStage15BucketsCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage15-instance"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateBucket", []byte(`{"bucketName":"stage15-bucket","bundleId":"small_1_0","enableObjectVersioning":true,"tags":[{"key":"env","value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetBuckets", []byte(`{"bucketName":"stage15-bucket"}`))
	assertStatus(t, resp, http.StatusOK)
	var getBucketsOut struct {
		Buckets []struct {
			Name string `json:"name"`
		} `json:"buckets"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getBucketsOut); err != nil {
		t.Fatalf("unmarshal GetBuckets: %v", err)
	}
	if len(getBucketsOut.Buckets) != 1 || getBucketsOut.Buckets[0].Name != "stage15-bucket" {
		t.Fatalf("unexpected GetBuckets output: %+v", getBucketsOut)
	}

	resp = lightsailRequest(t, ts, "SetResourceAccessForBucket", []byte(`{"bucketName":"stage15-bucket","resourceName":"stage15-instance","access":"allow"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetBuckets", []byte(`{"bucketName":"stage15-bucket","includeConnectedResources":true}`))
	assertStatus(t, resp, http.StatusOK)
	var getBucketsWithAccessOut struct {
		Buckets []struct {
			ResourcesReceivingAccess []struct {
				Name string `json:"name"`
			} `json:"resourcesReceivingAccess"`
		} `json:"buckets"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getBucketsWithAccessOut); err != nil {
		t.Fatalf("unmarshal GetBuckets includeConnectedResources: %v", err)
	}
	if len(getBucketsWithAccessOut.Buckets) != 1 || len(getBucketsWithAccessOut.Buckets[0].ResourcesReceivingAccess) != 1 {
		t.Fatalf("unexpected connected resources output: %+v", getBucketsWithAccessOut)
	}

	resp = lightsailRequest(t, ts, "UpdateBucket", []byte(`{"bucketName":"stage15-bucket","versioning":"Suspended","accessRules":{"allowPublicOverrides":true,"getObject":"public"},"readonlyAccessAccounts":["111122223333","222233334444"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetBucketBundles", []byte(`{"includeInactive":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetBucketMetricData", []byte(`{"bucketName":"stage15-bucket","metricName":"BucketSizeBytes","startTime":"2026-02-01T00:00:00Z","endTime":"2026-02-02T00:00:00Z","period":86400,"statistics":["Maximum"],"unit":"Bytes"}`))
	assertStatus(t, resp, http.StatusOK)
	var metricOut struct {
		MetricName string `json:"metricName"`
		MetricData []struct {
			Timestamp float64 `json:"timestamp"`
		} `json:"metricData"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &metricOut); err != nil {
		t.Fatalf("unmarshal GetBucketMetricData: %v", err)
	}
	if metricOut.MetricName != "BucketSizeBytes" || len(metricOut.MetricData) == 0 {
		t.Fatalf("unexpected metric output: %+v", metricOut)
	}

	resp = lightsailRequest(t, ts, "UpdateBucketBundle", []byte(`{"bucketName":"stage15-bucket","bundleId":"medium_1_0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "SetResourceAccessForBucket", []byte(`{"bucketName":"stage15-bucket","resourceName":"stage15-instance","access":"deny"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteBucket", []byte(`{"bucketName":"stage15-bucket"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage15SDKClientBucketsCore(t *testing.T) {
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

	if _, err := client.CreateInstances(ctx, &awslightsail.CreateInstancesInput{
		AvailabilityZone: aws.String("us-east-1a"),
		BlueprintId:      aws.String("amazon_linux_2"),
		BundleId:         aws.String("micro_2_0"),
		InstanceNames:    []string{"sdk-stage15-instance"},
	}); err != nil {
		t.Fatalf("create instances: %v", err)
	}

	createOut, err := client.CreateBucket(ctx, &awslightsail.CreateBucketInput{
		BucketName:             aws.String("sdk-stage15-bucket"),
		BundleId:               aws.String("small_1_0"),
		EnableObjectVersioning: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	if createOut.Bucket == nil || createOut.Bucket.Name == nil || *createOut.Bucket.Name != "sdk-stage15-bucket" {
		t.Fatalf("unexpected create bucket output: %+v", createOut.Bucket)
	}

	getBucketsOut, err := client.GetBuckets(ctx, &awslightsail.GetBucketsInput{
		BucketName: aws.String("sdk-stage15-bucket"),
	})
	if err != nil {
		t.Fatalf("get buckets: %v", err)
	}
	if len(getBucketsOut.Buckets) != 1 {
		t.Fatalf("expected one bucket, got %d", len(getBucketsOut.Buckets))
	}

	if _, err := client.SetResourceAccessForBucket(ctx, &awslightsail.SetResourceAccessForBucketInput{
		BucketName:   aws.String("sdk-stage15-bucket"),
		ResourceName: aws.String("sdk-stage15-instance"),
		Access:       awslightsailtypes.ResourceBucketAccessAllow,
	}); err != nil {
		t.Fatalf("set resource access allow: %v", err)
	}

	updateOut, err := client.UpdateBucket(ctx, &awslightsail.UpdateBucketInput{
		BucketName: aws.String("sdk-stage15-bucket"),
		AccessRules: &awslightsailtypes.AccessRules{
			AllowPublicOverrides: aws.Bool(true),
			GetObject:            awslightsailtypes.AccessTypePublic,
		},
		ReadonlyAccessAccounts: []string{"111122223333", "222233334444"},
		Versioning:             aws.String("Suspended"),
	})
	if err != nil {
		t.Fatalf("update bucket: %v", err)
	}
	if updateOut.Bucket == nil || updateOut.Bucket.ObjectVersioning == nil || *updateOut.Bucket.ObjectVersioning != "Suspended" {
		t.Fatalf("unexpected update bucket output: %+v", updateOut.Bucket)
	}

	bundlesOut, err := client.GetBucketBundles(ctx, &awslightsail.GetBucketBundlesInput{
		IncludeInactive: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("get bucket bundles: %v", err)
	}
	if len(bundlesOut.Bundles) == 0 {
		t.Fatalf("expected bucket bundles")
	}

	startTime := time.Now().UTC().Add(-24 * time.Hour)
	endTime := time.Now().UTC()
	metricsOut, err := client.GetBucketMetricData(ctx, &awslightsail.GetBucketMetricDataInput{
		BucketName: aws.String("sdk-stage15-bucket"),
		MetricName: awslightsailtypes.BucketMetricNameBucketSizeBytes,
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int32(86400),
		Statistics: []awslightsailtypes.MetricStatistic{awslightsailtypes.MetricStatisticMaximum},
		Unit:       awslightsailtypes.MetricUnitBytes,
	})
	if err != nil {
		t.Fatalf("get bucket metric data: %v", err)
	}
	if len(metricsOut.MetricData) == 0 {
		t.Fatalf("expected metric datapoints")
	}

	if _, err := client.UpdateBucketBundle(ctx, &awslightsail.UpdateBucketBundleInput{
		BucketName: aws.String("sdk-stage15-bucket"),
		BundleId:   aws.String("medium_1_0"),
	}); err != nil {
		t.Fatalf("update bucket bundle: %v", err)
	}

	if _, err := client.SetResourceAccessForBucket(ctx, &awslightsail.SetResourceAccessForBucketInput{
		BucketName:   aws.String("sdk-stage15-bucket"),
		ResourceName: aws.String("sdk-stage15-instance"),
		Access:       awslightsailtypes.ResourceBucketAccessDeny,
	}); err != nil {
		t.Fatalf("set resource access deny: %v", err)
	}

	if _, err := client.DeleteBucket(ctx, &awslightsail.DeleteBucketInput{
		BucketName: aws.String("sdk-stage15-bucket"),
	}); err != nil {
		t.Fatalf("delete bucket: %v", err)
	}
}
