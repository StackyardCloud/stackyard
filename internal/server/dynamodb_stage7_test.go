package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDynamoDBStage7GovernanceAndObservabilityCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage7-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage7-table"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeTableOut struct {
		Table struct {
			TableArn string `json:"TableArn"`
		} `json:"Table"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeTableOut); err != nil {
		t.Fatalf("unmarshal describe table response: %v", err)
	}
	tableArn := describeTableOut.Table.TableArn

	resp = dynamodbRequest(t, ts, "UpdateContinuousBackups", []byte(`{
		"TableName":"stage7-table",
		"PointInTimeRecoverySpecification":{"PointInTimeRecoveryEnabled":true}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeContinuousBackups", []byte(`{"TableName":"stage7-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "UpdateTimeToLive", []byte(`{
		"TableName":"stage7-table",
		"TimeToLiveSpecification":{"AttributeName":"ttl","Enabled":true}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTimeToLive", []byte(`{"TableName":"stage7-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "UpdateContributorInsights", []byte(`{
		"TableName":"stage7-table",
		"ContributorInsightsAction":"ENABLE"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeContributorInsights", []byte(`{"TableName":"stage7-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ListContributorInsights", []byte(`{"TableName":"stage7-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "TagResource", []byte(`{
		"ResourceArn":"`+tableArn+`",
		"Tags":[{"Key":"env","Value":"dev"},{"Key":"team","Value":"platform"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ListTagsOfResource", []byte(`{"ResourceArn":"`+tableArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "UntagResource", []byte(`{"ResourceArn":"`+tableArn+`","TagKeys":["team"]}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestDynamoDBStage7ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage7-implemented",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage7-implemented"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Table struct {
			TableArn string `json:"TableArn"`
		} `json:"Table"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe table response: %v", err)
	}

	actions := []struct {
		action string
		body   []byte
	}{
		{action: "UpdateContinuousBackups", body: []byte(`{"TableName":"stage7-implemented","PointInTimeRecoverySpecification":{"PointInTimeRecoveryEnabled":true}}`)},
		{action: "DescribeContinuousBackups", body: []byte(`{"TableName":"stage7-implemented"}`)},
		{action: "UpdateTimeToLive", body: []byte(`{"TableName":"stage7-implemented","TimeToLiveSpecification":{"AttributeName":"ttl","Enabled":true}}`)},
		{action: "DescribeTimeToLive", body: []byte(`{"TableName":"stage7-implemented"}`)},
		{action: "UpdateContributorInsights", body: []byte(`{"TableName":"stage7-implemented","ContributorInsightsAction":"ENABLE"}`)},
		{action: "DescribeContributorInsights", body: []byte(`{"TableName":"stage7-implemented"}`)},
		{action: "ListContributorInsights", body: []byte(`{"TableName":"stage7-implemented"}`)},
		{action: "TagResource", body: []byte(`{"ResourceArn":"` + describeOut.Table.TableArn + `","Tags":[{"Key":"env","Value":"test"}]}`)},
		{action: "UntagResource", body: []byte(`{"ResourceArn":"` + describeOut.Table.TableArn + `","TagKeys":["env"]}`)},
		{action: "ListTagsOfResource", body: []byte(`{"ResourceArn":"` + describeOut.Table.TableArn + `"}`)},
	}

	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}
