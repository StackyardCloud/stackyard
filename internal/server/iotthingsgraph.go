package server

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS IoT Things Graph: awsJson1_1, X-Amz-Target
// IotThingsGraphFrontEndService.*, signed iotthingsgraph. The account's
// namespace holds entities uploaded as GraphQL (TDM) documents; flow and
// system templates and system instances (deployments) are GraphQL documents
// whose ids and references are read from their annotations. Uploads settle
// IN_PROGRESS -> SUCCEEDED, deployments DEPLOY_IN_PROGRESS ->
// DEPLOYED_IN_TARGET and undeployments UNDEPLOY_IN_PROGRESS -> NOT_DEPLOYED,
// one step per read. Templates and instances live in
// iotthingsgraph_templates.go. budServe writes responses and errors with
// Content-Type application/x-amz-json-1.1.

const (
	itgAccount = "123456789012"
	itgRegion  = "us-east-1"
	itgARNBase = "arn:aws:iotthingsgraph:" + itgRegion + ":" + itgAccount + ":"
	// itgNSPrefix starts every id in the account's own namespace.
	itgNSPrefix = "urn:tdm:" + itgRegion + "/" + itgAccount + "/default:"
	itgNSName   = itgRegion + "/" + itgAccount + "/default"
	itgNSARN    = itgARNBase + "Namespace/default"
	// ponytail: fixed caps bound stored counts; itgMaxBytes bounds the
	// GraphQL text kept across entities, revisions and instances (each
	// request is already capped at budMaxBody).
	itgMaxEntities   = 1000
	itgMaxTemplates  = 100 // per kind
	itgMaxRevisions  = 10  // kept per template
	itgMaxInstances  = 100
	itgMaxThings     = 1000
	itgMaxUploads    = 100
	itgMaxExecutions = 1000
	itgMaxTags       = 50
	itgMaxBytes      = 16 << 20
	itgInvalid       = "InvalidRequestException"
)

var itgAPI = &budJSONAPI{
	prefix: "IotThingsGraphFrontEndService.", signing: "iotthingsgraph", contentType: "application/x-amz-json-1.1",
	invalidCode: itgInvalid, model: awsmodels.IoTThingsGraphModel,
}

func init() {
	h := map[string]func(*itgStore, map[string]any, time.Time) (map[string]any, *budErr){
		"UploadEntityDefinitions":    (*itgStore).upload,
		"GetUploadStatus":            (*itgStore).uploadStatus,
		"DescribeNamespace":          (*itgStore).describeNamespace,
		"DeleteNamespace":            (*itgStore).deleteNamespace,
		"GetNamespaceDeletionStatus": (*itgStore).deletionStatus,
		"GetEntities":                (*itgStore).getEntities,
		"SearchEntities":             (*itgStore).searchEntities,
		"AssociateEntityToThing":     (*itgStore).associate,
		"DissociateEntityFromThing":  (*itgStore).dissociate,
		"SearchThings":               (*itgStore).searchThings,
		"CreateFlowTemplate":         (*itgStore).createFlow,
		"UpdateFlowTemplate":         (*itgStore).updateFlow,
		"GetFlowTemplate":            (*itgStore).getFlow,
		"GetFlowTemplateRevisions":   (*itgStore).flowRevisions,
		"DeprecateFlowTemplate":      (*itgStore).deprecateFlow,
		"DeleteFlowTemplate":         (*itgStore).deleteFlow,
		"SearchFlowTemplates":        (*itgStore).searchFlows,
		"CreateSystemTemplate":       (*itgStore).createSystem,
		"UpdateSystemTemplate":       (*itgStore).updateSystem,
		"GetSystemTemplate":          (*itgStore).getSystem,
		"GetSystemTemplateRevisions": (*itgStore).systemRevisions,
		"DeprecateSystemTemplate":    (*itgStore).deprecateSystem,
		"DeleteSystemTemplate":       (*itgStore).deleteSystem,
		"SearchSystemTemplates":      (*itgStore).searchSystems,
		"CreateSystemInstance":       (*itgStore).createInstance,
		"GetSystemInstance":          (*itgStore).getInstance,
		"DeploySystemInstance":       (*itgStore).deployInstance,
		"UndeploySystemInstance":     (*itgStore).undeployInstance,
		"DeleteSystemInstance":       (*itgStore).deleteInstance,
		"SearchSystemInstances":      (*itgStore).searchInstances,
		"SearchFlowExecutions":       (*itgStore).searchExecutions,
		"ListFlowExecutionMessages":  (*itgStore).executionMessages,
		"TagResource":                (*itgStore).tagResource,
		"UntagResource":              (*itgStore).untagResource,
		"ListTagsForResource":        (*itgStore).listTags,
	}
	itgAPI.handlers = make(map[string]budHandler, len(h))
	for op, fn := range h {
		itgAPI.handlers[op] = func(s *Server, in map[string]any) (map[string]any, *budErr) {
			st := s.iotThingsGraph
			st.mu.Lock()
			defer st.mu.Unlock()
			return fn(st, in, time.Now().UTC())
		}
	}
}

func (s *Server) handleIoTThingsGraphJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	return s.budServe(w, r, itgAPI, iotThingsGraphOperationByName)
}

type itgEntity struct {
	id, typ, text string
	deviceModel   string // for DEVICE entities
	refs          map[string]bool
	created       time.Time
}

type itgUpload struct {
	id, status string
	failures   []string
	nsVersion  int
	created    time.Time
}

type itgStore struct {
	mu          sync.Mutex
	nsExists    bool
	nsVersion   int
	synced      bool // tracks the public aws namespace
	deletion    string
	entities    map[string]*itgEntity
	things      map[string]map[string]string // thing name -> entity type -> entity id
	uploads     map[string]*itgUpload
	uploadOrder []string
	flows       map[string]*itgTemplate
	systems     map[string]*itgTemplate
	instances   map[string]*itgInstance
	executions  map[string]*itgExecution
	execOrder   []string
	bytes       int // GraphQL text held
}

func newIoTThingsGraphStore() *itgStore {
	return &itgStore{
		entities: map[string]*itgEntity{}, things: map[string]map[string]string{}, uploads: map[string]*itgUpload{},
		flows: map[string]*itgTemplate{}, systems: map[string]*itgTemplate{}, instances: map[string]*itgInstance{},
		executions: map[string]*itgExecution{},
	}
}

func itgErr(format string, args ...any) *budErr { return budErrorf(itgInvalid, format, args...) }

func itgNotFound(format string, args ...any) *budErr {
	return budErrorf("ResourceNotFoundException", format, args...)
}

// reserve accounts for n more bytes of stored GraphQL text.
func (st *itgStore) reserve(n int) *budErr {
	if st.bytes+n > itgMaxBytes {
		return budErrorf("LimitExceededException", "stored definitions would exceed %d bytes", itgMaxBytes)
	}
	st.bytes += n
	return nil
}

// ---- GraphQL (TDM) parsing ----

var (
	itgEntityRe      = regexp.MustCompile(`@(device|service|deviceModel|capability|state|action|event|property|mapping|enum)\s*\(\s*id\s*:\s*"([^"]*)"`)
	itgDeviceModelRe = regexp.MustCompile(`deviceModel\s*:\s*"([^"]*)"`)
	itgURNRe         = regexp.MustCompile(`"(urn:tdm:[^"]*)"`)
	itgIDPattern     = regexp.MustCompile(`^urn:tdm:(([a-z]{2}-(gov-)?[a-z]{4,9}-[0-9]{1,3}/[0-9]+/)*[\pL\pN_]+(/[\pL\pN_]+)*):([\pL]*):([\pL\pN_]+(/[\pL\pN_]+)*)$`)
)

