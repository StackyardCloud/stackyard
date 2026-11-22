package server

import (
	"net/http"
	"testing"
)

func TestElasticsearchServiceStage1DomainLifecycle(t *testing.T) {
	ts := newElasticsearchServiceTestServer(t)
	created := eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/domain",
		`{"DomainName":"logs","ElasticsearchVersion":"7.9","ElasticsearchClusterConfig":{"InstanceType":"r5.large.elasticsearch","InstanceCount":2},
		  "EBSOptions":{"EBSEnabled":true,"VolumeSize":20},"TagList":[{"Key":"env","Value":"dev"}],
		  "AdvancedSecurityOptions":{"Enabled":true,"InternalUserDatabaseEnabled":true,"MasterUserOptions":{"MasterUserName":"admin","MasterUserPassword":"Secret-123"}}}`)
	status := created["DomainStatus"].(map[string]any)
	if status["ElasticsearchVersion"] != "7.9" || status["Processing"] != true ||
		status["ElasticsearchClusterConfig"].(map[string]any)["InstanceType"] != "r5.large.elasticsearch" {
		t.Fatalf("create: %v", status)
	}
	// Duplicate name, bad version, bad cluster config, invalid name.
	for body, code := range map[string]string{
		`{"DomainName":"logs"}`:                                                                               "ResourceAlreadyExistsException",
		`{"DomainName":"other","ElasticsearchVersion":"9.9"}`:                                                 "ValidationException",
		`{"DomainName":"other","ElasticsearchClusterConfig":{"DedicatedMasterEnabled":true}}`:                 "ValidationException",
		`{"DomainName":"other","ElasticsearchClusterConfig":{"ZoneAwarenessEnabled":true,"InstanceCount":3}}`: "ValidationException",
		`{"DomainName":"other","EBSOptions":{"EBSEnabled":true,"VolumeSize":5}}`:                              "ValidationException",
		`{"DomainName":"Bad_Name"}`:                                                                           "ValidationException",
	} {
		status, out := eslCall(t, ts, http.MethodPost, "/2015-01-01/es/domain", body)
		if out["__type"] != code || (code == "ValidationException" && status != http.StatusBadRequest) || (code != "ValidationException" && status != http.StatusConflict) {
			t.Fatalf("%s: %d %v (want %s)", body, status, out, code)
		}
	}

	// Processing settles on the next describe; the password is never stored or returned.
	eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/domain/logs", "")
	got := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/domain/logs", "")["DomainStatus"].(map[string]any)
	if got["Processing"] != false {
		t.Fatalf("still processing: %v", got)
	}
	if sec, _ := got["AdvancedSecurityOptions"].(map[string]any); sec["Enabled"] != true || sec["MasterUserOptions"] != nil {
		t.Fatalf("advanced security: %v", sec)
	}

	// The same domain is visible through the OpenSearch API, with its engine version.
	os := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2021-01-01/opensearch/domain/logs", "")["DomainStatus"].(map[string]any)
	if os["EngineVersion"] != "Elasticsearch_7.9" || os["ClusterConfig"].(map[string]any)["InstanceType"] != "r5.large.search" {
		t.Fatalf("OpenSearch view: %v", os)
	}
	// OpenSearch returns stored options as-is, so the password must never have been stored.
	if sec, _ := os["AdvancedSecurityOptions"].(map[string]any); sec["MasterUserOptions"] != nil {
		t.Fatalf("OpenSearch view leaks master user options: %v", sec)
	}

	// Config update, dry run, tags.
	cfg := eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/domain/logs/config", `{"ElasticsearchClusterConfig":{"InstanceCount":4},"DryRun":true}`)
	if cfg["DryRunResults"] == nil {
		t.Fatalf("dry run: %v", cfg)
	}
	cfg = eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/domain/logs/config", "")["DomainConfig"].(map[string]any)
	if n := cfg["ElasticsearchClusterConfig"].(map[string]any)["Options"].(map[string]any)["InstanceCount"]; n != float64(2) {
		t.Fatalf("dry run applied: %v", n)
	}
	cfg = eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/domain/logs/config", `{"ElasticsearchClusterConfig":{"InstanceCount":4}}`)["DomainConfig"].(map[string]any)
	if opts := cfg["ElasticsearchClusterConfig"].(map[string]any); opts["Options"].(map[string]any)["InstanceCount"] != float64(4) || opts["Status"].(map[string]any)["UpdateVersion"] != float64(2) {
		t.Fatalf("update: %v", opts)
	}
	arn := status["ARN"].(string)
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/tags", `{"ARN":"`+arn+`","TagList":[{"Key":"team","Value":"search"}]}`)
	tags := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/tags/?arn="+arn, "")["TagList"].([]any)
	if len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	if status, out := eslCall(t, ts, http.MethodPost, "/2015-01-01/tags", `{"ARN":"`+arn+`","TagList":[{"Key":"aws:x","Value":"y"}]}`); status != http.StatusBadRequest {
		t.Fatalf("aws: tag: %d %v", status, out)
	}

	// Upgrade 7.9 → 7.10; 7.10 → 6.8 is refused.
	if status, out := eslCall(t, ts, http.MethodPost, "/2015-01-01/es/upgradeDomain", `{"DomainName":"logs","TargetVersion":"6.8"}`); status != http.StatusBadRequest {
		t.Fatalf("downgrade: %d %v", status, out)
	}
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/upgradeDomain", `{"DomainName":"logs","TargetVersion":"7.10"}`)
	if got := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/upgradeDomain/logs/status", ""); got["UpgradeStep"] != "UPGRADE" {
		t.Fatalf("upgrade status: %v", got)
	}
	compat := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/compatibleVersions?domainName=logs", "")["CompatibleElasticsearchVersions"].([]any)
	if entry := compat[0].(map[string]any); entry["SourceVersion"] != "7.10" || entry["TargetVersions"].([]any)[0] != "OpenSearch_1.0" {
		t.Fatalf("compatible versions after upgrade: %v", compat)
	}

	// Delete, then the domain is gone from both APIs.
	if got := eslWant(t, ts, http.StatusOK, http.MethodDelete, "/2015-01-01/es/domain/logs", "")["DomainStatus"].(map[string]any); got["Deleted"] != true {
		t.Fatalf("delete: %v", got)
	}
	if status, out := eslCall(t, ts, http.MethodGet, "/2015-01-01/es/domain/logs", ""); status != http.StatusConflict || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("describe deleted: %d %v", status, out)
	}
}

