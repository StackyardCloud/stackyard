package server

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const ccMaxCommentBytes = 10240

type ccCommentKey struct {
	repo, pr, before, after, beforeBlob, afterBlob string
	path, version                                  string
	position                                       float64
	hasLocation, hasPosition                       bool
}

type ccComment struct {
	id, content, inReplyTo, token string
	created, modified             time.Time
	deleted                       bool
	key                           ccCommentKey
	reactions                     map[string]string // user ARN -> short code
}

// ccReactions are the reactions the service accepts, by short code.
// ponytail: a representative subset of the service's emoji list.
var ccReactions = map[string][2]string{
	":thumbsup:":   {"👍", "U+1F44D"},
	":thumbsdown:": {"👎", "U+1F44E"},
	":smile:":      {"😄", "U+1F604"},
	":laughing:":   {"😆", "U+1F606"},
	":confused:":   {"😕", "U+1F615"},
	":heart:":      {"❤️", "U+2764"},
	":tada:":       {"🎉", "U+1F389"},
	":rocket:":     {"🚀", "U+1F680"},
	":eyes:":       {"👀", "U+1F440"},
}

func ccReactionCode(value string) (string, bool) {
	if _, ok := ccReactions[value]; ok {
		return value, true
	}
	for code, forms := range ccReactions {
		if forms[0] == value || strings.EqualFold(forms[1], value) {
			return code, true
		}
	}
	return "", false
}

func registerCodeCommitCommentHandlers() {
	for name, h := range map[string]ccHandler{
		"PostCommentForComparedCommit": (*ccStore).postCommentForComparedCommit,
		"PostCommentForPullRequest":    (*ccStore).postCommentForPullRequest,
		"PostCommentReply":             (*ccStore).postCommentReply,
		"GetComment":                   (*ccStore).getComment,
		"UpdateComment":                (*ccStore).updateComment,
		"DeleteCommentContent":         (*ccStore).deleteCommentContent,
		"GetCommentsForComparedCommit": (*ccStore).getCommentsForComparedCommit,
		"GetCommentsForPullRequest":    (*ccStore).getCommentsForPullRequest,
		"PutCommentReaction":           (*ccStore).putCommentReaction,
		"GetCommentReactions":          (*ccStore).getCommentReactions,
	} {
		ccHandlers[name] = h
	}
}

func (c *ccComment) out() map[string]any {
	counts := map[string]any{}
	callers := []any{}
	for user, code := range c.reactions {
		counts[ccReactions[code][0]] = toInt(counts[ccReactions[code][0]]) + 1
		if user == ccCallerARN {
			callers = append(callers, ccReactions[code][0])
		}
	}
	out := map[string]any{
		"commentId": c.id, "authorArn": ccCallerARN, "creationDate": ccEpoch(c.created), "lastModifiedDate": ccEpoch(c.modified),
		"deleted": c.deleted, "callerReactions": callers, "reactionCounts": counts,
	}
	if !c.deleted {
		out["content"] = c.content
	}
	if c.inReplyTo != "" {
		out["inReplyTo"] = c.inReplyTo
	}
	if c.token != "" {
		out["clientRequestToken"] = c.token
	}
	return out
}

func toInt(v any) int {
	n, _ := v.(int)
	return n
}

func (k ccCommentKey) location() map[string]any {
	if !k.hasLocation {
		return nil
	}
	loc := map[string]any{"filePath": k.path, "relativeFileVersion": k.version}
	if k.hasPosition {
		loc["filePosition"] = k.position
	}
	return loc
}

func ccCheckContent(content string) *ccErr {
	if content == "" {
		return ccError("CommentContentRequiredException", "content is required")
	}
	if len(content) > ccMaxCommentBytes {
		return ccError("CommentContentSizeLimitExceededException", "comments can be at most 10,240 characters")
	}
	return nil
}

// newComment stores a comment, honouring clientRequestToken idempotency.
func (st *ccStore) newComment(in map[string]any, key ccCommentKey, inReplyTo string) (*ccComment, *ccErr) {
	content, token := ccStr(in, "content"), ccStr(in, "clientRequestToken")
	if id, ok := st.ctokens[token]; ok && token != "" {
		c := st.comments[id]
		if c.content != content {
			return nil, ccError("IdempotencyParameterMismatchException", "the client request token was already used with different content")
		}
		return c, nil
	}
	if err := ccCheckContent(content); err != nil {
		return nil, err
	}
	if len(st.comments) >= ccMaxComments {
		return nil, ccError("ValidationException", fmt.Sprintf("Stackyard holds at most %d comments", ccMaxComments))
	}
	now := ccNow()
	c := &ccComment{id: ccUUID(), content: content, inReplyTo: inReplyTo, token: token, created: now, modified: now, key: key, reactions: map[string]string{}}
	st.comments[c.id] = c
	if token != "" {
		st.ctokens[token] = c.id
	}
	return c, nil
}

