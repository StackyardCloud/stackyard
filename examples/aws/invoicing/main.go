// Command invoicing groups accounts into an AWS Invoicing invoice unit on
// Stackyard, lists the unit's invoices for last month, downloads one as a
// PDF, and connects a Coupa procurement portal preference, verifying it with
// the code Stackyard "emails" to the contact.
//
// The SDK has no Invoicing module in this repository's go.mod, so requests
// are signed by hand.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Invoicing client using %s\n", c.endpoint)
	n := time.Now().UnixNano()
	receiver := fmt.Sprintf("%012d", n%1_000_000_000_000)
	member := fmt.Sprintf("%012d", (n+1)%1_000_000_000_000)

	unit := c.call("CreateInvoiceUnit", map[string]any{
		"Name": fmt.Sprintf("research-%d", n%1_000_000), "InvoiceReceiver": receiver, "Description": "Research accounts",
		"Rule": map[string]any{"LinkedAccounts": []any{receiver, member}},
	})["InvoiceUnitArn"].(string)
	fmt.Printf("invoice unit %s\n", unit)
	c.call("UpdateInvoiceUnit", map[string]any{"InvoiceUnitArn": unit, "Description": "Research and lab accounts"})

	last := time.Now().UTC().AddDate(0, -2, 0)
	sums := c.call("ListInvoiceSummaries", map[string]any{
		"Selector": map[string]any{"ResourceType": "ACCOUNT_ID", "Value": receiver},
		"Filter":   map[string]any{"BillingPeriod": map[string]any{"Month": int(last.Month()), "Year": last.Year()}},
	})["InvoiceSummaries"].([]any)
	for _, s := range sums {
		sm := s.(map[string]any)
		amount := sm["BaseCurrencyAmount"].(map[string]any)
		fmt.Printf("invoice %s: %s %s for %v accounts\n", sm["InvoiceId"], amount["TotalAmount"], amount["CurrencyCode"], sm["BillSourceAccountsTotalCount"])
	}
	invoice := sums[len(sums)-1].(map[string]any)["InvoiceId"].(string)
	url := c.call("GetInvoicePDF", map[string]any{"InvoiceId": invoice})["InvoicePDF"].(map[string]any)["DocumentUrl"].(string)
	resp, err := http.Get(url)
	if err != nil {
		exitf("download: %v", err)
	}
	pdf, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.HasPrefix(pdf, []byte("%PDF")) {
		exitf("download: HTTP %d", resp.StatusCode)
	}
	fmt.Printf("downloaded %s.pdf (%d bytes)\n", invoice, len(pdf))

	pref := c.call("CreateProcurementPortalPreference", map[string]any{
		"ProcurementPortalName": "COUPA", "BuyerDomain": "NetworkID", "BuyerIdentifier": fmt.Sprintf("buyer-%d", n),
		"SupplierDomain": "NetworkID", "SupplierIdentifier": "CP0000001000", "ProcurementPortalSharedSecret": "example-secret",
		"Selector":                map[string]any{"InvoiceUnitArns": []any{unit}},
		"EinvoiceDeliveryEnabled": true, "PurchaseOrderRetrievalEnabled": false,
		"EinvoiceDeliveryPreference": map[string]any{
			"EinvoiceDeliveryDocumentTypes": []any{"AWS_CLOUD_INVOICE"}, "Protocol": "CXML",
			"ConnectionTestingMethod": "PROD_ENV_DOLLAR_TEST", "EinvoiceDeliveryActivationDate": time.Now().Unix(),
			"PurchaseOrderDataSources": []any{map[string]any{"EinvoiceDeliveryDocumentType": "AWS_CLOUD_INVOICE", "PurchaseOrderDataSourceType": "PURCHASE_ORDER_NOT_REQUIRED"}},
		},
		"Contacts": []any{map[string]any{"Name": "Accounts payable", "Email": "ap@example.com"}},
	})["ProcurementPortalPreferenceArn"].(string)
	c.call("SendProcurementPortalValidation", map[string]any{"ProcurementPortalPreferenceArn": pref})
	c.call("VerifyProcurementPortalValidation", map[string]any{"ProcurementPortalPreferenceArn": pref, "Code": validationCode(pref, 1)})
	c.call("UpdateProcurementPortalPreferenceStatus", map[string]any{"ProcurementPortalPreferenceArn": pref, "EinvoiceDeliveryPreferenceStatus": "ACTIVE"})
	got := c.call("GetProcurementPortalPreference", map[string]any{"ProcurementPortalPreferenceArn": pref})["ProcurementPortalPreference"].(map[string]any)
	fmt.Printf("Coupa e-invoice delivery: %s (version %v)\n", got["EinvoiceDeliveryPreferenceStatus"], got["Version"])

	c.call("DeleteProcurementPortalPreference", map[string]any{"ProcurementPortalPreferenceArn": pref})
	c.call("DeleteInvoiceUnit", map[string]any{"InvoiceUnitArn": unit})
	fmt.Println("cleaned up")
}

// validationCode is the code Stackyard sends for the n-th validation request.
func validationCode(arn string, n int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s#%d", arn, n)))
	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(sum[:4])%1_000_000)
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "Invoicing."+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, "invoicing", c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s: decode: %v", op, err)
	}
	return out
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
