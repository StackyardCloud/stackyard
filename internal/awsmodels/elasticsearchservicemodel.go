package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed elasticsearchservice/model.json
var elasticsearchServiceModelJSON []byte

var elasticsearchServiceModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(elasticsearchServiceModelJSON)
	if err != nil {
		panic("awsmodels: embedded es model is invalid: " + err.Error())
	}
	return m
})

// ElasticsearchServiceModel returns the validation model for the legacy
// Amazon Elasticsearch Service API (es 2015-01-01), from the AWS CLI 2.41 model.
func ElasticsearchServiceModel() *Model { return elasticsearchServiceModel() }
