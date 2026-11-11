package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const frontDoorServiceWebApplicationFirewallAPIVersion = "2025-11-01"

type frontDoorServiceWebApplicationFirewallClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type frontDoorServiceWebApplicationFirewallSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *frontDoorServiceWebApplicationFirewallSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newFrontDoorServiceWebApplicationFirewallClient(endpoint, account, subscriptionKey string) *frontDoorServiceWebApplicationFirewallClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"frontdoorservice-webapplicationfirewall-2025-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&frontDoorServiceWebApplicationFirewallSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey},
			},
		},
		&policy.ClientOptions{},
	)
	return &frontDoorServiceWebApplicationFirewallClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *frontDoorServiceWebApplicationFirewallClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_FRONTDOORSERVICE_WEBAPPLICATIONFIREWALL_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_FRONTDOORSERVICE_WEBAPPLICATIONFIREWALL_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-frontdoor-waf")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus")

	frontDoorName := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_FRONTDOOR_NAME", "frontdoorwaf1")
	frontendEndpointName := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_FRONTEND_ENDPOINT_NAME", "frontendEndpoint1")
	customHostName := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_CUSTOM_HOST", "www.example.com")
	originAddress := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_ORIGIN_ADDRESS", "origin.example.com")

	rulesEngineName := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_RULES_ENGINE_NAME", "rulesEngine1")
	policyName := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_POLICY_NAME", "policy1")
	profileName := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_EXPERIMENT_PROFILE_NAME", "profile1")
	experimentName := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_EXPERIMENT_NAME", "experiment1")
	experimentEndpointA := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_EXPERIMENT_ENDPOINT_A", "endpoint-a.example.net")
	experimentEndpointB := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_EXPERIMENT_ENDPOINT_B", "endpoint-b.example.net")
	reportAggregationInterval := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_REPORT_AGGREGATION_INTERVAL", "Hourly")
	timeSeriesType := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_REPORT_TIMESERIES_TYPE", "MeasurementCounts")
	startDateTimeUTC := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_REPORT_START_UTC", "2025-01-01T00:00:00Z")
	endDateTimeUTC := getenv("STACKYARD_AZURE_FRONTDOOR_WAF_REPORT_END_UTC", "2025-01-01T01:00:00Z")

	client := newFrontDoorServiceWebApplicationFirewallClient(endpoint, account, subscriptionKey)

	providerBase := "/azure/providers/Microsoft.Network"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network"

	frontDoorBase := resourceBase + "/frontDoors/" + frontDoorName
	frontendEndpointBase := frontDoorBase + "/frontendEndpoints/" + frontendEndpointName
	rulesEngineBase := frontDoorBase + "/rulesEngines/" + rulesEngineName
	policyBase := resourceBase + "/FrontDoorWebApplicationFirewallPolicies/" + policyName
	policiesByResourceGroupBase := resourceBase + "/FrontDoorWebApplicationFirewallPolicies"
	policiesBySubscriptionBase := subscriptionBase + "/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies"
	managedRuleSetsBase := subscriptionBase + "/providers/Microsoft.Network/frontDoorWebApplicationFirewallManagedRuleSets"

	networkProfileBase := resourceBase + "/NetworkExperimentProfiles/" + profileName
	networkProfilesByResourceGroupBase := resourceBase + "/NetworkExperimentProfiles"
	networkProfilesBySubscriptionBase := subscriptionBase + "/providers/Microsoft.Network/NetworkExperimentProfiles"
	preconfiguredEndpointsBase := networkProfileBase + "/preconfiguredEndpoints"
	experimentBase := networkProfileBase + "/Experiments/" + experimentName
	experimentsByProfileBase := networkProfileBase + "/Experiments"
	reportLatencyScorecardBase := experimentBase + "/latencyScorecard"
	reportTimeseriesBase := experimentBase + "/timeseries"

	armFrontDoorID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/frontDoors/%s", subscriptionID, resourceGroup, frontDoorName)
	armFrontendEndpointID := armFrontDoorID + "/frontendEndpoints/" + frontendEndpointName
	armDefaultFrontendEndpointID := armFrontDoorID + "/frontendEndpoints/default"
	armBackendPoolID := armFrontDoorID + "/backendPools/backendPool1"
	armHealthProbeSettingsID := armFrontDoorID + "/healthProbeSettings/healthProbeSettings1"
	armLoadBalancingSettingsID := armFrontDoorID + "/loadBalancingSettings/loadBalancingSettings1"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "FrontDoorNameAvailabilityProviderCheck",
			method: http.MethodPost,
			path:   providerBase + "/checkFrontDoorNameAvailability?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"name": frontDoorName,
				"type": "Microsoft.Network/frontDoors",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "FrontDoorNameAvailabilitySubscriptionCheck",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.Network/checkFrontDoorNameAvailability?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"name": frontDoorName,
				"type": "Microsoft.Network/frontDoors",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "FrontDoorsCreateOrUpdate",
			method: http.MethodPut,
			path:   frontDoorBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"environment": "local",
					"service":     "frontdoor-waf",
				},
				"properties": map[string]any{
					"frontendEndpoints": []map[string]any{
						{
							"name": frontendEndpointName,
							"properties": map[string]any{
								"hostName": customHostName,
							},
						},
						{
							"name": "default",
							"properties": map[string]any{
								"hostName": frontDoorName + ".azurefd.net",
							},
						},
					},
					"healthProbeSettings": []map[string]any{
						{
							"name": "healthProbeSettings1",
							"properties": map[string]any{
								"path":              "/",
								"protocol":          "Http",
								"intervalInSeconds": 120,
								"enabledState":      "Enabled",
								"healthProbeMethod": "HEAD",
							},
						},
					},
					"loadBalancingSettings": []map[string]any{
						{
							"name": "loadBalancingSettings1",
							"properties": map[string]any{
								"sampleSize":                4,
								"successfulSamplesRequired": 2,
							},
						},
					},
					"backendPools": []map[string]any{
						{
							"name": "backendPool1",
							"properties": map[string]any{
								"backends": []map[string]any{
									{
										"address":   originAddress,
										"httpPort":  80,
										"httpsPort": 443,
										"weight":    1,
										"priority":  1,
									},
								},
								"loadBalancingSettings": map[string]any{"id": armLoadBalancingSettingsID},
								"healthProbeSettings":   map[string]any{"id": armHealthProbeSettingsID},
							},
						},
					},
					"routingRules": []map[string]any{
						{
							"name": "routingRule1",
							"properties": map[string]any{
								"frontendEndpoints": []map[string]any{{"id": armFrontendEndpointID}, {"id": armDefaultFrontendEndpointID}},
								"acceptedProtocols": []string{"Http", "Https"},
								"patternsToMatch":   []string{"/*"},
								"routeConfiguration": map[string]any{
									"@odata.type": "#Microsoft.Azure.FrontDoor.Models.FrontdoorForwardingConfiguration",
									"backendPool": map[string]any{"id": armBackendPoolID},
								},
								"enabledState": "Enabled",
							},
						},
					},
					"enabledState": "Enabled",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "FrontDoorsGet",
			method:   http.MethodGet,
			path:     frontDoorBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "FrontDoorsListByResourceGroup",
			method:   http.MethodGet,
			path:     resourceBase + "/frontDoors?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "FrontDoorsListBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.Network/frontDoors?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "FrontDoorsValidateCustomDomain",
			method: http.MethodPost,
			path:   frontDoorBase + "/validateCustomDomain?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"hostName": customHostName,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "EndpointsPurgeContent",
			method: http.MethodPost,
			path:   frontDoorBase + "/purge?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"contentPaths": []string{"/pictures.aspx", "/pictures/*"},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "FrontendEndpointsListByFrontDoor",
			method:   http.MethodGet,
			path:     frontDoorBase + "/frontendEndpoints?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "FrontendEndpointsGet",
			method:   http.MethodGet,
			path:     frontendEndpointBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "FrontendEndpointsEnableHttps",
			method: http.MethodPost,
			path:   frontendEndpointBase + "/enableHttps?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"certificateSource": "FrontDoor",
				"protocolType":      "ServerNameIndication",
				"minimumTlsVersion": "1.2",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "FrontendEndpointsDisableHttps",
			method:   http.MethodPost,
			path:     frontendEndpointBase + "/disableHttps?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "RulesEnginesCreateOrUpdate",
			method: http.MethodPut,
			path:   rulesEngineBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"rules": []map[string]any{},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "RulesEnginesGet",
			method:   http.MethodGet,
			path:     rulesEngineBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "RulesEnginesListByFrontDoor",
			method:   http.MethodGet,
			path:     frontDoorBase + "/rulesEngines?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PoliciesCreateOrUpdate",
			method: http.MethodPut,
			path:   policyBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"environment": "local",
					"service":     "frontdoor-waf",
				},
				"sku": map[string]any{
					"name": "Classic_AzureFrontDoor",
				},
				"properties": map[string]any{
					"policySettings": map[string]any{
						"enabledState":                  "Enabled",
						"mode":                          "Prevention",
						"requestBodyCheck":              "Enabled",
						"customBlockResponseStatusCode": 403,
					},
					"managedRules": map[string]any{
						"managedRuleSets": []map[string]any{
							{
								"ruleSetType":    "DefaultRuleSet",
								"ruleSetVersion": "1.0",
							},
						},
					},
					"customRules": map[string]any{
						"rules": []map[string]any{},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "PoliciesGet",
			method:   http.MethodGet,
			path:     policyBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "PoliciesListByResourceGroup",
			method:   http.MethodGet,
			path:     policiesByResourceGroupBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "PoliciesListBySubscription",
			method:   http.MethodGet,
			path:     policiesBySubscriptionBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PoliciesUpdate",
			method: http.MethodPatch,
			path:   policyBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local",
					"updated":     "true",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ManagedRuleSetsList",
			method:   http.MethodGet,
			path:     managedRuleSetsBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "NetworkExperimentProfilesCreateOrUpdate",
			method: http.MethodPut,
			path:   networkProfileBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"environment": "local",
					"service":     "frontdoor-waf",
				},
				"properties": map[string]any{
					"enabledState": "Enabled",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "NetworkExperimentProfilesGet",
			method:   http.MethodGet,
			path:     networkProfileBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NetworkExperimentProfilesListByResourceGroup",
			method:   http.MethodGet,
			path:     networkProfilesByResourceGroupBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "NetworkExperimentProfilesListBySubscription",
			method:   http.MethodGet,
			path:     networkProfilesBySubscriptionBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "PreconfiguredEndpointsList",
			method:   http.MethodGet,
			path:     preconfiguredEndpointsBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ExperimentsCreateOrUpdate",
			method: http.MethodPut,
			path:   experimentBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"description":  "local experiment",
					"enabledState": "Enabled",
					"endpointA": map[string]any{
						"name":     "endpointA",
						"endpoint": experimentEndpointA,
					},
					"endpointB": map[string]any{
						"name":     "endpointB",
						"endpoint": experimentEndpointB,
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "ExperimentsGet",
			method:   http.MethodGet,
			path:     experimentBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ExperimentsListByProfile",
			method:   http.MethodGet,
			path:     experimentsByProfileBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ReportsGetLatencyScorecards",
			method:   http.MethodGet,
			path:     reportLatencyScorecardBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion + "&aggregationInterval=" + reportAggregationInterval,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ReportsGetTimeseries",
			method: http.MethodGet,
			path: reportTimeseriesBase +
				"?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion +
				"&startDateTimeUTC=" + startDateTimeUTC +
				"&endDateTimeUTC=" + endDateTimeUTC +
				"&aggregationInterval=" + reportAggregationInterval +
				"&timeseriesType=" + timeSeriesType,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ExperimentsUpdate",
			method: http.MethodPatch,
			path:   experimentBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"updated": "true",
				},
				"properties": map[string]any{
					"description":  "local experiment updated",
					"enabledState": "Disabled",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "ExperimentsDelete",
			method:   http.MethodDelete,
			path:     experimentBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:   "NetworkExperimentProfilesUpdate",
			method: http.MethodPatch,
			path:   networkProfileBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"updated": "true",
				},
				"properties": map[string]any{
					"enabledState": "Enabled",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "RulesEnginesDelete",
			method:   http.MethodDelete,
			path:     rulesEngineBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "PoliciesDelete",
			method:   http.MethodDelete,
			path:     policyBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "NetworkExperimentProfilesDelete",
			method:   http.MethodDelete,
			path:     networkProfileBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "FrontDoorsDelete",
			method:   http.MethodDelete,
			path:     frontDoorBase + "?api-version=" + frontDoorServiceWebApplicationFirewallAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		response, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
	}
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
