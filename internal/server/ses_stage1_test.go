package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
)

func TestSESStage1AllActionsImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	covered := map[string]struct{}{}
	run := func(action string, configure func(values url.Values)) {
		t.Helper()
		values := url.Values{}
		values.Set("Action", action)
		if configure != nil {
			configure(values)
		}
		resp := sesRequest(t, ts, values)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "<Code>NotImplemented</Code>") {
			t.Fatalf("action %s returned NotImplemented: status=%d body=%s", action, resp.StatusCode, body)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("action %s expected 200, got %d: %s", action, resp.StatusCode, body)
		}
		covered[action] = struct{}{}
	}

	run("VerifyEmailIdentity", func(values url.Values) {
		values.Set("EmailAddress", "sender@example.com")
	})
	run("VerifyEmailAddress", func(values url.Values) {
		values.Set("EmailAddress", "other@example.com")
	})
	run("VerifyDomainIdentity", func(values url.Values) {
		values.Set("Domain", "example.com")
	})
	run("VerifyDomainDkim", func(values url.Values) {
		values.Set("Domain", "example.com")
	})
	run("ListIdentities", func(values url.Values) {
		values.Set("MaxItems", "100")
	})
	run("ListVerifiedEmailAddresses", nil)
	run("GetIdentityVerificationAttributes", func(values url.Values) {
		values.Set("Identities.member.1", "sender@example.com")
		values.Set("Identities.member.2", "example.com")
	})
	run("GetIdentityDkimAttributes", func(values url.Values) {
		values.Set("Identities.member.1", "example.com")
	})
	run("GetIdentityMailFromDomainAttributes", func(values url.Values) {
		values.Set("Identities.member.1", "example.com")
	})
	run("GetIdentityNotificationAttributes", func(values url.Values) {
		values.Set("Identities.member.1", "example.com")
	})
	run("SetIdentityDkimEnabled", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("DkimEnabled", "true")
	})
	run("SetIdentityFeedbackForwardingEnabled", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("ForwardingEnabled", "true")
	})
	run("SetIdentityHeadersInNotificationsEnabled", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("NotificationType", "Bounce")
		values.Set("Enabled", "true")
	})
	run("SetIdentityMailFromDomain", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("MailFromDomain", "mail.example.com")
		values.Set("BehaviorOnMXFailure", "RejectMessage")
	})
	run("SetIdentityNotificationTopic", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("NotificationType", "Bounce")
		values.Set("SnsTopic", "arn:aws:sns:us-east-1:123456789012:bounces")
	})
	run("GetAccountSendingEnabled", nil)
	run("UpdateAccountSendingEnabled", func(values url.Values) {
		values.Set("Enabled", "true")
	})
	run("CreateTemplate", func(values url.Values) {
		values.Set("Template.TemplateName", "stage1-template")
		values.Set("Template.SubjectPart", "Hello {{name}}")
		values.Set("Template.TextPart", "Hi {{name}}")
		values.Set("Template.HtmlPart", "<h1>Hi {{name}}</h1>")
	})
	run("GetTemplate", func(values url.Values) {
		values.Set("TemplateName", "stage1-template")
	})
	run("ListTemplates", func(values url.Values) {
		values.Set("MaxItems", "100")
	})
	run("UpdateTemplate", func(values url.Values) {
		values.Set("Template.TemplateName", "stage1-template")
		values.Set("Template.SubjectPart", "Updated {{name}}")
		values.Set("Template.TextPart", "Updated text {{name}}")
		values.Set("Template.HtmlPart", "<strong>Updated {{name}}</strong>")
	})
	run("SendEmail", func(values url.Values) {
		values.Set("Source", "sender@example.com")
		values.Set("Destination.ToAddresses.member.1", "recipient@example.com")
		values.Set("Message.Subject.Data", "Stage1")
		values.Set("Message.Body.Text.Data", "body")
	})
	run("SendRawEmail", func(values url.Values) {
		raw := "From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Raw\r\n\r\nhello"
		values.Set("Source", "sender@example.com")
		values.Set("Destinations.member.1", "recipient@example.com")
		values.Set("RawMessage.Data", base64.StdEncoding.EncodeToString([]byte(raw)))
	})
	run("SendTemplatedEmail", func(values url.Values) {
		values.Set("Source", "sender@example.com")
		values.Set("Destination.ToAddresses.member.1", "recipient@example.com")
		values.Set("Template", "stage1-template")
		values.Set("TemplateData", `{"name":"Stackyard"}`)
	})
	run("SendBulkTemplatedEmail", func(values url.Values) {
		values.Set("Source", "sender@example.com")
		values.Set("Template", "stage1-template")
		values.Set("DefaultTemplateData", `{"name":"Bulk"}`)
		values.Set("Destinations.member.1.Destination.ToAddresses.member.1", "recipient@example.com")
		values.Set("Destinations.member.1.ReplacementTemplateData", `{"name":"Recipient"}`)
	})
	run("GetSendQuota", nil)
	run("GetSendStatistics", nil)
	run("TestRenderTemplate", func(values url.Values) {
		values.Set("TemplateName", "stage1-template")
		values.Set("TemplateData", `{"name":"Render"}`)
	})

	run("CreateConfigurationSet", func(values url.Values) {
		values.Set("ConfigurationSet.Name", "stage1-config")
	})
	run("ListConfigurationSets", func(values url.Values) {
		values.Set("MaxItems", "100")
	})
	run("DescribeConfigurationSet", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
	})
	run("CreateConfigurationSetTrackingOptions", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("TrackingOptions.CustomRedirectDomain", "click.example.com")
	})
	run("UpdateConfigurationSetTrackingOptions", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("TrackingOptions.CustomRedirectDomain", "tracking.example.com")
	})
	run("PutConfigurationSetDeliveryOptions", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("TlsPolicy", "Require")
	})
	run("UpdateConfigurationSetReputationMetricsEnabled", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("Enabled", "true")
	})
	run("UpdateConfigurationSetSendingEnabled", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("Enabled", "true")
	})
	run("CreateConfigurationSetEventDestination", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("EventDestination.Name", "stage1-destination")
		values.Set("EventDestination.Enabled", "true")
		values.Set("EventDestination.MatchingEventTypes.member.1", "send")
		values.Set("EventDestination.SNSDestination.TopicARN", "arn:aws:sns:us-east-1:123456789012:ses-events")
	})
	run("UpdateConfigurationSetEventDestination", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("EventDestination.Name", "stage1-destination")
		values.Set("EventDestination.Enabled", "false")
		values.Set("EventDestination.MatchingEventTypes.member.1", "bounce")
		values.Set("EventDestination.SNSDestination.TopicARN", "arn:aws:sns:us-east-1:123456789012:ses-events-updated")
	})
	run("DeleteConfigurationSetEventDestination", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
		values.Set("EventDestinationName", "stage1-destination")
	})
	run("DeleteConfigurationSetTrackingOptions", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
	})

	run("CreateCustomVerificationEmailTemplate", func(values url.Values) {
		values.Set("TemplateName", "stage1-custom-template")
		values.Set("FromEmailAddress", "sender@example.com")
		values.Set("TemplateSubject", "Verify your email")
		values.Set("TemplateContent", "<h1>Please verify</h1>")
		values.Set("SuccessRedirectionURL", "https://example.com/success")
		values.Set("FailureRedirectionURL", "https://example.com/failure")
	})
	run("ListCustomVerificationEmailTemplates", func(values url.Values) {
		values.Set("MaxResults", "100")
	})
	run("GetCustomVerificationEmailTemplate", func(values url.Values) {
		values.Set("TemplateName", "stage1-custom-template")
	})
	run("UpdateCustomVerificationEmailTemplate", func(values url.Values) {
		values.Set("TemplateName", "stage1-custom-template")
		values.Set("FromEmailAddress", "sender@example.com")
		values.Set("TemplateSubject", "Updated verify email")
		values.Set("TemplateContent", "<p>Updated content</p>")
		values.Set("SuccessRedirectionURL", "https://example.com/ok")
		values.Set("FailureRedirectionURL", "https://example.com/no")
	})
	run("SendCustomVerificationEmail", func(values url.Values) {
		values.Set("TemplateName", "stage1-custom-template")
		values.Set("EmailAddress", "verify@example.com")
	})

	run("CreateReceiptFilter", func(values url.Values) {
		values.Set("Filter.Name", "stage1-filter")
		values.Set("Filter.IpFilter.Policy", "Allow")
		values.Set("Filter.IpFilter.Cidr", "127.0.0.1/32")
	})
	run("ListReceiptFilters", nil)
	run("CreateReceiptRuleSet", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
	})
	run("ListReceiptRuleSets", func(values url.Values) {
		values.Set("MaxItems", "100")
	})
	run("CreateReceiptRule", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
		values.Set("Rule.Name", "stage1-rule")
		values.Set("Rule.Enabled", "true")
		values.Set("Rule.TlsPolicy", "Optional")
		values.Set("Rule.Recipients.member.1", "recipient@example.com")
		values.Set("Rule.ScanEnabled", "false")
	})
	run("DescribeReceiptRuleSet", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
	})
	run("DescribeReceiptRule", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
		values.Set("RuleName", "stage1-rule")
	})
	run("UpdateReceiptRule", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
		values.Set("Rule.Name", "stage1-rule")
		values.Set("Rule.Enabled", "false")
		values.Set("Rule.TlsPolicy", "Require")
		values.Set("Rule.Recipients.member.1", "recipient@example.com")
		values.Set("Rule.ScanEnabled", "true")
	})
	run("ReorderReceiptRuleSet", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
		values.Set("RuleNames.member.1", "stage1-rule")
	})
	run("SetReceiptRulePosition", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
		values.Set("RuleName", "stage1-rule")
	})
	run("SetActiveReceiptRuleSet", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
	})
	run("DescribeActiveReceiptRuleSet", nil)
	run("CloneReceiptRuleSet", func(values url.Values) {
		values.Set("OriginalRuleSetName", "stage1-ruleset")
		values.Set("RuleSetName", "stage1-ruleset-clone")
	})
	run("SendBounce", func(values url.Values) {
		values.Set("OriginalMessageId", "orig-message-id")
		values.Set("BounceSender", "sender@example.com")
	})

	run("PutIdentityPolicy", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("PolicyName", "stage1-policy")
		values.Set("Policy", `{"Version":"2012-10-17","Statement":[]}`)
	})
	run("ListIdentityPolicies", func(values url.Values) {
		values.Set("Identity", "example.com")
	})
	run("GetIdentityPolicies", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("PolicyNames.member.1", "stage1-policy")
	})
	run("DeleteIdentityPolicy", func(values url.Values) {
		values.Set("Identity", "example.com")
		values.Set("PolicyName", "stage1-policy")
	})

	run("DeleteReceiptRule", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset")
		values.Set("RuleName", "stage1-rule")
	})
	run("DeleteReceiptRuleSet", func(values url.Values) {
		values.Set("RuleSetName", "stage1-ruleset-clone")
	})
	run("DeleteReceiptFilter", func(values url.Values) {
		values.Set("FilterName", "stage1-filter")
	})
	run("DeleteConfigurationSet", func(values url.Values) {
		values.Set("ConfigurationSetName", "stage1-config")
	})
	run("DeleteCustomVerificationEmailTemplate", func(values url.Values) {
		values.Set("TemplateName", "stage1-custom-template")
	})
	run("DeleteTemplate", func(values url.Values) {
		values.Set("TemplateName", "stage1-template")
	})
	run("DeleteIdentity", func(values url.Values) {
		values.Set("Identity", "example.com")
	})
	run("DeleteVerifiedEmailAddress", func(values url.Values) {
		values.Set("EmailAddress", "other@example.com")
	})

	if len(covered) != len(sesActions) {
		missing := make([]string, 0)
		for action := range sesActions {
			if _, ok := covered[action]; !ok {
				missing = append(missing, action)
			}
		}
		sort.Strings(missing)
		t.Fatalf("coverage mismatch: covered=%d total=%d missing=%v", len(covered), len(sesActions), missing)
	}
	if len(covered) > len(sesActions) {
		extra := make([]string, 0)
		for action := range covered {
			if _, ok := sesActions[action]; !ok {
				extra = append(extra, action)
			}
		}
		sort.Strings(extra)
		t.Fatalf("covered actions outside sesActions map: %v", extra)
	}

	for action := range sesActions {
		if _, ok := covered[action]; !ok {
			t.Fatalf("missing action coverage for %s", action)
		}
	}

	if got := len(covered); got != 71 {
		t.Fatalf("expected 71 covered actions, got %d", got)
	}
	if got := len(sesActions); got != 71 {
		t.Fatalf("expected 71 declared SES actions, got %d", got)
	}

	t.Logf("validated %s SES actions", fmt.Sprintf("%d", len(covered)))
}
