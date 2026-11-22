package server

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	cxnAccountID = "123456789012"
	cxnRegion    = "us-east-1"
	// ponytail: per-kind caps well above AWS quotas (250 connections, 50
	// hosts) so repeated coverage runs against one server don't hit them.
	cxnMaxResources      = 1000
	cxnMaxTags           = 200
	cxnMaxBlockersPerCfg = 20
	cxnDefaultPageSize   = 100
)

type cxnError struct {
	status    int
	code, msg string
}

func cxnErr(status int, code, format string, args ...any) *cxnError {
	return &cxnError{status: status, code: code, msg: fmt.Sprintf(format, args...)}
}

func cxnNotFound(format string, args ...any) *cxnError {
	return cxnErr(http.StatusBadRequest, "ResourceNotFoundException", format, args...)
}

func cxnInvalid(format string, args ...any) *cxnError {
	return cxnErr(http.StatusBadRequest, "InvalidInputException", format, args...)
}

func cxnValidation(format string, args ...any) *cxnError {
	return cxnErr(http.StatusBadRequest, "ValidationException", format, args...)
}

type cxnTag struct{ Key, Value string }

type cxnConnection struct {
	id, name, providerType, hostID, status string
	seq                                    int
}

type cxnHost struct {
	id, name, providerType, endpoint, status string
	vpc                                      map[string]any
	seq                                      int
}

type cxnLink struct {
	id, connectionID, ownerID, repo, providerType, kmsKey string
	seq                                                   int
}

type cxnBlocker struct {
	id, reason, resolvedReason string
	created, resolved          time.Time
}

type cxnSync struct {
	syncType, resourceName, linkID, branch, configFile, roleARN string
	publish, trigger, prComment                                 string
	updated                                                     time.Time
	blockers                                                    []*cxnBlocker
	seq                                                         int
}

type codeConnectionsStore struct {
	mu          sync.Mutex
	seq         int
	connections map[string]*cxnConnection
	hosts       map[string]*cxnHost
	links       map[string]*cxnLink
	syncs       map[string]*cxnSync // syncType + "/" + resourceName
	tags        map[string][]cxnTag // kind + "/" + id
}

func newCodeConnectionsStore() *codeConnectionsStore {
	return &codeConnectionsStore{
		connections: map[string]*cxnConnection{},
		hosts:       map[string]*cxnHost{},
		links:       map[string]*cxnLink{},
		syncs:       map[string]*cxnSync{},
		tags:        map[string][]cxnTag{},
	}
}

func (st *codeConnectionsStore) nextSeq() int { st.seq++; return st.seq }

func (api *cxnAPI) arn(kind, id string) string {
	return "arn:aws:" + api.service + ":" + cxnRegion + ":" + cxnAccountID + ":" + kind + "/" + id
}

// cxnParseARN accepts either service prefix and returns kind and id.
func cxnParseARN(arn string) (kind, id string, ok bool) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 || parts[0] != "arn" || (parts[2] != "codeconnections" && parts[2] != "codestar-connections") || parts[4] != cxnAccountID {
		return "", "", false
	}
	kind, id, ok = strings.Cut(parts[5], "/")
	return kind, id, ok && id != "" && !strings.Contains(id, "/")
}

func cxnStr(in map[string]any, key string) string {
	s, _ := in[key].(string)
	return s
}

func cxnTagsIn(v any) []cxnTag {
	items, _ := v.([]any)
	out := make([]cxnTag, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		out = append(out, cxnTag{cxnStr(m, "Key"), cxnStr(m, "Value")})
	}
	return out
}

func cxnTagsOut(tags []cxnTag) []any {
	out := make([]any, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]any{"Key": t.Key, "Value": t.Value})
	}
	return out
}

