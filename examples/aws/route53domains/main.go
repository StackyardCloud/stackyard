// Command route53domains walks an Amazon Route 53 Domains lifecycle against
// Stackyard: check availability, register a domain, follow its operation to
// SUCCESSFUL, read the domain (its name servers come from the hosted zone made
// at registration), update name servers and tags, renew it, view billing and
// delete it.
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

const target = "Route53Domains_v20140515."

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   "us-east-1", // Route 53 Domains is served from us-east-1 only
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Route 53 Domains client using %s\n", c.endpoint)
	domain := fmt.Sprintf("stackyard-%d.com", time.Now().UnixNano()%1_000_000_000)

	avail := c.call("CheckDomainAvailability", map[string]any{"DomainName": domain})
	fmt.Printf("%s is %v\n", domain, avail["Availability"])

	contact := map[string]any{
		"FirstName": "Ada", "LastName": "Lovelace", "ContactType": "PERSON", "AddressLine1": "1 Main St",
		"City": "Seattle", "State": "WA", "CountryCode": "US", "ZipCode": "98101",
		"PhoneNumber": "+1.2065550100", "Email": "ada@example.com",
	}
	reg := c.call("RegisterDomain", map[string]any{
		"DomainName": domain, "DurationInYears": 1, "AdminContact": contact, "RegistrantContact": contact, "TechContact": contact,
	})
	opID, _ := reg["OperationId"].(string)
	for status := ""; status != "SUCCESSFUL"; {
		op := c.call("GetOperationDetail", map[string]any{"OperationId": opID})
		status, _ = op["Status"].(string)
		fmt.Printf("operation %s: %s\n", opID, status)
		if status == "ERROR" || status == "FAILED" {
			exitf("registration failed: %v", op)
		}
	}

	detail := c.call("GetDomainDetail", map[string]any{"DomainName": domain})
	exp, _ := detail["ExpirationDate"].(float64)
	fmt.Printf("registered %s, expires %s, name servers %v\n", domain, time.Unix(int64(exp), 0).UTC().Format("2006-01-02"), detail["Nameservers"])

	c.call("UpdateDomainNameservers", map[string]any{"DomainName": domain, "Nameservers": []map[string]any{{"Name": "ns1.example.net"}, {"Name": "ns2.example.net"}}})
	c.call("UpdateTagsForDomain", map[string]any{"DomainName": domain, "TagsToUpdate": []map[string]any{{"Key": "team", "Value": "platform"}}})
	tags := c.call("ListTagsForDomain", map[string]any{"DomainName": domain})
	fmt.Printf("tags: %v\n", tags["TagList"])

	expiry := time.Unix(int64(exp), 0).UTC()
	c.call("RenewDomain", map[string]any{"DomainName": domain, "DurationInYears": 1, "CurrentExpiryYear": expiry.Year()})
	bills := c.call("ViewBilling", map[string]any{})
	fmt.Printf("billing records: %v\n", bills["BillingRecords"])

	c.call("DeleteDomain", map[string]any{"DomainName": domain})
	fmt.Printf("deleted %s\n", domain)
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "route53domains", c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
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