var itgEntityTypes = map[string]string{
	"device": "DEVICE", "service": "SERVICE", "deviceModel": "DEVICE_MODEL", "capability": "CAPABILITY", "state": "STATE",
	"action": "ACTION", "event": "EVENT", "property": "PROPERTY", "mapping": "MAPPING", "enum": "ENUM",
}

// itgDeclarations splits a document into top-level declarations: text up to
// the brace that closes each top-level block.
func itgDeclarations(text string) []string {
	var out []string
	depth, start := 0, 0
	for i, c := range text {
		switch c {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 {
					out = append(out, strings.TrimSpace(text[start:i+1]))
					start = i + 1
				}
			}
		}
	}
	return out
}

// itgRefs returns the urn:tdm ids a document mentions, other than self.
func itgRefs(text, self string) map[string]bool {
	out := map[string]bool{}
	for _, m := range itgURNRe.FindAllStringSubmatch(text, -1) {
		if m[1] != self {
			out[m[1]] = true
		}
	}
	return out
}

// itgParseEntities reads the entity declarations of an upload.
func itgParseEntities(text string, now time.Time) ([]*itgEntity, []string) {
	var out []*itgEntity
	var problems []string
	seen := map[string]bool{}
	for _, decl := range itgDeclarations(text) {
		m := itgEntityRe.FindStringSubmatch(decl)
		if m == nil {
			continue
		}
		id := m[2]
		switch {
		case !itgIDPattern.MatchString(id):
			problems = append(problems, "invalid entity id "+id)
			continue
		case !strings.HasPrefix(id, itgNSPrefix):
			problems = append(problems, "entity "+id+" is outside the namespace "+itgNSName)
			continue
		case seen[id]:
			problems = append(problems, "entity "+id+" is defined more than once")
			continue
		}
		seen[id] = true
		e := &itgEntity{id: id, typ: itgEntityTypes[m[1]], text: decl, refs: itgRefs(decl, id), created: now}
		if e.typ == "DEVICE" {
			if dm := itgDeviceModelRe.FindStringSubmatch(decl); dm != nil {
				e.deviceModel = dm[1]
			}
		}
		out = append(out, e)
	}
	if len(out) == 0 && len(problems) == 0 {
		problems = append(problems, "the document defines no entities")
	}
	return out, problems
}

// ---- namespace and uploads ----

func (st *itgStore) upload(in map[string]any, now time.Time) (map[string]any, *budErr) {
	doc := budMap(in, "document")
	public := budBool(in, "syncWithPublicNamespace")
	if doc == nil && !public {
		return nil, itgErr("Specify a document, syncWithPublicNamespace, or both.")
	}
	up := &itgUpload{id: uuid.NewString(), status: "IN_PROGRESS", created: now}
	var parsed []*itgEntity
	if doc != nil {
		parsed, up.failures = itgParseEntities(budStr(doc, "text"), now)
	}
	if len(up.failures) == 0 {
		if err := st.applyUpload(parsed, budBool(in, "deprecateExistingEntities"), public); err != nil {
			up.failures = []string{err.msg}
		}
	}
	if len(up.failures) > 0 {
		up.status = "FAILED"
	}
	up.nsVersion = st.nsVersion
	if len(st.uploadOrder) >= itgMaxUploads {
		delete(st.uploads, st.uploadOrder[0])
		st.uploadOrder = append([]string{}, st.uploadOrder[1:]...)
	}
	st.uploads[up.id] = up
	st.uploadOrder = append(st.uploadOrder, up.id)
	return map[string]any{"uploadId": up.id}, nil
}