// mergeTagsLocked applies tags to a resource, replacing values of existing
// keys, within the per-resource cap.
func (st *codeConnectionsStore) mergeTagsLocked(key string, add []cxnTag) *cxnError {
	merged := append([]cxnTag(nil), st.tags[key]...)
	for _, t := range add {
		if strings.HasPrefix(strings.ToLower(t.Key), "aws:") {
			return cxnValidation("tag keys may not start with aws:")
		}
		replaced := false
		for i := range merged {
			if merged[i].Key == t.Key {
				merged[i].Value, replaced = t.Value, true
			}
		}
		if !replaced {
			merged = append(merged, t)
		}
	}
	if len(merged) > cxnMaxTags {
		return cxnErr(http.StatusBadRequest, "LimitExceededException", "a resource can have at most %d tags", cxnMaxTags)
	}
	st.tags[key] = merged
	return nil
}

// cxnPage slices sorted items for MaxResults/NextToken (an offset).
func cxnPage[T any](items []T, in map[string]any) ([]T, string, *cxnError) {
	size := cxnDefaultPageSize
	if n, ok := in["MaxResults"].(float64); ok && n > 0 {
		size = int(n) // the model bounds MaxResults to 0..100
	}
	start := 0
	if tok := cxnStr(in, "NextToken"); tok != "" {
		n, err := strconv.Atoi(tok)
		if err != nil || n < 0 || n > len(items) {
			return nil, "", cxnInvalid("invalid NextToken")
		}
		start = n
	}
	end := min(start+size, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

func cxnRandHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func cxnEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

var cxnSelfManaged = map[string]bool{"GitHubEnterpriseServer": true, "GitLabSelfManaged": true}

// handle runs op and shapes its result to the API's output shape while still
// holding the lock, so the response never aliases stored state.
func (st *codeConnectionsStore) handle(api *cxnAPI, op string, in map[string]any) (any, *cxnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := st.dispatchLocked(api, op, in)
	if err != nil {
		return nil, err
	}
	if out["NextToken"] == "" {
		delete(out, "NextToken") // last page: AWS omits the token
	}
	return api.model().Conform(api.model().Operations[op].Output, out), nil
}

func (st *codeConnectionsStore) dispatchLocked(api *cxnAPI, op string, in map[string]any) (map[string]any, *cxnError) {
	switch op {
	case "CreateConnection":
		return st.createConnection(api, in)
	case "GetConnection", "DeleteConnection":
		c, err := st.connectionByARN(cxnStr(in, "ConnectionArn"))
		if err != nil {
			return nil, err
		}
		if op == "DeleteConnection" {
			delete(st.connections, c.id)
			delete(st.tags, "connection/"+c.id)
			return map[string]any{}, nil
		}
		return map[string]any{"Connection": st.connectionOut(api, c)}, nil
	case "ListConnections":
		return st.listConnections(api, in)
	case "CreateHost":
		return st.createHost(api, in)
	case "GetHost":
		h, err := st.hostByARN(cxnStr(in, "HostArn"))
		if err != nil {
			return nil, err
		}
		out := st.hostOut(api, h)
		delete(out, "HostArn")
		delete(out, "StatusMessage")
		return out, nil
	case "DeleteHost":
		h, err := st.hostByARN(cxnStr(in, "HostArn"))
		if err != nil {
			return nil, err
		}
		for _, c := range st.connections {
			if c.hostID == h.id {
				return nil, cxnErr(http.StatusBadRequest, "ResourceUnavailableException", "host %s still has connections; delete them first", h.id)
			}
		}
		delete(st.hosts, h.id)
		delete(st.tags, "host/"+h.id)
		return map[string]any{}, nil
	case "UpdateHost":
		h, err := st.hostByARN(cxnStr(in, "HostArn"))
		if err != nil {
			return nil, err
		}
		if v := cxnStr(in, "ProviderEndpoint"); v != "" {
			h.endpoint = v
		}
		if vpc, ok := in["VpcConfiguration"].(map[string]any); ok {
			h.vpc = vpc
		}
		return map[string]any{}, nil
	case "ListHosts":
		hosts := make([]*cxnHost, 0, len(st.hosts))
		for _, h := range st.hosts {
			hosts = append(hosts, h)
		}
		sort.Slice(hosts, func(i, j int) bool { return hosts[i].seq < hosts[j].seq })
		page, next, err := cxnPage(hosts, in)
		if err != nil {
			return nil, err
		}
		out := make([]any, 0, len(page))
		for _, h := range page {
			out = append(out, st.hostOut(api, h))
		}
		return map[string]any{"Hosts": out, "NextToken": next}, nil
	case "CreateRepositoryLink":
		return st.createRepositoryLink(api, in)
	case "GetRepositoryLink", "DeleteRepositoryLink", "UpdateRepositoryLink":
		return st.repositoryLinkOp(api, op, in)
	case "ListRepositoryLinks":
		links := st.sortedLinks()
		page, next, err := cxnPage(links, in)
		if err != nil {
			return nil, err
		}
		out := make([]any, 0, len(page))
		for _, l := range page {
			out = append(out, st.linkOut(api, l))
		}
		return map[string]any{"RepositoryLinks": out, "NextToken": next}, nil
	case "CreateSyncConfiguration":
		return st.createSyncConfiguration(in)
	case "GetSyncConfiguration", "UpdateSyncConfiguration", "DeleteSyncConfiguration":
		return st.syncConfigurationOp(op, in)
	case "ListSyncConfigurations", "ListRepositorySyncDefinitions":
		return st.listSyncs(op, in)
	case "GetRepositorySyncStatus":
		return st.repositorySyncStatus(in)
	case "GetResourceSyncStatus":
		return st.resourceSyncStatus(in)
	case "GetSyncBlockerSummary", "UpdateSyncBlocker":
		return st.syncBlockerOp(op, in)
	case "TagResource", "UntagResource", "ListTagsForResource":
		return st.tagOp(op, in)
	}
	return nil, cxnErr(http.StatusBadRequest, "UnknownOperationException", "unknown operation %s", op)
}

// --- connections and hosts ---

func (st *codeConnectionsStore) createConnection(api *cxnAPI, in map[string]any) (map[string]any, *cxnError) {
	provider, hostARN := cxnStr(in, "ProviderType"), cxnStr(in, "HostArn")
	c := &cxnConnection{id: uuid.NewString(), name: cxnStr(in, "ConnectionName"), providerType: provider, status: "PENDING"}
	switch {
	case hostARN != "":
		h, err := st.hostByARN(hostARN)
		if err != nil {
			return nil, err
		}
		if h.status != "AVAILABLE" {
			return nil, cxnErr(http.StatusBadRequest, "ResourceUnavailableException", "host %s is %s, not AVAILABLE", h.id, h.status)
		}
		if provider != "" && provider != h.providerType {
			return nil, cxnValidation("ProviderType %s does not match the host's %s", provider, h.providerType)
		}
		c.hostID, c.providerType = h.id, h.providerType
	case provider == "":
		return nil, cxnValidation("one of ProviderType or HostArn is required")
	case cxnSelfManaged[provider]:
		return nil, cxnValidation("a %s connection needs a HostArn", provider)
	}
	if len(st.connections) >= cxnMaxResources {
		return nil, cxnErr(http.StatusBadRequest, "LimitExceededException", "connection limit of %d reached", cxnMaxResources)
	}
	tags := cxnTagsIn(in["Tags"])
	if err := st.mergeTagsLocked("connection/"+c.id, tags); err != nil {
		return nil, err
	}
	c.seq = st.nextSeq()
	st.connections[c.id] = c
	return map[string]any{"ConnectionArn": api.arn("connection", c.id), "Tags": cxnTagsOut(st.tags["connection/"+c.id])}, nil
}

func (st *codeConnectionsStore) connectionByARN(arn string) (*cxnConnection, *cxnError) {
	kind, id, ok := cxnParseARN(arn)
	c := st.connections[id]
	if !ok || kind != "connection" || c == nil {
		return nil, cxnNotFound("connection %s not found", arn)
	}
	return c, nil
}

func (st *codeConnectionsStore) connectionOut(api *cxnAPI, c *cxnConnection) map[string]any {
	out := map[string]any{
		"ConnectionName": c.name, "ConnectionArn": api.arn("connection", c.id), "ProviderType": c.providerType,
		"OwnerAccountId": cxnAccountID, "ConnectionStatus": c.status,
	}
	if c.hostID != "" {
		out["HostArn"] = api.arn("host", c.hostID)
	}
	return out
}

func (st *codeConnectionsStore) listConnections(api *cxnAPI, in map[string]any) (map[string]any, *cxnError) {
	hostFilter := ""
	if arn := cxnStr(in, "HostArnFilter"); arn != "" {
		h, err := st.hostByARN(arn)
		if err != nil {
			return nil, err
		}
		hostFilter = h.id
	}
	provider := cxnStr(in, "ProviderTypeFilter")
	conns := make([]*cxnConnection, 0, len(st.connections))
	for _, c := range st.connections {
		if (provider == "" || c.providerType == provider) && (hostFilter == "" || c.hostID == hostFilter) {
			conns = append(conns, c)
		}
	}
	sort.Slice(conns, func(i, j int) bool { return conns[i].seq < conns[j].seq })
	page, next, err := cxnPage(conns, in)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(page))
	for _, c := range page {
		out = append(out, st.connectionOut(api, c))
	}
	return map[string]any{"Connections": out, "NextToken": next}, nil
}

func (st *codeConnectionsStore) createHost(api *cxnAPI, in map[string]any) (map[string]any, *cxnError) {
	provider := cxnStr(in, "ProviderType")
	if !cxnSelfManaged[provider] {
		return nil, cxnValidation("hosts are only for GitHubEnterpriseServer or GitLabSelfManaged, not %s", provider)
	}
	if len(st.hosts) >= cxnMaxResources {
		return nil, cxnErr(http.StatusBadRequest, "LimitExceededException", "host limit of %d reached", cxnMaxResources)
	}
	h := &cxnHost{id: cxnHostIDPrefix(cxnStr(in, "Name")) + "-" + cxnRandHex(4), name: cxnStr(in, "Name"), providerType: provider, endpoint: cxnStr(in, "ProviderEndpoint"), status: "PENDING"}
	if vpc, ok := in["VpcConfiguration"].(map[string]any); ok {
		h.vpc = vpc
	}
	if err := st.mergeTagsLocked("host/"+h.id, cxnTagsIn(in["Tags"])); err != nil {
		return nil, err
	}
	h.seq = st.nextSeq()
	st.hosts[h.id] = h
	return map[string]any{"HostArn": api.arn("host", h.id), "Tags": cxnTagsOut(st.tags["host/"+h.id])}, nil
}

// cxnHostIDPrefix keeps a host ID ARN-safe: the name may hold any character.
func cxnHostIDPrefix(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)
}

