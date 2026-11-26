package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func wdocCall(t *testing.T, ts *httptest.Server, method, path, body string, headers map[string]string) (int, map[string]any) {
	t.Helper()
	return collabCall(t, ts, "workdocs", method, path, body, headers)
}

func wdocOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	status, out := wdocCall(t, ts, method, path, body, nil)
	if status < 200 || status > 299 {
		t.Fatalf("%s %s: %d %v", method, path, status, out)
	}
	return out
}

func wdocPut(t *testing.T, url, contentType string, data []byte) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestWorkDocsDocumentLifecycle(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	users := wdocOK(t, ts, http.MethodGet, "/api/v1/users?query=stackyard", "")
	admin, _ := users["Users"].([]any)[0].(map[string]any)
	root := admin["RootFolderId"].(string)
	adminID := admin["Id"].(string)
	// The Authentication header picks the acting user.
	if status, out := wdocCall(t, ts, http.MethodGet, "/api/v1/me", "", map[string]string{"Authentication": adminID}); status != http.StatusOK || out["User"] == nil {
		t.Fatalf("GetCurrentUser: %d %v", status, out)
	}
	if status, out := wdocCall(t, ts, http.MethodGet, "/api/v1/me", "", map[string]string{"Authentication": "forged"}); status != http.StatusForbidden || out["Message"] == nil {
		t.Fatalf("bad token: %d %v", status, out)
	}
	folder := wdocOK(t, ts, http.MethodPost, "/api/v1/folders", `{"Name":"Reports","ParentFolderId":"`+root+`"}`)["Metadata"].(map[string]any)["Id"].(string)
	if status, _ := wdocCall(t, ts, http.MethodPost, "/api/v1/folders", `{"Name":"reports","ParentFolderId":"`+root+`"}`, nil); status != http.StatusConflict {
		t.Fatalf("duplicate folder name: %d", status)
	}
	init := wdocOK(t, ts, http.MethodPost, "/api/v1/documents", `{"Name":"q3.txt","ParentFolderId":"`+folder+`","ContentType":"text/html","DocumentSizeInBytes":12}`)
	doc := init["Metadata"].(map[string]any)
	docID := doc["Id"].(string)
	ver := doc["LatestVersionMetadata"].(map[string]any)["Id"].(string)
	upload := init["UploadMetadata"].(map[string]any)["UploadUrl"].(string)
	// Activation before upload, wrong content type and wrong size all fail.
	if status, _ := wdocCall(t, ts, http.MethodPatch, "/api/v1/documents/"+docID+"/versions/"+ver, `{"VersionStatus":"ACTIVE"}`, nil); status != http.StatusConflict {
		t.Fatalf("activate before upload: %d", status)
	}
	if code := wdocPut(t, upload, "text/plain", []byte("<b>hello</b>")); code != http.StatusForbidden {
		t.Fatalf("wrong content type: %d", code)
	}
	if code := wdocPut(t, upload, "text/html", []byte("short")); code != http.StatusBadRequest {
		t.Fatalf("wrong size: %d", code)
	}
	if code := wdocPut(t, upload+"x", "text/html", []byte("<b>hello</b>")); code != http.StatusForbidden {
		t.Fatalf("bad key: %d", code)
	}
	if code := wdocPut(t, upload, "text/html", []byte("<b>hello</b>")); code != http.StatusOK {
		t.Fatalf("upload: %d", code)
	}
	wdocOK(t, ts, http.MethodPatch, "/api/v1/documents/"+docID+"/versions/"+ver, `{"VersionStatus":"ACTIVE"}`)
	got := wdocOK(t, ts, http.MethodGet, "/api/v1/documents/"+docID+"/versions/"+ver+"?fields=SOURCE", "")["Metadata"].(map[string]any)
	if got["Status"] != "ACTIVE" || got["Size"] != float64(12) || got["Signature"] == nil {
		t.Fatalf("GetDocumentVersion: %v", got)
	}
	src := got["Source"].(map[string]any)["ORIGINAL"].(string)
	resp, err := http.Get(src)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "<b>hello</b>" || resp.Header.Get("Content-Type") != "application/octet-stream" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("download: %q %v", body, resp.Header)
	}
	// A second version, delete and restore it.
	init2 := wdocOK(t, ts, http.MethodPost, "/api/v1/documents", `{"Id":"`+docID+`","ContentType":"text/plain"}`)
	v2 := init2["Metadata"].(map[string]any)["LatestVersionMetadata"].(map[string]any)["Id"].(string)
	if v2 != ver { // the active version stays latest until v2 is activated
		t.Fatalf("latest before activation: %s", v2)
	}
	vers := wdocOK(t, ts, http.MethodGet, "/api/v1/documents/"+docID+"/versions?include=INITIALIZED", "")["DocumentVersions"].([]any)
	pending := vers[0].(map[string]any)["Id"].(string)
	wdocPut(t, init2["UploadMetadata"].(map[string]any)["UploadUrl"].(string), "text/plain", []byte("v2"))
	wdocOK(t, ts, http.MethodPatch, "/api/v1/documents/"+docID+"/versions/"+pending, `{"VersionStatus":"ACTIVE"}`)
	wdocOK(t, ts, http.MethodDelete, "/api/v1/documentVersions/"+docID+"/versions/"+ver+"?deletePriorVersions=false", "")
	if status, _ := wdocCall(t, ts, http.MethodDelete, "/api/v1/documentVersions/"+docID+"/versions/"+pending+"?deletePriorVersions=true", "", nil); status != http.StatusConflict {
		t.Fatalf("deleting the last version: %d", status)
	}
	wdocOK(t, ts, http.MethodPost, "/api/v1/documentVersions/restore/"+docID, "")
	if n := len(wdocOK(t, ts, http.MethodGet, "/api/v1/documents/"+docID+"/versions", "")["DocumentVersions"].([]any)); n != 2 {
		t.Fatalf("versions after restore: %d", n)
	}
	// Comments, labels, metadata, sharing, paths and search.
	c := wdocOK(t, ts, http.MethodPost, "/api/v1/documents/"+docID+"/versions/"+ver+"/comment", `{"Text":"Looks good"}`)["Comment"].(map[string]any)
	reply := wdocOK(t, ts, http.MethodPost, "/api/v1/documents/"+docID+"/versions/"+ver+"/comment", `{"Text":"Agreed","ParentId":"`+c["CommentId"].(string)+`","Visibility":"PRIVATE"}`)["Comment"].(map[string]any)
	if reply["ThreadId"] != c["CommentId"] || reply["RecipientId"] != adminID {
		t.Fatalf("reply: %v", reply)
	}
	wdocOK(t, ts, http.MethodPut, "/api/v1/resources/"+docID+"/labels", `{"Labels":["finance","q3"]}`)
	if status, _ := wdocCall(t, ts, http.MethodPut, "/api/v1/resources/"+docID+"/labels", `{"Labels":["a","b","c","d","e","f","g","h","i","j","k","l","m","n","o","p","q","r","s"]}`, nil); status != http.StatusTooManyRequests {
		t.Fatalf("too many labels: %d", status)
	}
	wdocOK(t, ts, http.MethodPut, "/api/v1/resources/"+docID+"/customMetadata", `{"CustomMetadata":{"owner":"finance"}}`)
	if md := wdocOK(t, ts, http.MethodGet, "/api/v1/documents/"+docID+"?includeCustomMetadata=true", ""); md["CustomMetadata"].(map[string]any)["owner"] != "finance" {
		t.Fatalf("GetDocument metadata: %v", md)
	}
	bob := wdocOK(t, ts, http.MethodPost, "/api/v1/users", `{"Username":"bob","GivenName":"Bob","Surname":"B","Password":"s3cret-pass"}`)["User"].(map[string]any)
	if strings.Contains(strings.ToLower(string(mustJSON(t, bob))), "s3cret") {
		t.Fatal("password returned")
	}
	share := wdocOK(t, ts, http.MethodPost, "/api/v1/resources/"+docID+"/permissions", `{"Principals":[{"Id":"`+bob["Id"].(string)+`","Type":"USER","Role":"VIEWER"},{"Id":"ghost","Type":"USER","Role":"VIEWER"}]}`)
	results := share["ShareResults"].([]any)
	if results[0].(map[string]any)["Status"] != "SUCCESS" || results[1].(map[string]any)["Status"] != "FAILURE" {
		t.Fatalf("share results: %v", results)
	}
	if docs := wdocOK(t, ts, http.MethodGet, "/api/v1/resources?userId="+bob["Id"].(string)+"&collectionType=SHARED_WITH_ME", "")["Documents"].([]any); len(docs) != 1 {
		t.Fatalf("GetResources shared: %v", docs)
	}
	path := wdocOK(t, ts, http.MethodGet, "/api/v1/documents/"+docID+"/path?fields=NAME", "")["Path"].(map[string]any)["Components"].([]any)
	if len(path) != 3 || path[1].(map[string]any)["Name"] != "Reports" {
		t.Fatalf("GetDocumentPath: %v", path)
	}
	items := wdocOK(t, ts, http.MethodPost, "/api/v1/search", `{"QueryText":"q3","Filters":{"Labels":["finance"],"ResourceTypes":["DOCUMENT"]}}`)["Items"].([]any)
	if len(items) != 1 {
		t.Fatalf("SearchResources: %v", items)
	}
	acts := wdocOK(t, ts, http.MethodGet, "/api/v1/activities?resourceId="+docID, "")["UserActivities"].([]any)
	if len(acts) < 3 {
		t.Fatalf("DescribeActivities: %v", acts)
	}
	// Moving a folder under itself is refused; deleting it refunds storage.
	sub := wdocOK(t, ts, http.MethodPost, "/api/v1/folders", `{"Name":"Sub","ParentFolderId":"`+folder+`"}`)["Metadata"].(map[string]any)["Id"].(string)
	if status, _ := wdocCall(t, ts, http.MethodPatch, "/api/v1/folders/"+folder, `{"ParentFolderId":"`+sub+`"}`, nil); status != http.StatusConflict {
		t.Fatalf("move into itself: %d", status)
	}
	wdocOK(t, ts, http.MethodDelete, "/api/v1/folders/"+folder, "")
	if status, _ := wdocCall(t, ts, http.MethodGet, "/api/v1/documents/"+docID, "", nil); status != http.StatusNotFound {
		t.Fatalf("document after folder delete: %d", status)
	}
	srv.workDocs.mu.Lock()
	used, stored := srv.workDocs.users[adminID].used, srv.workDocs.bytes
	srv.workDocs.mu.Unlock()
	if used != 0 || stored != 0 {
		t.Fatalf("storage not refunded: used %d, stored %d", used, stored)
	}
}

