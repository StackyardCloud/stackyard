package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ccMaxApprovalRules  = 25
	ccMaxRepoTemplates  = 25
	ccMaxEventsPerPR    = 1000
	ccMaxOpenPRsPerRepo = 1000
)

type ccPRTarget struct {
	repo, sourceRef, destRef string
	sourceCommit, destCommit string
	mergeBase                string
	merged                   bool
	mergeCommit, mergeOption string
}

type ccApprovalRule struct {
	id, name, content string
	created, modified time.Time
	templateID        string
	templateName      string
}

type ccPullRequest struct {
	id, title, description, status, token, revision string
	created, activity                               time.Time
	targets                                         []*ccPRTarget
	rules                                           []*ccApprovalRule
	approvals                                       map[string]string // user ARN -> APPROVE (revision-scoped)
	overridden                                      bool
	events                                          []map[string]any
}

type ccTemplate struct {
	id, name, description, content string
	created, modified              time.Time
}

func registerCodeCommitPullRequestHandlers() {
	for name, h := range map[string]ccHandler{
		"CreatePullRequest":            (*ccStore).createPullRequest,
		"GetPullRequest":               (*ccStore).getPullRequest,
		"ListPullRequests":             (*ccStore).listPullRequests,
		"DescribePullRequestEvents":    (*ccStore).describePullRequestEvents,
		"UpdatePullRequestTitle":       (*ccStore).updatePullRequestTitle,
		"UpdatePullRequestDescription": (*ccStore).updatePullRequestDescription,
		"UpdatePullRequestStatus":      (*ccStore).updatePullRequestStatus,
		"MergePullRequestByFastForward": func(st *ccStore, in map[string]any) (map[string]any, *ccErr) {
			return st.mergePullRequest(in, "FAST_FORWARD_MERGE")
		},
		"MergePullRequestBySquash": func(st *ccStore, in map[string]any) (map[string]any, *ccErr) {
			return st.mergePullRequest(in, "SQUASH_MERGE")
		},
		"MergePullRequestByThreeWay": func(st *ccStore, in map[string]any) (map[string]any, *ccErr) {
			return st.mergePullRequest(in, "THREE_WAY_MERGE")
		},
		"CreatePullRequestApprovalRule":                         (*ccStore).createPullRequestApprovalRule,
		"UpdatePullRequestApprovalRuleContent":                  (*ccStore).updatePullRequestApprovalRuleContent,
		"DeletePullRequestApprovalRule":                         (*ccStore).deletePullRequestApprovalRule,
		"UpdatePullRequestApprovalState":                        (*ccStore).updatePullRequestApprovalState,
		"GetPullRequestApprovalStates":                          (*ccStore).getPullRequestApprovalStates,
		"EvaluatePullRequestApprovalRules":                      (*ccStore).evaluatePullRequestApprovalRules,
		"OverridePullRequestApprovalRules":                      (*ccStore).overridePullRequestApprovalRules,
		"GetPullRequestOverrideState":                           (*ccStore).getPullRequestOverrideState,
		"CreateApprovalRuleTemplate":                            (*ccStore).createApprovalRuleTemplate,
		"GetApprovalRuleTemplate":                               (*ccStore).getApprovalRuleTemplate,
		"ListApprovalRuleTemplates":                             (*ccStore).listApprovalRuleTemplates,
		"DeleteApprovalRuleTemplate":                            (*ccStore).deleteApprovalRuleTemplate,
		"UpdateApprovalRuleTemplateName":                        (*ccStore).updateApprovalRuleTemplateName,
		"UpdateApprovalRuleTemplateDescription":                 (*ccStore).updateApprovalRuleTemplateDescription,
		"UpdateApprovalRuleTemplateContent":                     (*ccStore).updateApprovalRuleTemplateContent,
		"AssociateApprovalRuleTemplateWithRepository":           (*ccStore).associateApprovalRuleTemplate,
		"BatchAssociateApprovalRuleTemplateWithRepositories":    (*ccStore).batchAssociateApprovalRuleTemplate,
		"DisassociateApprovalRuleTemplateFromRepository":        (*ccStore).disassociateApprovalRuleTemplate,
		"BatchDisassociateApprovalRuleTemplateFromRepositories": (*ccStore).batchDisassociateApprovalRuleTemplate,
		"ListAssociatedApprovalRuleTemplatesForRepository":      (*ccStore).listAssociatedApprovalRuleTemplates,
		"ListRepositoriesForApprovalRuleTemplate":               (*ccStore).listRepositoriesForApprovalRuleTemplate,
	} {
		ccHandlers[name] = h
	}
}

func ccSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ---- approval rule content ----

type ccRuleContent struct {
	Version               string   `json:"Version"`
	DestinationReferences []string `json:"DestinationReferences"`
	Statements            []struct {
		Type                    string   `json:"Type"`
		NumberOfApprovalsNeeded int      `json:"NumberOfApprovalsNeeded"`
		ApprovalPoolMembers     []string `json:"ApprovalPoolMembers"`
	} `json:"Statements"`
}

func ccParseRule(content string, template bool) (*ccRuleContent, *ccErr) {
	code := "InvalidApprovalRuleContentException"
	if template {
		code = "InvalidApprovalRuleTemplateContentException"
	}
	var rc ccRuleContent
	if err := json.Unmarshal([]byte(content), &rc); err != nil || rc.Version != "2018-11-08" || len(rc.Statements) == 0 {
		return nil, ccError(code, `approval rule content must be JSON with Version "2018-11-08" and at least one statement`)
	}
	for _, st := range rc.Statements {
		if st.Type != "Approvers" || st.NumberOfApprovalsNeeded < 1 {
			return nil, ccError(code, `each statement needs Type "Approvers" and NumberOfApprovalsNeeded of at least 1`)
		}
	}
	return &rc, nil
}

func ccPoolMatch(pool []string, arn string) bool {
	if len(pool) == 0 {
		return true
	}
	for _, member := range pool {
		pattern := member
		if rest, ok := strings.CutPrefix(member, "CodeCommitApprovers:"); ok {
			account, name, _ := strings.Cut(rest, ":")
			pattern = "arn:aws:*:" + account + ":*" + name
		}
		re := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*") + "$"
		if ok, _ := regexp.MatchString(re, arn); ok {
			return true
		}
	}
	return false
}

func (pr *ccPullRequest) ruleSatisfied(rule *ccApprovalRule) bool {
	rc, err := ccParseRule(rule.content, false)
	if err != nil {
		return false
	}
	for _, st := range rc.Statements {
		n := 0
		for user, state := range pr.approvals {
			if state == "APPROVE" && ccPoolMatch(st.ApprovalPoolMembers, user) {
				n++
			}
		}
		if n < st.NumberOfApprovalsNeeded {
			return false
		}
	}
	return true
}

// ---- output ----

func (r *ccApprovalRule) out() map[string]any {
	out := map[string]any{
		"approvalRuleId": r.id, "approvalRuleName": r.name, "approvalRuleContent": r.content, "ruleContentSha256": ccSHA256(r.content),
		"creationDate": ccEpoch(r.created), "lastModifiedDate": ccEpoch(r.modified), "lastModifiedUser": ccCallerARN,
	}
	if r.templateID != "" {
		out["originApprovalRuleTemplate"] = map[string]any{"approvalRuleTemplateId": r.templateID, "approvalRuleTemplateName": r.templateName}
	}
	return out
}

func (t *ccPRTarget) out() map[string]any {
	meta := map[string]any{"isMerged": t.merged}
	if t.merged {
		meta["mergedBy"], meta["mergeCommitId"], meta["mergeOption"] = ccCallerARN, t.mergeCommit, t.mergeOption
	}
	return map[string]any{
		"repositoryName": t.repo, "sourceReference": t.sourceRef, "destinationReference": t.destRef,
		"sourceCommit": t.sourceCommit, "destinationCommit": t.destCommit, "mergeBase": t.mergeBase, "mergeMetadata": meta,
	}
}

func (pr *ccPullRequest) out() map[string]any {
	targets, rules := []any{}, []any{}
	for _, t := range pr.targets {
		targets = append(targets, t.out())
	}
	for _, r := range pr.rules {
		rules = append(rules, r.out())
	}
	out := map[string]any{
		"pullRequestId": pr.id, "title": pr.title, "pullRequestStatus": pr.status, "authorArn": ccCallerARN,
		"creationDate": ccEpoch(pr.created), "lastActivityDate": ccEpoch(pr.activity), "revisionId": pr.revision,
		"pullRequestTargets": targets, "approvalRules": rules,
	}
	if pr.description != "" {
		out["description"] = pr.description
	}
	if pr.token != "" {
		out["clientRequestToken"] = pr.token
	}
	return out
}