func (st *codeConnectionsStore) hostByARN(arn string) (*cxnHost, *cxnError) {
	kind, id, ok := cxnParseARN(arn)
	h := st.hosts[id]
	if !ok || kind != "host" || h == nil {
		return nil, cxnNotFound("host %s not found", arn)
	}
	return h, nil
}

func (st *codeConnectionsStore) hostOut(api *cxnAPI, h *cxnHost) map[string]any {
	out := map[string]any{
		"Name": h.name, "HostArn": api.arn("host", h.id), "ProviderType": h.providerType,
		"ProviderEndpoint": h.endpoint, "Status": h.status,
	}
	if h.vpc != nil {
		out["VpcConfiguration"] = h.vpc // Conform copies it before encoding
	}
	return out
}

func (st *codeConnectionsStore) activateConnection(api *cxnAPI, id string) (map[string]any, *cxnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, arnID, ok := cxnParseARN(id); ok {
		id = arnID
	}
	c := st.connections[id]
	if c == nil {
		return nil, cxnNotFound("connection %s not found", id)
	}
	c.status = "AVAILABLE"
	return st.connectionOut(api, c), nil
}

func (st *codeConnectionsStore) activateHost(api *cxnAPI, id string) (map[string]any, *cxnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, arnID, ok := cxnParseARN(id); ok {
		id = arnID
	}
	h := st.hosts[id]
	if h == nil {
		return nil, cxnNotFound("host %s not found", id)
	}
	h.status = "AVAILABLE"
	out := st.hostOut(api, h)
	delete(out, "VpcConfiguration")
	return out, nil
}

