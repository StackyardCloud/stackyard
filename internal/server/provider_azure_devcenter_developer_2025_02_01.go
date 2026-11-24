package server

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureDevCenterDeveloper20250201APIVersion = "2025-02-01"
const azureDevCenterDeveloper20250201Prefix = "/azure/devcenter-developer/2025-02-01"

// Dev Center developer data plane. Projects, pools, schedules, catalogs,
// environment definitions and environment types are read from the
// Microsoft.DevCenter management resources (provider_azure_devcenter_
// administrator_2024_02_01.go); dev boxes, environments and their operations,
// actions, snapshots and customization groups live in the data-plane store.
// Every long-running operation finishes at once with status Succeeded.
func (s *Server) handleAzureDevCenterDeveloper20250201Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureDevCenterDeveloper20250201Prefix) {
		return false
	}
	if r.URL.Query().Get("api-version") != azureDevCenterDeveloper20250201APIVersion {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "api-version must be 2025-02-01.")
		return true
	}
	seg := splitPathSegments(strings.Trim(strings.TrimPrefix(path, azureDevCenterDeveloper20250201Prefix), "/"))
	switch {
	case len(seg) == 1 && strings.EqualFold(seg[0], "devboxes"):
		s.dcdevListDevBoxes(w, r, "", "")
	case len(seg) == 3 && strings.EqualFold(seg[0], "users") && strings.EqualFold(seg[2], "devboxes"):
		s.dcdevListDevBoxes(w, r, "", seg[1])
	case len(seg) >= 1 && strings.EqualFold(seg[0], "projects"):
		s.dcdevProjects(w, r, seg[1:])
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
	return true
}

const dcdevBase = "/dataplane/devcenter/projects/"

