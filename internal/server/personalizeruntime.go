package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Personalize Runtime: REST JSON, signed personalize. Campaigns and
// recommenders answer from their dataset group's current data (see
// personalize_rank.go).

var pszrtPaths = map[string]string{
	"/recommendations": "GetRecommendations", "/personalize-ranking": "GetPersonalizedRanking",
	"/action-recommendations": "GetActionRecommendations",
}

func (s *Server) handlePersonalizeRuntimeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	op, ok := pszRESTCandidate(r, "personalize-runtime.", pszrtPaths)
	if !ok {
		return false
	}
	if _, known := personalizeRuntimeOperationByName[op]; !known {
		return false
	}
	s.pszServeREST(w, r, awsmodels.PersonalizeRuntimeModel(), op, func(in map[string]any) (map[string]any, *pszErr) {
		return s.personalize.runtime(op, in)
	})
	return true
}

// pszServeREST runs one Personalize Events or Runtime request.
func (s *Server) pszServeREST(w http.ResponseWriter, r *http.Request, model *awsmodels.Model, op string, h func(map[string]any) (map[string]any, *pszErr)) {
	// A declared length bounds what net/http reads; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > pszMaxBody {
		respondPSZRESTError(w, http.StatusBadRequest, pszErrInvalid, "request body must declare a Content-Length of at most 1 MiB")
		return
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "personalize"); !ok {
		respondPSZRESTError(w, status, code, msg)
		return
	}
	in, perr := pszDecode(r, model, op)
	if perr == nil {
		var out map[string]any
		if out, perr = h(in); perr == nil {
			if out == nil {
				out = map[string]any{}
			}
			body, _ := json.Marshal(model.Conform(model.Operations[op].Output, out))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
			return
		}
	}
	status := map[string]int{pszErrNotFound: http.StatusNotFound, pszErrInUse: http.StatusConflict}[perr.code]
	if status == 0 {
		status = http.StatusBadRequest
	}
	respondPSZRESTError(w, status, perr.code, perr.msg)
}

// pszServing is what a runtime request is answered with.
type pszServing struct {
	arn, groupARN string
	recipe        *pszRecipe
	metadata      bool // enableMetadataWithRecommendations
}

func (st *personalizeStore) serving(campaignARN, recommenderARN string) (*pszServing, *pszErr) {
	switch {
	case (campaignARN == "") == (recommenderARN == ""):
		return nil, pszInvalid("Specify exactly one of campaignArn and recommenderArn.")
	case campaignARN != "":
		c, err := st.get("campaign", campaignARN)
		if err != nil {
			return nil, err
		}
		if c.status != pszStatusActive {
			return nil, pszInvalid("Campaign %s is %s, not ACTIVE.", c.arn, c.status)
		}
		return &pszServing{c.fields["solutionVersionArn"].(string), c.rel["datasetGroupArn"], c.recipe,
			pszBool(pszMap(c.fields, "campaignConfig"), "enableMetadataWithRecommendations", false)}, nil
	}
	rec, err := st.get("recommender", recommenderARN)
	if err != nil {
		return nil, err
	}
	if rec.status != pszStatusActive {
		return nil, pszInvalid("Recommender %s is %s, not ACTIVE.", rec.arn, rec.status)
	}
	return &pszServing{rec.arn, rec.rel["datasetGroupArn"], rec.recipe,
		pszBool(pszMap(rec.fields, "recommenderConfig"), "enableMetadataWithRecommendations", false)}, nil
}