// --- repository links ---

func (st *codeConnectionsStore) availableConnection(arn string) (*cxnConnection, *cxnError) {
	kind, id, ok := cxnParseARN(arn)
	c := st.connections[id]
	if !ok || kind != "connection" || c == nil {
		return nil, cxnInvalid("connection %s not found", arn)
	}
	if c.status != "AVAILABLE" {
		return nil, cxnInvalid("connection %s is %s, not AVAILABLE", arn, c.status)
	}
	return c, nil
}

func (st *codeConnectionsStore) createRepositoryLink(api *cxnAPI, in map[string]any) (map[string]any, *cxnError) {
	c, err := st.availableConnection(cxnStr(in, "ConnectionArn"))
	if err != nil {
		return nil, err
	}
	owner, repo := cxnStr(in, "OwnerId"), cxnStr(in, "RepositoryName")
	for _, l := range st.links {
		if l.connectionID == c.id && l.ownerID == owner && l.repo == repo {
			return nil, cxnErr(http.StatusBadRequest, "ResourceAlreadyExistsException", "repository %s/%s is already linked through this connection", owner, repo)
		}
	}
	if len(st.links) >= cxnMaxResources {
		return nil, cxnErr(http.StatusBadRequest, "LimitExceededException", "repository link limit of %d reached", cxnMaxResources)
	}
	l := &cxnLink{id: uuid.NewString(), connectionID: c.id, ownerID: owner, repo: repo, providerType: c.providerType, kmsKey: cxnStr(in, "EncryptionKeyArn")}
	if err := st.mergeTagsLocked("repository-link/"+l.id, cxnTagsIn(in["Tags"])); err != nil {
		return nil, err
	}
	l.seq = st.nextSeq()
	st.links[l.id] = l
	return map[string]any{"RepositoryLinkInfo": st.linkOut(api, l)}, nil
}

