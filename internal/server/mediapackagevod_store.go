package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	mpvodAccount = "123456789012"
	mpvodRegion  = "us-east-1"
	mpvodARNBase = "arn:aws:mediapackage-vod:" + mpvodRegion + ":" + mpvodAccount + ":"
	// ponytail: caps keep the store bounded; set above AWS quotas.
	mpvodMaxGroups    = 500
	mpvodMaxConfigs   = 2000
	mpvodMaxAssets    = 5000
	mpvodMaxTags      = 50
	mpvodMaxManifests = 20
	mpvodDefaultLogs  = "/aws/MediaPackage/VodEgressAccessLogs"
)

var mpvodIDPattern = regexp.MustCompile(`^[0-9a-zA-Z_-]{1,256}$`)

type mpvodErr struct {
	status    int
	code, msg string
}

func mpvodInvalid(format string, args ...any) *mpvodErr {
	return &mpvodErr{http.StatusUnprocessableEntity, "UnprocessableEntityException", fmt.Sprintf(format, args...)}
}

func mpvodNotFound(kind, id string) *mpvodErr {
	return &mpvodErr{http.StatusNotFound, "NotFoundException", fmt.Sprintf("%s %s does not exist", kind, id)}
}

type mpvodGroup struct {
	id, arn, domain string
	created         time.Time
	auth, logs      map[string]any // conformed copies; replaced, never mutated
	tags            map[string]string
}

type mpvodConfig struct {
	id, arn, group string
	created        time.Time
	kind           string         // cmafPackage, dashPackage, hlsPackage or mssPackage
	pkg            map[string]any // conformed copy
	tags           map[string]string
}

type mpvodAsset struct {
	id, arn, group, resourceID, sourceARN, roleARN string
	created                                        time.Time
	endpoints                                      []map[string]any
	settled                                        bool
	tags                                           map[string]string
}

type mpvodStore struct {
	mu      sync.Mutex
	groups  map[string]*mpvodGroup
	configs map[string]*mpvodConfig
	assets  map[string]*mpvodAsset
}

func newMediaPackageVODStore() *mpvodStore {
	return &mpvodStore{groups: map[string]*mpvodGroup{}, configs: map[string]*mpvodConfig{}, assets: map[string]*mpvodAsset{}}
}

func mpvodOut(op string, out map[string]any) map[string]any {
	m := awsmodels.MediaPackageVODModel()
	if shaped, ok := m.Conform(m.Operations[op].Output, out).(map[string]any); ok {
		return shaped
	}
	return map[string]any{}
}

func mpvodHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}

