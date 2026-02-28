package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awssesv2 "github.com/aws/aws-sdk-go-v2/service/sesv2"
)

func sesv2Request(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "ses")
}

func TestSESV2Stage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sesv2Request(t, ts, http.MethodPost, "/v2/email/configuration-sets", []byte(`{"ConfigurationSetName":"demo-config"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/identities", []byte(`{"EmailIdentity":"sender@example.com","ConfigurationSetName":"demo-config"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/identities/sender%40example.com", nil)
	assertStatus(t, resp, http.StatusOK)
	var identityOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &identityOut); err != nil {
		t.Fatalf("unmarshal get identity: %v", err)
	}
	if identityOut["IdentityType"] == "" {
		t.Fatalf("expected IdentityType in GetEmailIdentity response")
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/identities", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/templates", []byte(`{"TemplateName":"welcome","TemplateContent":{"Subject":"Hi {{name}}","Text":"Hello {{name}}","Html":"<h1>Hello {{name}}</h1>"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/templates/welcome", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/templates/welcome/render", []byte(`{"TemplateData":"{\"name\":\"Stackyard\"}"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/outbound-emails", []byte(`{"FromEmailAddress":"sender@example.com","Destination":{"ToAddresses":["recipient@example.com"]},"Content":{"Simple":{"Subject":{"Data":"Hello"},"Body":{"Text":{"Data":"Hello from Stackyard"}}}}}`))
	assertStatus(t, resp, http.StatusOK)
	var sendOut struct {
		MessageID string `json:"MessageId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &sendOut); err != nil {
		t.Fatalf("unmarshal send email: %v", err)
	}
	if sendOut.MessageID == "" {
		t.Fatalf("expected message id")
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/outbound-bulk-emails", []byte(`{"FromEmailAddress":"sender@example.com","DefaultContent":{"Simple":{"Subject":{"Data":"bulk"},"Body":{"Text":{"Data":"body"}}}},"BulkEmailEntries":[{"Destination":{"ToAddresses":["recipient@example.com"]}},{"Destination":{"ToAddresses":["recipient2@example.com"]}}]}`))
	assertStatus(t, resp, http.StatusOK)
	var bulkOut struct {
		Results []map[string]any `json:"BulkEmailEntryResults"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &bulkOut); err != nil {
		t.Fatalf("unmarshal send bulk email: %v", err)
	}
	if len(bulkOut.Results) != 2 {
		t.Fatalf("expected 2 bulk results, got %d", len(bulkOut.Results))
	}

	resp = sesv2Request(t, ts, http.MethodPut, "/v2/email/suppression/addresses", []byte(`{"EmailAddress":"recipient@example.com","Reason":"BOUNCE"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/suppression/addresses", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/suppression/addresses/recipient%40example.com", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodDelete, "/v2/email/suppression/addresses/recipient%40example.com", nil)
	assertStatus(t, resp, http.StatusOK)

	resourceARN := url.QueryEscape("arn:aws:ses:us-east-1:123456789012:configuration-set/demo-config")
	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/tags", []byte(`{"ResourceArn":"arn:aws:ses:us-east-1:123456789012:configuration-set/demo-config","Tags":[{"Key":"env","Value":"test"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/tags?ResourceArn="+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodDelete, "/v2/email/tags?ResourceArn="+resourceARN, []byte(`{"TagKeys":["env"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/contact-lists", []byte(`{"ContactListName":"audience"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/contact-lists/audience/contacts", []byte(`{"EmailAddress":"recipient@example.com","AttributesData":"{\"tier\":\"silver\"}"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/contact-lists/audience/contacts/list", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/contact-lists/audience/contacts/recipient%40example.com", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPut, "/v2/email/contact-lists/audience/contacts/recipient%40example.com", []byte(`{"AttributesData":"{\"tier\":\"gold\"}"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodDelete, "/v2/email/contact-lists/audience/contacts/recipient%40example.com", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodDelete, "/v2/email/contact-lists/audience", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPut, "/v2/email/account/sending", []byte(`{"SendingEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/outbound-emails", []byte(`{"FromEmailAddress":"sender@example.com","Destination":{"ToAddresses":["recipient@example.com"]},"Content":{"Simple":{"Subject":{"Data":"Disabled"},"Body":{"Text":{"Data":"Disabled"}}}}}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "SendingPausedException") {
		t.Fatalf("expected SendingPausedException")
	}
}

func TestSESV2Stage0OperationCoverage(t *testing.T) {
	if len(sesv2Operations) != 110 {
		t.Fatalf("expected 110 SESv2 operations from docs, got %d", len(sesv2Operations))
	}
	seen := map[string]bool{}
	for _, op := range sesv2Operations {
		if seen[op.Name] {
			t.Fatalf("duplicate operation: %s", op.Name)
		}
		seen[op.Name] = true

		concretePath := sesv2ConcretePathForTest(op.Pattern)
		matched, _, ok := matchSESV2Operation(op.Method, concretePath)
		if !ok {
			t.Fatalf("operation did not match by method/path: %s %s", op.Method, op.Pattern)
		}
		if matched.Name != op.Name {
			t.Fatalf("matched wrong operation for %s %s: got %s", op.Method, op.Pattern, matched.Name)
		}
	}

	required := []string{
		"CreateMultiRegionEndpoint",
		"CreateTenant",
		"CreateTenantResourceAssociation",
		"DeleteMultiRegionEndpoint",
		"DeleteTenant",
		"DeleteTenantResourceAssociation",
		"GetEmailAddressInsights",
		"GetMultiRegionEndpoint",
		"GetReputationEntity",
		"GetTenant",
		"ListMultiRegionEndpoints",
		"ListReputationEntities",
		"ListResourceTenants",
		"ListTenantResources",
		"ListTenants",
		"PutConfigurationSetArchivingOptions",
		"UpdateReputationEntityCustomerManagedStatus",
		"UpdateReputationEntityPolicy",
	}
	for _, name := range required {
		if _, ok := sesv2OperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestSESV2Stage0SigV4PathEncodingAtSign(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sesv2Request(t, ts, http.MethodPost, "/v2/email/identities", []byte(`{"EmailIdentity":"sender@example.com"}`))
	assertStatus(t, resp, http.StatusOK)

	body := []byte(`{"EmailForwardingEnabled":true}`)
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/v2/email/identities/sender%40example.com/feedback", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = req.URL.Host
	req.Header.Set("Content-Type", "application/json")
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", amzDate)
	payloadHash := sha256Hex(body)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	signedHeaders := buildSignedHeaders(req)
	canonicalHeaders, err := buildCanonicalHeaders(req, signedHeaders)
	if err != nil {
		t.Fatalf("build canonical headers: %v", err)
	}
	canonicalRequest := strings.Join([]string{
		req.Method,
		"/v2/email/identities/sender%40example.com/feedback",
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	stringToSign := buildStringToSign(amzDate, testRegion, "ses", canonicalRequest)
	signature := signString(stringToSign, testRegion, "ses", testSecretKey)
	credentialScope := amzDate[:8] + "/" + testRegion + "/ses/aws4_request"
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/"+credentialScope+", SignedHeaders="+signedHeaders+", Signature="+signature)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)
}

func TestSESV2Stage0SDKClientFeedbackAttributesSignature(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == awssesv2.ServiceID {
			return aws.Endpoint{
				URL:               ts.URL,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	client := awssesv2.NewFromConfig(cfg)

	if _, err := client.CreateConfigurationSet(ctx, &awssesv2.CreateConfigurationSetInput{
		ConfigurationSetName: aws.String("demo-config"),
	}); err != nil {
		t.Fatalf("create configuration set: %v", err)
	}
	if _, err := client.CreateEmailIdentity(ctx, &awssesv2.CreateEmailIdentityInput{
		EmailIdentity: aws.String("sender@example.com"),
	}); err != nil {
		t.Fatalf("create email identity: %v", err)
	}
	if _, err := client.PutEmailIdentityFeedbackAttributes(ctx, &awssesv2.PutEmailIdentityFeedbackAttributesInput{
		EmailIdentity:          aws.String("sender@example.com"),
		EmailForwardingEnabled: true,
	}); err != nil {
		t.Fatalf("put feedback attributes: %v", err)
	}
}

func TestSESV2Stage0SDKClientPutAccountSendingAttributesFalse(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...any) (aws.Endpoint, error) {
		if service == awssesv2.ServiceID {
			return aws.Endpoint{
				URL:               ts.URL,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	client := awssesv2.NewFromConfig(cfg)

	if _, err := client.PutAccountSendingAttributes(ctx, &awssesv2.PutAccountSendingAttributesInput{
		SendingEnabled: false,
	}); err != nil {
		t.Fatalf("put account sending attributes false: %v", err)
	}
	out, err := client.GetAccount(ctx, &awssesv2.GetAccountInput{})
	if err != nil {
		t.Fatalf("get account after false: %v", err)
	}
	if out.SendingEnabled {
		t.Fatalf("expected sending to be disabled")
	}

	if _, err := client.PutAccountSendingAttributes(ctx, &awssesv2.PutAccountSendingAttributesInput{
		SendingEnabled: true,
	}); err != nil {
		t.Fatalf("put account sending attributes true: %v", err)
	}
	out, err = client.GetAccount(ctx, &awssesv2.GetAccountInput{})
	if err != nil {
		t.Fatalf("get account after true: %v", err)
	}
	if !out.SendingEnabled {
		t.Fatalf("expected sending to be enabled")
	}
}

func sesv2ConcretePathForTest(pattern string) string {
	replacements := map[string]string{
		"{JobId}":                     "job-1",
		"{ConfigurationSetName}":      "demo-config",
		"{ContactListName}":           "audience",
		"{EmailIdentity}":             "sender%40example.com",
		"{PolicyName}":                "policy-default",
		"{TemplateName}":              "welcome",
		"{EventDestinationName}":      "event-dest",
		"{EmailAddress}":              "recipient%40example.com",
		"{PoolName}":                  "pool-1",
		"{EndpointName}":              "global-endpoint",
		"{Ip}":                        "1.2.3.4",
		"{ReportId}":                  "report-1",
		"{CampaignId}":                "campaign-1",
		"{Domain}":                    "example.com",
		"{MessageId}":                 "message-1",
		"{SubscribedDomain}":          "example.com",
		"{ReputationEntityType}":      "DOMAIN",
		"{ReputationEntityReference}": "example.com",
	}
	for old, newValue := range replacements {
		pattern = strings.ReplaceAll(pattern, old, newValue)
	}
	return normalizeSESV2Path(pattern)
}
