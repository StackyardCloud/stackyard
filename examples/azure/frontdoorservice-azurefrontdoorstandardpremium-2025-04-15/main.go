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

const frontDoorServiceStandardPremiumAPIVersion = "2025-04-15"

type frontDoorServiceStandardPremiumClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type frontDoorServiceStandardPremiumSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *frontDoorServiceStandardPremiumSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newFrontDoorServiceStandardPremiumClient(endpoint, account, subscriptionKey string) *frontDoorServiceStandardPremiumClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"frontdoorservice-azurefrontdoorstandardpremium-2025-04-15",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&frontDoorServiceStandardPremiumSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &frontDoorServiceStandardPremiumClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *frontDoorServiceStandardPremiumClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_FRONTDOORSERVICE_AZUREFRONTDOORSTANDARDPREMIUM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_FRONTDOORSERVICE_AZUREFRONTDOORSTANDARDPREMIUM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-frontdoor")
	location := getenv("STACKYARD_AZURE_LOCATION", "global")

	profileName := getenv("STACKYARD_AZURE_FRONTDOOR_PROFILE_NAME", "profile1")
	endpointName := getenv("STACKYARD_AZURE_FRONTDOOR_ENDPOINT_NAME", "endpoint1")
	originGroupName := getenv("STACKYARD_AZURE_FRONTDOOR_ORIGIN_GROUP_NAME", "origingroup1")
	originName := getenv("STACKYARD_AZURE_FRONTDOOR_ORIGIN_NAME", "origin1")
	routeName := getenv("STACKYARD_AZURE_FRONTDOOR_ROUTE_NAME", "route1")
	ruleSetName := getenv("STACKYARD_AZURE_FRONTDOOR_RULE_SET_NAME", "ruleset1")
	ruleName := getenv("STACKYARD_AZURE_FRONTDOOR_RULE_NAME", "rule1")
	customDomainName := getenv("STACKYARD_AZURE_FRONTDOOR_CUSTOM_DOMAIN_NAME", "domain1")
	secretName := getenv("STACKYARD_AZURE_FRONTDOOR_SECRET_NAME", "secret1")
	securityPolicyName := getenv("STACKYARD_AZURE_FRONTDOOR_SECURITY_POLICY_NAME", "securitypolicy1")

	customDomainHost := getenv("STACKYARD_AZURE_FRONTDOOR_CUSTOM_DOMAIN_HOST", "www.example.com")
	originHostName := getenv("STACKYARD_AZURE_FRONTDOOR_ORIGIN_HOST", "host1.blob.core.windows.net")
	originHostHeader := getenv("STACKYARD_AZURE_FRONTDOOR_ORIGIN_HOST_HEADER", "host1.example.com")
	certificateSecretID := getenv(
		"STACKYARD_AZURE_FRONTDOOR_CERTIFICATE_SECRET_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/kv-frontdoor/secrets/cert1", subscriptionID, resourceGroupName),
	)
	wafPolicyID := getenv(
		"STACKYARD_AZURE_FRONTDOOR_WAF_POLICY_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/frontdoorwebapplicationfirewallpolicies/waf-frontdoor", subscriptionID, resourceGroupName),
	)

	client := newFrontDoorServiceStandardPremiumClient(endpoint, account, subscriptionKey)

	resourceBase := fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Cdn", subscriptionID, resourceGroupName)
	profileBase := fmt.Sprintf("%s/profiles/%s", resourceBase, profileName)
	endpointBase := fmt.Sprintf("%s/afdEndpoints/%s", profileBase, endpointName)
	originGroupBase := fmt.Sprintf("%s/originGroups/%s", profileBase, originGroupName)
	originBase := fmt.Sprintf("%s/originGroups/%s/origins/%s", profileBase, originGroupName, originName)
	customDomainBase := fmt.Sprintf("%s/customDomains/%s", profileBase, customDomainName)
	routeBase := fmt.Sprintf("%s/routes/%s", endpointBase, routeName)
	ruleSetBase := fmt.Sprintf("%s/ruleSets/%s", profileBase, ruleSetName)
	ruleBase := fmt.Sprintf("%s/rules/%s", ruleSetBase, ruleName)
	secretBase := fmt.Sprintf("%s/secrets/%s", profileBase, secretName)
	securityPolicyBase := fmt.Sprintf("%s/securityPolicies/%s", profileBase, securityPolicyName)

	armProfileBase := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Cdn/profiles/%s", subscriptionID, resourceGroupName, profileName)
	armOriginGroupID := fmt.Sprintf("%s/originGroups/%s", armProfileBase, originGroupName)
	armRuleSetID := fmt.Sprintf("%s/ruleSets/%s", armProfileBase, ruleSetName)
	armCustomDomainID := fmt.Sprintf("%s/customDomains/%s", armProfileBase, customDomainName)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CheckEndpointNameAvailability",
			method: http.MethodPost,
			path:   resourceBase + "/checkEndpointNameAvailability?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"name":                              endpointName,
				"type":                              "Microsoft.Cdn/Profiles/AfdEndpoints",
				"autoGeneratedDomainNameLabelScope": "TenantReuse",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AFDProfilesCheckEndpointNameAvailability",
			method: http.MethodPost,
			path:   profileBase + "/checkEndpointNameAvailability?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"name":                              endpointName,
				"type":                              "Microsoft.Cdn/Profiles/AfdEndpoints",
				"autoGeneratedDomainNameLabelScope": "TenantReuse",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AFDProfilesCheckHostNameAvailability",
			method: http.MethodPost,
			path:   profileBase + "/checkHostNameAvailability?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"hostName": customDomainHost,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AFDEndpointsCreate",
			method: http.MethodPut,
			path:   endpointBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"environment": "local",
					"service":     "frontdoorservice",
				},
				"properties": map[string]any{
					"enabledState":                      "Enabled",
					"autoGeneratedDomainNameLabelScope": "TenantReuse",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "AFDEndpointsGet",
			method:   http.MethodGet,
			path:     endpointBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AFDEndpointsListByProfile",
			method:   http.MethodGet,
			path:     profileBase + "/afdEndpoints?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AFDOriginGroupsCreate",
			method: http.MethodPut,
			path:   originGroupBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"loadBalancingSettings": map[string]any{
						"sampleSize":                      3,
						"successfulSamplesRequired":       3,
						"additionalLatencyInMilliseconds": 1000,
					},
					"healthProbeSettings": map[string]any{
						"probePath":              "/healthz",
						"probeRequestType":       "HEAD",
						"probeProtocol":          "Https",
						"probeIntervalInSeconds": 10,
					},
					"sessionAffinityState": "Disabled",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:   "AFDOriginsCreate",
			method: http.MethodPut,
			path:   originBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"hostName":                    originHostName,
					"httpPort":                    80,
					"httpsPort":                   443,
					"originHostHeader":            originHostHeader,
					"enabledState":                "Enabled",
					"priority":                    1,
					"weight":                      1000,
					"enforceCertificateNameCheck": true,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:   "AFDCustomDomainsCreate",
			method: http.MethodPut,
			path:   customDomainBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"hostName": customDomainHost,
					"tlsSettings": map[string]any{
						"certificateType":   "ManagedCertificate",
						"minimumTlsVersion": "TLS12",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "RuleSetsCreate",
			method:   http.MethodPut,
			path:     ruleSetBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:   "RulesCreate",
			method: http.MethodPut,
			path:   ruleBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"order": 1,
					"conditions": []map[string]any{
						{
							"name": "RequestMethod",
							"parameters": map[string]any{
								"typeName":        "DeliveryRuleRequestMethodConditionParameters",
								"operator":        "Equal",
								"matchValues":     []string{"GET"},
								"negateCondition": false,
							},
						},
					},
					"actions": []map[string]any{
						{
							"name": "ModifyResponseHeader",
							"parameters": map[string]any{
								"typeName":     "DeliveryRuleHeaderActionParameters",
								"headerAction": "Overwrite",
								"headerName":   "X-Stackyard",
								"value":        "frontdoor",
							},
						},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:   "RoutesCreate",
			method: http.MethodPut,
			path:   routeBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"originGroup": map[string]any{
						"id": armOriginGroupID,
					},
					"customDomains": []map[string]any{
						{"id": armCustomDomainID},
					},
					"ruleSets": []map[string]any{
						{"id": armRuleSetID},
					},
					"supportedProtocols":  []string{"Https", "Http"},
					"patternsToMatch":     []string{"/*"},
					"forwardingProtocol":  "MatchRequest",
					"linkToDefaultDomain": "Enabled",
					"httpsRedirect":       "Enabled",
					"enabledState":        "Enabled",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:   "AFDEndpointsPurgeContent",
			method: http.MethodPost,
			path:   endpointBase + "/purge?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"contentPaths": []string{"/*"},
				"domains":      []string{customDomainHost},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "SecretsCreate",
			method: http.MethodPut,
			path:   secretBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"parameters": map[string]any{
						"type":          "CustomerCertificate",
						"secretVersion": "abcdef1234578900abcdef1234567890",
						"secretSource": map[string]any{
							"id": certificateSecretID,
						},
						"useLatestVersion": false,
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:   "SecurityPoliciesCreate",
			method: http.MethodPut,
			path:   securityPolicyBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"parameters": map[string]any{
						"type": "WebApplicationFirewall",
						"wafPolicy": map[string]any{
							"id": wafPolicyID,
						},
						"associations": []map[string]any{
							{
								"domains":         []map[string]any{{"id": armCustomDomainID}},
								"patternsToMatch": []string{"/*"},
							},
						},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:   "SecurityPoliciesPatch",
			method: http.MethodPatch,
			path:   securityPolicyBase + "?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"parameters": map[string]any{
						"type": "WebApplicationFirewall",
						"wafPolicy": map[string]any{
							"id": wafPolicyID,
						},
						"associations": []map[string]any{
							{
								"domains":         []map[string]any{{"id": armCustomDomainID}},
								"patternsToMatch": []string{"/api/*"},
							},
						},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "AFDProfilesValidateSecret",
			method: http.MethodPost,
			path:   profileBase + "/validateSecret?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"secretSource": map[string]any{"id": certificateSecretID},
				"secretType":   "CustomerCertificate",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AFDProfilesUpgrade",
			method: http.MethodPost,
			path:   profileBase + "/upgrade?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			payload: map[string]any{
				"wafMappingList": []map[string]any{
					{
						"securityPolicyName": securityPolicyName,
						"changeToWafPolicy":  map[string]any{"id": wafPolicyID},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "AFDCustomDomainsRefreshValidationToken",
			method:   http.MethodPost,
			path:     customDomainBase + "/refreshValidationToken?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "LogAnalyticsGetResources",
			method:   http.MethodGet,
			path:     profileBase + "/getLogAnalyticsResources?api-version=" + frontDoorServiceStandardPremiumAPIVersion,
			statuses: []int{http.StatusOK},
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