func mpvodTagsOut(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func mpvodCheckTags(tags map[string]string) *mpvodErr {
	if len(tags) > mpvodMaxTags {
		return mpvodInvalid("A resource can have at most %d tags", mpvodMaxTags)
	}
	for k := range tags {
		if k == "" || len(k) > 128 || strings.HasPrefix(strings.ToLower(k), "aws:") {
			return mpvodInvalid("Tag key %q is not valid", k)
		}
	}
	return nil
}

func mpvodCheckID(kind, id string) *mpvodErr {
	if !mpvodIDPattern.MatchString(id) {
		return mpvodInvalid("%s id %q must be 1 to 256 letters, digits, hyphens or underscores", kind, id)
	}
	return nil
}

func (g *mpvodGroup) out(assets int) map[string]any {
	m := map[string]any{
		"arn": g.arn, "id": g.id, "domainName": g.domain, "createdAt": g.created.Format(time.RFC3339),
		"tags": mpvodTagsOut(g.tags), "approximateAssetCount": assets,
	}
	if g.auth != nil {
		m["authorization"] = g.auth
	}
	if g.logs != nil {
		m["egressAccessLogs"] = g.logs
	}
	return m
}

func (c *mpvodConfig) out() map[string]any {
	return map[string]any{
		"arn": c.arn, "id": c.id, "packagingGroupId": c.group, "createdAt": c.created.Format(time.RFC3339),
		c.kind: c.pkg, "tags": mpvodTagsOut(c.tags),
	}
}

func (a *mpvodAsset) out(withEndpoints bool) map[string]any {
	m := map[string]any{
		"arn": a.arn, "id": a.id, "packagingGroupId": a.group, "resourceId": a.resourceID,
		"sourceArn": a.sourceARN, "sourceRoleArn": a.roleARN, "createdAt": a.created.Format(time.RFC3339),
		"tags": mpvodTagsOut(a.tags),
	}
	if withEndpoints {
		eps := make([]any, 0, len(a.endpoints))
		for _, e := range a.endpoints {
			eps = append(eps, e)
		}
		m["egressEndpoints"] = eps
	}
	return m
}

// settle makes a new asset's endpoints PLAYABLE: the create response reports
// QUEUED, every later read PLAYABLE. Endpoint maps are replaced, not mutated.
func (a *mpvodAsset) settle() {
	if a.settled {
		return
	}
	eps := make([]map[string]any, len(a.endpoints))
	for i, e := range a.endpoints {
		eps[i] = map[string]any{"packagingConfigurationId": e["packagingConfigurationId"], "url": e["url"], "status": "PLAYABLE"}
	}
	a.endpoints, a.settled = eps, true
}

func (st *mpvodStore) assetCount(group string) int {
	n := 0
	for _, a := range st.assets {
		if a.group == group {
			n++
		}
	}
	return n
}

// ---- packaging groups ----

func (st *mpvodStore) createGroup(id string, auth, logs map[string]any, tags map[string]string) (map[string]any, *mpvodErr) {
	if err := mpvodCheckID("PackagingGroup", id); err != nil {
		return nil, err
	}
	if err := mpvodCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.groups[id] != nil {
		return nil, mpvodInvalid("PackagingGroup %s already exists", id)
	}
	if len(st.groups) >= mpvodMaxGroups {
		return nil, mpvodInvalid("An account can have at most %d packaging groups", mpvodMaxGroups)
	}
	g := &mpvodGroup{
		id: id, arn: mpvodARNBase + "packaging-groups/" + id, created: time.Now().UTC(), auth: auth, logs: logs, tags: tags,
		domain: "https://" + mpvodHash("group", id)[:16] + ".egress.mediapackage-vod." + mpvodRegion + ".amazonaws.com",
	}
	st.groups[id] = g
	return mpvodOut("CreatePackagingGroup", g.out(0)), nil
}

func (st *mpvodStore) describeGroup(id string) (map[string]any, *mpvodErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g := st.groups[id]
	if g == nil {
		return nil, mpvodNotFound("PackagingGroup", id)
	}
	return mpvodOut("DescribePackagingGroup", g.out(st.assetCount(id))), nil
}

// updateGroup replaces the authorization (UpdatePackagingGroup) or the egress
// access logs (ConfigureLogs).
func (st *mpvodStore) updateGroup(op, id string, auth, logs map[string]any) (map[string]any, *mpvodErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	g := st.groups[id]
	if g == nil {
		return nil, mpvodNotFound("PackagingGroup", id)
	}
	next := *g
	if op == "ConfigureLogs" {
		next.logs = logs
	} else {
		next.auth = auth
	}
	st.groups[id] = &next
	return mpvodOut(op, next.out(st.assetCount(id))), nil
}

func (st *mpvodStore) deleteGroup(id string) *mpvodErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.groups[id] == nil {
		return mpvodNotFound("PackagingGroup", id)
	}
	if n := st.assetCount(id); n > 0 {
		return mpvodInvalid("PackagingGroup %s still has %d assets; delete them first", id, n)
	}
	for _, c := range st.configs {
		if c.group == id {
			return mpvodInvalid("PackagingGroup %s still has packaging configurations; delete them first", id)
		}
	}
	delete(st.groups, id)
	return nil
}

// ---- packaging configurations ----

func (st *mpvodStore) createConfig(id, group, kind string, pkg map[string]any, tags map[string]string) (map[string]any, *mpvodErr) {
	if err := mpvodCheckID("PackagingConfiguration", id); err != nil {
		return nil, err
	}
	if err := mpvodCheckTags(tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.groups[group] == nil {
		return nil, mpvodInvalid("PackagingGroup %s does not exist", group)
	}
	if st.configs[id] != nil {
		return nil, mpvodInvalid("PackagingConfiguration %s already exists", id)
	}
	if len(st.configs) >= mpvodMaxConfigs {
		return nil, mpvodInvalid("An account can have at most %d packaging configurations", mpvodMaxConfigs)
	}
	c := &mpvodConfig{id: id, arn: mpvodARNBase + "packaging-configurations/" + id, group: group, created: time.Now().UTC(), kind: kind, pkg: pkg, tags: tags}
	st.configs[id] = c
	return mpvodOut("CreatePackagingConfiguration", c.out()), nil
}

func (st *mpvodStore) describeConfig(id string) (map[string]any, *mpvodErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c := st.configs[id]
	if c == nil {
		return nil, mpvodNotFound("PackagingConfiguration", id)
	}
	return mpvodOut("DescribePackagingConfiguration", c.out()), nil
}

func (st *mpvodStore) deleteConfig(id string) *mpvodErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.configs[id] == nil {
		return mpvodNotFound("PackagingConfiguration", id)
	}
	delete(st.configs, id)
	return nil
}

// ---- assets ----

type mpvodAssetInput struct {
	id, group, resourceID, sourceARN, roleARN string
	tags                                      map[string]string
}

