package server

import (
	"fmt"
	"regexp"
	"strings"
)

// Microsoft.StandbyPool virtual machine and container group pools, served by
// the shared ARM handler. Pools are always full: the runtime view reports
// maxReadyCapacity instances in the pool's state, and a virtual machine
// pool lists that many standby machines from its scale set.
func init() {
	const ns = "Microsoft.StandbyPool"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "standbyVirtualMachinePools",
		ValidateName: azureStandbyPoolName,
		Validate:     azureStandbyVirtualMachinePoolValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "standbyContainerGroupPools",
		ValidateName: azureStandbyPoolName,
		Validate:     azureStandbyContainerGroupPoolValidate,
	})
	for _, pool := range []string{"standbyVirtualMachinePools", "standbyContainerGroupPools"} {
		registerAzureARMView(ns, pool+"/runtimeViews", azureStandbyPoolRuntimeViews)
	}
	registerAzureARMView(ns, "standbyVirtualMachinePools/standbyVirtualMachines", azureStandbyVirtualMachines)
}

// azureStandbyMaxCapacity is the largest maxReadyCapacity Azure accepts.
const azureStandbyMaxCapacity = 2000

var azureStandbyPoolNamePattern = regexp.MustCompile(`^[a-zA-Z0-9-]{3,24}$`)

func azureStandbyPoolName(name string) string {
	if !azureStandbyPoolNamePattern.MatchString(name) {
		return "Standby pool names must be 3 to 24 letters, digits or hyphens."
	}
	return ""
}

// azureWholeNumber reports whether raw is a JSON integer in [least, most].
func azureWholeNumber(raw any, least, most float64) bool {
	n, ok := raw.(float64)
	return ok && n == float64(int64(n)) && n >= least && n <= most
}

// azureStandbyCapacityValidate checks properties.elasticityProfile.
func azureStandbyCapacityValidate(props map[string]any) string {
	profile, ok := props["elasticityProfile"].(map[string]any)
	if !ok {
		return "properties.elasticityProfile is required."
	}
	if !azureWholeNumber(profile["maxReadyCapacity"], 0, azureStandbyMaxCapacity) {
		return fmt.Sprintf("properties.elasticityProfile.maxReadyCapacity must be an integer from 0 to %d.", azureStandbyMaxCapacity)
	}
	if raw, present := profile["minReadyCapacity"]; present && !azureWholeNumber(raw, 0, profile["maxReadyCapacity"].(float64)) {
		return "properties.elasticityProfile.minReadyCapacity must be an integer from 0 to maxReadyCapacity."
	}
	if raw, present := profile["refillPolicy"]; present && raw != "always" {
		return "properties.elasticityProfile.refillPolicy must be always."
	}
	return ""
}

func azureStandbyVirtualMachinePoolValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureStandbyCapacityValidate(props); msg != "" {
		return msg
	}
	if msg := azureEnumRequired("virtualMachineState", "Running", "Deallocated", "Hibernated", "Mix")(payload); msg != "" {
		return msg
	}
	if raw, present := props["attachedVirtualMachineScaleSetId"]; present && !strings.Contains(strings.ToLower(azureARMString(raw)), "/providers/microsoft.compute/virtualmachinescalesets/") {
		return "properties.attachedVirtualMachineScaleSetId must be a virtual machine scale set ID."
	}
	if props["virtualMachineState"] != "Mix" {
		return ""
	}
	// A mixed pool splits its machines between states by percentage.
	total := 0.0
	distribution := azureARMObject(props["vmStateDistribution"])
	for _, key := range []string{"runningPercent", "deallocatedPercent", "hibernatedPercent"} {
		raw, present := distribution[key]
		if !present {
			continue
		}
		if !azureWholeNumber(raw, 0, 100) {
			return "properties.vmStateDistribution." + key + " must be an integer from 0 to 100."
		}
		total += raw.(float64)
	}
	if total != 100 {
		return "properties.vmStateDistribution percentages must add up to 100 for a Mix pool."
	}
	return ""
}

func azureStandbyContainerGroupPoolValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureStandbyCapacityValidate(props); msg != "" {
		return msg
	}
	group := azureARMObject(props["containerGroupProperties"])
	if !strings.Contains(strings.ToLower(azureARMString(azureARMObject(group["containerGroupProfile"])["id"])), "/providers/microsoft.containerinstance/containergroupprofiles/") {
		return "properties.containerGroupProperties.containerGroupProfile.id must be a container group profile ID."
	}
	if raw, present := azureARMObject(group["containerGroupProfile"])["revision"]; present && !azureWholeNumber(raw, 0, 1<<53) {
		return "properties.containerGroupProperties.containerGroupProfile.revision must be a non-negative integer."
	}
	for _, subnet := range azureARMSlice(group["subnetIds"]) {
		if !strings.Contains(strings.ToLower(azureARMString(azureARMObject(subnet)["id"])), "/subnets/") {
			return "properties.containerGroupProperties.subnetIds items must be subnet IDs."
		}
	}
	return ""
}

// azureStandbyCapacity returns a pool's maxReadyCapacity, which PATCH can
// set without validation, clamped to what a PUT accepts.
func azureStandbyCapacity(pool *azureARMResource) int {
	n, _ := azureARMObject(pool.Properties["elasticityProfile"])["maxReadyCapacity"].(float64)
	return int(min(max(n, 0), azureStandbyMaxCapacity))
}

// azureStandbyPoolStateCounts splits a pool's maxReadyCapacity between
// instance states.
func azureStandbyPoolStateCounts(pool *azureARMResource) []any {
	capacity := azureStandbyCapacity(pool)
	state := azureARMString(pool.Properties["virtualMachineState"])
	switch state {
	case "":
		// Container groups are always running.
		state = "Running"
	case "Mix":
		distribution := azureARMObject(pool.Properties["vmStateDistribution"])
		counts := []any{}
		left := capacity
		for _, split := range [][2]string{{"Deallocated", "deallocatedPercent"}, {"Hibernated", "hibernatedPercent"}} {
			percent, _ := distribution[split[1]].(float64)
			n := min(capacity*int(min(max(percent, 0), 100))/100, left)
			left -= n
			counts = append(counts, map[string]any{"state": split[0], "count": n})
		}
		return append([]any{map[string]any{"state": "Running", "count": left}}, counts...)
	}
	return []any{map[string]any{"state": state, "count": capacity}}
}

// azureStandbyPoolRecord returns the pool a nested view's scope names.
func (s *Server) azureStandbyPoolRecord(scope string) *azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureARMResources[azureARMKey(scope)]
}

// azureStandbyPoolRuntimeViews has the one view Azure keeps, "latest".
func azureStandbyPoolRuntimeViews(s *Server, scope string) []map[string]any {
	pool := s.azureStandbyPoolRecord(scope)
	if pool == nil {
		return nil
	}
	return []map[string]any{{
		"id": scope + "/runtimeViews/latest", "name": "latest", "type": pool.Type + "/runtimeViews",
		"properties": map[string]any{
			"instanceCountSummary": []any{map[string]any{"instanceCountsByState": azureStandbyPoolStateCounts(pool)}},
			"status":               map[string]any{"code": "HealthState/healthy", "message": "The pool is healthy."},
			"provisioningState":    "Succeeded",
		},
	}}
}

// azureStandbyVirtualMachines lists a pool's machines, named after its
// scale set, when one is attached.
func azureStandbyVirtualMachines(s *Server, scope string) []map[string]any {
	pool := s.azureStandbyPoolRecord(scope)
	if pool == nil {
		return nil
	}
	scaleSet := azureARMString(pool.Properties["attachedVirtualMachineScaleSetId"])
	if scaleSet == "" {
		return nil
	}
	// PATCH can store any value, so recheck the scale set ID.
	i := strings.LastIndex(strings.ToLower(scaleSet), "/virtualmachinescalesets/")
	if i < 0 {
		return nil
	}
	capacity := azureStandbyCapacity(pool)
	group := scaleSet[:i]
	base := scaleSet[strings.LastIndex(scaleSet, "/")+1:]
	items := make([]map[string]any, 0, capacity)
	for i := range capacity {
		name := fmt.Sprintf("%s_%d", base, i)
		items = append(items, map[string]any{
			"id": scope + "/standbyVirtualMachines/" + name, "name": name, "type": pool.Type + "/standbyVirtualMachines",
			"properties": map[string]any{
				"virtualMachineResourceId": group + "/virtualMachines/" + name,
				"provisioningState":        "Succeeded",
			},
		})
	}
	return items
}
