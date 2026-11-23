package server

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

const aossEncryption = `{"Rules":[{"ResourceType":"collection","Resource":["collection/logs*"]}],"AWSOwnedKey":true}`

func aossPolicyBody(ptype, name, policy string) string {
	return fmt.Sprintf(`{"type":%q,"name":%q,"policy":%q}`, ptype, name, policy)
}

func aossCreateCollection(t *testing.T, ts *httptest.Server, name string) map[string]any {
	t.Helper()
	return aossOK(t, ts, "CreateCollection", fmt.Sprintf(`{"name":%q,"type":"TIMESERIES"}`, name))["createCollectionDetail"].(map[string]any)
}

func TestOpenSearchServerlessStage1Collections(t *testing.T) {
	_, ts := newAOSSTestServer(t)
	// No matching encryption policy yet.
	if code := aossErrCode(t, ts, "CreateCollection", `{"name":"logs-app"}`); code != "ValidationException" {
		t.Fatalf("collection without encryption policy: %s", code)
	}
	aossOK(t, ts, "CreateSecurityPolicy", aossPolicyBody("encryption", "logs", aossEncryption))
	if code := aossErrCode(t, ts, "CreateCollection", `{"name":"metrics"}`); code != "ValidationException" {
		t.Fatalf("collection outside the policy's pattern: %s", code)
	}
	c := aossCreateCollection(t, ts, "logs-app")
	if c["status"] != "CREATING" || c["kmsKeyArn"] != "auto" || c["standbyReplicas"] != "ENABLED" {
		t.Fatalf("create detail: %v", c)
	}
	id := c["id"].(string)
	if code := aossErrCode(t, ts, "CreateCollection", `{"name":"logs-app"}`); code != "ConflictException" {
		t.Fatalf("duplicate name: %s", code)
	}
	// Inline encryption config needs no policy.
	aossOK(t, ts, "CreateCollection", `{"name":"inline","encryptionConfig":{"aWSOwnedKey":true}}`)

	got := aossOK(t, ts, "BatchGetCollection", fmt.Sprintf(`{"ids":[%q,"missing000"]}`, id))
	details := got["collectionDetails"].([]any)
	if len(details) != 1 || details[0].(map[string]any)["status"] != "ACTIVE" || !strings.HasSuffix(details[0].(map[string]any)["collectionEndpoint"].(string), "/aoss/"+id) {
		t.Fatalf("batch get: %v", got)
	}
	if errs := got["collectionErrorDetails"].([]any); len(errs) != 1 || errs[0].(map[string]any)["errorCode"] != "NOT_FOUND" {
		t.Fatalf("batch get errors: %v", got)
	}
	if code := aossErrCode(t, ts, "BatchGetCollection", fmt.Sprintf(`{"ids":[%q],"names":["logs-app"]}`, id)); code != "ValidationException" {
		t.Fatalf("ids and names together: %s", code)
	}
	list := aossOK(t, ts, "ListCollections", `{"collectionFilters":{"name":"logs-app"}}`)["collectionSummaries"].([]any)
	if len(list) != 1 {
		t.Fatalf("list filter: %v", list)
	}

	aossOK(t, ts, "UpdateCollection", fmt.Sprintf(`{"id":%q,"deletionProtection":"ENABLED"}`, id))
	if code := aossErrCode(t, ts, "UpdateCollection", fmt.Sprintf(`{"id":%q,"description":"again"}`, id)); code != "ConflictException" {
		t.Fatalf("update while UPDATING: %s", code)
	}
	aossOK(t, ts, "BatchGetCollection", fmt.Sprintf(`{"ids":[%q]}`, id)) // settles to ACTIVE
	if code := aossErrCode(t, ts, "DeleteCollection", fmt.Sprintf(`{"id":%q}`, id)); code != "ValidationException" {
		t.Fatalf("delete with deletion protection: %s", code)
	}
	aossOK(t, ts, "UpdateCollection", fmt.Sprintf(`{"id":%q,"deletionProtection":"DISABLED"}`, id))
	aossOK(t, ts, "BatchGetCollection", fmt.Sprintf(`{"ids":[%q]}`, id))
	if d := aossOK(t, ts, "DeleteCollection", fmt.Sprintf(`{"id":%q}`, id))["deleteCollectionDetail"].(map[string]any); d["status"] != "DELETING" {
		t.Fatalf("delete: %v", d)
	}
	if code := aossErrCode(t, ts, "DeleteCollection", fmt.Sprintf(`{"id":%q}`, id)); code != "ResourceNotFoundException" {
		t.Fatalf("delete twice: %s", code)
	}
}

