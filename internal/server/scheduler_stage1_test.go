package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const schedulerTestRole = "arn:aws:iam::123456789012:role/scheduler"

func schedulerBody(expr, target string) string {
	return fmt.Sprintf(`{"ScheduleExpression":%q,"FlexibleTimeWindow":{"Mode":"OFF"},"Target":%s}`, expr, target)
}

func schedulerSQSTarget(queue string) string {
	return fmt.Sprintf(`{"Arn":"arn:aws:sqs:us-east-1:123456789012:%s","RoleArn":%q,"Input":"{\"hello\":1}"}`, queue, schedulerTestRole)
}

func TestSchedulerStage1ScheduleLifecycle(t *testing.T) {
	_, ts := newSchedulerTestServer(t)
	body := schedulerBody("rate(5 minutes)", schedulerSQSTarget("q1"))
	out := schedulerCall(t, ts, http.MethodPost, "/schedules/s1", body, http.StatusOK)
	if out["ScheduleArn"] != "arn:aws:scheduler:us-east-1:123456789012:schedule/default/s1" {
		t.Fatalf("create: %v", out)
	}
	if code := schedulerCallError(t, ts, http.MethodPost, "/schedules/s1", body, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("duplicate: %s", code)
	}

	got := schedulerCall(t, ts, http.MethodGet, "/schedules/s1", "", http.StatusOK)
	target := got["Target"].(map[string]any)
	retry := target["RetryPolicy"].(map[string]any)
	if got["State"] != "ENABLED" || got["GroupName"] != "default" || got["ActionAfterCompletion"] != "NONE" ||
		retry["MaximumEventAgeInSeconds"] != float64(86400) || retry["MaximumRetryAttempts"] != float64(185) {
		t.Fatalf("get: %v", got)
	}
	if _, ok := got["CreationDate"].(float64); !ok {
		t.Fatalf("CreationDate should be epoch seconds: %v", got["CreationDate"])
	}

	upd := schedulerBody("cron(0 12 ? * MON-FRI *)", schedulerSQSTarget("q1"))
	upd = strings.Replace(upd, `{"ScheduleExpression"`, `{"State":"DISABLED","ScheduleExpressionTimezone":"America/New_York","ScheduleExpression"`, 1)
	schedulerCall(t, ts, http.MethodPut, "/schedules/s1", upd, http.StatusOK)
	got = schedulerCall(t, ts, http.MethodGet, "/schedules/s1", "", http.StatusOK)
	if got["State"] != "DISABLED" || got["ScheduleExpressionTimezone"] != "America/New_York" {
		t.Fatalf("after update: %v", got)
	}
	schedulerCallError(t, ts, http.MethodPut, "/schedules/missing", upd, http.StatusNotFound)

	schedulerCall(t, ts, http.MethodPost, "/schedules/s2", body, http.StatusOK)
	list := schedulerCall(t, ts, http.MethodGet, "/schedules?MaxResults=1", "", http.StatusOK)
	page := list["Schedules"].([]any)
	if len(page) != 1 || list["NextToken"] == nil {
		t.Fatalf("page 1: %v", list)
	}
	list = schedulerCall(t, ts, http.MethodGet, "/schedules?MaxResults=1&NextToken="+list["NextToken"].(string), "", http.StatusOK)
	if s := list["Schedules"].([]any); len(s) != 1 || s[0].(map[string]any)["Name"] != "s2" || list["NextToken"] != nil {
		t.Fatalf("page 2: %v", list)
	}
	if s := schedulerCall(t, ts, http.MethodGet, "/schedules?State=DISABLED", "", http.StatusOK)["Schedules"].([]any); len(s) != 1 {
		t.Fatalf("state filter: %v", s)
	}
	schedulerCallError(t, ts, http.MethodGet, "/schedules?MaxResults=101", "", http.StatusBadRequest)
	schedulerCallError(t, ts, http.MethodGet, "/schedules?NextToken=bogus", "", http.StatusBadRequest)
	schedulerCallError(t, ts, http.MethodGet, "/schedules?ScheduleGroup=nope", "", http.StatusNotFound)

	schedulerCall(t, ts, http.MethodDelete, "/schedules/s1", "", http.StatusOK)
	if code := schedulerCallError(t, ts, http.MethodGet, "/schedules/s1", "", http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("after delete: %s", code)
	}
}

func TestSchedulerStage1ClientTokenIdempotency(t *testing.T) {
	_, ts := newSchedulerTestServer(t)
	body := strings.Replace(schedulerBody("rate(1 hour)", schedulerSQSTarget("q1")), `{"ScheduleExpression"`, `{"ClientToken":"tok-1","ScheduleExpression"`, 1)
	schedulerCall(t, ts, http.MethodPost, "/schedules/idem", body, http.StatusOK)
	schedulerCall(t, ts, http.MethodPost, "/schedules/idem", body, http.StatusOK)
	other := strings.Replace(body, "tok-1", "tok-2", 1)
	schedulerCallError(t, ts, http.MethodPost, "/schedules/idem", other, http.StatusConflict)
}

