package server

import (
	"encoding/base64"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const azureDefenderForEASMDataPlanePreview20241001APIVersion = "2024-10-01-preview"
const azureDefenderForEASMDataPlanePreview20241001Prefix = "/azure/defenderforeasm-dataplanepreview/2024-10-01-preview"

// Defender EASM data plane (2024-10-01-preview) for one emulated workspace:
// a seeded asset inventory with updates, exports and reports, CISA CVEs,
// data connections, discovery groups with runs, discovery templates, saved
// filters, policies and tasks. Discovery finds nothing new and tasks finish
// at once.
const (
	azureEASMDMaxItems    = 1000 // per collection
	azureEASMDMaxRuns     = 100  // per discovery group
	azureEASMDMaxSeeds    = 100
	azureEASMDMaxPageSize = 1000
)

var (
	azureEASMDNameRE        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	azureEASMDAssetKinds    = []string{"as", "contact", "domain", "host", "ipAddress", "ipBlock", "page", "sslCert"}
	azureEASMDStates        = []string{"candidate", "confirmed", "dismissed", "candidateInvestigate", "associatedPartner", "associatedThirdparty", "archived"}
	azureEASMDConnKinds     = []string{"logAnalytics", "azureDataExplorer"}
	azureEASMDFrequencies   = []string{"daily", "weekly", "monthly"}
	azureEASMDContents      = []string{"assets", "attackSurfaceInsights"}
	azureEASMDPolicyActions = []string{"addResource", "removeResource", "setState", "setExternalId", "addLabel", "removeLabel"}
	// azureEASMDClauseRE matches one filter clause: field op value.
	azureEASMDClauseRE = regexp.MustCompile(`^\s*([A-Za-z]+)\s*(=|!=|\^=|IN)\s*(.+?)\s*$`)
	azureEASMDAndRE    = regexp.MustCompile(`(?i)\s+AND\s+`)
)

type azureEASMDAsset struct {
	Kind, Name, State, ExternalID string
	Labels                        []string
	Created, Updated              time.Time
}

func (a azureEASMDAsset) id() string {
	return base64.RawURLEncoding.EncodeToString([]byte(a.Kind + "$$" + a.Name))
}

type azureEASMDStore struct {
	mu          sync.Mutex
	assets      []*azureEASMDAsset
	connections map[string]map[string]any
	groups      map[string]map[string]any
	runs        map[string][]map[string]any
	filters     map[string]map[string]any
	policies    map[string]map[string]any
	tasks       map[string]map[string]any
	taskOrder   []string
}

// azureEASMDState returns the server's EASM data plane store, seeded with
// a small Contoso inventory on first use.
func (s *Server) azureEASMDState() *azureEASMDStore {
	return azureDMSStateFor(s, "easm-dataplane", func() *azureEASMDStore {
		seeded := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
		st := &azureEASMDStore{
			connections: map[string]map[string]any{}, groups: map[string]map[string]any{}, runs: map[string][]map[string]any{},
			filters: map[string]map[string]any{}, policies: map[string]map[string]any{}, tasks: map[string]map[string]any{},
		}
		for _, seed := range [][3]string{
			{"domain", "contoso.com", "confirmed"}, {"host", "contoso.com", "confirmed"}, {"host", "www.contoso.com", "confirmed"},
			{"ipAddress", "192.0.2.10", "candidate"}, {"page", "https://www.contoso.com/", "confirmed"},
			{"sslCert", "4b2c6f3f0f8b1c1d8e7a6b5c4d3e2f1a0b9c8d7e", "confirmed"},
		} {
			st.assets = append(st.assets, &azureEASMDAsset{Kind: seed[0], Name: seed[1], State: seed[2], Labels: []string{}, Created: seeded, Updated: seeded})
		}
		return st
	})
}

// handleAzureDefenderForEasmDataPlanePreview20241001Router serves the data
// plane under azureDefenderForEasmDataPlanePreview20241001Prefix.
func (s *Server) handleAzureDefenderForEasmDataPlanePreview20241001Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureDefenderForEASMDataPlanePreview20241001Prefix+"/") {
		return false
	}
	if got := r.URL.Query().Get("api-version"); got != azureDefenderForEASMDataPlanePreview20241001APIVersion {
		azureEASMDError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be "+azureDefenderForEASMDataPlanePreview20241001APIVersion+".")
		return true
	}
	var payload map[string]any
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		var ok bool
		if payload, ok = azureARMReadBody(w, r); !ok {
			return true
		}
	}
	seg := splitPathSegments(strings.TrimPrefix(path, azureDefenderForEASMDataPlanePreview20241001Prefix+"/"))
	status, body := s.azureEASMDServe(r.Method, seg, r.URL.Query(), payload)
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return true
	}
	if code, isErr := body.(azureEASMDErr); isErr {
		azureEASMDError(w, status, code.code, code.msg)
		return true
	}
	respondJSON(w, status, body)
	return true
}

