package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const azureCommunicationRoomsAPIVersion = "2025-03-13"
const azureCommunicationRoomsPrefix = "/azure/communication-rooms/2025-03-13"

type azureCommunicationRoom struct {
	RoomID           string
	ValidFrom        time.Time
	ValidUntil       time.Time
	PSTNDialOut      bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	RepeatabilityKey string
}

type azureCommunicationRoomParticipant struct {
	RoomID    string
	RawID     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Server) handleAzureCommunicationRoomsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationRoomsPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationRoomsAPIVersion {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2025-03-13")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationRoomsPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 || !strings.EqualFold(segments[0], "rooms") {
		azureCommunicationRoomsError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	s.handleAzureCommunicationRooms(w, r, segments[1:])
	return true
}

func (s *Server) handleAzureCommunicationRooms(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodPost:
			s.handleAzureCommunicationRoomsCreate(w, r)
		case http.MethodGet:
			s.handleAzureCommunicationRoomsList(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	roomID := strings.TrimSpace(tail[0])
	if roomID == "" {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "roomId is required")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationRoomsGet(w, roomID)
		case http.MethodPatch:
			s.handleAzureCommunicationRoomsUpdate(w, r, roomID)
		case http.MethodDelete:
			s.handleAzureCommunicationRoomsDelete(w, roomID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 2 && strings.EqualFold(tail[1], "participants") {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationRoomsListParticipants(w, r, roomID)
		case http.MethodPatch:
			s.handleAzureCommunicationRoomsPatchParticipants(w, r, roomID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	azureCommunicationRoomsError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureCommunicationRoomsCreate(w http.ResponseWriter, r *http.Request) {
	payload, body, ok := azureCommunicationRoomsReadObject(w, r)
	if !ok {
		return
	}

	validFrom, validUntil, pstnDialOutEnabled, participants, ok := azureCommunicationRoomsCreateInput(w, payload)
	if !ok {
		return
	}

	now := time.Now().UTC()
	repeatabilityKey := azureCommunicationRoomsRepeatabilityKey(r)

	s.providerStorageMu.Lock()
	if repeatabilityKey != "" {
		for _, room := range s.azureCommunicationRooms {
			if room.RepeatabilityKey == repeatabilityKey && room.DeletedAt == nil {
				fixture := azureCommunicationRoomsRoomFixture(room)
				s.providerStorageMu.Unlock()
				respondJSON(w, http.StatusCreated, fixture)
				return
			}
		}
	}

	roomID := azureCommunicationRoomsDeterministicID("room", string(body), r, now)
	room := &azureCommunicationRoom{
		RoomID:           roomID,
		ValidFrom:        validFrom,
		ValidUntil:       validUntil,
		PSTNDialOut:      pstnDialOutEnabled,
		CreatedAt:        now,
		UpdatedAt:        now,
		RepeatabilityKey: repeatabilityKey,
	}
	s.azureCommunicationRooms[roomID] = room
	for _, participant := range participants {
		key := azureCommunicationRoomsParticipantKey(roomID, participant.RawID)
		clone := participant
		clone.RoomID = roomID
		clone.CreatedAt = now
		clone.UpdatedAt = now
		s.azureCommunicationRoomParticipants[key] = &clone
	}
	fixture := azureCommunicationRoomsRoomFixture(room)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, fixture)
}

func (s *Server) handleAzureCommunicationRoomsGet(w http.ResponseWriter, roomID string) {
	s.providerStorageMu.Lock()
	room := s.azureCommunicationRooms[roomID]
	s.providerStorageMu.Unlock()
	if room == nil || room.DeletedAt != nil {
		azureCommunicationRoomsError(w, http.StatusNotFound, "RoomNotFound", "room not found")
		return
	}
	respondJSON(w, http.StatusOK, azureCommunicationRoomsRoomFixture(room))
}

func (s *Server) handleAzureCommunicationRoomsList(w http.ResponseWriter, r *http.Request) {
	maxPageSize, ok := azureCommunicationRoomsParseOptionalPositiveIntQuery(w, r, "maxPageSize")
	if !ok {
		return
	}
	skip, ok := azureCommunicationRoomsParseOptionalNonNegativeIntQuery(w, r, "skip")
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	rooms := make([]*azureCommunicationRoom, 0, len(s.azureCommunicationRooms))
	for _, room := range s.azureCommunicationRooms {
		if room == nil || room.DeletedAt != nil {
			continue
		}
		rooms = append(rooms, room)
	}
	s.providerStorageMu.Unlock()

	slicesSortRooms(rooms)
	if maxPageSize <= 0 {
		maxPageSize = len(rooms)
	}
	if skip > len(rooms) {
		skip = len(rooms)
	}
	end := skip + maxPageSize
	if end > len(rooms) {
		end = len(rooms)
	}

	values := make([]any, 0, end-skip)
	for _, room := range rooms[skip:end] {
		values = append(values, azureCommunicationRoomsRoomFixture(room))
	}
	payload := map[string]any{
		"value": values,
	}
	if end < len(rooms) {
		payload["nextLink"] = fmt.Sprintf("%s/rooms?api-version=%s&skip=%d&maxPageSize=%d", azureCommunicationRoomsPrefix, azureCommunicationRoomsAPIVersion, end, maxPageSize)
	}
	respondJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAzureCommunicationRoomsUpdate(w http.ResponseWriter, r *http.Request, roomID string) {
	payload, _, ok := azureCommunicationRoomsReadObject(w, r)
	if !ok {
		return
	}
	validFrom, validUntil, pstnDialOutEnabled, hasValidFrom, hasValidUntil, hasPSTN, ok := azureCommunicationRoomsUpdateInput(w, payload)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	room := s.azureCommunicationRooms[roomID]
	if room == nil || room.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationRoomsError(w, http.StatusNotFound, "RoomNotFound", "room not found")
		return
	}

	candidateValidFrom := room.ValidFrom
	candidateValidUntil := room.ValidUntil
	if hasValidFrom {
		candidateValidFrom = validFrom
	}
	if hasValidUntil {
		candidateValidUntil = validUntil
	}
	if candidateValidUntil.Before(candidateValidFrom) {
		s.providerStorageMu.Unlock()
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "validUntil must not be earlier than validFrom")
		return
	}

	if hasValidFrom {
		room.ValidFrom = validFrom
	}
	if hasValidUntil {
		room.ValidUntil = validUntil
	}
	if hasPSTN {
		room.PSTNDialOut = pstnDialOutEnabled
	}
	room.UpdatedAt = time.Now().UTC()
	fixture := azureCommunicationRoomsRoomFixture(room)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationRoomsDelete(w http.ResponseWriter, roomID string) {
	s.providerStorageMu.Lock()
	room := s.azureCommunicationRooms[roomID]
	if room == nil || room.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationRoomsError(w, http.StatusNotFound, "RoomNotFound", "room not found")
		return
	}
	now := time.Now().UTC()
	room.DeletedAt = &now
	delete(s.azureCommunicationRooms, roomID)
	for key, participant := range s.azureCommunicationRoomParticipants {
		if participant != nil && participant.RoomID == roomID {
			delete(s.azureCommunicationRoomParticipants, key)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationRoomsListParticipants(w http.ResponseWriter, r *http.Request, roomID string) {
	maxPageSize, ok := azureCommunicationRoomsParseOptionalPositiveIntQuery(w, r, "maxPageSize")
	if !ok {
		return
	}
	skip, ok := azureCommunicationRoomsParseOptionalNonNegativeIntQuery(w, r, "skip")
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	room := s.azureCommunicationRooms[roomID]
	if room == nil || room.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationRoomsError(w, http.StatusNotFound, "RoomNotFound", "room not found")
		return
	}
	participants := make([]*azureCommunicationRoomParticipant, 0)
	for _, participant := range s.azureCommunicationRoomParticipants {
		if participant != nil && participant.RoomID == roomID {
			participants = append(participants, participant)
		}
	}
	s.providerStorageMu.Unlock()

	slicesSortParticipants(participants)
	if maxPageSize <= 0 {
		maxPageSize = len(participants)
	}
	if skip > len(participants) {
		skip = len(participants)
	}
	end := skip + maxPageSize
	if end > len(participants) {
		end = len(participants)
	}

	values := make([]any, 0, end-skip)
	for _, participant := range participants[skip:end] {
		values = append(values, azureCommunicationRoomsParticipantFixture(participant))
	}
	payload := map[string]any{
		"value": values,
	}
	if end < len(participants) {
		payload["nextLink"] = fmt.Sprintf("%s/rooms/%s/participants?api-version=%s&skip=%d&maxPageSize=%d", azureCommunicationRoomsPrefix, roomID, azureCommunicationRoomsAPIVersion, end, maxPageSize)
	}
	respondJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAzureCommunicationRoomsPatchParticipants(w http.ResponseWriter, r *http.Request, roomID string) {
	payload, _, ok := azureCommunicationRoomsReadObject(w, r)
	if !ok {
		return
	}

	participantsValue, exists := payload["participants"]
	if !exists {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "participants is required")
		return
	}
	participantsMap, ok := participantsValue.(map[string]any)
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "participants must be an object")
		return
	}

	s.providerStorageMu.Lock()
	room := s.azureCommunicationRooms[roomID]
	if room == nil || room.DeletedAt != nil {
		s.providerStorageMu.Unlock()
		azureCommunicationRoomsError(w, http.StatusNotFound, "RoomNotFound", "room not found")
		return
	}

	now := time.Now().UTC()
	for rawID, value := range participantsMap {
		rawID = strings.TrimSpace(rawID)
		if rawID == "" {
			s.providerStorageMu.Unlock()
			azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "participant raw IDs must be non-empty")
			return
		}
		key := azureCommunicationRoomsParticipantKey(roomID, rawID)
		if value == nil {
			delete(s.azureCommunicationRoomParticipants, key)
			continue
		}
		object, ok := value.(map[string]any)
		if !ok {
			s.providerStorageMu.Unlock()
			azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "participant values must be objects or null")
			return
		}
		role := strings.TrimSpace(azureCommunicationRoomsString(object["role"]))
		if !azureCommunicationRoomsValidRole(role) {
			s.providerStorageMu.Unlock()
			azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "participant role must be Presenter, Attendee, Consumer, or Collaborator")
			return
		}
		participant := s.azureCommunicationRoomParticipants[key]
		if participant == nil {
			participant = &azureCommunicationRoomParticipant{
				RoomID:    roomID,
				RawID:     rawID,
				CreatedAt: now,
			}
			s.azureCommunicationRoomParticipants[key] = participant
		}
		participant.Role = role
		participant.UpdatedAt = now
	}
	room.UpdatedAt = now
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{})
}

