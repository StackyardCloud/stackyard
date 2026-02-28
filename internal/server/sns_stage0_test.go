package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func snsRequest(t *testing.T, ts *httptest.Server, values url.Values) *http.Response {
	t.Helper()
	body := []byte(values.Encode())
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "sns")
}

func snsAttributesToMap(entries []snsAttributeEntry) map[string]string {
	out := map[string]string{}
	for _, entry := range entries {
		out[entry.Key] = entry.Value
	}
	return out
}

func TestSNSStage0TopicLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{}
	create.Set("Action", "CreateTopic")
	create.Set("Name", "demo-topic")
	create.Set("Version", "2010-03-31")
	resp := snsRequest(t, ts, create)
	assertStatus(t, resp, http.StatusOK)
	var createResp struct {
		Result snsCreateTopicResult `xml:"CreateTopicResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("unmarshal create topic: %v", err)
	}
	if createResp.Result.TopicArn == "" {
		t.Fatalf("expected topic arn")
	}
	topicArn := createResp.Result.TopicArn

	setAttrs := url.Values{}
	setAttrs.Set("Action", "SetTopicAttributes")
	setAttrs.Set("TopicArn", topicArn)
	setAttrs.Set("AttributeName", "DisplayName")
	setAttrs.Set("AttributeValue", "Demo Topic")
	resp = snsRequest(t, ts, setAttrs)
	assertStatus(t, resp, http.StatusOK)

	getAttrs := url.Values{}
	getAttrs.Set("Action", "GetTopicAttributes")
	getAttrs.Set("TopicArn", topicArn)
	resp = snsRequest(t, ts, getAttrs)
	assertStatus(t, resp, http.StatusOK)
	var getResp struct {
		Result snsGetTopicAttributesResult `xml:"GetTopicAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("unmarshal get topic attrs: %v", err)
	}
	attrs := snsAttributesToMap(getResp.Result.Attributes)
	if attrs["DisplayName"] != "Demo Topic" {
		t.Fatalf("expected DisplayName=Demo Topic, got %q", attrs["DisplayName"])
	}

	listTopics := url.Values{}
	listTopics.Set("Action", "ListTopics")
	resp = snsRequest(t, ts, listTopics)
	assertStatus(t, resp, http.StatusOK)
	var listResp struct {
		Result snsListTopicsResult `xml:"ListTopicsResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("unmarshal list topics: %v", err)
	}
	found := false
	for _, entry := range listResp.Result.Topics {
		if entry.TopicArn == topicArn {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected topic in list")
	}

	publish := url.Values{}
	publish.Set("Action", "Publish")
	publish.Set("TopicArn", topicArn)
	publish.Set("Message", "hello")
	resp = snsRequest(t, ts, publish)
	assertStatus(t, resp, http.StatusOK)
	var publishResp struct {
		Result snsPublishResult `xml:"PublishResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &publishResp); err != nil {
		t.Fatalf("unmarshal publish: %v", err)
	}
	if publishResp.Result.MessageID == "" {
		t.Fatalf("expected message id")
	}

	deleteTopic := url.Values{}
	deleteTopic.Set("Action", "DeleteTopic")
	deleteTopic.Set("TopicArn", topicArn)
	resp = snsRequest(t, ts, deleteTopic)
	assertStatus(t, resp, http.StatusOK)
}

