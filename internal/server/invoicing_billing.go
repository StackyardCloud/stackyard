package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Invoices are derived, not stored. Every issued billing period (issued on
// the 2nd of the following month) has one invoice per account,
// INV-<account>-<yyyymm>, plus one per invoice unit the account receives,
// INV-<account>-<yyyymm>-<unit id>. Amounts come from a hash of the invoice
// ID. GetInvoicePDF returns a signed, expiring URL on this server that
// renders the invoice as a one-page PDF.

const (
	invcEntity        = "Amazon Web Services, Inc."
	invcPDFPath       = "/_stackyard/invoicing/invoices/"
	invcPDFLifetime   = time.Hour
	invcMaxPeriods    = 120 // billing periods one summary request may cover
	invcDefaultPeriod = 12  // periods listed when no filter is given
)

var invcAccountRe = regexp.MustCompile(`^\d{12}$`)

var invcInvoiceRe = regexp.MustCompile(`^INV-(\d{12})-(\d{4})(\d{2})(?:-([0-9a-f]{8}))?$`)

type invcInvoice struct {
	id, account string
	year, month int
	unit        *invcUnit // nil for the account's own invoice
	billSource  []string
}

func invcIssued(year, month int) time.Time {
	return time.Date(year, time.Month(month)+1, 2, 0, 0, 0, 0, time.UTC)
}

// invoice resolves an invoice ID; callers hold st.mu.
func (st *invcStore) invoice(id string, now time.Time) (invcInvoice, bool) {
	m := invcInvoiceRe.FindStringSubmatch(id)
	if m == nil {
		return invcInvoice{}, false
	}
	year, _ := strconv.Atoi(m[2])
	month, _ := strconv.Atoi(m[3])
	if year < 2005 || month < 1 || month > 12 || invcIssued(year, month).After(now) {
		return invcInvoice{}, false
	}
	inv := invcInvoice{id: id, account: m[1], year: year, month: month, billSource: []string{m[1]}}
	if m[4] == "" {
		return inv, true
	}
	u := st.units["arn:aws:invoicing::"+invcAccount+":invoice-unit/"+m[4]]
	if u == nil || u.receiver != m[1] {
		return invcInvoice{}, false
	}
	inv.unit = u
	inv.billSource = u.current().linked
	return inv, true
}

// invoicesFor lists an account's invoices for the given periods; callers hold st.mu.
func (st *invcStore) invoicesFor(account string, periods [][2]int) []invcInvoice {
	var units []*invcUnit
	for _, u := range st.units {
		if u.receiver == account {
			units = append(units, u)
		}
	}
	sort.Slice(units, func(i, j int) bool { return units[i].arn < units[j].arn })
	var out []invcInvoice
	for _, p := range periods {
		base := fmt.Sprintf("INV-%s-%04d%02d", account, p[0], p[1])
		out = append(out, invcInvoice{id: base, account: account, year: p[0], month: p[1], billSource: []string{account}})
		for _, u := range units {
			out = append(out, invcInvoice{id: base + "-" + u.arn[len(u.arn)-8:], account: account, year: p[0], month: p[1], unit: u, billSource: u.current().linked})
		}
	}
	return out
}

// amounts returns the invoice's subtotal and tax in cents.
func (inv invcInvoice) amounts() (int64, int64) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(inv.id))
	sub := 2500 + int64(h.Sum64()%250000)
	tax := sub * 8 / 100
	if inv.unit != nil && inv.unit.current().taxInheritanceOff {
		tax = 0
	}
	return sub, tax
}

func invcMoney(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }

