package server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	padAccount = "123456789012"
	padRegion  = "us-east-1"
	// Caps keep the store bounded; AWS quotas are lower per account.
	padMaxConnectors    = 1000
	padMaxRegistrations = 1000
	padMaxSPNs          = 10000
	padMaxTemplates     = 10000
	padMaxACEs          = 50000
	padMaxTokens        = 10000
	padMaxTags          = 50
)

type padError struct {
	status       int
	code         string
	msg          string
	reason       string // ValidationException.Reason
	resourceID   string
	resourceType string
}

func padValidation(reason, format string, args ...any) *padError {
	return &padError{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...), reason: reason}
}

func padNotFound(resourceType, id string) *padError {
	return &padError{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: resourceType + " " + id + " does not exist.", resourceID: id, resourceType: resourceType}
}

func padConflict(resourceType, id, format string, args ...any) *padError {
	return &padError{status: http.StatusConflict, code: "ConflictException", msg: fmt.Sprintf(format, args...), resourceID: id, resourceType: resourceType}
}

func padQuota(resourceType string, limit int) *padError {
	return &padError{status: http.StatusPaymentRequired, code: "ServiceQuotaExceededException", msg: fmt.Sprintf("You can have at most %d %s resources.", limit, resourceType), resourceType: resourceType}
}

type padConnector struct {
	arn, caARN, dirID, status, reason string
	vpc                               map[string]any
	created, updated                  time.Time
}

type padRegistration struct {
	arn, dirID, status, reason string
	created, updated           time.Time
}

type padSPN struct {
	connARN, regARN, status string
	created, updated        time.Time
}

type padTemplate struct {
	arn, connARN, name, oid string
	def                     map[string]any
	schema, major, minor    int
	created, updated        time.Time
}

type padACE struct {
	templateARN, sid, display string
	rights                    map[string]any
	created, updated          time.Time
}

type padStore struct {
	mu            sync.Mutex
	seq           int
	connectors    map[string]*padConnector
	registrations map[string]*padRegistration
	spns          map[string]*padSPN      // by regARN|connARN
	templates     map[string]*padTemplate // by ARN
	aces          map[string]*padACE      // by templateARN|SID
	tokens        map[string]string       // op|ClientToken -> ARN
	tokenOrder    []string
	tags          map[string]map[string]string
}

func newPCAConnectorADStore() *padStore {
	return &padStore{
		connectors: map[string]*padConnector{}, registrations: map[string]*padRegistration{},
		spns: map[string]*padSPN{}, templates: map[string]*padTemplate{}, aces: map[string]*padACE{},
		tokens: map[string]string{}, tags: map[string]map[string]string{},
	}
}

func padConnectorARN(id string) string {
	return "arn:aws:pca-connector-ad:" + padRegion + ":" + padAccount + ":connector/" + id
}

func padRegistrationARN(dirID string) string {
	return "arn:aws:pca-connector-ad:" + padRegion + ":" + padAccount + ":directory-registration/" + dirID
}

func padTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// padSettle moves CREATING to its settled state; reads call it after
// rendering, so a new resource reports CREATING once.
// ponytail: settle-on-read instead of timers; there is no background work.
func padSettle(status *string, failed bool) {
	if *status == "CREATING" {
		if failed {
			*status = "FAILED"
		} else {
			*status = "ACTIVE"
		}
	}
}

// token returns the ARN created earlier with this client token, if any.
func (st *padStore) tokenLocked(op, token string) string {
	if token == "" {
		return ""
	}
	return st.tokens[op+"|"+token]
}

func (st *padStore) rememberLocked(op, token, arn string) {
	if token == "" {
		return
	}
	key := op + "|" + token
	if len(st.tokenOrder) >= padMaxTokens {
		delete(st.tokens, st.tokenOrder[0])
		st.tokenOrder = st.tokenOrder[1:]
	}
	st.tokens[key] = arn
	st.tokenOrder = append(st.tokenOrder, key)
}

