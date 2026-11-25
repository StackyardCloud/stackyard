// Command taxsettings runs Tax Settings against Stackyard: register a VAT
// number for the account and, in one batch, for the organization's member
// account with a verification document; copy that document to an S3 bucket;
// record a US resale exemption; and opt member accounts into inheriting the
// management account's tax settings.
//
// The SDK has no Tax Settings module in this repository's go.mod, so
// requests are signed by hand.
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

const member = "210987654321"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Tax Settings client using %s\n", c.endpoint)

	entry := map[string]any{
		"registrationId": "DE123456789", "registrationType": "VAT", "legalName": "Stackyard GmbH",
		"legalAddress": map[string]any{"addressLine1": "Hauptstr. 1", "city": "Berlin", "postalCode": "10115", "countryCode": "DE"},
	}
	put := c.call("PutTaxRegistration", map[string]any{"taxRegistrationEntry": entry})
	fmt.Printf("account registration: %v\n", put["status"])

	withDoc := map[string]any{}
	for k, v := range entry {
		withDoc[k] = v
	}
	withDoc["verificationDetails"] = map[string]any{"taxRegistrationDocuments": []any{map[string]any{
		"file": map[string]any{"fileName": "vat-certificate.pdf", "fileContent": base64.StdEncoding.EncodeToString([]byte("VAT certificate"))},
	}}}
	batch := c.call("BatchPutTaxRegistration", map[string]any{"accountIds": []any{member}, "taxRegistrationEntry": withDoc})
	fmt.Printf("member registration: %v (%d errors)\n", batch["status"], len(batch["errors"].([]any)))

	for _, d := range c.call("ListTaxRegistrations", map[string]any{})["accountDetails"].([]any) {
		reg := d.(map[string]any)["taxRegistration"].(map[string]any)
		fmt.Printf("  %v: %v %v %v\n", d.(map[string]any)["accountId"], reg["registrationType"], reg["registrationId"], reg["status"])
	}

	bucket := fmt.Sprintf("tax-documents-%d", time.Now().UnixNano())
	c.send("CreateBucket", http.MethodPut, "s3", c.endpoint+"/"+bucket, nil, nil)
	reg := c.call("GetTaxRegistration", map[string]any{"accountId": member})["taxRegistration"].(map[string]any)
	meta := reg["taxDocumentMetadatas"].([]any)[0]
	doc := c.call("GetTaxRegistrationDocument", map[string]any{
		"taxDocumentMetadata": meta, "destinationS3Location": map[string]any{"bucket": bucket, "prefix": "vat"},
	})
	fmt.Printf("document copied to %v\n", doc["destinationFilePath"])

	exemption := c.call("PutTaxExemption", map[string]any{
		"accountIds": []any{"123456789012"}, "authority": map[string]any{"country": "US", "state": "CA"}, "exemptionType": "Resale",
		"exemptionCertificate": map[string]any{"documentName": "resale.pdf", "documentFile": base64.StdEncoding.EncodeToString([]byte("resale"))},
	})
	fmt.Printf("exemption case %v\n", exemption["caseId"])

	c.call("PutTaxInheritance", map[string]any{"heritageStatus": "OptIn"})
	details := c.call("ListTaxExemptions", map[string]any{})["taxExemptionDetailsMap"].(map[string]any)
	if m, ok := details[member].(map[string]any); ok {
		fmt.Printf("member inherits from %v: %v\n", m["heritageObtainedParentEntity"], m["heritageObtainedDetails"])
	}
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

func decode(op string, raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// call sends one REST JSON operation (POST /<Op>) and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	return decode(op, c.send(op, http.MethodPost, "tax", c.endpoint+"/"+op, body, map[string]string{"Content-Type": "application/json"}))
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