func (st *codeConnectionsStore) linkOut(api *cxnAPI, l *cxnLink) map[string]any {
	out := map[string]any{
		"ConnectionArn": api.arn("connection", l.connectionID), "OwnerId": l.ownerID, "ProviderType": l.providerType,
		"RepositoryLinkArn": api.arn("repository-link", l.id), "RepositoryLinkId": l.id, "RepositoryName": l.repo,
	}
	if l.kmsKey != "" {
		out["EncryptionKeyArn"] = l.kmsKey
	}
	return out
}

func (st *codeConnectionsStore) sortedLinks() []*cxnLink {
	links := make([]*cxnLink, 0, len(st.links))
	for _, l := range st.links {
		links = append(links, l)
	}
	sort.Slice(links, func(i, j int) bool { return links[i].seq < links[j].seq })
	return links
}

func (st *codeConnectionsStore) linkByID(id string) (*cxnLink, *cxnError) {
	l := st.links[id]
	if l == nil {
		return nil, cxnNotFound("repository link %s not found", id)
	}
	return l, nil
}

func (st *codeConnectionsStore) repositoryLinkOp(api *cxnAPI, op string, in map[string]any) (map[string]any, *cxnError) {
	l, err := st.linkByID(cxnStr(in, "RepositoryLinkId"))
	if err != nil {
		return nil, err
	}
	switch op {
	case "DeleteRepositoryLink":
		for _, s := range st.syncs {
			if s.linkID == l.id {
				return nil, cxnErr(http.StatusBadRequest, "SyncConfigurationStillExistsException", "repository link %s still has sync configuration %s", l.id, s.resourceName)
			}
		}
		delete(st.links, l.id)
		delete(st.tags, "repository-link/"+l.id)
		return map[string]any{}, nil
	case "UpdateRepositoryLink":
		if arn := cxnStr(in, "ConnectionArn"); arn != "" {
			c, err := st.availableConnection(arn)
			if err != nil {
				return nil, err
			}
			if c.providerType != l.providerType {
				return nil, cxnInvalid("connection %s is %s, the repository is %s", arn, c.providerType, l.providerType)
			}
			l.connectionID = c.id
		}
		if key := cxnStr(in, "EncryptionKeyArn"); key != "" {
			l.kmsKey = key
		}
	}
	return map[string]any{"RepositoryLinkInfo": st.linkOut(api, l)}, nil
}

