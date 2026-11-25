// Command mailmanager runs Amazon SES Mail Manager against Stackyard: create
// an archive, point an SESv2 configuration set's archiving options at it,
// send two emails, search the archive and read a message back, import
// address list members through a pre-signed upload, and wire a traffic
// policy and rule set to an SMTP ingress point.
//
// The SDK has no Mail Manager module in this repository's go.mod, so
// requests are signed by hand.
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

const (
	signingName  = "ses"
	targetPrefix = "MailManagerSvc."
	contentType  = "application/x-amz-json-1.0"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard Mail Manager client using %s\n", c.endpoint)
	run := time.Now().UnixNano()

	archive := c.call("CreateArchive", map[string]any{
		"ArchiveName": fmt.Sprintf("support-%d", run), "Retention": map[string]any{"RetentionPeriod": "ONE_YEAR"},
	})["ArchiveId"].(string)
	archiveARN := c.call("GetArchive", map[string]any{"ArchiveId": archive})["ArchiveArn"].(string)
	fmt.Printf("archive %s\n", archive)

	// SESv2 archives what it sends through this configuration set.
	configSet := fmt.Sprintf("archived-%d", run)
	c.rest("CreateEmailIdentity", http.MethodPost, "/v2/email/identities", map[string]any{"EmailIdentity": "example.com"})
	c.rest("CreateConfigurationSet", http.MethodPost, "/v2/email/configuration-sets", map[string]any{"ConfigurationSetName": configSet})
	c.rest("PutConfigurationSetArchivingOptions", http.MethodPut, "/v2/email/configuration-sets/"+configSet+"/archiving-options", map[string]any{"ArchiveArn": archiveARN})
	for _, subject := range []string{"Ticket #1042 opened", "Ticket #1042 resolved"} {
		c.rest("SendEmail", http.MethodPost, "/v2/email/outbound-emails", map[string]any{
			"FromEmailAddress": "support@example.com", "ConfigurationSetName": configSet,
			"Destination": map[string]any{"ToAddresses": []string{"customer@example.org"}},
			"Content": map[string]any{"Simple": map[string]any{
				"Subject": map[string]any{"Data": subject}, "Body": map[string]any{"Text": map[string]any{"Data": "Thanks for contacting support."}},
			}},
		})
	}

	search := c.call("StartArchiveSearch", map[string]any{
		"ArchiveId": archive, "MaxResults": 10,
		"FromTimestamp": time.Now().Add(-time.Hour).Unix(), "ToTimestamp": time.Now().Add(time.Hour).Unix(),
		"Filters": map[string]any{"Include": []any{map[string]any{"StringExpression": map[string]any{
			"Evaluate": map[string]any{"Attribute": "SUBJECT"}, "Operator": "CONTAINS", "Values": []string{"resolved"},
		}}}},
	})["SearchId"].(string)
	rows := c.call("GetArchiveSearchResults", map[string]any{"SearchId": search})["Rows"].([]any)
	if len(rows) != 1 {
		exitf("expected 1 search result, got %d", len(rows))
	}
	row := rows[0].(map[string]any)
	body := c.call("GetArchiveMessageContent", map[string]any{"ArchivedMessageId": row["ArchivedMessageId"]})["Body"].(map[string]any)
	fmt.Printf("found %q from %s: %q\n", row["Subject"], row["From"], body["Text"])

	// Address list members from a CSV upload to the import job's pre-signed URL.
	list := c.call("CreateAddressList", map[string]any{"AddressListName": fmt.Sprintf("vip-%d", run)})["AddressListId"].(string)
	job := c.call("CreateAddressListImportJob", map[string]any{
		"AddressListId": list, "Name": "vip-import", "ImportDataFormat": map[string]any{"ImportDataType": "CSV"},
	})
	upload(job["PreSignedUrl"].(string), "address\nceo@example.org\ncfo@example.org\n")
	c.call("StartAddressListImportJob", map[string]any{"JobId": job["JobId"]})
	status := c.call("GetAddressListImportJob", map[string]any{"JobId": job["JobId"]})
	fmt.Printf("import job %v: %v addresses imported\n", status["Status"], status["ImportedItemsCount"])
	members := c.call("ListMembersOfAddressList", map[string]any{"AddressListId": list})["Addresses"].([]any)
	fmt.Printf("address list %s has %d members\n", list, len(members))

	// A traffic policy and a rule set that archives VIP mail, behind an ingress point.
	policy := c.call("CreateTrafficPolicy", map[string]any{
		"TrafficPolicyName": fmt.Sprintf("allow-%d", run), "DefaultAction": "DENY",
		"PolicyStatements": []any{map[string]any{"Action": "ALLOW", "Conditions": []any{map[string]any{"IpExpression": map[string]any{
			"Evaluate": map[string]any{"Attribute": "SENDER_IP"}, "Operator": "CIDR_MATCHES", "Values": []string{"10.0.0.0/8"},
		}}}}},
	})["TrafficPolicyId"].(string)
	listARN := c.call("GetAddressList", map[string]any{"AddressListId": list})["AddressListArn"]
	ruleSet := c.call("CreateRuleSet", map[string]any{
		"RuleSetName": fmt.Sprintf("vip-%d", run),
		"Rules": []any{map[string]any{
			"Name": "archive-vip",
			"Conditions": []any{map[string]any{"BooleanExpression": map[string]any{
				"Evaluate": map[string]any{"IsInAddressList": map[string]any{"Attribute": "RECIPIENT", "AddressLists": []any{listARN}}}, "Operator": "IS_TRUE",
			}}},
			"Actions": []any{map[string]any{"Archive": map[string]any{"TargetArchive": archive}}},
		}},
	})["RuleSetId"].(string)
	ingress := c.call("CreateIngressPoint", map[string]any{
		"IngressPointName": fmt.Sprintf("smtp-%d", run), "Type": "OPEN", "RuleSetId": ruleSet, "TrafficPolicyId": policy,
	})["IngressPointId"].(string)
	point := c.call("GetIngressPoint", map[string]any{"IngressPointId": ingress})
	fmt.Printf("ingress point %s is %v at %v\n", ingress, point["Status"], point["ARecord"])

	c.call("DeleteIngressPoint", map[string]any{"IngressPointId": ingress})
	c.call("DeleteRuleSet", map[string]any{"RuleSetId": ruleSet})
	c.call("DeleteTrafficPolicy", map[string]any{"TrafficPolicyId": policy})
	c.call("DeleteAddressList", map[string]any{"AddressListId": list})
	c.call("DeleteArchive", map[string]any{"ArchiveId": archive})
	fmt.Println("archive state:", c.call("GetArchive", map[string]any{"ArchiveId": archive})["ArchiveState"])
}

// upload PUTs data to a pre-signed URL (no signature needed).
func upload(url, data string) {
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(data))
	if err != nil {
		exitf("upload: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("upload: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		exitf("upload: HTTP %d", resp.StatusCode)
	}
}

// call sends one awsJson operation and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	return decode(op, c.send(op, http.MethodPost, signingName, c.endpoint+"/", body, map[string]string{
		"Content-Type": contentType, "X-Amz-Target": targetPrefix + op,
	}))
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

// rest sends one REST JSON operation and exits unless it returns 2xx.
func (c client) rest(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	return decode(op, c.send(op, method, signingName, c.endpoint+path, body, map[string]string{"Content-Type": "application/json"}))
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
