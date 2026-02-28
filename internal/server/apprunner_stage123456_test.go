package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppRunnerStage12ServiceAndConnectionLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	serviceArn := "arn:aws:apprunner:us-east-1:123456789012:service/stage-service/0000000000000001"

	resp := appRunnerRequest(t, ts, "CreateService", []byte(`{"ServiceName":"stage-service","SourceConfiguration":{"ImageRepository":{"ImageIdentifier":"public.ecr.aws/nginx/nginx:latest","ImageRepositoryType":"ECR_PUBLIC"}}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DescribeService", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListServices", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "UpdateService", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "PauseService", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ResumeService", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "StartDeployment", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListOperations", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "CreateConnection", []byte(`{"ConnectionName":"stage-connection"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListConnections", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DeleteConnection", []byte(`{"ConnectionArn":"arn:aws:apprunner:us-east-1:123456789012:connection/stage-connection/0000000000000001"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "DeleteService", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestAppRunnerStage34ConfigurationVpcDomainAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	serviceArn := "arn:aws:apprunner:us-east-1:123456789012:service/stackyard-service/0000000000000001"
	autoArn := "arn:aws:apprunner:us-east-1:123456789012:autoscalingconfiguration/stage-auto/1"
	obsArn := "arn:aws:apprunner:us-east-1:123456789012:observabilityconfiguration/stage-observability/1"
	vpcConnectorArn := "arn:aws:apprunner:us-east-1:123456789012:vpcconnector/stage-vpc-connector/0000000000000001"
	vpcIngressArn := "arn:aws:apprunner:us-east-1:123456789012:vpcingressconnection/stage-vpc-ingress/0000000000000001"

	resp := appRunnerRequest(t, ts, "CreateAutoScalingConfiguration", []byte(`{"AutoScalingConfigurationName":"stage-auto"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DescribeAutoScalingConfiguration", []byte(`{"AutoScalingConfigurationArn":"`+autoArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListAutoScalingConfigurations", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "UpdateDefaultAutoScalingConfiguration", []byte(`{"AutoScalingConfigurationArn":"`+autoArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListServicesForAutoScalingConfiguration", []byte(`{"AutoScalingConfigurationArn":"`+autoArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "CreateObservabilityConfiguration", []byte(`{"ObservabilityConfigurationName":"stage-observability"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DescribeObservabilityConfiguration", []byte(`{"ObservabilityConfigurationArn":"`+obsArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListObservabilityConfigurations", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "CreateVpcConnector", []byte(`{"VpcConnectorName":"stage-vpc-connector","Subnets":["subnet-12345"],"SecurityGroups":["sg-12345"]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DescribeVpcConnector", []byte(`{"VpcConnectorArn":"`+vpcConnectorArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListVpcConnectors", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "CreateVpcIngressConnection", []byte(`{"VpcIngressConnectionName":"stage-vpc-ingress","ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DescribeVpcIngressConnection", []byte(`{"VpcIngressConnectionArn":"`+vpcIngressArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListVpcIngressConnections", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "UpdateVpcIngressConnection", []byte(`{"VpcIngressConnectionArn":"`+vpcIngressArn+`","IngressVpcConfiguration":{"VpcId":"vpc-99999","VpcEndpointId":"vpce-99999"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "AssociateCustomDomain", []byte(`{"ServiceArn":"`+serviceArn+`","DomainName":"example.com","EnableWWWSubdomain":true}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DescribeCustomDomains", []byte(`{"ServiceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DisassociateCustomDomain", []byte(`{"ServiceArn":"`+serviceArn+`","DomainName":"example.com"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "TagResource", []byte(`{"ResourceArn":"`+serviceArn+`","Tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListTagsForResource", []byte(`{"ResourceArn":"`+serviceArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = appRunnerRequest(t, ts, "UntagResource", []byte(`{"ResourceArn":"`+serviceArn+`","TagKeys":["owner"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appRunnerRequest(t, ts, "DeleteVpcIngressConnection", []byte(`{"VpcIngressConnectionArn":"`+vpcIngressArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DeleteVpcConnector", []byte(`{"VpcConnectorArn":"`+vpcConnectorArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DeleteObservabilityConfiguration", []byte(`{"ObservabilityConfigurationArn":"`+obsArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "DeleteAutoScalingConfiguration", []byte(`{"AutoScalingConfigurationArn":"`+autoArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestAppRunnerStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{}`),
		map[string]string{"Content-Type": "application/x-amz-json-1.0"},
		"apprunner",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "missing X-Amz-Target") {
		t.Fatalf("expected missing X-Amz-Target error, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "AppRunner.ListServices",
		},
		"apprunner",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = appRunnerRequest(t, ts, "CreateService", []byte(`{"ServiceName":"idempotent-service","SourceConfiguration":{"ImageRepository":{"ImageIdentifier":"public.ecr.aws/nginx/nginx:latest","ImageRepositoryType":"ECR_PUBLIC"}}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "CreateService", []byte(`{"ServiceName":"idempotent-service","SourceConfiguration":{"ImageRepository":{"ImageIdentifier":"public.ecr.aws/nginx/nginx:latest","ImageRepositoryType":"ECR_PUBLIC"}}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = appRunnerRequest(t, ts, "ListServices", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
}
