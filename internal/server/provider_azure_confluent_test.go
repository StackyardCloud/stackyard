package server

import (
	"net/http"
	"testing"
)

func TestAzureConfluentLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-confluent"
	organizationName := "stackyard-confluent"
	environmentID := "env-dev"
	clusterID := "lkc-dev"
	topicName := "orders"
	connectorName := "sink-a"
	apiKeyName := "key-a"
	base := "/azure/subscriptions/" + subscriptionID
	orgBase := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Confluent/organizations/" + organizationName
	envBase := orgBase + "/environments/" + environmentID
	clusterBase := envBase + "/clusters/" + clusterID

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Confluent/operations?api-version="+azureConfluentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing confluent operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/providers/Microsoft.Confluent/agreements/default?api-version="+azureConfluentAPIVersion, []byte(`{
		"properties":{
			"accepted":true,
			"plan":"enterprise",
			"product":"AzureConfluent",
			"publisher":"Microsoft"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating marketplace agreement, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Confluent/agreements?api-version="+azureConfluentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing marketplace agreements, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	agreements := providerContractJSONMap(t, resp)
	if values, ok := agreements["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one agreement, got %#v", agreements)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, orgBase+"?api-version="+azureConfluentAPIVersion, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{
			"offerDetail":{"planId":"enterprise","publisherId":"microsoft"},
			"userDetail":{"firstName":"Stack","lastName":"Yard","emailAddress":"owner@example.com"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating organization, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, orgBase+"?api-version="+azureConfluentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading organization, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, orgBase+"?api-version="+azureConfluentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting organization, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	org := providerContractJSONMap(t, resp)
	properties, ok := org["properties"].(map[string]any)
	if !ok || properties["organizationId"] == "" {
		t.Fatalf("expected organization properties, got %#v", org)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Confluent/organizations?api-version="+azureConfluentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing organizations by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Confluent/organizations?api-version="+azureConfluentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing organizations by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, orgBase+"/access/default/createInvitation?api-version="+azureConfluentAPIVersion, []byte(`{
		"invitedUserDetails":{"invitedEmail":"invited@example.com","auth_type":"AUTH_TYPE_SSO"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating invitation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, orgBase+"/access/default/createRoleBinding?api-version="+azureConfluentAPIVersion, []byte(`{
		"principal":"sa-orders",
		"role_name":"DeveloperRead",
		"crn_pattern":"crn://confluent.cloud/organization=stackyard-confluent"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating role binding, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	roleBinding := providerContractJSONMap(t, resp)
	if roleBinding["id"] == "" {
		t.Fatalf("expected role binding id, got %#v", roleBinding)
	}

	for _, path := range []string{
		orgBase + "/access/default/listEnvironments?api-version=" + azureConfluentAPIVersion,
		orgBase + "/access/default/listClusters?api-version=" + azureConfluentAPIVersion,
		orgBase + "/access/default/listInvitations?api-version=" + azureConfluentAPIVersion,
		orgBase + "/access/default/listRoleBindings?api-version=" + azureConfluentAPIVersion,
		orgBase + "/access/default/listServiceAccounts?api-version=" + azureConfluentAPIVersion,
		orgBase + "/access/default/listUsers?api-version=" + azureConfluentAPIVersion,
		orgBase + "/listRegions?api-version=" + azureConfluentAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{}`), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPut, envBase+"?api-version="+azureConfluentAPIVersion, []byte(`{
		"location":"global",
		"properties":{"kind":"StreamGovernance"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterBase+"?api-version="+azureConfluentAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"region":"eastus"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterBase+"/topics/"+topicName+"?api-version="+azureConfluentAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"partitionsCount":6,"replicationFactor":3}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating topic, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterBase+"/connectors/"+connectorName+"?api-version="+azureConfluentAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"connectorType":"Sink","connectorClass":"FileStreamSink"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterBase+"/apiKeys/"+apiKeyName+"?api-version="+azureConfluentAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"description":"orders api key"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating api key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		envBase + "?api-version=" + azureConfluentAPIVersion,
		clusterBase + "?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/topics/" + topicName + "?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/connectors/" + connectorName + "?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/apiKeys/" + apiKeyName + "?api-version=" + azureConfluentAPIVersion,
		orgBase + "/schemaRegistryClusters/lsrc-main?api-version=" + azureConfluentAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 getting %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	for _, path := range []string{
		orgBase + "/environments?api-version=" + azureConfluentAPIVersion,
		envBase + "/clusters?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/topics?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/connectors?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/apiKeys?api-version=" + azureConfluentAPIVersion,
		orgBase + "/schemaRegistryClusters?api-version=" + azureConfluentAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 listing %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Confluent/validations/"+organizationName+"/orgValidate?api-version="+azureConfluentAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"offerDetail":{"planId":"enterprise"}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating organization, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, orgBase+"/access/default/deleteRoleBinding?api-version="+azureConfluentAPIVersion, []byte(`{
		"id":"`+roleBinding["id"].(string)+`"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting role binding, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		clusterBase + "/apiKeys/" + apiKeyName + "?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/connectors/" + connectorName + "?api-version=" + azureConfluentAPIVersion,
		clusterBase + "/topics/" + topicName + "?api-version=" + azureConfluentAPIVersion,
		clusterBase + "?api-version=" + azureConfluentAPIVersion,
		envBase + "?api-version=" + azureConfluentAPIVersion,
		orgBase + "?api-version=" + azureConfluentAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodHead, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for HEAD %s, got %d", path, resp.StatusCode)
		}
	}
}

func TestAzureConfluentValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.Confluent/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid subscription",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/bad/providers/Microsoft.Confluent/agreements?api-version=" + azureConfluentAPIVersion,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid organization name",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Confluent/organizations/bad_name?api-version=" + azureConfluentAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "organization missing location",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Confluent/organizations/stackyard?api-version=" + azureConfluentAPIVersion,
			body:       []byte(`{"properties":{"offerDetail":{"planId":"enterprise"}}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "organization missing offer detail",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Confluent/organizations/stackyard?api-version=" + azureConfluentAPIVersion,
			body:       []byte(`{"location":"eastus","properties":{}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invitation missing email",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Confluent/organizations/stackyard/access/default/createInvitation?api-version=" + azureConfluentAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "delete role binding missing id",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Confluent/organizations/stackyard/access/default/deleteRoleBinding?api-version=" + azureConfluentAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "validation missing offer detail",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Confluent/validations/stackyard/orgValidate?api-version=" + azureConfluentAPIVersion,
			body:       []byte(`{"location":"eastus","properties":{}}`),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