// applyUpload merges (or, with replace, swaps in) parsed entities and bumps
// the namespace version.
func (st *itgStore) applyUpload(parsed []*itgEntity, replace, public bool) *budErr {
	next := map[string]*itgEntity{}
	if !replace {
		for id, e := range st.entities {
			next[id] = e
		}
	}
	for _, e := range parsed {
		next[e.id] = e
	}
	if len(next) > itgMaxEntities {
		return itgErr("a namespace can hold at most %d entities", itgMaxEntities)
	}
	size := 0
	for _, e := range next {
		size += len(e.text)
	}
	old := 0
	for _, e := range st.entities {
		old += len(e.text)
	}
	st.bytes -= old
	if err := st.reserve(size); err != nil {
		st.bytes += old
		return err
	}
	st.entities, st.nsExists = next, true
	st.synced = st.synced || public
	st.nsVersion++
	return nil
}

func (st *itgStore) uploadStatus(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	up := st.uploads[budStr(in, "uploadId")]
	if up == nil {
		return nil, itgNotFound("Upload %s not found", budStr(in, "uploadId"))
	}
	out := map[string]any{"uploadId": up.id, "uploadStatus": up.status, "createdDate": budEpoch(up.created),
		"namespaceArn": itgNSARN, "namespaceName": itgNSName, "namespaceVersion": up.nsVersion}
	if len(up.failures) > 0 {
		reasons := make([]any, len(up.failures))
		for i, f := range up.failures {
			reasons[i] = f
		}
		out["failureReason"] = reasons
	}
	if up.status == "IN_PROGRESS" {
		up.status = "SUCCEEDED" // reported once, settled for the next read
	}
	return out, nil
}

func (st *itgStore) checkNSVersion(in map[string]any) *budErr {
	if v, ok := in["namespaceVersion"].(float64); ok && (v < 1 || int(v) > st.nsVersion) {
		return itgErr("namespaceVersion %v does not exist; the namespace is at version %d", v, st.nsVersion)
	}
	return nil
}

func (st *itgStore) describeNamespace(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	name := budStr(in, "namespaceName")
	if !st.nsExists || (name != "" && name != itgNSName) {
		return nil, itgNotFound("Namespace %s not found", itgNSName)
	}
	out := map[string]any{"namespaceArn": itgNSARN, "namespaceName": itgNSName, "namespaceVersion": st.nsVersion}
	if st.synced {
		out["trackingNamespaceName"], out["trackingNamespaceVersion"] = "aws", 1
	}
	return out, nil
}

// deleteNamespace drops every entity and thing association; templates and
// instances are left for the caller to delete, as in AWS.
func (st *itgStore) deleteNamespace(_ map[string]any, _ time.Time) (map[string]any, *budErr) {
	for _, e := range st.entities {
		st.bytes -= len(e.text)
	}
	st.entities, st.things = map[string]*itgEntity{}, map[string]map[string]string{}
	st.nsExists, st.nsVersion, st.synced, st.deletion = false, 0, false, "IN_PROGRESS"
	return map[string]any{"namespaceArn": itgNSARN, "namespaceName": itgNSName}, nil
}

func (st *itgStore) deletionStatus(_ map[string]any, _ time.Time) (map[string]any, *budErr) {
	if st.deletion == "" {
		return nil, itgErr("No namespace deletion has been requested.")
	}
	out := map[string]any{"namespaceArn": itgNSARN, "namespaceName": itgNSName, "status": st.deletion}
	st.deletion = "SUCCEEDED"
	return out, nil
}

// ---- entities ----

func (e *itgEntity) describe() map[string]any {
	return map[string]any{"id": e.id, "arn": itgARNBase + "Entity/" + e.id, "type": e.typ, "createdAt": budEpoch(e.created),
		"definition": map[string]any{"language": "GRAPHQL", "text": e.text}}
}

func (st *itgStore) getEntities(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if err := st.checkNSVersion(in); err != nil {
		return nil, err
	}
	out := []any{}
	for _, id := range budStrings(in, "ids") {
		e := st.entities[id]
		if e == nil {
			return nil, itgNotFound("Entity %s not found", id)
		}
		out = append(out, e.describe())
	}
	return map[string]any{"descriptions": out}, nil
}

func itgEntityName(id string) string { return id[strings.LastIndex(id, ":")+1:] }

