package ses

import "testing"

func TestServiceTemplateAndSend(t *testing.T) {
	svc := NewService()

	if err := svc.CreateTemplate(Template{
		Name:     "welcome",
		Subject:  "hello",
		TextPart: "hi",
	}); err != nil {
		t.Fatalf("create template: %v", err)
	}

	if _, err := svc.SendTemplatedEmail(SendTemplatedEmailInput{
		Source:       "sender@example.com",
		Destinations: []string{"recipient@example.com"},
		TemplateName: "welcome",
		TemplateData: `{"name":"Stackyard"}`,
	}); err != nil {
		t.Fatalf("send templated email: %v", err)
	}

	quota := svc.GetSendQuota()
	if quota.SentLast24Hours < 1 {
		t.Fatalf("expected sent count >= 1, got %f", quota.SentLast24Hours)
	}
}

func TestServiceIdentityLifecycle(t *testing.T) {
	svc := NewService()

	if err := svc.VerifyEmailIdentity("sender@example.com"); err != nil {
		t.Fatalf("verify email identity: %v", err)
	}
	if _, err := svc.VerifyDomainIdentity("example.com"); err != nil {
		t.Fatalf("verify domain identity: %v", err)
	}

	idents, next, err := svc.ListIdentities("", "", 10)
	if err != nil {
		t.Fatalf("list identities: %v", err)
	}
	if len(idents) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(idents))
	}
	if next != "" {
		t.Fatalf("expected empty next token, got %q", next)
	}

	attrs := svc.GetIdentityVerificationAttributes([]string{"sender@example.com", "example.com"})
	if len(attrs) != 2 {
		t.Fatalf("expected 2 verification attributes, got %d", len(attrs))
	}

	svc.DeleteIdentity("sender@example.com")
	idents, _, err = svc.ListIdentities("", "", 10)
	if err != nil {
		t.Fatalf("list identities after delete: %v", err)
	}
	if len(idents) != 1 {
		t.Fatalf("expected 1 identity after delete, got %d", len(idents))
	}
}

func TestServiceIdentitySettingsAndAccountSending(t *testing.T) {
	svc := NewService()
	if _, err := svc.VerifyDomainIdentity("example.com"); err != nil {
		t.Fatalf("verify domain identity: %v", err)
	}

	if err := svc.SetIdentityDkimEnabled("example.com", true); err != nil {
		t.Fatalf("set dkim enabled: %v", err)
	}
	if err := svc.SetIdentityFeedbackForwardingEnabled("example.com", false); err != nil {
		t.Fatalf("set feedback forwarding: %v", err)
	}
	if err := svc.SetIdentityHeadersInNotificationsEnabled("example.com", "Bounce", true); err != nil {
		t.Fatalf("set headers in notifications: %v", err)
	}
	if err := svc.SetIdentityNotificationTopic("example.com", "Bounce", "arn:aws:sns:us-east-1:123456789012:bounces"); err != nil {
		t.Fatalf("set notification topic: %v", err)
	}
	if err := svc.SetIdentityMailFromDomain("example.com", "mail.example.com", BehaviorOnMXFailureRejectMessage); err != nil {
		t.Fatalf("set mail from domain: %v", err)
	}

	dkim := svc.GetIdentityDkimAttributes([]string{"example.com"})
	if len(dkim) != 1 {
		t.Fatalf("expected 1 dkim attribute entry, got %d", len(dkim))
	}
	if !dkim["example.com"].DkimEnabled {
		t.Fatalf("expected dkim enabled")
	}

	notif := svc.GetIdentityNotificationAttributes([]string{"example.com"})
	if len(notif) != 1 {
		t.Fatalf("expected 1 notification attribute entry, got %d", len(notif))
	}
	if notif["example.com"].ForwardingEnabled {
		t.Fatalf("expected forwarding disabled")
	}
	if notif["example.com"].BounceTopic == "" {
		t.Fatalf("expected bounce topic")
	}
	if !notif["example.com"].HeadersInBounceNotifications {
		t.Fatalf("expected bounce headers notifications enabled")
	}

	mailFrom := svc.GetIdentityMailFromDomainAttributes([]string{"example.com"})
	if len(mailFrom) != 1 {
		t.Fatalf("expected 1 mail-from attribute entry, got %d", len(mailFrom))
	}
	if mailFrom["example.com"].MailFromDomain != "mail.example.com" {
		t.Fatalf("expected mail from domain mail.example.com, got %q", mailFrom["example.com"].MailFromDomain)
	}
	if mailFrom["example.com"].BehaviorOnMXFailure != BehaviorOnMXFailureRejectMessage {
		t.Fatalf("expected behavior %q, got %q", BehaviorOnMXFailureRejectMessage, mailFrom["example.com"].BehaviorOnMXFailure)
	}

	if !svc.GetAccountSendingEnabled() {
		t.Fatalf("expected account sending initially enabled")
	}
	svc.UpdateAccountSendingEnabled(false)
	if svc.GetAccountSendingEnabled() {
		t.Fatalf("expected account sending disabled")
	}
	if _, err := svc.SendEmail(SendEmailInput{
		Source:       "sender@example.com",
		Destinations: []string{"recipient@example.com"},
		Subject:      "subject",
		TextBody:     "body",
	}); err != ErrMessageRejected {
		t.Fatalf("expected ErrMessageRejected with sending disabled, got %v", err)
	}
}
