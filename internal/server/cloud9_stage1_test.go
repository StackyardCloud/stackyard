package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// c9Age moves an environment's creation (and deletion) back by d.
func c9Age(srv *Server, id string, d time.Duration) {
	st := srv.cloud9
	st.mu.Lock()
	defer st.mu.Unlock()
	if e := st.envs[id]; e != nil {
		e.created = e.created.Add(-d)
		if !e.deleting.IsZero() {
			e.deleting = e.deleting.Add(-d)
		}
	}
}

func c9Status(t *testing.T, ts *httptest.Server, id string) string {
	t.Helper()
	out := c9OK(t, ts, c9API, "DescribeEnvironmentStatus", `{"environmentId":"`+id+`"}`)
	s, _ := out["status"].(string)
	return s
}

func TestCloud9Stage1EnvironmentLifecycle(t *testing.T) {
	srv, ts := newC9TestServer(t)
	create := `{"name":"dev","instanceType":"t3.small","imageId":"amazonlinux-2023-x86_64","tags":[{"Key":"team","Value":"a"}]}`

	if code := c9Fail(t, ts, c9API, "CreateEnvironmentEC2", `{"name":"x","instanceType":"t3.small","imageId":"windows"}`, http.StatusBadRequest); code != "BadRequestException" {
		t.Fatalf("bad image: %s", code)
	}
	if code := c9Fail(t, ts, c9API, "CreateEnvironmentEC2", `{"name":"x","instanceType":"t3.small","imageId":"amazonlinux-2023-x86_64","subnetId":"subnet-0123abcd"}`, http.StatusNotFound); code != "NotFoundException" {
		t.Fatalf("missing subnet: %s", code)
	}
	if out := c9OK(t, ts, c9API, "CreateEnvironmentEC2", `{"name":"dry","instanceType":"t3.small","imageId":"ubuntu-22.04-x86_64","dryRun":true}`); out["environmentId"] != nil {
		t.Fatalf("dry run created %v", out)
	}
	id, _ := c9OK(t, ts, c9API, "CreateEnvironmentEC2", create)["environmentId"].(string)
	if code := c9Fail(t, ts, c9API, "CreateEnvironmentEC2", create, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("duplicate name: %s", code)
	}
	ids := c9OK(t, ts, c9API, "ListEnvironments", `{}`)["environmentIds"].([]any)
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("list: %v", ids)
	}

	// The environment runs on an EC2 instance from the EC2 store.
	srv.cloud9.mu.Lock()
	iid := srv.cloud9.envs[id].instanceID
	srv.cloud9.mu.Unlock()
	if res := srv.ec2.DescribeInstances([]string{iid}); len(res) != 1 || res[0].Instances[0].Tags["aws:cloud9:environment"] != id {
		t.Fatalf("instance %s: %+v", iid, res)
	}

	envs := c9OK(t, ts, c9API, "DescribeEnvironments", `{"environmentIds":["`+id+`"]}`)["environments"].([]any)
	if lc := envs[0].(map[string]any)["lifecycle"].(map[string]any)["status"]; lc != "CREATING" {
		t.Fatalf("lifecycle: %v", lc)
	}
	if got := c9Status(t, ts, id); got != "creating" {
		t.Fatalf("status: %s", got)
	}
	c9Age(srv, id, time.Minute)
	if got := c9Status(t, ts, id); got != "ready" {
		t.Fatalf("status after create: %s", got)
	}
	if _, err := srv.ec2.StopInstances([]string{iid}); err != nil {
		t.Fatal(err)
	}
	if got := c9Status(t, ts, id); got != "stopped" {
		t.Fatalf("status of stopped instance: %s", got)
	}

	c9OK(t, ts, c9API, "UpdateEnvironment", `{"environmentId":"`+id+`","name":"dev2","managedCredentialsAction":"DISABLE"}`)
	env := c9OK(t, ts, c9API, "DescribeEnvironments", `{"environmentIds":["`+id+`"]}`)["environments"].([]any)[0].(map[string]any)
	if env["name"] != "dev2" || env["managedCredentialsStatus"] != "DISABLED_BY_OWNER" {
		t.Fatalf("updated: %v", env)
	}

	arn := "arn:aws:cloud9:us-east-1:123456789012:environment:" + id
	if code := c9Fail(t, ts, c9API, "TagResource", `{"ResourceARN":"`+arn+`","Tags":[{"Key":"aws:x","Value":"y"}]}`, http.StatusBadRequest); code != "BadRequestException" {
		t.Fatalf("reserved tag: %s", code)
	}
	c9OK(t, ts, c9API, "UntagResource", `{"ResourceARN":"`+arn+`","TagKeys":["team"]}`)
	if tags := c9OK(t, ts, c9API, "ListTagsForResource", `{"ResourceARN":"`+arn+`"}`)["Tags"].([]any); len(tags) != 0 {
		t.Fatalf("tags: %v", tags)
	}

	c9OK(t, ts, c9API, "DeleteEnvironment", `{"environmentId":"`+id+`"}`)
	if got := c9Status(t, ts, id); got != "deleting" {
		t.Fatalf("status while deleting: %s", got)
	}
	if st := srv.ec2.DescribeInstances([]string{iid})[0].Instances[0].StateName; st != "terminated" {
		t.Fatalf("instance after delete: %s", st)
	}
	c9Age(srv, id, time.Minute)
	if code := c9Fail(t, ts, c9API, "DescribeEnvironmentStatus", `{"environmentId":"`+id+`"}`, http.StatusNotFound); code != "NotFoundException" {
		t.Fatalf("deleted: %s", code)
	}
}

