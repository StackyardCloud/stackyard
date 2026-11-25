package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Textract: awsJson1_1, X-Amz-Target Textract.*, signed textract. The
// CLI 2.41 model lists smithy-rpc-v2-cbor first, but the CLI sends JSON 1.1.

const (
	txTargetPrefix = "Textract."
	txMaxBody      = 16 << 20 // a 10 MiB document is ~14 MiB once base64-encoded
)

type txErr struct{ code, msg string }

func txError(code, msg string) *txErr { return &txErr{code: code, msg: msg} }

func txInvalid(format string, args ...any) *txErr {
	return txError("InvalidParameterException", fmt.Sprintf(format, args...))
}

type txHandler func(s *Server, in map[string]any) (map[string]any, *txErr)

var txHandlers map[string]txHandler

func init() {
	txHandlers = map[string]txHandler{
		"DetectDocumentText":         (*Server).txDetectDocumentText,
		"AnalyzeDocument":            (*Server).txAnalyzeDocument,
		"AnalyzeExpense":             (*Server).txAnalyzeExpense,
		"AnalyzeID":                  (*Server).txAnalyzeID,
		"StartDocumentTextDetection": (*Server).txStartDocumentTextDetection,
		"StartDocumentAnalysis":      (*Server).txStartDocumentAnalysis,
		"StartExpenseAnalysis":       (*Server).txStartExpenseAnalysis,
		"StartLendingAnalysis":       (*Server).txStartLendingAnalysis,
		"GetDocumentTextDetection":   txGetJob("StartDocumentTextDetection", false),
		"GetDocumentAnalysis":        txGetJob("StartDocumentAnalysis", false),
		"GetExpenseAnalysis":         txGetJob("StartExpenseAnalysis", false),
		"GetLendingAnalysis":         txGetJob("StartLendingAnalysis", false),
		"GetLendingAnalysisSummary":  txGetJob("StartLendingAnalysis", true),
		"CreateAdapter":              (*Server).txCreateAdapter,
		"GetAdapter":                 func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.getAdapter(in) },
		"UpdateAdapter":              func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.updateAdapter(in) },
		"DeleteAdapter":              func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.deleteAdapter(in) },
		"ListAdapters":               func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.listAdapters(in) },
		"CreateAdapterVersion":       (*Server).txCreateAdapterVersion,
		"GetAdapterVersion":          func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.getAdapterVersion(in) },
		"DeleteAdapterVersion": func(s *Server, in map[string]any) (map[string]any, *txErr) {
			return s.textract.deleteAdapterVersion(in)
		},
		"ListAdapterVersions": func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.listAdapterVersions(in) },
		"TagResource":         func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.tagResource(in) },
		"UntagResource":       func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.untagResource(in) },
		"ListTagsForResource": func(s *Server, in map[string]any) (map[string]any, *txErr) { return s.textract.listTags(in) },
	}
}

