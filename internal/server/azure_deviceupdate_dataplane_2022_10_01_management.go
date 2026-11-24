package server

import (
	"net/http"
	"strings"
	"time"
)

// Device management of the Device Update data plane (see
// provider_azure_deviceupdate_dataplane_2022_10_01.go): devices, groups,
// device classes, deployments, log collections and device health.

const (
	aduDPDefaultGroup = "default"
	aduDPDeviceClass  = "c0ffee00c0ffee00c0ffee00c0ffee00c0ffee00"
	aduDPMaxDevices   = 100
)

func (s *Server) aduDPManagement(w http.ResponseWriter, r *http.Request, base string, seg []string) {
	if len(seg) == 0 {
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
		return
	}
	head, action, _ := strings.Cut(seg[0], ":")
	switch strings.ToLower(head) {
	case "devices":
		if strings.EqualFold(action, "import") {
			s.aduDPImportDevices(w, r, base)
			return
		}
		s.dcdevChildren(w, r, base+"/devices", "adu/device", seg[1:])
	case "updatecompliance":
		if dcdevGetOnly(w, r) {
			respondJSON(w, http.StatusOK, s.aduDPCompliance(base, ""))
		}
	case "groups":
		s.aduDPGroups(w, r, base, seg[1:])
	case "deviceclasses":
		s.aduDPDeviceClasses(w, r, base, seg[1:])
	case "operations":
		s.dcdevChildren(w, r, base+"/management/operations", "adu/operation", seg[1:])
	case "devicediagnostics":
		s.aduDPDiagnostics(w, r, base, seg[1:])
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

// aduDPImportDevices adds the two sample devices; there is no IoT Hub to read.
func (s *Server) aduDPImportDevices(w http.ResponseWriter, r *http.Request, base string) {
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	body, err := readBodyBytes(r)
	kind := strings.Trim(strings.TrimSpace(string(body)), `"`)
	if err != nil || (kind != "Devices" && kind != "Modules" && kind != "All") {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", `the body must be "Devices", "Modules" or "All".`)
		return
	}
	for _, id := range []string{"device-1", "device-2"} {
		s.azureDPPut(azureDPRecord(base+"/devices/"+id, "adu/device", id, map[string]any{
			"deviceId": id, "deviceClassId": aduDPDeviceClass, "groupId": aduDPDefaultGroup, "onLatestUpdate": false, "deploymentStatus": "Succeeded",
		}, s.azureDPGet(base+"/devices/"+id)))
	}
	s.azureDPPut(azureDPRecord(base+"/groups/"+aduDPDefaultGroup, "adu/group", aduDPDefaultGroup, map[string]any{
		"groupId": aduDPDefaultGroup, "groupType": "IoTHubTag", "deviceCount": 2, "createdDateTime": time.Now().UTC().Format(time.RFC3339),
		"subgroupsWithNewUpdatesAvailableCount": 0, "subgroupsWithUpdatesInProgressCount": 0, "subgroupsWithOnLatestUpdateCount": 1,
	}, s.azureDPGet(base+"/groups/"+aduDPDefaultGroup)))
	s.azureDPPut(azureDPRecord(base+"/deviceclasses/"+aduDPDeviceClass, "adu/deviceClass", aduDPDeviceClass, map[string]any{
		"deviceClassId": aduDPDeviceClass, "friendlyName": "Contoso Sensor",
		"deviceClassProperties": map[string]any{"compatProperties": aduDPCompat, "contractModel": map[string]any{"id": "dtmi:azure:iot:deviceUpdateContractModel;2", "name": "Device Update Model V2"}},
	}, s.azureDPGet(base+"/deviceclasses/"+aduDPDeviceClass)))
	if s.aduDPOperation(w, base+"/management/operations", nil) {
		w.WriteHeader(http.StatusAccepted)
	}
}

func (s *Server) aduDPCompliance(base, group string) map[string]any {
	n := 0
	for _, d := range s.azureDPList(base+"/devices", "adu/device") {
		if group == "" || strings.EqualFold(azureARMString(d.Properties["groupId"]), group) {
			n++
		}
	}
	return map[string]any{"totalDeviceCount": n, "onLatestUpdateDeviceCount": 0, "newUpdatesAvailableDeviceCount": n, "updatesInProgressDeviceCount": 0}
}

// aduDPLatestUpdate returns the most recently imported update, or nil.
func (s *Server) aduDPLatestUpdate(base string) *azureARMResource {
	var latest *azureARMResource
	for _, u := range s.azureDPList(base+"/updates", "adu/update") {
		if latest == nil || u.CreatedAt.After(latest.CreatedAt) {
			latest = u
		}
	}
	return latest
}

func (s *Server) aduDPGroups(w http.ResponseWriter, r *http.Request, base string, seg []string) {
	if len(seg) == 0 {
		if dcdevGetOnly(w, r) {
			dcdevRespondList(w, aduDPSorted(s.azureDPList(base+"/groups", "adu/group")))
		}
		return
	}
	group := s.azureDPGet(base + "/groups/" + strings.ToLower(seg[0]))
	if group == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "group not found")
		return
	}
	if len(seg) == 1 {
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, dcdevRecordView(group))
		case http.MethodDelete:
			s.azureDPDelete(group.ID)
			w.WriteHeader(http.StatusNoContent)
		default:
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		}
		return
	}
	switch strings.ToLower(seg[1]) {
	case "updatecompliance":
		if dcdevGetOnly(w, r) {
			respondJSON(w, http.StatusOK, s.aduDPCompliance(base, group.Name))
		}
	case "bestupdates":
		if dcdevGetOnly(w, r) {
			dcdevRespondList(w, s.aduDPBestUpdates(base, group.Name))
		}
	case "deployments":
		s.aduDPDeployments(w, r, base, group, seg[2:], "")
	case "deviceclasssubgroups":
		s.aduDPSubgroups(w, r, base, group, seg[2:])
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) aduDPBestUpdates(base, group string) []any {
	latest := s.aduDPLatestUpdate(base)
	if latest == nil {
		return nil
	}
	return []any{map[string]any{
		"groupId": group, "deviceClassId": aduDPDeviceClass, "deviceCount": s.aduDPCompliance(base, group)["totalDeviceCount"],
		"update": map[string]any{"updateId": latest.Properties["updateId"], "description": latest.Properties["description"]},
	}}
}

