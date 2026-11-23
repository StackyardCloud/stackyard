// Command networkmanager walks an AWS Cloud WAN lifecycle against Stackyard:
// create a global network and a core network with a policy, execute the
// policy's change set, attach a VPC, read routes for its segment, then tear
// everything down.
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

const policy = `{"version":"2021.12",
 "core-network-configuration":{"asn-ranges":["64512-64555"],"edge-locations":[{"location":"us-east-1"}]},
 "segments":[{"name":"prod"}],
 "attachment-policies":[{"rule-number":100,"conditions":[{"type":"any"}],"action":{"association-method":"constant","segment":"prod"}}]}`

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Network Manager client using %s\n", c.endpoint)

	vpc := c.ec2("CreateVpc", `<vpcId>([^<]+)</vpcId>`, url.Values{"CidrBlock": {"10.42.0.0/16"}})
	subnet := c.ec2("CreateSubnet", `<subnetId>([^<]+)</subnetId>`, url.Values{"VpcId": {vpc}, "CidrBlock": {"10.42.1.0/24"}, "AvailabilityZone": {"us-east-1a"}})
	fmt.Println("VPC:", vpc, "subnet:", subnet)

	gn := c.call("CreateGlobalNetwork", http.MethodPost, "/global-networks", map[string]any{"Description": "stackyard"})["GlobalNetwork"].(map[string]any)
	gnID := gn["GlobalNetworkId"].(string)
	fmt.Println("Global network:", gnID, gn["State"])

	core := c.call("CreateCoreNetwork", http.MethodPost, "/core-networks", map[string]any{"GlobalNetworkId": gnID, "PolicyDocument": policy})["CoreNetwork"].(map[string]any)
	coreID := core["CoreNetworkId"].(string)
	fmt.Println("Core network:", coreID, core["State"])

	ver := c.call("GetCoreNetworkPolicy", http.MethodGet, "/core-networks/"+coreID+"/core-network-policy?alias=LATEST", nil)["CoreNetworkPolicy"].(map[string]any)
	fmt.Println("Policy version 1:", ver["ChangeSetState"])
	changes := c.call("GetCoreNetworkChangeSet", http.MethodGet, "/core-networks/"+coreID+"/core-network-change-sets/1", nil)["CoreNetworkChanges"].([]any)
	fmt.Println("Changes:", len(changes))
	c.call("ExecuteCoreNetworkChangeSet", http.MethodPost, "/core-networks/"+coreID+"/core-network-change-sets/1/execute", nil)

	arn := func(kind, id string) string { return "arn:aws:ec2:" + c.region + ":123456789012:" + kind + "/" + id }
	att := c.call("CreateVpcAttachment", http.MethodPost, "/vpc-attachments", map[string]any{
		"CoreNetworkId": coreID, "VpcArn": arn("vpc", vpc), "SubnetArns": []string{arn("subnet", subnet)},
	})["VpcAttachment"].(map[string]any)["Attachment"].(map[string]any)
	attID := att["AttachmentId"].(string)
	fmt.Println("Attachment:", attID, att["SegmentName"], att["State"])
	got := c.call("GetVpcAttachment", http.MethodGet, "/vpc-attachments/"+attID, nil)["VpcAttachment"].(map[string]any)["Attachment"].(map[string]any)
	fmt.Println("Attachment after read:", got["State"])

	routes := c.call("GetNetworkRoutes", http.MethodPost, "/global-networks/"+gnID+"/network-routes", map[string]any{
		"RouteTableIdentifier": map[string]any{"CoreNetworkSegmentEdge": map[string]any{"CoreNetworkId": coreID, "SegmentName": "prod", "EdgeLocation": "us-east-1"}},
	})["NetworkRoutes"].([]any)
	fmt.Println("Routes on prod/us-east-1:", len(routes))

	c.call("DeleteAttachment", http.MethodDelete, "/attachments/"+attID, nil)
	c.call("DeleteCoreNetwork", http.MethodDelete, "/core-networks/"+coreID, nil)
	fmt.Println("Deleted:", c.call("DeleteGlobalNetwork", http.MethodDelete, "/global-networks/"+gnID, nil)["GlobalNetwork"].(map[string]any)["State"])
}

// ec2 runs an EC2 Query action and returns the first match of pattern.
func (c client) ec2(action, pattern string, params url.Values) string {
	params.Set("Action", action)
	params.Set("Version", "2016-11-15")
	raw := c.send(action, "ec2", http.MethodPost, c.endpoint+"/", []byte(params.Encode()), "application/x-www-form-urlencoded")
	m := regexp.MustCompile(pattern).FindSubmatch(raw)
	if m == nil {
		exitf("%s: no match in %s", action, raw)
	}
	return string(m[1])
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, "networkmanager", method, c.endpoint+path, body, "application/json")
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx. Extra
// headers come in name/value pairs.
func (c client) send(op, service, method, target string, body []byte, contentType string, headers ...string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", contentType)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
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