func (s *Server) dcdevProjects(w http.ResponseWriter, r *http.Request, seg []string) {
	if len(seg) == 0 {
		if !dcdevGetOnly(w, r) {
			return
		}
		var out []any
		for _, p := range s.azureARMResourcesWhere(func(rec *azureARMResource) bool { return strings.EqualFold(rec.Type, dcadmNamespace+"/projects") }) {
			out = append(out, dcdevProjectView(p))
		}
		dcdevRespondList(w, out)
		return
	}
	project := s.dcdevProject(seg[0])
	if project == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "project not found")
		return
	}
	if len(seg) == 1 {
		if dcdevGetOnly(w, r) {
			respondJSON(w, http.StatusOK, dcdevProjectView(project))
		}
		return
	}
	name, action, _ := strings.Cut(seg[1], ":")
	rest := seg[2:]
	switch strings.ToLower(name) {
	case "pools":
		s.dcdevPools(w, r, project, rest)
	case "schedules":
		if dcdevGetOnly(w, r) && len(rest) == 0 {
			dcdevRespondList(w, dcdevViews(s.dcdevARMUnder(project.ID, "projects/pools/schedules"), dcdevScheduleView))
		}
	case "catalogs":
		s.dcdevCatalogs(w, r, project, rest)
	case "environmentdefinitions":
		if dcdevGetOnly(w, r) && len(rest) == 0 {
			dcdevRespondList(w, dcdevViews(s.dcdevDefinitions(project, ""), dcdevDefinitionView))
		}
	case "environmenttypes":
		if dcdevGetOnly(w, r) && len(rest) == 0 {
			dcdevRespondList(w, dcdevViews(s.dcdevARMUnder(project.ID, "projects/environmentTypes"), dcdevEnvironmentTypeView))
		}
	case "environments":
		if dcdevGetOnly(w, r) && len(rest) == 0 {
			dcdevRespondList(w, dcdevViews(s.azureDPList(dcdevBase+strings.ToLower(project.Name), "dcdev/environment"), dcdevRecordView))
		}
	case "operationstatuses":
		s.dcdevOperationStatus(w, r, project, rest)
	case "customizationtasks":
		s.dcdevCustomizationTasks(w, r, project, rest, action)
	case "users":
		s.dcdevUser(w, r, project, rest)
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) dcdevUser(w http.ResponseWriter, r *http.Request, project *azureARMResource, seg []string) {
	if len(seg) < 2 {
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
		return
	}
	user, rest := seg[0], seg[1:]
	switch {
	case len(rest) == 1 && strings.EqualFold(rest[0], "abilities"):
		if dcdevGetOnly(w, r) {
			respondJSON(w, http.StatusOK, map[string]any{
				"abilitiesAsAdmin":     []any{"DeleteDevBoxes", "ManageDevBoxActions", "StartDevBoxes", "StopDevBoxes"},
				"abilitiesAsDeveloper": []any{"CustomizeDevBoxes", "DeleteDevBoxes", "ManageDevBoxActions", "ReadDevBoxes", "StartDevBoxes", "StopDevBoxes", "WriteDevBoxes"},
			})
		}
	case len(rest) == 3 && strings.EqualFold(rest[0], "environmentTypes") && strings.EqualFold(rest[2], "abilities"):
		if !dcdevGetOnly(w, r) {
			return
		}
		if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			return strings.EqualFold(rec.ID, project.ID+"/environmentTypes/"+rest[1])
		})) == 0 {
			azureARMError(w, http.StatusNotFound, "NotFound", "environment type not found")
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"abilitiesAsAdmin":     []any{"DeleteEnvironments", "ManageEnvironmentActions"},
			"abilitiesAsDeveloper": []any{"CustomizeEnvironments", "DeleteEnvironments", "ManageEnvironmentActions", "ReadEnvironmentOutputs", "ReadEnvironments", "WriteEnvironments"},
		})
	case strings.EqualFold(rest[0], "environments"):
		s.dcdevEnvironments(w, r, project, user, rest[1:])
	case strings.HasPrefix(strings.ToLower(rest[0]), "devboxes"):
		s.dcdevDevBoxes(w, r, project, user, rest[1:])
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) dcdevPools(w http.ResponseWriter, r *http.Request, project *azureARMResource, seg []string) {
	if !dcdevGetOnly(w, r) {
		return
	}
	pools := s.dcdevARMUnder(project.ID, "projects/pools")
	if len(seg) == 0 {
		dcdevRespondList(w, dcdevViews(pools, dcdevPoolView))
		return
	}
	pool := dcdevNamed(pools, seg[0])
	if pool == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "pool not found")
		return
	}
	switch {
	case len(seg) == 1:
		respondJSON(w, http.StatusOK, dcdevPoolView(pool))
	case strings.EqualFold(seg[1], "schedules"):
		schedules := s.dcdevARMUnder(pool.ID, "projects/pools/schedules")
		if len(seg) == 2 {
			dcdevRespondList(w, dcdevViews(schedules, dcdevScheduleView))
			return
		}
		if sched := dcdevNamed(schedules, seg[2]); sched != nil {
			respondJSON(w, http.StatusOK, dcdevScheduleView(sched))
			return
		}
		azureARMError(w, http.StatusNotFound, "NotFound", "schedule not found")
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

func (s *Server) dcdevCatalogs(w http.ResponseWriter, r *http.Request, project *azureARMResource, seg []string) {
	if !dcdevGetOnly(w, r) {
		return
	}
	catalogs := s.dcdevCatalogList(project)
	if len(seg) == 0 {
		dcdevRespondList(w, dcdevViews(catalogs, dcdevNameView))
		return
	}
	catalog := dcdevNamed(catalogs, seg[0])
	if catalog == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "catalog not found")
		return
	}
	switch {
	case len(seg) == 1:
		respondJSON(w, http.StatusOK, dcdevNameView(catalog))
	case strings.EqualFold(seg[1], "environmentDefinitions"):
		defs := s.dcdevDefinitions(project, catalog.Name)
		if len(seg) == 2 {
			dcdevRespondList(w, dcdevViews(defs, dcdevDefinitionView))
			return
		}
		if def := dcdevNamed(defs, seg[2]); def != nil {
			respondJSON(w, http.StatusOK, dcdevDefinitionView(def))
			return
		}
		azureARMError(w, http.StatusNotFound, "NotFound", "environment definition not found")
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
	}
}

