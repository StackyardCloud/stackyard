package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	schAccount = "123456789012"
	schRegion  = "us-east-1"

	schAWSRegistry        = "aws.events"
	schDiscoveredRegistry = "discovered-schemas"

	schMaxRegistries  = 1000
	schMaxSchemas     = 10000 // across all registries
	schMaxVersions    = 100   // kept per schema
	schMaxDiscoverers = 100
	schMaxTags        = 50
	schMaxPolicy      = 20 << 10
)

var (
	schRegistryNameRe = regexp.MustCompile(`^[.\-_A-Za-z0-9]{1,64}$`)
	schSchemaNameRe   = regexp.MustCompile(`^[@.\-_A-Za-z0-9]{1,385}$`)
	schLanguages      = map[string]bool{"Java8": true, "Python36": true, "TypeScript3": true, "Go1": true}
)

type schError struct {
	status    int
	code, msg string
}

func schErr(status int, code, format string, args ...any) *schError {
	return &schError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func schBadRequest(format string, args ...any) *schError {
	return schErr(http.StatusBadRequest, "BadRequestException", format, args...)
}

func schNotFound(format string, args ...any) *schError {
	return schErr(http.StatusNotFound, "NotFoundException", format, args...)
}

func schConflict(format string, args ...any) *schError {
	return schErr(http.StatusConflict, "ConflictException", format, args...)
}

func schForbidden(format string, args ...any) *schError {
	return schErr(http.StatusForbidden, "ForbiddenException", format, args...)
}

type schVersion struct {
	version, content, typ string
	created               time.Time
}

type schBinding struct {
	created, modified time.Time
	complete          bool
}

type schSchema struct {
	name, desc   string
	versions     []*schVersion // oldest first
	next         int           // number the next version gets
	lastModified time.Time
	bindings     map[string]*schBinding // by language + "@" + version
}

func (sc *schSchema) latest() *schVersion { return sc.versions[len(sc.versions)-1] }

func (sc *schSchema) version(v string) *schVersion {
	if v == "" {
		return sc.latest()
	}
	for _, ver := range sc.versions {
		if ver.version == v {
			return ver
		}
	}
	return nil
}

type schRegistry struct {
	name, desc string
	aws        bool // AWS-managed (aws.events): read-only
	schemas    map[string]*schSchema
}

type schDiscoverer struct {
	id, source, desc, state string
	cross                   bool
}

type schPolicy struct{ policy, revision string }

type schStore struct {
	mu          sync.Mutex
	registries  map[string]*schRegistry
	schemaCount int
	discoverers map[string]*schDiscoverer
	policies    map[string]*schPolicy        // by registry name; "" is the account policy
	tags        map[string]map[string]string // by resource ARN
}

func newSchemasStore() *schStore {
	st := &schStore{
		registries:  map[string]*schRegistry{},
		discoverers: map[string]*schDiscoverer{},
		policies:    map[string]*schPolicy{},
		tags:        map[string]map[string]string{},
	}
	st.registries[schAWSRegistry] = &schRegistry{name: schAWSRegistry, desc: "Schemas for events emitted by AWS services.", aws: true, schemas: map[string]*schSchema{}}
	st.registries[schDiscoveredRegistry] = &schRegistry{name: schDiscoveredRegistry, desc: "Schemas found by schema discoverers.", schemas: map[string]*schSchema{}}
	// ponytail: two sample AWS event schemas, not the full aws.events catalog.
	now := time.Now().UTC()
	for name, content := range schBuiltinSchemas {
		st.registries[schAWSRegistry].schemas[name] = &schSchema{name: name, next: 2, lastModified: now, bindings: map[string]*schBinding{},
			versions: []*schVersion{{version: "1", content: content, typ: "OpenApi3", created: now}}}
		st.schemaCount++
	}
	return st
}

func schRegistryARN(name string) string {
	return "arn:aws:schemas:" + schRegion + ":" + schAccount + ":registry/" + name
}

func schSchemaARN(registry, name string) string {
	return "arn:aws:schemas:" + schRegion + ":" + schAccount + ":schema/" + registry + "/" + name
}

func schDiscovererARN(id string) string {
	return "arn:aws:schemas:" + schRegion + ":" + schAccount + ":discoverer/" + id
}

func schTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func (st *schStore) tagsOf(arn string) map[string]any {
	out := map[string]any{}
	for k, v := range st.tags[arn] {
		out[k] = v
	}
	return out
}

func (st *schStore) setTags(arn string, tags map[string]string) *schError {
	merged := map[string]string{}
	for k, v := range st.tags[arn] {
		merged[k] = v
	}
	for k, v := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return schBadRequest("Tag keys can't start with aws:.")
		}
		merged[k] = v
	}
	if len(merged) > schMaxTags {
		return schBadRequest("A resource can have at most %d tags.", schMaxTags)
	}
	if len(merged) > 0 {
		st.tags[arn] = merged
	}
	return nil
}

