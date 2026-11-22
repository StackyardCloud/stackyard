package server

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	// ponytail: per-stream byte cap with drop-oldest, so a long-retention
	// stream can't grow without bound in memory; raise it if a test needs more.
	kvsMaxStreamBytes = 256 << 20
	// A stream with DataRetentionInHours 0 does not persist media. Its
	// fragments stay this long only so live GetMedia readers can catch up.
	kvsLiveOnlyWindow   = 5 * time.Minute
	kvsMaxFragmentBytes = 50 << 20
	// kvsMaxHeaderBytes caps the EBML+Info+Tracks headers copied into each
	// fragment (real producer headers are a few KiB).
	kvsMaxHeaderBytes = 1 << 20
)

type kvsFault struct {
	status    int
	code, msg string
}

func kvsInvalidArgument(format string, args ...any) *kvsFault {
	return &kvsFault{http.StatusBadRequest, "InvalidArgumentException", fmt.Sprintf(format, args...)}
}

func kvsNotFound(format string, args ...any) *kvsFault {
	return &kvsFault{http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf(format, args...)}
}

// kvsFragment is one stored fragment. Data is the standalone MKV returned by
// the media APIs and is never mutated after it is stored.
type kvsFragment struct {
	Number     string
	seq        uint64
	Producer   time.Time
	Server     time.Time
	DurationMs int64
	Data       []byte
}

type kvsStreamMedia struct {
	fragments    []kvsFragment
	bytes        int
	nextSeq      uint64
	lastProducer time.Time
	notify       chan struct{} // closed and replaced whenever fragments land
}

// kvsMediaStream is a read view of one stream for the media APIs.
type kvsMediaStream struct {
	ARN            string
	RetentionHours int
	MediaType      string
}

func kvsFragmentNumber(seq uint64) string {
	// Real fragment numbers are long decimal strings; keep that shape.
	return fmt.Sprintf("91343852333181432%030d", seq)
}

// mediaStreamLocked resolves StreamName/StreamARN to an existing ACTIVE
// stream without creating one (the control-plane Handle creates on demand).
func (s *kinesisVideoStreamsStore) mediaStreamLocked(name, arn string) (kvsMediaStream, *kvsFault) {
	name, arn = strings.TrimSpace(name), strings.TrimSpace(arn)
	if name == "" && arn == "" {
		return kvsMediaStream{}, kvsInvalidArgument("StreamName or StreamARN must be specified")
	}
	var stream map[string]any
	if arn != "" {
		stream = s.streams[s.streamsByARN[arn]]
		if stream != nil && name != "" && kinesisVideoStreamsString(stream, "StreamName", "") != name {
			return kvsMediaStream{}, kvsInvalidArgument("StreamName and StreamARN refer to different streams")
		}
	} else {
		stream = s.streams[name]
	}
	if stream == nil || kinesisVideoStreamsString(stream, "Status", "") != "ACTIVE" {
		return kvsMediaStream{}, kvsNotFound("The requested stream was not found or is not active.")
	}
	return kvsMediaStream{
		ARN:            kinesisVideoStreamsString(stream, "StreamARN", ""),
		RetentionHours: kinesisVideoStreamsIntAny(stream["DataRetentionInHours"], 0),
		MediaType:      kinesisVideoStreamsString(stream, "MediaType", ""),
	}, nil
}

// streamMediaLocked returns the stream's fragment log after dropping
// fragments that are past retention.
func (s *kinesisVideoStreamsStore) streamMediaLocked(stream kvsMediaStream, now time.Time) *kvsStreamMedia {
	m := s.media[stream.ARN]
	if m == nil {
		m = &kvsStreamMedia{notify: make(chan struct{})}
		s.media[stream.ARN] = m
	}
	keep := kvsLiveOnlyWindow
	if stream.RetentionHours > 0 {
		keep = time.Duration(stream.RetentionHours) * time.Hour
	}
	cutoff := now.Add(-keep)
	drop := 0
	for drop < len(m.fragments) && m.fragments[drop].Server.Before(cutoff) {
		m.bytes -= len(m.fragments[drop].Data)
		drop++
	}
	m.fragments = m.fragments[drop:]
	return m
}

