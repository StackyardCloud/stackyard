package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	lambdasvc "github.com/stackyard/stackyard/internal/services/lambda"
)

type lambdaError struct {
	Type    string `json:"Type,omitempty"`
	Message string `json:"message"`
}

type lambdaRoute struct {
	Method    string
	Pattern   string
	Operation string
}

var lambdaRoutes = []lambdaRoute{
	{Method: http.MethodPost, Pattern: "/2018-10-31/layers/{LayerName}/versions/{VersionNumber}/policy", Operation: "AddLayerVersionPermission"},
	{Method: http.MethodPost, Pattern: "/2015-03-31/functions/{FunctionName}/policy", Operation: "AddPermission"},
	{Method: http.MethodPost, Pattern: "/2025-12-01/durable-executions/{DurableExecutionArn}/checkpoint", Operation: "CheckpointDurableExecution"},
	{Method: http.MethodPost, Pattern: "/2015-03-31/functions/{FunctionName}/aliases", Operation: "CreateAlias"},
	{Method: http.MethodPost, Pattern: "/2025-11-30/capacity-providers", Operation: "CreateCapacityProvider"},
	{Method: http.MethodPost, Pattern: "/2020-04-22/code-signing-configs", Operation: "CreateCodeSigningConfig"},
	{Method: http.MethodPost, Pattern: "/2015-03-31/event-source-mappings", Operation: "CreateEventSourceMapping"},
	{Method: http.MethodPost, Pattern: "/2015-03-31/functions", Operation: "CreateFunction"},
	{Method: http.MethodPost, Pattern: "/2021-10-31/functions/{FunctionName}/url", Operation: "CreateFunctionUrlConfig"},
	{Method: http.MethodDelete, Pattern: "/2015-03-31/functions/{FunctionName}/aliases/{Name}", Operation: "DeleteAlias"},
	{Method: http.MethodDelete, Pattern: "/2025-11-30/capacity-providers/{CapacityProviderName}", Operation: "DeleteCapacityProvider"},
	{Method: http.MethodDelete, Pattern: "/2020-04-22/code-signing-configs/{CodeSigningConfigArn}", Operation: "DeleteCodeSigningConfig"},
	{Method: http.MethodDelete, Pattern: "/2015-03-31/event-source-mappings/{UUID}", Operation: "DeleteEventSourceMapping"},
	{Method: http.MethodDelete, Pattern: "/2015-03-31/functions/{FunctionName}", Operation: "DeleteFunction"},
	{Method: http.MethodDelete, Pattern: "/2020-06-30/functions/{FunctionName}/code-signing-config", Operation: "DeleteFunctionCodeSigningConfig"},
	{Method: http.MethodDelete, Pattern: "/2017-10-31/functions/{FunctionName}/concurrency", Operation: "DeleteFunctionConcurrency"},
	{Method: http.MethodDelete, Pattern: "/2019-09-25/functions/{FunctionName}/event-invoke-config", Operation: "DeleteFunctionEventInvokeConfig"},
	{Method: http.MethodDelete, Pattern: "/2021-10-31/functions/{FunctionName}/url", Operation: "DeleteFunctionUrlConfig"},
	{Method: http.MethodDelete, Pattern: "/2018-10-31/layers/{LayerName}/versions/{VersionNumber}", Operation: "DeleteLayerVersion"},
	{Method: http.MethodDelete, Pattern: "/2019-09-30/functions/{FunctionName}/provisioned-concurrency", Operation: "DeleteProvisionedConcurrencyConfig"},
	{Method: http.MethodGet, Pattern: "/2016-08-19/account-settings", Operation: "GetAccountSettings"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/functions/{FunctionName}/aliases/{Name}", Operation: "GetAlias"},
	{Method: http.MethodGet, Pattern: "/2025-11-30/capacity-providers/{CapacityProviderName}", Operation: "GetCapacityProvider"},
	{Method: http.MethodGet, Pattern: "/2020-04-22/code-signing-configs/{CodeSigningConfigArn}", Operation: "GetCodeSigningConfig"},
	{Method: http.MethodGet, Pattern: "/2025-12-01/durable-executions/{DurableExecutionArn}", Operation: "GetDurableExecution"},
	{Method: http.MethodGet, Pattern: "/2025-12-01/durable-executions/{DurableExecutionArn}/history", Operation: "GetDurableExecutionHistory"},
	{Method: http.MethodGet, Pattern: "/2025-12-01/durable-executions/{DurableExecutionArn}/state", Operation: "GetDurableExecutionState"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/event-source-mappings/{UUID}", Operation: "GetEventSourceMapping"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/functions/{FunctionName}", Operation: "GetFunction"},
	{Method: http.MethodGet, Pattern: "/2020-06-30/functions/{FunctionName}/code-signing-config", Operation: "GetFunctionCodeSigningConfig"},
	{Method: http.MethodGet, Pattern: "/2019-09-30/functions/{FunctionName}/concurrency", Operation: "GetFunctionConcurrency"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/functions/{FunctionName}/configuration", Operation: "GetFunctionConfiguration"},
	{Method: http.MethodGet, Pattern: "/2019-09-25/functions/{FunctionName}/event-invoke-config", Operation: "GetFunctionEventInvokeConfig"},
	{Method: http.MethodGet, Pattern: "/2024-08-31/functions/{FunctionName}/recursion-config", Operation: "GetFunctionRecursionConfig"},
	{Method: http.MethodGet, Pattern: "/2025-11-30/functions/{FunctionName}/function-scaling-config", Operation: "GetFunctionScalingConfig"},
	{Method: http.MethodGet, Pattern: "/2021-10-31/functions/{FunctionName}/url", Operation: "GetFunctionUrlConfig"},
	{Method: http.MethodGet, Pattern: "/2018-10-31/layers/{LayerName}/versions/{VersionNumber}", Operation: "GetLayerVersion"},
	{Method: http.MethodGet, Pattern: "/2018-10-31/layers", Operation: "GetLayerVersionByArn"},
	{Method: http.MethodGet, Pattern: "/2018-10-31/layers/{LayerName}/versions/{VersionNumber}/policy", Operation: "GetLayerVersionPolicy"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/functions/{FunctionName}/policy", Operation: "GetPolicy"},
	{Method: http.MethodGet, Pattern: "/2019-09-30/functions/{FunctionName}/provisioned-concurrency", Operation: "GetProvisionedConcurrencyConfig"},
	{Method: http.MethodGet, Pattern: "/2021-07-20/functions/{FunctionName}/runtime-management-config", Operation: "GetRuntimeManagementConfig"},
	{Method: http.MethodPost, Pattern: "/2015-03-31/functions/{FunctionName}/invocations", Operation: "Invoke"},
	{Method: http.MethodPost, Pattern: "/2014-11-13/functions/{FunctionName}/invoke-async", Operation: "InvokeAsync"},
	{Method: http.MethodPost, Pattern: "/2021-11-15/functions/{FunctionName}/response-streaming-invocations", Operation: "InvokeWithResponseStream"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/functions/{FunctionName}/aliases", Operation: "ListAliases"},
	{Method: http.MethodGet, Pattern: "/2025-11-30/capacity-providers", Operation: "ListCapacityProviders"},
	{Method: http.MethodGet, Pattern: "/2020-04-22/code-signing-configs", Operation: "ListCodeSigningConfigs"},
	{Method: http.MethodGet, Pattern: "/2025-12-01/functions/{FunctionName}/durable-executions", Operation: "ListDurableExecutionsByFunction"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/event-source-mappings", Operation: "ListEventSourceMappings"},
	{Method: http.MethodGet, Pattern: "/2019-09-25/functions/{FunctionName}/event-invoke-config/list", Operation: "ListFunctionEventInvokeConfigs"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/functions", Operation: "ListFunctions"},
	{Method: http.MethodGet, Pattern: "/2020-04-22/code-signing-configs/{CodeSigningConfigArn}/functions", Operation: "ListFunctionsByCodeSigningConfig"},
	{Method: http.MethodGet, Pattern: "/2021-10-31/functions/{FunctionName}/urls", Operation: "ListFunctionUrlConfigs"},
	{Method: http.MethodGet, Pattern: "/2025-11-30/capacity-providers/{CapacityProviderName}/function-versions", Operation: "ListFunctionVersionsByCapacityProvider"},
	{Method: http.MethodGet, Pattern: "/2018-10-31/layers", Operation: "ListLayers"},
	{Method: http.MethodGet, Pattern: "/2018-10-31/layers/{LayerName}/versions", Operation: "ListLayerVersions"},
	{Method: http.MethodGet, Pattern: "/2019-09-30/functions/{FunctionName}/provisioned-concurrency", Operation: "ListProvisionedConcurrencyConfigs"},
	{Method: http.MethodGet, Pattern: "/2017-03-31/tags/{Resource+}", Operation: "ListTags"},
	{Method: http.MethodGet, Pattern: "/2015-03-31/functions/{FunctionName}/versions", Operation: "ListVersionsByFunction"},
	{Method: http.MethodPost, Pattern: "/2018-10-31/layers/{LayerName}/versions", Operation: "PublishLayerVersion"},
	{Method: http.MethodPost, Pattern: "/2015-03-31/functions/{FunctionName}/versions", Operation: "PublishVersion"},
	{Method: http.MethodPut, Pattern: "/2020-06-30/functions/{FunctionName}/code-signing-config", Operation: "PutFunctionCodeSigningConfig"},
	{Method: http.MethodPut, Pattern: "/2017-10-31/functions/{FunctionName}/concurrency", Operation: "PutFunctionConcurrency"},
	{Method: http.MethodPut, Pattern: "/2019-09-25/functions/{FunctionName}/event-invoke-config", Operation: "PutFunctionEventInvokeConfig"},
	{Method: http.MethodPut, Pattern: "/2024-08-31/functions/{FunctionName}/recursion-config", Operation: "PutFunctionRecursionConfig"},
	{Method: http.MethodPut, Pattern: "/2025-11-30/functions/{FunctionName}/function-scaling-config", Operation: "PutFunctionScalingConfig"},
	{Method: http.MethodPut, Pattern: "/2019-09-30/functions/{FunctionName}/provisioned-concurrency", Operation: "PutProvisionedConcurrencyConfig"},
	{Method: http.MethodPut, Pattern: "/2021-07-20/functions/{FunctionName}/runtime-management-config", Operation: "PutRuntimeManagementConfig"},
	{Method: http.MethodDelete, Pattern: "/2018-10-31/layers/{LayerName}/versions/{VersionNumber}/policy/{StatementId}", Operation: "RemoveLayerVersionPermission"},
	{Method: http.MethodDelete, Pattern: "/2015-03-31/functions/{FunctionName}/policy/{StatementId}", Operation: "RemovePermission"},
	{Method: http.MethodPost, Pattern: "/2025-12-01/durable-execution-callbacks/{CallbackId}/fail", Operation: "SendDurableExecutionCallbackFailure"},
	{Method: http.MethodPost, Pattern: "/2025-12-01/durable-execution-callbacks/{CallbackId}/heartbeat", Operation: "SendDurableExecutionCallbackHeartbeat"},
	{Method: http.MethodPost, Pattern: "/2025-12-01/durable-execution-callbacks/{CallbackId}/succeed", Operation: "SendDurableExecutionCallbackSuccess"},
	{Method: http.MethodPost, Pattern: "/2025-12-01/durable-executions/{DurableExecutionArn}/stop", Operation: "StopDurableExecution"},
	{Method: http.MethodPost, Pattern: "/2017-03-31/tags/{Resource+}", Operation: "TagResource"},
	{Method: http.MethodDelete, Pattern: "/2017-03-31/tags/{Resource+}", Operation: "UntagResource"},
	{Method: http.MethodPut, Pattern: "/2015-03-31/functions/{FunctionName}/aliases/{Name}", Operation: "UpdateAlias"},
	{Method: http.MethodPut, Pattern: "/2025-11-30/capacity-providers/{CapacityProviderName}", Operation: "UpdateCapacityProvider"},
	{Method: http.MethodPut, Pattern: "/2020-04-22/code-signing-configs/{CodeSigningConfigArn}", Operation: "UpdateCodeSigningConfig"},
	{Method: http.MethodPut, Pattern: "/2015-03-31/event-source-mappings/{UUID}", Operation: "UpdateEventSourceMapping"},
	{Method: http.MethodPut, Pattern: "/2015-03-31/functions/{FunctionName}/code", Operation: "UpdateFunctionCode"},
	{Method: http.MethodPut, Pattern: "/2015-03-31/functions/{FunctionName}/configuration", Operation: "UpdateFunctionConfiguration"},
	{Method: http.MethodPost, Pattern: "/2019-09-25/functions/{FunctionName}/event-invoke-config", Operation: "UpdateFunctionEventInvokeConfig"},
	{Method: http.MethodPut, Pattern: "/2021-10-31/functions/{FunctionName}/url", Operation: "UpdateFunctionUrlConfig"},
}

func (s *Server) handleLambdaRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isLambdaRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "lambda")
	if !ok {
		respondLambdaError(w, status, code, msg)
		return true
	}

	op, params, matched := matchLambdaRoute(r.Method, r.URL)
	if !matched {
		respondLambdaError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}
	if _, known := lambdaOperationByName[op.Operation]; !known {
		respondLambdaError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}

	qualifier := strings.TrimSpace(r.URL.Query().Get("Qualifier"))

	switch op.Operation {
	case "CreateFunction":
		var req lambdaCreateFunctionInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		codeBytes := append([]byte(nil), req.Code.ZipFile...)
		if len(codeBytes) == 0 && (strings.TrimSpace(req.Code.S3Bucket) != "" || strings.TrimSpace(req.Code.S3Key) != "") {
			codeBytes = []byte(req.Code.S3Bucket + "/" + req.Code.S3Key)
		}
		fn, err := s.lambda.CreateFunction(
			req.FunctionName,
			req.Role,
			req.Handler,
			req.Runtime,
			req.Description,
			req.Timeout,
			req.MemorySize,
			codeBytes,
			req.Tags,
			req.Architectures,
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		if req.Publish {
			published, err := s.lambda.PublishVersion(req.FunctionName, "")
			if err == nil {
				fn = published
			}
		}
		respondLambdaJSON(w, http.StatusCreated, lambdaFunctionConfigurationPayload(fn))
		return true
	case "DeleteFunction":
		if err := s.lambda.DeleteFunction(params["functionName"], qualifier); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListFunctions":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		functions, nextMarker := s.lambda.ListFunctions(maxItems, marker)
		out := make([]map[string]any, 0, len(functions))
		for _, fn := range functions {
			out = append(out, lambdaFunctionConfigurationPayload(fn))
		}
		resp := map[string]any{"Functions": out}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "GetFunction":
		fn, tags, err := s.lambda.GetFunction(params["functionName"], qualifier)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{
			"Configuration": lambdaFunctionConfigurationPayload(fn),
			"Code": map[string]any{
				"RepositoryType": "S3",
				"Location":       "https://stackyard.local/lambda/" + fn.Name,
			},
			"Tags": tags,
		})
		return true
	case "GetFunctionConfiguration":
		fn, err := s.lambda.GetFunctionConfiguration(params["functionName"], qualifier)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, lambdaFunctionConfigurationPayload(fn))
		return true
	case "UpdateFunctionConfiguration":
		var req lambdaUpdateFunctionConfigurationInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		fn, err := s.lambda.UpdateFunctionConfiguration(
			params["functionName"],
			qualifier,
			req.Runtime,
			req.Role,
			req.Handler,
			req.Description,
			req.Timeout,
			req.MemorySize,
			req.Architectures,
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, lambdaFunctionConfigurationPayload(fn))
		return true
	case "UpdateFunctionCode":
		var req lambdaUpdateFunctionCodeInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		codeBytes := append([]byte(nil), req.ZipFile...)
		if len(codeBytes) == 0 && (strings.TrimSpace(req.S3Bucket) != "" || strings.TrimSpace(req.S3Key) != "") {
			codeBytes = []byte(req.S3Bucket + "/" + req.S3Key)
		}
		fn, err := s.lambda.UpdateFunctionCode(params["functionName"], qualifier, codeBytes, req.Publish)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, lambdaFunctionConfigurationPayload(fn))
		return true
	case "PublishVersion":
		var req lambdaPublishVersionInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		fn, err := s.lambda.PublishVersion(params["functionName"], req.Description)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, lambdaFunctionConfigurationPayload(fn))
		return true
	case "ListVersionsByFunction":
		versions, err := s.lambda.ListVersionsByFunction(params["functionName"])
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(versions))
		for _, fn := range versions {
			out = append(out, lambdaFunctionConfigurationPayload(fn))
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"Versions": out})
		return true
	case "CreateAlias":
		var req lambdaCreateAliasInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		alias, err := s.lambda.CreateAlias(params["functionName"], req.Name, req.FunctionVersion, req.Description)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, lambdaAliasPayload(alias))
		return true
	case "GetAlias":
		alias, err := s.lambda.GetAlias(params["functionName"], params["name"])
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, lambdaAliasPayload(alias))
		return true
	case "ListAliases":
		aliases, err := s.lambda.ListAliases(params["functionName"])
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(aliases))
		for _, alias := range aliases {
			out = append(out, lambdaAliasPayload(alias))
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"Aliases": out})
		return true
	case "UpdateAlias":
		var req lambdaUpdateAliasInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		alias, err := s.lambda.UpdateAlias(params["functionName"], params["name"], req.FunctionVersion, req.Description)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, lambdaAliasPayload(alias))
		return true
	case "DeleteAlias":
		if err := s.lambda.DeleteAlias(params["functionName"], params["name"]); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "AddPermission":
		var req lambdaAddPermissionInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		statement, err := s.lambda.AddPermission(
			params["functionName"],
			qualifier,
			req.StatementID,
			req.Action,
			req.Principal,
			req.SourceARN,
			req.SourceAccount,
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, map[string]any{"Statement": statement})
		return true
	case "GetPolicy":
		policy, revisionID, err := s.lambda.GetPolicy(params["functionName"], qualifier)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{
			"Policy":     policy,
			"RevisionId": revisionID,
		})
		return true
	case "RemovePermission":
		if err := s.lambda.RemovePermission(params["functionName"], qualifier, params["statementId"]); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "Invoke":
		payload, err := readBodyBytes(r)
		if err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid request body")
			return true
		}
		result, err := s.lambda.Invoke(
			params["functionName"],
			qualifier,
			strings.TrimSpace(r.Header.Get("X-Amz-Invocation-Type")),
			payload,
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.Header().Set("Content-Type", "application/json")
		if result.ExecutedVersion != "" {
			w.Header().Set("X-Amz-Executed-Version", result.ExecutedVersion)
		}
		if strings.TrimSpace(result.FunctionError) != "" {
			w.Header().Set("X-Amz-Function-Error", result.FunctionError)
		}
		if strings.TrimSpace(result.LogResult) != "" {
			w.Header().Set("X-Amz-Log-Result", result.LogResult)
		}
		w.WriteHeader(result.StatusCode)
		if len(result.Payload) > 0 {
			_, _ = w.Write(result.Payload)
		}
		return true
	case "TagResource":
		var req lambdaTagResourceInput
		if err := decodeLambdaJSONBody(r, &req); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return true
		}
		resource := lambdaResourceARN(params, r.URL.Path)
		if err := s.lambda.TagResource(resource, req.Tags); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "UntagResource":
		resource := lambdaResourceARN(params, r.URL.Path)
		if err := s.lambda.UntagResource(resource, r.URL.Query()["tagKeys"]); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListTags":
		resource := lambdaResourceARN(params, r.URL.Path)
		tags, err := s.lambda.ListTags(resource)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"Tags": tags})
		return true
	default:
		if s.handleLambdaAdditionalOperation(w, r, op, params, qualifier) {
			return true
		}
		respondLambdaError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}
}

