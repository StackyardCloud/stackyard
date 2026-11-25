package server

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"strings"
	"time"
)

// The AWS-managed recipes, with the algorithm and feature transformation
// behind each. Recipe types decide which runtime API a campaign serves.

const (
	pszUserPersonalization = "USER_PERSONALIZATION"
	pszPersonalizedRanking = "PERSONALIZED_RANKING"
	pszRelatedItems        = "RELATED_ITEMS"
	pszPopularItems        = "POPULAR_ITEMS"
	pszUserSegmentation    = "USER_SEGMENTATION"
	pszPersonalizedActions = "PERSONALIZED_ACTIONS"
	pszCatalogBase         = "arn:aws:personalize:::"
)

type pszRecipe struct {
	name, recipeType, domain, description string
}

func (p *pszRecipe) arn() string          { return pszCatalogBase + "recipe/" + p.name }
func (p *pszRecipe) algorithmARN() string { return pszCatalogBase + "algorithm/" + p.name }
func (p *pszRecipe) featureARN() string   { return pszCatalogBase + "feature-transformation/" + p.name }

var pszRecipes = []*pszRecipe{
	{"aws-item-affinity", pszUserSegmentation, "", "Users most likely to interact with each item."},
	{"aws-item-attribute-affinity", pszUserSegmentation, "", "Users most likely to interact with items that have given attributes."},
	{"aws-next-best-action", pszPersonalizedActions, "", "The actions each user is most likely to take."},
	{"aws-personalized-ranking", pszPersonalizedRanking, "", "Ranks a list of items for a user."},
	{"aws-personalized-ranking-v2", pszPersonalizedRanking, "", "Ranks a list of items for a user (v2)."},
	{"aws-popularity-count", pszPopularItems, "", "The most popular items, by interaction count."},
	{"aws-semantic-similarity", pszRelatedItems, "", "Items semantically similar to an item."},
	{"aws-similar-items", pszRelatedItems, "", "Items similar to an item, from interactions and metadata."},
	{"aws-sims", pszRelatedItems, "", "Items that co-occur with an item in user histories."},
	{"aws-trending-now", pszPopularItems, "", "Items whose popularity is rising fastest."},
	{"aws-user-personalization", pszUserPersonalization, "", "Personalized item recommendations for a user."},
	{"aws-user-personalization-v2", pszUserPersonalization, "", "Personalized item recommendations for a user (v2)."},
	{"aws-ecomm-customers-who-viewed-x-also-viewed", pszRelatedItems, "ECOMMERCE", "Items customers viewed together with an item."},
	{"aws-ecomm-frequently-bought-together", pszRelatedItems, "ECOMMERCE", "Items frequently bought with an item."},
	{"aws-ecomm-popular-items-by-purchases", pszPopularItems, "ECOMMERCE", "The most purchased items."},
	{"aws-ecomm-popular-items-by-views", pszPopularItems, "ECOMMERCE", "The most viewed items."},
	{"aws-ecomm-recommended-for-you", pszUserPersonalization, "ECOMMERCE", "Personalized recommendations for a customer."},
	{"aws-vod-because-you-watched-x", pszRelatedItems, "VIDEO_ON_DEMAND", "Videos similar to one a user watched."},
	{"aws-vod-more-like-x", pszRelatedItems, "VIDEO_ON_DEMAND", "Videos like a given video."},
	{"aws-vod-most-popular", pszPopularItems, "VIDEO_ON_DEMAND", "The most watched videos."},
	{"aws-vod-top-picks", pszUserPersonalization, "VIDEO_ON_DEMAND", "Personalized video picks for a user."},
	{"aws-vod-trending-now", pszPopularItems, "VIDEO_ON_DEMAND", "Videos trending now."},
}

// pszCatalogTime is when the managed recipes were published.
var pszCatalogTime = pszEpoch(time.Date(2019, 6, 10, 0, 0, 0, 0, time.UTC))

func pszFindRecipe(arn string) *pszRecipe {
	for _, p := range pszRecipes {
		if p.arn() == arn {
			return p
		}
	}
	return nil
}

