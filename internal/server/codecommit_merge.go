package server

import (
	"bytes"
	"sort"
	"strings"
	"unicode/utf8"
)

func registerCodeCommitMergeHandlers() {
	for name, h := range map[string]ccHandler{
		"MergeBranchesByFastForward": (*ccStore).mergeBranchesByFastForward,
		"MergeBranchesBySquash": func(st *ccStore, in map[string]any) (map[string]any, *ccErr) {
			return st.mergeBranches(in, "SQUASH_MERGE")
		},
		"MergeBranchesByThreeWay": func(st *ccStore, in map[string]any) (map[string]any, *ccErr) {
			return st.mergeBranches(in, "THREE_WAY_MERGE")
		},
		"CreateUnreferencedMergeCommit": (*ccStore).createUnreferencedMergeCommit,
		"GetMergeCommit":                (*ccStore).getMergeCommit,
		"GetMergeOptions":               (*ccStore).getMergeOptions,
		"GetMergeConflicts":             (*ccStore).getMergeConflicts,
		"DescribeMergeConflicts":        (*ccStore).describeMergeConflicts,
		"BatchDescribeMergeConflicts":   (*ccStore).batchDescribeMergeConflicts,
	} {
		ccHandlers[name] = h
	}
}

// ccConflict is a path changed incompatibly on both sides of a merge.
type ccConflict struct {
	path                    string
	base, src, dst          ccEntry
	hasBase, hasSrc, hasDst bool
	hunks                   []map[string]any
}

type ccMergeResult struct {
	src, dst, base *ccCommit
	files          map[string]ccEntry
	conflicts      []*ccConflict
	pending        map[string][]byte // line-merged content, stored only if committed
}

// merge combines src into dst. FILE_LEVEL treats any file changed on both
// sides as a conflict; LINE_LEVEL (the default) first tries a line merge.
// ponytail: the line merge only handles files whose three versions have the
// same number of lines; anything else is a whole-file conflict.
func (r *ccRepo) merge(src, dst *ccCommit, detail, strategy string) *ccMergeResult {
	res := &ccMergeResult{src: src, dst: dst, base: r.mergeBase(src, dst), files: map[string]ccEntry{}, pending: map[string][]byte{}}
	baseFiles := map[string]ccEntry{}
	if res.base != nil {
		baseFiles = res.base.files
	}
	paths := map[string]bool{}
	for _, m := range []map[string]ccEntry{baseFiles, src.files, dst.files} {
		for p := range m {
			paths[p] = true
		}
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)
	for _, p := range sorted {
		b, hb := baseFiles[p]
		s, hs := src.files[p]
		d, hd := dst.files[p]
		put := func(e ccEntry, ok bool) {
			if ok {
				res.files[p] = e
			}
		}
		switch {
		case hs == hd && s == d:
			put(s, hs)
		case hs == hb && s == b:
			put(d, hd)
		case hd == hb && d == b:
			put(s, hs)
		default:
			if detail != "FILE_LEVEL" && hb && hs && hd && s.mode == d.mode {
				if merged, ok := ccMergeLines(r.blobs[b.blob], r.blobs[s.blob], r.blobs[d.blob]); ok {
					id := ccBlobID(merged)
					res.pending[id] = merged
					res.files[p] = ccEntry{blob: id, mode: s.mode}
					continue
				}
			}
			c := &ccConflict{path: p, base: b, src: s, dst: d, hasBase: hb, hasSrc: hs, hasDst: hd}
			switch strategy {
			case "ACCEPT_SOURCE":
				put(s, hs)
			case "ACCEPT_DESTINATION":
				put(d, hd)
			default:
				c.hunks = ccHunks(r, c)
				res.conflicts = append(res.conflicts, c)
				put(d, hd) // placeholder until resolved
			}
		}
	}
	return res
}

func ccSplitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	return strings.SplitAfter(string(b), "\n")
}

