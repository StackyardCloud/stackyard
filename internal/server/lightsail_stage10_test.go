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

func TestLightsailStage10LoadBalancerCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateInstances", []byte(`{"availabilityZone":"us-east-1a","blueprintId":"amazon_linux_2","bundleId":"micro_2_0","instanceNames":["stage10-instance-1","stage10-instance-2"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateLoadBalancer", []byte(`{"loadBalancerName":"stage10-lb","instancePort":80,"healthCheckPath":"/health","ipAddressType":"dualstack","tlsPolicyName":"TLS-1-2-2018-06"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "AttachInstancesToLoadBalancer", []byte(`{"loadBalancerName":"stage10-lb","instanceNames":["stage10-instance-1","stage10-instance-2"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetLoadBalancer", []byte(`{"loadBalancerName":"stage10-lb"}`))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		LoadBalancer struct {
			Name                  string `json:"name"`
			HTTPSRedirection      bool   `json:"httpsRedirectionEnabled"`
			InstanceHealthSummary []any  `json:"instanceHealthSummary"`
		} `json:"loadBalancer"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal GetLoadBalancer: %v", err)
	}
	if getOut.LoadBalancer.Name != "stage10-lb" {
		t.Fatalf("unexpected load balancer name: %+v", getOut.LoadBalancer)
	}
	if len(getOut.LoadBalancer.InstanceHealthSummary) != 2 {
		t.Fatalf("expected 2 instance health summaries, got %d", len(getOut.LoadBalancer.InstanceHealthSummary))
	}
	if getOut.LoadBalancer.HTTPSRedirection {
		t.Fatalf("expected https redirection disabled by default")
	}

	resp = lightsailRequest(t, ts, "UpdateLoadBalancerAttribute", []byte(`{"loadBalancerName":"stage10-lb","attributeName":"HttpsRedirectionEnabled","attributeValue":"true"}`))
	assertStatus(t, resp, http.StatusOK)

	start := float64(time.Now().UTC().Add(-5 * time.Minute).Unix())
	end := float64(time.Now().UTC().Unix())
	resp = lightsailRequest(t, ts, "GetLoadBalancerMetricData", []byte(fmt.Sprintf(`{
		"loadBalancerName":"stage10-lb",
		"startTime":%.0f,
		"endTime":%.0f,
		"period":60,
		"metricName":"RequestCount",
		"statistics":["Average","Sum"],
		"unit":"Count"
	}`, start, end)))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetLoadBalancers", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		LoadBalancers []struct {
			Name string `json:"name"`
		} `json:"loadBalancers"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal GetLoadBalancers: %v", err)
	}
	if len(listOut.LoadBalancers) != 1 || listOut.LoadBalancers[0].Name != "stage10-lb" {
		t.Fatalf("unexpected GetLoadBalancers output: %+v", listOut)
	}

	resp = lightsailRequest(t, ts, "DetachInstancesFromLoadBalancer", []byte(`{"loadBalancerName":"stage10-lb","instanceNames":["stage10-instance-2"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteLoadBalancer", []byte(`{"loadBalancerName":"stage10-lb"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage10SDKClientLoadBalancerCore(t *testing.T) {
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
		InstanceNames:    []string{"sdk-stage10-instance-1", "sdk-stage10-instance-2"},
	}); err != nil {
		t.Fatalf("create instances: %v", err)
	}

	if _, err := client.CreateLoadBalancer(ctx, &awslightsail.CreateLoadBalancerInput{
		LoadBalancerName: aws.String("sdk-stage10-lb"),
		InstancePort:     80,
		HealthCheckPath:  aws.String("/health"),
		IpAddressType:    awslightsailtypes.IpAddressTypeDualstack,
		TlsPolicyName:    aws.String("TLS-1-2-2018-06"),
	}); err != nil {
		t.Fatalf("create load balancer: %v", err)
	}

	if _, err := client.AttachInstancesToLoadBalancer(ctx, &awslightsail.AttachInstancesToLoadBalancerInput{
		LoadBalancerName: aws.String("sdk-stage10-lb"),
		InstanceNames:    []string{"sdk-stage10-instance-1", "sdk-stage10-instance-2"},
	}); err != nil {
		t.Fatalf("attach instances to load balancer: %v", err)
	}

	getOut, err := client.GetLoadBalancer(ctx, &awslightsail.GetLoadBalancerInput{
		LoadBalancerName: aws.String("sdk-stage10-lb"),
	})
	if err != nil {
		t.Fatalf("get load balancer: %v", err)
	}
	if getOut.LoadBalancer == nil || getOut.LoadBalancer.Name == nil || *getOut.LoadBalancer.Name != "sdk-stage10-lb" {
		t.Fatalf("unexpected get load balancer output: %+v", getOut.LoadBalancer)
	}

	metricOut, err := client.GetLoadBalancerMetricData(ctx, &awslightsail.GetLoadBalancerMetricDataInput{
		LoadBalancerName: aws.String("sdk-stage10-lb"),
		StartTime:        aws.Time(time.Now().UTC().Add(-5 * time.Minute)),
		EndTime:          aws.Time(time.Now().UTC()),
		Period:           aws.Int32(60),
		MetricName:       awslightsailtypes.LoadBalancerMetricName("RequestCount"),
		Statistics: []awslightsailtypes.MetricStatistic{
			awslightsailtypes.MetricStatistic("Average"),
			awslightsailtypes.MetricStatistic("Sum"),
		},
		Unit: awslightsailtypes.MetricUnit("Count"),
	})
	if err != nil {
		t.Fatalf("get load balancer metric data: %v", err)
	}
	if len(metricOut.MetricData) == 0 {
		t.Fatalf("expected metric datapoints")
	}

	if _, err := client.UpdateLoadBalancerAttribute(ctx, &awslightsail.UpdateLoadBalancerAttributeInput{
		LoadBalancerName: aws.String("sdk-stage10-lb"),
		AttributeName:    awslightsailtypes.LoadBalancerAttributeNameHttpsRedirectionEnabled,
		AttributeValue:   aws.String("true"),
	}); err != nil {
		t.Fatalf("update load balancer attribute: %v", err)
	}

	listOut, err := client.GetLoadBalancers(ctx, &awslightsail.GetLoadBalancersInput{})
	if err != nil {
		t.Fatalf("get load balancers: %v", err)
	}
	if len(listOut.LoadBalancers) == 0 {
		t.Fatalf("expected at least one load balancer")
	}

	if _, err := client.DetachInstancesFromLoadBalancer(ctx, &awslightsail.DetachInstancesFromLoadBalancerInput{
		LoadBalancerName: aws.String("sdk-stage10-lb"),
		InstanceNames:    []string{"sdk-stage10-instance-2"},
	}); err != nil {
		t.Fatalf("detach instances from load balancer: %v", err)
	}

	if _, err := client.DeleteLoadBalancer(ctx, &awslightsail.DeleteLoadBalancerInput{
		LoadBalancerName: aws.String("sdk-stage10-lb"),
	}); err != nil {
		t.Fatalf("delete load balancer: %v", err)
	}
}
