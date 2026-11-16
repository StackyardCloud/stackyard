package server

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// Microsoft.HealthcareApis workspaces with their FHIR, DICOM and MedTech
// (IoT connector) services, served by the shared ARM handler. No FHIR or
// DICOM server answers at the reported URLs and no device data flows.
func init() {
	const ns = "Microsoft.HealthcareApis"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces",
		ValidateName: azureHealthcareValidateWorkspaceName,
		Validate:     azureEnumProperty("publicNetworkAccess", "Enabled", "Disabled"),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return azureHealthcareNetworkDefault(rec, map[string]any{})
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/fhirservices",
		ValidateName:       azureHealthcareValidateServiceName,
		Validate:           azureHealthcareValidateFHIR,
		ReadOnlyProperties: azureHealthcareServiceProperties("fhir"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/dicomservices",
		ValidateName:       azureHealthcareValidateServiceName,
		Validate:           azureEnumProperty("publicNetworkAccess", "Enabled", "Disabled"),
		ReadOnlyProperties: azureHealthcareServiceProperties("dicom"),
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/iotconnectors",
		ValidateName: azureHealthcareValidateServiceName,
		Validate:     azureHealthcareValidateIoTConnector,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/iotconnectors/fhirdestinations",
		ValidateName: azureHealthcareValidateServiceName,
		Validate:     azureHealthcareValidateFHIRDestination,
	})
}

var (
	azureHealthcareWorkspaceNamePattern = regexp.MustCompile(`^[a-z0-9]{3,24}$`)
	azureHealthcareServiceNamePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,22}[a-z0-9]$`)
)

func azureHealthcareValidateWorkspaceName(name string) string {
	if !azureHealthcareWorkspaceNamePattern.MatchString(name) {
		return "Workspace names must be 3-24 lowercase letters or digits."
	}
	return ""
}

func azureHealthcareValidateServiceName(name string) string {
	if !azureHealthcareServiceNamePattern.MatchString(name) {
		return "Service names must be 3-24 lowercase letters, digits or hyphens, and start and end with a letter or digit."
	}
	return ""
}

func azureHealthcareNetworkDefault(rec *azureARMResource, props map[string]any) map[string]any {
	if _, present := rec.Properties["publicNetworkAccess"]; !present {
		props["publicNetworkAccess"] = "Enabled"
	}
	return props
}

// azureHealthcareServiceProperties reports the service URL under its
// workspace, {workspace}-{service}.{kind}.azurehealthcareapis.com: DICOM
// services return it as serviceUrl, and FHIR services use it as the default
// authentication audience.
func azureHealthcareServiceProperties(kind string) func(rec *azureARMResource) map[string]any {
	return func(rec *azureARMResource) map[string]any {
		workspace := path.Base(path.Dir(path.Dir(rec.ID)))
		url := "https://" + strings.ToLower(workspace+"-"+rec.Name) + "." + kind + ".azurehealthcareapis.com"
		props := azureHealthcareNetworkDefault(rec, map[string]any{"eventState": "Disabled"})
		switch {
		case kind == "dicom":
			props["serviceUrl"] = url
		case azureARMObject(rec.Properties["authenticationConfiguration"])["audience"] == nil:
			props["authenticationConfiguration"] = map[string]any{"audience": url}
		}
		return props
	}
}

func azureHealthcareValidateFHIR(payload map[string]any) string {
	if kind := azureARMString(payload["kind"]); kind != "fhir-R4" && kind != "fhir-Stu3" {
		return "kind must be fhir-R4 or fhir-Stu3."
	}
	if msg := azureEnumProperty("publicNetworkAccess", "Enabled", "Disabled")(payload); msg != "" {
		return msg
	}
	auth := azureARMObject(azureARMObject(payload["properties"])["authenticationConfiguration"])
	if authority, present := auth["authority"]; present && !strings.HasPrefix(azureARMString(authority), "https://") {
		return "properties.authenticationConfiguration.authority must be an https URL."
	}
	return ""
}

func azureHealthcareValidateIoTConnector(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	ingestion := azureARMObject(props["ingestionEndpointConfiguration"])
	for _, key := range []string{"eventHubName", "consumerGroup", "fullyQualifiedEventHubNamespace"} {
		if strings.TrimSpace(azureARMString(ingestion[key])) == "" {
			return "properties.ingestionEndpointConfiguration." + key + " is required."
		}
	}
	if _, ok := azureARMObject(props["deviceMapping"])["content"].(map[string]any); !ok {
		return "properties.deviceMapping.content must be an object."
	}
	return ""
}

func azureHealthcareValidateFHIRDestination(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !slices.Contains([]string{"Create", "Lookup"}, azureARMString(props["resourceIdentityResolutionType"])) {
		return "properties.resourceIdentityResolutionType must be Create or Lookup."
	}
	if id := strings.ToLower(azureARMString(props["fhirServiceResourceId"])); !strings.Contains(id, "/providers/microsoft.healthcareapis/workspaces/") || !strings.Contains(id, "/fhirservices/") {
		return "properties.fhirServiceResourceId must be a FHIR service ID."
	}
	if _, ok := azureARMObject(props["fhirMapping"])["content"].(map[string]any); !ok {
		return "properties.fhirMapping.content must be an object."
	}
	return ""
}
