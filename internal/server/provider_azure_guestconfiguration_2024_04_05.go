package server

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.GuestConfiguration assignments (api-version 2024-04-05), served
// by the shared ARM handler as extension resources on virtual machines,
// scale sets, Arc-enabled servers (HCRP) and vSphere VMs. No guest agent
// runs: every assignment reports Compliant, with one report derived from it.
const (
	azureGcfgNamespace = "Microsoft.GuestConfiguration"
	azureGcfgType      = azureGcfgNamespace + "/guestConfigurationAssignments"
)

var (
	azureGcfgScope = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/(Microsoft\.Compute/virtualMachines|Microsoft\.Compute/virtualMachineScaleSets|Microsoft\.HybridCompute/machines|Microsoft\.ConnectedVMwarevSphere/virtualMachines)/[^/]+$`)
	azureGcfgModes = []string{"Audit", "DeployAndAutoCorrect", "ApplyAndAutoCorrect", "ApplyAndMonitor"}
)

// azureGcfgMaxParameters bounds configurationParameter and
// configurationProtectedParameter.
const azureGcfgMaxParameters = 100

func init() {
	ns := azureGcfgNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "guestConfigurationAssignments",
		Extension: true, KeepLocation: true,
		ValidateScope: func(scope string) string {
			if !azureGcfgScope.MatchString(scope) {
				return "Guest configuration assignments attach to a virtual machine, scale set, Arc-enabled server or vSphere VM."
			}
			return ""
		},
		Validate:            azureGcfgValidate,
		ReadOnlyProperties:  azureGcfgProperties,
		WriteOnlyProperties: []string{"guestConfiguration.configurationProtectedParameter"},
	})
	// Lists assignments at any scope: those attached to a machine, or every
	// one under a subscription or resource group. Writes go to the type.
	registerAzureARMView(ns, "guestConfigurationAssignments", func(s *Server, scope string) []map[string]any {
		direct := strings.Contains(strings.ToLower(scope), "/providers/")
		prefix := azureARMKey(scope + "/")
		var out []map[string]any
		for _, rec := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			key := azureARMKey(rec.ID)
			return strings.EqualFold(rec.Type, azureGcfgType) && strings.HasPrefix(key, prefix) &&
				(!direct || strings.Count(key[len(prefix):], "/") == 3)
		}) {
			out = append(out, rec.fixture())
		}
		return out
	})
	registerAzureARMView(ns, "guestConfigurationAssignments/reports", func(s *Server, scope string) []map[string]any {
		rec, _ := s.getAzureARMResource(scope, "", "", "")
		if rec == nil {
			return nil
		}
		return []map[string]any{azureGcfgReport(rec)}
	})
	registerAzureARMView(ns, "operations", azureGsvcOperationsView(ns, "Microsoft Guest Configuration", "guestConfigurationAssignments"))
}

func azureGcfgValidate(payload map[string]any) string {
	gc := azureARMObject(azureARMObject(payload["properties"])["guestConfiguration"])
	if strings.TrimSpace(azureARMString(gc["name"])) == "" {
		return "properties.guestConfiguration.name is required."
	}
	if msg := azureWorkloadsEnums(gc, "properties.guestConfiguration.", nil, map[string][]string{"assignmentType": azureGcfgModes, "kind": {"DSC"}}); msg != "" {
		return msg
	}
	for _, key := range []string{"configurationParameter", "configurationProtectedParameter"} {
		params := azureARMSlice(gc[key])
		if len(params) > azureGcfgMaxParameters {
			return fmt.Sprintf("properties.guestConfiguration.%s is invalid: at most %d parameters.", key, azureGcfgMaxParameters)
		}
		for i, p := range params {
			if strings.TrimSpace(azureARMString(azureARMObject(p)["name"])) == "" {
				return fmt.Sprintf("properties.guestConfiguration.%s[%d].name is required.", key, i)
			}
		}
	}
	if uri := azureARMString(gc["contentUri"]); uri != "" && !strings.HasPrefix(strings.ToLower(uri), "https://") {
		return "properties.guestConfiguration.contentUri is invalid: it must be an https URL."
	}
	return ""
}

// azureGcfgProperties reports the assignment as compliant on its target.
func azureGcfgProperties(rec *azureARMResource) map[string]any {
	target := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/providers/")]
	parts := strings.Split(target, "/")
	gc := azureARMObject(rec.Properties["guestConfiguration"])
	shown := azureARMMerge(map[string]any{"kind": "DSC", "assignmentType": "ApplyAndMonitor"}, gc)
	delete(shown, "configurationProtectedParameter")
	return map[string]any{
		"targetResourceId":            target,
		"resourceType":                parts[len(parts)-3] + "/" + parts[len(parts)-2],
		"complianceStatus":            "Compliant",
		"lastComplianceStatusChecked": rec.UpdatedAt.Format(time.RFC3339),
		"latestReportId":              rec.ID + "/reports/" + azureGcfgReportID(rec),
		"assignmentHash":              strings.ToUpper(azureWorkloadsHash(rec.ID+azureARMString(gc["version"])+azureARMString(gc["contentHash"]), 40)),
		"parameterHash":               strings.ToUpper(azureWorkloadsHash(fmt.Sprint(gc["configurationParameter"]), 40)),
		"guestConfiguration":          shown,
	}
}

func azureGcfgReportID(rec *azureARMResource) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+rec.UpdatedAt.String())).String()
}

// azureGcfgReport is the compliance report for the assignment's last check.
func azureGcfgReport(rec *azureARMResource) map[string]any {
	id := azureGcfgReportID(rec)
	gc := azureARMObject(rec.Properties["guestConfiguration"])
	checked := rec.UpdatedAt.Format(time.RFC3339)
	return map[string]any{
		"id":   rec.ID + "/reports/" + id,
		"name": id,
		"properties": map[string]any{
			"complianceStatus": "Compliant",
			"reportId":         id,
			"startTime":        checked,
			"endTime":          checked,
			"assignment": map[string]any{
				"name":          rec.Name,
				"configuration": map[string]any{"name": gc["name"], "version": gc["version"]},
			},
			"vm":      map[string]any{"id": rec.Properties["targetResourceId"], "uuid": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(azureARMString(rec.Properties["targetResourceId"])))).String()},
			"details": map[string]any{"complianceStatus": "Compliant", "startTime": checked, "endTime": checked, "jobId": id, "operationType": "Consistency", "resources": []any{}},
		},
	}
}
