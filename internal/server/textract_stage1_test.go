package server

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
)

const txInvoice = "Acme Supplies\nInvoice Number: INV-42\nDate: 2026-01-31\nTotal: $123.45\nItem\tQty\tPrice\nWidget\t2\t$10.00\nGadget\t1\t$103.45\n/s/ Jane Doe\n"

func txBlocksOf(out map[string]any) []map[string]any {
	raw, _ := out["Blocks"].([]any)
	var blocks []map[string]any
	for _, b := range raw {
		blocks = append(blocks, b.(map[string]any))
	}
	return blocks
}

func txByType(blocks []map[string]any, typ string) []map[string]any {
	var out []map[string]any
	for _, b := range blocks {
		if b["BlockType"] == typ {
			out = append(out, b)
		}
	}
	return out
}

func TestTextractStage1DetectDocumentText(t *testing.T) {
	_, ts := newTextractTestServer(t)
	out := txOK(t, ts, "DetectDocumentText", `{"Document":`+txBytesDoc("Hello world\nSecond line here\n")+`}`)
	blocks := txBlocksOf(out)
	if len(txByType(blocks, "PAGE")) != 1 || len(txByType(blocks, "LINE")) != 2 || len(txByType(blocks, "WORD")) != 5 {
		t.Fatalf("blocks: %v", blocks)
	}
	if txByType(blocks, "LINE")[1]["Text"] != "Second line here" {
		t.Fatalf("second line: %v", txByType(blocks, "LINE")[1])
	}
	// The PAGE block's CHILD relationship lists both LINE ids, and each LINE
	// lists its WORD ids.
	pageRel := blocks[0]["Relationships"].([]any)[0].(map[string]any)
	lineRel := txByType(blocks, "LINE")[1]["Relationships"].([]any)[0].(map[string]any)
	if ids, _ := pageRel["Ids"].([]any); len(ids) != 2 || ids[1] != txByType(blocks, "LINE")[1]["Id"] {
		t.Fatalf("page relationship: %v", pageRel)
	}
	if ids, _ := lineRel["Ids"].([]any); len(ids) != 3 {
		t.Fatalf("line relationship: %v", lineRel)
	}
	// Block ids are stable for the same document.
	again := txBlocksOf(txOK(t, ts, "DetectDocumentText", `{"Document":`+txBytesDoc("Hello world\nSecond line here\n")+`}`))
	if again[0]["Id"] != blocks[0]["Id"] {
		t.Fatalf("ids changed: %v vs %v", again[0]["Id"], blocks[0]["Id"])
	}
	// Images have no extractable text here: a PAGE block only.
	png := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nrest-of-image"))
	img := txBlocksOf(txOK(t, ts, "DetectDocumentText", `{"Document":{"Bytes":"`+png+`"}}`))
	if len(img) != 1 || img[0]["BlockType"] != "PAGE" {
		t.Fatalf("image blocks: %v", img)
	}
	// Multi-page documents need the async API.
	if code := txErrCode(t, ts, "DetectDocumentText", `{"Document":`+txBytesDoc("page one\fpage two")+`}`); code != "UnsupportedDocumentException" {
		t.Fatalf("multi-page sync: %s", code)
	}
	binary := base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0xfe})
	if code := txErrCode(t, ts, "DetectDocumentText", `{"Document":{"Bytes":"`+binary+`"}}`); code != "UnsupportedDocumentException" {
		t.Fatalf("binary: %s", code)
	}
	if code := txErrCode(t, ts, "DetectDocumentText", `{"Document":{"S3Object":{"Bucket":"nope","Name":"x"}}}`); code != "InvalidS3ObjectException" {
		t.Fatalf("missing S3 object: %s", code)
	}
}

