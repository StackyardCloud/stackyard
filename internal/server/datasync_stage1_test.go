package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const dsyTestRole = `"S3Config":{"BucketAccessRoleArn":"arn:aws:iam::123456789012:role/ds"}`

func dsyTestAgent(t *testing.T, call func(op, body string) map[string]any, key string) string {
	t.Helper()
	return call("CreateAgent", `{"ActivationKey":"`+key+`","AgentName":"edge"}`)["AgentArn"].(string)
}

func TestDataSyncStage1AgentsAndLocations(t *testing.T) {
	srv, ts := newDataSyncTestServer(t)
	call := func(op, body string) map[string]any { return dsyOK(t, ts, op, body) }

	// Agents: activation keys are single-use and must look like one.
	if msg := dsyFail(t, ts, "CreateAgent", `{"ActivationKey":"nope"}`); !strings.Contains(msg, "activation key") {
		t.Fatalf("bad key: %s", msg)
	}
	agent := dsyTestAgent(t, call, "AAAAA-BBBBB-CCCCC-DDDDD-EEEEE")
	if msg := dsyFail(t, ts, "CreateAgent", `{"ActivationKey":"AAAAA-BBBBB-CCCCC-DDDDD-EEEEE"}`); !strings.Contains(msg, "already been used") {
		t.Fatalf("reused key: %s", msg)
	}
	if got := call("DescribeAgent", `{"AgentArn":"`+agent+`"}`); got["Status"] != "ONLINE" || got["EndpointType"] != "PUBLIC" || got["Name"] != "edge" {
		t.Fatalf("describe agent: %v", got)
	}

	// S3 locations need an existing bucket and a role.
	if msg := dsyFail(t, ts, "CreateLocationS3", `{"S3BucketArn":"arn:aws:s3:::missing-bucket",`+dsyTestRole+`}`); !strings.Contains(msg, "does not exist") {
		t.Fatalf("missing bucket: %s", msg)
	}
	if _, err := srv.s3.CreateBucket("dsy-bucket"); err != nil {
		t.Fatal(err)
	}
	if msg := dsyFail(t, ts, "CreateLocationS3", `{"S3BucketArn":"arn:aws:s3:::dsy-bucket","S3Config":{"BucketAccessRoleArn":"nope"}}`); !strings.Contains(msg, "IAM role") {
		t.Fatalf("bad role: %s", msg)
	}
	s3loc := call("CreateLocationS3", `{"S3BucketArn":"arn:aws:s3:::dsy-bucket","Subdirectory":"data",`+dsyTestRole+`}`)["LocationArn"].(string)
	if got := call("DescribeLocationS3", `{"LocationArn":"`+s3loc+`"}`); got["LocationUri"] != "s3://dsy-bucket/data/" || got["S3StorageClass"] != "STANDARD" {
		t.Fatalf("describe s3: %v", got)
	}
	// Describing it as the wrong type is refused.
	if msg := dsyFail(t, ts, "DescribeLocationNfs", `{"LocationArn":"`+s3loc+`"}`); !strings.Contains(msg, "S3 location") {
		t.Fatalf("type mismatch: %s", msg)
	}

	// On-premises locations need agents that exist.
	if msg := dsyFail(t, ts, "CreateLocationNfs", `{"ServerHostname":"nfs.local","Subdirectory":"/x","OnPremConfig":{"AgentArns":["arn:aws:datasync:us-east-1:123456789012:agent/agent-0123456789abcdef0"]}}`); !strings.Contains(msg, "does not exist") {
		t.Fatalf("missing agent: %s", msg)
	}
	nfs := call("CreateLocationNfs", `{"ServerHostname":"nfs.local","Subdirectory":"/exports","OnPremConfig":{"AgentArns":["`+agent+`"]}}`)["LocationArn"].(string)
	call("UpdateLocationNfs", `{"LocationArn":"`+nfs+`","Subdirectory":"/moved"}`)
	if got := call("DescribeLocationNfs", `{"LocationArn":"`+nfs+`"}`); got["LocationUri"] != "nfs://nfs.local/moved/" {
		t.Fatalf("updated nfs: %v", got)
	}

	// Secrets go in but never come back out.
	smb := call("CreateLocationSmb", `{"ServerHostname":"smb.local","Subdirectory":"/share","User":"admin","Password":"hunter2","AgentArns":["`+agent+`"]}`)["LocationArn"].(string)
	if got := call("DescribeLocationSmb", `{"LocationArn":"`+smb+`"}`); got["Password"] != nil || got["User"] != "admin" {
		t.Fatalf("describe smb leaked or lost fields: %v", got)
	}
	if msg := dsyFail(t, ts, "CreateLocationSmb", `{"ServerHostname":"smb.local","Subdirectory":"/s","AgentArns":["`+agent+`"]}`); !strings.Contains(msg, "User") {
		t.Fatalf("smb without user: %s", msg)
	}

	// EFS locations need a file system from the EFS store.
	efsBody := func(id string) string {
		return `{"EfsFilesystemArn":"arn:aws:elasticfilesystem:us-east-1:123456789012:file-system/` + id + `","Ec2Config":{"SubnetArn":"arn:aws:ec2:us-east-1:123456789012:subnet/subnet-0123456789abcdef0","SecurityGroupArns":["arn:aws:ec2:us-east-1:123456789012:security-group/sg-0123456789abcdef0"]}}`
	}
	if msg := dsyFail(t, ts, "CreateLocationEfs", efsBody("fs-0123456789abcdef0")); !strings.Contains(msg, "does not exist") {
		t.Fatalf("missing file system: %s", msg)
	}

	// ListLocations filters by type; deleting a location a task uses fails.
	if got := call("ListLocations", `{"Filters":[{"Name":"LocationType","Values":["NFS"],"Operator":"Equals"}]}`)["Locations"].([]any); len(got) != 1 {
		t.Fatalf("filtered locations: %v", got)
	}
	task := call("CreateTask", `{"SourceLocationArn":"`+nfs+`","DestinationLocationArn":"`+s3loc+`"}`)["TaskArn"].(string)
	if msg := dsyFail(t, ts, "DeleteLocation", `{"LocationArn":"`+s3loc+`"}`); !strings.Contains(msg, "used by task") {
		t.Fatalf("delete in-use location: %s", msg)
	}
	call("DeleteTask", `{"TaskArn":"`+task+`"}`)
	call("DeleteLocation", `{"LocationArn":"`+s3loc+`"}`)
	if msg := dsyFail(t, ts, "DescribeLocationS3", `{"LocationArn":"`+s3loc+`"}`); !strings.Contains(msg, "does not exist") {
		t.Fatalf("deleted location: %s", msg)
	}
}

