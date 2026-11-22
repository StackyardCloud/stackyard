package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS CodeCommit: awsJson 1.1, X-Amz-Target CodeCommit_20150413.<Op>,
// SigV4 service "codecommit". Every request is checked against the embedded
// botocore model (required members, types, enums, lengths, patterns) before
// the store sees it, and every response is shaped to the model's output.

const (
	codecommitTargetPrefix = "CodeCommit_20150413."
	// codecommitMaxBody bounds a request body. CreateCommit carries up to
	// 6 MB of file content, which base64 inflates by a third.
	codecommitMaxBody = 10 << 20
)

type ccErr struct {
	code string
	msg  string
}

func ccError(code, msg string) *ccErr { return &ccErr{code: code, msg: msg} }

func (s *Server) handleCodeCommitJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodeCommitJSONCandidate(r) {
		return false
	}
	action := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), codecommitTargetPrefix)
	if _, ok := codecommitOperationByName[action]; !ok {
		respondCodeCommitError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+action)
		return true
	}
	// A declared length bounds what net/http reads, so require one; chunked
	// bodies of unknown length are refused.
	if r.ContentLength < 0 || r.ContentLength > codecommitMaxBody {
		respondCodeCommitError(w, http.StatusRequestEntityTooLarge, "ValidationException", "request body must declare a Content-Length of at most 10 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "codecommit"); !ok {
		respondCodeCommitError(w, status, code, msg)
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondCodeCommitError(w, http.StatusBadRequest, "SerializationException", "unable to read request body")
		return true
	}
	in := map[string]any{}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			respondCodeCommitError(w, http.StatusBadRequest, "SerializationException", "request body is not a JSON object")
			return true
		}
	}
	model := awsmodels.CodeCommitModel()
	op := model.Operations[action]
	if e := codecommitValidate(model, op.Input, in); e != nil {
		respondCodeCommitError(w, http.StatusBadRequest, e.code, e.msg)
		return true
	}
	out, e := s.codecommit.handle(action, op.Output, in)
	if e != nil {
		respondCodeCommitError(w, http.StatusBadRequest, e.code, e.msg)
		return true
	}
	raw, err := json.Marshal(out)
	if err != nil {
		respondCodeCommitError(w, http.StatusInternalServerError, "InternalFailure", "unable to encode response")
		return true
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	return true
}

func isCodeCommitJSONCandidate(r *http.Request) bool {
	return r.Method == http.MethodPost && strings.HasPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), codecommitTargetPrefix)
}

func respondCodeCommitError(w http.ResponseWriter, status int, code, msg string) {
	raw, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

var codecommitProblemRE = regexp.MustCompile(`^([^:]*): (missing required member (\w+)|.*)$`)

// codecommitRequiredAlias maps request members whose "…RequiredException"
// isn't simply the member name capitalised.
var codecommitRequiredAlias = map[string]string{
	"filePath":                    "PathRequiredException",
	"folderPath":                  "PathRequiredException",
	"content":                     "CommentContentRequiredException",
	"approvalRuleContent":         "ApprovalRuleContentRequiredException",
	"approvalRuleTemplateContent": "ApprovalRuleTemplateContentRequiredException",
	"newRuleContent":              "ApprovalRuleContentRequiredException",
	"newRuleTemplateContent":      "ApprovalRuleTemplateContentRequiredException",
	"commitIds":                   "CommitIdsListRequiredException",
	"repositoryNames":             "RepositoryNamesRequiredException",
	"triggers":                    "RepositoryTriggersListRequiredException",
	"tags":                        "TagsMapRequiredException",
	"tagKeys":                     "TagKeysListRequiredException",
	"resourceArn":                 "ResourceArnRequiredException",
	"afterCommitSpecifier":        "CommitRequiredException",
	"sourceCommitSpecifier":       "CommitRequiredException",
	"destinationCommitSpecifier":  "CommitRequiredException",
	"commitSpecifier":             "CommitRequiredException",
	"defaultBranchName":           "BranchNameRequiredException",
	"oldName":                     "RepositoryNameRequiredException",
	"newName":                     "RepositoryNameRequiredException",
	"newApprovalRuleTemplateName": "ApprovalRuleTemplateNameRequiredException",
	"oldApprovalRuleTemplateName": "ApprovalRuleTemplateNameRequiredException",
	"overrideStatus":              "OverrideStatusRequiredException",
	"approvalState":               "ApprovalStateRequiredException",
	"reactionValue":               "ReactionValueRequiredException",
	"mergeOption":                 "MergeOptionRequiredException",
	"destinationArn":              "RepositoryTriggerDestinationArnRequiredException",
	"events":                      "RepositoryTriggerEventsListRequiredException",
}

// codecommitValidate checks in against the model and turns the first problem
// into the CodeCommit exception a real service would return.
func codecommitValidate(model *awsmodels.Model, shape string, in map[string]any) *ccErr {
	problems := model.Validate(shape, in)
	for _, p := range problems {
		if strings.Contains(p, "unknown member") {
			continue // botocore never sends these; ignore like the service does
		}
		m := codecommitProblemRE.FindStringSubmatch(p)
		member := ""
		if m != nil && m[3] != "" {
			member = m[3]
			if code, ok := codecommitRequiredAlias[member]; ok {
				return ccError(code, member+" is required")
			}
			if code := codecommitUpper(member) + "RequiredException"; codecommitIsError(model, code) {
				return ccError(code, member+" is required")
			}
			return ccError("ValidationException", p)
		}
		if m != nil {
			path := m[1]
			if i := strings.LastIndexAny(path, ".]"); i >= 0 {
				path = path[i+1:]
			}
			if code := "Invalid" + codecommitUpper(path) + "Exception"; path != "" && codecommitIsError(model, code) {
				return ccError(code, p)
			}
		}
		return ccError("ValidationException", p)
	}
	return nil
}

func codecommitIsError(model *awsmodels.Model, name string) bool {
	_, ok := model.Shapes[name]
	return ok && strings.HasSuffix(name, "Exception")
}

func codecommitUpper(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// codecommitConform round-trips out through JSON (so every list is []any and
// every object map[string]any, whatever Go types built it) and drops members
// the output shape doesn't define.
func codecommitConform(shape string, out map[string]any) any {
	raw, err := json.Marshal(out)
	if err != nil {
		return map[string]any{}
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return map[string]any{}
	}
	return awsmodels.CodeCommitModel().Conform(shape, generic)
}
