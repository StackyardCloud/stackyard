package server

import (
	"sort"
	"strings"
)

func registerCodeCommitFileHandlers() {
	for name, h := range map[string]ccHandler{
		"CreateCommit":          (*ccStore).createCommit,
		"PutFile":               (*ccStore).putFile,
		"DeleteFile":            (*ccStore).deleteFile,
		"GetFile":               (*ccStore).getFile,
		"GetFolder":             (*ccStore).getFolder,
		"GetBlob":               (*ccStore).getBlob,
		"GetCommit":             (*ccStore).getCommit,
		"BatchGetCommits":       (*ccStore).batchGetCommits,
		"GetDifferences":        (*ccStore).getDifferences,
		"GetBlobDifferences":    (*ccStore).getBlobDifferences,
		"ListFileCommitHistory": (*ccStore).listFileCommitHistory,
	} {
		ccHandlers[name] = h
	}
}

const ccMaxCommitMessage = 100 << 10

// ccChange is one file change requested by CreateCommit, PutFile or DeleteFile.
type ccChange struct {
	path    string
	content []byte
	source  string // copy content from this path in the parent
	move    bool
	mode    string
	delete  bool
	setMode bool
}

type ccCommitResult struct {
	commit                  *ccCommit
	added, updated, removed []any
}

// parentFor checks parentCommitId against branch and returns the parent
// commit (nil for the first commit of an empty repository).
func (r *ccRepo) parentFor(branch, parentID string) (*ccCommit, *ccErr) {
	tip, exists := r.branches[branch]
	if parentID == "" {
		if exists || len(r.commits) > 0 {
			return nil, ccError("ParentCommitIdRequiredException", "parentCommitId is required for a branch with commits")
		}
		return nil, nil
	}
	parent, ok := r.commits[parentID]
	if !ok {
		if len(parentID) != 40 {
			return nil, ccError("InvalidParentCommitIdException", "parent commit id "+parentID+" is not valid")
		}
		return nil, ccError("ParentCommitDoesNotExistException", "parent commit "+parentID+" does not exist")
	}
	if exists && tip != parentID {
		return nil, ccError("ParentCommitIdOutdatedException", "the parent commit is not the tip of branch "+branch)
	}
	return parent, nil
}

// applyChanges commits changes onto parent and moves branch to the result.
func (r *ccRepo) applyChanges(branch string, parent *ccCommit, changes []ccChange, author ccPerson, message string) (*ccCommitResult, *ccErr) {
	if len(message) > ccMaxCommitMessage {
		return nil, ccError("CommitMessageLengthExceededException", "the commit message exceeds 100 KB")
	}
	base := map[string]ccEntry{}
	var parents []string
	if parent != nil {
		base, parents = parent.files, []string{parent.id}
	}
	files := ccCopyFiles(base)
	seen := map[string]bool{}
	res := &ccCommitResult{added: []any{}, updated: []any{}, removed: []any{}}
	for _, ch := range changes {
		p, ok := ccCleanPath(ch.path)
		if !ok || p == "" {
			return nil, ccError("InvalidPathException", "path "+ch.path+" is not valid")
		}
		if seen[p] && !ch.setMode {
			return nil, ccError("PutFileEntryConflictException", "path "+p+" is changed more than once")
		}
		seen[p] = true
		existing, had := files[p]
		switch {
		case ch.delete:
			if !had {
				return nil, ccError("FileDoesNotExistException", "file "+p+" does not exist")
			}
			delete(files, p)
			res.removed = append(res.removed, map[string]any{"absolutePath": p, "blobId": existing.blob, "fileMode": existing.mode})
		case ch.setMode:
			if !had {
				return nil, ccError("FileDoesNotExistException", "file "+p+" does not exist")
			}
			files[p] = ccEntry{blob: existing.blob, mode: ch.mode}
			res.updated = append(res.updated, map[string]any{"absolutePath": p, "blobId": existing.blob, "fileMode": ch.mode})
		default:
			blob := ""
			if ch.source != "" {
				src, ok := ccCleanPath(ch.source)
				srcEntry, found := base[src]
				if !ok || !found {
					return nil, ccError("FileDoesNotExistException", "source file "+ch.source+" does not exist")
				}
				blob = srcEntry.blob
				if ch.move {
					delete(files, src)
					res.removed = append(res.removed, map[string]any{"absolutePath": src, "blobId": srcEntry.blob, "fileMode": srcEntry.mode})
				}
			} else {
				id, err := r.putBlob(ch.content)
				if err != nil {
					return nil, err
				}
				blob = id
			}
			if err := ccCheckPathConflicts(files, p); err != nil {
				return nil, err
			}
			mode := ch.mode
			if mode == "" {
				mode = "NORMAL"
				if had {
					mode = existing.mode
				}
			}
			files[p] = ccEntry{blob: blob, mode: mode}
			meta := map[string]any{"absolutePath": p, "blobId": blob, "fileMode": mode}
			if had {
				res.updated = append(res.updated, meta)
			} else {
				res.added = append(res.added, meta)
			}
		}
	}
	if parent != nil && ccTreeID(files) == parent.tree {
		return nil, ccError("NoChangeException", "the commit makes no changes")
	}
	c, err := r.newCommit(files, parents, author, message)
	if err != nil {
		return nil, err
	}
	r.branches[branch] = c.id
	if r.defaultBranch == "" {
		r.defaultBranch = branch
	}
	res.commit = c
	return res, nil
}

