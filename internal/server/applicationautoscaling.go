package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const (
	aasTargetPrefix = "AnyScaleFrontendService."
	aasMaxBody      = 1 << 20
)

// handleApplicationAutoScalingJSONRouter serves Application Auto Scaling
// (awsJson 1.1, X-Amz-Target AnyScaleFrontendService.*, signed
// application-autoscaling).
func (s *Server) handleApplicationAutoScalingJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost || !strings.HasPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), aasTargetPrefix) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > aasMaxBody {
		respondAASError(w, &aasErr{"ValidationException", "request body must declare a Content-Length of at most 1 MiB"})
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "application-autoscaling"); !ok {
		respondAASStatus(w, status, code, msg)
		return true
	}
	action := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), aasTargetPrefix)
	if _, ok := applicationautoscalingOperationByName[action]; !ok {
		respondAASStatus(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+action)
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondAASError(w, aasValidation("unable to read request body"))
		return true
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}
	out, aerr := s.applicationAutoScalingDispatch(action, body)
	if aerr != nil {
		respondAASError(w, aerr)
		return true
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
	return true
}

func (s *Server) applicationAutoScalingDispatch(action string, body []byte) ([]byte, *aasErr) {
	st := s.appAutoScaling
	decode := func(v any) *aasErr {
		if err := json.Unmarshal(body, v); err != nil {
			return &aasErr{"SerializationException", "invalid request body: " + err.Error()}
		}
		return nil
	}
	switch action {
	case "RegisterScalableTarget":
		var in aasRegisterInput
		if err := decode(&in); err != nil {
			return nil, err
		}
		// Resource existence is checked before taking the store lock: the
		// other services have locks of their own.
		if err := aasCheckKey(in.aasTargetKey); err != nil {
			return nil, err
		}
		if err := s.aasCheckResourceExists(in.aasTargetKey); err != nil {
			return nil, err
		}
		return st.run(func() (any, *aasErr) { return st.register(in) })
	case "DeregisterScalableTarget":
		var in aasTargetKey
		if err := decode(&in); err != nil {
			return nil, err
		}
		return st.run(func() (any, *aasErr) { return st.deregister(in) })
	case "DescribeScalableTargets", "DescribeScalingActivities", "DescribeScalingPolicies", "DescribeScheduledActions":
		var in aasDescribeInput
		if err := decode(&in); err != nil {
			return nil, err
		}
		describe := map[string]func(aasDescribeInput) (any, *aasErr){
			"DescribeScalableTargets":   st.describeTargets,
			"DescribeScalingActivities": st.describeActivities,
			"DescribeScalingPolicies":   st.describePolicies,
			"DescribeScheduledActions":  st.describeScheduled,
		}[action]
		return st.run(func() (any, *aasErr) { return describe(in) })
	case "PutScalingPolicy":
		var in aasPutPolicyInput
		if err := decode(&in); err != nil {
			return nil, err
		}
		return st.run(func() (any, *aasErr) { return st.putPolicy(in) })
	case "DeleteScalingPolicy":
		var in aasNamedInput
		if err := decode(&in); err != nil {
			return nil, err
		}
		return st.run(func() (any, *aasErr) { return st.deletePolicy(in) })
	case "PutScheduledAction":
		var in aasPutScheduledActionInput
		if err := decode(&in); err != nil {
			return nil, err
		}
		return st.run(func() (any, *aasErr) { return st.putScheduled(in) })
	case "DeleteScheduledAction":
		var in aasNamedInput
		if err := decode(&in); err != nil {
			return nil, err
		}
		return st.run(func() (any, *aasErr) { return st.deleteScheduled(in) })
	case "GetPredictiveScalingForecast":
		var in aasForecastInput
		if err := decode(&in); err != nil {
			return nil, err
		}
		return st.run(func() (any, *aasErr) { return st.forecast(in) })
	}
	var in aasTagInput
	if err := decode(&in); err != nil {
		return nil, err
	}
	tagOps := map[string]func(aasTagInput) (any, *aasErr){
		"TagResource": st.tag, "UntagResource": st.untag, "ListTagsForResource": st.listTags,
	}
	return st.run(func() (any, *aasErr) { return tagOps[action](in) })
}

// aasCheckResourceExists checks the resource behind a target in the stores
// Stackyard keeps for it (ECS services, DynamoDB tables, Lambda aliases and
// versions). Other namespaces are accepted by ResourceId format alone.
func (s *Server) aasCheckResourceExists(k aasTargetKey) *aasErr {
	parts := strings.Split(k.ResourceId, "/")
	switch k.ServiceNamespace {
	case "ecs":
		svcs, failures, err := s.ecs.DescribeServices(parts[1], []string{parts[2]})
		if err != nil || len(failures) > 0 || len(svcs) == 0 {
			return aasValidation("ECS service doesn't exist: %s", k.ResourceId)
		}
	case "dynamodb":
		if _, err := s.dynamodb.DescribeTable(parts[1]); err != nil {
			return aasValidation("Requested resource not found: Table: %s not found", parts[1])
		}
	case "lambda":
		fields := strings.Split(k.ResourceId, ":")
		if fields[2] == "$LATEST" {
			return aasValidation("Provisioned concurrency can't be configured on $LATEST; use an alias or version")
		}
		if _, _, err := s.lambda.GetFunction(fields[1], fields[2]); err != nil {
			return aasValidation("Function or qualifier doesn't exist: %s", k.ResourceId)
		}
	}
	return nil
}

func (s *Server) aasPutAlarm(name string) {
	s.cloudwatch.Handle("PutMetricAlarm", url.Values{"AlarmName": {name}})
}

func (s *Server) aasDeleteAlarm(name string) {
	s.cloudwatch.Handle("DeleteAlarms", url.Values{"AlarmNames.member": {name}})
}

func respondAASError(w http.ResponseWriter, err *aasErr) {
	respondAASStatus(w, http.StatusBadRequest, err.code, err.msg)
}

func respondAASStatus(w http.ResponseWriter, status int, code, msg string) {
	body, _ := json.Marshal(map[string]string{"__type": code, "message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