// ccMergeLines merges per line when base, source and destination have the
// same line count and no line changed differently on both sides.
func ccMergeLines(base, src, dst []byte) ([]byte, bool) {
	if !utf8.Valid(base) || !utf8.Valid(src) || !utf8.Valid(dst) {
		return nil, false
	}
	b, s, d := ccSplitLines(base), ccSplitLines(src), ccSplitLines(dst)
	if len(b) != len(s) || len(b) != len(d) {
		return nil, false
	}
	var out bytes.Buffer
	for i := range b {
		switch {
		case s[i] == b[i]:
			out.WriteString(d[i])
		case d[i] == b[i] || s[i] == d[i]:
			out.WriteString(s[i])
		default:
			return nil, false
		}
	}
	return out.Bytes(), true
}

func ccHunkSide(lines []string, from, to int) map[string]any {
	if from > len(lines) {
		from = len(lines)
	}
	if to > len(lines) {
		to = len(lines)
	}
	return map[string]any{"startLine": from + 1, "endLine": to, "hunkContent": strings.Join(lines[from:to], "")}
}

// ccHunks returns the conflicting regions of c: one per differing line when
// the line counts match, otherwise one whole-file hunk.
func ccHunks(r *ccRepo, c *ccConflict) []map[string]any {
	get := func(e ccEntry, ok bool) []string {
		if !ok {
			return nil
		}
		return ccSplitLines(r.blobs[e.blob])
	}
	b, s, d := get(c.base, c.hasBase), get(c.src, c.hasSrc), get(c.dst, c.hasDst)
	if len(b) == len(s) && len(b) == len(d) && len(b) > 0 {
		var hunks []map[string]any
		for i := range b {
			if s[i] != b[i] && d[i] != b[i] && s[i] != d[i] {
				hunks = append(hunks, map[string]any{"isConflict": true, "base": ccHunkSide(b, i, i+1), "source": ccHunkSide(s, i, i+1), "destination": ccHunkSide(d, i, i+1)})
			}
		}
		if len(hunks) > 0 {
			return hunks
		}
	}
	return []map[string]any{{"isConflict": true, "base": ccHunkSide(b, 0, len(b)), "source": ccHunkSide(s, 0, len(s)), "destination": ccHunkSide(d, 0, len(d))}}
}

func ccOp(had, has bool) string {
	switch {
	case !had && has:
		return "A"
	case had && !has:
		return "D"
	}
	return "M"
}

func (r *ccRepo) conflictMetadata(c *ccConflict) map[string]any {
	size := func(e ccEntry, ok bool) int {
		if !ok {
			return 0
		}
		return len(r.blobs[e.blob])
	}
	mode := func(e ccEntry, ok bool) string {
		if !ok {
			return ""
		}
		return e.mode
	}
	objType := func(ok bool) string {
		if ok {
			return "FILE"
		}
		return ""
	}
	binary := func(e ccEntry, ok bool) bool { return ok && !utf8.Valid(r.blobs[e.blob]) }
	fileModes := map[string]any{"source": mode(c.src, c.hasSrc), "destination": mode(c.dst, c.hasDst), "base": mode(c.base, c.hasBase)}
	return map[string]any{
		"filePath":           c.path,
		"fileSizes":          map[string]any{"source": size(c.src, c.hasSrc), "destination": size(c.dst, c.hasDst), "base": size(c.base, c.hasBase)},
		"fileModes":          fileModes,
		"objectTypes":        map[string]any{"source": objType(c.hasSrc), "destination": objType(c.hasDst), "base": objType(c.hasBase)},
		"numberOfConflicts":  len(c.hunks),
		"isBinaryFile":       map[string]any{"source": binary(c.src, c.hasSrc), "destination": binary(c.dst, c.hasDst), "base": binary(c.base, c.hasBase)},
		"contentConflict":    c.hasSrc && c.hasDst && c.src.blob != c.dst.blob,
		"fileModeConflict":   c.hasSrc && c.hasDst && c.src.mode != c.dst.mode,
		"objectTypeConflict": c.hasSrc != c.hasDst,
		"mergeOperations":    map[string]any{"source": ccOp(c.hasBase, c.hasSrc), "destination": ccOp(c.hasBase, c.hasDst)},
	}
}

