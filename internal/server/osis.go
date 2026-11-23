package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const osisPathPrefix = "/2022-01-01/osis/"

// handleOSISRESTRouter serves Amazon OpenSearch Ingestion (REST-JSON, signing
// name osis). The ingestion data plane isn't emulated.
func (s *Server) handleOSISRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isOSISCandidate(r) {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > osisMaxBody {
		respondOSISError(w, osisValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "osis"); !ok {
		respondOSISError(w, &osisError{status, code, msg})
		return true
	}
	out, err := s.osisRoute(r)
	if err != nil {
		respondOSISError(w, err)
		return true
	}
	raw, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	return true
}

func isOSISCandidate(r *http.Request) bool {
	if !strings.HasPrefix(rawRequestPath(r), osisPathPrefix) {
		return false
	}
	service := strings.TrimSpace(sigV4ServiceHint(r))
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return service == "osis" || (service == "" && (strings.HasPrefix(host, "osis.") || strings.Contains(host, ".osis.")))
}

func respondOSISError(w http.ResponseWriter, e *osisError) {
	raw, _ := json.Marshal(map[string]any{"message": e.msg})
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_, _ = w.Write(raw)
}

func (s *Server) osisRoute(r *http.Request) (any, *osisError) {
	rest := strings.TrimPrefix(rawRequestPath(r), osisPathPrefix)
	action, param, _ := strings.Cut(rest, "/")
	param, perr := url.PathUnescape(param)
	if perr != nil {
		return nil, osisValidation("the request path is not valid")
	}
	q := r.URL.Query()
	m := r.Method
	st := s.osis
	switch {
	case action == "createPipeline" && m == http.MethodPost && param == "":
		return s.osisCreate(r)
	case action == "updatePipeline" && m == http.MethodPut:
		if err := osisCheckName(param); err != nil {
			return nil, err
		}
		in, err := osisDecode(r, "UpdatePipelineRequest")
		if err != nil {
			return nil, err
		}
		spec, err := s.osisSpec(in, false)
		if err != nil {
			return nil, err
		}
		return st.update(param, spec)
	case (action == "startPipeline" || action == "stopPipeline") && m == http.MethodPut:
		if err := osisCheckName(param); err != nil {
			return nil, err
		}
		return st.setRunning(param, action == "startPipeline")
	case action == "getPipeline" && m == http.MethodGet:
		if err := osisCheckName(param); err != nil {
			return nil, err
		}
		return st.get(param)
	case action == "deletePipeline" && m == http.MethodDelete:
		if err := osisCheckName(param); err != nil {
			return nil, err
		}
		return st.remove(param)
	case action == "getPipelineChangeProgress" && m == http.MethodGet:
		if err := osisCheckName(param); err != nil {
			return nil, err
		}
		return st.changeProgress(param)
	case action == "listPipelines" && m == http.MethodGet && param == "":
		max, err := osisMaxResults(q)
		if err != nil {
			return nil, err
		}
		return st.list(q.Get("nextToken"), max)
	case action == "validatePipeline" && m == http.MethodPost && param == "":
		in, err := osisDecode(r, "ValidatePipelineRequest")
		if err != nil {
			return nil, err
		}
		problems := osisCheckConfig(osisStr(in, "PipelineConfigurationBody"))
		errs := make([]any, 0, len(problems))
		for _, p := range problems {
			errs = append(errs, map[string]any{"Message": p})
		}
		return map[string]any{"isValid": len(problems) == 0, "Errors": errs}, nil
	case action == "listPipelineBlueprints" && m == http.MethodPost && param == "":
		out := make([]any, 0, len(osisBlueprints))
		for _, b := range osisBlueprints {
			out = append(out, map[string]any{"BlueprintName": b.name, "DisplayName": b.display, "DisplayDescription": b.description, "Service": b.service, "UseCase": b.useCase})
		}
		return map[string]any{"Blueprints": out}, nil
	case action == "getPipelineBlueprint" && m == http.MethodGet:
		return osisGetBlueprint(param, q.Get("format"))
	case action == "resourcePolicy":
		return s.osisPolicyRoute(r, param)
	case action == "listTagsForResource" && m == http.MethodGet && param == "":
		arn, err := osisQueryARN(q)
		if err != nil {
			return nil, err
		}
		return st.listTags(arn)
	case action == "tagResource" && m == http.MethodPost && param == "":
		arn, err := osisQueryARN(q)
		if err != nil {
			return nil, err
		}
		in, err := osisDecode(r, "TagResourceRequest")
		if err != nil {
			return nil, err
		}
		tags, err := osisTags(in["Tags"])
		if err != nil {
			return nil, err
		}
		return map[string]any{}, st.tag(arn, tags)
	case action == "untagResource" && m == http.MethodPost && param == "":
		arn, err := osisQueryARN(q)
		if err != nil {
			return nil, err
		}
		in, err := osisDecode(r, "UntagResourceRequest")
		if err != nil {
			return nil, err
		}
		return map[string]any{}, st.untag(arn, osisStrings(in["TagKeys"]))
	case action == "createPipelineEndpoint" && m == http.MethodPost && param == "":
		return s.osisCreateEndpoint(r)
	case action == "deletePipelineEndpoint" && m == http.MethodDelete:
		if !osisEndpointID.MatchString(param) {
			return nil, osisValidation("EndpointId is not valid.")
		}
		return st.deleteEndpoint(param)
	case (action == "listPipelineEndpoints" || action == "listPipelineEndpointConnections") && m == http.MethodGet && param == "":
		max, err := osisMaxResults(q)
		if err != nil {
			return nil, err
		}
		return st.listEndpoints(q.Get("nextToken"), max, action == "listPipelineEndpointConnections")
	case action == "revokePipelineEndpointConnections" && m == http.MethodPost && param == "":
		in, err := osisDecode(r, "RevokePipelineEndpointConnectionsRequest")
		if err != nil {
			return nil, err
		}
		return st.revoke(osisStr(in, "PipelineArn"), osisStrings(in["EndpointIds"]))
	}
	return nil, &osisError{http.StatusNotFound, "UnknownOperationException", "no OpenSearch Ingestion operation matches " + m + " " + rawRequestPath(r)}
}

func (s *Server) osisCreate(r *http.Request) (any, *osisError) {
	in, err := osisDecode(r, "CreatePipelineRequest")
	if err != nil {
		return nil, err
	}
	name := osisStr(in, "PipelineName")
	if err := osisCheckName(name); err != nil {
		return nil, err
	}
	spec, err := s.osisSpec(in, true)
	if err != nil {
		return nil, err
	}
	return s.osis.create(name, osisHost(r), spec)
}

// osisSpec validates the pipeline fields shared by create and update. VPC
// checks read the EC2 store before the osis lock is taken.
func (s *Server) osisSpec(in map[string]any, create bool) (osisSpec, *osisError) {
	var spec osisSpec
	for key, dst := range map[string]**int{"MinUnits": &spec.minUnits, "MaxUnits": &spec.maxUnits} {
		if f, ok := in[key].(float64); ok {
			if f < 1 || f > osisMaxUnits {
				return spec, osisValidation("%s must be between 1 and %d.", key, osisMaxUnits)
			}
			v := int(f)
			*dst = &v
		}
	}
	if spec.minUnits != nil && spec.maxUnits != nil && *spec.minUnits > *spec.maxUnits {
		return spec, osisValidation("MinUnits (%d) can't be greater than MaxUnits (%d).", *spec.minUnits, *spec.maxUnits)
	}
	if body := osisStr(in, "PipelineConfigurationBody"); body != "" {
		if problems := osisCheckConfig(body); len(problems) > 0 {
			return spec, osisValidation("The pipeline configuration is invalid: %s", problems[0])
		}
		spec.body, spec.destinations = body, osisDestinations(body)
	}
	spec.roleArn = osisStr(in, "PipelineRoleArn")
	spec.logOptions, spec.hasLog = osisObject(in, "LogPublishingOptions")
	if spec.hasLog && osisBool(spec.logOptions, "IsLoggingEnabled") {
		if dest, _ := spec.logOptions["CloudWatchLogDestination"].(map[string]any); osisStr(dest, "LogGroup") == "" {
			return spec, osisValidation("CloudWatchLogDestination is required when IsLoggingEnabled is true.")
		}
	}
	spec.buffer, spec.hasBuffer = osisObject(in, "BufferOptions")
	spec.encryption, spec.hasEn = osisObject(in, "EncryptionAtRestOptions")
	if create && osisBool(spec.buffer, "PersistentBufferEnabled") && *spec.minUnits < 2 {
		return spec, osisValidation("A pipeline with persistent buffering needs MinUnits of at least 2.")
	}
	if vpc, ok := osisObject(in, "VpcOptions"); ok {
		vpcID, err := s.osisCheckNetwork(osisStrings(vpc["SubnetIds"]), osisStrings(vpc["SecurityGroupIds"]))
		if err != nil {
			return spec, err
		}
		spec.vpcOptions, spec.vpcID = vpc, vpcID
	}
	tags, err := osisTags(in["Tags"])
	if err != nil {
		return spec, err
	}
	spec.tags = tags
	return spec, nil
}

// osisCheckNetwork checks subnets and security groups against the EC2 store;
// all must share one VPC, whose id it returns.
func (s *Server) osisCheckNetwork(subnets, groups []string) (string, *osisError) {
	if len(subnets) == 0 {
		return "", osisValidation("SubnetIds must name at least one subnet.")
	}
	found := s.ec2.DescribeSubnets(subnets, nil)
	if len(found) != len(osisUnique(subnets)) {
		return "", osisValidation("One or more subnets in SubnetIds do not exist.")
	}
	vpc := found[0].VpcID
	for _, sn := range found {
		if sn.VpcID != vpc {
			return "", osisValidation("All subnets in SubnetIds must belong to the same VPC.")
		}
	}
	if len(groups) == 0 {
		return vpc, nil
	}
	sgs := s.ec2.DescribeSecurityGroups(groups, nil, "")
	if len(sgs) != len(osisUnique(groups)) {
		return "", osisValidation("One or more security groups in SecurityGroupIds do not exist.")
	}
	for _, g := range sgs {
		if g.VpcID != vpc {
			return "", osisValidation("Security group %s is not in the subnets' VPC %s.", g.ID, vpc)
		}
	}
	return vpc, nil
}

func (s *Server) osisCreateEndpoint(r *http.Request) (any, *osisError) {
	in, err := osisDecode(r, "CreatePipelineEndpointRequest")
	if err != nil {
		return nil, err
	}
	vpc, _ := in["VpcOptions"].(map[string]any)
	subnets, groups := osisStrings(vpc["SubnetIds"]), osisStrings(vpc["SecurityGroupIds"])
	if len(subnets) == 0 {
		return nil, osisValidation("VpcOptions.SubnetIds is required.")
	}
	vpcID, err := s.osisCheckNetwork(subnets, groups)
	if err != nil {
		return nil, err
	}
	return s.osis.createEndpoint(osisStr(in, "PipelineArn"), vpcID, osisHost(r), subnets, groups)
}

func (s *Server) osisPolicyRoute(r *http.Request, arn string) (any, *osisError) {
	if !osisPipelineARN.MatchString(arn) {
		return nil, osisValidation("ResourceArn is not a valid pipeline ARN.")
	}
	switch r.Method {
	case http.MethodGet:
		return s.osis.getPolicy(arn)
	case http.MethodDelete:
		return s.osis.deletePolicy(arn)
	case http.MethodPut:
		in, err := osisDecode(r, "PutResourcePolicyRequest")
		if err != nil {
			return nil, err
		}
		policy := osisStr(in, "Policy")
		var doc map[string]any
		if json.Unmarshal([]byte(policy), &doc) != nil || doc["Statement"] == nil {
			return nil, osisValidation("Policy must be a JSON policy document with a Statement.")
		}
		return s.osis.putPolicy(arn, policy)
	}
	return nil, &osisError{http.StatusNotFound, "UnknownOperationException", "no OpenSearch Ingestion operation matches " + r.Method + " " + rawRequestPath(r)}
}

func osisGetBlueprint(name, format string) (any, *osisError) {
	b := osisFindBlueprint(name)
	if b == nil {
		return nil, osisNotFound("Blueprint %s does not exist.", name)
	}
	switch format {
	case "", "YAML":
		format = "YAML"
	case "JSON":
	default:
		return nil, osisValidation("format must be YAML or JSON.")
	}
	body := b.yaml
	if format == "JSON" {
		body = osisYAMLToJSON(b.yaml)
	}
	return map[string]any{"Format": format, "Blueprint": map[string]any{
		"BlueprintName": b.name, "PipelineConfigurationBody": body, "DisplayName": b.display,
		"DisplayDescription": b.description, "Service": b.service, "UseCase": b.useCase,
	}}, nil
}

// osisDecode reads a JSON body and checks it against the model input shape;
// unknown members are rejected, so callers can't set anything else.
func osisDecode(r *http.Request, shape string) (map[string]any, *osisError) {
	raw, err := readBodyBytes(r)
	if err != nil {
		return nil, osisValidation("unable to read request body")
	}
	in := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &in); err != nil {
			return nil, osisValidation("request body is not a JSON object")
		}
	}
	if problems := awsmodels.OSISModel().Validate(shape, in); len(problems) > 0 {
		return nil, osisValidation("%s", problems[0])
	}
	conformed, _ := awsmodels.OSISModel().Conform(shape, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	return conformed, nil
}

