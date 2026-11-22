package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ponytail: a NOW GetMedia returns what arrives within this window instead of
// holding the connection open like the service; lengthen it for slow producers.
const kvsGetMediaNowWait = 2 * time.Second

// kvsDataPlaneRoutes maps the POST paths of both data planes (Kinesis Video
// Media and Archived Media) to their operations. They share the
// "kinesisvideo" signing name with the control plane, so this router must sit
// ahead of handleKinesisVideoStreamsRESTRouter, which claims every
// kinesisvideo-signed POST.
var kvsDataPlaneRoutes = func() map[string]string {
	out := map[string]string{}
	for _, op := range kinesisVideoMediaOperations {
		out[op.URI] = op.Name
	}
	for _, op := range kinesisVideoArchivedMediaOperations {
		out[op.URI] = op.Name
	}
	return out
}()

func (s *Server) handleKinesisVideoMediaRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		return s.handleKinesisVideoPlayback(w, r)
	}
	if r.Method != http.MethodPost {
		return false
	}
	action, ok := kvsDataPlaneRoutes[rawRequestPath(r)]
	if !ok {
		return false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "kinesisvideo" {
		return false
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "kinesisvideo"); !ok {
		respondKinesisVideoStreamsError(w, status, code, msg)
		return true
	}
	if action == "PutMedia" {
		s.kvsPutMedia(w, r)
		return true
	}
	payload, err := parseKinesisVideoStreamsPayload(r)
	if err != nil {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "invalid JSON body")
		return true
	}
	var fault *kvsFault
	switch action {
	case "GetMedia":
		fault = s.kvsGetMedia(w, r, payload)
	case "ListFragments":
		fault = s.kvsListFragments(w, payload)
	case "GetMediaForFragmentList":
		fault = s.kvsGetMediaForFragmentList(w, payload)
	case "GetClip":
		fault = s.kvsGetClip(w, payload)
	case "GetImages":
		fault = s.kvsGetImages(w, payload)
	case "GetHLSStreamingSessionURL", "GetDASHStreamingSessionURL":
		fault = s.kvsGetStreamingSessionURL(w, r, action, payload)
	}
	if fault != nil {
		respondKinesisVideoStreamsError(w, fault.status, fault.code, fault.msg)
	}
	return true
}

// kvsPutMedia ingests an MKV body. Stackyard reads the whole request (the
// SigV4 check already buffers it), so one PutMedia call carries at most
// 25 MiB; acknowledgements come back as one JSON event per line.
func (s *Server) kvsPutMedia(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	req := kvsPutMediaRequest{
		streamName:    r.Header.Get("x-amzn-stream-name"),
		streamARN:     r.Header.Get("x-amzn-stream-arn"),
		producerStart: now,
	}
	switch strings.ToUpper(strings.TrimSpace(r.Header.Get("x-amzn-fragment-timecode-type"))) {
	case "ABSOLUTE":
		req.absolute = true
	case "RELATIVE":
	default:
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "x-amzn-fragment-timecode-type must be ABSOLUTE or RELATIVE")
		return
	}
	if raw := strings.TrimSpace(r.Header.Get("x-amzn-producer-start-timestamp")); raw != "" {
		seconds, err := strconv.ParseFloat(raw, 64)
		if err != nil || seconds < 0 || math.IsInf(seconds, 0) {
			respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "x-amzn-producer-start-timestamp must be epoch seconds")
			return
		}
		req.producerStart = kvsTimeFromSeconds(seconds)
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "could not read the request body")
		return
	}
	if len(body) >= 25<<20 {
		respondKinesisVideoStreamsError(w, http.StatusBadRequest, "InvalidArgumentException", "Stackyard accepts at most 25 MiB of media per PutMedia request")
		return
	}
	req.clusters, req.parseErr = parseKVSMKV(body)
	acks, fault := s.kinesisvideostreams.putMedia(req, now)
	if fault != nil {
		respondKinesisVideoStreamsError(w, fault.status, fault.code, fault.msg)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	for _, ack := range acks {
		_ = enc.Encode(ack)
	}
}