// --- registries ---

func (st *schStore) registryLocked(name string) (*schRegistry, *schError) {
	reg := st.registries[name]
	if reg == nil {
		return nil, schNotFound("Registry %s does not exist.", name)
	}
	return reg, nil
}

func (st *schStore) registryOut(reg *schRegistry) map[string]any {
	out := map[string]any{"RegistryArn": schRegistryARN(reg.name), "RegistryName": reg.name, "tags": st.tagsOf(schRegistryARN(reg.name))}
	if reg.desc != "" {
		out["Description"] = reg.desc
	}
	return out
}

func (st *schStore) createRegistry(name, desc string, tags map[string]string) (map[string]any, *schError) {
	if !schRegistryNameRe.MatchString(name) || strings.HasPrefix(name, "aws.") {
		return nil, schBadRequest("Registry name %q isn't valid: use 1-64 letters, digits, '.', '-' or '_', not starting with aws.", name)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.registries[name] != nil {
		return nil, schConflict("Registry %s already exists.", name)
	}
	if len(st.registries) >= schMaxRegistries {
		return nil, schErr(http.StatusTooManyRequests, "TooManyRequestsException", "An account can have at most %d registries.", schMaxRegistries)
	}
	if err := st.setTags(schRegistryARN(name), tags); err != nil {
		return nil, err
	}
	reg := &schRegistry{name: name, desc: desc, schemas: map[string]*schSchema{}}
	st.registries[name] = reg
	return st.registryOut(reg), nil
}

func (st *schStore) describeRegistry(name string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, err := st.registryLocked(name)
	if err != nil {
		return nil, err
	}
	return st.registryOut(reg), nil
}

func (st *schStore) updateRegistry(name string, desc *string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, err := st.registryLocked(name)
	if err != nil {
		return nil, err
	}
	if reg.aws {
		return nil, schForbidden("Registry %s is managed by AWS and can't be changed.", name)
	}
	if desc != nil {
		reg.desc = *desc
	}
	return st.registryOut(reg), nil
}

func (st *schStore) deleteRegistry(name string) *schError {
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, err := st.registryLocked(name)
	if err != nil {
		return err
	}
	if reg.aws || name == schDiscoveredRegistry {
		return schForbidden("Registry %s is managed by AWS and can't be deleted.", name)
	}
	for sname := range reg.schemas {
		delete(st.tags, schSchemaARN(name, sname))
	}
	st.schemaCount -= len(reg.schemas)
	delete(st.registries, name)
	delete(st.tags, schRegistryARN(name))
	delete(st.policies, name)
	return nil
}

func (st *schStore) listRegistries(prefix, scope string, token string, limit int) (map[string]any, *schError) {
	if scope != "" && scope != "LOCAL" && scope != "AWS" {
		return nil, schBadRequest("Scope must be LOCAL or AWS.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	var names []string
	for name, reg := range st.registries {
		if !strings.HasPrefix(name, prefix) || (scope == "AWS" && !reg.aws) || (scope == "LOCAL" && reg.aws) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	page, next, err := schPage(names, token, limit)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, name := range page {
		out := st.registryOut(st.registries[name])
		delete(out, "Description")
		items = append(items, out)
	}
	return schWithToken(map[string]any{"Registries": items}, next), nil
}

// --- schemas ---

func (st *schStore) schemaLocked(registry, name string) (*schRegistry, *schSchema, *schError) {
	reg, err := st.registryLocked(registry)
	if err != nil {
		return nil, nil, err
	}
	sc := reg.schemas[name]
	if sc == nil {
		return nil, nil, schNotFound("Schema %s does not exist in registry %s.", name, registry)
	}
	return reg, sc, nil
}

func schSchemaOut(registry string, sc *schSchema, ver *schVersion, tags map[string]any, withContent bool) map[string]any {
	out := map[string]any{
		"SchemaArn": schSchemaARN(registry, sc.name), "SchemaName": sc.name, "SchemaVersion": ver.version, "Type": ver.typ,
		"LastModified": schTime(sc.lastModified), "VersionCreatedDate": schTime(ver.created), "tags": tags,
	}
	if sc.desc != "" {
		out["Description"] = sc.desc
	}
	if withContent {
		out["Content"] = ver.content
	}
	return out
}

func (st *schStore) createSchema(registry, name, typ, content, desc string, tags map[string]string) (map[string]any, *schError) {
	if !schSchemaNameRe.MatchString(name) {
		return nil, schBadRequest("Schema name %q isn't valid: use 1-385 letters, digits, '@', '.', '-' or '_'.", name)
	}
	if err := schCheckContent(typ, content); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, err := st.registryLocked(registry)
	if err != nil {
		return nil, err
	}
	if reg.aws {
		return nil, schForbidden("Registry %s is managed by AWS; schemas can't be added to it.", registry)
	}
	if reg.schemas[name] != nil {
		return nil, schConflict("Schema %s already exists in registry %s.", name, registry)
	}
	if st.schemaCount >= schMaxSchemas {
		return nil, schErr(http.StatusTooManyRequests, "TooManyRequestsException", "An account can have at most %d schemas.", schMaxSchemas)
	}
	arn := schSchemaARN(registry, name)
	if err := st.setTags(arn, tags); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	sc := &schSchema{name: name, desc: desc, next: 2, lastModified: now, bindings: map[string]*schBinding{},
		versions: []*schVersion{{version: "1", content: content, typ: typ, created: now}}}
	reg.schemas[name] = sc
	st.schemaCount++
	return schSchemaOut(registry, sc, sc.latest(), st.tagsOf(arn), false), nil
}

func (st *schStore) updateSchema(registry, name string, typ, content, desc *string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, sc, err := st.schemaLocked(registry, name)
	if err != nil {
		return nil, err
	}
	if reg.aws {
		return nil, schForbidden("Registry %s is managed by AWS and can't be changed.", registry)
	}
	if typ != nil && content == nil {
		return nil, schBadRequest("Content is required when Type is given.")
	}
	now := time.Now().UTC()
	if desc != nil {
		sc.desc = *desc
		sc.lastModified = now
	}
	// Unchanged content (including a retried request) never adds a version,
	// so ClientTokenId needs no bookkeeping of its own.
	if content != nil {
		newType := sc.latest().typ
		if typ != nil {
			newType = *typ
		}
		if err := schCheckContent(newType, *content); err != nil {
			return nil, err
		}
		if *content != sc.latest().content || newType != sc.latest().typ {
			if len(sc.versions) >= schMaxVersions {
				return nil, schErr(http.StatusTooManyRequests, "TooManyRequestsException", "A schema can keep at most %d versions; delete some first.", schMaxVersions)
			}
			sc.versions = append(sc.versions, &schVersion{version: strconv.Itoa(sc.next), content: *content, typ: newType, created: now})
			sc.next++
			sc.lastModified = now
		}
	}
	return schSchemaOut(registry, sc, sc.latest(), st.tagsOf(schSchemaARN(registry, name)), false), nil
}

func (st *schStore) describeSchema(registry, name, version string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, sc, err := st.schemaLocked(registry, name)
	if err != nil {
		return nil, err
	}
	ver := sc.version(version)
	if ver == nil {
		return nil, schNotFound("Schema %s has no version %s.", name, version)
	}
	return schSchemaOut(registry, sc, ver, st.tagsOf(schSchemaARN(registry, name)), true), nil
}

func (st *schStore) deleteSchema(registry, name string) *schError {
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, _, err := st.schemaLocked(registry, name)
	if err != nil {
		return err
	}
	if reg.aws {
		return schForbidden("Registry %s is managed by AWS and can't be changed.", registry)
	}
	delete(reg.schemas, name)
	delete(st.tags, schSchemaARN(registry, name))
	st.schemaCount--
	return nil
}

func (st *schStore) deleteSchemaVersion(registry, name, version string) *schError {
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, sc, err := st.schemaLocked(registry, name)
	if err != nil {
		return err
	}
	if reg.aws {
		return schForbidden("Registry %s is managed by AWS and can't be changed.", registry)
	}
	for i, ver := range sc.versions {
		if ver.version != version {
			continue
		}
		if len(sc.versions) == 1 {
			return schBadRequest("Version %s is the schema's only version; delete the schema instead.", version)
		}
		sc.versions = append(sc.versions[:i], sc.versions[i+1:]...)
		for key := range sc.bindings {
			if strings.HasSuffix(key, "@"+version) {
				delete(sc.bindings, key)
			}
		}
		return nil
	}
	return schNotFound("Schema %s has no version %s.", name, version)
}

func (st *schStore) listSchemas(registry, prefix, token string, limit int) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, err := st.registryLocked(registry)
	if err != nil {
		return nil, err
	}
	var names []string
	for name := range reg.schemas {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	page, next, err := schPage(names, token, limit)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, name := range page {
		sc := reg.schemas[name]
		items = append(items, map[string]any{
			"SchemaArn": schSchemaARN(registry, name), "SchemaName": name, "LastModified": schTime(sc.lastModified),
			"VersionCount": len(sc.versions), "tags": st.tagsOf(schSchemaARN(registry, name)),
		})
	}
	return schWithToken(map[string]any{"Schemas": items}, next), nil
}

func (st *schStore) listSchemaVersions(registry, name, token string, limit int) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, sc, err := st.schemaLocked(registry, name)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(sc.versions))
	for i := range sc.versions {
		keys[i] = strconv.Itoa(i)
	}
	page, next, err := schPage(keys, token, limit)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, k := range page {
		i, _ := strconv.Atoi(k)
		ver := sc.versions[i]
		items = append(items, map[string]any{"SchemaArn": schSchemaARN(registry, name), "SchemaName": name, "SchemaVersion": ver.version, "Type": ver.typ})
	}
	return schWithToken(map[string]any{"SchemaVersions": items}, next), nil
}

// searchSchemas matches every keyword (case-insensitively) against a schema's
// name and the content of each of its versions.
func (st *schStore) searchSchemas(registry, keywords, token string, limit int) (map[string]any, *schError) {
	words := strings.Fields(strings.ToLower(keywords))
	if len(words) == 0 {
		return nil, schBadRequest("Keywords is required.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	reg, err := st.registryLocked(registry)
	if err != nil {
		return nil, err
	}
	matches := func(text string) bool {
		text = strings.ToLower(text)
		for _, w := range words {
			if !strings.Contains(text, w) {
				return false
			}
		}
		return true
	}
	var names []string
	for name, sc := range reg.schemas {
		hit := matches(name)
		for _, ver := range sc.versions {
			hit = hit || matches(ver.content)
		}
		if hit {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	page, next, err := schPage(names, token, limit)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, name := range page {
		sc := reg.schemas[name]
		versions := []any{}
		for _, ver := range sc.versions {
			versions = append(versions, map[string]any{"CreatedDate": schTime(ver.created), "SchemaVersion": ver.version, "Type": ver.typ})
		}
		items = append(items, map[string]any{"RegistryName": registry, "SchemaArn": schSchemaARN(registry, name), "SchemaName": name, "SchemaVersions": versions})
	}
	return schWithToken(map[string]any{"Schemas": items}, next), nil
}

// exportSchema returns the version as a JSON Schema Draft 4 document.
func (st *schStore) exportSchema(registry, name, version, typ string) (map[string]any, *schError) {
	if typ != "JSONSchemaDraft4" {
		return nil, schBadRequest("Type must be JSONSchemaDraft4.")
	}
	st.mu.Lock()
	_, sc, err := st.schemaLocked(registry, name)
	if err != nil {
		st.mu.Unlock()
		return nil, err
	}
	ver := sc.version(version)
	if ver == nil {
		st.mu.Unlock()
		return nil, schNotFound("Schema %s has no version %s.", name, version)
	}
	content, srcType, verName := ver.content, ver.typ, ver.version
	st.mu.Unlock()
	if srcType != "JSONSchemaDraft4" {
		converted, cerr := schOpenAPIToJSONSchema(name, content)
		if cerr != nil {
			return nil, cerr
		}
		content = converted
	}
	return map[string]any{"Content": content, "SchemaArn": schSchemaARN(registry, name), "SchemaName": name, "SchemaVersion": verName, "Type": "JSONSchemaDraft4"}, nil
}

// --- code bindings ---

func schBindingOut(b *schBinding, version string) map[string]any {
	status := "CREATE_IN_PROGRESS"
	if b.complete {
		status = "CREATE_COMPLETE"
	}
	return map[string]any{"CreationDate": schTime(b.created), "LastModified": schTime(b.modified), "SchemaVersion": version, "Status": status}
}

func (st *schStore) putCodeBinding(registry, name, lang, version string) (map[string]any, *schError) {
	if !schLanguages[lang] {
		return nil, schBadRequest("Language must be one of Java8, Python36, TypeScript3 or Go1.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	_, sc, err := st.schemaLocked(registry, name)
	if err != nil {
		return nil, err
	}
	ver := sc.version(version)
	if ver == nil {
		return nil, schNotFound("Schema %s has no version %s.", name, version)
	}
	key := lang + "@" + ver.version
	b := sc.bindings[key]
	if b == nil {
		now := time.Now().UTC()
		b = &schBinding{created: now, modified: now}
		sc.bindings[key] = b
	}
	return schBindingOut(b, ver.version), nil
}

// describeCodeBinding reports a binding; generation finishes on the first read.
func (st *schStore) describeCodeBinding(registry, name, lang, version string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	b, ver, err := st.bindingLocked(registry, name, lang, version)
	if err != nil {
		return nil, err
	}
	out := schBindingOut(b, ver.version)
	if !b.complete { // ponytail: settles on read, no background generation.
		b.complete = true
		b.modified = time.Now().UTC()
	}
	return out, nil
}

func (st *schStore) bindingLocked(registry, name, lang, version string) (*schBinding, *schVersion, *schError) {
	if !schLanguages[lang] {
		return nil, nil, schBadRequest("Language must be one of Java8, Python36, TypeScript3 or Go1.")
	}
	_, sc, err := st.schemaLocked(registry, name)
	if err != nil {
		return nil, nil, err
	}
	ver := sc.version(version)
	if ver == nil {
		return nil, nil, schNotFound("Schema %s has no version %s.", name, version)
	}
	b := sc.bindings[lang+"@"+ver.version]
	if b == nil {
		return nil, nil, schNotFound("No %s code binding exists for schema %s version %s.", lang, name, ver.version)
	}
	return b, ver, nil
}

// codeBindingSource returns the generated archive for a completed binding.
func (st *schStore) codeBindingSource(registry, name, lang, version string) ([]byte, *schError) {
	st.mu.Lock()
	b, ver, err := st.bindingLocked(registry, name, lang, version)
	if err != nil {
		st.mu.Unlock()
		return nil, err
	}
	if !b.complete {
		st.mu.Unlock()
		return nil, schNotFound("The %s code binding for schema %s version %s is still being generated.", lang, name, ver.version)
	}
	content, typ := ver.content, ver.typ
	st.mu.Unlock()
	return schGenerateBinding(lang, name, typ, content)
}

// --- discoverers ---

func (st *schStore) discovererOut(d *schDiscoverer) map[string]any {
	out := map[string]any{
		"DiscovererArn": schDiscovererARN(d.id), "DiscovererId": d.id, "SourceArn": d.source, "State": d.state,
		"CrossAccount": d.cross, "tags": st.tagsOf(schDiscovererARN(d.id)),
	}
	if d.desc != "" {
		out["Description"] = d.desc
	}
	return out
}

// createDiscoverer records a discoverer; the caller has already checked that
// the source event bus exists (outside this lock).
func (st *schStore) createDiscoverer(source, busName, desc string, cross bool, tags map[string]string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, d := range st.discoverers {
		if d.source == source {
			return nil, schConflict("A discoverer already exists for %s.", source)
		}
	}
	if len(st.discoverers) >= schMaxDiscoverers {
		return nil, schErr(http.StatusTooManyRequests, "TooManyRequestsException", "An account can have at most %d discoverers.", schMaxDiscoverers)
	}
	id := "events-event-bus-" + busName
	if st.discoverers[id] != nil {
		id += "-" + uuid.NewString()[:8]
	}
	if err := st.setTags(schDiscovererARN(id), tags); err != nil {
		return nil, err
	}
	d := &schDiscoverer{id: id, source: source, desc: desc, state: "STARTED", cross: cross}
	st.discoverers[id] = d
	return st.discovererOut(d), nil
}

func (st *schStore) discovererLocked(id string) (*schDiscoverer, *schError) {
	d := st.discoverers[id]
	if d == nil {
		return nil, schNotFound("Discoverer %s does not exist.", id)
	}
	return d, nil
}

func (st *schStore) describeDiscoverer(id string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.discovererLocked(id)
	if err != nil {
		return nil, err
	}
	return st.discovererOut(d), nil
}

func (st *schStore) updateDiscoverer(id string, desc *string, cross *bool) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.discovererLocked(id)
	if err != nil {
		return nil, err
	}
	if desc != nil {
		d.desc = *desc
	}
	if cross != nil {
		d.cross = *cross
	}
	return st.discovererOut(d), nil
}

func (st *schStore) setDiscovererState(id, state string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.discovererLocked(id)
	if err != nil {
		return nil, err
	}
	d.state = state
	return map[string]any{"DiscovererId": d.id, "State": d.state}, nil
}

func (st *schStore) deleteDiscoverer(id string) *schError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.discovererLocked(id); err != nil {
		return err
	}
	delete(st.discoverers, id)
	delete(st.tags, schDiscovererARN(id))
	return nil
}

func (st *schStore) listDiscoverers(idPrefix, sourcePrefix, token string, limit int) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var ids []string
	for id, d := range st.discoverers {
		if strings.HasPrefix(id, idPrefix) && strings.HasPrefix(d.source, sourcePrefix) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	page, next, err := schPage(ids, token, limit)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, id := range page {
		out := st.discovererOut(st.discoverers[id])
		delete(out, "Description")
		items = append(items, out)
	}
	return schWithToken(map[string]any{"Discoverers": items}, next), nil
}

// --- resource policies ---

func (st *schStore) putPolicy(registry, policy, revision string) (map[string]any, *schError) {
	if len(policy) > schMaxPolicy {
		return nil, schBadRequest("Policy can be at most %d bytes.", schMaxPolicy)
	}
	var doc map[string]any
	if json.Unmarshal([]byte(policy), &doc) != nil || doc["Statement"] == nil {
		return nil, schBadRequest("Policy must be a JSON policy document with a Statement.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if registry != "" {
		if _, err := st.registryLocked(registry); err != nil {
			return nil, err
		}
	}
	cur := st.policies[registry]
	if revision != "" && (cur == nil || cur.revision != revision) {
		return nil, schErr(http.StatusPreconditionFailed, "PreconditionFailedException", "RevisionId %s doesn't match the current policy revision.", revision)
	}
	p := &schPolicy{policy: policy, revision: uuid.NewString()}
	st.policies[registry] = p
	return map[string]any{"Policy": p.policy, "RevisionId": p.revision}, nil
}

func (st *schStore) getPolicy(registry string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if registry != "" {
		if _, err := st.registryLocked(registry); err != nil {
			return nil, err
		}
	}
	p := st.policies[registry]
	if p == nil {
		return nil, schNotFound("No resource policy is set.")
	}
	return map[string]any{"Policy": p.policy, "RevisionId": p.revision}, nil
}

func (st *schStore) deletePolicy(registry string) *schError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.policies[registry] == nil {
		return schNotFound("No resource policy is set.")
	}
	delete(st.policies, registry)
	return nil
}

// --- tags ---

// resourceExistsLocked reports whether arn names a registry, schema or
// discoverer in this store.
func (st *schStore) resourceExistsLocked(arn string) bool {
	prefix := "arn:aws:schemas:" + schRegion + ":" + schAccount + ":"
	rest, ok := strings.CutPrefix(arn, prefix)
	if !ok {
		return false
	}
	kind, id, _ := strings.Cut(rest, "/")
	switch kind {
	case "registry":
		return st.registries[id] != nil
	case "schema":
		registry, name, _ := strings.Cut(id, "/")
		reg := st.registries[registry]
		return reg != nil && reg.schemas[name] != nil
	case "discoverer":
		return st.discoverers[id] != nil
	}
	return false
}

func (st *schStore) listTags(arn string) (map[string]any, *schError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.resourceExistsLocked(arn) {
		return nil, schNotFound("Resource %s does not exist.", arn)
	}
	return map[string]any{"tags": st.tagsOf(arn)}, nil
}

func (st *schStore) tag(arn string, tags map[string]string) *schError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.resourceExistsLocked(arn) {
		return schNotFound("Resource %s does not exist.", arn)
	}
	return st.setTags(arn, tags)
}

func (st *schStore) untag(arn string, keys []string) *schError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.resourceExistsLocked(arn) {
		return schNotFound("Resource %s does not exist.", arn)
	}
	for _, k := range keys {
		delete(st.tags[arn], k)
	}
	return nil
}

// --- paging ---

// schPage returns one page of keys. The token is the offset, as AWS tokens
// are opaque to callers.
func schPage(keys []string, token string, limit int) ([]string, string, *schError) {
	start := 0
	if token != "" {
		n, err := strconv.ParseInt(token, 10, 32)
		if err != nil || n < 0 || int(n) > len(keys) {
			return nil, "", schBadRequest("NextToken is invalid.")
		}
		start = int(n)
	}
	end := min(start+limit, len(keys))
	next := ""
	if end < len(keys) {
		next = strconv.Itoa(end)
	}
	return keys[start:end], next, nil
}

func schWithToken(out map[string]any, next string) map[string]any {
	if next != "" {
		out["NextToken"] = next
	}
	return out
}
