package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const (
	azureWorkloadsNamespace  = "Microsoft.Workloads"
	azureWorkloadsVISType    = "sapVirtualInstances"
	azureWorkloadsRegistered = "RegistrationComplete"
)

// azureWorkloadsInstanceTypes are the SAP instances of a Virtual Instance
// for SAP solutions.
var azureWorkloadsInstanceTypes = []string{"centralInstances", "databaseInstances", "applicationInstances"}

func init() {
	ns := azureWorkloadsNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureWorkloadsVISType,
		ValidateName:        azureWorkloadsVISName,
		Validate:            azureWorkloadsVISValidate,
		PatchBody:           azureWorkloadsTagsPatch(true),
		WriteOnlyProperties: azureWorkloadsVISSecrets(),
		Reconcile:           azureWorkloadsVISReconcile,
		// The system's status follows its instances whenever one is added,
		// changed or deleted; start and stop set it directly.
		ChildProperties: func(s *Server, rec *azureARMResource) map[string]any {
			if len(s.azureWorkloadsInstancesLocked(rec.ID, "")) == 0 {
				return nil
			}
			return azureWorkloadsStatus(s.azureWorkloadsSystemStatusLocked(rec.ID))
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"start": azureWorkloadsPower(true),
			"stop":  azureWorkloadsPower(false),
		},
	})
	for _, kind := range azureWorkloadsInstanceTypes {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: azureWorkloadsVISType + "/" + kind,
			Validate:         azureWorkloadsInstanceValidate,
			PatchBody:        azureWorkloadsTagsPatch(false),
			ValidateInParent: azureWorkloadsInstanceInParent,
			Reconcile:        azureWorkloadsInstanceReconcile,
			ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
				"start": azureWorkloadsPower(true),
				"stop":  azureWorkloadsPower(false),
			},
		})
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors",
		Validate:  azureWorkloadsMonitorValidate,
		PatchBody: azureWorkloadsTagsPatch(true),
		Reconcile: azureWorkloadsMonitorReconcile,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/providerInstances", Proxy: true,
		Validate:            azureWorkloadsProviderValidate,
		WriteOnlyProperties: []string{"providerSettings.dbPassword", "providerSettings.sapPassword"},
		NoPatch:             true,
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			kind := func(r *azureARMResource) any { return azureARMObject(r.Properties["providerSettings"])["providerType"] }
			if existing != nil && kind(existing) != kind(rec) {
				return nil, "properties.providerSettings.providerType can't be changed after creation."
			}
			return nil, ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "monitors/sapLandscapeMonitor", Proxy: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The SAP landscape monitor name must be 'default'."
			}
			return ""
		},
		Validate: azureWorkloadsLandscapeValidate,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			for key := range payload {
				if key != "properties" {
					return nil, nil, key + " can't be updated."
				}
			}
			return payload, nil, azureWorkloadsLandscapeValidate(payload)
		},
	})
	azureWorkloadsRegisterMetadata()
}

// azureWorkloadsVISSecrets are the passwords and private keys a VIS takes
// but never returns.
func azureWorkloadsVISSecrets() []string {
	const infra = "configuration.infrastructureConfiguration."
	paths := []string{
		"configuration.softwareConfiguration.highAvailabilitySoftwareConfiguration.fencingClientPassword",
		"configuration.softwareConfiguration.sshPrivateKey",
	}
	for _, vm := range []string{"", "centralServer.", "applicationServer.", "databaseServer."} {
		os := infra + vm + "virtualMachineConfiguration.osProfile."
		paths = append(paths, os+"adminPassword", os+"osConfiguration.sshKeyPair.privateKey")
	}
	return paths
}

