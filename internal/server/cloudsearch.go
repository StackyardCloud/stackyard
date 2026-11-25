package server

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon CloudSearch configuration API: AWS Query protocol (form-encoded
// Action=...&Version=2013-01-01 posted to /), XML responses, signed
// cloudsearch. The CloudSearch Domain data plane shares the signing name but
// lives under /2013-01-01/{documents/batch,search,suggest}, so this router only
// claims the root path.

const (
	csVersion   = "2013-01-01"
	csXMLNS     = "http://cloudsearch.amazonaws.com/doc/2013-01-01/"
	csMaxBody   = 1 << 20
	csAccountID = "123456789012"
	csRegion    = "us-east-1"
	csMaxItems  = 1000 // list members decoded from one request
)

type csErr struct {
	status    int
	code, msg string
}

func (e *csErr) Error() string { return e.code + ": " + e.msg }

func csValidation(format string, args ...any) *csErr {
	return &csErr{http.StatusBadRequest, "ValidationException", fmt.Sprintf(format, args...)}
}

func csNotFound(format string, args ...any) *csErr {
	return &csErr{http.StatusConflict, "ResourceNotFound", fmt.Sprintf(format, args...)}
}

func csInvalidType(format string, args ...any) *csErr {
	return &csErr{http.StatusConflict, "InvalidType", fmt.Sprintf(format, args...)}
}

func csLimit(format string, args ...any) *csErr {
	return &csErr{http.StatusConflict, "LimitExceeded", fmt.Sprintf(format, args...)}
}

type csHandler func(st *csStore, in map[string]any) (map[string]any, *csErr)

var csHandlers = map[string]csHandler{
	"CreateDomain":                  (*csStore).createDomain,
	"DeleteDomain":                  (*csStore).deleteDomain,
	"DescribeDomains":               (*csStore).describeDomains,
	"ListDomainNames":               (*csStore).listDomainNames,
	"IndexDocuments":                (*csStore).indexDocuments,
	"BuildSuggesters":               (*csStore).buildSuggesters,
	"DefineIndexField":              (*csStore).defineIndexField,
	"DeleteIndexField":              (*csStore).deleteIndexField,
	"DescribeIndexFields":           (*csStore).describeIndexFields,
	"DefineExpression":              (*csStore).defineExpression,
	"DeleteExpression":              (*csStore).deleteExpression,
	"DescribeExpressions":           (*csStore).describeExpressions,
	"DefineAnalysisScheme":          (*csStore).defineAnalysisScheme,
	"DeleteAnalysisScheme":          (*csStore).deleteAnalysisScheme,
	"DescribeAnalysisSchemes":       (*csStore).describeAnalysisSchemes,
	"DefineSuggester":               (*csStore).defineSuggester,
	"DeleteSuggester":               (*csStore).deleteSuggester,
	"DescribeSuggesters":            (*csStore).describeSuggesters,
	"DescribeAvailabilityOptions":   (*csStore).describeAvailabilityOptions,
	"UpdateAvailabilityOptions":     (*csStore).updateAvailabilityOptions,
	"DescribeDomainEndpointOptions": (*csStore).describeDomainEndpointOptions,
	"UpdateDomainEndpointOptions":   (*csStore).updateDomainEndpointOptions,
	"DescribeScalingParameters":     (*csStore).describeScalingParameters,
	"UpdateScalingParameters":       (*csStore).updateScalingParameters,
	"DescribeServiceAccessPolicies": (*csStore).describeServiceAccessPolicies,
	"UpdateServiceAccessPolicies":   (*csStore).updateServiceAccessPolicies,
}

func isCloudSearchQueryCandidate(r *http.Request) bool {
	if r.URL.Path != "/" || (r.Method != http.MethodPost && r.Method != http.MethodGet) {
		return false
	}
	if hint := sigV4ServiceHint(r); hint != "" {
		return hint == "cloudsearch"
	}
	host := strings.ToLower(r.Host)
	return strings.HasPrefix(host, "cloudsearch.")
}

func (s *Server) handleCloudSearchQueryRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudSearchQueryCandidate(r) {
		return false
	}
	// A declared length bounds what is read; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > csMaxBody {
		respondCloudSearchError(w, &csErr{http.StatusBadRequest, "ValidationException", "request body must declare a Content-Length of at most 1 MiB"})
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "cloudsearch"); !ok {
		respondCloudSearchError(w, &csErr{status, code, msg})
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, csMaxBody))
	if err != nil {
		respondCloudSearchError(w, csValidation("unable to read request body"))
		return true
	}
	form, err := url.ParseQuery(string(raw))
	if err != nil {
		respondCloudSearchError(w, csValidation("invalid form body"))
		return true
	}
	for k, v := range r.URL.Query() {
		if _, ok := form[k]; !ok {
			form[k] = v
		}
	}
	action := form.Get("Action")
	h := csHandlers[action]
	if !cloudSearchOperationByName[action] || h == nil {
		respondCloudSearchError(w, &csErr{http.StatusBadRequest, "InvalidAction", "The action " + action + " is not valid for this web service."})
		return true
	}
	if v := form.Get("Version"); v != "" && v != csVersion {
		respondCloudSearchError(w, &csErr{http.StatusBadRequest, "InvalidParameterValue", "unsupported Version " + v})
		return true
	}
	model := awsmodels.CloudSearchModel()
	shapes := model.Operations[action]
	in := map[string]any{}
	if shapes.Input != "" {
		decoded, derr := csDecodeQuery(model, shapes.Input, form, "")
		if derr != nil {
			respondCloudSearchError(w, derr)
			return true
		}
		in, _ = decoded.(map[string]any)
		if in == nil {
			in = map[string]any{}
		}
		if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
			respondCloudSearchError(w, csValidation("%s", strings.Join(problems, "; ")))
			return true
		}
	}
	out, cerr := h(s.cloudSearch, in)
	if cerr != nil {
		respondCloudSearchError(w, cerr)
		return true
	}
	// out is built fresh by every handler; nothing in it is shared with the store.
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, `<%sResponse xmlns="%s"><%sResult>`, action, csXMLNS, action)
	csEncodeXML(&b, model, shapes.Output, out)
	fmt.Fprintf(&b, `</%sResult><ResponseMetadata><RequestId>%s</RequestId></ResponseMetadata></%sResponse>`, action, uuid.NewString(), action)
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, b.String())
	return true
}