func azureCommunicationRoomsCreateInput(w http.ResponseWriter, payload map[string]any) (time.Time, time.Time, bool, []azureCommunicationRoomParticipant, bool) {
	now := time.Now().UTC()
	validFrom, hasValidFrom, ok := azureCommunicationRoomsParseTimeField(payload, "validFrom")
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "validFrom must be a valid RFC3339 timestamp")
		return time.Time{}, time.Time{}, false, nil, false
	}
	validUntil, hasValidUntil, ok := azureCommunicationRoomsParseTimeField(payload, "validUntil")
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "validUntil must be a valid RFC3339 timestamp")
		return time.Time{}, time.Time{}, false, nil, false
	}
	pstnDialOutEnabled, hasPSTN, ok := azureCommunicationRoomsParseBoolField(payload, "pstnDialOutEnabled")
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "pstnDialOutEnabled must be a boolean")
		return time.Time{}, time.Time{}, false, nil, false
	}
	if !hasValidFrom {
		validFrom = now
	}
	if !hasValidUntil {
		validUntil = validFrom.Add(180 * 24 * time.Hour)
	}
	if validUntil.Before(validFrom) {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "validUntil must not be earlier than validFrom")
		return time.Time{}, time.Time{}, false, nil, false
	}
	if !hasPSTN {
		pstnDialOutEnabled = false
	}
	participants, ok := azureCommunicationRoomsParseParticipantArray(payload["participants"])
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "participants must be an array of participant objects with valid rawId and role")
		return time.Time{}, time.Time{}, false, nil, false
	}
	return validFrom, validUntil, pstnDialOutEnabled, participants, true
}

