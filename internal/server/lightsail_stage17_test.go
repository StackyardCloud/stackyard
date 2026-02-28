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

func TestLightsailStage17ContainerServicesCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateContainerService", []byte(`{"serviceName":"stage17-service","power":"nano","scale":1}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetContainerServices", []byte(`{"serviceName":"stage17-service"}`))
	assertStatus(t, resp, http.StatusOK)
	var getServicesOut struct {
		ContainerServices []struct {
			ContainerServiceName string `json:"containerServiceName"`
		} `json:"containerServices"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getServicesOut); err != nil {
		t.Fatalf("unmarshal GetContainerServices: %v", err)
	}
	if len(getServicesOut.ContainerServices) != 1 || getServicesOut.ContainerServices[0].ContainerServiceName != "stage17-service" {
		t.Fatalf("unexpected GetContainerServices output: %+v", getServicesOut)
	}

	resp = lightsailRequest(t, ts, "UpdateContainerService", []byte(`{"serviceName":"stage17-service","power":"micro","scale":2,"isDisabled":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetContainerAPIMetadata", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	start := time.Now().UTC().Add(-15 * time.Minute).Format(time.RFC3339)
	end := time.Now().UTC().Format(time.RFC3339)
	resp = lightsailRequest(t, ts, "GetContainerServiceMetricData", []byte(`{"serviceName":"stage17-service","metricName":"CPUUtilization","startTime":"`+start+`","endTime":"`+end+`","period":300,"statistics":["Average","Maximum"]}`))
	assertStatus(t, resp, http.StatusOK)
	var metricOut struct {
		MetricName string `json:"metricName"`
		MetricData []struct {
			Timestamp float64 `json:"timestamp"`
		} `json:"metricData"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &metricOut); err != nil {
		t.Fatalf("unmarshal GetContainerServiceMetricData: %v", err)
	}
	if metricOut.MetricName != "CPUUtilization" || len(metricOut.MetricData) == 0 {
		t.Fatalf("unexpected metric output: %+v", metricOut)
	}

	resp = lightsailRequest(t, ts, "GetContainerServicePowers", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateContainerServiceRegistryLogin", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var loginOut struct {
		RegistryLogin struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Registry string `json:"registry"`
		} `json:"registryLogin"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &loginOut); err != nil {
		t.Fatalf("unmarshal CreateContainerServiceRegistryLogin: %v", err)
	}
	if loginOut.RegistryLogin.Username == "" || loginOut.RegistryLogin.Password == "" || loginOut.RegistryLogin.Registry == "" {
		t.Fatalf("unexpected registry login output: %+v", loginOut)
	}

	resp = lightsailRequest(t, ts, "DeleteContainerService", []byte(`{"serviceName":"stage17-service"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage17SDKClientContainerServicesCore(t *testing.T) {
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

	createOut, err := client.CreateContainerService(ctx, &awslightsail.CreateContainerServiceInput{
		ServiceName: aws.String("sdk-stage17-service"),
		Power:       awslightsailtypes.ContainerServicePowerNameNano,
		Scale:       aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("create container service: %v", err)
	}
	if createOut.ContainerService == nil || createOut.ContainerService.ContainerServiceName == nil || *createOut.ContainerService.ContainerServiceName != "sdk-stage17-service" {
		t.Fatalf("unexpected create container service output: %+v", createOut.ContainerService)
	}

	servicesOut, err := client.GetContainerServices(ctx, &awslightsail.GetContainerServicesInput{
		ServiceName: aws.String("sdk-stage17-service"),
	})
	if err != nil {
		t.Fatalf("get container services: %v", err)
	}
	if len(servicesOut.ContainerServices) != 1 {
		t.Fatalf("expected one container service, got %d", len(servicesOut.ContainerServices))
	}

	updatedOut, err := client.UpdateContainerService(ctx, &awslightsail.UpdateContainerServiceInput{
		ServiceName: aws.String("sdk-stage17-service"),
		Power:       awslightsailtypes.ContainerServicePowerNameMicro,
		Scale:       aws.Int32(2),
		IsDisabled:  aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("update container service: %v", err)
	}
	if updatedOut.ContainerService == nil || updatedOut.ContainerService.Power != awslightsailtypes.ContainerServicePowerNameMicro {
		t.Fatalf("unexpected update container service output: %+v", updatedOut.ContainerService)
	}

	apiMetadataOut, err := client.GetContainerAPIMetadata(ctx, &awslightsail.GetContainerAPIMetadataInput{})
	if err != nil {
		t.Fatalf("get container api metadata: %v", err)
	}
	if len(apiMetadataOut.Metadata) == 0 {
		t.Fatalf("expected container api metadata")
	}

	start := time.Now().UTC().Add(-15 * time.Minute)
	end := time.Now().UTC()
	metricOut, err := client.GetContainerServiceMetricData(ctx, &awslightsail.GetContainerServiceMetricDataInput{
		ServiceName: aws.String("sdk-stage17-service"),
		MetricName:  awslightsailtypes.ContainerServiceMetricNameCPUUtilization,
		StartTime:   aws.Time(start),
		EndTime:     aws.Time(end),
		Period:      aws.Int32(300),
		Statistics:  []awslightsailtypes.MetricStatistic{awslightsailtypes.MetricStatisticAverage, awslightsailtypes.MetricStatisticMaximum},
	})
	if err != nil {
		t.Fatalf("get container service metric data: %v", err)
	}
	if len(metricOut.MetricData) == 0 {
		t.Fatalf("expected metric datapoints")
	}

	powersOut, err := client.GetContainerServicePowers(ctx, &awslightsail.GetContainerServicePowersInput{})
	if err != nil {
		t.Fatalf("get container service powers: %v", err)
	}
	if len(powersOut.Powers) == 0 {
		t.Fatalf("expected powers")
	}

	registryLoginOut, err := client.CreateContainerServiceRegistryLogin(ctx, &awslightsail.CreateContainerServiceRegistryLoginInput{})
	if err != nil {
		t.Fatalf("create container service registry login: %v", err)
	}
	if registryLoginOut.RegistryLogin == nil || registryLoginOut.RegistryLogin.Username == nil || *registryLoginOut.RegistryLogin.Username == "" {
		t.Fatalf("unexpected registry login output: %+v", registryLoginOut.RegistryLogin)
	}

	if _, err := client.DeleteContainerService(ctx, &awslightsail.DeleteContainerServiceInput{
		ServiceName: aws.String("sdk-stage17-service"),
	}); err != nil {
		t.Fatalf("delete container service: %v", err)
	}
}
