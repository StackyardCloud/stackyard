package server

import (
	"strings"
)

// pipesMissingResource returns why a pipe can't run, naming the first of its
// source, enrichment and target that Stackyard knows doesn't exist ("" when
// all exist or can't be checked). AWS accepts such a pipe and then fails it,
// so this becomes the CREATE_FAILED/UPDATE_FAILED StateReason, not an error.
func (s *Server) pipesMissingResource(source, enrichment, target string) string {
	for _, ref := range []struct{ role, arn string }{{"source", source}, {"enrichment", enrichment}, {"target", target}} {
		if ref.arn != "" && !s.pipesResourceExists(ref.arn) {
			return "The " + ref.role + " " + ref.arn + " does not exist or the role can't access it."
		}
	}
	return ""
}

// pipesResourceExists checks the stores that track real existence. Resource
// types Stackyard doesn't model strictly (MQ, MSK, ECS, Batch, ...) pass.
func (s *Server) pipesResourceExists(arn string) bool {
	service, resource := pipesARNService(arn)
	switch service {
	case "sqs":
		_, err := s.sqs.GetQueue(resource)
		return err == nil
	case "sns":
		_, err := s.sns.GetTopic(arn)
		return err == nil
	case "lambda":
		name := strings.TrimPrefix(resource, "function:")
		if i := strings.IndexByte(name, ':'); i >= 0 {
			name = name[:i]
		}
		_, _, err := s.lambda.GetFunction(name, "")
		return err == nil
	case "dynamodb":
		if !strings.Contains(resource, "/stream/") {
			return true
		}
		_, err := s.dynamodb.DescribeStream(arn, 1, "")
		return err == nil
	case "kinesis":
		name, ok := strings.CutPrefix(resource, "stream/")
		if !ok {
			return true
		}
		s.kinesis.mu.Lock()
		defer s.kinesis.mu.Unlock()
		for _, stream := range s.kinesis.streams {
			if stream.ARN == arn || stream.Name == name {
				return true
			}
		}
		return false
	case "states":
		s.stepfunctions.mu.Lock()
		defer s.stepfunctions.mu.Unlock()
		_, ok := s.stepfunctions.stateMachines[arn]
		return ok
	}
	return true
}