func TestSNSStage0SubscriptionLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{}
	create.Set("Action", "CreateTopic")
	create.Set("Name", "demo-sub")
	resp := snsRequest(t, ts, create)
	assertStatus(t, resp, http.StatusOK)
	var createResp struct {
		Result snsCreateTopicResult `xml:"CreateTopicResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("unmarshal create topic: %v", err)
	}
	arn := createResp.Result.TopicArn

	subscribe := url.Values{}
	subscribe.Set("Action", "Subscribe")
	subscribe.Set("TopicArn", arn)
	subscribe.Set("Protocol", "sqs")
	subscribe.Set("Endpoint", "arn:aws:sqs:us-east-1:123456789012:demo")
	resp = snsRequest(t, ts, subscribe)
	assertStatus(t, resp, http.StatusOK)
	var subResp struct {
		Result snsSubscribeResult `xml:"SubscribeResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &subResp); err != nil {
		t.Fatalf("unmarshal subscribe: %v", err)
	}
	if subResp.Result.SubscriptionArn == "" || subResp.Result.SubscriptionArn == "pending confirmation" {
		t.Fatalf("expected subscription arn, got %q", subResp.Result.SubscriptionArn)
	}
	subArn := subResp.Result.SubscriptionArn

	getAttrs := url.Values{}
	getAttrs.Set("Action", "GetSubscriptionAttributes")
	getAttrs.Set("SubscriptionArn", subArn)
	resp = snsRequest(t, ts, getAttrs)
	assertStatus(t, resp, http.StatusOK)
	var getSubResp struct {
		Result snsGetSubscriptionAttributesResult `xml:"GetSubscriptionAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &getSubResp); err != nil {
		t.Fatalf("unmarshal get subscription: %v", err)
	}
	attrs := snsAttributesToMap(getSubResp.Result.Attributes)
	if attrs["PendingConfirmation"] != "false" {
		t.Fatalf("expected PendingConfirmation=false, got %q", attrs["PendingConfirmation"])
	}

	setAttrs := url.Values{}
	setAttrs.Set("Action", "SetSubscriptionAttributes")
	setAttrs.Set("SubscriptionArn", subArn)
	setAttrs.Set("AttributeName", "FilterPolicy")
	setAttrs.Set("AttributeValue", "{}")
	resp = snsRequest(t, ts, setAttrs)
	assertStatus(t, resp, http.StatusOK)

	list := url.Values{}
	list.Set("Action", "ListSubscriptionsByTopic")
	list.Set("TopicArn", arn)
	resp = snsRequest(t, ts, list)
	assertStatus(t, resp, http.StatusOK)
	var listResp struct {
		Result snsListSubscriptionsByTopicResult `xml:"ListSubscriptionsByTopicResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("unmarshal list subs: %v", err)
	}
	found := false
	for _, entry := range listResp.Result.Subscriptions {
		if entry.SubscriptionArn == subArn {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected subscription in list")
	}

	unsubscribe := url.Values{}
	unsubscribe.Set("Action", "Unsubscribe")
	unsubscribe.Set("SubscriptionArn", subArn)
	resp = snsRequest(t, ts, unsubscribe)
	assertStatus(t, resp, http.StatusOK)
}

func TestSNSStage0TagsPlatformAndEndpoint(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{}
	create.Set("Action", "CreateTopic")
	create.Set("Name", "demo-tags")
	resp := snsRequest(t, ts, create)
	assertStatus(t, resp, http.StatusOK)
	var createResp struct {
		Result snsCreateTopicResult `xml:"CreateTopicResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("unmarshal create topic: %v", err)
	}
	topicArn := createResp.Result.TopicArn

	tag := url.Values{}
	tag.Set("Action", "TagResource")
	tag.Set("ResourceArn", topicArn)
	tag.Set("Tags.member.1.Key", "env")
	tag.Set("Tags.member.1.Value", "test")
	resp = snsRequest(t, ts, tag)
	assertStatus(t, resp, http.StatusOK)

	listTags := url.Values{}
	listTags.Set("Action", "ListTagsForResource")
	listTags.Set("ResourceArn", topicArn)
	resp = snsRequest(t, ts, listTags)
	assertStatus(t, resp, http.StatusOK)
	var listTagsResp struct {
		Result snsListTagsForResourceResult `xml:"ListTagsForResourceResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listTagsResp); err != nil {
		t.Fatalf("unmarshal list tags: %v", err)
	}
	if len(listTagsResp.Result.Tags) != 1 || listTagsResp.Result.Tags[0].Key != "env" {
		t.Fatalf("expected tag env")
	}

	untag := url.Values{}
	untag.Set("Action", "UntagResource")
	untag.Set("ResourceArn", topicArn)
	untag.Set("TagKeys.member.1", "env")
	resp = snsRequest(t, ts, untag)
	assertStatus(t, resp, http.StatusOK)

	createApp := url.Values{}
	createApp.Set("Action", "CreatePlatformApplication")
	createApp.Set("Name", "demo-app")
	createApp.Set("Platform", "APNS")
	createApp.Set("Attributes.entry.1.key", "PlatformCredential")
	createApp.Set("Attributes.entry.1.value", "token")
	resp = snsRequest(t, ts, createApp)
	assertStatus(t, resp, http.StatusOK)
	var appResp struct {
		Result snsCreatePlatformApplicationResult `xml:"CreatePlatformApplicationResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &appResp); err != nil {
		t.Fatalf("unmarshal create platform app: %v", err)
	}
	appArn := appResp.Result.PlatformApplicationArn

	createEndpoint := url.Values{}
	createEndpoint.Set("Action", "CreatePlatformEndpoint")
	createEndpoint.Set("PlatformApplicationArn", appArn)
	createEndpoint.Set("Token", "device-token")
	createEndpoint.Set("Attributes.entry.1.key", "Enabled")
	createEndpoint.Set("Attributes.entry.1.value", "true")
	resp = snsRequest(t, ts, createEndpoint)
	assertStatus(t, resp, http.StatusOK)
	var endpointResp struct {
		Result snsCreatePlatformEndpointResult `xml:"CreatePlatformEndpointResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &endpointResp); err != nil {
		t.Fatalf("unmarshal create endpoint: %v", err)
	}
	endpointArn := endpointResp.Result.EndpointArn

	listEndpoints := url.Values{}
	listEndpoints.Set("Action", "ListEndpointsByPlatformApplication")
	listEndpoints.Set("PlatformApplicationArn", appArn)
	resp = snsRequest(t, ts, listEndpoints)
	assertStatus(t, resp, http.StatusOK)
	var listEndpointsResp struct {
		Result snsListEndpointsByPlatformApplicationResult `xml:"ListEndpointsByPlatformApplicationResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listEndpointsResp); err != nil {
		t.Fatalf("unmarshal list endpoints: %v", err)
	}
	if len(listEndpointsResp.Result.Endpoints) != 1 || listEndpointsResp.Result.Endpoints[0].EndpointArn != endpointArn {
		t.Fatalf("expected endpoint in list")
	}

	getEndpoint := url.Values{}
	getEndpoint.Set("Action", "GetEndpointAttributes")
	getEndpoint.Set("EndpointArn", endpointArn)
	resp = snsRequest(t, ts, getEndpoint)
	assertStatus(t, resp, http.StatusOK)
	var getEndpointResp struct {
		Result snsGetEndpointAttributesResult `xml:"GetEndpointAttributesResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &getEndpointResp); err != nil {
		t.Fatalf("unmarshal get endpoint: %v", err)
	}
	endpointAttrs := snsAttributesToMap(getEndpointResp.Result.Attributes)
	if endpointAttrs["Token"] != "device-token" {
		t.Fatalf("expected endpoint token")
	}

	deleteEndpoint := url.Values{}
	deleteEndpoint.Set("Action", "DeleteEndpoint")
	deleteEndpoint.Set("EndpointArn", endpointArn)
	resp = snsRequest(t, ts, deleteEndpoint)
	assertStatus(t, resp, http.StatusOK)

	deleteApp := url.Values{}
	deleteApp.Set("Action", "DeletePlatformApplication")
	deleteApp.Set("PlatformApplicationArn", appArn)
	resp = snsRequest(t, ts, deleteApp)
	assertStatus(t, resp, http.StatusOK)
}