type azureEASMDErr struct{ code, msg string }

func azureEASMDError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("x-ms-error-code", code)
	respondJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg}})
}

func azureEASMDBad(msg string) (int, any) {
	return http.StatusBadRequest, azureEASMDErr{"BadRequest", msg}
}

func azureEASMDNotFound(what, name string) (int, any) {
	return http.StatusNotFound, azureEASMDErr{"NotFound", what + " '" + name + "' was not found."}
}

// azureEASMDServe routes a data plane request; seg is the path after the prefix.
func (s *Server) azureEASMDServe(method string, seg []string, q map[string][]string, payload map[string]any) (int, any) {
	if len(seg) == 0 || len(seg) > 3 {
		return azureEASMDNotFound("Route", strings.Join(seg, "/"))
	}
	collection, action, _ := strings.Cut(seg[0], ":")
	name := ""
	if len(seg) >= 2 {
		name, action, _ = strings.Cut(seg[1], ":")
		if collection == "reports" && name == "assets" {
			return s.azureEASMDReport(method, action, payload)
		}
	}
	query := func(key string) string { return strings.Join(q[key], "") }
	st := s.azureEASMDState()
	st.mu.Lock()
	defer st.mu.Unlock()
	switch collection {
	case "assets":
		return st.serveAssets(method, name, action, query, payload)
	case "cisaCves":
		if method != http.MethodGet || action != "" {
			return azureEASMDMethod()
		}
		return azureEASMDCisaCves(name, query)
	case "discoTemplates":
		if method != http.MethodGet || action != "" {
			return azureEASMDMethod()
		}
		return azureEASMDTemplates(name, query)
	case "dataConnections":
		return st.serveNamed(method, name, action, query, payload, st.connections, "Data connection", azureEASMDConnection)
	case "savedFilters":
		return st.serveNamed(method, name, action, query, payload, st.filters, "Saved filter", azureEASMDSavedFilter)
	case "policies":
		return st.serveNamed(method, name, action, query, payload, st.policies, "Policy", st.policyBody)
	case "discoGroups":
		if len(seg) == 3 {
			if seg[2] != "runs" || method != http.MethodGet || action != "" {
				return azureEASMDNotFound("Route", strings.Join(seg, "/"))
			}
			if st.groups[name] == nil {
				return azureEASMDNotFound("Discovery group", name)
			}
			return http.StatusOK, azureEASMDPage(st.runs[name], query)
		}
		return st.serveGroups(method, name, action, query, payload)
	case "tasks":
		return st.serveTasks(method, name, action, query)
	}
	return azureEASMDNotFound("Route", strings.Join(seg, "/"))
}

func azureEASMDMethod() (int, any) {
	return http.StatusMethodNotAllowed, azureEASMDErr{"MethodNotAllowed", "The method is not allowed on this route."}
}

