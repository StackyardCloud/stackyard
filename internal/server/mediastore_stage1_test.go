package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMediaStoreStage1Containers(t *testing.T) {
	ts := newMediaStoreTestServer(t)
	c := mstOK(t, ts, "CreateContainer", `{"ContainerName":"live","Tags":[{"Key":"env","Value":"dev"}]}`)["Container"].(map[string]any)
	if c["Status"] != "CREATING" || c["Endpoint"] != nil || c["ARN"] != "arn:aws:mediastore:us-east-1:123456789012:container/live" {
		t.Fatalf("create: %v", c)
	}
	mstFail(t, ts, "CreateContainer", `{"ContainerName":"live"}`, "ContainerInUseException")
	d := mstOK(t, ts, "DescribeContainer", `{"ContainerName":"live"}`)["Container"].(map[string]any)
	if d["Status"] != "ACTIVE" || !strings.HasPrefix(d["Endpoint"].(string), ts.URL+mstdPathPrefix) {
		t.Fatalf("describe: %v", d)
	}
	mstFail(t, ts, "DescribeContainer", `{"ContainerName":"nope"}`, "ContainerNotFoundException")

	mstOK(t, ts, "CreateContainer", `{"ContainerName":"vod"}`)
	page := mstOK(t, ts, "ListContainers", `{"MaxResults":1}`)
	if len(page["Containers"].([]any)) != 1 || page["NextToken"] == nil {
		t.Fatalf("page 1: %v", page)
	}
	page = mstOK(t, ts, "ListContainers", `{"MaxResults":1,"NextToken":"`+page["NextToken"].(string)+`"}`)
	if got := page["Containers"].([]any)[0].(map[string]any)["Name"]; got != "vod" || page["NextToken"] != nil {
		t.Fatalf("page 2: %v", page)
	}

	// Policies round-trip and report their own not-found errors.
	mstFail(t, ts, "GetContainerPolicy", `{"ContainerName":"live"}`, "PolicyNotFoundException")
	mstFail(t, ts, "PutContainerPolicy", `{"ContainerName":"live","Policy":"not json"}`, "ValidationException")
	pol := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"mediastore:GetObject","Resource":"*"}]}`
	polJSON, _ := json.Marshal(pol)
	mstOK(t, ts, "PutContainerPolicy", `{"ContainerName":"live","Policy":`+string(polJSON)+`}`)
	if got := mstOK(t, ts, "GetContainerPolicy", `{"ContainerName":"live"}`)["Policy"]; got != pol {
		t.Fatalf("policy: %v", got)
	}
	mstOK(t, ts, "DeleteContainerPolicy", `{"ContainerName":"live"}`)
	mstFail(t, ts, "DeleteContainerPolicy", `{"ContainerName":"live"}`, "PolicyNotFoundException")

	mstFail(t, ts, "GetCorsPolicy", `{"ContainerName":"live"}`, "CorsPolicyNotFoundException")
	mstFail(t, ts, "PutCorsPolicy", `{"ContainerName":"live","CorsPolicy":[{"AllowedOrigins":["*"]}]}`, "ValidationException")
	mstOK(t, ts, "PutCorsPolicy", `{"ContainerName":"live","CorsPolicy":[{"AllowedOrigins":["*"],"AllowedHeaders":["*"],"AllowedMethods":["GET"],"MaxAgeSeconds":3000}]}`)
	cors := mstOK(t, ts, "GetCorsPolicy", `{"ContainerName":"live"}`)["CorsPolicy"].([]any)
	if cors[0].(map[string]any)["MaxAgeSeconds"] != float64(3000) {
		t.Fatalf("cors: %v", cors)
	}
	mstOK(t, ts, "DeleteCorsPolicy", `{"ContainerName":"live"}`)

	mstFail(t, ts, "PutLifecyclePolicy", `{"ContainerName":"live","LifecyclePolicy":"{\"rules\":[]}"}`, "ValidationException")
	lc := `{"rules":[{"definition":{"path":[{"prefix":"tmp/"}],"days_since_create":[{"numeric":[">",7]}]},"action":"EXPIRE"}]}`
	lcJSON, _ := json.Marshal(lc)
	mstOK(t, ts, "PutLifecyclePolicy", `{"ContainerName":"live","LifecyclePolicy":`+string(lcJSON)+`}`)
	if got := mstOK(t, ts, "GetLifecyclePolicy", `{"ContainerName":"live"}`)["LifecyclePolicy"]; got != lc {
		t.Fatalf("lifecycle: %v", got)
	}
	mstOK(t, ts, "DeleteLifecyclePolicy", `{"ContainerName":"live"}`)

	mstFail(t, ts, "PutMetricPolicy", `{"ContainerName":"live","MetricPolicy":{"ContainerLevelMetrics":"ENABLED","MetricPolicyRules":[{"ObjectGroup":"a/","ObjectGroupName":"g"},{"ObjectGroup":"b/","ObjectGroupName":"g"}]}}`, "ValidationException")
	mstOK(t, ts, "PutMetricPolicy", `{"ContainerName":"live","MetricPolicy":{"ContainerLevelMetrics":"ENABLED"}}`)
	if mp := mstOK(t, ts, "GetMetricPolicy", `{"ContainerName":"live"}`)["MetricPolicy"].(map[string]any); mp["ContainerLevelMetrics"] != "ENABLED" {
		t.Fatalf("metric: %v", mp)
	}
	mstOK(t, ts, "DeleteMetricPolicy", `{"ContainerName":"live"}`)
	mstFail(t, ts, "GetMetricPolicy", `{"ContainerName":"live"}`, "PolicyNotFoundException")

	mstOK(t, ts, "StartAccessLogging", `{"ContainerName":"live"}`)
	if d := mstOK(t, ts, "DescribeContainer", `{"ContainerName":"live"}`)["Container"].(map[string]any); d["AccessLoggingEnabled"] != true {
		t.Fatalf("logging: %v", d)
	}
	mstOK(t, ts, "StopAccessLogging", `{"ContainerName":"live"}`)

	arn := c["ARN"].(string)
	mstOK(t, ts, "TagResource", `{"Resource":"`+arn+`","Tags":[{"Key":"env","Value":"prod"},{"Key":"team","Value":"video"}]}`)
	mstOK(t, ts, "UntagResource", `{"Resource":"`+arn+`","TagKeys":["team"]}`)
	tags := mstOK(t, ts, "ListTagsForResource", `{"Resource":"`+arn+`"}`)["Tags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["Value"] != "prod" {
		t.Fatalf("tags: %v", tags)
	}
	mstFail(t, ts, "TagResource", `{"Resource":"`+arn+`","Tags":[{"Key":"aws:x","Value":"y"}]}`, "ValidationException")

	mstOK(t, ts, "DeleteContainer", `{"ContainerName":"vod"}`)
	mstFail(t, ts, "DeleteContainer", `{"ContainerName":"vod"}`, "ContainerNotFoundException")
}
