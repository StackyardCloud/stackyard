package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type ivsRealtimeStore struct {
	mu sync.Mutex

	nextStageID            int64
	nextStageSessionID     int64
	nextParticipantID      int64
	nextParticipantTokenID int64
	nextCompositionID      int64
	nextEncoderConfigID    int64
	nextIngestConfigID     int64
	nextStorageConfigID    int64
	nextPublicKeyID        int64

	stages                map[string]map[string]any
	stageSessions         map[string][]map[string]any
	participants          map[string]map[string]any
	participantTokens     map[string]map[string]any
	participantEvents     map[string][]map[string]any
	participantReplicas   map[string][]map[string]any
	compositions          map[string]map[string]any
	encoderConfigurations map[string]map[string]any
	ingestConfigurations  map[string]map[string]any
	storageConfigurations map[string]map[string]any
	publicKeys            map[string]map[string]any
	tags                  map[string]map[string]string
}

func newIVSRealtimeStore() *ivsRealtimeStore {
	s := &ivsRealtimeStore{
		nextStageID:            2,
		nextStageSessionID:     2,
		nextParticipantID:      2,
		nextParticipantTokenID: 2,
		nextCompositionID:      2,
		nextEncoderConfigID:    2,
		nextIngestConfigID:     2,
		nextStorageConfigID:    2,
		nextPublicKeyID:        2,
		stages:                 map[string]map[string]any{},
		stageSessions:          map[string][]map[string]any{},
		participants:           map[string]map[string]any{},
		participantTokens:      map[string]map[string]any{},
		participantEvents:      map[string][]map[string]any{},
		participantReplicas:    map[string][]map[string]any{},
		compositions:           map[string]map[string]any{},
		encoderConfigurations:  map[string]map[string]any{},
		ingestConfigurations:   map[string]map[string]any{},
		storageConfigurations:  map[string]map[string]any{},
		publicKeys:             map[string]map[string]any{},
		tags:                   map[string]map[string]string{},
	}

	stage := s.ensureStageLocked(ivsRealtimeArn("stage", "stage-00000001"))
	s.ensureStageSessionLocked(ivsRealtimeStringAny(stage, "arn"), "session-00000001")
	s.ensureParticipantLocked("participant-00000001", ivsRealtimeStringAny(stage, "arn"))
	s.ensureEncoderConfigurationLocked(ivsRealtimeArn("encoder-configuration", "encoder-00000001"))
	s.ensureIngestConfigurationLocked(ivsRealtimeArn("ingest-configuration", "ingest-00000001"))
	s.ensureStorageConfigurationLocked(ivsRealtimeArn("storage-configuration", "storage-00000001"))
	s.ensurePublicKeyLocked(ivsRealtimeArn("public-key", "pk-00000001"))
	s.tags[ivsRealtimeStringAny(stage, "arn")] = map[string]string{"seed": "true"}

	return s
}

