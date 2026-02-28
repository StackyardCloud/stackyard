package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestEKSStage7RouteCompatibility(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage7-cluster"
	addonName := "vpc-cni"
	principalArn := "arn:aws:iam::123456789012:role/stackyard-eks-access-stage7"
	policyArn := "arn:aws:eks::aws:cluster-access-policy/AmazonEKSViewPolicy"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/addons", []byte(`{"addonName":"`+addonName+`","addonVersion":"latest"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/addons/configuration-schemas?addonName="+url.QueryEscape(addonName)+"&addonVersion=latest", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/access-entries", []byte(`{"principalArn":"`+principalArn+`"}`))
	assertStatus(t, resp, http.StatusOK)

	encodedPrincipal := url.PathEscape(principalArn)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal+"/access-policies", []byte(`{"policyArn":"`+policyArn+`","accessScope":{"type":"cluster"}}`))
	assertStatus(t, resp, http.StatusOK)

	encodedPolicy := url.PathEscape(policyArn)
	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/access-entries/"+encodedPrincipal+"/access-policies/"+encodedPolicy, nil)
	assertStatus(t, resp, http.StatusOK)
}
