package server

import (
	"sort"
	"strings"
	"time"
)

// Expense, identity-document and lending analysis over the same text layout.
// ponytail: field types come from label keywords on "Key: Value" lines and
// table headers; real Textract uses trained models.

var txExpenseLabels = []struct{ typ, words string }{
	{"INVOICE_RECEIPT_DATE", "date|invoice date|receipt date|transaction date"},
	{"DUE_DATE", "due date|payment due"},
	{"INVOICE_RECEIPT_ID", "invoice|invoice #|invoice no|invoice number|receipt|receipt #|receipt no|receipt number"},
	{"VENDOR_NAME", "vendor|merchant|seller|from|store|company"},
	{"RECEIVER_NAME", "bill to|customer|sold to|to"},
	{"SUBTOTAL", "subtotal|sub total"},
	{"TAX", "tax|vat|sales tax|gst"},
	{"TOTAL", "total|amount due|balance due|grand total|total due"},
	{"PO_NUMBER", "po|po #|po number|purchase order"},
	{"PAYMENT_TERMS", "terms|payment terms"},
}

var txLineItemLabels = []struct{ typ, words string }{
	{"ITEM", "item|description|product|service|details"},
	{"QUANTITY", "qty|quantity|units"},
	{"UNIT_PRICE", "unit price|rate|price each|unit cost"},
	{"PRICE", "price|amount|total|line total|cost"},
	{"PRODUCT_CODE", "sku|code|product code|item code"},
}

func txLabelType(labels []struct{ typ, words string }, key string) string {
	k := strings.Join(strings.Fields(strings.ToLower(strings.Trim(key, " :#."))), " ")
	for _, l := range labels {
		for _, w := range strings.Split(l.words, "|") {
			if k == w {
				return l.typ
			}
		}
	}
	return ""
}

var txCurrencies = map[string]string{"$": "USD", "€": "EUR", "£": "GBP", "¥": "JPY", "₹": "INR"}

func txCurrency(value string) map[string]any {
	v := strings.TrimSpace(value)
	for sym, code := range txCurrencies {
		if strings.HasPrefix(v, sym) {
			return map[string]any{"Code": code, "Confidence": txKVConfidence}
		}
	}
	for _, code := range []string{"USD", "EUR", "GBP", "CAD", "INR", "JPY", "CHF", "AUD", "CNY", "SEK", "HKD"} {
		if strings.HasPrefix(v, code+" ") || strings.HasSuffix(v, " "+code) {
			return map[string]any{"Code": code, "Confidence": txKVConfidence}
		}
	}
	return nil
}

func txDetection(text string, box txBox) map[string]any {
	return map[string]any{"Text": text, "Geometry": box.geometry(), "Confidence": txKVConfidence}
}

// expenseDocument builds one ExpenseDocument over the given pages.
func (a *txAnalysis) expenseDocument(index int, pages []*txPageInfo) map[string]any {
	var fields []any
	haveVendor := false
	for _, p := range pages {
		for _, l := range p.lines {
			if l.kv == nil {
				continue
			}
			typ := txLabelType(txExpenseLabels, l.kv.key)
			haveVendor = haveVendor || typ == "VENDOR_NAME"
			if typ == "" {
				typ = "OTHER"
			}
			f := map[string]any{
				"Type":           map[string]any{"Text": typ, "Confidence": txKVConfidence},
				"LabelDetection": txDetection(l.kv.key, p.spanBox(l, 0, l.kv.keyEnd)),
				"ValueDetection": txDetection(l.kv.value, p.spanBox(l, l.kv.valueStart, l.runes)),
				"PageNumber":     p.number,
			}
			if c := txCurrency(l.kv.value); c != nil {
				f["Currency"] = c
			}
			fields = append(fields, f)
		}
	}
	// Receipts lead with the merchant: an unlabelled first line names the vendor.
	if !haveVendor && len(pages) > 0 {
		for _, l := range pages[0].lines {
			if l.kv == nil && !l.inTable {
				fields = append([]any{map[string]any{
					"Type":           map[string]any{"Text": "VENDOR_NAME", "Confidence": txKVConfidence},
					"ValueDetection": txDetection(l.text, pages[0].lineBox(l)),
					"PageNumber":     pages[0].number,
				}}, fields...)
				break
			}
		}
	}
	var groups []any
	for _, p := range pages {
		for _, t := range p.tables {
			groups = append(groups, a.lineItemGroup(len(groups)+1, p, t))
		}
	}
	var blocks []any
	for _, p := range pages {
		page, rest, lineIDs := a.pageBlocks(p)
		if len(lineIDs) > 0 {
			page["Relationships"] = []any{txRel("CHILD", lineIDs)}
		}
		blocks = append(append(blocks, page), rest...)
	}
	return map[string]any{"ExpenseIndex": index, "SummaryFields": txNonNil(fields), "LineItemGroups": txNonNil(groups), "Blocks": txNonNil(blocks)}
}

