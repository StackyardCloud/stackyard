package server

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const kvsMaxClipRange = 24 * time.Hour

func kvsNoDataRetention() *kvsFault {
	return &kvsFault{http.StatusBadRequest, "NoDataRetentionException", "The stream has DataRetentionInHours set to 0, so no media is archived."}
}

// kvsArchive returns the stream and its archived fragments. Streams with no
// data retention have nothing archived.
func (s *Server) kvsArchive(payload map[string]any) (kvsMediaStream, []kvsFragment, *kvsFault) {
	stream, fragments, _, fault := s.kinesisvideostreams.mediaSnapshot(kvsPayloadString(payload, "StreamName"), kvsPayloadString(payload, "StreamARN"), time.Now().UTC())
	if fault != nil {
		return stream, nil, fault
	}
	if stream.RetentionHours == 0 {
		fragments = nil
	}
	return stream, fragments, nil
}

// kvsRange is a fragment timestamp window. A zero end means open-ended.
type kvsRange struct {
	producer   bool
	start, end time.Time
}

func (rg kvsRange) contains(f kvsFragment) bool {
	t := kvsFragmentTime(f, rg.producer)
	return !t.Before(rg.start) && (rg.end.IsZero() || !t.After(rg.end))
}

func (rg kvsRange) filter(fragments []kvsFragment) []kvsFragment {
	out := []kvsFragment{}
	for _, f := range fragments {
		if rg.contains(f) {
			out = append(out, f)
		}
	}
	return out
}

// kvsParseRange reads a {<typeKey>, TimestampRange{StartTimestamp, EndTimestamp}}
// selector. needEnd requires both ends and enforces start < end.
func kvsParseRange(selector map[string]any, typeKey string, needType, needEnd bool) (kvsRange, bool, *kvsFault) {
	rg := kvsRange{}
	switch kvsString(selector[typeKey]) {
	case "PRODUCER_TIMESTAMP":
		rg.producer = true
	case "SERVER_TIMESTAMP":
	case "":
		if needType {
			return rg, false, kvsInvalidArgument("%s is required", typeKey)
		}
	default:
		return rg, false, kvsInvalidArgument("%s must be PRODUCER_TIMESTAMP or SERVER_TIMESTAMP", typeKey)
	}
	timestamps, _ := selector["TimestampRange"].(map[string]any)
	if timestamps == nil {
		if needEnd {
			return rg, false, kvsInvalidArgument("TimestampRange is required")
		}
		return rg, false, nil
	}
	start, hasStart, fault := kvsPayloadTime(timestamps, "StartTimestamp")
	if fault != nil {
		return rg, false, fault
	}
	end, hasEnd, fault := kvsPayloadTime(timestamps, "EndTimestamp")
	if fault != nil {
		return rg, false, fault
	}
	if !hasStart || (needEnd && !hasEnd) {
		return rg, false, kvsInvalidArgument("TimestampRange needs StartTimestamp and EndTimestamp")
	}
	if hasEnd && !end.After(start) {
		return rg, false, kvsInvalidArgument("EndTimestamp must be after StartTimestamp")
	}
	rg.start, rg.end = start, end
	return rg, true, nil
}