// azureEASMDPage returns the items as a page: skip and maxpagesize select it.
func azureEASMDPage(items []map[string]any, query func(string) string) map[string]any {
	skip, _ := strconv.Atoi(query("skip"))
	size, err := strconv.Atoi(query("maxpagesize"))
	if err != nil || size <= 0 || size > azureEASMDMaxPageSize {
		size = azureEASMDMaxPageSize
	}
	skip = min(max(skip, 0), len(items))
	end := min(skip+size, len(items))
	return map[string]any{"totalElements": len(items), "value": append([]map[string]any{}, items[skip:end]...)}
}

// azureEASMDMatch reports whether the asset satisfies a filter of clauses
// joined by AND, each `field = "v"`, `field != "v"`, `field ^= "prefix"` or
// `field IN ("a", "b")` over state, kind, name and externalId.
func azureEASMDMatch(filter string, a *azureEASMDAsset) (bool, string) {
	if strings.TrimSpace(filter) == "" {
		return true, ""
	}
	match := true
	for _, clause := range azureEASMDAndRE.Split(filter, -1) {
		m := azureEASMDClauseRE.FindStringSubmatch(clause)
		if m == nil {
			return false, "filter clause '" + strings.TrimSpace(clause) + "' is not supported."
		}
		field := map[string]string{"state": a.State, "kind": a.Kind, "name": a.Name, "externalId": a.ExternalID}
		value, known := field[m[1]]
		if !known {
			return false, "filter field '" + m[1] + "' is not supported."
		}
		var values []string
		for _, v := range strings.Split(strings.Trim(m[3], "()"), ",") {
			values = append(values, strings.Trim(strings.TrimSpace(v), `"`))
		}
		switch m[2] {
		case "=":
			match = match && value == values[0]
		case "!=":
			match = match && value != values[0]
		case "^=":
			match = match && strings.HasPrefix(value, values[0])
		case "IN":
			match = match && slices.Contains(values, value)
		}
	}
	return match, ""
}

func (a *azureEASMDAsset) fixture() map[string]any {
	return map[string]any{
		"kind": a.Kind, "id": a.id(), "name": a.Name, "displayName": a.Name, "uuid": uuid.NewSHA1(uuid.NameSpaceURL, []byte(a.id())).String(),
		"createdDate": a.Created.Format(time.RFC3339), "updatedDate": a.Updated.Format(time.RFC3339),
		"state": a.State, "externalId": a.ExternalID, "labels": slices.Clone(a.Labels), "wildcard": false,
		"discoGroupName": "", "auditTrail": []any{}, "reason": "", "asset": map[string]any{"name": a.Name},
	}
}

func (st *azureEASMDStore) filtered(filter string) ([]*azureEASMDAsset, string) {
	var out []*azureEASMDAsset
	for _, a := range st.assets {
		ok, msg := azureEASMDMatch(filter, a)
		if msg != "" {
			return nil, msg
		}
		if ok {
			out = append(out, a)
		}
	}
	return out, ""
}

func (st *azureEASMDStore) asset(id string) *azureEASMDAsset {
	for _, a := range st.assets {
		if a.id() == id || strings.TrimRight(id, "=") == a.id() {
			return a
		}
	}
	return nil
}