func TestSNSStage0SMSSandboxLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{}
	create.Set("Action", "CreateSMSSandboxPhoneNumber")
	create.Set("PhoneNumber", "+15550001111")
	create.Set("LanguageCode", "en-US")
	resp := snsRequest(t, ts, create)
	assertStatus(t, resp, http.StatusOK)

	list := url.Values{}
	list.Set("Action", "ListSMSSandboxPhoneNumbers")
	resp = snsRequest(t, ts, list)
	assertStatus(t, resp, http.StatusOK)
	var listResp struct {
		Result snsListSMSSandboxPhoneNumbersResult `xml:"ListSMSSandboxPhoneNumbersResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &listResp); err != nil {
		t.Fatalf("unmarshal list sandbox: %v", err)
	}
	if len(listResp.Result.PhoneNumbers) != 1 {
		t.Fatalf("expected one sandbox phone number")
	}

	verify := url.Values{}
	verify.Set("Action", "VerifySMSSandboxPhoneNumber")
	verify.Set("PhoneNumber", "+15550001111")
	verify.Set("OneTimePassword", "000000")
	resp = snsRequest(t, ts, verify)
	assertStatus(t, resp, http.StatusOK)

	status := url.Values{}
	status.Set("Action", "GetSMSSandboxAccountStatus")
	resp = snsRequest(t, ts, status)
	assertStatus(t, resp, http.StatusOK)
	var statusResp struct {
		Result snsGetSMSSandboxAccountStatusResult `xml:"GetSMSSandboxAccountStatusResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &statusResp); err != nil {
		t.Fatalf("unmarshal sandbox status: %v", err)
	}
	if !statusResp.Result.IsInSandbox {
		t.Fatalf("expected sandbox enabled")
	}

	deleteReq := url.Values{}
	deleteReq.Set("Action", "DeleteSMSSandboxPhoneNumber")
	deleteReq.Set("PhoneNumber", "+15550001111")
	resp = snsRequest(t, ts, deleteReq)
	assertStatus(t, resp, http.StatusOK)
}