func TestSchedulerStage1Validation(t *testing.T) {
	_, ts := newSchedulerTestServer(t)
	sqs := schedulerSQSTarget("q1")
	for name, body := range map[string]string{
		"bad rate":          schedulerBody("rate(0 minutes)", sqs),
		"rate unit":         schedulerBody("rate(5 weeks)", sqs),
		"bad at":            schedulerBody("at(2030-13-01T00:00:00)", sqs),
		"cron five fields":  schedulerBody("cron(0 12 * * ?)", sqs),
		"cron both ?":       schedulerBody("cron(0 12 ? * ? *)", sqs),
		"cron neither ?":    schedulerBody("cron(0 12 * * MON *)", sqs),
		"cron minute range": schedulerBody("cron(60 12 ? * MON *)", sqs),
		"cron L in hours":   schedulerBody("cron(0 L ? * MON *)", sqs),
		"unknown expr":      schedulerBody("every 5 minutes", sqs),
		"bad timezone":      strings.Replace(schedulerBody("rate(5 minutes)", sqs), `{"ScheduleExpression"`, `{"ScheduleExpressionTimezone":"Mars/Base","ScheduleExpression"`, 1),
		"flexible no max":   strings.Replace(schedulerBody("rate(5 minutes)", sqs), `"Mode":"OFF"`, `"Mode":"FLEXIBLE"`, 1),
		"off with max":      strings.Replace(schedulerBody("rate(5 minutes)", sqs), `"Mode":"OFF"`, `"Mode":"OFF","MaximumWindowInMinutes":5`, 1),
		"bad role":          schedulerBody("rate(5 minutes)", `{"Arn":"arn:aws:sqs:us-east-1:123456789012:q1","RoleArn":"role"}`),
		"unknown service":   schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:s3:::bucket","RoleArn":%q}`, schedulerTestRole)),
		"universal no json": schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:scheduler:::aws-sdk:sqs:sendMessage","RoleArn":%q,"Input":"nope"}`, schedulerTestRole)),
		"kinesis no key":    schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:kinesis:us-east-1:123456789012:stream/s","RoleArn":%q}`, schedulerTestRole)),
		"ecs params on sqs": schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:sqs:us-east-1:123456789012:q1","RoleArn":%q,"EcsParameters":{"TaskDefinitionArn":"td"}}`, schedulerTestRole)),
		"ecs task count":    schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:ecs:us-east-1:123456789012:cluster/c","RoleArn":%q,"EcsParameters":{"TaskDefinitionArn":"td","TaskCount":11}}`, schedulerTestRole)),
		"events no params":  schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:events:us-east-1:123456789012:event-bus/default","RoleArn":%q}`, schedulerTestRole)),
		"events aws source": schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:events:us-east-1:123456789012:event-bus/default","RoleArn":%q,"EventBridgeParameters":{"DetailType":"d","Source":"aws.ec2"}}`, schedulerTestRole)),
		"fifo no group":     schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:sqs:us-east-1:123456789012:q.fifo","RoleArn":%q}`, schedulerTestRole)),
		"retry range":       schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:sqs:us-east-1:123456789012:q1","RoleArn":%q,"RetryPolicy":{"MaximumRetryAttempts":186}}`, schedulerTestRole)),
		"dlq not sqs":       schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:sqs:us-east-1:123456789012:q1","RoleArn":%q,"DeadLetterConfig":{"Arn":"arn:aws:sns:us-east-1:123456789012:t"}}`, schedulerTestRole)),
		"unknown member":    strings.Replace(schedulerBody("rate(5 minutes)", sqs), `{"ScheduleExpression"`, `{"CreatedBy":"me","ScheduleExpression"`, 1),
		"past start date":   strings.Replace(schedulerBody("rate(5 minutes)", sqs), `{"ScheduleExpression"`, `{"StartDate":1000000000,"ScheduleExpression"`, 1),
		"bad state":         strings.Replace(schedulerBody("rate(5 minutes)", sqs), `{"ScheduleExpression"`, `{"State":"PAUSED","ScheduleExpression"`, 1),
		"missing window":    fmt.Sprintf(`{"ScheduleExpression":"rate(5 minutes)","Target":%s}`, sqs),
	} {
		if code := schedulerCallError(t, ts, http.MethodPost, "/schedules/v1", body, http.StatusBadRequest); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	schedulerCallError(t, ts, http.MethodPost, "/schedules/bad%20name", schedulerBody("rate(5 minutes)", sqs), http.StatusBadRequest)

	for name, body := range map[string]string{
		"cron L and W":     schedulerBody("cron(0 9 LW * ? *)", sqs),
		"cron nth weekday": schedulerBody("cron(15 10 ? JAN-JUL 2#3 2030-2035)", sqs),
		"at":               schedulerBody("at(2030-01-01T00:00:00)", sqs),
		"universal":        schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:scheduler:::aws-sdk:sqs:sendMessage","RoleArn":%q,"Input":"{}"}`, schedulerTestRole)),
		"ecs":              schedulerBody("rate(5 minutes)", fmt.Sprintf(`{"Arn":"arn:aws:ecs:us-east-1:123456789012:cluster/c","RoleArn":%q,"EcsParameters":{"TaskDefinitionArn":"td","TaskCount":2,"LaunchType":"FARGATE"}}`, schedulerTestRole)),
		"flexible":         strings.Replace(schedulerBody("rate(5 minutes)", sqs), `"Mode":"OFF"`, `"Mode":"FLEXIBLE","MaximumWindowInMinutes":15`, 1),
	} {
		schedulerCall(t, ts, http.MethodPost, "/schedules/"+strings.ReplaceAll(name, " ", "-"), body, http.StatusOK)
	}
}

func TestSchedulerStage1Groups(t *testing.T) {
	_, ts := newSchedulerTestServer(t)
	out := schedulerCall(t, ts, http.MethodPost, "/schedule-groups/g1", `{"Tags":[{"Key":"team","Value":"a"}]}`, http.StatusOK)
	arn := "arn:aws:scheduler:us-east-1:123456789012:schedule-group/g1"
	if out["ScheduleGroupArn"] != arn {
		t.Fatalf("create group: %v", out)
	}
	schedulerCallError(t, ts, http.MethodPost, "/schedule-groups/g1", `{}`, http.StatusConflict)
	if g := schedulerCall(t, ts, http.MethodGet, "/schedule-groups/default", "", http.StatusOK); g["State"] != "ACTIVE" {
		t.Fatalf("default group: %v", g)
	}
	if code := schedulerCallError(t, ts, http.MethodDelete, "/schedule-groups/default", "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("delete default: %s", code)
	}
	if code := schedulerCallError(t, ts, http.MethodPost, "/schedule-groups/g1", `{"Tags":[{"Key":"aws:x","Value":"v"}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("aws: tag: %s", code)
	}

	// Tags only apply to schedule groups.
	escaped := strings.ReplaceAll(arn, "/", "%2F")
	schedulerCall(t, ts, http.MethodPost, "/tags/"+escaped, `{"Tags":[{"Key":"env","Value":"dev"}]}`, http.StatusOK)
	tags := schedulerCall(t, ts, http.MethodGet, "/tags/"+escaped, "", http.StatusOK)["Tags"].([]any)
	if len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	schedulerCall(t, ts, http.MethodDelete, "/tags/"+escaped+"?TagKeys=team", "", http.StatusOK)
	if tags = schedulerCall(t, ts, http.MethodGet, "/tags/"+escaped, "", http.StatusOK)["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("after untag: %v", tags)
	}
	schedulerCallError(t, ts, http.MethodGet, "/tags/"+strings.ReplaceAll("arn:aws:scheduler:us-east-1:123456789012:schedule/default/x", "/", "%2F"), "", http.StatusBadRequest)
	schedulerCallError(t, ts, http.MethodGet, "/tags/"+strings.ReplaceAll(strings.Replace(arn, "g1", "nope", 1), "/", "%2F"), "", http.StatusNotFound)

	// Deleting a group deletes its schedules; the group is reported as
	// DELETING once, then gone.
	sched := strings.Replace(schedulerBody("rate(5 minutes)", schedulerSQSTarget("q1")), `{"ScheduleExpression"`, `{"GroupName":"g1","ScheduleExpression"`, 1)
	schedulerCall(t, ts, http.MethodPost, "/schedules/in-g1", sched, http.StatusOK)
	schedulerCall(t, ts, http.MethodDelete, "/schedule-groups/g1", "", http.StatusOK)
	if code := schedulerCallError(t, ts, http.MethodPost, "/schedules/in-g1-2", sched, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("schedule in deleting group: %s", code)
	}
	if g := schedulerCall(t, ts, http.MethodGet, "/schedule-groups/g1", "", http.StatusOK); g["State"] != "DELETING" {
		t.Fatalf("deleting group: %v", g)
	}
	schedulerCallError(t, ts, http.MethodGet, "/schedule-groups/g1", "", http.StatusNotFound)
	schedulerCallError(t, ts, http.MethodGet, "/schedules/in-g1?groupName=g1", "", http.StatusNotFound)
	schedulerCallError(t, ts, http.MethodPost, "/schedules/x", strings.Replace(sched, `"g1"`, `"nope"`, 1), http.StatusNotFound)
	if groups := schedulerCall(t, ts, http.MethodGet, "/schedule-groups?NamePrefix=def", "", http.StatusOK)["ScheduleGroups"].([]any); len(groups) != 1 {
		t.Fatalf("list groups: %v", groups)
	}
}
