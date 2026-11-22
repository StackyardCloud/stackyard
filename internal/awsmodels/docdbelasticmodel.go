package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed docdbelastic/model.json
var docdbElasticModelJSON []byte

var docdbElasticModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(docdbElasticModelJSON)
	if err != nil {
		panic("awsmodels: embedded docdb-elastic model is invalid: " + err.Error())
	}
	return m
})

// DocDBElasticModel returns the validation model for Amazon DocumentDB
// Elastic Clusters (from the AWS CLI 2.41 model via scripts/gen-aws-model-slim.py).
func DocDBElasticModel() *Model { return docdbElasticModel() }
