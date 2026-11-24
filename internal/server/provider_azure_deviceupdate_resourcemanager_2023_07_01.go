package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

const azureDeviceUpdateResourceManager20230701APIVersion = "2023-07-01"

const aduRMNamespace = "Microsoft.DeviceUpdate"

// Microsoft.DeviceUpdate accounts and instances, served by the shared ARM
// handler. Each account gets a host name its instances serve the data plane
// (provider_azure_deviceupdate_dataplane_2022_10_01.go) under.
func init() {
	ns := aduRMNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts", ValidateName: aduRMValidateName, Validate: aduRMValidateAccount,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"hostName": strings.ToLower(rec.Name) + ".api.adu.microsoft.com", "locations": []any{map[string]any{"name": rec.Location, "role": "Primary"}},
			}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/instances", ValidateName: aduRMValidateName,
		Validate: func(payload map[string]any) string {
			hubs, _ := azureARMObject(payload["properties"])["iotHubs"].([]any)
			for _, hub := range hubs {
				if strings.TrimSpace(azureARMString(azureARMObject(hub)["resourceId"])) == "" {
					return "properties.iotHubs[].resourceId is required."
				}
			}
			return ""
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"accountName": strings.Split(rec.ID, "/")[8]}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/privateEndpointConnections", Proxy: true,
		Validate: func(payload map[string]any) string {
			state := azureARMObject(azureARMObject(payload["properties"])["privateLinkServiceConnectionState"])
			if !slices.Contains([]string{"Pending", "Approved", "Rejected"}, azureARMString(state["status"])) {
				return "properties.privateLinkServiceConnectionState.status must be Pending, Approved or Rejected."
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/privateEndpointConnectionProxies", Proxy: true,
		Validate: func(payload map[string]any) string {
			if strings.TrimSpace(azureARMString(azureARMObject(payload["remotePrivateEndpoint"])["id"])) == "" &&
				strings.TrimSpace(azureARMString(azureARMObject(azureARMObject(payload["properties"])["remotePrivateEndpoint"])["id"])) == "" {
				return "remotePrivateEndpoint.id is required."
			}
			return ""
		},
		Actions: map[string]azureARMAction{
			"validate":                        aduRMProxyCheck,
			"updateprivateendpointproperties": aduRMProxyCheck,
		},
	})
	registerAzureARMView(ns, "accounts/privateLinkResources", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/privateLinkResources/DeviceUpdate", "name": "DeviceUpdate", "type": ns + "/accounts/privateLinkResources",
			"properties": map[string]any{"groupId": "DeviceUpdate", "requiredMembers": []any{"DeviceUpdate"}, "requiredZoneNames": []any{"privatelink.api.adu.microsoft.com"}},
		}}
	})
	registerAzureARMProviderAction(ns, "checkNameAvailability", func(s *Server, payload map[string]any) (int, any) {
		name, typ := strings.TrimSpace(azureARMString(payload["name"])), azureARMString(payload["type"])
		if name == "" || !strings.EqualFold(typ, ns+"/accounts") {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name and type 'Microsoft.DeviceUpdate/accounts' are required.")
		}
		if msg := aduRMValidateName(name); msg != "" {
			return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
		}
		if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			return strings.EqualFold(rec.Type, ns+"/accounts") && strings.EqualFold(rec.Name, name)
		})) > 0 {
			return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "Account name " + name + " is already in use."}
		}
		return http.StatusOK, map[string]any{"nameAvailable": true}
	})
	registerAzureARMView(ns, "operations", azureARMCrudOperations(ns, "Device Update", "accounts", "accounts/instances", "accounts/privateEndpointConnections"))
}

var aduRMNamePattern = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)

func aduRMValidateName(name string) string {
	if len(name) < 3 || len(name) > 24 || !aduRMNamePattern.MatchString(name) {
		return "names must be 3-24 letters, digits or single hyphens between them."
	}
	return ""
}

func aduRMValidateAccount(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if v, ok := props["publicNetworkAccess"]; ok && !slices.Contains([]string{"Enabled", "Disabled"}, azureARMString(v)) {
		return "properties.publicNetworkAccess must be Enabled or Disabled."
	}
	if v, ok := props["sku"]; ok && !slices.Contains([]string{"Free", "Standard"}, azureARMString(v)) {
		return "properties.sku must be Free or Standard."
	}
	return ""
}

func aduRMProxyCheck(_ *azureARMResource, payload map[string]any) (int, any) {
	remote := azureARMObject(payload["remotePrivateEndpoint"])
	if strings.TrimSpace(azureARMString(remote["id"])) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "remotePrivateEndpoint.id is required.")
	}
	return http.StatusOK, nil
}