func (s *Server) handleLambdaAdditionalOperation(w http.ResponseWriter, r *http.Request, op lambdaRoute, params map[string]string, qualifier string) bool {
	body := map[string]any{}
	decodeBody := func() bool {
		if err := decodeLambdaJSONBody(r, &body); err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid JSON body")
			return false
		}
		if body == nil {
			body = map[string]any{}
		}
		return true
	}

	switch op.Operation {
	case "GetAccountSettings":
		respondLambdaJSON(w, http.StatusOK, s.lambda.GetAccountSettings())
		return true
	case "CreateEventSourceMapping":
		if !decodeBody() {
			return true
		}
		mapping, err := s.lambda.CreateEventSourceMapping(lambdaMapString(body, "FunctionName"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, mapping)
		return true
	case "GetEventSourceMapping":
		mapping, err := s.lambda.GetEventSourceMapping(lambdaParam(params, "UUID", "uuid"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, mapping)
		return true
	case "ListEventSourceMappings":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListEventSourceMappings(
			r.URL.Query().Get("FunctionName"),
			r.URL.Query().Get("EventSourceArn"),
			maxItems,
			marker,
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"EventSourceMappings": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "UpdateEventSourceMapping":
		if !decodeBody() {
			return true
		}
		mapping, err := s.lambda.UpdateEventSourceMapping(lambdaParam(params, "UUID", "uuid"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, mapping)
		return true
	case "DeleteEventSourceMapping":
		mapping, err := s.lambda.DeleteEventSourceMapping(lambdaParam(params, "UUID", "uuid"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, mapping)
		return true
	case "PutFunctionConcurrency":
		if !decodeBody() {
			return true
		}
		value, err := s.lambda.PutFunctionConcurrency(lambdaParam(params, "FunctionName", "functionName"), lambdaMapInt32(body, "ReservedConcurrentExecutions"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"ReservedConcurrentExecutions": value})
		return true
	case "GetFunctionConcurrency":
		value, err := s.lambda.GetFunctionConcurrency(lambdaParam(params, "FunctionName", "functionName"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"ReservedConcurrentExecutions": value})
		return true
	case "DeleteFunctionConcurrency":
		if err := s.lambda.DeleteFunctionConcurrency(lambdaParam(params, "FunctionName", "functionName")); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "PutFunctionEventInvokeConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutFunctionEventInvokeConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier, body, false)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "UpdateFunctionEventInvokeConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutFunctionEventInvokeConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier, body, true)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "GetFunctionEventInvokeConfig":
		cfg, err := s.lambda.GetFunctionEventInvokeConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "DeleteFunctionEventInvokeConfig":
		if err := s.lambda.DeleteFunctionEventInvokeConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListFunctionEventInvokeConfigs":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListFunctionEventInvokeConfigs(lambdaParam(params, "FunctionName", "functionName"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"FunctionEventInvokeConfigs": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "CreateFunctionUrlConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutFunctionURLConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier, body, true)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, cfg)
		return true
	case "UpdateFunctionUrlConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutFunctionURLConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier, body, false)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "GetFunctionUrlConfig":
		cfg, err := s.lambda.GetFunctionURLConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "DeleteFunctionUrlConfig":
		if err := s.lambda.DeleteFunctionURLConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListFunctionUrlConfigs":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListFunctionURLConfigs(lambdaParam(params, "FunctionName", "functionName"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"FunctionUrlConfigs": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "PutRuntimeManagementConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutRuntimeManagementConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier, body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "GetRuntimeManagementConfig":
		cfg, err := s.lambda.GetRuntimeManagementConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "PutFunctionRecursionConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutFunctionRecursionConfig(lambdaParam(params, "FunctionName", "functionName"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "GetFunctionRecursionConfig":
		cfg, err := s.lambda.GetFunctionRecursionConfig(lambdaParam(params, "FunctionName", "functionName"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "PutFunctionScalingConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutFunctionScalingConfig(lambdaParam(params, "FunctionName", "functionName"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "GetFunctionScalingConfig":
		cfg, err := s.lambda.GetFunctionScalingConfig(lambdaParam(params, "FunctionName", "functionName"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "PutProvisionedConcurrencyConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutProvisionedConcurrencyConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier, body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "GetProvisionedConcurrencyConfig":
		cfg, err := s.lambda.GetProvisionedConcurrencyConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "DeleteProvisionedConcurrencyConfig":
		if err := s.lambda.DeleteProvisionedConcurrencyConfig(lambdaParam(params, "FunctionName", "functionName"), qualifier); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListProvisionedConcurrencyConfigs":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListProvisionedConcurrencyConfigs(lambdaParam(params, "FunctionName", "functionName"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"ProvisionedConcurrencyConfigs": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "CreateCodeSigningConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.CreateCodeSigningConfig(body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, map[string]any{"CodeSigningConfig": cfg})
		return true
	case "GetCodeSigningConfig":
		cfg, err := s.lambda.GetCodeSigningConfig(lambdaParam(params, "CodeSigningConfigArn", "codeSigningConfigArn"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"CodeSigningConfig": cfg})
		return true
	case "UpdateCodeSigningConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.UpdateCodeSigningConfig(lambdaParam(params, "CodeSigningConfigArn", "codeSigningConfigArn"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"CodeSigningConfig": cfg})
		return true
	case "DeleteCodeSigningConfig":
		if err := s.lambda.DeleteCodeSigningConfig(lambdaParam(params, "CodeSigningConfigArn", "codeSigningConfigArn")); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListCodeSigningConfigs":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker := s.lambda.ListCodeSigningConfigs(maxItems, marker)
		resp := map[string]any{"CodeSigningConfigs": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "PutFunctionCodeSigningConfig":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.PutFunctionCodeSigningConfig(lambdaParam(params, "FunctionName", "functionName"), lambdaMapString(body, "CodeSigningConfigArn"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "GetFunctionCodeSigningConfig":
		cfg, err := s.lambda.GetFunctionCodeSigningConfig(lambdaParam(params, "FunctionName", "functionName"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, cfg)
		return true
	case "DeleteFunctionCodeSigningConfig":
		if err := s.lambda.DeleteFunctionCodeSigningConfig(lambdaParam(params, "FunctionName", "functionName")); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListFunctionsByCodeSigningConfig":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListFunctionsByCodeSigningConfig(lambdaParam(params, "CodeSigningConfigArn", "codeSigningConfigArn"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"FunctionArns": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "PublishLayerVersion":
		if !decodeBody() {
			return true
		}
		layer, err := s.lambda.PublishLayerVersion(lambdaParam(params, "LayerName", "layerName"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, layer)
		return true
	case "GetLayerVersion":
		layer, err := s.lambda.GetLayerVersion(lambdaParam(params, "LayerName", "layerName"), lambdaParam(params, "VersionNumber", "versionNumber"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, layer)
		return true
	case "GetLayerVersionByArn":
		layer, err := s.lambda.GetLayerVersionByARN(strings.TrimSpace(r.URL.Query().Get("Arn")))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, layer)
		return true
	case "DeleteLayerVersion":
		if err := s.lambda.DeleteLayerVersion(lambdaParam(params, "LayerName", "layerName"), lambdaParam(params, "VersionNumber", "versionNumber")); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListLayerVersions":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListLayerVersions(lambdaParam(params, "LayerName", "layerName"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"LayerVersions": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "ListLayers":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker := s.lambda.ListLayers(maxItems, marker)
		resp := map[string]any{"Layers": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "AddLayerVersionPermission":
		if !decodeBody() {
			return true
		}
		statement, rev, err := s.lambda.AddLayerVersionPermission(
			lambdaParam(params, "LayerName", "layerName"),
			lambdaParam(params, "VersionNumber", "versionNumber"),
			lambdaMapString(body, "StatementId"),
			lambdaMapString(body, "Action"),
			lambdaMapString(body, "Principal"),
			lambdaMapString(body, "OrganizationId"),
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, map[string]any{"Statement": statement, "RevisionId": rev})
		return true
	case "GetLayerVersionPolicy":
		policy, rev, err := s.lambda.GetLayerVersionPolicy(lambdaParam(params, "LayerName", "layerName"), lambdaParam(params, "VersionNumber", "versionNumber"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"Policy": policy, "RevisionId": rev})
		return true
	case "RemoveLayerVersionPermission":
		if err := s.lambda.RemoveLayerVersionPermission(
			lambdaParam(params, "LayerName", "layerName"),
			lambdaParam(params, "VersionNumber", "versionNumber"),
			lambdaParam(params, "StatementId", "statementId"),
		); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "CreateCapacityProvider":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.CreateCapacityProvider(body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusCreated, map[string]any{"CapacityProvider": cfg})
		return true
	case "GetCapacityProvider":
		cfg, err := s.lambda.GetCapacityProvider(lambdaParam(params, "CapacityProviderName", "capacityProviderName"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"CapacityProvider": cfg})
		return true
	case "UpdateCapacityProvider":
		if !decodeBody() {
			return true
		}
		cfg, err := s.lambda.UpdateCapacityProvider(lambdaParam(params, "CapacityProviderName", "capacityProviderName"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{"CapacityProvider": cfg})
		return true
	case "DeleteCapacityProvider":
		if err := s.lambda.DeleteCapacityProvider(lambdaParam(params, "CapacityProviderName", "capacityProviderName")); err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	case "ListCapacityProviders":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker := s.lambda.ListCapacityProviders(maxItems, marker)
		resp := map[string]any{"CapacityProviders": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "ListFunctionVersionsByCapacityProvider":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListFunctionVersionsByCapacityProvider(lambdaParam(params, "CapacityProviderName", "capacityProviderName"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"FunctionVersions": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "ListDurableExecutionsByFunction":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.ListDurableExecutionsByFunction(lambdaParam(params, "FunctionName", "functionName"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"DurableExecutions": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "GetDurableExecution":
		item, err := s.lambda.GetDurableExecution(lambdaParam(params, "DurableExecutionArn", "durableExecutionArn"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, item)
		return true
	case "GetDurableExecutionHistory":
		maxItems := parseLambdaIntQuery(r.URL.Query(), "MaxItems")
		marker := parseLambdaIntQuery(r.URL.Query(), "Marker")
		items, nextMarker, err := s.lambda.GetDurableExecutionHistory(lambdaParam(params, "DurableExecutionArn", "durableExecutionArn"), maxItems, marker)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		resp := map[string]any{"Events": items}
		if nextMarker != "" {
			resp["NextMarker"] = nextMarker
		}
		respondLambdaJSON(w, http.StatusOK, resp)
		return true
	case "GetDurableExecutionState":
		item, err := s.lambda.GetDurableExecutionState(lambdaParam(params, "DurableExecutionArn", "durableExecutionArn"))
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, item)
		return true
	case "CheckpointDurableExecution":
		if !decodeBody() {
			return true
		}
		out, err := s.lambda.CheckpointDurableExecution(lambdaParam(params, "DurableExecutionArn", "durableExecutionArn"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, out)
		return true
	case "StopDurableExecution":
		if !decodeBody() {
			return true
		}
		out, err := s.lambda.StopDurableExecution(lambdaParam(params, "DurableExecutionArn", "durableExecutionArn"), body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, out)
		return true
	case "SendDurableExecutionCallbackFailure", "SendDurableExecutionCallbackHeartbeat", "SendDurableExecutionCallbackSuccess":
		if !decodeBody() {
			return true
		}
		status := strings.TrimPrefix(op.Operation, "SendDurableExecutionCallback")
		out, err := s.lambda.DurableExecutionCallback(lambdaParam(params, "CallbackId", "callbackId"), status, body)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, out)
		return true
	case "InvokeAsync":
		payload, err := readBodyBytes(r)
		if err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid request body")
			return true
		}
		result, err := s.lambda.Invoke(
			lambdaParam(params, "FunctionName", "functionName"),
			qualifier,
			"Event",
			payload,
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		if result.StatusCode == 0 {
			result.StatusCode = http.StatusAccepted
		}
		w.WriteHeader(result.StatusCode)
		return true
	case "InvokeWithResponseStream":
		payload, err := readBodyBytes(r)
		if err != nil {
			respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", "invalid request body")
			return true
		}
		result, err := s.lambda.Invoke(
			lambdaParam(params, "FunctionName", "functionName"),
			qualifier,
			"RequestResponse",
			payload,
		)
		if err != nil {
			respondLambdaErrorForErr(w, err)
			return true
		}
		respondLambdaJSON(w, http.StatusOK, map[string]any{
			"EventStreamError": nil,
			"Payload":          string(result.Payload),
		})
		return true
	}

	return false
}

func respondLambdaJSON(w http.ResponseWriter, status int, body any) {
	respondJSON(w, status, body)
}

func respondLambdaError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondLambdaJSON(w, status, lambdaError{
		Type:    code,
		Message: msg,
	})
}

func respondLambdaErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, lambdasvc.ErrInvalidParameter):
		respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", err.Error())
	case errors.Is(err, lambdasvc.ErrAlreadyExists):
		respondLambdaError(w, http.StatusConflict, "ResourceConflictException", err.Error())
	case errors.Is(err, lambdasvc.ErrNotFound):
		respondLambdaError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
	default:
		respondLambdaError(w, http.StatusBadRequest, "InvalidParameterValueException", err.Error())
	}
}

func isLambdaRESTCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	path := normalizeLambdaPath(r.URL.Path)
	prefixes := []string{
		"/2014-11-13/",
		"/2015-03-31/",
		"/2016-08-19/",
		"/2017-03-31/",
		"/2017-10-31/",
		"/2018-10-31/",
		"/2019-09-25/",
		"/2019-09-30/",
		"/2020-04-22/",
		"/2020-06-30/",
		"/2021-07-20/",
		"/2021-10-31/",
		"/2021-11-15/",
		"/2024-08-31/",
		"/2025-11-30/",
		"/2025-12-01/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func matchLambdaRoute(method string, u *url.URL) (lambdaRoute, map[string]string, bool) {
	path := normalizeLambdaPath(u.Path)
	query := u.Query()
	for _, route := range lambdaRoutes {
		if route.Method != method {
			continue
		}
		switch route.Operation {
		case "GetLayerVersionByArn":
			if !strings.EqualFold(strings.TrimSpace(query.Get("find")), "LayerVersion") {
				continue
			}
		case "ListLayers":
			if strings.EqualFold(strings.TrimSpace(query.Get("find")), "LayerVersion") {
				continue
			}
		case "ListProvisionedConcurrencyConfigs":
			if !strings.EqualFold(strings.TrimSpace(query.Get("List")), "ALL") {
				continue
			}
		case "GetProvisionedConcurrencyConfig":
			if strings.EqualFold(strings.TrimSpace(query.Get("List")), "ALL") {
				continue
			}
		}
		if params, ok := matchLambdaPathPattern(route.Pattern, path); ok {
			return route, params, true
		}
	}
	return lambdaRoute{}, nil, false
}

func matchLambdaPathPattern(pattern, actual string) (map[string]string, bool) {
	patternSegs := splitLambdaPath(pattern)
	actualSegs := splitLambdaPath(actual)
	params := map[string]string{}
	pi := 0
	ai := 0
	for pi < len(patternSegs) {
		if ai >= len(actualSegs) {
			return nil, false
		}
		seg := patternSegs[pi]
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") && len(seg) > 2 {
			name := seg[1 : len(seg)-1]
			greedy := strings.HasSuffix(name, "+")
			if greedy {
				name = strings.TrimSuffix(name, "+")
			}
			raw := actualSegs[ai]
			if greedy {
				raw = strings.Join(actualSegs[ai:], "/")
				ai = len(actualSegs)
			} else {
				ai++
			}
			value, err := url.PathUnescape(raw)
			if err != nil {
				value = raw
			}
			params[name] = value
			if lower := strings.ToLower(name); lower != name {
				params[lower] = value
			}
			if len(name) > 0 {
				camel := strings.ToLower(name[:1]) + name[1:]
				if camel != name {
					params[camel] = value
				}
			}
			pi++
			continue
		}
		if seg != actualSegs[ai] {
			return nil, false
		}
		pi++
		ai++
	}
	if ai != len(actualSegs) {
		return nil, false
	}
	return params, true
}

func normalizeLambdaPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
		if path == "" {
			return "/"
		}
	}
	return path
}

func splitLambdaPath(path string) []string {
	path = normalizeLambdaPath(path)
	if path == "/" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func decodeLambdaJSONBody(r *http.Request, out any) error {
	body, err := readBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func parseLambdaIntQuery(values url.Values, key string) int {
	value := strings.TrimSpace(values.Get(key))
	if value == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func lambdaParam(params map[string]string, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(params[key])
		if value != "" {
			return value
		}
		if lower := strings.ToLower(key); lower != key {
			value = strings.TrimSpace(params[lower])
			if value != "" {
				return value
			}
		}
		if len(key) > 0 {
			camel := strings.ToLower(key[:1]) + key[1:]
			value = strings.TrimSpace(params[camel])
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func lambdaMapString(values map[string]any, key string) string {
	raw, ok := values[key]
	if !ok || raw == nil {
		return ""
	}
	switch tv := raw.(type) {
	case string:
		return strings.TrimSpace(tv)
	default:
		return ""
	}
}

func lambdaMapInt32(values map[string]any, key string) int32 {
	raw, ok := values[key]
	if !ok || raw == nil {
		return 0
	}
	switch tv := raw.(type) {
	case int32:
		return tv
	case int64:
		return int32(tv)
	case int:
		return int32(tv)
	case float64:
		return int32(tv)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(tv))
		if err == nil {
			return int32(n)
		}
	}
	return 0
}

func lambdaResourceARN(params map[string]string, requestPath string) string {
	resource := strings.TrimSpace(params["resource"])
	if resource == "" {
		resource = strings.TrimSpace(params["Resource"])
	}
	if resource != "" {
		return resource
	}
	path := normalizeLambdaPath(requestPath)
	const prefix = "/2017-03-31/tags/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	resource = strings.TrimPrefix(path, prefix)
	decoded, err := url.PathUnescape(resource)
	if err != nil {
		return resource
	}
	return decoded
}

func lambdaFunctionConfigurationPayload(fn lambdasvc.Function) map[string]any {
	return map[string]any{
		"FunctionName":     fn.Name,
		"FunctionArn":      fn.ARN,
		"Runtime":          fn.Runtime,
		"Role":             fn.Role,
		"Handler":          fn.Handler,
		"CodeSize":         fn.CodeSize,
		"Description":      fn.Description,
		"Timeout":          fn.Timeout,
		"MemorySize":       fn.MemorySize,
		"LastModified":     fn.LastModified.Format(time.RFC3339),
		"CodeSha256":       fn.CodeSHA256,
		"Version":          fn.Version,
		"RevisionId":       fn.RevisionID,
		"PackageType":      fn.PackageType,
		"State":            fn.State,
		"LastUpdateStatus": fn.LastUpdate,
		"Architectures":    fn.Architectures,
	}
}

func lambdaAliasPayload(alias lambdasvc.Alias) map[string]any {
	return map[string]any{
		"AliasArn":        alias.ARN,
		"Name":            alias.Name,
		"FunctionVersion": alias.FunctionVersion,
		"Description":     alias.Description,
		"RevisionId":      alias.RevisionID,
	}
}

type lambdaCreateFunctionInput struct {
	FunctionName  string            `json:"FunctionName"`
	Role          string            `json:"Role"`
	Handler       string            `json:"Handler"`
	Runtime       string            `json:"Runtime"`
	Description   string            `json:"Description"`
	Timeout       int32             `json:"Timeout"`
	MemorySize    int32             `json:"MemorySize"`
	Architectures []string          `json:"Architectures"`
	Tags          map[string]string `json:"Tags"`
	Publish       bool              `json:"Publish"`
	Code          struct {
		ZipFile  []byte `json:"ZipFile"`
		S3Bucket string `json:"S3Bucket"`
		S3Key    string `json:"S3Key"`
	} `json:"Code"`
}

type lambdaUpdateFunctionConfigurationInput struct {
	Runtime       *string  `json:"Runtime"`
	Role          *string  `json:"Role"`
	Handler       *string  `json:"Handler"`
	Description   *string  `json:"Description"`
	Timeout       *int32   `json:"Timeout"`
	MemorySize    *int32   `json:"MemorySize"`
	Architectures []string `json:"Architectures"`
}

type lambdaUpdateFunctionCodeInput struct {
	ZipFile  []byte `json:"ZipFile"`
	S3Bucket string `json:"S3Bucket"`
	S3Key    string `json:"S3Key"`
	Publish  bool   `json:"Publish"`
}

type lambdaPublishVersionInput struct {
	Description string `json:"Description"`
}

type lambdaCreateAliasInput struct {
	Name            string `json:"Name"`
	FunctionVersion string `json:"FunctionVersion"`
	Description     string `json:"Description"`
}

type lambdaUpdateAliasInput struct {
	FunctionVersion string  `json:"FunctionVersion"`
	Description     *string `json:"Description"`
}

type lambdaAddPermissionInput struct {
	StatementID   string `json:"StatementId"`
	Action        string `json:"Action"`
	Principal     string `json:"Principal"`
	SourceARN     string `json:"SourceArn"`
	SourceAccount string `json:"SourceAccount"`
}

type lambdaTagResourceInput struct {
	Tags map[string]string `json:"Tags"`
}
