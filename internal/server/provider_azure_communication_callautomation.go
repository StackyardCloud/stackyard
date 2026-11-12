package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureCommunicationCallAutomationAPIVersion = "2026-03-12"
const azureCommunicationCallAutomationPrefix = "/azure/communication-callautomation/2026-03-12"

type azureCommunicationCallAutomationCall struct {
	CallConnectionID     string
	ServerCallID         string
	Source               map[string]any
	Targets              []any
	IncomingCallContext  string
	CallbackURI          string
	OperationCallbackURI string
	OperationContext     string
	AnsweredBy           map[string]any
	CallLocator          map[string]any
	State                string
	IsMuted              bool
	IsOnHold             bool
	MediaStreaming       bool
	Transcription        bool
	Participants         map[string]*azureCommunicationCallAutomationParticipant
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type azureCommunicationCallAutomationParticipant struct {
	RawID      string
	Identifier map[string]any
	IsMuted    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type azureCommunicationCallAutomationRecording struct {
	RecordingID      string
	ServerCallID     string
	CallConnectionID string
	Kind             string
	State            string
	Payload          map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s *Server) handleAzureCommunicationCallAutomationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationCallAutomationPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationCallAutomationAPIVersion {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2026-03-12")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationCallAutomationPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 || !strings.EqualFold(segments[0], "calling") {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	tail := segments[1:]
	if len(tail) == 0 {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "callconnections:answer":
		if len(tail) != 1 || r.Method != http.MethodPost {
			if len(tail) == 1 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return true
		}
		s.handleAzureCommunicationCallAutomationAnswerCall(w, r)
		return true
	case "callconnections":
		s.handleAzureCommunicationCallAutomationCallConnections(w, r, tail[1:])
		return true
	case "callconnections:connect":
		if len(tail) != 1 || r.Method != http.MethodPost {
			if len(tail) == 1 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return true
		}
		s.handleAzureCommunicationCallAutomationConnect(w, r)
		return true
	case "callconnections:redirect":
		if len(tail) != 1 || r.Method != http.MethodPost {
			if len(tail) == 1 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return true
		}
		s.handleAzureCommunicationCallAutomationRedirect(w, r)
		return true
	case "callconnections:reject":
		if len(tail) != 1 || r.Method != http.MethodPost {
			if len(tail) == 1 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return true
		}
		s.handleAzureCommunicationCallAutomationReject(w, r)
		return true
	case "recordings":
		s.handleAzureCommunicationCallAutomationRecordings(w, r, tail[1:])
		return true
	default:
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureCommunicationCallAutomationCallConnections(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationCallAutomationCreateCall(w, r)
		return
	}

	callSegment := strings.TrimSpace(tail[0])
	if callSegment == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "callConnectionId is required")
		return
	}
	if callID, action, ok := azureCommunicationCallAutomationSplitAction(callSegment); ok {
		if len(tail) != 1 {
			azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		s.handleAzureCommunicationCallAutomationCallAction(w, r, callID, action)
		return
	}

	callID := callSegment
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationCallAutomationGetCall(w, callID)
		case http.MethodDelete:
			s.handleAzureCommunicationCallAutomationDeleteCall(w, callID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "participants":
		s.handleAzureCommunicationCallAutomationParticipants(w, r, callID, tail[2:])
	case "participants:add":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureCommunicationCallAutomationAddParticipant(w, r, callID)
	case "participants:canceladdparticipant":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureCommunicationCallAutomationCancelAddParticipant(w, r, callID)
	case "participants:remove":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureCommunicationCallAutomationRemoveParticipant(w, r, callID)
	case "participants:moveparticipants":
		if len(tail) != 2 || r.Method != http.MethodPost {
			if len(tail) == 2 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureCommunicationCallAutomationMoveParticipants(w, r, callID)
	default:
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationCallAutomationParticipants(w http.ResponseWriter, r *http.Request, callID string, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationCallAutomationListParticipants(w, callID)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureCommunicationCallAutomationGetParticipant(w, callID, tail[0])
		return
	}
	if len(tail) == 1 {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureCommunicationCallAutomationRecordings(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationCallAutomationStartRecording(w, r)
		return
	}

	recordingSegment := strings.TrimSpace(tail[0])
	if recordingSegment == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "recordingId is required")
		return
	}
	if recordingID, action, ok := azureCommunicationCallAutomationSplitAction(recordingSegment); ok {
		if len(tail) != 1 {
			azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		s.handleAzureCommunicationCallAutomationRecordingAction(w, r, recordingID, action)
		return
	}

	if len(tail) != 1 {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureCommunicationCallAutomationGetRecording(w, recordingSegment)
	case http.MethodDelete:
		s.handleAzureCommunicationCallAutomationDeleteRecording(w, recordingSegment)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCommunicationCallAutomationAnswerCall(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}
	incomingCallContext := strings.TrimSpace(azureCommunicationCallAutomationString(payload["incomingCallContext"]))
	callbackURI := strings.TrimSpace(azureCommunicationCallAutomationString(payload["callbackUri"]))
	if incomingCallContext == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "incomingCallContext is required")
		return
	}
	if callbackURI == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "callbackUri is required")
		return
	}

	now := time.Now().UTC()
	callID, serverCallID := azureCommunicationCallAutomationBootstrapIDs("answer", body, r, now)
	answeredBy, _ := payload["answeredBy"].(map[string]any)

	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	if call == nil {
		call = &azureCommunicationCallAutomationCall{
			CallConnectionID:     callID,
			ServerCallID:         serverCallID,
			IncomingCallContext:  incomingCallContext,
			CallbackURI:          callbackURI,
			OperationCallbackURI: strings.TrimSpace(azureCommunicationCallAutomationString(payload["operationCallbackUri"])),
			OperationContext:     strings.TrimSpace(azureCommunicationCallAutomationString(payload["operationContext"])),
			AnsweredBy:           azureCommunicationCallAutomationCloneObject(answeredBy),
			State:                "connected",
			Participants:         map[string]*azureCommunicationCallAutomationParticipant{},
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		s.azureCommunicationCallAutomationCalls[callID] = call
	}
	call.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureCommunicationCallAutomationCallFixture(call))
}

func (s *Server) handleAzureCommunicationCallAutomationCreateCall(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}
	callbackURI := strings.TrimSpace(azureCommunicationCallAutomationString(payload["callbackUri"]))
	targets := azureCommunicationCallAutomationArray(payload["targets"])
	if callbackURI == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "callbackUri is required")
		return
	}
	if len(targets) == 0 {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "targets must contain at least one participant")
		return
	}

	now := time.Now().UTC()
	callID, serverCallID := azureCommunicationCallAutomationBootstrapIDs("create", body, r, now)
	source, _ := payload["source"].(map[string]any)

	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	if call == nil {
		call = &azureCommunicationCallAutomationCall{
			CallConnectionID:     callID,
			ServerCallID:         serverCallID,
			Source:               azureCommunicationCallAutomationCloneObject(source),
			Targets:              azureCommunicationCallAutomationCloneArray(targets),
			CallbackURI:          callbackURI,
			OperationCallbackURI: strings.TrimSpace(azureCommunicationCallAutomationString(payload["operationCallbackUri"])),
			OperationContext:     strings.TrimSpace(azureCommunicationCallAutomationString(payload["operationContext"])),
			State:                "connected",
			Participants:         map[string]*azureCommunicationCallAutomationParticipant{},
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		for _, target := range targets {
			rawID, identifier := azureCommunicationCallAutomationNormalizeIdentifier(target)
			call.Participants[rawID] = &azureCommunicationCallAutomationParticipant{
				RawID:      rawID,
				Identifier: identifier,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
		}
		s.azureCommunicationCallAutomationCalls[callID] = call
	}
	call.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureCommunicationCallAutomationCallFixture(call))
}

