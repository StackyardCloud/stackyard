package server

import (
	"fmt"
	"sync"
	"time"
)

type route53Store struct {
	mu sync.Mutex

	nextID int64

	hostedZoneID            string
	healthCheckID           string
	changeID                string
	delegationSetID         string
	cidrCollectionID        string
	queryLoggingConfigID    string
	trafficPolicyID         string
	trafficPolicyInstanceID string
	keySigningKeyName       string
}

func newRoute53Store() *route53Store {
	return &route53Store{
		nextID:                  1,
		hostedZoneID:            "ZSTACKYARD01",
		healthCheckID:           "hc-stackyard-01",
		changeID:                "CSTACKYARD01",
		delegationSetID:         "NSTACKYARD01",
		cidrCollectionID:        "cc-stackyard-01",
		queryLoggingConfigID:    "qlc-stackyard-01",
		trafficPolicyID:         "tp-stackyard-01",
		trafficPolicyInstanceID: "tpi-stackyard-01",
		keySigningKeyName:       "stackyard-ksk",
	}
}

type route53Result struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

func (s *route53Store) Handle(action string, pathParams map[string]string) route53Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	hostedZoneID := route53StringOr(pathParams["hostedZoneId"], s.hostedZoneID)
	healthCheckID := route53StringOr(pathParams["healthCheckId"], s.healthCheckID)
	changeID := route53StringOr(pathParams["changeId"], s.changeID)
	delegationSetID := route53StringOr(pathParams["delegationSetId"], s.delegationSetID)

	switch action {
	case "CreateHostedZone":
		s.hostedZoneID = s.route53NextHostedZoneIDLocked()
		s.changeID = s.route53NextChangeIDLocked()
		hostedZoneID = s.hostedZoneID
		changeID = s.changeID
		return route53XMLResult(
			"CreateHostedZoneResponse",
			route53LocationHeader("/2013-04-01/hostedzone/"+hostedZoneID),
			route53HostedZoneXML(hostedZoneID)+route53ChangeInfoXML(changeID, "PENDING", now)+route53DelegationSetXML(s.delegationSetID),
		)
	case "ListHostedZones":
		return route53XMLResult(
			"ListHostedZonesResponse",
			nil,
			"<HostedZones>"+route53HostedZoneXML(hostedZoneID)+"</HostedZones><IsTruncated>false</IsTruncated><Marker></Marker><MaxItems>100</MaxItems>",
		)
	case "ListHostedZonesByName":
		return route53XMLResult(
			"ListHostedZonesByNameResponse",
			nil,
			"<HostedZones>"+route53HostedZoneXML(hostedZoneID)+"</HostedZones><DNSName>stackyard.example.com.</DNSName><HostedZoneId>/hostedzone/"+hostedZoneID+"</HostedZoneId><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems>",
		)
	case "GetHostedZone":
		return route53XMLResult(
			"GetHostedZoneResponse",
			nil,
			route53HostedZoneXML(hostedZoneID)+route53DelegationSetXML(delegationSetID),
		)
	case "UpdateHostedZoneComment":
		return route53XMLResult("UpdateHostedZoneCommentResponse", nil, route53HostedZoneXML(hostedZoneID))
	case "DeleteHostedZone":
		return route53XMLResult("DeleteHostedZoneResponse", nil, route53ChangeInfoXML(changeID, "PENDING", now))
	case "ChangeResourceRecordSets":
		s.changeID = s.route53NextChangeIDLocked()
		return route53XMLResult("ChangeResourceRecordSetsResponse", nil, route53ChangeInfoXML(s.changeID, "PENDING", now))
	case "GetChange":
		return route53XMLResult("GetChangeResponse", nil, route53ChangeInfoXML(changeID, "INSYNC", now))
	case "ListResourceRecordSets":
		return route53XMLResult(
			"ListResourceRecordSetsResponse",
			nil,
			"<ResourceRecordSets><ResourceRecordSet><Name>stackyard.example.com.</Name><Type>A</Type><TTL>60</TTL><ResourceRecords><ResourceRecord><Value>127.0.0.1</Value></ResourceRecord></ResourceRecords></ResourceRecordSet></ResourceRecordSets><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems>",
		)

	case "CreateHealthCheck":
		s.healthCheckID = fmt.Sprintf("hc-stackyard-%02d", s.nextID)
		s.nextID++
		healthCheckID = s.healthCheckID
		return route53XMLResult(
			"CreateHealthCheckResponse",
			route53LocationHeader("/2013-04-01/healthcheck/"+healthCheckID),
			route53HealthCheckXML(healthCheckID),
		)
	case "GetHealthCheck", "UpdateHealthCheck":
		return route53XMLResult(action+"Response", nil, route53HealthCheckXML(healthCheckID))
	case "DeleteHealthCheck":
		return route53XMLResult("DeleteHealthCheckResponse", nil, "")
	case "ListHealthChecks":
		return route53XMLResult(
			"ListHealthChecksResponse",
			nil,
			"<HealthChecks>"+route53HealthCheckXML(healthCheckID)+"</HealthChecks><IsTruncated>false</IsTruncated><Marker></Marker><MaxItems>100</MaxItems>",
		)
	case "GetHealthCheckStatus":
		return route53XMLResult("GetHealthCheckStatusResponse", nil, route53HealthCheckObservationsXML(now, "Success: HTTP Status Code 200, OK"))
	case "GetHealthCheckLastFailureReason":
		return route53XMLResult("GetHealthCheckLastFailureReasonResponse", nil, route53HealthCheckObservationsXML(now, "None"))
	case "GetHealthCheckCount":
		return route53XMLResult("GetHealthCheckCountResponse", nil, "<HealthCheckCount>1</HealthCheckCount>")

	case "GetHostedZoneCount":
		return route53XMLResult("GetHostedZoneCountResponse", nil, "<HostedZoneCount>1</HostedZoneCount>")
	case "GetHostedZoneLimit":
		return route53XMLResult("GetHostedZoneLimitResponse", nil, "<Count>1</Count><Limit><Type>MAX_RRSETS_BY_ZONE</Type><Value>10000</Value></Limit>")
	case "GetCheckerIpRanges":
		return route53XMLResult("GetCheckerIpRangesResponse", nil, "<CheckerIpRanges><member>198.51.100.1/32</member><member>203.0.113.1/32</member></CheckerIpRanges>")
	case "GetAccountLimit":
		return route53XMLResult("GetAccountLimitResponse", nil, "<Count>1</Count><Limit><Type>MAX_HOSTED_ZONES_BY_OWNER</Type><Value>100</Value></Limit>")
	case "GetGeoLocation":
		return route53XMLResult("GetGeoLocationResponse", nil, "<GeoLocationDetails><CountryCode>US</CountryCode><CountryName>United States</CountryName></GeoLocationDetails>")

	case "CreateCidrCollection":
		s.cidrCollectionID = fmt.Sprintf("cc-stackyard-%02d", s.nextID)
		s.nextID++
		return route53XMLResult(
			"CreateCidrCollectionResponse",
			route53LocationHeader("/2013-04-01/cidrcollection/"+s.cidrCollectionID),
			route53CidrCollectionXML(s.cidrCollectionID),
		)
	case "ChangeCidrCollection":
		return route53XMLResult("ChangeCidrCollectionResponse", nil, "<Id>"+s.cidrCollectionID+"</Id>")
	case "DeleteCidrCollection":
		return route53XMLResult("DeleteCidrCollectionResponse", nil, "")
	case "ListCidrBlocks":
		return route53XMLResult(
			"ListCidrBlocksResponse",
			nil,
			"<CidrBlocks><CidrBlockSummary><CidrBlock>10.0.0.0/24</CidrBlock><LocationName>default</LocationName></CidrBlockSummary></CidrBlocks>",
		)
	case "ListCidrCollections":
		return route53XMLResult(
			"ListCidrCollectionsResponse",
			nil,
			"<CidrCollections><CollectionSummary><Arn>arn:aws:route53:::cidrcollection/"+s.cidrCollectionID+"</Arn><Id>"+s.cidrCollectionID+"</Id><Name>stackyard-cidr-collection</Name><Version>1</Version></CollectionSummary></CidrCollections>",
		)
	case "ListCidrLocations":
		return route53XMLResult(
			"ListCidrLocationsResponse",
			nil,
			"<CidrLocations><LocationSummary><LocationName>default</LocationName></LocationSummary></CidrLocations>",
		)

	case "CreateReusableDelegationSet":
		s.delegationSetID = fmt.Sprintf("NSTACKYARD%02d", s.nextID)
		s.nextID++
		delegationSetID = s.delegationSetID
		return route53XMLResult(
			"CreateReusableDelegationSetResponse",
			route53LocationHeader("/2013-04-01/delegationset/"+delegationSetID),
			route53DelegationSetXML(delegationSetID),
		)
	case "GetReusableDelegationSet":
		return route53XMLResult("GetReusableDelegationSetResponse", nil, route53DelegationSetXML(delegationSetID))
	case "GetReusableDelegationSetLimit":
		return route53XMLResult("GetReusableDelegationSetLimitResponse", nil, "<Count>1</Count><Limit><Type>MAX_ZONES_BY_REUSABLE_DELEGATION_SET</Type><Value>100</Value></Limit>")
	case "ListReusableDelegationSets":
		return route53XMLResult(
			"ListReusableDelegationSetsResponse",
			nil,
			"<DelegationSets>"+route53DelegationSetXML(delegationSetID)+"</DelegationSets><IsTruncated>false</IsTruncated><Marker></Marker><MaxItems>100</MaxItems>",
		)
	case "DeleteReusableDelegationSet":
		return route53XMLResult("DeleteReusableDelegationSetResponse", nil, "")

	case "CreateQueryLoggingConfig":
		s.queryLoggingConfigID = fmt.Sprintf("qlc-stackyard-%02d", s.nextID)
		s.nextID++
		return route53XMLResult(
			"CreateQueryLoggingConfigResponse",
			route53LocationHeader("/2013-04-01/queryloggingconfig/"+s.queryLoggingConfigID),
			route53QueryLoggingConfigXML(s.queryLoggingConfigID, hostedZoneID),
		)
	case "GetQueryLoggingConfig":
		return route53XMLResult("GetQueryLoggingConfigResponse", nil, route53QueryLoggingConfigXML(s.queryLoggingConfigID, hostedZoneID))
	case "ListQueryLoggingConfigs":
		return route53XMLResult(
			"ListQueryLoggingConfigsResponse",
			nil,
			"<QueryLoggingConfigs>"+route53QueryLoggingConfigXML(s.queryLoggingConfigID, hostedZoneID)+"</QueryLoggingConfigs>",
		)
	case "DeleteQueryLoggingConfig":
		return route53XMLResult("DeleteQueryLoggingConfigResponse", nil, "")

	case "CreateTrafficPolicy":
		s.trafficPolicyID = fmt.Sprintf("tp-stackyard-%02d", s.nextID)
		s.nextID++
		return route53XMLResult(
			"CreateTrafficPolicyResponse",
			route53LocationHeader("/2013-04-01/trafficpolicy/"+s.trafficPolicyID),
			route53TrafficPolicyXML(s.trafficPolicyID, 1),
		)
	case "CreateTrafficPolicyVersion":
		return route53XMLResult(
			"CreateTrafficPolicyVersionResponse",
			route53LocationHeader("/2013-04-01/trafficpolicy/"+s.trafficPolicyID+"/version/2"),
			route53TrafficPolicyXML(s.trafficPolicyID, 2),
		)
	case "GetTrafficPolicy":
		return route53XMLResult("GetTrafficPolicyResponse", nil, route53TrafficPolicyXML(s.trafficPolicyID, 1))
	case "UpdateTrafficPolicyComment":
		return route53XMLResult("UpdateTrafficPolicyCommentResponse", nil, route53TrafficPolicyXML(s.trafficPolicyID, 1))
	case "ListTrafficPolicies":
		return route53XMLResult(
			"ListTrafficPoliciesResponse",
			nil,
			"<IsTruncated>false</IsTruncated><MaxItems>100</MaxItems><TrafficPolicyIdMarker>"+s.trafficPolicyID+"</TrafficPolicyIdMarker><TrafficPolicySummaries><TrafficPolicySummary><Id>"+s.trafficPolicyID+"</Id><LatestVersion>1</LatestVersion><Name>stackyard-traffic-policy</Name><TrafficPolicyCount>1</TrafficPolicyCount><Type>A</Type></TrafficPolicySummary></TrafficPolicySummaries>",
		)
	case "ListTrafficPolicyVersions":
		return route53XMLResult(
			"ListTrafficPolicyVersionsResponse",
			nil,
			"<IsTruncated>false</IsTruncated><MaxItems>100</MaxItems><TrafficPolicyVersionMarker>1</TrafficPolicyVersionMarker><TrafficPolicies>"+route53TrafficPolicyXML(s.trafficPolicyID, 1)+"</TrafficPolicies>",
		)
	case "DeleteTrafficPolicy":
		return route53XMLResult("DeleteTrafficPolicyResponse", nil, "")

	case "CreateTrafficPolicyInstance":
		s.trafficPolicyInstanceID = fmt.Sprintf("tpi-stackyard-%02d", s.nextID)
		s.nextID++
		return route53XMLResult(
			"CreateTrafficPolicyInstanceResponse",
			route53LocationHeader("/2013-04-01/trafficpolicyinstance/"+s.trafficPolicyInstanceID),
			route53TrafficPolicyInstanceXML(s.trafficPolicyInstanceID, hostedZoneID, s.trafficPolicyID, 1),
		)
	case "GetTrafficPolicyInstance":
		return route53XMLResult("GetTrafficPolicyInstanceResponse", nil, route53TrafficPolicyInstanceXML(s.trafficPolicyInstanceID, hostedZoneID, s.trafficPolicyID, 1))
	case "GetTrafficPolicyInstanceCount":
		return route53XMLResult("GetTrafficPolicyInstanceCountResponse", nil, "<TrafficPolicyInstanceCount>1</TrafficPolicyInstanceCount>")
	case "UpdateTrafficPolicyInstance":
		return route53XMLResult("UpdateTrafficPolicyInstanceResponse", nil, route53TrafficPolicyInstanceXML(s.trafficPolicyInstanceID, hostedZoneID, s.trafficPolicyID, 1))
	case "ListTrafficPolicyInstances", "ListTrafficPolicyInstancesByHostedZone", "ListTrafficPolicyInstancesByPolicy":
		return route53XMLResult(
			action+"Response",
			nil,
			"<IsTruncated>false</IsTruncated><MaxItems>100</MaxItems><TrafficPolicyInstances>"+route53TrafficPolicyInstanceXML(s.trafficPolicyInstanceID, hostedZoneID, s.trafficPolicyID, 1)+"</TrafficPolicyInstances>",
		)
	case "DeleteTrafficPolicyInstance":
		return route53XMLResult("DeleteTrafficPolicyInstanceResponse", nil, "")

	case "CreateKeySigningKey":
		s.keySigningKeyName = "stackyard-ksk"
		s.changeID = s.route53NextChangeIDLocked()
		return route53XMLResult(
			"CreateKeySigningKeyResponse",
			route53LocationHeader("/2013-04-01/keysigningkey/"+s.keySigningKeyName),
			route53ChangeInfoXML(s.changeID, "PENDING", now)+route53KeySigningKeyXML(hostedZoneID, s.keySigningKeyName, now),
		)
	case "ActivateKeySigningKey", "DeactivateKeySigningKey", "DeleteKeySigningKey":
		return route53XMLResult(action+"Response", nil, route53ChangeInfoXML(changeID, "PENDING", now))

	case "ListGeoLocations":
		return route53XMLResult(
			"ListGeoLocationsResponse",
			nil,
			"<GeoLocationDetailsList><GeoLocationDetails><CountryCode>US</CountryCode><CountryName>United States</CountryName></GeoLocationDetails></GeoLocationDetailsList><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems>",
		)
	case "ListTagsForResource":
		return route53XMLResult("ListTagsForResourceResponse", nil, route53ResourceTagSetXML(hostedZoneID, "hostedzone"))
	case "ListTagsForResources":
		return route53XMLResult(
			"ListTagsForResourcesResponse",
			nil,
			"<ResourceTagSets>"+route53ResourceTagSetXML(hostedZoneID, "hostedzone")+"</ResourceTagSets>",
		)
	case "ChangeTagsForResource":
		return route53XMLResult("ChangeTagsForResourceResponse", nil, "")

	case "TestDNSAnswer":
		return route53XMLResult("TestDNSAnswerResponse", nil, "<Nameserver>ns-100.awsdns-01.com</Nameserver><RecordName>stackyard.example.com</RecordName><RecordType>A</RecordType><RecordData><member>127.0.0.1</member></RecordData><ResponseCode>NOERROR</ResponseCode><Protocol>UDP</Protocol>")
	default:
		requestID := fmt.Sprintf("route53-%012d", s.nextID)
		s.nextID++
		return route53XMLResult(action+"Response", nil, "<RequestId>"+requestID+"</RequestId>")
	}
}

