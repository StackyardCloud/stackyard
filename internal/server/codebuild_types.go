package server

type codeBuildDataType struct {
	Name string
}

// AWS CodeBuild data types sourced from:
// https://docs.aws.amazon.com/codebuild/latest/APIReference/API_Types.html
var codeBuildDataTypes = []codeBuildDataType{
	{Name: "AutoRetryConfig"},
	{Name: "BatchRestrictions"},
	{Name: "Build"},
	{Name: "BuildArtifacts"},
	{Name: "BuildBatch"},
	{Name: "BuildBatchFilter"},
	{Name: "BuildBatchPhase"},
	{Name: "BuildGroup"},
	{Name: "BuildNotDeleted"},
	{Name: "BuildPhase"},
	{Name: "BuildStatusConfig"},
	{Name: "BuildSummary"},
	{Name: "CloudWatchLogsConfig"},
	{Name: "CodeCoverage"},
	{Name: "CodeCoverageReportSummary"},
	{Name: "CommandExecution"},
	{Name: "ComputeConfiguration"},
	{Name: "DebugSession"},
	{Name: "DockerServer"},
	{Name: "DockerServerStatus"},
	{Name: "EnvironmentImage"},
	{Name: "EnvironmentLanguage"},
	{Name: "EnvironmentPlatform"},
	{Name: "EnvironmentVariable"},
	{Name: "ExportedEnvironmentVariable"},
	{Name: "Fleet"},
	{Name: "FleetProxyRule"},
	{Name: "FleetStatus"},
	{Name: "GitSubmodulesConfig"},
	{Name: "LogsConfig"},
	{Name: "LogsLocation"},
	{Name: "NetworkInterface"},
	{Name: "PhaseContext"},
	{Name: "Project"},
	{Name: "ProjectArtifacts"},
	{Name: "ProjectBadge"},
	{Name: "ProjectBuildBatchConfig"},
	{Name: "ProjectCache"},
	{Name: "ProjectEnvironment"},
	{Name: "ProjectFileSystemLocation"},
	{Name: "ProjectFleet"},
	{Name: "ProjectSource"},
	{Name: "ProjectSourceVersion"},
	{Name: "ProxyConfiguration"},
	{Name: "PullRequestBuildPolicy"},
	{Name: "RegistryCredential"},
	{Name: "Report"},
	{Name: "ReportExportConfig"},
	{Name: "ReportFilter"},
	{Name: "ReportGroup"},
	{Name: "ReportGroupTrendStats"},
	{Name: "ReportWithRawData"},
	{Name: "ResolvedArtifact"},
	{Name: "S3LogsConfig"},
	{Name: "S3ReportExportConfig"},
	{Name: "SSMSession"},
	{Name: "Sandbox"},
	{Name: "SandboxSession"},
	{Name: "SandboxSessionPhase"},
	{Name: "ScalingConfigurationInput"},
	{Name: "ScalingConfigurationOutput"},
	{Name: "ScopeConfiguration"},
	{Name: "SourceAuth"},
	{Name: "SourceCredentialsInfo"},
	{Name: "Tag"},
	{Name: "TargetTrackingScalingConfiguration"},
	{Name: "TestCase"},
	{Name: "TestCaseFilter"},
	{Name: "TestReportSummary"},
	{Name: "UpdateWebhook"},
	{Name: "VpcConfig"},
	{Name: "Webhook"},
	{Name: "WebhookFilter"},
}

var codeBuildDataTypeByName = func() map[string]codeBuildDataType {
	out := make(map[string]codeBuildDataType, len(codeBuildDataTypes))
	for _, dt := range codeBuildDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
