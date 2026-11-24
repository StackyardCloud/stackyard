package server

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Dev boxes and environments of the Dev Center developer data plane (see
// provider_azure_devcenter_developer_2025_02_01.go).

const dcdevMaxTasks = 100

var dcdevNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_.]{2,62}$`)

func dcdevUserID(project *azureARMResource, user string) string {
	return dcdevBase + strings.ToLower(project.Name) + "/users/" + strings.ToLower(user)
}

func (s *Server) dcdevListDevBoxes(w http.ResponseWriter, r *http.Request, prefix, user string) {
	if !dcdevGetOnly(w, r) {
		return
	}
	if prefix == "" {
		prefix = strings.TrimSuffix(dcdevBase, "/")
	}
	var out []any
	for _, box := range s.azureDPList(prefix, "dcdev/devbox") {
		if user == "" || strings.EqualFold(azureARMString(box.Properties["user"]), user) {
			out = append(out, dcdevRecordView(box))
		}
	}
	dcdevRespondList(w, out)
}

// dcdevStartOperation records an operation that already Succeeded on the
// resource at resID and points Operation-Location at its project status.
func (s *Server) dcdevStartOperation(w http.ResponseWriter, project *azureARMResource, resID, kind string) bool {
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	op := azureDPRecord(resID+"/operations/"+id, "dcdev/operation", id, map[string]any{
		"operationId": id, "kind": kind, "status": "Succeeded", "startTime": now, "endTime": now,
		"uri": azureDevCenterDeveloper20250201Prefix + "/projects/" + project.Name + "/operationstatuses/" + id,
	}, nil)
	if _, ok := s.azureDPPut(op); !ok {
		azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
		return false
	}
	location := azureDevCenterDeveloper20250201Prefix + "/projects/" + project.Name + "/operationstatuses/" + id + "?api-version=" + azureDevCenterDeveloper20250201APIVersion
	w.Header().Set("Operation-Location", location)
	w.Header().Set("Location", location)
	return true
}

func (s *Server) dcdevDevBoxes(w http.ResponseWriter, r *http.Request, project *azureARMResource, user string, seg []string) {
	base := dcdevUserID(project, user) + "/devboxes"
	if len(seg) == 0 {
		s.dcdevListDevBoxes(w, r, base, "")
		return
	}
	name, action, _ := strings.Cut(seg[0], ":")
	id := base + "/" + strings.ToLower(name)
	if len(seg) == 1 && action == "" && r.Method == http.MethodPut {
		s.dcdevPutDevBox(w, r, project, user, name, id)
		return
	}
	box := s.azureDPGet(id)
	if box == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "dev box not found")
		return
	}
	if len(seg) == 1 {
		switch {
		case action != "":
			s.dcdevDevBoxAction(w, r, project, box, action)
		case r.Method == http.MethodGet:
			respondJSON(w, http.StatusOK, dcdevRecordView(box))
		case r.Method == http.MethodDelete:
			if s.dcdevStartOperation(w, project, box.ID, "Delete") {
				s.azureDPDelete(box.ID)
				w.WriteHeader(http.StatusAccepted)
			}
		default:
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		}
		return
	}
	sub, subAction, _ := strings.Cut(seg[1], ":")
	switch strings.ToLower(sub) {
	case "remoteconnection":
		if dcdevGetOnly(w, r) {
			respondJSON(w, http.StatusOK, map[string]any{
				"webUrl":           "https://devbox.microsoft.com/connect?devbox=" + box.Name,
				"rdpConnectionUrl": "ms-cloudpc:connect?devbox=" + box.Name,
			})
		}
	case "actions":
		s.dcdevActions(w, r, box.ID, seg[2:], subAction)
	case "operations":
		s.dcdevOperations(w, r, box.ID, seg[2:])
	case "snapshots":
		s.dcdevChildren(w, r, box.ID+"/snapshots", "dcdev/snapshot", seg[2:])
	case "customizationgroups":
		s.dcdevCustomizationGroups(w, r, box.ID, seg[2:])
	case "imagingtasks":
		if len(seg) == 4 && strings.EqualFold(seg[3], "logs") && dcdevGetOnly(w, r) {
			dcdevRespondLog(w)
			return
		}
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) dcdevPutDevBox(w http.ResponseWriter, r *http.Request, project *azureARMResource, user, name, id string) {
	if !dcdevNamePattern.MatchString(name) || len(user) > 128 {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "dev box names are 3-63 letters, digits, '-', '_' or '.'.")
		return
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	pool := dcdevNamed(s.dcdevARMUnder(project.ID, "projects/pools"), azureARMString(payload["poolName"]))
	if pool == nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "poolName must name a pool of the project.")
		return
	}
	existing := s.azureDPGet(id)
	view := dcdevPoolView(pool)
	box := azureDPRecord(id, "dcdev/devbox", name, map[string]any{
		"projectName": project.Name, "poolName": pool.Name, "user": user, "location": pool.Location,
		"provisioningState": "Succeeded", "actionState": "Started", "powerState": "Running", "osType": "Windows",
		"uniqueId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(id))).String(), "hibernateSupport": view["hibernateSupport"],
		"hardwareProfile": view["hardwareProfile"], "storageProfile": view["storageProfile"], "localAdministrator": view["localAdministrator"],
		"imageReference": map[string]any{"name": dcadmGalleryImages[0], "version": "1.0.0", "operatingSystem": "Windows11"},
		"createdTime":    time.Now().UTC().Format(time.RFC3339),
	}, existing)
	if existing != nil {
		box.Properties["createdTime"] = existing.Properties["createdTime"]
	}
	created, ok := s.azureDPPut(box)
	if !ok {
		azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
		return
	}
	if created && len(s.dcdevARMUnder(pool.ID, "projects/pools/schedules")) > 0 {
		next := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour).Format(time.RFC3339)
		s.azureDPPut(azureDPRecord(id+"/actions/schedule-default", "dcdev/action", "schedule-default", map[string]any{
			"actionType": "Stop", "sourceId": "default", "next": map[string]any{"scheduledTime": next},
		}, nil))
	}
	if !s.dcdevStartOperation(w, project, id, "Create") {
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, dcdevRecordView(box))
}

var dcdevPowerActions = map[string][2]string{
	"start": {"Running", "Started"}, "stop": {"Deallocated", "Stopped"}, "restart": {"Running", "Started"}, "repair": {"Running", "Repaired"},
}

func (s *Server) dcdevDevBoxAction(w http.ResponseWriter, r *http.Request, project *azureARMResource, box *azureARMResource, action string) {
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	lower := strings.ToLower(action)
	switch {
	case dcdevPowerActions[lower] != [2]string{}:
		state := dcdevPowerActions[lower]
		if lower == "stop" && r.URL.Query().Get("hibernate") == "true" {
			state[0] = "Hibernated"
		}
		props := azureARMMerge(box.Properties, map[string]any{"powerState": state[0], "actionState": state[1]})
		s.azureDPPut(azureDPRecord(box.ID, box.Type, box.Name, props, box))
	case lower == "capturesnapshot":
		id := uuid.NewString()
		if _, ok := s.azureDPPut(azureDPRecord(box.ID+"/snapshots/"+id, "dcdev/snapshot", id, map[string]any{
			"snapshotId": id, "createdTime": time.Now().UTC().Format(time.RFC3339), "snapshotType": "Manual", "isImmutable": false,
		}, nil)); !ok {
			azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
			return
		}
	case lower == "restoresnapshot":
		if s.azureDPGet(box.ID+"/snapshots/"+strings.ToLower(r.URL.Query().Get("snapshotId"))) == nil {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "snapshotId must name a snapshot of the dev box.")
			return
		}
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "unknown dev box action")
		return
	}
	if s.dcdevStartOperation(w, project, box.ID, action) {
		w.WriteHeader(http.StatusAccepted)
	}
}

// dcdevActions serves scheduled actions: list, get, skip and delay, plus
// actions:delay, which delays every action of the resource.
func (s *Server) dcdevActions(w http.ResponseWriter, r *http.Request, resID string, seg []string, collectionAction string) {
	actions := s.azureDPList(resID+"/actions", "dcdev/action")
	if strings.EqualFold(collectionAction, "delay") || (len(seg) == 1 && strings.Contains(seg[0], ":")) {
		if r.Method != http.MethodPost {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return
		}
		until, err := time.Parse(time.RFC3339, r.URL.Query().Get("until"))
		name, verb, _ := strings.Cut(strings.Join(seg, "/"), ":")
		if collectionAction == "" && !strings.EqualFold(verb, "skip") && err != nil {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "until must be an RFC 3339 time.")
			return
		}
		if collectionAction != "" && err != nil {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "until must be an RFC 3339 time.")
			return
		}
		var results []any
		for _, act := range actions {
			if collectionAction == "" && !strings.EqualFold(act.Name, name) {
				continue
			}
			next := map[string]any{"scheduledTime": until.UTC().Format(time.RFC3339)}
			if strings.EqualFold(verb, "skip") {
				next = nil
			}
			props := azureARMMerge(act.Properties, map[string]any{"next": next})
			updated := azureDPRecord(act.ID, act.Type, act.Name, props, act)
			s.azureDPPut(updated)
			results = append(results, map[string]any{"name": act.Name, "result": "Succeeded", "action": dcdevRecordView(updated)})
		}
		switch {
		case collectionAction != "":
			dcdevRespondList(w, results)
		case len(results) == 0:
			azureARMError(w, http.StatusNotFound, "NotFound", "action not found")
		case strings.EqualFold(verb, "skip"):
			w.WriteHeader(http.StatusNoContent)
		default:
			respondJSON(w, http.StatusOK, results[0].(map[string]any)["action"])
		}
		return
	}
	s.dcdevChildren(w, r, resID+"/actions", "dcdev/action", seg)
}

func (s *Server) dcdevOperations(w http.ResponseWriter, r *http.Request, resID string, seg []string) {
	if len(seg) == 2 && strings.EqualFold(seg[1], "logs") {
		if !dcdevGetOnly(w, r) {
			return
		}
		if s.azureDPGet(resID+"/operations/"+strings.ToLower(seg[0])) == nil {
			azureARMError(w, http.StatusNotFound, "NotFound", "operation not found")
			return
		}
		dcdevRespondLog(w)
		return
	}
	s.dcdevChildren(w, r, resID+"/operations", "dcdev/operation", seg)
}

// dcdevChildren serves GET list and GET item for records of typ under prefix.
func (s *Server) dcdevChildren(w http.ResponseWriter, r *http.Request, prefix, typ string, seg []string) {
	if !dcdevGetOnly(w, r) {
		return
	}
	switch len(seg) {
	case 0:
		dcdevRespondList(w, dcdevViews(s.azureDPList(prefix, typ), dcdevRecordView))
	case 1:
		if rec := s.azureDPGet(prefix + "/" + strings.ToLower(seg[0])); rec != nil && rec.Type == typ {
			respondJSON(w, http.StatusOK, dcdevRecordView(rec))
			return
		}
		azureARMError(w, http.StatusNotFound, "NotFound", "resource not found")
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) dcdevCustomizationGroups(w http.ResponseWriter, r *http.Request, boxID string, seg []string) {
	prefix := boxID + "/customizationgroups"
	if len(seg) == 3 && strings.EqualFold(seg[1], "logs") {
		if dcdevGetOnly(w, r) {
			if s.azureDPGet(prefix+"/"+strings.ToLower(seg[0])) == nil {
				azureARMError(w, http.StatusNotFound, "NotFound", "customization group not found")
				return
			}
			dcdevRespondLog(w)
		}
		return
	}
	if len(seg) != 1 || r.Method != http.MethodPut {
		s.dcdevChildren(w, r, prefix, "dcdev/customizationGroup", seg)
		return
	}
	if !dcdevNamePattern.MatchString(seg[0]) {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "invalid customization group name.")
		return
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	tasks, _ := payload["tasks"].([]any)
	if len(tasks) == 0 || len(tasks) > dcdevMaxTasks {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "tasks must list 1 to 100 tasks.")
		return
	}
	var out []any
	for i, task := range tasks {
		t := azureARMObject(task)
		name := azureARMString(t["name"])
		if dcdevTaskDefinition(name) == nil {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", "unknown customization task "+name)
			return
		}
		taskID := strings.ToLower(seg[0]) + "-" + string(rune('a'+i%26))
		out = append(out, map[string]any{"name": name, "parameters": t["parameters"], "displayName": t["displayName"], "id": taskID, "status": "Succeeded",
			"logUri": "customizationGroups/" + seg[0] + "/logs/" + taskID})
	}
	now := time.Now().UTC().Format(time.RFC3339)
	group := azureDPRecord(prefix+"/"+strings.ToLower(seg[0]), "dcdev/customizationGroup", seg[0], map[string]any{
		"status": "Succeeded", "tasks": out, "startTime": now, "endTime": now,
	}, nil)
	if _, ok := s.azureDPPut(group); !ok {
		azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
		return
	}
	respondJSON(w, http.StatusCreated, dcdevRecordView(group))
}

// dcdevRespondLog writes the fixed log every finished task and operation has.
func dcdevRespondLog(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Started.\nCompleted successfully.\n"))
}

func (s *Server) dcdevEnvironments(w http.ResponseWriter, r *http.Request, project *azureARMResource, user string, seg []string) {
	base := dcdevUserID(project, user) + "/environments"
	if len(seg) == 0 {
		s.dcdevChildren(w, r, base, "dcdev/environment", nil)
		return
	}
	id := base + "/" + strings.ToLower(seg[0])
	if len(seg) == 1 && r.Method == http.MethodPut {
		s.dcdevPutEnvironment(w, r, project, user, seg[0], id)
		return
	}
	env := s.azureDPGet(id)
	if env == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "environment not found")
		return
	}
	if len(seg) == 1 {
		switch r.Method {
		case http.MethodGet:
			respondJSON(w, http.StatusOK, dcdevRecordView(env))
		case http.MethodPatch:
			payload, ok := azureARMReadBody(w, r)
			if !ok {
				return
			}
			patch := map[string]any{}
			if exp, present := payload["expirationDate"]; present {
				if _, err := time.Parse(time.RFC3339, azureARMString(exp)); exp != nil && err != nil {
					azureARMError(w, http.StatusBadRequest, "InvalidRequest", "expirationDate must be an RFC 3339 time.")
					return
				}
				patch["expirationDate"] = exp
			}
			updated := azureDPRecord(env.ID, env.Type, env.Name, azureARMMerge(env.Properties, patch), env)
			s.azureDPPut(updated)
			respondJSON(w, http.StatusOK, dcdevRecordView(updated))
		case http.MethodDelete:
			if s.dcdevStartOperation(w, project, env.ID, "Delete") {
				s.azureDPDelete(env.ID)
				w.WriteHeader(http.StatusAccepted)
			}
		default:
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		}
		return
	}
	sub, subAction, _ := strings.Cut(seg[1], ":")
	switch strings.ToLower(sub) {
	case "outputs":
		if dcdevGetOnly(w, r) {
			respondJSON(w, http.StatusOK, map[string]any{"outputs": map[string]any{}})
		}
	case "operations":
		s.dcdevOperations(w, r, env.ID, seg[2:])
	case "actions":
		s.dcdevActions(w, r, env.ID, seg[2:], subAction)
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) dcdevPutEnvironment(w http.ResponseWriter, r *http.Request, project *azureARMResource, user, name, id string) {
	if !dcdevNamePattern.MatchString(name) || len(user) > 128 {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "environment names are 3-63 letters, digits, '-', '_' or '.'.")
		return
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	for _, key := range []string{"environmentType", "catalogName", "environmentDefinitionName"} {
		if strings.TrimSpace(azureARMString(payload[key])) == "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequest", key+" is required.")
			return
		}
	}
	envType := dcdevNamed(s.dcdevARMUnder(project.ID, "projects/environmentTypes"), azureARMString(payload["environmentType"]))
	if envType == nil || dcdevNamed(s.dcdevDefinitions(project, azureARMString(payload["catalogName"])), azureARMString(payload["environmentDefinitionName"])) == nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "environmentType, catalogName and environmentDefinitionName must name resources of the project.")
		return
	}
	params := azureARMObject(payload["parameters"])
	if payload["parameters"] != nil && params == nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequest", "parameters must be an object.")
		return
	}
	target := azureARMString(envType.Properties["deploymentTargetId"])
	existing := s.azureDPGet(id)
	env := azureDPRecord(id, "dcdev/environment", name, map[string]any{
		"environmentType": envType.Name, "user": user, "provisioningState": "Succeeded",
		"catalogName": payload["catalogName"], "environmentDefinitionName": payload["environmentDefinitionName"], "parameters": params,
		"expirationDate":  payload["expirationDate"],
		"resourceGroupId": strings.TrimSuffix(target, "/") + "/resourceGroups/" + strings.ToLower(project.Name) + "-" + strings.ToLower(name),
	}, existing)
	created, ok := s.azureDPPut(env)
	if !ok {
		azureARMError(w, http.StatusConflict, "QuotaExceeded", "too many data-plane records")
		return
	}
	if exp := azureARMString(payload["expirationDate"]); exp != "" {
		s.azureDPPut(azureDPRecord(id+"/actions/expiration", "dcdev/action", "expiration", map[string]any{
			"actionType": "Delete", "sourceId": "expirationDate", "next": map[string]any{"scheduledTime": exp},
		}, nil))
	}
	if !s.dcdevStartOperation(w, project, id, "Deploy") {
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, dcdevRecordView(env))
}
