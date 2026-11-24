package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Microsoft.ContainerService fleets (Azure Kubernetes Fleet Manager,
// api-version 2025-03-01), served by the shared ARM handler: fleets with an
// optional hub, members, update strategies, update runs and auto-upgrade
// profiles. Update runs complete as soon as they start; no cluster is
// upgraded. Member clusters are referenced by ID and need not exist here.
const azureFltNamespace = "Microsoft.ContainerService"

var (
	azureFltClusterID     = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.ContainerService/managedClusters/[^/]+$`)
	azureFltStrategyID    = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.ContainerService/fleets/[^/]+/updateStrategies/[^/]+$`)
	azureFltUpgradeTypes  = []string{"Full", "NodeImageOnly", "ControlPlaneOnly"}
	azureFltChannels      = []string{"Stable", "Rapid", "NodeImage"}
	azureFltNodeSelection = []string{"Latest", "Consistent"}
	azureFltDNSPrefix     = regexp.MustCompile(`^[a-zA-Z0-9]$|^[a-zA-Z0-9][a-zA-Z0-9-]{0,52}[a-zA-Z0-9]$`)
	azureFltGroupName     = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,48}[a-z0-9])?$`)
)

// azureFltMaxStages bounds an update strategy, like the service's limit.
const azureFltMaxStages = 50

func init() {
	ns := azureFltNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "fleets",
		ETag:               true,
		Validate:           azureFltValidateFleet,
		ReadOnlyProperties: azureFltFleetProperties,
		Actions:            map[string]azureARMAction{"listcredentials": azureFltListCredentials},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "fleets/members", Proxy: true, ETag: true,
		Validate:    azureFltValidateMember,
		ValidatePut: azureFltOneMemberPerCluster,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"status": map[string]any{"lastOperationId": "", "lastOperationError": nil}}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "fleets/updateStrategies", Proxy: true, ETag: true, NoPatch: true,
		Validate: func(payload map[string]any) string {
			return azureFltValidateStrategy(azureARMObject(azureARMObject(payload["properties"])["strategy"]), "properties.strategy")
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "fleets/updateRuns", Proxy: true, ETag: true, NoPatch: true,
		Validate:        azureFltValidateRun,
		Reconcile:       azureFltResolveStrategy,
		StateProperties: map[string]any{"status": map[string]any{"status": map[string]any{"state": "NotStarted"}}},
		ValidateDelete: func(_ *Server, existing *azureARMResource) string {
			if azureFltRunState(existing) == "Running" {
				return "The update run is running; stop it before deleting it."
			}
			return ""
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"start": azureFltRunTransition("NotStarted", "Completed"),
			"stop":  azureFltRunTransition("Running", "Stopped"),
			"skip":  azureFltSkip,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "fleets/autoUpgradeProfiles", Proxy: true, ETag: true, NoPatch: true,
		Validate: azureFltValidateAutoUpgrade,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"generateupdaterun": azureFltGenerateUpdateRun,
		},
	})
}

func azureFltValidateFleet(payload map[string]any) string {
	if identity := azureARMObject(payload["identity"]); len(identity) > 0 {
		if msg := azureWorkloadsEnums(identity, "identity.", []string{"type"}, map[string][]string{"type": {"None", "SystemAssigned", "UserAssigned", "SystemAssigned, UserAssigned"}}); msg != "" {
			return msg
		}
	}
	hub := azureARMObject(azureARMObject(payload["properties"])["hubProfile"])
	if prefix, ok := hub["dnsPrefix"]; ok && !azureFltDNSPrefix.MatchString(azureARMString(prefix)) {
		return "properties.hubProfile.dnsPrefix is invalid: use 1 to 54 letters, digits and hyphens, starting and ending with a letter or digit."
	}
	return ""
}

// azureFltFleetProperties fills in the hub's server-owned addresses for a
// fleet created with a hub profile.
func azureFltFleetProperties(rec *azureARMResource) map[string]any {
	hub, ok := rec.Properties["hubProfile"].(map[string]any)
	if !ok {
		return nil
	}
	prefix := azureARMString(hub["dnsPrefix"])
	if prefix == "" {
		prefix = rec.Name
	}
	host := strings.ToLower(prefix) + "-" + azureWorkloadsHash(rec.ID, 8) + ".hcp." + strings.ToLower(rec.Location) + ".azmk8s.io"
	return map[string]any{"hubProfile": azureARMMerge(hub, map[string]any{
		"dnsPrefix": prefix, "fqdn": host, "portalFqdn": "portal-" + host, "kubernetesVersion": "1.31.2",
	})}
}

func azureFltListCredentials(rec *azureARMResource, _ map[string]any) (int, any) {
	hub, ok := rec.Properties["hubProfile"].(map[string]any)
	if !ok {
		return azureOracleBad("The fleet has no hub, so it has no credentials.")
	}
	kubeconfig := fmt.Sprintf("apiVersion: v1\nkind: Config\nclusters:\n- name: hub\n  cluster:\n    server: https://%s:443\ncontexts:\n- name: hub\n  context:\n    cluster: hub\n    user: hub-user\ncurrent-context: hub\nusers:\n- name: hub-user\n  user:\n    token: stackyard\n", azureARMString(hub["fqdn"]))
	return http.StatusOK, map[string]any{"kubeconfigs": []any{map[string]any{"name": "credentialUser", "value": base64.StdEncoding.EncodeToString([]byte(kubeconfig))}}}
}

func azureFltValidateMember(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !azureFltClusterID.MatchString(azureARMString(props["clusterResourceId"])) {
		return "properties.clusterResourceId is missing or invalid: it must be an AKS managed cluster resource ID."
	}
	if group, ok := props["group"]; ok && !azureFltGroupName.MatchString(azureARMString(group)) {
		return "properties.group is invalid: use 1 to 50 lowercase letters, digits and hyphens."
	}
	return ""
}

// azureFltOneMemberPerCluster rejects joining a cluster that is already a
// member of the fleet under another name.
func azureFltOneMemberPerCluster(s *Server, _, rec *azureARMResource) (string, string) {
	fleet := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/members/")]
	cluster := azureARMString(rec.Properties["clusterResourceId"])
	for _, other := range s.azureARMResourcesMatchingLocked(func(o *azureARMResource) bool {
		return strings.EqualFold(o.Type, rec.Type) && strings.HasPrefix(azureARMKey(o.ID), azureARMKey(fleet+"/")) && !strings.EqualFold(o.ID, rec.ID)
	}) {
		if strings.EqualFold(azureARMString(other.Properties["clusterResourceId"]), cluster) {
			return "Conflict", "The cluster is already a member of this fleet as " + other.Name + "."
		}
	}
	return "", ""
}

func azureFltValidateStrategy(strategy map[string]any, path string) string {
	stages := azureARMSlice(strategy["stages"])
	if len(stages) == 0 || len(stages) > azureFltMaxStages {
		return fmt.Sprintf("%s.stages is missing or invalid: list 1 to %d stages.", path, azureFltMaxStages)
	}
	seen := map[string]bool{}
	for i, raw := range stages {
		stage := azureARMObject(raw)
		name := strings.ToLower(azureARMString(stage["name"]))
		if name == "" || seen[name] {
			return fmt.Sprintf("%s.stages[%d].name is missing or duplicated.", path, i)
		}
		seen[name] = true
		for j, g := range azureARMSlice(stage["groups"]) {
			group := strings.ToLower(azureARMString(azureARMObject(g)["name"]))
			if group == "" || seen["group:"+group] {
				return fmt.Sprintf("%s.stages[%d].groups[%d].name is missing or duplicated.", path, i, j)
			}
			seen["group:"+group] = true
		}
		if wait, ok := stage["afterStageWaitInSeconds"]; ok {
			if n, ok := wait.(float64); !ok || n < 0 || n > 3600 || n != float64(int(n)) {
				return fmt.Sprintf("%s.stages[%d].afterStageWaitInSeconds must be an integer from 0 to 3600.", path, i)
			}
		}
	}
	return ""
}

func azureFltValidateRun(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	update := azureARMObject(props["managedClusterUpdate"])
	upgrade := azureARMObject(update["upgrade"])
	if msg := azureWorkloadsEnums(upgrade, "properties.managedClusterUpdate.upgrade.", []string{"type"}, map[string][]string{"type": azureFltUpgradeTypes}); msg != "" {
		return msg
	}
	if upgrade["type"] != "NodeImageOnly" && strings.TrimSpace(azureARMString(upgrade["kubernetesVersion"])) == "" {
		return "properties.managedClusterUpdate.upgrade.kubernetesVersion is required for a " + azureARMString(upgrade["type"]) + " upgrade."
	}
	if upgrade["type"] == "NodeImageOnly" && azureARMString(upgrade["kubernetesVersion"]) != "" {
		return "properties.managedClusterUpdate.upgrade.kubernetesVersion is invalid for a NodeImageOnly upgrade."
	}
	if sel := azureARMObject(update["nodeImageSelection"]); len(sel) > 0 {
		if msg := azureWorkloadsEnums(sel, "properties.managedClusterUpdate.nodeImageSelection.", []string{"type"}, map[string][]string{"type": azureFltNodeSelection}); msg != "" {
			return msg
		}
	}
	strategyID, hasID := props["updateStrategyId"]
	_, hasStrategy := props["strategy"]
	switch {
	case hasID && hasStrategy:
		return "Set either properties.updateStrategyId or properties.strategy, not both."
	case hasID && !azureFltStrategyID.MatchString(azureARMString(strategyID)):
		return "properties.updateStrategyId is invalid: it must be a fleet update strategy resource ID."
	case hasStrategy:
		return azureFltValidateStrategy(azureARMObject(props["strategy"]), "properties.strategy")
	}
	return ""
}

// azureFltResolveStrategy copies the referenced update strategy into the
// run, as the service snapshots it when the run is created.
func azureFltResolveStrategy(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if existing != nil {
		if azureFltRunState(existing) != "NotStarted" {
			return nil, "The update run has started and can no longer change."
		}
	}
	id := azureARMString(rec.Properties["updateStrategyId"])
	if id == "" {
		return nil, ""
	}
	fleet := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/updateruns/")]
	if !strings.HasPrefix(azureARMKey(id), azureARMKey(fleet+"/")) {
		return nil, "properties.updateStrategyId must name a strategy of the same fleet."
	}
	strategy := s.azureARMResources[azureARMKey(id)]
	if strategy == nil {
		return nil, "The update strategy " + id + " was not found."
	}
	return map[string]any{"strategy": strategy.Properties["strategy"]}, ""
}

func azureFltRunState(rec *azureARMResource) string {
	return azureARMString(azureARMObject(azureARMObject(rec.Properties["status"])["status"])["state"])
}

// azureFltRunTransition moves an update run from one state to another.
// ponytail: a started run completes at once (no cluster is upgraded), so stop
// only ever finds runs that are not running; model stages if clients poll progress.
func azureFltRunTransition(from, to string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		stored := s.azureARMResources[azureARMKey(rec.ID)]
		if stored == nil {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The update run was not found.")
		}
		if state := azureFltRunState(stored); state != from {
			return azureOracleConflict("The update run is " + state + "; this action needs " + from + ".")
		}
		updated := s.azureOracleUpdateLocked(stored.ID, map[string]any{"status": map[string]any{"status": map[string]any{"state": to}}})
		return http.StatusOK, updated.fixture()
	}
}

// azureFltSkip records the targets a run skips; it must not have started.
func azureFltSkip(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	targets := azureARMSlice(payload["targets"])
	if len(targets) == 0 {
		return azureOracleBad("targets is required: list the members, groups or stages to skip.")
	}
	for i, raw := range targets {
		target := azureARMObject(raw)
		if msg := azureWorkloadsEnums(target, fmt.Sprintf("targets[%d].", i), []string{"type", "name"}, map[string][]string{"type": {"Member", "Group", "Stage", "AfterStageWait"}}); msg != "" {
			return azureOracleBad(msg)
		}
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stored := s.azureARMResources[azureARMKey(rec.ID)]
	if stored == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The update run was not found.")
	}
	if state := azureFltRunState(stored); state == "Completed" || state == "Stopped" || state == "Failed" {
		return azureOracleConflict("The update run is " + state + "; nothing is left to skip.")
	}
	updated := s.azureOracleUpdateLocked(stored.ID, map[string]any{"skippedTargets": targets})
	return http.StatusOK, updated.fixture()
}

func azureFltValidateAutoUpgrade(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureWorkloadsEnums(props, "properties.", []string{"channel"}, map[string][]string{"channel": azureFltChannels}); msg != "" {
		return msg
	}
	if sel := azureARMObject(props["nodeImageSelection"]); len(sel) > 0 {
		if msg := azureWorkloadsEnums(sel, "properties.nodeImageSelection.", []string{"type"}, map[string][]string{"type": azureFltNodeSelection}); msg != "" {
			return msg
		}
	}
	if id, ok := props["updateStrategyId"]; ok && !azureFltStrategyID.MatchString(azureARMString(id)) {
		return "properties.updateStrategyId is invalid: it must be a fleet update strategy resource ID."
	}
	return ""
}

// azureFltGenerateUpdateRun creates an update run from an auto-upgrade
// profile, named after the profile and the time, and returns its ID.
func azureFltGenerateUpdateRun(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	if disabled, _ := rec.Properties["disabled"].(bool); disabled {
		return azureOracleConflict("The auto-upgrade profile is disabled.")
	}
	upgrade := map[string]any{"type": "Full", "kubernetesVersion": "1.31.2"}
	if rec.Properties["channel"] == "NodeImage" {
		upgrade = map[string]any{"type": "NodeImageOnly"}
	}
	props := map[string]any{
		"managedClusterUpdate": map[string]any{"upgrade": upgrade, "nodeImageSelection": azureARMMerge(map[string]any{"type": "Latest"}, azureARMObject(rec.Properties["nodeImageSelection"]))},
		"autoUpgradeProfileId": rec.ID,
		"status":               map[string]any{"status": map[string]any{"state": "NotStarted"}},
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	fleetID := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/autoupgradeprofiles/")]
	fleet := s.azureARMResources[azureARMKey(fleetID)]
	if fleet == nil || s.azureARMResources[azureARMKey(rec.ID)] == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The auto-upgrade profile was not found.")
	}
	if id := azureARMString(rec.Properties["updateStrategyId"]); id != "" {
		if strategy := s.azureARMResources[azureARMKey(id)]; strategy != nil {
			props["strategy"] = strategy.Properties["strategy"]
		}
		props["updateStrategyId"] = id
	}
	runTyp, _ := lookupAzureARMResourceType(azureFltNamespace, "fleets/updateRuns")
	name := rec.Name + "-" + azureWorkloadsHash(rec.ID+rec.UpdatedAt.String()+fmt.Sprint(len(s.azureARMResources)), 8)
	run := azureARMChild(fleet, runTyp, name, azureARMMerge(props, map[string]any{"provisioningState": "Succeeded"}))
	run.APIVersion = rec.APIVersion
	run.CreatedAt = time.Now().UTC()
	run.UpdatedAt = run.CreatedAt
	s.azureARMResources[azureARMKey(run.ID)] = run
	return http.StatusOK, map[string]any{"id": run.ID}
}