func (a *txAnalysis) lineItemGroup(index int, p *txPageInfo, t *txTable) map[string]any {
	var rows []*txLine
	for _, l := range t.lines {
		if l.cells != nil {
			rows = append(rows, l)
		}
	}
	var header []string
	body := rows
	if len(rows) > 1 {
		for _, c := range rows[0].cells {
			header = append(header, c.text)
		}
		body = rows[1:]
	}
	var items []any
	for _, row := range body {
		var fields []any
		for ci, c := range row.cells {
			typ, label := "OTHER", ""
			if ci < len(header) {
				label = header[ci]
				if lt := txLabelType(txLineItemLabels, label); lt != "" {
					typ = lt
				}
			}
			f := map[string]any{
				"Type":           map[string]any{"Text": typ, "Confidence": txKVConfidence},
				"ValueDetection": txDetection(c.text, p.spanBox(row, c.start, c.end)),
				"PageNumber":     p.number,
			}
			if label != "" {
				f["LabelDetection"] = map[string]any{"Text": label, "Confidence": txKVConfidence}
			}
			if cur := txCurrency(c.text); cur != nil {
				f["Currency"] = cur
			}
			fields = append(fields, f)
		}
		fields = append(fields, map[string]any{
			"Type":           map[string]any{"Text": "EXPENSE_ROW", "Confidence": txKVConfidence},
			"ValueDetection": txDetection(row.text, p.lineBox(row)),
			"PageNumber":     p.number,
		})
		items = append(items, map[string]any{"LineItemExpenseFields": fields})
	}
	return map[string]any{"LineItemGroupIndex": index, "LineItems": txNonNil(items)}
}

// AnalyzeID returns every field type, empty when the document doesn't show it.
var txIDFields = []struct{ typ, words string }{
	{"FIRST_NAME", "first name|given name|given names|fn"},
	{"LAST_NAME", "last name|surname|family name|ln"},
	{"MIDDLE_NAME", "middle name|middle"},
	{"SUFFIX", "suffix"},
	{"CITY_IN_ADDRESS", "city"},
	{"ZIP_CODE_IN_ADDRESS", "zip|zip code|postal code|postcode"},
	{"STATE_IN_ADDRESS", "state"},
	{"STATE_NAME", "issuing state|state name"},
	{"DOCUMENT_NUMBER", "document number|dl|dl no|license number|licence number|passport number|passport no|id number|id no"},
	{"EXPIRATION_DATE", "expiration date|expiration|expires|exp|expiry date|date of expiry"},
	{"DATE_OF_BIRTH", "date of birth|dob|birth date"},
	{"DATE_OF_ISSUE", "date of issue|issue date|issued|iss"},
	{"ID_TYPE", "id type|document type"},
	{"ENDORSEMENTS", "endorsements|end"},
	{"VETERAN", "veteran"},
	{"RESTRICTIONS", "restrictions|rest"},
	{"CLASS", "class"},
	{"ADDRESS", "address|addr"},
	{"COUNTY", "county"},
	{"PLACE_OF_BIRTH", "place of birth|birthplace"},
	{"MRZ_CODE", "mrz|mrz code"},
}

var txDateLayouts = []string{"2006-01-02", "01/02/2006", "1/2/2006", "01-02-2006", "02 Jan 2006", "Jan 2, 2006", "January 2, 2006", "2 January 2006"}

func (a *txAnalysis) identityDocument(index int, p *txPageInfo) map[string]any {
	found := map[string]string{}
	for _, l := range p.lines {
		if l.kv != nil {
			if typ := txLabelType(txIDFields, l.kv.key); typ != "" && found[typ] == "" {
				found[typ] = l.kv.value
			}
		}
	}
	if found["ID_TYPE"] == "" {
		text := strings.ToLower(strings.Join(func() []string {
			var s []string
			for _, l := range p.lines {
				s = append(s, l.text)
			}
			return s
		}(), " "))
		switch {
		case strings.Contains(text, "passport"):
			found["ID_TYPE"] = "PASSPORT"
		case strings.Contains(text, "driver"), strings.Contains(text, "licence"), strings.Contains(text, "license"):
			found["ID_TYPE"] = "DRIVER LICENSE FRONT"
		default:
			found["ID_TYPE"] = "OTHER"
		}
	}
	var fields []any
	for _, f := range txIDFields {
		value := map[string]any{"Text": found[f.typ], "Confidence": txKVConfidence}
		if strings.Contains(f.typ, "DATE") && found[f.typ] != "" {
			for _, layout := range txDateLayouts {
				if t, err := time.Parse(layout, found[f.typ]); err == nil {
					value["NormalizedValue"] = map[string]any{"Value": t.Format("2006-01-02T15:04:05"), "ValueType": "DATE"}
					break
				}
			}
		}
		fields = append(fields, map[string]any{
			"Type":           map[string]any{"Text": f.typ, "Confidence": txKVConfidence},
			"ValueDetection": value,
		})
	}
	page, rest, lineIDs := a.pageBlocks(p)
	if len(lineIDs) > 0 {
		page["Relationships"] = []any{txRel("CHILD", lineIDs)}
	}
	return map[string]any{"DocumentIndex": index, "IdentityDocumentFields": fields, "Blocks": append([]any{page}, rest...)}
}

