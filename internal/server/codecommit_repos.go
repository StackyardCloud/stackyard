package server

import (
	"fmt"
	"sort"
	"strings"
)

func init() {
	ccHandlers = map[string]ccHandler{
		// repositories
		"CreateRepository":              (*ccStore).createRepository,
		"GetRepository":                 (*ccStore).getRepository,
		"BatchGetRepositories":          (*ccStore).batchGetRepositories,
		"ListRepositories":              (*ccStore).listRepositories,
		"DeleteRepository":              (*ccStore).deleteRepository,
		"UpdateRepositoryName":          (*ccStore).updateRepositoryName,
		"UpdateRepositoryDescription":   (*ccStore).updateRepositoryDescription,
		"UpdateRepositoryEncryptionKey": (*ccStore).updateRepositoryEncryptionKey,
		"UpdateDefaultBranch":           (*ccStore).updateDefaultBranch,
		// branches
		"CreateBranch": (*ccStore).createBranch,
		"GetBranch":    (*ccStore).getBranch,
		"ListBranches": (*ccStore).listBranches,
		"DeleteBranch": (*ccStore).deleteBranch,
		// tags and triggers
		"TagResource":            (*ccStore).tagResource,
		"UntagResource":          (*ccStore).untagResource,
		"ListTagsForResource":    (*ccStore).listTagsForResource,
		"PutRepositoryTriggers":  (*ccStore).putRepositoryTriggers,
		"GetRepositoryTriggers":  (*ccStore).getRepositoryTriggers,
		"TestRepositoryTriggers": (*ccStore).testRepositoryTriggers,
	}
	registerCodeCommitFileHandlers()
	registerCodeCommitMergeHandlers()
	registerCodeCommitPullRequestHandlers()
	registerCodeCommitCommentHandlers()
}

func (st *ccStore) createRepository(in map[string]any) (map[string]any, *ccErr) {
	name := ccStr(in, "repositoryName")
	if _, ok := st.repos[name]; ok {
		return nil, ccError("RepositoryNameExistsException", "repository "+name+" already exists")
	}
	if len(st.repos) >= ccMaxRepos {
		return nil, ccError("RepositoryLimitExceededException", fmt.Sprintf("an account can have at most %d repositories", ccMaxRepos))
	}
	tags, err := ccTags(ccMap(in, "tags"), nil)
	if err != nil {
		return nil, err
	}
	kms := ccStr(in, "kmsKeyId")
	if kms == "" {
		kms = "alias/aws/codecommit"
	}
	now := ccNow()
	r := &ccRepo{
		id: ccUUID(), name: name, arn: ccRepoARN(name), description: ccStr(in, "repositoryDescription"), kmsKey: kms,
		created: now, modified: now, branches: map[string]string{}, commits: map[string]*ccCommit{},
		blobs: map[string][]byte{}, tags: tags, templates: map[string]bool{}, store: st,
	}
	st.repos[name] = r
	return map[string]any{"repositoryMetadata": r.metadata()}, nil
}

func (st *ccStore) getRepository(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"repositoryMetadata": r.metadata()}, nil
}

func (st *ccStore) batchGetRepositories(in map[string]any) (map[string]any, *ccErr) {
	names := ccList(in, "repositoryNames")
	if len(names) > 100 {
		return nil, ccError("MaximumRepositoryNamesExceededException", "at most 100 repository names can be requested")
	}
	found, missing, errs := []any{}, []any{}, []any{}
	for _, n := range names {
		name, _ := n.(string)
		if r, ok := st.repos[name]; ok {
			found = append(found, r.metadata())
		} else {
			missing = append(missing, name)
			errs = append(errs, map[string]any{"repositoryName": name, "errorCode": "RepositoryDoesNotExistException", "errorMessage": name + " does not exist"})
		}
	}
	return map[string]any{"repositories": found, "repositoriesNotFound": missing, "errors": errs}, nil
}