func (t *ccTemplate) out() map[string]any {
	return map[string]any{
		"approvalRuleTemplateId": t.id, "approvalRuleTemplateName": t.name, "approvalRuleTemplateDescription": t.description,
		"approvalRuleTemplateContent": t.content, "ruleContentSha256": ccSHA256(t.content),
		"creationDate": ccEpoch(t.created), "lastModifiedDate": ccEpoch(t.modified), "lastModifiedUser": ccCallerARN,
	}
}

func (pr *ccPullRequest) event(kind string, meta map[string]any) {
	if len(pr.events) >= ccMaxEventsPerPR {
		return
	}
	pr.activity = ccNow()
	e := map[string]any{"pullRequestId": pr.id, "pullRequestEventType": kind, "actorArn": ccCallerARN, "eventDate": ccEpoch(pr.activity)}
	for k, v := range meta {
		e[k] = v
	}
	pr.events = append(pr.events, e)
}

// ---- lookup ----

func (st *ccStore) pullRequest(in map[string]any) (*ccPullRequest, *ccErr) {
	id := ccStr(in, "pullRequestId")
	if _, err := strconv.ParseInt(id, 10, 32); err != nil {
		return nil, ccError("InvalidPullRequestIdException", "pull request id "+id+" is not valid")
	}
	pr, ok := st.prs[id]
	if !ok {
		return nil, ccError("PullRequestDoesNotExistException", "pull request "+id+" does not exist")
	}
	st.refresh(pr)
	return pr, nil
}

// refresh notices a source branch that moved since the pull request last
// looked: a new revision, with approvals cleared, as the service does.
func (st *ccStore) refresh(pr *ccPullRequest) {
	if pr.status != "OPEN" {
		return
	}
	for _, t := range pr.targets {
		r, ok := st.repos[t.repo]
		if !ok {
			continue
		}
		tip, ok := r.branches[ccBranchName(t.sourceRef)]
		if !ok || tip == t.sourceCommit {
			continue
		}
		before := t.sourceCommit
		t.sourceCommit = tip
		if d, ok := r.branches[ccBranchName(t.destRef)]; ok {
			t.destCommit = d
			if base := r.mergeBase(r.commits[tip], r.commits[d]); base != nil {
				t.mergeBase = base.id
			}
		}
		pr.revision, pr.approvals, pr.overridden = ccRandomSHA(), map[string]string{}, false
		pr.event("PULL_REQUEST_SOURCE_REFERENCE_UPDATED", map[string]any{"pullRequestSourceReferenceUpdatedEventMetadata": map[string]any{
			"repositoryName": t.repo, "beforeCommitId": before, "afterCommitId": tip, "mergeBase": t.mergeBase,
		}})
	}
}

func ccBranchName(ref string) string { return strings.TrimPrefix(ref, "refs/heads/") }

func (pr *ccPullRequest) checkRevision(in map[string]any) *ccErr {
	rev := ccStr(in, "revisionId")
	if rev != pr.revision {
		return ccError("RevisionNotCurrentException", "revision "+rev+" is not the current revision of the pull request")
	}
	return nil
}

func (pr *ccPullRequest) checkOpen() *ccErr {
	if pr.status != "OPEN" {
		return ccError("PullRequestAlreadyClosedException", "pull request "+pr.id+" is closed")
	}
	return nil
}

// ---- pull requests ----

