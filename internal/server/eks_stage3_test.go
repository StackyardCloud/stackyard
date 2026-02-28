package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestEKSStage3AddonsAndIdentityProvider(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage3-cluster"
	addonName := "vpc-cni"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/addons", []byte(`{"addonName":"`+addonName+`","addonVersion":"v1.18.0","serviceAccountRoleArn":"arn:aws:iam::123456789012:role/stackyard-eks"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/addons", nil)
	assertStatus(t, resp, http.StatusOK)
	var listAddonsOut struct {
		Addons []string `json:"addons"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listAddonsOut); err != nil {
		t.Fatalf("unmarshal list addons: %v", err)
	}
	if !slices.Contains(listAddonsOut.Addons, addonName) {
		t.Fatalf("expected addon %q in list %v", addonName, listAddonsOut.Addons)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/addons/"+addonName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/addons/"+addonName+"/update", []byte(`{"addonVersion":"v1.19.0"}`))
	assertStatus(t, resp, http.StatusOK)
	var updateAddonOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateAddonOut); err != nil {
		t.Fatalf("unmarshal update addon: %v", err)
	}
	if updateAddonOut.Update.ID == "" {
		t.Fatalf("expected update id from update addon")
	}

	resp = eksRequest(t, ts, http.MethodGet, "/addons/supported-versions?addonName="+addonName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/addons/configuration-schemas", []byte(`{"addonName":"`+addonName+`","addonVersion":"v1.19.0"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/identity-provider-configs/associate", []byte(`{"oidc":{"identityProviderConfigName":"idp-one","issuerUrl":"https://issuer.example.com","clientId":"sts.amazonaws.com"}}`))
	assertStatus(t, resp, http.StatusOK)
	var associateIDPOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &associateIDPOut); err != nil {
		t.Fatalf("unmarshal associate identity provider config: %v", err)
	}
	if associateIDPOut.Update.ID == "" {
		t.Fatalf("expected update id from associate identity provider config")
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/identity-provider-configs", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/identity-provider-configs/describe", []byte(`{"identityProviderConfig":{"type":"oidc","name":"idp-one"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/identity-provider-configs/disassociate", []byte(`{"identityProviderConfig":{"type":"oidc","name":"idp-one"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/addons/"+addonName, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestEKSStage3ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage3-implemented"
	addonName := "vpc-cni"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/addons", []byte(`{"addonName":"`+addonName+`","addonVersion":"latest"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/identity-provider-configs/associate", []byte(`{"oidc":{"identityProviderConfigName":"idp-one","issuerUrl":"https://issuer.example.com","clientId":"sts.amazonaws.com"}}`))
	assertStatus(t, resp, http.StatusOK)

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/addons/supported-versions"},
		{method: http.MethodPost, path: "/addons/configuration-schemas", body: []byte(`{"addonName":"` + addonName + `"}`)},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/addons"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/addons/" + addonName},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/addons/" + addonName + "/update", body: []byte(`{}`)},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/identity-provider-configs"},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/identity-provider-configs/describe", body: []byte(`{"identityProviderConfig":{"type":"oidc","name":"idp-one"}}`)},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/identity-provider-configs/disassociate", body: []byte(`{"identityProviderConfig":{"type":"oidc","name":"idp-one"}}`)},
	}

	for _, tc := range cases {
		resp := eksRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