func (s *Server) kvsListFragments(w http.ResponseWriter, payload map[string]any) *kvsFault {
	maxResults, fault := kvsPayloadInt(payload, "MaxResults", 1, 1000, 100)
	if fault != nil {
		return fault
	}
	var rg *kvsRange
	if selector, _ := payload["FragmentSelector"].(map[string]any); selector != nil {
		parsed, _, fault := kvsParseRange(selector, "FragmentSelectorType", true, true)
		if fault != nil {
			return fault
		}
		rg = &parsed
	}
	_, fragments, fault := s.kvsArchive(payload)
	if fault != nil {
		return fault
	}
	if rg != nil {
		fragments = rg.filter(fragments)
	}
	start := 0
	if token := kvsPayloadString(payload, "NextToken"); token != "" {
		number, err := base64.StdEncoding.DecodeString(token)
		idx := kvsFragmentIndex(fragments, string(number))
		if err != nil || idx < 0 {
			return kvsInvalidArgument("NextToken is invalid or has expired")
		}
		start = idx + 1
	}
	end := min(start+int(maxResults), len(fragments))
	items := make([]map[string]any, 0, end-start)
	for _, f := range fragments[start:end] {
		items = append(items, map[string]any{
			"FragmentNumber":               f.Number,
			"FragmentSizeInBytes":          len(f.Data),
			"ProducerTimestamp":            kvsEpochSeconds(f.Producer),
			"ServerTimestamp":              kvsEpochSeconds(f.Server),
			"FragmentLengthInMilliseconds": f.DurationMs,
		})
	}
	out := map[string]any{"Fragments": items}
	if end < len(fragments) && end > start {
		out["NextToken"] = base64.StdEncoding.EncodeToString([]byte(fragments[end-1].Number))
	}
	respondKinesisVideoStreamsJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) kvsGetMediaForFragmentList(w http.ResponseWriter, payload map[string]any) *kvsFault {
	numbers, _ := payload["Fragments"].([]any)
	if len(numbers) == 0 || len(numbers) > 1000 {
		return kvsInvalidArgument("Fragments must list between 1 and 1000 fragment numbers")
	}
	_, fragments, fault := s.kvsArchive(payload)
	if fault != nil {
		return fault
	}
	selected := make([]kvsFragment, 0, len(numbers))
	for _, raw := range numbers {
		number := kvsString(raw)
		idx := kvsFragmentIndex(fragments, number)
		if idx < 0 {
			return kvsNotFound("Fragment %s was not found in the stream.", number)
		}
		selected = append(selected, fragments[idx])
	}
	kvsWriteFragments(w, selected)
	return nil
}

// kvsGetClip returns the stored fragments in the range. ponytail: the
// service transmuxes clips to MP4; Stackyard returns the Matroska fragments
// as stored, so add a muxer if a test inspects MP4 boxes.
func (s *Server) kvsGetClip(w http.ResponseWriter, payload map[string]any) *kvsFault {
	selector, _ := payload["ClipFragmentSelector"].(map[string]any)
	if selector == nil {
		return kvsInvalidArgument("ClipFragmentSelector is required")
	}
	rg, _, fault := kvsParseRange(selector, "FragmentSelectorType", true, true)
	if fault != nil {
		return fault
	}
	if rg.end.Sub(rg.start) > kvsMaxClipRange {
		return kvsInvalidArgument("The clip time range can't exceed 24 hours")
	}
	stream, fragments, fault := s.kvsArchive(payload)
	if fault != nil {
		return fault
	}
	if stream.RetentionHours == 0 {
		return kvsNoDataRetention()
	}
	fragments = rg.filter(fragments)
	if len(fragments) == 0 {
		return kvsNotFound("No fragments found in the specified time range.")
	}
	kvsWriteFragments(w, fragments)
	return nil
}