func pszRecipeView(p *pszRecipe) map[string]any {
	out := map[string]any{
		"name": p.name, "recipeArn": p.arn(), "status": pszStatusActive, "description": p.description,
		"recipeType": p.recipeType, "creationDateTime": pszCatalogTime, "lastUpdatedDateTime": pszCatalogTime,
	}
	if p.domain == "" {
		out["algorithmArn"], out["featureTransformationArn"] = p.algorithmARN(), p.featureARN()
	} else {
		out["domain"] = p.domain
	}
	return out
}

func pszDescribeRecipe(arn string) (map[string]any, *pszErr) {
	p := pszFindRecipe(arn)
	if p == nil {
		return nil, pszMissing(arn)
	}
	return map[string]any{"recipe": pszRecipeView(p)}, nil
}

func pszListRecipes(in map[string]any) (map[string]any, *pszErr) {
	domain := pszStr(in, "domain")
	var matches []*pszRecipe
	for _, p := range pszRecipes {
		if domain == "" || p.domain == domain {
			matches = append(matches, p)
		}
	}
	page, next, err := pszPage(len(matches), pszStr(in, "nextToken"), pszMax(in))
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, page[1]-page[0])
	for _, p := range matches[page[0]:page[1]] {
		items = append(items, pszRecipeView(p))
	}
	out := map[string]any{"recipes": items}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func pszDescribeAlgorithm(arn string) (map[string]any, *pszErr) {
	for _, p := range pszRecipes {
		if p.domain == "" && p.algorithmARN() == arn {
			return map[string]any{"algorithm": map[string]any{
				"name": p.name, "algorithmArn": arn,
				"algorithmImage":         map[string]any{"name": p.name, "dockerURI": "stackyard/personalize/" + p.name + ":latest"},
				"defaultHyperParameters": map[string]any{"hidden_dimension": "149", "bptt": "32", "recency_mask": "true"},
				"defaultHyperParameterRanges": map[string]any{
					"integerHyperParameterRanges":     []any{map[string]any{"name": "hidden_dimension", "minValue": 32, "maxValue": 256, "isTunable": true}},
					"continuousHyperParameterRanges":  []any{map[string]any{"name": "exploration_weight", "minValue": 0.0, "maxValue": 1.0, "isTunable": false}},
					"categoricalHyperParameterRanges": []any{map[string]any{"name": "recency_mask", "values": []any{"true", "false"}, "isTunable": true}},
				},
				"defaultResourceConfig": map[string]any{"max_number_of_training_jobs": "20", "max_parallel_number_of_training_jobs": "5"},
				"trainingInputMode":     "File",
				"creationDateTime":      pszCatalogTime, "lastUpdatedDateTime": pszCatalogTime,
			}}, nil
		}
	}
	return nil, pszMissing(arn)
}

func pszDescribeFeatureTransformation(arn string) (map[string]any, *pszErr) {
	for _, p := range pszRecipes {
		if p.domain == "" && p.featureARN() == arn {
			return map[string]any{"featureTransformation": map[string]any{
				"name": p.name, "featureTransformationArn": arn, "status": pszStatusActive,
				"defaultParameters": map[string]any{"max_user_history_length_percentile": "0.99", "min_user_history_length_percentile": "0.00"},
				"creationDateTime":  pszCatalogTime, "lastUpdatedDateTime": pszCatalogTime,
			}}, nil
		}
	}
	return nil, pszMissing(arn)
}

// pszUnit maps parts to a deterministic value in [0, 1).
func pszUnit(parts ...string) float64 {
	// sha256, not FNV: FNV's top bits barely move when only the last byte differs.
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return float64(binary.BigEndian.Uint64(sum[:8])>>11) / float64(1<<53)
}

func pszRound(v float64) float64 { return math.Round(v*1e6) / 1e6 }

// pszModelMetrics are deterministic offline metrics for a trained model.
func pszModelMetrics(arn string) map[string]any {
	out := map[string]any{}
	for _, name := range []string{"coverage", "mean_reciprocal_rank_at_25", "normalized_discounted_cumulative_gain_at_5",
		"normalized_discounted_cumulative_gain_at_10", "normalized_discounted_cumulative_gain_at_25",
		"precision_at_5", "precision_at_10", "precision_at_25"} {
		out[name] = pszRound(0.05 + 0.4*pszUnit(arn, name))
	}
	return out
}
