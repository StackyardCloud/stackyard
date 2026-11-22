package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/services/ec2"
)

const iscCVEPackage = "arn:aws:inspector:us-east-1:316112463485:rulespackage/0-gEjTy7T7"

func iscRunInstance(t *testing.T, srv *Server, tags ...ec2.Tag) string {
	t.Helper()
	res, err := srv.ec2.RunInstances("ami-12345678", "t3.micro", "", "", "", nil, 1, 1, tags)
	if err != nil || len(res.Instances) != 1 {
		t.Fatalf("run instance: %v", err)
	}
	return res.Instances[0].ID
}

func TestInspectorStage1AssessmentLifecycle(t *testing.T) {
	srv, ts := newInspectorTestServer(t)
	web := iscRunInstance(t, srv, ec2.Tag{Key: "role", Value: "web"})
	stopped := iscRunInstance(t, srv, ec2.Tag{Key: "role", Value: "web"})
	iscRunInstance(t, srv, ec2.Tag{Key: "role", Value: "db"})
	if _, err := srv.ec2.StopInstances([]string{stopped}); err != nil {
		t.Fatal(err)
	}

	group := iscOK(t, ts, "CreateResourceGroup", `{"resourceGroupTags":[{"key":"role","value":"web"}]}`)["resourceGroupArn"].(string)
	target := iscOK(t, ts, "CreateAssessmentTarget", fmt.Sprintf(`{"assessmentTargetName":"web-target","resourceGroupArn":%q}`, group))["assessmentTargetArn"].(string)
	if code := iscFail(t, ts, "CreateAssessmentTarget", `{"assessmentTargetName":"web-target"}`); code != "InvalidInputException/ASSESSMENT_TARGET_NAME_ALREADY_TAKEN" {
		t.Fatalf("duplicate target: %s", code)
	}
	if code := iscFail(t, ts, "CreateAssessmentTarget", `{"assessmentTargetName":"x","resourceGroupArn":"arn:aws:inspector:us-east-1:123456789012:resourcegroup/0-AAAAAAAA"}`); code != "NoSuchEntityException/RESOURCE_GROUP_DOES_NOT_EXIST" {
		t.Fatalf("missing group: %s", code)
	}

	// Preview: the running web instance is an agent, the stopped one an exclusion.
	agents := iscOK(t, ts, "PreviewAgents", fmt.Sprintf(`{"previewAgentsArn":%q}`, target))["agentPreviews"].([]any)
	if len(agents) != 2 {
		t.Fatalf("agent previews: %v", agents)
	}

	if code := iscFail(t, ts, "CreateAssessmentTemplate", fmt.Sprintf(`{"assessmentTargetArn":%q,"assessmentTemplateName":"t","durationInSeconds":60,"rulesPackageArns":[]}`, target)); code != "InvalidInputException/INVALID_ASSESSMENT_TEMPLATE_DURATION" {
		t.Fatalf("short duration: %s", code)
	}
	if code := iscFail(t, ts, "CreateAssessmentTemplate", fmt.Sprintf(`{"assessmentTargetArn":%q,"assessmentTemplateName":"t","durationInSeconds":3600,"rulesPackageArns":["arn:aws:inspector:us-east-1:316112463485:rulespackage/0-ZZZZZZZZ"]}`, target)); code != "NoSuchEntityException/RULES_PACKAGE_DOES_NOT_EXIST" {
		t.Fatalf("unknown package: %s", code)
	}
	template := iscOK(t, ts, "CreateAssessmentTemplate", fmt.Sprintf(`{"assessmentTargetArn":%q,"assessmentTemplateName":"hourly","durationInSeconds":3600,"rulesPackageArns":[%q],"userAttributesForFindings":[{"key":"team","value":"blue"}]}`, target, iscCVEPackage))["assessmentTemplateArn"].(string)

	preview := iscOK(t, ts, "CreateExclusionsPreview", fmt.Sprintf(`{"assessmentTemplateArn":%q}`, template))["previewToken"].(string)
	got := iscOK(t, ts, "GetExclusionsPreview", fmt.Sprintf(`{"assessmentTemplateArn":%q,"previewToken":%q}`, template, preview))
	if got["previewStatus"] != "COMPLETED" || len(got["exclusionPreviews"].([]any)) != 1 {
		t.Fatalf("exclusions preview: %v", got)
	}

	topic, err := srv.sns.CreateTopic("inspector-events", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code := iscFail(t, ts, "SubscribeToEvent", fmt.Sprintf(`{"resourceArn":%q,"event":"ASSESSMENT_RUN_COMPLETED","topicArn":"arn:aws:sns:us-east-1:123456789012:nope"}`, template)); code != "NoSuchEntityException/SNS_TOPIC_DOES_NOT_EXIST" {
		t.Fatalf("missing topic: %s", code)
	}
	iscOK(t, ts, "SubscribeToEvent", fmt.Sprintf(`{"resourceArn":%q,"event":"ASSESSMENT_RUN_COMPLETED","topicArn":%q}`, template, topic.ARN))
	subs := iscOK(t, ts, "ListEventSubscriptions", fmt.Sprintf(`{"resourceArn":%q}`, template))["subscriptions"].([]any)
	if len(subs) != 1 {
		t.Fatalf("subscriptions: %v", subs)
	}

	run := iscOK(t, ts, "StartAssessmentRun", fmt.Sprintf(`{"assessmentTemplateArn":%q,"assessmentRunName":"first"}`, template))["assessmentRunArn"].(string)
	if code := iscFail(t, ts, "StartAssessmentRun", fmt.Sprintf(`{"assessmentTemplateArn":%q}`, template)); !strings.HasPrefix(code, "AgentsAlreadyRunningAssessmentException") {
		t.Fatalf("second run on busy agents: %s", code)
	}
	if code := iscFail(t, ts, "DeleteAssessmentTemplate", fmt.Sprintf(`{"assessmentTemplateArn":%q}`, template)); !strings.HasPrefix(code, "AssessmentRunInProgressException") {
		t.Fatalf("delete busy template: %s", code)
	}
	if code := iscFail(t, ts, "GetAssessmentReport", fmt.Sprintf(`{"assessmentRunArn":%q,"reportFileFormat":"HTML","reportType":"FULL"}`, run)); !strings.HasPrefix(code, "AssessmentRunInProgressException") {
		t.Fatalf("report before completion: %s", code)
	}

	// Each describe moves the run one phase: COLLECTING_DATA, EVALUATING_RULES, COMPLETED.
	var state string
	var desc map[string]any
	for i := 0; i < 3; i++ {
		desc = iscOK(t, ts, "DescribeAssessmentRuns", fmt.Sprintf(`{"assessmentRunArns":[%q,"arn:aws:inspector:us-east-1:123456789012:target/0-AAAAAAAA/template/0-AAAAAAAA/run/0-AAAAAAAA"]}`, run))
		state = desc["assessmentRuns"].([]any)[0].(map[string]any)["state"].(string)
	}
	if state != "COMPLETED" || len(desc["failedItems"].(map[string]any)) != 1 {
		t.Fatalf("run after 3 reads: %s %v", state, desc["failedItems"])
	}
	runOut := desc["assessmentRuns"].([]any)[0].(map[string]any)
	if counts := runOut["findingCounts"].(map[string]any); counts["High"] != float64(1) {
		t.Fatalf("finding counts: %v", counts)
	}
	if notes := runOut["notifications"].([]any); len(notes) != 1 || notes[0].(map[string]any)["event"] != "ASSESSMENT_RUN_COMPLETED" {
		t.Fatalf("notifications: %v", notes)
	}

	findings := iscOK(t, ts, "ListFindings", fmt.Sprintf(`{"assessmentRunArns":[%q],"filter":{"severities":["High"],"agentIds":[%q]}}`, run, web))["findingArns"].([]any)
	if len(findings) != 1 {
		t.Fatalf("findings: %v", findings)
	}
	if none := iscOK(t, ts, "ListFindings", `{"filter":{"severities":["Low"]}}`)["findingArns"].([]any); len(none) != 0 {
		t.Fatalf("low findings: %v", none)
	}
	finding := findings[0].(string)
	iscOK(t, ts, "AddAttributesToFindings", fmt.Sprintf(`{"findingArns":[%q],"attributes":[{"key":"ticket","value":"SEC-1"}]}`, finding))
	f := iscOK(t, ts, "DescribeFindings", fmt.Sprintf(`{"findingArns":[%q]}`, finding))["findings"].([]any)[0].(map[string]any)
	if len(f["userAttributes"].([]any)) != 2 || f["severity"] != "High" || f["assetAttributes"].(map[string]any)["agentId"] != web {
		t.Fatalf("finding: %v", f)
	}
	failed := iscOK(t, ts, "RemoveAttributesFromFindings", fmt.Sprintf(`{"findingArns":[%q,"bad"],"attributeKeys":["ticket"]}`, finding))["failedItems"].(map[string]any)
	if failed["bad"].(map[string]any)["failureCode"] != "INVALID_ARN" {
		t.Fatalf("failed items: %v", failed)
	}

	exclusions := iscOK(t, ts, "ListExclusions", fmt.Sprintf(`{"assessmentRunArn":%q}`, run))["exclusionArns"].([]any)
	if len(exclusions) != 1 {
		t.Fatalf("exclusions: %v", exclusions)
	}
	ex := iscOK(t, ts, "DescribeExclusions", fmt.Sprintf(`{"exclusionArns":[%q]}`, exclusions[0]))["exclusions"].(map[string]any)
	if !strings.Contains(fmt.Sprint(ex), stopped) {
		t.Fatalf("exclusion: %v", ex)
	}

	report := iscOK(t, ts, "GetAssessmentReport", fmt.Sprintf(`{"assessmentRunArn":%q,"reportFileFormat":"HTML","reportType":"FINDING"}`, run))
	resp, err := http.Get(report["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "text/html; charset=utf-8" || !strings.Contains(string(html), "CVE-2021-3156") {
		t.Fatalf("report: %d %s %s", resp.StatusCode, resp.Header.Get("Content-Type"), html)
	}
	pdf := iscOK(t, ts, "GetAssessmentReport", fmt.Sprintf(`{"assessmentRunArn":%q,"reportFileFormat":"PDF","reportType":"FULL"}`, run))
	resp, err = http.Get(pdf["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	pdfBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.HasPrefix(string(pdfBody), "%PDF-1.4") || resp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("pdf report: %s", pdfBody[:min(20, len(pdfBody))])
	}

	agentsOut := iscOK(t, ts, "ListAssessmentRunAgents", fmt.Sprintf(`{"assessmentRunArn":%q}`, run))["assessmentRunAgents"].([]any)
	if len(agentsOut) != 1 || agentsOut[0].(map[string]any)["agentHealthCode"] != "IDLE" {
		t.Fatalf("run agents: %v", agentsOut)
	}
	if runs := iscOK(t, ts, "ListAssessmentRuns", `{"filter":{"states":["COMPLETED"],"namePattern":"fir*"}}`)["assessmentRunArns"].([]any); len(runs) != 1 {
		t.Fatalf("list runs: %v", runs)
	}

	iscOK(t, ts, "SetTagsForResource", fmt.Sprintf(`{"resourceArn":%q,"tags":[{"key":"env","value":"dev"}]}`, template))
	if tags := iscOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"resourceArn":%q}`, template))["tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	if code := iscFail(t, ts, "SetTagsForResource", fmt.Sprintf(`{"resourceArn":%q}`, target)); code != "InvalidInputException/INVALID_RESOURCE_ARN" {
		t.Fatalf("tagging a target: %s", code)
	}

	iscOK(t, ts, "DeleteAssessmentRun", fmt.Sprintf(`{"assessmentRunArn":%q}`, run))
	if code := iscFail(t, ts, "GetTelemetryMetadata", fmt.Sprintf(`{"assessmentRunArn":%q}`, run)); code != "NoSuchEntityException/ASSESSMENT_RUN_DOES_NOT_EXIST" {
		t.Fatalf("deleted run: %s", code)
	}
	iscOK(t, ts, "DeleteAssessmentTarget", fmt.Sprintf(`{"assessmentTargetArn":%q}`, target))
	if left := iscOK(t, ts, "ListAssessmentTemplates", `{}`)["assessmentTemplateArns"].([]any); len(left) != 0 {
		t.Fatalf("templates after target delete: %v", left)
	}
}

func TestInspectorStage1StopAndRole(t *testing.T) {
	srv, ts := newInspectorTestServer(t)
	iscRunInstance(t, srv)
	target := iscOK(t, ts, "CreateAssessmentTarget", `{"assessmentTargetName":"all"}`)["assessmentTargetArn"].(string)
	template := iscOK(t, ts, "CreateAssessmentTemplate", fmt.Sprintf(`{"assessmentTargetArn":%q,"assessmentTemplateName":"t","durationInSeconds":900,"rulesPackageArns":[%q]}`, target, iscCVEPackage))["assessmentTemplateArn"].(string)
	run := iscOK(t, ts, "StartAssessmentRun", fmt.Sprintf(`{"assessmentTemplateArn":%q}`, template))["assessmentRunArn"].(string)
	iscOK(t, ts, "StopAssessmentRun", fmt.Sprintf(`{"assessmentRunArn":%q,"stopAction":"SKIP_EVALUATION"}`, run))
	desc := iscOK(t, ts, "DescribeAssessmentRuns", fmt.Sprintf(`{"assessmentRunArns":[%q]}`, run))["assessmentRuns"].([]any)[0].(map[string]any)
	if desc["state"] != "CANCELED" || desc["findingCounts"].(map[string]any)["High"] != float64(0) {
		t.Fatalf("skipped run: %v", desc)
	}
	if code := iscFail(t, ts, "GetAssessmentReport", fmt.Sprintf(`{"assessmentRunArn":%q,"reportFileFormat":"HTML","reportType":"FULL"}`, run)); code != "UnsupportedFeatureException/" {
		t.Fatalf("report on canceled run: %s", code)
	}

	if role := iscOK(t, ts, "DescribeCrossAccountAccessRole", `{}`); role["valid"] != false {
		t.Fatalf("role before register: %v", role)
	}
	if code := iscFail(t, ts, "RegisterCrossAccountAccessRole", `{"roleArn":"not-an-arn"}`); code != "InvalidInputException/INVALID_IAM_ROLE_ARN" {
		t.Fatalf("bad role: %s", code)
	}
	iscOK(t, ts, "RegisterCrossAccountAccessRole", `{"roleArn":"arn:aws:iam::123456789012:role/inspector"}`)
	if role := iscOK(t, ts, "DescribeCrossAccountAccessRole", `{}`); role["valid"] != true {
		t.Fatalf("role after register: %v", role)
	}
	if pkgs := iscOK(t, ts, "ListRulesPackages", `{}`)["rulesPackageArns"].([]any); len(pkgs) != 4 {
		t.Fatalf("rules packages: %v", pkgs)
	}
	described := iscOK(t, ts, "DescribeRulesPackages", fmt.Sprintf(`{"rulesPackageArns":[%q]}`, iscCVEPackage))["rulesPackages"].([]any)
	if described[0].(map[string]any)["provider"] != "Amazon Web Services, Inc." {
		t.Fatalf("rules package: %v", described)
	}
	// Unknown members are refused, so callers can't set internal state.
	if code := iscFail(t, ts, "ListRulesPackages", `{"state":"x"}`); !strings.HasPrefix(code, "InvalidInputException") {
		t.Fatalf("unknown member: %s", code)
	}
}