func (st *azureEASMDStore) serveAssets(method, id, action string, query func(string) string, payload map[string]any) (int, any) {
	switch {
	case id == "" && action == "" && method == http.MethodGet:
		assets, msg := st.filtered(query("filter"))
		if msg != "" {
			return azureEASMDBad(msg)
		}
		items := make([]map[string]any, 0, len(assets))
		for _, a := range assets {
			items = append(items, a.fixture())
		}
		return http.StatusOK, azureEASMDPage(items, query)
	case id == "" && action == "" && method == http.MethodPost:
		return st.updateAssets(query("filter"), payload)
	case id == "" && action == "export" && method == http.MethodPost:
		if _, msg := st.filtered(query("filter")); msg != "" {
			return azureEASMDBad(msg)
		}
		if strings.TrimSpace(azureARMString(payload["fileName"])) == "" || len(azureARMSlice(payload["columns"])) == 0 {
			return azureEASMDBad("fileName and columns are required.")
		}
		return http.StatusOK, st.newTask("export", map[string]any{"fileName": payload["fileName"], "filter": query("filter")})
	case id == "" && action == "getDeltaDetails" && method == http.MethodPost:
		if msg := azureEASMDDelta(payload); msg != "" {
			return azureEASMDBad(msg)
		}
		if kind := azureARMString(payload["deltaDetailType"]); kind != "" && kind != "added" && kind != "removed" {
			return azureEASMDBad("deltaDetailType must be added or removed.")
		}
		return http.StatusOK, azureEASMDPage(nil, query)
	case id == "" && action == "getDeltaSummary" && method == http.MethodPost:
		if msg := azureEASMDDelta(payload); msg != "" {
			return azureEASMDBad(msg)
		}
		return http.StatusOK, map[string]any{"range": []any{}, "priorDays": payload["priorDays"], "date": payload["date"], "summary": map[string]any{"kindSummaries": []any{}}}
	case id != "" && action == "" && method == http.MethodGet:
		if a := st.asset(id); a != nil {
			return http.StatusOK, a.fixture()
		}
		return azureEASMDNotFound("Asset", id)
	case id != "" && action == "getObservations" && method == http.MethodPost:
		if st.asset(id) == nil {
			return azureEASMDNotFound("Asset", id)
		}
		page := azureEASMDPage(nil, query)
		page["prioritySummary"] = map[string]any{"high": 0, "medium": 0, "low": 0}
		return http.StatusOK, page
	}
	return azureEASMDMethod()
}

func azureEASMDDelta(payload map[string]any) string {
	if days, ok := payload["priorDays"].(float64); payload["priorDays"] != nil && (!ok || days < 1 || days > 30 || days != float64(int(days))) {
		return "priorDays must be an integer from 1 to 30."
	}
	if date := azureARMString(payload["date"]); date != "" {
		if _, err := time.Parse(time.DateOnly, date); err != nil {
			return "date must be YYYY-MM-DD."
		}
	}
	return ""
}

// updateAssets applies state, externalId and labels to the matching assets.
func (st *azureEASMDStore) updateAssets(filter string, payload map[string]any) (int, any) {
	if strings.TrimSpace(filter) == "" {
		return azureEASMDBad("filter is required.")
	}
	assets, msg := st.filtered(filter)
	if msg != "" {
		return azureEASMDBad(msg)
	}
	state := azureARMString(payload["state"])
	if state != "" && !slices.Contains(azureEASMDStates, state) {
		return azureEASMDBad("state must be one of " + strings.Join(azureEASMDStates, ", ") + ".")
	}
	labels := azureARMObject(payload["labels"])
	if len(labels) > 100 {
		return azureEASMDBad("at most 100 labels may be set at once.")
	}
	now := time.Now().UTC()
	for _, a := range assets {
		if state != "" {
			a.State = state
		}
		if ext, ok := payload["externalId"].(string); ok {
			a.ExternalID = ext
		}
		for label, raw := range labels {
			keep := raw != false
			has := slices.Contains(a.Labels, label)
			switch {
			case keep && !has && len(a.Labels) < 100:
				a.Labels = append(a.Labels, label)
			case !keep && has:
				a.Labels = slices.DeleteFunc(slices.Clone(a.Labels), func(l string) bool { return l == label })
			}
		}
		a.Updated = now
	}
	return http.StatusOK, st.newTask("assetUpdate", map[string]any{"filter": filter, "updated": len(assets)})
}

// newTask records a finished task. Caller holds st.mu.
func (st *azureEASMDStore) newTask(phase string, metadata map[string]any) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	task := map[string]any{"id": uuid.NewString(), "startedAt": now, "completedAt": now, "lastPolledAt": now, "state": "complete", "phase": phase, "reason": "", "metadata": metadata}
	if len(st.taskOrder) >= azureEASMDMaxItems {
		delete(st.tasks, st.taskOrder[0])
		st.taskOrder = st.taskOrder[1:]
	}
	st.taskOrder = append(st.taskOrder, task["id"].(string))
	st.tasks[task["id"].(string)] = task
	return azureARMMerge(task, nil)
}

