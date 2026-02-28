package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage6GovernanceAndMetadata(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	instanceID := "rds-stage6-db"
	instanceARN := "arn:aws:rds:us-east-1:123456789012:db:" + instanceID

	status, body := rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create DB instance 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":           []string{"AddTagsToResource"},
		"ResourceName":     []string{instanceARN},
		"Tags.Tag.1.Key":   []string{"env"},
		"Tags.Tag.1.Value": []string{"stage6"},
		"Tags.Tag.2.Key":   []string{"owner"},
		"Tags.Tag.2.Value": []string{"qa"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected AddTagsToResource 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":       []string{"ListTagsForResource"},
		"ResourceName": []string{instanceARN},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ListTagsForResource 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Key>env</Key>")) {
		t.Fatalf("expected env tag in list response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":           []string{"RemoveTagsFromResource"},
		"ResourceName":     []string{instanceARN},
		"TagKeys.member.1": []string{"owner"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected RemoveTagsFromResource 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                   []string{"CreateEventSubscription"},
		"SubscriptionName":         []string{"rds-stage6-sub"},
		"SnsTopicArn":              []string{"arn:aws:sns:us-east-1:123456789012:rds-events"},
		"SourceType":               []string{"db-instance"},
		"SourceIds.member.1":       []string{instanceID},
		"EventCategories.member.1": []string{"availability"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateEventSubscription 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":           []string{"DescribeEventSubscriptions"},
		"SubscriptionName": []string{"rds-stage6-sub"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeEventSubscriptions 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<CustSubscriptionId>rds-stage6-sub</CustSubscriptionId>")) {
		t.Fatalf("expected subscription in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":           []string{"ModifyEventSubscription"},
		"SubscriptionName": []string{"rds-stage6-sub"},
		"Enabled":          []string{"false"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ModifyEventSubscription 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":             []string{"DescribePendingMaintenanceActions"},
		"ResourceIdentifier": []string{instanceARN},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribePendingMaintenanceActions 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ApplyAction>")) {
		t.Fatalf("expected pending maintenance actions in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":             []string{"ApplyPendingMaintenanceAction"},
		"ResourceIdentifier": []string{instanceARN},
		"ApplyAction":        []string{"system-update"},
		"OptInType":          []string{"immediate"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ApplyPendingMaintenanceAction 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":           []string{"DescribeEvents"},
		"SourceIdentifier": []string{instanceID},
		"SourceType":       []string{"db-instance"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeEvents 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{"Action": []string{"DescribeAccountAttributes"}})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeAccountAttributes 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action": []string{"DescribeDBEngineVersions"},
		"Engine": []string{"mysql"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeDBEngineVersions 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action": []string{"DescribeOrderableDBInstanceOptions"},
		"Engine": []string{"mysql"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeOrderableDBInstanceOptions 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":     []string{"DescribeSourceRegions"},
		"RegionName": []string{"us-east-1"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeSourceRegions 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"DescribeValidDBInstanceModifications"},
		"DBInstanceIdentifier": []string{instanceID},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeValidDBInstanceModifications 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":           []string{"DeleteEventSubscription"},
		"SubscriptionName": []string{"rds-stage6-sub"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DeleteEventSubscription 200, got %d: %s", status, string(body))
	}
}

func TestRDSStage6ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	instanceID := "rds-stage6-impl-db"
	instanceARN := "arn:aws:rds:us-east-1:123456789012:db:" + instanceID

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":           []string{"CreateEventSubscription"},
		"SubscriptionName": []string{"rds-stage6-impl-sub"},
		"SnsTopicArn":      []string{"arn:aws:sns:us-east-1:123456789012:rds-events"},
	})

	cases := []url.Values{
		{"Action": []string{"AddTagsToResource"}, "ResourceName": []string{instanceARN}, "Tags.Tag.1.Key": []string{"k"}, "Tags.Tag.1.Value": []string{"v"}},
		{"Action": []string{"ListTagsForResource"}, "ResourceName": []string{instanceARN}},
		{"Action": []string{"RemoveTagsFromResource"}, "ResourceName": []string{instanceARN}, "TagKeys.member.1": []string{"k"}},
		{"Action": []string{"DescribeEventSubscriptions"}, "SubscriptionName": []string{"rds-stage6-impl-sub"}},
		{"Action": []string{"ModifyEventSubscription"}, "SubscriptionName": []string{"rds-stage6-impl-sub"}, "Enabled": []string{"true"}},
		{"Action": []string{"DescribePendingMaintenanceActions"}, "ResourceIdentifier": []string{instanceARN}},
		{"Action": []string{"ApplyPendingMaintenanceAction"}, "ResourceIdentifier": []string{instanceARN}, "ApplyAction": []string{"system-update"}, "OptInType": []string{"immediate"}},
		{"Action": []string{"DescribeEvents"}},
		{"Action": []string{"DescribeAccountAttributes"}},
		{"Action": []string{"DescribeDBEngineVersions"}},
		{"Action": []string{"DescribeOrderableDBInstanceOptions"}, "Engine": []string{"mysql"}},
		{"Action": []string{"DescribeSourceRegions"}},
		{"Action": []string{"DescribeValidDBInstanceModifications"}, "DBInstanceIdentifier": []string{instanceID}},
		{"Action": []string{"DeleteEventSubscription"}, "SubscriptionName": []string{"rds-stage6-impl-sub"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