func (st *ccStore) createPullRequest(in map[string]any) (map[string]any, *ccErr) {
	token := ccStr(in, "clientRequestToken")
	if id, ok := st.prTokens[token]; ok && token != "" {
		pr := st.prs[id]
		if pr.title != ccStr(in, "title") {
			return nil, ccError("IdempotencyParameterMismatchException", "the client request token was already used with different parameters")
		}
		return map[string]any{"pullRequest": pr.out()}, nil
	}
	if len(st.prs) >= ccMaxPullRequests {
		return nil, ccError("MaximumOpenPullRequestsExceededException", "too many pull requests")
	}
	raw := ccList(in, "targets")
	if len(raw) == 0 {
		return nil, ccError("TargetsRequiredException", "targets is required")
	}
	if len(raw) > 1 {
		return nil, ccError("MultipleRepositoriesInPullRequestException", "a pull request can target one repository")
	}
	tm, _ := raw[0].(map[string]any)
	r, err := st.repo(tm)
	if err != nil {
		return nil, err
	}
	srcRef, dstRef := ccStr(tm, "sourceReference"), ccStr(tm, "destinationReference")
	if srcRef == "" {
		return nil, ccError("ReferenceNameRequiredException", "sourceReference is required")
	}
	if dstRef == "" {
		dstRef = r.defaultBranch
	}
	src, ok := r.branches[ccBranchName(srcRef)]
	if !ok {
		return nil, ccError("ReferenceDoesNotExistException", "reference "+srcRef+" does not exist")
	}
	dst, ok := r.branches[ccBranchName(dstRef)]
	if !ok {
		return nil, ccError("ReferenceDoesNotExistException", "reference "+dstRef+" does not exist")
	}
	if ccBranchName(srcRef) == ccBranchName(dstRef) {
		return nil, ccError("SourceAndDestinationAreSameException", "the source and destination references are the same")
	}
	open := 0
	for _, pr := range st.prs {
		if pr.status == "OPEN" && pr.targets[0].repo == r.name {
			open++
		}
	}
	if open >= ccMaxOpenPRsPerRepo {
		return nil, ccError("MaximumOpenPullRequestsExceededException", "the repository has too many open pull requests")
	}
	base := ""
	if b := r.mergeBase(r.commits[src], r.commits[dst]); b != nil {
		base = b.id
	}
	now := ccNow()
	pr := &ccPullRequest{
		id: strconv.Itoa(st.nextPR), title: ccStr(in, "title"), description: ccStr(in, "description"), status: "OPEN", token: token,
		revision: ccRandomSHA(), created: now, activity: now, approvals: map[string]string{},
		targets: []*ccPRTarget{{repo: r.name, sourceRef: srcRef, destRef: dstRef, sourceCommit: src, destCommit: dst, mergeBase: base}},
	}
	st.nextPR++
	// Rules come from templates associated with the repository whose
	// destination references (if any) include the target branch.
	names := make([]string, 0, len(r.templates))
	for n := range r.templates {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t, ok := st.templates[n]
		if !ok {
			continue
		}
		rc, err := ccParseRule(t.content, true)
		if err != nil {
			continue
		}
		applies := len(rc.DestinationReferences) == 0
		for _, ref := range rc.DestinationReferences {
			if ccBranchName(ref) == ccBranchName(dstRef) {
				applies = true
			}
		}
		if applies {
			pr.rules = append(pr.rules, &ccApprovalRule{id: ccUUID(), name: t.name, content: t.content, created: now, modified: now, templateID: t.id, templateName: t.name})
		}
	}
	st.prs[pr.id] = pr
	st.prOrder = append(st.prOrder, pr.id)
	if token != "" {
		st.prTokens[token] = pr.id
	}
	pr.event("PULL_REQUEST_CREATED", map[string]any{"pullRequestCreatedEventMetadata": map[string]any{
		"repositoryName": r.name, "sourceCommitId": src, "destinationCommitId": dst, "mergeBase": base,
	}})
	return map[string]any{"pullRequest": pr.out()}, nil
}

func (st *ccStore) getPullRequest(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"pullRequest": pr.out()}, nil
}

func (st *ccStore) listPullRequests(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	author, status := ccStr(in, "authorArn"), ccStr(in, "pullRequestStatus")
	if author != "" && !strings.HasPrefix(author, "arn:aws:") {
		return nil, ccError("InvalidAuthorArnException", "authorArn is not a valid ARN")
	}
	ids := []string{}
	for i := len(st.prOrder) - 1; i >= 0; i-- { // newest first
		pr := st.prs[st.prOrder[i]]
		if pr == nil || pr.targets[0].repo != r.name {
			continue
		}
		st.refresh(pr)
		if (status == "" || pr.status == status) && (author == "" || author == ccCallerARN) {
			ids = append(ids, pr.id)
		}
	}
	page, next, err := ccPage(ids, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"pullRequestIds": page, "nextToken": next}, nil
}