func (st *padStore) setTagsLocked(arn string, tags map[string]any) *padError {
	if len(tags) == 0 {
		return nil
	}
	cur := st.tags[arn]
	if cur == nil {
		cur = map[string]string{}
	}
	next := make(map[string]string, len(cur)+len(tags))
	for k, v := range cur {
		next[k] = v
	}
	for k, v := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return padValidation("FIELD_VALIDATION_FAILED", "Tag keys can't start with aws:.")
		}
		next[k], _ = v.(string)
	}
	if len(next) > padMaxTags {
		return padValidation("FIELD_VALIDATION_FAILED", "A resource can have at most %d tags.", padMaxTags)
	}
	st.tags[arn] = next
	return nil
}

// ---- connectors ----

func (c *padConnector) out() map[string]any {
	o := map[string]any{
		"Arn": c.arn, "CertificateAuthorityArn": c.caARN, "DirectoryId": c.dirID, "Status": c.status,
		"VpcInformation": padCopyMap(c.vpc), "CreatedAt": padTime(c.created), "UpdatedAt": padTime(c.updated),
		"CertificateEnrollmentPolicyServerEndpoint": "https://" + c.arn[strings.LastIndex(c.arn, "/")+1:] + ".pca-connector-ad." + padRegion + ".amazonaws.com/cep",
	}
	if c.reason != "" {
		o["StatusReason"] = c.reason
	}
	return o
}

func (st *padStore) createConnector(token, caARN, dirID string, vpc, tags map[string]any) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if arn := st.tokenLocked("CreateConnector", token); arn != "" {
		return map[string]any{"ConnectorArn": arn}, nil
	}
	if len(st.connectors) >= padMaxConnectors {
		return nil, padQuota("CONNECTOR", padMaxConnectors)
	}
	now := time.Now().UTC()
	c := &padConnector{arn: padConnectorARN(uuid.NewString()), caARN: caARN, dirID: dirID, status: "CREATING", vpc: padCopyMap(vpc), created: now, updated: now}
	if c.vpc["IpAddressType"] == nil {
		c.vpc["IpAddressType"] = "IPV4"
	}
	if err := st.setTagsLocked(c.arn, tags); err != nil {
		return nil, err
	}
	st.connectors[c.arn] = c
	st.rememberLocked("CreateConnector", token, c.arn)
	return map[string]any{"ConnectorArn": c.arn}, nil
}

func (st *padStore) getConnector(arn string) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c := st.connectors[arn]
	if c == nil {
		return nil, padNotFound("CONNECTOR", arn)
	}
	out := c.out()
	padSettle(&c.status, false)
	return map[string]any{"Connector": out}, nil
}

func (st *padStore) listConnectors(token string, max int) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var items []any
	for _, arn := range padSortedKeys(st.connectors) {
		c := st.connectors[arn]
		items = append(items, c.out())
		padSettle(&c.status, false)
	}
	return padPage("Connectors", items, token, max)
}

// deleteConnector removes a connector and its templates. A connector still
// named by a service principal name is in use.
// ponytail: deletion completes at once instead of reporting DELETING.
func (st *padStore) deleteConnector(arn string) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.connectors[arn] == nil {
		return padNotFound("CONNECTOR", arn)
	}
	for _, s := range st.spns {
		if s.connARN == arn {
			return padConflict("CONNECTOR", arn, "Connector %s still has service principal names; delete them first.", arn)
		}
	}
	for tarn, t := range st.templates {
		if t.connARN == arn {
			st.deleteTemplateLocked(tarn)
		}
	}
	delete(st.connectors, arn)
	delete(st.tags, arn)
	return nil
}

// ---- directory registrations ----

func (r *padRegistration) out() map[string]any {
	o := map[string]any{"Arn": r.arn, "DirectoryId": r.dirID, "Status": r.status, "CreatedAt": padTime(r.created), "UpdatedAt": padTime(r.updated)}
	if r.reason != "" && r.status == "FAILED" {
		o["StatusReason"] = r.reason
	}
	return o
}

