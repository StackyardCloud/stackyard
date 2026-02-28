package server

type vpcLatticeOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS VPC Lattice operations sourced from:
// https://docs.aws.amazon.com/vpc-lattice/latest/APIReference/API_Operations.html
var vpcLatticeOperations = []vpcLatticeOperation{
	{Name: "BatchUpdateRule", Method: "PATCH", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}/rules"},
	{Name: "CreateAccessLogSubscription", Method: "POST", URI: "/accesslogsubscriptions"},
	{Name: "CreateListener", Method: "POST", URI: "/services/{serviceIdentifier}/listeners"},
	{Name: "CreateResourceConfiguration", Method: "POST", URI: "/resourceconfigurations"},
	{Name: "CreateResourceGateway", Method: "POST", URI: "/resourcegateways"},
	{Name: "CreateRule", Method: "POST", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}/rules"},
	{Name: "CreateService", Method: "POST", URI: "/services"},
	{Name: "CreateServiceNetwork", Method: "POST", URI: "/servicenetworks"},
	{Name: "CreateServiceNetworkResourceAssociation", Method: "POST", URI: "/servicenetworkresourceassociations"},
	{Name: "CreateServiceNetworkServiceAssociation", Method: "POST", URI: "/servicenetworkserviceassociations"},
	{Name: "CreateServiceNetworkVpcAssociation", Method: "POST", URI: "/servicenetworkvpcassociations"},
	{Name: "CreateTargetGroup", Method: "POST", URI: "/targetgroups"},
	{Name: "DeleteAccessLogSubscription", Method: "DELETE", URI: "/accesslogsubscriptions/{accessLogSubscriptionIdentifier}"},
	{Name: "DeleteAuthPolicy", Method: "DELETE", URI: "/authpolicy/{resourceIdentifier}"},
	{Name: "DeleteDomainVerification", Method: "DELETE", URI: "/domainverifications/{domainVerificationIdentifier}"},
	{Name: "DeleteListener", Method: "DELETE", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}"},
	{Name: "DeleteResourceConfiguration", Method: "DELETE", URI: "/resourceconfigurations/{resourceConfigurationIdentifier}"},
	{Name: "DeleteResourceEndpointAssociation", Method: "DELETE", URI: "/resourceendpointassociations/{resourceEndpointAssociationIdentifier}"},
	{Name: "DeleteResourceGateway", Method: "DELETE", URI: "/resourcegateways/{resourceGatewayIdentifier}"},
	{Name: "DeleteResourcePolicy", Method: "DELETE", URI: "/resourcepolicy/{resourceArn}"},
	{Name: "DeleteRule", Method: "DELETE", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}/rules/{ruleIdentifier}"},
	{Name: "DeleteService", Method: "DELETE", URI: "/services/{serviceIdentifier}"},
	{Name: "DeleteServiceNetwork", Method: "DELETE", URI: "/servicenetworks/{serviceNetworkIdentifier}"},
	{Name: "DeleteServiceNetworkResourceAssociation", Method: "DELETE", URI: "/servicenetworkresourceassociations/{serviceNetworkResourceAssociationIdentifier}"},
	{Name: "DeleteServiceNetworkServiceAssociation", Method: "DELETE", URI: "/servicenetworkserviceassociations/{serviceNetworkServiceAssociationIdentifier}"},
	{Name: "DeleteServiceNetworkVpcAssociation", Method: "DELETE", URI: "/servicenetworkvpcassociations/{serviceNetworkVpcAssociationIdentifier}"},
	{Name: "DeleteTargetGroup", Method: "DELETE", URI: "/targetgroups/{targetGroupIdentifier}"},
	{Name: "DeregisterTargets", Method: "POST", URI: "/targetgroups/{targetGroupIdentifier}/deregistertargets"},
	{Name: "GetAccessLogSubscription", Method: "GET", URI: "/accesslogsubscriptions/{accessLogSubscriptionIdentifier}"},
	{Name: "GetAuthPolicy", Method: "GET", URI: "/authpolicy/{resourceIdentifier}"},
	{Name: "GetDomainVerification", Method: "GET", URI: "/domainverifications/{domainVerificationIdentifier}"},
	{Name: "GetListener", Method: "GET", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}"},
	{Name: "GetResourceConfiguration", Method: "GET", URI: "/resourceconfigurations/{resourceConfigurationIdentifier}"},
	{Name: "GetResourceGateway", Method: "GET", URI: "/resourcegateways/{resourceGatewayIdentifier}"},
	{Name: "GetResourcePolicy", Method: "GET", URI: "/resourcepolicy/{resourceArn}"},
	{Name: "GetRule", Method: "GET", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}/rules/{ruleIdentifier}"},
	{Name: "GetService", Method: "GET", URI: "/services/{serviceIdentifier}"},
	{Name: "GetServiceNetwork", Method: "GET", URI: "/servicenetworks/{serviceNetworkIdentifier}"},
	{Name: "GetServiceNetworkResourceAssociation", Method: "GET", URI: "/servicenetworkresourceassociations/{serviceNetworkResourceAssociationIdentifier}"},
	{Name: "GetServiceNetworkServiceAssociation", Method: "GET", URI: "/servicenetworkserviceassociations/{serviceNetworkServiceAssociationIdentifier}"},
	{Name: "GetServiceNetworkVpcAssociation", Method: "GET", URI: "/servicenetworkvpcassociations/{serviceNetworkVpcAssociationIdentifier}"},
	{Name: "GetTargetGroup", Method: "GET", URI: "/targetgroups/{targetGroupIdentifier}"},
	{Name: "ListAccessLogSubscriptions", Method: "GET", URI: "/accesslogsubscriptions?maxResults={maxResults}&nextToken={nextToken}&resourceIdentifier={resourceIdentifier}"},
	{Name: "ListDomainVerifications", Method: "GET", URI: "/domainverifications?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListListeners", Method: "GET", URI: "/services/{serviceIdentifier}/listeners?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListResourceConfigurations", Method: "GET", URI: "/resourceconfigurations?domainVerificationIdentifier={domainVerificationIdentifier}&maxResults={maxResults}&nextToken={nextToken}&resourceConfigurationGroupIdentifier={resourceConfigurationGroupIdentifier}&resourceGatewayIdentifier={resourceGatewayIdentifier}"},
	{Name: "ListResourceEndpointAssociations", Method: "GET", URI: "/resourceendpointassociations?maxResults={maxResults}&nextToken={nextToken}&resourceConfigurationIdentifier={resourceConfigurationIdentifier}&resourceEndpointAssociationIdentifier={resourceEndpointAssociationIdentifier}&vpcEndpointId={vpcEndpointId}&vpcEndpointOwner={vpcEndpointOwner}"},
	{Name: "ListResourceGateways", Method: "GET", URI: "/resourcegateways?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListRules", Method: "GET", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}/rules?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListServiceNetworkResourceAssociations", Method: "GET", URI: "/servicenetworkresourceassociations?includeChildren={includeChildren}&maxResults={maxResults}&nextToken={nextToken}&resourceConfigurationIdentifier={resourceConfigurationIdentifier}&serviceNetworkIdentifier={serviceNetworkIdentifier}"},
	{Name: "ListServiceNetworks", Method: "GET", URI: "/servicenetworks?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListServiceNetworkServiceAssociations", Method: "GET", URI: "/servicenetworkserviceassociations?maxResults={maxResults}&nextToken={nextToken}&serviceIdentifier={serviceIdentifier}&serviceNetworkIdentifier={serviceNetworkIdentifier}"},
	{Name: "ListServiceNetworkVpcAssociations", Method: "GET", URI: "/servicenetworkvpcassociations?maxResults={maxResults}&nextToken={nextToken}&serviceNetworkIdentifier={serviceNetworkIdentifier}&vpcIdentifier={vpcIdentifier}"},
	{Name: "ListServiceNetworkVpcEndpointAssociations", Method: "GET", URI: "/servicenetworkvpcendpointassociations?maxResults={maxResults}&nextToken={nextToken}&serviceNetworkIdentifier={serviceNetworkIdentifier}"},
	{Name: "ListServices", Method: "GET", URI: "/services?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListTargetGroups", Method: "GET", URI: "/targetgroups?maxResults={maxResults}&nextToken={nextToken}&targetGroupType={targetGroupType}&vpcIdentifier={vpcIdentifier}"},
	{Name: "ListTargets", Method: "POST", URI: "/targetgroups/{targetGroupIdentifier}/listtargets?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "PutAuthPolicy", Method: "PUT", URI: "/authpolicy/{resourceIdentifier}"},
	{Name: "PutResourcePolicy", Method: "PUT", URI: "/resourcepolicy/{resourceArn}"},
	{Name: "RegisterTargets", Method: "POST", URI: "/targetgroups/{targetGroupIdentifier}/registertargets"},
	{Name: "StartDomainVerification", Method: "POST", URI: "/domainverifications"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateAccessLogSubscription", Method: "PATCH", URI: "/accesslogsubscriptions/{accessLogSubscriptionIdentifier}"},
	{Name: "UpdateListener", Method: "PATCH", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}"},
	{Name: "UpdateResourceConfiguration", Method: "PATCH", URI: "/resourceconfigurations/{resourceConfigurationIdentifier}"},
	{Name: "UpdateResourceGateway", Method: "PATCH", URI: "/resourcegateways/{resourceGatewayIdentifier}"},
	{Name: "UpdateRule", Method: "PATCH", URI: "/services/{serviceIdentifier}/listeners/{listenerIdentifier}/rules/{ruleIdentifier}"},
	{Name: "UpdateService", Method: "PATCH", URI: "/services/{serviceIdentifier}"},
	{Name: "UpdateServiceNetwork", Method: "PATCH", URI: "/servicenetworks/{serviceNetworkIdentifier}"},
	{Name: "UpdateServiceNetworkVpcAssociation", Method: "PATCH", URI: "/servicenetworkvpcassociations/{serviceNetworkVpcAssociationIdentifier}"},
	{Name: "UpdateTargetGroup", Method: "PATCH", URI: "/targetgroups/{targetGroupIdentifier}"},
}

var vpcLatticeOperationByName = func() map[string]vpcLatticeOperation {
	out := make(map[string]vpcLatticeOperation, len(vpcLatticeOperations))
	for _, op := range vpcLatticeOperations {
		out[op.Name] = op
	}
	return out
}()
