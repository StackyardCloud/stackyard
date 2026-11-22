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

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	r53pAccount = "123456789012"
	r53pRegion  = "us-east-1"
	// Caps keep the store bounded; AWS quotas are lower per account.
	r53pMaxProfiles     = 1000
	r53pMaxAssocs       = 5000
	r53pMaxResAssocs    = 10000
	r53pMaxTags         = 200
	r53pMaxPropertyJSON = 4096
)

type r53pError struct {
	status       int
	code         string
	msg          string
	fieldName    string
	resourceType string
}

func r53pValidation(format string, args ...any) *r53pError {
	return &r53pError{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

func r53pInvalid(field, format string, args ...any) *r53pError {
	return &r53pError{status: http.StatusBadRequest, code: "InvalidParameterException", msg: fmt.Sprintf(format, args...), fieldName: field}
}

func r53pNotFound(resourceType, format string, args ...any) *r53pError {
	return &r53pError{status: http.StatusNotFound, code: "ResourceNotFoundException", msg: fmt.Sprintf(format, args...), resourceType: resourceType}
}

func r53pConflict(format string, args ...any) *r53pError {
	return &r53pError{status: http.StatusConflict, code: "ConflictException", msg: fmt.Sprintf(format, args...)}
}

func r53pExists(resourceType, format string, args ...any) *r53pError {
	return &r53pError{status: http.StatusBadRequest, code: "ResourceExistsException", msg: fmt.Sprintf(format, args...), resourceType: resourceType}
}

func r53pLimit(resourceType, format string, args ...any) *r53pError {
	return &r53pError{status: http.StatusBadRequest, code: "LimitExceededException", msg: fmt.Sprintf(format, args...), resourceType: resourceType}
}

type r53pProfile struct {
	id, name, token, status string
	created, modified       time.Time
}

type r53pAssoc struct {
	id, name, profileID, vpcID, status string
	priority                           int // 0 = unset
	created, modified                  time.Time
}

type r53pResAssoc struct {
	id, name, profileID, arn, resourceType, properties, status string
	created, modified                                          time.Time
}

type r53pStore struct {
	mu        sync.Mutex
	seq       int
	profiles  map[string]*r53pProfile
	byToken   map[string]string
	assocs    map[string]*r53pAssoc
	resAssocs map[string]*r53pResAssoc
	tags      map[string]map[string]string // by ARN
}

func newRoute53ProfilesStore() *r53pStore {
	return &r53pStore{
		profiles: map[string]*r53pProfile{}, byToken: map[string]string{},
		assocs: map[string]*r53pAssoc{}, resAssocs: map[string]*r53pResAssoc{},
		tags: map[string]map[string]string{},
	}
}

func (st *r53pStore) newIDLocked(prefix string) string {
	st.seq++
	// Multiplying by an odd constant is a bijection mod 2^64, so IDs never collide.
	return fmt.Sprintf("%s-%017x", prefix, uint64(st.seq)*0x9e3779b97f4a7c15)
}

func r53pProfileARN(id string) string {
	return "arn:aws:route53profiles:" + r53pRegion + ":" + r53pAccount + ":profile/" + id
}

func r53pAssocARN(id string) string {
	return "arn:aws:route53profiles:" + r53pRegion + ":" + r53pAccount + ":profile-association/" + id
}

func r53pTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// settle moves a transitional status to COMPLETE; reads call it so a fresh
// resource reports CREATING/UPDATING once, then COMPLETE.
// ponytail: settle-on-read instead of timers; there is no background work.
func r53pSettle(status *string) {
	if *status == "CREATING" || *status == "UPDATING" {
		*status = "COMPLETE"
	}
}

func (p *r53pProfile) out() map[string]any {
	return map[string]any{
		"Id": p.id, "Arn": r53pProfileARN(p.id), "Name": p.name, "OwnerId": r53pAccount, "Status": p.status,
		"StatusMessage": r53pStatusMessage(p.status), "ShareStatus": "NOT_SHARED", "ClientToken": p.token,
		"CreationTime": r53pTime(p.created), "ModificationTime": r53pTime(p.modified),
	}
}

func (a *r53pAssoc) out() map[string]any {
	m := map[string]any{
		"Id": a.id, "Name": a.name, "OwnerId": r53pAccount, "ProfileId": a.profileID, "ResourceId": a.vpcID,
		"Status": a.status, "StatusMessage": r53pStatusMessage(a.status),
		"CreationTime": r53pTime(a.created), "ModificationTime": r53pTime(a.modified),
	}
	if a.priority != 0 {
		m["Priority"] = a.priority
	}
	return m
}

func (a *r53pResAssoc) out() map[string]any {
	m := map[string]any{
		"Id": a.id, "Name": a.name, "OwnerId": r53pAccount, "ProfileId": a.profileID, "ResourceArn": a.arn,
		"ResourceType": a.resourceType, "Status": a.status, "StatusMessage": r53pStatusMessage(a.status),
		"CreationTime": r53pTime(a.created), "ModificationTime": r53pTime(a.modified),
	}
	if a.properties != "" {
		m["ResourceProperties"] = a.properties
	}
	return m
}

func r53pStatusMessage(status string) string {
	switch status {
	case "COMPLETE":
		return "Completed"
	case "DELETED":
		return "Deleted"
	}
	return strings.ToLower(status[:1]) + strings.ToLower(status[1:])
}

// conform shapes an operation's output to the model (dropping anything it
// doesn't define); always called with st.mu held.
func r53pConform(op string, out map[string]any) map[string]any {
	m := awsmodels.Route53ProfilesModel()
	if shaped, ok := m.Conform(m.Operations[op].Output, out).(map[string]any); ok {
		return shaped
	}
	return map[string]any{}
}

// Resource types a Profile can carry, keyed by ARN shape. Resolver rules,
// query log configs and DNS Firewall rule groups belong to Route 53 Resolver.
// ponytail: accepted by ARN pattern, because Stackyard's Route 53 store keeps no
// hosted zones and Resolver isn't emulated here; look them up once those exist.
var r53pResourceTypes = []struct {
	re  *regexp.Regexp
	typ string
}{
	{regexp.MustCompile(`^arn:aws:route53:::hostedzone/[A-Z0-9]{1,32}$`), "PRIVATE_HOSTED_ZONE"},
	{regexp.MustCompile(`^arn:aws:route53resolver:[a-z0-9-]+:\d{12}:resolver-rule/rslvr-(rr|autodefined-rr)-[a-z0-9]+$`), "RESOLVER_RULE"},
	{regexp.MustCompile(`^arn:aws:route53resolver:[a-z0-9-]+:\d{12}:firewall-rule-group/rslvr-frg-[a-z0-9]+$`), "FIREWALL_RULE_GROUP"},
	{regexp.MustCompile(`^arn:aws:route53resolver:[a-z0-9-]+:\d{12}:resolver-query-log-config/rslvr-qlc-[a-z0-9]+$`), "RESOLVER_QUERY_LOGGING_CONFIG"},
	{regexp.MustCompile(`^arn:aws:ec2:[a-z0-9-]+:\d{12}:vpc-endpoint/vpce-[a-z0-9]+$`), "INTERFACE_VPC_ENDPOINT"},
}

func r53pResourceType(arn string) string {
	for _, t := range r53pResourceTypes {
		if t.re.MatchString(arn) {
			return t.typ
		}
	}
	return ""
}

// r53pCheckProperties validates ResourceProperties: a JSON object; DNS
// Firewall rule groups need {"priority": 100..9900}.
func r53pCheckProperties(typ, props string) *r53pError {
	if props == "" {
		if typ == "FIREWALL_RULE_GROUP" {
			return r53pInvalid("ResourceProperties", `A DNS Firewall rule group needs ResourceProperties with a priority, e.g. {"priority":100}.`)
		}
		return nil
	}
	if len(props) > r53pMaxPropertyJSON {
		return r53pInvalid("ResourceProperties", "ResourceProperties is too long.")
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(props), &obj); err != nil {
		return r53pInvalid("ResourceProperties", "ResourceProperties must be a JSON object.")
	}
	if typ != "FIREWALL_RULE_GROUP" {
		return nil
	}
	p, ok := obj["priority"].(float64)
	if !ok || p != float64(int(p)) || p < 100 || p > 9900 {
		return r53pInvalid("ResourceProperties", "priority must be an integer between 100 and 9900.")
	}
	return nil
}

func r53pTagMap(list []any) map[string]string {
	out := map[string]string{}
	for _, item := range list {
		if tag, ok := item.(map[string]any); ok {
			k, _ := tag["Key"].(string)
			v, _ := tag["Value"].(string)
			out[k] = v
		}
	}
	return out
}

func (st *r53pStore) setTagsLocked(arn string, add map[string]string) *r53pError {
	cur := st.tags[arn]
	merged := make(map[string]string, len(cur)+len(add))
	for k, v := range cur {
		merged[k] = v
	}
	for k, v := range add {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return r53pValidation("Tag keys can't start with aws:.")
		}
		merged[k] = v
	}
	if len(merged) > r53pMaxTags {
		return r53pValidation("A resource can have at most %d tags.", r53pMaxTags)
	}
	st.tags[arn] = merged
	return nil
}

func (st *r53pStore) createProfile(name, token string, tags []any) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if id, ok := st.byToken[token]; ok {
		if p := st.profiles[id]; p != nil {
			if p.name != name {
				return nil, r53pConflict("ClientToken %s was already used for a different profile.", token)
			}
			return r53pConform("CreateProfile", map[string]any{"Profile": p.out()}), nil
		}
	}
	if len(st.profiles) >= r53pMaxProfiles {
		return nil, r53pLimit("PROFILE", "The account has reached its limit of %d profiles.", r53pMaxProfiles)
	}
	now := time.Now().UTC()
	p := &r53pProfile{id: st.newIDLocked("rp"), name: name, token: token, status: "CREATING", created: now, modified: now}
	if err := st.setTagsLocked(r53pProfileARN(p.id), r53pTagMap(tags)); err != nil {
		return nil, err
	}
	st.profiles[p.id] = p
	st.byToken[token] = p.id
	return r53pConform("CreateProfile", map[string]any{"Profile": p.out()}), nil
}

