package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFMSStage1ContractShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	var out map[string]any

	resp := fmsRequest(t, ts, "AssociateThirdPartyFirewall", string(mustJSON(t, map[string]any{
		"ThirdPartyFirewall": "PALO_ALTO_NETWORKS_CLOUD_NGFW",
	})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal AssociateThirdPartyFirewall output: %v", err)
	}
	if out["ThirdPartyFirewallStatus"] != "ONBOARD_COMPLETE" {
		t.Fatalf("expected third-party firewall onboarding status, got %#v", out["ThirdPartyFirewallStatus"])
	}

	resp = fmsRequest(t, ts, "GetThirdPartyFirewallAssociationStatus", string(mustJSON(t, map[string]any{
		"ThirdPartyFirewall": "PALO_ALTO_NETWORKS_CLOUD_NGFW",
	})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal GetThirdPartyFirewallAssociationStatus output: %v", err)
	}
	if out["MarketplaceOnboardingStatus"] != "COMPLETE" {
		t.Fatalf("expected marketplace onboarding status, got %#v", out["MarketplaceOnboardingStatus"])
	}
	if out["ThirdPartyFirewallStatus"] != "ONBOARD_COMPLETE" {
		t.Fatalf("expected third-party firewall status, got %#v", out["ThirdPartyFirewallStatus"])
	}

	resp = fmsRequest(t, ts, "PutPolicy", string(mustJSON(t, map[string]any{
		"Policy": map[string]any{
			"PolicyName":          "stage1-policy",
			"RemediationEnabled":  true,
			"ExcludeResourceTags": false,
			"ResourceType":        "AWS::EC2::Instance",
			"SecurityServicePolicyData": map[string]any{
				"Type":               "WAF",
				"ManagedServiceData": "{}",
			},
		},
	})))
	assertStatus(t, resp, http.StatusOK)

	resp = fmsRequest(t, ts, "GetPolicy", string(mustJSON(t, map[string]any{"PolicyId": "policy-00000001"})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal GetPolicy output: %v", err)
	}
	policy, ok := out["Policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected policy object in GetPolicy output, got %#v", out["Policy"])
	}
	if _, ok := policy["ExcludeResourceTags"]; !ok {
		t.Fatalf("expected ExcludeResourceTags in policy output")
	}

	resp = fmsRequest(t, ts, "ListComplianceStatus", string(mustJSON(t, map[string]any{"PolicyId": "policy-00000001"})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal ListComplianceStatus output: %v", err)
	}
	statusList, ok := out["PolicyComplianceStatusList"].([]any)
	if !ok || len(statusList) == 0 {
		t.Fatalf("expected PolicyComplianceStatusList entries, got %#v", out["PolicyComplianceStatusList"])
	}
	firstStatus, ok := statusList[0].(map[string]any)
	if !ok {
		t.Fatalf("expected compliance status object, got %#v", statusList[0])
	}
	results, ok := firstStatus["EvaluationResults"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("expected evaluation results, got %#v", firstStatus["EvaluationResults"])
	}
	firstResult, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected evaluation result object, got %#v", results[0])
	}
	if _, ok := firstResult["ComplianceStatus"].(string); !ok {
		t.Fatalf("expected string ComplianceStatus, got %#v", firstResult["ComplianceStatus"])
	}

	resp = fmsRequest(t, ts, "GetProtectionStatus", string(mustJSON(t, map[string]any{"PolicyId": "policy-00000001"})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal GetProtectionStatus output: %v", err)
	}
	if _, ok := out["Data"].(string); !ok {
		t.Fatalf("expected string Data field, got %#v", out["Data"])
	}

	resp = fmsRequest(t, ts, "GetViolationDetails", string(mustJSON(t, map[string]any{
		"PolicyId":      "policy-00000001",
		"MemberAccount": "123456789012",
		"ResourceId":    "arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001",
		"ResourceType":  "AWS::EC2::Instance",
	})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal GetViolationDetails output: %v", err)
	}
	violation, ok := out["ViolationDetail"].(map[string]any)
	if !ok {
		t.Fatalf("expected violation detail object, got %#v", out["ViolationDetail"])
	}
	if _, ok := violation["ResourceViolations"].([]any); !ok {
		t.Fatalf("expected ResourceViolations list, got %#v", violation["ResourceViolations"])
	}

	resp = fmsRequest(t, ts, "BatchAssociateResource", string(mustJSON(t, map[string]any{
		"ResourceSetIdentifier": "rs-00000001",
		"Items":                 []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001"},
	})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal BatchAssociateResource output: %v", err)
	}
	if out["ResourceSetIdentifier"] != "rs-00000001" {
		t.Fatalf("expected ResourceSetIdentifier in batch associate output, got %#v", out["ResourceSetIdentifier"])
	}

	resp = fmsRequest(t, ts, "BatchDisassociateResource", string(mustJSON(t, map[string]any{
		"ResourceSetIdentifier": "rs-00000001",
		"Items":                 []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001"},
	})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal BatchDisassociateResource output: %v", err)
	}
	if out["ResourceSetIdentifier"] != "rs-00000001" {
		t.Fatalf("expected ResourceSetIdentifier in batch disassociate output, got %#v", out["ResourceSetIdentifier"])
	}

	resp = fmsRequest(t, ts, "DisassociateThirdPartyFirewall", string(mustJSON(t, map[string]any{
		"ThirdPartyFirewall": "PALO_ALTO_NETWORKS_CLOUD_NGFW",
	})))
	assertStatus(t, resp, http.StatusOK)
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal DisassociateThirdPartyFirewall output: %v", err)
	}
	if out["ThirdPartyFirewallStatus"] != "OFFBOARD_COMPLETE" {
		t.Fatalf("expected third-party firewall offboarding status, got %#v", out["ThirdPartyFirewallStatus"])
	}
}
