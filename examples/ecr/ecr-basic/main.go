package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	repositoryName := getenv("STACKYARD_REPOSITORY_NAME", "ecr-basic-repo")

	ctx := context.Background()
	client := newECRClient(ctx, endpoint)

	fmt.Printf("Stackyard ECR basic client using %s\n", endpoint)

	authOut, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		exitf("get authorization token: %v", err)
	}
	logf("authorization entries: %d", len(authOut.AuthorizationData))

	createOut, err := client.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil {
		exitf("create repository: %v", err)
	}
	logf("created repository: %s", aws.ToString(createOut.Repository.RepositoryArn))

	putImageOut, err := client.PutImage(ctx, &ecr.PutImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageManifest:  aws.String(`{"schemaVersion":2}`),
		ImageTag:       aws.String("latest"),
	})
	if err != nil {
		exitf("put image: %v", err)
	}
	logf("pushed image digest: %s", aws.ToString(putImageOut.Image.ImageId.ImageDigest))

	listOut, err := client.ListImages(ctx, &ecr.ListImagesInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil {
		exitf("list images: %v", err)
	}
	logf("image ids: %d", len(listOut.ImageIds))

	describeOut, err := client.DescribeImages(ctx, &ecr.DescribeImagesInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil {
		exitf("describe images: %v", err)
	}
	logf("image details: %d", len(describeOut.ImageDetails))

	if _, err := client.DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(repositoryName),
		Force:          true,
	}); err != nil {
		exitf("delete repository: %v", err)
	}

	fmt.Println("Done.")
}

func newECRClient(ctx context.Context, endpoint string) *ecr.Client {
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

	return ecr.NewFromConfig(cfg, func(o *ecr.Options) {
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
