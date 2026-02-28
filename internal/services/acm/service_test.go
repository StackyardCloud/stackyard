package acm

import "testing"

func TestRequestImportLifecycle(t *testing.T) {
	svc := NewService()

	requestARN, err := svc.RequestCertificate(
		"example.com",
		[]string{"www.example.com"},
		"token-1",
		"EMAIL",
		CertificateOptions{CertificateTransparencyLoggingPreference: "ENABLED"},
		nil,
		map[string]string{"env": "test"},
	)
	if err != nil {
		t.Fatalf("request certificate: %v", err)
	}
	if requestARN == "" {
		t.Fatalf("expected request certificate arn")
	}

	requestARNDupe, err := svc.RequestCertificate(
		"example.com",
		nil,
		"token-1",
		"EMAIL",
		CertificateOptions{CertificateTransparencyLoggingPreference: "ENABLED"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("request certificate duplicate: %v", err)
	}
	if requestARNDupe != requestARN {
		t.Fatalf("expected idempotent request arn %q, got %q", requestARN, requestARNDupe)
	}

	if err := svc.ResendValidationEmail(requestARN, "example.com", "example.com"); err != nil {
		t.Fatalf("resend validation email: %v", err)
	}
	if err := svc.UpdateCertificateOptions(requestARN, CertificateOptions{CertificateTransparencyLoggingPreference: "DISABLED"}); err != nil {
		t.Fatalf("update certificate options: %v", err)
	}
	if _, _, err := svc.GetCertificate(requestARN); err != nil {
		t.Fatalf("get certificate: %v", err)
	}

	importARN, err := svc.ImportCertificate(
		"",
		"-----BEGIN CERTIFICATE-----\nimported\n-----END CERTIFICATE-----",
		"-----BEGIN PRIVATE KEY-----\nimported\n-----END PRIVATE KEY-----",
		"-----BEGIN CERTIFICATE-----\nchain\n-----END CERTIFICATE-----",
		nil,
	)
	if err != nil {
		t.Fatalf("import certificate: %v", err)
	}
	if importARN == "" {
		t.Fatalf("expected import certificate arn")
	}
	if _, _, _, err := svc.ExportCertificate(importARN, "stackyard-passphrase"); err != nil {
		t.Fatalf("export certificate: %v", err)
	}
	if err := svc.RenewCertificate(importARN); err != nil {
		t.Fatalf("renew certificate: %v", err)
	}
	if err := svc.RevokeCertificate(importARN, "KEY_COMPROMISE"); err != nil {
		t.Fatalf("revoke certificate: %v", err)
	}

	if err := svc.DeleteCertificate(importARN); err != nil {
		t.Fatalf("delete certificate: %v", err)
	}
	if _, err := svc.DescribeCertificate(importARN); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestListCertificatesPaginationAndValidation(t *testing.T) {
	svc := NewService()
	for i := 0; i < 3; i++ {
		if _, err := svc.ImportCertificate(
			"",
			"-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----",
			"-----BEGIN PRIVATE KEY-----\nx\n-----END PRIVATE KEY-----",
			"",
			nil,
		); err != nil {
			t.Fatalf("seed import certificate %d: %v", i, err)
		}
	}

	page1, next, err := svc.ListCertificates("", 2, nil)
	if err != nil {
		t.Fatalf("list certificates page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("expected 2 certs in page1, got %d", len(page1))
	}
	if next == "" {
		t.Fatalf("expected next token")
	}
	page2, next2, err := svc.ListCertificates(next, 2, nil)
	if err != nil {
		t.Fatalf("list certificates page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("expected 1 cert in page2, got %d", len(page2))
	}
	if next2 != "" {
		t.Fatalf("expected no next token on final page")
	}

	if _, _, err := svc.ListCertificates("bad-token", 2, nil); err != ErrInvalidParameter {
		t.Fatalf("expected invalid token error, got %v", err)
	}
	if _, _, err := svc.ListCertificates("", -1, nil); err != ErrInvalidParameter {
		t.Fatalf("expected invalid max items error, got %v", err)
	}
	if _, _, err := svc.ListCertificates("", 2, []string{"NOT_A_REAL_STATUS"}); err != ErrInvalidParameter {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestStage4TagsAndAccountConfig(t *testing.T) {
	svc := NewService()
	arn, err := svc.ImportCertificate(
		"",
		"-----BEGIN CERTIFICATE-----\ntag\n-----END CERTIFICATE-----",
		"-----BEGIN PRIVATE KEY-----\ntag\n-----END PRIVATE KEY-----",
		"",
		map[string]string{"env": "dev"},
	)
	if err != nil {
		t.Fatalf("import certificate: %v", err)
	}

	if err := svc.AddTagsToCertificate(arn, map[string]string{"team": "platform"}); err != nil {
		t.Fatalf("add tags: %v", err)
	}
	tags, err := svc.ListTagsForCertificate(arn)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected two tags, got %d", len(tags))
	}

	if err := svc.RemoveTagsFromCertificate(arn, []string{"team"}); err != nil {
		t.Fatalf("remove tags: %v", err)
	}
	tags, err = svc.ListTagsForCertificate(arn)
	if err != nil {
		t.Fatalf("list tags after remove: %v", err)
	}
	if _, ok := tags["team"]; ok {
		t.Fatalf("expected team tag removed")
	}

	config, err := svc.GetAccountConfiguration()
	if err != nil {
		t.Fatalf("get account config: %v", err)
	}
	if config.ExpiryEvents.DaysBeforeExpiry < 1 {
		t.Fatalf("expected valid default account config")
	}

	if err := svc.PutAccountConfiguration(AccountConfiguration{
		ExpiryEvents: ExpiryEventsConfiguration{DaysBeforeExpiry: 30},
	}); err != nil {
		t.Fatalf("put account config: %v", err)
	}
	config, err = svc.GetAccountConfiguration()
	if err != nil {
		t.Fatalf("get account config after put: %v", err)
	}
	if config.ExpiryEvents.DaysBeforeExpiry != 30 {
		t.Fatalf("expected account config days=30, got %d", config.ExpiryEvents.DaysBeforeExpiry)
	}
}

func TestStage5Invariants(t *testing.T) {
	svc := NewService()

	arn, err := svc.RequestCertificate(
		"example.com",
		nil,
		"token-stage5",
		"DNS",
		CertificateOptions{CertificateTransparencyLoggingPreference: "ENABLED"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("request certificate: %v", err)
	}
	if err := svc.ResendValidationEmail(arn, "example.com", "example.com"); err != ErrInvalidState {
		t.Fatalf("expected ErrInvalidState for DNS resend, got %v", err)
	}

	importARN, err := svc.ImportCertificate(
		"",
		"-----BEGIN CERTIFICATE-----\nexport\n-----END CERTIFICATE-----",
		"-----BEGIN PRIVATE KEY-----\nexport\n-----END PRIVATE KEY-----",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("import certificate: %v", err)
	}
	if err := svc.RevokeCertificate(importARN, "KEY_COMPROMISE"); err != nil {
		t.Fatalf("revoke certificate: %v", err)
	}
	if err := svc.RevokeCertificate(importARN, "KEY_COMPROMISE"); err != ErrInvalidState {
		t.Fatalf("expected ErrInvalidState on second revoke, got %v", err)
	}
	if _, _, _, err := svc.ExportCertificate(importARN, "passphrase"); err != ErrInvalidState {
		t.Fatalf("expected ErrInvalidState exporting revoked cert, got %v", err)
	}

	if err := svc.PutAccountConfiguration(AccountConfiguration{
		ExpiryEvents: ExpiryEventsConfiguration{DaysBeforeExpiry: 0},
	}); err != ErrInvalidParameter {
		t.Fatalf("expected ErrInvalidParameter for invalid account config, got %v", err)
	}

	overLimit := make(map[string]string, 60)
	for i := 0; i < 60; i++ {
		overLimit["k"+string(rune('a'+(i%26)))+string(rune('A'+(i/26)))] = "v"
	}
	if _, err := svc.RequestCertificate(
		"example.net",
		nil,
		"token-limit",
		"EMAIL",
		CertificateOptions{CertificateTransparencyLoggingPreference: "ENABLED"},
		nil,
		overLimit,
	); err != ErrLimitExceeded {
		t.Fatalf("expected ErrLimitExceeded, got %v", err)
	}
}