// resolve applies a ConflictResolution to res; it returns an error naming the
// first conflict that is still unresolved.
func (r *ccRepo) resolveConflicts(res *ccMergeResult, resolution map[string]any) *ccErr {
	resolved := map[string]bool{}
	for _, raw := range ccList(resolution, "replaceContents") {
		e, _ := raw.(map[string]any)
		p, ok := ccCleanPath(ccStr(e, "filePath"))
		if !ok || p == "" {
			return ccError("InvalidPathException", "path is not valid")
		}
		mode := ccStr(e, "fileMode")
		var c *ccConflict
		for _, cand := range res.conflicts {
			if cand.path == p {
				c = cand
			}
		}
		if c == nil {
			return ccError("InvalidConflictResolutionException", "path "+p+" has no conflict to resolve")
		}
		pick := func(entry ccEntry, ok bool) {
			if !ok {
				delete(res.files, p)
				return
			}
			if mode != "" {
				entry.mode = mode
			}
			res.files[p] = entry
		}
		switch ccStr(e, "replacementType") {
		case "KEEP_BASE":
			pick(c.base, c.hasBase)
		case "KEEP_SOURCE":
			pick(c.src, c.hasSrc)
		case "KEEP_DESTINATION":
			pick(c.dst, c.hasDst)
		case "USE_NEW_CONTENT":
			content, ok := ccDecodeB64(e, "content")
			if !ok {
				return ccError("ReplacementContentRequiredException", "USE_NEW_CONTENT needs content")
			}
			id, err := r.putBlob(content)
			if err != nil {
				return err
			}
			if mode == "" {
				mode = "NORMAL"
			}
			res.files[p] = ccEntry{blob: id, mode: mode}
		default:
			return ccError("ReplacementTypeRequiredException", "replacementType is required")
		}
		resolved[p] = true
	}
	for _, raw := range ccList(resolution, "deleteFiles") {
		e, _ := raw.(map[string]any)
		p, _ := ccCleanPath(ccStr(e, "filePath"))
		delete(res.files, p)
		resolved[p] = true
	}
	for _, raw := range ccList(resolution, "setFileModes") {
		e, _ := raw.(map[string]any)
		p, _ := ccCleanPath(ccStr(e, "filePath"))
		if f, ok := res.files[p]; ok {
			f.mode = ccStr(e, "fileMode")
			res.files[p] = f
		}
	}
	for _, c := range res.conflicts {
		if !resolved[c.path] {
			return ccError("ManualMergeRequiredException", "file "+c.path+" has conflicts that must be resolved")
		}
	}
	return nil
}

// specs resolves the source and destination of a merge request.
func (r *ccRepo) specs(in map[string]any) (*ccCommit, *ccCommit, *ccErr) {
	src, err := r.resolve(ccStr(in, "sourceCommitSpecifier"))
	if err != nil {
		return nil, nil, ccError("InvalidSourceCommitSpecifierException", err.msg)
	}
	dst, err := r.resolve(ccStr(in, "destinationCommitSpecifier"))
	if err != nil {
		return nil, nil, ccError("InvalidDestinationCommitSpecifierException", err.msg)
	}
	return src, dst, nil
}

// mergeCommit builds (but doesn't reference) the commit merging src into dst
// with option SQUASH_MERGE or THREE_WAY_MERGE.
func (r *ccRepo) mergeCommit(src, dst *ccCommit, option string, in map[string]any) (*ccCommit, *ccErr) {
	if src.id == dst.id {
		return nil, ccError("SourceAndDestinationAreSameException", "the source and destination are the same commit")
	}
	if r.isAncestor(src, dst) {
		return dst, nil // already merged
	}
	res := r.merge(src, dst, ccStr(in, "conflictDetailLevel"), ccStr(in, "conflictResolutionStrategy"))
	if err := r.resolveConflicts(res, ccMap(in, "conflictResolution")); err != nil {
		return nil, err
	}
	for _, e := range res.files {
		if content, ok := res.pending[e.blob]; ok {
			if _, err := r.putBlob(content); err != nil {
				return nil, err
			}
		}
	}
	parents := []string{dst.id}
	if option == "THREE_WAY_MERGE" {
		parents = append(parents, src.id)
	}
	msg := ccStr(in, "commitMessage")
	if msg == "" {
		msg = "Merge " + src.id[:7] + " into " + dst.id[:7]
	}
	return r.newCommit(res.files, parents, ccAuthor(in, "authorName", "email"), msg)
}