// dcdevCustomizationTasks serves the catalog task definitions every project
// offers and validates customization groups against them.
func (s *Server) dcdevCustomizationTasks(w http.ResponseWriter, r *http.Request, project *azureARMResource, seg []string, action string) {
	if strings.EqualFold(action, "validateGroup") {
		if r.Method != http.MethodPost {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
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
		var errs []any
		for _, task := range tasks {
			if name := azureARMString(azureARMObject(task)["name"]); dcdevTaskDefinition(name) == nil {
				errs = append(errs, map[string]any{"target": "tasks", "message": "unknown task " + name})
			}
		}
		status := "Succeeded"
		if len(errs) > 0 {
			status = "Failed"
		}
		respondJSON(w, http.StatusOK, map[string]any{"status": status, "validationResult": map[string]any{"status": status, "errors": errs}})
		return
	}
	if !dcdevGetOnly(w, r) {
		return
	}
	if len(seg) == 0 {
		var out []any
		for _, def := range dcdevTaskDefinitions {
			out = append(out, def)
		}
		dcdevRespondList(w, out)
		return
	}
	if def := dcdevTaskDefinition(seg[0]); def != nil {
		respondJSON(w, http.StatusOK, def)
		return
	}
	azureARMError(w, http.StatusNotFound, "NotFound", "customization task not found")
}

var dcdevTaskDefinitions = []map[string]any{
	{"name": "winget", "catalogName": "builtin", "description": "Installs a package with winget.", "parameters": map[string]any{"package": map[string]any{"type": "string", "required": true}}},
	{"name": "powershell", "catalogName": "builtin", "description": "Runs a PowerShell command.", "parameters": map[string]any{"command": map[string]any{"type": "string", "required": true}}},
}

func dcdevTaskDefinition(name string) map[string]any {
	name = strings.TrimPrefix(strings.ToLower(name), "builtin/")
	for _, def := range dcdevTaskDefinitions {
		if def["name"] == name {
			return def
		}
	}
	return nil
}

func (s *Server) dcdevOperationStatus(w http.ResponseWriter, r *http.Request, project *azureARMResource, seg []string) {
	if !dcdevGetOnly(w, r) || len(seg) != 1 {
		if len(seg) != 1 {
			azureARMError(w, http.StatusNotFound, "NotFound", "route not found")
		}
		return
	}
	for _, op := range s.azureDPList(dcdevBase+strings.ToLower(project.Name), "dcdev/operation") {
		if strings.EqualFold(op.Name, seg[0]) {
			respondJSON(w, http.StatusOK, dcdevRecordView(op))
			return
		}
	}
	azureARMError(w, http.StatusNotFound, "NotFound", "operation not found")
}

// --- management-resource lookups and views ---

func (s *Server) dcdevProject(name string) *azureARMResource {
	projects := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, dcadmNamespace+"/projects") && strings.EqualFold(rec.Name, name)
	})
	if len(projects) == 0 {
		return nil
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].ID < projects[j].ID })
	return projects[0]
}

// dcdevARMUnder lists management records of typ under parentID.
func (s *Server) dcdevARMUnder(parentID, typ string) []*azureARMResource {
	prefix := strings.ToLower(parentID) + "/"
	recs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, dcadmNamespace+"/"+typ) && strings.HasPrefix(strings.ToLower(rec.ID), prefix)
	})
	sort.Slice(recs, func(i, j int) bool { return recs[i].ID < recs[j].ID })
	return recs
}

func (s *Server) dcdevCatalogList(project *azureARMResource) []*azureARMResource {
	catalogs := s.dcdevARMUnder(project.ID, "projects/catalogs")
	if dc := azureARMString(project.Properties["devCenterId"]); dc != "" {
		catalogs = append(catalogs, s.dcdevARMUnder(dc, "devcenters/catalogs")...)
	}
	return catalogs
}

func (s *Server) dcdevDefinitions(project *azureARMResource, catalog string) []*azureARMResource {
	var out []*azureARMResource
	for _, c := range s.dcdevCatalogList(project) {
		if catalog != "" && !strings.EqualFold(c.Name, catalog) {
			continue
		}
		typ := "devcenters/catalogs/environmentDefinitions"
		if strings.Contains(strings.ToLower(c.Type), "/projects/") {
			typ = "projects/catalogs/environmentDefinitions"
		}
		out = append(out, s.dcdevARMUnder(c.ID, typ)...)
	}
	return out
}

func dcdevNamed(recs []*azureARMResource, name string) *azureARMResource {
	for _, rec := range recs {
		if strings.EqualFold(rec.Name, name) {
			return rec
		}
	}
	return nil
}

func dcdevViews(recs []*azureARMResource, view func(*azureARMResource) map[string]any) []any {
	out := make([]any, 0, len(recs))
	for _, rec := range recs {
		out = append(out, view(rec))
	}
	return out
}

func dcdevProjectView(p *azureARMResource) map[string]any {
	return map[string]any{
		"name": p.Name, "description": p.Properties["description"], "maxDevBoxesPerUser": p.Properties["maxDevBoxesPerUser"],
		"uri": azureDevCenterDeveloper20250201Prefix + "/projects/" + p.Name,
	}
}

func dcdevNameView(rec *azureARMResource) map[string]any { return map[string]any{"name": rec.Name} }

