package sesv2

import "testing"

func TestServiceIdentityTemplateAndSendLifecycle(t *testing.T) {
	svc := NewService()

	identity, err := svc.CreateEmailIdentity("sender@example.com", "demo-config", map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("create identity: %v", err)
	}
	if identity.IdentityType != "EMAIL_ADDRESS" {
		t.Fatalf("expected EMAIL_ADDRESS identity type, got %q", identity.IdentityType)
	}

	if err := svc.CreateEmailIdentityPolicy("sender@example.com", "default", "{\"Version\":\"2012-10-17\"}"); err != nil {
		t.Fatalf("put identity policy: %v", err)
	}
	policies, ok := svc.GetEmailIdentityPolicies("sender@example.com")
	if !ok || policies["default"] == "" {
		t.Fatalf("expected identity policy")
	}

	if err := svc.CreateEmailTemplate("welcome", "Hi {{name}}", "Hello {{name}}", ""); err != nil {
		t.Fatalf("create template: %v", err)
	}
	rendered, err := svc.RenderEmailTemplate("welcome", `{"name":"Stackyard"}`)
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	if rendered != "Hello Stackyard" {
		t.Fatalf("unexpected rendered template: %q", rendered)
	}

	msgID, err := svc.SendEmail("sender@example.com", []string{"recipient@example.com"})
	if err != nil {
		t.Fatalf("send email: %v", err)
	}
	if msgID == "" {
		t.Fatalf("expected message id")
	}

	bulk, err := svc.SendBulkEmail(2)
	if err != nil {
		t.Fatalf("send bulk email: %v", err)
	}
	if len(bulk) != 2 {
		t.Fatalf("expected two bulk results, got %d", len(bulk))
	}

	if !svc.DeleteEmailTemplate("welcome") {
		t.Fatalf("expected template deletion")
	}
	if !svc.DeleteEmailIdentity("sender@example.com") {
		t.Fatalf("expected identity deletion")
	}
}

func TestServiceConfigSuppressionTagsContactsAndAccount(t *testing.T) {
	svc := NewService()

	if err := svc.CreateConfigurationSet("demo-config"); err != nil {
		t.Fatalf("create configuration set: %v", err)
	}
	if err := svc.UpsertConfigurationSetEventDestination("demo-config", "events", map[string]any{"Enabled": true}); err != nil {
		t.Fatalf("upsert event destination: %v", err)
	}
	dests, ok := svc.GetConfigurationSetEventDestinations("demo-config")
	if !ok || len(dests) != 1 {
		t.Fatalf("expected one configuration set destination")
	}

	if _, err := svc.PutSuppressedDestination("recipient@example.com", "BOUNCE"); err != nil {
		t.Fatalf("put suppressed destination: %v", err)
	}
	if _, ok := svc.GetSuppressedDestination("recipient@example.com"); !ok {
		t.Fatalf("expected suppressed destination")
	}
	if !svc.DeleteSuppressedDestination("recipient@example.com") {
		t.Fatalf("expected suppressed destination deletion")
	}

	if err := svc.TagResource("arn:aws:ses:us-east-1:123456789012:configuration-set/demo-config", map[string]string{"env": "test"}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	tags := svc.ListTags("arn:aws:ses:us-east-1:123456789012:configuration-set/demo-config")
	if tags["env"] != "test" {
		t.Fatalf("expected env tag")
	}
	svc.UntagResource("arn:aws:ses:us-east-1:123456789012:configuration-set/demo-config", []string{"env"})
	if len(svc.ListTags("arn:aws:ses:us-east-1:123456789012:configuration-set/demo-config")) != 0 {
		t.Fatalf("expected tags to be removed")
	}

	if err := svc.CreateContactList("audience", "test"); err != nil {
		t.Fatalf("create contact list: %v", err)
	}
	if err := svc.CreateContact("audience", "recipient@example.com", "{}", false, nil); err != nil {
		t.Fatalf("create contact: %v", err)
	}
	if err := svc.UpdateContact("audience", "recipient@example.com", `{"tier":"gold"}`, false, nil); err != nil {
		t.Fatalf("update contact: %v", err)
	}
	contacts, ok := svc.ListContacts("audience")
	if !ok || len(contacts) != 1 {
		t.Fatalf("expected one contact")
	}

	svc.PutAccountSendingAttributes(false)
	if _, err := svc.SendEmail("sender@example.com", []string{"recipient@example.com"}); err != ErrSendingDisabled {
		t.Fatalf("expected sending disabled error, got %v", err)
	}
	acc := svc.GetAccount()
	if acc.SendingEnabled {
		t.Fatalf("expected sending to be disabled")
	}
}