// azureWorkloadsVISReconcile derives the system's state: a registered
// system, or one whose software was installed with it, is running; one
// with only infrastructure waits for software. Learn's update model takes
// only tags and identity, so the configuration can't change on PUT, except
// to add the software of a system that waits for it.
func azureWorkloadsVISReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	config := azureARMObject(props["configuration"])
	if existing == nil {
		state := azureWorkloadsRegistered
		if config["configurationType"] != "Discovery" && config["softwareConfiguration"] == nil {
			state = "SoftwareInstallationPending"
		}
		out := azureWorkloadsStatus(map[bool]string{true: "Running", false: "Unavailable"}[state == azureWorkloadsRegistered])
		out["state"] = state
		if props["managedResourceGroupConfiguration"] == nil {
			out["managedResourceGroupConfiguration"] = map[string]any{"name": "mrg-" + strings.ToLower(rec.Name) + "-" + rec.SubscriptionID}
		}
		return out, ""
	}
	old := existing.Properties
	out := map[string]any{"state": old["state"], "status": old["status"], "health": old["health"]}
	if props["managedResourceGroupConfiguration"] == nil {
		out["managedResourceGroupConfiguration"] = old["managedResourceGroupConfiguration"]
	}
	for _, key := range []string{"environment", "sapProduct", "managedResourceGroupConfiguration"} {
		if props[key] != nil && !reflect.DeepEqual(props[key], old[key]) {
			return nil, "properties." + key + " can't be changed after creation."
		}
	}
	oldConfig := azureARMObject(old["configuration"])
	if reflect.DeepEqual(config, oldConfig) {
		return out, ""
	}
	install := azureARMMerge(config, nil)
	delete(install, "softwareConfiguration")
	if old["state"] != "SoftwareInstallationPending" || config["softwareConfiguration"] == nil || !reflect.DeepEqual(install, oldConfig) {
		return nil, "properties.configuration can't be changed after creation, except to install software on a system waiting for it."
	}
	out = azureARMMerge(out, azureWorkloadsStatus("Running"))
	out["state"] = azureWorkloadsRegistered
	return out, ""
}

// azureWorkloadsStatus returns a status with the health that goes with it.
func azureWorkloadsStatus(status string) map[string]any {
	health := "Unknown"
	if status == "Running" {
		health = "Healthy"
	}
	return map[string]any{"status": status, "health": health}
}

func azureWorkloadsInstanceValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	return azureDataBoxObjects(azureARMObject(payload["properties"]), "properties.",
		"messageServerProperties", "enqueueServerProperties", "gatewayServerProperties", "enqueueReplicationServerProperties")
}

// azureWorkloadsInstanceInParent admits instances only to a registered
// system, as Azure Center for SAP solutions discovers them.
func azureWorkloadsInstanceInParent(parent map[string]any, _ string, _ map[string]any) string {
	if parent["state"] != azureWorkloadsRegistered {
		return "SAP instances can only be added to a registered system; this one is " + azureARMString(parent["state"]) + "."
	}
	return ""
}

func azureWorkloadsParentID(id string) string {
	parts := strings.Split(id, "/")
	return strings.Join(parts[:len(parts)-2], "/")
}

// azureWorkloadsInstanceReconcile keeps an instance in its system's region,
// allows one central services instance per system (one ASCS), and starts
// it in its system's running state.
func azureWorkloadsInstanceReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	parent := s.azureARMResources[azureARMKey(azureWorkloadsParentID(rec.ID))]
	if parent == nil {
		return nil, "The SAP system for " + rec.Name + " was not found."
	}
	norm := func(loc string) string { return strings.ToLower(strings.ReplaceAll(loc, " ", "")) }
	if norm(rec.Location) != norm(parent.Location) {
		return nil, "location must be the SAP system's location, " + parent.Location + "."
	}
	if strings.HasSuffix(rec.Type, "/centralInstances") {
		for _, other := range s.azureWorkloadsInstancesLocked(parent.ID, "centralInstances") {
			if !strings.EqualFold(other.ID, rec.ID) {
				return nil, "The SAP system already has the central services instance " + other.Name + "."
			}
		}
	}
	out := map[string]any{"instanceNo": "00"}
	if strings.HasSuffix(rec.Type, "/databaseInstances") {
		out = map[string]any{"databaseSid": parent.Name}
	}
	if existing != nil {
		out["status"], out["health"] = existing.Properties["status"], existing.Properties["health"]
		return out, ""
	}
	status := "Running"
	if parent.Properties["status"] == "Offline" {
		status = "Offline"
	}
	return azureARMMerge(out, azureWorkloadsStatus(status)), ""
}

// azureWorkloadsInstancesLocked returns a system's instances, of one kind
// or, with kind empty, of every kind. Caller holds providerStorageMu.
func (s *Server) azureWorkloadsInstancesLocked(visID, kind string) []*azureARMResource {
	prefix := azureARMKey(visID) + "/"
	return s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		rest, ok := strings.CutPrefix(azureARMKey(r.ID), prefix)
		return ok && strings.Count(rest, "/") == 1 && (kind == "" || strings.HasPrefix(rest, strings.ToLower(kind)+"/"))
	})
}

