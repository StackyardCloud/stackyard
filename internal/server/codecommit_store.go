package server

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Limits. AWS's own: 6 MB per file and per commit, 100 commits per
// BatchGetCommits, 10 triggers, 50 tags. Repository/commit/comment counts are
// Stackyard caps so a long-lived server can't be grown without bound.
const (
	ccMaxFileBytes       = 6 << 20
	ccMaxRepoBytes       = 64 << 20  // total blob bytes per repository
	ccMaxStoreBytes      = 512 << 20 // total blob bytes across repositories
	ccMaxRepoEntries     = 1 << 20   // file entries summed over a repository's commits
	ccMaxRepos           = 1000
	ccMaxCommitsPerRepo  = 10000
	ccMaxFilesPerTree    = 10000
	ccMaxPathBytes       = 4096
	ccMaxBranchesPerRepo = 1000
	ccMaxAncestry        = 10000 // commits walked when searching history
	ccMaxPullRequests    = 5000
	ccMaxComments        = 20000
	ccMaxTemplates       = 1000
	ccMaxTriggers        = 10
	ccMaxTags            = 50
	ccMaxResults         = 100
	ccCallerARN          = "arn:aws:iam::123456789012:user/stackyard"
	ccAccountID          = "123456789012"
	ccRegion             = "us-east-1"
)

type ccEntry struct {
	blob string
	mode string // NORMAL | EXECUTABLE | SYMLINK
}

type ccCommit struct {
	id, tree string
	parents  []string
	files    map[string]ccEntry // path (no leading slash) -> entry
	author   ccPerson
	message  string
}

type ccPerson struct {
	name, email string
	when        time.Time
}

type ccRepo struct {
	id, name, arn, description, defaultBranch, kmsKey string
	created, modified                                 time.Time
	branches                                          map[string]string // name -> commit id
	commits                                           map[string]*ccCommit
	blobs                                             map[string][]byte
	blobBytes, entries                                int
	store                                             *ccStore
	tags                                              map[string]string
	triggers                                          []any
	triggerConfigID                                   string
	templates                                         map[string]bool // associated approval rule template names
}

type ccStore struct {
	mu        sync.Mutex
	repos     map[string]*ccRepo // by name
	prs       map[string]*ccPullRequest
	prOrder   []string
	prTokens  map[string]string // clientRequestToken -> pull request id
	nextPR    int
	templates map[string]*ccTemplate // by name
	comments  map[string]*ccComment
	ctokens   map[string]string // comment clientRequestToken -> comment id
	blobBytes int               // content bytes held across all repositories
}

func newCodeCommitStore() *ccStore {
	return &ccStore{
		repos:     map[string]*ccRepo{},
		prs:       map[string]*ccPullRequest{},
		prTokens:  map[string]string{},
		nextPR:    1,
		templates: map[string]*ccTemplate{},
		comments:  map[string]*ccComment{},
		ctokens:   map[string]string{},
	}
}

type ccHandler func(st *ccStore, in map[string]any) (map[string]any, *ccErr)

var ccHandlers map[string]ccHandler

// handle runs op under the store lock and shapes the result to the model's
// output there too, so nothing read from the store escapes the lock.
func (st *ccStore) handle(op, outShape string, in map[string]any) (any, *ccErr) {
	h, ok := ccHandlers[op]
	if !ok {
		return nil, ccError("UnknownOperationException", op+" is not implemented")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, in)
	if err != nil {
		return nil, err
	}
	if out == nil || outShape == "" {
		return map[string]any{}, nil
	}
	return codecommitConform(outShape, out), nil
}

// ---- small helpers over decoded JSON ----

func ccStr(in map[string]any, key string) string {
	s, _ := in[key].(string)
	return s
}

func ccBool(in map[string]any, key string) bool {
	b, _ := in[key].(bool)
	return b
}

func ccList(in map[string]any, key string) []any {
	l, _ := in[key].([]any)
	return l
}

func ccMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}

func ccNow() time.Time { return time.Now().UTC() }

func ccEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func ccUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

