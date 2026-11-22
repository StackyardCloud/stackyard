package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ccActivate(t *testing.T, ts *httptest.Server, kind, arn string) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/_stackyard/codeconnections/"+kind+"/"+arn[strings.LastIndex(arn, "/")+1:]+"/activate", nil, nil, "codeconnections")
	assertStatus(t, resp, http.StatusOK)
	_ = mustBody(t, resp)
}

func TestCodeConnectionsStage1ConnectionsAndHosts(t *testing.T) {
	_, ts := newCodeConnectionsTestServer(t)
	conn := ccOK(t, ts, "CreateConnection", `{"ProviderType":"GitHub","ConnectionName":"gh","Tags":[{"Key":"team","Value":"a"}]}`)
	arn := conn["ConnectionArn"].(string)
	if !strings.HasPrefix(arn, "arn:aws:codeconnections:us-east-1:123456789012:connection/") || len(conn["Tags"].([]any)) != 1 {
		t.Fatalf("create connection: %v", conn)
	}
	got := ccOK(t, ts, "GetConnection", `{"ConnectionArn":"`+arn+`"}`)["Connection"].(map[string]any)
	if got["ConnectionStatus"] != "PENDING" || got["OwnerAccountId"] != "123456789012" {
		t.Fatalf("new connection: %v", got)
	}
	for body, want := range map[string]string{
		`{"ConnectionName":"x"}`: "ValidationException", // neither ProviderType nor HostArn
		`{"ConnectionName":"x","ProviderType":"GitHubEnterpriseServer"}`:                                       "ValidationException",
		`{"ConnectionName":"x","HostArn":"arn:aws:codeconnections:us-east-1:123456789012:host/nope-00000000"}`: "ResourceNotFoundException",
		`{"ConnectionName":"this-name-is-much-longer-than-thirty-two","ProviderType":"GitHub"}`:                "ValidationException",
		`{"ConnectionName":"x","ProviderType":"GitHub","Tags":[{"Key":"aws:reserved","Value":"v"}]}`:           "ValidationException",
	} {
		if code := ccErrCode(t, ts, "CreateConnection", body); code != want {
			t.Errorf("CreateConnection %s: %s, want %s", body, code, want)
		}
	}

	// Hosts are self-managed only, start PENDING, and gate connections.
	if code := ccErrCode(t, ts, "CreateHost", `{"Name":"h","ProviderType":"GitHub","ProviderEndpoint":"https://x"}`); code != "ValidationException" {
		t.Fatalf("cloud host: %s", code)
	}
	host := ccOK(t, ts, "CreateHost", `{"Name":"My Host/1","ProviderType":"GitHubEnterpriseServer","ProviderEndpoint":"https://ghe.example.com","VpcConfiguration":{"VpcId":"vpc-12345678","SubnetIds":["subnet-12345678"],"SecurityGroupIds":["sg-12345678"]}}`)["HostArn"].(string)
	if !strings.Contains(host, ":host/My-Host-1-") {
		t.Fatalf("host ARN: %s", host)
	}
	withHost := fmt.Sprintf(`{"ConnectionName":"ghe","HostArn":%q}`, host)
	if code := ccErrCode(t, ts, "CreateConnection", withHost); code != "ResourceUnavailableException" {
		t.Fatalf("connection on a pending host: %s", code)
	}
	ccActivate(t, ts, "hosts", host)
	gheConn := ccOK(t, ts, "CreateConnection", withHost)["ConnectionArn"].(string)
	h := ccOK(t, ts, "GetHost", `{"HostArn":"`+host+`"}`)
	if h["Status"] != "AVAILABLE" || h["VpcConfiguration"].(map[string]any)["VpcId"] != "vpc-12345678" {
		t.Fatalf("get host: %v", h)
	}
	ccOK(t, ts, "UpdateHost", `{"HostArn":"`+host+`","ProviderEndpoint":"https://ghe2.example.com"}`)
	if got := ccOK(t, ts, "GetHost", `{"HostArn":"`+host+`"}`)["ProviderEndpoint"]; got != "https://ghe2.example.com" {
		t.Fatalf("updated endpoint: %v", got)
	}
	filtered := ccOK(t, ts, "ListConnections", `{"HostArnFilter":"`+host+`"}`)["Connections"].([]any)
	if len(filtered) != 1 || filtered[0].(map[string]any)["ProviderType"] != "GitHubEnterpriseServer" {
		t.Fatalf("host filter: %v", filtered)
	}
	if code := ccErrCode(t, ts, "DeleteHost", `{"HostArn":"`+host+`"}`); code != "ResourceUnavailableException" {
		t.Fatalf("delete host with a connection: %s", code)
	}
	ccOK(t, ts, "DeleteConnection", `{"ConnectionArn":"`+gheConn+`"}`)
	ccOK(t, ts, "DeleteHost", `{"HostArn":"`+host+`"}`)
	if code := ccErrCode(t, ts, "GetHost", `{"HostArn":"`+host+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted host: %s", code)
	}

	// Pagination over connections.
	for i := 0; i < 3; i++ {
		ccOK(t, ts, "CreateConnection", fmt.Sprintf(`{"ProviderType":"Bitbucket","ConnectionName":"bb%d"}`, i))
	}
	page := ccOK(t, ts, "ListConnections", `{"ProviderTypeFilter":"Bitbucket","MaxResults":2}`)
	if len(page["Connections"].([]any)) != 2 || page["NextToken"] == nil {
		t.Fatalf("page 1: %v", page)
	}
	rest := ccOK(t, ts, "ListConnections", fmt.Sprintf(`{"ProviderTypeFilter":"Bitbucket","MaxResults":2,"NextToken":%q}`, page["NextToken"]))
	if len(rest["Connections"].([]any)) != 1 || rest["NextToken"] != nil {
		t.Fatalf("page 2: %v", rest)
	}
	if code := ccErrCode(t, ts, "ListConnections", `{"NextToken":"999"}`); code != "InvalidInputException" {
		t.Fatalf("bad token: %s", code)
	}

	// Tags.
	ccOK(t, ts, "TagResource", `{"ResourceArn":"`+arn+`","Tags":[{"Key":"team","Value":"b"},{"Key":"env","Value":"dev"}]}`)
	ccOK(t, ts, "UntagResource", `{"ResourceArn":"`+arn+`","TagKeys":["env"]}`)
	tags := ccOK(t, ts, "ListTagsForResource", `{"ResourceArn":"`+arn+`"}`)["Tags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["Value"] != "b" {
		t.Fatalf("tags: %v", tags)
	}
	ccOK(t, ts, "DeleteConnection", `{"ConnectionArn":"`+arn+`"}`)
	if code := ccErrCode(t, ts, "ListTagsForResource", `{"ResourceArn":"`+arn+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("tags of a deleted connection: %s", code)
	}
}

