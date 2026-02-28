package privateca

import (
	"errors"
	"strings"
	"testing"
)

func TestServiceLifecycleStage1(t *testing.T) {
	svc := NewService()

	arn, err := svc.CreateCertificateAuthority(CreateCertificateAuthorityInput{
		Configuration: CertificateAuthorityConfiguration{
			KeyAlgorithm:     "RSA_2048",
			SigningAlgorithm: "SHA256WITHRSA",
			Subject:          Subject{CommonName: "stackyard-privateca"},
		},
		CertificateAuthorityType: "ROOT",
		IdempotencyToken:         "token-1",
	})
	if err != nil {
		t.Fatalf("create certificate authority: %v", err)
	}
	if arn == "" {
		t.Fatalf("expected ARN")
	}

	ca, err := svc.DescribeCertificateAuthority(arn)
	if err != nil {
		t.Fatalf("describe certificate authority: %v", err)
	}
	if ca.ARN != arn {
		t.Fatalf("expected describe ARN %q, got %q", arn, ca.ARN)
	}

	listOut, err := svc.ListCertificateAuthorities(ListCertificateAuthoritiesInput{MaxResults: 10})
	if err != nil {
		t.Fatalf("list certificate authorities: %v", err)
	}
	if len(listOut.CertificateAuthorities) != 1 {
		t.Fatalf("expected 1 certificate authority, got %d", len(listOut.CertificateAuthorities))
	}

	if err := svc.UpdateCertificateAuthority(UpdateCertificateAuthorityInput{ARN: arn, Status: "DISABLED"}); err != nil {
		t.Fatalf("update certificate authority: %v", err)
	}

	if err := svc.DeleteCertificateAuthority(arn, 7); err != nil {
		t.Fatalf("delete certificate authority: %v", err)
	}

	if err := svc.RestoreCertificateAuthority(arn); err != nil {
		t.Fatalf("restore certificate authority: %v", err)
	}

	ca, err = svc.DescribeCertificateAuthority(arn)
	if err != nil {
		t.Fatalf("describe after restore: %v", err)
	}
	if ca.Status != "DISABLED" {
		t.Fatalf("expected DISABLED status after restore, got %q", ca.Status)
	}
}

func TestServiceStage2IssuanceMaterialAndRevocation(t *testing.T) {
	svc := NewService()

	caARN, err := svc.CreateCertificateAuthority(CreateCertificateAuthorityInput{
		Configuration: CertificateAuthorityConfiguration{
			KeyAlgorithm:     "RSA_2048",
			SigningAlgorithm: "SHA256WITHRSA",
			Subject:          Subject{CommonName: "stage2-privateca"},
		},
		CertificateAuthorityType: "ROOT",
	})
	if err != nil {
		t.Fatalf("create certificate authority: %v", err)
	}

	csr, err := svc.GetCertificateAuthorityCSR(caARN)
	if err != nil {
		t.Fatalf("get csr: %v", err)
	}
	if strings.TrimSpace(csr) == "" {
		t.Fatalf("expected csr output")
	}

	if err := svc.ImportCertificateAuthorityCertificate(caARN, "imported-cert", "imported-chain"); err != nil {
		t.Fatalf("import certificate authority certificate: %v", err)
	}

	caCert, caChain, err := svc.GetCertificateAuthorityCertificate(caARN)
	if err != nil {
		t.Fatalf("get certificate authority certificate: %v", err)
	}
	if caCert != "imported-cert" {
		t.Fatalf("expected imported cert, got %q", caCert)
	}
	if caChain != "imported-chain" {
		t.Fatalf("expected imported chain, got %q", caChain)
	}

	certARN, err := svc.IssueCertificate(IssueCertificateInput{
		CertificateAuthorityARN: caARN,
		Csr:                     "c3RhY2t5YXJk",
		SigningAlgorithm:        "SHA256WITHRSA",
		TemplateARN:             "arn:aws:acm-pca:::template/EndEntityCertificate/V1",
		Validity:                Validity{Type: "DAYS", Value: 365},
		IdempotencyToken:        "stage2-issue",
	})
	if err != nil {
		t.Fatalf("issue certificate: %v", err)
	}
	if strings.TrimSpace(certARN) == "" {
		t.Fatalf("expected certificate arn")
	}

	certBody, certChain, err := svc.GetCertificate(caARN, certARN)
	if err != nil {
		t.Fatalf("get certificate: %v", err)
	}
	if !strings.Contains(certBody, "BEGIN CERTIFICATE") {
		t.Fatalf("expected certificate body")
	}
	if strings.TrimSpace(certChain) == "" {
		t.Fatalf("expected certificate chain")
	}

	if err := svc.RevokeCertificate(RevokeCertificateInput{
		CertificateAuthorityARN: caARN,
		CertificateSerial:       "01",
		RevocationReason:        "KEY_COMPROMISE",
	}); err != nil {
		t.Fatalf("revoke certificate: %v", err)
	}
}

