package server

type securityLakeType struct {
	Name string
}

// AWS Security Lake data types sourced from:
// https://docs.aws.amazon.com/security-lake/latest/APIReference/API_Types.html
var securityLakeTypes = []securityLakeType{
	{Name: "AwsIdentity"},
	{Name: "AwsLogSourceConfiguration"},
	{Name: "AwsLogSourceResource"},
	{Name: "CustomLogSourceAttributes"},
	{Name: "CustomLogSourceConfiguration"},
	{Name: "CustomLogSourceCrawlerConfiguration"},
	{Name: "CustomLogSourceProvider"},
	{Name: "CustomLogSourceResource"},
	{Name: "DataLakeAutoEnableNewAccountConfiguration"},
	{Name: "DataLakeConfiguration"},
	{Name: "DataLakeEncryptionConfiguration"},
	{Name: "DataLakeException"},
	{Name: "DataLakeLifecycleConfiguration"},
	{Name: "DataLakeLifecycleExpiration"},
	{Name: "DataLakeLifecycleTransition"},
	{Name: "DataLakeReplicationConfiguration"},
	{Name: "DataLakeResource"},
	{Name: "DataLakeSource"},
	{Name: "DataLakeSourceStatus"},
	{Name: "DataLakeUpdateException"},
	{Name: "DataLakeUpdateStatus"},
	{Name: "HttpsNotificationConfiguration"},
	{Name: "LogSource"},
	{Name: "LogSourceResource"},
	{Name: "NotificationConfiguration"},
	{Name: "SqsNotificationConfiguration"},
	{Name: "SubscriberResource"},
	{Name: "Tag"},
	{Name: "UpdateSubscriberNotification"},
}

var securityLakeTypeByName = func() map[string]securityLakeType {
	out := make(map[string]securityLakeType, len(securityLakeTypes))
	for _, dt := range securityLakeTypes {
		out[dt.Name] = dt
	}
	return out
}()
