package server

import (
	"net/http"
	"testing"
)

const daxTestRole = "arn:aws:iam::123456789012:role/DAXServiceRole"

func daxCluster0(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	if c, ok := out["Cluster"].(map[string]any); ok {
		return c
	}
	clusters, _ := out["Clusters"].([]any)
	if len(clusters) != 1 {
		t.Fatalf("expected one cluster: %v", out)
	}
	return clusters[0].(map[string]any)
}

func TestDAXStage1ClusterLifecycle(t *testing.T) {
	_, ts := newDAXTestServer(t)
	created := daxCluster0(t, daxOK(t, ts, "CreateCluster", `{"ClusterName":"My-Cache","NodeType":"dax.r5.large","ReplicationFactor":3,"IamRoleArn":"`+daxTestRole+`","SSESpecification":{"Enabled":true},"ClusterEndpointEncryptionType":"TLS","Tags":[{"Key":"env","Value":"test"}]}`))
	if created["ClusterName"] != "my-cache" || created["Status"] != "creating" || created["ClusterArn"] != "arn:aws:dax:us-east-1:123456789012:cache/my-cache" {
		t.Fatalf("created: %v", created)
	}
	if created["TotalNodes"] != float64(3) || created["ActiveNodes"] != float64(0) || created["SubnetGroup"] != "default" || created["SSEDescription"].(map[string]any)["Status"] != "ENABLED" {
		t.Fatalf("created nodes/subnet/sse: %v", created)
	}
	ep := created["ClusterDiscoveryEndpoint"].(map[string]any)
	if ep["Port"] != float64(daxPortTLS) || ep["URL"] != "daxs://127.0.0.1:9111" {
		t.Fatalf("discovery endpoint: %v", ep)
	}
	if code := daxFail(t, ts, "CreateCluster", `{"ClusterName":"my-cache","NodeType":"dax.r5.large","ReplicationFactor":1,"IamRoleArn":"`+daxTestRole+`"}`); code != "ClusterAlreadyExistsFault" {
		t.Fatalf("duplicate: %s", code)
	}
	// Changes wait until the cluster is available.
	if code := daxFail(t, ts, "UpdateCluster", `{"ClusterName":"my-cache","Description":"x"}`); code != "InvalidClusterStateFault" {
		t.Fatalf("update while creating: %s", code)
	}
	described := daxCluster0(t, daxOK(t, ts, "DescribeClusters", `{"ClusterNames":["MY-CACHE"]}`))
	if described["Status"] != "available" || described["ActiveNodes"] != float64(3) {
		t.Fatalf("after describe: %v", described)
	}
	nodes := described["Nodes"].([]any)
	if nodes[0].(map[string]any)["NodeId"] != "my-cache-a" || nodes[2].(map[string]any)["NodeId"] != "my-cache-c" {
		t.Fatalf("node ids: %v", nodes)
	}
	if sgs := described["SecurityGroups"].([]any); len(sgs) != 1 || sgs[0].(map[string]any)["SecurityGroupIdentifier"] != "sg-00000000" {
		t.Fatalf("default security group: %v", sgs)
	}

	// Replication: increase, settle, decrease by node id.
	up := daxCluster0(t, daxOK(t, ts, "IncreaseReplicationFactor", `{"ClusterName":"my-cache","NewReplicationFactor":4}`))
	if up["Status"] != "modifying" || up["TotalNodes"] != float64(4) {
		t.Fatalf("increase: %v", up)
	}
	if code := daxFail(t, ts, "IncreaseReplicationFactor", `{"ClusterName":"my-cache","NewReplicationFactor":5}`); code != "InvalidClusterStateFault" {
		t.Fatalf("increase while modifying: %s", code)
	}
	daxOK(t, ts, "DescribeClusters", `{}`)
	if code := daxFail(t, ts, "IncreaseReplicationFactor", `{"ClusterName":"my-cache","NewReplicationFactor":12}`); code != "NodeQuotaForClusterExceededFault" {
		t.Fatalf("over node quota: %s", code)
	}
	if code := daxFail(t, ts, "DecreaseReplicationFactor", `{"ClusterName":"my-cache","NewReplicationFactor":3,"NodeIdsToRemove":["my-cache-z"]}`); code != "NodeNotFoundFault" {
		t.Fatalf("remove missing node: %s", code)
	}
	down := daxCluster0(t, daxOK(t, ts, "DecreaseReplicationFactor", `{"ClusterName":"my-cache","NewReplicationFactor":3,"NodeIdsToRemove":["my-cache-b"]}`))
	if ids := down["NodeIdsToRemove"].([]any); len(ids) != 1 || ids[0] != "my-cache-b" || down["TotalNodes"] != float64(3) {
		t.Fatalf("decrease: %v", down)
	}
	settled := daxCluster0(t, daxOK(t, ts, "DescribeClusters", `{"ClusterNames":["my-cache"]}`))
	if len(settled["Nodes"].([]any)) != 3 || settled["NodeIdsToRemove"] != nil {
		t.Fatalf("after decrease settles: %v", settled)
	}

	rebooted := daxCluster0(t, daxOK(t, ts, "RebootNode", `{"ClusterName":"my-cache","NodeId":"my-cache-a"}`))
	if rebooted["Nodes"].([]any)[0].(map[string]any)["NodeStatus"] != "rebooting" {
		t.Fatalf("reboot: %v", rebooted)
	}
	if code := daxFail(t, ts, "RebootNode", `{"ClusterName":"my-cache","NodeId":"my-cache-a"}`); code != "InvalidClusterStateFault" {
		t.Fatalf("reboot a rebooting node: %s", code)
	}
	daxOK(t, ts, "DescribeClusters", `{}`)

	updated := daxCluster0(t, daxOK(t, ts, "UpdateCluster", `{"ClusterName":"my-cache","Description":"cache","NotificationTopicArn":"arn:aws:sns:us-east-1:123456789012:dax-events"}`))
	if updated["Description"] != "cache" || updated["NotificationConfiguration"].(map[string]any)["TopicStatus"] != "active" {
		t.Fatalf("update: %v", updated)
	}

	deleted := daxCluster0(t, daxOK(t, ts, "DeleteCluster", `{"ClusterName":"my-cache"}`))
	if deleted["Status"] != "deleting" {
		t.Fatalf("delete: %v", deleted)
	}
	if code := daxFail(t, ts, "DescribeClusters", `{"ClusterNames":["my-cache"]}`); code != "ClusterNotFoundFault" {
		t.Fatalf("describe deleted: %s", code)
	}
}

