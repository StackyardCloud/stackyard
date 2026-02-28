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

func TestLightsailStage6KeyPairs(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateKeyPair", []byte(`{"keyPairName":"stage6-key","tags":[{"key":"env","value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		KeyPair struct {
			Name        string `json:"name"`
			Fingerprint string `json:"fingerprint"`
		} `json:"keyPair"`
		PrivateKeyBase64 string `json:"privateKeyBase64"`
		PublicKeyBase64  string `json:"publicKeyBase64"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal CreateKeyPair: %v", err)
	}
	if createOut.KeyPair.Name != "stage6-key" || createOut.PrivateKeyBase64 == "" || createOut.PublicKeyBase64 == "" {
		t.Fatalf("unexpected CreateKeyPair output: %+v", createOut)
	}

	resp = lightsailRequest(t, ts, "GetKeyPair", []byte(`{"keyPairName":"stage6-key"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetKeyPairs", []byte(`{"includeDefaultKeyPair":false}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		KeyPairs []struct {
			Name string `json:"name"`
		} `json:"keyPairs"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal GetKeyPairs: %v", err)
	}
	if len(listOut.KeyPairs) != 1 {
		t.Fatalf("expected one key pair before import, got %d", len(listOut.KeyPairs))
	}

	resp = lightsailRequest(t, ts, "ImportKeyPair", []byte(`{"keyPairName":"stage6-imported","publicKeyBase64":"c3NoLXJzYSBTVEFDS1lBUkQtaW1wb3J0ZWQ="}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DownloadDefaultKeyPair", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var defaultOut struct {
		CreatedAt        float64 `json:"createdAt"`
		PrivateKeyBase64 string  `json:"privateKeyBase64"`
		PublicKeyBase64  string  `json:"publicKeyBase64"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &defaultOut); err != nil {
		t.Fatalf("unmarshal DownloadDefaultKeyPair: %v", err)
	}
	if defaultOut.CreatedAt == 0 || defaultOut.PrivateKeyBase64 == "" || defaultOut.PublicKeyBase64 == "" {
		t.Fatalf("unexpected DownloadDefaultKeyPair output: %+v", defaultOut)
	}

	resp = lightsailRequest(t, ts, "GetKeyPairs", []byte(`{"includeDefaultKeyPair":true}`))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal GetKeyPairs include default: %v", err)
	}
	if len(listOut.KeyPairs) != 3 {
		t.Fatalf("expected three key pairs including default, got %d", len(listOut.KeyPairs))
	}

	resp = lightsailRequest(t, ts, "GetKeyPair", []byte(`{"keyPairName":"LightsailDefaultKeyPair"}`))
	assertStatus(t, resp, http.StatusOK)
	var getDefaultOut struct {
		KeyPair struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"keyPair"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getDefaultOut); err != nil {
		t.Fatalf("unmarshal GetKeyPair default: %v", err)
	}
	if getDefaultOut.KeyPair.Fingerprint == "" {
		t.Fatalf("expected default key pair fingerprint")
	}

	resp = lightsailRequest(t, ts, "DeleteKeyPair", []byte(`{"keyPairName":"stage6-key"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = lightsailRequest(t, ts, "DeleteKeyPair", []byte(`{"keyPairName":"LightsailDefaultKeyPair","expectedFingerprint":"`+getDefaultOut.KeyPair.Fingerprint+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage6SDKClientKeyPairs(t *testing.T) {
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

	createOut, err := client.CreateKeyPair(ctx, &awslightsail.CreateKeyPairInput{
		KeyPairName: aws.String("sdk-stage6-key"),
		Tags: []awslightsailtypes.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	})
	if err != nil {
		t.Fatalf("create key pair: %v", err)
	}
	if createOut.KeyPair == nil || aws.ToString(createOut.KeyPair.Name) != "sdk-stage6-key" {
		t.Fatalf("unexpected CreateKeyPair output")
	}
	if aws.ToString(createOut.PrivateKeyBase64) == "" || aws.ToString(createOut.PublicKeyBase64) == "" {
		t.Fatalf("expected key material in create response")
	}

	if _, err := client.GetKeyPair(ctx, &awslightsail.GetKeyPairInput{KeyPairName: aws.String("sdk-stage6-key")}); err != nil {
		t.Fatalf("get key pair: %v", err)
	}

	keyPairsOut, err := client.GetKeyPairs(ctx, &awslightsail.GetKeyPairsInput{IncludeDefaultKeyPair: aws.Bool(false)})
	if err != nil {
		t.Fatalf("get key pairs: %v", err)
	}
	if len(keyPairsOut.KeyPairs) != 1 {
		t.Fatalf("expected one key pair before import, got %d", len(keyPairsOut.KeyPairs))
	}

	if _, err := client.ImportKeyPair(ctx, &awslightsail.ImportKeyPairInput{
		KeyPairName:     aws.String("sdk-stage6-imported"),
		PublicKeyBase64: aws.String("c3NoLXJzYSBTVEFDS1lBUkQtaW1wb3J0ZWQ="),
	}); err != nil {
		t.Fatalf("import key pair: %v", err)
	}

	downloadOut, err := client.DownloadDefaultKeyPair(ctx, &awslightsail.DownloadDefaultKeyPairInput{})
	if err != nil {
		t.Fatalf("download default key pair: %v", err)
	}
	if downloadOut.CreatedAt == nil || aws.ToString(downloadOut.PrivateKeyBase64) == "" || aws.ToString(downloadOut.PublicKeyBase64) == "" {
		t.Fatalf("unexpected DownloadDefaultKeyPair output")
	}

	keyPairsOut, err = client.GetKeyPairs(ctx, &awslightsail.GetKeyPairsInput{IncludeDefaultKeyPair: aws.Bool(true)})
	if err != nil {
		t.Fatalf("get key pairs include default: %v", err)
	}
	if len(keyPairsOut.KeyPairs) != 3 {
		t.Fatalf("expected three key pairs including default, got %d", len(keyPairsOut.KeyPairs))
	}

	defaultOut, err := client.GetKeyPair(ctx, &awslightsail.GetKeyPairInput{KeyPairName: aws.String("LightsailDefaultKeyPair")})
	if err != nil {
		t.Fatalf("get default key pair: %v", err)
	}
	if defaultOut.KeyPair == nil || aws.ToString(defaultOut.KeyPair.Fingerprint) == "" {
		t.Fatalf("expected default key pair fingerprint")
	}

	if _, err := client.DeleteKeyPair(ctx, &awslightsail.DeleteKeyPairInput{KeyPairName: aws.String("sdk-stage6-key")}); err != nil {
		t.Fatalf("delete key pair: %v", err)
	}
	if _, err := client.DeleteKeyPair(ctx, &awslightsail.DeleteKeyPairInput{
		KeyPairName:         aws.String("LightsailDefaultKeyPair"),
		ExpectedFingerprint: defaultOut.KeyPair.Fingerprint,
	}); err != nil {
		t.Fatalf("delete default key pair: %v", err)
	}
}
