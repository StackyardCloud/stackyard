package server

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	aprAccount = "123456789012"
	aprRegion  = "us-east-1"
	// Caps keep the store bounded; they sit at or above the AWS quotas.
	aprMaxApps            = 2000
	aprMaxGroups          = 2000
	aprMaxGroupsPerApp    = 1000
	aprMaxResourcesPerApp = 1000
	aprMaxTags            = 50
	aprResourceGroupState = "CREATE_COMPLETE"
)

type aprError struct {
	status int
	code   string
	msg    string
}

func aprErr(status int, code, format string, args ...any) *aprError {
	return &aprError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func aprValidation(format string, args ...any) *aprError {
	return aprErr(http.StatusBadRequest, "ValidationException", format, args...)
}

func aprNotFound(format string, args ...any) *aprError {
	return aprErr(http.StatusNotFound, "ResourceNotFoundException", format, args...)
}

func aprConflict(format string, args ...any) *aprError {
	return aprErr(http.StatusConflict, "ConflictException", format, args...)
}

func aprQuota(format string, args ...any) *aprError {
	return aprErr(http.StatusPaymentRequired, "ServiceQuotaExceededException", format, args...)
}

type aprResource struct {
	typ, name, arn string
	options        []string
	associated     time.Time
}

type aprApp struct {
	id, name, description string
	rgName                string // name its resource groups were created under
	created, updated      time.Time
	groups                []string // attribute group ids, in association order
	resources             []*aprResource
}

type aprGroup struct {
	id, name, description, attributes string
	created, updated                  time.Time
}

type aprStore struct {
	mu          sync.Mutex
	apps        map[string]*aprApp // by id
	appByName   map[string]string
	groups      map[string]*aprGroup // by id
	groupByName map[string]string
	appTokens   map[string]string            // clientToken -> application id
	groupTokens map[string]string            // clientToken -> attribute group id
	stackOwner  map[string]string            // CloudFormation stack ARN -> application id
	tags        map[string]map[string]string // by resource ARN
	tagKey      string                       // PutConfiguration tagQueryConfiguration.tagKey
}

func newAppRegistryStore() *aprStore {
	return &aprStore{
		apps: map[string]*aprApp{}, appByName: map[string]string{},
		groups: map[string]*aprGroup{}, groupByName: map[string]string{},
		appTokens: map[string]string{}, groupTokens: map[string]string{},
		stackOwner: map[string]string{}, tags: map[string]map[string]string{},
	}
}

// aprNewID returns a 26-character lowercase alphanumeric id, the shape AWS
// uses for application and attribute group ids.
func aprNewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

func aprAppARN(id string) string {
	return "arn:aws:servicecatalog:" + aprRegion + ":" + aprAccount + ":/applications/" + id
}

func aprGroupARN(id string) string {
	return "arn:aws:servicecatalog:" + aprRegion + ":" + aprAccount + ":/attribute-groups/" + id
}

// Resource group names AppRegistry creates for each application.
func aprAppGroupName(app string) string    { return "AWS_AppRegistry_Application-" + app }
func aprAppTagGroupName(app string) string { return "AWS_AppRegistry_AppTag_" + aprAccount + "-" + app }
func aprGroupARNFor(name string) string {
	return "arn:aws:resource-groups:" + aprRegion + ":" + aprAccount + ":group/" + name
}

// ponytail: RESOURCE_TAG_VALUE associations have no AWS resource of their
// own, so their ARN is the application's tag-query resource group plus the value.
func aprTagValueARN(app, value string) string {
	return aprGroupARNFor(aprAppTagGroupName(app)) + "/" + value
}

func aprTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func aprConform(op string, out map[string]any) map[string]any {
	m := awsmodels.ServiceCatalogAppRegistryModel()
	if shaped, ok := m.Conform(m.Operations[op].Output, out).(map[string]any); ok {
		return shaped
	}
	return map[string]any{}
}

// aprSpecID reduces a name, id or ARN specifier to the trailing name/id.
func aprSpecID(spec, kind string) (string, bool) {
	if !strings.HasPrefix(spec, "arn:") {
		return spec, true
	}
	marker := ":/" + kind + "/"
	i := strings.Index(spec, marker)
	if i < 0 {
		return "", false
	}
	return spec[i+len(marker):], true
}

func (st *aprStore) appLocked(spec string) (*aprApp, *aprError) {
	if key, ok := aprSpecID(spec, "applications"); ok {
		if a := st.apps[key]; a != nil {
			return a, nil
		}
		if a := st.apps[st.appByName[key]]; a != nil {
			return a, nil
		}
	}
	return nil, aprNotFound("Application %s does not exist.", spec)
}

func (st *aprStore) groupLocked(spec string) (*aprGroup, *aprError) {
	if key, ok := aprSpecID(spec, "attribute-groups"); ok {
		if g := st.groups[key]; g != nil {
			return g, nil
		}
		if g := st.groups[st.groupByName[key]]; g != nil {
			return g, nil
		}
	}
	return nil, aprNotFound("Attribute group %s does not exist.", spec)
}

// aprCheckTags validates caller tags (the model checks key/value patterns).
func aprCheckTags(tags map[string]string) *aprError {
	if len(tags) > aprMaxTags {
		return aprValidation("A resource can have at most %d tags.", aprMaxTags)
	}
	for k := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return aprValidation("Tag keys starting with aws: are reserved.")
		}
	}
	return nil
}

