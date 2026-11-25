package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	pszTestRole = "arn:aws:iam::123456789012:role/personalize"
	pszInterSch = `{"type":"record","name":"Interactions","namespace":"com.amazonaws.personalize.schema","fields":[{"name":"USER_ID","type":"string"},{"name":"ITEM_ID","type":"string"},{"name":"EVENT_TYPE","type":"string"},{"name":"TIMESTAMP","type":"long"}],"version":"1.0"}`
	pszItemsSch = `{"type":"record","name":"Items","namespace":"com.amazonaws.personalize.schema","fields":[{"name":"ITEM_ID","type":"string"},{"name":"GENRE","type":["null","string"],"categorical":true}],"version":"1.0"}`
)

func pszJSON(v string) string { return fmt.Sprintf("%q", v) }

// pszSettle describes arn until it reports want (async resources settle on read).
func pszSettle(t *testing.T, ts *httptest.Server, op, key, arnKey, arn, want string) map[string]any {
	t.Helper()
	for i := 0; i < 4; i++ {
		obj, _ := pszOK(t, ts, op, fmt.Sprintf(`{%q:%q}`, arnKey, arn))[key].(map[string]any)
		if obj["status"] == want {
			return obj
		}
	}
	t.Fatalf("%s never reached %s", arn, want)
	return nil
}

