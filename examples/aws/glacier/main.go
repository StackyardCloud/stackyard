// Command glacier walks an S3 Glacier lifecycle against Stackyard: create a
// vault, upload an archive in one request and another in two parts, retrieve
// one through an archive-retrieval job, take an inventory, then clean up.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
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

const partSize = 1 << 20

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
	fmt.Printf("Stackyard S3 Glacier client using %s\n", c.endpoint)
	vault := fmt.Sprintf("/-/vaults/glacier-example-%d", time.Now().UnixNano())
	c.call("CreateVault", http.MethodPut, vault, nil, nil)

	small := []byte("archived at " + time.Now().UTC().Format(time.RFC3339))
	h, _ := c.call("UploadArchive", http.MethodPost, vault+"/archives", small, map[string]string{"x-amz-sha256-tree-hash": treeHash(small), "x-amz-archive-description": "small"})
	archiveID := h.Get("x-amz-archive-id")
	fmt.Printf("uploaded archive %s...\n", archiveID[:16])

	big := bytes.Repeat([]byte("stackyard "), (partSize+4096)/10)
	h, _ = c.call("InitiateMultipartUpload", http.MethodPost, vault+"/multipart-uploads", nil, map[string]string{"x-amz-part-size": fmt.Sprint(partSize)})
	upload := vault + "/multipart-uploads/" + h.Get("x-amz-multipart-upload-id")
	for start := 0; start < len(big); start += partSize {
		part := big[start:min(start+partSize, len(big))]
		c.call("UploadMultipartPart", http.MethodPut, upload, part, map[string]string{"Content-Range": fmt.Sprintf("bytes %d-%d/*", start, start+len(part)-1), "x-amz-sha256-tree-hash": treeHash(part)})
	}
	h, _ = c.call("CompleteMultipartUpload", http.MethodPost, upload, nil, map[string]string{"x-amz-archive-size": fmt.Sprint(len(big)), "x-amz-sha256-tree-hash": treeHash(big)})
	fmt.Printf("completed a %d-byte multipart archive\n", len(big))

	h, _ = c.call("InitiateJob", http.MethodPost, vault+"/jobs", jsonBody(map[string]string{"Type": "archive-retrieval", "ArchiveId": archiveID}), nil)
	job := vault + "/jobs/" + h.Get("x-amz-job-id")
	waitFor(c, job)
	_, out := c.call("GetJobOutput", http.MethodGet, job+"/output", nil, nil)
	if !bytes.Equal(out, small) {
		exitf("retrieved %q, want %q", out, small)
	}
	fmt.Printf("retrieved the archive: %q\n", out)

	h, _ = c.call("InitiateJob", http.MethodPost, vault+"/jobs", jsonBody(map[string]string{"Type": "inventory-retrieval"}), nil)
	inventory := vault + "/jobs/" + h.Get("x-amz-job-id")
	waitFor(c, inventory)
	_, raw := c.call("GetJobOutput", http.MethodGet, inventory+"/output", nil, nil)
	var inv struct{ ArchiveList []map[string]any }
	if err := json.Unmarshal(raw, &inv); err != nil || len(inv.ArchiveList) != 2 {
		exitf("inventory: %v %s", err, raw)
	}
	_, raw = c.call("DescribeVault", http.MethodGet, vault, nil, nil)
	fmt.Printf("vault after inventory: %s\n", raw)

	for _, a := range inv.ArchiveList {
		c.call("DeleteArchive", http.MethodDelete, vault+"/archives/"+fmt.Sprint(a["ArchiveId"]), nil, nil)
	}
	c.call("DeleteVault", http.MethodDelete, vault, nil, nil)
	fmt.Println("S3 Glacier example completed")
}

// waitFor reads a job until it succeeds (jobs settle on read).
func waitFor(c client, job string) {
	for i := 0; i < 5; i++ {
		_, raw := c.call("DescribeJob", http.MethodGet, job, nil, nil)
		var j struct{ StatusCode string }
		_ = json.Unmarshal(raw, &j)
		if j.StatusCode == "Succeeded" {
			return
		}
	}
	exitf("job %s never succeeded", job)
}

// treeHash is the Glacier SHA-256 tree hash over 1 MiB chunks.
func treeHash(data []byte) string {
	var level [][]byte
	for off := 0; off < len(data); off += partSize {
		sum := sha256.Sum256(data[off:min(off+partSize, len(data))])
		level = append(level, sum[:])
	}
	for len(level) > 1 {
		var next [][]byte
		for i := 0; i < len(level); i += 2 {
			if i+1 == len(level) {
				next = append(next, level[i])
				continue
			}
			sum := sha256.Sum256(append(append([]byte(nil), level[i]...), level[i+1]...))
			next = append(next, sum[:])
		}
		level = next
	}
	return hex.EncodeToString(level[0])
}

func jsonBody(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

// call signs a Glacier request and exits unless it returns 2xx.
func (c client) call(op, method, path string, body []byte, headers map[string]string) (http.Header, []byte) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("x-amz-glacier-version", "2012-06-01")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	req.Header.Set("x-amz-content-sha256", hex.EncodeToString(sum[:]))
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "glacier", c.region, time.Now()); err != nil {
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
	return resp.Header, raw
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
