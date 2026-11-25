// Command cloudsearch runs the Amazon CloudSearch configuration API against
// Stackyard: create a domain, define text, literal and int index fields, an
// expression and a suggester, index, inspect the deployed configuration,
// update scaling and access policies, and delete the domain.
//
// The SDK has no CloudSearch module in this repository's go.mod, so Query
// requests are signed by hand.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
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
	fmt.Printf("Stackyard CloudSearch client using %s\n", c.endpoint)
	domain := fmt.Sprintf("movies-%d", time.Now().UnixNano()%1_000_000_000)

	status := c.call("CreateDomain", url.Values{"DomainName": {domain}})
	fmt.Println("domain:", find(status, "ARN"), "search endpoint:", find(status, "Endpoint"))

	for _, f := range []url.Values{
		{"IndexField.IndexFieldName": {"title"}, "IndexField.IndexFieldType": {"text"}},
		{"IndexField.IndexFieldName": {"genre"}, "IndexField.IndexFieldType": {"literal"}, "IndexField.LiteralOptions.FacetEnabled": {"true"}},
		{"IndexField.IndexFieldName": {"year"}, "IndexField.IndexFieldType": {"int"}, "IndexField.IntOptions.SortEnabled": {"true"}},
	} {
		f.Set("DomainName", domain)
		out := c.call("DefineIndexField", f)
		fmt.Println("defined", f.Get("IndexField.IndexFieldName"), "state", find(out, "State"))
	}
	c.call("DefineExpression", url.Values{"DomainName": {domain}, "Expression.ExpressionName": {"recent"}, "Expression.ExpressionValue": {"year*2"}})
	c.call("DefineSuggester", url.Values{"DomainName": {domain}, "Suggester.SuggesterName": {"titles"},
		"Suggester.DocumentSuggesterOptions.SourceField": {"title"}, "Suggester.DocumentSuggesterOptions.FuzzyMatching": {"low"}})

	indexed := c.call("IndexDocuments", url.Values{"DomainName": {domain}})
	fmt.Println("indexed fields:", findAll(indexed, "member"))
	deployed := c.call("DescribeIndexFields", url.Values{"DomainName": {domain}, "Deployed": {"true"}})
	fmt.Println("deployed field states:", findAll(deployed, "State"))

	c.call("UpdateScalingParameters", url.Values{"DomainName": {domain}, "ScalingParameters.DesiredInstanceType": {"search.medium"}, "ScalingParameters.DesiredReplicationCount": {"2"}})
	c.call("UpdateServiceAccessPolicies", url.Values{"DomainName": {domain}, "AccessPolicies": {`{"Statement":[{"Effect":"Allow","Principal":"*","Action":["cloudsearch:search"]}]}`}})
	described := c.call("DescribeDomains", url.Values{"DomainNames.member.1": {domain}})
	fmt.Println("instance type:", find(described, "SearchInstanceType"), "instances:", find(described, "SearchInstanceCount"))

	names := c.call("ListDomainNames", url.Values{})
	fmt.Println("domains:", findAll(names, "key"))
	c.call("DeleteDomain", url.Values{"DomainName": {domain}})
	fmt.Println("deleted", domain)
}

func (c client) call(action string, params url.Values) []byte {
	params.Set("Action", action)
	params.Set("Version", "2013-01-01")
	return c.send(action, http.MethodPost, "cloudsearch", c.endpoint+"/", []byte(params.Encode()),
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
}

// find returns the first element named name in an XML body.
func find(body []byte, name string) string {
	if all := findAll(body, name); len(all) > 0 {
		return all[0]
	}
	return ""
}

func findAll(body []byte, name string) []string {
	var out []string
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == name {
			var v string
			if dec.DecodeElement(&v, &se) == nil {
				out = append(out, v)
			}
		}
	}
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
