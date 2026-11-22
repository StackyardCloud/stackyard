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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	ctx    context.Context
	region string
	creds  aws.CredentialsProvider
}

type reply struct {
	status int
	header http.Header
	body   []byte
}

// main creates a video stream, uploads three MKV fragments with PutMedia,
// reads them back with GetMedia and ListFragments, and opens an HLS session.
func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	stream := getenv("STACKYARD_KVS_STREAM", "camera-1")
	c := client{
		ctx:    context.Background(),
		region: getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		),
	}
	fmt.Printf("Stackyard Kinesis Video media client using %s\n", endpoint)

	r := c.json(endpoint+"/createStream", map[string]any{"StreamName": stream, "DataRetentionInHours": 24, "MediaType": "video/h264"})
	expect2xx("CreateStream", r)

	// Media APIs go to the stream's data endpoint (Stackyard returns itself).
	r = c.json(endpoint+"/getDataEndpoint", map[string]any{"StreamName": stream, "APIName": "PUT_MEDIA"})
	expect2xx("GetDataEndpoint", r)
	var de struct{ DataEndpoint string }
	if err := json.Unmarshal(r.body, &de); err != nil || de.DataEndpoint == "" {
		exitf("GetDataEndpoint returned no endpoint: %s", r.body)
	}
	data := strings.TrimRight(de.DataEndpoint, "/")
	logf("data endpoint %s", data)

	r = c.do(data+"/putMedia", testMKV(0, 2000, 4000), map[string]string{
		"x-amzn-stream-name":              stream,
		"x-amzn-fragment-timecode-type":   "RELATIVE",
		"x-amzn-producer-start-timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	})
	expect2xx("PutMedia", r)
	if strings.Contains(string(r.body), `"ERROR"`) || strings.Count(string(r.body), `"PERSISTED"`) != 3 {
		exitf("PutMedia acks: %s", r.body)
	}
	logf("PutMedia acks:\n%s", strings.TrimSpace(string(r.body)))

	r = c.json(data+"/getMedia", map[string]any{"StreamName": stream, "StartSelector": map[string]any{"StartSelectorType": "EARLIEST"}})
	expect2xx("GetMedia", r)
	if n := bytes.Count(r.body, []byte("AWS_KINESISVIDEO_FRAGMENT_NUMBER")); n != 3 {
		exitf("GetMedia returned %d fragments, want 3", n)
	}
	logf("GetMedia returned %d bytes of %s", len(r.body), r.header.Get("Content-Type"))

	r = c.json(data+"/listFragments", map[string]any{"StreamName": stream})
	expect2xx("ListFragments", r)
	var list struct {
		Fragments []struct{ FragmentNumber string }
	}
	if err := json.Unmarshal(r.body, &list); err != nil || len(list.Fragments) != 3 {
		exitf("ListFragments: %s", r.body)
	}

	r = c.json(data+"/getHLSStreamingSessionURL", map[string]any{"StreamName": stream, "PlaybackMode": "LIVE"})
	expect2xx("GetHLSStreamingSessionURL", r)
	var hls struct{ HLSStreamingSessionURL string }
	if err := json.Unmarshal(r.body, &hls); err != nil || hls.HLSStreamingSessionURL == "" {
		exitf("GetHLSStreamingSessionURL: %s", r.body)
	}
	resp, err := http.Get(hls.HLSStreamingSessionURL) // session URLs carry their own token
	if err != nil {
		exitf("HLS master playlist: %v", err)
	}
	playlist, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(string(playlist), "#EXTM3U") {
		exitf("HLS master playlist: %d %s", resp.StatusCode, playlist)
	}
	logf("HLS master playlist:\n%s", strings.TrimSpace(string(playlist)))
	fmt.Println("Kinesis Video media example completed")
}

// testMKV builds a tiny Matroska stream: one H.264 track and one frame per
// cluster at the given cluster timecodes (milliseconds).
func testMKV(clusterMillis ...uint64) []byte {
	el := func(id []byte, content []byte) []byte {
		size := make([]byte, 8)
		binary.BigEndian.PutUint64(size, uint64(len(content)))
		size[0] = 0x01
		return append(append(append([]byte{}, id...), size...), content...)
	}
	out := el([]byte{0x1A, 0x45, 0xDF, 0xA3}, el([]byte{0x42, 0x82}, []byte("matroska")))
	out = append(out, 0x18, 0x53, 0x80, 0x67, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF)
	out = append(out, el([]byte{0x15, 0x49, 0xA9, 0x66}, el([]byte{0x2A, 0xD7, 0xB1}, []byte{0x0F, 0x42, 0x40}))...)
	track := append(append(el([]byte{0xD7}, []byte{1}), el([]byte{0x83}, []byte{1})...), el([]byte{0x86}, []byte("V_MPEG4/ISO/AVC"))...)
	out = append(out, el([]byte{0x16, 0x54, 0xAE, 0x6B}, el([]byte{0xAE}, track))...)
	for _, ms := range clusterMillis {
		tc := make([]byte, 8)
		binary.BigEndian.PutUint64(tc, ms)
		cluster := append(el([]byte{0xE7}, tc), el([]byte{0xA3}, []byte{0x81, 0x00, 0x21, 0x80, 'f', 'r', 'a', 'm', 'e'})...)
		out = append(out, el([]byte{0x1F, 0x43, 0xB6, 0x75}, cluster)...)
	}
	return out
}

func (c client) json(url string, payload any) reply {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("marshal %s: %v", url, err)
	}
	return c.do(url, body, map[string]string{"Content-Type": "application/json"})
}

// do sends a POST signed for "kinesisvideo", the signing name of the
// control plane and both media planes.
func (c client) do(url string, body []byte, headers map[string]string) reply {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		exitf("build %s: %v", url, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	credValue, err := c.creds.Retrieve(c.ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(c.ctx, credValue, req, hex.EncodeToString(sum[:]), "kinesisvideo", c.region, time.Now()); err != nil {
		exitf("sign %s: %v", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", url, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read %s: %v", url, err)
	}
	return reply{status: resp.StatusCode, header: resp.Header, body: respBody}
}

func expect2xx(action string, r reply) {
	if r.status >= 200 && r.status < 300 {
		logf("%s returned %d", action, r.status)
		return
	}
	exitf("expected %s to return 2xx, got %d: %s", action, r.status, strings.TrimSpace(string(r.body)))
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
