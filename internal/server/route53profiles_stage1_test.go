package server

import (
	"fmt"
	"net/url"
	"testing"
)

func r53pTestVPC(t *testing.T, srv *Server, cidr string) string {
	t.Helper()
	vpc, err := srv.ec2.CreateVpc(cidr, nil)
	if err != nil {
		t.Fatal(err)
	}
	return vpc.ID
}

func TestRoute53ProfilesStage1ProfilesAndVPCs(t *testing.T) {
	srv, ts := newRoute53ProfilesTestServer(t)
	created := r53pCall(t, ts, 200, "POST", "/profile", `{"Name":"dev","ClientToken":"tok-1","Tags":[{"Key":"env","Value":"dev"}]}`)["Profile"].(map[string]any)
	id := created["Id"].(string)
	if created["Status"] != "CREATING" || created["ShareStatus"] != "NOT_SHARED" {
		t.Fatalf("created profile: %v", created)
	}
	// Same ClientToken returns the same profile; a different name is a conflict.
	if again := r53pCall(t, ts, 200, "POST", "/profile", `{"Name":"dev","ClientToken":"tok-1"}`)["Profile"].(map[string]any); again["Id"] != id {
		t.Fatalf("idempotent create returned %v", again["Id"])
	}
	if code := r53pErrCode(t, ts, "POST", "/profile", `{"Name":"other","ClientToken":"tok-1"}`); code != "ConflictException" {
		t.Fatalf("reused token: %s", code)
	}
	if code := r53pErrCode(t, ts, "POST", "/profile", `{"Name":"dev","ClientToken":"t","Bogus":1}`); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
	if got := r53pCall(t, ts, 200, "GET", "/profile/"+id, "")["Profile"].(map[string]any); got["Status"] != "COMPLETE" {
		t.Fatalf("profile should settle to COMPLETE: %v", got["Status"])
	}
	tags := r53pCall(t, ts, 200, "GET", "/tags/"+url.PathEscape(r53pProfileARN(id)), "")["Tags"].(map[string]any)
	if tags["env"] != "dev" {
		t.Fatalf("create-time tags: %v", tags)
	}

	vpc := r53pTestVPC(t, srv, "10.20.0.0/16")
	if code := r53pErrCode(t, ts, "POST", "/profileassociation", `{"ProfileId":"`+id+`","ResourceId":"vpc-missing","Name":"a"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing VPC: %s", code)
	}
	if code := r53pErrCode(t, ts, "POST", "/profileassociation", `{"ProfileId":"`+id+`","ResourceId":"`+vpc+`","Name":"a","Priority":50}`); code != "ValidationException" {
		t.Fatalf("priority below 100: %s", code)
	}
	assoc := r53pCall(t, ts, 200, "POST", "/profileassociation", `{"ProfileId":"`+id+`","ResourceId":"`+vpc+`","Name":"a","Priority":200}`)["ProfileAssociation"].(map[string]any)
	if code := r53pErrCode(t, ts, "POST", "/profileassociation", `{"ProfileId":"`+id+`","ResourceId":"`+vpc+`","Name":"a"}`); code != "ResourceExistsException" {
		t.Fatalf("same pair twice: %s", code)
	}
	other := r53pCall(t, ts, 200, "POST", "/profile", `{"Name":"prod","ClientToken":"tok-2"}`)["Profile"].(map[string]any)["Id"].(string)
	if code := r53pErrCode(t, ts, "POST", "/profileassociation", `{"ProfileId":"`+other+`","ResourceId":"`+vpc+`","Name":"b"}`); code != "ConflictException" {
		t.Fatalf("second profile on one VPC: %s", code)
	}
	// Updates wait until the association settles.
	if code := r53pErrCode(t, ts, "PATCH", "/profileassociation/"+assoc["Id"].(string), `{"Priority":300}`); code != "ConflictException" {
		t.Fatalf("update while CREATING: %s", code)
	}
	r53pCall(t, ts, 200, "GET", "/profileassociation/"+assoc["Id"].(string), "")
	if up := r53pCall(t, ts, 200, "PATCH", "/profileassociation/"+assoc["Id"].(string), `{"Priority":300}`)["ProfileAssociation"].(map[string]any); up["Priority"] != float64(300) || up["Status"] != "UPDATING" {
		t.Fatalf("update: %v", up)
	}
	list := r53pCall(t, ts, 200, "GET", "/profileassociations?profileId="+id, "")["ProfileAssociations"].([]any)
	if len(list) != 1 {
		t.Fatalf("associations for profile: %v", list)
	}

	if code := r53pErrCode(t, ts, "DELETE", "/profile/"+id, ""); code != "ConflictException" {
		t.Fatalf("delete with VPC association: %s", code)
	}
	r53pCall(t, ts, 200, "DELETE", "/profileassociation/Profileid/"+id+"/resourceid/"+vpc, "")
	if deleted := r53pCall(t, ts, 200, "DELETE", "/profile/"+id, "")["Profile"].(map[string]any); deleted["Status"] != "DELETED" {
		t.Fatalf("delete: %v", deleted)
	}
	if code := r53pErrCode(t, ts, "GET", "/profile/"+id, ""); code != "ResourceNotFoundException" {
		t.Fatalf("deleted profile: %s", code)
	}
}

func TestRoute53ProfilesStage1ResourceAssociations(t *testing.T) {
	_, ts := newRoute53ProfilesTestServer(t)
	id := r53pCall(t, ts, 200, "POST", "/profile", `{"Name":"res","ClientToken":"tok"}`)["Profile"].(map[string]any)["Id"].(string)
	zone := "arn:aws:route53:::hostedzone/Z0123456789ABC"
	// Resolver resources must exist: a made-up rule group is refused, a real one
	// (created through Route 53 Resolver on the same server) is accepted.
	missing := "arn:aws:route53resolver:us-east-1:123456789012:firewall-rule-group/rslvr-frg-0123abcd"
	fw := r53rOK(t, ts, "CreateFirewallRuleGroup", `{"Name":"profiles-fw","CreatorRequestId":"profiles-fw"}`)["FirewallRuleGroup"].(map[string]any)["Arn"].(string)
	assoc := func(arn, props string) string {
		return fmt.Sprintf(`{"ProfileId":%q,"ResourceArn":%q,"Name":"r","ResourceProperties":%q}`, id, arn, props)
	}
	if code := r53pErrCode(t, ts, "POST", "/profileresourceassociation", assoc("arn:aws:s3:::bucket", "")); code != "InvalidParameterException" {
		t.Fatalf("unsupported resource: %s", code)
	}
	if code := r53pErrCode(t, ts, "POST", "/profileresourceassociation", assoc(fw, "")); code != "InvalidParameterException" {
		t.Fatalf("firewall group without priority: %s", code)
	}
	if code := r53pErrCode(t, ts, "POST", "/profileresourceassociation", assoc(fw, `{"priority":99}`)); code != "InvalidParameterException" {
		t.Fatalf("priority out of range: %s", code)
	}
	if code := r53pErrCode(t, ts, "POST", "/profileresourceassociation", assoc(missing, `{"priority":150}`)); code != "ResourceNotFoundException" {
		t.Fatalf("missing firewall rule group: %s", code)
	}
	ra := r53pCall(t, ts, 200, "POST", "/profileresourceassociation", assoc(fw, `{"priority":150}`))["ProfileResourceAssociation"].(map[string]any)
	if ra["ResourceType"] != "FIREWALL_RULE_GROUP" {
		t.Fatalf("resource type: %v", ra)
	}
	r53pCall(t, ts, 200, "POST", "/profileresourceassociation", assoc(zone, ""))
	if code := r53pErrCode(t, ts, "POST", "/profileresourceassociation", assoc(zone, "")); code != "ConflictException" {
		t.Fatalf("duplicate resource: %s", code)
	}

	// Filters and paging.
	page := r53pCall(t, ts, 200, "GET", "/profileresourceassociations/profileid/"+id+"?maxResults=1", "")
	if len(page["ProfileResourceAssociations"].([]any)) != 1 || page["NextToken"] == nil {
		t.Fatalf("first page: %v", page)
	}
	rest := r53pCall(t, ts, 200, "GET", "/profileresourceassociations/profileid/"+id+"?maxResults=1&nextToken="+page["NextToken"].(string), "")
	if len(rest["ProfileResourceAssociations"].([]any)) != 1 || rest["NextToken"] != nil {
		t.Fatalf("second page: %v", rest)
	}
	if code := r53pErrCode(t, ts, "GET", "/profileresourceassociations/profileid/"+id+"?nextToken=bogus", ""); code != "InvalidNextTokenException" {
		t.Fatalf("bad token: %s", code)
	}
	byType := r53pCall(t, ts, 200, "GET", "/profileresourceassociations/profileid/"+id+"?resourceType=PRIVATE_HOSTED_ZONE", "")["ProfileResourceAssociations"].([]any)
	if len(byType) != 1 {
		t.Fatalf("type filter: %v", byType)
	}

	raID := ra["Id"].(string)
	r53pCall(t, ts, 200, "GET", "/profileresourceassociation/"+raID, "")
	if code := r53pErrCode(t, ts, "PATCH", "/profileresourceassociation/"+raID, `{"ResourceProperties":"{\"priority\":20000}"}`); code != "InvalidParameterException" {
		t.Fatalf("bad update priority: %s", code)
	}
	up := r53pCall(t, ts, 200, "PATCH", "/profileresourceassociation/"+raID, `{"ResourceProperties":"{\"priority\":300}"}`)["ProfileResourceAssociation"].(map[string]any)
	if up["ResourceProperties"] != `{"priority":300}` {
		t.Fatalf("update: %v", up)
	}
	// The ARN label arrives percent-encoded.
	r53pCall(t, ts, 200, "DELETE", "/profileresourceassociation/profileid/"+id+"/resourcearn/"+url.PathEscape(fw), "")
	if code := r53pErrCode(t, ts, "GET", "/profileresourceassociation/"+raID, ""); code != "ResourceNotFoundException" {
		t.Fatalf("removed association: %s", code)
	}
}

func TestRoute53ProfilesStage1Tags(t *testing.T) {
	_, ts := newRoute53ProfilesTestServer(t)
	id := r53pCall(t, ts, 200, "POST", "/profile", `{"Name":"tags","ClientToken":"tok"}`)["Profile"].(map[string]any)["Id"].(string)
	arn := url.PathEscape(r53pProfileARN(id))
	r53pCall(t, ts, 204, "POST", "/tags/"+arn, `{"Tags":{"a":"1","b":"2"}}`)
	if code := r53pErrCode(t, ts, "POST", "/tags/"+arn, `{"Tags":{"aws:x":"1"}}`); code != "ValidationException" {
		t.Fatalf("aws: prefix: %s", code)
	}
	r53pCall(t, ts, 204, "DELETE", "/tags/"+arn+"?tagKeys=a", "")
	if tags := r53pCall(t, ts, 200, "GET", "/tags/"+arn, "")["Tags"].(map[string]any); len(tags) != 1 || tags["b"] != "2" {
		t.Fatalf("tags: %v", tags)
	}
	if code := r53pErrCode(t, ts, "GET", "/tags/"+url.PathEscape(r53pProfileARN("rp-missing")), ""); code != "ResourceNotFoundException" {
		t.Fatalf("unknown ARN: %s", code)
	}
}
