package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const smfsTestGroup = `{"FeatureGroupName":"customers","RecordIdentifierFeatureName":"customer_id","EventTimeFeatureName":"event_time",
 "FeatureDefinitions":[{"FeatureName":"customer_id","FeatureType":"String"},{"FeatureName":"event_time","FeatureType":"Fractional"},
  {"FeatureName":"age","FeatureType":"Integral"},{"FeatureName":"score","FeatureType":"Fractional"},
  {"FeatureName":"tags","FeatureType":"String","CollectionType":"Set"},
  {"FeatureName":"embedding","FeatureType":"Fractional","CollectionType":"Vector","CollectionConfig":{"VectorConfig":{"Dimension":2}}}],
 "OnlineStoreConfig":{"EnableOnlineStore":true,"StorageType":"InMemory"}}`

func smfsRecordJSON(id string, at float64, extra string) string {
	return fmt.Sprintf(`{"Record":[{"FeatureName":"customer_id","ValueAsString":%q},{"FeatureName":"event_time","ValueAsString":"%v"}%s]}`, id, at, extra)
}

func smfsGet(t *testing.T, ts *httptest.Server, query string) map[string]any {
	t.Helper()
	status, out := smData(t, ts, http.MethodGet, "/FeatureGroup/customers?"+query, "")
	if status != http.StatusOK {
		t.Fatalf("GetRecord %s: %d %v", query, status, out)
	}
	return out
}

