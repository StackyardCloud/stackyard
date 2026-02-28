package secretsmanager

import (
	"errors"
	"strings"
	"testing"
)

func TestSecretsManagerStage9CreateSecretIdempotencyConflict(t *testing.T) {
	svc := NewService()

	first, err := svc.CreateSecret(CreateSecretInput{
		Name:               "stage9-idempotency",
		ClientRequestToken: "stage9-create-token",
		SecretString:       `{"value":"v1"}`,
		Tags:               map[string]string{"env": "dev"},
	})
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}

	second, err := svc.CreateSecret(CreateSecretInput{
		Name:               "stage9-idempotency",
		ClientRequestToken: "stage9-create-token",
		SecretString:       `{"value":"v1"}`,
		Tags:               map[string]string{"env": "dev"},
	})
	if err != nil {
		t.Fatalf("idempotent create secret: %v", err)
	}
	if second.ARN != first.ARN {
		t.Fatalf("expected idempotent ARN %q, got %q", first.ARN, second.ARN)
	}

	if _, err := svc.CreateSecret(CreateSecretInput{
		Name:               "stage9-idempotency-different",
		ClientRequestToken: "stage9-create-token",
		SecretString:       `{"value":"v1"}`,
		Tags:               map[string]string{"env": "dev"},
	}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("expected ErrInvalidParameter for token/name conflict, got %v", err)
	}
}

func TestSecretsManagerStage7RotateContractAndTokenConflict(t *testing.T) {
	svc := NewService()
	createOut, err := svc.CreateSecret(CreateSecretInput{
		Name:         "stage7-rotate",
		SecretString: `{"password":"initial"}`,
	})
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}

	if _, err := svc.RotateSecret(RotateSecretInput{
		SecretID:          createOut.ARN,
		RotateImmediately: true,
	}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("expected ErrInvalidParameter when lambda is missing, got %v", err)
	}

	rotateOut, err := svc.RotateSecret(RotateSecretInput{
		SecretID:               createOut.ARN,
		ClientRequestToken:     "stage7-rotate-token",
		RotationLambdaARN:      "arn:aws:lambda:us-east-1:123456789012:function:stage7-rotate",
		AutomaticallyAfterDays: 30,
		RotateImmediately:      true,
	})
	if err != nil {
		t.Fatalf("rotate secret: %v", err)
	}
	if strings.TrimSpace(rotateOut.VersionID) == "" {
		t.Fatalf("expected rotate output to include version id")
	}

	rotateOut2, err := svc.RotateSecret(RotateSecretInput{
		SecretID:               createOut.ARN,
		ClientRequestToken:     "stage7-rotate-token",
		RotationLambdaARN:      "arn:aws:lambda:us-east-1:123456789012:function:stage7-rotate",
		AutomaticallyAfterDays: 30,
		RotateImmediately:      true,
	})
	if err != nil {
		t.Fatalf("idempotent rotate: %v", err)
	}
	if rotateOut2.VersionID != rotateOut.VersionID {
		t.Fatalf("expected idempotent rotate version %q, got %q", rotateOut.VersionID, rotateOut2.VersionID)
	}

	if _, err := svc.RotateSecret(RotateSecretInput{
		SecretID:               createOut.ARN,
		ClientRequestToken:     "stage7-rotate-token",
		RotationLambdaARN:      "arn:aws:lambda:us-east-1:123456789012:function:stage7-rotate",
		AutomaticallyAfterDays: 60,
		RotateImmediately:      true,
	}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("expected ErrInvalidParameter for rotate token conflict, got %v", err)
	}

	if _, err := svc.CancelRotateSecret(createOut.ARN); err != nil {
		t.Fatalf("cancel rotate: %v", err)
	}
	if _, err := svc.CancelRotateSecret(createOut.ARN); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState when canceling disabled rotation, got %v", err)
	}
}

func TestSecretsManagerStage6MetadataAndStage8FilterContracts(t *testing.T) {
	svc := NewService()
	createOut, err := svc.CreateSecret(CreateSecretInput{
		Name:          "stage68-metadata",
		OwningService: "stackyard-example",
		SecretString:  `{"value":"v1"}`,
	})
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}

	if _, err := svc.GetSecretValue(GetSecretValueInput{SecretID: createOut.ARN}); err != nil {
		t.Fatalf("get secret value: %v", err)
	}

	listOut, err := svc.ListSecrets(ListSecretsInput{
		Filters: []SecretFilter{
			{Key: "owning-service", Values: []string{"stackyard-example"}},
		},
	})
	if err != nil {
		t.Fatalf("list secrets with owning-service filter: %v", err)
	}
	if len(listOut.SecretList) != 1 {
		t.Fatalf("expected one filtered secret, got %d", len(listOut.SecretList))
	}
	if listOut.SecretList[0].OwningService != "stackyard-example" {
		t.Fatalf("expected owning service metadata to be preserved")
	}
	if listOut.SecretList[0].LastAccessedDate == nil {
		t.Fatalf("expected last accessed metadata to be tracked")
	}

	tooManyFilters := make([]SecretFilter, 0, defaultMaxFilters+1)
	for i := 0; i < defaultMaxFilters+1; i++ {
		tooManyFilters = append(tooManyFilters, SecretFilter{Key: "name", Values: []string{"stage68"}})
	}
	if _, err := svc.ListSecrets(ListSecretsInput{Filters: tooManyFilters}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("expected ErrLimitExceeded for too many filters, got %v", err)
	}
}

func TestSecretsManagerStage8PolicyAndTagValidation(t *testing.T) {
	svc := NewService()
	createOut, err := svc.CreateSecret(CreateSecretInput{
		Name:         "stage8-policy-tag",
		SecretString: `{"value":"v1"}`,
	})
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}

	if err := svc.TagResource(TagResourceInput{
		SecretID: createOut.ARN,
		Tags: map[string]string{
			"aws:reserved": "nope",
		},
	}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("expected ErrInvalidParameter for reserved tag key, got %v", err)
	}

	if _, err := svc.PutResourcePolicy(PutResourcePolicyInput{
		SecretID:       createOut.ARN,
		ResourcePolicy: `{"Version":"2012-10-17","Statement":[]}`,
	}); !errors.Is(err, ErrInvalidParameter) {
		t.Fatalf("expected ErrInvalidParameter for empty statement policy, got %v", err)
	}

	validateOut, err := svc.ValidateResourcePolicy(ValidateResourcePolicyInput{
		ResourcePolicy:    strings.Repeat("a", defaultMaxPolicyBytes+1),
		BlockPublicPolicy: true,
	})
	if err != nil {
		t.Fatalf("validate long policy: %v", err)
	}
	if validateOut.PolicyValidationPassed {
		t.Fatalf("expected long policy validation to fail")
	}
	if len(validateOut.ValidationErrors) == 0 || validateOut.ValidationErrors[0].CheckName != "POLICY_LENGTH_EXCEEDED" {
		t.Fatalf("expected POLICY_LENGTH_EXCEEDED validation error")
	}
}
