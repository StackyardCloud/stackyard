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

// A Route 53 Resolver walkthrough: an outbound endpoint in the default VPC,
// a forwarding rule for an on-premises domain associated with the VPC, and a
// DNS Firewall rule group that blocks a domain list, then cleanup.
func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	vpc := getenv("STACKYARD_VPC_ID", "vpc-00000001")
	subnet := getenv("STACKYARD_SUBNET_ID", "subnet-00000001")
	group := getenv("STACKYARD_SECURITY_GROUP_ID", "sg-00000000")
	run := fmt.Sprint(time.Now().UnixNano())
	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	c := client{endpoint: endpoint, region: region, creds: creds}
	fmt.Printf("Stackyard Route 53 Resolver client using %s\n", endpoint)

	var ep struct{ ResolverEndpoint struct{ Id, Status string } }
	c.call(ctx, "CreateResolverEndpoint", map[string]any{
		"CreatorRequestId": "ep-" + run, "Name": "to-datacenter", "Direction": "OUTBOUND",
		"SecurityGroupIds": []string{group},
		"IpAddresses":      []map[string]string{{"SubnetId": subnet}, {"SubnetId": subnet}},
	}, &ep)
	var got struct{ ResolverEndpoint struct{ Status string } }
	c.call(ctx, "GetResolverEndpoint", map[string]any{"ResolverEndpointId": ep.ResolverEndpoint.Id}, &got)
	if got.ResolverEndpoint.Status != "OPERATIONAL" {
		exitf("expected an OPERATIONAL endpoint, got %s", got.ResolverEndpoint.Status)
	}

	var rule struct {
		ResolverRule struct{ Id, DomainName string }
	}
	c.call(ctx, "CreateResolverRule", map[string]any{
		"CreatorRequestId": "rule-" + run, "Name": "corp", "RuleType": "FORWARD", "DomainName": "corp-" + run + ".example",
		"TargetIps": []map[string]any{{"Ip": "192.0.2.53", "Port": 53}}, "ResolverEndpointId": ep.ResolverEndpoint.Id,
	}, &rule)
	logf("forwarding %s through %s", rule.ResolverRule.DomainName, ep.ResolverEndpoint.Id)
	c.call(ctx, "AssociateResolverRule", map[string]any{"ResolverRuleId": rule.ResolverRule.Id, "VPCId": vpc}, nil)

	var list struct{ FirewallDomainList struct{ Id string } }
	c.call(ctx, "CreateFirewallDomainList", map[string]any{"CreatorRequestId": "fdl-" + run, "Name": "blocked"}, &list)
	c.call(ctx, "UpdateFirewallDomains", map[string]any{"FirewallDomainListId": list.FirewallDomainList.Id, "Operation": "ADD", "Domains": []string{"ads.example", "*.tracker.example"}}, nil)
	var rg struct{ FirewallRuleGroup struct{ Id string } }
	c.call(ctx, "CreateFirewallRuleGroup", map[string]any{"CreatorRequestId": "frg-" + run, "Name": "baseline"}, &rg)
	c.call(ctx, "CreateFirewallRule", map[string]any{
		"CreatorRequestId": "fr-" + run, "FirewallRuleGroupId": rg.FirewallRuleGroup.Id, "FirewallDomainListId": list.FirewallDomainList.Id,
		"Priority": 100, "Action": "BLOCK", "BlockResponse": "NXDOMAIN", "Name": "block-ads",
	}, nil)
	var assoc struct{ FirewallRuleGroupAssociation struct{ Id string } }
	c.call(ctx, "AssociateFirewallRuleGroup", map[string]any{
		"CreatorRequestId": "fa-" + run, "FirewallRuleGroupId": rg.FirewallRuleGroup.Id, "VpcId": vpc, "Priority": 101, "Name": "baseline",
	}, &assoc)

	var domains struct{ Domains []string }
	c.call(ctx, "ListFirewallDomains", map[string]any{"FirewallDomainListId": list.FirewallDomainList.Id}, &domains)
	if len(domains.Domains) != 2 {
		exitf("expected two blocked domains, got %v", domains.Domains)
	}

	c.call(ctx, "DisassociateFirewallRuleGroup", map[string]any{"FirewallRuleGroupAssociationId": assoc.FirewallRuleGroupAssociation.Id}, nil)
	c.call(ctx, "DeleteFirewallRuleGroup", map[string]any{"FirewallRuleGroupId": rg.FirewallRuleGroup.Id}, nil)
	c.call(ctx, "DeleteFirewallDomainList", map[string]any{"FirewallDomainListId": list.FirewallDomainList.Id}, nil)
	c.call(ctx, "DisassociateResolverRule", map[string]any{"ResolverRuleId": rule.ResolverRule.Id, "VPCId": vpc}, nil)
	c.call(ctx, "DeleteResolverRule", map[string]any{"ResolverRuleId": rule.ResolverRule.Id}, nil)
	c.call(ctx, "DeleteResolverEndpoint", map[string]any{"ResolverEndpointId": ep.ResolverEndpoint.Id}, nil)
	logf("Route 53 Resolver walkthrough complete")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

// call sends one awsJson1_1 request signed for "route53resolver", exits on a non-2xx
// reply, and decodes into out.
func (c client) call(ctx context.Context, action string, payload, out any) {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Route53Resolver."+action)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "route53resolver", c.region, time.Now()); err != nil {
		exitf("%s: %v", action, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		exitf("expected %s to return 2xx, got %d: %s", action, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	logf("%s returned %d", action, resp.StatusCode)
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			exitf("%s: decode %s: %v", action, raw, err)
		}
	}
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