func (s *Server) handleAzureCommunicationCallAutomationConnect(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}
	callLocator, _ := payload["callLocator"].(map[string]any)
	callbackURI := strings.TrimSpace(azureCommunicationCallAutomationString(payload["callbackUri"]))
	if len(callLocator) == 0 {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "callLocator is required")
		return
	}
	if callbackURI == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "callbackUri is required")
		return
	}

	now := time.Now().UTC()
	callID, serverCallID := azureCommunicationCallAutomationBootstrapIDs("connect", body, r, now)
	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	if call == nil {
		call = &azureCommunicationCallAutomationCall{
			CallConnectionID:     callID,
			ServerCallID:         serverCallID,
			CallLocator:          azureCommunicationCallAutomationCloneObject(callLocator),
			CallbackURI:          callbackURI,
			OperationCallbackURI: strings.TrimSpace(azureCommunicationCallAutomationString(payload["operationCallbackUri"])),
			OperationContext:     strings.TrimSpace(azureCommunicationCallAutomationString(payload["operationContext"])),
			State:                "connected",
			Participants:         map[string]*azureCommunicationCallAutomationParticipant{},
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		s.azureCommunicationCallAutomationCalls[callID] = call
	}
	call.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureCommunicationCallAutomationCallFixture(call))
}