func azureCommunicationRoomsUpdateInput(w http.ResponseWriter, payload map[string]any) (time.Time, time.Time, bool, bool, bool, bool, bool) {
	validFrom, hasValidFrom, ok := azureCommunicationRoomsParseTimeField(payload, "validFrom")
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "validFrom must be a valid RFC3339 timestamp")
		return time.Time{}, time.Time{}, false, false, false, false, false
	}
	validUntil, hasValidUntil, ok := azureCommunicationRoomsParseTimeField(payload, "validUntil")
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "validUntil must be a valid RFC3339 timestamp")
		return time.Time{}, time.Time{}, false, false, false, false, false
	}
	pstnDialOutEnabled, hasPSTN, ok := azureCommunicationRoomsParseBoolField(payload, "pstnDialOutEnabled")
	if !ok {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "pstnDialOutEnabled must be a boolean")
		return time.Time{}, time.Time{}, false, false, false, false, false
	}
	return validFrom, validUntil, pstnDialOutEnabled, hasValidFrom, hasValidUntil, hasPSTN, true
}

func azureCommunicationRoomsParseParticipantArray(value any) ([]azureCommunicationRoomParticipant, bool) {
	if value == nil {
		return nil, true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]azureCommunicationRoomParticipant, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		rawID := strings.TrimSpace(azureCommunicationRoomsString(object["rawId"]))
		if rawID == "" {
			rawID = strings.TrimSpace(azureCommunicationRoomsString(object["id"]))
		}
		if rawID == "" {
			if identity, ok := object["communicationIdentifier"].(map[string]any); ok {
				rawID = azureCommunicationRoomsExtractRawID(identity)
			}
		}
		role := strings.TrimSpace(azureCommunicationRoomsString(object["role"]))
		if rawID == "" || !azureCommunicationRoomsValidRole(role) {
			return nil, false
		}
		out = append(out, azureCommunicationRoomParticipant{
			RawID: rawID,
			Role:  role,
		})
	}
	return out, true
}