func (inv invcInvoice) summary() map[string]any {
	sub, tax := inv.amounts()
	issued := invcIssued(inv.year, inv.month)
	src := inv.billSource
	shown := src[:min(len(src), 100)]
	amount := map[string]any{
		"TotalAmount": invcMoney(sub + tax), "TotalAmountBeforeTax": invcMoney(sub), "CurrencyCode": "USD",
		"AmountBreakdown": map[string]any{
			"SubTotalAmount": invcMoney(sub),
			"Discounts":      map[string]any{"TotalAmount": "0.00"},
			"Taxes":          map[string]any{"TotalAmount": invcMoney(tax)},
			"Fees":           map[string]any{"TotalAmount": "0.00"},
		},
	}
	return map[string]any{
		"AccountId": inv.account, "InvoiceId": inv.id, "IssuedDate": c9Epoch(issued), "DueDate": c9Epoch(issued.AddDate(0, 0, 30)),
		"BillSourceAccounts": c9AnyStrings(shown), "BillSourceAccountsTotalCount": float64(len(src)), "ReceiverRole": "BUYER",
		"Entity":           map[string]any{"InvoicingEntity": invcEntity, "BillingEntity": "AWS"},
		"BillingPeriod":    map[string]any{"Month": float64(inv.month), "Year": float64(inv.year)},
		"InvoiceFrequency": "RECURRING", "BillType": "ANNIVERSARY", "InvoiceType": "INVOICE",
		"BaseCurrencyAmount": amount, "PaymentCurrencyAmount": amount,
	}
}

// invcPeriods lists the issued billing periods a summaries filter selects, newest first.
func invcPeriods(filter map[string]any, now time.Time) ([][2]int, *c9Err) {
	if bp := c9Map(filter, "BillingPeriod"); bp != nil {
		y, m := c9Int(bp, "Year", 0), c9Int(bp, "Month", 0)
		if invcIssued(y, m).After(now) {
			return nil, nil
		}
		return [][2]int{{y, m}}, nil
	}
	newest := now.AddDate(0, -1, 0)
	if invcIssued(newest.Year(), int(newest.Month())).After(now) {
		newest = newest.AddDate(0, -1, 0)
	}
	from, to, count := time.Time{}, now, invcDefaultPeriod
	if ti := c9Map(filter, "TimeInterval"); ti != nil {
		start, _, err := c9Time(ti, "StartDate")
		if err != nil {
			return nil, err
		}
		end, _, err := c9Time(ti, "EndDate")
		if err != nil {
			return nil, err
		}
		if end.Before(start) || end.Sub(start) > 10*366*24*time.Hour {
			return nil, invcInvalid("invalidInput", "TimeInterval must end after it starts and span at most ten years.")
		}
		from, to, count = start, end, invcMaxPeriods
	}
	var out [][2]int
	for t := time.Date(newest.Year(), newest.Month(), 1, 0, 0, 0, 0, time.UTC); len(out) < count && t.Year() >= 2005; t = t.AddDate(0, -1, 0) {
		issued := invcIssued(t.Year(), int(t.Month()))
		if issued.Before(from) {
			break
		}
		if !issued.After(to) {
			out = append(out, [2]int{t.Year(), int(t.Month())})
		}
	}
	return out, nil
}

func (s *Server) invcListInvoiceSummaries(req c9Request) (map[string]any, *c9Err) {
	in := req.in
	sel, filter := c9Map(in, "Selector"), c9Map(in, "Filter")
	value := c9Str(sel, "Value")
	now := time.Now().UTC()
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	var invoices []invcInvoice
	if c9Str(sel, "ResourceType") == "INVOICE_ID" {
		inv, ok := st.invoice(value, now)
		if !ok {
			return nil, invcNotFound("Invoice %s doesn't exist.", value)
		}
		invoices = []invcInvoice{inv}
	} else {
		if !invcAccountRe.MatchString(value) {
			return nil, invcInvalid("invalidInput", "Selector.Value must be a 12-digit account ID.")
		}
		periods, err := invcPeriods(filter, now)
		if err != nil {
			return nil, err
		}
		invoices = st.invoicesFor(value, periods)
	}
	if e, ok := filter["InvoicingEntity"].(string); ok && e != invcEntity {
		invoices = nil
	}
	if role := c9Str(filter, "ReceiverRole"); role != "" && role != "BUYER" {
		invoices = nil
	}
	start, end, next, perr := c9Page(c9Str(in, "NextToken"), len(invoices), c9Int(in, "MaxResults", 100), "ValidationException")
	if perr != nil {
		perr.reason = "invalidNextToken"
		return nil, perr
	}
	out := make([]any, 0, end-start)
	for _, inv := range invoices[start:end] {
		out = append(out, inv.summary())
	}
	return map[string]any{"InvoiceSummaries": out, "NextToken": next}, nil
}

