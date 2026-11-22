package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// kvsPlaybackSession is an HLS or DASH session handed out by
// Get{HLS,DASH}StreamingSessionURL. The URLs it serves are authorized by the
// SessionToken alone, like the service's.
type kvsPlaybackSession struct {
	token      string
	dash       bool
	streamARN  string
	mode       string // LIVE, LIVE_REPLAY, ON_DEMAND
	window     kvsRange
	maxResults int64
	expires    time.Time
}

const (
	kvsHLSMasterPath   = "/hls/v1/getHLSMasterPlaylist.m3u8"
	kvsHLSMediaPath    = "/hls/v1/getHLSMediaPlaylist.m3u8"
	kvsHLSFragmentPath = "/hls/v1/getKVSFragment.mkv"
	kvsDASHPath        = "/dash/v1/getDASHManifest.mpd"
	kvsDASHFragment    = "/dash/v1/getKVSFragment.mkv"
)

func (s *Server) kvsGetStreamingSessionURL(w http.ResponseWriter, r *http.Request, action string, payload map[string]any) *kvsFault {
	dash := action == "GetDASHStreamingSessionURL"
	selectorKey, maxKey := "HLSFragmentSelector", "MaxMediaPlaylistFragmentResults"
	if dash {
		selectorKey, maxKey = "DASHFragmentSelector", "MaxManifestFragmentResults"
		if _, fault := kvsEnum(payload, "DisplayFragmentNumber", "NEVER", "ALWAYS", "NEVER"); fault != nil {
			return fault
		}
	} else {
		for key, allowed := range map[string][]string{
			"ContainerFormat":   {"FRAGMENTED_MP4", "MPEG_TS"},
			"DiscontinuityMode": {"ALWAYS", "NEVER", "ON_DISCONTINUITY"},
		} {
			if _, fault := kvsEnum(payload, key, allowed[0], allowed...); fault != nil {
				return fault
			}
		}
	}
	if _, fault := kvsEnum(payload, "DisplayFragmentTimestamp", "NEVER", "ALWAYS", "NEVER"); fault != nil {
		return fault
	}
	mode, fault := kvsEnum(payload, "PlaybackMode", "LIVE", "LIVE", "LIVE_REPLAY", "ON_DEMAND")
	if fault != nil {
		return fault
	}
	expires, fault := kvsPayloadInt(payload, "Expires", 300, 43200, 300)
	if fault != nil {
		return fault
	}
	defaultMax := int64(1000)
	if mode == "LIVE" {
		defaultMax = 5
	}
	maxResults, fault := kvsPayloadInt(payload, maxKey, 1, 5000, defaultMax)
	if fault != nil {
		return fault
	}
	selector, _ := payload[selectorKey].(map[string]any)
	if selector == nil {
		selector = map[string]any{}
	}
	window, hasRange, fault := kvsParseRange(selector, "FragmentSelectorType", false, mode == "ON_DEMAND")
	if fault != nil {
		return fault
	}
	switch {
	case mode == "LIVE" && hasRange:
		return kvsInvalidArgument("TimestampRange must not be set when PlaybackMode is LIVE")
	case mode == "LIVE_REPLAY" && !hasRange:
		return kvsInvalidArgument("TimestampRange.StartTimestamp is required when PlaybackMode is LIVE_REPLAY")
	case mode == "ON_DEMAND" && window.end.Sub(window.start) > kvsMaxClipRange:
		return kvsInvalidArgument("The ON_DEMAND time range can't exceed 24 hours")
	}
	stream, fragments, fault := s.kvsArchive(payload)
	if fault != nil {
		return fault
	}
	if stream.RetentionHours == 0 {
		return kvsNoDataRetention()
	}
	token := kvsNewSessionToken()
	session := kvsPlaybackSession{token: token, dash: dash, streamARN: stream.ARN, mode: mode, window: window, maxResults: maxResults, expires: time.Now().Add(time.Duration(expires) * time.Second)}
	if len(session.fragments(fragments)) == 0 {
		return kvsNotFound("No fragments found in the stream for the requested playback window.")
	}
	s.kinesisvideostreams.mu.Lock()
	now := time.Now()
	for t, existing := range s.kinesisvideostreams.sessions {
		if now.After(existing.expires) {
			delete(s.kinesisvideostreams.sessions, t)
		}
	}
	s.kinesisvideostreams.sessions[token] = session
	s.kinesisvideostreams.mu.Unlock()

	base := kvsRequestBaseURL(r)
	if dash {
		respondKinesisVideoStreamsJSON(w, http.StatusOK, map[string]any{"DASHStreamingSessionURL": base + kvsDASHPath + "?SessionToken=" + token})
	} else {
		respondKinesisVideoStreamsJSON(w, http.StatusOK, map[string]any{"HLSStreamingSessionURL": base + kvsHLSMasterPath + "?SessionToken=" + token})
	}
	return nil
}