func TestTextractStage1PDFText(t *testing.T) {
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	_, _ = w.Write([]byte("BT /F1 12 Tf 72 712 Td (Total: 9.99) Tj 0 -14 Td [(Hello) -300 (PDF)] TJ ET"))
	_ = w.Close()
	pdf := fmt.Sprintf("%%PDF-1.4\n1 0 obj << /Type /Page >> endobj\n2 0 obj << /Length %d /Filter /FlateDecode >>\nstream\n%sendstream\nendobj\n", z.Len(), z.String())
	doc, err := txParseDocument([]byte(pdf))
	if err != nil {
		t.Fatal(err.msg)
	}
	if len(doc.pages) != 1 || len(doc.pages[0]) != 2 || doc.pages[0][0] != "Total: 9.99" || doc.pages[0][1] != "Hello PDF" {
		t.Fatalf("pdf lines: %q", doc.pages)
	}
}

func TestTextractStage1AnalyzeDocument(t *testing.T) {
	_, ts := newTextractTestServer(t)
	body := `{"Document":` + txBytesDoc(txInvoice) + `,"FeatureTypes":["FORMS","TABLES","QUERIES","SIGNATURES","LAYOUT"],` +
		`"QueriesConfig":{"Queries":[{"Text":"What is the total?","Alias":"TOTAL"},{"Text":"What is the shipping method?"}]}}`
	blocks := txBlocksOf(txOK(t, ts, "AnalyzeDocument", body))
	kv := txByType(blocks, "KEY_VALUE_SET")
	if len(kv) != 6 { // three Key: Value lines, a KEY and a VALUE block each
		t.Fatalf("key/value blocks: %d", len(kv))
	}
	cells := txByType(blocks, "CELL")
	if len(txByType(blocks, "TABLE")) != 1 || len(cells) != 9 {
		t.Fatalf("table: %d tables, %d cells", len(txByType(blocks, "TABLE")), len(cells))
	}
	if cells[0]["EntityTypes"].([]any)[0] != "COLUMN_HEADER" || cells[8]["RowIndex"] != float64(3) || cells[8]["ColumnIndex"] != float64(3) {
		t.Fatalf("cells: %v / %v", cells[0], cells[8])
	}
	if len(txByType(blocks, "SIGNATURE")) != 1 || len(txByType(blocks, "LAYOUT_TABLE")) != 1 || len(txByType(blocks, "LAYOUT_TITLE")) != 1 {
		t.Fatalf("signature/layout blocks missing")
	}
	queries, results := txByType(blocks, "QUERY"), txByType(blocks, "QUERY_RESULT")
	if len(queries) != 2 || len(results) != 1 || results[0]["Text"] != "$123.45" {
		t.Fatalf("queries: %v / %v", queries, results)
	}
	if queries[1]["Relationships"] != nil {
		t.Fatalf("an unanswerable query has no answer: %v", queries[1])
	}
	if code := txErrCode(t, ts, "AnalyzeDocument", `{"Document":`+txBytesDoc("x")+`,"FeatureTypes":["QUERIES"]}`); code != "InvalidParameterException" {
		t.Fatalf("QUERIES without QueriesConfig: %s", code)
	}
	if code := txErrCode(t, ts, "AnalyzeDocument", `{"Document":`+txBytesDoc("x")+`,"FeatureTypes":["FORMS"],"AdaptersConfig":{"Adapters":[{"AdapterId":"aaaaaaaaaaaa","Version":"1"}]}}`); code != "InvalidParameterException" {
		t.Fatalf("unknown adapter: %s", code)
	}
}