func (st *r53pStore) getProfileLocked(id string) (*r53pProfile, *r53pError) {
	p := st.profiles[id]
	if p == nil {
		return nil, r53pNotFound("PROFILE", "Profile %s does not exist.", id)
	}
	return p, nil
}

func (st *r53pStore) getProfile(id string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.getProfileLocked(id)
	if err != nil {
		return nil, err
	}
	r53pSettle(&p.status)
	return r53pConform("GetProfile", map[string]any{"Profile": p.out()}), nil
}

func (st *r53pStore) deleteProfile(id string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.getProfileLocked(id)
	if err != nil {
		return nil, err
	}
	for _, a := range st.assocs {
		if a.profileID == id {
			return nil, r53pConflict("Profile %s is still associated with VPC %s; disassociate it first.", id, a.vpcID)
		}
	}
	for rid, ra := range st.resAssocs {
		if ra.profileID == id {
			delete(st.resAssocs, rid)
		}
	}
	delete(st.profiles, id)
	delete(st.byToken, p.token)
	delete(st.tags, r53pProfileARN(id))
	p.status, p.modified = "DELETED", time.Now().UTC()
	return r53pConform("DeleteProfile", map[string]any{"Profile": p.out()}), nil
}

// r53pPage slices ids sorted for stable paging; the token is an offset.
func r53pPage(ids []string, nextToken string, maxResults int) ([]string, string, *r53pError) {
	sort.Strings(ids)
	start := 0
	if nextToken != "" {
		n, err := strconv.ParseInt(strings.TrimPrefix(nextToken, "r53p-"), 10, 32)
		if err != nil || !strings.HasPrefix(nextToken, "r53p-") || n < 0 || int(n) > len(ids) {
			return nil, "", &r53pError{status: http.StatusBadRequest, code: "InvalidNextTokenException", msg: "The NextToken is not valid."}
		}
		start = int(n)
	}
	end := min(start+maxResults, len(ids))
	token := ""
	if end < len(ids) {
		token = "r53p-" + strconv.Itoa(end)
	}
	return ids[start:end], token, nil
}