func respondCloudSearchError(w http.ResponseWriter, e *csErr) {
	kind := "Sender"
	if e.status >= 500 {
		kind = "Receiver"
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, `<ErrorResponse xmlns="%s"><Error><Type>%s</Type><Code>`, csXMLNS, kind)
	_ = xml.EscapeText(&b, []byte(e.code))
	b.WriteString("</Code><Message>")
	_ = xml.EscapeText(&b, []byte(e.msg))
	fmt.Fprintf(&b, "</Message></Error><RequestId>%s</RequestId></ErrorResponse>", uuid.NewString())
	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(e.status)
	_, _ = io.WriteString(w, b.String())
}

// csDecodeQuery turns Query-protocol parameters into the JSON-like value the
// model validates: structures by member name, lists as Name.member.N. Only
// members the model defines are read, so callers can't set anything else.
// ponytail: CloudSearch inputs have no maps or flattened lists; add
// Name.entry.N.key/value decoding when a model needs it.
func csDecodeQuery(model *awsmodels.Model, shapeName string, form url.Values, prefix string) (any, *csErr) {
	s := model.Shapes[shapeName]
	if s == nil {
		return nil, nil
	}
	switch s.Type {
	case "structure":
		out := map[string]any{}
		for name, ref := range s.Members {
			key := name
			if prefix != "" {
				key = prefix + "." + name
			}
			v, err := csDecodeQuery(model, ref.Shape, form, key)
			if err != nil {
				return nil, err
			}
			if v != nil {
				out[name] = v
			}
		}
		if len(out) == 0 && prefix != "" {
			return nil, nil
		}
		return out, nil
	case "list":
		if s.Member == nil {
			return nil, nil
		}
		var items []any
		for n := 1; n <= csMaxItems+1; n++ {
			v, err := csDecodeQuery(model, s.Member.Shape, form, prefix+".member."+strconv.Itoa(n))
			if err != nil {
				return nil, err
			}
			if v == nil {
				break
			}
			if n > csMaxItems {
				return nil, csValidation("%s has more than %d members", prefix, csMaxItems)
			}
			items = append(items, v)
		}
		if items == nil {
			if _, ok := form[prefix]; ok { // an explicitly empty list
				return []any{}, nil
			}
			return nil, nil
		}
		return items, nil
	}
	vals, ok := form[prefix]
	if !ok || len(vals) == 0 {
		return nil, nil
	}
	raw := vals[0]
	switch s.Type {
	case "boolean":
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, csValidation("%s must be true or false", prefix)
		}
		return b, nil
	case "integer", "long":
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, csValidation("%s must be an integer", prefix)
		}
		return float64(n), nil
	case "double", "float":
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, csValidation("%s must be a number", prefix)
		}
		return f, nil
	}
	return raw, nil
}

// csEncodeXML writes v as the Query-protocol XML body of shapeName: lists as
// <member> items, maps as <entry><key/><value/></entry>.
func csEncodeXML(b *strings.Builder, model *awsmodels.Model, shapeName string, v any) {
	s := model.Shapes[shapeName]
	if s == nil || v == nil {
		return
	}
	switch s.Type {
	case "structure":
		obj, _ := v.(map[string]any)
		names := make([]string, 0, len(s.Members))
		for name := range s.Members {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			item, ok := obj[name]
			if !ok || item == nil {
				continue
			}
			b.WriteString("<" + name + ">")
			csEncodeXML(b, model, s.Members[name].Shape, item)
			b.WriteString("</" + name + ">")
		}
	case "list":
		items, _ := v.([]any)
		for _, item := range items {
			b.WriteString("<member>")
			csEncodeXML(b, model, s.Member.Shape, item)
			b.WriteString("</member>")
		}
	case "map":
		obj, _ := v.(map[string]any)
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString("<entry><key>")
			_ = xml.EscapeText(b, []byte(k))
			b.WriteString("</key><value>")
			csEncodeXML(b, model, s.Value.Shape, obj[k])
			b.WriteString("</value></entry>")
		}
	default:
		_ = xml.EscapeText(b, []byte(csScalar(v)))
	}
}

func csScalar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case time.Time:
		return t.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	return fmt.Sprint(v)
}

// ---- request helpers ----

func csStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func csBool(in map[string]any, key string) (bool, bool) {
	v, ok := in[key].(bool)
	return v, ok
}

func csMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}

func csStrList(in map[string]any, key string) []string {
	list, _ := in[key].([]any)
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