func TestElasticsearchServiceStage1OpenSearchDomainsAreVisible(t *testing.T) {
	ts := newElasticsearchServiceTestServer(t)
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2021-01-01/opensearch/domain",
		`{"DomainName":"search","EngineVersion":"OpenSearch_2.11","ClusterConfig":{"InstanceType":"m6g.large.search","InstanceCount":1}}`)
	got := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/domain/search", "")["DomainStatus"].(map[string]any)
	if got["ElasticsearchVersion"] != "OpenSearch_2.11" || got["ElasticsearchClusterConfig"].(map[string]any)["InstanceType"] != "m6g.large.elasticsearch" {
		t.Fatalf("es view of an OpenSearch domain: %v", got)
	}
	names := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/domain?engineType=OpenSearch", "")["DomainNames"].([]any)
	if len(names) != 1 || names[0].(map[string]any)["DomainName"] != "search" {
		t.Fatalf("ListDomainNames: %v", names)
	}
	if names := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/domain?engineType=Elasticsearch", "")["DomainNames"].([]any); len(names) != 0 {
		t.Fatalf("engine filter: %v", names)
	}
	// Deleting through es removes it from OpenSearch too (which falls back to its stub).
	eslWant(t, ts, http.StatusOK, http.MethodDelete, "/2015-01-01/es/domain/search", "")
	os := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2021-01-01/opensearch/domain/search", "")["DomainStatus"].(map[string]any)
	if os["EngineVersion"] != nil {
		t.Fatalf("OpenSearch still sees the deleted domain: %v", os)
	}
}