// Lending page classes, matched by phrases on the page.
var txLendingClasses = []struct{ typ, phrases string }{
	{"W2", "w-2|wage and tax statement"},
	{"1040", "form 1040|u.s. individual income tax return"},
	{"PAYSLIPS", "pay stub|payslip|earnings statement|pay statement"},
	{"BANK_STATEMENTS", "bank statement|statement period|account statement"},
	{"CHECKS", "pay to the order of"},
	{"UTILITY_BILLS", "utility bill|electric bill|water bill"},
	{"INVOICES", "invoice"},
	{"RECEIPTS", "receipt"},
	{"IDENTITY_DOCUMENT", "driver license|driver's license|passport|identification card"},
}

func (p *txPageInfo) lendingClass() string {
	var sb strings.Builder
	for _, l := range p.lines {
		sb.WriteString(strings.ToLower(l.text))
		sb.WriteByte('\n')
	}
	text := sb.String()
	for _, c := range txLendingClasses {
		for _, ph := range strings.Split(c.phrases, "|") {
			if strings.Contains(text, ph) {
				return c.typ
			}
		}
	}
	return "UNCLASSIFIED"
}

func (p *txPageInfo) printedPageNumber() string {
	if n := len(p.lines); n > 0 && txPageNumber.MatchString(p.lines[n-1].text) {
		fields := strings.Fields(p.lines[n-1].text)
		for _, f := range fields {
			if f != "" && strings.Trim(f, "0123456789") == "" {
				return f
			}
		}
	}
	return "undetected"
}

// lendingResults returns one LendingResult per page.
func (a *txAnalysis) lendingResults() []any {
	var out []any
	for _, p := range a.pages {
		class := p.lendingClass()
		var extraction map[string]any
		switch class {
		case "INVOICES", "RECEIPTS":
			extraction = map[string]any{"ExpenseDocument": a.expenseDocument(1, []*txPageInfo{p})}
		case "IDENTITY_DOCUMENT":
			extraction = map[string]any{"IdentityDocument": a.identityDocument(1, p)}
		default:
			var fields, sigs []any
			for _, l := range p.lines {
				if l.kv != nil {
					fields = append(fields, map[string]any{
						"Type":            strings.ToUpper(strings.Join(strings.Fields(l.kv.key), "_")),
						"KeyDetection":    txDetection(l.kv.key, p.spanBox(l, 0, l.kv.keyEnd)),
						"ValueDetections": []any{txDetection(l.kv.value, p.spanBox(l, l.kv.valueStart, l.runes))},
					})
				}
				if box, ok := txSignature(p, l); ok {
					sigs = append(sigs, map[string]any{"Confidence": txKVConfidence, "Geometry": box.geometry()})
				}
			}
			extraction = map[string]any{"LendingDocument": map[string]any{"LendingFields": txNonNil(fields), "SignatureDetections": txNonNil(sigs)}}
		}
		out = append(out, map[string]any{
			"Page": p.number,
			"PageClassification": map[string]any{
				"PageType":   []any{map[string]any{"Value": class, "Confidence": txKVConfidence}},
				"PageNumber": []any{map[string]any{"Value": p.printedPageNumber(), "Confidence": txKVConfidence}},
			},
			"Extractions": []any{extraction},
		})
	}
	return out
}

// lendingSummary groups consecutive pages of one class into split documents.
func (a *txAnalysis) lendingSummary() map[string]any {
	type group struct {
		splits [][]int
		sigs   []int
	}
	groups := map[string]*group{}
	var order []string
	prev := ""
	for _, p := range a.pages {
		class := p.lendingClass()
		g := groups[class]
		if g == nil {
			g = &group{}
			groups[class] = g
			order = append(order, class)
		}
		if class == prev {
			g.splits[len(g.splits)-1] = append(g.splits[len(g.splits)-1], p.number)
		} else {
			g.splits = append(g.splits, []int{p.number})
		}
		prev = class
		for _, l := range p.lines {
			if _, ok := txSignature(p, l); ok {
				g.sigs = append(g.sigs, p.number)
				break
			}
		}
	}
	var docGroups []any
	for _, class := range order {
		g := groups[class]
		var splits, sigs []any
		for i, pages := range g.splits {
			ps := make([]any, len(pages))
			for j, n := range pages {
				ps[j] = n
			}
			splits = append(splits, map[string]any{"Index": i + 1, "Pages": ps})
		}
		for _, n := range g.sigs {
			sigs = append(sigs, map[string]any{"Page": n})
		}
		docGroups = append(docGroups, map[string]any{"Type": class, "SplitDocuments": splits, "DetectedSignatures": txNonNil(sigs), "UndetectedSignatures": []any{}})
	}
	var undetected []any
	for _, c := range txLendingClasses {
		if groups[c.typ] == nil {
			undetected = append(undetected, c.typ)
		}
	}
	sort.Slice(undetected, func(i, j int) bool { return undetected[i].(string) < undetected[j].(string) })
	return map[string]any{"DocumentGroups": txNonNil(docGroups), "UndetectedDocumentTypes": txNonNil(undetected)}
}

func txNonNil(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
