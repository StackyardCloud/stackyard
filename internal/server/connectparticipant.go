package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Connect Participant Service: REST JSON. The CLI signs it for
// execute-api, but the service authenticates with the participant or
// connection token in X-Amz-Bearer, not SigV4. Requests are claimed only on
// the participant paths when signed for execute-api, sent to a
// participant.connect host, or unsigned with a bearer token, so API Gateway
// Management (also execute-api) keeps /@connections and S3 keeps unsigned
// /participant/... object paths.

const cnpMaxBody = 1 << 20

var cnpPaths = map[string]string{
	"/participant/cancel-authentication":      "CancelParticipantAuthentication",
	"/participant/complete-attachment-upload": "CompleteAttachmentUpload",
	"/participant/connection":                 "CreateParticipantConnection",
	"/participant/disconnect":                 "DisconnectParticipant",
	"/participant/attachment":                 "GetAttachment",
	"/participant/authentication-url":         "GetAuthenticationUrl",
	"/participant/transcript":                 "GetTranscript",
	"/participant/event":                      "SendEvent",
	"/participant/message":                    "SendMessage",
	"/participant/start-attachment-upload":    "StartAttachmentUpload",
}

// cnpRequest is one decoded participant call.
type cnpRequest struct {
	in     map[string]any
	bearer string
	base   string // scheme://host of this server, for attachment URLs
}

type cnpHandler func(st *cnpStore, req cnpRequest) (map[string]any, *cnpErr)

var cnpHandlers = map[string]cnpHandler{
	"CancelParticipantAuthentication": (*cnpStore).cancelAuthentication,
	"CompleteAttachmentUpload":        (*cnpStore).completeAttachmentUpload,
	"CreateParticipantConnection":     (*cnpStore).createConnection,
	"DescribeView":                    (*cnpStore).describeView,
	"DisconnectParticipant":           (*cnpStore).disconnect,
	"GetAttachment":                   (*cnpStore).getAttachment,
	"GetAuthenticationUrl":            (*cnpStore).getAuthenticationURL,
	"GetTranscript":                   (*cnpStore).getTranscript,
	"SendEvent":                       (*cnpStore).sendEvent,
	"SendMessage":                     (*cnpStore).sendMessage,
	"StartAttachmentUpload":           (*cnpStore).startAttachmentUpload,
}

// cnpRoute maps a method and path to an operation (and DescribeView's token).
func cnpRoute(method, path string) (string, string) {
	if method == http.MethodPost {
		return cnpPaths[path], ""
	}
	if method == http.MethodGet {
		if token, ok := strings.CutPrefix(path, "/participant/views/"); ok && token != "" && !strings.Contains(token, "/") {
			return "DescribeView", token
		}
	}
	return "", ""
}

func cnpCandidate(r *http.Request) bool {
	switch sigV4ServiceHint(r) {
	case "execute-api":
		return true
	case "":
		return r.Header.Get("X-Amz-Bearer") != "" || strings.HasPrefix(strings.ToLower(r.Host), "participant.connect.")
	}
	return false
}

func (s *Server) handleConnectParticipantRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	op, viewToken := cnpRoute(r.Method, rawRequestPath(r))
	if op == "" || !cnpCandidate(r) {
		return false
	}
	// A declared length bounds what is read; refuse chunked bodies.
	if r.ContentLength < 0 || r.ContentLength > cnpMaxBody {
		respondCnpError(w, cnpValidation("request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	model := awsmodels.ConnectParticipantModel()
	raw, err := io.ReadAll(io.LimitReader(r.Body, cnpMaxBody))
	in := map[string]any{}
	if err == nil && strings.TrimSpace(string(raw)) != "" {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondCnpError(w, cnpValidation("invalid JSON body"))
		return true
	}
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondCnpError(w, cnpValidation("%s", strings.Join(problems, "; ")))
		return true
	}
	conformed, _ := model.Conform(shapes.Input, in).(map[string]any)
	if conformed == nil {
		conformed = map[string]any{}
	}
	if viewToken != "" {
		conformed["ViewToken"] = viewToken
	}
	bearer := r.Header.Get("X-Amz-Bearer")
	if bearer == "" || len(bearer) > 1000 {
		respondCnpError(w, cnpValidation("the X-Amz-Bearer header must hold a token of 1 to 1000 characters"))
		return true
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	out, cerr := cnpHandlers[op](s.connectParticipant, cnpRequest{in: conformed, bearer: bearer, base: scheme + "://" + r.Host})
	if cerr != nil {
		respondCnpError(w, cerr)
		return true
	}
	if out == nil {
		out = map[string]any{}
	}
	// out is built fresh by every handler; nothing in it is shared with the store.
	body, _ := json.Marshal(model.Conform(shapes.Output, out))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondCnpError(w http.ResponseWriter, e *cnpErr) {
	body, _ := json.Marshal(map[string]string{"__type": e.code, "Message": e.msg})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", e.code)
	w.WriteHeader(e.status)
	_, _ = w.Write(body)
}

func cnpStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func cnpInt(in map[string]any, key string, def int) int {
	// The model has already range-checked integer members.
	if v, ok := in[key].(float64); ok {
		return int(v)
	}
	return def
}
