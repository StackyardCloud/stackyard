package server

import (
	"fmt"
	"strings"
	"testing"
)

func TestDAXStage2ParameterGroups(t *testing.T) {
	_, ts := newDAXTestServer(t)
	defaults := daxOK(t, ts, "DescribeDefaultParameters", `{}`)["Parameters"].([]any)
	if len(defaults) != 2 || defaults[0].(map[string]any)["ParameterName"] != "query-ttl-millis" || defaults[0].(map[string]any)["ParameterValue"] != "300000" {
		t.Fatalf("defaults: %v", defaults)
	}
	daxOK(t, ts, "CreateParameterGroup", `{"ParameterGroupName":"ttl","Description":"short ttl"}`)
	if code := daxFail(t, ts, "CreateParameterGroup", `{"ParameterGroupName":"TTL"}`); code != "ParameterGroupAlreadyExistsFault" {
		t.Fatalf("duplicate: %s", code)
	}
	for name, body := range map[string]string{
		"unknown param": `{"ParameterGroupName":"ttl","ParameterNameValues":[{"ParameterName":"nope","ParameterValue":"1"}]}`,
		"bad value":     `{"ParameterGroupName":"ttl","ParameterNameValues":[{"ParameterName":"query-ttl-millis","ParameterValue":"-1"}]}`,
		"empty":         `{"ParameterGroupName":"ttl","ParameterNameValues":[]}`,
	} {
		if code := daxFail(t, ts, "UpdateParameterGroup", body); code != "InvalidParameterValueException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	if code := daxFail(t, ts, "UpdateParameterGroup", `{"ParameterGroupName":"default.dax1.0","ParameterNameValues":[{"ParameterName":"query-ttl-millis","ParameterValue":"1"}]}`); code != "InvalidParameterGroupStateFault" {
		t.Fatalf("modify default: %s", code)
	}

	// A cluster on the group picks up changes: applying, then in-sync.
	daxOK(t, ts, "CreateCluster", `{"ClusterName":"pg","NodeType":"dax.t3.small","ReplicationFactor":1,"IamRoleArn":"`+daxTestRole+`","ParameterGroupName":"ttl"}`)
	daxOK(t, ts, "DescribeClusters", `{}`)
	daxOK(t, ts, "UpdateParameterGroup", `{"ParameterGroupName":"ttl","ParameterNameValues":[{"ParameterName":"query-ttl-millis","ParameterValue":"60000"}]}`)
	params := daxOK(t, ts, "DescribeParameters", `{"ParameterGroupName":"ttl","Source":"user"}`)["Parameters"].([]any)
	if len(params) != 1 || params[0].(map[string]any)["ParameterValue"] != "60000" {
		t.Fatalf("user parameters: %v", params)
	}
	c := daxCluster0(t, daxOK(t, ts, "DescribeClusters", `{"ClusterNames":["pg"]}`))
	if c["ParameterGroup"].(map[string]any)["ParameterApplyStatus"] != "in-sync" {
		t.Fatalf("param status after settle: %v", c["ParameterGroup"])
	}
	if code := daxFail(t, ts, "DeleteParameterGroup", `{"ParameterGroupName":"ttl"}`); code != "InvalidParameterGroupStateFault" {
		t.Fatalf("delete in-use group: %s", code)
	}
	daxOK(t, ts, "DeleteCluster", `{"ClusterName":"pg"}`)
	daxOK(t, ts, "DeleteParameterGroup", `{"ParameterGroupName":"ttl"}`)
	if code := daxFail(t, ts, "DescribeParameters", `{"ParameterGroupName":"ttl"}`); code != "ParameterGroupNotFoundFault" {
		t.Fatalf("deleted group: %s", code)
	}
}

func TestDAXStage2SubnetGroups(t *testing.T) {
	_, ts := newDAXTestServer(t)
	g := daxOK(t, ts, "CreateSubnetGroup", `{"SubnetGroupName":"net","SubnetIds":["subnet-00000001"]}`)["SubnetGroup"].(map[string]any)
	if g["VpcId"] != "vpc-00000001" || g["Subnets"].([]any)[0].(map[string]any)["SubnetAvailabilityZone"] != "us-east-1a" {
		t.Fatalf("subnet group: %v", g)
	}
	if code := daxFail(t, ts, "CreateSubnetGroup", `{"SubnetGroupName":"bad","SubnetIds":["subnet-0000dead"]}`); code != "InvalidSubnet" {
		t.Fatalf("missing subnet: %s", code)
	}
	if code := daxFail(t, ts, "CreateSubnetGroup", `{"SubnetGroupName":"net","SubnetIds":["subnet-00000001"]}`); code != "SubnetGroupAlreadyExistsFault" {
		t.Fatalf("duplicate: %s", code)
	}
	daxOK(t, ts, "CreateCluster", `{"ClusterName":"sn","NodeType":"dax.t3.small","ReplicationFactor":1,"IamRoleArn":"`+daxTestRole+`","SubnetGroupName":"net"}`)
	if code := daxFail(t, ts, "DeleteSubnetGroup", `{"SubnetGroupName":"net"}`); code != "SubnetGroupInUseFault" {
		t.Fatalf("delete in-use: %s", code)
	}
	updated := daxOK(t, ts, "UpdateSubnetGroup", `{"SubnetGroupName":"net","Description":"cache subnets"}`)["SubnetGroup"].(map[string]any)
	if updated["Description"] != "cache subnets" {
		t.Fatalf("update: %v", updated)
	}
	daxOK(t, ts, "DeleteCluster", `{"ClusterName":"sn"}`)
	if msg := daxOK(t, ts, "DeleteSubnetGroup", `{"SubnetGroupName":"net"}`)["DeletionMessage"]; !strings.Contains(fmt.Sprint(msg), "net") {
		t.Fatalf("delete message: %v", msg)
	}
	if code := daxFail(t, ts, "DescribeSubnetGroups", `{"SubnetGroupNames":["net"]}`); code != "SubnetGroupNotFoundFault" {
		t.Fatalf("deleted group: %s", code)
	}
}

