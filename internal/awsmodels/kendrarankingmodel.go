package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../kendra-ranking/2022-10-19/service-2.json internal/awsmodels/kendraranking/model.json

//go:embed kendraranking/model.json
var kendraRankingModelJSON []byte

var kendraRankingModel = sync.OnceValue(func() *Model { return mustLoadModel("kendraranking", kendraRankingModelJSON) })

// KendraRankingModel returns the validation model for Amazon Kendra Intelligent Ranking.
func KendraRankingModel() *Model { return kendraRankingModel() }
