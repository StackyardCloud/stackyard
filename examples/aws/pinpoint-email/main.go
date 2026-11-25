// Command pinpoint-email runs Amazon Pinpoint Email against Stackyard: verify
// a domain, create a dedicated IP pool and a configuration set that sends
// from it with an SNS event destination, send an email, read the
// deliverability dashboard's statistics and campaigns, and run a
// deliverability test.
//
// Pinpoint Email shares identities and configuration sets with SESv2. The
// SDK has no Pinpoint Email module in this repository's go.mod, so requests
// are signed by hand.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const signingName = "ses"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Pinpoint Email client using %s\n", c.endpoint)
	run := time.Now().UnixNano()
	pool, configSet := fmt.Sprintf("transactional-%d", run), fmt.Sprintf("receipts-%d", run)

	id := c.rest("CreateEmailIdentity", http.MethodPost, "/v1/email/identities", map[string]any{"EmailIdentity": "example.com"})
	fmt.Printf("identity example.com: %v, verified for sending %v\n", id["IdentityType"], id["VerifiedForSendingStatus"])

	c.rest("CreateDedicatedIpPool", http.MethodPost, "/v1/email/dedicated-ip-pools", map[string]any{"PoolName": pool})
	ips := c.rest("GetDedicatedIps", http.MethodGet, "/v1/email/dedicated-ips", nil)["DedicatedIps"].([]any)
	ip := ips[0].(map[string]any)["Ip"].(string)
	c.rest("PutDedicatedIpInPool", http.MethodPut, "/v1/email/dedicated-ips/"+ip+"/pool", map[string]any{"DestinationPoolName": pool})
	fmt.Printf("dedicated IP %s moved to pool %s\n", ip, pool)

	c.rest("CreateConfigurationSet", http.MethodPost, "/v1/email/configuration-sets", map[string]any{
		"ConfigurationSetName": configSet,
		"DeliveryOptions":      map[string]any{"SendingPoolName": pool, "TlsPolicy": "REQUIRE"},
		"ReputationOptions":    map[string]any{"ReputationMetricsEnabled": true},
		"Tags":                 []any{map[string]any{"Key": "team", "Value": "billing"}},
	})
	c.rest("CreateConfigurationSetEventDestination", http.MethodPost, "/v1/email/configuration-sets/"+configSet+"/event-destinations", map[string]any{
		"EventDestinationName": "bounces",
		"EventDestination": map[string]any{
			"Enabled": true, "MatchingEventTypes": []string{"BOUNCE", "COMPLAINT"},
			"SnsDestination": map[string]any{"TopicArn": "arn:aws:sns:us-east-1:123456789012:mail-feedback"},
		},
	})
	cfg := c.rest("GetConfigurationSet", http.MethodGet, "/v1/email/configuration-sets/"+configSet, nil)
	fmt.Printf("configuration set %s sends from %v\n", configSet, cfg["DeliveryOptions"].(map[string]any)["SendingPoolName"])

	sent := c.rest("SendEmail", http.MethodPost, "/v1/email/outbound-emails", map[string]any{
		"FromEmailAddress": "billing@example.com", "ConfigurationSetName": configSet,
		"Destination": map[string]any{"ToAddresses": []string{"customer@example.org", "accounts@example.org"}},
		"Content": map[string]any{"Simple": map[string]any{
			"Subject": map[string]any{"Data": "Your receipt"}, "Body": map[string]any{"Text": map[string]any{"Data": "Thank you for your order."}},
		}},
	})
	fmt.Printf("sent message %v\n", sent["MessageId"])

	c.rest("PutDeliverabilityDashboardOption", http.MethodPut, "/v1/email/deliverability-dashboard", map[string]any{
		"DashboardEnabled": true, "SubscribedDomains": []any{map[string]any{"Domain": "example.com"}},
	})
	window := "StartDate=" + url.QueryEscape(time.Now().Add(-24*time.Hour).UTC().Format(time.RFC3339)) +
		"&EndDate=" + url.QueryEscape(time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	stats := c.rest("GetDomainStatisticsReport", http.MethodGet, "/v1/email/deliverability-dashboard/statistics-report/example.com?"+window, nil)
	volume := stats["OverallVolume"].(map[string]any)["VolumeStatistics"].(map[string]any)
	fmt.Printf("example.com inbox volume in the last day: %v\n", volume["InboxRawCount"])
	camps := c.rest("ListDomainDeliverabilityCampaigns", http.MethodGet, "/v1/email/deliverability-dashboard/domains/example.com/campaigns?"+window, nil)
	for _, item := range camps["DomainDeliverabilityCampaigns"].([]any) {
		camp := item.(map[string]any)
		fmt.Printf("campaign %q: %v recipients\n", camp["Subject"], camp["InboxCount"])
	}

	report := c.rest("CreateDeliverabilityTestReport", http.MethodPost, "/v1/email/deliverability-dashboard/test", map[string]any{
		"ReportName": "receipt-check", "FromEmailAddress": "billing@example.com",
		"Content": map[string]any{"Simple": map[string]any{
			"Subject": map[string]any{"Data": "Your receipt"}, "Body": map[string]any{"Html": map[string]any{"Data": "<p>Thank you</p>"}},
		}},
	})
	result := c.rest("GetDeliverabilityTestReport", http.MethodGet, "/v1/email/deliverability-dashboard/test-reports/"+report["ReportId"].(string), nil)
	fmt.Printf("test report %v: inbox placement %v%%\n",
		result["DeliverabilityTestReport"].(map[string]any)["DeliverabilityTestStatus"], result["OverallPlacement"].(map[string]any)["InboxPercentage"])

	c.rest("DeleteConfigurationSet", http.MethodDelete, "/v1/email/configuration-sets/"+configSet, nil)
	c.rest("DeleteDedicatedIpPool", http.MethodDelete, "/v1/email/dedicated-ip-pools/"+pool, nil)
	fmt.Println("cleaned up")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// rest sends one REST JSON operation and exits unless it returns 2xx.
func (c client) rest(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	return decode(op, c.send(op, method, signingName, c.endpoint+path, body, map[string]string{"Content-Type": "application/json"}))
}

func decode(op string, raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