func TestWorkDocsUsersAndSubscriptions(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	u := wdocOK(t, ts, http.MethodPost, "/api/v1/users", `{"Username":"carol","GivenName":"Carol","Surname":"C","Password":"abcd1234"}`)["User"].(map[string]any)
	id := u["Id"].(string)
	if status, _ := wdocCall(t, ts, http.MethodPost, "/api/v1/users", `{"Username":"Carol","GivenName":"C","Surname":"C","Password":"abcd1234"}`, nil); status != http.StatusConflict {
		t.Fatalf("duplicate user: %d", status)
	}
	if out := wdocOK(t, ts, http.MethodPatch, "/api/v1/users/"+id, `{"GrantPoweruserPrivileges":"TRUE"}`); out["User"].(map[string]any)["Type"] != "POWERUSER" {
		t.Fatalf("UpdateUser: %v", out)
	}
	wdocOK(t, ts, http.MethodDelete, "/api/v1/users/"+id+"/activation", "")
	if status, _ := wdocCall(t, ts, http.MethodGet, "/api/v1/me", "", map[string]string{"Authentication": id}); status != http.StatusForbidden {
		t.Fatalf("inactive user token: %d", status)
	}
	if n := wdocOK(t, ts, http.MethodGet, "/api/v1/users?include=ACTIVE_PENDING", "")["TotalNumberOfUsers"]; n != float64(1) {
		t.Fatalf("active users: %v", n)
	}
	wdocOK(t, ts, http.MethodPost, "/api/v1/users/"+id+"/activation", "")
	wdocOK(t, ts, http.MethodDelete, "/api/v1/users/"+id, "")
	sub := wdocOK(t, ts, http.MethodPost, "/api/v1/organizations/"+wdocOrgID+"/subscriptions", `{"Endpoint":"https://hooks.example.com/wd","Protocol":"HTTPS","SubscriptionType":"ALL"}`)["Subscription"].(map[string]any)
	if status, _ := wdocCall(t, ts, http.MethodPost, "/api/v1/organizations/"+wdocOrgID+"/subscriptions", `{"Endpoint":"http://plain","Protocol":"HTTPS","SubscriptionType":"ALL"}`, nil); status != http.StatusBadRequest {
		t.Fatalf("non-https endpoint: %d", status)
	}
	if status, _ := wdocCall(t, ts, http.MethodGet, "/api/v1/organizations/d-0000000000/subscriptions", "", nil); status != http.StatusNotFound {
		t.Fatalf("unknown organization: %d", status)
	}
	wdocOK(t, ts, http.MethodDelete, "/api/v1/organizations/"+wdocOrgID+"/subscriptions/"+sub["SubscriptionId"].(string), "")
	if subs := wdocOK(t, ts, http.MethodGet, "/api/v1/organizations/"+wdocOrgID+"/subscriptions", "")["Subscriptions"].([]any); len(subs) != 0 {
		t.Fatalf("subscriptions: %v", subs)
	}
	if groups := wdocOK(t, ts, http.MethodGet, "/api/v1/groups?searchQuery=domain", "")["Groups"].([]any); len(groups) != 2 {
		t.Fatalf("DescribeGroups: %v", groups)
	}
	// Chunked bodies and oversized uploads are refused.
	req, _ := http.NewRequest(http.MethodPut, ts.URL+wdocUploadPath+"x/y?key=z", io.MultiReader(strings.NewReader("data")))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked upload: %d", resp.StatusCode)
	}
}
