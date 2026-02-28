package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awslightsail "github.com/aws/aws-sdk-go-v2/service/lightsail"
	awslightsailtypes "github.com/aws/aws-sdk-go-v2/service/lightsail/types"
)

func TestLightsailStage16BucketKeysAndContactMethods(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateBucket", []byte(`{"bucketName":"stage16-bucket","bundleId":"small_1_0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateBucketAccessKey", []byte(`{"bucketName":"stage16-bucket"}`))
	assertStatus(t, resp, http.StatusOK)
	var createAccessKeyOut struct {
		AccessKey struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
		} `json:"accessKey"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createAccessKeyOut); err != nil {
		t.Fatalf("unmarshal CreateBucketAccessKey: %v", err)
	}
	if createAccessKeyOut.AccessKey.AccessKeyID == "" || createAccessKeyOut.AccessKey.SecretAccessKey == "" {
		t.Fatalf("expected access key id and secret: %+v", createAccessKeyOut)
	}

	resp = lightsailRequest(t, ts, "GetBucketAccessKeys", []byte(`{"bucketName":"stage16-bucket"}`))
	assertStatus(t, resp, http.StatusOK)
	var getAccessKeysOut struct {
		AccessKeys []struct {
			AccessKeyID string `json:"accessKeyId"`
		} `json:"accessKeys"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getAccessKeysOut); err != nil {
		t.Fatalf("unmarshal GetBucketAccessKeys: %v", err)
	}
	if len(getAccessKeysOut.AccessKeys) != 1 || getAccessKeysOut.AccessKeys[0].AccessKeyID == "" {
		t.Fatalf("unexpected GetBucketAccessKeys output: %+v", getAccessKeysOut)
	}

	resp = lightsailRequest(t, ts, "DeleteBucketAccessKey", []byte(`{"bucketName":"stage16-bucket","accessKeyId":"`+getAccessKeysOut.AccessKeys[0].AccessKeyID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateContactMethod", []byte(`{"protocol":"Email","contactEndpoint":"alerts@example.com"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetContactMethods", []byte(`{"protocols":["Email"]}`))
	assertStatus(t, resp, http.StatusOK)
	var getContactMethodsOut struct {
		ContactMethods []struct {
			Protocol string `json:"protocol"`
			Status   string `json:"status"`
		} `json:"contactMethods"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getContactMethodsOut); err != nil {
		t.Fatalf("unmarshal GetContactMethods: %v", err)
	}
	if len(getContactMethodsOut.ContactMethods) != 1 || getContactMethodsOut.ContactMethods[0].Protocol != "Email" {
		t.Fatalf("unexpected GetContactMethods output: %+v", getContactMethodsOut)
	}

	resp = lightsailRequest(t, ts, "SendContactMethodVerification", []byte(`{"protocol":"Email"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetContactMethods", []byte(`{"protocols":["Email"]}`))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &getContactMethodsOut); err != nil {
		t.Fatalf("unmarshal GetContactMethods after verify: %v", err)
	}
	if len(getContactMethodsOut.ContactMethods) != 1 || getContactMethodsOut.ContactMethods[0].Status != "Valid" {
		t.Fatalf("expected verified contact method status Valid: %+v", getContactMethodsOut)
	}

	resp = lightsailRequest(t, ts, "DeleteContactMethod", []byte(`{"protocol":"Email"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage16SDKClientBucketKeysAndContactMethods(t *testing.T) {
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

	client := awslightsail.NewFromConfig(cfg, func(o *awslightsail.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	if _, err := client.CreateBucket(ctx, &awslightsail.CreateBucketInput{
		BucketName: aws.String("sdk-stage16-bucket"),
		BundleId:   aws.String("small_1_0"),
	}); err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	createKeyOut, err := client.CreateBucketAccessKey(ctx, &awslightsail.CreateBucketAccessKeyInput{
		BucketName: aws.String("sdk-stage16-bucket"),
	})
	if err != nil {
		t.Fatalf("create bucket access key: %v", err)
	}
	if createKeyOut.AccessKey == nil || createKeyOut.AccessKey.AccessKeyId == nil || *createKeyOut.AccessKey.AccessKeyId == "" || createKeyOut.AccessKey.SecretAccessKey == nil || *createKeyOut.AccessKey.SecretAccessKey == "" {
		t.Fatalf("unexpected create bucket access key output: %+v", createKeyOut.AccessKey)
	}

	getKeysOut, err := client.GetBucketAccessKeys(ctx, &awslightsail.GetBucketAccessKeysInput{
		BucketName: aws.String("sdk-stage16-bucket"),
	})
	if err != nil {
		t.Fatalf("get bucket access keys: %v", err)
	}
	if len(getKeysOut.AccessKeys) != 1 || getKeysOut.AccessKeys[0].AccessKeyId == nil || *getKeysOut.AccessKeys[0].AccessKeyId == "" {
		t.Fatalf("unexpected get bucket access keys output: %+v", getKeysOut.AccessKeys)
	}

	if _, err := client.DeleteBucketAccessKey(ctx, &awslightsail.DeleteBucketAccessKeyInput{
		BucketName:  aws.String("sdk-stage16-bucket"),
		AccessKeyId: getKeysOut.AccessKeys[0].AccessKeyId,
	}); err != nil {
		t.Fatalf("delete bucket access key: %v", err)
	}

	if _, err := client.CreateContactMethod(ctx, &awslightsail.CreateContactMethodInput{
		Protocol:        awslightsailtypes.ContactProtocolEmail,
		ContactEndpoint: aws.String("alerts@example.com"),
	}); err != nil {
		t.Fatalf("create contact method: %v", err)
	}

	contactsOut, err := client.GetContactMethods(ctx, &awslightsail.GetContactMethodsInput{
		Protocols: []awslightsailtypes.ContactProtocol{awslightsailtypes.ContactProtocolEmail},
	})
	if err != nil {
		t.Fatalf("get contact methods: %v", err)
	}
	if len(contactsOut.ContactMethods) != 1 {
		t.Fatalf("expected one contact method, got %d", len(contactsOut.ContactMethods))
	}

	if _, err := client.SendContactMethodVerification(ctx, &awslightsail.SendContactMethodVerificationInput{
		Protocol: awslightsailtypes.ContactMethodVerificationProtocolEmail,
	}); err != nil {
		t.Fatalf("send contact method verification: %v", err)
	}

	if _, err := client.DeleteContactMethod(ctx, &awslightsail.DeleteContactMethodInput{
		Protocol: awslightsailtypes.ContactProtocolEmail,
	}); err != nil {
		t.Fatalf("delete contact method: %v", err)
	}
}
