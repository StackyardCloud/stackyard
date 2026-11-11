package server

import (
	"net/http"
	"testing"
)

func TestAzureBlueprintsLifecycle(t *testing.T) {
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
	managementGroup := "contoso"
	apiVersion := "?api-version=" + azureBlueprintsAPIVersion
	mgBase := "/azure/providers/Microsoft.Management/managementGroups/" + managementGroup + "/providers/Microsoft.Blueprint"
	subBase := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Blueprint"

	blueprintPath := mgBase + "/blueprints/platform" + apiVersion
	resp := providerContractRequest(t, ts, http.MethodPut, blueprintPath, []byte(`{
		"properties":{
			"displayName":"Platform Blueprint",
			"description":"management-group blueprint",
			"targetScope":"subscription",
			"parameters":{"location":{"type":"string"}},
			"resourceGroups":{"core":{"name":"rg-core","location":"eastus"}}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating blueprint definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	blueprint := providerContractJSONMap(t, resp)
	if blueprint["type"] != "Microsoft.Blueprint/blueprints" {
		t.Fatalf("unexpected blueprint payload: %#v", blueprint)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, blueprintPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting blueprint definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, blueprintPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading blueprint definition, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, mgBase+"/blueprints"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing management-group blueprints, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listPayload := providerContractJSONMap(t, resp)
	if values, ok := listPayload["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one management-group blueprint, got %#v", listPayload)
	}

	artifactPath := mgBase + "/blueprints/platform/artifacts/storage-account" + apiVersion
	resp = providerContractRequest(t, ts, http.MethodPut, artifactPath, []byte(`{
		"kind":"template",
		"properties":{
			"displayName":"Storage Account",
			"template":{"resources":[{"type":"Microsoft.Storage/storageAccounts","name":"stackyardbp"}]}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating blueprint artifact, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, mgBase+"/blueprints/platform/artifacts"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing blueprint artifacts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	artifacts := providerContractJSONMap(t, resp)
	if values, ok := artifacts["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one blueprint artifact, got %#v", artifacts)
	}

	versionPath := mgBase + "/blueprints/platform/versions/v1" + apiVersion
	resp = providerContractRequest(t, ts, http.MethodPut, versionPath, []byte(`{
		"displayName":"Platform Blueprint v1",
		"description":"first published version"
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 publishing blueprint version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	version := providerContractJSONMap(t, resp)
	if version["type"] != "Microsoft.Blueprint/blueprints/versions" {
		t.Fatalf("unexpected version payload: %#v", version)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, mgBase+"/blueprints/platform/versions"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing published blueprint versions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, mgBase+"/blueprints/platform/versions/v1/artifacts"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing published artifacts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	publishedArtifacts := providerContractJSONMap(t, resp)
	values, ok := publishedArtifacts["value"].([]any)
	if !ok || len(values) != 1 {
		t.Fatalf("expected one published artifact, got %#v", publishedArtifacts)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, mgBase+"/blueprints/platform/versions/v1/artifacts/storage-account"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting published artifact, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	subBlueprintPath := subBase + "/blueprints/workload" + apiVersion
	resp = providerContractRequest(t, ts, http.MethodPut, subBlueprintPath, []byte(`{
		"properties":{
			"displayName":"Workload Blueprint",
			"targetScope":"subscription",
			"parameters":{"environment":{"type":"string"}},
			"resourceGroups":{"app":{"name":"rg-app","location":"westus2"}}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating subscription blueprint definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subBase+"/blueprints"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing subscription blueprints, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	subBlueprints := providerContractJSONMap(t, resp)
	if values, ok := subBlueprints["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one subscription blueprint, got %#v", subBlueprints)
	}

	assignmentPath := subBase + "/blueprintAssignments/assignment-a" + apiVersion
	resp = providerContractRequest(t, ts, http.MethodPut, assignmentPath, []byte(`{
		"location":"eastus",
		"identity":{"type":"SystemAssigned"},
		"properties":{
			"blueprintId":"`+azureBlueprintVersionID("/providers/Microsoft.Management/managementGroups/"+managementGroup, "platform", "v1")+`",
			"locks":{"mode":"AllResourcesDoNotDelete"},
			"parameters":{"location":{"value":"eastus"}},
			"resourceGroups":{"core":{"name":"rg-core","location":"eastus"}}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating blueprint assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	assignment := providerContractJSONMap(t, resp)
	if assignment["type"] != "Microsoft.Blueprint/Assignment" {
		t.Fatalf("unexpected assignment payload: %#v", assignment)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, assignmentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting blueprint assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, assignmentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading blueprint assignment, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subBase+"/blueprintAssignments"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing blueprint assignments, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, subBase+"/blueprintAssignments/assignment-a/whoIsBlueprint"+apiVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from whoIsBlueprint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	whoIs := providerContractJSONMap(t, resp)
	if whoIs["objectId"] == "" {
		t.Fatalf("expected objectId from whoIsBlueprint, got %#v", whoIs)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subBase+"/blueprintAssignments/assignment-a/assignmentOperations"+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing assignment operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	ops := providerContractJSONMap(t, resp)
	values, ok = ops["value"].([]any)
	if !ok || len(values) == 0 {
		t.Fatalf("expected assignment operations, got %#v", ops)
	}
	firstOp, ok := values[0].(map[string]any)
	if !ok {
		t.Fatalf("expected assignment operation map, got %#v", values[0])
	}
	opName, _ := firstOp["name"].(string)
	if opName == "" {
		t.Fatalf("expected assignment operation name, got %#v", firstOp)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subBase+"/blueprintAssignments/assignment-a/assignmentOperations/"+opName+apiVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting assignment operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, assignmentPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting blueprint assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, versionPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting published blueprint version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, artifactPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting blueprint artifact, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, blueprintPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting management-group blueprint definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, subBlueprintPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting subscription blueprint definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureBlueprintsAssignmentValidation(t *testing.T) {
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
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Blueprint"

	resp := providerContractRequest(t, ts, http.MethodPut, base+"/blueprintAssignments/invalid-assignment?api-version="+azureBlueprintsAPIVersion, []byte(`{
		"location":"eastus",
		"identity":{"type":"SystemAssigned"},
		"properties":{
			"resourceGroups":{"core":{"name":"rg-core","location":"eastus"}}
		}
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 creating invalid blueprint assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/blueprints/missing?api-version="+azureBlueprintsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing blueprint definition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