func (s *Server) handleTextractJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), txTargetPrefix)
	if !ok {
		return false
	}
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > txMaxBody {
		respondTextractError(w, http.StatusBadRequest, "InvalidParameterException", "request body must declare a Content-Length of at most 16 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "textract"); !ok {
		respondTextractError(w, status, code, msg)
		return true
	}
	h := txHandlers[op]
	if _, known := textractOperationByName[op]; !known || h == nil {
		respondTextractError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return true
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondTextractError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
		return true
	}
	model := awsmodels.TextractModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondTextractError(w, http.StatusBadRequest, "InvalidParameterException", strings.Join(problems, "; "))
		return true
	}
	// Conform drops members the input shape doesn't define, so callers can't
	// reach the store's own fields.
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	out, terr := h(s, s.withS3Caller(conformed, r))
	if terr != nil {
		respondTextractError(w, http.StatusBadRequest, terr.code, terr.msg)
		return true
	}
	// out holds fresh maps; stored results it references are never mutated.
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondTextractError(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// ---- documents ----

const txServicePrincipal = "textract.amazonaws.com"

// txS3Bytes reads an S3Object {Bucket, Name, Version} from the S3 store.
func (s *Server) txS3Bytes(c s3Caller, obj map[string]any, limit int) ([]byte, *txErr) {
	bucket, _ := obj["Bucket"].(string)
	name, _ := obj["Name"].(string)
	version, _ := obj["Version"].(string)
	if bucket == "" || name == "" {
		return nil, txInvalid("S3Object needs a Bucket and a Name")
	}
	if !s.authorizeServiceS3As(c, txServicePrincipal, bucket, name, "s3:GetObject") {
		return nil, txError("AccessDeniedException", "Access denied reading s3://"+bucket+"/"+name)
	}
	o, err := s.s3.GetObject(bucket, name)
	if version != "" {
		o, err = s.s3.GetObjectVersion(bucket, name, version)
	}
	if err != nil {
		return nil, txError("InvalidS3ObjectException", "Unable to get object metadata from S3. Check object key, region and/or access permissions.")
	}
	if len(o.Body) > limit {
		return nil, txError("DocumentTooLargeException", fmt.Sprintf("S3 object size %d is more than the maximum limit %d", len(o.Body), limit))
	}
	return o.Body, nil
}

// txLoadDocument reads a Document {Bytes | S3Object} and parses it.
func (s *Server) txLoadDocument(c s3Caller, doc map[string]any, limit int, singlePage bool) (*txDocument, *txErr) {
	b64, hasBytes := doc["Bytes"].(string)
	obj, hasS3 := doc["S3Object"].(map[string]any)
	var raw []byte
	switch {
	case hasBytes && hasS3:
		return nil, txInvalid("Document accepts either Bytes or S3Object, not both")
	case hasBytes:
		var err error
		if raw, err = base64.StdEncoding.DecodeString(b64); err != nil {
			if raw, err = base64.RawStdEncoding.DecodeString(b64); err != nil {
				return nil, txInvalid("Document Bytes is not valid base64")
			}
		}
		if len(raw) > limit {
			return nil, txError("DocumentTooLargeException", fmt.Sprintf("Document size %d is more than the maximum limit %d", len(raw), limit))
		}
	case hasS3:
		var terr *txErr
		if raw, terr = s.txS3Bytes(c, obj, limit); terr != nil {
			return nil, terr
		}
	default:
		return nil, txInvalid("Document needs Bytes or an S3Object")
	}
	parsed, terr := txParseDocument(raw)
	if terr != nil {
		return nil, terr
	}
	if singlePage && len(parsed.pages) > 1 {
		return nil, txError("UnsupportedDocumentException", "Request has unsupported document format: synchronous operations accept single-page documents")
	}
	return parsed, nil
}

func txFeatures(in map[string]any) (map[string]bool, *txErr) {
	list, _ := in["FeatureTypes"].([]any)
	if len(list) == 0 {
		return nil, txInvalid("FeatureTypes must name at least one feature")
	}
	features := map[string]bool{}
	for _, f := range list {
		name, _ := f.(string)
		features[name] = true
	}
	queries := txQueries(in)
	if features["QUERIES"] && len(queries) == 0 {
		return nil, txInvalid("QueriesConfig is required when FeatureTypes includes QUERIES")
	}
	if !features["QUERIES"] && len(queries) > 0 {
		return nil, txInvalid("QueriesConfig requires FeatureTypes to include QUERIES")
	}
	return features, nil
}

func txQueries(in map[string]any) []any {
	qc, _ := in["QueriesConfig"].(map[string]any)
	q, _ := qc["Queries"].([]any)
	return q
}

// ---- synchronous analysis ----

func (s *Server) txDetectDocumentText(in map[string]any) (map[string]any, *txErr) {
	doc, err := s.txLoadDocument(s3CallerIn(in), txMap(in, "Document"), txSyncMaxBytes, true)
	if err != nil {
		return nil, err
	}
	a := txAnalyze(doc)
	return map[string]any{"DocumentMetadata": txMeta(doc), "Blocks": a.textBlocks(), "DetectDocumentTextModelVersion": "1.0"}, nil
}

func (s *Server) txAnalyzeDocument(in map[string]any) (map[string]any, *txErr) {
	features, err := txFeatures(in)
	if err != nil {
		return nil, err
	}
	if err := s.textract.checkAdapters(in); err != nil {
		return nil, err
	}
	doc, err := s.txLoadDocument(s3CallerIn(in), txMap(in, "Document"), txSyncMaxBytes, true)
	if err != nil {
		return nil, err
	}
	// ponytail: HumanLoopConfig is accepted but no human review loop starts (no A2I here).
	blocks, err := txAnalyze(doc).analyzeBlocks(features, txQueries(in))
	if err != nil {
		return nil, err
	}
	return map[string]any{"DocumentMetadata": txMeta(doc), "Blocks": blocks, "AnalyzeDocumentModelVersion": "1.0"}, nil
}

func (s *Server) txAnalyzeExpense(in map[string]any) (map[string]any, *txErr) {
	doc, err := s.txLoadDocument(s3CallerIn(in), txMap(in, "Document"), txSyncMaxBytes, true)
	if err != nil {
		return nil, err
	}
	a := txAnalyze(doc)
	return map[string]any{"DocumentMetadata": txMeta(doc), "ExpenseDocuments": []any{a.expenseDocument(1, a.pages)}}, nil
}

func (s *Server) txAnalyzeID(in map[string]any) (map[string]any, *txErr) {
	pages, _ := in["DocumentPages"].([]any)
	var docs []any
	for i, p := range pages {
		pm, _ := p.(map[string]any)
		doc, err := s.txLoadDocument(s3CallerIn(in), pm, txSyncMaxBytes, true)
		if err != nil {
			return nil, err
		}
		a := txAnalyze(doc)
		docs = append(docs, a.identityDocument(i+1, a.pages[0]))
	}
	return map[string]any{"DocumentMetadata": map[string]any{"Pages": len(pages)}, "IdentityDocuments": docs, "AnalyzeIDModelVersion": "1.0"}, nil
}

func txMeta(doc *txDocument) map[string]any { return map[string]any{"Pages": len(doc.pages)} }

func txMap(in map[string]any, key string) map[string]any { m, _ := in[key].(map[string]any); return m }

func txStr(in map[string]any, key string) string { v, _ := in[key].(string); return v }
