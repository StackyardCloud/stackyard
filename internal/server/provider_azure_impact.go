package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Impact: Azure Impact Reporting. Customers report workload
// impacts against their resources, list the categories they may report
// under, and read the insights Microsoft correlates with each report. HPC
// Guest Health Reporting impacts (Resource.Hpc.*) are acknowledged with a
// MitigationAction insight; other impacts get insights only when one is
// created through the (admin) insights API. Connectors only store their
// configuration: no alerts are read.
const azureImpactNamespace = "Microsoft.Impact"

const azureImpactInsightType = azureImpactNamespace + "/workloadImpacts/insights"

// azureImpactCategories are the categories Learn lists for Azure Impact
// Reporting and for HPC Guest Health Reporting, with their descriptions.
var azureImpactCategories = [][2]string{
	{"ARMOperation.CreateOrUpdate", "To report problems related to creating a new Azure virtual machine, such as provisioning or allocation failures."},
	{"ARMOperation.Delete", "To report failures in deleting a resource."},
	{"ARMOperation.Get", "To report failures in querying resource metadata."},
	{"ARMOperation.Start", "To report failures in starting a resource."},
	{"ARMOperation.Stop", "To report failures in stopping a resource."},
	{"ARMOperation.Other", "To report control plane operation failures that don't fall into other ARMOperation categories."},
	{"Resource.Performance", "To report general performance issues. For example, high usage of CPU, IOPS, disk space, or memory."},
	{"Resource.Performance.Network", "To report performance issues that are networking related. For example, degraded network throughput."},
	{"Resource.Performance.Disk", "To report performance issues that are disk related. For example, degraded IOPS."},
	{"Resource.Performance.CPU", "To report performance issues that are CPU related."},
	{"Resource.Performance.Other", "To report issues that don't fall under other Resource.Performance subcategories."},
	{"Resource.Connectivity", "To report general connectivity issues to or from a resource."},
	{"Resource.Connectivity.Inbound", "To report inbound connectivity issues to a resource."},
	{"Resource.Connectivity.Outbound", "To report outbound connectivity issues from a resource."},
	{"Resource.Connectivity.Other", "To report issues that don't fall under other Resource.Connectivity subcategories."},
	{"Resource.Availability", "To report general unavailability issues."},
	{"Resource.Availability.Restart", "To report if an unexpected virtual machine restarts."},
	{"Resource.Availability.Boot", "To report virtual machines that are in a nonbootable state, not booting at all, or on a reboot loop."},
	{"Resource.Availability.Disk", "To report availability issues related to disks."},
	{"Resource.Availability.UnResponsive", "To report a resource that isn't responsive now or for a time in the past."},
	{"Resource.Availability.Storage", "To report availability issues related to storage."},
	{"Resource.Availability.Network", "To report network availability issues."},
	{"Resource.Availability.DNS", "To report Domain Name System availability issues."},
	{"Resource.Availability.Other", "To report issues that don't fall under other Resource.Availability subcategories."},
	{"Resource.Hpc.Reset", "Reset node health status."},
	{"Resource.Hpc.Reboot", "Restart the node."},
	{"Resource.Hpc.Unhealthy.HpcMissingGpu", "Missing GPU."},
	{"Resource.Hpc.Unhealthy.MissingIB", "Missing InfiniBand port."},
	{"Resource.Hpc.Unhealthy.IBPerformance", "Degraded InfiniBand performance."},
	{"Resource.Hpc.Unhealthy.IBPortDown", "InfiniBand port is in a down state."},
	{"Resource.Hpc.Unhealthy.IBPortFlapping", "InfiniBand port flapping."},
	{"Resource.Hpc.Unhealthy.HpcGpuDcgmDiagFailure", "Diagnostic failure of Data Center GPU Management Interface (DCGMI) for the GPU datacenter."},
	{"Resource.Hpc.Unhealthy.HpcRowRemapFailure", "Failure of GPU row remapping."},
	{"Resource.Hpc.Unhealthy.HpcInforomCorruption", "GPU infoROM corruption."},
	{"Resource.Hpc.Unhealthy.HpcGenericFailure", "Issue doesn't fall into any other category."},
	{"Resource.Hpc.Unhealthy.ManualInvestigation", "Request further manual investigation by the HPC team."},
	{"Resource.Hpc.Unhealthy.XID95UncontainedECCError", "GPU uncontained Error-Correcting Code (ECC) error (Xid 95)."},
	{"Resource.Hpc.Unhealthy.XID94ContainedECCError", "GPU contained ECC error (Xid 94)."},
	{"Resource.Hpc.Unhealthy.XID79FallenOffBus", "GPU fell off the Peripheral Component Interconnect Express (PCIe) bus (Xid 79)."},
	{"Resource.Hpc.Unhealthy.XID48DoubleBitECC", "GPU reports a double-bit ECC error (Xid 48)."},
	{"Resource.Hpc.Unhealthy.UnhealthyGPUNvidiasmi", "NVIDIA System Management Interface (nvidia-smi) stops responding and might not recover."},
	{"Resource.Hpc.Unhealthy.NvLink", "NvLink is down."},
	{"Resource.Hpc.Unhealthy.HpcDcgmiThermalReport", "DCGMI reports thermal violations."},
	{"Resource.Hpc.Unhealthy.ECCPageRetirementTableFull", "Page retirements for double-bit ECC errors are over the threshold."},
	{"Resource.Hpc.Unhealthy.DBEOverLimit", "GPU has more than 10 retired pages for double-bit ECC errors in seven days."},
	{"Resource.Hpc.Unhealthy.GpuXIDError", "GPU reports a Xid error other than 48, 79, 94, or 95."},
	{"Resource.Hpc.Unhealthy.AmdGpuResetFailed", "AMD GPU error for an unrecoverable reset failure."},
	{"Resource.Hpc.Unhealthy.EROTFailure", "Failure of GPU memory External Root of Trust (eRoT)."},
	{"Resource.Hpc.Unhealthy.GPUMemoryBWFailure", "Failure of GPU memory bandwidth."},
	{"Resource.Hpc.Unhealthy.CPUPerformance", "CPU performance issue."},
}

