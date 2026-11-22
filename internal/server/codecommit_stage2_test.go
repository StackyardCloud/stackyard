package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const ccRuleOneApproval = `{\"Version\":\"2018-11-08\",\"Statements\":[{\"Type\":\"Approvers\",\"NumberOfApprovalsNeeded\":1}]}`

// ccBranchCommit commits path=content on branch (created from base when new).
func ccBranchCommit(t *testing.T, ts *httptest.Server, repo, branch, base, path, content string) string {
	t.Helper()
	call := func(op, body string) map[string]any { return ccOK(t, ts, op, body) }
	parent := base
	if status, got := ccRequest(t, ts, "GetBranch", fmt.Sprintf(`{"repositoryName":%q,"branchName":%q}`, repo, branch)); status == http.StatusOK {
		parent = got["branch"].(map[string]any)["commitId"].(string)
	} else {
		call("CreateBranch", fmt.Sprintf(`{"repositoryName":%q,"branchName":%q,"commitId":%q}`, repo, branch, base))
	}
	return call("PutFile", fmt.Sprintf(`{"repositoryName":%q,"branchName":%q,"parentCommitId":%q,"filePath":%q,"fileContent":%q}`, repo, branch, parent, path, ccB64s(content)))["commitId"].(string)
}

func TestCodeCommitStage2MergesAndConflicts(t *testing.T) {
	ts := newCodeCommitTestServer(t)
	call := func(op, body string) map[string]any { return ccOK(t, ts, op, body) }
	first := ccSeedRepo(t, ts, call, "m")
	// Three-line file so different lines can change on each side.
	base := call("PutFile", fmt.Sprintf(`{"repositoryName":"m","branchName":"main","parentCommitId":%q,"filePath":"f.txt","fileContent":%q}`, first, ccB64s("a\nb\nc\n")))["commitId"].(string)

	// Fast-forward: feature descends from main.
	ccBranchCommit(t, ts, "m", "ff", base, "new.txt", "n\n")
	ffMain := call("MergeBranchesByFastForward", `{"repositoryName":"m","sourceCommitSpecifier":"ff","destinationCommitSpecifier":"main"}`)
	if got := call("GetBranch", `{"repositoryName":"m","branchName":"main"}`)["branch"].(map[string]any)["commitId"]; got != ffMain["commitId"] {
		t.Fatalf("fast-forward didn't move main: %v vs %v", got, ffMain)
	}
	mainTip := ffMain["commitId"].(string)

	// Different lines on each side merge cleanly (line level).
	ccBranchCommit(t, ts, "m", "left", mainTip, "f.txt", "A\nb\nc\n")
	ccBranchCommit(t, ts, "m", "right", mainTip, "f.txt", "a\nb\nC\n")
	opts := call("GetMergeOptions", `{"repositoryName":"m","sourceCommitSpecifier":"left","destinationCommitSpecifier":"right"}`)["mergeOptions"].([]any)
	if len(opts) != 2 {
		t.Fatalf("merge options: %v", opts)
	}
	if code := ccFail(t, ts, "MergeBranchesByFastForward", `{"repositoryName":"m","sourceCommitSpecifier":"left","destinationCommitSpecifier":"right"}`); code != "ManualMergeRequiredException" {
		t.Fatalf("diverged fast-forward: %s", code)
	}
	merged := call("MergeBranchesByThreeWay", `{"repositoryName":"m","sourceCommitSpecifier":"left","destinationCommitSpecifier":"right","targetBranch":"right"}`)
	got := call("GetFile", `{"repositoryName":"m","filePath":"f.txt","commitSpecifier":"right"}`)["fileContent"]
	if got != ccB64s("A\nb\nC\n") {
		t.Fatalf("line merge: %v", got)
	}
	if parents := call("GetCommit", fmt.Sprintf(`{"repositoryName":"m","commitId":%q}`, merged["commitId"]))["commit"].(map[string]any)["parents"].([]any); len(parents) != 2 {
		t.Fatalf("three-way parents: %v", parents)
	}

	// The same line on both sides conflicts.
	ccBranchCommit(t, ts, "m", "x", mainTip, "f.txt", "X\nb\nc\n")
	ccBranchCommit(t, ts, "m", "y", mainTip, "f.txt", "Y\nb\nc\n")
	conf := call("GetMergeConflicts", `{"repositoryName":"m","sourceCommitSpecifier":"x","destinationCommitSpecifier":"y","mergeOption":"THREE_WAY_MERGE"}`)
	if conf["mergeable"] != false || len(conf["conflictMetadataList"].([]any)) != 1 {
		t.Fatalf("conflicts: %v", conf)
	}
	desc := call("DescribeMergeConflicts", `{"repositoryName":"m","sourceCommitSpecifier":"x","destinationCommitSpecifier":"y","mergeOption":"THREE_WAY_MERGE","filePath":"f.txt"}`)
	hunk := desc["mergeHunks"].([]any)[0].(map[string]any)
	if hunk["source"].(map[string]any)["hunkContent"] != "X\n" || hunk["destination"].(map[string]any)["hunkContent"] != "Y\n" {
		t.Fatalf("merge hunk: %v", hunk)
	}
	batch := call("BatchDescribeMergeConflicts", `{"repositoryName":"m","sourceCommitSpecifier":"x","destinationCommitSpecifier":"y","mergeOption":"SQUASH_MERGE"}`)
	if len(batch["conflicts"].([]any)) != 1 {
		t.Fatalf("batch conflicts: %v", batch)
	}
	if code := ccFail(t, ts, "MergeBranchesBySquash", `{"repositoryName":"m","sourceCommitSpecifier":"x","destinationCommitSpecifier":"y"}`); code != "ManualMergeRequiredException" {
		t.Fatalf("unresolved conflict: %s", code)
	}
	call("MergeBranchesBySquash", `{"repositoryName":"m","sourceCommitSpecifier":"x","destinationCommitSpecifier":"y","targetBranch":"y",
		"conflictResolution":{"replaceContents":[{"filePath":"f.txt","replacementType":"USE_NEW_CONTENT","content":"`+ccB64s("XY\nb\nc\n")+`"}]}}`)
	if got := call("GetFile", `{"repositoryName":"m","filePath":"f.txt","commitSpecifier":"y"}`)["fileContent"]; got != ccB64s("XY\nb\nc\n") {
		t.Fatalf("resolved content: %v", got)
	}
	// ACCEPT_SOURCE resolves without content; FILE_LEVEL treats any shared file as a conflict.
	call("CreateUnreferencedMergeCommit", `{"repositoryName":"m","sourceCommitSpecifier":"x","destinationCommitSpecifier":"main","mergeOption":"SQUASH_MERGE","conflictResolutionStrategy":"ACCEPT_SOURCE"}`)
	fileLevel := call("GetMergeConflicts", `{"repositoryName":"m","sourceCommitSpecifier":"left","destinationCommitSpecifier":"x","mergeOption":"THREE_WAY_MERGE","conflictDetailLevel":"FILE_LEVEL"}`)
	if fileLevel["mergeable"] != false {
		t.Fatalf("FILE_LEVEL should conflict: %v", fileLevel)
	}
}