// fragments selects the session's fragments: the newest maxResults for
// LIVE, else the first maxResults inside the window.
func (p kvsPlaybackSession) fragments(all []kvsFragment) []kvsFragment {
	if p.mode == "LIVE" {
		return all[max(0, len(all)-int(p.maxResults)):]
	}
	in := p.window.filter(all)
	return in[:min(len(in), int(p.maxResults))]
}

func kvsNewSessionToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// handleKinesisVideoPlayback serves the HLS/DASH URLs. It claims only the
// exact session paths that carry a SessionToken.
func (s *Server) handleKinesisVideoPlayback(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	switch path {
	case kvsHLSMasterPath, kvsHLSMediaPath, kvsHLSFragmentPath, kvsDASHPath, kvsDASHFragment:
	default:
		return false
	}
	token := r.URL.Query().Get("SessionToken")
	if token == "" {
		return false
	}
	s.kinesisvideostreams.mu.Lock()
	session, ok := s.kinesisvideostreams.sessions[token]
	if ok && time.Now().After(session.expires) {
		delete(s.kinesisvideostreams.sessions, token)
		ok = false
	}
	s.kinesisvideostreams.mu.Unlock()
	if !ok || session.dash != strings.HasPrefix(path, "/dash/") {
		respondKinesisVideoStreamsError(w, http.StatusForbidden, "NotAuthorizedException", "The session token is invalid or has expired.")
		return true
	}
	_, all, _, fault := s.kinesisvideostreams.mediaSnapshot("", session.streamARN, time.Now().UTC())
	if fault != nil {
		respondKinesisVideoStreamsError(w, fault.status, fault.code, fault.msg)
		return true
	}
	// Build links from the stored session, not from request input.
	query := "SessionToken=" + url.QueryEscape(session.token)
	switch path {
	case kvsHLSMasterPath:
		kvsWritePlaylist(w, "application/x-mpegURL", "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-STREAM-INF:BANDWIDTH=2000000\ngetHLSMediaPlaylist.m3u8?"+query+"\n")
	case kvsHLSMediaPath:
		kvsWritePlaylist(w, "application/x-mpegURL", kvsHLSMediaPlaylist(session, session.fragments(all), query))
	case kvsDASHPath:
		kvsWritePlaylist(w, "application/dash+xml", kvsDASHManifest(session, session.fragments(all), query))
	default:
		// Only fragments inside this session's window are reachable.
		visible := session.fragments(all)
		idx := kvsFragmentIndex(visible, r.URL.Query().Get("FragmentNumber"))
		if idx < 0 {
			respondKinesisVideoStreamsError(w, http.StatusNotFound, "ResourceNotFoundException", "The fragment was not found in the stream.")
			return true
		}
		kvsWriteFragments(w, visible[idx:idx+1])
	}
	return true
}

// kvsWritePlaylist writes a generated playlist; every value in it is a
// Stackyard-made token or fragment number.
func kvsWritePlaylist(w http.ResponseWriter, contentType, body string) {
	switch contentType {
	case "application/dash+xml":
		w.Header().Set("Content-Type", "application/dash+xml")
	default:
		w.Header().Set("Content-Type", "application/x-mpegURL")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func kvsFragmentSeconds(f kvsFragment) float64 {
	return float64(max(f.DurationMs, 1)) / 1000
}

func kvsHLSMediaPlaylist(p kvsPlaybackSession, fragments []kvsFragment, query string) string {
	target := 1.0
	for _, f := range fragments {
		target = math.Max(target, math.Ceil(kvsFragmentSeconds(f)))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%d\n", int(target))
	if len(fragments) > 0 {
		fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", fragments[0].seq)
	}
	if p.mode == "ON_DEMAND" {
		b.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	}
	for _, f := range fragments {
		fmt.Fprintf(&b, "#EXTINF:%.3f,\ngetKVSFragment.mkv?%s&FragmentNumber=%s\n", kvsFragmentSeconds(f), query, f.Number)
	}
	if p.mode == "ON_DEMAND" || (p.mode == "LIVE_REPLAY" && !p.window.end.IsZero()) {
		b.WriteString("#EXT-X-ENDLIST\n")
	}
	return b.String()
}

func kvsDASHManifest(p kvsPlaybackSession, fragments []kvsFragment, query string) string {
	kind, total := "dynamic", 0.0
	if p.mode == "ON_DEMAND" {
		kind = "static"
	}
	var segments strings.Builder
	for _, f := range fragments {
		total += kvsFragmentSeconds(f)
		fmt.Fprintf(&segments, "          <SegmentURL media=\"getKVSFragment.mkv?%s&amp;FragmentNumber=%s\"/>\n", query, f.Number)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="%s" profiles="urn:mpeg:dash:profile:isoff-live:2011" minBufferTime="PT2S" mediaPresentationDuration="PT%.3fS">
  <Period id="0">
    <AdaptationSet mimeType="video/webm" segmentAlignment="true">
      <Representation id="0" bandwidth="2000000">
        <SegmentList timescale="1000">
%s        </SegmentList>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>
`, kind, total, segments.String())
}