func (st *personalizeStore) runtime(op string, in map[string]any) (map[string]any, *pszErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	sv, err := st.serving(pszStr(in, "campaignArn"), pszStr(in, "recommenderArn"))
	if err != nil {
		return nil, err
	}
	want := map[string][]string{
		"GetRecommendations":       {pszUserPersonalization, pszRelatedItems, pszPopularItems},
		"GetPersonalizedRanking":   {pszPersonalizedRanking},
		"GetActionRecommendations": {pszPersonalizedActions},
	}[op]
	if !pszContains(want, sv.recipe.recipeType) {
		return nil, pszInvalid("%s serves a %s recipe; %s doesn't apply.", sv.arn, sv.recipe.recipeType, op)
	}
	f, err := st.filterFor(pszStr(in, "filterArn"), sv.groupARN)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for k, v := range pszMap(in, "filterValues") {
		values[k], _ = v.(string)
	}
	g := st.group(sv.groupARN)
	rid := "RID-" + uuid.NewString()
	user := pszStr(in, "userId")
	switch op {
	case "GetActionRecommendations":
		n := pszInt(in, "numResults", 5)
		if n < 1 || n > pszMaxActionResults {
			return nil, pszInvalid("numResults must be between 1 and %d.", pszMaxActionResults)
		}
		scored, err := g.recommendActions(sv.arn, user, n, f, values)
		if err != nil {
			return nil, err
		}
		list := make([]any, len(scored))
		for i, s := range scored {
			list[i] = map[string]any{"actionId": s.id, "score": s.score}
		}
		return map[string]any{"actionList": list, "recommendationId": rid}, nil
	case "GetPersonalizedRanking":
		input := pszStrings(in["inputList"])
		if len(input) == 0 || len(input) > pszMaxRecommendations {
			return nil, pszInvalid("inputList must have between 1 and %d items.", pszMaxRecommendations)
		}
		scored, err := g.rank(sv.arn, user, input, f, values)
		if err != nil {
			return nil, err
		}
		items, err := st.itemList(scored, g, sv, in, "", 0)
		if err != nil {
			return nil, err
		}
		return map[string]any{"personalizedRanking": items, "recommendationId": rid}, nil
	}
	item := pszStr(in, "itemId")
	switch {
	case sv.recipe.recipeType == pszUserPersonalization && user == "":
		return nil, pszInvalid("userId is required for %s recipes.", sv.recipe.recipeType)
	case sv.recipe.recipeType == pszRelatedItems && item == "":
		return nil, pszInvalid("itemId is required for %s recipes.", sv.recipe.recipeType)
	}
	n := pszInt(in, "numResults", 25)
	if n < 1 || n > pszMaxRecommendations {
		return nil, pszInvalid("numResults must be between 1 and %d.", pszMaxRecommendations)
	}
	scored, err := g.recommendItems(sv.arn, sv.recipe.recipeType, user, item, n, f, values)
	if err != nil {
		return nil, err
	}
	promoName, promoCount := "", 0
	if promos := pszList(in, "promotions"); len(promos) > 0 {
		p, _ := promos[0].(map[string]any)
		pf, err := st.filterFor(pszStr(p, "filterArn"), sv.groupARN)
		if err != nil {
			return nil, err
		}
		pv := map[string]string{}
		for k, v := range pszMap(p, "filterValues") {
			pv[k], _ = v.(string)
		}
		promoted, err := g.recommendItems(sv.arn+"#promotion", sv.recipe.recipeType, user, item, n, pf, pv)
		if err != nil {
			return nil, err
		}
		promoName = pszStr(p, "name")
		promoCount = min(len(promoted), (n*pszInt(p, "percentPromotedItems", 1)+99)/100)
		scored = pszMergePromoted(scored, promoted[:promoCount], n)
	}
	items, err := st.itemList(scored, g, sv, in, promoName, promoCount)
	if err != nil {
		return nil, err
	}
	return map[string]any{"itemList": items, "recommendationId": rid}, nil
}

// pszMergePromoted puts promoted items first and fills up to n with the rest.
func pszMergePromoted(scored, promoted []pszScored, n int) []pszScored {
	seen := map[string]bool{}
	out := append([]pszScored(nil), promoted...)
	for _, p := range promoted {
		seen[p.id] = true
	}
	for _, s := range scored {
		if len(out) >= n {
			break
		}
		if !seen[s.id] {
			out = append(out, s)
		}
	}
	return out
}

// itemList shapes scored items, with metadata columns when requested.
func (st *personalizeStore) itemList(scored []pszScored, g *pszGroup, sv *pszServing, in map[string]any, promoName string, promoCount int) ([]any, *pszErr) {
	var cols []string
	if mc := pszMap(in, "metadataColumns"); len(mc) > 0 {
		if !sv.metadata {
			return nil, pszInvalid("Enable enableMetadataWithRecommendations on %s to request metadataColumns.", sv.arn)
		}
		for typ, v := range mc {
			if typ != "ITEMS" {
				return nil, pszInvalid("metadataColumns supports the ITEMS dataset only.")
			}
			for _, c := range pszStrings(v) {
				if g.items == nil || pszField(g.items.fields, c) == "" {
					return nil, pszInvalid("%s isn't a column of the Items dataset.", c)
				}
				cols = append(cols, c)
			}
		}
	}
	out := make([]any, len(scored))
	for i, s := range scored {
		entry := map[string]any{"itemId": s.id, "score": s.score}
		if i < promoCount {
			entry["promotionName"] = promoName
		}
		if cols != nil {
			meta := map[string]any{}
			if e := g.items.entities[s.id]; e != nil {
				for _, c := range cols {
					if v, ok := e.props[strings.ToUpper(pszField(g.items.fields, c))]; ok {
						meta[c] = v
					}
				}
			}
			entry["metadata"] = meta
		}
		out[i] = entry
	}
	return out, nil
}
