package server

import (
	"encoding/binary"
	"errors"
	"strconv"
	"time"
)

// Matroska (EBML) element IDs that PutMedia ingest and GetMedia output use.
const (
	mkvIDEBML          = 0x1A45DFA3
	mkvIDSegment       = 0x18538067
	mkvIDInfo          = 0x1549A966
	mkvIDTimecodeScale = 0x2AD7B1
	mkvIDTracks        = 0x1654AE6B
	mkvIDCluster       = 0x1F43B675
	mkvIDTimecode      = 0xE7
	mkvIDSimpleBlock   = 0xA3
	mkvIDBlockGroup    = 0xA0
	mkvIDBlock         = 0xA1
	mkvIDPosition      = 0xA7
	mkvIDPrevSize      = 0xAB
	mkvIDSilentTracks  = 0x5854
	mkvIDTags          = 0x1254C367
	mkvIDTag           = 0x7373
	mkvIDSimpleTag     = 0x67C8
	mkvIDTagName       = 0x45A3
	mkvIDTagString     = 0x4487

	mkvDefaultTimecodeScale = 1_000_000 // nanoseconds per cluster/block tick
)

var errMKVInvalid = errors.New("invalid MKV data")

// kvsMKVCluster is one ingested cluster (a KVS fragment) together with the
// stream headers that make it decodable on its own.
type kvsMKVCluster struct {
	ebml, info, tracks []byte
	scale              int64 // TimecodeScale, ns per tick
	timecode           int64 // cluster Timecode, ticks
	content            []byte
	maxBlockTicks      int64 // largest block timecode relative to the cluster
}

type mkvElement struct {
	id               uint32
	start, dataStart int
	end              int // -1 while the size is unknown
	unknownSize      bool
}

// parseKVSMKV splits a PutMedia body into clusters. It returns the clusters
// that parsed cleanly before any error, so earlier fragments can be acked.
func parseKVSMKV(data []byte) ([]kvsMKVCluster, error) {
	var (
		out                []kvsMKVCluster
		ebml, info, tracks []byte
		scale              int64 = mkvDefaultTimecodeScale
	)
	pos := 0
	for pos < len(data) {
		el, err := readMKVElement(data, pos, len(data))
		if err != nil {
			return out, err
		}
		switch el.id {
		case mkvIDEBML:
			if el.unknownSize {
				return out, errMKVInvalid
			}
			ebml = data[el.start:el.end]
			info, tracks, scale = nil, nil, mkvDefaultTimecodeScale
			pos = el.end
		case mkvIDSegment:
			if ebml == nil {
				return out, errMKVInvalid
			}
			end := len(data)
			if !el.unknownSize {
				end = el.end
			}
			next, err := parseMKVSegment(data, el.dataStart, end, func(child mkvElement, body []byte) error {
				switch child.id {
				case mkvIDInfo:
					info = data[child.start:child.end]
					scale = mkvTimecodeScale(body)
				case mkvIDTracks:
					tracks = data[child.start:child.end]
				case mkvIDCluster:
					if tracks == nil {
						return errMKVInvalid
					}
					tc, maxTicks, err := mkvClusterTimes(body)
					if err != nil {
						return err
					}
					out = append(out, kvsMKVCluster{ebml: ebml, info: info, tracks: tracks, scale: scale, timecode: tc, content: body, maxBlockTicks: maxTicks})
				}
				return nil
			})
			if err != nil {
				return out, err
			}
			pos = next
		default:
			return out, errMKVInvalid
		}
	}
	return out, nil
}

// parseMKVSegment walks segment children in [start, end). An unknown-size
// segment ends where the next EBML header or segment begins.
func parseMKVSegment(data []byte, start, end int, visit func(mkvElement, []byte) error) (int, error) {
	pos := start
	for pos < end {
		el, err := readMKVElement(data, pos, end)
		if err != nil {
			return pos, err
		}
		if el.id == mkvIDEBML || el.id == mkvIDSegment {
			return pos, nil
		}
		if el.unknownSize {
			if el.id != mkvIDCluster {
				return pos, errMKVInvalid
			}
			el.end = mkvUnknownClusterEnd(data, el.dataStart, end)
		}
		if err := visit(el, data[el.dataStart:el.end]); err != nil {
			return pos, err
		}
		pos = el.end
	}
	return pos, nil
}

// mkvUnknownClusterEnd finds the end of an unknown-size cluster: the first
// element that cannot be a cluster child.
func mkvUnknownClusterEnd(data []byte, pos, end int) int {
	for pos < end {
		el, err := readMKVElement(data, pos, end)
		if err != nil || el.unknownSize {
			return pos
		}
		switch el.id {
		case mkvIDTimecode, mkvIDSimpleBlock, mkvIDBlockGroup, mkvIDPosition, mkvIDPrevSize, mkvIDSilentTracks:
			pos = el.end
		default:
			return pos
		}
	}
	return pos
}

func readMKVElement(data []byte, pos, limit int) (mkvElement, error) {
	id, idLen, err := readMKVVint(data, pos, limit, 4, true)
	if err != nil {
		return mkvElement{}, err
	}
	size, sizeLen, err := readMKVVint(data, pos+idLen, limit, 8, false)
	if err != nil {
		return mkvElement{}, err
	}
	el := mkvElement{id: uint32(id), start: pos, dataStart: pos + idLen + sizeLen}
	// An all-ones size value means "unknown size" (streamed element).
	if size == (uint64(1)<<(7*uint(sizeLen)))-1 {
		el.unknownSize = true
		el.end = -1
		return el, nil
	}
	if size > uint64(limit-el.dataStart) {
		return mkvElement{}, errMKVInvalid
	}
	el.end = el.dataStart + int(size)
	return el, nil
}