// --- sync configurations ---

func cxnSyncKey(syncType, resource string) string { return syncType + "/" + resource }

func (st *codeConnectionsStore) syncByKey(syncType, resource string) (*cxnSync, *cxnError) {
	s := st.syncs[cxnSyncKey(syncType, resource)]
	if s == nil {
		return nil, cxnNotFound("no %s sync configuration for resource %s", syncType, resource)
	}
	return s, nil
}

func (st *codeConnectionsStore) syncOut(s *cxnSync) map[string]any {
	l := st.links[s.linkID]
	out := map[string]any{
		"Branch": s.branch, "ConfigFile": s.configFile, "RepositoryLinkId": s.linkID, "ResourceName": s.resourceName,
		"RoleArn": s.roleARN, "SyncType": s.syncType,
	}
	if l != nil {
		out["OwnerId"], out["ProviderType"], out["RepositoryName"] = l.ownerID, l.providerType, l.repo
	}
	for key, v := range map[string]string{"PublishDeploymentStatus": s.publish, "TriggerResourceUpdateOn": s.trigger, "PullRequestComment": s.prComment} {
		if v != "" {
			out[key] = v
		}
	}
	return out
}

func (st *codeConnectionsStore) createSyncConfiguration(in map[string]any) (map[string]any, *cxnError) {
	key := cxnSyncKey(cxnStr(in, "SyncType"), cxnStr(in, "ResourceName"))
	if st.syncs[key] != nil {
		return nil, cxnErr(http.StatusBadRequest, "ResourceAlreadyExistsException", "resource %s already has a %s sync configuration", cxnStr(in, "ResourceName"), cxnStr(in, "SyncType"))
	}
	if st.links[cxnStr(in, "RepositoryLinkId")] == nil {
		return nil, cxnInvalid("repository link %s not found", cxnStr(in, "RepositoryLinkId"))
	}
	if len(st.syncs) >= cxnMaxResources {
		return nil, cxnErr(http.StatusBadRequest, "LimitExceededException", "sync configuration limit of %d reached", cxnMaxResources)
	}
	s := &cxnSync{
		syncType: cxnStr(in, "SyncType"), resourceName: cxnStr(in, "ResourceName"), linkID: cxnStr(in, "RepositoryLinkId"),
		branch: cxnStr(in, "Branch"), configFile: cxnStr(in, "ConfigFile"), roleARN: cxnStr(in, "RoleArn"),
		publish: cxnStr(in, "PublishDeploymentStatus"), trigger: cxnStr(in, "TriggerResourceUpdateOn"), prComment: cxnStr(in, "PullRequestComment"),
		updated: time.Now().UTC(), seq: st.nextSeq(),
	}
	st.syncs[key] = s
	return map[string]any{"SyncConfiguration": st.syncOut(s)}, nil
}

func (st *codeConnectionsStore) syncConfigurationOp(op string, in map[string]any) (map[string]any, *cxnError) {
	syncType, resource := cxnStr(in, "SyncType"), cxnStr(in, "ResourceName")
	s := st.syncs[cxnSyncKey(syncType, resource)]
	if s == nil {
		if op == "DeleteSyncConfiguration" { // the model lists no ResourceNotFoundException here
			return nil, cxnInvalid("no %s sync configuration for resource %s", syncType, resource)
		}
		return nil, cxnNotFound("no %s sync configuration for resource %s", syncType, resource)
	}
	switch op {
	case "DeleteSyncConfiguration":
		delete(st.syncs, cxnSyncKey(syncType, resource))
		return map[string]any{}, nil
	case "UpdateSyncConfiguration":
		if id := cxnStr(in, "RepositoryLinkId"); id != "" {
			if st.links[id] == nil {
				return nil, cxnNotFound("repository link %s not found", id)
			}
			s.linkID = id
		}
		for key, field := range map[string]*string{
			"Branch": &s.branch, "ConfigFile": &s.configFile, "RoleArn": &s.roleARN, "PublishDeploymentStatus": &s.publish,
			"TriggerResourceUpdateOn": &s.trigger, "PullRequestComment": &s.prComment,
		} {
			if v := cxnStr(in, key); v != "" {
				*field = v
			}
		}
		s.updated = time.Now().UTC()
	}
	return map[string]any{"SyncConfiguration": st.syncOut(s)}, nil
}