func (st *ccStore) describePullRequestEvents(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	kind, actor := ccStr(in, "pullRequestEventType"), ccStr(in, "actorArn")
	if actor != "" && !strings.HasPrefix(actor, "arn:aws:") {
		return nil, ccError("InvalidActorArnException", "actorArn is not a valid ARN")
	}
	events := []map[string]any{}
	for _, e := range pr.events {
		if (kind == "" || e["pullRequestEventType"] == kind) && (actor == "" || actor == ccCallerARN) {
			events = append(events, e)
		}
	}
	page, next, err := ccPage(events, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"pullRequestEvents": page, "nextToken": next}, nil
}

func (st *ccStore) updatePullRequestTitle(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	pr.title, pr.activity = ccStr(in, "title"), ccNow()
	return map[string]any{"pullRequest": pr.out()}, nil
}

func (st *ccStore) updatePullRequestDescription(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	pr.description, pr.activity = ccStr(in, "description"), ccNow()
	return map[string]any{"pullRequest": pr.out()}, nil
}

func (st *ccStore) updatePullRequestStatus(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	status := ccStr(in, "pullRequestStatus")
	if pr.status == "CLOSED" && status == "OPEN" {
		return nil, ccError("InvalidPullRequestStatusUpdateException", "a closed pull request can't be reopened")
	}
	if pr.status != status {
		pr.status = status
		pr.event("PULL_REQUEST_STATUS_CHANGED", map[string]any{"pullRequestStatusChangedEventMetadata": map[string]any{"pullRequestStatus": status}})
	}
	return map[string]any{"pullRequest": pr.out()}, nil
}

func (st *ccStore) mergePullRequest(in map[string]any, option string) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	t := pr.targets[0]
	if name := ccStr(in, "repositoryName"); name != t.repo {
		return nil, ccError("RepositoryNotAssociatedWithPullRequestException", "repository "+name+" is not the pull request's repository")
	}
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	if want := ccStr(in, "sourceCommitId"); want != "" && want != t.sourceCommit {
		return nil, ccError("TipOfSourceReferenceIsDifferentException", "the source branch has moved since commit "+want)
	}
	if !pr.overridden {
		for _, rule := range pr.rules {
			if !pr.ruleSatisfied(rule) {
				return nil, ccError("PullRequestApprovalRulesNotSatisfiedException", "approval rule "+rule.name+" is not satisfied")
			}
		}
	}
	src, dst := r.commits[t.sourceCommit], r.commits[t.destCommit]
	if tip, ok := r.branches[ccBranchName(t.destRef)]; ok {
		dst = r.commits[tip]
	}
	var merged *ccCommit
	if option == "FAST_FORWARD_MERGE" {
		if !r.isAncestor(dst, src) {
			return nil, ccError("ManualMergeRequiredException", "the destination can't be fast-forwarded to the source")
		}
		merged = src
	} else {
		merged, err = r.mergeCommit(src, dst, option, in)
		if err != nil {
			return nil, err
		}
	}
	r.branches[ccBranchName(t.destRef)] = merged.id
	r.modified = ccNow()
	t.destCommit, t.merged, t.mergeCommit, t.mergeOption = dst.id, true, merged.id, option
	pr.status = "CLOSED"
	pr.event("PULL_REQUEST_MERGE_STATE_CHANGED", map[string]any{"pullRequestMergedStateChangedEventMetadata": map[string]any{
		"repositoryName": t.repo, "destinationReference": t.destRef, "mergeMetadata": t.out()["mergeMetadata"],
	}})
	return map[string]any{"pullRequest": pr.out()}, nil
}

// ---- approval rules on a pull request ----

func (pr *ccPullRequest) rule(name string) (int, *ccApprovalRule) {
	for i, r := range pr.rules {
		if r.name == name {
			return i, r
		}
	}
	return -1, nil
}

