package server

type finspaceManagementDataType struct {
	Name string
}

// Amazon FinSpace Management API data types sourced from:
// https://docs.aws.amazon.com/finspace/latest/management-api/API_Types.html
var finspaceManagementDataTypes = []finspaceManagementDataType{
	{Name: "AutoScalingConfiguration"},
	{Name: "CapacityConfiguration"},
	{Name: "ChangeRequest"},
	{Name: "CodeConfiguration"},
	{Name: "CustomDNSServer"},
	{Name: "Environment"},
	{Name: "ErrorInfo"},
	{Name: "FederationParameters"},
	{Name: "IcmpTypeCode"},
	{Name: "KxAttachedCluster"},
	{Name: "KxCacheStorageConfiguration"},
	{Name: "KxChangesetListEntry"},
	{Name: "KxCluster"},
	{Name: "KxClusterCodeDeploymentConfiguration"},
	{Name: "KxCommandLineArgument"},
	{Name: "KxDatabaseCacheConfiguration"},
	{Name: "KxDatabaseConfiguration"},
	{Name: "KxDatabaseListEntry"},
	{Name: "KxDataviewActiveVersion"},
	{Name: "KxDataviewConfiguration"},
	{Name: "KxDataviewListEntry"},
	{Name: "KxDataviewSegmentConfiguration"},
	{Name: "KxDeploymentConfiguration"},
	{Name: "KxEnvironment"},
	{Name: "KxNAS1Configuration"},
	{Name: "KxNode"},
	{Name: "KxSavedownStorageConfiguration"},
	{Name: "KxScalingGroup"},
	{Name: "KxScalingGroupConfiguration"},
	{Name: "KxUser"},
	{Name: "KxVolume"},
	{Name: "NetworkACLEntry"},
	{Name: "PortRange"},
	{Name: "SuperuserParameters"},
	{Name: "TickerplantLogConfiguration"},
	{Name: "TransitGatewayConfiguration"},
	{Name: "Volume"},
	{Name: "VpcConfiguration"},
	{Name: "UpdateKxVolume"},
}

var finspaceManagementDataTypeByName = func() map[string]finspaceManagementDataType {
	out := make(map[string]finspaceManagementDataType, len(finspaceManagementDataTypes))
	for _, dt := range finspaceManagementDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