func (s *ivsRealtimeStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	stageArn := ivsRealtimeFirstNonEmpty(
		ivsRealtimePath(pathParams, "resourceArn"),
		ivsRealtimeStringAny(payload, "stageArn"),
		s.firstStageArnLocked(),
		ivsRealtimeArn("stage", "stage-00000001"),
	)
	participantID := ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "participantId"), s.firstParticipantIDLocked(), "participant-00000001")
	compositionArn := ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "arn", "compositionArn"), s.firstCompositionArnLocked())
	encoderArn := ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "arn", "encoderConfigurationArn"), s.firstEncoderConfigurationArnLocked())
	ingestArn := ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "arn", "ingestConfigurationArn"), s.firstIngestConfigurationArnLocked())
	storageArn := ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "arn", "storageConfigurationArn"), s.firstStorageConfigurationArnLocked())
	publicKeyArn := ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "arn", "publicKeyArn"), s.firstPublicKeyArnLocked())
	resourceArn := ivsRealtimeFirstNonEmpty(ivsRealtimePath(pathParams, "resourceArn"), ivsRealtimeStringAny(payload, "resourceArn", "arn"), stageArn)

	sessionID := ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "sessionId", "stageSessionId"), "session-00000001")

	switch action {
	case "CreateStage":
		id := s.nextStageIDLocked()
		stageArn = ivsRealtimeArn("stage", fmt.Sprintf("stage-%08d", id))
		stage := s.ensureStageLocked(stageArn)
		for k, v := range payload {
			stage[k] = v
		}
		stage["arn"] = stageArn
		stage["name"] = ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "name"), fmt.Sprintf("stackyard-stage-%d", id))
		stage["activeSessionId"] = fmt.Sprintf("session-%08d", s.nextStageSessionID)
		stage["updatedAt"] = now
		s.ensureStageSessionLocked(stageArn, ivsRealtimeStringAny(stage, "activeSessionId"))
		return map[string]any{"stage": ivsRealtimeCloneMap(stage)}
	case "GetStage":
		return map[string]any{"stage": ivsRealtimeCloneMap(s.ensureStageLocked(stageArn))}
	case "ListStages":
		return map[string]any{"stages": s.listStageSummariesLocked(), "nextToken": ""}
	case "UpdateStage":
		stage := s.ensureStageLocked(stageArn)
		for k, v := range payload {
			stage[k] = v
		}
		stage["updatedAt"] = now
		return map[string]any{"stage": ivsRealtimeCloneMap(stage)}
	case "DeleteStage":
		delete(s.stages, stageArn)
		delete(s.stageSessions, stageArn)
		return map[string]any{}

	case "GetStageSession":
		return map[string]any{"stageSession": ivsRealtimeCloneMap(s.ensureStageSessionLocked(stageArn, sessionID))}
	case "ListStageSessions":
		sessions := s.ensureStageSessionsLocked(stageArn)
		out := make([]map[string]any, 0, len(sessions))
		for _, session := range sessions {
			out = append(out, ivsRealtimeCloneMap(session))
		}
		return map[string]any{"stageSessions": out, "nextToken": ""}

	case "CreateParticipantToken":
		tokenID := s.nextParticipantTokenIDLocked()
		participant := s.ensureParticipantLocked(participantID, stageArn)
		tokenValue := fmt.Sprintf("token-%08d", tokenID)
		token := map[string]any{
			"token":          tokenValue,
			"participantId":  ivsRealtimeStringAny(participant, "participantId"),
			"userId":         ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "userId"), fmt.Sprintf("user-%08d", tokenID)),
			"stageArn":       stageArn,
			"expirationTime": time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339),
		}
		s.participantTokens[tokenValue] = token
		return map[string]any{"participantToken": ivsRealtimeCloneMap(token)}
	case "GetParticipant":
		return map[string]any{"participant": ivsRealtimeCloneMap(s.ensureParticipantLocked(participantID, stageArn))}
	case "ListParticipants":
		participants := s.listParticipantsForStageLocked(stageArn)
		return map[string]any{"participants": participants, "nextToken": ""}
	case "DisconnectParticipant":
		participant := s.ensureParticipantLocked(participantID, stageArn)
		participant["state"] = "DISCONNECTED"
		participant["updatedAt"] = now
		s.appendParticipantEventLocked(participantID, "PARTICIPANT_DISCONNECTED", now)
		return map[string]any{}
	case "ListParticipantEvents":
		return map[string]any{"events": ivsRealtimeCloneSliceMap(s.ensureParticipantEventsLocked(participantID)), "nextToken": ""}
	case "StartParticipantReplication":
		replica := map[string]any{
			"participantId": participantID,
			"stageArn":      stageArn,
			"state":         "REPLICATING",
			"updatedAt":     now,
		}
		s.participantReplicas[participantID] = append(s.participantReplicas[participantID], replica)
		return map[string]any{"participantReplica": ivsRealtimeCloneMap(replica)}
	case "StopParticipantReplication":
		replicas := s.participantReplicas[participantID]
		for i := range replicas {
			replicas[i]["state"] = "STOPPED"
			replicas[i]["updatedAt"] = now
		}
		s.participantReplicas[participantID] = replicas
		return map[string]any{}
	case "ListParticipantReplicas":
		return map[string]any{"participantReplicas": ivsRealtimeCloneSliceMap(s.participantReplicas[participantID]), "nextToken": ""}

	case "StartComposition":
		id := s.nextCompositionIDLocked()
		compositionArn = ivsRealtimeArn("composition", fmt.Sprintf("composition-%08d", id))
		composition := map[string]any{
			"arn":       compositionArn,
			"stageArn":  stageArn,
			"state":     "ACTIVE",
			"startTime": now,
			"layout":    ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "layout"), "GRID"),
		}
		s.compositions[compositionArn] = composition
		return map[string]any{"composition": ivsRealtimeCloneMap(composition)}
	case "StopComposition":
		composition := s.ensureCompositionLocked(compositionArn, stageArn)
		composition["state"] = "STOPPED"
		composition["stopTime"] = now
		return map[string]any{}
	case "GetComposition":
		return map[string]any{"composition": ivsRealtimeCloneMap(s.ensureCompositionLocked(compositionArn, stageArn))}
	case "ListCompositions":
		return map[string]any{"compositions": s.listCompositionsLocked(), "nextToken": ""}

	case "CreateEncoderConfiguration":
		id := s.nextEncoderConfigurationIDLocked()
		encoderArn = ivsRealtimeArn("encoder-configuration", fmt.Sprintf("encoder-%08d", id))
		encoder := s.ensureEncoderConfigurationLocked(encoderArn)
		for k, v := range payload {
			encoder[k] = v
		}
		encoder["arn"] = encoderArn
		encoder["name"] = ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "name"), fmt.Sprintf("stackyard-encoder-%d", id))
		encoder["updatedAt"] = now
		return map[string]any{"encoderConfiguration": ivsRealtimeCloneMap(encoder)}
	case "GetEncoderConfiguration":
		return map[string]any{"encoderConfiguration": ivsRealtimeCloneMap(s.ensureEncoderConfigurationLocked(encoderArn))}
	case "ListEncoderConfigurations":
		return map[string]any{"encoderConfigurations": s.listEncoderConfigurationsLocked(), "nextToken": ""}
	case "DeleteEncoderConfiguration":
		delete(s.encoderConfigurations, encoderArn)
		return map[string]any{}

	case "CreateIngestConfiguration":
		id := s.nextIngestConfigurationIDLocked()
		ingestArn = ivsRealtimeArn("ingest-configuration", fmt.Sprintf("ingest-%08d", id))
		ingest := s.ensureIngestConfigurationLocked(ingestArn)
		for k, v := range payload {
			ingest[k] = v
		}
		ingest["arn"] = ingestArn
		ingest["name"] = ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "name"), fmt.Sprintf("stackyard-ingest-%d", id))
		ingest["updatedAt"] = now
		return map[string]any{"ingestConfiguration": ivsRealtimeCloneMap(ingest)}
	case "GetIngestConfiguration":
		return map[string]any{"ingestConfiguration": ivsRealtimeCloneMap(s.ensureIngestConfigurationLocked(ingestArn))}
	case "ListIngestConfigurations":
		return map[string]any{"ingestConfigurations": s.listIngestConfigurationsLocked(), "nextToken": ""}
	case "UpdateIngestConfiguration":
		ingest := s.ensureIngestConfigurationLocked(ingestArn)
		for k, v := range payload {
			ingest[k] = v
		}
		ingest["updatedAt"] = now
		return map[string]any{"ingestConfiguration": ivsRealtimeCloneMap(ingest)}
	case "DeleteIngestConfiguration":
		delete(s.ingestConfigurations, ingestArn)
		return map[string]any{}

	case "CreateStorageConfiguration":
		id := s.nextStorageConfigurationIDLocked()
		storageArn = ivsRealtimeArn("storage-configuration", fmt.Sprintf("storage-%08d", id))
		storage := s.ensureStorageConfigurationLocked(storageArn)
		for k, v := range payload {
			storage[k] = v
		}
		storage["arn"] = storageArn
		storage["name"] = ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "name"), fmt.Sprintf("stackyard-storage-%d", id))
		storage["updatedAt"] = now
		return map[string]any{"storageConfiguration": ivsRealtimeCloneMap(storage)}
	case "GetStorageConfiguration":
		return map[string]any{"storageConfiguration": ivsRealtimeCloneMap(s.ensureStorageConfigurationLocked(storageArn))}
	case "ListStorageConfigurations":
		return map[string]any{"storageConfigurations": s.listStorageConfigurationsLocked(), "nextToken": ""}
	case "DeleteStorageConfiguration":
		delete(s.storageConfigurations, storageArn)
		return map[string]any{}

	case "ImportPublicKey":
		id := s.nextPublicKeyIDLocked()
		publicKeyArn = ivsRealtimeArn("public-key", fmt.Sprintf("pk-%08d", id))
		pk := s.ensurePublicKeyLocked(publicKeyArn)
		for k, v := range payload {
			pk[k] = v
		}
		pk["arn"] = publicKeyArn
		pk["name"] = ivsRealtimeFirstNonEmpty(ivsRealtimeStringAny(payload, "name"), fmt.Sprintf("stackyard-public-key-%d", id))
		pk["updatedAt"] = now
		return map[string]any{"publicKey": ivsRealtimeCloneMap(pk)}
	case "GetPublicKey":
		return map[string]any{"publicKey": ivsRealtimeCloneMap(s.ensurePublicKeyLocked(publicKeyArn))}
	case "ListPublicKeys":
		return map[string]any{"publicKeys": s.listPublicKeysLocked(), "nextToken": ""}
	case "DeletePublicKey":
		delete(s.publicKeys, publicKeyArn)
		return map[string]any{}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for key, value := range ivsRealtimeTagsFromPayload(payload) {
			tags[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range ivsRealtimeTagKeysFromPayload(payload) {
			delete(tags, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": ivsRealtimeCloneTags(s.ensureTagsLocked(resourceArn))}
	}

	return map[string]any{}
}

func (s *ivsRealtimeStore) nextStageIDLocked() int64 { id := s.nextStageID; s.nextStageID++; return id }
func (s *ivsRealtimeStore) nextStageSessionIDLocked() int64 {
	id := s.nextStageSessionID
	s.nextStageSessionID++
	return id
}
func (s *ivsRealtimeStore) nextParticipantIDLocked() int64 {
	id := s.nextParticipantID
	s.nextParticipantID++
	return id
}
func (s *ivsRealtimeStore) nextParticipantTokenIDLocked() int64 {
	id := s.nextParticipantTokenID
	s.nextParticipantTokenID++
	return id
}
func (s *ivsRealtimeStore) nextCompositionIDLocked() int64 {
	id := s.nextCompositionID
	s.nextCompositionID++
	return id
}
func (s *ivsRealtimeStore) nextEncoderConfigurationIDLocked() int64 {
	id := s.nextEncoderConfigID
	s.nextEncoderConfigID++
	return id
}
func (s *ivsRealtimeStore) nextIngestConfigurationIDLocked() int64 {
	id := s.nextIngestConfigID
	s.nextIngestConfigID++
	return id
}
func (s *ivsRealtimeStore) nextStorageConfigurationIDLocked() int64 {
	id := s.nextStorageConfigID
	s.nextStorageConfigID++
	return id
}
func (s *ivsRealtimeStore) nextPublicKeyIDLocked() int64 {
	id := s.nextPublicKeyID
	s.nextPublicKeyID++
	return id
}

func (s *ivsRealtimeStore) firstStageArnLocked() string {
	for _, arn := range sortedKeysAny(s.stages) {
		return arn
	}
	return ""
}
func (s *ivsRealtimeStore) firstParticipantIDLocked() string {
	for _, id := range sortedKeysAny(s.participants) {
		return id
	}
	return ""
}
func (s *ivsRealtimeStore) firstCompositionArnLocked() string {
	for _, arn := range sortedKeysAny(s.compositions) {
		return arn
	}
	return ivsRealtimeArn("composition", "composition-00000001")
}
func (s *ivsRealtimeStore) firstEncoderConfigurationArnLocked() string {
	for _, arn := range sortedKeysAny(s.encoderConfigurations) {
		return arn
	}
	return ivsRealtimeArn("encoder-configuration", "encoder-00000001")
}
func (s *ivsRealtimeStore) firstIngestConfigurationArnLocked() string {
	for _, arn := range sortedKeysAny(s.ingestConfigurations) {
		return arn
	}
	return ivsRealtimeArn("ingest-configuration", "ingest-00000001")
}
func (s *ivsRealtimeStore) firstStorageConfigurationArnLocked() string {
	for _, arn := range sortedKeysAny(s.storageConfigurations) {
		return arn
	}
	return ivsRealtimeArn("storage-configuration", "storage-00000001")
}
func (s *ivsRealtimeStore) firstPublicKeyArnLocked() string {
	for _, arn := range sortedKeysAny(s.publicKeys) {
		return arn
	}
	return ivsRealtimeArn("public-key", "pk-00000001")
}

func (s *ivsRealtimeStore) ensureStageLocked(arn string) map[string]any {
	if stage, ok := s.stages[arn]; ok {
		return stage
	}
	id := strings.TrimPrefix(pathBase(arn), "")
	stage := map[string]any{
		"arn":             arn,
		"name":            ivsRealtimeFirstNonEmpty(id, "stackyard-stage"),
		"activeSessionId": fmt.Sprintf("session-%08d", s.nextStageSessionIDLocked()),
		"updatedAt":       time.Now().UTC().Format(time.RFC3339),
	}
	s.stages[arn] = stage
	return stage
}

func (s *ivsRealtimeStore) ensureStageSessionsLocked(stageArn string) []map[string]any {
	sessions, ok := s.stageSessions[stageArn]
	if ok && len(sessions) > 0 {
		return sessions
	}
	session := s.ensureStageSessionLocked(stageArn, fmt.Sprintf("session-%08d", s.nextStageSessionIDLocked()))
	return []map[string]any{session}
}

func (s *ivsRealtimeStore) ensureStageSessionLocked(stageArn, sessionID string) map[string]any {
	sessions := s.stageSessions[stageArn]
	for _, session := range sessions {
		if ivsRealtimeStringAny(session, "sessionId") == sessionID {
			return session
		}
	}
	session := map[string]any{
		"arn":       ivsRealtimeArn("stage-session", sessionID),
		"stageArn":  stageArn,
		"sessionId": sessionID,
		"startTime": time.Now().UTC().Format(time.RFC3339),
		"state":     "ACTIVE",
	}
	s.stageSessions[stageArn] = append(s.stageSessions[stageArn], session)
	return session
}

func (s *ivsRealtimeStore) ensureParticipantLocked(participantID, stageArn string) map[string]any {
	if participant, ok := s.participants[participantID]; ok {
		return participant
	}
	if strings.TrimSpace(participantID) == "" {
		participantID = fmt.Sprintf("participant-%08d", s.nextParticipantIDLocked())
	}
	participant := map[string]any{
		"participantId": participantID,
		"stageArn":      stageArn,
		"state":         "CONNECTED",
		"firstSeenAt":   time.Now().UTC().Format(time.RFC3339),
	}
	s.participants[participantID] = participant
	s.appendParticipantEventLocked(participantID, "PARTICIPANT_CONNECTED", time.Now().UTC().Format(time.RFC3339))
	return participant
}

func (s *ivsRealtimeStore) ensureParticipantEventsLocked(participantID string) []map[string]any {
	events := s.participantEvents[participantID]
	if len(events) == 0 {
		s.appendParticipantEventLocked(participantID, "PARTICIPANT_CONNECTED", time.Now().UTC().Format(time.RFC3339))
		events = s.participantEvents[participantID]
	}
	return events
}

func (s *ivsRealtimeStore) appendParticipantEventLocked(participantID, eventType, at string) {
	event := map[string]any{
		"participantId": participantID,
		"eventType":     eventType,
		"eventTime":     at,
	}
	s.participantEvents[participantID] = append(s.participantEvents[participantID], event)
}

func (s *ivsRealtimeStore) ensureCompositionLocked(arn, stageArn string) map[string]any {
	if composition, ok := s.compositions[arn]; ok {
		return composition
	}
	composition := map[string]any{
		"arn":       arn,
		"stageArn":  stageArn,
		"state":     "ACTIVE",
		"startTime": time.Now().UTC().Format(time.RFC3339),
	}
	s.compositions[arn] = composition
	return composition
}

func (s *ivsRealtimeStore) ensureEncoderConfigurationLocked(arn string) map[string]any {
	if cfg, ok := s.encoderConfigurations[arn]; ok {
		return cfg
	}
	cfg := map[string]any{"arn": arn, "name": "stackyard-encoder", "video": map[string]any{"height": 720, "width": 1280, "framerate": 30}}
	s.encoderConfigurations[arn] = cfg
	return cfg
}

func (s *ivsRealtimeStore) ensureIngestConfigurationLocked(arn string) map[string]any {
	if cfg, ok := s.ingestConfigurations[arn]; ok {
		return cfg
	}
	cfg := map[string]any{"arn": arn, "name": "stackyard-ingest", "state": "ACTIVE"}
	s.ingestConfigurations[arn] = cfg
	return cfg
}

func (s *ivsRealtimeStore) ensureStorageConfigurationLocked(arn string) map[string]any {
	if cfg, ok := s.storageConfigurations[arn]; ok {
		return cfg
	}
	cfg := map[string]any{"arn": arn, "name": "stackyard-storage", "s3": map[string]any{"bucketName": "stackyard-recordings"}}
	s.storageConfigurations[arn] = cfg
	return cfg
}

func (s *ivsRealtimeStore) ensurePublicKeyLocked(arn string) map[string]any {
	if pk, ok := s.publicKeys[arn]; ok {
		return pk
	}
	pk := map[string]any{"arn": arn, "name": "stackyard-public-key", "fingerprint": "stackyard-fingerprint"}
	s.publicKeys[arn] = pk
	return pk
}

func (s *ivsRealtimeStore) ensureTagsLocked(resourceArn string) map[string]string {
	if tags, ok := s.tags[resourceArn]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceArn] = tags
	return tags
}

func (s *ivsRealtimeStore) listStageSummariesLocked() []map[string]any {
	out := make([]map[string]any, 0, len(s.stages))
	for _, arn := range sortedKeysAny(s.stages) {
		stage := s.stages[arn]
		out = append(out, map[string]any{"arn": stage["arn"], "name": stage["name"], "activeSessionId": stage["activeSessionId"]})
	}
	return out
}

func (s *ivsRealtimeStore) listParticipantsForStageLocked(stageArn string) []map[string]any {
	out := []map[string]any{}
	for _, id := range sortedKeysAny(s.participants) {
		participant := s.participants[id]
		if stageArn != "" && ivsRealtimeStringAny(participant, "stageArn") != stageArn {
			continue
		}
		out = append(out, ivsRealtimeCloneMap(participant))
	}
	if len(out) == 0 {
		out = append(out, ivsRealtimeCloneMap(s.ensureParticipantLocked("", stageArn)))
	}
	return out
}

func (s *ivsRealtimeStore) listCompositionsLocked() []map[string]any {
	out := make([]map[string]any, 0, len(s.compositions))
	for _, arn := range sortedKeysAny(s.compositions) {
		composition := s.compositions[arn]
		out = append(out, map[string]any{"arn": composition["arn"], "state": composition["state"], "stageArn": composition["stageArn"]})
	}
	return out
}

func (s *ivsRealtimeStore) listEncoderConfigurationsLocked() []map[string]any {
	out := make([]map[string]any, 0, len(s.encoderConfigurations))
	for _, arn := range sortedKeysAny(s.encoderConfigurations) {
		cfg := s.encoderConfigurations[arn]
		out = append(out, map[string]any{"arn": cfg["arn"], "name": cfg["name"]})
	}
	return out
}

func (s *ivsRealtimeStore) listIngestConfigurationsLocked() []map[string]any {
	out := make([]map[string]any, 0, len(s.ingestConfigurations))
	for _, arn := range sortedKeysAny(s.ingestConfigurations) {
		cfg := s.ingestConfigurations[arn]
		out = append(out, map[string]any{"arn": cfg["arn"], "name": cfg["name"], "state": cfg["state"]})
	}
	return out
}

func (s *ivsRealtimeStore) listStorageConfigurationsLocked() []map[string]any {
	out := make([]map[string]any, 0, len(s.storageConfigurations))
	for _, arn := range sortedKeysAny(s.storageConfigurations) {
		cfg := s.storageConfigurations[arn]
		out = append(out, map[string]any{"arn": cfg["arn"], "name": cfg["name"]})
	}
	return out
}

func (s *ivsRealtimeStore) listPublicKeysLocked() []map[string]any {
	out := make([]map[string]any, 0, len(s.publicKeys))
	for _, arn := range sortedKeysAny(s.publicKeys) {
		pk := s.publicKeys[arn]
		out = append(out, map[string]any{"arn": pk["arn"], "name": pk["name"], "fingerprint": pk["fingerprint"]})
	}
	return out
}

func sortedKeysAny[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func ivsRealtimePath(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	return strings.TrimSpace(pathParams[key])
}

func ivsRealtimeStringAny(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func ivsRealtimeFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func ivsRealtimeArn(resourceType, id string) string {
	return fmt.Sprintf("arn:aws:ivs-realtime:us-east-1:123456789012:%s/%s", strings.TrimSpace(resourceType), strings.TrimSpace(id))
}

func ivsRealtimeTagsFromPayload(payload map[string]any) map[string]string {
	out := map[string]string{}
	raw, ok := payload["tags"]
	if !ok || raw == nil {
		return out
	}
	switch typed := raw.(type) {
	case map[string]any:
		for key, value := range typed {
			if key = strings.TrimSpace(key); key == "" {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprint(value))
		}
	case map[string]string:
		for key, value := range typed {
			if key = strings.TrimSpace(key); key == "" {
				continue
			}
			out[key] = strings.TrimSpace(value)
		}
	}
	return out
}

func ivsRealtimeTagKeysFromPayload(payload map[string]any) []string {
	keys := []string{}
	raw, ok := payload["tagKeys"]
	if !ok || raw == nil {
		return keys
	}
	switch typed := raw.(type) {
	case []any:
		for _, value := range typed {
			key := strings.TrimSpace(fmt.Sprint(value))
			if key != "" {
				keys = append(keys, key)
			}
		}
	case []string:
		for _, value := range typed {
			key := strings.TrimSpace(value)
			if key != "" {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func ivsRealtimeCloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = ivsRealtimeCloneAny(value)
	}
	return out
}

func ivsRealtimeCloneSliceMap(input []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(input))
	for _, item := range input {
		out = append(out, ivsRealtimeCloneMap(item))
	}
	return out
}

func ivsRealtimeCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return ivsRealtimeCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, ivsRealtimeCloneAny(item))
		}
		return out
	default:
		return typed
	}
}

func ivsRealtimeCloneTags(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func pathBase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	idx := strings.LastIndex(value, "/")
	if idx < 0 || idx == len(value)-1 {
		return value
	}
	return value[idx+1:]
}