// azureWorkloadsPower starts or stops a system, with all its instances, or
// one instance, whose system then reports Running, Offline or
// PartiallyRunning. Only an installed system can start or stop.
func azureWorkloadsPower(running bool) func(*Server, *azureARMResource, map[string]any) (int, any) {
	status := map[bool]string{true: "Running", false: "Offline"}[running]
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		if raw, present := payload["softStopTimeoutSeconds"]; present && !running {
			if n, ok := azureFileShareInt(raw); !ok || n < 0 {
				return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "softStopTimeoutSeconds must be a non-negative integer.")
			}
		}
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		visID := rec.ID
		instance := !strings.EqualFold(rec.Type, azureWorkloadsNamespace+"/"+azureWorkloadsVISType)
		if instance {
			visID = azureWorkloadsParentID(rec.ID)
		}
		vis := s.azureARMResources[azureARMKey(visID)]
		if vis == nil || s.azureARMResources[azureARMKey(rec.ID)] == nil {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
		}
		if vis.Properties["state"] != azureWorkloadsRegistered {
			return http.StatusConflict, azureARMErrorBody("Conflict", "The SAP system "+vis.Name+" can't start or stop until its software is installed; it is "+azureARMString(vis.Properties["state"])+".")
		}
		for _, child := range s.azureWorkloadsInstancesLocked(visID, "") {
			if !instance || strings.EqualFold(child.ID, rec.ID) {
				s.azureWorkloadsSetStatusLocked(child, status)
			}
		}
		system := status
		if instance {
			system = s.azureWorkloadsSystemStatusLocked(visID)
		}
		s.azureWorkloadsSetStatusLocked(vis, system)
		return http.StatusOK, azureWorkloadsOperation(rec)
	}
}

// azureWorkloadsSystemStatusLocked derives a system's status from its
// instances'. Caller holds providerStorageMu.
func (s *Server) azureWorkloadsSystemStatusLocked(visID string) string {
	seen := map[any]bool{}
	for _, child := range s.azureWorkloadsInstancesLocked(visID, "") {
		seen[child.Properties["status"]] = true
	}
	if len(seen) == 1 && seen["Offline"] {
		return "Offline"
	}
	if seen["Offline"] {
		return "PartiallyRunning"
	}
	return "Running"
}

func (s *Server) azureWorkloadsSetStatusLocked(rec *azureARMResource, status string) {
	updated := *rec
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(rec.Properties, azureWorkloadsStatus(status))
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
}

// azureWorkloadsOperation is the OperationStatusResult a finished
// long-running operation returns.
func azureWorkloadsOperation(rec *azureARMResource) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	name := azureWorkloadsHash(rec.ID+"#"+now, 32)
	return map[string]any{
		"id":     "/subscriptions/" + rec.SubscriptionID + "/providers/" + azureWorkloadsNamespace + "/locations/" + strings.ToLower(strings.ReplaceAll(rec.Location, " ", "")) + "/operationStatuses/" + name,
		"name":   name,
		"status": "Succeeded", "startTime": now, "endTime": now,
	}
}

func azureWorkloadsHash(text string, n int) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ToLower(text))))[:n]
}

// azureWorkloadsMonitorReconcile fills in the monitoring resources Azure
// Monitor for SAP solutions creates in its managed resource group: a
// storage account, an identity and, unless one is given, a Log Analytics
// workspace. The managed resource group can't change after creation.
func azureWorkloadsMonitorReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	mrg := azureARMString(azureARMObject(props["managedResourceGroupConfiguration"])["name"])
	if existing != nil {
		old := azureARMString(azureARMObject(existing.Properties["managedResourceGroupConfiguration"])["name"])
		if mrg != "" && mrg != old {
			return nil, "properties.managedResourceGroupConfiguration.name can't be changed after creation."
		}
		mrg = old
	}
	if mrg == "" {
		mrg = "sapmonitor-" + strings.ToLower(rec.Name) + "-mrg"
	}
	group := "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + mrg + "/providers/"
	out := map[string]any{
		"managedResourceGroupConfiguration": map[string]any{"name": mrg},
		"storageAccountArmId":               group + "Microsoft.Storage/storageAccounts/sapmon" + azureWorkloadsHash(rec.ID, 18),
		"msiArmId":                          group + "Microsoft.ManagedIdentity/userAssignedIdentities/sapmon-msi-" + azureWorkloadsHash(rec.ID, 12),
	}
	if props["logAnalyticsWorkspaceArmId"] == nil {
		out["logAnalyticsWorkspaceArmId"] = group + "Microsoft.OperationalInsights/workspaces/sapmon-law-" + azureWorkloadsHash(rec.ID, 12)
	}
	if props["routingPreference"] == nil {
		out["routingPreference"] = "Default"
	}
	if props["appLocation"] == nil {
		out["appLocation"] = rec.Location
	}
	return out, ""
}
