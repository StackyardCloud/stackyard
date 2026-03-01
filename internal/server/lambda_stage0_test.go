package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	awslambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func lambdaRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "lambda")
}

func TestLambdaStage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBody := []byte(`{
		"FunctionName":"demo-fn",
		"Role":"arn:aws:iam::123456789012:role/lambda-role",
		"Runtime":"provided.al2",
		"Handler":"bootstrap",
		"Code":{"ZipFile":"c3RhY2t5YXJk"},
		"Tags":{"env":"test"}
	}`)
	resp := lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions", createBody)
	assertStatus(t, resp, http.StatusCreated)
	var createOut struct {
		FunctionArn string `json:"FunctionArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create function: %v", err)
	}
	if createOut.FunctionArn == "" {
		t.Fatalf("expected function arn")
	}

	resp = lambdaRequest(t, ts, http.MethodGet, "/2015-03-31/functions", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodGet, "/2015-03-31/functions/demo-fn", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodGet, "/2015-03-31/functions/demo-fn/configuration", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodPut, "/2015-03-31/functions/demo-fn/configuration", []byte(`{"Description":"updated","Timeout":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodPut, "/2015-03-31/functions/demo-fn/code", []byte(`{"ZipFile":"bmV3LWNvZGU="}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions/demo-fn/versions", []byte(`{"Description":"v1"}`))
	assertStatus(t, resp, http.StatusCreated)

	resp = lambdaRequest(t, ts, http.MethodGet, "/2015-03-31/functions/demo-fn/versions", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions/demo-fn/aliases", []byte(`{"Name":"live","FunctionVersion":"1","Description":"live alias"}`))
	assertStatus(t, resp, http.StatusCreated)

	resp = lambdaRequest(t, ts, http.MethodGet, "/2015-03-31/functions/demo-fn/aliases", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodGet, "/2015-03-31/functions/demo-fn/aliases/live", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodPut, "/2015-03-31/functions/demo-fn/aliases/live", []byte(`{"Description":"prod alias"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions/demo-fn/policy", []byte(`{"StatementId":"stmt-1","Action":"lambda:InvokeFunction","Principal":"123456789012"}`))
	assertStatus(t, resp, http.StatusCreated)

	resp = lambdaRequest(t, ts, http.MethodGet, "/2015-03-31/functions/demo-fn/policy", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodDelete, "/2015-03-31/functions/demo-fn/policy/stmt-1", nil)
	assertStatus(t, resp, http.StatusNoContent)

	tagPath := "/2017-03-31/tags/" + url.PathEscape(createOut.FunctionArn)
	resp = lambdaRequest(t, ts, http.MethodPost, tagPath, []byte(`{"Tags":{"team":"platform"}}`))
	assertStatus(t, resp, http.StatusNoContent)

	resp = lambdaRequest(t, ts, http.MethodGet, tagPath, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodDelete, tagPath+"?tagKeys=team", nil)
	assertStatus(t, resp, http.StatusNoContent)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions/demo-fn/invocations", []byte(`{"hello":"world"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lambdaRequest(t, ts, http.MethodDelete, "/2015-03-31/functions/demo-fn/aliases/live", nil)
	assertStatus(t, resp, http.StatusNoContent)

	resp = lambdaRequest(t, ts, http.MethodDelete, "/2015-03-31/functions/demo-fn", nil)
	assertStatus(t, resp, http.StatusNoContent)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/event-source-mappings", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
}

func TestLambdaStage0OperationCoverage(t *testing.T) {
	if len(lambdaOperations) != 85 {
		t.Fatalf("expected 85 Lambda operations from docs, got %d", len(lambdaOperations))
	}
	if len(lambdaOperationByName) != len(lambdaOperations) {
		t.Fatalf("expected unique operation names")
	}
	implemented := make(map[string]struct{}, len(lambdaRoutes))
	for _, route := range lambdaRoutes {
		if _, ok := lambdaOperationByName[route.Operation]; !ok {
			t.Fatalf("route has unknown operation %s", route.Operation)
		}
		implemented[route.Operation] = struct{}{}
	}
	if len(implemented) != len(lambdaOperations) {
		t.Fatalf("expected all lambda operations to be routed, missing=%d", len(lambdaOperations)-len(implemented))
	}
	for _, op := range lambdaOperations {
		if _, ok := implemented[op.Name]; !ok {
			t.Fatalf("operation %s is not routed", op.Name)
		}
	}
	required := []string{
		"CreateFunction",
		"UpdateFunctionCode",
		"UpdateFunctionConfiguration",
		"PublishVersion",
		"Invoke",
		"CreateAlias",
		"GetPolicy",
		"TagResource",
		"ListTags",
		"CreateEventSourceMapping",
	}
	for _, name := range required {
		if _, ok := lambdaOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestLambdaStage0SDKClientLifecycle(t *testing.T) {
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
	client := awslambda.NewFromConfig(cfg, func(o *awslambda.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	createOut, err := client.CreateFunction(ctx, &awslambda.CreateFunctionInput{
		FunctionName: aws.String("sdk-fn"),
		Role:         aws.String("arn:aws:iam::123456789012:role/lambda-role"),
		Runtime:      awslambdatypes.Runtime("provided.al2"),
		Handler:      aws.String("bootstrap"),
		Code:         &awslambdatypes.FunctionCode{ZipFile: []byte("stackyard")},
	})
	if err != nil {
		t.Fatalf("create function: %v", err)
	}
	if createOut.FunctionArn == nil {
		t.Fatalf("expected function arn")
	}

	if _, err := client.GetFunction(ctx, &awslambda.GetFunctionInput{FunctionName: aws.String("sdk-fn")}); err != nil {
		t.Fatalf("get function: %v", err)
	}
	if _, err := client.UpdateFunctionConfiguration(ctx, &awslambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String("sdk-fn"),
		Description:  aws.String("updated"),
	}); err != nil {
		t.Fatalf("update function configuration: %v", err)
	}
	if _, err := client.UpdateFunctionCode(ctx, &awslambda.UpdateFunctionCodeInput{
		FunctionName: aws.String("sdk-fn"),
		ZipFile:      []byte("new-code"),
	}); err != nil {
		t.Fatalf("update function code: %v", err)
	}
	if _, err := client.PublishVersion(ctx, &awslambda.PublishVersionInput{FunctionName: aws.String("sdk-fn")}); err != nil {
		t.Fatalf("publish version: %v", err)
	}
	if _, err := client.ListVersionsByFunction(ctx, &awslambda.ListVersionsByFunctionInput{FunctionName: aws.String("sdk-fn")}); err != nil {
		t.Fatalf("list versions by function: %v", err)
	}

	if _, err := client.CreateAlias(ctx, &awslambda.CreateAliasInput{
		FunctionName:    aws.String("sdk-fn"),
		Name:            aws.String("live"),
		FunctionVersion: aws.String("1"),
	}); err != nil {
		t.Fatalf("create alias: %v", err)
	}
	if _, err := client.GetAlias(ctx, &awslambda.GetAliasInput{FunctionName: aws.String("sdk-fn"), Name: aws.String("live")}); err != nil {
		t.Fatalf("get alias: %v", err)
	}
	if _, err := client.ListAliases(ctx, &awslambda.ListAliasesInput{FunctionName: aws.String("sdk-fn")}); err != nil {
		t.Fatalf("list aliases: %v", err)
	}
	if _, err := client.UpdateAlias(ctx, &awslambda.UpdateAliasInput{
		FunctionName:    aws.String("sdk-fn"),
		Name:            aws.String("live"),
		FunctionVersion: aws.String("1"),
	}); err != nil {
		t.Fatalf("update alias: %v", err)
	}

	if _, err := client.Invoke(ctx, &awslambda.InvokeInput{
		FunctionName: aws.String("sdk-fn"),
		Payload:      []byte(`{"hello":"world"}`),
	}); err != nil {
		t.Fatalf("invoke: %v", err)
	}

	addPerm, err := client.AddPermission(ctx, &awslambda.AddPermissionInput{
		FunctionName: aws.String("sdk-fn"),
		StatementId:  aws.String("stmt-1"),
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String("123456789012"),
	})
	if err != nil {
		t.Fatalf("add permission: %v", err)
	}
	if addPerm.Statement == nil {
		t.Fatalf("expected permission statement")
	}
	if _, err := client.GetPolicy(ctx, &awslambda.GetPolicyInput{FunctionName: aws.String("sdk-fn")}); err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if _, err := client.RemovePermission(ctx, &awslambda.RemovePermissionInput{
		FunctionName: aws.String("sdk-fn"),
		StatementId:  aws.String("stmt-1"),
	}); err != nil {
		t.Fatalf("remove permission: %v", err)
	}

	if _, err := client.TagResource(ctx, &awslambda.TagResourceInput{
		Resource: createOut.FunctionArn,
		Tags:     map[string]string{"env": "test"},
	}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	if _, err := client.ListTags(ctx, &awslambda.ListTagsInput{Resource: createOut.FunctionArn}); err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if _, err := client.UntagResource(ctx, &awslambda.UntagResourceInput{
		Resource: createOut.FunctionArn,
		TagKeys:  []string{"env"},
	}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}

	if _, err := client.DeleteAlias(ctx, &awslambda.DeleteAliasInput{FunctionName: aws.String("sdk-fn"), Name: aws.String("live")}); err != nil {
		t.Fatalf("delete alias: %v", err)
	}
	if _, err := client.DeleteFunction(ctx, &awslambda.DeleteFunctionInput{FunctionName: aws.String("sdk-fn")}); err != nil {
		t.Fatalf("delete function: %v", err)
	}
}

func TestLambdaStage0AllRoutesAreImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, route := range lambdaRoutes {
		path := lambdaRouteSamplePath(route)
		var body []byte
		switch route.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			body = []byte(`{}`)
		}
		resp := lambdaRequest(t, ts, route.Method, path, body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("route %s %s (%s) returned 501", route.Method, path, route.Operation)
		}
	}
}

func TestLambdaInvokeLocalExecutionHeaders(t *testing.T) {
	srv := New(Config{
		Addr:                "127.0.0.1:0",
		AccessKey:           testAccessKey,
		SecretKey:           testSecretKey,
		LogLevel:            "error",
		LambdaExecutionMode: "local",
		LambdaWorkDir:       t.TempDir(),
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	codeB64 := base64.StdEncoding.EncodeToString(lambdaZipBootstrap(t, `#!/bin/sh
echo "boom from function" 1>&2
exit 1
`))

	createBody := []byte(fmt.Sprintf(`{
		"FunctionName":"local-fail-fn",
		"Role":"arn:aws:iam::123456789012:role/lambda-role",
		"Runtime":"provided.al2",
		"Handler":"bootstrap",
		"Code":{"ZipFile":"%s"}
	}`, codeB64))
	resp := lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions", createBody)
	assertStatus(t, resp, http.StatusCreated)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions/local-fail-fn/invocations", []byte(`{"hello":"world"}`))
	assertStatus(t, resp, http.StatusOK)
	if got := strings.TrimSpace(resp.Header.Get("X-Amz-Function-Error")); got != "Unhandled" {
		t.Fatalf("expected X-Amz-Function-Error=Unhandled, got %q", got)
	}
	if got := strings.TrimSpace(resp.Header.Get("X-Amz-Log-Result")); got == "" {
		t.Fatalf("expected X-Amz-Log-Result header to be present")
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, `"errorType":"RuntimeError"`) {
		t.Fatalf("expected runtime error payload, got %s", body)
	}
}

func TestLambdaInvokeLocalExecutionReturnsFunctionPayload(t *testing.T) {
	srv := New(Config{
		Addr:                "127.0.0.1:0",
		AccessKey:           testAccessKey,
		SecretKey:           testSecretKey,
		LogLevel:            "error",
		LambdaExecutionMode: "local",
		LambdaWorkDir:       t.TempDir(),
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	codeB64 := base64.StdEncoding.EncodeToString(lambdaZipBootstrap(t, `#!/bin/sh
payload="$(cat)"
if [ -z "$payload" ]; then
  payload='{}'
fi
printf '{"ok":true,"payload":%s}' "$payload"
`))
	createBody := []byte(fmt.Sprintf(`{
		"FunctionName":"local-ok-fn",
		"Role":"arn:aws:iam::123456789012:role/lambda-role",
		"Runtime":"provided.al2",
		"Handler":"bootstrap",
		"Code":{"ZipFile":"%s"}
	}`, codeB64))
	resp := lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions", createBody)
	assertStatus(t, resp, http.StatusCreated)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions/local-ok-fn/invocations", []byte(`{"hello":"world"}`))
	assertStatus(t, resp, http.StatusOK)
	if got := strings.TrimSpace(resp.Header.Get("X-Amz-Function-Error")); got != "" {
		t.Fatalf("expected empty X-Amz-Function-Error header, got %q", got)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, `"ok":true`) {
		t.Fatalf("expected successful payload body, got %s", body)
	}
	if !strings.Contains(body, `"hello":"world"`) {
		t.Fatalf("expected invocation payload in response, got %s", body)
	}
}

func lambdaRouteSamplePath(route lambdaRoute) string {
	path := route.Pattern
	replacements := map[string]string{
		"{FunctionName}":         "demo-fn",
		"{Name}":                 "live",
		"{StatementId}":          "stmt-1",
		"{VersionNumber}":        "1",
		"{LayerName}":            "demo-layer",
		"{UUID}":                 "esm-00000001",
		"{CodeSigningConfigArn}": url.PathEscape("arn:aws:lambda:us-east-1:123456789012:code-signing-config:csc-00000001"),
		"{CapacityProviderName}": "demo-provider",
		"{DurableExecutionArn}":  url.PathEscape("arn:aws:lambda:us-east-1:123456789012:durable-execution:demo"),
		"{CallbackId}":           "cb-1",
		"{Resource+}":            url.PathEscape("arn:aws:lambda:us-east-1:123456789012:function:demo-fn"),
	}
	for from, to := range replacements {
		path = strings.ReplaceAll(path, from, to)
	}
	switch route.Operation {
	case "GetLayerVersionByArn":
		path += "?find=LayerVersion&Arn=" + url.QueryEscape("arn:aws:lambda:us-east-1:123456789012:layer:demo-layer:1")
	case "ListProvisionedConcurrencyConfigs":
		path += "?List=ALL"
	}
	return path
}

func lambdaZipBootstrap(t *testing.T, script string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	header := &zip.FileHeader{
		Name:   "bootstrap",
		Method: zip.Deflate,
	}
	header.SetMode(0o755)
	w, err := zw.CreateHeader(header)
	if err != nil {
		t.Fatalf("create zip header: %v", err)
	}
	if _, err := w.Write([]byte(script)); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}