// pszFixture builds a dataset group with interactions and items, an event
// tracker and an ACTIVE user-personalization campaign.
func pszFixture(t *testing.T, srv *Server, ts *httptest.Server) map[string]string {
	t.Helper()
	fx := map[string]string{}
	dg := pszOK(t, ts, "CreateDatasetGroup", `{"name":"shop"}`)["datasetGroupArn"].(string)
	if obj := pszOK(t, ts, "DescribeDatasetGroup", fmt.Sprintf(`{"datasetGroupArn":%q}`, dg))["datasetGroup"].(map[string]any); obj["status"] != pszStatusCreate {
		t.Fatalf("first read: %v", obj)
	}
	pszSettle(t, ts, "DescribeDatasetGroup", "datasetGroup", "datasetGroupArn", dg, "ACTIVE")
	isch := pszOK(t, ts, "CreateSchema", `{"name":"inter","schema":`+pszJSON(pszInterSch)+`}`)["schemaArn"].(string)
	itsch := pszOK(t, ts, "CreateSchema", `{"name":"items","schema":`+pszJSON(pszItemsSch)+`}`)["schemaArn"].(string)
	inter := pszOK(t, ts, "CreateDataset", fmt.Sprintf(`{"name":"i","schemaArn":%q,"datasetGroupArn":%q,"datasetType":"Interactions"}`, isch, dg))["datasetArn"].(string)
	items := pszOK(t, ts, "CreateDataset", fmt.Sprintf(`{"name":"it","schemaArn":%q,"datasetGroupArn":%q,"datasetType":"Items"}`, itsch, dg))["datasetArn"].(string)
	pszSettle(t, ts, "DescribeDataset", "dataset", "datasetArn", inter, "ACTIVE")
	pszSettle(t, ts, "DescribeDataset", "dataset", "datasetArn", items, "ACTIVE")

	if _, err := srv.s3.CreateBucket("psz-data"); err != nil {
		t.Fatal(err)
	}
	csv := "USER_ID,ITEM_ID,EVENT_TYPE,TIMESTAMP\nu1,i1,click,1700000000\nu1,i2,click,1700000001\nu2,i1,purchase,1700000002\nu2,i3,click,1700000003\n"
	if _, err := srv.s3.PutObject("psz-data", "interactions.csv", "text/csv", []byte(csv), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	job := pszOK(t, ts, "CreateDatasetImportJob", fmt.Sprintf(`{"jobName":"imp","datasetArn":%q,"dataSource":{"dataLocation":"s3://psz-data/interactions.csv"},"roleArn":%q}`, inter, pszTestRole))["datasetImportJobArn"].(string)
	pszSettle(t, ts, "DescribeDatasetImportJob", "datasetImportJob", "datasetImportJobArn", job, "ACTIVE")
	bad := pszOK(t, ts, "CreateDatasetImportJob", fmt.Sprintf(`{"jobName":"imp-missing","datasetArn":%q,"dataSource":{"dataLocation":"s3://psz-data/none.csv"}}`, inter))["datasetImportJobArn"].(string)
	if obj := pszSettle(t, ts, "DescribeDatasetImportJob", "datasetImportJob", "datasetImportJobArn", bad, "CREATE FAILED"); obj["failureReason"] == "" {
		t.Fatalf("failed import has no reason: %v", obj)
	}

	tracker := pszOK(t, ts, "CreateEventTracker", fmt.Sprintf(`{"name":"web","datasetGroupArn":%q}`, dg))
	fx["trackingId"] = tracker["trackingId"].(string)
	sol := pszOK(t, ts, "CreateSolution", fmt.Sprintf(`{"name":"up","datasetGroupArn":%q,"recipeArn":"arn:aws:personalize:::recipe/aws-user-personalization"}`, dg))["solutionArn"].(string)
	pszSettle(t, ts, "DescribeSolution", "solution", "solutionArn", sol, "ACTIVE")
	ver := pszOK(t, ts, "CreateSolutionVersion", fmt.Sprintf(`{"solutionArn":%q}`, sol))["solutionVersionArn"].(string)
	if code := pszErrCode(t, ts, "CreateCampaign", fmt.Sprintf(`{"name":"early","solutionVersionArn":%q}`, ver)); code != pszErrInUse {
		t.Fatalf("campaign on a training version: %s", code)
	}
	pszSettle(t, ts, "DescribeSolutionVersion", "solutionVersion", "solutionVersionArn", ver, "ACTIVE")
	camp := pszOK(t, ts, "CreateCampaign", fmt.Sprintf(`{"name":"home","solutionVersionArn":%q,"campaignConfig":{"enableMetadataWithRecommendations":true}}`, ver))["campaignArn"].(string)
	pszSettle(t, ts, "DescribeCampaign", "campaign", "campaignArn", camp, "ACTIVE")
	for k, v := range map[string]string{"group": dg, "interactions": inter, "items": items, "solution": sol, "version": ver, "campaign": camp} {
		fx[k] = v
	}
	return fx
}

func pszItemIDs(t *testing.T, out map[string]any, key string) []string {
	t.Helper()
	list, _ := out[key].([]any)
	ids := make([]string, 0, len(list))
	for _, v := range list {
		ids = append(ids, v.(map[string]any)["itemId"].(string))
	}
	return ids
}

func TestPersonalizeStage1RecommendationsFollowData(t *testing.T) {
	srv, ts := newPersonalizeTestServer(t)
	fx := pszFixture(t, srv, ts)

	status, out := pszREST(t, ts, "/items", fmt.Sprintf(`{"datasetArn":%q,"items":[{"itemId":"i4","properties":"{\"genre\":\"drama\"}"},{"itemId":"i1","properties":"{\"genre\":\"comedy\"}"}]}`, fx["items"]))
	if status != http.StatusOK {
		t.Fatalf("PutItems: %d %v", status, out)
	}
	if status, out := pszREST(t, ts, "/items", fmt.Sprintf(`{"datasetArn":%q,"items":[{"itemId":"i5","properties":"{\"color\":\"red\"}"}]}`, fx["items"])); status != http.StatusBadRequest {
		t.Fatalf("PutItems with an unknown property: %d %v", status, out)
	}
	if status, out := pszREST(t, ts, "/items", fmt.Sprintf(`{"datasetArn":%q,"items":[{"itemId":"i5"}]}`, fx["interactions"])); status != http.StatusBadRequest {
		t.Fatalf("PutItems on an Interactions dataset: %d %v", status, out)
	}
	if status, out := pszREST(t, ts, "/events", fmt.Sprintf(`{"trackingId":%q,"userId":"u3","sessionId":"s1","eventList":[{"eventType":"click","itemId":"i6","sentAt":1700000100}]}`, fx["trackingId"])); status != http.StatusOK {
		t.Fatalf("PutEvents: %d %v", status, out)
	}

	req := fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","numResults":10,"metadataColumns":{"ITEMS":["GENRE"]}}`, fx["campaign"])
	status, first := pszREST(t, ts, "/recommendations", req)
	if status != http.StatusOK {
		t.Fatalf("GetRecommendations: %d %v", status, first)
	}
	ids := pszItemIDs(t, first, "itemList")
	if strings.Join(ids, ",") == "" || !strings.Contains(strings.Join(ids, ","), "i6") || !strings.Contains(strings.Join(ids, ","), "i4") {
		t.Fatalf("recommendations should cover imported, put and streamed items: %v", ids)
	}
	_, again := pszREST(t, ts, "/recommendations", req)
	if strings.Join(pszItemIDs(t, again, "itemList"), ",") != strings.Join(ids, ",") {
		t.Fatalf("recommendations aren't deterministic: %v vs %v", ids, pszItemIDs(t, again, "itemList"))
	}
	for _, v := range first["itemList"].([]any) {
		e := v.(map[string]any)
		if e["itemId"] == "i4" && e["metadata"].(map[string]any)["GENRE"] != "drama" {
			t.Fatalf("metadata missing: %v", e)
		}
	}

	filter := pszOK(t, ts, "CreateFilter", fmt.Sprintf(`{"name":"no-genre","datasetGroupArn":%q,"filterExpression":"EXCLUDE ItemID WHERE Items.GENRE IN ($G)"}`, fx["group"]))["filterArn"].(string)
	pszSettle(t, ts, "DescribeFilter", "filter", "filterArn", filter, "ACTIVE")
	_, filtered := pszREST(t, ts, "/recommendations", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","filterArn":%q,"filterValues":{"G":"\"drama\""}}`, fx["campaign"], filter))
	if got := strings.Join(pszItemIDs(t, filtered, "itemList"), ","); strings.Contains(got, "i4") || got == "" {
		t.Fatalf("filter didn't exclude drama: %s", got)
	}
	if status, _ := pszREST(t, ts, "/recommendations", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","filterArn":%q}`, fx["campaign"], filter)); status != http.StatusBadRequest {
		t.Fatalf("missing filter value: %d", status)
	}
	if code := pszErrCode(t, ts, "CreateFilter", fmt.Sprintf(`{"name":"bad","datasetGroupArn":%q,"filterExpression":"EXCLUDE ItemID WHERE Users.AGE > 3"}`, fx["group"])); code != pszErrInvalid {
		t.Fatalf("filter on a missing dataset: %s", code)
	}
	if status, _ := pszREST(t, ts, "/personalize-ranking", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","inputList":["i1"]}`, fx["campaign"])); status != http.StatusBadRequest {
		t.Fatalf("ranking on a user-personalization campaign: %d", status)
	}

	// Data deletion removes a user's interactions.
	if _, err := srv.s3.PutObject("psz-data", "delete.csv", "text/csv", []byte("USER_ID\nu2\n"), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	del := pszOK(t, ts, "CreateDataDeletionJob", fmt.Sprintf(`{"jobName":"forget","datasetGroupArn":%q,"dataSource":{"dataLocation":"s3://psz-data/delete.csv"},"roleArn":%q}`, fx["group"], pszTestRole))["dataDeletionJobArn"].(string)
	if obj := pszSettle(t, ts, "DescribeDataDeletionJob", "dataDeletionJob", "dataDeletionJobArn", del, "COMPLETED"); obj["numDeleted"] != float64(1) {
		t.Fatalf("deletion: %v", obj)
	}
	exp := pszOK(t, ts, "CreateDatasetExportJob", fmt.Sprintf(`{"jobName":"dump","datasetArn":%q,"ingestionMode":"ALL","roleArn":%q,"jobOutput":{"s3DataDestination":{"path":"s3://psz-data/export/"}}}`, fx["interactions"], pszTestRole))["datasetExportJobArn"].(string)
	pszSettle(t, ts, "DescribeDatasetExportJob", "datasetExportJob", "datasetExportJobArn", exp, "ACTIVE")
	obj, err := srv.s3.GetObject("psz-data", "export/dump.csv")
	if err != nil || strings.Contains(string(obj.Body), "u2") || !strings.Contains(string(obj.Body), "u3,i6,click") {
		t.Fatalf("export: %v %q", err, obj.Body)
	}
}