func (st *mpvodStore) createAsset(in mpvodAssetInput) (map[string]any, *mpvodErr) {
	if err := mpvodCheckID("Asset", in.id); err != nil {
		return nil, err
	}
	if err := mpvodCheckTags(in.tags); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	g := st.groups[in.group]
	if g == nil {
		return nil, mpvodInvalid("PackagingGroup %s does not exist", in.group)
	}
	if st.assets[in.id] != nil {
		return nil, mpvodInvalid("Asset %s already exists", in.id)
	}
	if len(st.assets) >= mpvodMaxAssets {
		return nil, mpvodInvalid("An account can have at most %d assets", mpvodMaxAssets)
	}
	if in.resourceID == "" {
		in.resourceID = in.id
	}
	a := &mpvodAsset{
		id: in.id, arn: mpvodARNBase + "assets/" + in.id, group: in.group, resourceID: in.resourceID,
		sourceARN: in.sourceARN, roleARN: in.roleARN, created: time.Now().UTC(), tags: in.tags,
	}
	// One egress endpoint per packaging configuration in the group.
	var configs []*mpvodConfig
	for _, c := range st.configs {
		if c.group == in.group {
			configs = append(configs, c)
		}
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].id < configs[j].id })
	for _, c := range configs {
		url := g.domain + "/out/v1/" + mpvodHash("asset", a.id) + "/" + mpvodHash("config", c.id) + "/" + mpvodManifestFile(c)
		a.endpoints = append(a.endpoints, map[string]any{"packagingConfigurationId": c.id, "url": url, "status": "QUEUED"})
	}
	st.assets[in.id] = a
	return mpvodOut("CreateAsset", a.out(true)), nil
}

// mpvodManifestFile is the first manifest's file name for a configuration.
func mpvodManifestFile(c *mpvodConfig) string {
	key, ext := "hlsManifests", ".m3u8"
	switch c.kind {
	case "dashPackage":
		key, ext = "dashManifests", ".mpd"
	case "mssPackage":
		key, ext = "mssManifests", ".ism/Manifest"
	}
	name := "index"
	if list, _ := c.pkg[key].([]any); len(list) > 0 {
		if m, _ := list[0].(map[string]any); m != nil {
			if n, _ := m["manifestName"].(string); n != "" {
				name = n
			}
		}
	}
	return name + ext
}

func (st *mpvodStore) describeAsset(id string) (map[string]any, *mpvodErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	a := st.assets[id]
	if a == nil {
		return nil, mpvodNotFound("Asset", id)
	}
	a.settle()
	return mpvodOut("DescribeAsset", a.out(true)), nil
}

func (st *mpvodStore) deleteAsset(id string) *mpvodErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.assets[id] == nil {
		return mpvodNotFound("Asset", id)
	}
	delete(st.assets, id)
	return nil
}

// ---- lists ----

// list pages one resource kind, optionally filtered by packaging group.
func (st *mpvodStore) list(op, group, token string, max int) (map[string]any, *mpvodErr) {
	offset := 0
	if token != "" {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		n, perr := strconv.ParseInt(string(raw), 10, 32)
		if err != nil || perr != nil || n < 0 {
			return nil, mpvodInvalid("nextToken is not valid")
		}
		offset = int(n)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	type row struct {
		id   string
		item map[string]any
	}
	var rows []row
	key := ""
	switch op {
	case "ListPackagingGroups":
		key = "packagingGroups"
		for id, g := range st.groups {
			rows = append(rows, row{id, g.out(st.assetCount(id))})
		}
	case "ListPackagingConfigurations":
		key = "packagingConfigurations"
		for id, c := range st.configs {
			if group == "" || c.group == group {
				rows = append(rows, row{id, c.out()})
			}
		}
	default:
		key = "assets"
		for id, a := range st.assets {
			if group == "" || a.group == group {
				rows = append(rows, row{id, a.out(false)})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	offset = min(offset, len(rows))
	end := min(offset+max, len(rows))
	items := make([]any, 0, end-offset)
	for _, r := range rows[offset:end] {
		items = append(items, r.item)
	}
	out := map[string]any{key: items}
	if end < len(rows) {
		out["nextToken"] = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	return mpvodOut(op, out), nil
}

// ---- tags ----

// tagsOf resolves an ARN to its tag map's owner. Callers hold st.mu.
func (st *mpvodStore) tagsOf(arn string) (*map[string]string, *mpvodErr) {
	rest, ok := strings.CutPrefix(arn, mpvodARNBase)
	kind, id, ok2 := strings.Cut(rest, "/")
	if ok && ok2 {
		switch kind {
		case "packaging-groups":
			if g := st.groups[id]; g != nil {
				return &g.tags, nil
			}
		case "packaging-configurations":
			if c := st.configs[id]; c != nil {
				return &c.tags, nil
			}
		case "assets":
			if a := st.assets[id]; a != nil {
				return &a.tags, nil
			}
		}
	}
	return nil, mpvodNotFound("Resource", arn)
}

func (st *mpvodStore) listTags(arn string) (map[string]any, *mpvodErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagsOf(arn)
	if err != nil {
		return nil, err
	}
	return mpvodOut("ListTagsForResource", map[string]any{"tags": mpvodTagsOut(*tags)}), nil
}

// retag replaces the resource's tag map with a new one (copy-on-write).
func (st *mpvodStore) retag(arn string, add map[string]string, remove []string) *mpvodErr {
	st.mu.Lock()
	defer st.mu.Unlock()
	tags, err := st.tagsOf(arn)
	if err != nil {
		return err
	}
	next := make(map[string]string, len(*tags)+len(add))
	for k, v := range *tags {
		next[k] = v
	}
	for k, v := range add {
		next[k] = v
	}
	for _, k := range remove {
		delete(next, k)
	}
	if err := mpvodCheckTags(next); err != nil {
		return err
	}
	*tags = next
	return nil
}
