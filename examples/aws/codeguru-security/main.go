// Command codegurusecurity uploads a small Python project to Amazon CodeGuru
// Security on Stackyard, scans it, prints the findings, fixes one issue and
// rescans to show it closed, then reads the account's finding metrics.
//
// The SDK has no CodeGuru Security module in this repository's go.mod, so
// requests are signed by hand.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const source = `import hashlib
import subprocess

DB_PASSWORD = "s3cr3t-pass"

def checksum(path):
    return hashlib.md5(open(path, "rb").read()).hexdigest()

def run(cmd):
    return subprocess.run(cmd, shell=True)
`

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
	fmt.Printf("Stackyard CodeGuru Security client using %s\n", c.endpoint)
	scan := fmt.Sprintf("payments-%d", time.Now().UnixNano()%1_000_000_000)

	c.scan(scan, source)
	findings := c.call(http.MethodGet, "/findings/"+scan, nil)["findings"].([]any)
	for _, f := range findings {
		fm := f.(map[string]any)
		fp := fm["vulnerability"].(map[string]any)["filePath"].(map[string]any)
		fmt.Printf("%-8s %s (%s line %v)\n", fm["severity"], fm["title"], fp["path"], fp["startLine"])
	}

	fixed := strings.Replace(source, `DB_PASSWORD = "s3cr3t-pass"`, `DB_PASSWORD = os.environ["DB_PASSWORD"]`, 1)
	c.scan(scan, fixed)
	closed := c.call(http.MethodGet, "/findings/"+scan+"?status=Closed", nil)["findings"].([]any)
	fmt.Printf("after the fix: %d finding(s) closed\n", len(closed))

	today := time.Now().UTC().Format("2006-01-02")
	summary := c.call(http.MethodGet, "/metrics/summary?date="+today, nil)["metricsSummary"].(map[string]any)
	fmt.Printf("open findings today: %v\n", summary["openFindings"])
	arn := url.PathEscape("arn:aws:codeguru-security:" + c.region + ":123456789012:scans/" + scan)
	c.call(http.MethodPost, "/tags/"+arn, map[string]any{"tags": map[string]any{"team": "payments"}})
	fmt.Printf("tags: %v\n", c.call(http.MethodGet, "/tags/"+arn, nil)["tags"])
}

// scan uploads one file as a zip and runs a scan over it to completion.
func (c client) scan(name, code string) {
	up := c.call(http.MethodPost, "/uploadUrl", map[string]any{"scanName": name})
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("billing/app.py")
	_, _ = w.Write([]byte(code))
	if err := zw.Close(); err != nil {
		exitf("zip: %v", err)
	}
	req, _ := http.NewRequest(http.MethodPut, up["s3Url"].(string), bytes.NewReader(buf.Bytes()))
	for k, v := range up["requestHeaders"].(map[string]any) {
		req.Header.Set(k, fmt.Sprint(v))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		exitf("upload: %v %v", err, resp)
	}
	resp.Body.Close()
	c.call(http.MethodPost, "/scans", map[string]any{
		"scanName": name, "analysisType": "Security",
		"resourceId": map[string]any{"codeArtifactId": up["codeArtifactId"]},
	})
	for i := 0; i < 40; i++ {
		state := c.call(http.MethodGet, "/scans/"+name, nil)["scanState"]
		if state == "Successful" {
			fmt.Printf("scan %s finished\n", name)
			return
		}
		if state == "Failed" {
			exitf("scan %s failed", name)
		}
		time.Sleep(250 * time.Millisecond)
	}
	exitf("scan %s never finished", name)
}

func (c client) call(method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s %s: %v", method, path, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, "codeguru-security", c.region, time.Now()); err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		exitf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s %s: decode: %v", method, path, err)
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