func TestElasticsearchServiceStage1PackagesEndpointsConnectionsReservations(t *testing.T) {
	ts := newElasticsearchServiceTestServer(t)
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/domain", `{"DomainName":"core","ElasticsearchVersion":"7.10"}`)
	arn := "arn:aws:es:us-east-1:123456789012:domain/core"

	pkg := eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/packages",
		`{"PackageName":"synonyms","PackageType":"TXT-DICTIONARY","PackageSource":{"S3BucketName":"dicts","S3Key":"k.txt"}}`)["PackageDetails"].(map[string]any)
	id := pkg["PackageID"].(string)
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/packages/associate/"+id+"/core", "")
	if status, out := eslCall(t, ts, http.MethodDelete, "/2015-01-01/packages/"+id, ""); status != http.StatusConflict || out["__type"] != "ConflictException" {
		t.Fatalf("delete associated package: %d %v", status, out)
	}
	if list := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/domain/core/packages", "")["DomainPackageDetailsList"].([]any); len(list) != 1 {
		t.Fatalf("packages for domain: %v", list)
	}
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/packages/dissociate/"+id+"/core", "")
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/packages/update", `{"PackageID":"`+id+`","PackageSource":{"S3BucketName":"dicts","S3Key":"k2.txt"}}`)
	if hist := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/packages/"+id+"/history", "")["PackageVersionHistoryList"].([]any); len(hist) != 2 {
		t.Fatalf("package history: %v", hist)
	}
	eslWant(t, ts, http.StatusOK, http.MethodDelete, "/2015-01-01/packages/"+id, "")

	ep := eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/vpcEndpoints", `{"DomainArn":"`+arn+`","VpcOptions":{"SubnetIds":["subnet-1"]}}`)["VpcEndpoint"].(map[string]any)
	desc := eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/vpcEndpoints/describe", `{"VpcEndpointIds":["`+ep["VpcEndpointId"].(string)+`","aos-missing"]}`)
	if len(desc["VpcEndpoints"].([]any)) != 1 || len(desc["VpcEndpointErrors"].([]any)) != 1 {
		t.Fatalf("describe endpoints: %v", desc)
	}
	if status, _ := eslCall(t, ts, http.MethodPost, "/2015-01-01/es/vpcEndpoints", `{"DomainArn":"arn:aws:es:us-east-1:123456789012:domain/nope","VpcOptions":{"SubnetIds":["subnet-1"]}}`); status != http.StatusConflict {
		t.Fatalf("endpoint for a missing domain: %d", status)
	}

	conn := eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/ccs/outboundConnection",
		`{"SourceDomainInfo":{"OwnerId":"123456789012","DomainName":"core","Region":"us-east-1"},"DestinationDomainInfo":{"OwnerId":"999999999999","DomainName":"remote","Region":"us-west-2"},"ConnectionAlias":"remote"}`)
	cid := conn["CrossClusterSearchConnectionId"].(string)
	accepted := eslWant(t, ts, http.StatusOK, http.MethodPut, "/2015-01-01/es/ccs/inboundConnection/"+cid+"/accept", "")["CrossClusterSearchConnection"].(map[string]any)
	if accepted["ConnectionStatus"].(map[string]any)["StatusCode"] != "APPROVED" {
		t.Fatalf("accept: %v", accepted)
	}
	if status, _ := eslCall(t, ts, http.MethodPut, "/2015-01-01/es/ccs/inboundConnection/"+cid+"/reject", ""); status != http.StatusConflict {
		t.Fatalf("reject after accept: %d", status)
	}
	out := eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/ccs/outboundConnection/search",
		`{"Filters":[{"Name":"source-domain-info.domain-name","Values":["core"]}]}`)["CrossClusterSearchConnections"].([]any)
	if len(out) != 1 || out[0].(map[string]any)["ConnectionStatus"].(map[string]any)["StatusCode"] != "ACTIVE" {
		t.Fatalf("outbound search: %v", out)
	}

	offers := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/reservedInstanceOfferings", "")["ReservedElasticsearchInstanceOfferings"].([]any)
	offer := offers[0].(map[string]any)["ReservedElasticsearchInstanceOfferingId"].(string)
	eslWant(t, ts, http.StatusOK, http.MethodPost, "/2015-01-01/es/purchaseReservedInstanceOffering", `{"ReservedElasticsearchInstanceOfferingId":"`+offer+`","ReservationName":"nightly","InstanceCount":2}`)
	if list := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/reservedInstances", "")["ReservedElasticsearchInstances"].([]any); len(list) != 1 {
		t.Fatalf("reservations: %v", list)
	}
	if status, out := eslCall(t, ts, http.MethodGet, "/2015-01-01/es/versions?maxResults=2&nextToken=bad", ""); status != http.StatusBadRequest || out["__type"] != "InvalidPaginationTokenException" {
		t.Fatalf("bad token: %d %v", status, out)
	}
	page := eslWant(t, ts, http.StatusOK, http.MethodGet, "/2015-01-01/es/versions?maxResults=2", "")
	if len(page["ElasticsearchVersions"].([]any)) != 2 || page["NextToken"] != "2" {
		t.Fatalf("versions page: %v", page)
	}
}