func TestServiceStage3PermissionsAndPolicy(t *testing.T) {
	svc := NewService()

	caARN, err := svc.CreateCertificateAuthority(CreateCertificateAuthorityInput{
		Configuration: CertificateAuthorityConfiguration{
			KeyAlgorithm:     "RSA_2048",
			SigningAlgorithm: "SHA256WITHRSA",
			Subject:          Subject{CommonName: "stage3-privateca"},
		},
		CertificateAuthorityType: "ROOT",
	})
	if err != nil {
		t.Fatalf("create certificate authority: %v", err)
	}

	if err := svc.CreatePermission(CreatePermissionInput{
		CertificateAuthorityARN: caARN,
		Principal:               "123456789012",
		SourceAccount:           "123456789012",
		Actions:                 []string{"IssueCertificate", "GetCertificate", "ListPermissions"},
	}); err != nil {
		t.Fatalf("create permission: %v", err)
	}

	permissions, err := svc.ListPermissions(ListPermissionsInput{CertificateAuthorityARN: caARN, MaxResults: 10})
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	if len(permissions.Permissions) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(permissions.Permissions))
	}

	if err := svc.DeletePermission(caARN, "123456789012", "123456789012"); err != nil {
		t.Fatalf("delete permission: %v", err)
	}

	policy := `{"Version":"2012-10-17","Statement":[]}`
	if err := svc.PutPolicy(caARN, policy); err != nil {
		t.Fatalf("put policy: %v", err)
	}

	storedPolicy, err := svc.GetPolicy(caARN)
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if storedPolicy != policy {
		t.Fatalf("expected policy %q, got %q", policy, storedPolicy)
	}

	if err := svc.DeletePolicy(caARN); err != nil {
		t.Fatalf("delete policy: %v", err)
	}

	if _, err := svc.GetPolicy(caARN); err == nil {
		t.Fatalf("expected get policy to fail after delete")
	}
}

