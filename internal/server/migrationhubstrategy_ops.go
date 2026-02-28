package server

type migrationHubStrategyOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Migration Hub Strategy Recommendations operations sourced from:
// https://docs.aws.amazon.com/migrationhub-strategy/latest/APIReference/API_Operations.html
var migrationHubStrategyOperations = []migrationHubStrategyOperation{
	{Name: "GetApplicationComponentDetails", Method: "GET", URI: "/get-applicationcomponent-details/{applicationComponentId}"},
	{Name: "GetApplicationComponentStrategies", Method: "GET", URI: "/get-applicationcomponent-strategies/{applicationComponentId}"},
	{Name: "GetAssessment", Method: "GET", URI: "/get-assessment/{id}"},
	{Name: "GetImportFileTask", Method: "GET", URI: "/get-import-file-task/{id}"},
	{Name: "GetLatestAssessmentId", Method: "GET", URI: "/get-latest-assessment-id"},
	{Name: "GetPortfolioPreferences", Method: "GET", URI: "/get-portfolio-preferences"},
	{Name: "GetPortfolioSummary", Method: "GET", URI: "/get-portfolio-summary"},
	{Name: "GetRecommendationReportDetails", Method: "GET", URI: "/get-recommendation-report-details/{id}"},
	{Name: "GetServerDetails", Method: "GET", URI: "/get-server-details/{serverId}"},
	{Name: "GetServerStrategies", Method: "GET", URI: "/get-server-strategies/{serverId}"},
	{Name: "ListAnalyzableServers", Method: "POST", URI: "/list-analyzable-servers"},
	{Name: "ListApplicationComponents", Method: "POST", URI: "/list-applicationcomponents"},
	{Name: "ListCollectors", Method: "GET", URI: "/list-collectors"},
	{Name: "ListImportFileTask", Method: "GET", URI: "/list-import-file-task"},
	{Name: "ListServers", Method: "POST", URI: "/list-servers"},
	{Name: "PutPortfolioPreferences", Method: "POST", URI: "/put-portfolio-preferences"},
	{Name: "StartAssessment", Method: "POST", URI: "/start-assessment"},
	{Name: "StartImportFileTask", Method: "POST", URI: "/start-import-file-task"},
	{Name: "StartRecommendationReportGeneration", Method: "POST", URI: "/start-recommendation-report-generation"},
	{Name: "StopAssessment", Method: "POST", URI: "/stop-assessment"},
	{Name: "UpdateApplicationComponentConfig", Method: "POST", URI: "/update-applicationcomponent-config/"},
	{Name: "UpdateServerConfig", Method: "POST", URI: "/update-server-config/"},
}

var migrationHubStrategyOperationByName = func() map[string]migrationHubStrategyOperation {
	out := make(map[string]migrationHubStrategyOperation, len(migrationHubStrategyOperations))
	for _, op := range migrationHubStrategyOperations {
		out[op.Name] = op
	}
	return out
}()
