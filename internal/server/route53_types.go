package server

type route53DataType struct {
	Name string
}

// Amazon Route 53 data types sourced from:
// https://docs.aws.amazon.com/Route53/latest/APIReference/API_Types_Amazon_Route_53.html
var route53DataTypes = []route53DataType{
	{Name: "AccountLimit"},
	{Name: "AlarmIdentifier"},
	{Name: "AliasTarget"},
	{Name: "Change"},
	{Name: "ChangeBatch"},
	{Name: "ChangeInfo"},
	{Name: "CidrBlockSummary"},
	{Name: "CidrCollection"},
	{Name: "CidrCollectionChange"},
	{Name: "CidrRoutingConfig"},
	{Name: "CloudWatchAlarmConfiguration"},
	{Name: "CollectionSummary"},
	{Name: "Coordinates"},
	{Name: "DNSSECStatus"},
	{Name: "DelegationSet"},
	{Name: "Dimension"},
	{Name: "GeoLocation"},
	{Name: "GeoLocationDetails"},
	{Name: "GeoProximityLocation"},
	{Name: "HealthCheck"},
	{Name: "HealthCheckConfig"},
	{Name: "HealthCheckObservation"},
	{Name: "HostedZone"},
	{Name: "HostedZoneConfig"},
	{Name: "HostedZoneFailureReasons"},
	{Name: "HostedZoneFeatures"},
	{Name: "HostedZoneLimit"},
	{Name: "HostedZoneOwner"},
	{Name: "HostedZoneSummary"},
	{Name: "KeySigningKey"},
	{Name: "LinkedService"},
	{Name: "LocationSummary"},
	{Name: "QueryLoggingConfig"},
	{Name: "ResourceRecord"},
	{Name: "ResourceRecordSet"},
	{Name: "ResourceTagSet"},
	{Name: "ReusableDelegationSetLimit"},
	{Name: "StatusReport"},
	{Name: "Tag"},
	{Name: "TrafficPolicy"},
	{Name: "TrafficPolicyInstance"},
	{Name: "TrafficPolicySummary"},
	{Name: "VPC"},
}

var route53DataTypeByName = func() map[string]route53DataType {
	out := make(map[string]route53DataType, len(route53DataTypes))
	for _, typ := range route53DataTypes {
		out[typ.Name] = typ
	}
	return out
}()