func (st *ccStore) createPullRequestApprovalRule(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	name, content := ccStr(in, "approvalRuleName"), ccStr(in, "approvalRuleContent")
	if _, existing := pr.rule(name); existing != nil {
		return nil, ccError("ApprovalRuleNameAlreadyExistsException", "approval rule "+name+" already exists")
	}
	if len(pr.rules) >= ccMaxApprovalRules {
		return nil, ccError("NumberOfRulesExceededException", fmt.Sprintf("a pull request can have at most %d approval rules", ccMaxApprovalRules))
	}
	if _, err := ccParseRule(content, false); err != nil {
		return nil, err
	}
	now := ccNow()
	rule := &ccApprovalRule{id: ccUUID(), name: name, content: content, created: now, modified: now}
	pr.rules = append(pr.rules, rule)
	pr.event("PULL_REQUEST_APPROVAL_RULE_CREATED", map[string]any{"approvalRuleEventMetadata": map[string]any{"approvalRuleName": name, "approvalRuleId": rule.id, "approvalRuleContent": content}})
	return map[string]any{"approvalRule": rule.out()}, nil
}

func (st *ccStore) updatePullRequestApprovalRuleContent(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	_, rule := pr.rule(ccStr(in, "approvalRuleName"))
	if rule == nil {
		return nil, ccError("ApprovalRuleDoesNotExistException", "approval rule "+ccStr(in, "approvalRuleName")+" does not exist")
	}
	if rule.templateID != "" {
		return nil, ccError("CannotModifyApprovalRuleFromTemplateException", "rules created from a template can't be changed")
	}
	if sha := ccStr(in, "existingRuleContentSha256"); sha != "" && sha != ccSHA256(rule.content) {
		return nil, ccError("InvalidRuleContentSha256Exception", "existingRuleContentSha256 doesn't match the current content")
	}
	content := ccStr(in, "newRuleContent")
	if _, err := ccParseRule(content, false); err != nil {
		return nil, err
	}
	rule.content, rule.modified = content, ccNow()
	pr.event("PULL_REQUEST_APPROVAL_RULE_UPDATED", map[string]any{"approvalRuleEventMetadata": map[string]any{"approvalRuleName": rule.name, "approvalRuleId": rule.id, "approvalRuleContent": content}})
	return map[string]any{"approvalRule": rule.out()}, nil
}

func (st *ccStore) deletePullRequestApprovalRule(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	i, rule := pr.rule(ccStr(in, "approvalRuleName"))
	if rule == nil {
		return map[string]any{"approvalRuleId": ""}, nil
	}
	if rule.templateID != "" {
		return nil, ccError("CannotDeleteApprovalRuleFromTemplateException", "rules created from a template can't be deleted")
	}
	pr.rules = append(pr.rules[:i], pr.rules[i+1:]...)
	pr.event("PULL_REQUEST_APPROVAL_RULE_DELETED", map[string]any{"approvalRuleEventMetadata": map[string]any{"approvalRuleName": rule.name, "approvalRuleId": rule.id}})
	return map[string]any{"approvalRuleId": rule.id}, nil
}

// updatePullRequestApprovalState records the caller's approval.
// ponytail: Stackyard has one identity, so the author may approve their own
// pull request (the service would refuse with
// PullRequestCannotBeApprovedByAuthorException).
func (st *ccStore) updatePullRequestApprovalState(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	if err := pr.checkRevision(in); err != nil {
		return nil, err
	}
	state := ccStr(in, "approvalState")
	if state == "APPROVE" {
		pr.approvals[ccCallerARN] = "APPROVE"
	} else {
		delete(pr.approvals, ccCallerARN)
	}
	pr.event("PULL_REQUEST_APPROVAL_STATE_CHANGED", map[string]any{"approvalStateChangedEventMetadata": map[string]any{"revisionId": pr.revision, "approvalStatus": state}})
	return map[string]any{}, nil
}

func (st *ccStore) getPullRequestApprovalStates(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkRevision(in); err != nil {
		return nil, err
	}
	approvals := []any{}
	for user, state := range pr.approvals {
		approvals = append(approvals, map[string]any{"userArn": user, "approvalState": state})
	}
	return map[string]any{"approvals": approvals}, nil
}

func (st *ccStore) evaluatePullRequestApprovalRules(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkRevision(in); err != nil {
		return nil, err
	}
	sat, unsat := []any{}, []any{}
	for _, rule := range pr.rules {
		if pr.ruleSatisfied(rule) {
			sat = append(sat, rule.name)
		} else {
			unsat = append(unsat, rule.name)
		}
	}
	return map[string]any{"evaluation": map[string]any{
		"revisionId": pr.revision, "approved": pr.overridden || len(unsat) == 0, "overridden": pr.overridden,
		"approvalRulesSatisfied": sat, "approvalRulesNotSatisfied": unsat,
	}}, nil
}

