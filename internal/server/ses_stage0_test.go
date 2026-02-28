package server

import (
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func sesRequest(t *testing.T, ts *httptest.Server, values url.Values) *http.Response {
	t.Helper()
	body := []byte(values.Encode())
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "ses")
}

func TestSESStage0Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	verifyEmail := url.Values{}
	verifyEmail.Set("Action", "VerifyEmailIdentity")
	verifyEmail.Set("EmailAddress", "sender@example.com")
	resp := sesRequest(t, ts, verifyEmail)
	assertStatus(t, resp, http.StatusOK)

	verifyDomain := url.Values{}
	verifyDomain.Set("Action", "VerifyDomainIdentity")
	verifyDomain.Set("Domain", "example.com")
	resp = sesRequest(t, ts, verifyDomain)
	assertStatus(t, resp, http.StatusOK)
	var verifyDomainResp struct {
		Result sesVerifyDomainIdentityResult `xml:"VerifyDomainIdentityResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &verifyDomainResp); err != nil {
		t.Fatalf("unmarshal verify domain: %v", err)
	}
	if verifyDomainResp.Result.VerificationToken == "" {
		t.Fatalf("expected verification token")
	}

	listIdentities := url.Values{}
	listIdentities.Set("Action", "ListIdentities")
	resp = sesRequest(t, ts, listIdentities)
	assertStatus(t, resp, http.StatusOK)
	var listIdentitiesResp struct {
		Result sesListIdentitiesResult `xml:"ListIdentitiesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listIdentitiesResp); err != nil {
		t.Fatalf("unmarshal list identities: %v", err)
	}
	if len(listIdentitiesResp.Result.Identities) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(listIdentitiesResp.Result.Identities))
	}

	getAttrs := url.Values{}
	getAttrs.Set("Action", "GetIdentityVerificationAttributes")
	getAttrs.Set("Identities.member.1", "sender@example.com")
	getAttrs.Set("Identities.member.2", "example.com")
	resp = sesRequest(t, ts, getAttrs)
	assertStatus(t, resp, http.StatusOK)
	var attrsResp struct {
		Result sesGetIdentityVerificationAttributesResult `xml:"GetIdentityVerificationAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &attrsResp); err != nil {
		t.Fatalf("unmarshal get identity attrs: %v", err)
	}
	if len(attrsResp.Result.VerificationAttributes) != 2 {
		t.Fatalf("expected 2 attribute entries, got %d", len(attrsResp.Result.VerificationAttributes))
	}

	createTemplate := url.Values{}
	createTemplate.Set("Action", "CreateTemplate")
	createTemplate.Set("Template.TemplateName", "welcome")
	createTemplate.Set("Template.SubjectPart", "Hello")
	createTemplate.Set("Template.TextPart", "Hi {{name}}")
	createTemplate.Set("Template.HtmlPart", "<h1>Hi {{name}}</h1>")
	resp = sesRequest(t, ts, createTemplate)
	assertStatus(t, resp, http.StatusOK)

	getTemplate := url.Values{}
	getTemplate.Set("Action", "GetTemplate")
	getTemplate.Set("TemplateName", "welcome")
	resp = sesRequest(t, ts, getTemplate)
	assertStatus(t, resp, http.StatusOK)
	var getTemplateResp struct {
		Result sesGetTemplateResult `xml:"GetTemplateResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &getTemplateResp); err != nil {
		t.Fatalf("unmarshal get template: %v", err)
	}
	if getTemplateResp.Result.Template.TemplateName != "welcome" {
		t.Fatalf("expected template welcome, got %q", getTemplateResp.Result.Template.TemplateName)
	}

	sendTemplated := url.Values{}
	sendTemplated.Set("Action", "SendTemplatedEmail")
	sendTemplated.Set("Source", "sender@example.com")
	sendTemplated.Set("Destination.ToAddresses.member.1", "recipient@example.com")
	sendTemplated.Set("Template", "welcome")
	sendTemplated.Set("TemplateData", `{"name":"Stackyard"}`)
	resp = sesRequest(t, ts, sendTemplated)
	assertStatus(t, resp, http.StatusOK)
	var templatedResp struct {
		Result sesSendTemplatedEmailResult `xml:"SendTemplatedEmailResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &templatedResp); err != nil {
		t.Fatalf("unmarshal send templated: %v", err)
	}
	if templatedResp.Result.MessageID == "" {
		t.Fatalf("expected message id from templated send")
	}

	sendEmail := url.Values{}
	sendEmail.Set("Action", "SendEmail")
	sendEmail.Set("Source", "sender@example.com")
	sendEmail.Set("Destination.ToAddresses.member.1", "recipient@example.com")
	sendEmail.Set("Message.Subject.Data", "Subject")
	sendEmail.Set("Message.Body.Text.Data", "Body")
	resp = sesRequest(t, ts, sendEmail)
	assertStatus(t, resp, http.StatusOK)

	rawMessage := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Raw\r\n\r\nhello"
	sendRaw := url.Values{}
	sendRaw.Set("Action", "SendRawEmail")
	sendRaw.Set("RawMessage.Data", base64.StdEncoding.EncodeToString([]byte(rawMessage)))
	resp = sesRequest(t, ts, sendRaw)
	assertStatus(t, resp, http.StatusOK)

	quota := url.Values{}
	quota.Set("Action", "GetSendQuota")
	resp = sesRequest(t, ts, quota)
	assertStatus(t, resp, http.StatusOK)
	var quotaResp struct {
		Result sesGetSendQuotaResult `xml:"GetSendQuotaResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &quotaResp); err != nil {
		t.Fatalf("unmarshal get send quota: %v", err)
	}
	if quotaResp.Result.SentLast24Hours < 3 {
		t.Fatalf("expected sent count >= 3, got %f", quotaResp.Result.SentLast24Hours)
	}

	stats := url.Values{}
	stats.Set("Action", "GetSendStatistics")
	resp = sesRequest(t, ts, stats)
	assertStatus(t, resp, http.StatusOK)
	var statsResp struct {
		Result sesGetSendStatisticsResult `xml:"GetSendStatisticsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &statsResp); err != nil {
		t.Fatalf("unmarshal get send statistics: %v", err)
	}
	if len(statsResp.Result.SendDataPoints) == 0 {
		t.Fatalf("expected at least one send datapoint")
	}

	deleteTemplate := url.Values{}
	deleteTemplate.Set("Action", "DeleteTemplate")
	deleteTemplate.Set("TemplateName", "welcome")
	resp = sesRequest(t, ts, deleteTemplate)
	assertStatus(t, resp, http.StatusOK)

	deleteIdentity := url.Values{}
	deleteIdentity.Set("Action", "DeleteIdentity")
	deleteIdentity.Set("Identity", "sender@example.com")
	resp = sesRequest(t, ts, deleteIdentity)
	assertStatus(t, resp, http.StatusOK)
}

func TestSESStage0TemplateNotFound(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	sendTemplated := url.Values{}
	sendTemplated.Set("Action", "SendTemplatedEmail")
	sendTemplated.Set("Source", "sender@example.com")
	sendTemplated.Set("Destination.ToAddresses.member.1", "recipient@example.com")
	sendTemplated.Set("Template", "missing-template")
	sendTemplated.Set("TemplateData", `{"name":"Stackyard"}`)
	resp := sesRequest(t, ts, sendTemplated)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "TemplateDoesNotExist") {
		t.Fatalf("expected TemplateDoesNotExist error, got: %s", body)
	}
}

func TestSESStage0CreateConfigurationSet(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req := url.Values{}
	req.Set("Action", "CreateConfigurationSet")
	req.Set("ConfigurationSet.Name", "marketing")
	resp := sesRequest(t, ts, req)
	assertStatus(t, resp, http.StatusOK)

	listReq := url.Values{}
	listReq.Set("Action", "ListConfigurationSets")
	resp = sesRequest(t, ts, listReq)
	assertStatus(t, resp, http.StatusOK)

	var out struct {
		Result sesListConfigurationSetsResult `xml:"ListConfigurationSetsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal list configuration sets: %v", err)
	}
	if len(out.Result.ConfigurationSets) != 1 || out.Result.ConfigurationSets[0].Name != "marketing" {
		t.Fatalf("expected configuration set marketing, got %+v", out.Result.ConfigurationSets)
	}
}

