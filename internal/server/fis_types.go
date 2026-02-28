package server

type fisDataType struct {
	Name string
}

// AWS Fault Injection Service data types sourced from:
// https://docs.aws.amazon.com/fis/latest/APIReference/API_Types.html
var fisDataTypes = []fisDataType{
	{Name: "Action"},
	{Name: "ActionParameter"},
	{Name: "ActionSummary"},
	{Name: "ActionTarget"},
	{Name: "CreateExperimentTemplateActionInput"},
	{Name: "CreateExperimentTemplateExperimentOptionsInput"},
	{Name: "CreateExperimentTemplateLogConfigurationInput"},
	{Name: "CreateExperimentTemplateReportConfigurationInput"},
	{Name: "CreateExperimentTemplateStopConditionInput"},
	{Name: "CreateExperimentTemplateTargetInput"},
	{Name: "Experiment"},
	{Name: "ExperimentAction"},
	{Name: "ExperimentActionState"},
	{Name: "ExperimentCloudWatchLogsLogConfiguration"},
	{Name: "ExperimentError"},
	{Name: "ExperimentLogConfiguration"},
	{Name: "ExperimentOptions"},
	{Name: "ExperimentReport"},
	{Name: "ExperimentReportConfiguration"},
	{Name: "ExperimentReportConfigurationCloudWatchDashboard"},
	{Name: "ExperimentReportConfigurationDataSources"},
	{Name: "ExperimentReportConfigurationOutputs"},
	{Name: "ExperimentReportConfigurationOutputsS3Configuration"},
	{Name: "ExperimentReportError"},
	{Name: "ExperimentReportS3Report"},
	{Name: "ExperimentReportState"},
	{Name: "ExperimentS3LogConfiguration"},
	{Name: "ExperimentState"},
	{Name: "ExperimentStopCondition"},
	{Name: "ExperimentSummary"},
	{Name: "ExperimentTarget"},
	{Name: "ExperimentTargetAccountConfiguration"},
	{Name: "ExperimentTargetAccountConfigurationSummary"},
	{Name: "ExperimentTargetFilter"},
	{Name: "ExperimentTemplate"},
	{Name: "ExperimentTemplateAction"},
	{Name: "ExperimentTemplateCloudWatchLogsLogConfiguration"},
	{Name: "ExperimentTemplateCloudWatchLogsLogConfigurationInput"},
	{Name: "ExperimentTemplateExperimentOptions"},
	{Name: "ExperimentTemplateLogConfiguration"},
	{Name: "ExperimentTemplateReportConfiguration"},
	{Name: "ExperimentTemplateReportConfigurationCloudWatchDashboard"},
	{Name: "ExperimentTemplateReportConfigurationDataSources"},
	{Name: "ExperimentTemplateReportConfigurationDataSourcesInput"},
	{Name: "ExperimentTemplateReportConfigurationOutputs"},
	{Name: "ExperimentTemplateReportConfigurationOutputsInput"},
	{Name: "ExperimentTemplateS3LogConfiguration"},
	{Name: "ExperimentTemplateS3LogConfigurationInput"},
	{Name: "ExperimentTemplateStopCondition"},
	{Name: "ExperimentTemplateSummary"},
	{Name: "ExperimentTemplateTarget"},
	{Name: "ExperimentTemplateTargetFilter"},
	{Name: "ExperimentTemplateTargetInputFilter"},
	{Name: "ReportConfigurationCloudWatchDashboardInput"},
	{Name: "ReportConfigurationS3Output"},
	{Name: "ReportConfigurationS3OutputInput"},
	{Name: "ResolvedTarget"},
	{Name: "SafetyLever"},
	{Name: "SafetyLeverState"},
	{Name: "StartExperimentExperimentOptionsInput"},
	{Name: "TargetAccountConfiguration"},
	{Name: "TargetAccountConfigurationSummary"},
	{Name: "TargetResourceType"},
	{Name: "TargetResourceTypeParameter"},
	{Name: "TargetResourceTypeSummary"},
	{Name: "UpdateExperimentTemplateActionInputItem"},
	{Name: "UpdateExperimentTemplateExperimentOptionsInput"},
	{Name: "UpdateExperimentTemplateLogConfigurationInput"},
	{Name: "UpdateExperimentTemplateReportConfigurationInput"},
	{Name: "UpdateExperimentTemplateStopConditionInput"},
	{Name: "UpdateExperimentTemplateTargetInput"},
	{Name: "UpdateSafetyLeverStateInput"},
	{Name: "UpdateTargetAccountConfiguration"},
}

var fisDataTypeByName = func() map[string]fisDataType {
	out := make(map[string]fisDataType, len(fisDataTypes))
	for _, t := range fisDataTypes {
		out[t.Name] = t
	}
	return out
}()