func (st *ccStore) overridePullRequestApprovalRules(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkOpen(); err != nil {
		return nil, err
	}
	if err := pr.checkRevision(in); err != nil {
		return nil, err
	}
	status := ccStr(in, "overrideStatus")
	if (status == "OVERRIDE") == pr.overridden {
		return nil, ccError("OverrideAlreadySetException", "the override status is already "+status)
	}
	pr.overridden = status == "OVERRIDE"
	pr.event("PULL_REQUEST_APPROVAL_RULE_OVERRIDDEN", map[string]any{"approvalRuleOverriddenEventMetadata": map[string]any{"revisionId": pr.revision, "overrideStatus": status}})
	return map[string]any{}, nil
}

func (st *ccStore) getPullRequestOverrideState(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if err := pr.checkRevision(in); err != nil {
		return nil, err
	}
	out := map[string]any{"overridden": pr.overridden}
	if pr.overridden {
		out["overrider"] = ccCallerARN
	}
	return out, nil
}

// ---- approval rule templates ----

func (st *ccStore) template(name string) (*ccTemplate, *ccErr) {
	t, ok := st.templates[name]
	if !ok {
		return nil, ccError("ApprovalRuleTemplateDoesNotExistException", "approval rule template "+name+" does not exist")
	}
	return t, nil
}

func (st *ccStore) createApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	name, content := ccStr(in, "approvalRuleTemplateName"), ccStr(in, "approvalRuleTemplateContent")
	if _, ok := st.templates[name]; ok {
		return nil, ccError("ApprovalRuleTemplateNameAlreadyExistsException", "approval rule template "+name+" already exists")
	}
	if len(st.templates) >= ccMaxTemplates {
		return nil, ccError("NumberOfRuleTemplatesExceededException", fmt.Sprintf("an account can have at most %d approval rule templates", ccMaxTemplates))
	}
	if _, err := ccParseRule(content, true); err != nil {
		return nil, err
	}
	now := ccNow()
	t := &ccTemplate{id: ccUUID(), name: name, description: ccStr(in, "approvalRuleTemplateDescription"), content: content, created: now, modified: now}
	st.templates[name] = t
	return map[string]any{"approvalRuleTemplate": t.out()}, nil
}

func (st *ccStore) getApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	t, err := st.template(ccStr(in, "approvalRuleTemplateName"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"approvalRuleTemplate": t.out()}, nil
}

func (st *ccStore) listApprovalRuleTemplates(in map[string]any) (map[string]any, *ccErr) {
	names := make([]string, 0, len(st.templates))
	for n := range st.templates {
		names = append(names, n)
	}
	sort.Strings(names)
	page, next, err := ccPage(names, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"approvalRuleTemplateNames": page, "nextToken": next}, nil
}

func (st *ccStore) templateRepos(name string) []string {
	var out []string
	for _, r := range st.repos {
		if r.templates[name] {
			out = append(out, r.name)
		}
	}
	sort.Strings(out)
	return out
}

func (st *ccStore) deleteApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	name := ccStr(in, "approvalRuleTemplateName")
	t, ok := st.templates[name]
	if !ok {
		return map[string]any{"approvalRuleTemplateId": ""}, nil
	}
	if len(st.templateRepos(name)) > 0 {
		return nil, ccError("ApprovalRuleTemplateInUseException", "approval rule template "+name+" is associated with repositories")
	}
	delete(st.templates, name)
	return map[string]any{"approvalRuleTemplateId": t.id}, nil
}

func (st *ccStore) updateApprovalRuleTemplateName(in map[string]any) (map[string]any, *ccErr) {
	oldName, newName := ccStr(in, "oldApprovalRuleTemplateName"), ccStr(in, "newApprovalRuleTemplateName")
	t, err := st.template(oldName)
	if err != nil {
		return nil, err
	}
	if _, taken := st.templates[newName]; taken && newName != oldName {
		return nil, ccError("ApprovalRuleTemplateNameAlreadyExistsException", "approval rule template "+newName+" already exists")
	}
	delete(st.templates, oldName)
	t.name, t.modified = newName, ccNow()
	st.templates[newName] = t
	for _, r := range st.repos {
		if r.templates[oldName] {
			delete(r.templates, oldName)
			r.templates[newName] = true
		}
	}
	return map[string]any{"approvalRuleTemplate": t.out()}, nil
}

