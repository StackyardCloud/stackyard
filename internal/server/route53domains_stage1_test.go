package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const r53dTestContact = `{"FirstName":"Ada","LastName":"Lovelace","ContactType":"PERSON","AddressLine1":"1 Main St","City":"Seattle","State":"WA","CountryCode":"US","ZipCode":"98101","PhoneNumber":"+1.2065550100","Email":"ada@example.com"}`

func r53dRegisterBody(name string, years int, extra string) string {
	return fmt.Sprintf(`{"DomainName":%q,"DurationInYears":%d,"AdminContact":%s,"RegistrantContact":%s,"TechContact":%s%s}`,
		name, years, r53dTestContact, r53dTestContact, r53dTestContact, extra)
}

func TestRoute53DomainsStage1RegisterLifecycle(t *testing.T) {
	srv, ts := newRoute53DomainsTestServer(t)
	if out := r53dOK(t, ts, "CheckDomainAvailability", `{"DomainName":"stackyard-demo.com"}`); out["Availability"] != "AVAILABLE" {
		t.Fatalf("availability: %v", out)
	}
	if out := r53dOK(t, ts, "CheckDomainAvailability", `{"DomainName":"example.com"}`); out["Availability"] != "UNAVAILABLE" {
		t.Fatalf("reserved name: %v", out)
	}
	// Members outside the input shape are rejected, so callers can't reach store fields.
	if code := r53dErrCode(t, ts, "RegisterDomain", r53dRegisterBody("stackyard-demo.com", 2, `,"_internal":"x"`)); code != "InvalidInput" {
		t.Fatalf("unknown member: %s", code)
	}
	op := r53dOK(t, ts, "RegisterDomain", r53dRegisterBody("stackyard-demo.com", 2, ""))["OperationId"].(string)
	for _, want := range []string{"IN_PROGRESS", "SUCCESSFUL", "SUCCESSFUL"} {
		if got := r53dOK(t, ts, "GetOperationDetail", fmt.Sprintf(`{"OperationId":%q}`, op))["Status"]; got != want {
			t.Fatalf("operation status: %v, want %s", got, want)
		}
	}
	detail := r53dOK(t, ts, "GetDomainDetail", `{"DomainName":"stackyard-demo.com"}`)
	ns := detail["Nameservers"].([]any)
	if len(ns) != 4 || !strings.Contains(ns[0].(map[string]any)["Name"].(string), "awsdns") {
		t.Fatalf("name servers: %v", ns)
	}
	if exp := time.Unix(int64(detail["ExpirationDate"].(float64)), 0); exp.Year() != time.Now().Year()+2 {
		t.Fatalf("expiry: %v", exp)
	}
	if detail["RegistrantContact"].(map[string]any)["Email"] != "ada@example.com" || detail["AdminPrivacy"] != true {
		t.Fatalf("detail: %v", detail)
	}
	srv.route53Domains.mu.Lock()
	zone := srv.route53Domains.domains["stackyard-demo.com"].zoneID
	srv.route53Domains.mu.Unlock()
	if zone == "" {
		t.Fatal("registration didn't create a hosted zone")
	}
	if out := r53dOK(t, ts, "CheckDomainAvailability", `{"DomainName":"stackyard-demo.com"}`); out["Availability"] != "UNAVAILABLE" {
		t.Fatalf("registered domain: %v", out)
	}
	if code := r53dErrCode(t, ts, "RegisterDomain", r53dRegisterBody("stackyard-demo.com", 1, "")); code != "InvalidInput" {
		t.Fatalf("duplicate registration: %s", code)
	}
	for body, want := range map[string]string{
		r53dRegisterBody("x.badtld", 1, ""):                                             "UnsupportedTLD",
		r53dRegisterBody("toolong.com", 11, ""):                                         "InvalidInput", // model range 1..10
		r53dRegisterBody("co-years.co", 6, ""):                                          "TLDRulesViolation",
		r53dRegisterBody("sub.domain.com", 1, ""):                                       "InvalidInput",
		r53dRegisterBody("pricey.io", 1, `,"Consent":{"MaxPrice":10,"Currency":"USD"}`): "InvalidInput",
		strings.Replace(r53dRegisterBody("bad-country.com", 1, ""), `"US"`, `"ZZ"`, 1):  "InvalidInput",
	} {
		if code := r53dErrCode(t, ts, "RegisterDomain", body); code != want {
			t.Errorf("%.60s: %s, want %s", body, code, want)
		}
	}

	// Renew needs the current expiry year; billing records registration and renewal.
	year := time.Now().Year() + 2
	if code := r53dErrCode(t, ts, "RenewDomain", fmt.Sprintf(`{"DomainName":"stackyard-demo.com","CurrentExpiryYear":%d}`, year-1)); code != "InvalidInput" {
		t.Fatalf("wrong expiry year: %s", code)
	}
	r53dOK(t, ts, "RenewDomain", fmt.Sprintf(`{"DomainName":"stackyard-demo.com","DurationInYears":1,"CurrentExpiryYear":%d}`, year))
	bills := r53dOK(t, ts, "ViewBilling", `{}`)["BillingRecords"].([]any)
	if len(bills) != 2 || bills[0].(map[string]any)["Price"] != float64(30) || bills[1].(map[string]any)["Operation"] != "RENEW_DOMAIN" {
		t.Fatalf("billing: %v", bills)
	}

	// Delete, then restore within the redemption window.
	r53dOK(t, ts, "DeleteDomain", `{"DomainName":"stackyard-demo.com"}`)
	if code := r53dErrCode(t, ts, "GetDomainDetail", `{"DomainName":"stackyard-demo.com"}`); code != "InvalidInput" {
		t.Fatalf("deleted domain: %s", code)
	}
	r53dOK(t, ts, "RestoreDomain", `{"DomainName":"stackyard-demo.com"}`)
	r53dOK(t, ts, "GetDomainDetail", `{"DomainName":"stackyard-demo.com"}`)
	if code := r53dErrCode(t, ts, "RestoreDomain", `{"DomainName":"never-deleted.com"}`); code != "InvalidInput" {
		t.Fatalf("restore without delete: %s", code)
	}
}

