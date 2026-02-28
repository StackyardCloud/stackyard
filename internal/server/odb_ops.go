package server

type odbOperation struct {
	Name   string
	Method string
	URI    string
}

// Oracle Database@AWS actions sourced from:
// https://docs.aws.amazon.com/odb/latest/APIReference/API_Operations.html
var odbOperations = []odbOperation{
	{Name: "AcceptMarketplaceRegistration", Method: "POST", URI: "/AcceptMarketplaceRegistration"},
	{Name: "AssociateIamRoleToResource", Method: "POST", URI: "/AssociateIamRoleToResource"},
	{Name: "CreateCloudAutonomousVmCluster", Method: "POST", URI: "/CreateCloudAutonomousVmCluster"},
	{Name: "CreateCloudExadataInfrastructure", Method: "POST", URI: "/CreateCloudExadataInfrastructure"},
	{Name: "CreateCloudVmCluster", Method: "POST", URI: "/CreateCloudVmCluster"},
	{Name: "CreateOdbNetwork", Method: "POST", URI: "/CreateOdbNetwork"},
	{Name: "CreateOdbPeeringConnection", Method: "POST", URI: "/CreateOdbPeeringConnection"},
	{Name: "DeleteCloudAutonomousVmCluster", Method: "POST", URI: "/DeleteCloudAutonomousVmCluster"},
	{Name: "DeleteCloudExadataInfrastructure", Method: "POST", URI: "/DeleteCloudExadataInfrastructure"},
	{Name: "DeleteCloudVmCluster", Method: "POST", URI: "/DeleteCloudVmCluster"},
	{Name: "DeleteOdbNetwork", Method: "POST", URI: "/DeleteOdbNetwork"},
	{Name: "DeleteOdbPeeringConnection", Method: "POST", URI: "/DeleteOdbPeeringConnection"},
	{Name: "DisassociateIamRoleFromResource", Method: "POST", URI: "/DisassociateIamRoleFromResource"},
	{Name: "GetCloudAutonomousVmCluster", Method: "POST", URI: "/GetCloudAutonomousVmCluster"},
	{Name: "GetCloudExadataInfrastructure", Method: "POST", URI: "/GetCloudExadataInfrastructure"},
	{Name: "GetCloudExadataInfrastructureUnallocatedResources", Method: "POST", URI: "/GetCloudExadataInfrastructureUnallocatedResources"},
	{Name: "GetCloudVmCluster", Method: "POST", URI: "/GetCloudVmCluster"},
	{Name: "GetDbNode", Method: "POST", URI: "/GetDbNode"},
	{Name: "GetDbServer", Method: "POST", URI: "/GetDbServer"},
	{Name: "GetOciOnboardingStatus", Method: "POST", URI: "/GetOciOnboardingStatus"},
	{Name: "GetOdbNetwork", Method: "POST", URI: "/GetOdbNetwork"},
	{Name: "GetOdbPeeringConnection", Method: "POST", URI: "/GetOdbPeeringConnection"},
	{Name: "InitializeService", Method: "POST", URI: "/InitializeService"},
	{Name: "ListAutonomousVirtualMachines", Method: "POST", URI: "/ListAutonomousVirtualMachines"},
	{Name: "ListCloudAutonomousVmClusters", Method: "POST", URI: "/ListCloudAutonomousVmClusters"},
	{Name: "ListCloudExadataInfrastructures", Method: "POST", URI: "/ListCloudExadataInfrastructures"},
	{Name: "ListCloudVmClusters", Method: "POST", URI: "/ListCloudVmClusters"},
	{Name: "ListDbNodes", Method: "POST", URI: "/ListDbNodes"},
	{Name: "ListDbServers", Method: "POST", URI: "/ListDbServers"},
	{Name: "ListDbSystemShapes", Method: "POST", URI: "/ListDbSystemShapes"},
	{Name: "ListGiVersions", Method: "POST", URI: "/ListGiVersions"},
	{Name: "ListOdbNetworks", Method: "POST", URI: "/ListOdbNetworks"},
	{Name: "ListOdbPeeringConnections", Method: "POST", URI: "/ListOdbPeeringConnections"},
	{Name: "ListSystemVersions", Method: "POST", URI: "/ListSystemVersions"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/ListTagsForResource"},
	{Name: "RebootDbNode", Method: "POST", URI: "/RebootDbNode"},
	{Name: "StartDbNode", Method: "POST", URI: "/StartDbNode"},
	{Name: "StopDbNode", Method: "POST", URI: "/StopDbNode"},
	{Name: "TagResource", Method: "POST", URI: "/TagResource"},
	{Name: "UntagResource", Method: "POST", URI: "/UntagResource"},
	{Name: "UpdateCloudExadataInfrastructure", Method: "POST", URI: "/UpdateCloudExadataInfrastructure"},
	{Name: "UpdateOdbNetwork", Method: "POST", URI: "/UpdateOdbNetwork"},
	{Name: "UpdateOdbPeeringConnection", Method: "POST", URI: "/UpdateOdbPeeringConnection"},
}

var odbOperationByName = func() map[string]odbOperation {
	out := make(map[string]odbOperation, len(odbOperations))
	for _, op := range odbOperations {
		out[op.Name] = op
	}
	return out
}()
