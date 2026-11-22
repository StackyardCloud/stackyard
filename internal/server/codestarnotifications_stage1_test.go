package server

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestCodeStarNotificationsStage1RuleLifecycle(t *testing.T) {
	srv, ts := newCodeStarNotificationsTestServer(t)
	topic, err := srv.sns.CreateTopic("csn-topic", nil)
	if err != nil {
		t.Fatal(err)
	}
	project := codeBuildProjectARN("stackyard-codebuild-project")
	create := func(name, token string) string {
		return fmt.Sprintf(`{"Name":%q,"EventTypeIds":["codebuild-project-build-state-failed"],"Resource":%q,"Targets":[{"TargetType":"SNS","TargetAddress":%q}],"DetailType":"FULL","ClientRequestToken":%q,"Tags":{"team":"ci"}}`, name, project, topic.ARN, token)
	}

	arn := csnCall(t, ts, http.StatusOK, "/createNotificationRule", create("builds", "tok-1"))["Arn"].(string)
	if again := csnCall(t, ts, http.StatusOK, "/createNotificationRule", create("builds", "tok-1"))["Arn"]; again != arn {
		t.Fatalf("same ClientRequestToken should return the same rule: %v vs %s", again, arn)
	}
	if code := csnErrorType(t, ts, "/createNotificationRule", create("builds", "tok-2")); code != "ResourceAlreadyExistsException" {
		t.Fatalf("duplicate name: %s", code)
	}

	desc := csnCall(t, ts, http.StatusOK, "/describeNotificationRule", fmt.Sprintf(`{"Arn":%q}`, arn))
	targets := desc["Targets"].([]any)
	if desc["Status"] != "ENABLED" || desc["DetailType"] != "FULL" || len(targets) != 1 || targets[0].(map[string]any)["TargetStatus"] != "ACTIVE" {
		t.Fatalf("describe: %v", desc)
	}
	if desc["EventTypes"].([]any)[0].(map[string]any)["ServiceName"] != "CodeBuild" {
		t.Fatalf("event types: %v", desc["EventTypes"])
	}

	// Update: status, detail type and event types (which must fit the resource).
	csnCall(t, ts, http.StatusOK, "/updateNotificationRule", fmt.Sprintf(`{"Arn":%q,"Status":"DISABLED","DetailType":"BASIC","EventTypeIds":["codebuild-project-build-state-succeeded"]}`, arn))
	if code := csnErrorType(t, ts, "/updateNotificationRule", fmt.Sprintf(`{"Arn":%q,"EventTypeIds":["codedeploy-application-deployment-failed"]}`, arn)); code != "ValidationException" {
		t.Fatalf("cross-service event type on update: %s", code)
	}
	desc = csnCall(t, ts, http.StatusOK, "/describeNotificationRule", fmt.Sprintf(`{"Arn":%q}`, arn))
	if desc["Status"] != "DISABLED" || desc["DetailType"] != "BASIC" {
		t.Fatalf("after update: %v", desc)
	}

	// Filters.
	rules := csnCall(t, ts, http.StatusOK, "/listNotificationRules", fmt.Sprintf(`{"Filters":[{"Name":"TARGET_ADDRESS","Value":%q}]}`, topic.ARN))["NotificationRules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("rules by target: %v", rules)
	}
	if rules := csnCall(t, ts, http.StatusOK, "/listNotificationRules", `{"Filters":[{"Name":"RESOURCE","Value":"arn:aws:codebuild:us-east-1:123456789012:project/other"}]}`)["NotificationRules"].([]any); len(rules) != 0 {
		t.Fatalf("rules by other resource: %v", rules)
	}

	// Tags.
	csnCall(t, ts, http.StatusOK, "/tagResource", fmt.Sprintf(`{"Arn":%q,"Tags":{"env":"dev"}}`, arn))
	csnCall(t, ts, http.StatusOK, "/untagResource/"+url.PathEscape(arn)+"?tagKeys=team", "")
	tags := csnCall(t, ts, http.StatusOK, "/listTagsForResource", fmt.Sprintf(`{"Arn":%q}`, arn))["Tags"].(map[string]any)
	if len(tags) != 1 || tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}

	// Delete, then describe is a 404; deleting again still succeeds.
	csnCall(t, ts, http.StatusOK, "/deleteNotificationRule", fmt.Sprintf(`{"Arn":%q}`, arn))
	if code := csnErrorType(t, ts, "/describeNotificationRule", fmt.Sprintf(`{"Arn":%q}`, arn)); code != "ResourceNotFoundException" {
		t.Fatalf("describe deleted: %s", code)
	}
	csnCall(t, ts, http.StatusOK, "/deleteNotificationRule", fmt.Sprintf(`{"Arn":%q}`, arn))
}