func TestCodeConnectionsStage1RepositoryLinksAndSync(t *testing.T) {
	_, ts := newCodeConnectionsTestServer(t)
	conn := ccOK(t, ts, "CreateConnection", `{"ProviderType":"GitHub","ConnectionName":"gh"}`)["ConnectionArn"].(string)
	linkBody := `{"ConnectionArn":"` + conn + `","OwnerId":"acme","RepositoryName":"infra"}`
	if code := ccErrCode(t, ts, "CreateRepositoryLink", linkBody); code != "InvalidInputException" {
		t.Fatalf("link on a pending connection: %s", code)
	}
	ccActivate(t, ts, "connections", conn)
	link := ccOK(t, ts, "CreateRepositoryLink", linkBody)["RepositoryLinkInfo"].(map[string]any)
	linkID := link["RepositoryLinkId"].(string)
	if link["ProviderType"] != "GitHub" || !strings.Contains(link["RepositoryLinkArn"].(string), ":repository-link/"+linkID) {
		t.Fatalf("link: %v", link)
	}
	if code := ccErrCode(t, ts, "CreateRepositoryLink", linkBody); code != "ResourceAlreadyExistsException" {
		t.Fatalf("duplicate link: %s", code)
	}

	syncBody := `{"Branch":"main","ConfigFile":"deploy/stack.yaml","RepositoryLinkId":"` + linkID + `","ResourceName":"web","RoleArn":"arn:aws:iam::123456789012:role/sync","SyncType":"CFN_STACK_SYNC","TriggerResourceUpdateOn":"FILE_CHANGE"}`
	cfg := ccOK(t, ts, "CreateSyncConfiguration", syncBody)["SyncConfiguration"].(map[string]any)
	if cfg["OwnerId"] != "acme" || cfg["RepositoryName"] != "infra" || cfg["TriggerResourceUpdateOn"] != "FILE_CHANGE" {
		t.Fatalf("sync config: %v", cfg)
	}
	if code := ccErrCode(t, ts, "CreateSyncConfiguration", syncBody); code != "ResourceAlreadyExistsException" {
		t.Fatalf("duplicate sync config: %s", code)
	}
	if code := ccErrCode(t, ts, "DeleteRepositoryLink", `{"RepositoryLinkId":"`+linkID+`"}`); code != "SyncConfigurationStillExistsException" {
		t.Fatalf("delete a linked repository in use: %s", code)
	}
	key := `"SyncType":"CFN_STACK_SYNC","ResourceName":"web"`
	updated := ccOK(t, ts, "UpdateSyncConfiguration", `{`+key+`,"Branch":"release"}`)["SyncConfiguration"].(map[string]any)
	if updated["Branch"] != "release" || updated["ConfigFile"] != "deploy/stack.yaml" {
		t.Fatalf("update keeps unspecified fields: %v", updated)
	}
	status := ccOK(t, ts, "GetResourceSyncStatus", `{`+key+`}`)
	if status["LatestSync"].(map[string]any)["Status"] != "SUCCEEDED" || status["DesiredState"].(map[string]any)["Branch"] != "release" {
		t.Fatalf("resource sync status: %v", status)
	}
	if latest := ccOK(t, ts, "GetRepositorySyncStatus", `{"Branch":"release","RepositoryLinkId":"`+linkID+`","SyncType":"CFN_STACK_SYNC"}`)["LatestSync"].(map[string]any); latest["Status"] != "SUCCEEDED" {
		t.Fatalf("repository sync status: %v", latest)
	}
	if code := ccErrCode(t, ts, "GetRepositorySyncStatus", `{"Branch":"main","RepositoryLinkId":"`+linkID+`","SyncType":"CFN_STACK_SYNC"}`); code != "ResourceNotFoundException" {
		t.Fatalf("branch without a sync: %s", code)
	}
	defs := ccOK(t, ts, "ListRepositorySyncDefinitions", `{"RepositoryLinkId":"`+linkID+`","SyncType":"CFN_STACK_SYNC"}`)["RepositorySyncDefinitions"].([]any)
	if len(defs) != 1 || defs[0].(map[string]any)["Directory"] != "deploy" {
		t.Fatalf("sync definitions: %v", defs)
	}

	// Blockers come from the Stackyard endpoint and are resolved through the API.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/_stackyard/codeconnections/sync-configurations/CFN_STACK_SYNC/web/blockers", []byte(`{"CreatedReason":"stack drifted"}`), nil, "codeconnections")
	assertStatus(t, resp, http.StatusOK)
	blockerID := strings.Split(strings.Split(string(mustBody(t, resp)), `"Id":"`)[1], `"`)[0]
	summary := ccOK(t, ts, "GetSyncBlockerSummary", `{`+key+`}`)["SyncBlockerSummary"].(map[string]any)
	if b := summary["LatestBlockers"].([]any)[0].(map[string]any); b["Status"] != "ACTIVE" || b["CreatedReason"] != "stack drifted" {
		t.Fatalf("blocker: %v", b)
	}
	resolved := ccOK(t, ts, "UpdateSyncBlocker", `{`+key+`,"Id":"`+blockerID+`","ResolvedReason":"fixed"}`)["SyncBlocker"].(map[string]any)
	if resolved["Status"] != "RESOLVED" || resolved["ResolvedReason"] != "fixed" {
		t.Fatalf("resolved blocker: %v", resolved)
	}
	if code := ccErrCode(t, ts, "UpdateSyncBlocker", `{`+key+`,"Id":"`+blockerID+`","ResolvedReason":"again"}`); code != "InvalidInputException" {
		t.Fatalf("resolve twice: %s", code)
	}
	if code := ccErrCode(t, ts, "UpdateSyncBlocker", `{`+key+`,"Id":"nope","ResolvedReason":"x"}`); code != "SyncBlockerDoesNotExistException" {
		t.Fatalf("unknown blocker: %s", code)
	}

	ccOK(t, ts, "DeleteSyncConfiguration", `{`+key+`}`)
	if code := ccErrCode(t, ts, "GetSyncConfiguration", `{`+key+`}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted sync config: %s", code)
	}
	ccOK(t, ts, "DeleteRepositoryLink", `{"RepositoryLinkId":"`+linkID+`"}`)
	if code := ccErrCode(t, ts, "GetRepositoryLink", `{"RepositoryLinkId":"`+linkID+`"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted link: %s", code)
	}
}
