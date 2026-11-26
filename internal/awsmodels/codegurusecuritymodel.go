package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../codeguru-security/2018-05-10/service-2.json internal/awsmodels/codegurusecurity/model.json

//go:embed codegurusecurity/model.json
var codeGuruSecurityModelJSON []byte

var codeGuruSecurityModel = sync.OnceValue(func() *Model { return mustLoadModel("codegurusecurity", codeGuruSecurityModelJSON) })

// CodeGuruSecurityModel returns the validation model for Amazon CodeGuru Security.
func CodeGuruSecurityModel() *Model { return codeGuruSecurityModel() }