func TestRoute53DomainsStage1TransfersTagsAndLists(t *testing.T) {
	_, ts := newRoute53DomainsTestServer(t)
	for _, name := range []string{"alpha.com", "beta.net", "gamma.co.uk"} {
		r53dOK(t, ts, "RegisterDomain", r53dRegisterBody(name, 1, ""))
	}
	page := r53dOK(t, ts, "ListDomains", `{"MaxItems":2}`)
	if len(page["Domains"].([]any)) != 2 || page["NextPageMarker"] != "2" {
		t.Fatalf("page 1: %v", page)
	}
	if rest := r53dOK(t, ts, "ListDomains", `{"MaxItems":2,"Marker":"2"}`); len(rest["Domains"].([]any)) != 1 || rest["NextPageMarker"] != nil {
		t.Fatalf("page 2: %v", rest)
	}
	if out := r53dOK(t, ts, "ListDomains", `{"FilterConditions":[{"Name":"DomainName","Operator":"BEGINS_WITH","Values":["be"]}]}`); len(out["Domains"].([]any)) != 1 {
		t.Fatalf("filter: %v", out)
	}
	if code := r53dErrCode(t, ts, "ListDomains", `{"Marker":"99"}`); code != "InvalidInput" {
		t.Fatalf("bad marker: %s", code)
	}

	// Transfer to another account: password only from the transfer call, then accept removes the domain.
	if code := r53dErrCode(t, ts, "TransferDomainToAnotherAwsAccount", `{"DomainName":"alpha.com","AccountId":"123456789012"}`); code != "InvalidInput" {
		t.Fatalf("transfer to own account: %s", code)
	}
	tr := r53dOK(t, ts, "TransferDomainToAnotherAwsAccount", `{"DomainName":"alpha.com","AccountId":"210987654321"}`)
	if code := r53dErrCode(t, ts, "TransferDomainToAnotherAwsAccount", `{"DomainName":"alpha.com","AccountId":"210987654321"}`); code != "DuplicateRequest" {
		t.Fatalf("second transfer: %s", code)
	}
	if out := r53dOK(t, ts, "CheckDomainTransferability", `{"DomainName":"alpha.com"}`); out["Transferability"].(map[string]any)["Transferable"] != "DOMAIN_IN_ANOTHER_ACCOUNT" {
		t.Fatalf("transferability: %v", out)
	}
	if code := r53dErrCode(t, ts, "AcceptDomainTransferFromAnotherAwsAccount", `{"DomainName":"alpha.com","Password":"wrong"}`); code != "InvalidInput" {
		t.Fatalf("wrong password: %s", code)
	}
	r53dOK(t, ts, "AcceptDomainTransferFromAnotherAwsAccount", fmt.Sprintf(`{"DomainName":"alpha.com","Password":%q}`, tr["Password"]))
	if code := r53dErrCode(t, ts, "GetDomainDetail", `{"DomainName":"alpha.com"}`); code != "InvalidInput" {
		t.Fatalf("transferred domain still here: %s", code)
	}
	ops := r53dOK(t, ts, "ListOperations", `{"Type":["INTERNAL_TRANSFER_OUT_DOMAIN"]}`)["Operations"].([]any)
	if len(ops) != 1 || ops[0].(map[string]any)["Status"] != "SUCCESSFUL" {
		t.Fatalf("transfer-out operation: %v", ops)
	}

	// Push is for .uk only; it hands the domain to another registrar.
	if code := r53dErrCode(t, ts, "PushDomain", `{"DomainName":"beta.net","Target":"TAG"}`); code != "TLDRulesViolation" {
		t.Fatalf("push .net: %s", code)
	}
	r53dOK(t, ts, "PushDomain", `{"DomainName":"gamma.co.uk","Target":"TAG"}`)

	// Tags.
	r53dOK(t, ts, "UpdateTagsForDomain", `{"DomainName":"beta.net","TagsToUpdate":[{"Key":"team","Value":"a"},{"Key":"env","Value":"dev"}]}`)
	if code := r53dErrCode(t, ts, "UpdateTagsForDomain", `{"DomainName":"beta.net","TagsToUpdate":[{"Key":"aws:x","Value":"a"}]}`); code != "InvalidInput" {
		t.Fatalf("aws: tag: %s", code)
	}
	r53dOK(t, ts, "DeleteTagsForDomain", `{"DomainName":"beta.net","TagsToDelete":["env"]}`)
	if tags := r53dOK(t, ts, "ListTagsForDomain", `{"DomainName":"beta.net"}`)["TagList"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}

	// Auth code comes only from RetrieveDomainAuthCode.
	if code := r53dOK(t, ts, "RetrieveDomainAuthCode", `{"DomainName":"beta.net"}`)["AuthCode"].(string); len(code) != 16 {
		t.Fatalf("auth code: %q", code)
	}
	if strings.Contains(fmt.Sprint(r53dOK(t, ts, "GetDomainDetail", `{"DomainName":"beta.net"}`)), "AuthCode") {
		t.Fatal("GetDomainDetail leaked the auth code")
	}

	// Transfer lock shows in the status list.
	r53dOK(t, ts, "DisableDomainTransferLock", `{"DomainName":"beta.net"}`)
	if status := fmt.Sprint(r53dOK(t, ts, "GetDomainDetail", `{"DomainName":"beta.net"}`)["StatusList"]); strings.Contains(status, "clientTransferProhibited") {
		t.Fatalf("lock disabled but still listed: %s", status)
	}
}