func (st *aprStore) tagsOut(arn string) map[string]any {
	out := map[string]any{}
	for k, v := range st.tags[arn] {
		out[k] = v
	}
	return out
}

func (a *aprApp) summary() map[string]any {
	return map[string]any{
		"id": a.id, "arn": aprAppARN(a.id), "name": a.name, "description": a.description,
		"creationTime": aprTime(a.created), "lastUpdateTime": aprTime(a.updated),
	}
}

func (st *aprStore) appOut(a *aprApp) map[string]any {
	out := a.summary()
	out["tags"] = st.tagsOut(aprAppARN(a.id))
	out["applicationTag"] = map[string]any{"awsApplication": aprAppARN(a.id)}
	return out
}

func (g *aprGroup) summary() map[string]any {
	return map[string]any{
		"id": g.id, "arn": aprGroupARN(g.id), "name": g.name, "description": g.description,
		"creationTime": aprTime(g.created), "lastUpdateTime": aprTime(g.updated),
	}
}

func (st *aprStore) groupOut(g *aprGroup) map[string]any {
	out := g.summary()
	out["tags"] = st.tagsOut(aprGroupARN(g.id))
	return out
}

// createApplication returns the application and whether it was newly created
// (a retried clientToken returns the original).
func (st *aprStore) createApplication(name, description, token string, tags map[string]string) (map[string]any, bool, *aprError) {
	if err := aprCheckTags(tags); err != nil {
		return nil, false, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if id, ok := st.appTokens[token]; ok {
		a := st.apps[id]
		if a == nil || a.name != name {
			return nil, false, aprConflict("The client token %s was already used with different parameters.", token)
		}
		return aprConform("CreateApplication", map[string]any{"application": st.appOut(a)}), false, nil
	}
	if _, ok := st.appByName[name]; ok {
		return nil, false, aprConflict("An application named %s already exists.", name)
	}
	if len(st.apps) >= aprMaxApps {
		return nil, false, aprQuota("An account can have at most %d applications.", aprMaxApps)
	}
	now := time.Now().UTC()
	a := &aprApp{id: aprNewID(), name: name, rgName: name, description: description, created: now, updated: now}
	st.apps[a.id], st.appByName[name], st.appTokens[token] = a, a.id, a.id
	if len(tags) > 0 {
		st.tags[aprAppARN(a.id)] = tags
	}
	return aprConform("CreateApplication", map[string]any{"application": st.appOut(a)}), true, nil
}

func (st *aprStore) getApplication(spec string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(spec)
	if err != nil {
		return nil, err
	}
	out := st.appOut(a)
	out["associatedResourceCount"] = len(a.resources)
	out["integrations"] = map[string]any{
		"resourceGroup":               map[string]any{"state": aprResourceGroupState, "arn": aprGroupARNFor(aprAppGroupName(a.rgName))},
		"applicationTagResourceGroup": map[string]any{"state": aprResourceGroupState, "arn": aprGroupARNFor(aprAppTagGroupName(a.rgName))},
	}
	return aprConform("GetApplication", out), nil
}

func (st *aprStore) updateApplication(spec string, name, description *string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(spec)
	if err != nil {
		return nil, err
	}
	if name != nil && *name != a.name {
		if _, taken := st.appByName[*name]; taken {
			return nil, aprConflict("An application named %s already exists.", *name)
		}
		// The application's resource groups keep the name they were created
		// with (rgName), as in AWS.
		delete(st.appByName, a.name)
		a.name = *name
		st.appByName[a.name] = a.id
	}
	if description != nil {
		a.description = *description
	}
	a.updated = time.Now().UTC()
	return aprConform("UpdateApplication", map[string]any{"application": st.appOut(a)}), nil
}

// deleteApplication returns the deleted summary and the name the resource
// groups were created under, so the caller can remove them.
func (st *aprStore) deleteApplication(spec string) (map[string]any, string, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(spec)
	if err != nil {
		return nil, "", err
	}
	if len(a.groups) > 0 || len(a.resources) > 0 {
		return nil, "", aprValidation("Disassociate all attribute groups and resources from application %s before deleting it.", a.name)
	}
	delete(st.apps, a.id)
	delete(st.appByName, a.name)
	delete(st.tags, aprAppARN(a.id))
	for tok, id := range st.appTokens {
		if id == a.id {
			delete(st.appTokens, tok)
		}
	}
	return aprConform("DeleteApplication", map[string]any{"application": a.summary()}), a.rgName, nil
}

func (st *aprStore) listApplications(token string, max int) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	apps := make([]*aprApp, 0, len(st.apps))
	for _, a := range st.apps {
		apps = append(apps, a)
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].name < apps[j].name })
	start, err := aprPage(token, len(apps))
	if err != nil {
		return nil, err
	}
	end := min(start+max, len(apps))
	items := []any{}
	for _, a := range apps[start:end] {
		items = append(items, a.summary())
	}
	out := map[string]any{"applications": items}
	if end < len(apps) {
		out["nextToken"] = aprToken(end)
	}
	return aprConform("ListApplications", out), nil
}