func TestCloud9Stage1Memberships(t *testing.T) {
	_, ts := newC9TestServer(t)
	id, _ := c9OK(t, ts, c9API, "CreateEnvironmentEC2", `{"name":"team","instanceType":"t3.small","imageId":"ubuntu-22.04-x86_64","ownerArn":"arn:aws:iam::123456789012:user/owner"}`)["environmentId"].(string)
	user := func(i int) string { return fmt.Sprintf("arn:aws:iam::123456789012:user/dev%d", i) }
	m := c9OK(t, ts, c9API, "CreateEnvironmentMembership", `{"environmentId":"`+id+`","userArn":"`+user(0)+`","permissions":"read-only"}`)["membership"].(map[string]any)
	if m["permissions"] != "read-only" || m["userId"] == "" {
		t.Fatalf("membership: %v", m)
	}
	if code := c9Fail(t, ts, c9API, "CreateEnvironmentMembership", `{"environmentId":"`+id+`","userArn":"`+user(0)+`","permissions":"read-only"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("duplicate: %s", code)
	}
	for i := 1; i < c9MaxMembers; i++ {
		c9OK(t, ts, c9API, "CreateEnvironmentMembership", `{"environmentId":"`+id+`","userArn":"`+user(i)+`","permissions":"read-write"}`)
	}
	if code := c9Fail(t, ts, c9API, "CreateEnvironmentMembership", `{"environmentId":"`+id+`","userArn":"`+user(99)+`","permissions":"read-write"}`, http.StatusTooManyRequests); code != "LimitExceededException" {
		t.Fatalf("member cap: %s", code)
	}
	owner := "arn:aws:iam::123456789012:user/owner"
	if code := c9Fail(t, ts, c9API, "DeleteEnvironmentMembership", `{"environmentId":"`+id+`","userArn":"`+owner+`"}`, http.StatusBadRequest); code != "BadRequestException" {
		t.Fatalf("owner removal: %s", code)
	}
	up := c9OK(t, ts, c9API, "UpdateEnvironmentMembership", `{"environmentId":"`+id+`","userArn":"`+user(0)+`","permissions":"read-write"}`)["membership"].(map[string]any)
	if up["permissions"] != "read-write" {
		t.Fatalf("update: %v", up)
	}
	c9OK(t, ts, c9API, "DeleteEnvironmentMembership", `{"environmentId":"`+id+`","userArn":"`+user(0)+`"}`)
	page := c9OK(t, ts, c9API, "DescribeEnvironmentMemberships", `{"environmentId":"`+id+`","maxResults":3}`)
	if len(page["memberships"].([]any)) != 3 || page["nextToken"] == nil {
		t.Fatalf("page: %v", page)
	}
	owners := c9OK(t, ts, c9API, "DescribeEnvironmentMemberships", `{"userArn":"`+owner+`","permissions":["owner"]}`)["memberships"].([]any)
	if len(owners) != 1 {
		t.Fatalf("owners: %v", owners)
	}
}