func init() {
	ns := azureImpactNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workloadImpacts", Subscription: true, NoPatch: true,
		ValidateName:    azureImpactNameError(azureImpactName, "Workload impact"),
		Validate:        azureImpactValidate,
		Reconcile:       azureImpactReconcile,
		DefaultChildren: azureImpactAcknowledgement,
		ChildProperties: azureImpactInsightsByCategory,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workloadImpacts/insights", Subscription: true, NoPatch: true,
		ValidateName: azureImpactNameError(azureImpactInsightName, "Insight"),
		Validate:     azureImpactInsightValidate,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"provisioningState": "Succeeded"}
		},
		Reconcile: azureImpactInsightReconcile,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "connectors", Subscription: true,
		ValidateName: azureImpactNameError(azureImpactConnectorName, "Connector"),
		Validate:     azureImpactConnectorValidate,
		PatchBody:    azureImpactConnectorPatch,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"tenantId": azureDefaultTenantID, "processingState": "Initialized", "processingStateMessage": "", "provisioningState": "Succeeded"}
		},
		Reconcile: func(_ *Server, existing, _ *azureARMResource) (map[string]any, string) {
			if existing != nil {
				return map[string]any{"connectorId": existing.Properties["connectorId"]}, ""
			}
			return map[string]any{"connectorId": uuid.NewString()}, ""
		},
	})
	registerAzureARMQueryView(ns, "impactCategories", azureImpactCategoryView)
	registerAzureARMScopeAction(ns, "getUploadToken", azureImpactUploadToken)
}

// azureImpactCategoryID is a category's stable emulator ID, derived from
// its name so parents can be named before they're listed.
func azureImpactCategoryID(name string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("stackyard:impact-category:"+strings.ToLower(name))).String()
}

// azureImpactFindCategory returns the catalog spelling of a category name,
// matched case-insensitively.
func azureImpactFindCategory(name string) (string, bool) {
	i := slices.IndexFunc(azureImpactCategories, func(c [2]string) bool { return strings.EqualFold(c[0], name) })
	if i < 0 {
		return "", false
	}
	return azureImpactCategories[i][0], true
}