func TestDataSyncStage1TagsAndPaging(t *testing.T) {
	_, ts := newDataSyncTestServer(t)
	call := func(op, body string) map[string]any { return dsyOK(t, ts, op, body) }
	agent := dsyTestAgent(t, call, "AAAAA-BBBBB-CCCCC-DDDDD-FFFFF")
	call("TagResource", `{"ResourceArn":"`+agent+`","Tags":[{"Key":"b","Value":"2"},{"Key":"a","Value":"1"}]}`)
	tags := call("ListTagsForResource", `{"ResourceArn":"`+agent+`"}`)["Tags"].([]any)
	if len(tags) != 2 || tags[0].(map[string]any)["Key"] != "a" {
		t.Fatalf("tags: %v", tags)
	}
	if msg := dsyFail(t, ts, "TagResource", `{"ResourceArn":"`+agent+`","Tags":[{"Key":"aws:x","Value":"1"}]}`); !strings.Contains(msg, "aws:") {
		t.Fatalf("aws: tag: %s", msg)
	}
	var many []string
	for i := 0; i < dsyMaxTags+1; i++ {
		many = append(many, fmt.Sprintf(`{"Key":"k%d","Value":"v"}`, i))
	}
	if msg := dsyFail(t, ts, "TagResource", `{"ResourceArn":"`+agent+`","Tags":[`+strings.Join(many, ",")+`]}`); !strings.Contains(msg, "at most") {
		t.Fatalf("tag limit: %s", msg)
	}
	call("UntagResource", `{"ResourceArn":"`+agent+`","Keys":["a"]}`)
	if tags := call("ListTagsForResource", `{"ResourceArn":"`+agent+`"}`)["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("after untag: %v", tags)
	}

	for _, key := range []string{"AAAAA-BBBBB-CCCCC-DDDDD-GGGGG", "AAAAA-BBBBB-CCCCC-DDDDD-HHHHH"} {
		dsyTestAgent(t, call, key)
	}
	page := call("ListAgents", `{"MaxResults":2}`)
	if len(page["Agents"].([]any)) != 2 || page["NextToken"] == nil {
		t.Fatalf("first page: %v", page)
	}
	rest := call("ListAgents", `{"MaxResults":2,"NextToken":"`+page["NextToken"].(string)+`"}`)
	if len(rest["Agents"].([]any)) != 1 || rest["NextToken"] != nil {
		t.Fatalf("second page: %v", rest)
	}
	if status, _ := dsyCall(t, ts, "ListAgents", `{"NextToken":"garbage"}`); status != http.StatusBadRequest {
		t.Fatalf("bad token: %d", status)
	}
}
