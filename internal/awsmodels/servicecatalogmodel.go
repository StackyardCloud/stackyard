package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../servicecatalog/2015-12-10/service-2.json internal/awsmodels/servicecatalog/model.json

//go:embed servicecatalog/model.json
var serviceCatalogModelJSON []byte

var serviceCatalogModel = sync.OnceValue(func() *Model { return mustLoadModel("servicecatalog", serviceCatalogModelJSON) })

// ServiceCatalogModel returns the validation model for AWS Service Catalog.
func ServiceCatalogModel() *Model { return serviceCatalogModel() }