func azureImpactIsHpc(category string) bool {
	return strings.HasPrefix(strings.ToLower(category), "resource.hpc.")
}

// azureImpactRequired lists the properties a category requires beyond
// the ones every report needs: ARM correlation IDs for control plane
// failures, and a description and host name for Guest Health Reporting.
func azureImpactRequired(category string) []string {
	switch {
	case strings.HasPrefix(strings.ToLower(category), "armoperation."):
		return []string{"armCorrelationIds"}
	case azureImpactIsHpc(category):
		return []string{"impactDescription", "additionalProperties.PhysicalHostName"}
	}
	return nil
}

// azureImpactCategoryView lists the catalog at subscription scope.
// categoryName keeps a category and its subcategories; resourceType keeps
// the categories that apply to it: HPC categories apply to virtual
// machines only, the others to every resource type.
func azureImpactCategoryView(_ *Server, scope string, query url.Values) []map[string]any {
	sub, ok := azureARMSubscriptionScope(scope)
	if !ok {
		return nil
	}
	name := strings.ToLower(query.Get("categoryName"))
	resourceType := strings.ToLower(query.Get("resourceType"))
	var items []map[string]any
	for _, c := range azureImpactCategories {
		lower := strings.ToLower(c[0])
		if name != "" && lower != name && !strings.HasPrefix(lower, name+".") {
			continue
		}
		if resourceType != "" && azureImpactIsHpc(c[0]) && resourceType != "microsoft.compute/virtualmachines" {
			continue
		}
		props := map[string]any{"categoryId": azureImpactCategoryID(c[0]), "description": c[1], "provisioningState": "Succeeded"}
		if i := strings.LastIndex(c[0], "."); i > 0 {
			props["parentCategoryId"] = azureImpactCategoryID(c[0][:i])
		}
		if required := azureImpactRequired(c[0]); required != nil {
			list := []any{}
			for _, p := range required {
				list = append(list, map[string]any{"name": p})
			}
			props["requiredImpactProperties"] = list
		}
		items = append(items, map[string]any{
			"id": "/subscriptions/" + sub + "/providers/" + azureImpactNamespace + "/impactCategories/" + c[0], "name": c[0],
			"type": azureImpactNamespace + "/impactCategories", "properties": props,
		})
	}
	return items
}

// azureImpactReconcile stamps a new report. Reports can't be changed: a
// PUT to an existing name is rejected, as Guest Health Reporting
// documents.
func azureImpactReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if existing != nil {
		return nil, "A workload impact named '" + existing.Name + "' already exists; reported impacts can't be changed."
	}
	category, _ := azureImpactFindCategory(azureARMString(rec.Properties["impactCategory"]))
	return map[string]any{
		"impactCategory": category, "impactUniqueId": uuid.NewString(),
		"reportedTimeUtc": rec.CreatedAt.Format(time.RFC3339Nano), "provisioningState": "Succeeded",
	}, ""
}

// azureImpactAcknowledgement is the MitigationAction insight Guest Health
// Reporting posts when it accepts an HPC report.
func azureImpactAcknowledgement(parent *azureARMResource) []*azureARMResource {
	category := azureARMString(parent.Properties["impactCategory"])
	if !azureImpactIsHpc(category) {
		return nil
	}
	// Resource.Hpc.{Reset,Reboot,Unhealthy.*}
	kind, _, _ := strings.Cut(strings.TrimPrefix(category, "Resource.Hpc."), ".")
	sum := sha256.Sum256([]byte(azureARMKey(parent.ID)))
	insight, _ := lookupAzureARMResourceType(azureImpactNamespace, "workloadImpacts/insights")
	title := "Customer reports investigate state - HPC Acknowledge"
	return []*azureARMResource{azureARMChild(parent, insight, hex.EncodeToString(sum[:8]), map[string]any{
		"category": "MitigationAction", "status": "Acknowledged",
		"eventTime": parent.CreatedAt.Format(time.RFC3339Nano), "insightUniqueId": uuid.NewString(),
		"impact":            azureImpactDetails(parent),
		"additionalDetails": map[string]any{"statusCode": "Acknowledged" + kind, "terminalInsight": false},
		"content":           map[string]any{"title": title, "description": "<h2><strong>" + title + "</strong></h2>"},
	})}
}