func (s *route53Store) route53NextHostedZoneIDLocked() string {
	id := fmt.Sprintf("ZSTACKYARD%02d", s.nextID)
	s.nextID++
	return id
}

func (s *route53Store) route53NextChangeIDLocked() string {
	id := fmt.Sprintf("CSTACKYARD%02d", s.nextID)
	s.nextID++
	return id
}

func route53XMLResult(root string, headers map[string]string, body string) route53Result {
	return route53Result{
		Status:  200,
		Headers: headers,
		Body:    route53XMLBody(root, body),
	}
}

func route53XMLBody(root, inner string) []byte {
	return []byte(`<` + root + ` xmlns="https://route53.amazonaws.com/doc/2013-04-01/">` + inner + `</` + root + `>`)
}

func route53LocationHeader(location string) map[string]string {
	return map[string]string{"Location": location}
}

func route53StringOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func route53HostedZoneXML(id string) string {
	return `<HostedZone><Id>/hostedzone/` + id + `</Id><Name>stackyard.example.com.</Name><CallerReference>stackyard</CallerReference><Config><Comment>stackyard</Comment><PrivateZone>false</PrivateZone></Config><ResourceRecordSetCount>1</ResourceRecordSetCount></HostedZone>`
}

func route53ChangeInfoXML(changeID, status, submittedAt string) string {
	return `<ChangeInfo><Id>/change/` + changeID + `</Id><Status>` + status + `</Status><SubmittedAt>` + submittedAt + `</SubmittedAt></ChangeInfo>`
}

