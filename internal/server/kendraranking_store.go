package server

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Rescore execution plans, their tags, and Rescore itself. A new plan reports
// CREATING on its first read and ACTIVE after; an update reports UPDATING once.
// Rescore ranks documents by how many query terms their title (weighted
// double) and body contain; documents sharing a GroupId share the group's best
// relevance, and the caller's OriginalScore breaks ties.

const (
	kdrMaxPlans        = 50
	kdrMaxTags         = 200
	kdrMaxDocuments    = 500
	kdrDefaultPageSize = 50
	kdrTitleWeight     = 2.0
	kdrStatusActive    = "ACTIVE"
)

type kdrPlan struct {
	id, name, description string
	capacity              float64
	created, updated      time.Time
	pending               string // CREATING or UPDATING until the next read
	tags                  map[string]string
}

func (p *kdrPlan) arn() string {
	return fmt.Sprintf("arn:aws:kendra-ranking:%s:%s:rescore-execution-plan/%s", kdrRegion, kdrAccountID, p.id)
}

// readStatus returns the status a read reports and settles a pending one.
func (p *kdrPlan) readStatus() string {
	if s := p.pending; s != "" {
		p.pending = ""
		return s
	}
	return kdrStatusActive
}

type kdrStore struct {
	mu     sync.Mutex
	plans  map[string]*kdrPlan
	tokens map[string]string // ClientToken -> plan id
}

func newKendraRankingStore() *kdrStore {
	return &kdrStore{plans: map[string]*kdrPlan{}, tokens: map[string]string{}}
}

func (st *kdrStore) plan(id string) (*kdrPlan, *kdrErr) {
	p := st.plans[id]
	if p == nil {
		return nil, kdrNotFound("Rescore execution plan %s doesn't exist.", id)
	}
	return p, nil
}

func kdrCapacity(in map[string]any) (float64, bool) {
	c, _ := in["CapacityUnits"].(map[string]any)
	if c == nil {
		return 0, false
	}
	f, _ := c["RescoreCapacityUnits"].(float64)
	return f, true
}

func kdrTags(list []any) (map[string]string, *kdrErr) {
	out := map[string]string{}
	for _, t := range list {
		tag, _ := t.(map[string]any)
		k := kdrStr(tag, "Key")
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, kdrValidation("Tag keys can't start with aws:.")
		}
		out[k] = kdrStr(tag, "Value")
	}
	return out, nil
}

func (st *kdrStore) createPlan(in map[string]any) (map[string]any, *kdrErr) {
	tags, err := kdrTags(kdrList(in, "Tags"))
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	token := kdrStr(in, "ClientToken")
	if id, ok := st.tokens[token]; ok && token != "" {
		if p := st.plans[id]; p != nil {
			return map[string]any{"Id": p.id, "Arn": p.arn()}, nil
		}
	}
	if len(st.plans) >= kdrMaxPlans {
		return nil, &kdrErr{http.StatusPaymentRequired, "ServiceQuotaExceededException", fmt.Sprintf("You can have at most %d rescore execution plans.", kdrMaxPlans)}
	}
	capacity, _ := kdrCapacity(in)
	now := time.Now().UTC()
	p := &kdrPlan{
		id: uuid.NewString(), name: kdrStr(in, "Name"), description: kdrStr(in, "Description"),
		capacity: capacity, created: now, updated: now, pending: "CREATING", tags: tags,
	}
	st.plans[p.id] = p
	if token != "" {
		st.tokens[token] = p.id
	}
	return map[string]any{"Id": p.id, "Arn": p.arn()}, nil
}

func (st *kdrStore) describePlan(in map[string]any) (map[string]any, *kdrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.plan(kdrStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"Id": p.id, "Arn": p.arn(), "Name": p.name, "CapacityUnits": map[string]any{"RescoreCapacityUnits": p.capacity},
		"CreatedAt": kdrEpoch(p.created), "UpdatedAt": kdrEpoch(p.updated), "Status": p.readStatus(),
	}
	if p.description != "" {
		out["Description"] = p.description
	}
	return out, nil
}

func kdrEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func (st *kdrStore) listPlans(in map[string]any) (map[string]any, *kdrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	plans := make([]*kdrPlan, 0, len(st.plans))
	for _, p := range st.plans {
		plans = append(plans, p)
	}
	sort.Slice(plans, func(i, j int) bool {
		if !plans[i].created.Equal(plans[j].created) {
			return plans[i].created.Before(plans[j].created)
		}
		return plans[i].id < plans[j].id
	})
	start := 0
	if tok := kdrStr(in, "NextToken"); tok != "" {
		raw, err := base64.RawURLEncoding.DecodeString(tok)
		n, perr := strconv.ParseInt(strings.TrimPrefix(string(raw), "o:"), 10, 32)
		if err != nil || perr != nil || !strings.HasPrefix(string(raw), "o:") || n < 0 || int(n) > len(plans) {
			return nil, kdrValidation("The NextToken is not valid.")
		}
		start = int(n)
	}
	size := kdrDefaultPageSize
	if f, ok := in["MaxResults"].(float64); ok {
		size = int(f)
	}
	end := min(len(plans), start+size)
	items := []any{}
	for _, p := range plans[start:end] {
		items = append(items, map[string]any{
			"Id": p.id, "Name": p.name, "CreatedAt": kdrEpoch(p.created), "UpdatedAt": kdrEpoch(p.updated), "Status": p.readStatus(),
		})
	}
	out := map[string]any{"SummaryItems": items}
	if end < len(plans) {
		out["NextToken"] = base64.RawURLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(end)))
	}
	return out, nil
}