func (s *Server) aduDPSubgroups(w http.ResponseWriter, r *http.Request, base string, group *azureARMResource, seg []string) {
	subgroup := map[string]any{"groupId": group.Name, "deviceClassId": aduDPDeviceClass, "deviceCount": group.Properties["deviceCount"], "createdDateTime": group.Properties["createdDateTime"]}
	if len(seg) == 0 {
		if dcdevGetOnly(w, r) {
			dcdevRespondList(w, []any{subgroup})
		}
		return
	}
	if !strings.EqualFold(seg[0], aduDPDeviceClass) {
		azureARMError(w, http.StatusNotFound, "NotFound", "device class subgroup not found")
		return
	}
	switch {
	case len(seg) == 1 && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, subgroup)
	case len(seg) == 1 && r.Method == http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	case len(seg) == 2 && strings.EqualFold(seg[1], "updateCompliance") && dcdevGetOnly(w, r):
		respondJSON(w, http.StatusOK, s.aduDPCompliance(base, group.Name))
	case len(seg) == 2 && strings.EqualFold(seg[1], "bestUpdates") && dcdevGetOnly(w, r):
		if best := s.aduDPBestUpdates(base, group.Name); len(best) > 0 {
			respondJSON(w, http.StatusOK, best[0])
			return
		}
		azureARMError(w, http.StatusNotFound, "NotFound", "no update is available")
	case len(seg) >= 2 && strings.EqualFold(seg[1], "deployments"):
		s.aduDPDeployments(w, r, base, group, seg[2:], aduDPDeviceClass)
	case len(seg) <= 2:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

// aduDPDeployments serves group deployments and, with subgroup set, the
// same deployments as seen by that device class subgroup.
func (s *Server) aduDPDeployments(w http.ResponseWriter, r *http.Request, base string, group *azureARMResource, seg []string, subgroup string) {
	prefix := group.ID + "/deployments"
	if len(seg) == 0 {
		if dcdevGetOnly(w, r) {
			dcdevRespondList(w, aduDPSorted(s.azureDPList(prefix, "adu/deployment")))
		}
		return
	}
	name, action, _ := strings.Cut(seg[0], ":")
	id := prefix + "/" + strings.ToLower(name)
	if len(seg) == 1 && action == "" && r.Method == http.MethodPut && subgroup == "" {
		s.aduDPPutDeployment(w, r, base, group, name, id)
		return
	}
	dep := s.azureDPGet(id)
	if dep == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "deployment not found")
		return
	}
	switch {
	case len(seg) == 1 && action != "":
		flag := map[string]string{"cancel": "isCanceled", "retry": "isRetried"}[strings.ToLower(action)]
		if r.Method != http.MethodPost || flag == "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "action must be cancel or retry.")
			return
		}
		updated := azureDPRecord(dep.ID, dep.Type, dep.Name, azureARMMerge(dep.Properties, map[string]any{flag: true}), dep)
		s.azureDPPut(updated)
		respondJSON(w, http.StatusOK, dcdevRecordView(updated))
	case len(seg) == 1 && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, dcdevRecordView(dep))
	case len(seg) == 1 && r.Method == http.MethodDelete:
		s.azureDPDelete(dep.ID)
		w.WriteHeader(http.StatusNoContent)
	case len(seg) == 2 && strings.EqualFold(seg[1], "status") && dcdevGetOnly(w, r):
		n := s.aduDPCompliance(base, group.Name)["totalDeviceCount"]
		state := "Active"
		if dep.Properties["isCanceled"] == true {
			state = "Inactive"
		}
		status := map[string]any{"groupId": group.Name, "deviceClassId": aduDPDeviceClass, "deploymentState": state, "totalDevices": n,
			"devicesInProgressCount": 0, "devicesCompletedFailedCount": 0, "devicesCompletedSucceededCount": n, "devicesCanceledCount": 0}
		if subgroup != "" {
			respondJSON(w, http.StatusOK, status)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"groupId": group.Name, "deploymentState": state, "subgroupStatus": []any{status}})
	case len(seg) == 2 && strings.EqualFold(seg[1], "deviceStates") && subgroup != "" && dcdevGetOnly(w, r):
		var out []any
		for _, d := range s.azureDPList(base+"/devices", "adu/device") {
			if strings.EqualFold(azureARMString(d.Properties["groupId"]), group.Name) {
				out = append(out, map[string]any{"deviceId": d.Name, "retryCount": 0, "movedOnToNewDeployment": false, "deviceState": "Succeeded"})
			}
		}
		dcdevRespondList(w, out)
	case len(seg) <= 2:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) aduDPPutDeployment(w http.ResponseWriter, r *http.Request, base string, group *azureARMResource, name, id string) {
	if !aduDPIDPattern.MatchString(name) {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "invalid deploymentId.")
		return
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	if !strings.EqualFold(azureARMString(payload["deploymentId"]), name) || !strings.EqualFold(azureARMString(payload["groupId"]), group.Name) {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "deploymentId and groupId must match the path.")
		return
	}
	if _, err := time.Parse(time.RFC3339, azureARMString(payload["startDateTime"])); err != nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "startDateTime must be an RFC 3339 time.")
		return
	}
	updateID := azureARMObject(azureARMObject(payload["update"])["updateId"])
	key := strings.ToLower(azureARMString(updateID["provider"]) + "/" + azureARMString(updateID["name"]) + "/" + azureARMString(updateID["version"]))
	if s.azureDPGet(base+"/updates/"+key) == nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "update.updateId must name an imported update.")
		return
	}
	dep := azureDPRecord(id, "adu/deployment", name, map[string]any{
		"deploymentId": name, "groupId": group.Name, "startDateTime": payload["startDateTime"],
		"update": map[string]any{"updateId": updateID}, "isCanceled": false, "isRetried": false, "isCloudInitiatedRollback": false,
		"deviceClassSubgroups": []any{aduDPDeviceClass}, "rollbackPolicy": payload["rollbackPolicy"],
	}, s.azureDPGet(id))
	if _, ok := s.azureDPPut(dep); !ok {
		azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
		return
	}
	respondJSON(w, http.StatusOK, dcdevRecordView(dep))
}

