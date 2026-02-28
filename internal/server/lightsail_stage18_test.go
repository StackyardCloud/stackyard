package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage18ContainersDeployImagesLogs(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateContainerService", []byte(`{"serviceName":"stage18-service","power":"nano","scale":1}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "RegisterContainerImage", []byte(`{"serviceName":"stage18-service","label":"web","digest":"sha256:stage18"}`))
	assertStatus(t, resp, http.StatusOK)
	var registerOut struct {
		ContainerImage struct {
			Image string `json:"image"`
		} `json:"containerImage"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerOut); err != nil {
		t.Fatalf("unmarshal RegisterContainerImage: %v", err)
	}
	if registerOut.ContainerImage.Image == "" {
		t.Fatalf("expected registered image name")
	}

	resp = lightsailRequest(t, ts, "GetContainerImages", []byte(`{"serviceName":"stage18-service"}`))
	assertStatus(t, resp, http.StatusOK)
	var imagesOut struct {
		ContainerImages []struct {
			Image string `json:"image"`
		} `json:"containerImages"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &imagesOut); err != nil {
		t.Fatalf("unmarshal GetContainerImages: %v", err)
	}
	if len(imagesOut.ContainerImages) != 1 || imagesOut.ContainerImages[0].Image == "" {
		t.Fatalf("unexpected GetContainerImages output: %+v", imagesOut)
	}

	resp = lightsailRequest(t, ts, "CreateContainerServiceDeployment", []byte(`{"serviceName":"stage18-service","containers":{"web":{"image":"`+registerOut.ContainerImage.Image+`","ports":{"80":"HTTP"}}},"publicEndpoint":{"containerName":"web","containerPort":80,"healthCheck":{"healthyThreshold":2,"intervalSeconds":10}}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetContainerServiceDeployments", []byte(`{"serviceName":"stage18-service"}`))
	assertStatus(t, resp, http.StatusOK)
	var deploymentsOut struct {
		Deployments []struct {
			Version int32  `json:"version"`
			State   string `json:"state"`
		} `json:"deployments"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &deploymentsOut); err != nil {
		t.Fatalf("unmarshal GetContainerServiceDeployments: %v", err)
	}
	if len(deploymentsOut.Deployments) != 1 || deploymentsOut.Deployments[0].Version != 1 || deploymentsOut.Deployments[0].State != "ACTIVE" {
		t.Fatalf("unexpected deployments output: %+v", deploymentsOut)
	}

	resp = lightsailRequest(t, ts, "GetContainerLog", []byte(`{"serviceName":"stage18-service","containerName":"web","filterPattern":"activated"}`))
	assertStatus(t, resp, http.StatusOK)
	var logsOut struct {
		LogEvents []struct {
			Message string `json:"message"`
		} `json:"logEvents"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &logsOut); err != nil {
		t.Fatalf("unmarshal GetContainerLog: %v", err)
	}
	if len(logsOut.LogEvents) == 0 {
		t.Fatalf("expected container log events")
	}

	resp = lightsailRequest(t, ts, "DeleteContainerImage", []byte(`{"serviceName":"stage18-service","image":"`+registerOut.ContainerImage.Image+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteContainerService", []byte(`{"serviceName":"stage18-service"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage18SDKClientContainersDeployImagesLogs(t *testing.T) {
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

	if _, err := client.CreateContainerService(ctx, &awslightsail.CreateContainerServiceInput{
		ServiceName: aws.String("sdk-stage18-service"),
		Power:       awslightsailtypes.ContainerServicePowerNameNano,
		Scale:       aws.Int32(1),
	}); err != nil {
		t.Fatalf("create container service: %v", err)
	}

	registerOut, err := client.RegisterContainerImage(ctx, &awslightsail.RegisterContainerImageInput{
		ServiceName: aws.String("sdk-stage18-service"),
		Label:       aws.String("web"),
		Digest:      aws.String("sha256:sdk-stage18"),
	})
	if err != nil {
		t.Fatalf("register container image: %v", err)
	}
	if registerOut.ContainerImage == nil || registerOut.ContainerImage.Image == nil || *registerOut.ContainerImage.Image == "" {
		t.Fatalf("unexpected register container image output: %+v", registerOut.ContainerImage)
	}

	imagesOut, err := client.GetContainerImages(ctx, &awslightsail.GetContainerImagesInput{
		ServiceName: aws.String("sdk-stage18-service"),
	})
	if err != nil {
		t.Fatalf("get container images: %v", err)
	}
	if len(imagesOut.ContainerImages) != 1 {
		t.Fatalf("expected one container image, got %d", len(imagesOut.ContainerImages))
	}

	createDeploymentOut, err := client.CreateContainerServiceDeployment(ctx, &awslightsail.CreateContainerServiceDeploymentInput{
		ServiceName: aws.String("sdk-stage18-service"),
		Containers: map[string]awslightsailtypes.Container{
			"web": {
				Image: aws.String(*registerOut.ContainerImage.Image),
				Ports: map[string]awslightsailtypes.ContainerServiceProtocol{"80": awslightsailtypes.ContainerServiceProtocolHttp},
			},
		},
		PublicEndpoint: &awslightsailtypes.EndpointRequest{
			ContainerName: aws.String("web"),
			ContainerPort: aws.Int32(80),
		},
	})
	if err != nil {
		t.Fatalf("create container service deployment: %v", err)
	}
	if createDeploymentOut.ContainerService == nil || createDeploymentOut.ContainerService.ContainerServiceName == nil || *createDeploymentOut.ContainerService.ContainerServiceName != "sdk-stage18-service" {
		t.Fatalf("unexpected create container service deployment output: %+v", createDeploymentOut.ContainerService)
	}

	deploymentsOut, err := client.GetContainerServiceDeployments(ctx, &awslightsail.GetContainerServiceDeploymentsInput{
		ServiceName: aws.String("sdk-stage18-service"),
	})
	if err != nil {
		t.Fatalf("get container service deployments: %v", err)
	}
	if len(deploymentsOut.Deployments) != 1 {
		t.Fatalf("expected one deployment, got %d", len(deploymentsOut.Deployments))
	}

	logOut, err := client.GetContainerLog(ctx, &awslightsail.GetContainerLogInput{
		ServiceName:   aws.String("sdk-stage18-service"),
		ContainerName: aws.String("web"),
		FilterPattern: aws.String("activated"),
	})
	if err != nil {
		t.Fatalf("get container log: %v", err)
	}
	if len(logOut.LogEvents) == 0 {
		t.Fatalf("expected log events")
	}

	if _, err := client.DeleteContainerImage(ctx, &awslightsail.DeleteContainerImageInput{
		ServiceName: aws.String("sdk-stage18-service"),
		Image:       registerOut.ContainerImage.Image,
	}); err != nil {
		t.Fatalf("delete container image: %v", err)
	}

	if _, err := client.DeleteContainerService(ctx, &awslightsail.DeleteContainerServiceInput{
		ServiceName: aws.String("sdk-stage18-service"),
	}); err != nil {
		t.Fatalf("delete container service: %v", err)
	}
}