func TestDAXStage1CreateValidation(t *testing.T) {
	_, ts := newDAXTestServer(t)
	base := `"NodeType":"dax.r5.large","ReplicationFactor":1,"IamRoleArn":"` + daxTestRole + `"`
	for name, tc := range map[string]struct{ body, code string }{
		"bad name":          {`{"ClusterName":"1bad",` + base + `}`, "InvalidParameterValueException"},
		"double hyphen":     {`{"ClusterName":"a--b",` + base + `}`, "InvalidParameterValueException"},
		"too long":          {`{"ClusterName":"abcdefghijklmnopqrstu",` + base + `}`, "InvalidParameterValueException"},
		"bad node type":     {`{"ClusterName":"c","NodeType":"cache.m5.large","ReplicationFactor":1,"IamRoleArn":"` + daxTestRole + `"}`, "InvalidParameterValueException"},
		"bad role":          {`{"ClusterName":"c","NodeType":"dax.r5.large","ReplicationFactor":1,"IamRoleArn":"role"}`, "InvalidParameterValueException"},
		"zero replicas":     {`{"ClusterName":"c","NodeType":"dax.r5.large","ReplicationFactor":0,"IamRoleArn":"` + daxTestRole + `"}`, "InvalidParameterValueException"},
		"AZ count mismatch": {`{"ClusterName":"c",` + base + `,"AvailabilityZones":["us-east-1a","us-east-1a"]}`, "InvalidParameterCombinationException"},
		"AZ not in group":   {`{"ClusterName":"c",` + base + `,"AvailabilityZones":["us-east-1z"]}`, "InvalidParameterValueException"},
		"missing subnets":   {`{"ClusterName":"c",` + base + `,"SubnetGroupName":"nope"}`, "SubnetGroupNotFoundFault"},
		"missing params":    {`{"ClusterName":"c",` + base + `,"ParameterGroupName":"nope"}`, "ParameterGroupNotFoundFault"},
		"missing SG":        {`{"ClusterName":"c",` + base + `,"SecurityGroupIds":["sg-0000dead"]}`, "InvalidParameterValueException"},
		"ipv6":              {`{"ClusterName":"c",` + base + `,"NetworkType":"ipv6"}`, "InvalidParameterCombinationException"},
		"bad window":        {`{"ClusterName":"c",` + base + `,"PreferredMaintenanceWindow":"sunday"}`, "InvalidParameterValueException"},
		"aws: tag":          {`{"ClusterName":"c",` + base + `,"Tags":[{"Key":"aws:x","Value":"y"}]}`, "InvalidParameterValueException"},
	} {
		if code := daxFail(t, ts, "CreateCluster", tc.body); code != tc.code {
			t.Fatalf("%s: %s (want %s)", name, code, tc.code)
		}
	}
	if status, _ := daxRequest(t, ts, "DescribeClusters", `{}`); status != http.StatusOK {
		t.Fatalf("describe after failed creates: %d", status)
	}
}