func osisCheckName(name string) *osisError {
	if len(name) < 3 || len(name) > 28 || !osisNamePattern.MatchString(name) {
		return osisValidation("PipelineName must be 3 to 28 lowercase letters, numbers or hyphens, starting with a letter.")
	}
	return nil
}

func osisQueryARN(q url.Values) (string, *osisError) {
	arn := q.Get("arn")
	if !osisPipelineARN.MatchString(arn) {
		return "", osisValidation("arn must be a pipeline ARN.")
	}
	return arn, nil
}

func osisMaxResults(q url.Values) (int, *osisError) {
	raw := q.Get("maxResults")
	if raw == "" {
		return 100, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 || n > 100 {
		return 0, osisValidation("maxResults must be between 1 and 100.")
	}
	return int(n), nil
}

func osisTags(v any) ([]osisTag, *osisError) {
	list, _ := v.([]any)
	if len(list) > osisMaxTags {
		return nil, osisLimit("A pipeline can have at most %d tags.", osisMaxTags)
	}
	out := []osisTag{}
	for _, item := range list {
		m, _ := item.(map[string]any)
		key := osisStr(m, "Key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, osisValidation("Tag keys can't start with aws:.")
		}
		out = osisMergeTags(out, []osisTag{{key, osisStr(m, "Value")}})
	}
	return out, nil
}

// osisObject returns a member decoded as an object, and whether it was given.
func osisObject(in map[string]any, key string) (map[string]any, bool) {
	m, ok := in[key].(map[string]any)
	return m, ok
}

func osisStrings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func osisUnique(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// osisHost is the base URL ingest endpoints point at (Stackyard itself).
func osisHost(r *http.Request) string {
	host := r.Host
	if host == "" {
		host = "localhost"
	}
	return host
}