func azureCommunicationRoomsExtractRawID(identity map[string]any) string {
	if identity == nil {
		return ""
	}
	if rawID := strings.TrimSpace(azureCommunicationRoomsString(identity["rawId"])); rawID != "" {
		return rawID
	}
	if rawID := strings.TrimSpace(azureCommunicationRoomsString(identity["id"])); rawID != "" {
		return rawID
	}
	for _, key := range []string{"communicationUser", "microsoftTeamsUser", "microsoftTeamsApp", "phoneNumber", "unknownIdentifier"} {
		nested, _ := identity[key].(map[string]any)
		if len(nested) == 0 {
			continue
		}
		if rawID := strings.TrimSpace(azureCommunicationRoomsString(nested["rawId"])); rawID != "" {
			return rawID
		}
		if id := strings.TrimSpace(azureCommunicationRoomsString(nested["id"])); id != "" {
			return id
		}
		if phoneNumber := strings.TrimSpace(azureCommunicationRoomsString(nested["phoneNumber"])); phoneNumber != "" {
			return phoneNumber
		}
	}
	return ""
}

func azureCommunicationRoomsParseOptionalPositiveIntQuery(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", key+" must be a positive integer")
		return 0, false
	}
	return value, true
}

func azureCommunicationRoomsParseOptionalNonNegativeIntQuery(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", key+" must be a non-negative integer")
		return 0, false
	}
	return value, true
}

