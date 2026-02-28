package server

type cleanRoomsMLDataType struct {
	Name string
}

// AWS Clean Rooms ML data types sourced from:
// https://docs.aws.amazon.com/cleanrooms-ml/latest/APIReference/API_Types.html
var cleanRoomsMLDataTypes = []cleanRoomsMLDataType{
	{Name: "AccessBudget"},
	{Name: "AccessBudgetDetails"},
	{Name: "AudienceDestination"},
	{Name: "AudienceExportJobSummary"},
	{Name: "AudienceGenerationJobDataSource"},
	{Name: "AudienceGenerationJobSummary"},
	{Name: "AudienceModelSummary"},
	{Name: "AudienceQualityMetrics"},
	{Name: "AudienceSize"},
	{Name: "AudienceSizeConfig"},
	{Name: "CollaborationConfiguredModelAlgorithmAssociationSummary"},
	{Name: "CollaborationMLInputChannelSummary"},
	{Name: "CollaborationTrainedModelExportJobSummary"},
	{Name: "CollaborationTrainedModelInferenceJobSummary"},
	{Name: "CollaborationTrainedModelSummary"},
	{Name: "ColumnClassificationDetails"},
	{Name: "ColumnSchema"},
	{Name: "ComputeConfiguration"},
	{Name: "ConfiguredAudienceModelOutputConfig"},
	{Name: "ConfiguredAudienceModelSummary"},
	{Name: "ConfiguredModelAlgorithmAssociationSummary"},
	{Name: "ConfiguredModelAlgorithmSummary"},
	{Name: "ContainerConfig"},
	{Name: "CustomEntityConfig"},
	{Name: "DataPrivacyScores"},
	{Name: "Dataset"},
	{Name: "DatasetInputConfig"},
	{Name: "DataSource"},
	{Name: "Destination"},
	{Name: "GlueDataSource"},
	{Name: "IncrementalTrainingDataChannel"},
	{Name: "IncrementalTrainingDataChannelOutput"},
	{Name: "InferenceContainerConfig"},
	{Name: "InferenceContainerExecutionParameters"},
	{Name: "InferenceOutputConfiguration"},
	{Name: "InferenceReceiverMember"},
	{Name: "InferenceResourceConfig"},
	{Name: "InputChannel"},
	{Name: "InputChannelDataSource"},
	{Name: "LogRedactionConfiguration"},
	{Name: "LogsConfigurationPolicy"},
	{Name: "MembershipInferenceAttackScore"},
	{Name: "MetricDefinition"},
	{Name: "MetricsConfigurationPolicy"},
	{Name: "MLInputChannelSummary"},
	{Name: "MLOutputConfiguration"},
	{Name: "MLSyntheticDataParameters"},
	{Name: "ModelInferenceDataSource"},
	{Name: "ModelTrainingDataChannel"},
	{Name: "PrivacyBudgets"},
	{Name: "PrivacyConfiguration"},
	{Name: "PrivacyConfigurationPolicies"},
	{Name: "ProtectedQueryInputParameters"},
	{Name: "ProtectedQuerySQLParameters"},
	{Name: "RelevanceMetric"},
	{Name: "ResourceConfig"},
	{Name: "S3ConfigMap"},
	{Name: "StatusDetails"},
	{Name: "StoppingCondition"},
	{Name: "SyntheticDataColumnProperties"},
	{Name: "SyntheticDataConfiguration"},
	{Name: "SyntheticDataEvaluationScores"},
	{Name: "TrainedModelArtifactMaxSize"},
	{Name: "TrainedModelExportOutputConfiguration"},
	{Name: "TrainedModelExportReceiverMember"},
	{Name: "TrainedModelExportsConfigurationPolicy"},
	{Name: "TrainedModelExportsMaxSize"},
	{Name: "TrainedModelInferenceJobsConfigurationPolicy"},
	{Name: "TrainedModelInferenceJobSummary"},
	{Name: "TrainedModelInferenceMaxOutputSize"},
	{Name: "TrainedModelsConfigurationPolicy"},
	{Name: "TrainedModelSummary"},
	{Name: "TrainingDatasetSummary"},
	{Name: "WorkerComputeConfiguration"},
	{Name: "WorkerComputeConfigurationProperties"},
}

var cleanRoomsMLDataTypeByName = func() map[string]cleanRoomsMLDataType {
	out := make(map[string]cleanRoomsMLDataType, len(cleanRoomsMLDataTypes))
	for _, dt := range cleanRoomsMLDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
