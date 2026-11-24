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

// Azure Virtual Desktop example: drives the emulated API through the Azure SDK pipeline.

type sharedKeyPolicy struct{ account string }

func (p *sharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	return req.Next()
}

type step struct {
	name, method, path string
	body               any
	statuses           []int
}

func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	sub := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	rg := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-desktopvirtualization-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	pipeline := runtime.NewPipeline("stackyard", "desktopvirtualization-2025-10-10", runtime.PipelineOptions{
		PerRetry: []policy.Policy{&sharedKeyPolicy{account: getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")}},
	}, &policy.ClientOptions{})
	fmt.Printf("Stackyard Azure Virtual Desktop (desktopvirtualization-2025-10-10) example using %s\n", endpoint)

	group := "/azure/subscriptions/" + sub + "/resourcegroups/" + rg
	base := "/azure/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/"
	armID := strings.TrimPrefix(base, "/azure")
	ok := []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent}
	_ = armID
	q := "?api-version=2025-10-10"
	steps := []step{
		{"CreateResourceGroup", http.MethodPut, group + "?api-version=2021-04-01", map[string]any{"location": region}, ok},
		{"CreateHostPool", http.MethodPut, base + "Microsoft.DesktopVirtualization/hostPools/hp1" + q, map[string]any{"location": region, "properties": map[string]any{"hostPoolType": "Pooled", "loadBalancerType": "BreadthFirst", "preferredAppGroupType": "Desktop", "maxSessionLimit": 10, "registrationInfo": map[string]any{"registrationTokenOperation": "Update", "expirationTime": "2030-01-01T00:00:00Z"}}}, ok},
		{"RetrieveRegistrationToken", http.MethodPost, base + "Microsoft.DesktopVirtualization/hostPools/hp1/retrieveRegistrationToken" + q, nil, ok},
		{"CreateDesktopGroup", http.MethodPut, base + "Microsoft.DesktopVirtualization/applicationGroups/desktop" + q, map[string]any{"location": region, "properties": map[string]any{"hostPoolArmPath": armID + "Microsoft.DesktopVirtualization/hostPools/hp1", "applicationGroupType": "Desktop"}}, ok},
		{"UpdateDesktop", http.MethodPatch, base + "Microsoft.DesktopVirtualization/applicationGroups/desktop/desktops/SessionDesktop" + q, map[string]any{"properties": map[string]any{"friendlyName": "Work"}}, ok},
		{"CreateWorkspace", http.MethodPut, base + "Microsoft.DesktopVirtualization/workspaces/ws" + q, map[string]any{"location": region, "properties": map[string]any{"applicationGroupReferences": []any{armID + "Microsoft.DesktopVirtualization/applicationGroups/desktop"}}}, ok},
		{"CreateScalingPlan", http.MethodPut, base + "Microsoft.DesktopVirtualization/scalingPlans/plan" + q, map[string]any{"location": region, "properties": map[string]any{"timeZone": "UTC", "hostPoolType": "Pooled", "hostPoolReferences": []any{map[string]any{"hostPoolArmPath": armID + "Microsoft.DesktopVirtualization/hostPools/hp1", "scalingPlanEnabled": true}}}}, ok},
		{"CreatePooledSchedule", http.MethodPut, base + "Microsoft.DesktopVirtualization/scalingPlans/plan/pooledSchedules/weekdays" + q, map[string]any{"properties": map[string]any{"daysOfWeek": []any{"Monday", "Friday"}}}, ok},
		{"ListHostPoolScalingPlans", http.MethodGet, base + "Microsoft.DesktopVirtualization/hostPools/hp1/scalingPlans" + q, nil, ok},
		{"DeleteHostPool", http.MethodDelete, base + "Microsoft.DesktopVirtualization/hostPools/hp1" + q, nil, ok},
		{"DeleteResourceGroup", http.MethodDelete, group + "?api-version=2021-04-01", nil, ok},
	}
	for _, s := range steps {
		if err := run(context.Background(), pipeline, endpoint, s); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", s.name, err)
			os.Exit(1)
		}
	}
}

func run(ctx context.Context, pipeline runtime.Pipeline, endpoint string, s step) error {
	req, err := runtime.NewRequest(ctx, s.method, endpoint+s.path)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if s.body != nil {
		if err := runtime.MarshalAsJSON(req, s.body); err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
	}
	resp, err := pipeline.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	for _, status := range s.statuses {
		if resp.StatusCode == status {
			fmt.Printf("%s -> HTTP %d\n", s.name, resp.StatusCode)
			var pretty any
			if json.Unmarshal(body, &pretty) == nil {
				out, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Println(string(out))
			}
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
