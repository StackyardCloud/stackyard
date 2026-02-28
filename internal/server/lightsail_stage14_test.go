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

func TestLightsailStage14DomainsDNS(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lightsailRequest(t, ts, "CreateDomain", []byte(`{"domainName":"example.com"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "CreateDomainEntry", []byte(`{"domainName":"example.com","domainEntry":{"name":"www","type":"A","target":"198.51.100.10"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "GetDomain", []byte(`{"domainName":"example.com"}`))
	assertStatus(t, resp, http.StatusOK)
	var getDomainOut struct {
		Domain struct {
			Name          string `json:"name"`
			DomainEntries []struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Target string `json:"target"`
			} `json:"domainEntries"`
		} `json:"domain"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getDomainOut); err != nil {
		t.Fatalf("unmarshal GetDomain: %v", err)
	}
	if getDomainOut.Domain.Name != "example.com" || len(getDomainOut.Domain.DomainEntries) != 1 {
		t.Fatalf("unexpected GetDomain output: %+v", getDomainOut)
	}

	resp = lightsailRequest(t, ts, "GetDomains", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "UpdateDomainEntry", []byte(`{"domainName":"example.com","domainEntry":{"name":"www","type":"A","target":"198.51.100.11"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteDomainEntry", []byte(`{"domainName":"example.com","domainEntry":{"name":"www","type":"A"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = lightsailRequest(t, ts, "DeleteDomain", []byte(`{"domainName":"example.com"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestLightsailStage14SDKClientDomainsDNS(t *testing.T) {
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

	if _, err := client.CreateDomain(ctx, &awslightsail.CreateDomainInput{
		DomainName: aws.String("sdk-example.com"),
	}); err != nil {
		t.Fatalf("create domain: %v", err)
	}

	if _, err := client.CreateDomainEntry(ctx, &awslightsail.CreateDomainEntryInput{
		DomainName: aws.String("sdk-example.com"),
		DomainEntry: &awslightsailtypes.DomainEntry{
			Name:   aws.String("www"),
			Type:   aws.String("A"),
			Target: aws.String("198.51.100.10"),
		},
	}); err != nil {
		t.Fatalf("create domain entry: %v", err)
	}

	getDomainOut, err := client.GetDomain(ctx, &awslightsail.GetDomainInput{
		DomainName: aws.String("sdk-example.com"),
	})
	if err != nil {
		t.Fatalf("get domain: %v", err)
	}
	if getDomainOut.Domain == nil || getDomainOut.Domain.Name == nil || *getDomainOut.Domain.Name != "sdk-example.com" {
		t.Fatalf("unexpected get domain output: %+v", getDomainOut.Domain)
	}

	getDomainsOut, err := client.GetDomains(ctx, &awslightsail.GetDomainsInput{})
	if err != nil {
		t.Fatalf("get domains: %v", err)
	}
	if len(getDomainsOut.Domains) == 0 {
		t.Fatalf("expected at least one domain")
	}

	updateDomainOut, err := client.UpdateDomainEntry(ctx, &awslightsail.UpdateDomainEntryInput{
		DomainName: aws.String("sdk-example.com"),
		DomainEntry: &awslightsailtypes.DomainEntry{
			Name:   aws.String("www"),
			Type:   aws.String("A"),
			Target: aws.String("198.51.100.11"),
		},
	})
	if err != nil {
		t.Fatalf("update domain entry: %v", err)
	}
	if len(updateDomainOut.Operations) == 0 {
		t.Fatalf("expected update operations")
	}

	if _, err := client.DeleteDomainEntry(ctx, &awslightsail.DeleteDomainEntryInput{
		DomainName: aws.String("sdk-example.com"),
		DomainEntry: &awslightsailtypes.DomainEntry{
			Name: aws.String("www"),
			Type: aws.String("A"),
		},
	}); err != nil {
		t.Fatalf("delete domain entry: %v", err)
	}

	if _, err := client.DeleteDomain(ctx, &awslightsail.DeleteDomainInput{
		DomainName: aws.String("sdk-example.com"),
	}); err != nil {
		t.Fatalf("delete domain: %v", err)
	}
}