func (st *ccStore) listRepositories(in map[string]any) (map[string]any, *ccErr) {
	repos := make([]*ccRepo, 0, len(st.repos))
	for _, r := range st.repos {
		repos = append(repos, r)
	}
	byDate := ccStr(in, "sortBy") == "lastModifiedDate"
	desc := ccStr(in, "order") == "descending"
	sort.Slice(repos, func(i, j int) bool {
		less := repos[i].name < repos[j].name
		if byDate && !repos[i].modified.Equal(repos[j].modified) {
			less = repos[i].modified.Before(repos[j].modified)
		}
		if desc {
			return !less
		}
		return less
	})
	page, next, err := ccPage(repos, in, "nextToken", "")
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, r := range page {
		out = append(out, map[string]any{"repositoryName": r.name, "repositoryId": r.id})
	}
	return map[string]any{"repositories": out, "nextToken": next}, nil
}

func (st *ccStore) deleteRepository(in map[string]any) (map[string]any, *ccErr) {
	name := ccStr(in, "repositoryName")
	r, ok := st.repos[name]
	if !ok {
		return map[string]any{}, nil // the service returns no id for a missing repository
	}
	delete(st.repos, name)
	st.blobBytes -= r.blobBytes
	return map[string]any{"repositoryId": r.id}, nil
}

func (st *ccStore) updateRepositoryName(in map[string]any) (map[string]any, *ccErr) {
	oldName, newName := ccStr(in, "oldName"), ccStr(in, "newName")
	r, ok := st.repos[oldName]
	if !ok {
		return nil, ccError("RepositoryDoesNotExistException", oldName+" does not exist")
	}
	if oldName == newName {
		return map[string]any{}, nil
	}
	if _, taken := st.repos[newName]; taken {
		return nil, ccError("RepositoryNameExistsException", "repository "+newName+" already exists")
	}
	delete(st.repos, oldName)
	r.name, r.arn, r.modified = newName, ccRepoARN(newName), ccNow()
	st.repos[newName] = r
	for _, pr := range st.prs {
		for _, t := range pr.targets {
			if t.repo == oldName {
				t.repo = newName
			}
		}
	}
	return map[string]any{}, nil
}

func (st *ccStore) updateRepositoryDescription(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	r.description, r.modified = ccStr(in, "repositoryDescription"), ccNow()
	return map[string]any{}, nil
}

func (st *ccStore) updateRepositoryEncryptionKey(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	original := r.kmsKey
	r.kmsKey, r.modified = ccStr(in, "kmsKeyId"), ccNow()
	return map[string]any{"repositoryId": r.id, "kmsKeyId": r.kmsKey, "originalKmsKeyId": original}, nil
}

func (st *ccStore) updateDefaultBranch(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	name := ccStr(in, "defaultBranchName")
	if _, err := r.branch(name); err != nil {
		return nil, err
	}
	r.defaultBranch, r.modified = name, ccNow()
	return map[string]any{}, nil
}

// ---- branches ----

func (st *ccStore) createBranch(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	name := ccStr(in, "branchName")
	if _, ok := r.branches[name]; ok {
		return nil, ccError("BranchNameExistsException", "branch "+name+" already exists")
	}
	if len(r.branches) >= ccMaxBranchesPerRepo {
		return nil, ccError("MaximumBranchesExceededException", fmt.Sprintf("a repository can have at most %d branches", ccMaxBranchesPerRepo))
	}
	c, err := r.commit(ccStr(in, "commitId"))
	if err != nil {
		return nil, err
	}
	r.branches[name] = c.id
	if r.defaultBranch == "" {
		r.defaultBranch = name
	}
	r.modified = ccNow()
	return map[string]any{}, nil
}

func (st *ccStore) getBranch(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	name := ccStr(in, "branchName")
	id, err := r.branch(name)
	if err != nil {
		return nil, err
	}
	return map[string]any{"branch": map[string]any{"branchName": name, "commitId": id}}, nil
}

func (st *ccStore) listBranches(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(r.branches))
	for n := range r.branches {
		names = append(names, n)
	}
	sort.Strings(names)
	page, next, err := ccPage(names, in, "nextToken", "")
	if err != nil {
		return nil, err
	}
	return map[string]any{"branches": page, "nextToken": next}, nil
}

func (st *ccStore) deleteBranch(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	name := ccStr(in, "branchName")
	if name == r.defaultBranch {
		return nil, ccError("DefaultBranchCannotBeDeletedException", "the default branch can't be deleted")
	}
	id, ok := r.branches[name]
	if !ok {
		return map[string]any{}, nil
	}
	delete(r.branches, name)
	r.modified = ccNow()
	return map[string]any{"deletedBranch": map[string]any{"branchName": name, "commitId": id}}, nil
}

