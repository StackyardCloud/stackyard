package server

type mqDataType struct {
	Name string
}

// Amazon MQ data types sourced from:
// https://docs.aws.amazon.com/amazon-mq/latest/api-reference/resources.html
var mqDataTypes = []mqDataType{
	{Name: "ActionRequired"},
	{Name: "AuthenticationStrategy"},
	{Name: "AvailabilityZone"},
	{Name: "BrokerEngineType"},
	{Name: "BrokerEngineTypeOutput"},
	{Name: "BrokerInstance"},
	{Name: "BrokerInstanceOption"},
	{Name: "BrokerInstanceOptionsOutput"},
	{Name: "BrokerState"},
	{Name: "BrokerStorageConfiguration"},
	{Name: "BrokerStorageType"},
	{Name: "BrokerSummary"},
	{Name: "ChangeType"},
	{Name: "Configuration"},
	{Name: "ConfigurationId"},
	{Name: "ConfigurationRevision"},
	{Name: "Configurations"},
	{Name: "CreateBrokerInput"},
	{Name: "CreateBrokerOutput"},
	{Name: "CreateConfigurationInput"},
	{Name: "CreateConfigurationOutput"},
	{Name: "CreateUserInput"},
	{Name: "DataReplicationCounterpart"},
	{Name: "DataReplicationMetadataOutput"},
	{Name: "DataReplicationMode"},
	{Name: "DeleteBrokerOutput"},
	{Name: "DeleteConfigurationOutput"},
	{Name: "DeploymentMode"},
	{Name: "DescribeBrokerOutput"},
	{Name: "DescribeConfigurationRevisionOutput"},
	{Name: "DescribeUserOutput"},
	{Name: "EfsBrokerStorageConfiguration"},
	{Name: "EncryptionOptions"},
	{Name: "EngineType"},
	{Name: "EngineVersion"},
	{Name: "Error"},
	{Name: "LdapServerMetadataInput"},
	{Name: "LdapServerMetadataOutput"},
	{Name: "ListBrokersOutput"},
	{Name: "ListConfigurationRevisionsOutput"},
	{Name: "ListConfigurationsOutput"},
	{Name: "ListUsersOutput"},
	{Name: "Logs"},
	{Name: "LogsSummary"},
	{Name: "PendingLogs"},
	{Name: "PromoteInput"},
	{Name: "PromoteMode"},
	{Name: "PromoteOutput"},
	{Name: "SanitizationWarning"},
	{Name: "SanitizationWarningReason"},
	{Name: "Tags"},
	{Name: "UpdateBrokerInput"},
	{Name: "UpdateBrokerOutput"},
	{Name: "UpdateConfigurationInput"},
	{Name: "UpdateConfigurationOutput"},
	{Name: "UpdateUserInput"},
	{Name: "User"},
	{Name: "UserPendingChanges"},
	{Name: "UserSummary"},
	{Name: "WeeklyStartTime"},
}

var mqDataTypeByName = func() map[string]mqDataType {
	out := make(map[string]mqDataType, len(mqDataTypes))
	for _, dt := range mqDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
