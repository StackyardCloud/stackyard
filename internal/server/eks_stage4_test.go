package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestEKSStage4AccessEntriesAndPolicies(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage4-cluster"
	principalArn := "arn:aws:iam::123456789012:role/stackyard-eks-access"
	policyArn := "arn:aws:eks::aws:cluster-access-policy/AmazonEKSViewPolicy"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/access-entries", []byte(`{"principalArn":"`+principalArn+`","type":"STANDARD","kubernetesGroups":["team-a"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/access-entries", nil)
	assertStatus(t, resp, http.StatusOK)
	var listEntriesOut struct {
		AccessEntries []string `json:"accessEntries"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listEntriesOut); err != nil {
		t.Fatalf("unmarshal list access entries: %v", err)
	}
	if !slices.Contains(listEntriesOut.AccessEntries, principalArn) {
		t.Fatalf("expected principal arn %q in list %v", principalArn, listEntriesOut.AccessEntries)
	}

	encodedPrincipal := url.PathEscape(principalArn)
	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal, []byte(`{"username":"dev-user","kubernetesGroups":["team-a","team-b"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/access-policies", nil)
	assertStatus(t, resp, http.StatusOK)
	var listPoliciesOut struct {
		AccessPolicies []struct {
			Arn string `json:"arn"`
		} `json:"accessPolicies"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listPoliciesOut); err != nil {
		t.Fatalf("unmarshal list access policies: %v", err)
	}
	if len(listPoliciesOut.AccessPolicies) == 0 {
		t.Fatalf("expected at least one access policy")
	}

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal+"/access-policies", []byte(`{"policyArn":"`+policyArn+`","accessScope":{"type":"cluster"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal+"/access-policies", nil)
	assertStatus(t, resp, http.StatusOK)
	var listAssociatedOut struct {
		AssociatedAccessPolicies []struct {
			PolicyArn string `json:"policyArn"`
		} `json:"associatedAccessPolicies"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listAssociatedOut); err != nil {
		t.Fatalf("unmarshal list associated access policies: %v", err)
	}
	found := false
	for _, policy := range listAssociatedOut.AssociatedAccessPolicies {
		if policy.PolicyArn == policyArn {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected associated policy %q in response", policyArn)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/access-entries?associatedPolicyArn="+url.QueryEscape(policyArn), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal+"/access-policies?policyArn="+url.QueryEscape(policyArn), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete access entry, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(mustBody(t, resp)), "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException after delete access entry")
	}
}

func TestEKSStage4ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage4-implemented"
	principalArn := "arn:aws:iam::123456789012:role/stackyard-eks-access"
	policyArn := "arn:aws:eks::aws:cluster-access-policy/AmazonEKSViewPolicy"
	encodedPrincipal := url.PathEscape(principalArn)

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/access-entries", []byte(`{"principalArn":"`+principalArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal+"/access-policies", []byte(`{"policyArn":"`+policyArn+`","accessScope":{"type":"cluster"}}`))
	assertStatus(t, resp, http.StatusOK)

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/access-policies"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/access-entries"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/access-entries/" + encodedPrincipal},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/access-entries/" + encodedPrincipal, body: []byte(`{}`)},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/access-entries/" + encodedPrincipal + "/access-policies"},
		{method: http.MethodDelete, path: "/clusters/" + clusterName + "/access-entries/" + encodedPrincipal + "/access-policies?policyArn=" + url.QueryEscape(policyArn)},
	}

	for _, tc := range cases {
		resp := eksRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
