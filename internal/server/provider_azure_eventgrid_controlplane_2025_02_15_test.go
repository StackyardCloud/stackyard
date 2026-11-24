package server

import (
	"net/http"
	"strings"
	"testing"
)

const egcpTestAPI = "?api-version=2025-02-15"

func TestAzureEventGridTopicsAndSubscriptions(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-eg")
	base := c.sub() + "/resourceGroups/rg-eg/providers/Microsoft.EventGrid"
	topic := base + "/topics/orders"

	created := c.expect(http.MethodPut, topic+egcpTestAPI, `{"location":"westus2","properties":{"inputSchema":"CloudEventSchemaV1_0"}}`, http.StatusCreated)
	if created["properties"].(map[string]any)["endpoint"] != "https://orders.westus2-1.eventgrid.azure.net/api/events" {
		t.Fatalf("unexpected topic: %#v", created)
	}
	keys := c.expect(http.MethodPost, topic+"/listKeys"+egcpTestAPI, "", http.StatusOK)
	rotated := c.expect(http.MethodPost, topic+"/regenerateKey"+egcpTestAPI, `{"keyName":"key2"}`, http.StatusOK)
	if keys["key1"] != rotated["key1"] || keys["key2"] == rotated["key2"] {
		t.Fatalf("regenerateKey key2 must change only key2: %#v -> %#v", keys, rotated)
	}
	c.expectError(http.MethodPost, topic+"/regenerateKey"+egcpTestAPI, `{"keyName":"key3"}`, http.StatusBadRequest, "InvalidRequestContent")

	webhook := `{"properties":{"destination":{"endpointType":"WebHook","properties":{"endpointUrl":"https://hooks.example.com/eg?code=secret","deliveryAttributeMappings":[{"name":"x-source","type":"Static","properties":{"value":"eg"}}]}},"filter":{"includedEventTypes":["Order.Created"]}}}`
	sub := c.expect(http.MethodPut, topic+"/eventSubscriptions/audit"+egcpTestAPI, webhook, http.StatusCreated)
	if sub["properties"].(map[string]any)["topic"] != strings.TrimPrefix(topic, "/azure") {
		t.Fatalf("a topic subscription's topic is the topic: %#v", sub)
	}
	full := c.expect(http.MethodPost, topic+"/eventSubscriptions/audit/getFullUrl"+egcpTestAPI, "", http.StatusOK)
	if full["endpointUrl"] != "https://hooks.example.com/eg?code=secret" {
		t.Fatalf("unexpected full URL: %#v", full)
	}
	if attrs := azureTestList(t, c.expect(http.MethodPost, topic+"/eventSubscriptions/audit/getDeliveryAttributes"+egcpTestAPI, "", http.StatusOK)); len(attrs) != 1 {
		t.Fatalf("unexpected delivery attributes: %#v", attrs)
	}
	resp := c.do(http.MethodPut, topic+"/eventSubscriptions/bad"+egcpTestAPI, `{"properties":{"destination":{"endpointType":"WebHook","properties":{"endpointUrl":"http://insecure"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusBadRequest)
	if msg, _ := providerContractJSONMap(t, resp)["error"].(map[string]any)["message"].(string); !strings.Contains(msg, "endpointUrl must be an https URL") {
		t.Fatalf("unexpected error message %q", msg)
	}
	c.expectError(http.MethodPut, topic+"/eventSubscriptions/bad"+egcpTestAPI, `{"properties":{"destination":{"endpointType":"Carrier"}}}`, http.StatusBadRequest, "InvalidRequestContent")

	// Extension subscriptions attach to any scope: here the resource group (a
	// global topic) and the topic itself (regional).
	group := c.sub() + "/resourceGroups/rg-eg"
	queue := `{"properties":{"destination":{"endpointType":"StorageQueue","properties":{"resourceId":"/subscriptions/` + azureResourcesTestSubscription + `/resourceGroups/rg-eg/providers/Microsoft.Storage/storageAccounts/st1","queueName":"events"}}}}`
	c.expect(http.MethodPut, group+"/providers/Microsoft.EventGrid/eventSubscriptions/rg-watch"+egcpTestAPI, queue, http.StatusCreated)
	c.expect(http.MethodPut, topic+"/providers/Microsoft.EventGrid/eventSubscriptions/regional"+egcpTestAPI, queue, http.StatusCreated)
	c.expectError(http.MethodPost, group+"/providers/Microsoft.EventGrid/eventSubscriptions/rg-watch/getFullUrl"+egcpTestAPI, "", http.StatusBadRequest, "InvalidRequest")

	global := azureTestList(t, c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.EventGrid/topicTypes/Microsoft.Resources.ResourceGroups/eventSubscriptions"+egcpTestAPI, "", http.StatusOK))
	if len(global) != 1 || global[0].(map[string]any)["name"] != "rg-watch" {
		t.Fatalf("expected the resource group subscription, got %#v", global)
	}
	regional := azureTestList(t, c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.EventGrid/locations/westus2/eventSubscriptions"+egcpTestAPI, "", http.StatusOK))
	if len(regional) != 2 {
		t.Fatalf("expected the topic's two subscriptions in westus2, got %#v", regional)
	}
	if none := azureTestList(t, c.expect(http.MethodGet, c.sub()+"/providers/Microsoft.EventGrid/locations/eastus/topicTypes/Microsoft.EventGrid.Topics/eventSubscriptions"+egcpTestAPI, "", http.StatusOK)); len(none) != 0 {
		t.Fatalf("no subscriptions are in eastus: %#v", none)
	}

	types := azureTestList(t, c.expect(http.MethodGet, "/azure/providers/Microsoft.EventGrid/topicTypes/Microsoft.Storage.StorageAccounts/eventTypes"+egcpTestAPI, "", http.StatusOK))
	if len(types) != 2 {
		t.Fatalf("unexpected storage event types: %#v", types)
	}
	ext := c.expect(http.MethodGet, topic+"/providers/Microsoft.EventGrid/extensionTopics/default"+egcpTestAPI, "", http.StatusOK)
	if ext["name"] != "default" {
		t.Fatalf("unexpected extension topic: %#v", ext)
	}

	c.expect(http.MethodDelete, topic+egcpTestAPI, "", http.StatusOK)
	c.expectError(http.MethodGet, topic+"/eventSubscriptions/audit"+egcpTestAPI, "", http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureEventGridPartnersAndDomains(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-egp")
	base := c.sub() + "/resourceGroups/rg-egp/providers/Microsoft.EventGrid"

	domain := c.expect(http.MethodPut, base+"/domains/shop"+egcpTestAPI, `{"location":"eastus"}`, http.StatusCreated)
	if domain["properties"].(map[string]any)["autoCreateTopicWithFirstSubscription"] != true {
		t.Fatalf("unexpected domain defaults: %#v", domain)
	}
	c.expect(http.MethodPut, base+"/domains/shop/topics/carts"+egcpTestAPI, `{}`, http.StatusCreated)
	c.expect(http.MethodPut, base+"/domains/shop/topics/carts/eventSubscriptions/func-sub"+egcpTestAPI,
		`{"properties":{"destination":{"endpointType":"AzureFunction","properties":{"resourceId":"/subscriptions/s/resourceGroups/r/providers/Microsoft.Web/sites/app/functions/f"}}}}`, http.StatusCreated)

	registration := c.expect(http.MethodPut, base+"/partnerRegistrations/acme"+egcpTestAPI, `{"location":"global"}`, http.StatusCreated)
	registrationID := registration["id"].(string)
	c.expectError(http.MethodPut, base+"/partnerNamespaces/acme-ns"+egcpTestAPI, `{"location":"eastus","properties":{}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, base+"/partnerNamespaces/acme-ns"+egcpTestAPI, `{"location":"eastus","properties":{"partnerRegistrationFullyQualifiedId":"`+registrationID+`"}}`, http.StatusCreated)
	c.expect(http.MethodPut, base+"/partnerNamespaces/acme-ns/channels/to-customer"+egcpTestAPI,
		`{"properties":{"channelType":"PartnerTopic","partnerTopicInfo":{"azureSubscriptionId":"`+azureResourcesTestSubscription+`","resourceGroupName":"rg-egp","name":"acme-topic","source":"acme"}}}`, http.StatusCreated)
	c.expectError(http.MethodPost, base+"/partnerNamespaces/acme-ns/channels/to-customer/getFullUrl"+egcpTestAPI, "", http.StatusBadRequest, "InvalidRequest")

	topic := c.expect(http.MethodPut, base+"/partnerTopics/acme-topic"+egcpTestAPI, `{"location":"eastus","properties":{"source":"acme","activationState":"Activated"}}`, http.StatusCreated)
	if topic["properties"].(map[string]any)["activationState"] != "NeverActivated" {
		t.Fatalf("clients can't set the activation state: %#v", topic)
	}
	activated := c.expect(http.MethodPost, base+"/partnerTopics/acme-topic/activate"+egcpTestAPI, "", http.StatusOK)
	if activated["properties"].(map[string]any)["activationState"] != "Activated" {
		t.Fatalf("unexpected activation: %#v", activated)
	}

	config := base + "/partnerConfigurations/default"
	c.expectError(http.MethodPut, base+"/partnerConfigurations/other"+egcpTestAPI, `{"location":"global"}`, http.StatusBadRequest, "InvalidResourceName")
	c.expect(http.MethodPut, config+egcpTestAPI, `{"location":"global","properties":{"partnerAuthorization":{"defaultMaximumExpirationTimeInDays":7}}}`, http.StatusCreated)
	authorized := c.expect(http.MethodPost, config+"/authorizePartner"+egcpTestAPI, `{"partnerName":"acme","authorizationExpirationTimeInUtc":"2027-01-01T00:00:00Z"}`, http.StatusOK)
	partners := authorized["properties"].(map[string]any)["partnerAuthorization"].(map[string]any)["authorizedPartnersList"].([]any)
	if len(partners) != 1 {
		t.Fatalf("expected one authorized partner, got %#v", authorized)
	}
	c.expectError(http.MethodPost, config+"/authorizePartner"+egcpTestAPI, `{}`, http.StatusBadRequest, "InvalidRequestContent")
	unauthorized := c.expect(http.MethodPost, config+"/unauthorizePartner"+egcpTestAPI, `{"partnerName":"acme"}`, http.StatusOK)
	if left := unauthorized["properties"].(map[string]any)["partnerAuthorization"].(map[string]any)["authorizedPartnersList"].([]any); len(left) != 0 {
		t.Fatalf("expected no partners after unauthorizing, got %#v", left)
	}
	if partners := azureTestList(t, c.expect(http.MethodGet, "/azure/providers/Microsoft.EventGrid/verifiedPartners"+egcpTestAPI, "", http.StatusOK)); len(partners) == 0 {
		t.Fatalf("expected verified partners")
	}
}

func TestAzureEventGridNamespaceResources(t *testing.T) {
	t.Parallel()
	c := newAzureARMTestClient(t, "rg-egns")
	ns := c.sub() + "/resourceGroups/rg-egns/providers/Microsoft.EventGrid/namespaces/fleet"

	created := c.expect(http.MethodPut, ns+egcpTestAPI,
		`{"location":"eastus","sku":{"name":"Standard","capacity":1},"properties":{"topicSpacesConfiguration":{"state":"Enabled"},"topicsConfiguration":{"customDomains":[{"fullyQualifiedDomainName":"events.contoso.com"}]}}}`, http.StatusCreated)
	props := created["properties"].(map[string]any)
	if props["topicsConfiguration"].(map[string]any)["hostname"] != "fleet.eastus-1.eventgrid.azure.net" || props["topicSpacesConfiguration"].(map[string]any)["hostname"] != "fleet.eastus-1.ts.eventgrid.azure.net" {
		t.Fatalf("unexpected namespace hostnames: %#v", props)
	}
	owned := c.expect(http.MethodPost, ns+"/validateCustomDomainOwnership"+egcpTestAPI, "", http.StatusOK)
	if domains := owned["customDomainsForTopicsConfiguration"].([]any); len(domains) != 1 || domains[0].(map[string]any)["validationState"] != "Approved" {
		t.Fatalf("unexpected custom domain validation: %#v", owned)
	}
	c.expect(http.MethodPut, ns+"/topics/telemetry"+egcpTestAPI, `{"properties":{"eventRetentionInDays":3}}`, http.StatusCreated)
	c.expectError(http.MethodPut, ns+"/topics/long"+egcpTestAPI, `{"properties":{"eventRetentionInDays":30}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, ns+"/topics/telemetry/eventSubscriptions/pull"+egcpTestAPI, `{"properties":{"deliveryConfiguration":{"deliveryMode":"Queue","queue":{"receiveLockDurationInSeconds":60}}}}`, http.StatusCreated)
	c.expectError(http.MethodPut, ns+"/topics/telemetry/eventSubscriptions/bad"+egcpTestAPI, `{"properties":{"deliveryConfiguration":{"deliveryMode":"Carrier"}}}`, http.StatusBadRequest, "InvalidRequestContent")

	c.expect(http.MethodPut, ns+"/topicSpaces/vehicles"+egcpTestAPI, `{"properties":{"topicTemplates":["vehicles/${client.authenticationName}/#"]}}`, http.StatusCreated)
	c.expectError(http.MethodPut, ns+"/topicSpaces/empty"+egcpTestAPI, `{"properties":{"topicTemplates":[]}}`, http.StatusBadRequest, "InvalidRequestContent")
	client := c.expect(http.MethodPut, ns+"/clients/car-001"+egcpTestAPI, `{"properties":{"authenticationName":"car-001","attributes":{"type":"car"}}}`, http.StatusCreated)
	if client["properties"].(map[string]any)["state"] != "Enabled" {
		t.Fatalf("clients start enabled: %#v", client)
	}
	c.expect(http.MethodPut, ns+"/clientGroups/cars"+egcpTestAPI, `{"properties":{"query":"attributes.type = 'car'"}}`, http.StatusCreated)
	c.expect(http.MethodPut, ns+"/permissionBindings/cars-publish"+egcpTestAPI, `{"properties":{"topicSpaceName":"vehicles","clientGroupName":"cars","permission":"Publisher"}}`, http.StatusCreated)
	c.expectError(http.MethodPut, ns+"/permissionBindings/bad"+egcpTestAPI, `{"properties":{"topicSpaceName":"vehicles","clientGroupName":"cars","permission":"Admin"}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expectError(http.MethodPut, ns+"/caCertificates/root"+egcpTestAPI, `{"properties":{"encodedCertificate":"not a cert"}}`, http.StatusBadRequest, "InvalidRequestContent")
	c.expect(http.MethodPut, ns+"/caCertificates/root"+egcpTestAPI, `{"properties":{"encodedCertificate":"-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"}}`, http.StatusCreated)
	c.expect(http.MethodPut, ns+"/privateEndpointConnections/pe1"+egcpTestAPI, `{"properties":{"privateLinkServiceConnectionState":{"status":"Approved"}}}`, http.StatusCreated)
	c.expectError(http.MethodPut, ns+"/privateEndpointConnections/pe2"+egcpTestAPI, `{"properties":{"privateLinkServiceConnectionState":{"status":"Maybe"}}}`, http.StatusBadRequest, "InvalidRequestContent")
}
