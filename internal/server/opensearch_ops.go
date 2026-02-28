package server

type opensearchOperation struct {
	Name   string
	Method string
	URI    string
}

// OpenSearch Service operations sourced from:
// https://docs.aws.amazon.com/opensearch-service/latest/APIReference/API_Operations.html
// (OpenSearch Service section only; OpenSearch Ingestion Service excluded).
var opensearchOperations = []opensearchOperation{
	{Name: "AcceptInboundConnection", Method: "PUT", URI: "/2021-01-01/opensearch/cc/inboundConnection/{ConnectionId}/accept"},
	{Name: "AddDataSource", Method: "POST", URI: "/2021-01-01/opensearch/domain/{DomainName}/dataSource"},
	{Name: "AddDirectQueryDataSource", Method: "POST", URI: "/2021-01-01/opensearch/directQueryDataSource"},
	{Name: "AddTags", Method: "POST", URI: "/2021-01-01/tags"},
	{Name: "AssociatePackage", Method: "POST", URI: "/2021-01-01/packages/associate/{PackageID}/{DomainName}"},
	{Name: "AssociatePackages", Method: "POST", URI: "/2021-01-01/packages/associateMultiple"},
	{Name: "AuthorizeVpcEndpointAccess", Method: "POST", URI: "/2021-01-01/opensearch/domain/{DomainName}/authorizeVpcEndpointAccess"},
	{Name: "CancelDomainConfigChange", Method: "POST", URI: "/2021-01-01/opensearch/domain/{DomainName}/config/cancel"},
	{Name: "CancelServiceSoftwareUpdate", Method: "POST", URI: "/2021-01-01/opensearch/serviceSoftwareUpdate/cancel"},
	{Name: "CreateApplication", Method: "POST", URI: "/2021-01-01/opensearch/application"},
	{Name: "CreateDomain", Method: "POST", URI: "/2021-01-01/opensearch/domain"},
	{Name: "CreateIndex", Method: "POST", URI: "/2021-01-01/opensearch/domain/{DomainName}/index"},
	{Name: "CreateOutboundConnection", Method: "POST", URI: "/2021-01-01/opensearch/cc/outboundConnection"},
	{Name: "CreatePackage", Method: "POST", URI: "/2021-01-01/packages"},
	{Name: "CreateVpcEndpoint", Method: "POST", URI: "/2021-01-01/opensearch/vpcEndpoints"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/2021-01-01/opensearch/application/{id}"},
	{Name: "DeleteDataSource", Method: "DELETE", URI: "/2021-01-01/opensearch/domain/{DomainName}/dataSource/{DataSourceName}"},
	{Name: "DeleteDirectQueryDataSource", Method: "DELETE", URI: "/2021-01-01/opensearch/directQueryDataSource/{DataSourceName}"},
	{Name: "DeleteDomain", Method: "DELETE", URI: "/2021-01-01/opensearch/domain/{DomainName}"},
	{Name: "DeleteInboundConnection", Method: "DELETE", URI: "/2021-01-01/opensearch/cc/inboundConnection/{ConnectionId}"},
	{Name: "DeleteIndex", Method: "DELETE", URI: "/2021-01-01/opensearch/domain/{DomainName}/index/{IndexName}"},
	{Name: "DeleteOutboundConnection", Method: "DELETE", URI: "/2021-01-01/opensearch/cc/outboundConnection/{ConnectionId}"},
	{Name: "DeletePackage", Method: "DELETE", URI: "/2021-01-01/packages/{PackageID}"},
	{Name: "DeleteVpcEndpoint", Method: "DELETE", URI: "/2021-01-01/opensearch/vpcEndpoints/{VpcEndpointId}"},
	{Name: "DescribeDomain", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}"},
	{Name: "DescribeDomainAutoTunes", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/autoTunes"},
	{Name: "DescribeDomainChangeProgress", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/progress"},
	{Name: "DescribeDomainConfig", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/config"},
	{Name: "DescribeDomainHealth", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/health"},
	{Name: "DescribeDomainNodes", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/nodes"},
	{Name: "DescribeDomains", Method: "POST", URI: "/2021-01-01/opensearch/domain-info"},
	{Name: "DescribeDryRunProgress", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/dryRun"},
	{Name: "DescribeInboundConnections", Method: "POST", URI: "/2021-01-01/opensearch/cc/inboundConnection/search"},
	{Name: "DescribeInstanceTypeLimits", Method: "GET", URI: "/2021-01-01/opensearch/instanceTypeLimits/{EngineVersion}/{InstanceType}"},
	{Name: "DescribeOutboundConnections", Method: "POST", URI: "/2021-01-01/opensearch/cc/outboundConnection/search"},
	{Name: "DescribePackages", Method: "POST", URI: "/2021-01-01/packages/describe"},
	{Name: "DescribeReservedInstanceOfferings", Method: "GET", URI: "/2021-01-01/opensearch/reservedInstanceOfferings"},
	{Name: "DescribeReservedInstances", Method: "GET", URI: "/2021-01-01/opensearch/reservedInstances"},
	{Name: "DescribeVpcEndpoints", Method: "POST", URI: "/2021-01-01/opensearch/vpcEndpoints/describe"},
	{Name: "DissociatePackage", Method: "POST", URI: "/2021-01-01/packages/dissociate/{PackageID}/{DomainName}"},
	{Name: "DissociatePackages", Method: "POST", URI: "/2021-01-01/packages/dissociateMultiple"},
	{Name: "GetApplication", Method: "GET", URI: "/2021-01-01/opensearch/application/{id}"},
	{Name: "GetCompatibleVersions", Method: "GET", URI: "/2021-01-01/opensearch/compatibleVersions"},
	{Name: "GetDataSource", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/dataSource/{DataSourceName}"},
	{Name: "GetDefaultApplicationSetting", Method: "GET", URI: "/2021-01-01/opensearch/defaultApplicationSetting"},
	{Name: "GetDirectQueryDataSource", Method: "GET", URI: "/2021-01-01/opensearch/directQueryDataSource/{DataSourceName}"},
	{Name: "GetDomainMaintenanceStatus", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/domainMaintenance"},
	{Name: "GetIndex", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/index/{IndexName}"},
	{Name: "GetPackageVersionHistory", Method: "GET", URI: "/2021-01-01/packages/{PackageID}/history"},
	{Name: "GetUpgradeHistory", Method: "GET", URI: "/2021-01-01/opensearch/upgradeDomain/{DomainName}/history"},
	{Name: "GetUpgradeStatus", Method: "GET", URI: "/2021-01-01/opensearch/upgradeDomain/{DomainName}/status"},
	{Name: "ListApplications", Method: "GET", URI: "/2021-01-01/opensearch/list-applications"},
	{Name: "ListDataSources", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/dataSource"},
	{Name: "ListDirectQueryDataSources", Method: "GET", URI: "/2021-01-01/opensearch/directQueryDataSource"},
	{Name: "ListDomainMaintenances", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/domainMaintenances"},
	{Name: "ListDomainNames", Method: "GET", URI: "/2021-01-01/domain"},
	{Name: "ListDomainsForPackage", Method: "GET", URI: "/2021-01-01/packages/{PackageID}/domains"},
	{Name: "ListInstanceTypeDetails", Method: "GET", URI: "/2021-01-01/opensearch/instanceTypeDetails/{EngineVersion}"},
	{Name: "ListPackagesForDomain", Method: "GET", URI: "/2021-01-01/domain/{DomainName}/packages"},
	{Name: "ListScheduledActions", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/scheduledActions"},
	{Name: "ListTags", Method: "GET", URI: "/2021-01-01/tags/"},
	{Name: "ListVersions", Method: "GET", URI: "/2021-01-01/opensearch/versions"},
	{Name: "ListVpcEndpointAccess", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/listVpcEndpointAccess"},
	{Name: "ListVpcEndpoints", Method: "GET", URI: "/2021-01-01/opensearch/vpcEndpoints"},
	{Name: "ListVpcEndpointsForDomain", Method: "GET", URI: "/2021-01-01/opensearch/domain/{DomainName}/vpcEndpoints"},
	{Name: "PurchaseReservedInstanceOffering", Method: "POST", URI: "/2021-01-01/opensearch/purchaseReservedInstanceOffering"},
	{Name: "PutDefaultApplicationSetting", Method: "PUT", URI: "/2021-01-01/opensearch/defaultApplicationSetting"},
	{Name: "RejectInboundConnection", Method: "PUT", URI: "/2021-01-01/opensearch/cc/inboundConnection/{ConnectionId}/reject"},
	{Name: "RemoveTags", Method: "POST", URI: "/2021-01-01/tags-removal"},
	{Name: "RevokeVpcEndpointAccess", Method: "POST", URI: "/2021-01-01/opensearch/domain/{DomainName}/revokeVpcEndpointAccess"},
	{Name: "StartDomainMaintenance", Method: "POST", URI: "/2021-01-01/opensearch/domain/{DomainName}/domainMaintenance"},
	{Name: "StartServiceSoftwareUpdate", Method: "POST", URI: "/2021-01-01/opensearch/serviceSoftwareUpdate/start"},
	{Name: "UpdateApplication", Method: "PUT", URI: "/2021-01-01/opensearch/application/{id}"},
	{Name: "UpdateDataSource", Method: "PUT", URI: "/2021-01-01/opensearch/domain/{DomainName}/dataSource/{DataSourceName}"},
	{Name: "UpdateDirectQueryDataSource", Method: "PUT", URI: "/2021-01-01/opensearch/directQueryDataSource/{DataSourceName}"},
	{Name: "UpdateDomainConfig", Method: "POST", URI: "/2021-01-01/opensearch/domain/{DomainName}/config"},
	{Name: "UpdateIndex", Method: "PUT", URI: "/2021-01-01/opensearch/domain/{DomainName}/index/{IndexName}"},
	{Name: "UpdatePackage", Method: "POST", URI: "/2021-01-01/packages/update"},
	{Name: "UpdatePackageScope", Method: "POST", URI: "/2021-01-01/packages/updateScope"},
	{Name: "UpdateScheduledAction", Method: "PUT", URI: "/2021-01-01/opensearch/domain/{DomainName}/scheduledAction/update"},
	{Name: "UpdateVpcEndpoint", Method: "POST", URI: "/2021-01-01/opensearch/vpcEndpoints/update"},
	{Name: "UpgradeDomain", Method: "POST", URI: "/2021-01-01/opensearch/upgradeDomain"},
}

var opensearchOperationByName = func() map[string]opensearchOperation {
	out := make(map[string]opensearchOperation, len(opensearchOperations))
	for _, op := range opensearchOperations {
		out[op.Name] = op
	}
	return out
}()
