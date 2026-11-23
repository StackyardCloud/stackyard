// Command textract runs Amazon Textract against Stackyard: detect text in a
// document, analyze its forms, tables and queries, read it as an expense, then
// run an async text-detection job over a two-page document stored in S3.
//
// Stackyard has no OCR engine: it analyzes plain-text and PDF documents (and
// returns only a PAGE block for images). The SDK has no Textract module in
// this repository's go.mod, so requests are signed by hand.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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

const invoice = "Acme Supplies\nInvoice Number: INV-42\nDate: 2026-01-31\nTotal: $123.45\nItem\tQty\tPrice\nWidget\t2\t$10.00\nGadget\t1\t$103.45\n"

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
	fmt.Printf("Stackyard Textract client using %s\n", c.endpoint)
	doc := map[string]any{"Bytes": base64.StdEncoding.EncodeToString([]byte(invoice))}

	text := c.call("DetectDocumentText", map[string]any{"Document": doc})
	fmt.Printf("lines: %v\n", texts(text["Blocks"], "LINE"))

	analysis := c.call("AnalyzeDocument", map[string]any{
		"Document":      doc,
		"FeatureTypes":  []string{"FORMS", "TABLES", "QUERIES"},
		"QueriesConfig": map[string]any{"Queries": []map[string]any{{"Text": "What is the total?", "Alias": "TOTAL"}}},
	})
	fmt.Printf("key/value blocks: %d, table cells: %d, query answers: %v\n",
		count(analysis["Blocks"], "KEY_VALUE_SET"), count(analysis["Blocks"], "CELL"), texts(analysis["Blocks"], "QUERY_RESULT"))

	expense := c.call("AnalyzeExpense", map[string]any{"Document": doc})
	for _, d := range expense["ExpenseDocuments"].([]any) {
		for _, f := range d.(map[string]any)["SummaryFields"].([]any) {
			fm := f.(map[string]any)
			fmt.Printf("expense %v = %v\n", fm["Type"].(map[string]any)["Text"], fm["ValueDetection"].(map[string]any)["Text"])
		}
	}

	// Async: a two-page text document (pages split on form feed) in S3.
	bucket := fmt.Sprintf("stackyard-textract-%d", time.Now().UnixNano()%1_000_000_000)
	c.s3("PUT", "/"+bucket, nil)
	c.s3("PUT", "/"+bucket+"/report.txt", []byte("Quarterly Report\nRevenue: $10M\n1\fAppendix\nRegion: West\n2\n"))
	job := c.call("StartDocumentTextDetection", map[string]any{
		"DocumentLocation": map[string]any{"S3Object": map[string]any{"Bucket": bucket, "Name": "report.txt"}},
		"OutputConfig":     map[string]any{"S3Bucket": bucket, "S3Prefix": "out"},
	})
	jobID := job["JobId"].(string)
	for {
		res := c.call("GetDocumentTextDetection", map[string]any{"JobId": jobID})
		fmt.Printf("job %s… %v\n", jobID[:12], res["JobStatus"])
		if res["JobStatus"] == "SUCCEEDED" {
			fmt.Printf("pages: %v, lines: %v\n", res["DocumentMetadata"].(map[string]any)["Pages"], texts(res["Blocks"], "LINE"))
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	out := c.s3("GET", "/"+bucket+"/out/"+jobID+"/1", nil)
	fmt.Printf("OutputConfig wrote %d bytes of results\n", len(out))
}

func texts(blocks any, typ string) []string {
	var out []string
	list, _ := blocks.([]any)
	for _, b := range list {
		if bm := b.(map[string]any); bm["BlockType"] == typ {
			out = append(out, fmt.Sprint(bm["Text"]))
		}
	}
	return out
}

func count(blocks any, typ string) int { return len(texts(blocks, typ)) }

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, "textract", http.MethodPost, c.endpoint+"/", body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Textract." + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

func (c client) s3(method, path string, body []byte) []byte {
	return c.send(method+" "+path, "s3", method, c.endpoint+path, body, nil)
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, service, method, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
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
