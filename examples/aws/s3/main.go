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
	bucket := getenv("STACKYARD_BUCKET", "demo")
	key := "notes/advanced.txt"
	payload := "advanced example payload"

	ctx := context.Background()
	client := newS3Client(ctx, endpoint)

	fmt.Printf("Stackyard S3 advanced client using %s\n", endpoint)

	if err := createBucket(ctx, client, bucket); err != nil {
		exitf("create bucket: %v", err)
	}
	logf("created bucket: %s", bucket)

	if err := putObject(ctx, client, bucket, key, payload); err != nil {
		exitf("put object: %v", err)
	}
	logf("put object: %s", key)

	if err := listObjects(ctx, client, bucket); err != nil {
		exitf("list objects: %v", err)
	}

	if err := headObject(ctx, client, bucket, key); err != nil {
		exitf("head object: %v", err)
	}

	if err := rangeRead(ctx, client, bucket, key, "bytes=0-7"); err != nil {
		exitf("range read: %v", err)
	}

	if err := deleteObject(ctx, client, bucket, key); err != nil {
		exitf("delete object: %v", err)
	}
	logf("deleted object: %s", key)

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
		Metadata: map[string]string{
			"env": "dev",
		},
	})
	return err
}

func listObjects(ctx context.Context, client *s3.Client, bucket string) error {
	resp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return err
	}
	for _, obj := range resp.Contents {
		fmt.Printf("Object: %s (%d bytes)\n", aws.ToString(obj.Key), obj.Size)
	}
	return nil
}

func headObject(ctx context.Context, client *s3.Client, bucket, key string) error {
	resp, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	fmt.Printf("Head object Content-Length=%d ETag=%s\n", resp.ContentLength, aws.ToString(resp.ETag))
	return nil
}

func rangeRead(ctx context.Context, client *s3.Client, bucket, key, rng string) error {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String(rng),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("Range read %s -> %q\n", rng, string(data))
	return nil
}

func deleteObject(ctx context.Context, client *s3.Client, bucket, key string) error {
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
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