func (s *Server) aduDPDeviceClasses(w http.ResponseWriter, r *http.Request, base string, seg []string) {
	if len(seg) == 0 {
		if dcdevGetOnly(w, r) {
			dcdevRespondList(w, aduDPSorted(s.azureDPList(base+"/deviceclasses", "adu/deviceClass")))
		}
		return
	}
	class := s.azureDPGet(base + "/deviceclasses/" + strings.ToLower(seg[0]))
	if class == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "device class not found")
		return
	}
	switch {
	case len(seg) == 2 && strings.EqualFold(seg[1], "installableUpdates") && dcdevGetOnly(w, r):
		var out []any
		for _, u := range s.azureDPList(base+"/updates", "adu/update") {
			out = append(out, map[string]any{"updateId": u.Properties["updateId"], "description": u.Properties["description"]})
		}
		dcdevRespondList(w, out)
	case len(seg) != 1:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	case r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, dcdevRecordView(class))
	case r.Method == http.MethodPatch:
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		name := strings.TrimSpace(azureARMString(payload["friendlyName"]))
		if name == "" || len(name) > 100 {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "friendlyName is required and at most 100 characters.")
			return
		}
		updated := azureDPRecord(class.ID, class.Type, class.Name, azureARMMerge(class.Properties, map[string]any{"friendlyName": name}), class)
		s.azureDPPut(updated)
		respondJSON(w, http.StatusOK, dcdevRecordView(updated))
	case r.Method == http.MethodDelete:
		s.azureDPDelete(class.ID)
		w.WriteHeader(http.StatusNoContent)
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) aduDPDiagnostics(w http.ResponseWriter, r *http.Request, base string, seg []string) {
	if len(seg) == 0 {
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
		return
	}
	switch strings.ToLower(seg[0]) {
	case "devicehealth":
		if !dcdevGetOnly(w, r) {
			return
		}
		if strings.TrimSpace(r.URL.Query().Get("filter")) == "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "the filter query parameter is required.")
			return
		}
		var out []any
		for _, d := range s.azureDPList(base+"/devices", "adu/device") {
			out = append(out, map[string]any{"deviceId": d.Name, "state": "healthy", "digitalTwinModelId": "dtmi:azure:iot:deviceUpdateModel;2",
				"healthChecks": []any{map[string]any{"name": "AdsService", "result": "success"}}})
		}
		dcdevRespondList(w, out)
	case "logcollections":
		s.aduDPLogCollections(w, r, base, seg[1:])
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) aduDPLogCollections(w http.ResponseWriter, r *http.Request, base string, seg []string) {
	prefix := base + "/logcollections"
	if len(seg) == 1 && r.Method == http.MethodPut {
		if !aduDPIDPattern.MatchString(seg[0]) {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "invalid log collection id.")
			return
		}
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		devices, _ := payload["deviceList"].([]any)
		if len(devices) == 0 || len(devices) > aduDPMaxDevices {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "deviceList must list 1 to 100 devices.")
			return
		}
		for _, d := range devices {
			if azureARMString(azureARMObject(d)["deviceId"]) == "" {
				azureARMError(w, http.StatusBadRequest, "InvalidRequest", "deviceList[].deviceId is required.")
				return
			}
		}
		now := time.Now().UTC().Format(time.RFC3339)
		rec := azureDPRecord(prefix+"/"+strings.ToLower(seg[0]), "adu/logCollection", seg[0], map[string]any{
			"operationId": seg[0], "deviceList": devices, "description": payload["description"], "status": "Succeeded",
			"createdDateTime": now, "lastActionDateTime": now,
		}, nil)
		if _, ok := s.azureDPPut(rec); !ok {
			azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
			return
		}
		respondJSON(w, http.StatusCreated, dcdevRecordView(rec))
		return
	}
	if len(seg) == 2 && strings.EqualFold(seg[1], "detailedStatus") {
		if !dcdevGetOnly(w, r) {
			return
		}
		rec := s.azureDPGet(prefix + "/" + strings.ToLower(seg[0]))
		if rec == nil {
			azureARMError(w, http.StatusNotFound, "NotFound", "log collection not found")
			return
		}
		devices, _ := rec.Properties["deviceList"].([]any)
		var status []any
		for _, d := range devices {
			dev := azureARMObject(d)
			status = append(status, map[string]any{"deviceId": dev["deviceId"], "moduleId": dev["moduleId"], "status": "Succeeded",
				"resultCode": "200", "extendedResultCode": "0", "logLocation": "arn://logs/" + azureARMString(dev["deviceId"])})
		}
		respondJSON(w, http.StatusOK, map[string]any{"operationId": rec.Name, "status": "Succeeded", "description": rec.Properties["description"],
			"createdDateTime": rec.Properties["createdDateTime"], "lastActionDateTime": rec.Properties["lastActionDateTime"], "deviceStatus": status})
		return
	}
	s.dcdevChildren(w, r, prefix, "adu/logCollection", seg)
}