// readMKVVint reads an EBML variable-length integer. IDs keep their length
// marker bit; sizes drop it.
func readMKVVint(data []byte, pos, limit, maxLen int, keepMarker bool) (uint64, int, error) {
	if pos >= limit || pos >= len(data) {
		return 0, 0, errMKVInvalid
	}
	first := data[pos]
	length := 1
	for mask := byte(0x80); length <= maxLen && first&mask == 0; mask >>= 1 {
		length++
	}
	if length > maxLen || pos+length > limit {
		return 0, 0, errMKVInvalid
	}
	value := uint64(first)
	if !keepMarker {
		value &= uint64(0xFF >> uint(length))
	}
	for i := 1; i < length; i++ {
		value = value<<8 | uint64(data[pos+i])
	}
	return value, length, nil
}

func mkvTimecodeScale(info []byte) int64 {
	pos := 0
	for pos < len(info) {
		el, err := readMKVElement(info, pos, len(info))
		if err != nil || el.unknownSize {
			break
		}
		if el.id == mkvIDTimecodeScale {
			if v := mkvUint(info[el.dataStart:el.end]); v > 0 && v <= uint64(time.Second) {
				return int64(v)
			}
		}
		pos = el.end
	}
	return mkvDefaultTimecodeScale
}

// mkvClusterTimes returns the cluster Timecode and the largest relative block
// timecode, both in ticks. A cluster without a Timecode is invalid.
func mkvClusterTimes(cluster []byte) (int64, int64, error) {
	var (
		timecode int64 = -1
		maxTicks int64
	)
	blockTicks := func(block []byte) {
		_, n, err := readMKVVint(block, 0, len(block), 8, false) // track number
		if err != nil || n+2 > len(block) {
			return
		}
		if rel := int64(int16(binary.BigEndian.Uint16(block[n : n+2]))); rel > maxTicks {
			maxTicks = rel
		}
	}
	pos := 0
	for pos < len(cluster) {
		el, err := readMKVElement(cluster, pos, len(cluster))
		if err != nil || el.unknownSize {
			return 0, 0, errMKVInvalid
		}
		body := cluster[el.dataStart:el.end]
		switch el.id {
		case mkvIDTimecode:
			v := mkvUint(body)
			if v > 1<<62 {
				return 0, 0, errMKVInvalid
			}
			timecode = int64(v)
		case mkvIDSimpleBlock:
			blockTicks(body)
		case mkvIDBlockGroup:
			for p := 0; p < len(body); {
				child, err := readMKVElement(body, p, len(body))
				if err != nil || child.unknownSize {
					break
				}
				if child.id == mkvIDBlock {
					blockTicks(body[child.dataStart:child.end])
				}
				p = child.end
			}
		}
		pos = el.end
	}
	if timecode < 0 {
		return 0, 0, errMKVInvalid
	}
	return timecode, maxTicks, nil
}

func mkvUint(b []byte) uint64 {
	if len(b) > 8 {
		return 0
	}
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}

// mkvEncode writes an element with an 8-byte known size.
func mkvEncode(id uint32, content []byte) []byte {
	var idBytes []byte
	switch {
	case id > 0xFFFFFF:
		idBytes = []byte{byte(id >> 24), byte(id >> 16), byte(id >> 8), byte(id)}
	case id > 0xFFFF:
		idBytes = []byte{byte(id >> 16), byte(id >> 8), byte(id)}
	case id > 0xFF:
		idBytes = []byte{byte(id >> 8), byte(id)}
	default:
		idBytes = []byte{byte(id)}
	}
	size := make([]byte, 8)
	binary.BigEndian.PutUint64(size, uint64(len(content)))
	size[0] = 0x01
	out := make([]byte, 0, len(idBytes)+8+len(content))
	return append(append(append(out, idBytes...), size...), content...)
}

// kvsFragmentMKV builds the standalone MKV that GetMedia and friends return
// for one fragment: EBML header, an unknown-size segment with the stream's
// Info and Tracks, the cluster, then the AWS_KINESISVIDEO_* tags.
func kvsFragmentMKV(c kvsMKVCluster, number string, producer, server time.Time) []byte {
	tag := func(name, value string) []byte {
		return mkvEncode(mkvIDSimpleTag, append(mkvEncode(mkvIDTagName, []byte(name)), mkvEncode(mkvIDTagString, []byte(value))...))
	}
	var tags []byte
	tags = append(tags, tag("AWS_KINESISVIDEO_FRAGMENT_NUMBER", number)...)
	tags = append(tags, tag("AWS_KINESISVIDEO_SERVER_TIMESTAMP", kvsSecondsString(server))...)
	tags = append(tags, tag("AWS_KINESISVIDEO_PRODUCER_TIMESTAMP", kvsSecondsString(producer))...)
	tags = append(tags, tag("AWS_KINESISVIDEO_CONTINUATION_TOKEN", number)...)

	out := append([]byte{}, c.ebml...)
	out = append(out, 0x18, 0x53, 0x80, 0x67, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF)
	out = append(out, c.info...)
	out = append(out, c.tracks...)
	out = append(out, mkvEncode(mkvIDCluster, c.content)...)
	return append(out, mkvEncode(mkvIDTags, mkvEncode(mkvIDTag, tags))...)
}

func kvsSecondsString(t time.Time) string {
	return strconv.FormatFloat(float64(t.UnixMilli())/1000, 'f', 3, 64)
}
