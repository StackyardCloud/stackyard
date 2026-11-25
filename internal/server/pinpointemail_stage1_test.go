package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func ppeCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	return msgREST(t, ts, "ses", method, path, body)
}

func ppeOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	status, out := ppeCall(t, ts, method, path, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func ppeFail(t *testing.T, ts *httptest.Server, method, path, body string, wantStatus int, wantCode string) {
	t.Helper()
	status, out := ppeCall(t, ts, method, path, body)
	if status != wantStatus || out["__type"] != wantCode {
		t.Fatalf("%s %s: want %d %s, got %d %v", method, path, wantStatus, wantCode, status, out)
	}
}

// Identities and configuration sets are SESv2's: each API sees the other's.
func TestPinpointEmailStage1SharedWithSESv2(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	out := ppeOK(t, ts, http.MethodPost, "/v1/email/identities", `{"EmailIdentity":"example.com","Tags":[{"Key":"team","Value":"mail"}]}`)
	if out["IdentityType"] != "DOMAIN" || out["DkimAttributes"].(map[string]any)["Tokens"] == nil {
		t.Fatalf("CreateEmailIdentity: %v", out)
	}
	ppeFail(t, ts, http.MethodPost, "/v1/email/identities", `{"EmailIdentity":"not an identity"}`, http.StatusBadRequest, "BadRequestException")
	if status, got := msgREST(t, ts, "ses", http.MethodGet, "/v2/email/identities/example.com", ""); status != http.StatusOK || got["IdentityType"] != "DOMAIN" {
		t.Fatalf("SESv2 sees the identity: %d %v", status, got)
	}
	msgREST(t, ts, "ses", http.MethodPost, "/v2/email/configuration-sets", `{"ConfigurationSetName":"from-v2"}`)
	got := ppeOK(t, ts, http.MethodGet, "/v1/email/configuration-sets/from-v2", "")
	if got["SendingOptions"].(map[string]any)["SendingEnabled"] != true {
		t.Fatalf("defaults for a SESv2 configuration set: %v", got)
	}

	ppeOK(t, ts, http.MethodPut, "/v1/email/identities/example.com/mail-from", `{"MailFromDomain":"bounce.example.com","BehaviorOnMxFailure":"REJECT_MESSAGE"}`)
	ppeFail(t, ts, http.MethodPut, "/v1/email/identities/example.com/mail-from", `{"MailFromDomain":"bounce.other.com"}`, http.StatusBadRequest, "BadRequestException")
	ppeOK(t, ts, http.MethodPut, "/v1/email/identities/example.com/dkim", `{"SigningEnabled":false}`)
	ppeOK(t, ts, http.MethodPut, "/v1/email/identities/example.com/feedback", `{"EmailForwardingEnabled":false}`)
	ident := ppeOK(t, ts, http.MethodGet, "/v1/email/identities/example.com", "")
	mf := ident["MailFromAttributes"].(map[string]any)
	if mf["MailFromDomain"] != "bounce.example.com" || mf["BehaviorOnMxFailure"] != "REJECT_MESSAGE" || ident["FeedbackForwardingStatus"] != false ||
		ident["DkimAttributes"].(map[string]any)["SigningEnabled"] != false || len(ident["Tags"].([]any)) != 1 {
		t.Fatalf("GetEmailIdentity: %v", ident)
	}
	if ids := ppeOK(t, ts, http.MethodGet, "/v1/email/identities", "")["EmailIdentities"].([]any); len(ids) != 1 {
		t.Fatalf("ListEmailIdentities: %v", ids)
	}

	// Configuration set options, pools and event destinations.
	ppeFail(t, ts, http.MethodPost, "/v1/email/configuration-sets", `{"ConfigurationSetName":"marketing","DeliveryOptions":{"SendingPoolName":"missing"}}`, http.StatusNotFound, "NotFoundException")
	ppeOK(t, ts, http.MethodPost, "/v1/email/dedicated-ip-pools", `{"PoolName":"marketing-ips"}`)
	ppeFail(t, ts, http.MethodPost, "/v1/email/dedicated-ip-pools", `{"PoolName":"marketing-ips"}`, http.StatusBadRequest, "AlreadyExistsException")
	ppeOK(t, ts, http.MethodPost, "/v1/email/configuration-sets", `{"ConfigurationSetName":"marketing","DeliveryOptions":{"SendingPoolName":"marketing-ips","TlsPolicy":"REQUIRE"},
		"ReputationOptions":{"ReputationMetricsEnabled":true},"TrackingOptions":{"CustomRedirectDomain":"click.example.com"},"Tags":[{"Key":"env","Value":"test"}]}`)
	ppeFail(t, ts, http.MethodPost, "/v1/email/configuration-sets", `{"ConfigurationSetName":"marketing"}`, http.StatusBadRequest, "AlreadyExistsException")
	ppeOK(t, ts, http.MethodPut, "/v1/email/configuration-sets/marketing/sending", `{"SendingEnabled":false}`)
	got = ppeOK(t, ts, http.MethodGet, "/v1/email/configuration-sets/marketing", "")
	if got["DeliveryOptions"].(map[string]any)["SendingPoolName"] != "marketing-ips" || got["ReputationOptions"].(map[string]any)["LastFreshStart"] == nil ||
		got["SendingOptions"].(map[string]any)["SendingEnabled"] != false || got["TrackingOptions"].(map[string]any)["CustomRedirectDomain"] != "click.example.com" ||
		len(got["Tags"].([]any)) != 1 {
		t.Fatalf("GetConfigurationSet: %v", got)
	}
	ppeOK(t, ts, http.MethodPut, "/v1/email/dedicated-ips/198.51.100.10/pool", `{"DestinationPoolName":"marketing-ips"}`)
	ppeOK(t, ts, http.MethodPut, "/v1/email/dedicated-ips/198.51.100.10/warmup", `{"WarmupPercentage":40}`)
	ppeFail(t, ts, http.MethodPut, "/v1/email/dedicated-ips/198.51.100.10/warmup", `{"WarmupPercentage":140}`, http.StatusBadRequest, "BadRequestException")
	ip := ppeOK(t, ts, http.MethodGet, "/v1/email/dedicated-ips/198.51.100.10", "")["DedicatedIp"].(map[string]any)
	if ip["PoolName"] != "marketing-ips" || ip["WarmupStatus"] != "IN_PROGRESS" || ip["WarmupPercentage"] != 40.0 {
		t.Fatalf("GetDedicatedIp: %v", ip)
	}
	if ips := ppeOK(t, ts, http.MethodGet, "/v1/email/dedicated-ips?PoolName=marketing-ips", "")["DedicatedIps"].([]any); len(ips) != 1 {
		t.Fatalf("GetDedicatedIps in pool: %v", ips)
	}
	ppeFail(t, ts, http.MethodGet, "/v1/email/dedicated-ips/203.0.113.1", "", http.StatusNotFound, "NotFoundException")
	ppeFail(t, ts, http.MethodGet, "/v1/email/dedicated-ips?PageSize=abc", "", http.StatusBadRequest, "BadRequestException")
	ppeOK(t, ts, http.MethodDelete, "/v1/email/dedicated-ip-pools/marketing-ips", "")
	if got := ppeOK(t, ts, http.MethodGet, "/v1/email/configuration-sets/marketing", ""); got["DeliveryOptions"].(map[string]any)["SendingPoolName"] != nil {
		t.Fatalf("deleted pool still assigned: %v", got)
	}

	dest := `{"EventDestinationName":"to-sns","EventDestination":{"Enabled":true,"MatchingEventTypes":["SEND","BOUNCE"],"SnsDestination":{"TopicArn":"arn:aws:sns:us-east-1:123456789012:mail"}}}`
	ppeOK(t, ts, http.MethodPost, "/v1/email/configuration-sets/marketing/event-destinations", dest)
	ppeFail(t, ts, http.MethodPost, "/v1/email/configuration-sets/marketing/event-destinations", dest, http.StatusBadRequest, "AlreadyExistsException")
	ppeFail(t, ts, http.MethodPost, "/v1/email/configuration-sets/marketing/event-destinations", `{"EventDestinationName":"none","EventDestination":{"MatchingEventTypes":["SEND"]}}`, http.StatusBadRequest, "BadRequestException")
	ppeFail(t, ts, http.MethodPut, "/v1/email/configuration-sets/marketing/event-destinations/missing", `{"EventDestination":{"MatchingEventTypes":["SEND"],"SnsDestination":{"TopicArn":"arn:aws:sns:us-east-1:123456789012:x"}}}`, http.StatusNotFound, "NotFoundException")
	dests := ppeOK(t, ts, http.MethodGet, "/v1/email/configuration-sets/marketing/event-destinations", "")["EventDestinations"].([]any)
	if len(dests) != 1 || dests[0].(map[string]any)["Name"] != "to-sns" || dests[0].(map[string]any)["SnsDestination"] == nil {
		t.Fatalf("event destinations: %v", dests)
	}
	// SESv2 sees the destination Pinpoint Email created.
	if _, v2 := msgREST(t, ts, "ses", http.MethodGet, "/v2/email/configuration-sets/marketing/event-destinations", ""); len(v2["EventDestinations"].([]any)) != 1 {
		t.Fatalf("SESv2 event destinations: %v", v2)
	}
	ppeOK(t, ts, http.MethodDelete, "/v1/email/configuration-sets/marketing/event-destinations/to-sns", "")

	// Tags live on resource ARNs and require the resource.
	arn := url.QueryEscape(ppeARNBase + "configuration-set/marketing")
	ppeOK(t, ts, http.MethodPost, "/v1/email/tags", fmt.Sprintf(`{"ResourceArn":%q,"Tags":[{"Key":"owner","Value":"growth"}]}`, ppeARNBase+"configuration-set/marketing"))
	ppeOK(t, ts, http.MethodDelete, "/v1/email/tags?ResourceArn="+arn+"&TagKeys=env", "")
	if tags := ppeOK(t, ts, http.MethodGet, "/v1/email/tags?ResourceArn="+arn, "")["Tags"].([]any); len(tags) != 1 || tags[0].(map[string]any)["Key"] != "owner" {
		t.Fatalf("tags: %v", tags)
	}
	ppeFail(t, ts, http.MethodGet, "/v1/email/tags?ResourceArn="+url.QueryEscape(ppeARNBase+"configuration-set/missing"), "", http.StatusNotFound, "NotFoundException")
	ppeOK(t, ts, http.MethodDelete, "/v1/email/configuration-sets/marketing", "")
	ppeFail(t, ts, http.MethodGet, "/v1/email/configuration-sets/marketing", "", http.StatusNotFound, "NotFoundException")
	ppeOK(t, ts, http.MethodDelete, "/v1/email/identities/example.com", "")
	ppeFail(t, ts, http.MethodGet, "/v1/email/identities/example.com", "", http.StatusNotFound, "NotFoundException")
}

func TestPinpointEmailStage1SendingAndDashboard(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	ppeOK(t, ts, http.MethodPost, "/v1/email/identities", `{"EmailIdentity":"example.com"}`)
	send := func(body string) (int, map[string]any) {
		return ppeCall(t, ts, http.MethodPost, "/v1/email/outbound-emails", body)
	}
	simple := `{"Simple":{"Subject":{"Data":"Spring sale"},"Body":{"Text":{"Data":"20% off"}}}}`
	if status, out := send(`{"FromEmailAddress":"news@example.com","Destination":{"ToAddresses":["a@example.org","b@example.org"]},"Content":` + simple + `}`); status != http.StatusOK || out["MessageId"] == nil {
		t.Fatalf("SendEmail: %d %v", status, out)
	}
	ppeFail(t, ts, http.MethodPost, "/v1/email/outbound-emails", `{"FromEmailAddress":"x@unverified.com","Destination":{"ToAddresses":["a@example.org"]},"Content":`+simple+`}`, http.StatusBadRequest, "MessageRejected")
	ppeFail(t, ts, http.MethodPost, "/v1/email/outbound-emails", `{"FromEmailAddress":"news@example.com","Destination":{},"Content":`+simple+`}`, http.StatusBadRequest, "BadRequestException")
	ppeFail(t, ts, http.MethodPost, "/v1/email/outbound-emails", `{"FromEmailAddress":"news@example.com","Destination":{"ToAddresses":["a@example.org"]},"Content":{}}`, http.StatusBadRequest, "BadRequestException")
	ppeFail(t, ts, http.MethodPost, "/v1/email/outbound-emails", `{"FromEmailAddress":"news@example.com","Destination":{"ToAddresses":["a@example.org"]},"ConfigurationSetName":"missing","Content":`+simple+`}`, http.StatusNotFound, "NotFoundException")
	ppeOK(t, ts, http.MethodPost, "/v1/email/configuration-sets", `{"ConfigurationSetName":"paused","SendingOptions":{"SendingEnabled":false}}`)
	ppeFail(t, ts, http.MethodPost, "/v1/email/outbound-emails", `{"FromEmailAddress":"news@example.com","Destination":{"ToAddresses":["a@example.org"]},"ConfigurationSetName":"paused","Content":`+simple+`}`, http.StatusBadRequest, "SendingPausedException")
	raw := base64.StdEncoding.EncodeToString([]byte("From: news@example.com\r\nTo: c@example.org\r\nSubject: Raw hello\r\n\r\nbody\r\n"))
	if status, out := send(`{"Destination":{},"Content":{"Raw":{"Data":"` + raw + `"}}}`); status != http.StatusOK {
		t.Fatalf("raw SendEmail: %d %v", status, out)
	}
	acct := ppeOK(t, ts, http.MethodGet, "/v1/email/account", "")
	if acct["SendQuota"].(map[string]any)["SentLast24Hours"] != 2.0 || acct["EnforcementStatus"] != "HEALTHY" {
		t.Fatalf("GetAccount: %v", acct)
	}
	ppeOK(t, ts, http.MethodPut, "/v1/email/account/sending", `{"SendingEnabled":false}`)
	ppeFail(t, ts, http.MethodPost, "/v1/email/outbound-emails", `{"FromEmailAddress":"news@example.com","Destination":{"ToAddresses":["a@example.org"]},"Content":`+simple+`}`, http.StatusBadRequest, "SendingPausedException")
	ppeOK(t, ts, http.MethodPut, "/v1/email/account/sending", `{"SendingEnabled":true}`)
	ppeOK(t, ts, http.MethodPut, "/v1/email/account/dedicated-ips/warmup", `{"AutoWarmupEnabled":true}`)
	if acct := ppeOK(t, ts, http.MethodGet, "/v1/email/account", ""); acct["DedicatedIpAutoWarmupEnabled"] != true {
		t.Fatalf("auto warmup: %v", acct)
	}

	// The dashboard tracks verified domains; statistics and campaigns come
	// from what was sent.
	window := fmt.Sprintf("StartDate=%s&EndDate=%s", url.QueryEscape(time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)), url.QueryEscape(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)))
	ppeFail(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/statistics-report/example.com?"+window, "", http.StatusNotFound, "NotFoundException")
	ppeFail(t, ts, http.MethodPut, "/v1/email/deliverability-dashboard", `{"DashboardEnabled":true,"SubscribedDomains":[{"Domain":"other.com"}]}`, http.StatusNotFound, "NotFoundException")
	ppeOK(t, ts, http.MethodPut, "/v1/email/deliverability-dashboard", `{"DashboardEnabled":true,"SubscribedDomains":[{"Domain":"example.com","InboxPlacementTrackingOption":{"Global":true}}]}`)
	dash := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard", "")
	if dash["AccountStatus"] != "ACTIVE" || len(dash["ActiveSubscribedDomains"].([]any)) != 1 {
		t.Fatalf("dashboard: %v", dash)
	}
	stats := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/statistics-report/example.com?"+window, "")
	if stats["OverallVolume"].(map[string]any)["VolumeStatistics"].(map[string]any)["InboxRawCount"] != 3.0 {
		t.Fatalf("statistics: %v", stats)
	}
	camps := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/domains/example.com/campaigns?"+window, "")["DomainDeliverabilityCampaigns"].([]any)
	if len(camps) != 2 {
		t.Fatalf("campaigns: %v", camps)
	}
	id := camps[0].(map[string]any)["CampaignId"].(string)
	if c := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/campaigns/"+id, "")["DomainDeliverabilityCampaign"].(map[string]any); c["Subject"] != "Spring sale" || c["InboxCount"] != 2.0 {
		t.Fatalf("campaign: %v", c)
	}
	ppeFail(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/campaigns/nope", "", http.StatusNotFound, "NotFoundException")
	if bl := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/blacklist-report?BlacklistItemNames=198.51.100.11&BlacklistItemNames=example.com", "")["BlacklistReport"].(map[string]any); len(bl) != 2 {
		t.Fatalf("blacklist report: %v", bl)
	}
	ppeFail(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/blacklist-report?BlacklistItemNames=203.0.113.9", "", http.StatusNotFound, "NotFoundException")
	ppeOK(t, ts, http.MethodPut, "/v1/email/deliverability-dashboard", `{"DashboardEnabled":false}`)
	if dash := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard", ""); dash["AccountStatus"] != "PENDING_EXPIRATION" || dash["SubscriptionExpiryDate"] == nil {
		t.Fatalf("disabled dashboard: %v", dash)
	}

	// Test reports keep the rendered message.
	rep := ppeOK(t, ts, http.MethodPost, "/v1/email/deliverability-dashboard/test", `{"ReportName":"launch","FromEmailAddress":"news@example.com","Content":`+simple+`,"Tags":[{"Key":"k","Value":"v"}]}`)
	if rep["DeliverabilityTestStatus"] != "IN_PROGRESS" {
		t.Fatalf("CreateDeliverabilityTestReport: %v", rep)
	}
	got := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/test-reports/"+rep["ReportId"].(string), "")
	if got["DeliverabilityTestReport"].(map[string]any)["DeliverabilityTestStatus"] != "COMPLETED" || !strings.Contains(got["Message"].(string), "Subject: Spring sale") ||
		len(got["IspPlacements"].([]any)) == 0 || len(got["Tags"].([]any)) != 1 {
		t.Fatalf("GetDeliverabilityTestReport: %v", got)
	}
	ppeFail(t, ts, http.MethodPost, "/v1/email/deliverability-dashboard/test", `{"FromEmailAddress":"x@unverified.com","Content":`+simple+`}`, http.StatusBadRequest, "MessageRejected")
	if reps := ppeOK(t, ts, http.MethodGet, "/v1/email/deliverability-dashboard/test-reports", "")["DeliverabilityTestReports"].([]any); len(reps) != 1 {
		t.Fatalf("ListDeliverabilityTestReports: %v", reps)
	}
}

func TestPinpointEmailStage1Limits(t *testing.T) {
	st := newPinpointEmailStore()
	for i := 0; i < ppeMaxReports+3; i++ {
		st.addReport(&ppeReport{id: fmt.Sprint(i)})
	}
	if len(st.reports) != ppeMaxReports || st.reports["0"] != nil {
		t.Fatalf("report eviction: %d", len(st.reports))
	}
	for i := 0; i < ppeMaxSent+10; i++ {
		st.recordSent("a@example.com", strings.Repeat("s", 300), 1)
	}
	if len(st.sent) != ppeMaxSent || len(st.sent[0].subject) != 256 {
		t.Fatalf("send log: %d", len(st.sent))
	}
	for i := 0; i < ppeMaxPools; i++ {
		if err := st.createPool(fmt.Sprint(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.createPool("one-more"); err == nil || err.code != "LimitExceededException" {
		t.Fatalf("pool limit: %v", err)
	}
}
