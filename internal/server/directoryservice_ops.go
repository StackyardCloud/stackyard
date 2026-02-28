package server

type directoryServiceOperation struct {
	Name string
}

// AWS Directory Service operations sourced from:
// https://docs.aws.amazon.com/directoryservice/latest/devguide/API_Operations.html
var directoryServiceOperations = []directoryServiceOperation{
	{Name: "AcceptSharedDirectory"},
	{Name: "AddIpRoutes"},
	{Name: "AddRegion"},
	{Name: "AddTagsToResource"},
	{Name: "CancelSchemaExtension"},
	{Name: "ConnectDirectory"},
	{Name: "CreateAlias"},
	{Name: "CreateComputer"},
	{Name: "CreateConditionalForwarder"},
	{Name: "CreateDirectory"},
	{Name: "CreateLogSubscription"},
	{Name: "CreateMicrosoftAD"},
	{Name: "CreateSnapshot"},
	{Name: "CreateTrust"},
	{Name: "DeleteConditionalForwarder"},
	{Name: "DeleteDirectory"},
	{Name: "DeleteLogSubscription"},
	{Name: "DeleteSnapshot"},
	{Name: "DeleteTrust"},
	{Name: "DeregisterCertificate"},
	{Name: "DeregisterEventTopic"},
	{Name: "DescribeCertificate"},
	{Name: "DescribeClientAuthenticationSettings"},
	{Name: "DescribeConditionalForwarders"},
	{Name: "DescribeDirectories"},
	{Name: "DescribeDirectoryDataAccess"},
	{Name: "DescribeDomainControllers"},
	{Name: "DescribeEventTopics"},
	{Name: "DescribeLDAPSSettings"},
	{Name: "DescribeRegions"},
	{Name: "DescribeSettings"},
	{Name: "DescribeSharedDirectories"},
	{Name: "DescribeSnapshots"},
	{Name: "DescribeTrusts"},
	{Name: "DescribeUpdateDirectory"},
	{Name: "DisableClientAuthentication"},
	{Name: "DisableDirectoryDataAccess"},
	{Name: "DisableLDAPS"},
	{Name: "DisableRadius"},
	{Name: "DisableSso"},
	{Name: "EnableClientAuthentication"},
	{Name: "EnableDirectoryDataAccess"},
	{Name: "EnableLDAPS"},
	{Name: "EnableRadius"},
	{Name: "EnableSso"},
	{Name: "GetDirectoryLimits"},
	{Name: "GetSnapshotLimits"},
	{Name: "ListCertificates"},
	{Name: "ListIpRoutes"},
	{Name: "ListLogSubscriptions"},
	{Name: "ListSchemaExtensions"},
	{Name: "ListTagsForResource"},
	{Name: "RegisterCertificate"},
	{Name: "RegisterEventTopic"},
	{Name: "RejectSharedDirectory"},
	{Name: "RemoveIpRoutes"},
	{Name: "RemoveRegion"},
	{Name: "RemoveTagsFromResource"},
	{Name: "ResetUserPassword"},
	{Name: "RestoreFromSnapshot"},
	{Name: "ShareDirectory"},
	{Name: "StartSchemaExtension"},
	{Name: "UnshareDirectory"},
	{Name: "UpdateConditionalForwarder"},
	{Name: "UpdateDirectorySetup"},
	{Name: "UpdateNumberOfDomainControllers"},
	{Name: "UpdateRadius"},
	{Name: "UpdateSettings"},
	{Name: "UpdateTrust"},
	{Name: "VerifyTrust"},
}

var directoryServiceOperationByName = func() map[string]directoryServiceOperation {
	out := make(map[string]directoryServiceOperation, len(directoryServiceOperations))
	for _, op := range directoryServiceOperations {
		out[op.Name] = op
	}
	return out
}()
