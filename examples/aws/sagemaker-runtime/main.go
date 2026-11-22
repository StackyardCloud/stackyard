package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		),
	}
	bucket := getenv("STACKYARD_BUCKET", "sagemaker-runtime-example")
	ctx := context.Background()
	fmt.Printf("Stackyard SageMaker Runtime client using %s\n", c.endpoint)

	// Control plane: a model, a real-time endpoint and an asynchronous endpoint.
	c.sagemaker(ctx, "CreateModel", map[string]any{
		"ModelName":        "example-model",
		"ExecutionRoleArn": "arn:aws:iam::123456789012:role/example-sagemaker",
		"PrimaryContainer": map[string]any{"Image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/model:latest"},
	})
	variants := []any{map[string]any{"VariantName": "blue", "ModelName": "example-model", "InitialInstanceCount": 1, "InstanceType": "ml.m5.large"}}
	c.sagemaker(ctx, "CreateEndpointConfig", map[string]any{"EndpointConfigName": "example-rt-config", "ProductionVariants": variants})
	c.sagemaker(ctx, "CreateEndpointConfig", map[string]any{
		"EndpointConfigName": "example-async-config",
		"ProductionVariants": variants,
		"AsyncInferenceConfig": map[string]any{"OutputConfig": map[string]any{
			"S3OutputPath":  "s3://" + bucket + "/output",
			"S3FailurePath": "s3://" + bucket + "/failures",
		}},
	})
	for name, config := range map[string]string{"example-rt": "example-rt-config", "example-async": "example-async-config"} {
		c.sagemaker(ctx, "CreateEndpoint", map[string]any{"EndpointName": name, "EndpointConfigName": config})
		c.sagemaker(ctx, "DescribeEndpoint", map[string]any{"EndpointName": name})
	}

	// InvokeEndpoint echoes the request body back.
	input := []byte(`{"inputs":"hello from stackyard"}`)
	status, header, body := c.do(ctx, "sagemaker", http.MethodPost, "/endpoints/example-rt/invocations", input, map[string]string{
		"Content-Type": "application/json", "Accept": "application/json",
	})
	expect2xx("InvokeEndpoint", status, body)
	if !bytes.Equal(body, input) || header.Get("x-Amzn-Invoked-Production-Variant") != "blue" {
		exitf("InvokeEndpoint returned %q from variant %q", body, header.Get("x-Amzn-Invoked-Production-Variant"))
	}
	logf("InvokeEndpoint variant=%s body=%s", header.Get("x-Amzn-Invoked-Production-Variant"), body)

	// InvokeEndpointWithResponseStream returns the same body as PayloadPart events.
	status, _, body = c.do(ctx, "sagemaker", http.MethodPost, "/endpoints/example-rt/invocations-response-stream", input, map[string]string{"Content-Type": "application/json"})
	expect2xx("InvokeEndpointWithResponseStream", status, body)
	var streamed []byte
	reader, dec := bytes.NewReader(body), eventstream.NewDecoder()
	parts := 0
	for {
		msg, err := dec.Decode(reader, nil)
		if err == io.EOF {
			break
		}
		if err != nil {
			exitf("decode event stream: %v", err)
		}
		streamed = append(streamed, msg.Payload...)
		parts++
	}
	if !bytes.Equal(streamed, input) {
		exitf("stream returned %q", streamed)
	}
	logf("InvokeEndpointWithResponseStream parts=%d", parts)

	// InvokeEndpointAsync reads the input from S3 and writes the result back to S3.
	status, _, body = c.do(ctx, "s3", http.MethodPut, "/"+bucket, nil, nil)
	expect2xx("CreateBucket", status, body)
	status, _, body = c.do(ctx, "s3", http.MethodPut, "/"+bucket+"/input.json", input, map[string]string{"Content-Type": "application/json"})
	expect2xx("PutObject", status, body)
	status, header, body = c.do(ctx, "sagemaker", http.MethodPost, "/endpoints/example-async/async-invocations", nil, map[string]string{
		"X-Amzn-SageMaker-InputLocation": "s3://" + bucket + "/input.json",
		"X-Amzn-SageMaker-Content-Type":  "application/json",
	})
	expect2xx("InvokeEndpointAsync", status, body)
	output := header.Get("X-Amzn-SageMaker-OutputLocation")
	logf("InvokeEndpointAsync %s output=%s", strings.TrimSpace(string(body)), output)
	status, _, body = c.do(ctx, "s3", http.MethodGet, "/"+strings.TrimPrefix(output, "s3://"), nil, nil)
	expect2xx("GetObject(output)", status, body)
	if !bytes.Equal(body, input) {
		exitf("async output = %q", body)
	}
	logf("async output object matches the input")

	for _, name := range []string{"example-rt", "example-async"} {
		c.sagemaker(ctx, "DeleteEndpoint", map[string]any{"EndpointName": name})
		c.sagemaker(ctx, "DeleteEndpointConfig", map[string]any{"EndpointConfigName": name + "-config"})
	}
	c.sagemaker(ctx, "DeleteModel", map[string]any{"ModelName": "example-model"})
	fmt.Println("SageMaker Runtime example completed")
}

func (c client) sagemaker(ctx context.Context, action string, payload any) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	status, _, body := c.do(ctx, "sagemaker", http.MethodPost, "/", encoded, map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "SageMaker." + action,
	})
	expect2xx(action, status, body)
}

func (c client) do(ctx context.Context, service, method, path string, body []byte, headers map[string]string) (int, http.Header, []byte) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read %s %s: %v", method, path, err)
	}
	return resp.StatusCode, resp.Header, out
}

func expect2xx(action string, status int, body []byte) {
	if status < 200 || status > 299 {
		exitf("%s failed: status=%d body=%s", action, status, strings.TrimSpace(string(body)))
	}
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf("  "+format+"\n", args...)
}
