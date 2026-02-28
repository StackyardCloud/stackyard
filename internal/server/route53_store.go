package server

import (
	"fmt"
	"sync"
	"time"
)

type route53Store struct {
	mu sync.Mutex

	nextID        int64
	hostedZoneID  string
	healthCheckID string
	changeID      string
}

func newRoute53Store() *route53Store {
	return &route53Store{
		nextID:        1,
		hostedZoneID:  "ZSTACKYARD01",
		healthCheckID: "hc-stackyard-01",
		changeID:      "CSTACKYARD01",
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
	hostedZoneID := pathParams["hostedZoneId"]
	if hostedZoneID == "" {
		hostedZoneID = s.hostedZoneID
	}
	healthCheckID := pathParams["healthCheckId"]
	if healthCheckID == "" {
		healthCheckID = s.healthCheckID
	}
	changeID := pathParams["changeId"]
	if changeID == "" {
		changeID = s.changeID
	}

	switch action {
	case "CreateHostedZone":
		s.hostedZoneID = fmt.Sprintf("ZSTACKYARD%02d", s.nextID)
		s.changeID = fmt.Sprintf("CSTACKYARD%02d", s.nextID)
		s.nextID++
		body := fmt.Sprintf(`<CreateHostedZoneResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HostedZone><Id>/hostedzone/%s</Id><Name>stackyard.example.com.</Name><CallerReference>stackyard</CallerReference><Config><PrivateZone>false</PrivateZone></Config><ResourceRecordSetCount>0</ResourceRecordSetCount></HostedZone><ChangeInfo><Id>/change/%s</Id><Status>PENDING</Status><SubmittedAt>%s</SubmittedAt></ChangeInfo><DelegationSet><CallerReference>stackyard</CallerReference><NameServers><NameServer>ns-100.awsdns-01.com</NameServer><NameServer>ns-200.awsdns-02.net</NameServer></NameServers></DelegationSet></CreateHostedZoneResponse>`, s.hostedZoneID, s.changeID, now)
		return route53Result{Status: 200, Body: []byte(body)}
	case "ListHostedZones":
		body := fmt.Sprintf(`<ListHostedZonesResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HostedZones><HostedZone><Id>/hostedzone/%s</Id><Name>stackyard.example.com.</Name><CallerReference>stackyard</CallerReference><Config><PrivateZone>false</PrivateZone></Config><ResourceRecordSetCount>1</ResourceRecordSetCount></HostedZone></HostedZones><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems></ListHostedZonesResponse>`, s.hostedZoneID)
		return route53Result{Status: 200, Body: []byte(body)}
	case "GetHostedZone":
		body := fmt.Sprintf(`<GetHostedZoneResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HostedZone><Id>/hostedzone/%s</Id><Name>stackyard.example.com.</Name><CallerReference>stackyard</CallerReference><Config><PrivateZone>false</PrivateZone></Config><ResourceRecordSetCount>1</ResourceRecordSetCount></HostedZone><DelegationSet><CallerReference>stackyard</CallerReference><NameServers><NameServer>ns-100.awsdns-01.com</NameServer><NameServer>ns-200.awsdns-02.net</NameServer></NameServers></DelegationSet></GetHostedZoneResponse>`, hostedZoneID)
		return route53Result{Status: 200, Body: []byte(body)}
	case "DeleteHostedZone":
		body := fmt.Sprintf(`<DeleteHostedZoneResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ChangeInfo><Id>/change/%s</Id><Status>PENDING</Status><SubmittedAt>%s</SubmittedAt></ChangeInfo></DeleteHostedZoneResponse>`, s.changeID, now)
		return route53Result{Status: 200, Body: []byte(body)}
	case "ChangeResourceRecordSets":
		s.changeID = fmt.Sprintf("CSTACKYARD%02d", s.nextID)
		s.nextID++
		body := fmt.Sprintf(`<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ChangeInfo><Id>/change/%s</Id><Status>PENDING</Status><SubmittedAt>%s</SubmittedAt></ChangeInfo></ChangeResourceRecordSetsResponse>`, s.changeID, now)
		return route53Result{Status: 200, Body: []byte(body)}
	case "GetChange":
		body := fmt.Sprintf(`<GetChangeResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ChangeInfo><Id>/change/%s</Id><Status>INSYNC</Status><SubmittedAt>%s</SubmittedAt></ChangeInfo></GetChangeResponse>`, changeID, now)
		return route53Result{Status: 200, Body: []byte(body)}
	case "ListResourceRecordSets":
		body := `<ListResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ResourceRecordSets><ResourceRecordSet><Name>stackyard.example.com.</Name><Type>A</Type><TTL>60</TTL><ResourceRecords><ResourceRecord><Value>127.0.0.1</Value></ResourceRecord></ResourceRecords></ResourceRecordSet></ResourceRecordSets><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems></ListResourceRecordSetsResponse>`
		return route53Result{Status: 200, Body: []byte(body)}
	case "CreateHealthCheck":
		s.healthCheckID = fmt.Sprintf("hc-stackyard-%02d", s.nextID)
		s.nextID++
		body := fmt.Sprintf(`<CreateHealthCheckResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HealthCheck><Id>%s</Id><CallerReference>stackyard</CallerReference><HealthCheckConfig><IPAddress>127.0.0.1</IPAddress><Port>443</Port><Type>HTTPS</Type><ResourcePath>/health</ResourcePath><RequestInterval>30</RequestInterval><FailureThreshold>3</FailureThreshold></HealthCheckConfig><HealthCheckVersion>1</HealthCheckVersion></HealthCheck></CreateHealthCheckResponse>`, s.healthCheckID)
		return route53Result{Status: 200, Body: []byte(body)}
	case "GetHealthCheck", "UpdateHealthCheck":
		body := fmt.Sprintf(`<%sResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HealthCheck><Id>%s</Id><CallerReference>stackyard</CallerReference><HealthCheckConfig><IPAddress>127.0.0.1</IPAddress><Port>443</Port><Type>HTTPS</Type><ResourcePath>/health</ResourcePath><RequestInterval>30</RequestInterval><FailureThreshold>3</FailureThreshold></HealthCheckConfig><HealthCheckVersion>1</HealthCheckVersion></HealthCheck></%sResponse>`, action, healthCheckID, action)
		return route53Result{Status: 200, Body: []byte(body)}
	case "DeleteHealthCheck":
		body := `<DeleteHealthCheckResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"/>`
		return route53Result{Status: 200, Body: []byte(body)}
	case "ListHealthChecks":
		body := fmt.Sprintf(`<ListHealthChecksResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HealthChecks><HealthCheck><Id>%s</Id><CallerReference>stackyard</CallerReference><HealthCheckConfig><Type>HTTPS</Type></HealthCheckConfig><HealthCheckVersion>1</HealthCheckVersion></HealthCheck></HealthChecks><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems></ListHealthChecksResponse>`, s.healthCheckID)
		return route53Result{Status: 200, Body: []byte(body)}
	case "GetHealthCheckCount":
		body := `<GetHealthCheckCountResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HealthCheckCount>1</HealthCheckCount></GetHealthCheckCountResponse>`
		return route53Result{Status: 200, Body: []byte(body)}
	case "GetHostedZoneCount":
		body := `<GetHostedZoneCountResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><HostedZoneCount>1</HostedZoneCount></GetHostedZoneCountResponse>`
		return route53Result{Status: 200, Body: []byte(body)}
	case "GetCheckerIpRanges":
		body := `<GetCheckerIpRangesResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><CheckerIpRanges><member>198.51.100.1/32</member><member>203.0.113.1/32</member></CheckerIpRanges></GetCheckerIpRangesResponse>`
		return route53Result{Status: 200, Body: []byte(body)}
	case "TestDNSAnswer":
		body := `<TestDNSAnswerResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><Nameserver>ns-100.awsdns-01.com</Nameserver><RecordName>stackyard.example.com</RecordName><RecordType>A</RecordType><RecordData><member>127.0.0.1</member></RecordData><ResponseCode>NOERROR</ResponseCode><Protocol>UDP</Protocol></TestDNSAnswerResponse>`
		return route53Result{Status: 200, Body: []byte(body)}
	default:
		requestID := fmt.Sprintf("route53-%012d", s.nextID)
		s.nextID++
		body := fmt.Sprintf(`<%sResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><RequestId>%s</RequestId></%sResponse>`, action, requestID, action)
		return route53Result{Status: 200, Body: []byte(body)}
	}
}
