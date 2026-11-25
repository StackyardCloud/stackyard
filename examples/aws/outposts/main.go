// Command outposts runs AWS Outposts against Stackyard: create a site and an
// outpost, quote and order a rack, follow the order to FULFILLED, list the
// installed assets, re-slot capacity with a capacity task, read billing and
// renewal pricing, then decommission and clean up.
//
// The SDK has no Outposts module in this repository's go.mod, so requests
// are signed by hand.
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

const signingName = "outposts"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Outposts client using %s\n", c.endpoint)
	run := time.Now().UnixNano()
	address := map[string]any{"ContactName": "Field Ops", "ContactPhoneNumber": "+15555550100", "AddressLine1": "1 Main St",
		"City": "Seattle", "StateOrRegion": "WA", "PostalCode": "98101", "CountryCode": "US"}

	site := c.rest("CreateSite", http.MethodPost, "/sites", map[string]any{
		"Name": fmt.Sprintf("plant-%d", run), "OperatingAddress": address, "ShippingAddress": address,
	})["Site"].(map[string]any)
	siteID := site["SiteId"].(string)
	fmt.Printf("site %s in %v\n", siteID, site["OperatingAddressCity"])
	op := c.rest("CreateOutpost", http.MethodPost, "/outposts", map[string]any{"Name": "line-1", "SiteId": siteID})["Outpost"].(map[string]any)
	opID := op["OutpostId"].(string)
	fmt.Printf("outpost %s is %v\n", opID, op["LifeCycleStatus"])

	quote := c.rest("CreateQuote", http.MethodPost, "/quotes", map[string]any{
		"OutpostIdentifier": opID, "CountryCode": "US",
		"RequestedCapacities": []any{map[string]any{"QuoteCapacityType": "EC2", "Unit": "m5", "Quantity": 96}},
	})["Quote"].(map[string]any)
	option := quote["QuoteOptions"].([]any)[0].(map[string]any)
	price := option["PricingOptions"].([]any)[0].(map[string]any)["SubscriptionPricingDetails"].(map[string]any)
	fmt.Printf("quote %s: %d rack(s), %v upfront\n", quote["QuoteId"], len(option["Specifications"].([]any)), price["UpfrontPrice"])

	order := c.rest("CreateOrder", http.MethodPost, "/orders", map[string]any{
		"OutpostIdentifier": opID, "QuoteIdentifier": quote["QuoteId"], "QuoteOptionIdentifier": option["QuoteOptionIdentifier"],
		"PaymentOption": "ALL_UPFRONT", "PaymentTerm": "THREE_YEARS",
	})["Order"].(map[string]any)
	orderID := order["OrderId"].(string)
	for order["Status"] != "FULFILLED" {
		order = c.rest("GetOrder", http.MethodGet, "/orders/"+orderID, nil)["Order"].(map[string]any)
		fmt.Printf("order %s: %v\n", orderID, order["Status"])
	}

	assets := c.rest("ListAssets", http.MethodGet, "/outposts/"+opID+"/assets", nil)["Assets"].([]any)
	for _, a := range assets {
		attrs := a.(map[string]any)["ComputeAttributes"].(map[string]any)
		fmt.Printf("asset %v: %v vCPUs, %v\n", a.(map[string]any)["AssetId"], attrs["MaxVcpus"], attrs["State"])
	}

	base := "/outposts/" + opID + "/capacity"
	task := c.rest("StartCapacityTask", http.MethodPost, base, map[string]any{
		"InstancePools": []any{map[string]any{"InstanceType": "m5.4xlarge", "Count": 4}, map[string]any{"InstanceType": "m5.xlarge", "Count": 8}},
	})
	taskID := task["CapacityTaskId"].(string)
	for task["CapacityTaskStatus"] != "COMPLETED" && task["CapacityTaskStatus"] != "FAILED" {
		task = c.rest("GetCapacityTask", http.MethodGet, base+"/"+taskID, nil)
		fmt.Printf("capacity task %s: %v\n", taskID, task["CapacityTaskStatus"])
	}

	bill := c.rest("GetOutpostBillingInformation", http.MethodGet, "/outpost/"+opID+"/billing-information", nil)
	fmt.Printf("contract ends %v with %d subscription(s)\n", bill["ContractEndDate"], len(bill["Subscriptions"].([]any)))
	renew := c.rest("GetRenewalPricing", http.MethodGet, "/outpost/"+opID+"/renewal-pricing", nil)
	fmt.Printf("renewal pricing: %v (%d options)\n", renew["PricingResult"], len(renew["PricingOptions"].([]any)))

	arn := url.PathEscape(op["OutpostArn"].(string))
	c.rest("TagResource", http.MethodPost, "/tags/"+arn, map[string]any{"Tags": map[string]any{"env": "dev"}})
	fmt.Println("tags:", c.rest("ListTagsForResource", http.MethodGet, "/tags/"+arn, nil)["Tags"])

	dec := c.rest("StartOutpostDecommission", http.MethodPost, "/outposts/"+opID+"/decommission", map[string]any{})
	fmt.Printf("decommission: %v\n", dec["Status"])
	c.rest("DeleteOutpost", http.MethodDelete, "/outposts/"+opID, nil)
	c.rest("DeleteSite", http.MethodDelete, "/sites/"+siteID, nil)
	fmt.Println("deleted outpost and site")
}

// rest sends one REST JSON operation and exits unless it returns 2xx.
func (c client) rest(op, method, path string, payload any) map[string]any {
	var body []byte
	headers := map[string]string{}
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
		headers["Content-Type"] = "application/json"
	}
	raw := c.send(op, method, signingName, c.endpoint+path, body, headers)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
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
