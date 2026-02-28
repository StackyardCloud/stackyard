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

func TestEC2Stage12SDKLifecycle(t *testing.T) {
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

	describeOut, err := client.DescribeIdFormat(ctx, &awsec2.DescribeIdFormatInput{
		Resource: aws.String("instance"),
	})
	if err != nil || len(describeOut.Statuses) != 1 || aws.ToString(describeOut.Statuses[0].Resource) != "instance" || !aws.ToBool(describeOut.Statuses[0].UseLongIds) {
		t.Fatalf("describe id format: %v", err)
	}

	if _, err := client.ModifyIdFormat(ctx, &awsec2.ModifyIdFormatInput{
		Resource:   aws.String("instance"),
		UseLongIds: aws.Bool(false),
	}); err != nil {
		t.Fatalf("modify id format: %v", err)
	}

	describeAfterModifyOut, err := client.DescribeIdFormat(ctx, &awsec2.DescribeIdFormatInput{
		Resource: aws.String("instance"),
	})
	if err != nil || len(describeAfterModifyOut.Statuses) != 1 || aws.ToBool(describeAfterModifyOut.Statuses[0].UseLongIds) {
		t.Fatalf("describe id format after modify: %v", err)
	}

	describeIdentityOut, err := client.DescribeIdentityIdFormat(ctx, &awsec2.DescribeIdentityIdFormatInput{
		PrincipalArn: aws.String("arn:aws:iam::123456789012:user/stage12"),
		Resource:     aws.String("instance"),
	})
	if err != nil || len(describeIdentityOut.Statuses) != 1 || aws.ToString(describeIdentityOut.Statuses[0].Resource) != "instance" || aws.ToBool(describeIdentityOut.Statuses[0].UseLongIds) {
		t.Fatalf("describe identity id format: %v", err)
	}

	describePrincipalOut, err := client.DescribePrincipalIdFormat(ctx, &awsec2.DescribePrincipalIdFormatInput{
		Resources: []string{"instance"},
	})
	if err != nil {
		t.Fatalf("describe principal id format: %v", err)
	}
	if len(describePrincipalOut.Principals) == 0 || aws.ToString(describePrincipalOut.Principals[0].Arn) == "" || len(describePrincipalOut.Principals[0].Statuses) != 1 || aws.ToString(describePrincipalOut.Principals[0].Statuses[0].Resource) != "instance" || aws.ToBool(describePrincipalOut.Principals[0].Statuses[0].UseLongIds) {
		t.Fatalf("unexpected describe principal id format output: %+v", describePrincipalOut.Principals)
	}
}

func TestEC2Stage12ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"DescribeIdFormat",
		"DescribeIdentityIdFormat",
		"DescribePrincipalIdFormat",
		"ModifyIdFormat",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "DescribeIdFormat":
			params["Resource"] = "instance"
		case "DescribeIdentityIdFormat":
			params["PrincipalArn"] = "arn:aws:iam::123456789012:user/stage12"
			params["Resource"] = "instance"
		case "DescribePrincipalIdFormat":
			params["Resource.1"] = "instance"
		case "ModifyIdFormat":
			params["Resource"] = "instance"
			params["UseLongIds"] = "false"
		}
		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
