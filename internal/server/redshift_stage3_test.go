package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRedshiftStage3NetworkingAndSecurity(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createSubnet := url.Values{
		"Action":                 []string{"CreateClusterSubnetGroup"},
		"ClusterSubnetGroupName": []string{"subnet-group-1"},
		"Description":            []string{"demo subnet group"},
		"SubnetIds.member.1":     []string{"subnet-aaa"},
		"SubnetIds.member.2":     []string{"subnet-bbb"},
	}
	status, body := redshiftRequest(t, ts, createSubnet)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create subnet group, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ClusterSubnetGroupName>subnet-group-1</ClusterSubnetGroupName>")) {
		t.Fatalf("missing subnet group name: %s", string(body))
	}

	createSec := url.Values{
		"Action":                   []string{"CreateClusterSecurityGroup"},
		"ClusterSecurityGroupName": []string{"sg-1"},
		"Description":              []string{"demo security group"},
	}
	status, body = redshiftRequest(t, ts, createSec)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create security group, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ClusterSecurityGroupName>sg-1</ClusterSecurityGroupName>")) {
		t.Fatalf("missing security group name: %s", string(body))
	}

	createCluster := url.Values{
		"Action":                       []string{"CreateCluster"},
		"ClusterIdentifier":            []string{"demo"},
		"NodeType":                     []string{"dc2.large"},
		"MasterUsername":               []string{"admin"},
		"MasterUserPassword":           []string{"Secret123"},
		"ClusterSubnetGroupName":       []string{"subnet-group-1"},
		"VpcSecurityGroupIds.member.1": []string{"sg-1"},
	}
	status, body = redshiftRequest(t, ts, createCluster)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create cluster, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ClusterSubnetGroupName>subnet-group-1</ClusterSubnetGroupName>")) {
		t.Fatalf("missing subnet group in cluster response: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<VpcSecurityGroupId>sg-1</VpcSecurityGroupId>")) {
		t.Fatalf("missing vpc security group in cluster response: %s", string(body))
	}

	badSubnet := url.Values{
		"Action":                 []string{"CreateCluster"},
		"ClusterIdentifier":      []string{"bad-subnet"},
		"NodeType":               []string{"dc2.large"},
		"MasterUsername":         []string{"admin"},
		"MasterUserPassword":     []string{"Secret123"},
		"ClusterSubnetGroupName": []string{"missing"},
	}
	status, body = redshiftRequest(t, ts, badSubnet)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 bad subnet, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterSubnetGroupNotFound</Code>")) {
		t.Fatalf("expected ClusterSubnetGroupNotFound: %s", string(body))
	}

	badSec := url.Values{
		"Action":                       []string{"CreateCluster"},
		"ClusterIdentifier":            []string{"bad-sec"},
		"NodeType":                     []string{"dc2.large"},
		"MasterUsername":               []string{"admin"},
		"MasterUserPassword":           []string{"Secret123"},
		"VpcSecurityGroupIds.member.1": []string{"missing"},
	}
	status, body = redshiftRequest(t, ts, badSec)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 bad security group, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterSecurityGroupNotFound</Code>")) {
		t.Fatalf("expected ClusterSecurityGroupNotFound: %s", string(body))
	}

	modifyBadSubnet := url.Values{
		"Action":                 []string{"ModifyCluster"},
		"ClusterIdentifier":      []string{"demo"},
		"ClusterSubnetGroupName": []string{"missing"},
	}
	status, body = redshiftRequest(t, ts, modifyBadSubnet)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 modify bad subnet, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterSubnetGroupNotFound</Code>")) {
		t.Fatalf("expected ClusterSubnetGroupNotFound: %s", string(body))
	}

	modifyBadSec := url.Values{
		"Action":                       []string{"ModifyCluster"},
		"ClusterIdentifier":            []string{"demo"},
		"VpcSecurityGroupIds.member.1": []string{"missing"},
	}
	status, body = redshiftRequest(t, ts, modifyBadSec)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 modify bad sec group, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterSecurityGroupNotFound</Code>")) {
		t.Fatalf("expected ClusterSecurityGroupNotFound: %s", string(body))
	}

	createEndpoint := url.Values{
		"Action":                       []string{"CreateEndpointAccess"},
		"EndpointName":                 []string{"ep-1"},
		"ClusterIdentifier":            []string{"demo"},
		"SubnetGroupName":              []string{"subnet-group-1"},
		"VpcSecurityGroupIds.member.1": []string{"sg-1"},
	}
	status, body = redshiftRequest(t, ts, createEndpoint)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create endpoint access, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<EndpointName>ep-1</EndpointName>")) {
		t.Fatalf("missing endpoint name: %s", string(body))
	}

	badEndpoint := url.Values{
		"Action":            []string{"CreateEndpointAccess"},
		"EndpointName":      []string{"ep-bad"},
		"ClusterIdentifier": []string{"missing"},
		"SubnetGroupName":   []string{"subnet-group-1"},
	}
	status, body = redshiftRequest(t, ts, badEndpoint)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 bad endpoint, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterNotFound</Code>")) {
		t.Fatalf("expected ClusterNotFound: %s", string(body))
	}

	describeEndpoint := url.Values{
		"Action":       []string{"DescribeEndpointAccess"},
		"EndpointName": []string{"ep-1"},
	}
	status, body = redshiftRequest(t, ts, describeEndpoint)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe endpoint access, got %d: %s", status, string(body))
	}

	deleteEndpoint := url.Values{
		"Action":       []string{"DeleteEndpointAccess"},
		"EndpointName": []string{"ep-1"},
	}
	status, body = redshiftRequest(t, ts, deleteEndpoint)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete endpoint access, got %d: %s", status, string(body))
	}

	status, body = redshiftRequest(t, ts, describeEndpoint)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 describe deleted endpoint, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>EndpointAccessNotFound</Code>")) {
		t.Fatalf("expected EndpointAccessNotFound: %s", string(body))
	}
}

func TestRedshiftStage3ModifyClusterIamRoles(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createCluster := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"role-demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body := redshiftRequest(t, ts, createCluster)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create cluster, got %d: %s", status, string(body))
	}

	roleArn := "arn:aws:iam::123456789012:role/RedshiftRole"
	addRoles := url.Values{
		"Action":               []string{"ModifyClusterIamRoles"},
		"ClusterIdentifier":    []string{"role-demo"},
		"AddIamRoles.member.1": []string{roleArn},
	}
	status, body = redshiftRequest(t, ts, addRoles)
	if status != http.StatusOK {
		t.Fatalf("expected 200 add iam role, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<IamRoleArn>"+roleArn+"</IamRoleArn>")) {
		t.Fatalf("missing iam role in response: %s", string(body))
	}

	removeRoles := url.Values{
		"Action":                  []string{"ModifyClusterIamRoles"},
		"ClusterIdentifier":       []string{"role-demo"},
		"RemoveIamRoles.member.1": []string{roleArn},
	}
	status, body = redshiftRequest(t, ts, removeRoles)
	if status != http.StatusOK {
		t.Fatalf("expected 200 remove iam role, got %d: %s", status, string(body))
	}
	if bytes.Contains(body, []byte(roleArn)) {
		t.Fatalf("expected iam role removed from response: %s", string(body))
	}

	badRole := url.Values{
		"Action":               []string{"ModifyClusterIamRoles"},
		"ClusterIdentifier":    []string{"role-demo"},
		"AddIamRoles.member.1": []string{"bad-arn"},
	}
	status, body = redshiftRequest(t, ts, badRole)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid iam role, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}
}