func TestOpenSearchServerlessStage1PolicyValidationAndVersions(t *testing.T) {
	_, ts := newAOSSTestServer(t)
	for name, body := range map[string]string{
		"not json":           aossPolicyBody("encryption", "bad", `{`),
		"no key choice":      aossPolicyBody("encryption", "bad", `{"Rules":[{"ResourceType":"collection","Resource":["collection/a*"]}]}`),
		"wrong prefix":       aossPolicyBody("encryption", "bad", `{"Rules":[{"ResourceType":"collection","Resource":["index/a*"]}],"AWSOwnedKey":true}`),
		"network not array":  aossPolicyBody("network", "bad", `{"Rules":[]}`),
		"data no principal":  aossPolicyBody("data", "bad", `[{"Rules":[{"ResourceType":"index","Resource":["index/a/*"],"Permission":["aoss:ReadDocument"]}]}]`),
		"data bad perm":      aossPolicyBody("data", "bad", `[{"Rules":[{"ResourceType":"index","Resource":["index/a/*"],"Permission":["es:*"]}],"Principal":["arn:aws:iam::123456789012:root"]}]`),
		"retention two opts": aossPolicyBody("retention", "bad", `{"Rules":[{"ResourceType":"index","Resource":["index/a/*"],"MinIndexRetention":"5d","NoMinIndexRetention":true}]}`),
	} {
		op := "CreateSecurityPolicy"
		if strings.Contains(body, `"data"`) {
			op = "CreateAccessPolicy"
		} else if strings.Contains(body, `"retention"`) {
			op = "CreateLifecyclePolicy"
		}
		if code := aossErrCode(t, ts, op, body); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	p := aossOK(t, ts, "CreateSecurityPolicy", aossPolicyBody("encryption", "logs", aossEncryption))["securityPolicyDetail"].(map[string]any)
	if _, ok := p["policy"].(map[string]any); !ok {
		t.Fatalf("policy should come back as a JSON document: %v", p)
	}
	if code := aossErrCode(t, ts, "CreateSecurityPolicy", aossPolicyBody("encryption", "logs", aossEncryption)); code != "ConflictException" {
		t.Fatalf("duplicate policy: %s", code)
	}
	stale := p["policyVersion"].(string)
	upd := fmt.Sprintf(`{"type":"encryption","name":"logs","policyVersion":%q,"description":"v2"}`, stale)
	next := aossOK(t, ts, "UpdateSecurityPolicy", upd)["securityPolicyDetail"].(map[string]any)
	if next["policyVersion"] == stale || next["description"] != "v2" {
		t.Fatalf("update: %v", next)
	}
	if code := aossErrCode(t, ts, "UpdateSecurityPolicy", upd); code != "ConflictException" {
		t.Fatalf("stale policyVersion: %s", code)
	}

	// Lifecycle: effective policy per index.
	aossOK(t, ts, "CreateLifecyclePolicy", aossPolicyBody("retention", "keep", `{"Rules":[{"ResourceType":"index","Resource":["index/logs-app/app-*"],"MinIndexRetention":"15d"}]}`))
	eff := aossOK(t, ts, "BatchGetEffectiveLifecyclePolicy", `{"resourceIdentifiers":[{"type":"retention","resource":"index/logs-app/app-2026"},{"type":"retention","resource":"index/other/x"}]}`)
	d := eff["effectiveLifecyclePolicyDetails"].([]any)
	if len(d) != 1 || d[0].(map[string]any)["retentionPeriod"] != "15d" || len(eff["effectiveLifecyclePolicyErrorDetails"].([]any)) != 1 {
		t.Fatalf("effective lifecycle: %v", eff)
	}
	stats := aossOK(t, ts, "GetPoliciesStats", `{}`)
	if stats["TotalPolicyCount"] != float64(2) {
		t.Fatalf("stats: %v", stats)
	}
	if got := aossOK(t, ts, "ListSecurityPolicies", `{"type":"encryption","resource":["logs-app"]}`)["securityPolicySummaries"].([]any); len(got) != 1 {
		t.Fatalf("resource filter: %v", got)
	}
	if got := aossOK(t, ts, "ListSecurityPolicies", `{"type":"encryption","resource":["metrics"]}`)["securityPolicySummaries"].([]any); len(got) != 0 {
		t.Fatalf("resource filter miss: %v", got)
	}
}

func TestOpenSearchServerlessStage1VpcEndpointsUseEC2(t *testing.T) {
	_, ts := newAOSSTestServer(t)
	subnets := ec2DescribeIDs(t, ts, "DescribeSubnets", "subnetId")
	if len(subnets) == 0 {
		t.Skip("no default subnets in the EC2 store")
	}
	if code := aossErrCode(t, ts, "CreateVpcEndpoint", fmt.Sprintf(`{"name":"vpce-a","vpcId":"vpc-ffffffff","subnetIds":[%q]}`, subnets[0])); code != "ValidationException" {
		t.Fatalf("unknown VPC: %s", code)
	}
	if code := aossErrCode(t, ts, "CreateVpcEndpoint", `{"name":"vpce-a","vpcId":"vpc-00000001","subnetIds":["subnet-0123456789abcdef0"]}`); code != "ValidationException" {
		t.Fatalf("unknown subnet: %s", code)
	}
	e := aossOK(t, ts, "CreateVpcEndpoint", fmt.Sprintf(`{"name":"vpce-a","vpcId":"vpc-00000001","subnetIds":[%q]}`, subnets[0]))["createVpcEndpointDetail"].(map[string]any)
	id := e["id"].(string)
	if e["status"] != "PENDING" {
		t.Fatalf("create: %v", e)
	}
	got := aossOK(t, ts, "BatchGetVpcEndpoint", fmt.Sprintf(`{"ids":[%q]}`, id))["vpcEndpointDetails"].([]any)[0].(map[string]any)
	if got["status"] != "ACTIVE" || len(got["securityGroupIds"].([]any)) != 1 {
		t.Fatalf("endpoint should be ACTIVE with the VPC's default group: %v", got)
	}
	if code := aossErrCode(t, ts, "UpdateVpcEndpoint", fmt.Sprintf(`{"id":%q,"removeSubnetIds":[%q]}`, id, subnets[0])); code != "ValidationException" {
		t.Fatalf("removing the last subnet: %s", code)
	}
	aossOK(t, ts, "DeleteVpcEndpoint", fmt.Sprintf(`{"id":%q}`, id))
}

// ec2DescribeIDs lists ids from an EC2 Describe* call against the test server.
func ec2DescribeIDs(t *testing.T, ts *httptest.Server, action, tag string) []string {
	t.Helper()
	resp := signedRequestWithService(t, "POST", ts.URL+"/", []byte("Action="+action+"&Version=2016-11-15"), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "ec2")
	body := string(mustBody(t, resp))
	var out []string
	for _, part := range strings.Split(body, "<"+tag+">")[1:] {
		out = append(out, strings.SplitN(part, "<", 2)[0])
	}
	return out
}

func TestOpenSearchServerlessStage1IndexesIngestionAndTags(t *testing.T) {
	_, ts := newAOSSTestServer(t)
	aossOK(t, ts, "CreateSecurityPolicy", aossPolicyBody("encryption", "logs", aossEncryption))
	id := aossCreateCollection(t, ts, "logs-app")["id"].(string)
	arn := "arn:aws:aoss:us-east-1:123456789012:collection/" + id

	aossOK(t, ts, "CreateIndex", fmt.Sprintf(`{"id":%q,"indexName":"books","indexSchema":{"mappings":{"properties":{"title":{"type":"text"}}}}}`, id))
	if code := aossErrCode(t, ts, "CreateIndex", fmt.Sprintf(`{"id":%q,"indexName":"books"}`, id)); code != "ConflictException" {
		t.Fatalf("duplicate index: %s", code)
	}
	if code := aossErrCode(t, ts, "CreateIndex", fmt.Sprintf(`{"id":%q,"indexName":"_bad"}`, id)); code != "ValidationException" {
		t.Fatalf("bad index name: %s", code)
	}
	schema := aossOK(t, ts, "GetIndex", fmt.Sprintf(`{"id":%q,"indexName":"books"}`, id))["indexSchema"].(map[string]any)
	if _, ok := schema["mappings"]; !ok {
		t.Fatalf("schema: %v", schema)
	}

	job := aossOK(t, ts, "CreateIngestionJob", fmt.Sprintf(`{"collectionId":%q,"ingestionJobName":"load","indexName":"books","source":{"s3":{"bucket":"data"}}}`, id))["ingestionJobDetail"].(map[string]any)
	jid := job["ingestionJobId"].(string)
	read := func() string {
		return aossOK(t, ts, "GetIngestionJob", fmt.Sprintf(`{"collectionId":%q,"ingestionJobId":%q}`, id, jid))["ingestionJobDetail"].(map[string]any)["status"].(string)
	}
	if a, b := read(), read(); a != "ACTIVE" || b != "COMPLETED" {
		t.Fatalf("job progression: %s then %s", a, b)
	}
	if code := aossErrCode(t, ts, "CancelIngestionJob", fmt.Sprintf(`{"collectionId":%q,"ingestionJobId":%q}`, id, jid)); code != "ConflictException" {
		t.Fatalf("cancel a completed job: %s", code)
	}

	aossOK(t, ts, "TagResource", fmt.Sprintf(`{"resourceArn":%q,"tags":[{"key":"env","value":"dev"}]}`, arn))
	tags := aossOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"resourceArn":%q}`, arn))["tags"].([]any)
	if len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	if code := aossErrCode(t, ts, "TagResource", `{"resourceArn":"arn:aws:aoss:us-east-1:123456789012:collection/nope","tags":[{"key":"a","value":"b"}]}`); code != "ResourceNotFoundException" {
		t.Fatalf("tag unknown resource: %s", code)
	}

	// Callers can't reach internal request keys: an unknown member is refused.
	if code := aossErrCode(t, ts, "CreateVpcEndpoint", `{"name":"x-y","vpcId":"vpc-00000001","subnetIds":["subnet-00000001"],"__ec2":{}}`); code != "ValidationException" {
		t.Fatalf("internal key accepted: %s", code)
	}
}

func TestOpenSearchServerlessStage1SecurityConfigsAndGroups(t *testing.T) {
	_, ts := newAOSSTestServer(t)
	if code := aossErrCode(t, ts, "CreateSecurityConfig", `{"type":"saml","name":"corp"}`); code != "ValidationException" {
		t.Fatalf("saml without options: %s", code)
	}
	cfg := aossOK(t, ts, "CreateSecurityConfig", `{"type":"saml","name":"corp","samlOptions":{"metadata":"<xml/>"}}`)["securityConfigDetail"].(map[string]any)
	if cfg["id"] != "saml/123456789012/corp" {
		t.Fatalf("config id: %v", cfg)
	}
	upd := fmt.Sprintf(`{"id":"saml/123456789012/corp","configVersion":%q,"description":"d"}`, cfg["configVersion"])
	aossOK(t, ts, "UpdateSecurityConfig", upd)
	if code := aossErrCode(t, ts, "UpdateSecurityConfig", upd); code != "ConflictException" {
		t.Fatalf("stale configVersion: %s", code)
	}

	g := aossOK(t, ts, "CreateCollectionGroup", `{"name":"team","standbyReplicas":"DISABLED"}`)["createCollectionGroupDetail"].(map[string]any)
	aossOK(t, ts, "CreateSecurityPolicy", aossPolicyBody("encryption", "logs", aossEncryption))
	aossOK(t, ts, "CreateCollection", `{"name":"logs-a","collectionGroupName":"team"}`)
	if code := aossErrCode(t, ts, "DeleteCollectionGroup", fmt.Sprintf(`{"id":%q}`, g["id"])); code != "ConflictException" {
		t.Fatalf("delete a group that still has collections: %s", code)
	}
	got := aossOK(t, ts, "BatchGetCollectionGroup", `{"names":["team"]}`)["collectionGroupDetails"].([]any)[0].(map[string]any)
	if got["numberOfCollections"] != float64(1) {
		t.Fatalf("group detail: %v", got)
	}
	if code := aossErrCode(t, ts, "CreateCollectionGroup", `{"name":"bad","standbyReplicas":"DISABLED","capacityLimits":{"minSearchCapacityInOCU":8,"maxSearchCapacityInOCU":4}}`); code != "ValidationException" {
		t.Fatalf("min above max: %s", code)
	}
}

func TestOpenSearchServerlessStage1ClientTokenIdempotency(t *testing.T) {
	srv, ts := newAOSSTestServer(t)
	aossOK(t, ts, "CreateSecurityPolicy", aossPolicyBody("encryption", "logs", aossEncryption))
	a := aossOK(t, ts, "CreateCollection", `{"name":"logs-a","clientToken":"tok-1"}`)["createCollectionDetail"].(map[string]any)
	b := aossOK(t, ts, "CreateCollection", `{"name":"logs-a","clientToken":"tok-1"}`)["createCollectionDetail"].(map[string]any)
	if a["id"] != b["id"] {
		t.Fatalf("same clientToken gave two collections: %v %v", a["id"], b["id"])
	}
	aossOK(t, ts, "BatchGetCollection", fmt.Sprintf(`{"ids":[%q]}`, a["id"]))
	aossOK(t, ts, "DeleteCollection", fmt.Sprintf(`{"id":%q}`, a["id"]))
	srv.openSearchServerless.mu.Lock()
	n := len(srv.openSearchServerless.tokens)
	srv.openSearchServerless.mu.Unlock()
	if n != 0 {
		t.Fatalf("deleting the collection left %d client tokens", n)
	}
}