// aprCheckAttributes requires a JSON object (the model bounds its size).
func aprCheckAttributes(attrs string) *aprError {
	var v map[string]any
	if err := json.Unmarshal([]byte(attrs), &v); err != nil {
		return aprValidation("attributes must be a JSON object.")
	}
	return nil
}

func (st *aprStore) createAttributeGroup(name, description, attrs, token string, tags map[string]string) (map[string]any, *aprError) {
	if err := aprCheckAttributes(attrs); err != nil {
		return nil, err
	}
	if err := aprCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if id, ok := st.groupTokens[token]; ok {
		g := st.groups[id]
		if g == nil || g.name != name {
			return nil, aprConflict("The client token %s was already used with different parameters.", token)
		}
		return aprConform("CreateAttributeGroup", map[string]any{"attributeGroup": st.groupOut(g)}), nil
	}
	if _, ok := st.groupByName[name]; ok {
		return nil, aprConflict("An attribute group named %s already exists.", name)
	}
	if len(st.groups) >= aprMaxGroups {
		return nil, aprQuota("An account can have at most %d attribute groups.", aprMaxGroups)
	}
	now := time.Now().UTC()
	g := &aprGroup{id: aprNewID(), name: name, description: description, attributes: attrs, created: now, updated: now}
	st.groups[g.id], st.groupByName[name], st.groupTokens[token] = g, g.id, g.id
	if len(tags) > 0 {
		st.tags[aprGroupARN(g.id)] = tags
	}
	return aprConform("CreateAttributeGroup", map[string]any{"attributeGroup": st.groupOut(g)}), nil
}

func (st *aprStore) getAttributeGroup(spec string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.groupLocked(spec)
	if err != nil {
		return nil, err
	}
	out := st.groupOut(g)
	out["attributes"] = g.attributes
	return aprConform("GetAttributeGroup", out), nil
}