func azureCommunicationRoomsParseTimeField(payload map[string]any, key string) (time.Time, bool, bool) {
	value, exists := payload[key]
	if !exists {
		return time.Time{}, false, true
	}
	text := strings.TrimSpace(azureCommunicationRoomsString(value))
	if text == "" {
		return time.Time{}, false, false
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Time{}, false, false
	}
	return parsed.UTC(), true, true
}

func azureCommunicationRoomsParseBoolField(payload map[string]any, key string) (bool, bool, bool) {
	value, exists := payload[key]
	if !exists {
		return false, false, true
	}
	typed, ok := value.(bool)
	if !ok {
		return false, false, false
	}
	return typed, true, true
}

func azureCommunicationRoomsRoomFixture(room *azureCommunicationRoom) map[string]any {
	return map[string]any{
		"id":                 room.RoomID,
		"createdAt":          room.CreatedAt.Format(time.RFC3339Nano),
		"validFrom":          room.ValidFrom.Format(time.RFC3339Nano),
		"validUntil":         room.ValidUntil.Format(time.RFC3339Nano),
		"pstnDialOutEnabled": room.PSTNDialOut,
	}
}

func azureCommunicationRoomsParticipantFixture(participant *azureCommunicationRoomParticipant) map[string]any {
	return map[string]any{
		"rawId": participant.RawID,
		"role":  participant.Role,
	}
}

func azureCommunicationRoomsParticipantKey(roomID, rawID string) string {
	return roomID + "|" + rawID
}

func azureCommunicationRoomsRepeatabilityKey(r *http.Request) string {
	requestID := strings.TrimSpace(r.Header.Get("Repeatability-Request-ID"))
	if requestID == "" {
		requestID = strings.TrimSpace(r.Header.Get("repeatability-request-id"))
	}
	firstSent := strings.TrimSpace(r.Header.Get("Repeatability-First-Sent"))
	if firstSent == "" {
		firstSent = strings.TrimSpace(r.Header.Get("repeatability-first-sent"))
	}
	if requestID == "" || firstSent == "" {
		return ""
	}
	return requestID + "|" + firstSent
}

func azureCommunicationRoomsDeterministicID(prefix, seed string, r *http.Request, now time.Time) string {
	material := prefix + "|" + seed
	if repeatabilityKey := azureCommunicationRoomsRepeatabilityKey(r); repeatabilityKey != "" {
		material += "|" + repeatabilityKey
	} else {
		material += "|" + now.Format(time.RFC3339Nano)
	}
	return prefix + "-" + azureCommunicationRoomsDigest(material)[:18]
}

func azureCommunicationRoomsReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationRoomsError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationRoomsString(value any) string {
	text, _ := value.(string)
	return text
}

func azureCommunicationRoomsValidRole(role string) bool {
	switch strings.TrimSpace(role) {
	case "Presenter", "Attendee", "Consumer", "Collaborator":
		return true
	default:
		return false
	}
}

func azureCommunicationRoomsDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationRoomsError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func slicesSortRooms(rooms []*azureCommunicationRoom) {
	if len(rooms) < 2 {
		return
	}
	for i := 0; i < len(rooms)-1; i++ {
		for j := i + 1; j < len(rooms); j++ {
			if rooms[j].CreatedAt.Before(rooms[i].CreatedAt) || (rooms[j].CreatedAt.Equal(rooms[i].CreatedAt) && rooms[j].RoomID < rooms[i].RoomID) {
				rooms[i], rooms[j] = rooms[j], rooms[i]
			}
		}
	}
}

func slicesSortParticipants(participants []*azureCommunicationRoomParticipant) {
	if len(participants) < 2 {
		return
	}
	for i := 0; i < len(participants)-1; i++ {
		for j := i + 1; j < len(participants); j++ {
			if participants[j].RawID < participants[i].RawID {
				participants[i], participants[j] = participants[j], participants[i]
			}
		}
	}
}
