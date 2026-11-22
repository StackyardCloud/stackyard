// Command route53profiles walks a Route 53 Profiles lifecycle against
// Stackyard: create a VPC and a Profile, attach a private hosted zone and a
// DNS Firewall rule group, associate the Profile with the VPC, tag it, then
// tear everything down.
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
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

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
	account := getenv("AWS_ACCOUNT_ID", "123456789012")
	fmt.Printf("Stackyard Route 53 Profiles client using %s\n", c.endpoint)

	vpcXML := c.send("CreateVpc", "ec2", http.MethodPost, c.endpoint+"/",
		[]byte(url.Values{"Action": {"CreateVpc"}, "Version": {"2016-11-15"}, "CidrBlock": {"10.42.0.0/16"}}.Encode()),
		"application/x-www-form-urlencoded")
	m := regexp.MustCompile(`<vpcId>([^<]+)</vpcId>`).FindSubmatch(vpcXML)
	if m == nil {
		exitf("CreateVpc: no vpcId in %s", vpcXML)
	}
	vpc := string(m[1])
	fmt.Println("VPC:", vpc)

	token := fmt.Sprintf("stackyard-%d", time.Now().UnixNano())
	profile := c.call("CreateProfile", http.MethodPost, "/profile", map[string]any{"Name": "stackyard-profile", "ClientToken": token})["Profile"].(map[string]any)
	id := profile["Id"].(string)
	fmt.Println("Profile:", id, profile["Status"])
	fmt.Println("Status after read:", c.call("GetProfile", http.MethodGet, "/profile/"+id, nil)["Profile"].(map[string]any)["Status"])

	zone := "arn:aws:route53:::hostedzone/Z0STACKYARD42"
	firewall := "arn:aws:route53resolver:" + c.region + ":" + account + ":firewall-rule-group/rslvr-frg-stackyard42"
	for _, r := range []struct{ arn, props string }{{zone, ""}, {firewall, `{"priority":101}`}} {
		body := map[string]any{"ProfileId": id, "ResourceArn": r.arn, "Name": "stackyard"}
		if r.props != "" {
			body["ResourceProperties"] = r.props
		}
		ra := c.call("AssociateResourceToProfile", http.MethodPost, "/profileresourceassociation", body)["ProfileResourceAssociation"].(map[string]any)
		fmt.Println("Resource:", ra["ResourceType"], ra["Id"])
	}

	assoc := c.call("AssociateProfile", http.MethodPost, "/profileassociation", map[string]any{"ProfileId": id, "ResourceId": vpc, "Name": "stackyard-vpc"})["ProfileAssociation"].(map[string]any)
	fmt.Println("Association:", assoc["Id"], assoc["Status"])

	arn := url.PathEscape(profile["Arn"].(string))
	c.call("TagResource", http.MethodPost, "/tags/"+arn, map[string]any{"Tags": map[string]string{"env": "dev"}})
	fmt.Println("Tags:", c.call("ListTagsForResource", http.MethodGet, "/tags/"+arn, nil)["Tags"])

	list := c.call("ListProfileResourceAssociations", http.MethodGet, "/profileresourceassociations/profileid/"+id, nil)["ProfileResourceAssociations"].([]any)
	fmt.Println("Resources on profile:", len(list))

	c.call("DisassociateProfile", http.MethodDelete, "/profileassociation/Profileid/"+id+"/resourceid/"+vpc, nil)
	for _, r := range []string{zone, firewall} {
		c.call("DisassociateResourceFromProfile", http.MethodDelete, "/profileresourceassociation/profileid/"+id+"/resourcearn/"+url.PathEscape(r), nil)
	}
	fmt.Println("Deleted:", c.call("DeleteProfile", http.MethodDelete, "/profile/"+id, nil)["Profile"].(map[string]any)["Status"])
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, "route53profiles", method, c.endpoint+path, body, "application/json")
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, service, method, target string, body []byte, contentType string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", contentType)
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