// RFC 6605 §6.1: example.net KSK, key tag 55648, SHA-256 DS digest.
func TestRoute53DomainsStage1DNSSECMatchesRFC6605(t *testing.T) {
	key, _ := base64.StdEncoding.DecodeString("GojIhhXUN/u4v54ZQqGSnyhWJwaubCvTmeexv7bR6edbkrSqQpF64cYbcB7wNcP+e+MAnLr+Wi9xMWyQLc8NAA==")
	rdata := append([]byte{1, 1, 3, 13}, key...) // flags 257, protocol 3, algorithm 13
	if tag := r53dKeyTag(rdata); tag != 55648 {
		t.Fatalf("key tag %d, want 55648", tag)
	}
	digest := sha256.Sum256(append(r53dWireName("example.net"), rdata...))
	if got := hex.EncodeToString(digest[:]); got != "b4c8c1fe2e7477127b27115656ad6256f424625bf5c1e2770ce6d6e37df61d17" {
		t.Fatalf("digest %s", got)
	}

	_, ts := newRoute53DomainsTestServer(t)
	r53dOK(t, ts, "RegisterDomain", r53dRegisterBody("signed.dev", 1, ""))
	attrs := `{"Algorithm":13,"Flags":257,"PublicKey":"GojIhhXUN/u4v54ZQqGSnyhWJwaubCvTmeexv7bR6edbkrSqQpF64cYbcB7wNcP+e+MAnLr+Wi9xMWyQLc8NAA=="}`
	r53dOK(t, ts, "AssociateDelegationSignerToDomain", `{"DomainName":"signed.dev","SigningAttributes":`+attrs+`}`)
	if code := r53dErrCode(t, ts, "AssociateDelegationSignerToDomain", `{"DomainName":"signed.dev","SigningAttributes":`+attrs+`}`); code != "DuplicateRequest" {
		t.Fatalf("same key twice: %s", code)
	}
	if code := r53dErrCode(t, ts, "AssociateDelegationSignerToDomain", `{"DomainName":"signed.dev","SigningAttributes":{"Algorithm":13,"Flags":1,"PublicKey":"AAAA"}}`); code != "InvalidInput" {
		t.Fatalf("bad flags: %s", code)
	}
	detail := r53dOK(t, ts, "GetDomainDetail", `{"DomainName":"signed.dev"}`)
	k := detail["DnssecKeys"].([]any)[0].(map[string]any)
	if detail["DnsSec"] != "ENABLED" || k["KeyTag"] != float64(55648) {
		t.Fatalf("dnssec: %v", detail)
	}
	r53dOK(t, ts, "DisassociateDelegationSignerFromDomain", fmt.Sprintf(`{"DomainName":"signed.dev","Id":%q}`, k["Id"]))
	if status, out := r53dCall(t, ts, "GetDomainDetail", `{"DomainName":"signed.dev"}`); status != http.StatusOK || out["DnsSec"] != "DISABLED" {
		t.Fatalf("after disassociate: %v", out)
	}
}
