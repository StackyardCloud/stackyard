package server

import (
	"net/http"
	"testing"
)

func TestAzureCommunicationLifecycle(t *testing.T) {
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
	resourceGroup := "rg-communication"
	location := "global"
	communicationName := "stackyardcomm"
	emailServiceName := "stackyardemail"
	domainName := "mail.contoso.com"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	communicationBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Communication/communicationServices/" + communicationName
	emailServiceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Communication/emailServices/" + emailServiceName
	domainBase := emailServiceBase + "/domains/" + domainName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Communication/operations?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, subscriptionBase+"/providers/Microsoft.Communication/checkNameAvailability?api-version="+azureCommunicationAPIVersion, []byte(`{
		"name":"stackyardcomm",
		"type":"Microsoft.Communication/CommunicationServices"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	availability := providerContractJSONMap(t, resp)
	if availability["nameAvailable"] != true {
		t.Fatalf("expected name available, got %#v", availability)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, communicationBase+"?api-version="+azureCommunicationAPIVersion, []byte(`{
		"location":"global",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"dev"},
		"properties":{
			"dataLocation":"United States",
			"disableLocalAuth":false,
			"publicNetworkAccess":"Enabled"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating communication service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, communicationBase+"?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading communication service, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, communicationBase+"?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting communication service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	communication := providerContractJSONMap(t, resp)
	if communication["location"] != location {
		t.Fatalf("expected communication service location %q, got %#v", location, communication)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Communication/communicationServices?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing communication services by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/Microsoft.Communication/communicationServices?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing communication services by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, communicationBase+"/listKeys?api-version="+azureCommunicationAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing keys, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	keys := providerContractJSONMap(t, resp)
	if keys["primaryKey"] == "" || keys["secondaryKey"] == "" {
		t.Fatalf("expected communication service keys, got %#v", keys)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, communicationBase+"/regenerateKey?api-version="+azureCommunicationAPIVersion, []byte(`{"keyType":"Primary"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 regenerating key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, communicationBase+"/linkNotificationHub?api-version="+azureCommunicationAPIVersion, []byte(`{
		"resourceId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-notify/providers/Microsoft.NotificationHubs/namespaces/ns-a/notificationHubs/hub-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 linking notification hub, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, communicationBase+"?api-version="+azureCommunicationAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"disableLocalAuth":true}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching communication service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, communicationBase+"/smtpUsernames/smtp-a?api-version="+azureCommunicationAPIVersion, []byte(`{
		"properties":{
			"username":"smtp-a",
			"entraApplicationId":"11111111-1111-1111-1111-111111111111",
			"tenantId":"22222222-2222-2222-2222-222222222222"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating smtp username, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, communicationBase+"/smtpUsernames?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing smtp usernames, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, emailServiceBase+"?api-version="+azureCommunicationAPIVersion, []byte(`{
		"location":"global",
		"tags":{"env":"dev"},
		"properties":{"dataLocation":"United States"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating email service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, subscriptionBase+"/providers/Microsoft.Communication/listVerifiedExchangeOnlineDomains?api-version="+azureCommunicationAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing verified exchange online domains, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, emailServiceBase+"?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting email service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/Microsoft.Communication/emailServices?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing email services by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, domainBase+"?api-version="+azureCommunicationAPIVersion, []byte(`{
		"location":"global",
		"tags":{"env":"dev"},
		"properties":{
			"domainManagement":"CustomerManaged",
			"userEngagementTracking":"Enabled"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating domain, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, domainBase+"/initiateVerification?api-version="+azureCommunicationAPIVersion, []byte(`{"verificationType":"DKIM"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 initiating verification, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, domainBase+"/cancelVerification?api-version="+azureCommunicationAPIVersion, []byte(`{"verificationType":"DKIM"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 canceling verification, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, domainBase+"?api-version="+azureCommunicationAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"userEngagementTracking":"Disabled"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching domain, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, emailServiceBase+"/domains?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing domains, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, domainBase+"/senderUsernames/sender-a?api-version="+azureCommunicationAPIVersion, []byte(`{
		"properties":{"username":"sender-a","displayName":"Sender A"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating sender username, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, domainBase+"/senderUsernames?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing sender usernames, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, domainBase+"/suppressionLists/list-a?api-version="+azureCommunicationAPIVersion, []byte(`{
		"properties":{"listName":"List A"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating suppression list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, domainBase+"/suppressionLists/list-a/suppressionListAddresses/address-a?api-version="+azureCommunicationAPIVersion, []byte(`{
		"properties":{
			"email":"user@example.com",
			"firstName":"User",
			"lastName":"Example",
			"notes":"seeded"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating suppression list address, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, domainBase+"/suppressionLists/list-a/suppressionListAddresses?api-version="+azureCommunicationAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing suppression list addresses, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		domainBase + "/suppressionLists/list-a/suppressionListAddresses/address-a?api-version=" + azureCommunicationAPIVersion,
		domainBase + "/suppressionLists/list-a?api-version=" + azureCommunicationAPIVersion,
		domainBase + "/senderUsernames/sender-a?api-version=" + azureCommunicationAPIVersion,
		domainBase + "?api-version=" + azureCommunicationAPIVersion,
		emailServiceBase + "?api-version=" + azureCommunicationAPIVersion,
		communicationBase + "/smtpUsernames/smtp-a?api-version=" + azureCommunicationAPIVersion,
		communicationBase + "?api-version=" + azureCommunicationAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodHead, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for HEAD %s, got %d", path, resp.StatusCode)
		}
	}

	for _, path := range []string{
		domainBase + "/suppressionLists/list-a/suppressionListAddresses/address-a?api-version=" + azureCommunicationAPIVersion,
		domainBase + "/suppressionLists/list-a?api-version=" + azureCommunicationAPIVersion,
		domainBase + "/senderUsernames/sender-a?api-version=" + azureCommunicationAPIVersion,
		domainBase + "?api-version=" + azureCommunicationAPIVersion,
		emailServiceBase + "?api-version=" + azureCommunicationAPIVersion,
		communicationBase + "/smtpUsernames/smtp-a?api-version=" + azureCommunicationAPIVersion,
		communicationBase + "?api-version=" + azureCommunicationAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 deleting %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}
}

func TestAzureCommunicationValidation(t *testing.T) {
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
			path:       "/azure/providers/Microsoft.Communication/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid name availability type",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Communication/checkNameAvailability?api-version=" + azureCommunicationAPIVersion,
			body:       []byte(`{"name":"comm-a","type":"Wrong.Type"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "create communication service missing data location",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Communication/communicationServices/comm-a?api-version=" + azureCommunicationAPIVersion,
			body:       []byte(`{"location":"global","properties":{}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "regenerate invalid key type",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Communication/communicationServices/comm-a/regenerateKey?api-version=" + azureCommunicationAPIVersion,
			body:       []byte(`{"keyType":"Tertiary"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "create domain without parent email service",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Communication/emailServices/email-a/domains/example.com?api-version=" + azureCommunicationAPIVersion,
			body:       []byte(`{"location":"global","properties":{"domainManagement":"CustomerManaged"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid suppression address email",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Communication/emailServices/email-a/domains/example.com/suppressionLists/list-a/suppressionListAddresses/address-a?api-version=" + azureCommunicationAPIVersion,
			body:       []byte(`{"properties":{"email":"not-an-email"}}`),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