// createDirectoryRegistration registers a directory. A directory that isn't
// Active settles to FAILED with DIRECTORY_NOT_ACTIVE, as in AWS.
func (st *padStore) createDirectoryRegistration(token, dirID string, active bool, tags map[string]any) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if arn := st.tokenLocked("CreateDirectoryRegistration", token); arn != "" {
		return map[string]any{"DirectoryRegistrationArn": arn}, nil
	}
	arn := padRegistrationARN(dirID)
	if st.registrations[arn] != nil {
		return nil, padConflict("DIRECTORY_REGISTRATION", arn, "Directory %s is already registered.", dirID)
	}
	if len(st.registrations) >= padMaxRegistrations {
		return nil, padQuota("DIRECTORY_REGISTRATION", padMaxRegistrations)
	}
	now := time.Now().UTC()
	r := &padRegistration{arn: arn, dirID: dirID, status: "CREATING", created: now, updated: now}
	if !active {
		r.reason = "DIRECTORY_NOT_ACTIVE"
	}
	if err := st.setTagsLocked(arn, tags); err != nil {
		return nil, err
	}
	st.registrations[arn] = r
	st.rememberLocked("CreateDirectoryRegistration", token, arn)
	return map[string]any{"DirectoryRegistrationArn": arn}, nil
}

func (st *padStore) getDirectoryRegistration(arn string) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r := st.registrations[arn]
	if r == nil {
		return nil, padNotFound("DIRECTORY_REGISTRATION", arn)
	}
	out := r.out()
	padSettle(&r.status, r.reason != "")
	return map[string]any{"DirectoryRegistration": out}, nil
}

func (st *padStore) listDirectoryRegistrations(token string, max int) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	var items []any
	for _, arn := range padSortedKeys(st.registrations) {
		r := st.registrations[arn]
		items = append(items, r.out())
		padSettle(&r.status, r.reason != "")
	}
	return padPage("DirectoryRegistrations", items, token, max)
}

// deleteDirectoryRegistration is idempotent (the operation has no
// ResourceNotFoundException); a registration with service principal names
// still attached is in use.
func (st *padStore) deleteDirectoryRegistration(arn string) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.registrations[arn] == nil {
		return nil
	}
	for _, s := range st.spns {
		if s.regARN == arn {
			return padConflict("DIRECTORY_REGISTRATION", arn, "Directory registration %s still has service principal names; delete them first.", arn)
		}
	}
	delete(st.registrations, arn)
	delete(st.tags, arn)
	return nil
}

// ---- service principal names ----

func (s *padSPN) out() map[string]any {
	return map[string]any{"ConnectorArn": s.connARN, "DirectoryRegistrationArn": s.regARN, "Status": s.status, "CreatedAt": padTime(s.created), "UpdatedAt": padTime(s.updated)}
}

func (st *padStore) createSPN(regARN, connARN string) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	r := st.registrations[regARN]
	if r == nil {
		return padNotFound("DIRECTORY_REGISTRATION", regARN)
	}
	c := st.connectors[connARN]
	if c == nil {
		return padNotFound("CONNECTOR", connARN)
	}
	if c.dirID != r.dirID {
		return padValidation("MISMATCHED_CONNECTOR", "Connector %s serves directory %s, not %s.", connARN, c.dirID, r.dirID)
	}
	key := regARN + "|" + connARN
	if st.spns[key] != nil {
		return padConflict("SERVICE_PRINCIPAL_NAME", connARN, "A service principal name for connector %s already exists in %s.", connARN, regARN)
	}
	if len(st.spns) >= padMaxSPNs {
		return padQuota("SERVICE_PRINCIPAL_NAME", padMaxSPNs)
	}
	now := time.Now().UTC()
	st.spns[key] = &padSPN{connARN: connARN, regARN: regARN, status: "CREATING", created: now, updated: now}
	return nil
}

func (st *padStore) getSPN(regARN, connARN string) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.spns[regARN+"|"+connARN]
	if s == nil {
		return nil, padNotFound("SERVICE_PRINCIPAL_NAME", connARN)
	}
	out := s.out()
	padSettle(&s.status, false)
	return map[string]any{"ServicePrincipalName": out}, nil
}

func (st *padStore) listSPNs(regARN, token string, max int) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.registrations[regARN] == nil {
		return nil, padNotFound("DIRECTORY_REGISTRATION", regARN)
	}
	var items []any
	for _, key := range padSortedKeys(st.spns) {
		if s := st.spns[key]; s.regARN == regARN {
			items = append(items, s.out())
			padSettle(&s.status, false)
		}
	}
	return padPage("ServicePrincipalNames", items, token, max)
}

// deleteSPN is idempotent: the operation has no ResourceNotFoundException.
func (st *padStore) deleteSPN(regARN, connARN string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.spns, regARN+"|"+connARN)
}

// ---- templates ----