func ccSHA(parts ...string) string {
	h := sha1.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func ccRandomSHA() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func ccBlobID(content []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

func ccB64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func ccDecodeB64(in map[string]any, key string) ([]byte, bool) {
	s, ok := in[key].(string)
	if !ok {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(s)
	return b, err == nil
}

// ccCleanPath normalises a repository path: no leading or trailing slash,
// no empty, "." or ".." segments.
func ccCleanPath(p string) (string, bool) {
	p = strings.Trim(p, "/")
	if p == "" {
		return "", true
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", false
		}
	}
	return p, true
}

// ccPage returns one page of items and the next token (an offset).
func ccPage[T any](items []T, in map[string]any, tokenKey, maxKey string) ([]T, string, *ccErr) {
	start := 0
	if tok := ccStr(in, tokenKey); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", ccError("InvalidContinuationTokenException", "the continuation token is not valid")
		}
		start = int(n)
	}
	limit := ccMaxResults
	if v, ok := in[maxKey].(float64); ok && v > 0 && v < float64(ccMaxResults) {
		limit = int(v)
	}
	end := min(start+limit, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

// ---- repositories, commits and references ----

func (st *ccStore) repo(in map[string]any) (*ccRepo, *ccErr) {
	name := ccStr(in, "repositoryName")
	if name == "" {
		return nil, ccError("RepositoryNameRequiredException", "repositoryName is required")
	}
	r, ok := st.repos[name]
	if !ok {
		return nil, ccError("RepositoryDoesNotExistException", name+" does not exist")
	}
	return r, nil
}

func ccRepoARN(name string) string {
	return "arn:aws:codecommit:" + ccRegion + ":" + ccAccountID + ":" + name
}

// resolve turns a commit specifier (branch name, commit id, or HEAD-like
// default) into a commit in repo.
func (r *ccRepo) resolve(spec string) (*ccCommit, *ccErr) {
	if spec == "" {
		spec = r.defaultBranch
		if spec == "" {
			return nil, ccError("CommitDoesNotExistException", "the repository is empty")
		}
	}
	if id, ok := r.branches[strings.TrimPrefix(spec, "refs/heads/")]; ok {
		return r.commits[id], nil
	}
	if c, ok := r.commits[spec]; ok {
		return c, nil
	}
	return nil, ccError("CommitDoesNotExistException", "commit "+spec+" does not exist")
}

func (r *ccRepo) commit(id string) (*ccCommit, *ccErr) {
	if id == "" {
		return nil, ccError("CommitIdRequiredException", "commitId is required")
	}
	if len(id) != 40 {
		return nil, ccError("InvalidCommitIdException", "commit id "+id+" is not valid")
	}
	c, ok := r.commits[id]
	if !ok {
		return nil, ccError("CommitIdDoesNotExistException", "commit "+id+" does not exist")
	}
	return c, nil
}

func (r *ccRepo) branch(name string) (string, *ccErr) {
	if name == "" {
		return "", ccError("BranchNameRequiredException", "branchName is required")
	}
	id, ok := r.branches[name]
	if !ok {
		return "", ccError("BranchDoesNotExistException", "branch "+name+" does not exist")
	}
	return id, nil
}

// ancestors returns c's ancestry (c first) breadth-first, bounded.
func (r *ccRepo) ancestors(c *ccCommit) []*ccCommit {
	seen := map[string]bool{c.id: true}
	queue := []*ccCommit{c}
	for i := 0; i < len(queue) && len(queue) < ccMaxAncestry; i++ {
		for _, p := range queue[i].parents {
			if pc, ok := r.commits[p]; ok && !seen[p] {
				seen[p] = true
				queue = append(queue, pc)
			}
		}
	}
	return queue
}

func (r *ccRepo) isAncestor(anc, of *ccCommit) bool {
	for _, c := range r.ancestors(of) {
		if c.id == anc.id {
			return true
		}
	}
	return false
}

// mergeBase is the nearest common ancestor of a and b (nil if none).
func (r *ccRepo) mergeBase(a, b *ccCommit) *ccCommit {
	inA := map[string]bool{}
	for _, c := range r.ancestors(a) {
		inA[c.id] = true
	}
	for _, c := range r.ancestors(b) {
		if inA[c.id] {
			return c
		}
	}
	return nil
}

func ccTreeID(files map[string]ccEntry) string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	parts := []string{"tree"}
	for _, p := range paths {
		parts = append(parts, p, files[p].mode, files[p].blob)
	}
	return ccSHA(parts...)
}

// treeIDFor returns the id of the folder at dir ("" = root) in files.
func ccFolderTreeID(files map[string]ccEntry, dir string) string {
	if dir == "" {
		return ccTreeID(files)
	}
	sub := map[string]ccEntry{}
	for p, e := range files {
		if strings.HasPrefix(p, dir+"/") {
			sub[strings.TrimPrefix(p, dir+"/")] = e
		}
	}
	return ccTreeID(sub)
}

// newCommit stores a commit with files on top of parents and returns it.
func (r *ccRepo) newCommit(files map[string]ccEntry, parents []string, author ccPerson, message string) (*ccCommit, *ccErr) {
	if len(r.commits) >= ccMaxCommitsPerRepo {
		return nil, ccError("RepositoryLimitExceededException", fmt.Sprintf("a repository can hold at most %d commits", ccMaxCommitsPerRepo))
	}
	if len(files) > ccMaxFilesPerTree {
		return nil, ccError("MaximumFileEntriesExceededException", fmt.Sprintf("a commit can hold at most %d files", ccMaxFilesPerTree))
	}
	// Each commit keeps its own file map, so bound the total.
	if r.entries+len(files) > ccMaxRepoEntries {
		return nil, ccError("RepositoryLimitExceededException", "the repository has reached its history size limit")
	}
	// Messages, authors and paths are stored per commit too, so count them
	// against the same byte budgets as blob content.
	meta := len(message) + len(author.name) + len(author.email)
	for p := range files {
		if len(p) > ccMaxPathBytes {
			return nil, ccError("InvalidPathException", fmt.Sprintf("file paths can be at most %d bytes", ccMaxPathBytes))
		}
		meta += len(p)
	}
	if r.blobBytes+meta > ccMaxRepoBytes || r.store.blobBytes+meta > ccMaxStoreBytes {
		return nil, ccError("RepositoryLimitExceededException", "the repository has reached its content size limit")
	}
	r.entries += len(files)
	r.blobBytes += meta
	r.store.blobBytes += meta
	tree := ccTreeID(files)
	c := &ccCommit{tree: tree, parents: parents, files: files, author: author, message: message}
	c.id = ccSHA("commit", tree, strings.Join(parents, ","), author.name, author.email, message, author.when.Format(time.RFC3339Nano), ccRandomSHA())
	r.commits[c.id] = c
	r.modified = ccNow()
	return c, nil
}

// putBlob stores content in the repository and returns its id.
func (r *ccRepo) putBlob(content []byte) (string, *ccErr) {
	if len(content) > ccMaxFileBytes {
		return "", ccError("FileContentSizeLimitExceededException", "file content exceeds 6 MB")
	}
	id := ccBlobID(content)
	if _, ok := r.blobs[id]; ok {
		return id, nil
	}
	if r.blobBytes+len(content) > ccMaxRepoBytes || r.store.blobBytes+len(content) > ccMaxStoreBytes {
		return "", ccError("RepositoryLimitExceededException", "the repository has reached its content size limit")
	}
	r.blobs[id] = append([]byte(nil), content...)
	r.blobBytes += len(content)
	r.store.blobBytes += len(content)
	return id, nil
}

func ccCopyFiles(files map[string]ccEntry) map[string]ccEntry {
	out := make(map[string]ccEntry, len(files))
	for k, v := range files {
		out[k] = v
	}
	return out
}

// ---- output shapes ----

func ccPersonOut(p ccPerson) map[string]any {
	return map[string]any{"name": p.name, "email": p.email, "date": fmt.Sprintf("%d +0000", p.when.Unix())}
}

func ccCommitOut(c *ccCommit) map[string]any {
	parents := make([]any, 0, len(c.parents))
	for _, p := range c.parents {
		parents = append(parents, p)
	}
	return map[string]any{
		"commitId": c.id, "treeId": c.tree, "parents": parents, "message": c.message,
		"author": ccPersonOut(c.author), "committer": ccPersonOut(c.author), "additionalData": "",
	}
}

func (r *ccRepo) metadata() map[string]any {
	out := map[string]any{
		"accountId": ccAccountID, "repositoryId": r.id, "repositoryName": r.name, "Arn": r.arn,
		"creationDate": ccEpoch(r.created), "lastModifiedDate": ccEpoch(r.modified),
		"cloneUrlHttp": "https://git-codecommit." + ccRegion + ".amazonaws.com/v1/repos/" + r.name,
		"cloneUrlSsh":  "ssh://git-codecommit." + ccRegion + ".amazonaws.com/v1/repos/" + r.name,
		"kmsKeyId":     r.kmsKey,
	}
	if r.description != "" {
		out["repositoryDescription"] = r.description
	}
	if r.defaultBranch != "" {
		out["defaultBranch"] = r.defaultBranch
	}
	return out
}

func ccAuthor(in map[string]any, nameKey, emailKey string) ccPerson {
	name := ccStr(in, nameKey)
	if name == "" {
		name = "stackyard"
	}
	return ccPerson{name: name, email: ccStr(in, emailKey), when: ccNow()}
}
