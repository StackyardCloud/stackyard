package server

type elasticsearchserviceOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Elasticsearch Service (legacy es 2015-01-01 API) operations, from the
// AWS CLI 2.41 botocore model:
// https://docs.aws.amazon.com/opensearch-service/latest/APIReference/API_Operations.html
var elasticsearchserviceOperations = []elasticsearchserviceOperation{
	{Name: "AcceptInboundCrossClusterSearchConnection", Method: "PUT", URI: "/2015-01-01/es/ccs/inboundConnection/{ConnectionId}/accept"},
	{Name: "AddTags", Method: "POST", URI: "/2015-01-01/tags"},
	{Name: "AssociatePackage", Method: "POST", URI: "/2015-01-01/packages/associate/{PackageID}/{DomainName}"},
	{Name: "AuthorizeVpcEndpointAccess", Method: "POST", URI: "/2015-01-01/es/domain/{DomainName}/authorizeVpcEndpointAccess"},
	{Name: "CancelDomainConfigChange", Method: "POST", URI: "/2015-01-01/es/domain/{DomainName}/config/cancel"},
	{Name: "CancelElasticsearchServiceSoftwareUpdate", Method: "POST", URI: "/2015-01-01/es/serviceSoftwareUpdate/cancel"},
	{Name: "CreateElasticsearchDomain", Method: "POST", URI: "/2015-01-01/es/domain"},
	{Name: "CreateOutboundCrossClusterSearchConnection", Method: "POST", URI: "/2015-01-01/es/ccs/outboundConnection"},
	{Name: "CreatePackage", Method: "POST", URI: "/2015-01-01/packages"},
	{Name: "CreateVpcEndpoint", Method: "POST", URI: "/2015-01-01/es/vpcEndpoints"},
	{Name: "DeleteElasticsearchDomain", Method: "DELETE", URI: "/2015-01-01/es/domain/{DomainName}"},
	{Name: "DeleteElasticsearchServiceRole", Method: "DELETE", URI: "/2015-01-01/es/role"},
	{Name: "DeleteInboundCrossClusterSearchConnection", Method: "DELETE", URI: "/2015-01-01/es/ccs/inboundConnection/{ConnectionId}"},
	{Name: "DeleteOutboundCrossClusterSearchConnection", Method: "DELETE", URI: "/2015-01-01/es/ccs/outboundConnection/{ConnectionId}"},
	{Name: "DeletePackage", Method: "DELETE", URI: "/2015-01-01/packages/{PackageID}"},
	{Name: "DeleteVpcEndpoint", Method: "DELETE", URI: "/2015-01-01/es/vpcEndpoints/{VpcEndpointId}"},
	{Name: "DescribeDomainAutoTunes", Method: "GET", URI: "/2015-01-01/es/domain/{DomainName}/autoTunes"},
	{Name: "DescribeDomainChangeProgress", Method: "GET", URI: "/2015-01-01/es/domain/{DomainName}/progress"},
	{Name: "DescribeElasticsearchDomain", Method: "GET", URI: "/2015-01-01/es/domain/{DomainName}"},
	{Name: "DescribeElasticsearchDomainConfig", Method: "GET", URI: "/2015-01-01/es/domain/{DomainName}/config"},
	{Name: "DescribeElasticsearchDomains", Method: "POST", URI: "/2015-01-01/es/domain-info"},
	{Name: "DescribeElasticsearchInstanceTypeLimits", Method: "GET", URI: "/2015-01-01/es/instanceTypeLimits/{ElasticsearchVersion}/{InstanceType}"},
	{Name: "DescribeInboundCrossClusterSearchConnections", Method: "POST", URI: "/2015-01-01/es/ccs/inboundConnection/search"},
	{Name: "DescribeOutboundCrossClusterSearchConnections", Method: "POST", URI: "/2015-01-01/es/ccs/outboundConnection/search"},
	{Name: "DescribePackages", Method: "POST", URI: "/2015-01-01/packages/describe"},
	{Name: "DescribeReservedElasticsearchInstanceOfferings", Method: "GET", URI: "/2015-01-01/es/reservedInstanceOfferings"},
	{Name: "DescribeReservedElasticsearchInstances", Method: "GET", URI: "/2015-01-01/es/reservedInstances"},
	{Name: "DescribeVpcEndpoints", Method: "POST", URI: "/2015-01-01/es/vpcEndpoints/describe"},
	{Name: "DissociatePackage", Method: "POST", URI: "/2015-01-01/packages/dissociate/{PackageID}/{DomainName}"},
	{Name: "GetCompatibleElasticsearchVersions", Method: "GET", URI: "/2015-01-01/es/compatibleVersions"},
	{Name: "GetPackageVersionHistory", Method: "GET", URI: "/2015-01-01/packages/{PackageID}/history"},
	{Name: "GetUpgradeHistory", Method: "GET", URI: "/2015-01-01/es/upgradeDomain/{DomainName}/history"},
	{Name: "GetUpgradeStatus", Method: "GET", URI: "/2015-01-01/es/upgradeDomain/{DomainName}/status"},
	{Name: "ListDomainNames", Method: "GET", URI: "/2015-01-01/domain"},
	{Name: "ListDomainsForPackage", Method: "GET", URI: "/2015-01-01/packages/{PackageID}/domains"},
	{Name: "ListElasticsearchInstanceTypes", Method: "GET", URI: "/2015-01-01/es/instanceTypes/{ElasticsearchVersion}"},
	{Name: "ListElasticsearchVersions", Method: "GET", URI: "/2015-01-01/es/versions"},
	{Name: "ListPackagesForDomain", Method: "GET", URI: "/2015-01-01/domain/{DomainName}/packages"},
	{Name: "ListTags", Method: "GET", URI: "/2015-01-01/tags/"},
	{Name: "ListVpcEndpointAccess", Method: "GET", URI: "/2015-01-01/es/domain/{DomainName}/listVpcEndpointAccess"},
	{Name: "ListVpcEndpoints", Method: "GET", URI: "/2015-01-01/es/vpcEndpoints"},
	{Name: "ListVpcEndpointsForDomain", Method: "GET", URI: "/2015-01-01/es/domain/{DomainName}/vpcEndpoints"},
	{Name: "PurchaseReservedElasticsearchInstanceOffering", Method: "POST", URI: "/2015-01-01/es/purchaseReservedInstanceOffering"},
	{Name: "RejectInboundCrossClusterSearchConnection", Method: "PUT", URI: "/2015-01-01/es/ccs/inboundConnection/{ConnectionId}/reject"},
	{Name: "RemoveTags", Method: "POST", URI: "/2015-01-01/tags-removal"},
	{Name: "RevokeVpcEndpointAccess", Method: "POST", URI: "/2015-01-01/es/domain/{DomainName}/revokeVpcEndpointAccess"},
	{Name: "StartElasticsearchServiceSoftwareUpdate", Method: "POST", URI: "/2015-01-01/es/serviceSoftwareUpdate/start"},
	{Name: "UpdateClone", Method: "POST", URI: "/2015-01-01/es/domain/{DomainName}/clone"},
	{Name: "UpdateElasticsearchDomainConfig", Method: "POST", URI: "/2015-01-01/es/domain/{DomainName}/config"},
	{Name: "UpdatePackage", Method: "POST", URI: "/2015-01-01/packages/update"},
	{Name: "UpdateVpcEndpoint", Method: "POST", URI: "/2015-01-01/es/vpcEndpoints/update"},
	{Name: "UpgradeElasticsearchDomain", Method: "POST", URI: "/2015-01-01/es/upgradeDomain"},
}

var elasticsearchserviceOperationByName = func() map[string]elasticsearchserviceOperation {
	out := make(map[string]elasticsearchserviceOperation, len(elasticsearchserviceOperations))
	for _, op := range elasticsearchserviceOperations {
		out[op.Name] = op
	}
	return out
}()
