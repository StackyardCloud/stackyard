package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Connect Cases: REST JSON, signed "cases". Only requests signed for
// cases (or unsigned to a cases.* host) are claimed, so the many services that
// share /tags/{arn} keep their traffic. State lives in connectcases_store.go.

const (
	cncMaxBody  = 1 << 20
	cncAccount  = "123456789012"
	cncRegion   = "us-east-1"
	cncTimeForm = "2006-01-02T15:04:05.000Z"
)

type cncErr struct {
	status    int
	code, msg string
}

func cncValidation(format string, args ...any) *cncErr {
	return &cncErr{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func cncNotFound(format string, args ...any) *cncErr {
	return &cncErr{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf(format, args...)}
}

func cncConflict(format string, args ...any) *cncErr {
	return &cncErr{http.StatusConflict, "ConflictException", fmt.Sprintf(format, args...)}
}

func cncQuota(format string, args ...any) *cncErr {
	return &cncErr{http.StatusPaymentRequired, "ServiceQuotaExceededException", fmt.Sprintf(format, args...)}
}

type cncHandler func(st *cncStore, in map[string]any) (map[string]any, *cncErr)

var cncHandlers = map[string]cncHandler{
	"BatchCreateRelatedItem":    (*cncStore).batchCreateRelatedItem,
	"BatchGetCaseRule":          (*cncStore).batchGetCaseRule,
	"BatchGetField":             (*cncStore).batchGetField,
	"BatchPutFieldOptions":      (*cncStore).batchPutFieldOptions,
	"BatchUpdateCase":           (*cncStore).batchUpdateCase,
	"CreateCase":                (*cncStore).createCase,
	"CreateCaseRule":            (*cncStore).createCaseRule,
	"CreateDomain":              (*cncStore).createDomain,
	"CreateField":               (*cncStore).createField,
	"CreateLayout":              (*cncStore).createLayout,
	"CreateRelatedItem":         (*cncStore).createRelatedItem,
	"CreateTemplate":            (*cncStore).createTemplate,
	"DeleteCase":                (*cncStore).deleteCase,
	"DeleteCaseRule":            (*cncStore).deleteCaseRule,
	"DeleteDomain":              (*cncStore).deleteDomain,
	"DeleteField":               (*cncStore).deleteField,
	"DeleteLayout":              (*cncStore).deleteLayout,
	"DeleteRelatedItem":         (*cncStore).deleteRelatedItem,
	"DeleteTemplate":            (*cncStore).deleteTemplate,
	"GetCase":                   (*cncStore).getCase,
	"GetCaseAuditEvents":        (*cncStore).getCaseAuditEvents,
	"GetCaseEventConfiguration": (*cncStore).getCaseEventConfiguration,
	"GetDomain":                 (*cncStore).getDomain,
	"GetLayout":                 (*cncStore).getLayout,
	"GetMergeCasesStatus":       (*cncStore).getMergeCasesStatus,
	"GetTemplate":               (*cncStore).getTemplate,
	"ListCaseRules":             (*cncStore).listCaseRules,
	"ListCasesForContact":       (*cncStore).listCasesForContact,
	"ListDomains":               (*cncStore).listDomains,
	"ListFieldOptions":          (*cncStore).listFieldOptions,
	"ListFields":                (*cncStore).listFields,
	"ListLayouts":               (*cncStore).listLayouts,
	"ListTagsForResource":       (*cncStore).listTags,
	"ListTemplates":             (*cncStore).listTemplates,
	"PutCaseEventConfiguration": (*cncStore).putCaseEventConfiguration,
	"SearchAllRelatedItems":     (*cncStore).searchAllRelatedItems,
	"SearchCases":               (*cncStore).searchCases,
	"SearchRelatedItems":        (*cncStore).searchRelatedItems,
	"StartMergeCases":           (*cncStore).startMergeCases,
	"TagResource":               (*cncStore).tagResource,
	"UntagResource":             (*cncStore).untagResource,
	"UpdateCase":                (*cncStore).updateCase,
	"UpdateCaseRule":            (*cncStore).updateCaseRule,
	"UpdateField":               (*cncStore).updateField,
	"UpdateLayout":              (*cncStore).updateLayout,
	"UpdateRelatedItem":         (*cncStore).updateRelatedItem,
	"UpdateTemplate":            (*cncStore).updateTemplate,
}

func cncRoute(method, path string) (string, map[string]string) {
	for _, op := range connectCasesOperations {
		if op.Method != method {
			continue
		}
		if params, ok := connectPathParams(op.URI, path); ok {
			return op.Name, params
		}
	}
	return "", nil
}

// cncQueryMembers are the query-string members, with whether each is a list.
var cncQueryMembers = map[string]bool{"maxResults": false, "nextToken": false, "values": true, "status": true, "tagKeys": true}

func (s *Server) handleConnectCasesRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	hint := sigV4ServiceHint(r)
	if hint != "cases" && !(hint == "" && strings.HasPrefix(strings.ToLower(r.Host), "cases.")) {
		return false
	}
	if r.ContentLength < 0 || r.ContentLength > cncMaxBody {
		respondCncError(w, cncValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "cases"); !ok {
		respondCncError(w, &cncErr{status, code, msg})
		return true
	}
	op, params := cncRoute(r.Method, rawRequestPath(r))
	if op == "" {
		respondCncError(w, &cncErr{http.StatusNotFound, "UnknownOperationException", "unknown operation"})
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, cncMaxBody))
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondCncError(w, cncValidation("invalid JSON body"))
		return true
	}
	model := awsmodels.ConnectCasesModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondCncError(w, cncValidation("%s", strings.Join(problems, "; ")))
		return true
	}
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	// URI and query members are bound after Conform so a body can't set them.
	for k, v := range params {
		conformed[k] = v
	}
	if cerr := cncBindQuery(r, model.Shapes[shapes.Input], conformed); cerr != nil {
		respondCncError(w, cerr)
		return true
	}
	out, cerr := cncHandlers[op](s.connectCases, conformed)
	if cerr != nil {
		respondCncError(w, cerr)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	// out is built fresh by every handler; nothing in it is shared with the store.
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

// cncBindQuery copies the input shape's query-string members into in.
func cncBindQuery(r *http.Request, shape *awsmodels.ModelShape, in map[string]any) *cncErr {
	if shape == nil {
		return nil
	}
	q := r.URL.Query()
	for name, ref := range shape.Members {
		if ref.Location != "querystring" {
			continue
		}
		list, known := cncQueryMembers[ref.LocationName]
		values := q[ref.LocationName]
		switch {
		case !known || len(values) == 0:
		case list:
			if len(values) > 50 {
				return cncValidation("%s takes at most 50 values", name)
			}
			items := make([]any, len(values))
			for i, v := range values {
				items[i] = v
			}
			in[name] = items
		case name == "maxResults":
			n, err := strconv.ParseInt(values[0], 10, 32)
			if err != nil || n < 1 || n > 100 {
				return cncValidation("maxResults must be an integer from 1 to 100")
			}
			in[name] = float64(n)
		default:
			in[name] = values[0]
		}
	}
	return nil
}

func respondCncError(w http.ResponseWriter, e *cncErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

// ---- request helpers ----

func cncStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func cncMap(in map[string]any, key string) map[string]any {
	v, _ := in[key].(map[string]any)
	return v
}

func cncList(in map[string]any, key string) []any {
	v, _ := in[key].([]any)
	return v
}

func cncTime(t time.Time) string { return t.UTC().Format(cncTimeForm) }

// cncUnion returns the single member set in a union, or an error naming it.
func cncUnion(v map[string]any, what string) (string, any, *cncErr) {
	if len(v) != 1 {
		return "", nil, cncValidation("%s must set exactly one member", what)
	}
	for k, val := range v {
		return k, val, nil
	}
	return "", nil, nil
}

// cncPage slices n items by nextToken (an offset) and maxResults.
func cncPage(in map[string]any, n, def, maxAllowed int) (int, int, string, *cncErr) {
	start := 0
	if tok := cncStr(in, "nextToken"); tok != "" {
		v, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", cncValidation("nextToken is not valid")
		}
		start = int(v)
	}
	limit := def
	if v, ok := in["maxResults"].(float64); ok {
		limit = int(v)
	}
	if limit < 1 || limit > maxAllowed {
		return 0, 0, "", cncValidation("maxResults must be from 1 to %d", maxAllowed)
	}
	end := min(start+limit, n)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

// cncCopy deep-copies decoded JSON for storage or output.
func cncCopy(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[k] = cncCopy(item)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = cncCopy(item)
		}
		return out
	}
	return v
}

func cncCopyMap(m map[string]any) map[string]any {
	out, _ := cncCopy(m).(map[string]any)
	if out == nil {
		out = map[string]any{}
	}
	return out
}