func route53DelegationSetXML(id string) string {
	return `<DelegationSet><Id>/delegationset/` + id + `</Id><CallerReference>stackyard</CallerReference><NameServers><NameServer>ns-100.awsdns-01.com</NameServer><NameServer>ns-200.awsdns-02.net</NameServer><NameServer>ns-300.awsdns-03.org</NameServer><NameServer>ns-400.awsdns-04.co.uk</NameServer></NameServers></DelegationSet>`
}

func route53HealthCheckXML(id string) string {
	return `<HealthCheck><Id>` + id + `</Id><CallerReference>stackyard</CallerReference><HealthCheckConfig><IPAddress>127.0.0.1</IPAddress><Port>443</Port><Type>HTTPS</Type><ResourcePath>/health</ResourcePath><RequestInterval>30</RequestInterval><FailureThreshold>3</FailureThreshold></HealthCheckConfig><HealthCheckVersion>1</HealthCheckVersion></HealthCheck>`
}

func route53HealthCheckObservationsXML(checkedTime, status string) string {
	return `<HealthCheckObservations><HealthCheckObservation><IPAddress>127.0.0.1</IPAddress><Region>us-east-1</Region><StatusReport><CheckedTime>` + checkedTime + `</CheckedTime><Status>` + status + `</Status></StatusReport></HealthCheckObservation></HealthCheckObservations>`
}