func TestSESStage0AccountSendingEnabled(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	verifyEmail := url.Values{}
	verifyEmail.Set("Action", "VerifyEmailIdentity")
	verifyEmail.Set("EmailAddress", "sender@example.com")
	resp := sesRequest(t, ts, verifyEmail)
	assertStatus(t, resp, http.StatusOK)

	disableSending := url.Values{}
	disableSending.Set("Action", "UpdateAccountSendingEnabled")
	disableSending.Set("Enabled", "false")
	resp = sesRequest(t, ts, disableSending)
	assertStatus(t, resp, http.StatusOK)

	getSendingEnabled := url.Values{}
	getSendingEnabled.Set("Action", "GetAccountSendingEnabled")
	resp = sesRequest(t, ts, getSendingEnabled)
	assertStatus(t, resp, http.StatusOK)
	var sendingResp struct {
		Result sesGetAccountSendingEnabledResult `xml:"GetAccountSendingEnabledResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &sendingResp); err != nil {
		t.Fatalf("unmarshal account sending enabled: %v", err)
	}
	if sendingResp.Result.Enabled {
		t.Fatalf("expected account sending to be disabled")
	}

	sendEmail := url.Values{}
	sendEmail.Set("Action", "SendEmail")
	sendEmail.Set("Source", "sender@example.com")
	sendEmail.Set("Destination.ToAddresses.member.1", "recipient@example.com")
	sendEmail.Set("Message.Subject.Data", "Subject")
	sendEmail.Set("Message.Body.Text.Data", "Body")
	resp = sesRequest(t, ts, sendEmail)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "MessageRejected") {
		t.Fatalf("expected MessageRejected when account sending disabled, got: %s", body)
	}
}

func TestSESStage0IdentityAttributeSettings(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	verifyDomain := url.Values{}
	verifyDomain.Set("Action", "VerifyDomainIdentity")
	verifyDomain.Set("Domain", "example.com")
	resp := sesRequest(t, ts, verifyDomain)
	assertStatus(t, resp, http.StatusOK)

	setDkim := url.Values{}
	setDkim.Set("Action", "SetIdentityDkimEnabled")
	setDkim.Set("Identity", "example.com")
	setDkim.Set("DkimEnabled", "true")
	resp = sesRequest(t, ts, setDkim)
	assertStatus(t, resp, http.StatusOK)

	setHeaders := url.Values{}
	setHeaders.Set("Action", "SetIdentityHeadersInNotificationsEnabled")
	setHeaders.Set("Identity", "example.com")
	setHeaders.Set("NotificationType", "Bounce")
	setHeaders.Set("Enabled", "true")
	resp = sesRequest(t, ts, setHeaders)
	assertStatus(t, resp, http.StatusOK)

	setForwarding := url.Values{}
	setForwarding.Set("Action", "SetIdentityFeedbackForwardingEnabled")
	setForwarding.Set("Identity", "example.com")
	setForwarding.Set("ForwardingEnabled", "false")
	resp = sesRequest(t, ts, setForwarding)
	assertStatus(t, resp, http.StatusOK)

	setTopic := url.Values{}
	setTopic.Set("Action", "SetIdentityNotificationTopic")
	setTopic.Set("Identity", "example.com")
	setTopic.Set("NotificationType", "Bounce")
	setTopic.Set("SnsTopic", "arn:aws:sns:us-east-1:123456789012:bounces")
	resp = sesRequest(t, ts, setTopic)
	assertStatus(t, resp, http.StatusOK)

	setMailFrom := url.Values{}
	setMailFrom.Set("Action", "SetIdentityMailFromDomain")
	setMailFrom.Set("Identity", "example.com")
	setMailFrom.Set("MailFromDomain", "mail.example.com")
	setMailFrom.Set("BehaviorOnMXFailure", "RejectMessage")
	resp = sesRequest(t, ts, setMailFrom)
	assertStatus(t, resp, http.StatusOK)

	getDkim := url.Values{}
	getDkim.Set("Action", "GetIdentityDkimAttributes")
	getDkim.Set("Identities.member.1", "example.com")
	resp = sesRequest(t, ts, getDkim)
	assertStatus(t, resp, http.StatusOK)
	var dkimResp struct {
		Result sesGetIdentityDkimAttributesResult `xml:"GetIdentityDkimAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &dkimResp); err != nil {
		t.Fatalf("unmarshal get dkim attrs: %v", err)
	}
	if len(dkimResp.Result.DkimAttributes) != 1 {
		t.Fatalf("expected dkim attributes for one identity, got %d", len(dkimResp.Result.DkimAttributes))
	}
	if !dkimResp.Result.DkimAttributes[0].Value.DkimEnabled {
		t.Fatalf("expected dkim enabled")
	}

	getNotification := url.Values{}
	getNotification.Set("Action", "GetIdentityNotificationAttributes")
	getNotification.Set("Identities.member.1", "example.com")
	resp = sesRequest(t, ts, getNotification)
	assertStatus(t, resp, http.StatusOK)
	var notifResp struct {
		Result sesGetIdentityNotificationAttributesResult `xml:"GetIdentityNotificationAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &notifResp); err != nil {
		t.Fatalf("unmarshal get notification attrs: %v", err)
	}
	if len(notifResp.Result.NotificationAttributes) != 1 {
		t.Fatalf("expected notification attributes for one identity, got %d", len(notifResp.Result.NotificationAttributes))
	}
	if notifResp.Result.NotificationAttributes[0].Value.BounceTopic == "" {
		t.Fatalf("expected bounce topic to be set")
	}
	if notifResp.Result.NotificationAttributes[0].Value.ForwardingEnabled {
		t.Fatalf("expected forwarding disabled")
	}
	if !notifResp.Result.NotificationAttributes[0].Value.HeadersInBounceNotificationsEnabled {
		t.Fatalf("expected bounce headers notifications enabled")
	}

	getMailFrom := url.Values{}
	getMailFrom.Set("Action", "GetIdentityMailFromDomainAttributes")
	getMailFrom.Set("Identities.member.1", "example.com")
	resp = sesRequest(t, ts, getMailFrom)
	assertStatus(t, resp, http.StatusOK)
	var mailFromResp struct {
		Result sesGetIdentityMailFromDomainAttributesResult `xml:"GetIdentityMailFromDomainAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &mailFromResp); err != nil {
		t.Fatalf("unmarshal get mail from attrs: %v", err)
	}
	if len(mailFromResp.Result.MailFromDomainAttributes) != 1 {
		t.Fatalf("expected mail from attributes for one identity, got %d", len(mailFromResp.Result.MailFromDomainAttributes))
	}
	value := mailFromResp.Result.MailFromDomainAttributes[0].Value
	if value.MailFromDomain != "mail.example.com" {
		t.Fatalf("expected mail from domain mail.example.com, got %q", value.MailFromDomain)
	}
	if value.BehaviorOnMXFailure != "RejectMessage" {
		t.Fatalf("expected behavior RejectMessage, got %q", value.BehaviorOnMXFailure)
	}
}