func (st *r53pStore) listProfiles(nextToken string, maxResults int) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ids := make([]string, 0, len(st.profiles))
	for id := range st.profiles {
		ids = append(ids, id)
	}
	page, token, err := r53pPage(ids, nextToken, maxResults)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, id := range page {
		p := st.profiles[id]
		r53pSettle(&p.status)
		items = append(items, map[string]any{"Id": p.id, "Arn": r53pProfileARN(p.id), "Name": p.name, "ShareStatus": "NOT_SHARED"})
	}
	out := map[string]any{"ProfileSummaries": items}
	if token != "" {
		out["NextToken"] = token
	}
	return r53pConform("ListProfiles", out), nil
}

func (st *r53pStore) associateProfile(profileID, vpcID, name string, priority int, tags []any) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getProfileLocked(profileID); err != nil {
		return nil, err
	}
	for _, a := range st.assocs {
		if a.vpcID != vpcID {
			continue
		}
		if a.profileID == profileID {
			return nil, r53pExists("PROFILE_ASSOCIATION", "Profile %s is already associated with VPC %s.", profileID, vpcID)
		}
		return nil, r53pConflict("VPC %s is already associated with profile %s; a VPC can have one profile.", vpcID, a.profileID)
	}
	if len(st.assocs) >= r53pMaxAssocs {
		return nil, r53pLimit("PROFILE_ASSOCIATION", "The account has reached its limit of %d profile associations.", r53pMaxAssocs)
	}
	now := time.Now().UTC()
	a := &r53pAssoc{id: st.newIDLocked("rpassoc"), name: name, profileID: profileID, vpcID: vpcID, priority: priority, status: "CREATING", created: now, modified: now}
	if err := st.setTagsLocked(r53pAssocARN(a.id), r53pTagMap(tags)); err != nil {
		return nil, err
	}
	st.assocs[a.id] = a
	return r53pConform("AssociateProfile", map[string]any{"ProfileAssociation": a.out()}), nil
}

