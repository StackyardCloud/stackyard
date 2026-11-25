package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestInvoicingStage1Routing(t *testing.T) {
	_, ts := newC9TestServer(t)
	c9OK(t, ts, invcAPI, "ListInvoiceUnits", `{}`)

	// The cost-management router keeps its AWSBillingAndCostManagement.* traffic.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSBillingAndCostManagement.invoicing_ListInvoiceUnits",
	}, "ce")
	if body := mustBody(t, resp); resp.Header.Get("Content-Type") == invcAPI.contentType {
		t.Fatalf("cost-management request was served by Invoicing: %s", body)
	}
	// Invoicing targets signed for another service aren't served here.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": invcTargetPrefix + "ListInvoiceUnits",
	}, "ce")
	if body := mustBody(t, resp); strings.Contains(string(body), "InvoiceUnits") {
		t.Fatalf("ce-signed request was served by Invoicing: %s", body)
	}
	if code := c9Fail(t, ts, invcAPI, "ListInvoiceUnits", `{"MaxResults":0}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("MaxResults 0: %s", code)
	}
}

func TestInvoicingStage1InvoiceUnits(t *testing.T) {
	srv, ts := newC9TestServer(t)
	unit := func(name, accounts string) string {
		out := c9OK(t, ts, invcAPI, "CreateInvoiceUnit", `{"Name":"`+name+`","InvoiceReceiver":"111122223333","Rule":{"LinkedAccounts":[`+accounts+`]}}`)
		arn, _ := out["InvoiceUnitArn"].(string)
		return arn
	}
	arn := unit("Team A", `"111122223333","444455556666"`)
	for _, c := range []struct{ body, code string }{
		{`{"Name":" lead","InvoiceReceiver":"111122223333","Rule":{"LinkedAccounts":["777788889999"]}}`, "ValidationException"},
		{`{"Name":"Team A","InvoiceReceiver":"111122223333","Rule":{"LinkedAccounts":["777788889999"]}}`, "ValidationException"},
		{`{"Name":"Team B","InvoiceReceiver":"111122223333","Rule":{"LinkedAccounts":["444455556666"]}}`, "ValidationException"},
		{`{"Name":"Team C","InvoiceReceiver":"111122223333","Rule":{}}`, "ValidationException"},
	} {
		if code := c9Fail(t, ts, invcAPI, "CreateInvoiceUnit", c.body, http.StatusBadRequest); code != c.code {
			t.Fatalf("%s: %s", c.body, code)
		}
	}

	time.Sleep(10 * time.Millisecond)
	before := time.Now().UTC()
	time.Sleep(10 * time.Millisecond)
	c9OK(t, ts, invcAPI, "UpdateInvoiceUnit", `{"InvoiceUnitArn":"`+arn+`","Description":"v2","Rule":{"LinkedAccounts":["111122223333"]}}`)
	if got := c9OK(t, ts, invcAPI, "GetInvoiceUnit", `{"InvoiceUnitArn":"`+arn+`"}`); got["Description"] != "v2" {
		t.Fatalf("current: %v", got)
	}
	old := c9OK(t, ts, invcAPI, "GetInvoiceUnit", fmt.Sprintf(`{"InvoiceUnitArn":"%s","AsOf":%.6f}`, arn, float64(before.UnixMicro())/1e6))
	if old["Description"] != "" || len(old["Rule"].(map[string]any)["LinkedAccounts"].([]any)) != 2 {
		t.Fatalf("as of before the update: %v", old)
	}
	if code := c9Fail(t, ts, invcAPI, "GetInvoiceUnit", `{"InvoiceUnitArn":"`+arn+`","AsOf":1000}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("before creation: %s", code)
	}
	// The account freed by the update can join another unit.
	other := unit("Team B", `"444455556666"`)
	if units := c9OK(t, ts, invcAPI, "ListInvoiceUnits", `{"Filters":{"Accounts":["444455556666"]}}`)["InvoiceUnits"].([]any); len(units) != 1 || units[0].(map[string]any)["InvoiceUnitArn"] != other {
		t.Fatalf("filter: %v", units)
	}

	// Revisions are capped, and the account-ID budget follows them.
	for i := 0; i < 2*invcMaxRevisions; i++ {
		c9OK(t, ts, invcAPI, "UpdateInvoiceUnit", fmt.Sprintf(`{"InvoiceUnitArn":"%s","Description":"r%d"}`, arn, i))
	}
	srv.invoicing.mu.Lock()
	revs, ids := len(srv.invoicing.units[arn].revisions), srv.invoicing.accountIDs
	srv.invoicing.mu.Unlock()
	if revs != invcMaxRevisions || ids != invcMaxRevisions*1+1 {
		t.Fatalf("revisions %d, account IDs %d", revs, ids)
	}

	c9OK(t, ts, invcAPI, "TagResource", `{"ResourceArn":"`+arn+`","ResourceTags":[{"Key":"team","Value":"a"}]}`)
	if tags := c9OK(t, ts, invcAPI, "ListTagsForResource", `{"ResourceArn":"`+arn+`"}`)["ResourceTags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	c9OK(t, ts, invcAPI, "DeleteInvoiceUnit", `{"InvoiceUnitArn":"`+arn+`"}`)
	if code := c9Fail(t, ts, invcAPI, "ListTagsForResource", `{"ResourceArn":"`+arn+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted unit tags: %s", code)
	}
	srv.invoicing.mu.Lock()
	ids = srv.invoicing.accountIDs
	srv.invoicing.mu.Unlock()
	if ids != 1 {
		t.Fatalf("account IDs after delete: %d", ids)
	}
}