func (st *kdrStore) updatePlan(in map[string]any) (map[string]any, *kdrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.plan(kdrStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	if v, ok := in["Name"].(string); ok {
		p.name = v
	}
	if v, ok := in["Description"].(string); ok {
		p.description = v
	}
	if c, ok := kdrCapacity(in); ok {
		p.capacity = c
	}
	p.updated, p.pending = time.Now().UTC(), "UPDATING"
	return nil, nil
}

func (st *kdrStore) deletePlan(in map[string]any) (map[string]any, *kdrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.plan(kdrStr(in, "Id"))
	if err != nil {
		return nil, err
	}
	delete(st.plans, p.id)
	for tok, id := range st.tokens {
		if id == p.id {
			delete(st.tokens, tok)
		}
	}
	return nil, nil
}

// ---- rescore ----

type kdrDoc struct {
	id, group string
	original  float64
	relevance float64
}

func kdrDocTokens(doc map[string]any, text, tokenized string) []string {
	if s := kdrStr(doc, text); s != "" {
		return csdTokens(s)
	}
	var out []string
	for _, t := range kdrList(doc, tokenized) {
		if s, ok := t.(string); ok {
			out = append(out, strings.ToLower(s))
		}
	}
	return out
}

func kdrOverlap(query map[string]bool, words []string) float64 {
	seen := map[string]bool{}
	for _, w := range words {
		if query[w] {
			seen[w] = true
		}
	}
	return float64(len(seen))
}

func (st *kdrStore) rescore(in map[string]any) (map[string]any, *kdrErr) {
	query := map[string]bool{}
	for _, w := range csdTokens(kdrStr(in, "SearchQuery")) {
		query[w] = true
	}
	list := kdrList(in, "Documents")
	if len(list) > kdrMaxDocuments {
		return nil, kdrValidation("You can rescore at most %d documents.", kdrMaxDocuments)
	}
	docs := make([]*kdrDoc, 0, len(list))
	seen := map[string]bool{}
	best := map[string]float64{}
	for _, item := range list {
		doc, _ := item.(map[string]any)
		id := kdrStr(doc, "Id")
		if seen[id] {
			return nil, kdrValidation("Document Id %s appears more than once.", id)
		}
		seen[id] = true
		if (doc["Title"] != nil && doc["TokenizedTitle"] != nil) || (doc["Body"] != nil && doc["TokenizedBody"] != nil) {
			return nil, kdrValidation("Document %s: give Title or TokenizedTitle, and Body or TokenizedBody, not both.", id)
		}
		if doc["Title"] == nil && doc["TokenizedTitle"] == nil && doc["Body"] == nil && doc["TokenizedBody"] == nil {
			return nil, kdrValidation("Document %s needs a title or a body.", id)
		}
		d := &kdrDoc{id: id, group: kdrStr(doc, "GroupId")}
		d.original, _ = doc["OriginalScore"].(float64)
		if len(query) > 0 {
			hits := kdrTitleWeight*kdrOverlap(query, kdrDocTokens(doc, "Title", "TokenizedTitle")) + kdrOverlap(query, kdrDocTokens(doc, "Body", "TokenizedBody"))
			d.relevance = hits / ((kdrTitleWeight + 1) * float64(len(query)))
		}
		if d.group != "" {
			best[d.group] = math.Max(best[d.group], d.relevance)
		}
		docs = append(docs, d)
	}
	for _, d := range docs {
		if d.group != "" {
			d.relevance = best[d.group]
		}
	}
	st.mu.Lock()
	_, err := st.plan(kdrStr(in, "RescoreExecutionPlanId"))
	st.mu.Unlock()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(docs, func(i, j int) bool {
		if docs[i].relevance != docs[j].relevance {
			return docs[i].relevance > docs[j].relevance
		}
		return docs[i].original > docs[j].original
	})
	items := make([]any, len(docs))
	for i, d := range docs {
		items[i] = map[string]any{"DocumentId": d.id, "Score": math.Round(d.relevance*1e4) / 1e4}
	}
	return map[string]any{"RescoreId": uuid.NewString(), "ResultItems": items}, nil
}

// ---- tags ----

func (st *kdrStore) planByARN(arn string) (*kdrPlan, *kdrErr) {
	id, ok := strings.CutPrefix(arn, fmt.Sprintf("arn:aws:kendra-ranking:%s:%s:rescore-execution-plan/", kdrRegion, kdrAccountID))
	if p := st.plans[id]; ok && p != nil {
		return p, nil
	}
	return nil, &kdrErr{http.StatusNotFound, "ResourceUnavailableException", "Resource " + arn + " doesn't exist."}
}

func (st *kdrStore) tagResource(in map[string]any) (map[string]any, *kdrErr) {
	tags, err := kdrTags(kdrList(in, "Tags"))
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.planByARN(kdrStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	merged := map[string]string{}
	for k, v := range p.tags {
		merged[k] = v
	}
	for k, v := range tags {
		merged[k] = v
	}
	if len(merged) > kdrMaxTags {
		return nil, kdrValidation("A resource can have at most %d tags.", kdrMaxTags)
	}
	p.tags = merged
	return nil, nil
}

func (st *kdrStore) untagResource(in map[string]any) (map[string]any, *kdrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.planByARN(kdrStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	next := map[string]string{}
	for k, v := range p.tags {
		next[k] = v
	}
	for _, k := range kdrList(in, "TagKeys") {
		if s, ok := k.(string); ok {
			delete(next, s)
		}
	}
	p.tags = next
	return nil, nil
}

func (st *kdrStore) listTags(in map[string]any) (map[string]any, *kdrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	p, err := st.planByARN(kdrStr(in, "ResourceARN"))
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(p.tags))
	for k := range p.tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tags := []any{}
	for _, k := range keys {
		tags = append(tags, map[string]any{"Key": k, "Value": p.tags[k]})
	}
	return map[string]any{"Tags": tags}, nil
}
