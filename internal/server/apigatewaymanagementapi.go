package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// API Gateway Management API (signing name execute-api) is the callback API a
// WebSocket backend uses to talk to connected clients. Stackyard has no
// WebSocket runtime, so clients are registered through the
// /_stackyard/apigatewaymanagementapi control endpoints, and the messages a
// backend posts are kept per connection for a test to read back.

const (
	apiGatewayManagementMaxData     = 128 * 1024
	apiGatewayManagementMaxMessages = 100
)

type apiGatewayManagementMessage struct {
	Data       []byte    `json:"data"`
	ReceivedAt time.Time `json:"receivedAt"`
}

type apiGatewayManagementConnection struct {
	ID           string
	APIID        string
	Stage        string
	SourceIP     string
	UserAgent    string
	ConnectedAt  time.Time
	LastActiveAt time.Time
	Messages     []apiGatewayManagementMessage
}

type apiGatewayManagementStore struct {
	mu          sync.Mutex
	connections map[string]*apiGatewayManagementConnection
}

func newAPIGatewayManagementStore() *apiGatewayManagementStore {
	return &apiGatewayManagementStore{connections: map[string]*apiGatewayManagementConnection{}}
}

func (s *apiGatewayManagementStore) connect(apiID, stage, sourceIP, userAgent string) apiGatewayManagementConnection {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newAPIGatewayConnectionID()
	for s.connections[id] != nil {
		id = newAPIGatewayConnectionID()
	}
	now := time.Now().UTC()
	conn := &apiGatewayManagementConnection{
		ID: id, APIID: apiID, Stage: stage, SourceIP: sourceIP, UserAgent: userAgent,
		ConnectedAt: now, LastActiveAt: now,
	}
	s.connections[id] = conn
	return *conn
}

// lookupLocked returns the connection when it exists and belongs to the API
// and stage the request was addressed to. An empty scope matches anything.
func (s *apiGatewayManagementStore) lookupLocked(id, apiID, stage string) *apiGatewayManagementConnection {
	conn := s.connections[id]
	if conn == nil {
		return nil
	}
	if apiID != "" && conn.APIID != "" && apiID != conn.APIID {
		return nil
	}
	if stage != "" && conn.Stage != "" && stage != conn.Stage {
		return nil
	}
	return conn
}

func (s *apiGatewayManagementStore) get(id, apiID, stage string) (apiGatewayManagementConnection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn := s.lookupLocked(id, apiID, stage)
	if conn == nil {
		return apiGatewayManagementConnection{}, false
	}
	return *conn, true
}

func (s *apiGatewayManagementStore) post(id, apiID, stage string, data []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn := s.lookupLocked(id, apiID, stage)
	if conn == nil {
		return false
	}
	now := time.Now().UTC()
	conn.LastActiveAt = now
	// ponytail: keep only the newest messages so an undrained connection can't grow without bound.
	conn.Messages = append(conn.Messages, apiGatewayManagementMessage{Data: append([]byte(nil), data...), ReceivedAt: now})
	if extra := len(conn.Messages) - apiGatewayManagementMaxMessages; extra > 0 {
		conn.Messages = append([]apiGatewayManagementMessage(nil), conn.Messages[extra:]...)
	}
	return true
}

func (s *apiGatewayManagementStore) disconnect(id, apiID, stage string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lookupLocked(id, apiID, stage) == nil {
		return false
	}
	delete(s.connections, id)
	return true
}

func (s *apiGatewayManagementStore) messages(id string) ([]apiGatewayManagementMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn := s.connections[id]
	if conn == nil {
		return nil, false
	}
	return append([]apiGatewayManagementMessage{}, conn.Messages...), true
}

