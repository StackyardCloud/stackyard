package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage3ConfigurationSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := rdsRequest(t, ts, url.Values{
		"Action":                   []string{"CreateDBSubnetGroup"},
		"DBSubnetGroupName":        []string{"rds-stage3-subnet"},
		"DBSubnetGroupDescription": []string{"stage3 subnet"},
		"SubnetIds.member.1":       []string{"subnet-12345678"},
		"SubnetIds.member.2":       []string{"subnet-87654321"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create subnet group 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                 []string{"CreateDBParameterGroup"},
		"DBParameterGroupName":   []string{"rds-stage3-param"},
		"DBParameterGroupFamily": []string{"mysql8.0"},
		"Description":            []string{"stage3 param"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create parameter group 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                             []string{"ModifyDBParameterGroup"},
		"DBParameterGroupName":               []string{"rds-stage3-param"},
		"Parameters.member.1.ParameterName":  []string{"autocommit"},
		"Parameters.member.1.ParameterValue": []string{"1"},
		"Parameters.member.1.ApplyMethod":    []string{"immediate"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected modify parameter group 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"DescribeDBParameters"},
		"DBParameterGroupName": []string{"rds-stage3-param"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe DB parameters 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ParameterName>autocommit</ParameterName>")) {
		t.Fatalf("expected autocommit parameter in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                            []string{"ResetDBParameterGroup"},
		"DBParameterGroupName":              []string{"rds-stage3-param"},
		"Parameters.member.1.ParameterName": []string{"autocommit"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected reset parameter group 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                 []string{"CreateOptionGroup"},
		"OptionGroupName":        []string{"rds-stage3-option"},
		"EngineName":             []string{"mysql"},
		"MajorEngineVersion":     []string{"8.0"},
		"OptionGroupDescription": []string{"stage3 option"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create option group 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                               []string{"ModifyOptionGroup"},
		"OptionGroupName":                      []string{"rds-stage3-option"},
		"OptionsToInclude.member.1.OptionName": []string{"MARIADB_AUDIT_PLUGIN"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected modify option group 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<OptionName>MARIADB_AUDIT_PLUGIN</OptionName>")) {
		t.Fatalf("expected option to be included: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                     []string{"CreateDBSecurityGroup"},
		"DBSecurityGroupName":        []string{"rds-stage3-sec"},
		"DBSecurityGroupDescription": []string{"stage3 sec"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create security group 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"AuthorizeDBSecurityGroupIngress"},
		"DBSecurityGroupName": []string{"rds-stage3-sec"},
		"CIDRIP":              []string{"10.0.0.0/24"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected authorize ingress 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"DescribeDBSecurityGroups"},
		"DBSecurityGroupName": []string{"rds-stage3-sec"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe security groups 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<CIDRIP>10.0.0.0/24</CIDRIP>")) {
		t.Fatalf("expected CIDR ingress in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"RevokeDBSecurityGroupIngress"},
		"DBSecurityGroupName": []string{"rds-stage3-sec"},
		"CIDRIP":              []string{"10.0.0.0/24"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected revoke ingress 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{"Action": []string{"DescribeCertificates"}})
	if status != http.StatusOK {
		t.Fatalf("expected describe certificates 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<CertificateIdentifier>")) {
		t.Fatalf("expected certificate entries in response: %s", string(body))
	}
}

func TestRDSStage3ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                   []string{"CreateDBSubnetGroup"},
		"DBSubnetGroupName":        []string{"rds-stage3-impl-subnet"},
		"DBSubnetGroupDescription": []string{"impl"},
		"SubnetIds.member.1":       []string{"subnet-12345678"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                 []string{"CreateDBParameterGroup"},
		"DBParameterGroupName":   []string{"rds-stage3-impl-param"},
		"DBParameterGroupFamily": []string{"mysql8.0"},
		"Description":            []string{"impl"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                 []string{"CreateOptionGroup"},
		"OptionGroupName":        []string{"rds-stage3-impl-option"},
		"EngineName":             []string{"mysql"},
		"MajorEngineVersion":     []string{"8.0"},
		"OptionGroupDescription": []string{"impl"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                     []string{"CreateDBSecurityGroup"},
		"DBSecurityGroupName":        []string{"rds-stage3-impl-sec"},
		"DBSecurityGroupDescription": []string{"impl"},
	})

	cases := []url.Values{
		{"Action": []string{"DescribeDBParameterGroups"}},
		{"Action": []string{"DescribeDBParameters"}, "DBParameterGroupName": []string{"rds-stage3-impl-param"}},
		{"Action": []string{"ModifyDBParameterGroup"}, "DBParameterGroupName": []string{"rds-stage3-impl-param"}, "Parameters.member.1.ParameterName": []string{"autocommit"}, "Parameters.member.1.ParameterValue": []string{"1"}},
		{"Action": []string{"ResetDBParameterGroup"}, "DBParameterGroupName": []string{"rds-stage3-impl-param"}, "ResetAllParameters": []string{"true"}},
		{"Action": []string{"DescribeOptionGroups"}, "OptionGroupName": []string{"rds-stage3-impl-option"}},
		{"Action": []string{"ModifyOptionGroup"}, "OptionGroupName": []string{"rds-stage3-impl-option"}, "OptionsToInclude.member.1.OptionName": []string{"MARIADB_AUDIT_PLUGIN"}},
		{"Action": []string{"DescribeDBSubnetGroups"}, "DBSubnetGroupName": []string{"rds-stage3-impl-subnet"}},
		{"Action": []string{"ModifyDBSubnetGroup"}, "DBSubnetGroupName": []string{"rds-stage3-impl-subnet"}, "SubnetIds.member.1": []string{"subnet-12345678"}},
		{"Action": []string{"DescribeDBSecurityGroups"}, "DBSecurityGroupName": []string{"rds-stage3-impl-sec"}},
		{"Action": []string{"AuthorizeDBSecurityGroupIngress"}, "DBSecurityGroupName": []string{"rds-stage3-impl-sec"}, "CIDRIP": []string{"10.0.0.0/24"}},
		{"Action": []string{"RevokeDBSecurityGroupIngress"}, "DBSecurityGroupName": []string{"rds-stage3-impl-sec"}, "CIDRIP": []string{"10.0.0.0/24"}},
		{"Action": []string{"DescribeCertificates"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
