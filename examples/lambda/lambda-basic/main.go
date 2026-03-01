package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	functionName := getenv("STACKYARD_FUNCTION_NAME", "lambda-basic-function")

	ctx := context.Background()
	client := newLambdaClient(ctx, endpoint)

	fmt.Printf("Stackyard Lambda basic client using %s\n", endpoint)

	createOut, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(functionName),
		Role:         aws.String("arn:aws:iam::123456789012:role/lambda-role"),
		Runtime:      lambdatypes.Runtime("provided.al2"),
		Handler:      aws.String("bootstrap"),
		Code: &lambdatypes.FunctionCode{
			ZipFile: bootstrapArchiveFromSource(`package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	body, _ := io.ReadAll(os.Stdin)
	payload := strings.TrimSpace(string(body))
	if payload == "" {
		payload = "{}"
	}
	fmt.Printf("{\"ok\":true,\"payload\":%s}", payload)
}
`),
		},
	})
	if err != nil {
		exitf("create function: %v", err)
	}
	logf("created function: %s", aws.ToString(createOut.FunctionArn))

	invokeOut, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      []byte(`{"hello":"stackyard"}`),
	})
	if err != nil {
		exitf("invoke function: %v", err)
	}
	logf("invoke status: %d", invokeOut.StatusCode)
	if ferr := strings.TrimSpace(aws.ToString(invokeOut.FunctionError)); ferr != "" {
		exitf("invoke returned function error: %s (payload=%s)", ferr, string(invokeOut.Payload))
	}
	if !bytes.Contains(invokeOut.Payload, []byte(`"ok":true`)) {
		exitf("unexpected invoke payload: %s", string(invokeOut.Payload))
	}
	if !bytes.Contains(invokeOut.Payload, []byte(`"hello":"stackyard"`)) {
		exitf("invoke payload did not include request body: %s", string(invokeOut.Payload))
	}
	logf("invoke payload: %s", string(invokeOut.Payload))

	listOut, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		exitf("list functions: %v", err)
	}
	logf("functions: %d", len(listOut.Functions))

	if _, err := client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(functionName),
	}); err != nil {
		exitf("delete function: %v", err)
	}

	fmt.Println("Done.")
}

func newLambdaClient(ctx context.Context, endpoint string) *lambda.Client {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(getenv("AWS_REGION", "us-east-1")),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		)),
	)
	if err != nil {
		exitf("load aws config: %v", err)
	}

	return lambda.NewFromConfig(cfg, func(o *lambda.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func bootstrapArchiveFromSource(source string) []byte {
	dir, err := os.MkdirTemp("", "stackyard-lambda-basic-*")
	if err != nil {
		exitf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	sourcePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		exitf("write bootstrap source: %v", err)
	}
	bootstrapPath := filepath.Join(dir, "bootstrap")

	cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", bootstrapPath, sourcePath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		exitf("build bootstrap binary: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	bootstrapBytes, err := os.ReadFile(bootstrapPath)
	if err != nil {
		exitf("read bootstrap binary: %v", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	header := &zip.FileHeader{
		Name:   "bootstrap",
		Method: zip.Deflate,
	}
	header.SetMode(0o755)
	w, err := zw.CreateHeader(header)
	if err != nil {
		exitf("create zip entry: %v", err)
	}
	if _, err := io.Copy(w, bytes.NewReader(bootstrapBytes)); err != nil {
		exitf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		exitf("close zip: %v", err)
	}
	return buf.Bytes()
}
