// Command account runs AWS Account Management against Stackyard: read the
// account, set an alternate contact and contact information, verify the
// phone number, change a member account's primary email, and opt into an
// opt-in Region and back out again.
//
// The caller is the organization's management account; the member account
// comes from Stackyard's Organizations. Stackyard sends no email or SMS: a
// one-time code is the first six characters (upper-case hex) of
// SHA-256("<account>:<email or phone>"). The SDK has no Account module in
// this repository's go.mod, so requests are signed by hand.
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

const member = "210987654321"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Account client using %s\n", c.endpoint)

	info := c.call("GetAccountInformation", map[string]any{})
	fmt.Printf("account %v (%v) is %v\n", info["AccountId"], info["AccountName"], info["AccountState"])

	c.call("PutAlternateContact", map[string]any{
		"AlternateContactType": "SECURITY", "Name": "Security Team", "Title": "SecOps",
		"EmailAddress": "security@example.com", "PhoneNumber": "+1 206 555 0199",
	})
	contact := c.call("GetAlternateContact", map[string]any{"AlternateContactType": "SECURITY"})["AlternateContact"].(map[string]any)
	fmt.Printf("security contact: %v <%v>\n", contact["Name"], contact["EmailAddress"])

	phone := fmt.Sprintf("+1 206 555 %04d", time.Now().UnixNano()%10000)
	c.call("PutContactInformation", map[string]any{"ContactInformation": map[string]any{
		"FullName": "Stackyard Example", "AddressLine1": "1 Main St", "City": "Seattle", "StateOrRegion": "WA",
		"PostalCode": "98101", "CountryCode": "US", "PhoneNumber": phone,
	}})
	c.call("SendPhoneNumberVerification", map[string]any{})
	verified := c.call("VerifyPhoneNumber", map[string]any{"Otp": otp("123456789012", phone)})
	fmt.Printf("phone %s: %v\n", phone, verified["Status"])

	email := fmt.Sprintf("member-%d@example.com", time.Now().UnixNano())
	c.call("StartPrimaryEmailUpdate", map[string]any{"AccountId": member, "PrimaryEmail": email})
	accepted := c.call("AcceptPrimaryEmailUpdate", map[string]any{"AccountId": member, "PrimaryEmail": email, "Otp": otp(member, email)})
	now := c.call("GetPrimaryEmail", map[string]any{"AccountId": member})
	fmt.Printf("member primary email update %v: now %v\n", accepted["Status"], now["PrimaryEmail"])

	region := "eu-central-2"
	c.call("EnableRegion", map[string]any{"RegionName": region})
	fmt.Printf("%s: %s\n", region, c.waitRegion(region, "ENABLED"))
	c.call("DisableRegion", map[string]any{"RegionName": region})
	fmt.Printf("%s: %s\n", region, c.waitRegion(region, "DISABLED"))
}

// waitRegion polls a Region until it reaches want, printing each status.
func (c client) waitRegion(region, want string) string {
	var seen []string
	for range 5 {
		status := c.call("GetRegionOptStatus", map[string]any{"RegionName": region})["RegionOptStatus"].(string)
		seen = append(seen, status)
		if status == want {
			return strings.Join(seen, " -> ")
		}
	}
	exitf("%s never reached %s: %v", region, want, seen)
	return ""
}

func otp(account, value string) string {
	sum := sha256.Sum256([]byte(account + ":" + value))
	return strings.ToUpper(hex.EncodeToString(sum[:3]))
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

// call sends one REST JSON operation (POST /<lowerCamelOp>) and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	path := "/" + strings.ToLower(op[:1]) + op[1:]
	return decode(op, c.send(op, http.MethodPost, "account", c.endpoint+path, body, map[string]string{"Content-Type": "application/json"}))
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