func (e *itgEntity) matches(filter map[string]any) bool {
	values := budStrings(filter, "value")
	if len(values) == 0 {
		return true
	}
	for _, v := range values {
		switch budStr(filter, "name") {
		case "NAME":
			if itgEntityName(e.id) == v {
				return true
			}
		case "NAMESPACE":
			if v == itgNSName || v == "default" {
				return true
			}
		case "REFERENCED_ENTITY_ID":
			if e.refs[v] {
				return true
			}
		case "SEMANTIC_TYPE_PATH":
			if strings.Contains(e.text, v) {
				return true
			}
		}
	}
	return false
}

func (st *itgStore) searchEntities(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if err := st.checkNSVersion(in); err != nil {
		return nil, err
	}
	types := map[string]bool{}
	for _, t := range budStrings(in, "entityTypes") {
		types[t] = true
	}
	var match []*itgEntity
	for _, e := range st.entities {
		ok := types[e.typ]
		for _, f := range budList(in, "filters") {
			fm, _ := f.(map[string]any)
			ok = ok && e.matches(fm)
		}
		if ok {
			match = append(match, e)
		}
	}
	sort.Slice(match, func(i, j int) bool { return match[i].id < match[j].id })
	start, end, next, err := budPage(budStr(in, "nextToken"), len(match), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, e := range match[start:end] {
		out = append(out, e.describe())
	}
	return map[string]any{"descriptions": out, "nextToken": next}, nil
}

// ---- things ----

func itgThingARN(name string) string {
	return "arn:aws:iot:" + itgRegion + ":" + itgAccount + ":thing/" + name
}

func (st *itgStore) associate(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if err := st.checkNSVersion(in); err != nil {
		return nil, err
	}
	thing, id := budStr(in, "thingName"), budStr(in, "entityId")
	e := st.entities[id]
	if e == nil {
		return nil, itgNotFound("Entity %s not found", id)
	}
	if e.typ != "DEVICE" {
		return nil, itgErr("Only DEVICE entities can be associated with a thing; %s is a %s", id, e.typ)
	}
	cur := st.things[thing]
	if cur == nil && len(st.things) >= itgMaxThings {
		return nil, itgErr("at most %d things can have associations", itgMaxThings)
	}
	next := map[string]string{e.typ: id}
	for k, v := range cur {
		if k != e.typ {
			next[k] = v
		}
	}
	st.things[thing] = next
	return map[string]any{}, nil
}

func (st *itgStore) dissociate(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	thing, typ := budStr(in, "thingName"), budStr(in, "entityType")
	if _, ok := st.things[thing][typ]; !ok {
		return nil, itgNotFound("Thing %s has no associated %s entity", thing, typ)
	}
	next := map[string]string{}
	for k, v := range st.things[thing] {
		if k != typ {
			next[k] = v
		}
	}
	if len(next) == 0 {
		delete(st.things, thing)
	} else {
		st.things[thing] = next
	}
	return map[string]any{}, nil
}

// searchThings lists things associated with a device, or with any device of
// a device model.
func (st *itgStore) searchThings(in map[string]any, _ time.Time) (map[string]any, *budErr) {
	if err := st.checkNSVersion(in); err != nil {
		return nil, err
	}
	id := budStr(in, "entityId")
	e := st.entities[id]
	if e == nil {
		return nil, itgNotFound("Entity %s not found", id)
	}
	var names []string
	for name, assoc := range st.things {
		dev := st.entities[assoc["DEVICE"]]
		if dev != nil && (dev.id == id || (e.typ == "DEVICE_MODEL" && dev.deviceModel == id)) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	start, end, next, err := budPage(budStr(in, "nextToken"), len(names), budInt(in, "maxResults", 100), itgInvalid)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, end-start)
	for _, n := range names[start:end] {
		out = append(out, map[string]any{"thingName": n, "thingArn": itgThingARN(n)})
	}
	return map[string]any{"things": out, "nextToken": next}, nil
}
