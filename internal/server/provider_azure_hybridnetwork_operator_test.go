package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const azureHNTestSchema = `{"type":"object","properties":{"network":{"type":"object","properties":{"vlan":{"type":"integer"}},"required":["vlan"]}},"required":["network"]}`

// design publishes contoso with a CNF definition (version 1.0.0), a
// configuration group schema and a design (1.0.0) deploying to the
// AzureCore NFVI edge-core.
func (h *azureHNTest) design() {
	h.t.Helper()
	h.publish("contoso")
	pub := h.base + "publishers/contoso"
	h.call(http.MethodPut, pub+"/networkFunctionDefinitionGroups/nginx", azureHNResource("eastus", nil), http.StatusCreated)
	h.call(http.MethodPut, pub+"/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions/1.0.0", h.cnfDefinition("contoso"), http.StatusCreated)
	h.call(http.MethodPut, pub+"/configurationGroupSchemas/net", azureHNResource("eastus", map[string]any{"schemaDefinition": azureHNTestSchema}), http.StatusCreated)
	h.call(http.MethodPut, pub+"/networkServiceDesignGroups/edge", azureHNResource("eastus", nil), http.StatusCreated)
	h.call(http.MethodPut, pub+"/networkServiceDesignGroups/edge/networkServiceDesignVersions/1.0.0", azureHNResource("eastus", map[string]any{
		"configurationGroupSchemaReferences": map[string]any{"net": map[string]any{"id": h.arm + "publishers/contoso/configurationGroupSchemas/net"}},
		"nfvisFromSite":                      map[string]any{"core": map[string]any{"name": "edge-core", "type": "AzureCore"}},
	}), http.StatusCreated)
}

func azureHNTestNF(nfdv, idType string, props map[string]any) map[string]any {
	out := map[string]any{
		azureHNNFDVRef:        map[string]any{"id": nfdv, "idType": idType},
		"nfviType":            "AzureArcKubernetes",
		"nfviId":              "/subscriptions/sub/resourceGroups/edge/providers/Microsoft.ExtendedLocation/customLocations/edge-cl",
		"configurationType":   "Open",
		"deploymentValues":    `{"releaseName":"nginx"}`,
		"allowSoftwareUpdate": false,
	}
	for k, v := range props {
		out[k] = v
	}
	return azureHNResource("eastus", out)
}

