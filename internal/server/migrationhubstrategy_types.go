package server

type migrationHubStrategyDataType struct {
	Name string
}

// AWS Migration Hub Strategy Recommendations data types sourced from:
// https://docs.aws.amazon.com/migrationhub-strategy/latest/APIReference/API_Types.html
var migrationHubStrategyDataTypes = []migrationHubStrategyDataType{
	{Name: "AnalysisStatusUnion"},
	{Name: "AnalyzableServerSummary"},
	{Name: "AnalyzerNameUnion"},
	{Name: "AntipatternReportResult"},
	{Name: "AntipatternSeveritySummary"},
	{Name: "ApplicationComponentDetail"},
	{Name: "ApplicationComponentStatusSummary"},
	{Name: "ApplicationComponentStrategy"},
	{Name: "ApplicationComponentSummary"},
	{Name: "ApplicationPreferences"},
	{Name: "AppUnitError"},
	{Name: "AssessmentSummary"},
	{Name: "AssessmentTarget"},
	{Name: "AssociatedApplication"},
	{Name: "AwsManagedResources"},
	{Name: "BusinessGoals"},
	{Name: "Collector"},
	{Name: "ConfigurationSummary"},
	{Name: "DatabaseConfigDetail"},
	{Name: "DatabaseMigrationPreference"},
	{Name: "DatabasePreferences"},
	{Name: "DataCollectionDetails"},
	{Name: "Group"},
	{Name: "Heterogeneous"},
	{Name: "Homogeneous"},
	{Name: "ImportFileTaskInformation"},
	{Name: "IPAddressBasedRemoteInfo"},
	{Name: "ManagementPreference"},
	{Name: "NetworkInfo"},
	{Name: "NoDatabaseMigrationPreference"},
	{Name: "NoManagementPreference"},
	{Name: "OSInfo"},
	{Name: "PipelineInfo"},
	{Name: "PrioritizeBusinessGoals"},
	{Name: "RecommendationReportDetails"},
	{Name: "RecommendationSet"},
	{Name: "RemoteSourceCodeAnalysisServerInfo"},
	{Name: "Result"},
	{Name: "S3Object"},
	{Name: "SelfManageResources"},
	{Name: "ServerDetail"},
	{Name: "ServerError"},
	{Name: "ServerStatusSummary"},
	{Name: "ServerStrategy"},
	{Name: "ServerSummary"},
	{Name: "SourceCode"},
	{Name: "SourceCodeRepository"},
	{Name: "StrategyOption"},
	{Name: "StrategySummary"},
	{Name: "SystemInfo"},
	{Name: "TransformationTool"},
	{Name: "VcenterBasedRemoteInfo"},
	{Name: "VersionControlInfo"},
}

var migrationHubStrategyDataTypeByName = func() map[string]migrationHubStrategyDataType {
	out := make(map[string]migrationHubStrategyDataType, len(migrationHubStrategyDataTypes))
	for _, dt := range migrationHubStrategyDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