func (st *ccStore) updateApprovalRuleTemplateDescription(in map[string]any) (map[string]any, *ccErr) {
	t, err := st.template(ccStr(in, "approvalRuleTemplateName"))
	if err != nil {
		return nil, err
	}
	t.description, t.modified = ccStr(in, "approvalRuleTemplateDescription"), ccNow()
	return map[string]any{"approvalRuleTemplate": t.out()}, nil
}

func (st *ccStore) updateApprovalRuleTemplateContent(in map[string]any) (map[string]any, *ccErr) {
	t, err := st.template(ccStr(in, "approvalRuleTemplateName"))
	if err != nil {
		return nil, err
	}
	if sha := ccStr(in, "existingRuleContentSha256"); sha != "" && sha != ccSHA256(t.content) {
		return nil, ccError("InvalidRuleContentSha256Exception", "existingRuleContentSha256 doesn't match the current content")
	}
	content := ccStr(in, "newRuleContent")
	if _, err := ccParseRule(content, true); err != nil {
		return nil, err
	}
	t.content, t.modified = content, ccNow()
	return map[string]any{"approvalRuleTemplate": t.out()}, nil
}

func (st *ccStore) associate(name, repo string) *ccErr {
	if _, err := st.template(name); err != nil {
		return err
	}
	r, ok := st.repos[repo]
	if !ok {
		return ccError("RepositoryDoesNotExistException", repo+" does not exist")
	}
	if !r.templates[name] && len(r.templates) >= ccMaxRepoTemplates {
		return ccError("MaximumRuleTemplatesAssociatedWithRepositoryException", fmt.Sprintf("a repository can have at most %d approval rule templates", ccMaxRepoTemplates))
	}
	r.templates[name] = true
	return nil
}

func (st *ccStore) associateApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	if err := st.associate(ccStr(in, "approvalRuleTemplateName"), ccStr(in, "repositoryName")); err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (st *ccStore) batchTemplateRepos(in map[string]any, apply func(name, repo string) *ccErr, okKey string) (map[string]any, *ccErr) {
	name := ccStr(in, "approvalRuleTemplateName")
	if _, err := st.template(name); err != nil {
		return nil, err
	}
	repos := ccList(in, "repositoryNames")
	if len(repos) > 100 {
		return nil, ccError("MaximumRepositoryNamesExceededException", "at most 100 repository names can be given")
	}
	done, errs := []any{}, []any{}
	for _, raw := range repos {
		repo, _ := raw.(string)
		if err := apply(name, repo); err != nil {
			errs = append(errs, map[string]any{"repositoryName": repo, "errorCode": err.code, "errorMessage": err.msg})
		} else {
			done = append(done, repo)
		}
	}
	return map[string]any{okKey: done, "errors": errs}, nil
}

func (st *ccStore) batchAssociateApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	return st.batchTemplateRepos(in, st.associate, "associatedRepositoryNames")
}

func (st *ccStore) disassociate(name, repo string) *ccErr {
	if _, err := st.template(name); err != nil {
		return err
	}
	r, ok := st.repos[repo]
	if !ok {
		return ccError("RepositoryDoesNotExistException", repo+" does not exist")
	}
	delete(r.templates, name)
	return nil
}

func (st *ccStore) disassociateApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	if err := st.disassociate(ccStr(in, "approvalRuleTemplateName"), ccStr(in, "repositoryName")); err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (st *ccStore) batchDisassociateApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	return st.batchTemplateRepos(in, st.disassociate, "disassociatedRepositoryNames")
}

func (st *ccStore) listAssociatedApprovalRuleTemplates(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(r.templates))
	for n := range r.templates {
		names = append(names, n)
	}
	sort.Strings(names)
	page, next, err := ccPage(names, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"approvalRuleTemplateNames": page, "nextToken": next}, nil
}

func (st *ccStore) listRepositoriesForApprovalRuleTemplate(in map[string]any) (map[string]any, *ccErr) {
	name := ccStr(in, "approvalRuleTemplateName")
	if _, err := st.template(name); err != nil {
		return nil, err
	}
	page, next, err := ccPage(st.templateRepos(name), in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"repositoryNames": page, "nextToken": next}, nil
}