func (st *r53pStore) getAssocLocked(id string) (*r53pAssoc, *r53pError) {
	a := st.assocs[id]
	if a == nil {
		return nil, r53pNotFound("PROFILE_ASSOCIATION", "Profile association %s does not exist.", id)
	}
	return a, nil
}

func (st *r53pStore) getAssoc(id string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.getAssocLocked(id)
	if err != nil {
		return nil, err
	}
	r53pSettle(&a.status)
	return r53pConform("GetProfileAssociation", map[string]any{"ProfileAssociation": a.out()}), nil
}

func (st *r53pStore) updateAssoc(id string, name *string, priority int) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a, err := st.getAssocLocked(id)
	if err != nil {
		return nil, err
	}
	if a.status == "CREATING" || a.status == "UPDATING" {
		return nil, r53pConflict("Profile association %s is %s; retry once it is COMPLETE.", id, a.status)
	}
	if name != nil {
		a.name = *name
	}
	if priority != 0 {
		a.priority = priority
	}
	a.status, a.modified = "UPDATING", time.Now().UTC()
	return r53pConform("UpdateProfileAssociation", map[string]any{"ProfileAssociation": a.out()}), nil
}

func (st *r53pStore) disassociateProfile(profileID, vpcID string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getProfileLocked(profileID); err != nil {
		return nil, err
	}
	for id, a := range st.assocs {
		if a.profileID == profileID && a.vpcID == vpcID {
			delete(st.assocs, id)
			delete(st.tags, r53pAssocARN(id))
			a.status, a.modified = "DELETING", time.Now().UTC()
			return r53pConform("DisassociateProfile", map[string]any{"ProfileAssociation": a.out()}), nil
		}
	}
	return nil, r53pNotFound("PROFILE_ASSOCIATION", "Profile %s is not associated with %s.", profileID, vpcID)
}

func (st *r53pStore) listAssocs(profileID, vpcID, nextToken string, maxResults int) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ids := []string{}
	for id, a := range st.assocs {
		if (profileID == "" || a.profileID == profileID) && (vpcID == "" || a.vpcID == vpcID) {
			ids = append(ids, id)
		}
	}
	page, token, err := r53pPage(ids, nextToken, maxResults)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, id := range page {
		a := st.assocs[id]
		r53pSettle(&a.status)
		items = append(items, a.out())
	}
	out := map[string]any{"ProfileAssociations": items}
	if token != "" {
		out["NextToken"] = token
	}
	return r53pConform("ListProfileAssociations", out), nil
}

func (st *r53pStore) associateResource(profileID, arn, typ, name, props string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getProfileLocked(profileID); err != nil {
		return nil, err
	}
	for _, ra := range st.resAssocs {
		if ra.profileID == profileID && ra.arn == arn {
			return nil, r53pConflict("Resource %s is already associated with profile %s.", arn, profileID)
		}
	}
	if len(st.resAssocs) >= r53pMaxResAssocs {
		return nil, r53pLimit("PROFILE_RESOURCE_ASSOCIATION", "The account has reached its limit of %d profile resource associations.", r53pMaxResAssocs)
	}
	now := time.Now().UTC()
	ra := &r53pResAssoc{id: st.newIDLocked("rpr"), name: name, profileID: profileID, arn: arn, resourceType: typ, properties: props, status: "CREATING", created: now, modified: now}
	st.resAssocs[ra.id] = ra
	return r53pConform("AssociateResourceToProfile", map[string]any{"ProfileResourceAssociation": ra.out()}), nil
}