func TestSageMakerFeatureStoreRuntimeStage1Records(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)
	arn := smOK(t, ts, "CreateFeatureGroup", smfsTestGroup)["FeatureGroupArn"].(string)
	if status, out := smCall(t, ts, "CreateFeatureGroup", smfsTestGroup); status != http.StatusBadRequest || out["__type"] != "ResourceInUse" {
		t.Fatalf("duplicate group: %d %v", status, out)
	}
	bad := strings.Replace(smfsTestGroup, `"EventTimeFeatureName":"event_time"`, `"EventTimeFeatureName":"age"`, 1)
	bad = strings.Replace(bad, `"customers"`, `"other"`, 1)
	if status, out := smCall(t, ts, "CreateFeatureGroup", bad); status != http.StatusBadRequest || out["__type"] != "ValidationException" {
		t.Fatalf("Integral event time: %d %v", status, out)
	}
	if out := smOK(t, ts, "DescribeFeatureGroup", fmt.Sprintf(`{"FeatureGroupName":%q}`, arn)); out["RecordIdentifierFeatureName"] != "customer_id" {
		t.Fatalf("describe by ARN: %v", out)
	}

	put := func(body string) (int, map[string]any) {
		return smData(t, ts, http.MethodPut, "/FeatureGroup/customers", body)
	}
	if status, out := put(smfsRecordJSON("c1", 1700000000, `,{"FeatureName":"age","ValueAsString":"41"},{"FeatureName":"tags","ValueAsStringList":["a","b"]},{"FeatureName":"embedding","ValueAsStringList":["0.1","0.2"]}`)); status != http.StatusOK {
		t.Fatalf("PutRecord: %d %v", status, out)
	}
	for name, body := range map[string]string{
		"undefined feature": smfsRecordJSON("c2", 1, `,{"FeatureName":"color","ValueAsString":"red"}`),
		"bad integral":      smfsRecordJSON("c2", 1, `,{"FeatureName":"age","ValueAsString":"old"}`),
		"duplicate in set":  smfsRecordJSON("c2", 1, `,{"FeatureName":"tags","ValueAsStringList":["a","a"]}`),
		"vector dimension":  smfsRecordJSON("c2", 1, `,{"FeatureName":"embedding","ValueAsStringList":["1"]}`),
		"no identifier":     `{"Record":[{"FeatureName":"event_time","ValueAsString":"1"}]}`,
		"bad event time":    `{"Record":[{"FeatureName":"customer_id","ValueAsString":"c2"},{"FeatureName":"event_time","ValueAsString":"yesterday"}]}`,
		"offline target":    `{"Record":[{"FeatureName":"customer_id","ValueAsString":"c2"},{"FeatureName":"event_time","ValueAsString":"1"}],"TargetStores":["OfflineStore"]}`,
	} {
		if status, out := put(body); status != http.StatusBadRequest || out["__type"] != "ValidationError" {
			t.Errorf("%s: %d %v", name, status, out)
		}
	}
	// An older event is history and leaves the online record alone.
	if status, _ := put(smfsRecordJSON("c1", 1600000000, `,{"FeatureName":"age","ValueAsString":"20"}`)); status != http.StatusOK {
		t.Fatalf("historic put: %d", status)
	}
	out := smfsGet(t, ts, "RecordIdentifierValueAsString=c1")
	if got := smJSON(out["Record"]); !strings.Contains(got, `"ValueAsString":"41"`) || !strings.Contains(got, `"ValueAsStringList":["a","b"]`) {
		t.Fatalf("GetRecord: %s", got)
	}
	out = smfsGet(t, ts, "RecordIdentifierValueAsString=c1&FeatureName=age")
	if got := smJSON(out["Record"]); got != `[{"FeatureName":"age","ValueAsString":"41"}]` {
		t.Fatalf("projected GetRecord: %s", got)
	}
	if status, _ := smData(t, ts, http.MethodGet, "/FeatureGroup/customers?RecordIdentifierValueAsString=c1&FeatureName=nope", ""); status != http.StatusBadRequest {
		t.Fatalf("unknown FeatureName: %d", status)
	}
	if out := smfsGet(t, ts, "RecordIdentifierValueAsString=missing"); len(out) != 0 {
		t.Fatalf("missing record: %v", out)
	}

	// UpdateRecord merges, refuses older event times and missing records.
	update := func(body string) (int, map[string]any) {
		return smData(t, ts, http.MethodPost, "/FeatureGroup/customers/Record", body)
	}
	if status, out := update(`{"RecordIdentifierValueAsString":"c1","Features":[{"FeatureName":"score","ValueAsString":"0.5"},{"FeatureName":"age"}]}`); status != http.StatusOK {
		t.Fatalf("UpdateRecord: %d %v", status, out)
	}
	if got := smJSON(smfsGet(t, ts, "RecordIdentifierValueAsString=c1")["Record"]); !strings.Contains(got, `"score","ValueAsString":"0.5"`) || strings.Contains(got, `"age"`) {
		t.Fatalf("after update: %s", got)
	}
	if status, out := update(`{"RecordIdentifierValueAsString":"c1","Features":[{"FeatureName":"event_time","ValueAsString":"5"}]}`); status != http.StatusConflict || out["__type"] != "ConflictException" {
		t.Fatalf("older event time: %d %v", status, out)
	}
	if status, out := update(`{"RecordIdentifierValueAsString":"c1","Features":[{"FeatureName":"score","ValueAsString":"1"}],"TtlDuration":{"Unit":"Days","Value":1}}`); status != http.StatusBadRequest {
		t.Fatalf("TTL without event time: %d %v", status, out)
	}
	if status, out := update(`{"RecordIdentifierValueAsString":"zz","Features":[{"FeatureName":"score","ValueAsString":"1"}]}`); status != http.StatusNotFound || out["__type"] != "ResourceNotFound" {
		t.Fatalf("update missing: %d %v", status, out)
	}

	// BatchWriteRecord reports bad entries; BatchGetRecord reports missing records.
	status, out := smData(t, ts, http.MethodPost, "/BatchWriteRecord", `{"Entries":[
		{"FeatureGroupName":"customers","Record":[{"FeatureName":"customer_id","ValueAsString":"c3"},{"FeatureName":"event_time","ValueAsString":"1700000001"}]},
		{"FeatureGroupName":"customers","Record":[{"FeatureName":"customer_id","ValueAsString":"c4"}]},
		{"FeatureGroupName":"nope","Record":[{"FeatureName":"customer_id","ValueAsString":"c5"}]}]}`)
	if errs, _ := out["Errors"].([]any); status != http.StatusOK || len(errs) != 2 || !strings.Contains(smJSON(errs), `"ErrorCode":"ResourceNotFound"`) {
		t.Fatalf("BatchWriteRecord: %d %v", status, out)
	}
	status, out = smData(t, ts, http.MethodPost, "/BatchGetRecord", `{"Identifiers":[{"FeatureGroupName":"customers","RecordIdentifiersValueAsString":["c1","c3","c9"],"FeatureNames":["customer_id"]}]}`)
	if recs, _ := out["Records"].([]any); status != http.StatusOK || len(recs) != 2 || len(out["Errors"].([]any)) != 1 {
		t.Fatalf("BatchGetRecord: %d %v", status, out)
	}

	// Soft deletes hide the record but ListRecords can still show it; a hard
	// delete with an older event time fails.
	if status, out := smData(t, ts, http.MethodDelete, "/FeatureGroup/customers?RecordIdentifierValueAsString=c3&EventTime=1700000002", ""); status != http.StatusOK {
		t.Fatalf("DeleteRecord: %d %v", status, out)
	}
	if out := smfsGet(t, ts, "RecordIdentifierValueAsString=c3"); len(out) != 0 {
		t.Fatalf("soft-deleted record: %v", out)
	}
	_, out = smData(t, ts, http.MethodPost, "/FeatureGroup/customers/ListRecords", `{}`)
	if got := smJSON(out["RecordIdentifiers"]); got != `["c1"]` {
		t.Fatalf("ListRecords: %s", got)
	}
	_, out = smData(t, ts, http.MethodPost, "/FeatureGroup/customers/ListRecords", `{"IncludeSoftDeletedRecords":true,"MaxResults":1}`)
	if got := smJSON(out["RecordIdentifiers"]); got != `["c1"]` || out["NextToken"] != "1" {
		t.Fatalf("ListRecords page: %v", out)
	}
	if status, _ := smData(t, ts, http.MethodDelete, "/FeatureGroup/customers?RecordIdentifierValueAsString=c1&EventTime=1&DeletionMode=HardDelete", ""); status != http.StatusBadRequest {
		t.Fatalf("hard delete with older event time: %d", status)
	}

	// A record whose TTL has passed is gone.
	if status, _ := put(`{"Record":[{"FeatureName":"customer_id","ValueAsString":"old"},{"FeatureName":"event_time","ValueAsString":"1000"}],"TtlDuration":{"Unit":"Seconds","Value":1}}`); status != http.StatusOK {
		t.Fatalf("put with TTL: %d", status)
	}
	if out := smfsGet(t, ts, "RecordIdentifierValueAsString=old"); len(out) != 0 {
		t.Fatalf("expired record: %v", out)
	}
	now := time.Now().Unix()
	put(fmt.Sprintf(`{"Record":[{"FeatureName":"customer_id","ValueAsString":"fresh"},{"FeatureName":"event_time","ValueAsString":"%d"}],"TtlDuration":{"Unit":"Days","Value":1}}`, now))
	if out := smfsGet(t, ts, "RecordIdentifierValueAsString=fresh&ExpirationTimeResponse=Enabled"); out["ExpiresAt"] != time.Unix(now, 0).UTC().Add(24*time.Hour).Format(time.RFC3339) {
		t.Fatalf("ExpiresAt: %v", out)
	}

	// Deleting the group drops its records and their bytes.
	smOK(t, ts, "DeleteFeatureGroup", `{"FeatureGroupName":"customers"}`)
	if status, _ := smData(t, ts, http.MethodGet, "/FeatureGroup/customers?RecordIdentifierValueAsString=c1", ""); status != http.StatusNotFound {
		t.Fatalf("record of a deleted group: %d", status)
	}
	srv.sagemaker.mu.Lock()
	bytes := srv.sagemaker.smfsBytes
	srv.sagemaker.mu.Unlock()
	if bytes != 0 {
		t.Fatalf("stored bytes after delete: %d", bytes)
	}
}

