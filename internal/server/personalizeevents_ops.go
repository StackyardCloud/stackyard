package server

type personalizeEventsOperation struct{ Name string }

// Amazon Personalize Events operations sourced from:
// https://docs.aws.amazon.com/personalize/latest/dg/API_Operations_Amazon_Personalize_Events.html
var personalizeEventsOperations = []personalizeEventsOperation{
	{Name: "PutActionInteractions"},
	{Name: "PutActions"},
	{Name: "PutEvents"},
	{Name: "PutItems"},
	{Name: "PutUsers"},
}

var personalizeEventsOperationByName = func() map[string]personalizeEventsOperation {
	out := make(map[string]personalizeEventsOperation, len(personalizeEventsOperations))
	for _, op := range personalizeEventsOperations {
		out[op.Name] = op
	}
	return out
}()
