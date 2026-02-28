package server

type appFabricDataType struct {
	Name string
}

// AWS AppFabric data types sourced from:
// https://docs.aws.amazon.com/appfabric/latest/api/API_Types.html
var appFabricDataTypes = []appFabricDataType{
	{Name: "ApiKeyCredential"},
	{Name: "AppAuthorization"},
	{Name: "AppAuthorizationSummary"},
	{Name: "AppBundle"},
	{Name: "AppBundleSummary"},
	{Name: "AuditLogDestinationConfiguration"},
	{Name: "AuditLogProcessingConfiguration"},
	{Name: "AuthRequest"},
	{Name: "Credential"},
	{Name: "Destination"},
	{Name: "DestinationConfiguration"},
	{Name: "FirehoseStream"},
	{Name: "Ingestion"},
	{Name: "IngestionDestination"},
	{Name: "IngestionDestinationSummary"},
	{Name: "IngestionSummary"},
	{Name: "Oauth2Credential"},
	{Name: "ProcessingConfiguration"},
	{Name: "S3Bucket"},
	{Name: "Tag"},
	{Name: "TaskError"},
	{Name: "Tenant"},
	{Name: "UpdateIngestionDestination"},
	{Name: "UserAccessResultItem"},
	{Name: "UserAccessTaskItem"},
	{Name: "ValidationExceptionField"},
}

var appFabricDataTypeByName = func() map[string]appFabricDataType {
	out := make(map[string]appFabricDataType, len(appFabricDataTypes))
	for _, dt := range appFabricDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