func TestSageMakerFeatureStoreRuntimeStage1ControlPlane(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	smOK(t, ts, "CreateFeatureGroup", smfsTestGroup)
	offline := strings.Replace(smfsTestGroup, `"customers"`, `"offline-only"`, 1)
	offline = strings.Replace(offline, `"OnlineStoreConfig":{"EnableOnlineStore":true,"StorageType":"InMemory"}`,
		`"OfflineStoreConfig":{"S3StorageConfig":{"S3Uri":"s3://bucket/fs"}},"RoleArn":"arn:aws:iam::123456789012:role/fs"`, 1)
	smOK(t, ts, "CreateFeatureGroup", offline)
	out := smOK(t, ts, "DescribeFeatureGroup", `{"FeatureGroupName":"offline-only"}`)
	if got := smJSON(out["OfflineStoreConfig"]); !strings.Contains(got, `"ResolvedOutputS3Uri":"s3://bucket/fs/123456789012/sagemaker/us-east-1/offline-store/offline-only-`) || !strings.Contains(got, `"Database":"sagemaker_featurestore"`) {
		t.Fatalf("offline config: %s", got)
	}
	// An offline-only group has no online store to read.
	if status, out := smData(t, ts, http.MethodGet, "/FeatureGroup/offline-only?RecordIdentifierValueAsString=1", ""); status != http.StatusBadRequest {
		t.Fatalf("GetRecord on an offline-only group: %d %v", status, out)
	}
	smOK(t, ts, "UpdateFeatureGroup", `{"FeatureGroupName":"customers","FeatureAdditions":[{"FeatureName":"city","FeatureType":"String"}],"OnlineStoreConfig":{"TtlDuration":{"Unit":"Hours","Value":2}}}`)
	out = smOK(t, ts, "DescribeFeatureGroup", `{"FeatureGroupName":"customers"}`)
	if !strings.Contains(smJSON(out["FeatureDefinitions"]), `"city"`) || !strings.Contains(smJSON(out["OnlineStoreConfig"]), `"Unit":"Hours"`) || out["LastUpdateStatus"] == nil {
		t.Fatalf("after update: %v", out)
	}
	if status, _ := smCall(t, ts, "UpdateFeatureGroup", `{"FeatureGroupName":"customers","FeatureAdditions":[{"FeatureName":"CITY","FeatureType":"String"}]}`); status != http.StatusBadRequest {
		t.Fatalf("duplicate feature (case-insensitive): %d", status)
	}
	out = smOK(t, ts, "ListFeatureGroups", `{"SortBy":"Name","SortOrder":"Ascending","MaxResults":1}`)
	if got := smJSON(out["FeatureGroupSummaries"]); !strings.Contains(got, `"customers"`) || out["NextToken"] != "1" {
		t.Fatalf("ListFeatureGroups: %v", out)
	}
	if out := smOK(t, ts, "ListFeatureGroups", `{"NameContains":"OFFLINE"}`); len(out["FeatureGroupSummaries"].([]any)) != 1 {
		t.Fatalf("NameContains: %v", out)
	}
	if status, out := smCall(t, ts, "DescribeFeatureGroup", `{"FeatureGroupName":"missing"}`); status != http.StatusBadRequest || out["__type"] != "ResourceNotFound" {
		t.Fatalf("missing group: %d %v", status, out)
	}
	if status, out := smCall(t, ts, "CreateFeatureGroup", `{"FeatureGroupName":"-bad"}`); status != http.StatusBadRequest || out["__type"] != "ValidationException" {
		t.Fatalf("model validation: %d %v", status, out)
	}
}

