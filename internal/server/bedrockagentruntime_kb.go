package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Knowledge-base retrieval. There are no embeddings: a document's score is the
// share of the query's terms it contains, which is deterministic and enough to
// rank the documents a test ingests.
// ponytail: term overlap, not vectors; swap in embeddings if ranking ever matters.

const barNoAnswer = "Sorry, I am unable to assist you with this request."

var barStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "by": true, "can": true, "do": true,
	"does": true, "for": true, "from": true, "how": true, "in": true, "is": true, "it": true, "me": true, "of": true, "on": true,
	"or": true, "tell": true, "that": true, "the": true, "this": true, "to": true, "what": true, "when": true, "where": true,
	"which": true, "who": true, "why": true, "with": true, "you": true, "about": true,
}

// barHit is one scored document.
type barHit struct {
	kbID  string
	doc   map[string]any
	text  string
	score float64
}

func barTokens(s string) []string {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	out := words[:0]
	for _, w := range words {
		if len(w) > 1 && !barStopWords[w] {
			out = append(out, w)
		}
	}
	return out
}

func barScore(query []string, text string) float64 {
	if len(query) == 0 {
		return 0
	}
	have := map[string]bool{}
	for _, t := range barTokens(text) {
		have[t] = true
	}
	matched := 0
	for _, t := range query {
		if have[t] {
			matched++
		}
	}
	return float64(matched) / float64(len(query))
}

// barDocText returns the text of an ingested document (inline text, or
// text-like inline bytes) and its media type. S3 documents have no text here.
func barDocText(doc map[string]any) (string, string) {
	content := barMap(doc, "content")
	inline := barMap(barMap(content, "custom"), "inlineContent")
	switch baStr(inline, "type") {
	case "TEXT":
		return barPathStr(inline, "textContent", "data"), "text/plain"
	case "BYTE":
		mime := barPathStr(inline, "byteContent", "mimeType")
		raw, err := base64.StdEncoding.DecodeString(barPathStr(inline, "byteContent", "data"))
		if err == nil && (strings.HasPrefix(mime, "text/") || strings.Contains(mime, "json") || strings.Contains(mime, "xml")) {
			return string(raw), mime
		}
		return "", mime
	}
	return "", "application/octet-stream"
}

// barDocID is the runtime document ID: the custom ID or the S3 URI.
func barDocID(doc map[string]any) string {
	ident := barMap(doc, "identifier")
	if id := barPathStr(ident, "custom", "id"); id != "" {
		return id
	}
	return barPathStr(ident, "s3", "uri")
}

func barDocLocation(doc map[string]any) map[string]any {
	ident := barMap(doc, "identifier")
	if uri := barPathStr(ident, "s3", "uri"); uri != "" {
		return map[string]any{"type": "S3", "s3Location": map[string]any{"uri": uri}}
	}
	return map[string]any{"type": "CUSTOM", "customDocumentLocation": map[string]any{"id": barPathStr(ident, "custom", "id")}}
}

// barDocMetadata is the retrieval metadata: source fields plus the document's
// inline attributes.
func barDocMetadata(doc map[string]any) map[string]any {
	meta := map[string]any{
		"x-amz-bedrock-kb-source-uri":     barDocID(doc),
		"x-amz-bedrock-kb-data-source-id": baStr(doc, "dataSourceId"),
	}
	for _, a := range barList(barMap(doc, "metadata"), "inlineAttributes") {
		attr, _ := a.(map[string]any)
		v := barMap(attr, "value")
		for _, field := range []string{"stringValue", "numberValue", "booleanValue", "stringListValue"} {
			if x, ok := v[field]; ok {
				meta[baStr(attr, "key")] = x
				break
			}
		}
	}
	return meta
}

