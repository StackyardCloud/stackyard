package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestEKSStage5CapabilitiesInsightsPodIdentityAndSubscriptions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage5-cluster"
	capabilityName := "compute"
	registrationName := "eks-stage5-registered"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/capabilities", []byte(`{"capabilityName":"`+capabilityName+`","tags":{"env":"test"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/capabilities", nil)
	assertStatus(t, resp, http.StatusOK)
	var listCapabilitiesOut struct {
		Capabilities []struct {
			CapabilityName string `json:"capabilityName"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listCapabilitiesOut); err != nil {
		t.Fatalf("unmarshal list capabilities: %v", err)
	}
	foundCapability := false
	for _, capability := range listCapabilitiesOut.Capabilities {
		if capability.CapabilityName == capabilityName {
			foundCapability = true
			break
		}
	}
	if !foundCapability {
		t.Fatalf("expected capability %q in list", capabilityName)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/capabilities/"+capabilityName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/capabilities/"+capabilityName, []byte(`{"tags":{"owner":"platform"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/insights", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var listInsightsOut struct {
		Insights []struct {
			ID string `json:"id"`
		} `json:"insights"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listInsightsOut); err != nil {
		t.Fatalf("unmarshal list insights: %v", err)
	}
	if len(listInsightsOut.Insights) == 0 || listInsightsOut.Insights[0].ID == "" {
		t.Fatalf("expected at least one insight")
	}
	insightID := listInsightsOut.Insights[0].ID

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/insights/"+insightID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/insights-refresh", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/insights-refresh", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/pod-identity-associations", []byte(`{"namespace":"default","serviceAccount":"app","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks-pod"}`))
	assertStatus(t, resp, http.StatusOK)
	var createAssociationOut struct {
		Association struct {
			AssociationID string `json:"associationId"`
		} `json:"association"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createAssociationOut); err != nil {
		t.Fatalf("unmarshal create pod identity association: %v", err)
	}
	if createAssociationOut.Association.AssociationID == "" {
		t.Fatalf("expected association id")
	}
	associationID := createAssociationOut.Association.AssociationID

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/pod-identity-associations?namespace=default", nil)
	assertStatus(t, resp, http.StatusOK)
	var listAssociationsOut struct {
		Associations []struct {
			AssociationID string `json:"associationId"`
		} `json:"associations"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listAssociationsOut); err != nil {
		t.Fatalf("unmarshal list pod identity associations: %v", err)
	}
	ids := make([]string, 0, len(listAssociationsOut.Associations))
	for _, association := range listAssociationsOut.Associations {
		ids = append(ids, association.AssociationID)
	}
	if !slices.Contains(ids, associationID) {
		t.Fatalf("expected association %q in list %v", associationID, ids)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/pod-identity-associations/"+associationID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/pod-identity-associations/"+associationID, []byte(`{"roleArn":"arn:aws:iam::123456789012:role/stackyard-eks-pod-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/pod-identity-associations/"+associationID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/eks-anywhere-subscriptions", []byte(`{"name":"stackyard-subscription","term":{"duration":12,"unit":"MONTHS"},"licenseQuantity":1,"licenseType":"Cluster","autoRenew":true}`))
	assertStatus(t, resp, http.StatusOK)
	var createSubscriptionOut struct {
		Subscription struct {
			ID string `json:"id"`
		} `json:"subscription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createSubscriptionOut); err != nil {
		t.Fatalf("unmarshal create subscription: %v", err)
	}
	if createSubscriptionOut.Subscription.ID == "" {
		t.Fatalf("expected subscription id")
	}
	subscriptionID := createSubscriptionOut.Subscription.ID

	resp = eksRequest(t, ts, http.MethodGet, "/eks-anywhere-subscriptions", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/eks-anywhere-subscriptions/"+subscriptionID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/eks-anywhere-subscriptions/"+subscriptionID, []byte(`{"autoRenew":false}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/eks-anywhere-subscriptions/"+subscriptionID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/cluster-registrations", []byte(`{"name":"`+registrationName+`","connectorConfig":{"roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","provider":"OTHER"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/cluster-registrations/"+registrationName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/capabilities/"+capabilityName, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestEKSStage5ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage5-implemented"
	capabilityName := "compute"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/capabilities", []byte(`{"capabilityName":"`+capabilityName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/pod-identity-associations", []byte(`{"namespace":"default","serviceAccount":"app","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks-pod"}`))
	assertStatus(t, resp, http.StatusOK)

	var createAssociationOut struct {
		Association struct {
			AssociationID string `json:"associationId"`
		} `json:"association"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createAssociationOut); err != nil {
		t.Fatalf("unmarshal create pod identity association: %v", err)
	}
	associationID := createAssociationOut.Association.AssociationID

	resp = eksRequest(t, ts, http.MethodPost, "/eks-anywhere-subscriptions", []byte(`{"name":"stackyard-subscription","term":{"duration":12,"unit":"MONTHS"},"licenseQuantity":1,"licenseType":"Cluster","autoRenew":true}`))
	assertStatus(t, resp, http.StatusOK)
	var createSubscriptionOut struct {
		Subscription struct {
			ID string `json:"id"`
		} `json:"subscription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createSubscriptionOut); err != nil {
		t.Fatalf("unmarshal create subscription: %v", err)
	}
	subscriptionID := createSubscriptionOut.Subscription.ID

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/capabilities"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/capabilities/" + capabilityName},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/capabilities/" + capabilityName, body: []byte(`{}`)},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/insights", body: []byte(`{}`)},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/insights/insight-upgrade-readiness"},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/insights-refresh", body: []byte(`{}`)},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/insights-refresh"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/pod-identity-associations"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/pod-identity-associations/" + associationID},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/pod-identity-associations/" + associationID, body: []byte(`{"roleArn":"arn:aws:iam::123456789012:role/stackyard-eks-pod-updated"}`)},
		{method: http.MethodGet, path: "/eks-anywhere-subscriptions"},
		{method: http.MethodGet, path: "/eks-anywhere-subscriptions/" + subscriptionID},
		{method: http.MethodPost, path: "/eks-anywhere-subscriptions/" + subscriptionID, body: []byte(`{"autoRenew":false}`)},
		{method: http.MethodPost, path: "/cluster-registrations", body: []byte(`{"name":"eks-stage5-reg","connectorConfig":{"roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","provider":"OTHER"}}`)},
		{method: http.MethodDelete, path: "/cluster-registrations/eks-stage5-reg"},
	}

	for _, tc := range cases {
		resp := eksRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