func (s *Server) kvsGetMedia(w http.ResponseWriter, r *http.Request, payload map[string]any) *kvsFault {
	selector, _ := payload["StartSelector"].(map[string]any)
	if selector == nil {
		return kvsInvalidArgument("StartSelector is required")
	}
	selectorType := kvsString(selector["StartSelectorType"])
	now := time.Now().UTC()
	_, fragments, notify, fault := s.kinesisvideostreams.mediaSnapshot(kvsPayloadString(payload, "StreamName"), kvsPayloadString(payload, "StreamARN"), now)
	if fault != nil {
		return fault
	}
	start := len(fragments)
	switch selectorType {
	case "EARLIEST":
		start = 0
	case "NOW":
		var after uint64
		if n := len(fragments); n > 0 {
			after = fragments[n-1].seq
		}
		fragments = s.kvsWaitForFragments(r, payload, notify, after)
		start = 0
	case "FRAGMENT_NUMBER", "CONTINUATION_TOKEN":
		key := "AfterFragmentNumber"
		if selectorType == "CONTINUATION_TOKEN" {
			key = "ContinuationToken"
		}
		number := kvsString(selector[key])
		if number == "" {
			return kvsInvalidArgument("%s is required for StartSelectorType %s", key, selectorType)
		}
		idx := kvsFragmentIndex(fragments, number)
		if idx < 0 {
			return kvsNotFound("No fragment with number %s was found in the stream.", number)
		}
		start = idx
		if selectorType == "CONTINUATION_TOKEN" {
			start = idx + 1 // the token names the last fragment already read
		}
	case "PRODUCER_TIMESTAMP", "SERVER_TIMESTAMP":
		at, ok, fault := kvsPayloadTime(selector, "StartTimestamp")
		if fault != nil {
			return fault
		}
		if !ok {
			return kvsInvalidArgument("StartTimestamp is required for StartSelectorType %s", selectorType)
		}
		for i, f := range fragments {
			if !kvsFragmentTime(f, selectorType == "PRODUCER_TIMESTAMP").Before(at) {
				start = i
				break
			}
		}
	default:
		return kvsInvalidArgument("StartSelectorType must be one of FRAGMENT_NUMBER, SERVER_TIMESTAMP, PRODUCER_TIMESTAMP, NOW, EARLIEST, CONTINUATION_TOKEN")
	}
	kvsWriteFragments(w, fragments[start:])
	return nil
}

// kvsWaitForFragments waits up to kvsGetMediaNowWait for fragments newer
// than seq `after`, then returns them.
func (s *Server) kvsWaitForFragments(r *http.Request, payload map[string]any, notify <-chan struct{}, after uint64) []kvsFragment {
	timer := time.NewTimer(kvsGetMediaNowWait)
	defer timer.Stop()
	select {
	case <-notify:
	case <-timer.C:
	case <-r.Context().Done():
	}
	_, fragments, _, fault := s.kinesisvideostreams.mediaSnapshot(kvsPayloadString(payload, "StreamName"), kvsPayloadString(payload, "StreamARN"), time.Now().UTC())
	if fault != nil {
		return nil
	}
	for i, f := range fragments {
		if f.seq > after {
			return fragments[i:]
		}
	}
	return nil
}

// kvsWriteFragments sends stored MKV fragments. The bytes came from a
// producer, so the media type is a constant non-HTML type set right here.
func kvsWriteFragments(w http.ResponseWriter, fragments []kvsFragment) {
	w.Header().Set("Content-Type", "video/webm")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	for _, f := range fragments {
		_, _ = w.Write(f.Data)
	}
}

func kvsFragmentIndex(fragments []kvsFragment, number string) int {
	for i, f := range fragments {
		if f.Number == number {
			return i
		}
	}
	return -1
}

func kvsFragmentTime(f kvsFragment, producer bool) time.Time {
	if producer {
		return f.Producer
	}
	return f.Server
}

func kvsPayloadString(payload map[string]any, key string) string {
	return kvsString(payload[key])
}

// kvsString reads a JSON string or number member; anything else is "".
func kvsString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	}
	return ""
}

// kvsPayloadTime reads a rest-json timestamp: epoch seconds (the wire form)
// or an RFC 3339 string.
func kvsPayloadTime(payload map[string]any, key string) (time.Time, bool, *kvsFault) {
	raw, present := payload[key]
	if !present || raw == nil {
		return time.Time{}, false, nil
	}
	text := strings.TrimSpace(fmt.Sprint(raw))
	if seconds, err := strconv.ParseFloat(text, 64); err == nil && seconds >= 0 && !math.IsInf(seconds, 0) {
		return kvsTimeFromSeconds(seconds), true, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, text); err == nil {
		return t.UTC(), true, nil
	}
	return time.Time{}, false, kvsInvalidArgument("%s must be a timestamp", key)
}

func kvsTimeFromSeconds(seconds float64) time.Time {
	whole, frac := math.Modf(seconds)
	return time.Unix(int64(whole), int64(frac*1e9)).UTC()
}

// kvsPayloadInt reads an integer member, enforcing the model's bounds. Every
// bound these APIs use fits in 32 bits, so values parse at that size.
func kvsPayloadInt(payload map[string]any, key string, lo, hi, def int64) (int64, *kvsFault) {
	raw, present := payload[key]
	if !present || raw == nil {
		return def, nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(raw)), 10, 32)
	if err != nil || n < lo || n > hi {
		return 0, kvsInvalidArgument("%s must be an integer between %d and %d", key, lo, hi)
	}
	return n, nil
}

func kvsEpochSeconds(t time.Time) float64 {
	return float64(t.UnixMilli()) / 1000
}

func kvsRequestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "localhost:4566"
	}
	return scheme + "://" + host
}
