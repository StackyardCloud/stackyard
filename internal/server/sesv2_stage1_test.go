package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func decodeSESV2Body(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(mustBody(t, resp)))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decode SESv2 body: %v", err)
	}
	return out
}

func TestSESV2Stage1EventDestinationAndIdentityShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sesv2Request(t, ts, http.MethodPost, "/v2/email/configuration-sets", []byte(`{"ConfigurationSetName":"demo-config"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/configuration-sets/demo-config/event-destinations", []byte(`{
		"EventDestinationName":"demo-destination",
		"EventDestination":{
			"Enabled":true,
			"MatchingEventTypes":["SEND"],
			"SnsDestination":{"TopicArn":"arn:aws:sns:us-east-1:123456789012:demo-topic"}
		}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/configuration-sets/demo-config/event-destinations", nil)
	assertStatus(t, resp, http.StatusOK)
	destinationsBody := decodeSESV2Body(t, resp)
	destinations, ok := destinationsBody["EventDestinations"].([]any)
	if !ok || len(destinations) != 1 {
		t.Fatalf("expected one event destination, got %#v", destinationsBody["EventDestinations"])
	}
	destination, _ := destinations[0].(map[string]any)
	if got, _ := destination["Name"].(string); got != "demo-destination" {
		t.Fatalf("expected event destination name, got %#v", destination["Name"])
	}
	matchingEventTypes, ok := destination["MatchingEventTypes"].([]any)
	if !ok || len(matchingEventTypes) != 1 || matchingEventTypes[0] != "SEND" {
		t.Fatalf("expected MatchingEventTypes [SEND], got %#v", destination["MatchingEventTypes"])
	}
	snsDestination, _ := destination["SnsDestination"].(map[string]any)
	if got, _ := snsDestination["TopicArn"].(string); got == "" {
		t.Fatalf("expected SnsDestination.TopicArn, got %#v", snsDestination["TopicArn"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/identities", []byte(`{"EmailIdentity":"sender@example.com"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = sesv2Request(t, ts, http.MethodPut, "/v1/email/identities/sender%40example.com/dkim/signing", []byte(`{
		"SigningAttributesOrigin":"AWS_SES",
		"SigningAttributes":{"NextSigningKeyLength":"RSA_2048_BIT"}
	}`))
	assertStatus(t, resp, http.StatusOK)
	dkimBody := decodeSESV2Body(t, resp)
	if got, _ := dkimBody["DkimStatus"].(string); got == "" {
		t.Fatalf("expected DkimStatus, got %#v", dkimBody["DkimStatus"])
	}
	dkimTokens, ok := dkimBody["DkimTokens"].([]any)
	if !ok || len(dkimTokens) == 0 {
		t.Fatalf("expected DkimTokens, got %#v", dkimBody["DkimTokens"])
	}
	if _, ok := dkimBody["SigningHostedZone"]; ok {
		t.Fatalf("expected SigningHostedZone to be omitted, got %#v", dkimBody["SigningHostedZone"])
	}
}

func TestSESV2Stage1DeliverabilityAndAnalyticsShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sesv2Request(t, ts, http.MethodPost, "/v2/email/deliverability-dashboard/test", []byte(`{
		"FromEmailAddress":"sender@example.com",
		"Content":{"Simple":{"Subject":{"Data":"stackyard subject"},"Body":{"Text":{"Data":"stackyard body"}}}}
	}`))
	assertStatus(t, resp, http.StatusOK)
	createReportBody := decodeSESV2Body(t, resp)
	reportID, _ := createReportBody["ReportId"].(string)
	if reportID == "" {
		t.Fatalf("expected ReportId, got %#v", createReportBody["ReportId"])
	}
	if got, _ := createReportBody["DeliverabilityTestStatus"].(string); got != "COMPLETED" {
		t.Fatalf("expected DeliverabilityTestStatus COMPLETED, got %#v", createReportBody["DeliverabilityTestStatus"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/deliverability-dashboard/test-reports/"+url.PathEscape(reportID), nil)
	assertStatus(t, resp, http.StatusOK)
	reportBody := decodeSESV2Body(t, resp)
	if _, ok := reportBody["DeliverabilityTestReport"].(map[string]any); !ok {
		t.Fatalf("expected DeliverabilityTestReport object, got %#v", reportBody["DeliverabilityTestReport"])
	}
	if _, ok := reportBody["OverallPlacement"].(map[string]any); !ok {
		t.Fatalf("expected OverallPlacement object, got %#v", reportBody["OverallPlacement"])
	}
	if ispPlacements, ok := reportBody["IspPlacements"].([]any); !ok || len(ispPlacements) == 0 {
		t.Fatalf("expected IspPlacements, got %#v", reportBody["IspPlacements"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/deliverability-dashboard/blacklist-report", nil)
	assertStatus(t, resp, http.StatusOK)
	blacklistBody := decodeSESV2Body(t, resp)
	if _, ok := blacklistBody["BlacklistReport"].(map[string]any); !ok {
		t.Fatalf("expected BlacklistReport, got %#v", blacklistBody["BlacklistReport"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/deliverability-dashboard", nil)
	assertStatus(t, resp, http.StatusOK)
	optionsBody := decodeSESV2Body(t, resp)
	if got, ok := optionsBody["DashboardEnabled"].(bool); !ok || !got {
		t.Fatalf("expected DashboardEnabled true, got %#v", optionsBody["DashboardEnabled"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/deliverability-dashboard/campaigns/sesv2-campaign-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	campaignBody := decodeSESV2Body(t, resp)
	campaign, _ := campaignBody["DomainDeliverabilityCampaign"].(map[string]any)
	if got, _ := campaign["CampaignId"].(string); got != "sesv2-campaign-000001" {
		t.Fatalf("expected campaign id, got %#v", campaign["CampaignId"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/deliverability-dashboard/domains/example.com/campaigns", nil)
	assertStatus(t, resp, http.StatusOK)
	listCampaignsBody := decodeSESV2Body(t, resp)
	if campaigns, ok := listCampaignsBody["DomainDeliverabilityCampaigns"].([]any); !ok || len(campaigns) == 0 {
		t.Fatalf("expected DomainDeliverabilityCampaigns, got %#v", listCampaignsBody["DomainDeliverabilityCampaigns"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/deliverability-dashboard/statistics-report/example.com", nil)
	assertStatus(t, resp, http.StatusOK)
	statsBody := decodeSESV2Body(t, resp)
	if _, ok := statsBody["OverallVolume"].(map[string]any); !ok {
		t.Fatalf("expected OverallVolume, got %#v", statsBody["OverallVolume"])
	}
	if dailyVolumes, ok := statsBody["DailyVolumes"].([]any); !ok || len(dailyVolumes) == 0 {
		t.Fatalf("expected DailyVolumes, got %#v", statsBody["DailyVolumes"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/vdm/recommendations", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	recommendationsBody := decodeSESV2Body(t, resp)
	if recommendations, ok := recommendationsBody["Recommendations"].([]any); !ok || len(recommendations) == 0 {
		t.Fatalf("expected Recommendations, got %#v", recommendationsBody["Recommendations"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/reputation/entities", []byte(`{"Filter":{"ReputationEntityType":"IP_ADDRESS"}}`))
	assertStatus(t, resp, http.StatusOK)
	reputationBody := decodeSESV2Body(t, resp)
	if entities, ok := reputationBody["ReputationEntities"].([]any); !ok || len(entities) == 0 {
		t.Fatalf("expected ReputationEntities, got %#v", reputationBody["ReputationEntities"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/metrics/batch", []byte(`{
		"Queries":[{"Id":"delivery-rate","Namespace":"VDM","Metric":"SEND","StartDate":"2025-01-01T00:00:00Z","EndDate":"2025-01-02T00:00:00Z"}]
	}`))
	assertStatus(t, resp, http.StatusOK)
	metricBody := decodeSESV2Body(t, resp)
	if results, ok := metricBody["Results"].([]any); !ok || len(results) != 1 {
		t.Fatalf("expected one metric result, got %#v", metricBody["Results"])
	}
}

func TestSESV2Stage1JobTemplateAndDedicatedIPShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sesv2Request(t, ts, http.MethodPost, "/v2/email/export-jobs", []byte(`{
		"ExportDataSource":{"MetricsDataSource":{"Dimensions":{"EMAIL_IDENTITY":["sender@example.com"]},"Namespace":"VDM","Metrics":[{"Name":"SEND","Aggregation":"VOLUME"}],"StartDate":"2025-01-01T00:00:00Z","EndDate":"2025-01-02T00:00:00Z"}},
		"ExportDestination":{"DataFormat":"CSV","S3Url":"s3://stackyard-bucket/exports/report.csv"}
	}`))
	assertStatus(t, resp, http.StatusOK)
	createExportBody := decodeSESV2Body(t, resp)
	exportJobID, _ := createExportBody["JobId"].(string)
	if exportJobID == "" {
		t.Fatalf("expected export JobId, got %#v", createExportBody["JobId"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/export-jobs/"+url.PathEscape(exportJobID), nil)
	assertStatus(t, resp, http.StatusOK)
	exportJobBody := decodeSESV2Body(t, resp)
	if got, _ := exportJobBody["JobId"].(string); got != exportJobID {
		t.Fatalf("expected export JobId %q, got %#v", exportJobID, exportJobBody["JobId"])
	}
	exportDestination, _ := exportJobBody["ExportDestination"].(map[string]any)
	if got, _ := exportDestination["DataFormat"].(string); got != "CSV" {
		t.Fatalf("expected export destination DataFormat CSV, got %#v", exportDestination["DataFormat"])
	}
	exportDataSource, _ := exportJobBody["ExportDataSource"].(map[string]any)
	metricsDataSource, _ := exportDataSource["MetricsDataSource"].(map[string]any)
	dimensions, _ := metricsDataSource["Dimensions"].(map[string]any)
	emailIdentityDimensions, _ := dimensions["EMAIL_IDENTITY"].([]any)
	if len(emailIdentityDimensions) != 1 || emailIdentityDimensions[0] != "sender@example.com" {
		t.Fatalf("expected EMAIL_IDENTITY dimension [sender@example.com], got %#v", dimensions["EMAIL_IDENTITY"])
	}
	metrics, _ := metricsDataSource["Metrics"].([]any)
	if len(metrics) != 1 {
		t.Fatalf("expected one metric entry, got %#v", metricsDataSource["Metrics"])
	}
	metric, _ := metrics[0].(map[string]any)
	if got, _ := metric["Name"].(string); got != "SEND" {
		t.Fatalf("expected export metric SEND, got %#v", metric["Name"])
	}
	if got, _ := metric["Aggregation"].(string); got != "VOLUME" {
		t.Fatalf("expected export metric aggregation VOLUME, got %#v", metric["Aggregation"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/list-export-jobs", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listExportBody := decodeSESV2Body(t, resp)
	if exportJobs, ok := listExportBody["ExportJobs"].([]any); !ok || len(exportJobs) == 0 {
		t.Fatalf("expected ExportJobs, got %#v", listExportBody["ExportJobs"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/import-jobs", []byte(`{
		"ImportDestination":{"SuppressionListDestination":{"SuppressionListImportAction":"PUT"}},
		"ImportDataSource":{"S3Url":"s3://stackyard-bucket/imports/contacts.csv","DataFormat":"CSV"}
	}`))
	assertStatus(t, resp, http.StatusOK)
	createImportBody := decodeSESV2Body(t, resp)
	importJobID, _ := createImportBody["JobId"].(string)
	if importJobID == "" {
		t.Fatalf("expected import JobId, got %#v", createImportBody["JobId"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/import-jobs/"+url.PathEscape(importJobID), nil)
	assertStatus(t, resp, http.StatusOK)
	importJobBody := decodeSESV2Body(t, resp)
	if got, _ := importJobBody["JobId"].(string); got != importJobID {
		t.Fatalf("expected import JobId %q, got %#v", importJobID, importJobBody["JobId"])
	}
	importDataSource, _ := importJobBody["ImportDataSource"].(map[string]any)
	if got, _ := importDataSource["DataFormat"].(string); got != "CSV" {
		t.Fatalf("expected import data format CSV, got %#v", importDataSource["DataFormat"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/import-jobs/list", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listImportBody := decodeSESV2Body(t, resp)
	if importJobs, ok := listImportBody["ImportJobs"].([]any); !ok || len(importJobs) == 0 {
		t.Fatalf("expected ImportJobs, got %#v", listImportBody["ImportJobs"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/custom-verification-email-templates/stackyard-sesv2-custom-verify", nil)
	assertStatus(t, resp, http.StatusOK)
	customTemplateBody := decodeSESV2Body(t, resp)
	if got, _ := customTemplateBody["TemplateName"].(string); got != "stackyard-sesv2-custom-verify" {
		t.Fatalf("expected custom template name, got %#v", customTemplateBody["TemplateName"])
	}
	if _, ok := customTemplateBody["Tags"]; ok {
		t.Fatalf("expected GetCustomVerificationEmailTemplate to omit Tags, got %#v", customTemplateBody["Tags"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/custom-verification-email-templates", nil)
	assertStatus(t, resp, http.StatusOK)
	listCustomTemplatesBody := decodeSESV2Body(t, resp)
	if templates, ok := listCustomTemplatesBody["CustomVerificationEmailTemplates"].([]any); !ok || len(templates) == 0 {
		t.Fatalf("expected CustomVerificationEmailTemplates, got %#v", listCustomTemplatesBody["CustomVerificationEmailTemplates"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/dedicated-ips/198.51.100.10", nil)
	assertStatus(t, resp, http.StatusOK)
	dedicatedIPBody := decodeSESV2Body(t, resp)
	dedicatedIP, _ := dedicatedIPBody["DedicatedIp"].(map[string]any)
	if got, _ := dedicatedIP["Ip"].(string); got != "198.51.100.10" {
		t.Fatalf("expected dedicated IP value, got %#v", dedicatedIP["Ip"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/dedicated-ip-pools/stackyard-sesv2-pool", nil)
	assertStatus(t, resp, http.StatusOK)
	dedicatedPoolBody := decodeSESV2Body(t, resp)
	dedicatedPool, _ := dedicatedPoolBody["DedicatedIpPool"].(map[string]any)
	if got, _ := dedicatedPool["PoolName"].(string); got != "stackyard-sesv2-pool" {
		t.Fatalf("expected dedicated IP pool name, got %#v", dedicatedPool["PoolName"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/dedicated-ips", nil)
	assertStatus(t, resp, http.StatusOK)
	dedicatedIPsBody := decodeSESV2Body(t, resp)
	if dedicatedIPs, ok := dedicatedIPsBody["DedicatedIps"].([]any); !ok || len(dedicatedIPs) == 0 {
		t.Fatalf("expected DedicatedIps, got %#v", dedicatedIPsBody["DedicatedIps"])
	}

	resp = sesv2Request(t, ts, http.MethodGet, "/v2/email/dedicated-ip-pools", nil)
	assertStatus(t, resp, http.StatusOK)
	dedicatedPoolsBody := decodeSESV2Body(t, resp)
	if dedicatedPools, ok := dedicatedPoolsBody["DedicatedIpPools"].([]any); !ok || len(dedicatedPools) == 0 {
		t.Fatalf("expected DedicatedIpPools, got %#v", dedicatedPoolsBody["DedicatedIpPools"])
	}

	resp = sesv2Request(t, ts, http.MethodPost, "/v2/email/outbound-custom-verification-emails", []byte(`{
		"EmailAddress":"recipient@example.com",
		"TemplateName":"stackyard-sesv2-custom-verify"
	}`))
	assertStatus(t, resp, http.StatusOK)
	sendCustomVerificationBody := decodeSESV2Body(t, resp)
	if got, _ := sendCustomVerificationBody["MessageId"].(string); got == "" {
		t.Fatalf("expected MessageId, got %#v", sendCustomVerificationBody["MessageId"])
	}
}
