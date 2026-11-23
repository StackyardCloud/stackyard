package server

import (
	"fmt"
	"net/http"
	"testing"
)

const r53rTestEndpoint = `{"CreatorRequestId":"%s","Name":"out","Direction":"%s","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001","Ip":"10.0.0.10"},{"SubnetId":"subnet-00000001"}]}`

func TestRoute53ResolverStage1Endpoints(t *testing.T) {
	_, ts := newRoute53ResolverTestServer(t)

	ep := r53rOK(t, ts, "CreateResolverEndpoint", fmt.Sprintf(r53rTestEndpoint, "e1", "OUTBOUND"))["ResolverEndpoint"].(map[string]any)
	id := ep["Id"].(string)
	if ep["Status"] != "CREATING" || ep["HostVPCId"] != "vpc-00000001" || ep["IpAddressCount"] != float64(2) {
		t.Fatalf("created endpoint: %v", ep)
	}
	got := r53rOK(t, ts, "GetResolverEndpoint", `{"ResolverEndpointId":"`+id+`"}`)["ResolverEndpoint"].(map[string]any)
	if got["Status"] != "OPERATIONAL" {
		t.Fatalf("endpoint should settle to OPERATIONAL: %v", got)
	}

	for name, c := range map[string]struct{ body, want string }{
		"duplicate request id": {fmt.Sprintf(r53rTestEndpoint, "e1", "OUTBOUND"), "ResourceExistsException"},
		"ip in use":            {`{"CreatorRequestId":"e2","Direction":"INBOUND","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001","Ip":"10.0.0.10"},{"SubnetId":"subnet-00000001"}]}`, "ResourceExistsException"},
		"ip outside subnet":    {`{"CreatorRequestId":"e3","Direction":"INBOUND","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001","Ip":"10.9.0.5"},{"SubnetId":"subnet-00000001"}]}`, "InvalidParameterException"},
		"reserved ip":          {`{"CreatorRequestId":"e4","Direction":"INBOUND","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001","Ip":"10.0.0.1"},{"SubnetId":"subnet-00000001"}]}`, "InvalidParameterException"},
		"missing subnet":       {`{"CreatorRequestId":"e5","Direction":"INBOUND","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-nope"},{"SubnetId":"subnet-00000001"}]}`, "InvalidParameterException"},
		"missing group":        {`{"CreatorRequestId":"e6","Direction":"INBOUND","SecurityGroupIds":["sg-nope"],"IpAddresses":[{"SubnetId":"subnet-00000001"},{"SubnetId":"subnet-00000001"}]}`, "InvalidParameterException"},
		"one ip":               {`{"CreatorRequestId":"e7","Direction":"INBOUND","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001"}]}`, "InvalidParameterException"},
		"fips outbound":        {`{"CreatorRequestId":"e8","Direction":"OUTBOUND","Protocols":["DoH-FIPS"],"SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001"},{"SubnetId":"subnet-00000001"}]}`, "InvalidParameterException"},
		"all-digit name":       {`{"CreatorRequestId":"e9","Name":"123","Direction":"INBOUND","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001"},{"SubnetId":"subnet-00000001"}]}`, "InvalidParameterException"},
	} {
		if code := r53rFail(t, ts, "CreateResolverEndpoint", c.body); code != c.want {
			t.Fatalf("%s: got %s, want %s", name, code, c.want)
		}
	}

	r53rOK(t, ts, "AssociateResolverEndpointIpAddress", `{"ResolverEndpointId":"`+id+`","IpAddress":{"SubnetId":"subnet-00000001"}}`)
	ips := r53rOK(t, ts, "ListResolverEndpointIpAddresses", `{"ResolverEndpointId":"`+id+`"}`)["IpAddresses"].([]any)
	if len(ips) != 3 {
		t.Fatalf("ips: %v", ips)
	}
	r53rOK(t, ts, "DisassociateResolverEndpointIpAddress", `{"ResolverEndpointId":"`+id+`","IpAddress":{"Ip":"10.0.0.10"}}`)
	last := ips[2].(map[string]any)["IpId"].(string)
	if code := r53rFail(t, ts, "DisassociateResolverEndpointIpAddress", `{"ResolverEndpointId":"`+id+`","IpAddress":{"IpId":"`+last+`"}}`); code != "InvalidRequestException" {
		t.Fatalf("keeping two ips: %s", code)
	}

	list := r53rOK(t, ts, "ListResolverEndpoints", `{"Filters":[{"Name":"Direction","Values":["OUTBOUND"]}]}`)["ResolverEndpoints"].([]any)
	if len(list) != 1 {
		t.Fatalf("filtered endpoints: %v", list)
	}
	if code := r53rFail(t, ts, "ListResolverEndpoints", `{"Filters":[{"Name":"Bogus","Values":["x"]}]}`); code != "InvalidParameterException" {
		t.Fatalf("bad filter: %s", code)
	}
	if code := r53rFail(t, ts, "ListResolverEndpoints", `{"NextToken":"zzz"}`); code != "InvalidNextTokenException" {
		t.Fatalf("bad token: %s", code)
	}
	r53rOK(t, ts, "UpdateResolverEndpoint", `{"ResolverEndpointId":"`+id+`","Name":"renamed","Protocols":["Do53","DoH"]}`)
	r53rOK(t, ts, "DeleteResolverEndpoint", `{"ResolverEndpointId":"`+id+`"}`)
	if code := r53rFail(t, ts, "GetResolverEndpoint", `{"ResolverEndpointId":"`+id+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted endpoint: %s", code)
	}
}

