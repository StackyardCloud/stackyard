package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/google/uuid"
)

// Amazon SageMaker Runtime (runtime.sagemaker, REST-JSON, SigV4 name
// "sagemaker"). Invocations resolve against the SageMaker control-plane store;
// there is no model, so every invocation echoes its input back.
//
// API reference:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_SageMaker_Runtime.html

const (
	sagemakerRuntimeMaxBody      = 6291456 // BodyBlob max
	sagemakerRuntimeMaxAsyncBody = 128000  // AsyncBodyBlob max
	// ponytail: streamed responses are cut into fixed-size PayloadPart events.
	sagemakerRuntimeStreamChunk = 256
)

var (
	sagemakerRuntimePath = regexp.MustCompile(`^/endpoints/([^/]+)/(invocations|async-invocations|invocations-response-stream)$`)

	sagemakerRuntimeNameRE      = regexp.MustCompile(`^[a-zA-Z0-9](-*[a-zA-Z0-9])*$`)
	sagemakerRuntimeComponentRE = regexp.MustCompile(`^[a-zA-Z0-9]([\-a-zA-Z0-9]*[a-zA-Z0-9])?$`)
	sagemakerRuntimePrintableRE = regexp.MustCompile(`^\S[[:print:]]*$`)
	sagemakerRuntimeASCIIRE     = regexp.MustCompile(`^[[:ascii:]]*$`)
	sagemakerRuntimeLocationRE  = regexp.MustCompile(`^(https|s3)://([^/]+)/?(.*)$`)
	sagemakerRuntimeExtensionRE = regexp.MustCompile(`^[a-zA-Z0-9!_.*'()/-]+$`)
	sagemakerRuntimeFilenameRE  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_.]*$`)
)

var sagemakerRuntimeOpByPath = map[string]string{
	"invocations":                 "InvokeEndpoint",
	"async-invocations":           "InvokeEndpointAsync",
	"invocations-response-stream": "InvokeEndpointWithResponseStream",
}

// sagemakerRuntimeHeader is one validated request header from the model.
type sagemakerRuntimeHeader struct {
	name     string
	min, max int
	re       *regexp.Regexp
}

var sagemakerRuntimeHeaders = map[string][]sagemakerRuntimeHeader{
	"InvokeEndpoint": {
		{"Content-Type", 0, 1024, sagemakerRuntimeASCIIRE},
		{"Accept", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Custom-Attributes", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Target-Model", 1, 1024, sagemakerRuntimePrintableRE},
		{"X-Amzn-SageMaker-Target-Variant", 0, 63, sagemakerRuntimeNameRE},
		{"X-Amzn-SageMaker-Target-Container-Hostname", 0, 63, sagemakerRuntimeNameRE},
		{"X-Amzn-SageMaker-Inference-Id", 1, 64, sagemakerRuntimePrintableRE},
		{"X-Amzn-SageMaker-Enable-Explanations", 1, 64, nil},
		{"X-Amzn-SageMaker-Inference-Component", 0, 63, sagemakerRuntimeComponentRE},
		{"X-Amzn-SageMaker-Prefix-Aware-Id", 1, 64, sagemakerRuntimePrintableRE},
	},
	"InvokeEndpointWithResponseStream": {
		{"Content-Type", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Accept", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Custom-Attributes", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Target-Variant", 0, 63, sagemakerRuntimeNameRE},
		{"X-Amzn-SageMaker-Target-Container-Hostname", 0, 63, sagemakerRuntimeNameRE},
		{"X-Amzn-SageMaker-Inference-Id", 1, 64, sagemakerRuntimePrintableRE},
		{"X-Amzn-SageMaker-Inference-Component", 0, 63, sagemakerRuntimeComponentRE},
		{"X-Amzn-SageMaker-Session-Id", 0, 256, sagemakerRuntimeNameRE},
		{"X-Amzn-SageMaker-Prefix-Aware-Id", 1, 64, sagemakerRuntimePrintableRE},
	},
	"InvokeEndpointAsync": {
		{"X-Amzn-SageMaker-Content-Type", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Accept", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Custom-Attributes", 0, 1024, sagemakerRuntimeASCIIRE},
		{"X-Amzn-SageMaker-Inference-Id", 1, 64, sagemakerRuntimePrintableRE},
		{"X-Amzn-SageMaker-InputLocation", 1, 1024, sagemakerRuntimeLocationRE},
		{"X-Amzn-SageMaker-S3OutputPathExtension", 1, 512, sagemakerRuntimeExtensionRE},
		{"X-Amzn-SageMaker-Filename", 1, 32, sagemakerRuntimeFilenameRE},
	},
}

// sagemakerRuntimeRoute returns the endpoint name and operation for a runtime
// request. It only claims POSTs to the three invocation paths that are signed
// for sagemaker or sent to a runtime.sagemaker host, so the control plane
// (X-Amz-Target "SageMaker.*") and Augmented AI keep their requests.
func sagemakerRuntimeRoute(r *http.Request) (string, string, bool) {
	if r.Method != http.MethodPost || strings.TrimSpace(r.Header.Get("X-Amz-Target")) != "" {
		return "", "", false
	}
	m := sagemakerRuntimePath.FindStringSubmatch(rawRequestPath(r))
	if m == nil {
		return "", "", false
	}
	if sigV4ServiceHint(r) != "sagemaker" && !strings.Contains(strings.ToLower(r.Host), "runtime.sagemaker") {
		return "", "", false
	}
	name, err := url.PathUnescape(m[1])
	if err != nil {
		name = m[1]
	}
	return name, sagemakerRuntimeOpByPath[m[2]], true
}

func (s *Server) handleSageMakerRuntimeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	name, op, ok := sagemakerRuntimeRoute(r)
	if !ok {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "sagemaker"); !ok {
		respondSageMakerRuntimeError(w, status, code, msg)
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", "unable to read request body")
		return true
	}
	if msg := sagemakerRuntimeValidate(r, op, name, body); msg != "" {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", msg)
		return true
	}
	target, msg := s.sagemaker.runtimeTarget(name, r.Header.Get("X-Amzn-SageMaker-Target-Variant"), op == "InvokeEndpointAsync")
	if msg != "" {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", msg)
		return true
	}
	switch op {
	case "InvokeEndpoint":
		s.sagemakerRuntimeInvoke(w, r, name, target, body)
	case "InvokeEndpointWithResponseStream":
		s.sagemakerRuntimeInvokeStream(w, r, name, target, body)
	default:
		s.sagemakerRuntimeInvokeAsync(w, r, target, body)
	}
	return true
}

func sagemakerRuntimeValidate(r *http.Request, op, name string, body []byte) string {
	if len(name) > 63 || !sagemakerRuntimeNameRE.MatchString(name) {
		return fmt.Sprintf("1 validation error detected: Value '%s' at 'endpointName' failed to satisfy constraint: Member must satisfy regular expression pattern: ^[a-zA-Z0-9](-*[a-zA-Z0-9])*", name)
	}
	for _, h := range sagemakerRuntimeHeaders[op] {
		value, present := r.Header[http.CanonicalHeaderKey(h.name)]
		if !present {
			continue
		}
		v := strings.Join(value, ",")
		if len(v) < h.min || len(v) > h.max || (h.re != nil && v != "" && !h.re.MatchString(v)) {
			return fmt.Sprintf("1 validation error detected: Value at '%s' failed to satisfy constraint: Member must have length between %d and %d and match the documented pattern", h.name, h.min, h.max)
		}
	}
	switch op {
	case "InvokeEndpoint":
		if sid := r.Header.Get("X-Amzn-SageMaker-Session-Id"); sid != "" && sid != "NEW_SESSION" && (len(sid) > 256 || !sagemakerRuntimeNameRE.MatchString(sid)) {
			return "1 validation error detected: Value at 'sessionId' failed to satisfy constraint: Member must be NEW_SESSION or a session ID"
		}
		if len(body) > sagemakerRuntimeMaxBody {
			return "Request body exceeds the 6 MB invocation payload limit"
		}
	case "InvokeEndpointWithResponseStream":
		if len(body) > sagemakerRuntimeMaxBody {
			return "Request body exceeds the 6 MB invocation payload limit"
		}
	case "InvokeEndpointAsync":
		if len(body) > sagemakerRuntimeMaxAsyncBody {
			return "Request body exceeds the 128000 byte inline payload limit; use InputLocation"
		}
		if len(body) == 0 && r.Header.Get("X-Amzn-SageMaker-InputLocation") == "" {
			return "InputLocation is required when no inline Body is sent"
		}
		if strings.Count(r.Header.Get("X-Amzn-SageMaker-Filename"), ".") > 1 {
			return "Filename may contain at most one '.'"
		}
		for header, bounds := range map[string][2]int{"X-Amzn-SageMaker-RequestTTLSeconds": {60, 21600}, "X-Amzn-SageMaker-InvocationTimeoutSeconds": {1, 3600}} {
			raw := r.Header.Get(header)
			if raw == "" {
				continue
			}
			if n, err := strconv.Atoi(raw); err != nil || n < bounds[0] || n > bounds[1] {
				return fmt.Sprintf("%s must be an integer between %d and %d", header, bounds[0], bounds[1])
			}
		}
	}
	return ""
}

// sagemakerRuntimeTarget is what an invocation needs from the control plane.
type sagemakerRuntimeTarget struct {
	Variant     string
	OutputPath  string
	FailurePath string
}

// runtimeTarget resolves an endpoint for invocation. It returns a
// ValidationError message when the endpoint is missing, not InService, of the
// wrong kind (real-time vs. asynchronous) or has no such production variant.
func (s *sagemakerStore) runtimeTarget(name, variant string, async bool) (sagemakerRuntimeTarget, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoint, ok := s.endpoints[name]
	if !ok {
		return sagemakerRuntimeTarget{}, fmt.Sprintf("Endpoint %s of account %s not found.", name, "123456789012")
	}
	if endpoint.Status == "Creating" || endpoint.Status == "Updating" {
		// Endpoints finish deploying at once, as DescribeEndpoint already reports.
		endpoint.Status = "InService"
		endpoint.LastModifiedTime = time.Now().UTC().Format(time.RFC3339)
	}
	if endpoint.Status != "InService" {
		return sagemakerRuntimeTarget{}, fmt.Sprintf("Endpoint %s is not in service (status %s).", name, endpoint.Status)
	}
	variants := []string{"AllTraffic"}
	var asyncConfig map[string]any
	if ec, ok := s.endpointConfigs[endpoint.EndpointConfigName]; ok {
		asyncConfig = ec.AsyncInferenceConfig
		if names := sagemakerVariantNames(ec.ProductionVariants); len(names) > 0 {
			variants = names
		}
	}
	target := sagemakerRuntimeTarget{Variant: variants[0]}
	if variant != "" {
		found := false
		for _, v := range variants {
			found = found || v == variant
		}
		if !found {
			return sagemakerRuntimeTarget{}, fmt.Sprintf("Variant %s not found for endpoint %s.", variant, name)
		}
		target.Variant = variant
	}
	output, _ := asyncConfig["OutputConfig"].(map[string]any)
	target.OutputPath, _ = output["S3OutputPath"].(string)
	target.FailurePath, _ = output["S3FailurePath"].(string)
	switch {
	case async && asyncConfig == nil:
		return sagemakerRuntimeTarget{}, fmt.Sprintf("Endpoint %s is not configured for asynchronous inference.", name)
	case !async && asyncConfig != nil:
		return sagemakerRuntimeTarget{}, fmt.Sprintf("Endpoint %s is configured for asynchronous inference; use InvokeEndpointAsync.", name)
	}
	return target, ""
}

// runtimeSession opens a stateful session ("NEW_SESSION") or checks that an
// existing one belongs to the endpoint. It returns the new session ID, if any.
func (s *sagemakerStore) runtimeSession(endpoint, sessionID string) (string, string) {
	if sessionID == "" {
		return "", ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtimeSessions == nil {
		s.runtimeSessions = map[string]string{}
	}
	if sessionID == "NEW_SESSION" {
		// ponytail: sessions never expire; add a TTL sweep if clients rely on expiry.
		id := strings.ReplaceAll(uuid.NewString(), "-", "")
		s.runtimeSessions[id] = endpoint
		return id, ""
	}
	if s.runtimeSessions[sessionID] != endpoint {
		return "", fmt.Sprintf("Session %s not found for endpoint %s.", sessionID, endpoint)
	}
	return "", ""
}

func sagemakerVariantNames(variants []any) []string {
	out := make([]string, 0, len(variants))
	for _, v := range variants {
		if m, ok := v.(map[string]any); ok {
			if name, _ := m["VariantName"].(string); name != "" {
				out = append(out, name)
			}
		}
	}
	return out
}

func (s *Server) sagemakerRuntimeInvoke(w http.ResponseWriter, r *http.Request, name string, target sagemakerRuntimeTarget, body []byte) {
	newSession, msg := s.sagemaker.runtimeSession(name, r.Header.Get("X-Amzn-SageMaker-Session-Id"))
	if msg != "" {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", msg)
		return
	}
	h := w.Header()
	h.Set("Content-Type", firstNonEmptyString(r.Header.Get("Accept"), r.Header.Get("Content-Type"), "application/octet-stream"))
	h.Set("x-Amzn-Invoked-Production-Variant", target.Variant)
	if attrs := r.Header.Get("X-Amzn-SageMaker-Custom-Attributes"); attrs != "" {
		h.Set("X-Amzn-SageMaker-Custom-Attributes", attrs)
	}
	if newSession != "" {
		h.Set("X-Amzn-SageMaker-New-Session-Id", newSession+"; Expires="+time.Now().UTC().Add(20*time.Minute).Format(time.RFC3339))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) sagemakerRuntimeInvokeStream(w http.ResponseWriter, r *http.Request, name string, target sagemakerRuntimeTarget, body []byte) {
	if sid := r.Header.Get("X-Amzn-SageMaker-Session-Id"); sid != "" {
		if _, msg := s.sagemaker.runtimeSession(name, sid); msg != "" {
			respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", msg)
			return
		}
	}
	var stream bytes.Buffer
	enc := eventstream.NewEncoder()
	for start := 0; start == 0 || start < len(body); start += sagemakerRuntimeStreamChunk {
		end := min(start+sagemakerRuntimeStreamChunk, len(body))
		msg := eventstream.Message{Payload: body[start:end]}
		msg.Headers.Set(":message-type", eventstream.StringValue("event"))
		msg.Headers.Set(":event-type", eventstream.StringValue("PayloadPart"))
		msg.Headers.Set(":content-type", eventstream.StringValue("application/octet-stream"))
		if err := enc.Encode(&stream, msg); err != nil {
			respondSageMakerRuntimeError(w, http.StatusInternalServerError, "InternalFailure", "unable to encode response stream")
			return
		}
	}
	h := w.Header()
	h.Set("Content-Type", "application/vnd.amazon.eventstream")
	h.Set("X-Amzn-SageMaker-Content-Type", firstNonEmptyString(r.Header.Get("X-Amzn-SageMaker-Accept"), r.Header.Get("Content-Type"), "application/octet-stream"))
	h.Set("x-Amzn-Invoked-Production-Variant", target.Variant)
	if attrs := r.Header.Get("X-Amzn-SageMaker-Custom-Attributes"); attrs != "" {
		h.Set("X-Amzn-SageMaker-Custom-Attributes", attrs)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(stream.Bytes())
}

// sagemakerRuntimeInvokeAsync "runs" the inference at once: the echoed input
// is written to the endpoint's S3OutputPath in the emulator's S3, or an error
// object to S3FailurePath when the input can't be read or written.
func (s *Server) sagemakerRuntimeInvokeAsync(w http.ResponseWriter, r *http.Request, target sagemakerRuntimeTarget, body []byte) {
	id := firstNonEmptyString(r.Header.Get("X-Amzn-SageMaker-Inference-Id"), uuid.NewString())
	ext := strings.Trim(r.Header.Get("X-Amzn-SageMaker-S3OutputPathExtension"), "/")
	output := sagemakerRuntimeJoinS3(target.OutputPath, ext, firstNonEmptyString(r.Header.Get("X-Amzn-SageMaker-Filename"), id+".out"))
	failure := ""
	if target.FailurePath != "" {
		failure = sagemakerRuntimeJoinS3(target.FailurePath, ext, id+"-error.out")
	}

	input, readErr := body, ""
	if len(input) == 0 {
		input, readErr = s.sagemakerRuntimeReadS3(r.Header.Get("X-Amzn-SageMaker-InputLocation"))
	}
	contentType := firstNonEmptyString(r.Header.Get("X-Amzn-SageMaker-Accept"), r.Header.Get("X-Amzn-SageMaker-Content-Type"), "application/octet-stream")
	if readErr == "" {
		readErr = s.sagemakerRuntimeWriteS3(output, contentType, input)
	}
	if readErr != "" && failure != "" {
		// ponytail: a failure path that can't be written is dropped silently, as
		// there is no notification channel to report it on.
		_ = s.sagemakerRuntimeWriteS3(failure, "application/json", []byte(`{"error":`+strconv.Quote(readErr)+`}`))
	}

	h := w.Header()
	h.Set("X-Amzn-SageMaker-OutputLocation", output)
	if failure != "" {
		h.Set("X-Amzn-SageMaker-FailureLocation", failure)
	}
	respondSageMakerRuntimeJSON(w, http.StatusAccepted, map[string]string{"InferenceId": id})
}

func sagemakerRuntimeJoinS3(base string, parts ...string) string {
	out := strings.TrimRight(base, "/")
	for _, p := range parts {
		if p != "" {
			out += "/" + p
		}
	}
	return out
}

func sagemakerRuntimeSplitS3(location string) (string, string, bool) {
	rest, ok := strings.CutPrefix(location, "s3://")
	if !ok {
		return "", "", false
	}
	bucket, key, _ := strings.Cut(rest, "/")
	return bucket, key, bucket != "" && key != ""
}

func (s *Server) sagemakerRuntimeReadS3(location string) ([]byte, string) {
	bucket, key, ok := sagemakerRuntimeSplitS3(location)
	if !ok {
		return nil, fmt.Sprintf("InputLocation %s is not an s3:// object URI", location)
	}
	obj, err := s.s3.GetObject(bucket, key)
	if err != nil {
		return nil, fmt.Sprintf("Unable to read input from %s: %v", location, err)
	}
	return bytes.Clone(obj.Body), ""
}

func (s *Server) sagemakerRuntimeWriteS3(location, contentType string, body []byte) string {
	bucket, key, ok := sagemakerRuntimeSplitS3(location)
	if !ok {
		return fmt.Sprintf("Output location %s is not an s3:// object URI", location)
	}
	if _, err := s.s3.PutObject(bucket, key, contentType, body, nil, "private", "STANDARD"); err != nil {
		return fmt.Sprintf("Unable to write output to %s: %v", location, err)
	}
	return ""
}

func respondSageMakerRuntimeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSageMakerRuntimeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSageMakerRuntimeJSON(w, status, map[string]string{"__type": code, "message": msg})
}
