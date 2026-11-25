// Command cloudsearch-domain uploads documents to an Amazon CloudSearch domain
// on Stackyard and searches them: simple and structured queries, facets,
// stats, highlights and suggestions.
//
// AWS serves each domain at its own doc-/search- host; Stackyard also takes
// the domain name as a path prefix (http://localhost:4566/<domain>), which is
// what this example uses. Requests are signed by hand (service cloudsearch).
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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard CloudSearch Domain client using %s\n", c.endpoint)
	domain := fmt.Sprintf("films-%d", time.Now().UnixNano()%1_000_000_000)

	// Configure the domain through the CloudSearch configuration API.
	query := func(action string, v url.Values) {
		v.Set("Action", action)
		v.Set("Version", "2013-01-01")
		v.Set("DomainName", domain)
		c.send(action, http.MethodPost, "cloudsearch", c.endpoint+"/", []byte(v.Encode()), map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	}
	query("CreateDomain", url.Values{})
	query("DefineIndexField", url.Values{"IndexField.IndexFieldName": {"title"}, "IndexField.IndexFieldType": {"text"}})
	query("DefineIndexField", url.Values{"IndexField.IndexFieldName": {"genre"}, "IndexField.IndexFieldType": {"literal"}})
	query("DefineIndexField", url.Values{"IndexField.IndexFieldName": {"year"}, "IndexField.IndexFieldType": {"int"}})
	query("DefineSuggester", url.Values{"Suggester.SuggesterName": {"titles"}, "Suggester.DocumentSuggesterOptions.SourceField": {"title"}})
	query("IndexDocuments", url.Values{})
	base := c.endpoint + "/" + domain + "/2013-01-01"

	batch, _ := json.Marshal([]any{
		map[string]any{"type": "add", "id": "1", "fields": map[string]any{"title": "Star Wars", "genre": "scifi", "year": 1977}},
		map[string]any{"type": "add", "id": "2", "fields": map[string]any{"title": "Star Trek", "genre": "scifi", "year": 1979}},
		map[string]any{"type": "add", "id": "3", "fields": map[string]any{"title": "Casablanca", "genre": "drama", "year": 1942}},
	})
	up := decode(c.send("UploadDocuments", http.MethodPost, "cloudsearch", base+"/documents/batch?format=sdk", batch, map[string]string{"Content-Type": "application/json"}))
	fmt.Println("upload:", up["status"], "adds:", up["adds"])

	res := c.get("Search", base+"/search", url.Values{"q": {"star"}, "sort": {"year desc"}, "return": {"title,year"},
		"facet": {`{"genre":{}}`}, "stats": {`{"year":{}}`}, "highlight": {`{"title":{}}`}})
	hits := res["hits"].(map[string]any)
	fmt.Println("found:", hits["found"])
	for _, h := range hits["hit"].([]any) {
		hit := h.(map[string]any)
		fmt.Println(" ", hit["id"], hit["fields"], hit["highlights"])
	}
	fmt.Println("genre facet:", res["facets"].(map[string]any)["genre"])
	fmt.Println("year stats:", res["stats"].(map[string]any)["year"])

	structured := c.get("Search", base+"/search", url.Values{"q": {"(and genre:'scifi' (range field=year [1978,}))"}, "q.parser": {"structured"}})
	fmt.Println("structured hits:", structured["hits"].(map[string]any)["found"])

	sug := c.get("Suggest", base+"/suggest", url.Values{"q": {"cas"}, "suggester": {"titles"}})
	fmt.Println("suggestions:", sug["suggest"].(map[string]any)["suggestions"])
}

func (c client) get(op, target string, params url.Values) map[string]any {
	return decode(c.send(op, http.MethodGet, "cloudsearch", target+"?"+params.Encode(), nil, nil))
}

func decode(raw []byte) map[string]any {
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("decode: %v", err)
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
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

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
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
