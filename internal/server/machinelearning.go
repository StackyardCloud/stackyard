package server

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Machine Learning: awsJson1_1, X-Amz-Target AmazonML_20141212.*,
// signed machinelearning. Input is validated and conformed against the slim
// CLI model by fcDecode (shared with Forecast); state lives in
// machinelearning_store.go and S3 reads/writes in machinelearning_data.go.

const amlTargetPrefix = "AmazonML_20141212."

// amlHostRe bounds what a Host header may contribute to a realtime endpoint URL.
var amlHostRe = regexp.MustCompile(`^[A-Za-z0-9.\-]{1,253}(:[0-9]{1,5})?$`)

func (s *Server) handleMachineLearningJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), amlTargetPrefix)
	if !ok {
		return false
	}
	in, ferr := fcDecode(s, w, r, "machinelearning", awsmodels.MachineLearningModel(), op)
	if ferr != nil {
		writeAMLJSON(w, http.StatusBadRequest, amlErrorBody(ferr), ferr.code)
		return true
	}
	if in == nil { // signature failure already written
		return true
	}
	out, ferr := s.amlRun(op, s.withS3Caller(in, r), amlEndpointBase(r))
	if ferr != nil {
		writeAMLJSON(w, http.StatusBadRequest, amlErrorBody(ferr), ferr.code)
		return true
	}
	writeAMLJSON(w, http.StatusOK, out, "")
	return true
}

// amlEndpointBase is the URL realtime endpoints report: this server as the
// caller reached it, so Predict calls sent to EndpointUrl come back here.
func amlEndpointBase(r *http.Request) string {
	if !amlHostRe.MatchString(r.Host) {
		return "https://realtime.machinelearning." + fcRegion + ".amazonaws.com"
	}
	if r.TLS != nil {
		return "https://" + r.Host
	}
	return "http://" + r.Host
}

func amlErrorBody(e *fcErr) map[string]any {
	body := map[string]any{"__type": e.code, "message": e.msg}
	// Amazon ML errors carry a numeric code next to the message.
	switch e.code {
	case "InvalidInputException", "ResourceNotFoundException", "PredictorNotMountedException",
		"IdempotentParameterMismatchException", "InvalidTagException", "TagLimitExceededException":
		body["code"] = 400
	}
	return body
}

func writeAMLJSON(w http.ResponseWriter, status int, v any, errorType string) {
	body, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	if errorType != "" {
		w.Header().Set("X-Amzn-ErrorType", errorType)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
