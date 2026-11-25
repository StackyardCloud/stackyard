package server

import (
	"net/http"
	"testing"
)

const acctMember = "210987654321"

func TestAccountStage1AccountResolution(t *testing.T) {
	_, ts := newBudTestServer(t)
	info := acctOK(t, ts, "GetAccountInformation", `{}`)
	if info["AccountId"] != acctCaller || info["AccountName"] != "stackyard-management" || info["AccountState"] != "ACTIVE" {
		t.Fatalf("caller info: %v", info)
	}
	acctOK(t, ts, "PutAccountName", `{"AccountId":"`+acctMember+`","AccountName":"renamed member"}`)
	if got := acctOK(t, ts, "GetAccountInformation", `{"AccountId":"`+acctMember+`"}`); got["AccountName"] != "renamed member" {
		t.Fatalf("member rename: %v", got)
	}
	if got := acctOK(t, ts, "GetAccountInformation", `{}`); got["AccountName"] != "stackyard-management" {
		t.Fatalf("rename leaked into the caller: %v", got)
	}
	if code := acctErr(t, ts, "GetAccountInformation", `{"AccountId":"`+acctCaller+`"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("own AccountId: %s", code)
	}
	if code := acctErr(t, ts, "GetAccountInformation", `{"AccountId":"999999999999"}`, http.StatusForbidden); code != "AccessDeniedException" {
		t.Fatalf("outside account: %s", code)
	}
	if got := acctOK(t, ts, "GetGovCloudAccountInformation", `{}`); got["GovCloudAccountId"] != "123456789013" {
		t.Fatalf("govcloud: %v", got)
	}
	if code := acctErr(t, ts, "GetGovCloudAccountInformation", `{"StandardAccountId":"`+acctMember+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("member govcloud: %s", code)
	}
}

func TestAccountStage1Regions(t *testing.T) {
	_, ts := newBudTestServer(t)
	status := func() string {
		return acctOK(t, ts, "GetRegionOptStatus", `{"RegionName":"af-south-1"}`)["RegionOptStatus"].(string)
	}
	if s := status(); s != "DISABLED" {
		t.Fatalf("initial: %s", s)
	}
	acctOK(t, ts, "EnableRegion", `{"RegionName":"af-south-1"}`)
	if code := acctErr(t, ts, "DisableRegion", `{"RegionName":"af-south-1"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("disable while enabling: %s", code)
	}
	if s := status(); s != "ENABLING" {
		t.Fatalf("after enable: %s", s)
	}
	if s := status(); s != "ENABLED" {
		t.Fatalf("settled: %s", s)
	}
	acctOK(t, ts, "EnableRegion", `{"RegionName":"af-south-1"}`) // already enabled: no-op
	acctOK(t, ts, "DisableRegion", `{"RegionName":"af-south-1"}`)
	if s, s2 := status(), status(); s != "DISABLING" || s2 != "DISABLED" {
		t.Fatalf("disable: %s then %s", s, s2)
	}
	// Errors use the Account model's shape: message plus a reason.
	if status, out := acctCallREST(t, ts, "account", "/enableRegion", `{"RegionName":"us-east-1"}`); status != http.StatusBadRequest ||
		out["__type"] != "ValidationException" || out["reason"] != "invalidRegionOptTarget" || out["message"] == nil {
		t.Fatalf("default region: %d %v", status, out)
	}
	if _, out := acctCallREST(t, ts, "account", "/putAccountName", `{}`); out["reason"] != "fieldValidationFailed" {
		t.Fatalf("missing member: %v", out)
	}
	if code := acctErr(t, ts, "GetRegionOptStatus", `{"RegionName":"mars-north-1"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown region: %s", code)
	}
	// Member Regions are separate from the caller's.
	acctOK(t, ts, "EnableRegion", `{"AccountId":"`+acctMember+`","RegionName":"me-south-1"}`)
	if got := acctOK(t, ts, "GetRegionOptStatus", `{"RegionName":"me-south-1"}`); got["RegionOptStatus"] != "DISABLED" {
		t.Fatalf("member change leaked: %v", got)
	}

	all := acctOK(t, ts, "ListRegions", `{"MaxResults":50}`)["Regions"].([]any)
	if len(all) != len(acctRegionNames) {
		t.Fatalf("regions: %d", len(all))
	}
	page := acctOK(t, ts, "ListRegions", `{"MaxResults":5,"RegionOptStatusContains":["ENABLED_BY_DEFAULT"]}`)
	if len(page["Regions"].([]any)) != 5 || page["NextToken"] == nil {
		t.Fatalf("page: %v", page)
	}
	rest := acctOK(t, ts, "ListRegions", `{"MaxResults":50,"RegionOptStatusContains":["ENABLED_BY_DEFAULT"],"NextToken":"`+page["NextToken"].(string)+`"}`)
	if n := len(rest["Regions"].([]any)); n != len(acctDefaultRegions)-5 || rest["NextToken"] != nil {
		t.Fatalf("second page: %d %v", n, rest["NextToken"])
	}
}

func TestAccountStage1Contacts(t *testing.T) {
	_, ts := newBudTestServer(t)
	if code := acctErr(t, ts, "GetAlternateContact", `{"AlternateContactType":"BILLING"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("missing contact: %s", code)
	}
	acctOK(t, ts, "PutAlternateContact", `{"AlternateContactType":"BILLING","Name":"Ana","Title":"CFO","EmailAddress":"ana@example.com","PhoneNumber":"+1 206 555 0100"}`)
	got := acctOK(t, ts, "GetAlternateContact", `{"AlternateContactType":"BILLING"}`)["AlternateContact"].(map[string]any)
	if got["Name"] != "Ana" || got["AlternateContactType"] != "BILLING" {
		t.Fatalf("contact: %v", got)
	}
	if code := acctErr(t, ts, "PutAlternateContact", `{"AlternateContactType":"BILLING","Name":"Ana","Title":"CFO","EmailAddress":"not-an-email","PhoneNumber":"1"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad email: %s", code)
	}
	acctOK(t, ts, "DeleteAlternateContact", `{"AlternateContactType":"BILLING"}`)
	acctErr(t, ts, "DeleteAlternateContact", `{"AlternateContactType":"BILLING"}`, http.StatusNotFound)

	// Contact information and phone verification.
	acctErr(t, ts, "SendPhoneNumberVerification", `{}`, http.StatusNotFound)
	info := `{"ContactInformation":{"FullName":"Ana","AddressLine1":"1 Main St","City":"Seattle","PostalCode":"98101","CountryCode":"US","PhoneNumber":"+1 206 555 0100"}}`
	acctOK(t, ts, "PutContactInformation", info)
	if got := acctOK(t, ts, "GetContactInformation", `{}`); got["VerificationStatus"] != "UNVERIFIED" {
		t.Fatalf("contact info: %v", got)
	}
	acctErr(t, ts, "VerifyPhoneNumber", `{"Otp":"ABC123"}`, http.StatusConflict)
	if got := acctOK(t, ts, "SendPhoneNumberVerification", `{}`); got["Status"] != "PENDING" {
		t.Fatalf("send: %v", got)
	}
	if code := acctErr(t, ts, "VerifyPhoneNumber", `{"Otp":"ZZZZZZ"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("wrong otp: %s", code)
	}
	otp := acctOTP(acctCaller, "+1 206 555 0100")
	if got := acctOK(t, ts, "VerifyPhoneNumber", `{"Otp":"`+otp+`"}`); got["Status"] != "VERIFIED" {
		t.Fatalf("verify: %v", got)
	}
	acctErr(t, ts, "SendPhoneNumberVerification", `{}`, http.StatusConflict)
	acctOK(t, ts, "PutContactInformation", info) // same number: stays verified
	if got := acctOK(t, ts, "GetContactInformation", `{}`); got["VerificationStatus"] != "VERIFIED" {
		t.Fatalf("same number reset verification: %v", got)
	}
}

func TestAccountStage1PrimaryEmail(t *testing.T) {
	_, ts := newBudTestServer(t)
	member := `"AccountId":"` + acctMember + `"`
	if got := acctOK(t, ts, "GetPrimaryEmail", `{`+member+`}`); got["PrimaryEmail"] != "member@example.com" {
		t.Fatalf("email: %v", got)
	}
	acctErr(t, ts, "GetPrimaryEmailUpdateStatus", `{`+member+`}`, http.StatusNotFound)
	if code := acctErr(t, ts, "StartPrimaryEmailUpdate", `{`+member+`,"PrimaryEmail":"owner@example.com"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("email in use: %s", code)
	}
	if got := acctOK(t, ts, "StartPrimaryEmailUpdate", `{`+member+`,"PrimaryEmail":"new@example.com"}`); got["Status"] != "PENDING" {
		t.Fatalf("start: %v", got)
	}
	acctErr(t, ts, "AcceptPrimaryEmailUpdate", `{`+member+`,"PrimaryEmail":"new@example.com","Otp":"ZZZZZZ"}`, http.StatusBadRequest)
	acctErr(t, ts, "AcceptPrimaryEmailUpdate", `{`+member+`,"PrimaryEmail":"other@example.com","Otp":"ZZZZZZ"}`, http.StatusNotFound)
	otp := acctOTP(acctMember, "new@example.com")
	if got := acctOK(t, ts, "AcceptPrimaryEmailUpdate", `{`+member+`,"PrimaryEmail":"new@example.com","Otp":"`+otp+`"}`); got["Status"] != "ACCEPTED" {
		t.Fatalf("accept: %v", got)
	}
	acctErr(t, ts, "AcceptPrimaryEmailUpdate", `{`+member+`,"PrimaryEmail":"new@example.com","Otp":"`+otp+`"}`, http.StatusConflict)
	if got := acctOK(t, ts, "GetPrimaryEmail", `{`+member+`}`); got["PrimaryEmail"] != "new@example.com" {
		t.Fatalf("email after accept: %v", got)
	}
	first := acctOK(t, ts, "GetPrimaryEmailUpdateStatus", `{`+member+`}`)
	second := acctOK(t, ts, "GetPrimaryEmailUpdateStatus", `{`+member+`}`)
	if first["Status"] != "ACCEPTED" || second["Status"] != "COMPLETED" || second["UpdatedAt"] == nil {
		t.Fatalf("status: %v then %v", first, second)
	}
	// The old address is free again; the new one is now taken.
	acctErr(t, ts, "StartPrimaryEmailUpdate", `{`+member+`,"PrimaryEmail":"new@example.com"}`, http.StatusConflict)
	// The management account can't name itself.
	acctErr(t, ts, "GetPrimaryEmail", `{"AccountId":"`+acctCaller+`"}`, http.StatusBadRequest)
}
