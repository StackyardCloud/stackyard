package server

type m2DataType struct {
	Name string
}

// AWS Mainframe Modernization data types sourced from:
// https://docs.aws.amazon.com/m2/latest/APIReference/API_Types.html
var m2DataTypes = []m2DataType{
	{Name: "AlternateKey"},
	{Name: "ApplicationSummary"},
	{Name: "ApplicationVersionSummary"},
	{Name: "BatchJobDefinition"},
	{Name: "BatchJobExecutionSummary"},
	{Name: "BatchJobIdentifier"},
	{Name: "DataSet"},
	{Name: "DataSetExportConfig"},
	{Name: "DataSetExportItem"},
	{Name: "DataSetExportSummary"},
	{Name: "DataSetExportTask"},
	{Name: "DataSetImportConfig"},
	{Name: "DataSetImportItem"},
	{Name: "DataSetImportSummary"},
	{Name: "DataSetImportTask"},
	{Name: "DataSetSummary"},
	{Name: "DatasetDetailOrgAttributes"},
	{Name: "DatasetOrgAttributes"},
	{Name: "Definition"},
	{Name: "DeployedVersionSummary"},
	{Name: "DeploymentSummary"},
	{Name: "EfsStorageConfiguration"},
	{Name: "EngineVersionsSummary"},
	{Name: "EnvironmentSummary"},
	{Name: "ExternalLocation"},
	{Name: "FileBatchJobDefinition"},
	{Name: "FileBatchJobIdentifier"},
	{Name: "FsxStorageConfiguration"},
	{Name: "GdgAttributes"},
	{Name: "GdgDetailAttributes"},
	{Name: "HighAvailabilityConfig"},
	{Name: "JobIdentifier"},
	{Name: "JobStep"},
	{Name: "JobStepRestartMarker"},
	{Name: "LogGroupSummary"},
	{Name: "MaintenanceSchedule"},
	{Name: "PendingMaintenance"},
	{Name: "PoAttributes"},
	{Name: "PoDetailAttributes"},
	{Name: "PrimaryKey"},
	{Name: "PsAttributes"},
	{Name: "PsDetailAttributes"},
	{Name: "RecordLength"},
	{Name: "RestartBatchJobIdentifier"},
	{Name: "S3BatchJobIdentifier"},
	{Name: "ScriptBatchJobDefinition"},
	{Name: "ScriptBatchJobIdentifier"},
	{Name: "StorageConfiguration"},
	{Name: "UpdateEnvironment"},
	{Name: "ValidationExceptionField"},
	{Name: "VsamAttributes"},
	{Name: "VsamDetailAttributes"},
}

var m2DataTypeByName = func() map[string]m2DataType {
	out := make(map[string]m2DataType, len(m2DataTypes))
	for _, dt := range m2DataTypes {
		out[dt.Name] = dt
	}
	return out
}()
