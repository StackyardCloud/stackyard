package server

import "testing"

func TestPaymentCryptographyStage0CatalogCoverage(t *testing.T) {
	if len(paymentCryptographyOperations) != 20 {
		t.Fatalf("expected 20 Payment Cryptography operations from docs, got %d", len(paymentCryptographyOperations))
	}
	if len(paymentCryptographyOperationByName) != len(paymentCryptographyOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateKey",
		"GetKey",
		"ListKeys",
		"CreateAlias",
		"TagResource",
		"GetParametersForImport",
	}
	for _, action := range requiredActions {
		if _, ok := paymentCryptographyOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(paymentCryptographyDataTypes) != 60 {
		t.Fatalf("expected 60 Payment Cryptography data types from docs, got %d", len(paymentCryptographyDataTypes))
	}
	if len(paymentCryptographyDataTypeByName) != len(paymentCryptographyDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Key",
		"KeyAttributes",
		"Alias",
		"KeyModesOfUse",
		"Tag",
		"WrappedKey",
	}
	for _, typeName := range requiredTypes {
		if _, ok := paymentCryptographyDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}