func dcdevPoolView(rec *azureARMResource) map[string]any {
	return map[string]any{
		"name": rec.Name, "location": rec.Location, "osType": "Windows", "localAdministrator": rec.Properties["localAdministrator"],
		"hibernateSupport": "Disabled", "healthStatus": "Healthy", "devBoxCount": 0,
		"hardwareProfile":  map[string]any{"skuName": "general_i_8c32gb256ssd_v2", "vCPUs": 8, "memoryGB": 32},
		"storageProfile":   map[string]any{"osDisk": map[string]any{"diskSizeGB": 256}},
		"stopOnDisconnect": rec.Properties["stopOnDisconnect"],
	}
}

func dcdevScheduleView(rec *azureARMResource) map[string]any {
	return map[string]any{
		"name": rec.Name, "type": rec.Properties["type"], "frequency": rec.Properties["frequency"],
		"time": rec.Properties["time"], "timeZone": rec.Properties["timeZone"],
	}
}

func dcdevDefinitionView(rec *azureARMResource) map[string]any {
	parts := strings.Split(rec.ID, "/")
	return map[string]any{
		"id": "/catalogs/" + parts[len(parts)-3] + "/environmentDefinitions/" + rec.Name, "name": rec.Name,
		"catalogName": parts[len(parts)-3], "description": rec.Properties["description"], "parameters": rec.Properties["parameters"],
		"templatePath": rec.Properties["templatePath"],
	}
}

func dcdevEnvironmentTypeView(rec *azureARMResource) map[string]any {
	status := azureARMString(rec.Properties["status"])
	if status == "" {
		status = "Enabled"
	}
	return map[string]any{"name": rec.Name, "deploymentTargetId": rec.Properties["deploymentTargetId"], "status": status}
}

func dcdevGetOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return false
	}
	return true
}

func dcdevRespondList(w http.ResponseWriter, items []any) {
	if items == nil {
		items = []any{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

// --- data-plane store, shared with the Device Update data plane ---

// azureDPMaxRecords bounds the data-plane records all such services keep.
// ponytail: each write counts by scanning the store; keep a counter if the cap grows.
const azureDPMaxRecords = 20000

// azureDPPut stores rec, a data-plane record whose ID starts with /dataplane/,
// in the ARM store (no ARM route lists its types). It reports whether the
// record is new, or false for ok when the store is full.
func (s *Server) azureDPPut(rec *azureARMResource) (created, ok bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureARMKey(rec.ID)
	if s.azureARMResources[key] != nil {
		s.azureARMResources[key] = rec
		return false, true
	}
	n := 0
	for k := range s.azureARMResources {
		if strings.HasPrefix(k, "/dataplane/") {
			n++
		}
	}
	if n >= azureDPMaxRecords {
		return false, false
	}
	s.azureARMResources[key] = rec
	return true, true
}

func (s *Server) azureDPGet(id string) *azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureARMResources[azureARMKey(id)]
}

// azureDPDelete removes the record and everything under it.
func (s *Server) azureDPDelete(id string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureARMKey(id)
	if s.azureARMResources[key] == nil {
		return false
	}
	for k := range s.azureARMResources {
		if k == key || strings.HasPrefix(k, key+"/") {
			delete(s.azureARMResources, k)
		}
	}
	return true
}

// azureDPList lists records of typ anywhere under prefix, ordered by ID.
func (s *Server) azureDPList(prefix, typ string) []*azureARMResource {
	p := azureARMKey(prefix) + "/"
	recs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return rec.Type == typ && strings.HasPrefix(azureARMKey(rec.ID), p)
	})
	sort.Slice(recs, func(i, j int) bool { return recs[i].ID < recs[j].ID })
	return recs
}

// azureDPRecord builds a data-plane record; props must not be shared.
func azureDPRecord(id, typ, name string, props map[string]any, existing *azureARMResource) *azureARMResource {
	now := time.Now().UTC()
	rec := &azureARMResource{ID: id, Type: typ, Name: name, Properties: props, CreatedAt: now, UpdatedAt: now}
	if existing != nil {
		rec.CreatedAt = existing.CreatedAt
	}
	return rec
}

// dcdevRecordView returns a data-plane record's properties with its name.
func dcdevRecordView(rec *azureARMResource) map[string]any {
	out := make(map[string]any, len(rec.Properties)+1)
	for k, v := range rec.Properties {
		if !strings.HasPrefix(k, "_") {
			out[k] = v
		}
	}
	out["name"] = rec.Name
	return out
}
