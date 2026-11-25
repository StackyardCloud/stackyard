package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../connectparticipant/2018-09-07/service-2.json internal/awsmodels/connectparticipant/model.json

//go:embed connectparticipant/model.json
var connectParticipantModelJSON []byte

var connectParticipantModel = sync.OnceValue(func() *Model { return mustLoadModel("connectparticipant", connectParticipantModelJSON) })

// ConnectParticipantModel returns the validation model for Amazon Connect Participant Service.
func ConnectParticipantModel() *Model { return connectParticipantModel() }