func TestInvoicingStage1SummariesAndPDF(t *testing.T) {
	_, ts := newC9TestServer(t)
	unit, _ := c9OK(t, ts, invcAPI, "CreateInvoiceUnit", `{"Name":"Ops","InvoiceReceiver":"111122223333","Rule":{"LinkedAccounts":["111122223333"]}}`)["InvoiceUnitArn"].(string)
	list := c9OK(t, ts, invcAPI, "ListInvoiceSummaries", `{"Selector":{"ResourceType":"ACCOUNT_ID","Value":"111122223333"},"Filter":{"BillingPeriod":{"Month":1,"Year":2026}}}`)
	sums := list["InvoiceSummaries"].([]any)
	if len(sums) != 2 || sums[1].(map[string]any)["InvoiceId"] != "INV-111122223333-202601-"+unit[len(unit)-8:] {
		t.Fatalf("summaries: %v", sums)
	}
	again := c9OK(t, ts, invcAPI, "ListInvoiceSummaries", `{"Selector":{"ResourceType":"INVOICE_ID","Value":"INV-111122223333-202601"}}`)["InvoiceSummaries"].([]any)
	if fmt.Sprint(again[0]) != fmt.Sprint(sums[0]) {
		t.Fatalf("not deterministic: %v vs %v", again[0], sums[0])
	}
	if paged := c9OK(t, ts, invcAPI, "ListInvoiceSummaries", `{"Selector":{"ResourceType":"ACCOUNT_ID","Value":"123456789012"},"MaxResults":5}`); len(paged["InvoiceSummaries"].([]any)) != 5 || paged["NextToken"] == nil {
		t.Fatalf("default periods page: %v", paged)
	}
	future := time.Now().AddDate(0, 1, 0).Format("200601")
	if code := c9Fail(t, ts, invcAPI, "GetInvoicePDF", `{"InvoiceId":"INV-123456789012-`+future+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("unissued invoice: %s", code)
	}
	pdf := c9OK(t, ts, invcAPI, "GetInvoicePDF", `{"InvoiceId":"INV-123456789012-202601"}`)["InvoicePDF"].(map[string]any)
	url, _ := pdf["DocumentUrl"].(string)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/pdf" || !strings.HasPrefix(string(body), "%PDF-1.4") || !strings.Contains(string(body), "INV-123456789012-202601") {
		t.Fatalf("pdf: %d %s", resp.StatusCode, body)
	}
	resp, err = http.Get(strings.Replace(url, "INV-123456789012-202601", "INV-123456789012-202602", 1))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("tampered URL: %d", resp.StatusCode)
	}

	profiles := c9OK(t, ts, invcAPI, "BatchGetInvoiceProfile", `{"AccountIds":["111122223333","123456789012"]}`)["Profiles"].([]any)
	if len(profiles) != 2 {
		t.Fatalf("profiles: %v", profiles)
	}
	if code := c9Fail(t, ts, invcAPI, "BatchGetInvoiceProfile", `{"AccountIds":["999999999999"]}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("outside account: %s", code)
	}
}

const invcTestPref = `{"ProcurementPortalName":"SAP_BUSINESS_NETWORK","BuyerDomain":"NetworkID","BuyerIdentifier":"AN-BUYER","SupplierDomain":"NetworkID","SupplierIdentifier":"AN0000001000",
"ProcurementPortalSharedSecret":"s3cret","EinvoiceDeliveryEnabled":true,"PurchaseOrderRetrievalEnabled":true,
"EinvoiceDeliveryPreference":{"EinvoiceDeliveryDocumentTypes":["AWS_CLOUD_INVOICE"],"Protocol":"CXML","PurchaseOrderDataSources":[],"ConnectionTestingMethod":"PROD_ENV_DOLLAR_TEST","EinvoiceDeliveryActivationDate":1798761600},
"Contacts":[{"Name":"Ann","Email":"ann@example.com"}]`

func TestInvoicingStage1ProcurementPreferences(t *testing.T) {
	_, ts := newC9TestServer(t)
	arn, _ := c9OK(t, ts, invcAPI, "CreateProcurementPortalPreference", invcTestPref+`}`)["ProcurementPortalPreferenceArn"].(string)
	if code := c9Fail(t, ts, invcAPI, "CreateProcurementPortalPreference", invcTestPref+`}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("duplicate: %s", code)
	}
	bad := strings.Replace(invcTestPref, "AN0000001000", "AN9", 1)
	if code := c9Fail(t, ts, invcAPI, "CreateProcurementPortalPreference", bad+`}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown supplier: %s", code)
	}
	get := func() map[string]any {
		return c9OK(t, ts, invcAPI, "GetProcurementPortalPreference", `{"ProcurementPortalPreferenceArn":"`+arn+`"}`)["ProcurementPortalPreference"].(map[string]any)
	}
	p := get()
	if p["EinvoiceDeliveryPreferenceStatus"] != "PENDING_VERIFICATION" || p["ProcurementPortalSharedSecret"] != nil || p["PurchaseOrderRetrievalEndpoint"] == nil {
		t.Fatalf("created: %v", p)
	}
	if code := c9Fail(t, ts, invcAPI, "UpdateProcurementPortalPreferenceStatus", `{"ProcurementPortalPreferenceArn":"`+arn+`","EinvoiceDeliveryPreferenceStatus":"ACTIVE"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("active before verification: %s", code)
	}
	if code := c9Fail(t, ts, invcAPI, "VerifyProcurementPortalValidation", `{"ProcurementPortalPreferenceArn":"`+arn+`","Code":"123456"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("verify before send: %s", code)
	}
	c9OK(t, ts, invcAPI, "SendProcurementPortalValidation", `{"ProcurementPortalPreferenceArn":"`+arn+`"}`)
	wrong := "000000"
	if invcCode(arn, 1) == wrong {
		wrong = "000001"
	}
	if code := c9Fail(t, ts, invcAPI, "VerifyProcurementPortalValidation", `{"ProcurementPortalPreferenceArn":"`+arn+`","Code":"`+wrong+`"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("wrong code: %s", code)
	}
	c9OK(t, ts, invcAPI, "VerifyProcurementPortalValidation", `{"ProcurementPortalPreferenceArn":"`+arn+`","Code":"`+invcCode(arn, 1)+`"}`)
	c9OK(t, ts, invcAPI, "UpdateProcurementPortalPreferenceStatus", `{"ProcurementPortalPreferenceArn":"`+arn+`","EinvoiceDeliveryPreferenceStatus":"ACTIVE"}`)
	if p = get(); p["EinvoiceDeliveryPreferenceStatus"] != "ACTIVE" || p["PurchaseOrderRetrievalPreferenceStatus"] != "VALIDATED" || p["Version"] != float64(4) {
		t.Fatalf("after verification: %v", p)
	}

	// Put replaces the configuration and asks for verification again.
	unit, _ := c9OK(t, ts, invcAPI, "CreateInvoiceUnit", `{"Name":"Ops","InvoiceReceiver":"111122223333","Rule":{"LinkedAccounts":["111122223333"]}}`)["InvoiceUnitArn"].(string)
	put := `{"ProcurementPortalPreferenceArn":"` + arn + `","EinvoiceDeliveryEnabled":false,"PurchaseOrderRetrievalEnabled":true,"Contacts":[{"Email":"bo@example.com"}],"Selector":{"InvoiceUnitArns":["` + unit + `"]}}`
	c9OK(t, ts, invcAPI, "PutProcurementPortalPreference", put)
	if p = get(); p["EinvoiceDeliveryPreferenceStatus"] != nil || p["PurchaseOrderRetrievalPreferenceStatus"] != "PENDING_VERIFICATION" {
		t.Fatalf("after put: %v", p)
	}
	if code := c9Fail(t, ts, invcAPI, "UpdateProcurementPortalPreferenceStatus", `{"ProcurementPortalPreferenceArn":"`+arn+`","EinvoiceDeliveryPreferenceStatus":"SUSPENDED"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("status of disabled feature: %s", code)
	}
	if code := c9Fail(t, ts, invcAPI, "DeleteInvoiceUnit", `{"InvoiceUnitArn":"`+unit+`"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("delete of selected unit: %s", code)
	}

	suppliers := c9OK(t, ts, invcAPI, "ListProcurementPortalSuppliers", `{"PortalIdentifier":"sap-business-network"}`)["ProcurementPortalSuppliers"].([]any)
	if len(suppliers) != 6 {
		t.Fatalf("suppliers: %v", suppliers)
	}
	if code := c9Fail(t, ts, invcAPI, "ListProcurementPortalSuppliers", `{"PortalIdentifier":"ariba"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("unknown portal: %s", code)
	}
	c9OK(t, ts, invcAPI, "DeleteProcurementPortalPreference", `{"ProcurementPortalPreferenceArn":"`+arn+`"}`)
	if prefs := c9OK(t, ts, invcAPI, "ListProcurementPortalPreferences", `{}`)["ProcurementPortalPreferences"].([]any); len(prefs) != 0 {
		t.Fatalf("after delete: %v", prefs)
	}
}
