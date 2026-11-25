package awsmodels

import (
	_ "embed"
	"sync"
)

// Slim copy of the AWS CLI 2.42.1 model, written by scripts/gen-aws-model-slim.py:
//   gen-aws-model-slim.py .../personalize-events/2018-03-22/service-2.json internal/awsmodels/personalizeevents/model.json

//go:embed personalizeevents/model.json
var personalizeeventsModelJSON []byte

var personalizeeventsModel = sync.OnceValue(func() *Model { return mustLoadModel("personalize-events", personalizeeventsModelJSON) })

// PersonalizeEventsModel returns the validation model for Amazon PersonalizeEvents.
func PersonalizeEventsModel() *Model { return personalizeeventsModel() }
