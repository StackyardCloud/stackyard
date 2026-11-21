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
	"unicode/utf8"

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
	// ponytail: streamed responses are cut into PayloadPart events of at most this size.
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
	host := strings.ToLower(r.Host)
	if sigV4ServiceHint(r) != "sagemaker" && !strings.HasPrefix(host, "runtime.sagemaker.") && !strings.HasPrefix(host, "runtime-fips.sagemaker.") {
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
	// Bound the body before SigV4 validation reads it (readBodyBytes only caps
	// at 25 MB); MaxBytesReader also bounds chunked bodies without a length.
	limit, limitMsg := int64(sagemakerRuntimeMaxBody), "Request body exceeds the 6 MB invocation payload limit"
	if op == "InvokeEndpointAsync" {
		limit, limitMsg = sagemakerRuntimeMaxAsyncBody, "Request body exceeds the 128000 byte inline payload limit; use InputLocation"
	}
	if r.ContentLength > limit {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", limitMsg)
		return true
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, limit+1)
	}
	body, err := readBodyBytes(r)
	if err != nil || int64(len(body)) > limit {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", limitMsg)
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "sagemaker"); !ok {
		respondSageMakerRuntimeError(w, status, code, msg)
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
	if op == "InvokeEndpointWithResponseStream" && r.Header.Get("X-Amzn-SageMaker-Session-Id") == "NEW_SESSION" {
		return "InvokeEndpointWithResponseStream cannot open a session; start one with InvokeEndpoint and SessionId NEW_SESSION"
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
	case "InvokeEndpointAsync":
		location := r.Header.Get("X-Amzn-SageMaker-InputLocation")
		if len(body) == 0 && location == "" {
			return "InputLocation is required when no inline Body is sent"
		}
		// ponytail: the pattern also allows https:// S3 URLs; add parsing for
		// virtual-hosted/path-style URLs if a client needs them. Checked even when
		// an inline Body (which takes precedence) is sent.
		if location != "" && !strings.HasPrefix(location, "s3://") {
			return "InputLocation must be an s3:// URI; https:// input locations are not supported by this emulator"
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
		return sagemakerRuntimeTarget{}, fmt.Sprintf("Endpoint %s of account %s not found.", name, sagemakerAccountID)
	}
	if endpoint.Status == "Creating" || endpoint.Status == "Updating" {
		// Endpoints finish deploying at once, as DescribeEndpoint already reports;
		// an invocation is not a modification, so LastModifiedTime stays put.
		endpoint.Status = "InService"
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
	// ponytail: untargeted calls always go to the first variant, ignoring
	// InitialVariantWeight; add weighted random routing if traffic splits matter.
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
	case async && target.OutputPath == "":
		return sagemakerRuntimeTarget{}, fmt.Sprintf("Endpoint %s has no AsyncInferenceConfig.OutputConfig.S3OutputPath to write results to.", name)
	}
	return target, ""
}

// sagemakerRuntimeSessionTTL is the lifetime advertised in the
// X-Amzn-SageMaker-New-Session-Id "Expires" attribute and enforced on use.
const sagemakerRuntimeSessionTTL = 20 * time.Minute

// sagemakerAccountID is the account the SageMaker emulation reports in ARNs.
const sagemakerAccountID = "123456789012"

type sagemakerRuntimeSession struct {
	Endpoint string
	Expires  time.Time
}

// runtimeSession opens a stateful session ("NEW_SESSION") or checks that an
// existing, unexpired one belongs to the endpoint. It returns the new session
// ID and its expiry when one was opened.
func (s *sagemakerStore) runtimeSession(endpoint, sessionID string) (string, time.Time, string) {
	if sessionID == "" {
		return "", time.Time{}, ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if sessionID == "NEW_SESSION" {
		for id, session := range s.runtimeSessions {
			if !now.Before(session.Expires) {
				delete(s.runtimeSessions, id)
			}
		}
		if s.runtimeSessions == nil {
			s.runtimeSessions = map[string]sagemakerRuntimeSession{}
		}
		id := strings.ReplaceAll(uuid.NewString(), "-", "")
		expires := now.Add(sagemakerRuntimeSessionTTL)
		s.runtimeSessions[id] = sagemakerRuntimeSession{Endpoint: endpoint, Expires: expires}
		return id, expires, ""
	}
	session, ok := s.runtimeSessions[sessionID]
	switch {
	case !ok || session.Endpoint != endpoint:
		return "", time.Time{}, fmt.Sprintf("Session %s not found for endpoint %s.", sessionID, endpoint)
	case !now.Before(session.Expires):
		delete(s.runtimeSessions, sessionID)
		return "", time.Time{}, fmt.Sprintf("Session %s for endpoint %s has expired.", sessionID, endpoint)
	}
	return "", time.Time{}, ""
}

// dropRuntimeSessionsLocked forgets the sessions of a deleted endpoint, so a
// recreated endpoint of the same name rejects the old session IDs.
func (s *sagemakerStore) dropRuntimeSessionsLocked(endpoint string) {
	for id, session := range s.runtimeSessions {
		if session.Endpoint == endpoint {
			delete(s.runtimeSessions, id)
		}
	}
}

var sagemakerS3URIRE = regexp.MustCompile(`^(https|s3)://([^/])/?(.*)$`)

// sagemakerAsyncInferenceConfigError checks CreateEndpointConfig's
// AsyncInferenceConfig, which async invocations depend on: OutputConfig is
// required, and its S3 paths must be S3 URIs of at most 512 characters.
func sagemakerAsyncInferenceConfigError(payload map[string]any) string {
	raw, present := payload["AsyncInferenceConfig"]
	if !present || raw == nil {
		return ""
	}
	config, ok := raw.(map[string]any)
	if !ok {
		return "AsyncInferenceConfig must be an object"
	}
	output, ok := config["OutputConfig"].(map[string]any)
	if !ok {
		return "1 validation error detected: Value null at 'asyncInferenceConfig.outputConfig' failed to satisfy constraint: Member must not be null"
	}
	for _, key := range []string{"S3OutputPath", "S3FailurePath"} {
		value, present := output[key]
		if !present {
			continue
		}
		path, ok := value.(string)
		if !ok || len(path) > 512 || !sagemakerS3URIRE.MatchString(path) {
			return fmt.Sprintf("AsyncInferenceConfig.OutputConfig.%s must be an S3 URI of at most 512 characters", key)
		}
	}
	return ""
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
	newSession, expires, msg := s.sagemaker.runtimeSession(name, r.Header.Get("X-Amzn-SageMaker-Session-Id"))
	if msg != "" {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", msg)
		return
	}
	h := w.Header()
	setSageMakerRuntimeContentType(w, sagemakerRuntimeResponseType(r.Header.Get("Accept"), r.Header.Get("Content-Type")))
	h.Set("x-Amzn-Invoked-Production-Variant", target.Variant)
	if attrs := r.Header.Get("X-Amzn-SageMaker-Custom-Attributes"); attrs != "" {
		h.Set("X-Amzn-SageMaker-Custom-Attributes", attrs)
	}
	if newSession != "" {
		h.Set("X-Amzn-SageMaker-New-Session-Id", newSession+"; Expires="+expires.Format(time.RFC3339))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) sagemakerRuntimeInvokeStream(w http.ResponseWriter, r *http.Request, name string, target sagemakerRuntimeTarget, body []byte) {
	if _, _, msg := s.sagemaker.runtimeSession(name, r.Header.Get("X-Amzn-SageMaker-Session-Id")); msg != "" {
		respondSageMakerRuntimeError(w, http.StatusBadRequest, "ValidationError", msg)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "application/vnd.amazon.eventstream")
	h.Set("X-Amzn-SageMaker-Content-Type", sagemakerRuntimeResponseType(r.Header.Get("X-Amzn-SageMaker-Accept"), r.Header.Get("Content-Type")))
	h.Set("x-Amzn-Invoked-Production-Variant", target.Variant)
	if attrs := r.Header.Get("X-Amzn-SageMaker-Custom-Attributes"); attrs != "" {
		h.Set("X-Amzn-SageMaker-Custom-Attributes", attrs)
	}
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	enc := eventstream.NewEncoder()
	for _, part := range sagemakerRuntimeChunks(body, sagemakerRuntimeStreamChunk) {
		msg := eventstream.Message{Payload: part}
		msg.Headers.Set(":message-type", eventstream.StringValue("event"))
		msg.Headers.Set(":event-type", eventstream.StringValue("PayloadPart"))
		msg.Headers.Set(":content-type", eventstream.StringValue("application/octet-stream"))
		// ponytail: the status is already sent, so a failed write just ends the stream.
		if err := enc.Encode(w, msg); err != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// sagemakerRuntimeChunks cuts body into parts of at most size bytes, ending a
// part after its last newline when there is one and never inside a UTF-8
// rune. An empty body still gives one empty part.
func sagemakerRuntimeChunks(body []byte, size int) [][]byte {
	if len(body) == 0 {
		return [][]byte{{}}
	}
	var parts [][]byte
	for len(body) > 0 {
		n := min(size, len(body))
		if n < len(body) {
			if nl := bytes.LastIndexByte(body[:n], '\n'); nl >= 0 {
				n = nl + 1
			} else {
				cut := n
				for cut > 0 && !utf8.RuneStart(body[cut]) {
					cut--
				}
				if cut > 0 {
					n = cut
				}
			}
		}
		parts = append(parts, body[:n])
		body = body[n:]
	}
	return parts
}

// sagemakerRuntimeResponseType picks the echoed response's content type: a
// concrete Accept wins, while a wildcard or a list falls back to the request's
// own Content-Type.
func sagemakerRuntimeResponseType(accept, contentType string) string {
	if strings.ContainsAny(accept, "*,") {
		accept = ""
	}
	return firstNonEmptyString(accept, contentType, "application/octet-stream")
}

// sagemakerRuntimeInvokeAsync "runs" the inference at once: the echoed input
// is written to the endpoint's S3OutputPath in the emulator's S3, or an error
// object to S3FailurePath when the input can't be read or written.
func (s *Server) sagemakerRuntimeInvokeAsync(w http.ResponseWriter, r *http.Request, target sagemakerRuntimeTarget, body []byte) {
	id := firstNonEmptyString(r.Header.Get("X-Amzn-SageMaker-Inference-Id"), uuid.NewString())
	ext := strings.Trim(r.Header.Get("X-Amzn-SageMaker-S3OutputPathExtension"), "/")
	// InferenceId may hold any printable character, so it is escaped into the
	// key; Filename and the extension are already limited to key-safe characters.
	key := url.PathEscape(id)
	output := sagemakerRuntimeJoinS3(target.OutputPath, ext, firstNonEmptyString(r.Header.Get("X-Amzn-SageMaker-Filename"), key+".out"))
	failure := ""
	if target.FailurePath != "" {
		failure = sagemakerRuntimeJoinS3(target.FailurePath, ext, key+"-error.out")
	}

	input, readErr := body, ""
	if len(input) == 0 {
		input, readErr = s.sagemakerRuntimeReadS3(r.Header.Get("X-Amzn-SageMaker-InputLocation"))
	}
	contentType := sagemakerRuntimeResponseType(r.Header.Get("X-Amzn-SageMaker-Accept"), r.Header.Get("X-Amzn-SageMaker-Content-Type"))
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

// setSageMakerRuntimeContentType sets the content type of an echoed inference
// body. The body is caller bytes, so only known non-HTML inference media types
// are echoed (each as a constant); anything else, including text/html, is served
// as application/octet-stream, always with nosniff.
func setSageMakerRuntimeContentType(w http.ResponseWriter, contentType string) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])) {
	case "application/json":
		h.Set("Content-Type", "application/json")
	case "application/jsonlines":
		h.Set("Content-Type", "application/jsonlines")
	case "text/csv":
		h.Set("Content-Type", "text/csv")
	case "text/plain":
		h.Set("Content-Type", "text/plain")
	case "application/x-npy":
		h.Set("Content-Type", "application/x-npy")
	case "application/x-recordio-protobuf":
		h.Set("Content-Type", "application/x-recordio-protobuf")
	case "application/x-image":
		h.Set("Content-Type", "application/x-image")
	default:
		h.Set("Content-Type", "application/octet-stream")
	}
}
