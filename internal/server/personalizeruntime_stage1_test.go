package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Runtime APIs answer only for the recipe type a campaign serves.
func TestPersonalizeRuntimeStage1RecipeTypes(t *testing.T) {
	srv, ts := newPersonalizeTestServer(t)
	fx := pszFixture(t, srv, ts)
	cases := []struct {
		name, path, body string
		status           int
	}{
		{"recommendations", "/recommendations", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","numResults":2}`, fx["campaign"]), http.StatusOK},
		{"no user", "/recommendations", fmt.Sprintf(`{"campaignArn":%q}`, fx["campaign"]), http.StatusBadRequest},
		{"both targets", "/recommendations", fmt.Sprintf(`{"campaignArn":%q,"recommenderArn":%q,"userId":"u1"}`, fx["campaign"], fx["campaign"]), http.StatusBadRequest},
		{"too many results", "/recommendations", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","numResults":501}`, fx["campaign"]), http.StatusBadRequest},
		{"metadata off", "/recommendations", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","metadataColumns":{"ITEMS":["NOPE"]}}`, fx["campaign"]), http.StatusBadRequest},
		{"ranking on user-personalization", "/personalize-ranking", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1","inputList":["i1"]}`, fx["campaign"]), http.StatusBadRequest},
		{"actions on user-personalization", "/action-recommendations", fmt.Sprintf(`{"campaignArn":%q,"userId":"u1"}`, fx["campaign"]), http.StatusBadRequest},
		{"missing campaign", "/recommendations", `{"campaignArn":"arn:aws:personalize:us-east-1:123456789012:campaign/none","userId":"u1"}`, http.StatusNotFound},
	}
	for _, c := range cases {
		status, out := pszREST(t, ts, c.path, c.body)
		if status != c.status {
			t.Errorf("%s: want %d, got %d %v", c.name, c.status, status, out)
		}
	}
	// A ranking campaign orders the given list and nothing else.
	sol := pszOK(t, ts, "CreateSolution", fmt.Sprintf(`{"name":"rank","datasetGroupArn":%q,"recipeArn":"arn:aws:personalize:::recipe/aws-personalized-ranking"}`, fx["group"]))["solutionArn"].(string)
	pszSettle(t, ts, "DescribeSolution", "solution", "solutionArn", sol, "ACTIVE")
	ver := pszOK(t, ts, "CreateSolutionVersion", fmt.Sprintf(`{"solutionArn":%q}`, sol))["solutionVersionArn"].(string)
	pszSettle(t, ts, "DescribeSolutionVersion", "solutionVersion", "solutionVersionArn", ver, "ACTIVE")
	camp := pszOK(t, ts, "CreateCampaign", fmt.Sprintf(`{"name":"rank","solutionVersionArn":%q}`, ver))["campaignArn"].(string)
	pszSettle(t, ts, "DescribeCampaign", "campaign", "campaignArn", camp, "ACTIVE")
	status, out := pszREST(t, ts, "/personalize-ranking", fmt.Sprintf(`{"campaignArn":%q,"userId":"u2","inputList":["i3","i1","zz","i1"]}`, camp))
	if status != http.StatusOK {
		t.Fatalf("ranking: %d %v", status, out)
	}
	ids := pszItemIDs(t, out, "personalizedRanking")
	raw, _ := json.Marshal(out)
	if len(ids) != 3 || !strings.Contains(strings.Join(ids, ","), "zz") || !strings.Contains(string(raw), "RID-") {
		t.Fatalf("ranking should cover each distinct input once: %s", raw)
	}
	total := 0.0
	for _, v := range out["personalizedRanking"].([]any) {
		total += v.(map[string]any)["score"].(float64)
	}
	if total < 0.99 || total > 1.01 {
		t.Fatalf("scores should sum to 1: %v", total)
	}
}
