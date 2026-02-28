package main

import (
	"context"
	"fmt"
	"os"

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
			ZipFile: []byte("stackyard-lambda-basic"),
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
