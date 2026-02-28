package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
)

func TestEC2Stage14SDKLifecycle(t *testing.T) {
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
	client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	beforeOut, err := client.DescribeAggregateIdFormat(ctx, &awsec2.DescribeAggregateIdFormatInput{})
	if err != nil || len(beforeOut.Statuses) == 0 || !aws.ToBool(beforeOut.UseLongIdsAggregated) {
		t.Fatalf("describe aggregate id format before override: %v", err)
	}

	if _, err := client.ModifyIdentityIdFormat(ctx, &awsec2.ModifyIdentityIdFormatInput{
		PrincipalArn: aws.String("arn:aws:iam::123456789012:user/stage14"),
		Resource:     aws.String("instance"),
		UseLongIds:   aws.Bool(false),
	}); err != nil {
		t.Fatalf("modify identity id format: %v", err)
	}

	afterOut, err := client.DescribeAggregateIdFormat(ctx, &awsec2.DescribeAggregateIdFormatInput{})
	if err != nil || len(afterOut.Statuses) == 0 || aws.ToBool(afterOut.UseLongIdsAggregated) {
		t.Fatalf("describe aggregate id format after override: %v", err)
	}

	var foundInstance bool
	for _, status := range afterOut.Statuses {
		if aws.ToString(status.Resource) != "instance" {
			continue
		}
		foundInstance = true
		if aws.ToBool(status.UseLongIds) {
			t.Fatalf("expected aggregate instance useLongIds false after override")
		}
	}
	if !foundInstance {
		t.Fatalf("expected instance resource in aggregate status set")
	}
}

func TestEC2Stage14ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ec2Request(t, ts, "DescribeAggregateIdFormat", nil)
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("action DescribeAggregateIdFormat returned not implemented")
	}
}