func (s *Server) handleAzureCommunicationCallAutomationRedirect(w http.ResponseWriter, r *http.Request) {
	payload, _, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}
	if strings.TrimSpace(azureCommunicationCallAutomationString(payload["incomingCallContext"])) == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "incomingCallContext is required")
		return
	}
	if payload["targetParticipant"] == nil {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "targetParticipant is required")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationCallAutomationReject(w http.ResponseWriter, r *http.Request) {
	payload, _, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}
	if strings.TrimSpace(azureCommunicationCallAutomationString(payload["incomingCallContext"])) == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "incomingCallContext is required")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationCallAutomationGetCall(w http.ResponseWriter, callID string) {
	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	s.providerStorageMu.Unlock()
	if call == nil {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationCallAutomationCallFixture(call))
}

func (s *Server) handleAzureCommunicationCallAutomationDeleteCall(w http.ResponseWriter, callID string) {
	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	if call != nil {
		call.State = "disconnected"
		call.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationCallAutomationCallAction(w http.ResponseWriter, r *http.Request, callID, action string) {
	var requestPayload map[string]any
	if strings.EqualFold(action, "transferToParticipant") {
		payload, _, ok := azureCommunicationCallAutomationReadObject(w, r)
		if !ok {
			return
		}
		requestPayload = payload
	}

	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	if call == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	now := time.Now().UTC()
	call.UpdatedAt = now

	var status int
	var payload map[string]any
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "terminate":
		if r.Method != http.MethodPost {
			s.providerStorageMu.Unlock()
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		call.State = "disconnected"
		status = http.StatusAccepted
		payload = azureCommunicationCallAutomationActionFixture(callID, "terminate", now, r, nil)
	case "mute":
		if r.Method != http.MethodPost {
			s.providerStorageMu.Unlock()
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		call.IsMuted = true
		status = http.StatusAccepted
		payload = azureCommunicationCallAutomationActionFixture(callID, "mute", now, r, nil)
	case "hold":
		if r.Method != http.MethodPost {
			s.providerStorageMu.Unlock()
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		call.IsOnHold = true
		status = http.StatusAccepted
		payload = azureCommunicationCallAutomationActionFixture(callID, "hold", now, r, nil)
	case "unhold":
		if r.Method != http.MethodPost {
			s.providerStorageMu.Unlock()
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		call.IsOnHold = false
		status = http.StatusAccepted
		payload = azureCommunicationCallAutomationActionFixture(callID, "unhold", now, r, nil)
	case "transfertoparticipant":
		if r.Method != http.MethodPost {
			s.providerStorageMu.Unlock()
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		target := requestPayload["targetParticipant"]
		if target == nil {
			s.providerStorageMu.Unlock()
			azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "targetParticipant is required")
			return
		}
		status = http.StatusAccepted
		payload = azureCommunicationCallAutomationActionFixture(callID, "transferToParticipant", now, r, map[string]any{
			"targetParticipant": target,
		})
	case "cancelallmediaoperations", "play", "recognize", "senddtmftones", "startcontinuousdtmfrecognition", "stopcontinuousdtmfrecognition", "startmediastreaming", "stopmediastreaming", "starttranscription", "stoptranscription", "updatetranscription":
		if r.Method != http.MethodPost {
			s.providerStorageMu.Unlock()
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if strings.EqualFold(action, "startMediaStreaming") {
			call.MediaStreaming = true
		}
		if strings.EqualFold(action, "stopMediaStreaming") || strings.EqualFold(action, "cancelAllMediaOperations") {
			call.MediaStreaming = false
		}
		if strings.EqualFold(action, "startTranscription") {
			call.Transcription = true
		}
		if strings.EqualFold(action, "stopTranscription") || strings.EqualFold(action, "cancelAllMediaOperations") {
			call.Transcription = false
		}
		status = http.StatusAccepted
		payload = azureCommunicationCallAutomationActionFixture(callID, action, now, r, nil)
	default:
		s.providerStorageMu.Unlock()
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, status, payload)
}

func (s *Server) handleAzureCommunicationCallAutomationListParticipants(w http.ResponseWriter, callID string) {
	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	s.providerStorageMu.Unlock()
	if call == nil {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	value := make([]any, 0, len(call.Participants))
	participants := make([]*azureCommunicationCallAutomationParticipant, 0, len(call.Participants))
	for _, participant := range call.Participants {
		participants = append(participants, participant)
	}
	sort.Slice(participants, func(i, j int) bool { return participants[i].RawID < participants[j].RawID })
	for _, participant := range participants {
		value = append(value, azureCommunicationCallAutomationParticipantFixture(participant))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureCommunicationCallAutomationGetParticipant(w http.ResponseWriter, callID, participantRawID string) {
	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	var participant *azureCommunicationCallAutomationParticipant
	if call != nil {
		participant = call.Participants[participantRawID]
	}
	s.providerStorageMu.Unlock()
	if call == nil {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	if participant == nil {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "ParticipantNotFound", "participant not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationCallAutomationParticipantFixture(participant))
}

func (s *Server) handleAzureCommunicationCallAutomationAddParticipant(w http.ResponseWriter, r *http.Request, callID string) {
	payload, _, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}
	target := payload["participant"]
	if target == nil {
		target = payload["participantToAdd"]
	}
	if target == nil {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "participant is required")
		return
	}
	rawID, identifier := azureCommunicationCallAutomationNormalizeIdentifier(target)

	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	if call == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	now := time.Now().UTC()
	call.Participants[rawID] = &azureCommunicationCallAutomationParticipant{
		RawID:      rawID,
		Identifier: identifier,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	call.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusAccepted, azureCommunicationCallAutomationActionFixture(callID, "addParticipant", now, r, map[string]any{
		"participant": azureCommunicationCallAutomationParticipantFixture(&azureCommunicationCallAutomationParticipant{
			RawID:      rawID,
			Identifier: identifier,
			CreatedAt:  now,
			UpdatedAt:  now,
		}),
	}))
}

func (s *Server) handleAzureCommunicationCallAutomationCancelAddParticipant(w http.ResponseWriter, r *http.Request, callID string) {
	if _, _, ok := azureCommunicationCallAutomationReadObject(w, r); !ok {
		return
	}
	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	s.providerStorageMu.Unlock()
	if call == nil {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	respondJSON(w, http.StatusAccepted, azureCommunicationCallAutomationActionFixture(callID, "cancelAddParticipant", time.Now().UTC(), r, nil))
}

func (s *Server) handleAzureCommunicationCallAutomationRemoveParticipant(w http.ResponseWriter, r *http.Request, callID string) {
	payload, _, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}
	target := payload["participant"]
	if target == nil {
		target = payload["participantToRemove"]
	}
	if target == nil {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "participant is required")
		return
	}
	rawID, _ := azureCommunicationCallAutomationNormalizeIdentifier(target)

	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	if call == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	delete(call.Participants, rawID)
	call.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureCommunicationCallAutomationMoveParticipants(w http.ResponseWriter, r *http.Request, callID string) {
	if _, _, ok := azureCommunicationCallAutomationReadObject(w, r); !ok {
		return
	}
	s.providerStorageMu.Lock()
	call := s.azureCommunicationCallAutomationCalls[callID]
	s.providerStorageMu.Unlock()
	if call == nil {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "CallConnectionNotFound", "call connection not found")
		return
	}
	respondJSON(w, http.StatusAccepted, azureCommunicationCallAutomationActionFixture(callID, "moveParticipants", time.Now().UTC(), r, nil))
}

func (s *Server) handleAzureCommunicationCallAutomationStartRecording(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationCallAutomationReadObject(w, r)
	if !ok {
		return
	}

	callConnectionID := strings.TrimSpace(azureCommunicationCallAutomationString(payload["callConnectionId"]))
	serverCallID := strings.TrimSpace(azureCommunicationCallAutomationString(payload["serverCallId"]))
	if callConnectionID == "" && serverCallID == "" {
		if callLocator, ok := payload["callLocator"].(map[string]any); ok {
			serverCallID = strings.TrimSpace(azureCommunicationCallAutomationString(callLocator["serverCallId"]))
		}
	}
	if callConnectionID == "" && serverCallID == "" {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "callConnectionId, serverCallId, or callLocator.serverCallId is required")
		return
	}

	now := time.Now().UTC()
	recordingID := azureCommunicationCallAutomationDeterministicID("recording", "recording|"+string(body), r, now)
	s.providerStorageMu.Lock()
	recording := s.azureCommunicationCallAutomationRecordings[recordingID]
	if recording == nil {
		recording = &azureCommunicationCallAutomationRecording{
			RecordingID:      recordingID,
			ServerCallID:     serverCallID,
			CallConnectionID: callConnectionID,
			Kind:             strings.TrimSpace(azureCommunicationCallAutomationString(payload["recordingChannelType"])),
			State:            "active",
			Payload:          azureCommunicationCallAutomationCloneObject(payload),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.azureCommunicationCallAutomationRecordings[recordingID] = recording
	}
	recording.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureCommunicationCallAutomationRecordingFixture(recording))
}

func (s *Server) handleAzureCommunicationCallAutomationGetRecording(w http.ResponseWriter, recordingID string) {
	s.providerStorageMu.Lock()
	recording := s.azureCommunicationCallAutomationRecordings[recordingID]
	s.providerStorageMu.Unlock()
	if recording == nil {
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "RecordingNotFound", "recording not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationCallAutomationRecordingFixture(recording))
}

func (s *Server) handleAzureCommunicationCallAutomationDeleteRecording(w http.ResponseWriter, recordingID string) {
	s.providerStorageMu.Lock()
	delete(s.azureCommunicationCallAutomationRecordings, recordingID)
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationCallAutomationRecordingAction(w http.ResponseWriter, r *http.Request, recordingID, action string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.providerStorageMu.Lock()
	recording := s.azureCommunicationCallAutomationRecordings[recordingID]
	if recording == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "RecordingNotFound", "recording not found")
		return
	}
	recording.UpdatedAt = time.Now().UTC()
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "pause":
		recording.State = "inactive"
	case "resume":
		recording.State = "active"
	default:
		s.providerStorageMu.Unlock()
		azureCommunicationCallAutomationError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	fixture := azureCommunicationCallAutomationRecordingFixture(recording)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusAccepted, fixture)
}

func azureCommunicationCallAutomationReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationCallAutomationError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationCallAutomationActionFixture(callID, action string, now time.Time, r *http.Request, extra map[string]any) map[string]any {
	response := map[string]any{
		"callConnectionId": callID,
		"operationId":      azureCommunicationCallAutomationDeterministicID("operation", action, r, now),
		"status":           "running",
		"action":           action,
		"createdOn":        now.Format(time.RFC3339Nano),
	}
	if operationContext := strings.TrimSpace(r.Header.Get("Operation-Context")); operationContext != "" {
		response["operationContext"] = operationContext
	}
	for key, value := range extra {
		response[key] = value
	}
	return response
}

func azureCommunicationCallAutomationBootstrapIDs(label string, body []byte, r *http.Request, now time.Time) (string, string) {
	seed := label + "|" + string(body)
	callID := azureCommunicationCallAutomationDeterministicID("call", seed, r, now)
	serverCallID := azureCommunicationCallAutomationDeterministicID("server", seed, r, now)
	return callID, serverCallID
}

func azureCommunicationCallAutomationDeterministicID(prefix, seed string, r *http.Request, now time.Time) string {
	repeatabilityID := strings.TrimSpace(r.Header.Get("Repeatability-Request-ID"))
	repeatabilityFirstSent := strings.TrimSpace(r.Header.Get("Repeatability-First-Sent"))
	material := prefix + "|" + seed
	if repeatabilityID != "" && repeatabilityFirstSent != "" {
		material += "|" + repeatabilityID + "|" + repeatabilityFirstSent
	} else {
		material += "|" + now.Format(time.RFC3339Nano)
	}
	return prefix + "-" + azureCommunicationCallAutomationDigest(material)[:18]
}

func azureCommunicationCallAutomationSplitAction(segment string) (string, string, bool) {
	index := strings.Index(segment, ":")
	if index <= 0 || index >= len(segment)-1 {
		return "", "", false
	}
	return segment[:index], segment[index+1:], true
}

func azureCommunicationCallAutomationCallFixture(call *azureCommunicationCallAutomationCall) map[string]any {
	participants := make([]any, 0, len(call.Participants))
	items := make([]*azureCommunicationCallAutomationParticipant, 0, len(call.Participants))
	for _, participant := range call.Participants {
		items = append(items, participant)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RawID < items[j].RawID })
	for _, participant := range items {
		participants = append(participants, azureCommunicationCallAutomationParticipantFixture(participant))
	}
	return map[string]any{
		"callConnectionId":     call.CallConnectionID,
		"serverCallId":         call.ServerCallID,
		"callConnectionState":  call.State,
		"callbackUri":          call.CallbackURI,
		"operationCallbackUri": call.OperationCallbackURI,
		"operationContext":     call.OperationContext,
		"incomingCallContext":  call.IncomingCallContext,
		"source":               azureCommunicationCallAutomationCloneObject(call.Source),
		"targets":              azureCommunicationCallAutomationCloneArray(call.Targets),
		"answeredBy":           azureCommunicationCallAutomationCloneObject(call.AnsweredBy),
		"callLocator":          azureCommunicationCallAutomationCloneObject(call.CallLocator),
		"isMuted":              call.IsMuted,
		"isOnHold":             call.IsOnHold,
		"mediaStreaming": map[string]any{
			"isActive": call.MediaStreaming,
		},
		"transcription": map[string]any{
			"isActive": call.Transcription,
		},
		"participants": participants,
		"createdOn":    call.CreatedAt.Format(time.RFC3339Nano),
		"updatedOn":    call.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func azureCommunicationCallAutomationParticipantFixture(participant *azureCommunicationCallAutomationParticipant) map[string]any {
	return map[string]any{
		"participantId": participant.RawID,
		"identifier":    azureCommunicationCallAutomationCloneObject(participant.Identifier),
		"isMuted":       participant.IsMuted,
		"createdOn":     participant.CreatedAt.Format(time.RFC3339Nano),
		"updatedOn":     participant.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func azureCommunicationCallAutomationRecordingFixture(recording *azureCommunicationCallAutomationRecording) map[string]any {
	return map[string]any{
		"recordingId":      recording.RecordingID,
		"serverCallId":     recording.ServerCallID,
		"callConnectionId": recording.CallConnectionID,
		"recordingKind":    recording.Kind,
		"recordingState":   recording.State,
		"createdOn":        recording.CreatedAt.Format(time.RFC3339Nano),
		"updatedOn":        recording.UpdatedAt.Format(time.RFC3339Nano),
		"properties":       azureCommunicationCallAutomationCloneObject(recording.Payload),
	}
}

func azureCommunicationCallAutomationNormalizeIdentifier(value any) (string, map[string]any) {
	switch typed := value.(type) {
	case string:
		rawID := strings.TrimSpace(typed)
		if rawID == "" {
			rawID = "participant-" + azureCommunicationCallAutomationDigest("empty-string")[:12]
		}
		return rawID, map[string]any{"rawId": rawID, "kind": "raw"}
	case map[string]any:
		if rawID := strings.TrimSpace(azureCommunicationCallAutomationString(typed["rawId"])); rawID != "" {
			return rawID, azureCommunicationCallAutomationCloneObject(typed)
		}
		if id := strings.TrimSpace(azureCommunicationCallAutomationString(typed["id"])); id != "" {
			return id, azureCommunicationCallAutomationCloneObject(typed)
		}
		for _, key := range []string{"communicationUser", "phoneNumber", "microsoftTeamsUser", "microsoftTeamsApp", "unknownIdentifier"} {
			nested, _ := typed[key].(map[string]any)
			if len(nested) == 0 {
				continue
			}
			if id := strings.TrimSpace(azureCommunicationCallAutomationString(nested["id"])); id != "" {
				return id, azureCommunicationCallAutomationCloneObject(typed)
			}
			if phone := strings.TrimSpace(azureCommunicationCallAutomationString(nested["phoneNumber"])); phone != "" {
				return phone, azureCommunicationCallAutomationCloneObject(typed)
			}
			if rawID := strings.TrimSpace(azureCommunicationCallAutomationString(nested["rawId"])); rawID != "" {
				return rawID, azureCommunicationCallAutomationCloneObject(typed)
			}
		}
		rawID := "participant-" + azureCommunicationCallAutomationDigest(fmt.Sprintf("%v", typed))[:12]
		clone := azureCommunicationCallAutomationCloneObject(typed)
		clone["rawId"] = rawID
		return rawID, clone
	default:
		rawID := "participant-" + azureCommunicationCallAutomationDigest(fmt.Sprintf("%v", value))[:12]
		return rawID, map[string]any{"rawId": rawID, "kind": "generated"}
	}
}

func azureCommunicationCallAutomationCloneObject(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		out := make(map[string]any, len(in))
		for key, value := range in {
			out[key] = value
		}
		return out
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		out = make(map[string]any, len(in))
		for key, value := range in {
			out[key] = value
		}
	}
	return out
}

func azureCommunicationCallAutomationCloneArray(in []any) []any {
	if in == nil {
		return nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return append([]any(nil), in...)
	}
	var out []any
	if err := json.Unmarshal(raw, &out); err != nil {
		return append([]any(nil), in...)
	}
	return out
}

func azureCommunicationCallAutomationArray(value any) []any {
	items, _ := value.([]any)
	if items == nil {
		return []any{}
	}
	return items
}

func azureCommunicationCallAutomationString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func azureCommunicationCallAutomationDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationCallAutomationError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