func TestDAXStage2TagsEventsAndPaging(t *testing.T) {
	_, ts := newDAXTestServer(t)
	arn := "arn:aws:dax:us-east-1:123456789012:cache/tagged"
	daxOK(t, ts, "CreateCluster", `{"ClusterName":"tagged","NodeType":"dax.t3.small","ReplicationFactor":1,"IamRoleArn":"`+daxTestRole+`"}`)
	tags := daxOK(t, ts, "TagResource", `{"ResourceName":"`+arn+`","Tags":[{"Key":"team","Value":"data"}]}`)["Tags"].([]any)
	if len(tags) != 1 {
		t.Fatalf("tag: %v", tags)
	}
	if code := daxFail(t, ts, "UntagResource", `{"ResourceName":"`+arn+`","TagKeys":["missing"]}`); code != "TagNotFoundFault" {
		t.Fatalf("untag missing: %s", code)
	}
	if code := daxFail(t, ts, "ListTags", `{"ResourceName":"not-an-arn"}`); code != "InvalidARNFault" {
		t.Fatalf("bad arn: %s", code)
	}
	if code := daxFail(t, ts, "ListTags", `{"ResourceName":"arn:aws:dax:us-east-1:123456789012:cache/gone"}`); code != "ClusterNotFoundFault" {
		t.Fatalf("missing cluster: %s", code)
	}
	daxOK(t, ts, "UntagResource", `{"ResourceName":"`+arn+`","TagKeys":["team"]}`)
	if left := daxOK(t, ts, "ListTags", `{"ResourceName":"`+arn+`"}`)["Tags"].([]any); len(left) != 0 {
		t.Fatalf("after untag: %v", left)
	}

	events := daxOK(t, ts, "DescribeEvents", `{"SourceName":"tagged","SourceType":"CLUSTER"}`)["Events"].([]any)
	if len(events) == 0 || events[0].(map[string]any)["Message"] != "Cluster created" {
		t.Fatalf("events: %v", events)
	}
	if code := daxFail(t, ts, "DescribeEvents", `{"SourceType":"NODE"}`); code != "InvalidParameterValueException" {
		t.Fatalf("bad source type: %s", code)
	}

	for i := 0; i < 25; i++ {
		daxOK(t, ts, "CreateParameterGroup", fmt.Sprintf(`{"ParameterGroupName":"pg%02d"}`, i))
	}
	page := daxOK(t, ts, "DescribeParameterGroups", `{"MaxResults":20}`)
	if len(page["ParameterGroups"].([]any)) != 20 || page["NextToken"] != "20" {
		t.Fatalf("first page: %v", page["NextToken"])
	}
	rest := daxOK(t, ts, "DescribeParameterGroups", `{"MaxResults":20,"NextToken":"20"}`)
	if len(rest["ParameterGroups"].([]any)) != 6 || rest["NextToken"] != nil {
		t.Fatalf("second page: %v", rest)
	}
	for _, body := range []string{`{"MaxResults":5}`, `{"NextToken":"zz"}`, `{"NextToken":"999"}`} {
		if code := daxFail(t, ts, "DescribeParameterGroups", body); code != "InvalidParameterValueException" {
			t.Fatalf("%s: %s", body, code)
		}
	}
}

// Group names are stored lowercase; clusters may name them in any case and
// must then be tied to the stored group.
func TestDAXStage2GroupNamesAreCaseInsensitive(t *testing.T) {
	_, ts := newDAXTestServer(t)
	daxOK(t, ts, "CreateSubnetGroup", `{"SubnetGroupName":"Net-A","SubnetIds":["subnet-00000001"]}`)
	daxOK(t, ts, "CreateParameterGroup", `{"ParameterGroupName":"TTL-A"}`)
	c := daxCluster0(t, daxOK(t, ts, "CreateCluster", `{"ClusterName":"mixed","NodeType":"dax.t3.small","ReplicationFactor":1,"IamRoleArn":"`+daxTestRole+`","SubnetGroupName":"NET-A","ParameterGroupName":"ttl-A"}`))
	if c["SubnetGroup"] != "net-a" || c["ParameterGroup"].(map[string]any)["ParameterGroupName"] != "ttl-a" {
		t.Fatalf("stored group names: %v %v", c["SubnetGroup"], c["ParameterGroup"])
	}
	if code := daxFail(t, ts, "DeleteSubnetGroup", `{"SubnetGroupName":"net-a"}`); code != "SubnetGroupInUseFault" {
		t.Fatalf("subnet group in use: %s", code)
	}
	if code := daxFail(t, ts, "DeleteParameterGroup", `{"ParameterGroupName":"TTL-a"}`); code != "InvalidParameterGroupStateFault" {
		t.Fatalf("parameter group in use: %s", code)
	}
	daxOK(t, ts, "DescribeClusters", `{}`)
	daxOK(t, ts, "IncreaseReplicationFactor", `{"ClusterName":"mixed","NewReplicationFactor":2}`)
}