func TestAzureHybridNetworkConfigurationGroupValues(t *testing.T) {
	t.Parallel()
	h := newAzureHNTest(t)
	h.design()
	schema := h.arm + "publishers/contoso/configurationGroupSchemas/net"
	cgv := h.base + "configurationGroupValues/"
	value := func(kind, idType, field, text string) map[string]any {
		return azureHNResource("eastus", map[string]any{"configurationType": kind, field: text,
			azureHNCGSRef: map[string]any{"id": schema, "idType": idType}})
	}
	open := h.props(http.MethodPut, cgv+"open", value("Open", "Open", "configurationValue", `{"network":{"vlan":10}}`), http.StatusCreated)
	if open["publisherName"] != "contoso" || open["publisherScope"] != "Private" || open["configurationGroupSchemaName"] != "net" ||
		open["configurationGroupSchemaOfferingLocation"] != "eastus" || azureARMObject(open[azureHNCGSRef])["id"] != schema {
		t.Fatalf("open value = %#v", open)
	}
	h.rejects(http.MethodPut, cgv+"bad", value("Open", "Open", "configurationValue", `{"network":{}}`), "lacks network.vlan")
	h.rejects(http.MethodPut, cgv+"bad", value("Open", "Open", "configurationValue", `{}`), "lacks network,")
	h.rejects(http.MethodPut, cgv+"bad", value("Open", "Open", "secretConfigurationValue", `{}`), "isn't allowed with configurationType Open")
	h.rejects(http.MethodPut, cgv+"bad", value("Shared", "Open", "configurationValue", `{}`), "configurationType must be Open or Secret")
	h.rejects(http.MethodPut, cgv+"bad", value("Open", "Hidden", "configurationValue", `{}`), "idType must be one of")
	h.rejects(http.MethodPut, cgv+"bad", azureHNResource("eastus", map[string]any{"configurationType": "Open", "configurationValue": `{}`}), "configurationGroupSchemaResourceReference.id is required")
	missing := value("Open", "Open", "configurationValue", `{"network":{"vlan":1}}`)
	missing["properties"].(map[string]any)[azureHNCGSRef] = map[string]any{"id": schema + "x"}
	h.rejects(http.MethodPut, cgv+"bad", missing, "was not found")
	// Secret values are checked against the schema too.
	h.rejects(http.MethodPut, cgv+"secret", value("Secret", "Secret", "secretConfigurationValue", `{"network":{}}`), "lacks network.vlan")

	secret := value("Secret", "Secret", "secretConfigurationValue", `{"network":{"vlan":"hunter2-vlan"}}`)
	for _, out := range []map[string]any{
		h.call(http.MethodPut, cgv+"secret", secret, http.StatusCreated),
		h.call(http.MethodPatch, cgv+"secret", map[string]any{"tags": map[string]any{"a": "b"}}, http.StatusOK),
		h.call(http.MethodGet, cgv+"secret", nil, http.StatusOK),
	} {
		if raw, _ := json.Marshal(out); strings.Contains(string(raw), "hunter2") || strings.Contains(string(raw), schema) {
			t.Fatalf("secret value response leaks: %s", raw)
		}
	}
	h.call(http.MethodPatch, cgv+"open", map[string]any{"tags": map[string]any{"a": "b"}}, http.StatusOK)
	for _, path := range []string{h.base + "configurationGroupValues", h.sub + "/providers/Microsoft.HybridNetwork/configurationGroupValues"} {
		body := string(mustBody(t, h.do(http.MethodGet, path+azureHNTestAPI, "")))
		if strings.Contains(body, "hunter2") || strings.Count(body, schema) != 1 || !strings.Contains(body, `"idType":"Secret"`) {
			t.Fatalf("GET %s leaks a secret or loses the open reference: %s", path, body)
		}
	}
	if ref := azureARMObject(h.props(http.MethodGet, cgv+"open", nil, http.StatusOK)[azureHNCGSRef]); ref["id"] != schema || ref["idType"] != "Open" {
		t.Fatalf("open reference after PATCH = %#v", ref)
	}
	// A PUT that turns an Open reference Secret drops its ID.
	h.call(http.MethodPut, cgv+"open", value("Open", "Secret", "configurationValue", `{"network":{"vlan":11}}`), http.StatusOK)
	if ref := azureARMObject(h.props(http.MethodGet, cgv+"open", nil, http.StatusOK)[azureHNCGSRef]); ref["id"] != nil {
		t.Fatalf("secret reference = %#v", ref)
	}
	// A Deprecated schema can't take new values.
	h.call(http.MethodPost, h.base+"publishers/contoso/configurationGroupSchemas/net/updateState", map[string]any{"versionState": "Deprecated"}, http.StatusOK)
	h.rejects(http.MethodPut, cgv+"late", value("Open", "Open", "configurationValue", `{"network":{"vlan":12}}`), "Deprecated and can't be deployed")
}

