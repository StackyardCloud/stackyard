package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../mailmanager/2023-10-17/service-2.json internal/awsmodels/mailmanager/model.json

//go:embed mailmanager/model.json
var mailmanagerModelJSON []byte

var mailmanagerModel = sync.OnceValue(func() *Model { return mustLoadModel("mailmanager", mailmanagerModelJSON) })

// MailManagerModel returns the validation model for Amazon SES Mail Manager.
func MailManagerModel() *Model { return mailmanagerModel() }