// ccCheckPathConflicts rejects a file whose path is an existing folder, or
// whose folder is an existing file.
func ccCheckPathConflicts(files map[string]ccEntry, p string) *ccErr {
	for other := range files {
		if strings.HasPrefix(other, p+"/") {
			return ccError("FileNameConflictsWithDirectoryNameException", "file "+p+" would replace a folder")
		}
	}
	for dir := p; strings.Contains(dir, "/"); {
		dir = dir[:strings.LastIndex(dir, "/")]
		if _, ok := files[dir]; ok {
			return ccError("DirectoryNameConflictsWithFileNameException", "folder "+dir+" is already a file")
		}
	}
	return nil
}

func (st *ccStore) createCommit(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	branch := ccStr(in, "branchName")
	parent, err := r.parentFor(branch, ccStr(in, "parentCommitId"))
	if err != nil {
		return nil, err
	}
	var changes []ccChange
	for _, raw := range ccList(in, "putFiles") {
		e, _ := raw.(map[string]any)
		ch := ccChange{path: ccStr(e, "filePath"), mode: ccStr(e, "fileMode")}
		content, hasContent := ccDecodeB64(e, "fileContent")
		src := ccMap(e, "sourceFile")
		switch {
		case hasContent && src != nil:
			return nil, ccError("FileContentAndSourceFileSpecifiedException", "give either fileContent or sourceFile, not both")
		case src != nil:
			ch.source, ch.move = ccStr(src, "filePath"), ccBool(src, "isMove")
		case hasContent:
			ch.content = content
		case e["fileContent"] != nil:
			return nil, ccError("InvalidFileLocationException", "fileContent is not valid base64")
		default:
			return nil, ccError("SourceFileOrContentRequiredException", "a putFiles entry needs fileContent or sourceFile")
		}
		changes = append(changes, ch)
	}
	for _, raw := range ccList(in, "deleteFiles") {
		e, _ := raw.(map[string]any)
		changes = append(changes, ccChange{path: ccStr(e, "filePath"), delete: true})
	}
	for _, raw := range ccList(in, "setFileModes") {
		e, _ := raw.(map[string]any)
		changes = append(changes, ccChange{path: ccStr(e, "filePath"), mode: ccStr(e, "fileMode"), setMode: true})
	}
	if len(changes) == 0 {
		return nil, ccError("FileEntryRequiredException", "a commit needs at least one file change")
	}
	res, err := r.applyChanges(branch, parent, changes, ccAuthor(in, "authorName", "email"), ccStr(in, "commitMessage"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"commitId": res.commit.id, "treeId": res.commit.tree, "filesAdded": res.added, "filesUpdated": res.updated, "filesDeleted": res.removed}, nil
}

func (st *ccStore) putFile(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	content, ok := ccDecodeB64(in, "fileContent")
	if !ok {
		return nil, ccError("FileContentRequiredException", "fileContent is required and must be base64")
	}
	branch := ccStr(in, "branchName")
	parent, err := r.parentFor(branch, ccStr(in, "parentCommitId"))
	if err != nil {
		return nil, err
	}
	p, _ := ccCleanPath(ccStr(in, "filePath"))
	mode := ccStr(in, "fileMode")
	if parent != nil {
		if e, had := parent.files[p]; had && e.blob == ccBlobID(content) && (mode == "" || mode == e.mode) {
			return nil, ccError("SameFileContentException", "the file already has this content")
		}
	}
	res, err := r.applyChanges(branch, parent, []ccChange{{path: ccStr(in, "filePath"), content: content, mode: mode}}, ccAuthor(in, "name", "email"), ccStr(in, "commitMessage"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"commitId": res.commit.id, "blobId": res.commit.files[p].blob, "treeId": res.commit.tree}, nil
}

func (st *ccStore) deleteFile(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	branch := ccStr(in, "branchName")
	if _, err := r.branch(branch); err != nil {
		return nil, err
	}
	parent, err := r.parentFor(branch, ccStr(in, "parentCommitId"))
	if err != nil {
		return nil, err
	}
	p, _ := ccCleanPath(ccStr(in, "filePath"))
	old, had := parent.files[p]
	if !had {
		return nil, ccError("FileDoesNotExistException", "file "+p+" does not exist")
	}
	res, err := r.applyChanges(branch, parent, []ccChange{{path: p, delete: true}}, ccAuthor(in, "name", "email"), ccStr(in, "commitMessage"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"commitId": res.commit.id, "blobId": old.blob, "treeId": res.commit.tree, "filePath": p}, nil
}

func (st *ccStore) getFile(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	c, err := r.resolve(ccStr(in, "commitSpecifier"))
	if err != nil {
		return nil, err
	}
	p, ok := ccCleanPath(ccStr(in, "filePath"))
	if !ok {
		return nil, ccError("InvalidPathException", "path is not valid")
	}
	e, had := c.files[p]
	if !had {
		return nil, ccError("FileDoesNotExistException", "file "+p+" does not exist at commit "+c.id)
	}
	content := r.blobs[e.blob]
	return map[string]any{"commitId": c.id, "blobId": e.blob, "filePath": p, "fileMode": e.mode, "fileSize": len(content), "fileContent": ccB64(content)}, nil
}

func (st *ccStore) getFolder(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	c, err := r.resolve(ccStr(in, "commitSpecifier"))
	if err != nil {
		return nil, err
	}
	dir, ok := ccCleanPath(ccStr(in, "folderPath"))
	if !ok {
		return nil, ccError("InvalidPathException", "path is not valid")
	}
	prefix := ""
	if dir != "" {
		prefix = dir + "/"
	}
	files, links, subs := []any{}, []any{}, []any{}
	seenDir := map[string]bool{}
	found := dir == ""
	paths := make([]string, 0, len(c.files))
	for p := range c.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		found = true
		rel := strings.TrimPrefix(p, prefix)
		if i := strings.Index(rel, "/"); i >= 0 {
			name := rel[:i]
			if !seenDir[name] {
				seenDir[name] = true
				subs = append(subs, map[string]any{"absolutePath": prefix + name, "relativePath": name, "treeId": ccFolderTreeID(c.files, prefix+name)})
			}
			continue
		}
		e := c.files[p]
		entry := map[string]any{"absolutePath": p, "relativePath": rel, "blobId": e.blob, "fileMode": e.mode}
		if e.mode == "SYMLINK" {
			links = append(links, entry)
		} else {
			files = append(files, entry)
		}
	}
	if !found {
		return nil, ccError("FolderDoesNotExistException", "folder "+dir+" does not exist at commit "+c.id)
	}
	folderPath := "/" + dir
	return map[string]any{"commitId": c.id, "folderPath": folderPath, "treeId": ccFolderTreeID(c.files, dir), "files": files, "symbolicLinks": links, "subFolders": subs, "subModules": []any{}}, nil
}

func (st *ccStore) getBlob(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	id := ccStr(in, "blobId")
	if len(id) != 40 {
		return nil, ccError("InvalidBlobIdException", "blob id "+id+" is not valid")
	}
	content, ok := r.blobs[id]
	if !ok {
		return nil, ccError("BlobIdDoesNotExistException", "blob "+id+" does not exist")
	}
	return map[string]any{"content": ccB64(content)}, nil
}

func (st *ccStore) getCommit(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	c, err := r.commit(ccStr(in, "commitId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"commit": ccCommitOut(c)}, nil
}

func (st *ccStore) batchGetCommits(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	ids := ccList(in, "commitIds")
	if len(ids) > 100 {
		return nil, ccError("CommitIdsLimitExceededException", "at most 100 commit ids can be requested")
	}
	commits, errs := []any{}, []any{}
	for _, raw := range ids {
		id, _ := raw.(string)
		if c, ok := r.commits[id]; ok {
			commits = append(commits, ccCommitOut(c))
		} else {
			errs = append(errs, map[string]any{"commitId": id, "errorCode": "CommitDoesNotExistException", "errorMessage": "commit " + id + " does not exist"})
		}
	}
	return map[string]any{"commits": commits, "errors": errs}, nil
}

// ccDiff lists the file differences from before to after (nil = empty tree),
// limited to paths under the given prefixes.
func ccDiff(before, after map[string]ccEntry, beforePath, afterPath string) []map[string]any {
	under := func(p, prefix string) bool {
		return prefix == "" || p == prefix || strings.HasPrefix(p, prefix+"/")
	}
	paths := map[string]bool{}
	for p := range before {
		paths[p] = true
	}
	for p := range after {
		paths[p] = true
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)
	out := []map[string]any{}
	for _, p := range sorted {
		b, hadB := before[p]
		a, hadA := after[p]
		if hadB && hadA && b == a {
			continue
		}
		if !under(p, beforePath) && !under(p, afterPath) {
			continue
		}
		d := map[string]any{}
		switch {
		case !hadB:
			d["changeType"] = "A"
		case !hadA:
			d["changeType"] = "D"
		default:
			d["changeType"] = "M"
		}
		if hadB {
			d["beforeBlob"] = map[string]any{"blobId": b.blob, "path": p, "mode": ccGitMode(b.mode)}
		}
		if hadA {
			d["afterBlob"] = map[string]any{"blobId": a.blob, "path": p, "mode": ccGitMode(a.mode)}
		}
		out = append(out, d)
	}
	return out
}

func ccGitMode(mode string) string {
	switch mode {
	case "EXECUTABLE":
		return "100755"
	case "SYMLINK":
		return "120000"
	}
	return "100644"
}

func (st *ccStore) getDifferences(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	after, err := r.resolve(ccStr(in, "afterCommitSpecifier"))
	if err != nil {
		return nil, err
	}
	var beforeFiles map[string]ccEntry
	if spec := ccStr(in, "beforeCommitSpecifier"); spec != "" {
		before, err := r.resolve(spec)
		if err != nil {
			return nil, err
		}
		beforeFiles = before.files
	}
	beforePath, _ := ccCleanPath(ccStr(in, "beforePath"))
	afterPath, _ := ccCleanPath(ccStr(in, "afterPath"))
	diffs := ccDiff(beforeFiles, after.files, beforePath, afterPath)
	if (beforePath != "" || afterPath != "") && len(diffs) == 0 && !ccHasPath(after.files, afterPath) && !ccHasPath(beforeFiles, beforePath) {
		return nil, ccError("PathDoesNotExistException", "the path does not exist")
	}
	page, next, err := ccPage(diffs, in, "NextToken", "MaxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"differences": page, "NextToken": next}, nil
}

func ccHasPath(files map[string]ccEntry, p string) bool {
	if p == "" {
		return false
	}
	for f := range files {
		if f == p || strings.HasPrefix(f, p+"/") {
			return true
		}
	}
	return false
}

func (st *ccStore) listFileCommitHistory(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	start, err := r.resolve(ccStr(in, "commitSpecifier"))
	if err != nil {
		return nil, err
	}
	p, _ := ccCleanPath(ccStr(in, "filePath"))
	versions := []map[string]any{}
	children := map[string][]any{}
	for _, c := range r.ancestors(start) {
		for _, parent := range c.parents {
			children[parent] = append(children[parent], c.id)
		}
	}
	for _, c := range r.ancestors(start) {
		e, has := c.files[p]
		if !has {
			continue
		}
		changed := len(c.parents) == 0
		for _, parent := range c.parents {
			pc, ok := r.commits[parent]
			if !ok {
				changed = true
				continue
			}
			if pe, ok := pc.files[p]; !ok || pe != e {
				changed = true
			}
		}
		if !changed {
			continue
		}
		kids := children[c.id]
		if kids == nil {
			kids = []any{}
		}
		versions = append(versions, map[string]any{"commit": ccCommitOut(c), "blobId": e.blob, "path": p, "revisionChildren": kids})
	}
	if len(versions) == 0 {
		return nil, ccError("FileDoesNotExistException", "file "+p+" does not exist in this history")
	}
	page, next, err := ccPage(versions, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"revisionDag": page, "nextToken": next}, nil
}