func (st *codeConnectionsStore) listSyncs(op string, in map[string]any) (map[string]any, *cxnError) {
	l, err := st.linkByID(cxnStr(in, "RepositoryLinkId"))
	if err != nil {
		return nil, err
	}
	syncs := []*cxnSync{}
	for _, s := range st.syncs {
		if s.linkID == l.id && s.syncType == cxnStr(in, "SyncType") {
			syncs = append(syncs, s)
		}
	}
	sort.Slice(syncs, func(i, j int) bool { return syncs[i].seq < syncs[j].seq })
	page, next, perr := cxnPage(syncs, in)
	if perr != nil {
		return nil, perr
	}
	out := make([]any, 0, len(page))
	for _, s := range page {
		if op == "ListRepositorySyncDefinitions" {
			// ponytail: one definition per sync configuration, read from the
			// config rather than from the repository's deployment files.
			out = append(out, map[string]any{"Branch": s.branch, "Directory": path.Dir(s.configFile), "Parent": s.resourceName, "Target": s.resourceName})
		} else {
			out = append(out, st.syncOut(s))
		}
	}
	if op == "ListRepositorySyncDefinitions" {
		return map[string]any{"RepositorySyncDefinitions": out, "NextToken": next}, nil
	}
	return map[string]any{"SyncConfigurations": out, "NextToken": next}, nil
}

// --- sync status and blockers ---

// cxnSha is a stable stand-in commit for a sync configuration's current state.
func cxnSha(s *cxnSync) string {
	sum := sha1.Sum([]byte(s.linkID + "\x00" + s.branch + "\x00" + s.configFile + "\x00" + s.updated.String()))
	return hex.EncodeToString(sum[:])
}

func cxnSyncEvents(s *cxnSync) []any {
	t := cxnEpoch(s.updated)
	return []any{
		map[string]any{"Event": "Sync initiated for branch " + s.branch, "Time": t, "Type": "INFO"},
		map[string]any{"Event": "Sync succeeded", "Time": t, "Type": "INFO"},
	}
}

func (st *codeConnectionsStore) repositorySyncStatus(in map[string]any) (map[string]any, *cxnError) {
	l, err := st.linkByID(cxnStr(in, "RepositoryLinkId"))
	if err != nil {
		return nil, err
	}
	var latest *cxnSync
	for _, s := range st.syncs {
		if s.linkID == l.id && s.branch == cxnStr(in, "Branch") && s.syncType == cxnStr(in, "SyncType") && (latest == nil || s.updated.After(latest.updated)) {
			latest = s
		}
	}
	if latest == nil {
		return nil, cxnNotFound("no sync has run for branch %s of repository link %s", cxnStr(in, "Branch"), l.id)
	}
	return map[string]any{"LatestSync": map[string]any{"StartedAt": cxnEpoch(latest.updated), "Status": "SUCCEEDED", "Events": cxnSyncEvents(latest)}}, nil
}

func (st *codeConnectionsStore) resourceSyncStatus(in map[string]any) (map[string]any, *cxnError) {
	s, err := st.syncByKey(cxnStr(in, "SyncType"), cxnStr(in, "ResourceName"))
	if err != nil {
		return nil, err
	}
	l := st.links[s.linkID]
	revision := map[string]any{"Branch": s.branch, "Directory": s.configFile, "Sha": cxnSha(s)}
	if l != nil {
		revision["OwnerId"], revision["RepositoryName"], revision["ProviderType"] = l.ownerID, l.repo, l.providerType
	}
	attempt := func() map[string]any {
		return map[string]any{
			"Events": cxnSyncEvents(s), "InitialRevision": revision, "StartedAt": cxnEpoch(s.updated),
			"Status": "SUCCEEDED", "TargetRevision": revision, "Target": s.resourceName,
		}
	}
	return map[string]any{"DesiredState": revision, "LatestSync": attempt(), "LatestSuccessfulSync": attempt()}, nil
}

