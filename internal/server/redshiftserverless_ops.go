package server

type redshiftServerlessOperation struct {
	Name string
}

// Amazon Redshift Serverless actions sourced from:
// https://docs.aws.amazon.com/redshift-serverless/latest/APIReference/API_Operations.html
var redshiftServerlessOperations = []redshiftServerlessOperation{
	{Name: "ConvertRecoveryPointToSnapshot"},
	{Name: "CreateCustomDomainAssociation"},
	{Name: "CreateEndpointAccess"},
	{Name: "CreateNamespace"},
	{Name: "CreateReservation"},
	{Name: "CreateScheduledAction"},
	{Name: "CreateSnapshot"},
	{Name: "CreateSnapshotCopyConfiguration"},
	{Name: "CreateUsageLimit"},
	{Name: "CreateWorkgroup"},
	{Name: "DeleteCustomDomainAssociation"},
	{Name: "DeleteEndpointAccess"},
	{Name: "DeleteNamespace"},
	{Name: "DeleteResourcePolicy"},
	{Name: "DeleteScheduledAction"},
	{Name: "DeleteSnapshot"},
	{Name: "DeleteSnapshotCopyConfiguration"},
	{Name: "DeleteUsageLimit"},
	{Name: "DeleteWorkgroup"},
	{Name: "GetCredentials"},
	{Name: "GetCustomDomainAssociation"},
	{Name: "GetEndpointAccess"},
	{Name: "GetIdentityCenterAuthToken"},
	{Name: "GetNamespace"},
	{Name: "GetRecoveryPoint"},
	{Name: "GetReservation"},
	{Name: "GetReservationOffering"},
	{Name: "GetResourcePolicy"},
	{Name: "GetScheduledAction"},
	{Name: "GetSnapshot"},
	{Name: "GetTableRestoreStatus"},
	{Name: "GetTrack"},
	{Name: "GetUsageLimit"},
	{Name: "GetWorkgroup"},
	{Name: "ListCustomDomainAssociations"},
	{Name: "ListEndpointAccess"},
	{Name: "ListManagedWorkgroups"},
	{Name: "ListNamespaces"},
	{Name: "ListRecoveryPoints"},
	{Name: "ListReservationOfferings"},
	{Name: "ListReservations"},
	{Name: "ListScheduledActions"},
	{Name: "ListSnapshotCopyConfigurations"},
	{Name: "ListSnapshots"},
	{Name: "ListTableRestoreStatus"},
	{Name: "ListTagsForResource"},
	{Name: "ListTracks"},
	{Name: "ListUsageLimits"},
	{Name: "ListWorkgroups"},
	{Name: "PutResourcePolicy"},
	{Name: "RestoreFromRecoveryPoint"},
	{Name: "RestoreFromSnapshot"},
	{Name: "RestoreTableFromRecoveryPoint"},
	{Name: "RestoreTableFromSnapshot"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateCustomDomainAssociation"},
	{Name: "UpdateEndpointAccess"},
	{Name: "UpdateLakehouseConfiguration"},
	{Name: "UpdateNamespace"},
	{Name: "UpdateScheduledAction"},
	{Name: "UpdateSnapshot"},
	{Name: "UpdateSnapshotCopyConfiguration"},
	{Name: "UpdateUsageLimit"},
	{Name: "UpdateWorkgroup"},
}

var redshiftServerlessOperationByName = func() map[string]redshiftServerlessOperation {
	out := make(map[string]redshiftServerlessOperation, len(redshiftServerlessOperations))
	for _, op := range redshiftServerlessOperations {
		out[op.Name] = op
	}
	return out
}()