// commentKey validates the compared commits and location of a new comment.
func (st *ccStore) commentKey(r *ccRepo, in map[string]any, pr string) (ccCommentKey, *ccErr) {
	key := ccCommentKey{repo: r.name, pr: pr}
	after, err := r.commit(ccStr(in, "afterCommitId"))
	if err != nil {
		return key, err
	}
	key.after = after.id
	var before *ccCommit
	if id := ccStr(in, "beforeCommitId"); id != "" {
		if before, err = r.commit(id); err != nil {
			return key, err
		}
		if before.id == after.id {
			return key, ccError("BeforeCommitIdAndAfterCommitIdAreSameException", "beforeCommitId and afterCommitId are the same")
		}
		key.before = before.id
	}
	loc := ccMap(in, "location")
	if loc == nil {
		return key, nil
	}
	p, ok := ccCleanPath(ccStr(loc, "filePath"))
	if !ok || p == "" {
		return key, ccError("InvalidPathException", "location.filePath is not valid")
	}
	version := ccStr(loc, "relativeFileVersion")
	if version == "" {
		version = "AFTER"
	}
	target := after
	if version == "BEFORE" {
		if before == nil {
			return key, ccError("InvalidRelativeFileVersionEnumException", "BEFORE needs a beforeCommitId")
		}
		target = before
	}
	e, has := target.files[p]
	if !has {
		return key, ccError("PathDoesNotExistException", "path "+p+" does not exist in the compared commit")
	}
	key.hasLocation, key.path, key.version = true, p, version
	if pos, ok := loc["filePosition"].(float64); ok {
		if pos < 0 || pos > float64(len(r.blobs[e.blob])) {
			return key, ccError("InvalidFilePositionException", "filePosition is outside the file")
		}
		key.position, key.hasPosition = pos, true
	}
	if fe, ok := after.files[p]; ok {
		key.afterBlob = fe.blob
	}
	if before != nil {
		if fe, ok := before.files[p]; ok {
			key.beforeBlob = fe.blob
		}
	}
	return key, nil
}

func (k ccCommentKey) threadOut(comments []any) map[string]any {
	out := map[string]any{"repositoryName": k.repo, "afterCommitId": k.after, "comments": comments}
	if k.before != "" {
		out["beforeCommitId"] = k.before
	}
	if k.beforeBlob != "" {
		out["beforeBlobId"] = k.beforeBlob
	}
	if k.afterBlob != "" {
		out["afterBlobId"] = k.afterBlob
	}
	if loc := k.location(); loc != nil {
		out["location"] = loc
	}
	if k.pr != "" {
		out["pullRequestId"] = k.pr
	}
	return out
}

func (st *ccStore) postCommentForComparedCommit(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	key, err := st.commentKey(r, in, "")
	if err != nil {
		return nil, err
	}
	c, err := st.newComment(in, key, "")
	if err != nil {
		return nil, err
	}
	out := key.threadOut(nil)
	delete(out, "comments")
	out["comment"] = c.out()
	return out, nil
}

func (st *ccStore) postCommentForPullRequest(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	if name := ccStr(in, "repositoryName"); name != pr.targets[0].repo {
		return nil, ccError("RepositoryNotAssociatedWithPullRequestException", "repository "+name+" is not the pull request's repository")
	}
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	key, err := st.commentKey(r, in, pr.id)
	if err != nil {
		return nil, err
	}
	c, err := st.newComment(in, key, "")
	if err != nil {
		return nil, err
	}
	pr.activity = ccNow()
	out := key.threadOut(nil)
	delete(out, "comments")
	out["comment"] = c.out()
	return out, nil
}

func (st *ccStore) comment(in map[string]any, key string) (*ccComment, *ccErr) {
	id := ccStr(in, key)
	if id == "" {
		return nil, ccError("CommentIdRequiredException", key+" is required")
	}
	c, ok := st.comments[id]
	if !ok {
		return nil, ccError("CommentDoesNotExistException", "comment "+id+" does not exist")
	}
	return c, nil
}

func (st *ccStore) postCommentReply(in map[string]any) (map[string]any, *ccErr) {
	parent, err := st.comment(in, "inReplyTo")
	if err != nil {
		return nil, err
	}
	c, err := st.newComment(in, parent.key, parent.id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"comment": c.out()}, nil
}

func (st *ccStore) getComment(in map[string]any) (map[string]any, *ccErr) {
	c, err := st.comment(in, "commentId")
	if err != nil {
		return nil, err
	}
	if c.deleted {
		return nil, ccError("CommentDeletedException", "comment "+c.id+" has been deleted")
	}
	return map[string]any{"comment": c.out()}, nil
}

func (st *ccStore) updateComment(in map[string]any) (map[string]any, *ccErr) {
	c, err := st.comment(in, "commentId")
	if err != nil {
		return nil, err
	}
	if c.deleted {
		return nil, ccError("CommentDeletedException", "comment "+c.id+" has been deleted")
	}
	content := ccStr(in, "content")
	if err := ccCheckContent(content); err != nil {
		return nil, err
	}
	c.content, c.modified = content, ccNow()
	return map[string]any{"comment": c.out()}, nil
}