// barRetrieveHits scores a knowledge base's documents against a query.
func (s *Server) barRetrieveHits(kbID, query string, n int, filter map[string]any) ([]barHit, *bedrockAgentError) {
	_, docs, ok := s.bedrockagent.LookupKnowledgeBase(kbID)
	if !ok {
		return nil, barNotFound("Knowledge base " + kbID + " was not found")
	}
	terms := barTokens(query)
	var hits []barHit
	for _, doc := range docs {
		text, _ := barDocText(doc)
		score := barScore(terms, text)
		if score == 0 {
			continue
		}
		if filter != nil {
			match, err := barFilterMatch(filter, barDocMetadata(doc))
			if err != nil {
				return nil, err
			}
			if !match {
				continue
			}
		}
		hits = append(hits, barHit{kbID: kbID, doc: doc, text: text, score: score})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return barDocID(hits[i].doc) < barDocID(hits[j].doc)
	})
	if len(hits) > n {
		hits = hits[:n]
	}
	return hits, nil
}

func barRetrievalResult(h barHit) map[string]any {
	return map[string]any{
		"content":    map[string]any{"type": "TEXT", "text": h.text},
		"location":   barDocLocation(h.doc),
		"metadata":   barDocMetadata(h.doc),
		"score":      h.score,
		"documentId": barDocID(h.doc),
	}
}

func barReference(h barHit) map[string]any {
	ref := barRetrievalResult(h)
	delete(ref, "score")
	delete(ref, "documentId")
	return ref
}

// barRetrievalConfig reads numberOfResults and filter from a
// retrievalConfiguration.vectorSearchConfiguration.
func barRetrievalConfig(cfg map[string]any) (int, map[string]any, *bedrockAgentError) {
	vector := barMap(cfg, "vectorSearchConfiguration")
	n, err := barCount(vector["numberOfResults"], "numberOfResults", 5, 1, 100)
	if err != nil {
		return 0, nil, err
	}
	return n, barMap(vector, "filter"), nil
}

func (s *Server) barRetrieveOp(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	query := barMap(in, "retrievalQuery")
	if query == nil {
		return nil, baValidation("retrievalQuery is required")
	}
	if t := baStr(query, "type"); t != "" && t != "TEXT" {
		return nil, baValidation("only TEXT retrieval queries are supported")
	}
	if err := barCheckLen("retrievalQuery.text", baStr(query, "text"), 1, 20000); err != nil {
		return nil, err
	}
	n, filter, err := barRetrievalConfig(barMap(in, "retrievalConfiguration"))
	if err != nil {
		return nil, err
	}
	hits, err := s.barRetrieveHits(baStr(in, "knowledgeBaseId"), baStr(query, "text"), n, filter)
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, len(hits))
	for i, h := range hits {
		results[i] = barRetrievalResult(h)
	}
	return map[string]any{"retrievalResults": results}, nil
}

// --- RetrieveAndGenerate ---

type barRAGSession struct {
	config  map[string]any
	created time.Time
}