// kvsGetImages samples the range at SamplingInterval. ponytail: Stackyard has
// no video decoder, so a sample covered by a fragment gets a solid gray
// placeholder image of the requested size; uncovered samples report NO_MEDIA.
func (s *Server) kvsGetImages(w http.ResponseWriter, payload map[string]any) *kvsFault {
	rg, _, fault := kvsParseRange(map[string]any{
		"ImageSelectorType": payload["ImageSelectorType"],
		"TimestampRange":    map[string]any{"StartTimestamp": payload["StartTimestamp"], "EndTimestamp": payload["EndTimestamp"]},
	}, "ImageSelectorType", true, true)
	if fault != nil {
		return fault
	}
	format := kvsPayloadString(payload, "Format")
	if format != "JPEG" && format != "PNG" {
		return kvsInvalidArgument("Format must be JPEG or PNG")
	}
	interval, fault := kvsPayloadInt(payload, "SamplingInterval", 1, int64(24*time.Hour/time.Millisecond), 3000)
	if fault != nil {
		return fault
	}
	maxResults, fault := kvsPayloadInt(payload, "MaxResults", 1, 100, 25)
	if fault != nil {
		return fault
	}
	width, fault := kvsPayloadInt(payload, "WidthPixels", 1, 3840, 0)
	if fault != nil {
		return fault
	}
	height, fault := kvsPayloadInt(payload, "HeightPixels", 1, 2160, 0)
	if fault != nil {
		return fault
	}
	quality := 75
	if cfg, _ := payload["FormatConfig"].(map[string]any); cfg != nil {
		for key, value := range cfg {
			q, err := strconv.ParseInt(kvsString(value), 10, 32)
			if key != "JPEGQuality" || format != "JPEG" || err != nil || q < 1 || q > 100 {
				return kvsInvalidArgument("FormatConfig supports only JPEGQuality (1-100) with Format JPEG")
			}
			quality = int(q)
		}
	}
	stream, fragments, fault := s.kvsArchive(payload)
	if fault != nil {
		return fault
	}
	if stream.RetentionHours == 0 {
		return kvsNoDataRetention()
	}
	step := time.Duration(interval) * time.Millisecond
	total := int64(rg.end.Sub(rg.start)/step) + 1
	first := int64(0)
	if token := kvsPayloadString(payload, "NextToken"); token != "" {
		decoded, err := base64.StdEncoding.DecodeString(token)
		n, convErr := strconv.ParseInt(string(decoded), 10, 64)
		if err != nil || convErr != nil || n < 0 || n >= total {
			return kvsInvalidArgument("NextToken is invalid")
		}
		first = n
	}
	last := min(first+maxResults, total)
	var content string
	images := make([]map[string]any, 0, last-first)
	for i := first; i < last; i++ {
		at := rg.start.Add(time.Duration(i) * step)
		img := map[string]any{"TimeStamp": kvsEpochSeconds(at)}
		if kvsCovered(fragments, at, step, rg.producer) {
			if content == "" {
				content = kvsPlaceholderImage(format, width, height, quality)
			}
			img["ImageContent"] = content
		} else {
			img["Error"] = "NO_MEDIA"
		}
		images = append(images, img)
	}
	out := map[string]any{"Images": images}
	if last < total {
		out["NextToken"] = base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(last, 10)))
	}
	respondKinesisVideoStreamsJSON(w, http.StatusOK, out)
	return nil
}

// kvsCovered reports whether a fragment's media spans time at. A fragment
// covers at least one sampling step so single-frame fragments still match.
func kvsCovered(fragments []kvsFragment, at time.Time, step time.Duration, producer bool) bool {
	for _, f := range fragments {
		start := kvsFragmentTime(f, producer)
		span := max(time.Duration(f.DurationMs)*time.Millisecond, step)
		if !at.Before(start) && at.Before(start.Add(span)) {
			return true
		}
	}
	return false
}

func kvsPlaceholderImage(format string, width, height int64, quality int) string {
	switch {
	case width == 0 && height == 0:
		width, height = 640, 360
	case height == 0:
		height = max(1, min(2160, width*9/16))
	case width == 0:
		width = max(1, min(3840, height*16/9))
	}
	img := image.NewGray(image.Rect(0, 0, int(width), int(height)))
	for i := range img.Pix {
		img.Pix[i] = color.Gray{Y: 128}.Y
	}
	var buf bytes.Buffer
	if format == "PNG" {
		_ = png.Encode(&buf, img)
	} else {
		_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func kvsEnum(payload map[string]any, key, def string, allowed ...string) (string, *kvsFault) {
	value := kvsPayloadString(payload, key)
	if value == "" {
		return def, nil
	}
	for _, a := range allowed {
		if value == a {
			return value, nil
		}
	}
	return "", kvsInvalidArgument("%s must be one of %s", key, strings.Join(allowed, ", "))
}