func TestAzureHybridNetworkNetworkFunctions(t *testing.T) {
	t.Parallel()
	h := newAzureHNTest(t)
	h.design()
	nfdv := h.arm + "publishers/contoso/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions/1.0.0"
	nf := h.base + "networkFunctions/nginx-01"
	created := h.props(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", nil), http.StatusCreated)
	if created["publisherName"] != "contoso" || created["networkFunctionDefinitionGroupName"] != "nginx" || created["networkFunctionDefinitionVersion"] != "1.0.0" ||
		created["networkFunctionDefinitionOfferingLocation"] != "eastus" || azureARMObject(created[azureHNNFDVRef])["id"] != nfdv {
		t.Fatalf("network function = %#v", created)
	}
	h.rejects(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", map[string]any{"deploymentValues": `{"namespace":"x"}`}), "lacks releaseName")
	h.rejects(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", map[string]any{"nfviType": "AzureCore"}), "must be AzureArcKubernetes")
	h.rejects(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", map[string]any{"nfviId": "/subscriptions/sub/resourceGroups/edge"}), "custom location ID")
	h.rejects(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", map[string]any{"roleOverrideValues": []any{`{"name":"redis"}`}}), "overrides redis")
	h.rejects(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", map[string]any{"roleOverrideValues": []any{`{}`}}), "must name the application")
	h.call(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", map[string]any{"roleOverrideValues": []any{`{"name":"NGINX","deployParametersMappingRuleProfile":{}}`}}), http.StatusOK)
	h.rejects(http.MethodPut, nf, azureHNTestNF(nfdv+"9", "Open", nil), "was not found")
	h.rejects(http.MethodPut, nf, azureHNTestNF(h.arm+"publishers/contoso/configurationGroupSchemas/net", "Open", nil), "was not found")

	// Secret deployment values and Secret references never come back.
	secret := azureHNTestNF(nfdv, "Secret", map[string]any{"configurationType": "Secret", "secretDeploymentValues": `{"releaseName":"s3cr3t-release"}`})
	delete(secret["properties"].(map[string]any), "deploymentValues")
	for _, out := range []map[string]any{
		h.call(http.MethodPut, h.base+"networkFunctions/secret-nf", secret, http.StatusCreated),
		h.call(http.MethodPatch, h.base+"networkFunctions/secret-nf", map[string]any{"tags": map[string]any{"a": "b"}}, http.StatusOK),
		h.call(http.MethodGet, h.base+"networkFunctions/secret-nf", nil, http.StatusOK),
	} {
		if raw, _ := json.Marshal(out); strings.Contains(string(raw), "s3cr3t") || strings.Contains(string(raw), nfdv) {
			t.Fatalf("secret network function response leaks: %s", raw)
		}
	}
	if body := string(mustBody(t, h.do(http.MethodGet, h.sub+"/providers/Microsoft.HybridNetwork/networkFunctions"+azureHNTestAPI, ""))); strings.Contains(body, "s3cr3t") || strings.Count(body, nfdv) != 1 {
		t.Fatalf("network function list leaks a secret: %s", body)
	}

	h.rejects(http.MethodPost, nf+"/executeRequest", map[string]any{"serviceEndpoint": "svc"}, "requestMetadata is required")
	h.rejects(http.MethodPost, nf+"/executeRequest", map[string]any{"serviceEndpoint": "svc", "requestMetadata": map[string]any{"relativePath": "/v1", "httpMethod": "Head"}}, "httpMethod")
	h.rejects(http.MethodPost, nf+"/executeRequest", map[string]any{"serviceEndpoint": "svc", "requestMetadata": map[string]any{"relativePath": "/v1", "httpMethod": "Post", "serializedBody": "{"}}, "serializedBody")
	h.rejects(http.MethodPost, nf+"/executeRequest", map[string]any{"serviceEndpoint": "svc", "requestMetadata": map[string]any{"relativePath": "https://evil.example/x", "httpMethod": "Get"}}, "relativePath")
	h.call(http.MethodPost, nf+"/executeRequest", map[string]any{"serviceEndpoint": "svc", "requestMetadata": map[string]any{"relativePath": "/v1", "httpMethod": "Post", "serializedBody": "[1,2]"}}, http.StatusOK)
	resp := h.do(http.MethodPost, nf+"/executeRequest"+azureHNTestAPI, `{"serviceEndpoint":"nginx.svc","requestMetadata":{"relativePath":"/reload","httpMethod":"Post","serializedBody":"{\"force\":true}","apiVersion":"v1"}}`)
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || len(body) != 0 {
		t.Fatalf("executeRequest = %d %s", resp.StatusCode, body)
	}

	// An Active version deploys only artifacts in Uploaded manifests; a
	// Deprecated one doesn't deploy at all.
	version := h.base + "publishers/contoso/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions/1.0.0"
	h.call(http.MethodPost, version+"/updateState", map[string]any{"versionState": "Active"}, http.StatusOK)
	// An Uploaded template named like the helm package doesn't count.
	tmpl := h.base + "publishers/contoso/artifactStores/acr/artifactManifests/tmpl"
	h.call(http.MethodPut, tmpl, azureHNResource("eastus", map[string]any{"artifacts": []any{
		map[string]any{"artifactName": "nginx", "artifactType": "ArmTemplate", "artifactVersion": "1.0.0"}}}), http.StatusCreated)
	h.call(http.MethodPost, tmpl+"/updateState", map[string]any{"artifactManifestState": "Uploaded"}, http.StatusOK)
	h.rejects(http.MethodPut, h.base+"networkFunctions/nf2", azureHNTestNF(nfdv, "Open", nil), "Uploaded artifact manifest")
	h.call(http.MethodPost, h.base+"publishers/contoso/artifactStores/acr/artifactManifests/cnf/updateState", map[string]any{"artifactManifestState": "Uploaded"}, http.StatusOK)
	h.call(http.MethodPut, h.base+"networkFunctions/nf2", azureHNTestNF(nfdv, "Open", nil), http.StatusCreated)
	h.call(http.MethodPost, version+"/updateState", map[string]any{"versionState": "Deprecated"}, http.StatusOK)
	h.rejects(http.MethodPut, h.base+"networkFunctions/nf3", azureHNTestNF(nfdv, "Open", nil), "Deprecated and can't be deployed")
	// Existing network functions keep working: a tags update needs no checks.
	h.call(http.MethodPatch, nf, map[string]any{"tags": map[string]any{"a": "b"}}, http.StatusOK)
	if ref := azureARMObject(h.props(http.MethodGet, nf, nil, http.StatusOK)[azureHNNFDVRef]); ref["id"] != nfdv {
		t.Fatalf("reference after PATCH = %#v", ref)
	}
}

func TestAzureHybridNetworkPreviewIsSubscriptionScoped(t *testing.T) {
	t.Parallel()
	h := newAzureHNTest(t)
	h.design()
	nfdv := h.arm + "publishers/contoso/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions/1.0.0"
	otherSub := "/azure/subscriptions/11111111-2222-3333-4444-555555555555"
	h.call(http.MethodPut, otherSub+"/resourcegroups/rg-op", map[string]any{"location": "eastus"}, http.StatusCreated)
	nf := otherSub + "/resourceGroups/rg-op/providers/Microsoft.HybridNetwork/networkFunctions/nf"
	h.rejects(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", nil), "in Preview")
	h.call(http.MethodPost, h.base+"publishers/contoso/artifactStores/acr/artifactManifests/cnf/updateState", map[string]any{"artifactManifestState": "Uploaded"}, http.StatusOK)
	h.call(http.MethodPost, h.base+"publishers/contoso/networkFunctionDefinitionGroups/nginx/networkFunctionDefinitionVersions/1.0.0/updateState", map[string]any{"versionState": "Active"}, http.StatusOK)
	h.call(http.MethodPut, nf, azureHNTestNF(nfdv, "Open", nil), http.StatusCreated)
}

func TestAzureHybridNetworkSiteNetworkServices(t *testing.T) {
	t.Parallel()
	h := newAzureHNTest(t)
	h.design()
	site := func(location string, nfvis ...any) map[string]any {
		return azureHNResource(location, map[string]any{"nfvis": nfvis})
	}
	core := map[string]any{"nfviType": "AzureCore", "name": "edge-core", "location": "eastus"}
	h.rejects(http.MethodPut, h.base+"sites/bad", site("eastus", core, core), "two NFVIs named edge-core")
	h.rejects(http.MethodPut, h.base+"sites/bad", site("eastus", map[string]any{"nfviType": "AzureCore", "name": "x"}), "location is required")
	h.rejects(http.MethodPut, h.base+"sites/bad", site("eastus", map[string]any{"nfviType": "AzureArcKubernetes", "name": "x"}), "custom location ID")
	h.rejects(http.MethodPut, h.base+"sites/bad", site("eastus", map[string]any{"nfviType": "Mainframe", "name": "x"}), "nfviType must be one of")
	h.call(http.MethodPut, h.base+"sites/edge", site("eastus", core), http.StatusCreated)
	h.call(http.MethodPut, h.base+"sites/west", site("westus", core), http.StatusCreated)
	h.call(http.MethodPut, h.base+"sites/arc", site("eastus", map[string]any{"nfviType": "AzureArcKubernetes", "name": "edge-core",
		"customLocationReference": map[string]any{"id": "/subscriptions/s/resourceGroups/r/providers/Microsoft.ExtendedLocation/customLocations/cl"}}), http.StatusCreated)
	schema := h.arm + "publishers/contoso/configurationGroupSchemas/net"
	h.call(http.MethodPut, h.base+"configurationGroupValues/net-east", azureHNResource("westus", map[string]any{
		"configurationType": "Open", "configurationValue": `{"network":{"vlan":7}}`, azureHNCGSRef: map[string]any{"id": schema}}), http.StatusCreated)
	h.call(http.MethodPut, h.base+"publishers/contoso/configurationGroupSchemas/other", azureHNResource("eastus", map[string]any{"schemaDefinition": "{}"}), http.StatusCreated)
	h.call(http.MethodPut, h.base+"configurationGroupValues/wrong", azureHNResource("eastus", map[string]any{
		"configurationType": "Open", "configurationValue": `{}`, azureHNCGSRef: map[string]any{"id": h.arm + "publishers/contoso/configurationGroupSchemas/other"}}), http.StatusCreated)

	// A same-named publisher and schema in another region make a different schema.
	h.call(http.MethodPut, h.sub+"/resourcegroups/rg-west", map[string]any{"location": "westus"}, http.StatusCreated)
	west := strings.TrimPrefix(h.sub, "/azure") + "/resourceGroups/rg-west/providers/Microsoft.HybridNetwork/"
	h.call(http.MethodPut, "/azure"+west+"publishers/contoso", azureHNResource("westus", nil), http.StatusCreated)
	h.call(http.MethodPut, "/azure"+west+"publishers/contoso/configurationGroupSchemas/net", azureHNResource("westus", map[string]any{"schemaDefinition": "{}"}), http.StatusCreated)
	h.call(http.MethodPut, h.base+"configurationGroupValues/net-west", azureHNResource("eastus", map[string]any{
		"configurationType": "Open", "configurationValue": `{}`, azureHNCGSRef: map[string]any{"id": west + "publishers/contoso/configurationGroupSchemas/net"}}), http.StatusCreated)

	nsdv := h.arm + "publishers/contoso/networkServiceDesignGroups/edge/networkServiceDesignVersions/1.0.0"
	sns := func(siteName string, values map[string]any) map[string]any {
		return map[string]any{"location": "eastus", "sku": map[string]any{"name": "Standard"}, "properties": map[string]any{
			"siteReference": map[string]any{"id": h.arm + "sites/" + siteName},
			azureHNNSDVRef:  map[string]any{"id": nsdv, "idType": "Open"},
			"desiredStateConfigurationGroupValueReferences": values,
		}}
	}
	good := map[string]any{"net": map[string]any{"id": h.arm + "configurationGroupValues/net-east"}}
	path := h.base + "siteNetworkServices/edge-svc"
	h.rejects(http.MethodPut, path, sns("west", good), "in its own region")
	h.rejects(http.MethodPut, path, sns("missing", good), "site")
	h.rejects(http.MethodPut, path, sns("arc", good), "no AzureCore NFVI named edge-core")
	h.rejects(http.MethodPut, path, sns("edge", map[string]any{}), "needs a value for net")
	h.rejects(http.MethodPut, path, sns("edge", map[string]any{"net": good["net"], "dns": good["net"]}), "dns isn't a configuration group")
	h.rejects(http.MethodPut, path, sns("edge", map[string]any{"net": map[string]any{"id": h.arm + "configurationGroupValues/wrong"}}), "isn't made from net")
	h.rejects(http.MethodPut, path, sns("edge", map[string]any{"net": map[string]any{"id": h.arm + "configurationGroupValues/net-west"}}), "isn't made from net")
	bad := sns("edge", good)
	bad["sku"] = map[string]any{"name": "Premium"}
	h.rejects(http.MethodPut, path, bad, "sku.name")

	out := h.call(http.MethodPut, path, sns("edge", good), http.StatusCreated)
	props := azureARMObject(out["properties"])
	if azureARMObject(out["sku"])["tier"] != "Standard" || props["publisherName"] != "contoso" || props["networkServiceDesignGroupName"] != "edge" ||
		props["networkServiceDesignVersionName"] != "1.0.0" || props["lastStateNetworkServiceDesignVersionName"] != "1.0.0" ||
		azureARMObject(props["managedResourceGroupConfiguration"])["name"] == nil || azureARMObject(props[azureHNNSDVRef])["id"] != nsdv {
		t.Fatalf("site network service = %#v", out)
	}
	if items := h.call(http.MethodGet, h.sub+"/providers/Microsoft.HybridNetwork/siteNetworkServices", nil, http.StatusOK)["value"].([]any); len(items) != 1 {
		t.Fatalf("services = %d", len(items))
	}

	// An Active design needs Active schemas.
	design := h.base + "publishers/contoso/networkServiceDesignGroups/edge/networkServiceDesignVersions/1.0.0"
	h.call(http.MethodPost, design+"/updateState", map[string]any{"versionState": "Active"}, http.StatusOK)
	h.rejects(http.MethodPut, path, sns("edge", good), "must be Active too")
	h.call(http.MethodPost, h.base+"publishers/contoso/configurationGroupSchemas/net/updateState", map[string]any{"versionState": "Active"}, http.StatusOK)
	h.call(http.MethodPut, path, sns("edge", good), http.StatusOK)
	h.rejects(http.MethodPut, h.base+"publishers/contoso/configurationGroupSchemas/net", azureHNResource("eastus", map[string]any{"schemaDefinition": "{}"}), "immutable")
	h.call(http.MethodDelete, path, nil, http.StatusOK)
	h.call(http.MethodDelete, path, nil, http.StatusNoContent)
}