// newAPIGatewayConnectionID mimics the 16-character, "="-terminated IDs API
// Gateway hands out, using the URL-safe alphabet so an ID never contains "/".
func newAPIGatewayConnectionID() string {
	b := make([]byte, 11)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (s *Server) handleAPIGatewayManagementRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	stage, rawID, ok := parseAPIGatewayManagementPath(rawRequestPath(r))
	if !ok || !isAPIGatewayManagementCandidate(r, stage) {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "execute-api"); !ok {
		respondAPIGatewayV2Error(w, status, code, msg)
		return true
	}
	action, known := apiGatewayManagementOperationByMethod[r.Method]
	if !known {
		respondAPIGatewayV2Error(w, http.StatusNotImplemented, "NotImplementedException", "unsupported method "+r.Method+" for @connections")
		return true
	}
	id, err := url.PathUnescape(rawID)
	if err != nil || strings.TrimSpace(id) == "" {
		respondAPIGatewayV2Error(w, http.StatusBadRequest, "BadRequestException", "invalid connectionId")
		return true
	}
	apiID := apiGatewayManagementHostAPIID(r.Host)
	gone := func() {
		respondAPIGatewayV2Error(w, http.StatusGone, "GoneException", "connection "+id+" is gone")
	}

	switch action {
	case "GetConnection":
		conn, found := s.apigatewaymanagementapi.get(id, apiID, stage)
		if !found {
			gone()
			return true
		}
		respondAPIGatewayV2JSON(w, http.StatusOK, map[string]any{
			"connectedAt":  conn.ConnectedAt.Format(time.RFC3339Nano),
			"lastActiveAt": conn.LastActiveAt.Format(time.RFC3339Nano),
			"identity":     map[string]string{"sourceIp": conn.SourceIP, "userAgent": conn.UserAgent},
		})
	case "PostToConnection":
		data, err := readBodyBytes(r)
		if err != nil {
			respondAPIGatewayV2Error(w, http.StatusBadRequest, "BadRequestException", "invalid request body")
			return true
		}
		if len(data) > apiGatewayManagementMaxData {
			respondAPIGatewayV2Error(w, http.StatusRequestEntityTooLarge, "PayloadTooLargeException", "Message too long")
			return true
		}
		if !s.apigatewaymanagementapi.post(id, apiID, stage, data) {
			gone()
			return true
		}
		w.WriteHeader(http.StatusOK)
	case "DeleteConnection":
		if !s.apigatewaymanagementapi.disconnect(id, apiID, stage) {
			gone()
			return true
		}
		w.WriteHeader(http.StatusNoContent)
	}
	return true
}

// parseAPIGatewayManagementPath accepts /@connections/{id} and
// /{stage}/@connections/{id}, the latter being what an SDK sends when its
// endpoint is https://{api-id}.execute-api.{region}.amazonaws.com/{stage}.
func parseAPIGatewayManagementPath(rawPath string) (stage, id string, ok bool) {
	path := strings.SplitN(rawPath, "?", 2)[0]
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	switch {
	case len(parts) == 2 && parts[0] == "@connections":
		return "", parts[1], true
	case len(parts) == 3 && parts[1] == "@connections" && parts[0] != "":
		return parts[0], parts[2], true
	}
	return "", "", false
}

// isAPIGatewayManagementCandidate claims requests signed for execute-api. An
// unsigned bare /@connections path is also claimed so the caller gets a SigV4
// error, but a stage-prefixed path needs an execute-api signature or host,
// because /{bucket}/@connections/{key} is also a valid S3 object path.
func isAPIGatewayManagementCandidate(r *http.Request, stage string) bool {
	switch sigV4ServiceHint(r) {
	case "execute-api":
		return true
	case "":
		return stage == "" || strings.Contains(strings.ToLower(r.Host), ".execute-api.")
	}
	return false
}

func apiGatewayManagementHostAPIID(host string) string {
	host = strings.ToLower(host)
	if i := strings.Index(host, ".execute-api."); i > 0 {
		return host[:i]
	}
	return ""
}

type apiGatewayManagementConnectRequest struct {
	APIID     string `json:"apiId"`
	Stage     string `json:"stage"`
	SourceIP  string `json:"sourceIp"`
	UserAgent string `json:"userAgent"`
}

// handleAPIGatewayManagementConnect stands in for a WebSocket client
// connecting: it registers a connection and returns its ID.
func (s *Server) handleAPIGatewayManagementConnect(w http.ResponseWriter, r *http.Request) {
	var req apiGatewayManagementConnectRequest
	body, err := readBodyBytes(r)
	if err == nil && len(strings.TrimSpace(string(body))) > 0 {
		err = json.Unmarshal(body, &req)
	}
	if err != nil {
		respondAPIGatewayV2Error(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}
	req.APIID = strings.TrimSpace(req.APIID)
	if req.APIID != "" {
		if protocol, found := s.apigatewayv2.protocolType(req.APIID); !found || protocol != "WEBSOCKET" {
			respondAPIGatewayV2Error(w, http.StatusBadRequest, "BadRequestException", "apiId "+req.APIID+" is not a WEBSOCKET API")
			return
		}
	}
	if req.SourceIP == "" {
		req.SourceIP, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	if req.UserAgent == "" {
		req.UserAgent = r.UserAgent()
	}
	conn := s.apigatewaymanagementapi.connect(req.APIID, strings.TrimSpace(req.Stage), req.SourceIP, req.UserAgent)
	respondAPIGatewayV2JSON(w, http.StatusCreated, map[string]any{
		"connectionId": conn.ID,
		"apiId":        conn.APIID,
		"stage":        conn.Stage,
		"connectedAt":  conn.ConnectedAt.Format(time.RFC3339Nano),
	})
}

func (s *Server) handleAPIGatewayManagementMessages(w http.ResponseWriter, r *http.Request) {
	msgs, found := s.apigatewaymanagementapi.messages(r.PathValue("connectionId"))
	if !found {
		respondAPIGatewayV2Error(w, http.StatusNotFound, "NotFoundException", "unknown connection")
		return
	}
	respondAPIGatewayV2JSON(w, http.StatusOK, map[string]any{"messages": msgs})
}