// taskList returns copies of the tasks, oldest first.
func (st *azureEASMDStore) taskList() []map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]map[string]any, 0, len(st.taskOrder))
	for _, id := range st.taskOrder {
		out = append(out, azureARMMerge(st.tasks[id], nil))
	}
	return out
}

func (st *azureEASMDStore) serveTasks(method, id, action string, query func(string) string) (int, any) {
	if id == "" {
		if method != http.MethodGet || action != "" {
			return azureEASMDMethod()
		}
		items := make([]map[string]any, 0, len(st.taskOrder))
		for _, tid := range st.taskOrder {
			items = append(items, azureARMMerge(st.tasks[tid], nil))
		}
		return http.StatusOK, azureEASMDPage(items, query)
	}
	task := st.tasks[id]
	if task == nil {
		return azureEASMDNotFound("Task", id)
	}
	switch {
	case method == http.MethodGet && action == "":
	case method == http.MethodPost && action == "cancel":
		if task["state"] != "complete" && task["state"] != "failed" {
			task["state"] = "incomplete"
		}
	case method == http.MethodPost && action == "run":
		task["state"], task["lastPolledAt"] = "complete", time.Now().UTC().Format(time.RFC3339)
	case method == http.MethodPost && action == "download":
		meta := azureARMMerge(azureARMObject(task["metadata"]), map[string]any{"downloadUrl": "https://easm.stackyard.local/downloads/" + id})
		task["metadata"] = meta
	default:
		return azureEASMDMethod()
	}
	return http.StatusOK, azureARMMerge(task, nil)
}

// serveNamed serves a named collection: list, get, put (create or replace),
// delete and a :validate action that checks a body without storing it.
func (st *azureEASMDStore) serveNamed(method, name, action string, query func(string) string, payload map[string]any, items map[string]map[string]any, what string, build func(name string, payload map[string]any) (map[string]any, string)) (int, any) {
	switch {
	case name == "" && action == "" && method == http.MethodGet:
		return http.StatusOK, azureEASMDPage(azureEASMDSorted(items), query)
	case name == "" && action == "validate" && method == http.MethodPost:
		if _, msg := build(azureARMString(payload["name"]), payload); msg != "" {
			return http.StatusOK, map[string]any{"error": map[string]any{"code": "BadRequest", "message": msg}}
		}
		return http.StatusOK, map[string]any{}
	case name == "" || action != "":
		return azureEASMDMethod()
	case !azureEASMDNameRE.MatchString(name):
		return azureEASMDBad("names must be 1-128 letters, digits, '.', '-' or '_'.")
	}
	switch method {
	case http.MethodGet:
		if items[name] == nil {
			return azureEASMDNotFound(what, name)
		}
		return http.StatusOK, azureARMMerge(items[name], nil)
	case http.MethodPut:
		item, msg := build(name, payload)
		if msg != "" {
			return azureEASMDBad(msg)
		}
		if items[name] == nil && len(items) >= azureEASMDMaxItems {
			return http.StatusConflict, azureEASMDErr{"Conflict", "The collection is full."}
		}
		items[name] = item
		return http.StatusOK, azureARMMerge(item, nil)
	case http.MethodDelete:
		if items[name] == nil {
			return azureEASMDNotFound(what, name)
		}
		delete(items, name)
		return http.StatusNoContent, nil
	}
	return azureEASMDMethod()
}

func azureEASMDSorted(items map[string]map[string]any) []map[string]any {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	slices.Sort(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, azureARMMerge(items[name], nil))
	}
	return out
}

