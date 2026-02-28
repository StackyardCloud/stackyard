package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	bucket := getenv("STACKYARD_BUCKET", "demo-basic")
	key := "notes/hello.txt"
	payload := "hello from stackyard s3 basic example"

	ctx := context.Background()
	client := newS3Client(ctx, endpoint)

	fmt.Printf("Stackyard S3 basic client using %s\n", endpoint)

	if err := createBucket(ctx, client, bucket); err != nil {
		exitf("create bucket: %v", err)
	}
	logf("created bucket: %s", bucket)

	if err := putObject(ctx, client, bucket, key, payload); err != nil {
		exitf("put object: %v", err)
	}
	logf("put object: %s", key)

	data, err := getObject(ctx, client, bucket, key)
	if err != nil {
		exitf("get object: %v", err)
	}

	fmt.Printf("Get object %s/%s -> %q\n", bucket, key, string(data))
	fmt.Println("Done.")
}

func newS3Client(ctx context.Context, endpoint string) *s3.Client {
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

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
}

func createBucket(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	return err
}

func putObject(ctx context.Context, client *s3.Client, bucket, key, payload string) error {
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(payload),
		ContentType: aws.String("text/plain"),
	})
	return err
}

func getObject(ctx context.Context, client *s3.Client, bucket, key string) ([]byte, error) {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
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

func init() {
	_ = time.Now()
}