func TestCodeStarNotificationsStage1Validation(t *testing.T) {
	srv, ts := newCodeStarNotificationsTestServer(t)
	topic, _ := srv.sns.CreateTopic("csn-validate", nil)
	project := codeBuildProjectARN("stackyard-codebuild-project")
	rule := func(resource, event, targetType, address string) string {
		return fmt.Sprintf(`{"Name":"r","EventTypeIds":[%q],"Resource":%q,"Targets":[{"TargetType":%q,"TargetAddress":%q}],"DetailType":"BASIC"}`, event, resource, targetType, address)
	}
	for name, body := range map[string]string{
		"missing codebuild project":   rule(codeBuildProjectARN("nope"), "codebuild-project-build-state-failed", "SNS", topic.ARN),
		"unknown event type":          rule(project, "codebuild-project-build-exploded", "SNS", topic.ARN),
		"event type of other service": rule(project, "codepipeline-pipeline-manual-approval-needed", "SNS", topic.ARN),
		"missing SNS topic":           rule(project, "codebuild-project-build-state-failed", "SNS", "arn:aws:sns:us-east-1:123456789012:nope"),
		"bad chatbot address":         rule(project, "codebuild-project-build-state-failed", "AWSChatbotSlack", "arn:aws:chatbot::123456789012:chat-configuration/microsoft-teams-channel/x"),
		"unsupported resource":        rule("arn:aws:s3:::bucket/key:x:123456789012:y", "codebuild-project-build-state-failed", "SNS", topic.ARN),
		"bad detail type":             `{"Name":"r","EventTypeIds":["codebuild-project-build-state-failed"],"Resource":"` + project + `","Targets":[],"DetailType":"LOUD"}`,
		"bad name":                    `{"Name":"no!","EventTypeIds":["codebuild-project-build-state-failed"],"Resource":"` + project + `","Targets":[],"DetailType":"BASIC"}`,
	} {
		if code := csnErrorType(t, ts, "/createNotificationRule", body); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	// CodeCommit isn't emulated: repositories are accepted by ARN.
	csnCall(t, ts, http.StatusOK, "/createNotificationRule", rule("arn:aws:codecommit:us-east-1:123456789012:repo", "codecommit-repository-pull-request-created", "AWSChatbotSlack", "arn:aws:chatbot::123456789012:chat-configuration/slack-channel/dev"))
	if code := csnErrorType(t, ts, "/describeNotificationRule", `{"Arn":"arn:aws:codestar-notifications:us-east-1:123456789012:notificationrule/missing"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing rule: %s", code)
	}
	if code := csnErrorType(t, ts, "/listEventTypes", `{"NextToken":"999"}`); code != "InvalidNextTokenException" {
		t.Fatalf("bad token: %s", code)
	}
	if code := csnErrorType(t, ts, "/listEventTypes", `{"MaxResults":101}`); code != "ValidationException" {
		t.Fatalf("MaxResults over 100: %s", code)
	}
}

func TestCodeStarNotificationsStage1TargetsAndEventTypes(t *testing.T) {
	srv, ts := newCodeStarNotificationsTestServer(t)
	topic, _ := srv.sns.CreateTopic("csn-targets", nil)
	other, _ := srv.sns.CreateTopic("csn-other", nil)
	project := codeBuildProjectARN("stackyard-codebuild-project")
	arn := csnCall(t, ts, http.StatusOK, "/createNotificationRule", fmt.Sprintf(`{"Name":"t","EventTypeIds":["codebuild-project-build-state-failed"],"Resource":%q,"Targets":[{"TargetType":"SNS","TargetAddress":%q}],"DetailType":"BASIC"}`, project, topic.ARN))["Arn"].(string)

	csnCall(t, ts, http.StatusOK, "/subscribe", fmt.Sprintf(`{"Arn":%q,"Target":{"TargetType":"SNS","TargetAddress":%q}}`, arn, other.ARN))
	if n := len(csnCall(t, ts, http.StatusOK, "/describeNotificationRule", fmt.Sprintf(`{"Arn":%q}`, arn))["Targets"].([]any)); n != 2 {
		t.Fatalf("targets after subscribe: %d", n)
	}
	csnCall(t, ts, http.StatusOK, "/unsubscribe", fmt.Sprintf(`{"Arn":%q,"TargetAddress":%q}`, arn, other.ARN))

	// A deleted topic shows as UNREACHABLE.
	if err := srv.sns.DeleteTopic(other.ARN); err != nil {
		t.Fatal(err)
	}
	unreachable := csnCall(t, ts, http.StatusOK, "/listTargets", `{"Filters":[{"Name":"TARGET_STATUS","Value":"UNREACHABLE"}]}`)["Targets"].([]any)
	if len(unreachable) != 1 || unreachable[0].(map[string]any)["TargetAddress"] != other.ARN {
		t.Fatalf("unreachable targets: %v", unreachable)
	}

	// A target still in use needs ForceUnsubscribeAll.
	if code := csnErrorType(t, ts, "/deleteTarget", fmt.Sprintf(`{"TargetAddress":%q}`, topic.ARN)); code != "ValidationException" {
		t.Fatalf("delete in-use target: %s", code)
	}
	csnCall(t, ts, http.StatusOK, "/deleteTarget", fmt.Sprintf(`{"TargetAddress":%q,"ForceUnsubscribeAll":true}`, topic.ARN))
	if n := len(csnCall(t, ts, http.StatusOK, "/describeNotificationRule", fmt.Sprintf(`{"Arn":%q}`, arn))["Targets"].([]any)); n != 0 {
		t.Fatalf("targets after forced delete: %d", n)
	}

	// Event types filter and page.
	builds := csnCall(t, ts, http.StatusOK, "/listEventTypes", `{"Filters":[{"Name":"SERVICE_NAME","Value":"CodeBuild"}]}`)["EventTypes"].([]any)
	if len(builds) != 6 {
		t.Fatalf("CodeBuild event types: %d", len(builds))
	}
	page := csnCall(t, ts, http.StatusOK, "/listEventTypes", `{"MaxResults":5}`)
	if len(page["EventTypes"].([]any)) != 5 || page["NextToken"] == nil {
		t.Fatalf("first page: %v", page)
	}
	rest := csnCall(t, ts, http.StatusOK, "/listEventTypes", fmt.Sprintf(`{"NextToken":%q}`, page["NextToken"]))
	if got := len(rest["EventTypes"].([]any)) + 5; got != len(csnEventTypes) || rest["NextToken"] != nil {
		t.Fatalf("pages cover %d of %d event types", got, len(csnEventTypes))
	}
}