func (st *invcStore) pdfSignature(id string, expires int64) string {
	mac := hmac.New(sha256.New, st.pdfKey)
	fmt.Fprintf(mac, "%s:%d", id, expires)
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) invcGetInvoicePDF(req c9Request) (map[string]any, *c9Err) {
	id := c9Str(req.in, "InvoiceId")
	now := time.Now().UTC()
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.invoice(id, now); !ok {
		return nil, invcNotFound("Invoice %s doesn't exist.", id)
	}
	expires := now.Add(invcPDFLifetime).Truncate(time.Second)
	url := fmt.Sprintf("%s%s%s.pdf?expires=%d&signature=%s", req.base, invcPDFPath, id, expires.Unix(), st.pdfSignature(id, expires.Unix()))
	return map[string]any{"InvoicePDF": map[string]any{
		"InvoiceId": id, "DocumentUrl": url, "DocumentUrlExpirationDate": c9Epoch(expires), "SupplementalDocuments": []any{},
	}}, nil
}

// handleInvoicingPDF serves the document behind a GetInvoicePDF URL.
func (s *Server) handleInvoicingPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := strings.CutSuffix(r.PathValue("file"), ".pdf")
	expires, err := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
	now := time.Now().UTC()
	st := s.invoicing
	st.mu.Lock()
	var inv invcInvoice
	if ok && err == nil && now.Unix() < expires {
		want := st.pdfSignature(id, expires)
		if hmac.Equal([]byte(want), []byte(r.URL.Query().Get("signature"))) {
			inv, ok = st.invoice(id, now)
		} else {
			ok = false
		}
	} else {
		ok = false
	}
	var doc []byte
	if ok {
		doc = inv.pdf() // rendered under the lock: it reads the unit's revision
	}
	st.mu.Unlock()
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("the invoice URL is not valid or has expired\n"))
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(doc)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}

// pdf renders a one-page PDF. Every string in it is built from the invoice
// ID's digits and fixed text, so nothing needs PDF string escaping.
func (inv invcInvoice) pdf() []byte {
	sub, tax := inv.amounts()
	lines := []string{
		"AWS Invoice " + inv.id, invcEntity,
		fmt.Sprintf("Account %s - billing period %04d-%02d", inv.account, inv.year, inv.month),
		"Issued " + invcIssued(inv.year, inv.month).Format("2006-01-02"),
		fmt.Sprintf("Bill source accounts: %d", len(inv.billSource)),
		"Subtotal USD " + invcMoney(sub), "Tax USD " + invcMoney(tax), "Total USD " + invcMoney(sub+tax),
	}
	var content strings.Builder
	content.WriteString("BT /F1 12 Tf 72 720 Td 16 TL\n")
	for _, l := range lines {
		content.WriteString("(" + l + ") Tj T*\n")
	}
	content.WriteString("ET\n")
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", content.Len(), content.String()),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, o := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes()
}

// ---- invoice profiles ----

// orgAccounts returns the payer and every account an invoice unit names
// (at most invcMaxAccountIDs); callers hold st.mu.
func (st *invcStore) orgAccounts() map[string]bool {
	out := map[string]bool{invcAccount: true}
	for _, u := range st.units {
		out[u.receiver] = true
		rev := u.current()
		for _, a := range rev.linked {
			out[a] = true
		}
		for _, a := range rev.billSource {
			out[a] = true
		}
	}
	return out
}

func (s *Server) invcBatchGetInvoiceProfile(req c9Request) (map[string]any, *c9Err) {
	ids := invcDedupe(c9Strings(req.in, "AccountIds"))
	st := s.invoicing
	st.mu.Lock()
	defer st.mu.Unlock()
	known := st.orgAccounts()
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if !known[id] {
			return nil, invcNotFound("Account %s isn't in this organization.", id)
		}
		out = append(out, map[string]any{
			"AccountId": id, "ReceiverName": "Stackyard-" + id, "ReceiverEmail": "billing+" + id + "@example.com", "Issuer": invcEntity,
			"ReceiverAddress": map[string]any{
				"AddressLine1": "410 Terry Ave N", "City": "Seattle", "StateOrRegion": "WA", "PostalCode": "98109",
				"CountryCode": "US", "CompanyName": "Stackyard account " + id,
			},
		})
	}
	return map[string]any{"Profiles": out}, nil
}