func TestCodeCommitStage2PullRequestsApprovalsAndComments(t *testing.T) {
	ts := newCodeCommitTestServer(t)
	call := func(op, body string) map[string]any { return ccOK(t, ts, op, body) }
	first := ccSeedRepo(t, ts, call, "p")

	// A template that applies to main is copied onto new pull requests.
	tmpl := call("CreateApprovalRuleTemplate", `{"approvalRuleTemplateName":"one","approvalRuleTemplateContent":"{\"Version\":\"2018-11-08\",\"DestinationReferences\":[\"refs/heads/main\"],\"Statements\":[{\"Type\":\"Approvers\",\"NumberOfApprovalsNeeded\":1}]}"}`)["approvalRuleTemplate"].(map[string]any)
	if code := ccFail(t, ts, "CreateApprovalRuleTemplate", `{"approvalRuleTemplateName":"bad","approvalRuleTemplateContent":"{}"}`); code != "InvalidApprovalRuleTemplateContentException" {
		t.Fatalf("bad template content: %s", code)
	}
	call("AssociateApprovalRuleTemplateWithRepository", `{"approvalRuleTemplateName":"one","repositoryName":"p"}`)
	if code := ccFail(t, ts, "DeleteApprovalRuleTemplate", `{"approvalRuleTemplateName":"one"}`); code != "ApprovalRuleTemplateInUseException" {
		t.Fatalf("delete template in use: %s", code)
	}
	src := ccBranchCommit(t, ts, "p", "feature", first, "f.txt", "feature\n")
	pr := call("CreatePullRequest", `{"title":"Add f","clientRequestToken":"tok-1","targets":[{"repositoryName":"p","sourceReference":"feature","destinationReference":"main"}]}`)["pullRequest"].(map[string]any)
	id, rev := pr["pullRequestId"].(string), pr["revisionId"].(string)
	rules := pr["approvalRules"].([]any)
	if len(rules) != 1 || rules[0].(map[string]any)["originApprovalRuleTemplate"].(map[string]any)["approvalRuleTemplateId"] != tmpl["approvalRuleTemplateId"] {
		t.Fatalf("template rule: %v", rules)
	}
	again := call("CreatePullRequest", `{"title":"Add f","clientRequestToken":"tok-1","targets":[{"repositoryName":"p","sourceReference":"feature","destinationReference":"main"}]}`)["pullRequest"].(map[string]any)
	if again["pullRequestId"] != id {
		t.Fatalf("idempotent create gave %v", again["pullRequestId"])
	}
	if code := ccFail(t, ts, "CreatePullRequest", `{"title":"same","targets":[{"repositoryName":"p","sourceReference":"main","destinationReference":"main"}]}`); code != "SourceAndDestinationAreSameException" {
		t.Fatalf("same refs: %s", code)
	}

	// Rules block the merge until approved (or overridden).
	if code := ccFail(t, ts, "MergePullRequestByFastForward", `{"pullRequestId":"`+id+`","repositoryName":"p"}`); code != "PullRequestApprovalRulesNotSatisfiedException" {
		t.Fatalf("unapproved merge: %s", code)
	}
	if code := ccFail(t, ts, "UpdatePullRequestApprovalState", `{"pullRequestId":"`+id+`","revisionId":"stale","approvalState":"APPROVE"}`); code != "RevisionNotCurrentException" {
		t.Fatalf("stale revision: %s", code)
	}
	call("UpdatePullRequestApprovalState", `{"pullRequestId":"`+id+`","revisionId":"`+rev+`","approvalState":"APPROVE"}`)
	eval := call("EvaluatePullRequestApprovalRules", `{"pullRequestId":"`+id+`","revisionId":"`+rev+`"}`)["evaluation"].(map[string]any)
	if eval["approved"] != true {
		t.Fatalf("evaluation: %v", eval)
	}
	if code := ccFail(t, ts, "DeletePullRequestApprovalRule", `{"pullRequestId":"`+id+`","approvalRuleName":"one"}`); code != "CannotDeleteApprovalRuleFromTemplateException" {
		t.Fatalf("delete template rule: %s", code)
	}

	// Comments on the compared commits, a reply and a reaction.
	c := call("PostCommentForPullRequest", fmt.Sprintf(`{"pullRequestId":%q,"repositoryName":"p","beforeCommitId":%q,"afterCommitId":%q,"content":"nice","location":{"filePath":"f.txt","filePosition":0,"relativeFileVersion":"AFTER"}}`, id, first, src))["comment"].(map[string]any)
	reply := call("PostCommentReply", `{"inReplyTo":"`+c["commentId"].(string)+`","content":"thanks"}`)["comment"].(map[string]any)
	call("PutCommentReaction", `{"commentId":"`+reply["commentId"].(string)+`","reactionValue":":thumbsup:"}`)
	if code := ccFail(t, ts, "PutCommentReaction", `{"commentId":"`+reply["commentId"].(string)+`","reactionValue":"nope"}`); code != "InvalidReactionValueException" {
		t.Fatalf("bad reaction: %s", code)
	}
	threads := call("GetCommentsForPullRequest", `{"pullRequestId":"`+id+`"}`)["commentsForPullRequestData"].([]any)
	if len(threads) != 1 || len(threads[0].(map[string]any)["comments"].([]any)) != 2 {
		t.Fatalf("comment threads: %v", threads)
	}
	reactions := call("GetCommentReactions", `{"commentId":"`+reply["commentId"].(string)+`"}`)["reactionsForComment"].([]any)
	if len(reactions) != 1 {
		t.Fatalf("reactions: %v", reactions)
	}
	call("DeleteCommentContent", `{"commentId":"`+c["commentId"].(string)+`"}`)
	if code := ccFail(t, ts, "GetComment", `{"commentId":"`+c["commentId"].(string)+`"}`); code != "CommentDeletedException" {
		t.Fatalf("deleted comment: %s", code)
	}

	// Merge, then the pull request is closed and can't be reopened.
	merged := call("MergePullRequestByFastForward", `{"pullRequestId":"`+id+`","repositoryName":"p","sourceCommitId":"`+src+`"}`)["pullRequest"].(map[string]any)
	target := merged["pullRequestTargets"].([]any)[0].(map[string]any)
	if merged["pullRequestStatus"] != "CLOSED" || target["mergeMetadata"].(map[string]any)["isMerged"] != true {
		t.Fatalf("merged pull request: %v", merged)
	}
	if code := ccFail(t, ts, "UpdatePullRequestStatus", `{"pullRequestId":"`+id+`","pullRequestStatus":"OPEN"}`); code != "InvalidPullRequestStatusUpdateException" {
		t.Fatalf("reopen: %s", code)
	}
	events := call("DescribePullRequestEvents", `{"pullRequestId":"`+id+`"}`)["pullRequestEvents"].([]any)
	kinds := []string{}
	for _, e := range events {
		kinds = append(kinds, e.(map[string]any)["pullRequestEventType"].(string))
	}
	if !strings.Contains(strings.Join(kinds, ","), "PULL_REQUEST_MERGE_STATE_CHANGED") {
		t.Fatalf("events: %v", kinds)
	}
	if code := ccFail(t, ts, "GetPullRequest", `{"pullRequestId":"999"}`); code != "PullRequestDoesNotExistException" {
		t.Fatalf("missing pull request: %s", code)
	}

	// A new commit on the source of an open PR starts a new revision.
	src2 := ccBranchCommit(t, ts, "p", "f2", src, "g.txt", "g\n")
	pr2 := call("CreatePullRequest", `{"title":"g","targets":[{"repositoryName":"p","sourceReference":"f2"}]}`)["pullRequest"].(map[string]any)
	ccBranchCommit(t, ts, "p", "f2", src2, "h.txt", "h\n")
	after := call("GetPullRequest", `{"pullRequestId":"`+pr2["pullRequestId"].(string)+`"}`)["pullRequest"].(map[string]any)
	if after["revisionId"] == pr2["revisionId"] {
		t.Fatalf("revision did not change after the source moved")
	}
	_ = ccRuleOneApproval
}