// ---- tags ----

func ccTags(add map[string]any, base map[string]string) (map[string]string, *ccErr) {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range add {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, ccError("InvalidSystemTagUsageException", "tag keys can't start with aws:")
		}
		s, _ := v.(string)
		out[k] = s
	}
	if len(out) > ccMaxTags {
		return nil, ccError("TooManyTagsException", fmt.Sprintf("a resource can have at most %d tags", ccMaxTags))
	}
	return out, nil
}

func (st *ccStore) repoByARN(arn string) (*ccRepo, *ccErr) {
	prefix := "arn:aws:codecommit:" + ccRegion + ":" + ccAccountID + ":"
	if !strings.HasPrefix(arn, prefix) {
		return nil, ccError("InvalidResourceArnException", "resource "+arn+" is not a CodeCommit repository ARN")
	}
	r, ok := st.repos[strings.TrimPrefix(arn, prefix)]
	if !ok {
		return nil, ccError("RepositoryDoesNotExistException", arn+" does not exist")
	}
	return r, nil
}

func (st *ccStore) tagResource(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repoByARN(ccStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	tags, err := ccTags(ccMap(in, "tags"), r.tags)
	if err != nil {
		return nil, err
	}
	r.tags = tags
	return map[string]any{}, nil
}

func (st *ccStore) untagResource(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repoByARN(ccStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	for _, k := range ccList(in, "tagKeys") {
		if key, _ := k.(string); strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, ccError("InvalidSystemTagUsageException", "tag keys can't start with aws:")
		} else {
			delete(r.tags, key)
		}
	}
	return map[string]any{}, nil
}

func (st *ccStore) listTagsForResource(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repoByARN(ccStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"tags": r.tags}, nil
}

// ---- triggers ----

func ccValidTriggers(triggers []any) *ccErr {
	if len(triggers) > ccMaxTriggers {
		return ccError("MaximumRepositoryTriggersExceededException", fmt.Sprintf("a repository can have at most %d triggers", ccMaxTriggers))
	}
	names := map[string]bool{}
	for _, t := range triggers {
		m, _ := t.(map[string]any)
		name := ccStr(m, "name")
		if names[name] {
			return ccError("InvalidRepositoryTriggerNameException", "trigger "+name+" is listed twice")
		}
		names[name] = true
		dest := ccStr(m, "destinationArn")
		if !strings.HasPrefix(dest, "arn:aws:sns:") && !strings.HasPrefix(dest, "arn:aws:lambda:") {
			return ccError("InvalidRepositoryTriggerDestinationArnException", "trigger destinations must be an SNS topic or Lambda function ARN")
		}
		events := ccList(m, "events")
		for _, e := range events {
			if e == "all" && len(events) > 1 {
				return ccError("InvalidRepositoryTriggerEventsException", "the all event can't be combined with other events")
			}
		}
	}
	return nil
}

func (st *ccStore) putRepositoryTriggers(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	triggers := ccList(in, "triggers")
	if err := ccValidTriggers(triggers); err != nil {
		return nil, err
	}
	r.triggers, r.triggerConfigID = triggers, ccUUID()
	return map[string]any{"configurationId": r.triggerConfigID}, nil
}

func (st *ccStore) getRepositoryTriggers(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	if r.triggerConfigID == "" {
		r.triggerConfigID = ccUUID()
	}
	triggers := r.triggers
	if triggers == nil {
		triggers = []any{}
	}
	return map[string]any{"configurationId": r.triggerConfigID, "triggers": triggers}, nil
}

// testRepositoryTriggers reports every valid trigger as delivered; nothing
// is actually sent.
func (st *ccStore) testRepositoryTriggers(in map[string]any) (map[string]any, *ccErr) {
	if _, err := st.repo(in); err != nil {
		return nil, err
	}
	triggers := ccList(in, "triggers")
	if err := ccValidTriggers(triggers); err != nil {
		return nil, err
	}
	ok := []any{}
	for _, t := range triggers {
		m, _ := t.(map[string]any)
		ok = append(ok, ccStr(m, "name"))
	}
	return map[string]any{"successfulExecutions": ok, "failedExecutions": []any{}}, nil
}