func (st *ccStore) deleteCommentContent(in map[string]any) (map[string]any, *ccErr) {
	c, err := st.comment(in, "commentId")
	if err != nil {
		return nil, err
	}
	c.content, c.deleted, c.modified = "", true, ccNow()
	return map[string]any{"comment": c.out()}, nil
}

// threads groups matching comments by compared commits and location, oldest
// first, top-level comments followed by their replies.
func (st *ccStore) threads(match func(ccCommentKey) bool) []map[string]any {
	groups := map[ccCommentKey][]*ccComment{}
	var keys []ccCommentKey
	for _, c := range st.comments {
		if !match(c.key) {
			continue
		}
		if _, ok := groups[c.key]; !ok {
			keys = append(keys, c.key)
		}
		groups[c.key] = append(groups[c.key], c)
	}
	first := func(k ccCommentKey) time.Time {
		t := groups[k][0].created
		for _, c := range groups[k] {
			if c.created.Before(t) {
				t = c.created
			}
		}
		return t
	}
	sort.Slice(keys, func(i, j int) bool { return first(keys[i]).Before(first(keys[j])) })
	out := []map[string]any{}
	for _, k := range keys {
		cs := groups[k]
		sort.Slice(cs, func(i, j int) bool { return cs[i].created.Before(cs[j].created) })
		comments := []any{}
		for _, c := range cs {
			comments = append(comments, c.out())
		}
		out = append(out, k.threadOut(comments))
	}
	return out
}

func (st *ccStore) getCommentsForComparedCommit(in map[string]any) (map[string]any, *ccErr) {
	r, err := st.repo(in)
	if err != nil {
		return nil, err
	}
	after, err := r.commit(ccStr(in, "afterCommitId"))
	if err != nil {
		return nil, err
	}
	before := ccStr(in, "beforeCommitId")
	threads := st.threads(func(k ccCommentKey) bool {
		return k.pr == "" && k.repo == r.name && k.after == after.id && (before == "" || k.before == before)
	})
	page, next, err := ccPage(threads, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"commentsForComparedCommitData": page, "nextToken": next}, nil
}

func (st *ccStore) getCommentsForPullRequest(in map[string]any) (map[string]any, *ccErr) {
	pr, err := st.pullRequest(in)
	if err != nil {
		return nil, err
	}
	repo, before, after := ccStr(in, "repositoryName"), ccStr(in, "beforeCommitId"), ccStr(in, "afterCommitId")
	if repo != "" && repo != pr.targets[0].repo {
		return nil, ccError("RepositoryNotAssociatedWithPullRequestException", "repository "+repo+" is not the pull request's repository")
	}
	threads := st.threads(func(k ccCommentKey) bool {
		return k.pr == pr.id && (before == "" || k.before == before) && (after == "" || k.after == after)
	})
	page, next, err := ccPage(threads, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"commentsForPullRequestData": page, "nextToken": next}, nil
}

func (st *ccStore) putCommentReaction(in map[string]any) (map[string]any, *ccErr) {
	c, err := st.comment(in, "commentId")
	if err != nil {
		return nil, err
	}
	if c.deleted {
		return nil, ccError("CommentDeletedException", "comment "+c.id+" has been deleted")
	}
	code, ok := ccReactionCode(ccStr(in, "reactionValue"))
	if !ok {
		return nil, ccError("InvalidReactionValueException", "reaction "+ccStr(in, "reactionValue")+" is not supported")
	}
	c.reactions[ccCallerARN] = code // one reaction per user, as in the service
	return map[string]any{}, nil
}

func (st *ccStore) getCommentReactions(in map[string]any) (map[string]any, *ccErr) {
	c, err := st.comment(in, "commentId")
	if err != nil {
		return nil, err
	}
	user := ccStr(in, "reactionUserArn")
	if user != "" && !strings.HasPrefix(user, "arn:aws:") {
		return nil, ccError("InvalidReactionUserArnException", "reactionUserArn is not a valid ARN")
	}
	byCode := map[string][]any{}
	for u, code := range c.reactions {
		if user == "" || u == user {
			byCode[code] = append(byCode[code], u)
		}
	}
	codes := make([]string, 0, len(byCode))
	for code := range byCode {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	list := []map[string]any{}
	for _, code := range codes {
		forms := ccReactions[code]
		list = append(list, map[string]any{
			"reaction":                       map[string]any{"emoji": forms[0], "shortCode": code, "unicode": forms[1]},
			"reactionUsers":                  byCode[code],
			"reactionsFromDeletedUsersCount": 0,
		})
	}
	page, next, err := ccPage(list, in, "nextToken", "maxResults")
	if err != nil {
		return nil, err
	}
	return map[string]any{"reactionsForComment": page, "nextToken": next}, nil
}