func (st *aprStore) updateAttributeGroup(spec string, name, description, attrs *string) (map[string]any, *aprError) {
	if attrs != nil {
		if err := aprCheckAttributes(*attrs); err != nil {
			return nil, err
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.groupLocked(spec)
	if err != nil {
		return nil, err
	}
	if name != nil && *name != g.name {
		if _, taken := st.groupByName[*name]; taken {
			return nil, aprConflict("An attribute group named %s already exists.", *name)
		}
		delete(st.groupByName, g.name)
		g.name = *name
		st.groupByName[g.name] = g.id
	}
	if description != nil {
		g.description = *description
	}
	if attrs != nil {
		g.attributes = *attrs
	}
	g.updated = time.Now().UTC()
	return aprConform("UpdateAttributeGroup", map[string]any{"attributeGroup": st.groupOut(g)}), nil
}

func (st *aprStore) deleteAttributeGroup(spec string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g, err := st.groupLocked(spec)
	if err != nil {
		return nil, err
	}
	// ponytail: an associated group is refused, matching DeleteApplication's
	// rule; disassociate it from every application first.
	for _, a := range st.apps {
		if slices.Contains(a.groups, g.id) {
			return nil, aprValidation("Attribute group %s is associated with application %s; disassociate it first.", g.name, a.name)
		}
	}
	delete(st.groups, g.id)
	delete(st.groupByName, g.name)
	delete(st.tags, aprGroupARN(g.id))
	for tok, id := range st.groupTokens {
		if id == g.id {
			delete(st.groupTokens, tok)
		}
	}
	return aprConform("DeleteAttributeGroup", map[string]any{"attributeGroup": g.summary()}), nil
}

func (st *aprStore) listAttributeGroups(token string, max int) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	groups := make([]*aprGroup, 0, len(st.groups))
	for _, g := range st.groups {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].name < groups[j].name })
	start, err := aprPage(token, len(groups))
	if err != nil {
		return nil, err
	}
	end := min(start+max, len(groups))
	items := []any{}
	for _, g := range groups[start:end] {
		items = append(items, g.summary())
	}
	out := map[string]any{"attributeGroups": items}
	if end < len(groups) {
		out["nextToken"] = aprToken(end)
	}
	return aprConform("ListAttributeGroups", out), nil
}

func (st *aprStore) associateAttributeGroup(appSpec, groupSpec string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appSpec)
	if err != nil {
		return nil, err
	}
	g, err := st.groupLocked(groupSpec)
	if err != nil {
		return nil, err
	}
	if slices.Contains(a.groups, g.id) {
		return nil, aprConflict("Attribute group %s is already associated with application %s.", g.name, a.name)
	}
	if len(a.groups) >= aprMaxGroupsPerApp {
		return nil, aprQuota("An application can have at most %d attribute groups.", aprMaxGroupsPerApp)
	}
	a.groups = append(a.groups, g.id)
	return aprConform("AssociateAttributeGroup", map[string]any{"applicationArn": aprAppARN(a.id), "attributeGroupArn": aprGroupARN(g.id)}), nil
}

func (st *aprStore) disassociateAttributeGroup(appSpec, groupSpec string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appSpec)
	if err != nil {
		return nil, err
	}
	g, err := st.groupLocked(groupSpec)
	if err != nil {
		return nil, err
	}
	i := slices.Index(a.groups, g.id)
	if i < 0 {
		return nil, aprNotFound("Attribute group %s isn't associated with application %s.", g.name, a.name)
	}
	a.groups = slices.Delete(a.groups, i, i+1)
	return aprConform("DisassociateAttributeGroup", map[string]any{"applicationArn": aprAppARN(a.id), "attributeGroupArn": aprGroupARN(g.id)}), nil
}

