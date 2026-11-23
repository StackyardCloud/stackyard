// Command polly walks an Amazon Polly session against Stackyard: list voices,
// synthesize MP3 audio and word speech marks, store a lexicon and use it,
// then run a synthesis task that writes to an S3 bucket.
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

const lexicon = `<?xml version="1.0" encoding="UTF-8"?>
<lexicon version="1.0" xmlns="http://www.w3.org/2005/01/pronunciation-lexicon" alphabet="ipa" xml:lang="en-US">
  <lexeme><grapheme>W3C</grapheme><alias>World Wide Web Consortium</alias></lexeme>
</lexicon>`

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
	fmt.Printf("Stackyard Amazon Polly client using %s\n", c.endpoint)

	voices := c.json("DescribeVoices", http.MethodGet, "/v1/voices?Engine=neural&LanguageCode=en-US", nil)["Voices"].([]any)
	fmt.Println("Neural en-US voices:", len(voices))

	audio, hdr := c.send("SynthesizeSpeech", "polly", http.MethodPost, c.endpoint+"/v1/speech",
		mustJSON(map[string]any{"OutputFormat": "mp3", "Text": "Hello from Stackyard", "VoiceId": "Joanna"}))
	fmt.Println("MP3:", hdr.Get("Content-Type"), len(audio), "bytes,", hdr.Get("x-amzn-RequestCharacters"), "characters")

	marks, _ := c.send("SynthesizeSpeech", "polly", http.MethodPost, c.endpoint+"/v1/speech",
		mustJSON(map[string]any{"OutputFormat": "json", "SpeechMarkTypes": []string{"word"}, "Text": "Hello from Stackyard", "VoiceId": "Joanna"}))
	fmt.Println("Word marks:", strings.Count(string(marks), "\n"))

	name := fmt.Sprintf("sy%d", time.Now().UnixNano()%1_000_000_000)
	c.json("PutLexicon", http.MethodPut, "/v1/lexicons/"+name, map[string]any{"Content": lexicon})
	attrs := c.json("GetLexicon", http.MethodGet, "/v1/lexicons/"+name, nil)["LexiconAttributes"].(map[string]any)
	fmt.Println("Lexicon:", name, attrs["Alphabet"], attrs["LanguageCode"], attrs["LexemesCount"], "lexeme")
	c.send("SynthesizeSpeech", "polly", http.MethodPost, c.endpoint+"/v1/speech",
		mustJSON(map[string]any{"OutputFormat": "pcm", "LexiconNames": []string{name}, "Text": "W3C", "VoiceId": "Joanna"}))

	bucket := "stackyard-polly-example"
	c.send("CreateBucket", "s3", http.MethodPut, c.endpoint+"/"+bucket, nil)
	task := c.json("StartSpeechSynthesisTask", http.MethodPost, "/v1/synthesisTasks", map[string]any{
		"OutputFormat": "mp3", "OutputS3BucketName": bucket, "Text": "A longer passage for a task.", "VoiceId": "Joanna",
	})["SynthesisTask"].(map[string]any)
	id := task["TaskId"].(string)
	for i := 0; i < 2; i++ {
		task = c.json("GetSpeechSynthesisTask", http.MethodGet, "/v1/synthesisTasks/"+id, nil)["SynthesisTask"].(map[string]any)
	}
	fmt.Println("Task:", task["TaskStatus"], task["OutputUri"])

	c.json("DeleteLexicon", http.MethodDelete, "/v1/lexicons/"+name, nil)
	fmt.Println("Deleted lexicon", name)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		exitf("encode: %v", err)
	}
	return b
}

func (c client) json(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		body = mustJSON(payload)
	}
	raw, _ := c.send(op, "polly", method, c.endpoint+path, body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, service, method, target string, body []byte) ([]byte, http.Header) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
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
	return raw, resp.Header
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
