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
	functionName := getenv("STACKYARD_FUNCTION_NAME", "lambda-advanced-function")
	aliasName := getenv("STACKYARD_ALIAS_NAME", "live")

	ctx := context.Background()
	client := newLambdaClient(ctx, endpoint)

	fmt.Printf("Stackyard Lambda advanced client using %s\n", endpoint)

	createOut, err := client.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(functionName),
		Role:         aws.String("arn:aws:iam::123456789012:role/lambda-role"),
		Runtime:      lambdatypes.Runtime("provided.al2"),
		Handler:      aws.String("bootstrap"),
		Code:         &lambdatypes.FunctionCode{ZipFile: []byte("stackyard-lambda-advanced-v1")},
		Tags:         map[string]string{"env": "dev"},
	})
	if err != nil {
		exitf("create function: %v", err)
	}

	if _, err := client.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(functionName),
		Description:  aws.String("advanced function"),
		Timeout:      aws.Int32(15),
		MemorySize:   aws.Int32(256),
	}); err != nil {
		exitf("update function configuration: %v", err)
	}

	if _, err := client.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(functionName),
		ZipFile:      []byte("stackyard-lambda-advanced-v2"),
	}); err != nil {
		exitf("update function code: %v", err)
	}

	versionOut, err := client.PublishVersion(ctx, &lambda.PublishVersionInput{
		FunctionName: aws.String(functionName),
		Description:  aws.String("release v1"),
	})
	if err != nil {
		exitf("publish version: %v", err)
	}

	if _, err := client.CreateAlias(ctx, &lambda.CreateAliasInput{
		FunctionName:    aws.String(functionName),
		Name:            aws.String(aliasName),
		FunctionVersion: versionOut.Version,
		Description:     aws.String("live traffic"),
	}); err != nil {
		exitf("create alias: %v", err)
	}

	addPermOut, err := client.AddPermission(ctx, &lambda.AddPermissionInput{
		FunctionName: aws.String(functionName),
		StatementId:  aws.String("allow-account"),
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String("123456789012"),
	})
	if err != nil {
		exitf("add permission: %v", err)
	}
	logf("permission statement bytes: %d", len(aws.ToString(addPermOut.Statement)))

	if _, err := client.GetPolicy(ctx, &lambda.GetPolicyInput{
		FunctionName: aws.String(functionName),
	}); err != nil {
		exitf("get policy: %v", err)
	}

	if _, err := client.TagResource(ctx, &lambda.TagResourceInput{
		Resource: createOut.FunctionArn,
		Tags:     map[string]string{"team": "platform"},
	}); err != nil {
		exitf("tag resource: %v", err)
	}
	listTagsOut, err := client.ListTags(ctx, &lambda.ListTagsInput{
		Resource: createOut.FunctionArn,
	})
	if err != nil {
		exitf("list tags: %v", err)
	}
	logf("tag count: %d", len(listTagsOut.Tags))

	invokeOut, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(functionName + ":" + aliasName),
		Payload:      []byte(`{"mode":"advanced"}`),
	})
	if err != nil {
		exitf("invoke alias: %v", err)
	}
	logf("invoke status: %d", invokeOut.StatusCode)

	versionsOut, err := client.ListVersionsByFunction(ctx, &lambda.ListVersionsByFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		exitf("list versions: %v", err)
	}
	logf("versions: %d", len(versionsOut.Versions))

	aliasesOut, err := client.ListAliases(ctx, &lambda.ListAliasesInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		exitf("list aliases: %v", err)
	}
	logf("aliases: %d", len(aliasesOut.Aliases))

	if _, err := client.RemovePermission(ctx, &lambda.RemovePermissionInput{
		FunctionName: aws.String(functionName),
		StatementId:  aws.String("allow-account"),
	}); err != nil {
		exitf("remove permission: %v", err)
	}

	if _, err := client.UntagResource(ctx, &lambda.UntagResourceInput{
		Resource: createOut.FunctionArn,
		TagKeys:  []string{"team"},
	}); err != nil {
		exitf("untag resource: %v", err)
	}

	if _, err := client.DeleteAlias(ctx, &lambda.DeleteAliasInput{
		FunctionName: aws.String(functionName),
		Name:         aws.String(aliasName),
	}); err != nil {
		exitf("delete alias: %v", err)
	}

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
