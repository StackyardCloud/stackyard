package server

type route53Operation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Route 53 operations sourced from:
// https://docs.aws.amazon.com/Route53/latest/APIReference/API_Operations_Amazon_Route_53.html
var route53Operations = []route53Operation{
	{Name: "ActivateKeySigningKey", Method: "POST", URI: "/2013-04-01"},
	{Name: "AssociateVPCWithHostedZone", Method: "POST", URI: "/2013-04-01"},
	{Name: "ChangeCidrCollection", Method: "POST", URI: "/2013-04-01"},
	{Name: "ChangeResourceRecordSets", Method: "POST", URI: "/2013-04-01"},
	{Name: "ChangeTagsForResource", Method: "POST", URI: "/2013-04-01"},
	{Name: "CreateCidrCollection", Method: "POST", URI: "/2013-04-01/cidrcollection"},
	{Name: "CreateHealthCheck", Method: "POST", URI: "/2013-04-01/healthcheck"},
	{Name: "CreateHostedZone", Method: "POST", URI: "/2013-04-01/hostedzone"},
	{Name: "CreateKeySigningKey", Method: "POST", URI: "/2013-04-01/keysigningkey"},
	{Name: "CreateQueryLoggingConfig", Method: "POST", URI: "/2013-04-01/queryloggingconfig"},
	{Name: "CreateReusableDelegationSet", Method: "POST", URI: "/2013-04-01/delegationset"},
	{Name: "CreateTrafficPolicy", Method: "POST", URI: "/2013-04-01/trafficpolicy"},
	{Name: "CreateTrafficPolicyInstance", Method: "POST", URI: "/2013-04-01/trafficpolicyinstance"},
	{Name: "CreateTrafficPolicyVersion", Method: "POST", URI: "/2013-04-01"},
	{Name: "CreateVPCAssociationAuthorization", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeactivateKeySigningKey", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteCidrCollection", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteHealthCheck", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteHostedZone", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteKeySigningKey", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteQueryLoggingConfig", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteReusableDelegationSet", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteTrafficPolicy", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteTrafficPolicyInstance", Method: "POST", URI: "/2013-04-01"},
	{Name: "DeleteVPCAssociationAuthorization", Method: "POST", URI: "/2013-04-01"},
	{Name: "DisableHostedZoneDNSSEC", Method: "POST", URI: "/2013-04-01"},
	{Name: "DisassociateVPCFromHostedZone", Method: "POST", URI: "/2013-04-01"},
	{Name: "EnableHostedZoneDNSSEC", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetAccountLimit", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetChange", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetCheckerIpRanges", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetDNSSEC", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetGeoLocation", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetHealthCheck", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetHealthCheckCount", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetHealthCheckLastFailureReason", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetHealthCheckStatus", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetHostedZone", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetHostedZoneCount", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetHostedZoneLimit", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetQueryLoggingConfig", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetReusableDelegationSet", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetReusableDelegationSetLimit", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetTrafficPolicy", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetTrafficPolicyInstance", Method: "POST", URI: "/2013-04-01"},
	{Name: "GetTrafficPolicyInstanceCount", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListCidrBlocks", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListCidrCollections", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListCidrLocations", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListGeoLocations", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListHealthChecks", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListHostedZones", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListHostedZonesByName", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListHostedZonesByVPC", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListQueryLoggingConfigs", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListResourceRecordSets", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListReusableDelegationSets", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListTagsForResources", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListTrafficPolicies", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListTrafficPolicyInstances", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListTrafficPolicyInstancesByHostedZone", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListTrafficPolicyInstancesByPolicy", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListTrafficPolicyVersions", Method: "POST", URI: "/2013-04-01"},
	{Name: "ListVPCAssociationAuthorizations", Method: "POST", URI: "/2013-04-01"},
	{Name: "TestDNSAnswer", Method: "POST", URI: "/2013-04-01"},
	{Name: "UpdateHealthCheck", Method: "POST", URI: "/2013-04-01/healthcheck"},
	{Name: "UpdateHostedZoneComment", Method: "POST", URI: "/2013-04-01"},
	{Name: "UpdateHostedZoneFeatures", Method: "POST", URI: "/2013-04-01"},
	{Name: "UpdateTrafficPolicyComment", Method: "POST", URI: "/2013-04-01"},
	{Name: "UpdateTrafficPolicyInstance", Method: "POST", URI: "/2013-04-01"},
}

var route53OperationByName = func() map[string]route53Operation {
	out := make(map[string]route53Operation, len(route53Operations))
	for _, op := range route53Operations {
		out[op.Name] = op
	}
	return out
}()