// azureImpactDetails is the impact an insight was generated for.
func azureImpactDetails(impact *azureARMResource) map[string]any {
	details := map[string]any{
		"impactId": impact.ID, "impactedResourceId": impact.Properties["impactedResourceId"],
		"startTime": impact.Properties["startDateTime"],
	}
	if end, ok := impact.Properties["endDateTime"].(string); ok {
		details["endTime"] = end
	}
	return details
}

// azureImpactInsightsByCategory groups a report's insights by category;
// a category's status is that of its most recently written insight.
func azureImpactInsightsByCategory(s *Server, rec *azureARMResource) map[string]any {
	insight, _ := lookupAzureARMResourceType(azureImpactNamespace, "workloadImpacts/insights")
	type group struct {
		status  any
		updated time.Time
		refs    []any
	}
	groups := map[string]*group{}
	var order []string
	for _, child := range s.azureARMInlineChildrenLocked(rec.ID, insight) {
		category := azureARMString(child.Properties["category"])
		g := groups[category]
		if g == nil {
			g = &group{}
			groups[category] = g
			order = append(order, category)
		}
		g.refs = append(g.refs, map[string]any{"id": child.ID})
		if !child.UpdatedAt.Before(g.updated) {
			g.status, g.updated = child.Properties["status"], child.UpdatedAt
		}
	}
	slices.Sort(order)
	list := []any{}
	for _, category := range order {
		g := groups[category]
		list = append(list, map[string]any{"category": category, "status": g.status, "insights": g.refs})
	}
	return map[string]any{"insightsByCategory": list}
}

// azureImpactInsightReconcile fills an insight's impact from its report
// and gives it a unique ID unless the caller sent one; the ID never
// changes afterward.
func azureImpactInsightReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	parentID := rec.ID[:strings.LastIndex(rec.ID[:strings.LastIndex(rec.ID, "/")], "/")]
	parent := s.azureARMResources[azureARMKey(parentID)]
	if parent == nil {
		return nil, "The workload impact '" + parentID + "' was not found."
	}
	sent := azureARMObject(rec.Properties["impact"])
	if id := azureARMString(sent["impactId"]); id != "" && !strings.EqualFold(id, parent.ID) {
		return nil, "properties.impact.impactId must be the insight's workload impact, '" + parent.ID + "'."
	}
	out := map[string]any{"impact": azureARMMerge(azureImpactDetails(parent), sent)}
	switch {
	case existing != nil && existing.Properties["insightUniqueId"] != nil:
		out["insightUniqueId"] = existing.Properties["insightUniqueId"]
	case azureARMString(rec.Properties["insightUniqueId"]) == "":
		out["insightUniqueId"] = uuid.NewString()
	}
	return out, ""
}

// azureImpactUploadToken returns a write URL for one Guest Health
// Reporting log file. The blob name is the UTC time and an eight-digit
// hex suffix, as Learn describes; the host is the emulator's and nothing
// is uploaded.
func azureImpactUploadToken(_ *Server, scope string, _ map[string]any) (int, any) {
	sub, ok := azureARMSubscriptionScope(scope)
	if !ok {
		return http.StatusBadRequest, azureARMErrorBody("InvalidScope", "Upload tokens are issued at subscription scope.")
	}
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	sum := sha256.Sum256([]byte(strings.ToLower(sub)))
	blob := time.Now().UTC().Format("20060102150405") + "_" + hex.EncodeToString(suffix) + ".gz"
	return http.StatusOK, map[string]any{
		"uploadUrl": azureImpactLogHost + "/sub" + hex.EncodeToString(sum[:6]) + "/" + blob + "?sv=2024-11-04&sr=b&sp=cw&sig=stackyard",
	}
}
