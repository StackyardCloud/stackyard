package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../backup-gateway/2021-01-01/service-2.json internal/awsmodels/backupgateway/model.json

//go:embed backupgateway/model.json
var backupgatewayModelJSON []byte

var backupgatewayModel = sync.OnceValue(func() *Model { return mustLoadModel("backupgateway", backupgatewayModelJSON) })

// BackupGatewayModel returns the validation model for AWS Backup gateway.
func BackupGatewayModel() *Model { return backupgatewayModel() }
