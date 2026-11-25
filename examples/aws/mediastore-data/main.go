// Command mediastore-data stores and serves objects through AWS Elemental
// MediaStore Data against Stackyard: create a container (MediaStore control
// plane), put an HLS playlist and a segment at its endpoint, list folders,
// read a byte range, then delete the objects and the container.
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

func main() {
	c := newClient()
	fmt.Printf("Stackyard MediaStore Data client using %s\n", c.endpoint)
	name := fmt.Sprintf("stackyard-data-example-%d", time.Now().UnixNano())
	c.mediastore("CreateContainer", map[string]any{"ContainerName": name})
	endpoint := c.mediastore("DescribeContainer", map[string]any{"ContainerName": name})["Container"].(map[string]any)["Endpoint"].(string)
	fmt.Println("Container endpoint:", endpoint)

	playlist := []byte("#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXTINF:6.0,\nseg1.ts\n#EXT-X-ENDLIST\n")
	_, raw := c.send("PutObject", "mediastore", http.MethodPut, endpoint+"/shows/ep1/index.m3u8", playlist,
		"Content-Type", "application/vnd.apple.mpegurl", "Cache-Control", "max-age=60")
	fmt.Println("Put playlist, ETag:", decode("PutObject", raw)["ETag"])
	c.send("PutObject", "mediastore", http.MethodPut, endpoint+"/shows/ep1/seg1.ts", bytes.Repeat([]byte{0x47}, 188), "Content-Type", "video/MP2T")

	_, raw = c.send("ListItems", "mediastore", http.MethodGet, endpoint+"/", nil)
	fmt.Println("Root items:", decode("ListItems", raw)["Items"])
	_, raw = c.send("ListItems", "mediastore", http.MethodGet, endpoint+"/?Path=shows/ep1", nil)
	fmt.Println("shows/ep1 items:", len(decode("ListItems", raw)["Items"].([]any)))

	h, body := c.send("GetObject", "mediastore", http.MethodGet, endpoint+"/shows/ep1/index.m3u8", nil, "Range", "bytes=0-6")
	fmt.Printf("Range read: %q (%s, %s)\n", body, h.Get("Content-Range"), h.Get("Content-Type"))
	if string(body) != "#EXTM3U" {
		exitf("GetObject: unexpected range body %q", body)
	}

	for _, p := range []string{"/shows/ep1/index.m3u8", "/shows/ep1/seg1.ts"} {
		c.send("DeleteObject", "mediastore", http.MethodDelete, endpoint+p, nil)
	}
	c.mediastore("DeleteContainer", map[string]any{"ContainerName": name})
	fmt.Println("Deleted objects and container", name)
}

// mediastore calls a MediaStore control-plane (JSON 1.1) operation.
func (c client) mediastore(op string, payload any) map[string]any {
	body, _ := json.Marshal(payload)
	_, raw := c.send(op, "mediastore", http.MethodPost, c.endpoint+"/", body,
		"Content-Type", "application/x-amz-json-1.1", "X-Amz-Target", "MediaStore_20170901."+op)
	return decode(op, raw)
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// send signs a request for service and exits unless it returns 2xx. Extra
// headers come in name/value pairs.
func (c client) send(op, service, method, target string, body []byte, headers ...string) (http.Header, []byte) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
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

func decode(op string, raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
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
