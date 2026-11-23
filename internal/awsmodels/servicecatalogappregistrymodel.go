package awsmodels

import (
	_ "embed"
	"sync"
)

//go:embed servicecatalogappregistry/model.json
var servicecatalogappregistryModelJSON []byte

var servicecatalogappregistryModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(servicecatalogappregistryModelJSON)
	if err != nil {
		panic("awsmodels: embedded servicecatalogappregistry model is invalid: " + err.Error())
	}
	return m
})

// ServiceCatalogAppRegistryModel returns the validation model for AWS Service
// Catalog AppRegistry.
func ServiceCatalogAppRegistryModel() *Model { return servicecatalogappregistryModel() }
