package server

import "testing"

func TestSecurityIRStage0CatalogCoverage(t *testing.T) {
	if len(securityIROperations) != 24 {
		t.Fatalf("expected 24 Security Incident Response operations from docs, got %d", len(securityIROperations))
	}
	if len(securityIROperationByName) != len(securityIROperations) {
		t.Fatalf("expected unique Security Incident Response operation names")
	}

	requiredActions := []string{
		"CreateCase",
		"GetCase",
		"ListCases",
		"ListInvestigations",
		"SendFeedback",
		"CreateMembership",
		"GetMembership",
		"ListMemberships",
		"UpdateMembership",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := securityIROperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}
	if op := securityIROperationByName["ListInvestigations"]; op.Method != "GET" || op.URI != "/v1/cases/{caseId}/list-investigations" {
		t.Fatalf("expected ListInvestigations route metadata to be populated")
	}
	if op := securityIROperationByName["SendFeedback"]; op.Method != "POST" || op.URI != "/v1/cases/{caseId}/feedback/{resultId}/send-feedback" {
		t.Fatalf("expected SendFeedback route metadata to be populated")
	}

	if len(securityIRTypes) != 19 {
		t.Fatalf("expected 19 Security Incident Response data types from docs, got %d", len(securityIRTypes))
	}
	if len(securityIRTypeByName) != len(securityIRTypes) {
		t.Fatalf("expected unique Security Incident Response data type names")
	}

	requiredTypes := []string{
		"CaseMetadataEntry",
		"InvestigationAction",
		"InvestigationFeedback",
		"MembershipAccountsConfigurations",
		"Watcher",
	}
	for _, typeName := range requiredTypes {
		if _, ok := securityIRTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
