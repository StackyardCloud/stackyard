package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const azureHNTestAPI = "?api-version=2023-09-01"

type azureHNTest struct {
	t    *testing.T
	sub  string
	base string // the resource group's Microsoft.HybridNetwork path
	arm  string // base without the /azure prefix, as resource IDs have it
	do   func(method, path, body string) *http.Response
}

func newAzureHNTest(t *testing.T) *azureHNTest {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-aosm")
	base := sub + "/resourceGroups/rg-aosm/providers/Microsoft.HybridNetwork/"
	return &azureHNTest{t: t, sub: sub, base: base, arm: strings.TrimPrefix(base, "/azure"), do: do}
}

// call sends body (marshalled unless a string) and checks the status,
// returning the decoded response, if any.
func (h *azureHNTest) call(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	text, ok := body.(string)
	if !ok && body != nil {
		raw, _ := json.Marshal(body)
		text = string(raw)
	}
	resp := h.do(method, path+azureHNTestAPI, text)
	raw := mustBody(h.t, resp)
	if resp.StatusCode != want {
		h.t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func (h *azureHNTest) props(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	props, _ := h.call(method, path, body, want)["properties"].(map[string]any)
	return props
}

// rejects checks that a request fails with 400 naming text.
func (h *azureHNTest) rejects(method, path string, body any, text string) {
	h.t.Helper()
	out := h.call(method, path, body, http.StatusBadRequest)
	if msg := azureARMString(azureARMObject(out["error"])["message"]); !strings.Contains(msg, text) {
		h.t.Fatalf("%s %s error = %q, want it to mention %q", method, path, msg, text)
	}
}

func azureHNResource(location string, props map[string]any) map[string]any {
	if props == nil {
		return map[string]any{"location": location}
	}
	return map[string]any{"location": location, "properties": props}
}

// publish creates publisher pub with an ACR store and a storage account
// store holding uploaded artifacts.
func (h *azureHNTest) publish(pub string) {
	h.t.Helper()
	p := h.base + "publishers/" + pub
	h.call(http.MethodPut, p, azureHNResource("eastus", map[string]any{"scope": "Private"}), http.StatusCreated)
	h.call(http.MethodPut, p+"/artifactStores/acr", azureHNResource("eastus", map[string]any{"storeType": "AzureContainerRegistry"}), http.StatusCreated)
	h.call(http.MethodPut, p+"/artifactStores/sa", azureHNResource("eastus", map[string]any{"storeType": "AzureStorageAccount"}), http.StatusCreated)
	h.call(http.MethodPut, p+"/artifactStores/acr/artifactManifests/cnf", azureHNResource("eastus", map[string]any{"artifacts": []any{
		map[string]any{"artifactName": "nginx", "artifactType": "OCIArtifact", "artifactVersion": "1.0.0"},
	}}), http.StatusCreated)
}

func (h *azureHNTest) cnfDefinition(pub string) map[string]any {
	return azureHNResource("eastus", map[string]any{
		"networkFunctionType": "ContainerizedNetworkFunction",
		"deployParameters":    `{"type":"object","properties":{"releaseName":{"type":"string"}},"required":["releaseName"]}`,
		"networkFunctionTemplate": map[string]any{"nfviType": "AzureArcKubernetes", "networkFunctionApplications": []any{map[string]any{
			"name": "nginx", "artifactType": "HelmPackage",
			"artifactProfile": map[string]any{
				"artifactStore":       map[string]any{"id": h.arm + "publishers/" + pub + "/artifactStores/acr"},
				"helmArtifactProfile": map[string]any{"helmPackageName": "nginx", "helmPackageVersionRange": "~1.0.0"},
			},
			"deployParametersMappingRuleProfile": map[string]any{"applicationEnablement": "Enabled"},
		}}},
	})
}

func TestAzureHybridNetworkPublisherAndArtifacts(t *testing.T) {
	t.Parallel()
	h := newAzureHNTest(t)
	pub := h.base + "publishers/contoso"
	if props := h.props(http.MethodPut, pub, azureHNResource("eastus", nil), http.StatusCreated); props["scope"] != "Private" || props["provisioningState"] != "Succeeded" {
		t.Fatalf("publisher = %#v", props)
	}
	h.rejects(http.MethodPut, pub, azureHNResource("eastus", map[string]any{"scope": "Public"}), "scope can't be changed")
	h.rejects(http.MethodPut, pub, azureHNResource("eastus", map[string]any{"scope": "Shared"}), "properties.scope must be one of")
	h.call(http.MethodPut, h.sub+"/resourcegroups/rg-other", map[string]any{"location": "westus"}, http.StatusCreated)
	other := h.sub + "/resourceGroups/rg-other/providers/Microsoft.HybridNetwork/publishers/"
	h.rejects(http.MethodPut, other+"Contoso", azureHNResource("eastus", nil), "unique in a region")
	h.call(http.MethodPut, other+"contoso", azureHNResource("westus", nil), http.StatusCreated)
	h.rejects(http.MethodPut, h.base+"publishers/-bad", azureHNResource("eastus", nil), "name '-bad' is invalid")
	if items := h.call(http.MethodGet, h.sub+"/providers/Microsoft.HybridNetwork/publishers", nil, http.StatusOK)["value"].([]any); len(items) != 2 {
		t.Fatalf("subscription publishers = %d", len(items))
	}

	acr := h.props(http.MethodPut, pub+"/artifactStores/acr", azureHNResource("eastus", nil), http.StatusCreated)
	if acr["storeType"] != "AzureContainerRegistry" || acr["replicationStrategy"] != "SingleReplication" ||
		!strings.Contains(azureARMString(acr["storageResourceId"]), "/Microsoft.ContainerRegistry/registries/acr") {
		t.Fatalf("acr store = %#v", acr)
	}
	h.rejects(http.MethodPut, pub+"/artifactStores/acr", azureHNResource("eastus", map[string]any{"storeType": "AzureStorageAccount"}), "storeType can't be changed")
	sa := h.props(http.MethodPut, pub+"/artifactStores/sa", azureHNResource("eastus", map[string]any{"storeType": "AzureStorageAccount",
		"managedResourceGroupConfiguration": map[string]any{"name": "aosm-mrg", "location": "eastus"}}), http.StatusCreated)
	if !strings.Contains(azureARMString(sa["storageResourceId"]), "/resourceGroups/aosm-mrg/providers/Microsoft.Storage/storageAccounts/sa") {
		t.Fatalf("storage store = %#v", sa)
	}
	// A re-PUT that sends part of the managed resource group is unchanged.
	again := h.props(http.MethodPut, pub+"/artifactStores/sa", azureHNResource("eastus", map[string]any{"storeType": "AzureStorageAccount",
		"managedResourceGroupConfiguration": map[string]any{"name": "aosm-mrg"}}), http.StatusOK)
	if again["storageResourceId"] != sa["storageResourceId"] || azureARMObject(again["managedResourceGroupConfiguration"])["location"] != "eastus" {
		t.Fatalf("re-PUT store = %#v", again)
	}
	h.rejects(http.MethodPut, pub+"/artifactStores/bad", azureHNResource("eastus", map[string]any{
		"managedResourceGroupConfiguration": map[string]any{"name": "a/b"}}), "managedResourceGroupConfiguration.name must be")
	h.rejects(http.MethodPut, pub+"/artifactStores/bad", azureHNResource("eastus", map[string]any{
		"managedResourceGroupConfiguration": map[string]any{"name": "mrg."}}), "managedResourceGroupConfiguration.name must be")
	h.rejects(http.MethodPut, pub+"/artifactStores/bad", azureHNResource("eastus", map[string]any{
		"managedResourceGroupConfiguration": map[string]any{"location": 5}}), "location must be a string")
	h.rejects(http.MethodPut, pub+"/artifactStores/sa", azureHNResource("eastus", map[string]any{
		"managedResourceGroupConfiguration": map[string]any{"location": "westus"}}), "managedResourceGroupConfiguration can't be changed")

	manifest := pub + "/artifactStores/acr/artifactManifests/cnf"
	artifacts := []any{map[string]any{"artifactName": "nginx", "artifactType": "OCIArtifact", "artifactVersion": "1.0.0"}}
	if props := h.props(http.MethodPut, manifest, azureHNResource("eastus", map[string]any{"artifacts": artifacts, "artifactManifestState": "Uploaded"}), http.StatusCreated); props["artifactManifestState"] != "Uploading" {
		t.Fatalf("new manifest = %#v", props)
	}
	h.rejects(http.MethodPut, manifest+"x", azureHNResource("eastus", map[string]any{"artifacts": []any{map[string]any{"artifactName": "disk", "artifactType": "VhdImageFile", "artifactVersion": "1-0-0"}}}), "can't hold the VhdImageFile")
	h.rejects(http.MethodPut, manifest+"x", azureHNResource("eastus", map[string]any{"artifacts": append(artifacts, artifacts[0])}), "twice")
	h.rejects(http.MethodPut, manifest+"x", azureHNResource("eastus", map[string]any{"artifacts": []any{map[string]any{"artifactName": "a", "artifactType": "Zip", "artifactVersion": "1"}}}), "artifactType must be one of")
	h.rejects(http.MethodPut, pub+"/artifactStores/acr/artifactManifests/"+strings.Repeat("m", 65), azureHNResource("eastus", nil), "at most 64")

	cred := h.call(http.MethodPost, manifest+"/listCredential", nil, http.StatusOK)
	token := azureARMString(cred["acrToken"])
	if cred["credentialType"] != "AzureContainerRegistryScopedToken" || len(token) != 48 || !strings.HasSuffix(azureARMString(cred["acrServerUrl"]), ".azurecr.io") ||
		len(cred["repositories"].([]any)) != 1 || cred["repositories"].([]any)[0] != "nginx" {
		t.Fatalf("acr credential = %#v", cred)
	}
	if again := h.call(http.MethodPost, manifest+"/listCredential", nil, http.StatusOK); again["acrToken"] == token {
		t.Fatal("credential not regenerated")
	}
	if strings.Contains(string(mustBody(t, h.do(http.MethodGet, manifest+azureHNTestAPI, ""))), token) {
		t.Fatal("token stored on the manifest")
	}
	vhd := pub + "/artifactStores/sa/artifactManifests/vnf"
	h.call(http.MethodPut, vhd, azureHNResource("eastus", map[string]any{"artifacts": []any{map[string]any{"artifactName": "ubuntu-vhd", "artifactType": "VhdImageFile", "artifactVersion": "1-0-0"}}}), http.StatusCreated)
	h.rejects(http.MethodPut, pub+"/artifactStores/sa/artifactManifests/arm", azureHNResource("eastus", map[string]any{"artifacts": []any{map[string]any{"artifactName": "t", "artifactType": "ArmTemplate", "artifactVersion": "1.0.0"}}}), "can't hold the ArmTemplate")
	cred = h.call(http.MethodPost, vhd+"/listCredential", nil, http.StatusOK)
	containers := cred["containerCredentials"].([]any)
	if cred["credentialType"] != "AzureStorageAccountToken" || cred["storageAccountId"] != sa["storageResourceId"] || len(containers) != 1 ||
		!strings.Contains(azureARMString(azureARMObject(containers[0])["containerSasUri"]), "&sig=") {
		t.Fatalf("storage credential = %#v", cred)
	}

	// Artifacts whose names differ only in punctuation get distinct containers.
	twins := pub + "/artifactStores/sa/artifactManifests/twins"
	h.call(http.MethodPut, twins, azureHNResource("eastus", map[string]any{"artifacts": []any{
		map[string]any{"artifactName": "app", "artifactType": "VhdImageFile", "artifactVersion": "1-0-0"},
		map[string]any{"artifactName": "app", "artifactType": "VhdImageFile", "artifactVersion": "10-0"},
		map[string]any{"artifactName": "a-pp", "artifactType": "VhdImageFile", "artifactVersion": "1-0-0"},
		map[string]any{"artifactName": "___", "artifactType": "VhdImageFile", "artifactVersion": "1"},
	}}), http.StatusCreated)
	seen := map[string]bool{}
	for _, c := range h.call(http.MethodPost, twins+"/listCredential", nil, http.StatusOK)["containerCredentials"].([]any) {
		name := azureARMString(azureARMObject(c)["containerName"])
		if seen[name] || strings.HasPrefix(name, "-") {
			t.Fatalf("bad or duplicate container %q", name)
		}
		seen[name] = true
	}
	h.rejects(http.MethodPost, manifest+"/updateState", map[string]any{"artifactManifestState": "Succeeded"}, "must be Uploaded")
	if out := h.call(http.MethodPost, manifest+"/updateState", map[string]any{"artifactManifestState": "Uploaded"}, http.StatusOK); out["artifactManifestState"] != "Uploaded" {
		t.Fatalf("updateState = %#v", out)
	}
	h.rejects(http.MethodPut, manifest, azureHNResource("eastus", map[string]any{"artifacts": []any{}}), "immutable")
	h.call(http.MethodPut, manifest, azureHNResource("eastus", map[string]any{"artifacts": artifacts}), http.StatusOK)
	if out := h.call(http.MethodPatch, manifest, map[string]any{"tags": map[string]any{"env": "prod"}}, http.StatusOK); azureARMObject(out["properties"])["artifactManifestState"] != "Uploaded" || azureARMObject(out["tags"])["env"] != "prod" {
		t.Fatalf("patched manifest = %#v", out)
	}
	h.rejects(http.MethodPatch, manifest, map[string]any{"properties": map[string]any{}}, "can't be updated")

	h.call(http.MethodDelete, pub, nil, http.StatusOK)
	h.call(http.MethodGet, manifest, nil, http.StatusNotFound)
}

func TestAzureHybridNetworkDefinitionVersions(t *testing.T) {
	t.Parallel()
	h := newAzureHNTest(t)
	h.publish("contoso")
	pub := h.base + "publishers/contoso"
	h.call(http.MethodPut, pub+"/networkFunctionDefinitionGroups/nginx", azureHNResource("eastus", map[string]any{"description": "nginx"}), http.StatusCreated)
	nfdv := pub + "/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions/"
	def := h.cnfDefinition("contoso")
	h.rejects(http.MethodPut, nfdv+"1.0", def, "is invalid")
	h.publish("fabrikam")
	h.rejects(http.MethodPut, nfdv+"9.0.0", h.cnfDefinition("fabrikam"), "belongs to another publisher")
	h.call(http.MethodPut, pub+"/networkFunctionDefinitionGroups/missing/networkFunctionDefinitionVersions/1.0.0", def, http.StatusNotFound)
	if props := h.props(http.MethodPut, nfdv+"1.0.0-beta.1+build.5", def, http.StatusCreated); props["versionState"] != "Preview" {
		t.Fatalf("new version = %#v", props)
	}

	edit := func(change func(props, app map[string]any)) map[string]any {
		raw, _ := json.Marshal(h.cnfDefinition("contoso"))
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		props := out["properties"].(map[string]any)
		change(props, props["networkFunctionTemplate"].(map[string]any)["networkFunctionApplications"].([]any)[0].(map[string]any))
		return out
	}
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(_, app map[string]any) {
		app["artifactProfile"].(map[string]any)["artifactStore"] = map[string]any{"id": h.arm + "publishers/contoso/artifactStores/nope"}
	}), "artifact store")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(_, app map[string]any) {
		app["artifactProfile"].(map[string]any)["artifactStore"] = map[string]any{"id": h.arm + "publishers/contoso/artifactStores/sa"}
	}), "can't hold the HelmPackage")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(_, app map[string]any) { app["artifactType"] = "VhdImageFile" }), "artifactType must be one of HelmPackage")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(props, _ map[string]any) { props["networkFunctionType"] = "VirtualNetworkFunction" }), "nfviType must be AzureCore or AzureOperatorNexus")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(props, _ map[string]any) { props["deployParameters"] = "{not json" }), "well-formed JSON")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(_, app map[string]any) {
		app["dependsOnProfile"] = map[string]any{"installDependsOn": []any{"redis"}}
	}), "names redis")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(props, app map[string]any) {
		template := props["networkFunctionTemplate"].(map[string]any)
		template["networkFunctionApplications"] = []any{app, map[string]any{"name": "NGINX", "artifactType": "HelmPackage"}}
	}), "two items named NGINX")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(_, app map[string]any) { delete(app["artifactProfile"].(map[string]any), "helmArtifactProfile") }), "helmPackageName is required")
	h.rejects(http.MethodPut, nfdv+"2.0.0", edit(func(_, app map[string]any) {
		app["deployParametersMappingRuleProfile"] = map[string]any{"applicationEnablement": "Sometimes"}
	}), "applicationEnablement")

	vnf := azureHNResource("eastus", map[string]any{"networkFunctionType": "VirtualNetworkFunction", "networkFunctionTemplate": map[string]any{
		"nfviType": "AzureCore", "networkFunctionApplications": []any{map[string]any{
			"name": "template", "artifactType": "ArmTemplate", "artifactProfile": map[string]any{
				"artifactStore":           map[string]any{"id": h.arm + "publishers/contoso/artifactStores/acr"},
				"templateArtifactProfile": map[string]any{"templateName": "vm", "templateVersion": "1.0.0"},
			}}}}})
	h.rejects(http.MethodPut, nfdv+"3.0.0", vnf, "needs at least one VhdImageFile and one ArmTemplate")

	version := nfdv + "1.0.0-beta.1+build.5"
	h.rejects(http.MethodPost, version+"/updateState", map[string]any{"versionState": "Preview"}, "must be Active or Deprecated")
	if out := h.call(http.MethodPost, version+"/updateState", map[string]any{"versionState": "Active"}, http.StatusOK); out["versionState"] != "Active" {
		t.Fatalf("updateState = %#v", out)
	}
	h.call(http.MethodPut, version, def, http.StatusOK)
	h.rejects(http.MethodPut, version, edit(func(props, _ map[string]any) { props["description"] = "changed" }), "immutable")
	h.call(http.MethodPatch, version, map[string]any{"tags": map[string]any{"release": "ga"}}, http.StatusOK)
	if props := h.props(http.MethodGet, version, nil, http.StatusOK); props["versionState"] != "Active" {
		t.Fatalf("after patch = %#v", props)
	}
	h.call(http.MethodPost, version+"/updateState", map[string]any{"versionState": "Deprecated"}, http.StatusOK)
	h.call(http.MethodPost, version+"/updateState", map[string]any{"versionState": "Active"}, http.StatusOK)
	if items := h.call(http.MethodGet, pub+"/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions", nil, http.StatusOK)["value"].([]any); len(items) != 1 {
		t.Fatalf("versions = %d", len(items))
	}

	cgs := pub + "/configurationGroupSchemas/vm-config"
	h.rejects(http.MethodPut, cgs, azureHNResource("eastus", map[string]any{"schemaDefinition": "[1]"}), "JSON object")
	h.call(http.MethodPut, cgs, azureHNResource("eastus", map[string]any{"schemaDefinition": `{"type":"object"}`}), http.StatusCreated)

	h.call(http.MethodPut, pub+"/networkServiceDesignGroups/edge", azureHNResource("eastus", nil), http.StatusCreated)
	nsdv := pub + "/networkServiceDesignGroups/edge/networkServiceDesignVersions/1.0.0"
	design := func(schema string) map[string]any {
		return azureHNResource("eastus", map[string]any{
			"configurationGroupSchemaReferences": map[string]any{"vm": map[string]any{"id": h.arm + "publishers/contoso/configurationGroupSchemas/" + schema}},
			"nfvisFromSite":                      map[string]any{"core": map[string]any{"name": "edge-core", "type": "AzureCore"}},
			"resourceElementTemplates": []any{map[string]any{"name": "nf", "type": "NetworkFunctionDefinition", "configuration": map[string]any{
				"templateType": "ArmTemplate", "parameterValues": `{"location":"{configurationparameters('vm').location}"}`,
				"artifactProfile": map[string]any{"artifactName": "nf-template", "artifactVersion": "1.0.0", "artifactStoreReference": map[string]any{"id": h.arm + "publishers/contoso/artifactStores/acr"}},
			}}},
		})
	}
	h.rejects(http.MethodPut, nsdv, design("missing"), "configuration group schema")
	bad := design("vm-config")
	bad["properties"].(map[string]any)["nfvisFromSite"] = map[string]any{"core": map[string]any{"name": "edge-core", "type": "Mainframe"}}
	h.rejects(http.MethodPut, nsdv, bad, "type must be one of")
	if props := h.props(http.MethodPut, nsdv, design("vm-config"), http.StatusCreated); props["versionState"] != "Preview" {
		t.Fatalf("design = %#v", props)
	}
}
