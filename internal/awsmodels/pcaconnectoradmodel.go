package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.41 model, written by scripts/gen-aws-model-slim.py:
//
//	gen-aws-model-slim.py .../pca-connector-ad/2018-05-10/service-2.json internal/awsmodels/pcaconnectorad/model.json
//
//go:embed pcaconnectorad/model.json
var pcaConnectorADModelJSON []byte

var pcaConnectorADModel = sync.OnceValue(func() *Model {
	m, err := LoadModel(pcaConnectorADModelJSON)
	if err != nil {
		panic("awsmodels: embedded pca-connector-ad model is invalid: " + err.Error())
	}
	return m
})

// PCAConnectorADModel returns the validation model for Private CA Connector for Active Directory.
func PCAConnectorADModel() *Model { return pcaConnectorADModel() }