func azureEASMDConnection(name string, payload map[string]any) (map[string]any, string) {
	kind := azureARMString(payload["kind"])
	props := azureARMObject(payload["properties"])
	switch {
	case !slices.Contains(azureEASMDConnKinds, kind):
		return nil, "kind must be logAnalytics or azureDataExplorer."
	case kind == "logAnalytics" && (azureARMString(props["apiKey"]) == "" || azureARMString(props["workspaceId"]) == ""):
		return nil, "logAnalytics connections need properties.apiKey and properties.workspaceId."
	case kind == "azureDataExplorer" && (azureARMString(props["clusterName"]) == "" || azureARMString(props["databaseName"]) == "" || azureARMString(props["region"]) == ""):
		return nil, "azureDataExplorer connections need properties.clusterName, databaseName and region."
	case payload["frequency"] != nil && !slices.Contains(azureEASMDFrequencies, azureARMString(payload["frequency"])):
		return nil, "frequency must be daily, weekly or monthly."
	case payload["content"] != nil && !slices.Contains(azureEASMDContents, azureARMString(payload["content"])):
		return nil, "content must be assets or attackSurfaceInsights."
	}
	offset, isNum := payload["frequencyOffset"].(float64)
	if payload["frequencyOffset"] != nil && (!isNum || offset < 0 || offset > 31 || offset != float64(int(offset))) {
		return nil, "frequencyOffset must be an integer from 0 to 31."
	}
	// The API key is write-only.
	shown := azureARMMerge(props, nil)
	delete(shown, "apiKey")
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"kind": kind, "id": name, "name": name, "displayName": name, "content": azureEASMDDefault(payload["content"], "assets"),
		"frequency": azureEASMDDefault(payload["frequency"], "weekly"), "frequencyOffset": int(offset),
		"createdDate": now, "updatedDate": now, "userUpdatedAt": now, "active": true, "inactiveMessage": "", "properties": shown,
	}, ""
}

func azureEASMDDefault(raw any, fallback string) string {
	if text := azureARMString(raw); text != "" {
		return text
	}
	return fallback
}

func azureEASMDSavedFilter(name string, payload map[string]any) (map[string]any, string) {
	filter := azureARMString(payload["filter"])
	if strings.TrimSpace(filter) == "" || len(filter) > 4096 {
		return nil, "filter is required and must be at most 4096 characters."
	}
	if _, msg := azureEASMDMatch(filter, &azureEASMDAsset{}); msg != "" {
		return nil, msg
	}
	return map[string]any{"id": name, "name": name, "displayName": name, "filter": filter, "description": azureARMString(payload["description"])}, ""
}

// policyBody checks a policy, whose filterName must name a saved filter.
// Caller holds st.mu.
func (st *azureEASMDStore) policyBody(name string, payload map[string]any) (map[string]any, string) {
	filterName := azureARMString(payload["filterName"])
	if st.filters[filterName] == nil {
		return nil, "filterName must name an existing saved filter."
	}
	action := azureARMString(payload["action"])
	if !slices.Contains(azureEASMDPolicyActions, action) {
		return nil, "action must be one of " + strings.Join(azureEASMDPolicyActions, ", ") + "."
	}
	params := azureARMObject(payload["actionParameters"])
	if action == "setState" && !slices.Contains(azureEASMDStates, azureARMString(params["value"])) {
		return nil, "setState policies need actionParameters.value set to an asset state."
	}
	return map[string]any{
		"id": name, "name": name, "displayName": name, "filterName": filterName, "action": action,
		"actionParameters": params, "description": azureARMString(payload["description"]), "user": "stackyard", "createdDate": time.Now().UTC().Format(time.RFC3339),
	}, ""
}