func TestRoute53ResolverStage1RulesAndAssociations(t *testing.T) {
	_, ts := newRoute53ResolverTestServer(t)
	out := r53rOK(t, ts, "CreateResolverEndpoint", fmt.Sprintf(r53rTestEndpoint, "o", "OUTBOUND"))["ResolverEndpoint"].(map[string]any)["Id"].(string)
	in := r53rOK(t, ts, "CreateResolverEndpoint", `{"CreatorRequestId":"i","Direction":"INBOUND","SecurityGroupIds":["sg-00000000"],"IpAddresses":[{"SubnetId":"subnet-00000001"},{"SubnetId":"subnet-00000001"}]}`)["ResolverEndpoint"].(map[string]any)["Id"].(string)

	rule := r53rOK(t, ts, "CreateResolverRule", `{"CreatorRequestId":"r1","Name":"corp","RuleType":"FORWARD","DomainName":"Example.COM","TargetIps":[{"Ip":"192.0.2.10"}],"ResolverEndpointId":"`+out+`"}`)["ResolverRule"].(map[string]any)
	rid := rule["Id"].(string)
	if rule["DomainName"] != "example.com." || rule["TargetIps"].([]any)[0].(map[string]any)["Port"] != float64(53) {
		t.Fatalf("rule: %v", rule)
	}
	for name, body := range map[string]string{
		"inbound endpoint":  `{"CreatorRequestId":"r2","RuleType":"FORWARD","DomainName":"a.com","TargetIps":[{"Ip":"192.0.2.10"}],"ResolverEndpointId":"` + in + `"}`,
		"recursive":         `{"CreatorRequestId":"r3","RuleType":"RECURSIVE","DomainName":"a.com"}`,
		"system targets":    `{"CreatorRequestId":"r4","RuleType":"SYSTEM","DomainName":"a.com","TargetIps":[{"Ip":"192.0.2.10"}]}`,
		"forward no target": `{"CreatorRequestId":"r5","RuleType":"FORWARD","DomainName":"a.com","ResolverEndpointId":"` + out + `"}`,
	} {
		if code := r53rFail(t, ts, "CreateResolverRule", body); code != "InvalidRequestException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	if code := r53rFail(t, ts, "CreateResolverRule", `{"CreatorRequestId":"r6","RuleType":"SYSTEM","DomainName":"bad domain!"}`); code != "InvalidParameterException" {
		t.Fatalf("bad domain: %s", code)
	}

	rules := r53rOK(t, ts, "ListResolverRules", `{}`)["ResolverRules"].([]any)
	if len(rules) != 2 {
		t.Fatalf("rules should include the autodefined internet resolver: %v", rules)
	}
	byDomain := r53rOK(t, ts, "ListResolverRules", `{"Filters":[{"Name":"DomainName","Values":["example.com"]}]}`)["ResolverRules"].([]any)
	if len(byDomain) != 1 {
		t.Fatalf("domain filter: %v", byDomain)
	}
	if code := r53rFail(t, ts, "DeleteResolverRule", `{"ResolverRuleId":"`+r53rAutodefinedID+`"}`); code != "InvalidRequestException" {
		t.Fatalf("autodefined delete: %s", code)
	}

	if code := r53rFail(t, ts, "AssociateResolverRule", `{"ResolverRuleId":"`+rid+`","VPCId":"vpc-nope"}`); code != "InvalidParameterException" {
		t.Fatalf("missing vpc: %s", code)
	}
	assoc := r53rOK(t, ts, "AssociateResolverRule", `{"ResolverRuleId":"`+rid+`","VPCId":"vpc-00000001","Name":"link"}`)["ResolverRuleAssociation"].(map[string]any)
	if assoc["Status"] != "CREATING" {
		t.Fatalf("association: %v", assoc)
	}
	if code := r53rFail(t, ts, "AssociateResolverRule", `{"ResolverRuleId":"`+rid+`","VPCId":"vpc-00000001"}`); code != "ResourceExistsException" {
		t.Fatalf("duplicate association: %s", code)
	}
	got := r53rOK(t, ts, "GetResolverRuleAssociation", `{"ResolverRuleAssociationId":"`+assoc["Id"].(string)+`"}`)["ResolverRuleAssociation"].(map[string]any)
	if got["Status"] != "COMPLETE" {
		t.Fatalf("association should settle: %v", got)
	}
	if code := r53rFail(t, ts, "DeleteResolverRule", `{"ResolverRuleId":"`+rid+`"}`); code != "ResourceInUseException" {
		t.Fatalf("delete associated rule: %s", code)
	}
	if code := r53rFail(t, ts, "DeleteResolverEndpoint", `{"ResolverEndpointId":"`+out+`"}`); code != "InvalidRequestException" {
		t.Fatalf("delete endpoint used by rule: %s", code)
	}
	r53rOK(t, ts, "UpdateResolverRule", `{"ResolverRuleId":"`+rid+`","Config":{"Name":"corp2","TargetIps":[{"Ip":"192.0.2.20","Port":5353}]}}`)

	if code := r53rFail(t, ts, "PutResolverRulePolicy", `{"Arn":"`+rule["Arn"].(string)+`","ResolverRulePolicy":"not json"}`); code != "InvalidPolicyDocument" {
		t.Fatalf("bad policy: %s", code)
	}
	r53rOK(t, ts, "PutResolverRulePolicy", `{"Arn":"`+rule["Arn"].(string)+`","ResolverRulePolicy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)
	if p := r53rOK(t, ts, "GetResolverRulePolicy", `{"Arn":"`+rule["Arn"].(string)+`"}`)["ResolverRulePolicy"]; p == nil {
		t.Fatal("policy not stored")
	}

	if code := r53rFail(t, ts, "TagResource", `{"ResourceArn":"`+rule["Arn"].(string)+`","Tags":[{"Key":"aws:x","Value":"y"}]}`); code != "InvalidTagException" {
		t.Fatalf("aws: tag: %s", code)
	}
	r53rOK(t, ts, "TagResource", `{"ResourceArn":"`+rule["Arn"].(string)+`","Tags":[{"Key":"team","Value":"dns"}]}`)
	if tags := r53rOK(t, ts, "ListTagsForResource", `{"ResourceArn":"`+rule["Arn"].(string)+`"}`)["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}

	r53rOK(t, ts, "DisassociateResolverRule", `{"ResolverRuleId":"`+rid+`","VPCId":"vpc-00000001"}`)
	r53rOK(t, ts, "DeleteResolverRule", `{"ResolverRuleId":"`+rid+`"}`)
	if code := r53rFail(t, ts, "ListTagsForResource", `{"ResourceArn":"`+rule["Arn"].(string)+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("tags of deleted rule: %s", code)
	}
}

func TestRoute53ResolverStage1QueryLoggingAndVPCSettings(t *testing.T) {
	srv, ts := newRoute53ResolverTestServer(t)
	if code := r53rFail(t, ts, "CreateResolverQueryLogConfig", `{"Name":"q","DestinationArn":"arn:aws:s3:::missing-bucket","CreatorRequestId":"q0"}`); code != "InvalidParameterException" {
		t.Fatalf("missing bucket: %s", code)
	}
	if _, err := srv.s3.CreateBucket("dns-logs"); err != nil {
		t.Fatal(err)
	}
	q := r53rOK(t, ts, "CreateResolverQueryLogConfig", `{"Name":"q","DestinationArn":"arn:aws:s3:::dns-logs","CreatorRequestId":"q1"}`)["ResolverQueryLogConfig"].(map[string]any)
	qid := q["Id"].(string)
	r53rOK(t, ts, "CreateResolverQueryLogConfig", `{"Name":"logs","DestinationArn":"arn:aws:logs:us-east-1:123456789012:log-group:dns","CreatorRequestId":"q2"}`)
	if code := r53rFail(t, ts, "CreateResolverQueryLogConfig", `{"Name":"x","DestinationArn":"arn:aws:sqs:us-east-1:123456789012:q","CreatorRequestId":"q3"}`); code != "InvalidParameterException" {
		t.Fatalf("bad destination: %s", code)
	}
	r53rOK(t, ts, "AssociateResolverQueryLogConfig", `{"ResolverQueryLogConfigId":"`+qid+`","ResourceId":"vpc-00000001"}`)
	if code := r53rFail(t, ts, "AssociateResolverQueryLogConfig", `{"ResolverQueryLogConfigId":"`+qid+`","ResourceId":"vpc-00000001"}`); code != "ResourceExistsException" {
		t.Fatalf("duplicate association: %s", code)
	}
	got := r53rOK(t, ts, "GetResolverQueryLogConfig", `{"ResolverQueryLogConfigId":"`+qid+`"}`)["ResolverQueryLogConfig"].(map[string]any)
	if got["Status"] != "CREATED" || got["AssociationCount"] != float64(1) {
		t.Fatalf("config: %v", got)
	}
	if code := r53rFail(t, ts, "DeleteResolverQueryLogConfig", `{"ResolverQueryLogConfigId":"`+qid+`"}`); code != "InvalidRequestException" {
		t.Fatalf("delete associated config: %s", code)
	}
	listed := r53rOK(t, ts, "ListResolverQueryLogConfigs", `{"Filters":[{"Name":"DestinationType","Values":["S3"]}],"SortBy":"Name","SortOrder":"DESCENDING"}`)
	if listed["TotalCount"] != float64(2) || listed["TotalFilteredCount"] != float64(1) {
		t.Fatalf("list counts: %v", listed)
	}
	r53rOK(t, ts, "DisassociateResolverQueryLogConfig", `{"ResolverQueryLogConfigId":"`+qid+`","ResourceId":"vpc-00000001"}`)
	r53rOK(t, ts, "DeleteResolverQueryLogConfig", `{"ResolverQueryLogConfigId":"`+qid+`"}`)

	cfg := r53rOK(t, ts, "GetResolverConfig", `{"ResourceId":"vpc-00000001"}`)["ResolverConfig"].(map[string]any)
	if cfg["AutodefinedReverse"] != "ENABLED" {
		t.Fatalf("default resolver config: %v", cfg)
	}
	up := r53rOK(t, ts, "UpdateResolverConfig", `{"ResourceId":"vpc-00000001","AutodefinedReverseFlag":"DISABLE"}`)["ResolverConfig"].(map[string]any)
	if up["AutodefinedReverse"] != "DISABLING" {
		t.Fatalf("update: %v", up)
	}
	if v := r53rOK(t, ts, "GetResolverConfig", `{"ResourceId":"vpc-00000001"}`)["ResolverConfig"].(map[string]any)["AutodefinedReverse"]; v != "DISABLED" {
		t.Fatalf("settled: %v", v)
	}
	if code := r53rFail(t, ts, "GetResolverConfig", `{"ResourceId":"vpc-nope"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing vpc: %s", code)
	}
	r53rOK(t, ts, "UpdateResolverDnssecConfig", `{"ResourceId":"vpc-00000001","Validation":"ENABLE"}`)
	if v := r53rOK(t, ts, "GetResolverDnssecConfig", `{"ResourceId":"vpc-00000001"}`)["ResolverDNSSECConfig"].(map[string]any)["ValidationStatus"]; v != "ENABLED" {
		t.Fatalf("dnssec: %v", v)
	}
	if n := len(r53rOK(t, ts, "ListResolverConfigs", `{}`)["ResolverConfigs"].([]any)); n == 0 {
		t.Fatal("no resolver configs listed")
	}

	o := r53rOK(t, ts, "CreateOutpostResolver", `{"CreatorRequestId":"or1","Name":"edge","PreferredInstanceType":"m5.large","OutpostArn":"arn:aws:outposts:us-east-1:123456789012:outpost/op-0123456789abcdef0"}`)["OutpostResolver"].(map[string]any)
	if o["InstanceCount"] != float64(4) {
		t.Fatalf("outpost resolver: %v", o)
	}
	if code := r53rFail(t, ts, "UpdateOutpostResolver", `{"Id":"`+o["Id"].(string)+`","InstanceCount":2}`); code != "ValidationException" {
		t.Fatalf("instance count: %s", code)
	}
	if code := r53rFail(t, ts, "CreateOutpostResolver", `{"CreatorRequestId":"or2","Name":"x","PreferredInstanceType":"m5.large","OutpostArn":"arn:aws:outposts:us-east-1:123456789012:outpost/bad"}`); code != "ValidationException" {
		t.Fatalf("outpost arn: %s", code)
	}
}

func TestRoute53ResolverStage1ResponsesDontAliasStore(t *testing.T) {
	srv, ts := newRoute53ResolverTestServer(t)
	r53rOK(t, ts, "CreateResolverRule", `{"CreatorRequestId":"alias","RuleType":"SYSTEM","DomainName":"a.example"}`)
	out, err := srv.route53resolver.handle("ListResolverRules", "ListResolverRulesResponse", map[string]any{})
	if err != nil {
		t.Fatal(err.msg)
	}
	for _, r := range out.(map[string]any)["ResolverRules"].([]any) {
		r.(map[string]any)["Name"] = "mutated"
	}
	for _, r := range srv.route53resolver.rules {
		if r.rec["Name"] == "mutated" {
			t.Fatalf("a response map aliased the stored rule")
		}
	}
	if status, _ := r53rRequest(t, ts, "GetResolverRule", `{"ResolverRuleId":"nope"}`); status != http.StatusBadRequest {
		t.Fatalf("missing rule: %d", status)
	}
}