// barRAG answers from the best-scoring passage. It returns the session ID,
// the answer text and the retrieved hits used as its citation.
func (s *Server) barRAG(in map[string]any) (string, string, []barHit, *bedrockAgentError) {
	text := barPathStr(in, "input", "text")
	if err := barCheckLen("input.text", text, 1, 1000); err != nil {
		return "", "", nil, err
	}
	config := barMap(in, "retrieveAndGenerateConfiguration")
	sessionID := baStr(in, "sessionId")
	st := s.bedrockagentruntime
	st.mu.Lock()
	if sessionID != "" {
		sess, ok := st.ragSessions[sessionID]
		if !ok {
			st.mu.Unlock()
			return "", "", nil, barNotFound("Session " + sessionID + " was not found")
		}
		if config == nil {
			config = sess.config
		}
	}
	st.mu.Unlock()
	if config == nil {
		return "", "", nil, baValidation("retrieveAndGenerateConfiguration is required to start a session")
	}

	var hits []barHit
	switch baStr(config, "type") {
	case "KNOWLEDGE_BASE":
		kb := barMap(config, "knowledgeBaseConfiguration")
		if err := baRequire(kb, "knowledgeBaseId", "modelArn"); err != nil {
			return "", "", nil, err
		}
		n, filter, err := barRetrievalConfig(barMap(kb, "retrievalConfiguration"))
		if err != nil {
			return "", "", nil, err
		}
		if hits, err = s.barRetrieveHits(baStr(kb, "knowledgeBaseId"), text, n, filter); err != nil {
			return "", "", nil, err
		}
	case "EXTERNAL_SOURCES":
		ext := barMap(config, "externalSourcesConfiguration")
		if err := baRequire(ext, "modelArn", "sources"); err != nil {
			return "", "", nil, err
		}
		terms := barTokens(text)
		for i, src := range barList(ext, "sources") {
			doc, body := barExternalSource(src, i)
			if score := barScore(terms, body); score > 0 {
				hits = append(hits, barHit{doc: doc, text: body, score: score})
			}
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	default:
		return "", "", nil, baValidation("retrieveAndGenerateConfiguration.type must be KNOWLEDGE_BASE or EXTERNAL_SOURCES")
	}

	answer := barNoAnswer
	if len(hits) > 0 {
		answer = hits[0].text
		if r := []rune(answer); len(r) > 2000 {
			answer = string(r[:2000])
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if sessionID == "" {
		if len(st.ragSessions) >= barMaxSessions {
			barEvictOldestRAG(st.ragSessions)
		}
		sessionID = barNewUUID()
		st.ragSessions[sessionID] = barRAGSession{config: config, created: barNowTime()}
	}
	return sessionID, answer, hits, nil
}

func barEvictOldestRAG(m map[string]barRAGSession) {
	oldest := ""
	for id, sess := range m {
		if oldest == "" || sess.created.Before(m[oldest].created) {
			oldest = id
		}
	}
	delete(m, oldest)
}

// barExternalSource turns an EXTERNAL_SOURCES source into a pseudo document.
func barExternalSource(src any, i int) (map[string]any, string) {
	m, _ := src.(map[string]any)
	if uri := barPathStr(m, "s3Location", "uri"); uri != "" {
		return map[string]any{"identifier": map[string]any{"s3": map[string]any{"uri": uri}}}, ""
	}
	bc := barMap(m, "byteContent")
	raw, _ := base64.StdEncoding.DecodeString(baStr(bc, "data"))
	id := baStr(bc, "identifier")
	if id == "" {
		id = fmt.Sprintf("source-%d", i)
	}
	return map[string]any{"identifier": map[string]any{"custom": map[string]any{"id": id}}}, string(raw)
}

func barCitations(answer string, hits []barHit) []map[string]any {
	if len(hits) == 0 {
		return []map[string]any{}
	}
	refs := make([]map[string]any, len(hits))
	for i, h := range hits {
		refs[i] = barReference(h)
	}
	return []map[string]any{{
		"generatedResponsePart": map[string]any{"textResponsePart": map[string]any{
			"text": answer, "span": map[string]any{"start": 0, "end": len([]rune(answer)) - 1},
		}},
		"retrievedReferences": refs,
	}}
}

func (s *Server) barRetrieveAndGenerateOp(req *barRequest) (any, *bedrockAgentError) {
	sessionID, answer, hits, err := s.barRAG(req.in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"sessionId": sessionID, "output": map[string]any{"text": answer}, "citations": barCitations(answer, hits)}, nil
}

func (s *Server) barRetrieveAndGenerateStream(w http.ResponseWriter, req *barRequest) *bedrockAgentError {
	sessionID, answer, hits, err := s.barRAG(req.in)
	if err != nil {
		return err
	}
	w.Header().Set("x-amzn-bedrock-knowledge-base-session-id", sessionID)
	st := brtStartStream(w)
	st.event("output", map[string]any{"text": answer})
	for _, c := range barCitations(answer, hits) {
		st.event("citation", map[string]any{"citation": c, "generatedResponsePart": c["generatedResponsePart"], "retrievedReferences": c["retrievedReferences"]})
	}
	return nil
}

// --- Rerank, GenerateQuery, OptimizePrompt, AgenticRetrieve ---

func (s *Server) barRerank(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	queries, sources := barList(in, "queries"), barList(in, "sources")
	if len(queries) != 1 {
		return nil, baValidation("queries must contain exactly 1 query")
	}
	if len(sources) < 1 || len(sources) > 1000 {
		return nil, baValidation("sources must contain between 1 and 1000 sources")
	}
	cfg := barMap(barMap(in, "rerankingConfiguration"), "bedrockRerankingConfiguration")
	if barPathStr(cfg, "modelConfiguration", "modelArn") == "" {
		return nil, baValidation("rerankingConfiguration.bedrockRerankingConfiguration.modelConfiguration.modelArn is required")
	}
	n, err := barCount(cfg["numberOfResults"], "numberOfResults", len(sources), 1, 1000)
	if err != nil {
		return nil, err
	}
	q, _ := queries[0].(map[string]any)
	qText := barPathStr(q, "textQuery", "text")
	if err := barCheckLen("textQuery.text", qText, 1, 32000); err != nil {
		return nil, err
	}
	terms := barTokens(qText)
	results := make([]map[string]any, 0, len(sources))
	for i, src := range sources {
		m, _ := src.(map[string]any)
		doc := barMap(m, "inlineDocumentSource")
		var text string
		switch baStr(doc, "type") {
		case "TEXT":
			text = barPathStr(doc, "textDocument", "text")
		case "JSON":
			raw, _ := json.Marshal(doc["jsonDocument"])
			text = string(raw)
		default:
			return nil, baValidation("inlineDocumentSource.type must be TEXT or JSON")
		}
		results = append(results, map[string]any{"index": i, "relevanceScore": barScore(terms, text), "document": doc})
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i]["relevanceScore"].(float64) > results[j]["relevanceScore"].(float64)
	})
	if len(results) > n {
		results = results[:n]
	}
	return map[string]any{"results": results}, nil
}

func (s *Server) barGenerateQuery(req *barRequest) (any, *bedrockAgentError) {
	in := req.in
	input := barMap(in, "queryGenerationInput")
	if baStr(input, "type") != "TEXT" {
		return nil, baValidation("queryGenerationInput.type must be TEXT")
	}
	text := strings.TrimSpace(baStr(input, "text"))
	if err := barCheckLen("queryGenerationInput.text", text, 1, 20000); err != nil {
		return nil, err
	}
	cfg := barMap(in, "transformationConfiguration")
	if baStr(cfg, "mode") != "TEXT_TO_SQL" {
		return nil, baValidation("transformationConfiguration.mode must be TEXT_TO_SQL")
	}
	arn := barPathStr(cfg, "textToSqlConfiguration", "knowledgeBaseConfiguration", "knowledgeBaseArn")
	if arn == "" {
		return nil, baValidation("textToSqlConfiguration.knowledgeBaseConfiguration.knowledgeBaseArn is required")
	}
	kbID := arn[strings.LastIndex(arn, "/")+1:]
	if _, _, ok := s.bedrockagent.LookupKnowledgeBase(kbID); !ok {
		return nil, barNotFound("Knowledge base " + kbID + " was not found")
	}
	// ponytail: a fixed query that names the question; no schema-aware SQL.
	question := strings.Join(strings.Fields(text), " ")
	if r := []rune(question); len(r) > 200 {
		question = string(r[:200])
	}
	sql := "-- " + question + "\nSELECT 1 AS stackyard_generated_query;"
	return map[string]any{"queries": []map[string]any{{"type": "REDSHIFT_SQL", "sql": sql}}}, nil
}

func (s *Server) barOptimizePrompt(w http.ResponseWriter, req *barRequest) *bedrockAgentError {
	in := req.in
	if baStr(in, "targetModelId") == "" {
		return baValidation("targetModelId is required")
	}
	text := barPathStr(in, "input", "textPrompt", "text")
	if err := barCheckLen("input.textPrompt.text", text, 1, 200000); err != nil {
		return err
	}
	optimized := "You are a precise assistant. " + strings.Join(strings.Fields(text), " ")
	if !strings.HasSuffix(optimized, ".") && !strings.HasSuffix(optimized, "?") {
		optimized += "."
	}
	optimized += " Answer concisely and state any assumptions."
	st := brtStartStream(w)
	st.event("analyzePromptEvent", map[string]any{"message": "Analyzed the prompt for " + baStr(in, "targetModelId") + "."})
	st.event("optimizedPromptEvent", map[string]any{"optimizedPrompt": map[string]any{"textPrompt": map[string]any{"text": optimized}}})
	return nil
}

func (s *Server) barAgenticRetrieve(w http.ResponseWriter, req *barRequest) *bedrockAgentError {
	in := req.in
	messages, retrievers := barList(in, "messages"), barList(in, "retrievers")
	if len(messages) == 0 || len(retrievers) == 0 {
		return baValidation("messages and retrievers must each contain at least one item")
	}
	if in["agenticRetrieveConfiguration"] == nil {
		return baValidation("agenticRetrieveConfiguration is required")
	}
	query := ""
	for _, m := range messages {
		msg, _ := m.(map[string]any)
		if baStr(msg, "role") == "user" {
			query = barPathStr(msg, "content", "text")
		}
	}
	if err := barCheckLen("user message text", query, 1, 20000); err != nil {
		return err
	}
	var hits []barHit
	var ids []string
	for _, rv := range retrievers {
		m, _ := rv.(map[string]any)
		kbID := barPathStr(m, "configuration", "knowledgeBase", "knowledgeBaseId")
		if kbID == "" {
			return baValidation("each retriever needs configuration.knowledgeBase.knowledgeBaseId")
		}
		found, err := s.barRetrieveHits(kbID, query, 10, nil)
		if err != nil {
			return err
		}
		hits = append(hits, found...)
		ids = append(ids, kbID)
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	results := make([]map[string]any, len(hits))
	for i, h := range hits {
		results[i] = map[string]any{
			"content":         map[string]any{"mimeType": "text/plain", "text": h.text},
			"metadata":        barDocMetadata(h.doc),
			"sourceRetriever": map[string]any{"identifier": h.kbID},
		}
	}
	st := brtStartStream(w)
	now := barNowTime().UnixMilli()
	st.event("traceEvent", map[string]any{"id": barNewUUID(), "timestamp": now, "attributes": map[string]any{
		"message": "Retrieved from " + strings.Join(ids, ", "), "status": "SUCCEEDED", "step": "Retrieval",
	}})
	result := map[string]any{"results": results}
	if baBool(in["generateResponse"]) {
		answer := barNoAnswer
		citations := []map[string]any{}
		if len(hits) > 0 {
			answer = hits[0].text
			citations = append(citations, map[string]any{"startIndex": 0, "endIndex": len([]rune(answer)), "references": []map[string]any{{"resultIndex": 0}}})
		}
		result["generatedResponse"] = map[string]any{"answer": answer, "citations": citations}
		st.event("responseEvent", map[string]any{"text": answer})
	}
	st.event("result", result)
	return nil
}

// --- document content and ACLs ---

type barDocToken struct {
	kbID, docID string
	expires     time.Time
}

// barFindDocument finds an ingested document by its runtime ID in a data source.
func (s *Server) barFindDocument(in map[string]any) (map[string]any, *bedrockAgentError) {
	kbID, dsID, docID := baStr(in, "knowledgeBaseId"), baStr(in, "dataSourceId"), baStr(in, "documentId")
	if err := barCheckLen("documentId", docID, 1, 1825); err != nil {
		return nil, err
	}
	_, docs, ok := s.bedrockagent.LookupKnowledgeBase(kbID)
	if !ok {
		return nil, barNotFound("Knowledge base " + kbID + " was not found")
	}
	for _, doc := range docs {
		if baStr(doc, "dataSourceId") == dsID && barDocID(doc) == docID {
			return doc, nil
		}
	}
	return nil, barNotFound("Document " + docID + " was not found in data source " + dsID)
}

func (s *Server) barGetDocumentContent(req *barRequest) (any, *bedrockAgentError) {
	doc, err := s.barFindDocument(req.in)
	if err != nil {
		return nil, err
	}
	text, mime := barDocText(doc)
	token := baNewID("abcdefghijklmnopqrstuvwxyz0123456789", 32)
	st := s.bedrockagentruntime
	st.mu.Lock()
	now := barNowTime()
	for k, t := range st.docTokens {
		if now.After(t.expires) || len(st.docTokens) >= barMaxDocTokens {
			delete(st.docTokens, k)
		}
	}
	st.docTokens[token] = barDocToken{kbID: baStr(doc, "knowledgeBaseId"), docID: baStr(doc, "dataSourceId") + "/" + barDocID(doc), expires: now.Add(15 * time.Minute)}
	st.mu.Unlock()
	scheme := "http"
	if req.r.TLS != nil {
		scheme = "https"
	}
	return map[string]any{
		"mimeType":              mime,
		"documentContentLength": len(text),
		"presignedUrl":          scheme + "://" + req.r.Host + barDocumentPath + token,
	}, nil
}

// barServeDocument serves a GetDocumentContent link. Document bytes are
// caller-supplied, so the content type is a constant non-HTML type set in the
// same function as the body write.
func (s *Server) barServeDocument(w http.ResponseWriter, token string) {
	st := s.bedrockagentruntime
	st.mu.Lock()
	t, ok := st.docTokens[token]
	st.mu.Unlock()
	if !ok || barNowTime().After(t.expires) {
		respondBedrockAgentError(w, &bedrockAgentError{status: http.StatusForbidden, code: "AccessDenied", msg: "the document link is invalid or expired"})
		return
	}
	dsID, docID, _ := strings.Cut(t.docID, "/")
	doc, err := s.barFindDocument(map[string]any{"knowledgeBaseId": t.kbID, "dataSourceId": dsID, "documentId": docID})
	if err != nil {
		respondBedrockAgentError(w, err)
		return
	}
	text, _ := barDocText(doc)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(text))
}

func (s *Server) barCheckDocumentAcl(req *barRequest) (any, *bedrockAgentError) {
	if barPathStr(req.in, "userContext", "userId") == "" {
		return nil, baValidation("userContext.userId is required")
	}
	if _, err := s.barFindDocument(req.in); err != nil {
		return nil, err
	}
	// Ingestion carries no ACL, so every user may read every document.
	return map[string]any{"hasAccess": true}, nil
}

func (s *Server) barGetDocumentAcl(req *barRequest) (any, *bedrockAgentError) {
	if _, err := s.barFindDocument(req.in); err != nil {
		return nil, err
	}
	return map[string]any{"documentAcl": map[string]any{}}, nil
}

// --- metadata filters ---

func barFilterMatch(f map[string]any, meta map[string]any) (bool, *bedrockAgentError) {
	for op, raw := range f {
		switch op {
		case "andAll", "orAll":
			list, _ := raw.([]any)
			if len(list) < 2 {
				return false, baValidation(op + " needs at least 2 filters")
			}
			all := op == "andAll"
			for _, sub := range list {
				m, _ := sub.(map[string]any)
				ok, err := barFilterMatch(m, meta)
				if err != nil {
					return false, err
				}
				if ok != all {
					return !all, nil
				}
			}
			return all, nil
		}
		attr, _ := raw.(map[string]any)
		key := baStr(attr, "key")
		if key == "" {
			return false, baValidation(op + ".key is required")
		}
		have, want := meta[key], attr["value"]
		switch op {
		case "equals":
			return barSame(have, want), nil
		case "notEquals":
			return !barSame(have, want), nil
		case "in", "notIn":
			list, _ := want.([]any)
			found := false
			for _, v := range list {
				found = found || barSame(have, v)
			}
			return found == (op == "in"), nil
		case "listContains":
			list, _ := have.([]any)
			for _, v := range list {
				if barSame(v, want) {
					return true, nil
				}
			}
			return false, nil
		case "startsWith", "stringContains":
			hs, ws := fmt.Sprint(have), fmt.Sprint(want)
			if have == nil {
				return false, nil
			}
			if op == "startsWith" {
				return strings.HasPrefix(hs, ws), nil
			}
			return strings.Contains(hs, ws), nil
		case "greaterThan", "greaterThanOrEquals", "lessThan", "lessThanOrEquals":
			h, ok1 := barNumber(have)
			v, ok2 := barNumber(want)
			if !ok1 || !ok2 {
				return false, nil
			}
			switch op {
			case "greaterThan":
				return h > v, nil
			case "greaterThanOrEquals":
				return h >= v, nil
			case "lessThan":
				return h < v, nil
			}
			return h <= v, nil
		default:
			return false, baValidation("unsupported filter " + op)
		}
	}
	return true, nil
}

func barNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case float64:
		return n, true
	}
	return 0, false
}

func barSame(a, b any) bool {
	if x, ok := barNumber(a); ok {
		y, ok := barNumber(b)
		return ok && x == y
	}
	return a != nil && fmt.Sprint(a) == fmt.Sprint(b)
}