func TestTextractStage1ExpenseAndID(t *testing.T) {
	_, ts := newTextractTestServer(t)
	out := txOK(t, ts, "AnalyzeExpense", `{"Document":`+txBytesDoc(txInvoice)+`}`)
	doc := out["ExpenseDocuments"].([]any)[0].(map[string]any)
	types := map[string]string{}
	for _, f := range doc["SummaryFields"].([]any) {
		fm := f.(map[string]any)
		types[fm["Type"].(map[string]any)["Text"].(string)] = fm["ValueDetection"].(map[string]any)["Text"].(string)
	}
	if types["TOTAL"] != "$123.45" || types["INVOICE_RECEIPT_ID"] != "INV-42" || types["VENDOR_NAME"] != "Acme Supplies" {
		t.Fatalf("summary fields: %v", types)
	}
	items := doc["LineItemGroups"].([]any)[0].(map[string]any)["LineItems"].([]any)
	if len(items) != 2 {
		t.Fatalf("line items: %v", items)
	}
	id := txOK(t, ts, "AnalyzeID", `{"DocumentPages":[`+txBytesDoc("DRIVER LICENSE\nFirst Name: Ada\nLast Name: Lovelace\nDate of Birth: 1815-12-10\n")+`]}`)
	fields := id["IdentityDocuments"].([]any)[0].(map[string]any)["IdentityDocumentFields"].([]any)
	got := map[string]map[string]any{}
	for _, f := range fields {
		fm := f.(map[string]any)
		got[fm["Type"].(map[string]any)["Text"].(string)] = fm["ValueDetection"].(map[string]any)
	}
	if got["FIRST_NAME"]["Text"] != "Ada" || got["ID_TYPE"]["Text"] != "DRIVER LICENSE FRONT" || got["DATE_OF_BIRTH"]["NormalizedValue"] == nil {
		t.Fatalf("id fields: %v", got)
	}
}

