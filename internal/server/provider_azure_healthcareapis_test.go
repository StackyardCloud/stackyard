package server

import (
	"net/http"
	"testing"
)

const azureHealthcareTestAPIVersion = "?api-version=2024-03-31"

func TestAzureHealthcareLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-health")
	ws := sub + "/resourceGroups/rg-health/providers/Microsoft.HealthcareApis/workspaces/clinic"

	resp := do(http.MethodPut, ws+azureHealthcareTestAPIVersion, `{"location":"eastus","properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["publicNetworkAccess"]; got != "Enabled" {
		t.Fatalf("expected the network default, got %v", got)
	}

	fhir := ws + "/fhirservices/records"
	resp = do(http.MethodPut, fhir+azureHealthcareTestAPIVersion, `{"location":"eastus","kind":"fhir-R4","properties":{"authenticationConfiguration":{"authority":"https://login.microsoftonline.com/tenant"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	auth := providerContractJSONMap(t, resp)["properties"].(map[string]any)["authenticationConfiguration"].(map[string]any)
	if auth["audience"] != "https://clinic-records.fhir.azurehealthcareapis.com" || auth["authority"] != "https://login.microsoftonline.com/tenant" {
		t.Fatalf("unexpected FHIR authentication: %#v", auth)
	}
	// A client audience is kept.
	resp = do(http.MethodPut, fhir+azureHealthcareTestAPIVersion, `{"location":"eastus","kind":"fhir-R4","properties":{"authenticationConfiguration":{"audience":"https://custom"}}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["authenticationConfiguration"].(map[string]any)["audience"]; got != "https://custom" {
		t.Fatalf("expected the client audience kept, got %v", got)
	}

	resp = do(http.MethodPut, ws+"/dicomservices/imaging"+azureHealthcareTestAPIVersion, `{"location":"eastus","properties":{"serviceUrl":"https://forged"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["serviceUrl"]; got != "https://clinic-imaging.dicom.azurehealthcareapis.com" {
		t.Fatalf("unexpected DICOM URL %v", got)
	}

	connector := ws + "/iotconnectors/devices"
	azureResourcesExpectStatus(t, do(http.MethodPut, connector+azureHealthcareTestAPIVersion, `{"location":"eastus","properties":{"ingestionEndpointConfiguration":{"eventHubName":"telemetry","consumerGroup":"medtech","fullyQualifiedEventHubNamespace":"clinic.servicebus.windows.net"},"deviceMapping":{"content":{"templateType":"CollectionContent","template":[]}}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, connector+"/fhirdestinations/records"+azureHealthcareTestAPIVersion, `{"location":"eastus","properties":{"resourceIdentityResolutionType":"Create","fhirServiceResourceId":"`+fhir+`","fhirMapping":{"content":{"templateType":"CollectionFhirTemplate","template":[]}}}}`), http.StatusCreated)

	azureResourcesExpectStatus(t, do(http.MethodDelete, ws+azureHealthcareTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, connector+"/fhirdestinations/records"+azureHealthcareTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureHealthcareValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-health")
	ws := sub + "/resourceGroups/rg-health/providers/Microsoft.HealthcareApis/workspaces/clinic"
	azureResourcesExpectStatus(t, do(http.MethodPut, ws+azureHealthcareTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, ws+"/iotconnectors/devices"+azureHealthcareTestAPIVersion, `{"location":"eastus","properties":{"ingestionEndpointConfiguration":{"eventHubName":"t","consumerGroup":"c","fullyQualifiedEventHubNamespace":"n"},"deviceMapping":{"content":{}}}}`), http.StatusCreated)
	cases := []struct{ name, path, body, code string }{
		{"workspace name", sub + "/resourceGroups/rg-health/providers/Microsoft.HealthcareApis/workspaces/Clinic-1", `{"location":"eastus","properties":{}}`, "InvalidResourceName"},
		{"service name", ws + "/fhirservices/-records", `{"location":"eastus","kind":"fhir-R4","properties":{}}`, "InvalidResourceName"},
		{"fhir kind", ws + "/fhirservices/records", `{"location":"eastus","kind":"fhir","properties":{}}`, "InvalidRequestContent"},
		{"fhir authority", ws + "/fhirservices/records", `{"location":"eastus","kind":"fhir-R4","properties":{"authenticationConfiguration":{"authority":"http://login"}}}`, "InvalidRequestContent"},
		{"dicom network", ws + "/dicomservices/imaging", `{"location":"eastus","properties":{"publicNetworkAccess":"Open"}}`, "InvalidRequestContent"},
		{"connector event hub", ws + "/iotconnectors/devices2", `{"location":"eastus","properties":{"deviceMapping":{"content":{}}}}`, "InvalidRequestContent"},
		{"connector mapping", ws + "/iotconnectors/devices2", `{"location":"eastus","properties":{"ingestionEndpointConfiguration":{"eventHubName":"t","consumerGroup":"c","fullyQualifiedEventHubNamespace":"n"}}}`, "InvalidRequestContent"},
		{"destination resolution", ws + "/iotconnectors/devices/fhirdestinations/rec", `{"location":"eastus","properties":{"resourceIdentityResolutionType":"Guess","fhirServiceResourceId":"` + ws + `/fhirservices/records","fhirMapping":{"content":{}}}}`, "InvalidRequestContent"},
		{"destination service", ws + "/iotconnectors/devices/fhirdestinations/rec", `{"location":"eastus","properties":{"resourceIdentityResolutionType":"Create","fhirServiceResourceId":"` + ws + `/dicomservices/imaging","fhirMapping":{"content":{}}}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureHealthcareTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
}
