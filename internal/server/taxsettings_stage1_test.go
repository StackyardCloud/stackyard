package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func taxsOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := acctCallREST(t, ts, "tax", "/"+op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func taxsErr(t *testing.T, ts *httptest.Server, op, body string, wantStatus int) string {
	t.Helper()
	status, out := acctCallREST(t, ts, "tax", "/"+op, body)
	if status != wantStatus {
		t.Fatalf("%s: want %d, got %d %v", op, wantStatus, status, out)
	}
	return out["__type"].(string)
}

const taxsEntryJSON = `{"registrationId":"DE123456789","registrationType":"VAT","legalName":"Stackyard GmbH","legalAddress":{"addressLine1":"Hauptstr. 1","city":"Berlin","postalCode":"10115","countryCode":"DE"}%s}`

func taxsDocEntry(content string) string {
	file := base64.StdEncoding.EncodeToString([]byte(content))
	return strings.Replace(taxsEntryJSON, "%s", `,"verificationDetails":{"taxRegistrationDocuments":[{"file":{"fileName":"vat.pdf","fileContent":"`+file+`"}}]}`, 1)
}

func TestTaxSettingsStage1Registrations(t *testing.T) {
	srv, ts := newBudTestServer(t)
	plain := strings.Replace(taxsEntryJSON, "%s", "", 1)
	// Errors use the Tax Settings model's shape: message plus errorCode.
	if status, out := acctCallREST(t, ts, "tax", "/GetTaxRegistration", `{}`); status != http.StatusNotFound ||
		out["errorCode"] != "ResourceNotFound" || out["message"] == nil {
		t.Fatalf("missing registration: %d %v", status, out)
	}
	if _, out := acctCallREST(t, ts, "tax", "/PutTaxRegistration", `{}`); out["errorCode"] != "FieldValidationFailed" {
		t.Fatalf("missing entry: %v", out)
	}
	if code := taxsErr(t, ts, "PutTaxRegistration", `{"taxRegistrationEntry":{"registrationId":"X","registrationType":"VAT"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("no legal name: %s", code)
	}
	if got := taxsOK(t, ts, "PutTaxRegistration", `{"taxRegistrationEntry":`+plain+`}`); got["status"] != "Verified" {
		t.Fatalf("put: %v", got)
	}
	reg := taxsOK(t, ts, "GetTaxRegistration", `{}`)["taxRegistration"].(map[string]any)
	if reg["legalName"] != "Stackyard GmbH" || reg["status"] != "Verified" {
		t.Fatalf("get: %v", reg)
	}
	taxsErr(t, ts, "PutTaxRegistration", `{"accountId":"999999999999","taxRegistrationEntry":`+plain+`}`, http.StatusBadRequest)

	// Batch put reports per-account errors; the member gets a document.
	batch := taxsOK(t, ts, "BatchPutTaxRegistration", `{"accountIds":["`+acctMember+`","`+acctMember+`","999999999999","999999999999"],"taxRegistrationEntry":`+taxsDocEntry("member vat")+`}`)
	errs := batch["errors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["accountId"] != "999999999999" || batch["status"] != "Pending" {
		t.Fatalf("batch put: %v", batch)
	}
	if n := srv.taxSettings.docBytes; n != len("member vat") { // duplicates stored once
		t.Fatalf("document bytes after batch: %d", n)
	}
	list := taxsOK(t, ts, "ListTaxRegistrations", `{}`)["accountDetails"].([]any)
	if len(list) != 2 {
		t.Fatalf("list: %v", list)
	}
	member := list[1].(map[string]any)["taxRegistration"].(map[string]any)
	if member["status"] != "Pending" || member["jurisdiction"].(map[string]any)["countryCode"] != "DE" {
		t.Fatalf("member registration: %v", member)
	}
	got := taxsOK(t, ts, "GetTaxRegistration", `{"accountId":"`+acctMember+`"}`)["taxRegistration"].(map[string]any)
	if got["status"] != "Verified" {
		t.Fatalf("pending didn't settle: %v", got)
	}
	meta := got["taxDocumentMetadatas"].([]any)[0].(map[string]any)

	// The document goes back to S3.
	if _, err := srv.s3.CreateBucket("tax-docs"); err != nil {
		t.Fatal(err)
	}
	metaJSON := `{"taxDocumentAccessToken":"` + meta["taxDocumentAccessToken"].(string) + `","taxDocumentName":"vat.pdf"}`
	taxsErr(t, ts, "GetTaxRegistrationDocument", `{"taxDocumentMetadata":`+metaJSON+`}`, http.StatusBadRequest)
	doc := taxsOK(t, ts, "GetTaxRegistrationDocument", `{"taxDocumentMetadata":`+metaJSON+`,"destinationS3Location":{"bucket":"tax-docs","prefix":"in"}}`)
	if doc["destinationFilePath"] != "s3://tax-docs/in/vat.pdf" {
		t.Fatalf("document: %v", doc)
	}
	if obj, err := srv.s3.GetObject("tax-docs", "in/vat.pdf"); err != nil || string(obj.Body) != "member vat" {
		t.Fatalf("s3 object: %v %q", err, obj.Body)
	}

	// Deletes.
	del := taxsOK(t, ts, "BatchDeleteTaxRegistration", `{"accountIds":["`+acctMember+`","999999999999"]}`)
	if len(del["errors"].([]any)) != 1 {
		t.Fatalf("batch delete: %v", del)
	}
	taxsOK(t, ts, "DeleteTaxRegistration", `{}`)
	taxsErr(t, ts, "DeleteTaxRegistration", `{}`, http.StatusNotFound)
	if n := srv.taxSettings.docBytes; n != 0 {
		t.Fatalf("document bytes after deletes: %d", n)
	}
}

func TestTaxSettingsStage1DocumentCapRechargedOnUpdate(t *testing.T) {
	srv, ts := newBudTestServer(t)
	big := strings.Repeat("x", taxsMaxDoc)
	taxsErr(t, ts, "PutTaxRegistration", `{"taxRegistrationEntry":`+taxsDocEntry(big+"y")+`}`, http.StatusBadRequest)
	// Replacing a registration re-charges its documents instead of adding them.
	for i := 0; i < 3; i++ {
		taxsOK(t, ts, "PutTaxRegistration", `{"taxRegistrationEntry":`+taxsDocEntry(big)+`}`)
	}
	if n := srv.taxSettings.docBytes; n != taxsMaxDoc {
		t.Fatalf("document bytes: %d", n)
	}
	srv.taxSettings.mu.Lock()
	srv.taxSettings.docBytes = taxsMaxDocBytes // as if the store were full
	srv.taxSettings.mu.Unlock()
	taxsOK(t, ts, "PutTaxRegistration", `{"taxRegistrationEntry":`+taxsDocEntry(big)+`}`) // same size: fits
	taxsErr(t, ts, "PutTaxRegistration", `{"accountId":"`+acctMember+`","taxRegistrationEntry":`+taxsDocEntry("more")+`}`, http.StatusBadRequest)
}

func TestTaxSettingsStage1SupplementalExemptionsInheritance(t *testing.T) {
	_, ts := newBudTestServer(t)
	supp := `{"taxRegistrationEntry":{"registrationId":"MY-1","registrationType":"VAT","legalName":"Stackyard","address":{"postalCode":"50000","countryCode":"MY"}}}`
	put := taxsOK(t, ts, "PutSupplementalTaxRegistration", supp)
	taxsErr(t, ts, "PutSupplementalTaxRegistration", supp, http.StatusConflict)
	if list := taxsOK(t, ts, "ListSupplementalTaxRegistrations", `{}`)["taxRegistrations"].([]any); len(list) != 1 {
		t.Fatalf("supplemental: %v", list)
	}
	taxsOK(t, ts, "DeleteSupplementalTaxRegistration", `{"authorityId":"`+put["authorityId"].(string)+`"}`)
	taxsErr(t, ts, "DeleteSupplementalTaxRegistration", `{"authorityId":"`+put["authorityId"].(string)+`"}`, http.StatusNotFound)

	types := taxsOK(t, ts, "GetTaxExemptionTypes", `{}`)["taxExemptionTypes"].([]any)
	if len(types) != len(taxsExemptionTypes) {
		t.Fatalf("types: %v", types)
	}
	cert := `"exemptionCertificate":{"documentName":"resale.pdf","documentFile":"` + base64.StdEncoding.EncodeToString([]byte("cert")) + `"}`
	taxsErr(t, ts, "PutTaxExemption", `{"accountIds":["`+acctCaller+`"],"authority":{"country":"US","state":"OR"},"exemptionType":"Resale",`+cert+`}`, http.StatusBadRequest)
	taxsErr(t, ts, "PutTaxExemption", `{"accountIds":["`+acctCaller+`"],"authority":{"country":"US","state":"CA"},"exemptionType":"Nope",`+cert+`}`, http.StatusBadRequest)
	if got := taxsOK(t, ts, "PutTaxExemption", `{"accountIds":["`+acctCaller+`"],"authority":{"country":"US","state":"CA"},"exemptionType":"Resale",`+cert+`}`); got["caseId"] == nil {
		t.Fatalf("put exemption: %v", got)
	}
	status := func() string {
		details := taxsOK(t, ts, "BatchGetTaxExemptions", `{"accountIds":["`+acctCaller+`"]}`)["taxExemptionDetailsMap"].(map[string]any)
		return details[acctCaller].(map[string]any)["taxExemptions"].([]any)[0].(map[string]any)["status"].(string)
	}
	if s1, s2 := status(), status(); s1 != "Pending" || s2 != "Valid" {
		t.Fatalf("exemption status: %s then %s", s1, s2)
	}
	failed := taxsOK(t, ts, "BatchGetTaxExemptions", `{"accountIds":["999999999999"]}`)["failedAccounts"].([]any)
	if len(failed) != 1 {
		t.Fatalf("failed accounts: %v", failed)
	}

	// Inheritance needs a management registration, then members report it.
	if got := taxsOK(t, ts, "GetTaxInheritance", `{}`); got["heritageStatus"] != "OptOut" {
		t.Fatalf("inheritance: %v", got)
	}
	taxsErr(t, ts, "PutTaxInheritance", `{"heritageStatus":"OptIn"}`, http.StatusNotFound)
	taxsOK(t, ts, "PutTaxRegistration", `{"taxRegistrationEntry":`+strings.Replace(taxsEntryJSON, "%s", "", 1)+`}`)
	taxsOK(t, ts, "PutTaxInheritance", `{"heritageStatus":"OptIn"}`)
	details := taxsOK(t, ts, "ListTaxExemptions", `{}`)["taxExemptionDetailsMap"].(map[string]any)
	member, ok := details[acctMember].(map[string]any)
	if !ok || member["heritageObtainedDetails"] != true || member["heritageObtainedParentEntity"] != acctCaller {
		t.Fatalf("inherited details: %v", details)
	}
}
