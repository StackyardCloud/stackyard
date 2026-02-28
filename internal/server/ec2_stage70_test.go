package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	awsec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestEC2Stage70SDKLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}
	client := awsec2.NewFromConfig(cfg, func(o *awsec2.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	getInitialOut, err := client.GetAllowedImagesSettings(ctx, &awsec2.GetAllowedImagesSettingsInput{})
	if err != nil {
		t.Fatalf("get allowed images settings initial: %v", err)
	}
	if aws.ToString(getInitialOut.State) != "disabled" {
		t.Fatalf("unexpected initial state: %q", aws.ToString(getInitialOut.State))
	}
	if getInitialOut.ManagedBy != awsec2types.ManagedByAccount {
		t.Fatalf("unexpected initial managed by: %q", getInitialOut.ManagedBy)
	}

	replaceCriteriaOut, err := client.ReplaceImageCriteriaInAllowedImagesSettings(ctx, &awsec2.ReplaceImageCriteriaInAllowedImagesSettingsInput{
		ImageCriteria: []awsec2types.ImageCriterionRequest{
			{ImageProviders: []string{"amazon", "aws-marketplace"}},
			{ImageProviders: []string{"123456789012"}},
		},
	})
	if err != nil {
		t.Fatalf("replace image criteria in allowed images settings: %v", err)
	}
	if !aws.ToBool(replaceCriteriaOut.ReturnValue) {
		t.Fatalf("expected true return value from replace image criteria")
	}

	enableOut, err := client.EnableAllowedImagesSettings(ctx, &awsec2.EnableAllowedImagesSettingsInput{
		AllowedImagesSettingsState: awsec2types.AllowedImagesSettingsEnabledStateAuditMode,
	})
	if err != nil {
		t.Fatalf("enable allowed images settings: %v", err)
	}
	if enableOut.AllowedImagesSettingsState != awsec2types.AllowedImagesSettingsEnabledStateAuditMode {
		t.Fatalf("unexpected enabled state: %q", enableOut.AllowedImagesSettingsState)
	}

	getEnabledOut, err := client.GetAllowedImagesSettings(ctx, &awsec2.GetAllowedImagesSettingsInput{})
	if err != nil {
		t.Fatalf("get allowed images settings after enable: %v", err)
	}
	if aws.ToString(getEnabledOut.State) != "audit-mode" {
		t.Fatalf("unexpected state after enable: %q", aws.ToString(getEnabledOut.State))
	}
	if getEnabledOut.ManagedBy != awsec2types.ManagedByAccount {
		t.Fatalf("unexpected managed by after enable: %q", getEnabledOut.ManagedBy)
	}
	if len(getEnabledOut.ImageCriteria) != 2 {
		t.Fatalf("expected two image criteria, got %d", len(getEnabledOut.ImageCriteria))
	}

	disableOut, err := client.DisableAllowedImagesSettings(ctx, &awsec2.DisableAllowedImagesSettingsInput{})
	if err != nil {
		t.Fatalf("disable allowed images settings: %v", err)
	}
	if disableOut.AllowedImagesSettingsState != awsec2types.AllowedImagesSettingsDisabledStateDisabled {
		t.Fatalf("unexpected disabled state: %q", disableOut.AllowedImagesSettingsState)
	}

	startOneOut, err := client.StartDeclarativePoliciesReport(ctx, &awsec2.StartDeclarativePoliciesReportInput{
		S3Bucket: aws.String("stage70-bucket"),
		S3Prefix: aws.String("reports/stage70"),
		TargetId: aws.String("123456789012"),
		TagSpecifications: []awsec2types.TagSpecification{
			{
				ResourceType: awsec2types.ResourceTypeDeclarativePoliciesReport,
				Tags: []awsec2types.Tag{
					{Key: aws.String("env"), Value: aws.String("stage70")},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("start declarative policies report one: %v", err)
	}
	reportOneID := aws.ToString(startOneOut.ReportId)
	if reportOneID == "" {
		t.Fatalf("expected report one id")
	}

	describeOneOut, err := client.DescribeDeclarativePoliciesReports(ctx, &awsec2.DescribeDeclarativePoliciesReportsInput{
		ReportIds: []string{reportOneID},
	})
	if err != nil {
		t.Fatalf("describe declarative policies reports for first report: %v", err)
	}
	if len(describeOneOut.Reports) != 1 {
		t.Fatalf("expected one report in describe output, got %d", len(describeOneOut.Reports))
	}
	if aws.ToString(describeOneOut.Reports[0].ReportId) != reportOneID {
		t.Fatalf("unexpected report id in describe output: %q", aws.ToString(describeOneOut.Reports[0].ReportId))
	}
	if describeOneOut.Reports[0].Status != awsec2types.ReportStateRunning {
		t.Fatalf("unexpected report one status: %q", describeOneOut.Reports[0].Status)
	}

	getSummaryOut, err := client.GetDeclarativePoliciesReportSummary(ctx, &awsec2.GetDeclarativePoliciesReportSummaryInput{
		ReportId: aws.String(reportOneID),
	})
	if err != nil {
		t.Fatalf("get declarative policies report summary: %v", err)
	}
	if aws.ToString(getSummaryOut.ReportId) != reportOneID {
		t.Fatalf("unexpected summary report id: %q", aws.ToString(getSummaryOut.ReportId))
	}
	if aws.ToString(getSummaryOut.TargetId) != "123456789012" {
		t.Fatalf("unexpected summary target id: %q", aws.ToString(getSummaryOut.TargetId))
	}
	if len(getSummaryOut.AttributeSummaries) == 0 {
		t.Fatalf("expected non-empty attribute summaries")
	}

	cancelOut, err := client.CancelDeclarativePoliciesReport(ctx, &awsec2.CancelDeclarativePoliciesReportInput{
		ReportId: aws.String(reportOneID),
	})
	if err != nil {
		t.Fatalf("cancel declarative policies report: %v", err)
	}
	if !aws.ToBool(cancelOut.Return) {
		t.Fatalf("expected cancel return true")
	}

	describeCanceledOut, err := client.DescribeDeclarativePoliciesReports(ctx, &awsec2.DescribeDeclarativePoliciesReportsInput{
		ReportIds: []string{reportOneID},
	})
	if err != nil {
		t.Fatalf("describe canceled report: %v", err)
	}
	if len(describeCanceledOut.Reports) != 1 {
		t.Fatalf("expected one report for canceled report describe, got %d", len(describeCanceledOut.Reports))
	}
	if describeCanceledOut.Reports[0].Status != awsec2types.ReportStateCancelled {
		t.Fatalf("unexpected canceled report status: %q", describeCanceledOut.Reports[0].Status)
	}

	startTwoOut, err := client.StartDeclarativePoliciesReport(ctx, &awsec2.StartDeclarativePoliciesReportInput{
		S3Bucket: aws.String("stage70-bucket"),
		TargetId: aws.String("ou-ab12-cdef1234"),
	})
	if err != nil {
		t.Fatalf("start declarative policies report two: %v", err)
	}
	reportTwoID := aws.ToString(startTwoOut.ReportId)
	if reportTwoID == "" {
		t.Fatalf("expected report two id")
	}

	describePageOneOut, err := client.DescribeDeclarativePoliciesReports(ctx, &awsec2.DescribeDeclarativePoliciesReportsInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		t.Fatalf("describe declarative policies reports page one: %v", err)
	}
	if len(describePageOneOut.Reports) != 1 {
		t.Fatalf("expected one report in page one, got %d", len(describePageOneOut.Reports))
	}
	if describePageOneOut.NextToken == nil {
		t.Fatalf("expected next token in page one")
	}

	describePageTwoOut, err := client.DescribeDeclarativePoliciesReports(ctx, &awsec2.DescribeDeclarativePoliciesReportsInput{
		NextToken: describePageOneOut.NextToken,
	})
	if err != nil {
		t.Fatalf("describe declarative policies reports page two: %v", err)
	}
	if len(describePageTwoOut.Reports) == 0 {
		t.Fatalf("expected reports in page two")
	}

	foundReportTwo := false
	for _, report := range append(describePageOneOut.Reports, describePageTwoOut.Reports...) {
		if aws.ToString(report.ReportId) == reportTwoID {
			foundReportTwo = true
			break
		}
	}
	if !foundReportTwo {
		t.Fatalf("expected to find report two id %q in paginated describe results", reportTwoID)
	}
}

func TestEC2Stage70ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	implemented := []string{
		"CancelDeclarativePoliciesReport",
		"DescribeDeclarativePoliciesReports",
		"DisableAllowedImagesSettings",
		"EnableAllowedImagesSettings",
		"GetAllowedImagesSettings",
		"GetDeclarativePoliciesReportSummary",
		"ReplaceImageCriteriaInAllowedImagesSettings",
		"StartDeclarativePoliciesReport",
	}

	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, action := range implemented {
		params := map[string]string{}
		switch action {
		case "CancelDeclarativePoliciesReport", "GetDeclarativePoliciesReportSummary":
			params["ReportId"] = "dpr-00000000"
		case "EnableAllowedImagesSettings":
			params["AllowedImagesSettingsState"] = "enabled"
		case "ReplaceImageCriteriaInAllowedImagesSettings":
			params["ImageCriterion.1.ImageProvider.1"] = "amazon"
		case "StartDeclarativePoliciesReport":
			params["S3Bucket"] = "stage70-bucket"
			params["TargetId"] = "123456789012"
		}

		resp := ec2Request(t, ts, action, params)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action)
		}
	}
}