func (st *azureEASMDStore) serveGroups(method, name, action string, query func(string) string, payload map[string]any) (int, any) {
	switch {
	case name == "" && (action == "getAssetChainSummary" || action == "dismissAssetChain") && method == http.MethodPost:
		source := azureARMString(payload["assetChainSource"])
		ids := azureARMSlice(payload["sourceIds"])
		if (source != "DISCO_GROUP" && source != "ASSET") || len(ids) == 0 || len(ids) > 100 {
			return azureEASMDBad("assetChainSource must be DISCO_GROUP or ASSET with 1-100 sourceIds.")
		}
		if action == "dismissAssetChain" {
			return http.StatusOK, st.newTask("dismissAssetChain", map[string]any{"assetChainSource": source, "sourceIds": ids})
		}
		return http.StatusOK, map[string]any{"affectedAssetsSummary": []any{}, "affectedGroupsSummary": []any{}, "errors": []any{}}
	case name != "" && action == "run" && method == http.MethodPost:
		group := st.groups[name]
		if group == nil {
			return azureEASMDNotFound("Discovery group", name)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		run := map[string]any{"submittedDate": now, "startedDate": now, "completedDate": now, "tier": group["tier"], "state": "completed", "totalAssetsFoundCount": 0, "seeds": group["seeds"], "excludes": group["excludes"], "names": group["names"]}
		runs := append([]map[string]any{run}, st.runs[name]...)
		st.runs[name] = runs[:min(len(runs), azureEASMDMaxRuns)]
		return http.StatusNoContent, nil
	}
	return st.serveNamed(method, name, action, query, payload, st.groups, "Discovery group", azureEASMDGroup)
}

func azureEASMDGroup(name string, payload map[string]any) (map[string]any, string) {
	seeds := azureARMSlice(payload["seeds"])
	if len(seeds) == 0 && azureARMString(payload["templateId"]) == "" {
		return nil, "seeds or templateId is required."
	}
	if len(seeds) > azureEASMDMaxSeeds {
		return nil, "at most 100 seeds are allowed."
	}
	for _, raw := range seeds {
		seed := azureARMObject(raw)
		if !slices.Contains(azureEASMDAssetKinds, azureARMString(seed["kind"])) || azureARMString(seed["name"]) == "" {
			return nil, "each seed needs a name and an asset kind (" + strings.Join(azureEASMDAssetKinds, ", ") + ")."
		}
	}
	freq, isNum := payload["frequencyMilliseconds"].(float64)
	if payload["frequencyMilliseconds"] != nil && (!isNum || freq < 0) {
		return nil, "frequencyMilliseconds must be a non-negative number."
	}
	tier := azureEASMDDefault(payload["tier"], "advanced")
	if tier != "advanced" && tier != "basic" {
		return nil, "tier must be advanced or basic."
	}
	return map[string]any{
		"id": name, "name": name, "displayName": name, "description": azureARMString(payload["description"]), "tier": tier,
		"frequencyMilliseconds": int64(freq), "seeds": seeds, "names": azureEASMDList(payload["names"]), "excludes": azureEASMDList(payload["excludes"]),
		"latestRun": nil, "createdDate": time.Now().UTC().Format(time.RFC3339), "templateId": azureARMString(payload["templateId"]),
	}, ""
}

func azureEASMDList(raw any) []any {
	if list := azureARMSlice(raw); list != nil {
		return list
	}
	return []any{}
}

var azureEASMDCVEs = []map[string]any{
	{"cveId": "CVE-2023-12345", "vendorProject": "Contoso", "product": "Contoso Web Server", "vulnerabilityName": "Contoso Web Server Remote Code Execution", "shortDescription": "A crafted request lets an attacker run code.", "requiredAction": "Apply the vendor update.", "notes": "", "dateAdded": "2023-06-01T00:00:00Z", "dueDate": "2023-06-22T00:00:00Z", "updatedAt": "2023-06-01T00:00:00Z", "count": 1},
	{"cveId": "CVE-2024-3400", "vendorProject": "Palo Alto Networks", "product": "PAN-OS", "vulnerabilityName": "PAN-OS Command Injection", "shortDescription": "Command injection in GlobalProtect.", "requiredAction": "Apply the vendor update.", "notes": "", "dateAdded": "2024-04-12T00:00:00Z", "dueDate": "2024-04-19T00:00:00Z", "updatedAt": "2024-04-12T00:00:00Z", "count": 0},
}

func azureEASMDCisaCves(id string, query func(string) string) (int, any) {
	if id == "" {
		return http.StatusOK, azureEASMDPage(azureEASMDCVEs, query)
	}
	for _, cve := range azureEASMDCVEs {
		if cve["cveId"] == id {
			return http.StatusOK, cve
		}
	}
	return azureEASMDNotFound("CVE", id)
}

var azureEASMDTemplateList = []map[string]any{
	{"id": "12345", "name": "Contoso", "displayName": "Contoso", "industry": "Technology", "region": "North America", "countryCode": "US", "stateCode": "WA", "city": "Redmond", "seeds": []any{map[string]any{"kind": "domain", "name": "contoso.com"}}, "names": []any{"Contoso"}},
	{"id": "67890", "name": "Fabrikam", "displayName": "Fabrikam", "industry": "Manufacturing", "region": "Europe", "countryCode": "DE", "stateCode": "", "city": "Berlin", "seeds": []any{map[string]any{"kind": "domain", "name": "fabrikam.com"}}, "names": []any{"Fabrikam"}},
}

func azureEASMDTemplates(id string, query func(string) string) (int, any) {
	if id == "" {
		filter := strings.ToLower(query("filter"))
		var items []map[string]any
		for _, t := range azureEASMDTemplateList {
			if strings.Contains(strings.ToLower(azureARMString(t["name"])), filter) {
				items = append(items, t)
			}
		}
		return http.StatusOK, azureEASMDPage(items, query)
	}
	for _, t := range azureEASMDTemplateList {
		if t["id"] == id {
			return http.StatusOK, t
		}
	}
	return azureEASMDNotFound("Discovery template", id)
}

// azureEASMDReport serves reports/assets:{getBillable|getSummary|getSnapshot|getSnapshotExport}.
func (s *Server) azureEASMDReport(method, action string, payload map[string]any) (int, any) {
	if method != http.MethodPost {
		return azureEASMDMethod()
	}
	st := s.azureEASMDState()
	st.mu.Lock()
	defer st.mu.Unlock()
	switch action {
	case "getBillable":
		counts := map[string]int{}
		for _, a := range st.assets {
			if a.State == "confirmed" {
				counts[a.Kind]++
			}
		}
		return http.StatusOK, map[string]any{"assetSummaries": []any{map[string]any{
			"date": time.Now().UTC().Format(time.DateOnly), "total": len(st.assets),
			"assetHostCount": counts["host"], "assetDomainCount": counts["domain"], "assetIpAddressCount": counts["ipAddress"], "assetSslCertCount": counts["sslCert"],
		}}}
	case "getSummary":
		filters := azureARMSlice(payload["filters"])
		if len(filters) > 50 {
			return azureEASMDBad("at most 50 filters are allowed.")
		}
		summaries := make([]any, 0, len(filters))
		for _, raw := range filters {
			assets, msg := st.filtered(azureARMString(raw))
			if msg != "" {
				return azureEASMDBad(msg)
			}
			summaries = append(summaries, map[string]any{"displayName": raw, "filter": raw, "count": len(assets), "updatedAt": time.Now().UTC().Format(time.RFC3339), "children": []any{}})
		}
		return http.StatusOK, map[string]any{"assetSummaries": summaries}
	case "getSnapshot", "getSnapshotExport":
		metric := azureARMString(payload["metric"])
		if metric == "" {
			return azureEASMDBad("metric is required.")
		}
		if action == "getSnapshotExport" {
			if azureARMString(payload["fileName"]) == "" || len(azureARMSlice(payload["columns"])) == 0 {
				return azureEASMDBad("fileName and columns are required.")
			}
			return http.StatusOK, st.newTask("snapshotExport", map[string]any{"metric": metric, "fileName": payload["fileName"]})
		}
		return http.StatusOK, map[string]any{"displayName": metric, "metric": metric, "description": "Assets for " + metric + ".", "updatedAt": time.Now().UTC().Format(time.RFC3339), "assets": map[string]any{"totalElements": 0, "value": []any{}}}
	}
	return azureEASMDNotFound("Report", action)
}