func (st *aprStore) listAssociatedAttributeGroups(appSpec, token string, max int, details bool) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appSpec)
	if err != nil {
		return nil, err
	}
	start, perr := aprPage(token, len(a.groups))
	if perr != nil {
		return nil, perr
	}
	end := min(start+max, len(a.groups))
	items := []any{}
	for _, id := range a.groups[start:end] {
		if !details {
			items = append(items, id)
			continue
		}
		items = append(items, map[string]any{"id": id, "arn": aprGroupARN(id), "name": st.groups[id].name})
	}
	op, key := "ListAssociatedAttributeGroups", "attributeGroups"
	if details {
		op, key = "ListAttributeGroupsForApplication", "attributeGroupsDetails"
	}
	out := map[string]any{key: items}
	if end < len(a.groups) {
		out["nextToken"] = aprToken(end)
	}
	return aprConform(op, out), nil
}

// aprResourceRef is a resolved AssociateResource target: for CFN_STACK the
// handler has already looked the stack up in the CloudFormation store.
type aprResourceRef struct {
	typ, name, arn string
}

func (a *aprApp) resourceIndex(typ, spec string) int {
	for i, r := range a.resources {
		if r.typ == typ && (r.name == spec || r.arn == spec) {
			return i
		}
	}
	return -1
}

func (st *aprStore) associateResource(appSpec string, ref aprResourceRef, options []string) (map[string]any, *aprError) {
	if slices.Contains(options, "APPLY_APPLICATION_TAG") && slices.Contains(options, "SKIP_APPLICATION_TAG") {
		return nil, aprValidation("options can't contain both APPLY_APPLICATION_TAG and SKIP_APPLICATION_TAG.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appSpec)
	if err != nil {
		return nil, err
	}
	if a.resourceIndex(ref.typ, ref.arn) >= 0 {
		return nil, aprConflict("Resource %s is already associated with application %s.", ref.name, a.name)
	}
	if ref.typ == "CFN_STACK" {
		if owner, ok := st.stackOwner[ref.arn]; ok {
			return nil, aprConflict("Stack %s is already associated with application %s.", ref.name, st.apps[owner].name)
		}
	}
	if len(a.resources) >= aprMaxResourcesPerApp {
		return nil, aprQuota("An application can have at most %d associated resources.", aprMaxResourcesPerApp)
	}
	opts := slices.Compact(slices.Sorted(slices.Values(options)))
	a.resources = append(a.resources, &aprResource{typ: ref.typ, name: ref.name, arn: ref.arn, options: opts, associated: time.Now().UTC()})
	if ref.typ == "CFN_STACK" {
		st.stackOwner[ref.arn] = a.id
	}
	return aprConform("AssociateResource", map[string]any{"applicationArn": aprAppARN(a.id), "resourceArn": ref.arn, "options": aprStrings(opts)}), nil
}

func (st *aprStore) disassociateResource(appSpec, typ, spec string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appSpec)
	if err != nil {
		return nil, err
	}
	i := a.resourceIndex(typ, spec)
	if i < 0 {
		return nil, aprNotFound("Resource %s isn't associated with application %s.", spec, a.name)
	}
	r := a.resources[i]
	a.resources = slices.Delete(a.resources, i, i+1)
	if r.typ == "CFN_STACK" {
		delete(st.stackOwner, r.arn)
	}
	return aprConform("DisassociateResource", map[string]any{"applicationArn": aprAppARN(a.id), "resourceArn": r.arn}), nil
}

func (st *aprStore) getAssociatedResource(appSpec, typ, spec string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appSpec)
	if err != nil {
		return nil, err
	}
	i := a.resourceIndex(typ, spec)
	if i < 0 {
		return nil, aprNotFound("Resource %s isn't associated with application %s.", spec, a.name)
	}
	r := a.resources[i]
	out := map[string]any{
		"resource": map[string]any{
			"name": r.name, "arn": r.arn, "associationTime": aprTime(r.associated),
			"integrations": map[string]any{"resourceGroup": map[string]any{"state": aprResourceGroupState, "arn": aprGroupARNFor(aprAppGroupName(a.rgName))}},
		},
		"options": aprStrings(r.options),
	}
	if slices.Contains(r.options, "APPLY_APPLICATION_TAG") {
		// ponytail: the CloudFormation store keeps no per-stack resources, so
		// tagging always succeeds with nothing to list (the resourceTagStatus
		// filter has nothing to filter).
		out["applicationTagResult"] = map[string]any{"applicationTagStatus": "SUCCESS", "resources": []any{}}
	}
	return aprConform("GetAssociatedResource", out), nil
}