func (t *padTemplate) out() map[string]any {
	return map[string]any{
		"Arn": t.arn, "ConnectorArn": t.connARN, "Name": t.name, "ObjectIdentifier": t.oid,
		"Definition": padCopyMap(t.def), "PolicySchema": t.schema, "Status": "ACTIVE",
		"Revision":  map[string]any{"MajorRevision": t.major, "MinorRevision": t.minor},
		"CreatedAt": padTime(t.created), "UpdatedAt": padTime(t.updated),
	}
}

func (st *padStore) createTemplate(token, connARN, name string, def map[string]any, schema int, tags map[string]any) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if arn := st.tokenLocked("CreateTemplate", token); arn != "" {
		return map[string]any{"TemplateArn": arn}, nil
	}
	if st.connectors[connARN] == nil {
		return nil, padNotFound("CONNECTOR", connARN)
	}
	for _, t := range st.templates {
		if t.connARN == connARN && strings.EqualFold(t.name, name) {
			return nil, padConflict("TEMPLATE", t.arn, "Connector %s already has a template named %s.", connARN, name)
		}
	}
	if len(st.templates) >= padMaxTemplates {
		return nil, padQuota("TEMPLATE", padMaxTemplates)
	}
	st.seq++
	now := time.Now().UTC()
	t := &padTemplate{
		arn: connARN + "/template/" + uuid.NewString(), connARN: connARN, name: name, def: padCopyMap(def),
		schema: schema, major: 1, created: now, updated: now,
		// Microsoft's certificate template OID arc plus a per-template suffix.
		oid: "1.3.6.1.4.1.311.21.8." + strconv.Itoa(1000000+st.seq) + "." + strconv.Itoa(st.seq%1000),
	}
	if err := st.setTagsLocked(t.arn, tags); err != nil {
		return nil, err
	}
	st.templates[t.arn] = t
	st.rememberLocked("CreateTemplate", token, t.arn)
	return map[string]any{"TemplateArn": t.arn}, nil
}

func (st *padStore) getTemplate(arn string) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t := st.templates[arn]
	if t == nil {
		return nil, padNotFound("TEMPLATE", arn)
	}
	return map[string]any{"Template": t.out()}, nil
}

func (st *padStore) listTemplates(connARN, token string, max int) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.connectors[connARN] == nil {
		return nil, padNotFound("CONNECTOR", connARN)
	}
	var items []any
	for _, arn := range padSortedKeys(st.templates) {
		if t := st.templates[arn]; t.connARN == connARN {
			items = append(items, t.out())
		}
	}
	return padPage("Templates", items, token, max)
}

// updateTemplate replaces the definition (minor revision +1) and/or
// re-enrolls all holders (major revision +1), as the API documents.
func (st *padStore) updateTemplate(arn string, def map[string]any, schema int, reenroll bool) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	t := st.templates[arn]
	if t == nil {
		return padNotFound("TEMPLATE", arn)
	}
	if def != nil {
		t.def, t.schema = padCopyMap(def), schema
		t.minor++
	}
	if reenroll {
		t.major++
	}
	t.updated = time.Now().UTC()
	return nil
}

func (st *padStore) deleteTemplate(arn string) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.templates[arn] == nil {
		return padNotFound("TEMPLATE", arn)
	}
	st.deleteTemplateLocked(arn)
	return nil
}

func (st *padStore) deleteTemplateLocked(arn string) {
	for key, a := range st.aces {
		if a.templateARN == arn {
			delete(st.aces, key)
		}
	}
	delete(st.templates, arn)
	delete(st.tags, arn)
}

// ---- template group access control entries ----

func (a *padACE) out() map[string]any {
	return map[string]any{
		"TemplateArn": a.templateARN, "GroupSecurityIdentifier": a.sid, "GroupDisplayName": a.display,
		"AccessRights": padCopyMap(a.rights), "CreatedAt": padTime(a.created), "UpdatedAt": padTime(a.updated),
	}
}

