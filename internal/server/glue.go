package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// AWS Glue: JSON 1.1, X-Amz-Target "AWSGlue.<Operation>", SigV4 service "glue".

const (
	glueTargetPrefix = "AWSGlue."
	// glueMaxBody bounds a request body (Glue's largest requests, batch
	// partition writes, stay well under this).
	glueMaxBody = 8 << 20
)

type glueHandler func(st *glueStore, in glueRecord) (glueRecord, *glueError)

// glueHandlers holds operations with behaviour beyond the generic verbs; each
// glue_*.go file registers its own in init.
var glueHandlers = map[string]glueHandler{}

func registerGlueHandlers(handlers map[string]glueHandler) {
	for name, h := range handlers {
		glueHandlers[name] = h
	}
}

func (s *Server) handleGlueJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isGlueJSONCandidate(r) {
		return false
	}
	op := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), glueTargetPrefix)
	r.Body = http.MaxBytesReader(w, r.Body, glueMaxBody)
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "glue"); !ok {
		respondGlueError(w, status, code, msg)
		return true
	}
	model, known := awsmodels.GlueModel().Operations[op]
	if _, listed := glueOperationByName[op]; !known || !listed {
		respondGlueError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+op)
		return true
	}
	in, err := decodeGlueBody(r)
	if err != nil {
		respondGlueError(w, http.StatusBadRequest, "SerializationException", err.Error())
		return true
	}
	if msg := glueValidate(model.Input, in, ""); msg != "" {
		respondGlueError(w, http.StatusBadRequest, "InvalidInputException", msg)
		return true
	}
	out, gerr := s.glue.handle(op, model.Output, in)
	if gerr != nil {
		respondGlueError(w, http.StatusBadRequest, gerr.code, gerr.msg)
		return true
	}
	respondGlueJSON(w, out)
	return true
}

func isGlueJSONCandidate(r *http.Request) bool {
	return r.Method == http.MethodPost && strings.HasPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), glueTargetPrefix)
}

func decodeGlueBody(r *http.Request) (glueRecord, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return nil, errors.New("request body is too large")
		}
		return nil, errors.New("unable to read request body")
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return glueRecord{}, nil
	}
	var in glueRecord
	dec := json.NewDecoder(strings.NewReader(string(body)))
	if err := dec.Decode(&in); err != nil || in == nil {
		return nil, errors.New("request body must be a JSON object")
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("request body must be a single JSON object")
	}
	return in, nil
}

func respondGlueJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body)
}

func respondGlueError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"__type": code, "Message": msg})
}