func cxnBlockerOut(b *cxnBlocker) map[string]any {
	out := map[string]any{"Id": b.id, "Type": "AUTOMATED", "Status": "ACTIVE", "CreatedReason": b.reason, "CreatedAt": cxnEpoch(b.created)}
	if !b.resolved.IsZero() {
		out["Status"], out["ResolvedReason"], out["ResolvedAt"] = "RESOLVED", b.resolvedReason, cxnEpoch(b.resolved)
	}
	return out
}

func (st *codeConnectionsStore) syncBlockerOp(op string, in map[string]any) (map[string]any, *cxnError) {
	s, err := st.syncByKey(cxnStr(in, "SyncType"), cxnStr(in, "ResourceName"))
	if err != nil {
		return nil, err
	}
	if op == "GetSyncBlockerSummary" {
		blockers := make([]any, 0, len(s.blockers))
		for _, b := range s.blockers {
			blockers = append(blockers, cxnBlockerOut(b))
		}
		return map[string]any{"SyncBlockerSummary": map[string]any{"ResourceName": s.resourceName, "LatestBlockers": blockers}}, nil
	}
	for _, b := range s.blockers {
		if b.id != cxnStr(in, "Id") {
			continue
		}
		if !b.resolved.IsZero() {
			return nil, cxnInvalid("sync blocker %s is already resolved", b.id)
		}
		b.resolved, b.resolvedReason = time.Now().UTC(), cxnStr(in, "ResolvedReason")
		return map[string]any{"ResourceName": s.resourceName, "SyncBlocker": cxnBlockerOut(b)}, nil
	}
	return nil, cxnErr(http.StatusBadRequest, "SyncBlockerDoesNotExistException", "sync blocker %s does not exist", cxnStr(in, "Id"))
}

func (st *codeConnectionsStore) addSyncBlocker(syncType, resource, reason string) (map[string]any, *cxnError) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, err := st.syncByKey(syncType, resource)
	if err != nil {
		return nil, err
	}
	if len(s.blockers) >= cxnMaxBlockersPerCfg {
		return nil, cxnErr(http.StatusBadRequest, "LimitExceededException", "a sync configuration can have at most %d blockers", cxnMaxBlockersPerCfg)
	}
	if reason == "" {
		reason = "Stackyard: sync failed"
	}
	if len(reason) > 1024 {
		return nil, cxnValidation("CreatedReason must be at most 1024 characters")
	}
	b := &cxnBlocker{id: cxnRandHex(8), reason: reason, created: time.Now().UTC()}
	s.blockers = append(s.blockers, b)
	return cxnBlockerOut(b), nil
}

// --- tagging ---

func (st *codeConnectionsStore) tagOp(op string, in map[string]any) (map[string]any, *cxnError) {
	arn := cxnStr(in, "ResourceArn")
	kind, id, ok := cxnParseARN(arn)
	exists := false
	switch kind {
	case "connection":
		exists = st.connections[id] != nil
	case "host":
		exists = st.hosts[id] != nil
	case "repository-link":
		exists = st.links[id] != nil
	}
	if !ok || !exists {
		return nil, cxnNotFound("resource %s not found", arn)
	}
	key := kind + "/" + id
	switch op {
	case "TagResource":
		if err := st.mergeTagsLocked(key, cxnTagsIn(in["Tags"])); err != nil {
			return nil, err
		}
	case "UntagResource":
		drop := map[string]bool{}
		for _, k := range cxnListStrings(in["TagKeys"]) {
			drop[k] = true
		}
		kept := []cxnTag{}
		for _, t := range st.tags[key] {
			if !drop[t.Key] {
				kept = append(kept, t)
			}
		}
		st.tags[key] = kept
	case "ListTagsForResource":
		return map[string]any{"Tags": cxnTagsOut(st.tags[key])}, nil
	}
	return map[string]any{}, nil
}

func cxnListStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