func (st *padStore) createACE(token, templateARN, sid, display string, rights map[string]any) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	key := templateARN + "|" + sid
	if st.tokenLocked("CreateTemplateGroupAccessControlEntry", token) == key {
		return nil
	}
	if st.templates[templateARN] == nil {
		return padNotFound("TEMPLATE", templateARN)
	}
	if st.aces[key] != nil {
		return padConflict("ACCESS_CONTROL_ENTRY", sid, "Template %s already has an access control entry for %s.", templateARN, sid)
	}
	if len(st.aces) >= padMaxACEs {
		return padQuota("ACCESS_CONTROL_ENTRY", padMaxACEs)
	}
	now := time.Now().UTC()
	st.aces[key] = &padACE{templateARN: templateARN, sid: sid, display: display, rights: padCopyMap(rights), created: now, updated: now}
	st.rememberLocked("CreateTemplateGroupAccessControlEntry", token, key)
	return nil
}

func (st *padStore) getACE(templateARN, sid string) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.templates[templateARN] == nil {
		return nil, padNotFound("TEMPLATE", templateARN)
	}
	a := st.aces[templateARN+"|"+sid]
	if a == nil {
		return nil, padNotFound("ACCESS_CONTROL_ENTRY", sid)
	}
	return map[string]any{"AccessControlEntry": a.out()}, nil
}

func (st *padStore) listACEs(templateARN, token string, max int) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.templates[templateARN] == nil {
		return nil, padNotFound("TEMPLATE", templateARN)
	}
	var items []any
	for _, key := range padSortedKeys(st.aces) {
		if a := st.aces[key]; a.templateARN == templateARN {
			items = append(items, a.out())
		}
	}
	return padPage("AccessControlEntries", items, token, max)
}

// updateACE merges the given access rights into the entry.
func (st *padStore) updateACE(templateARN, sid string, display *string, rights map[string]any) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.templates[templateARN] == nil {
		return padNotFound("TEMPLATE", templateARN)
	}
	a := st.aces[templateARN+"|"+sid]
	if a == nil {
		return padNotFound("ACCESS_CONTROL_ENTRY", sid)
	}
	next := padCopyMap(a.rights)
	for k, v := range rights {
		next[k] = v
	}
	a.rights = next
	if display != nil {
		a.display = *display
	}
	a.updated = time.Now().UTC()
	return nil
}

func (st *padStore) deleteACE(templateARN, sid string) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.templates[templateARN] == nil {
		return padNotFound("TEMPLATE", templateARN)
	}
	key := templateARN + "|" + sid
	if st.aces[key] == nil {
		return padNotFound("ACCESS_CONTROL_ENTRY", sid)
	}
	delete(st.aces, key)
	return nil
}

// ---- tags ----

func (st *padStore) existsLocked(arn string) bool {
	return st.connectors[arn] != nil || st.registrations[arn] != nil || st.templates[arn] != nil
}

func (st *padStore) listTags(arn string) (map[string]any, *padError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return nil, padNotFound("RESOURCE", arn)
	}
	tags := map[string]any{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return map[string]any{"Tags": tags}, nil
}

func (st *padStore) tag(arn string, tags map[string]any) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return padNotFound("RESOURCE", arn)
	}
	return st.setTagsLocked(arn, tags)
}

func (st *padStore) untag(arn string, keys []string) *padError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.existsLocked(arn) {
		return padNotFound("RESOURCE", arn)
	}
	for _, k := range keys {
		delete(st.tags[arn], k)
	}
	return nil
}

// ---- helpers ----

func padSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// padPage slices items by an offset token. Tokens are padded base64url, the
// shape the model's NextToken pattern requires.
func padPage(field string, items []any, token string, max int) (map[string]any, *padError) {
	start := 0
	if token != "" {
		raw, err := base64.URLEncoding.DecodeString(token)
		n, perr := strconv.ParseInt(strings.TrimPrefix(string(raw), "o:"), 10, 32)
		if err != nil || perr != nil || !strings.HasPrefix(string(raw), "o:") || n < 0 || int(n) > len(items) {
			return nil, padValidation("FIELD_VALIDATION_FAILED", "NextToken is not valid.")
		}
		start = int(n)
	}
	end := min(start+max, len(items))
	page := items[start:end]
	if page == nil {
		page = []any{}
	}
	out := map[string]any{field: page}
	if end < len(items) {
		out["NextToken"] = base64.URLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(end)))
	}
	return out, nil
}

// padCopyMap copies a decoded JSON object so stored state never aliases
// request or response maps.
func padCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = padDeepCopy(v)
	}
	return out
}

func padDeepCopy(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return padCopyMap(t)
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = padDeepCopy(x)
		}
		return out
	}
	return v
}
