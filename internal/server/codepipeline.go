package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type codePipelineError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCodePipelineJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodePipelineJSONCandidate(r) {
		return false
	}

	action := parseCodePipelineTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCodePipelineError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := codePipelineOperationByName[action]; !known {
		respondCodePipelineError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "codepipeline")
	if !ok {
		respondCodePipelineError(w, status, code, msg)
		return true
	}

	payload, err := parseCodePipelinePayload(r)
	if err != nil {
		respondCodePipelineError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	var response map[string]any
	switch action {
	case "CreatePipeline":
		response = s.codepipeline.CreatePipeline(payload)
	case "GetPipeline":
		response = s.codepipeline.GetPipeline(payload)
	case "ListPipelines":
		response = s.codepipeline.ListPipelines()
	case "UpdatePipeline":
		response = s.codepipeline.UpdatePipeline(payload)
	case "DeletePipeline":
		response = s.codepipeline.DeletePipeline(payload)

	case "StartPipelineExecution":
		response = s.codepipeline.StartPipelineExecution(payload)
	case "GetPipelineExecution":
		response = s.codepipeline.GetPipelineExecution(payload)
	case "ListPipelineExecutions":
		response = s.codepipeline.ListPipelineExecutions(payload)
	case "StopPipelineExecution":
		response = s.codepipeline.StopPipelineExecution(payload)
	case "RetryStageExecution":
		response = s.codepipeline.RetryStageExecution(payload)
	case "RollbackStage":
		response = s.codepipeline.RollbackStage(payload)
	case "OverrideStageCondition":
		response = s.codepipeline.OverrideStageCondition()

	case "GetPipelineState":
		response = s.codepipeline.GetPipelineState(payload)
	case "EnableStageTransition":
		response = s.codepipeline.EnableStageTransition()
	case "DisableStageTransition":
		response = s.codepipeline.DisableStageTransition()
	case "ListActionExecutions":
		response = s.codepipeline.ListActionExecutions(payload)
	case "ListDeployActionExecutionTargets":
		response = s.codepipeline.ListDeployActionExecutionTargets(payload)

	case "CreateCustomActionType":
		response = s.codepipeline.CreateCustomActionType(payload)
	case "DeleteCustomActionType":
		response = s.codepipeline.DeleteCustomActionType(payload)
	case "UpdateActionType":
		response = s.codepipeline.UpdateActionType(payload)
	case "GetActionType":
		response = s.codepipeline.GetActionType(payload)
	case "ListActionTypes":
		response = s.codepipeline.ListActionTypes()
	case "ListRuleTypes":
		response = s.codepipeline.ListRuleTypes()
	case "ListRuleExecutions":
		response = s.codepipeline.ListRuleExecutions(payload)

	case "PutWebhook":
		response = s.codepipeline.PutWebhook(payload)
	case "ListWebhooks":
		response = s.codepipeline.ListWebhooks()
	case "DeleteWebhook":
		response = s.codepipeline.DeleteWebhook(payload)
	case "RegisterWebhookWithThirdParty":
		response = s.codepipeline.RegisterWebhookWithThirdParty(payload)
	case "DeregisterWebhookWithThirdParty":
		response = s.codepipeline.DeregisterWebhookWithThirdParty(payload)

	case "TagResource":
		response = s.codepipeline.TagResource(payload)
	case "UntagResource":
		response = s.codepipeline.UntagResource(payload)
	case "ListTagsForResource":
		response = s.codepipeline.ListTagsForResource(payload)

	case "PollForJobs":
		response = s.codepipeline.PollForJobs()
	case "AcknowledgeJob":
		response = s.codepipeline.AcknowledgeJob(payload)
	case "GetJobDetails":
		response = s.codepipeline.GetJobDetails(payload)
	case "PutJobSuccessResult":
		response = s.codepipeline.PutJobSuccessResult(payload)
	case "PutJobFailureResult":
		response = s.codepipeline.PutJobFailureResult(payload)
	case "PutActionRevision":
		response = s.codepipeline.PutActionRevision(payload)
	case "PutApprovalResult":
		response = s.codepipeline.PutApprovalResult()

	case "PollForThirdPartyJobs":
		response = s.codepipeline.PollForThirdPartyJobs()
	case "AcknowledgeThirdPartyJob":
		response = s.codepipeline.AcknowledgeThirdPartyJob(payload)
	case "GetThirdPartyJobDetails":
		response = s.codepipeline.GetThirdPartyJobDetails(payload)
	case "PutThirdPartyJobSuccessResult":
		response = s.codepipeline.PutThirdPartyJobSuccessResult(payload)
	case "PutThirdPartyJobFailureResult":
		response = s.codepipeline.PutThirdPartyJobFailureResult(payload)
	default:
		respondCodePipelineError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}

	if response == nil {
		response = map[string]any{}
	}
	respondCodePipelineJSON(w, http.StatusOK, response)
	return true
}

func isCodePipelineJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "CodePipeline_20150709") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "CodePipeline_20150709")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "codepipeline" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".codepipeline.") || strings.HasPrefix(host, "codepipeline.")
}

func parseCodePipelineTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "CodePipeline_20150709.") {
		return strings.TrimPrefix(target, "CodePipeline_20150709.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCodePipelinePayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func respondCodePipelineJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCodePipelineError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCodePipelineJSON(w, status, codePipelineError{Type: code, Message: msg})
}