func (st *aprStore) listAssociatedResources(appSpec, token string, max int) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.appLocked(appSpec)
	if err != nil {
		return nil, err
	}
	start, perr := aprPage(token, len(a.resources))
	if perr != nil {
		return nil, perr
	}
	end := min(start+max, len(a.resources))
	items := []any{}
	for _, r := range a.resources[start:end] {
		item := map[string]any{"name": r.name, "arn": r.arn, "resourceType": r.typ, "options": aprStrings(r.options)}
		if r.typ == "RESOURCE_TAG_VALUE" {
			item["resourceDetails"] = map[string]any{"tagValue": r.name}
		}
		items = append(items, item)
	}
	out := map[string]any{"resources": items}
	if end < len(a.resources) {
		out["nextToken"] = aprToken(end)
	}
	return aprConform("ListAssociatedResources", out), nil
}

// syncResource reports whether the resource's AppRegistry tags need syncing:
// START_SYNC when it's associated with an application, NO_ACTION otherwise.
func (st *aprStore) syncResource(ref aprResourceRef) map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := map[string]any{"resourceArn": ref.arn, "actionTaken": "NO_ACTION"}
	for _, a := range st.apps {
		if i := a.resourceIndex(ref.typ, ref.arn); i >= 0 {
			out["applicationArn"], out["resourceArn"], out["actionTaken"] = aprAppARN(a.id), a.resources[i].arn, "START_SYNC"
			break
		}
	}
	return aprConform("SyncResource", out)
}

func (st *aprStore) getConfiguration() map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	tq := map[string]any{}
	if st.tagKey != "" {
		tq["tagKey"] = st.tagKey
	}
	return aprConform("GetConfiguration", map[string]any{"configuration": map[string]any{"tagQueryConfiguration": tq}})
}

func (st *aprStore) putConfiguration(tagKey string) *aprError {
	// The model's pattern uses a lookahead Go can't compile, so check it here.
	if tagKey != "" && strings.TrimSpace(tagKey) == "" {
		return aprValidation("tagKey can't be only whitespace.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.tagKey = tagKey
	return nil
}

func (st *aprStore) arnExistsLocked(arn string) bool {
	if id, ok := aprSpecID(arn, "applications"); ok && strings.HasPrefix(arn, "arn:") && st.apps[id] != nil {
		return true
	}
	if id, ok := aprSpecID(arn, "attribute-groups"); ok && strings.HasPrefix(arn, "arn:") && st.groups[id] != nil {
		return true
	}
	return false
}

func (st *aprStore) listTags(arn string) (map[string]any, *aprError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.arnExistsLocked(arn) {
		return nil, aprNotFound("Resource %s does not exist.", arn)
	}
	return aprConform("ListTagsForResource", map[string]any{"tags": st.tagsOut(arn)}), nil
}

func (st *aprStore) tagResource(arn string, tags map[string]string) *aprError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.arnExistsLocked(arn) {
		return aprNotFound("Resource %s does not exist.", arn)
	}
	merged := map[string]string{}
	for k, v := range st.tags[arn] {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	if err := aprCheckTags(merged); err != nil {
		return err
	}
	st.tags[arn] = merged
	return nil
}

func (st *aprStore) untagResource(arn string, keys []string) *aprError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.arnExistsLocked(arn) {
		return aprNotFound("Resource %s does not exist.", arn)
	}
	for _, k := range keys {
		delete(st.tags[arn], k)
	}
	return nil
}

func aprStrings(in []string) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

// Page tokens are base64 offsets (the model's NextToken pattern is base64).
func aprToken(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte("o:" + strconv.Itoa(offset)))
}

func aprPage(token string, total int) (int, *aprError) {
	if token == "" {
		return 0, nil
	}
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil || !strings.HasPrefix(string(raw), "o:") {
		return 0, aprValidation("nextToken is invalid.")
	}
	n, err := strconv.ParseInt(string(raw[2:]), 10, 32)
	if err != nil || n < 0 || int(n) > total {
		return 0, aprValidation("nextToken is invalid.")
	}
	return int(n), nil
}