func TestTextractStage1AsyncJobs(t *testing.T) {
	srv, ts := newTextractTestServer(t)
	if _, err := srv.s3.CreateBucket("tx-docs"); err != nil {
		t.Fatal(err)
	}
	doc := "Statement Period: Jan 2026\nBalance: $10\n1\fW-2 Wage and Tax Statement\nWages: 1000\n/s/ Pat\n2\n"
	if _, err := srv.s3.PutObject("tx-docs", "two-pages.txt", "text/plain", []byte(doc), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	topic, err := srv.sns.CreateTopic("tx-done", nil)
	if err != nil {
		t.Fatal(err)
	}
	loc := `"DocumentLocation":{"S3Object":{"Bucket":"tx-docs","Name":"two-pages.txt"}}`
	start := txOK(t, ts, "StartDocumentTextDetection", `{`+loc+`,"ClientRequestToken":"tok1","NotificationChannel":{"SNSTopicArn":"`+topic.ARN+`","RoleArn":"arn:aws:iam::123456789012:role/tx"},"OutputConfig":{"S3Bucket":"tx-docs","S3Prefix":"out"}}`)
	jobID := start["JobId"].(string)
	// Same token, same request: same job; different request: mismatch.
	if again := txOK(t, ts, "StartDocumentTextDetection", `{`+loc+`,"ClientRequestToken":"tok1","NotificationChannel":{"SNSTopicArn":"`+topic.ARN+`","RoleArn":"arn:aws:iam::123456789012:role/tx"},"OutputConfig":{"S3Bucket":"tx-docs","S3Prefix":"out"}}`); again["JobId"] != jobID {
		t.Fatalf("idempotent start: %v", again)
	}
	if code := txErrCode(t, ts, "StartDocumentTextDetection", `{`+loc+`,"ClientRequestToken":"tok1","JobTag":"other"}`); code != "IdempotentParameterMismatchException" {
		t.Fatalf("token reuse: %s", code)
	}
	first := txOK(t, ts, "GetDocumentTextDetection", `{"JobId":"`+jobID+`"}`)
	if first["JobStatus"] != "IN_PROGRESS" || first["Blocks"] != nil {
		t.Fatalf("first read: %v", first)
	}
	page := txOK(t, ts, "GetDocumentTextDetection", `{"JobId":"`+jobID+`","MaxResults":3}`)
	if page["JobStatus"] != "SUCCEEDED" || len(page["Blocks"].([]any)) != 3 || page["NextToken"] == nil {
		t.Fatalf("paged read: %v", page)
	}
	if page["DocumentMetadata"].(map[string]any)["Pages"] != float64(2) {
		t.Fatalf("pages: %v", page["DocumentMetadata"])
	}
	if out, err := srv.s3.GetObject("tx-docs", "out/"+jobID+"/1"); err != nil || !bytes.Contains(out.Body, []byte(`"JobStatus":"SUCCEEDED"`)) {
		t.Fatalf("output config: %v", err)
	}
	// The SNS store keeps no message log; the job records that it notified.
	srv.textract.mu.Lock()
	notified := srv.textract.jobs[jobID].notified
	srv.textract.mu.Unlock()
	if !notified {
		t.Fatal("completion was not published")
	}
	if code := txErrCode(t, ts, "GetDocumentAnalysis", `{"JobId":"`+jobID+`"}`); code != "InvalidJobIdException" {
		t.Fatalf("job from another API: %s", code)
	}
	if code := txErrCode(t, ts, "GetDocumentTextDetection", `{"JobId":"`+jobID+`","NextToken":"bogus"}`); code != "InvalidParameterException" {
		t.Fatalf("bad token: %s", code)
	}
	// Lending: per-page classification and a summary.
	lend := txOK(t, ts, "StartLendingAnalysis", `{`+loc+`}`)["JobId"].(string)
	txOK(t, ts, "GetLendingAnalysis", `{"JobId":"`+lend+`"}`)
	results := txOK(t, ts, "GetLendingAnalysis", `{"JobId":"`+lend+`"}`)["Results"].([]any)
	class := func(i int) any {
		return results[i].(map[string]any)["PageClassification"].(map[string]any)["PageType"].([]any)[0].(map[string]any)["Value"]
	}
	if len(results) != 2 || class(0) != "BANK_STATEMENTS" || class(1) != "W2" {
		t.Fatalf("lending results: %v", results)
	}
	summary := txOK(t, ts, "GetLendingAnalysisSummary", `{"JobId":"`+lend+`"}`)["Summary"].(map[string]any)
	if len(summary["DocumentGroups"].([]any)) != 2 {
		t.Fatalf("summary: %v", summary)
	}
	if code := txErrCode(t, ts, "StartDocumentTextDetection", `{"DocumentLocation":{"S3Object":{"Bucket":"tx-docs","Name":"missing"}}}`); code != "InvalidS3ObjectException" {
		t.Fatalf("missing object: %s", code)
	}
	if code := txErrCode(t, ts, "StartDocumentTextDetection", `{`+loc+`,"NotificationChannel":{"SNSTopicArn":"arn:aws:sns:us-east-1:123456789012:nope","RoleArn":"arn:aws:iam::123456789012:role/tx"}}`); code != "InvalidParameterException" {
		t.Fatalf("missing topic: %s", code)
	}
}

func TestTextractStage1AdaptersAndTags(t *testing.T) {
	srv, ts := newTextractTestServer(t)
	if _, err := srv.s3.CreateBucket("tx-train"); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.s3.PutObject("tx-train", "manifest.json", "application/json", []byte(`{}`), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	id := txOK(t, ts, "CreateAdapter", `{"AdapterName":"invoices","FeatureTypes":["QUERIES"],"ClientRequestToken":"a1","Tags":{"team":"ap"}}`)["AdapterId"].(string)
	if again := txOK(t, ts, "CreateAdapter", `{"AdapterName":"invoices","FeatureTypes":["QUERIES"],"ClientRequestToken":"a1","Tags":{"team":"ap"}}`); again["AdapterId"] != id {
		t.Fatalf("idempotent create: %v", again)
	}
	version := `{"AdapterId":"` + id + `","DatasetConfig":{"ManifestS3Object":{"Bucket":"tx-train","Name":"manifest.json"}},"OutputConfig":{"S3Bucket":"tx-train"}}`
	v := txOK(t, ts, "CreateAdapterVersion", version)["AdapterVersion"].(string)
	if code := txErrCode(t, ts, "CreateAdapterVersion", `{"AdapterId":"`+id+`","DatasetConfig":{"ManifestS3Object":{"Bucket":"tx-train","Name":"missing"}},"OutputConfig":{"S3Bucket":"tx-train"}}`); code != "InvalidS3ObjectException" {
		t.Fatalf("missing manifest: %s", code)
	}
	ref := `{"AdapterId":"` + id + `","AdapterVersion":"` + v + `"}`
	if s := txOK(t, ts, "GetAdapterVersion", ref)["Status"]; s != "CREATION_IN_PROGRESS" {
		t.Fatalf("new version: %v", s)
	}
	if code := txErrCode(t, ts, "DeleteAdapter", `{"AdapterId":"aaaaaaaaaaaa"}`); code != "ResourceNotFoundException" {
		t.Fatalf("delete missing: %s", code)
	}
	if got := txOK(t, ts, "GetAdapterVersion", ref); got["Status"] != "ACTIVE" || got["EvaluationMetrics"] == nil {
		t.Fatalf("settled version: %v", got)
	}
	// An ACTIVE adapter version can be used for analysis.
	txOK(t, ts, "AnalyzeDocument", `{"Document":`+txBytesDoc("Total: 5")+`,"FeatureTypes":["QUERIES"],"QueriesConfig":{"Queries":[{"Text":"total"}]},"AdaptersConfig":{"Adapters":[{"AdapterId":"`+id+`","Version":"`+v+`"}]}}`)
	arn := txAdapterARN(id)
	txOK(t, ts, "TagResource", `{"ResourceARN":"`+arn+`","Tags":{"env":"dev"}}`)
	if tags := txOK(t, ts, "ListTagsForResource", `{"ResourceARN":"`+arn+`"}`)["Tags"].(map[string]any); tags["env"] != "dev" || tags["team"] != "ap" {
		t.Fatalf("tags: %v", tags)
	}
	if code := txErrCode(t, ts, "TagResource", `{"ResourceARN":"`+arn+`","Tags":{"aws:x":"y"}}`); code != "ValidationException" {
		t.Fatalf("aws: tag: %s", code)
	}
	txOK(t, ts, "UntagResource", `{"ResourceARN":"`+arn+`","TagKeys":["env"]}`)
	if updated := txOK(t, ts, "UpdateAdapter", `{"AdapterId":"`+id+`","Description":"AP invoices","AutoUpdate":"ENABLED"}`); updated["AutoUpdate"] != "ENABLED" {
		t.Fatalf("update: %v", updated)
	}
	if list := txOK(t, ts, "ListAdapterVersions", `{"AdapterId":"`+id+`"}`)["AdapterVersions"].([]any); len(list) != 1 {
		t.Fatalf("versions: %v", list)
	}
	txOK(t, ts, "DeleteAdapterVersion", ref)
	txOK(t, ts, "DeleteAdapter", `{"AdapterId":"`+id+`"}`)
	if code := txErrCode(t, ts, "GetAdapter", `{"AdapterId":"`+id+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted adapter: %s", code)
	}
	raw, _ := json.Marshal(txOK(t, ts, "ListAdapters", `{}`))
	if bytes.Contains(raw, []byte(id)) {
		t.Fatalf("deleted adapter still listed: %s", raw)
	}
}

// Stored results share one block budget: a large job evicts the oldest ones.
func TestTextractStage1JobStoreIsBounded(t *testing.T) {
	st := newTextractStore()
	first := st.addJob(&txJob{api: "StartDocumentTextDetection"}, &txResult{cost: txMaxStoredBlocks / 2}, "", "")
	second := st.addJob(&txJob{api: "StartDocumentTextDetection"}, &txResult{cost: txMaxStoredBlocks / 2}, "", "")
	st.addJob(&txJob{api: "StartDocumentTextDetection"}, &txResult{cost: 10}, "", "")
	if st.jobs[first] != nil || st.jobs[second] == nil || st.storedBlocks > txMaxStoredBlocks {
		t.Fatalf("budget: first kept=%v second kept=%v stored=%d", st.jobs[first] != nil, st.jobs[second] != nil, st.storedBlocks)
	}
}
