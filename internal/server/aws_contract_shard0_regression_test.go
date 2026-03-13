package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func shard0XMLTagValue(body, tag string) string {
	matches := regexp.MustCompile(`<` + tag + `>([^<]+)</` + tag + `>`).FindStringSubmatch(body)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func assertShard0BodyOmits(t *testing.T, body string, members ...string) {
	t.Helper()
	for _, member := range members {
		if strings.Contains(body, "<"+member+">") || strings.Contains(body, `"`+member+`"`) {
			t.Fatalf("expected response body to omit %q, got %s", member, body)
		}
	}
}

func TestEC2Shard0InstanceConnectAndIpamShapesOmitLegacyFields(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ec2Request(t, ts, "CreateInstanceConnectEndpoint", map[string]string{
		"SubnetId": "subnet-00000001",
	})
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	endpointID := shard0XMLTagValue(body, "instanceConnectEndpointId")
	if strings.TrimSpace(endpointID) == "" {
		t.Fatalf("expected instanceConnectEndpointId in CreateInstanceConnectEndpoint response, got %s", body)
	}
	assertShard0BodyOmits(t, body, "ipAddressType")

	resp = ec2Request(t, ts, "DescribeInstanceConnectEndpoints", map[string]string{
		"InstanceConnectEndpointId.1": endpointID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "ipAddressType")

	resp = ec2Request(t, ts, "DeleteInstanceConnectEndpoint", map[string]string{
		"InstanceConnectEndpointId": endpointID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "ipAddressType")

	resp = ec2Request(t, ts, "CreateIpam", map[string]string{
		"Description":                  "shard0-ipam",
		"OperatingRegion.1.RegionName": "us-east-1",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	ipamID := shard0XMLTagValue(body, "ipamId")
	if strings.TrimSpace(ipamID) == "" {
		t.Fatalf("expected ipamId in CreateIpam response, got %s", body)
	}
	assertShard0BodyOmits(t, body, "enablePrivateGua", "meteredAccount")

	resp = ec2Request(t, ts, "DescribeIpams", map[string]string{
		"IpamId.1": ipamID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "enablePrivateGua", "meteredAccount")

	resp = ec2Request(t, ts, "ModifyIpam", map[string]string{
		"IpamId":      ipamID,
		"Description": "shard0-ipam-updated",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "enablePrivateGua", "meteredAccount")

	resp = ec2Request(t, ts, "DeleteIpam", map[string]string{
		"IpamId": ipamID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "enablePrivateGua", "meteredAccount")
}

func TestEC2Shard0VpcEndpointAndTrafficMirrorShapesOmitLegacyFields(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ec2Request(t, ts, "CreateVpcEndpointServiceConfiguration", map[string]string{
		"NetworkLoadBalancerArn.1": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/shard0/1234567890abcdef",
		"SupportedIpAddressType.1": "ipv4",
		"SupportedRegion.1":        "us-east-1",
	})
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	serviceID := shard0XMLTagValue(body, "serviceId")
	serviceName := shard0XMLTagValue(body, "serviceName")
	if serviceID == "" || serviceName == "" {
		t.Fatalf("expected service identifiers in CreateVpcEndpointServiceConfiguration response, got %s", body)
	}
	assertShard0BodyOmits(t, body, "supportedRegionSet", "serviceRegion")

	resp = ec2Request(t, ts, "DescribeVpcEndpointServiceConfigurations", map[string]string{
		"ServiceId.1": serviceID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "supportedRegionSet")

	resp = ec2Request(t, ts, "CreateVpcEndpoint", map[string]string{
		"VpcId":             "vpc-00000001",
		"ServiceName":       serviceName,
		"VpcEndpointType":   "Interface",
		"SubnetId.1":        "subnet-00000001",
		"SecurityGroupId.1": "sg-00000000",
		"PrivateDnsEnabled": "true",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	endpointID := shard0XMLTagValue(body, "vpcEndpointId")
	if endpointID == "" {
		t.Fatalf("expected vpcEndpointId in CreateVpcEndpoint response, got %s", body)
	}
	assertShard0BodyOmits(t, body, "serviceRegion")

	resp = ec2Request(t, ts, "DescribeVpcEndpointServices", map[string]string{
		"ServiceName.1": serviceName,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "serviceRegion")

	resp = ec2Request(t, ts, "DescribeVpcEndpoints", map[string]string{
		"VpcEndpointId.1": endpointID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "serviceRegion")

	resp = ec2Request(t, ts, "CreateVpcEndpointConnectionNotification", map[string]string{
		"ServiceId":                 serviceID,
		"VpcEndpointId":             endpointID,
		"ConnectionNotificationArn": "arn:aws:sns:us-east-1:123456789012:shard0-topic",
		"ConnectionEvents.1":        "Accept",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "serviceRegion")

	resp = ec2Request(t, ts, "CreateTrafficMirrorFilter", map[string]string{
		"Description": "shard0-filter",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	filterID := shard0XMLTagValue(body, "trafficMirrorFilterId")
	if filterID == "" {
		t.Fatalf("expected trafficMirrorFilterId in CreateTrafficMirrorFilter response, got %s", body)
	}

	resp = ec2Request(t, ts, "CreateTrafficMirrorFilterRule", map[string]string{
		"DestinationCidrBlock":  "10.0.1.0/24",
		"RuleAction":            "accept",
		"RuleNumber":            "1",
		"SourceCidrBlock":       "10.0.0.0/24",
		"TrafficDirection":      "ingress",
		"TrafficMirrorFilterId": filterID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	ruleID := shard0XMLTagValue(body, "trafficMirrorFilterRuleId")
	if ruleID == "" {
		t.Fatalf("expected trafficMirrorFilterRuleId in CreateTrafficMirrorFilterRule response, got %s", body)
	}
	assertShard0BodyOmits(t, body, "tagSet")

	resp = ec2Request(t, ts, "ModifyTrafficMirrorFilterRule", map[string]string{
		"TrafficMirrorFilterRuleId": ruleID,
		"Description":               "shard0-rule",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "tagSet")
}

func TestEC2Shard0StatusAndDescribeShapesOmitLegacyFields(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ec2Request(t, ts, "GetImageBlockPublicAccessState", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "managedBy")

	resp = ec2Request(t, ts, "GetSnapshotBlockPublicAccessState", nil)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "managedBy")

	resp = ec2Request(t, ts, "GetSerialConsoleAccessStatus", nil)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "managedBy")

	resp = ec2Request(t, ts, "RunInstances", map[string]string{
		"ImageId":      "ami-shard0",
		"MinCount":     "1",
		"MaxCount":     "1",
		"InstanceType": "t3.micro",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	instanceID := shard0XMLTagValue(body, "instanceId")
	if instanceID == "" {
		t.Fatalf("expected instanceId in RunInstances response, got %s", body)
	}

	resp = ec2Request(t, ts, "CreateImage", map[string]string{
		"InstanceId": instanceID,
		"Name":       "shard0-image",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	imageID := shard0XMLTagValue(body, "imageId")
	if imageID == "" {
		t.Fatalf("expected imageId in CreateImage response, got %s", body)
	}

	resp = ec2Request(t, ts, "DescribeImages", map[string]string{
		"ImageId.1": imageID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "deregistrationProtection")

	resp = ec2Request(t, ts, "DescribeSecurityGroupRules", nil)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "securityGroupRuleArn")

	resp = ec2Request(t, ts, "DescribeReservedInstancesOfferings", nil)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "availabilityZoneId")

	resp = ec2Request(t, ts, "DescribeSpotPriceHistory", map[string]string{
		"MaxResults": "1",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "availabilityZoneId")

	resp = ec2Request(t, ts, "CreateVolume", map[string]string{
		"AvailabilityZone": "us-east-1a",
		"Size":             "10",
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	volumeID := shard0XMLTagValue(body, "volumeId")
	if volumeID == "" {
		t.Fatalf("expected volumeId in CreateVolume response, got %s", body)
	}

	resp = ec2Request(t, ts, "DescribeVolumeStatus", map[string]string{
		"VolumeId.1": volumeID,
	})
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	assertShard0BodyOmits(t, body, "availabilityZoneId")
}

func TestKeyspacesShard0GetKeyspaceOmitsLegacyReplicationGroupStatuses(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateKeyspace", []byte(`{"keyspaceName":"shard0_ks"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "GetKeyspace", []byte(`{"keyspaceName":"shard0_ks"}`))
	assertStatus(t, resp, http.StatusOK)
	body := mustBody(t, resp)
	assertShard0BodyOmits(t, string(body), "replicationGroupStatuses")

	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode GetKeyspace body: %v", err)
	}
	if got, _ := out["keyspaceName"].(string); got != "shard0_ks" {
		t.Fatalf("expected keyspaceName shard0_ks, got %#v", out["keyspaceName"])
	}
}
