package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func appRunnerRequest(t *testing.T, ts *httptest.Server, action string, payload []byte) *http.Response {
	t.Helper()
	if payload == nil {
		payload = []byte(`{}`)
	}
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		payload,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "AppRunner." + action,
		},
		"apprunner",
	)
}

func TestAppRunnerStage0CatalogCoverage(t *testing.T) {
	if len(appRunnerOperations) != 37 {
		t.Fatalf("expected 37 App Runner actions from docs, got %d", len(appRunnerOperations))
	}
	if len(appRunnerOperationByName) != len(appRunnerOperations) {
		t.Fatalf("expected unique App Runner action names")
	}

	requiredActions := []string{
		"CreateService",
		"DescribeService",
		"ListServices",
		"UpdateService",
		"StartDeployment",
		"AssociateCustomDomain",
		"TagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := appRunnerOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(appRunnerDataTypes) != 34 {
		t.Fatalf("expected 34 App Runner data types from docs, got %d", len(appRunnerDataTypes))
	}
	if len(appRunnerDataTypeByName) != len(appRunnerDataTypes) {
		t.Fatalf("expected unique App Runner data type names")
	}

	requiredTypes := []string{
		"Service",
		"ServiceSummary",
		"Connection",
		"AutoScalingConfiguration",
		"CustomDomain",
		"VpcIngressConnection",
	}
	for _, typeName := range requiredTypes {
		if _, ok := appRunnerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAppRunnerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{}`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "AppRunner.UnknownAction",
		},
		"apprunner",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAppRunnerStage0KnownActionReturnsListServices(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appRunnerRequest(t, ts, "ListServices", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ServiceSummaryList") {
		t.Fatalf("expected ListServices response body to include ServiceSummaryList, got %q", body)
	}
}

func TestAppRunnerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceArn := "arn:aws:apprunner:us-east-1:123456789012:service/stackyard-service/0000000000000001"
	serviceArn := resourceArn
	autoScalingArn := "arn:aws:apprunner:us-east-1:123456789012:autoscalingconfiguration/stackyard-auto-scaling/1"
	connectionArn := "arn:aws:apprunner:us-east-1:123456789012:connection/stackyard-connection/0000000000000001"
	observabilityArn := "arn:aws:apprunner:us-east-1:123456789012:observabilityconfiguration/stackyard-observability/1"
	vpcConnectorArn := "arn:aws:apprunner:us-east-1:123456789012:vpcconnector/stackyard-vpc-connector/0000000000000001"
	vpcIngressArn := "arn:aws:apprunner:us-east-1:123456789012:vpcingressconnection/stackyard-vpc-ingress/0000000000000001"

	for _, op := range appRunnerOperations {
		payload := []byte(`{}`)
		switch op.Name {
		case "AssociateCustomDomain":
			payload = []byte(`{"ServiceArn":"` + serviceArn + `","DomainName":"example.com"}`)
		case "CreateAutoScalingConfiguration":
			payload = []byte(`{"AutoScalingConfigurationName":"stage-auto"}`)
		case "CreateConnection":
			payload = []byte(`{"ConnectionName":"stage-connection"}`)
		case "CreateObservabilityConfiguration":
			payload = []byte(`{"ObservabilityConfigurationName":"stage-observability"}`)
		case "CreateService":
			payload = []byte(`{"ServiceName":"stage-service","SourceConfiguration":{"ImageRepository":{"ImageIdentifier":"public.ecr.aws/nginx/nginx:latest","ImageRepositoryType":"ECR_PUBLIC"}}}`)
		case "CreateVpcConnector":
			payload = []byte(`{"VpcConnectorName":"stage-vpc-connector","Subnets":["subnet-12345"]}`)
		case "CreateVpcIngressConnection":
			payload = []byte(`{"VpcIngressConnectionName":"stage-vpc-ingress","ServiceArn":"` + serviceArn + `"}`)
		case "DeleteAutoScalingConfiguration", "UpdateDefaultAutoScalingConfiguration", "DescribeAutoScalingConfiguration", "ListServicesForAutoScalingConfiguration":
			payload = []byte(`{"AutoScalingConfigurationArn":"` + autoScalingArn + `"}`)
		case "DeleteConnection":
			payload = []byte(`{"ConnectionArn":"` + connectionArn + `"}`)
		case "DeleteObservabilityConfiguration", "DescribeObservabilityConfiguration":
			payload = []byte(`{"ObservabilityConfigurationArn":"` + observabilityArn + `"}`)
		case "DeleteService", "DescribeService", "PauseService", "ResumeService", "StartDeployment", "UpdateService", "ListOperations", "DescribeCustomDomains", "DisassociateCustomDomain":
			payload = []byte(`{"ServiceArn":"` + serviceArn + `","DomainName":"example.com"}`)
		case "DeleteVpcConnector", "DescribeVpcConnector":
			payload = []byte(`{"VpcConnectorArn":"` + vpcConnectorArn + `"}`)
		case "DeleteVpcIngressConnection", "DescribeVpcIngressConnection", "UpdateVpcIngressConnection":
			payload = []byte(`{"VpcIngressConnectionArn":"` + vpcIngressArn + `"}`)
		case "ListTagsForResource", "TagResource", "UntagResource":
			payload = []byte(`{"ResourceArn":"` + resourceArn + `","Tags":[{"Key":"env","Value":"stage0"}],"TagKeys":["env"]}`)
		}

		resp := appRunnerRequest(t, ts, op.Name, payload)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