type kvsPutMediaRequest struct {
	streamName, streamARN string
	absolute              bool
	producerStart         time.Time
	clusters              []kvsMKVCluster
	parseErr              error
}

// putMedia stores parsed clusters as fragments and returns the PutMedia
// acknowledgement events in order. The first bad cluster ends the request
// with an ERROR event, as the service does.
func (s *kinesisVideoStreamsStore) putMedia(req kvsPutMediaRequest, now time.Time) ([]map[string]any, *kvsFault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream, fault := s.mediaStreamLocked(req.streamName, req.streamARN)
	if fault != nil {
		return nil, fault
	}
	m := s.streamMediaLocked(stream, now)
	acks := []map[string]any{}
	fail := func(timecode int64, id int) {
		acks = append(acks, map[string]any{"EventType": "ERROR", "FragmentTimecode": timecode, "ErrorId": id})
	}
	stored := false
	for _, c := range req.clusters {
		if c.timecode > math.MaxInt64/c.scale {
			fail(c.timecode, 4006) // INVALID_MKV_DATA
			break
		}
		offset := time.Duration(c.timecode * c.scale)
		producer := req.producerStart.Add(offset)
		if req.absolute {
			producer = time.Unix(0, int64(offset)).UTC()
		}
		if producer.Before(m.lastProducer) {
			fail(c.timecode, 4004) // FRAGMENT_TIMECODE_LESSER_THAN_PREVIOUS
			break
		}
		// Every fragment repeats the EBML/Info/Tracks headers, so cap them:
		// otherwise one large header plus many tiny clusters multiplies the
		// stored bytes far past the request body size.
		if len(c.ebml)+len(c.info)+len(c.tracks) > kvsMaxHeaderBytes {
			fail(c.timecode, 4006) // INVALID_MKV_DATA
			break
		}
		if len(c.content) > kvsMaxFragmentBytes {
			fail(c.timecode, 4001) // MAX_FRAGMENT_SIZE_REACHED
			break
		}
		m.nextSeq++
		number := kvsFragmentNumber(m.nextSeq)
		frag := kvsFragment{
			Number:     number,
			seq:        m.nextSeq,
			Producer:   producer,
			Server:     now,
			DurationMs: c.maxBlockTicks * c.scale / int64(time.Millisecond),
			Data:       kvsFragmentMKV(c, number, producer, now),
		}
		m.fragments = append(m.fragments, frag)
		m.bytes += len(frag.Data)
		m.lastProducer = producer
		stored = true
		for m.bytes > kvsMaxStreamBytes && len(m.fragments) > 1 {
			m.bytes -= len(m.fragments[0].Data)
			m.fragments = m.fragments[1:]
		}
		for _, event := range []string{"BUFFERING", "RECEIVED", "PERSISTED"} {
			if event == "PERSISTED" && stream.RetentionHours == 0 {
				continue
			}
			acks = append(acks, map[string]any{"EventType": event, "FragmentTimecode": c.timecode, "FragmentNumber": number})
		}
	}
	if req.parseErr != nil && (len(acks) == 0 || acks[len(acks)-1]["EventType"] != "ERROR") {
		fail(0, 4006)
	}
	if stored {
		close(m.notify)
		m.notify = make(chan struct{})
	}
	return acks, nil
}

// mediaSnapshot returns the stream view, a copy of its retained fragments
// and the channel that closes when new fragments land.
func (s *kinesisVideoStreamsStore) mediaSnapshot(name, arn string, now time.Time) (kvsMediaStream, []kvsFragment, <-chan struct{}, *kvsFault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream, fault := s.mediaStreamLocked(name, arn)
	if fault != nil {
		return kvsMediaStream{}, nil, nil, fault
	}
	m := s.streamMediaLocked(stream, now)
	return stream, append([]kvsFragment(nil), m.fragments...), m.notify, nil
}