func (st *r53pStore) getResAssocLocked(id string) (*r53pResAssoc, *r53pError) {
	ra := st.resAssocs[id]
	if ra == nil {
		return nil, r53pNotFound("PROFILE_RESOURCE_ASSOCIATION", "Profile resource association %s does not exist.", id)
	}
	return ra, nil
}

func (st *r53pStore) getResAssoc(id string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ra, err := st.getResAssocLocked(id)
	if err != nil {
		return nil, err
	}
	r53pSettle(&ra.status)
	return r53pConform("GetProfileResourceAssociation", map[string]any{"ProfileResourceAssociation": ra.out()}), nil
}

func (st *r53pStore) updateResAssoc(id string, name, props *string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ra, err := st.getResAssocLocked(id)
	if err != nil {
		return nil, err
	}
	if ra.status == "CREATING" || ra.status == "UPDATING" {
		return nil, r53pConflict("Profile resource association %s is %s; retry once it is COMPLETE.", id, ra.status)
	}
	if props != nil {
		if err := r53pCheckProperties(ra.resourceType, *props); err != nil {
			return nil, err
		}
		ra.properties = *props
	}
	if name != nil {
		ra.name = *name
	}
	ra.status, ra.modified = "UPDATING", time.Now().UTC()
	return r53pConform("UpdateProfileResourceAssociation", map[string]any{"ProfileResourceAssociation": ra.out()}), nil
}

func (st *r53pStore) disassociateResource(profileID, arn string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getProfileLocked(profileID); err != nil {
		return nil, err
	}
	for id, ra := range st.resAssocs {
		if ra.profileID == profileID && ra.arn == arn {
			delete(st.resAssocs, id)
			ra.status, ra.modified = "DELETING", time.Now().UTC()
			return r53pConform("DisassociateResourceFromProfile", map[string]any{"ProfileResourceAssociation": ra.out()}), nil
		}
	}
	return nil, r53pNotFound("PROFILE_RESOURCE_ASSOCIATION", "Resource %s is not associated with profile %s.", arn, profileID)
}

func (st *r53pStore) listResAssocs(profileID, typ, arn, nextToken string, maxResults int) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, err := st.getProfileLocked(profileID); err != nil {
		return nil, err
	}
	ids := []string{}
	for id, ra := range st.resAssocs {
		if ra.profileID == profileID && (typ == "" || ra.resourceType == typ) && (arn == "" || ra.arn == arn) {
			ids = append(ids, id)
		}
	}
	page, token, err := r53pPage(ids, nextToken, maxResults)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(page))
	for _, id := range page {
		ra := st.resAssocs[id]
		r53pSettle(&ra.status)
		items = append(items, ra.out())
	}
	out := map[string]any{"ProfileResourceAssociations": items}
	if token != "" {
		out["NextToken"] = token
	}
	return r53pConform("ListProfileResourceAssociations", out), nil
}

// taggableLocked reports whether arn names an existing profile or profile
// association, the resources this API tags.
func (st *r53pStore) taggableLocked(arn string) bool {
	if id, ok := strings.CutPrefix(arn, r53pProfileARN("")); ok {
		return st.profiles[id] != nil
	}
	if id, ok := strings.CutPrefix(arn, r53pAssocARN("")); ok {
		return st.assocs[id] != nil
	}
	return false
}

func (st *r53pStore) listTags(arn string) (map[string]any, *r53pError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.taggableLocked(arn) {
		return nil, r53pNotFound("", "Resource %s does not exist.", arn)
	}
	tags := map[string]any{}
	for k, v := range st.tags[arn] {
		tags[k] = v
	}
	return r53pConform("ListTagsForResource", map[string]any{"Tags": tags}), nil
}

func (st *r53pStore) tag(arn string, tags map[string]string) *r53pError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.taggableLocked(arn) {
		return r53pNotFound("", "Resource %s does not exist.", arn)
	}
	return st.setTagsLocked(arn, tags)
}

func (st *r53pStore) untag(arn string, keys []string) *r53pError {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.taggableLocked(arn) {
		return r53pNotFound("", "Resource %s does not exist.", arn)
	}
	cur := st.tags[arn]
	next := make(map[string]string, len(cur))
	for k, v := range cur {
		next[k] = v
	}
	for _, k := range keys {
		delete(next, k)
	}
	st.tags[arn] = next
	return nil
}