func route53CidrCollectionXML(id string) string {
	return `<Collection><Arn>arn:aws:route53:::cidrcollection/` + id + `</Arn><Id>` + id + `</Id><Name>stackyard-cidr-collection</Name><Version>1</Version></Collection>`
}

func route53QueryLoggingConfigXML(id, hostedZoneID string) string {
	return `<QueryLoggingConfig><CloudWatchLogsLogGroupArn>arn:aws:logs:us-east-1:123456789012:log-group:/aws/route53/stackyard</CloudWatchLogsLogGroupArn><HostedZoneId>` + hostedZoneID + `</HostedZoneId><Id>` + id + `</Id></QueryLoggingConfig>`
}

func route53TrafficPolicyXML(id string, version int) string {
	return `<TrafficPolicy><Comment>stackyard</Comment><Document>{"AWSPolicyFormatVersion":"2015-10-01"}</Document><Id>` + id + `</Id><Name>stackyard-traffic-policy</Name><Type>A</Type><Version>` + fmt.Sprintf("%d", version) + `</Version></TrafficPolicy>`
}

func route53TrafficPolicyInstanceXML(id, hostedZoneID, policyID string, version int) string {
	return `<TrafficPolicyInstance><HostedZoneId>` + hostedZoneID + `</HostedZoneId><Id>` + id + `</Id><Message>stackyard</Message><Name>stackyard.example.com.</Name><State>Applied</State><TrafficPolicyId>` + policyID + `</TrafficPolicyId><TrafficPolicyType>A</TrafficPolicyType><TrafficPolicyVersion>` + fmt.Sprintf("%d", version) + `</TrafficPolicyVersion><TTL>60</TTL></TrafficPolicyInstance>`
}