// Caller-controlled growth is bounded: records per group and BatchGetRecord
// response bytes.
func TestSageMakerFeatureStoreRuntimeStage1Limits(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)
	smOK(t, ts, "CreateFeatureGroup", smfsTestGroup)
	st := srv.sagemaker
	st.mu.Lock()
	defer st.mu.Unlock()
	val := func(s string) *string { return &s }
	write := func(id string) *sagemakerDataError {
		return st.smfsPutLocked("customers", []smfsValue{{FeatureName: "customer_id", ValueAsString: val(id)},
			{FeatureName: "event_time", ValueAsString: val("1")}}, nil, nil, time.Now())
	}
	for i := 0; i < smfsMaxRecordsPerGroup; i++ {
		if err := write(fmt.Sprint(i)); err != nil {
			t.Fatalf("record %d: %v", i, err.msg)
		}
	}
	if err := write("one-too-many"); err == nil || err.code != "ValidationError" {
		t.Fatalf("record cap not enforced: %v", err)
	}
	// Overwriting an existing record is still allowed at the cap.
	if err := write("0"); err != nil {
		t.Fatalf("overwrite at cap: %v", err.msg)
	}
	big := strings.Repeat("x", 400<<10)
	g := st.featureGroups["customers"]
	for i := 0; i < 30; i++ {
		id := fmt.Sprint("big", i)
		g.records[id] = &smfsRecord{id: id, values: []smfsValue{{FeatureName: "customer_id", ValueAsString: &big}}, bytes: len(big)}
	}
	ids := make([]string, 30)
	for i := range ids {
		ids[i] = fmt.Sprint("big", i)
	}
	out := st.smfsBatchGetLocked(smfsBatchGetReq{Identifiers: []smfsBatchIdentifier{{FeatureGroupName: "customers", RecordIdentifiersValueAsString: ids}}}, time.Now()).(map[string]any)
	if recs := out["Records"].([]any); len(recs) >= 30 || len(out["UnprocessedIdentifiers"].([]any)) != 1 {
		t.Fatalf("BatchGetRecord returned %d records, unprocessed %v", len(recs), len(out["UnprocessedIdentifiers"].([]any)))
	}
}