// moveTarget points targetBranch (or a destination that names a branch) at c.
func (r *ccRepo) moveTarget(in map[string]any, c *ccCommit) *ccErr {
	target := ccStr(in, "targetBranch")
	if target == "" {
		if _, ok := r.branches[ccStr(in, "destinationCommitSpecifier")]; ok {
			target = ccStr(in, "destinationCommitSpecifier")
		}
	}
	if target == "" {
		return nil
	}
	if _, ok := r.branches[target]; !ok {
		return ccError("BranchDoesNotExistException", "branch "+target+" does not exist")
	}
	r.branches[target] = c.id
	r.modified = ccNow()
	return nil
}

func (st *ccStore) mergeBranchesByFastForward(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	src, dst, err := r.specs(in)
	if err != nil {
		return nil, err
	}
	if !r.isAncestor(dst, src) {
		return nil, ccError("ManualMergeRequiredException", "the destination is not an ancestor of the source, so it can't be fast-forwarded")
	}
	if err := r.moveTarget(in, src); err != nil {
		return nil, err
	}
	return map[string]any{"commitId": src.id, "treeId": src.tree}, nil
}

func (st *ccStore) mergeBranches(in map[string]any, option string) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	src, dst, err := r.specs(in)
	if err != nil {
		return nil, err
	}
	c, err := r.mergeCommit(src, dst, option, in)
	if err != nil {
		return nil, err
	}
	if err := r.moveTarget(in, c); err != nil {
		return nil, err
	}
	return map[string]any{"commitId": c.id, "treeId": c.tree}, nil
}

func (st *ccStore) createUnreferencedMergeCommit(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	src, dst, err := r.specs(in)
	if err != nil {
		return nil, err
	}
	option := ccStr(in, "mergeOption")
	if option == "FAST_FORWARD_MERGE" {
		if !r.isAncestor(dst, src) {
			return nil, ccError("ManualMergeRequiredException", "the destination can't be fast-forwarded to the source")
		}
		return map[string]any{"commitId": src.id, "treeId": src.tree}, nil
	}
	c, err := r.mergeCommit(src, dst, option, in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"commitId": c.id, "treeId": c.tree}, nil
}

func ccBaseID(c *ccCommit) string {
	if c == nil {
		return ""
	}
	return c.id
}

func (st *ccStore) getMergeCommit(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	src, dst, err := r.specs(in)
	if err != nil {
		return nil, err
	}
	res := r.merge(src, dst, ccStr(in, "conflictDetailLevel"), ccStr(in, "conflictResolutionStrategy"))
	out := map[string]any{"sourceCommitId": src.id, "destinationCommitId": dst.id, "baseCommitId": ccBaseID(res.base)}
	// mergedCommitId names an existing merge commit of these two, if any.
	for _, c := range r.commits {
		if len(c.parents) == 2 && c.parents[0] == dst.id && c.parents[1] == src.id {
			out["mergedCommitId"] = c.id
		}
	}
	return out, nil
}

func (st *ccStore) getMergeOptions(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	src, dst, err := r.specs(in)
	if err != nil {
		return nil, err
	}
	res := r.merge(src, dst, ccStr(in, "conflictDetailLevel"), ccStr(in, "conflictResolutionStrategy"))
	options := []any{}
	if r.isAncestor(dst, src) {
		options = append(options, "FAST_FORWARD_MERGE")
	}
	if len(res.conflicts) == 0 {
		options = append(options, "SQUASH_MERGE", "THREE_WAY_MERGE")
	}
	return map[string]any{"mergeOptions": options, "sourceCommitId": src.id, "destinationCommitId": dst.id, "baseCommitId": ccBaseID(res.base)}, nil
}