func TestServiceStage4AuditReportsAndTags(t *testing.T) {
	svc := NewService()

	caARN, err := svc.CreateCertificateAuthority(CreateCertificateAuthorityInput{
		Configuration: CertificateAuthorityConfiguration{
			KeyAlgorithm:     "RSA_2048",
			SigningAlgorithm: "SHA256WITHRSA",
			Subject:          Subject{CommonName: "stage4-privateca"},
		},
		CertificateAuthorityType: "ROOT",
	})
	if err != nil {
		t.Fatalf("create certificate authority: %v", err)
	}

	reportID, s3Key, err := svc.CreateCertificateAuthorityAuditReport(caARN, "stackyard-privateca-reports", "JSON")
	if err != nil {
		t.Fatalf("create audit report: %v", err)
	}
	if strings.TrimSpace(reportID) == "" || strings.TrimSpace(s3Key) == "" {
		t.Fatalf("expected non-empty audit report id and s3 key")
	}

	report, err := svc.DescribeCertificateAuthorityAuditReport(caARN, reportID)
	if err != nil {
		t.Fatalf("describe audit report: %v", err)
	}
	if report.AuditReportID != reportID {
		t.Fatalf("expected report id %q, got %q", reportID, report.AuditReportID)
	}

	if err := svc.TagCertificateAuthority(caARN, map[string]string{"env": "dev", "team": "platform"}); err != nil {
		t.Fatalf("tag certificate authority: %v", err)
	}
	tagPage1, err := svc.ListTags(ListTagsInput{CertificateAuthorityARN: caARN, MaxResults: 1})
	if err != nil {
		t.Fatalf("list tags page1: %v", err)
	}
	if len(tagPage1.Tags) != 1 {
		t.Fatalf("expected 1 tag on first page, got %d", len(tagPage1.Tags))
	}
	if strings.TrimSpace(tagPage1.NextToken) == "" {
		t.Fatalf("expected next token for first tag page")
	}

	tagPage2, err := svc.ListTags(ListTagsInput{
		CertificateAuthorityARN: caARN,
		MaxResults:              10,
		NextToken:               tagPage1.NextToken,
	})
	if err != nil {
		t.Fatalf("list tags page2: %v", err)
	}
	if len(tagPage2.Tags) != 1 {
		t.Fatalf("expected 1 remaining tag, got %d", len(tagPage2.Tags))
	}

	if err := svc.UntagCertificateAuthority(caARN, []string{"team"}); err != nil {
		t.Fatalf("untag certificate authority: %v", err)
	}
	allTags, err := svc.ListTags(ListTagsInput{CertificateAuthorityARN: caARN, MaxResults: 10})
	if err != nil {
		t.Fatalf("list tags after untag: %v", err)
	}
	if _, ok := allTags.Tags["team"]; ok {
		t.Fatalf("expected team tag to be removed")
	}
	if _, ok := allTags.Tags["env"]; !ok {
		t.Fatalf("expected env tag to remain")
	}
}

func TestServiceStage5ContractsAndInvariants(t *testing.T) {
	svc := NewService()

	createInput := CreateCertificateAuthorityInput{
		Configuration: CertificateAuthorityConfiguration{
			KeyAlgorithm:     "RSA_2048",
			SigningAlgorithm: "SHA256WITHRSA",
			Subject:          Subject{CommonName: "stage5-privateca"},
		},
		CertificateAuthorityType: "ROOT",
		IdempotencyToken:         "stage5-create-token",
	}
	caARN1, err := svc.CreateCertificateAuthority(createInput)
	if err != nil {
		t.Fatalf("create certificate authority #1: %v", err)
	}
	caARN2, err := svc.CreateCertificateAuthority(createInput)
	if err != nil {
		t.Fatalf("create certificate authority #2: %v", err)
	}
	if caARN1 != caARN2 {
		t.Fatalf("expected create idempotency to return same arn, got %q and %q", caARN1, caARN2)
	}

	issueInput := IssueCertificateInput{
		CertificateAuthorityARN: caARN1,
		Csr:                     "c3RhY2t5YXJk",
		SigningAlgorithm:        "SHA256WITHRSA",
		TemplateARN:             "arn:aws:acm-pca:::template/EndEntityCertificate/V1",
		Validity:                Validity{Type: "DAYS", Value: 365},
		IdempotencyToken:        "stage5-issue-token",
	}
	certARN1, err := svc.IssueCertificate(issueInput)
	if err != nil {
		t.Fatalf("issue certificate #1: %v", err)
	}
	certARN2, err := svc.IssueCertificate(issueInput)
	if err != nil {
		t.Fatalf("issue certificate #2: %v", err)
	}
	if certARN1 != certARN2 {
		t.Fatalf("expected issue idempotency to return same certificate arn, got %q and %q", certARN1, certARN2)
	}

	if _, err := svc.ListPermissions(ListPermissionsInput{CertificateAuthorityARN: caARN1, NextToken: "bad-token"}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("expected invalid parameter for bad next token, got %v", err)
	}

	if err := svc.DeleteCertificateAuthority(caARN1, 7); err != nil {
		t.Fatalf("delete certificate authority: %v", err)
	}
	if err := svc.UpdateCertificateAuthority(UpdateCertificateAuthorityInput{ARN: caARN1, Status: "ACTIVE"}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid state for update after delete, got %v", err)
	}
	if err := svc.PutPolicy(caARN1, `{"Version":"2012-10-17","Statement":[]}`); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected invalid state for put policy after delete, got %v", err)
	}
}