func route53KeySigningKeyXML(hostedZoneID, name, now string) string {
	return `<KeySigningKey><CreatedDate>` + now + `</CreatedDate><DNSKEYRecord>257 3 13 stackyard</DNSKEYRecord><DSRecord>12345 13 2 STACKYARD</DSRecord><DigestAlgorithmMnemonic>SHA-256</DigestAlgorithmMnemonic><DigestAlgorithmType>2</DigestAlgorithmType><DigestValue>STACKYARD</DigestValue><Flag>257</Flag><HostedZoneId>` + hostedZoneID + `</HostedZoneId><KeyTag>12345</KeyTag><KmsArn>arn:aws:kms:us-east-1:123456789012:key/stackyard</KmsArn><LastModifiedDate>` + now + `</LastModifiedDate><Name>` + name + `</Name><SigningAlgorithmMnemonic>ECDSAP256SHA256</SigningAlgorithmMnemonic><SigningAlgorithmType>13</SigningAlgorithmType><Status>INACTIVE</Status><StatusMessage>stackyard</StatusMessage></KeySigningKey>`
}

func route53ResourceTagSetXML(resourceID, resourceType string) string {
	return `<ResourceTagSet><ResourceId>` + resourceID + `</ResourceId><ResourceType>` + resourceType + `</ResourceType><Tags><Tag><Key>env</Key><Value>coverage</Value></Tag></Tags></ResourceTagSet>`
}