func ccMaxInt(in map[string]any, key string, def int) int {
	if v, ok := in[key].(float64); ok && v > 0 && v < float64(def) {
		return int(v)
	}
	return def
}

// conflicts runs a merge for the given mergeOption; fast-forward has none
// but needs the destination to be an ancestor.
func (r *ccRepo) conflicts(in map[string]any) (*ccMergeResult, *ccErr) {
	src, dst, err := r.specs(in)
	if err != nil {
		return nil, err
	}
	if ccStr(in, "mergeOption") == "FAST_FORWARD_MERGE" {
		if !r.isAncestor(dst, src) {
			return nil, ccError("ManualMergeRequiredException", "the destination can't be fast-forwarded to the source")
		}
		return &ccMergeResult{src: src, dst: dst, base: dst, files: src.files}, nil
	}
	return r.merge(src, dst, ccStr(in, "conflictDetailLevel"), ccStr(in, "conflictResolutionStrategy")), nil
}

func (st *ccStore) getMergeConflicts(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	res, err := r.conflicts(in)
	if err != nil {
		return nil, err
	}
	in["maxResults"] = float64(ccMaxInt(in, "maxConflictFiles", ccMaxResults))
	page, next, err := ccPage(res.conflicts, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	list := []any{}
	for _, c := range page {
		list = append(list, r.conflictMetadata(c))
	}
	return map[string]any{"mergeable": len(res.conflicts) == 0, "sourceCommitId": res.src.id, "destinationCommitId": res.dst.id, "baseCommitId": ccBaseID(res.base), "conflictMetadataList": list, "nextToken": next}, nil
}

func (st *ccStore) describeMergeConflicts(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	res, err := r.conflicts(in)
	if err != nil {
		return nil, err
	}
	p, _ := ccCleanPath(ccStr(in, "filePath"))
	var found *ccConflict
	for _, c := range res.conflicts {
		if c.path == p {
			found = c
		}
	}
	if found == nil {
		if _, ok := res.files[p]; !ok {
			return nil, ccError("FileDoesNotExistException", "file "+p+" is not part of this merge")
		}
		found = &ccConflict{path: p}
		found.src, found.hasSrc = res.src.files[p]
		found.dst, found.hasDst = res.dst.files[p]
		if res.base != nil {
			found.base, found.hasBase = res.base.files[p]
		}
	}
	hunks := found.hunks
	if len(hunks) > ccMaxInt(in, "maxMergeHunks", 1000) {
		hunks = hunks[:ccMaxInt(in, "maxMergeHunks", 1000)]
	}
	return map[string]any{"conflictMetadata": r.conflictMetadata(found), "mergeHunks": hunks, "sourceCommitId": res.src.id, "destinationCommitId": res.dst.id, "baseCommitId": ccBaseID(res.base)}, nil
}

func (st *ccStore) batchDescribeMergeConflicts(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	res, err := r.conflicts(in)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, raw := range ccList(in, "filePaths") {
		if s, ok := raw.(string); ok {
			p, _ := ccCleanPath(s)
			want[p] = true
		}
	}
	var picked []*ccConflict
	for _, c := range res.conflicts {
		if len(want) == 0 || want[c.path] {
			picked = append(picked, c)
		}
	}
	in["maxResults"] = float64(ccMaxInt(in, "maxConflictFiles", ccMaxResults))
	page, next, err := ccPage(picked, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	maxHunks := ccMaxInt(in, "maxMergeHunks", 1000)
	conflicts := []any{}
	for _, c := range page {
		hunks := c.hunks
		if len(hunks) > maxHunks {
			hunks = hunks[:maxHunks]
		}
		conflicts = append(conflicts, map[string]any{"conflictMetadata": r.conflictMetadata(c), "mergeHunks": hunks})
	}
	return map[string]any{"conflicts": conflicts, "errors": []any{}, "nextToken": next, "sourceCommitId": res.src.id, "destinationCommitId": res.dst.id, "baseCommitId": ccBaseID(res.base)}, nil
}
