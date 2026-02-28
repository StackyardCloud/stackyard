package server

type odbDataType struct {
	Name string
}

// Oracle Database@AWS data types sourced from:
// https://docs.aws.amazon.com/odb/latest/APIReference/API_Types.html
var odbDataTypes = []odbDataType{
	{Name: "AutonomousVirtualMachineSummary"},
	{Name: "CloudAutonomousVmCluster"},
	{Name: "CloudAutonomousVmClusterResourceDetails"},
	{Name: "CloudAutonomousVmClusterSummary"},
	{Name: "CloudExadataInfrastructure"},
	{Name: "CloudExadataInfrastructureSummary"},
	{Name: "CloudExadataInfrastructureUnallocatedResources"},
	{Name: "CloudVmCluster"},
	{Name: "CloudVmClusterSummary"},
	{Name: "CrossRegionS3RestoreSourcesAccess"},
	{Name: "CustomerContact"},
	{Name: "DataCollectionOptions"},
	{Name: "DayOfWeek"},
	{Name: "DbIormConfig"},
	{Name: "DbNode"},
	{Name: "DbNodeSummary"},
	{Name: "DbServer"},
	{Name: "DbServerPatchingDetails"},
	{Name: "DbServerSummary"},
	{Name: "DbSystemShapeSummary"},
	{Name: "ExadataIormConfig"},
	{Name: "GiVersionSummary"},
	{Name: "IamRole"},
	{Name: "KmsAccess"},
	{Name: "MaintenanceWindow"},
	{Name: "ManagedS3BackupAccess"},
	{Name: "ManagedServices"},
	{Name: "Month"},
	{Name: "OciDnsForwardingConfig"},
	{Name: "OciIdentityDomain"},
	{Name: "OdbNetwork"},
	{Name: "OdbNetworkSummary"},
	{Name: "OdbPeeringConnection"},
	{Name: "OdbPeeringConnectionSummary"},
	{Name: "S3Access"},
	{Name: "ServiceNetworkEndpoint"},
	{Name: "StsAccess"},
	{Name: "SystemVersionSummary"},
	{Name: "UpdateOdbPeeringConnection"},
	{Name: "ValidationExceptionField"},
	{Name: "ZeroEtlAccess"},
}

var odbDataTypeByName = func() map[string]odbDataType {
	out := make(map[string]odbDataType, len(odbDataTypes))
	for _, dt := range odbDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
