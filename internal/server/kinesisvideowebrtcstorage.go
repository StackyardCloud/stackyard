package server

import (
	"net/http"
	"time"
)

// kvsStorageMaxViewers is the WebRTC ingestion viewer limit per session.
const kvsStorageMaxViewers = 3

// kvsStorageSession records who joined a channel's WebRTC storage session.
type kvsStorageSession struct {
	MasterJoinedAt time.Time
	Viewers        map[string]time.Time
}

// ponytail: joining only records the session; Stackyard does not open a
// WebRTC peer to the master or ingest the call into the linked stream.
func (s *Server) kvsJoinStorageSession(w http.ResponseWriter, action string, payload map[string]any) *kvsFault {
	arn := kinesisVideoStreamsPayloadString(payload, []string{"channelArn", "ChannelArn", "ChannelARN"}, "")
	clientID := kinesisVideoStreamsPayloadString(payload, []string{"clientId", "ClientId"}, "")
	viewer := action == "JoinStorageSessionAsViewer"
	if viewer && !kvsClientIDPattern.MatchString(clientID) {
		return kvsInvalidArgument("invalid clientId: required, [a-zA-Z0-9_.-]+, at most 256 characters")
	}
	store := s.kinesisvideostreams
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, fault := store.kvsChannelLocked(arn); fault != nil {
		return fault
	}
	cfg := store.mediaStorageConfigs[arn]
	streamARN := kinesisVideoStreamsString(cfg, "StreamARN", "")
	stream := store.streams[store.streamsByARN[streamARN]]
	if kinesisVideoStreamsString(cfg, "Status", "") != "ENABLED" || stream == nil || kinesisVideoStreamsString(stream, "Status", "") != "ACTIVE" {
		return kvsInvalidArgument("The signaling channel is not configured for media storage with an active stream.")
	}
	session := store.storageSessions[arn]
	if session == nil {
		session = &kvsStorageSession{Viewers: map[string]time.Time{}}
		store.storageSessions[arn] = session
	}
	now := time.Now().UTC()
	if viewer {
		if _, joined := session.Viewers[clientID]; !joined && len(session.Viewers) >= kvsStorageMaxViewers {
			return &kvsFault{status: http.StatusBadRequest, code: "ClientLimitExceededException", msg: "The storage session already has the maximum number of viewers."}
		}
		session.Viewers[clientID] = now
	} else {
		session.MasterJoinedAt = now
	}
	w.WriteHeader(http.StatusOK)
	return nil
}