func TestPersonalizeStage1LifecycleErrors(t *testing.T) {
	srv, ts := newPersonalizeTestServer(t)
	fx := pszFixture(t, srv, ts)
	if code := pszErrCode(t, ts, "CreateDatasetGroup", `{"name":"shop"}`); code != pszErrExists {
		t.Fatalf("duplicate group: %s", code)
	}
	if code := pszErrCode(t, ts, "DeleteDatasetGroup", fmt.Sprintf(`{"datasetGroupArn":%q}`, fx["group"])); code != pszErrInUse {
		t.Fatalf("delete group in use: %s", code)
	}
	if code := pszErrCode(t, ts, "DeleteSolution", fmt.Sprintf(`{"solutionArn":%q}`, fx["solution"])); code != pszErrInUse {
		t.Fatalf("delete solution with a campaign: %s", code)
	}
	if code := pszErrCode(t, ts, "ListDatasetGroups", `{"nextToken":"garbage"}`); code != pszErrToken {
		t.Fatalf("bad token: %s", code)
	}
	if code := pszErrCode(t, ts, "CreateEventTracker", fmt.Sprintf(`{"name":"second","datasetGroupArn":%q}`, fx["group"])); code != pszErrLimit {
		t.Fatalf("second tracker: %s", code)
	}
	if code := pszErrCode(t, ts, "DescribeCampaign", `{"campaignArn":"arn:aws:personalize:us-east-1:123456789012:campaign/none"}`); code != pszErrNotFound {
		t.Fatalf("missing campaign: %s", code)
	}
	// A stopped version settles to CREATE STOPPED.
	ver := pszOK(t, ts, "CreateSolutionVersion", fmt.Sprintf(`{"solutionArn":%q,"name":"v2"}`, fx["solution"]))["solutionVersionArn"].(string)
	pszOK(t, ts, "StopSolutionVersionCreation", fmt.Sprintf(`{"solutionVersionArn":%q}`, ver))
	pszSettle(t, ts, "DescribeSolutionVersion", "solutionVersion", "solutionVersionArn", ver, "CREATE STOPPED")
	if code := pszErrCode(t, ts, "StopSolutionVersionCreation", fmt.Sprintf(`{"solutionVersionArn":%q}`, ver)); code != pszErrInUse {
		t.Fatalf("stop a stopped version: %s", code)
	}
	// Paging and tags.
	for i := 0; i < 3; i++ {
		pszOK(t, ts, "CreateDatasetGroup", fmt.Sprintf(`{"name":"g%d","tags":[{"tagKey":"team","tagValue":"ml"}]}`, i))
	}
	page := pszOK(t, ts, "ListDatasetGroups", `{"maxResults":2}`)
	rest := pszOK(t, ts, "ListDatasetGroups", fmt.Sprintf(`{"maxResults":2,"nextToken":%q}`, page["nextToken"]))
	if len(page["datasetGroups"].([]any))+len(rest["datasetGroups"].([]any)) != 4 {
		t.Fatalf("paging: %v %v", page, rest)
	}
	g0 := pszARNBase + "dataset-group/g0"
	pszOK(t, ts, "TagResource", fmt.Sprintf(`{"resourceArn":%q,"tags":[{"tagKey":"env","tagValue":"dev"}]}`, g0))
	pszOK(t, ts, "UntagResource", fmt.Sprintf(`{"resourceArn":%q,"tagKeys":["team"]}`, g0))
	tags := pszOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"resourceArn":%q}`, g0))["tags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["tagKey"] != "env" {
		t.Fatalf("tags: %v", tags)
	}
	if code := pszErrCode(t, ts, "TagResource", fmt.Sprintf(`{"resourceArn":%q,"tags":[{"tagKey":"aws:x","tagValue":"y"}]}`, g0)); code != pszErrInvalid {
		t.Fatalf("aws: tag: %s", code)
	}
	recipes := pszOK(t, ts, "ListRecipes", `{"domain":"ECOMMERCE"}`)["recipes"].([]any)
	if len(recipes) != 5 {
		t.Fatalf("ECOMMERCE recipes: %d", len(recipes))
	}
}
